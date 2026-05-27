//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ══════════════════════════════════════════════════════════════════
//  Performance History — 성능 스냅샷 시계열 저장/조회
//  JSON 파일 기반 (SQLite 의존성 없음)
// ══════════════════════════════════════════════════════════════════

type PerfSnapshot struct {
	Timestamp string  `json:"timestamp"`
	CPU       float64 `json:"cpu"`
	Mem       float64 `json:"mem"`
	Disk      float64 `json:"disk"`
	CPUTemp   float64 `json:"cpu_temp"`
	GPU       float64 `json:"gpu,omitempty"`
	NetDown   float64 `json:"net_down"`
	NetUp     float64 `json:"net_up"`
}

type PerfHistory struct {
	Snapshots []PerfSnapshot `json:"snapshots"`
}

func historyFilePath() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = os.TempDir()
	}
	dir := filepath.Join(appData, "Nexus", "history")
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "perf_history.json")
}

func loadHistory() PerfHistory {
	data, err := os.ReadFile(historyFilePath())
	if err != nil {
		return PerfHistory{Snapshots: []PerfSnapshot{}}
	}
	var h PerfHistory
	if err := json.Unmarshal(data, &h); err != nil {
		return PerfHistory{Snapshots: []PerfSnapshot{}}
	}
	return h
}

func saveHistory(h PerfHistory) {
	// 최대 2016개 (7일 * 24시간 * 12 = 5분 간격) 유지
	if len(h.Snapshots) > 2016 {
		h.Snapshots = h.Snapshots[len(h.Snapshots)-2016:]
	}
	data, _ := json.Marshal(h)
	os.WriteFile(historyFilePath(), data, 0644)
}

// POST /api/history/snapshot — 현재 스냅샷 저장 (백그라운드에서 주기적 호출)
func handleHistorySnapshot(w http.ResponseWriter, r *http.Request) {
	saveCurrentSnapshot()
	h := loadHistory()
	snap := h.Snapshots[len(h.Snapshots)-1]

	json200(w, map[string]interface{}{
		"success":  true,
		"snapshot": snap,
		"total":    len(h.Snapshots),
	})
}

// GET /api/history/stats — 과거 성능 이력 조회
func handleHistoryStats(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	daysStr := r.URL.Query().Get("days")
	days := 7
	if daysStr != "" {
		fmt.Sscanf(daysStr, "%d", &days)
	}
	if days > 30 {
		days = 30
	}

	h := loadHistory()
	cutoff := time.Now().Add(-time.Duration(days) * 24 * time.Hour)

	var filtered []PerfSnapshot
	for _, s := range h.Snapshots {
		t, err := time.Parse("2006-01-02T15:04:05", s.Timestamp)
		if err == nil && t.After(cutoff) {
			filtered = append(filtered, s)
		}
	}

	// 정렬
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Timestamp < filtered[j].Timestamp
	})

	// 일별 집계
	dayMap := map[string][]PerfSnapshot{}
	for _, s := range filtered {
		day := s.Timestamp[:10]
		dayMap[day] = append(dayMap[day], s)
	}

	type DaySummary struct {
		Date    string  `json:"date"`
		AvgCPU  float64 `json:"avg_cpu"`
		MaxCPU  float64 `json:"max_cpu"`
		AvgMem  float64 `json:"avg_mem"`
		MaxMem  float64 `json:"max_mem"`
		AvgTemp float64 `json:"avg_temp"`
		MaxTemp float64 `json:"max_temp"`
		Samples int     `json:"samples"`
	}

	var dailySummaries []DaySummary
	for day, snaps := range dayMap {
		var sumCPU, maxCPU, sumMem, maxMem, sumTemp, maxTemp float64
		for _, s := range snaps {
			sumCPU += s.CPU
			sumMem += s.Mem
			sumTemp += s.CPUTemp
			if s.CPU > maxCPU {
				maxCPU = s.CPU
			}
			if s.Mem > maxMem {
				maxMem = s.Mem
			}
			if s.CPUTemp > maxTemp {
				maxTemp = s.CPUTemp
			}
		}
		n := float64(len(snaps))
		dailySummaries = append(dailySummaries, DaySummary{
			Date:    day,
			AvgCPU:  round2(sumCPU / n),
			MaxCPU:  round2(maxCPU),
			AvgMem:  round2(sumMem / n),
			MaxMem:  round2(maxMem),
			AvgTemp: round2(sumTemp / n),
			MaxTemp: round2(maxTemp),
			Samples: len(snaps),
		})
	}

	sort.Slice(dailySummaries, func(i, j int) bool {
		return dailySummaries[i].Date < dailySummaries[j].Date
	})

	// 전체 평균
	var totalCPU, totalMem float64
	for _, s := range filtered {
		totalCPU += s.CPU
		totalMem += s.Mem
	}
	n := float64(len(filtered))
	avgCPU, avgMem := 0.0, 0.0
	if n > 0 {
		avgCPU = round2(totalCPU / n)
		avgMem = round2(totalMem / n)
	}

	// 트렌드 분석 — 최근 vs 이전
	trend := "stable"
	if len(filtered) >= 20 {
		recentN := len(filtered) / 4
		var recentCPU, prevCPU float64
		for _, s := range filtered[len(filtered)-recentN:] {
			recentCPU += s.CPU
		}
		for _, s := range filtered[:recentN] {
			prevCPU += s.CPU
		}
		rc := recentCPU / float64(recentN)
		pc := prevCPU / float64(recentN)
		if rc > pc+10 {
			trend = "up"
		} else if rc < pc-10 {
			trend = "down"
		}
	}

	msg := fmt.Sprintf(msgT("최근 %d일 성능 분석 — 평균 CPU %.0f%%, 메모리 %.0f%%", "Performance analysis for last %d days — Avg CPU %.0f%%, Memory %.0f%%", lang), days, avgCPU, avgMem)

	json200(w, map[string]interface{}{
		"success":        true,
		"days":           days,
		"total_samples":  len(filtered),
		"snapshots":      filtered,
		"daily_summary":  dailySummaries,
		"avg_cpu":        avgCPU,
		"avg_mem":        avgMem,
		"cpu_trend":      trend,
		"message":        msg,
	})
}

// GET /api/history/anomalies — 이상 탐지 (평균 대비 이상 데이터)
func handleHistoryAnomalies(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	h := loadHistory()
	cutoff := time.Now().Add(-7 * 24 * time.Hour)

	var recent []PerfSnapshot
	for _, s := range h.Snapshots {
		t, err := time.Parse("2006-01-02T15:04:05", s.Timestamp)
		if err == nil && t.After(cutoff) {
			recent = append(recent, s)
		}
	}

	if len(recent) < 10 {
		json200(w, map[string]interface{}{
			"success":   false,
			"anomalies": []interface{}{},
			"message":   msgT("데이터가 부족해요. 최소 1일 이상 사용 후 이상 탐지가 가능해요.", "Not enough data. Anomaly detection requires at least 1 day of usage.", lang),
		})
		return
	}

	// 평균 + 표준편차 계산
	var sumCPU, sumMem float64
	for _, s := range recent {
		sumCPU += s.CPU
		sumMem += s.Mem
	}
	n := float64(len(recent))
	avgCPU := sumCPU / n
	avgMem := sumMem / n

	type Anomaly struct {
		Timestamp string  `json:"timestamp"`
		Type      string  `json:"type"`
		Value     float64 `json:"value"`
		AvgValue  float64 `json:"avg_value"`
		DiffPct   float64 `json:"diff_pct"`
		Message   string  `json:"message"`
	}

	var anomalies []Anomaly
	for _, s := range recent {
		if s.CPU > avgCPU*1.5 && s.CPU > 80 {
			anomalies = append(anomalies, Anomaly{
				Timestamp: s.Timestamp,
				Type:      "cpu_spike",
				Value:     s.CPU,
				AvgValue:  round2(avgCPU),
				DiffPct:   round2((s.CPU - avgCPU) / avgCPU * 100),
				Message:   fmt.Sprintf(msgT("CPU 급등: %.0f%% (평균 %.0f%%보다 %.0f%% 높음)", "CPU spike: %.0f%% (%.0f%% above avg %.0f%%)", lang), s.CPU, avgCPU, s.CPU-avgCPU),
			})
		}
		if s.Mem > avgMem*1.3 && s.Mem > 85 {
			anomalies = append(anomalies, Anomaly{
				Timestamp: s.Timestamp,
				Type:      "mem_spike",
				Value:     s.Mem,
				AvgValue:  round2(avgMem),
				DiffPct:   round2((s.Mem - avgMem) / avgMem * 100),
				Message:   fmt.Sprintf(msgT("메모리 급등: %.0f%% (평균 %.0f%%보다 %.0f%% 높음)", "Memory spike: %.0f%% (%.0f%% above avg %.0f%%)", lang), s.Mem, avgMem, s.Mem-avgMem),
			})
		}
		if s.CPUTemp > 80 {
			anomalies = append(anomalies, Anomaly{
				Timestamp: s.Timestamp,
				Type:      "temp_high",
				Value:     s.CPUTemp,
				AvgValue:  0,
				DiffPct:   0,
				Message:   fmt.Sprintf(msgT("과열 감지: %.0f°C", "Overheating detected: %.0f°C", lang), s.CPUTemp),
			})
		}
	}

	// 최근 10개만 반환
	if len(anomalies) > 10 {
		anomalies = anomalies[len(anomalies)-10:]
	}

	msg := fmt.Sprintf(msgT("이상 이벤트 %d개 발견됐어요", "%d anomaly event(s) detected", lang), len(anomalies))
	if len(anomalies) == 0 {
		msg = msgT("✅ 최근 7일간 이상 없어요. PC가 건강하게 동작하고 있어요!", "✅ No anomalies in the last 7 days. Your PC is running healthy!", lang)
	}

	json200(w, map[string]interface{}{
		"success":   true,
		"anomalies": anomalies,
		"avg_cpu":   round2(avgCPU),
		"avg_mem":   round2(avgMem),
		"message":   msg,
	})
}

func round2(v float64) float64 {
	return float64(int(v*100)) / 100
}

// startHistoryCollector — 5분 간격으로 성능 스냅샷 자동 저장
func startHistoryCollector() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	// 시작 직후 1회 저장
	saveCurrentSnapshot()

	for range ticker.C {
		saveCurrentSnapshot()
	}
}

func saveCurrentSnapshot() {
	cpu := getRealCPU()
	mem := float64(getMemoryUsage())
	_, total := getDiskSpace()
	free, _ := getDiskSpace()
	disk := 0.0
	if total > 0 {
		disk = float64(total-free) / float64(total) * 100
	}
	temp := getCPUTempEstimate()
	gpuUsage, _ := getGPUInfo()

	snap := PerfSnapshot{
		Timestamp: time.Now().Format("2006-01-02T15:04:05"),
		CPU:       cpu,
		Mem:       mem,
		Disk:      disk,
		CPUTemp:   temp,
		GPU:       gpuUsage,
	}
	h := loadHistory()
	h.Snapshots = append(h.Snapshots, snap)
	saveHistory(h)
}

// getCPUTempEstimate: PowerShell로 CPU 온도 추정 (WMI)
func getCPUTempEstimate() float64 {
	out, err := newHiddenCmd("powershell", "-NoProfile", "-Command",
		`(Get-WmiObject MSAcpi_ThermalZoneTemperature -Namespace "root/wmi" | Select-Object -First 1).CurrentTemperature`,
	).Output()
	if err != nil {
		return 0
	}
	val := strings.TrimSpace(string(out))
	if val == "" {
		return 0
	}
	// WMI 반환값: 켈빈×10 → 섭씨 변환
	raw := 0.0
	if _, err := fmt.Sscanf(val, "%f", &raw); err == nil && raw > 2000 {
		return (raw/10.0) - 273.15
	}
	return 0
}
