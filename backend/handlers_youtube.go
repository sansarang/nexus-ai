//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

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

	ctx, cancel, err := withBrowserTimeout(45 * time.Second)
	if err != nil {
		llmMu.RLock()
		tKey := llmTavilyKey
		llmMu.RUnlock()
		if tKey != "" {
			tr, ok := tavilySearch(tKey, "유튜브 오늘 인기 영상 추천 2025", 8)
			if ok {
				json200(w, map[string]any{
					"success": true, "source": "search_fallback",
					"message": msgT("Chrome이 필요합니다. 대신 오늘 인기 영상을 검색했어요.", "Chrome required. Showing trending videos instead.", lang),
					"items": tr.Items, "summary": tr.Summary,
				})
				return
			}
		}
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}
	defer cancel()

	var videoTitles, videoChannels, videoLinks, uploadAges []string

	crawlErr := chromedp.Run(ctx,
		chromedp.Navigate("https://www.youtube.com/feed/subscriptions"),
		chromedp.WaitVisible(`ytd-rich-item-renderer`, chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('#video-title')).slice(0,20).map(e=>e.textContent.trim())`, &videoTitles),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('#channel-name a, #channel-name yt-formatted-string')).slice(0,20).map(e=>e.textContent.trim())`, &videoChannels),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('a#video-title-link, a#thumbnail')).slice(0,20).map(e=>e.href||'').filter(h=>h.includes('/watch'))`, &videoLinks),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('#metadata-line span:first-child')).slice(0,20).map(e=>e.textContent.trim())`, &uploadAges),
	)
	if crawlErr != nil {
		llmMu.RLock()
		tKey := llmTavilyKey
		llmMu.RUnlock()
		if tKey != "" {
			tr, ok := tavilySearch(tKey, "유튜브 오늘 인기 영상 추천 2025", 8)
			if ok {
				json200(w, map[string]any{
					"success": true, "source": "search_fallback",
					"message": msgT("YouTube 로그인이 필요해요. 대신 오늘 인기 영상을 검색했어요.", "YouTube login required. Showing trending videos instead.", lang),
					"items": tr.Items, "summary": tr.Summary,
				})
				return
			}
		}
		writeJSON(w, 500, map[string]any{"success": false, "message": fmt.Sprintf(msgT("YouTube 크롤링 실패: %v", "YouTube crawl failed: %v", lang), crawlErr)})
		return
	}

	var items []YTVideoItem
	for i := range videoTitles {
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

	if summarize && len(items) > 0 {
		var titles []string
		for _, it := range items {
			titles = append(titles, fmt.Sprintf("- [%s] %s", it.Channel, it.Title))
		}
		prompt := "다음 유튜브 오늘 영상 중 중요하거나 흥미로운 것만 골라서 한국어로 간단히 요약해줘:\n" + strings.Join(titles, "\n")
		summary, _, _ := callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 400, false)
		json200(w, map[string]any{
			"success": true, "items": items, "count": len(items), "summary": summary,
			"message": fmt.Sprintf(msgT("구독 채널 오늘 영상 %d개", "Today's videos from subscriptions: %d", lang), len(items)),
		})
		return
	}

	json200(w, map[string]any{
		"success": true, "items": items, "count": len(items),
		"message": fmt.Sprintf(msgT("구독 채널 오늘 영상 %d개", "Today's videos from subscriptions: %d", lang), len(items)),
	})
}

// POST /api/youtube/playlist/add
func handleYouTubePlaylistAdd(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		VideoURL string `json:"video_url"`
		Playlist string `json:"playlist"`
	}
	tryDecodeBody(r, &req)
	if req.VideoURL == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("video_url 필요", "video_url required", lang)})
		return
	}
	if req.Playlist == "" {
		req.Playlist = "나중에 볼 동영상"
	}

	ctx, cancel, err := withBrowserTimeout(30 * time.Second)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}
	defer cancel()

	runErr := chromedp.Run(ctx,
		chromedp.Navigate(req.VideoURL),
		chromedp.WaitVisible(`ytd-watch-metadata`, chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
		chromedp.Click(`ytd-button-renderer.ytd-watch-metadata button[aria-label*="저장"], ytd-button-renderer button[aria-label*="Save"]`, chromedp.ByQuery),
		chromedp.Sleep(1*time.Second),
		chromedp.Evaluate(fmt.Sprintf(`
			var items = document.querySelectorAll('yt-formatted-string');
			for(var i=0;i<items.length;i++){
				if(items[i].textContent.includes('%s') || items[i].textContent.includes('나중에')) {
					items[i].closest('ytd-playlist-add-to-option-renderer')?.querySelector('tp-yt-paper-checkbox')?.click();
					break;
				}
			}
			true;
		`, req.Playlist), nil),
		chromedp.Sleep(500*time.Millisecond),
	)
	if runErr != nil {
		json200(w, map[string]any{
			"success": false,
			"message": fmt.Sprintf(msgT("플레이리스트 추가 실패 (로그인 필요하거나 UI 변경): %s", "Playlist add failed (login required or UI changed): %s", lang), runErr.Error()),
			"url":     req.VideoURL,
		})
		return
	}
	json200(w, map[string]any{
		"success":   true,
		"message":   fmt.Sprintf(msgT("'%s' 플레이리스트에 추가됨", "Added to '%s' playlist", lang), req.Playlist),
		"video_url": req.VideoURL,
	})
}

// POST /api/youtube/playlist/batch
func handleYouTubePlaylistBatch(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		URLs     []string `json:"urls"`
		Playlist string   `json:"playlist"`
	}
	tryDecodeBody(r, &req)
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
		"success": true, "added": added, "failed": failed,
		"message": fmt.Sprintf(msgT("플레이리스트 추가: %d개 성공, %d개 실패", "Playlist add: %d succeeded, %d failed", lang), added, failed),
	})
}

// POST /api/youtube/search  — Python yt-dlp 우선, Tavily fallback
func handleYouTubeSearch(w http.ResponseWriter, r *http.Request) {
	handleYouTubeSearchWithPython(w, r)
}
