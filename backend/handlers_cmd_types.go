package main

// Shared types used by both windows and non-windows command handlers.

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
