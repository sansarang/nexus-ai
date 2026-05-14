// types.go — 모든 플랫폼에서 공유하는 타입 및 상수 (빌드 태그 없음)
package main

// ── LLM API 상수 ──────────────────────────────────────────────
const (
	// Perplexity — 메인 LLM (검색 내장)
	pplxChatModel = "sonar-pro" // 검색 내장 고품질 모델
	pplxFastModel = "sonar"     // 빠른 의도 파악용
	pplxAPIBase   = "https://api.perplexity.ai/chat/completions"

	// 하위 호환 별칭 (기존 코드 callGroq 호출 유지)
	groqChatModel = pplxChatModel
	groqFastModel = pplxFastModel

	// Groq — Structured Outputs 전용 (Clarify 판단)
	groqRealAPIBase    = "https://api.groq.com/openai/v1/chat/completions"
	groqStructuredModel = "llama-3.1-8b-instant" // 131k TPM, rate limit 여유

	// Claude (Anthropic) — fallback LLM
	claudeModel   = "claude-sonnet-4-6"
	claudeAPIBase = "https://api.anthropic.com/v1/messages"
)

// groqMsg: OpenAI 호환 메시지 (Perplexity/Groq/OpenAI 공통 포맷)
type groqMsg struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

// msgPart: Vision API 멀티파트 콘텐츠
type msgPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *imageURL `json:"image_url,omitempty"`
}

type imageURL struct {
	URL string `json:"url"`
}

// ConvHistoryMsg: 대화 이력 메시지 (멀티턴 컨텍스트용)
type ConvHistoryMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// DeepSearchResult: 파일 검색 결과
type DeepSearchResult struct {
	Name    string  `json:"name"`
	Path    string  `json:"path"`
	Ext     string  `json:"ext"`
	SizeMB  float64 `json:"size_mb"`
	ModTime string  `json:"mod_time"`
	Snippet string  `json:"snippet"`
	Score   int     `json:"score"`
}
