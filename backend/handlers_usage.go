package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
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
	jwt := getJWT()
	plan := getPlanFromJWT(jwt)
	uid, _ := resolveUserID(r.URL.Query().Get("user_id"))

	limits, ok := planLimits[plan]
	if !ok {
		limits = planLimits["free"]
	}

	_, aiUsed, aiLimit := checkUsageLimit(uid, "ai_request")
	reset := time.Now().Truncate(24 * time.Hour).Add(24 * time.Hour)

	featureStatus := map[string]any{}
	for feature, lim := range limits {
		_, used, _ := checkUsageLimit(uid, feature)
		featureStatus[feature] = map[string]any{
			"used": used, "limit": lim, "left": max(lim-used, 0),
		}
	}

	json200(w, map[string]any{
		"user_id":  uid,
		"plan":     plan,
		"date":     time.Now().Format("2006-01-02"),
		"reset_at": reset.Format(time.RFC3339),
		"ai_request": map[string]any{
			"used": aiUsed, "limit": aiLimit, "left": max(aiLimit-aiUsed, 0),
		},
		"features": featureStatus,
	})
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
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
//  Feature-based daily usage limits — per plan
//  free: 가입 없이 사용 가능한 기본 한도
//  pro:  월 구독 사용자 (API 비용을 요금제로 충당)
//  team: 팀 요금제 사용자
// ══════════════════════════════════════════════════════════════════

var planLimits = map[string]map[string]int{
	"free": {
		"ai_request":      15,
		"stock_analysis":  3,
		"medical_search":  3,
		"contract_review": 1,
		"legal_search":    3,
		"content_script":  5,
		"workflow_run":    10,
	},
	"pro": {
		"ai_request":      200,
		"stock_analysis":  50,
		"medical_search":  50,
		"contract_review": 20,
		"legal_search":    50,
		"content_script":  100,
		"workflow_run":    200,
	},
	"team": {
		"ai_request":      1000,
		"stock_analysis":  200,
		"medical_search":  200,
		"contract_review": 100,
		"legal_search":    200,
		"content_script":  500,
		"workflow_run":    1000,
	},
	"admin": {
		"ai_request":      99999,
		"stock_analysis":  99999,
		"medical_search":  99999,
		"contract_review": 99999,
		"legal_search":    99999,
		"content_script":  99999,
		"workflow_run":    99999,
	},
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
	// generate simple UUID v4 (crypto/rand works on all platforms including Windows)
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "default-machine"
	}
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

	// 1. 최상위 plan 클레임
	if p, ok := claims["plan"].(string); ok && p != "" {
		return normalizePlan(p)
	}
	// 2. app_metadata.plan (Supabase 관리자 설정)
	if am, ok := claims["app_metadata"].(map[string]any); ok {
		if p, ok := am["plan"].(string); ok && p != "" {
			return normalizePlan(p)
		}
	}
	// 3. user_metadata.plan (사용자 프로필)
	if um, ok := claims["user_metadata"].(map[string]any); ok {
		if p, ok := um["plan"].(string); ok && p != "" {
			return normalizePlan(p)
		}
	}
	// 4. role 클레임 (service_role / admin)
	if role, ok := claims["role"].(string); ok {
		if role == "service_role" || role == "admin" {
			return "admin"
		}
	}
	return "free"
}

// normalizePlan maps variant spellings to canonical plan names.
func normalizePlan(p string) string {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case "admin", "administrator", "superadmin", "super_admin":
		return "admin"
	case "team", "business", "enterprise":
		return "team"
	case "pro", "premium", "professional":
		return "pro"
	default:
		return p
	}
}

// resolveUserID: JWT sub 우선, 없으면 machine ID 폴백
func resolveUserID(fallback string) (subID string, isAuth bool) {
	if sub := extractSubFromJWT(getJWT()); sub != "" {
		return sub, true
	}
	if fallback != "" {
		return fallback, false
	}
	return getMachineID(), false
}

// checkUsageLimit returns whether the user can use the feature today,
// and the current used/limit counts.
// Supabase가 연결되어 있으면 서버 카운트 우선, 오프라인이면 로컬 폴백.
func checkUsageLimit(userID, feature string) (allowed bool, used int, limit int) {
	jwt := getJWT()
	plan := getPlanFromJWT(jwt)

	limits, planKnown := planLimits[plan]
	if !planKnown {
		limits = planLimits["free"]
	}
	limit, known := limits[feature]
	if !known {
		return true, 0, -1
	}

	today := time.Now().Format("2006-01-02")
	uid, isAuth := resolveUserID(userID)

	// Supabase 서버 카운트 조회 (인증된 사용자만)
	if isAuth {
		count, err := supabaseFetchCount(jwt, uid, feature, today)
		if err == nil {
			return count < limit, count, limit
		}
		log.Printf("[Usage] Supabase fetch failed (%v) — local fallback", err)
	}

	// 로컬 폴백 (오프라인 or 미인증)
	featureUsageMu.Lock()
	defer featureUsageMu.Unlock()
	d := loadFeatureUsage()
	if d[uid] == nil {
		d[uid] = map[string]int{}
	}
	used = d[uid][feature]
	return used < limit, used, limit
}

// incrementUsage bumps the counter for a feature.
// Supabase에 원자적으로 기록하고, 로컬에도 즉시 반영 (오프라인 대비).
func incrementUsage(userID, feature string) {
	today := time.Now().Format("2006-01-02")
	uid, isAuth := resolveUserID(userID)
	jwt := getJWT()

	// Supabase 원자적 증가 (비동기 — 요청 블로킹 없음)
	if isAuth {
		go func() {
			if err := supabaseIncrementRPC(jwt, uid, feature, today); err != nil {
				log.Printf("[Usage] Supabase increment failed (%v) — local only", err)
			}
		}()
	}

	// 로컬 즉시 반영 (오프라인 폴백 + 즉각적인 UI 반영)
	featureUsageMu.Lock()
	defer featureUsageMu.Unlock()
	d := loadFeatureUsage()
	if d[uid] == nil {
		d[uid] = map[string]int{}
	}
	d[uid][feature]++
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
	proLimit := planLimits["pro"][feature]
	teamLimit := planLimits["team"][feature]
	msg := fmt.Sprintf(
		"오늘 %s을(를) %d/%d회 사용했어요.\n\n"+
			"• Free  : %d회/일\n"+
			"• Pro   : %d회/일\n"+
			"• Team  : %d회/일\n\n"+
			"업그레이드하면 더 많이 사용할 수 있어요 🚀",
		label, used, limit,
		limit, proLimit, teamLimit,
	)
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
		incrementUsage(userID, "ai_request")
	}

	allowed, used, limit := checkUsageLimit(userID, "ai_request")

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
