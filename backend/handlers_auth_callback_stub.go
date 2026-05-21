//go:build !windows

package main

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"
)

var (
	pendingOAuthToken string
	pendingOAuthMu    sync.Mutex
)

func handleAuthCallback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html><html><head><meta charset="UTF-8"><title>로그인</title></head><body style="background:#0d0d14;color:#fff;font-family:sans-serif;display:flex;align-items:center;justify-content:center;height:100vh;margin:0">
<div style="text-align:center"><div id="icon" style="font-size:52px">⏳</div><h1 id="t" style="color:#4ade80">처리 중...</h1><p id="m" style="color:rgba(255,255,255,0.5)">잠시만요.</p></div>
<script>(function(){var h=window.location.hash.substring(1);if(!h){document.getElementById('icon').textContent='❌';document.getElementById('t').textContent='실패';document.getElementById('t').style.color='#f87171';return;}fetch('http://127.0.0.1:17891/api/auth/token',{method:'POST',headers:{'Content-Type':'application/x-www-form-urlencoded'},body:h}).then(function(){document.getElementById('icon').textContent='✅';document.getElementById('t').textContent='완료!';document.getElementById('m').textContent='창을 닫아도 됩니다.';setTimeout(function(){try{window.close()}catch(e){}},2000)}).catch(function(){document.getElementById('icon').textContent='❌';document.getElementById('t').textContent='전송 실패';document.getElementById('t').style.color='#f87171';})})()</script>
</body></html>`))
}

func handleAuthToken(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 4096))
	if err != nil || len(body) == 0 {
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}
	pendingOAuthMu.Lock()
	pendingOAuthToken = string(body)
	pendingOAuthMu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func handleAuthCallbackPoll(w http.ResponseWriter, r *http.Request) {
	pendingOAuthMu.Lock()
	token := pendingOAuthToken
	pendingOAuthToken = ""
	pendingOAuthMu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}
