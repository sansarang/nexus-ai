//go:build !windows

package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// ── 브라우저 히스토리 경로 ─────────────────────────────────

func chromePaths() []string {
	home, _ := os.UserHomeDir()
	if runtime.GOOS == "darwin" {
		return []string{
			filepath.Join(home, "Library/Application Support/Google/Chrome/Default/History"),
			filepath.Join(home, "Library/Application Support/Google/Chrome/Profile 1/History"),
			filepath.Join(home, "Library/Application Support/Microsoft Edge/Default/History"),
			filepath.Join(home, "Library/Application Support/Chromium/Default/History"),
			filepath.Join(home, "Library/Application Support/Brave Browser/Default/History"),
		}
	}
	// Linux
	return []string{
		filepath.Join(home, ".config/google-chrome/Default/History"),
		filepath.Join(home, ".config/chromium/Default/History"),
	}
}

func findChromeHistory() string {
	for _, p := range chromePaths() {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// Chrome 히스토리는 실행 중에 잠겨 있으므로 임시 복사 후 sqlite3 CLI로 읽기
func queryChromeHistory(filter string, days int) ([]map[string]string, error) {
	histPath := findChromeHistory()
	if histPath == "" {
		return nil, fmt.Errorf("Chrome/Edge 히스토리 파일을 찾을 수 없어요")
	}

	// 임시 복사 (잠금 우회)
	tmpPath := filepath.Join(os.TempDir(), fmt.Sprintf("nexus_hist_%d.db", time.Now().UnixMilli()))
	defer os.Remove(tmpPath)

	src, err := os.Open(histPath)
	if err != nil {
		return nil, fmt.Errorf("히스토리 읽기 실패 (Chrome 실행 중이면 잠겨 있을 수 있음): %v", err)
	}
	dst, _ := os.Create(tmpPath)
	io.Copy(dst, src)
	src.Close()
	dst.Close()

	// Chrome 시간 = 1601-01-01 기준 마이크로초
	// days 전 Chrome 타임스탬프 계산
	cutoff := time.Now().AddDate(0, 0, -days)
	// Chrome epoch: 1601-01-01
	chromeEpoch := time.Date(1601, 1, 1, 0, 0, 0, 0, time.UTC)
	cutoffChrome := cutoff.Sub(chromeEpoch).Microseconds()

	whereClause := fmt.Sprintf("last_visit_time > %d", cutoffChrome)
	if filter != "" {
		whereClause += fmt.Sprintf(` AND url LIKE '%%%s%%'`, filter)
	}

	query := fmt.Sprintf(
		`SELECT url, title, last_visit_time FROM urls WHERE %s ORDER BY last_visit_time DESC LIMIT 200`,
		whereClause,
	)

	out, err := exec.Command("sqlite3", "-separator", "|", tmpPath, query).Output()
	if err != nil {
		return nil, fmt.Errorf("sqlite3 실행 실패 (sqlite3 미설치?): %v", err)
	}

	var results []map[string]string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.SplitN(line, "|", 3)
		if len(parts) < 2 {
			continue
		}
		rawURL := parts[0]
		title := ""
		if len(parts) > 1 {
			title = parts[1]
		}
		visitTimeStr := ""
		if len(parts) > 2 {
			visitTimeStr = parts[2]
		}
		// Chrome timestamp → readable
		visitTime := ""
		if visitTimeStr != "" {
			var chromeMicro int64
			fmt.Sscanf(visitTimeStr, "%d", &chromeMicro)
			if chromeMicro > 0 {
				t := chromeEpoch.Add(time.Duration(chromeMicro) * time.Microsecond)
				visitTime = t.Local().Format("01-02 15:04")
			}
		}
		results = append(results, map[string]string{
			"url":        rawURL,
			"title":      title,
			"visit_time": visitTime,
		})
	}
	return results, nil
}

// ── TikTok 히스토리 ───────────────────────────────────────

type TikTokHistoryItem struct {
	URL       string `json:"url"`
	Title     string `json:"title"`
	VideoID   string `json:"video_id"`
	VisitTime string `json:"visit_time"`
}

// GET /api/history/tiktok?days=7
func handleTikTokHistory(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	days := 7
	fmt.Sscanf(r.URL.Query().Get("days"), "%d", &days)

	rows, err := queryChromeHistory("tiktok.com", days)
	if err != nil {
		// 히스토리 못 읽으면 Tavily로 인기 영상 대체
		llmMu.RLock()
		tKey := llmTavilyKey
		llmMu.RUnlock()
		msg := fmt.Sprintf(msgT("브라우저 히스토리 접근 불가 (%v). 대신 TikTok 인기 영상을 검색할게요.", "Cannot access browser history (%v). Searching TikTok trending videos instead.", lang), err)
		if tKey != "" {
			tr, ok := tavilySearch(tKey, "tiktok 이번주 인기 영상 트렌드", 8)
			if ok {
				json200(w, map[string]any{
					"success":  true,
					"source":   "search_fallback",
					"message":  msg,
					"items":    tr.Items,
					"summary":  tr.Summary,
				})
				return
			}
		}
		writeJSON(w, 500, map[string]any{"success": false, "message": msg})
		return
	}

	var items []TikTokHistoryItem
	seen := map[string]bool{}
	for _, row := range rows {
		rawURL := row["url"]
		if !strings.Contains(rawURL, "tiktok.com") {
			continue
		}
		// 영상 URL만 (/@user/video/...)
		if !strings.Contains(rawURL, "/video/") {
			continue
		}
		if seen[rawURL] {
			continue
		}
		seen[rawURL] = true
		// video ID 추출
		videoID := ""
		parsed, err := url.Parse(rawURL)
		if err == nil {
			parts := strings.Split(parsed.Path, "/")
			for i, p := range parts {
				if p == "video" && i+1 < len(parts) {
					videoID = parts[i+1]
					break
				}
			}
		}
		items = append(items, TikTokHistoryItem{
			URL:       rawURL,
			Title:     row["title"],
			VideoID:   videoID,
			VisitTime: row["visit_time"],
		})
		if len(items) >= 30 {
			break
		}
	}

	json200(w, map[string]any{
		"success": true,
		"items":   items,
		"count":   len(items),
		"days":    days,
		"message": fmt.Sprintf(msgT("최근 %d일간 TikTok 시청 기록 %d개", "TikTok watch history for the last %d days: %d items", lang), days, len(items)),
	})
}

// ── YouTube 히스토리 ──────────────────────────────────────

// GET /api/history/youtube?days=7
func handleYouTubeHistory(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	days := 7
	fmt.Sscanf(r.URL.Query().Get("days"), "%d", &days)

	rows, err := queryChromeHistory("youtube.com/watch", days)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}

	type YTItem struct {
		URL       string `json:"url"`
		Title     string `json:"title"`
		VideoID   string `json:"video_id"`
		VisitTime string `json:"visit_time"`
	}

	var items []YTItem
	seen := map[string]bool{}
	for _, row := range rows {
		rawURL := row["url"]
		if !strings.Contains(rawURL, "youtube.com/watch") {
			continue
		}
		parsed, _ := url.Parse(rawURL)
		vid := parsed.Query().Get("v")
		if vid == "" || seen[vid] {
			continue
		}
		seen[vid] = true
		items = append(items, YTItem{
			URL:       rawURL,
			Title:     row["title"],
			VideoID:   vid,
			VisitTime: row["visit_time"],
		})
		if len(items) >= 50 {
			break
		}
	}

	json200(w, map[string]any{
		"success": true,
		"items":   items,
		"count":   len(items),
		"days":    days,
		"message": fmt.Sprintf(msgT("최근 %d일간 YouTube 시청 기록 %d개", "YouTube watch history for the last %d days: %d items", lang), days, len(items)),
	})
}

// ── 키워드 분석 ────────────────────────────────────────────

// GET /api/history/keywords?days=7
func handleHistoryKeywords(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	days := 7
	fmt.Sscanf(r.URL.Query().Get("days"), "%d", &days)

	rows, err := queryChromeHistory("", days)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}

	// 제목에서 키워드 빈도 분석
	freq := map[string]int{}
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"is": true, "in": true, "on": true, "at": true, "to": true,
		"이": true, "가": true, "은": true, "는": true, "을": true, "를": true,
		"의": true, "에": true, "와": true, "과": true, "도": true, "로": true,
		"YouTube": true, "Google": true, "Naver": true,
	}
	for _, row := range rows {
		title := row["title"]
		words := strings.Fields(title)
		for _, w := range words {
			w = strings.Trim(w, ".,!?\"'()[]{}|")
			if len([]rune(w)) < 2 || stopWords[w] {
				continue
			}
			freq[w]++
		}
	}

	type KW struct {
		Word  string `json:"word"`
		Count int    `json:"count"`
	}
	var keywords []KW
	for word, count := range freq {
		if count >= 2 {
			keywords = append(keywords, KW{word, count})
		}
	}
	sort.Slice(keywords, func(i, j int) bool { return keywords[i].Count > keywords[j].Count })
	if len(keywords) > 20 {
		keywords = keywords[:20]
	}

	// LLM으로 콘텐츠 추천
	var kwList []string
	for _, k := range keywords {
		kwList = append(kwList, fmt.Sprintf("%s(%d회)", k.Word, k.Count))
	}
	recommendation := ""
	if len(kwList) > 0 {
		llmMu.RLock()
		tKey := llmTavilyKey
		llmMu.RUnlock()
		prompt := fmt.Sprintf(`사용자가 최근 %d일간 자주 본 키워드: %s
이 키워드 기반으로 관심 분야를 분석하고, 관련 추천 콘텐츠 3가지를 제안해줘. 한국어로.`,
			days, strings.Join(kwList[:min(10, len(kwList))], ", "))
		_ = tKey
		recommendation, _, _ = callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 400, false)
	}

	json200(w, map[string]any{
		"success":        true,
		"keywords":       keywords,
		"total_pages":    len(rows),
		"days":           days,
		"recommendation": recommendation,
		"message":        fmt.Sprintf(msgT("최근 %d일 키워드 분석 완료 (총 %d페이지 방문)", "Keyword analysis for the last %d days complete (total %d pages visited)", lang), days, len(rows)),
	})
}

// ── 전체 히스토리 요약 ────────────────────────────────────

// GET /api/history/summary?days=7
func handleHistorySummary(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	days := 7
	fmt.Sscanf(r.URL.Query().Get("days"), "%d", &days)

	rows, err := queryChromeHistory("", days)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}

	// 도메인별 방문 수
	domainCount := map[string]int{}
	tiktokCount, ytCount := 0, 0
	for _, row := range rows {
		parsed, err := url.Parse(row["url"])
		if err != nil {
			continue
		}
		host := parsed.Hostname()
		host = strings.TrimPrefix(host, "www.")
		domainCount[host]++
		if strings.Contains(host, "tiktok") {
			tiktokCount++
		}
		if strings.Contains(host, "youtube") {
			ytCount++
		}
	}

	type DomainStat struct {
		Domain string `json:"domain"`
		Count  int    `json:"count"`
	}
	var domains []DomainStat
	for d, c := range domainCount {
		domains = append(domains, DomainStat{d, c})
	}
	sort.Slice(domains, func(i, j int) bool { return domains[i].Count > domains[j].Count })
	if len(domains) > 10 {
		domains = domains[:10]
	}

	json200(w, map[string]any{
		"success":      true,
		"total_visits": len(rows),
		"tiktok_visits": tiktokCount,
		"youtube_visits": ytCount,
		"top_domains":  domains,
		"days":         days,
		"message":      fmt.Sprintf(msgT("최근 %d일: 총 %d회 방문, TikTok %d회, YouTube %d회", "Last %d days: %d total visits, TikTok %d, YouTube %d", lang), days, len(rows), tiktokCount, ytCount),
	})
}
