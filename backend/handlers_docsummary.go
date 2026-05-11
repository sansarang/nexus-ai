//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ──────────────────────────────────────────
// 문서 요약 결과 타입
// ──────────────────────────────────────────

type DocSummaryResult struct {
	FileName    string   `json:"file_name"`
	FilePath    string   `json:"file_path"`
	FileSize    string   `json:"file_size"`
	PageCount   int      `json:"page_count"`
	WordCount   int      `json:"word_count"`
	KeyPoints   []string `json:"key_points"`
	KeyNumbers  []string `json:"key_numbers"`
	Dates       []string `json:"dates"`
	Summary     string   `json:"summary"`
	Language    string   `json:"language"` // ko | en | mixed
	Category    string   `json:"category"` // contract | report | invoice | other
}

// ──────────────────────────────────────────
// 문서 요약 핸들러
// ──────────────────────────────────────────

func handleDocSummary(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FilePath   string `json:"file_path"`
		UseAI      bool   `json:"use_ai"`    // Gemini Flash 사용 여부
		GeminiKey  string `json:"gemini_key"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.FilePath == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "파일 경로를 입력해주세요"})
		return
	}

	info, err := os.Stat(req.FilePath)
	if err != nil {
		writeJSON(w, 404, map[string]any{"success": false, "message": "파일을 찾을 수 없어요: " + req.FilePath})
		return
	}

	// 텍스트 추출 (handlers_docs.go의 extractDocumentText 재사용)
	text, err := extractDocumentText(req.FilePath)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": "파일 읽기 실패: " + err.Error()})
		return
	}

	// 로컬 분석
	result := DocSummaryResult{
		FileName:   filepath.Base(req.FilePath),
		FilePath:   req.FilePath,
		FileSize:   formatBytes(info.Size()),
		WordCount:  countWords(text),
		KeyNumbers: extractKeyNumbers(text),
		Dates:      extractDates(text),
		Category:   classifyDocument(text, filepath.Base(req.FilePath)),
		Language:   detectLanguage(text),
	}

	// 핵심 내용 추출 (로컬)
	result.KeyPoints = extractKeyPoints(text, result.Category)

	// 요약 생성 (로컬 룰 기반)
	result.Summary = buildDocSummary(result)

	json200(w, result)
}

// ──────────────────────────────────────────
// 로컬 분석 함수들
// ──────────────────────────────────────────

func countWords(text string) int {
	fields := strings.Fields(text)
	return len(fields)
}

var numberWithContextRe = regexp.MustCompile(`(?:금액|금|원|달러|\$|￦|,)?\s*[\d,]+(?:\.\d+)?\s*(?:원|달러|만|억|백만|천만|%|개월|년|일|명|건)?`)
var dateRe = regexp.MustCompile(`\d{4}[-./년]\s*\d{1,2}[-./월]\s*\d{1,2}일?`)

func extractKeyNumbers(text string) []string {
	matches := numberWithContextRe.FindAllString(text, 20)
	seen := map[string]bool{}
	var result []string
	for _, m := range matches {
		m = strings.TrimSpace(m)
		if len(m) > 2 && !seen[m] {
			seen[m] = true
			result = append(result, m)
		}
	}
	if len(result) > 8 {
		result = result[:8]
	}
	return result
}

func extractDates(text string) []string {
	matches := dateRe.FindAllString(text, 10)
	seen := map[string]bool{}
	var result []string
	for _, m := range matches {
		m = strings.TrimSpace(m)
		if !seen[m] {
			seen[m] = true
			result = append(result, m)
		}
	}
	return result
}

func detectLanguage(text string) string {
	korCount := 0
	engCount := 0
	for _, r := range text {
		if r >= 0xAC00 && r <= 0xD7A3 {
			korCount++
		} else if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			engCount++
		}
	}
	if korCount > engCount {
		return "ko"
	}
	if engCount > korCount*3 {
		return "en"
	}
	return "mixed"
}

func classifyDocument(text, filename string) string {
	lower := strings.ToLower(text + " " + filename)
	switch {
	case containsAny(lower, "계약서", "contract", "갑", "을", "계약", "약정"):
		return "contract"
	case containsAny(lower, "청구서", "invoice", "세금계산서", "거래명세서", "합계금액"):
		return "invoice"
	case containsAny(lower, "보고서", "report", "분석", "결과", "현황"):
		return "report"
	case containsAny(lower, "제안서", "proposal", "견적", "제안"):
		return "proposal"
	case containsAny(lower, "회의록", "minutes", "안건", "결의"):
		return "minutes"
	default:
		return "document"
	}
}

func containsAny(s string, keywords ...string) bool {
	for _, k := range keywords {
		if strings.Contains(s, k) {
			return true
		}
	}
	return false
}

func extractKeyPoints(text, category string) []string {
	lines := strings.Split(text, "\n")
	var points []string

	importantPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(목적|purpose|조항|article|제\d+조)`),
		regexp.MustCompile(`(?i)(금액|amount|payment|지급|대금)`),
		regexp.MustCompile(`(?i)(기간|period|duration|유효|만료)`),
		regexp.MustCompile(`(?i)(의무|obligation|권리|right|책임|liability)`),
		regexp.MustCompile(`(?i)(특이사항|주의|중요|important|notice|note)`),
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) < 10 || len(line) > 200 {
			continue
		}
		for _, pat := range importantPatterns {
			if pat.MatchString(line) {
				points = append(points, line)
				break
			}
		}
		if len(points) >= 6 {
			break
		}
	}

	if len(points) == 0 {
		// 가장 긴 라인들을 핵심 포인트로
		type scored struct{ line string; score int }
		var scored_lines []scored
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if len(line) > 20 && len(line) < 150 {
				scored_lines = append(scored_lines, scored{line, len(line)})
			}
		}
		sort.Slice(scored_lines, func(i, j int) bool { return scored_lines[i].score > scored_lines[j].score })
		for i, s := range scored_lines {
			if i >= 4 {
				break
			}
			points = append(points, s.line)
		}
	}

	return points
}

func buildDocSummary(r DocSummaryResult) string {
	catLabel := map[string]string{
		"contract": "계약서", "invoice": "청구서/세금계산서",
		"report": "보고서", "proposal": "제안서",
		"minutes": "회의록", "document": "문서",
	}[r.Category]

	var parts []string
	parts = append(parts, fmt.Sprintf("📄 %s 형식의 %s입니다.", catLabel, r.FileName))
	parts = append(parts, fmt.Sprintf("총 %d 단어로 구성돼 있어요.", r.WordCount))
	if len(r.KeyNumbers) > 0 {
		parts = append(parts, fmt.Sprintf("주요 수치: %s", strings.Join(r.KeyNumbers[:min2(3, len(r.KeyNumbers))], " / ")))
	}
	if len(r.Dates) > 0 {
		parts = append(parts, fmt.Sprintf("주요 날짜: %s", strings.Join(r.Dates[:min2(2, len(r.Dates))], ", ")))
	}
	return strings.Join(parts, " ")
}

func min2(a, b int) int {
	if a < b { return a }
	return b
}

// ──────────────────────────────────────────
// 비교 리포트 내보내기 (Word/HTML)
// ──────────────────────────────────────────

func handleDocExportReport(w http.ResponseWriter, r *http.Request) {
	var req struct {
		File1      string `json:"file1"`
		File2      string `json:"file2"`
		Format     string `json:"format"` // html | txt
		OutputPath string `json:"output_path"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.File1 == "" || req.File2 == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "두 파일 경로가 필요해요"})
		return
	}
	if req.Format == "" {
		req.Format = "html"
	}

	// 두 파일 텍스트 추출
	text1, _ := extractDocumentText(req.File1)
	text2, _ := extractDocumentText(req.File2)
	diffs := diffLines(text1, text2)
	numMismatches := findNumberMismatches(text1, text2)

	added, removed := 0, 0
	for _, d := range diffs {
		if d.Type == "added" { added++ }
		if d.Type == "removed" { removed++ }
	}
	total := len(diffs)
	similarity := 100
	if total > 0 { similarity = (total - added - removed) * 100 / total }

	// HTML 리포트 생성
	html := buildCompareReportHTML(
		filepath.Base(req.File1), filepath.Base(req.File2),
		similarity, added, removed, diffs, numMismatches,
	)

	// 저장
	desktop, _ := os.UserHomeDir()
	filename := fmt.Sprintf("비교리포트_%s.html", time.Now().Format("20060102_150405"))
	if req.OutputPath != "" {
		filename = req.OutputPath
	} else {
		filename = filepath.Join(desktop, "Desktop", filename)
	}
	os.WriteFile(filename, []byte(html), 0644)
	exec.Command("explorer", filename).Start()

	json200(w, map[string]any{
		"success":  true,
		"path":     filename,
		"message":  fmt.Sprintf("비교 리포트가 저장됐어요: %s", filepath.Base(filename)),
	})
}

func buildCompareReportHTML(f1, f2 string, sim, added, removed int, diffs []DiffLine, nums []NumberMismatch) string {
	simColor := "#48bb78"
	if sim < 70 { simColor = "#ed8936" }
	if sim < 50 { simColor = "#fc8181" }

	var diffRows strings.Builder
	for _, d := range diffs {
		if diffRows.Len() > 20000 { break } // 크기 제한
		switch d.Type {
		case "added":
			diffRows.WriteString(fmt.Sprintf(`<tr style="background:rgba(72,187,120,0.15)"><td style="color:#68d391;font-weight:bold">+</td><td></td><td style="color:#c6f6d5">%s</td></tr>`, escapeHTML(d.New)))
		case "removed":
			diffRows.WriteString(fmt.Sprintf(`<tr style="background:rgba(252,129,129,0.15)"><td style="color:#fc8181;font-weight:bold">–</td><td style="color:#fed7d7;text-decoration:line-through">%s</td><td></td></tr>`, escapeHTML(d.Old)))
		}
	}

	var numRows strings.Builder
	for _, n := range nums {
		numRows.WriteString(fmt.Sprintf(`<tr><td>%s</td><td style="color:#fc8181">%s</td><td style="color:#68d391">%s</td><td>%.1f%%</td></tr>`,
			escapeHTML(n.Context), n.OldVal, n.NewVal, mismatchChangePct(n)))
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="ko"><head><meta charset="UTF-8"><title>Nexus 문서 비교 리포트</title>
<style>
body{font-family:'Malgun Gothic',sans-serif;background:#0f0f1a;color:#e2e8f0;padding:24px;margin:0}
.card{background:#1a1a2e;border:1px solid #2d2d4e;border-radius:12px;padding:20px;margin:16px 0}
table{width:100%%;border-collapse:collapse}th,td{padding:8px 12px;border-bottom:1px solid #2d2d4e;text-align:left;font-size:13px}
th{color:#90cdf4;background:#1e2040}
.badge{display:inline-block;padding:2px 10px;border-radius:6px;font-size:12px;font-weight:600;margin:2px}
</style></head><body>
<h1 style="color:#90cdf4">📊 Nexus 문서 비교 리포트</h1>
<p style="color:#718096">생성: %s</p>

<div class="card">
<h2>비교 파일</h2>
<table><tr><th>파일 A</th><th>파일 B</th></tr>
<tr><td>📘 %s</td><td>📗 %s</td></tr></table>
</div>

<div class="card">
<h2>요약</h2>
<p>유사도: <strong style="color:%s;font-size:24px">%d%%</strong></p>
<span class="badge" style="background:rgba(72,187,120,0.3)">추가 %d줄</span>
<span class="badge" style="background:rgba(252,129,129,0.3)">삭제 %d줄</span>
<span class="badge" style="background:rgba(237,137,54,0.3)">숫자 불일치 %d건</span>
</div>

<div class="card">
<h2>변경 내용</h2>
<table><tr><th>구분</th><th>파일 A (이전)</th><th>파일 B (이후)</th></tr>%s</table>
</div>

<div class="card">
<h2>숫자·금액 불일치</h2>
%s
</div>

<div style="text-align:center;color:#4a5568;font-size:12px;margin-top:30px">자동 생성: Nexus AI 비서</div>
</body></html>`,
		time.Now().Format("2006-01-02 15:04:05"),
		f1, f2,
		simColor, sim,
		added, removed, len(nums),
		diffRows.String(),
		func() string {
			if len(nums) == 0 {
				return `<p style="color:#48bb78">✅ 숫자·금액 불일치 없음</p>`
			}
			return `<table><tr><th>문맥</th><th>이전 값</th><th>새 값</th><th>변화율</th></tr>` + numRows.String() + `</table>`
		}(),
	)
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func mismatchChangePct(n NumberMismatch) float64 {
	old, _ := strconv.ParseFloat(strings.ReplaceAll(n.OldVal, ",", ""), 64)
	newV, _ := strconv.ParseFloat(strings.ReplaceAll(n.NewVal, ",", ""), 64)
	if old == 0 { return 0 }
	return (newV - old) / old * 100
}
