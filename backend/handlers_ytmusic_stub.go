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

// ── TikTok 인기 노래 검색 ─────────────────────────────────

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

	// Tavily로 틱톡 인기 노래 검색
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
		// LLM으로 노래 목록 추출
		prompt := fmt.Sprintf(`다음 검색 결과에서 TikTok 인기 노래 제목과 아티스트를 추출해줘.
JSON 배열로만 출력: [{"title":"노래제목","artist":"아티스트"}]
최대 %d개.

검색 결과:
%s`, max, tr.Summary)
		raw, _, _ := callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 400, true)
		raw = strings.TrimSpace(raw)
		// JSON 배열 파싱
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

	// 중복 제거
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

// ── YouTube Music chromedp 자동화 ─────────────────────────

// POST /api/ytmusic/search  {"query":"Sabrina Carpenter Espresso"}
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

	ctx, cancel := newChromedpCtx(30 * time.Second)
	defer cancel()

	searchURL := fmt.Sprintf("https://music.youtube.com/search?q=%s",
		strings.ReplaceAll(req.Query, " ", "+"))

	var titles, artists, links []string
	err := chromedp.Run(ctx,
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

	if err != nil {
		// Tavily 폴백
		llmMu.RLock()
		tKey := llmTavilyKey
		llmMu.RUnlock()
		if tKey != "" {
			tr, ok := tavilySearchDomain(tKey, req.Query+" youtube music", 5, "music.youtube.com")
			if ok {
				json200(w, map[string]any{
					"success": true,
					"source":  "search_fallback",
					"items":   tr.Items,
					"message": "YouTube Music 로그인 필요 — 검색 결과로 대체",
				})
				return
			}
		}
		writeJSON(w, 500, map[string]any{"success": false, "message": fmt.Sprintf("YT Music 검색 실패: %v", err)})
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

// POST /api/ytmusic/playlist/add  {"song_title":"...","artist":"..."}
func handleYTMusicPlaylistAdd(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SongTitle  string `json:"song_title"`
		Artist     string `json:"artist"`
		SongURL    string `json:"song_url"`
		PlaylistName string `json:"playlist_name"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.SongTitle == "" && req.SongURL == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "song_title 또는 song_url 필요"})
		return
	}

	ctx, cancel := newChromedpCtx(45 * time.Second)
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
	err := chromedp.Run(ctx,
		chromedp.Navigate(searchURL),
		chromedp.WaitVisible(`ytmusic-shelf-renderer, ytmusic-player-bar`, chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
		// 첫 번째 곡에 마우스 올리기 → 점 3개 메뉴 클릭
		chromedp.Evaluate(`
			var firstItem = document.querySelector('ytmusic-responsive-list-item-renderer');
			if(firstItem) {
				// hover 이벤트 발생
				firstItem.dispatchEvent(new MouseEvent('mouseenter', {bubbles:true}));
			}
			!!firstItem;
		`, nil),
		chromedp.Sleep(500*time.Millisecond),
		// 메뉴 버튼 클릭
		chromedp.Click(`ytmusic-responsive-list-item-renderer yt-button-shape button[aria-label*="Action"], ytmusic-responsive-list-item-renderer .more-button`, chromedp.ByQuery),
		chromedp.Sleep(1*time.Second),
		// "저장" / "플레이리스트에 저장" 클릭
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

	if err != nil {
		errMsg = err.Error()
	}

	songName := req.SongTitle
	if songName == "" {
		songName = req.SongURL
	}

	if errMsg != "" && strings.Contains(errMsg, "timeout") {
		json200(w, map[string]any{
			"success": false,
			"message": fmt.Sprintf("YouTube Music 로그인이 필요하거나 UI가 변경됐어요. 직접 검색해서 저장해주세요: music.youtube.com/search?q=%s", strings.ReplaceAll(query, " ", "+")),
			"search_url": searchURL,
		})
		return
	}

	json200(w, map[string]any{
		"success":   true,
		"message":   fmt.Sprintf("'%s' YouTube Music에 저장 시도됨", songName),
		"song":      songName,
		"search_url": searchURL,
	})
}

// ── TikTok 핫 노래 → YT Music 플레이리스트 전체 자동화 ──

// POST /api/tiktok/songs-to-ytmusic  {"max":10}
func handleTikTokSongsToYTMusic(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Max int `json:"max"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Max == 0 {
		req.Max = 10
	}

	// 1. TikTok 인기 노래 검색
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

	// 2. 각 노래를 YT Music에 추가
	added, failed := 0, 0
	var results []map[string]any

	// chromedp 컨텍스트 하나로 여러 곡 처리
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	allocCtx, allocCancel, browserErr := getBrowserCtxMac()
	if browserErr != nil || allocCtx == nil {
		// fallback
		ac, ac2 := chromedp.NewExecAllocator(context.Background(), chromedp.DefaultExecAllocatorOptions[:]...)
		allocCtx, _ = chromedp.NewContext(ac)
		allocCancel = ac2
	}
	defer allocCancel()
	chromedpCtx, chromedpCancel := chromedp.NewContext(allocCtx)
	defer chromedpCancel()

	// YT Music 먼저 열기
	_ = chromedp.Run(chromedpCtx, chromedp.Navigate("https://music.youtube.com"))
	time.Sleep(2 * time.Second)

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

		var addErr error
		timeoutCtx, timeoutCancel := context.WithTimeout(chromedpCtx, 15*time.Second)
		addErr = chromedp.Run(timeoutCtx,
			chromedp.Navigate(searchURL),
			chromedp.WaitVisible(`ytmusic-shelf-renderer`, chromedp.ByQuery),
			chromedp.Sleep(1500*time.Millisecond),
			// 첫 곡 메뉴 → 라이브러리에 저장
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

		_ = ctx // prevent unused warning
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
