//go:build windows

package main

// crawlSiteForItems: Windows — HTTP GET 폴백 (chromedp는 별도 프로세스)
func crawlSiteForItems(site, query string, maxItems int) []map[string]string {
	return httpCrawlSite(site, query, maxItems)
}
