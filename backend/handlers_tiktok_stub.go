//go:build !windows

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// TikTok 크롤러 — !windows stub (Tavily fallback만 사용)

func handleTikTokSearch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Query == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "query 필요"})
		return
	}
	if req.Limit == 0 {
		req.Limit = 10
	}

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
				"message": fmt.Sprintf("🎵 TikTok \"%s\" 검색 결과 %d개", req.Query, len(tr.Items)),
			})
			return
		}
	}
	writeJSON(w, 200, map[string]any{"success": false, "message": "TikTok 크롤링은 Windows 전용입니다."})
}

func handleTikTokTrending(w http.ResponseWriter, r *http.Request) {
	llmMu.RLock()
	tKey := llmTavilyKey
	llmMu.RUnlock()
	if tKey != "" {
		if tr, ok := tavilySearchDomain(tKey, "tiktok 인기 트렌딩 viral 2026", 10, "tiktok.com"); ok {
			json200(w, map[string]any{"success": true, "source": "search_fallback", "items": tr.Items, "message": "🔥 TikTok 트렌딩"})
			return
		}
	}
	writeJSON(w, 200, map[string]any{"success": false, "message": "Windows 전용 기능입니다."})
}

func handleTikTokProfile(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"success": false, "message": "TikTok 프로필 크롤링은 Windows 전용입니다."})
}
