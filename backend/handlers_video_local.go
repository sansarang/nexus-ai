package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// findFFmpeg: PATH → 설치된 경로 → Nexus AppData 순으로 ffmpeg.exe 탐색
func findFFmpeg() string {
	if p, err := exec.LookPath("ffmpeg"); err == nil {
		return p
	}
	candidates := []string{
		`C:\ffmpeg\bin\ffmpeg.exe`,
		`C:\Program Files\ffmpeg\bin\ffmpeg.exe`,
		`C:\tools\ffmpeg\bin\ffmpeg.exe`,
	}
	if appdata := os.Getenv("APPDATA"); appdata != "" {
		candidates = append(candidates, filepath.Join(appdata, "Nexus", "ffmpeg", "bin", "ffmpeg.exe"))
	}
	for _, p := range candidates {
		if fileExists(p) {
			return p
		}
	}
	return ""
}

// findFFprobe: ffmpeg과 같은 디렉토리에서 ffprobe.exe 탐색
func findFFprobe() string {
	if p, err := exec.LookPath("ffprobe"); err == nil {
		return p
	}
	ffmpeg := findFFmpeg()
	if ffmpeg != "" {
		probe := filepath.Join(filepath.Dir(ffmpeg), "ffprobe.exe")
		if fileExists(probe) {
			return probe
		}
	}
	return ""
}

// GET /api/video/check-deps
func handleVideoCheckDeps(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	ffmpegPath, _ := exec.LookPath("ffmpeg")
	if ffmpegPath == "" {
		ffmpegPath = findFFmpeg()
	}
	ytdlpPath := findYtDlp()

	llmMu.RLock()
	groqKey := llmGroqKey
	llmMu.RUnlock()

	result := map[string]any{
		"ffmpeg":      ffmpegPath != "",
		"ffmpeg_path": ffmpegPath,
		"ytdlp":       ytdlpPath != "",
		"ytdlp_path":  ytdlpPath,
		"groq_key":    groqKey != "",
	}

	var missing []string
	if ffmpegPath == "" {
		missing = append(missing, "ffmpeg")
	}
	if groqKey == "" {
		missing = append(missing, msgT("Groq API 키", "Groq API key", lang))
	}

	if len(missing) > 0 {
		result["ready"] = false
		result["message"] = msgT(
			fmt.Sprintf("영상 분석에 필요한 항목이 없습니다: %s", strings.Join(missing, ", ")),
			fmt.Sprintf("Missing required items for video analysis: %s", strings.Join(missing, ", ")),
			lang,
		)
		result["install_hint"] = map[string]string{
			"ffmpeg": msgT("Windows: https://ffmpeg.org/download.html 에서 다운로드 후 C:\\ffmpeg\\bin 에 설치하거나, winget install ffmpeg 또는 choco install ffmpeg 실행", "Windows: Download from https://ffmpeg.org/download.html and install to C:\\ffmpeg\\bin, or run: winget install ffmpeg", lang),
			"groq":   msgT("설정 > API 키에서 Groq API 키를 입력하세요", "Enter your Groq API key in Settings > API Keys", lang),
		}
	} else {
		result["ready"] = true
		result["message"] = msgT("영상 분석 준비 완료", "Ready for video analysis", lang)
	}

	writeJSON(w, 200, result)
}

// POST /api/video/analyze-file
// body: { "file_data": "<base64>", "file_name": "video.mp4", "lang": "ko", "query": "요약해줘" }
// 로컬에 첨부된 영상 파일 → ffmpeg 오디오 추출 → Groq Whisper 전사 → LLM 요약
func handleVideoAnalyzeFile(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		FileData string `json:"file_data"` // base64 (data:video/mp4;base64,... 또는 순수 base64)
		FileName string `json:"file_name"`
		Lang     string `json:"lang"`
		Query    string `json:"query"`
	}
	tryDecodeBody(r, &req)
	if req.FileData == "" || req.FileName == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("file_data와 file_name이 필요합니다", "file_data and file_name are required", lang)})
		return
	}
	if req.Lang == "" {
		req.Lang = "ko"
	}

	// base64 디코딩 (data URL 헤더 제거)
	raw := req.FileData
	if idx := strings.Index(raw, ","); idx >= 0 {
		raw = raw[idx+1:]
	}
	videoBytes, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("base64 디코딩 실패: ", "base64 decode failed: ", lang) + err.Error()})
		return
	}

	// 임시 디렉토리 생성
	tmp, err := os.MkdirTemp("", "nexus_video_*")
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": msgT("임시 디렉토리 생성 실패", "Failed to create temporary directory", lang)})
		return
	}
	defer os.RemoveAll(tmp)

	// 영상 파일 저장
	ext := strings.ToLower(filepath.Ext(req.FileName))
	if ext == "" {
		ext = ".mp4"
	}
	videoPath := filepath.Join(tmp, "input"+ext)
	if err := os.WriteFile(videoPath, videoBytes, 0644); err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": msgT("파일 저장 실패", "File save failed", lang)})
		return
	}

	fileSizeMB := float64(len(videoBytes)) / 1024 / 1024

	// 영상 길이 사전 확인 (10분 초과 시 사용자에게 알림)
	const maxSecs = 600
	videoDurSecs := getVideoDurationSecs(videoPath)
	durationWarning := ""
	if videoDurSecs > maxSecs {
		mins := int(videoDurSecs) / 60
		durationWarning = fmt.Sprintf("\n\n⚠️ **영상이 %d분으로 길어서 처음 10분만 분석합니다.** 전체 분석이 필요하면 영상을 분할해 주세요.", mins)
	}

	// ── Step 1: 내장 자막 추출 시도 (yt-dlp) ─────────────────────
	var transcript string
	if ytdlp := findYtDlp(); ytdlp != "" {
		subArgs := []string{
			"--skip-download",
			"--write-sub", "--write-auto-sub",
			"--sub-langs", req.Lang + ",en",
			"--sub-format", "srt/vtt/best",
			"--convert-subs", "srt",
			"-o", filepath.Join(tmp, "sub"),
			"--no-warnings",
			videoPath,
		}
		subCmd := exec.Command(ytdlp, subArgs...)
		subCmd.Env = append(os.Environ(), "PYTHONIOENCODING=utf-8")
		subCmd.Run()

		srtFiles, _ := filepath.Glob(filepath.Join(tmp, "*.srt"))
		if len(srtFiles) > 0 {
			data, _ := os.ReadFile(srtFiles[0])
			transcript = parseSRT(string(data))
		}
	}

	// ── Step 2: 자막 없으면 ffmpeg → Groq Whisper 전사 ──────────
	method := "subtitle"
	if transcript == "" {
		ffmpegPath := findFFmpeg()

		if ffmpegPath != "" {
			audioPath := filepath.Join(tmp, "audio.mp3")
			ffArgs := []string{
				"-i", videoPath,
				"-vn",                // 비디오 스트림 제거
				"-ar", "16000",       // 16kHz 샘플레이트 (Whisper 최적)
				"-ac", "1",           // 모노
				"-b:a", "64k",        // 비트레이트 (파일 크기 최소화)
				"-t", "600",          // 최대 10분만 처리
				"-y",
				audioPath,
			}
			ffCmd := exec.Command(ffmpegPath, ffArgs...)
			ffCmd.Run()

			if fileExists(audioPath) {
				transcript = groqWhisperTranscribe(audioPath, req.Lang)
				method = "whisper"
			}
		}
	}

	// ── Step 3: 트랜스크립트 → LLM 요약 ─────────────────────────
	if transcript == "" {
		// 오디오 추출도 실패한 경우 → 파일 메타데이터라도 제공
		meta := getVideoMetadata(videoPath)
		msg := fmt.Sprintf("🎬 **%s** (%.1fMB)\n\n자막과 오디오 전사를 추출할 수 없었습니다.\n\n%s\n\n다음 작업은 가능합니다:\n• \"GIF로 만들어줘\" — 영상 → 애니메이션 GIF\n• \"MP4로 변환해줘\" — 포맷 변환\n• \"유튜브 썸네일 크기로 리사이즈\" — 플랫폼 맞춤 크기 조정",
			req.FileName, fileSizeMB, meta)
		writeJSON(w, 200, map[string]any{
			"success":    false,
			"message":    msg,
			"transcript": "",
		})
		return
	}

	transcript = limitStr(transcript, 6000)

	userQ := req.Query
	if userQ == "" {
		if req.Lang == "en" {
			userQ = "Summarize this video content"
		} else {
			userQ = "이 영상 내용을 요약해줘"
		}
	}

	summary, tip := summarizeTranscriptWithQuery(transcript, req.Lang, userQ)

	methodLabel := map[string]string{
		"subtitle": "내장 자막",
		"whisper":  "Whisper AI 전사",
	}[method]

	msg := fmt.Sprintf("🎬 **%s** (%.1fMB) — %s\n\n%s", req.FileName, fileSizeMB, methodLabel, summary)
	if tip != "" && tip != "없음" && !strings.EqualFold(strings.TrimSpace(tip), "none") {
		msg += "\n\n💡 **액션 아이템**\n" + tip
	}
	msg += durationWarning

	writeJSON(w, 200, map[string]any{
		"success":    true,
		"message":    msg,
		"transcript": transcript,
		"summary":    summary,
		"tip":        tip,
		"method":     method,
		"file_name":  req.FileName,
	})
}

// groqWhisperTranscribe: Groq Whisper API로 오디오 전사
func groqWhisperTranscribe(audioPath, lang string) string {
	llmMu.RLock()
	apiKey := llmGroqKey
	llmMu.RUnlock()

	if apiKey == "" {
		return ""
	}

	audioData, err := os.ReadFile(audioPath)
	if err != nil {
		return ""
	}

	// 25MB 제한 (Groq Whisper 제한)
	if len(audioData) > 24*1024*1024 {
		audioData = audioData[:24*1024*1024]
	}

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	// 파일 파트
	fw, err := mw.CreateFormFile("file", filepath.Base(audioPath))
	if err != nil {
		return ""
	}
	if _, err = fw.Write(audioData); err != nil {
		return ""
	}

	mw.WriteField("model", "whisper-large-v3-turbo")
	mw.WriteField("response_format", "text")
	if lang != "" && lang != "auto" {
		// ISO 639-1 코드 변환
		isoLang := lang
		if lang == "ko" {
			isoLang = "ko"
		} else if lang == "en" {
			isoLang = "en"
		}
		mw.WriteField("language", isoLang)
	}
	mw.Close()

	client := &http.Client{Timeout: 120 * time.Second}
	req, err := http.NewRequest("POST", "https://api.groq.com/openai/v1/audio/transcriptions", &buf)
	if err != nil {
		return ""
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	// response_format=text이면 순수 텍스트 반환
	return strings.TrimSpace(string(body))
}

// summarizeTranscriptWithQuery: 사용자 질문을 반영한 요약
func summarizeTranscriptWithQuery(transcript, lang, query string) (summary, tip string) {
	langInstr := "한국어"
	if lang == "en" {
		langInstr = "영어"
	}

	prompt := fmt.Sprintf(`다음은 영상 자막/전사 내용입니다.

사용자 질문: "%s"

자막:
%s

위 자막을 기반으로 사용자 질문에 %s로 직접 답해줘.
답변 형식:
• 핵심 답변 (1~2줄)
• 주요 내용 3~5개 (불릿 포인트)
• 결론 또는 인사이트`, query, transcript, langInstr)

	summary, _, _ = callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 700, false)

	tipPrompt := fmt.Sprintf("위 내용에서 실행 가능한 팁이나 핵심 포인트 3개를 %s로 한 줄씩 뽑아줘. 없으면 '없음'이라고 해.", langInstr)
	tip, _, _ = callGroqWithFallback([]groqMsg{
		{Role: "user", Content: prompt},
		{Role: "assistant", Content: summary},
		{Role: "user", Content: tipPrompt},
	}, 300, false)
	return
}

// getVideoDurationSecs: ffprobe로 영상 길이(초) 반환. 실패 시 0
func getVideoDurationSecs(videoPath string) float64 {
	probe := findFFprobe()
	if probe == "" {
		return 0
	}
	for _, probe := range []string{probe} {
		cmd := exec.Command(probe,
			"-v", "quiet", "-print_format", "json",
			"-show_format", videoPath,
		)
		out, err := cmd.Output()
		if err != nil {
			continue
		}
		var meta map[string]any
		if json.Unmarshal(out, &meta) != nil {
			continue
		}
		format, _ := meta["format"].(map[string]any)
		if d, ok := format["duration"].(string); ok {
			var secs float64
			fmt.Sscanf(d, "%f", &secs)
			return secs
		}
	}
	return 0
}

// getVideoMetadata: ffprobe로 영상 메타데이터 추출
func getVideoMetadata(videoPath string) string {
	probePath := findFFprobe()
	if probePath == "" {
		return ""
	}
	for _, probe := range []string{probePath} {
		if _, err := exec.LookPath(probe); err == nil || fileExists(probe) {
			cmd := exec.Command(probe,
				"-v", "quiet",
				"-print_format", "json",
				"-show_format",
				"-show_streams",
				videoPath,
			)
			out, err := cmd.Output()
			if err == nil && len(out) > 0 {
				var meta map[string]any
				if json.Unmarshal(out, &meta) == nil {
					format, _ := meta["format"].(map[string]any)
					duration := ""
					if d, ok := format["duration"].(string); ok {
						secs := 0.0
						fmt.Sscanf(d, "%f", &secs)
						mins := int(secs) / 60
						duration = fmt.Sprintf("%d분 %d초", mins, int(secs)%60)
					}
					if duration != "" {
						return fmt.Sprintf("📊 영상 길이: %s", duration)
					}
				}
			}
			break
		}
	}
	return ""
}
