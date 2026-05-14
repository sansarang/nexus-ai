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
	NeedsClarify     bool     `json:"needs_clarify"`
	ClarifyQuestions []string `json:"clarify_questions"`
	Action           string   `json:"action"`
	Confidence       float64  `json:"confidence"`
	Reason           string   `json:"reason"`
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

// callGroqStructured: Groq json_object 모드로 Clarify 여부를 판단
// json_schema strict 모드는 Llama의 instruction-following을 억제함 → json_object 사용
func callGroqStructured(userMsg string) (*ClarifyResult, error) {
	llmMu.RLock()
	gKey := llmGroqKey
	llmMu.RUnlock()
	if gKey == "" {
		return nil, fmt.Errorf("groq key not set")
	}

	sysPrompt := `You classify Korean AI assistant requests as needing clarification or not.
Output JSON: {"needs_clarify": bool, "clarify_question": "Korean question", "reason": "brief"}

## ALWAYS clarify these exact requests (no exceptions, highest priority):
- "선물 뭐가 좋을까" → true (누구에게, 예산 없음)
- "출장 계획 짜줘" → true (목적지 없음)
- "할 일 정리해줘" → true (AI가 할 일 목록을 모름, 반드시 물어볼 것)
- "여행 일정 만들어줘" → true (목적지/날짜 없음)
- "동남아 여행 가고 싶어" → true (구체적 나라 없음)

## ALWAYS execute these patterns (no exceptions):
- "오늘 오후 N시 치과" or "치과 예약 넣어줘" with time → false (치과 이름 절대 묻지 말 것)
- "유튜브에서 [주제] 영상" → false (주제가 있으면 바로 실행)

## CRITICAL "DO NOT ASK" rules (override everything):
- If destination+duration exists (도쿄 3박 4일, 제주도 1박 2일, 다음달 제주도) → NEVER ask for exact dates → false
- If city name exists (서울, 부산, 도쿄) for weather → NEVER ask for date → false
- If time+event exists (오후 2시 치과, 오전 10시 회의) → NEVER ask for clinic/venue name → false
- If "근처" appears → NEVER ask for location (근처=GPS) → false
- If system task (바탕화면 정리, 중복 파일) → NEVER ask for file details → false
- If named recipient+clear purpose (팀장님한테 반차 이메일) → NEVER ask for content → false

## needs_clarify=true (key info TRULY missing):
쿠팡에서 찾아줘 → true
11번가에서 사고 싶어 → true
노트북 추천해줘 → true (no budget/purpose)
선물 뭐가 좋을까 → true
유튜브 찾아줘 → true (no topic)
틱톡 영상 찾아줘 → true (no topic — same rule as 유튜브 찾아줘)
넷플릭스 뭐 볼까 → true (no genre)
출장 계획 짜줘 → true (no destination)
여행 일정 만들어줘 → true (no destination)
혼자 여행하기 좋은 곳 추천해줘 → true
동남아 여행 가고 싶어 → true (no specific country)
맛집 추천해줘 → true (no location)
데이트 코스 알려줘 → true (no location)
주말 가족 나들이 장소 추천 → true (no location)
날씨 어때 → true (no city)
오늘 비 와? → true (no city)
일정 추가해줘 → true
내일 회의 일정 추가해줘 → true (no time)
회의 일정 잡아줘 → true (no date/time)
파일 찾아줘 → true (no filename)
문서 요약해줘 → true
이메일 보내줘 → true (no recipient)
검색해줘 → true
도와줘 → true
할 일 정리해줘 → true
요즘 핫한 거 뭐야 → true
비교해줘 → true

## needs_clarify=false (sufficient info):
쿠팡에서 에어팟 프로 2 최저가 찾아줘 → false
다나와에서 RTX 4070 가격 비교해줘 → false
30만원 이하 무선 이어폰 추천해줘 → false
갤럭시 S25 울트라 사고 싶어 → false
강남 한식 맛집 추천해줘 → false
근처 카페 찾아줘 → false
홍대 근처 이탈리안 레스토랑 → false
서울 날씨 알려줘 → false
내일 부산 날씨 → false
도쿄 3박 4일 여행 일정 짜줘 → false
다음달 제주도 1박 2일 계획 세워줘 → false
유튜브에서 주식 투자 영상 찾아줘 → false
틱톡에서 요리 레시피 영상 보여줘 → false
유튜브 인기 영상 보여줘 → false
오늘 오후 2시 치과 예약 넣어줘 → false
5월 20일 오후 3시 팀 회의 일정 잡아줘 → false
팀장님한테 오늘 오후 반차 이메일 보내줘 → false
아이폰 16 vs 갤럭시 S25 비교해줘 → false
바탕화면 정리해줘 → false
중복 파일 찾아줘 → false
삼성전자 주가 알려줘 → false
파이썬 리스트 정렬하는 법 알려줘 → false
최근 뉴스 보여줘 → false`

	type reqBody struct {
		Model          string         `json:"model"`
		Messages       []groqMsg      `json:"messages"`
		Temperature    float64        `json:"temperature"`
		MaxTokens      int            `json:"max_tokens"`
		ResponseFormat map[string]any `json:"response_format"`
	}

	msgs := []groqMsg{
		{Role: "system", Content: sysPrompt},
		{Role: "user", Content: userMsg},
	}

	rb := reqBody{
		Model:       groqStructuredModel,
		Messages:    msgs,
		Temperature: 0.0,
		MaxTokens:   150,
		ResponseFormat: map[string]any{
			"type": "json_object",
		},
	}

	body, _ := json.Marshal(rb)

	client := &http.Client{Timeout: 15 * time.Second}
	var raw []byte
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt*2) * time.Second)
		}
		httpReq, _ := http.NewRequest("POST", groqRealAPIBase, bytes.NewReader(body))
		httpReq.Header.Set("Authorization", "Bearer "+gKey)
		httpReq.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(httpReq)
		if err != nil {
			if attempt == 2 {
				return nil, fmt.Errorf("groq 연결 실패: %w", err)
			}
			continue
		}
		raw, _ = io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode == 429 {
			// rate limit → retry with short backoff
			if attempt == 2 {
				return nil, fmt.Errorf("groq rate limit")
			}
			time.Sleep(1 * time.Second)
			continue
		}
		break
	}

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

	// json_object 모드: clarify_question (string) 또는 clarify_questions (array) 모두 처리
	var raw2 struct {
		NeedsClarify     bool     `json:"needs_clarify"`
		ClarifyQuestion  string   `json:"clarify_question"`
		ClarifyQuestions []string `json:"clarify_questions"`
		Action           string   `json:"action"`
		Confidence       float64  `json:"confidence"`
		Reason           string   `json:"reason"`
	}
	if err := json.Unmarshal([]byte(gr.Choices[0].Message.Content), &raw2); err != nil {
		return nil, fmt.Errorf("clarify JSON 파싱 실패: %w", err)
	}
	result := &ClarifyResult{
		NeedsClarify: raw2.NeedsClarify,
		Action:       raw2.Action,
		Confidence:   raw2.Confidence,
		Reason:       raw2.Reason,
	}
	// string → array 통합
	if len(raw2.ClarifyQuestions) > 0 {
		result.ClarifyQuestions = raw2.ClarifyQuestions
	} else if raw2.ClarifyQuestion != "" {
		result.ClarifyQuestions = []string{raw2.ClarifyQuestion}
	}
	return result, nil
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
