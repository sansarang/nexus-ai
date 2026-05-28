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
// 드라이버 목록 및 오래된 드라이버 탐지
// ──────────────────────────────────────────

func handleDrivers(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	// Get-PnpDevice 은 느릴 수 있으므로 15초 타임아웃
	out, _ := safePS(15*time.Second, `Get-PnpDevice | Where-Object {$_.Status -ne 'OK'} | Select-Object FriendlyName,Status,Class | ConvertTo-Json -Compress`)

	var problematic []struct {
		FriendlyName string `json:"FriendlyName"`
		Status       string `json:"Status"`
		Class        string `json:"Class"`
	}
	json.Unmarshal(out, &problematic)

	allOut, _ := safePS(15*time.Second, `(Get-PnpDevice | Measure-Object).Count`)
	total := 0
	fmt.Sscanf(strings.TrimSpace(string(allOut)), "%d", &total)

	type DriverItem struct {
		Name   string `json:"name"`
		Status string `json:"status"`
		Class  string `json:"class"`
		Risk   string `json:"risk"`
	}

	var items []DriverItem
	for _, d := range problematic {
		risk := "medium"
		if strings.EqualFold(d.Status, "Error") {
			risk = "high"
		}
		items = append(items, DriverItem{Name: d.FriendlyName, Status: d.Status, Class: d.Class, Risk: risk})
	}

	score := 100 - len(items)*10
	if score < 0 {
		score = 0
	}
	json200(w, map[string]any{
		"total":        total,
		"problematic":  items,
		"problem_count": len(items),
		"score":        score,
		"message":      fmt.Sprintf(msgT("전체 %d개 드라이버 중 %d개에 문제가 있어요", "Found %d driver issues out of %d total", lang), total, len(items)),
	})
}

// ──────────────────────────────────────────
// 레지스트리 정리
// ──────────────────────────────────────────

func handleRegistryClean(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	// 무효 .exe 참조 제거 (MRU / 최근 파일 목록)
	script := `
$cleaned = 0
$paths = @(
  'HKCU:\Software\Microsoft\Windows\CurrentVersion\Explorer\RecentDocs',
  'HKCU:\Software\Microsoft\Windows\CurrentVersion\Explorer\RunMRU',
  'HKCU:\Software\Microsoft\Windows\CurrentVersion\Explorer\TypedPaths'
)
foreach ($p in $paths) {
  if (Test-Path $p) {
    Remove-Item -Path $p -Recurse -Force -EA SilentlyContinue
    $cleaned++
  }
}
$cleaned
`
	out, _ := execPS(script)
	cleaned := 0
	fmt.Sscanf(strings.TrimSpace(string(out)), "%d", &cleaned)

	json200(w, map[string]any{
		"success":       true,
		"cleaned_keys":  cleaned,
		"message":       fmt.Sprintf(msgT("레지스트리 %d개 항목 정리 완료 ✅", "Cleaned %d registry entries ✅", lang), cleaned),
	})
}

// ──────────────────────────────────────────
// 전원 계획 관리
// ──────────────────────────────────────────

func handlePowerPlans(w http.ResponseWriter, r *http.Request) {
	out, _ := execPS(`powercfg /list`)

	lines := strings.Split(string(out), "\n")
	type Plan struct {
		Name    string `json:"name"`
		GUID    string `json:"guid"`
		Active  bool   `json:"active"`
	}
	var plans []Plan
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "전원 구성표 GUID:") && !strings.Contains(line, "Power Scheme GUID:") {
			continue
		}
		active := strings.Contains(line, "*")
		// Extract GUID
		parts := strings.Fields(line)
		guid := ""
		name := ""
		for i, p := range parts {
			if len(p) == 36 && strings.Count(p, "-") == 4 {
				guid = p
				if i+1 < len(parts) {
					name = strings.Trim(strings.Join(parts[i+1:], " "), "()")
				}
			}
		}
		if guid != "" {
			plans = append(plans, Plan{Name: name, GUID: guid, Active: active})
		}
	}

	json200(w, map[string]any{"plans": plans, "count": len(plans)})
}

func handleSetPowerPlan(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		GUID string `json:"guid"`
		Name string `json:"name"` // balanced | performance | powersaver
	}
	json.NewDecoder(r.Body).Decode(&req)

	guid := req.GUID
	if guid == "" {
		switch strings.ToLower(req.Name) {
		case "performance", "고성능":
			guid = "8c5e7fda-e8bf-4a96-9a85-a6e23a8c635c"
		case "powersaver", "절전":
			guid = "a1841308-3541-4fab-bc81-f71556f20b4a"
		default: // balanced
			guid = "381b4222-f694-41f0-9685-ff5bb260df2e"
		}
	}

	err := execPSRun(fmt.Sprintf("powercfg /setactive %s", guid))
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": msgT("전원 계획 변경 실패", "Failed to change power plan", lang)})
		return
	}
	json200(w, map[string]any{"success": true, "guid": guid,
		"message": msgT("전원 계획을 변경했어요 ⚡", "Power plan changed ⚡", lang)})
}

// ──────────────────────────────────────────
// 네트워크 상세 분석
// ──────────────────────────────────────────

func handleNetworkAnalysis(w http.ResponseWriter, r *http.Request) {
	// 어댑터 정보 (10초 타임아웃)
	adapterOut, _ := execPS(`Get-NetAdapter | Where-Object {$_.Status -eq 'Up'} | Select-Object Name,InterfaceDescription,LinkSpeed,MacAddress | ConvertTo-Json -Compress`)

	var adapters []struct {
		Name                 string `json:"Name"`
		InterfaceDescription string `json:"InterfaceDescription"`
		LinkSpeed            uint64 `json:"LinkSpeed"`
		MacAddress           string `json:"MacAddress"`
	}
	json.Unmarshal(adapterOut, &adapters)

	// DNS 서버
	dnsOut, _ := execPS(`(Get-DnsClientServerAddress -AddressFamily IPv4 | Where-Object {$_.ServerAddresses} | Select-Object -First 1).ServerAddresses -join ','`)
	dnsServers := strings.TrimSpace(string(dnsOut))

	// 외부 IP (3초 타임아웃 명시)
	ipOut, _ := execPS(`try { (Invoke-WebRequest -Uri 'https://api.ipify.org' -UseBasicParsing -TimeoutSec 3).Content } catch { '' }`)
	publicIP := strings.TrimSpace(string(ipOut))

	// ping 지연
	pingOut, _ := execPS(`(Test-Connection -ComputerName 8.8.8.8 -Count 1 -EA SilentlyContinue).ResponseTime`)
	ping := strings.TrimSpace(string(pingOut))

	type AdapterInfo struct {
		Name       string `json:"name"`
		Desc       string `json:"desc"`
		SpeedMbps  float64 `json:"speed_mbps"`
		MacAddress string `json:"mac_address"`
	}
	var adapterList []AdapterInfo
	for _, a := range adapters {
		adapterList = append(adapterList, AdapterInfo{
			Name: a.Name, Desc: a.InterfaceDescription,
			SpeedMbps:  float64(a.LinkSpeed) / 1_000_000,
			MacAddress: a.MacAddress,
		})
	}

	json200(w, map[string]any{
		"adapters":    adapterList,
		"dns_servers": dnsServers,
		"public_ip":   publicIP,
		"ping_ms":     ping,
		"connected":   len(adapterList) > 0,
	})
}

// ──────────────────────────────────────────
// 시스템 복구 포인트 생성
// ──────────────────────────────────────────

func handleRestoreCreate(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		Description string `json:"description"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Description == "" {
		req.Description = fmt.Sprintf("Nexus 자동 복구 포인트 %s", time.Now().Format("2006-01-02 15:04"))
	}

	script := fmt.Sprintf(`Checkpoint-Computer -Description "%s" -RestorePointType "APPLICATION_INSTALL"`, req.Description)
	err := execPSRun(script)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false,
			"message": msgT("복구 포인트 생성 실패 (시스템 보호 활성화 필요)", "Failed to create restore point (system protection must be enabled)", lang)})
		return
	}
	json200(w, map[string]any{"success": true, "description": req.Description,
		"message": fmt.Sprintf(msgT("복구 포인트 생성 완료: %s", "Restore point created: %s", lang), req.Description)})
}

// ──────────────────────────────────────────
// 디스크 검사 (CHKDSK) 예약
// ──────────────────────────────────────────

func handleDiskCheck(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		Drive string `json:"drive"` // default "C:"
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Drive == "" {
		req.Drive = "C:"
	}

	// 즉시 실행은 수시간 소요 가능 — 항상 재시작 예약 방식으로만 처리
	execPSRun(fmt.Sprintf(`chkntfs /c %s`, req.Drive))
	json200(w, map[string]any{"success": true, "scheduled": true,
		"message": fmt.Sprintf(msgT("%s 디스크 검사가 다음 재시작 시 자동으로 실행됩니다 ✅", "Disk check for %s scheduled for next restart ✅", lang), req.Drive)})
}

// ──────────────────────────────────────────
// 브라우저 데이터 고급 정리
// ──────────────────────────────────────────

func handleBrowserClean(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		Browsers []string `json:"browsers"` // chrome | edge | firefox | all
		Targets  []string `json:"targets"`  // cache | history | cookies | downloads
	}
	json.NewDecoder(r.Body).Decode(&req)

	if len(req.Browsers) == 0 {
		req.Browsers = []string{"chrome", "edge"}
	}
	if len(req.Targets) == 0 {
		req.Targets = []string{"cache"}
	}

	home, _ := os.UserHomeDir()
	type CleanResult struct {
		Browser string `json:"browser"`
		Target  string `json:"target"`
		FreedMB float64 `json:"freed_mb"`
	}
	var results []CleanResult
	var totalFreed int64

	pathMap := map[string]map[string]string{
		"chrome": {
			"cache":     filepath.Join(home, `AppData\Local\Google\Chrome\User Data\Default\Cache`),
			"history":   filepath.Join(home, `AppData\Local\Google\Chrome\User Data\Default`),
			"cookies":   filepath.Join(home, `AppData\Local\Google\Chrome\User Data\Default`),
		},
		"edge": {
			"cache":   filepath.Join(home, `AppData\Local\Microsoft\Edge\User Data\Default\Cache`),
			"history": filepath.Join(home, `AppData\Local\Microsoft\Edge\User Data\Default`),
			"cookies": filepath.Join(home, `AppData\Local\Microsoft\Edge\User Data\Default`),
		},
		"firefox": {
			"cache":   filepath.Join(home, `AppData\Local\Mozilla\Firefox\Profiles`),
		},
	}

	for _, br := range req.Browsers {
		if br == "all" {
			req.Browsers = []string{"chrome", "edge", "firefox"}
			break
		}
	}

	for _, br := range req.Browsers {
		dirs, ok := pathMap[strings.ToLower(br)]
		if !ok {
			continue
		}
		for _, tgt := range req.Targets {
			dir, ok := dirs[tgt]
			if !ok {
				continue
			}
			if tgt == "cache" {
				freed := dirSize(dir)
				os.RemoveAll(dir)
				os.MkdirAll(dir, 0755)
				totalFreed += freed
				results = append(results, CleanResult{
					Browser: br, Target: tgt, FreedMB: float64(freed) / (1 << 20),
				})
			} else {
				var files []string
				if tgt == "history" {
					files = []string{"History", "History-journal"}
				} else if tgt == "cookies" {
					files = []string{"Cookies", "Cookies-journal"}
				}
				var freed int64
				for _, f := range files {
					p := filepath.Join(dir, f)
					if info, err := os.Stat(p); err == nil {
						freed += info.Size()
						os.Remove(p)
					}
				}
				totalFreed += freed
				results = append(results, CleanResult{
					Browser: br, Target: tgt, FreedMB: float64(freed) / (1 << 20),
				})
			}
		}
	}

	json200(w, map[string]any{
		"results":    results,
		"total_mb":   float64(totalFreed) / (1 << 20),
		"total_freed": formatBytes(totalFreed),
		"message":    fmt.Sprintf(msgT("브라우저 데이터 정리 완료: %s 확보", "Browser data cleaned: %s freed", lang), formatBytes(totalFreed)),
	})
}

// ──────────────────────────────────────────
// 설치된 프로그램 목록
// ──────────────────────────────────────────

func handleProgramsList(w http.ResponseWriter, r *http.Request) {
	out, _ := execPS(`Get-Package | Select-Object Name,Version,Source | Sort-Object Name | ConvertTo-Json -Compress`)

	var programs []struct {
		Name    string `json:"Name"`
		Version string `json:"Version"`
		Source  string `json:"Source"`
	}
	json.Unmarshal(out, &programs)

	if len(programs) == 0 {
		// WMIC fallback — 30초 타임아웃 적용
		wmicOut, _ := execPS(`wmic product get name,version /format:csv`)
		lines := strings.Split(string(wmicOut), "\n")
		for _, line := range lines[2:] {
			parts := strings.Split(strings.TrimSpace(line), ",")
			if len(parts) >= 3 && parts[1] != "" {
				programs = append(programs, struct {
					Name    string `json:"Name"`
					Version string `json:"Version"`
					Source  string `json:"Source"`
				}{Name: parts[1], Version: parts[2]})
			}
		}
	}

	type ProgramItem struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	var items []ProgramItem
	for _, p := range programs {
		items = append(items, ProgramItem{Name: p.Name, Version: p.Version})
	}

	json200(w, map[string]any{
		"programs": items,
		"total":    len(items),
	})
}

// ──────────────────────────────────────────
// 부팅 시간 분석
// ──────────────────────────────────────────

func handleBootAnalysis(w http.ResponseWriter, r *http.Request) {
	// 최근 부팅 이벤트 (10초 타임아웃)
	out, _ := safePS(10*time.Second, `Get-WinEvent -FilterHashtable @{LogName='System';Id=12} -MaxEvents 5 -EA SilentlyContinue | Select-Object TimeCreated,Message | ConvertTo-Json -Compress`)

	var events []struct {
		TimeCreated string `json:"TimeCreated"`
		Message     string `json:"Message"`
	}
	json.Unmarshal(out, &events)

	// 마지막 부팅 시간
	bootOut, _ := safePS(10*time.Second, `(Get-Date) - (gcim Win32_OperatingSystem).LastBootUpTime | Select-Object -ExpandProperty TotalMinutes`)
	uptime := strings.TrimSpace(string(bootOut))

	// 시작 프로그램 수
	startupOut, _ := safePS(10*time.Second, `(Get-CimInstance Win32_StartupCommand | Measure-Object).Count`)
	startupCount := strings.TrimSpace(string(startupOut))

	type BootEvent struct {
		Time    string `json:"time"`
		Message string `json:"message"`
	}
	var bootHistory []BootEvent
	for _, e := range events {
		bootHistory = append(bootHistory, BootEvent{Time: e.TimeCreated, Message: e.Message})
	}

	json200(w, map[string]any{
		"uptime_minutes":  uptime,
		"startup_count":   startupCount,
		"recent_boots":    bootHistory,
		"message":         msgT(fmt.Sprintf("현재 가동 시간: %s분, 시작 프로그램: %s개", uptime, startupCount), fmt.Sprintf("Uptime: %s min, Startup programs: %s", uptime, startupCount), getLang(r)),
	})
}

// ──────────────────────────────────────────
// 생산성: 집중 모드
// ──────────────────────────────────────────

func handleFocusMode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action   string `json:"action"`   // on | off
		Duration int    `json:"duration"` // minutes
	}
	json.NewDecoder(r.Body).Decode(&req)
	lang := getLang(r)

	if req.Action == "off" {
		// 알림 다시 켜기
		execPSRun(`Set-ItemProperty -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Notifications\Settings' -Name 'NOC_GLOBAL_SETTING_TOASTS_ENABLED' -Value 1 -EA SilentlyContinue`)
		json200(w, map[string]any{"success": true, "active": false, "message": msgT("집중 모드 해제 완료 🔔", "Focus mode disabled 🔔", lang)})
		return
	}

	// 알림 끄기 (방해 금지)
	execPSRun(`Set-ItemProperty -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Notifications\Settings' -Name 'NOC_GLOBAL_SETTING_TOASTS_ENABLED' -Value 0 -EA SilentlyContinue`)

	if req.Duration == 0 {
		req.Duration = 25
	}
	json200(w, map[string]any{
		"success":  true,
		"active":   true,
		"duration": req.Duration,
		"message":  msgT(fmt.Sprintf("집중 모드 시작! %d분 동안 알림이 차단됩니다 🎯", req.Duration), fmt.Sprintf("Focus mode on! Notifications blocked for %d min 🎯", req.Duration), lang),
	})
}

// ──────────────────────────────────────────
// 클립보드 히스토리
// ──────────────────────────────────────────

type clipEntry struct {
	Text      string `json:"text"`
	Timestamp string `json:"timestamp"`
}

func clipboardHistoryPath() string {
	dir := filepath.Join(os.Getenv("APPDATA"), "Nexus")
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "clipboard_history.json")
}

func loadClipboardHistory() []clipEntry {
	data, err := os.ReadFile(clipboardHistoryPath())
	if err != nil {
		return nil
	}
	var entries []clipEntry
	json.Unmarshal(data, &entries)
	return entries
}

func saveClipboardHistory(entries []clipEntry) {
	data, _ := json.Marshal(entries)
	os.WriteFile(clipboardHistoryPath(), data, 0644)
}

// 클립보드 변경 감지 → 히스토리 자동 저장 (30초 폴링)
var lastClipText string

func startClipboardMonitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		out, err := execPS(`Get-Clipboard -Format Text`)
		if err != nil {
			continue
		}
		text := strings.TrimSpace(string(out))
		if text == "" || text == lastClipText {
			continue
		}
		lastClipText = text
		entries := loadClipboardHistory()
		// 중복 제거
		filtered := make([]clipEntry, 0, len(entries))
		for _, e := range entries {
			if e.Text != text {
				filtered = append(filtered, e)
			}
		}
		// 최신을 맨 앞에
		filtered = append([]clipEntry{{Text: text, Timestamp: time.Now().Format(time.RFC3339)}}, filtered...)
		// 최대 50개 유지
		if len(filtered) > 50 {
			filtered = filtered[:50]
		}
		saveClipboardHistory(filtered)
	}
}

func handleClipboard(w http.ResponseWriter, r *http.Request) {
	out, _ := execPS(`Get-Clipboard -Format Text`)
	current := strings.TrimSpace(string(out))

	// 현재 클립보드를 히스토리에 추가 (변경된 경우)
	if current != "" && current != lastClipText {
		lastClipText = current
		entries := loadClipboardHistory()
		filtered := make([]clipEntry, 0, len(entries))
		for _, e := range entries {
			if e.Text != current {
				filtered = append(filtered, e)
			}
		}
		filtered = append([]clipEntry{{Text: current, Timestamp: time.Now().Format(time.RFC3339)}}, filtered...)
		if len(filtered) > 50 {
			filtered = filtered[:50]
		}
		saveClipboardHistory(filtered)
	}

	json200(w, map[string]any{
		"current": current,
		"tip":     "Windows + V 로 클립보드 히스토리를 볼 수 있어요",
	})
}

// GET  /api/clipboard/history — 히스토리 목록
// DELETE /api/clipboard/history — 히스토리 삭제
func handleClipboardHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodDelete {
		os.Remove(clipboardHistoryPath())
		lastClipText = ""
		json200(w, map[string]any{"success": true, "message": "클립보드 히스토리 삭제됨"})
		return
	}
	entries := loadClipboardHistory()
	if entries == nil {
		entries = []clipEntry{}
	}
	json200(w, map[string]any{"success": true, "history": entries, "total": len(entries)})
}

// ──────────────────────────────────────────
// 메모 저장/조회
// ──────────────────────────────────────────

type NoteItem struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	Created string `json:"created"`
}

func notesPath() string {
	dir := filepath.Join(os.Getenv("APPDATA"), "Nexus")
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "notes.json")
}

func loadNotes() []NoteItem {
	data, err := os.ReadFile(notesPath())
	if err != nil {
		return []NoteItem{}
	}
	var notes []NoteItem
	json.Unmarshal(data, &notes)
	return notes
}

func saveNotes(notes []NoteItem) error {
	data, _ := json.Marshal(notes)
	return os.WriteFile(notesPath(), data, 0644)
}

func handleNotes(w http.ResponseWriter, r *http.Request) {
	notes := loadNotes()
	json200(w, map[string]any{"notes": notes, "total": len(notes)})
}

func handleSaveNote(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content string `json:"content"`
		Delete  string `json:"delete_id"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	notes := loadNotes()

	if req.Delete != "" {
		filtered := notes[:0]
		for _, n := range notes {
			if n.ID != req.Delete {
				filtered = append(filtered, n)
			}
		}
		saveNotes(filtered)
		json200(w, map[string]any{"success": true, "message": msgT("메모 삭제 완료", "Note deleted", getLang(r))})
		return
	}

	if req.Content == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("메모 내용을 입력해주세요", "Please enter note content", getLang(r))})
		return
	}

	note := NoteItem{
		ID:      fmt.Sprintf("%d", time.Now().UnixNano()),
		Content: req.Content,
		Created: time.Now().Format("2006-01-02 15:04"),
	}
	notes = append([]NoteItem{note}, notes...)
	if len(notes) > 100 {
		notes = notes[:100]
	}
	saveNotes(notes)
	json200(w, map[string]any{"success": true, "note": note, "message": msgT("메모 저장 완료 📝", "Note saved 📝", getLang(r))})
}
