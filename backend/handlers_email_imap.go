//go:build windows

package main

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ══════════════════════════════════════════════════════════════════
//  IMAP/SMTP Email Backend (Naver / Daum / Kakao / Gmail / Custom)
//  - AES-256 암호화된 계정 저장
//  - IMAP 수신 + SMTP 발신
//  - AI 자동 분류 + 스마트 답장 제안
// ══════════════════════════════════════════════════════════════════

// ── 암호화 키 (32바이트 AES-256) ────────────────────────────────
var imapAESKey = []byte("Nexus-IMAP-Key-2025-AES256-Secure")[:32]

func imapEncrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(imapAESKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func imapDecrypt(enc string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(imapAESKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(data) < gcm.NonceSize() {
		return "", fmt.Errorf("암호화 데이터가 너무 짧습니다")
	}
	nonce, ciphertext := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// ── Provider 설정 ────────────────────────────────────────────────

type EmailProvider struct {
	IMAPHost string
	IMAPPort int
	SMTPHost string
	SMTPPort int
}

var emailProviders = map[string]EmailProvider{
	"naver":   {IMAPHost: "imap.naver.com", IMAPPort: 993, SMTPHost: "smtp.naver.com", SMTPPort: 587},
	"daum":    {IMAPHost: "imap.daum.net", IMAPPort: 993, SMTPHost: "smtp.daum.net", SMTPPort: 465},
	"kakao":   {IMAPHost: "imap.kakao.com", IMAPPort: 993, SMTPHost: "smtp.kakao.com", SMTPPort: 465},
	"gmail":   {IMAPHost: "imap.gmail.com", IMAPPort: 993, SMTPHost: "smtp.gmail.com", SMTPPort: 587},
	"outlook": {IMAPHost: "outlook.office365.com", IMAPPort: 993, SMTPHost: "smtp.office365.com", SMTPPort: 587},
	"custom":  {IMAPHost: "", IMAPPort: 993, SMTPHost: "", SMTPPort: 587},
}

// ── 계정 데이터 ──────────────────────────────────────────────────

type IMAPAccount struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	EncPassword  string `json:"enc_password"`
	Provider     string `json:"provider"`
	IMAPHost     string `json:"imap_host,omitempty"`
	IMAPPort     int    `json:"imap_port,omitempty"`
	SMTPHost     string `json:"smtp_host,omitempty"`
	SMTPPort     int    `json:"smtp_port,omitempty"`
	CreatedAt    string `json:"created_at"`
}

type IMAPAccountPublic struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Provider  string `json:"provider"`
	CreatedAt string `json:"created_at"`
}

var (
	imapAccountsMu sync.RWMutex
	imapAccounts   []IMAPAccount
)

func imapAccountsPath() string {
	appdata := os.Getenv("APPDATA")
	if appdata == "" {
		appdata = os.TempDir()
	}
	dir := filepath.Join(appdata, "Nexus")
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "email_accounts.json")
}

func loadIMAPAccounts() {
	data, err := os.ReadFile(imapAccountsPath())
	if err != nil {
		return
	}
	imapAccountsMu.Lock()
	defer imapAccountsMu.Unlock()
	json.Unmarshal(data, &imapAccounts)
}

func saveIMAPAccounts() error {
	imapAccountsMu.RLock()
	defer imapAccountsMu.RUnlock()
	data, err := json.MarshalIndent(imapAccounts, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(imapAccountsPath(), data, 0600)
}

func init() {
	loadIMAPAccounts()
}

// ── 간이 IMAP 클라이언트 (TCP 직접 구현) ────────────────────────

type imapConn struct {
	conn   net.Conn
	reader *bufio.Reader
	tag    int
}

func dialIMAP(host string, port int) (*imapConn, error) {
	addr := fmt.Sprintf("%s:%d", host, port)
	tlsCfg := &tls.Config{ServerName: host}
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 15 * time.Second}, "tcp", addr, tlsCfg)
	if err != nil {
		return nil, fmt.Errorf("IMAP 연결 실패 (%s): %w", addr, err)
	}
	ic := &imapConn{conn: conn, reader: bufio.NewReader(conn)}
	// Read greeting
	ic.readLine()
	return ic, nil
}

func (ic *imapConn) readLine() (string, error) {
	ic.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	return ic.reader.ReadString('\n')
}

func (ic *imapConn) readUntilTagged(tag string) ([]string, error) {
	var lines []string
	for {
		line, err := ic.readLine()
		if err != nil {
			return lines, err
		}
		line = strings.TrimRight(line, "\r\n")
		lines = append(lines, line)
		if strings.HasPrefix(line, tag) {
			break
		}
	}
	return lines, nil
}

func (ic *imapConn) cmd(format string, args ...any) ([]string, error) {
	ic.tag++
	tag := fmt.Sprintf("A%04d", ic.tag)
	cmd := fmt.Sprintf(tag+" "+format+"\r\n", args...)
	ic.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if _, err := io.WriteString(ic.conn, cmd); err != nil {
		return nil, err
	}
	return ic.readUntilTagged(tag)
}

func (ic *imapConn) login(user, pass string) error {
	lines, err := ic.cmd("LOGIN %q %q", user, pass)
	if err != nil {
		return err
	}
	for _, l := range lines {
		if strings.Contains(l, "OK") {
			return nil
		}
		if strings.Contains(l, "NO") || strings.Contains(l, "BAD") {
			return fmt.Errorf("로그인 실패: %s", l)
		}
	}
	return fmt.Errorf("로그인 응답 없음")
}

func (ic *imapConn) close() {
	ic.cmd("LOGOUT")
	ic.conn.Close()
}

// fetchIMAPInbox: IMAP으로 받은 메일 가져오기
func fetchIMAPInbox(acc IMAPAccount, limit int) ([]EmailItem, error) {
	prov := emailProviders[acc.Provider]
	host := prov.IMAPHost
	port := prov.IMAPPort
	if acc.IMAPHost != "" {
		host = acc.IMAPHost
		port = acc.IMAPPort
	}

	pw, err := imapDecrypt(acc.EncPassword)
	if err != nil {
		return nil, fmt.Errorf("비밀번호 복호화 실패: %w", err)
	}

	ic, err := dialIMAP(host, port)
	if err != nil {
		return nil, err
	}
	defer ic.close()

	if err := ic.login(acc.Email, pw); err != nil {
		return nil, err
	}

	// SELECT INBOX
	lines, err := ic.cmd("SELECT INBOX")
	if err != nil {
		return nil, fmt.Errorf("INBOX 선택 실패: %w", err)
	}

	// 총 메일 수 파악
	total := 0
	for _, l := range lines {
		var n int
		cnt, _ := fmt.Sscanf(l, "* %d EXISTS", &n)
		if cnt == 1 {
			total = n
			break
		}
	}
	if total == 0 {
		return []EmailItem{}, nil
	}

	// 최근 N개 fetch
	start := total - limit + 1
	if start < 1 {
		start = 1
	}
	fetchRange := fmt.Sprintf("%d:%d", start, total)

	lines, err = ic.cmd("FETCH %s (ENVELOPE BODY[TEXT]<0.2000>)", fetchRange)
	if err != nil {
		return nil, fmt.Errorf("메일 가져오기 실패: %w", err)
	}

	// 간이 파싱
	var emails []EmailItem
	var current EmailItem
	inFetch := false

	for _, l := range lines {
		if strings.Contains(l, "FETCH") && strings.Contains(l, "ENVELOPE") {
			inFetch = true
			current = EmailItem{}
		}
		if inFetch {
			if strings.Contains(l, "ENVELOPE") {
				// 간이 파싱: subject와 sender 추출
				parts := splitIMAPEnvelope(l)
				if len(parts) >= 2 {
					current.Subject = decodeIMAPString(parts[1])
				}
				if len(parts) >= 3 {
					current.Sender = decodeIMAPString(parts[2])
				}
				current.ReceivedAt = time.Now().Format(time.RFC3339)
			}
			if strings.Contains(l, "BODY[TEXT]") || (len(l) > 0 && !strings.HasPrefix(l, "*") && !strings.HasPrefix(l, "A") && inFetch) {
				current.Body += l
			}
			if l == ")" || strings.HasSuffix(l, ")") {
				if current.Subject != "" || current.Sender != "" {
					if current.Subject == "" {
						current.Subject = "(제목 없음)"
					}
					emails = append(emails, current)
					inFetch = false
				}
			}
		}
	}

	// 역순 정렬 (최신 먼저)
	for i, j := 0, len(emails)-1; i < j; i, j = i+1, j-1 {
		emails[i], emails[j] = emails[j], emails[i]
	}

	if len(emails) > limit {
		emails = emails[:limit]
	}

	return emails, nil
}

func splitIMAPEnvelope(s string) []string {
	var parts []string
	depth := 0
	current := ""
	inQuote := false
	for _, c := range s {
		switch {
		case c == '"' && !inQuote:
			inQuote = true
			current += string(c)
		case c == '"' && inQuote:
			inQuote = false
			current += string(c)
			parts = append(parts, current)
			current = ""
		case c == '(' && !inQuote:
			depth++
			current += string(c)
		case c == ')' && !inQuote:
			depth--
			current += string(c)
			if depth == 0 {
				parts = append(parts, current)
				current = ""
			}
		case c == ' ' && !inQuote && depth == 0:
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		default:
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func decodeIMAPString(s string) string {
	s = strings.Trim(s, `"`)
	// =?UTF-8?B?...?= 디코딩
	if strings.Contains(s, "=?") {
		// base64 인코딩된 제목 처리
		for strings.Contains(s, "=?") {
			start := strings.Index(s, "=?")
			end := strings.Index(s[start:], "?=")
			if end < 0 {
				break
			}
			encoded := s[start : start+end+2]
			parts := strings.Split(encoded, "?")
			if len(parts) >= 4 {
				encoding := strings.ToUpper(parts[2])
				payload := parts[3]
				var decoded string
				if encoding == "B" {
					b, err := base64.StdEncoding.DecodeString(payload)
					if err == nil {
						decoded = string(b)
					}
				} else if encoding == "Q" {
					decoded = strings.ReplaceAll(payload, "_", " ")
				}
				if decoded != "" {
					s = s[:start] + decoded + s[start+end+2:]
					continue
				}
			}
			break
		}
	}
	return s
}

// ── SMTP 발송 ────────────────────────────────────────────────────

func sendIMAPEmail(acc IMAPAccount, to, subject, body string) error {
	prov := emailProviders[acc.Provider]
	smtpHost := prov.SMTPHost
	smtpPort := prov.SMTPPort
	if acc.SMTPHost != "" {
		smtpHost = acc.SMTPHost
		smtpPort = acc.SMTPPort
	}

	pw, err := imapDecrypt(acc.EncPassword)
	if err != nil {
		return fmt.Errorf("비밀번호 복호화 실패: %w", err)
	}

	from := acc.Email
	addr := fmt.Sprintf("%s:%d", smtpHost, smtpPort)

	msg := buildMIMEMessage(from, to, subject, body)

	// TLS 기반 포트 (465) vs STARTTLS 기반 포트 (587)
	if smtpPort == 465 {
		tlsCfg := &tls.Config{ServerName: smtpHost}
		conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 15 * time.Second}, "tcp", addr, tlsCfg)
		if err != nil {
			return fmt.Errorf("SMTP TLS 연결 실패: %w", err)
		}
		c, err := smtp.NewClient(conn, smtpHost)
		if err != nil {
			return err
		}
		defer c.Close()
		auth := smtp.PlainAuth("", from, pw, smtpHost)
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("SMTP 인증 실패: %w", err)
		}
		if err := c.Mail(from); err != nil {
			return err
		}
		if err := c.Rcpt(to); err != nil {
			return err
		}
		wc, err := c.Data()
		if err != nil {
			return err
		}
		defer wc.Close()
		_, err = wc.Write([]byte(msg))
		return err
	}

	// STARTTLS (587)
	auth := smtp.PlainAuth("", from, pw, smtpHost)
	return smtp.SendMail(addr, auth, from, []string{to}, []byte(msg))
}

func buildMIMEMessage(from, to, subject, body string) string {
	var buf bytes.Buffer
	buf.WriteString("From: " + from + "\r\n")
	buf.WriteString("To: " + to + "\r\n")
	subjectB64 := base64.StdEncoding.EncodeToString([]byte(subject))
	buf.WriteString("Subject: =?UTF-8?B?" + subjectB64 + "?=\r\n")
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	buf.WriteString("Content-Transfer-Encoding: base64\r\n")
	buf.WriteString("\r\n")
	bodyB64 := base64.StdEncoding.EncodeToString([]byte(body))
	// 76자마다 줄바꿈 (RFC 2045)
	for i := 0; i < len(bodyB64); i += 76 {
		end := i + 76
		if end > len(bodyB64) {
			end = len(bodyB64)
		}
		buf.WriteString(bodyB64[i:end] + "\r\n")
	}
	return buf.String()
}

// ── 자동 분류 (키워드 기반 간이 분류) ─────────────────────────

type imapEmailCategory struct {
	Category   string  `json:"category"` // work|personal|promotion|spam
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason"`
}

func imapClassifyEmail(subject, body string) imapEmailCategory {
	combined := strings.ToLower(subject + " " + body)

	spamKeywords := []string{"무료", "당첨", "광고", "쿠폰", "이벤트", "할인", "혜택", "free", "win", "prize", "offer", "sale", "discount", "unsubscribe"}
	workKeywords := []string{"회의", "보고", "업무", "프로젝트", "일정", "계약", "견적", "거래", "서류", "invoice", "meeting", "report", "project", "deadline", "contract"}
	promoKeywords := []string{"newsletter", "뉴스레터", "구독", "소식", "공지", "업데이트", "notice", "announcement"}

	spamScore := 0
	workScore := 0
	promoScore := 0

	for _, kw := range spamKeywords {
		if strings.Contains(combined, kw) {
			spamScore++
		}
	}
	for _, kw := range workKeywords {
		if strings.Contains(combined, kw) {
			workScore++
		}
	}
	for _, kw := range promoKeywords {
		if strings.Contains(combined, kw) {
			promoScore++
		}
	}

	if spamScore >= 3 {
		return imapEmailCategory{Category: "spam", Confidence: 0.85, Reason: "스팸 키워드 다수 감지"}
	}
	if workScore > promoScore && workScore >= 2 {
		return imapEmailCategory{Category: "work", Confidence: 0.8, Reason: "업무 관련 키워드 감지"}
	}
	if promoScore >= 2 {
		return imapEmailCategory{Category: "promotion", Confidence: 0.75, Reason: "프로모션 키워드 감지"}
	}
	return imapEmailCategory{Category: "personal", Confidence: 0.6, Reason: "개인 이메일로 분류"}
}

// ── AI 스마트 답장 제안 ──────────────────────────────────────────

func getEmailReplySuggestions(subject, body string) ([]string, error) {
	llmMu.RLock()
	pKey := llmPerplexityKey
	llmMu.RUnlock()

	if pKey == "" {
		// 폴백: 키워드 기반 기본 답장
		return []string{
			"감사합니다. 확인 후 연락 드리겠습니다.",
			"네, 알겠습니다. 검토해보겠습니다.",
			"감사합니다. 좋은 하루 되세요.",
		}, nil
	}

	prompt := fmt.Sprintf(`다음 이메일에 대한 한국어 답장 3가지를 제안해주세요. 각각 다른 톤(공식적/친근함/간결함)으로 작성해주세요.

제목: %s
내용: %s

JSON으로만 응답하세요:
{"replies": ["답장1", "답장2", "답장3"]}`, subject, body)

	raw, _, err := callGroq(pKey, groqChatModel, []groqMsg{
		{Role: "user", Content: prompt},
	}, 400, true)
	if err != nil {
		return []string{
			"감사합니다. 확인 후 연락 드리겠습니다.",
			"네, 알겠습니다. 검토해보겠습니다.",
			"좋은 하루 되세요.",
		}, nil
	}

	// JSON 추출
	clean := strings.TrimSpace(raw)
	if idx := strings.Index(clean, "{"); idx >= 0 {
		clean = clean[idx:]
	}
	if idx := strings.LastIndex(clean, "}"); idx >= 0 {
		clean = clean[:idx+1]
	}

	var resp struct {
		Replies []string `json:"replies"`
	}
	if err := json.Unmarshal([]byte(clean), &resp); err != nil || len(resp.Replies) == 0 {
		return []string{
			"감사합니다. 확인 후 연락 드리겠습니다.",
			"네, 알겠습니다. 검토해보겠습니다.",
			"좋은 하루 되세요.",
		}, nil
	}

	return resp.Replies, nil
}

// ── HTTP 핸들러 ──────────────────────────────────────────────────

// GET /api/imap/accounts
func handleIMAPAccountList(w http.ResponseWriter, r *http.Request) {
	imapAccountsMu.RLock()
	defer imapAccountsMu.RUnlock()
	var pub []IMAPAccountPublic
	for _, a := range imapAccounts {
		pub = append(pub, IMAPAccountPublic{
			ID: a.ID, Name: a.Name, Email: a.Email,
			Provider: a.Provider, CreatedAt: a.CreatedAt,
		})
	}
	if pub == nil {
		pub = []IMAPAccountPublic{}
	}
	json200(w, map[string]any{"accounts": pub, "count": len(pub)})
}

// POST /api/imap/accounts
func handleIMAPAccountAdd(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Provider string `json:"provider"`
		IMAPHost string `json:"imap_host"`
		IMAPPort int    `json:"imap_port"`
		SMTPHost string `json:"smtp_host"`
		SMTPPort int    `json:"smtp_port"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" || req.Password == "" {
		json200(w, map[string]any{"success": false, "message": "이메일과 비밀번호가 필요합니다"})
		return
	}

	if req.Provider == "" {
		req.Provider = "custom"
	}
	if _, ok := emailProviders[req.Provider]; !ok {
		req.Provider = "custom"
	}

	// 비밀번호 암호화
	encPw, err := imapEncrypt(req.Password)
	if err != nil {
		json200(w, map[string]any{"success": false, "message": "암호화 실패: " + err.Error()})
		return
	}

	// 연결 테스트
	testAcc := IMAPAccount{
		Email: req.Email, EncPassword: encPw,
		Provider: req.Provider,
		IMAPHost: req.IMAPHost, IMAPPort: req.IMAPPort,
		SMTPHost: req.SMTPHost, SMTPPort: req.SMTPPort,
	}

	prov := emailProviders[req.Provider]
	host := prov.IMAPHost
	port := prov.IMAPPort
	if req.IMAPHost != "" {
		host = req.IMAPHost
		port = req.IMAPPort
	}
	if port == 0 {
		port = 993
	}

	ic, connErr := dialIMAP(host, port)
	if connErr != nil {
		json200(w, map[string]any{"success": false, "message": "IMAP 연결 실패: " + connErr.Error()})
		return
	}
	loginErr := ic.login(req.Email, req.Password)
	ic.close()
	if loginErr != nil {
		json200(w, map[string]any{"success": false, "message": "로그인 실패: " + loginErr.Error()})
		return
	}

	_ = testAcc

	acc := IMAPAccount{
		ID:          fmt.Sprintf("imap_%d", time.Now().UnixNano()),
		Name:        req.Name,
		Email:       req.Email,
		EncPassword: encPw,
		Provider:    req.Provider,
		IMAPHost:    req.IMAPHost,
		IMAPPort:    req.IMAPPort,
		SMTPHost:    req.SMTPHost,
		SMTPPort:    req.SMTPPort,
		CreatedAt:   time.Now().Format(time.RFC3339),
	}

	imapAccountsMu.Lock()
	imapAccounts = append(imapAccounts, acc)
	imapAccountsMu.Unlock()

	saveIMAPAccounts()

	json200(w, map[string]any{
		"success": true,
		"id":      acc.ID,
		"message": fmt.Sprintf("%s 계정이 추가됐습니다", req.Email),
	})
}

// DELETE /api/imap/accounts/:id — query param: ?id=xxx
func handleIMAPAccountDelete(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		json200(w, map[string]any{"success": false, "message": "id가 필요합니다"})
		return
	}

	imapAccountsMu.Lock()
	found := false
	for i, a := range imapAccounts {
		if a.ID == id {
			imapAccounts = append(imapAccounts[:i], imapAccounts[i+1:]...)
			found = true
			break
		}
	}
	imapAccountsMu.Unlock()

	if !found {
		json200(w, map[string]any{"success": false, "message": "계정을 찾을 수 없습니다"})
		return
	}

	saveIMAPAccounts()
	json200(w, map[string]any{"success": true, "message": "계정이 삭제됐습니다"})
}

// GET /api/imap/inbox?account_id=&limit=
func handleIMAPInbox(w http.ResponseWriter, r *http.Request) {
	accID := r.URL.Query().Get("account_id")
	limit := 20
	fmt.Sscanf(r.URL.Query().Get("limit"), "%d", &limit)
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	imapAccountsMu.RLock()
	var acc *IMAPAccount
	for i, a := range imapAccounts {
		if a.ID == accID {
			cp := imapAccounts[i]
			acc = &cp
			break
		}
	}
	imapAccountsMu.RUnlock()

	if acc == nil {
		json200(w, map[string]any{"success": false, "message": "계정을 찾을 수 없습니다"})
		return
	}

	emails, err := fetchIMAPInbox(*acc, limit)
	if err != nil {
		json200(w, map[string]any{
			"success": false,
			"emails":  []EmailItem{},
			"message": "메일 가져오기 실패: " + err.Error(),
		})
		return
	}

	unread := 0
	for _, e := range emails {
		if !e.IsRead {
			unread++
		}
	}

	json200(w, map[string]any{
		"success": true,
		"emails":  emails,
		"total":   len(emails),
		"unread":  unread,
		"message": fmt.Sprintf("메일 %d개를 가져왔습니다", len(emails)),
	})
}

// POST /api/imap/send
func handleIMAPSend(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AccountID string `json:"account_id"`
		To        string `json:"to"`
		Subject   string `json:"subject"`
		Body      string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.To == "" {
		json200(w, map[string]any{"success": false, "message": "수신자가 필요합니다"})
		return
	}

	imapAccountsMu.RLock()
	var acc *IMAPAccount
	for i, a := range imapAccounts {
		if a.ID == req.AccountID {
			cp := imapAccounts[i]
			acc = &cp
			break
		}
	}
	imapAccountsMu.RUnlock()

	if acc == nil && req.AccountID != "" {
		json200(w, map[string]any{"success": false, "message": "계정을 찾을 수 없습니다"})
		return
	}

	// 계정이 없으면 Outlook으로 폴백
	if acc == nil {
		err := sendOutlookEmail(req.To, req.Subject, req.Body)
		if err != nil {
			json200(w, map[string]any{"success": false, "message": "발송 실패: " + err.Error()})
			return
		}
		json200(w, map[string]any{"success": true, "message": "이메일을 발송했습니다 (Outlook)"})
		return
	}

	if err := sendIMAPEmail(*acc, req.To, req.Subject, req.Body); err != nil {
		json200(w, map[string]any{"success": false, "message": "발송 실패: " + err.Error()})
		return
	}
	json200(w, map[string]any{"success": true, "message": fmt.Sprintf("%s에게 이메일을 발송했습니다", req.To)})
}

// GET /api/imap/reply-suggestions?subject=&body=
func handleIMAPReplySuggestions(w http.ResponseWriter, r *http.Request) {
	subject := r.URL.Query().Get("subject")
	body := r.URL.Query().Get("body")

	suggestions, err := getEmailReplySuggestions(subject, body)
	if err != nil {
		json200(w, map[string]any{"success": false, "message": err.Error()})
		return
	}
	json200(w, map[string]any{
		"success":     true,
		"suggestions": suggestions,
		"count":       len(suggestions),
	})
}

// POST /api/imap/classify
func handleIMAPClassify(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Subject string `json:"subject"`
		Body    string `json:"body"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	result := imapClassifyEmail(req.Subject, req.Body)
	json200(w, map[string]any{
		"success":    true,
		"category":   result.Category,
		"confidence": result.Confidence,
		"reason":     result.Reason,
	})
}
