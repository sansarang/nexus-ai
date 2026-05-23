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
func handleScan(w http.ResponseWriter, r *http.Request)            {}
func handleRepair(w http.ResponseWriter, r *http.Request)          {}
func handleClean(w http.ResponseWriter, r *http.Request)           {}
func handleLicenseActivate(w http.ResponseWriter, r *http.Request) {}
func handleLicenseCheck(w http.ResponseWriter, r *http.Request)    {}

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
func handleAutoClean(w http.ResponseWriter, r *http.Request)       {}
func handlePrivacy(w http.ResponseWriter, r *http.Request)         {}
func handleDailyReport(w http.ResponseWriter, r *http.Request)     {}
func handleFolderOpen(w http.ResponseWriter, r *http.Request)      {}

// handlers_security.go stubs
func handleRemoteAccess(w http.ResponseWriter, r *http.Request)    {}
func handleProcessSecurity(w http.ResponseWriter, r *http.Request) {}
func handleHostsCheck(w http.ResponseWriter, r *http.Request)      {}
func handleStartupItems(w http.ResponseWriter, r *http.Request)    {}
func handleDefender(w http.ResponseWriter, r *http.Request)        {}
func handleAccountCheck(w http.ResponseWriter, r *http.Request)    {}

// handlers_system.go stubs
func handleVolume(w http.ResponseWriter, r *http.Request)          {}
func handleBrightness(w http.ResponseWriter, r *http.Request)      {}
func handleWifi(w http.ResponseWriter, r *http.Request)            {}
func handlePower(w http.ResponseWriter, r *http.Request)           {}
func handleLaunchApp(w http.ResponseWriter, r *http.Request)       {}
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

// handlers_advanced.go stubs
func handleDrivers(w http.ResponseWriter, r *http.Request)         {}
func handleRegistryClean(w http.ResponseWriter, r *http.Request)   {}
func handlePowerPlans(w http.ResponseWriter, r *http.Request)      {}
func handleSetPowerPlan(w http.ResponseWriter, r *http.Request)    {}
func handleNetworkAnalysis(w http.ResponseWriter, r *http.Request) {}
func handleRestoreCreate(w http.ResponseWriter, r *http.Request)   {}
func handleDiskCheck(w http.ResponseWriter, r *http.Request)       {}
func handleBrowserClean(w http.ResponseWriter, r *http.Request)    {}
func handleProgramsList(w http.ResponseWriter, r *http.Request)    {}
func handleBootAnalysis(w http.ResponseWriter, r *http.Request)    {}
func handleFocusMode(w http.ResponseWriter, r *http.Request)       {}
func handleNotes(w http.ResponseWriter, r *http.Request) {
	notes := loadNotesMac()
	json200(w, map[string]any{"notes": notes, "success": true})
}

func handleSaveNote(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	json.NewDecoder(r.Body).Decode(&req)
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
func handleDocCompare(w http.ResponseWriter, r *http.Request)      {}
func handleDocFind(w http.ResponseWriter, r *http.Request)         {}

// handlers_vision.go stubs
func handleDeepSearch(w http.ResponseWriter, r *http.Request)      {}
func handleScreenshot(w http.ResponseWriter, r *http.Request)      {}
func handleActiveWindow(w http.ResponseWriter, r *http.Request)    {}
func handleOCRClipboard(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{
		"success": false,
		"message": "OCR 클립보드 기능은 Windows에서만 사용 가능합니다.",
		"text":    "",
	})
}

// handlers_journal.go stubs
func handleJournalToday(w http.ResponseWriter, r *http.Request)    {}
func handleJournalGenerate(w http.ResponseWriter, r *http.Request) {}
func handleJournalHistory(w http.ResponseWriter, r *http.Request)  {}

// handlers_macro.go stubs
func handleMacroList(w http.ResponseWriter, r *http.Request)       {}
func handleMacroCreate(w http.ResponseWriter, r *http.Request)     {}
func handleMacroRun(w http.ResponseWriter, r *http.Request)        {}
func handleMacroDelete(w http.ResponseWriter, r *http.Request)     {}
func handleMacroParse(w http.ResponseWriter, r *http.Request)      {}

// handlers_report.go stubs
func handleReportGenerate(w http.ResponseWriter, r *http.Request)  {}
func handleReportEmail(w http.ResponseWriter, r *http.Request)     {}
func handleReportSchedule(w http.ResponseWriter, r *http.Request)  {}
func handleEmailConfig(w http.ResponseWriter, r *http.Request)     {}

// handlers_docsummary.go stubs
func handleDocSummary(w http.ResponseWriter, r *http.Request)      {}
func handleDocExportReport(w http.ResponseWriter, r *http.Request) {}

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
