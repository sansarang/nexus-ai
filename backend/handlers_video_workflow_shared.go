// handlers_video_workflow_shared.go — 영상 통합 워크플로 (Mac/Windows 공용)
// "이 영상 다운로드 + 자막 + 요약" 한 번에
//
// 라우팅:
//   "이 영상 요약해줘 https://youtube.com/..." → 자막 추출 + LLM 요약
//   "이 영상 다운로드" → yt-dlp 다운로드
//   "유튜브 자막 추출 + 핵심 정리" → 자막 + 인사이트
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// cmdVideoWorkflow: 영상 URL + 사용자 의도 → 다운로드/자막/요약 자동 선택
// params: { url?, mode? (download|transcript|summary|all) }
func cmdVideoWorkflow(params map[string]any, original, gKey, lang string) (result any, message string) {
	url, _ := params["url"].(string)
	mode, _ := params["mode"].(string)
	if url == "" {
		url = extractVideoURL(original)
	}
	if url == "" {
		errMsg := "영상 URL을 알려주세요. 예: https://youtube.com/watch?v=..."
		if lang == "en" {
			errMsg = "Provide a video URL, e.g., https://youtube.com/watch?v=..."
		}
		return map[string]any{"success": false}, errMsg
	}
	if mode == "" {
		mode = detectVideoMode(original)
	}

	// Python sidecar 호출 (port 17893)
	switch mode {
	case "download":
		return videoDownloadOnly(url, original, lang)
	case "transcript":
		return videoTranscriptOnly(url, original, gKey, lang)
	case "summary", "all":
		// 자막 추출 → LLM 요약 (이미 backend/handlers_video_transcript.go 에 통합돼 있음)
		return videoTranscriptOnly(url, original, gKey, lang)
	}

	// 기본: 자막 + 요약
	return videoTranscriptOnly(url, original, gKey, lang)
}

// videoDownloadOnly: Python sidecar 의 yt-dlp 호출
func videoDownloadOnly(url, original, lang string) (any, string) {
	body, _ := json.Marshal(map[string]any{
		"url":     url,
		"quality": "best",
	})
	resp, err := http.Post("http://127.0.0.1:17893/api/video/download", "application/json", strings.NewReader(string(body)))
	if err != nil {
		errMsg := "Python 사이드카 연결 실패 — yt-dlp 사용 불가"
		if lang == "en" {
			errMsg = "Python sidecar unavailable — yt-dlp not running"
		}
		return map[string]any{"success": false, "error": err.Error()}, errMsg
	}
	defer resp.Body.Close()

	var data map[string]any
	json.NewDecoder(resp.Body).Decode(&data)
	if data == nil {
		data = map[string]any{}
	}
	data["success"] = true
	data["operation"] = "video_download"

	savePath, _ := data["save_path"].(string)
	if savePath == "" {
		savePath, _ = data["path"].(string)
	}
	msg := fmt.Sprintf("✅ 영상 다운로드 완료 — %s", savePath)
	if lang == "en" {
		msg = fmt.Sprintf("✅ Video downloaded — %s", savePath)
	}
	return data, msg
}

// videoTranscriptOnly: 자막 추출 + LLM 요약 (자비스급)
func videoTranscriptOnly(url, original, gKey, lang string) (any, string) {
	// Python sidecar 의 /api/video/transcript
	body, _ := json.Marshal(map[string]any{
		"url":  url,
		"lang": lang,
	})
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Post("http://127.0.0.1:17893/api/video/transcript", "application/json", strings.NewReader(string(body)))
	if err != nil {
		errMsg := "Python 사이드카 연결 실패 — 자막 추출 불가"
		if lang == "en" {
			errMsg = "Python sidecar unavailable — transcript extraction unavailable"
		}
		return map[string]any{"success": false, "error": err.Error()}, errMsg
	}
	defer resp.Body.Close()

	var data struct {
		Success    bool   `json:"success"`
		Transcript string `json:"transcript"`
		Language   string `json:"language"`
		Duration   int    `json:"duration"`
		Source     string `json:"source"`
		Message    string `json:"message"`
	}
	json.NewDecoder(resp.Body).Decode(&data)
	if !data.Success || data.Transcript == "" {
		errMsg := "자막을 가져올 수 없어요. 영상에 자막이 있는지 확인해주세요."
		if data.Message != "" {
			errMsg = data.Message
		}
		return map[string]any{"success": false}, errMsg
	}

	// LLM 요약 (자비스 톤: 3줄 핵심)
	sysPrompt := ""
	if lang == "en" {
		sysPrompt = `Summarize this video transcript in 3 bullet lines:
- Topic + main idea (1 line)
- Key insight or takeaway (1 line)
- Practical action or tip (1 line)
Max 60 words total. No markdown.`
	} else {
		sysPrompt = `이 영상 자막을 3줄로 요약하세요:
- 주제 + 핵심 메시지 (1줄)
- 핵심 인사이트 (1줄)
- 실용 팁/액션 (1줄)
총 100자 이내, 마크다운 X, "~이에요/예요" 친근체.`
	}

	// 토큰 절약: 자막이 너무 길면 처음 + 끝
	transcript := data.Transcript
	if len(transcript) > 8000 {
		transcript = transcript[:4000] + "\n[...중간 생략...]\n" + transcript[len(transcript)-2000:]
	}

	summary, _, _ := callGroqWithFallback([]groqMsg{
		{Role: "system", Content: sysPrompt},
		{Role: "user", Content: transcript},
	}, 200, false)

	if summary == "" {
		summary = data.Transcript[:min(200, len(data.Transcript))] + "..."
	}

	msg := fmt.Sprintf("🎬 영상 요약\n\n%s\n\n📝 자막: %d자 (%s)", summary, len([]rune(data.Transcript)), data.Language)
	if lang == "en" {
		msg = fmt.Sprintf("🎬 Video Summary\n\n%s\n\n📝 Transcript: %d chars (%s)", summary, len([]rune(data.Transcript)), data.Language)
	}

	return map[string]any{
		"success":    true,
		"url":        url,
		"summary":    summary,
		"transcript": data.Transcript,
		"language":   data.Language,
		"duration":   data.Duration,
		"source":     data.Source,
		"operation":  "video_transcript_summary",
	}, msg
}

// extractVideoURL: 메시지에서 영상 URL 추출 (YouTube/Vimeo 등)
func extractVideoURL(s string) string {
	urlRe := regexp.MustCompile(`https?://[^\s]+`)
	matches := urlRe.FindAllString(s, -1)
	for _, u := range matches {
		lower := strings.ToLower(u)
		if strings.Contains(lower, "youtube.com") || strings.Contains(lower, "youtu.be") ||
			strings.Contains(lower, "vimeo.com") || strings.Contains(lower, "tiktok.com") ||
			strings.Contains(lower, "twitch.tv") || strings.Contains(lower, ".mp4") {
			return u
		}
	}
	// URL 없는데 영상 명시 키워드 있는 경우
	return ""
}

// detectVideoMode: 메시지에서 의도 자동 감지
func detectVideoMode(msg string) string {
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "다운로드") || strings.Contains(lower, "download") || strings.Contains(lower, "받아") || strings.Contains(lower, "save"):
		return "download"
	case strings.Contains(lower, "자막") || strings.Contains(lower, "subtitle") || strings.Contains(lower, "transcript") || strings.Contains(lower, "스크립트"):
		return "transcript"
	case strings.Contains(lower, "요약") || strings.Contains(lower, "summary") || strings.Contains(lower, "정리") || strings.Contains(lower, "summarize"):
		return "summary"
	}
	return "summary" // 기본 자막+요약
}

// (min 헬퍼는 다른 파일에 이미 정의됨 — 사용)
