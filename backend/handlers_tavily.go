package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type tavilyResult struct {
	Summary string
	Items   []map[string]string
}

// isNewsQuery 뉴스/최신 관련 쿼리 여부 판별
func isNewsQuery(query string) bool {
	keywords := []string{"뉴스", "news", "최신", "오늘", "어제", "이번주", "최근", "속보", "이슈", "today", "latest", "breaking"}
	q := strings.ToLower(query)
	for _, k := range keywords {
		if strings.Contains(q, k) {
			return true
		}
	}
	return false
}

func tavilySearch(apiKey, query string, maxItems int) (tavilyResult, bool) {
	// 1순위: Supabase Edge Function 프록시 (JWT 있을 때)
	if res, ok := callTavilyViaProxy(query, maxItems); ok {
		return res, true
	}
	// 2순위: 번들 키 직접 호출
	return tavilySearchDomain(apiKey, query, maxItems, "")
}

// tavilySearchImages: 이미지 URL 포함 검색
func tavilySearchImages(apiKey, query string, maxItems int) ([]string, bool) {
	payload := map[string]any{
		"api_key":        apiKey,
		"query":          query,
		"max_results":    maxItems,
		"search_depth":   "basic",
		"include_images": true,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", "https://api.tavily.com/search", bytes.NewReader(body))
	if err != nil {
		return nil, false
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return nil, false
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	var data struct {
		Images []string `json:"images"`
	}
	if json.Unmarshal(raw, &data) != nil || len(data.Images) == 0 {
		return nil, false
	}
	return data.Images, true
}

// 특정 도메인에서만 검색 (include_domains 사용)
func tavilySearchDomain(apiKey, query string, maxItems int, domain string) (tavilyResult, bool) {
	payload := map[string]any{
		"api_key":      apiKey,
		"query":        query,
		"max_results":  maxItems,
		"search_depth": "basic",
	}
	if domain != "" {
		// 도메인 지정 시 topic/days 제거 — 조합하면 결과 0개 발생
		payload["include_domains"] = []string{domain}
	} else if isNewsQuery(query) {
		// 일반 뉴스 검색만 최근 7일 적용 (3일은 너무 좁음)
		payload["days"] = 7
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", "https://api.tavily.com/search", bytes.NewReader(body))
	if err != nil {
		return tavilyResult{}, false
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return tavilyResult{}, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return tavilyResult{}, false
	}
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	var data struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
		} `json:"results"`
		Answer string `json:"answer"`
	}
	if json.Unmarshal(raw, &data) != nil || len(data.Results) == 0 {
		return tavilyResult{}, false
	}

	// 블로그 쓰레기 패턴 제거용 정규식
	urlPattern := regexp.MustCompile(`https?://\S+`)
	junkPattern := regexp.MustCompile(`(URL 복사|이웃추가|공유하기|신고하기|본문 기타|카테고리 이동|ALL DAY|프로파일|뽕개|\d{4}\.\s*\d{1,2}\.\s*\d{1,2}\.?\s*\d{1,2}:\d{2})`)

	// 봇 차단 징후 — 이 패턴이 content에 포함된 결과는 제외
	antiBotSignals := []string{
		"Access Denied", "403 Forbidden", "Bot detected", "CAPTCHA", "captcha",
		"자동화된 접근", "비정상적인 트래픽", "Blocked", "cf-browser-verification",
		"Ray ID", "인증이 필요합니다", "보안 문자", "로봇이 아님을 확인",
		"Too Many Requests", "429", "Service Unavailable",
	}
	isBotBlocked := func(content string) bool {
		lower := strings.ToLower(content)
		for _, sig := range antiBotSignals {
			if strings.Contains(lower, strings.ToLower(sig)) {
				return true
			}
		}
		// 컨텐츠가 50자 미만이면 실질적 내용 없음 → 봇차단 가능성
		if len([]rune(strings.TrimSpace(content))) < 50 {
			return true
		}
		return false
	}

	items := make([]map[string]string, 0, len(data.Results))
	contentLines := make([]string, 0, len(data.Results))
	for _, r := range data.Results {
		item := map[string]string{"title": r.Title, "url": r.URL}

		if isBotBlocked(r.Content) {
			items = append(items, item)
			continue
		}
		// URL·블로그 메타 제거 후 핵심 텍스트만 추출
		snippet := urlPattern.ReplaceAllString(r.Content, "")
		snippet = junkPattern.ReplaceAllString(snippet, "")
		snippet = strings.Join(strings.Fields(snippet), " ")

		// 파일 저장용: 300자까지 content에 포함
		if len([]rune(snippet)) > 300 {
			runes := []rune(snippet)
			snippet = string(runes[:300]) + "..."
		}
		if snippet != "" {
			item["content"] = snippet
			contentLines = append(contentLines, fmt.Sprintf("• %s: %s", r.Title, snippet))
		}
		items = append(items, item)
	}

	// Tavily answer가 있으면 그걸 우선 (이미 자연어 요약)
	// 없으면 정제된 본문만 전달 (Groq가 최종 정제)
	summary := data.Answer
	if summary == "" {
		summary = strings.Join(contentLines, "\n")
	}

	return tavilyResult{Summary: summary, Items: items}, true
}

// ddgSimpleResult: DuckDuckGo 결과 내부 타입 (Windows/Mac 공통)
type ddgSimpleResult struct {
	Title   string
	URL     string
	Content string
}

// searchDDGSimple: DuckDuckGo instant API — OS 무관 공통 구현
func searchDDGSimple(query string, limit int) []ddgSimpleResult {
	params := url.Values{}
	params.Set("q", query)
	params.Set("format", "json")
	params.Set("no_html", "1")
	apiURL := "https://api.duckduckgo.com/?" + params.Encode()
	client := &http.Client{Timeout: 8 * time.Second}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/124.0.0.0 Safari/537.36")
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
	if json.Unmarshal(body, &raw) != nil {
		return nil
	}
	var results []ddgSimpleResult
	if raw.AbstractText != "" && raw.AbstractURL != "" {
		results = append(results, ddgSimpleResult{Title: raw.AbstractTitle, URL: raw.AbstractURL, Content: raw.AbstractText})
	}
	for i, t := range raw.RelatedTopics {
		if i+1 >= limit || t.FirstURL == "" {
			break
		}
		title := t.Text
		if idx := strings.Index(title, " - "); idx != -1 {
			title = title[:idx]
		}
		results = append(results, ddgSimpleResult{Title: title, URL: t.FirstURL, Content: t.Text})
	}
	return results
}

// webSearchWithFallback: Tavily → DDG 순서 fallback (OS 무관 공통)
func webSearchWithFallback(tKey, query string, maxItems int) (tavilyResult, bool) {
	// 1차: Tavily
	if tKey != "" {
		if tr, ok := tavilySearch(tKey, query, maxItems); ok && len(tr.Items) > 0 {
			return tr, true
		}
	}
	// 2차: DuckDuckGo (공통 구현 사용)
	ddg := searchDDGSimple(query, maxItems)
	if len(ddg) > 0 {
		items := make([]map[string]string, 0, len(ddg))
		var lines []string
		for _, r := range ddg {
			item := map[string]string{"title": r.Title, "url": r.URL}
			if r.Content != "" {
				item["content"] = r.Content
				lines = append(lines, "• "+r.Title+": "+r.Content)
			}
			items = append(items, item)
		}
		return tavilyResult{Summary: strings.Join(lines, "\n"), Items: items}, true
	}
	return tavilyResult{}, false
}

// handleVideoQuickSearch: 카테고리 미리보기용 YouTube/TikTok 영상 빠른 검색
// Mac/Windows 모두 동작 (Tavily API만 사용, chromedp 없음)
func handleVideoQuickSearch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query    string `json:"query"`
		Platform string `json:"platform"` // "youtube"|"tiktok"|"instagram"|"x"|"all"
		MaxItems int    `json:"max_items"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	lang := getLang(r)
	if req.Query == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("query 필요", "query is required", lang)})
		return
	}
	if req.MaxItems == 0 {
		req.MaxItems = 5
	}

	llmMu.RLock()
	tKey := llmTavilyKey
	llmMu.RUnlock()

	enc := strings.ReplaceAll(req.Query, " ", "%20")
	var items []map[string]string

	p := req.Platform
	searchYouTube  := p == "" || p == "youtube"  || p == "all"
	searchTikTok   := p == "tiktok"   || p == "all"
	searchInstagram := p == "instagram" || p == "all"
	searchX        := p == "x"        || p == "all"

	keywords := queryKeywords(req.Query)

	if searchYouTube && tKey != "" {
		if tr, ok := tavilySearchDomain(tKey, req.Query, req.MaxItems, "youtube.com"); ok {
			for _, it := range tr.Items {
				u := it["url"]
				if (strings.Contains(u, "youtube.com/watch") || strings.Contains(u, "youtu.be/")) &&
					titleMatchesQuery(it["title"], keywords) {
					it["type"] = "video"
					it["platform"] = "youtube"
					items = append(items, it)
				}
			}
		}
	}

	if searchTikTok && tKey != "" {
		if tr, ok := tavilySearchDomain(tKey, req.Query, req.MaxItems, "tiktok.com"); ok {
			for _, it := range tr.Items {
				u := it["url"]
				if strings.Contains(u, "tiktok.com/@") && strings.Contains(u, "/video/") &&
					titleMatchesQuery(it["title"], keywords) {
					it["type"] = "video"
					it["platform"] = "tiktok"
					items = append(items, it)
				}
			}
		}
	}

	if searchInstagram && tKey != "" {
		if tr, ok := tavilySearchDomain(tKey, req.Query, req.MaxItems, "instagram.com"); ok {
			for _, it := range tr.Items {
				u := it["url"]
				if (strings.Contains(u, "instagram.com/p/") || strings.Contains(u, "instagram.com/reel/")) &&
					titleMatchesQuery(it["title"], keywords) {
					it["type"] = "social"
					it["platform"] = "instagram"
					items = append(items, it)
				}
			}
		}
	}

	if searchX && tKey != "" {
		for _, domain := range []string{"x.com", "twitter.com"} {
			if tr, ok := tavilySearchDomain(tKey, req.Query, req.MaxItems, domain); ok {
				for _, it := range tr.Items {
					u := it["url"]
					if strings.Contains(u, "/status/") && titleMatchesQuery(it["title"], keywords) {
						it["type"] = "social"
						it["platform"] = "x"
						items = append(items, it)
					}
				}
			}
		}
	}

	// fallback: 결과 없을 때만 검색 링크 제공
	if len(items) == 0 && searchYouTube {
		items = append(items, map[string]string{
			"title": "YouTube: " + req.Query, "url": "https://www.youtube.com/results?search_query=" + enc,
			"type": "video", "platform": "youtube",
		})
	}
	if len(items) == 0 && searchTikTok {
		items = append(items, map[string]string{
			"title": "TikTok: " + req.Query, "url": "https://www.tiktok.com/search?q=" + enc,
			"type": "video", "platform": "tiktok",
		})
	}
	if len(items) == 0 && searchInstagram {
		items = append(items, map[string]string{
			"title": "Instagram: " + req.Query, "url": "https://www.instagram.com/explore/tags/" + enc,
			"type": "social", "platform": "instagram",
		})
	}
	if len(items) == 0 && searchX {
		items = append(items, map[string]string{
			"title": "X: " + req.Query, "url": "https://x.com/search?q=" + enc,
			"type": "social", "platform": "x",
		})
	}

	json200(w, map[string]any{"success": true, "items": items, "total": len(items)})
}
