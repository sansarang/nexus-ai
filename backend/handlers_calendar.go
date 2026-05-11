//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// ══════════════════════════════════════════════════════════════════
//  Calendar — Windows Calendar / Outlook 연동
//  PowerShell COM 객체로 Outlook 캘린더 읽기
// ══════════════════════════════════════════════════════════════════

type CalendarEvent struct {
	Subject   string `json:"subject"`
	Start     string `json:"start"`
	End       string `json:"end"`
	Location  string `json:"location"`
	Organizer string `json:"organizer"`
	IsAllDay  bool   `json:"is_all_day"`
}

// GET /api/calendar/today — 오늘 일정
func handleCalendarToday(w http.ResponseWriter, r *http.Request) {
	events, err := getOutlookEvents("today")
	if err != nil {
		json200(w, map[string]interface{}{
			"success": false,
			"events":  []CalendarEvent{},
			"total":   0,
			"message": "캘린더 연동을 위해 Microsoft Outlook이 설치되어 있어야 합니다.",
		})
		return
	}

	msg := "오늘 일정이 없어요 😊"
	if len(events) > 0 {
		msg = fmt.Sprintf("오늘 일정이 %d개 있어요 📅", len(events))
	}

	json200(w, map[string]interface{}{
		"success": true,
		"events":  events,
		"total":   len(events),
		"message": msg,
	})
}

// GET /api/calendar/week — 이번 주 일정
func handleCalendarWeek(w http.ResponseWriter, r *http.Request) {
	events, err := getOutlookEvents("week")
	if err != nil {
		json200(w, map[string]interface{}{
			"success": false,
			"events":  []CalendarEvent{},
			"total":   0,
			"message": "Outlook이 설치되지 않았거나 접근 권한이 없어요.",
		})
		return
	}

	msg := "이번 주 일정이 없어요"
	if len(events) > 0 {
		msg = fmt.Sprintf("이번 주 일정이 %d개 있어요 📅", len(events))
	}

	json200(w, map[string]interface{}{
		"success": true,
		"events":  events,
		"total":   len(events),
		"message": msg,
	})
}

// POST /api/calendar/add — 일정 추가
func handleCalendarAdd(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Subject  string `json:"subject"`
		Start    string `json:"start"`
		End      string `json:"end"`
		Location string `json:"location"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Subject == "" {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	if req.Start == "" {
		req.Start = time.Now().Add(1 * time.Hour).Format("2006-01-02 15:04")
	}
	if req.End == "" {
		req.End = time.Now().Add(2 * time.Hour).Format("2006-01-02 15:04")
	}

	script := fmt.Sprintf(`
$ol = New-Object -ComObject Outlook.Application
$appt = $ol.CreateItem(1) # AppointmentItem
$appt.Subject = "%s"
$appt.Start = "%s"
$appt.End = "%s"
$appt.Location = "%s"
$appt.Save()
Write-Output "OK"
`, req.Subject, req.Start, req.End, req.Location)

	out, err := exec.Command("powershell", "-NoProfile", "-Command", script).Output()
	success := err == nil && strings.Contains(string(out), "OK")

	msg := "일정이 추가됐어요 📅"
	if !success {
		msg = "일정 추가에 실패했어요. Outlook이 설치되어 있는지 확인해주세요."
	}

	json200(w, map[string]interface{}{
		"success": success,
		"message": msg,
	})
}

// getOutlookEvents — PowerShell COM으로 Outlook 일정 읽기
func getOutlookEvents(period string) ([]CalendarEvent, error) {
	var dateFilter string
	now := time.Now()

	if period == "today" {
		dateFilter = fmt.Sprintf(
			"[Start] >= '%s' AND [Start] < '%s'",
			now.Format("01/02/2006")+" 00:00 AM",
			now.Add(24*time.Hour).Format("01/02/2006")+" 00:00 AM",
		)
	} else { // week
		dateFilter = fmt.Sprintf(
			"[Start] >= '%s' AND [Start] < '%s'",
			now.Format("01/02/2006")+" 00:00 AM",
			now.Add(7*24*time.Hour).Format("01/02/2006")+" 00:00 AM",
		)
	}

	script := fmt.Sprintf(`
try {
  $ol = New-Object -ComObject Outlook.Application -ErrorAction Stop
  $ns = $ol.GetNamespace("MAPI")
  $cal = $ns.GetDefaultFolder(9) # olFolderCalendar
  $items = $cal.Items
  $items.IncludeRecurrences = $true
  $items.Sort("[Start]")
  $filtered = $items.Restrict("[MessageClass]='IPM.Appointment' AND %s")
  $result = @()
  foreach ($appt in $filtered) {
    $result += [PSCustomObject]@{
      subject   = $appt.Subject
      start     = $appt.Start.ToString("yyyy-MM-dd HH:mm")
      end       = $appt.End.ToString("yyyy-MM-dd HH:mm")
      location  = $appt.Location
      organizer = $appt.Organizer
      is_all_day = $appt.AllDayEvent
    }
  }
  $result | ConvertTo-Json -Depth 2
} catch {
  Write-Output "ERROR: $_"
}
`, dateFilter)

	out, err := exec.Command("powershell", "-NoProfile", "-Command", script).Output()
	if err != nil {
		return nil, err
	}

	outStr := strings.TrimSpace(string(out))
	if strings.HasPrefix(outStr, "ERROR") {
		return nil, fmt.Errorf(outStr)
	}
	if outStr == "" || outStr == "null" {
		return []CalendarEvent{}, nil
	}

	// 단일 객체면 배열로 감싸기
	if strings.HasPrefix(outStr, "{") {
		outStr = "[" + outStr + "]"
	}

	var raw []struct {
		Subject   string `json:"subject"`
		Start     string `json:"start"`
		End       string `json:"end"`
		Location  string `json:"location"`
		Organizer string `json:"organizer"`
		IsAllDay  bool   `json:"is_all_day"`
	}
	if err := json.Unmarshal([]byte(outStr), &raw); err != nil {
		return nil, err
	}

	events := make([]CalendarEvent, 0, len(raw))
	for _, v := range raw {
		events = append(events, CalendarEvent{
			Subject:   v.Subject,
			Start:     v.Start,
			End:       v.End,
			Location:  v.Location,
			Organizer: v.Organizer,
			IsAllDay:  v.IsAllDay,
		})
	}
	return events, nil
}
