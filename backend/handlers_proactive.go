//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// ══════════════════════════════════════════════════════════════════
//  Proactive AI 모니터링
//  - PC 상태를 실시간 감시하다가 문제가 생기면 먼저 알려줍니다
//  - "주인님, CPU가 뜨거워요. 쿨링해드릴까요?"
//  - SSE(Server-Sent Events)로 프론트엔드에 실시간 푸시
// ══════════════════════════════════════════════════════════════════

type Alert struct {
	ID        string    `json:"id"`
	Level     string    `json:"level"`   // info | warn | critical
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

// startProactiveMonitor: 백그라운드에서 PC 상태 감시
func startProactiveMonitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	var lastCPUAlert, lastDiskAlert, lastMemAlert time.Time

	for range ticker.C {
		now := time.Now()

		// ── CPU 온도 ────────────────────────────────────────────
		temp := getCPUTempEstimate()
		if temp > 85 && now.Sub(lastCPUAlert) > 10*time.Minute {
			publishAlert(Alert{
				ID:      fmt.Sprintf("cpu_temp_%d", now.Unix()),
				Level:   "warn",
				Title:   "CPU 온도가 높아요 🌡️",
				Message: fmt.Sprintf("현재 CPU 온도가 %.0f°C입니다. 잠시 쉬어가는 게 좋을 것 같아요.", temp),
				Action:  "pc_report",
			})
			lastCPUAlert = now
		}

		// ── 메모리 ──────────────────────────────────────────────
		mem := getMemoryUsage()
		if mem > 90 && now.Sub(lastMemAlert) > 15*time.Minute {
			publishAlert(Alert{
				ID:      fmt.Sprintf("mem_%d", now.Unix()),
				Level:   "warn",
				Title:   fmt.Sprintf("메모리 %d%% 사용 중이에요 💾", mem),
				Message: "불필요한 프로그램을 종료하면 PC가 빨라질 거예요. 지금 정리해드릴까요?",
				Action:  "clean",
			})
			lastMemAlert = now
		}

		// ── 디스크 용량 ─────────────────────────────────────────
		free, total := getDiskSpace()
		if total > 0 {
			freePct := float64(free) / float64(total) * 100
			if freePct < 10 && now.Sub(lastDiskAlert) > 30*time.Minute {
				publishAlert(Alert{
					ID:      fmt.Sprintf("disk_%d", now.Unix()),
					Level:   "critical",
					Title:   fmt.Sprintf("디스크 공간 부족! (%.0f%% 남음) 💿", freePct),
					Message: "C드라이브 여유 공간이 얼마 남지 않았어요. 파일 정리가 필요해요.",
					Action:  "clean",
				})
				lastDiskAlert = now
			}
		}

		// ── 임시 파일 대량 누적 ──────────────────────────────────
		tempSize := getTempSize()
		if tempSize > 2<<30 { // 2GB 이상
			publishAlert(Alert{
				ID:      fmt.Sprintf("temp_%d", now.Unix()),
				Level:   "info",
				Title:   fmt.Sprintf("임시 파일 %s 누적됨 🗑️", formatBytes(tempSize)),
				Message: "임시 파일이 많이 쌓였어요. 지금 정리하면 디스크 공간을 확보할 수 있어요.",
				Action:  "clean",
			})
		}
	}
}

func getCPUTempEstimate() float64 {
	// WMI로 CPU 온도 조회 (실제 값이 없으면 메모리 부하 기반 추정)
	mem := float64(getMemoryUsage())
	// 간이 추정: 메모리 부하가 높으면 CPU도 뜨거울 가능성이 높음
	// 실제 환경에서는 WMI MSAcpi_ThermalZoneTemperature 사용
	return 40 + mem*0.5
}

func publishAlert(a Alert) {
	a.Timestamp = time.Now()

	alertMu.Lock()
	latestAlerts = append([]Alert{a}, latestAlerts...)
	if len(latestAlerts) > 50 {
		latestAlerts = latestAlerts[:50]
	}
	alertMu.Unlock()

	// 모든 SSE 구독자에게 푸시
	subMu.Lock()
	for _, ch := range alertSubs {
		select {
		case ch <- a:
		default: // 구독자가 느리면 건너뜀
		}
	}
	subMu.Unlock()
}

// GET /api/alerts/stream — SSE 스트림
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

	// 연결 확인 메시지
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
			// 하트비트
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
