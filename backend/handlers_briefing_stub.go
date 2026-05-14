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

	// 날씨 정보 가져오기
	weatherInfo := "날씨 정보를 가져오는 중..."
	wResp, err := (&http.Client{Timeout: 5 * time.Second}).Get("http://127.0.0.1:17891/api/weather?city=Seoul")
	if err == nil {
		defer wResp.Body.Close()
		body, _ := io.ReadAll(wResp.Body)
		weatherInfo = string(body)
	}

	// 시스템 상태
	statsInfo := ""
	sResp, err := (&http.Client{Timeout: 3 * time.Second}).Get("http://127.0.0.1:17891/api/stats")
	if err == nil {
		defer sResp.Body.Close()
		body, _ := io.ReadAll(sResp.Body)
		statsInfo = string(body)
	}

	prompt := fmt.Sprintf(`당신은 Nexus AI 비서입니다. 지금은 %s입니다.
사용자에게 %s 인사와 함께 아침 브리핑을 해주세요.

현재 날씨 데이터: %s
현재 시스템 상태: %s

다음 내용을 포함해서 한국어로 친절하게 브리핑해주세요:
1. 인사 및 날짜/시간
2. 날씨 요약
3. 시스템 상태 (CPU/메모리)
4. 오늘의 팁 하나

200자 이내로 짧고 명확하게 작성해주세요.`,
		now.Format("2006-01-02 15:04"), greeting, weatherInfo, statsInfo)

	briefingText, _, _ := callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 400, false)
	if strings.TrimSpace(briefingText) == "" {
		briefingText = fmt.Sprintf("%s! 오늘은 %s입니다. 좋은 하루 되세요.", greeting, now.Format("1월 2일"))
	}

	json200(w, map[string]any{
		"success":  true,
		"briefing": briefingText,
		"greeting": greeting,
		"datetime": now.Format("2006-01-02 15:04"),
	})
}

func handleBriefingConfig(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{
		"success": true,
		"config": map[string]any{
			"enabled": true,
			"time":    "08:00",
		},
	})
}
