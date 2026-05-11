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
	meetingMu.Lock()
	defer meetingMu.Unlock()

	ts := time.Now().Format("20060102_150405")
	fp := filepath.Join(meetingsDir(), "meeting_"+ts+".wav")

	var cmd *exec.Cmd
	_, err := exec.LookPath("ffmpeg")
	if err == nil {
		cmd = exec.Command("ffmpeg", "-y", "-f", "dshow",
			"-i", "audio=@device_cm_{33D9A762-90C8-11D0-BD43-00A0C911CE86}\\wave_{00000000-0000-0000-0000-000000000000}",
			fp)
	} else {
		cmd = exec.Command("powershell", "-NoProfile", "-Command",
			"Start-Process -FilePath SoundRecorder -ArgumentList '/file', '"+fp+"' -WindowStyle Hidden")
	}

	if err2 := cmd.Start(); err2 != nil {
		json200(w, map[string]interface{}{"success": false, "message": "녹음 시작 실패: " + err2.Error()})
		return
	}

	meetingProc = cmd
	meetingFilePath = fp
	meetingStart = time.Now()

	json200(w, map[string]interface{}{
		"success":   true,
		"file_path": fp,
		"message":   "녹음을 시작했어요",
	})
}

// POST /api/meeting/stop
func handleMeetingStop(w http.ResponseWriter, r *http.Request) {
	meetingMu.Lock()
	defer meetingMu.Unlock()

	if meetingProc == nil {
		json200(w, map[string]interface{}{"success": false, "message": "진행 중인 녹음이 없어요"})
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
		"message":      fmt.Sprintf("녹음 완료 (%.0f초)", duration),
	})
}

// POST /api/meeting/transcribe — 오디오 전사 (현재 미지원)
func handleMeetingTranscribe(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]interface{}{
		"success": false,
		"message": "오디오 전사 기능은 현재 지원되지 않습니다.",
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
	var req struct {
		Text string `json:"text"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.Text == "" {
		json200(w, map[string]interface{}{"success": false, "message": "text가 필요해요"})
		return
	}

	systemPrompt := "당신은 회의록 요약 전문가입니다. 다음 회의 내용을 분석하여 JSON으로만 응답하세요:\n" +
		"{\"summary\": \"회의 전체 요약 (3-5문장)\", \"action_items\": [\"실행 항목\"], \"decisions\": [\"결정 사항\"]}"

	msgs := []groqMsg{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: req.Text},
	}
	contentStr, _, err := callGroqWithFallback(msgs, 1024, true)
	if err != nil {
		json200(w, map[string]interface{}{"success": false, "message": "AI 호출 실패: " + err.Error()})
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
		"message":      "회의 요약 완료",
	})
}
