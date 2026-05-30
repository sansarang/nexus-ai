//go:build !windows

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// handlers.go stubs
func handleScan(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"score": 100, "issues": []any{}, "message": "PC 스캔 완료 (개발 환경)"})
}
func handleRepair(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": true, "freed": 0, "message": "수리 완료 (개발 환경)"})
}
func handleClean(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"freed": 0, "message": "정리 완료 (개발 환경)"})
}
func handleLicenseActivate(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"valid": true, "message": "오프라인 인증 완료 (개발 환경)"})
}
func handleLicenseCheck(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"valid": true, "key": "DEV-MODE"})
}

func handleStats(w http.ResponseWriter, r *http.Request) {
	stats := map[string]any{}

	// CPU
	if out, err := exec.Command("sh", "-c", "top -l 1 -n 0 | grep 'CPU usage'").Output(); err == nil {
		line := string(out)
		// "CPU usage: 5.11% user, 9.58% sys, 85.30% idle"
		if idx := strings.Index(line, "idle"); idx > 0 {
			parts := strings.Fields(line[:idx])
			if len(parts) > 0 {
				idleStr := strings.TrimSuffix(parts[len(parts)-1], "%")
				if idle, err := strconv.ParseFloat(idleStr, 64); err == nil {
					stats["cpu_percent"] = 100 - idle
				}
			}
		}
	}

	// RAM via vm_stat
	if out, err := exec.Command("vm_stat").Output(); err == nil {
		pageSize := 16384.0
		vals := map[string]float64{}
		for _, line := range strings.Split(string(out), "\n") {
			if strings.Contains(line, ":") {
				parts := strings.SplitN(line, ":", 2)
				key := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(parts[1]), "."))
				if n, err := strconv.ParseFloat(val, 64); err == nil {
					vals[key] = n * pageSize
				}
			}
		}
		total := vals["Pages free"] + vals["Pages active"] + vals["Pages inactive"] + vals["Pages speculative"] + vals["Pages wired down"]
		if total > 0 {
			used := total - vals["Pages free"] - vals["Pages speculative"]
			stats["memory_percent"] = used / total * 100
			stats["memory_used_gb"] = used / (1 << 30)
			stats["memory_total_gb"] = total / (1 << 30)
		}
	}

	// Disk
	if out, err := exec.Command("df", "-H", "/").Output(); err == nil {
		lines := strings.Split(string(out), "\n")
		if len(lines) > 1 {
			fields := strings.Fields(lines[1])
			if len(fields) >= 5 {
				pct := strings.TrimSuffix(fields[4], "%")
				if p, err := strconv.ParseFloat(pct, 64); err == nil {
					stats["disk_percent"] = p
				}
				stats["disk_used"] = fields[2]
				stats["disk_total"] = fields[1]
			}
		}
	}

	stats["success"] = true
	json200(w, stats)
}
func handleAutoClean(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": true, "freed": 0, "message": "자동 정리 완료 (개발 환경)"})
}
func handlePrivacy(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": false, "message": "개인정보 보호 설정은 Windows에서만 사용 가능합니다."})
}
func handleDailyReport(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"date": time.Now().Format("2006-01-02"), "pc_score": 85, "cpu_avg": 30.0, "mem_avg": 55.0, "disk_free_gb": 50.0, "recommendations": []string{"개발 환경입니다."}, "predictions": []any{}})
}
func handleFolderOpen(w http.ResponseWriter, r *http.Request) {
	var req struct{ Path string `json:"path"` }
	tryDecodeBody(r, &req)
	if req.Path == "" {
		req.Path = os.Getenv("HOME")
	}
	exec.Command("open", req.Path).Start()
	json200(w, map[string]any{"success": true, "path": req.Path, "message": "폴더를 열었어요"})
}

// handlers_security.go stubs
func handleRemoteAccess(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"found": false, "tools": []any{}, "rdp_open": false, "score": 100, "message": "원격 접속 도구 없음 (개발 환경)"})
}
func handleProcessSecurity(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"suspicious_processes": []any{}, "open_ports": []any{}, "score": 100, "message": "수상한 프로세스 없음 (개발 환경)"})
}
func handleHostsCheck(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"score": 100, "modified": false, "entries": 0, "suspicious": []any{}, "message": "hosts 파일 정상 (개발 환경)"})
}
func handleStartupItems(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"items": []any{}, "total": 0, "suspicious_count": 0, "message": "시작 항목 없음 (개발 환경)"})
}
func handleDefender(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": false, "message": "Windows Defender는 Windows에서만 사용 가능합니다.", "antivirus_enabled": false, "realtime_protection": false, "score": 0, "issues": []string{"Windows 전용 기능"}})
}
func handleAccountCheck(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"total": 1, "suspicious": []any{}, "suspicious_count": 0, "score": 100, "message": "계정 정상 (개발 환경)"})
}

// handlers_system.go stubs
func handleVolume(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action string `json:"action"`
		Value  int    `json:"value"`
	}
	tryDecodeBody(r, &req)
	if req.Action == "get" {
		out, _ := exec.Command("osascript", "-e", "output volume of (get volume settings)").Output()
		vol := strings.TrimSpace(string(out))
		json200(w, map[string]any{"success": true, "volume": vol, "message": "현재 볼륨: " + vol + "%"})
		return
	}
	val := fmt.Sprintf("%d", req.Value)
	if req.Action == "mute" {
		exec.Command("osascript", "-e", "set volume output muted true").Run()
		json200(w, map[string]any{"success": true, "message": "음소거됐어요 🔇"})
		return
	}
	exec.Command("osascript", "-e", "set volume output volume "+val).Run()
	json200(w, map[string]any{"success": true, "volume": req.Value, "message": fmt.Sprintf("볼륨을 %d%%로 설정했어요 🔊", req.Value)})
}
func handleBrightness(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": false, "message": "밝기 조절은 Windows 노트북에서만 지원됩니다."})
}
func handleWifi(w http.ResponseWriter, r *http.Request) {
	var req struct{ Action string `json:"action"` }
	tryDecodeBody(r, &req)
	out, _ := exec.Command("networksetup", "-getairportnetwork", "en0").Output()
	status := strings.TrimSpace(string(out))
	json200(w, map[string]any{"success": true, "status": status, "message": "Wi-Fi 상태: " + status})
}
func handlePower(w http.ResponseWriter, r *http.Request) {
	var req struct{ Action string `json:"action"` }
	tryDecodeBody(r, &req)
	switch req.Action {
	case "sleep":
		exec.Command("pmset", "sleepnow").Start()
		json200(w, map[string]any{"success": true, "message": "절전 모드로 전환합니다 😴"})
	case "shutdown":
		json200(w, map[string]any{"success": false, "message": "종료는 안전을 위해 직접 실행해주세요."})
	default:
		json200(w, map[string]any{"success": false, "message": "지원하지 않는 전원 명령입니다."})
	}
}
func handleLaunchApp(w http.ResponseWriter, r *http.Request) {
	var req struct{ App string `json:"app"` }
	tryDecodeBody(r, &req)
	if req.App == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "앱 이름을 입력해주세요"})
		return
	}
	err := exec.Command("open", "-a", req.App).Run()
	if err != nil {
		json200(w, map[string]any{"success": false, "message": req.App + " 앱을 찾을 수 없어요"})
		return
	}
	json200(w, map[string]any{"success": true, "message": req.App + " 앱을 실행했어요 🚀"})
}
func handleProcessTop(w http.ResponseWriter, r *http.Request) {
	out, err := exec.Command("sh", "-c", "ps aux --sort=-%cpu 2>/dev/null || ps aux | sort -rk3 | head -10").Output()
	if err != nil {
		out, err = exec.Command("sh", "-c", "ps aux | sort -rk3 | head -10").Output()
	}
	type Proc struct {
		PID  string `json:"pid"`
		Name string `json:"name"`
		CPU  string `json:"cpu"`
		Mem  string `json:"mem"`
	}
	var procs []Proc
	if err == nil {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines[1:] {
			fields := strings.Fields(line)
			if len(fields) < 11 {
				continue
			}
			name := fields[10]
			if idx := strings.LastIndex(name, "/"); idx >= 0 {
				name = name[idx+1:]
			}
			procs = append(procs, Proc{PID: fields[1], CPU: fields[2], Mem: fields[3], Name: name})
			if len(procs) >= 10 {
				break
			}
		}
	}
	json200(w, map[string]any{"processes": procs, "success": true})
}

// handlers_stats_collector.go stubs
type StatEntry struct {
	Time     string  `json:"time"`
	CPU      float64 `json:"cpu"`
	Mem      float64 `json:"mem"`
	DiskFree float64 `json:"disk_free"`
}

func startStatsCollector()                    {}
func collectAndSaveStat()                     {}
func loadDailyStats(date string) []StatEntry  { return nil }

// handlers_advanced.go stubs
func handleDrivers(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": false, "message": "드라이버 관리는 Windows에서만 사용 가능합니다.", "drivers": []any{}})
}
func handleRegistryClean(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": false, "message": "레지스트리 정리는 Windows에서만 사용 가능합니다.", "freed": 0})
}
func handlePowerPlans(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": false, "message": "전원 계획은 Windows에서만 사용 가능합니다.", "plans": []any{}, "active": ""})
}
func handleSetPowerPlan(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": false, "message": "전원 계획 설정은 Windows에서만 사용 가능합니다."})
}
func handleNetworkAnalysis(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": true, "connected": true, "latency_ms": 0, "download_mbps": 0, "upload_mbps": 0, "message": "네트워크 분석 (개발 환경)"})
}
func handleRestoreCreate(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": false, "message": "복원 지점 생성은 Windows에서만 사용 가능합니다."})
}
func handleDiskCheck(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": true, "errors": 0, "message": "디스크 검사 완료 (개발 환경)", "health": "good"})
}
func handleBrowserClean(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": true, "freed": 0, "message": "브라우저 정리 완료 (개발 환경)"})
}
func handleProgramsList(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": true, "programs": []any{}, "total": 0, "message": "프로그램 목록 (개발 환경)"})
}
func handleBootAnalysis(w http.ResponseWriter, r *http.Request) {
	out, _ := exec.Command("sh", "-c", "uptime | awk '{print $3, $4}'").Output()
	uptime := strings.TrimSpace(string(out))
	json200(w, map[string]any{"success": true, "boot_time_sec": 0, "uptime": uptime, "slow_items": []any{}, "message": "부팅 분석 (개발 환경)"})
}
func handleFocusMode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action   string   `json:"action"`
		Duration int      `json:"duration"`
		Block    []string `json:"block"`
	}
	tryDecodeBody(r, &req)
	json200(w, map[string]any{"success": true, "action": req.Action, "duration": req.Duration, "message": "집중 모드 (개발 환경)"})
}
func handleNotes(w http.ResponseWriter, r *http.Request) {
	notes := loadNotesMac()
	json200(w, map[string]any{"notes": notes, "success": true})
}

func handleSaveNote(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	tryDecodeBody(r, &req)
	if req.Content == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "content 필요"})
		return
	}
	notes := loadNotesMac()
	note := map[string]any{
		"id":      fmt.Sprintf("%d", time.Now().UnixMilli()),
		"title":   req.Title,
		"content": req.Content,
		"created": time.Now().Format("2006-01-02 15:04"),
	}
	notes = append([]map[string]any{note}, notes...)
	if len(notes) > 100 {
		notes = notes[:100]
	}
	saveNotesMac(notes)
	json200(w, map[string]any{"success": true, "message": msgT("노트가 저장됐어요", "Note saved", getLang(r)), "note": note})
}

func loadNotesMac() []map[string]any {
	path := notesPathMac()
	data, err := os.ReadFile(path)
	if err != nil {
		return []map[string]any{}
	}
	var notes []map[string]any
	json.Unmarshal(data, &notes)
	return notes
}

func saveNotesMac(notes []map[string]any) {
	data, _ := json.Marshal(notes)
	os.WriteFile(notesPathMac(), data, 0644)
}

func notesPathMac() string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".nexus")
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "notes.json")
}

// handlers_docs.go stubs
func handleDocCompare(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": false, "message": "문서 비교는 파일 첨부가 필요합니다. (개발 환경)", "differences": []any{}})
}
func handleDocFind(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": true, "files": []any{}, "total": 0, "message": "파일 검색 완료 (개발 환경)"})
}

// handlers_vision.go stubs
func handleDeepSearch(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": true, "results": []any{}, "total": 0, "message": "검색 완료 (개발 환경)"})
}
func handleScreenshot(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": false, "message": "스크린샷 캡처는 Windows에서만 사용 가능합니다.", "path": ""})
}
func handleActiveWindow(w http.ResponseWriter, r *http.Request) {
	out, _ := exec.Command("osascript", "-e", "tell application \"System Events\" to get name of first application process whose frontmost is true").Output()
	title := strings.TrimSpace(string(out))
	if title == "" {
		title = "Unknown"
	}
	json200(w, map[string]any{"success": true, "title": title, "process": title})
}
func handleOCRClipboard(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{
		"success": false,
		"message": "OCR 클립보드 기능은 Windows에서만 사용 가능합니다.",
		"text":    "",
	})
}

// handlers_journal.go stubs
func handleJournalToday(w http.ResponseWriter, r *http.Request) {
	today := time.Now().Format("2006-01-02")
	json200(w, map[string]any{
		"success":    true,
		"date":       today,
		"work_hours": 0.0,
		"file_count": 0,
		"app_count":  0,
		"top_app":    "",
		"summary":    "",
		"message":    "오늘 일지 (개발 환경)",
	})
}
func handleJournalGenerate(w http.ResponseWriter, r *http.Request) {
	today := time.Now().Format("2006-01-02")
	json200(w, map[string]any{
		"success": true,
		"date":    today,
		"summary": "개발 환경에서 생성된 일지입니다.",
		"message": "일지가 생성됐어요 ✅",
	})
}
func handleJournalHistory(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": true, "history": []any{}, "total": 0, "message": "일지 기록 (개발 환경)"})
}

// handlers_macro.go stubs
func handleMacroList(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": true, "macros": []any{}, "total": 0})
}
func handleMacroCreate(w http.ResponseWriter, r *http.Request) {
	var req map[string]any
	tryDecodeBody(r, &req)
	name, _ := req["name"].(string)
	if name == "" {
		name = "새 매크로"
	}
	json200(w, map[string]any{"success": true, "message": name + " 매크로가 생성됐어요 ✅", "id": fmt.Sprintf("%d", time.Now().UnixMilli())})
}
func handleMacroRun(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": false, "message": "매크로 실행은 Windows에서만 사용 가능합니다."})
}
func handleMacroDelete(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": true, "message": "매크로가 삭제됐어요"})
}
func handleMacroParse(w http.ResponseWriter, r *http.Request) {
	var req struct{ Text string `json:"text"` }
	tryDecodeBody(r, &req)
	json200(w, map[string]any{"success": true, "steps": []any{}, "name": req.Text, "message": "매크로 파싱 완료 (개발 환경)"})
}

// handlers_report.go stubs
func handleReportGenerate(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	json200(w, map[string]any{
		"success":    true,
		"date":       now.Format("2006-01-02"),
		"pc_score":   85,
		"cpu_avg":    30.0,
		"mem_avg":    55.0,
		"disk_free":  "50 GB",
		"summary":    "PC 상태가 양호합니다. (개발 환경)",
		"message":    "보고서가 생성됐어요 📊",
	})
}
func handleReportEmail(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": false, "message": "이메일 전송은 이메일 설정이 필요합니다."})
}
func handleReportSchedule(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": true, "message": "보고서 예약이 설정됐어요 📅"})
}
func handleEmailConfig(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": true, "smtp_host": "", "smtp_port": 587, "email": "", "configured": false, "message": "이메일 설정을 입력해주세요"})
}

// handlers_docsummary.go stubs
func handleDocSummary(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": true, "summary": "", "key_points": []any{}, "message": "문서 요약 (개발 환경 — 파일 첨부 필요)"})
}
func handleDocExportReport(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": false, "message": "문서 내보내기는 Windows에서만 사용 가능합니다.", "path": ""})
}

// handlers_proactive.go stubs (SSE alert stream) — Mac 실제 구현

type Alert struct {
	ID        string `json:"id"`
	Level     string `json:"level"`
	Title     string `json:"title"`
	Message   string `json:"message"`
	Action    string `json:"action,omitempty"`
	Dismissed bool   `json:"dismissed"`
}

var (
	macAlertClients  = map[chan Alert]struct{}{}
	macAlertMu       sync.RWMutex
	macLatestAlerts  []Alert
)

func publishAlert(a Alert) {
	macAlertMu.Lock()
	macLatestAlerts = append([]Alert{a}, macLatestAlerts...)
	if len(macLatestAlerts) > 20 {
		macLatestAlerts = macLatestAlerts[:20]
	}
	for ch := range macAlertClients {
		select {
		case ch <- a:
		default:
		}
	}
	macAlertMu.Unlock()
}

func handleAlertStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write([]byte("data: {\"type\":\"connected\"}\n\n"))
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	ch := make(chan Alert, 8)
	macAlertMu.Lock()
	macAlertClients[ch] = struct{}{}
	macAlertMu.Unlock()
	defer func() {
		macAlertMu.Lock()
		delete(macAlertClients, ch)
		macAlertMu.Unlock()
	}()
	for {
		select {
		case <-r.Context().Done():
			return
		case a := <-ch:
			data, _ := json.Marshal(a)
			fmt.Fprintf(w, "data: %s\n\n", data)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
}

// startMacProactiveMonitor: Mac 시스템 리소스 모니터링 (30초 간격)
func startMacProactiveMonitor() {
	go func() {
		time.Sleep(10 * time.Second) // 앱 시작 후 10초 대기
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			checkMacSystemStats()
		}
	}()
}

func checkMacSystemStats() {
	eng := IsUserEng()
	// CPU 체크
	out, err := exec.Command("sh", "-c", "top -l 1 -n 0 | grep 'CPU usage'").Output()
	if err == nil {
		line := string(out)
		if idx := strings.Index(line, "idle"); idx > 0 {
			parts := strings.Fields(line[:idx])
			if len(parts) > 0 {
				idleStr := strings.TrimSuffix(parts[len(parts)-1], "%")
				if idle, err2 := strconv.ParseFloat(idleStr, 64); err2 == nil {
					cpu := 100 - idle
					if cpu > 85 {
						var title, msg string
						if eng {
							title = "High CPU Usage"
							msg = fmt.Sprintf("CPU usage is %.0f%%. Consider closing unused apps.", cpu)
						} else {
							title = "CPU 사용량 높음"
							msg = fmt.Sprintf("CPU 사용량이 %.0f%%입니다. 불필요한 앱을 닫아보세요.", cpu)
						}
						publishAlert(Alert{
							ID: fmt.Sprintf("cpu_%d", time.Now().Unix()),
							Level: "warning", Title: title, Message: msg,
						})
					}
				}
			}
		}
	}
	// 디스크 체크
	out2, err2 := exec.Command("df", "-H", "/").Output()
	if err2 == nil {
		lines := strings.Split(string(out2), "\n")
		if len(lines) > 1 {
			fields := strings.Fields(lines[1])
			if len(fields) >= 5 {
				pctStr := strings.TrimSuffix(fields[4], "%")
				if pct, err3 := strconv.ParseFloat(pctStr, 64); err3 == nil && pct > 90 {
					var title, msg string
					if eng {
						title = "Low Disk Space"
						msg = fmt.Sprintf("Disk usage is %.0f%%. Free up space soon.", pct)
					} else {
						title = "디스크 공간 부족"
						msg = fmt.Sprintf("디스크 사용량이 %.0f%%입니다. 정리가 필요합니다.", pct)
					}
					publishAlert(Alert{
						ID: fmt.Sprintf("disk_%d", time.Now().Unix()),
						Level: "warning", Title: title, Message: msg,
					})
				}
			}
		}
	}
}
