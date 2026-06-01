// intent_classify_shared.go — Claude Haiku 의도 분류 (Mac/Windows 공통)
// Supabase Edge Function `claude_intent` 액션 활용 → 사용자가 직접 키 설정 불필요
package main

import (
	"encoding/json"
	"fmt"
)

// haikuIntentPrompt: Claude Haiku 의도 분류용 시스템 프롬프트
// Mac 빌드의 callClaudeIntent 와 동일 — 응답 포맷 일치 보장
const haikuIntentPrompt = `You are an intent classifier for a Korean AI assistant. Analyze the user's request and output JSON.

Output format (STRICT — always include all keys):
{"needs_clarify":false,"clarify_question":"","intents":[{"action":"action_name","params":{},"description":"brief"}]}

Available actions:
- "web_search": params: {"query":"검색어","site":"google|auto"} — for news, info, recommendations, directions, ANY informational query
- "weather": params: {"city":"도시명"}
- "pc_status": params: {} — CPU/메모리/디스크 상태
- "scan" or "security_scan": params: {} — 보안 스캔
- "clean": params: {} — 디스크 정리
- "price_compare": params: {"query":"상품명"}
- "video_search" or "youtube_search": params: {"query":"키워드","platform":"youtube|tiktok"}
- "news_search": params: {"query":"키워드"}
- "calendar_today" / "calendar_week" / "calendar_add"
- "email_inbox" / "email_summarize" / "email_send"
- "translate": params: {"text":"원문","target":"ko|en"}
- "volume_control" / "brightness" / "wifi_toggle" / "power_action" / "launch_app"
- "file_search" / "file_organize" / "open_folder"
- "chat": params: {} — 인사, 잡담, 일반 대화에만. 정보 검색은 web_search 우선.

KEY RULES (정확한 매핑 — 절대 chat 으로 떨어뜨리지 말 것):
- 메모리/RAM/CPU/디스크/하드/저장공간/PC 상태/내 컴퓨터/내 PC + (확인|상태|얼마|남았|봐|보여) → pc_status
- 보안/스캔/바이러스/악성/감염/해킹 → security_scan
- 정리/청소/캐시/임시파일/공간 + (해줘|정리|비워) → clean
- "느려"/"버벅"/"렉"/"왜 이래" + PC/컴퓨터 맥락 → full_scan
- 도시명 + (날씨|기온|비|눈|미세먼지) → weather
- 상품명 + (가격|최저가|얼마|비교|쇼핑|쿠팡|네이버) → price_compare
- 유튜브/youtube/틱톡/tiktok/영상/비디오/노래/음악/플레이리스트/MV/뮤직비디오 → video_search (절대 web_search 금지)
- 뉴스/news/오늘 일/최신 소식 → news_search
- 일정/스케줄/캘린더/오늘 약속 → calendar_today
- 이메일/메일/받은편지 → email_inbox
- "알려줘"/"찾아줘"/"보여줘"/"추천해줘" + 일반 정보 → web_search
- 인사 (안녕/hi/hello)/감사 (고마워)/잡담 + 시스템 키워드 0개 → chat

needs_clarify=true ONLY when critical info is TRULY missing (도시 없는 날씨, 지역 없는 맛집 등).
Always output valid JSON, nothing else.`

// callHaikuIntentClassifyViaProxy: Supabase claude_intent 액션 호출
// 반환: 액션명, 파라미터, clarify 여부, clarify 질문, 에러
// JWT 필요 — 없으면 즉시 에러 (Groq fallback으로 진행)
func callHaikuIntentClassifyViaProxy(userMsg string) (action string, params map[string]any, needsClarify bool, clarifyQ string, err error) {
	payload := map[string]any{
		"model":      claudeHaikuModel,
		"max_tokens": 512,
		"system":     haikuIntentPrompt,
		"messages":   []map[string]any{{"role": "user", "content": userMsg}},
	}

	pr, err := callProxy("claude_intent", payload)
	if err != nil {
		return "", nil, false, "", err
	}

	// Claude API 응답 구조: { "content": [{ "type": "text", "text": "..." }] }
	rawJSON, _ := json.Marshal(pr.Result)
	var claudeResp struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(rawJSON, &claudeResp); err != nil || len(claudeResp.Content) == 0 {
		return "", nil, false, "", fmt.Errorf("haiku 응답 파싱 실패")
	}
	text := claudeResp.Content[0].Text

	// 모델이 추가 텍스트를 붙일 수 있으니 JSON 추출 (``` 블록 포함 가능)
	text = extractJSONFromHaiku(text)

	var parsed struct {
		NeedsClarify    bool   `json:"needs_clarify"`
		ClarifyQuestion string `json:"clarify_question"`
		Intents         []struct {
			Action      string         `json:"action"`
			Params      map[string]any `json:"params"`
			Description string         `json:"description"`
		} `json:"intents"`
	}
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		return "", nil, false, "", fmt.Errorf("haiku JSON 파싱 실패: %w", err)
	}

	if parsed.NeedsClarify {
		return "", nil, true, parsed.ClarifyQuestion, nil
	}
	if len(parsed.Intents) == 0 {
		return "", nil, false, "", fmt.Errorf("haiku 응답에 intent 없음")
	}

	first := parsed.Intents[0]
	if first.Params == nil {
		first.Params = map[string]any{}
	}
	return first.Action, first.Params, false, "", nil
}

// extractJSONFromHaiku: 모델이 ```json ... ``` 또는 설명 + JSON 으로 반환할 수 있음
func extractJSONFromHaiku(s string) string {
	// ``` 블록 제거
	if idx := indexOf(s, "```json"); idx >= 0 {
		s = s[idx+len("```json"):]
		if end := indexOf(s, "```"); end >= 0 {
			s = s[:end]
		}
	} else if idx := indexOf(s, "```"); idx >= 0 {
		s = s[idx+3:]
		if end := indexOf(s, "```"); end >= 0 {
			s = s[:end]
		}
	}
	// 첫 { 부터 마지막 } 까지
	start := indexOf(s, "{")
	end := lastIndexOf(s, "}")
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return s
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func lastIndexOf(s, sub string) int {
	for i := len(s) - len(sub); i >= 0; i-- {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
