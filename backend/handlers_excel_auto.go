//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// ══════════════════════════════════════════════════════════════════
//  Excel 자동 생성 (사장님 원칙: 데이터 없으면 LLM이 만들어서 저장)
//  Jarvis 흐름:
//    사용자: "엑셀 매출 정리해" 또는 "고객 명단 만들어줘"
//    백엔드: LLM 에게 헤더+데이터 JSON 생성 요청 → saveToExcel → 경로 응답
//    프론트: file_result 카드 자동 렌더 (파일 경로 + 📂 열기 버튼)
// ══════════════════════════════════════════════════════════════════

// cmdExcelAutoCreate: dispatchAction 에서 호출됨
// params: { topic?: string }   (없으면 사용자 메시지에서 자동 추출)
func cmdExcelAutoCreate(params map[string]any, original, gKey, lang string) (result any, message string) {
	// 주제 추출: params.topic > 메시지에서 키워드 제거
	topic, _ := params["topic"].(string)
	if topic == "" {
		topic = extractExcelTopic(original)
	}
	if topic == "" {
		topic = "데이터 정리"
	}

	// LLM 프롬프트: JSON 만 반환하도록 엄격하게
	sysPrompt := ""
	if lang == "en" {
		sysPrompt = `You generate sample data for Excel files. Output STRICT JSON only — no markdown, no explanation:
{
  "title": "<sheet name, 1-3 words>",
  "headers": ["col1", "col2", "col3"],
  "rows": [
    ["v1", "v2", "v3"],
    ...
  ]
}
Rules:
- 5-15 rows of realistic sample data
- 3-7 columns matching topic
- Realistic Korean/English values (names, dates, numbers)
- No empty cells`
	} else {
		sysPrompt = `당신은 엑셀 샘플 데이터 생성기입니다. 반드시 JSON 만 출력하세요. 마크다운/설명 금지:
{
  "title": "<시트 이름, 1-3단어>",
  "headers": ["컬럼1", "컬럼2", "컬럼3"],
  "rows": [
    ["값1", "값2", "값3"],
    ...
  ]
}
규칙:
- 주제에 맞는 현실적 샘플 데이터 5~15행
- 3~7 컬럼
- 실제 같은 값 (이름/날짜/금액 등)
- 빈 셀 금지`
	}

	userPrompt := fmt.Sprintf("주제: %s\n위 주제에 맞는 Excel 샘플 데이터를 JSON 으로 만들어줘.", topic)
	if lang == "en" {
		userPrompt = fmt.Sprintf("Topic: %s\nGenerate Excel sample data as JSON.", topic)
	}

	raw, _, err := callGroqWithFallback([]groqMsg{
		{Role: "system", Content: sysPrompt},
		{Role: "user", Content: userPrompt},
	}, 1500, true)
	if err != nil || raw == "" {
		return map[string]any{"success": false, "error": "LLM 호출 실패"},
			fmt.Sprintf("Excel 데이터 생성 실패: %v", err)
	}

	// JSON 파싱 — 코드 블록/설명 제거
	raw = strings.TrimSpace(raw)
	if start := strings.Index(raw, "{"); start > 0 {
		raw = raw[start:]
	}
	if end := strings.LastIndex(raw, "}"); end > 0 {
		raw = raw[:end+1]
	}

	var parsed struct {
		Title   string     `json:"title"`
		Headers []string   `json:"headers"`
		Rows    [][]string `json:"rows"`
	}
	if jsonErr := json.Unmarshal([]byte(raw), &parsed); jsonErr != nil {
		return map[string]any{"success": false, "error": "LLM JSON 파싱 실패"},
			fmt.Sprintf("LLM 응답 파싱 실패. 다시 시도해주세요.")
	}
	if len(parsed.Headers) == 0 || len(parsed.Rows) == 0 {
		return map[string]any{"success": false, "error": "empty data"},
			"생성된 데이터가 비어있어요. 주제를 좀 더 구체적으로 알려주세요."
	}
	if parsed.Title == "" {
		parsed.Title = topic
	}

	// 데이터 정리: 헤더 + 행 = saveToExcel 입력 형식 [][]string
	data := [][]string{parsed.Headers}
	for _, row := range parsed.Rows {
		// 헤더 길이에 맞춰 패딩
		if len(row) < len(parsed.Headers) {
			row = append(row, make([]string, len(parsed.Headers)-len(row))...)
		}
		data = append(data, row[:len(parsed.Headers)])
	}

	// 파일명: 주제 + 타임스탬프
	safeTitle := sanitizeExcelFilename(parsed.Title)
	filename := fmt.Sprintf("%s_%s.xlsx", safeTitle, time.Now().Format("20060102_150405"))
	home, _ := os.UserHomeDir()
	savePath := filepath.Join(home, "Desktop", filename)

	// 저장
	if err := saveToExcel(data, savePath, parsed.Title); err != nil {
		return map[string]any{"success": false, "error": err.Error()},
			fmt.Sprintf("Excel 저장 실패: %v", err)
	}

	// 응답 — file_result 카드 자동 렌더
	rows := len(parsed.Rows)
	cols := len(parsed.Headers)
	msg := fmt.Sprintf("✅ Excel 저장 완료: %s (%d행 × %d열) — 바탕화면", filename, rows, cols)
	if lang == "en" {
		msg = fmt.Sprintf("✅ Excel saved: %s (%d rows × %d cols) — Desktop", filename, rows, cols)
	}

	return map[string]any{
		"success":   true,
		"fileName":  filename,
		"path":      savePath,
		"url":       "file:///" + filepath.ToSlash(savePath),
		"mimeType":  "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"rows":      rows,
		"cols":      cols,
		"operation": "excel_create",
		"title":     parsed.Title,
		"topic":     topic,
	}, msg
}

// extractExcelTopic: 사용자 메시지에서 Excel 주제 추출
// "엑셀 매출 정리해" → "매출 정리"
// "고객 명단 엑셀로 만들어줘" → "고객 명단"
func extractExcelTopic(s string) string {
	if s == "" {
		return ""
	}
	// 명령어 단어 제거
	excelWords := []string{
		"엑셀", "excel", "스프레드시트", "spreadsheet", "표", "table",
		"만들어줘", "만들어", "정리해줘", "정리해", "정리", "생성해줘", "생성해", "생성",
		"으로", "로", "좀", "해줘", "줘",
	}
	q := s
	for _, w := range excelWords {
		q = strings.ReplaceAll(q, w, " ")
	}
	q = strings.TrimSpace(q)
	// 연속 공백 정리
	multiSpaceRe := regexp.MustCompile(`\s+`)
	q = multiSpaceRe.ReplaceAllString(q, " ")
	return q
}

// sanitizeExcelFilename: Windows 파일명 금지 문자 제거 (Excel 자동 생성 전용)
func sanitizeExcelFilename(s string) string {
	forbidden := []string{"\\", "/", ":", "*", "?", "\"", "<", ">", "|"}
	for _, c := range forbidden {
		s = strings.ReplaceAll(s, c, "_")
	}
	s = strings.TrimSpace(s)
	if len(s) > 50 {
		runes := []rune(s)
		if len(runes) > 50 {
			s = string(runes[:50])
		}
	}
	if s == "" {
		s = "data"
	}
	return s
}
