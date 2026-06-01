//go:build !windows

package main

import (
	"fmt"
	"net/http"
)

func jwtMiddleware(next http.Handler) http.Handler { return next }

func requireAuth(w http.ResponseWriter, r *http.Request) bool { return true }

// Mac 개발 환경: pro 플랜 더미 JWT (테스트용)
// Header: {"alg":"HS256","typ":"JWT"}
// Payload: {"sub":"dev-mac","email":"dev@nexus.ai","plan":"pro","iat":1716999999}
func getJWT() string {
	return "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJkZXYtbWFjIiwiZW1haWwiOiJkZXZAbmV4dXMuYWkiLCJwbGFuIjoicHJvIiwiaWF0IjoxNzE2OTk5OTk5fQ.test"
}

func callGroqViaProxy(msgs []groqMsg, maxTokens int, jsonMode bool) (string, error) {
	return "", fmt.Errorf("proxy not available on non-Windows")
}

func callTavilyViaProxy(query string, maxResults int) (tavilyResult, bool) {
	return tavilyResult{}, false
}

func callTavilyDomainViaProxy(query string, maxResults int, domain string) (tavilyResult, bool) {
	return tavilyResult{}, false
}

// proxyResp / callProxy 스텁 — intent_classify_shared.go (공통 파일) 의존성 충족
// Mac 빌드에선 Supabase 프록시 호출 안 함 (개발용 직접 키 사용)
type proxyResp struct {
	Success bool                   `json:"success"`
	Result  map[string]interface{} `json:"result"`
	Error   string                 `json:"error"`
	Code    string                 `json:"code"`
}

func callProxy(action string, payload map[string]interface{}) (*proxyResp, error) {
	return nil, fmt.Errorf("proxy not available on non-Windows (action: %s)", action)
}
