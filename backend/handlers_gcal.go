//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ══════════════════════════════════════════════════════════════════
//  Google Calendar OAuth2 (Authorization Code Flow)
//  - GET  /api/calendar/google/auth     → OAuth URL 반환
//  - GET  /api/calendar/google/callback → 코드 교환 + 토큰 저장
//  - GET  /api/calendar/google/status   → 연결 상태 확인
//  - POST /api/calendar/google/disconnect → 토큰 삭제
// ══════════════════════════════════════════════════════════════════

const (
	gcalTokenFile   = "gcal_token.json"
	gcalRedirectURI = "http://127.0.0.1:17891/api/calendar/google/callback"
	gcalScope       = "https://www.googleapis.com/auth/calendar"
)

type gcalToken struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at"`
}

var (
	gcalMu    sync.RWMutex
	gcalCache *gcalToken
)

func gcalTokenPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".nexus", gcalTokenFile)
}

func gcalClientID() string {
	if v := os.Getenv("GOOGLE_CLIENT_ID"); v != "" {
		return v
	}
	// 번들 설정 파일에서 읽기
	home, _ := os.UserHomeDir()
	b, err := os.ReadFile(filepath.Join(home, ".nexus", "google_oauth.json"))
	if err == nil {
		var cfg struct{ ClientID string `json:"client_id"` }
		if json.Unmarshal(b, &cfg) == nil && cfg.ClientID != "" {
			return cfg.ClientID
		}
	}
	return ""
}

func gcalClientSecret() string {
	if v := os.Getenv("GOOGLE_CLIENT_SECRET"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	b, err := os.ReadFile(filepath.Join(home, ".nexus", "google_oauth.json"))
	if err == nil {
		var cfg struct{ ClientSecret string `json:"client_secret"` }
		if json.Unmarshal(b, &cfg) == nil && cfg.ClientSecret != "" {
			return cfg.ClientSecret
		}
	}
	return ""
}

func loadGcalToken() *gcalToken {
	gcalMu.RLock()
	if gcalCache != nil {
		defer gcalMu.RUnlock()
		return gcalCache
	}
	gcalMu.RUnlock()

	b, err := os.ReadFile(gcalTokenPath())
	if err != nil {
		return nil
	}
	var tok gcalToken
	if err := json.Unmarshal(b, &tok); err != nil {
		return nil
	}
	gcalMu.Lock()
	gcalCache = &tok
	gcalMu.Unlock()
	return &tok
}

func saveGcalToken(tok *gcalToken) error {
	path := gcalTokenPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	b, err := json.Marshal(tok)
	if err != nil {
		return err
	}
	gcalMu.Lock()
	gcalCache = tok
	gcalMu.Unlock()
	return os.WriteFile(path, b, 0600)
}

// getValidAccessToken: 만료 시 자동 갱신
func getValidAccessToken() (string, error) {
	tok := loadGcalToken()
	if tok == nil {
		return "", fmt.Errorf("Google Calendar 연결 안 됨 — 먼저 연동해주세요")
	}
	// 만료 5분 전부터 갱신
	if time.Now().Before(tok.ExpiresAt.Add(-5 * time.Minute)) {
		return tok.AccessToken, nil
	}
	if tok.RefreshToken == "" {
		return "", fmt.Errorf("리프레시 토큰 없음 — 재연결 필요")
	}
	return refreshGcalToken(tok.RefreshToken)
}

func refreshGcalToken(refreshToken string) (string, error) {
	data := url.Values{
		"client_id":     {gcalClientID()},
		"client_secret": {gcalClientSecret()},
		"refresh_token": {refreshToken},
		"grant_type":    {"refresh_token"},
	}
	resp, err := http.PostForm("https://oauth2.googleapis.com/token", data)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		Error       string `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil || result.Error != "" {
		return "", fmt.Errorf("토큰 갱신 실패: %s", result.Error)
	}
	tok := loadGcalToken()
	tok.AccessToken = result.AccessToken
	tok.ExpiresAt = time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
	_ = saveGcalToken(tok)
	return result.AccessToken, nil
}

// GET /api/calendar/google/auth — OAuth URL 반환
func handleGCalAuth(w http.ResponseWriter, r *http.Request) {
	clientID := gcalClientID()
	if clientID == "" {
		http.Error(w, `{"error":"Google OAuth 미설정 — ~/.nexus/google_oauth.json 확인"}`, http.StatusServiceUnavailable)
		return
	}
	params := url.Values{
		"client_id":     {clientID},
		"redirect_uri":  {gcalRedirectURI},
		"response_type": {"code"},
		"scope":         {gcalScope + " https://www.googleapis.com/auth/gmail.readonly https://www.googleapis.com/auth/gmail.send"},
		"access_type":   {"offline"},
		"prompt":        {"consent"},
	}
	authURL := "https://accounts.google.com/o/oauth2/v2/auth?" + params.Encode()
	json200(w, map[string]string{"url": authURL})
}

// GET /api/calendar/google/callback — 코드 교환 + 토큰 저장
func handleGCalCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	errParam := r.URL.Query().Get("error")
	if errParam != "" {
		http.Error(w, fmt.Sprintf("OAuth 거부됨: %s", errParam), http.StatusBadRequest)
		return
	}
	if code == "" {
		http.Error(w, "코드 없음", http.StatusBadRequest)
		return
	}

	data := url.Values{
		"client_id":     {gcalClientID()},
		"client_secret": {gcalClientSecret()},
		"code":          {code},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {gcalRedirectURI},
	}
	resp, err := http.PostForm("https://oauth2.googleapis.com/token", data)
	if err != nil {
		http.Error(w, "토큰 요청 실패: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
		Error        string `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil || result.Error != "" {
		http.Error(w, "토큰 파싱 실패: "+result.Error, http.StatusInternalServerError)
		return
	}

	tok := &gcalToken{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		TokenType:    result.TokenType,
		ExpiresAt:    time.Now().Add(time.Duration(result.ExpiresIn) * time.Second),
	}
	if err := saveGcalToken(tok); err != nil {
		http.Error(w, "토큰 저장 실패: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 브라우저에 성공 페이지 표시
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!DOCTYPE html><html><head><meta charset="utf-8">
<style>body{background:#060612;color:#fff;font-family:sans-serif;display:flex;align-items:center;justify-content:center;height:100vh;flex-direction:column;gap:16px}
.ok{font-size:64px}.title{font-size:24px;font-weight:700;color:#22c55e}.sub{color:rgba(255,255,255,.5);font-size:14px}</style></head>
<body><div class="ok">✅</div><div class="title">Google 계정 연결 완료!</div>
<div class="sub">이 창을 닫고 Nexus AI로 돌아가세요</div>
<script>setTimeout(()=>window.close(),3000)</script></body></html>`)
}

// GET /api/calendar/google/status — 연결 상태 확인
func handleGCalStatus(w http.ResponseWriter, r *http.Request) {
	tok := loadGcalToken()
	if tok == nil {
		json200(w, map[string]any{"connected": false})
		return
	}
	// 이메일 확인 (선택적)
	email := ""
	if accessToken, err := getValidAccessToken(); err == nil {
		req, _ := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v3/userinfo", nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)
		if res, err := http.DefaultClient.Do(req); err == nil {
			defer res.Body.Close()
			var info struct{ Email string `json:"email"` }
			b, _ := io.ReadAll(res.Body)
			_ = json.Unmarshal(b, &info)
			email = info.Email
		}
	}
	json200(w, map[string]any{
		"connected": true,
		"email":     email,
		"expires":   tok.ExpiresAt.Format(time.RFC3339),
	})
}

// POST /api/calendar/google/disconnect — 토큰 삭제
func handleGCalDisconnect(w http.ResponseWriter, r *http.Request) {
	tok := loadGcalToken()
	if tok != nil && tok.AccessToken != "" {
		// Google 토큰 revoke (베스트 에포트)
		http.PostForm("https://oauth2.googleapis.com/revoke",
			url.Values{"token": {tok.AccessToken}})
	}
	gcalMu.Lock()
	gcalCache = nil
	gcalMu.Unlock()
	os.Remove(gcalTokenPath())
	json200(w, map[string]any{"ok": true})
}

// ── Google Calendar API 실제 연동 ────────────────────────────────

// gcalListEvents: Google Calendar API로 일정 조회
func gcalListEvents(timeMin, timeMax string) ([]map[string]any, error) {
	accessToken, err := getValidAccessToken()
	if err != nil {
		return nil, err
	}

	apiURL := fmt.Sprintf(
		"https://www.googleapis.com/calendar/v3/calendars/primary/events?timeMin=%s&timeMax=%s&singleEvents=true&orderBy=startTime&maxResults=50",
		url.QueryEscape(timeMin), url.QueryEscape(timeMax),
	)
	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Items []struct {
			Summary  string `json:"summary"`
			Location string `json:"location"`
			Start    struct {
				DateTime string `json:"dateTime"`
				Date     string `json:"date"`
			} `json:"start"`
			End struct {
				DateTime string `json:"dateTime"`
				Date     string `json:"date"`
			} `json:"end"`
			Organizer struct {
				Email string `json:"email"`
			} `json:"organizer"`
		} `json:"items"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	events := make([]map[string]any, 0, len(result.Items))
	for _, item := range result.Items {
		start := item.Start.DateTime
		if start == "" {
			start = item.Start.Date
		}
		end := item.End.DateTime
		if end == "" {
			end = item.End.Date
		}
		events = append(events, map[string]any{
			"subject":   item.Summary,
			"start":     start,
			"end":       end,
			"location":  item.Location,
			"organizer": item.Organizer.Email,
			"is_all_day": item.Start.DateTime == "",
		})
	}
	return events, nil
}

// gcalAddEvent: Google Calendar API로 일정 추가
func gcalAddEvent(subject, start, end, location string) error {
	accessToken, err := getValidAccessToken()
	if err != nil {
		return err
	}

	body := map[string]any{
		"summary":  subject,
		"location": location,
		"start":    map[string]string{"dateTime": start, "timeZone": "Asia/Seoul"},
		"end":      map[string]string{"dateTime": end, "timeZone": "Asia/Seoul"},
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST",
		"https://www.googleapis.com/calendar/v3/calendars/primary/events",
		strings.NewReader(string(b)),
	)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		rb, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Google Calendar 오류: %s", string(rb))
	}
	return nil
}
