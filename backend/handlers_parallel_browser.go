//go:build windows

package main

import (
	"sync"
	"time"
)

// browserParallelScrape: 카테고리에 맞는 실제 사이트를 스텔스 브라우저로 병렬 크롤링
func browserParallelScrape(query string, maxItems int) []map[string]string {
	if !isChromeInstalled() {
		return nil
	}

	ctx, cancel, err := withStealthBrowserTimeout(45 * time.Second)
	if err != nil {
		return nil
	}
	defer cancel()

	cat := detectCategory(query)

	// 카테고리별 직접 스크래핑 대상 사이트
	type siteJob struct{ site, src string }
	var jobs []siteJob

	switch cat {
	case catShopping:
		jobs = []siteJob{
			{"coupang.com", "shop"},
			{"shopping.naver.com", "shop"},
		}
	case catFood:
		jobs = []siteJob{
			{"place.naver.com", "food"},
			{"baemin.com", "food"},
		}
	case catEntertainment:
		jobs = []siteJob{
			{"youtube.com", "youtube"},
			{"tiktok.com", "tiktok"},
		}
	case catTravel:
		jobs = []siteJob{
			{"yanolja.com", "travel"},
			{"naver.com", "travel"},
		}
	case catRealEstate:
		jobs = []siteJob{
			{"land.naver.com", "realestate"},
			{"zigbang.com", "realestate"},
		}
	case catNews:
		jobs = []siteJob{
			{"naver.com", "news"},
		}
	default:
		// 일반 검색: Google + Naver
		jobs = []siteJob{
			{"google.com", "web"},
			{"naver.com", "web"},
		}
	}

	ch := make(chan []map[string]string, len(jobs))
	var wg sync.WaitGroup
	perSite := maxItems/len(jobs) + 2

	for _, job := range jobs {
		job := job
		wg.Add(1)
		go func() {
			defer wg.Done()
			items, err := scrapeSearchResults(ctx, query, job.site, perSite)
			if err != nil || len(items) == 0 {
				return
			}
			// title/url 필드 통일
			normalized := make([]map[string]string, 0, len(items))
			for _, it := range items {
				item := map[string]string{
					"title":  it["name"],
					"url":    it["link"],
					"price":  it["price"],
					"source": job.src,
				}
				if item["title"] == "" {
					item["title"] = it["title"]
				}
				if item["url"] == "" {
					item["url"] = it["url"]
				}
				if item["url"] != "" && item["title"] != "" {
					normalized = append(normalized, item)
				}
			}
			if len(normalized) > 0 {
				ch <- normalized
			}
		}()
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	var all []map[string]string
	for r := range ch {
		all = append(all, r...)
	}
	return all
}
