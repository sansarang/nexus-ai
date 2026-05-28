//go:build windows

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	llmMu            sync.RWMutex
	llmPerplexityKey string
	llmGroqKey       string // Groq 전용 키 (gsk_ 접두어)
	llmClaudeKey     string
	llmTavilyKey     string
	llmShodanKey     string
	llmUserLang      string // "ko" | "en" — persisted user language preference
)

// ── 번들 기본 API 키 — 빌드 시 -ldflags로 주입됨 ──────────────────────────
// var 로 선언해야 go build -ldflags "-X main.bundledGroqKey=..." 주입 가능
var (
	bundledGroqKey   = ""
	bundledTavilyKey = ""
	bundledOpenAIKey = ""
)

func injectBundledKeys() {
	llmMu.Lock()
	defer llmMu.Unlock()
	if llmGroqKey == "" {
		llmGroqKey = bundledGroqKey
	}
	if llmPerplexityKey == "" {
		llmPerplexityKey = bundledGroqKey // Groq 키를 양쪽 슬롯에 주입
	}
	if llmTavilyKey == "" {
		llmTavilyKey = bundledTavilyKey
	}
	if llmClaudeKey == "" {
		llmClaudeKey = bundledOpenAIKey
	}
}

// resolveEndpointAndModel: API 키 타입에 따라 엔드포인트 + 모델 자동 결정
// gsk_ → Groq API + Groq 모델  /  pplx- → Perplexity API + Perplexity 모델
func resolveEndpointAndModel(key, requestedModel string, fast bool) (endpoint, model string) {
	if strings.HasPrefix(key, "gsk_") {
		endpoint = groqAPIBase
		if fast {
			model = "llama-3.1-8b-instant"
		} else {
			model = "llama-3.3-70b-versatile"
		}
		switch requestedModel {
		case "llama-3.3-70b-versatile", "llama-3.1-70b-versatile",
			"llama-3.1-8b-instant", "llama-3.2-3b-preview",
			"llama-4-scout-17b-16e-instruct":
			model = requestedModel
		}
	} else {
		endpoint = pplxAPIBase
		if fast {
			model = pplxFastModel
		} else {
			model = pplxChatModel
		}
		if requestedModel == pplxChatModel || requestedModel == pplxFastModel {
			model = requestedModel
		}
	}
	return
}

// GetUserLang returns the persisted user language ("en" or "ko").
// All features — including background jobs — should call this instead of detecting from input text.
func GetUserLang() string {
	llmMu.RLock()
	defer llmMu.RUnlock()
	if llmUserLang == "en" {
		return "en"
	}
	return "ko"
}

// IsUserEng is a convenience wrapper for GetUserLang() == "en".
func IsUserEng() bool { return GetUserLang() == "en" }

// callOpenAICompat: Perplexity API 호출 (OpenAI 호환 포맷)
func callOpenAICompat(apiKey, baseURL, model string, msgs []groqMsg, maxTokens int, jsonMode bool) (string, int, error) {
	if apiKey == "" {
		return "", 0, fmt.Errorf("Perplexity API 키가 설정되지 않았습니다")
	}

	type reqBody struct {
		Model       string    `json:"model"`
		Messages    []groqMsg `json:"messages"`
		Temperature float64   `json:"temperature"`
		MaxTokens   int       `json:"max_tokens"`
		RespFmt     *struct {
			Type string `json:"type"`
		} `json:"response_format,omitempty"`
	}

	rb := reqBody{Model: model, Messages: msgs, Temperature: 0.1, MaxTokens: maxTokens}
	// Perplexity sonar는 response_format을 지원하지 않음 — JSON은 system prompt로 강제

	body, _ := json.Marshal(rb)

	// ── 429 재시도: 최대 3회, 지수 백오프 (1s → 2s → 4s) ──────────
	const maxRetries = 3
	var (
		resp *http.Response
		err  error
		raw  []byte
	)
	client := &http.Client{Timeout: 60 * time.Second}
	backoff := time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		httpReq, rerr := http.NewRequest("POST", baseURL, bytes.NewReader(body))
		if rerr != nil {
			return "", 0, rerr
		}
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err = client.Do(httpReq)
		if err != nil {
			if attempt < maxRetries-1 {
				time.Sleep(backoff)
				backoff *= 2
				continue
			}
			return "", 0, fmt.Errorf("연결 실패 (%s): %w", model, err)
		}

		raw, _ = io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
		resp.Body.Close()

		// 429 Too Many Requests — Retry-After 헤더 또는 지수 백오프
		if resp.StatusCode == 429 && attempt < maxRetries-1 {
			wait := backoff
			if ra := resp.Header.Get("Retry-After"); ra != "" {
				if secs, pe := strconv.Atoi(ra); pe == nil {
					wait = time.Duration(secs) * time.Second
					if wait > 30*time.Second {
						wait = 30 * time.Second // 상한 30초
					}
				}
			}
			log.Printf("[LLM] 429 rate-limit (%s) — %v 후 재시도 (%d/%d)", model, wait, attempt+1, maxRetries)
			time.Sleep(wait)
			backoff *= 2
			continue
		}
		break
	}

	var gr struct {
		Choices []struct {
			Message struct{ Content string `json:"content"` } `json:"message"`
		} `json:"choices"`
		Citations []string `json:"citations"`
		Usage     *struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
		Error *struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &gr); err != nil {
		return "", 0, fmt.Errorf("응답 파싱 실패: %w", err)
	}
	if gr.Error != nil {
		return "", 0, fmt.Errorf("[%s] %s: %s", model, gr.Error.Type, gr.Error.Message)
	}
	if len(gr.Choices) == 0 {
		return "", 0, fmt.Errorf("응답 비어 있음 (%s)", model)
	}
	tokens := 0
	if gr.Usage != nil {
		tokens = gr.Usage.PromptTokens + gr.Usage.CompletionTokens
	}
	// citations를 전역 슬라이스에 임시 저장 (callGroqWithCitations에서 사용)
	lastCitationsMu.Lock()
	lastCitations = gr.Citations
	lastCitationsMu.Unlock()
	return gr.Choices[0].Message.Content, tokens, nil
}

var (
	lastCitationsMu sync.Mutex
	lastCitations   []string
)

// callGroqWithCitations: 답변 텍스트 + Perplexity citations URL 목록 반환
func callGroqWithCitations(apiKey, model string, msgs []groqMsg, maxTokens int) (string, []string, error) {
	llmMu.RLock()
	key := llmPerplexityKey
	if key == "" {
		key = llmGroqKey
	}
	llmMu.RUnlock()
	if key == "" {
		key = apiKey
	}
	endpoint, resolvedModel := resolveEndpointAndModel(key, model, false)
	text, _, err := callOpenAICompat(key, endpoint, resolvedModel, msgs, maxTokens, false)
	if err != nil {
		return "", nil, err
	}
	lastCitationsMu.Lock()
	cites := make([]string, len(lastCitations))
	copy(cites, lastCitations)
	lastCitationsMu.Unlock()
	return text, cites, nil
}

// callGroq: 키 타입에 따라 Groq 또는 Perplexity API 자동 선택
func callGroq(apiKey, model string, msgs []groqMsg, maxTokens int, jsonMode bool) (string, int, error) {
	llmMu.RLock()
	key := llmPerplexityKey
	if key == "" {
		key = llmGroqKey
	}
	llmMu.RUnlock()
	if key == "" {
		key = apiKey
	}
	endpoint, resolvedModel := resolveEndpointAndModel(key, model, false)
	return callOpenAICompat(key, endpoint, resolvedModel, msgs, maxTokens, jsonMode)
}

func callGroqVision(_, _, _, _ string) (string, error) {
	return "", fmt.Errorf("Vision 기능은 현재 지원되지 않습니다")
}

// ToolDef: OpenAI 호환 함수 호출 도구 정의
type ToolDef struct {
	Type     string         `json:"type"`
	Function ToolFunctionDef `json:"function"`
}

type ToolFunctionDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// ToolCallResult: LLM이 선택한 함수 호출 결과
type ToolCallResult struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

// callGroqWithTools: Groq function calling — 도구 목록을 포함해 호출하고 tool_calls 파싱
// Groq(gsk_) 키만 function calling 지원. Perplexity(pplx-) 키면 일반 프롬프트로 폴백.
func callGroqWithTools(msgs []groqMsg, tools []ToolDef, maxTokens int) (*ToolCallResult, string, error) {
	llmMu.RLock()
	key := llmGroqKey
	if key == "" {
		key = llmPerplexityKey
	}
	llmMu.RUnlock()

	if key == "" {
		return nil, "", fmt.Errorf("API 키 없음")
	}

	// Perplexity는 function calling 미지원 — 일반 callGroqWithFallback으로 위임
	if !strings.HasPrefix(key, "gsk_") {
		return nil, "", fmt.Errorf("function calling은 Groq 키(gsk_)만 지원")
	}

	endpoint := groqAPIBase
	model := "llama-3.3-70b-versatile"

	type reqBody struct {
		Model     string    `json:"model"`
		Messages  []groqMsg `json:"messages"`
		Tools     []ToolDef `json:"tools"`
		MaxTokens int       `json:"max_tokens"`
	}

	rb := reqBody{Model: model, Messages: msgs, Tools: tools, MaxTokens: maxTokens}
	body, _ := json.Marshal(rb)

	httpReq, err := http.NewRequest("POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, "", err
	}
	httpReq.Header.Set("Authorization", "Bearer "+key)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))

	var gr struct {
		Choices []struct {
			Message struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &gr); err != nil {
		return nil, "", fmt.Errorf("응답 파싱 실패: %w", err)
	}
	if gr.Error != nil {
		return nil, "", fmt.Errorf("Groq 오류: %s", gr.Error.Message)
	}
	if len(gr.Choices) == 0 {
		return nil, "", fmt.Errorf("응답 비어 있음")
	}

	msg := gr.Choices[0].Message

	// tool_calls가 있으면 첫 번째 호출 반환
	if len(msg.ToolCalls) > 0 {
		tc := msg.ToolCalls[0]
		var args map[string]any
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			args = map[string]any{}
		}
		return &ToolCallResult{Name: tc.Function.Name, Arguments: args}, "", nil
	}

	// tool_calls 없으면 텍스트 content 반환 (일반 응답 폴백)
	return nil, strings.TrimSpace(msg.Content), nil
}

// callGroqWithFallback: Supabase 프록시 → Groq/Perplexity → OpenAI → Claude 순서로 폴백
func isProxyLimitError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.HasPrefix(msg, "[usage_limit]") || strings.HasPrefix(msg, "[subscription_expired]")
}

func callGroqWithFallback(msgs []groqMsg, maxTokens int, jsonMode bool) (string, string, error) {
	// 1순위: Supabase Edge Function 프록시 (JWT 있을 때 — 키가 EXE에 없음)
	if content, err := callGroqViaProxy(msgs, maxTokens, jsonMode); err == nil {
		return content, "groq-proxy", nil
	} else if isProxyLimitError(err) {
		return "", "", err // 한도 초과 — 직접 키 폴백 없이 즉시 반환
	}

	llmMu.RLock()
	pKey := llmPerplexityKey
	if pKey == "" {
		pKey = llmGroqKey
	}
	cKey := llmClaudeKey
	llmMu.RUnlock()

	// 2순위: Claude API — sk-ant- 키가 있으면 최우선 직접 호출
	if strings.HasPrefix(cKey, "sk-ant-") {
		ans, err := callClaude(cKey, msgs, maxTokens)
		if err == nil {
			return ans, "claude", nil
		}
		// Claude 실패 시 Groq/Perplexity로 폴백
	}

	// 3순위: Groq / Perplexity
	if pKey != "" {
		endpoint, model := resolveEndpointAndModel(pKey, "", false)
		provider := "groq"
		if strings.HasPrefix(pKey, "pplx-") {
			provider = "perplexity"
		}
		answer, _, err := callOpenAICompat(pKey, endpoint, model, msgs, maxTokens, jsonMode)
		if err == nil {
			return answer, provider, nil
		}
		return "", "", fmt.Errorf("AI API 오류: %v", err)
	}

	// 4순위: Claude (비 sk-ant- 키 — OpenAI 슬롯에 넣은 경우)
	if cKey != "" {
		ans, err := callClaude(cKey, msgs, maxTokens)
		if err == nil {
			return ans, "claude-fallback", nil
		}
		return "", "", err
	}

	return "", "", fmt.Errorf("API 키 미설정 (Groq/Perplexity/Claude)")
}

// callClaude: Anthropic 직접 호출 (fallback)
func callClaude(apiKey string, msgs []groqMsg, maxTokens int) (string, error) {
	if apiKey == "" {
		return "", fmt.Errorf("Claude API 키 미설정")
	}

	type cContent struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	type cMsg struct {
		Role    string     `json:"role"`
		Content []cContent `json:"content"`
	}

	var system string
	var cMsgs []cMsg
	for _, m := range msgs {
		if m.Role == "system" {
			if s, ok := m.Content.(string); ok {
				system = s
			}
			continue
		}
		if text, ok := m.Content.(string); ok && text != "" {
			cMsgs = append(cMsgs, cMsg{
				Role:    m.Role,
				Content: []cContent{{Type: "text", Text: text}},
			})
		}
	}
	if maxTokens == 0 {
		maxTokens = 1024
	}

	reqBody := map[string]any{
		"model":      claudeModel,
		"max_tokens": maxTokens,
		"system":     system,
		"messages":   cMsgs,
	}
	body, _ := json.Marshal(reqBody)
	httpReq, _ := http.NewRequest("POST", claudeAPIBase, bytes.NewReader(body))
	httpReq.Header.Set("x-api-key", apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("Claude 연결 실패: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Content []struct{ Text string `json:"text"` } `json:"content"`
		Error   *struct{ Message string `json:"message"` } `json:"error"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Error != nil {
		return "", fmt.Errorf("Claude: %s", result.Error.Message)
	}
	if len(result.Content) == 0 {
		return "", fmt.Errorf("Claude 응답 없음")
	}
	return result.Content[0].Text, nil
}

// ── 설정 영속화 ──────────────────────────────────────────────────

func llmConfigPath() string {
	appdata := os.Getenv("APPDATA")
	if appdata == "" {
		appdata = os.TempDir()
	}
	dir := filepath.Join(appdata, "Nexus")
	os.MkdirAll(dir, 0700)
	return filepath.Join(dir, "nexus_llm_config.json")
}

type llmConfigFile struct {
	PerplexityKey string `json:"perplexity_key"`
	GroqKey       string `json:"groq_key"`
	ClaudeKey     string `json:"claude_key"`
	TavilyKey     string `json:"tavily_key"`
	UserLang      string `json:"user_lang"` // "ko" | "en"
}

func loadLLMConfig() {
	// 1순위: 환경변수 (개발/배포 오버라이드)
	if v := os.Getenv("GROQ_API_KEY"); v != "" {
		llmMu.Lock()
		llmGroqKey = v
		llmPerplexityKey = v
		llmMu.Unlock()
	}
	if v := os.Getenv("TAVILY_API_KEY"); v != "" {
		llmMu.Lock()
		llmTavilyKey = v
		llmMu.Unlock()
	}
	if v := os.Getenv("OPENAI_API_KEY"); v != "" {
		llmMu.Lock()
		llmClaudeKey = v
		llmMu.Unlock()
	}

	data, err := os.ReadFile(llmConfigPath())
	if err != nil {
		// 설정 파일 없으면 언어 자동 감지 + 번들 키 주입
		llmMu.Lock()
		llmUserLang = detectSystemLang()
		llmMu.Unlock()
		injectBundledKeys()
		return
	}
	var raw map[string]string
	if json.Unmarshal(data, &raw) == nil {
		llmMu.Lock()
		if v := raw["groq_key"]; v != "" {
			llmGroqKey = decryptDPAPI(v)
			llmPerplexityKey = llmGroqKey // Groq 키는 양쪽 슬롯
		}
		if v := raw["perplexity_key"]; v != "" {
			llmPerplexityKey = decryptDPAPI(v)
		}
		if v := raw["claude_key"]; v != "" {
			llmClaudeKey = decryptDPAPI(v)
		}
		if v := raw["tavily_key"]; v != "" {
			llmTavilyKey = decryptDPAPI(v)
		}
		if v := raw["shodan_key"]; v != "" {
			llmShodanKey = v
		}
		if v := raw["user_lang"]; v == "en" || v == "ko" {
			llmUserLang = v
		} else {
			llmUserLang = detectSystemLang()
		}
		llmMu.Unlock()
	}
	// 설정 파일 로드 후에도 빈 키가 있으면 번들 기본 키로 보완
	injectBundledKeys()
}

func saveLLMConfig() {
	llmMu.RLock()
	cfg := llmConfigFile{
		PerplexityKey: encryptDPAPI(llmPerplexityKey),
		GroqKey:       encryptDPAPI(llmGroqKey),
		ClaudeKey:     encryptDPAPI(llmClaudeKey),
		TavilyKey:     encryptDPAPI(llmTavilyKey),
		UserLang:      llmUserLang,
	}
	llmMu.RUnlock()
	data, _ := json.MarshalIndent(cfg, "", "  ")
	os.WriteFile(llmConfigPath(), data, 0600)
}

// GET|POST /api/settings/lang — 사용자 언어 영속 설정
func handleSettingsLang(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		json200(w, map[string]any{"lang": GetUserLang(), "system_lang": detectSystemLang()})
		return
	}
	var req struct {
		Lang string `json:"lang"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Lang != "en" && req.Lang != "ko" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "lang must be 'en' or 'ko'"})
		return
	}
	llmMu.Lock()
	llmUserLang = req.Lang
	llmMu.Unlock()
	saveLLMConfig()
	json200(w, map[string]any{"success": true, "lang": req.Lang})
}

// GET|POST /api/llm/config
func handleLLMConfig(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	if r.Method == http.MethodGet {
		llmMu.RLock()
		pSet := llmPerplexityKey != ""
		gSet := llmGroqKey != ""
		cSet := llmClaudeKey != ""
		tSet := llmTavilyKey != ""
		llmMu.RUnlock()
		json200(w, map[string]any{
			"perplexity_configured": pSet,
			"groq_configured":       gSet,
			"claude_configured":     cSet,
			"tavily_configured":     tSet,
			"ai_ready":              pSet || gSet, // 핵심 AI 기능 가용 여부
			"search_ready":          tSet,
			"models": map[string]string{
				"chat": pplxChatModel,
				"fast": pplxFastModel,
			},
			"provider": "perplexity",
		})
		return
	}
	var req struct {
		PerplexityKey string `json:"perplexity_key"`
		GroqKey       string `json:"groq_key"`   // Groq API 키 (gsk_...)
		ApiKey        string `json:"apiKey"`      // 하위 호환
		ClaudeKey     string `json:"claude_key"`
		TavilyKey     string `json:"tavily_key"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	// 하위 호환: apiKey → perplexity_key
	if req.PerplexityKey == "" && req.ApiKey != "" {
		req.PerplexityKey = req.ApiKey
	}
	// gsk_ 접두어면 Groq 키로 취급
	if req.PerplexityKey != "" && strings.HasPrefix(strings.TrimSpace(req.PerplexityKey), "gsk_") {
		if req.GroqKey == "" {
			req.GroqKey = req.PerplexityKey
		}
		req.PerplexityKey = ""
	}
	llmMu.Lock()
	if s := strings.TrimSpace(req.GroqKey); s != "" {
		llmGroqKey = s
		llmPerplexityKey = s // Groq 키는 양쪽 슬롯에 주입
	}
	if s := strings.TrimSpace(req.PerplexityKey); s != "" {
		llmPerplexityKey = s
	}
	if s := strings.TrimSpace(req.ClaudeKey); s != "" {
		llmClaudeKey = s
	}
	if s := strings.TrimSpace(req.TavilyKey); s != "" {
		llmTavilyKey = s
	}
	llmMu.Unlock()
	saveLLMConfig()
	json200(w, map[string]any{"success": true, "message": msgT("API 키 저장 완료", "API key saved", lang)})
}

// POST /api/llm/chat
func handleLLMChat(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		Messages  []groqMsg `json:"messages"`
		MaxTokens int       `json:"max_tokens"`
		JSONMode  bool      `json:"json_mode"`
		Fast      bool      `json:"fast"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Messages) == 0 {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("messages 배열 필요", "messages array required", lang)})
		return
	}
	if req.MaxTokens == 0 {
		req.MaxTokens = 2048
	}

	llmMu.RLock()
	pKey := llmPerplexityKey
	if pKey == "" {
		pKey = llmGroqKey
	}
	cKey := llmClaudeKey
	llmMu.RUnlock()

	if pKey == "" {
		// 로컬 키 없음 → JWT 있으면 Supabase Edge Function 프록시로 폴백
		if content, err := callGroqViaProxy(req.Messages, req.MaxTokens, req.JSONMode); err == nil {
			json200(w, map[string]any{"success": true, "answer": content, "model": "proxy", "tokens": 0})
			return
		}
		writeJSON(w, 503, map[string]any{"success": false, "message": "API 키가 없습니다. 설정에서 Groq 또는 Perplexity 키를 입력해주세요."})
		return
	}

	endpoint, model := resolveEndpointAndModel(pKey, "", req.Fast)
	answer, tokens, err := callOpenAICompat(pKey, endpoint, model, req.Messages, req.MaxTokens, req.JSONMode)
	if err != nil {
		if cKey != "" {
			ans, err2 := callClaude(cKey, req.Messages, req.MaxTokens)
			if err2 == nil {
				json200(w, map[string]any{"success": true, "answer": ans, "model": "openai-fallback", "tokens": 0})
				return
			}
		}
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}
	json200(w, map[string]any{"success": true, "answer": answer, "model": model, "tokens": tokens})
}

// POST /api/llm/vision — Vision 미지원
func handleLLMVision(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 400, map[string]any{
		"success": false,
		"message": "Vision(이미지 분석) 기능은 현재 지원되지 않습니다.",
	})
}

// POST /api/llm/doc-summary
func handleLLMDocSummary(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FilePath string `json:"file_path"`
		Question string `json:"question"`
	}
	lang := getLang(r)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.FilePath == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("file_path 필요", "file_path required", lang)})
		return
	}

	// 경로 순회 공격 방지
	cleaned := filepath.Clean(req.FilePath)
	if strings.Contains(cleaned, "..") {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("잘못된 파일 경로", "Invalid file path", lang)})
		return
	}

	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()
	if gKey == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("Perplexity API 키 미설정", "Perplexity API key not configured", lang)})
		return
	}

	text, err := extractDocumentText(cleaned)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": msgT("문서 읽기 실패: "+err.Error(), "Failed to read document: "+err.Error(), lang)})
		return
	}
	if len(text) > 8000 {
		text = text[:8000] + "\n...(문서가 길어 앞부분만 분석)"
	}

	question := req.Question
	if question == "" {
		question = "이 문서의 핵심 내용을 5줄로 요약하고, 중요 수치·날짜·이름을 목록으로 정리해주세요."
	}

	prompt := fmt.Sprintf("다음 문서를 분석해주세요:\n\n%s\n\n요청: %s", text, question)
	answer, _, err := callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 2048, false)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}
	json200(w, map[string]any{"success": true, "summary": answer, "file": req.FilePath})
}

// POST /api/llm/doc-compare
func handleLLMDocCompare(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FileA string `json:"file_a"`
		FileB string `json:"file_b"`
		Focus string `json:"focus"`
	}
	lang2 := getLang(r)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.FileA == "" || req.FileB == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("file_a, file_b 필요", "file_a, file_b required", lang2)})
		return
	}
	// 경로 순회 방지
	cleanA := filepath.Clean(req.FileA)
	cleanB := filepath.Clean(req.FileB)
	if strings.Contains(cleanA, "..") || strings.Contains(cleanB, "..") {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("잘못된 파일 경로", "Invalid file path", lang2)})
		return
	}

	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()
	if gKey == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("Perplexity API 키 미설정", "Perplexity API key not configured", lang2)})
		return
	}

	textA, errA := extractDocumentText(cleanA)
	textB, errB := extractDocumentText(cleanB)
	if errA != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": msgT("파일A 오류: "+errA.Error(), "File A error: "+errA.Error(), lang2)})
		return
	}
	if errB != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": msgT("파일B 오류: "+errB.Error(), "File B error: "+errB.Error(), lang2)})
		return
	}

	if len(textA) > 4000 {
		textA = textA[:4000] + "..."
	}
	if len(textB) > 4000 {
		textB = textB[:4000] + "..."
	}

	focusMap := map[string]string{
		"numbers": "숫자·금액·날짜 불일치 집중 분석",
		"changes": "추가·삭제·수정 문장 집중 분석",
		"both":    "숫자 불일치, 추가/삭제/수정, 의미 변화 종합 분석",
	}
	focus := req.Focus
	if focus == "" {
		focus = "both"
	}
	instr := focusMap[focus]
	if instr == "" {
		instr = focusMap["both"]
	}

	prompt := fmt.Sprintf(`두 문서를 비교 분석하세요. %s

=== 문서 A ===
%s

=== 문서 B ===
%s

반드시 다음 JSON 형식으로만 응답:
{
  "summary": "전체 차이 요약 2-3문장",
  "total_differences": 숫자,
  "differences": [
    {"type":"added|deleted|modified|number_mismatch","location":"위치","description":"설명","a_value":"A값","b_value":"B값","severity":"low|medium|high"}
  ],
  "risk_level": "low|medium|high",
  "recommendation": "검토 권고사항"
}`, instr, textA, textB)

	answer, _, err := callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 2048, true)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}

	var parsed map[string]any
	json.Unmarshal([]byte(answer), &parsed)
	json200(w, map[string]any{"success": true, "result": parsed, "file_a": req.FileA, "file_b": req.FileB})
}

// POST /api/llm/deep-search — AI 보강 파일 검색
func handleLLMDeepSearch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query      string `json:"query"`
		Folder     string `json:"folder"`
		MaxResults int    `json:"max_results"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Query == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "query 필요"})
		return
	}
	if req.MaxResults == 0 {
		req.MaxResults = 15
	}

	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()

	keywords := strings.Fields(req.Query)
	if gKey != "" {
		extractPrompt := fmt.Sprintf(`검색 쿼리에서 파일 검색에 쓸 핵심 키워드만 추출:
쿼리: "%s"
JSON 응답: {"keywords": ["키워드1","키워드2"]}`, req.Query)

		raw, _, _ := callGroqWithFallback([]groqMsg{
			{Role: "user", Content: extractPrompt},
		}, 128, true)

		var p struct {
			Keywords []string `json:"keywords"`
		}
		if json.Unmarshal([]byte(raw), &p) == nil && len(p.Keywords) > 0 {
			keywords = p.Keywords
		}
	}

	searchFolder := req.Folder
	if searchFolder == "" {
		searchFolder, _ = os.UserHomeDir()
	}
	hits := deepSearchFiles(strings.Join(keywords, " "), searchFolder, req.MaxResults*3)

	if gKey != "" && len(hits) > 3 {
		type hitItem struct {
			Path    string `json:"path"`
			Snippet string `json:"snippet"`
		}
		hitList := make([]hitItem, 0, len(hits))
		for _, h := range hits {
			hitList = append(hitList, hitItem{Path: h.Path, Snippet: h.Snippet})
		}
		hitJSON, _ := json.Marshal(hitList)

		rankPrompt := fmt.Sprintf(`사용자 검색 의도: "%s"
파일 목록에서 관련성 높은 순으로 점수(0-100) 부여:
%s
JSON: {"ranked":[{"path":"경로","score":85},...]}`, req.Query, string(hitJSON))

		raw, _, _ := callGroqWithFallback([]groqMsg{
			{Role: "user", Content: rankPrompt},
		}, 512, true)

		var ranked struct {
			Ranked []struct {
				Path  string `json:"path"`
				Score int    `json:"score"`
			} `json:"ranked"`
		}
		if json.Unmarshal([]byte(raw), &ranked) == nil {
			scoreMap := map[string]int{}
			for _, rv := range ranked.Ranked {
				scoreMap[rv.Path] = rv.Score
			}
			for i, h := range hits {
				if s, ok := scoreMap[h.Path]; ok {
					hits[i].Score = s
				}
			}
			sortByScore(hits)
		}
	}

	if len(hits) > req.MaxResults {
		hits = hits[:req.MaxResults]
	}

	json200(w, map[string]any{
		"success":       true,
		"results":       hits,
		"total":         len(hits),
		"query":         req.Query,
		"keywords_used": keywords,
		"ai_enhanced":   gKey != "",
	})
}

func deepSearchFiles(query, folder string, maxResults int) []DeepSearchResult {
	searchExts := map[string]bool{
		".txt": true, ".md": true, ".csv": true, ".log": true,
		".docx": true, ".doc": true, ".xlsx": true, ".xls": true,
		".pdf": true, ".hwp": true, ".json": true, ".xml": true,
	}
	queryTerms := strings.Fields(strings.ToLower(query))
	var results []DeepSearchResult

	filepath.Walk(folder, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || len(results) >= maxResults {
			return nil
		}
		for _, skip := range []string{"Windows", "AppData", "node_modules", ".git", "Temp"} {
			if strings.Contains(p, skip) {
				return nil
			}
		}
		ext := strings.ToLower(filepath.Ext(p))
		if !searchExts[ext] {
			return nil
		}

		score := 0
		snippet := ""
		nameLow := strings.ToLower(info.Name())

		for _, term := range queryTerms {
			if strings.Contains(nameLow, term) {
				score += 30
			}
		}
		if info.Size() < 10<<20 {
			text, err := extractDocumentText(p)
			if err == nil {
				tl := strings.ToLower(text)
				for _, term := range queryTerms {
					cnt := strings.Count(tl, term)
					if cnt > 0 {
						score += min(cnt*10, 40)
						if snippet == "" {
							idx := strings.Index(tl, term)
							if idx >= 0 {
								s := max2(idx-40, 0)
								e := idx + len(term) + 80
								if e > len(text) {
									e = len(text)
								}
								snippet = "..." + text[s:e] + "..."
							}
						}
					}
				}
			}
		}

		if score > 0 {
			results = append(results, DeepSearchResult{
				Name:    info.Name(),
				Path:    p,
				Ext:     ext,
				SizeMB:  float64(info.Size()) / (1 << 20),
				ModTime: info.ModTime().Format("2006-01-02 15:04"),
				Snippet: snippet,
				Score:   min(score, 100),
			})
		}
		return nil
	})

	sortByScore(results)
	return results
}

func max2(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// POST /api/llm/route — 번들 Groq 키로 인텐트 라우팅 (프론트엔드 LLM 폴백용)
// 클라이언트에 API 키가 없을 때 백엔드가 번들 키로 도구 선택을 대신 처리
func handleLLMRoute(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Message string `json:"message"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if strings.TrimSpace(req.Message) == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "message required"})
		return
	}

	systemPrompt := `You are a tool selector for NEXUS AI, a Korean Windows PC assistant.
Given the user message, pick the SINGLE BEST tool and extract arguments.

Available tools: pc_status, security_scan, full_scan, clean, repair, gpu_stats, process_top,
volume_control, brightness, wifi_toggle, power_action, launch_app,
driver_check, network_analysis, windows_updates, defender_status, startup_items,
file_search, file_organize, file_duplicates, deep_search, doc_compare, doc_summary, smart_organize, open_folder,
calendar_today, calendar_week, calendar_add, calendar_find_slot,
email_inbox, email_send, email_summarize, email_classify, email_draft,
web_search, news_search, youtube_search, price_compare, video_download, search_pdf,
weather, travel_time, focus_mode, notes, translate, briefing_now,
meeting_start, meeting_stop, meeting_summary, vision_screen, vision_ocr,
caption_start, caption_stop, recall_search, recall_capture,
workflow_run, workflow_list, multi_agent, journal_today, persona_switch,
general_answer

Rules:
- news/뉴스 → web_search with {query: "뉴스 주제"}
- youtube/유튜브/영상 → youtube_search with {query: "..."}
- 날씨 → weather with {city: "도시명"}
- 가격/최저가/쇼핑/쿠팡 → price_compare with {query: "상품명"}
- 검색/알려줘/찾아줘 → web_search with {query: "..."}
- 일반 대화 → general_answer
- Respond ONLY with valid JSON (no explanation): {"tool":"tool_name","args":{"key":"value"}}`

	msgs := []groqMsg{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: req.Message},
	}

	answer, _, err := callGroqWithFallback(msgs, 200, true)
	if err != nil {
		writeJSON(w, 503, map[string]any{"success": false, "message": err.Error()})
		return
	}

	json200(w, map[string]any{"success": true, "tool_call": answer})
}
