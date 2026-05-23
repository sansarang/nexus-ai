//go:build windows

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// ──────────────────────────────────────────
// 보안: 원격 접속 탐지
// ──────────────────────────────────────────

type RemoteToolInfo struct {
	Name   string `json:"name"`
	Status string `json:"status"` // running | not_running
	Risk   string `json:"risk"`   // high | medium | low
}

func handleRemoteAccess(w http.ResponseWriter, r *http.Request) {
	catalog := []struct{ name, proc, risk string }{
		{"TeamViewer", "teamviewer", "medium"},
		{"AnyDesk", "anydesk", "medium"},
		{"Chrome Remote Desktop", "remoting_host", "low"},
		{"UltraVNC", "winvnc", "high"},
		{"TightVNC", "tvnserver", "high"},
		{"AeroAdmin", "aeroadmin", "high"},
		{"Ammyy Admin", "aa_v3", "high"},
		{"LogMeIn", "logmein", "medium"},
		{"RustDesk", "rustdesk", "medium"},
	}

	psOut, _ := execPS(`(Get-Process | Select-Object -ExpandProperty Name) -join ','`)
	runningLower := strings.ToLower(string(psOut))

	result := struct {
		Found   bool             `json:"found"`
		Tools   []RemoteToolInfo `json:"tools"`
		RdpOpen bool             `json:"rdp_open"`
		Score   int              `json:"score"`
	}{Score: 100}

	for _, t := range catalog {
		rt := RemoteToolInfo{Name: t.name, Risk: t.risk}
		if strings.Contains(runningLower, t.proc) {
			rt.Status = "running"
			result.Found = true
			switch t.risk {
			case "high":
				result.Score -= 20
			case "medium":
				result.Score -= 8
			}
		} else {
			rt.Status = "not_running"
		}
		result.Tools = append(result.Tools, rt)
	}

	// RDP 포트 3389 수신 여부
	netOut, _ := execPS(`(Get-NetTCPConnection -LocalPort 3389 -State Listen -EA SilentlyContinue | Measure-Object).Count`)
	if cnt, _ := strconv.Atoi(strings.TrimSpace(string(netOut))); cnt > 0 {
		result.RdpOpen = true
		result.Score -= 10
	}

	if result.Score < 0 {
		result.Score = 0
	}
	json200(w, result)
}

// ──────────────────────────────────────────
// 보안: 수상한 프로세스·열린 포트
// ──────────────────────────────────────────

type SuspProc struct {
	Name   string  `json:"name"`
	PID    int     `json:"pid"`
	CPU    float64 `json:"cpu"`
	MemMB  float64 `json:"mem_mb"`
	Risk   string  `json:"risk"`
	Reason string  `json:"reason"`
}

type OpenPortInfo struct {
	Port   int    `json:"port"`
	State  string `json:"state"`
	PID    int    `json:"pid"`
	Risk   string `json:"risk"`
	Reason string `json:"reason"`
}

func handleProcessSecurity(w http.ResponseWriter, r *http.Request) {
	psOut, _ := execPS(`Get-Process | Sort-Object CPU -Desc | Select-Object -First 20 Name,Id,CPU,WorkingSet | ConvertTo-Json -Compress`)

	var rawProcs []struct {
		Name       string  `json:"Name"`
		Id         int     `json:"Id"`
		CPU        float64 `json:"CPU"`
		WorkingSet int64   `json:"WorkingSet"`
	}
	json.Unmarshal(psOut, &rawProcs)

	blacklist := map[string]string{
		"xmrig": "암호화폐 채굴기", "minerd": "채굴기", "cpuminer": "채굴기",
		"mimikatz": "자격증명 탈취", "meterpreter": "원격 쉘",
		"ncat": "네트워크 백도어",
	}

	var suspProcs []SuspProc
	score := 100

	for _, p := range rawProcs {
		low := strings.ToLower(p.Name)
		reason, risk := "", "low"
		for kw, desc := range blacklist {
			if strings.Contains(low, kw) {
				reason, risk = desc, "high"
				score -= 25
				break
			}
		}
		if p.CPU > 90 && p.Name != "Idle" && risk == "low" {
			reason = fmt.Sprintf("CPU %.0f%% 과부하", p.CPU)
			risk = "medium"
			score -= 5
		}
		if risk != "low" {
			suspProcs = append(suspProcs, SuspProc{
				Name: p.Name, PID: p.Id, CPU: p.CPU,
				MemMB: float64(p.WorkingSet) / (1 << 20), Risk: risk, Reason: reason,
			})
		}
	}

	portOut, _ := execPS(`Get-NetTCPConnection -State Listen | Select-Object LocalPort,State,OwningProcess | Sort-Object LocalPort | Select-Object -First 30 | ConvertTo-Json -Compress`)

	var rawPorts []struct {
		LocalPort     int    `json:"LocalPort"`
		State         string `json:"State"`
		OwningProcess int    `json:"OwningProcess"`
	}
	json.Unmarshal(portOut, &rawPorts)

	dangerPorts := map[int]string{
		4444: "Metasploit 기본 포트", 1337: "해킹 도구 포트",
		31337: "BackOrifice", 12345: "NetBus", 6666: "악성 IRC", 5900: "VNC",
	}

	var openPorts []OpenPortInfo
	for _, p := range rawPorts {
		risk, reason := "low", ""
		if desc, ok := dangerPorts[p.LocalPort]; ok {
			risk, reason = "high", desc
			score -= 15
		}
		openPorts = append(openPorts, OpenPortInfo{
			Port: p.LocalPort, State: p.State, PID: p.OwningProcess, Risk: risk, Reason: reason,
		})
	}

	if score < 0 {
		score = 0
	}
	json200(w, map[string]any{
		"suspicious_processes": suspProcs,
		"open_ports":           openPorts,
		"score":                score,
	})
}

// ──────────────────────────────────────────
// 보안: hosts 파일 변조 탐지
// ──────────────────────────────────────────

func handleHostsCheck(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile(`C:\Windows\System32\drivers\etc\hosts`)
	if err != nil {
		json200(w, map[string]any{"score": 100, "modified": false, "entries": 0, "suspicious": []string{}})
		return
	}

	score := 100
	var suspicious []string
	total := 0
	protected := []string{"microsoft.com", "windowsupdate.com", "google.com", "github.com"}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		total++
		lineLow := strings.ToLower(line)
		for _, d := range protected {
			if strings.Contains(lineLow, d) {
				suspicious = append(suspicious, line)
				score -= 15
			}
		}
		if !strings.HasPrefix(line, "127.") && !strings.HasPrefix(line, "0.0.0.0") && !strings.HasPrefix(line, "::1") {
			suspicious = append(suspicious, line)
			score -= 10
		}
	}

	if score < 0 {
		score = 0
	}
	json200(w, map[string]any{
		"score":      score,
		"modified":   len(suspicious) > 0,
		"entries":    total,
		"suspicious": suspicious,
	})
}

// ──────────────────────────────────────────
// 보안: 시작 프로그램 관리
// ──────────────────────────────────────────

type StartupItem struct {
	Name     string `json:"name"`
	Command  string `json:"command"`
	Location string `json:"location"`
	Risk     string `json:"risk"`
}

func handleStartupItems(w http.ResponseWriter, r *http.Request) {
	out, _ := execPS(`Get-CimInstance Win32_StartupCommand | Select-Object Name,Command,Location | ConvertTo-Json -Compress`)

	var raw []struct {
		Name     string `json:"Name"`
		Command  string `json:"Command"`
		Location string `json:"Location"`
	}
	json.Unmarshal(out, &raw)

	suspKw := []string{"%temp%", "appdata\\roaming", "cmd /c", "powershell -e", "wscript", "cscript"}
	var items []StartupItem
	suspicious := 0

	for _, s := range raw {
		cmdLow := strings.ToLower(s.Command)
		risk := "low"
		for _, kw := range suspKw {
			if strings.Contains(cmdLow, kw) {
				risk = "high"
				suspicious++
				break
			}
		}
		items = append(items, StartupItem{Name: s.Name, Command: s.Command, Location: s.Location, Risk: risk})
	}

	json200(w, map[string]any{
		"items":            items,
		"total":            len(items),
		"suspicious_count": suspicious,
	})
}

// ──────────────────────────────────────────
// 보안: Windows Defender 상태
// ──────────────────────────────────────────

func handleDefender(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	out, _ := execPS(`Get-MpComputerStatus | Select-Object AntivirusEnabled,RealTimeProtectionEnabled,QuickScanAge,FullScanAge,AntivirusSignatureLastUpdated | ConvertTo-Json -Compress`)

	var status struct {
		AntivirusEnabled              bool   `json:"AntivirusEnabled"`
		RealTimeProtectionEnabled     bool   `json:"RealTimeProtectionEnabled"`
		QuickScanAge                  int    `json:"QuickScanAge"`
		FullScanAge                   int    `json:"FullScanAge"`
		AntivirusSignatureLastUpdated string `json:"AntivirusSignatureLastUpdated"`
	}
	json.Unmarshal(out, &status)

	score := 100
	var issues []string

	if !status.AntivirusEnabled {
		score -= 30
		issues = append(issues, msgT("바이러스 백신 비활성화", "Antivirus disabled", lang))
	}
	if !status.RealTimeProtectionEnabled {
		score -= 25
		issues = append(issues, msgT("실시간 보호 꺼짐", "Real-time protection off", lang))
	}
	if status.QuickScanAge > 7 {
		score -= 10
		issues = append(issues, fmt.Sprintf(msgT("마지막 빠른 검사 %d일 전", "Last quick scan %d days ago", lang), status.QuickScanAge))
	}
	if score < 0 {
		score = 0
	}
	json200(w, map[string]any{
		"antivirus_enabled":      status.AntivirusEnabled,
		"realtime_protection":    status.RealTimeProtectionEnabled,
		"quick_scan_age":         status.QuickScanAge,
		"full_scan_age":          status.FullScanAge,
		"signature_last_updated": status.AntivirusSignatureLastUpdated,
		"score":                  score,
		"issues":                 issues,
	})
}

// ──────────────────────────────────────────
// 보안: 이상 계정 탐지
// ──────────────────────────────────────────

func handleAccountCheck(w http.ResponseWriter, r *http.Request) {
	out, _ := execPS(`Get-LocalUser | Select-Object Name,Enabled,LastLogon,PasswordRequired | ConvertTo-Json -Compress`)

	var accounts []struct {
		Name             string `json:"Name"`
		Enabled          bool   `json:"Enabled"`
		LastLogon        string `json:"LastLogon"`
		PasswordRequired bool   `json:"PasswordRequired"`
	}
	json.Unmarshal(out, &accounts)

	score := 100
	var suspicious []map[string]any

	sysAccounts := map[string]bool{
		"administrator": true, "guest": true, "defaultaccount": true,
		"wdagutilityaccount": true,
	}

	for _, a := range accounts {
		nameLow := strings.ToLower(a.Name)
		reason := ""
		if a.Enabled && sysAccounts[nameLow] && nameLow == "administrator" {
			reason = "기본 관리자 계정 활성화"
			score -= 10
		}
		if !a.PasswordRequired && a.Enabled {
			reason = "비밀번호 없는 활성 계정"
			score -= 15
		}
		if reason != "" {
			suspicious = append(suspicious, map[string]any{
				"name": a.Name, "enabled": a.Enabled,
				"last_logon": a.LastLogon, "reason": reason,
			})
		}
	}

	total := len(accounts)
	if score < 0 {
		score = 0
	}
	json200(w, map[string]any{
		"total":            total,
		"suspicious":       suspicious,
		"suspicious_count": len(suspicious),
		"score":            score,
	})
}
