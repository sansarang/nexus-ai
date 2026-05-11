//go:build windows

package main

import (
	"sync"
	"time"
)

// browserParallelScrape: Windows에서 여러 사이트를 goroutine으로 동시 크롤링
// Google, Naver, YouTube를 병렬 실행 후 결과 합산
func browserParallelScrape(query string, maxItems int) []map[string]string {
	ctx, cancel, err := withStealthBrowserTimeout(30 * time.Second)
	if err != nil {
		return nil
	}
	defer cancel()

	type siteResult struct {
		items []map[string]string
	}

	sites := []string{"google", "naver"}
	ch := make(chan siteResult, len(sites))
	var wg sync.WaitGroup

	perSite := maxItems/len(sites) + 1

	for _, site := range sites {
		wg.Add(1)
		go func(s string) {
			defer wg.Done()
			items, _ := scrapeSearchResults(ctx, query, s, perSite)
			if len(items) > 0 {
				// scrapeSearchResults 반환 타입(map)을 통일
				normalized := make([]map[string]string, 0, len(items))
				for _, it := range items {
					item := map[string]string{
						"title": it["name"],
						"url":   it["url"],
					}
					if item["title"] == "" {
						item["title"] = it["title"]
					}
					if item["url"] != "" {
						normalized = append(normalized, item)
					}
				}
				ch <- siteResult{items: normalized}
			}
		}(site)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	var all []map[string]string
	for r := range ch {
		all = append(all, r.items...)
	}
	return all
}
