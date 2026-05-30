//go:build windows

package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════════
//  Office COM 자동화 (PowerShell 브리지)
//  열린 Excel/Word 인스턴스 조작 — 셀 편집, 수식, 차트, 매크로 실행
// ═══════════════════════════════════════════════════════════════════════════

// runPowerShellCOM: 짧은 PS COM 스크립트 실행 + stdout 반환
func runPowerShellCOM(ctx context.Context, script string) (string, error) {
	cmd := newHiddenCmdCtx(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

// ── Excel COM ─────────────────────────────────────────────────────────────

// GET /api/excel/com/workbooks — 열려있는 Excel 워크북 목록
func handleExcelComWorkbooks(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	script := `
$ErrorActionPreference = 'SilentlyContinue'
try {
  $excel = [Runtime.InteropServices.Marshal]::GetActiveObject('Excel.Application')
  $wbs = @()
  foreach ($wb in $excel.Workbooks) {
    $wbs += [PSCustomObject]@{
      name = $wb.Name
      path = $wb.FullName
      sheet = $wb.ActiveSheet.Name
      sheets = @($wb.Worksheets | ForEach-Object { $_.Name })
    }
  }
  $wbs | ConvertTo-Json -Compress -Depth 5
} catch {
  '[]'
}
`
	out, err := runPowerShellCOM(ctx, script)
	if err != nil {
		json200(w, map[string]any{"success": false, "message": fmt.Sprintf("Excel COM 접근 실패: %v", err), "workbooks": []any{}})
		return
	}
	if out == "" || out == "[]" {
		json200(w, map[string]any{"success": true, "workbooks": []any{}, "message": "열려있는 Excel 워크북 없음. Excel을 먼저 실행해주세요."})
		return
	}
	// JSON 그대로 전달
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"success":true,"workbooks":%s}`, out)
}

// POST /api/excel/com/set-cell — {workbook, sheet, cell, value}
func handleExcelComSetCell(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		Workbook string      `json:"workbook"` // 빈 문자열이면 활성 워크북
		Sheet    string      `json:"sheet"`    // 빈 문자열이면 활성 시트
		Cell     string      `json:"cell"`     // 예: "A1", "B3"
		Value    interface{} `json:"value"`    // 문자열/숫자
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Cell == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "cell required"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	valStr := fmt.Sprintf("%v", req.Value)
	valStr = strings.ReplaceAll(valStr, "'", "''")

	script := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
try {
  $excel = [Runtime.InteropServices.Marshal]::GetActiveObject('Excel.Application')
  $wb = if ('%s' -ne '') { $excel.Workbooks | Where-Object { $_.Name -like '*%s*' } | Select-Object -First 1 } else { $excel.ActiveWorkbook }
  if ($null -eq $wb) { Write-Output "NO_WORKBOOK"; exit }
  $ws = if ('%s' -ne '') { $wb.Worksheets | Where-Object { $_.Name -eq '%s' } | Select-Object -First 1 } else { $wb.ActiveSheet }
  if ($null -eq $ws) { Write-Output "NO_SHEET"; exit }
  $ws.Range('%s').Value2 = '%s'
  Write-Output "OK"
} catch {
  Write-Output "ERROR: $_"
}
`, req.Workbook, req.Workbook, req.Sheet, req.Sheet, req.Cell, valStr)

	out, err := runPowerShellCOM(ctx, script)
	if err != nil {
		json200(w, map[string]any{"success": false, "message": err.Error()})
		return
	}
	if out == "OK" {
		json200(w, map[string]any{"success": true, "message": msgT(fmt.Sprintf("%s 셀에 '%s' 입력 완료", req.Cell, valStr), fmt.Sprintf("Set %s = '%s'", req.Cell, valStr), lang)})
	} else if out == "NO_WORKBOOK" {
		json200(w, map[string]any{"success": false, "message": "열린 워크북이 없어요. Excel을 먼저 실행해주세요."})
	} else if out == "NO_SHEET" {
		json200(w, map[string]any{"success": false, "message": fmt.Sprintf("'%s' 시트를 찾을 수 없어요.", req.Sheet)})
	} else {
		json200(w, map[string]any{"success": false, "message": out})
	}
}

// POST /api/excel/com/formula — {workbook, sheet, cell, formula}
// 예: {cell:"B1", formula:"=SUM(A1:A10)"}
func handleExcelComFormula(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		Workbook string `json:"workbook"`
		Sheet    string `json:"sheet"`
		Cell     string `json:"cell"`
		Formula  string `json:"formula"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Cell == "" || req.Formula == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "cell and formula required"})
		return
	}
	// = 로 시작하지 않으면 자동 추가
	if !strings.HasPrefix(req.Formula, "=") {
		req.Formula = "=" + req.Formula
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	formula := strings.ReplaceAll(req.Formula, "'", "''")
	script := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
try {
  $excel = [Runtime.InteropServices.Marshal]::GetActiveObject('Excel.Application')
  $wb = if ('%s' -ne '') { $excel.Workbooks | Where-Object { $_.Name -like '*%s*' } | Select-Object -First 1 } else { $excel.ActiveWorkbook }
  if ($null -eq $wb) { Write-Output "NO_WORKBOOK"; exit }
  $ws = if ('%s' -ne '') { $wb.Worksheets | Where-Object { $_.Name -eq '%s' } | Select-Object -First 1 } else { $wb.ActiveSheet }
  if ($null -eq $ws) { Write-Output "NO_SHEET"; exit }
  $ws.Range('%s').Formula = '%s'
  $result = $ws.Range('%s').Value2
  Write-Output "OK:$result"
} catch {
  Write-Output "ERROR: $_"
}
`, req.Workbook, req.Workbook, req.Sheet, req.Sheet, req.Cell, formula, req.Cell)

	out, err := runPowerShellCOM(ctx, script)
	if err != nil {
		json200(w, map[string]any{"success": false, "message": err.Error()})
		return
	}
	if strings.HasPrefix(out, "OK:") {
		result := strings.TrimPrefix(out, "OK:")
		json200(w, map[string]any{
			"success": true,
			"result":  result,
			"message": msgT(fmt.Sprintf("%s = %s 입력 완료, 결과: %s", req.Cell, req.Formula, result),
				fmt.Sprintf("Set %s = %s, result: %s", req.Cell, req.Formula, result), lang),
		})
	} else if out == "NO_WORKBOOK" {
		json200(w, map[string]any{"success": false, "message": "열린 워크북이 없어요."})
	} else {
		json200(w, map[string]any{"success": false, "message": out})
	}
}

// POST /api/excel/com/read-range — {workbook, sheet, range} → 2D 배열
func handleExcelComReadRange(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Workbook string `json:"workbook"`
		Sheet    string `json:"sheet"`
		Range    string `json:"range"` // 예: "A1:C10"
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Range == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "range required"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	script := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
try {
  $excel = [Runtime.InteropServices.Marshal]::GetActiveObject('Excel.Application')
  $wb = if ('%s' -ne '') { $excel.Workbooks | Where-Object { $_.Name -like '*%s*' } | Select-Object -First 1 } else { $excel.ActiveWorkbook }
  if ($null -eq $wb) { Write-Output '[]'; exit }
  $ws = if ('%s' -ne '') { $wb.Worksheets | Where-Object { $_.Name -eq '%s' } | Select-Object -First 1 } else { $wb.ActiveSheet }
  $range = $ws.Range('%s')
  $values = $range.Value2
  if ($values -is [object[,]]) {
    $rows = $values.GetUpperBound(0)
    $cols = $values.GetUpperBound(1)
    $arr = @()
    for ($i = 1; $i -le $rows; $i++) {
      $row = @()
      for ($j = 1; $j -le $cols; $j++) { $row += $values[$i, $j] }
      $arr += , $row
    }
    $arr | ConvertTo-Json -Compress -Depth 5
  } else {
    @(@($values)) | ConvertTo-Json -Compress -Depth 5
  }
} catch { '[]' }
`, req.Workbook, req.Workbook, req.Sheet, req.Sheet, req.Range)

	out, err := runPowerShellCOM(ctx, script)
	if err != nil || out == "" {
		json200(w, map[string]any{"success": false, "data": [][]any{}, "message": "범위 읽기 실패"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"success":true,"data":%s,"range":"%s"}`, out, req.Range)
}

// POST /api/excel/com/macro — VBA 매크로 실행 {workbook, macro_name, args}
func handleExcelComMacro(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		Workbook  string   `json:"workbook"`
		MacroName string   `json:"macro_name"`
		Args      []string `json:"args"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.MacroName == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "macro_name required"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// args 를 PowerShell array로 변환
	argsStr := ""
	if len(req.Args) > 0 {
		quoted := make([]string, len(req.Args))
		for i, a := range req.Args {
			quoted[i] = "'" + strings.ReplaceAll(a, "'", "''") + "'"
		}
		argsStr = "," + strings.Join(quoted, ",")
	}

	script := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
try {
  $excel = [Runtime.InteropServices.Marshal]::GetActiveObject('Excel.Application')
  $result = $excel.Run('%s'%s)
  Write-Output "OK:$result"
} catch {
  Write-Output "ERROR: $_"
}
`, req.MacroName, argsStr)

	out, err := runPowerShellCOM(ctx, script)
	if err != nil {
		json200(w, map[string]any{"success": false, "message": err.Error()})
		return
	}
	if strings.HasPrefix(out, "OK:") {
		result := strings.TrimPrefix(out, "OK:")
		json200(w, map[string]any{
			"success": true, "result": result,
			"message": msgT(fmt.Sprintf("매크로 '%s' 실행 완료", req.MacroName), fmt.Sprintf("Macro '%s' executed", req.MacroName), lang),
		})
	} else {
		json200(w, map[string]any{"success": false, "message": out})
	}
}

// POST /api/excel/com/chart — 차트 추가 {workbook, sheet, range, chart_type, title}
// chart_type: bar|line|pie|column|area|scatter
func handleExcelComChart(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		Workbook  string `json:"workbook"`
		Sheet     string `json:"sheet"`
		Range     string `json:"range"`     // 예: "A1:B10"
		ChartType string `json:"chart_type"` // bar|line|pie|column|area|scatter
		Title     string `json:"title"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Range == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "range required"})
		return
	}
	if req.ChartType == "" {
		req.ChartType = "column"
	}
	// Excel ChartType 상수 (xlChartType enum)
	chartTypeMap := map[string]int{
		"column":  51, // xlColumnClustered
		"bar":     57, // xlBarClustered
		"line":    4,  // xlLine
		"pie":     5,  // xlPie
		"area":    1,  // xlArea
		"scatter": -4169,
	}
	chartCode, ok := chartTypeMap[strings.ToLower(req.ChartType)]
	if !ok {
		chartCode = 51
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	title := strings.ReplaceAll(req.Title, "'", "''")
	script := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
try {
  $excel = [Runtime.InteropServices.Marshal]::GetActiveObject('Excel.Application')
  $wb = if ('%s' -ne '') { $excel.Workbooks | Where-Object { $_.Name -like '*%s*' } | Select-Object -First 1 } else { $excel.ActiveWorkbook }
  $ws = if ('%s' -ne '') { $wb.Worksheets | Where-Object { $_.Name -eq '%s' } | Select-Object -First 1 } else { $wb.ActiveSheet }
  $chart = $ws.Shapes.AddChart2(-1, %d).Chart
  $chart.SetSourceData($ws.Range('%s'))
  if ('%s' -ne '') {
    $chart.HasTitle = $true
    $chart.ChartTitle.Text = '%s'
  }
  Write-Output "OK"
} catch {
  Write-Output "ERROR: $_"
}
`, req.Workbook, req.Workbook, req.Sheet, req.Sheet, chartCode, req.Range, title, title)

	out, err := runPowerShellCOM(ctx, script)
	if err != nil {
		json200(w, map[string]any{"success": false, "message": err.Error()})
		return
	}
	if out == "OK" {
		json200(w, map[string]any{
			"success": true,
			"message": msgT(fmt.Sprintf("%s 차트 (%s 범위) 추가 완료", req.ChartType, req.Range),
				fmt.Sprintf("%s chart added (range %s)", req.ChartType, req.Range), lang),
		})
	} else {
		json200(w, map[string]any{"success": false, "message": out})
	}
}

// POST /api/excel/com/save — {workbook} 저장
func handleExcelComSave(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct{ Workbook string `json:"workbook"` }
	tryDecodeBody(r, &req)
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	script := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
try {
  $excel = [Runtime.InteropServices.Marshal]::GetActiveObject('Excel.Application')
  $wb = if ('%s' -ne '') { $excel.Workbooks | Where-Object { $_.Name -like '*%s*' } | Select-Object -First 1 } else { $excel.ActiveWorkbook }
  $wb.Save()
  Write-Output "OK"
} catch { Write-Output "ERROR: $_" }
`, req.Workbook, req.Workbook)

	out, _ := runPowerShellCOM(ctx, script)
	if out == "OK" {
		json200(w, map[string]any{"success": true, "message": msgT("저장 완료 ✓", "Saved ✓", lang)})
	} else {
		json200(w, map[string]any{"success": false, "message": out})
	}
}

// ── Word COM ──────────────────────────────────────────────────────────────

// GET /api/word/com/documents — 열려있는 Word 문서 목록
func handleWordComDocs(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	script := `
$ErrorActionPreference = 'SilentlyContinue'
try {
  $word = [Runtime.InteropServices.Marshal]::GetActiveObject('Word.Application')
  $docs = @()
  foreach ($doc in $word.Documents) {
    $docs += [PSCustomObject]@{
      name = $doc.Name
      path = $doc.FullName
      page_count = $doc.ComputeStatistics(2)
      word_count = $doc.ComputeStatistics(0)
    }
  }
  $docs | ConvertTo-Json -Compress -Depth 3
} catch { '[]' }
`
	out, err := runPowerShellCOM(ctx, script)
	if err != nil || out == "" || out == "[]" {
		json200(w, map[string]any{"success": true, "documents": []any{}, "message": "열려있는 Word 문서 없음"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"success":true,"documents":%s}`, out)
}

// POST /api/word/com/replace — {document, find, replace, all}
func handleWordComReplace(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		Document string `json:"document"`
		Find     string `json:"find"`
		Replace  string `json:"replace"`
		All      bool   `json:"all"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Find == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "find required"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	find := strings.ReplaceAll(req.Find, "'", "''")
	replace := strings.ReplaceAll(req.Replace, "'", "''")
	replaceMode := "2" // wdReplaceAll
	if !req.All {
		replaceMode = "1" // wdReplaceOne
	}

	script := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
try {
  $word = [Runtime.InteropServices.Marshal]::GetActiveObject('Word.Application')
  $doc = if ('%s' -ne '') { $word.Documents | Where-Object { $_.Name -like '*%s*' } | Select-Object -First 1 } else { $word.ActiveDocument }
  $find = $doc.Content.Find
  $find.Text = '%s'
  $find.Replacement.Text = '%s'
  $find.Forward = $true
  $count = 0
  while ($find.Execute([ref] '%s', [ref] $false, [ref] $false, [ref] $false, [ref] $false, [ref] $false, [ref] $true, [ref] 1, [ref] $false, [ref] '%s', [ref] %s)) {
    $count++
    if (-not %s) { break }
  }
  Write-Output "OK:$count"
} catch { Write-Output "ERROR: $_" }
`, req.Document, req.Document, find, replace, find, replace, replaceMode, fmt.Sprintf("$%v", req.All))

	out, _ := runPowerShellCOM(ctx, script)
	if strings.HasPrefix(out, "OK:") {
		count := strings.TrimPrefix(out, "OK:")
		json200(w, map[string]any{
			"success": true, "count": count,
			"message": msgT(fmt.Sprintf("'%s' → '%s' 치환 %s건 완료", req.Find, req.Replace, count),
				fmt.Sprintf("Replaced '%s' → '%s' (%s occurrences)", req.Find, req.Replace, count), lang),
		})
	} else {
		json200(w, map[string]any{"success": false, "message": out})
	}
}

// POST /api/word/com/insert — {document, text, where}
// where: end|start|cursor
func handleWordComInsert(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		Document string `json:"document"`
		Text     string `json:"text"`
		Where    string `json:"where"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Text == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "text required"})
		return
	}
	if req.Where == "" {
		req.Where = "cursor"
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	text := strings.ReplaceAll(req.Text, "'", "''")
	var rangeCode string
	switch req.Where {
	case "start": rangeCode = "$doc.Content.InsertBefore('" + text + "')"
	case "end":   rangeCode = "$doc.Content.InsertAfter('" + text + "')"
	default:      rangeCode = "$word.Selection.TypeText('" + text + "')"
	}

	script := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
try {
  $word = [Runtime.InteropServices.Marshal]::GetActiveObject('Word.Application')
  $doc = if ('%s' -ne '') { $word.Documents | Where-Object { $_.Name -like '*%s*' } | Select-Object -First 1 } else { $word.ActiveDocument }
  %s
  Write-Output "OK"
} catch { Write-Output "ERROR: $_" }
`, req.Document, req.Document, rangeCode)

	out, _ := runPowerShellCOM(ctx, script)
	if out == "OK" {
		json200(w, map[string]any{"success": true, "message": msgT(fmt.Sprintf("Word에 '%s' 삽입 완료", req.Text[:min(30, len(req.Text))]), "Inserted into Word", lang)})
	} else {
		json200(w, map[string]any{"success": false, "message": out})
	}
}

// Note: min() helper already defined in handlers_vision.go
