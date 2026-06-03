// handlers_gmail.go — Gmail API 호출 (Google OAuth 통합)
// 사장님 비전: Google 로그인 1회 = Gmail 자동 연동 (별도 IMAP/앱비번 입력 X)
//
// 흐름:
//   1. signInWithGoogle scope 에 gmail.readonly + gmail.send 포함 (이미 적용)
//   2. Supabase 가 provider_token 제공 (Google access token)
//   3. 프론트가 JWT Bearer 와 provider_token 을 백엔드로 전달
//   4. 백엔드가 Google Gmail API 호출 (https://gmail.googleapis.com/gmail/v1/...)
//   5. Outlook COM 실패시 자동 폴백 (Windows 만)

package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// GmailMessage — 간소화된 Gmail 메일
type GmailMessage struct {
	ID        string `json:"id"`
	From      string `json:"from"`
	To        string `json:"to"`
	Subject   string `json:"subject"`
	Snippet   string `json:"snippet"`
	Date      string `json:"date"`
	Unread    bool   `json:"unread"`
}

var gmailHTTPClient = &http.Client{Timeout: 10 * time.Second}

// gmailListInbox — 받은편지함 N개 조회
// accessToken: Google OAuth provider token (Supabase 가 발급)
func gmailListInbox(accessToken string, max int) ([]GmailMessage, error) {
	if accessToken == "" {
		return nil, fmt.Errorf("gmail access token required")
	}
	if max <= 0 || max > 50 {
		max = 10
	}
	// 1) message ID 리스트
	listURL := fmt.Sprintf("https://gmail.googleapis.com/gmail/v1/users/me/messages?labelIds=INBOX&maxResults=%d", max)
	req, _ := http.NewRequest("GET", listURL, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := gmailHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("gmail list %d: %s", resp.StatusCode, string(body)[:200])
	}
	var listResp struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(body, &listResp); err != nil {
		return nil, err
	}
	// 2) 각 message 메타 fetch (병렬 X — 간단화, 향후 goroutine)
	out := make([]GmailMessage, 0, len(listResp.Messages))
	for _, m := range listResp.Messages {
		msg, err := gmailGetMessage(accessToken, m.ID)
		if err == nil {
			out = append(out, msg)
		}
	}
	return out, nil
}

// gmailGetMessage — 단일 메일 메타 fetch
func gmailGetMessage(accessToken, msgID string) (GmailMessage, error) {
	u := fmt.Sprintf("https://gmail.googleapis.com/gmail/v1/users/me/messages/%s?format=metadata&metadataHeaders=From&metadataHeaders=To&metadataHeaders=Subject&metadataHeaders=Date", msgID)
	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := gmailHTTPClient.Do(req)
	if err != nil {
		return GmailMessage{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var raw struct {
		ID         string   `json:"id"`
		Snippet    string   `json:"snippet"`
		LabelIDs   []string `json:"labelIds"`
		Payload    struct {
			Headers []struct {
				Name  string `json:"name"`
				Value string `json:"value"`
			} `json:"headers"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return GmailMessage{}, err
	}
	msg := GmailMessage{ID: raw.ID, Snippet: raw.Snippet}
	for _, h := range raw.Payload.Headers {
		switch strings.ToLower(h.Name) {
		case "from":
			msg.From = h.Value
		case "to":
			msg.To = h.Value
		case "subject":
			msg.Subject = h.Value
		case "date":
			msg.Date = h.Value
		}
	}
	for _, l := range raw.LabelIDs {
		if l == "UNREAD" {
			msg.Unread = true
			break
		}
	}
	return msg, nil
}

// gmailSend — 메일 전송
func gmailSend(accessToken, to, subject, body string) error {
	if accessToken == "" {
		return fmt.Errorf("gmail access token required")
	}
	// RFC 2822 포맷 메시지 작성
	raw := fmt.Sprintf("To: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		to, subjectEncode(subject), body)
	// Gmail API는 base64url 인코딩
	rawB64 := base64.URLEncoding.EncodeToString([]byte(raw))
	rawB64 = strings.TrimRight(rawB64, "=")

	payload, _ := json.Marshal(map[string]string{"raw": rawB64})
	req, _ := http.NewRequest("POST", "https://gmail.googleapis.com/gmail/v1/users/me/messages/send",
		strings.NewReader(string(payload)))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := gmailHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gmail send %d: %s", resp.StatusCode, string(raw)[:200])
	}
	return nil
}

// gmailSearch — 검색어로 조회 (Gmail 검색 문법)
func gmailSearch(accessToken, query string, max int) ([]GmailMessage, error) {
	if max <= 0 || max > 50 {
		max = 10
	}
	u := fmt.Sprintf("https://gmail.googleapis.com/gmail/v1/users/me/messages?q=%s&maxResults=%d",
		url.QueryEscape(query), max)
	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := gmailHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var listResp struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(body, &listResp); err != nil {
		return nil, err
	}
	out := make([]GmailMessage, 0, len(listResp.Messages))
	for _, m := range listResp.Messages {
		if msg, err := gmailGetMessage(accessToken, m.ID); err == nil {
			out = append(out, msg)
		}
	}
	return out, nil
}

func subjectEncode(s string) string {
	// UTF-8 Subject 인코딩 (Base64 MIME)
	for _, r := range s {
		if r > 127 {
			b64 := base64.StdEncoding.EncodeToString([]byte(s))
			return "=?UTF-8?B?" + b64 + "?="
		}
	}
	return s
}

// ── HTTP 엔드포인트 ────────────────────────────────────────

// handleGmailInbox — GET /api/gmail/inbox
// Header: X-Provider-Token (Google OAuth access token)
func handleGmailInbox(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Provider-Token")
	if token == "" {
		writeJSON(w, 401, map[string]any{
			"success": false,
			"message": "Google access token required. Sign in with Google + grant Gmail scope.",
			"action":  "google_oauth_required",
		})
		return
	}
	maxN := 10
	if m := r.URL.Query().Get("max"); m != "" {
		var n int
		fmt.Sscanf(m, "%d", &n)
		if n > 0 {
			maxN = n
		}
	}
	msgs, err := gmailListInbox(token, maxN)
	if err != nil {
		writeJSON(w, 502, map[string]any{"success": false, "message": err.Error()})
		return
	}
	json200(w, map[string]any{
		"success":  true,
		"messages": msgs,
		"count":    len(msgs),
		"provider": "gmail",
	})
}

// handleGmailSend — POST /api/gmail/send
// Header: X-Provider-Token
// Body: {"to":"...", "subject":"...", "body":"..."}
func handleGmailSend(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Provider-Token")
	if token == "" {
		writeJSON(w, 401, map[string]any{"success": false, "message": "Google access token required"})
		return
	}
	var req struct {
		To      string `json:"to"`
		Subject string `json:"subject"`
		Body    string `json:"body"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.To == "" || req.Subject == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "to + subject required"})
		return
	}
	if err := gmailSend(token, req.To, req.Subject, req.Body); err != nil {
		writeJSON(w, 502, map[string]any{"success": false, "message": err.Error()})
		return
	}
	json200(w, map[string]any{"success": true, "message": "✅ 메일 전송 완료"})
}

// handleGmailSearch — GET /api/gmail/search?q=...&max=10
func handleGmailSearch(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Provider-Token")
	if token == "" {
		writeJSON(w, 401, map[string]any{"success": false, "message": "Google access token required"})
		return
	}
	q := r.URL.Query().Get("q")
	if q == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "q required"})
		return
	}
	msgs, err := gmailSearch(token, q, 10)
	if err != nil {
		writeJSON(w, 502, map[string]any{"success": false, "message": err.Error()})
		return
	}
	json200(w, map[string]any{"success": true, "messages": msgs, "count": len(msgs)})
}
