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
		writeJSON(w, 400, map[string]any{"success": false, "message": "query 필요"})
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
	links := buildMapLinks(from, to, req.Mode)

	var (
		travelSummary string
		placeImages   []string
	)

	if tKey != "" && from != "" && to != "" {
		var wg sync.WaitGroup

		// ── 이동 시간 검색 (KTX/버스/자가 병렬) ──────────────────
		wg.Add(1)
		go func() {
			defer wg.Done()
			timeQuery := fmt.Sprintf("%s에서 %s KTX 버스 자동차 소요시간 얼마나 걸려", from, to)
			if tr, ok := tavilySearch(tKey, timeQuery, 5); ok && gKey != "" {
				titles := []string{}
				for i, it := range tr.Items {
					if i >= 5 { break }
					if t := it["title"]; t != "" { titles = append(titles, "• "+t) }
				}
				hint := tr.Summary
				if hint == "" { hint = strings.Join(titles, "\n") }
				msgs := []groqMsg{
					{Role: "system", Content: "당신은 교통 정보 전문가입니다. 아래 검색 결과를 바탕으로 KTX/고속버스/자가용 소요시간과 요금을 자연스러운 한국어 3~4문장으로 요약하세요. URL/링크 금지. 숫자가 없으면 '정확한 시간은 아래 앱에서 확인하세요'로 마무리."},
					{Role: "user", Content: fmt.Sprintf("출발: %s\n도착: %s\n검색 결과:\n%s", from, to, hint)},
				}
				ans, _, _ := callGroq(gKey, groqChatModel, msgs, 300, false)
				travelSummary = ans
			}
		}()

		// ── 목적지 사진 검색 ──────────────────────────────────────
		wg.Add(1)
		go func() {
			defer wg.Done()
			imgQuery := to + " 여행 관광지 사진"
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

	links := []map[string]string{
		{
			"title":    "🗺️ 네이버 지도 로드뷰",
			"url":      fmt.Sprintf("https://map.naver.com/v5/search/%s", enc),
			"type":     "roadview",
			"service":  "naver",
		},
		{
			"title":    "🗺️ 카카오맵 로드뷰",
			"url":      fmt.Sprintf("https://map.kakao.com/?q=%s&map_type=roadview", enc),
			"type":     "roadview",
			"service":  "kakao",
		},
		{
			"title":    "🌐 구글 스트리트뷰",
			"url":      fmt.Sprintf("https://www.google.com/maps/search/%s", enc),
			"type":     "roadview",
			"service":  "google",
		},
		{
			"title":    "📍 카카오맵 위치",
			"url":      fmt.Sprintf("https://map.kakao.com/?q=%s", enc),
			"type":     "map",
			"service":  "kakao",
		},
	}

	// Tavily로 장소 정보 검색
	llmMu.RLock()
	tKey := llmTavilyKey
	llmMu.RUnlock()

	var placeInfo []map[string]string
	if tKey != "" {
		infoQuery := place + " 위치 주소 정보"
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

// buildMapLinks: 출발지/도착지로 교통수단별 전체 지도 링크 생성
func buildMapLinks(from, to, mode string) []map[string]string {
	encFrom := url.QueryEscape(from)
	encTo   := url.QueryEscape(to)

	// Google Maps 교통수단 코드 (텍스트 기반 경로 표시 — 좌표 불필요)
	googleModes := map[string]string{
		"transit": "r", "car": "d", "walk": "w", "bicycle": "b", "ktx": "r",
	}

	// 카카오맵 교통수단 코드 (숫자: 0=대중교통 1=자동차 2=도보 3=자전거)
	kakaoModes := map[string]string{
		"transit": "0", "car": "1", "walk": "2", "bicycle": "3",
	}

	type modeInfo struct {
		id    string
		emoji string
		ko    string
	}
	modes := []modeInfo{
		{"transit", "🚌", "대중교통"},
		{"car", "🚗", "자동차"},
		{"walk", "🚶", "도보"},
		{"bicycle", "🚲", "자전거"},
		{"ktx", "🚂", "기차/KTX"},
	}

	var links []map[string]string
	for _, m := range modes {
		// ① Google Maps — 텍스트만으로 실제 경로 표시
		gCode := googleModes[m.id]
		if gCode == "" { gCode = "r" }
		links = append(links, map[string]string{
			"title":     fmt.Sprintf("%s %s — %s→%s", m.emoji, m.ko, from, to),
			"url":       fmt.Sprintf("https://www.google.com/maps/dir/%s/%s/data=!4m2!4m1!3e%s", encFrom, encTo, gCode),
			"type":      "directions",
			"service":   "google",
			"mode":      m.id,
			"modeKo":    m.ko,
			"modeEmoji": m.emoji,
		})
		// ② 카카오맵 (기차 제외) — 텍스트 기반 경로 표시
		if kCode, ok := kakaoModes[m.id]; ok {
			links = append(links, map[string]string{
				"title":     fmt.Sprintf("%s %s — %s→%s (카카오)", m.emoji, m.ko, from, to),
				"url":       fmt.Sprintf("https://map.kakao.com/?sName=%s&eName=%s&pathType=%s", encFrom, encTo, kCode),
				"type":      "directions",
				"service":   "kakao",
				"mode":      m.id,
				"modeKo":    m.ko,
				"modeEmoji": m.emoji,
			})
		}
		// ③ 네이버 지도 웹 경로 (브라우저에서 바로 열림)
		links = append(links, map[string]string{
			"title":     fmt.Sprintf("%s %s — %s→%s (네이버)", m.emoji, m.ko, from, to),
			"url":       fmt.Sprintf("https://map.naver.com/v5/directions/-/-/%s/%s/-/transit", encFrom, encTo),
			"type":      "directions",
			"service":   "naver",
			"mode":      m.id,
			"modeKo":    m.ko,
			"modeEmoji": m.emoji,
		})
	}

	// 시외버스·기차 예매 링크
	links = append(links,
		map[string]string{
			"title": "🚌 버스타고 시외버스 예매", "url": "https://www.bustago.or.kr/newweb/kr/main.do",
			"type": "bus", "service": "bustago",
		},
		map[string]string{
			"title": "🚂 코레일 기차 예매", "url": "https://www.letskorail.com/ebizprd/prdMain.do",
			"type": "bus", "service": "korail",
		},
	)

	_ = mode
	return links
}

func buildDirectionsSummary(from, to, mode string) string {
	if from == "" {
		from = "출발지"
	}
	if to == "" {
		to = "도착지"
	}
	modeKo := map[string]string{"transit": "대중교통", "car": "자동차", "walk": "도보"}[mode]
	if modeKo == "" {
		modeKo = "대중교통"
	}
	return fmt.Sprintf("**%s → %s** %s 경로를 지도 앱에서 확인하세요. 아래 버튼을 클릭하면 실시간 버스/지하철 시간표와 환승 정보를 볼 수 있어요.", from, to, modeKo)
}
