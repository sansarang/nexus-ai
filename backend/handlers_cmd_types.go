package main

import "strings"

// Shared types used by both windows and non-windows command handlers.

// detectMultiStep — 사용자 입력이 여러 액션을 순차/병렬로 요구하는지 감지
// (Windows + Mac 공용)
// 트리거: (의도 키워드 2개 이상) + (연결사 1개 이상)
// 예) "엑셀로 매출 정리하고 PDF로도 저장해" — excel + pdf + 연결사
// 예) "오늘 일정 확인하고 빈 시간에 회의 잡아" — 일정 + 미팅 + 연결사
// 예) "메일 확인하고 중요한 거 요약해서 노트에 저장" — 메일 + 요약 + 저장 + 연결사
// 단일 의도 + multi_action(검색+저장)은 false 반환 (기존 multi_action 핸들러가 처리)
func detectMultiStep(msg string) bool {
	lower := strings.ToLower(msg)
	// 1) 연결사 — 한/영
	connectors := []string{
		"하고 ", "한 다음", "그리고", "그다음", "그 다음", "한뒤", "한 뒤", "이후에", "다음에",
		"해서 ", "하면서", "한후", "한 후", "고 ", "찾고", "받고", "보고", "쓰고", "만들고",
		" and then ", " then ", " also ", " and ", " after ", " next ", " plus ",
	}
	hasConnector := false
	for _, c := range connectors {
		if strings.Contains(lower, c) {
			hasConnector = true
			break
		}
	}
	if !hasConnector {
		return false
	}

	// 2) 의도 키워드 그룹 — 2개 이상 다른 그룹이 매칭되면 멀티스텝
	intentGroups := [][]string{
		{"엑셀", "excel", "xlsx", "스프레드시트"},
		{"pdf", "피디에프"},
		{"문서", "보고서", "메모", "회의록", "노트", "report", "document", "memo"},
		{"검색", "찾", "search", "find", "lookup"},
		{"메일", "이메일", "email", "mail", "inbox", "받은편지"},
		{"일정", "캘린더", "스케줄", "schedule", "calendar"},
		{"회의", "미팅", "meeting"},
		{"요약", "summarize", "summary"},
		{"분석", "analyze", "analysis"},
		{"저장", "save", "기록"},
		{"정리", "organize", "sort"},
		{"가격", "최저가", "price"},
		{"유튜브", "youtube", "영상", "video"},
		{"뉴스", "news"},
		{"날씨", "weather"},
		{"진단", "스캔", "scan", "diagnose"},
		{"청소", "clean"},
		{"번역", "translate"},
		{"알림", "리마인더", "remind", "notify"},
		{"실행", "열어", "launch", "open"},
	}
	matchedGroups := 0
	for _, group := range intentGroups {
		for _, kw := range group {
			if strings.Contains(lower, kw) {
				matchedGroups++
				break
			}
		}
		if matchedGroups >= 2 {
			return true
		}
	}
	return false
}

type CommandRequest struct {
	Message         string           `json:"message"`
	Context         string           `json:"context"`
	Lang            string           `json:"lang"`
	PendingIntent   string           `json:"pending_intent"`
	PendingParams   map[string]any   `json:"pending_params"`
	PendingQuestion string           `json:"pending_question"`
	History         []ConvHistoryMsg `json:"history"`
	UserEmail       string           `json:"user_email"`
}

type CommandResponse struct {
	Success          bool           `json:"success"`
	Message          string         `json:"message"`
	Action           string         `json:"action"`
	Result           any            `json:"result"`
	Duration         string         `json:"duration"`
	// CardType — 프론트 cardRegistry 에 어떤 카드를 그릴지 명시 (선택)
	//   비어있으면 ACTION_TO_CARD 매핑표 또는 result 구조로 자동 추론
	//   백엔드 핸들러가 명확히 지정하고 싶을 때 (예: 같은 action 이 다른 카드를 그릴 때)
	CardType         string         `json:"card_type,omitempty"`
	NeedsClarify     bool           `json:"needs_clarify,omitempty"`
	ClarifyQuestion  string         `json:"clarify_question,omitempty"`
	ClarifyQuestions []string       `json:"clarify_questions,omitempty"`
	PendingIntent    string         `json:"pending_intent,omitempty"`
	PendingParams    map[string]any `json:"pending_params,omitempty"`
	UpgradeRequired  bool           `json:"upgrade_required,omitempty"`
	UsedCount        int            `json:"used_count,omitempty"`
	LimitCount       int            `json:"limit_count,omitempty"`
	FeatureName      string         `json:"feature_name,omitempty"`
}
