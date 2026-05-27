//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ══════════════════════════════════════════════════════════════════
//  Meeting Assistant — 녹음 + Whisper 전사 + 요약
// ══════════════════════════════════════════════════════════════════

var (
	meetingMu       sync.Mutex
	meetingProc     *exec.Cmd
	meetingFilePath string
	meetingStart    time.Time
)

func meetingsDir() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = os.TempDir()
	}
	dir := filepath.Join(appData, "Nexus", "meetings")
	os.MkdirAll(dir, 0755)
	return dir
}

// POST /api/meeting/start
func handleMeetingStart(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	meetingMu.Lock()
	defer meetingMu.Unlock()

	ts := time.Now().Format("20060102_150405")
	fp := filepath.Join(meetingsDir(), "meeting_"+ts+".wav")

	var cmd *exec.Cmd
	_, err := exec.LookPath("ffmpeg")
	if err == nil {
		cmd = newHiddenCmd("ffmpeg", "-y", "-f", "dshow",
			"-i", "audio=@device_cm_{33D9A762-90C8-11D0-BD43-00A0C911CE86}\\wave_{00000000-0000-0000-0000-000000000000}",
			fp)
	} else {
		cmd = newHiddenCmd("powershell", "-NoProfile", "-Command",
			"Start-Process -FilePath SoundRecorder -ArgumentList '/file', '"+fp+"' -WindowStyle Hidden")
	}

	if err2 := cmd.Start(); err2 != nil {
		json200(w, map[string]interface{}{"success": false, "message": msgT("녹음 시작 실패: ", "Recording start failed: ", lang) + err2.Error()})
		return
	}

	meetingProc = cmd
	meetingFilePath = fp
	meetingStart = time.Now()

	json200(w, map[string]interface{}{
		"success":   true,
		"file_path": fp,
		"message":   msgT("녹음을 시작했어요", "Recording started", lang),
	})
}

// POST /api/meeting/stop
func handleMeetingStop(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	meetingMu.Lock()
	defer meetingMu.Unlock()

	if meetingProc == nil {
		json200(w, map[string]interface{}{"success": false, "message": msgT("진행 중인 녹음이 없어요", "No recording in progress", lang)})
		return
	}

	duration := time.Since(meetingStart).Seconds()
	fp := meetingFilePath

	_ = meetingProc.Process.Kill()
	meetingProc = nil
	meetingFilePath = ""

	json200(w, map[string]interface{}{
		"success":      true,
		"file_path":    fp,
		"duration_sec": int(duration),
		"message":      fmt.Sprintf(msgT("녹음 완료 (%.0f초)", "Recording complete (%.0fs)", lang), duration),
	})
}

// POST /api/meeting/transcribe — Groq Whisper로 오디오 전사
func handleMeetingTranscribe(w http.ResponseWriter, r *http.Request) {
	uiLang := getLang(r)
	var req struct {
		MeetingID string `json:"meeting_id"` // 파일명 (확장자 제외)
		FilePath  string `json:"file_path"`  // 직접 경로 지정 시
		Lang      string `json:"lang"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	// Groq API 키 확인
	llmMu.RLock()
	groqKey := llmGroqKey
	if groqKey == "" {
		groqKey = llmPerplexityKey
	}
	llmMu.RUnlock()
	if groqKey == "" || !strings.HasPrefix(groqKey, "gsk_") {
		json200(w, map[string]interface{}{
			"success": false,
			"message": msgT("Groq API 키가 필요합니다. 설정 > API 키에서 Groq 키(gsk_...)를 입력해주세요.", "Groq API key required. Please enter your Groq key (gsk_...) in Settings > API Keys.", uiLang),
		})
		return
	}

	// 파일 경로 결정
	audioPath := req.FilePath
	if audioPath == "" && req.MeetingID != "" {
		// meetings 디렉터리에서 meeting_id로 검색
		dir := meetingsDir()
		for _, ext := range []string{".wav", ".mp3", ".m4a", ".ogg"} {
			candidate := filepath.Join(dir, req.MeetingID+ext)
			if _, err := os.Stat(candidate); err == nil {
				audioPath = candidate
				break
			}
		}
	}
	if audioPath == "" {
		json200(w, map[string]interface{}{
			"success": false,
			"message": msgT("오디오 파일을 찾을 수 없습니다. meeting_id 또는 file_path를 지정해주세요.", "Audio file not found. Please specify meeting_id or file_path.", uiLang),
		})
		return
	}
	if _, err := os.Stat(audioPath); err != nil {
		json200(w, map[string]interface{}{"success": false, "message": msgT("파일이 존재하지 않습니다: ", "File does not exist: ", uiLang) + audioPath})
		return
	}

	lang := req.Lang
	if lang == "" {
		lang = GetUserLang()
	}

	// 전사 실행
	transcript := groqWhisperTranscribe(audioPath, lang)
	if transcript == "" {
		json200(w, map[string]interface{}{
			"success": false,
			"message": msgT("전사 실패: Groq Whisper API 오류이거나 오디오에 음성이 없을 수 있습니다.", "Transcription failed: Groq Whisper API error or no speech in audio.", uiLang),
		})
		return
	}

	json200(w, map[string]interface{}{
		"success":    true,
		"transcript": transcript,
		"message":    msgT("전사 완료", "Transcription complete", uiLang),
		"lang":       lang,
	})
}

// GET /api/meeting/list
func handleMeetingList(w http.ResponseWriter, r *http.Request) {
	dir := meetingsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		json200(w, map[string]interface{}{"success": false, "meetings": []interface{}{}, "total": 0})
		return
	}

	type MeetingFile struct {
		File      string  `json:"file"`
		Timestamp string  `json:"timestamp"`
		SizeMB    float64 `json:"size_mb"`
	}

	var meetings []MeetingFile
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".wav") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		meetings = append(meetings, MeetingFile{
			File:      filepath.Join(dir, e.Name()),
			Timestamp: info.ModTime().Format(time.RFC3339),
			SizeMB:    float64(info.Size()) / 1024 / 1024,
		})
	}

	json200(w, map[string]interface{}{
		"success":  true,
		"meetings": meetings,
		"total":    len(meetings),
	})
}

// POST /api/meeting/summarize — body: {text, groq_key}
func handleMeetingSummarize(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		Text string `json:"text"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.Text == "" {
		json200(w, map[string]interface{}{"success": false, "message": msgT("text가 필요해요", "text is required", lang)})
		return
	}

	var systemPrompt string
	if isEnglishQuery(req.Text) {
		systemPrompt = "You are a meeting notes expert. Analyze the following meeting content and respond ONLY in JSON:\n" +
			"{\"summary\": \"Overall meeting summary (3-5 sentences)\", \"action_items\": [\"action item\"], \"decisions\": [\"decision made\"]}"
	} else {
		systemPrompt = "당신은 회의록 요약 전문가입니다. 다음 회의 내용을 분석하여 JSON으로만 응답하세요:\n" +
			"{\"summary\": \"회의 전체 요약 (3-5문장)\", \"action_items\": [\"실행 항목\"], \"decisions\": [\"결정 사항\"]}"
	}

	msgs := []groqMsg{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: req.Text},
	}
	contentStr, _, err := callGroqWithFallback(msgs, 1024, true)
	if err != nil {
		json200(w, map[string]interface{}{"success": false, "message": msgT("AI 호출 실패: ", "AI call failed: ", lang) + err.Error()})
		return
	}
	content := contentStr

	content = strings.TrimSpace(content)
	if idx := strings.Index(content, "```json"); idx >= 0 {
		content = content[idx+7:]
	} else if idx := strings.Index(content, "```"); idx >= 0 {
		content = content[idx+3:]
	}
	if idx := strings.LastIndex(content, "```"); idx >= 0 {
		content = content[:idx]
	}
	content = strings.TrimSpace(content)

	var parsed struct {
		Summary     string   `json:"summary"`
		ActionItems []string `json:"action_items"`
		Decisions   []string `json:"decisions"`
	}
	json.Unmarshal([]byte(content), &parsed)

	if parsed.ActionItems == nil {
		parsed.ActionItems = []string{}
	}
	if parsed.Decisions == nil {
		parsed.Decisions = []string{}
	}

	json200(w, map[string]interface{}{
		"success":      true,
		"summary":      parsed.Summary,
		"action_items": parsed.ActionItems,
		"decisions":    parsed.Decisions,
		"message":      msgT("회의 요약 완료", "Meeting summary complete", lang),
	})
}
