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
	queryWords := queryKeywords(req.Query)
	siteKey := strings.Split(req.Site, ".")[0]

	if tKey != "" {
		// 1차: include_domains로 해당 사이트만 정밀 검색
		if req.Site != "" {
			if tr, ok := tavilySearchDomain(tKey, req.Query, req.MaxItems, req.Site); ok {
				for _, it := range tr.Items {
					if strings.Contains(it["url"], siteKey) && titleMatchesQuery(it["title"], queryWords) {
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
					if (req.Site == "" || strings.Contains(it["url"], siteKey)) && titleMatchesQuery(it["title"], queryWords) {
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

// 사이트별 상세페이지 URL 패턴
var siteDetailPatterns = map[string]string{
	"heydealer.com":      `/car/`,
	"encar.com":          `cardetailview`,
	"kbchachacha.com":    `/public/car/detail`,
	"bobaedream.co.kr":   `/car/`,
	"daangn.com":         `/articles/`,
	"bunjang.co.kr":      `/product/`,
	"joongna.com":        `/product/`,
	"coupang.com":        `/vp/products/`,
	"shopping.naver.com": `/product/`,
	"11st.co.kr":         `/product/`,
	"gmarket.co.kr":      `/goods/`,
	"auction.co.kr":      `/auction/`,
	"musinsa.com":        `/store/goods/`,
	"a-bly.com":          `/products/`,
	"zigzag.kr":          `/catalog/`,
	"ohou.se":            `/productions/`,
	"temu.com":           `/goods.html`,
	"aliexpress.com":     `/item/`,
	"amazon.com":         `/dp/`,
	"ebay.com":           `/itm/`,
	"etsy.com":           `/listing/`,
	"walmart.com":        `/ip/`,
	"target.com":         `/p/`,
	"bestbuy.com":        `/site/`,
	"booking.com":        `/hotel/`,
	"airbnb.com":         `/rooms/`,
	"expedia.com":        `/Hotel-Search`,
	"tripadvisor.com":    `/Hotel_Review`,
	"yelp.com":           `/biz/`,
	"zillow.com":         `/homedetails/`,
	"realtor.com":        `/realestateandhomes-detail/`,
	"imdb.com":           `/title/`,
	"reddit.com":         `/comments/`,
	"github.com":         `/blob/`,
	"stackoverflow.com":  `/questions/`,
	"danawa.com":         `pcode=`,
	"zigbang.com":        `/home/`,
	"dabangapp.com":      `/room/`,
	"yanolja.com":        `/accommodation/`,
	"goodchoice.kr":      `/product/detail`,
	"baemin.com":         `/shop/`,
}

// httpCrawlSite: HTTP GET + regex로 상세 링크 추출 (공통 폴백)
func httpCrawlSite(site, query string, maxItems int) []map[string]string {
	searchURL := buildSearchURL(site, query)
	detailPattern := siteDetailPatterns[site]

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return []map[string]string{{"name": site + "에서 " + query + " 검색", "link": searchURL, "price": "", "site": site}}
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "ko-KR,ko;q=0.9")

	resp, err := client.Do(req)
	if err != nil || resp.StatusCode >= 400 {
		return []map[string]string{{"name": site + "에서 " + query + " 검색", "link": searchURL, "price": "", "site": site}}
	}
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)
	html := string(bodyBytes)

	var linkPat *regexp.Regexp
	if detailPattern != "" {
		linkPat = regexp.MustCompile(`href="(https?://[^"]*` + regexp.QuoteMeta(detailPattern) + `[^"]*)"[^>]*>([^<]{3,60})`)
	} else {
		linkPat = regexp.MustCompile(`href="(https?://[^"]*` + regexp.QuoteMeta(strings.Split(site, ".")[0]) + `[^"]{5,})"[^>]*>([^<]{3,60})`)
	}
	matches := linkPat.FindAllStringSubmatch(html, maxItems*5)
	queryWords := queryKeywords(query)
	seen := map[string]bool{}
	var items []map[string]string
	for _, m := range matches {
		link, title := m[1], strings.TrimSpace(m[2])
		if seen[link] || len(title) < 3 {
			continue
		}
		if !titleMatchesQuery(title, queryWords) {
			continue
		}
		seen[link] = true
		items = append(items, map[string]string{"name": title, "link": link, "price": "", "site": site})
		if len(items) >= maxItems {
			break
		}
	}
	if len(items) == 0 {
		items = []map[string]string{{"name": site + "에서 " + query + " 검색", "link": searchURL, "price": "", "site": site}}
	}
	return items
}
