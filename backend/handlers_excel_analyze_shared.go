// handlers_excel_analyze_shared.go — 기존 Excel 파일 분석/요약 (Mac/Windows 공용)
// 사장님 원칙 #3: 사용자가 가진 파일 활용
//
// 라우팅 시나리오:
//   "내 매출 엑셀 분석해줘" → 바탕화면 가장 최근 xlsx 자동 선택
//   "C:\path\file.xlsx 요약" → 명시 경로
//   "오늘 만든 엑셀 보여줘" → 시간 기준 검색
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// cmdExcelAnalyze: 기존 Excel 파일 읽기 + LLM 분석 요약
// params: { path?: string }   (없으면 바탕화면에서 최근 .xlsx 자동)
func cmdExcelAnalyze(params map[string]any, original, gKey, lang string) (result any, message string) {
	path, _ := params["path"].(string)
	if path == "" {
		// 메시지에서 경로 추출 시도
		path = extractFilePath(original)
	}
	if path == "" {
		// 바탕화면에서 가장 최근 .xlsx
		path = findLatestExcel()
	}
	if path == "" {
		errMsg := "분석할 Excel 파일을 찾을 수 없어요. 파일 경로를 알려주거나 바탕화면에 .xlsx 파일을 두세요."
		if lang == "en" {
			errMsg = "Could not find an Excel file. Specify a path or place .xlsx on Desktop."
		}
		return map[string]any{"success": false}, errMsg
	}

	// 파일 존재 확인
	info, err := os.Stat(path)
	if err != nil {
		return map[string]any{"success": false},
			fmt.Sprintf("파일 접근 실패: %s — %v", path, err)
	}

	// excelize 로 읽기
	f, err := excelize.OpenFile(path)
	if err != nil {
		return map[string]any{"success": false},
			fmt.Sprintf("Excel 파일 읽기 실패: %v", err)
	}
	defer f.Close()

	// 모든 시트 검사
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return map[string]any{"success": false}, "시트가 비어있어요."
	}
	firstSheet := sheets[0]
	rows, err := f.GetRows(firstSheet)
	if err != nil {
		return map[string]any{"success": false},
			fmt.Sprintf("시트 읽기 실패: %v", err)
	}

	totalRows := len(rows)
	totalCols := 0
	if totalRows > 0 {
		totalCols = len(rows[0])
	}
	// 샘플: 처음 5행 + 마지막 3행 (LLM 토큰 절약)
	sample := []map[string]any{}
	previewRows := []string{}
	maxSample := 8
	if totalRows <= maxSample {
		for i, row := range rows {
			previewRows = append(previewRows, strings.Join(row, " | "))
			if i < 5 {
				sample = append(sample, rowToMap(rows[0], row))
			}
		}
	} else {
		for i := 0; i < 5 && i < totalRows; i++ {
			previewRows = append(previewRows, strings.Join(rows[i], " | "))
			sample = append(sample, rowToMap(rows[0], rows[i]))
		}
		previewRows = append(previewRows, "...")
		for i := totalRows - 3; i < totalRows; i++ {
			previewRows = append(previewRows, strings.Join(rows[i], " | "))
		}
	}

	// LLM 분석 프롬프트
	sysPrompt := ""
	if lang == "en" {
		sysPrompt = `You analyze Excel data. Output 2-3 short insights about the data.
- What it appears to contain (1 line)
- Key observation (1 line)
- Suggestion (1 line, optional)
Total: max 3 lines. No markdown.`
	} else {
		sysPrompt = `당신은 Excel 데이터 분석 비서입니다. 짧고 핵심적인 인사이트 2~3줄.
- 어떤 데이터인지 (1줄)
- 핵심 관찰 (1줄)
- 제안 (선택, 1줄)
총 3줄, 마크다운 X.`
	}

	userPrompt := fmt.Sprintf("파일: %s\n시트: %s\n행/열: %d × %d\n\n샘플 데이터:\n%s",
		filepath.Base(path), firstSheet, totalRows, totalCols, strings.Join(previewRows, "\n"))

	insight, _, _ := callGroqWithFallback([]groqMsg{
		{Role: "system", Content: sysPrompt},
		{Role: "user", Content: userPrompt},
	}, 300, false)

	if insight == "" {
		insight = fmt.Sprintf("%d행 × %d열 — %s 시트", totalRows, totalCols, firstSheet)
	}

	// 헤더 추출
	headers := []string{}
	if totalRows > 0 {
		headers = rows[0]
	}

	// 응답 구성
	sizeStr := fmt.Sprintf("%.1f KB", float64(info.Size())/1024)
	msg := fmt.Sprintf("📊 %s 분석\n   %d행 × %d열 · %s · %s\n\n%s",
		filepath.Base(path), totalRows-1, totalCols, sizeStr,
		info.ModTime().Format("2006-01-02 15:04"), insight)
	if lang == "en" {
		msg = fmt.Sprintf("📊 %s analysis\n   %d rows × %d cols · %s · %s\n\n%s",
			filepath.Base(path), totalRows-1, totalCols, sizeStr,
			info.ModTime().Format("2006-01-02 15:04"), insight)
	}

	resultBytes, _ := json.Marshal(sample)
	_ = resultBytes // for future use

	return map[string]any{
		"success":   true,
		"fileName":  filepath.Base(path),
		"path":      path,
		"url":       "file:///" + filepath.ToSlash(path),
		"mimeType":  "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"sheet":     firstSheet,
		"rows":      totalRows - 1,
		"cols":      totalCols,
		"headers":   headers,
		"sample":    sample,
		"insight":   insight,
		"operation": "excel_analyze",
		"size_bytes": info.Size(),
		"modified":  info.ModTime().Format("2006-01-02 15:04:05"),
	}, msg
}

// findLatestExcel: 바탕화면에서 가장 최근 .xlsx
func findLatestExcel() string {
	home, _ := os.UserHomeDir()
	desktop := filepath.Join(home, "Desktop")
	entries, err := os.ReadDir(desktop)
	if err != nil {
		return ""
	}
	type fileInfo struct {
		path    string
		modTime time.Time
	}
	files := []fileInfo{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := strings.ToLower(e.Name())
		if !strings.HasSuffix(name, ".xlsx") && !strings.HasSuffix(name, ".xls") {
			continue
		}
		info, _ := e.Info()
		files = append(files, fileInfo{
			path:    filepath.Join(desktop, e.Name()),
			modTime: info.ModTime(),
		})
	}
	if len(files) == 0 {
		return ""
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.After(files[j].modTime)
	})
	return files[0].path
}

// extractFilePath: 메시지에서 파일 경로 추출 (절대 경로)
func extractFilePath(s string) string {
	// Windows 경로
	if idx := strings.Index(s, "C:\\"); idx >= 0 {
		rest := s[idx:]
		// 공백/줄바꿈으로 끝
		for i, c := range rest {
			if c == ' ' || c == '\n' || c == '\t' {
				return rest[:i]
			}
		}
		return rest
	}
	// Unix 경로 (Mac)
	if idx := strings.Index(s, "/Users/"); idx >= 0 {
		rest := s[idx:]
		for i, c := range rest {
			if c == ' ' || c == '\n' || c == '\t' {
				return rest[:i]
			}
		}
		return rest
	}
	return ""
}

// rowToMap: 첫 행(헤더)을 키로, 데이터 행을 value 로 매핑
func rowToMap(headers, row []string) map[string]any {
	m := map[string]any{}
	for i, h := range headers {
		if i < len(row) {
			m[h] = row[i]
		} else {
			m[h] = ""
		}
	}
	return m
}
