// llm_cleanup_shared.go — LLM 응답 자비스 톤 후처리 (Mac/Windows 공통)
package main

import (
	"regexp"
	"strings"
)

// 정규식 사전 컴파일 (성능 + 재사용)
var (
	jarvisHeaderRe  = regexp.MustCompile(`(?m)^#{1,6}\s+`)
	jarvisBoldRe    = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	jarvisItalicRe  = regexp.MustCompile(`(?:^|[\s])\*([^*\n]{2,40})\*(?:[\s.,!?]|$)`)
	jarvisBulletRe  = regexp.MustCompile(`(?m)^\s*[•\-]\s+`)
	jarvisMultiNlRe = regexp.MustCompile(`\n{3,}`)
)

// cleanJarvisTone: LLM 응답을 자비스 톤으로 후처리
//   - 마크다운 헤더 (#, ##, ###) 줄 → 일반 텍스트
//   - **굵게** 별표 제거 (텍스트만)
//   - *italic* 별표 제거
//   - 시작 불릿 (•, -) → 인라인
//   - "정리하면", "요약하면" 비즈니스 preamble 제거 (한/영)
//   - 연속 빈 줄 3+ → 2
//
// 멱등 (idempotent): 두 번 호출해도 결과 동일.
func cleanJarvisTone(s string) string {
	if s == "" {
		return s
	}
	s = jarvisHeaderRe.ReplaceAllString(s, "")
	s = jarvisBoldRe.ReplaceAllString(s, "$1")
	s = jarvisItalicRe.ReplaceAllStringFunc(s, func(m string) string {
		return strings.ReplaceAll(m, "*", "")
	})
	s = jarvisBulletRe.ReplaceAllString(s, "")
	// preamble 제거 — 응답 시작 부분의 비즈니스/회피 문구
	preambles := []string{
		// KO
		"정리하면, ", "정리하면 ", "요약하면, ", "요약하면 ",
		"다음과 같습니다.\n", "다음과 같이 정리해드릴게요.\n",
		"답변드리자면, ", "말씀드리자면, ",
		"질문해주신 ", "문의하신 ",
		// EN
		"In summary, ", "To summarize, ", "Here's a summary: ",
		"To answer your question, ", "In short, ",
		"Sure! ", "Of course! ", "Certainly, ",
		"As an AI, ", "I'm just an AI ",
	}
	for _, p := range preambles {
		s = strings.TrimPrefix(s, p)
	}
	s = strings.ReplaceAll(s, "다음과 같이 정리해드릴게요.\n", "")
	s = strings.ReplaceAll(s, "여러 옵션이 있습니다.", "")
	s = jarvisMultiNlRe.ReplaceAllString(s, "\n\n")
	s = strings.TrimSpace(s)

	// 안전망: 너무 긴 답변 자동 자르기 (시스템 프롬프트 무시한 경우)
	// 줄 수 제한: 5줄, 글자 수 제한: 500자
	const maxLines = 5
	const maxChars = 500
	lines := strings.Split(s, "\n")
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		s = strings.Join(lines, "\n")
	}
	// 글자 수 초과 시 마지막 문장 종료점에서 자르기 (한글 + 영어)
	if runeCount := len([]rune(s)); runeCount > maxChars {
		runes := []rune(s)[:maxChars]
		s = string(runes)
		// 끝에서 가장 가까운 문장 부호 (. ! ? 。 ! ?)를 찾아 자르기
		lastSentenceEnd := -1
		for i := len(s) - 1; i >= maxChars/2 && i >= 0; i-- {
			r := rune(s[i])
			if r == '.' || r == '!' || r == '?' || r == '。' {
				lastSentenceEnd = i
				break
			}
		}
		if lastSentenceEnd > 0 {
			s = s[:lastSentenceEnd+1]
		} else {
			s = s + "…"
		}
	}
	return s
}
