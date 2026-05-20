package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
)

// ── Types ──────────────────────────────────────────────────────────

type VerticalConfig struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Theme          string   `json:"theme"`
	Logo           string   `json:"logo"`
	DefaultPersona string   `json:"default_persona"`
	Features       []string `json:"features"`
	WelcomeMsg     string   `json:"welcome_msg"`
	Watermark      string   `json:"watermark"`
}

// ── Presets ────────────────────────────────────────────────────────

var verticalPresets = []VerticalConfig{
	{
		ID:             "general",
		Name:           "Nexus AI",
		Theme:          "#4f7ef7",
		Logo:           "",
		DefaultPersona: "general",
		Features:       []string{"chat", "search", "stock", "legal", "medical", "browser", "calendar", "files"},
		WelcomeMsg:     "안녕하세요! Nexus AI입니다. 무엇을 도와드릴까요?",
		Watermark:      "Powered by Nexus AI",
	},
	{
		ID:             "legal",
		Name:           "Nexus for 법무사",
		Theme:          "#7c3aed",
		Logo:           "",
		DefaultPersona: "legal",
		Features:       []string{"chat", "legal", "search", "files"},
		WelcomeMsg:     "안녕하세요! 법률 전문 AI 어시스턴트입니다. 법률 상담을 도와드리겠습니다.",
		Watermark:      "Powered by Nexus for 법무사",
	},
	{
		ID:             "medical",
		Name:           "Nexus for 의원",
		Theme:          "#0891b2",
		Logo:           "",
		DefaultPersona: "medical",
		Features:       []string{"chat", "medical", "search", "files", "calendar"},
		WelcomeMsg:     "안녕하세요! 의료 전문 AI 어시스턴트입니다. 의학 정보를 안내해드리겠습니다.",
		Watermark:      "Powered by Nexus for 의원",
	},
	{
		ID:             "finance",
		Name:           "Nexus for 투자사",
		Theme:          "#059669",
		Logo:           "",
		DefaultPersona: "investor",
		Features:       []string{"chat", "stock", "search", "files", "browser"},
		WelcomeMsg:     "안녕하세요! 금융 전문 AI 어시스턴트입니다. 투자 분석을 도와드리겠습니다.",
		Watermark:      "Powered by Nexus for 투자사",
	},
	{
		ID:             "content",
		Name:           "Nexus for 크리에이터",
		Theme:          "#dc2626",
		Logo:           "",
		DefaultPersona: "creator",
		Features:       []string{"chat", "search", "browser", "files", "calendar"},
		WelcomeMsg:     "안녕하세요! 크리에이터 전문 AI 어시스턴트입니다. 콘텐츠 제작을 도와드리겠습니다.",
		Watermark:      "Powered by Nexus for 크리에이터",
	},
}

// ── Storage ────────────────────────────────────────────────────────

func verticalConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".nexus", "vertical.json")
}

func loadVerticalConfig() VerticalConfig {
	data, err := os.ReadFile(verticalConfigPath())
	if err != nil {
		return verticalPresets[0] // default: general
	}
	var cfg VerticalConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return verticalPresets[0]
	}
	return cfg
}

func saveVerticalConfig(cfg VerticalConfig) error {
	path := verticalConfigPath()
	os.MkdirAll(filepath.Dir(path), 0755)
	data, _ := json.MarshalIndent(cfg, "", "  ")
	return os.WriteFile(path, data, 0600)
}

// ── Handlers ───────────────────────────────────────────────────────

func handleVerticalGetConfig(w http.ResponseWriter, r *http.Request) {
	cfg := loadVerticalConfig()
	json200(w, map[string]any{"ok": true, "config": cfg})
}

func handleVerticalSetConfig(w http.ResponseWriter, r *http.Request) {
	var cfg VerticalConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, `{"ok":false,"error":"invalid body"}`, 400)
		return
	}
	if cfg.ID == "" {
		http.Error(w, `{"ok":false,"error":"id required"}`, 400)
		return
	}
	if err := saveVerticalConfig(cfg); err != nil {
		http.Error(w, `{"ok":false,"error":"failed to save"}`, 500)
		return
	}
	json200(w, map[string]any{"ok": true, "config": cfg})
}

func handleVerticalPresets(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"ok": true, "presets": verticalPresets})
}
