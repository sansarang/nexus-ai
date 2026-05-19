//go:build windows

package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// readClipboard: Windows 클립보드 읽기 (PowerShell Get-Clipboard)
func readClipboard() string {
	out, err := safePS(5*time.Second, "Get-Clipboard")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func detectClipboardAction(msg string) string {
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "번역") || strings.Contains(lower, "translat"):
		return "translate"
	case strings.Contains(lower, "요약") || strings.Contains(lower, "summarize"):
		return "summarize"
	case strings.Contains(lower, "교정") || strings.Contains(lower, "맞춤법") || strings.Contains(lower, "grammar"):
		return "proofread"
	case strings.Contains(lower, "설명") || strings.Contains(lower, "explain") || strings.Contains(lower, "무슨 뜻"):
		return "explain"
	case strings.Contains(lower, "코드") || strings.Contains(lower, "code") || strings.Contains(lower, "분석"):
		return "analyze_code"
	case strings.Contains(lower, "다시 써") || strings.Contains(lower, "rewrite") || strings.Contains(lower, "고쳐"):
		return "rewrite"
	case strings.Contains(lower, "영어"):
		return "translate_en"
	case strings.Contains(lower, "한국어"):
		return "translate_ko"
	default:
		return "summarize"
	}
}

func processClipboardContent(cbText, action, userMsg, apiKey string, eng bool) string {
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
		prompt = fmt.Sprintf("다음 텍스트를 한국어로 번역해줘. 번역문만 출력:\n\n%s", preview)
	case "summarize":
		if eng {
			prompt = fmt.Sprintf("Summarize the following text in 3-5 key points:\n\n%s", preview)
		} else {
			prompt = fmt.Sprintf("다음 텍스트를 3~5줄로 핵심만 요약해줘:\n\n%s", preview)
		}
	case "proofread":
		if eng {
			prompt = fmt.Sprintf("Proofread and correct the following text:\n\n%s", preview)
		} else {
			prompt = fmt.Sprintf("다음 텍스트의 맞춤법과 문법을 교정해줘:\n\n%s", preview)
		}
	case "explain":
		if eng {
			prompt = fmt.Sprintf("Explain what the following text means:\n\n%s", preview)
		} else {
			prompt = fmt.Sprintf("다음 내용이 무슨 뜻인지 설명해줘:\n\n%s", preview)
		}
	case "analyze_code":
		if eng {
			prompt = fmt.Sprintf("Analyze this code — explain, find issues, suggest improvements:\n\n%s", preview)
		} else {
			prompt = fmt.Sprintf("다음 코드를 분석해줘. 동작, 문제점, 개선점:\n\n%s", preview)
		}
	case "rewrite":
		if eng {
			prompt = fmt.Sprintf("Rewrite the following to be clearer and more professional:\n\n%s", preview)
		} else {
			prompt = fmt.Sprintf("다음을 더 명확하고 자연스럽게 다시 써줘:\n\n%s", preview)
		}
	default:
		if eng {
			prompt = fmt.Sprintf("User request: %s\n\nClipboard content:\n%s", userMsg, preview)
		} else {
			prompt = fmt.Sprintf("사용자 요청: %s\n\n클립보드 내용:\n%s", userMsg, preview)
		}
	}
	result, _, err := callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 1024, false)
	if err != nil {
		if eng {
			return "Failed to process: " + err.Error()
		}
		return "처리 실패: " + err.Error()
	}
	return result
}

// Windows에서 exec 패키지가 필요한 경우
var _ = exec.Command
