//go:build !windows

package main

import (
	"archive/zip"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/xuri/excelize/v2"
)

// cmdCtx: handleCommand 내 switch 케이스에 전달되는 공통 컨텍스트
type cmdCtx struct {
	w      http.ResponseWriter
	req    CommandRequest
	params map[string]any
	msg    string // intent.Message (LLM 응답 메시지)
	dur    string
	gKey   string
	tKey   string
	userID string
	start  time.Time
}
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

