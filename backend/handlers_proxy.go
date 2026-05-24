//go:build windows

package main

import (
	"bytes"
	"context"
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

// jwtContextKey is the context key type for JWT storage (unexported to prevent collisions).
type jwtContextKey struct{}

// ── 글로벌 JWT 폴백 (단일 사용자 데스크탑 앱 — 딥 콜체인용) ─────────────────
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

// getJWTFromCtx extracts the JWT from context (preferred) or falls back to the global.
func getJWTFromCtx(ctx context.Context) string {
	if v, ok := ctx.Value(jwtContextKey{}).(string); ok && v != "" {
		return v
	}
	return getJWT()
}

// jwtMiddleware: Authorization: Bearer <token> 헤더에서 JWT 추출 →
// request context와 글로벌 변수 양쪽에 저장.
func jwtMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); len(auth) > 7 && auth[:7] == "Bearer " {
			token := auth[7:]
			setJWT(token)
			r = r.WithContext(context.WithValue(r.Context(), jwtContextKey{}, token))
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

// callProxyCtx: Supabase Edge Function (ai-proxy) 호출 — request-scoped JWT 우선.
func callProxyCtx(ctx context.Context, action string, payload map[string]interface{}) (*proxyResp, error) {
	jwt := getJWTFromCtx(ctx)
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

	req, err := http.NewRequestWithContext(ctx, "POST", edgeFunctionURL, bytes.NewReader(body))
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

// callProxy: backward-compatible wrapper using background context.
func callProxy(action string, payload map[string]interface{}) (*proxyResp, error) {
	return callProxyCtx(context.Background(), action, payload)
}

// callGroqViaProxy: Edge Function을 통해 Perplexity 호출 (sonar-pro 모델)
func callGroqViaProxy(msgs []groqMsg, maxTokens int, jsonMode bool) (string, error) {
	payload := map[string]interface{}{
		"model":      pplxChatModel,
		"messages":   msgs,
		"max_tokens": maxTokens,
	}

	pr, err := callProxy("perplexity_chat", payload)
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
	return callTavilyDomainViaProxy(query, maxResults, "")
}

// callTavilyDomainViaProxy: 도메인 필터 포함 Tavily 검색
func callTavilyDomainViaProxy(query string, maxResults int, domain string) (tavilyResult, bool) {
	payload := map[string]interface{}{
		"query":        query,
		"max_results":  maxResults,
		"search_depth": "basic",
	}
	if domain != "" {
		payload["include_domains"] = []string{domain}
	}
	action := "tavily_search"
	if domain != "" {
		action = "tavily_search_domain"
	}
	pr, err := callProxy(action, payload)
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

// callVisionViaProxy: Edge Function을 통해 이미지 분석 (Groq Vision)
func callVisionViaProxy(b64img, question, lang string) (string, error) {
	if question == "" {
		if lang == "en" {
			question = "What is on this screen? Describe the main content and notable elements."
		} else {
			question = "이 화면에 무엇이 있나요? 주요 내용을 설명해주세요."
		}
	}
	systemMsg := "화면 캡처를 분석하는 AI 비서입니다."
	if lang == "en" {
		systemMsg = "You are an AI assistant analyzing screen captures."
	}
	payload := map[string]interface{}{
		"model":      groqVisionModel,
		"max_tokens": 1024,
		"messages": []map[string]interface{}{
			{"role": "system", "content": systemMsg},
			{"role": "user", "content": []map[string]interface{}{
				{"type": "text", "text": question},
				{"type": "image_url", "image_url": map[string]string{
					"url": "data:image/png;base64," + b64img,
				}},
			}},
		},
	}
	pr, err := callProxy("vision_analyze", payload)
	if err != nil {
		return "", err
	}
	raw, _ := json.Marshal(pr.Result)
	var gr struct {
		Choices []struct {
			Message struct{ Content string `json:"content"` } `json:"message"`
		} `json:"choices"`
	}
	if json.Unmarshal(raw, &gr) != nil || len(gr.Choices) == 0 {
		return "", fmt.Errorf("vision proxy 응답 파싱 실패")
	}
	return gr.Choices[0].Message.Content, nil
}

// requireAuth: 인증 필요 엔드포인트 — JWT 없으면 401
func requireAuth(w http.ResponseWriter, r *http.Request) bool {
	if getJWTFromCtx(r.Context()) == "" {
		lang := getLang(r)
		writeJSON(w, 401, map[string]any{"success": false, "message": msgT("로그인이 필요합니다.", "Login is required.", lang), "code": "auth_required"})
		return false
	}
	return true
}
