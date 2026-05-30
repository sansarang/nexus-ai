//go:build windows

package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

// ══════════════════════════════════════════════════════════════
//  Reddit 크롤러 (스텔스 브라우저 — 로그인 불필요)
//  withMobileStealthTimeout 사용 — TikTok 방식 동일
// ══════════════════════════════════════════════════════════════

type RedditPost struct {
	Title     string `json:"title"`
	URL       string `json:"url"`
	Subreddit string `json:"subreddit"`
	Author    string `json:"author"`
	Score     string `json:"score"`
	Comments  string `json:"comments"`
	Body      string `json:"body"`
}

// POST /api/reddit/search
// body: { "query": "...", "subreddit": "stocks", "limit": 10, "sort": "hot" }
func handleRedditSearch(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		Query     string `json:"query"`
		Subreddit string `json:"subreddit"`
		Limit     int    `json:"limit"`
		Sort      string `json:"sort"`
	}
	tryDecodeBody(r, &req)
	if req.Query == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("query 필요", "query is required", lang)})
		return
	}
	if req.Limit == 0 || req.Limit > 25 {
		req.Limit = 10
	}
	if req.Sort == "" {
		req.Sort = "relevance"
	}

	posts, err := crawlRedditSearch(req.Query, req.Subreddit, req.Limit, req.Sort)
	if err != nil || len(posts) == 0 {
		// Tavily fallback
		llmMu.RLock()
		tKey := llmTavilyKey
		llmMu.RUnlock()
		query := req.Query
		if req.Subreddit != "" {
			query = fmt.Sprintf("site:reddit.com/r/%s %s", req.Subreddit, req.Query)
		} else {
			query = fmt.Sprintf("site:reddit.com %s", req.Query)
		}
		if tKey != "" {
			if tr, ok := tavilySearchDomain(tKey, query, req.Limit, "reddit.com"); ok && len(tr.Items) > 0 {
				json200(w, map[string]any{
					"success": true,
					"source":  "search_fallback",
					"posts":   tr.Items,
					"count":   len(tr.Items),
					"message": fmt.Sprintf("🔴 Reddit \"%s\" 검색 결과 %d개 (검색 기반)", req.Query, len(tr.Items)),
				})
				return
			}
		}
		msg := msgT("Reddit 크롤링 실패", "Reddit crawl failed", lang)
		if err != nil {
			msg += ": " + err.Error()
		}
		writeJSON(w, 200, map[string]any{"success": false, "message": msg})
		return
	}

	json200(w, map[string]any{
		"success": true,
		"source":  "browser",
		"posts":   posts,
		"count":   len(posts),
		"message": fmt.Sprintf("🔴 Reddit \"%s\" 검색 결과 %d개", req.Query, len(posts)),
	})
}

// GET /api/reddit/trending?subreddit=stocks
func handleRedditTrending(w http.ResponseWriter, r *http.Request) {
	subreddit := r.URL.Query().Get("subreddit")
	if subreddit == "" {
		subreddit = "all"
	}

	pageURL := fmt.Sprintf("https://www.reddit.com/r/%s/hot/", subreddit)
	posts, err := crawlRedditFeed(pageURL, 15)
	if err != nil || len(posts) == 0 {
		llmMu.RLock()
		tKey := llmTavilyKey
		llmMu.RUnlock()
		if tKey != "" {
			query := fmt.Sprintf("site:reddit.com/r/%s hot trending 2026", subreddit)
			if tr, ok := tavilySearchDomain(tKey, query, 10, "reddit.com"); ok && len(tr.Items) > 0 {
				json200(w, map[string]any{
					"success":   true,
					"source":    "search_fallback",
					"subreddit": subreddit,
					"posts":     tr.Items,
					"message":   fmt.Sprintf("🔥 r/%s 트렌딩 (검색 기반)", subreddit),
				})
				return
			}
		}
		writeJSON(w, 200, map[string]any{
			"success": false,
			"message": fmt.Sprintf("r/%s 트렌딩 수집 실패", subreddit),
		})
		return
	}

	json200(w, map[string]any{
		"success":   true,
		"source":    "browser",
		"subreddit": subreddit,
		"posts":     posts,
		"count":     len(posts),
		"message":   fmt.Sprintf("🔥 r/%s 트렌딩 %d개", subreddit, len(posts)),
	})
}

// ── 내부 크롤러 ─────────────────────────────────────────────────────────────

func crawlRedditSearch(query, subreddit string, limit int, sort string) ([]RedditPost, error) {
	var pageURL string
	if subreddit != "" {
		pageURL = fmt.Sprintf("https://www.reddit.com/r/%s/search/?q=%s&sort=%s&restrict_sr=1",
			subreddit, urlEncode(query), sort)
	} else {
		pageURL = fmt.Sprintf("https://www.reddit.com/search/?q=%s&sort=%s",
			urlEncode(query), sort)
	}
	return crawlRedditFeed(pageURL, limit)
}

func crawlRedditFeed(pageURL string, limit int) ([]RedditPost, error) {
	ctx, cancel, err := withMobileStealthTimeout(35 * time.Second)
	if err != nil {
		return nil, fmt.Errorf("스텔스 브라우저 시작 실패: %w", err)
	}
	defer cancel()

	var titles, urls, subreddits, authors, scores, comments []string

	err = chromedp.Run(ctx,
		chromedp.Navigate(pageURL),
		chromedp.Sleep(3*time.Second),
		chromedp.Evaluate(`window.scrollBy(0, 1500)`, nil),
		chromedp.Sleep(1500*time.Millisecond),
		chromedp.Evaluate(`window.scrollBy(0, 1500)`, nil),
		chromedp.Sleep(1000*time.Millisecond),

		// 게시물 제목
		chromedp.Evaluate(`
			(function(){
				var els = document.querySelectorAll(
					'h3[class*="title"], '+
					'[data-testid="post-title"], '+
					'a[data-click-id="body"] h3, '+
					'shreddit-post h1, '+
					'div[data-testid="post-container"] h3, '+
					'[slot="title"]'
				);
				return Array.from(els).slice(0,25).map(e=>e.innerText.trim()).filter(t=>t.length>3);
			})()
		`, &titles),

		// 게시물 URL
		chromedp.Evaluate(`
			(function(){
				var els = document.querySelectorAll(
					'a[data-click-id="body"][href*="/comments/"], '+
					'a[href*="/r/"][href*="/comments/"], '+
					'shreddit-post a[slot="full-post-link"]'
				);
				return Array.from(els).slice(0,25).map(e=>{
					var h = e.href||'';
					return h.startsWith('http') ? h : 'https://www.reddit.com'+h;
				}).filter(h=>h.includes('/comments/'));
			})()
		`, &urls),

		// 서브레딧
		chromedp.Evaluate(`
			(function(){
				var els = document.querySelectorAll(
					'a[href*="/r/"][data-click-id="subreddit"], '+
					'[data-testid="subreddit-name"], '+
					'shreddit-post [slot="subredditName"]'
				);
				return Array.from(els).slice(0,25).map(e=>e.innerText.trim().replace('r/',''));
			})()
		`, &subreddits),

		// 작성자
		chromedp.Evaluate(`
			(function(){
				var els = document.querySelectorAll(
					'a[href*="/user/"][data-click-id="user"], '+
					'[data-testid="post_author_link"], '+
					'shreddit-post [slot="authorName"]'
				);
				return Array.from(els).slice(0,25).map(e=>e.innerText.trim().replace('u/',''));
			})()
		`, &authors),

		// 추천 수
		chromedp.Evaluate(`
			(function(){
				var els = document.querySelectorAll(
					'button[aria-label*="upvote"] ~ faceplate-number, '+
					'[data-testid="vote-arrows"] ~ span, '+
					'div[class*="score"], '+
					'shreddit-post [slot="vote-score"]'
				);
				return Array.from(els).slice(0,25).map(e=>e.innerText.trim()).filter(t=>t!='');
			})()
		`, &scores),

		// 댓글 수
		chromedp.Evaluate(`
			(function(){
				var els = document.querySelectorAll(
					'a[data-click-id="comments"] span, '+
					'[data-testid="comments-page-link-num-comments"], '+
					'shreddit-post [slot="commentCount"]'
				);
				return Array.from(els).slice(0,25).map(e=>e.innerText.trim()).filter(t=>t!='');
			})()
		`, &comments),
	)

	if err != nil {
		return nil, fmt.Errorf("Reddit 크롤링 오류: %w", err)
	}

	var posts []RedditPost
	max := len(titles)
	if len(urls) > 0 && len(urls) < max {
		max = len(urls)
	}
	if max > limit {
		max = limit
	}

	for i := 0; i < max; i++ {
		post := RedditPost{}
		if i < len(titles) {
			post.Title = titles[i]
		}
		if i < len(urls) {
			post.URL = urls[i]
		}
		if i < len(subreddits) {
			post.Subreddit = subreddits[i]
		}
		if i < len(authors) {
			post.Author = authors[i]
		}
		if i < len(scores) {
			post.Score = scores[i]
		}
		if i < len(comments) {
			post.Comments = comments[i]
		}
		if post.Title != "" || post.URL != "" {
			posts = append(posts, post)
		}
	}

	// URL만 있고 title 없는 경우 보완
	if len(posts) == 0 && len(urls) > 0 {
		for i, u := range urls {
			if i >= limit {
				break
			}
			sr := ""
			if i < len(subreddits) {
				sr = subreddits[i]
			}
			// URL에서 서브레딧 추출
			if sr == "" {
				parts := strings.Split(u, "/r/")
				if len(parts) > 1 {
					sr = strings.Split(parts[1], "/")[0]
				}
			}
			posts = append(posts, RedditPost{
				URL:       u,
				Title:     fmt.Sprintf("Reddit 게시물 #%d", i+1),
				Subreddit: sr,
			})
		}
	}

	return posts, nil
}
