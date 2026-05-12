//go:build !windows

package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

// crawlSiteForItems: Mac — chromedp로 JS 렌더링 후 상세페이지 링크 추출
func crawlSiteForItems(site, query string, maxItems int) []map[string]string {
	searchURL := buildSearchURL(site, query)
	detailPattern := siteDetailPatterns[site]

	ctx, cancel, err := getBrowserCtxMac()
	if err == nil {
		defer cancel()
		if chromedp.Run(ctx,
			chromedp.Navigate(searchURL),
			chromedp.Sleep(3*time.Second),
		) == nil {
			var jsScript string
			if detailPattern != "" {
				jsScript = fmt.Sprintf(`
JSON.stringify(Array.from(document.querySelectorAll('a[href]'))
  .filter(a => a.href.includes('%s') && a.offsetParent !== null)
  .slice(0, %d)
  .map(a => ({
    title: (a.innerText || a.title || a.getAttribute('aria-label') || '').trim().replace(/\s+/g,' ').slice(0,80),
    link: a.href
  }))
  .filter(x => x.title.length > 1))`, detailPattern, maxItems*2)
			} else {
				jsScript = fmt.Sprintf(`
JSON.stringify(Array.from(document.querySelectorAll('a[href]'))
  .filter(a => a.href.includes('%s') && a.href.length > 35 && a.offsetParent !== null)
  .slice(0, %d)
  .map(a => ({
    title: (a.innerText || a.title || '').trim().replace(/\s+/g,' ').slice(0,80),
    link: a.href
  }))
  .filter(x => x.title.length > 1))`, strings.Split(site, ".")[0], maxItems*2)
			}
			var raw string
			chromedp.Run(ctx, chromedp.Evaluate(jsScript, &raw))
			var parsed []struct {
				Title string `json:"title"`
				Link  string `json:"link"`
			}
			if json.Unmarshal([]byte(raw), &parsed) == nil && len(parsed) > 0 {
				seen := map[string]bool{}
				var items []map[string]string
				for _, p := range parsed {
					if seen[p.Link] || p.Title == "" {
						continue
					}
					seen[p.Link] = true
					items = append(items, map[string]string{"name": p.Title, "link": p.Link, "price": "", "site": site})
					if len(items) >= maxItems {
						break
					}
				}
				if len(items) > 0 {
					return items
				}
			}
		}
	}
	// chromedp 실패 시 HTTP 폴백
	return httpCrawlSite(site, query, maxItems)
}
