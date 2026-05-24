//go:build windows

package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// captureScreenWindows: PowerShell로 화면 캡처 → base64 PNG
func captureScreenWindows() (string, error) {
	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("nexus_screen_%d.png", time.Now().UnixMilli()))
	ps := fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
$screen = [System.Windows.Forms.Screen]::PrimaryScreen.Bounds
$bmp = New-Object System.Drawing.Bitmap($screen.Width, $screen.Height)
$g = [System.Drawing.Graphics]::FromImage($bmp)
$g.CopyFromScreen($screen.Location, [System.Drawing.Point]::Empty, $screen.Size)
$g.Dispose()
$bmp.Save('%s', [System.Drawing.Imaging.ImageFormat]::Png)
$bmp.Dispose()
`, tmp)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", ps)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("화면 캡처 실패: %w — %s", err, string(out))
	}
	data, err := os.ReadFile(tmp)
	os.Remove(tmp)
	if err != nil {
		return "", fmt.Errorf("캡처 파일 읽기 실패: %w", err)
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func analyzeImageWithGroqVision(b64img, question, lang string) (string, error) {
	// 1순위: Supabase Edge Function 프록시
	if content, err := callVisionViaProxy(b64img, question, lang); err == nil {
		return content, nil
	}
	// 2순위: 로컬 Groq 키 직접 호출
	llmMu.RLock()
	gKey := llmGroqKey
	llmMu.RUnlock()
	if gKey == "" {
		return "", fmt.Errorf("Groq API 키가 설정되지 않았습니다")
	}
	if question == "" {
		if lang == "en" {
			question = "What is on this screen? Describe the main content, text, apps, and notable elements."
		} else {
			question = "이 화면에 무엇이 있나요? 주요 내용, 텍스트, 앱, 눈에 띄는 요소를 설명해주세요."
		}
	}
	systemMsg := "화면을 분석하는 AI 비서입니다. 화면에 보이는 내용을 명확하고 자세하게 설명해주세요."
	if lang == "en" {
		systemMsg = "You are an AI assistant analyzing screen captures. Be clear and detailed."
	}
	body, _ := json.Marshal(map[string]any{
		"model": groqVisionModel, "max_tokens": 1024,
		"messages": []map[string]any{
			{"role": "system", "content": systemMsg},
			{"role": "user", "content": []map[string]any{
				{"type": "text", "text": question},
				{"type": "image_url", "image_url": map[string]string{
					"url": "data:image/png;base64," + b64img,
				}},
			}},
		},
	})
	client := &http.Client{Timeout: 30 * time.Second}
	req, _ := http.NewRequest("POST", groqRealAPIBase, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+gKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Vision API 연결 실패: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	var gr struct {
		Choices []struct {
			Message struct{ Content string `json:"content"` } `json:"message"`
		} `json:"choices"`
		Error *struct{ Message string `json:"message"` } `json:"error"`
	}
	if json.Unmarshal(raw, &gr) != nil {
		return "", fmt.Errorf("Vision 응답 파싱 실패")
	}
	if gr.Error != nil {
		return "", fmt.Errorf("Vision 오류: %s", gr.Error.Message)
	}
	if len(gr.Choices) == 0 {
		return "", fmt.Errorf("Vision 응답 없음")
	}
	return gr.Choices[0].Message.Content, nil
}

func analyzeImageWithClaude(b64img, question, lang string) (string, error) {
	llmMu.RLock()
	cKey := llmClaudeKey
	llmMu.RUnlock()
	if cKey == "" {
		return analyzeImageWithGroqVision(b64img, question, lang)
	}
	if question == "" {
		if lang == "en" {
			question = "What is on this screen? Describe the main content and notable elements."
		} else {
			question = "이 화면에 무엇이 있나요? 주요 내용을 설명해주세요."
		}
	}
	systemMsg := "화면 캡처를 분석하는 AI 비서입니다."
	if lang == "en" {
		systemMsg = "You are an AI assistant analyzing screen captures."
	}
	body, _ := json.Marshal(map[string]any{
		"model": claudeModel, "max_tokens": 1024, "system": systemMsg,
		"messages": []map[string]any{
			{"role": "user", "content": []map[string]any{
				{"type": "image", "source": map[string]any{
					"type": "base64", "media_type": "image/png", "data": b64img,
				}},
				{"type": "text", "text": question},
			}},
		},
	})
	client := &http.Client{Timeout: 30 * time.Second}
	req, _ := http.NewRequest("POST", claudeAPIBase, bytes.NewReader(body))
	req.Header.Set("x-api-key", cKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return analyzeImageWithGroqVision(b64img, question, lang)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	var cr struct {
		Content []struct{ Text string `json:"text"` } `json:"content"`
		Error   *struct{ Message string `json:"message"` } `json:"error"`
	}
	if json.Unmarshal(raw, &cr) != nil || cr.Error != nil || len(cr.Content) == 0 {
		return analyzeImageWithGroqVision(b64img, question, lang)
	}
	return cr.Content[0].Text, nil
}

// POST /api/screenshot/analyze
func handleScreenshotAnalyze(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Question string `json:"question"`
		Lang     string `json:"lang"`
		Provider string `json:"provider"` // "groq" | "claude" | "auto"
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Lang == "" {
		req.Lang = GetUserLang()
	}
	b64, err := captureScreenWindows()
	if err != nil {
		msg := "화면 캡처에 실패했습니다: " + err.Error()
		if req.Lang == "en" {
			msg = "Screen capture failed: " + err.Error()
		}
		writeJSON(w, 500, map[string]any{"success": false, "message": msg})
		return
	}
	var analysis string
	if req.Provider == "groq" {
		analysis, err = analyzeImageWithGroqVision(b64, req.Question, req.Lang)
	} else {
		analysis, err = analyzeImageWithClaude(b64, req.Question, req.Lang)
	}
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}
	json200(w, map[string]any{"success": true, "analysis": analysis, "message": analysis})
}

// POST /api/screenshot/translate
func handleScreenshotTranslate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TargetLang string `json:"target_lang"`
		Lang       string `json:"lang"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Lang == "" {
		req.Lang = GetUserLang()
	}
	if req.TargetLang == "" {
		if req.Lang == "en" {
			req.TargetLang = "ko"
		} else {
			req.TargetLang = "en"
		}
	}
	b64, err := captureScreenWindows()
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": msgT("캡처 실패: "+err.Error(), "Capture failed: "+err.Error(), getLang(r))})
		return
	}
	q := "이 화면의 모든 텍스트를 한국어로 번역해주세요. 원문 → 번역 순서로 보여주세요."
	if req.TargetLang == "en" {
		q = "Translate all text on this screen to English. Show: original → translation."
	}
	analysis, err := analyzeImageWithClaude(b64, q, req.Lang)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}
	json200(w, map[string]any{"success": true, "translation": analysis, "message": analysis})
}
