package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// fetchWeatherText: wttr.in 실시간 날씨 → 2문장 요약
func fetchWeatherText(city, _ string) string {
	apiURL := "https://wttr.in/" + url.PathEscape(city) + "?format=j1"
	client := &http.Client{Timeout: 8 * time.Second}
	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("User-Agent", "NexusAssistant/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return city + " 날씨 정보를 가져오지 못했습니다. 네트워크 상태를 확인해주세요."
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var wttr map[string]interface{}
	if json.Unmarshal(body, &wttr) != nil {
		return city + " 날씨 데이터를 파싱하지 못했습니다."
	}

	// 현재 기상 파싱
	tempC, feelsLike, humidity := 0.0, 0.0, 0.0
	condition := ""
	if cc, ok := wttr["current_condition"].([]interface{}); ok && len(cc) > 0 {
		cur := cc[0].(map[string]interface{})
		tempC = parseFloatSafe(cur, "temp_C")
		feelsLike = parseFloatSafe(cur, "FeelsLikeC")
		humidity = parseFloatSafe(cur, "humidity")
		if desc, ok := cur["weatherDesc"].([]interface{}); ok && len(desc) > 0 {
			if d, ok := desc[0].(map[string]interface{}); ok {
				condition, _ = d["value"].(string)
			}
		}
	}

	// 내일 예보
	maxC, minC := 0.0, 0.0
	if weather, ok := wttr["weather"].([]interface{}); ok && len(weather) > 1 {
		d := weather[1].(map[string]interface{})
		maxC = parseFloatSafe(d, "maxtempC")
		minC = parseFloatSafe(d, "mintempC")
	}

	condKo := translateCondition(condition)
	msg := fmt.Sprintf("%s 현재 기온 %.0f°C (체감 %.0f°C), %s, 습도 %.0f%%입니다.", city, tempC, feelsLike, condKo, humidity)
	if maxC != 0 || minC != 0 {
		msg += fmt.Sprintf(" 내일은 최고 %.0f°C / 최저 %.0f°C 예상됩니다.", maxC, minC)
	}
	return msg
}

func parseFloatSafe(m map[string]interface{}, key string) float64 {
	v, ok := m[key]
	if !ok {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return val
	case string:
		var f float64
		fmt.Sscanf(val, "%f", &f)
		return f
	}
	return 0
}

func translateCondition(en string) string {
	m := map[string]string{
		"Sunny": "맑음", "Clear": "맑음", "Partly cloudy": "구름 조금",
		"Cloudy": "흐림", "Overcast": "흐림", "Mist": "안개",
		"Patchy rain possible": "비 가능", "Light rain": "가벼운 비",
		"Moderate rain": "보통 비", "Heavy rain": "강한 비",
		"Light snow": "가벼운 눈", "Moderate snow": "보통 눈",
		"Heavy snow": "강한 눈", "Thundery outbreaks possible": "천둥 가능",
		"Blowing snow": "눈보라", "Freezing drizzle": "어는 이슬비",
	}
	if ko, ok := m[en]; ok {
		return ko
	}
	if en == "" {
		return "날씨 정보 없음"
	}
	return en
}
