//go:build !windows

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

// ── YouTube chromedp 자동화 ───────────────────────────────

func newChromedpCtx(timeout time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel, err := getBrowserCtxMac()
	if err != nil || ctx == nil {
		// fallback: 기본 chromedp 컨텍스트
		allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), chromedp.DefaultExecAllocatorOptions[:]...)
		c, cCancel := chromedp.NewContext(allocCtx)
		tCtx, tCancel := context.WithTimeout(c, timeout)
		return tCtx, func() { tCancel(); cCancel(); allocCancel() }
	}
	tCtx, tCancel := context.WithTimeout(ctx, timeout)
	return tCtx, func() { tCancel(); cancel() }
}

// ── 구독 채널 오늘 영상 크롤링 ───────────────────────────

type YTVideoItem struct {
	Title     string `json:"title"`
	Channel   string `json:"channel"`
	URL       string `json:"url"`
	Duration  string `json:"duration"`
	Views     string `json:"views"`
	UploadAge string `json:"upload_age"`
	Summary   string `json:"summary,omitempty"`
}

// GET /api/youtube/subscriptions
func handleYouTubeSubscriptions(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	summarize := r.URL.Query().Get("summarize") == "true"

	ctx, cancel := newChromedpCtx(45 * time.Second)
	defer cancel()

	var videoTitles []string
	var videoChannels []string
	var videoLinks []string
	var uploadAges []string

	err := chromedp.Run(ctx,
		chromedp.Navigate("https://www.youtube.com/feed/subscriptions"),
		chromedp.WaitVisible(`ytd-rich-item-renderer`, chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
		// 제목
		chromedp.Evaluate(`
			Array.from(document.querySelectorAll('#video-title')).slice(0,20).map(e=>e.textContent.trim())
		`, &videoTitles),
		// 채널명
		chromedp.Evaluate(`
			Array.from(document.querySelectorAll('#channel-name a, #channel-name yt-formatted-string')).slice(0,20).map(e=>e.textContent.trim())
		`, &videoChannels),
		// 링크
		chromedp.Evaluate(`
			Array.from(document.querySelectorAll('a#video-title-link, a#thumbnail')).slice(0,20).map(e=>e.href||'').filter(h=>h.includes('/watch'))
		`, &videoLinks),
		// 업로드 시간
		chromedp.Evaluate(`
			Array.from(document.querySelectorAll('#metadata-line span:first-child')).slice(0,20).map(e=>e.textContent.trim())
		`, &uploadAges),
	)

	if err != nil {
		// 로그인 안 됐거나 chromedp 실패 → Tavily 폴백
		llmMu.RLock()
		tKey := llmTavilyKey
		llmMu.RUnlock()
		if tKey != "" {
			tr, ok := tavilySearch(tKey, "유튜브 오늘 인기 영상 추천 2025", 8)
			if ok {
				json200(w, map[string]any{
					"success": true,
					"source":  "search_fallback",
					"message": msgT("YouTube 로그인이 필요해요. 대신 오늘 인기 영상을 검색했어요.", "YouTube login required. Showing trending videos instead.", lang),
					"items":   tr.Items,
					"summary": tr.Summary,
				})
				return
			}
		}
		writeJSON(w, 500, map[string]any{"success": false, "message": fmt.Sprintf(msgT("YouTube 크롤링 실패: %v", "YouTube crawl failed: %v", lang), err)})
		return
	}

	var items []YTVideoItem
	for i := range videoTitles {
		if i >= len(videoTitles) {
			break
		}
		item := YTVideoItem{Title: videoTitles[i]}
		if i < len(videoChannels) {
			item.Channel = videoChannels[i]
		}
		if i < len(videoLinks) {
			item.URL = videoLinks[i]
		}
		if i < len(uploadAges) {
			item.UploadAge = uploadAges[i]
		}
		// 오늘 영상만 필터 (시간/분 전, 방금)
		if item.UploadAge != "" {
			age := strings.ToLower(item.UploadAge)
			isToday := strings.Contains(age, "분 전") || strings.Contains(age, "시간 전") ||
				strings.Contains(age, "방금") || strings.Contains(age, "hour") ||
				strings.Contains(age, "minute") || strings.Contains(age, "just now")
			if !isToday {
				continue
			}
		}
		items = append(items, item)
	}

	// LLM 요약 (요청 시)
	if summarize && len(items) > 0 {
		var titles []string
		for _, it := range items {
			titles = append(titles, fmt.Sprintf("- [%s] %s", it.Channel, it.Title))
		}
		prompt := "다음 유튜브 오늘 영상 중 중요하거나 흥미로운 것만 골라서 한국어로 간단히 요약해줘:\n" + strings.Join(titles, "\n")
		summary, _, _ := callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 400, false)
		json200(w, map[string]any{
			"success": true,
			"items":   items,
			"count":   len(items),
			"summary": summary,
			"message": fmt.Sprintf(msgT("구독 채널 오늘 영상 %d개", "Today's videos from subscriptions: %d", lang), len(items)),
		})
		return
	}

	json200(w, map[string]any{
		"success": true,
		"items":   items,
		"count":   len(items),
		"message": fmt.Sprintf(msgT("구독 채널 오늘 영상 %d개", "Today's videos from subscriptions: %d", lang), len(items)),
	})
}

// ── 유튜브 검색 결과를 플레이리스트에 추가 ──────────────

// POST /api/youtube/playlist/add  {"video_url":"https://youtube.com/watch?v=xxx","playlist":"나중에 볼 동영상"}
func handleYouTubePlaylistAdd(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		VideoURL string `json:"video_url"`
		Playlist string `json:"playlist"` // "나중에 볼 동영상" or playlist name
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.VideoURL == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("video_url 필요", "video_url required", lang)})
		return
	}
	if req.Playlist == "" {
		req.Playlist = "나중에 볼 동영상"
	}

	ctx, cancel := newChromedpCtx(30 * time.Second)
	defer cancel()

	var errMsg string
	err := chromedp.Run(ctx,
		chromedp.Navigate(req.VideoURL),
		chromedp.WaitVisible(`ytd-watch-metadata`, chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
		// "저장" 버튼 클릭
		chromedp.Click(`ytd-button-renderer.ytd-watch-metadata button[aria-label*="저장"], ytd-button-renderer button[aria-label*="Save"]`, chromedp.ByQuery),
		chromedp.Sleep(1*time.Second),
		// 팝업에서 "나중에 볼 동영상" 체크
		chromedp.Evaluate(fmt.Sprintf(`
			var items = document.querySelectorAll('yt-formatted-string');
			var found = false;
			for(var i=0;i<items.length;i++){
				if(items[i].textContent.includes('%s') || items[i].textContent.includes('나중에')) {
					items[i].closest('ytd-playlist-add-to-option-renderer')?.querySelector('tp-yt-paper-checkbox')?.click();
					found = true; break;
				}
			}
			found;
		`, req.Playlist), nil),
		// 닫기
		chromedp.Sleep(500*time.Millisecond),
	)

	if err != nil {
		errMsg = err.Error()
	}

	if errMsg != "" {
		json200(w, map[string]any{
			"success": false,
			"message": fmt.Sprintf(msgT("플레이리스트 추가 실패 (로그인 필요하거나 UI 변경): %s", "Playlist add failed (login required or UI changed): %s", lang), errMsg),
			"url":     req.VideoURL,
		})
		return
	}

	json200(w, map[string]any{
		"success":  true,
		"message":  fmt.Sprintf(msgT("'%s' 플레이리스트에 추가됨", "Added to '%s' playlist", lang), req.Playlist),
		"video_url": req.VideoURL,
	})
}

// POST /api/youtube/playlist/batch  {"urls":["...","..."],"playlist":"나중에 볼 동영상"}
func handleYouTubePlaylistBatch(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		URLs     []string `json:"urls"`
		Playlist string   `json:"playlist"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if len(req.URLs) == 0 {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("urls 필요", "urls required", lang)})
		return
	}
	if req.Playlist == "" {
		req.Playlist = "나중에 볼 동영상"
	}

	added, failed := 0, 0
	for _, u := range req.URLs {
		resp, err := (&http.Client{Timeout: 40 * time.Second}).Post(
			"http://127.0.0.1:17891/api/youtube/playlist/add", "application/json",
			strings.NewReader(fmt.Sprintf(`{"video_url":%q,"playlist":%q}`, u, req.Playlist)))
		if err != nil {
			failed++
			continue
		}
		defer resp.Body.Close()
		var d map[string]any
		json.NewDecoder(resp.Body).Decode(&d)
		if ok, _ := d["success"].(bool); ok {
			added++
		} else {
			failed++
		}
	}

	json200(w, map[string]any{
		"success": true,
		"added":   added,
		"failed":  failed,
		"message": fmt.Sprintf(msgT("플레이리스트 추가: %d개 성공, %d개 실패", "Playlist add: %d succeeded, %d failed", lang), added, failed),
	})
}

// ── 유튜브 검색 ────────────────────────────────────────────

// POST /api/youtube/search  {"query":"...", "max_items":10}
func handleYouTubeSearch(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		Query    string `json:"query"`
		MaxItems int    `json:"max_items"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Query == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("query 필요", "query required", lang)})
		return
	}
	if req.MaxItems == 0 {
		req.MaxItems = 10
	}

	llmMu.RLock()
	tKey := llmTavilyKey
	llmMu.RUnlock()

	if tKey != "" {
		tr, ok := tavilySearchDomain(tKey, req.Query, req.MaxItems, "youtube.com")
		if ok {
			json200(w, map[string]any{
				"success": true,
				"items":   tr.Items,
				"summary": tr.Summary,
				"count":   len(tr.Items),
				"message": fmt.Sprintf("YouTube '%s' 검색 결과 %d개", req.Query, len(tr.Items)),
			})
			return
		}
	}
	writeJSON(w, 500, map[string]any{"success": false, "message": msgT("검색 실패", "Search failed", lang)})
}
