//go:build !windows

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// Mac 도 동작 (excelize 는 CGO 없이 Pure Go) — 검증용
func cmdExcelAutoCreate(params map[string]any, original, gKey, lang string) (result any, message string) {
	topic, _ := params["topic"].(string)
	if topic == "" {
		topic = extractExcelTopicMac(original)
	}
	if topic == "" {
		topic = "데이터 정리"
	}

	sysPrompt := `당신은 엑셀 샘플 데이터 생성기입니다. 반드시 JSON 만 출력하세요:
{"title":"<시트 이름>","headers":["col1","col2","col3"],"rows":[["v1","v2","v3"]]}
주제에 맞는 5~15행, 3~7컬럼 현실적 데이터. 빈 셀 금지.`
	if lang == "en" {
		sysPrompt = `Generate Excel sample data as STRICT JSON only:
{"title":"<sheet name>","headers":["col1","col2"],"rows":[["v1","v2"]]}
5-15 rows, 3-7 cols, realistic values. No empty cells.`
	}

	raw, _, err := callGroqWithFallback([]groqMsg{
		{Role: "system", Content: sysPrompt},
		{Role: "user", Content: "주제: " + topic},
	}, 1500, true)
	if err != nil || raw == "" {
		return map[string]any{"success": false}, "Excel 데이터 생성 실패: " + fmt.Sprint(err)
	}
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
		return map[string]any{"success": false}, "LLM 응답 파싱 실패"
	}
	if len(parsed.Headers) == 0 || len(parsed.Rows) == 0 {
		return map[string]any{"success": false}, "데이터가 비어있어요."
	}
	if parsed.Title == "" {
		parsed.Title = topic
	}

	// excelize 로 직접 저장 (Mac stub의 saveToExcel은 no-op이라 직접 작성)
	f := excelize.NewFile()
	sheet := parsed.Title
	if len(sheet) > 31 {
		sheet = sheet[:31]
	}
	f.SetSheetName("Sheet1", sheet)
	// 헤더
	for ci, h := range parsed.Headers {
		cell, _ := excelize.CoordinatesToCellName(ci+1, 1)
		f.SetCellValue(sheet, cell, h)
	}
	// 데이터 행
	for ri, row := range parsed.Rows {
		for ci := 0; ci < len(parsed.Headers); ci++ {
			cell, _ := excelize.CoordinatesToCellName(ci+1, ri+2)
			val := ""
			if ci < len(row) {
				val = row[ci]
			}
			f.SetCellValue(sheet, cell, val)
		}
	}

	safeTitle := sanitizeFilenameMac(parsed.Title)
	filename := fmt.Sprintf("%s_%s.xlsx", safeTitle, time.Now().Format("20060102_150405"))
	home, _ := os.UserHomeDir()
	savePath := filepath.Join(home, "Desktop", filename)
	if err := f.SaveAs(savePath); err != nil {
		return map[string]any{"success": false}, "Excel 저장 실패: " + err.Error()
	}

	rows := len(parsed.Rows)
	cols := len(parsed.Headers)
	msg := fmt.Sprintf("✅ Excel 저장 완료 (가상 샘플): %s (%d행 × %d열) — 바탕화면", filename, rows, cols)
	if lang == "en" {
		msg = fmt.Sprintf("✅ Excel saved (sample template): %s (%d rows × %d cols) — Desktop", filename, rows, cols)
	}
	return map[string]any{
		"success":     true,
		"fileName":    filename,
		"path":        savePath,
		"url":         "file:///" + filepath.ToSlash(savePath),
		"mimeType":    "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"rows":        rows,
		"cols":        cols,
		"operation":   "excel_create",
		"title":       parsed.Title,
		"topic":       topic,
		"is_sample":   true,
		"sample_note": sampleNote(lang),
	}, msg
}

func extractExcelTopicMac(s string) string {
	if s == "" {
		return ""
	}
	words := []string{"엑셀", "excel", "스프레드시트", "표", "만들어줘", "만들어", "정리해줘", "정리해", "정리", "생성", "으로", "로", "좀", "해줘", "줘"}
	q := s
	for _, w := range words {
		q = strings.ReplaceAll(q, w, " ")
	}
	q = strings.TrimSpace(q)
	q = regexp.MustCompile(`\s+`).ReplaceAllString(q, " ")
	return q
}

func sanitizeFilenameMac(s string) string {
	forbidden := []string{"/", ":", "*", "?", "\"", "<", ">", "|", "\\"}
	for _, c := range forbidden {
		s = strings.ReplaceAll(s, c, "_")
	}
	s = strings.TrimSpace(s)
	if r := []rune(s); len(r) > 50 {
		s = string(r[:50])
	}
	if s == "" {
		s = "data"
	}
	return s
}
