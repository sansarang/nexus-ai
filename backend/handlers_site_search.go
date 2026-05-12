package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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
		if tr, ok := tavilySearch(tKey, searchQuery, req.MaxItems); ok {
			sitePart := ""
			if req.Site != "" {
				sitePart = strings.Split(req.Site, ".")[0]
			}
			for _, it := range tr.Items {
				if sitePart == "" || strings.Contains(it["url"], sitePart) {
					items = append(items, map[string]string{
						"name":  it["title"],
						"link":  it["url"],
						"price": "",
						"site":  req.Site,
					})
				}
			}
			// 사이트 필터 후 결과 없으면 전체 사용
			if len(items) == 0 {
				for _, it := range tr.Items {
					items = append(items, map[string]string{
						"name":  it["title"],
						"link":  it["url"],
						"price": "",
						"site":  req.Site,
					})
				}
			}
		}
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
