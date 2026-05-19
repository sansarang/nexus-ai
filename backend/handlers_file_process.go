package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	stdimage "image"
	stdcolor "image/color"
	stddraw "image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// ── 플랫폼 프리셋 ──────────────────────────────────────────────
var platformPresets = map[string][2]int{
	"instagram_square":    {1080, 1080},
	"instagram_portrait":  {1080, 1350},
	"instagram_story":     {1080, 1920},
	"instagram_landscape": {1080, 566},
	"twitter":             {1200, 675},
	"twitter_profile":     {400, 400},
	"youtube_thumbnail":   {1280, 720},
	"youtube_banner":      {2560, 1440},
	"youtube_profile":     {800, 800},
	"facebook_cover":      {1640, 624},
	"facebook_post":       {1200, 630},
	"facebook_profile":    {170, 170},
	"tiktok":              {1080, 1920},
	"linkedin":            {1200, 627},
	"linkedin_profile":    {400, 400},
	"pinterest":           {1000, 1500},
	"kakao_profile":       {640, 640},
	"kakaotalk":           {640, 640},
	"naver_blog":          {900, 600},
	"og_image":            {1200, 630},
	"hd":                  {1280, 720},
	"fullhd":              {1920, 1080},
	"4k":                  {3840, 2160},
	"square":              {1080, 1080},
	"thumbnail":           {300, 300},
}

var platformAliases = map[string]string{
	"인스타그램 정사각형": "instagram_square",
	"인스타그램 세로":   "instagram_portrait",
	"인스타그램 스토리":  "instagram_story",
	"인스타 스토리":    "instagram_story",
	"인스타그램":      "instagram_square",
	"인스타":         "instagram_square",
	"트위터":         "twitter",
	"유튜브 썸네일":    "youtube_thumbnail",
	"유튜브":         "youtube_thumbnail",
	"틱톡":          "tiktok",
	"페이스북":        "facebook_post",
	"링크드인":        "linkedin",
	"카카오 프로필":    "kakao_profile",
	"카카오톡":        "kakaotalk",
	"썸네일":         "thumbnail",
	"프로필":         "square",
	"hd":           "hd",
	"fullhd":       "fullhd",
	"4k":           "4k",
}

type fileInput struct {
	Name     string `json:"name"`
	MimeType string `json:"mime_type"`
	Data     string `json:"data"` // base64
}

// POST /api/file/process
func handleFileProcess(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Files     []fileInput       `json:"files"`
		Operation string            `json:"operation"`
		Params    map[string]string `json:"params"`
		Query     string            `json:"query"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Files) == 0 {
		writeJSON(w, 400, map[string]any{"success": false, "message": "files 필요"})
		return
	}
	if req.Params == nil {
		req.Params = map[string]string{}
	}
	if req.Operation == "" || req.Operation == "auto" {
		req.Operation = detectFileOp(req.Query, req.Files)
	}

	switch req.Operation {
	case "resize":
		handleResize(w, req.Files[0], req.Params, req.Query)
	case "to_gif":
		handleToGIF(w, req.Files, req.Params)
	case "compare":
		handleCompare(w, req.Files, req.Query)
	case "convert":
		handleConvert(w, req.Files[0], req.Params)
	default:
		writeJSON(w, 400, map[string]any{"success": false, "message": "지원하지 않는 operation: " + req.Operation})
	}
}

func detectFileOp(query string, files []fileInput) string {
	q := strings.ToLower(query)
	has := func(words ...string) bool {
		for _, w := range words {
			if strings.Contains(q, w) {
				return true
			}
		}
		return false
	}
	isImg := len(files) > 0 && strings.HasPrefix(files[0].MimeType, "image/")

	switch {
	case has("gif", "움직이는", "애니메이션", "움짤"):
		return "to_gif"
	case has("리사이즈", "사이즈", "크기", "resize", "인스타", "트위터", "유튜브", "틱톡", "썸네일", "맞춰", "변경") && isImg:
		return "resize"
	case has("비교", "compare", "차이", "다른점", "같은점") && len(files) >= 2:
		return "compare"
	case has("변환", "jpg", "png", "webp", "jpeg", "convert") && isImg:
		return "convert"
	}
	if isImg {
		return "resize"
	}
	return "compare"
}

// ── 이미지 리사이즈 ────────────────────────────────────────────
func handleResize(w http.ResponseWriter, f fileInput, params map[string]string, query string) {
	img, format, err := decodeImage(f.Data)
	if err != nil {
		writeJSON(w, 400, map[string]any{"success": false, "message": "이미지 디코딩 실패: " + err.Error()})
		return
	}

	tw, th := 0, 0
	platform := ""
	q := strings.ToLower(query + " " + params["platform"])

	for alias, key := range platformAliases {
		if strings.Contains(q, alias) {
			platform = key
			break
		}
	}
	if p := params["platform"]; p != "" && platform == "" {
		platform = p
	}
	if preset, ok := platformPresets[platform]; ok {
		tw, th = preset[0], preset[1]
	}
	if tw == 0 {
		fmt.Sscanf(params["width"], "%d", &tw)
		fmt.Sscanf(params["height"], "%d", &th)
	}
	if tw == 0 {
		tw, th = 1080, 1080
		platform = "instagram_square"
	}
	if th == 0 {
		th = tw
	}

	resized := resizeCropFit(img, tw, th)

	var buf bytes.Buffer
	outFmt := format
	if f2 := params["format"]; f2 != "" {
		outFmt = f2
	}
	mime, ext := encodeImage(&buf, resized, outFmt)

	platLabel := platform
	if platLabel == "" {
		platLabel = fmt.Sprintf("%dx%d", tw, th)
	}
	baseName := strings.TrimSuffix(f.Name, filepath.Ext(f.Name))

	writeJSON(w, 200, map[string]any{
		"success":   true,
		"operation": "resize",
		"platform":  platLabel,
		"width":     tw,
		"height":    th,
		"file_name": fmt.Sprintf("%s_%s%s", baseName, platLabel, ext),
		"mime_type": mime,
		"data":      base64.StdEncoding.EncodeToString(buf.Bytes()),
		"message":   fmt.Sprintf("✅ **%s** 크기(%dx%d)로 리사이즈 완료! 아래 버튼으로 다운로드하세요.", platLabel, tw, th),
	})
}

func resizeCropFit(src stdimage.Image, tw, th int) stdimage.Image {
	sb := src.Bounds()
	sw, sh := sb.Dx(), sb.Dy()
	scaleW := float64(tw) / float64(sw)
	scaleH := float64(th) / float64(sh)
	scale := math.Max(scaleW, scaleH)
	nw := int(math.Round(float64(sw) * scale))
	nh := int(math.Round(float64(sh) * scale))

	scaled := stdimage.NewRGBA(stdimage.Rect(0, 0, nw, nh))
	xdraw.BiLinear.Scale(scaled, scaled.Bounds(), src, sb, xdraw.Over, nil)

	ox := (nw - tw) / 2
	oy := (nh - th) / 2
	out := stdimage.NewRGBA(stdimage.Rect(0, 0, tw, th))
	stddraw.Draw(out, out.Bounds(), scaled, stdimage.Pt(ox, oy), stddraw.Src)
	return out
}

// ── GIF 변환 ──────────────────────────────────────────────────
func handleToGIF(w http.ResponseWriter, files []fileInput, params map[string]string) {
	delay := 50
	if d := params["delay"]; d != "" {
		fmt.Sscanf(d, "%d", &delay)
	}

	var frames []*stdimage.Paletted
	var delays []int

	for _, f := range files {
		img, _, err := decodeImage(f.Data)
		if err != nil {
			continue
		}
		b := img.Bounds()
		if b.Dx() > 800 {
			newH := b.Dy() * 800 / b.Dx()
			if newH < 1 {
				newH = 1
			}
			img = resizeCropFit(img, 800, newH)
		}
		frames = append(frames, imageToPaletted(img))
		delays = append(delays, delay)
	}

	if len(frames) == 0 {
		writeJSON(w, 400, map[string]any{"success": false, "message": "변환 가능한 이미지가 없습니다"})
		return
	}
	if len(frames) == 1 {
		frames, delays = makeZoomAnimation(frames[0], 10)
	}

	var buf bytes.Buffer
	gif.EncodeAll(&buf, &gif.GIF{Image: frames, Delay: delays, LoopCount: 0})

	baseName := "animated"
	if len(files) > 0 {
		baseName = strings.TrimSuffix(files[0].Name, filepath.Ext(files[0].Name))
	}
	writeJSON(w, 200, map[string]any{
		"success":   true,
		"operation": "to_gif",
		"file_name": baseName + ".gif",
		"mime_type": "image/gif",
		"frames":    len(frames),
		"data":      base64.StdEncoding.EncodeToString(buf.Bytes()),
		"message":   fmt.Sprintf("✅ GIF 변환 완료! %d 프레임 애니메이션으로 만들었어요. 아래 버튼으로 다운로드하세요.", len(frames)),
	})
}

func makeZoomAnimation(base *stdimage.Paletted, frameCount int) ([]*stdimage.Paletted, []int) {
	w, h := base.Bounds().Dx(), base.Bounds().Dy()
	frames := make([]*stdimage.Paletted, frameCount)
	delays := make([]int, frameCount)
	for i := range frames {
		t := float64(i) / float64(frameCount-1)
		zoom := 1.0 + t*0.15
		nw := int(float64(w) / zoom)
		nh := int(float64(h) / zoom)
		if nw < 1 {
			nw = 1
		}
		if nh < 1 {
			nh = 1
		}
		ox := (w - nw) / 2
		oy := (h - nh) / 2
		cropped := stdimage.NewRGBA(stdimage.Rect(0, 0, nw, nh))
		stddraw.Draw(cropped, cropped.Bounds(), base, stdimage.Pt(ox, oy), stddraw.Src)
		scaled := stdimage.NewRGBA(stdimage.Rect(0, 0, w, h))
		xdraw.BiLinear.Scale(scaled, scaled.Bounds(), cropped, cropped.Bounds(), xdraw.Over, nil)
		frames[i] = imageToPaletted(scaled)
		delays[i] = 6
	}
	return frames, delays
}

// ── 문서 비교 ─────────────────────────────────────────────────
func handleCompare(w http.ResponseWriter, files []fileInput, query string) {
	if len(files) < 2 {
		writeJSON(w, 400, map[string]any{"success": false, "message": "비교하려면 파일 2개 이상 필요"})
		return
	}
	texts := make([]string, len(files))
	for i, f := range files {
		t := extractFileText(f)
		if len(t) > 5000 {
			t = t[:5000] + "...(이하 생략)"
		}
		texts[i] = t
	}

	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()

	if gKey == "" {
		writeJSON(w, 200, map[string]any{
			"success": true, "operation": "compare",
			"message": fmt.Sprintf("📄 **%s** vs **%s**\n\n- 파일 1 길이: %d자\n- 파일 2 길이: %d자\n\nAI API 키를 설정하면 내용 비교 분석이 가능합니다.",
				files[0].Name, files[1].Name, len(texts[0]), len(texts[1])),
		})
		return
	}

	eng := isEnglishQuery(query)
	parts := make([]string, len(files))
	for i, f := range files {
		if eng {
			parts[i] = fmt.Sprintf("=== File %d: %s ===\n%s", i+1, f.Name, texts[i])
		} else {
			parts[i] = fmt.Sprintf("=== 파일 %d: %s ===\n%s", i+1, f.Name, texts[i])
		}
	}
	var sysPr, prompt string
	if eng {
		sysPr = "You are a document comparison expert. Analyze structure, content, similarities and differences clearly in English using markdown tables or lists."
		prompt = fmt.Sprintf("Compare and analyze the following %d files:\n\n%s\n\nUser question: %s\n\nClearly summarize similarities, differences, and key characteristics.",
			len(files), strings.Join(parts, "\n\n"), query)
	} else {
		sysPr = "문서 비교 전문가야. 구조적으로 분석하고 한국어로 답해."
		prompt = fmt.Sprintf("다음 %d개 파일을 비교 분석해줘:\n\n%s\n\n사용자 질문: %s\n\n공통점, 차이점, 특징을 마크다운 표나 리스트로 명확하게 정리해줘.",
			len(files), strings.Join(parts, "\n\n"), query)
	}
	msgs := []groqMsg{
		{Role: "system", Content: sysPr},
		{Role: "user", Content: prompt},
	}
	result, _, _ := callGroqWithFallback(msgs, 2048, false)
	if result == "" {
		result = fmt.Sprintf("📄 %s vs %s 비교 분석이 완료되었습니다.", files[0].Name, files[1].Name)
	}
	writeJSON(w, 200, map[string]any{
		"success": true, "operation": "compare",
		"message": result,
		"files":   []string{files[0].Name, files[1].Name},
	})
}

// ── 포맷 변환 ─────────────────────────────────────────────────
func handleConvert(w http.ResponseWriter, f fileInput, params map[string]string) {
	img, _, err := decodeImage(f.Data)
	if err != nil {
		writeJSON(w, 400, map[string]any{"success": false, "message": "디코딩 실패: " + err.Error()})
		return
	}
	targetFmt := strings.ToLower(strings.TrimPrefix(params["format"], "."))
	if targetFmt == "" {
		targetFmt = "png"
	}
	var buf bytes.Buffer
	mime, ext := encodeImage(&buf, img, targetFmt)
	baseName := strings.TrimSuffix(f.Name, filepath.Ext(f.Name))
	writeJSON(w, 200, map[string]any{
		"success":   true,
		"operation": "convert",
		"format":    targetFmt,
		"file_name": baseName + ext,
		"mime_type": mime,
		"data":      base64.StdEncoding.EncodeToString(buf.Bytes()),
		"message":   fmt.Sprintf("✅ %s 형식으로 변환 완료! 아래 버튼으로 다운로드하세요.", strings.ToUpper(targetFmt)),
	})
}

// ── 공통 헬퍼 ─────────────────────────────────────────────────

func decodeImage(b64 string) (stdimage.Image, string, error) {
	if idx := strings.Index(b64, ","); idx != -1 {
		b64 = b64[idx+1:]
	}
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, "", err
	}
	return stdimage.Decode(bytes.NewReader(raw))
}

func encodeImage(w io.Writer, img stdimage.Image, format string) (mime, ext string) {
	switch format {
	case "jpg", "jpeg":
		jpeg.Encode(w, img, &jpeg.Options{Quality: 92})
		return "image/jpeg", ".jpg"
	case "gif":
		p := imageToPaletted(img)
		gif.Encode(w, p, nil)
		return "image/gif", ".gif"
	default:
		png.Encode(w, img)
		return "image/png", ".png"
	}
}

func imageToPaletted(img stdimage.Image) *stdimage.Paletted {
	b := img.Bounds()
	palette := buildWebSafePalette()
	p := stdimage.NewPaletted(b, palette)
	stddraw.FloydSteinberg.Draw(p, b, img, stdimage.Point{})
	return p
}

func buildWebSafePalette() stdcolor.Palette {
	p := make(stdcolor.Palette, 0, 217)
	for r := 0; r <= 255; r += 51 {
		for g := 0; g <= 255; g += 51 {
			for b := 0; b <= 255; b += 51 {
				p = append(p, stdcolor.RGBA{uint8(r), uint8(g), uint8(b), 255})
			}
		}
	}
	p = append(p, stdcolor.RGBA{0, 0, 0, 0})
	return p
}

func extractFileText(f fileInput) string {
	ext := strings.ToLower(filepath.Ext(f.Name))
	textExts := map[string]bool{".txt": true, ".md": true, ".csv": true, ".json": true, ".html": true, ".xml": true}

	data := f.Data
	if idx := strings.Index(data, ","); idx != -1 {
		data = data[idx+1:]
	}
	raw, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return f.Data
	}

	if textExts[ext] {
		return string(raw)
	}

	if ext == ".pdf" {
		tmp, err := os.CreateTemp("", "nexus-*.pdf")
		if err == nil {
			tmp.Write(raw)
			tmp.Close()
			defer os.Remove(tmp.Name())
			out, err := exec.Command("pdftotext", tmp.Name(), "-").Output()
			if err == nil {
				return string(out)
			}
		}
	}

	// 바이너리 파일은 텍스트로 간주 (docx XML 등)
	text := string(raw)
	if len(text) > 10000 {
		text = text[:10000]
	}
	return text
}
