package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ── 사용량 제한 상수 ────────────────────────────────────────
const (
	dailyFreeLimit    = 500 // Groq 무료 모델 (단순 채팅)
	dailyPremiumLimit = 50  // Claude/Perplexity/Tavily (웹검색·복잡 분석)
)

// ── 사용량 데이터 구조 ───────────────────────────────────────
type UsageRecord struct {
	Date         string `json:"date"`           // YYYY-MM-DD
	FreeCount    int    `json:"free_count"`     // Groq 호출 수
	PremiumCount int    `json:"premium_count"`  // Claude/Perplexity/Tavily 호출 수
	UserID       string `json:"user_id"`        // 사용자 식별자 (이메일 or IP)
}

type UsageStore struct {
	mu      sync.Mutex
	records map[string]*UsageRecord // key: "userID:date"
	path    string
}

var globalUsage = &UsageStore{
	records: map[string]*UsageRecord{},
}

func usagePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".nexus", "usage.json")
}

func (u *UsageStore) load() {
	data, err := os.ReadFile(usagePath())
	if err != nil {
		return
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	json.Unmarshal(data, &u.records)
}

func (u *UsageStore) save() {
	data, _ := json.MarshalIndent(u.records, "", "  ")
	os.WriteFile(usagePath(), data, 0600)
}

func (u *UsageStore) getRecord(userID string) *UsageRecord {
	today := time.Now().Format("2006-01-02")
	key := userID + ":" + today
	if r, ok := u.records[key]; ok {
		return r
	}
	r := &UsageRecord{Date: today, UserID: userID}
	u.records[key] = r
	return r
}

// CheckAndIncrement: 사용 가능 여부 확인 + 카운트 증가
// tier: "free" or "premium"
func (u *UsageStore) CheckAndIncrement(userID, tier string) (allowed bool, freeLeft, premiumLeft int) {
	u.mu.Lock()
	defer u.mu.Unlock()

	r := u.getRecord(userID)
	freeLeft = dailyFreeLimit - r.FreeCount
	premiumLeft = dailyPremiumLimit - r.PremiumCount

	if tier == "premium" {
		if r.PremiumCount >= dailyPremiumLimit {
			return false, freeLeft, 0
		}
		r.PremiumCount++
		premiumLeft--
	} else {
		if r.FreeCount >= dailyFreeLimit {
			return false, 0, premiumLeft
		}
		r.FreeCount++
		freeLeft--
	}

	go u.save()
	return true, freeLeft, premiumLeft
}

// GetStatus: 현재 사용량 조회
func (u *UsageStore) GetStatus(userID string) (freeUsed, premiumUsed, freeLeft, premiumLeft int) {
	u.mu.Lock()
	defer u.mu.Unlock()
	r := u.getRecord(userID)
	return r.FreeCount, r.PremiumCount,
		dailyFreeLimit - r.FreeCount,
		dailyPremiumLimit - r.PremiumCount
}

// ── 모델 티어 결정 ────────────────────────────────────────────
type ModelTier string

const (
	TierFree    ModelTier = "free"    // Groq llama (무료)
	TierPremium ModelTier = "premium" // Claude/Perplexity (유료)
)

// DecideModelTier: 액션 타입에 따라 필요한 tier 결정
func DecideModelTier(action string) ModelTier {
	// 프리미엄 액션: 웹 검색, 여행 계획, 가격 비교, 영상 검색, 워크플로우
	premiumActions := map[string]bool{
		"web_search":      true,
		"trip_plan":       true,
		"price_compare":   true,
		"video_search":    true,
		"workflow_preset": true,
		"multi_action":    true,
		"weather":         true,
	}
	if premiumActions[action] {
		return TierPremium
	}
	return TierFree // chat, calendar, persona_switch 등
}

// ── HTTP 핸들러: 사용량 조회 ──────────────────────────────────
func handleUsageStatus(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = r.RemoteAddr // 미로그인 시 IP로 식별
	}

	freeUsed, premiumUsed, freeLeft, premiumLeft := globalUsage.GetStatus(userID)
	json200(w, map[string]any{
		"user_id":       userID,
		"date":          time.Now().Format("2006-01-02"),
		"free": map[string]any{
			"used":  freeUsed,
			"left":  freeLeft,
			"limit": dailyFreeLimit,
		},
		"premium": map[string]any{
			"used":  premiumUsed,
			"left":  premiumLeft,
			"limit": dailyPremiumLimit,
		},
		"reset_at": time.Now().Truncate(24*time.Hour).Add(24*time.Hour).Format(time.RFC3339),
	})
}

// ── 사용량 초과 응답 생성 ────────────────────────────────────
func usageLimitResponse(tier ModelTier, freeLeft, premiumLeft int) CommandResponse {
	var msg string
	if tier == TierPremium {
		msg = fmt.Sprintf(
			"오늘 프리미엄 검색 횟수(%d회)를 모두 사용했어요 😅\n\n"+
				"• 내일 자정에 50회 자동 충전돼요\n"+
				"• 일반 대화(채팅)는 아직 %d회 가능해요\n"+
				"• 더 많이 쓰시려면 요금제를 확인해주세요",
			dailyPremiumLimit, freeLeft,
		)
	} else {
		msg = fmt.Sprintf(
			"오늘 사용 가능한 횟수(%d회)를 모두 사용했어요.\n내일 자정에 자동으로 충전돼요.",
			dailyFreeLimit,
		)
	}
	return CommandResponse{
		Success:  false,
		Message:  msg,
		Action:   "usage_limit",
		Duration: "0.00s",
	}
}

// ══════════════════════════════════════════════════════════════════
//  Feature-based daily usage limits (Pro paywall)
// ══════════════════════════════════════════════════════════════════

// featureLimits defines free-tier daily limits per feature.
// -1 means unlimited (pro/team users always get -1).
var featureLimits = map[string]int{
	"ai_request":      15, // free 티어 하루 15회
	"stock_analysis":  3,
	"medical_search":  3,
	"contract_review": 1,
	"legal_search":    3,
	"content_script":  5,
	"workflow_run":    10,
}

// featureUsageFile returns path like ~/.nexus/usage_20260521.json
func featureUsageFile() string {
	home, _ := os.UserHomeDir()
	date := time.Now().Format("20060102")
	return filepath.Join(home, ".nexus", fmt.Sprintf("usage_%s.json", date))
}

type featureUsageData map[string]map[string]int // userID → feature → count

var featureUsageMu sync.Mutex

func loadFeatureUsage() featureUsageData {
	data := make(featureUsageData)
	raw, err := os.ReadFile(featureUsageFile())
	if err != nil {
		return data
	}
	_ = json.Unmarshal(raw, &data)
	return data
}

func saveFeatureUsage(d featureUsageData) {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".nexus")
	_ = os.MkdirAll(dir, 0700)
	raw, _ := json.MarshalIndent(d, "", "  ")
	_ = os.WriteFile(featureUsageFile(), raw, 0600)
}

// getMachineID returns a stable machine UUID from ~/.nexus/machine_id
func getMachineID() string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".nexus")
	_ = os.MkdirAll(dir, 0700)
	p := filepath.Join(dir, "machine_id")
	if raw, err := os.ReadFile(p); err == nil {
		id := strings.TrimSpace(string(raw))
		if id != "" {
			return id
		}
	}
	// generate simple UUID v4
	f, err := os.Open("/dev/urandom")
	if err != nil {
		return "default-machine"
	}
	defer f.Close()
	b := make([]byte, 16)
	_, _ = f.Read(b)
	id := fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
	_ = os.WriteFile(p, []byte(id), 0600)
	return id
}

// getPlanFromJWT parses the JWT payload and returns the "plan" claim.
// Falls back to "free" if the token is missing or invalid.
func getPlanFromJWT(token string) string {
	if token == "" {
		return "free"
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "free"
	}
	payload := parts[1]
	// add padding
	switch len(payload) % 4 {
	case 2:
		payload += "=="
	case 3:
		payload += "="
	}
	raw, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return "free"
	}
	var claims map[string]any
	if err := json.Unmarshal(raw, &claims); err != nil {
		return "free"
	}
	if p, ok := claims["plan"].(string); ok && p != "" {
		return p
	}
	return "free"
}

// checkUsageLimit returns whether the user can use the feature today,
// and the current used/limit counts.
func checkUsageLimit(userID, feature string) (allowed bool, used int, limit int) {
	limit, known := featureLimits[feature]
	if !known {
		return true, 0, -1 // unknown features are unrestricted
	}

	jwt := getJWT()
	plan := getPlanFromJWT(jwt)
	if plan == "pro" || plan == "team" {
		return true, 0, -1 // unlimited
	}

	featureUsageMu.Lock()
	defer featureUsageMu.Unlock()

	d := loadFeatureUsage()
	if d[userID] == nil {
		d[userID] = map[string]int{}
	}
	used = d[userID][feature]
	if used >= limit {
		return false, used, limit
	}
	return true, used, limit
}

// incrementUsage bumps the counter for a feature.
func incrementUsage(userID, feature string) {
	featureUsageMu.Lock()
	defer featureUsageMu.Unlock()
	d := loadFeatureUsage()
	if d[userID] == nil {
		d[userID] = map[string]int{}
	}
	d[userID][feature]++
	saveFeatureUsage(d)
}

// upgradeRequiredResponse builds the CommandResponse for a paywall hit.
func upgradeRequiredResponse(feature string, used, limit int) CommandResponse {
	featureLabel := map[string]string{
		"stock_analysis":  "주식 분석",
		"medical_search":  "의료 정보 검색",
		"contract_review": "계약서 검토",
		"legal_search":    "법률 검색",
		"content_script":  "콘텐츠 스크립트",
		"workflow_run":    "워크플로우 실행",
		"ai_request":      "AI 요청",
	}
	label := featureLabel[feature]
	if label == "" {
		label = feature
	}
	var msg string
	if limit == 0 {
		msg = fmt.Sprintf("오늘 %s 사용량을 모두 소진했습니다. Pro로 업그레이드하면 하루 2,000회 사용할 수 있어요.", label)
	} else {
		msg = fmt.Sprintf("%s은(는) 오늘 %d/%d회 사용했습니다. Pro로 업그레이드하면 무제한으로 사용할 수 있어요.", label, used, limit)
	}
	return CommandResponse{
		Success:         false,
		Message:         msg,
		Action:          "upgrade_required",
		Duration:        "0.00s",
		UpgradeRequired: true,
		UsedCount:       used,
		LimitCount:      limit,
		FeatureName:     feature,
	}
}

// ── HTTP 핸들러: AI 요청 사용량 조회 + 증가 ─────────────────────
// GET  /api/usage/ai        → { used, limit, allowed, reset_at }
// POST /api/usage/ai        → 카운트 증가 후 동일 형식 반환
func handleUsageAI(w http.ResponseWriter, r *http.Request) {
	jwt := getJWT()
	userID := extractSubFromJWT(jwt)
	if userID == "" {
		userID = getMachineID()
	}

	plan := getPlanFromJWT(jwt)
	if r.Method == http.MethodPost {
		if plan != "pro" && plan != "team" {
			incrementUsage(userID, "ai_request")
		}
	}

	allowed, used, limit := checkUsageLimit(userID, "ai_request")
	if plan == "pro" || plan == "team" {
		allowed = true
		used = 0
		limit = -1
	}

	reset := time.Now().Truncate(24 * time.Hour).Add(24 * time.Hour)
	json200(w, map[string]any{
		"used":     used,
		"limit":    limit,
		"allowed":  allowed,
		"plan":     plan,
		"reset_at": reset.Format(time.RFC3339),
	})
}

// extractSubFromJWT parses JWT payload and returns the "sub" claim.
func extractSubFromJWT(token string) string {
	if token == "" {
		return ""
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return ""
	}
	payload := parts[1]
	switch len(payload) % 4 {
	case 2:
		payload += "=="
	case 3:
		payload += "="
	}
	raw, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return ""
	}
	var claims map[string]any
	if err := json.Unmarshal(raw, &claims); err != nil {
		return ""
	}
	if sub, ok := claims["sub"].(string); ok {
		return sub
	}
	return ""
}
