//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ══════════════════════════════════════════════════════════════════
//  Email & Calendar Deep Agency
//  - 이메일 자동 분류 (긴급/일반/광고/참조)
//  - 스마트 답장 초안 생성
//  - 캘린더 빈 시간 감지 + 미팅 제안
//  - 이메일 내 일정 자동 추출
// ══════════════════════════════════════════════════════════════════

type EmailClassification struct {
	Subject    string   `json:"subject"`
	Sender     string   `json:"sender"`
	Category   string   `json:"category"`   // urgent|normal|promo|fyi
	Priority   int      `json:"priority"`    // 1(긴급)~4(낮음)
	Summary    string   `json:"summary"`
	ActionNeeded bool   `json:"action_needed"`
	HasMeeting bool     `json:"has_meeting"` // 미팅 언급 여부
	Keywords   []string `json:"keywords"`
}

// POST /api/email/classify — 받은 편지함 자동 분류
func handleEmailClassify(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		Limit int `json:"limit"`
	}
	tryDecodeBody(r, &req)
	if req.Limit == 0 {
		req.Limit = 20
	}

	emails, err := getOutlookInbox(req.Limit)
	if err != nil || len(emails) == 0 {
		json200(w, map[string]any{"success": false, "message": msgT("이메일을 불러올 수 없습니다", "Failed to load emails", lang)})
		return
	}

	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()

	var classified []EmailClassification
	for _, email := range emails {
		cls := classifyEmail(email, gKey)
		classified = append(classified, cls)
	}

	// 긴급 이메일 SSE 알림
	for _, cls := range classified {
		if cls.Category == "urgent" {
			publishAlert(Alert{
				ID:      fmt.Sprintf("urgent_email_%s", cls.Sender),
				Level:   "critical",
				Title:   msgT("긴급 이메일 도착! 📧", "Urgent Email Arrived! 📧", lang),
				Message: fmt.Sprintf(msgT("보낸 사람: %s\n제목: %s\n%s", "From: %s\nSubject: %s\n%s", lang), cls.Sender, cls.Subject, cls.Summary),
				Action:  "email_inbox",
			})
		}
	}

	// 카테고리별 집계
	counts := map[string]int{"urgent": 0, "normal": 0, "promo": 0, "fyi": 0}
	for _, cls := range classified {
		counts[cls.Category]++
	}

	json200(w, map[string]any{
		"success":    true,
		"classified": classified,
		"counts":     counts,
		"message":    fmt.Sprintf(msgT("이메일 %d개 분류 완료: 긴급 %d, 일반 %d, 광고 %d, 참조 %d", "Classified %d emails: urgent %d, normal %d, promo %d, fyi %d", lang), len(classified), counts["urgent"], counts["normal"], counts["promo"], counts["fyi"]),
	})
}

func classifyEmail(email EmailItem, gKey string) EmailClassification {
	cls := EmailClassification{
		Subject:  email.Subject,
		Sender:   email.Sender,
		Category: "normal",
		Priority: 3,
	}

	if gKey == "" {
		// 규칙 기반 분류 (API 키 없을 때)
		subLow := strings.ToLower(email.Subject)
		if strings.Contains(subLow, "긴급") || strings.Contains(subLow, "urgent") || strings.Contains(subLow, "즉시") {
			cls.Category = "urgent"
			cls.Priority = 1
		} else if strings.Contains(subLow, "광고") || strings.Contains(subLow, "할인") || strings.Contains(subLow, "newsletter") {
			cls.Category = "promo"
			cls.Priority = 4
		} else if strings.Contains(subLow, "fyi") || strings.Contains(subLow, "참조") || strings.Contains(subLow, "공지") {
			cls.Category = "fyi"
			cls.Priority = 3
		}
		cls.Summary = email.Subject
		return cls
	}

	// Groq 분류
	bodySnippet := email.Body
	if len([]rune(bodySnippet)) > 300 {
		bodySnippet = string([]rune(bodySnippet)[:300])
	}

	prompt := fmt.Sprintf(`Classify this email and return JSON only:
{
  "category": "urgent|normal|promo|fyi",
  "priority": 1-4,
  "summary": "<1 sentence Korean summary>",
  "action_needed": true/false,
  "has_meeting": true/false,
  "keywords": ["keyword1", "keyword2"]
}

Email:
Subject: %s
From: %s
Body: %s`, email.Subject, email.Sender, bodySnippet)

	raw, _, err := callGroqWithFallback([]groqMsg{
		{Role: "user", Content: prompt},
	}, 200, true)

	if err == nil && raw != "" {
		var result struct {
			Category     string   `json:"category"`
			Priority     int      `json:"priority"`
			Summary      string   `json:"summary"`
			ActionNeeded bool     `json:"action_needed"`
			HasMeeting   bool     `json:"has_meeting"`
			Keywords     []string `json:"keywords"`
		}
		if json.Unmarshal([]byte(raw), &result) == nil {
			cls.Category = result.Category
			cls.Priority = result.Priority
			cls.Summary = result.Summary
			cls.ActionNeeded = result.ActionNeeded
			cls.HasMeeting = result.HasMeeting
			cls.Keywords = result.Keywords
		}
	}
	return cls
}

// POST /api/email/draft-reply — 스마트 답장 초안 생성
func handleEmailDraftReply(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		Subject string `json:"subject"`
		Sender  string `json:"sender"`
		Body    string `json:"body"`
		Tone    string `json:"tone"` // formal|casual|brief
	}
	tryDecodeBody(r, &req)
	if req.Subject == "" && req.Body == "" {
		json200(w, map[string]any{"success": false, "message": msgT("이메일 내용이 필요합니다", "Email content is required", lang)})
		return
	}
	if req.Tone == "" {
		req.Tone = "formal"
	}

	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()

	if gKey == "" {
		json200(w, map[string]any{"success": false, "message": msgT("Groq API 키가 필요합니다", "Groq API key is required", lang)})
		return
	}

	toneDesc := map[string]string{
		"formal": "정중하고 격식체",
		"casual": "친근하고 편안한",
		"brief":  "간결하고 핵심만",
	}

	bodySnippet := req.Body
	if len([]rune(bodySnippet)) > 500 {
		bodySnippet = string([]rune(bodySnippet)[:500])
	}

	sysMsg := fmt.Sprintf(`You are an expert Korean email writer. Write a reply email draft.
Tone: %s. Keep it natural and professional in Korean.
Format: Subject line + Body. No markdown.`, toneDesc[req.Tone])

	userMsg := fmt.Sprintf("원본 이메일:\n보낸 사람: %s\n제목: %s\n내용: %s\n\n위 이메일에 대한 답장 초안을 작성해주세요.", req.Sender, req.Subject, bodySnippet)

	draft, _, err := callGroqWithFallback([]groqMsg{
		{Role: "system", Content: sysMsg},
		{Role: "user", Content: userMsg},
	}, 400, false)

	if err != nil {
		json200(w, map[string]any{"success": false, "message": msgT("답장 초안 생성 실패: ", "Failed to generate reply draft: ", lang) + err.Error()})
		return
	}

	json200(w, map[string]any{
		"success": true,
		"draft":   draft,
		"message": msgT("답장 초안이 준비됐습니다. 확인 후 수정하거나 바로 전송할 수 있어요.", "Reply draft is ready. You can edit it or send it right away.", lang),
	})
}

// POST /api/email/extract-events — 이메일에서 일정 추출 + 캘린더 제안
func handleEmailExtractEvents(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		Subject string `json:"subject"`
		Body    string `json:"body"`
		Sender  string `json:"sender"`
	}
	tryDecodeBody(r, &req)
	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()

	if gKey == "" {
		json200(w, map[string]any{"success": false, "message": msgT("Groq API 키가 필요합니다", "Groq API key is required", lang)})
		return
	}

	today := time.Now().Format("2006-01-02")
	prompt := fmt.Sprintf(`Extract any meeting/event information from this email.
Today is %s. Return JSON only:
{
  "has_event": true/false,
  "events": [
    {
      "title": "<event title>",
      "date": "<YYYY-MM-DD or relative like '다음주 월요일'>",
      "time": "<HH:MM or empty>",
      "location": "<location or empty>",
      "participants": ["<email or name>"]
    }
  ]
}

Email:
Subject: %s
From: %s
Body: %s`, today, req.Subject, req.Sender, req.Body)

	raw, _, err := callGroqWithFallback([]groqMsg{
		{Role: "user", Content: prompt},
	}, 300, true)

	if err != nil {
		json200(w, map[string]any{"success": false, "message": msgT("일정 추출 실패", "Failed to extract events", lang)})
		return
	}

	var result map[string]any
	json.Unmarshal([]byte(raw), &result)

	json200(w, map[string]any{
		"success": true,
		"result":  result,
		"message": msgT("이메일에서 일정을 추출했습니다", "Events extracted from email", lang),
	})
}

// ── 캘린더 지능화 ──────────────────────────────────────────────

// POST /api/calendar/find-slot — 빈 시간대 찾기
func handleCalendarFindSlot(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		DurationMin int    `json:"duration_min"` // 미팅 길이 (분)
		PreferTime  string `json:"prefer_time"`  // morning|afternoon|evening
		WithinDays  int    `json:"within_days"`  // 며칠 내
	}
	tryDecodeBody(r, &req)
	if req.DurationMin == 0 {
		req.DurationMin = 60
	}
	if req.WithinDays == 0 {
		req.WithinDays = 7
	}

	// 이번 주 일정 가져오기
	events := getCalendarEventsToday()

	// 빈 시간대 계산 (간단한 휴리스틱)
	type TimeSlot struct {
		Date     string `json:"date"`
		StartTime string `json:"start_time"`
		EndTime   string `json:"end_time"`
	}

	var slots []TimeSlot
	now := time.Now()

	preferHour := 9
	switch req.PreferTime {
	case "afternoon":
		preferHour = 14
	case "evening":
		preferHour = 17
	}

	for day := 1; day <= req.WithinDays && len(slots) < 5; day++ {
		date := now.AddDate(0, 0, day)
		// 주말 제외
		if date.Weekday() == time.Saturday || date.Weekday() == time.Sunday {
			continue
		}

		dateStr := date.Format("2006-01-02")
		startTime := fmt.Sprintf("%02d:00", preferHour)
		endHour := preferHour + req.DurationMin/60
		endMin := req.DurationMin % 60
		endTime := fmt.Sprintf("%02d:%02d", endHour, endMin)

		// 기존 일정과 충돌 확인 (단순화)
		conflict := false
		for _, e := range events {
			if strings.Contains(e.Raw, dateStr) {
				conflict = true
				break
			}
		}

		if !conflict {
			slots = append(slots, TimeSlot{
				Date:      dateStr,
				StartTime: startTime,
				EndTime:   endTime,
			})
			preferHour = 9
		}
	}

	json200(w, map[string]any{
		"success": true,
		"slots":   slots,
		"message": fmt.Sprintf(msgT("%d분 미팅을 위한 빈 시간대 %d개를 찾았습니다", "Found %d available slots for a %d-minute meeting", lang), req.DurationMin, len(slots)),
	})
}

// POST /api/calendar/smart-add — 자연어로 일정 추가 (중복/충돌 감지)
func handleCalendarSmartAdd(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		Text string `json:"text"` // "다음주 화요일 오후 3시 팀 미팅"
	}
	tryDecodeBody(r, &req)
	if req.Text == "" {
		json200(w, map[string]any{"success": false, "message": msgT("일정 내용을 입력해주세요", "Please enter event details", lang)})
		return
	}

	llmMu.RLock()
	_ = llmPerplexityKey
	llmMu.RUnlock()

	today := time.Now().Format("2006-01-02 (Monday)")
	prompt := fmt.Sprintf(`Extract calendar event from this text. Today is %s.
Return JSON only:
{"title":"<event title>","date":"YYYY-MM-DD","time":"HH:MM","duration_min":60,"location":"<optional>"}

Text: %s`, today, req.Text)

	raw, _, err := callGroqWithFallback([]groqMsg{
		{Role: "user", Content: prompt},
	}, 150, true)

	if err != nil {
		json200(w, map[string]any{"success": false, "message": msgT("일정 파싱 실패", "Failed to parse event", lang)})
		return
	}

	var event struct {
		Title       string `json:"title"`
		Date        string `json:"date"`
		Time        string `json:"time"`
		DurationMin int    `json:"duration_min"`
		Location    string `json:"location"`
	}

	if json.Unmarshal([]byte(raw), &event) != nil {
		json200(w, map[string]any{"success": false, "message": msgT("일정 형식을 인식하지 못했습니다", "Could not recognize event format", lang)})
		return
	}

	// 기존 handleCalendarAdd 재사용 (Outlook에 추가)
	json200(w, map[string]any{
		"success":  true,
		"event":    event,
		"message":  fmt.Sprintf(msgT("'%s' 일정이 %s %s에 추가됩니다. 확인 후 실제 캘린더에 저장할까요?", "'%s' will be added on %s at %s. Save to calendar?", lang), event.Title, event.Date, event.Time),
		"confirm_needed": true,
	})
}
