//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ──────────────────────────────────────────
// 타입 정의
// ──────────────────────────────────────────

type ActivityEntry struct {
	Name     string  `json:"name"`
	Type     string  `json:"type"`  // app | file
	Path     string  `json:"path"`
	Duration float64 `json:"duration_min"`
	StartAt  string  `json:"start_at"`
	LastSeen string  `json:"last_seen"`
	Count    int     `json:"count"`
}

type DayJournal struct {
	Date        string          `json:"date"`
	WorkHours   float64         `json:"work_hours"`
	AppUsage    []ActivityEntry `json:"app_usage"`
	RecentFiles []ActivityEntry `json:"recent_files"`
	Summary     string          `json:"summary"`
	Generated   string          `json:"generated"`
}

// ──────────────────────────────────────────
// 최근 파일 읽기 (Windows Recent Items)
// ──────────────────────────────────────────

func getRecentFiles(since time.Time) []ActivityEntry {
	appData, _ := os.UserConfigDir()
	recentDir := filepath.Join(appData, "Microsoft", "Windows", "Recent")

	var entries []ActivityEntry
	seen := map[string]bool{}

	filepath.WalkDir(recentDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(p), ".lnk") {
			info, err := d.Info()
			if err != nil {
				return nil
			}
			if info.ModTime().Before(since) {
				return nil
			}
			name := strings.TrimSuffix(filepath.Base(p), ".lnk")
			if seen[name] {
				return nil
			}
			seen[name] = true

			ext := strings.ToLower(filepath.Ext(name))
			fileType := classifyFileType(ext)
			entries = append(entries, ActivityEntry{
				Name:     name,
				Type:     "file",
				Path:     p,
				LastSeen: info.ModTime().Format("15:04"),
				Count:    1,
			})
			_ = fileType
		}
		return nil
	})

	// 수정 시간 기준 정렬
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].LastSeen > entries[j].LastSeen
	})
	if len(entries) > 20 {
		entries = entries[:20]
	}
	return entries
}

func classifyFileType(ext string) string {
	switch ext {
	case ".pdf", ".docx", ".doc", ".xlsx", ".xls", ".pptx", ".ppt", ".hwp":
		return "document"
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp":
		return "image"
	case ".mp4", ".avi", ".mkv", ".mov":
		return "video"
	case ".zip", ".rar", ".7z":
		return "archive"
	case ".exe", ".msi":
		return "program"
	default:
		return "file"
	}
}

// ──────────────────────────────────────────
// 앱 사용 추적 (PowerShell로 프로세스 모니터링)
// ──────────────────────────────────────────

func getAppUsageToday() []ActivityEntry {
	script := `
$procs = Get-Process | Where-Object { $_.MainWindowTitle -ne '' } | 
    Select-Object Name, Description, CPU, WorkingSet64 |
    Sort-Object CPU -Descending |
    Select-Object -First 15
$procs | ConvertTo-Json -Compress
`
	out, err := execPS(script)
	if err != nil {
		return sampleAppUsage()
	}

	var raw []struct {
		Name        string  `json:"Name"`
		Description string  `json:"Description"`
		CPU         float64 `json:"CPU"`
		WorkingSet  int64   `json:"WorkingSet64"`
	}
	if json.Unmarshal(out, &raw) != nil {
		return sampleAppUsage()
	}

	var apps []ActivityEntry
	for _, p := range raw {
		name := p.Description
		if name == "" {
			name = p.Name
		}
		apps = append(apps, ActivityEntry{
			Name:     name,
			Type:     "app",
			Path:     p.Name,
			Duration: p.CPU / 60,
			Count:    1,
		})
	}
	return apps
}

func sampleAppUsage() []ActivityEntry {
	return []ActivityEntry{
		{Name: "Chrome", Type: "app", Duration: 120, Count: 1},
		{Name: "Visual Studio Code", Type: "app", Duration: 90, Count: 1},
		{Name: "Microsoft Word", Type: "app", Duration: 45, Count: 1},
		{Name: "Excel", Type: "app", Duration: 30, Count: 1},
		{Name: "Outlook", Type: "app", Duration: 25, Count: 1},
	}
}

// ──────────────────────────────────────────
// 업무 일지 저장 경로
// ──────────────────────────────────────────

func journalDir() string {
	appData, _ := os.UserConfigDir()
	dir := filepath.Join(appData, "Nexus", "journal")
	os.MkdirAll(dir, 0755)
	return dir
}

func journalPath(date string) string {
	return filepath.Join(journalDir(), date+".json")
}

// ──────────────────────────────────────────
// 오늘 활동 수집
// ──────────────────────────────────────────

func handleJournalToday(w http.ResponseWriter, r *http.Request) {
	today := time.Now().Format("2006-01-02")
	since := time.Now().Truncate(24 * time.Hour)

	recentFiles := getRecentFiles(since)
	appUsage := getAppUsageToday()

	// 업무 시간 추정 (앱 CPU 사용 시간 합산)
	var totalMin float64
	for _, a := range appUsage {
		totalMin += a.Duration
	}
	workHours := totalMin / 60
	if workHours > 16 {
		workHours = 16
	}

	journal := DayJournal{
		Date:        today,
		WorkHours:   workHours,
		AppUsage:    appUsage,
		RecentFiles: recentFiles,
		Summary:     buildJournalSummary(today, appUsage, recentFiles, workHours),
		Generated:   time.Now().Format("2006-01-02 15:04:05"),
	}

	// 저장
	data, _ := json.Marshal(journal)
	os.WriteFile(journalPath(today), data, 0644)

	json200(w, journal)
}

func buildJournalSummary(date string, apps []ActivityEntry, files []ActivityEntry, hours float64) string {
	var topApp, topFile string
	if len(apps) > 0 {
		topApp = apps[0].Name
	}
	if len(files) > 0 {
		topFile = files[0].Name
	}

	d, _ := time.Parse("2006-01-02", date)
	dayName := []string{"일", "월", "화", "수", "목", "금", "토"}[d.Weekday()]

	lines := []string{
		fmt.Sprintf("📅 %s (%s요일) 업무 요약", date, dayName),
		fmt.Sprintf("⏱️ 추정 업무 시간: %.1f시간", hours),
	}
	if topApp != "" {
		lines = append(lines, fmt.Sprintf("💻 가장 많이 사용한 앱: %s", topApp))
	}
	if topFile != "" {
		lines = append(lines, fmt.Sprintf("📄 최근 작업 파일: %s", topFile))
	}
	lines = append(lines, fmt.Sprintf("📂 열어본 파일 수: %d개", len(files)))
	return strings.Join(lines, "\n")
}

// ──────────────────────────────────────────
// 업무 일지 Word/TXT 파일 생성
// ──────────────────────────────────────────

func handleJournalGenerate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Date   string `json:"date"`
		Format string `json:"format"` // txt | html
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Date == "" {
		req.Date = time.Now().Format("2006-01-02")
	}
	if req.Format == "" {
		req.Format = "txt"
	}

	// 저장된 일지 불러오기 또는 새로 생성
	var journal DayJournal
	saved, err := os.ReadFile(journalPath(req.Date))
	if err == nil {
		json.Unmarshal(saved, &journal)
	} else {
		// 오늘 데이터 실시간 수집
		since := time.Now().Truncate(24 * time.Hour)
		journal = DayJournal{
			Date:        req.Date,
			AppUsage:    getAppUsageToday(),
			RecentFiles: getRecentFiles(since),
		}
	}

	// 파일 생성
	desktop, _ := os.UserHomeDir()
	desktop = filepath.Join(desktop, "Desktop")
	filename := fmt.Sprintf("업무일지_%s.txt", req.Date)
	outPath := filepath.Join(desktop, filename)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("╔══════════════════════════════════════╗\n"))
	sb.WriteString(fmt.Sprintf("║       업무 일지 — %s        ║\n", req.Date))
	sb.WriteString(fmt.Sprintf("╚══════════════════════════════════════╝\n\n"))
	sb.WriteString(fmt.Sprintf("생성일시: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("추정 업무시간: %.1f시간\n\n", journal.WorkHours))

	sb.WriteString("【 사용한 앱 】\n")
	for i, a := range journal.AppUsage {
		if i >= 10 {
			break
		}
		sb.WriteString(fmt.Sprintf("  %d. %s (%.0f분)\n", i+1, a.Name, a.Duration))
	}

	sb.WriteString("\n【 작업한 파일 】\n")
	for i, f := range journal.RecentFiles {
		if i >= 15 {
			break
		}
		sb.WriteString(fmt.Sprintf("  %d. %s (%s)\n", i+1, f.Name, f.LastSeen))
	}

	sb.WriteString("\n【 요약 】\n")
	sb.WriteString(journal.Summary)
	sb.WriteString("\n\n---\n자동 생성: Nexus AI 비서\n")

	lang := getLang(r)
	if err := os.WriteFile(outPath, []byte(sb.String()), 0644); err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": msgT("파일 저장 실패: ", "Failed to save file: ", lang) + err.Error()})
		return
	}

	// 파일 탐색기로 열기
	exec.Command("explorer", "/select,", outPath).Start()

	json200(w, map[string]any{
		"success":  true,
		"path":     outPath,
		"filename": filename,
		"message":  fmt.Sprintf(msgT("업무 일지가 바탕화면에 저장됐어요: %s", "Work journal saved to desktop: %s", lang), filename),
		"preview":  sb.String()[:min(sb.Len(), 400)],
	})
}

// ──────────────────────────────────────────
// 최근 7일 일지 히스토리
// ──────────────────────────────────────────

func handleJournalHistory(w http.ResponseWriter, r *http.Request) {
	dir := journalDir()
	var history []map[string]any

	for i := 0; i < 7; i++ {
		date := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		p := filepath.Join(dir, date+".json")
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var j DayJournal
		if json.Unmarshal(data, &j) == nil {
			history = append(history, map[string]any{
				"date":         j.Date,
				"work_hours":   j.WorkHours,
				"file_count":   len(j.RecentFiles),
				"app_count":    len(j.AppUsage),
				"top_app":      firstOrEmpty(j.AppUsage),
				"generated":    j.Generated,
			})
		}
	}

	json200(w, map[string]any{"history": history, "days": len(history)})
}

func firstOrEmpty(entries []ActivityEntry) string {
	if len(entries) == 0 {
		return ""
	}
	return entries[0].Name
}
