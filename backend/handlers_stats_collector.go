//go:build windows

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type StatEntry struct {
	Time     string  `json:"time"`
	CPU      float64 `json:"cpu"`
	Mem      float64 `json:"mem"`
	DiskFree float64 `json:"disk_free"`
}

func startStatsCollector() {
	collectAndSaveStat()
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		collectAndSaveStat()
	}
}

func collectAndSaveStat() {
	stats, err := getWindowsStats()
	if err != nil {
		return
	}
	cpu, ok1 := stats["cpu"].(float64)
	mem, ok2 := stats["mem"].(float64)
	if !ok1 || !ok2 {
		return
	}
	free, total := getDiskSpace()
	if total == 0 {
		return
	}
	diskFree := float64(free) / (1 << 30)

	entry := StatEntry{
		Time:     time.Now().Format("15:04"),
		CPU:      cpu,
		Mem:      mem,
		DiskFree: diskFree,
	}

	date := time.Now().Format("2006-01-02")
	path := filepath.Join(nexusDataDir(), "stats_"+date+".json")

	var entries []StatEntry
	if raw, err := os.ReadFile(path); err == nil {
		json.Unmarshal(raw, &entries)
	}
	entries = append(entries, entry)

	raw, _ := json.Marshal(entries)
	os.WriteFile(path, raw, 0644)
}

func loadDailyStats(date string) []StatEntry {
	path := filepath.Join(nexusDataDir(), "stats_"+date+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var entries []StatEntry
	json.Unmarshal(raw, &entries)
	return entries
}
