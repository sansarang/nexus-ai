//go:build !windows

package main

import (
	"fmt"
	"os/exec"
	"strings"
)

// ══════════════════════════════════════════════════════════════
//  클립보드 읽기 + AI 처리
//  Mac: pbpaste
//  "방금 복사한 거 번역해줘", "클립보드 내용 요약해줘" 등
// ══════════════════════════════════════════════════════════════

// readClipboard: 현재 클립보드 텍스트 반환
func readClipboard() string {
	out, err := exec.Command("pbpaste").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// detectClipboardAction: 메시지에서 원하는 처리 유추
func detectClipboardAction(msg string) string {
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "번역") || strings.Contains(lower, "translat"):
		return "translate"
	case strings.Contains(lower, "요약") || strings.Contains(lower, "summarize") || strings.Contains(lower, "summary"):
		return "summarize"
	case strings.Contains(lower, "교정") || strings.Contains(lower, "맞춤법") || strings.Contains(lower, "proofreading") || strings.Contains(lower, "grammar"):
		return "proofread"
	case strings.Contains(lower, "설명") || strings.Contains(lower, "explain") || strings.Contains(lower, "무슨 뜻") || strings.Contains(lower, "what does"):
		return "explain"
	case strings.Contains(lower, "코드") || strings.Contains(lower, "code") || strings.Contains(lower, "분석"):
		return "analyze_code"
	case strings.Contains(lower, "다시 써") || strings.Contains(lower, "rewrite") || strings.Contains(lower, "고쳐") || strings.Contains(lower, "개선"):
		return "rewrite"
	case strings.Contains(lower, "영어") || strings.Contains(lower, "english"):
		return "translate_en"
	case strings.Contains(lower, "한국어") || strings.Contains(lower, "korean"):
		return "translate_ko"
	default:
		return "summarize"
	}
}

// processClipboardContent: 클립보드 내용 + 액션 → LLM 처리
func processClipboardContent(cbText, action, userMsg, apiKey string, eng bool) string {
	// 클립보드 내용 미리보기 (너무 길면 자름)
	preview := cbText
	if len([]rune(preview)) > 2000 {
		preview = string([]rune(preview)[:2000]) + "...(이하 생략)"
	}

	var prompt string
	switch action {
	case "translate", "translate_en":
		if eng {
			prompt = fmt.Sprintf("Translate the following text to English. Output only the translation:\n\n%s", preview)
		} else {
			prompt = fmt.Sprintf("다음 텍스트를 영어로 번역해줘. 번역문만 출력:\n\n%s", preview)
		}
	case "translate_ko":
		if eng {
			prompt = fmt.Sprintf("Translate the following text to Korean. Output only the translation:\n\n%s", preview)
		} else {
			prompt = fmt.Sprintf("다음 텍스트를 한국어로 번역해줘. 번역문만 출력:\n\n%s", preview)
		}
	case "summarize":
		if eng {
			prompt = fmt.Sprintf("Summarize the following text in 3-5 key points:\n\n%s", preview)
		} else {
			prompt = fmt.Sprintf("다음 텍스트를 3~5줄로 핵심만 요약해줘:\n\n%s", preview)
		}
	case "proofread":
		if eng {
			prompt = fmt.Sprintf("Proofread and correct the following text. Show corrections:\n\n%s", preview)
		} else {
			prompt = fmt.Sprintf("다음 텍스트의 맞춤법과 문법을 교정해줘. 수정 사항을 보여줘:\n\n%s", preview)
		}
	case "explain":
		if eng {
			prompt = fmt.Sprintf("Explain what the following text means in simple terms:\n\n%s", preview)
		} else {
			prompt = fmt.Sprintf("다음 텍스트가 무슨 뜻인지 쉽게 설명해줘:\n\n%s", preview)
		}
	case "analyze_code":
		if eng {
			prompt = fmt.Sprintf("Analyze this code. Explain what it does, any issues, and improvements:\n\n%s", preview)
		} else {
			prompt = fmt.Sprintf("다음 코드를 분석해줘. 동작 설명, 문제점, 개선점을 알려줘:\n\n%s", preview)
		}
	case "rewrite":
		if eng {
			prompt = fmt.Sprintf("Rewrite the following text to be clearer and more professional:\n\n%s", preview)
		} else {
			prompt = fmt.Sprintf("다음 텍스트를 더 명확하고 자연스럽게 다시 써줘:\n\n%s", preview)
		}
	default:
		// 사용자 요청 자체를 인스트럭션으로 활용
		if eng {
			prompt = fmt.Sprintf("User request: %s\n\nClipboard content:\n%s", userMsg, preview)
		} else {
			prompt = fmt.Sprintf("사용자 요청: %s\n\n클립보드 내용:\n%s", userMsg, preview)
		}
	}

	result, _, err := callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 1024, false)
	if err != nil {
		if eng {
			return "Failed to process clipboard content: " + err.Error()
		}
		return "클립보드 내용 처리 실패: " + err.Error()
	}
	return result
}
