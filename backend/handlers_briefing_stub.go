//go:build !windows

package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func startBriefingScheduler() {}

func handleBriefingNow(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	hour := now.Hour()
	var greeting string
	switch {
	case hour < 12:
		greeting = "좋은 아침이에요"
	case hour < 18:
		greeting = "좋은 오후예요"
	default:
		greeting = "좋은 저녁이에요"
	}

	client := &http.Client{Timeout: 6 * time.Second}

	// 1. 날씨
	weatherInfo := ""
	if resp, err := client.Get("http://127.0.0.1:17891/api/weather?city=Seoul"); err == nil {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		weatherInfo = string(body)
	}

	// 2. PC 상태
	statsInfo := ""
	if resp, err := client.Get("http://127.0.0.1:17891/api/stats"); err == nil {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		statsInfo = string(body)
	}

	// 3. 오늘 뉴스 (Tavily)
	newsInfo := ""
	llmMu.RLock()
	tKey := llmTavilyKey
	llmMu.RUnlock()
	if tKey != "" {
		tr, ok := tavilySearch(tKey, "오늘 주요 뉴스 한국 2025", 5)
		if ok {
			newsInfo = tr.Summary
		}
	}

	// 4. 주식 시세
	stockInfo := stockBriefSummary()

	// 5. 오늘 일정 (캘린더)
	calendarInfo := ""
	if resp, err := client.Get("http://127.0.0.1:17891/api/calendar/today"); err == nil {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		calendarInfo = string(body)
	}

	// LLM 통합 브리핑 생성
	var sections []string
	sections = append(sections, fmt.Sprintf("현재 시각: %s", now.Format("2006-01-02 15:04 (Monday)")))
	if weatherInfo != "" {
		sections = append(sections, "날씨: "+weatherInfo)
	}
	if calendarInfo != "" {
		sections = append(sections, "오늘 일정: "+calendarInfo)
	}
	if newsInfo != "" {
		sections = append(sections, "주요 뉴스:\n"+newsInfo)
	}
	if statsInfo != "" {
		sections = append(sections, "PC 상태: "+statsInfo)
	}
	if stockInfo != "" {
		sections = append(sections, "주식: "+stockInfo)
	}

	prompt := fmt.Sprintf(`당신은 Nexus AI 비서입니다. 사용자에게 %s 인사와 함께 아침 브리핑을 해줘.

다음 데이터를 참고해서 한국어로 친절하게 요약해줘:
%s

형식:
- 인사 + 날짜
- 날씨 한 줄
- 오늘 일정 (있으면)
- 주요 뉴스 2~3개 요약
- PC 상태 한 줄
- 주식 동향 (있으면)
- 오늘의 한마디

300자 이내로 간결하게.`, greeting, strings.Join(sections, "\n\n"))

	briefingText, _, _ := callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 500, false)
	if strings.TrimSpace(briefingText) == "" {
		briefingText = fmt.Sprintf("%s! 오늘은 %s입니다.", greeting, now.Format("1월 2일"))
	}

	json200(w, map[string]any{
		"success":  true,
		"briefing": briefingText,
		"greeting": greeting,
		"datetime": now.Format("2006-01-02 15:04"),
		"sections": map[string]string{
			"weather":  weatherInfo,
			"stats":    statsInfo,
			"news":     newsInfo,
			"stock":    stockInfo,
			"calendar": calendarInfo,
		},
	})
}

func handleBriefingConfig(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{
		"success": true,
		"config": map[string]any{
			"enabled": true,
			"time":    "08:00",
			"include": []string{"weather", "calendar", "news", "stats", "stock"},
		},
	})
}
