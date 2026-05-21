//go:build windows

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

// GET /auth/callback?code=XXX  — Google OAuth 리다이렉트 수신
func handleAuthCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code != "" {
		pendingOAuthMu.Lock()
		pendingOAuthCode = code
		pendingOAuthMu.Unlock()
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<!DOCTYPE html>
<html lang="ko">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>로그인 완료</title>
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body {
    background: #0d0d14;
    color: #fff;
    font-family: -apple-system, 'Segoe UI', sans-serif;
    display: flex; align-items: center; justify-content: center;
    height: 100vh;
  }
  .card {
    text-align: center;
    padding: 48px 40px;
    background: #1a1a2e;
    border: 1px solid rgba(255,255,255,0.1);
    border-radius: 20px;
    max-width: 360px;
    width: 90%;
    box-shadow: 0 20px 60px rgba(0,0,0,0.5);
  }
  .icon { font-size: 52px; margin-bottom: 16px; }
  h1 { font-size: 22px; font-weight: 800; margin-bottom: 8px; color: #4ade80; }
  p { font-size: 14px; color: rgba(255,255,255,0.5); line-height: 1.6; }
  .close-btn {
    margin-top: 24px;
    padding: 12px 32px;
    background: rgba(255,255,255,0.08);
    border: 1px solid rgba(255,255,255,0.15);
    border-radius: 10px;
    color: rgba(255,255,255,0.6);
    font-size: 13px;
    cursor: pointer;
  }
</style>
</head>
<body>
<div class="card">
  <div class="icon">✅</div>
  <h1>로그인 완료!</h1>
  <p>Nexus 앱으로 자동으로 돌아갑니다.<br>이 창을 닫아도 됩니다.</p>
  <button class="close-btn" onclick="window.close()">창 닫기</button>
</div>
<script>
  // 3초 후 자동 닫기
  setTimeout(() => { try { window.close() } catch(e) {} }, 3000)
</script>
</body>
</html>`))
}

// GET /api/auth/callback/pending — 프론트엔드 폴링용
func handleAuthCallbackPoll(w http.ResponseWriter, r *http.Request) {
	pendingOAuthMu.Lock()
	code := pendingOAuthCode
	pendingOAuthCode = ""
	pendingOAuthMu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"code": code})
}
