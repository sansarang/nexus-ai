//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
)

// ──────────────────────────────────────────
// 시스템 제어: 볼륨
// ──────────────────────────────────────────

func handleVolume(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action string `json:"action"` // set | get | mute | unmute
		Value  int    `json:"value"`  // 0-100
	}
	json.NewDecoder(r.Body).Decode(&req)

	switch req.Action {
	case "get":
		out, _ := exec.Command("powershell", "-NoProfile", "-Command",
			`Add-Type -TypeDefinition 'using System.Runtime.InteropServices; public class Vol { [DllImport("winmm.dll")] public static extern int waveOutGetVolume(System.IntPtr h, out uint v); }'; $v = [uint32]0; [Vol]::waveOutGetVolume([System.IntPtr]::Zero, [ref]$v); [math]::Round(($v -band 0xFFFF) / 65535 * 100)`).Output()
		vol, _ := strconv.Atoi(strings.TrimSpace(string(out)))
		json200(w, map[string]any{"volume": vol, "message": fmt.Sprintf("현재 볼륨: %d%%", vol)})

	case "mute":
		exec.Command("powershell", "-NoProfile", "-Command",
			`(New-Object -ComObject WScript.Shell).SendKeys([char]173)`).Run()
		json200(w, map[string]any{"success": true, "message": "음소거 처리했어요 🔇"})

	case "unmute":
		exec.Command("powershell", "-NoProfile", "-Command",
			`(New-Object -ComObject WScript.Shell).SendKeys([char]173)`).Run()
		json200(w, map[string]any{"success": true, "message": "음소거 해제했어요 🔊"})

	default: // "set"
		if req.Value < 0 {
			req.Value = 0
		}
		if req.Value > 100 {
			req.Value = 100
		}
		script := fmt.Sprintf(
			`Add-Type -TypeDefinition 'using System.Runtime.InteropServices; public class Vol { [DllImport("winmm.dll")] public static extern int waveOutSetVolume(System.IntPtr h, uint v); }'; $v = [uint32](%d / 100.0 * 65535); [Vol]::waveOutSetVolume([System.IntPtr]::Zero, ($v -bor ($v -shl 16)))`,
			req.Value,
		)
		exec.Command("powershell", "-NoProfile", "-Command", script).Run()
		json200(w, map[string]any{"success": true, "volume": req.Value,
			"message": fmt.Sprintf("볼륨을 %d%%로 설정했어요 🔊", req.Value)})
	}
}

// ──────────────────────────────────────────
// 시스템 제어: 화면 밝기
// ──────────────────────────────────────────

func handleBrightness(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action string `json:"action"` // set | get
		Value  int    `json:"value"`  // 0-100
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.Action == "get" {
		out, _ := exec.Command("powershell", "-NoProfile", "-Command",
			`(Get-WmiObject -Namespace root/WMI -Class WmiMonitorBrightness).CurrentBrightness`).Output()
		val, _ := strconv.Atoi(strings.TrimSpace(string(out)))
		json200(w, map[string]any{"brightness": val, "message": fmt.Sprintf("현재 밝기: %d%%", val)})
		return
	}

	if req.Value < 0 {
		req.Value = 0
	}
	if req.Value > 100 {
		req.Value = 100
	}
	script := fmt.Sprintf(
		`(Get-WmiObject -Namespace root/WMI -Class WmiMonitorBrightnessMethods).WmiSetBrightness(1, %d)`,
		req.Value,
	)
	err := exec.Command("powershell", "-NoProfile", "-Command", script).Run()
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": "밝기 조절 실패 (노트북 전용 기능이에요)"})
		return
	}
	json200(w, map[string]any{"success": true, "brightness": req.Value,
		"message": fmt.Sprintf("밝기를 %d%%로 설정했어요 ☀️", req.Value)})
}

// ──────────────────────────────────────────
// 시스템 제어: Wi-Fi
// ──────────────────────────────────────────

func handleWifi(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action string `json:"action"` // on | off | status
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.Action == "status" {
		out, _ := exec.Command("powershell", "-NoProfile", "-Command",
			`(Get-NetAdapter -Name 'Wi-Fi' -EA SilentlyContinue).Status`).Output()
		status := strings.TrimSpace(string(out))
		connected := strings.EqualFold(status, "Up")
		json200(w, map[string]any{"connected": connected, "status": status})
		return
	}

	action := "enable"
	msg := "Wi-Fi를 켰어요 📶"
	if req.Action == "off" {
		action = "disable"
		msg = "Wi-Fi를 껐어요 📵"
	}

	script := fmt.Sprintf(`netsh interface set interface "Wi-Fi" %s`, action)
	exec.Command("powershell", "-NoProfile", "-Command", script).Run()
	json200(w, map[string]any{"success": true, "message": msg})
}

// ──────────────────────────────────────────
// 시스템 제어: 전원 (잠금·절전·재시작·종료)
// ──────────────────────────────────────────

func handlePower(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action string `json:"action"` // lock | sleep | restart | shutdown
	}
	json.NewDecoder(r.Body).Decode(&req)

	var cmd *exec.Cmd
	var msg string

	switch req.Action {
	case "lock":
		cmd = exec.Command("rundll32.exe", "user32.dll,LockWorkStation")
		msg = "화면을 잠갔어요 🔒"
	case "sleep":
		cmd = exec.Command("powershell", "-NoProfile", "-Command",
			`Add-Type -Assembly System.Windows.Forms; [System.Windows.Forms.Application]::SetSuspendState('Suspend',$false,$false)`)
		msg = "절전 모드로 전환해요 😴"
	case "restart":
		cmd = exec.Command("shutdown", "/r", "/t", "10")
		msg = "10초 후 재시작합니다 🔄"
	case "shutdown":
		cmd = exec.Command("shutdown", "/s", "/t", "10")
		msg = "10초 후 종료합니다 ⏻"
	default:
		writeJSON(w, 400, map[string]any{"success": false, "message": "알 수 없는 명령"})
		return
	}

	cmd.Start()
	json200(w, map[string]any{"success": true, "action": req.Action, "message": msg})
}

// ──────────────────────────────────────────
// 시스템 제어: 앱 실행
// ──────────────────────────────────────────

var appAliases = map[string]string{
	"크롬": "chrome", "chrome": "chrome",
	"파이어폭스": "firefox", "firefox": "firefox",
	"엣지": "msedge", "edge": "msedge",
	"메모장": "notepad", "notepad": "notepad",
	"계산기": "calc", "calc": "calc",
	"탐색기": "explorer", "explorer": "explorer",
	"워드패드": "wordpad", "wordpad": "wordpad",
	"페인트": "mspaint", "paint": "mspaint",
	"cmd": "cmd", "명령프롬프트": "cmd",
	"powershell": "powershell",
	"작업관리자": "taskmgr", "taskmgr": "taskmgr",
	"제어판": "control", "control": "control",
	"설정": "ms-settings:", "settings": "ms-settings:",
	"스팀": "steam", "steam": "steam",
	"디스코드": "discord", "discord": "discord",
	"카카오": "kakaotalk", "kakao": "kakaotalk",
	"vs code": "code", "vscode": "code",
}

func handleLaunchApp(w http.ResponseWriter, r *http.Request) {
	var req struct {
		App string `json:"app"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	appLow := strings.ToLower(strings.TrimSpace(req.App))
	target, ok := appAliases[appLow]
	if !ok {
		target = appLow // 직접 실행 시도
	}

	var cmd *exec.Cmd
	if strings.HasPrefix(target, "ms-") {
		cmd = exec.Command("cmd", "/c", "start", "", target)
	} else {
		cmd = exec.Command("cmd", "/c", "start", "", target)
	}

	if err := cmd.Start(); err != nil {
		writeJSON(w, 500, map[string]any{
			"success": false,
			"message": fmt.Sprintf("'%s' 실행 실패: 설치되어 있는지 확인해주세요", req.App),
		})
		return
	}
	json200(w, map[string]any{
		"success": true,
		"app":     req.App,
		"message": fmt.Sprintf("'%s'을(를) 실행했어요 🚀", req.App),
	})
}

// ──────────────────────────────────────────
// 실시간: 상위 프로세스 (CPU·메모리 기준)
// ──────────────────────────────────────────

func handleProcessTop(w http.ResponseWriter, r *http.Request) {
	cpuOut, _ := exec.Command("powershell", "-NoProfile", "-Command",
		`Get-Process | Sort-Object CPU -Desc | Select-Object -First 10 Name,Id,CPU,WorkingSet | ConvertTo-Json -Compress`).Output()

	var cpuProcs []struct {
		Name       string  `json:"Name"`
		Id         int     `json:"Id"`
		CPU        float64 `json:"CPU"`
		WorkingSet int64   `json:"WorkingSet"`
	}
	json.Unmarshal(cpuOut, &cpuProcs)

	type ProcItem struct {
		Name  string  `json:"name"`
		PID   int     `json:"pid"`
		CPU   float64 `json:"cpu"`
		MemMB float64 `json:"mem_mb"`
	}

	var byCPU []ProcItem
	for _, p := range cpuProcs {
		byCPU = append(byCPU, ProcItem{
			Name: p.Name, PID: p.Id, CPU: p.CPU,
			MemMB: float64(p.WorkingSet) / (1 << 20),
		})
	}

	memOut, _ := exec.Command("powershell", "-NoProfile", "-Command",
		`Get-Process | Sort-Object WorkingSet -Desc | Select-Object -First 10 Name,Id,CPU,WorkingSet | ConvertTo-Json -Compress`).Output()

	var memProcs []struct {
		Name       string  `json:"Name"`
		Id         int     `json:"Id"`
		CPU        float64 `json:"CPU"`
		WorkingSet int64   `json:"WorkingSet"`
	}
	json.Unmarshal(memOut, &memProcs)

	var byMem []ProcItem
	for _, p := range memProcs {
		byMem = append(byMem, ProcItem{
			Name: p.Name, PID: p.Id, CPU: p.CPU,
			MemMB: float64(p.WorkingSet) / (1 << 20),
		})
	}

	json200(w, map[string]any{
		"by_cpu": byCPU,
		"by_mem": byMem,
	})
}
