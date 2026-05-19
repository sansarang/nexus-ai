package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ══════════════════════════════════════════════════════════════
//  Wayback Machine — 인터넷 아카이브 스냅샷 조회
// ══════════════════════════════════════════════════════════════

type WaybackSnapshot struct {
	Timestamp   string `json:"timestamp"`    // YYYYMMDDHHmmSS
	Date        string `json:"date"`         // YYYY-MM-DD
	URL         string `json:"url"`          // 원본 URL
	ArchiveURL  string `json:"archive_url"`  // web.archive.org 링크
	Status      string `json:"status_code"`
	MimeType    string `json:"mime_type"`
	Available   bool   `json:"available"`
}

// POST /api/wayback/snapshots
// body: { "url": "https://...", "from_year": 2015, "to_year": 2024, "limit": 20 }
func handleWaybackSnapshots(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL      string `json:"url"`
		FromYear int    `json:"from_year"`
		ToYear   int    `json:"to_year"`
		Limit    int    `json:"limit"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.URL == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "url 필요"})
		return
	}
	if !strings.HasPrefix(req.URL, "http") {
		req.URL = "https://" + req.URL
	}
	if req.Limit == 0 {
		req.Limit = 20
	}
	if req.ToYear == 0 {
		req.ToYear = time.Now().Year()
	}
	if req.FromYear == 0 {
		req.FromYear = req.ToYear - 5
	}

	snapshots, err := fetchWaybackCDX(req.URL, req.FromYear, req.ToYear, req.Limit)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": "Wayback API 오류: " + err.Error()})
		return
	}

	summary := buildWaybackSummary(req.URL, snapshots, req.FromYear, req.ToYear)

	writeJSON(w, 200, map[string]any{
		"success":   true,
		"url":       req.URL,
		"from_year": req.FromYear,
		"to_year":   req.ToYear,
		"total":     len(snapshots),
		"snapshots": snapshots,
		"message":   summary,
	})
}

// GET /api/wayback/available?url=...
// 해당 URL의 가장 최근 스냅샷 단건 조회
func handleWaybackAvailable(w http.ResponseWriter, r *http.Request) {
	targetURL := r.URL.Query().Get("url")
	if targetURL == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "url 쿼리 필요"})
		return
	}
	if !strings.HasPrefix(targetURL, "http") {
		targetURL = "https://" + targetURL
	}

	snap, err := fetchWaybackAvailability(targetURL)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}

	var msg string
	if snap.Available {
		msg = fmt.Sprintf("✅ **%s** 의 가장 최근 아카이브 스냅샷을 찾았습니다.\n\n📅 캡처일: **%s**\n🔗 [아카이브 보기](%s)", targetURL, snap.Date, snap.ArchiveURL)
	} else {
		msg = fmt.Sprintf("❌ **%s** 의 아카이브 스냅샷을 찾을 수 없습니다.", targetURL)
	}

	writeJSON(w, 200, map[string]any{
		"success":  true,
		"snapshot": snap,
		"message":  msg,
	})
}

// ── Wayback CDX API 호출 ──────────────────────────────────────
// CDX API: http://web.archive.org/cdx/search/cdx
func fetchWaybackCDX(targetURL string, fromYear, toYear, limit int) ([]WaybackSnapshot, error) {
	from := fmt.Sprintf("%d0101000000", fromYear)
	to   := fmt.Sprintf("%d1231235959", toYear)

	params := url.Values{}
	params.Set("url", targetURL)
	params.Set("output", "json")
	params.Set("from", from)
	params.Set("to", to)
	params.Set("limit", fmt.Sprintf("%d", limit))
	params.Set("fl", "timestamp,statuscode,mimetype,original")
	params.Set("filter", "statuscode:200")
	params.Set("collapse", "timestamp:8") // 날짜별 1개

	apiURL := "http://web.archive.org/cdx/search/cdx?" + params.Encode()

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var rows [][]string
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("CDX 파싱 실패")
	}

	var snapshots []WaybackSnapshot
	for i, row := range rows {
		if i == 0 && len(row) > 0 && row[0] == "timestamp" {
			continue // 헤더 건너뜀
		}
		if len(row) < 4 {
			continue
		}
		ts := row[0]
		date := formatWaybackDate(ts)
		archiveURL := fmt.Sprintf("https://web.archive.org/web/%s/%s", ts, row[3])

		snapshots = append(snapshots, WaybackSnapshot{
			Timestamp:  ts,
			Date:       date,
			URL:        row[3],
			ArchiveURL: archiveURL,
			Status:     row[1],
			MimeType:   row[2],
			Available:  true,
		})
	}

	return snapshots, nil
}

// Wayback Availability API (단건 최신 스냅샷)
func fetchWaybackAvailability(targetURL string) (WaybackSnapshot, error) {
	apiURL := "https://archive.org/wayback/available?url=" + url.QueryEscape(targetURL)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return WaybackSnapshot{}, err
	}
	defer resp.Body.Close()

	var result struct {
		URL               string `json:"url"`
		ArchivedSnapshots struct {
			Closest struct {
				Available bool   `json:"available"`
				URL       string `json:"url"`
				Timestamp string `json:"timestamp"`
				Status    string `json:"status"`
			} `json:"closest"`
		} `json:"archived_snapshots"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	c := result.ArchivedSnapshots.Closest

	return WaybackSnapshot{
		Timestamp:  c.Timestamp,
		Date:       formatWaybackDate(c.Timestamp),
		URL:        targetURL,
		ArchiveURL: c.URL,
		Status:     c.Status,
		Available:  c.Available,
	}, nil
}

func formatWaybackDate(ts string) string {
	if len(ts) < 8 {
		return ts
	}
	return fmt.Sprintf("%s-%s-%s", ts[0:4], ts[4:6], ts[6:8])
}

func buildWaybackSummary(targetURL string, snaps []WaybackSnapshot, fromYear, toYear int) string {
	if len(snaps) == 0 {
		return fmt.Sprintf("❌ **%s** 의 %d~%d년 사이 아카이브 스냅샷이 없습니다.", targetURL, fromYear, toYear)
	}

	oldest := snaps[0].Date
	newest := snaps[len(snaps)-1].Date
	firstLink := snaps[0].ArchiveURL
	lastLink  := snaps[len(snaps)-1].ArchiveURL

	return fmt.Sprintf(
		"✅ **%s** 아카이브 스냅샷 **%d개** 발견 (%d~%d년)\n\n"+
			"📅 가장 오래된 버전: [%s](%s)\n"+
			"📅 가장 최근 버전: [%s](%s)\n\n"+
			"아래 목록에서 원하는 날짜의 버전을 클릭하세요.",
		targetURL, len(snaps), fromYear, toYear,
		oldest, firstLink,
		newest, lastLink,
	)
}
