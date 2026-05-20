//go:build windows

package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ── 마켓플레이스 데이터 구조 ─────────────────────────────────────

type MarketPreset struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
	Author      string  `json:"author"`
	AuthorID    string  `json:"author_id"`
	Price       float64 `json:"price"`
	Currency    string  `json:"currency"`
	Steps       []any   `json:"steps"`
	Tags        []string `json:"tags"`
	Rating      float64 `json:"rating"`
	Downloads   int     `json:"downloads"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
	Preview     string  `json:"preview"`
	IsOwned     bool    `json:"is_owned"`
	IsFree      bool    `json:"is_free"`
}

type PurchaseRecord struct {
	PresetID    string  `json:"preset_id"`
	UserID      string  `json:"user_id"`
	PaidAmount  float64 `json:"paid_amount"`
	Commission  float64 `json:"commission"` // 30%
	PurchasedAt string  `json:"purchased_at"`
}

// ── 파일 경로 ────────────────────────────────────────────────────

func randomID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func marketplaceDir() string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".nexus", "marketplace")
	os.MkdirAll(dir, 0755)
	return dir
}

func presetsPath() string   { return filepath.Join(marketplaceDir(), "presets.json") }
func purchasesPath() string { return filepath.Join(marketplaceDir(), "purchases.json") }
func myPresetsPath() string { return filepath.Join(marketplaceDir(), "my_presets.json") }

// ── 파일 잠금 ────────────────────────────────────────────────────

var marketMu sync.Mutex

// ── 파일 I/O ────────────────────────────────────────────────────

func loadPresets() []MarketPreset {
	data, err := os.ReadFile(presetsPath())
	if err != nil {
		return initDefaultPresets()
	}
	var presets []MarketPreset
	if err := json.Unmarshal(data, &presets); err != nil {
		return initDefaultPresets()
	}
	if len(presets) == 0 {
		return initDefaultPresets()
	}
	return presets
}

func savePresets(presets []MarketPreset) error {
	data, _ := json.MarshalIndent(presets, "", "  ")
	return os.WriteFile(presetsPath(), data, 0644)
}

func loadPurchases() []PurchaseRecord {
	data, err := os.ReadFile(purchasesPath())
	if err != nil {
		return []PurchaseRecord{}
	}
	var purchases []PurchaseRecord
	json.Unmarshal(data, &purchases)
	return purchases
}

func savePurchases(purchases []PurchaseRecord) error {
	data, _ := json.MarshalIndent(purchases, "", "  ")
	return os.WriteFile(purchasesPath(), data, 0644)
}

func loadMyPresets() []string {
	data, err := os.ReadFile(myPresetsPath())
	if err != nil {
		return []string{}
	}
	var ids []string
	json.Unmarshal(data, &ids)
	return ids
}

func saveMyPresets(ids []string) error {
	data, _ := json.MarshalIndent(ids, "", "  ")
	return os.WriteFile(myPresetsPath(), data, 0644)
}

// ── 기본 샘플 프리셋 ────────────────────────────────────────────

func initDefaultPresets() []MarketPreset {
	now := time.Now().Format(time.RFC3339)
	presets := []MarketPreset{
		{
			ID: "mp-001", Name: "주간 업무 보고서 자동화",
			Description: "매주 월요일 아침, 지난 주 업무를 자동으로 정리하여 보고서를 생성합니다.",
			Category: "productivity", Author: "Nexus Team", AuthorID: "nexus-official",
			Price: 0, Currency: "USD", IsFree: true, Rating: 4.7, Downloads: 312,
			Tags: []string{"보고서", "자동화", "생산성"},
			Preview: "📊 주간 업무 보고서 생성 완료: 총 15개 작업, 완료율 87%...",
			Steps: []any{
				map[string]any{"action": "briefing", "params": map[string]any{"type": "weekly_summary"}},
				map[string]any{"action": "summarize", "params": map[string]any{"format": "report"}},
			},
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "mp-002", Name: "이메일 요약 + 답장 초안",
			Description: "받은 이메일을 AI가 요약하고 적절한 답장 초안을 자동으로 작성합니다.",
			Category: "productivity", Author: "워크플로우 Pro", AuthorID: "wf-pro-001",
			Price: 2.99, Currency: "USD", IsFree: false, Rating: 4.8, Downloads: 189,
			Tags: []string{"이메일", "답장", "자동화"},
			Preview: "📧 3개 중요 이메일 요약 완료. 답장 초안 준비됨...",
			Steps: []any{
				map[string]any{"action": "web_search", "params": map[string]any{"query": "email inbox"}},
				map[string]any{"action": "summarize", "params": map[string]any{"type": "email"}},
				map[string]any{"action": "content_script", "params": map[string]any{"mode": "email_reply"}},
			},
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "mp-003", Name: "회의록 자동 정리",
			Description: "회의 내용을 음성 또는 텍스트로 입력하면 체계적인 회의록으로 자동 변환합니다.",
			Category: "productivity", Author: "Nexus Team", AuthorID: "nexus-official",
			Price: 0, Currency: "USD", IsFree: true, Rating: 4.6, Downloads: 445,
			Tags: []string{"회의록", "정리", "문서"},
			Preview: "📝 회의록 작성 완료: 참석자 5명, 결정사항 3건, 액션아이템 7개...",
			Steps: []any{
				map[string]any{"action": "summarize", "params": map[string]any{"type": "meeting"}},
				map[string]any{"action": "briefing", "params": map[string]any{"format": "minutes"}},
			},
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "mp-004", Name: "포트폴리오 일일 리포트",
			Description: "보유 주식 및 ETF 포트폴리오의 일일 수익률, 리스크, 뉴스를 자동으로 분석합니다.",
			Category: "finance", Author: "투자분석 AI", AuthorID: "invest-ai-001",
			Price: 4.99, Currency: "USD", IsFree: false, Rating: 4.8, Downloads: 234,
			Tags: []string{"주식", "포트폴리오", "투자", "리포트"},
			Preview: "📈 오늘 포트폴리오: +2.3% / 삼성전자 +1.2%, NVDA +4.5%...",
			Steps: []any{
				map[string]any{"action": "stock_analysis", "params": map[string]any{"type": "portfolio_daily"}},
				map[string]any{"action": "web_search", "params": map[string]any{"query": "market news today"}},
				map[string]any{"action": "briefing", "params": map[string]any{"format": "investment_report"}},
			},
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "mp-005", Name: "부동산 시세 분석 보고서",
			Description: "관심 지역의 부동산 시세, 거래량, 상승/하락 트렌드를 자동으로 분석합니다.",
			Category: "real_estate", Author: "부동산 AI 분석가", AuthorID: "realty-ai-001",
			Price: 4.99, Currency: "USD", IsFree: false, Rating: 4.5, Downloads: 167,
			Tags: []string{"부동산", "시세", "분석"},
			Preview: "🏠 강남구 아파트 평균 시세: 22억 (+3.2% MoM)...",
			Steps: []any{
				map[string]any{"action": "web_search", "params": map[string]any{"query": "real estate price"}},
				map[string]any{"action": "summarize", "params": map[string]any{"type": "real_estate"}},
				map[string]any{"action": "briefing", "params": map[string]any{"format": "market_report"}},
			},
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "mp-006", Name: "계약서 리스크 체크리스트",
			Description: "계약서를 업로드하면 AI가 독소조항, 불리한 조건, 누락 항목을 자동으로 검토합니다.",
			Category: "legal", Author: "법률 AI 어시스턴트", AuthorID: "legal-ai-001",
			Price: 9.99, Currency: "USD", IsFree: false, Rating: 4.9, Downloads: 456,
			Tags: []string{"계약서", "법률", "리스크", "검토"},
			Preview: "⚖️ 계약서 검토 완료: 위험 조항 3개 발견, 주의 사항 5건...",
			Steps: []any{
				map[string]any{"action": "legal_search", "params": map[string]any{"type": "contract_review"}},
				map[string]any{"action": "summarize", "params": map[string]any{"type": "legal_risk"}},
				map[string]any{"action": "briefing", "params": map[string]any{"format": "checklist"}},
			},
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "mp-007", Name: "근로계약서 검토 자동화",
			Description: "근로계약서의 임금, 근무시간, 휴가, 해고 조항 등을 근로기준법과 대조하여 검토합니다.",
			Category: "legal", Author: "노동법 AI", AuthorID: "labor-ai-001",
			Price: 7.99, Currency: "USD", IsFree: false, Rating: 4.7, Downloads: 298,
			Tags: []string{"근로계약", "노동법", "검토"},
			Preview: "📋 근로계약서 분석: 연장근로수당 미기재, 퇴직금 조항 불명확...",
			Steps: []any{
				map[string]any{"action": "legal_search", "params": map[string]any{"type": "labor_contract"}},
				map[string]any{"action": "summarize", "params": map[string]any{"type": "contract_issues"}},
			},
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "mp-008", Name: "임상 가이드라인 요약",
			Description: "최신 임상 가이드라인을 검색하고 핵심 내용을 의료진이 바로 활용할 수 있도록 요약합니다.",
			Category: "medical", Author: "의료 AI 리서처", AuthorID: "med-ai-001",
			Price: 4.99, Currency: "USD", IsFree: false, Rating: 4.8, Downloads: 123,
			Tags: []string{"임상", "가이드라인", "의료"},
			Preview: "🏥 고혈압 치료 가이드라인 2024: 1차 목표 혈압 130/80 mmHg...",
			Steps: []any{
				map[string]any{"action": "web_search", "params": map[string]any{"query": "clinical guidelines"}},
				map[string]any{"action": "summarize", "params": map[string]any{"type": "medical_guideline"}},
			},
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "mp-009", Name: "유튜브 콘텐츠 캘린더 30일",
			Description: "채널 주제와 타겟 오디언스를 입력하면 30일치 콘텐츠 캘린더를 자동으로 생성합니다.",
			Category: "content", Author: "콘텐츠 크리에이터 AI", AuthorID: "content-ai-001",
			Price: 2.99, Currency: "USD", IsFree: false, Rating: 4.7, Downloads: 678,
			Tags: []string{"유튜브", "콘텐츠", "캘린더"},
			Preview: "🎬 30일 콘텐츠 캘린더 완성: 트렌드 영상 12개, 튜토리얼 8개...",
			Steps: []any{
				map[string]any{"action": "web_search", "params": map[string]any{"query": "youtube trending"}},
				map[string]any{"action": "content_script", "params": map[string]any{"type": "content_calendar", "days": 30}},
			},
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "mp-010", Name: "SNS 마케팅 콘텐츠 패키지",
			Description: "브랜드 정보를 입력하면 인스타그램, 트위터, 링크드인용 콘텐츠를 한번에 생성합니다.",
			Category: "content", Author: "마케팅 AI", AuthorID: "marketing-ai-001",
			Price: 4.99, Currency: "USD", IsFree: false, Rating: 4.6, Downloads: 389,
			Tags: []string{"SNS", "마케팅", "인스타그램"},
			Preview: "📱 인스타 3개, 트위터 5개, 링크드인 2개 게시물 초안 완성...",
			Steps: []any{
				map[string]any{"action": "web_search", "params": map[string]any{"query": "SNS trends"}},
				map[string]any{"action": "content_script", "params": map[string]any{"type": "social_media_package"}},
				map[string]any{"action": "summarize", "params": map[string]any{"format": "sns_post"}},
			},
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "mp-011", Name: "상권 분석 자동 보고서",
			Description: "특정 상권의 유동인구, 경쟁업체, 업종별 매출 트렌드를 자동으로 분석합니다.",
			Category: "real_estate", Author: "상권분석 AI", AuthorID: "commerce-ai-001",
			Price: 6.99, Currency: "USD", IsFree: false, Rating: 4.5, Downloads: 145,
			Tags: []string{"상권", "부동산", "창업", "분석"},
			Preview: "🏪 홍대 상권 분석: 주말 유동인구 12만명, 카페 업종 포화도 78%...",
			Steps: []any{
				map[string]any{"action": "web_search", "params": map[string]any{"query": "commercial district analysis"}},
				map[string]any{"action": "stock_analysis", "params": map[string]any{"type": "commercial_zone"}},
				map[string]any{"action": "briefing", "params": map[string]any{"format": "commercial_report"}},
			},
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "mp-012", Name: "임대차 계약 검토",
			Description: "임대차 계약서의 보증금, 월세, 관리비, 특약사항 등을 주택임대차보호법과 대조 검토합니다.",
			Category: "real_estate", Author: "부동산 법률 AI", AuthorID: "realty-legal-001",
			Price: 9.99, Currency: "USD", IsFree: false, Rating: 4.9, Downloads: 523,
			Tags: []string{"임대차", "계약", "법률", "부동산"},
			Preview: "🔍 임대차 계약 검토: 확정일자 미기재, 특약사항 임차인 불리 조항 발견...",
			Steps: []any{
				map[string]any{"action": "legal_search", "params": map[string]any{"type": "rental_contract"}},
				map[string]any{"action": "summarize", "params": map[string]any{"type": "lease_issues"}},
				map[string]any{"action": "briefing", "params": map[string]any{"format": "legal_checklist"}},
			},
			CreatedAt: now, UpdatedAt: now,
		},
	}

	// 파일에 저장
	savePresets(presets)
	return presets
}

// ── 핸들러 ───────────────────────────────────────────────────────

// GET /api/marketplace/presets
func handleMarketplaceList(w http.ResponseWriter, r *http.Request) {
	marketMu.Lock()
	presets := loadPresets()
	purchases := loadPurchases()
	marketMu.Unlock()

	userID := getMachineID()
	purchasedIDs := map[string]bool{}
	for _, p := range purchases {
		if p.UserID == userID {
			purchasedIDs[p.PresetID] = true
		}
	}
	myPresets := loadMyPresets()
	myPresetIDs := map[string]bool{}
	for _, id := range myPresets {
		myPresetIDs[id] = true
	}

	// 필터
	category := r.URL.Query().Get("category")
	search := strings.ToLower(r.URL.Query().Get("search"))
	sort := r.URL.Query().Get("sort")

	var result []MarketPreset
	for _, p := range presets {
		if category != "" && category != "all" && p.Category != category {
			continue
		}
		if search != "" {
			if !strings.Contains(strings.ToLower(p.Name), search) &&
				!strings.Contains(strings.ToLower(p.Description), search) {
				continue
			}
		}
		if sort == "free" && !p.IsFree {
			continue
		}
		p.IsOwned = purchasedIDs[p.ID] || myPresetIDs[p.ID] || p.IsFree
		result = append(result, p)
	}

	// 정렬
	if len(result) > 1 {
		switch sort {
		case "newest":
			for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
				result[i], result[j] = result[j], result[i]
			}
		case "price_asc":
			for i := 0; i < len(result)-1; i++ {
				for j := i + 1; j < len(result); j++ {
					if result[j].Price < result[i].Price {
						result[i], result[j] = result[j], result[i]
					}
				}
			}
		default: // popular
			for i := 0; i < len(result)-1; i++ {
				for j := i + 1; j < len(result); j++ {
					if result[j].Downloads > result[i].Downloads {
						result[i], result[j] = result[j], result[i]
					}
				}
			}
		}
	}

	if result == nil {
		result = []MarketPreset{}
	}
	json200(w, map[string]any{"presets": result, "total": len(result)})
}

// GET /api/marketplace/preset/:id
func handleMarketplaceDetail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	marketMu.Lock()
	presets := loadPresets()
	purchases := loadPurchases()
	marketMu.Unlock()

	userID := getMachineID()
	purchasedIDs := map[string]bool{}
	for _, p := range purchases {
		if p.UserID == userID {
			purchasedIDs[p.PresetID] = true
		}
	}

	for _, p := range presets {
		if p.ID == id {
			p.IsOwned = purchasedIDs[p.ID] || p.IsFree
			json200(w, p)
			return
		}
	}
	writeJSON(w, 404, map[string]any{"error": "preset not found"})
}

// POST /api/marketplace/publish
func handleMarketplacePublish(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Category    string   `json:"category"`
		Price       float64  `json:"price"`
		Steps       []any    `json:"steps"`
		Tags        []string `json:"tags"`
		Preview     string   `json:"preview"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	userID := getMachineID()
	now := time.Now().Format(time.RFC3339)

	preset := MarketPreset{
		ID:          "mp-" + randomID(),
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		Author:      "나",
		AuthorID:    userID,
		Price:       req.Price,
		Currency:    "USD",
		IsFree:      req.Price == 0,
		Steps:       req.Steps,
		Tags:        req.Tags,
		Preview:     req.Preview,
		Rating:      0,
		Downloads:   0,
		CreatedAt:   now,
		UpdatedAt:   now,
		IsOwned:     true,
	}

	marketMu.Lock()
	presets := loadPresets()
	presets = append(presets, preset)
	savePresets(presets)
	myIDs := loadMyPresets()
	myIDs = append(myIDs, preset.ID)
	saveMyPresets(myIDs)
	marketMu.Unlock()

	json200(w, map[string]any{"ok": true, "preset_id": preset.ID})
}

// POST /api/marketplace/purchase/:id
func handleMarketplacePurchase(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := getMachineID()

	marketMu.Lock()
	presets := loadPresets()
	purchases := loadPurchases()
	marketMu.Unlock()

	var found *MarketPreset
	for i, p := range presets {
		if p.ID == id {
			found = &presets[i]
			break
		}
	}
	if found == nil {
		writeJSON(w, 404, map[string]any{"error": "preset not found"})
		return
	}

	// 이미 구매했는지 확인
	for _, p := range purchases {
		if p.PresetID == id && p.UserID == userID {
			json200(w, map[string]any{"ok": true, "already_owned": true})
			return
		}
	}

	// 유료 프리셋: Paddle 결제 필요
	if found.Price > 0 {
		json200(w, map[string]any{
			"ok":               true,
			"requires_payment": true,
			"price":            found.Price,
			"preset_id":        id,
		})
		return
	}

	// 무료: 바로 등록
	rec := PurchaseRecord{
		PresetID:    id,
		UserID:      userID,
		PaidAmount:  0,
		Commission:  0,
		PurchasedAt: time.Now().Format(time.RFC3339),
	}
	marketMu.Lock()
	purchases = append(purchases, rec)
	savePurchases(purchases)
	// 다운로드 수 증가
	for i, p := range presets {
		if p.ID == id {
			presets[i].Downloads++
			break
		}
	}
	savePresets(presets)
	marketMu.Unlock()

	json200(w, map[string]any{"ok": true, "preset_id": id})
}

// POST /api/marketplace/purchase/:id/confirm  (Paddle 결제 완료 후 호출)
func handleMarketplacePurchaseConfirm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := getMachineID()

	marketMu.Lock()
	presets := loadPresets()
	purchases := loadPurchases()

	var price float64
	for _, p := range presets {
		if p.ID == id {
			price = p.Price
			break
		}
	}

	rec := PurchaseRecord{
		PresetID:    id,
		UserID:      userID,
		PaidAmount:  price,
		Commission:  price * 0.30,
		PurchasedAt: time.Now().Format(time.RFC3339),
	}
	purchases = append(purchases, rec)
	savePurchases(purchases)

	for i, p := range presets {
		if p.ID == id {
			presets[i].Downloads++
			break
		}
	}
	savePresets(presets)
	marketMu.Unlock()

	json200(w, map[string]any{"ok": true, "preset_id": id})
}

// GET /api/marketplace/my-presets
func handleMarketplaceMyPresets(w http.ResponseWriter, r *http.Request) {
	userID := getMachineID()
	marketMu.Lock()
	presets := loadPresets()
	marketMu.Unlock()

	var result []MarketPreset
	for _, p := range presets {
		if p.AuthorID == userID {
			p.IsOwned = true
			result = append(result, p)
		}
	}
	if result == nil {
		result = []MarketPreset{}
	}
	json200(w, map[string]any{"presets": result})
}

// GET /api/marketplace/purchased
func handleMarketplacePurchased(w http.ResponseWriter, r *http.Request) {
	userID := getMachineID()
	marketMu.Lock()
	presets := loadPresets()
	purchases := loadPurchases()
	marketMu.Unlock()

	purchasedIDs := map[string]bool{}
	for _, p := range purchases {
		if p.UserID == userID {
			purchasedIDs[p.PresetID] = true
		}
	}

	var result []MarketPreset
	for _, p := range presets {
		if purchasedIDs[p.ID] || p.IsFree {
			p.IsOwned = true
			result = append(result, p)
		}
	}
	if result == nil {
		result = []MarketPreset{}
	}
	json200(w, map[string]any{"presets": result})
}

// DELETE /api/marketplace/preset/:id
func handleMarketplaceDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := getMachineID()

	marketMu.Lock()
	presets := loadPresets()
	newPresets := []MarketPreset{}
	found := false
	for _, p := range presets {
		if p.ID == id {
			if p.AuthorID != userID {
				marketMu.Unlock()
				writeJSON(w, 403, map[string]any{"error": "not your preset"})
				return
			}
			found = true
			continue
		}
		newPresets = append(newPresets, p)
	}
	if !found {
		marketMu.Unlock()
		writeJSON(w, 404, map[string]any{"error": "preset not found"})
		return
	}
	savePresets(newPresets)
	myIDs := loadMyPresets()
	newMyIDs := []string{}
	for _, mid := range myIDs {
		if mid != id {
			newMyIDs = append(newMyIDs, mid)
		}
	}
	saveMyPresets(newMyIDs)
	marketMu.Unlock()

	json200(w, map[string]any{"ok": true})
}
