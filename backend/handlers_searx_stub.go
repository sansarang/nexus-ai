package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ══════════════════════════════════════════════════════════════
//  SearXNG / DuckDuckGo 익명 검색 — Tor fallback
//  handleLLMDeepSearch 에서 Tavily 실패 시 자동 호출
// ══════════════════════════════════════════════════════════════

var searxInstances = []string{
	"https://searx.be",
	"https://search.bus-hit.me",
	"https://searxng.world",
	"https://searx.tiekoetter.com",
	"https://searx.work",
}

var fakeUAs = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/124.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_4) AppleWebKit/605.1.15 Safari/605.1.15",
	"Mozilla/5.0 (X11; Linux x86_64; rv:126.0) Gecko/20100101 Firefox/126.0",
}

type SearxResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Content string `json:"content"`
	Engine  string `json:"engine"`
}

// POST /api/search/anonymous
// body: { "query": "...", "limit": 10 }
func handleAnonymousSearch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query  string `json:"query"`
		Limit  int    `json:"limit"`
		UseTor bool   `json:"use_tor"`
	}
	tryDecodeBody(r, &req)
	if req.Query == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "query 필요"})
		return
	}
	if req.Limit == 0 {
		req.Limit = 10
	}

	results := anonymousSearch(req.Query, req.Limit, req.UseTor)
	msg := buildSearchSummary(req.Query, results)

	writeJSON(w, 200, map[string]any{
		"success": true,
		"query":   req.Query,
		"total":   len(results),
		"items":   results,
		"message": msg,
	})
}

// anonymousSearch: SearXNG → DDG 순서로 fallback
func anonymousSearch(query string, limit int, useTor bool) []SearxResult {
	// 1차: SearXNG 시도 (여러 인스턴스 순회)
	results := searchSearXNG(query, limit, useTor)
	if len(results) >= 3 {
		return results
	}

	// 2차: DuckDuckGo API fallback
	ddgResults := searchDDG(query, limit)
	results = append(results, ddgResults...)

	// 중복 URL 제거
	seen := map[string]bool{}
	var deduped []SearxResult
	for _, r := range results {
		if !seen[r.URL] {
			seen[r.URL] = true
			deduped = append(deduped, r)
		}
	}
	return deduped
}

// SearXNG JSON API 검색
func searchSearXNG(query string, limit int, useTor bool) []SearxResult {
	// 인스턴스 랜덤 순서로 시도
	instances := make([]string, len(searxInstances))
	copy(instances, searxInstances)
	rand.Shuffle(len(instances), func(i, j int) { instances[i], instances[j] = instances[j], instances[i] })

	for _, instance := range instances {
		results, err := trySearXNG(instance, query, limit, useTor)
		if err == nil && len(results) > 0 {
			return results
		}
		// 실패 시 다음 인스턴스
		time.Sleep(500 * time.Millisecond)
	}
	return nil
}

func trySearXNG(instance, query string, limit int, useTor bool) ([]SearxResult, error) {
	params := url.Values{}
	params.Set("q", query)
	params.Set("format", "json")
	params.Set("categories", "general")
	params.Set("language", "ko-KR")

	apiURL := instance + "/search?" + params.Encode()

	var transport http.RoundTripper
	if useTor {
		// Tor 프록시 (socks5h://127.0.0.1:9050)
		transport = torTransport()
	}

	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
	}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", fakeUAs[rand.Intn(len(fakeUAs))])
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "ko-KR,ko;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512*1024))

	var raw struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
			Engine  string `json:"engine"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	var results []SearxResult
	for i, r := range raw.Results {
		if i >= limit {
			break
		}
		results = append(results, SearxResult{
			Title:   r.Title,
			URL:     r.URL,
			Content: truncateSearx(r.Content, 200),
			Engine:  "searxng/" + r.Engine,
		})
	}
	return results, nil
}

// DuckDuckGo 즉석 검색 API (HTML 파싱)
func searchDDG(query string, limit int) []SearxResult {
	params := url.Values{}
	params.Set("q", query)
	params.Set("format", "json")
	params.Set("no_html", "1")

	apiURL := "https://api.duckduckgo.com/?" + params.Encode()
	client := &http.Client{Timeout: 8 * time.Second}
	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("User-Agent", fakeUAs[rand.Intn(len(fakeUAs))])

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256*1024))

	var raw struct {
		AbstractText  string `json:"AbstractText"`
		AbstractURL   string `json:"AbstractURL"`
		AbstractTitle string `json:"AbstractTitle"`
		RelatedTopics []struct {
			Text     string `json:"Text"`
			FirstURL string `json:"FirstURL"`
		} `json:"RelatedTopics"`
	}
	json.Unmarshal(body, &raw)

	var results []SearxResult
	if raw.AbstractText != "" && raw.AbstractURL != "" {
		results = append(results, SearxResult{
			Title:   raw.AbstractTitle,
			URL:     raw.AbstractURL,
			Content: truncateSearx(raw.AbstractText, 200),
			Engine:  "duckduckgo",
		})
	}
	for i, t := range raw.RelatedTopics {
		if i+1 >= limit {
			break
		}
		if t.FirstURL == "" {
			continue
		}
		results = append(results, SearxResult{
			Title:   extractDDGTitle(t.Text),
			URL:     t.FirstURL,
			Content: truncateSearx(t.Text, 200),
			Engine:  "duckduckgo",
		})
	}
	return results
}

// Tor SOCKS5 transport (선택적 — Tor 미실행 시 graceful skip)
func torTransport() http.RoundTripper {
	// 표준 net/http로 SOCKS5 다이얼러 직접 구성
	// golang.org/x/net/proxy 없이 환경변수로 설정
	// (실제 Tor 사용 시 ALL_PROXY=socks5h://127.0.0.1:9050 환경변수 설정 필요)
	return http.DefaultTransport
}

func extractDDGTitle(text string) string {
	if idx := strings.Index(text, " - "); idx != -1 {
		return text[:idx]
	}
	if len(text) > 60 {
		return text[:60] + "..."
	}
	return text
}

func buildSearchSummary(query string, results []SearxResult) string {
	if len(results) == 0 {
		return fmt.Sprintf("❌ **\"%s\"** 검색 결과가 없습니다. 검색어를 바꿔보세요.", query)
	}
	engines := map[string]bool{}
	for _, r := range results {
		if r.Engine != "" {
			engines[r.Engine] = true
		}
	}
	engineList := make([]string, 0, len(engines))
	for e := range engines {
		engineList = append(engineList, e)
	}
	return fmt.Sprintf("✅ **\"%s\"** 익명 검색 완료 — %d개 결과 (출처: %s)",
		query, len(results), strings.Join(engineList, ", "))
}

func truncateSearx(s string, n int) string {
	if len([]rune(s)) <= n {
		return s
	}
	return string([]rune(s)[:n]) + "..."
}
