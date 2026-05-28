//go:build windows

package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ── AES-256-GCM 암호화 헬퍼 ──────────────────────────────────────────

// deriveEncKey: 머신 ID에서 32바이트 AES-256 키 유도 (SHA-256)
func deriveEncKey() []byte {
	mid := getMachineID()
	h := sha256.Sum256([]byte("nexus-api-keys-v1:" + mid))
	return h[:]
}

// encryptJSON: JSON 바이트 → AES-256-GCM 암호화 → hex 인코딩 + "enc:" 마커
func encryptJSON(plain []byte) ([]byte, error) {
	key := deriveEncKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return nil, err
	}
	sealed := gcm.Seal(nonce, nonce, plain, nil)
	return []byte("enc:" + hex.EncodeToString(sealed)), nil
}

// decryptJSON: "enc:" + hex → AES-256-GCM 복호화 → 원본 JSON
// "enc:" 마커가 없으면 평문 JSON으로 간주 (하위 호환)
func decryptJSON(data []byte) ([]byte, error) {
	s := string(data)
	if !strings.HasPrefix(s, "enc:") {
		return data, nil // 평문 (구버전 호환)
	}
	key := deriveEncKey()
	cipherHex := s[4:]
	cipherBytes, err := hex.DecodeString(cipherHex)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(cipherBytes) < gcm.NonceSize() {
		return nil, fmt.Errorf("암호화 데이터가 너무 짧습니다")
	}
	nonce, ciphertext := cipherBytes[:gcm.NonceSize()], cipherBytes[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("복호화 실패 (다른 기기의 파일이거나 손상됨): %w", err)
	}
	return plain, nil
}

// ── Types ──────────────────────────────────────────────────────────

type APIKey struct {
	ID             string   `json:"id"`
	Key            string   `json:"key"`
	Name           string   `json:"name"`
	Plan           string   `json:"plan"`
	OwnerID        string   `json:"owner_id"`
	CreatedAt      string   `json:"created_at"`
	LastUsedAt     string   `json:"last_used_at"`
	MonthlyLimit   int      `json:"monthly_limit"`
	UsedThisMonth  int      `json:"used_this_month"`
	Endpoints      []string `json:"endpoints"`
	Active         bool     `json:"active"`
}

type APIUsageLog struct {
	KeyID     string `json:"key_id"`
	Endpoint  string `json:"endpoint"`
	Timestamp string `json:"timestamp"`
	Success   bool   `json:"success"`
}

type PlanInfo struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Price        int      `json:"price"`
	MonthlyLimit int      `json:"monthly_limit"`
	Endpoints    []string `json:"endpoints"`
	Description  string   `json:"description"`
}

// ── Plans ──────────────────────────────────────────────────────────

var coreEndpoints = []string{"/v1/chat", "/v1/search"}
var allEndpoints = []string{"/v1/chat", "/v1/search", "/v1/stock", "/v1/legal", "/v1/medical"}

var plans = []PlanInfo{
	{ID: "starter", Name: "Starter", Price: 49, MonthlyLimit: 1000, Endpoints: coreEndpoints, Description: "월 1,000 호출, 핵심 엔드포인트"},
	{ID: "growth", Name: "Growth", Price: 149, MonthlyLimit: 10000, Endpoints: allEndpoints, Description: "월 10,000 호출, 모든 엔드포인트"},
	{ID: "enterprise", Name: "Enterprise", Price: 499, MonthlyLimit: -1, Endpoints: allEndpoints, Description: "무제한, 우선 처리"},
}

func planInfo(id string) *PlanInfo {
	for i := range plans {
		if plans[i].ID == id {
			return &plans[i]
		}
	}
	return nil
}

// ── Storage ────────────────────────────────────────────────────────

func apiKeysPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".nexus", "api_keys.json")
}

func loadAPIKeys() []APIKey {
	data, err := os.ReadFile(apiKeysPath())
	if err != nil {
		return []APIKey{}
	}
	// 복호화 시도 (평문 파일은 그대로 통과)
	plain, err := decryptJSON(data)
	if err != nil {
		// 복호화 실패 시 비어 있는 슬라이스 반환 (손상 파일 보호)
		return []APIKey{}
	}
	var keys []APIKey
	json.Unmarshal(plain, &keys)
	return keys
}

func saveAPIKeys(keys []APIKey) error {
	path := apiKeysPath()
	os.MkdirAll(filepath.Dir(path), 0755)
	plain, _ := json.MarshalIndent(keys, "", "  ")
	// AES-256-GCM 암호화 후 저장
	enc, err := encryptJSON(plain)
	if err != nil {
		// 암호화 실패 시 평문으로 폴백 (기능 중단 방지)
		return os.WriteFile(path, plain, 0600)
	}
	return os.WriteFile(path, enc, 0600)
}

// ── Validation ─────────────────────────────────────────────────────

func validateAPIKey(key string) (*APIKey, error) {
	if !strings.HasPrefix(key, "nxs_live_") {
		return nil, fmt.Errorf("invalid key format")
	}
	keys := loadAPIKeys()
	for i := range keys {
		if keys[i].Key == key && keys[i].Active {
			return &keys[i], nil
		}
	}
	return nil, fmt.Errorf("key not found or inactive")
}

func checkAndIncrementUsage(keyID string, endpoint string) error {
	keys := loadAPIKeys()
	for i := range keys {
		if keys[i].ID == keyID {
			if keys[i].MonthlyLimit != -1 && keys[i].UsedThisMonth >= keys[i].MonthlyLimit {
				return fmt.Errorf("monthly limit exceeded")
			}
			// Check endpoint allowed
			allowed := false
			for _, ep := range keys[i].Endpoints {
				if ep == endpoint {
					allowed = true
					break
				}
			}
			if !allowed {
				return fmt.Errorf("endpoint not allowed for this plan")
			}
			keys[i].UsedThisMonth++
			keys[i].LastUsedAt = time.Now().UTC().Format(time.RFC3339)
			saveAPIKeys(keys)
			return nil
		}
	}
	return fmt.Errorf("key not found")
}

// ── Management Handlers ────────────────────────────────────────────

func handleEnterpriseListKeys(w http.ResponseWriter, r *http.Request) {
	keys := loadAPIKeys()
	json200(w,map[string]any{"ok": true, "keys": keys})
}

func handleEnterpriseCreateKey(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
		Plan string `json:"plan"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.Name == "" || body.Plan == "" {
		http.Error(w, `{"ok":false,"error":"name and plan required"}`, 400)
		return
	}
	p := planInfo(body.Plan)
	if p == nil {
		http.Error(w, `{"ok":false,"error":"invalid plan"}`, 400)
		return
	}

	// Generate key
	buf := make([]byte, 16)
	rand.Read(buf)
	keyStr := "nxs_live_" + hex.EncodeToString(buf)

	idBuf := make([]byte, 8)
	rand.Read(idBuf)

	key := APIKey{
		ID:            hex.EncodeToString(idBuf),
		Key:           keyStr,
		Name:          body.Name,
		Plan:          body.Plan,
		OwnerID:       "local",
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
		LastUsedAt:    "",
		MonthlyLimit:  p.MonthlyLimit,
		UsedThisMonth: 0,
		Endpoints:     p.Endpoints,
		Active:        true,
	}

	keys := loadAPIKeys()
	keys = append(keys, key)
	saveAPIKeys(keys)
	json200(w,map[string]any{"ok": true, "key": key})
}

func handleEnterpriseRevokeKey(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	keys := loadAPIKeys()
	found := false
	for i := range keys {
		if keys[i].ID == id {
			keys[i].Active = false
			found = true
			break
		}
	}
	if !found {
		http.Error(w, `{"ok":false,"error":"not found"}`, 404)
		return
	}
	saveAPIKeys(keys)
	json200(w,map[string]any{"ok": true})
}

func handleEnterpriseKeyUsage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	keys := loadAPIKeys()
	for _, k := range keys {
		if k.ID == id {
			p := planInfo(k.Plan)
			resp := map[string]any{
				"ok":              true,
				"key_id":          k.ID,
				"plan":            k.Plan,
				"monthly_limit":   k.MonthlyLimit,
				"used_this_month": k.UsedThisMonth,
				"last_used_at":    k.LastUsedAt,
			}
			if p != nil {
				resp["plan_price"] = p.Price
			}
			json200(w,resp)
			return
		}
	}
	http.Error(w, `{"ok":false,"error":"not found"}`, 404)
}

func handleEnterprisePlans(w http.ResponseWriter, r *http.Request) {
	json200(w,map[string]any{"ok": true, "plans": plans})
}

// ── External API Handlers (/v1/*) ──────────────────────────────────


func handleV1Chat(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Message string `json:"message"`
		APIKey  string `json:"api_key"`
		Lang    string `json:"lang"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	key := body.APIKey
	if key == "" {
		key = r.Header.Get("X-API-Key")
	}
	apiKey, err := validateAPIKey(key)
	if err != nil {
		http.Error(w, `{"ok":false,"error":"unauthorized"}`, 401)
		return
	}
	if err := checkAndIncrementUsage(apiKey.ID, "/v1/chat"); err != nil {
		http.Error(w, fmt.Sprintf(`{"ok":false,"error":"%s"}`, err.Error()), 429)
		return
	}

	if body.Message == "" {
		http.Error(w, `{"ok":false,"error":"message required"}`, 400)
		return
	}
	lang := body.Lang
	if lang == "" {
		lang = "ko"
	}
	// dispatchAction(action, params, original, gKey, lang, history)
	result, msg := dispatchAction("web_search", map[string]any{"query": body.Message}, body.Message, llmGroqKey, lang, nil)
	json200(w,map[string]any{"ok": true, "result": result, "message": msg})
}

func handleV1Search(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Query  string `json:"query"`
		APIKey string `json:"api_key"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	key := body.APIKey
	if key == "" {
		key = r.Header.Get("X-API-Key")
	}
	apiKey, err := validateAPIKey(key)
	if err != nil {
		http.Error(w, `{"ok":false,"error":"unauthorized"}`, 401)
		return
	}
	if err := checkAndIncrementUsage(apiKey.ID, "/v1/search"); err != nil {
		http.Error(w, fmt.Sprintf(`{"ok":false,"error":"%s"}`, err.Error()), 429)
		return
	}
	if body.Query == "" {
		http.Error(w, `{"ok":false,"error":"query required"}`, 400)
		return
	}
	result, _ := tavilySearch(llmTavilyKey, body.Query, 10)
	json200(w, map[string]any{"ok": true, "result": result})
}

func handleV1Stock(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Ticker string `json:"ticker"`
		Query  string `json:"query"`
		APIKey string `json:"api_key"`
		Lang   string `json:"lang"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	key := body.APIKey
	if key == "" {
		key = r.Header.Get("X-API-Key")
	}
	apiKey, err := validateAPIKey(key)
	if err != nil {
		http.Error(w, `{"ok":false,"error":"unauthorized"}`, 401)
		return
	}
	if err := checkAndIncrementUsage(apiKey.ID, "/v1/stock"); err != nil {
		http.Error(w, fmt.Sprintf(`{"ok":false,"error":"%s"}`, err.Error()), 429)
		return
	}
	lang := body.Lang
	if lang == "" {
		lang = "ko"
	}
	data, summary := stockAnalysisLogic(body.Ticker, body.Query, lang)
	json200(w,map[string]any{"ok": true, "data": data, "summary": summary})
}

func handleV1Legal(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Query      string `json:"query"`
		SearchType string `json:"search_type"`
		APIKey     string `json:"api_key"`
		Lang       string `json:"lang"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	key := body.APIKey
	if key == "" {
		key = r.Header.Get("X-API-Key")
	}
	apiKey, err := validateAPIKey(key)
	if err != nil {
		http.Error(w, `{"ok":false,"error":"unauthorized"}`, 401)
		return
	}
	if err := checkAndIncrementUsage(apiKey.ID, "/v1/legal"); err != nil {
		http.Error(w, fmt.Sprintf(`{"ok":false,"error":"%s"}`, err.Error()), 429)
		return
	}
	lang := body.Lang
	if lang == "" {
		lang = "ko"
	}
	if body.SearchType == "" {
		body.SearchType = "general"
	}
	data, summary := legalSearchLogic(body.Query, body.SearchType, lang)
	json200(w,map[string]any{"ok": true, "data": data, "summary": summary})
}

func handleV1Medical(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Query      string `json:"query"`
		SearchType string `json:"search_type"`
		APIKey     string `json:"api_key"`
		Lang       string `json:"lang"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	key := body.APIKey
	if key == "" {
		key = r.Header.Get("X-API-Key")
	}
	apiKey, err := validateAPIKey(key)
	if err != nil {
		http.Error(w, `{"ok":false,"error":"unauthorized"}`, 401)
		return
	}
	if err := checkAndIncrementUsage(apiKey.ID, "/v1/medical"); err != nil {
		http.Error(w, fmt.Sprintf(`{"ok":false,"error":"%s"}`, err.Error()), 429)
		return
	}
	lang := body.Lang
	if lang == "" {
		lang = "ko"
	}
	if body.SearchType == "" {
		body.SearchType = "general"
	}
	data, summary := medicalSearchLogic(body.Query, body.SearchType, lang)
	json200(w,map[string]any{"ok": true, "data": data, "summary": summary})
}
