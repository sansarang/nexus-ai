// intent_classify_shared.go — Claude Haiku 의도 분류 (Mac/Windows 공통)
// Supabase Edge Function `claude_intent` 액션 활용 → 사용자가 직접 키 설정 불필요
package main

import (
	"encoding/json"
	"fmt"
)

// haikuIntentPromptDefault: 기본 의도 분류 프롬프트 (자가치유 전 원본)
// 힐링된 버전은 ~/.nexus/prompts/intent.txt 에서 로드됨
const haikuIntentPromptDefault = `You are an intent classifier for a Korean AI assistant. Analyze the user's request and output JSON.

Output format (STRICT — always include all keys):
{"needs_clarify":false,"clarify_question":"","intents":[{"action":"action_name","params":{},"description":"brief"}]}

Available actions (60+):

[시스템/PC]
- "pc_status": params:{} — CPU/메모리/디스크/GPU 상태
- "security_scan": params:{} — 보안·바이러스 스캔
- "full_scan": params:{} — 전체 PC 진단 (느림/버벅)
- "clean": params:{} — 디스크·캐시 정리
- "repair": params:{} — PC 수리·복구
- "drivers": params:{} — 드라이버 목록·업데이트
- "programs_list": params:{} — 설치 프로그램 목록
- "process_top": params:{} — 실행 중 프로세스 목록
- "network_analysis": params:{} — 네트워크 연결 분석
- "boot_analysis": params:{} — 부팅 속도·시작프로그램
- "gpu_stats": params:{} — GPU 사용률·온도
- "power_plan": params:{"plan":"balanced|performance|power_saver"}
- "volume_control": params:{"action":"up|down|mute","level":50}
- "brightness": params:{"level":50}
- "wifi_toggle": params:{"action":"on|off|status"}
- "launch_app": params:{"app":"앱이름"}
- "process_kill": params:{"name":"프로세스명"}
- "power_action": params:{"action":"shutdown|restart|sleep|lock"}
- "system_updates": params:{} — Windows 업데이트 확인
- "clipboard_history": params:{} — 클립보드 기록

[검색/정보]
- "web_search": params:{"query":"검색어","site":"google|naver|auto"}
- "news_search": params:{"query":"키워드","lang":"ko|en"}
- "deep_search": params:{"query":"키워드"} — 심층 멀티소스 검색
- "reddit_search": params:{"query":"키워드","subreddit":""}
- "weather": params:{"city":"도시명","days":1}
- "price_compare": params:{"query":"상품명","site":"coupang|naver|auto"}
- "stock_price": params:{"ticker":"종목코드|종목명","exchange":"KRX|NASDAQ|NYSE"}
- "map_search": params:{"query":"장소명","location":"현재위치"}
- "directions": params:{"from":"출발지","to":"도착지","mode":"transit|drive"}
- "translate": params:{"text":"원문","target":"ko|en|ja|zh|es|fr"}
- "business_lookup": params:{"brno":"사업자등록번호10자리"} — 국세청 사업자 상태 조회
- "wayback": params:{"url":"URL"} — 웹페이지 아카이브 조회

[영상/미디어]
- "video_search": params:{"query":"키워드","platform":"youtube|tiktok|all"}
- "video_transcript": params:{"url":"유튜브URL"} — 자막·요약 추출
- "ytmusic_search": params:{"query":"노래|아티스트"}
- "tiktok_trending": params:{} — 틱톡 트렌딩
- "netflix_trending": params:{} — 넷플릭스 인기작

[파일/문서]
- "file_search": params:{"query":"파일명","path":"폴더경로"}
- "file_organize": params:{"path":"폴더경로"} — 파일 자동 정리
- "file_duplicates": params:{"path":"폴더경로"} — 중복 파일 탐색
- "file_large": params:{"path":"폴더경로","min_mb":100}
- "open_folder": params:{"path":"폴더경로"}
- "doc_summary": params:{"path":"파일경로"} — 문서 AI 요약
- "doc_compare": params:{"path1":"","path2":""} — 문서 비교
- "excel_analyze": params:{"path":"엑셀파일경로"}
- "excel_auto_create": params:{"topic":"주제","rows":10}
- "pdf_auto_create": params:{"topic":"주제","pages":1}
- "vision_screen": params:{} — 현재 화면 AI 분석
- "vision_ocr": params:{} — 화면/이미지 텍스트 추출

[이메일/캘린더]
- "email_inbox": params:{"count":10} — 받은 메일 목록
- "email_summarize": params:{"count":20} — 메일 요약
- "email_send": params:{"to":"수신자","subject":"제목","body":"내용"}
- "email_draft_reply": params:{"subject":"","body":""} — 답장 초안 생성
- "calendar_today": params:{} — 오늘 일정
- "calendar_week": params:{} — 이번 주 일정
- "calendar_add": params:{"title":"제목","datetime":"YYYY-MM-DDTHH:mm","duration":60}
- "calendar_find_slot": params:{"duration":60,"days":7}

[메모/생산성]
- "notes": params:{"action":"list|create|search","content":"","query":""}
- "memory_search": params:{"query":"키워드"} — 대화 기억 검색
- "journal_today": params:{} — 오늘 일지
- "workflow_run": params:{"id":"워크플로우ID"}
- "macro_run": params:{"name":"매크로명"}
- "briefing_now": params:{} — 지금 즉시 브리핑
- "meeting_summarize": params:{"text":"회의 내용"}
- "recall_search": params:{"query":"키워드"} — 과거 화면 기억 검색

[잡담/기타]
- "chat": params:{} — 인사·잡담·일반 대화 (시스템 키워드 없을 때만)
- "clarify": params:{"question":"무엇이 필요한지"} — 정보 부족

KEY RULES (절대 chat으로 떨어뜨리지 말 것):
- 메모리/RAM/CPU/디스크/하드/저장공간/PC상태/내컴퓨터 → pc_status
- 보안/스캔/바이러스/악성/감염/해킹 → security_scan
- 정리/청소/캐시/임시파일/공간 → clean
- 느려/버벅/렉 + PC맥락 → full_scan
- 도시명 + 날씨/기온/비/눈/미세먼지 → weather
- 상품명 + 가격/최저가/비교/쿠팡/네이버 → price_compare
- 유튜브/틱톡/영상/비디오/노래/음악/MV → video_search
- 뉴스/오늘일/최신소식 → news_search
- 일정/스케줄/캘린더/약속 → calendar_today
- 이메일/메일/받은편지 → email_inbox
- 주가/주식/코스피/나스닥 + 종목명 → stock_price
- 사업자번호/사업자등록번호 + 조회/확인 → business_lookup
- 번역/~로 번역/translate → translate
- 자막/스크립트 + 유튜브URL → video_transcript
- 알려줘/찾아줘/보여줘/추천해줘 + 일반정보 → web_search
- 인사(안녕/hi)/감사(고마워)/잡담 + 시스템키워드 0개 → chat

MULTI-ACTION: "A하고 B도 해줘" → intents 배열에 2개 이상 넣기
needs_clarify=true ONLY when TRULY critical info missing (도시없는 날씨, 지역없는 맛집 등).
Always output valid JSON, nothing else.`

// callHaikuIntentClassifyViaProxy: Supabase claude_intent 액션 호출
// 반환: 액션명, 파라미터, clarify 여부, clarify 질문, 에러
// JWT 필요 — 없으면 즉시 에러 (Groq fallback으로 진행)
func callHaikuIntentClassifyViaProxy(userMsg string) (action string, params map[string]any, needsClarify bool, clarifyQ string, err error) {
	payload := map[string]any{
		"model":      claudeHaikuModel,
		"max_tokens": 512,
		"system":     getIntentPrompt(),
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

// getIntentPrompt: 힐링된 버전 우선, 없으면 기본값
// handlers_prompt_heal.go 의 loadHealedPrompt() 사용
func getIntentPrompt() string {
	return loadHealedPrompt("intent", haikuIntentPromptDefault)
}
