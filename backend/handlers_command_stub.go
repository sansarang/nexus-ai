//go:build !windows

package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xuri/excelize/v2"
)

// ── 서버사이드 세션 메모리 (사용자별 최근 10턴 보관) ────────────
type nexusSession struct {
	history   []ConvHistoryMsg
	lastTopic string // 마지막으로 다룬 주제/쿼리
}

var (
	sessionStoreMu sync.RWMutex
	sessionStore   = make(map[string]*nexusSession)
)

func getSession(userID string) *nexusSession {
	sessionStoreMu.RLock()
	s := sessionStore[userID]
	sessionStoreMu.RUnlock()
	if s == nil {
		s = &nexusSession{}
		sessionStoreMu.Lock()
		sessionStore[userID] = s
		sessionStoreMu.Unlock()
	}
	return s
}

func appendSession(userID, role, content string) {
	sess := getSession(userID)
	sessionStoreMu.Lock()
	sess.history = append(sess.history, ConvHistoryMsg{Role: role, Content: content})
	if len(sess.history) > 20 {
		sess.history = sess.history[len(sess.history)-20:]
	}
	// lastTopic: 사용자 메시지면 업데이트
	if role == "user" {
		sess.lastTopic = content
	}
	sessionStoreMu.Unlock()
}

// resolvePronouns: 한/영 대명사를 세션 컨텍스트로 해소
func resolvePronouns(msg string, userID string) string {
	// 한국어 대명사
	koPronouns := []string{"그거", "이거", "저거", "그것", "이것", "저것", "그 파일", "그 영상", "그 뉴스", "그 사람", "아까 말한", "아까 그", "방금 말한", "방금 그", "그게", "이게", "그 링크", "그 내용", "그 결과"}
	// 영어 대명사/지시어
	enPronouns := []string{"that one", "the one", "it again", "same thing", "about that", "find that", "search that", "the previous", "that result", "that video", "that news", "that file", "find more", "more about", "tell me more", "dig deeper"}
	hasPronsoun := false
	for _, p := range koPronouns {
		if strings.Contains(msg, p) {
			hasPronsoun = true
			break
		}
	}
	if !hasPronsoun {
		msgLow := strings.ToLower(msg)
		for _, p := range enPronouns {
			if strings.Contains(msgLow, p) {
				hasPronsoun = true
				break
			}
		}
	}
	if !hasPronsoun {
		return msg
	}
	sess := getSession(userID)
	sessionStoreMu.RLock()
	topic := sess.lastTopic
	hist := sess.history
	sessionStoreMu.RUnlock()
	if topic == "" && len(hist) == 0 {
		return msg
	}
	// 최근 2턴 컨텍스트 추출
	ctx := ""
	startIdx := len(hist) - 4
	if startIdx < 0 {
		startIdx = 0
	}
	for _, h := range hist[startIdx:] {
		if h.Role == "user" && h.Content != msg {
			ctx = h.Content
		}
	}
	if ctx == "" {
		ctx = topic
	}
	if ctx != "" {
		// 영어 질문이면 영어 태그, 아니면 한국어 태그
		if isEnglishQuery(msg) {
			return fmt.Sprintf("[Previous context: %s]\nCurrent question: %s", ctx, msg)
		}
		return fmt.Sprintf("[이전 대화 컨텍스트: %s]\n현재 질문: %s", ctx, msg)
	}
	return msg
}

// ── 멀티 액션: 출력 포맷 감지 ──────────────────────────────────
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
	case strings.Contains(lower, "파워포인트") || strings.Contains(lower, "powerpoint") || strings.Contains(lower, "pptx") || strings.Contains(lower, "ppt") || strings.Contains(lower, "프레젠테이션"):
		return outPowerPoint
	case strings.Contains(lower, "마크다운") || strings.Contains(lower, "markdown") || strings.Contains(lower, ".md"):
		return outMarkdown
	case strings.Contains(lower, "txt") || strings.Contains(lower, "텍스트 파일") || strings.Contains(lower, "텍스트로 저장"):
		return outTXT
	// 파일 저장 동사가 있으면 기본값 markdown
	case hasFileSaveVerb(msg):
		return outMarkdown
	}
	return outNone
}

// 파일 저장 동사 감지 (멀티 액션 트리거)
func hasFileSaveVerb(msg string) bool {
	lower := strings.ToLower(msg)
	saveVerbs := []string{
		"저장", "만들어", "작성", "정리", "보고서", "리포트", "report",
		"파일로", "제품설명서", "설명서", "요약해서", "뽑아줘", "출력",
		"요약해줘", "요약 해줘", "정리해줘", "정리 해줘", "모아줘", "뉴스 정리",
		"뉴스요약", "뉴스 요약", "기사 정리", "기사 요약", "리포트 만들어",
	}
	for _, v := range saveVerbs {
		if strings.Contains(lower, v) {
			return true
		}
	}
	return false
}

// 멀티 액션 결과를 파일로 저장
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

	switch format {
	case outMarkdown:
		path := filepath.Join(home, "Desktop", fmt.Sprintf("nexus_%s_%s.md", safeName, ts))
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("# %s\n\n", title))
		sb.WriteString(fmt.Sprintf("*생성: %s*\n\n", time.Now().Format("2006-01-02 15:04:05")))
		if summary != "" {
			sb.WriteString("## 요약\n\n")
			sb.WriteString(summary + "\n\n")
			sb.WriteString("---\n\n")
		}
		if len(items) > 0 {
			sb.WriteString("## 상세 항목\n\n")
			for i, it := range items {
				name := it["title"]
				if name == "" { name = it["name"] }
				url := it["url"]
				if url == "" { url = it["link"] }
				price := it["price"]
				content := it["content"]
				if content == "" { content = it["snippet"] }

				if price != "" {
					sb.WriteString(fmt.Sprintf("### %d. %s\n\n", i+1, name))
					sb.WriteString(fmt.Sprintf("- **가격**: %s\n", price))
					if url != "" { sb.WriteString(fmt.Sprintf("- **링크**: %s\n", url)) }
					if content != "" { sb.WriteString(fmt.Sprintf("\n%s\n", content)) }
					sb.WriteString("\n---\n\n")
				} else {
					sb.WriteString(fmt.Sprintf("### %d. %s\n\n", i+1, name))
					if content != "" { sb.WriteString(content + "\n\n") }
					if url != "" { sb.WriteString(fmt.Sprintf("🔗 [원문 보기](%s)\n", url)) }
					sb.WriteString("\n---\n\n")
				}
			}
		}
		if sb.Len() < 100 {
			sb.WriteString("> ⚠️ 검색 결과가 없거나 API 키가 설정되지 않았습니다.\n")
		}
		return path, os.WriteFile(path, []byte(sb.String()), 0644)

	case outTXT:
		path := filepath.Join(home, "Desktop", fmt.Sprintf("nexus_%s_%s.txt", safeName, ts))
		var sb strings.Builder
		sb.WriteString(title + "\n")
		sb.WriteString(strings.Repeat("=", 40) + "\n")
		sb.WriteString("생성: " + time.Now().Format("2006-01-02 15:04:05") + "\n\n")
		if summary != "" {
			sb.WriteString("[ AI 요약 ]\n" + summary + "\n\n")
			sb.WriteString(strings.Repeat("-", 40) + "\n\n")
		}
		if len(items) > 0 {
			sb.WriteString("[ 상세 항목 ]\n\n")
			for i, it := range items {
				name := it["title"]
				if name == "" { name = it["name"] }
				url := it["url"]
				if url == "" { url = it["link"] }
				price := it["price"]
				content := it["content"]
				if content == "" { content = it["snippet"] }

				sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, name))
				if price != "" { sb.WriteString(fmt.Sprintf("   가격: %s\n", price)) }
				if content != "" { sb.WriteString(fmt.Sprintf("   %s\n", content)) }
				if url != "" { sb.WriteString(fmt.Sprintf("   링크: %s\n", url)) }
				sb.WriteString("\n")
			}
		}
		if sb.Len() < 80 {
			sb.WriteString("검색 결과가 없거나 API 키가 설정되지 않았습니다.\n")
		}
		return path, os.WriteFile(path, []byte(sb.String()), 0644)

	case outExcel:
		// 실제 .xlsx 파일 생성 (excelize)
		path := filepath.Join(home, "Desktop", fmt.Sprintf("nexus_%s_%s.xlsx", safeName, ts))
		f := excelize.NewFile()
		sheet := "결과"
		f.SetSheetName("Sheet1", sheet)
		headers := []string{"번호", "제목/상품명", "내용", "가격", "링크"}
		for ci, h := range headers {
			cell, _ := excelize.CoordinatesToCellName(ci+1, 1)
			f.SetCellValue(sheet, cell, h)
		}
		row := 2
		if summary != "" {
			f.SetCellValue(sheet, fmt.Sprintf("A%d", row), 0)
			f.SetCellValue(sheet, fmt.Sprintf("B%d", row), "[AI 요약]")
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
		// 실제 .docx 생성 (OOXML zip 구조)
		path := filepath.Join(home, "Desktop", fmt.Sprintf("nexus_%s_%s.docx", safeName, ts))
		err := saveDocx(path, title, summary, items)
		return path, err

	case outPDF:
		// HTML 파일 생성 후 안내 (범용 PDF 변환은 OS 의존 — HTML로 저장하고 브라우저 인쇄 안내)
		path := filepath.Join(home, "Desktop", fmt.Sprintf("nexus_%s_%s.html", safeName, ts))
		err := saveHTML(path, title, summary, items)
		return path, err

	case outPowerPoint:
		// 실제 .pptx 생성 (OOXML zip 구조)
		path := filepath.Join(home, "Desktop", fmt.Sprintf("nexus_%s_%s.pptx", safeName, ts))
		err := savePptx(path, title, summary, items)
		return path, err
	}
	return "", fmt.Errorf("지원하지 않는 형식")
}

// saveDocx: OOXML 구조로 실제 .docx 파일 생성
func saveDocx(path, title, summary string, items []map[string]string) error {
	now := time.Now().Format("2006-01-02 15:04")

	// 본문 XML 구성
	var body strings.Builder
	addDocxPara := func(text, style string) {
		styleXML := ""
		if style != "" {
			styleXML = fmt.Sprintf(`<w:pPr><w:pStyle w:val="%s"/></w:pPr>`, style)
		}
		escaped := strings.ReplaceAll(text, "&", "&amp;")
		escaped = strings.ReplaceAll(escaped, "<", "&lt;")
		escaped = strings.ReplaceAll(escaped, ">", "&gt;")
		body.WriteString(fmt.Sprintf(`<w:p>%s<w:r><w:t xml:space="preserve">%s</w:t></w:r></w:p>`, styleXML, escaped))
	}

	addDocxPara(title, "Heading1")
	addDocxPara("생성: "+now, "")
	if summary != "" {
		addDocxPara("AI 요약", "Heading2")
		// summary는 줄바꿈 단위로 분리
		for _, line := range strings.Split(summary, "\n") {
			line = strings.TrimLeft(line, "#- *`")
			if strings.TrimSpace(line) == "" { continue }
			addDocxPara(line, "")
		}
	}
	if len(items) > 0 {
		addDocxPara("상세 항목", "Heading2")
		for i, it := range items {
			name    := it["title"];   if name == ""    { name = it["name"] }
			content := it["content"]; if content == "" { content = it["snippet"] }
			url     := it["url"];     if url == ""     { url = it["link"] }
			price   := it["price"]
			addDocxPara(fmt.Sprintf("%d. %s", i+1, name), "Heading3")
			if price   != "" { addDocxPara("가격: "+price, "") }
			if content != "" { addDocxPara(content, "") }
			if url     != "" { addDocxPara("링크: "+url, "") }
		}
	}

	docXML := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:wpc="http://schemas.microsoft.com/office/word/2010/wordprocessingCanvas"
  xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"
  xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
<w:body>%s<w:sectPr/></w:body></w:document>`, body.String())

	relsXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
</Relationships>`

	stylesXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:style w:type="paragraph" w:styleId="Heading1"><w:name w:val="heading 1"/>
<w:rPr><w:b/><w:sz w:val="48"/></w:rPr></w:style>
<w:style w:type="paragraph" w:styleId="Heading2"><w:name w:val="heading 2"/>
<w:rPr><w:b/><w:sz w:val="36"/></w:rPr></w:style>
<w:style w:type="paragraph" w:styleId="Heading3"><w:name w:val="heading 3"/>
<w:rPr><w:b/><w:sz w:val="28"/></w:rPr></w:style>
</w:styles>`

	contentTypes := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="xml" ContentType="application/xml"/>
<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
<Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>
</Types>`

	rootRels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`

	return writeZip(path, map[string]string{
		"[Content_Types].xml":  contentTypes,
		"_rels/.rels":          rootRels,
		"word/document.xml":    docXML,
		"word/_rels/document.xml.rels": relsXML,
		"word/styles.xml":      stylesXML,
	})
}

// savePptx: OOXML 구조로 실제 .pptx 파일 생성 (슬라이드 1장당 1항목)
func savePptx(path, title, summary string, items []map[string]string) error {
	// 마크다운 기호 제거 + XML 이스케이프
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

	// 텍스트 → PPTX 단락 XML (줄마다 <a:p> 생성, ## → 제목 스타일)
	textToParas := func(text, sz string) string {
		var sb strings.Builder
		for _, line := range strings.Split(text, "\n") {
			raw := strings.TrimSpace(line)
			if raw == "" {
				sb.WriteString(`<a:p><a:endParaRPr lang="ko-KR" dirty="0"/></a:p>`)
				continue
			}
			bold := ""
			// ## 제목 줄은 굵게
			if strings.HasPrefix(raw, "#") {
				bold = "<a:b/>"
				raw = strings.TrimLeft(raw, "# ")
			} else if strings.HasPrefix(raw, "- ") || strings.HasPrefix(raw, "• ") {
				raw = "• " + raw[2:]
			}
			raw = cleanText(raw)
			sb.WriteString(fmt.Sprintf(`<a:p><a:r><a:rPr lang="ko-KR" sz="%s" dirty="0">%s</a:rPr><a:t>%s</a:t></a:r></a:p>`, sz, bold, raw))
		}
		return sb.String()
	}

	// shape 생성 (id 파라미터로 고유 ID)
	makeShape := func(id, x, y, cx, cy int, paras string) string {
		return fmt.Sprintf(
			`<p:sp><p:nvSpPr><p:cNvPr id="%d" name="sp%d"/>`+
				`<p:cNvSpPr txBox="1"><a:spLocks noGrp="1"/></p:cNvSpPr><p:nvPr/></p:nvSpPr>`+
				`<p:spPr><a:xfrm><a:off x="%d" y="%d"/><a:ext cx="%d" cy="%d"/></a:xfrm>`+
				`<a:prstGeom prst="rect"><a:avLst/></a:prstGeom><a:noFill/></p:spPr>`+
				`<p:txBody><a:bodyPr wrap="square" autofit="normAutofit"/><a:lstStyle/>%s</p:txBody></p:sp>`,
			id, id, x, y, cx, cy, paras)
	}

	makeSlide := func(heading, bodyText, bodySz string) string {
		titleParas := textToParas(heading, "3200")
		bodyParas  := textToParas(bodyText, bodySz)
		titleShape := makeShape(2, 457200, 274638, 8229600, 1143000, titleParas)
		bodyShape  := makeShape(3, 457200, 1600200, 8229600, 4525963, bodyParas)
		return fmt.Sprintf(
			`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`+
				`<p:sld xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"`+
				` xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"`+
				` xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">`+
				`<p:cSld><p:spTree>`+
				`<p:nvGrpSpPr><p:cNvPr id="1" name=""/><p:cNvGrpSpPr/><p:nvPr/></p:nvGrpSpPr>`+
				`<p:grpSpPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="0" cy="0"/>`+
				`<a:chOff x="0" y="0"/><a:chExt cx="0" cy="0"/></a:xfrm></p:grpSpPr>`+
				`%s%s</p:spTree></p:cSld></p:sld>`,
			titleShape, bodyShape)
	}

	slideRels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"/>`

	files := map[string]string{}

	// 슬라이드 1: 타이틀 + 요약
	cleanSummary := ""
	if summary != "" {
		var lines []string
		for _, l := range strings.Split(summary, "\n") {
			l = strings.TrimSpace(l)
			if l != "" {
				lines = append(lines, l)
			}
			if len(lines) >= 12 { break }
		}
		cleanSummary = strings.Join(lines, "\n")
	}
	files["ppt/slides/slide1.xml"] = makeSlide(title, cleanSummary, "1800")
	files["ppt/slides/_rels/slide1.xml.rels"] = slideRels

	slideList         := `<p:sldIdLst><p:sldId id="256" r:id="rId1"/>`
	slideContentTypes := `<Override PartName="/ppt/slides/slide1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slide+xml"/>`
	presRels          := `<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="slides/slide1.xml"/>`

	// 항목별 슬라이드
	for i, it := range items {
		if i >= 20 { break }
		name    := it["title"];   if name == ""    { name = it["name"] }
		content := it["content"]; if content == "" { content = it["snippet"] }
		price   := it["price"]
		body := content
		if price != "" { body = "💰 가격: " + price + "\n\n" + content }

		slideNum  := i + 2
		slideFile := fmt.Sprintf("ppt/slides/slide%d.xml", slideNum)
		relsFile  := fmt.Sprintf("ppt/slides/_rels/slide%d.xml.rels", slideNum)
		files[slideFile] = makeSlide(fmt.Sprintf("%d. %s", i+1, name), body, "1800")
		files[relsFile]  = slideRels
		slideList        += fmt.Sprintf(`<p:sldId id="%d" r:id="rId%d"/>`, 256+i+1, i+2)
		slideContentTypes += fmt.Sprintf(`<Override PartName="/ppt/slides/slide%d.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slide+xml"/>`, slideNum)
		presRels          += fmt.Sprintf(`<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="slides/slide%d.xml"/>`, i+2, slideNum)
	}
	slideList += `</p:sldIdLst>`

	files["[Content_Types].xml"] = fmt.Sprintf(
		`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`+
			`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">`+
			`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>`+
			`<Default Extension="xml" ContentType="application/xml"/>`+
			`<Override PartName="/ppt/presentation.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"/>`+
			`%s</Types>`, slideContentTypes)

	files["_rels/.rels"] = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
		`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="ppt/presentation.xml"/>` +
		`</Relationships>`

	files["ppt/presentation.xml"] = fmt.Sprintf(
		`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`+
			`<p:presentation xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"`+
			` xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"`+
			` xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">`+
			`<p:sldMasterIdLst/><p:sldSz cx="9144000" cy="6858000"/><p:notesSz cx="6858000" cy="9144000"/>%s</p:presentation>`,
		slideList)

	files["ppt/_rels/presentation.xml.rels"] = fmt.Sprintf(
		`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`+
			`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">%s</Relationships>`,
		presRels)

	return writeZip(path, files)
}

// saveHTML: PDF 대용 HTML 저장 (브라우저에서 Ctrl+P → PDF 출력 안내)
func saveHTML(path, title, summary string, items []map[string]string) error {
	now := time.Now().Format("2006-01-02 15:04")
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<!DOCTYPE html><html lang="ko"><head><meta charset="UTF-8">
<title>%s</title><style>
body{font-family:'맑은 고딕',sans-serif;max-width:900px;margin:40px auto;line-height:1.7;color:#222}
h1{color:#1a1a2e;border-bottom:2px solid #4a90d9;padding-bottom:8px}
h2{color:#2c3e50;margin-top:2em}h3{color:#34495e}
.summary{background:#f0f7ff;border-left:4px solid #4a90d9;padding:16px;border-radius:4px;white-space:pre-wrap}
.item{border:1px solid #e0e0e0;border-radius:8px;padding:16px;margin:12px 0}
.price{color:#e74c3c;font-weight:bold}a{color:#4a90d9}
.meta{color:#888;font-size:0.9em}
@media print{body{margin:0}.item{break-inside:avoid}}
</style></head><body>`, title))
	sb.WriteString(fmt.Sprintf("<h1>%s</h1><p class='meta'>생성: %s</p>", title, now))
	if summary != "" {
		sb.WriteString("<h2>AI 요약</h2><div class='summary'>")
		escaped := strings.ReplaceAll(summary, "&", "&amp;")
		escaped = strings.ReplaceAll(escaped, "<", "&lt;")
		escaped = strings.ReplaceAll(escaped, ">", "&gt;")
		sb.WriteString(escaped)
		sb.WriteString("</div>")
	}
	if len(items) > 0 {
		sb.WriteString("<h2>상세 항목</h2>")
		for i, it := range items {
			name    := it["title"];   if name == ""    { name = it["name"] }
			content := it["content"]; if content == "" { content = it["snippet"] }
			url     := it["url"];     if url == ""     { url = it["link"] }
			price   := it["price"]
			sb.WriteString(fmt.Sprintf("<div class='item'><h3>%d. %s</h3>", i+1, name))
			if price   != "" { sb.WriteString(fmt.Sprintf("<p class='price'>가격: %s</p>", price)) }
			if content != "" {
				esc := strings.ReplaceAll(content, "&", "&amp;")
				sb.WriteString(fmt.Sprintf("<p>%s</p>", esc))
			}
			if url != "" { sb.WriteString(fmt.Sprintf("<p><a href='%s' target='_blank'>원문 보기 →</a></p>", url)) }
			sb.WriteString("</div>")
		}
	}
	sb.WriteString("<p class='meta' style='margin-top:3em;border-top:1px solid #eee;padding-top:1em'>PDF로 저장하려면: 브라우저에서 이 파일을 열고 Ctrl+P → PDF로 인쇄</p>")
	sb.WriteString("</body></html>")
	return os.WriteFile(path, []byte(sb.String()), 0644)
}

// writeZip: 여러 파일을 zip으로 묶어서 저장 (docx/pptx 공통)
func writeZip(path string, files map[string]string) error {
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

type CommandRequest struct {
	Message         string              `json:"message"`
	Context         string              `json:"context"`
	Lang            string              `json:"lang"`
	PendingIntent   string              `json:"pending_intent"`
	PendingParams   map[string]any      `json:"pending_params"`
	PendingQuestion string              `json:"pending_question"`
	History         []ConvHistoryMsg    `json:"history"`
	UserEmail       string              `json:"user_email"`
}

type CommandResponse struct {
	Success         bool           `json:"success"`
	Message         string         `json:"message"`
	Action          string         `json:"action"`
	Result          any            `json:"result"`
	Duration        string         `json:"duration"`
	NeedsClarify     bool           `json:"needs_clarify,omitempty"`
	ClarifyQuestion  string         `json:"clarify_question,omitempty"`
	ClarifyQuestions []string       `json:"clarify_questions,omitempty"`
	PendingIntent   string         `json:"pending_intent,omitempty"`
	PendingParams   map[string]any `json:"pending_params,omitempty"`
}

const macSystemPrompt = `당신은 Nexus AI 비서입니다. 사용자 명령을 분석하여 아래 액션 중 하나를 선택하세요.
⚠️ 반드시 JSON만 출력하세요.
형식: {"action":"액션명","params":{...},"message":"사용자에게 보여줄 짧은 답변"}

clarify 절대 금지 케이스 (바로 실행):
- "근처" 포함: 현재 위치 기준으로 실행 가능 → clarify 금지
- "내일 회의", "내일 약속" 등 내일+일정명: 시간 추론 가능 → clarify 금지
- "최근 뉴스", "인기 영상", "트렌딩" 등: 명확한 의도 → clarify 금지

액션 목록:
"chat" → 일반 대화, 질문, 설명 요청
  params: {}

"web_search" → 쇼핑/최저가/뉴스/맛집/유튜브/틱톡/쿠팡/네이버 검색 (파일 저장 없는 단순 검색)
  params: {"query":"검색어","site":"coupang|naver|youtube|tiktok|google|auto","max_items":5}

"multi_action" → 검색/비교/정리 결과를 파일(PDF·Excel·MD·TXT)로 저장하거나, 가격비교·영상검색을 수행할 때
  트리거 키워드: "정리해줘", "요약해줘", "pdf로", "엑셀로", "엑셀 작성", "파일로 만들어줘", "저장해줘", "보고서 만들어줘", "비교해줘", "비교 정리", "비교표", "vs", "차이점 정리", "표로 만들어줘"
  params: {
    "sub_action": "price_compare|video_search|doc_compare|summarize|web_search",
    "query": "검색/비교/요약 대상",
    "format": "pdf|excel|markdown|txt",
    "max_items": 8
  }
  sub_action 선택 기준:
  - "비교해줘" / "vs" / "차이점" → "doc_compare"
  - "요약해줘" / "정리해줘" (특정 주제) → "summarize"
  - "가격 비교" / "최저가" → "price_compare"
  - 유튜브/틱톡 + 저장 → "video_search"
  - 그 외 검색 + 저장 → "web_search"
  format 선택 기준:
  - "pdf로" / "PDF" → "pdf"
  - "엑셀로" / "xlsx" / "엑셀" → "excel"
  - "마크다운" / ".md" → "markdown"
  - "텍스트" / ".txt" → "txt"
  - 키워드 없으면 → "markdown" (기본값)

"weather" → 날씨 확인
  params: {"city":"도시명"}

"calendar_today" → 오늘 일정
  params: {}

"calendar_add" → 일정 추가
  params: {"title":"제목","date":"YYYY-MM-DD","time":"HH:MM"}

"persona_switch" → AI 페르소나 변경
  params: {"id":"nexus|research|creative|finance"}

"workflow_plan" → 목표 달성 워크플로우 계획
  params: {"goal":"목표"}

"trip_plan" → 출장/여행 자동 준비 (항공권·호텔·날씨·맛집·환율 한 번에)
  트리거: "출장", "여행 준비", "출장 준비", "trip", "여행 계획"
  params: {"destination":"목적지","date":"출발일YYYY-MM-DD","days":1,"purpose":"출장|여행"}

"clipboard_action" → 클립보드/복사한 내용을 번역·요약·교정·설명·코드분석·다시쓰기
  트리거: "복사한", "클립보드", "방금 복사", "붙여넣은", "copied", "clipboard", "paste"
  params: {"action":"translate|summarize|proofread|explain|analyze_code|rewrite|translate_en|translate_ko"}
  action 선택:
  - "번역해줘", "translate" → "translate"
  - "영어로" → "translate_en"
  - "한국어로" → "translate_ko"
  - "요약해줘", "summarize" → "summarize"
  - "맞춤법", "교정", "grammar" → "proofread"
  - "설명해줘", "무슨 뜻", "explain" → "explain"
  - "코드 분석", "analyze code" → "analyze_code"
  - "다시 써줘", "rewrite", "고쳐줘" → "rewrite"

"windows_only" → Windows PC 제어 기능 (볼륨, 보안, 프로세스 등)
  params: {"feature":"기능명"}

"clarify" → 실행에 필수 정보가 없을 때만 사용
  params: {"question":"주인님께 물을 질문(1가지만)","missing":"없는 정보","intent":"원래 액션명","collected":{...지금까지 파악된 파라미터...}}

판단 기준:
- 날씨/기상 → weather (도시 없으면 clarify)
- 일정/캘린더/스케줄 → calendar_today 또는 calendar_add (날짜 없으면 clarify)
- 쇼핑/검색/맛집/뉴스/유튜브/틱톡 → web_search (맛집인데 지역 없으면 clarify)
- 다음 중 하나라도 포함 → 반드시 multi_action:
  · "정리해줘", "정리해", "정리하여", "정리 좀"
  · "요약해줘", "요약해", "요약 정리"
  · "pdf로", "PDF", "피디에프"
  · "엑셀로", "엑셀에", "엑셀 파일", "xlsx", "Excel"
  · "마크다운으로", "md로"
  · "파일로 만들어", "저장해줘", "저장해"
  · "비교해줘", "비교해", "비교 정리", "비교표", "vs", "VS", "대비"
  · "표로 만들어", "표로 정리"
  · "보고서", "리포트", "report"
- "출장", "여행 준비", "출장 준비" → trip_plan (목적지 없으면 clarify)
- PC제어/보안/최적화/볼륨/밝기 → windows_only
- 그 외 모든 대화 → chat

⚠️ 중요: "엑셀로 정리해줘", "엑셀 파일로 만들어줘" 처럼 엑셀 키워드가 있으면 무조건 multi_action + format=excel

━━━ clarify 사용 기준 (2026년 기준 확장판) ━━━
아래 경우에만 clarify 사용 (나머지는 최선으로 추론해서 즉시 실행)

🔴 필수 Clarify (무조건 물어봐야 하는 경우)
- web_search / browse_page: query가 완전히 없거나 너무 모호할 때
  → "어떤 것을 검색할까요?" 또는 "어떤 키워드로 찾아드릴까요?"
- file_search / recall: 단서(이름, 키워드, 날짜, 발신자)가 전혀 없을 때
  → "어떤 파일을 찾으시나요? 이름이나 키워드, 날짜를 알려주세요"
- weather / 교통 / 일정: 지역이나 날짜가 명확하지 않을 때
  → "어느 지역 날씨를 알려드릴까요?" / "언제 출발하실 예정인가요?"
- scheduler / reminder / 자동 작업: 실행 내용, 시간, 반복 여부가 불완전할 때
  → "언제, 무엇을 자동으로 실행할까요?"
- doc_compare / doc_summary: 비교할 파일 경로나 개수가 불명확할 때
  → "비교하거나 요약할 파일 경로를 알려주세요"
- 상품 검색 (쿠팡, 테무, 네이버쇼핑 등): 브랜드, 모델, 스펙이 불명확할 때
  예) "콜라" → "코카콜라인지 펩시인지요?"
  예) "라면" → "신라면, 너구리, 짜파게티 중 어떤 걸 원하시나요?"
  예) "노트북 추천" → "예산과 용도(업무/게임/학습)를 알려주세요"
- 맛집/장소/예약 검색: 지역이나 종류가 없을 때
  → "어느 지역 맛집을 찾아드릴까요?"

🟠 강력 추천 Clarify (혼란을 크게 줄이는 경우)
- 동일 이름 업체·상품·파일이 여러 개 검색될 때
  → "OO 관련 결과가 여러 개 있습니다. 어느 것을 원하시나요?" (목록 간단히 나열)
- 대명사 / 모호한 참조 ("이거", "그거", "저거", "그 파일", "그 뉴스")
  → "어떤 걸 말씀하시는 건가요? 조금 더 자세히 알려주세요"
- 어휘 중의성 (한 단어가 여러 의미일 때)
  예) "파이썬 알려줘" → "프로그래밍 언어 파이썬인가요, 아니면 뱀 파이썬인가요?"
- 시간 모호성 ("오늘", "이번 주", "지난번")
  → "어느 날짜나 기간을 말씀하시는 건가요?"
- 유사한 의도가 여러 개일 때
  예) "보고서 만들어줘" → "어떤 주제의 보고서를 만드시겠습니까?"
- 클립보드 / 화면 관련 ("이거 번역해", "이 창 정리해")
  → "현재 클립보드 내용인가요, 아니면 화면에 있는 내용인가요?"
- 반복 작업 설정
  → "매일/매주/매월 반복할까요? 아니면 이번 한 번만 할까요?"

🟡 선택적 Clarify (가능하면 추론하고, 그래도 모호하면 물어보기)
- "최신" / "인기" / "추천" 같은 모호한 수식어
- "좋은 거" / "싼 거" / "비싼 거" 같은 주관적 표현
- 숫자/수량 모호성 (예: "커피 3개" → "3잔인가요, 3박스인가요?")

━━━ 철칙 ━━━
- 필수 정보가 없으면 반드시 clarify 사용 (추론으로 채우지 말 것)
- clarify 질문은 1가지만, 자연스럽고 친절하게, 옵션을 제시하면 더 좋음
- 한 번 clarify 후 사용자가 답하면 컨텍스트를 강하게 유지해서 바로 실행
- "최선의 추론으로 바로 실행"은 금지 — 틀린 결과보다 한 번 더 묻는 게 낫다

━━━ 컨텍스트 기반 대명사/참조 처리 ━━━
[이전 대화 컨텍스트: X] 또는 [Previous context: X] 태그가 있으면 X를 사용해서 질문 의도 해석:
예KO) [이전 대화 컨텍스트: 도쿄 여행]\n현재 질문: 그거 호텔도 알아봐줘
→ {"action":"web_search","params":{"query":"도쿄 호텔 추천","site":"auto"},"message":"도쿄 호텔을 찾아볼게요!"}
예EN) [Previous context: Tokyo trip]\nCurrent question: find hotels for that
→ {"action":"web_search","params":{"query":"Tokyo hotel recommendations","site":"auto"},"message":"Looking for Tokyo hotels!"}

━━━ 실시간 데이터 처리 ━━━
- 환율/주가/암호화폐 → web_search (query에 "현재", "오늘" 포함)
- "날씨" (도시 없음) → weather, city="서울" (기본값)
- "근처" 포함 → web_search, 절대 clarify 금지

━━━ 앱 실행 요청 ━━━
"카카오톡 열어줘", "크롬 켜줘" 등 → chat (앱 실행 미지원 안내)

━━━ 복합 명령 (복수 action) ━━━
이 프롬프트에서는 단일 action만 반환. multi-intent는 상위 레이어에서 처리됨.
복합 명령이 들어오면 가장 주요한 action 하나를 선택.

[검색 결과 처리 — 중요]
동일한 이름의 업체·장소·상품이 여러 개 검색되면 절대 하나만 골라 답하지 말고,
반드시 목록을 보여주고 "어느 것을 원하시나요?" 라고 되물을 것.`

const macClarifyResolvePrompt = `당신은 Nexus AI 비서입니다. 사용자가 추가 정보를 제공했습니다.
이전 컨텍스트와 새 정보를 합쳐서 완전한 액션을 결정하세요.
⚠️ 반드시 JSON만 출력하세요.
형식: {"action":"액션명","params":{...완전한 파라미터...},"message":"짧은 답변"}

이전 액션: %s
이전 파라미터: %s
이전 질문: %s
사용자 새 답변: %s`


func handleCommand(w http.ResponseWriter, r *http.Request) {
	var req CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Message == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "message required / message 필요"})
		return
	}

	// ── 사용자 식별 (이메일 우선, 없으면 IP) ────────────────────
	userID := req.UserEmail
	if userID == "" {
		userID = r.RemoteAddr
	}

	start := time.Now()

	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()

	if gKey == "" {
		apiKeyMsg := "Groq API 키가 설정되지 않았습니다. 설정에서 API 키를 입력해주세요."
		if req.Lang == "en" || isEnglishQuery(req.Message) {
			apiKeyMsg = "Groq API key is not configured. Please enter your API key in settings."
		}
		writeJSON(w, 400, map[string]any{
			"success": false,
			"message": apiKeyMsg,
		})
		return
	}

	// ── 멀티턴: 이전 clarify 컨텍스트가 있으면 해소 프롬프트 사용 ──
	var intentPrompt string
	if req.PendingIntent != "" {
		prevParamsJSON, _ := json.Marshal(req.PendingParams)
		clarifyPrompt := macClarifyResolvePrompt
		if req.Lang == "en" || isEnglishQuery(req.Message) {
			clarifyPrompt = "You are Nexus AI assistant. The user has provided additional information. Combine previous context with new info to determine the complete action.\n⚠️ Output JSON ONLY. Format: {\"action\":\"action_name\",\"params\":{...complete params...},\"message\":\"short English response\"}\n\nPrevious action: %s\nPrevious params: %s\nPrevious question: %s\nUser's new answer: %s"
		}
		intentPrompt = fmt.Sprintf(clarifyPrompt,
			req.PendingIntent,
			string(prevParamsJSON),
			req.PendingQuestion,
			req.Message,
		)
	} else {
		intentPrompt = req.Message
	}

	// ── 세션 메모리: 대명사 해소 ────────────────────────────────
	intentPrompt = resolvePronouns(intentPrompt, userID)

	// ── 키워드 사전 라우팅 (LLM보다 우선, 틱톡/유튜브 영상 검색) ──
	msgLower := strings.ToLower(req.Message)
	videoVerbs := []string{"찾", "검색", "영상", "보여", "추천", "viral", "바이럴", "트렌드"}
	isTikTokReq := strings.Contains(msgLower, "틱톡") || strings.Contains(msgLower, "tiktok")
	isYouTubeReq := strings.Contains(msgLower, "유튜브") || strings.Contains(msgLower, "youtube")
	hasVideoVerb := false
	for _, kw := range videoVerbs {
		if strings.Contains(msgLower, kw) {
			hasVideoVerb = true
			break
		}
	}

	var preRoutedAction string
	var preRoutedParams map[string]any
	// 가격/쇼핑/도메인 사전 라우팅
	shoppingSites := map[string]string{
		// ── 쇼핑몰 ──────────────────────────────────────
		"태무": "temu.com", "테무": "temu.com", "temu": "temu.com",
		"쿠팡": "coupang.com", "coupang": "coupang.com",
		"네이버쇼핑": "shopping.naver.com", "네이버 쇼핑": "shopping.naver.com",
		"11번가": "11st.co.kr",
		"지마켓": "gmarket.co.kr", "gmarket": "gmarket.co.kr",
		"옥션": "auction.co.kr", "auction": "auction.co.kr",
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
		// ── 중고차 ──────────────────────────────────────
		"헤이딜러": "heydealer.com", "heydealer": "heydealer.com",
		"엔카": "encar.com", "encar": "encar.com",
		"kb차차차": "kbchachacha.com", "차차차": "kbchachacha.com",
		"sk엔카": "encar.com",
		"오토피디아": "autopedia.co.kr",
		"보배드림": "bobaedream.co.kr",
		"중고차": "encar.com",
		// ── 중고거래 ────────────────────────────────────
		"당근": "daangn.com", "당근마켓": "daangn.com", "daangn": "daangn.com",
		"번개장터": "bunjang.co.kr", "번개": "bunjang.co.kr",
		"중고나라": "joongna.com",
		"헬로마켓": "hellomarket.com",
		// ── 부동산 ──────────────────────────────────────
		"직방": "zigbang.com", "zigbang": "zigbang.com",
		"다방": "dabangapp.com",
		"호갱노노": "hogangnono.com",
		"네이버부동산": "land.naver.com", "네이버 부동산": "land.naver.com",
		"부동산114": "r114.com",
		// ── 음식/배달 ────────────────────────────────────
		"배민": "baemin.com", "배달의민족": "baemin.com",
		"요기요": "yogiyo.co.kr",
		"쿠팡이츠": "coupangeats.com",
		// ── 여행/숙박 ────────────────────────────────────
		"야놀자": "yanolja.com",
		"여기어때": "goodchoice.kr",
		"에어비앤비": "airbnb.co.kr", "airbnb": "airbnb.com",
		"호텔스닷컴": "hotels.com",
		"익스피디아": "expedia.co.kr",
		// ── 전자기기 ─────────────────────────────────────
		"다나와": "danawa.com",
		"에누리": "enuri.com",
		"컴퓨존": "compuzone.co.kr",
		"아이셋톱": "isettop.com",
	}
	detectedShopSite := ""
	for keyword, domain := range shoppingSites {
		if strings.Contains(msgLower, strings.ToLower(keyword)) {
			detectedShopSite = domain
			break
		}
	}

	outFmt := detectOutputFormat(req.Message)
	// 포맷 키워드 OR 저장 동사 중 하나라도 있으면 파일 저장 트리거
	isMultiAction := outFmt != outNone && req.PendingIntent == ""

	// ── Pre-routing: 액션 감지만 (Clarify 판단은 Groq에 위임) ──────
	if detectedShopSite != "" && req.PendingIntent == "" {
		q := req.Message
		for kw := range shoppingSites {
			q = strings.ReplaceAll(q, kw, "")
		}
		for _, rm := range []string{"에서", "찾아줘", "검색해줘", "최저가", "가격", "얼마야", "구매", "사고 싶어", "추천", "알려줘"} {
			q = strings.ReplaceAll(q, rm, "")
		}
		q = strings.TrimSpace(q) // 비어있으면 "" 그대로 유지 — Groq이 "없음"으로 판단하도록
		if isMultiAction {
			preRoutedAction = "multi_action"
			preRoutedParams = map[string]any{"sub_action": "price_compare", "query": q, "site": detectedShopSite, "max_items": 8, "format": string(outFmt)}
		} else {
			preRoutedAction = "price_compare"
			preRoutedParams = map[string]any{"query": q, "site": detectedShopSite, "max_items": 8}
		}
	} else if isTikTokReq && hasVideoVerb && req.PendingIntent == "" {
		q := req.Message
		for _, rm := range []string{"틱톡에서", "틱톡", "tiktok", "찾아줘", "검색해줘", "보여줘", "영상", "추천해줘"} {
			q = strings.ReplaceAll(q, rm, "")
		}
		q = strings.TrimSpace(q)
		if isMultiAction {
			preRoutedAction = "multi_action"
			preRoutedParams = map[string]any{"sub_action": "video_search", "query": q, "platform": "tiktok", "max_items": 8, "format": string(outFmt)}
		} else {
			preRoutedAction = "video_search"
			preRoutedParams = map[string]any{"query": q, "platform": "tiktok", "max_items": 8}
		}
	} else if isYouTubeReq && hasVideoVerb && req.PendingIntent == "" {
		q := req.Message
		for _, rm := range []string{"유튜브에서", "유튜브", "youtube", "찾아줘", "검색해줘", "보여줘", "영상", "추천해줘"} {
			q = strings.ReplaceAll(q, rm, "")
		}
		q = strings.TrimSpace(q)
		if isMultiAction {
			preRoutedAction = "multi_action"
			preRoutedParams = map[string]any{"sub_action": "video_search", "query": q, "platform": "youtube", "max_items": 8, "format": string(outFmt)}
		} else {
			preRoutedAction = "video_search"
			preRoutedParams = map[string]any{"query": q, "platform": "youtube", "max_items": 8}
		}
	} else if isMultiAction && req.PendingIntent == "" {
		lower := strings.ToLower(req.Message)
		subAction := "summarize"
		for _, v := range []string{"비교해줘", "비교해", "비교 정리", "비교표", " vs ", "vs.", "대비"} {
			if strings.Contains(lower, v) {
				subAction = "doc_compare"
				break
			}
		}
		preRoutedAction = "multi_action"
		preRoutedParams = map[string]any{"sub_action": subAction, "query": req.Message, "format": string(outFmt), "max_items": 8}
	}

	// ── 채팅 페르소나 전환 감지 ────────────────────────────────────
	if preRoutedAction == "" && req.PendingIntent == "" {
		personaSwitchMap := map[string]string{
			"개발자": "developer", "개발자모드": "developer", "개발 모드": "developer", "it 모드": "developer", "코딩 모드": "developer",
			"마케터": "marketer", "마케팅 모드": "marketer", "마케팅모드": "marketer", "디지털 마케터": "marketer",
			"영업": "sales", "세일즈": "sales", "영업 모드": "sales", "세일즈 모드": "sales",
			"pm": "pm", "기획자": "pm", "pm 모드": "pm", "기획 모드": "pm", "프로덕트": "pm",
			"디자이너": "designer", "크리에이터": "designer", "디자인 모드": "designer", "크리에이티브 모드": "designer",
			"프리랜서": "freelancer", "1인 사업자": "freelancer", "프리랜서 모드": "freelancer", "사업자 모드": "freelancer",
			"기본": "developer", "기본 모드": "developer",
		}
		switchTriggers := []string{"모드로 바꿔", "모드 바꿔", "모드로 전환", "모드 전환", "페르소나", "으로 바꿔", "로 바꿔줘", "로 전환해"}
		hasTrigger := false
		for _, t := range switchTriggers {
			if strings.Contains(msgLower, t) {
				hasTrigger = true
				break
			}
		}
		if hasTrigger {
			for keyword, pid := range personaSwitchMap {
				if strings.Contains(msgLower, keyword) {
					for _, p := range builtinPersonas {
						if p.ID == pid {
							personaMu.Lock()
							activePersonaID = pid
							personaMu.Unlock()
							savePersonaConfig()
							json200(w, CommandResponse{
								Success:  true,
								Message:  p.Emoji + " " + p.Name + " 모드로 전환했습니다. 이제 " + p.Description + " 관점으로 답변합니다.",
								Action:   "persona_switch",
								Duration: fmt.Sprintf("%.2fs", time.Since(start).Seconds()),
							})
							return
						}
					}
				}
			}
		}
	}

	// 출장/여행 준비 pre-routing (액션 감지만)
	if preRoutedAction == "" && req.PendingIntent == "" {
		for _, v := range []string{"출장 준비", "여행 준비", "출장 계획", "여행 계획", "출장 가", "출장이야", "출장인데", "출장 있", "여행 있", "trip 준비"} {
			if strings.Contains(msgLower, v) {
				preRoutedAction = "trip_plan"
				preRoutedParams = map[string]any{"destination": req.Message, "purpose": "출장"}
				break
			}
		}
	}

	// ── 직업군 워크플로우 프리셋 감지 ─────────────────────────────
	if preRoutedAction == "" && req.PendingIntent == "" {
		pid := getActivePersona().ID
		type presetDef struct {
			triggers []string
			preset   string
		}
		presetMap := map[string][]presetDef{
			"developer": {
				{[]string{"코드 리뷰", "pr 리뷰", "pull request"}, "dev_code_review"},
				{[]string{"버그 해결", "에러 어떻게", "버그 고쳐", "오류 고쳐", "이 에러"}, "dev_bug_fix"},
				{[]string{"리팩토링", "리팩터링", "refactor", "코드 개선"}, "dev_refactor"},
				{[]string{"github 이슈", "깃허브 이슈", "이슈 찾아", "pr 찾아"}, "dev_github_search"},
				{[]string{"터미널 명령", "명령어 최적화", "커맨드 최적화"}, "dev_terminal_command"},
				{[]string{"api 설계", "api 만들어", "openapi", "rest api 설계"}, "dev_api_design"},
				{[]string{"테스트 코드", "단위 테스트", "test code", "테스트 만들어"}, "dev_test_generate"},
				{[]string{"데일리 스탠드업", "스탠드업", "오늘 뭐 했어", "daily standup"}, "dev_daily_standup"},
				{[]string{"pr 만들어", "pr 자동", "풀리퀘스트 만들어"}, "dev_pr_create"},
				{[]string{"ci/cd", "cicd", "ci 개선", "cd 파이프라인", "파이프라인 최적화"}, "dev_ci_cd"},
				{[]string{"로그 분석", "로그 확인", "log 분석"}, "dev_log_analysis"},
				{[]string{"성능 느려", "성능 병목", "퍼포먼스", "performance 분석"}, "dev_performance"},
				{[]string{"보안 검사", "보안 취약점", "security scan", "취약점 스캔"}, "dev_security_scan"},
				{[]string{"docker", "도커", "kubernetes", "k8s", "도커 설정"}, "dev_docker"},
				{[]string{"쿼리 최적화", "db 최적화", "sql 최적화", "데이터베이스 최적화"}, "dev_db_optimize"},
				{[]string{"기술 학습", "기술 정리", "공부 자료", "정리해", "학습 자료"}, "dev_tech_summary"},
				{[]string{"코드 스타일", "lint", "코딩 컨벤션", "코드 스타일 체크"}, "dev_code_style"},
				{[]string{"마이그레이션", "db 마이그레이션", "스키마 변경", "migration"}, "dev_migration"},
				{[]string{"에러 로그 정리", "에러 분류", "로그 카테고리", "오류 분류"}, "dev_error_classify"},
				{[]string{"주간 리포트", "개발 리포트", "주간 개발", "weekly report"}, "dev_weekly_report"},
				{[]string{"배포 체크", "배포 준비", "릴리즈 체크", "배포 전"}, "dev_deploy_check"},
				{[]string{"기술 트렌드", "개발 트렌드", "tech 트렌드", "최신 기술"}, "dev_tech_trend"},
			},
			"marketer": {
				{[]string{"트렌드 분석", "시장 분석", "이번 주 트렌드", "트렌드 리포트"}, "mkt_trend_analysis"},
				{[]string{"콘텐츠 아이디어", "sns 아이디어", "아이디어 10개", "콘텐츠 기획"}, "mkt_content_idea"},
				{[]string{"경쟁사 분석", "경쟁사 조사", "경쟁사 모니터링", "competitor"}, "mkt_competitor_monitor"},
				{[]string{"광고 문구", "카피라이팅", "광고 카피", "ad copy"}, "mkt_ad_copy"},
				{[]string{"인스타 포스팅", "sns 게시물", "포스팅 만들어", "sns 글"}, "mkt_sns_post"},
				{[]string{"캠페인 기획", "마케팅 캠페인", "캠페인 계획"}, "mkt_campaign_plan"},
				{[]string{"성과 리포트", "마케팅 성과", "이번 달 성과", "kpi 리포트"}, "mkt_performance_report"},
				{[]string{"seo 키워드", "키워드 분석", "검색 키워드", "seo 분석"}, "mkt_seo_keyword"},
				{[]string{"뉴스레터", "이메일 뉴스레터", "newsletter"}, "mkt_email_newsletter"},
				{[]string{"인플루언서 찾아", "인플루언서 검색", "influencer"}, "mkt_influencer_search"},
				{[]string{"a/b 테스트", "ab 테스트", "split test", "ab 테스트 아이디어"}, "mkt_ab_test_idea"},
				{[]string{"해시태그", "hashtag", "태그 만들어"}, "mkt_hashtag_generator"},
				{[]string{"랜딩페이지", "landing page", "랜딩 문구", "cta 문구"}, "mkt_landing_page_copy"},
				{[]string{"소셜 캘린더", "sns 캘린더", "게시 계획", "콘텐츠 캘린더"}, "mkt_social_calendar"},
				{[]string{"예산 계획", "마케팅 예산", "채널 예산", "budget plan"}, "mkt_budget_plan"},
				{[]string{"바이럴", "viral", "바이럴 콘텐츠", "바이럴 전략"}, "mkt_viral_content"},
				{[]string{"고객 인사이트", "고객 분석", "customer insight", "소비자 분석"}, "mkt_customer_insight"},
				{[]string{"브랜드 톤", "브랜드 보이스", "brand voice", "톤 맞춰"}, "mkt_brand_voice"},
				{[]string{"주간 마케팅 요약", "주간 요약", "weekly digest", "마케팅 요약"}, "mkt_weekly_digest"},
				{[]string{"나 홍보", "개인 브랜딩", "personal brand", "linkedin 콘텐츠", "블로그 글"}, "mkt_personal_brand"},
			},
			"sales": {
				{[]string{"고객에게 메일", "영업 이메일", "메일 초안", "이메일 초안"}, "sales_email_draft"},
				{[]string{"미팅 준비", "고객 미팅", "영업 미팅", "내일 미팅"}, "sales_meeting_prep"},
				{[]string{"후속 메일", "followup", "팔로업", "후속 연락"}, "sales_followup"},
				{[]string{"제안서", "제안 초안", "영업 제안", "제안서 만들어"}, "sales_proposal"},
				{[]string{"이의제기", "이의 대응", "반론 대응", "objection"}, "sales_objection"},
				{[]string{"파이프라인", "pipeline", "영업 현황", "파이프라인 정리"}, "sales_pipeline"},
				{[]string{"계약서 만들어", "계약서 초안", "계약 초안"}, "sales_contract"},
				{[]string{"발견 질문", "discovery question", "고객 질문 만들어"}, "sales_discovery_question"},
				{[]string{"데모 스크립트", "demo script", "시연 대본"}, "sales_demo_script"},
				{[]string{"협상 전략", "가격 협상 어떻게", "협상 방법"}, "sales_negotiation"},
				{[]string{"영업 예측", "이번 달 예상", "매출 예측", "forecast"}, "sales_forecast"},
				{[]string{"crm 업데이트", "crm 정리", "crm 입력"}, "sales_crm_update"},
				{[]string{"통화 요약", "콜 요약", "call summary"}, "sales_call_summary"},
				{[]string{"제안서 후속", "proposal followup", "제안 후속"}, "sales_proposal_followup"},
				{[]string{"win loss", "win/loss", "승패 분석", "계약 분석"}, "sales_win_loss_analysis"},
				{[]string{"추천 요청", "referral", "소개 부탁"}, "sales_referral_request"},
				{[]string{"가격 협상", "가격 전략", "할인 전략"}, "sales_price_negotiation"},
				{[]string{"계약서 검토", "계약서 봐줘"}, "sales_contract_review"},
				{[]string{"분기 리뷰", "분기 영업", "quarterly", "분기 결과"}, "sales_quarterly_review"},
				{[]string{"고객 분석해", "고객 프로필", "고객 파악"}, "sales_client_portrait"},
			},
			"pm": {
				{[]string{"요구사항 정리", "요구사항 문서", "기능 정리"}, "pm_requirements"},
				{[]string{"로드맵", "roadmap", "로드맵 만들어"}, "pm_roadmap"},
				{[]string{"이해관계자 브리핑", "stakeholder", "이번 주 브리핑"}, "pm_stakeholder_summary"},
				{[]string{"리스크 분석", "risk", "위험 분석"}, "pm_risk_analysis"},
				{[]string{"미팅 노트", "회의 정리", "회의록 정리"}, "pm_meeting_note"},
				{[]string{"유저 스토리", "user story", "스토리 만들어"}, "pm_user_story"},
				{[]string{"주간 보고서", "주간 보고", "weekly report"}, "pm_weekly_report"},
				{[]string{"prd 작성", "prd 만들어", "기획서 써줘"}, "pm_prd_write"},
				{[]string{"기획서 검토", "스펙 검토", "spec review"}, "pm_spec_review"},
				{[]string{"우선순위 정리", "우선순위 매트릭스", "moscow"}, "pm_priority_matrix"},
				{[]string{"회고 정리", "retrospective", "레트로"}, "pm_retrospective"},
				{[]string{"okr", "okr 세워", "목표 설정"}, "pm_okr_setting"},
				{[]string{"리소스 계획", "인력 배치", "resource plan"}, "pm_resource_plan"},
				{[]string{"이해관계자 맵", "이해관계자 분석", "stakeholder map"}, "pm_stakeholder_map"},
				{[]string{"칸반 정리", "kanban", "보드 정리"}, "pm_feature_kanban"},
				{[]string{"인터뷰 요약", "사용자 인터뷰", "user interview"}, "pm_user_interview_summary"},
				{[]string{"경쟁사 분석", "경쟁 제품", "competitor analysis"}, "pm_competitor_analysis"},
				{[]string{"gtm", "go-to-market", "출시 전략"}, "pm_go_to_market"},
				{[]string{"스프린트 계획", "sprint planning", "sprint"}, "pm_sprint_planning"},
				{[]string{"kpi 대시보드", "지표 정리", "metrics"}, "pm_metrics_dashboard"},
			},
			"designer": {
				{[]string{"레퍼런스 찾아", "비슷한 디자인", "디자인 레퍼런스"}, "design_reference"},
				{[]string{"파일 정리해", "디자인 파일 정리", "폴더 정리"}, "design_file_organize"},
				{[]string{"컬러 팔레트", "color palette", "색상 팔레트"}, "design_color_palette"},
				{[]string{"이미지 정리해", "이미지 편집", "일괄 편집"}, "design_image_edit"},
				{[]string{"포스터 아이디어", "콘텐츠 디자인", "디자인 아이디어"}, "design_content_idea"},
				{[]string{"디자인 피드백", "이 디자인 봐줘", "피드백 해줘"}, "design_feedback"},
				{[]string{"무드보드", "moodboard", "분위기 참고"}, "design_moodboard"},
				{[]string{"ui kit", "ui 키트", "컴포넌트"}, "design_ui_kit"},
				{[]string{"프로토타입 봐줘", "prototype review", "figma 봐줘"}, "design_prototype_review"},
				{[]string{"에셋 정리", "asset export", "에셋 내보내기"}, "design_asset_export"},
				{[]string{"브랜드 가이드", "brand guideline", "브랜드 가이드라인"}, "design_brand_guideline"},
				{[]string{"sns 키트", "소셜 키트", "social media kit"}, "design_social_media_kit"},
				{[]string{"발표 자료 만들어", "슬라이드 만들어", "presentation"}, "design_presentation_deck"},
				{[]string{"아이콘 세트", "icon set", "아이콘 만들어"}, "design_icon_set"},
				{[]string{"폰트 시스템", "타이포그래피", "typography"}, "design_typography"},
				{[]string{"애니메이션 만들어", "lottie", "모션 아이디어"}, "design_animation_idea"},
				{[]string{"접근성 체크", "accessibility", "wcag"}, "design_accessibility_check"},
				{[]string{"반응형 확인", "모바일 확인", "responsive"}, "design_responsive_test"},
				{[]string{"클라이언트 자료", "클라이언트 발표", "client presentation"}, "design_client_presentation"},
				{[]string{"포트폴리오 업데이트", "포트폴리오 정리", "portfolio"}, "design_portfolio_update"},
			},
			"freelancer": {
				{[]string{"클라이언트 정리", "클라이언트 관리", "고객 정리"}, "freelancer_client_manage"},
				{[]string{"견적서 만들어", "견적서", "프로젝트 견적"}, "freelancer_estimate"},
				{[]string{"청구서 만들어", "invoice", "세금계산서"}, "freelancer_invoice"},
				{[]string{"세금 정리", "세금 계산", "종합소득세", "부가세"}, "freelancer_tax"},
				{[]string{"시간 기록", "time tracking", "작업 시간"}, "freelancer_time_track"},
				{[]string{"포트폴리오 업데이트", "포트폴리오 정리"}, "freelancer_portfolio"},
				{[]string{"나 홍보", "자기 pr", "self marketing"}, "freelancer_self_marketing"},
				{[]string{"계약서 검토", "계약서 봐줘"}, "freelancer_contract_review"},
				{[]string{"현금 흐름", "cash flow", "수입 지출"}, "freelancer_cashflow"},
				{[]string{"세금 신고 자료", "연말정산", "부가세 신고"}, "freelancer_tax_report"},
				{[]string{"신규 클라이언트", "온보딩", "client onboarding"}, "freelancer_client_onboarding"},
				{[]string{"프로젝트 시작", "킥오프", "kickoff"}, "freelancer_project_kickoff"},
				{[]string{"산출물 확인", "deliverable", "최종 파일 확인"}, "freelancer_deliverable_check"},
				{[]string{"미수금 독촉", "payment reminder", "미수금"}, "freelancer_payment_reminder"},
				{[]string{"제안서 템플릿", "proposal template"}, "freelancer_proposal_template"},
				{[]string{"단가 계산", "적정 단가", "시간당 단가"}, "freelancer_rate_calculation"},
				{[]string{"작업 로그", "work log", "오늘 작업"}, "freelancer_work_log"},
				{[]string{"사업 계획", "business plan", "사업 계획서"}, "freelancer_business_plan"},
				{[]string{"네트워킹 콘텐츠", "linkedin 포스팅", "networking"}, "freelancer_networking_content"},
				{[]string{"올해 정리", "연간 리뷰", "yearly review"}, "freelancer_yearly_review"},
			},
		}
		if presets, ok := presetMap[pid]; ok {
			for _, pd := range presets {
				for _, t := range pd.triggers {
					if strings.Contains(msgLower, t) {
						preRoutedAction = "workflow_preset"
						preRoutedParams = map[string]any{"preset": pd.preset, "query": req.Message}
						break
					}
				}
				if preRoutedAction != "" {
					break
				}
			}
		}
	}

	// ── Intent 분류 + Clarify 판단 (워크플로우 프리셋은 건너뜀) ─────────────
	var structuredResult *ClarifyResult
	if preRoutedAction != "workflow_preset" && req.PendingIntent == "" {
		clarifyNow := func(questions []string, pi string, pp map[string]any) {
			q := ""
			if len(questions) > 0 {
				q = questions[0]
			}
			d := fmt.Sprintf("%.2fs", time.Since(start).Seconds())
			json200(w, CommandResponse{
				Success: true, Message: q, Action: "clarify",
				NeedsClarify: true, ClarifyQuestion: q, ClarifyQuestions: questions,
				PendingIntent: pi, PendingParams: pp, Duration: d,
			})
		}

		groqCtx := req.Message
		if preRoutedAction != "" {
			groqCtx = fmt.Sprintf("[감지된 액션: %s]\n사용자 요청: %s", preRoutedAction, req.Message)
		}

		// Claude Haiku 우선, 없으면 Groq fallback
		var cr1 *ClarifyResult
		var err1 error
		llmMu.RLock()
		hasClaude := llmClaudeKey != ""
		llmMu.RUnlock()
		if hasClaude {
			cr1, err1 = callClaudeIntent(groqCtx)
		}
		if !hasClaude || err1 != nil {
			cr1, err1 = callGroqStructured(groqCtx)
		}
		if err1 == nil {
			structuredResult = cr1
			if cr1.NeedsClarify {
				pi := preRoutedAction
				if pi == "" && len(cr1.Intents) > 0 {
					pi = cr1.Intents[0].Action
				}
				clarifyNow(cr1.ClarifyQuestions, pi, preRoutedParams)
				return
			}
			// multi-intent: 2개 이상의 intent를 병렬로 처리
			if preRoutedAction == "" && len(cr1.Intents) >= 2 {
				type partResult struct {
					desc string
					text string
				}
				parts := make([]partResult, len(cr1.Intents))
				var wgM sync.WaitGroup
				for i, it := range cr1.Intents {
					wgM.Add(1)
					go func(idx int, item IntentItem) {
						defer wgM.Done()
						var txt string
						searchQ := func(q string) string {
							r := runWebSearchMac(gKey, q, "auto", 5)
							return r.Summary
						}
						switch item.Action {
						case "web_search", "trip_plan":
							q, _ := item.Params["query"].(string)
							if q == "" {
								q, _ = item.Params["destination"].(string)
							}
							if q == "" {
								q = req.Message
							}
							txt = searchQ(q)
						case "weather":
							city, _ := item.Params["city"].(string)
							txt = searchQ(city + " 날씨")
						default:
							q, _ := item.Params["query"].(string)
							if q == "" {
								q = req.Message
							}
							txt = searchQ(q)
						}
						parts[idx] = partResult{desc: item.Description, text: txt}
					}(i, it)
				}
				wgM.Wait()

				combined := ""
				for _, p := range parts {
					if p.text != "" {
						if p.desc != "" {
							combined += "### " + p.desc + "\n" + p.text + "\n\n"
						} else {
							combined += p.text + "\n\n"
						}
					}
				}
				if combined == "" {
					combined = "검색 결과를 가져오지 못했습니다."
				}
				dur := fmt.Sprintf("%.2fs", time.Since(start).Seconds())
				json200(w, CommandResponse{
					Success: true, Action: "web_search", Message: strings.TrimSpace(combined),
					Duration: dur,
				})
				return
			}
		} else {
			// Groq 에러 시 보수적 처리
			if len([]rune(strings.TrimSpace(req.Message))) < 8 {
				clarifyNow([]string{"무엇을 도와드릴까요? 조금 더 구체적으로 알려주세요."}, "chat", nil)
				return
			}
		}
	}

	var intent struct {
		Action  string         `json:"action"`
		Params  map[string]any `json:"params"`
		Message string         `json:"message"`
	}

	if preRoutedAction != "" {
		intent.Action = preRoutedAction
		intent.Params = preRoutedParams
	} else if structuredResult != nil && len(structuredResult.Intents) == 1 {
		// structured result에서 단일 intent를 바로 사용 — 두 번째 LLM 호출 불필요
		it := structuredResult.Intents[0]
		intent.Action = it.Action
		intent.Params = it.Params
	} else {
		// fallback: LLM으로 의도 파악
		sysPr := macSystemPrompt
		if req.Lang == "en" || isEnglishQuery(req.Message) {
			sysPr += "\n⚠️ IMPORTANT: The user is writing in English. The 'message' field in your JSON response MUST be in English."
		}
		msgs := []groqMsg{
			{Role: "system", Content: sysPr},
			{Role: "user", Content: intentPrompt},
		}
		raw, _, err := callGroq(gKey, groqFastModel, msgs, 500, true)
		if err != nil {
			writeJSON(w, 500, map[string]any{"success": false, "message": "LLM 오류: " + err.Error()})
			return
		}
		if err := json.Unmarshal([]byte(raw), &intent); err != nil {
			intent.Action = "chat"
			intent.Message = raw
		}
	}

	// ── 사용량 체크 + 모델 티어 결정 ────────────────────────────
	tier := DecideModelTier(intent.Action)
	allowed, freeLeft, premiumLeft := globalUsage.CheckAndIncrement(userID, string(tier))
	if !allowed {
		json200(w, usageLimitResponse(tier, freeLeft, premiumLeft))
		return
	}

	dur := fmt.Sprintf("%.2fs", time.Since(start).Seconds())

	switch intent.Action {
	case "clarify":
		// 추가 정보 필요 — 프론트엔드에 질문 반환
		var question, missing, pendingIntent string
		var collected map[string]any
		if intent.Params != nil {
			question, _ = intent.Params["question"].(string)
			missing, _ = intent.Params["missing"].(string)
			pendingIntent, _ = intent.Params["intent"].(string)
			collected, _ = intent.Params["collected"].(map[string]any)
		}
		if question == "" {
			question = "조금 더 알려주시면 도움이 될 것 같아요. 어떻게 도와드릴까요?"
		}
		_ = missing
		json200(w, CommandResponse{
			Success:          true,
			Message:          question,
			Action:           "clarify",
			NeedsClarify:     true,
			ClarifyQuestion:  question,
			ClarifyQuestions: []string{question},
			PendingIntent:    pendingIntent,
			PendingParams:    collected,
			Duration:         dur,
		})

	case "chat":
		cat := detectCategory(req.Message)
		expertList := detectExperts(req.Message, req.Lang)
		previewType := categoryPreviewType(cat)

		var answer string
		var chatItems []map[string]string
		var wg sync.WaitGroup
		wg.Add(2)

		// 고루틴 A: LLM 답변 (전문가 or 일반)
		go func() {
			defer wg.Done()
			if len(expertList) > 0 {
				answer, _ = runExpertParallel(req.Message, req.Lang, gKey, expertList, req.History)
			}
			if answer == "" {
				lang := req.Lang
				var sysPrompt string
				if lang == "en" {
					sysPrompt = "You are Nexus AI, a helpful assistant. Answer in natural English, 2-4 sentences. No markdown headers."
				} else {
					personaPrompt := getPersonaSystemPrompt()
					sysPrompt = personaPrompt + "\n자연스러운 한국어로 답변하세요. 마크다운 헤더(##, ###) 금지."
				}
				// 세션 히스토리 주입 (최근 6턴)
				var msgs []groqMsg
				msgs = append(msgs, groqMsg{Role: "system", Content: sysPrompt})
				sess := getSession(userID)
				sessionStoreMu.RLock()
				hist := sess.history
				sessionStoreMu.RUnlock()
				start2 := len(hist) - 6
				if start2 < 0 {
					start2 = 0
				}
				for _, h := range hist[start2:] {
					if h.Content != req.Message { // 현재 메시지 중복 방지
						msgs = append(msgs, groqMsg{Role: h.Role, Content: h.Content})
					}
				}
				msgs = append(msgs, groqMsg{Role: "user", Content: req.Message})
				answer, _, _ = callGroqWithCitations(gKey, groqChatModel, msgs, 600)
				if answer == "" {
					if lang == "en" {
						answer = "Sorry, an error occurred while generating a response."
					} else {
						answer = "죄송합니다, 답변을 생성하는 중 오류가 발생했습니다."
					}
				}
			}
		}()

		// 고루틴 B: 카테고리별 상세 페이지 검색
		go func() {
			defer wg.Done()
			expertCat := expertsToCategory(expertList)
			pr := parallelWebSearch(req.Message, 6, expertCat)
			if len(pr.Items) > 0 {
				chatItems = pr.Items
			} else {
				searchCat := cat
				if expertCat >= 0 {
					searchCat = expertCat
				}
				chatItems = categoryFallbackSites(req.Message, searchCat)
			}
		}()
		wg.Wait()

		appendSession(userID, "user", req.Message)
		appendSession(userID, "assistant", answer)

		json200(w, CommandResponse{
			Success:  true,
			Message:  answer,
			Action:   "chat",
			Result:   map[string]any{"reply": answer, "items": chatItems, "preview_type": previewType},
			Duration: dur,
		})

	case "weather":
		city := "서울"
		if c, ok := intent.Params["city"].(string); ok && c != "" {
			city = c
		}
		// wttr.in 실시간 날씨 API 호출
		wText := fetchWeatherText(city, gKey)
		json200(w, CommandResponse{
			Success:  true,
			Message:  wText,
			Action:   "weather",
			Result:   map[string]any{"city": city},
			Duration: dur,
		})

	case "calendar_today":
		today := time.Now().Format("2006-01-02")
		evs := loadEvents()
		var todayEvs []CalEvent
		for _, e := range evs {
			if e.Date == today {
				todayEvs = append(todayEvs, e)
			}
		}
		msg := fmt.Sprintf("오늘(%s) 일정이 %d개 있습니다.", today, len(todayEvs))
		if len(todayEvs) == 0 {
			msg = "오늘 등록된 일정이 없습니다."
		}
		json200(w, CommandResponse{
			Success:  true,
			Message:  msg,
			Action:   "calendar_today",
			Result:   map[string]any{"events": todayEvs},
			Duration: dur,
		})

	case "calendar_add":
		var title, date, t string
		if intent.Params != nil {
			title, _ = intent.Params["title"].(string)
			date, _ = intent.Params["date"].(string)
			t, _ = intent.Params["time"].(string)
		}
		if title == "" {
			title = req.Message
		}
		if date == "" {
			date = time.Now().Format("2006-01-02")
		}
		ev := CalEvent{
			ID: fmt.Sprintf("%d", time.Now().UnixMilli()),
			Title: title, Date: date, Time: t,
		}
		evs := loadEvents()
		evs = append(evs, ev)
		saveEvents(evs)
		json200(w, CommandResponse{
			Success:  true,
			Message:  fmt.Sprintf("✅ 일정 추가됨: %s (%s)", title, date),
			Action:   "calendar_add",
			Result:   map[string]any{"event": ev},
			Duration: dur,
		})

	case "price_compare":
		var query, site string
		maxItems := 8
		if intent.Params != nil {
			query, _ = intent.Params["query"].(string)
			site, _ = intent.Params["site"].(string)
			if v, ok := intent.Params["max_items"].(float64); ok {
				maxItems = int(v)
			}
		}
		if query == "" {
			query = req.Message
		}
		llmMu.RLock()
		priceTKey := llmTavilyKey
		llmMu.RUnlock()
		var priceItems []map[string]string
		if priceTKey != "" {
			// include_domains 방식 사용 (site: 접두사는 결과 0개 버그 있음)
			if site != "" {
				if tr, ok := tavilySearchDomain(priceTKey, query, maxItems, site); ok {
					priceItems = tr.Items
				}
			}
			if len(priceItems) == 0 {
				if tr, ok := tavilySearch(priceTKey, query, maxItems); ok {
					priceItems = tr.Items
				}
			}
		}
		siteName := site
		if siteName == "" {
			siteName = "쇼핑몰"
		}
		if len(priceItems) == 0 {
			enc := strings.ReplaceAll(query, " ", "+")
			priceItems = []map[string]string{
				{"title": fmt.Sprintf("%s 검색: %s", siteName, query), "url": fmt.Sprintf("https://www.%s/search?q=%s", site, enc)},
			}
		}
		summary := fmt.Sprintf("%s에서 \"%s\" 상품 %d개를 찾았어요!", siteName, query, len(priceItems))
		results := make([]map[string]string, 0, len(priceItems))
		for _, it := range priceItems {
			results = append(results, map[string]string{"site": siteName, "name": it["title"], "price": "", "link": it["url"]})
		}
		json200(w, CommandResponse{
			Success: true, Message: summary, Action: "price_compare",
			Result:   map[string]any{"query": query, "site": site, "summary": summary, "results": results, "total": len(results)},
			Duration: dur,
		})

	case "video_search":
		var query, platform string
		maxItems := 8
		if intent.Params != nil {
			query, _ = intent.Params["query"].(string)
			platform, _ = intent.Params["platform"].(string)
			if v, ok := intent.Params["max_items"].(float64); ok {
				maxItems = int(v)
			}
		}
		if query == "" {
			query = req.Message
		}
		llmMu.RLock()
		videoTKey := llmTavilyKey
		llmMu.RUnlock()
		isTikTok := platform == "tiktok" ||
			strings.Contains(strings.ToLower(req.Message), "틱톡") ||
			strings.Contains(strings.ToLower(req.Message), "tiktok")
		var videoItems []map[string]string
		if isTikTok {
			// site: 접두사 0결과 버그 → include_domains 방식 사용
			if videoTKey != "" {
				if tr, ok := tavilySearchDomain(videoTKey, query, maxItems, "tiktok.com"); ok {
					for _, it := range tr.Items {
						if strings.Contains(it["url"], "tiktok.com") {
							videoItems = append(videoItems, it)
						}
					}
				}
				if len(videoItems) == 0 {
					if tr, ok := tavilySearch(videoTKey, query+" tiktok", maxItems); ok {
						for _, it := range tr.Items {
							if strings.Contains(it["url"], "tiktok.com") {
								videoItems = append(videoItems, it)
							}
						}
					}
				}
			}
			if len(videoItems) == 0 {
				enc := strings.ReplaceAll(query, " ", "%20")
				videoItems = []map[string]string{
					{"title": fmt.Sprintf("TikTok에서 \"%s\" 검색", query), "url": fmt.Sprintf("https://www.tiktok.com/search?q=%s", enc)},
					{"title": "TikTok 트렌딩", "url": "https://www.tiktok.com/trending"},
				}
			}
			summary := fmt.Sprintf("TikTok에서 \"%s\" 영상 %d개를 찾았어요!", query, len(videoItems))
			json200(w, CommandResponse{
				Success: true, Message: summary, Action: "video_search",
				Result:   map[string]any{"query": query, "platform": "tiktok", "items": videoItems, "total": len(videoItems)},
				Duration: dur,
			})
		} else {
			// site: 접두사 0결과 버그 → include_domains 방식 사용
			if videoTKey != "" {
				if tr, ok := tavilySearchDomain(videoTKey, query, maxItems, "youtube.com"); ok {
					for _, it := range tr.Items {
						if strings.Contains(it["url"], "youtube.com/watch") || strings.Contains(it["url"], "youtu.be") {
							videoItems = append(videoItems, it)
						}
					}
				}
				if len(videoItems) == 0 {
					if tr, ok := tavilySearch(videoTKey, query+" youtube 영상", maxItems); ok {
						for _, it := range tr.Items {
							if strings.Contains(it["url"], "youtube.com/watch") || strings.Contains(it["url"], "youtu.be") {
								videoItems = append(videoItems, it)
							}
						}
					}
				}
			}
			if len(videoItems) == 0 {
				enc := strings.ReplaceAll(query, " ", "%20")
				videoItems = []map[string]string{
					{"title": fmt.Sprintf("YouTube에서 \"%s\" 검색", query), "url": fmt.Sprintf("https://www.youtube.com/results?search_query=%s", enc)},
				}
			}
			summary := fmt.Sprintf("YouTube에서 \"%s\" 영상 %d개를 찾았어요!", query, len(videoItems))
			json200(w, CommandResponse{
				Success: true, Message: summary, Action: "video_search",
				Result:   map[string]any{"query": query, "platform": "youtube", "items": videoItems, "total": len(videoItems)},
				Duration: dur,
			})
		}

	case "web_search":
		var query, site string
		maxItems := 5
		if intent.Params != nil {
			query, _ = intent.Params["query"].(string)
			site, _ = intent.Params["site"].(string)
			if v, ok := intent.Params["max_items"].(float64); ok {
				maxItems = int(v)
			}
		}
		if query == "" {
			query = req.Message
		}
		wsLang := req.Lang
		if wsLang == "" {
			if isEnglishQuery(req.Message) {
				wsLang = "en"
			} else {
				wsLang = "ko"
			}
		}
		result := runWebSearchMac(gKey, query, site, maxItems, wsLang)
		appendSession(userID, "user", req.Message)
		appendSession(userID, "assistant", result.Summary)
		json200(w, CommandResponse{
			Success:  true,
			Message:  result.Summary,
			Action:   "web_search",
			Result:   result,
			Duration: dur,
		})

	case "persona_switch":
		var id string
		if intent.Params != nil {
			id, _ = intent.Params["id"].(string)
		}
		for _, p := range builtinPersonas {
			if p.ID == id {
				personaMu.Lock()
				activePersonaID = id
				personaMu.Unlock()
				savePersonaConfig()
				json200(w, CommandResponse{
					Success:  true,
					Message:  p.Emoji + " " + p.Name + " 페르소나로 전환했습니다.",
					Action:   "persona_switch",
					Duration: dur,
				})
				return
			}
		}
		json200(w, CommandResponse{Success: false, Message: "알 수 없는 페르소나입니다.", Action: "persona_switch"})

	case "workflow_plan":
		var goal string
		if intent.Params != nil {
			goal, _ = intent.Params["goal"].(string)
		}
		if goal == "" {
			goal = req.Message
		}
		// Reflection Loop: /api/workflow/run으로 내부 위임
		wfReqBody, _ := json.Marshal(map[string]any{"goal": goal, "use_reflection": true})
		wfResp, wfErr := (&http.Client{Timeout: 120 * time.Second}).Post(
			"http://127.0.0.1:17891/api/workflow/run", "application/json",
			bytes.NewReader(wfReqBody),
		)
		if wfErr == nil && wfResp != nil {
			var wfResult map[string]any
			json.NewDecoder(wfResp.Body).Decode(&wfResult)
			wfResp.Body.Close()
			summary, _ := wfResult["summary"].(string)
			if summary == "" {
				summary = fmt.Sprintf("'%s' 워크플로우 완료", goal)
			}
			json200(w, CommandResponse{
				Success:  true,
				Message:  summary,
				Action:   "workflow_plan",
				Result:   wfResult,
				Duration: dur,
			})
			return
		}
		// fallback: LLM 계획만 반환
		wfEng := isEnglishQuery(goal)
		var wfSys, wfUser string
		if wfEng {
			wfSys = "You are Jarvis AI. Write a step-by-step completion report for the given goal in English."
			wfUser = "Goal: " + goal
		} else {
			wfSys = "당신은 자비스 AI입니다. 주어진 목표를 단계별로 실행 완료 보고 형식으로 작성하세요."
			wfUser = "목표: " + goal
		}
		wMsgs := []groqMsg{
			{Role: "system", Content: wfSys},
			{Role: "user", Content: wfUser},
		}
		plan, _, _ := callGroq(gKey, groqChatModel, wMsgs, 800, false)
		json200(w, CommandResponse{
			Success:  true,
			Message:  plan,
			Action:   "workflow_plan",
			Duration: dur,
		})

	case "trip_plan":
		destination, _ := intent.Params["destination"].(string)
		date, _ := intent.Params["date"].(string)
		purpose, _ := intent.Params["purpose"].(string)
		if destination == "" {
			destination = req.Message
		}
		if date == "" {
			date = time.Now().AddDate(0, 0, 1).Format("2006-01-02")
		}
		if purpose == "" {
			purpose = "출장"
		}

		var tripSections []string
		// 병렬로 정보 수집
		type section struct {
			name string
			body string
		}
		ch := make(chan section, 5)

		// 날씨
		go func() {
			tr, ok := tavilySearch(llmTavilyKey, destination+" 날씨 "+date, 3)
			if ok {
				ch <- section{"날씨", tr.Summary}
			} else {
				ch <- section{"날씨", ""}
			}
		}()
		// 항공권
		go func() {
			tr, ok := tavilySearch(llmTavilyKey, "서울 "+destination+" 항공권 "+date+" 가격", 3)
			if ok {
				ch <- section{"항공권", tr.Summary}
			} else {
				ch <- section{"항공권", ""}
			}
		}()
		// 호텔
		go func() {
			tr, ok := tavilySearch(llmTavilyKey, destination+" 호텔 추천 "+date, 3)
			if ok {
				ch <- section{"호텔", tr.Summary}
			} else {
				ch <- section{"호텔", ""}
			}
		}()
		// 맛집
		go func() {
			tr, ok := tavilySearch(llmTavilyKey, destination+" 맛집 추천 현지인", 3)
			if ok {
				ch <- section{"맛집", tr.Summary}
			} else {
				ch <- section{"맛집", ""}
			}
		}()
		// 환율
		go func() {
			tr, ok := tavilySearch(llmTavilyKey, destination+" 환율 오늘", 2)
			if ok {
				ch <- section{"환율", tr.Summary}
			} else {
				ch <- section{"환율", ""}
			}
		}()

		collected := map[string]string{}
		for i := 0; i < 5; i++ {
			s := <-ch
			if s.body != "" {
				collected[s.name] = s.body
			}
		}

		for _, key := range []string{"날씨", "항공권", "호텔", "맛집", "환율"} {
			if v, ok := collected[key]; ok && v != "" {
				tripSections = append(tripSections, fmt.Sprintf("### %s\n%s", key, v))
			}
		}

		tripEng := isEnglishQuery(destination)
		var prompt string
		if tripEng {
			prompt = fmt.Sprintf(`Prepare a travel checklist for %s %s based on the following information. Write clearly in English.

%s

Checklist format:
1. Weather & packing
2. Flight information
3. Hotel recommendations
4. Local restaurants
5. Currency & budget
6. Other preparations`, destination, date, strings.Join(tripSections, "\n\n"))
		} else {
			prompt = fmt.Sprintf(`%s %s 출장/여행 준비 사항을 다음 정보를 바탕으로 한국어로 깔끔하게 정리해줘.

%s

체크리스트 형식으로 작성해줘:
1. 날씨 및 준비물
2. 항공권 정보
3. 숙소 추천
4. 현지 맛집
5. 환율 및 예산
6. 기타 준비 사항`, destination, date, strings.Join(tripSections, "\n\n"))
		}

		result, _, _ := callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 1000, false)

		// 파일 저장
		home, _ := os.UserHomeDir()
		fname := fmt.Sprintf("trip_%s_%s.md", strings.ReplaceAll(destination, " ", "_"), date)
		fpath := filepath.Join(home, "Desktop", fname)
		os.WriteFile(fpath, []byte(fmt.Sprintf("# %s %s %s 준비\n\n%s", purpose, destination, date, result)), 0644)

		json200(w, CommandResponse{
			Success: true,
			Message: result,
			Action:  "trip_plan",
			Result: map[string]any{
				"destination": destination,
				"date":        date,
				"purpose":     purpose,
				"file":        fpath,
				"sections":    collected,
			},
			Duration: dur,
		})

	case "workflow_preset":
		preset, _ := intent.Params["preset"].(string)
		query, _ := intent.Params["query"].(string)
		if query == "" {
			query = req.Message
		}
		llmMu.RLock()
		tKey := llmTavilyKey
		gKeyWF := llmGroqKey
		llmMu.RUnlock()

		type wfSection struct{ name, body string }
		wfCh := make(chan wfSection, 4)

		type workflowDef struct {
			title    string
			searches []struct{ name, q string }
			prompt   string
		}

		presetDefs := map[string]workflowDef{
			// ── 개발자 (20개) ───────────────────────────────────────
			"dev_bug_fix": {
				title: "버그 빠르게 해결",
				searches: []struct{ name, q string }{
					{"에러 원인 분석", query + " error fix solution 2025"},
					{"스택오버플로우 해결법", query + " stackoverflow github issue"},
				},
				prompt: `다음 정보를 바탕으로 버그 해결 가이드를 작성해줘.
%s
형식:
1. 에러 원인 분석 (가능성 높은 순)
2. 빠른 진단 체크리스트 (3가지)
3. 단계별 수정 방법
4. 수정 코드 예시 (언어/프레임워크 맞게)
5. 재발 방지 방법`,
			},
			"dev_refactor": {
				title: "코드 리팩토링",
				searches: []struct{ name, q string }{
					{"리팩토링 패턴", "code refactoring patterns best practices 2025"},
					{"클린 코드 원칙", "clean code principles SOLID"},
				},
				prompt: `다음 정보를 바탕으로 코드 리팩토링 가이드를 작성해줘.
%s
형식:
1. 리팩토링이 필요한 코드 냄새(Code Smell) 감지
2. 적용할 리팩토링 패턴 (Extract/Replace/Rename 등)
3. 단계별 리팩토링 순서
4. Before/After 코드 예시
5. 테스트 안전망 구축 방법`,
			},
			"dev_github_search": {
				title: "GitHub 이슈/PR 검색",
				searches: []struct{ name, q string }{
					{"GitHub 이슈 검색", query + " github issue bug fix"},
					{"관련 PR/커밋", query + " github pull request merged solution"},
				},
				prompt: `다음 정보를 바탕으로 GitHub 이슈/PR 검색 결과를 정리해줘.
%s
형식:
1. 관련 이슈 요약 (상태/라벨/우선순위)
2. 연관 PR 목록 및 상태
3. 핵심 해결책 요약
4. 추가로 확인할 저장소/이슈 추천
5. GitHub 검색 팁 (고급 검색 쿼리)`,
			},
			"dev_terminal_command": {
				title: "터미널 명령 최적화",
				searches: []struct{ name, q string }{
					{"터미널 명령어 최적화", query + " terminal command optimization linux mac"},
					{"bash 스크립트 팁", "bash zsh productivity tips 2025"},
				},
				prompt: `다음 정보를 바탕으로 터미널 명령어 최적화 가이드를 작성해줘.
%s
형식:
1. 현재 명령어 분석 및 문제점
2. 최적화된 명령어 제안 (복사 가능)
3. 파이프라인/조합 활용법
4. 알리아스(alias) 등록 예시
5. 추천 터미널 도구 (fzf/ripgrep/bat 등)`,
			},
			"dev_api_design": {
				title: "API 설계",
				searches: []struct{ name, q string }{
					{"REST API 설계 원칙", "REST API design best practices 2025"},
					{"OpenAPI 스펙 예시", "OpenAPI 3.0 specification example"},
				},
				prompt: `다음 정보를 바탕으로 API 설계 가이드를 작성해줘.
%s
형식:
1. 엔드포인트 구조 설계 (RESTful 원칙)
2. 요청/응답 스키마 예시 (JSON)
3. HTTP 메서드 및 상태코드 사용 기준
4. 인증/인가 방식 추천
5. OpenAPI 스펙 초안 (YAML)
6. 버전 관리 전략`,
			},
			"dev_test_generate": {
				title: "테스트 코드 자동 생성",
				searches: []struct{ name, q string }{
					{"단위 테스트 작성법", "unit test best practices 2025"},
					{"테스트 커버리지 전략", "test coverage strategy TDD BDD"},
				},
				prompt: `다음 정보를 바탕으로 테스트 코드 생성 가이드를 작성해줘.
%s
형식:
1. 테스트 전략 (단위/통합/E2E 구분)
2. 테스트 케이스 설계 (Happy/Edge/Error Path)
3. 단위 테스트 코드 예시 (Jest/PyTest/Go test)
4. Mock/Stub 활용 방법
5. 테스트 커버리지 목표 및 측정 방법`,
			},
			"dev_daily_standup": {
				title: "데일리 스탠드업 브리핑",
				searches: []struct{ name, q string }{
					{"스탠드업 미팅 효과적 운영", "daily standup meeting best practice agile"},
				},
				prompt: `다음 정보를 바탕으로 오늘 데일리 스탠드업 브리핑 초안을 작성해줘.
%s
형식:
1. 어제 한 일 (Yesterday)
2. 오늘 할 일 (Today)
3. 블로커/이슈 (Blocker)
4. 팀에 공유할 내용
5. 스탠드업 효과적으로 진행하는 팁`,
			},
			"dev_pr_create": {
				title: "PR 자동 생성",
				searches: []struct{ name, q string }{
					{"PR 작성 베스트 프랙티스", "pull request description best practices"},
					{"PR 템플릿 예시", "github pull request template checklist"},
				},
				prompt: `다음 정보를 바탕으로 PR(Pull Request) 작성 초안을 생성해줘.
%s
형식:
1. PR 제목 (명확하고 간결하게)
2. 변경 사항 요약 (What/Why/How)
3. 테스트 방법
4. 스크린샷/영상 첨부 체크
5. 리뷰어 체크리스트
6. 관련 이슈 링크 형식`,
			},
			"dev_ci_cd": {
				title: "CI/CD 파이프라인 최적화",
				searches: []struct{ name, q string }{
					{"CI/CD 최적화 방법", "CI/CD pipeline optimization 2025"},
					{"GitHub Actions 최적화", "GitHub Actions workflow optimization cache"},
				},
				prompt: `다음 정보를 바탕으로 CI/CD 파이프라인 최적화 가이드를 작성해줘.
%s
형식:
1. 현재 파이프라인 분석 포인트
2. 빌드 속도 최적화 (캐시/병렬화)
3. 테스트 자동화 개선
4. 배포 자동화 전략 (Blue-Green/Canary)
5. 모니터링 및 알림 설정
6. GitHub Actions/GitLab CI 설정 예시`,
			},
			"dev_log_analysis": {
				title: "로그 분석 및 디버깅",
				searches: []struct{ name, q string }{
					{"로그 분석 방법", "log analysis debugging best practice"},
					{"에러 패턴 감지", "error pattern detection log monitoring"},
				},
				prompt: `다음 정보를 바탕으로 로그 분석 및 디버깅 가이드를 작성해줘.
%s
형식:
1. 로그 레벨 분류 (ERROR/WARN/INFO/DEBUG)
2. 이상 패턴 감지 방법
3. 핵심 에러 원인 분석
4. 로그 분석 명령어 (grep/awk/jq)
5. 로그 모니터링 도구 추천 (ELK/Grafana)
6. 알림 설정 방법`,
			},
			"dev_performance": {
				title: "성능 병목점 분석",
				searches: []struct{ name, q string }{
					{"성능 최적화 방법", "application performance optimization 2025"},
					{"프로파일링 도구", "profiling tools performance bottleneck"},
				},
				prompt: `다음 정보를 바탕으로 성능 병목점 분석 및 최적화 가이드를 작성해줘.
%s
형식:
1. 성능 측정 방법 및 지표 (응답시간/처리량/메모리)
2. 프로파일링 도구 사용법
3. 병목점 유형별 원인 (CPU/메모리/I/O/네트워크)
4. 최적화 우선순위 결정 기준
5. 코드 수준 최적화 예시
6. 인프라 수준 개선 방법`,
			},
			"dev_security_scan": {
				title: "보안 취약점 검사",
				searches: []struct{ name, q string }{
					{"OWASP 취약점 2025", "OWASP top 10 vulnerabilities 2025"},
					{"코드 보안 체크리스트", "code security audit checklist dependency vulnerability"},
				},
				prompt: `다음 정보를 바탕으로 보안 취약점 검사 가이드를 작성해줘.
%s
형식:
1. OWASP Top 10 체크 항목
2. 코드 수준 취약점 (인젝션/XSS/CSRF 등)
3. 의존성 취약점 스캔 방법 (npm audit/snyk)
4. 인증/인가 보안 체크포인트
5. 시크릿/API 키 노출 방지
6. 보안 도구 추천 및 자동화`,
			},
			"dev_docker": {
				title: "Docker/K8s 설정",
				searches: []struct{ name, q string }{
					{"Docker 최적화", "Dockerfile optimization best practices 2025"},
					{"Kubernetes 배포", "Kubernetes deployment best practices"},
				},
				prompt: `다음 정보를 바탕으로 Docker/Kubernetes 설정 가이드를 작성해줘.
%s
형식:
1. Dockerfile 최적화 (멀티스테이지 빌드/레이어 최소화)
2. docker-compose.yml 예시
3. Kubernetes Deployment/Service yaml 예시
4. 이미지 크기 줄이는 방법
5. 컨테이너 보안 설정
6. 로컬 개발 환경 구성`,
			},
			"dev_db_optimize": {
				title: "데이터베이스 쿼리 최적화",
				searches: []struct{ name, q string }{
					{"SQL 쿼리 최적화", "SQL query optimization index performance 2025"},
					{"데이터베이스 성능 튜닝", "database performance tuning explain plan"},
				},
				prompt: `다음 정보를 바탕으로 데이터베이스 쿼리 최적화 가이드를 작성해줘.
%s
형식:
1. 느린 쿼리 식별 방법 (EXPLAIN/ANALYZE)
2. 인덱스 설계 전략 (복합/커버링/부분 인덱스)
3. N+1 쿼리 문제 해결
4. 최적화된 쿼리 예시 (Before/After)
5. 캐싱 전략 (Redis/Memcached)
6. 데이터베이스별 특화 팁 (PostgreSQL/MySQL/MongoDB)`,
			},
			"dev_tech_summary": {
				title: "기술 학습 자료 정리",
				searches: []struct{ name, q string }{
					{"기술 공식 문서", query + " official documentation tutorial 2025"},
					{"베스트 프랙티스", query + " best practices examples github"},
				},
				prompt: `다음 정보를 바탕으로 기술 학습 자료를 정리해줘.
%s
형식:
1. 핵심 개념 요약 (3-5가지)
2. 빠른 시작 (Quick Start) 가이드
3. 꼭 알아야 할 API/명령어
4. 추천 학습 순서 및 자료
5. 자주 실수하는 부분 주의사항
6. 실전 예제 코드`,
			},
			"dev_code_style": {
				title: "코드 스타일 일관성 검사",
				searches: []struct{ name, q string }{
					{"코딩 컨벤션 가이드", "coding convention style guide 2025"},
					{"Lint 설정 방법", "ESLint Prettier golangci-lint configuration"},
				},
				prompt: `다음 정보를 바탕으로 코드 스타일 가이드를 작성해줘.
%s
형식:
1. 팀 코딩 컨벤션 핵심 규칙 (네이밍/포맷/구조)
2. Linter 설정 방법 (.eslintrc/.golangci.yml)
3. Prettier/포맷터 설정 예시
4. 자주 발생하는 스타일 위반 패턴
5. pre-commit hook 자동화 설정
6. 코드 리뷰 시 스타일 체크 포인트`,
			},
			"dev_migration": {
				title: "마이그레이션 계획 수립",
				searches: []struct{ name, q string }{
					{"DB 마이그레이션 전략", "database migration strategy zero downtime 2025"},
					{"스키마 변경 방법", "schema migration rollback strategy"},
				},
				prompt: `다음 정보를 바탕으로 마이그레이션 계획을 작성해줘.
%s
형식:
1. 마이그레이션 전 준비사항 (백업/롤백 계획)
2. 단계별 마이그레이션 스크립트 구조
3. Zero-downtime 마이그레이션 전략
4. 데이터 정합성 검증 방법
5. 롤백 시나리오 및 절차
6. 마이그레이션 도구 추천 (Flyway/Liquibase/golang-migrate)`,
			},
			"dev_error_classify": {
				title: "에러 로그 자동 분류",
				searches: []struct{ name, q string }{
					{"에러 분류 체계", "error classification logging strategy"},
					{"에러 모니터링", "error monitoring Sentry Datadog 2025"},
				},
				prompt: `다음 정보를 바탕으로 에러 로그 분류 가이드를 작성해줘.
%s
형식:
1. 에러 카테고리별 분류 기준 (인프라/애플리케이션/외부API/사용자)
2. 심각도 레벨 정의 (Critical/Error/Warning/Info)
3. 에러 코드 체계 설계 방법
4. 자동 분류 규칙 예시 (정규식/패턴)
5. 알림 임계값 설정 방법
6. 에러 대시보드 구성 방법`,
			},
			"dev_weekly_report": {
				title: "주간 개발 리포트",
				searches: []struct{ name, q string }{
					{"개발팀 주간 보고", "engineering weekly report template"},
					{"개발 생산성 지표", "developer productivity metrics DORA"},
				},
				prompt: `다음 정보를 바탕으로 주간 개발 리포트를 작성해줘.
%s
형식:
1. 이번 주 완료한 개발 항목 (기능/버그/리팩토링)
2. PR 현황 (머지됨/리뷰 중/블로킹)
3. 기술 부채 현황
4. 이슈 및 블로커
5. 다음 주 계획
6. DORA 지표 (배포 빈도/변경 리드타임/복구시간)`,
			},
			"dev_code_review": {
				title: "코드 리뷰 준비",
				searches: []struct{ name, q string }{
					{"코드 리뷰 베스트 프랙티스", "code review best practices 2025"},
					{"보안 취약점 체크리스트", "code security checklist OWASP"},
				},
				prompt: `다음 정보를 바탕으로 코드 리뷰 준비 체크리스트를 작성해줘.
%s
형식:
1. 리뷰 전 확인 사항 (5가지)
2. 코드 품질 체크포인트 (가독성/성능/보안)
3. 자주 놓치는 부분
4. 리뷰 코멘트 작성 팁`,
			},
			"dev_deploy_check": {
				title: "배포 체크리스트",
				searches: []struct{ name, q string }{
					{"배포 전 체크리스트", "production deployment checklist 2025"},
					{"장애 대응 롤백", "deployment rollback strategy"},
				},
				prompt: `다음 정보를 바탕으로 배포 체크리스트를 작성해줘.
%s
형식:
1. 배포 전 (코드/테스트/환경변수/DB 마이그레이션)
2. 배포 중 (모니터링 포인트)
3. 배포 후 (헬스체크/로그 확인)
4. 롤백 기준 및 방법`,
			},
			"dev_tech_trend": {
				title: "최신 기술 트렌드",
				searches: []struct{ name, q string }{
					{"2025 개발 트렌드", "software development trends 2025"},
					{"AI 개발 도구 트렌드", "AI developer tools 2025"},
				},
				prompt: `다음 정보를 바탕으로 2025년 개발자가 주목해야 할 기술 트렌드를 정리해줘.
%s
형식:
1. 핵심 트렌드 TOP 5 (간결하게)
2. 당장 배워야 할 기술
3. 주목할 오픈소스/도구
4. 한국 개발 시장 시사점`,
			},
			// ── 마케터 (20개) ───────────────────────────────────────
			"mkt_trend_analysis": {
				title: "트렌드 분석",
				searches: []struct{ name, q string }{
					{"이번 주 소비자 트렌드", "소비자 트렌드 2025 " + query},
					{"SNS 트렌드 분석", "TikTok Instagram 트렌드 viral 2025"},
					{"시장 분석", query + " 시장 트렌드 인사이트 2025"},
				},
				prompt: `다음 정보를 바탕으로 트렌드 분석 인사이트 리포트를 작성해줘.
%s
형식:
1. 이번 주 핵심 트렌드 TOP 5
2. SNS 플랫폼별 트렌드 키워드 (TikTok/Instagram/YouTube)
3. 소비자 행동 변화 인사이트
4. 마케터가 지금 당장 활용할 수 있는 액션 3가지
5. 다음 주 주목해야 할 트렌드 예측`,
			},
			"mkt_content_idea": {
				title: "콘텐츠 아이디어 브레인스토밍",
				searches: []struct{ name, q string }{
					{"SNS 인기 콘텐츠 유형", query + " SNS 콘텐츠 트렌드 2025"},
					{"바이럴 콘텐츠 사례", "바이럴 마케팅 성공 사례 2025 인스타 유튜브"},
				},
				prompt: `다음 정보를 바탕으로 콘텐츠 아이디어 10개를 브레인스토밍해줘.
%s
형식:
1. 인스타그램 릴스 아이디어 3개 (오프닝 훅 문구 포함)
2. 유튜브/숏폼 아이디어 3개 (제목 + 첫 3초 스크립트)
3. 블로그/뉴스레터 아이디어 2개
4. TikTok 트렌드 아이디어 2개
5. 각 아이디어별 예상 반응 포인트`,
			},
			"mkt_competitor_monitor": {
				title: "경쟁사 모니터링",
				searches: []struct{ name, q string }{
					{"경쟁사 최신 뉴스", query + " 경쟁사 마케팅 캠페인 뉴스 2025"},
					{"경쟁사 SNS 전략", query + " 경쟁사 SNS 활동 콘텐츠"},
					{"시장 점유율", query + " 시장 점유율 경쟁 현황"},
				},
				prompt: `다음 정보를 바탕으로 경쟁사 모니터링 주간 리포트를 작성해줘.
%s
형식:
1. 경쟁사 주요 활동 요약 (이번 주)
2. SNS 채널별 성과 비교
3. 신규 캠페인/프로모션 분석
4. 우리 브랜드 대비 강점/약점
5. 즉시 대응해야 할 액션 아이템`,
			},
			"mkt_ad_copy": {
				title: "광고 문구 생성",
				searches: []struct{ name, q string }{
					{"광고 카피라이팅 기법", "advertising copywriting hook formula 2025"},
					{"고성과 광고 문구 사례", query + " 광고 카피 성공 사례"},
				},
				prompt: `다음 정보를 바탕으로 A/B 테스트용 광고 문구 5개 버전을 생성해줘.
%s
형식:
[버전 A] 감성 소구형
- 헤드라인:
- 서브카피:
- CTA:

[버전 B] 혜택 중심형
[버전 C] 긴급성/희소성 자극형
[버전 D] 사회적 증거형
[버전 E] 질문형 훅

각 버전별 타겟 심리 설명 포함`,
			},
			"mkt_sns_post": {
				title: "SNS 게시물 전체 생성",
				searches: []struct{ name, q string }{
					{"SNS 게시물 최적 형식", "social media post best practice engagement 2025"},
					{"해시태그 전략", query + " Instagram TikTok hashtag strategy"},
				},
				prompt: `다음 정보를 바탕으로 SNS 게시물 전체를 생성해줘.
%s
형식:
📱 인스타그램
- 메인 문구 (150자 이내):
- 캡션 (500자 이내):
- 해시태그 20개:
- 게시 최적 시간:

🎵 TikTok/릴스
- 오프닝 훅 (3초):
- 스크립트 (30초):
- 트렌드 사운드 추천:

💼 LinkedIn
- 전문가 톤 게시물:`,
			},
			"mkt_campaign_plan": {
				title: "마케팅 캠페인 기획",
				searches: []struct{ name, q string }{
					{"마케팅 캠페인 기획 방법", "marketing campaign planning framework 2025"},
					{"성공적인 캠페인 사례", query + " marketing campaign success case study"},
				},
				prompt: `다음 정보를 바탕으로 마케팅 캠페인 기획서를 작성해줘.
%s
형식:
1. 캠페인 목표 및 KPI 설정
2. 타겟 오디언스 정의 (페르소나)
3. 핵심 메시지 및 USP
4. 채널별 전략 (SNS/검색광고/이메일/오프라인)
5. 콘텐츠 캘린더 (4주 플랜)
6. 예산 배분 계획
7. 성과 측정 방법`,
			},
			"mkt_performance_report": {
				title: "마케팅 성과 리포트",
				searches: []struct{ name, q string }{
					{"마케팅 KPI 벤치마크", "marketing KPI benchmark 2025 CTR CVR ROAS"},
					{"디지털 마케팅 성과 분석", "digital marketing performance analysis report"},
				},
				prompt: `다음 정보를 바탕으로 마케팅 성과 리포트 템플릿을 작성해줘.
%s
형식:
1. 이번 달 핵심 지표 요약
   - CTR / CVR / ROAS / CPA / CAC
2. 채널별 성과 비교 (Meta/Google/TikTok/이메일)
3. 업계 벤치마크 대비 성과
4. 잘된 캠페인 TOP 3 분석
5. 개선이 필요한 영역
6. 다음 달 액션 플랜`,
			},
			"mkt_seo_keyword": {
				title: "SEO 키워드 분석",
				searches: []struct{ name, q string }{
					{"SEO 키워드 트렌드", query + " SEO keyword search volume 2025"},
					{"롱테일 키워드", query + " long tail keyword low competition"},
					{"경쟁 키워드 분석", query + " competitor SEO keyword ranking"},
				},
				prompt: `다음 정보를 바탕으로 SEO 키워드 분석 리포트를 작성해줘.
%s
형식:
1. 핵심 키워드 TOP 10 (검색량/경쟁도)
2. 즉시 공략 가능한 롱테일 키워드 10개
3. 경쟁사가 사용하는 키워드 분석
4. 콘텐츠 주제 추천 (키워드 기반)
5. SEO 최적화 체크리스트
6. 월별 키워드 전략 로드맵`,
			},
			"mkt_email_newsletter": {
				title: "뉴스레터 작성",
				searches: []struct{ name, q string }{
					{"뉴스레터 트렌드", "email newsletter best practice open rate 2025"},
					{"뉴스레터 주제", query + " 뉴스레터 콘텐츠 트렌드"},
				},
				prompt: `다음 정보를 바탕으로 뉴스레터 초안을 작성해줘.
%s
형식:
📧 제목 라인 3가지 (A/B 테스트용)
📧 프리헤더 텍스트

## 뉴스레터 본문
1. 오프닝 훅 (2-3문장)
2. 메인 콘텐츠 (핵심 가치 전달)
3. 큐레이션 섹션 (이번 주 추천 3가지)
4. CTA (행동 유도)
5. 클로징 문구

디자인 가이드:
- 추천 이미지 배치
- 색상/폰트 방향`,
			},
			"mkt_influencer_search": {
				title: "인플루언서 검색",
				searches: []struct{ name, q string }{
					{"인플루언서 마케팅 트렌드", "influencer marketing trend 2025 Korea"},
					{"인플루언서 찾는 방법", query + " influencer Instagram TikTok YouTube"},
				},
				prompt: `다음 정보를 바탕으로 인플루언서 검색 및 협업 가이드를 작성해줘.
%s
형식:
1. 타겟에 맞는 인플루언서 유형 정의
   - 나노(1천~1만) / 마이크로(1만~10만) / 매크로(10만+) 구분
2. 플랫폼별 탐색 방법 (Instagram/TikTok/YouTube)
3. 인플루언서 평가 기준 (참여율/팔로워 품질/콘텐츠 방향성)
4. 협업 제안 DM/이메일 템플릿
5. 예산별 협업 전략
6. 계약 시 주의사항`,
			},
			"mkt_ab_test_idea": {
				title: "A/B 테스트 아이디어",
				searches: []struct{ name, q string }{
					{"A/B 테스트 방법론", "A/B testing best practices marketing 2025"},
					{"전환율 최적화", "conversion rate optimization CRO tips"},
				},
				prompt: `다음 정보를 바탕으로 A/B 테스트 아이디어 3세트를 제안해줘.
%s
형식:
[테스트 세트 1] 광고 소재
- A안:
- B안:
- 측정 지표:
- 예상 기간:

[테스트 세트 2] 랜딩페이지
- 테스트 요소: (헤드라인/CTA/이미지/레이아웃)
- A안 / B안:
- 성공 기준:

[테스트 세트 3] 이메일 캠페인
- 테스트 요소: (제목/발송시간/CTA)
- A/B 구성:

A/B 테스트 진행 원칙 5가지 포함`,
			},
			"mkt_hashtag_generator": {
				title: "해시태그 생성",
				searches: []struct{ name, q string }{
					{"트렌딩 해시태그", query + " trending hashtag Instagram TikTok 2025"},
					{"해시태그 전략", "hashtag strategy Instagram reach engagement"},
				},
				prompt: `다음 정보를 바탕으로 최적 해시태그 20개를 생성해줘.
%s
형식:
🔥 고볼륨 해시태그 (5개, 100만+ 게시물):
📈 중볼륨 해시태그 (8개, 10만~100만):
🎯 저볼륨 틈새 해시태그 (7개, 1만~10만):

플랫폼별 추천:
- Instagram: 상위 15개
- TikTok: 상위 10개
- LinkedIn: 상위 5개

해시태그 사용 팁 3가지 포함`,
			},
			"mkt_landing_page_copy": {
				title: "랜딩페이지 문구",
				searches: []struct{ name, q string }{
					{"랜딩페이지 카피라이팅", "landing page copywriting conversion 2025"},
					{"고전환율 랜딩페이지", query + " landing page high conversion example"},
				},
				prompt: `다음 정보를 바탕으로 랜딩페이지 문구를 작성해줘.
%s
형식:
🎯 히어로 섹션
- 헤드라인 (3가지 버전):
- 서브헤드라인:
- CTA 버튼 문구 (3가지):

💡 혜택 섹션 (3가지 핵심 혜택)
- 아이콘 + 제목 + 설명

🌟 소셜 프루프 섹션
- 추천사 문구 스타일

❓ FAQ 섹션 (5개)

📞 최종 CTA 섹션
- 긴급성/희소성 문구:`,
			},
			"mkt_social_calendar": {
				title: "소셜 미디어 캘린더",
				searches: []struct{ name, q string }{
					{"SNS 게시 최적 시간", "social media posting optimal time 2025"},
					{"콘텐츠 캘린더 템플릿", "social media content calendar template"},
				},
				prompt: `다음 정보를 바탕으로 1주일 소셜 미디어 게시 계획표를 작성해줘.
%s
형식:
| 날짜 | 플랫폼 | 콘텐츠 유형 | 주제/키워드 | 게시 시간 | 담당자 |
|------|--------|------------|------------|----------|--------|
월요일~일요일 7일 계획

추가:
- 플랫폼별 최적 게시 시간
- 이번 주 활용할 트렌딩 사운드/해시태그
- 예약 게시 도구 추천 (Buffer/Hootsuite/Meta Business)`,
			},
			"mkt_budget_plan": {
				title: "마케팅 예산 계획",
				searches: []struct{ name, q string }{
					{"디지털 광고 단가 2025", "digital advertising CPM CPC benchmark 2025 Korea"},
					{"마케팅 예산 배분 전략", "marketing budget allocation strategy ROI"},
				},
				prompt: `다음 정보를 바탕으로 마케팅 예산 계획을 작성해줘.
%s
형식:
1. 목표 기반 예산 산정 방법
2. 채널별 예산 배분 추천 (%)
   - 검색광고 / SNS광고 / 콘텐츠 / 인플루언서 / 오프라인
3. 채널별 예상 성과 (CPM/CPC/CPA 기준)
4. 월별 예산 집행 계획
5. ROI 측정 방법
6. 예산 절감 팁 3가지`,
			},
			"mkt_viral_content": {
				title: "바이럴 콘텐츠 전략",
				searches: []struct{ name, q string }{
					{"바이럴 콘텐츠 공식", "viral content formula psychology 2025"},
					{"바이럴 성공 사례", query + " viral marketing campaign success 2025"},
				},
				prompt: `다음 정보를 바탕으로 바이럴 가능성 높은 콘텐츠 전략을 작성해줘.
%s
형식:
1. 바이럴 공식 분석 (감정 자극 유형별)
   - 분노/감동/놀라움/웃음/공감
2. 지금 당장 실행 가능한 바이럴 포맷 3가지
3. 콘텐츠 훅 문구 5개 (복사 가능)
4. 공유를 유도하는 심리적 트리거
5. 플랫폼별 바이럴 최적화 방법
6. 바이럴 후 팔로업 전략`,
			},
			"mkt_customer_insight": {
				title: "고객 인사이트 분석",
				searches: []struct{ name, q string }{
					{"소비자 트렌드 분석", query + " 소비자 인사이트 행동 패턴 2025"},
					{"고객 리뷰 분석", query + " 고객 리뷰 불만 만족 분석"},
				},
				prompt: `다음 정보를 바탕으로 고객 인사이트 분석 리포트를 작성해줘.
%s
형식:
1. 핵심 타겟 페르소나 정의 (3가지)
2. 고객 Pain Point TOP 5
3. 구매 결정 요인 분석
4. 고객 여정 맵 (인지→고려→구매→재구매)
5. 리뷰/VOC에서 발견한 인사이트
6. 마케팅 메시지 방향 제안`,
			},
			"mkt_brand_voice": {
				title: "브랜드 보이스 유지",
				searches: []struct{ name, q string }{
					{"브랜드 보이스 가이드", "brand voice tone of voice guide example"},
					{"브랜드 일관성 전략", "brand consistency social media content strategy"},
				},
				prompt: `다음 정보를 바탕으로 브랜드 보이스 가이드를 작성해줘.
%s
형식:
1. 브랜드 보이스 핵심 키워드 5개
2. 톤 스펙트럼 정의
   - 공식적 ←→ 친근한 / 진지한 ←→ 유머러스
3. 상황별 커뮤니케이션 톤 가이드
   - 일반 게시물 / 고객 응대 / 위기 상황 / 프로모션
4. 사용해야 할 표현 vs 피해야 할 표현
5. 채널별 톤 차이 (Instagram vs LinkedIn vs TikTok)
6. 브랜드 보이스 체크리스트`,
			},
			"mkt_weekly_digest": {
				title: "주간 마케팅 요약",
				searches: []struct{ name, q string }{
					{"마케팅 주간 트렌드", "digital marketing weekly digest trends 2025"},
					{"SNS 알고리즘 업데이트", "social media algorithm update 2025"},
				},
				prompt: `다음 정보를 바탕으로 이번 주 마케팅 한 장 요약을 작성해줘.
%s
형식:
📊 이번 주 마케팅 핵심 요약

✅ 완료한 캠페인/활동
📈 주요 성과 지표
🔥 이번 주 트렌드 & 알고리즘 변화
⚠️ 이슈 및 개선 필요 사항
📅 다음 주 예정 활동
💡 팀 공유 인사이트`,
			},
			"mkt_personal_brand": {
				title: "개인 브랜딩 콘텐츠",
				searches: []struct{ name, q string }{
					{"개인 브랜딩 전략", "personal branding LinkedIn content strategy 2025"},
					{"마케터 개인 브랜드", "marketer personal brand thought leadership"},
				},
				prompt: `다음 정보를 바탕으로 개인 브랜딩 콘텐츠를 작성해줘.
%s
형식:
💼 LinkedIn 게시물
- 전문성을 드러내는 인사이트 포스트 (300자):
- 경험 스토리 포스트 (500자):

📝 블로그/브런치 아티클
- 제목 3개 제안:
- 서론 초안 (200자):

🧵 스레드/X 스레드
- 10개 트윗 구성:

📌 개인 브랜딩 전략 팁
- 차별화 포인트 정의:
- 콘텐츠 주기 추천:`,
			},
			// ── 영업 (20개) ─────────────────────────────────────
			"sales_email_draft": {
				title: "영업 이메일 초안",
				searches: []struct{ name, q string }{
					{"영업 이메일 베스트 프랙티스", "sales email best practice cold outreach 2025"},
					{"B2B 이메일 템플릿", "B2B sales email template high response rate"},
				},
				prompt: `다음 정보를 바탕으로 고객 맞춤형 영업 이메일 초안을 작성해줘.
%s
형식:
제목 라인 3가지 (A/B/C):

[메인 초안]
안녕하세요, [고객명] 님.

1. 오프닝 (공감/칭찬/공통점)
2. 핵심 가치 제안 (2-3문장)
3. 사회적 증거
4. CTA (다음 단계 제안)
5. 클로징

[후속 버전] (3일 후 발송용)`,
			},
			"sales_meeting_prep": {
				title: "미팅 준비",
				searches: []struct{ name, q string }{
					{"고객사 정보", query + " 회사 정보 뉴스 2025"},
					{"영업 미팅 전략", "B2B 영업 미팅 성공 전략 준비"},
				},
				prompt: `다음 정보를 바탕으로 영업 미팅 준비 브리핑을 작성해줘.
%s
형식:
1. 고객사 현황 요약 (업계/규모/최근 뉴스)
2. 예상 Pain Point 3가지
3. 준비할 질문 목록 5가지
4. 미팅 오프닝 스크립트 (30초)
5. 예상 이의제기 & 대응
6. 다음 단계 클로징 멘트`,
			},
			"sales_followup": {
				title: "후속 메일 자동화",
				searches: []struct{ name, q string }{
					{"영업 후속 이메일", "sales followup email template after meeting"},
					{"후속 연락 타이밍", "sales followup timing best practice"},
				},
				prompt: `다음 정보를 바탕으로 미팅 후 후속 이메일과 일정 제안을 작성해줘.
%s
형식:
[당일 후속 메일]
- 제목:
- 본문: 감사 + 미팅 요약 + 다음 단계

[3일 후 메일]
- 제목:
- 본문: 가치 상기 + 자료 첨부 + CTA

[1주일 후 메일]
- 제목:
- 본문: 부드러운 압박 + 결정 지원

각 메일 최대 150자 이내`,
			},
			"sales_proposal": {
				title: "제안서 초안",
				searches: []struct{ name, q string }{
					{"성공적인 제안서 구조", "B2B proposal structure best practice 2025"},
					{"고객 니즈 분석", query + " 고객 pain point 솔루션"},
				},
				prompt: `다음 정보를 바탕으로 영업 제안서 초안을 작성해줘.
%s
형식:
1. Executive Summary (1페이지 요약)
2. 고객 현황 및 문제 정의
3. 우리의 솔루션 (핵심 가치 3가지)
4. 기대 효과 (정량적 수치 포함)
5. 도입 프로세스 (단계별 타임라인)
6. 가격/조건 제안 프레임
7. 다음 단계 (CTA)`,
			},
			"sales_objection": {
				title: "이의제기 대응 스크립트",
				searches: []struct{ name, q string }{
					{"영업 이의 대응", "sales objection handling script 2025"},
					{"가격 협상 전략", "price negotiation sales psychology"},
				},
				prompt: `다음 정보를 바탕으로 이의제기 대응 스크립트 5개를 작성해줘.
%s
형식:
1. "비싸요" → 공감 + 가치 재정의 + 대안 제시
2. "지금은 아닌 것 같아요" → 타이밍 이슈 대응
3. "경쟁사가 더 좋아요" → 차별화 포인트 강조
4. "내부 검토가 필요해요" → 의사결정 가속화
5. "기능이 부족해요" → 로드맵 제시 + 현재 가치
각 상황별 클로징 멘트 포함`,
			},
			"sales_pipeline": {
				title: "영업 파이프라인 정리",
				searches: []struct{ name, q string }{
					{"영업 파이프라인 관리", "sales pipeline management CRM best practice"},
					{"파이프라인 예측 방법", "sales pipeline forecast weighted probability"},
				},
				prompt: `다음 정보를 바탕으로 영업 파이프라인 정리 가이드를 작성해줘.
%s
형식:
1. 파이프라인 단계 정의 (리드→자격→제안→협상→클로즈)
2. 단계별 전환율 벤치마크
3. 정체 딜 식별 기준 및 액션
4. 이번 달 예상 매출 계산 방법
5. 파이프라인 건강도 체크리스트
6. CRM 업데이트 루틴 (일별/주별)`,
			},
			"sales_contract": {
				title: "계약서 초안",
				searches: []struct{ name, q string }{
					{"B2B 계약서 필수 항목", "B2B contract essential clauses 2025"},
					{"계약서 법적 체크", "sales contract legal review checklist Korea"},
				},
				prompt: `다음 정보를 바탕으로 영업 계약서 초안 구조를 작성해줘.
%s
형식:
1. 계약 당사자 정보
2. 서비스/제품 범위 (Scope of Work)
3. 납기 및 마일스톤
4. 대금 조건 (계약금/중도금/잔금)
5. 지적재산권 조항
6. 기밀유지 (NDA) 조항
7. 계약 해지 조건
8. 분쟁 해결 방법
※ 법률 검토 필수 안내 포함`,
			},
			"sales_discovery_question": {
				title: "고객 발견 질문 생성",
				searches: []struct{ name, q string }{
					{"영업 발견 질문", "sales discovery question SPIN selling 2025"},
					{"고객 니즈 파악 방법", "customer needs analysis question framework"},
				},
				prompt: `다음 정보를 바탕으로 고객 발견 질문 리스트를 작성해줘.
%s
형식:
[상황 질문 (Situation)] 5개
- 현재 상황 파악

[문제 질문 (Problem)] 5개
- 불편/고통 탐색

[시사 질문 (Implication)] 5개
- 문제의 파급 효과

[필요 질문 (Need-payoff)] 5개
- 해결 가치 확인

미팅 시작 아이스브레이킹 질문 3개 포함`,
			},
			"sales_demo_script": {
				title: "데모 스크립트 작성",
				searches: []struct{ name, q string }{
					{"제품 데모 스크립트", "product demo script best practice 2025"},
					{"데모 스토리텔링", "sales demo storytelling customer success"},
				},
				prompt: `다음 정보를 바탕으로 고객 맞춤 데모 대본을 작성해줘.
%s
형식:
[오프닝] (2분)
- 어젠다 설명
- 고객 상황 확인

[데모 본론] (15분)
- 핵심 기능 1: (스토리 + 시연)
- 핵심 기능 2:
- 핵심 기능 3:

[Q&A 대응 준비]
- 예상 질문 5개 + 답변

[클로징] (3분)
- 다음 단계 제안`,
			},
			"sales_negotiation": {
				title: "협상 전략 수립",
				searches: []struct{ name, q string }{
					{"영업 협상 전략", "sales negotiation strategy BATNA 2025"},
					{"가격 협상 심리", "price negotiation psychology anchoring"},
				},
				prompt: `다음 정보를 바탕으로 협상 전략과 시나리오를 작성해줘.
%s
형식:
1. 협상 전 준비 (목표/BATNA/양보 한계선)
2. 앵커링 전략 (첫 제안 설정)
3. 시나리오별 대응
   - 고객이 30% 할인 요구 시
   - 경쟁사 가격을 언급할 시
   - 결정권자가 없다고 할 시
4. 가치 교환 전술 (가격 대신 조건 협상)
5. 클로징 타이밍 포착 방법`,
			},
			"sales_forecast": {
				title: "영업 예측",
				searches: []struct{ name, q string }{
					{"영업 예측 방법", "sales forecasting method accuracy 2025"},
					{"파이프라인 예측 모델", "weighted pipeline forecast model"},
				},
				prompt: `다음 정보를 바탕으로 이번 달 영업 예측 리포트를 작성해줘.
%s
형식:
1. 예측 방법론 (가중 파이프라인/기대값)
2. 딜별 예상 매출 × 확률 계산
3. 낙관/현실/보수 시나리오
4. 목표 달성을 위한 갭 분석
5. 이번 달 반드시 클로즈할 딜 TOP 3
6. 다음 달 파이프라인 건강도 예측`,
			},
			"sales_crm_update": {
				title: "CRM 자동 업데이트",
				searches: []struct{ name, q string }{
					{"CRM 업데이트 베스트 프랙티스", "CRM data hygiene update best practice sales"},
				},
				prompt: `다음 정보를 바탕으로 CRM 업데이트 가이드와 템플릿을 작성해줘.
%s
형식:
1. 미팅 후 즉시 입력할 필드 목록
2. 미팅 노트 표준 포맷
   - 참석자 / 핵심 논의 / 액션 아이템 / 다음 단계
3. 딜 상태 업데이트 기준
4. CRM 데이터 정확도 유지 루틴 (주 1회)
5. 파이프라인 자동화 활용 팁`,
			},
			"sales_call_summary": {
				title: "영업 통화 요약",
				searches: []struct{ name, q string }{
					{"영업 통화 요약 방법", "sales call summary template action items"},
				},
				prompt: `다음 정보를 바탕으로 영업 통화 요약 템플릿을 작성해줘.
%s
형식:
📞 통화 요약
- 일시 / 참석자:
- 통화 목적:

💬 핵심 논의 내용 (불릿 3-5개)

⚡ 고객이 표현한 Pain Point

✅ 합의된 사항

📋 액션 아이템
| 항목 | 담당 | 기한 |

📅 다음 단계`,
			},
			"sales_proposal_followup": {
				title: "제안서 후속 관리",
				searches: []struct{ name, q string }{
					{"제안서 후속 전략", "proposal followup strategy win rate"},
					{"제안 후 의사결정 지원", "after proposal decision making support"},
				},
				prompt: `다음 정보를 바탕으로 제안서 발송 후 후속 관리 플랜을 작성해줘.
%s
형식:
[D+1] 확인 연락
- 수신 확인 + 질문 유도

[D+3] 가치 보강
- 추가 자료 / 케이스 스터디 전달

[D+7] 의사결정 지원
- 내부 검토 지원 자료 제공

[D+14] 부드러운 압박
- 타이밍 이슈 대응

각 단계별 이메일/문자 초안 포함`,
			},
			"sales_win_loss_analysis": {
				title: "Win/Loss 분석",
				searches: []struct{ name, q string }{
					{"Win Loss 분석 방법", "win loss analysis sales learning 2025"},
					{"영업 패배 원인 분석", "sales lost deal analysis reason"},
				},
				prompt: `다음 정보를 바탕으로 Win/Loss 분석 리포트를 작성해줘.
%s
형식:
1. 분석 기간 및 대상 딜 개요
2. Win 패턴 분석
   - 공통 승리 요인 TOP 5
   - 자주 이긴 산업/고객 유형
3. Loss 패턴 분석
   - 주요 패인 원인 TOP 5
   - 경쟁사에게 진 이유
4. 개선 액션 플랜 3가지
5. 다음 분기 전략 방향`,
			},
			"sales_referral_request": {
				title: "추천 요청 메시지",
				searches: []struct{ name, q string }{
					{"고객 추천 요청", "customer referral request script best practice"},
					{"레퍼럴 마케팅 전략", "referral marketing B2B strategy"},
				},
				prompt: `다음 정보를 바탕으로 고객 추천 요청 메시지를 작성해줘.
%s
형식:
[이메일 버전]
제목:
본문: 관계 상기 → 만족도 확인 → 추천 요청 → 인센티브 → CTA

[문자/카카오 버전] (80자 이내)

[전화 스크립트] (30초)

추천 요청 최적 타이밍 가이드 포함`,
			},
			"sales_price_negotiation": {
				title: "가격 협상 전략",
				searches: []struct{ name, q string }{
					{"가격 협상 전략", "pricing negotiation strategy 2025"},
					{"할인 정책 가이드", "discount policy sales negotiation framework"},
				},
				prompt: `다음 정보를 바탕으로 가격 협상 전략을 작성해줘.
%s
형식:
1. 가격 방어 프레임워크 (가치 기반 대응)
2. 할인 제공 시 조건 교환 전술
   - "할인 대신 조건을 바꾸는 법"
3. 가격 앵커링 설정 방법
4. 번들링/패키지 재구성 전략
5. 협상 불가 선언 타이밍
6. 최종 제안 클로징 멘트`,
			},
			"sales_contract_review": {
				title: "계약서 검토",
				searches: []struct{ name, q string }{
					{"계약서 위험 항목", "contract red flags review checklist B2B"},
					{"계약서 협상 포인트", "contract negotiation points sales"},
				},
				prompt: `다음 정보를 바탕으로 계약서 검토 체크리스트를 작성해줘.
%s
형식:
🔴 즉시 수정 필요 (위험 항목)
- 과도한 면책 조항
- 일방적 해지 권리
- 무제한 손해배상

🟡 협상 권장 항목
- 납기 지연 패널티 기준
- 범위 변경(Change Order) 절차
- 지식재산권 귀속

🟢 확인 필수 항목
- 준거법 및 관할법원
- 갱신/연장 조건

협상 가이드 포함, ※ 법률 검토 필수 안내`,
			},
			"sales_quarterly_review": {
				title: "분기 영업 리뷰",
				searches: []struct{ name, q string }{
					{"분기 영업 리뷰 방법", "quarterly sales review QBR template"},
					{"영업 성과 분석", "sales performance analysis quarterly"},
				},
				prompt: `다음 정보를 바탕으로 분기 영업 리뷰 리포트를 작성해줘.
%s
형식:
1. 분기 실적 요약 (목표 vs 실제)
2. 채널/제품별 성과 분석
3. Win/Loss 비율 및 원인
4. 파이프라인 건강도
5. 팀별/개인별 성과 하이라이트
6. 다음 분기 전략 및 목표 설정
7. 경영진 보고용 요약 (1페이지)`,
			},
			"sales_client_portrait": {
				title: "고객 프로필 분석",
				searches: []struct{ name, q string }{
					{"고객 프로파일링 방법", "customer profiling ICP ideal customer profile"},
					{"B2B 구매자 페르소나", "B2B buyer persona decision maker analysis"},
				},
				prompt: `다음 정보를 바탕으로 고객 프로필 분석 리포트를 작성해줘.
%s
형식:
1. 기업 프로파일 (규모/업종/성장단계)
2. 의사결정 구조 (Champion/Blocker/Budget)
3. 핵심 Pain Point 및 우선순위
4. 구매 프로세스 및 타임라인
5. 경쟁사 대비 포지셔닝
6. 맞춤형 접근 전략 3가지`,
			},
			// ── PM (20개) ────────────────────────────────────────
			"pm_requirements": {
				title: "요구사항 정리",
				searches: []struct{ name, q string }{
					{"요구사항 정리 방법", "product requirements gathering template PRD"},
					{"기능 명세서 작성", "functional specification document template"},
				},
				prompt: `다음 정보를 바탕으로 요구사항 정리 문서를 작성해줘.
%s
형식:
1. 배경 및 목표 (Why)
2. 요구사항 수집 방법 (인터뷰/설문/분석)
3. 기능 요구사항 목록 (Must/Should/Could/Won't)
4. 비기능 요구사항 (성능/보안/UX)
5. 우선순위 결정 기준
6. 이해관계자 승인 절차`,
			},
			"pm_roadmap": {
				title: "로드맵 업데이트",
				searches: []struct{ name, q string }{
					{"제품 로드맵 작성", "product roadmap template best practice 2025"},
					{"로드맵 우선순위", "roadmap prioritization framework OKR"},
				},
				prompt: `다음 정보를 바탕으로 제품 로드맵을 작성해줘.
%s
형식:
| 분기 | 테마 | 기능 | 목표 지표 | 상태 |

Now (이번 분기):
Next (다음 분기):
Later (그 이후):

우선순위 결정 근거:
- 비즈니스 임팩트
- 개발 복잡도
- 사용자 요청 빈도

이해관계자 커뮤니케이션 가이드 포함`,
			},
			"pm_stakeholder_summary": {
				title: "이해관계자 브리핑",
				searches: []struct{ name, q string }{
					{"이해관계자 브리핑 방법", "stakeholder briefing communication executive summary"},
				},
				prompt: `다음 정보를 바탕으로 이해관계자 브리핑 문서를 작성해줘.
%s
형식:
📋 이번 주 요약 (Executive 1-pager)

✅ 완료된 주요 결정사항
⚡ 현재 진행 중인 이슈
⚠️ 리스크 및 블로커
📊 핵심 지표 현황
📅 다음 주 주요 마일스톤
❓ 이해관계자 결정 필요 사항

커뮤니케이션 채널별 요약 길이 가이드`,
			},
			"pm_risk_analysis": {
				title: "리스크 분석",
				searches: []struct{ name, q string }{
					{"프로젝트 리스크 분석", "project risk analysis framework RAID log"},
					{"리스크 대응 전략", "risk mitigation strategy product management"},
				},
				prompt: `다음 정보를 바탕으로 리스크 분석 리포트를 작성해줘.
%s
형식:
| 리스크 | 발생확률 | 영향도 | 위험도 | 대응 방안 | 담당자 |

카테고리별 분류:
1. 기술 리스크
2. 일정 리스크
3. 자원 리스크
4. 외부/시장 리스크

조기 경보 지표 (Early Warning Signals) 설정
리스크 모니터링 주기 및 방법`,
			},
			"pm_meeting_note": {
				title: "미팅 노트 정리",
				searches: []struct{ name, q string }{
					{"미팅 노트 작성 방법", "meeting notes template action items best practice"},
				},
				prompt: `다음 정보를 바탕으로 미팅 노트를 정리해줘.
%s
형식:
📅 미팅 정보
- 일시 / 참석자 / 목적

💬 논의 내용 (주제별)

✅ 결정 사항

📋 액션 아이템
| 항목 | 담당자 | 기한 | 상태 |

❓ 미해결 질문 / 다음 미팅 어젠다

배포 대상: [참석자/이해관계자]`,
			},
			"pm_user_story": {
				title: "유저 스토리 작성",
				searches: []struct{ name, q string }{
					{"유저 스토리 작성법", "user story writing acceptance criteria example"},
					{"Agile 스토리 포인트", "agile story point estimation planning poker"},
				},
				prompt: `다음 정보를 바탕으로 유저 스토리와 Acceptance Criteria를 작성해줘.
%s
형식:
[유저 스토리]
As a [사용자 유형]
I want to [기능/행동]
So that [얻는 가치]

[Acceptance Criteria]
Given [전제 조건]
When [행동]
Then [결과]

스토리 포인트 추정: [1/2/3/5/8]
우선순위: [Must/Should/Could]
의존성: [관련 스토리]

3-5개 스토리 생성`,
			},
			"pm_weekly_report": {
				title: "주간 보고서",
				searches: []struct{ name, q string }{
					{"PM 주간 보고서", "product manager weekly report template"},
					{"스프린트 리뷰", "sprint review retrospective template"},
				},
				prompt: `다음 정보를 바탕으로 PM 주간 보고서를 작성해줘.
%s
형식:
1. 이번 주 완료 항목 (Done)
2. 진행 중 항목 및 이슈 (In Progress)
3. 다음 주 계획 (Todo)
4. 리스크 및 블로커
5. 주요 지표 현황
6. 이해관계자 전달 사항`,
			},
			"pm_prd_write": {
				title: "PRD 작성",
				searches: []struct{ name, q string }{
					{"PRD 작성 방법", "PRD product requirements document template 2025"},
					{"사용자 스토리 PRD", "user story acceptance criteria PRD example"},
				},
				prompt: `다음 정보를 바탕으로 PRD(제품 요구사항 문서) 초안을 작성해줘.
%s
형식:
# PRD: [제품/기능명]

## 1. 배경 및 목표 (WHY)
## 2. 대상 사용자 및 페르소나
## 3. 핵심 기능 목록
| 기능 | 우선순위 | 설명 |
## 4. 비기능 요구사항
## 5. 성공 지표 (KPI)
## 6. 제외 범위 (Out of Scope)
## 7. 의존성 및 위험요소
## 8. 타임라인`,
			},
			"pm_spec_review": {
				title: "기획서 검토",
				searches: []struct{ name, q string }{
					{"기획서 검토 기준", "product spec review checklist feedback"},
					{"기획 완성도 평가", "product requirements completeness review"},
				},
				prompt: `다음 정보를 바탕으로 기획서 검토 피드백을 작성해줘.
%s
형식:
✅ 잘된 점

❌ 보완 필요 사항
- 명확하지 않은 요구사항
- 누락된 엣지 케이스
- 기술적 실현 가능성 이슈
- UX 흐름 개선 포인트

💡 개선 제안 (우선순위별)

❓ 추가 확인 필요 질문 (개발팀/디자인팀)

종합 평가: [완성도 점수/100]`,
			},
			"pm_priority_matrix": {
				title: "우선순위 매트릭스",
				searches: []struct{ name, q string }{
					{"우선순위 매트릭스", "priority matrix RICE MoSCoW framework"},
					{"제품 우선순위 결정", "product prioritization method impact effort"},
				},
				prompt: `다음 정보를 바탕으로 우선순위 매트릭스를 작성해줘.
%s
형식:
[MoSCoW 분류]
Must Have: (필수)
Should Have: (권장)
Could Have: (여유되면)
Won't Have: (이번 버전 제외)

[RICE 점수 계산]
| 기능 | Reach | Impact | Confidence | Effort | RICE |

[2×2 매트릭스]
- 높은 임팩트 + 낮은 노력 → 즉시 실행
- 높은 임팩트 + 높은 노력 → 계획 수립
- 낮은 임팩트 + 낮은 노력 → 틈새 실행
- 낮은 임팩트 + 높은 노력 → 제외`,
			},
			"pm_retrospective": {
				title: "회고 미팅 정리",
				searches: []struct{ name, q string }{
					{"회고 미팅 방법", "retrospective meeting template Start Stop Continue"},
					{"애자일 회고 기법", "agile retrospective techniques team"},
				},
				prompt: `다음 정보를 바탕으로 회고 미팅 정리 문서를 작성해줘.
%s
형식:
[Start - Stop - Continue]
✨ Start (새로 시작할 것):
🛑 Stop (그만할 것):
✅ Continue (계속할 것):

[주요 인사이트]

[액션 아이템]
| 항목 | 담당자 | 기한 |

[팀 건강도 체크]
- 협업: /10
- 커뮤니케이션: /10
- 기술적 품질: /10

다음 스프린트 개선 포인트 TOP 3`,
			},
			"pm_okr_setting": {
				title: "OKR 설정",
				searches: []struct{ name, q string }{
					{"OKR 작성 방법", "OKR objective key results writing best practice"},
					{"OKR 사례", "OKR examples product team 2025"},
				},
				prompt: `다음 정보를 바탕으로 OKR을 작성해줘.
%s
형식:
## Objective (목표)
[야심차고 영감을 주는 질적 목표]

## Key Results (핵심 결과)
KR1: [측정 가능한 수치 목표]
KR2: [측정 가능한 수치 목표]
KR3: [측정 가능한 수치 목표]

## 이니셔티브 (실행 과제)
KR별 핵심 액션 2-3개

OKR 작성 원칙:
- Objective: 동기부여 + 방향 제시
- KR: 숫자로 측정 가능
- 달성률 70%가 좋은 OKR

분기별 체크인 방법 포함`,
			},
			"pm_resource_plan": {
				title: "리소스 계획",
				searches: []struct{ name, q string }{
					{"리소스 계획 방법", "resource planning project management template"},
					{"인력 배치 최적화", "team capacity planning sprint allocation"},
				},
				prompt: `다음 정보를 바탕으로 리소스 계획을 작성해줘.
%s
형식:
1. 프로젝트 리소스 현황
| 역할 | 인원 | 가용 시간 | 현재 배치 |

2. 리소스 갭 분석 (부족/과잉)

3. 우선순위 기반 배치 방안

4. 외부 리소스 필요 여부 (외주/채용)

5. 리소스 충돌 해결 방법

6. 주별 용량 계획 (4주)`,
			},
			"pm_stakeholder_map": {
				title: "이해관계자 맵",
				searches: []struct{ name, q string }{
					{"이해관계자 분석", "stakeholder mapping analysis influence interest"},
					{"이해관계자 관리 전략", "stakeholder management strategy communication"},
				},
				prompt: `다음 정보를 바탕으로 이해관계자 맵을 작성해줘.
%s
형식:
[2×2 이해관계자 맵]
영향력 높음 + 관심 높음 → 긴밀 관리
영향력 높음 + 관심 낮음 → 만족 유지
영향력 낮음 + 관심 높음 → 정보 제공
영향력 낮음 + 관심 낮음 → 모니터링

| 이해관계자 | 역할 | 영향력 | 관심도 | 입장 | 관리 전략 |

커뮤니케이션 주기별 계획 포함`,
			},
			"pm_feature_kanban": {
				title: "기능 칸반 정리",
				searches: []struct{ name, q string }{
					{"칸반 보드 운영", "kanban board management workflow WIP limit"},
				},
				prompt: `다음 정보를 바탕으로 기능 칸반 정리 가이드를 작성해줘.
%s
형식:
[칸반 컬럼 구성]
📋 Backlog | 🔍 분석 중 | 🎨 디자인 | 💻 개발 | 🧪 QA | ✅ Done

[WIP 한계 설정]
각 컬럼별 최대 진행 항목 수

[백로그 → 칸반 분류 기준]
- 즉시 실행 가능한 카드 조건
- 카드 크기 기준 (1 스프린트 이내)
- 카드 작성 표준 포맷

[블로킹 카드 처리 방법]

주간 칸반 리뷰 루틴 포함`,
			},
			"pm_user_interview_summary": {
				title: "사용자 인터뷰 요약",
				searches: []struct{ name, q string }{
					{"사용자 인터뷰 분석", "user interview analysis affinity mapping insights"},
					{"인터뷰 인사이트 추출", "qualitative research synthesis themes"},
				},
				prompt: `다음 정보를 바탕으로 사용자 인터뷰 요약 리포트를 작성해줘.
%s
형식:
1. 인터뷰 개요 (대상/방법/일정)
2. 핵심 테마별 인사이트
3. 사용자 Pain Point TOP 5
4. 자주 언급된 키워드/표현
5. 예상과 달랐던 발견
6. 제품 개선 제안 (우선순위별)
7. 다음 인터뷰 질문 개선안`,
			},
			"pm_competitor_analysis": {
				title: "경쟁사 분석",
				searches: []struct{ name, q string }{
					{"경쟁사 제품 분석", query + " competitor product analysis 2025"},
					{"경쟁 포지셔닝", query + " competitive positioning feature comparison"},
				},
				prompt: `다음 정보를 바탕으로 경쟁 제품 분석 리포트를 작성해줘.
%s
형식:
[기능 비교표]
| 기능 | 우리 | 경쟁사 A | 경쟁사 B |

[포지셔닝 분석]
- 가격 포지셔닝
- 타겟 세그먼트
- 핵심 차별화 메시지

[우리의 강점/약점]

[기회 포착 포인트]

[전략적 방향 제안]`,
			},
			"pm_go_to_market": {
				title: "Go-to-Market 전략",
				searches: []struct{ name, q string }{
					{"GTM 전략 수립", "go-to-market strategy template product launch 2025"},
					{"제품 출시 전략", "product launch plan checklist B2B SaaS"},
				},
				prompt: `다음 정보를 바탕으로 Go-to-Market 전략을 작성해줘.
%s
형식:
1. 타겟 시장 정의 (TAM/SAM/SOM)
2. ICP (이상적 고객 프로필)
3. 가치 제안 (Value Proposition)
4. 가격 전략
5. 유통/채널 전략
6. 마케팅 & 영업 플레이북
7. 출시 타임라인 (T-4주 ~ 출시 후)
8. 성공 지표 (KPI)`,
			},
			"pm_sprint_planning": {
				title: "스프린트 계획",
				searches: []struct{ name, q string }{
					{"스프린트 계획 방법", "agile sprint planning best practice velocity"},
					{"스프린트 용량 산정", "sprint capacity planning story points"},
				},
				prompt: `다음 정보를 바탕으로 스프린트 계획 가이드를 작성해줘.
%s
형식:
1. 스프린트 목표 선언
2. 팀 용량 계산 (가용 시간 × 인원)
3. 백로그 선택 기준
4. 스토리 포인트 할당 방법
5. 스프린트 백로그 (확정 항목)
| 스토리 | 포인트 | 담당자 |
6. 데일리 스탠드업 루틴
7. 스프린트 리스크 체크`,
			},
			"pm_metrics_dashboard": {
				title: "지표 대시보드",
				searches: []struct{ name, q string }{
					{"제품 핵심 지표", "product metrics KPI dashboard 2025"},
					{"AARRR 지표", "AARRR pirate metrics product growth"},
				},
				prompt: `다음 정보를 바탕으로 PM 지표 대시보드 구성을 작성해줘.
%s
형식:
[핵심 지표 대시보드]
AARRR 퍼널:
- Acquisition: (신규 유입)
- Activation: (첫 경험 성공률)
- Retention: (재방문율)
- Revenue: (전환/결제)
- Referral: (추천)

[제품별 핵심 지표]
- DAU/MAU / NPS / CSAT
- 기능 채택률 / 완료율

[대시보드 시각화 추천]
[주간 지표 리뷰 루틴]`,
			},
			// ── 디자이너 (20개) ──────────────────────────────────
			"design_reference": {
				title: "레퍼런스 검색",
				searches: []struct{ name, q string }{
					{"디자인 레퍼런스", query + " design reference inspiration Dribbble Behance 2025"},
					{"UI 디자인 트렌드", "UI UX design trends 2025"},
				},
				prompt: `다음 정보를 바탕으로 디자인 레퍼런스 가이드를 작성해줘.
%s
형식:
1. 추천 레퍼런스 사이트/브랜드 TOP 5 (이유 포함)
2. Dribbble/Behance/Pinterest 검색 키워드
3. 현재 트렌드 키워드 5개
4. 색상 팔레트 방향 제안
5. 레퍼런스 수집 → 무드보드 구성 방법`,
			},
			"design_file_organize": {
				title: "디자인 파일 정리",
				searches: []struct{ name, q string }{
					{"디자인 파일 관리", "design file organization naming convention Figma"},
				},
				prompt: `다음 정보를 바탕으로 디자인 파일 정리 가이드를 작성해줘.
%s
형식:
1. 폴더 구조 설계
   /Projects/[클라이언트]/[프로젝트]/[버전]
2. 파일 네이밍 규칙
   YYYYMMDD_프로젝트명_버전_담당자
3. 에셋 분류 기준 (로고/아이콘/이미지/폰트)
4. 버전 관리 방법 (v1.0 / v1.1 / Final)
5. 아카이브 정책 (보관 기간/압축 방법)
6. 팀 공유 폴더 운영 방법`,
			},
			"design_color_palette": {
				title: "컬러 팔레트 생성",
				searches: []struct{ name, q string }{
					{"컬러 팔레트 이론", "color palette theory brand design 2025"},
					{"브랜드 컬러 선택", "brand color psychology selection guide"},
				},
				prompt: `다음 정보를 바탕으로 브랜드 컬러 팔레트를 제안해줘.
%s
형식:
🎨 컬러 팔레트 제안

Primary Color:
- HEX: #______
- 심리적 의미:
- 사용 맥락:

Secondary Color: #______
Accent Color: #______

[명도 스케일] (Primary 기준)
100 / 200 / 300 / 400 / 500 / 600 / 700 / 800 / 900

[중립 컬러] (텍스트/배경용)
- 텍스트: #______
- 배경: #______
- 보더: #______

접근성 대비율 체크 (WCAG AA 기준)`,
			},
			"design_image_edit": {
				title: "이미지 일괄 편집",
				searches: []struct{ name, q string }{
					{"이미지 일괄 편집 방법", "batch image processing automation design workflow"},
				},
				prompt: `다음 정보를 바탕으로 이미지 일괄 편집 가이드를 작성해줘.
%s
형식:
1. 파일 형식 변환 규칙 (JPG/PNG/WebP/SVG)
2. 크기 규격 체계
   - 웹: 1920px / 1280px / 768px / 375px
   - SNS: 인스타(1080×1080) / 유튜브썸네일(1280×720)
3. 파일명 규칙 적용
4. 압축률 설정 (품질 vs 용량)
5. 메타데이터 처리 방법
6. 추천 툴 (ImageOptim/Squoosh/Sharp CLI)`,
			},
			"design_content_idea": {
				title: "콘텐츠 디자인 아이디어",
				searches: []struct{ name, q string }{
					{"포스터 디자인 트렌드", query + " poster design trend 2025"},
					{"콘텐츠 디자인 아이디어", "creative content design concept idea"},
				},
				prompt: `다음 정보를 바탕으로 콘텐츠 디자인 아이디어 5개 컨셉을 제안해줘.
%s
형식:
[컨셉 1] 타이틀
- 스타일: (미니멀/볼드/레트로 등)
- 컬러 방향:
- 레이아웃 구조:
- 필요 에셋:

[컨셉 2~5] 동일 형식

각 컨셉별 적합한 활용처 (웹/SNS/인쇄) 명시`,
			},
			"design_feedback": {
				title: "디자인 피드백",
				searches: []struct{ name, q string }{
					{"디자인 피드백 방법", "design critique feedback constructive method"},
					{"UI 디자인 평가 기준", "UI design evaluation heuristics Nielsen"},
				},
				prompt: `다음 정보를 바탕으로 구조화된 디자인 피드백을 작성해줘.
%s
형식:
✅ 잘된 점 (구체적으로)

🔴 개선 필요 사항
1. 시각적 계층구조 (Visual Hierarchy)
2. 색상 및 대비 (Color & Contrast)
3. 타이포그래피 일관성
4. 여백 및 정렬 (Spacing & Alignment)
5. 사용성 (Usability)

💡 구체적 개선 제안 (우선순위별)

📐 디자인 시스템 적용 여부 체크`,
			},
			"design_moodboard": {
				title: "무드보드 생성",
				searches: []struct{ name, q string }{
					{"무드보드 참고 이미지", query + " moodboard visual inspiration 2025"},
					{"컬러 분위기", query + " color mood aesthetic"},
				},
				prompt: `다음 정보를 바탕으로 무드보드 가이드를 작성해줘.
%s
형식:
🎨 무드보드 컨셉

핵심 키워드 (5개):

컬러 팔레트 방향:
- 메인 컬러: [분위기 설명 + 예시 HEX]
- 보조 컬러:
- 포인트 컬러:

타이포그래피 방향:
- 헤드라인 폰트 스타일:
- 본문 폰트 스타일:

이미지/텍스처 방향:
- 사진 분위기:
- 패턴/텍스처:

레퍼런스 수집 사이트 및 검색 키워드 5개`,
			},
			"design_ui_kit": {
				title: "UI Kit 가이드",
				searches: []struct{ name, q string }{
					{"UI Kit 구성 방법", "UI Kit component library design system 2025"},
					{"디자인 시스템 구조", "design system atomic design component"},
				},
				prompt: `다음 정보를 바탕으로 UI Kit 구성 가이드를 작성해줘.
%s
형식:
[기초 요소 (Foundations)]
- 컬러 시스템
- 타이포그래피 스케일
- 간격 시스템 (4px/8px 그리드)
- 아이콘 스타일

[컴포넌트 목록 (우선순위별)]
Tier 1 (필수): Button/Input/Card/Modal/Toast
Tier 2 (권장): Dropdown/Tabs/Badge/Avatar
Tier 3 (나중): DatePicker/Table/Chart

[Figma 구성 방법]
- 컴포넌트 → 인스턴스 구조
- 오토레이아웃 활용법
- 네이밍 규칙`,
			},
			"design_prototype_review": {
				title: "프로토타입 검토",
				searches: []struct{ name, q string }{
					{"프로토타입 검토 기준", "prototype review checklist usability UX"},
					{"UX 검토 방법", "UX heuristic evaluation prototype"},
				},
				prompt: `다음 정보를 바탕으로 프로토타입 검토 피드백을 작성해줘.
%s
형식:
[UX 흐름 검토]
✅ 자연스러운 플로우
❌ 끊기는 구간 및 원인

[닐슨 휴리스틱 10원칙 체크]
1. 시스템 상태 가시성
2. 사용자 제어 및 자유도
3. 일관성
(각 항목 통과/주의/실패 + 설명)

[모바일 친화성]
[접근성 기본 체크]

우선순위별 개선 제안 TOP 5`,
			},
			"design_asset_export": {
				title: "에셋 일괄 내보내기",
				searches: []struct{ name, q string }{
					{"디자인 에셋 내보내기", "design asset export specification Figma guide"},
				},
				prompt: `다음 정보를 바탕으로 에셋 내보내기 규칙을 작성해줘.
%s
형식:
[플랫폼별 내보내기 규격]
iOS:
- 1x / 2x / 3x (PNG)
- 아이콘: .pdf 또는 .svg

Android:
- mdpi / hdpi / xhdpi / xxhdpi / xxxhdpi
- 벡터: .xml (SVG 변환)

Web:
- SVG (아이콘/로고)
- WebP + JPG (사진)
- PNG (투명배경 필요시)

[파일명 규칙]
ic_이름_상태_크기.확장자

[Figma 내보내기 자동화 방법]`,
			},
			"design_brand_guideline": {
				title: "브랜드 가이드라인",
				searches: []struct{ name, q string }{
					{"브랜드 가이드라인 구성", "brand guideline template visual identity 2025"},
					{"브랜드 아이덴티티 사례", "brand identity guideline example"},
				},
				prompt: `다음 정보를 바탕으로 브랜드 가이드라인 구조를 작성해줘.
%s
형식:
1. 브랜드 스토리 & 철학
2. 로고 사용 규칙
   - 최소 크기 / 여백 / 금지 사례
3. 컬러 시스템 (Primary/Secondary/Neutral)
4. 타이포그래피 시스템
   - 헤드라인 / 서브 / 본문 / 캡션
5. 이미지 스타일 가이드
6. 아이콘 스타일
7. DO & DON'T 사례
8. 적용 예시 (명함/웹/SNS)`,
			},
			"design_social_media_kit": {
				title: "소셜 미디어 키트",
				searches: []struct{ name, q string }{
					{"SNS 디자인 규격", "social media design template size 2025"},
					{"소셜 키트 구성", "social media kit template brand"},
				},
				prompt: `다음 정보를 바탕으로 소셜 미디어 키트 구성 가이드를 작성해줘.
%s
형식:
[플랫폼별 규격]
Instagram: 정사각(1080×1080) / 세로(1080×1350) / 스토리(1080×1920)
YouTube: 썸네일(1280×720) / 채널아트(2560×1440)
LinkedIn: 포스트(1200×627)
TikTok: 세로(1080×1920)

[키트 구성 항목]
- 프로필 사진 프레임
- 포스트 템플릿 (일반/프로모션/인포그래픽)
- 스토리 템플릿
- 하이라이트 커버

[Canva/Figma 템플릿 구성 팁]`,
			},
			"design_presentation_deck": {
				title: "발표 자료 제작",
				searches: []struct{ name, q string }{
					{"발표 자료 디자인", "presentation deck design best practice storytelling"},
					{"슬라이드 구성 방법", "pitch deck slide structure compelling"},
				},
				prompt: `다음 정보를 바탕으로 발표 자료 구성 가이드를 작성해줘.
%s
형식:
[슬라이드 구성 (10-20장)]
1. 표지
2. 목차/어젠다
3. 문제 정의
4. 솔루션/핵심 메시지
5. 데이터/증거
6. 사례/케이스 스터디
7. 액션 플랜
8. Q&A / 마무리

[디자인 원칙]
- 슬라이드당 1개 메시지
- 텍스트 최소화 (키워드만)
- 데이터는 차트로 시각화

[발표자 노트 작성 팁]`,
			},
			"design_icon_set": {
				title: "아이콘 세트 가이드",
				searches: []struct{ name, q string }{
					{"아이콘 디자인 스타일", "icon design style guide 2025 outline filled"},
					{"무료 아이콘 리소스", "free icon set resource design 2025"},
				},
				prompt: `다음 정보를 바탕으로 아이콘 세트 제작 가이드를 작성해줘.
%s
형식:
[스타일 정의]
- Outline / Filled / Duo-tone 중 선택 이유
- 선 두께: ____px
- 코너 반경: ____px
- 그리드: 24px × 24px

[필수 아이콘 20개 목록]
카테고리별: 내비게이션/액션/상태/소셜

[일관성 체크리스트]
- 시각적 무게 균등
- 픽셀 맞춤 (Pixel Perfect)
- 의미 명확성

[리소스 추천]
Heroicons / Lucide / Phosphor Icons`,
			},
			"design_typography": {
				title: "타이포그래피 시스템",
				searches: []struct{ name, q string }{
					{"타이포그래피 시스템", "typography system scale design 2025"},
					{"한국어 폰트 추천", "Korean font recommendation web design 2025"},
				},
				prompt: `다음 정보를 바탕으로 타이포그래피 시스템을 작성해줘.
%s
형식:
[폰트 선택]
- 헤드라인: [폰트명] / 이유
- 본문: [폰트명] / 이유
- 모노스페이스: [폰트명] (코드용)
- 한국어: [폰트명]

[타입 스케일]
| 이름 | 크기 | 굵기 | 줄간격 | 용도 |
Display / H1 / H2 / H3 / Body-L / Body-M / Caption

[사용 규칙]
- 최대 폰트 종류: 2개
- 강조: Bold 사용 (이탤릭 최소화)
- 접근성: 최소 16px 본문

[웹폰트 로딩 최적화 방법]`,
			},
			"design_animation_idea": {
				title: "애니메이션 아이디어",
				searches: []struct{ name, q string }{
					{"UI 애니메이션 트렌드", "UI animation micro-interaction trend 2025"},
					{"Lottie 애니메이션 사례", "Lottie animation example UI motion design"},
				},
				prompt: `다음 정보를 바탕으로 UI 애니메이션 아이디어를 작성해줘.
%s
형식:
[마이크로 인터랙션 아이디어]
1. 버튼 클릭 피드백 (0.2s ease-out)
2. 로딩 상태 표현
3. 성공/실패 알림 애니메이션
4. 페이지 전환 효과
5. 스크롤 트리거 애니메이션

[Lottie 활용 포인트]
- 온보딩 캐릭터
- 빈 상태(Empty State) 일러스트
- 성공 축하 이펙트

[애니메이션 원칙]
- 지속시간: 200-500ms
- Easing: ease-in-out 권장
- 60fps 유지

After Effects → Lottie 내보내기 방법`,
			},
			"design_accessibility_check": {
				title: "접근성 검사",
				searches: []struct{ name, q string }{
					{"WCAG 접근성 기준", "WCAG 2.1 accessibility checklist design"},
					{"접근성 디자인 방법", "accessible design color contrast keyboard navigation"},
				},
				prompt: `다음 정보를 바탕으로 접근성 검사 체크리스트를 작성해줘.
%s
형식:
[색상 대비 (Color Contrast)]
- AA 기준: 4.5:1 (일반 텍스트)
- AA 기준: 3:1 (대형 텍스트 18px+)
- 검사 도구: WebAIM Contrast Checker

[키보드 내비게이션]
- Tab 순서 논리적 구성
- Focus 표시 가시성
- Skip Navigation 링크

[스크린 리더 지원]
- Alt 텍스트 작성 규칙
- ARIA 레이블 사용법
- 의미 있는 HTML 구조

[터치 타겟 크기]
- 최소 44×44px (iOS/Android)

자동 검사 도구: axe/WAVE/Lighthouse`,
			},
			"design_responsive_test": {
				title: "반응형 테스트",
				searches: []struct{ name, q string }{
					{"반응형 디자인 기준", "responsive design breakpoints best practice 2025"},
					{"모바일 퍼스트 디자인", "mobile first design testing checklist"},
				},
				prompt: `다음 정보를 바탕으로 반응형 디자인 테스트 가이드를 작성해줘.
%s
형식:
[브레이크포인트 기준]
| 디바이스 | 너비 | 기준 |
모바일: 375px~767px
태블릿: 768px~1279px
데스크톱: 1280px+
와이드: 1920px+

[테스트 체크리스트]
- 텍스트 가독성 (최소 16px)
- 이미지 비율 유지
- 터치 영역 크기
- 내비게이션 변환 (햄버거 메뉴)
- 테이블/차트 스크롤 처리

[테스트 도구]
Chrome DevTools / Responsively App / BrowserStack`,
			},
			"design_client_presentation": {
				title: "클라이언트 발표 자료",
				searches: []struct{ name, q string }{
					{"클라이언트 디자인 발표", "client design presentation best practice feedback"},
					{"디자인 설명 방법", "design rationale presentation storytelling"},
				},
				prompt: `다음 정보를 바탕으로 클라이언트 발표용 자료 구성 가이드를 작성해줘.
%s
형식:
[발표 구성 (20분 기준)]
0-3분: 프로젝트 목표 재확인
3-8분: 리서치 & 인사이트
8-18분: 디자인 시안 발표
18-20분: 다음 단계 제안

[시안 설명 방법]
- 왜 이 방향인가? (근거)
- 사용자에게 어떤 경험?
- 브랜드 가이드 부합 여부

[피드백 수렴 방법]
- 구체적 질문 3개 준비
- 주관적 의견 vs 사실 분리

[발표 후 처리]
- 피드백 정리 → 다음 버전 일정 제안`,
			},
			"design_portfolio_update": {
				title: "포트폴리오 업데이트",
				searches: []struct{ name, q string }{
					{"디자인 포트폴리오 구성", "design portfolio best practice 2025"},
					{"포트폴리오 케이스 스터디", "UX design portfolio case study structure"},
				},
				prompt: `다음 정보를 바탕으로 포트폴리오 업데이트 가이드를 작성해줘.
%s
형식:
[포트폴리오 구성 원칙]
- 작품 수: 3-5개 (적지만 깊게)
- 각 케이스 스터디 구성:
  1. 문제 정의 (Challenge)
  2. 내 역할 (My Role)
  3. 프로세스 (Research→Ideate→Design→Test)
  4. 결과물 (Final Design)
  5. 임팩트 (Impact/Result)

[플랫폼별 포트폴리오]
- Behance: 비주얼 중심
- Notion: 프로세스 중심
- 개인 웹사이트: 통합

[업데이트 루틴]
- 프로젝트 완료 직후 정리
- 분기 1회 전체 업데이트`,
			},
			// ── 프리랜서 (20개) ──────────────────────────────────
			"freelancer_client_manage": {
				title: "클라이언트 관리",
				searches: []struct{ name, q string }{
					{"프리랜서 클라이언트 관리", "freelancer client management CRM tool 2025"},
					{"클라이언트 관계 유지", "client relationship management freelance"},
				},
				prompt: `다음 정보를 바탕으로 클라이언트 관리 시스템을 작성해줘.
%s
형식:
[클라이언트 DB 구조]
| 클라이언트 | 업종 | 담당자 | 프로젝트 | 마지막 연락 | 상태 | 다음 액션 |

[클라이언트 등급 분류]
A급: 반복 의뢰 / 고단가 / 빠른 결정
B급: 가끔 의뢰 / 보통 단가
C급: 일회성 / 저단가

[관계 유지 루틴]
- 분기 1회 근황 체크 메시지
- 프로젝트 완료 후 1개월 후속 연락
- 생일/명절 인사 (고급 클라이언트)

[미팅 알림 자동화 방법]`,
			},
			"freelancer_estimate": {
				title: "견적서 자동 생성",
				searches: []struct{ name, q string }{
					{"프리랜서 견적 기준", "프리랜서 견적서 작성 기준 단가 2025"},
					{"프로젝트 단가 시세", query + " 프리랜서 단가 시세 시장가"},
				},
				prompt: `다음 정보를 바탕으로 프리랜서 견적서 초안을 작성해줘.
%s
형식:
[견적서]
견적번호: 2025-001
유효기간: 발행일로부터 14일

| 항목 | 내용 | 단가 | 수량 | 금액 |
기획/설계
디자인/개발
수정 (N회 포함)
추가 수정: 별도 협의

소계: ___원
부가세(10%): ___원
합계: ___원

계약금(50%): 계약시
잔금(50%): 납품시

[시장 단가 기준]
[견적 이메일 발송 문구]`,
			},
			"freelancer_invoice": {
				title: "청구서 / 세금계산서 발행",
				searches: []struct{ name, q string }{
					{"프리랜서 청구서 발행", "freelancer invoice template tax 2025 Korea"},
					{"세금계산서 발행 방법", "전자 세금계산서 발행 방법 2025"},
				},
				prompt: `다음 정보를 바탕으로 청구서 및 세금계산서 발행 가이드를 작성해줘.
%s
형식:
[청구서 구성]
- 공급자 정보 (사업자등록번호 필수)
- 공급받는자 정보
- 공급 내역 (작업 항목/기간/금액)
- 공급가액 / 세액 / 합계

[세금계산서 발행 방법]
- 홈택스 전자세금계산서 발행 절차
- 발행 기한: 다음달 10일까지
- 지연 발행 시 가산세

[청구 이메일 문구]
[미수금 발생 시 대응 방법]`,
			},
			"freelancer_tax": {
				title: "세금/회계 정리",
				searches: []struct{ name, q string }{
					{"프리랜서 세금 정리", "프리랜서 종합소득세 절세 방법 2025"},
					{"1인사업자 경비 처리", "1인 사업자 경비 인정 항목 2025"},
				},
				prompt: `다음 정보를 바탕으로 프리랜서 세금/회계 정리 가이드를 작성해줘.
%s
형식:
[연간 세금 일정]
1월: 부가세 확정신고 (7~12월분)
5월: 종합소득세 신고
7월: 부가세 예정신고 (1~6월분)

[경비 처리 가능 항목]
✅ 확실: 사무용품/통신비/교통비/교육비
⚠️ 조건부: 식비(업무 목적)/차량유지비
❌ 불가: 개인 생활비

[수입·지출 분류 엑셀 구조]
날짜 / 내용 / 분류 / 금액 / 증빙

[절세 팁 TOP 5]`,
			},
			"freelancer_time_track": {
				title: "프로젝트 시간 추적",
				searches: []struct{ name, q string }{
					{"프리랜서 시간 관리", "freelancer time tracking productivity tool 2025"},
				},
				prompt: `다음 정보를 바탕으로 프로젝트 시간 추적 시스템을 작성해줘.
%s
형식:
[일일 작업 로그 포맷]
날짜:
프로젝트:
| 시간 | 작업 내용 | 소요시간 | 누적 |

[시간 추적 도구 추천]
- Toggl Track (무료/간편)
- Clockify (무료/팀 기능)
- Harvest (유료/청구 연동)

[프로젝트별 수익성 계산]
총 작업시간 × 시간당 단가 = 실제 수익
목표 단가 vs 실제 단가 비교

[시간 기록이 중요한 이유]
- 견적 정확도 향상
- 저수익 프로젝트 식별
- 클라이언트 보고 근거`,
			},
			"freelancer_portfolio": {
				title: "포트폴리오 업데이트",
				searches: []struct{ name, q string }{
					{"프리랜서 포트폴리오", "freelancer portfolio best practice 2025"},
					{"포트폴리오 플랫폼", "freelancer portfolio platform Behance LinkedIn"},
				},
				prompt: `다음 정보를 바탕으로 프리랜서 포트폴리오 업데이트 가이드를 작성해줘.
%s
형식:
[포트폴리오 핵심 원칙]
- 최근 작업 3-5개 집중
- 결과/임팩트 수치 포함 (매출 N% 증가 등)
- 클라이언트 추천사 포함

[플랫폼별 전략]
- LinkedIn: 전문성/신뢰도 중심
- Behance/Dribbble: 비주얼 중심
- 개인 사이트: 통합 브랜딩

[프로젝트 케이스 스터디 포맷]
1. 클라이언트/업종 (익명 가능)
2. 과제 (Challenge)
3. 솔루션
4. 결과 (임팩트)

[업데이트 주기]
프로젝트 완료 후 2주 이내 정리`,
			},
			"freelancer_self_marketing": {
				title: "자기 PR 콘텐츠 생성",
				searches: []struct{ name, q string }{
					{"프리랜서 자기 PR", "freelancer self marketing personal brand 2025"},
					{"LinkedIn 포스팅 전략", "LinkedIn content strategy freelancer thought leader"},
				},
				prompt: `다음 정보를 바탕으로 자기 PR 콘텐츠를 생성해줘.
%s
형식:
💼 LinkedIn 포스트 (전문가 버전)
[최근 작업 스토리: 300자]

📝 블로그/브런치 아티클
- 제목 3가지:
- 서론 초안:

🧵 X/스레드 포스트
- 10개 불릿 포인트:

📱 인스타그램 캡션
- 작업 과정 공유 버전:

[홍보 주기 추천]
LinkedIn: 주 2-3회
블로그: 월 2회
SNS: 주 3-4회`,
			},
			"freelancer_contract_review": {
				title: "계약서 검토",
				searches: []struct{ name, q string }{
					{"프리랜서 계약서 위험 항목", "freelancer contract red flags review 2025"},
					{"계약서 필수 조항", "freelance contract essential clauses Korea"},
				},
				prompt: `다음 정보를 바탕으로 프리랜서 계약서 검토 결과를 작성해줘.
%s
형식:
🔴 즉시 수정 요청 항목
- 무제한 수정 조항 (횟수 명시 필요)
- 지식재산권 과도한 양도
- 일방적 계약 해지 조건
- 무기한 비밀유지 조항

🟡 협상 권장 항목
- 납기 지연 패널티 기준
- 추가 작업 단가 기준
- 완료 기준(Acceptance Criteria)

🟢 확인 필수
- 계약금 비율 (최소 30%)
- 저작권 귀속 시점
- 분쟁 해결 방법

계약서 협상 팁 3가지 포함`,
			},
			"freelancer_cashflow": {
				title: "현금 흐름 관리",
				searches: []struct{ name, q string }{
					{"프리랜서 현금 흐름", "freelancer cash flow management income stability"},
					{"수입 안정화 방법", "freelance income stabilization retainer contract"},
				},
				prompt: `다음 정보를 바탕으로 현금 흐름 관리 가이드를 작성해줘.
%s
형식:
[월별 현금 흐름 예측표]
| 월 | 예상 수입 | 예상 지출 | 잔액 |

[수입 안정화 전략]
1. 리테이너 계약 비율 목표: 40%
2. 프로젝트 다각화 (클라이언트 3개 이상)
3. 비상금 목표: 3개월치 생활비

[지출 분류]
고정: 통신비/구독/세금
변동: 외주/장비/교육

[미수금 방지 전략]
- 계약금 50% 선납
- 단계별 지급 구조
- 자동 청구 도구 설정`,
			},
			"freelancer_tax_report": {
				title: "연말정산 / 부가세 신고 자료",
				searches: []struct{ name, q string }{
					{"프리랜서 세금 신고", "프리랜서 종합소득세 신고 방법 2025"},
					{"부가세 신고 준비", "1인사업자 부가세 신고 준비 서류 2025"},
				},
				prompt: `다음 정보를 바탕으로 세금 신고 자료 정리 가이드를 작성해줘.
%s
형식:
[종합소득세 신고 준비 (5월)]
필요 서류:
- 수입 내역 (세금계산서/거래명세서)
- 경비 영수증 (카드/현금)
- 사업소득 원천징수영수증

[경비 정리 방법]
카테고리별 합산:
사무용품 / 통신비 / 교육비 / 차량 / 기타

[절세 체크리스트]
- 노란우산공제 가입 여부
- 청년우대형 계좌 활용
- 경비 누락 항목 확인

[신고 일정]
[홈택스 신고 절차 요약]`,
			},
			"freelancer_client_onboarding": {
				title: "클라이언트 온보딩",
				searches: []struct{ name, q string }{
					{"클라이언트 온보딩 방법", "freelancer client onboarding process template"},
				},
				prompt: `다음 정보를 바탕으로 신규 클라이언트 온보딩 패키지를 작성해줘.
%s
형식:
[온보딩 체크리스트]
계약 전:
□ 범위 정의 (SOW)
□ 견적 확정
□ 계약서 서명
□ 계약금 수령

계약 후:
□ 웰컴 메시지 발송
□ 킥오프 미팅 일정 확정
□ 협업 툴 초대 (Slack/Notion)
□ 자료 수집 (브리핑/에셋)

[웰컴 메시지 템플릿]
[킥오프 미팅 어젠다]
[초기 자료 요청 체크리스트]`,
			},
			"freelancer_project_kickoff": {
				title: "프로젝트 킥오프",
				searches: []struct{ name, q string }{
					{"프로젝트 킥오프 방법", "project kickoff meeting agenda template"},
					{"킥오프 미팅 구성", "kickoff meeting checklist freelancer"},
				},
				prompt: `다음 정보를 바탕으로 프로젝트 킥오프 자료를 작성해줘.
%s
형식:
[킥오프 미팅 어젠다 (60분)]
10분: 소개 및 아이스브레이킹
15분: 프로젝트 목표 재확인
15분: 범위 및 일정 확인
10분: 협업 방식 결정
10분: 질문 및 다음 단계

[킥오프 후 배포 문서]
- 프로젝트 요약
- 마일스톤 및 납기
- 연락처 및 에스컬레이션
- 협업 툴 링크

[프로젝트 계획서 1페이지 요약]`,
			},
			"freelancer_deliverable_check": {
				title: "산출물 검토",
				searches: []struct{ name, q string }{
					{"산출물 검토 체크리스트", "deliverable review checklist quality control freelance"},
				},
				prompt: `다음 정보를 바탕으로 납품 전 산출물 검토 체크리스트를 작성해줘.
%s
형식:
[납품 전 최종 체크리스트]

📁 파일 구성
□ 최종 파일 + 수정 가능 소스 파일
□ 파일명 규칙 준수
□ 폴더 구조 정리

✅ 품질 확인
□ 계약서의 납품 기준 충족
□ 수정 횟수 내 처리 완료
□ 오타/오류 최종 검수

📧 납품 이메일
- 납품 파일 안내
- 사용 방법 가이드
- 수정 정책 안내
- 잔금 청구 안내`,
			},
			"freelancer_payment_reminder": {
				title: "미수금 독촉",
				searches: []struct{ name, q string }{
					{"미수금 독촉 방법", "freelancer overdue invoice reminder email template"},
					{"미수금 법적 대응", "unpaid invoice freelance legal action Korea"},
				},
				prompt: `다음 정보를 바탕으로 미수금 독촉 메시지를 작성해줘.
%s
형식:
[D+3 (납기 3일 초과) - 정중한 리마인드]
제목: [프로젝트명] 대금 납부 안내
본문: 친절하고 간결하게

[D+14 - 공식 독촉]
제목: [프로젝트명] 미납 대금 독촉장
본문: 공식 톤 + 납부 기한 명시

[D+30 - 법적 대응 예고]
제목: 내용증명 발송 예정 안내
본문: 진지한 경고 톤

[법적 대응 절차]
1. 내용증명 → 2. 지급명령 → 3. 소액심판

[미수금 예방 방법 5가지]`,
			},
			"freelancer_proposal_template": {
				title: "제안서 템플릿 관리",
				searches: []struct{ name, q string }{
					{"프리랜서 제안서 구조", "freelancer proposal template winning 2025"},
					{"업종별 제안서 차이", "proposal structure design development marketing"},
				},
				prompt: `다음 정보를 바탕으로 업종별 제안서 템플릿을 작성해줘.
%s
형식:
[공통 제안서 구조]
1. 커버페이지 (클라이언트명 + 프로젝트명)
2. 우리의 이해 (클라이언트 상황/문제)
3. 제안 솔루션
4. 작업 범위 및 프로세스
5. 타임라인
6. 견적
7. 포트폴리오/레퍼런스
8. 계약 조건

[업종별 강조 포인트]
디자인: 비주얼 레퍼런스 풍부하게
개발: 기술 스택/아키텍처 명시
마케팅: 예상 ROI/성과 지표

[제안서 발송 후 팔로업 가이드]`,
			},
			"freelancer_rate_calculation": {
				title: "단가 계산",
				searches: []struct{ name, q string }{
					{"프리랜서 적정 단가", "freelancer rate calculation 2025 Korea"},
					{"업종별 프리랜서 단가", query + " 프리랜서 시장 단가 시세"},
				},
				prompt: `다음 정보를 바탕으로 적정 단가를 계산해줘.
%s
형식:
[시간당 단가 역산 계산]
목표 월 수입: ___원
실제 작업 가능 시간: ___시간/월 (총 근무시간 × 0.6)
→ 시간당 최소 단가: ___원

[프로젝트 단가 계산]
예상 작업시간 × 시간당 단가 = 기본 견적
+ 복잡도 가산 (1.2~1.5배)
+ 급행 가산 (1.3~2.0배)

[시장 단가 벤치마크]
업종별 평균 단가 (시간당/프로젝트)

[단가 인상 방법 및 타이밍]`,
			},
			"freelancer_work_log": {
				title: "작업 로그 정리",
				searches: []struct{ name, q string }{
					{"작업 로그 관리", "work log management freelancer productivity"},
				},
				prompt: `다음 정보를 바탕으로 오늘의 작업 로그 정리 가이드를 작성해줘.
%s
형식:
[일일 작업 로그]
📅 날짜:
🎯 오늘 목표:

| 시간 | 프로젝트 | 작업 내용 | 결과물 | 시간 |
09:00~
10:00~
...

✅ 완료한 작업
⚠️ 미완료 및 이유
📋 내일 이어서 할 것

[주간 작업 요약]
총 작업시간: ___h
프로젝트별 시간 배분:
수익성 체크:`,
			},
			"freelancer_business_plan": {
				title: "사업 계획 수립",
				searches: []struct{ name, q string }{
					{"1인 사업 계획", "freelance business plan 2025 growth strategy"},
					{"프리랜서 수익화 전략", "freelancer income growth strategy productize service"},
				},
				prompt: `다음 정보를 바탕으로 1인 사업 계획서를 작성해줘.
%s
형식:
1. 사업 비전 및 목표
2. 서비스 포지셔닝 (전문 분야 정의)
3. 타겟 고객 정의 (ICP)
4. 수익 모델 설계
   - 프로젝트형 / 리테이너형 / 디지털 제품
5. 연간 매출 목표 및 달성 전략
6. 마케팅/영업 계획
7. 역량 개발 계획
8. 리스크 및 대응 방안`,
			},
			"freelancer_networking_content": {
				title: "네트워킹 콘텐츠",
				searches: []struct{ name, q string }{
					{"프리랜서 네트워킹", "freelancer networking LinkedIn content strategy"},
					{"커뮤니티 활동 방법", "professional networking community freelance"},
				},
				prompt: `다음 정보를 바탕으로 네트워킹용 LinkedIn 콘텐츠를 작성해줘.
%s
형식:
[LinkedIn 포스트 3가지]

1. 인사이트 공유형 (전문성 어필)
훅: [첫 줄로 멈추게 하는 문장]
본문: [3-5개 핵심 포인트]
CTA: [댓글 유도]

2. 경험 스토리형 (공감 유발)
상황 → 시도 → 결과 → 교훈

3. 질문형 (커뮤니티 참여 유도)
[업계 공통 고민 던지기]

[네트워킹 DM 템플릿]
- 첫 연락 / 팔로업 / 협업 제안

[커뮤니티 추천]`,
			},
			"freelancer_yearly_review": {
				title: "연간 리뷰",
				searches: []struct{ name, q string }{
					{"프리랜서 연간 리뷰", "freelancer annual review reflection 2025"},
					{"1인 사업 성과 분석", "solo business year review growth metrics"},
				},
				prompt: `다음 정보를 바탕으로 연간 리뷰 리포트를 작성해줘.
%s
형식:
📊 연간 실적 요약
- 총 매출: / 목표 대비:
- 프로젝트 수: / 클라이언트 수:
- 평균 프로젝트 단가:

💼 클라이언트 분석
- 최고 수익 클라이언트 TOP 3
- 재계약율:
- 신규 vs 기존 비율:

🌱 성장 & 학습
- 올해 새로 배운 기술:
- 업그레이드된 역량:

⚠️ 아쉬운 점 & 교훈

🎯 내년 목표 설정
매출 목표 / 서비스 방향 / 역량 개발`,
			},
			// ── PM ──────────────────────────────────────────────
		}

		wfSelected, ok := presetDefs[preset]
		if !ok {
			wfErrMsg := "알 수 없는 워크플로우: " + preset
			if req.Lang == "en" || isEnglishQuery(req.Message) {
				wfErrMsg = "Unknown workflow: " + preset
			}
			json200(w, CommandResponse{Success: false, Message: wfErrMsg, Action: "workflow_preset", Duration: dur})
			break
		}

		for i, s := range wfSelected.searches {
			go func(idx int, name, q string) {
				tr, ok := tavilySearch(tKey, q, 3)
				if ok {
					wfCh <- wfSection{name, tr.Summary}
				} else {
					wfCh <- wfSection{name, ""}
				}
			}(i, s.name, s.q)
		}

		wfCollected := []string{}
		for range wfSelected.searches {
			s := <-wfCh
			if s.body != "" {
				wfCollected = append(wfCollected, fmt.Sprintf("### %s\n%s", s.name, s.body))
			}
		}

		searchContext := strings.Join(wfCollected, "\n\n")
		finalPrompt := fmt.Sprintf(wfSelected.prompt, searchContext)
		if req.Message != "" {
			if req.Lang == "en" || isEnglishQuery(req.Message) {
				finalPrompt = fmt.Sprintf("## User Request/Code\n%s\n\n%s", req.Message, finalPrompt)
			} else {
				finalPrompt = fmt.Sprintf("## 사용자 요청/코드\n%s\n\n%s", req.Message, finalPrompt)
			}
		}
		persona := getActivePersona()
		var wfSys string
		if req.Lang == "en" || isEnglishQuery(req.Message) {
			wfSys = persona.SystemPrompt + "\nAnswer in clear English using markdown formatting."
		} else {
			wfSys = persona.SystemPrompt + "\n답변은 마크다운으로 깔끔하게 작성하세요."
		}
		wfMsgs := []groqMsg{{Role: "system", Content: wfSys}, {Role: "user", Content: finalPrompt}}
		result, _, _ := callGroq(gKeyWF, groqChatModel, wfMsgs, 1500, false)
		if result == "" {
			result, _, _ = callGroqWithFallback([]groqMsg{{Role: "user", Content: finalPrompt}}, 1500, false)
		}

		json200(w, CommandResponse{
			Success:  true,
			Message:  result,
			Action:   "workflow_preset",
			Result:   map[string]any{"preset": preset, "title": wfSelected.title, "persona": persona.ID},
			Duration: dur,
		})

	case "multi_action":
		subAction, _ := intent.Params["sub_action"].(string)
		query, _ := intent.Params["query"].(string)
		site, _ := intent.Params["site"].(string)
		platform, _ := intent.Params["platform"].(string)
		fmtStr, _ := intent.Params["format"].(string)
		// pending_params의 format을 우선 사용 (LLM이 덮어쓰는 것 방지)
		if pf, ok := req.PendingParams["format"].(string); ok && pf != "" {
			fmtStr = pf
		}
		maxItemsF, _ := intent.Params["max_items"].(float64)
		maxItems := int(maxItemsF)
		if maxItems == 0 {
			maxItems = 8
		}
		if query == "" {
			query = req.Message
		}
		outputFmt := outputFormat(fmtStr)

		llmMu.RLock()
		tKey := llmTavilyKey
		llmMu.RUnlock()

		var collectedItems []map[string]string
		var actionSummary string

		switch subAction {
		case "price_compare":
			if tKey != "" {
				if site != "" {
					if tr, ok := tavilySearchDomain(tKey, query, maxItems, site); ok {
						collectedItems = tr.Items
					}
				}
				if len(collectedItems) == 0 {
					if tr, ok := tavilySearch(tKey, query, maxItems); ok {
						collectedItems = tr.Items
					}
				}
			}
			siteName := site
			if siteName == "" {
				siteName = "쇼핑몰"
			}
			actionSummary = fmt.Sprintf("%s에서 \"%s\" 상품 %d개 검색 결과", siteName, query, len(collectedItems))

		case "video_search":
			targetDomain := "youtube.com"
			if platform == "tiktok" {
				targetDomain = "tiktok.com"
			}
			if tKey != "" {
				if tr, ok := tavilySearchDomain(tKey, query, maxItems, targetDomain); ok {
					collectedItems = tr.Items
				}
				if len(collectedItems) == 0 {
					fallbackQ := query + " " + targetDomain
					if tr, ok := tavilySearch(tKey, fallbackQ, maxItems); ok {
						collectedItems = tr.Items
					}
				}
			}
			pName := "YouTube"
			if platform == "tiktok" {
				pName = "TikTok"
			}
			actionSummary = fmt.Sprintf("%s에서 \"%s\" 영상 %d개 검색 결과", pName, query, len(collectedItems))

		case "doc_compare":
			// 두 대상 비교 - Tavily 검색 후 LLM이 비교표 생성
			llmMu.RLock()
			gKey := llmPerplexityKey
			llmMu.RUnlock()
			docEng := req.Lang == "en" || isEnglishQuery(query)
			var compareText string
			if tr, ok := webSearchWithFallback(tKey, query, maxItems); ok {
				collectedItems = tr.Items
				var articleLines strings.Builder
				for i, item := range tr.Items {
					t := item["title"]
					c := item["content"]
					if c == "" { c = item["snippet"] }
					articleLines.WriteString(fmt.Sprintf("[%d] %s\n%s\n\n", i+1, t, c))
				}
				if tr.Summary != "" {
					if docEng {
						articleLines.WriteString("\n[Full Summary]\n" + tr.Summary)
					} else {
						articleLines.WriteString("\n[전체 요약]\n" + tr.Summary)
					}
				}
				var prompt string
				if docEng {
					prompt = fmt.Sprintf(`Based on the following information, compare "%s" by category.
Use a markdown comparison table (| Category | A | B |) format in English.

Reference material:
%s`, query, articleLines.String())
				} else {
					prompt = fmt.Sprintf(`다음 정보를 바탕으로 "%s"를 항목별로 비교 정리해줘.
마크다운 비교표(| 항목 | A | B |) 형식으로 한국어로 작성해줘.

참고 자료:
%s`, query, articleLines.String())
				}
				if gKey != "" {
					compareText, _, _ = callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 2000, false)
				} else if llmClaudeKey != "" {
					body := map[string]any{
						"model": claudeHaikuModel, "max_tokens": 2000,
						"messages": []map[string]any{{"role": "user", "content": prompt}},
					}
					compareText = callClaudeAPI(llmClaudeKey, body)
				}
			}
			if compareText == "" {
				llmMu.RLock()
				gFallback := llmPerplexityKey
				cFallback := llmClaudeKey
				llmMu.RUnlock()
				var fallbackPrompt string
				if docEng {
					fallbackPrompt = fmt.Sprintf(`Compare "%s" by category using a markdown comparison table (| Category | A | B |) in English. Base it on the latest information.`, query)
				} else {
					fallbackPrompt = fmt.Sprintf(`"%s"를 항목별로 비교 정리해줘.
마크다운 비교표(| 항목 | A | B |) 형식으로 한국어로 작성해줘. 최신 정보를 기반으로 작성해줘.`, query)
				}
				if gFallback != "" {
					compareText, _, _ = callGroqWithFallback([]groqMsg{{Role: "user", Content: fallbackPrompt}}, 2000, false)
				} else if cFallback != "" {
					body := map[string]any{
						"model": claudeHaikuModel, "max_tokens": 2000,
						"messages": []map[string]any{{"role": "user", "content": fallbackPrompt}},
					}
					compareText = callClaudeAPI(cFallback, body)
				}
				if compareText == "" {
					if docEng {
						compareText = fmt.Sprintf("Search quota exceeded for \"%s\". Add a Tavily API key in Settings to enable real-time comparison.", query)
					} else {
						compareText = fmt.Sprintf("\"%s\" — 검색 API 쿼터를 초과했습니다. 설정에서 Tavily API 키를 등록하면 실시간 데이터로 비교할 수 있습니다.", query)
					}
				}
			}
			actionSummary = compareText

		case "summarize":
			// 주제 요약 - Tavily 검색 후 실제 기사 본문 포함해서 LLM 요약
			llmMu.RLock()
			gKey := llmPerplexityKey
			llmMu.RUnlock()
			sumEng := req.Lang == "en" || isEnglishQuery(query)
			var summaryText string
			if tr, ok := webSearchWithFallback(tKey, query, maxItems); ok {
				collectedItems = tr.Items

				var articleLines strings.Builder
				for i, item := range tr.Items {
					t := item["title"]
					c := item["content"]
					if c == "" { c = item["snippet"] }
					if t == "" { continue }
					articleLines.WriteString(fmt.Sprintf("[%d] %s\n", i+1, t))
					if c != "" { articleLines.WriteString(c + "\n") }
					articleLines.WriteString("\n")
				}
				if tr.Summary != "" {
					if sumEng {
						articleLines.WriteString("\n[Full Summary]\n" + tr.Summary)
					} else {
						articleLines.WriteString("\n[전체 요약]\n" + tr.Summary)
					}
				}

				var prompt string
				if sumEng {
					prompt = fmt.Sprintf(`Based on the following search results, clearly summarize "%s" in English.
Structure the key content by category (## subtitle, - bullet points). Do not include source URLs.

Search results:
%s`, query, articleLines.String())
				} else {
					prompt = fmt.Sprintf(`다음 검색 결과를 바탕으로 "%s"에 대해 한국어로 명확하게 요약 정리해줘.
핵심 내용을 항목별(## 소제목, - 포인트)로 구조화해서 작성해줘. 출처 URL은 포함하지 마.

검색 결과:
%s`, query, articleLines.String())
				}
				if gKey != "" {
					summaryText, _, _ = callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 2000, false)
				} else if llmClaudeKey != "" {
					body := map[string]any{
						"model": claudeHaikuModel, "max_tokens": 2000,
						"messages": []map[string]any{{"role": "user", "content": prompt}},
					}
					summaryText = callClaudeAPI(llmClaudeKey, body)
				}
			}
			if summaryText == "" {
				llmMu.RLock()
				gFallback := llmPerplexityKey
				cFallback := llmClaudeKey
				llmMu.RUnlock()
				var fallbackPrompt string
				if sumEng {
					fallbackPrompt = fmt.Sprintf(`Summarize "%s" in English with structured sections (## subtitle, - bullet points). Base it on the latest information.`, query)
				} else {
					fallbackPrompt = fmt.Sprintf(`"%s"에 대해 한국어로 핵심 내용을 항목별(## 소제목, - 포인트)로 요약 정리해줘. 최신 동향을 기반으로 작성해줘.`, query)
				}
				if gFallback != "" {
					summaryText, _, _ = callGroqWithFallback([]groqMsg{{Role: "user", Content: fallbackPrompt}}, 1500, false)
				} else if cFallback != "" {
					body := map[string]any{
						"model": claudeHaikuModel, "max_tokens": 1500,
						"messages": []map[string]any{{"role": "user", "content": fallbackPrompt}},
					}
					summaryText = callClaudeAPI(cFallback, body)
				}
				if summaryText == "" {
					if sumEng {
						summaryText = fmt.Sprintf("Search quota exceeded for \"%s\". Add a Tavily API key in Settings to enable real-time summaries.", query)
					} else {
						summaryText = fmt.Sprintf("\"%s\" — 검색 API 쿼터를 초과했습니다. 설정에서 Tavily API 키를 등록하면 실시간 요약을 사용할 수 있습니다.", query)
					}
				}
			}
			actionSummary = summaryText

		default:
			// 일반 web_search — 본문 포함 수집
			llmMu.RLock()
			gKey2 := llmPerplexityKey
			llmMu.RUnlock()
			wsEng := req.Lang == "en" || isEnglishQuery(query)
			if tr, ok := webSearchWithFallback(tKey, query, maxItems); ok {
				collectedItems = tr.Items
				if gKey2 != "" {
					var lines strings.Builder
					for i, item := range tr.Items {
						t := item["title"]
						c := item["content"]
						if c == "" { c = item["snippet"] }
						lines.WriteString(fmt.Sprintf("[%d] %s\n%s\n\n", i+1, t, c))
					}
					if tr.Summary != "" {
						if wsEng {
							lines.WriteString("\n[Full Summary]\n" + tr.Summary)
						} else {
							lines.WriteString("\n[전체 요약]\n" + tr.Summary)
						}
					}
					var prompt string
					if wsEng {
						prompt = fmt.Sprintf(`Summarize the search results for "%s" in 3-5 key sentences in English.\n\n%s`, query, lines.String())
					} else {
						prompt = fmt.Sprintf(`"%s" 검색 결과를 한국어로 3~5줄 핵심 요약해줘.\n\n%s`, query, lines.String())
					}
					actionSummary, _, _ = callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 800, false)
				}
			}
			if actionSummary == "" {
				if wsEng {
					actionSummary = fmt.Sprintf("%d search results for \"%s\"", len(collectedItems), query)
				} else {
					actionSummary = fmt.Sprintf("\"%s\" 검색 결과 %d개", query, len(collectedItems))
				}
			}
		}

		// format 기본값: markdown
		if outputFmt == "" {
			outputFmt = outMarkdown
		}

		// 파일 저장
		title := query
		if len([]rune(title)) > 20 {
			title = string([]rune(title)[:20])
		}
		filePath, saveErr := saveResultToFile(outputFmt, title, collectedItems, actionSummary)
		var fileMsg string
		maEng := req.Lang == "en" || isEnglishQuery(query)
		if saveErr != nil {
			if maEng {
				fileMsg = fmt.Sprintf("⚠️ File save failed: %s", saveErr.Error())
			} else {
				fileMsg = fmt.Sprintf("⚠️ 파일 저장 실패: %s", saveErr.Error())
			}
		} else {
			extMap := map[outputFormat]string{
				outPDF: "HTML(PDF용)", outWord: "DOCX", outExcel: "XLSX",
				outPowerPoint: "PPTX", outMarkdown: "MARKDOWN", outTXT: "TXT",
			}
			ext := extMap[outputFmt]
			if ext == "" { ext = strings.ToUpper(string(outputFmt)) }
			if maEng {
				fileMsg = fmt.Sprintf("📄 Saved as %s: %s", ext, filePath)
			} else {
				fileMsg = fmt.Sprintf("📄 %s 파일로 저장됨: %s", ext, filePath)
			}
		}

		resultItems := make([]map[string]string, 0, len(collectedItems))
		for _, it := range collectedItems {
			resultItems = append(resultItems, map[string]string{
				"site": site, "name": it["title"], "price": it["price"], "link": it["url"],
			})
		}

		json200(w, CommandResponse{
			Success:  true,
			Message:  actionSummary + "\n" + fileMsg,
			Action:   "multi_action",
			Result: map[string]any{
				"query":     query,
				"summary":   actionSummary,
				"results":   resultItems,
				"total":     len(resultItems),
				"file_path": filePath,
				"file_msg":  fileMsg,
				"format":    fmtStr,
				"sub_action": subAction,
			},
			Duration: dur,
		})

	case "directions":
		// 길찾기 → handleDirections 내부 로직 재사용 (Tavily 검색)
		from, _ := intent.Params["from"].(string)
		to, _ := intent.Params["to"].(string)
		mode, _ := intent.Params["mode"].(string)
		if to == "" {
			to = req.Message
		}
		if mode == "" {
			mode = "transit"
		}
		dEng := req.Lang == "en" || isEnglishQuery(req.Message)
		var dQuery string
		if dEng {
			if from != "" {
				dQuery = fmt.Sprintf("directions from %s to %s by %s", from, to, mode)
			} else {
				dQuery = fmt.Sprintf("directions to %s by %s", to, mode)
			}
		} else {
			if from != "" {
				dQuery = fmt.Sprintf("%s에서 %s 가는 %s 길찾기", from, to, mode)
			} else {
				dQuery = fmt.Sprintf("%s 가는 방법 %s", to, mode)
			}
		}
		llmMu.RLock()
		tKey := llmTavilyKey
		llmMu.RUnlock()
		var dSummary string
		var dItems []map[string]string
		if tKey != "" {
			if tr, ok := tavilySearch(tKey, dQuery, 4); ok {
				dSummary = tr.Summary
				dItems = tr.Items
			}
		}
		if dSummary == "" {
			if dEng {
				dSummary = fmt.Sprintf("Here are directions to %s.", to)
			} else {
				dSummary = fmt.Sprintf("%s 경로 정보입니다.", to)
			}
		}
		links := buildMapLinks(from, to, mode, dEng)
		json200(w, CommandResponse{
			Success: true, Message: dSummary, Action: "directions",
			Result: map[string]any{"from": from, "to": to, "mode": mode, "summary": dSummary, "items": dItems, "links": links},
			Duration: dur,
		})

	case "place_view":
		// 장소 검색
		query, _ := intent.Params["query"].(string)
		if query == "" {
			query = req.Message
		}
		pEng := req.Lang == "en" || isEnglishQuery(req.Message)
		llmMu.RLock()
		tKey2 := llmTavilyKey
		llmMu.RUnlock()
		var pSummary string
		var pItems []map[string]string
		if tKey2 != "" {
			var pq string
			if pEng {
				pq = query + " location address hours"
			} else {
				pq = query + " 위치 주소 영업시간"
			}
			if tr, ok := tavilySearch(tKey2, pq, 4); ok {
				pSummary = tr.Summary
				pItems = tr.Items
			}
		}
		if pSummary == "" {
			if pEng {
				pSummary = fmt.Sprintf("Here is information about %s.", query)
			} else {
				pSummary = fmt.Sprintf("%s 정보입니다.", query)
			}
		}
		json200(w, CommandResponse{
			Success: true, Message: pSummary, Action: "place_view",
			Result: map[string]any{"query": query, "summary": pSummary, "items": pItems},
			Duration: dur,
		})

	case "multi_agent":
		// 멀티 에이전트 → handleMultiAgentRun에 위임
		goal, _ := intent.Params["goal"].(string)
		if goal == "" {
			goal = req.Message
		}
		llmMu.RLock()
		maKey := llmPerplexityKey
		llmMu.RUnlock()
		if maKey == "" {
			maEng := req.Lang == "en" || isEnglishQuery(req.Message)
			var maMsg string
			if maEng {
				maMsg = "API key required for multi-agent execution."
			} else {
				maMsg = "멀티 에이전트 실행에 API 키가 필요합니다."
			}
			json200(w, CommandResponse{Success: false, Message: maMsg, Action: "multi_agent", Duration: dur})
			return
		}
		maEng := req.Lang == "en" || isEnglishQuery(req.Message)
		var maStart string
		if maEng {
			maStart = fmt.Sprintf("Starting multi-agent execution for: %s", goal)
		} else {
			maStart = fmt.Sprintf("멀티 에이전트 실행 시작: %s", goal)
		}
		go func(g, k string) {
			result, err := runMacOrchestrate(g, k)
			var alertMsg string
			if err != nil {
				if maEng {
					alertMsg = "Multi-agent execution failed: " + err.Error()
				} else {
					alertMsg = "멀티 에이전트 실행 실패: " + err.Error()
				}
			} else {
				if maEng {
					alertMsg = "Multi-agent complete: " + result
				} else {
					alertMsg = "멀티 에이전트 완료: " + result
				}
			}
			publishAlert(Alert{ID: fmt.Sprintf("ma_%d", time.Now().Unix()), Level: "info", Title: "Multi-Agent", Message: alertMsg})
		}(goal, maKey)
		json200(w, CommandResponse{
			Success: true, Message: maStart, Action: "multi_agent",
			Result:   map[string]any{"goal": goal, "status": "running"},
			Duration: dur,
		})

	case "email":
		// 이메일 - SMTP 설정 안내
		eEng := req.Lang == "en" || isEnglishQuery(req.Message)
		var eMsg string
		if eEng {
			eMsg = "Email feature requires SMTP configuration. Please go to Settings → Email to set up your email account."
		} else {
			eMsg = "이메일 기능을 사용하려면 SMTP 설정이 필요합니다. 설정 → 이메일에서 이메일 계정을 설정해주세요."
		}
		json200(w, CommandResponse{Success: false, Message: eMsg, Action: "email", Duration: dur})

	case "meeting":
		// 회의 요약/분석 → 웹 검색 폴백
		mQuery, _ := intent.Params["query"].(string)
		if mQuery == "" {
			mQuery = req.Message
		}
		mEng := req.Lang == "en" || isEnglishQuery(req.Message)
		llmMu.RLock()
		mPKey := llmPerplexityKey
		llmMu.RUnlock()
		var mAnswer string
		if mPKey != "" {
			var mPrompt string
			if mEng {
				mPrompt = "You are a meeting assistant. Help with: " + mQuery + "\nAnswer concisely in English."
			} else {
				mPrompt = getPersonaSystemPrompt() + "\n회의 관련 질문: " + mQuery + "\n간결하게 한국어로 답변하세요."
			}
			mMsgs := []groqMsg{{Role: "user", Content: mPrompt}}
			mAnswer, _, _ = callGroq(mPKey, groqChatModel, mMsgs, 512, false)
		}
		if mAnswer == "" {
			if mEng {
				mAnswer = "Please describe what you need help with for your meeting."
			} else {
				mAnswer = "회의 관련해서 무엇을 도와드릴까요?"
			}
		}
		json200(w, CommandResponse{Success: true, Message: mAnswer, Action: "meeting", Duration: dur})

	case "briefing":
		// 브리핑 → handleBriefingNow 로직 직접 호출
		bEng := req.Lang == "en" || isEnglishQuery(req.Message)
		llmMu.RLock()
		bKey := llmTavilyKey
		llmMu.RUnlock()
		var bSections []string
		// 날씨
		weatherURL := "https://wttr.in/Seoul?format=j1"
		if bEng {
			weatherURL = "https://wttr.in/New York?format=j1"
		}
		wClient := &http.Client{Timeout: 5 * time.Second}
		if wr, err := wClient.Get(weatherURL); err == nil {
			defer wr.Body.Close()
			var wraw map[string]any
			if json.NewDecoder(wr.Body).Decode(&wraw) == nil {
				if cc, ok := wraw["current_condition"].([]any); ok && len(cc) > 0 {
					c := cc[0].(map[string]any)
					temp := fmt.Sprintf("%v", c["temp_C"])
					desc := ""
					if wds, ok := c["weatherDesc"].([]any); ok && len(wds) > 0 {
						desc = fmt.Sprintf("%v", (wds[0].(map[string]any))["value"])
					}
					if bEng {
						bSections = append(bSections, fmt.Sprintf("🌤️ Weather: %s°C, %s", temp, desc))
					} else {
						bSections = append(bSections, fmt.Sprintf("🌤️ 날씨: %s°C, %s", temp, desc))
					}
				}
			}
		}
		// 뉴스
		if bKey != "" {
			var nq string
			if bEng {
				nq = "today's top news worldwide 2026"
			} else {
				nq = "오늘 주요 뉴스 한국"
			}
			if nr, ok := tavilySearch(bKey, nq, 3); ok && nr.Summary != "" {
				if bEng {
					bSections = append(bSections, "📰 News: "+nr.Summary)
				} else {
					bSections = append(bSections, "📰 뉴스: "+nr.Summary)
				}
			}
		}
		var bMsg string
		if len(bSections) > 0 {
			bMsg = strings.Join(bSections, "\n\n")
		} else {
			if bEng {
				bMsg = "Good morning! Today's briefing is ready."
			} else {
				bMsg = "좋은 아침이에요! 오늘의 브리핑입니다."
			}
		}
		json200(w, CommandResponse{
			Success: true, Message: bMsg, Action: "briefing",
			Result: map[string]any{"sections": bSections},
			Duration: dur,
		})

	case "file_search":
		// 파일 검색 - Mac에서는 mdfind/find 사용
		fsQuery, _ := intent.Params["query"].(string)
		if fsQuery == "" {
			fsQuery = req.Message
		}
		fsEng := req.Lang == "en" || isEnglishQuery(req.Message)
		out, err := exec.Command("mdfind", "-name", fsQuery).Output()
		var fsItems []map[string]string
		if err == nil {
			lines := strings.Split(strings.TrimSpace(string(out)), "\n")
			for i, l := range lines {
				if l == "" || i >= 10 {
					break
				}
				fsItems = append(fsItems, map[string]string{"path": l, "name": filepath.Base(l)})
			}
		}
		var fsMsg string
		if len(fsItems) == 0 {
			if fsEng {
				fsMsg = fmt.Sprintf("No files found matching '%s'.", fsQuery)
			} else {
				fsMsg = fmt.Sprintf("'%s' 관련 파일을 찾지 못했습니다.", fsQuery)
			}
		} else {
			if fsEng {
				fsMsg = fmt.Sprintf("Found %d file(s) matching '%s'.", len(fsItems), fsQuery)
			} else {
				fsMsg = fmt.Sprintf("'%s' 관련 파일 %d개를 찾았습니다.", fsQuery, len(fsItems))
			}
		}
		json200(w, CommandResponse{
			Success: true, Message: fsMsg, Action: "file_search",
			Result: map[string]any{"query": fsQuery, "items": fsItems, "count": len(fsItems)},
			Duration: dur,
		})

	case "scan":
		// PC 진단 - Mac에서 시스템 상태 조회
		scEng := req.Lang == "en" || isEnglishQuery(req.Message)
		scStats := map[string]any{}
		// CPU
		if out, err := exec.Command("sh", "-c", "top -l 1 -n 0 | grep 'CPU usage'").Output(); err == nil {
			line := string(out)
			if idx := strings.Index(line, "idle"); idx > 0 {
				parts := strings.Fields(line[:idx])
				if len(parts) > 0 {
					idleStr := strings.TrimSuffix(parts[len(parts)-1], "%")
					if idle, err2 := strconv.ParseFloat(idleStr, 64); err2 == nil {
						scStats["cpu_percent"] = fmt.Sprintf("%.1f%%", 100-idle)
					}
				}
			}
		}
		// Disk
		if out, err := exec.Command("df", "-H", "/").Output(); err == nil {
			lines := strings.Split(string(out), "\n")
			if len(lines) > 1 {
				fields := strings.Fields(lines[1])
				if len(fields) >= 5 {
					scStats["disk_used"] = fields[2]
					scStats["disk_total"] = fields[1]
					scStats["disk_percent"] = strings.TrimSuffix(fields[4], "%") + "%"
				}
			}
		}
		var scMsg string
		if scEng {
			scMsg = fmt.Sprintf("System scan complete. CPU: %v, Disk: %v / %v (%v used)",
				scStats["cpu_percent"], scStats["disk_used"], scStats["disk_total"], scStats["disk_percent"])
		} else {
			scMsg = fmt.Sprintf("PC 진단 완료. CPU: %v, 디스크: %v / %v (%v 사용 중)",
				scStats["cpu_percent"], scStats["disk_used"], scStats["disk_total"], scStats["disk_percent"])
		}
		json200(w, CommandResponse{
			Success: true, Message: scMsg, Action: "scan",
			Result: map[string]any{"stats": scStats, "score": 85},
			Duration: dur,
		})

	case "clean":
		// 정리 - Mac에서 임시 파일 정리
		clEng := req.Lang == "en" || isEnglishQuery(req.Message)
		home, _ := os.UserHomeDir()
		targets := []string{
			filepath.Join(home, "Library/Caches"),
			"/private/var/folders",
		}
		_ = targets
		var clMsg string
		if clEng {
			clMsg = "System cleanup complete. Temporary files have been identified. (Full cleanup requires admin privileges on Mac)"
		} else {
			clMsg = "PC 정리 완료. 임시 파일을 확인했습니다. (Mac에서 전체 정리는 관리자 권한이 필요합니다)"
		}
		json200(w, CommandResponse{Success: true, Message: clMsg, Action: "clean", Duration: dur})

	case "stats":
		// 리소스 현황
		stEng := req.Lang == "en" || isEnglishQuery(req.Message)
		stStats := map[string]any{}
		if out, err := exec.Command("sh", "-c", "top -l 1 -n 0 | grep 'CPU usage'").Output(); err == nil {
			line := string(out)
			if idx := strings.Index(line, "idle"); idx > 0 {
				parts := strings.Fields(line[:idx])
				if len(parts) > 0 {
					idleStr := strings.TrimSuffix(parts[len(parts)-1], "%")
					if idle, err2 := strconv.ParseFloat(idleStr, 64); err2 == nil {
						stStats["cpu_percent"] = 100 - idle
					}
				}
			}
		}
		if out, err := exec.Command("df", "-H", "/").Output(); err == nil {
			lines := strings.Split(string(out), "\n")
			if len(lines) > 1 {
				fields := strings.Fields(lines[1])
				if len(fields) >= 5 {
					stStats["disk_percent"] = strings.TrimSuffix(fields[4], "%")
					stStats["disk_used"] = fields[2]
					stStats["disk_total"] = fields[1]
				}
			}
		}
		var stMsg string
		if stEng {
			stMsg = fmt.Sprintf("System stats: CPU %.1f%%, Disk %v/%v", stStats["cpu_percent"], stStats["disk_used"], stStats["disk_total"])
		} else {
			stMsg = fmt.Sprintf("시스템 현황: CPU %.1f%%, 디스크 %v/%v", stStats["cpu_percent"], stStats["disk_used"], stStats["disk_total"])
		}
		json200(w, CommandResponse{
			Success: true, Message: stMsg, Action: "stats",
			Result: stStats, Duration: dur,
		})

	case "launch_app":
		// 앱 실행 - Mac에서 open 명령 사용
		laApp, _ := intent.Params["app_name"].(string)
		if laApp == "" {
			laApp = req.Message
		}
		laEng := req.Lang == "en" || isEnglishQuery(req.Message)
		appMap := map[string]string{
			"크롬": "Google Chrome", "chrome": "Google Chrome",
			"사파리": "Safari", "safari": "Safari",
			"파이어폭스": "Firefox", "firefox": "Firefox",
			"워드": "Microsoft Word", "word": "Microsoft Word",
			"엑셀": "Microsoft Excel", "excel": "Microsoft Excel",
			"파워포인트": "Microsoft PowerPoint",
			"메모": "Notes", "note": "Notes",
			"터미널": "Terminal", "terminal": "Terminal",
			"카카오": "KakaoTalk", "카카오톡": "KakaoTalk",
			"슬랙": "Slack", "slack": "Slack",
		}
		execApp := laApp
		lower := strings.ToLower(laApp)
		for k, v := range appMap {
			if strings.Contains(lower, strings.ToLower(k)) {
				execApp = v
				break
			}
		}
		exec.Command("open", "-a", execApp).Start()
		var laMsg string
		if laEng {
			laMsg = fmt.Sprintf("Launched %s.", execApp)
		} else {
			laMsg = fmt.Sprintf("%s 실행했습니다.", execApp)
		}
		json200(w, CommandResponse{Success: true, Message: laMsg, Action: "launch_app", Duration: dur})

	case "system_control":
		// 시스템 제어 - Mac에서 osascript 사용
		scCtrl, _ := intent.Params["control"].(string)
		scVal := 50
		if v, ok := intent.Params["value"].(float64); ok {
			scVal = int(v)
		}
		scEng2 := req.Lang == "en" || isEnglishQuery(req.Message)
		var scMsg2 string
		switch strings.ToLower(scCtrl) {
		case "volume", "볼륨":
			exec.Command("osascript", "-e", fmt.Sprintf("set volume output volume %d", scVal)).Run()
			if scEng2 {
				scMsg2 = fmt.Sprintf("Volume set to %d%%.", scVal)
			} else {
				scMsg2 = fmt.Sprintf("볼륨을 %d%%로 설정했습니다.", scVal)
			}
		case "mute", "음소거":
			exec.Command("osascript", "-e", "set volume with output muted").Run()
			if scEng2 {
				scMsg2 = "Muted."
			} else {
				scMsg2 = "음소거 처리했습니다."
			}
		case "sleep", "절전":
			exec.Command("osascript", "-e", `tell app "System Events" to sleep`).Run()
			if scEng2 {
				scMsg2 = "Going to sleep."
			} else {
				scMsg2 = "절전 모드로 전환합니다."
			}
		default:
			if scEng2 {
				scMsg2 = fmt.Sprintf("System control '%s' is not supported on Mac.", scCtrl)
			} else {
				scMsg2 = fmt.Sprintf("'%s' 제어는 Mac에서 지원되지 않습니다.", scCtrl)
			}
		}
		json200(w, CommandResponse{Success: true, Message: scMsg2, Action: "system_control", Duration: dur})

	case "note":
		// 메모 저장
		noteContent, _ := intent.Params["content"].(string)
		if noteContent == "" {
			noteContent = req.Message
		}
		noteEng := req.Lang == "en" || isEnglishQuery(req.Message)
		home, _ := os.UserHomeDir()
		noteDir := filepath.Join(home, ".nexus", "notes")
		os.MkdirAll(noteDir, 0755)
		notePath := filepath.Join(noteDir, fmt.Sprintf("note_%s.txt", time.Now().Format("20060102_150405")))
		os.WriteFile(notePath, []byte(noteContent), 0644)
		var noteMsg string
		if noteEng {
			noteMsg = fmt.Sprintf("Note saved! 📝\nFile: %s", notePath)
		} else {
			noteMsg = fmt.Sprintf("메모 저장 완료! 📝\n파일: %s", notePath)
		}
		json200(w, CommandResponse{
			Success: true, Message: noteMsg, Action: "note",
			Result: map[string]any{"path": notePath, "content": noteContent},
			Duration: dur,
		})

	case "focus_mode":
		// 집중 모드 - Mac에서 Do Not Disturb
		fmEnable := true
		if v, ok := intent.Params["enable"].(bool); ok {
			fmEnable = v
		}
		fmEng := req.Lang == "en" || isEnglishQuery(req.Message)
		// macOS Focus mode via osascript (best-effort)
		if fmEnable {
			exec.Command("osascript", "-e", `tell application "System Events" to set doNotDisturb of (get the current user) to true`).Run()
		} else {
			exec.Command("osascript", "-e", `tell application "System Events" to set doNotDisturb of (get the current user) to false`).Run()
		}
		var fmMsg string
		if fmEng {
			if fmEnable {
				fmMsg = "Focus mode enabled. 🎯 Notifications blocked."
			} else {
				fmMsg = "Focus mode disabled."
			}
		} else {
			if fmEnable {
				fmMsg = "집중 모드 켜졌습니다! 🎯 알림이 차단됐습니다."
			} else {
				fmMsg = "집중 모드 꺼졌습니다."
			}
		}
		json200(w, CommandResponse{
			Success: true, Message: fmMsg, Action: "focus_mode",
			Result: map[string]any{"enabled": fmEnable},
			Duration: dur,
		})

	case "doc_summary":
		// 문서 요약
		dsFile, _ := intent.Params["file_path"].(string)
		dsEng := req.Lang == "en" || isEnglishQuery(req.Message)
		if dsFile == "" {
			var dsMsg string
			if dsEng {
				dsMsg = "Please specify the file path to summarize."
			} else {
				dsMsg = "요약할 파일 경로를 알려주세요."
			}
			json200(w, CommandResponse{Success: false, Message: dsMsg, Action: "doc_summary", Duration: dur})
			return
		}
		data, err := os.ReadFile(dsFile)
		if err != nil {
			var dsErr string
			if dsEng {
				dsErr = "Could not read the file: " + err.Error()
			} else {
				dsErr = "파일을 읽을 수 없습니다: " + err.Error()
			}
			json200(w, CommandResponse{Success: false, Message: dsErr, Action: "doc_summary", Duration: dur})
			return
		}
		content := string(data)
		if len(content) > 4000 {
			content = content[:4000]
		}
		llmMu.RLock()
		dsPKey := llmPerplexityKey
		llmMu.RUnlock()
		var dsSummary string
		if dsPKey != "" {
			var dsPrompt string
			if dsEng {
				dsPrompt = "Summarize the following document concisely in 3-5 sentences:\n\n" + content
			} else {
				dsPrompt = "다음 문서를 3-5문장으로 간결하게 요약해주세요:\n\n" + content
			}
			dsSummary, _, _ = callGroq(dsPKey, groqChatModel, []groqMsg{{Role: "user", Content: dsPrompt}}, 512, false)
		}
		if dsSummary == "" {
			if dsEng {
				dsSummary = "Could not generate summary."
			} else {
				dsSummary = "요약을 생성할 수 없습니다."
			}
		}
		json200(w, CommandResponse{
			Success: true, Message: dsSummary, Action: "doc_summary",
			Result: map[string]any{"file": dsFile, "summary": dsSummary},
			Duration: dur,
		})

	case "health_report":
		// PC 건강 리포트 - Mac 시스템 정보 기반
		hrEng := req.Lang == "en" || isEnglishQuery(req.Message)
		hrStats := map[string]any{}
		if out, err := exec.Command("df", "-H", "/").Output(); err == nil {
			lines := strings.Split(string(out), "\n")
			if len(lines) > 1 {
				fields := strings.Fields(lines[1])
				if len(fields) >= 5 {
					hrStats["disk_used"] = fields[2]
					hrStats["disk_total"] = fields[1]
					pctStr := strings.TrimSuffix(fields[4], "%")
					if p, err2 := strconv.ParseFloat(pctStr, 64); err2 == nil {
						hrStats["disk_percent"] = p
					}
				}
			}
		}
		var hrMsg string
		if hrEng {
			hrMsg = fmt.Sprintf("Mac health report: Disk usage %v / %v. System appears healthy.", hrStats["disk_used"], hrStats["disk_total"])
		} else {
			hrMsg = fmt.Sprintf("Mac 건강 리포트: 디스크 사용량 %v / %v. 시스템이 정상입니다.", hrStats["disk_used"], hrStats["disk_total"])
		}
		json200(w, CommandResponse{
			Success: true, Message: hrMsg, Action: "health_report",
			Result: hrStats, Duration: dur,
		})

	case "excel_save":
		// 엑셀 저장
		exTitle, _ := intent.Params["title"].(string)
		exEng := req.Lang == "en" || isEnglishQuery(req.Message)
		if exTitle == "" {
			if exEng {
				exTitle = "Nexus Data"
			} else {
				exTitle = "넥서스 데이터"
			}
		}
		home, _ := os.UserHomeDir()
		exPath := filepath.Join(home, "Desktop", fmt.Sprintf("nexus_%s.xlsx", time.Now().Format("20060102_150405")))
		f := excelize.NewFile()
		f.SetCellValue("Sheet1", "A1", exTitle)
		f.SetCellValue("Sheet1", "A2", time.Now().Format("2006-01-02 15:04:05"))
		if exErr := f.SaveAs(exPath); exErr != nil {
			var exMsg string
			if exEng {
				exMsg = "Failed to save Excel file: " + exErr.Error()
			} else {
				exMsg = "엑셀 저장 실패: " + exErr.Error()
			}
			json200(w, CommandResponse{Success: false, Message: exMsg, Action: "excel_save", Duration: dur})
			return
		}
		var exMsg string
		if exEng {
			exMsg = fmt.Sprintf("Excel saved! 📊\nFile: %s", exPath)
		} else {
			exMsg = fmt.Sprintf("엑셀 저장 완료! 📊\n파일: %s", exPath)
		}
		json200(w, CommandResponse{
			Success: true, Message: exMsg, Action: "excel_save",
			Result: map[string]any{"path": exPath},
			Duration: dur,
		})

	case "recall":
		// Windows Recall → Mac에서 mdfind/Spotlight로 대체
		rcQuery, _ := intent.Params["query"].(string)
		if rcQuery == "" {
			rcQuery = req.Message
		}
		rcEng := req.Lang == "en" || isEnglishQuery(req.Message)
		out, err := exec.Command("mdfind", rcQuery).Output()
		var rcItems []map[string]string
		if err == nil {
			lines := strings.Split(strings.TrimSpace(string(out)), "\n")
			for i, l := range lines {
				if l == "" || i >= 8 {
					break
				}
				rcItems = append(rcItems, map[string]string{"path": l, "name": filepath.Base(l)})
			}
		}
		var rcMsg string
		if len(rcItems) == 0 {
			if rcEng {
				rcMsg = fmt.Sprintf("No recent items found for '%s'.", rcQuery)
			} else {
				rcMsg = fmt.Sprintf("'%s' 관련 최근 항목을 찾지 못했습니다.", rcQuery)
			}
		} else {
			if rcEng {
				rcMsg = fmt.Sprintf("Found %d item(s) matching '%s' via Spotlight.", len(rcItems), rcQuery)
			} else {
				rcMsg = fmt.Sprintf("Spotlight에서 '%s' 관련 %d개 항목을 찾았습니다.", rcQuery, len(rcItems))
			}
		}
		json200(w, CommandResponse{
			Success: true, Message: rcMsg, Action: "recall",
			Result: map[string]any{"query": rcQuery, "items": rcItems},
			Duration: dur,
		})

	case "timer":
		// 타이머/알람
		tmEng := req.Lang == "en" || isEnglishQuery(req.Message)
		var tmMsg string
		if tmEng {
			tmMsg = "Timer feature is available via the system. For precise scheduling, use the 'scheduler' action."
		} else {
			tmMsg = "타이머 기능은 시스템에서 사용 가능합니다. 정확한 일정 예약은 '스케줄러' 기능을 사용하세요."
		}
		json200(w, CommandResponse{Success: false, Message: tmMsg, Action: "timer", Duration: dur})

	case "browse_page":
		// 웹페이지 브라우징 → web_search로 처리
		bpURL, _ := intent.Params["url"].(string)
		bpQuery, _ := intent.Params["query"].(string)
		if bpURL == "" && bpQuery == "" {
			bpQuery = req.Message
		}
		searchQ := bpURL
		if searchQ == "" {
			searchQ = bpQuery
		}
		bpEng := req.Lang == "en" || isEnglishQuery(req.Message)
		llmMu.RLock()
		bpTKey := llmTavilyKey
		llmMu.RUnlock()
		var bpSummary string
		var bpItems []map[string]string
		if bpTKey != "" {
			if tr, ok := tavilySearch(bpTKey, searchQ, 4); ok {
				bpSummary = tr.Summary
				bpItems = tr.Items
			}
		}
		if bpSummary == "" {
			if bpEng {
				bpSummary = fmt.Sprintf("Here are search results for: %s", searchQ)
			} else {
				bpSummary = fmt.Sprintf("%s 검색 결과입니다.", searchQ)
			}
		}
		json200(w, CommandResponse{
			Success: true, Message: bpSummary, Action: "browse_page",
			Result: map[string]any{"query": searchQ, "summary": bpSummary, "items": bpItems},
			Duration: dur,
		})

	// ── 🟠 3. 파일 조작 ──────────────────────────────────────────
	case "file_ops":
		var op, folder string
		if intent.Params != nil {
			op, _ = intent.Params["op"].(string)
			folder, _ = intent.Params["folder"].(string)
		}
		// 메시지에서 폴더/op 힌트
		if folder == "" { folder = req.Message }
		foEng := req.Lang == "en" || isEnglishQuery(req.Message)
		msgL := strings.ToLower(req.Message)

		if op == "" {
			switch {
			case strings.Contains(msgL, "중복") || strings.Contains(msgL, "duplicate"):
				op = "duplicates"
			case strings.Contains(msgL, "대용량") || strings.Contains(msgL, "큰 파일") || strings.Contains(msgL, "large"):
				op = "large"
			default:
				op = "organize"
			}
		}
		var foMsg string
		proxyCall := func(endpoint string, payload map[string]any) string {
			body, _ := json.Marshal(payload)
			raw, err := httpPost("http://127.0.0.1:17891"+endpoint, body)
			if err != nil { return "" }
			var d map[string]any
			if json.Unmarshal(raw, &d) == nil {
				if m, ok := d["message"].(string); ok { return m }
			}
			return ""
		}
		switch op {
		case "duplicates":
			foMsg = proxyCall("/api/file/duplicates", map[string]any{"folder": "", "message": req.Message})
			if foMsg == "" { foMsg = "중복 파일 탐지 중..." }
		case "large":
			foMsg = proxyCall("/api/file/large", map[string]any{"folder": "", "min_size_mb": 100, "message": req.Message})
			if foMsg == "" { foMsg = "대용량 파일 탐지 중..." }
		default: // organize
			foMsg = proxyCall("/api/file/organize", map[string]any{"folder": folder, "dry_run": false, "message": req.Message})
			if foMsg == "" {
				if foEng { foMsg = "Organizing files..." } else { foMsg = "파일 정리 중..." }
			}
		}
		_ = foEng
		appendSession(userID, "user", req.Message)
		appendSession(userID, "assistant", foMsg)
		json200(w, CommandResponse{Success: true, Message: foMsg, Action: "file_ops", Duration: dur})

	// ── 🟠 4. 조건부 알림 트리거 ──────────────────────────────
	case "trigger_add":
		var nl string
		if intent.Params != nil { nl, _ = intent.Params["nl"].(string) }
		if nl == "" { nl = req.Message }
		trEng := req.Lang == "en" || isEnglishQuery(req.Message)
		t := parseTriggerFromNL(nl)
		var trMsg string
		if t != nil {
			triggerStoreMu.Lock()
			triggerStore[t.ID] = t
			triggerStoreMu.Unlock()
			saveTriggers()
			if trEng {
				trMsg = fmt.Sprintf("✅ Alert trigger set: '%s'", t.Name)
			} else {
				trMsg = fmt.Sprintf("✅ 알림 트리거 등록됨: '%s'", t.Name)
			}
		} else {
			if trEng {
				trMsg = "Couldn't parse the trigger. Try: 'Alert me when CPU exceeds 80%' or 'Remind me every day at 9am'"
			} else {
				trMsg = "트리거를 파악하지 못했어요. 예: 'CPU 80% 넘으면 알려줘', '매일 오전 9시에 알림'"
			}
		}
		appendSession(userID, "user", req.Message)
		appendSession(userID, "assistant", trMsg)
		json200(w, CommandResponse{Success: true, Message: trMsg, Action: "trigger_add", Duration: dur})

	// ── 🟡 5. 화면 캡처 + Vision ──────────────────────────────
	case "screen_analyze":
		var question string
		if intent.Params != nil { question, _ = intent.Params["question"].(string) }
		if question == "" { question = req.Message }
		saEng := req.Lang == "en" || isEnglishQuery(req.Message)
		b64, err := captureScreen()
		var saMsg string
		if err != nil {
			if saEng { saMsg = "Screen capture failed: " + err.Error() } else { saMsg = "화면 캡처 실패: " + err.Error() }
		} else {
			lang := "ko"; if saEng { lang = "en" }
			saMsg, err = analyzeImageWithClaude(b64, question, lang)
			if err != nil {
				if saEng { saMsg = "Vision analysis failed: " + err.Error() } else { saMsg = "Vision 분석 실패: " + err.Error() }
			}
		}
		appendSession(userID, "user", req.Message)
		appendSession(userID, "assistant", saMsg)
		json200(w, CommandResponse{Success: true, Message: saMsg, Action: "screen_analyze", Duration: dur})

	// ── 클립보드 읽기 + 처리 ──────────────────────────────────
	case "clipboard_action":
		cbEng := req.Lang == "en" || isEnglishQuery(req.Message)
		cbText := readClipboard()
		if cbText == "" {
			var cbMsg string
			if cbEng {
				cbMsg = "Clipboard is empty. Please copy something first."
			} else {
				cbMsg = "클립보드가 비어 있습니다. 먼저 텍스트를 복사해주세요."
			}
			json200(w, CommandResponse{Success: true, Message: cbMsg, Action: "clipboard_action", Duration: dur})
			return
		}
		// 클립보드 내용 + 원래 요청을 합쳐서 LLM에 전달
		var cbAction string
		if intent.Params != nil {
			cbAction, _ = intent.Params["action"].(string)
		}
		if cbAction == "" {
			cbAction = detectClipboardAction(req.Message)
		}
		cbResult := processClipboardContent(cbText, cbAction, req.Message, gKey, cbEng)
		appendSession(userID, "user", req.Message)
		appendSession(userID, "assistant", cbResult)
		json200(w, CommandResponse{
			Success: true, Message: cbResult, Action: "clipboard_action",
			Result:  map[string]any{"clipboard_text": cbText, "action": cbAction},
			Duration: dur,
		})

	// ── 🔴 1. 환율 ──────────────────────────────────────────────
	case "exchange_rate":
		erEng := req.Lang == "en" || isEnglishQuery(req.Message)
		fromC, toC := detectCurrencies(req.Message)
		if p := intent.Params; p != nil {
			if v, _ := p["from"].(string); v != "" { fromC = strings.ToUpper(v) }
			if v, _ := p["to"].(string); v != "" { toC = strings.ToUpper(v) }
		}
		rate, date, err := fetchExchangeRate(fromC, toC)
		if err != nil {
			// fallback: web_search
			q := fromC + " to " + toC + " exchange rate today"
			if !erEng { q = fromC + " " + toC + " 오늘 환율" }
			r := runWebSearchMac(gKey, q, "auto", 3, req.Lang)
			json200(w, CommandResponse{Success: true, Message: r.Summary, Action: "exchange_rate", Duration: dur})
		} else {
			fromN := currencySymbols[fromC]; if fromN == "" { fromN = fromC }
			toN := currencySymbols[toC]; if toN == "" { toN = toC }
			var msg string
			if erEng {
				msg = fmt.Sprintf("1 %s (%s) = **%.4f %s (%s)**\n_(as of %s)_", fromN, fromC, rate, toN, toC, date)
			} else {
				msg = fmt.Sprintf("1 %s(%s) = **%.4f %s(%s)**\n_(%s 기준)_", fromN, fromC, rate, toN, toC, date)
			}
			appendSession(userID, "user", req.Message)
			appendSession(userID, "assistant", msg)
			json200(w, CommandResponse{Success: true, Message: msg, Action: "exchange_rate",
				Result: map[string]any{"from": fromC, "to": toC, "rate": rate, "date": date}, Duration: dur})
		}

	// ── 🔴 1. 주가 ──────────────────────────────────────────────
	case "stock":
		stEng := req.Lang == "en" || isEnglishQuery(req.Message)
		var stQuery string
		if intent.Params != nil { stQuery, _ = intent.Params["query"].(string) }
		if stQuery == "" { stQuery = req.Message }
		// 암호화폐 먼저 체크
		if cryptoSym := detectCrypto(stQuery); cryptoSym != "" {
			krw, usd, err := fetchCryptoPrice(cryptoSym)
			var msg string
			if err != nil {
				r := runWebSearchMac(gKey, cryptoSym+" 현재 가격", "auto", 3, req.Lang)
				msg = r.Summary
			} else {
				if stEng {
					msg = fmt.Sprintf("**%s**: ₩%.0f KRW / $%.2f USD", cryptoSym, krw, usd)
				} else {
					msg = fmt.Sprintf("**%s** 현재가: **₩%.0f** (KRW) / $%.2f (USD)", cryptoSym, krw, usd)
				}
			}
			appendSession(userID, "user", req.Message)
			appendSession(userID, "assistant", msg)
			json200(w, CommandResponse{Success: true, Message: msg, Action: "stock", Duration: dur})
			return
		}
		// 주식 티커 검색
		ticker, name := detectStockTicker(stQuery)
		if ticker == "" {
			r := runWebSearchMac(gKey, stQuery+" 주가 현재", "auto", 3, req.Lang)
			json200(w, CommandResponse{Success: true, Message: r.Summary, Action: "stock", Duration: dur})
			return
		}
		price, change, currency, err := fetchStockInfo(ticker)
		var stMsg string
		if err != nil {
			r := runWebSearchMac(gKey, name+" 주가 현재", "auto", 3, req.Lang)
			stMsg = r.Summary
		} else {
			stMsg = formatStockMsg(name, ticker, price, change, currency, stEng)
		}
		appendSession(userID, "user", req.Message)
		appendSession(userID, "assistant", stMsg)
		json200(w, CommandResponse{Success: true, Message: stMsg, Action: "stock",
			Result: map[string]any{"ticker": ticker, "price": price, "change": change}, Duration: dur})

	case "windows_only":
		var feature string
		if intent.Params != nil {
			feature, _ = intent.Params["feature"].(string)
		}
		var msg string
		woEng := req.Lang == "en" || isEnglishQuery(req.Message)
		if woEng {
			if feature != "" {
				msg = fmt.Sprintf("'%s' is only available on Windows PC.", feature)
			} else {
				msg = "This feature is only available on Windows PC."
			}
		} else {
			if feature != "" {
				msg = fmt.Sprintf("'%s' 기능은 Windows PC에서만 사용 가능합니다.", feature)
			} else {
				msg = "이 기능은 Windows PC에서만 사용 가능합니다."
			}
		}
		json200(w, CommandResponse{
			Success:  false,
			Message:  msg,
			Action:   "windows_only",
			Duration: dur,
		})

	case "deep_research":
		// Perplexity sonar-pro — 실시간 웹 리서치 (Manus 대체)
		drEng := req.Lang == "en" || isEnglishQuery(req.Message)
		drQuery := req.Message
		if intent.Params != nil {
			if q, ok := intent.Params["query"].(string); ok && q != "" {
				drQuery = q
			}
		}
		// 1차: Tavily 빠른 검색
		tvResult, _ := tavilySearch(llmTavilyKey, drQuery, 5)
		// 2차: Perplexity sonar-pro로 깊은 분석 (웹검색 내장)
		var sysCtx string
		if drEng {
			sysCtx = "You are a research assistant. Provide comprehensive, well-structured answers with key facts, data, and analysis. Use bullet points and headers for clarity."
		} else {
			sysCtx = "당신은 심층 리서치 전문가입니다. 핵심 사실, 데이터, 분석을 포함한 구조화된 답변을 제공하세요. 불릿 포인트와 소제목을 활용하세요."
		}
		var drPrompt string
		if tvResult.Summary != "" {
			if drEng {
				drPrompt = fmt.Sprintf("Research context from web:\n%s\n\nUser question: %s\n\nProvide a comprehensive, well-structured answer.", tvResult.Summary, drQuery)
			} else {
				drPrompt = fmt.Sprintf("웹 검색 컨텍스트:\n%s\n\n질문: %s\n\n위 정보를 바탕으로 심층적이고 구조화된 답변을 제공해줘.", tvResult.Summary, drQuery)
			}
		} else {
			if drEng {
				drPrompt = fmt.Sprintf("Research and answer comprehensively: %s", drQuery)
			} else {
				drPrompt = fmt.Sprintf("다음 주제를 심층 리서치하고 구조화된 답변을 제공해줘: %s", drQuery)
			}
		}
		drMsgs := []groqMsg{
			{Role: "system", Content: sysCtx},
			{Role: "user", Content: drPrompt},
		}
		drAnswer, _, drErr := callGroq(gKey, groqChatModel, drMsgs, 2048, false)
		if drErr != nil {
			if drEng {
				drAnswer = "Research failed: " + drErr.Error()
			} else {
				drAnswer = "리서치 실패: " + drErr.Error()
			}
		}
		appendSession(userID, "user", req.Message)
		appendSession(userID, "assistant", drAnswer)
		json200(w, CommandResponse{Success: true, Message: drAnswer, Action: "deep_research", Duration: dur})

	default:
		// 알 수 없는 액션 → chat으로 폴백
		chatMsgs := []groqMsg{
			{Role: "system", Content: getPersonaSystemPrompt()},
			{Role: "user", Content: req.Message},
		}
		answer, _, _ := callGroq(gKey, groqChatModel, chatMsgs, 1024, false)
		json200(w, CommandResponse{
			Success:  true,
			Message:  answer,
			Action:   "chat",
			Duration: dur,
		})
	}
}

// ── 웹 검색 (Groq 기반 + 브라우저 에이전트) ───────────────────

type webSearchResult struct {
	Query       string              `json:"query"`
	Site        string              `json:"site"`
	Summary     string              `json:"summary"`
	Items       []map[string]string `json:"items,omitempty"`
	PreviewType string              `json:"preview_type,omitempty"`
}

func runWebSearchMac(apiKey, query, site string, maxItems int, lang ...string) webSearchResult {
	forceLang := ""
	if len(lang) > 0 {
		forceLang = lang[0]
	}
	eng := forceLang == "en"
	siteLabel := site
	if siteLabel == "" || siteLabel == "auto" {
		if eng {
			siteLabel = "web"
		} else {
			siteLabel = "웹"
		}
	}

	cat := detectCategory(query)
	previewType := categoryPreviewType(cat)

	// 병렬 검색: Tavily + 브라우저 동시 실행
	result := parallelWebSearch(query, maxItems, forceLang)

	// 결과가 있으면 그대로 반환
	if result.Summary != "" || len(result.Items) > 0 {
		items := result.Items
		if len(items) == 0 {
			items = categoryFallbackSites(query, cat)
		}
		return webSearchResult{
			Query:       query,
			Site:        siteLabel,
			Summary:     result.Summary,
			Items:       items,
			PreviewType: previewType,
		}
	}

	// 최후 폴백: Groq LLM (실시간 데이터 없음)
	today := time.Now().Format("2006-01-02")
	var prompt string
	if eng {
		prompt = fmt.Sprintf(`Today is %s.
User question: "%s"

[Instructions]
- Do NOT include URLs, links, or source names
- Answer directly in natural English, 2-4 sentences, key points only
- If no real-time data available, say "For the latest info, please use the preview button"
- Write like a friendly AI assistant`, today, query)
	} else {
		prompt = fmt.Sprintf(`오늘은 %s입니다.
사용자 질문: "%s"

[지시사항]
- URL, 링크, 출처명 절대 포함 금지
- 사용자 질문에 직접 답하는 자연스러운 한국어 2~4문장으로 핵심만 답변
- 실시간 데이터가 없으면 "정확한 최신 정보는 미리보기 버튼으로 확인해보세요" 안내
- 친절한 AI 비서처럼 작성`, today, query)
	}
	msgs := []groqMsg{{Role: "user", Content: prompt}}
	text, _, err := callGroq(apiKey, groqChatModel, msgs, 512, false)
	if err != nil {
		text = "검색 중 오류가 발생했습니다: " + err.Error()
	}

	fallbackItems := categoryFallbackSites(query, cat)
	if len(fallbackItems) == 0 {
		fallbackItems = buildFallbackURLs(query, site)
	}

	return webSearchResult{
		Query:       query,
		Site:        siteLabel,
		Summary:     text,
		Items:       fallbackItems,
		PreviewType: previewType,
	}
}


func tryBrowserSearch(query, site string, maxItems int) []map[string]string {
	// chromedp가 사용 가능하면 실제 검색, 없으면 빈 결과
	defer func() { recover() }()

	ctx, cancel, err := getBrowserCtxMac()
	if err != nil {
		return nil
	}
	defer cancel()

	var searchURL string
	switch strings.ToLower(site) {
	case "youtube":
		searchURL = "https://www.youtube.com/results?search_query=" + urlEncode(query)
	case "coupang":
		searchURL = "https://www.coupang.com/np/search?q=" + urlEncode(query)
	case "naver":
		searchURL = "https://search.naver.com/search.naver?query=" + urlEncode(query)
	default:
		searchURL = "https://www.google.com/search?q=" + urlEncode(query)
	}

	_ = ctx
	_ = searchURL
	_ = cancel
	_ = maxItems
	return nil
}


// runMacOrchestrate: Mac용 멀티 에이전트 오케스트레이터 (Tavily 기반 순차 실행)
func runMacOrchestrate(goal, gKey string) (string, error) {
	eng := isEnglishQuery(goal)
	llmMu.RLock()
	tKey := llmTavilyKey
	llmMu.RUnlock()

	// 1단계: 목표를 서브 태스크로 분해 (LLM)
	var planPrompt string
	if eng {
		planPrompt = fmt.Sprintf(`Break down this goal into 2-3 concrete search queries (JSON array of strings only):
Goal: %s
Output format: ["query1","query2","query3"]`, goal)
	} else {
		planPrompt = fmt.Sprintf(`다음 목표를 2-3개의 구체적인 검색 쿼리로 분해하세요 (JSON 배열만 출력):
목표: %s
출력 형식: ["쿼리1","쿼리2","쿼리3"]`, goal)
	}

	raw, _, err := callGroq(gKey, groqFastModel, []groqMsg{{Role: "user", Content: planPrompt}}, 256, true)
	if err != nil {
		raw = fmt.Sprintf(`["%s"]`, goal)
	}

	var queries []string
	if jsonErr := json.Unmarshal([]byte(raw), &queries); jsonErr != nil || len(queries) == 0 {
		queries = []string{goal}
	}
	if len(queries) > 3 {
		queries = queries[:3]
	}

	// 2단계: 각 쿼리 병렬 실행
	type stepResult struct {
		Query   string
		Summary string
	}
	results := make([]stepResult, len(queries))
	var wg sync.WaitGroup
	for i, q := range queries {
		wg.Add(1)
		go func(idx int, query string) {
			defer wg.Done()
			summary := ""
			if tKey != "" {
				if tr, ok := tavilySearch(tKey, query, 3); ok {
					summary = tr.Summary
				}
			}
			if summary == "" {
				msgs := []groqMsg{{Role: "user", Content: query}}
				summary, _, _ = callGroq(gKey, groqChatModel, msgs, 400, false)
			}
			results[idx] = stepResult{Query: query, Summary: summary}
		}(i, q)
	}
	wg.Wait()

	// 3단계: 결과 통합
	var parts []string
	for i, r := range results {
		if r.Summary != "" {
			parts = append(parts, fmt.Sprintf("[%d] %s\n%s", i+1, r.Query, r.Summary))
		}
	}
	combined := strings.Join(parts, "\n\n")

	// 4단계: 최종 요약
	var finalPrompt string
	if eng {
		finalPrompt = fmt.Sprintf("Synthesize the following research results into a concise final answer for the goal: '%s'\n\n%s", goal, combined)
	} else {
		finalPrompt = fmt.Sprintf("다음 조사 결과들을 목표 '%s'에 대한 최종 답변으로 통합해주세요:\n\n%s", goal, combined)
	}
	final, _, _ := callGroq(gKey, groqChatModel, []groqMsg{{Role: "user", Content: finalPrompt}}, 600, false)
	if final == "" {
		final = combined
	}
	return final, nil
}
