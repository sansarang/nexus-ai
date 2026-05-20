//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// actionsSkipLLM: LLM 응답 생성을 건너뛸 액션 목록
// (이미 LLM이 생성한 응답이거나, 스트리밍/SSE 기반인 경우)
var actionsSkipLLM = map[string]bool{
	"chat":          true, // 이미 LLM 응답
	"web_search":    true, // 이미 LLM 응답
	"clarify":       true, // 질문이므로 그대로 사용
	"caption_start": true,
	"caption_stop":  true,
	"meeting_start": true,
	"meeting_stop":  true,
	"briefing_now":  true,
	"multi_agent":   true,
	"task_cancel":   true,
}

// generateLLMResponse: 실제 실행 결과를 LLM에 넘겨 자연어 응답 생성
// action: 실행된 액션명
// result: 실행 결과 데이터 (any)
// fallback: dispatchAction이 만든 기존 메시지 (LLM 실패 시 사용)
// original: 사용자 원본 메시지
// lang: "ko" | "en"
// gKey: Groq/Perplexity API 키
func generateLLMResponse(action string, result any, fallback, original, lang, gKey string) string {
	if gKey == "" {
		return fallback
	}
	if actionsSkipLLM[action] {
		return fallback
	}

	// 결과를 JSON으로 직렬화 (너무 길면 잘라냄)
	resultJSON := ""
	if result != nil {
		b, err := json.Marshal(result)
		if err == nil {
			s := string(b)
			if len(s) > 1200 {
				s = s[:1200] + "..."
			}
			resultJSON = s
		}
	}
	if resultJSON == "" || resultJSON == "null" {
		resultJSON = fmt.Sprintf(`{"message":"%s"}`, fallback)
	}

	langInstr := "한국어로 답하세요."
	if lang == "en" {
		langInstr = "Answer in English."
	}

	prompt := fmt.Sprintf(`You are Nexus AI, a helpful PC assistant. The user said: "%s"

The system executed the action "%s" and got this result:
%s

%s
Write a natural, friendly response (2-4 sentences max) that:
- Summarizes what was done or what the result means
- Highlights the most important numbers or findings
- Sounds like a helpful assistant, not a system log
- Does NOT mention action names, JSON, or technical details
- If it's a success action (opening folder, adjusting volume, etc.), just confirm it was done naturally

Response only, no prefixes:`, original, action, resultJSON, langInstr)

	msgs := []groqMsg{{Role: "user", Content: prompt}}
	raw, _, err := callGroqWithFallback(msgs, 200, false)
	if err != nil || strings.TrimSpace(raw) == "" {
		return fallback
	}

	// JSON이 실수로 나오면 fallback
	trimmed := strings.TrimSpace(raw)
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		return fallback
	}

	return trimmed
}
