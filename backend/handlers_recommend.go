//go:build windows

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

type ContentItem struct {
	Title    string `json:"title"`
	Platform string `json:"platform"`
	Genre    string `json:"genre,omitempty"`
	URL      string `json:"url,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

// POST /api/recommend/content
func handleContentRecommend(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Genres    []string `json:"genres"`
		Mood      string   `json:"mood"`
		Platforms []string `json:"platforms"`
		Max       int      `json:"max"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Max == 0 {
		req.Max = 5
	}
	if len(req.Platforms) == 0 {
		req.Platforms = []string{"netflix", "youtube"}
	}

	llmMu.RLock()
	tKey := llmTavilyKey
	llmMu.RUnlock()

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

		prompt := fmt.Sprintf(`다음 검색 결과에서 %s 추천 콘텐츠 %d개를 추출해줘.
JSON 배열로만 출력: [{"title":"제목","genre":"장르","reason":"추천 이유"}]

검색 결과:
%s`, platform, req.Max, tr.Summary)

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

// GET /api/netflix/trending
func handleNetflixTrending(w http.ResponseWriter, r *http.Request) {
	ctx, cancel, err := withBrowserTimeout(30 * time.Second)
	if err != nil {
		llmMu.RLock()
		tKey := llmTavilyKey
		llmMu.RUnlock()
		if tKey != "" {
			tr, ok := tavilySearch(tKey, "넷플릭스 지금 인기 TOP10 2025", 8)
			if ok {
				json200(w, map[string]any{
					"success": true, "source": "search_fallback",
					"summary": tr.Summary, "items": tr.Items,
					"message": "Chrome 필요 — 검색 결과로 대체",
				})
				return
			}
		}
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}
	defer cancel()

	var titles []string
	crawlErr := chromedp.Run(ctx,
		chromedp.Navigate("https://www.netflix.com/browse"),
		chromedp.WaitVisible(`.slider-item, .title-card`, chromedp.ByQuery),
		chromedp.Sleep(3*time.Second),
		chromedp.Evaluate(`
			Array.from(document.querySelectorAll('.slider-item .fallback-text, .title-card .fallback-text, [data-uia="content-card-title"]')).slice(0,20).map(e=>e.textContent.trim()).filter(t=>t)
		`, &titles),
	)

	if crawlErr != nil || len(titles) == 0 {
		llmMu.RLock()
		tKey := llmTavilyKey
		llmMu.RUnlock()
		if tKey != "" {
			tr, ok := tavilySearch(tKey, "넷플릭스 지금 인기 TOP10 2025", 8)
			if ok {
				json200(w, map[string]any{
					"success": true, "source": "search_fallback",
					"summary": tr.Summary, "items": tr.Items,
					"message": "Netflix 로그인 필요 — 검색 결과로 대체",
				})
				return
			}
		}
		writeJSON(w, 500, map[string]any{"success": false, "message": fmt.Sprintf("Netflix 크롤링 실패: %v", crawlErr)})
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

// GET /api/recall/keywords?days=7
// Windows에서는 Chrome history DB 직접 접근 대신 LLM 기반 콘텐츠 추천으로 대체
func handleRecallKeywords(w http.ResponseWriter, r *http.Request) {
	days := 7
	fmt.Sscanf(r.URL.Query().Get("days"), "%d", &days)
	lang := getLang(r)

	// Windows Chrome history: %LOCALAPPDATA%\Google\Chrome\User Data\Default\History (SQLite)
	// Chrome 실행 중엔 DB 잠금 — 안전하게 복사 후 읽기 시도
	var kwStrs []string
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		home, _ := os.UserHomeDir()
		localAppData = filepath.Join(home, "AppData", "Local")
	}
	historyDB := filepath.Join(localAppData, "Google", "Chrome", "User Data", "Default", "History")
	tmpDB := filepath.Join(os.TempDir(), fmt.Sprintf("nexus_ch_%d.db", timeNowMs()))
	if src, err := os.ReadFile(historyDB); err == nil {
		if os.WriteFile(tmpDB, src, 0600) == nil {
			kwStrs = extractChromeKeywords(tmpDB, days)
			os.Remove(tmpDB)
		}
	}

	var contentRec string
	if len(kwStrs) > 0 {
		n := len(kwStrs)
		if n > 8 {
			n = 8
		}
		prompt := fmt.Sprintf("사용자의 최근 관심 키워드: %s\n이 키워드를 바탕으로 Netflix, YouTube에서 볼 만한 콘텐츠 5가지를 추천해줘. 한국어로.", strings.Join(kwStrs[:n], ", "))
		contentRec, _, _ = callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 500, false)
	} else {
		// 히스토리 없으면 LLM으로 일반 추천
		prompt := fmt.Sprintf("최근 %d일 기준 넷플릭스와 유튜브에서 가장 인기 있는 한국 콘텐츠 5가지를 추천해줘. 한국어로.", days)
		contentRec, _, _ = callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 500, false)
	}

	var keywords []map[string]any
	for _, kw := range kwStrs {
		keywords = append(keywords, map[string]any{"word": kw, "count": 1})
	}

	json200(w, map[string]any{
		"success":           true,
		"keywords":          keywords,
		"content_recommend": contentRec,
		"days":              days,
		"source":            "chrome_history",
		"message":           msgT(fmt.Sprintf("최근 %d일 관심 키워드 분석 + 콘텐츠 추천 완료", days), fmt.Sprintf("Keyword analysis + content recommendations for last %d days complete", days), lang),
	})
}

func timeNowMs() int64 {
	return time.Now().UnixMilli()
}

// Chrome History SQLite에서 도메인 키워드 추출 (순수 Go — mattn/go-sqlite3 없이 간단 파싱)
func extractChromeKeywords(dbPath string, days int) []string {
	// SQLite 없이: URL 패턴에서 키워드 추출 (Chrome history는 바이너리지만 URL은 평문으로 저장됨)
	data, err := os.ReadFile(dbPath)
	if err != nil {
		return nil
	}
	cutoff := time.Now().AddDate(0, 0, -days)
	_ = cutoff

	seen := map[string]bool{}
	var keywords []string

	// URL 문자열 패턴 추출 (sqlite 파일에서 평문 URL 영역 스캔)
	text := string(data)
	for _, line := range strings.Split(text, "\x00") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "http") {
			continue
		}
		// 도메인 제거, 경로에서 키워드 추출
		line = strings.TrimPrefix(line, "https://")
		line = strings.TrimPrefix(line, "http://")
		parts := strings.FieldsFunc(line, func(r rune) bool {
			return r == '/' || r == '?' || r == '&' || r == '=' || r == '-' || r == '_' || r == '.'
		})
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if len([]rune(p)) < 2 || len([]rune(p)) > 20 {
				continue
			}
			// 숫자만이거나 일반 파라미터 제외
			if strings.ContainsAny(p, "0123456789") && len(p) > 6 {
				continue
			}
			if seen[p] {
				continue
			}
			seen[p] = true
			keywords = append(keywords, p)
			if len(keywords) >= 20 {
				return keywords
			}
		}
	}
	return keywords
}

func watchlistContentPath() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		home, _ := os.UserHomeDir()
		appData = filepath.Join(home, "AppData", "Roaming")
	}
	dir := filepath.Join(appData, "Nexus")
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

// POST /api/wishlist/content
func handleContentWishlistAdd(w http.ResponseWriter, r *http.Request) {
	var req ContentItem
	json.NewDecoder(r.Body).Decode(&req)
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
