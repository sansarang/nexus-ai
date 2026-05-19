//go:build !windows

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
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
