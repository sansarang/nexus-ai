//go:build !windows

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	llmMu            sync.RWMutex
	llmPerplexityKey string
	llmClaudeKey     string
	llmTavilyKey     string
	llmGroqKey       string // Groq 전용 — Structured Outputs Clarify 판단
)

type llmConfigFile struct {
	PerplexityKey string `json:"perplexity_key"`
	ClaudeKey     string `json:"claude_key"`
	TavilyKey     string `json:"tavily_key"`
	GroqKey       string `json:"groq_key"`
}

func llmConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".nexus", "llm_config.json")
}

func loadLLMConfig() {
	data, err := os.ReadFile(llmConfigPath())
	if err != nil {
		return
	}
	// 하위 호환: 기존 groq_key → perplexity_key 마이그레이션
	var raw map[string]string
	if json.Unmarshal(data, &raw) == nil {
		llmMu.Lock()
		if v := raw["perplexity_key"]; v != "" {
			llmPerplexityKey = v
		} else if v := raw["groq_key"]; v != "" {
			llmPerplexityKey = v
		}
		if v := raw["claude_key"]; v != "" {
			llmClaudeKey = v
		}
		if v := raw["tavily_key"]; v != "" {
			llmTavilyKey = v
		}
		if v := raw["groq_key"]; v != "" {
			llmGroqKey = v
		}
		llmMu.Unlock()
	}
}

func saveLLMConfig() {
	llmMu.RLock()
	cfg := llmConfigFile{
		PerplexityKey: llmPerplexityKey,
		ClaudeKey:     llmClaudeKey,
		TavilyKey:     llmTavilyKey,
		GroqKey:       llmGroqKey,
	}
	llmMu.RUnlock()
	data, _ := json.MarshalIndent(cfg, "", "  ")
	os.MkdirAll(filepath.Dir(llmConfigPath()), 0755)
	os.WriteFile(llmConfigPath(), data, 0600)
}

// callOpenAICompat: OpenAI 호환 엔드포인트 범용 호출 (Perplexity 전용)
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
	req, _ := http.NewRequest("POST", baseURL, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("연결 실패 (%s): %w", model, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var gr struct {
		Choices []struct {
			Message struct{ Content string `json:"content"` } `json:"message"`
		} `json:"choices"`
		Citations []string `json:"citations"`
		Error     *struct{ Message string `json:"message"` } `json:"error"`
	}
	if err := json.Unmarshal(raw, &gr); err != nil {
		return "", 0, fmt.Errorf("응답 파싱 실패: %w", err)
	}
	if gr.Error != nil {
		return "", 0, fmt.Errorf("[%s] %s", model, gr.Error.Message)
	}
	if len(gr.Choices) == 0 {
		return "", 0, fmt.Errorf("응답 없음 (%s)", model)
	}
	lastCitationsMu.Lock()
	lastCitations = gr.Citations
	lastCitationsMu.Unlock()
	return gr.Choices[0].Message.Content, maxTokens, nil
}

var (
	lastCitationsMu sync.Mutex
	lastCitations   []string
)

func callGroqWithCitations(apiKey, model string, msgs []groqMsg, maxTokens int) (string, []string, error) {
	llmMu.RLock()
	pKey := llmPerplexityKey
	llmMu.RUnlock()
	useKey := pKey
	if useKey == "" {
		useKey = apiKey
	}
	pModel := model
	switch model {
	case "llama-3.3-70b-versatile", "llama-3.1-70b-versatile":
		pModel = pplxChatModel
	case "llama-3.1-8b-instant", "llama-3.2-3b-preview":
		pModel = pplxFastModel
	}
	text, _, err := callOpenAICompat(useKey, pplxAPIBase, pModel, msgs, maxTokens, false)
	if err != nil {
		return "", nil, err
	}
	lastCitationsMu.Lock()
	cites := make([]string, len(lastCitations))
	copy(cites, lastCitations)
	lastCitationsMu.Unlock()
	return text, cites, nil
}

// callGroq: Perplexity API 호출 (groqChatModel/groqFastModel은 pplx 별칭)
func callGroq(apiKey, model string, msgs []groqMsg, maxTokens int, jsonMode bool) (string, int, error) {
	llmMu.RLock()
	pKey := llmPerplexityKey
	llmMu.RUnlock()
	useKey := pKey
	if useKey == "" {
		useKey = apiKey
	}
	// 구 Groq 모델명이 넘어와도 Perplexity 모델로 교정
	pModel := model
	switch model {
	case "llama-3.3-70b-versatile", "llama-3.1-70b-versatile":
		pModel = pplxChatModel
	case "llama-3.1-8b-instant", "llama-3.2-3b-preview":
		pModel = pplxFastModel
	}
	return callOpenAICompat(useKey, pplxAPIBase, pModel, msgs, maxTokens, jsonMode)
}

func callGroqWithFallback(msgs []groqMsg, maxTokens int, jsonMode bool) (string, string, error) {
	llmMu.RLock()
	pKey := llmPerplexityKey
	llmMu.RUnlock()
	if pKey == "" {
		return "", "", fmt.Errorf("Perplexity API 키가 설정되지 않았습니다")
	}
	text, _, err := callOpenAICompat(pKey, pplxAPIBase, pplxChatModel, msgs, maxTokens, jsonMode)
	if err != nil {
		return "", "", err
	}
	return text, "perplexity", nil
}

// ClarifyResult: Groq Structured Outputs로 받는 Clarify 판단 결과
type ClarifyResult struct {
	NeedsClarify    bool    `json:"needs_clarify"`
	ClarifyQuestion string  `json:"clarify_question"`
	Action          string  `json:"action"`
	Confidence      float64 `json:"confidence"`
}

// actionRequiredFields: 각 액션별 필수 파라미터와 설명
var actionRequiredFields = map[string]string{
	"price_compare": "상품명 또는 카테고리 (예: 에어팟 프로 2, 갤럭시 S25, 다이슨 청소기) — 쇼핑몰 이름만으로는 부족함",
	"video_search":  "검색할 주제/키워드 (예: 요리, 주식 투자 전략, 운동) — 플랫폼 이름만으로는 부족함",
	"trip_plan":     "목적지 도시명 (예: 도쿄, 뉴욕, 싱가포르) AND 날짜/시기 (예: 내일, 다음주, 5월 20일)",
	"web_search":    "검색할 구체적인 내용 (맛집이면 지역, 추천이면 카테고리)",
	"calendar_add":  "일정 제목 AND 날짜",
	"weather":       "도시명 또는 지역명",
	"multi_action":  "비교/요약할 구체적인 대상이나 주제",
}

// callGroqStructured: Groq json_schema strict mode로 Clarify 여부를 판단
func callGroqStructured(userMsg string) (*ClarifyResult, error) {
	llmMu.RLock()
	gKey := llmGroqKey
	llmMu.RUnlock()
	if gKey == "" {
		return nil, fmt.Errorf("groq key not set")
	}

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"needs_clarify": map[string]any{
				"type":        "boolean",
				"description": "true if essential information is missing and user must be asked",
			},
			"clarify_question": map[string]any{
				"type":        "string",
				"description": "The single friendly question to ask user. Empty string if needs_clarify is false.",
			},
			"action": map[string]any{
				"type":        "string",
				"description": "Predicted action: price_compare|video_search|trip_plan|web_search|calendar_add|weather|chat|multi_action",
			},
			"confidence": map[string]any{
				"type":        "number",
				"description": "0.0~1.0. How confident the intent is understood. Below 0.6 should trigger clarify.",
			},
		},
		"required":             []string{"needs_clarify", "clarify_question", "action", "confidence"},
		"additionalProperties": false,
	}

	sysPrompt := `당신은 AI 비서의 Clarification 판단 엔진입니다.
사용자 요청에서 실행에 필수적인 정보가 빠져 있으면 반드시 needs_clarify=true로 설정하세요.
"최선 추론"은 절대 금지 — 정보가 없으면 항상 물어보세요.

액션별 필수 정보 (없으면 무조건 clarify):
- price_compare: 구체적인 상품명 또는 카테고리 (단순히 "찾아줘", "사고싶어"만 있으면 clarify)
- video_search: 검색할 주제 또는 키워드 (플랫폼만 있고 주제가 없으면 clarify)
- trip_plan: 목적지 도시명 (없으면 clarify), 날짜 맥락 (없으면 clarify)
- web_search + 맛집: 지역/동네 (없으면 clarify)
- web_search + 추천: 추천받을 카테고리/종류 (없으면 clarify)
- calendar_add: 일정 제목, 날짜 (둘 중 하나라도 없으면 clarify)
- weather: 도시명 또는 지역 맥락 (없으면 clarify)
- multi_action + 비교: 비교할 대상 2개 이상 (없으면 clarify)
- multi_action + 요약: 요약할 주제나 내용 (없으면 clarify)
- confidence < 0.65: 의도 자체가 불명확 → clarify

[감지된 액션]이 컨텍스트에 있으면 해당 액션의 필수 정보가 있는지 더 엄격하게 확인하세요.

clarify_question: 한국어, 1문장, 구체적인 예시 포함 (예: "어떤 상품을 찾아드릴까요? (예: 에어팟 프로 2, 갤럭시 S25)")
needs_clarify=false이면 clarify_question은 ""로 설정.`

	type reqBody struct {
		Model          string         `json:"model"`
		Messages       []groqMsg      `json:"messages"`
		Temperature    float64        `json:"temperature"`
		MaxTokens      int            `json:"max_tokens"`
		ResponseFormat map[string]any `json:"response_format"`
	}

	rb := reqBody{
		Model: groqStructuredModel,
		Messages: []groqMsg{
			{Role: "system", Content: sysPrompt},
			{Role: "user", Content: userMsg},
		},
		Temperature: 0.0,
		MaxTokens:   200,
		ResponseFormat: map[string]any{
			"type": "json_schema",
			"json_schema": map[string]any{
				"name":   "clarify_check",
				"strict": true,
				"schema": schema,
			},
		},
	}

	body, _ := json.Marshal(rb)
	httpReq, _ := http.NewRequest("POST", groqRealAPIBase, bytes.NewReader(body))
	httpReq.Header.Set("Authorization", "Bearer "+gKey)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("groq 연결 실패: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	var gr struct {
		Choices []struct {
			Message struct{ Content string `json:"content"` } `json:"message"`
		} `json:"choices"`
		Error *struct{ Message string `json:"message"` } `json:"error"`
	}
	if err := json.Unmarshal(raw, &gr); err != nil {
		return nil, fmt.Errorf("groq 응답 파싱 실패: %w", err)
	}
	if gr.Error != nil {
		return nil, fmt.Errorf("groq 오류: %s", gr.Error.Message)
	}
	if len(gr.Choices) == 0 {
		return nil, fmt.Errorf("groq 응답 없음")
	}

	var result ClarifyResult
	if err := json.Unmarshal([]byte(gr.Choices[0].Message.Content), &result); err != nil {
		return nil, fmt.Errorf("clarify JSON 파싱 실패: %w", err)
	}
	return &result, nil
}

func callGroqVision(_, _, _, _ string) (string, error) {
	return "", fmt.Errorf("Vision 기능은 현재 지원되지 않습니다")
}

func callClaude(apiKey string, msgs []groqMsg, maxTokens int) (string, error) {
	return "", fmt.Errorf("Claude 미구현")
}

func deepSearchFiles(_, _ string, _ int) []DeepSearchResult { return nil }

func max2(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func handleLLMConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		llmMu.RLock()
		pSet := llmPerplexityKey != ""
		cSet := llmClaudeKey != ""
		tSet := llmTavilyKey != ""
		llmMu.RUnlock()
		json200(w, map[string]any{
			"perplexity_configured": pSet,
			"claude_configured":     cSet,
			"tavily_configured":     tSet,
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
		ApiKey        string `json:"apiKey"`
		ClaudeKey     string `json:"claude_key"`
		TavilyKey     string `json:"tavily_key"`
		GroqKey       string `json:"groq_key"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.PerplexityKey == "" && req.ApiKey != "" {
		req.PerplexityKey = req.ApiKey
	}
	llmMu.Lock()
	if s := strings.TrimSpace(req.PerplexityKey); s != "" {
		llmPerplexityKey = s
	}
	if s := strings.TrimSpace(req.ClaudeKey); s != "" {
		llmClaudeKey = s
	}
	if s := strings.TrimSpace(req.TavilyKey); s != "" {
		llmTavilyKey = s
	}
	if s := strings.TrimSpace(req.GroqKey); s != "" {
		llmGroqKey = s
	}
	llmMu.Unlock()
	saveLLMConfig()
	json200(w, map[string]any{"success": true, "message": "API 키 저장 완료"})
}

func handleLLMChat(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Messages  []groqMsg `json:"messages"`
		MaxTokens int       `json:"max_tokens"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Messages) == 0 {
		writeJSON(w, 400, map[string]any{"success": false, "message": "messages 필요"})
		return
	}
	if req.MaxTokens == 0 {
		req.MaxTokens = 1024
	}
	text, provider, err := callGroqWithFallback(req.Messages, req.MaxTokens, false)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}
	json200(w, map[string]any{"success": true, "text": text, "provider": provider})
}

func handleLLMDeepSearch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query string `json:"query"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Query == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "query 필요"})
		return
	}
	// Tavily 웹 검색 후 LLM 요약
	tvResult, _ := tavilySearch(llmTavilyKey, req.Query, 5)
	searchResults := tvResult.Summary
	prompt := "다음 웹 검색 결과를 바탕으로 '" + req.Query + "'에 대해 한국어로 명확하게 요약해줘:\n\n" + searchResults
	answer, _, err := callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 800, false)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}
	json200(w, map[string]any{"success": true, "answer": answer, "query": req.Query})
}
