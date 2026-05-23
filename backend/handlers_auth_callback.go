//go:build windows

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

// GET /auth/callback — implicit flow: 토큰이 URL 해시에 있음 (#access_token=...)
// 브라우저 JS가 해시를 읽어서 /api/auth/token 으로 POST
func handleAuthCallback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<!DOCTYPE html>
<html lang="ko">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>로그인 완료</title>
<style>
  *{margin:0;padding:0;box-sizing:border-box}
  body{background:#0d0d14;color:#fff;font-family:-apple-system,'Segoe UI',sans-serif;display:flex;align-items:center;justify-content:center;height:100vh}
  .card{text-align:center;padding:48px 40px;background:#1a1a2e;border:1px solid rgba(255,255,255,0.1);border-radius:20px;max-width:360px;width:90%;box-shadow:0 20px 60px rgba(0,0,0,0.5)}
  .icon{font-size:52px;margin-bottom:16px}
  h1{font-size:22px;font-weight:800;margin-bottom:8px;color:#4ade80}
  p{font-size:14px;color:rgba(255,255,255,0.5);line-height:1.6}
  .close-btn{margin-top:24px;padding:12px 32px;background:#4ade80;border:none;border-radius:10px;color:#000;font-size:14px;font-weight:700;cursor:pointer}
</style>
</head>
<body>
<div class="card">
  <div class="icon" id="icon">⏳</div>
  <h1 id="title">로그인 처리 중...</h1>
  <p id="msg">잠시만 기다려주세요.</p>
  <button class="close-btn" onclick="try{window.open('','_self').close()}catch(e){window.close()}" style="display:none" id="closeBtn">이 창 닫기 ✕</button>
</div>
<script>
(function() {
  var hash = window.location.hash.substring(1)
  if (!hash) {
    // 에러 파라미터 확인
    var search = window.location.search
    document.getElementById('icon').textContent = '❌'
    document.getElementById('title').textContent = '로그인 실패'
    document.getElementById('title').style.color = '#f87171'
    document.getElementById('msg').textContent = decodeURIComponent(search.match(/error_description=([^&]+)/)?.[1] || '다시 시도해주세요.')
    document.getElementById('closeBtn').style.display = 'inline-block'
    return
  }
  fetch('http://127.0.0.1:17891/api/auth/token', {
    method: 'POST',
    headers: {'Content-Type': 'application/x-www-form-urlencoded'},
    body: hash
  }).then(function(r) {
    document.getElementById('icon').textContent = '✅'
    document.getElementById('title').textContent = '로그인 완료!'
    document.getElementById('title').style.color = '#4ade80'
    document.getElementById('msg').textContent = '로그인 완료! 이 창을 닫고 Nexus 앱으로 돌아가세요.'
    document.getElementById('closeBtn').style.display = 'inline-block'
    setTimeout(function(){ try{window.open('','_self').close()}catch(e){try{window.close()}catch(e2){}} }, 1500)
  }).catch(function() {
    document.getElementById('icon').textContent = '❌'
    document.getElementById('title').textContent = '전송 실패'
    document.getElementById('title').style.color = '#f87171'
    document.getElementById('msg').textContent = 'Nexus 앱이 실행 중인지 확인해주세요.'
    document.getElementById('closeBtn').style.display = 'inline-block'
  })
})()
</script>
</body>
</html>`))
}

// POST /api/auth/token — 브라우저 JS가 해시 데이터 전달
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

// GET /api/auth/callback/pending — 프론트엔드 폴링용
func handleAuthCallbackPoll(w http.ResponseWriter, r *http.Request) {
	pendingOAuthMu.Lock()
	token := pendingOAuthToken
	pendingOAuthToken = ""
	pendingOAuthMu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}
