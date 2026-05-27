//go:build windows

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ── Live Caption (실시간 자막 + 번역) ─────────────────────────
// ffmpeg으로 시스템 오디오 캡처 → Groq Whisper 전사 → SSE 전송

type CaptionEntry struct {
	Text      string `json:"text"`
	Translated string `json:"translated,omitempty"`
	Timestamp string `json:"timestamp"`
	Lang      string `json:"lang"`
}

var (
	captionMu      sync.RWMutex
	captionRunning bool
	captionCmd     *exec.Cmd
	captionBuffer  []CaptionEntry
	captionCh      = make(chan CaptionEntry, 64)
	captionLang    = "ko" // 타깃 번역 언어
)

func captionDir() string {
	return filepath.Join(os.Getenv("APPDATA"), "Nexus", "caption")
}

func handleCaptionStart(w http.ResponseWriter, r *http.Request) {
	captionMu.Lock()
	if captionRunning {
		captionMu.Unlock()
		json200(w, map[string]any{"ok": false, "message": msgT("이미 자막이 실행 중입니다.", "Caption is already running.", getLang(r))})
		return
	}
	captionRunning = true
	captionBuffer = nil

	var req struct {
		Lang string `json:"lang"` // 번역 대상 언어 (ko, en, ja ...)
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Lang != "" {
		captionLang = req.Lang
	}
	captionMu.Unlock()

	go runCaptionLoop()

	json200(w, map[string]any{"ok": true, "message": msgT("🎙️ 실시간 자막을 시작했습니다.", "🎙️ Live caption started.", getLang(r)), "target_lang": captionLang})
}

func handleCaptionStop(w http.ResponseWriter, r *http.Request) {
	captionMu.Lock()
	defer captionMu.Unlock()
	if !captionRunning {
		json200(w, map[string]any{"ok": false, "message": msgT("자막이 실행 중이 아닙니다.", "Caption is not running.", getLang(r))})
		return
	}
	captionRunning = false
	if captionCmd != nil && captionCmd.Process != nil {
		captionCmd.Process.Kill()
		captionCmd = nil
	}
	json200(w, map[string]any{"ok": true, "message": msgT("자막을 종료했습니다.", "Caption stopped.", getLang(r)), "entries": len(captionBuffer)})
}

func handleCaptionStream(w http.ResponseWriter, r *http.Request) {
	// SSE 헤더
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	// 클라이언트 연결 유지
	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case entry, open := <-captionCh:
			if !open {
				return
			}
			data, _ := json.Marshal(entry)
			fmt.Fprintf(w, "data: %s\n\n", string(data))
			flusher.Flush()
		case <-time.After(5 * time.Second):
			// 연결 유지용 heartbeat
			fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}

func handleCaptionLatest(w http.ResponseWriter, r *http.Request) {
	captionMu.RLock()
	defer captionMu.RUnlock()

	limit := 20
	entries := captionBuffer
	if len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}
	json200(w, map[string]any{
		"entries":  entries,
		"running":  captionRunning,
		"total":    len(captionBuffer),
	})
}

// ── 캡처 루프 ─────────────────────────────────────────────────

func runCaptionLoop() {
	os.MkdirAll(captionDir(), 0755)
	chunkSec := 5 // 5초 청크
	ticker := time.NewTicker(time.Duration(chunkSec) * time.Second)
	defer ticker.Stop()

	for {
		captionMu.RLock()
		running := captionRunning
		captionMu.RUnlock()
		if !running {
			return
		}

		chunkPath := filepath.Join(captionDir(), fmt.Sprintf("chunk_%d.wav", time.Now().UnixMilli()))

		// ffmpeg으로 시스템 오디오 캡처 (Windows WASAPI 루프백)
		cmd := newHiddenCmd("ffmpeg",
			"-f", "dshow",
			"-i", "audio=CABLE Output (VB-Audio Virtual Cable)", // VB-Cable 없으면 실제 기기로 대체
			"-t", fmt.Sprintf("%d", chunkSec),
			"-ar", "16000",
			"-ac", "1",
			"-y",
			chunkPath,
		)

		// VB-Cable이 없을 경우 기본 스테레오 믹스 시도
		if err := cmd.Run(); err != nil {
			cmd2 := newHiddenCmd("ffmpeg",
				"-f", "dshow",
				"-i", "audio=Stereo Mix (Realtek Audio)",
				"-t", fmt.Sprintf("%d", chunkSec),
				"-ar", "16000", "-ac", "1", "-y",
				chunkPath,
			)
			if err2 := cmd2.Run(); err2 != nil {
				// 캡처 실패 — 5초 후 재시도
				<-ticker.C
				continue
			}
		}

		// Whisper 전사
		text, err := transcribeWithWhisper(chunkPath)
		os.Remove(chunkPath)
		if err != nil || strings.TrimSpace(text) == "" {
			<-ticker.C
			continue
		}

		// 번역 (타깃 언어가 감지 언어와 다를 때)
		translated := ""
		if captionLang != "" && captionLang != "auto" {
			translated = translateCaption(text, captionLang)
		}

		entry := CaptionEntry{
			Text:       text,
			Translated: translated,
			Timestamp:  time.Now().Format("15:04:05"),
			Lang:       captionLang,
		}

		captionMu.Lock()
		captionBuffer = append(captionBuffer, entry)
		// 최대 200개 유지
		if len(captionBuffer) > 200 {
			captionBuffer = captionBuffer[1:]
		}
		captionMu.Unlock()

		// SSE 채널에 전송 (비블로킹)
		select {
		case captionCh <- entry:
		default:
		}

		<-ticker.C
	}
}

func transcribeWithWhisper(filePath string) (string, error) {
	openaiKey := os.Getenv("OPENAI_API_KEY")
	if openaiKey == "" {
		return "", fmt.Errorf("OpenAI API 키가 설정되지 않았습니다 (OPENAI_API_KEY 환경변수 필요)")
	}

	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("오디오 파일 열기 실패: %w", err)
	}
	defer f.Close()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return "", err
	}
	if _, err = io.Copy(fw, f); err != nil {
		return "", err
	}
	mw.WriteField("model", "whisper-1")
	mw.WriteField("language", "ko")
	mw.Close()

	req, _ := http.NewRequest("POST", "https://api.openai.com/v1/audio/transcriptions", &buf)
	req.Header.Set("Authorization", "Bearer "+openaiKey)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("Whisper 요청 실패: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("Whisper 응답 파싱 실패")
	}
	return result.Text, nil
}

func translateCaption(text, targetLang string) string {
	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()
	if gKey == "" {
		return ""
	}

	langName := map[string]string{
		"ko": "한국어", "en": "영어", "ja": "일본어",
		"zh": "중국어", "es": "스페인어", "fr": "프랑스어",
	}
	target := langName[targetLang]
	if target == "" {
		target = targetLang
	}

	prompt := fmt.Sprintf(`다음 텍스트를 %s로 번역하세요. 번역문만 반환:
"%s"`, target, text)

	result, _, _ := callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 256, false)
	return strings.TrimSpace(result)
}
