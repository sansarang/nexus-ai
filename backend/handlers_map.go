package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// ──────────────────────────────────────────────────────────────
// POST /api/directions  — 출발지→도착지 지도 링크 + 경로 정보
// ──────────────────────────────────────────────────────────────
func handleDirections(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query string `json:"query"`
		From  string `json:"from"`  // 직접 지정 시
		To    string `json:"to"`
		Mode  string `json:"mode"` // transit|car|walk
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Query == "" && req.From == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "query required / query 필요"})
		return
	}
	if req.Mode == "" {
		req.Mode = "transit"
	}

	// 출발지/도착지 추출 (LLM 또는 직접)
	from, to := req.From, req.To
	if (from == "" || to == "") && req.Query != "" {
		from, to = extractFromTo(req.Query)
	}
	if to == "" {
		to = req.Query
	}

	llmMu.RLock()
	tKey := llmTavilyKey
	gKey := llmPerplexityKey
	llmMu.RUnlock()

	// 지도 앱 링크 생성
	links := buildMapLinks(from, to, req.Mode, isEnglishQuery(req.Query+" "+to+" "+from))

	var (
		travelSummary string
		placeImages   []string
	)

	eng := isEnglishQuery(req.Query + " " + to + " " + from)

	if tKey != "" && from != "" && to != "" {
		var wg sync.WaitGroup

		// ── 이동 시간 검색 병렬 ──────────────────────────────────
		wg.Add(1)
		go func() {
			defer wg.Done()
			var timeQuery string
			if eng {
				timeQuery = fmt.Sprintf("travel time from %s to %s by train bus car duration", from, to)
			} else {
				timeQuery = fmt.Sprintf("%s에서 %s KTX 버스 자동차 소요시간 얼마나 걸려", from, to)
			}
			if tr, ok := tavilySearch(tKey, timeQuery, 5); ok && gKey != "" {
				titles := []string{}
				for i, it := range tr.Items {
					if i >= 5 { break }
					if t := it["title"]; t != "" { titles = append(titles, "• "+t) }
				}
				hint := tr.Summary
				if hint == "" { hint = strings.Join(titles, "\n") }
				var transSys, transUser string
				if eng {
					transSys = "You are a transit information expert. Summarize travel time and cost by train, bus, and car in 3-4 natural English sentences. No URLs. If no specific times found, end with: 'Please check the official app for accurate real-time schedules.'"
					transUser = fmt.Sprintf("From: %s\nTo: %s\nSearch results:\n%s", from, to, hint)
				} else {
					transSys = "당신은 교통 정보 전문가입니다. 아래 검색 결과를 바탕으로 KTX/고속버스/자가용 소요시간과 요금을 자연스러운 한국어 3~4문장으로 요약하세요. URL/링크 금지. 숫자가 없으면 '정확한 시간은 아래 앱에서 확인하세요'로 마무리."
					transUser = fmt.Sprintf("출발: %s\n도착: %s\n검색 결과:\n%s", from, to, hint)
				}
				msgs := []groqMsg{
					{Role: "system", Content: transSys},
					{Role: "user", Content: transUser},
				}
				ans, _, _ := callGroq(gKey, groqChatModel, msgs, 300, false)
				travelSummary = ans
			}
		}()

		// ── 목적지 사진 검색 ──────────────────────────────────────
		wg.Add(1)
		go func() {
			defer wg.Done()
			var imgQuery string
			if eng {
				imgQuery = to + " tourist attractions travel photos"
			} else {
				imgQuery = to + " 여행 관광지 사진"
			}
			if imgs, ok := tavilySearchImages(tKey, imgQuery, 6); ok {
				placeImages = imgs
			}
		}()

		wg.Wait()
	}

	if travelSummary == "" {
		travelSummary = buildDirectionsSummary(from, to, req.Mode)
	}

	writeJSON(w, 200, map[string]any{
		"success":        true,
		"from":           from,
		"to":             to,
		"mode":           req.Mode,
		"map_links":      links,
		"travel_summary": travelSummary,
		"place_images":   placeImages,
		"summary":        travelSummary,
	})
}

// ──────────────────────────────────────────────────────────────
// POST /api/place-view  — 특정 장소 로드뷰 + 지도 링크
// ──────────────────────────────────────────────────────────────
func handlePlaceView(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query string `json:"query"`
		Place string `json:"place"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	place := req.Place
	if place == "" {
		place = extractPlaceName(req.Query)
	}
	if place == "" {
		place = req.Query
	}

	enc := url.QueryEscape(place)
	engPlace := isEnglishQuery(req.Query + " " + place)

	var links []map[string]string
	if engPlace {
		links = []map[string]string{
			{"title": "🌐 Google Street View — " + place, "url": fmt.Sprintf("https://www.google.com/maps/search/%s", enc), "type": "roadview", "service": "google"},
			{"title": "📍 Google Maps — " + place, "url": fmt.Sprintf("https://www.google.com/maps/place/%s", enc), "type": "map", "service": "google"},
			{"title": "🗺️ Apple Maps — " + place, "url": fmt.Sprintf("https://maps.apple.com/?q=%s", enc), "type": "map", "service": "apple"},
			{"title": "📌 Bing Maps — " + place, "url": fmt.Sprintf("https://www.bing.com/maps?q=%s", enc), "type": "map", "service": "bing"},
		}
	} else {
		links = []map[string]string{
			{"title": "🗺️ 네이버 지도 로드뷰", "url": fmt.Sprintf("https://map.naver.com/v5/search/%s", enc), "type": "roadview", "service": "naver"},
			{"title": "🗺️ 카카오맵 로드뷰", "url": fmt.Sprintf("https://map.kakao.com/?q=%s&map_type=roadview", enc), "type": "roadview", "service": "kakao"},
			{"title": "🌐 구글 스트리트뷰", "url": fmt.Sprintf("https://www.google.com/maps/search/%s", enc), "type": "roadview", "service": "google"},
			{"title": "📍 카카오맵 위치", "url": fmt.Sprintf("https://map.kakao.com/?q=%s", enc), "type": "map", "service": "kakao"},
		}
	}

	// Tavily로 장소 정보 검색
	llmMu.RLock()
	tKey := llmTavilyKey
	llmMu.RUnlock()

	var placeInfo []map[string]string
	if tKey != "" {
		var infoQuery string
		if engPlace {
			infoQuery = place + " location address info"
		} else {
			infoQuery = place + " 위치 주소 정보"
		}
		if tr, ok := tavilySearch(tKey, infoQuery, 4); ok {
			for _, it := range tr.Items {
				if it["url"] != "" && it["title"] != "" {
					placeInfo = append(placeInfo, map[string]string{"title": it["title"], "url": it["url"]})
				}
			}
		}
	}

	writeJSON(w, 200, map[string]any{
		"success":    true,
		"place":      place,
		"map_links":  links,
		"place_info": placeInfo,
	})
}

// ── 헬퍼 ──────────────────────────────────────────────────────

// extractFromTo: 쿼리에서 출발지/도착지 추출 (한/영)
func extractFromTo(query string) (from, to string) {
	q := strings.TrimSpace(query)
	lower := strings.ToLower(q)

	// English: "from X to Y" pattern
	if idx := strings.Index(lower, " to "); idx > 0 {
		fromIdx := strings.Index(lower, "from ")
		if fromIdx >= 0 {
			left := strings.TrimSpace(q[fromIdx+5 : idx])
			right := strings.TrimSpace(q[idx+4:])
			// strip trailing direction words
			for _, s := range []string{"directions", "route", "how to get", "by bus", "by train", "by subway", "transit"} {
				right = strings.TrimSuffix(strings.ToLower(right), s)
			}
			right = strings.TrimSpace(right)
			if left != "" && right != "" {
				return left, right
			}
		}
	}

	// Korean/Symbol separators
	for _, sep := range []string{"에서", "→", "->", "~", "까지"} {
		if idx := strings.Index(q, sep); idx > 0 {
			left := strings.TrimSpace(q[:idx])
			right := strings.TrimSpace(q[idx+len(sep):])
			for _, s := range []string{"가는방법", "가는법", "가는 방법", "가는길", "가는 길", "가는", "경로", "어떻게가", "어떻게 가", "가려면", "길", "방법"} {
				right = strings.TrimSuffix(right, s)
			}
			right = strings.TrimSpace(right)
			if left != "" && right != "" {
				return left, right
			}
		}
	}

	// Remove direction suffix words and split remainder
	suffixes := []string{
		"가는 방법", "가는방법", "가는길", "경로", "어떻게 가", "어떻게가", "대중교통", "버스", "지하철", "길찾기",
		"directions", "how to get there", "how to go", "route", "transit",
	}
	cleaned := q
	for _, s := range suffixes {
		cleaned = strings.ReplaceAll(strings.ToLower(cleaned), s, " ")
	}
	cleaned = strings.TrimSpace(cleaned)

	parts := strings.Fields(cleaned)
	if len(parts) >= 2 {
		mid := len(parts) / 2
		from = strings.Join(parts[:mid], " ")
		to = strings.Join(parts[mid:], " ")
	} else if len(parts) == 1 {
		to = parts[0]
	}
	return
}

// extractPlaceName: 쿼리에서 장소명 추출 (한/영)
func extractPlaceName(query string) string {
	q := query
	removes := []string{
		// 한국어
		"위치 알려줘", "위치알려줘", "어디야", "어디에 있어", "어디 있어", "어디에있어",
		"주소 알려줘", "주소알려줘", "로드뷰", "지도", "찾아줘", "알려줘", "보여줘",
		"어디", "위치", "주소",
		// 영어
		"where is", "where's", "location of", "address of", "street view of",
		"show me", "find", "locate", "map of", "directions to",
	}
	for _, r := range removes {
		q = strings.ReplaceAll(strings.ToLower(q), strings.ToLower(r), "")
	}
	return strings.TrimSpace(q)
}

// buildMapLinks: 출발지/도착지로 교통수단별 전체 지도 링크 생성 (한/영 분기)
func buildMapLinks(from, to, mode string, eng bool) []map[string]string {
	encFrom := url.QueryEscape(from)
	encTo   := url.QueryEscape(to)

	googleModes := map[string]string{
		"transit": "r", "car": "d", "walk": "w", "bicycle": "b", "ktx": "r",
	}
	kakaoModes := map[string]string{
		"transit": "0", "car": "1", "walk": "2", "bicycle": "3",
	}

	type modeInfo struct {
		id    string
		emoji string
		label string
	}

	var modes []modeInfo
	if eng {
		modes = []modeInfo{
			{"transit", "🚌", "Public Transit"},
			{"car", "🚗", "Drive"},
			{"walk", "🚶", "Walk"},
			{"bicycle", "🚲", "Bicycle"},
		}
	} else {
		modes = []modeInfo{
			{"transit", "🚌", "대중교통"},
			{"car", "🚗", "자동차"},
			{"walk", "🚶", "도보"},
			{"bicycle", "🚲", "자전거"},
			{"ktx", "🚂", "기차/KTX"},
		}
	}

	var links []map[string]string
	for _, m := range modes {
		gCode := googleModes[m.id]
		if gCode == "" { gCode = "r" }
		links = append(links, map[string]string{
			"title":     fmt.Sprintf("%s %s — %s → %s", m.emoji, m.label, from, to),
			"url":       fmt.Sprintf("https://www.google.com/maps/dir/%s/%s/data=!4m2!4m1!3e%s", encFrom, encTo, gCode),
			"type":      "directions",
			"service":   "google",
			"mode":      m.id,
			"modeKo":    m.label,
			"modeEmoji": m.emoji,
		})

		if !eng {
			// 한국 사용자: 카카오맵 + 네이버 지도 추가
			if kCode, ok := kakaoModes[m.id]; ok {
				links = append(links, map[string]string{
					"title":     fmt.Sprintf("%s %s — %s→%s (카카오)", m.emoji, m.label, from, to),
					"url":       fmt.Sprintf("https://map.kakao.com/?sName=%s&eName=%s&pathType=%s", encFrom, encTo, kCode),
					"type":      "directions",
					"service":   "kakao",
					"mode":      m.id,
					"modeKo":    m.label,
					"modeEmoji": m.emoji,
				})
			}
			links = append(links, map[string]string{
				"title":     fmt.Sprintf("%s %s — %s→%s (네이버)", m.emoji, m.label, from, to),
				"url":       fmt.Sprintf("https://map.naver.com/v5/directions/-/-/%s/%s/-/transit", encFrom, encTo),
				"type":      "directions",
				"service":   "naver",
				"mode":      m.id,
				"modeKo":    m.label,
				"modeEmoji": m.emoji,
			})
		} else {
			// 영어권: Apple Maps + Waze 추가
			if m.id == "transit" || m.id == "car" || m.id == "walk" {
				wazeMode := map[string]string{"transit": "0", "car": "0", "walk": "9"}[m.id]
				links = append(links, map[string]string{
					"title":     fmt.Sprintf("%s %s — %s → %s (Waze)", m.emoji, m.label, from, to),
					"url":       fmt.Sprintf("https://waze.com/ul?q=%s&navigate=yes&zoom=17", encTo),
					"type":      "directions",
					"service":   "waze",
					"mode":      m.id,
					"modeKo":    m.label,
					"modeEmoji": m.emoji,
				})
				_ = wazeMode
			}
		}
	}

	if eng {
		// 영어권: Greyhound / Amtrak
		links = append(links,
			map[string]string{"title": "🚌 Greyhound Bus", "url": "https://www.greyhound.com/", "type": "bus", "service": "greyhound"},
			map[string]string{"title": "🚂 Amtrak Train", "url": "https://www.amtrak.com/", "type": "bus", "service": "amtrak"},
			map[string]string{"title": "✈️ Google Flights", "url": fmt.Sprintf("https://www.google.com/flights#flt=%s..%s", encFrom, encTo), "type": "flight", "service": "google"},
		)
	} else {
		// 한국: 버스타고 + 코레일
		links = append(links,
			map[string]string{"title": "🚌 버스타고 시외버스 예매", "url": "https://www.bustago.or.kr/newweb/kr/main.do", "type": "bus", "service": "bustago"},
			map[string]string{"title": "🚂 코레일 기차 예매", "url": "https://www.letskorail.com/ebizprd/prdMain.do", "type": "bus", "service": "korail"},
		)
	}

	_ = mode
	return links
}

func buildDirectionsSummary(from, to, mode string) string {
	eng := isEnglishQuery(from + " " + to)
	if eng {
		if from == "" { from = "origin" }
		if to == "" { to = "destination" }
		modeEn := map[string]string{"transit": "public transit", "car": "driving", "walk": "walking", "bicycle": "cycling"}[mode]
		if modeEn == "" { modeEn = "public transit" }
		return fmt.Sprintf("**%s → %s** — Check %s directions in the map app. Click a button below for real-time schedules and transfer information.", from, to, modeEn)
	}
	if from == "" { from = "출발지" }
	if to == "" { to = "도착지" }
	modeKo := map[string]string{"transit": "대중교통", "car": "자동차", "walk": "도보", "bicycle": "자전거"}[mode]
	if modeKo == "" { modeKo = "대중교통" }
	return fmt.Sprintf("**%s → %s** %s 경로를 지도 앱에서 확인하세요. 아래 버튼을 클릭하면 실시간 버스/지하철 시간표와 환승 정보를 볼 수 있어요.", from, to, modeKo)
}
