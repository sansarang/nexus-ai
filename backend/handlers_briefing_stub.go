//go:build !windows

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func startBriefingScheduler() {
	go func() {
		for {
			now := time.Now()
			// 매일 오전 8시 트리거
			next := time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, now.Location())
			if now.After(next) {
				next = next.Add(24 * time.Hour)
			}
			time.Sleep(time.Until(next))
			eng := IsUserEng()
			llmMu.RLock()
			tKey := llmTavilyKey
			llmMu.RUnlock()
			var sections []string
			// 날씨
			city := "Seoul"
			if eng {
				city = "New York"
			}
			wURL := fmt.Sprintf("https://wttr.in/%s?format=j1", city)
			if wr, err := (&http.Client{Timeout: 5 * time.Second}).Get(wURL); err == nil {
				defer wr.Body.Close()
				var wraw map[string]any
				if json.NewDecoder(wr.Body).Decode(&wraw) == nil {
					if cc, ok := wraw["current_condition"].([]any); ok && len(cc) > 0 {
						c := cc[0].(map[string]any)
						temp := fmt.Sprintf("%v", c["temp_C"])
						desc := ""
						if wds, ok := c["weatherDesc"].([]any); ok && len(wds) > 0 {
							desc = fmt.Sprintf("%v", (wds[0].(map[string]any))["value"])
						}
						if eng {
							sections = append(sections, fmt.Sprintf("🌤️ Weather: %s°C, %s", temp, desc))
						} else {
							sections = append(sections, fmt.Sprintf("🌤️ 날씨: %s°C, %s", temp, desc))
						}
					}
				}
			}
			// 뉴스
			if tKey != "" {
				nq := "오늘 주요 뉴스"
				if eng {
					nq = "today's top news"
				}
				if nr, ok := tavilySearch(tKey, nq, 3); ok && nr.Summary != "" {
					if eng {
						sections = append(sections, "📰 News: "+nr.Summary)
					} else {
						sections = append(sections, "📰 뉴스: "+nr.Summary)
					}
				}
			}
			var msg string
			if len(sections) > 0 {
				msg = strings.Join(sections, "\n\n")
			} else {
				if eng {
					msg = "Good morning! Have a great day."
				} else {
					msg = "좋은 아침이에요! 오늘도 좋은 하루 되세요."
				}
			}
			publishAlert(Alert{
				ID:      fmt.Sprintf("briefing_%d", time.Now().Unix()),
				Level:   "info",
				Title:   func() string { if eng { return "Morning Briefing" }; return "모닝 브리핑" }(),
				Message: msg,
			})
		}
	}()
}

func handleBriefingNow(w http.ResponseWriter, r *http.Request) {
	// ?lang=en 명시 > 저장된 사용자 설정 순서로 언어 결정
	lang := r.URL.Query().Get("lang")
	var eng bool
	if lang == "en" || lang == "ko" {
		eng = lang == "en"
	} else {
		eng = IsUserEng()
	}
	now := time.Now()
	hour := now.Hour()
	var greeting string
	if eng {
		switch {
		case hour < 12:
			greeting = "Good morning"
		case hour < 18:
			greeting = "Good afternoon"
		default:
			greeting = "Good evening"
		}
	} else {
		switch {
		case hour < 12:
			greeting = "좋은 아침이에요"
		case hour < 18:
			greeting = "좋은 오후예요"
		default:
			greeting = "좋은 저녁이에요"
		}
	}

	client := &http.Client{Timeout: 6 * time.Second}

	// 1. 날씨
	weatherLang := "ko"
	if eng { weatherLang = "en" }
	weatherInfo := ""
	if resp, err := client.Get("http://127.0.0.1:17891/api/weather?city=Seoul&lang=" + weatherLang); err == nil {
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
		newsQuery := "오늘 주요 뉴스 한국"
		if eng { newsQuery = "today's top news worldwide " + time.Now().Format("2006") }
		tr, ok := tavilySearch(tKey, newsQuery, 5)
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
	if eng {
		sections = append(sections, fmt.Sprintf("Current time: %s", now.Format("2006-01-02 15:04 (Monday)")))
		if weatherInfo != "" { sections = append(sections, "Weather: "+weatherInfo) }
		if calendarInfo != "" { sections = append(sections, "Today's schedule: "+calendarInfo) }
		if newsInfo != "" { sections = append(sections, "Top news:\n"+newsInfo) }
		if statsInfo != "" { sections = append(sections, "PC status: "+statsInfo) }
		if stockInfo != "" { sections = append(sections, "Market: "+stockInfo) }
	} else {
		sections = append(sections, fmt.Sprintf("현재 시각: %s", now.Format("2006-01-02 15:04 (Monday)")))
		if weatherInfo != "" { sections = append(sections, "날씨: "+weatherInfo) }
		if calendarInfo != "" { sections = append(sections, "오늘 일정: "+calendarInfo) }
		if newsInfo != "" { sections = append(sections, "주요 뉴스:\n"+newsInfo) }
		if statsInfo != "" { sections = append(sections, "PC 상태: "+statsInfo) }
		if stockInfo != "" { sections = append(sections, "주식: "+stockInfo) }
	}

	var prompt string
	if eng {
		prompt = fmt.Sprintf(`You are Nexus AI assistant. Greet the user with "%s" and deliver a morning briefing.

Summarize the following data kindly in English:
%s

Format:
- Greeting + date
- Weather in one line
- Today's schedule (if any)
- 2-3 top news highlights
- PC status in one line
- Stock market update (if any)
- Thought for the day

Keep it under 300 words.`, greeting, strings.Join(sections, "\n\n"))
	} else {
		prompt = fmt.Sprintf(`당신은 Nexus AI 비서입니다. 사용자에게 %s 인사와 함께 아침 브리핑을 해줘.

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
	}

	briefingText, _, _ := callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 500, false)
	if strings.TrimSpace(briefingText) == "" {
		if eng {
			briefingText = fmt.Sprintf("%s! Today is %s.", greeting, now.Format("January 2"))
		} else {
			briefingText = fmt.Sprintf("%s! 오늘은 %s입니다.", greeting, now.Format("1월 2일"))
		}
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
