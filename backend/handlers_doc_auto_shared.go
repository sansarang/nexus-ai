// handlers_doc_auto_shared.go — 문서 자동 생성 (Mac/Windows 공용)
// 사장님 원칙: 데이터 없으면 LLM이 만들고 → 실제 파일 저장 → 경로 출력
//
// 지원 포맷:
//   .txt — 일반 텍스트
//   .md  — 마크다운 (PDF 변환은 Python sidecar 또는 외부 도구 활용)
//   .html — 간단한 HTML (브라우저로 PDF 인쇄 가능)
//
// 라우팅 시나리오:
//   "보고서 작성해줘"      → format=md, topic="보고서"
//   "메모 저장"            → format=txt, topic="메모"
//   "회의록 만들어줘"      → format=md, topic="회의록"
//   "분기별 매출 보고서"   → format=md, topic="분기별 매출 보고서"
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// cmdDocAutoCreate: LLM 이 문서 본문 생성 → 파일 저장 → 경로 응답
// params: { topic?, format? (txt|md|html), tone? (formal|casual) }
func cmdDocAutoCreate(params map[string]any, original, gKey, lang string) (result any, message string) {
	topic, _ := params["topic"].(string)
	format, _ := params["format"].(string)
	tone, _ := params["tone"].(string)
	if topic == "" {
		topic = extractDocTopic(original)
	}
	if topic == "" {
		topic = "문서"
	}
	if format == "" {
		format = detectDocFormat(original) // 메시지에서 자동 감지
	}
	if format == "" {
		format = "md" // 기본 마크다운
	}
	if tone == "" {
		tone = "formal"
	}

	// LLM 프롬프트 — 포맷별 가이드
	formatHint := "마크다운 형식 (# 제목, ## 섹션, 본문). 5~10섹션."
	if format == "txt" {
		formatHint = "일반 텍스트 (마크다운 X). 단락으로 구성. 5~10단락."
	} else if format == "html" {
		formatHint = "간단한 HTML (<h1>, <h2>, <p>, <ul>). CSS 없음. 5~10섹션."
	}

	sysPrompt := ""
	if lang == "en" {
		sysPrompt = fmt.Sprintf(`You are a document writer. Create a complete document on the given topic.

Format: %s
Tone: %s
Output: ONLY the document content, no preamble, no "Here is your document"
Length: comprehensive — 300~800 words

Make it useful and practical. Include realistic examples/data.
IMPORTANT: This is sample template data. Real values should be filled in by user.`, formatHint, tone)
	} else {
		sysPrompt = fmt.Sprintf(`당신은 문서 작성 비서입니다. 주제에 맞는 완성된 문서를 작성하세요.

포맷: %s
어조: %s ("formal"=정중, "casual"=친근)
출력: 문서 본문만. "다음과 같이 작성했습니다" 같은 서론 금지.
길이: 충실하게 — 300~800단어

실용적으로. 현실적 예시/숫자 포함.
중요: 이건 샘플 템플릿 데이터입니다. 실제 값은 사용자가 채워야 함.`, formatHint, tone)
	}

	// 성능 최적화: 2000 → 1000 (응답 ~8초 단축)
	content, _, err := callGroqWithFallback([]groqMsg{
		{Role: "system", Content: sysPrompt},
		{Role: "user", Content: "주제: " + topic},
	}, 1000, false)
	if err != nil || content == "" {
		return map[string]any{"success": false, "error": "LLM 호출 실패"},
			fmt.Sprintf("문서 생성 실패: %v", err)
	}

	// 파일 저장
	safeTitle := sanitizeDocAutoFilename(topic)
	filename := fmt.Sprintf("%s_%s.%s", safeTitle, time.Now().Format("20060102_150405"), format)
	home, _ := os.UserHomeDir()
	savePath := filepath.Join(home, "Desktop", filename)

	// HTML 은 head/body 래핑
	finalContent := content
	if format == "html" {
		finalContent = fmt.Sprintf(`<!DOCTYPE html>
<html lang="%s"><head>
<meta charset="UTF-8">
<title>%s</title>
<style>
  body { font-family: -apple-system, "Segoe UI", sans-serif; max-width: 800px; margin: 40px auto; padding: 20px; line-height: 1.7; color: #1f2937; }
  h1 { color: #0f172a; border-bottom: 2px solid #e5e7eb; padding-bottom: 8px; }
  h2 { color: #1e40af; margin-top: 28px; }
  p { margin: 12px 0; }
  ul, ol { padding-left: 24px; }
  .meta { color: #9ca3af; font-size: 12px; margin-bottom: 24px; }
</style>
</head><body>
<div class="meta">자동 생성 · %s · Nexus AI</div>
%s
</body></html>`, lang, topic, time.Now().Format("2006-01-02 15:04"), content)
	}

	if err := os.WriteFile(savePath, []byte(finalContent), 0644); err != nil {
		return map[string]any{"success": false, "error": err.Error()},
			fmt.Sprintf("파일 저장 실패: %v", err)
	}

	// 응답
	lines := strings.Count(content, "\n") + 1
	chars := len([]rune(content))
	msg := fmt.Sprintf("✅ %s 저장 완료 (가상 샘플 — %d줄, %d자) — 바탕화면\n📄 %s", strings.ToUpper(format), lines, chars, filename)
	if lang == "en" {
		msg = fmt.Sprintf("✅ %s saved (sample template — %d lines, %d chars) — Desktop\n📄 %s", strings.ToUpper(format), lines, chars, filename)
	}

	return map[string]any{
		"success":      true,
		"fileName":     filename,
		"path":         savePath,
		"url":          "file:///" + filepath.ToSlash(savePath),
		"mimeType":     mimeTypeFor(format),
		"format":       format,
		"lines":        lines,
		"chars":        chars,
		"operation":    "doc_create",
		"topic":        topic,
		"is_sample":    true, // 가상 샘플 플래그 (사장님 요구 #2)
		"sample_note":  sampleNote(lang),
	}, msg
}

// detectDocFormat: 메시지에서 포맷 자동 감지
func detectDocFormat(msg string) string {
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "html") || strings.Contains(lower, "웹페이지"):
		return "html"
	case strings.Contains(lower, "txt") || strings.Contains(lower, "텍스트"):
		return "txt"
	case strings.Contains(lower, " md") || strings.Contains(lower, "마크다운") || strings.Contains(lower, "markdown"):
		return "md"
	case strings.Contains(lower, "보고서") || strings.Contains(lower, "report") ||
		strings.Contains(lower, "회의록") || strings.Contains(lower, "meeting") ||
		strings.Contains(lower, "기획서") || strings.Contains(lower, "proposal"):
		return "md"
	case strings.Contains(lower, "메모") || strings.Contains(lower, "memo") ||
		strings.Contains(lower, "노트") || strings.Contains(lower, "note"):
		return "txt"
	}
	return "md"
}

// extractDocTopic: 메시지에서 문서 주제 추출
func extractDocTopic(s string) string {
	if s == "" {
		return ""
	}
	words := []string{
		"문서", "보고서", "report", "메모", "memo", "노트", "note", "회의록",
		"기획서", "proposal", "html", "텍스트", "마크다운", "markdown",
		"작성해줘", "작성해", "작성", "만들어줘", "만들어", "생성해", "생성", "써줘", "써",
		"으로", "로", "좀", "해줘", "줘", "저장",
	}
	q := s
	for _, w := range words {
		q = strings.ReplaceAll(q, w, " ")
	}
	q = strings.TrimSpace(q)
	// 연속 공백 정리
	for strings.Contains(q, "  ") {
		q = strings.ReplaceAll(q, "  ", " ")
	}
	return q
}

// sanitizeDocAutoFilename: 파일명 안전화
func sanitizeDocAutoFilename(s string) string {
	forbidden := []string{"/", ":", "*", "?", "\"", "<", ">", "|", "\\"}
	for _, c := range forbidden {
		s = strings.ReplaceAll(s, c, "_")
	}
	s = strings.TrimSpace(s)
	if r := []rune(s); len(r) > 50 {
		s = string(r[:50])
	}
	if s == "" {
		s = "document"
	}
	return s
}

func mimeTypeFor(format string) string {
	switch format {
	case "txt":
		return "text/plain"
	case "md":
		return "text/markdown"
	case "html":
		return "text/html"
	}
	return "text/plain"
}

func sampleNote(lang string) string {
	if lang == "en" {
		return "⚠️ This is sample template data generated by AI. Replace values with your real data."
	}
	return "⚠️ 이건 AI가 생성한 가상 샘플입니다. 실제 데이터로 교체해서 사용하세요."
}
