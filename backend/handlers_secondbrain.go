//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// ── Second Brain 데이터 구조 ───────────────────────────────────

type BrainEntry struct {
	ID        string   `json:"id"`
	Source    string   `json:"source"`    // "memory" | "note" | "recall" | "email" | "calendar" | "file"
	Title     string   `json:"title"`
	Content   string   `json:"content"`
	Tags      []string `json:"tags"`
	Timestamp string   `json:"timestamp"`
	FilePath  string   `json:"file_path,omitempty"`
}

type BrainIndex struct {
	Entries   []BrainEntry `json:"entries"`
	UpdatedAt string       `json:"updated_at"`
}

var (
	brainMu    sync.RWMutex
	brainIndex BrainIndex
)

func brainIndexPath() string {
	return filepath.Join(os.Getenv("APPDATA"), "Nexus", "brain", "index.json")
}

func loadBrainIndex() {
	brainMu.Lock()
	defer brainMu.Unlock()
	data, err := os.ReadFile(brainIndexPath())
	if err != nil {
		brainIndex = BrainIndex{Entries: []BrainEntry{}}
		return
	}
	json.Unmarshal(data, &brainIndex)
}

func saveBrainIndex() {
	brainIndex.UpdatedAt = time.Now().Format(time.RFC3339)
	data, _ := json.MarshalIndent(brainIndex, "", "  ")
	os.MkdirAll(filepath.Dir(brainIndexPath()), 0755)
	os.WriteFile(brainIndexPath(), data, 0644)
}

func upsertBrainEntry(e BrainEntry) {
	brainMu.Lock()
	defer brainMu.Unlock()
	for i, existing := range brainIndex.Entries {
		if existing.ID == e.ID {
			brainIndex.Entries[i] = e
			saveBrainIndex()
			return
		}
	}
	brainIndex.Entries = append(brainIndex.Entries, e)
	saveBrainIndex()
}

// ── 키워드 검색 + LLM 랭킹 ────────────────────────────────────

type BrainSearchResult struct {
	Entry     BrainEntry `json:"entry"`
	Score     int        `json:"score"`
	Highlight string     `json:"highlight"`
}

func searchBrainEntries(query string, limit int) []BrainSearchResult {
	keywords := strings.Fields(strings.ToLower(query))
	var results []BrainSearchResult

	brainMu.RLock()
	entries := make([]BrainEntry, len(brainIndex.Entries))
	copy(entries, brainIndex.Entries)
	brainMu.RUnlock()

	for _, entry := range entries {
		combined := strings.ToLower(entry.Title + " " + entry.Content + " " + strings.Join(entry.Tags, " "))
		score := 0
		for _, kw := range keywords {
			count := strings.Count(combined, kw)
			if count > 0 {
				score += count
				if strings.Contains(strings.ToLower(entry.Title), kw) {
					score += 3
				}
			}
		}
		if score > 0 {
			// 간단한 하이라이트: 첫 200자
			highlight := entry.Content
			if len(highlight) > 200 {
				highlight = highlight[:200] + "..."
			}
			results = append(results, BrainSearchResult{Entry: entry, Score: score, Highlight: highlight})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	if len(results) > limit {
		results = results[:limit]
	}
	return results
}

// ── 자동 인덱싱: 메모리 + 노트 + Recall ──────────────────────

func rebuildBrainIndex() {
	// 1) 노트 인덱싱
	notesPath := filepath.Join(os.Getenv("APPDATA"), "Nexus", "notes.json")
	if data, err := os.ReadFile(notesPath); err == nil {
		var notes []struct {
			ID      string `json:"id"`
			Content string `json:"content"`
			Created string `json:"created"`
		}
		if json.Unmarshal(data, &notes) == nil {
			for _, n := range notes {
				title := n.Content
				if len(title) > 50 {
					title = title[:50] + "..."
				}
				upsertBrainEntry(BrainEntry{
					ID:        "note_" + n.ID,
					Source:    "note",
					Title:     title,
					Content:   n.Content,
					Tags:      extractTags(n.Content),
					Timestamp: n.Created,
				})
			}
		}
	}

	// 2) Recall 인덱싱
	recallPath := filepath.Join(os.Getenv("APPDATA"), "Nexus", "recall", "index.json")
	if data, err := os.ReadFile(recallPath); err == nil {
		var entries []struct {
			ID        string `json:"id"`
			Text      string `json:"text"`
			App       string `json:"app"`
			Timestamp string `json:"timestamp"`
		}
		if json.Unmarshal(data, &entries) == nil {
			for _, e := range entries {
				if len(e.Text) < 20 {
					continue
				}
				title := e.App + ": " + e.Text
				if len(title) > 60 {
					title = title[:60] + "..."
				}
				upsertBrainEntry(BrainEntry{
					ID:        "recall_" + e.ID,
					Source:    "recall",
					Title:     title,
					Content:   e.Text,
					Tags:      append([]string{e.App}, extractTags(e.Text)...),
					Timestamp: e.Timestamp,
				})
			}
		}
	}

	// 3) 메모리(장기 메모리) 인덱싱
	memPath := filepath.Join(os.Getenv("APPDATA"), "Nexus", "agent_memory.json")
	if data, err := os.ReadFile(memPath); err == nil {
		var mem struct {
			Entries []struct {
				ID      string `json:"id"`
				Content string `json:"content"`
				Tags    []string `json:"tags"`
				Created string `json:"created"`
			} `json:"entries"`
		}
		if json.Unmarshal(data, &mem) == nil {
			for _, e := range mem.Entries {
				title := e.Content
				if len(title) > 60 {
					title = title[:60] + "..."
				}
				upsertBrainEntry(BrainEntry{
					ID:        "mem_" + e.ID,
					Source:    "memory",
					Title:     title,
					Content:   e.Content,
					Tags:      e.Tags,
					Timestamp: e.Created,
				})
			}
		}
	}
}

func extractTags(content string) []string {
	var tags []string
	words := strings.Fields(content)
	seen := map[string]bool{}
	for _, w := range words {
		w = strings.Trim(w, ".,!?;:\"'")
		if len([]rune(w)) >= 3 && !seen[w] {
			tags = append(tags, w)
			seen[w] = true
			if len(tags) >= 8 {
				break
			}
		}
	}
	return tags
}

// ── HTTP 핸들러 ────────────────────────────────────────────────

func handleBrainIndex(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var entry BrainEntry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		writeJSON(w, 400, map[string]string{"error": msgT("잘못된 요청", "Invalid request", lang)})
		return
	}
	if entry.ID == "" {
		entry.ID = fmt.Sprintf("manual_%d", time.Now().UnixMilli())
	}
	if entry.Timestamp == "" {
		entry.Timestamp = time.Now().Format(time.RFC3339)
	}
	upsertBrainEntry(entry)
	json200(w, map[string]any{"ok": true, "id": entry.ID})
}

func handleBrainSearch(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		Query  string `json:"query"`
		Limit  int    `json:"limit"`
		Source string `json:"source"` // 필터: "" = 전체
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Query == "" {
		writeJSON(w, 400, map[string]string{"error": msgT("query 필드가 필요합니다", "query field is required", lang)})
		return
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}

	results := searchBrainEntries(req.Query, req.Limit*3)

	// 소스 필터
	if req.Source != "" {
		var filtered []BrainSearchResult
		for _, r := range results {
			if r.Entry.Source == req.Source {
				filtered = append(filtered, r)
			}
		}
		results = filtered
	}
	if len(results) > req.Limit {
		results = results[:req.Limit]
	}

	// LLM으로 요약 생성
	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()

	summary := ""
	if gKey != "" && len(results) > 0 {
		var snippets []string
		for i, r := range results {
			if i >= 5 {
				break
			}
			snippets = append(snippets, fmt.Sprintf("[%s] %s: %s", r.Entry.Source, r.Entry.Title, r.Highlight))
		}
		prompt := fmt.Sprintf(`사용자가 "%s"를 검색했습니다. 아래 검색 결과를 2-3문장으로 요약해주세요:

%s`, req.Query, strings.Join(snippets, "\n"))
		summary, _, _ = callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 256, false)
	}

	brainMu.RLock()
	total := len(brainIndex.Entries)
	brainMu.RUnlock()

	json200(w, map[string]any{
		"results": results,
		"total":   total,
		"summary": summary,
		"query":   req.Query,
	})
}

func handleBrainRebuild(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	go rebuildBrainIndex()
	brainMu.RLock()
	total := len(brainIndex.Entries)
	brainMu.RUnlock()
	json200(w, map[string]any{"ok": true, "message": msgT("인덱스 재구축 중...", "Rebuilding index...", lang), "current_entries": total})
}

func handleBrainStats(w http.ResponseWriter, r *http.Request) {
	brainMu.RLock()
	defer brainMu.RUnlock()

	counts := map[string]int{}
	for _, e := range brainIndex.Entries {
		counts[e.Source]++
	}
	json200(w, map[string]any{
		"total":      len(brainIndex.Entries),
		"by_source":  counts,
		"updated_at": brainIndex.UpdatedAt,
	})
}
