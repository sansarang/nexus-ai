//go:build windows

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

// crawlSiteForItems: Windows — 스텔스 브라우저로 봇 우회 크롤링
// Chrome/Edge 미설치 시 httpCrawlSite로 폴백
func crawlSiteForItems(site, query string, maxItems int) []map[string]string {
	if !isChromeInstalled() {
		return httpCrawlSite(site, query, maxItems)
	}

	ctx, cancel, err := withStealthBrowserTimeout(30 * time.Second)
	if err != nil {
		return httpCrawlSite(site, query, maxItems)
	}
	defer cancel()

	profile, _ := getSiteProfile(site)
	searchURL := buildSearchURL(site, query)

	// 쿠키 복원 (이전 세션 지속성)
	_ = loadCookies(ctx, site)

	// 페이지 이동
	if err := chromedp.Run(ctx,
		chromedp.Navigate(searchURL),
		humanDelay(800, 1500),
	); err != nil {
		return httpCrawlSite(site, query, maxItems)
	}

	// 봇 차단 감지
	if blocked, _ := detectAntiBot(ctx); blocked {
		return stealthFallbackLink(site, query)
	}

	// 페이지 안정화 대기
	_ = waitForPageStable(ctx)
	chromedp.Run(ctx, chromedp.Sleep(profile.WaitAfterSearch))

	// 자연스러운 스크롤
	_ = humanScroll(ctx, 400)

	// 상품/결과 추출
	items, err := extractStructuredProducts(ctx, profile, maxItems)
	if err != nil || len(items) == 0 {
		// DOM 추출 실패 시 링크 텍스트 기반 추출
		items = stealthExtractLinks(ctx, site, query, maxItems)
	}

	// 쿠키 저장
	_ = saveCookies(ctx, site)

	if len(items) == 0 {
		return stealthFallbackLink(site, query)
	}
	return items
}

// stealthExtractLinks: 현재 페이지에서 JS로 링크+텍스트 추출
func stealthExtractLinks(ctx context.Context, site, query string, maxItems int) []map[string]string {
	detailPat := siteDetailPatterns[site]
	siteKey := strings.Split(site, ".")[0]
	filterPat := detailPat
	if filterPat == "" {
		filterPat = "/" + siteKey + "/"
	}

	extractJS := fmt.Sprintf(`
	JSON.stringify(
		Array.from(document.querySelectorAll('a[href]'))
			.filter(a => a.href && a.href.includes(%q) && a.innerText && a.innerText.trim().length > 3)
			.slice(0, %d)
			.map(a => ({ name: a.innerText.trim().slice(0, 80), link: a.href }))
	)`, filterPat, maxItems*3)

	var raw string
	tCtx, tCancel := context.WithTimeout(ctx, 8*time.Second)
	defer tCancel()

	if err := chromedp.Run(tCtx, chromedp.Evaluate(extractJS, &raw)); err != nil || raw == "" {
		return nil
	}

	type linkItem struct {
		Name string `json:"name"`
		Link string `json:"link"`
	}
	var parsed []linkItem
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil
	}

	queryWords := queryKeywords(query)
	var items []map[string]string
	seen := map[string]bool{}
	for _, p := range parsed {
		if seen[p.Link] || p.Name == "" || p.Link == "" {
			continue
		}
		if !titleMatchesQuery(p.Name, queryWords) {
			continue
		}
		seen[p.Link] = true
		items = append(items, map[string]string{"name": p.Name, "link": p.Link, "price": "", "site": site})
		if len(items) >= maxItems {
			break
		}
	}
	return items
}

// stealthFallbackLink: 크롤링 실패 시 검색 링크 제공
func stealthFallbackLink(site, query string) []map[string]string {
	enc := strings.ReplaceAll(query, " ", "+")
	url := buildSearchURL(site, query)
	return []map[string]string{
		{"name": site + "에서 '" + query + "' 검색하기", "link": url, "price": "", "site": site},
		{"name": "네이버쇼핑에서 '" + query + "' 최저가 비교", "link": "https://search.shopping.naver.com/search/all?query=" + enc, "price": "", "site": "shopping.naver.com"},
	}
}
