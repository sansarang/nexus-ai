//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ── TikTok 검색: Python yt-dlp 우선, chromedp → Tavily fallback ──

func handleTikTokSearchWithPython(w http.ResponseWriter, r *http.Request) {
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
	if req.Limit == 0 {
		req.Limit = 10
	}

	result, err := callPython("POST", "/tiktok/search", map[string]any{
		"query": req.Query, "limit": req.Limit,
	})
	if err == nil {
		if ok, _ := result["success"].(bool); ok {
			if items, _ := result["items"].([]any); len(items) > 0 {
				json200(w, result)
				return
			}
		}
	}

	items, crawlErr := crawlTikTokSearch(req.Query, req.Limit)
	if crawlErr == nil && len(items) > 0 {
		json200(w, map[string]any{
			"success": true, "source": "chromedp",
			"items": items, "count": len(items),
			"message": fmt.Sprintf("🎵 TikTok \"%s\" 검색 결과 %d개", req.Query, len(items)),
		})
		return
	}

	llmMu.RLock()
	tKey := llmTavilyKey
	llmMu.RUnlock()
	if tKey != "" {
		if tr, ok := tavilySearchDomain(tKey, req.Query, req.Limit, "tiktok.com"); ok && len(tr.Items) > 0 {
			json200(w, map[string]any{
				"success": true, "source": "search_fallback",
				"items": tr.Items, "count": len(tr.Items),
				"message": fmt.Sprintf("🎵 TikTok \"%s\" 검색 결과 %d개", req.Query, len(tr.Items)),
			})
			return
		}
	}

	json200(w, map[string]any{
		"success": false, "items": []any{}, "count": 0,
		"message": fmt.Sprintf(msgT("'%s' TikTok 검색 결과가 없어요.", "No TikTok results for '%s'.", lang), req.Query),
	})
}

// ── YTMusic 검색: Python ytmusicapi 우선, chromedp fallback ──────

func handleYTMusicSearchWithPython(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query string `json:"query"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Query == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "query 필요"})
		return
	}

	result, err := callPython("POST", "/ytmusic/search", map[string]any{"query": req.Query})
	if err == nil {
		if ok, _ := result["success"].(bool); ok {
			if songs, _ := result["songs"].([]any); len(songs) > 0 {
				json200(w, result)
				return
			}
		}
	}

	handleYTMusicSearchChromedp(w, r, req.Query)
}
