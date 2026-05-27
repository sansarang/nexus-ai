//go:build windows

package main

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/xuri/excelize/v2"
)

// safePS: 타임아웃이 보장된 PowerShell 실행 래퍼 (기본 20초)
func safePS(timeout time.Duration, script string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return newHiddenCmdCtx(ctx, "powershell", "-NoProfile", "-Command", script).Output()
}

// safePSRun: 출력 없는 safePS
func safePSRun(timeout time.Duration, script string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return newHiddenCmdCtx(ctx, "powershell", "-NoProfile", "-Command", script).Run()
}

// ── 멀티 액션: 출력 포맷 (Windows 빌드용) ────────────────────────
type outputFormat string

const (
	outPDF        outputFormat = "pdf"
	outExcel      outputFormat = "excel"
	outWord       outputFormat = "word"
	outPowerPoint outputFormat = "pptx"
	outMarkdown   outputFormat = "markdown"
	outTXT        outputFormat = "txt"
	outNone       outputFormat = ""
)

func detectOutputFormat(msg string) outputFormat {
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "pdf") || strings.Contains(lower, "피디에프"):
		return outPDF
	case strings.Contains(lower, "excel") || strings.Contains(lower, "엑셀") || strings.Contains(lower, "xlsx"):
		return outExcel
	case strings.Contains(lower, "word") || strings.Contains(lower, "워드") || strings.Contains(lower, "docx"):
		return outWord
	case strings.Contains(lower, "powerpoint") || strings.Contains(lower, "파워포인트") ||
		strings.Contains(lower, "pptx") || strings.Contains(lower, "프레젠테이션") ||
		strings.Contains(lower, "presentation") || strings.Contains(lower, "slides") ||
		strings.Contains(lower, "슬라이드"):
		return outPowerPoint
	case strings.Contains(lower, "마크다운") || strings.Contains(lower, "markdown") || strings.Contains(lower, ".md"):
		return outMarkdown
	case strings.Contains(lower, "txt") || strings.Contains(lower, "텍스트 파일") ||
		strings.Contains(lower, "텍스트로 저장") || strings.Contains(lower, "text file"):
		return outTXT
	case hasFileSaveVerb(msg):
		return outMarkdown
	}
	return outNone
}

func hasFileSaveVerb(msg string) bool {
	lower := strings.ToLower(msg)
	saveVerbs := []string{
		"저장", "만들어", "작성", "정리", "보고서", "리포트", "report", "save", "export",
		"파일로", "제품설명서", "설명서", "요약해서", "뽑아줘", "출력",
		"요약해줘", "요약 해줘", "정리해줘", "정리 해줘", "모아줘",
		"뉴스 정리", "뉴스요약", "뉴스 요약", "기사 정리", "기사 요약",
		"summarize", "summary", "compile", "generate report", "make a report",
	}
	for _, v := range saveVerbs {
		if strings.Contains(lower, v) {
			return true
		}
	}
	return false
}

// winWriteZip: zip 파일 생성 (docx/pptx 공통)
func winWriteZip(path string, files map[string]string) error {
	f, err := os.Create(path)
	if err != nil { return err }
	defer f.Close()
	w := zip.NewWriter(f)
	defer w.Close()
	for name, content := range files {
		fw, err := w.Create(name)
		if err != nil { return err }
		if _, err := fw.Write([]byte(content)); err != nil { return err }
	}
	return nil
}

// winSaveDocx: 실제 .docx 생성 (OOXML)
func winSaveDocx(path, title, summary string, items []map[string]string) error {
	now := time.Now().Format("2006-01-02 15:04")
	var body strings.Builder
	addPara := func(text, style string) {
		styleXML := ""
		if style != "" {
			styleXML = fmt.Sprintf(`<w:pPr><w:pStyle w:val="%s"/></w:pPr>`, style)
		}
		esc := strings.ReplaceAll(text, "&", "&amp;")
		esc = strings.ReplaceAll(esc, "<", "&lt;")
		esc = strings.ReplaceAll(esc, ">", "&gt;")
		body.WriteString(fmt.Sprintf(`<w:p>%s<w:r><w:t xml:space="preserve">%s</w:t></w:r></w:p>`, styleXML, esc))
	}
	addPara(title, "Heading1")
	addPara("Generated: "+now, "")
	if summary != "" {
		addPara("AI Summary", "Heading2")
		for _, line := range strings.Split(summary, "\n") {
			line = strings.TrimLeft(line, "#- *`")
			if strings.TrimSpace(line) == "" { continue }
			addPara(line, "")
		}
	}
	if len(items) > 0 {
		addPara("Details", "Heading2")
		for i, it := range items {
			name := it["title"]; if name == "" { name = it["name"] }
			content := it["content"]; if content == "" { content = it["snippet"] }
			url := it["url"]; if url == "" { url = it["link"] }
			price := it["price"]
			addPara(fmt.Sprintf("%d. %s", i+1, name), "Heading3")
			if price != "" { addPara("Price: "+price, "") }
			if content != "" { addPara(content, "") }
			if url != "" { addPara("Link: "+url, "") }
		}
	}
	docXML := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:body>%s<w:sectPr/></w:body></w:document>`, body.String())

	return winWriteZip(path, map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="xml" ContentType="application/xml"/>
<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
<Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>
</Types>`,
		"_rels/.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`,
		"word/document.xml": docXML,
		"word/_rels/document.xml.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
</Relationships>`,
		"word/styles.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:style w:type="paragraph" w:styleId="Heading1"><w:name w:val="heading 1"/><w:rPr><w:b/><w:sz w:val="48"/></w:rPr></w:style>
<w:style w:type="paragraph" w:styleId="Heading2"><w:name w:val="heading 2"/><w:rPr><w:b/><w:sz w:val="36"/></w:rPr></w:style>
<w:style w:type="paragraph" w:styleId="Heading3"><w:name w:val="heading 3"/><w:rPr><w:b/><w:sz w:val="28"/></w:rPr></w:style>
</w:styles>`,
	})
}

// winSavePptx: 실제 .pptx 생성 (OOXML)
func winSavePptx(path, title, summary string, items []map[string]string) error {
	cleanText := func(s string) string {
		s = strings.ReplaceAll(s, "**", "")
		s = strings.ReplaceAll(s, "__", "")
		s = strings.ReplaceAll(s, "`", "")
		s = strings.ReplaceAll(s, "&", "&amp;")
		s = strings.ReplaceAll(s, "<", "&lt;")
		s = strings.ReplaceAll(s, ">", "&gt;")
		s = strings.ReplaceAll(s, "\"", "&quot;")
		return s
	}
	textToParas := func(text, sz string) string {
		var sb strings.Builder
		for _, line := range strings.Split(text, "\n") {
			raw := strings.TrimSpace(line)
			if raw == "" {
				sb.WriteString(`<a:p><a:endParaRPr lang="en-US" dirty="0"/></a:p>`)
				continue
			}
			bold := ""
			if strings.HasPrefix(raw, "#") {
				bold = "<a:b/>"
				raw = strings.TrimLeft(raw, "# ")
			} else if strings.HasPrefix(raw, "- ") || strings.HasPrefix(raw, "• ") {
				raw = "• " + raw[2:]
			}
			sb.WriteString(fmt.Sprintf(`<a:p><a:r><a:rPr lang="en-US" sz="%s" dirty="0">%s</a:rPr><a:t>%s</a:t></a:r></a:p>`, sz, bold, cleanText(raw)))
		}
		return sb.String()
	}
	makeShape := func(id, x, y, cx, cy int, paras string) string {
		return fmt.Sprintf(
			`<p:sp><p:nvSpPr><p:cNvPr id="%d" name="sp%d"/>` +
			`<p:cNvSpPr txBox="1"><a:spLocks noGrp="1"/></p:cNvSpPr><p:nvPr/></p:nvSpPr>` +
			`<p:spPr><a:xfrm><a:off x="%d" y="%d"/><a:ext cx="%d" cy="%d"/></a:xfrm>` +
			`<a:prstGeom prst="rect"><a:avLst/></a:prstGeom><a:noFill/></p:spPr>` +
			`<p:txBody><a:bodyPr wrap="square" autofit="normAutofit"/><a:lstStyle/>%s</p:txBody></p:sp>`,
			id, id, x, y, cx, cy, paras)
	}
	makeSlide := func(heading, bodyText string) string {
		return fmt.Sprintf(
			`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<p:sld xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"` +
			` xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"` +
			` xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">` +
			`<p:cSld><p:spTree>` +
			`<p:nvGrpSpPr><p:cNvPr id="1" name=""/><p:cNvGrpSpPr/><p:nvPr/></p:nvGrpSpPr>` +
			`<p:grpSpPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="0" cy="0"/>` +
			`<a:chOff x="0" y="0"/><a:chExt cx="0" cy="0"/></a:xfrm></p:grpSpPr>` +
			`%s%s</p:spTree></p:cSld></p:sld>`,
			makeShape(2, 457200, 274638, 8229600, 1143000, textToParas(heading, "3200")),
			makeShape(3, 457200, 1600200, 8229600, 4525963, textToParas(bodyText, "1800")))
	}
	slideRels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"/>`

	files := map[string]string{}
	cleanSummary := ""
	if summary != "" {
		var lines []string
		for _, l := range strings.Split(summary, "\n") {
			l = strings.TrimSpace(l)
			if l != "" { lines = append(lines, l) }
			if len(lines) >= 12 { break }
		}
		cleanSummary = strings.Join(lines, "\n")
	}
	files["ppt/slides/slide1.xml"] = makeSlide(title, cleanSummary)
	files["ppt/slides/_rels/slide1.xml.rels"] = slideRels

	slideList := `<p:sldIdLst><p:sldId id="256" r:id="rId1"/>`
	slideContentTypes := `<Override PartName="/ppt/slides/slide1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slide+xml"/>`
	presRels := `<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="slides/slide1.xml"/>`

	for i, it := range items {
		if i >= 20 { break }
		name := it["title"]; if name == "" { name = it["name"] }
		content := it["content"]; if content == "" { content = it["snippet"] }
		price := it["price"]
		body := content
		if price != "" { body = "Price: " + price + "\n\n" + content }
		sn := i + 2
		files[fmt.Sprintf("ppt/slides/slide%d.xml", sn)] = makeSlide(fmt.Sprintf("%d. %s", i+1, name), body)
		files[fmt.Sprintf("ppt/slides/_rels/slide%d.xml.rels", sn)] = slideRels
		slideList += fmt.Sprintf(`<p:sldId id="%d" r:id="rId%d"/>`, 256+i+1, i+2)
		slideContentTypes += fmt.Sprintf(`<Override PartName="/ppt/slides/slide%d.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slide+xml"/>`, sn)
		presRels += fmt.Sprintf(`<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="slides/slide%d.xml"/>`, i+2, sn)
	}
	slideList += `</p:sldIdLst>`

	files["[Content_Types].xml"] = fmt.Sprintf(
		`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
		`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
		`<Default Extension="xml" ContentType="application/xml"/>` +
		`<Override PartName="/ppt/presentation.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"/>` +
		`%s</Types>`, slideContentTypes)
	files["_rels/.rels"] = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
		`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="ppt/presentation.xml"/>` +
		`</Relationships>`
	files["ppt/presentation.xml"] = fmt.Sprintf(
		`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<p:presentation xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"` +
		` xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"` +
		` xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">` +
		`<p:sldMasterIdLst/><p:sldSz cx="9144000" cy="6858000"/><p:notesSz cx="6858000" cy="9144000"/>%s</p:presentation>`,
		slideList)
	files["ppt/_rels/presentation.xml.rels"] = fmt.Sprintf(
		`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">%s</Relationships>`, presRels)
	return winWriteZip(path, files)
}

// winSaveHTML: PDF 대용 HTML (브라우저 Ctrl+P → PDF)
func winSaveHTML(path, title, summary string, items []map[string]string) error {
	now := time.Now().Format("2006-01-02 15:04")
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8"><title>%s</title><style>
body{font-family:Calibri,Arial,sans-serif;max-width:900px;margin:40px auto;line-height:1.7;color:#222}
h1{color:#1a1a2e;border-bottom:2px solid #4a90d9;padding-bottom:8px}
h2{color:#2c3e50;margin-top:2em}h3{color:#34495e}
.summary{background:#f0f7ff;border-left:4px solid #4a90d9;padding:16px;border-radius:4px;white-space:pre-wrap}
.item{border:1px solid #e0e0e0;border-radius:8px;padding:16px;margin:12px 0}
.price{color:#e74c3c;font-weight:bold}a{color:#4a90d9}.meta{color:#888;font-size:0.9em}
@media print{body{margin:0}.item{break-inside:avoid}}
</style></head><body>`, title))
	sb.WriteString(fmt.Sprintf("<h1>%s</h1><p class='meta'>Generated: %s</p>", title, now))
	if summary != "" {
		esc := strings.ReplaceAll(summary, "&", "&amp;")
		esc = strings.ReplaceAll(esc, "<", "&lt;")
		esc = strings.ReplaceAll(esc, ">", "&gt;")
		sb.WriteString("<h2>AI Summary</h2><div class='summary'>" + esc + "</div>")
	}
	if len(items) > 0 {
		sb.WriteString("<h2>Details</h2>")
		for i, it := range items {
			name := it["title"]; if name == "" { name = it["name"] }
			content := it["content"]; if content == "" { content = it["snippet"] }
			url := it["url"]; if url == "" { url = it["link"] }
			price := it["price"]
			sb.WriteString(fmt.Sprintf("<div class='item'><h3>%d. %s</h3>", i+1, name))
			if price != "" { sb.WriteString(fmt.Sprintf("<p class='price'>Price: %s</p>", price)) }
			if content != "" {
				esc := strings.ReplaceAll(content, "&", "&amp;")
				sb.WriteString(fmt.Sprintf("<p>%s</p>", esc))
			}
			if url != "" { sb.WriteString(fmt.Sprintf("<p><a href='%s' target='_blank'>View source →</a></p>", url)) }
			sb.WriteString("</div>")
		}
	}
	sb.WriteString("<p class='meta' style='margin-top:3em;border-top:1px solid #eee;padding-top:1em'>To save as PDF: Open this file in browser → Ctrl+P → Save as PDF</p></body></html>")
	return os.WriteFile(path, []byte(sb.String()), 0644)
}

func saveResultToFile(format outputFormat, title string, items []map[string]string, summary string) (string, error) {
	home, _ := os.UserHomeDir()
	ts := time.Now().Format("20060102_150405")
	safeName := strings.Map(func(r rune) rune {
		if r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r >= '가' && r <= '힣' {
			return r
		}
		return '_'
	}, title)
	if len([]rune(safeName)) > 20 {
		safeName = string([]rune(safeName)[:20])
	}

	buildMD := func() string {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("# %s\n\n", title))
		sb.WriteString(fmt.Sprintf("*생성: %s*\n\n", time.Now().Format("2006-01-02 15:04:05")))
		if summary != "" {
			sb.WriteString("## 요약\n\n" + summary + "\n\n")
		}
		if len(items) > 0 {
			sb.WriteString("## 항목\n\n")
			for i, it := range items {
				name := it["title"]
				if name == "" { name = it["name"] }
				url := it["url"]
				if url == "" { url = it["link"] }
				price := it["price"]
				if price != "" {
					sb.WriteString(fmt.Sprintf("%d. **%s** — %s\n   %s\n\n", i+1, name, price, url))
				} else {
					sb.WriteString(fmt.Sprintf("%d. [%s](%s)\n\n", i+1, name, url))
				}
			}
		}
		return sb.String()
	}

	switch format {
	case outMarkdown:
		path := filepath.Join(home, "Desktop", fmt.Sprintf("nexus_%s_%s.md", safeName, ts))
		return path, os.WriteFile(path, []byte(buildMD()), 0644)

	case outTXT:
		path := filepath.Join(home, "Desktop", fmt.Sprintf("nexus_%s_%s.txt", safeName, ts))
		var sb strings.Builder
		sb.WriteString(title + "\n" + strings.Repeat("=", 40) + "\n")
		sb.WriteString("생성: " + time.Now().Format("2006-01-02 15:04:05") + "\n\n")
		if summary != "" { sb.WriteString("[ 요약 ]\n" + summary + "\n\n") }
		if len(items) > 0 {
			sb.WriteString("[ 항목 ]\n")
			for i, it := range items {
				name := it["title"]; if name == "" { name = it["name"] }
				url := it["url"]; if url == "" { url = it["link"] }
				price := it["price"]
				if price != "" {
					sb.WriteString(fmt.Sprintf("%d. %s — %s\n   %s\n\n", i+1, name, price, url))
				} else {
					sb.WriteString(fmt.Sprintf("%d. %s\n   %s\n\n", i+1, name, url))
				}
			}
		}
		return path, os.WriteFile(path, []byte(sb.String()), 0644)

	case outExcel:
		path := filepath.Join(home, "Desktop", fmt.Sprintf("nexus_%s_%s.xlsx", safeName, ts))
		f := excelize.NewFile()
		sheet := "Results"
		f.SetSheetName("Sheet1", sheet)
		headers := []string{"No", "Title/Product", "Content", "Price", "Link"}
		for ci, h := range headers {
			cell, _ := excelize.CoordinatesToCellName(ci+1, 1)
			f.SetCellValue(sheet, cell, h)
		}
		row := 2
		if summary != "" {
			f.SetCellValue(sheet, fmt.Sprintf("A%d", row), 0)
			f.SetCellValue(sheet, fmt.Sprintf("B%d", row), "[AI Summary]")
			f.SetCellValue(sheet, fmt.Sprintf("C%d", row), summary)
			row++
		}
		for i, it := range items {
			name := it["title"]; if name == "" { name = it["name"] }
			url  := it["url"];   if url == ""  { url = it["link"] }
			price   := it["price"]
			content := it["content"]; if content == "" { content = it["snippet"] }
			f.SetCellValue(sheet, fmt.Sprintf("A%d", row), i+1)
			f.SetCellValue(sheet, fmt.Sprintf("B%d", row), name)
			f.SetCellValue(sheet, fmt.Sprintf("C%d", row), content)
			f.SetCellValue(sheet, fmt.Sprintf("D%d", row), price)
			f.SetCellValue(sheet, fmt.Sprintf("E%d", row), url)
			row++
		}
		f.SetColWidth(sheet, "B", "C", 40)
		f.SetColWidth(sheet, "E", "E", 50)
		return path, f.SaveAs(path)

	case outWord:
		path := filepath.Join(home, "Desktop", fmt.Sprintf("nexus_%s_%s.docx", safeName, ts))
		return path, winSaveDocx(path, title, summary, items)

	case outPowerPoint:
		path := filepath.Join(home, "Desktop", fmt.Sprintf("nexus_%s_%s.pptx", safeName, ts))
		return path, winSavePptx(path, title, summary, items)

	case outPDF:
		path := filepath.Join(home, "Desktop", fmt.Sprintf("nexus_%s_%s.html", safeName, ts))
		return path, winSaveHTML(path, title, summary, items)
	}
	return "", fmt.Errorf("지원하지 않는 형식")
}

// ══════════════════════════════════════════════════════════════════
//  POST /api/command
//  사용자가 어떤 자연어로 말해도 Nexus가 알아서 처리합니다.
//  LLM이 의도를 파악 → 올바른 백엔드 함수 호출 → 결과 반환
// ══════════════════════════════════════════════════════════════════




// buildNexusRoutingTools: Function Calling용 도구 목록 반환
// 핵심 액션만 포함 — LLM이 정확한 파라미터로 선택하도록 강제
func buildNexusRoutingTools() []ToolDef {
	mustParam := func(props string) json.RawMessage {
		return json.RawMessage(`{"type":"object","properties":{` + props + `},"additionalProperties":false}`)
	}
	strProp := func(desc string) string {
		return `"type":"string","description":"` + desc + `"`
	}
	return []ToolDef{
		{Type: "function", Function: ToolFunctionDef{
			Name:        "web_search",
			Description: "웹 검색: 뉴스, 날씨, 교통, 쇼핑, 맛집, 환율, 주가, 맛집, 일반 정보 검색",
			Parameters:  mustParam(`"query":{` + strProp("검색 질의어") + `},"site":{` + strProp("검색 사이트: google|naver|coupang|amazon|danawa|auto") + `},"max_items":{"type":"integer","description":"최대 결과 수 (기본 5)"}`),
		}},
		{Type: "function", Function: ToolFunctionDef{
			Name:        "video_search",
			Description: "유튜브/틱톡 영상 검색",
			Parameters:  mustParam(`"query":{` + strProp("검색어") + `},"platform":{` + strProp("youtube|tiktok") + `},"max_items":{"type":"integer"}`),
		}},
		{Type: "function", Function: ToolFunctionDef{
			Name:        "scan",
			Description: "PC 상태 진단, CPU/메모리/디스크 점검",
			Parameters:  mustParam(``),
		}},
		{Type: "function", Function: ToolFunctionDef{
			Name:        "clean",
			Description: "임시 파일 삭제, PC 정리, 디스크 최적화",
			Parameters:  mustParam(``),
		}},
		{Type: "function", Function: ToolFunctionDef{
			Name:        "security_scan",
			Description: "해킹 탐지, 악성코드 스캔, 보안 점검",
			Parameters:  mustParam(``),
		}},
		{Type: "function", Function: ToolFunctionDef{
			Name:        "launch_app",
			Description: "앱 실행 (크롬, 워드, 카카오톡 등)",
			Parameters:  mustParam(`"app_name":{` + strProp("실행할 앱 이름") + `}`),
		}},
		{Type: "function", Function: ToolFunctionDef{
			Name:        "system_control",
			Description: "볼륨 조절, 밝기 변경, 와이파이 제어, 절전/재시작/종료",
			Parameters:  mustParam(`"control":{` + strProp("volume|brightness|wifi|sleep|restart|shutdown|mute") + `},"value":{"type":"integer","description":"설정값 (볼륨/밝기: 0-100)"}`),
		}},
		{Type: "function", Function: ToolFunctionDef{
			Name:        "file_search",
			Description: "파일 검색 (이름, 키워드, 날짜로)",
			Parameters:  mustParam(`"query":{` + strProp("파일명 또는 키워드") + `},"folder":{` + strProp("검색 폴더 경로") + `}`),
		}},
		{Type: "function", Function: ToolFunctionDef{
			Name:        "note",
			Description: "메모 저장",
			Parameters:  mustParam(`"content":{` + strProp("메모할 내용") + `}`),
		}},
		{Type: "function", Function: ToolFunctionDef{
			Name:        "workflow_run",
			Description: "여러 단계 복합 작업 (검색+저장, 메일+캘린더 등 2개 이상 연결)",
			Parameters:  mustParam(`"goal":{` + strProp("사용자 복합 요청 전체 문장") + `}`),
		}},
		{Type: "function", Function: ToolFunctionDef{
			Name:        "clarify",
			Description: "실행에 필수 정보가 전혀 없어서 진행 불가능할 때만 사용 (파일명 없는 파일 찾기, 수신자 없는 이메일 전송 등)",
			Parameters: mustParam(`"question":{` + strProp("추가로 물어볼 질문") + `},"missing":{` + strProp("없는 정보 설명") + `},"intent":{` + strProp("원래 실행하려던 액션명") + `},"options":{"type":"array","items":{"type":"string"},"description":"사용자가 선택할 수 있는 2-4개 선택지 (예: [\"웹 검색\",\"뉴스\",\"쇼핑\",\"유튜브\"])"}`),
		}},
		{Type: "function", Function: ToolFunctionDef{
			Name:        "chat",
			Description: "인사, 잡담, AI 자체 질문에만 사용. 실시간 정보/검색이 필요한 모든 질문은 web_search 사용",
			Parameters:  mustParam(``),
		}},
	}
}

// Nexus가 할 수 있는 모든 일을 LLM에게 알려줍니다.
// 사용자가 어떤 말을 해도 이 중 가장 적합한 action을 고릅니다.
const nexusSystemPrompt = `당신은 Nexus AI 비서입니다. 사용자 명령을 분석하여 아래 액션 중 하나를 반드시 선택하세요.

⚠️ 규칙: 반드시 JSON만 출력하세요. 설명 금지.
형식: {"action":"액션명","params":{...}}

━━━ 액션 목록 & 트리거 키워드 ━━━

"web_search" → shopping/price/news/deals/search/transit/travel/food/restaurant/weather/exchange rate/stock/hospital/movie/show/booking/directions/route/schedule/Amazon/eBay/Google/Coupang/Naver
  예) "쿠팡에서 에어팟 찾아줘" "네이버 AI 뉴스 10개" "삼성 노트북 최저가"
  예) "부산터미널에서 인천 청라 가는 버스" "서울역 부산 KTX 시간표" "강남 맛집 추천"
  예) "오늘 달러 환율" "삼성전자 주가" "가까운 응급실 위치" "CGV 영화 시간표"
  ex) "cheapest iPhone on Amazon" "weather in New York" "flights from Seoul to Tokyo" "restaurants near me" "bitcoin price today"
  ex) "how to get from LA to San Francisco by bus" "NBA scores today" "best hotels in Paris"
  params: {"query":"search query","site":"amazon|ebay|google|coupang|naver|temu|danawa|auto","max_items":5}
  ⚠️ output 파라미터 절대 포함 금지 — 사용자가 명시적으로 "PDF로 저장", "엑셀로 저장" 이라고 할 때만 output 포함
  ⚠️ site 값은 반드시 위 목록 중 하나만 사용 (youtube.com 형식 금지, 축약형 사용)
  ⚠️ 교통/맛집/장소/실시간 정보는 무조건 web_search 사용 (chat 사용 금지)
  ⚠️ transit/food/places/real-time info → always use web_search (never chat)
  ⚠️ 유튜브/틱톡/YouTube/TikTok 관련 쿼리는 반드시 video_search 사용 (web_search 절대 금지)

"video_search" → YouTube/TikTok/유튜브/틱톡/영상/동영상/쇼츠/릴스/video/shorts/reels
  예) "유튜브에서 요리 영상 찾아줘" "틱톡에서 댄스 영상" "틱톡 유행하는 노래" "유튜브 강의 찾아줘"
  ex) "find cooking videos on YouTube" "TikTok viral dance" "YouTube tutorial for Python"
  params: {"query":"search query","platform":"youtube|tiktok","max_items":8}
  ⚠️ 유튜브/틱톡이 언급된 모든 쿼리는 무조건 이 액션 사용

"file_search" → 파일찾기/문서검색/계약서/보고서/~보낸 파일/~관련 파일
  예) "박부장이 보낸 계약서 찾아줘" "지난달 여행 사진 찾아줘" "엑셀 파일 찾아줘"
  params: {"query":"검색어","folder":"경로(없으면 홈)","max_results":10}

"doc_compare" → 두 문서 비교/변경사항/버전 비교/차이점
  예) "계약서 v1과 v2 비교해줘" "이 두 파일 다른 점 알려줘"
  params: {"file_a":"경로A","file_b":"경로B"}

"doc_summary" → 문서요약/보고서 핵심/파일 내용 분석/요약해줘
  예) "이 PDF 요약해줘" "계약서 핵심만 알려줘"
  params: {"file_path":"경로"}

"organize_folder" → 폴더정리/파일정리/바탕화면정리/다운로드정리/파일분류
  예) "다운로드 폴더 정리해" "바탕화면 깔끔하게 정리해줘"
  params: {"folder":"Downloads|Desktop|Documents","mode":"type|date|auto"}

"vision" → 화면보기/오류분석/지금 화면/창 내용/오류해결/화면 뭐라고 써있어
  예) "지금 화면에 뭐라고 써있어?" "이 오류 어떻게 고쳐?" "화면 분석해줘"
  params: {"question":"질문"}

"scan" → PC상태/건강점검/속도진단/PC 문제확인/진단해줘/PC 어때
  예) "PC 상태 알려줘" "PC 건강 체크해줘" "PC 진단해줘" "지금 PC 어때"
  params: {}

"clean" → 임시파일정리/디스크정리/느려졌어/빠르게해줘/용량확보/청소
  예) "PC 정리해줘" "임시 파일 지워줘" "디스크 청소해줘" "PC가 느려"
  params: {}

"security_scan" → 해킹탐지/악성코드/보안점검/원격접속/바이러스/수상한프로세스/침입탐지
  예) "해킹 탐지해" "해킹당했나 확인해줘" "바이러스 있어?" "악성코드 스캔해"
  예) "보안 점검해줘" "원격 접속 탐지해" "이상한 프로세스 있어?" "해킹 확인해"
  params: {}

"stats" → CPU온도/메모리/디스크용량/네트워크속도/현재 리소스 현황
  예) "CPU 온도 알려줘" "메모리 얼마나 써?" "지금 네트워크 속도 어때"
  params: {}

"focus_mode" → 집중모드/방해금지/알림차단/집중하고싶어/자동모드
  예) "집중 모드 켜줘" "방해 금지 설정해줘" "25분 집중 모드"
  params: {"enable":true}

"journal" → 업무일지/일지작성/오늘뭐했어/작업기록/일일리포트/오늘 정리
  예) "오늘 업무 일지 써줘" "오늘 업무 일지 작성해줘" "오늘 뭐 했어?" "일지 만들어줘"
  예) "오늘 작업 기록 정리해줘" "일일 리포트 만들어" "오늘 업무 정리해줘"
  params: {}

"health_report" → PC건강리포트/진단리포트/점검결과/리포트PDF/건강점수
  예) "PC 건강 리포트 만들어줘" "진단 리포트 PDF로 저장해줘"
  params: {}

"scheduler" → 매일/매주/내일/특정시간에/자동실행/반복/스케줄/예약
  예) "매주 월요일 9시에 보고서 정리해줘" "내일 아침 8시에 메일 요약해줘"
  params: {"command":"사용자 원문 전체"}

"launch_app" → 앱실행/프로그램열어/크롬열어/워드열어/카카오톡/실행해줘
  예) "크롬 열어줘" "워드 실행해줘" "카카오톡 켜줘"
  params: {"app_name":"앱이름"}

"system_control" → 볼륨/밝기/와이파이/절전/재시작/종료/음소거/꺼줘
  예) "볼륨 낮춰" "밝기 올려줘" "와이파이 꺼줘" "절전 모드로" "PC 재시작해"
  params: {"control":"volume|brightness|wifi|sleep|restart|shutdown|mute","value":50}

"excel_save" → 엑셀저장/표로정리/xlsx/스프레드시트
  예) "이 데이터 엑셀로 저장해줘" "표로 정리해줘"
  params: {"title":"제목","data":[["헤더1","헤더2"],["값1","값2"]]}

"note" → 메모/기록/적어둬/저장해줘(단순텍스트)
  예) "이거 메모해줘" "기록해줘" "적어둬"
  params: {"content":"내용"}

"workflow_run" → 2단계 이상의 복합 작업 / 여러 액션을 연결해야 할 때 / ~하고 ~해줘 / ~한 다음 ~해줘
  예) "뉴스 찾아서 엑셀로 저장해줘"
  예) "메일 온 거 읽고 요약해서 캘린더에 일정 추가해줘"
  예) "유튜브에서 요리 영상 찾아서 PDF로 정리해줘"
  예) "PC 진단하고 문제 있으면 수리해줘"
  예) "가격 비교해서 가장 싼 거 엑셀로 정리해줘"
  예) "수신 메일 분류하고 중요한 것만 요약해서 보고서로 만들어줘"
  예) "오늘 일정 확인하고 빈 시간에 미팅 잡아줘"
  ex) "find news about AI and save to Excel"
  ex) "check my email and summarize important ones"
  ex) "search YouTube for tutorials and save the list as PDF"
  ex) "scan PC then fix any issues found"
  params: {"goal":"사용자가 원하는 복합 작업 전체 문장 그대로"}
  ⚠️ "~하고 ~해줘", "~한 다음 ~해줘", "~해서 ~해줘" 패턴 → 무조건 workflow_run
  ⚠️ 단일 액션이면 workflow_run 사용 금지 (해당 액션 직접 사용)

"chat" → 오직 인사/잡담/AI 자체 질문에만 사용 (레시피·요리법·역사·과학 지식도 web_search 우선)
  예) "안녕" "고마워" "넌 누구야" "오늘 기분 어때"
  ex) "hello" "who are you" "tell me a joke"
  ⚠️ 레시피/요리법/만드는법/재료/칼로리 → 무조건 web_search (chat 사용 금지)
  ⚠️ recipe/how to cook/how to make/ingredients/calories → 무조건 web_search
  ⚠️ 역사/과학/IT 지식이라도 상세 설명이 필요하면 web_search 사용
  ⚠️ 실시간/외부 데이터가 필요한 모든 질문은 chat 금지 → web_search 사용
  ⚠️ Any question needing real-time or external data → web_search, never chat

━━━ 중요 판단 규칙 ━━━
1. "해킹" 키워드 → 무조건 security_scan
2. "업무 일지", "일지 써", "일지 작성" → 무조건 journal
3. "PC 상태", "PC 어때", "진단" → scan
4. 시간/날짜 + 자동화 키워드 → scheduler
5. 의심스러우면 chat 대신 가장 가까운 액션을 선택하세요.
6. 교통(버스/기차/지하철/KTX/고속버스/시외버스/항공편/노선/시간표/요금) → 무조건 web_search
7. 맛집/식당/카페/병원/약국/마트/장소 → 무조건 web_search
8. 환율/날씨(특정 도시)/영화/공연 → 무조건 web_search
9. "어떻게 가?" "얼마야?" "몇 시에?" 같은 실시간 정보 → 무조건 web_search
10. chat은 오직 인사/잡담/AI 자체에 대한 질문만 사용
11. "~하고 ~해줘" / "~한 다음 ~해줘" / "~해서 ~해줘" 패턴 → 무조건 workflow_run
12. 두 개 이상의 서로 다른 서비스(검색+저장, 메일+캘린더 등)를 연결하는 요청 → workflow_run
13. 주가/코인/ETF/종목 분석·전망 → 무조건 stock_analysis (web_search 금지)
14. 의학 논문/약물/임상 가이드라인 검색 → 무조건 medical_search
15. 계약서/법률 문서 검토·분석 → 무조건 contract_review
16. 유튜브/SNS/블로그 스크립트·콘텐츠 기획 작성 → 무조건 content_script
17. 판례/법령 검색 → 무조건 legal_search

"stock_analysis" → 주식/코인/ETF 종목 분석, 포트폴리오 점검, 투자 인사이트
  예) "삼성전자 주가 분석해줘" "비트코인 전망" "내 포트폴리오 점검해줘" "AAPL 재무제표"
  ex) "analyze Tesla stock" "Bitcoin price forecast" "S&P 500 outlook"
  params: {"ticker":"종목명/코드","query":"분석 내용","lang":"ko|en"}

"medical_search" → 의학 논문, 약물 정보, 임상 가이드라인, 증상 정보
  예) "메트포르민 부작용 찾아줘" "당뇨 최신 가이드라인" "COVID 치료 논문 요약"
  ex) "metformin side effects" "latest hypertension guidelines" "search pubmed for diabetes RCT"
  params: {"query":"검색어","type":"drug|paper|guideline|symptom","lang":"ko|en"}

"contract_review" → 계약서, 법률 문서, 협약서 검토 및 리스크 분석
  예) "이 계약서 독소조항 찾아줘" "근로계약서 검토해줘" "NDA 위험 조항 분석"
  ex) "review this contract for risks" "check employment agreement" "analyze NDA clauses"
  params: {"file_path":"파일경로","content":"직접 텍스트","focus":"risk|unfair|summary|all","lang":"ko|en"}

"legal_search" → 판례, 법령, 규정 검색
  예) "해고 무효 판례 찾아줘" "개인정보보호법 위반 사례" "근로기준법 52조 내용"
  ex) "Korean labor law wrongful termination precedent" "GDPR violation cases"
  params: {"query":"검색어","type":"case|law|regulation","lang":"ko|en"}

"content_script" → 유튜브/인스타/틱톡/블로그 스크립트, 썸네일, 해시태그, 제목 생성
  예) "AI 트렌드 유튜브 스크립트 써줘" "다이어트 틱톡 대본" "인스타 릴스 기획"
  ex) "write YouTube script about AI" "create TikTok script for cooking" "Instagram reel ideas"
  params: {"topic":"주제","platform":"youtube|instagram|tiktok|blog","duration":"short|medium|long","style":"educational|entertainment|vlog|review","lang":"ko|en"}

"clarify" → 의도는 파악됐지만 핵심 정보가 없어서 실행 불가능할 때만 사용
  예) "날씨 어때?" → 지역 모름
  예) "파일 찾아줘" → 무슨 파일인지 모름  
  예) "뉴스 알려줘" → 어떤 주제인지 모름
  params: {"question":"친절한 추가 질문","missing":"없는 정보","intent":"원래 액션명","collected":{...지금까지 파악된 파라미터...}}

━━━ CLARIFY 원칙 (2026 BEST-EFFORT MODE) ━━━
✅ 기본 원칙: 최선의 추론으로 즉시 실행. clarify는 최후의 수단.
✅ 부분 정보라도 있으면 추론해서 실행하고, 결과와 함께 "더 구체적으로 원하시면 말씀해 주세요" 안내.

🔴 clarify가 반드시 필요한 경우 (실행 자체가 불가능할 때만)
- file_search/doc_summary: 파일 힌트가 전혀 없을 때 (이름·키워드·날짜 모두 없음)
  예) "파일 찾아줘" → "파일 이름이나 키워드를 알려주세요"
- 이메일 발송: 수신자가 완전히 없을 때
- doc_compare: 비교할 파일이 특정되지 않을 때
- scheduler: 시간과 내용이 모두 없을 때

✅ clarify 없이 바로 실행 (추론 실행)
- "날씨 어때?" → 한국 현재 날씨로 검색 실행
- "뉴스 알려줘" → 오늘 주요 뉴스로 검색 실행
- "버스 시간표 알려줘" → 질문 그대로 검색 실행
- "노트북 추천해줘" → 인기 노트북 추천으로 검색 실행
- 지역/날짜가 없어도 추론 가능하면 실행 후 결과 제공

━━━ 철칙 ━━━
- clarify 후 사용자가 답하면 즉시 실행 (재질문 금지)
- 한 번 물어본 내용은 다시 묻지 않음 (컨텍스트 유지)
- clarify는 하루 대화에서 최소화 — 사용자를 번거롭게 하지 않음

동일 이름 업체·상품 여러 개 검색 결과 → 목록 나열 후 "어느 것을 원하시나요?" 물어볼 것.
`

// clarify 해소 시 사용하는 별도 시스템 프롬프트
const nexusClarifyResolvePrompt = `당신은 Nexus AI 비서입니다. 사용자가 이전 질문에 대한 추가 정보를 제공했습니다.

이전 컨텍스트와 새 정보를 합쳐서 완전한 액션을 결정하세요.

반드시 JSON만 출력하세요:
{"action":"액션명","params":{...완전한 파라미터...}}

이전에 파악한 액션: %s
이전에 수집한 파라미터: %s
이전 질문: %s
사용자 새 답변: %s

위 정보를 모두 합쳐서 실행 가능한 완전한 액션으로 만드세요.
예시: 이전 액션=web_search, 이전 파라미터={"site":"naver"}, 사용자 답변="서울 날씨"
→ {"action":"web_search","params":{"query":"서울 날씨","site":"naver"}}
`

func handleCommand(w http.ResponseWriter, r *http.Request) {
	if !requireAuth(w, r) {
		return
	}
	start := time.Now()

	var req CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Message) == "" {
		lang := getLang(r)
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("message 필요", "message required", lang)})
		return
	}

	llmMu.RLock()
	gKey := llmPerplexityKey; if gKey == "" { gKey = llmGroqKey }
	llmMu.RUnlock()

	// 언어 자동 감지: 클라이언트 lang 우선, 없으면 메시지 내용으로 판별
	lang := req.Lang
	if lang == "" {
		if isEnglishQuery(req.Message) {
			lang = "en"
		} else {
			lang = "ko"
		}
	}
	var intentAction string
	var intentParams map[string]any

	// ── 멀티턴: 이전 clarify 컨텍스트가 있으면 해소 프롬프트 사용 ──
	if req.PendingIntent != "" {
		prevParamsJSON, _ := json.Marshal(req.PendingParams)
		resolvePrompt := fmt.Sprintf(nexusClarifyResolvePrompt,
			req.PendingIntent,
			string(prevParamsJSON),
			req.PendingQuestion,
			req.Message,
		)
		raw, _, err := callGroqWithFallback([]groqMsg{
			{Role: "user", Content: resolvePrompt},
		}, 256, true)
		if isProxyLimitError(err) {
			dur := fmt.Sprintf("%.2fs", time.Since(start).Seconds())
			resp := upgradeRequiredResponse("ai_request", 0, 0)
			resp.Duration = dur
			json200(w, resp)
			return
		}
		if err != nil || raw == "" {
			// 해소 실패 → 사용자 답변을 pending 파라미터에 병합해서 직접 실행
			intentAction = req.PendingIntent
			intentParams = req.PendingParams
			if intentParams == nil {
				intentParams = map[string]any{}
			}
			// 가장 빈번한 missing 필드에 사용자 답변 적용
			missing := ""
			if req.PendingParams != nil {
				if m, ok := req.PendingParams["__missing__"]; ok {
					missing = fmt.Sprintf("%v", m)
				}
			}
			if missing != "" {
				intentParams[missing] = req.Message
			} else {
				intentParams["query"] = req.Message
			}
		} else {
			var resolved struct {
				Action string         `json:"action"`
				Params map[string]any `json:"params"`
			}
			if err := json.Unmarshal([]byte(raw), &resolved); err == nil && resolved.Action != "" {
				intentAction = resolved.Action
				intentParams = resolved.Params
			} else {
				intentAction = req.PendingIntent
				intentParams = req.PendingParams
				if intentParams == nil {
					intentParams = map[string]any{}
				}
				intentParams["query"] = req.Message
			}
		}
	} else {
		// ── 키워드 사전 라우팅 (LLM이 무시하는 액션들) ────────────
		msgLower := strings.ToLower(req.Message)
		videoKeywords := []string{"찾", "검색", "영상", "보여", "추천", "viral", "바이럴", "트렌드"}
		isTikTokReq := strings.Contains(msgLower, "틱톡") || strings.Contains(msgLower, "tiktok")
		isYouTubeReq := strings.Contains(msgLower, "유튜브") || strings.Contains(msgLower, "youtube")
		hasVideoVerb := false
		for _, kw := range videoKeywords {
			if strings.Contains(msgLower, kw) {
				hasVideoVerb = true
				break
			}
		}

		// ── 쇼핑/도메인 사전 라우팅 ─────────────────────────────
		shoppingSites := map[string]string{
			// 쇼핑몰
			"태무": "temu.com", "테무": "temu.com", "temu": "temu.com",
			"쿠팡": "coupang.com", "coupang": "coupang.com",
			"네이버쇼핑": "shopping.naver.com", "네이버 쇼핑": "shopping.naver.com",
			"11번가": "11st.co.kr",
			"지마켓": "gmarket.co.kr", "gmarket": "gmarket.co.kr",
			"옥션": "auction.co.kr",
			"위메프": "wemakeprice.com",
			"티몬": "tmon.co.kr",
			"알리": "aliexpress.com", "aliexpress": "aliexpress.com", "알리익스프레스": "aliexpress.com",
			"아마존": "amazon.com", "amazon": "amazon.com",
			"무신사": "musinsa.com",
			"에이블리": "a-bly.com",
			"지그재그": "zigzag.kr",
			"브랜디": "brandi.co.kr",
			"오늘의집": "ohou.se",
			"이케아": "ikea.com/kr", "ikea": "ikea.com/kr",
			// 중고차
			"헤이딜러": "heydealer.com", "heydealer": "heydealer.com",
			"엔카": "encar.com", "encar": "encar.com",
			"kb차차차": "kbchachacha.com", "차차차": "kbchachacha.com",
			"오토피디아": "autopedia.co.kr",
			"보배드림": "bobaedream.co.kr",
			"중고차": "encar.com",
			// 중고거래
			"당근": "daangn.com", "당근마켓": "daangn.com", "daangn": "daangn.com",
			"번개장터": "bunjang.co.kr", "번개": "bunjang.co.kr",
			"중고나라": "joongna.com",
			// 부동산
			"직방": "zigbang.com",
			"다방": "dabangapp.com",
			"호갱노노": "hogangnono.com",
			"네이버부동산": "land.naver.com", "네이버 부동산": "land.naver.com",
			"부동산114": "r114.com",
			// 음식/배달
			"배민": "baemin.com", "배달의민족": "baemin.com",
			"요기요": "yogiyo.co.kr",
			"쿠팡이츠": "coupangeats.com",
			// 여행/숙박
			"야놀자": "yanolja.com",
			"여기어때": "goodchoice.kr",
			"에어비앤비": "airbnb.co.kr", "airbnb": "airbnb.com",
			// 전자기기 가격비교
			"다나와": "danawa.com",
			"에누리": "enuri.com",
			"컴퓨존": "compuzone.co.kr",
		}
		detectedShopSite := ""
		for keyword, domain := range shoppingSites {
			if strings.Contains(msgLower, strings.ToLower(keyword)) {
				detectedShopSite = domain
				break
			}
		}

		outFmt := detectOutputFormat(req.Message)
		isMultiAct := outFmt != outNone && hasFileSaveVerb(req.Message)

		if detectedShopSite != "" {
			q := req.Message
			for kw := range shoppingSites {
				q = strings.ReplaceAll(q, kw, "")
			}
			for _, rm := range []string{"에서", "찾아줘", "검색해줘", "최저가", "가격", "얼마야", "구매", "사고 싶어"} {
				q = strings.ReplaceAll(q, rm, "")
			}
			q = strings.TrimSpace(q)
			if q == "" {
				q = req.Message
			}
			if isMultiAct {
				intentAction = "multi_action"
				intentParams = map[string]any{"sub_action": "price_compare", "query": q, "site": detectedShopSite, "max_items": 8, "format": string(outFmt)}
			} else {
				intentAction = "price_compare"
				intentParams = map[string]any{"query": q, "site": detectedShopSite, "max_items": 8}
			}
		} else if isTikTokReq && hasVideoVerb {
			query := req.Message
			for _, rm := range []string{"틱톡에서", "틱톡", "tiktok", "찾아줘", "검색해줘", "보여줘", "영상", "추천해줘"} {
				query = strings.ReplaceAll(query, rm, "")
			}
			query = strings.TrimSpace(query)
			if query == "" {
				query = "바이럴 트렌드"
			}
			intentAction = "video_search"
			intentParams = map[string]any{"query": query, "platform": "tiktok", "max_items": 8}
		} else if isYouTubeReq && hasVideoVerb {
			query := req.Message
			for _, rm := range []string{"유튜브에서", "유튜브", "youtube", "찾아줘", "검색해줘", "보여줘", "영상", "추천해줘"} {
				query = strings.ReplaceAll(query, rm, "")
			}
			query = strings.TrimSpace(query)
			if query == "" {
				query = "인기 영상"
			}
			intentAction = "video_search"
			intentParams = map[string]any{"query": query, "platform": "youtube", "max_items": 8}
		} else {
			// ── 일반 모드: LLM 의도 파악 (대화 이력 포함) ────────────
			// 페르소나 컨텍스트를 라우팅 프롬프트 앞에 주입
			personaCtx := getPersonaSystemPrompt()
			routingPrompt := nexusSystemPrompt
			if lang == "en" {
				routingPrompt = "Respond in English.\n\n" + routingPrompt
			}
			if personaCtx != "" {
				routingPrompt = "▶ 현재 사용자 컨텍스트:\n" + personaCtx + "\n\n" + routingPrompt
			}
			intentMsgs := []groqMsg{{Role: "system", Content: routingPrompt}}
			for _, h := range req.History {
				if len(h.Content) == 0 {
					continue
				}
				role := "user"
				if h.Role == "assistant" {
					role = "assistant"
				}
				content := h.Content
				if len([]rune(content)) > 200 {
					content = string([]rune(content)[:200]) + "..."
				}
				intentMsgs = append(intentMsgs, groqMsg{Role: role, Content: content})
			}
			intentMsgs = append(intentMsgs, groqMsg{Role: "user", Content: req.Message})

			// ── Function Calling 시도 (Groq gsk_ 키일 때) ──
			// 성공하면 tool_call 결과 사용. 실패/비-Groq 키면 기존 JSON 프롬프트 폴백.
			fcRouted := false
			llmMu.RLock()
			isGroqKey := strings.HasPrefix(llmGroqKey, "gsk_")
			llmMu.RUnlock()

			if isGroqKey {
				nexusTools := buildNexusRoutingTools()
				if tcResult, _, tcErr := callGroqWithTools(intentMsgs, nexusTools, 256); tcErr == nil && tcResult != nil {
					intentAction = tcResult.Name
					intentParams = tcResult.Arguments
					if intentParams == nil {
						intentParams = map[string]any{}
					}
					fcRouted = true
				}
			}

			if !fcRouted {
				// 기존 JSON 프롬프트 방식 (Perplexity/프록시 공통)
				raw, _, err := callGroqWithFallback(intentMsgs, 512, true)
				if isProxyLimitError(err) {
					dur := fmt.Sprintf("%.2fs", time.Since(start).Seconds())
					resp := upgradeRequiredResponse("ai_request", 0, 0)
					resp.Duration = dur
					json200(w, resp)
					return
				}
				if err != nil {
					raw = `{"action":"chat","params":{}}`
				}
				var intent struct {
					Action string         `json:"action"`
					Params map[string]any `json:"params"`
				}
				if jsonErr := json.Unmarshal([]byte(raw), &intent); jsonErr != nil || intent.Action == "" {
					intent.Action = "chat"
					intent.Params = map[string]any{}
				}
				if intent.Params == nil {
					intent.Params = map[string]any{}
				}
				intentAction = intent.Action
				intentParams = intent.Params
			}
		}
	}

	// ── clarify 액션: 실행 없이 질문 반환 ────────────────────
	if intentAction == "clarify" {
		question, _ := intentParams["question"].(string)
		missing, _ := intentParams["missing"].(string)
		pendingIntent, _ := intentParams["intent"].(string)
		collected, _ := intentParams["collected"].(map[string]any)
		if collected == nil {
			collected = map[string]any{}
		}
		if missing != "" {
			collected["__missing__"] = missing
		}
		if question == "" {
			if lang == "en" {
				question = "Could you provide more details?"
			} else {
				question = "조금 더 자세히 알려주시겠어요?"
			}
		}
		if pendingIntent == "" {
			pendingIntent = "chat"
		}

		// LLM이 제공한 선택지 파싱
		var clarifyOpts []string
		if v, ok := intentParams["options"]; ok {
			if arr, ok2 := v.([]interface{}); ok2 {
				for _, o := range arr {
					if s, ok3 := o.(string); ok3 && s != "" {
						clarifyOpts = append(clarifyOpts, s)
					}
				}
			}
		}
		// LLM이 선택지 안 줬으면 missing 필드 기반으로 자동 생성
		if len(clarifyOpts) == 0 {
			switch missing {
			case "query", "topic", "subject":
				if lang == "en" {
					clarifyOpts = []string{"Web Search", "News", "Shopping", "YouTube"}
				} else {
					clarifyOpts = []string{"웹 검색", "뉴스", "쇼핑", "유튜브"}
				}
			case "product":
				if lang == "en" {
					clarifyOpts = []string{"Price Comparison", "Reviews", "Shopping", "News"}
				} else {
					clarifyOpts = []string{"가격 비교", "리뷰", "쇼핑", "뉴스"}
				}
			case "location", "city":
				if lang == "en" {
					clarifyOpts = []string{"Seoul", "Busan", "Incheon", "Jeju"}
				} else {
					clarifyOpts = []string{"서울", "부산", "인천", "제주"}
				}
			case "intent":
				if lang == "en" {
					clarifyOpts = []string{"Search Info", "Set Schedule", "Check Files", "Control PC"}
				} else {
					clarifyOpts = []string{"정보 검색", "일정 등록", "파일 확인", "PC 제어"}
				}
			}
		}

		json200(w, CommandResponse{
			Success:          true,
			Message:          question,
			Action:           "clarify",
			NeedsClarify:     true,
			ClarifyQuestion:  question,
			ClarifyQuestions: clarifyOpts,
			PendingIntent:    pendingIntent,
			PendingParams:    collected,
			Duration:         time.Since(start).String(),
		})
		return
	}

	// ── 액션 실행 ────────────────────────────────────────────
	// dispatchAction의 직접 메시지를 사용 (이중 LLM 호출 제거)
	// req.Context에 페르소나 ID("persona:medical" 등)가 있으면 params에 주입
	if req.Context != "" {
		if intentParams == nil {
			intentParams = map[string]any{}
		}
		intentParams["_persona_ctx"] = req.Context
	}
	result, msg := dispatchAction(intentAction, intentParams, req.Message, gKey, lang, req.History)

	saveAgentMemory(AgentMemoryEntry{
		ID:        fmt.Sprintf("cmd_%d", time.Now().Unix()),
		Timestamp: time.Now().Format(time.RFC3339),
		Type:      "command",
		Command:   req.Message,
		Result:    fmt.Sprintf("action=%s msg=%s", intentAction, truncateStr(msg, 100)),
		Tags:      []string{intentAction},
		Success:   true,
	})

	json200(w, CommandResponse{
		Success:  true,
		Message:  msg,
		Action:   intentAction,
		Result:   result,
		Duration: time.Since(start).String(),
	})
}

// ══════════════════════════════════════════════════════════════════
//  dispatchAction: 액션 → 실제 함수 실행
// ══════════════════════════════════════════════════════════════════
func dispatchAction(action string, params map[string]any, original, gKey, lang string, history []ConvHistoryMsg) (result any, message string) {
	str := func(key string) string {
		if v, ok := params[key]; ok {
			return fmt.Sprintf("%v", v)
		}
		return ""
	}
	boolVal := func(key string, def bool) bool {
		if v, ok := params[key]; ok {
			if b, ok := v.(bool); ok {
				return b
			}
		}
		return def
	}
	intVal := func(key string, def int) int {
		if v, ok := params[key]; ok {
			if f, ok := v.(float64); ok {
				return int(f)
			}
		}
		return def
	}

	switch action {

	// ── 복합 워크플로: 2단계 이상 작업 ──────────────────────
	case "workflow_run":
		if uid := getMachineID(); true {
			if allowed, used, lim := checkUsageLimit(uid, "workflow_run"); !allowed {
				r := upgradeRequiredResponse("workflow_run", used, lim)
				return r, r.Message
			} else {
				incrementUsage(uid, "workflow_run")
				_ = used
				_ = lim
			}
		}
		goal := str("goal")
		if goal == "" {
			goal = original
		}
		steps, summary, _ := runWithReflection(goal)
		doneCount := 0
		for _, s := range steps {
			if s.Status == "done" {
				doneCount++
			}
		}
		return map[string]any{"goal": goal, "steps": steps, "summary": summary, "ok": doneCount > 0},
			fmt.Sprintf(msgT("워크플로 완료 (%d/%d단계). %s", "Workflow complete (%d/%d steps). %s", lang), doneCount, len(steps), summary)

	// ── 💼 Pro Persona 전용 액션 ────────────────────────────

	case "stock_analysis":
		if uid := getMachineID(); true {
			if allowed, used, lim := checkUsageLimit(uid, "stock_analysis"); !allowed {
				r := upgradeRequiredResponse("stock_analysis", used, lim)
				return r, r.Message
			} else {
				incrementUsage(uid, "stock_analysis")
				_ = used
				_ = lim
			}
		}
		ticker := str("ticker")
		query := str("query")
		if ticker == "" {
			ticker = query
		}
		if ticker == "" {
			ticker = original
		}
		return stockAnalysisLogic(ticker, query, lang)

	case "medical_search":
		if uid := getMachineID(); true {
			if allowed, used, lim := checkUsageLimit(uid, "medical_search"); !allowed {
				r := upgradeRequiredResponse("medical_search", used, lim)
				return r, r.Message
			} else {
				incrementUsage(uid, "medical_search")
				_ = used
				_ = lim
			}
		}
		query := str("query")
		if query == "" {
			query = original
		}
		return medicalSearchLogic(query, str("type"), lang)

	case "contract_review":
		if uid := getMachineID(); true {
			if allowed, used, lim := checkUsageLimit(uid, "contract_review"); !allowed {
				r := upgradeRequiredResponse("contract_review", used, lim)
				return r, r.Message
			} else {
				incrementUsage(uid, "contract_review")
				_ = used
				_ = lim
			}
		}
		return contractReviewLogic(str("file_path"), str("content"), str("focus"), lang)

	case "legal_search":
		if uid := getMachineID(); true {
			if allowed, used, lim := checkUsageLimit(uid, "legal_search"); !allowed {
				r := upgradeRequiredResponse("legal_search", used, lim)
				return r, r.Message
			} else {
				incrementUsage(uid, "legal_search")
				_ = used
				_ = lim
			}
		}
		query := str("query")
		if query == "" {
			query = original
		}
		return legalSearchLogic(query, str("type"), lang)

	case "content_script":
		if uid := getMachineID(); true {
			if allowed, used, lim := checkUsageLimit(uid, "content_script"); !allowed {
				r := upgradeRequiredResponse("content_script", used, lim)
				return r, r.Message
			} else {
				incrementUsage(uid, "content_script")
				_ = used
				_ = lim
			}
		}
		topic := str("topic")
		if topic == "" {
			topic = original
		}
		return contentScriptLogic(topic, str("platform"), str("duration"), str("style"), lang)

	// ── 일반 대화 ───────────────────────────────────────────
	case "chat":
		// 실시간 정보 카테고리면 web_search로 리다이렉트
		// (이전 대화에서 이미 카테고리가 정해진 경우도 포함)
		resolvedQuery := resolveWithHistory(original, history)
		cat := detectCategory(resolvedQuery)
		realtime := cat == catTransit || cat == catFood || cat == catShopping ||
			cat == catFinance || cat == catWeather || cat == catNews ||
			cat == catMedical || cat == catEntertainment || cat == catTravel ||
			cat == catRealEstate
		if realtime {
			pr := parallelWebSearch(resolvedQuery, 5, lang)
			items := pr.Items
			if len(items) == 0 {
				items = categoryFallbackSites(resolvedQuery, cat)
			}
			msg := pr.Summary
			if msg == "" || containsBotBlockText(msg) {
				if cleaned := cleanPerplexityCall(resolvedQuery, gKey); cleaned != "" {
					msg = cleaned
				} else if msg == "" {
					msg = buildNoResultMessage(resolvedQuery, cat, "")
				}
			}
			return map[string]any{"query": resolvedQuery, "summary": msg, "items": items}, msg
		}
		// ── 전문가 자동 감지 + 병렬 실행 ──────────────────────────
		expertList := detectExperts(original, lang)

		var ans string
		var expertCites []string
		var chatSearchItems []map[string]string
		var chatWg sync.WaitGroup
		chatWg.Add(2)

		go func() {
			defer chatWg.Done()
			// 전문가 있으면 전문가 시스템으로, 없으면 일반 Nexus AI 답변
			if len(expertList) > 0 {
				ans, expertCites = runExpertParallel(original, lang, gKey, expertList, history)
			}
			// 전문가 결과 없거나 전문가 미감지 → 일반 채팅 (citations 포함)
			if ans == "" {
				// 직업군별 system prompt 주입
				verticalCfg := loadVerticalConfig()
				verticalSys := VerticalSystemPrompts[verticalCfg.ID]
				if verticalSys == "" {
					verticalSys = VerticalSystemPrompts["general"]
				}
				var chatSys string
				if lang == "en" {
					chatSys = VerticalSystemPromptsEN[verticalCfg.ID]
					if chatSys == "" {
						chatSys = VerticalSystemPromptsEN["general"]
					}
				} else {
					chatSys = verticalSys
				}
				if lang == "en" {
					chatSys += fmt.Sprintf("\nCurrent time: %s (local)", time.Now().Format("2006-01-02 15:04"))
				} else {
					chatSys += fmt.Sprintf("\n현재 시각: %s (로컬)", time.Now().Format("2006-01-02 15:04"))
				}
				chatMsgs := []groqMsg{{Role: "system", Content: chatSys}}
				for _, h := range history {
					role := "user"
					if h.Role == "assistant" {
						role = "assistant"
					}
					content := h.Content
					if len([]rune(content)) > 300 {
						content = string([]rune(content)[:300]) + "..."
					}
					chatMsgs = append(chatMsgs, groqMsg{Role: role, Content: content})
				}
				chatMsgs = append(chatMsgs, groqMsg{Role: "user", Content: original})
				ans, expertCites, _ = callGroqWithCitations(gKey, groqChatModel, chatMsgs, 600)
			}
		}()

		go func() {
			defer chatWg.Done()
			// 전문가 카테고리 힌트 전달 → 전문가 분야에 맞는 상세 페이지 검색
			expertCat := expertsToCategory(expertList)
			pr := parallelWebSearch(original, 6, expertCat, lang)
			if len(pr.Items) > 0 {
				chatSearchItems = pr.Items
			} else {
				searchCat := cat
				if expertCat >= 0 {
					searchCat = expertCat
				}
				chatSearchItems = categoryFallbackSites(original, searchCat)
			}
		}()
		chatWg.Wait()

		// citations를 items로 변환 (전문가가 실제로 본 URL 우선)
		if len(expertCites) > 0 {
			citeItems := make([]map[string]string, 0, len(expertCites))
			for _, u := range expertCites {
				citeItems = append(citeItems, map[string]string{"title": extractDomain(u), "url": u})
			}
			chatSearchItems = citeItems
		}

		previewType := categoryPreviewType(cat)
		return map[string]any{"reply": ans, "items": chatSearchItems, "preview_type": previewType}, ans

	// ── 날씨 ─────────────────────────────────────────────
	case "weather":
		city := str("city")
		if city == "" {
			city = "서울"
		}
		text := fetchWeatherText(city, gKey)
		return map[string]any{"reply": text}, text

	// ── 웹 검색 & 쇼핑 ──────────────────────────────────────
	case "web_search":
		query := str("query")
		if query == "" {
			query = original
		}
		// 이전 대화로 모호한 쿼리 보완
		query = resolveWithHistory(query, history)
		site := str("site")
		output := str("output")
		maxItems := intVal("max_items", 8)

		// ── 페르소나 도메인 필터 ──────────────────────────────
		personaCtx := str("_persona_ctx") // "persona:medical" 등
		// 페르소나별 Tavily include_domains 매핑
		personaDomains := map[string][]string{
			"persona:medical":    {"pubmed.ncbi.nlm.nih.gov", "who.int", "health.gov", "medscape.com", "webmd.com", "healthline.com"},
			"persona:legal":      {"law.go.kr", "lawnb.com", "lawmake.go.kr", "courts.go.kr", "legalengine.co.kr"},
			"persona:developer":  {"stackoverflow.com", "github.com", "docs.microsoft.com", "developer.mozilla.org", "npmjs.com"},
			"persona:finance":    {"finance.naver.com", "investing.com", "bloomberg.com", "reuters.com", "hankyung.com"},
			"persona:accountant": {"nts.go.kr", "taxnet.or.kr", "kacpta.or.kr", "bizforms.co.kr"},
			"persona:realtor":    {"realestate.daum.net", "land.naver.com", "zigbang.com", "dabangapp.com", "molit.go.kr"},
			"persona:hr":         {"사람인.com", "jobkorea.co.kr", "wanted.co.kr", "moel.go.kr", "laborlaw.mohw.go.kr"},
			"persona:engineer":   {"iiec.or.kr", "kssc.or.kr", "kats.go.kr", "iso.org", "ieee.org"},
		}
		llmMu.RLock()
		wsKey := llmTavilyKey
		llmMu.RUnlock()

		// 사용자 메시지에 명시적 파일 저장 요청이 있을 때만 runWebSearch 호출
		// LLM이 임의로 output 파라미터를 생성해도 무시
		userWantsFile := detectOutputFormat(original) != outNone && hasFileSaveVerb(original)
		if !userWantsFile {
			cat := detectCategory(query)
			var pr parallelSearchResult

			// 페르소나 도메인 필터가 있고 Tavily 키가 있으면 도메인 필터링 검색
			if domains, ok := personaDomains[personaCtx]; ok && wsKey != "" && len(domains) > 0 {
				// 도메인별로 Tavily 검색 (최대 2개 도메인)
				var filteredItems []map[string]string
				for _, dom := range domains[:min(2, len(domains))] {
					if tr, ok2 := tavilySearchDomain(wsKey, query, maxItems/2+1, dom); ok2 {
						filteredItems = append(filteredItems, tr.Items...)
					}
					if len(filteredItems) >= maxItems {
						break
					}
				}
				if len(filteredItems) > maxItems {
					filteredItems = filteredItems[:maxItems]
				}
				// 도메인 필터 결과로 pr 구성
				pr = parallelWebSearch(query, 3, lang) // 요약은 일반 검색에서
				pr.Items = append(filteredItems, pr.Items...)
				if len(pr.Items) > maxItems {
					pr.Items = pr.Items[:maxItems]
				}
			} else {
				pr = parallelWebSearch(query, maxItems, lang)
			}

			items := pr.Items
			if len(items) == 0 {
				items = buildFallbackURLs(query, site)
			}
			if len(items) == 0 {
				items = categoryFallbackSites(query, cat)
			}
			msg := pr.Summary
			if msg == "" || containsBotBlockText(msg) {
				if cleaned := cleanPerplexityCall(query, gKey); cleaned != "" {
					msg = cleaned
				} else if msg == "" {
					msg = buildNoResultMessage(query, cat, "")
				}
			}
			result := map[string]any{
				"query":        query,
				"site":         site,
				"summary":      msg,
				"items":        items,
				"preview_type": categoryPreviewType(cat),
			}
			return result, msg
		}
		return runWebSearch(query, site, output, maxItems, gKey, lang)

	// ── 영상 검색 (YouTube / TikTok) ─────────────────────────
	case "video_search":
		query := str("query")
		if query == "" {
			query = original
		}
		platform := strings.ToLower(str("platform"))
		maxItems := intVal("max_items", 8)

		// tKey를 dispatchAction 스코프 내에서 직접 조회
		llmMu.RLock()
		videoTKey := llmTavilyKey
		llmMu.RUnlock()

		isTikTok := platform == "tiktok" ||
			strings.Contains(strings.ToLower(original), "틱톡") ||
			strings.Contains(strings.ToLower(original), "tiktok")

		if isTikTok {
			// TikTok: site: 접두사는 Tavily에서 0결과 → include_domains 방식 사용
			var items []map[string]string

			if videoTKey != "" {
				if tr, ok := tavilySearchDomain(videoTKey, query, maxItems, "tiktok.com"); ok {
					for _, it := range tr.Items {
						if strings.Contains(it["url"], "tiktok.com") {
							items = append(items, it)
						}
					}
				}
				// include_domains로 결과 없으면 일반 검색 후 tiktok.com URL 필터
				if len(items) == 0 {
					if tr, ok := tavilySearch(videoTKey, query+" tiktok", maxItems); ok {
						for _, it := range tr.Items {
							if strings.Contains(it["url"], "tiktok.com") {
								items = append(items, it)
							}
						}
					}
				}
			}
			// Tavily 결과가 없으면 LLM으로 보완
			if len(items) == 0 && gKey != "" {
				pplxPrompt := fmt.Sprintf(`TikTok에서 "%s" 관련 실제 영상 링크를 최대 %d개 찾아줘. 반드시 tiktok.com URL만 포함. JSON 배열로만 출력: [{"title":"...", "url":"https://tiktok.com/..."}]`, query, maxItems)
				raw, _, _ := callGroqWithFallback([]groqMsg{{Role: "user", Content: pplxPrompt}}, 512, true)
				var parsed []map[string]string
				if json.Unmarshal([]byte(raw), &parsed) == nil {
					for _, it := range parsed {
						if strings.Contains(it["url"], "tiktok.com") {
							items = append(items, it)
						}
					}
				}
			}
			// 최후 fallback: TikTok 검색 페이지 링크 제공
			if len(items) == 0 {
				enc := strings.ReplaceAll(query, " ", "%20")
				items = []map[string]string{
					{"title": fmt.Sprintf("TikTok에서 \"%s\" 검색", query), "url": fmt.Sprintf("https://www.tiktok.com/search?q=%s", enc)},
					{"title": "TikTok 트렌딩", "url": "https://www.tiktok.com/trending"},
				}
			}
			summary := fmt.Sprintf("TikTok에서 \"%s\" 영상 %d개를 찾았어요!", query, len(items))
			return map[string]any{"query": query, "platform": "tiktok", "items": items, "total": len(items), "summary": summary}, summary
		}

		// YouTube: site: 접두사는 Tavily에서 0결과 → include_domains 방식 사용
		var ytItems []map[string]string
		if videoTKey != "" {
			if tr, ok := tavilySearchDomain(videoTKey, query, maxItems, "youtube.com"); ok {
				for _, it := range tr.Items {
					if strings.Contains(it["url"], "youtube.com/watch") || strings.Contains(it["url"], "youtu.be") {
						ytItems = append(ytItems, it)
					}
				}
			}
			// include_domains 결과 없으면 일반 검색 후 youtube URL 필터
			if len(ytItems) == 0 {
				if tr, ok := tavilySearch(videoTKey, query+" youtube 영상", maxItems); ok {
					for _, it := range tr.Items {
						if strings.Contains(it["url"], "youtube.com/watch") || strings.Contains(it["url"], "youtu.be") {
							ytItems = append(ytItems, it)
						}
					}
				}
			}
		}
		if len(ytItems) == 0 {
			enc := strings.ReplaceAll(query, " ", "%20")
			ytItems = []map[string]string{
				{"title": fmt.Sprintf("YouTube에서 \"%s\" 검색", query), "url": fmt.Sprintf("https://www.youtube.com/results?search_query=%s", enc)},
			}
		}
		ytSummary := fmt.Sprintf("YouTube에서 \"%s\" 영상 %d개를 찾았어요!", query, len(ytItems))
		return map[string]any{"query": query, "platform": "youtube", "items": ytItems, "total": len(ytItems), "summary": ytSummary}, ytSummary

	// ── 가격/쇼핑 검색 ───────────────────────────────────────
	case "price_compare":
		pcQuery := str("query")
		if pcQuery == "" { pcQuery = original }
		pcSite := str("site")
		pcMax := intVal("max_items", 8)
		llmMu.RLock()
		pcTKey := llmTavilyKey
		llmMu.RUnlock()
		var priceItems []map[string]string
		if pcTKey != "" {
			// include_domains 방식 사용 (site: 접두사는 결과 0개 버그 있음)
			if pcSite != "" {
				if tr, ok := tavilySearchDomain(pcTKey, pcQuery, pcMax, pcSite); ok {
					priceItems = tr.Items
				}
			}
			// 도메인 검색 결과 없으면 일반 검색
			if len(priceItems) == 0 {
				if tr, ok := tavilySearch(pcTKey, pcQuery, pcMax); ok {
					priceItems = tr.Items
				}
			}
		}
		siteName := pcSite; if siteName == "" { siteName = "쇼핑몰" }
		if len(priceItems) == 0 {
			enc := strings.ReplaceAll(pcQuery, " ", "+")
			priceItems = []map[string]string{{"title": pcQuery + " 검색", "url": "https://www." + pcSite + "/search?q=" + enc}}
		}
		results := make([]map[string]string, 0, len(priceItems))
		for _, it := range priceItems {
			results = append(results, map[string]string{"site": siteName, "name": it["title"], "price": "", "link": it["url"]})
		}
		pcSummary := fmt.Sprintf("%s에서 \"%s\" 상품 %d개를 찾았어요!", siteName, pcQuery, len(results))
		return map[string]any{"query": pcQuery, "site": pcSite, "summary": pcSummary, "results": results, "total": len(results)}, pcSummary

	// ── 멀티 액션: 검색 + 파일 저장 ────────────────────────────
	case "multi_action":
		maSubAction := str("sub_action")
		maQuery := str("query"); if maQuery == "" { maQuery = original }
		maSite := str("site")
		maPlatform := str("platform")
		maFmtStr := str("format")
		maMax := intVal("max_items", 8)
		maFmt := outputFormat(maFmtStr)
		if maFmt == "" { maFmt = outMarkdown }
		llmMu.RLock()
		maTKey := llmTavilyKey
		llmMu.RUnlock()

		var maItems []map[string]string
		var maActionSummary string

		switch maSubAction {
		case "price_compare":
			if maSite != "" {
				if tr, ok := tavilySearchDomain(maTKey, maQuery, maMax, maSite); ok {
					maItems = tr.Items
				}
			}
			if len(maItems) == 0 {
				if tr, ok := webSearchWithFallback(maTKey, maQuery, maMax); ok {
					maItems = tr.Items
				}
			}
			sn := maSite; if sn == "" { sn = "store" }
			maActionSummary = fmt.Sprintf("%s: \"%s\" — %d results", sn, maQuery, len(maItems))

		case "video_search":
			targetDomain := "youtube.com"
			if maPlatform == "tiktok" { targetDomain = "tiktok.com" }
			if maTKey != "" {
				if tr, ok := tavilySearchDomain(maTKey, maQuery, maMax, targetDomain); ok {
					maItems = tr.Items
				}
				if len(maItems) == 0 {
					if tr, ok := tavilySearch(maTKey, maQuery+" "+targetDomain, maMax); ok {
						maItems = tr.Items
					}
				}
			}
			pn := "YouTube"; if maPlatform == "tiktok" { pn = "TikTok" }
			maActionSummary = fmt.Sprintf("%s: \"%s\" — %d results", pn, maQuery, len(maItems))

		case "doc_compare":
			llmMu.RLock()
			gKey := llmPerplexityKey; if gKey == "" { gKey = llmGroqKey }
			llmMu.RUnlock()
			var compareText string
			if tr, ok := webSearchWithFallback(maTKey, maQuery, maMax); ok {
				maItems = tr.Items
				var articleLines strings.Builder
				for i, item := range tr.Items {
					t := item["title"]; c := item["content"]
					if c == "" { c = item["snippet"] }
					articleLines.WriteString(fmt.Sprintf("[%d] %s\n%s\n\n", i+1, t, c))
				}
				if tr.Summary != "" {
					articleLines.WriteString("\n[Summary]\n" + tr.Summary)
				}
				prompt := fmt.Sprintf(`Compare "%s" in a structured markdown table (| Feature | A | B |) in English based on:\n%s`, maQuery, articleLines.String())
				if gKey != "" {
					compareText, _, _ = callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 2000, false)
				} else if llmClaudeKey != "" {
					compareText, _ = callClaude(llmClaudeKey, []groqMsg{{Role: "user", Content: prompt}}, 2000)
				}
			}
			if compareText == "" {
				llmMu.RLock()
				gKey2 := llmPerplexityKey; cKey2 := llmClaudeKey
				llmMu.RUnlock()
				prompt := fmt.Sprintf(`Compare "%s" in a detailed markdown table (| Feature | A | B |) in English.`, maQuery)
				if gKey2 != "" {
					compareText, _, _ = callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 2000, false)
				} else if cKey2 != "" {
					compareText, _ = callClaude(cKey2, []groqMsg{{Role: "user", Content: prompt}}, 2000)
				}
			}
			if compareText == "" {
				compareText = fmt.Sprintf(`"%s" comparison — no API key configured.`, maQuery)
			}
			maActionSummary = compareText

		case "summarize":
			llmMu.RLock()
			gKey := llmPerplexityKey; if gKey == "" { gKey = llmGroqKey }
			llmMu.RUnlock()
			var summaryText string
			if tr, ok := webSearchWithFallback(maTKey, maQuery, maMax); ok {
				maItems = tr.Items
				var articleLines strings.Builder
				for i, item := range tr.Items {
					t := item["title"]; c := item["content"]
					if c == "" { c = item["snippet"] }
					if t == "" { continue }
					articleLines.WriteString(fmt.Sprintf("[%d] %s\n%s\n\n", i+1, t, c))
				}
				if tr.Summary != "" {
					articleLines.WriteString("\n[Overall Summary]\n" + tr.Summary)
				}
				prompt := fmt.Sprintf(`Summarize "%s" in English with clear headings (## Section, - bullet points) based on:\n%s`, maQuery, articleLines.String())
				if gKey != "" {
					summaryText, _, _ = callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 2000, false)
				} else if llmClaudeKey != "" {
					summaryText, _ = callClaude(llmClaudeKey, []groqMsg{{Role: "user", Content: prompt}}, 2000)
				}
			}
			if summaryText == "" {
				llmMu.RLock()
				gKey2 := llmPerplexityKey; cKey2 := llmClaudeKey
				llmMu.RUnlock()
				prompt := fmt.Sprintf(`Summarize "%s" in English with clear headings (## Section, - bullet points).`, maQuery)
				if gKey2 != "" {
					summaryText, _, _ = callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 2000, false)
				} else if cKey2 != "" {
					summaryText, _ = callClaude(cKey2, []groqMsg{{Role: "user", Content: prompt}}, 2000)
				}
			}
			if summaryText == "" {
				summaryText = fmt.Sprintf(`"%s" — search quota exceeded. Register a Brave Search API key for continued access.`, maQuery)
			}
			maActionSummary = summaryText

		default:
			llmMu.RLock()
			gKey := llmPerplexityKey; if gKey == "" { gKey = llmGroqKey }
			llmMu.RUnlock()
			if tr, ok := webSearchWithFallback(maTKey, maQuery, maMax); ok {
				maItems = tr.Items
				if gKey != "" {
					var lines strings.Builder
					for i, item := range tr.Items {
						t := item["title"]; c := item["content"]
						if c == "" { c = item["snippet"] }
						lines.WriteString(fmt.Sprintf("[%d] %s\n%s\n\n", i+1, t, c))
					}
					if tr.Summary != "" { lines.WriteString("\n[Summary]\n" + tr.Summary) }
					prompt := fmt.Sprintf(`Summarize "%s" in 3-5 key points in English.\n\n%s`, maQuery, lines.String())
					maActionSummary, _, _ = callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 800, false)
				}
			}
			if maActionSummary == "" {
				maActionSummary = fmt.Sprintf(`"%s" — %d results`, maQuery, len(maItems))
			}
		}

		maTitle := maQuery
		if len([]rune(maTitle)) > 20 { maTitle = string([]rune(maTitle)[:20]) }
		maFilePath, maSaveErr := saveResultToFile(maFmt, maTitle, maItems, maActionSummary)
		var maFileMsg string
		if maSaveErr != nil {
			maFileMsg = "⚠️ Save failed: " + maSaveErr.Error()
		} else {
			extMap := map[outputFormat]string{
				outPDF: "HTML(PDF)", outWord: "DOCX", outExcel: "XLSX",
				outPowerPoint: "PPTX", outMarkdown: "MARKDOWN", outTXT: "TXT",
			}
			ext := extMap[maFmt]
			if ext == "" { ext = strings.ToUpper(maFmtStr) }
			maFileMsg = fmt.Sprintf("📄 Saved as %s: %s", ext, maFilePath)
		}
		maResults := make([]map[string]string, 0, len(maItems))
		for _, it := range maItems {
			maResults = append(maResults, map[string]string{"site": maSite, "name": it["title"], "price": it["price"], "link": it["url"]})
		}
		return map[string]any{
			"query": maQuery, "summary": maActionSummary, "results": maResults, "total": len(maResults),
			"file_path": maFilePath, "file_msg": maFileMsg, "format": maFmtStr, "sub_action": maSubAction,
		}, maActionSummary + "\n" + maFileMsg

	// ── 파일 검색 ────────────────────────────────────────────
	case "file_search":
		query := str("query")
		if query == "" {
			query = original
		}
		folder := str("folder")
		if folder == "" {
			folder, _ = os.UserHomeDir()
		}
		maxResults := intVal("max_results", 15)
		// AI 키워드 추출 후 검색
		keywords := []string{query}
		if gKey != "" {
			var ep string
			if lang == "en" {
				ep = fmt.Sprintf(`Extract key search keywords from this query: "%s"\nJSON only: {"keywords":["k1","k2"]}`, query)
			} else {
				ep = fmt.Sprintf(`파일 검색 쿼리에서 핵심 키워드만 추출: "%s"\nJSON: {"keywords":["k1","k2"]}`, query)
			}
			raw, _, _ := callGroqWithFallback([]groqMsg{{Role: "user", Content: ep}}, 128, true)
			var kw struct {
				Keywords []string `json:"keywords"`
			}
			if json.Unmarshal([]byte(raw), &kw) == nil && len(kw.Keywords) > 0 {
				keywords = kw.Keywords
			}
		}
		hits := deepSearchFiles(strings.Join(keywords, " "), folder, maxResults)
		if len(hits) == 0 {
			return hits, fmt.Sprintf(msgT("'%s'와 관련된 파일을 찾지 못했습니다.", "No files found matching '%s'.", lang), query)
		}
		var msg string
		if lang == "en" {
			msg = fmt.Sprintf("Found %d file(s) for '%s':\n", len(hits), query)
		} else {
			msg = fmt.Sprintf("'%s' 검색 결과: %d개 파일 발견\n", query, len(hits))
		}
		for i, h := range hits {
			if i >= 5 {
				msg += fmt.Sprintf(msgT("  ... 외 %d개\n", "  ... and %d more\n", lang), len(hits)-5)
				break
			}
			msg += fmt.Sprintf("  • %s\n", h.Path)
		}
		return hits, msg

	// ── 문서 비교 ────────────────────────────────────────────
	case "doc_compare":
		fileA := str("file_a")
		fileB := str("file_b")
		if fileA == "" || fileB == "" {
			return nil, msgT("비교할 두 파일 경로를 알려주세요.\n예: '계약서_v1.pdf 와 계약서_v2.pdf 비교해줘'", "Please provide two file paths to compare.\nExample: 'Compare contract_v1.pdf and contract_v2.pdf'", lang)
		}
		textA, errA := extractDocumentText(fileA)
		textB, errB := extractDocumentText(fileB)
		if errA != nil {
			return nil, msgT("파일A를 읽을 수 없습니다: ", "Cannot read File A: ", lang) + fileA
		}
		if errB != nil {
			return nil, msgT("파일B를 읽을 수 없습니다: ", "Cannot read File B: ", lang) + fileB
		}
		if len(textA) > 4000 { textA = textA[:4000] }
		if len(textB) > 4000 { textB = textB[:4000] }
		var prompt string
		if lang == "en" {
			prompt = fmt.Sprintf(`Compare and analyze the two documents and respond ONLY in JSON:
=== Document A: %s ===
%s
=== Document B: %s ===
%s
{"summary":"summary","total_differences":number,"differences":[{"type":"added|deleted|modified","description":"description","severity":"low|medium|high"}],"risk_level":"low|medium|high","recommendation":"recommendation"}`,
				fileA, textA, fileB, textB)
		} else {
			prompt = fmt.Sprintf(`두 문서를 비교 분석해서 JSON으로만 응답:
=== 문서A: %s ===
%s
=== 문서B: %s ===
%s
{"summary":"요약","total_differences":숫자,"differences":[{"type":"added|deleted|modified","description":"설명","severity":"low|medium|high"}],"risk_level":"low|medium|high","recommendation":"권고사항"}`,
				fileA, textA, fileB, textB)
		}
		ans, _, err := callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 2048, true)
		if err != nil {
			return nil, msgT("문서 비교 실패: ", "Document comparison failed: ", lang) + err.Error()
		}
		var parsed map[string]any
		json.Unmarshal([]byte(ans), &parsed)
		summary := msgT("문서 비교 완료", "Document comparison complete", lang)
		if s, ok := parsed["summary"].(string); ok { summary = s }
		return parsed, msgT("문서 비교 완료!\n", "Document comparison complete!\n", lang) + summary

	// ── 문서 요약 ────────────────────────────────────────────
	case "doc_summary":
		filePath := str("file_path")
		if filePath == "" {
			return nil, msgT("요약할 파일 경로를 알려주세요.", "Please provide the file path to summarize.", lang)
		}
		question := str("question")
		if question == "" {
			question = msgT("핵심 내용을 5줄로 요약하고 중요 수치·날짜·이름을 정리해주세요.", "Summarize the key content in 5 lines and list important figures, dates, and names.", lang)
		}
		text, err := extractDocumentText(filePath)
		if err != nil {
			return nil, msgT("파일 읽기 실패: ", "Failed to read file: ", lang) + err.Error()
		}
		if len(text) > 8000 { text = text[:8000] }
		docMsg := fmt.Sprintf(msgT("문서:\n%s\n\n요청: %s", "Document:\n%s\n\nRequest: %s", lang), text, question)
		ans, _, err := callGroqWithFallback([]groqMsg{{Role: "user", Content: docMsg}}, 2048, false)
		if err != nil {
			return nil, msgT("요약 실패: ", "Summary failed: ", lang) + err.Error()
		}
		return map[string]any{"summary": ans, "file": filePath}, ans

	// ── 폴더 정리 ────────────────────────────────────────────
	case "organize_folder":
		folder := str("folder")
		home, _ := os.UserHomeDir()
		switch strings.ToLower(folder) {
		case "downloads", "다운로드":
			folder = filepath.Join(home, "Downloads")
		case "desktop", "바탕화면":
			folder = filepath.Join(home, "Desktop")
		case "documents", "문서":
			folder = filepath.Join(home, "Documents")
		default:
			if folder == "" { folder = filepath.Join(home, "Downloads") }
		}
		freed, fileCount := organizeFolder(folder)
		return map[string]any{"folder": folder, "files_organized": fileCount, "freed_mb": freed},
			fmt.Sprintf(msgT("'%s' 폴더 정리 완료!\n%d개 파일 정리됨", "'%s' folder organized!\n%d files sorted", lang), folder, fileCount)

	// ── 화면 분석 ────────────────────────────────────────────
	case "vision":
		question := str("question")
		if question == "" {
			question = msgT("지금 화면을 분석해서 무슨 내용인지, 오류가 있으면 원인과 해결법을 한국어로 알려주세요.", "Analyze the current screen and describe what it shows. If there is an error, explain the cause and how to fix it.", lang)
		}
		b64, _, _, err := captureScreenPowerShell()
		if err != nil {
			return nil, msgT("화면 캡처 실패: ", "Screen capture failed: ", lang) + err.Error()
		}
		ans, err := callGroqVision(gKey, b64, "image/png", question)
		if err != nil {
			return nil, msgT("화면 분석 실패: ", "Screen analysis failed: ", lang) + err.Error()
		}
		return map[string]any{"answer": ans}, ans

	// ── PC 진단 ─────────────────────────────────────────────
	case "scan":
		sr := buildScanResult()
		msg := fmt.Sprintf(msgT("PC 점수: %d점", "PC Score: %d/100", lang), sr.Score)
		if len(sr.Issues) == 0 {
			msg += msgT(" — 모두 정상이에요! ✅", " — Everything looks good! ✅", lang)
		} else {
			msg += fmt.Sprintf(msgT("\n발견된 문제 %d개:\n", "\n%d issue(s) found:\n", lang), len(sr.Issues))
			for _, i := range sr.Issues { msg += "  • " + i.Title + "\n" }
		}
		return sr, msg

	// ── PC 정리 ──────────────────────────────────────────────
	case "clean":
		freed := cleanTempFiles()
		freedMB := float64(freed) / (1024 * 1024)
		return map[string]any{"freed_mb": freedMB},
			fmt.Sprintf(msgT("PC 정리 완료! %.0fMB 확보됐습니다. 🗑️", "PC cleanup complete! %.0fMB freed. 🗑️", lang), freedMB)

	// ── 보안 탐지 ────────────────────────────────────────────
	case "security_scan":
		result := runSecurityScan()
		riskCount := 0
		for _, v := range result {
			if m, ok := v.(map[string]any); ok {
				if risk, ok := m["risk"].(string); ok && risk != "low" && risk != "none" { riskCount++ }
			}
		}
		if riskCount == 0 {
			return result, msgT("보안 점검 완료! 위협 요소가 발견되지 않았습니다. 🛡️", "Security scan complete! No threats found. 🛡️", lang)
		}
		return result, fmt.Sprintf(msgT("⚠️ 보안 경고: %d개 위협 요소가 발견됐습니다. 상세 결과를 확인하세요.", "⚠️ Security Alert: %d threat(s) detected. Please review the details.", lang), riskCount)

	// ── PC 통계 ──────────────────────────────────────────────
	case "stats":
		mem := getMemoryUsage()
		free, total := getDiskSpace()
		diskPct := 0
		if total > 0 { diskPct = int(100 - float64(free)/float64(total)*100) }
		stats := map[string]any{"mem": mem, "disk": diskPct}
		return stats, fmt.Sprintf(msgT("현재 PC 상태:\n  💾 RAM: %d%% 사용 중\n  💿 디스크(C:): %d%% 사용 중", "Current PC Status:\n  💾 RAM: %d%% used\n  💿 Disk (C:): %d%% used", lang), mem, diskPct)

	// ── 집중 모드 ────────────────────────────────────────────
	case "focus_mode":
		enable := boolVal("enable", true)
		r, _ := runFocusMode(enable)
		if enable {
			return r, msgT("집중 모드 켜졌습니다! 🎯\n알림이 차단됐습니다. 집중하세요!", "Focus mode ON! 🎯\nNotifications are blocked. Stay focused!", lang)
		}
		return r, msgT("집중 모드 꺼졌습니다. 알림이 다시 켜졌어요.", "Focus mode OFF. Notifications are back on.", lang)

	// ── 업무 일지 ────────────────────────────────────────────
	case "journal":
		j := buildJournalData(gKey, lang)
		return j, fmt.Sprintf(msgT("오늘 업무 일지 작성 완료! 📝\n%s", "Daily work log created! 📝\n%s", lang), j["summary"])

	// ── PC 건강 리포트 ───────────────────────────────────────
	case "health_report":
		reportPath, err := generateHealthReport(gKey, lang)
		if err != nil {
			return nil, msgT("리포트 생성 실패: ", "Report generation failed: ", lang) + err.Error()
		}
		return map[string]any{"path": reportPath}, msgT("PC 건강 리포트 생성 완료! 📊\n파일: ", "PC health report created! 📊\nFile: ", lang) + reportPath

	// ── 일정 등록 ────────────────────────────────────────────
	case "scheduler":
		command := str("command")
		if command == "" {
			command = original
		}
		parsed, err := parseNaturalSchedule(command, gKey)
		if err != nil {
			return nil, msgT("일정 파싱 실패: ", "Schedule parsing failed: ", lang) + err.Error()
		}
		paramsJSON, _ := json.Marshal(parsed.Params)
		task := &ScheduledTask{
			ID:           fmt.Sprintf("task_%d", time.Now().Unix()),
			Name:         parsed.TaskName,
			Command:      command,
			Action:       parsed.Action,
			ActionParams: string(paramsJSON),
			CronExpr:     parsed.CronExpr,
			NextRun:      parsed.NextRun,
			Active:       true,
			CreatedAt:    time.Now(),
		}
		globalScheduler.mu.Lock()
		globalScheduler.tasks[task.ID] = task
		globalScheduler.mu.Unlock()
		globalScheduler.save()
		return task, fmt.Sprintf(msgT("일정 등록 완료! ⏰\n'%s' (%s)", "Schedule registered! ⏰\n'%s' (%s)", lang), task.Name, task.CronExpr)

	// ── 앱 실행 ──────────────────────────────────────────────
	case "launch_app":
		appName := str("app_name")
		if appName == "" { appName = original }
		r, _ := runLaunchApp(appName)
		return r, fmt.Sprintf(msgT("%s 실행했습니다! 🚀", "Launched %s! 🚀", lang), appName)

	// ── 시스템 제어 ──────────────────────────────────────────
	case "system_control":
		control := str("control")
		value := intVal("value", -1)
		r, koMsg := runSystemControl(control, value)
		if lang == "en" {
			switch control {
			case "volume":
				return r, fmt.Sprintf("Volume set to %d%%. 🔊", value)
			case "mute":
				return r, "Muted. 🔇"
			case "brightness":
				return r, fmt.Sprintf("Brightness set to %d%%. ☀️", value)
			case "wifi":
				return r, "Wi-Fi turned on. 📶"
			case "sleep":
				return r, "Entering sleep mode. 💤"
			case "restart":
				return r, "Restarting in 10 seconds. 🔄"
			case "shutdown":
				return r, "Shutting down in 10 seconds. ⏻"
			default:
				return r, fmt.Sprintf("'%s' control executed.", control)
			}
		}
		return r, koMsg

	// ── 엑셀 저장 ────────────────────────────────────────────
	case "excel_save":
		title := str("title")
		rawData, _ := params["data"]
		var data [][]string
		if b, err := json.Marshal(rawData); err == nil {
			json.Unmarshal(b, &data)
		}
		if len(data) == 0 {
			return nil, msgT("저장할 데이터가 없어요.", "No data to save.", lang)
		}
		home, _ := os.UserHomeDir()
		savePath := fmt.Sprintf(`%s\Desktop\nexus_%s_%s.xlsx`,
			home, sanitizeFilename(title), time.Now().Format("20060102_150405"))
		if err := saveToExcel(data, savePath, title); err != nil {
			return nil, msgT("엑셀 저장 실패: ", "Excel save failed: ", lang) + err.Error()
		}
		return map[string]any{"path": savePath}, msgT("엑셀 저장 완료! 📊\n파일: ", "Excel saved! 📊\nFile: ", lang) + savePath

	// ── 메모 저장 ────────────────────────────────────────────
	case "note":
		content := str("content")
		if content == "" { content = original }
		notePath := saveQuickNote(content)
		return map[string]any{"path": notePath, "content": content}, msgT("메모 저장 완료! 📝", "Note saved! 📝", lang)

	// ── 일반 지식/대화 답변 (LLM 직접 응답) ─────────────────────
	case "general_answer":
		verticalCfg := loadVerticalConfig()
		verticalSys := VerticalSystemPrompts[verticalCfg.ID]
		if verticalSys == "" {
			verticalSys = VerticalSystemPrompts["general"]
		}
		var gaSys string
		if lang == "en" {
			gaSys = VerticalSystemPromptsEN[verticalCfg.ID]
			if gaSys == "" {
				gaSys = VerticalSystemPromptsEN["general"]
			}
		} else {
			gaSys = verticalSys
		}
		if lang == "en" {
			gaSys += fmt.Sprintf("\nCurrent time: %s (local)", time.Now().Format("2006-01-02 15:04"))
		} else {
			gaSys += fmt.Sprintf("\n현재 시각: %s (로컬)", time.Now().Format("2006-01-02 15:04"))
		}
		gaMsgs := []groqMsg{{Role: "system", Content: gaSys}}
		for _, h := range history {
			role := "user"
			if h.Role == "assistant" {
				role = "assistant"
			}
			content := h.Content
			if len([]rune(content)) > 300 {
				content = string([]rune(content)[:300]) + "..."
			}
			gaMsgs = append(gaMsgs, groqMsg{Role: role, Content: content})
		}
		gaMsgs = append(gaMsgs, groqMsg{Role: "user", Content: original})
		gaAns, gaCites, _ := callGroqWithCitations(gKey, groqChatModel, gaMsgs, 600)
		gaItems := make([]map[string]string, 0, len(gaCites))
		for _, u := range gaCites {
			gaItems = append(gaItems, map[string]string{"title": extractDomain(u), "url": u})
		}
		return map[string]any{"reply": gaAns, "items": gaItems}, gaAns

	default:
		// 분류 안 된 질문 → 이력 보완 후 web_search
		resolved := resolveWithHistory(original, history)
		cat := detectCategory(resolved)
		pr := parallelWebSearch(resolved, 5, lang)
		items := pr.Items
		if len(items) == 0 {
			items = categoryFallbackSites(resolved, cat)
		}
		msg := pr.Summary
		if msg == "" || containsBotBlockText(msg) {
			if cleaned := cleanPerplexityCall(resolved, gKey); cleaned != "" {
				msg = cleaned
			} else if msg == "" {
				msg = buildNoResultMessage(resolved, cat, "")
			}
		}
		return map[string]any{"query": resolved, "summary": msg, "items": items}, msg
	}
}

// ══════════════════════════════════════════════════════════════════
//  액션 구현 함수들
// ══════════════════════════════════════════════════════════════════

// runWebSearch: 웹 검색 → 결과 PDF/Excel/텍스트 저장
func runWebSearch(query, site, output string, maxItems int, gKey string, lang string) (any, string) {
	if maxItems == 0 {
		maxItems = 5
	}

	ctx, cancel, err := withStealthBrowserTimeout(3 * time.Minute)
	if err != nil {
		return nil, msgT("브라우저 시작 실패: ", "Browser start failed: ", lang) + err.Error()
	}
	defer cancel()

	products, _ := scrapeSearchResults(ctx, query, site, maxItems)
	if len(products) == 0 {
		// Tavily로 실시간 검색 시도
		llmMu.RLock()
		tKey := llmTavilyKey
		llmMu.RUnlock()
		if tKey != "" {
			if tr, ok := tavilySearch(tKey, query, maxItems); ok {
				return map[string]any{"summary": tr.Summary, "items": tr.Items}, tr.Summary
			}
		}
		products = generateFallbackProducts(query)
	}

	// AI 요약 (URL/출처 제외, 자연어 답변)
	var summary string
	if gKey != "" {
		lines := make([]string, 0, len(products))
		for _, p := range products {
			lines = append(lines, fmt.Sprintf("%s: %s — %s", p["rank"], p["name"], p["price"]))
		}
		kst := time.FixedZone("KST", 9*3600)
		today := time.Now().In(kst).Format("2006-01-02 15:04 KST")
		var prompt string
		if isEnglishQuery(query) {
			prompt = fmt.Sprintf(`Current time: %s
User question: "%s"
Search results:
%s

[Instructions]
- No URLs, links, or source names
- Answer the user's question directly in natural English, 2-4 sentences, key points only
- Include specific figures (price, rank, etc.)
- Act like a helpful AI assistant`, today, query, strings.Join(lines, "\n"))
		} else {
			prompt = fmt.Sprintf(`현재 시각(KST): %s
사용자 질문: "%s"
검색 결과:
%s

[지시사항]
- URL, 링크, 출처명 절대 포함 금지
- 사용자 질문에 직접 답하는 자연스러운 한국어 2~4문장으로 핵심만 요약
- 수치(가격, 등수 등)는 포함해도 됨
- 시간 언급 시 반드시 KST(한국 표준시) 기준 표현, UTC 절대 금지
- 친절한 AI 비서처럼 작성`, today, query, strings.Join(lines, "\n"))
		}
		s, _, _ := callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 512, false)
		if s == "" || containsBotBlockText(s) {
			if cleaned := cleanPerplexityCall(query, gKey); cleaned != "" {
				s = cleaned
			}
		}
		summary = s
	}

	htmlContent := buildProductHTML(query, products, summary)
	home, _ := os.UserHomeDir()
	safeName := sanitizeFilename(query)
	ts := time.Now().Format("20060102_150405")

	// 출력 형식에 따라 저장
	if output == "excel" {
		data := [][]string{{"순위", "제품명", "가격", "배송", "평점"}}
		for _, p := range products {
			data = append(data, []string{p["rank"], p["name"], p["price"], p["delivery"], p["rating"]})
		}
		xlsxPath := fmt.Sprintf(`%s\Desktop\%s_%s.xlsx`, home, safeName, ts)
		if err := saveToExcel(data, xlsxPath, query); err != nil {
			return nil, msgT("엑셀 저장 실패: ", "Excel save failed: ", lang) + err.Error()
		}
		return map[string]any{"path": xlsxPath, "count": len(products), "summary": summary},
			fmt.Sprintf(msgT("'%s' 검색 완료! %d개 수집 → 엑셀 저장됨\n%s\n파일: %s", "'%s' search complete! %d results collected → saved to Excel\n%s\nFile: %s", lang), query, len(products), summary, xlsxPath)
	}

	// 기본: HTML → PDF
	htmlPath := fmt.Sprintf(`%s\Desktop\%s_%s.html`, home, safeName, ts)
	pdfPath := fmt.Sprintf(`%s\Desktop\%s_%s.pdf`, home, safeName, ts)
	os.WriteFile(htmlPath, []byte(htmlContent), 0644)

	finalPath := htmlPath
	if pdfErr := chromeToPDF(ctx, htmlPath, pdfPath); pdfErr == nil {
		os.Remove(htmlPath)
		finalPath = pdfPath
	}

	msg := fmt.Sprintf(msgT("'%s' 검색 완료! %d개 결과 수집\n", "'%s' search complete! %d results collected\n", lang), query, len(products))
	if summary != "" {
		msg += summary + "\n"
	}
	msg += msgT("파일: ", "File: ", lang) + finalPath
	return map[string]any{"path": finalPath, "count": len(products), "summary": summary}, msg
}

// runSecurityScan: 보안 전반 점검
func runSecurityScan() map[string]any {
	result := map[string]any{}

	// 원격 접속 확인 (10초 타임아웃)
	out, _ := safePS(10*time.Second, `Get-NetTCPConnection | Where-Object {$_.State -eq 'Established' -and $_.RemoteAddress -notlike '127.*' -and $_.RemoteAddress -ne '::1'} | Select-Object LocalPort,RemoteAddress,RemotePort,OwningProcess | ConvertTo-Json -Compress -Depth 2`)
	var connections []map[string]any
	json.Unmarshal(out, &connections)
	result["remote_connections"] = connections
	result["connection_count"] = len(connections)

	// 의심 프로세스 확인 (10초 타임아웃)
	out2, _ := safePS(10*time.Second, `Get-Process | Where-Object {$_.CPU -gt 50} | Select-Object Name,Id,CPU | ConvertTo-Json -Compress`)
	var procs []map[string]any
	json.Unmarshal(out2, &procs)
	result["high_cpu_processes"] = procs

	// Windows Defender 상태 (10초 타임아웃)
	out3, _ := safePS(10*time.Second, `(Get-MpComputerStatus | Select-Object -Property AMServiceEnabled,AntispywareEnabled,RealTimeProtectionEnabled | ConvertTo-Json -Compress)`)
	var defender map[string]any
	json.Unmarshal(out3, &defender)
	result["defender"] = defender

	if len(connections) > 20 {
		result["risk"] = "high"
	} else if len(connections) > 10 {
		result["risk"] = "medium"
	} else {
		result["risk"] = "low"
	}
	return result
}

// runFocusMode: 집중 모드 켜기/끄기
func runFocusMode(enable bool) (any, string) {
	if enable {
		safePSRun(8*time.Second, `Set-ItemProperty -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Notifications\Settings' -Name 'NOC_GLOBAL_SETTING_TOASTS_ENABLED' -Value 0 -ErrorAction SilentlyContinue`)
		return map[string]any{"enabled": true}, "집중 모드 켜졌습니다! 🎯\n알림이 차단됐습니다. 집중하세요!"
	}
	safePSRun(8*time.Second, `Set-ItemProperty -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Notifications\Settings' -Name 'NOC_GLOBAL_SETTING_TOASTS_ENABLED' -Value 1 -ErrorAction SilentlyContinue`)
	return map[string]any{"enabled": false}, "집중 모드 꺼졌습니다. 알림이 다시 켜졌어요."
}

// buildJournalData: 오늘 업무 일지 생성
func buildJournalData(gKey string, lang string) map[string]any {
	today := time.Now().Format("2006-01-02")
	appUsage := getAppUsageToday()
	recentFiles := getRecentFiles(time.Now().Truncate(24 * time.Hour))

	summary := buildJournalSummary(today, appUsage, recentFiles, 0)

	// AI로 더 풍부한 일지 생성
	if gKey != "" && len(appUsage) > 0 {
		appNames := make([]string, 0)
		for _, a := range appUsage {
			appNames = append(appNames, a.Name)
		}
		var prompt string
		if lang == "en" {
			prompt = fmt.Sprintf("Apps used today (%s): %s\nFiles worked on: %d\n\nWrite a natural work journal entry for today in English. (3-5 sentences)",
				today, strings.Join(appNames, ", "), len(recentFiles))
		} else {
			prompt = fmt.Sprintf("오늘 %s에 사용한 앱: %s\n오늘 작업한 파일: %d개\n\n오늘 업무를 자연스럽게 일지로 작성해주세요. (3-5줄)",
				today, strings.Join(appNames, ", "), len(recentFiles))
		}
		aiSummary, _, _ := callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 512, false)
		if aiSummary != "" {
			summary = aiSummary
		}
	}

	return map[string]any{
		"date":         today,
		"summary":      summary,
		"app_usage":    appUsage,
		"recent_files": recentFiles,
	}
}

// generateHealthReport: PC 건강 리포트 PDF 생성
func generateHealthReport(gKey string, lang string) (string, error) {
	sr := buildScanResult()
	mem := getMemoryUsage()
	free, total := getDiskSpace()
	diskPct := 0
	if total > 0 {
		diskPct = int(100 - float64(free)/float64(total)*100)
	}

	var aiAnalysis string
	if gKey != "" {
		var prompt string
		if lang == "en" {
			prompt = fmt.Sprintf("PC Score: %d/100\nMemory: %d%%\nDisk: %d%%\nIssues found: %d\n\nWrite a brief PC health diagnosis in 3-4 sentences in English.",
				sr.Score, mem, diskPct, len(sr.Issues))
		} else {
			prompt = fmt.Sprintf("PC 점수: %d점\n메모리: %d%%\n디스크: %d%%\n문제: %d개\n\n간단한 PC 건강 진단 보고서를 3-4줄로 작성해주세요.",
				sr.Score, mem, diskPct, len(sr.Issues))
		}
		aiAnalysis, _, _ = callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 512, false)
	}

	issueRows := ""
	for _, issue := range sr.Issues {
		color := "#ffc107"
		if issue.Severity == "high" {
			color = "#dc3545"
		}
		issueRows += fmt.Sprintf(`<tr><td>%s</td><td style="color:%s">%s</td><td>%s</td></tr>`,
			issue.Title, color, issue.Severity, issue.Description)
	}
	if issueRows == "" {
		if lang == "en" {
			issueRows = `<tr><td colspan="3" style="text-align:center;color:#28a745">✅ All checks passed</td></tr>`
		} else {
			issueRows = `<tr><td colspan="3" style="text-align:center;color:#28a745">✅ 모든 항목 정상</td></tr>`
		}
	}

	var title, generated, ramLabel, diskLabel, issuesLabel, aiLabel, detailTitle, colItem, colSeverity, colDesc string
	if lang == "en" {
		title = "🖥️ Nexus PC Health Report"
		generated = "Generated: "
		ramLabel = "💾 RAM Usage"
		diskLabel = "💿 Disk (C:)"
		issuesLabel = "⚠️ Issues Found"
		aiLabel = "AI Diagnosis"
		detailTitle = "📋 Detailed Scan Results"
		colItem, colSeverity, colDesc = "Item", "Severity", "Description"
	} else {
		title = "🖥️ Nexus PC 건강 리포트"
		generated = "생성일시: "
		ramLabel = "💾 RAM 사용률"
		diskLabel = "💿 디스크(C:)"
		issuesLabel = "⚠️ 발견된 문제"
		aiLabel = "AI 진단"
		detailTitle = "📋 상세 점검 결과"
		colItem, colSeverity, colDesc = "항목", "심각도", "설명"
	}

	scoreUnit := "점"
	issueUnit := "개"
	if lang == "en" {
		scoreUnit = "/100"
		issueUnit = ""
	}

	html := fmt.Sprintf(`<!DOCTYPE html><html><head><meta charset="UTF-8">
<style>body{font-family:Arial,sans-serif;margin:40px;color:#333}
h1{color:#2c3e50;border-bottom:3px solid #3498db;padding-bottom:10px}
.score{font-size:72px;font-weight:bold;color:%s;text-align:center;margin:20px}
.grid{display:grid;grid-template-columns:1fr 1fr 1fr;gap:20px;margin:20px 0}
.card{background:#f8f9fa;border-radius:8px;padding:20px;text-align:center}
.card h3{margin:0;color:#666;font-size:14px}
.card p{margin:5px 0;font-size:32px;font-weight:bold;color:#2c3e50}
table{width:100%%;border-collapse:collapse;margin:20px 0}
th{background:#3498db;color:white;padding:10px}
td{padding:8px;border-bottom:1px solid #dee2e6}
.analysis{background:#e8f4fd;border-left:4px solid #3498db;padding:15px;margin:20px 0}
</style></head><body>
<h1>%s</h1>
<p>%s%s</p>
<div class="score">%d%s</div>
<div class="grid">
<div class="card"><h3>%s</h3><p>%d%%</p></div>
<div class="card"><h3>%s</h3><p>%d%%</p></div>
<div class="card"><h3>%s</h3><p>%d%s</p></div>
</div>
<div class="analysis"><strong>%s:</strong> %s</div>
<h2>%s</h2>
<table><thead><tr><th>%s</th><th>%s</th><th>%s</th></tr></thead>
<tbody>%s</tbody></table>
<p style="color:#999;font-size:12px;text-align:center">Nexus AI — PC Health Report</p>
</body></html>`,
		scoreColor(sr.Score), title, generated, time.Now().Format("2006-01-02 15:04:05"),
		sr.Score, scoreUnit, ramLabel, mem, diskLabel, diskPct, issuesLabel, len(sr.Issues), issueUnit,
		aiLabel, aiAnalysis, detailTitle, colItem, colSeverity, colDesc, issueRows)

	home, _ := os.UserHomeDir()
	htmlPath := filepath.Join(home, "Desktop", "nexus_health_report_"+time.Now().Format("20060102_150405")+".html")
	pdfPath := strings.Replace(htmlPath, ".html", ".pdf", 1)

	if err := os.WriteFile(htmlPath, []byte(html), 0644); err != nil {
		return "", err
	}

	ctx, cancel, err := withStealthBrowserTimeout(2 * time.Minute)
	if err != nil {
		return htmlPath, nil
	}
	defer cancel()

	if pdfErr := chromeToPDF(ctx, htmlPath, pdfPath); pdfErr == nil {
		os.Remove(htmlPath)
		return pdfPath, nil
	}
	return htmlPath, nil
}

func scoreColor(score int) string {
	if score >= 80 {
		return "#28a745"
	} else if score >= 60 {
		return "#ffc107"
	}
	return "#dc3545"
}

// runLaunchApp: 앱 실행 (Windows)
func runLaunchApp(appName string) (any, string) {
	// exe 이름 → 자연어 별칭 매핑
	appMap := map[string]string{
		// 브라우저
		"크롬": "chrome", "chrome": "chrome", "구글 크롬": "chrome",
		"엣지": "msedge", "edge": "msedge", "마이크로소프트 엣지": "msedge",
		"파이어폭스": "firefox", "firefox": "firefox",
		// Office
		"워드": "winword", "word": "winword", "마이크로소프트 워드": "winword",
		"엑셀": "excel", "마이크로소프트 엑셀": "excel",
		"파워포인트": "powerpnt", "ppt": "powerpnt", "powerpoint": "powerpnt",
		"아웃룩": "outlook", "outlook": "outlook",
		"원노트": "onenote", "onenote": "onenote",
		"팀즈": "teams", "teams": "teams", "ms teams": "teams",
		// 시스템
		"메모장": "notepad", "notepad": "notepad",
		"탐색기": "explorer", "파일탐색기": "explorer", "explorer": "explorer",
		"계산기": "calc", "calculator": "calc",
		"작업관리자": "taskmgr", "task manager": "taskmgr",
		"제어판": "control", "control panel": "control",
		"설정": "ms-settings:", "settings": "ms-settings:",
		"cmd": "cmd", "명령프롬프트": "cmd", "command prompt": "cmd",
		"파워쉘": "powershell", "powershell": "powershell",
		// 앱
		"카카오": "KakaoTalk", "카카오톡": "KakaoTalk", "kakaotalk": "KakaoTalk",
		"슬랙": "slack", "slack": "slack",
		"줌": "zoom", "zoom": "zoom",
		"디스코드": "discord", "discord": "discord",
		"노션": "notion", "notion": "notion",
		"비주얼스튜디오": "code", "vscode": "code", "vs code": "code",
		"메모": "notepad",
		"그림판": "mspaint", "paint": "mspaint",
		"스팀": "steam", "steam": "steam",
		"스포티파이": "spotify", "spotify": "spotify",
	}

	lower := strings.ToLower(strings.TrimSpace(appName))
	for k, v := range appMap {
		if strings.Contains(lower, strings.ToLower(k)) {
			// ms-settings: 는 start 없이 직접 실행
			if strings.HasPrefix(v, "ms-") {
				newHiddenCmd("cmd", "/c", "start", v).Start()
			} else {
				newHiddenCmd("cmd", "/c", "start", "", v).Start()
			}
			return map[string]any{"app": v, "requested": appName}, fmt.Sprintf("%s 실행했습니다! 🚀", appName)
		}
	}
	// 알 수 없는 앱: 직접 start 시도 (설치된 앱이면 동작)
	newHiddenCmd("cmd", "/c", "start", "", appName).Start()
	return map[string]any{"app": appName}, fmt.Sprintf("'%s' 실행을 시도했습니다.", appName)
}

// runSystemControl: 볼륨/밝기/와이파이 등 시스템 제어
func runSystemControl(control string, value int) (any, string) {
	switch strings.ToLower(control) {
	case "volume", "볼륨":
		if value < 0 {
			value = 50
		}
		script := fmt.Sprintf(`Add-Type -TypeDefinition 'using System.Runtime.InteropServices; public class V{[DllImport("winmm.dll")]public static extern int waveOutSetVolume(System.IntPtr h,uint v);}';$v=[uint32](%d/100.0*65535);[V]::waveOutSetVolume([System.IntPtr]::Zero,($v -bor ($v -shl 16)))`, value)
		execPSRun(script)
		return map[string]any{"volume": value}, fmt.Sprintf("볼륨을 %d%%로 설정했습니다. 🔊", value)

	case "mute", "음소거":
		safePSRun(5*time.Second, `(New-Object -ComObject WScript.Shell).SendKeys([char]173)`)
		return map[string]any{"muted": true}, "음소거 처리했습니다. 🔇"

	case "brightness", "밝기":
		if value < 0 {
			value = 70
		}
		script := fmt.Sprintf(`(Get-WmiObject -Namespace root/WMI -Class WmiMonitorBrightnessMethods).WmiSetBrightness(1,%d)`, value)
		execPSRun(script)
		return map[string]any{"brightness": value}, fmt.Sprintf("밝기를 %d%%로 설정했습니다. ☀️", value)

	case "wifi", "와이파이":
		safePSRun(10*time.Second, `(Get-NetAdapter | Where-Object {$_.InterfaceDescription -like '*Wi-Fi*' -or $_.Name -like '*Wi-Fi*'} | Enable-NetAdapter -Confirm:$false) 2>$null`)
		return map[string]any{"wifi": "enabled"}, "Wi-Fi를 켰습니다. 📶"

	case "sleep", "절전":
		safePSRun(8*time.Second, `Add-Type -Assembly System.Windows.Forms; [System.Windows.Forms.Application]::SetSuspendState('Suspend',$false,$false)`)
		return map[string]any{"sleep": true}, "절전 모드로 전환합니다. 💤"

	case "restart", "재시작":
		// 안전장치: 백엔드에서 직접 재시작 금지 — 프론트엔드 confirm 필요
		return map[string]any{"requires_confirm": true, "action": "restart"},
			"⚠️ 재시작하면 작업 중인 내용이 저장되지 않을 수 있습니다. 프론트엔드에서 확인 후 진행하세요."

	case "restart_confirmed", "재시작_확인됨":
		ctx10, c10 := context.WithTimeout(context.Background(), 5*time.Second)
		defer c10()
		newHiddenCmdCtx(ctx10, "shutdown", "/r", "/t", "10").Run()
		return map[string]any{"restart": true}, "10초 후 재시작합니다. 🔄"

	case "shutdown", "종료":
		return map[string]any{"requires_confirm": true, "action": "shutdown"},
			"⚠️ 종료하면 작업 중인 내용이 저장되지 않을 수 있습니다. 프론트엔드에서 확인 후 진행하세요."

	case "shutdown_confirmed", "종료_확인됨":
		ctx10, c10 := context.WithTimeout(context.Background(), 5*time.Second)
		defer c10()
		newHiddenCmdCtx(ctx10, "shutdown", "/s", "/t", "10").Run()
		return map[string]any{"shutdown": true}, "10초 후 종료합니다. ⏻"
	}
	return nil, fmt.Sprintf("'%s' 제어를 수행할 수 없습니다.", control)
}

// organizeFolder: 폴더 파일을 유형별로 분류
func organizeFolder(folder string) (float64, int) {
	entries, err := os.ReadDir(folder)
	if err != nil {
		return 0, 0
	}

	extMap := map[string]string{
		".jpg": "사진", ".jpeg": "사진", ".png": "사진", ".gif": "사진", ".bmp": "사진", ".webp": "사진",
		".mp4": "동영상", ".avi": "동영상", ".mov": "동영상", ".mkv": "동영상",
		".mp3": "음악", ".wav": "음악", ".flac": "음악",
		".pdf": "문서", ".docx": "문서", ".doc": "문서", ".txt": "문서",
		".xlsx": "스프레드시트", ".xls": "스프레드시트", ".csv": "스프레드시트",
		".pptx": "프레젠테이션", ".ppt": "프레젠테이션",
		".zip": "압축파일", ".rar": "압축파일", ".7z": "압축파일",
		".exe": "프로그램", ".msi": "프로그램",
	}

	count := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		subDir, ok := extMap[ext]
		if !ok {
			subDir = "기타"
		}
		targetDir := filepath.Join(folder, subDir)
		os.MkdirAll(targetDir, 0755)
		src := filepath.Join(folder, e.Name())
		dst := filepath.Join(targetDir, e.Name())
		if err := os.Rename(src, dst); err == nil {
			count++
		}
	}
	return 0, count
}

// saveQuickNote: 빠른 메모 저장
func saveQuickNote(content string) string {
	home, _ := os.UserHomeDir()
	notesDir := filepath.Join(home, "Documents", "Nexus메모")
	os.MkdirAll(notesDir, 0755)
	path := filepath.Join(notesDir, "메모_"+time.Now().Format("20060102_150405")+".txt")
	os.WriteFile(path, []byte(content), 0644)
	return path
}

// buildScanResult: PC 현재 상태 분석
func buildScanResult() ScanResult {
	var issues []Issue
	score := 100

	tempSize := getTempSize()
	if tempSize > 500<<20 {
		issues = append(issues, Issue{
			ID: "temp-files", Title: formatBytes(tempSize) + " 임시 파일이 쌓여있어요",
			Description: "정리하면 디스크 공간을 확보할 수 있어요", Severity: "medium", Category: "clean", Fixable: true,
		})
		score -= 10
	}
	free, total := getDiskSpace()
	if total > 0 && float64(free)/float64(total) < 0.1 {
		issues = append(issues, Issue{
			ID: "disk-space", Title: "디스크 공간 부족 (" + formatBytes(int64(free)) + " 남음)",
			Description: "불필요한 파일을 정리하세요", Severity: "high", Category: "disk", Fixable: false,
		})
		score -= 20
	}
	memUsage := getMemoryUsage()
	if memUsage > 85 {
		issues = append(issues, Issue{
			ID: "memory", Title: fmt.Sprintf("메모리 사용량 %d%% 높음", memUsage),
			Description: "불필요한 프로그램을 종료하면 빨라져요", Severity: "medium", Category: "memory", Fixable: false,
		})
		score -= 5
	}
	if score < 0 {
		score = 0
	}
	return ScanResult{Score: score, Issues: issues}
}

// ── 유틸 ────────────────────────────────────────────────────────

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// resolveWithHistory: 이전 대화 이력을 참고해 모호한 질문을 구체화
// 예) "버스 시간 알려줘" + 이전 대화 "부산 정관→인천터미널 버스" → "부산 정관에서 인천터미널 버스 시간"
// containsBotBlockText: LLM 응답에 봇 차단 언급이 있는지 확인
func containsBotBlockText(s string) bool {
	signals := []string{
		"봇 차단", "봇을 감지", "봇으로 감지", "차단으로 인해", "접근이 차단", "접근 불가",
		"차단되어", "차단되었", "수집할 수 없", "수집이 불가", "가져올 수 없",
		"결과를 제공할 수 없", "정보를 제공할 수 없", "검색 결과가 차단",
		"bot detected", "bot blocked", "access denied", "403 forbidden",
		"자동화된 접근", "비정상적인 트래픽", "captcha", "보안 문자",
		"스크래핑이 차단", "크롤링이 차단",
	}
	lower := strings.ToLower(s)
	for _, sig := range signals {
		if strings.Contains(lower, strings.ToLower(sig)) {
			return true
		}
	}
	return false
}

// cleanPerplexityCall: 봇 차단 없이 Perplexity에 직접 질문 (Mac 스타일 폴백)
func cleanPerplexityCall(query, gKey string) string {
	kst := time.FixedZone("KST", 9*3600)
	today := time.Now().In(kst).Format("2006-01-02 15:04 KST")
	var sys string
	if isEnglishQuery(query) {
		sys = "You are Nexus AI assistant. Use real-time web search to find accurate, up-to-date information and answer in natural English in 2-4 sentences. No markdown headers. Never say 'bot blocked', 'access denied', or 'unable to retrieve'. If no info, guide the user to the official site or app."
	} else {
		sys = "당신은 Nexus AI 한국어 비서입니다. 실시간 웹 검색으로 정확한 최신 정보를 찾아 자연스러운 한국어로 2~4문장 답변하세요. 마크다운 헤더 금지. '봇 차단', '접근 불가', '차단으로 인해' 같은 표현 절대 금지. 정보 없으면 공식 사이트나 앱 이용을 안내하세요."
	}
	msgs := []groqMsg{
		{Role: "system", Content: sys},
		{Role: "user", Content: fmt.Sprintf("Current time: %s\n%s", today, query)},
	}
	text, _, err := callGroqWithFallback(msgs, 512, false)
	if err != nil || text == "" {
		return ""
	}
	return text
}

func resolveWithHistory(current string, history []ConvHistoryMsg) string {
	if len(history) == 0 {
		return current
	}
	// 명시적 대명사/지시어가 있을 때만 이전 대화 컨텍스트 참조
	// (짧은 질문이라는 이유만으로 이력을 붙이면 엉뚱한 라우팅 발생)
	isVague := strings.Contains(current, "그거") ||
		strings.Contains(current, "이거") ||
		strings.Contains(current, "그때") ||
		strings.Contains(current, "거기") ||
		strings.Contains(current, "아까") ||
		strings.Contains(current, "그 버스") ||
		strings.Contains(current, "그 노선") ||
		strings.Contains(current, "더 알려") ||
		strings.Contains(current, "자세히 알려")
	if !isVague {
		return current
	}
	// 직전 2턴만 참조 (4턴은 컨텍스트 오염 위험)
	var contextParts []string
	start := len(history) - 2
	if start < 0 {
		start = 0
	}
	for _, h := range history[start:] {
		if h.Content == "" {
			continue
		}
		content := h.Content
		if len([]rune(content)) > 100 {
			content = string([]rune(content)[:100])
		}
		contextParts = append(contextParts, content)
	}
	if len(contextParts) == 0 {
		return current
	}
	return strings.Join(contextParts, " / ") + " → " + current
}

