// handlers_pdf_analyze_shared.go — PDF 읽기 + LLM 분석 (Mac/Windows 공용)
// ledongthuc/pdf 사용 (Pure Go, CGO 없음)
//
// 라우팅:
//   "이 PDF 요약해줘 /path/to/file.pdf"
//   "PDF 분석"  → 바탕화면 최근 .pdf 자동
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ledongthuc/pdf"
)

// cmdPdfAnalyze: PDF 텍스트 추출 + LLM 요약/인사이트
func cmdPdfAnalyze(params map[string]any, original, gKey, lang string) (result any, message string) {
	path, _ := params["path"].(string)
	if path == "" {
		path = extractFilePath(original)
	}
	if path == "" {
		path = findLatestPdf()
	}
	if path == "" {
		errMsg := "분석할 PDF 파일을 찾을 수 없어요. 파일 경로를 알려주거나 바탕화면에 .pdf 파일을 두세요."
		if lang == "en" {
			errMsg = "No PDF file found. Specify a path or place .pdf on Desktop."
		}
		return map[string]any{"success": false}, errMsg
	}

	info, err := os.Stat(path)
	if err != nil {
		return map[string]any{"success": false},
			fmt.Sprintf("파일 접근 실패: %s — %v", path, err)
	}

	// PDF 텍스트 추출 (Pure Go)
	text, pages, err := extractPdfTextV2(path)
	if err != nil {
		return map[string]any{"success": false},
			fmt.Sprintf("PDF 읽기 실패: %v", err)
	}
	if strings.TrimSpace(text) == "" {
		return map[string]any{"success": false},
			"PDF에서 텍스트를 추출하지 못했어요. 이미지 기반 PDF 일 수 있어요 (OCR 필요)."
	}

	// 토큰 절약: 처음 5000자 + 끝 2000자
	preview := text
	if len(preview) > 7000 {
		preview = text[:5000] + "\n[...중간 생략...]\n" + text[len(text)-2000:]
	}

	// LLM 분석
	sysPrompt := ""
	if lang == "en" {
		sysPrompt = `Analyze this PDF and output:
- Topic + document type (1 line)
- Key points (1-2 lines)
- Action item or takeaway (1 line)
Max 4 lines. No markdown.`
	} else {
		sysPrompt = `이 PDF 분석:
- 주제 + 문서 유형 (1줄)
- 핵심 포인트 (1~2줄)
- 액션/시사점 (1줄)
총 4줄 이내, "~이에요/예요" 친근체, 마크다운 X.`
	}

	insight, _, _ := callGroqWithFallback([]groqMsg{
		{Role: "system", Content: sysPrompt},
		{Role: "user", Content: fmt.Sprintf("PDF: %s\n페이지: %d\n\n본문:\n%s", filepath.Base(path), pages, preview)},
	}, 400, false)

	if insight == "" {
		insight = fmt.Sprintf("%d페이지 PDF · 추출 텍스트 %d자", pages, len([]rune(text)))
	}

	sizeKB := float64(info.Size()) / 1024
	chars := len([]rune(text))
	msg := fmt.Sprintf("📄 %s 분석\n   %d페이지 · %d자 · %.1f KB · %s\n\n%s",
		filepath.Base(path), pages, chars, sizeKB,
		info.ModTime().Format("2006-01-02 15:04"), insight)
	if lang == "en" {
		msg = fmt.Sprintf("📄 %s analysis\n   %d pages · %d chars · %.1f KB · %s\n\n%s",
			filepath.Base(path), pages, chars, sizeKB,
			info.ModTime().Format("2006-01-02 15:04"), insight)
	}

	return map[string]any{
		"success":    true,
		"fileName":   filepath.Base(path),
		"path":       path,
		"url":        "file:///" + filepath.ToSlash(path),
		"mimeType":   "application/pdf",
		"pages":      pages,
		"chars":      chars,
		"insight":    insight,
		"text_preview": preview,
		"operation":  "pdf_analyze",
		"size_bytes": info.Size(),
		"modified":   info.ModTime().Format("2006-01-02 15:04:05"),
	}, msg
}

// extractPdfTextV2: Pure Go PDF 텍스트 추출 (ledongthuc/pdf)
func extractPdfTextV2(path string) (string, int, error) {
	f, r, err := pdf.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	totalPages := r.NumPage()

	var sb strings.Builder
	for i := 1; i <= totalPages; i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		sb.WriteString(text)
		sb.WriteString("\n")
	}
	return sb.String(), totalPages, nil
}

// findLatestPdf: 바탕화면에서 가장 최근 .pdf
func findLatestPdf() string {
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
		if !strings.HasSuffix(strings.ToLower(e.Name()), ".pdf") {
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
