// phase_d_telemetry.go — Phase D: 모든 명령 응답 메타 수집 (cross-platform)
// 사장님 비전: "자가 치유 자비스" — 12개 분야 Agent가 분석할 raw 데이터
//
// 수집 항목:
//   - action / message 길이 / 응답 시간 / 카드 타입 / 페르소나 / 에러
//   - 사용자 거부 (재질문/취소) — 향후 프론트 hook
//
// 저장: 메모리 ring buffer (최근 1000건) + ~/.nexus/telemetry.jsonl (영구)

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// TelemetryEntry — 단일 명령 호출 메타
type TelemetryEntry struct {
	Timestamp   int64          `json:"ts"`
	UserID      string         `json:"uid"`
	UserMessage string         `json:"msg"`
	Action      string         `json:"action"`
	CardType    string         `json:"card_type"`
	Persona     string         `json:"persona,omitempty"`
	MessageLen  int            `json:"msg_len"`
	DurationMs  int64          `json:"dur_ms"`
	Success     bool           `json:"success"`
	Empty       bool           `json:"empty"`   // 응답 message 빈 string
	Clarify     bool           `json:"clarify"` // action == "clarify"
	Error       string         `json:"error,omitempty"`
	Params      map[string]any `json:"params,omitempty"`
}

var (
	telemetryMu    sync.RWMutex
	telemetryRing  = make([]TelemetryEntry, 0, 1000)
	telemetryStart = time.Now()
)

const telemetryRingCap = 1000

// recordTelemetry — 명령 응답 직후 호출
func recordTelemetry(e TelemetryEntry) {
	if e.Timestamp == 0 {
		e.Timestamp = time.Now().Unix()
	}
	telemetryMu.Lock()
	telemetryRing = append(telemetryRing, e)
	if len(telemetryRing) > telemetryRingCap {
		telemetryRing = telemetryRing[len(telemetryRing)-telemetryRingCap:]
	}
	telemetryMu.Unlock()
	// 비동기 영구 저장 (실패해도 무시)
	go func() {
		path := telemetryFilePath()
		os.MkdirAll(filepath.Dir(path), 0755)
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return
		}
		defer f.Close()
		data, _ := json.Marshal(e)
		f.Write(data)
		f.Write([]byte("\n"))
	}()
}

func telemetryFilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".nexus", "telemetry.jsonl")
}

// getTelemetrySnapshot — Agent 분석용 스냅샷 (최근 N건)
func getTelemetrySnapshot(maxN int) []TelemetryEntry {
	telemetryMu.RLock()
	defer telemetryMu.RUnlock()
	if maxN <= 0 || maxN > len(telemetryRing) {
		maxN = len(telemetryRing)
	}
	out := make([]TelemetryEntry, maxN)
	copy(out, telemetryRing[len(telemetryRing)-maxN:])
	return out
}

// getTelemetryStats — 분석된 핵심 통계
type TelemetryStats struct {
	Total           int                `json:"total"`
	AvgDurationMs   float64            `json:"avg_duration_ms"`
	P95DurationMs   float64            `json:"p95_duration_ms"`
	EmptyRate       float64            `json:"empty_rate"`
	ClarifyRate     float64            `json:"clarify_rate"`
	ErrorRate       float64            `json:"error_rate"`
	ActionCounts    map[string]int     `json:"action_counts"`
	SlowActions     map[string]float64 `json:"slow_actions"`     // action → avg dur (>3000ms)
	EmptyByAction   map[string]int     `json:"empty_by_action"`
	WindowStartTs   int64              `json:"window_start_ts"`
	WindowEndTs     int64              `json:"window_end_ts"`
}

func computeTelemetryStats(entries []TelemetryEntry) TelemetryStats {
	stats := TelemetryStats{
		ActionCounts:  make(map[string]int),
		SlowActions:   make(map[string]float64),
		EmptyByAction: make(map[string]int),
	}
	if len(entries) == 0 {
		return stats
	}
	stats.Total = len(entries)
	stats.WindowStartTs = entries[0].Timestamp
	stats.WindowEndTs = entries[len(entries)-1].Timestamp

	durations := make([]int64, 0, len(entries))
	actionDur := make(map[string][]int64)
	emptyCnt, clarifyCnt, errorCnt := 0, 0, 0
	for _, e := range entries {
		durations = append(durations, e.DurationMs)
		stats.ActionCounts[e.Action]++
		actionDur[e.Action] = append(actionDur[e.Action], e.DurationMs)
		if e.Empty {
			emptyCnt++
			stats.EmptyByAction[e.Action]++
		}
		if e.Clarify {
			clarifyCnt++
		}
		if e.Error != "" || !e.Success {
			errorCnt++
		}
	}
	// avg / p95
	var sum int64
	for _, d := range durations {
		sum += d
	}
	stats.AvgDurationMs = float64(sum) / float64(len(durations))
	// 간단한 p95
	sorted := make([]int64, len(durations))
	copy(sorted, durations)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j] < sorted[i] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	p95Idx := int(float64(len(sorted))*0.95)
	if p95Idx >= len(sorted) {
		p95Idx = len(sorted) - 1
	}
	stats.P95DurationMs = float64(sorted[p95Idx])

	stats.EmptyRate = float64(emptyCnt) / float64(stats.Total)
	stats.ClarifyRate = float64(clarifyCnt) / float64(stats.Total)
	stats.ErrorRate = float64(errorCnt) / float64(stats.Total)

	// 액션별 평균 시간 (3초 초과만)
	for action, durs := range actionDur {
		var s int64
		for _, d := range durs {
			s += d
		}
		avg := float64(s) / float64(len(durs))
		if avg > 3000 {
			stats.SlowActions[action] = avg
		}
	}
	return stats
}
