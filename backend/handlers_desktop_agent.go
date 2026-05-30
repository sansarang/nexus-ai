//go:build windows

package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

// ══════════════════════════════════════════════════════════════════
//  Desktop Computer Use Agent
//  화면을 보고 마우스·키보드로 어떤 앱이든 제어
//  Claude Computer Use / OpenAI Operator 방식
// ══════════════════════════════════════════════════════════════════

// ── Windows API 로드 ────────────────────────────────────────────

var (
	user32dll        = syscall.NewLazyDLL("user32.dll")
	procSetCursor    = user32dll.NewProc("SetCursorPos")
	procSendInput    = user32dll.NewProc("SendInput")
	procMouseEvent   = user32dll.NewProc("mouse_event")
	procGetCursorPos = user32dll.NewProc("GetCursorPos")
	procFindWindow   = user32dll.NewProc("FindWindowW")
	procShowWindow   = user32dll.NewProc("ShowWindow")
	procSetForeground = user32dll.NewProc("SetForegroundWindow")
)

// INPUT 구조체 (Windows SendInput용)
type POINT struct{ X, Y int32 }

type MOUSEINPUT struct {
	Dx, Dy        int32
	MouseData     uint32
	DwFlags       uint32
	Time          uint32
	DwExtraInfo   uintptr
}

type KEYBDINPUT struct {
	WVk         uint16
	WScan       uint16
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
	_           [8]byte
}

type INPUT struct {
	Type uint32
	Mi   MOUSEINPUT
	_    [8]byte
}

const (
	INPUT_MOUSE    = 0
	INPUT_KEYBOARD = 1
	MOUSEEVENTF_MOVE        = 0x0001
	MOUSEEVENTF_LEFTDOWN    = 0x0002
	MOUSEEVENTF_LEFTUP      = 0x0004
	MOUSEEVENTF_RIGHTDOWN   = 0x0008
	MOUSEEVENTF_RIGHTUP     = 0x0010
	MOUSEEVENTF_ABSOLUTE    = 0x8000
	KEYEVENTF_KEYUP         = 0x0002
	KEYEVENTF_UNICODE       = 0x0004
)

// ── 마우스 제어 ─────────────────────────────────────────────────

// moveMouse: 절대 좌표로 마우스 이동
func moveMouse(x, y int) error {
	ret, _, err := procSetCursor.Call(uintptr(x), uintptr(y))
	if ret == 0 {
		return fmt.Errorf("마우스 이동 실패: %v", err)
	}
	return nil
}

// clickAt: 지정 좌표 클릭 (left/right)
func clickAt(x, y int, button string) error {
	if err := moveMouse(x, y); err != nil {
		return err
	}
	time.Sleep(80 * time.Millisecond)

	script := fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
[System.Windows.Forms.Cursor]::Position = New-Object System.Drawing.Point(%d, %d)
`, x, y)

	if button == "right" {
		script += `
[System.Windows.Forms.Application]::DoEvents()
Add-Type @"
using System;
using System.Runtime.InteropServices;
public class MouseClick {
    [DllImport("user32.dll")] public static extern void mouse_event(uint dwFlags, int dx, int dy, uint cButtons, int dwExtraInfo);
}
"@
[MouseClick]::mouse_event(0x0008, 0, 0, 0, 0)
Start-Sleep -Milliseconds 50
[MouseClick]::mouse_event(0x0010, 0, 0, 0, 0)
`
	} else {
		script += `
Add-Type @"
using System;
using System.Runtime.InteropServices;
public class MouseClick {
    [DllImport("user32.dll")] public static extern void mouse_event(uint dwFlags, int dx, int dy, uint cButtons, int dwExtraInfo);
}
"@
[MouseClick]::mouse_event(0x0002, 0, 0, 0, 0)
Start-Sleep -Milliseconds 50
[MouseClick]::mouse_event(0x0004, 0, 0, 0, 0)
`
	}

	return execPSRun(script)
}

// doubleClick: 더블클릭
func doubleClick(x, y int) error {
	if err := clickAt(x, y, "left"); err != nil {
		return err
	}
	time.Sleep(100 * time.Millisecond)
	return clickAt(x, y, "left")
}

// dragTo: 드래그
func dragTo(fromX, fromY, toX, toY int) error {
	script := fmt.Sprintf(`
Add-Type @"
using System;
using System.Runtime.InteropServices;
public class MouseDrag {
    [DllImport("user32.dll")] public static extern void mouse_event(uint dwFlags, int dx, int dy, uint cButtons, int dwExtraInfo);
    [DllImport("user32.dll")] public static extern bool SetCursorPos(int x, int y);
}
"@
[MouseDrag]::SetCursorPos(%d, %d)
Start-Sleep -Milliseconds 50
[MouseDrag]::mouse_event(0x0002, 0, 0, 0, 0)
Start-Sleep -Milliseconds 50
[MouseDrag]::SetCursorPos(%d, %d)
Start-Sleep -Milliseconds 50
[MouseDrag]::mouse_event(0x0004, 0, 0, 0, 0)
`, fromX, fromY, toX, toY)
	return execPSRun(script)
}

// scrollAt: 스크롤 (delta: 양수=위, 음수=아래)
func scrollAt(x, y, delta int) error {
	script := fmt.Sprintf(`
Add-Type @"
using System;
using System.Runtime.InteropServices;
public class MouseScroll {
    [DllImport("user32.dll")] public static extern void mouse_event(uint dwFlags, int dx, int dy, uint cButtons, int dwExtraInfo);
    [DllImport("user32.dll")] public static extern bool SetCursorPos(int x, int y);
}
"@
[MouseScroll]::SetCursorPos(%d, %d)
[MouseScroll]::mouse_event(0x0800, 0, 0, %d, 0)
`, x, y, delta*120)
	return execPSRun(script)
}

// ── 키보드 제어 ─────────────────────────────────────────────────

// typeText: 텍스트 입력 (클립보드 경유 — 한글/특수문자 안전)
func typeText(text string) error {
	escaped := strings.ReplaceAll(text, `"`, "`\"")
	escaped = strings.ReplaceAll(escaped, "`", "``")
	script := fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
[System.Windows.Forms.Clipboard]::SetText("%s")
Start-Sleep -Milliseconds 100
[System.Windows.Forms.SendKeys]::SendWait("^v")
`, escaped)
	return execPSRun(script)
}

// pressKey: 단일 키 또는 단축키 입력
// key 예시: "Enter", "Tab", "Escape", "^c" (Ctrl+C), "%{F4}" (Alt+F4)
func pressKey(key string) error {
	script := fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
[System.Windows.Forms.SendKeys]::SendWait("%s")
`, key)
	return execPSRun(script)
}

// hotkey: Windows 단축키 (Win+D 등)
func hotkey(modifiers []string, key string) error {
	// wscript.exe Shell.SendKeys 방식으로 Win키 포함 단축키 지원
	combo := ""
	for _, m := range modifiers {
		switch strings.ToLower(m) {
		case "ctrl":
			combo += "^"
		case "alt":
			combo += "%"
		case "shift":
			combo += "+"
		case "win":
			combo = "" // Win키는 별도 처리
		}
	}

	hasWin := false
	for _, m := range modifiers {
		if strings.ToLower(m) == "win" {
			hasWin = true
		}
	}

	if hasWin {
		script := fmt.Sprintf(`
$wshell = New-Object -com "Wscript.Shell"
$wshell.SendKeys("^{ESC}")
Start-Sleep -Milliseconds 200
$wshell.SendKeys("%s")
`, key)
		return execPSRun(script)
	}

	return pressKey(combo + key)
}

// ── 앱 실행 & 창 제어 ───────────────────────────────────────────

// launchAndFocus: 앱 실행 후 포커스
func launchAndFocus(appName string) error {
	script := fmt.Sprintf(`Start-Process "%s"`, appName)
	if err := execPSRun(script); err != nil {
		return err
	}
	time.Sleep(1500 * time.Millisecond)
	return bringToFront(appName)
}

// bringToFront: 창 최상단으로
func bringToFront(titleKeyword string) error {
	script := fmt.Sprintf(`
Add-Type @"
using System;
using System.Runtime.InteropServices;
public class WinApi {
    [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr hWnd);
    [DllImport("user32.dll")] public static extern bool ShowWindow(IntPtr hWnd, int nCmdShow);
}
"@
$proc = Get-Process | Where-Object { $_.MainWindowTitle -like "*%s*" } | Select-Object -First 1
if ($proc) {
    [WinApi]::ShowWindow($proc.MainWindowHandle, 9)
    [WinApi]::SetForegroundWindow($proc.MainWindowHandle)
    Write-Output "OK"
} else {
    Write-Output "NOT_FOUND"
}
`, titleKeyword)
	out, _ := execPS(script)
	if strings.TrimSpace(string(out)) == "NOT_FOUND" {
		return fmt.Errorf("창을 찾을 수 없어요: %s", titleKeyword)
	}
	return nil
}

// ── Vision-Action 루프 핵심 ─────────────────────────────────────

// DesktopAction: Groq Vision이 결정하는 다음 액션
type DesktopAction struct {
	Type      string `json:"type"`    // click|type|key|scroll|launch|done|wait|error
	X         int    `json:"x,omitempty"`
	Y         int    `json:"y,omitempty"`
	Text      string `json:"text,omitempty"`
	Key       string `json:"key,omitempty"`
	AppName   string `json:"app_name,omitempty"`
	Direction string `json:"direction,omitempty"` // up|down
	Reason    string `json:"reason"`
	Done      bool   `json:"done"`
}

// captureAndAnalyze: 화면 캡처 → Groq Vision → 다음 액션 결정
func captureAndAnalyze(goal, history string) (DesktopAction, string, error) {
	// 1. 화면 캡처
	b64, w, h, err := captureScreenPowerShell()
	if err != nil {
		return DesktopAction{}, "", err
	}

	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()

	if gKey == "" {
		return DesktopAction{}, b64, fmt.Errorf("Groq API 키가 설정되지 않았습니다")
	}

	// 2. Groq Vision에 화면 전달 + 다음 액션 결정
	sysPrompt := fmt.Sprintf(`You are a Desktop Computer Use Agent controlling a Windows PC (resolution: %dx%d).
Your goal: "%s"

Previous actions taken:
%s

Analyze the screenshot and decide the NEXT SINGLE action to take.
Return JSON only:
{
  "type": "click|type|key|scroll|launch|done|wait",
  "x": <pixel x for click/scroll>,
  "y": <pixel y for click/scroll>,
  "text": "<text to type>",
  "key": "<SendKeys string e.g. {Enter}, ^c, %%{F4}>",
  "app_name": "<executable name for launch>",
  "direction": "up|down (for scroll)",
  "reason": "<why this action>",
  "done": <true if goal is complete>
}

Rules:
1. One action at a time
2. If goal is complete, set done=true and type=done
3. If stuck after 3 same actions, try different approach
4. Prefer keyboard shortcuts over mouse when possible
5. Wait 1-2 seconds after launching apps`, w, h, goal, history)

	question := "What is the current state of the screen, and what single action should I take next to achieve the goal?"

	answer, err := callGroqVision(gKey, b64, "image/png", sysPrompt+"\n\n"+question)
	if err != nil {
		return DesktopAction{}, b64, fmt.Errorf("Vision 분석 실패: %w", err)
	}

	// JSON 파싱
	clean := strings.TrimSpace(answer)
	if idx := strings.Index(clean, "{"); idx >= 0 {
		clean = clean[idx:]
	}
	if idx := strings.LastIndex(clean, "}"); idx >= 0 {
		clean = clean[:idx+1]
	}

	var action DesktopAction
	if err := json.Unmarshal([]byte(clean), &action); err != nil {
		return DesktopAction{}, b64, fmt.Errorf("액션 파싱 실패: %w (raw: %s)", err, clean)
	}

	return action, b64, nil
}

// executeAction: DesktopAction 실행
func executeAction(action DesktopAction) error {
	switch action.Type {
	case "click":
		return clickAt(action.X, action.Y, "left")
	case "right_click":
		return clickAt(action.X, action.Y, "right")
	case "double_click":
		return doubleClick(action.X, action.Y)
	case "type":
		return typeText(action.Text)
	case "key":
		return pressKey(action.Key)
	case "scroll":
		delta := 3
		if action.Direction == "down" {
			delta = -3
		}
		return scrollAt(action.X, action.Y, delta)
	case "launch":
		return launchAndFocus(action.AppName)
	case "wait":
		time.Sleep(1500 * time.Millisecond)
		return nil
	case "done":
		return nil
	default:
		return fmt.Errorf("알 수 없는 액션 타입: %s", action.Type)
	}
}

// ── 사용자 승인 플로우 ──────────────────────────────────────────

type ApprovalRequest struct {
	TaskID   string        `json:"task_id"`
	Action   DesktopAction `json:"action"`
	Question string        `json:"question"`
}

var (
	pendingApprovals   = make(map[string]chan bool)
	approvalMu         sync.Mutex
)

func requestApproval(taskID string, action DesktopAction) bool {
	ch := make(chan bool, 1)
	approvalMu.Lock()
	pendingApprovals[taskID] = ch
	approvalMu.Unlock()

	defer func() {
		approvalMu.Lock()
		delete(pendingApprovals, taskID)
		approvalMu.Unlock()
	}()

	// SSE로 프론트에 승인 요청 push
	question := fmt.Sprintf("다음 작업을 수행할까요? [%s]", action.Reason)
	publishAlert(Alert{
		ID:      "approval_" + taskID,
		Level:   "warn",
		Title:   "작업 승인 요청 ✋",
		Message: question,
		Action:  "approve:" + taskID,
	})

	// 30초 대기 (타임아웃 시 거부)
	select {
	case approved := <-ch:
		return approved
	case <-time.After(30 * time.Second):
		return false
	}
}

// POST /api/agent/desktop/approve — 사용자가 승인/거부
func handleDesktopApprove(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		TaskID   string `json:"task_id"`
		Approved bool   `json:"approved"`
	}
	tryDecodeBody(r, &req)
	approvalMu.Lock()
	ch, ok := pendingApprovals[req.TaskID]
	approvalMu.Unlock()

	if !ok {
		json200(w, map[string]any{"success": false, "message": msgT("승인 요청을 찾을 수 없어요", "Approval request not found", lang)})
		return
	}
	ch <- req.Approved
	json200(w, map[string]any{"success": true, "approved": req.Approved})
}

// ── 위험 액션 판별 ────────────────────────────────────────────

func isDangerousAction(action DesktopAction) bool {
	dangerous := []string{"del", "delete", "format", "rm", "remove", "uninstall",
		"shutdown", "restart", "payment", "pay", "purchase", "buy", "transfer"}
	combined := strings.ToLower(action.Text + action.Key + action.AppName + action.Reason)
	for _, d := range dangerous {
		if strings.Contains(combined, d) {
			return true
		}
	}
	return false
}

// ── 메인 에이전트 실행 루프 ─────────────────────────────────────

func runDesktopAgent(task *AgentTask, goal string, requireApproval bool, maxSteps int) {
	if maxSteps == 0 {
		maxSteps = 20
	}

	actionHistory := []string{}
	sameActionCount := 0
	lastActionType := ""

	// 스크린샷 저장 디렉토리
	ssDir := filepath.Join(os.TempDir(), "nexus_agent_screenshots")
	os.MkdirAll(ssDir, 0755)

	for step := 1; step <= maxSteps; step++ {
		if task.IsCancelled() {
			task.Status = TaskCancelled
			task.Message = "사용자가 취소했습니다"
			return
		}

		task.UpdateProgress(step*100/maxSteps, fmt.Sprintf("단계 %d/%d: 화면 분석 중...", step, maxSteps))

		// 화면 분석
		histStr := strings.Join(actionHistory, "\n")
		action, b64, err := captureAndAnalyze(goal, histStr)
		if err != nil {
			task.Status = TaskFailed
			task.Error = err.Error()
			task.Message = "화면 분석 실패: " + err.Error()
			return
		}

		// 스크린샷 저장 (디버그용)
		ssPath := filepath.Join(ssDir, fmt.Sprintf("step_%02d.png", step))
		if imgData, decErr := base64.StdEncoding.DecodeString(b64); decErr == nil {
			os.WriteFile(ssPath, imgData, 0644)
		}

		// 완료 판단
		if action.Done || action.Type == "done" {
			task.Status = TaskDone
			task.Progress = 100
			task.Message = "목표 달성 완료!"
			task.Result = map[string]any{
				"goal":        goal,
				"steps_taken": step,
				"last_reason": action.Reason,
			}
			return
		}

		// 동일 액션 반복 감지
		if action.Type == lastActionType {
			sameActionCount++
			if sameActionCount >= 3 {
				task.Status = TaskFailed
				task.Error = fmt.Sprintf("같은 액션(%s)이 3번 반복됩니다. 목표를 달성할 수 없어요.", action.Type)
				return
			}
		} else {
			sameActionCount = 0
			lastActionType = action.Type
		}

		// 위험 액션 → 승인 필요
		if isDangerousAction(action) || requireApproval {
			task.UpdateProgress(step*100/maxSteps, fmt.Sprintf("단계 %d: 승인 대기 중... [%s]", step, action.Reason))
			if !requestApproval(task.ID, action) {
				task.Status = TaskCancelled
				task.Message = "사용자가 작업을 거부했습니다"
				return
			}
		}

		// 액션 실행
		task.UpdateProgress(step*100/maxSteps, fmt.Sprintf("단계 %d: %s 실행 중...", step, action.Type))
		if err := executeAction(action); err != nil {
			actionHistory = append(actionHistory, fmt.Sprintf("Step %d: %s (실패: %s)", step, action.Reason, err.Error()))
		} else {
			actionHistory = append(actionHistory, fmt.Sprintf("Step %d: %s → %s", step, action.Type, action.Reason))
		}

		// 액션 후 대기 (UI 반응 시간)
		time.Sleep(500 * time.Millisecond)
	}

	task.Status = TaskFailed
	task.Error = fmt.Sprintf("최대 단계(%d) 초과 — 목표를 완료하지 못했습니다", maxSteps)
}

// ── HTTP 핸들러 ─────────────────────────────────────────────────

// POST /api/agent/desktop/run
func handleDesktopAgentRun(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		Goal            string `json:"goal"`
		RequireApproval bool   `json:"require_approval"`
		MaxSteps        int    `json:"max_steps"`
	}
	tryDecodeBody(r, &req)
	if req.Goal == "" {
		json200(w, map[string]any{"success": false, "message": msgT("goal이 필요해요", "goal is required", lang)})
		return
	}
	if req.MaxSteps == 0 {
		req.MaxSteps = 20
	}

	// 기본적으로 승인 필요
	if !req.RequireApproval {
		req.RequireApproval = true
	}

	task := globalTaskQueue.Enqueue(
		"Desktop Agent: "+req.Goal,
		PriorityNormal,
		map[string]any{"goal": req.Goal},
		func(t *AgentTask) {
			runDesktopAgent(t, req.Goal, req.RequireApproval, req.MaxSteps)
		},
	)

	json200(w, map[string]any{
		"success": true,
		"task_id": task.ID,
		"message": msgT(fmt.Sprintf("Desktop Agent 시작: '%s'", req.Goal), fmt.Sprintf("Desktop Agent started: '%s'", req.Goal), lang),
	})
}

// POST /api/agent/desktop/click — 직접 클릭
func handleDesktopClick(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		X      int    `json:"x"`
		Y      int    `json:"y"`
		Button string `json:"button"` // left|right|double
	}
	tryDecodeBody(r, &req)
	var err error
	switch req.Button {
	case "right":
		err = clickAt(req.X, req.Y, "right")
	case "double":
		err = doubleClick(req.X, req.Y)
	default:
		err = clickAt(req.X, req.Y, "left")
	}

	if err != nil {
		json200(w, map[string]any{"success": false, "message": err.Error()})
		return
	}
	json200(w, map[string]any{"success": true, "message": msgT(fmt.Sprintf("(%d,%d) 클릭 완료", req.X, req.Y), fmt.Sprintf("(%d,%d) click done", req.X, req.Y), lang)})
}

// POST /api/agent/desktop/type — 텍스트 입력
func handleDesktopType(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct{ Text string `json:"text"` }
	tryDecodeBody(r, &req)
	if err := typeText(req.Text); err != nil {
		json200(w, map[string]any{"success": false, "message": err.Error()})
		return
	}
	json200(w, map[string]any{"success": true, "message": msgT("입력 완료", "Input done", lang)})
}

// POST /api/agent/desktop/key — 키 입력
func handleDesktopKey(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct{ Key string `json:"key"` }
	tryDecodeBody(r, &req)
	if err := pressKey(req.Key); err != nil {
		json200(w, map[string]any{"success": false, "message": err.Error()})
		return
	}
	json200(w, map[string]any{"success": true, "message": msgT("키 입력 완료", "Key input done", lang)})
}

// POST /api/agent/desktop/scroll
func handleDesktopScroll(w http.ResponseWriter, r *http.Request) {
	var req struct {
		X         int    `json:"x"`
		Y         int    `json:"y"`
		Direction string `json:"direction"` // up|down
		Amount    int    `json:"amount"`
	}
	tryDecodeBody(r, &req)
	if req.Amount == 0 {
		req.Amount = 3
	}
	delta := req.Amount
	if req.Direction == "down" {
		delta = -req.Amount
	}
	if err := scrollAt(req.X, req.Y, delta); err != nil {
		json200(w, map[string]any{"success": false, "message": err.Error()})
		return
	}
	json200(w, map[string]any{"success": true})
}

// POST /api/agent/desktop/drag
func handleDesktopDrag(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FromX int `json:"from_x"`
		FromY int `json:"from_y"`
		ToX   int `json:"to_x"`
		ToY   int `json:"to_y"`
	}
	tryDecodeBody(r, &req)
	if err := dragTo(req.FromX, req.FromY, req.ToX, req.ToY); err != nil {
		json200(w, map[string]any{"success": false, "message": err.Error()})
		return
	}
	json200(w, map[string]any{"success": true})
}

// GET /api/agent/desktop/screenshot — 현재 화면 캡처 + OCR
func handleDesktopScreenshot(w http.ResponseWriter, r *http.Request) {
	b64, width, height, err := captureScreenPowerShell()
	if err != nil {
		json200(w, map[string]any{"success": false, "message": err.Error()})
		return
	}

	withOCR := r.URL.Query().Get("ocr") == "true"
	result := map[string]any{
		"success": true,
		"base64":  b64,
		"width":   width,
		"height":  height,
		"mime":    "image/png",
	}

	if withOCR {
		tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("nexus_da_%d.png", time.Now().UnixNano()))
		defer os.Remove(tmpFile)
		if imgData, err := base64.StdEncoding.DecodeString(b64); err == nil {
			os.WriteFile(tmpFile, imgData, 0644)
			ocrText, _ := runWindowsOCR(tmpFile)
			result["ocr_text"] = ocrText
		}
	}

	json200(w, result)
}

// GET /api/agent/desktop/status — 현재 활성 창 + 마우스 위치
func handleDesktopStatus(w http.ResponseWriter, r *http.Request) {
	title := getActiveWindowTitle()

	type POINTSTRUCT struct{ X, Y int32 }
	var pos POINTSTRUCT
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pos)))

	width, height := getScreenSize()

	json200(w, map[string]any{
		"success":      true,
		"active_title": title,
		"cursor_x":     pos.X,
		"cursor_y":     pos.Y,
		"screen_w":     width,
		"screen_h":     height,
	})
}

// POST /api/agent/desktop/window — 창 제어 (focus/maximize/minimize/restore/close/hide/show)
// {title: "Chrome", action: "maximize"} — title 부분일치 / action 7종
func handleDesktopWindow(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		Title  string `json:"title"`   // 창 제목 부분일치
		Action string `json:"action"`  // focus|maximize|minimize|restore|close|hide|show
	}
	tryDecodeBody(r, &req)
	if req.Title == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("title 필요", "title required", lang)})
		return
	}

	// SW_ 상수
	const (
		SW_HIDE      = 0
		SW_SHOWNORMAL = 1
		SW_MINIMIZE  = 6
		SW_MAXIMIZE  = 3
		SW_RESTORE   = 9
		SW_SHOW      = 5
	)

	// FindWindowW(NULL, lpWindowName) — 정확한 매칭만 됨 — 부분일치는 EnumWindows 필요
	// 우선 PowerShell로 부분일치 + 핸들 가져오기
	psScript := fmt.Sprintf(`
$ErrorActionPreference = 'SilentlyContinue'
$proc = Get-Process | Where-Object { $_.MainWindowTitle -like "*%s*" -and $_.MainWindowHandle -ne 0 } | Select-Object -First 1
if ($null -eq $proc) { Write-Output "0"; exit }
Write-Output $proc.MainWindowHandle.ToInt64()
`, strings.ReplaceAll(req.Title, `"`, `'`))
	out, err := newHiddenCmd("powershell", "-NoProfile", "-Command", psScript).Output()
	if err != nil {
		json200(w, map[string]any{"success": false, "message": fmt.Sprintf("창 검색 실패: %v", err)})
		return
	}
	hwndStr := strings.TrimSpace(string(out))
	if hwndStr == "" || hwndStr == "0" {
		json200(w, map[string]any{"success": false, "message": msgT(fmt.Sprintf("'%s' 창을 찾을 수 없어요", req.Title), fmt.Sprintf("Window '%s' not found", req.Title), lang)})
		return
	}
	// 핸들 정수 파싱
	var hwnd int64
	fmt.Sscanf(hwndStr, "%d", &hwnd)
	if hwnd == 0 {
		json200(w, map[string]any{"success": false, "message": msgT("창 핸들 파싱 실패", "Window handle parse failed", lang)})
		return
	}

	var swCmd int
	switch req.Action {
	case "focus":     swCmd = SW_SHOW
	case "maximize":  swCmd = SW_MAXIMIZE
	case "minimize":  swCmd = SW_MINIMIZE
	case "restore":   swCmd = SW_RESTORE
	case "hide":      swCmd = SW_HIDE
	case "show":      swCmd = SW_SHOWNORMAL
	case "close":
		// WM_CLOSE 전송
		const WM_CLOSE = 0x0010
		procPostMessage := user32dll.NewProc("PostMessageW")
		procPostMessage.Call(uintptr(hwnd), uintptr(WM_CLOSE), 0, 0)
		json200(w, map[string]any{"success": true, "message": msgT(fmt.Sprintf("'%s' 창을 닫았어요", req.Title), fmt.Sprintf("Closed '%s'", req.Title), lang)})
		return
	default:
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("action: focus|maximize|minimize|restore|close|hide|show", "action: focus|maximize|minimize|restore|close|hide|show", lang)})
		return
	}

	procShowWindow.Call(uintptr(hwnd), uintptr(swCmd))
	// focus 액션은 ShowWindow + SetForegroundWindow 둘 다
	if req.Action == "focus" || req.Action == "maximize" || req.Action == "restore" {
		procSetForeground.Call(uintptr(hwnd))
	}
	json200(w, map[string]any{
		"success": true,
		"message": msgT(fmt.Sprintf("'%s' 창 %s 완료", req.Title, req.Action), fmt.Sprintf("Window '%s' %s done", req.Title, req.Action), lang),
		"hwnd":    hwnd,
	})
}

// GET /api/agent/desktop/windows — 열린 창 목록
func handleDesktopWindowList(w http.ResponseWriter, r *http.Request) {
	psScript := `
Get-Process | Where-Object { $_.MainWindowTitle -ne "" -and $_.MainWindowHandle -ne 0 } |
  Select-Object @{N='title';E={$_.MainWindowTitle}}, @{N='process';E={$_.ProcessName}}, @{N='pid';E={$_.Id}} |
  ConvertTo-Json -Compress
`
	out, err := newHiddenCmd("powershell", "-NoProfile", "-Command", psScript).Output()
	if err != nil {
		json200(w, map[string]any{"success": false, "message": err.Error(), "windows": []any{}})
		return
	}
	var windows any
	json.Unmarshal(out, &windows)
	json200(w, map[string]any{"success": true, "windows": windows})
}

// POST /api/desktop/agent/cancel — 실행 중인 Desktop Agent 취소
func handleDesktopAgentCancel(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		TaskID string `json:"task_id"`
	}
	tryDecodeBody(r, &req)
	if req.TaskID == "" {
		json200(w, map[string]any{"success": false, "message": msgT("task_id가 필요합니다", "task_id is required", lang)})
		return
	}

	task, ok := globalTaskQueue.GetTask(req.TaskID)
	if !ok {
		json200(w, map[string]any{"success": false, "message": msgT("태스크를 찾을 수 없습니다", "Task not found", lang)})
		return
	}
	task.Cancel()
	task.Status = TaskCancelled
	fin := time.Now()
	task.FinishedAt = &fin
	json200(w, map[string]any{"success": true, "message": msgT("Desktop Agent 취소 완료", "Desktop Agent cancelled", lang)})
}
