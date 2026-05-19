package main

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"path/filepath"
	"strings"
)

// ══════════════════════════════════════════════════════════════
//  POST /api/file/metadata
//  파일(이미지·PDF·Office) 메타데이터 추출 → Claude AI 해석
// ══════════════════════════════════════════════════════════════

type MetaResult struct {
	FileName string            `json:"file_name"`
	FileType string            `json:"file_type"`
	Fields   map[string]string `json:"fields"`
	GPS      *GPSInfo          `json:"gps,omitempty"`
	MapURL   string            `json:"map_url,omitempty"`
	AIReport string            `json:"ai_report"`
}

type GPSInfo struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

func handleFileMetadata(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Files []fileInput `json:"files"`
		Query string      `json:"query"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Files) == 0 {
		writeJSON(w, 400, map[string]any{"success": false, "message": "files 필요"})
		return
	}

	var results []MetaResult
	for _, f := range req.Files {
		res := extractMeta(f)
		res.AIReport = generateMetaReport(f.Name, res, req.Query)
		results = append(results, res)
	}

	summary := buildMetaSummary(results)
	writeJSON(w, 200, map[string]any{
		"success": true,
		"results": results,
		"message": summary,
	})
}

// ── 메타데이터 추출 라우터 ─────────────────────────────────────
func extractMeta(f fileInput) MetaResult {
	ext := strings.ToLower(filepath.Ext(f.Name))
	res := MetaResult{
		FileName: f.Name,
		FileType: ext,
		Fields:   map[string]string{},
	}

	raw := decodeBase64Data(f.Data)
	if raw == nil {
		res.Fields["error"] = "디코딩 실패"
		return res
	}

	switch {
	case isImageExt(ext):
		extractEXIF(raw, &res)
	case ext == ".pdf":
		extractPDFMeta(raw, &res)
	case ext == ".docx" || ext == ".xlsx" || ext == ".pptx":
		extractOfficeMeta(raw, &res)
	default:
		res.Fields["size_bytes"] = fmt.Sprintf("%d", len(raw))
		res.Fields["encoding"] = detectEncoding(raw)
	}

	return res
}

// ── EXIF 파서 (순수 Go, 외부 라이브러리 없음) ────────────────
func extractEXIF(data []byte, res *MetaResult) {
	// JPEG EXIF 시작 마커 탐색
	if len(data) < 4 {
		return
	}

	// JPEG: FF D8
	if data[0] != 0xFF || data[1] != 0xD8 {
		// PNG: 기본 정보만
		if len(data) >= 8 && string(data[1:4]) == "PNG" {
			res.Fields["format"] = "PNG"
			w, h := parsePNGDimensions(data)
			if w > 0 {
				res.Fields["width"] = fmt.Sprintf("%d px", w)
				res.Fields["height"] = fmt.Sprintf("%d px", h)
			}
		}
		return
	}

	res.Fields["format"] = "JPEG"

	// APP1 세그먼트 탐색 (EXIF)
	i := 2
	for i < len(data)-4 {
		if data[i] != 0xFF {
			break
		}
		marker := data[i+1]
		segLen := int(binary.BigEndian.Uint16(data[i+2 : i+4]))

		if marker == 0xE1 && i+4+segLen <= len(data) {
			seg := data[i+4 : i+2+segLen]
			if len(seg) > 6 && string(seg[0:4]) == "Exif" {
				parseIFD(seg[6:], res)
			}
		}
		i += 2 + segLen
	}
}

func parseIFD(data []byte, res *MetaResult) {
	if len(data) < 8 {
		return
	}
	var bo binary.ByteOrder
	if string(data[0:2]) == "II" {
		bo = binary.LittleEndian
	} else {
		bo = binary.BigEndian
	}

	if len(data) < 8 {
		return
	}
	offset := int(bo.Uint32(data[4:8]))
	if offset >= len(data) || offset < 0 {
		return
	}

	if offset+2 > len(data) {
		return
	}
	count := int(bo.Uint16(data[offset : offset+2]))
	pos := offset + 2

	var gpsIFDOffset int

	for i := 0; i < count && pos+12 <= len(data); i++ {
		tag := bo.Uint16(data[pos : pos+2])
		typ := bo.Uint16(data[pos+2 : pos+4])
		cnt := bo.Uint32(data[pos+4 : pos+8])
		val := data[pos+8 : pos+12]

		switch tag {
		case 0x010F: // Make
			res.Fields["camera_make"] = readASCII(data, bo, typ, cnt, val)
		case 0x0110: // Model
			res.Fields["camera_model"] = readASCII(data, bo, typ, cnt, val)
		case 0x0132: // DateTime
			res.Fields["datetime"] = readASCII(data, bo, typ, cnt, val)
		case 0x013B: // Artist
			res.Fields["artist"] = readASCII(data, bo, typ, cnt, val)
		case 0x8769: // ExifIFD
			off := int(bo.Uint32(val))
			parseExifSubIFD(data, bo, off, res)
		case 0x8825: // GPS IFD
			gpsIFDOffset = int(bo.Uint32(val))
		case 0xA001: // ColorSpace
			_ = cnt
		}
		pos += 12
	}

	if gpsIFDOffset > 0 {
		parseGPSIFD(data, bo, gpsIFDOffset, res)
	}
}

func parseExifSubIFD(data []byte, bo binary.ByteOrder, offset int, res *MetaResult) {
	if offset+2 > len(data) || offset < 0 {
		return
	}
	count := int(bo.Uint16(data[offset : offset+2]))
	pos := offset + 2
	for i := 0; i < count && pos+12 <= len(data); i++ {
		tag := bo.Uint16(data[pos : pos+2])
		typ := bo.Uint16(data[pos+2 : pos+4])
		cnt := bo.Uint32(data[pos+4 : pos+8])
		val := data[pos+8 : pos+12]
		switch tag {
		case 0xA002: // PixelXDimension
			if typ == 3 {
				res.Fields["width"] = fmt.Sprintf("%d px", bo.Uint16(val[:2]))
			} else {
				res.Fields["width"] = fmt.Sprintf("%d px", bo.Uint32(val))
			}
		case 0xA003: // PixelYDimension
			if typ == 3 {
				res.Fields["height"] = fmt.Sprintf("%d px", bo.Uint16(val[:2]))
			} else {
				res.Fields["height"] = fmt.Sprintf("%d px", bo.Uint32(val))
			}
		case 0x829A: // ExposureTime
			off := int(bo.Uint32(val))
			if off+8 <= len(data) {
				num := bo.Uint32(data[off : off+4])
				den := bo.Uint32(data[off+4 : off+8])
				if den > 0 {
					res.Fields["exposure_time"] = fmt.Sprintf("%d/%d초", num, den)
				}
			}
		case 0x829D: // FNumber
			off := int(bo.Uint32(val))
			if off+8 <= len(data) {
				num := bo.Uint32(data[off : off+4])
				den := bo.Uint32(data[off+4 : off+8])
				if den > 0 {
					res.Fields["f_number"] = fmt.Sprintf("f/%.1f", float64(num)/float64(den))
				}
			}
		case 0x8827: // ISO
			res.Fields["iso"] = fmt.Sprintf("%d", bo.Uint16(val[:2]))
		case 0x9003: // DateTimeOriginal
			res.Fields["datetime_original"] = readASCII(data, bo, typ, cnt, val)
		case 0x9286: // UserComment
			res.Fields["user_comment"] = strings.TrimSpace(readASCII(data, bo, typ, cnt, val))
		case 0xA434: // LensModel
			res.Fields["lens_model"] = readASCII(data, bo, typ, cnt, val)
		}
		pos += 12
	}
}

func parseGPSIFD(data []byte, bo binary.ByteOrder, offset int, res *MetaResult) {
	if offset+2 > len(data) || offset < 0 {
		return
	}
	count := int(bo.Uint16(data[offset : offset+2]))
	pos := offset + 2

	gps := map[uint16]interface{}{}

	for i := 0; i < count && pos+12 <= len(data); i++ {
		tag := bo.Uint16(data[pos : pos+2])
		val := data[pos+8 : pos+12]
		off := int(bo.Uint32(val))

		switch tag {
		case 1: // GPSLatitudeRef
			gps[1] = string([]byte{val[0]})
		case 2: // GPSLatitude
			if off+24 <= len(data) {
				gps[2] = parseGPSRational(data[off:off+24], bo)
			}
		case 3: // GPSLongitudeRef
			gps[3] = string([]byte{val[0]})
		case 4: // GPSLongitude
			if off+24 <= len(data) {
				gps[4] = parseGPSRational(data[off:off+24], bo)
			}
		case 6: // GPSAltitude
			if off+8 <= len(data) {
				num := bo.Uint32(data[off : off+4])
				den := bo.Uint32(data[off+4 : off+8])
				if den > 0 {
					res.Fields["altitude"] = fmt.Sprintf("%.1f m", float64(num)/float64(den))
				}
			}
		}
		pos += 12
	}

	latArr, ok2 := gps[2].([]float64)
	lngArr, ok4 := gps[4].([]float64)
	if ok2 && ok4 && len(latArr) == 3 && len(lngArr) == 3 {
		lat := latArr[0] + latArr[1]/60 + latArr[2]/3600
		lng := lngArr[0] + lngArr[1]/60 + lngArr[2]/3600
		if ref, ok := gps[1].(string); ok && ref == "S" {
			lat = -lat
		}
		if ref, ok := gps[3].(string); ok && ref == "W" {
			lng = -lng
		}
		if !math.IsNaN(lat) && !math.IsNaN(lng) {
			res.GPS = &GPSInfo{Lat: lat, Lng: lng}
			res.Fields["gps_lat"] = fmt.Sprintf("%.6f", lat)
			res.Fields["gps_lng"] = fmt.Sprintf("%.6f", lng)
			res.MapURL = fmt.Sprintf("https://maps.google.com/?q=%.6f,%.6f", lat, lng)
			res.Fields["map_url"] = res.MapURL
		}
	}
}

func parseGPSRational(data []byte, bo binary.ByteOrder) []float64 {
	result := make([]float64, 3)
	for i := 0; i < 3; i++ {
		off := i * 8
		if off+8 > len(data) {
			break
		}
		num := bo.Uint32(data[off : off+4])
		den := bo.Uint32(data[off+4 : off+8])
		if den > 0 {
			result[i] = float64(num) / float64(den)
		}
	}
	return result
}

func readASCII(data []byte, bo binary.ByteOrder, typ uint16, cnt uint32, val []byte) string {
	if cnt <= 4 {
		return strings.TrimRight(string(val[:cnt]), "\x00")
	}
	off := int(bo.Uint32(val))
	end := off + int(cnt)
	if end > len(data) {
		end = len(data)
	}
	if off < 0 || off >= len(data) {
		return ""
	}
	return strings.TrimRight(string(data[off:end]), "\x00")
}

func parsePNGDimensions(data []byte) (int, int) {
	// PNG IHDR: offset 16, 4+4 bytes width/height
	if len(data) < 24 {
		return 0, 0
	}
	w := int(binary.BigEndian.Uint32(data[16:20]))
	h := int(binary.BigEndian.Uint32(data[20:24]))
	return w, h
}

// ── PDF 메타데이터 파서 ───────────────────────────────────────
func extractPDFMeta(data []byte, res *MetaResult) {
	res.Fields["format"] = "PDF"
	res.Fields["size"] = fmt.Sprintf("%.1f KB", float64(len(data))/1024)

	text := string(data)

	// /Author, /Title, /Creator, /Producer, /CreationDate
	pdfFields := map[string]string{
		"/Title":        "title",
		"/Author":       "author",
		"/Creator":      "creator",
		"/Producer":     "producer",
		"/CreationDate": "creation_date",
		"/ModDate":      "mod_date",
		"/Subject":      "subject",
		"/Keywords":     "keywords",
	}
	for pdf, key := range pdfFields {
		if idx := strings.Index(text, pdf); idx != -1 {
			rest := text[idx+len(pdf):]
			val := extractPDFValue(rest)
			if val != "" {
				res.Fields[key] = val
			}
		}
	}

	// 페이지 수 추정
	pageCount := strings.Count(text, "/Page")
	if pageCount > 0 {
		res.Fields["page_count_est"] = fmt.Sprintf("약 %d페이지", pageCount)
	}
}

func extractPDFValue(s string) string {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return ""
	}
	// (value) 형식
	if s[0] == '(' {
		end := strings.Index(s[1:], ")")
		if end >= 0 {
			return strings.TrimSpace(s[1 : end+1])
		}
	}
	// /value 형식
	if s[0] == '/' {
		parts := strings.Fields(s[1:])
		if len(parts) > 0 {
			return parts[0]
		}
	}
	return ""
}

// ── Office (OOXML) 메타데이터 파서 ───────────────────────────
func extractOfficeMeta(data []byte, res *MetaResult) {
	ext := res.FileType
	switch ext {
	case ".docx":
		res.Fields["format"] = "Word Document"
	case ".xlsx":
		res.Fields["format"] = "Excel Spreadsheet"
	case ".pptx":
		res.Fields["format"] = "PowerPoint Presentation"
	}
	res.Fields["size"] = fmt.Sprintf("%.1f KB", float64(len(data))/1024)

	// ZIP 내부 docProps/core.xml 탐색
	text := string(data)
	xmlFields := map[string]string{
		"<dc:title>":         "title",
		"<dc:creator>":       "author",
		"<dc:subject>":       "subject",
		"<dc:description>":   "description",
		"<cp:lastModifiedBy>": "last_modified_by",
		"<dcterms:created":   "created",
		"<dcterms:modified":  "modified",
		"<cp:revision>":      "revision",
	}
	for xml, key := range xmlFields {
		if idx := strings.Index(text, xml); idx != -1 {
			rest := text[idx+len(xml):]
			end := strings.Index(rest, "<")
			if end > 0 {
				val := strings.TrimSpace(rest[:end])
				// xsd:dateTime 형식 정리
				if strings.Contains(val, "T") {
					val = strings.ReplaceAll(val, "T", " ")
					val = strings.TrimSuffix(val, "Z")
				}
				if val != "" {
					res.Fields[key] = val
				}
			}
		}
	}
}

// ── AI 보고서 생성 ─────────────────────────────────────────────
func generateMetaReport(fileName string, res MetaResult, query string) string {
	llmMu.RLock()
	claudeKey := llmClaudeKey
	llmMu.RUnlock()

	// 메타데이터 요약 텍스트 구성
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("파일명: %s\n", fileName))
	for k, v := range res.Fields {
		if k == "map_url" {
			continue
		}
		sb.WriteString(fmt.Sprintf("- %s: %s\n", k, v))
	}
	if res.GPS != nil {
		sb.WriteString(fmt.Sprintf("- GPS: 위도 %.6f, 경도 %.6f\n", res.GPS.Lat, res.GPS.Lng))
		sb.WriteString(fmt.Sprintf("- 지도 링크: %s\n", res.MapURL))
	}
	metaText := sb.String()

	if claudeKey == "" {
		return buildSimpleReport(res, metaText)
	}

	userQ := query
	if userQ == "" {
		userQ = "이 파일의 메타데이터를 분석하고 중요한 정보를 알려줘"
	}

	prompt := fmt.Sprintf("다음 파일의 메타데이터를 분석해줘:\n\n%s\n\n사용자 질문: %s\n\n중요한 발견사항(GPS 위치, 촬영 기기, 작성자, 날짜 등)을 한국어로 설명하고, GPS가 있으면 어떤 지역인지 추정해줘.", metaText, userQ)

	body := map[string]any{
		"model":      claudeHaikuModel,
		"max_tokens": 500,
		"messages": []map[string]any{
			{"role": "user", "content": prompt},
		},
	}
	result := callClaudeAPI(claudeKey, body)
	if result != "" {
		if res.GPS != nil {
			result += fmt.Sprintf("\n\n📍 **[지도에서 보기](%s)**", res.MapURL)
		}
		return result
	}
	return buildSimpleReport(res, metaText)
}

func buildSimpleReport(res MetaResult, metaText string) string {
	var parts []string
	if v, ok := res.Fields["camera_model"]; ok {
		parts = append(parts, fmt.Sprintf("📷 카메라: %s", v))
	}
	if v, ok := res.Fields["datetime_original"]; ok {
		parts = append(parts, fmt.Sprintf("📅 촬영일: %s", v))
	}
	if v, ok := res.Fields["author"]; ok {
		parts = append(parts, fmt.Sprintf("✍️ 작성자: %s", v))
	}
	if res.GPS != nil {
		parts = append(parts, fmt.Sprintf("📍 GPS 위치 발견! [지도에서 보기](%s)", res.MapURL))
	}
	if len(parts) == 0 {
		return "✅ 메타데이터 추출 완료\n\n```\n" + metaText + "\n```"
	}
	return "✅ 메타데이터 분석 결과\n\n" + strings.Join(parts, "\n")
}

func buildMetaSummary(results []MetaResult) string {
	if len(results) == 0 {
		return "분석할 파일이 없습니다."
	}
	if len(results) == 1 {
		return results[0].AIReport
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("✅ %d개 파일 메타데이터 분석 완료\n\n", len(results)))
	for i, r := range results {
		sb.WriteString(fmt.Sprintf("**%d. %s**\n%s\n\n", i+1, r.FileName, r.AIReport))
	}
	return sb.String()
}

// ── 유틸 ──────────────────────────────────────────────────────
func isImageExt(ext string) bool {
	return ext == ".jpg" || ext == ".jpeg" || ext == ".png" ||
		ext == ".tiff" || ext == ".tif" || ext == ".webp" || ext == ".heic"
}

func decodeBase64Data(b64 string) []byte {
	if idx := strings.Index(b64, ","); idx != -1 {
		b64 = b64[idx+1:]
	}
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil
	}
	return raw
}

func detectEncoding(data []byte) string {
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return "UTF-8 with BOM"
	}
	for _, b := range data[:min(512, len(data))] {
		if b > 127 {
			return "Binary / Non-UTF8"
		}
	}
	return "UTF-8 / ASCII"
}

// callClaudeAPI: Claude API 직접 호출 (ai_report 전용)
func callClaudeAPI(key string, body map[string]any) string {
	bodyBytes, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", claudeAPIBase, bytes.NewReader(bodyBytes))
	if err != nil {
		return ""
	}
	req.Header.Set("x-api-key", key)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var res struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	json.NewDecoder(resp.Body).Decode(&res)
	if len(res.Content) > 0 {
		return res.Content[0].Text
	}
	return ""
}
