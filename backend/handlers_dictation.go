//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"unicode/utf8"
)

// ══════════════════════════════════════════════════════════════════
//  Voice Dictation — PowerShell SendKeys로 활성 창에 텍스트 입력
// ══════════════════════════════════════════════════════════════════

// POST /api/dictation/type — body: {text: string, app?: string}
func handleDictationType(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		Text string `json:"text"`
		App  string `json:"app,omitempty"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.Text == "" {
		json200(w, map[string]interface{}{"success": false, "message": msgT("text가 필요해요", "text is required", lang)})
		return
	}

	// 특수문자 이스케이프 (SendKeys 문법)
	escaped := escapeSendKeys(req.Text)

	script := ""
	if req.App != "" {
		script += "Start-Process '" + req.App + "'\n"
		script += "Start-Sleep -Milliseconds 800\n"
	}
	script += "Add-Type -AssemblyName System.Windows.Forms\n"
	script += "[System.Windows.Forms.SendKeys]::SendWait('" + escaped + "')"

	out, err := exec.Command("powershell", "-NoProfile", "-Command", script).CombinedOutput()
	if err != nil {
		json200(w, map[string]interface{}{
			"success": false,
			"message": msgT("입력 실패: ", "Input failed: ", lang) + err.Error() + " " + string(out),
		})
		return
	}

	json200(w, map[string]interface{}{
		"success":     true,
		"typed_chars": utf8.RuneCountInString(req.Text),
		"message":     fmt.Sprintf(msgT("%d자 입력 완료", "%d characters typed", lang), utf8.RuneCountInString(req.Text)),
	})
}

// POST /api/dictation/paste — body: {text: string}
func handleDictationPaste(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		Text string `json:"text"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.Text == "" {
		json200(w, map[string]interface{}{"success": false, "message": msgT("text가 필요해요", "text is required", lang)})
		return
	}

	// 텍스트를 클립보드에 복사 후 Ctrl+V
	// PowerShell에서 여러 줄 텍스트도 처리
	escaped := strings.ReplaceAll(req.Text, "'", "''") // single-quote 이스케이프
	script := "Set-Clipboard -Value '" + escaped + "'\n" +
		"Add-Type -AssemblyName System.Windows.Forms\n" +
		"[System.Windows.Forms.SendKeys]::SendWait('^v')"

	out, err := exec.Command("powershell", "-NoProfile", "-Command", script).CombinedOutput()
	if err != nil {
		json200(w, map[string]interface{}{
			"success": false,
			"message": msgT("붙여넣기 실패: ", "Paste failed: ", lang) + err.Error() + " " + string(out),
		})
		return
	}

	json200(w, map[string]interface{}{
		"success": true,
		"message": msgT("클립보드 붙여넣기 완료", "Clipboard paste complete", lang),
	})
}

// SendKeys 특수문자 이스케이프
// +, ^, %, ~, (, ), [, ], {, } 는 중괄호로 감싸야 함
func escapeSendKeys(text string) string {
	special := map[rune]string{
		'+': "{+}",
		'^': "{^}",
		'%': "{%}",
		'~': "{~}",
		'(': "{(}",
		')': "{)}",
		'[': "{[}",
		']': "{]}",
		'{': "{{}",
		'}': "{}}",
		'\'': "''",
	}
	var sb strings.Builder
	for _, ch := range text {
		if esc, ok := special[ch]; ok {
			sb.WriteString(esc)
		} else {
			sb.WriteRune(ch)
		}
	}
	return sb.String()
}
