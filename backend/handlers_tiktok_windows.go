//go:build windows

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

// ══════════════════════════════════════════════════════════════
//  TikTok 실제 피드·검색 크롤러 (모바일 스텔스 브라우저)
//  withMobileStealthTimeout 사용 — 봇 탐지 우회
// ══════════════════════════════════════════════════════════════

type TikTokItem struct {
	Title    string `json:"title"`
	URL      string `json:"url"`
	Author   string `json:"author"`
	Likes    string `json:"likes"`
	Comments string `json:"comments"`
	Shares   string `json:"shares"`
}

// POST /api/tiktok/search
// body: { "query": "...", "limit": 10 }
func handleTikTokSearch(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Query == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("query 필요", "query required", lang)})
		return
	}
	if req.Limit == 0 || req.Limit > 20 {
		req.Limit = 10
	}

	items, err := crawlTikTokSearch(req.Query, req.Limit)
	if err != nil || len(items) == 0 {
		// Tavily fallback
		llmMu.RLock()
		tKey := llmTavilyKey
		llmMu.RUnlock()
		if tKey != "" {
			if tr, ok := tavilySearchDomain(tKey, req.Query, req.Limit, "tiktok.com"); ok && len(tr.Items) > 0 {
				json200(w, map[string]any{
					"success": true,
					"source":  "search_fallback",
					"items":   tr.Items,
					"count":   len(tr.Items),
					"message": fmt.Sprintf("🎵 TikTok \"%s\" 검색 결과 %d개 (검색 기반)", req.Query, len(tr.Items)),
				})
				return
			}
		}
		msg := "TikTok 크롤링 실패"
		if err != nil {
			msg += ": " + err.Error()
		}
		writeJSON(w, 200, map[string]any{"success": false, "message": msg})
		return
	}

	json200(w, map[string]any{
		"success": true,
		"source":  "browser",
		"items":   items,
		"count":   len(items),
		"message": fmt.Sprintf("🎵 TikTok \"%s\" 검색 결과 %d개", req.Query, len(items)),
	})
}

// GET /api/tiktok/trending
// TikTok 한국 트렌딩 영상 수집
func handleTikTokTrending(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	items, err := crawlTikTokFeed("https://www.tiktok.com/foryou?lang=ko-KR", 15)
	if err != nil || len(items) == 0 {
		// Tavily fallback: 트렌딩 키워드 검색
		llmMu.RLock()
		tKey := llmTavilyKey
		llmMu.RUnlock()
		if tKey != "" {
			if tr, ok := tavilySearchDomain(tKey, "tiktok 인기 영상 트렌딩 korea 2026", 10, "tiktok.com"); ok {
				json200(w, map[string]any{
					"success": true,
					"source":  "search_fallback",
					"items":   tr.Items,
					"message": msgT("🔥 TikTok 트렌딩 (검색 기반)", "🔥 TikTok Trending (search-based)", lang),
				})
				return
			}
		}
		writeJSON(w, 200, map[string]any{
			"success": false,
			"message": msgT("TikTok 트렌딩 수집 실패 (로그인 필요하거나 일시적 차단)", "TikTok trending collection failed (login required or temporarily blocked)", lang),
		})
		return
	}

	json200(w, map[string]any{
		"success": true,
		"source":  "browser",
		"items":   items,
		"count":   len(items),
		"message": fmt.Sprintf("🔥 TikTok 트렌딩 %d개", len(items)),
	})
}

// POST /api/tiktok/profile
// body: { "username": "@username", "limit": 10 }
// 특정 계정의 최근 영상 수집
func handleTikTokProfile(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		Username string `json:"username"`
		Limit    int    `json:"limit"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Username == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("username 필요", "username required", lang)})
		return
	}
	if req.Limit == 0 {
		req.Limit = 10
	}
	username := strings.TrimPrefix(req.Username, "@")
	profileURL := fmt.Sprintf("https://www.tiktok.com/@%s", username)

	items, err := crawlTikTokFeed(profileURL, req.Limit)
	if err != nil || len(items) == 0 {
		writeJSON(w, 200, map[string]any{
			"success": false,
			"message": msgT(fmt.Sprintf("@%s 프로필 수집 실패 (비공개 계정이거나 일시적 오류)", username), fmt.Sprintf("@%s profile collection failed (private account or temporary error)", username), lang),
		})
		return
	}
	json200(w, map[string]any{
		"success": true,
		"items":   items,
		"count":   len(items),
		"message": fmt.Sprintf("👤 @%s 최근 영상 %d개", username, len(items)),
	})
}

// ── 내부 크롤러 ──────────────────────────────────────────────

func crawlTikTokSearch(query string, limit int) ([]TikTokItem, error) {
	searchURL := fmt.Sprintf("https://www.tiktok.com/search/video?q=%s", urlEncode(query))
	return crawlTikTokFeed(searchURL, limit)
}

func crawlTikTokFeed(pageURL string, limit int) ([]TikTokItem, error) {
	ctx, cancel, err := withMobileStealthTimeout(35 * time.Second)
	if err != nil {
		return nil, fmt.Errorf("스텔스 브라우저 시작 실패: %w", err)
	}
	defer cancel()

	// 쿠키 로드 (저장된 경우)
	cookieFile := videoCookiePath("tiktok")
	if fileExists(cookieFile) {
		loadCookieFile(ctx, cookieFile)
	}

	var titles, urls, authors, likes []string

	err = chromedp.Run(ctx,
		chromedp.Navigate(pageURL),
		// TikTok은 JS 렌더링이 느림 — 충분히 대기
		chromedp.Sleep(4*time.Second),
		// 스크롤 다운해서 더 많은 영상 로드
		chromedp.Evaluate(`window.scrollBy(0, 1500)`, nil),
		chromedp.Sleep(2*time.Second),
		chromedp.Evaluate(`window.scrollBy(0, 1500)`, nil),
		chromedp.Sleep(1*time.Second),

		// 영상 제목/설명 (검색 결과 + 피드 둘 다 커버)
		chromedp.Evaluate(`
			(function(){
				var els = document.querySelectorAll(
					'[data-e2e="search-card-desc"] span, '+
					'[data-e2e="video-desc"] span, '+
					'.tiktok-j2DtDd, '+
					'[class*="DivContainer"] p[class*="Text"], '+
					'h1[class*="Title"], '+
					'div[class*="desc"] span'
				);
				return Array.from(els).slice(0,20).map(e=>e.innerText.trim()).filter(t=>t.length>2);
			})()
		`, &titles),

		// 영상 URL
		chromedp.Evaluate(`
			(function(){
				var els = document.querySelectorAll(
					'a[href*="/video/"], '+
					'a[data-e2e="search-card-video-link"], '+
					'a[class*="AVideoContainer"]'
				);
				return Array.from(els).slice(0,20).map(e=>e.href||'').filter(h=>h.includes('/video/'));
			})()
		`, &urls),

		// 작성자
		chromedp.Evaluate(`
			(function(){
				var els = document.querySelectorAll(
					'[data-e2e="search-card-user-unique-id"], '+
					'[data-e2e="user-unique-id"], '+
					'a[href*="/@"] span[class*="UniqueId"], '+
					'p[class*="AuthorTitle"]'
				);
				return Array.from(els).slice(0,20).map(e=>e.innerText.trim());
			})()
		`, &authors),

		// 좋아요 수
		chromedp.Evaluate(`
			(function(){
				var els = document.querySelectorAll(
					'strong[data-e2e="like-count"], '+
					'[class*="LikesCount"], '+
					'[data-e2e="search-card-like-count"]'
				);
				return Array.from(els).slice(0,20).map(e=>e.innerText.trim());
			})()
		`, &likes),
	)

	if err != nil {
		return nil, fmt.Errorf("TikTok 크롤링 오류: %w", err)
	}

	var items []TikTokItem
	max := len(titles)
	if len(urls) < max {
		max = len(urls)
	}
	if max > limit {
		max = limit
	}

	for i := 0; i < max; i++ {
		item := TikTokItem{
			Title: titles[i],
			URL:   urls[i],
		}
		if i < len(authors) {
			item.Author = authors[i]
		}
		if i < len(likes) {
			item.Likes = likes[i]
		}
		if item.Title != "" || item.URL != "" {
			items = append(items, item)
		}
	}

	// URL은 있지만 title 없는 경우 (피드)
	if len(items) == 0 && len(urls) > 0 {
		for i, u := range urls {
			if i >= limit {
				break
			}
			author := ""
			if i < len(authors) {
				author = authors[i]
			}
			items = append(items, TikTokItem{
				URL:    u,
				Title:  fmt.Sprintf("TikTok 영상 #%d", i+1),
				Author: author,
			})
		}
	}

	return items, nil
}

// Netscape 쿠키 파일을 chromedp 쿠키로 로드
func loadCookieFile(ctx context.Context, path string) {
	// yt-dlp Netscape 형식 파싱은 복잡 — 여기선 도메인 쿠키만 간단히 처리
	// 실제 구현에서는 network.SetCookies 사용 가능
	// 현재는 no-op (chromedp session 기반 로그인이 더 안정적)
	_ = ctx
	_ = path
}
