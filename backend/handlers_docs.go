//go:build windows

package main

import (
	"archive/zip"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// ──────────────────────────────────────────
// 문서 텍스트 추출
// ──────────────────────────────────────────

func extractDocumentText(path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".txt", ".md", ".csv", ".log", ".ini", ".cfg":
		data, err := os.ReadFile(path)
		return string(data), err
	case ".docx", ".dotx":
		return extractDocxText(path)
	case ".xlsx", ".xlsm":
		return extractXlsxText(path)
	case ".pdf":
		return extractPdfText(path)
	default:
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		// 바이너리 파일 감지
		text := string(data)
		if !isPrintable(text) {
			return fmt.Sprintf("[바이너리 파일: %s - 텍스트 추출 불가]", filepath.Base(path)), nil
		}
		return text, nil
	}
}

func isPrintable(s string) bool {
	nonPrint := 0
	for i, r := range s {
		if i > 1000 {
			break
		}
		if !unicode.IsPrint(r) && r != '\n' && r != '\r' && r != '\t' {
			nonPrint++
		}
	}
	return nonPrint < 50
}

// DOCX: ZIP 안의 word/document.xml 파싱
func extractDocxText(path string) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", fmt.Errorf("DOCX 열기 실패: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			defer rc.Close()
			data, err := io.ReadAll(rc)
			if err != nil {
				return "", err
			}
			return stripXMLTags(string(data)), nil
		}
	}
	return "", fmt.Errorf("DOCX에서 document.xml을 찾을 수 없어요")
}

// XLSX: ZIP 안의 공유 문자열 + 시트 데이터 파싱
func extractXlsxText(path string) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", fmt.Errorf("XLSX 열기 실패: %w", err)
	}
	defer r.Close()

	// 1) 공유 문자열 테이블 로드
	sharedStrings := []string{}
	for _, f := range r.File {
		if f.Name == "xl/sharedStrings.xml" {
			rc, _ := f.Open()
			data, _ := io.ReadAll(rc)
			rc.Close()
			sharedStrings = parseSharedStrings(data)
			break
		}
	}

	// 2) 모든 시트 텍스트 수집
	var sb strings.Builder
	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "xl/worksheets/sheet") && strings.HasSuffix(f.Name, ".xml") {
			rc, err := f.Open()
			if err != nil {
				continue
			}
			data, _ := io.ReadAll(rc)
			rc.Close()
			sb.WriteString(parseXlsxSheet(data, sharedStrings))
			sb.WriteString("\n")
		}
	}
	return sb.String(), nil
}

// XML 공유 문자열 파싱
type sst struct {
	Si []struct {
		T  []string `xml:"t"`
		R  []struct{ T string `xml:"t"` } `xml:"r"`
	} `xml:"si"`
}

func parseSharedStrings(data []byte) []string {
	var s sst
	xml.Unmarshal(data, &s)
	result := make([]string, len(s.Si))
	for i, si := range s.Si {
		if len(si.T) > 0 {
			result[i] = strings.Join(si.T, "")
		} else {
			var parts []string
			for _, r := range si.R {
				parts = append(parts, r.T)
			}
			result[i] = strings.Join(parts, "")
		}
	}
	return result
}

// XLSX 시트 데이터에서 셀 값 추출
type worksheet struct {
	SheetData struct {
		Row []struct {
			C []struct {
				T string `xml:"t,attr"` // 타입: s=shared, n=number
				V string `xml:"v"`
			} `xml:"c"`
		} `xml:"row"`
	} `xml:"sheetData"`
}

func parseXlsxSheet(data []byte, shared []string) string {
	var ws worksheet
	xml.Unmarshal(data, &ws)

	var sb strings.Builder
	for _, row := range ws.SheetData.Row {
		var cells []string
		for _, c := range row.C {
			val := c.V
			if c.T == "s" {
				idx, err := strconv.Atoi(val)
				if err == nil && idx < len(shared) {
					val = shared[idx]
				}
			}
			cells = append(cells, val)
		}
		sb.WriteString(strings.Join(cells, "\t"))
		sb.WriteString("\n")
	}
	return sb.String()
}

// XML 태그 제거
func stripXMLTags(s string) string {
	// <w:t> 태그 내 텍스트만 추출
	re := regexp.MustCompile(`<w:t[^>]*>([^<]*)</w:t>`)
	matches := re.FindAllStringSubmatch(s, -1)
	var parts []string
	for _, m := range matches {
		if m[1] != "" {
			parts = append(parts, m[1])
		}
	}
	text := strings.Join(parts, " ")
	// 연속 공백 정리
	spaceRe := regexp.MustCompile(`\s{2,}`)
	return spaceRe.ReplaceAllString(strings.TrimSpace(text), " ")
}

// PDF 텍스트 추출 (Windows PowerShell / pdftotext)
func extractPdfText(path string) (string, error) {
	// 방법1: pdftotext (poppler) 가 설치된 경우
	out, err := exec.Command("pdftotext", "-layout", path, "-").Output()
	if err == nil {
		return strings.TrimSpace(string(out)), nil
	}

	// 방법2: PowerShell + Windows Built-in PDF API
	script := fmt.Sprintf(`
try {
    Add-Type -AssemblyName PresentationCore
    $doc = [System.Windows.Xps.Packaging.XpsDocument]::new('%s',[System.IO.FileAccess]::Read)
    $seq = $doc.GetFixedDocumentSequence()
    $reader = [System.Windows.Documents.DocumentPaginator]
    $doc.Close()
} catch {}
# Fallback: raw text extraction
$raw = [System.IO.File]::ReadAllBytes('%s')
$text = [System.Text.Encoding]::Default.GetString($raw)
$text -replace '[^\x20-\x7E\r\n]','' | Select-String -Pattern '\S' | ForEach-Object {$_.Line} | Select-Object -First 200
`, path, path)

	psOut, psErr := exec.Command("powershell", "-NoProfile", "-Command", script).Output()
	if psErr == nil && len(psOut) > 50 {
		return strings.TrimSpace(string(psOut)), nil
	}

	// 방법3: Raw PDF 바이트에서 텍스트 객체 추출
	return extractPdfRaw(path)
}

func extractPdfRaw(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	// PDF 텍스트 객체 패턴: (text) Tj / [(text)] TJ
	re := regexp.MustCompile(`\(([^\)\\]{1,200})\)\s*Tj`)
	matches := re.FindAllStringSubmatch(string(data), -1)
	var parts []string
	for _, m := range matches {
		cleaned := strings.Map(func(r rune) rune {
			if r >= 32 && r < 127 {
				return r
			}
			return -1
		}, m[1])
		if len(cleaned) > 1 {
			parts = append(parts, cleaned)
		}
	}

	if len(parts) == 0 {
		return fmt.Sprintf("[PDF: %s — 텍스트 추출 불가. 이미지 기반 PDF이거나 암호화된 파일입니다.]", filepath.Base(path)), nil
	}
	return strings.Join(parts, " "), nil
}

// ──────────────────────────────────────────
// 간단 Diff 알고리즘 (Myers-like)
// ──────────────────────────────────────────

type DiffLine struct {
	Type string `json:"type"` // equal | added | removed | changed
	Old  string `json:"old,omitempty"`
	New  string `json:"new,omitempty"`
	Line int    `json:"line"`
}

func diffLines(text1, text2 string) []DiffLine {
	lines1 := strings.Split(strings.TrimSpace(text1), "\n")
	lines2 := strings.Split(strings.TrimSpace(text2), "\n")

	// LCS 기반 diff
	m, n := len(lines1), len(lines2)
	if m > 500 {
		lines1 = lines1[:500]
		m = 500
	}
	if n > 500 {
		lines2 = lines2[:500]
		n = 500
	}

	// dp[i][j] = LCS length
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if strings.TrimSpace(lines1[i-1]) == strings.TrimSpace(lines2[j-1]) {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] >= dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}

	// 역추적
	var result []DiffLine
	i, j := m, n
	lineNum := 1
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && strings.TrimSpace(lines1[i-1]) == strings.TrimSpace(lines2[j-1]) {
			result = append([]DiffLine{{Type: "equal", Old: lines1[i-1], New: lines2[j-1], Line: lineNum}}, result...)
			i--
			j--
		} else if j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]) {
			result = append([]DiffLine{{Type: "added", New: lines2[j-1], Line: lineNum}}, result...)
			j--
		} else {
			result = append([]DiffLine{{Type: "removed", Old: lines1[i-1], Line: lineNum}}, result...)
			i--
		}
		lineNum++
	}
	return result
}

// ──────────────────────────────────────────
// 숫자·금액 불일치 탐지
// ──────────────────────────────────────────

type NumberMismatch struct {
	Context string  `json:"context"`
	OldVal  string  `json:"old_val"`
	NewVal  string  `json:"new_val"`
	Change  float64 `json:"change_pct"`
}

var numberRe = regexp.MustCompile(`[\d,]+\.?\d*`)

func findNumberMismatches(text1, text2 string) []NumberMismatch {
	lines1 := strings.Split(text1, "\n")
	lines2 := strings.Split(text2, "\n")
	var mismatches []NumberMismatch

	maxLines := len(lines1)
	if len(lines2) < maxLines {
		maxLines = len(lines2)
	}
	if maxLines > 200 {
		maxLines = 200
	}

	for i := 0; i < maxLines; i++ {
		nums1 := numberRe.FindAllString(lines1[i], -1)
		nums2 := numberRe.FindAllString(lines2[i], -1)
		for j := 0; j < len(nums1) && j < len(nums2); j++ {
			n1 := parseNumber(nums1[j])
			n2 := parseNumber(nums2[j])
			if n1 != n2 && n1 > 0 {
				changePct := math.Abs(n2-n1) / n1 * 100
				ctx := truncate(lines1[i], 60)
				mismatches = append(mismatches, NumberMismatch{
					Context: ctx,
					OldVal:  nums1[j],
					NewVal:  nums2[j],
					Change:  math.Round(changePct*10) / 10,
				})
			}
		}
	}
	return mismatches
}

func parseNumber(s string) float64 {
	cleaned := strings.ReplaceAll(s, ",", "")
	f, _ := strconv.ParseFloat(cleaned, 64)
	return f
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}

// ──────────────────────────────────────────
// 문서 비교 핸들러
// ──────────────────────────────────────────

type DocCompareResult struct {
	File1Name       string           `json:"file1_name"`
	File2Name       string           `json:"file2_name"`
	File1Size       string           `json:"file1_size"`
	File2Size       string           `json:"file2_size"`
	SimilarityPct   int              `json:"similarity_pct"`
	AddedCount      int              `json:"added_count"`
	RemovedCount    int              `json:"removed_count"`
	ChangedCount    int              `json:"changed_count"`
	Diff            []DiffLine       `json:"diff"`
	NumberMismatches []NumberMismatch `json:"number_mismatches"`
	Summary         string           `json:"summary"`
}

func handleDocCompare(w http.ResponseWriter, r *http.Request) {
	var req struct {
		File1 string `json:"file1"`
		File2 string `json:"file2"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.File1 == "" || req.File2 == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "두 파일 경로를 모두 입력해주세요"})
		return
	}

	// 파일 존재 확인
	info1, err1 := os.Stat(req.File1)
	info2, err2 := os.Stat(req.File2)
	if err1 != nil || err2 != nil {
		writeJSON(w, 404, map[string]any{"success": false,
			"message": fmt.Sprintf("파일을 찾을 수 없어요: %s / %s", req.File1, req.File2)})
		return
	}

	// 텍스트 추출
	text1, e1 := extractDocumentText(req.File1)
	text2, e2 := extractDocumentText(req.File2)
	if e1 != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": "파일1 읽기 실패: " + e1.Error()})
		return
	}
	if e2 != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": "파일2 읽기 실패: " + e2.Error()})
		return
	}

	// Diff 계산
	diffs := diffLines(text1, text2)
	added, removed := 0, 0
	for _, d := range diffs {
		switch d.Type {
		case "added":
			added++
		case "removed":
			removed++
		}
	}

	// 유사도
	total := len(diffs)
	equal := total - added - removed
	similarity := 100
	if total > 0 {
		similarity = int(float64(equal) / float64(total) * 100)
	}

	// 숫자 불일치
	numMismatches := findNumberMismatches(text1, text2)

	// 요약
	summary := fmt.Sprintf(
		"두 문서의 유사도는 %d%%입니다. 추가 %d줄, 삭제 %d줄, 숫자 불일치 %d건 발견.",
		similarity, added, removed, len(numMismatches),
	)

	// diff 결과 필터링 (equal은 너무 많으면 제외)
	var filteredDiff []DiffLine
	for _, d := range diffs {
		if d.Type != "equal" {
			filteredDiff = append(filteredDiff, d)
		}
	}
	if len(filteredDiff) > 50 {
		filteredDiff = filteredDiff[:50]
	}

	json200(w, DocCompareResult{
		File1Name:        filepath.Base(req.File1),
		File2Name:        filepath.Base(req.File2),
		File1Size:        formatBytes(info1.Size()),
		File2Size:        formatBytes(info2.Size()),
		SimilarityPct:    similarity,
		AddedCount:       added,
		RemovedCount:     removed,
		ChangedCount:     len(numMismatches),
		Diff:             filteredDiff,
		NumberMismatches: numMismatches,
		Summary:          summary,
	})
}

// ──────────────────────────────────────────
// 파일 찾기 (이름 + 내용 기반)
// ──────────────────────────────────────────

type DocFindResult struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	SizeMB  float64 `json:"size_mb"`
	ModTime string `json:"mod_time"`
	Match   string `json:"match"` // filename | content | both
	Snippet string `json:"snippet,omitempty"`
}

func handleDocFind(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query    string `json:"query"`
		FileType string `json:"file_type"` // pdf|docx|xlsx|any
		MaxDays  int    `json:"max_days"`
		Folder   string `json:"folder"`
		MaxItems int    `json:"max_items"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.Query == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "검색어를 입력해주세요"})
		return
	}
	if req.MaxDays == 0 {
		req.MaxDays = 90
	}
	if req.MaxItems == 0 {
		req.MaxItems = 20
	}
	if req.Folder == "" {
		req.Folder, _ = os.UserHomeDir()
	}

	docExts := map[string]bool{
		".pdf": true, ".docx": true, ".doc": true,
		".xlsx": true, ".xls": true, ".pptx": true,
		".hwp": true, ".txt": true, ".md": true, ".csv": true,
	}
	if req.FileType != "" && req.FileType != "any" {
		docExts = map[string]bool{"." + strings.TrimPrefix(req.FileType, "."): true}
	}

	queryLow := strings.ToLower(req.Query)
	var results []DocFindResult

	filepath.Walk(req.Folder, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || len(results) >= req.MaxItems {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(p))
		if !docExts[ext] {
			return nil
		}

		nameLow := strings.ToLower(info.Name())
		matchType := ""
		snippet := ""

		if strings.Contains(nameLow, queryLow) {
			matchType = "filename"
		}

		// 내용 검색 (작은 파일만)
		if info.Size() < 10<<20 { // 10MB 이하
			text, err := extractDocumentText(p)
			if err == nil {
				textLow := strings.ToLower(text)
				idx := strings.Index(textLow, queryLow)
				if idx >= 0 {
					if matchType == "filename" {
						matchType = "both"
					} else {
						matchType = "content"
					}
					start := idx - 30
					if start < 0 {
						start = 0
					}
					end := idx + len(queryLow) + 60
					if end > len(text) {
						end = len(text)
					}
					snippet = "..." + text[start:end] + "..."
				}
			}
		}

		if matchType != "" {
			results = append(results, DocFindResult{
				Name:    info.Name(),
				Path:    p,
				SizeMB:  float64(info.Size()) / (1 << 20),
				ModTime: info.ModTime().Format("2006-01-02 15:04"),
				Match:   matchType,
				Snippet: snippet,
			})
		}
		return nil
	})

	json200(w, map[string]any{
		"results": results,
		"total":   len(results),
		"message": fmt.Sprintf("'%s' 검색 결과 %d개", req.Query, len(results)),
	})
}
