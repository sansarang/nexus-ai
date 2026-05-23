//go:build windows

package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/xuri/excelize/v2"
)

// ──────────────────────────────────────────────────────────────────────────────
// AI가 반환하는 Excel 편집 명령 단위
// ──────────────────────────────────────────────────────────────────────────────

type ExcelOp struct {
	Type      string     `json:"type"`
	Sheet     string     `json:"sheet,omitempty"`
	Cell      string     `json:"cell,omitempty"`
	Value     string     `json:"value,omitempty"`
	Rows      []int      `json:"rows,omitempty"`
	At        int        `json:"at,omitempty"`
	Values    [][]string `json:"values,omitempty"`
	Start     string     `json:"start,omitempty"`
	Col       string     `json:"col,omitempty"`
	Asc       *bool      `json:"ascending,omitempty"`
	Formula   string     `json:"formula,omitempty"`
	FillRows  int        `json:"rows_count,omitempty"` // fill_formula 에서 행 수
	Range     string     `json:"range,omitempty"`
	Style     string     `json:"style,omitempty"`
	ChartType string     `json:"chart_type,omitempty"`
	Title     string     `json:"title,omitempty"`
	OldName   string     `json:"old,omitempty"`
	NewName   string     `json:"new,omitempty"`
	Width     float64    `json:"width,omitempty"`
	Row       int        `json:"row,omitempty"`
}

// AI 응답 전체 구조
type AIDocEditReply struct {
	Summary    string    `json:"summary"`
	Operations []ExcelOp `json:"operations"`
	NewText    string    `json:"new_text,omitempty"`
}

// ──────────────────────────────────────────────────────────────────────────────
// POST /api/docs/upload
// ──────────────────────────────────────────────────────────────────────────────

func handleDocUpload(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	r.ParseMultipartForm(50 << 20)

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("파일 필드 'file'이 없습니다", "File field 'file' is missing", lang)})
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowed := map[string]bool{
		".xlsx": true, ".xls": true, ".xlsm": true,
		".docx": true, ".doc": true,
		".txt": true, ".csv": true, ".md": true, ".pdf": true,
	}
	if !allowed[ext] {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("지원하지 않는 파일 형식: ", "Unsupported file format: ", lang) + ext})
		return
	}

	tmpDir := filepath.Join(os.TempDir(), "nexus_docs")
	os.MkdirAll(tmpDir, 0755)

	ts := time.Now().Format("20060102_150405")
	safeName := sanitizeDocFilename(strings.TrimSuffix(header.Filename, ext))
	savePath := filepath.Join(tmpDir, safeName+"_"+ts+ext)

	dst, err := os.Create(savePath)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": msgT("저장 실패: ", "Save failed: ", lang) + err.Error()})
		return
	}
	defer dst.Close()
	io.Copy(dst, file)
	dst.Close()

	preview, previewErr := buildDocPreview(savePath, ext)
	resp := map[string]any{
		"success":   true,
		"file_path": savePath,
		"filename":  header.Filename,
		"ext":       ext,
		"size":      header.Size,
	}
	if previewErr == nil {
		resp["preview"] = preview
	} else {
		resp["preview_error"] = previewErr.Error()
	}
	json200(w, resp)
}

// ──────────────────────────────────────────────────────────────────────────────
// POST /api/docs/ai-edit
// ──────────────────────────────────────────────────────────────────────────────

func handleDocAIEdit(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		FilePath    string `json:"file_path"`
		Instruction string `json:"instruction"`
		SaveAs      string `json:"save_as"`
		SheetName   string `json:"sheet_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.FilePath == "" || req.Instruction == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("file_path와 instruction 필수", "file_path and instruction are required", lang)})
		return
	}
	if _, err := os.Stat(req.FilePath); err != nil {
		writeJSON(w, 404, map[string]any{"success": false, "message": msgT("파일을 찾을 수 없습니다: ", "File not found: ", lang) + req.FilePath})
		return
	}

	ext := strings.ToLower(filepath.Ext(req.FilePath))

	var outPath, summary string
	var ops []ExcelOp
	var aiErr error

	switch ext {
	case ".xlsx", ".xlsm", ".xls":
		outPath, summary, ops, aiErr = aiEditExcel(req.FilePath, req.Instruction, req.SheetName, req.SaveAs)
	default:
		outPath, summary, aiErr = aiEditTextDoc(req.FilePath, req.Instruction, req.SaveAs)
	}

	if aiErr != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": aiErr.Error()})
		return
	}

	resp := map[string]any{
		"success":  true,
		"out_path": outPath,
		"summary":  summary,
		"message":  fmt.Sprintf(msgT("문서 편집 완료: %s", "Document editing complete: %s", lang), filepath.Base(outPath)),
	}
	if len(ops) > 0 {
		resp["operations_count"] = len(ops)
		opTypes := make([]string, 0, len(ops))
		for _, op := range ops {
			opTypes = append(opTypes, op.Type)
		}
		resp["operations"] = opTypes
	}
	json200(w, resp)
}

// ──────────────────────────────────────────────────────────────────────────────
// GET /api/excel/read   ?path=...&sheet=...
// ──────────────────────────────────────────────────────────────────────────────

func handleReadExcel(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	filePath := r.URL.Query().Get("path")
	sheetName := r.URL.Query().Get("sheet")
	if filePath == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("path 파라미터 필요", "path parameter is required", lang)})
		return
	}
	data, sheets, err := readExcelToJSON(filePath, sheetName)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}
	json200(w, map[string]any{
		"success":    true,
		"sheets":     sheets,
		"data":       data,
		"rows":       len(data),
		"sheet_name": func() string { if len(sheets) > 0 { return sheets[0] } else { return "" } }(),
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Excel AI 편집 핵심 로직
// ──────────────────────────────────────────────────────────────────────────────

func aiEditExcel(filePath, instruction, sheetHint, saveAs string) (outPath, summary string, ops []ExcelOp, err error) {
	content, sheets, readErr := readExcelToJSON(filePath, sheetHint)
	if readErr != nil {
		err = fmt.Errorf("Excel 읽기 실패: %w", readErr)
		return
	}

	targetSheet := sheetHint
	if targetSheet == "" && len(sheets) > 0 {
		targetSheet = sheets[0]
	}

	contentJSON, _ := json.Marshal(content)
	if len(contentJSON) > 12000 {
		contentJSON = contentJSON[:12000]
	}

	summaryLang := "수행한 작업을 한국어로 간략히 요약"
	if isEnglishQuery(instruction) {
		summaryLang = "Brief summary of the operation performed in English"
	}
	systemPrompt := fmt.Sprintf(`You are an Excel AI editor. Follow the user's instructions to modify the Excel file and respond ONLY in the JSON format below:
{
  "summary": "%s",`, summaryLang) + `
  "operations": [
    {"type": "set_cell", "sheet": "시트명", "cell": "A1", "value": "새값"},
    {"type": "set_range", "sheet": "시트명", "start": "A2", "values": [["행1열1","행1열2"],["행2열1","행2열2"]]},
    {"type": "delete_rows", "sheet": "시트명", "rows": [3,5]},
    {"type": "insert_row", "sheet": "시트명", "at": 2, "values": [["값1","값2","값3"]]},
    {"type": "sort", "sheet": "시트명", "col": "B", "ascending": false},
    {"type": "formula", "sheet": "시트명", "cell": "D2", "formula": "=SUM(A2:C2)"},
    {"type": "fill_formula", "sheet": "시트명", "cell": "D2", "formula": "=SUM(A2:C2)", "rows_count": 10},
    {"type": "style", "sheet": "시트명", "range": "A1:Z1", "style": "header"},
    {"type": "col_width", "sheet": "시트명", "col": "A", "width": 20},
    {"type": "freeze", "sheet": "시트명", "row": 1},
    {"type": "rename_sheet", "old": "Sheet1", "new": "데이터"},
    {"type": "add_sheet", "new": "분석결과"},
    {"type": "delete_sheet", "sheet": "삭제할시트"},
    {"type": "merge", "sheet": "시트명", "range": "A1:C1"},
    {"type": "chart", "sheet": "시트명", "chart_type": "bar", "range": "A1:B10", "title": "차트제목"}
  ]
}
style 종류: header(노란배경+굵음) highlight(연파랑) currency(통화형식) percent(퍼센트) bold(굵음) date(날짜형식)
chart_type: bar(가로막대) col(세로막대) line(꺾은선) pie(원형)`

	userMsg := fmt.Sprintf(
		"시트 목록: %v\n현재 시트('%s') 데이터:\n%s\n\n지시사항: %s",
		sheets, targetSheet, string(contentJSON), instruction,
	)

	msgs := []groqMsg{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userMsg},
	}

	answer, _, aiErr := callGroqWithFallback(msgs, 4096, true)
	if aiErr != nil {
		err = fmt.Errorf("AI 호출 실패: %w", aiErr)
		return
	}

	var reply AIDocEditReply
	if jsonErr := json.Unmarshal([]byte(cleanJSONString(answer)), &reply); jsonErr != nil {
		err = fmt.Errorf("AI 응답 파싱 실패: %w (응답: %.300s)", jsonErr, answer)
		return
	}
	summary = reply.Summary
	ops = reply.Operations

	outPath = buildDocOutPath(filePath, saveAs, "_edited")

	if copyErr := copyDocFile(filePath, outPath); copyErr != nil {
		err = fmt.Errorf("파일 복사 실패: %w", copyErr)
		return
	}

	if applyErr := applyExcelOps(outPath, reply.Operations, targetSheet); applyErr != nil {
		err = fmt.Errorf("Excel 편집 적용 실패: %w", applyErr)
		return
	}
	return
}

// ──────────────────────────────────────────────────────────────────────────────
// Excel 연산 적용 (excelize)
// ──────────────────────────────────────────────────────────────────────────────

func applyExcelOps(filePath string, ops []ExcelOp, defaultSheet string) error {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return fmt.Errorf("excelize 열기 실패: %w", err)
	}
	defer f.Close()

	styleCache := map[string]int{}

	resolveSheet := func(op ExcelOp) string {
		if op.Sheet != "" {
			return op.Sheet
		}
		return defaultSheet
	}

	for _, op := range ops {
		sheet := resolveSheet(op)

		switch op.Type {

		case "set_cell":
			if op.Cell != "" {
				f.SetCellValue(sheet, op.Cell, op.Value)
			}

		case "set_range":
			if op.Start == "" || len(op.Values) == 0 {
				continue
			}
			startCol, startRow, pErr := excelize.CellNameToCoordinates(op.Start)
			if pErr != nil {
				continue
			}
			for ri, row := range op.Values {
				for ci, val := range row {
					colName, _ := excelize.ColumnNumberToName(startCol + ci)
					f.SetCellValue(sheet, fmt.Sprintf("%s%d", colName, startRow+ri), val)
				}
			}

		case "delete_rows":
			rowNums := make([]int, len(op.Rows))
			copy(rowNums, op.Rows)
			sort.Sort(sort.Reverse(sort.IntSlice(rowNums)))
			for _, row := range rowNums {
				if row > 0 {
					f.RemoveRow(sheet, row)
				}
			}

		case "insert_row":
			if op.At <= 0 {
				continue
			}
			f.InsertRows(sheet, op.At, 1)
			if len(op.Values) > 0 {
				for ci, val := range op.Values[0] {
					colName, _ := excelize.ColumnNumberToName(ci + 1)
					f.SetCellValue(sheet, fmt.Sprintf("%s%d", colName, op.At), val)
				}
			}

		case "sort":
			if op.Col != "" {
				asc := op.Asc == nil || *op.Asc
				sortExcelByCol(f, sheet, op.Col, asc)
			}

		case "formula":
			if op.Cell != "" && op.Formula != "" {
				f.SetCellFormula(sheet, op.Cell, op.Formula)
			}

		case "fill_formula":
			if op.Cell == "" || op.Formula == "" || op.FillRows <= 0 {
				continue
			}
			startCol, startRow, pErr := excelize.CellNameToCoordinates(op.Cell)
			if pErr != nil {
				continue
			}
			colName, _ := excelize.ColumnNumberToName(startCol)
			for i := 0; i < op.FillRows; i++ {
				cellRef := fmt.Sprintf("%s%d", colName, startRow+i)
				adjusted := shiftFormulaRowNums(op.Formula, i)
				f.SetCellFormula(sheet, cellRef, adjusted)
			}

		case "style":
			if op.Range == "" {
				continue
			}
			sid, ok := styleCache[op.Style]
			if !ok {
				sid = buildExcelStyleID(f, op.Style)
				styleCache[op.Style] = sid
			}
			parts := strings.SplitN(op.Range, ":", 2)
			if len(parts) == 2 {
				f.SetCellStyle(sheet, parts[0], parts[1], sid)
			} else {
				f.SetCellStyle(sheet, op.Range, op.Range, sid)
			}

		case "col_width":
			if op.Col != "" && op.Width > 0 {
				f.SetColWidth(sheet, op.Col, op.Col, op.Width)
			}

		case "freeze":
			row := op.Row
			if row <= 0 {
				row = 1
			}
			f.SetPanes(sheet, &excelize.Panes{
				Freeze:      true,
				YSplit:      row,
				TopLeftCell: fmt.Sprintf("A%d", row+1),
				ActivePane:  "bottomLeft",
			})

		case "rename_sheet":
			if op.OldName != "" && op.NewName != "" {
				f.SetSheetName(op.OldName, op.NewName)
			}

		case "add_sheet":
			name := op.NewName
			if name == "" {
				name = op.Title
			}
			if name != "" {
				f.NewSheet(name)
			}

		case "delete_sheet":
			target := op.Sheet
			if target == "" {
				target = op.OldName
			}
			if target != "" {
				f.DeleteSheet(target)
			}

		case "merge":
			if op.Range != "" {
				parts := strings.SplitN(op.Range, ":", 2)
				if len(parts) == 2 {
					f.MergeCell(sheet, parts[0], parts[1])
				}
			}

		case "chart":
			if op.Range == "" {
				continue
			}
			chartType := excelize.Bar
			switch strings.ToLower(op.ChartType) {
			case "col", "column":
				chartType = excelize.Col
			case "line":
				chartType = excelize.Line
			case "pie":
				chartType = excelize.Pie
			}
			parts := strings.SplitN(op.Range, ":", 2)
			if len(parts) != 2 {
				continue
			}
			series := []excelize.ChartSeries{{
				Name:   op.Title,
				Values: fmt.Sprintf("%s!$%s:$%s", sheet, parts[0], parts[1]),
			}}
			chartCell := nextFreeCell(f, sheet)
			f.AddChart(sheet, chartCell, &excelize.Chart{
				Type:   chartType,
				Series: series,
				Title:  []excelize.RichTextRun{{Text: op.Title}},
				PlotArea: excelize.ChartPlotArea{ShowVal: true},
				Dimension: excelize.ChartDimension{Width: 480, Height: 320},
			})
		}
	}

	return f.Save()
}

// ──────────────────────────────────────────────────────────────────────────────
// 텍스트 문서 AI 편집 (TXT / CSV / DOCX / MD)
// ──────────────────────────────────────────────────────────────────────────────

func aiEditTextDoc(filePath, instruction, saveAs string) (outPath, summary string, err error) {
	original, readErr := extractDocumentText(filePath)
	if readErr != nil {
		err = fmt.Errorf("문서 읽기 실패: %w", readErr)
		return
	}

	if utf8.RuneCountInString(original) > 8000 {
		runes := []rune(original)
		original = string(runes[:8000]) + "\n...[이하 내용 생략]..."
	}

	var systemPrompt string
	if isEnglishQuery(instruction) {
		systemPrompt = `You are a document editing AI. Follow the user's instructions to edit the document and respond ONLY in the JSON format below:
{"summary":"Brief summary of the operation in English","new_text":"The complete modified document content"}`
	} else {
		systemPrompt = `당신은 문서 편집 AI입니다. 사용자 지시에 따라 문서 내용을 수정하고 반드시 아래 JSON으로만 응답하세요:
{"summary":"수행한 작업을 한국어로 간략히 요약","new_text":"수정된 전체 문서 내용"}`
	}

	msgs := []groqMsg{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: fmt.Sprintf("원본 문서:\n%s\n\n지시사항: %s", original, instruction)},
	}

	answer, _, aiErr := callGroqWithFallback(msgs, 8192, true)
	if aiErr != nil {
		err = fmt.Errorf("AI 호출 실패: %w", aiErr)
		return
	}

	var reply AIDocEditReply
	if jsonErr := json.Unmarshal([]byte(cleanJSONString(answer)), &reply); jsonErr != nil {
		err = fmt.Errorf("AI 응답 파싱 실패: %w", jsonErr)
		return
	}
	summary = reply.Summary

	ext := strings.ToLower(filepath.Ext(filePath))
	outPath = buildDocOutPath(filePath, saveAs, "_edited")

	if ext == ".docx" || ext == ".doc" {
		err = buildSimpleDocx(outPath, reply.NewText)
		return
	}

	err = os.WriteFile(outPath, []byte(reply.NewText), 0644)
	return
}

// ──────────────────────────────────────────────────────────────────────────────
// Excel 읽기 → 2D 배열
// ──────────────────────────────────────────────────────────────────────────────

func readExcelToJSON(filePath, sheetName string) ([][]string, []string, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("excelize 열기 실패: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, nil, fmt.Errorf("시트가 없습니다")
	}

	target := sheets[0]
	for _, s := range sheets {
		if s == sheetName {
			target = s
			break
		}
	}

	rows, err := f.GetRows(target)
	if err != nil {
		return nil, sheets, fmt.Errorf("시트 읽기 실패: %w", err)
	}
	if len(rows) > 500 {
		rows = rows[:500]
	}
	return rows, sheets, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// 미리보기 생성
// ──────────────────────────────────────────────────────────────────────────────

func buildDocPreview(filePath, ext string) (any, error) {
	switch ext {
	case ".xlsx", ".xlsm", ".xls":
		rows, sheets, err := readExcelToJSON(filePath, "")
		if err != nil {
			return nil, err
		}
		preview := rows
		if len(preview) > 20 {
			preview = preview[:20]
		}
		return map[string]any{"sheets": sheets, "rows": preview, "total_rows": len(rows)}, nil
	default:
		text, err := extractDocumentText(filePath)
		if err != nil {
			return nil, err
		}
		runes := []rune(text)
		if len(runes) > 2000 {
			text = string(runes[:2000]) + "..."
		}
		return map[string]any{"text": text}, nil
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// excelize 스타일 빌더
// ──────────────────────────────────────────────────────────────────────────────

func buildExcelStyleID(f *excelize.File, styleName string) int {
	var s excelize.Style
	switch strings.ToLower(styleName) {
	case "header":
		s = excelize.Style{
			Font: &excelize.Font{Bold: true, Size: 11, Color: "1F1F1F"},
			Fill: excelize.Fill{Type: "pattern", Color: []string{"FFF176"}, Pattern: 1},
			Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
			Border: []excelize.Border{{Type: "bottom", Color: "9E9E9E", Style: 2}},
		}
	case "highlight":
		s = excelize.Style{
			Fill: excelize.Fill{Type: "pattern", Color: []string{"E3F2FD"}, Pattern: 1},
		}
	case "currency":
		s = excelize.Style{NumFmt: 44}
	case "percent":
		s = excelize.Style{NumFmt: 10}
	case "date":
		s = excelize.Style{NumFmt: 14}
	case "bold":
		s = excelize.Style{Font: &excelize.Font{Bold: true}}
	}
	id, _ := f.NewStyle(&s)
	return id
}

// ──────────────────────────────────────────────────────────────────────────────
// 열 기준 정렬 (헤더 1행 유지)
// ──────────────────────────────────────────────────────────────────────────────

func sortExcelByCol(f *excelize.File, sheet, col string, asc bool) {
	rows, err := f.GetRows(sheet)
	if err != nil || len(rows) < 2 {
		return
	}

	colIdx := 0
	for i := 1; i <= 702; i++ {
		name, _ := excelize.ColumnNumberToName(i)
		if strings.EqualFold(name, col) {
			colIdx = i - 1
			break
		}
	}

	header := rows[0]
	data := rows[1:]

	sort.SliceStable(data, func(i, j int) bool {
		vi, vj := "", ""
		if colIdx < len(data[i]) {
			vi = data[i][colIdx]
		}
		if colIdx < len(data[j]) {
			vj = data[j][colIdx]
		}
		ni, ei := strconv.ParseFloat(strings.ReplaceAll(vi, ",", ""), 64)
		nj, ej := strconv.ParseFloat(strings.ReplaceAll(vj, ",", ""), 64)
		if ei == nil && ej == nil {
			if asc {
				return ni < nj
			}
			return ni > nj
		}
		if asc {
			return vi < vj
		}
		return vi > vj
	})

	allRows := append([][]string{header}, data...)
	for ri, row := range allRows {
		for ci, val := range row {
			colName, _ := excelize.ColumnNumberToName(ci + 1)
			f.SetCellValue(sheet, fmt.Sprintf("%s%d", colName, ri+1), val)
		}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// 수식 행 번호 오프셋 조정 (fill_formula 용)
// ──────────────────────────────────────────────────────────────────────────────

func shiftFormulaRowNums(formula string, offset int) string {
	if offset == 0 {
		return formula
	}
	var result strings.Builder
	runes := []rune(formula)
	i := 0
	for i < len(runes) {
		ch := runes[i]
		if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') {
			j := i + 1
			for j < len(runes) && ((runes[j] >= 'A' && runes[j] <= 'Z') || (runes[j] >= 'a' && runes[j] <= 'z')) {
				j++
			}
			result.WriteString(string(runes[i:j]))
			i = j
			if i < len(runes) && runes[i] >= '1' && runes[i] <= '9' {
				k := i
				for k < len(runes) && runes[k] >= '0' && runes[k] <= '9' {
					k++
				}
				rowNum, _ := strconv.Atoi(string(runes[i:k]))
				result.WriteString(strconv.Itoa(rowNum + offset))
				i = k
			}
		} else {
			result.WriteRune(ch)
			i++
		}
	}
	return result.String()
}

// ──────────────────────────────────────────────────────────────────────────────
// 차트 삽입 위치: 데이터 오른쪽 2칸 빈 열
// ──────────────────────────────────────────────────────────────────────────────

func nextFreeCell(f *excelize.File, sheet string) string {
	rows, err := f.GetRows(sheet)
	if err != nil || len(rows) == 0 {
		return "H1"
	}
	maxCol := 0
	for _, row := range rows {
		if len(row) > maxCol {
			maxCol = len(row)
		}
	}
	col, _ := excelize.ColumnNumberToName(maxCol + 2)
	return col + "1"
}

// ──────────────────────────────────────────────────────────────────────────────
// 간단한 DOCX 생성 (plain text → OOXML ZIP)
// ──────────────────────────────────────────────────────────────────────────────

func buildSimpleDocx(outPath, text string) error {
	os.MkdirAll(filepath.Dir(outPath), 0755)

	xmlEsc := strings.NewReplacer(
		"&", "&amp;", "<", "&lt;", ">", "&gt;", "'", "&apos;", `"`, "&quot;",
	)

	var paras strings.Builder
	for _, line := range strings.Split(text, "\n") {
		escaped := xmlEsc.Replace(line)
		paras.WriteString(fmt.Sprintf(
			`<w:p><w:r><w:t xml:space="preserve">%s</w:t></w:r></w:p>`, escaped,
		))
	}

	docXML := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`+
		`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">`+
		`<w:body>%s<w:sectPr><w:pgSz w:w="12240" w:h="15840"/></w:sectPr></w:body></w:document>`,
		paras.String(),
	)

	contentTypes := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
		`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
		`<Default Extension="xml" ContentType="application/xml"/>` +
		`<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>` +
		`</Types>`

	rootRels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
		`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>` +
		`</Relationships>`

	wordRels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"></Relationships>`

	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	entries := []struct{ name, data string }{
		{"[Content_Types].xml", contentTypes},
		{"_rels/.rels", rootRels},
		{"word/document.xml", docXML},
		{"word/_rels/document.xml.rels", wordRels},
	}
	for _, e := range entries {
		w, zerr := zw.Create(e.name)
		if zerr != nil {
			return zerr
		}
		if _, werr := io.WriteString(w, e.data); werr != nil {
			return werr
		}
	}
	return nil
}

// ──────────────────────────────────────────────────────────────────────────────
// 헬퍼 함수들
// ──────────────────────────────────────────────────────────────────────────────

func copyDocFile(src, dst string) error {
	os.MkdirAll(filepath.Dir(dst), 0755)
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func buildDocOutPath(src, saveAs, suffix string) string {
	if saveAs != "" {
		return saveAs
	}
	ext := filepath.Ext(src)
	base := strings.TrimSuffix(filepath.Base(src), ext)
	// 타임스탬프 부분 제거 (업로드 시 붙은 것)
	if idx := strings.LastIndex(base, "_"); idx > 0 {
		possible := base[idx+1:]
		if len(possible) == 15 { // "20060102_150405" 형태
			base = base[:idx]
		}
	}
	ts := time.Now().Format("20060102_150405")
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Desktop", base+suffix+"_"+ts+ext)
}

func cleanJSONString(s string) string {
	s = strings.TrimSpace(s)
	// 마크다운 코드 블록 제거
	for _, fence := range []string{"```json", "```"} {
		if idx := strings.Index(s, fence); idx >= 0 {
			s = s[idx+len(fence):]
			break
		}
	}
	if idx := strings.LastIndex(s, "```"); idx >= 0 {
		s = s[:idx]
	}
	// JSON 객체만 추출
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		s = s[start : end+1]
	}
	return strings.TrimSpace(s)
}

func sanitizeDocFilename(s string) string {
	replacer := strings.NewReplacer(
		"/", "_", "\\", "_", ":", "_", "*", "_",
		"?", "_", `"`, "_", "<", "_", ">", "_", "|", "_",
	)
	s = replacer.Replace(s)
	runes := []rune(s)
	if len(runes) > 50 {
		s = string(runes[:50])
	}
	s = strings.TrimSpace(s)
	if s == "" {
		s = "document"
	}
	return s
}

// bytes.Buffer 패키지 참조 확인용 (미사용 import 방지)
var _ = bytes.NewBuffer
