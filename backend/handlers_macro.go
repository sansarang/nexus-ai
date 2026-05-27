//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ──────────────────────────────────────────
// 매크로 타입 정의
// ──────────────────────────────────────────

type MacroAction struct {
	Type    string            `json:"type"`   // launch | clean | folder | volume | brightness | delay | shell | message
	Params  map[string]string `json:"params"` // type별 파라미터
	Label   string            `json:"label"`
}

type Macro struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Trigger     MacroTrigger  `json:"trigger"`
	Actions     []MacroAction `json:"actions"`
	Enabled     bool          `json:"enabled"`
	LastRun     string        `json:"last_run"`
	RunCount    int           `json:"run_count"`
	CreatedAt   string        `json:"created_at"`
}

type MacroTrigger struct {
	Type    string `json:"type"`  // time | startup | interval | manual
	Time    string `json:"time"`  // "09:00"
	Days    []int  `json:"days"`  // 0=일 1=월 ... 6=토
	Interval int   `json:"interval_min"` // N분마다
}

// ──────────────────────────────────────────
// 저장소
// ──────────────────────────────────────────

func macroStorePath() string {
	appData, _ := os.UserConfigDir()
	dir := filepath.Join(appData, "Nexus")
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "macros.json")
}

func loadMacros() []Macro {
	data, err := os.ReadFile(macroStorePath())
	if err != nil {
		return []Macro{}
	}
	var macros []Macro
	json.Unmarshal(data, &macros)
	return macros
}

func saveMacros(macros []Macro) error {
	data, err := json.MarshalIndent(macros, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(macroStorePath(), data, 0644)
}

// ──────────────────────────────────────────
// 매크로 목록
// ──────────────────────────────────────────

func handleMacroList(w http.ResponseWriter, r *http.Request) {
	macros := loadMacros()
	json200(w, map[string]any{
		"macros": macros,
		"total":  len(macros),
	})
}

// ──────────────────────────────────────────
// 매크로 생성
// ──────────────────────────────────────────

func handleMacroCreate(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var macro Macro
	if err := json.NewDecoder(r.Body).Decode(&macro); err != nil {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("잘못된 요청입니다", "Invalid request", lang)})
		return
	}

	if macro.Name == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("매크로 이름을 입력해주세요", "Please enter a macro name", lang)})
		return
	}

	macro.ID = fmt.Sprintf("macro_%d", time.Now().UnixNano())
	macro.Enabled = true
	macro.CreatedAt = time.Now().Format("2006-01-02 15:04:05")
	macro.RunCount = 0

	macros := loadMacros()
	macros = append(macros, macro)
	if err := saveMacros(macros); err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": msgT("저장 실패", "Save failed", lang)})
		return
	}

	// 시간 기반 트리거 → Windows 작업 스케줄러 등록
	if macro.Trigger.Type == "time" && macro.Trigger.Time != "" {
		scheduleWindowsTask(macro)
	}

	json200(w, map[string]any{
		"success": true,
		"macro":   macro,
		"message": fmt.Sprintf(msgT("매크로 '%s' 등록 완료!", "Macro '%s' registered!", lang), macro.Name),
	})
}

// ──────────────────────────────────────────
// 매크로 실행
// ──────────────────────────────────────────

func handleMacroRun(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		ID string `json:"id"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	macros := loadMacros()
	var target *Macro
	for i := range macros {
		if macros[i].ID == req.ID {
			target = &macros[i]
			break
		}
	}
	if target == nil {
		writeJSON(w, 404, map[string]any{"success": false, "message": msgT("매크로를 찾을 수 없어요", "Macro not found", lang)})
		return
	}

	results := executeMacroActions(target.Actions)

	// 실행 기록
	target.LastRun = time.Now().Format("2006-01-02 15:04:05")
	target.RunCount++
	saveMacros(macros)

	json200(w, map[string]any{
		"success": true,
		"name":    target.Name,
		"results": results,
		"message": fmt.Sprintf(msgT("매크로 '%s' 실행 완료 (%d개 동작)", "Macro '%s' executed (%d actions)", lang), target.Name, len(results)),
	})
}

// ──────────────────────────────────────────
// 매크로 액션 실행
// ──────────────────────────────────────────

type MacroResult struct {
	Action  string `json:"action"`
	Label   string `json:"label"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func executeMacroActions(actions []MacroAction) []MacroResult {
	var results []MacroResult

	for _, action := range actions {
		result := MacroResult{Action: action.Type, Label: action.Label}

		switch action.Type {

		case "launch":
			app := action.Params["app"]
			if app == "" {
				app = action.Params["path"]
			}
			err := launchKnownApp(app)
			result.Success = err == nil
			if err != nil {
				result.Message = fmt.Sprintf("%s 실행 실패: %v", app, err)
			} else {
				result.Message = fmt.Sprintf("%s 실행됨", app)
			}

		case "folder":
			folder := action.Params["path"]
			if folder == "" {
				folder = action.Params["folder"]
			}
			resolved := resolveFolder(folder)
			err := newHiddenCmd("explorer", resolved).Start()
			result.Success = err == nil
			result.Message = fmt.Sprintf("폴더 열림: %s", folder)

		case "clean":
			targets := strings.Split(action.Params["targets"], ",")
			freed := performClean(targets)
			result.Success = true
			result.Message = fmt.Sprintf("정리 완료 (%.1f MB 해제)", float64(freed)/(1024*1024))

		case "volume":
			level := action.Params["level"]
			out, err := newHiddenCmd("powershell", "-NoProfile", "-Command",
				fmt.Sprintf(`$vol = [System.Math]::Round(%s/100 * 65535); $wsh = New-Object -ComObject WScript.Shell; $wsh.SendKeys([char]0xAD)`, level)).Output()
			result.Success = err == nil
			result.Message = fmt.Sprintf("볼륨 %s%%로 설정 (결과: %s)", level, strings.TrimSpace(string(out)))

		case "message":
			title := action.Params["title"]
			msg := action.Params["body"]
			if title == "" {
				title = "Nexus 매크로"
			}
			newHiddenCmd("powershell", "-NoProfile", "-Command",
				fmt.Sprintf(`Add-Type -AssemblyName System.Windows.Forms; [System.Windows.Forms.MessageBox]::Show('%s','%s')`, msg, title)).Start()
			result.Success = true
			result.Message = "알림 표시됨"

		case "delay":
			secs := action.Params["seconds"]
			var s int
			fmt.Sscanf(secs, "%d", &s)
			if s > 0 && s <= 60 {
				time.Sleep(time.Duration(s) * time.Second)
			}
			result.Success = true
			result.Message = fmt.Sprintf("%s초 대기 완료", secs)

		case "shell":
			cmd := action.Params["command"]
			out, err := newHiddenCmd("powershell", "-NoProfile", "-Command", cmd).Output()
			result.Success = err == nil
			result.Message = strings.TrimSpace(string(out))
			if len(result.Message) > 100 {
				result.Message = result.Message[:100] + "..."
			}

		default:
			result.Success = false
			result.Message = fmt.Sprintf("알 수 없는 동작: %s", action.Type)
		}

		results = append(results, result)
	}
	return results
}

func launchKnownApp(name string) error {
	lower := strings.ToLower(name)
	appMap := map[string]string{
		"chrome": `C:\Program Files\Google\Chrome\Application\chrome.exe`,
		"크롬":    `C:\Program Files\Google\Chrome\Application\chrome.exe`,
		"edge":   `C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
		"엣지":    `C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
		"notepad": "notepad.exe", "메모장": "notepad.exe",
		"explorer": "explorer.exe", "탐색기": "explorer.exe",
		"calculator": "calc.exe", "계산기": "calc.exe",
		"cmd": "cmd.exe", "powershell": "powershell.exe",
		"word": "winword.exe", "워드": "winword.exe",
		"excel": "excel.exe", "엑셀": "excel.exe",
		"outlook": "outlook.exe", "아웃룩": "outlook.exe",
	}
	if path, ok := appMap[lower]; ok {
		return newHiddenCmd("cmd", "/c", "start", "", path).Start()
	}
	// 직접 실행 시도
	return newHiddenCmd("cmd", "/c", "start", "", name).Start()
}

// ──────────────────────────────────────────
// Windows 작업 스케줄러 등록
// ──────────────────────────────────────────

func scheduleWindowsTask(macro Macro) {
	// 매크로를 PowerShell 스크립트로 변환 후 작업 스케줄러 등록
	var cmds []string
	for _, action := range macro.Actions {
		switch action.Type {
		case "launch":
			app := action.Params["app"]
			if path, ok := map[string]string{
				"chrome": `C:\Program Files\Google\Chrome\Application\chrome.exe`,
				"크롬":    `C:\Program Files\Google\Chrome\Application\chrome.exe`,
			}[strings.ToLower(app)]; ok {
				cmds = append(cmds, fmt.Sprintf(`Start-Process '%s'`, path))
			}
		case "folder":
			cmds = append(cmds, fmt.Sprintf(`Start-Process 'explorer' -ArgumentList '%s'`, action.Params["path"]))
		case "shell":
			cmds = append(cmds, action.Params["command"])
		}
	}

	if len(cmds) == 0 {
		return
	}

	scriptContent := strings.Join(cmds, "\n")
	scriptPath := filepath.Join(func() string {
		d, _ := os.UserConfigDir()
		return filepath.Join(d, "Nexus", "macros")
	}(), macro.ID+".ps1")
	os.MkdirAll(filepath.Dir(scriptPath), 0755)
	os.WriteFile(scriptPath, []byte(scriptContent), 0644)

	// 시간 파싱
	timeParts := strings.Split(macro.Trigger.Time, ":")
	if len(timeParts) != 2 {
		return
	}
	hour, min := timeParts[0], timeParts[1]

	taskScript := fmt.Sprintf(`
$action = New-ScheduledTaskAction -Execute "PowerShell.exe" -Argument "-NoProfile -File '%s'"
$trigger = New-ScheduledTaskTrigger -Daily -At "%s:%s"
$settings = New-ScheduledTaskSettingsSet -ExecutionTimeLimit (New-TimeSpan -Minutes 5)
Register-ScheduledTask -TaskName "Nexus_%s" -Action $action -Trigger $trigger -Settings $settings -Force
`, scriptPath, hour, min, macro.ID)

	newHiddenCmd("powershell", "-NoProfile", "-Command", taskScript).Start()
}

// ──────────────────────────────────────────
// 매크로 삭제
// ──────────────────────────────────────────

func handleMacroDelete(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		ID string `json:"id"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	macros := loadMacros()
	var updated []Macro
	found := false
	for _, m := range macros {
		if m.ID == req.ID {
			found = true
			// 작업 스케줄러에서도 제거
			newHiddenCmd("powershell", "-NoProfile", "-Command",
				fmt.Sprintf(`Unregister-ScheduledTask -TaskName "Nexus_%s" -Confirm:$false -ErrorAction SilentlyContinue`, req.ID)).Start()
		} else {
			updated = append(updated, m)
		}
	}
	if !found {
		writeJSON(w, 404, map[string]any{"success": false, "message": msgT("매크로를 찾을 수 없어요", "Macro not found", lang)})
		return
	}
	saveMacros(updated)
	json200(w, map[string]any{"success": true, "message": msgT("매크로가 삭제됐어요", "Macro deleted", lang)})
}

// ──────────────────────────────────────────
// 자연어 → 매크로 파싱 (LLM 없이)
// ──────────────────────────────────────────

func handleMacroParse(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		Text string `json:"text"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	text := req.Text
	lower := strings.ToLower(text)

	macro := Macro{
		Name:    "새 매크로",
		Enabled: true,
	}

	// 시간 파싱: "매일 아침 9시" / "오전 9시" / "09:00"
	timePatterns := []struct{ pattern, time string }{
		{"아침 9시", "09:00"}, {"아침 8시", "08:00"}, {"아침 7시", "07:00"},
		{"오전 9시", "09:00"}, {"오전 8시", "08:00"},
		{"점심 12시", "12:00"}, {"낮 12시", "12:00"},
		{"오후 2시", "14:00"}, {"오후 3시", "15:00"},
		{"오후 5시", "17:00"}, {"오후 6시", "18:00"},
		{"저녁 6시", "18:00"}, {"저녁 7시", "19:00"},
	}
	for _, tp := range timePatterns {
		if strings.Contains(lower, tp.pattern) {
			macro.Trigger = MacroTrigger{Type: "time", Time: tp.time}
			break
		}
	}
	if macro.Trigger.Type == "" && strings.Contains(lower, "시작할 때") {
		macro.Trigger = MacroTrigger{Type: "startup"}
	}

	// 액션 파싱
	var actions []MacroAction
	apps := []string{"크롬", "chrome", "엣지", "edge", "메모장", "notepad", "엑셀", "excel", "워드", "word", "카카오", "kakaotalk", "디스코드", "discord"}
	for _, app := range apps {
		if strings.Contains(lower, app) {
			actions = append(actions, MacroAction{
				Type:   "launch",
				Label:  app + " 실행",
				Params: map[string]string{"app": app},
			})
		}
	}
	if strings.Contains(lower, "정리") || strings.Contains(lower, "청소") {
		actions = append(actions, MacroAction{
			Type:   "clean",
			Label:  "PC 정리",
			Params: map[string]string{"targets": "temp,cache"},
		})
	}
	folders := []struct{ keyword, path string }{
		{"바탕화면", "desktop"}, {"다운로드", "downloads"}, {"문서", "documents"},
	}
	for _, f := range folders {
		if strings.Contains(lower, f.keyword) && strings.Contains(lower, "열") {
			actions = append(actions, MacroAction{
				Type:   "folder",
				Label:  f.keyword + " 폴더 열기",
				Params: map[string]string{"folder": f.path},
			})
		}
	}

	macro.Actions = actions

	// 이름 자동 생성
	if macro.Trigger.Type == "time" && macro.Trigger.Time != "" {
		macro.Name = fmt.Sprintf("%s 자동 실행", macro.Trigger.Time)
		macro.Description = text
	}

	json200(w, map[string]any{
		"macro":   macro,
		"parsed":  true,
		"message": fmt.Sprintf(msgT("'%s' 매크로를 만들었어요. 확인 후 등록해주세요!", "Macro '%s' created. Please review before registering!", lang), macro.Name),
	})
}

// ──────────────────────────────────────────
// 정리 헬퍼 (간단 버전)
// ──────────────────────────────────────────

func performClean(targets []string) int64 {
	var total int64
	for _, target := range targets {
		target = strings.TrimSpace(target)
		switch target {
		case "temp":
			total += cleanDir(os.TempDir())
		case "cache":
			appData, _ := os.UserConfigDir()
			total += cleanDir(filepath.Join(appData, "Local", "Temp"))
		}
	}
	return total
}

func cleanDir(dir string) int64 {
	var freed int64
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	for _, e := range entries {
		p := filepath.Join(dir, e.Name())
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		if info.ModTime().Before(time.Now().Add(-24 * time.Hour)) {
			freed += info.Size()
			os.Remove(p)
		}
	}
	return freed
}
