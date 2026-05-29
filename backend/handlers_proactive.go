//go:build windows

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// ══════════════════════════════════════════════════════════════════
//  Proactive AI — 24/7 Background Intelligence
//  - 매 30초 PC 상태 감시
//  - CPU > 85% 3회 연속 → 자동 정리 + 알림
//  - 디스크 > 90% → 임시파일 자동 정리 + 알림
//  - 의심 프로세스 → 알림
//  - 캘린더 미팅 30분 전 → 알림
//  - 업무 시간 2시간 무활동 → 집중 모드 제안
//  - 오전 08:00 → 모닝 브리핑 (날씨+뉴스+캘린더+PC)
// ══════════════════════════════════════════════════════════════════

type Alert struct {
	ID        string    `json:"id"`
	Level     string    `json:"level"`            // info | warn | critical
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Action    string    `json:"action,omitempty"` // 권장 액션
	Timestamp time.Time `json:"timestamp"`
	Dismissed bool      `json:"dismissed"`
}

var (
	alertMu      sync.RWMutex
	latestAlerts []Alert
	alertSubs    = make(map[string]chan Alert) // SSE 구독자
	subMu        sync.Mutex
)

// publishAlert: 모든 SSE 구독자에게 알림 push + 최근 목록 유지
func publishAlert(a Alert) {
	a.Timestamp = time.Now()

	alertMu.Lock()
	latestAlerts = append([]Alert{a}, latestAlerts...)
	if len(latestAlerts) > 50 {
		latestAlerts = latestAlerts[:50]
	}
	alertMu.Unlock()

	subMu.Lock()
	for _, ch := range alertSubs {
		select {
		case ch <- a:
		default:
		}
	}
	subMu.Unlock()
}

// ── 의심 프로세스 감지 ────────────────────────────────────────

var suspiciousProcessKeywords = []string{
	"keylog", "miner", "cryptominer", "xmrig", "coinhive",
	"njrat", "darkcomet", "nanocore", "remcos", "asyncrat",
	"mimikatz", "pwdump", "lazagne", "wce.exe",
}

func detectSuspiciousProcesses() []string {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	out, err := newHiddenCmdCtx(ctx, "powershell", "-NoProfile", "-Command",
		"Get-Process | Select-Object -ExpandProperty Name").Output()
	if err != nil {
		return nil
	}
	procs := strings.Split(strings.ToLower(string(out)), "\n")
	var found []string
	for _, p := range procs {
		p = strings.TrimSpace(p)
		for _, kw := range suspiciousProcessKeywords {
			if strings.Contains(p, kw) {
				found = append(found, p)
				break
			}
		}
	}
	return found
}

// ── 임시 파일 자동 정리 ──────────────────────────────────────

func autoCleanTempFiles() int64 {
	var freed int64
	tempDirs := []string{
		os.Getenv("TEMP"),
		os.Getenv("TMP"),
		os.ExpandEnv(`%SystemRoot%\Temp`),
	}
	for _, dir := range tempDirs {
		if dir == "" {
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			path := dir + string(os.PathSeparator) + e.Name()
			fi, err := os.Stat(path)
			if err != nil {
				continue
			}
			// 1일 이상 지난 파일만 삭제
			if time.Since(fi.ModTime()) < 24*time.Hour {
				continue
			}
			if fi.IsDir() {
				continue
			}
			freed += fi.Size()
			os.Remove(path)
		}
	}
	return freed
}

// ── 모닝 브리핑 생성 ─────────────────────────────────────────

func generateProactiveMorningBriefing() string {
	llmMu.RLock()
	pKey := llmPerplexityKey
	llmMu.RUnlock()

	// PC 건강 요약
	mem := getMemoryUsage()
	free, total := getDiskSpace()
	diskUsed := 0.0
	if total > 0 {
		diskUsed = (1 - float64(free)/float64(total)) * 100
	}
	eng := IsUserEng()

	var pcHealth string
	if eng {
		pcHealth = fmt.Sprintf("Memory: %d%% used, Disk: %.0f%% used", mem, diskUsed)
	} else {
		pcHealth = fmt.Sprintf("메모리: %d%% 사용, 디스크: %.0f%% 사용", mem, diskUsed)
	}

	if pKey == "" {
		if eng {
			return fmt.Sprintf("🌅 Good morning! Have a great day.\n\n💻 PC Status: %s", pcHealth)
		}
		return fmt.Sprintf("🌅 좋은 아침이에요! 오늘도 좋은 하루 되세요.\n\n💻 PC 상태: %s", pcHealth)
	}

	var prompt string
	if eng {
		prompt = fmt.Sprintf(`Write a morning briefing in English. Include:
1. Today's weather (major cities)
2. Top 3 news highlights (brief)
3. PC health summary: %s
4. An uplifting message for the day

Keep it short and friendly, include emojis, under 300 words.`, pcHealth)
	} else {
		prompt = fmt.Sprintf(`오늘 아침 브리핑을 한국어로 작성해주세요. 포함 내용:
1. 오늘 날씨 (한국 서울 기준)
2. 오늘의 주요 뉴스 3가지 (간략하게)
3. PC 건강 요약: %s
4. 오늘의 한마디 응원 메시지

짧고 친근하게, 이모지 포함, 500자 이내로 작성하세요.`, pcHealth)
	}

	briefing, _, err := callGroqWithFallback([]groqMsg{
		{Role: "user", Content: prompt},
	}, 400, false)
	if err != nil {
		if eng {
			return fmt.Sprintf("🌅 Good morning!\n\n💻 PC Status: %s", pcHealth)
		}
		return fmt.Sprintf("🌅 좋은 아침이에요!\n\n💻 PC 상태: %s", pcHealth)
	}
	return briefing
}

// ── 캘린더 미팅 사전 알림 ────────────────────────────────────

func checkUpcomingMeetings(eng bool) {
	// 캘린더 이벤트 확인 (PowerShell Outlook COM)
	script := outlookProfileCheckPS + `
try {
  $outlook = New-Object -ComObject Outlook.Application
  $cal = $outlook.GetNamespace("MAPI").GetDefaultFolder(9)
  $now = Get-Date
  $soon = $now.AddMinutes(35)
  $items = $cal.Items
  $items.Sort("[Start]")
  $items.IncludeRecurrences = $true
  $restrict = $items.Restrict("[Start] >= '" + $now.ToString("g") + "' AND [Start] <= '" + $soon.ToString("g") + "'")
  foreach ($item in $restrict) {
    Write-Output ($item.Subject + "|" + $item.Start.ToString("HH:mm"))
  }
} catch { }
`
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	out, err := newHiddenCmdCtx(ctx, "powershell", "-NoProfile", "-Command", script).Output()
	if err != nil || len(out) == 0 {
		return
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "ERROR:") {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		subject := line
		startTime := ""
		if len(parts) == 2 {
			subject = parts[0]
			startTime = parts[1]
		}
		var meetTitle, meetMsg string
		if eng {
			meetTitle = "📅 Meeting in 30 minutes!"
			meetMsg = fmt.Sprintf("'%s' is scheduled at %s. Time to prepare!", subject, startTime)
		} else {
			meetTitle = "📅 미팅 30분 전!"
			meetMsg = fmt.Sprintf("'%s' 미팅이 %s에 예정되어 있어요. 준비하세요!", subject, startTime)
		}
		publishAlert(Alert{
			ID:      fmt.Sprintf("meeting_%d", time.Now().UnixNano()),
			Level:   "warn",
			Title:   meetTitle,
			Message: meetMsg,
			Action:  "calendar",
		})
	}
}

// ── 무활동 감지 ──────────────────────────────────────────────

var (
	lastActivityMu sync.Mutex
	lastActivity   = time.Now()
)

// UpdateActivity: 사용자 활동 발생 시 호출
func updateLastActivity() {
	lastActivityMu.Lock()
	lastActivity = time.Now()
	lastActivityMu.Unlock()
}

// ── 메인 모니터링 루프 ───────────────────────────────────────

func startProactiveMonitor() {
	// 5분 간격으로 변경 — 30초마다 PowerShell 실행 시 VM 메모리 폭발 방지
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	var (
		lastCPUAlert       time.Time
		lastDiskAlert      time.Time
		lastMemAlert       time.Time
		lastSuspAlert      time.Time
		lastBriefingDate   string
		lastFocusSuggested time.Time
		cpuHighCount       int
		lastMeetingCheck   time.Time
	)

	for range ticker.C {
		now := time.Now()

		// ── 모닝 브리핑 (08:00 매일) ────────────────────────────
		today := now.Format("2006-01-02")
		if now.Hour() == 8 && now.Minute() < 6 && lastBriefingDate != today {
			lastBriefingDate = today
			go func(t time.Time, d string) {
				bEng := IsUserEng()
				briefing := generateProactiveMorningBriefing()
				var briefTitle string
				if bEng {
					briefTitle = fmt.Sprintf("☀️ Morning Briefing — %s", t.Format("January 2"))
				} else {
					briefTitle = fmt.Sprintf("☀️ %s 아침 브리핑", t.Format("1월 2일"))
				}
				publishAlert(Alert{
					ID:      "morning_briefing_" + d,
					Level:   "info",
					Title:   briefTitle,
					Message: briefing,
					Action:  "briefing",
				})
			}(now, today)
		}

		// 저장된 사용자 언어 설정 사용 (입력 없는 자동 알림도 정확하게 처리)
		proEng := IsUserEng()

		// ── CPU/메모리 고부하 감지 (3회 연속 > 85%) ─────────────
		mem := getMemoryUsage()
		cpuEst := mem
		if cpuEst > 85 {
			cpuHighCount++
			if cpuHighCount >= 3 && now.Sub(lastCPUAlert) > 30*time.Minute {
				lastCPUAlert = now
				cpuHighCount = 0
				var cpuTitle, cpuMsg string
				if proEng {
					cpuTitle = "🌡️ High CPU/Memory Usage"
					cpuMsg = fmt.Sprintf("Memory usage at %d%%. Recommend cleaning temp files. Clean now?", mem)
				} else {
					cpuTitle = "🌡️ CPU/메모리 과부하 감지"
					cpuMsg = fmt.Sprintf("메모리 %d%% 사용 중입니다. 임시 파일 정리를 권장해요. 정리해드릴까요?", mem)
				}
				publishAlert(Alert{
					ID: fmt.Sprintf("cpu_high_%d", now.Unix()), Level: "warn",
					Title: cpuTitle, Message: cpuMsg, Action: "clean",
				})
			}
		} else {
			cpuHighCount = 0
		}

		// ── 메모리 경고 ──────────────────────────────────────────
		if mem > 90 && now.Sub(lastMemAlert) > 30*time.Minute {
			lastMemAlert = now
			var memTitle, memMsg string
			if proEng {
				memTitle = fmt.Sprintf("💾 Memory %d%% Used", mem)
				memMsg = "Close unused apps to speed up your PC. Clean now?"
			} else {
				memTitle = fmt.Sprintf("💾 메모리 %d%% 사용 중", mem)
				memMsg = "불필요한 프로그램을 종료하면 PC가 빨라질 거예요. 지금 정리해드릴까요?"
			}
			publishAlert(Alert{
				ID: fmt.Sprintf("mem_%d", now.Unix()), Level: "warn",
				Title: memTitle, Message: memMsg, Action: "clean",
			})
		}

		// ── 디스크 > 90% → 알림만 (자동 삭제 제거) ──────────────
		free, total := getDiskSpace()
		if total > 0 {
			usedPct := (1 - float64(free)/float64(total)) * 100
			if usedPct > 90 && now.Sub(lastDiskAlert) > 60*time.Minute {
				lastDiskAlert = now
				var diskTitle, diskMsg string
				if proEng {
					diskTitle = fmt.Sprintf("💿 Disk %.0f%% Full!", usedPct)
					diskMsg = "Disk space is running low. Clean temp files now?"
				} else {
					diskTitle = fmt.Sprintf("💿 디스크 %.0f%% 사용!", usedPct)
					diskMsg = "디스크 공간이 부족합니다. 임시 파일 정리를 실행할까요?"
				}
				publishAlert(Alert{
					ID: fmt.Sprintf("disk_full_%d", now.Unix()), Level: "critical",
					Title: diskTitle, Message: diskMsg, Action: "clean",
				})
			}
		}

		// ── 의심 프로세스 감지 (10분 간격) ──────────────────────
		if now.Sub(lastSuspAlert) > 10*time.Minute {
			lastSuspAlert = now
			go func(eng bool) {
				suspicious := detectSuspiciousProcesses()
				if len(suspicious) > 0 {
					var suspTitle, suspMsg string
					if eng {
						suspTitle = "🚨 Suspicious Process Detected!"
						suspMsg = fmt.Sprintf("Suspicious processes running: %s", strings.Join(suspicious, ", "))
					} else {
						suspTitle = "🚨 의심 프로세스 감지!"
						suspMsg = fmt.Sprintf("의심스러운 프로세스가 실행 중입니다: %s", strings.Join(suspicious, ", "))
					}
					publishAlert(Alert{
						ID: fmt.Sprintf("suspicious_%d", now.Unix()), Level: "critical",
						Title: suspTitle, Message: suspMsg, Action: "security",
					})
				}
			}(proEng)
		}

		// ── 미팅 30분 전 알림 (10분마다 확인) ──────────────────
		if now.Sub(lastMeetingCheck) > 10*time.Minute {
			lastMeetingCheck = now
			go checkUpcomingMeetings(proEng)
		}

		// ── 업무 시간 2시간 무활동 → 집중 모드 제안 ────────────
		isWorkHour := now.Hour() >= 9 && now.Hour() < 18
		lastActivityMu.Lock()
		inactiveDur := now.Sub(lastActivity)
		lastActivityMu.Unlock()

		if isWorkHour && inactiveDur > 2*time.Hour && now.Sub(lastFocusSuggested) > 2*time.Hour {
			lastFocusSuggested = now
			var focusTitle, focusMsg string
			if proEng {
				focusTitle = "🎯 Focus Mode Suggested"
				focusMsg = fmt.Sprintf("No activity for ~%d min. Enable Focus Mode?", int(inactiveDur.Minutes()))
			} else {
				focusTitle = "🎯 집중 모드 제안"
				focusMsg = fmt.Sprintf("약 %d분간 활동이 없었어요. 집중 모드를 활성화할까요?", int(inactiveDur.Minutes()))
			}
			publishAlert(Alert{
				ID: fmt.Sprintf("focus_%d", now.Unix()), Level: "info",
				Title: focusTitle, Message: focusMsg, Action: "focus",
			})
		}

		// ── 대용량 임시 파일 알림 ────────────────────────────────
		tempSize := getTempSize()
		if tempSize > 2<<30 { // 2GB
			var tempTitle, tempMsg string
			if proEng {
				tempTitle = fmt.Sprintf("🗑️ %s of Temp Files", formatBytes(tempSize))
				tempMsg = "Large temp files accumulated. Clean now to free disk space."
			} else {
				tempTitle = fmt.Sprintf("🗑️ 임시 파일 %s 누적", formatBytes(tempSize))
				tempMsg = "임시 파일이 많이 쌓였어요. 지금 정리하면 디스크를 확보할 수 있어요."
			}
			publishAlert(Alert{
				ID: fmt.Sprintf("temp_%d", now.Unix()), Level: "info",
				Title: tempTitle, Message: tempMsg, Action: "clean",
			})
		}
	}
}

// ── SSE 핸들러 ──────────────────────────────────────────────────

// GET /api/alerts/stream — SSE 실시간 알림 스트림
func handleAlertStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := make(chan Alert, 10)
	id := fmt.Sprintf("sub_%d", time.Now().UnixNano())

	subMu.Lock()
	alertSubs[id] = ch
	subMu.Unlock()

	defer func() {
		subMu.Lock()
		delete(alertSubs, id)
		subMu.Unlock()
		close(ch)
	}()

	flusher, ok := w.(http.Flusher)
	if !ok {
		return
	}

	fmt.Fprintf(w, "data: {\"type\":\"connected\"}\n\n")
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case alert, ok := <-ch:
			if !ok {
				return
			}
			data, _ := json.Marshal(alert)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-time.After(25 * time.Second):
			fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}

// GET /api/alerts/latest — 최근 알림 목록
func handleAlertLatest(w http.ResponseWriter, r *http.Request) {
	alertMu.RLock()
	alerts := make([]Alert, len(latestAlerts))
	copy(alerts, latestAlerts)
	alertMu.RUnlock()

	json200(w, map[string]any{
		"alerts": alerts,
		"count":  len(alerts),
	})
}
