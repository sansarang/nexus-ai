//go:build !windows

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

// ── 콘텐츠 추천 ───────────────────────────────────────────

type ContentItem struct {
	Title    string `json:"title"`
	Platform string `json:"platform"`
	Genre    string `json:"genre,omitempty"`
	URL      string `json:"url,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

// POST /api/recommend/content  {"genres":["액션","SF"],"mood":"신나는","max":5}
func handleContentRecommend(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Genres    []string `json:"genres"`
		Mood      string   `json:"mood"`
		Platforms []string `json:"platforms"` // ["netflix","youtube"]
		Max       int      `json:"max"`
		History   bool     `json:"history"` // 브라우저 히스토리 기반 여부
	}
	tryDecodeBody(r, &req)
	if req.Max == 0 {
		req.Max = 5
	}
	if len(req.Platforms) == 0 {
		req.Platforms = []string{"netflix", "youtube"}
	}

	llmMu.RLock()
	tKey := llmTavilyKey
	llmMu.RUnlock()

	var contextData []string

	// 취향 파악: 브라우저 히스토리 키워드 활용
	if req.History {
		rows, err := queryChromeHistory("", 14)
		if err == nil && len(rows) > 0 {
			freq := map[string]int{}
			for _, row := range rows {
				if strings.Contains(row["url"], "netflix") || strings.Contains(row["url"], "youtube") {
					words := strings.Fields(row["title"])
					for _, w := range words {
						if len([]rune(w)) >= 2 {
							freq[w]++
						}
					}
				}
			}
			var kws []string
			for k, v := range freq {
				if v >= 2 {
					kws = append(kws, k)
				}
				if len(kws) >= 10 {
					break
				}
			}
			if len(kws) > 0 {
				contextData = append(contextData, "최근 시청 키워드: "+strings.Join(kws, ", "))
			}
		}
	}

	// 플랫폼별 인기 콘텐츠 검색
	var allItems []ContentItem

	for _, platform := range req.Platforms {
		if tKey == "" {
			break
		}
		var query string
		switch platform {
		case "netflix":
			query = "넷플릭스 지금 인기 드라마 영화 추천 2025"
			if len(req.Genres) > 0 {
				query = "넷플릭스 " + strings.Join(req.Genres, " ") + " 추천 2025"
			}
		case "youtube":
			query = "유튜브 인기 영상 추천 오늘 2025"
			if req.Mood != "" {
				query = "유튜브 " + req.Mood + " 영상 추천"
			}
		case "watcha":
			query = "왓챠 인기 콘텐츠 추천 2025"
		default:
			query = platform + " 인기 콘텐츠 추천 2025"
		}

		tr, ok := tavilySearch(tKey, query, 5)
		if !ok {
			continue
		}

		// LLM으로 콘텐츠 목록 추출
		contextStr := strings.Join(contextData, "\n")
		prompt := fmt.Sprintf(`다음 검색 결과에서 %s 추천 콘텐츠 %d개를 추출해줘.
%s
JSON 배열로만 출력: [{"title":"제목","genre":"장르","reason":"추천 이유"}]

검색 결과:
%s`, platform, req.Max, contextStr, tr.Summary)

		raw, _, _ := callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 500, true)
		raw = strings.TrimSpace(raw)
		if startIdx := strings.Index(raw, "["); startIdx >= 0 {
			if endIdx := strings.LastIndex(raw, "]"); endIdx > startIdx {
				raw = raw[startIdx : endIdx+1]
			}
		}
		var parsed []struct {
			Title  string `json:"title"`
			Genre  string `json:"genre"`
			Reason string `json:"reason"`
		}
		if json.Unmarshal([]byte(raw), &parsed) == nil {
			for _, p := range parsed {
				allItems = append(allItems, ContentItem{
					Title:    p.Title,
					Platform: platform,
					Genre:    p.Genre,
					Reason:   p.Reason,
				})
			}
		}
	}

	if len(allItems) > req.Max*len(req.Platforms) {
		allItems = allItems[:req.Max*len(req.Platforms)]
	}

	json200(w, map[string]any{
		"success": true,
		"items":   allItems,
		"count":   len(allItems),
		"message": msgT(fmt.Sprintf("추천 콘텐츠 %d개", len(allItems)), fmt.Sprintf("%d recommended items", len(allItems)), getLang(r)),
	})
}

// ── Netflix 크롤링 ────────────────────────────────────────

// GET /api/netflix/trending
func handleNetflixTrending(w http.ResponseWriter, r *http.Request) {
	// chromedp로 Netflix 인기 콘텐츠 가져오기 (로그인 상태 활용)
	ctx, cancel := newChromedpCtx(30 * time.Second)
	defer cancel()

	var titles []string
	err := chromedp.Run(ctx,
		chromedp.Navigate("https://www.netflix.com/browse"),
		chromedp.WaitVisible(`.slider-item, .title-card`, chromedp.ByQuery),
		chromedp.Sleep(3*time.Second),
		chromedp.Evaluate(`
			Array.from(document.querySelectorAll('.slider-item .fallback-text, .title-card .fallback-text, [data-uia="content-card-title"]')).slice(0,20).map(e=>e.textContent.trim()).filter(t=>t)
		`, &titles),
	)

	if err != nil || len(titles) == 0 {
		// 폴백: Tavily
		llmMu.RLock()
		tKey := llmTavilyKey
		llmMu.RUnlock()
		if tKey != "" {
			tr, ok := tavilySearch(tKey, "넷플릭스 지금 인기 TOP10 2025", 8)
			if ok {
				json200(w, map[string]any{
					"success": true,
					"source":  "search_fallback",
					"summary": tr.Summary,
					"items":   tr.Items,
					"message": "Netflix 로그인 필요 — 검색 결과로 대체",
				})
				return
			}
		}
		writeJSON(w, 500, map[string]any{"success": false, "message": fmt.Sprintf("Netflix 크롤링 실패: %v", err)})
		return
	}

	var items []ContentItem
	for _, t := range titles {
		if t != "" {
			items = append(items, ContentItem{Title: t, Platform: "netflix"})
		}
	}

	json200(w, map[string]any{
		"success": true,
		"items":   items,
		"count":   len(items),
		"message": fmt.Sprintf("Netflix 인기 콘텐츠 %d개", len(items)),
	})
}

// ── Recall 키워드 분석 → 콘텐츠 추천 ────────────────────

// GET /api/recall/keywords?days=7
func handleRecallKeywordsMac(w http.ResponseWriter, r *http.Request) {
	days := 7
	fmt.Sscanf(r.URL.Query().Get("days"), "%d", &days)

	// Mac에서는 브라우저 히스토리로 대체
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Get(
		fmt.Sprintf("http://127.0.0.1:17891/api/history/keywords?days=%d", days))
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}
	defer resp.Body.Close()

	var data map[string]any
	json.NewDecoder(resp.Body).Decode(&data)

	keywords, _ := data["keywords"].([]any)
	recommendation, _ := data["recommendation"].(string)

	// 키워드 기반 콘텐츠 검색
	var kwStrs []string
	for _, k := range keywords {
		if km, ok := k.(map[string]any); ok {
			if w, ok := km["word"].(string); ok {
				kwStrs = append(kwStrs, w)
			}
		}
	}

	var contentRec string
	if len(kwStrs) > 0 && recommendation == "" {
		llmMu.RLock()
		tKey := llmTavilyKey
		llmMu.RUnlock()
		_ = tKey
		prompt := fmt.Sprintf("사용자의 최근 관심 키워드: %s\n이 키워드를 바탕으로 Netflix, YouTube에서 볼 만한 콘텐츠 5가지를 추천해줘. 한국어로.", strings.Join(kwStrs[:min(8, len(kwStrs))], ", "))
		contentRec, _, _ = callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 500, false)
	} else {
		contentRec = recommendation
	}

	json200(w, map[string]any{
		"success":           true,
		"keywords":          keywords,
		"content_recommend": contentRec,
		"days":              days,
		"source":            "browser_history",
			"message":           msgT(fmt.Sprintf("최근 %d일 관심 키워드 분석 + 콘텐츠 추천 완료", days), fmt.Sprintf("Keyword analysis + content recommendations for last %d days complete", days), getLang(r)),
	})
}

// ── 위시리스트 (보고 싶은 목록) ──────────────────────────

func watchlistContentPath() string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".nexus")
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "content_wishlist.json")
}

func loadContentWishlist() []ContentItem {
	data, err := os.ReadFile(watchlistContentPath())
	if err != nil {
		return []ContentItem{}
	}
	var items []ContentItem
	json.Unmarshal(data, &items)
	return items
}

func saveContentWishlist(items []ContentItem) {
	data, _ := json.Marshal(items)
	os.WriteFile(watchlistContentPath(), data, 0644)
}

// GET /api/wishlist/content
func handleContentWishlist(w http.ResponseWriter, r *http.Request) {
	items := loadContentWishlist()
	json200(w, map[string]any{"success": true, "items": items, "count": len(items)})
}

// POST /api/wishlist/content  {"title":"...","platform":"netflix","url":"..."}
func handleContentWishlistAdd(w http.ResponseWriter, r *http.Request) {
	var req ContentItem
	tryDecodeBody(r, &req)
	if req.Title == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "title 필요"})
		return
	}
	items := loadContentWishlist()
	for _, item := range items {
		if item.Title == req.Title {
			json200(w, map[string]any{"success": true, "message": msgT(req.Title+"은 이미 위시리스트에 있어요", req.Title+" is already in your wishlist", getLang(r))})
			return
		}
	}
	items = append([]ContentItem{req}, items...)
	if len(items) > 100 {
		items = items[:100]
	}
	saveContentWishlist(items)
	json200(w, map[string]any{"success": true, "message": req.Title + " 위시리스트 추가됨", "item": req})
}

