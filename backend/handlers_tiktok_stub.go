//go:build !windows

package main

import "net/http"

// TikTok — !windows: Python yt-dlp 기반 구현으로 프록시

func handleTikTokSearch(w http.ResponseWriter, r *http.Request) {
	proxyToPython(w, r, "/tiktok/search")
}

func handleTikTokTrending(w http.ResponseWriter, r *http.Request) {
	proxyToPythonGET(w, r, "/tiktok/trending")
}

func handleTikTokProfile(w http.ResponseWriter, r *http.Request) {
	proxyToPython(w, r, "/tiktok/profile")
}
