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
	llmShodanKey     string // Shodan API 키 (선택적)
	llmUserLang      string // "ko" | "en" — 영속 사용자 언어 설정
)

// GetUserLang: 저장된 사용자 언어 반환 — 모든 기능이 이걸 사용
func GetUserLang() string {
	llmMu.RLock()
	defer llmMu.RUnlock()
	if llmUserLang == "en" {
		return "en"
	}
	return "ko"
}

// IsUserEng: 영어 사용자 여부 편의 함수
func IsUserEng() bool { return GetUserLang() == "en" }

type llmConfigFile struct {
	PerplexityKey string `json:"perplexity_key"`
	ClaudeKey     string `json:"claude_key"`
	TavilyKey     string `json:"tavily_key"`
	GroqKey       string `json:"groq_key"`
	UserLang      string `json:"user_lang"`
}

func llmConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".nexus", "llm_config.json")
}

// ── 번들 기본 API 키 (설치 즉시 작동 — 사용자 설정 불필요) ──────────────
// Nexus 서비스 계정 키. 사용자별 quota는 handlers_usage.go에서 관리.
// llmPerplexityKey = Groq 키 공유 (pplxAPIBase → Groq API endpoint 사용)
const (
	bundledGroqKey   = "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"
	bundledTavilyKey = "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"
	bundledOpenAIKey = "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"
)

// injectBundledKeys: 사용자 키 미설정 시 번들 기본 키 자동 주입
func injectBundledKeys() {
	llmMu.Lock()
	defer llmMu.Unlock()
	// Perplexity와 Groq는 동일 키 사용 (코드에서 pplxAPIBase → Groq endpoint로 매핑)
	if llmGroqKey == "" {
		llmGroqKey = bundledGroqKey
	}
	if llmPerplexityKey == "" {
		llmPerplexityKey = bundledGroqKey // Groq 키를 Perplexity 슬롯에도 주입
	}
	if llmTavilyKey == "" {
		llmTavilyKey = bundledTavilyKey
	}
	if llmClaudeKey == "" {
		llmClaudeKey = bundledOpenAIKey // OpenAI 키를 Claude 슬롯(폴백용)에도 주입
	}
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
		llmMu.Lock()
		llmUserLang = detectSystemLang()
		llmMu.Unlock()
		// 저장된 설정 없으면 번들 기본 키 주입 → 즉시 사용 가능
		injectBundledKeys()
		return
	}
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
	// 2순위: 설정 파일 로드 후 여전히 빈 키 → 번들 기본 키로 보완
	injectBundledKeys()
}

func saveLLMConfig() {
	llmMu.RLock()
	cfg := map[string]string{
		"perplexity_key": llmPerplexityKey,
		"claude_key":     llmClaudeKey,
		"tavily_key":     llmTavilyKey,
		"groq_key":       llmGroqKey,
		"shodan_key":     llmShodanKey,
		"user_lang":      llmUserLang,
	}
	llmMu.RUnlock()
	data, _ := json.MarshalIndent(cfg, "", "  ")
	os.MkdirAll(filepath.Dir(llmConfigPath()), 0755)
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

// resolveEndpointAndModel: API 키 타입에 따라 올바른 엔드포인트 + 모델 자동 결정
// gsk_ → Groq API + Groq 모델
// pplx- → Perplexity API + Perplexity 모델
// 그 외 → Groq 기본값
func resolveEndpointAndModel(key, requestedModel string, fast bool) (endpoint, model string) {
	isGroqKey := strings.HasPrefix(key, "gsk_")
	if isGroqKey {
		endpoint = groqAPIBase
		if fast {
			model = "llama-3.1-8b-instant"
		} else {
			model = "llama-3.3-70b-versatile"
		}
		// 요청 모델이 이미 Groq 모델이면 그대로 사용
		switch requestedModel {
		case "llama-3.3-70b-versatile", "llama-3.1-70b-versatile",
			"llama-3.1-8b-instant", "llama-3.2-3b-preview",
			"llama-4-scout-17b-16e-instruct", "llama-4-maverick-17b-128e-instruct":
			model = requestedModel
		}
	} else {
		endpoint = pplxAPIBase
		if fast {
			model = pplxFastModel
		} else {
			model = pplxChatModel
		}
		// 구 Groq 모델명이 넘어오면 Perplexity로 교정
		switch requestedModel {
		case pplxChatModel, pplxFastModel:
			model = requestedModel
		}
	}
	return
}

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

func callGroqWithFallback(msgs []groqMsg, maxTokens int, jsonMode bool) (string, string, error) {
	// 1순위: Supabase Edge Function 프록시
	if content, err := callGroqViaProxy(msgs, maxTokens, jsonMode); err == nil {
		return content, "groq-proxy", nil
	}

	// 2순위: 번들 키 직접 호출
	llmMu.RLock()
	key := llmPerplexityKey
	if key == "" {
		key = llmGroqKey
	}
	llmMu.RUnlock()
	if key == "" {
		return "", "", fmt.Errorf("API 키가 설정되지 않았습니다 (Groq 또는 Perplexity)")
	}
	endpoint, model := resolveEndpointAndModel(key, "", false)
	text, _, err := callOpenAICompat(key, endpoint, model, msgs, maxTokens, jsonMode)
	if err != nil {
		return "", "", err
	}
	provider := "perplexity"
	if strings.HasPrefix(key, "gsk_") {
		provider = "groq"
	}
	return text, provider, nil
}

// IntentItem: 단일 intent (action + params)
type IntentItem struct {
	Action      string         `json:"action"`
	Params      map[string]any `json:"params"`
	Description string         `json:"description"`
}

// ClarifyResult: Groq Structured Outputs로 받는 Clarify 판단 결과
type ClarifyResult struct {
	NeedsClarify     bool         `json:"needs_clarify"`
	ClarifyQuestions []string     `json:"clarify_questions"`
	Action           string       `json:"action"`
	Intents          []IntentItem `json:"intents"`
	Confidence       float64      `json:"confidence"`
	Reason           string       `json:"reason"`
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

	sysPrompt := `You are an intent classifier for a Korean AI assistant. Analyze the user's request and output JSON.

Output format (STRICT — always include all keys):
{"needs_clarify":false,"clarify_question":"","intents":[{"action":"action_name","params":{},"description":"brief"}]}

## Available actions:
- "web_search": params: {"query":"검색어","site":"google|auto"}  — for news, info, recommendations, directions, timetables, hotel search, ANY informational query
- "weather": params: {"city":"도시명"}
- "trip_plan": params: {"destination":"목적지","date":"YYYY-MM-DD","days":1,"purpose":"여행|출장"}
- "calendar_add": params: {"title":"제목","date":"YYYY-MM-DD","time":"HH:MM"}
- "calendar_today": params: {}
- "price_compare": params: {"query":"상품명","site":"auto|coupang.com|..."}
- "video_search": params: {"query":"키워드","platform":"youtube|tiktok"}
- "exchange_rate": params: {"from":"USD","to":"KRW"}  — 환율/달러/엔화/유로/위안/원/환전
- "stock": params: {"query":"삼성전자 주가|비트코인|코스피|TSLA"}  — 주가/코스피/나스닥/암호화폐/코인
- "deep_research": params: {"query":"자세히 알아볼 주제"}  — 심층리서치/자세히/깊게/분석해줘/리서치해줘/조사해줘
- "file_ops": params: {"op":"organize|duplicates|large","folder":"바탕화면|다운로드|Documents"}  — 파일정리/중복파일/용량
- "trigger_add": params: {"nl":"CPU 80% 넘으면 알려줘"}  — 조건부 알림 (CPU/메모리/시간/주기)
- "screen_analyze": params: {"question":"화면에 뭐가 있어?"}  — 화면 캡처+Vision 분석
- "launch_app": params: {"app_name":"크롬|카카오톡|메모장|엑셀"}  — 앱 실행
- "clipboard_action": params: {"action":"translate|summarize|proofread|explain|analyze_code|rewrite|translate_en|translate_ko"}  — 클립보드/복사한 내용 처리
- "chat": params: {}  — for coding questions, explanations, general knowledge

## MULTI-INTENT: If user asks multiple things in one message, return MULTIPLE items in intents[].
Examples:
"부산에서 경주 가는 교통편이랑 경주 호텔도 알려줘" →
{"needs_clarify":false,"clarify_question":"","intents":[{"action":"web_search","params":{"query":"부산 경주 교통편 시간표 KTX 버스","site":"auto"},"description":"교통편"},{"action":"web_search","params":{"query":"경주 호텔 추천","site":"auto"},"description":"호텔"}]}

"내일 도쿄 날씨랑 맛집도 알려줘" →
{"needs_clarify":false,"clarify_question":"","intents":[{"action":"weather","params":{"city":"도쿄"},"description":"날씨"},{"action":"web_search","params":{"query":"도쿄 맛집 추천","site":"auto"},"description":"맛집"}]}

## KEY RULE — "알려줘", "찾아줘", "보여줘", "추천해줘" = informational request → web_search or appropriate action. NEVER treat as file save.

## needs_clarify=true ONLY when critical info is truly missing (return intents=[]):
- "쿠팡에서 찾아줘" (no product) → true, ask "어떤 제품을 찾으시나요?"
- "노트북 추천해줘" (no budget/purpose) → true
- "맛집 추천해줘" (no location) → true
- "날씨 어때" (no city) → true
- "출장 계획 짜줘" (no destination) → true
- "여행 일정 만들어줘" (no destination) → true
- "동남아 여행 가고 싶어" (region too vague) → true
- "검색해줘", "도와줘", "비교해줘", "이메일 보내줘" (no object) → true

## needs_clarify=false (execute immediately, no questions):
- Specific product + site → price_compare
- City name present → weather: false
- Destination + duration/date → trip_plan or web_search: false
- Topic present for YouTube/TikTok → video_search: false
- "근처" → use GPS, never ask location: false
- 숙소/호텔 + 지역 + 기간 → web_search: false (DO NOT ask for more info)
- Coding question, explanation request → chat: false
- "삼성전자 주가", "파이썬 리스트 정렬" → web_search or chat: false

## PRONOUN RESOLUTION: If message contains "[이전 대화 컨텍스트: X]" OR "[Previous context: X]", use X to resolve the intent.
Example KO: "[이전 대화 컨텍스트: 도쿄 맛집 검색]\n현재 질문: 그거 유튜브 영상도 찾아줘" →
{"needs_clarify":false,"clarify_question":"","intents":[{"action":"video_search","params":{"query":"도쿄 맛집","platform":"youtube"},"description":"도쿄 맛집 유튜브 영상"}]}
Example EN: "[Previous context: Tokyo restaurants]\nCurrent question: find more about that" →
{"needs_clarify":false,"clarify_question":"","intents":[{"action":"web_search","params":{"query":"Tokyo restaurants guide","site":"auto"},"description":"more Tokyo restaurant info"}]}

## EXCHANGE RATE:
- "환율", "달러 얼마", "엔화", "원달러", "환율 알려줘", "USD", "EUR", "JPY" → use "web_search" with query="현재 환율 USD KRW" or similar
- "삼성전자 주가", "코스피", "나스닥", "비트코인" → web_search with current stock query

## COMPOUND COMMANDS: If user wants to do TWO DIFFERENT THINGS, always return 2 intents.
"날씨 알려주고 맛집도 찾아줘" → 2 intents: weather + web_search
"주가 확인하고 뉴스도 알려줘" → 2 intents: web_search(주가) + web_search(뉴스)
"유튜브 영상 찾고 가격도 비교해줘" → 2 intents: video_search + price_compare

## AMBIGUOUS QUERIES — smart defaults (needs_clarify=false):
- "요즘 뭐가 핫해?" → web_search: {"query":"2026년 트렌드 핫이슈"}
- "추천해줘" (with prior context from [이전 대화 컨텍스트]) → use context as query
- "그거 더 알아봐줘" (with context) → web_search with context as query
- "어때?" / "좋아?" (with context) → web_search or chat based on context
- "정리해줘" (with context) → web_search + summarize
- Slang/informal: "ㅋㅋ 그거 찾아봐" → web_search
- Mixed Korean-English: "요즘 hot한 AI startup" → web_search: {"query":"2026 AI startup trends Korea"}
- "오늘 환율" → web_search: {"query":"오늘 달러 원화 환율"}
- "비트코인 지금" → web_search: {"query":"비트코인 현재 가격 KRW"}
- "삼성 주가" → web_search: {"query":"삼성전자 주가 현재"}
- "날씨" (no city, user in Korea) → weather: {"city":"서울"} (default Seoul)
- "근처 맛집" → web_search: {"query":"서울 근처 맛집 추천"} — NEVER clarify for location if "근처"
- "유명한 거" / "인기 있는 거" → web_search with topic from context or "인기 트렌드"
- App run requests ("카카오톡 열어줘", "크롬 켜줘") → chat: {"message":"앱 실행은 현재 지원되지 않습니다."}
- Calculation ("1달러 오늘 환율로") → web_search with exchange rate query`

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
		MaxTokens:   400,
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

	var raw2 struct {
		NeedsClarify     bool         `json:"needs_clarify"`
		ClarifyQuestion  string       `json:"clarify_question"`
		ClarifyQuestions []string     `json:"clarify_questions"`
		Action           string       `json:"action"`
		Intents          []IntentItem `json:"intents"`
		Confidence       float64      `json:"confidence"`
		Reason           string       `json:"reason"`
	}
	if err := json.Unmarshal([]byte(gr.Choices[0].Message.Content), &raw2); err != nil {
		return nil, fmt.Errorf("clarify JSON 파싱 실패: %w", err)
	}
	result := &ClarifyResult{
		NeedsClarify: raw2.NeedsClarify,
		Action:       raw2.Action,
		Intents:      raw2.Intents,
		Confidence:   raw2.Confidence,
		Reason:       raw2.Reason,
	}
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

// callClaudeIntent: Claude Haiku로 intent 분류 (callGroqStructured 대체)
func callClaudeIntent(userMsg string) (*ClarifyResult, error) {
	llmMu.RLock()
	cKey := llmClaudeKey
	llmMu.RUnlock()
	if cKey == "" {
		return nil, fmt.Errorf("claude key not set")
	}

	sysPrompt := `You are an intent classifier for a Korean AI assistant. Analyze the user's request and output JSON.

Output format (STRICT — always include all keys):
{"needs_clarify":false,"clarify_question":"","intents":[{"action":"action_name","params":{},"description":"brief"}]}

Available actions:
- "web_search": params: {"query":"검색어","site":"google|auto"}  — for news, info, recommendations, directions, timetables, hotel search, ANY informational query
- "weather": params: {"city":"도시명"}
- "trip_plan": params: {"destination":"목적지","date":"YYYY-MM-DD","days":1,"purpose":"여행|출장"}
- "calendar_add": params: {"title":"제목","date":"YYYY-MM-DD","time":"HH:MM"}
- "calendar_today": params: {}
- "price_compare": params: {"query":"상품명","site":"auto|coupang.com"}
- "video_search": params: {"query":"키워드","platform":"youtube|tiktok"}
- "clipboard_action": params: {"action":"translate|summarize|proofread|explain|analyze_code|rewrite|translate_en|translate_ko"}
- "chat": params: {}  — for coding questions, explanations, general knowledge

## CLIPBOARD DETECTION (clipboard_action):
Trigger when message contains ANY of: "복사한", "클립보드", "방금 복사", "복붙", "붙여넣은", "copied", "clipboard", "paste", "just copied"
Examples:
"방금 복사한 거 번역해줘" → {"action":"clipboard_action","params":{"action":"translate"}}
"클립보드 내용 요약해줘" → {"action":"clipboard_action","params":{"action":"summarize"}}
"복사한 코드 분석해줘" → {"action":"clipboard_action","params":{"action":"analyze_code"}}
"방금 복사한 거 영어로" → {"action":"clipboard_action","params":{"action":"translate_en"}}
"클립보드 맞춤법 교정해줘" → {"action":"clipboard_action","params":{"action":"proofread"}}
"이거 다시 써줘" (with clipboard context) → {"action":"clipboard_action","params":{"action":"rewrite"}}

MULTI-INTENT: If user asks multiple things, return MULTIPLE items in intents[].
Example: "부산에서 경주 교통편이랑 호텔도 알려줘" →
{"needs_clarify":false,"clarify_question":"","intents":[{"action":"web_search","params":{"query":"부산 경주 교통편 시간표","site":"auto"},"description":"교통편"},{"action":"web_search","params":{"query":"경주 호텔 추천","site":"auto"},"description":"호텔"}]}

KEY RULES:
- "알려줘","찾아줘","보여줘","추천해줘" = informational → web_search (NEVER file save)
- 숙소/호텔 + 지역 있으면 → web_search, needs_clarify=false
- 도시명 있으면 날씨 → needs_clarify=false
- 목적지+기간 있으면 trip → needs_clarify=false
- 코딩/설명 질문 → chat
- "근처" → GPS 사용, 절대 위치 묻지 말 것

needs_clarify=true ONLY when critical info is TRULY missing:
- "맛집 추천해줘" (지역 없음), "날씨 어때" (도시 없음), "출장 계획 짜줘" (목적지 없음)
- "여행 일정 만들어줘" (목적지 없음), "동남아 여행 가고 싶어" (너무 모호)
- "검색해줘","도와줘","비교해줘","이메일 보내줘" (대상 없음)`

	body, _ := json.Marshal(map[string]any{
		"model":      claudeHaikuModel,
		"max_tokens": 512,
		"system":     sysPrompt,
		"messages":   []map[string]any{{"role": "user", "content": userMsg}},
	})

	client := &http.Client{Timeout: 15 * time.Second}
	httpReq, _ := http.NewRequest("POST", claudeAPIBase, bytes.NewReader(body))
	httpReq.Header.Set("x-api-key", cKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("content-type", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("claude 연결 실패: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	var cr struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Error *struct{ Message string `json:"message"` } `json:"error"`
	}
	if err := json.Unmarshal(raw, &cr); err != nil {
		return nil, fmt.Errorf("claude 응답 파싱 실패: %w", err)
	}
	if cr.Error != nil {
		return nil, fmt.Errorf("claude 오류: %s", cr.Error.Message)
	}
	if len(cr.Content) == 0 {
		return nil, fmt.Errorf("claude 응답 없음")
	}

	text := cr.Content[0].Text
	// JSON 블록 추출
	if idx := strings.Index(text, "{"); idx >= 0 {
		text = text[idx:]
	}
	if idx := strings.LastIndex(text, "}"); idx >= 0 {
		text = text[:idx+1]
	}

	var raw2 struct {
		NeedsClarify    bool         `json:"needs_clarify"`
		ClarifyQuestion string       `json:"clarify_question"`
		Intents         []IntentItem `json:"intents"`
	}
	if err := json.Unmarshal([]byte(text), &raw2); err != nil {
		return nil, fmt.Errorf("claude intent JSON 파싱 실패: %w", err)
	}

	result := &ClarifyResult{
		NeedsClarify: raw2.NeedsClarify,
		Intents:      raw2.Intents,
	}
	if raw2.ClarifyQuestion != "" {
		result.ClarifyQuestions = []string{raw2.ClarifyQuestion}
	}
	return result, nil
}

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

// POST /api/translate/realtime — 실시간 타이핑 번역 (500ms 디바운스용)
func handleTranslateRealtime(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Text string `json:"text"`
		From string `json:"from"` // "ko" | "en" | "auto"
		To   string `json:"to"`   // "ko" | "en"
	}
	json.NewDecoder(r.Body).Decode(&req)
	if strings.TrimSpace(req.Text) == "" {
		json200(w, map[string]any{"success": true, "translation": ""})
		return
	}
	// 언어 방향 결정
	from := req.From
	to := req.To
	if from == "" || from == "auto" {
		// 한글 문자 포함 여부로 판단
		hasKorean := false
		for _, r := range req.Text {
			if r >= 0xAC00 && r <= 0xD7A3 {
				hasKorean = true
				break
			}
		}
		if hasKorean {
			from = "ko"
			to = "en"
		} else {
			from = "en"
			to = "ko"
		}
	}
	var prompt string
	if from == "ko" && to == "en" {
		prompt = fmt.Sprintf("Translate to English. Output only the translation, nothing else:\n%s", req.Text)
	} else {
		prompt = fmt.Sprintf("한국어로 번역해줘. 번역문만 출력:\n%s", req.Text)
	}
	result, _, err := callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 256, false)
	if err != nil {
		json200(w, map[string]any{"success": false, "translation": ""})
		return
	}
	json200(w, map[string]any{"success": true, "translation": strings.TrimSpace(result), "from": from, "to": to})
}

// POST /api/llm/vision — Mac stub (Vision은 Windows에서 실제 구현)
func handleLLMVision(w http.ResponseWriter, r *http.Request) {
	eng := IsUserEng()
	msg := "Vision (image analysis) is not supported on this platform."
	if !eng {
		msg = "Vision(이미지 분석) 기능은 현재 지원되지 않습니다."
	}
	writeJSON(w, 400, map[string]any{"success": false, "message": msg})
}

// POST /api/llm/doc-summary — Mac stub
func handleLLMDocSummary(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FilePath string `json:"file_path"`
		Question string `json:"question"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.FilePath == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "file_path required"})
		return
	}
	eng := IsUserEng()
	question := req.Question
	if question == "" {
		if eng {
			question = "Summarize the key contents of this document in 5 lines and list important figures, dates, and names."
		} else {
			question = "이 문서의 핵심 내용을 5줄로 요약하고, 중요 수치·날짜·이름을 목록으로 정리해주세요."
		}
	}
	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()
	if gKey == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "API key not configured"})
		return
	}
	prompt := fmt.Sprintf("File: %s\n\nRequest: %s", req.FilePath, question)
	answer, _, err := callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 1024, false)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}
	json200(w, map[string]any{"success": true, "summary": answer, "file": req.FilePath})
}

// POST /api/llm/doc-compare — Mac stub
func handleLLMDocCompare(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FileA string `json:"file_a"`
		FileB string `json:"file_b"`
		Focus string `json:"focus"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.FileA == "" || req.FileB == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "file_a and file_b required"})
		return
	}
	eng := IsUserEng()
	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()
	if gKey == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "API key not configured"})
		return
	}
	var prompt string
	if eng {
		prompt = fmt.Sprintf("Compare these two documents:\nDoc A: %s\nDoc B: %s\nFocus: %s\nProvide a detailed comparison in English.", req.FileA, req.FileB, req.Focus)
	} else {
		prompt = fmt.Sprintf("다음 두 문서를 비교해주세요:\n문서A: %s\n문서B: %s\n초점: %s\n한국어로 상세히 비교해주세요.", req.FileA, req.FileB, req.Focus)
	}
	answer, _, err := callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 1500, false)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}
	json200(w, map[string]any{"success": true, "comparison": answer})
}

// ── 설치 후 의존성 상태 체크 (/api/setup/status) ─────────────────────────
func handleSetupStatus(w http.ResponseWriter, r *http.Request) {
	llmMu.RLock()
	apiReady := llmPerplexityKey != "" || llmGroqKey != ""
	tavilyReady := llmTavilyKey != ""
	llmMu.RUnlock()

	// Chrome 존재 여부 (Mac dev: 위치만 확인)
	chromePaths := []string{
		`/Applications/Google Chrome.app/Contents/MacOS/Google Chrome`,
		`/usr/bin/google-chrome`,
		`/usr/bin/chromium-browser`,
	}
	chromeOk := false
	for _, p := range chromePaths {
		if _, err := os.Stat(p); err == nil {
			chromeOk = true
			break
		}
	}

	// Outlook: Mac에서는 항상 false (Windows 전용)
	outlookOk := false

	// yt-dlp 확인
	ytdlpOk := false
	home, _ := os.UserHomeDir()
	ytdlpPaths := []string{
		filepath.Join(home, ".nexus", "yt-dlp"),
		"/usr/local/bin/yt-dlp",
		"/usr/bin/yt-dlp",
	}
	for _, p := range ytdlpPaths {
		if _, err := os.Stat(p); err == nil {
			ytdlpOk = true
			break
		}
	}

	json200(w, map[string]any{
		"platform": "mac-dev",
		"ready":    apiReady,
		"deps": map[string]any{
			"api_keys": map[string]any{
				"ok":      apiReady,
				"tavily":  tavilyReady,
				"message": map[bool]string{true: "API 키 설정됨", false: "번들 기본 키 사용 중"}[apiReady],
			},
			"chrome": map[string]any{
				"ok":      chromeOk,
				"message": map[bool]string{true: "Chrome 설치됨", false: "Chrome 없음 — 웹 자동화 비활성화"}[chromeOk],
			},
			"outlook": map[string]any{
				"ok":      outlookOk,
				"message": "Windows 전용 — Mac에서 비활성화",
			},
			"ytdlp": map[string]any{
				"ok":      ytdlpOk,
				"message": map[bool]string{true: "yt-dlp 설치됨", false: "yt-dlp 없음 — 영상 다운로드 비활성화"}[ytdlpOk],
			},
		},
	})
}
