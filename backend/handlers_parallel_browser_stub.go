//go:build !windows

package main

// browserParallelScrape: Mac에서는 chromedp stub이라 브라우저 크롤링 불가
// tryBrowserSearch 를 래핑해서 반환
func browserParallelScrape(query string, maxItems int) []map[string]string {
	return tryBrowserSearch(query, "auto", maxItems)
}
