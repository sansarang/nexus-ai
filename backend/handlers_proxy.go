//go:build windows

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// ── Supabase 공개 설정 ──────────────────────────────────────────────────────
// anon key는 공개 가능 (RLS로 보호됨). 실제 API 키는 Edge Function Secrets에만 존재.
const (
	supabaseProjectURL = "https://dnlkhzoffyomqlqykmnc.supabase.co"
	supabaseAnonKey    = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6ImRubGtoem9mZnlvbXFscXlrbW5jIiwicm9sZSI6ImFub24iLCJpYXQiOjE3NzkxNDk0NjEsImV4cCI6MjA5NDcyNTQ2MX0.Eibduxx5EPg9pYTn2xdJfSL4YBtwwbN70iPaqOAR5Q8"
	edgeFunctionURL    = supabaseProjectURL + "/functions/v1/ai-proxy"
)

// ── 현재 요청 JWT (단일 사용자 데스크탑 앱 전용) ────────────────────────────
var (
	jwtMu      sync.RWMutex
	currentJWT string
)

func setJWT(token string) {
	jwtMu.Lock()
	currentJWT = token
	jwtMu.Unlock()
}

func getJWT() string {
	jwtMu.RLock()
	defer jwtMu.RUnlock()
	return currentJWT
}

// jwtMiddleware: Authorization: Bearer <token> 헤더에서 JWT 추출 후 저장
func jwtMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); len(auth) > 7 && auth[:7] == "Bearer " {
			setJWT(auth[7:])
		}
		next.ServeHTTP(w, r)
	})
}

// ── Edge Function 응답 구조 ──────────────────────────────────────────────────
type proxyResp struct {
	Success bool                   `json:"success"`
	Result  map[string]interface{} `json:"result"`
	Error   string                 `json:"error"`
	Code    string                 `json:"code"`
}

// callProxy: Supabase Edge Function (ai-proxy) 호출
func callProxy(action string, payload map[string]interface{}) (*proxyResp, error) {
	jwt := getJWT()
	if jwt == "" {
		return nil, fmt.Errorf("jwt not set")
	}

	body, err := json.Marshal(map[string]interface{}{
		"action":  action,
		"payload": payload,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", edgeFunctionURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("apikey", supabaseAnonKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	var pr proxyResp
	if err := json.Unmarshal(raw, &pr); err != nil {
		return nil, fmt.Errorf("proxy parse error: %w", err)
	}
	if resp.StatusCode != 200 {
		if pr.Error != "" {
			return nil, fmt.Errorf("%s", pr.Error)
		}
		return nil, fmt.Errorf("proxy HTTP %d", resp.StatusCode)
	}
	return &pr, nil
}

// callGroqViaProxy: Edge Function을 통해 Groq 호출
func callGroqViaProxy(msgs []groqMsg, maxTokens int, jsonMode bool) (string, error) {
	payload := map[string]interface{}{
		"model":      groqChatModel,
		"messages":   msgs,
		"max_tokens": maxTokens,
	}
	if jsonMode {
		payload["response_format"] = map[string]string{"type": "json_object"}
	}

	pr, err := callProxy("groq_chat", payload)
	if err != nil {
		return "", err
	}

	choices, ok := pr.Result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", fmt.Errorf("no choices in proxy response")
	}
	msg, ok := choices[0].(map[string]interface{})["message"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid message in proxy response")
	}
	content, ok := msg["content"].(string)
	if !ok {
		return "", fmt.Errorf("content not string in proxy response")
	}
	return content, nil
}

// callTavilyViaProxy: Edge Function을 통해 Tavily 검색
func callTavilyViaProxy(query string, maxResults int) (tavilyResult, bool) {
	pr, err := callProxy("tavily_search", map[string]interface{}{
		"query":        query,
		"max_results":  maxResults,
		"search_depth": "basic",
	})
	if err != nil {
		return tavilyResult{}, false
	}

	raw, _ := json.Marshal(pr.Result)
	var tv struct {
		Answer  string `json:"answer"`
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
		} `json:"results"`
	}
	if err := json.Unmarshal(raw, &tv); err != nil {
		return tavilyResult{}, false
	}

	res := tavilyResult{Summary: tv.Answer}
	for _, r := range tv.Results {
		res.Items = append(res.Items, map[string]string{
			"title":   r.Title,
			"url":     r.URL,
			"content": r.Content,
		})
	}
	return res, true
}

// requireAuth: 인증 필요 엔드포인트 — JWT 없으면 401
func requireAuth(w http.ResponseWriter, r *http.Request) bool {
	if getJWT() == "" {
		writeJSON(w, 401, map[string]any{"success": false, "message": "로그인이 필요합니다.", "code": "auth_required"})
		return false
	}
	return true
}
