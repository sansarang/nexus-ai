//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ══════════════════════════════════════════════════════════════════
//  Proactive Persistent Agent
//  - 매일 아침 자동 브리핑 (날씨 + 일정 + PC 상태)
//  - Anticipatory Actions: 미팅 전 자료 준비, 이상 감지 자동 대응
//  - 24/7 백그라운드 동작
// ══════════════════════════════════════════════════════════════════

type BriefingConfig struct {
	Enabled     bool   `json:"enabled"`
	Hour        int    `json:"hour"`        // 브리핑 시각 (기본 8시)
	WeatherCity string `json:"weather_city"` // 날씨 도시
	LastDate    string `json:"last_date"`   // 마지막 브리핑 날짜 (중복 방지)
}

var (
	briefingMu  sync.RWMutex
	briefingCfg BriefingConfig
)

func briefingConfigPath() string {
	appdata := os.Getenv("APPDATA")
	if appdata == "" {
		appdata = os.TempDir()
	}
	dir := filepath.Join(appdata, "Nexus")
	os.MkdirAll(dir, 0700)
	return filepath.Join(dir, "briefing_config.json")
}

func loadBriefingConfig() {
	data, err := os.ReadFile(briefingConfigPath())
	if err != nil {
		briefingCfg = BriefingConfig{Enabled: true, Hour: 8, WeatherCity: "Seoul"}
		return
	}
	json.Unmarshal(data, &briefingCfg)
	if briefingCfg.Hour == 0 {
		briefingCfg.Hour = 8
	}
	if briefingCfg.WeatherCity == "" {
		briefingCfg.WeatherCity = "Seoul"
	}
}

func saveBriefingConfig() {
	briefingMu.RLock()
	data, _ := json.MarshalIndent(briefingCfg, "", "  ")
	briefingMu.RUnlock()
	os.WriteFile(briefingConfigPath(), data, 0600)
}

// ── 브리핑 생성 ─────────────────────────────────────────────────

func generateMorningBriefing() string {
	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()

	var parts []string

	// 1. 날씨
	briefingMu.RLock()
	city := briefingCfg.WeatherCity
	briefingMu.RUnlock()

	weather := fetchWeatherText(city, gKey)
	parts = append(parts, "🌤️ 날씨: "+weather)

	// 2. 오늘 일정
	calEvents := getCalendarEventsToday()
	if len(calEvents) == 0 {
		parts = append(parts, "📅 오늘 일정: 예약된 일정이 없습니다.")
	} else {
		eventLines := "📅 오늘 일정:\n"
		for _, e := range calEvents {
			eventLines += fmt.Sprintf("  • %s — %s\n", e.Time, e.Title)
		}
		parts = append(parts, strings.TrimRight(eventLines, "\n"))
	}

	// 3. PC 상태
	mem := getMemoryUsage()
	_, total := getDiskSpace()
	free, _ := getDiskSpace()
	diskPct := 0.0
	if total > 0 {
		diskPct = float64(free) / float64(total) * 100
	}
	pcStatus := fmt.Sprintf("💻 PC 상태: 메모리 %d%% 사용 중, 디스크 여유 %.0f%%", mem, diskPct)
	if mem > 80 {
		pcStatus += " ⚠️ 메모리 주의"
	}
	if diskPct < 15 {
		pcStatus += " ⚠️ 디스크 부족"
	}
	parts = append(parts, pcStatus)

	// 4. Groq로 자연스러운 브리핑 문장 생성
	if gKey != "" {
		rawData := strings.Join(parts, "\n")
		sysMsg := `You are Nexus, a Korean AI personal secretary. Generate a warm, natural morning briefing in Korean based on the data provided.
Keep it concise (3-5 sentences). Be friendly and helpful. No markdown.`
		userMsg := fmt.Sprintf("오늘 날짜: %s\n\n데이터:\n%s\n\n위 정보를 바탕으로 자연스러운 아침 브리핑을 작성해주세요.",
			time.Now().Format("2006년 1월 2일 (Monday)"), rawData)

		briefing, _, err := callGroq(gKey, groqChatModel, []groqMsg{
			{Role: "system", Content: sysMsg},
			{Role: "user", Content: userMsg},
		}, 300, false)
		if err == nil && briefing != "" {
			return briefing
		}
	}

	// Groq 실패 시 데이터 직접 반환
	return "좋은 아침입니다! 오늘 하루를 시작하겠습니다.\n" + strings.Join(parts, "\n")
}

// ── 일정 기반 사전 준비 (Anticipatory Action) ──────────────────

type CalEvent struct {
	Time  string
	Title string
	Raw   string
}

func getCalendarEventsToday() []CalEvent {
	// Google Calendar API 또는 Windows Calendar 연동
	// 현재는 handlers_calendar.go의 데이터 재활용
	script := `
$today = Get-Date -Format "yyyy-MM-dd"
$outlook = $null
try {
    $outlook = New-Object -ComObject Outlook.Application
    $ns = $outlook.GetNamespace("MAPI")
    $cal = $ns.GetDefaultFolder(9)
    $items = $cal.Items
    $items.IncludeRecurrences = $true
    $items.Sort("[Start]")
    $filter = "[Start] >= '" + $today + " 00:00' AND [Start] <= '" + $today + " 23:59'"
    $filtered = $items.Restrict($filter)
    foreach ($item in $filtered) {
        $time = $item.Start.ToString("HH:mm")
        $subject = $item.Subject
        Write-Output "${time}|${subject}"
    }
} catch {
    Write-Output "NO_OUTLOOK"
}
`
	out, err := execPowerShell(script)
	if err != nil || strings.Contains(out, "NO_OUTLOOK") {
		return nil
	}

	var events []CalEvent
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		if len(parts) == 2 {
			events = append(events, CalEvent{Time: parts[0], Title: parts[1]})
		}
	}
	return events
}

func execPowerShell(script string) (string, error) {
	out, err := exec.Command("powershell", "-NoProfile", "-Command", script).Output()
	return strings.TrimSpace(string(out)), err
}

// anticipatoryCheck: 미팅 1시간 전 감지 → 자동 준비
func anticipatoryCheck() {
	events := getCalendarEventsToday()
	now := time.Now()

	for _, e := range events {
		// 시간 파싱
		t, err := time.Parse("15:04", e.Time)
		if err != nil {
			continue
		}
		// 오늘 날짜 붙이기
		eventTime := time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, now.Location())
		diff := eventTime.Sub(now)

		// 1시간 전 ± 2분
		if diff >= 58*time.Minute && diff <= 62*time.Minute {
			publishAlert(Alert{
				ID:      fmt.Sprintf("meeting_prep_%s", e.Time),
				Level:   "info",
				Title:   fmt.Sprintf("📅 %s — %s 시작 1시간 전입니다", e.Title, e.Time),
				Message: fmt.Sprintf("'%s' 미팅까지 1시간 남았습니다. 관련 자료를 준비해드릴까요?", e.Title),
				Action:  fmt.Sprintf("meeting_prep:%s", e.Title),
			})
		}

		// 10분 전 알림
		if diff >= 8*time.Minute && diff <= 12*time.Minute {
			publishAlert(Alert{
				ID:      fmt.Sprintf("meeting_soon_%s", e.Time),
				Level:   "warn",
				Title:   fmt.Sprintf("⏰ %s 10분 후 시작!", e.Title),
				Message: fmt.Sprintf("'%s' 미팅이 10분 후 시작됩니다.", e.Title),
			})
		}
	}
}

// ── 브리핑 스케줄러 (24/7 백그라운드) ────────────────────────────

func startBriefingScheduler() {
	loadBriefingConfig()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		today := now.Format("2006-01-02")

		briefingMu.RLock()
		enabled := briefingCfg.Enabled
		hour := briefingCfg.Hour
		lastDate := briefingCfg.LastDate
		briefingMu.RUnlock()

		// 아침 브리핑: 설정 시각, 오늘 아직 안 보냈을 때
		if enabled && now.Hour() == hour && now.Minute() < 5 && lastDate != today {
			briefing := generateMorningBriefing()
			publishAlert(Alert{
				ID:      fmt.Sprintf("briefing_%s", today),
				Level:   "info",
				Title:   fmt.Sprintf("☀️ %s 아침 브리핑", now.Format("1월 2일")),
				Message: briefing,
			})

			briefingMu.Lock()
			briefingCfg.LastDate = today
			briefingMu.Unlock()
			saveBriefingConfig()
		}

		// 미팅 사전 감지 (매분 확인)
		anticipatoryCheck()
	}
}

// ── HTTP 핸들러 ─────────────────────────────────────────────────

// POST /api/briefing/now — 지금 즉시 브리핑 실행
func handleBriefingNow(w http.ResponseWriter, r *http.Request) {
	task := globalTaskQueue.Enqueue("아침 브리핑 생성", PriorityNormal, nil, func(t *AgentTask) {
		t.UpdateProgress(30, "날씨 정보 수집 중...")
		briefing := generateMorningBriefing()
		t.UpdateProgress(80, "브리핑 전송 중...")
		publishAlert(Alert{
			ID:      fmt.Sprintf("briefing_now_%d", time.Now().Unix()),
			Level:   "info",
			Title:   "📋 Nexus 브리핑",
			Message: briefing,
		})
		t.Result = map[string]any{"briefing": briefing}
	})

	json200(w, map[string]any{
		"success": true,
		"task_id": task.ID,
		"message": "브리핑 생성 중...",
	})
}

// GET/POST /api/briefing/config
func handleBriefingConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		briefingMu.RLock()
		cfg := briefingCfg
		briefingMu.RUnlock()
		json200(w, cfg)
		return
	}

	var req BriefingConfig
	json.NewDecoder(r.Body).Decode(&req)
	briefingMu.Lock()
	if req.Hour > 0 {
		briefingCfg.Hour = req.Hour
	}
	if req.WeatherCity != "" {
		briefingCfg.WeatherCity = req.WeatherCity
	}
	briefingCfg.Enabled = req.Enabled
	briefingMu.Unlock()
	saveBriefingConfig()

	json200(w, map[string]any{"success": true, "message": "브리핑 설정 저장 완료"})
}
