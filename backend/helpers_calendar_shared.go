// helpers_calendar_shared.go — 플랫폼 공통 캘린더 로컬 JSON 헬퍼 (빌드 태그 없음)
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// CalEvent: 로컬 JSON 캘린더 이벤트
type CalEvent struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Date     string `json:"date"`
	Time     string `json:"time"`
	Duration int    `json:"duration"`
	Location string `json:"location"`
}

func nexusDataDir() string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".nexus")
	os.MkdirAll(dir, 0755)
	return dir
}

func calendarPath() string { return filepath.Join(nexusDataDir(), "calendar.json") }

func loadEvents() []CalEvent {
	data, err := os.ReadFile(calendarPath())
	if err != nil {
		return []CalEvent{}
	}
	var evs []CalEvent
	json.Unmarshal(data, &evs)
	return evs
}

func saveEvents(evs []CalEvent) {
	data, _ := json.MarshalIndent(evs, "", "  ")
	os.WriteFile(calendarPath(), data, 0644)
}
