// phase_b_proactive.go — Phase B4: 능동성 (cross-platform)
// 사장님 요구: "백그라운드 PC 모니터링 → 임계치 도달 → 자동 알림"
//
// 이벤트 종류:
//   - 디스크 90% 이상 → cleanup 권장
//   - CPU 100% 5분 연속 → 이상 프로세스 의심
//   - 메모리 누수 의심 → 재시작 권장
//   - 매주 동일 시간 명령 반복 → 자동화 제안
//
// 발견된 이벤트는 ProactiveAlerts 큐에 쌓이고, 프론트가 폴링하거나 SSE/WebSocket 으로 받음

package main

import (
	"sync"
	"time"
)

// ProactiveAlert — 능동 알림 단위
type ProactiveAlert struct {
	ID        string         `json:"id"`
	Severity  string         `json:"severity"` // info / warning / critical
	Title     string         `json:"title"`
	Message   string         `json:"message"`
	Action    string         `json:"action"`     // 추천 액션 (예: "clean", "scan")
	Params    map[string]any `json:"params"`
	Timestamp int64          `json:"timestamp"`
	Read      bool           `json:"read"`
}

var (
	proactiveAlertsMu sync.RWMutex
	proactiveAlerts   = make([]ProactiveAlert, 0, 50)
	cpuHistory        = make([]float64, 0, 60)
	cpuHistoryMu      sync.Mutex
)

// pushProactiveAlert — 알림 큐에 추가 (최대 50개 유지)
func pushProactiveAlert(a ProactiveAlert) {
	proactiveAlertsMu.Lock()
	defer proactiveAlertsMu.Unlock()
	a.Timestamp = time.Now().Unix()
	if a.ID == "" {
		a.ID = "pa-" + time.Now().Format("20060102150405")
	}
	proactiveAlerts = append(proactiveAlerts, a)
	if len(proactiveAlerts) > 50 {
		proactiveAlerts = proactiveAlerts[len(proactiveAlerts)-50:]
	}
}

// getProactiveAlerts — 읽지 않은 알림 반환
func getProactiveAlerts(includeRead bool) []ProactiveAlert {
	proactiveAlertsMu.RLock()
	defer proactiveAlertsMu.RUnlock()
	out := make([]ProactiveAlert, 0, len(proactiveAlerts))
	for _, a := range proactiveAlerts {
		if includeRead || !a.Read {
			out = append(out, a)
		}
	}
	return out
}

// markAlertRead
func markAlertRead(id string) {
	proactiveAlertsMu.Lock()
	defer proactiveAlertsMu.Unlock()
	for i := range proactiveAlerts {
		if proactiveAlerts[i].ID == id {
			proactiveAlerts[i].Read = true
		}
	}
}

// checkProactiveConditions — 시스템 상태 평가 → 임계치 도달시 알림 생성
// 매 60초 호출. cpu/mem/disk 등은 OS-specific 호출 결과를 입력 받음
func checkProactiveConditions(cpu, mem, disk float64, cpuTemp float64) {
	// 1) 디스크 90%+ → critical
	if disk >= 90 {
		pushProactiveAlert(ProactiveAlert{
			Severity: "critical",
			Title:    "💾 디스크 공간 부족",
			Message:  "디스크 사용량이 " + intStr(disk) + "% 입니다. 정리가 시급합니다.",
			Action:   "clean",
			Params:   map[string]any{"aggressive": false},
		})
	} else if disk >= 80 {
		pushProactiveAlert(ProactiveAlert{
			Severity: "warning",
			Title:    "💾 디스크 사용량 주의",
			Message:  "디스크 사용량이 " + intStr(disk) + "% 입니다.",
			Action:   "clean",
		})
	}

	// 2) CPU 5분 연속 90%+
	cpuHistoryMu.Lock()
	cpuHistory = append(cpuHistory, cpu)
	if len(cpuHistory) > 5 {
		cpuHistory = cpuHistory[len(cpuHistory)-5:]
	}
	allHigh := len(cpuHistory) >= 5
	for _, c := range cpuHistory {
		if c < 90 {
			allHigh = false
			break
		}
	}
	cpuHistoryMu.Unlock()
	if allHigh {
		pushProactiveAlert(ProactiveAlert{
			Severity: "critical",
			Title:    "⚡ CPU 과부하 5분 지속",
			Message:  "CPU 사용률이 5분 연속 90% 이상입니다. 이상 프로세스 확인 권장.",
			Action:   "process_top",
		})
	}

	// 3) 메모리 95%+
	if mem >= 95 {
		pushProactiveAlert(ProactiveAlert{
			Severity: "warning",
			Title:    "🧠 메모리 거의 가득",
			Message:  "메모리 사용률 " + intStr(mem) + "%. 큰 작업 전 정리 권장.",
			Action:   "clean",
		})
	}

	// 4) CPU 온도 85°C+
	if cpuTemp >= 85 {
		pushProactiveAlert(ProactiveAlert{
			Severity: "critical",
			Title:    "🌡️ CPU 온도 위험",
			Message:  "CPU 온도 " + intStr(cpuTemp) + "°C. 쿨러/팬 점검 필요.",
			Action:   "scan",
		})
	}
}

// startProactiveAlertEngine — 60초마다 자동 검사 (기존 startProactiveMonitor 와 별개)
// 호출: 서버 시작 시 1회
func startProactiveAlertEngine(getStats func() (cpu, mem, disk, cpuTemp float64)) {
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for {
			<-ticker.C
			cpu, mem, disk, temp := getStats()
			if cpu > 0 || mem > 0 || disk > 0 {
				checkProactiveConditions(cpu, mem, disk, temp)
			}
		}
	}()
}

func intStr(f float64) string {
	if f < 0 {
		return "0"
	}
	if f > 100000 {
		return "100000+"
	}
	return formatFloatRound0(f)
}

func formatFloatRound0(f float64) string {
	// 1.0 → "1"; 67.4 → "67"
	i := int(f + 0.5)
	if i < 0 {
		i = 0
	}
	// 간단한 itoa
	if i == 0 {
		return "0"
	}
	digits := ""
	for i > 0 {
		digits = string(rune('0'+(i%10))) + digits
		i /= 10
	}
	return digits
}
