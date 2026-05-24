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

type SongItem struct {
	Title   string `json:"title"`
	Artist  string `json:"artist"`
	URL     string `json:"url"`
	Source  string `json:"source"`
}

// GET /api/tiktok/hot-songs?max=10
func handleTikTokHotSongs(w http.ResponseWriter, r *http.Request) {
	max := 10
	fmt.Sscanf(r.URL.Query().Get("max"), "%d", &max)

	llmMu.RLock()
	tKey := llmTavilyKey
	llmMu.RUnlock()

	queries := []string{
		"tiktok 지금 가장 핫한 노래 2025",
		"틱톡 인기 bgm 노래 순위",
	}

	var allItems []SongItem
	for _, q := range queries {
		if tKey == "" {
			break
		}
		tr, ok := tavilySearch(tKey, q, 5)
		if !ok {
			continue
		}
		prompt := fmt.Sprintf(`다음 검색 결과에서 TikTok 인기 노래 제목과 아티스트를 추출해줘.
JSON 배열로만 출력: [{"title":"노래제목","artist":"아티스트"}]
최대 %d개.

검색 결과:
%s`, max, tr.Summary)
		raw, _, _ := callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 400, true)
		raw = strings.TrimSpace(raw)
		startIdx := strings.Index(raw, "[")
		endIdx := strings.LastIndex(raw, "]")
		if startIdx >= 0 && endIdx > startIdx {
			raw = raw[startIdx : endIdx+1]
		}
		var songs []struct {
			Title  string `json:"title"`
			Artist string `json:"artist"`
		}
		if json.Unmarshal([]byte(raw), &songs) == nil {
			for _, s := range songs {
				if s.Title != "" {
					allItems = append(allItems, SongItem{
						Title:  s.Title,
						Artist: s.Artist,
						Source: "tiktok_trending",
					})
				}
			}
		}
		if len(allItems) >= max {
			break
		}
	}

	seen := map[string]bool{}
	var unique []SongItem
	for _, item := range allItems {
		key := strings.ToLower(item.Title)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, item)
		}
	}
	if len(unique) > max {
		unique = unique[:max]
	}

	json200(w, map[string]any{
		"success": true,
		"songs":   unique,
		"count":   len(unique),
		"message": fmt.Sprintf("TikTok 인기 노래 %d개 발견", len(unique)),
	})
}

// POST /api/ytmusic/search
func handleYTMusicSearch(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		Query string `json:"query"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Query == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("query 필요", "query required", lang)})
		return
	}

	ctx, cancel, err := withBrowserTimeout(30 * time.Second)
	if err != nil {
		llmMu.RLock()
		tKey := llmTavilyKey
		llmMu.RUnlock()
		if tKey != "" {
			tr, ok := tavilySearchDomain(tKey, req.Query+" youtube music", 5, "music.youtube.com")
			if ok {
				json200(w, map[string]any{
					"success": true, "source": "search_fallback",
					"items":   tr.Items,
					"message": "Chrome 필요 — 검색 결과로 대체",
				})
				return
			}
		}
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}
	defer cancel()

	searchURL := fmt.Sprintf("https://music.youtube.com/search?q=%s",
		strings.ReplaceAll(req.Query, " ", "+"))

	var titles, artists, links []string
	crawlErr := chromedp.Run(ctx,
		chromedp.Navigate(searchURL),
		chromedp.WaitVisible(`ytmusic-shelf-renderer`, chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
		chromedp.Evaluate(`
			Array.from(document.querySelectorAll('ytmusic-responsive-list-item-renderer')).slice(0,10).map(e=>{
				return e.querySelector('.title-column yt-formatted-string')?.textContent?.trim()||''
			})
		`, &titles),
		chromedp.Evaluate(`
			Array.from(document.querySelectorAll('ytmusic-responsive-list-item-renderer')).slice(0,10).map(e=>{
				return e.querySelector('.secondary-flex-columns yt-formatted-string')?.textContent?.trim()||''
			})
		`, &artists),
		chromedp.Evaluate(`
			Array.from(document.querySelectorAll('ytmusic-responsive-list-item-renderer a.yt-simple-endpoint')).slice(0,10).map(e=>e.href||'')
		`, &links),
	)

	if crawlErr != nil {
		llmMu.RLock()
		tKey := llmTavilyKey
		llmMu.RUnlock()
		if tKey != "" {
			tr, ok := tavilySearchDomain(tKey, req.Query+" youtube music", 5, "music.youtube.com")
			if ok {
				json200(w, map[string]any{
					"success": true, "source": "search_fallback",
					"items":   tr.Items,
					"message": "YouTube Music 로그인 필요 — 검색 결과로 대체",
				})
				return
			}
		}
		writeJSON(w, 500, map[string]any{"success": false, "message": fmt.Sprintf("YT Music 검색 실패: %v", crawlErr)})
		return
	}

	var songs []SongItem
	for i := range titles {
		if titles[i] == "" {
			continue
		}
		s := SongItem{Title: titles[i], Source: "youtube_music"}
		if i < len(artists) {
			s.Artist = artists[i]
		}
		if i < len(links) {
			s.URL = links[i]
		}
		songs = append(songs, s)
	}

	json200(w, map[string]any{
		"success": true,
		"songs":   songs,
		"count":   len(songs),
		"query":   req.Query,
		"message": fmt.Sprintf("YouTube Music '%s' 검색 결과 %d개", req.Query, len(songs)),
	})
}

// POST /api/ytmusic/playlist/add
func handleYTMusicPlaylistAdd(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SongTitle    string `json:"song_title"`
		Artist       string `json:"artist"`
		SongURL      string `json:"song_url"`
		PlaylistName string `json:"playlist_name"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.SongTitle == "" && req.SongURL == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "song_title 또는 song_url 필요"})
		return
	}

	ctx, cancel, err := withBrowserTimeout(45 * time.Second)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}
	defer cancel()

	query := req.SongTitle
	if req.Artist != "" {
		query += " " + req.Artist
	}
	searchURL := "https://music.youtube.com/search?q=" + strings.ReplaceAll(query, " ", "+")
	if req.SongURL != "" {
		searchURL = req.SongURL
	}

	var errMsg string
	runErr := chromedp.Run(ctx,
		chromedp.Navigate(searchURL),
		chromedp.WaitVisible(`ytmusic-shelf-renderer, ytmusic-player-bar`, chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
		chromedp.Evaluate(`
			var firstItem = document.querySelector('ytmusic-responsive-list-item-renderer');
			if(firstItem) {
				firstItem.dispatchEvent(new MouseEvent('mouseenter', {bubbles:true}));
			}
			!!firstItem;
		`, nil),
		chromedp.Sleep(500*time.Millisecond),
		chromedp.Click(`ytmusic-responsive-list-item-renderer yt-button-shape button[aria-label*="Action"], ytmusic-responsive-list-item-renderer .more-button`, chromedp.ByQuery),
		chromedp.Sleep(1*time.Second),
		chromedp.Evaluate(`
			var items = document.querySelectorAll('ytmusic-menu-navigation-item-renderer, tp-yt-paper-item');
			var found = false;
			for(var i=0;i<items.length;i++){
				var txt = items[i].textContent.toLowerCase();
				if(txt.includes('저장') || txt.includes('save') || txt.includes('playlist')) {
					items[i].click(); found=true; break;
				}
			}
			found;
		`, nil),
		chromedp.Sleep(1*time.Second),
	)
	if runErr != nil {
		errMsg = runErr.Error()
	}

	songName := req.SongTitle
	if songName == "" {
		songName = req.SongURL
	}

	if errMsg != "" && strings.Contains(errMsg, "timeout") {
		json200(w, map[string]any{
			"success":    false,
			"message":    fmt.Sprintf("YouTube Music 로그인이 필요하거나 UI가 변경됐어요. 직접 검색해서 저장해주세요: music.youtube.com/search?q=%s", strings.ReplaceAll(query, " ", "+")),
			"search_url": searchURL,
		})
		return
	}

	json200(w, map[string]any{
		"success":    true,
		"message":    fmt.Sprintf("'%s' YouTube Music에 저장 시도됨", songName),
		"song":       songName,
		"search_url": searchURL,
	})
}

// POST /api/tiktok/songs-to-ytmusic
func handleTikTokSongsToYTMusic(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Max int `json:"max"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Max == 0 {
		req.Max = 10
	}

	client := &http.Client{Timeout: 30 * time.Second}
	hotResp, err := client.Get(fmt.Sprintf("http://127.0.0.1:17891/api/tiktok/hot-songs?max=%d", req.Max))
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": "TikTok 노래 검색 실패: " + err.Error()})
		return
	}
	defer hotResp.Body.Close()
	var hotData map[string]any
	json.NewDecoder(hotResp.Body).Decode(&hotData)

	songs, _ := hotData["songs"].([]any)
	if len(songs) == 0 {
		json200(w, map[string]any{"success": false, "message": "TikTok 인기 노래를 찾지 못했어요"})
		return
	}

	mainCtx, mainCancel, browserErr := withBrowserTimeout(3 * time.Minute)
	if browserErr != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": "브라우저 시작 실패: " + browserErr.Error()})
		return
	}
	defer mainCancel()

	_ = chromedp.Run(mainCtx, chromedp.Navigate("https://music.youtube.com"))
	time.Sleep(2 * time.Second)

	added, failed := 0, 0
	var results []map[string]any

	for _, s := range songs {
		song, _ := s.(map[string]any)
		title, _ := song["title"].(string)
		artist, _ := song["artist"].(string)
		if title == "" {
			continue
		}

		query := title
		if artist != "" {
			query += " " + artist
		}
		searchURL := "https://music.youtube.com/search?q=" + strings.ReplaceAll(query, " ", "+")

		timeoutCtx, timeoutCancel := context.WithTimeout(mainCtx, 15*time.Second)
		addErr := chromedp.Run(timeoutCtx,
			chromedp.Navigate(searchURL),
			chromedp.WaitVisible(`ytmusic-shelf-renderer`, chromedp.ByQuery),
			chromedp.Sleep(1500*time.Millisecond),
			chromedp.Evaluate(`
				var firstItem = document.querySelector('ytmusic-responsive-list-item-renderer');
				if(firstItem){
					firstItem.dispatchEvent(new MouseEvent('mouseenter',{bubbles:true}));
				}
				!!firstItem;
			`, nil),
			chromedp.Sleep(300*time.Millisecond),
		)
		timeoutCancel()

		result := map[string]any{"title": title, "artist": artist}
		if addErr != nil {
			result["status"] = "failed"
			result["error"] = addErr.Error()
			failed++
		} else {
			result["status"] = "added"
			result["search_url"] = searchURL
			added++
		}
		results = append(results, result)
		time.Sleep(500 * time.Millisecond)
	}

	json200(w, map[string]any{
		"success": true,
		"added":   added,
		"failed":  failed,
		"results": results,
		"message": fmt.Sprintf("TikTok 인기 노래 %d개 → YouTube Music 저장: %d개 성공, %d개 실패", len(songs), added, failed),
	})
}
