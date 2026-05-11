//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ══════════════════════════════════════════════════════════════════
//  Smart Home — Home Assistant REST API 연동
// ══════════════════════════════════════════════════════════════════

type SmartHomeConfig struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}

func smartHomeConfigPath() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = os.TempDir()
	}
	dir := filepath.Join(appData, "Nexus")
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "smarthome.json")
}

func loadSmartHomeConfig() (SmartHomeConfig, error) {
	data, err := os.ReadFile(smartHomeConfigPath())
	if err != nil {
		return SmartHomeConfig{}, err
	}
	var cfg SmartHomeConfig
	err = json.Unmarshal(data, &cfg)
	return cfg, err
}

func saveSmartHomeConfig(cfg SmartHomeConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(smartHomeConfigPath(), data, 0644)
}

// GET|POST /api/smarthome/config
func handleSmartHomeConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		cfg, err := loadSmartHomeConfig()
		if err != nil {
			json200(w, map[string]interface{}{"success": false, "message": "설정 없음"})
			return
		}
		// 토큰 마스킹
		masked := cfg.Token
		if len(masked) > 8 {
			masked = masked[:4] + strings.Repeat("*", len(masked)-8) + masked[len(masked)-4:]
		} else if len(masked) > 0 {
			masked = strings.Repeat("*", len(masked))
		}
		json200(w, map[string]interface{}{
			"success": true,
			"url":     cfg.URL,
			"token":   masked,
		})
		return
	}

	// POST
	var req SmartHomeConfig
	json.NewDecoder(r.Body).Decode(&req)
	if req.URL == "" {
		json200(w, map[string]interface{}{"success": false, "message": "url이 필요해요"})
		return
	}
	if err := saveSmartHomeConfig(req); err != nil {
		json200(w, map[string]interface{}{"success": false, "message": "설정 저장 실패"})
		return
	}
	json200(w, map[string]interface{}{"success": true, "message": "스마트홈 설정 저장 완료"})
}

// GET /api/smarthome/devices
func handleSmartHomeDevices(w http.ResponseWriter, r *http.Request) {
	cfg, err := loadSmartHomeConfig()
	if err != nil {
		json200(w, map[string]interface{}{"success": false, "message": "스마트홈 설정이 없어요. 먼저 설정해 주세요."})
		return
	}

	url := strings.TrimRight(cfg.URL, "/") + "/api/states"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+cfg.Token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		json200(w, map[string]interface{}{"success": false, "message": "Home Assistant 연결 실패: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var states []map[string]interface{}
	json.Unmarshal(body, &states)

	allowedDomains := map[string]bool{
		"light":   true,
		"switch":  true,
		"climate": true,
		"sensor":  true,
	}

	type Device struct {
		ID         string                 `json:"id"`
		Name       string                 `json:"name"`
		State      string                 `json:"state"`
		Domain     string                 `json:"domain"`
		Attributes map[string]interface{} `json:"attributes"`
	}

	var devices []Device
	for _, s := range states {
		entityID, _ := s["entity_id"].(string)
		parts := strings.SplitN(entityID, ".", 2)
		if len(parts) != 2 {
			continue
		}
		domain := parts[0]
		if !allowedDomains[domain] {
			continue
		}
		state, _ := s["state"].(string)
		attrs, _ := s["attributes"].(map[string]interface{})
		name := entityID
		if attrs != nil {
			if fn, ok := attrs["friendly_name"].(string); ok {
				name = fn
			}
		}
		devices = append(devices, Device{
			ID:         entityID,
			Name:       name,
			State:      state,
			Domain:     domain,
			Attributes: attrs,
		})
	}

	json200(w, map[string]interface{}{
		"success": true,
		"devices": devices,
		"total":   len(devices),
		"message": fmt.Sprintf("디바이스 %d개 발견", len(devices)),
	})
}

// POST /api/smarthome/control — body: {entity_id, action, params?}
func handleSmartHomeControl(w http.ResponseWriter, r *http.Request) {
	cfg, err := loadSmartHomeConfig()
	if err != nil {
		json200(w, map[string]interface{}{"success": false, "message": "스마트홈 설정이 없어요"})
		return
	}

	var req struct {
		EntityID string                 `json:"entity_id"`
		Action   string                 `json:"action"`
		Params   map[string]interface{} `json:"params,omitempty"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.EntityID == "" || req.Action == "" {
		json200(w, map[string]interface{}{"success": false, "message": "entity_id와 action이 필요해요"})
		return
	}

	parts := strings.SplitN(req.EntityID, ".", 2)
	if len(parts) != 2 {
		json200(w, map[string]interface{}{"success": false, "message": "유효하지 않은 entity_id"})
		return
	}
	domain := parts[0]

	payload := map[string]interface{}{"entity_id": req.EntityID}
	if req.Params != nil {
		for k, v := range req.Params {
			payload[k] = v
		}
	}

	payloadBytes, _ := json.Marshal(payload)
	serviceURL := strings.TrimRight(cfg.URL, "/") + "/api/services/" + domain + "/" + req.Action

	httpReq, _ := http.NewRequest("POST", serviceURL, strings.NewReader(string(payloadBytes)))
	httpReq.Header.Set("Authorization", "Bearer "+cfg.Token)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		json200(w, map[string]interface{}{"success": false, "message": "제어 실패: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	json200(w, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("%s → %s 명령 전송 완료", req.EntityID, req.Action),
	})
}
