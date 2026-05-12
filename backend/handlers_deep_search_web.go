package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// POST /api/llm/deep-search-web
// 여러 소스를 병렬로 검색하고 AI로 통합 요약
func handleLLMDeepSearchWeb(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query      string `json:"query"`
		MaxResults int    `json:"max_results"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Query == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "query 필요"})
		return
	}
	if req.MaxResults == 0 {
		req.MaxResults = 10
	}

	llmMu.RLock()
	tKey := llmTavilyKey
	gKey := llmPerplexityKey
	llmMu.RUnlock()

	type source struct {
		name    string
		summary string
		items   []map[string]string
	}

	ch := make(chan source, 5)
	var wg sync.WaitGroup

	// ── 소스 1: Tavily 일반 검색 ──────────────────────────────
	if tKey != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if r, ok := tavilySearch(tKey, req.Query, req.MaxResults); ok {
				ch <- source{name: "web", summary: r.Summary, items: r.Items}
			}
		}()
	}

	// ── 소스 2: Tavily 뉴스 검색 ─────────────────────────────
	if tKey != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			newsQuery := req.Query + " 최신 뉴스"
			if r, ok := tavilySearch(tKey, newsQuery, req.MaxResults/2+1); ok {
				ch <- source{name: "news", summary: r.Summary, items: r.Items}
			}
		}()
	}

	// ── 소스 3: YouTube 영상 검색 ────────────────────────────
	if tKey != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ytQuery := req.Query + " site:youtube.com"
			if r, ok := tavilySearchDomain(tKey, req.Query, req.MaxResults/2+2, "youtube.com"); ok {
				_ = ytQuery
				ch <- source{name: "youtube", items: r.Items}
			}
		}()
	}

	// ── 소스 4: 네이버 TV·VOD 검색 ──────────────────────────
	if tKey != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if r, ok := tavilySearchDomain(tKey, req.Query, req.MaxResults/2+2, "tv.naver.com"); ok {
				ch <- source{name: "video", items: r.Items}
			}
		}()
	}

	// ── 소스 5: 플랫폼 브라우저 검색 ────────────────────────
	wg.Add(1)
	go func() {
		defer wg.Done()
		items := browserParallelScrape(req.Query, req.MaxResults)
		if len(items) > 0 {
			ch <- source{name: "browser", items: items}
		}
	}()

	go func() {
		wg.Wait()
		close(ch)
	}()

	// ── 결과 수집 + URL 중복 제거 + 타입 태깅 ───────────────
	seen := map[string]bool{}
	var allItems []map[string]string
	var summaries []string

	for s := range ch {
		if s.summary != "" {
			summaries = append(summaries, s.summary)
		}
		for _, item := range s.items {
			url := item["url"]
			if url == "" || seen[url] {
				continue
			}
			seen[url] = true
			item["source"] = s.name
			// 타입 자동 태깅
			item["type"] = classifyItemType(url, s.name)
			allItems = append(allItems, item)
		}
	}

	// ── 폴백: 검색 엔진 URL ─────────────────────────────────
	if len(allItems) == 0 {
		allItems = buildFallbackURLs(req.Query, "auto")
	}

	// ── AI 통합 요약 (환각 방지 + 출력 형식 강제) ──────────────
	var finalSummary string
	if gKey != "" && len(allItems) > 0 {
		// 제목만 추출 (URL 전달 금지)
		titleLines := make([]string, 0, len(allItems))
		for i, it := range allItems {
			if i >= 10 { break }
			if t := it["title"]; t != "" {
				titleLines = append(titleLines, fmt.Sprintf("• %s", t))
			}
		}
		today := time.Now().Format("2006-01-02")
		cat := detectCategory(req.Query)
		sysMsg := fmt.Sprintf(`당신은 Nexus AI 한국어 비서입니다.

[규칙]
1. URL, 링크, 출처명 절대 포함 금지
2. 검색 결과에 없는 내용 추측 금지
3. 자연스러운 한국어 3~5문장으로 답변
4. 마크다운 헤더(##), 불릿, 이모지 금지
5. 결과가 부족하면 공식 사이트 안내 (절대 "모른다"로 끝내지 말 것)

[결과 부족 시 안내]
%s`, buildOfficialSiteHint(cat))
		userMsg := fmt.Sprintf("오늘: %s\n질문: \"%s\"\n검색된 콘텐츠 제목:\n%s\n\n위 검색 결과만 근거로 질문에 직접 답하세요. 결과에 없는 내용은 절대 추측하지 마세요.",
			today, req.Query, strings.Join(titleLines, "\n"))
		msgs := []groqMsg{
			{Role: "system", Content: sysMsg},
			{Role: "user", Content: userMsg},
		}
		finalSummary, _, _ = callGroq(gKey, groqChatModel, msgs, 512, false)
	}

	if finalSummary == "" {
		if len(allItems) > 0 {
			finalSummary = fmt.Sprintf("검색 결과 %d개를 찾았습니다. 오른쪽 미리보기 버튼으로 직접 확인해보세요.", len(allItems))
		} else {
			finalSummary = fmt.Sprintf(`"%s"에 대한 검색 결과를 찾지 못했습니다. 검색어를 바꿔서 다시 시도해보세요.`, req.Query)
		}
	}

	json200(w, map[string]any{
		"success": true,
		"query":   req.Query,
		"summary": finalSummary,
		"items":   allItems,
		"total":   len(allItems),
	})
}

// buildFallbackURLs: 실제 검색 결과가 없을 때 카테고리별 최적 링크 반환
func buildFallbackURLs(query, site string) []map[string]string {
	enc := urlEncode(query)

	// 특정 사이트 지정된 경우 우선 처리
	switch strings.ToLower(site) {
	case "coupang":
		return []map[string]string{
			{"title": "쿠팡 검색: " + query, "url": "https://www.coupang.com/np/search?q=" + enc},
			{"title": "네이버쇼핑: " + query, "url": "https://search.shopping.naver.com/search/all?query=" + enc},
		}
	case "youtube":
		return []map[string]string{
			{"title": "YouTube 검색: " + query, "url": "https://www.youtube.com/results?search_query=" + enc},
		}
	case "naver":
		return []map[string]string{
			{"title": "네이버 검색: " + query, "url": "https://search.naver.com/search.naver?query=" + enc},
		}
	case "temu":
		return []map[string]string{
			{"title": "테무 검색: " + query, "url": "https://www.temu.com/search_result.html?search_key=" + enc},
		}
	case "danawa":
		return []map[string]string{
			{"title": "다나와 검색: " + query, "url": "https://search.danawa.com/dsearch.php?query=" + enc},
		}
	case "gmarket":
		return []map[string]string{
			{"title": "지마켓 검색: " + query, "url": "https://search.gmarket.co.kr/search?keyword=" + enc},
		}
	case "11st":
		return []map[string]string{
			{"title": "11번가 검색: " + query, "url": "https://search.11st.co.kr/Search.tmall?kwd=" + enc},
		}
	}

	// 카테고리 자동 감지 → 최적 사이트 링크
	cat := detectCategory(query)
	return categoryFallbackSites(query, cat)
}

// fetchPageContent: URL의 본문 텍스트를 가져옴 (딥서치용)
func fetchPageContent(url string, maxBytes int) string {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil || resp.StatusCode != 200 {
		return ""
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, int64(maxBytes)))
	// HTML 태그 제거 (간단)
	text := string(raw)
	var sb strings.Builder
	inTag := false
	for _, c := range text {
		if c == '<' {
			inTag = true
		} else if c == '>' {
			inTag = false
		} else if !inTag {
			sb.WriteRune(c)
		}
	}
	result := strings.Join(strings.Fields(sb.String()), " ")
	if len(result) > 800 {
		result = result[:800]
	}
	return result
}
