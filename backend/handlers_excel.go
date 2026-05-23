//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// ──────────────────────────────────────────────────────────────
// saveToExcel: 2D 슬라이스를 Excel 파일로 저장
// ──────────────────────────────────────────────────────────────

func saveToExcel(data [][]string, outPath, sheetTitle string) error {
	if len(data) == 0 {
		return fmt.Errorf("데이터 없음")
	}

	f := excelize.NewFile()
	defer f.Close()

	sheetName := "Sheet1"
	if sheetTitle != "" {
		safe := sanitizeExcelSheet(sheetTitle)
		if safe != "" {
			sheetName = safe
		}
	}

	if sheetName != "Sheet1" {
		f.NewSheet(sheetName)
		f.DeleteSheet("Sheet1")
	}

	// 헤더 스타일 (노랑 배경, 굵은 글씨)
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 11, Color: "1F1F1F"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"FFF9C4"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})

	// 데이터 스타일 (교대 행 색상)
	dataStyleEven, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"FFFFFF"}, Pattern: 1},
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "EEEEEE", Style: 1},
			{Type: "right", Color: "EEEEEE", Style: 1},
			{Type: "top", Color: "EEEEEE", Style: 1},
			{Type: "bottom", Color: "EEEEEE", Style: 1},
		},
	})
	dataStyleOdd, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"F5F5F5"}, Pattern: 1},
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "EEEEEE", Style: 1},
			{Type: "right", Color: "EEEEEE", Style: 1},
			{Type: "top", Color: "EEEEEE", Style: 1},
			{Type: "bottom", Color: "EEEEEE", Style: 1},
		},
	})

	colWidths := make(map[int]int)

	for rowIdx, row := range data {
		excelRow := rowIdx + 1
		for colIdx, cell := range row {
			colLetter, _ := excelize.ColumnNumberToName(colIdx + 1)
			cellRef := fmt.Sprintf("%s%d", colLetter, excelRow)
			f.SetCellValue(sheetName, cellRef, cell)

			// 열 너비 계산 (최대 50자)
			w := len([]rune(cell))
			if w > colWidths[colIdx] {
				colWidths[colIdx] = w
			}

			// 스타일 적용
			if rowIdx == 0 {
				f.SetCellStyle(sheetName, cellRef, cellRef, headerStyle)
			} else if rowIdx%2 == 0 {
				f.SetCellStyle(sheetName, cellRef, cellRef, dataStyleEven)
			} else {
				f.SetCellStyle(sheetName, cellRef, cellRef, dataStyleOdd)
			}
		}
	}

	// 열 너비 자동 조정 (최소 10, 최대 50)
	for colIdx, w := range colWidths {
		colLetter, _ := excelize.ColumnNumberToName(colIdx + 1)
		adjustedW := float64(w) * 1.2
		if adjustedW < 10 {
			adjustedW = 10
		}
		if adjustedW > 50 {
			adjustedW = 50
		}
		f.SetColWidth(sheetName, colLetter, colLetter, adjustedW)
	}

	// 첫 행 고정 (헤더)
	f.SetPanes(sheetName, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})

	// 메타 시트 추가 (생성 정보)
	metaSheet := "메타"
	f.NewSheet(metaSheet)
	f.SetCellValue(metaSheet, "A1", "생성일시")
	f.SetCellValue(metaSheet, "B1", time.Now().Format("2006-01-02 15:04:05"))
	f.SetCellValue(metaSheet, "A2", "제목")
	f.SetCellValue(metaSheet, "B2", sheetTitle)
	f.SetCellValue(metaSheet, "A3", "데이터 행 수")
	f.SetCellValue(metaSheet, "B3", len(data)-1)
	f.SetCellValue(metaSheet, "A4", "생성 도구")
	f.SetCellValue(metaSheet, "B4", "Nexus AI Assistant")

	// 첫 번째 시트 활성화
	if idx, err := f.GetSheetIndex(sheetName); err == nil {
		f.SetActiveSheet(idx)
	}

	// 디렉토리 생성
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return err
	}

	return f.SaveAs(outPath)
}

func sanitizeExcelSheet(s string) string {
	replacer := strings.NewReplacer(
		"\\", "", "/", "", "?", "", "*", "",
		"[", "", "]", "", ":", "",
	)
	s = replacer.Replace(s)
	if len([]rune(s)) > 31 {
		runes := []rune(s)
		s = string(runes[:31])
	}
	return strings.TrimSpace(s)
}

// ──────────────────────────────────────────────────────────────
// POST /api/excel/save
// 데이터를 Excel로 저장
// ──────────────────────────────────────────────────────────────

func handleExcelSave(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		Data     [][]string `json:"data"`       // 행 × 열 데이터
		Title    string     `json:"title"`      // 시트 제목
		Filename string     `json:"filename"`   // 파일명 (확장자 없이)
		SavePath string     `json:"save_path"`  // 저장 경로 (비어있으면 바탕화면)
	}
	json.NewDecoder(r.Body).Decode(&req)

	if len(req.Data) == 0 {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("data 필요", "data is required", lang)})
		return
	}

	filename := req.Filename
	if filename == "" {
		filename = fmt.Sprintf("nexus_export_%s", time.Now().Format("20060102_150405"))
	}
	if !strings.HasSuffix(filename, ".xlsx") {
		filename += ".xlsx"
	}

	savePath := req.SavePath
	if savePath == "" {
		home, _ := os.UserHomeDir()
		savePath = filepath.Join(home, "Desktop", filename)
	}

	if err := saveToExcel(req.Data, savePath, req.Title); err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": msgT("Excel 저장 실패: ", "Excel save failed: ", lang) + err.Error()})
		return
	}

	json200(w, map[string]any{
		"success":   true,
		"path":      savePath,
		"rows":      len(req.Data) - 1,
		"message":   fmt.Sprintf(msgT("Excel 저장 완료: %s (%d행)", "Excel saved: %s (%d rows)", lang), savePath, len(req.Data)-1),
	})
}

// ──────────────────────────────────────────────────────────────
// GET /api/excel/list
// 바탕화면의 Nexus 생성 Excel 목록
// ──────────────────────────────────────────────────────────────

func handleExcelList(w http.ResponseWriter, r *http.Request) {
	home, _ := os.UserHomeDir()
	desktop := filepath.Join(home, "Desktop")

	type ExcelFileInfo struct {
		Name     string `json:"name"`
		Path     string `json:"path"`
		Size     int64  `json:"size"`
		Modified string `json:"modified"`
	}

	var files []ExcelFileInfo
	entries, _ := os.ReadDir(desktop)
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".xlsx") {
			info, _ := e.Info()
			files = append(files, ExcelFileInfo{
				Name:     e.Name(),
				Path:     filepath.Join(desktop, e.Name()),
				Size:     info.Size(),
				Modified: info.ModTime().Format("2006-01-02 15:04:05"),
			})
		}
	}

	json200(w, map[string]any{
		"success": true,
		"files":   files,
		"total":   len(files),
	})
}
