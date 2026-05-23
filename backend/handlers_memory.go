//go:build windows

package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// ──────────────────────────────────────────────────────────────
// 에이전트 장기 메모리 (LTM: Long-Term Memory)
// 브라우저 작업, 스케줄 결과, 사용자 선호도를 JSON에 영구 저장
// ──────────────────────────────────────────────────────────────

type AgentMemoryEntry struct {
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`      // browser_agent|scheduled_task|user_prefs|search|vision
	Command   string `json:"command"`
	Result    string `json:"result"`
	Success   bool   `json:"success"`
	Tags      []string `json:"tags,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type MemoryStore struct {
	mu      sync.RWMutex
	entries []AgentMemoryEntry
	path    string
}

var globalMemory *MemoryStore

func initMemory() {
	storePath := filepath.Join(os.TempDir(), "nexus_memory.json")
	globalMemory = &MemoryStore{
		path: storePath,
	}
	globalMemory.load()
}

func (m *MemoryStore) load() {
	data, err := os.ReadFile(m.path)
	if err != nil {
		return
	}
	json.Unmarshal(data, &m.entries)
}

func (m *MemoryStore) save() {
	m.mu.RLock()
	data, _ := json.MarshalIndent(m.entries, "", "  ")
	m.mu.RUnlock()
	os.WriteFile(m.path, data, 0644)
}

// saveAgentMemory: 메모리 추가 (전역 함수 — 모든 핸들러에서 사용)
func saveAgentMemory(entry AgentMemoryEntry) {
	if globalMemory == nil {
		return
	}
	globalMemory.mu.Lock()
	globalMemory.entries = append(globalMemory.entries, entry)
	// 최대 500개 유지
	if len(globalMemory.entries) > 500 {
		globalMemory.entries = globalMemory.entries[len(globalMemory.entries)-500:]
	}
	globalMemory.mu.Unlock()
	globalMemory.save()
}

// ──────────────────────────────────────────────────────────────
// 메모리 검색 (키워드 + 타입 필터)
// ──────────────────────────────────────────────────────────────

func searchMemory(keyword, memType string, limit int) []AgentMemoryEntry {
	if globalMemory == nil {
		return nil
	}
	globalMemory.mu.RLock()
	all := make([]AgentMemoryEntry, len(globalMemory.entries))
	copy(all, globalMemory.entries)
	globalMemory.mu.RUnlock()

	// 최신 순 정렬
	sort.Slice(all, func(i, j int) bool {
		return all[i].Timestamp > all[j].Timestamp
	})

	var results []AgentMemoryEntry
	keyword = strings.ToLower(keyword)
	for _, e := range all {
		if memType != "" && e.Type != memType {
			continue
		}
		if keyword != "" {
			haystack := strings.ToLower(e.Command + " " + e.Result)
			if !strings.Contains(haystack, keyword) {
				continue
			}
		}
		results = append(results, e)
		if limit > 0 && len(results) >= limit {
			break
		}
	}
	return results
}

// buildContextFromMemory: 현재 명령과 관련된 과거 경험을 컨텍스트로 조합
func buildContextFromMemory(command string, maxEntries int) string {
	if globalMemory == nil || len(command) == 0 {
		return ""
	}

	// 키워드 추출 (간단히 공백 기준)
	words := strings.Fields(strings.ToLower(command))
	var relevant []AgentMemoryEntry

	seen := make(map[string]bool)
	for _, word := range words {
		if len(word) < 2 {
			continue
		}
		entries := searchMemory(word, "", 5)
		for _, e := range entries {
			if !seen[e.ID] && e.Success {
				seen[e.ID] = true
				relevant = append(relevant, e)
			}
		}
		if len(relevant) >= maxEntries {
			break
		}
	}

	if len(relevant) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("=== 관련 과거 경험 ===\n")
	for _, e := range relevant {
		if len(e.Result) > 200 {
			e.Result = e.Result[:200] + "..."
		}
		sb.WriteString("• [")
		sb.WriteString(e.Timestamp[:10])
		sb.WriteString("] ")
		sb.WriteString(e.Command)
		if e.Result != "" {
			sb.WriteString(" → ")
			sb.WriteString(e.Result)
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// ──────────────────────────────────────────────────────────────
// HTTP 핸들러들
// ──────────────────────────────────────────────────────────────

// GET /api/memory/list?type=xxx&keyword=yyy&limit=20
func handleMemoryList(w http.ResponseWriter, r *http.Request) {
	memType := r.URL.Query().Get("type")
	keyword := r.URL.Query().Get("keyword")
	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if limitStr != "" {
		if l, err := parseIntSafe(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	entries := searchMemory(keyword, memType, limit)
	if entries == nil {
		entries = []AgentMemoryEntry{}
	}

	json200(w, map[string]any{
		"success": true,
		"entries": entries,
		"total":   len(entries),
	})
}

// POST /api/memory/search
func handleMemorySearch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Keyword string `json:"keyword"`
		Type    string `json:"type"`
		Limit   int    `json:"limit"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Limit == 0 {
		req.Limit = 10
	}

	entries := searchMemory(req.Keyword, req.Type, req.Limit)
	if entries == nil {
		entries = []AgentMemoryEntry{}
	}

	// 검색 결과를 LLM으로 요약
	summary := ""
	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()
	if gKey != "" && len(entries) > 0 && req.Keyword != "" {
		lines := make([]string, 0, len(entries))
		for _, e := range entries {
			result := e.Result
			if len(result) > 100 {
				result = result[:100] + "..."
			}
			lines = append(lines, "• "+e.Timestamp[:10]+": "+e.Command+" → "+result)
		}
		prompt := "'" + req.Keyword + "'에 관한 과거 기록:\n" + strings.Join(lines, "\n") + "\n\n위 기록을 바탕으로 유용한 인사이트를 2-3줄로 요약하세요."
		summary, _, _ = callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 256, false)
	}

	json200(w, map[string]any{
		"success": true,
		"entries": entries,
		"total":   len(entries),
		"summary": summary,
	})
}

// DELETE /api/memory/clear?type=xxx
func handleMemoryClear(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	memType := r.URL.Query().Get("type")

	if globalMemory == nil {
		json200(w, map[string]any{"success": true, "message": msgT("메모리 없음", "No memory", lang)})
		return
	}

	globalMemory.mu.Lock()
	if memType == "" {
		globalMemory.entries = nil
	} else {
		var filtered []AgentMemoryEntry
		for _, e := range globalMemory.entries {
			if e.Type != memType {
				filtered = append(filtered, e)
			}
		}
		globalMemory.entries = filtered
	}
	globalMemory.mu.Unlock()
	globalMemory.save()

	json200(w, map[string]any{
		"success": true,
		"message": msgT("메모리 초기화 완료", "Memory cleared", lang),
	})
}

// GET /api/memory/stats
func handleMemoryStats(w http.ResponseWriter, r *http.Request) {
	if globalMemory == nil {
		json200(w, map[string]any{"success": true, "total": 0, "by_type": map[string]int{}})
		return
	}

	globalMemory.mu.RLock()
	total := len(globalMemory.entries)
	byType := make(map[string]int)
	successCount := 0
	for _, e := range globalMemory.entries {
		byType[e.Type]++
		if e.Success {
			successCount++
		}
	}
	globalMemory.mu.RUnlock()

	successRate := 0.0
	if total > 0 {
		successRate = float64(successCount) / float64(total) * 100
	}

	json200(w, map[string]any{
		"success":      true,
		"total":        total,
		"by_type":      byType,
		"success_rate": successRate,
	})
}

// 안전한 int 파싱
func parseIntSafe(s string) (int, error) {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, &strconvError{s}
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

type strconvError struct{ s string }
func (e *strconvError) Error() string { return "invalid number: " + e.s }
