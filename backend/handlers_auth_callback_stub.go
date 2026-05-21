//go:build !windows

package main

import (
	"encoding/json"
	"net/http"
	"sync"
)

var (
	pendingOAuthCode string
	pendingOAuthMu   sync.Mutex
)

func handleAuthCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code != "" {
		pendingOAuthMu.Lock()
		pendingOAuthCode = code
		pendingOAuthMu.Unlock()
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html><html lang="ko"><head><meta charset="UTF-8"><title>로그인 완료</title>
<style>body{background:#0d0d14;color:#fff;font-family:sans-serif;display:flex;align-items:center;justify-content:center;height:100vh;margin:0}.card{text-align:center;padding:48px;background:#1a1a2e;border-radius:20px}</style></head>
<body><div class="card"><div style="font-size:52px">✅</div><h1 style="color:#4ade80;margin:12px 0">로그인 완료!</h1><p style="color:rgba(255,255,255,0.5)">이 창을 닫아도 됩니다.</p></div>
<script>setTimeout(()=>{try{window.close()}catch(e){}},3000)</script></body></html>`))
}

func handleAuthCallbackPoll(w http.ResponseWriter, r *http.Request) {
	pendingOAuthMu.Lock()
	code := pendingOAuthCode
	pendingOAuthCode = ""
	pendingOAuthMu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"code": code})
}
