package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// POST /api/site-search
// LLM 라우팅 없이 Tavily로 사이트별 직접 검색 → 항상 링크 목록 반환
func handleSiteSearch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query    string `json:"query"`
		Site     string `json:"site"`     // e.g. "daangn.com"
		MaxItems int    `json:"max_items"` // default 8
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Query == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "query 필요"})
		return
	}
	if req.MaxItems == 0 {
		req.MaxItems = 8
	}

	llmMu.RLock()
	tKey := llmTavilyKey
	llmMu.RUnlock()

	searchQuery := req.Query
	if req.Site != "" {
		searchQuery = "site:" + req.Site + " " + req.Query
	}

	var items []map[string]string

	if tKey != "" {
		// 1차: include_domains로 해당 사이트만 정밀 검색
		if req.Site != "" {
			if tr, ok := tavilySearchDomain(tKey, req.Query, req.MaxItems, req.Site); ok {
				for _, it := range tr.Items {
					if strings.Contains(it["url"], strings.Split(req.Site, ".")[0]) {
						items = append(items, map[string]string{
							"name": it["title"], "link": it["url"], "price": "", "site": req.Site,
						})
					}
				}
			}
		}
		// 2차: site: 접두어 방식
		if len(items) == 0 {
			if tr, ok := tavilySearch(tKey, searchQuery, req.MaxItems); ok {
				for _, it := range tr.Items {
					if req.Site == "" || strings.Contains(it["url"], strings.Split(req.Site, ".")[0]) {
						items = append(items, map[string]string{
							"name": it["title"], "link": it["url"], "price": "", "site": req.Site,
						})
					}
				}
				if len(items) == 0 {
					for _, it := range tr.Items {
						items = append(items, map[string]string{
							"name": it["title"], "link": it["url"], "price": "", "site": req.Site,
						})
					}
				}
			}
		}
	}

	// 3차: 브라우저로 실제 사이트 크롤링
	if len(items) == 0 && req.Site != "" {
		items = crawlSiteForItems(req.Site, req.Query, req.MaxItems)
	}

	// 결과 없으면 해당 사이트 검색 페이지 링크 제공
	if len(items) == 0 {
		enc := strings.ReplaceAll(req.Query, " ", "+")
		siteURL := ""
		switch req.Site {
		case "daangn.com":
			siteURL = fmt.Sprintf("https://www.daangn.com/search/%s", enc)
		case "bunjang.co.kr":
			siteURL = fmt.Sprintf("https://m.bunjang.co.kr/search/products?q=%s", enc)
		case "joongna.com":
			siteURL = fmt.Sprintf("https://web.joongna.com/search/%s", enc)
		case "encar.com":
			siteURL = fmt.Sprintf("https://www.encar.com/search/car?searchKey=%s", enc)
		case "heydealer.com":
			siteURL = fmt.Sprintf("https://www.heydealer.com/car/search?keyword=%s", enc)
		case "kbchachacha.com":
			siteURL = fmt.Sprintf("https://www.kbchachacha.com/public/car/list.kbc?keyword=%s", enc)
		case "coupang.com":
			siteURL = fmt.Sprintf("https://www.coupang.com/np/search?q=%s", enc)
		case "shopping.naver.com":
			siteURL = fmt.Sprintf("https://search.shopping.naver.com/search/all?query=%s", enc)
		case "musinsa.com":
			siteURL = fmt.Sprintf("https://www.musinsa.com/search/musinsa/integration?q=%s", enc)
		case "zigbang.com":
			siteURL = fmt.Sprintf("https://www.zigbang.com/search?q=%s", enc)
		case "yanolja.com":
			siteURL = fmt.Sprintf("https://www.yanolja.com/keyword/%s", enc)
		case "danawa.com":
			siteURL = fmt.Sprintf("https://search.danawa.com/dsearch.php?query=%s", enc)
		default:
			if req.Site != "" {
				siteURL = fmt.Sprintf("https://www.%s/search?q=%s", req.Site, enc)
			} else {
				siteURL = fmt.Sprintf("https://search.naver.com/search.naver?query=%s", enc)
			}
		}
		siteName := req.Site
		if siteName == "" {
			siteName = "검색"
		}
		items = []map[string]string{
			{"name": fmt.Sprintf("%s 검색: %s", siteName, req.Query), "link": siteURL, "price": "", "site": req.Site},
		}
	}

	summary := fmt.Sprintf("%s에서 \"%s\" 결과 %d개", req.Site, req.Query, len(items))
	json200(w, map[string]any{
		"success": true,
		"query":   req.Query,
		"site":    req.Site,
		"summary": summary,
		"results": items,
		"total":   len(items),
	})
}

// 사이트별 검색 URL 구성 → HTTP 크롤링 → 실제 링크 추출
func crawlSiteForItems(site, query string, maxItems int) []map[string]string {
	enc := strings.ReplaceAll(query, " ", "+")
	searchURLs := map[string]string{
		"heydealer.com":       fmt.Sprintf("https://www.heydealer.com/car/search?keyword=%s", enc),
		"encar.com":           fmt.Sprintf("https://www.encar.com/search/car?searchKey=%s", enc),
		"kbchachacha.com":     fmt.Sprintf("https://www.kbchachacha.com/public/car/list.kbc?keyword=%s", enc),
		"bobaedream.co.kr":    fmt.Sprintf("https://www.bobaedream.co.kr/search?search_params=%s", enc),
		"daangn.com":          fmt.Sprintf("https://www.daangn.com/search/%s", strings.ReplaceAll(query, " ", "%%20")),
		"bunjang.co.kr":       fmt.Sprintf("https://m.bunjang.co.kr/search/products?q=%s", enc),
		"joongna.com":         fmt.Sprintf("https://web.joongna.com/search/%s", enc),
		"coupang.com":         fmt.Sprintf("https://www.coupang.com/np/search?q=%s", enc),
		"shopping.naver.com":  fmt.Sprintf("https://search.shopping.naver.com/search/all?query=%s", enc),
		"11st.co.kr":          fmt.Sprintf("https://search.11st.co.kr/Search.tmall?kwd=%s", enc),
		"gmarket.co.kr":       fmt.Sprintf("https://search.gmarket.co.kr/search?keyword=%s", enc),
		"musinsa.com":         fmt.Sprintf("https://www.musinsa.com/search/musinsa/integration?q=%s", enc),
		"danawa.com":          fmt.Sprintf("https://search.danawa.com/dsearch.php?query=%s", enc),
		"yanolja.com":         fmt.Sprintf("https://www.yanolja.com/keyword/%s", enc),
		"goodchoice.kr":       fmt.Sprintf("https://www.goodchoice.kr/product/search?keyword=%s", enc),
		"zigbang.com":         fmt.Sprintf("https://www.zigbang.com/search?q=%s", enc),
		"dabangapp.com":       fmt.Sprintf("https://www.dabangapp.com/map/oneroom?search_type=keyword&keyword=%s", enc),
		"temu.com":            fmt.Sprintf("https://www.temu.com/search_result.html?search_key=%s", enc),
		"aliexpress.com":      fmt.Sprintf("https://www.aliexpress.com/wholesale?SearchText=%s", enc),
		"amazon.com":          fmt.Sprintf("https://www.amazon.com/s?k=%s", enc),
	}

	searchURL, ok := searchURLs[site]
	if !ok {
		searchURL = fmt.Sprintf("https://www.%s/search?q=%s", site, enc)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil
	}
	// 브라우저처럼 보이는 헤더
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xhtml+xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "ko-KR,ko;q=0.9,en-US;q=0.8")
	req.Header.Set("Referer", "https://www.google.com/")

	resp, err := client.Do(req)
	if err != nil || resp.StatusCode >= 400 {
		return nil
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}
	html := string(bodyBytes)

	// <a href="...">...</a> 패턴에서 해당 사이트 내부 링크 추출
	linkRe := regexp.MustCompile(`(?i)<a[^>]+href="(https?://[^"]*` + regexp.QuoteMeta(strings.Split(site, ".")[0]) + `[^"]*)"[^>]*>([^<]{3,80})</a>`)
	matches := linkRe.FindAllStringSubmatch(html, maxItems*3)

	seen := map[string]bool{}
	var items []map[string]string
	for _, m := range matches {
		link := m[1]
		title := strings.TrimSpace(m[2])
		// 불필요한 nav/메뉴 링크 제거
		if seen[link] || len(title) < 4 || strings.Contains(strings.ToLower(title), "javascript") {
			continue
		}
		seen[link] = true
		items = append(items, map[string]string{
			"name": title, "link": link, "price": "", "site": site,
		})
		if len(items) >= maxItems {
			break
		}
	}

	// 링크 추출 실패 시 검색 페이지 자체를 링크로 제공
	if len(items) == 0 {
		items = []map[string]string{
			{"name": fmt.Sprintf("%s에서 \"%s\" 직접 검색", site, query), "link": searchURL, "price": "", "site": site},
		}
	}
	return items
}
