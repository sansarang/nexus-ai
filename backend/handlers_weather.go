//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"time"
)

// ══════════════════════════════════════════════════════════════════
//  Weather + Travel Time — 무료 API (키 불필요)
// ══════════════════════════════════════════════════════════════════

// GET /api/weather?city=서울&lang=en
func handleWeather(w http.ResponseWriter, r *http.Request) {
	city := r.URL.Query().Get("city")
	lang := r.URL.Query().Get("lang")
	if city == "" {
		city = "Seoul"
	}
	eng := lang == "en" || (lang == "" && isEnglishQuery(city))

	apiURL := "https://wttr.in/" + url.PathEscape(city) + "?format=j1"
	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("User-Agent", "NexusAssistant/1.0")

	resp, err := client.Do(req)
	if err != nil {
		msg := "날씨 데이터 가져오기 실패: " + err.Error()
		if eng { msg = "Failed to fetch weather data: " + err.Error() }
		json200(w, map[string]interface{}{"success": false, "message": msg})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var wttr map[string]interface{}
	if err := json.Unmarshal(body, &wttr); err != nil {
		msg := "날씨 데이터 파싱 실패"
		if eng { msg = "Failed to parse weather data" }
		json200(w, map[string]interface{}{"success": false, "message": msg})
		return
	}

	// current_condition
	var tempC, feelsLike, humidity, windKmh float64
	condition := ""

	if cc, ok := wttr["current_condition"].([]interface{}); ok && len(cc) > 0 {
		cur := cc[0].(map[string]interface{})
		tempC = parseFloatField(cur, "temp_C")
		feelsLike = parseFloatField(cur, "FeelsLikeC")
		humidity = parseFloatField(cur, "humidity")
		windKmh = parseFloatField(cur, "windspeedKmph")
		if desc, ok := cur["weatherDesc"].([]interface{}); ok && len(desc) > 0 {
			if d, ok := desc[0].(map[string]interface{}); ok {
				condition, _ = d["value"].(string)
			}
		}
	}

	type ForecastDay struct {
		Date      string  `json:"date"`
		Max       float64 `json:"max"`
		Min       float64 `json:"min"`
		Condition string  `json:"condition"`
	}

	var forecast []ForecastDay
	if weather, ok := wttr["weather"].([]interface{}); ok {
		for i, day := range weather {
			if i >= 3 {
				break
			}
			d := day.(map[string]interface{})
			date, _ := d["date"].(string)
			maxC := parseFloatField(d, "maxtempC")
			minC := parseFloatField(d, "mintempC")
			cond := ""
			if hourly, ok := d["hourly"].([]interface{}); ok && len(hourly) > 0 {
				h := hourly[len(hourly)/2].(map[string]interface{})
				if desc, ok := h["weatherDesc"].([]interface{}); ok && len(desc) > 0 {
					if dd, ok := desc[0].(map[string]interface{}); ok {
						cond, _ = dd["value"].(string)
					}
				}
			}
			forecast = append(forecast, ForecastDay{Date: date, Max: maxC, Min: minC, Condition: cond})
		}
	}

	var weatherMsg string
	if eng {
		weatherMsg = fmt.Sprintf("%s: %.0f°C, %s", city, tempC, condition)
	} else {
		weatherMsg = fmt.Sprintf("%s 현재 %.0f°C, %s", city, tempC, condition)
	}
	json200(w, map[string]interface{}{
		"success":    true,
		"city":       city,
		"temp_c":     tempC,
		"feels_like": feelsLike,
		"condition":  condition,
		"humidity":   humidity,
		"wind_kmh":   windKmh,
		"forecast":   forecast,
		"message":    weatherMsg,
	})
}

func parseFloatField(m map[string]interface{}, key string) float64 {
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

// POST /api/travel/time — body: {origin, destination, departure_time?}
func handleTravelTime(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Origin        string `json:"origin"`
		Destination   string `json:"destination"`
		DepartureTime string `json:"departure_time,omitempty"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	tEng := isEnglishQuery(req.Origin + " " + req.Destination)
	if req.Origin == "" || req.Destination == "" {
		msg := "origin과 destination이 필요해요"
		if tEng { msg = "origin and destination are required" }
		json200(w, map[string]interface{}{"success": false, "message": msg})
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}

	// Geocoding
	originCoords, err := geocode(client, req.Origin)
	if err != nil {
		msg := "출발지 좌표 검색 실패: " + err.Error()
		if tEng { msg = "Failed to locate origin: " + err.Error() }
		json200(w, map[string]interface{}{"success": false, "message": msg})
		return
	}
	destCoords, err := geocode(client, req.Destination)
	if err != nil {
		msg := "목적지 좌표 검색 실패: " + err.Error()
		if tEng { msg = "Failed to locate destination: " + err.Error() }
		json200(w, map[string]interface{}{"success": false, "message": msg})
		return
	}

	// OSRM 경로 계산
	osrmURL := fmt.Sprintf(
		"https://router.project-osrm.org/route/v1/driving/%f,%f;%f,%f?overview=false",
		originCoords[0], originCoords[1], destCoords[0], destCoords[1],
	)

	osrmReq, _ := http.NewRequest("GET", osrmURL, nil)
	osrmReq.Header.Set("User-Agent", "NexusAssistant/1.0")
	osrmResp, err := client.Do(osrmReq)
	if err != nil {
		msg := "경로 계산 실패: " + err.Error()
		if tEng { msg = "Route calculation failed: " + err.Error() }
		json200(w, map[string]interface{}{"success": false, "message": msg})
		return
	}
	defer osrmResp.Body.Close()

	var osrmData struct {
		Routes []struct {
			Distance float64 `json:"distance"` // meters
			Duration float64 `json:"duration"` // seconds
		} `json:"routes"`
	}
	body, _ := io.ReadAll(osrmResp.Body)
	json.Unmarshal(body, &osrmData)

	if len(osrmData.Routes) == 0 {
		msg := "경로를 찾을 수 없어요"
		if tEng { msg = "No route found" }
		json200(w, map[string]interface{}{"success": false, "message": msg})
		return
	}

	distKm := math.Round(osrmData.Routes[0].Distance/100) / 10
	durMin := int(osrmData.Routes[0].Duration / 60)

	departureTime := req.DepartureTime
	if departureTime == "" {
		departureTime = time.Now().Format("15:04")
	}

	// 도착 시간 계산
	arrivalTime := ""
	t, err := time.Parse("15:04", departureTime)
	if err == nil {
		arrival := t.Add(time.Duration(durMin) * time.Minute)
		arrivalTime = arrival.Format("15:04")
	}

	var travelMsg string
	if tEng {
		travelMsg = fmt.Sprintf("%s → %s: %.1f km, ~%d min, ETA %s", req.Origin, req.Destination, distKm, durMin, arrivalTime)
	} else {
		travelMsg = fmt.Sprintf("%s → %s: %.1fkm, 약 %d분 소요, 도착 예상 %s", req.Origin, req.Destination, distKm, durMin, arrivalTime)
	}
	json200(w, map[string]interface{}{
		"success":        true,
		"origin":         req.Origin,
		"destination":    req.Destination,
		"distance_km":    distKm,
		"duration_min":   durMin,
		"departure_time": departureTime,
		"arrival_time":   arrivalTime,
		"message":        travelMsg,
	})
}

type Coords [2]float64 // [lon, lat]

func geocode(client *http.Client, place string) (Coords, error) {
	apiURL := "https://nominatim.openstreetmap.org/search?q=" + url.QueryEscape(place) + "&format=json&limit=1"
	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("User-Agent", "NexusAssistant/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return Coords{}, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var results []struct {
		Lat string `json:"lat"`
		Lon string `json:"lon"`
	}
	if err := json.Unmarshal(body, &results); err != nil || len(results) == 0 {
		return Coords{}, fmt.Errorf("'%s' 위치를 찾을 수 없어요", place)
	}

	var lon, lat float64
	fmt.Sscanf(results[0].Lon, "%f", &lon)
	fmt.Sscanf(results[0].Lat, "%f", &lat)
	return Coords{lon, lat}, nil
}
