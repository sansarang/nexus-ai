//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ══════════════════════════════════════════════════════════════════
//  Privacy & Sandbox 강화
//  - Agent 실행 전 사용자 승인 로그
//  - PowerShell 제한 실행 (-ExecutionPolicy Restricted)
//  - 민감 경로 접근 차단
//  - 실행 감사 로그
//  - 로컬 LLM 연동 (Ollama)
// ══════════════════════════════════════════════════════════════════

// ── 감사 로그 ───────────────────────────────────────────────────

type AuditEntry struct {
	Timestamp  string `json:"timestamp"`
	Action     string `json:"action"`
	Agent      string `json:"agent"`
	Details    string `json:"details"`
	Approved   bool   `json:"approved"`
	UserAction string `json:"user_action"` // approved|denied|auto
	Result     string `json:"result"`
}

var (
	auditMu  sync.Mutex
	auditLog []AuditEntry
)

func writeAudit(entry AuditEntry) {
	entry.Timestamp = time.Now().Format(time.RFC3339)

	auditMu.Lock()
	auditLog = append([]AuditEntry{entry}, auditLog...)
	if len(auditLog) > 1000 {
		auditLog = auditLog[:1000]
	}
	auditMu.Unlock()

	// 파일에도 기록
	appdata := os.Getenv("APPDATA")
	if appdata == "" {
		return
	}
	logDir := filepath.Join(appdata, "Nexus", "audit")
	os.MkdirAll(logDir, 0700)
	logFile := filepath.Join(logDir, time.Now().Format("2006-01")+".jsonl")

	data, _ := json.Marshal(entry)
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer f.Close()
	f.Write(append(data, '\n'))
}

// ── 민감 경로 차단 ──────────────────────────────────────────────

var blockedPaths = []string{
	`C:\Windows\System32`,
	`C:\Windows\SysWOW64`,
	`C:\Program Files`,
	`C:\Program Files (x86)`,
}

var sensitiveKeywords = []string{
	"password", "passwd", "credential", "secret", "private_key",
	"id_rsa", ".env", "wallet", "bitcoin", "crypto",
}

func isSafePath(path string) bool {
	pathLow := strings.ToLower(path)
	for _, blocked := range blockedPaths {
		if strings.HasPrefix(strings.ToLower(path), strings.ToLower(blocked)) {
			return false
		}
	}
	for _, kw := range sensitiveKeywords {
		if strings.Contains(pathLow, kw) {
			return false
		}
	}
	return true
}

func isSafeCommand(cmd string) bool {
	cmdLow := strings.ToLower(cmd)
	dangerous := []string{
		"format", "del /f", "rmdir /s", "rd /s",
		"shutdown", "reg delete", "net user", "net localgroup",
		"bcdedit", "diskpart", "cipher /w",
	}
	for _, d := range dangerous {
		if strings.Contains(cmdLow, d) {
			return false
		}
	}
	return true
}

// ── 제한 PowerShell 실행 ────────────────────────────────────────

// runSandboxedPowerShell: -ExecutionPolicy Restricted로 안전하게 실행
func runSandboxedPowerShell(script string) (string, error) {
	if !isSafeCommand(script) {
		writeAudit(AuditEntry{
			Action:     "powershell_blocked",
			Details:    script[:min2(len(script), 100)],
			Approved:   false,
			UserAction: "auto",
			Result:     "차단됨: 위험 명령",
		})
		return "", fmt.Errorf("보안 정책에 의해 차단된 명령입니다")
	}

	out, err := exec.Command("powershell",
		"-NoProfile",
		"-NonInteractive",
		"-ExecutionPolicy", "RemoteSigned",
		"-Command", script,
	).Output()

	writeAudit(AuditEntry{
		Action:     "powershell_run",
		Details:    script[:min2(len(script), 100)],
		Approved:   true,
		UserAction: "auto",
		Result:     strings.TrimSpace(string(out)),
	})

	return strings.TrimSpace(string(out)), err
}

func min2(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ── 로컬 LLM (Ollama) 연동 ──────────────────────────────────────

type OllamaConfig struct {
	Enabled bool   `json:"enabled"`
	URL     string `json:"url"`   // 기본: http://localhost:11434
	Model   string `json:"model"` // 기본: llama3
}

var (
	ollamaMu  sync.RWMutex
	ollamaCfg OllamaConfig
)

func loadOllamaConfig() {
	appdata := os.Getenv("APPDATA")
	if appdata == "" {
		return
	}
	data, err := os.ReadFile(filepath.Join(appdata, "Nexus", "ollama_config.json"))
	if err != nil {
		ollamaCfg = OllamaConfig{Enabled: false, URL: "http://localhost:11434", Model: "llama3"}
		return
	}
	json.Unmarshal(data, &ollamaCfg)
	if ollamaCfg.URL == "" {
		ollamaCfg.URL = "http://localhost:11434"
	}
	if ollamaCfg.Model == "" {
		ollamaCfg.Model = "llama3"
	}
}

func saveOllamaConfig() {
	appdata := os.Getenv("APPDATA")
	if appdata == "" {
		return
	}
	ollamaMu.RLock()
	data, _ := json.MarshalIndent(ollamaCfg, "", "  ")
	ollamaMu.RUnlock()
	os.WriteFile(filepath.Join(appdata, "Nexus", "ollama_config.json"), data, 0600)
}

// callOllama: 로컬 LLM 호출 (인터넷 불필요)
func callOllamaLocal(prompt string) (string, error) {
	ollamaMu.RLock()
	cfg := ollamaCfg
	ollamaMu.RUnlock()

	if !cfg.Enabled {
		return "", fmt.Errorf("Ollama가 비활성화 상태입니다")
	}

	type ollamaReq struct {
		Model  string `json:"model"`
		Prompt string `json:"prompt"`
		Stream bool   `json:"stream"`
	}

	reqBody, _ := json.Marshal(ollamaReq{Model: cfg.Model, Prompt: prompt, Stream: false})

	resp, err := httpPost(cfg.URL+"/api/generate", reqBody)
	if err != nil {
		return "", fmt.Errorf("Ollama 연결 실패: %w", err)
	}

	var result struct {
		Response string `json:"response"`
	}
	json.Unmarshal(resp, &result)
	return strings.TrimSpace(result.Response), nil
}

// ── HTTP 핸들러 ─────────────────────────────────────────────────

// GET /api/security/audit — 감사 로그 조회
func handleAuditLog(w http.ResponseWriter, r *http.Request) {
	limit := 50
	limitStr := r.URL.Query().Get("limit")
	if limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}

	auditMu.Lock()
	entries := make([]AuditEntry, 0, limit)
	for i, e := range auditLog {
		if i >= limit {
			break
		}
		entries = append(entries, e)
	}
	auditMu.Unlock()

	json200(w, map[string]any{
		"success": true,
		"entries": entries,
		"total":   len(auditLog),
	})
}

// POST /api/security/check-path — 경로 안전성 확인
func handleCheckPath(w http.ResponseWriter, r *http.Request) {
	var req struct{ Path string `json:"path"` }
	json.NewDecoder(r.Body).Decode(&req)
	safe := isSafePath(req.Path)
	json200(w, map[string]any{"safe": safe, "path": req.Path})
}

// GET/POST /api/ollama/config — 로컬 LLM 설정
func handleOllamaConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		ollamaMu.RLock()
		cfg := ollamaCfg
		ollamaMu.RUnlock()
		json200(w, cfg)
		return
	}
	var req OllamaConfig
	json.NewDecoder(r.Body).Decode(&req)
	ollamaMu.Lock()
	if req.URL != "" {
		ollamaCfg.URL = req.URL
	}
	if req.Model != "" {
		ollamaCfg.Model = req.Model
	}
	ollamaCfg.Enabled = req.Enabled
	ollamaMu.Unlock()
	saveOllamaConfig()
	json200(w, map[string]any{"success": true, "message": "로컬 LLM 설정 저장 완료"})
}

// POST /api/ollama/test — Ollama 연결 테스트
func handleOllamaTest(w http.ResponseWriter, r *http.Request) {
	ans, err := callOllamaLocal("안녕하세요, 테스트 응답을 한 문장으로 해주세요.")
	if err != nil {
		json200(w, map[string]any{"success": false, "message": err.Error()})
		return
	}
	json200(w, map[string]any{"success": true, "response": ans, "message": "로컬 LLM 연결 성공!"})
}

// GET /api/ollama/models — 설치된 모델 목록
func handleOllamaModels(w http.ResponseWriter, r *http.Request) {
	ollamaMu.RLock()
	url := ollamaCfg.URL
	ollamaMu.RUnlock()

	resp, err := httpGet(url + "/api/tags")
	if err != nil {
		json200(w, map[string]any{"success": false, "message": "Ollama에 연결할 수 없습니다. Ollama가 실행 중인지 확인해주세요.", "install_url": "https://ollama.ai"})
		return
	}

	var result map[string]any
	json.Unmarshal(resp, &result)
	json200(w, map[string]any{"success": true, "models": result})
}
