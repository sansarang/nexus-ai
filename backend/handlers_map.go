package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
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
	llmMu.RUnlock()

	// 지도 앱 링크 생성
	links := buildMapLinks(from, to, req.Mode)

	// Tavily로 실제 버스 노선 검색
	var routeInfo []map[string]string
	if tKey != "" && from != "" && to != "" {
		busQuery := fmt.Sprintf("%s %s 버스 노선 대중교통 경로", from, to)
		if tr, ok := tavilySearch(tKey, busQuery, 6); ok {
			for _, it := range tr.Items {
				u := it["url"]
				t := it["title"]
				if u != "" && t != "" &&
					!strings.Contains(strings.ToLower(u), "tiktok") &&
					!strings.Contains(strings.ToLower(u), "instagram") {
					routeInfo = append(routeInfo, map[string]string{"title": t, "url": u})
				}
			}
		}
	}

	// LLM으로 경로 요약
	summary := buildDirectionsSummary(from, to, req.Mode)

	writeJSON(w, 200, map[string]any{
		"success":    true,
		"from":       from,
		"to":         to,
		"mode":       req.Mode,
		"map_links":  links,
		"route_info": routeInfo,
		"summary":    summary,
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

// extractFromTo: 쿼리에서 출발지/도착지 추출
func extractFromTo(query string) (from, to string) {
	q := strings.TrimSpace(query)

	// "A에서 B" 패턴
	for _, sep := range []string{"에서", "→", "->", "to ", "~", "까지"} {
		if idx := strings.Index(q, sep); idx > 0 {
			left := strings.TrimSpace(q[:idx])
			right := strings.TrimSpace(q[idx+len(sep):])
			// 불필요한 suffix 제거
			for _, s := range []string{"가는방법", "가는법", "경로", "어떻게가", "어떻게 가", "가려면", "길", "방법"} {
				right = strings.TrimSuffix(right, s)
			}
			right = strings.TrimSpace(right)
			if left != "" && right != "" {
				return left, right
			}
		}
	}

	// "A → B 경로/방법" 패턴
	suffixes := []string{"가는 방법", "가는방법", "가는길", "경로", "어떻게 가", "어떻게가", "대중교통", "버스", "지하철", "길찾기"}
	cleaned := q
	for _, s := range suffixes {
		cleaned = strings.ReplaceAll(cleaned, s, " ")
	}
	cleaned = strings.TrimSpace(cleaned)

	// 키워드로 분리 시도
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

// extractPlaceName: 쿼리에서 장소명 추출
func extractPlaceName(query string) string {
	q := query
	removes := []string{
		"위치 알려줘", "위치알려줘", "어디야", "어디에 있어", "어디 있어", "어디에있어",
		"주소 알려줘", "주소알려줘", "로드뷰", "지도", "찾아줘", "알려줘", "보여줘",
		"어디", "위치", "주소",
	}
	for _, r := range removes {
		q = strings.ReplaceAll(q, r, "")
	}
	return strings.TrimSpace(q)
}

// buildMapLinks: 출발지/도착지로 지도 앱 링크 생성
func buildMapLinks(from, to, mode string) []map[string]string {
	encFrom := url.QueryEscape(from)
	encTo := url.QueryEscape(to)
	encQuery := url.QueryEscape(from + " " + to + " 대중교통 경로")

	modeKo := map[string]string{"transit": "대중교통", "car": "자동차", "walk": "도보"}[mode]
	if modeKo == "" {
		modeKo = "대중교통"
	}

	naverMode := map[string]string{"transit": "transit", "car": "car", "walk": "walk"}[mode]
	if naverMode == "" {
		naverMode = "transit"
	}

	links := []map[string]string{
		{
			"title":   fmt.Sprintf("🗺️ 네이버 지도 — %s→%s (%s)", from, to, modeKo),
			"url":     fmt.Sprintf("https://map.naver.com/v5/directions/-/%s/-/%s/-/%s", encFrom, encTo, naverMode),
			"type":    "directions",
			"service": "naver",
		},
		{
			"title":   fmt.Sprintf("🗺️ 카카오맵 — %s→%s", from, to),
			"url":     fmt.Sprintf("https://map.kakao.com/?sName=%s&eName=%s", encFrom, encTo),
			"type":    "directions",
			"service": "kakao",
		},
		{
			"title":   fmt.Sprintf("🌐 구글 지도 — %s→%s", from, to),
			"url":     fmt.Sprintf("https://www.google.com/maps/dir/%s/%s/", encFrom, encTo),
			"type":    "directions",
			"service": "google",
		},
	}

	// 버스 전용 링크
	if mode == "transit" || mode == "" {
		links = append(links, map[string]string{
			"title":   fmt.Sprintf("🚌 버스타고 시외버스 — %s", encQuery),
			"url":     fmt.Sprintf("https://www.bustago.or.kr/newweb/kr/main.do"),
			"type":    "bus",
			"service": "bustago",
		})
	}

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
