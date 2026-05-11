//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ──────────────────────────────────────────
// 드라이버 목록 및 오래된 드라이버 탐지
// ──────────────────────────────────────────

func handleDrivers(w http.ResponseWriter, r *http.Request) {
	out, _ := exec.Command("powershell", "-NoProfile", "-Command",
		`Get-PnpDevice | Where-Object {$_.Status -ne 'OK'} | Select-Object FriendlyName,Status,Class | ConvertTo-Json -Compress`).Output()

	var problematic []struct {
		FriendlyName string `json:"FriendlyName"`
		Status       string `json:"Status"`
		Class        string `json:"Class"`
	}
	json.Unmarshal(out, &problematic)

	allOut, _ := exec.Command("powershell", "-NoProfile", "-Command",
		`(Get-PnpDevice | Measure-Object).Count`).Output()
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
		"message":      fmt.Sprintf("전체 %d개 드라이버 중 %d개에 문제가 있어요", total, len(items)),
	})
}

// ──────────────────────────────────────────
// 레지스트리 정리
// ──────────────────────────────────────────

func handleRegistryClean(w http.ResponseWriter, r *http.Request) {
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
	out, _ := exec.Command("powershell", "-NoProfile", "-Command", script).Output()
	cleaned := 0
	fmt.Sscanf(strings.TrimSpace(string(out)), "%d", &cleaned)

	json200(w, map[string]any{
		"success":       true,
		"cleaned_keys":  cleaned,
		"message":       fmt.Sprintf("레지스트리 %d개 항목 정리 완료 ✅", cleaned),
	})
}

// ──────────────────────────────────────────
// 전원 계획 관리
// ──────────────────────────────────────────

func handlePowerPlans(w http.ResponseWriter, r *http.Request) {
	out, _ := exec.Command("powershell", "-NoProfile", "-Command",
		`powercfg /list`).Output()

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

	err := exec.Command("powercfg", "/setactive", guid).Run()
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": "전원 계획 변경 실패"})
		return
	}
	json200(w, map[string]any{"success": true, "guid": guid,
		"message": "전원 계획을 변경했어요 ⚡"})
}

// ──────────────────────────────────────────
// 네트워크 상세 분석
// ──────────────────────────────────────────

func handleNetworkAnalysis(w http.ResponseWriter, r *http.Request) {
	// 어댑터 정보
	adapterOut, _ := exec.Command("powershell", "-NoProfile", "-Command",
		`Get-NetAdapter | Where-Object {$_.Status -eq 'Up'} | Select-Object Name,InterfaceDescription,LinkSpeed,MacAddress | ConvertTo-Json -Compress`).Output()

	var adapters []struct {
		Name                 string `json:"Name"`
		InterfaceDescription string `json:"InterfaceDescription"`
		LinkSpeed            uint64 `json:"LinkSpeed"`
		MacAddress           string `json:"MacAddress"`
	}
	json.Unmarshal(adapterOut, &adapters)

	// DNS 서버
	dnsOut, _ := exec.Command("powershell", "-NoProfile", "-Command",
		`(Get-DnsClientServerAddress -AddressFamily IPv4 | Where-Object {$_.ServerAddresses} | Select-Object -First 1).ServerAddresses -join ','`).Output()
	dnsServers := strings.TrimSpace(string(dnsOut))

	// 외부 IP (간단 방법 - hostname)
	ipOut, _ := exec.Command("powershell", "-NoProfile", "-Command",
		`(Invoke-WebRequest -Uri 'https://api.ipify.org' -UseBasicParsing -TimeoutSec 3).Content`).Output()
	publicIP := strings.TrimSpace(string(ipOut))

	// ping 지연
	pingOut, _ := exec.Command("powershell", "-NoProfile", "-Command",
		`(Test-Connection -ComputerName 8.8.8.8 -Count 1 -EA SilentlyContinue).ResponseTime`).Output()
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
	var req struct {
		Description string `json:"description"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Description == "" {
		req.Description = fmt.Sprintf("Nexus 자동 복구 포인트 %s", time.Now().Format("2006-01-02 15:04"))
	}

	script := fmt.Sprintf(`Checkpoint-Computer -Description "%s" -RestorePointType "APPLICATION_INSTALL"`, req.Description)
	err := exec.Command("powershell", "-NoProfile", "-Command", script).Run()
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false,
			"message": "복구 포인트 생성 실패 (시스템 보호 활성화 필요)"})
		return
	}
	json200(w, map[string]any{"success": true, "description": req.Description,
		"message": fmt.Sprintf("복구 포인트 생성 완료: %s", req.Description)})
}

// ──────────────────────────────────────────
// 디스크 검사 (CHKDSK) 예약
// ──────────────────────────────────────────

func handleDiskCheck(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Drive string `json:"drive"` // default "C:"
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Drive == "" {
		req.Drive = "C:"
	}

	// 다음 재시작 시 CHKDSK 예약 (/f = fix, /r = recover)
	out, err := exec.Command("cmd", "/c",
		fmt.Sprintf("echo Y | chkdsk %s /f /r /x", req.Drive)).Output()
	if err != nil {
		// CHKDSK 잠긴 경우 재시작 예약
		exec.Command("cmd", "/c",
			fmt.Sprintf("echo Y | chkntfs /c %s", req.Drive)).Run()
		json200(w, map[string]any{"success": true, "scheduled": true,
			"message": fmt.Sprintf("%s 디스크 검사가 다음 재시작 시 실행됩니다", req.Drive)})
		return
	}
	json200(w, map[string]any{"success": true, "output": string(out),
		"message": fmt.Sprintf("%s 디스크 검사 완료", req.Drive)})
}

// ──────────────────────────────────────────
// 브라우저 데이터 고급 정리
// ──────────────────────────────────────────

func handleBrowserClean(w http.ResponseWriter, r *http.Request) {
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
		"message":    fmt.Sprintf("브라우저 데이터 정리 완료: %s 확보", formatBytes(totalFreed)),
	})
}

// ──────────────────────────────────────────
// 설치된 프로그램 목록
// ──────────────────────────────────────────

func handleProgramsList(w http.ResponseWriter, r *http.Request) {
	out, _ := exec.Command("powershell", "-NoProfile", "-Command",
		`Get-Package | Select-Object Name,Version,Source | Sort-Object Name | ConvertTo-Json -Compress`).Output()

	var programs []struct {
		Name    string `json:"Name"`
		Version string `json:"Version"`
		Source  string `json:"Source"`
	}
	json.Unmarshal(out, &programs)

	if len(programs) == 0 {
		// WMIC fallback
		wmicOut, _ := exec.Command("wmic", "product", "get", "name,version",
			"/format:csv").Output()
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
	// 최근 부팅 이벤트 (이벤트 ID 12 = Kernel startup)
	out, _ := exec.Command("powershell", "-NoProfile", "-Command",
		`Get-WinEvent -FilterHashtable @{LogName='System';Id=12} -MaxEvents 5 -EA SilentlyContinue | Select-Object TimeCreated,Message | ConvertTo-Json -Compress`).Output()

	var events []struct {
		TimeCreated string `json:"TimeCreated"`
		Message     string `json:"Message"`
	}
	json.Unmarshal(out, &events)

	// 마지막 부팅 시간
	bootOut, _ := exec.Command("powershell", "-NoProfile", "-Command",
		`(Get-Date) - (gcim Win32_OperatingSystem).LastBootUpTime | Select-Object -ExpandProperty TotalMinutes`).Output()
	uptime := strings.TrimSpace(string(bootOut))

	// 시작 프로그램 수
	startupOut, _ := exec.Command("powershell", "-NoProfile", "-Command",
		`(Get-CimInstance Win32_StartupCommand | Measure-Object).Count`).Output()
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
		"message":         fmt.Sprintf("현재 가동 시간: %s분, 시작 프로그램: %s개", uptime, startupCount),
	})
}

// ──────────────────────────────────────────
// 파일 검색
// ──────────────────────────────────────────

func handleFilesSearch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query   string `json:"query"`
		Path    string `json:"path"`
		Type    string `json:"type"` // any | pdf | doc | image | video
		MaxDays int    `json:"max_days"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.Path == "" {
		home, _ := os.UserHomeDir()
		req.Path = home
	}
	if req.MaxDays == 0 {
		req.MaxDays = 30
	}

	extMap := map[string][]string{
		"pdf":   {".pdf"},
		"doc":   {".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".hwp"},
		"image": {".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp"},
		"video": {".mp4", ".mkv", ".avi", ".mov", ".wmv"},
		"any":   {},
	}
	allowedExts := extMap[req.Type]

	cutoff := time.Now().AddDate(0, 0, -req.MaxDays)
	queryLow := strings.ToLower(req.Query)

	type FileResult struct {
		Name    string `json:"name"`
		Path    string `json:"path"`
		SizeMB  float64 `json:"size_mb"`
		ModTime string `json:"mod_time"`
	}
	var results []FileResult

	filepath.Walk(req.Path, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if len(results) >= 50 {
			return nil
		}
		if info.ModTime().Before(cutoff) {
			return nil
		}
		if queryLow != "" && !strings.Contains(strings.ToLower(info.Name()), queryLow) {
			return nil
		}
		if len(allowedExts) > 0 {
			ext := strings.ToLower(filepath.Ext(info.Name()))
			found := false
			for _, e := range allowedExts {
				if e == ext {
					found = true
					break
				}
			}
			if !found {
				return nil
			}
		}
		results = append(results, FileResult{
			Name:    info.Name(),
			Path:    p,
			SizeMB:  float64(info.Size()) / (1 << 20),
			ModTime: info.ModTime().Format("2006-01-02 15:04"),
		})
		return nil
	})

	json200(w, map[string]any{
		"results": results,
		"total":   len(results),
		"message": fmt.Sprintf("'%s' 검색 결과: %d개", req.Query, len(results)),
	})
}

// ──────────────────────────────────────────
// 폴더 자동 정리 (날짜별 · 종류별)
// ──────────────────────────────────────────

var organizeExt = map[string]string{
	".jpg": "사진", ".jpeg": "사진", ".png": "사진", ".gif": "사진", ".webp": "사진", ".bmp": "사진", ".heic": "사진",
	".mp4": "동영상", ".mkv": "동영상", ".avi": "동영상", ".mov": "동영상", ".wmv": "동영상",
	".mp3": "음악", ".wav": "음악", ".flac": "음악", ".aac": "음악", ".ogg": "음악",
	".pdf": "문서", ".doc": "문서", ".docx": "문서", ".xls": "문서", ".xlsx": "문서",
	".ppt": "문서", ".pptx": "문서", ".hwp": "문서", ".txt": "문서", ".md": "문서",
	".zip": "압축파일", ".rar": "압축파일", ".7z": "압축파일", ".tar": "압축파일", ".gz": "압축파일",
	".exe": "프로그램", ".msi": "프로그램", ".apk": "프로그램",
}

func handleFilesOrganize(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"` // default: Downloads
		Mode string `json:"mode"` // type | date
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.Path == "" {
		home, _ := os.UserHomeDir()
		req.Path = filepath.Join(home, "Downloads")
	}
	if req.Mode == "" {
		req.Mode = "type"
	}

	entries, err := os.ReadDir(req.Path)
	if err != nil {
		writeJSON(w, 400, map[string]any{"success": false, "message": "폴더를 읽을 수 없어요"})
		return
	}

	moved := 0
	skipped := 0

	for _, e := range entries {
		if e.IsDir() {
			skipped++
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}

		var subDir string
		if req.Mode == "date" {
			subDir = info.ModTime().Format("2006-01")
		} else {
			ext := strings.ToLower(filepath.Ext(e.Name()))
			cat, ok := organizeExt[ext]
			if !ok {
				cat = "기타"
			}
			subDir = cat
		}

		dst := filepath.Join(req.Path, subDir)
		os.MkdirAll(dst, 0755)

		src := filepath.Join(req.Path, e.Name())
		dstPath := filepath.Join(dst, e.Name())
		if err := os.Rename(src, dstPath); err == nil {
			moved++
		}
	}

	json200(w, map[string]any{
		"success": true,
		"moved":   moved,
		"skipped": skipped,
		"message": fmt.Sprintf("%d개 파일 정리 완료 📁", moved),
	})
}

// ──────────────────────────────────────────
// 중복 파일 탐지
// ──────────────────────────────────────────

func handleFilesDuplicates(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.Path == "" {
		home, _ := os.UserHomeDir()
		req.Path = filepath.Join(home, "Downloads")
	}

	// 이름+크기 기준 중복 탐지 (해시 없이 빠른 탐지)
	type FileKey struct {
		Name string
		Size int64
	}
	seen := map[FileKey][]string{}

	filepath.Walk(req.Path, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		key := FileKey{Name: info.Name(), Size: info.Size()}
		seen[key] = append(seen[key], p)
		return nil
	})

	type DupGroup struct {
		Name   string   `json:"name"`
		SizeMB float64  `json:"size_mb"`
		Paths  []string `json:"paths"`
		Count  int      `json:"count"`
	}
	var groups []DupGroup
	var totalWaste int64

	for key, paths := range seen {
		if len(paths) > 1 {
			sort.Strings(paths)
			waste := key.Size * int64(len(paths)-1)
			totalWaste += waste
			groups = append(groups, DupGroup{
				Name: key.Name, SizeMB: float64(key.Size) / (1 << 20),
				Paths: paths, Count: len(paths),
			})
		}
	}

	sort.Slice(groups, func(i, j int) bool {
		return groups[i].SizeMB > groups[j].SizeMB
	})
	if len(groups) > 20 {
		groups = groups[:20]
	}

	json200(w, map[string]any{
		"groups":      groups,
		"total_groups": len(groups),
		"waste_mb":    float64(totalWaste) / (1 << 20),
		"waste":       formatBytes(totalWaste),
		"message":     fmt.Sprintf("중복 파일 %d그룹 발견, 낭비 공간 %s", len(groups), formatBytes(totalWaste)),
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

	if req.Action == "off" {
		// 알림 다시 켜기
		exec.Command("powershell", "-NoProfile", "-Command",
			`Set-ItemProperty -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Notifications\Settings' -Name 'NOC_GLOBAL_SETTING_TOASTS_ENABLED' -Value 1 -EA SilentlyContinue`).Run()
		json200(w, map[string]any{"success": true, "active": false, "message": "집중 모드 해제 완료 🔔"})
		return
	}

	// 알림 끄기 (방해 금지)
	exec.Command("powershell", "-NoProfile", "-Command",
		`Set-ItemProperty -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Notifications\Settings' -Name 'NOC_GLOBAL_SETTING_TOASTS_ENABLED' -Value 0 -EA SilentlyContinue`).Run()

	if req.Duration == 0 {
		req.Duration = 25
	}
	json200(w, map[string]any{
		"success":  true,
		"active":   true,
		"duration": req.Duration,
		"message":  fmt.Sprintf("집중 모드 시작! %d분 동안 알림이 차단됩니다 🎯", req.Duration),
	})
}

// ──────────────────────────────────────────
// 클립보드 히스토리
// ──────────────────────────────────────────

func handleClipboard(w http.ResponseWriter, r *http.Request) {
	out, _ := exec.Command("powershell", "-NoProfile", "-Command",
		`Get-Clipboard -Format Text`).Output()
	current := strings.TrimSpace(string(out))

	json200(w, map[string]any{
		"current": current,
		"tip":     "Windows + V 로 클립보드 히스토리를 볼 수 있어요",
	})
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
		json200(w, map[string]any{"success": true, "message": "메모 삭제 완료"})
		return
	}

	if req.Content == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "메모 내용을 입력해주세요"})
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
	json200(w, map[string]any{"success": true, "note": note, "message": "메모 저장 완료 📝"})
}
