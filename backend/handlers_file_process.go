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
	"regexp"
	"strconv"
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
	lang := getLang(r)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Files) == 0 {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("files 필요", "files required", lang)})
		return
	}
	if req.Params == nil {
		req.Params = map[string]string{}
	}
	if req.Operation == "" || req.Operation == "auto" {
		req.Operation = detectFileOp(req.Query, req.Files)
	}

	// 영상 편집 작업은 base64 크기 200 MB 제한 (~267 MB base64)
	const maxVideoBase64 = 360 * 1024 * 1024
	isVideoOp := req.Operation == "video_trim" || req.Operation == "video_compress" ||
		req.Operation == "video_speed" || req.Operation == "video_subtitle"
	if isVideoOp && len(req.Files[0].Data) > maxVideoBase64 {
		writeJSON(w, 413, map[string]any{
			"success": false,
			"message": msgT(
				"파일이 너무 큽니다 (최대 200 MB). 먼저 압축 후 다시 시도해주세요.",
				"File too large (max 200 MB). Please compress it first.",
				lang,
			),
		})
		return
	}

	switch req.Operation {
	case "resize":
		handleResize(w, req.Files[0], req.Params, req.Query, lang)
	case "to_gif":
		handleToGIF(w, req.Files, req.Params, lang)
	case "compare":
		handleCompare(w, req.Files, req.Query, lang)
	case "convert":
		handleConvert(w, req.Files[0], req.Params, lang)
	case "video_trim":
		handleVideoTrim(w, req.Files[0], req.Params, req.Query, lang)
	case "video_compress":
		handleVideoCompress(w, req.Files[0], req.Params, req.Query, lang)
	case "video_speed":
		handleVideoSpeed(w, req.Files[0], req.Params, req.Query, lang)
	case "video_subtitle":
		handleVideoSubtitle(w, req.Files, req.Params, req.Query, lang)
	default:
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("지원하지 않는 operation: "+req.Operation, "Unsupported operation: "+req.Operation, lang)})
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
	isVid := len(files) > 0 && strings.HasPrefix(files[0].MimeType, "video/")

	switch {
	// 영상 편집 ops (이미지보다 먼저 체크)
	case isVid && has("잘라", "자르기", "구간", "trim", "초부터", "분부터", "부터", "까지"):
		return "video_trim"
	case isVid && has("압축", "용량", "줄여", "compress", "작게", "가볍게"):
		return "video_compress"
	case isVid && has("배속", "빠르게", "느리게", "speed", "빨리", "천천히"):
		return "video_speed"
	case isVid && (has("자막", "subtitle", "srt") || (len(files) >= 2 && strings.HasSuffix(strings.ToLower(files[1].Name), ".srt"))):
		return "video_subtitle"
	// 이미지 ops
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
func handleResize(w http.ResponseWriter, f fileInput, params map[string]string, query string, lang string) {
	img, format, err := decodeImage(f.Data)
	if err != nil {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("이미지 디코딩 실패: "+err.Error(), "Image decode failed: "+err.Error(), lang)})
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
func handleToGIF(w http.ResponseWriter, files []fileInput, params map[string]string, lang string) {
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
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("변환 가능한 이미지가 없습니다", "No convertible images found", lang)})
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
func handleCompare(w http.ResponseWriter, files []fileInput, query string, lang string) {
	if len(files) < 2 {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("비교하려면 파일 2개 이상 필요", "At least 2 files required for comparison", lang)})
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
func handleConvert(w http.ResponseWriter, f fileInput, params map[string]string, lang string) {
	img, _, err := decodeImage(f.Data)
	if err != nil {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("디코딩 실패: "+err.Error(), "Decode failed: "+err.Error(), lang)})
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

// ── 영상 편집 공통 헬퍼 ───────────────────────────────────────────

// decodeVideoToTemp: base64 영상을 임시 파일로 저장하고 경로 반환
func decodeVideoToTemp(data, name string) (string, string, error) {
	raw := data
	if idx := strings.Index(raw, ","); idx >= 0 {
		raw = raw[idx+1:]
	}
	b, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return "", "", fmt.Errorf("base64 decode: %w", err)
	}
	ext := filepath.Ext(name)
	if ext == "" {
		ext = ".mp4"
	}
	tmp, err := os.MkdirTemp("", "nexus_vedit_*")
	if err != nil {
		return "", "", err
	}
	inPath := filepath.Join(tmp, "input"+ext)
	if err := os.WriteFile(inPath, b, 0644); err != nil {
		os.RemoveAll(tmp)
		return "", "", err
	}
	return tmp, inPath, nil
}

// encodeFileToBase64: 파일을 base64로 읽어 반환
func encodeFileToBase64(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// parseTimeToSeconds: "1분30초", "90초", "1:30" 등 → 초 단위 float64
func parseTimeToSeconds(s string) float64 {
	s = strings.TrimSpace(s)
	// HH:MM:SS 또는 MM:SS
	if regexp.MustCompile(`^\d+:\d{2}(:\d{2})?$`).MatchString(s) {
		parts := strings.Split(s, ":")
		var total float64
		for _, p := range parts {
			v, _ := strconv.ParseFloat(p, 64)
			total = total*60 + v
		}
		return total
	}
	// "1분 30초" 형태
	var total float64
	if m := regexp.MustCompile(`(\d+)\s*분`).FindStringSubmatch(s); m != nil {
		v, _ := strconv.ParseFloat(m[1], 64)
		total += v * 60
	}
	if m := regexp.MustCompile(`(\d+)\s*초`).FindStringSubmatch(s); m != nil {
		v, _ := strconv.ParseFloat(m[1], 64)
		total += v
	}
	if total == 0 {
		// 순수 숫자 → 초
		v, _ := strconv.ParseFloat(regexp.MustCompile(`\d+`).FindString(s), 64)
		total = v
	}
	return total
}

// parseTrimTimes: 쿼리에서 시작/끝 시간 추출
// "30초부터 2분까지" → (30, 120)
// "처음 1분" → (0, 60)
func parseTrimTimes(q string) (start, end float64) {
	q = strings.ToLower(q)

	// "처음 N분/초" 패턴
	if m := regexp.MustCompile(`처음\s*([\d분초: ]+)`).FindStringSubmatch(q); m != nil {
		end = parseTimeToSeconds(m[1])
		return 0, end
	}
	// "부터 ~ 까지" 패턴
	re := regexp.MustCompile(`([\d분초: ]+?)\s*부터\s*([\d분초: ]+?)\s*까지`)
	if m := re.FindStringSubmatch(q); m != nil {
		return parseTimeToSeconds(m[1]), parseTimeToSeconds(m[2])
	}
	// "~부터 ~" (까지 없음)
	re2 := regexp.MustCompile(`([\d분초: ]+?)\s*부터\s*([\d분초: ]+)`)
	if m := re2.FindStringSubmatch(q); m != nil {
		return parseTimeToSeconds(m[1]), parseTimeToSeconds(m[2])
	}
	return 0, 0
}

// parseSpeedFactor: "2배속", "0.5배", "1.5x" 등 → float64
func parseSpeedFactor(q string) float64 {
	re := regexp.MustCompile(`(\d+\.?\d*)\s*(?:배속|배|x|X)`)
	if m := re.FindStringSubmatch(q); m != nil {
		v, _ := strconv.ParseFloat(m[1], 64)
		if v > 0 {
			return v
		}
	}
	if strings.Contains(q, "절반") || strings.Contains(q, "0.5") {
		return 0.5
	}
	return 2.0 // 기본값
}

// ── handleVideoTrim ────────────────────────────────────────────
// POST /api/file/process  operation=video_trim
// params: start (초), end (초) — 없으면 쿼리에서 자동 파싱
func handleVideoTrim(w http.ResponseWriter, f fileInput, params map[string]string, query, lang string) {
	ffmpeg := findFFmpeg()
	if ffmpeg == "" {
		writeJSON(w, 500, map[string]any{"success": false, "message": msgT("ffmpeg 미설치. 설치 후 재시도해주세요.", "ffmpeg not found. Please install and retry.", lang)})
		return
	}
	tmp, inPath, err := decodeVideoToTemp(f.Data, f.Name)
	if err != nil {
		writeJSON(w, 400, map[string]any{"success": false, "message": err.Error()})
		return
	}
	defer os.RemoveAll(tmp)

	start, end := parseTrimTimes(query)
	if ps, ok := params["start"]; ok {
		start = parseTimeToSeconds(ps)
	}
	if pe, ok := params["end"]; ok {
		end = parseTimeToSeconds(pe)
	}
	if end <= start {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("시간 범위가 잘못됐어요. '30초부터 2분까지 잘라줘'처럼 말해보세요.", "Invalid time range. Try: 'trim from 0:30 to 2:00'.", lang)})
		return
	}

	ext := filepath.Ext(f.Name)
	if ext == "" {
		ext = ".mp4"
	}
	outPath := filepath.Join(tmp, "trimmed"+ext)
	cmd := exec.Command(ffmpeg,
		"-y",
		"-ss", fmt.Sprintf("%.3f", start),
		"-to", fmt.Sprintf("%.3f", end),
		"-i", inPath,
		"-c", "copy",
		outPath,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": msgT("영상 자르기 실패: ", "Trim failed: ", lang) + string(out)})
		return
	}

	b64, err := encodeFileToBase64(outPath)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": "output read failed"})
		return
	}
	baseName := strings.TrimSuffix(f.Name, ext)
	startMin, startSec := int(start)/60, int(start)%60
	endMin, endSec := int(end)/60, int(end)%60
	writeJSON(w, 200, map[string]any{
		"success":   true,
		"operation": "video_trim",
		"message":   msgT(fmt.Sprintf("✂️ %d분%02d초 ~ %d분%02d초 구간을 잘랐어요!", startMin, startSec, endMin, endSec), fmt.Sprintf("✂️ Trimmed %d:%02d ~ %d:%02d!", startMin, startSec, endMin, endSec), lang),
		"data":      b64,
		"file_name": fmt.Sprintf("%s_trimmed%s", baseName, ext),
		"mime_type": f.MimeType,
	})
}

// ── handleVideoCompress ────────────────────────────────────────
// operation=video_compress  params: crf (기본 28)
func handleVideoCompress(w http.ResponseWriter, f fileInput, params map[string]string, query, lang string) {
	ffmpeg := findFFmpeg()
	if ffmpeg == "" {
		writeJSON(w, 500, map[string]any{"success": false, "message": msgT("ffmpeg 미설치.", "ffmpeg not found.", lang)})
		return
	}
	tmp, inPath, err := decodeVideoToTemp(f.Data, f.Name)
	if err != nil {
		writeJSON(w, 400, map[string]any{"success": false, "message": err.Error()})
		return
	}
	defer os.RemoveAll(tmp)

	crf := "28"
	if v, ok := params["crf"]; ok {
		crf = v
	} else {
		q := strings.ToLower(query)
		if strings.Contains(q, "많이") || strings.Contains(q, "최대") || strings.Contains(q, "많은") {
			crf = "34"
		} else if strings.Contains(q, "조금") || strings.Contains(q, "살짝") {
			crf = "26"
		}
	}

	ext := filepath.Ext(f.Name)
	if ext == "" {
		ext = ".mp4"
	}
	outPath := filepath.Join(tmp, "compressed.mp4")
	cmd := exec.Command(ffmpeg,
		"-y", "-i", inPath,
		"-vcodec", "libx264",
		"-crf", crf,
		"-preset", "fast",
		"-acodec", "aac",
		"-b:a", "128k",
		outPath,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": msgT("압축 실패: ", "Compress failed: ", lang) + string(out)})
		return
	}

	origSize, _ := os.Stat(inPath)
	newSize, _ := os.Stat(outPath)
	ratio := 0.0
	if origSize != nil && origSize.Size() > 0 {
		ratio = float64(newSize.Size()) / float64(origSize.Size()) * 100
	}

	b64, _ := encodeFileToBase64(outPath)
	baseName := strings.TrimSuffix(f.Name, ext)
	writeJSON(w, 200, map[string]any{
		"success":   true,
		"operation": "video_compress",
		"message":   msgT(fmt.Sprintf("📦 압축 완료! 원본 대비 %.0f%% 크기로 줄었어요.", ratio), fmt.Sprintf("📦 Compressed to %.0f%% of original size.", ratio), lang),
		"data":      b64,
		"file_name": baseName + "_compressed.mp4",
		"mime_type": "video/mp4",
	})
}

// ── handleVideoSpeed ───────────────────────────────────────────
// operation=video_speed  params: speed (예: "2.0", "0.5")
func handleVideoSpeed(w http.ResponseWriter, f fileInput, params map[string]string, query, lang string) {
	ffmpeg := findFFmpeg()
	if ffmpeg == "" {
		writeJSON(w, 500, map[string]any{"success": false, "message": msgT("ffmpeg 미설치.", "ffmpeg not found.", lang)})
		return
	}
	tmp, inPath, err := decodeVideoToTemp(f.Data, f.Name)
	if err != nil {
		writeJSON(w, 400, map[string]any{"success": false, "message": err.Error()})
		return
	}
	defer os.RemoveAll(tmp)

	speed := parseSpeedFactor(query)
	if v, ok := params["speed"]; ok {
		if sv, err := strconv.ParseFloat(v, 64); err == nil && sv > 0 {
			speed = sv
		}
	}
	// ffmpeg atempo는 0.5~2.0 범위만 지원 → 체이닝 필요
	if speed < 0.25 || speed > 4.0 {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("0.25~4.0배속만 지원해요.", "Only 0.25x to 4.0x speed is supported.", lang)})
		return
	}

	// video: pts 배율 = 1/speed, audio: atempo 체이닝
	pts := fmt.Sprintf("%.4f", 1.0/speed)
	var audioFilter string
	if speed > 2.0 {
		audioFilter = fmt.Sprintf("atempo=2.0,atempo=%.4f", speed/2.0)
	} else if speed < 0.5 {
		audioFilter = fmt.Sprintf("atempo=0.5,atempo=%.4f", speed/0.5)
	} else {
		audioFilter = fmt.Sprintf("atempo=%.4f", speed)
	}

	ext := filepath.Ext(f.Name)
	if ext == "" {
		ext = ".mp4"
	}
	outPath := filepath.Join(tmp, "speed.mp4")
	cmd := exec.Command(ffmpeg,
		"-y", "-i", inPath,
		"-filter_complex", fmt.Sprintf("[0:v]setpts=%s*PTS[v];[0:a]%s[a]", pts, audioFilter),
		"-map", "[v]", "-map", "[a]",
		"-vcodec", "libx264", "-preset", "fast",
		"-acodec", "aac",
		outPath,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": msgT("속도 변환 실패: ", "Speed change failed: ", lang) + string(out)})
		return
	}

	b64, _ := encodeFileToBase64(outPath)
	baseName := strings.TrimSuffix(f.Name, ext)
	speedLabel := fmt.Sprintf("%.2g", speed)
	writeJSON(w, 200, map[string]any{
		"success":   true,
		"operation": "video_speed",
		"message":   msgT(fmt.Sprintf("⚡ %s배속 변환 완료!", speedLabel), fmt.Sprintf("⚡ Speed changed to %sx!", speedLabel), lang),
		"data":      b64,
		"file_name": fmt.Sprintf("%s_%sx.mp4", baseName, speedLabel),
		"mime_type": "video/mp4",
	})
}

// ── handleVideoSubtitle ────────────────────────────────────────
// operation=video_subtitle
// files[0]=영상, files[1]=.srt 자막 (또는 params["srt_text"]에 SRT 내용)
func handleVideoSubtitle(w http.ResponseWriter, files []fileInput, params map[string]string, query, lang string) {
	ffmpeg := findFFmpeg()
	if ffmpeg == "" {
		writeJSON(w, 500, map[string]any{"success": false, "message": msgT("ffmpeg 미설치.", "ffmpeg not found.", lang)})
		return
	}
	if len(files) == 0 {
		writeJSON(w, 400, map[string]any{"success": false, "message": "no files"})
		return
	}
	tmp, inPath, err := decodeVideoToTemp(files[0].Data, files[0].Name)
	if err != nil {
		writeJSON(w, 400, map[string]any{"success": false, "message": err.Error()})
		return
	}
	defer os.RemoveAll(tmp)

	// SRT 파일 경로 확보
	srtPath := filepath.Join(tmp, "sub.srt")
	if len(files) >= 2 && (strings.HasSuffix(strings.ToLower(files[1].Name), ".srt") || strings.HasSuffix(strings.ToLower(files[1].Name), ".vtt")) {
		raw := files[1].Data
		if idx := strings.Index(raw, ","); idx >= 0 {
			raw = raw[idx+1:]
		}
		srtBytes, err2 := base64.StdEncoding.DecodeString(raw)
		if err2 != nil {
			writeJSON(w, 400, map[string]any{"success": false, "message": "srt decode failed"})
			return
		}
		os.WriteFile(srtPath, srtBytes, 0644)
	} else if v, ok := params["srt_text"]; ok && v != "" {
		os.WriteFile(srtPath, []byte(v), 0644)
	} else {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("자막 파일(.srt)을 함께 첨부해주세요.", "Please attach a subtitle file (.srt) along with the video.", lang)})
		return
	}

	ext := filepath.Ext(files[0].Name)
	if ext == "" {
		ext = ".mp4"
	}
	outPath := filepath.Join(tmp, "subtitled.mp4")
	// Windows 경로의 백슬래시를 escape
	escapedSrt := strings.ReplaceAll(srtPath, `\`, `\\`)
	escapedSrt = strings.ReplaceAll(escapedSrt, `:`, `\\:`)
	cmd := exec.Command(ffmpeg,
		"-y", "-i", inPath,
		"-vf", fmt.Sprintf("subtitles='%s'", escapedSrt),
		"-vcodec", "libx264", "-preset", "fast",
		"-acodec", "copy",
		outPath,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": msgT("자막 삽입 실패: ", "Subtitle burn-in failed: ", lang) + string(out)})
		return
	}

	b64, _ := encodeFileToBase64(outPath)
	baseName := strings.TrimSuffix(files[0].Name, ext)
	writeJSON(w, 200, map[string]any{
		"success":   true,
		"operation": "video_subtitle",
		"message":   msgT("🎬 자막이 영상에 삽입됐어요!", "🎬 Subtitles burned into video!", lang),
		"data":      b64,
		"file_name": baseName + "_subtitled.mp4",
		"mime_type": "video/mp4",
	})
}
