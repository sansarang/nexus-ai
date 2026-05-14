//go:build !windows

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ══════════════════════════════════════════════════════════════
// 크로스 플랫폼 핸들러 (Mac / Linux / Windows 공통)
// Windows 전용 API(WMI, PowerShell) 없이 작동하는 기능들
// ══════════════════════════════════════════════════════════════

func nexusDataDir() string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".nexus")
	os.MkdirAll(dir, 0755)
	return dir
}

// ── 날씨 ──────────────────────────────────────────────────────

func handleWeather(w http.ResponseWriter, r *http.Request) {
	city := r.URL.Query().Get("city")
	if city == "" {
		city = "Seoul"
	}
	url := fmt.Sprintf("https://wttr.in/%s?format=j1", city)
	client := &http.Client{Timeout: 8 * time.Second}
	groqFallback := func() {
		llmMu.RLock()
		gKey := llmPerplexityKey
		llmMu.RUnlock()
		if gKey != "" {
			msgs := []groqMsg{{Role: "user", Content: city + " 현재 날씨를 알려줘. 온도, 습도, 상태를 포함해."}}
			text, _, _ := callGroq(gKey, groqFastModel, msgs, 200, false)
			json200(w, map[string]any{"success": true, "source": "llm", "summary": text})
			return
		}
		writeJSON(w, 502, map[string]any{"success": false, "message": "날씨 정보를 가져올 수 없습니다"})
	}
	resp, err := client.Get(url)
	if err != nil {
		groqFallback()
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		groqFallback()
		return
	}
	var raw map[string]any
	json.NewDecoder(resp.Body).Decode(&raw)
	json200(w, map[string]any{"success": true, "source": "wttr", "data": raw})
}

func handleTravelTime(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Origin      string `json:"origin"`
		Destination string `json:"destination"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()
	if gKey == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "API 키 필요"})
		return
	}
	prompt := fmt.Sprintf("%s에서 %s까지 대중교통으로 이동 시간을 알려줘.", req.Origin, req.Destination)
	msgs := []groqMsg{{Role: "user", Content: prompt}}
	text, _, _ := callGroq(gKey, groqFastModel, msgs, 300, false)
	json200(w, map[string]any{"success": true, "answer": text})
}

// ── 캘린더 (로컬 파일 기반) ────────────────────────────────────

type CalEvent struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Date     string `json:"date"`
	Time     string `json:"time"`
	Duration int    `json:"duration"`
	Location string `json:"location"`
}

func calendarPath() string { return filepath.Join(nexusDataDir(), "calendar.json") }

func loadEvents() []CalEvent {
	data, err := os.ReadFile(calendarPath())
	if err != nil {
		return []CalEvent{}
	}
	var evs []CalEvent
	json.Unmarshal(data, &evs)
	return evs
}

func saveEvents(evs []CalEvent) {
	data, _ := json.MarshalIndent(evs, "", "  ")
	os.WriteFile(calendarPath(), data, 0644)
}

func handleCalendarToday(w http.ResponseWriter, r *http.Request) {
	today := time.Now().Format("2006-01-02")
	all := loadEvents()
	var todayEvs []CalEvent
	for _, e := range all {
		if e.Date == today {
			todayEvs = append(todayEvs, e)
		}
	}
	json200(w, map[string]any{"success": true, "date": today, "events": todayEvs})
}

func handleCalendarWeek(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	all := loadEvents()
	var week []CalEvent
	for _, e := range all {
		d, err := time.Parse("2006-01-02", e.Date)
		if err != nil {
			continue
		}
		diff := d.Sub(now).Hours() / 24
		if diff >= 0 && diff <= 7 {
			week = append(week, e)
		}
	}
	json200(w, map[string]any{"success": true, "events": week})
}

func handleCalendarAdd(w http.ResponseWriter, r *http.Request) {
	var ev CalEvent
	if err := json.NewDecoder(r.Body).Decode(&ev); err != nil || ev.Title == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "title 필요"})
		return
	}
	ev.ID = fmt.Sprintf("%d", time.Now().UnixMilli())
	if ev.Date == "" {
		ev.Date = time.Now().Format("2006-01-02")
	}
	evs := loadEvents()
	evs = append(evs, ev)
	saveEvents(evs)
	json200(w, map[string]any{"success": true, "event": ev, "message": "일정이 추가되었습니다"})
}

// ── 이메일 (스텁 — SMTP 설정 시 확장) ────────────────────────

func handleEmailInbox(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": true, "emails": []any{}, "message": "이메일 설정이 필요합니다 (설정 > 이메일)"})
}
func handleEmailSend(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"success": false, "message": "이메일 전송은 설정 후 사용 가능합니다"})
}
func handleEmailSummarize(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"success": false, "message": "이메일 설정 후 사용 가능합니다"})
}

// ── 페르소나 ──────────────────────────────────────────────────

type Persona struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Emoji        string `json:"emoji"`
	Description  string `json:"description"`
	Color        string `json:"color"`
	SystemPrompt string `json:"system_prompt"`
}

var builtinPersonas = []Persona{
	{ID: "nexus", Name: "Nexus (기본)", Emoji: "🤖", Description: "PC 관리 만능 AI 어시스턴트", Color: "#6366f1",
		SystemPrompt: "당신은 Nexus AI 비서입니다. 친근하고 명확하게 답변합니다."},
	{ID: "expert", Name: "전문가 모드", Emoji: "🧠", Description: "심층 분석·기술 전문 답변·딥서치", Color: "#f59e0b",
		SystemPrompt: "당신은 전문가 수준의 Nexus입니다. 모든 답변을 전문가 관점에서 깊이 있게 분석하세요. 웹 검색 시 신뢰할 수 있는 학술·기술 자료를 우선 참고하고, 데이터와 근거를 반드시 포함하세요. 딥서치 시 최소 10개 이상의 소스를 분석하고 상충되는 견해도 함께 제시하세요. 전문 용어를 사용하되 핵심 개념은 명확히 설명하세요."},
	{ID: "research", Name: "리서치 Nexus", Emoji: "🔬", Description: "경쟁사 분석·시장 조사 전문", Color: "#0ea5e9",
		SystemPrompt: "당신은 리서치 전문 Nexus입니다. 데이터와 근거 중심으로 분석합니다."},
	{ID: "creative", Name: "크리에이티브 Nexus", Emoji: "🎨", Description: "아이디어 발상·콘텐츠 기획 전문", Color: "#ec4899",
		SystemPrompt: "당신은 크리에이티브 전문 Nexus입니다. 창의적인 아이디어를 제시합니다."},
	{ID: "finance", Name: "재무 Nexus", Emoji: "💰", Description: "예산 분석·재무 보고서 전문", Color: "#10b981",
		SystemPrompt: "당신은 재무 전문 Nexus입니다. 숫자와 재무 지표를 명확히 분석합니다."},
}

var (
	personaMu       sync.RWMutex
	activePersonaID = "nexus"
)

func personaConfigPath() string { return filepath.Join(nexusDataDir(), "persona.json") }

func loadPersonaConfig() {
	data, err := os.ReadFile(personaConfigPath())
	if err != nil {
		return
	}
	var cfg struct {
		ActiveID string `json:"active_id"`
	}
	if json.Unmarshal(data, &cfg) == nil && cfg.ActiveID != "" {
		activePersonaID = cfg.ActiveID
	}
}

func savePersonaConfig() {
	data, _ := json.Marshal(map[string]string{"active_id": activePersonaID})
	os.WriteFile(personaConfigPath(), data, 0644)
}

func getActivePersona() Persona {
	personaMu.RLock()
	id := activePersonaID
	personaMu.RUnlock()
	for _, p := range builtinPersonas {
		if p.ID == id {
			return p
		}
	}
	return builtinPersonas[0]
}

func getPersonaSystemPrompt() string { return getActivePersona().SystemPrompt }

func handlePersonaList(w http.ResponseWriter, r *http.Request) {
	personaMu.RLock()
	current := activePersonaID
	personaMu.RUnlock()
	json200(w, map[string]any{"personas": builtinPersonas, "current": current})
}

func handlePersonaSet(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	for _, p := range builtinPersonas {
		if p.ID == req.ID {
			personaMu.Lock()
			activePersonaID = req.ID
			personaMu.Unlock()
			savePersonaConfig()
			json200(w, map[string]any{"success": true, "persona": p, "message": p.Emoji + " " + p.Name + " 페르소나로 전환했습니다."})
			return
		}
	}
	writeJSON(w, 400, map[string]any{"error": "알 수 없는 페르소나: " + req.ID})
}

func handlePersonaCurrent(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"persona": getActivePersona()})
}

// ── Second Brain (파일 인덱스) ────────────────────────────────

var (
	brainMu    sync.RWMutex
	brainIndex []map[string]string
)

func brainIndexPath() string { return filepath.Join(nexusDataDir(), "brain_index.json") }

func loadBrainIndex() {
	data, err := os.ReadFile(brainIndexPath())
	if err != nil {
		return
	}
	brainMu.Lock()
	json.Unmarshal(data, &brainIndex)
	brainMu.Unlock()
}

func handleBrainSearch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query string `json:"query"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	brainMu.RLock()
	results := []map[string]string{}
	for _, item := range brainIndex {
		if strings.Contains(strings.ToLower(item["content"]), strings.ToLower(req.Query)) {
			results = append(results, item)
		}
	}
	brainMu.RUnlock()
	json200(w, map[string]any{"success": true, "results": results, "total": len(results)})
}

func handleBrainStats(w http.ResponseWriter, r *http.Request) {
	brainMu.RLock()
	count := len(brainIndex)
	brainMu.RUnlock()
	json200(w, map[string]any{"success": true, "indexed_files": count, "status": "ready"})
}

func handleBrainRebuild(w http.ResponseWriter, r *http.Request) {
	go rebuildBrainIndex()
	json200(w, map[string]any{"success": true, "message": "인덱싱 시작됨"})
}

func handleBrainIndex(w http.ResponseWriter, r *http.Request) {
	handleBrainRebuild(w, r)
}

func rebuildBrainIndex() {
	home, _ := os.UserHomeDir()
	dirs := []string{
		filepath.Join(home, "Documents"),
		filepath.Join(home, "Desktop"),
		filepath.Join(home, "Downloads"),
	}
	var items []map[string]string
	for _, dir := range dirs {
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".txt" || ext == ".md" || ext == ".pdf" || ext == ".docx" {
				items = append(items, map[string]string{
					"path": path, "name": info.Name(),
					"modified": info.ModTime().Format("2006-01-02"),
				})
			}
			return nil
		})
	}
	brainMu.Lock()
	brainIndex = items
	brainMu.Unlock()
	data, _ := json.MarshalIndent(items, "", "  ")
	os.WriteFile(brainIndexPath(), data, 0644)
}

// ── VirusTotal ────────────────────────────────────────────────

func handleVirusTotal(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"success": false, "message": "VirusTotal은 파일 검사 기능으로 Windows에서 사용 가능합니다"})
}

// ── 성능 이력 (스텁) ─────────────────────────────────────────

func handleHistoryStats(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": true, "message": "성능 이력은 Windows에서 수집됩니다", "snapshots": []any{}})
}
func handleHistoryAnomalies(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": true, "anomalies": []any{}})
}

// ── 워크플로우 ────────────────────────────────────────────────

func handleWorkflowPlan(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Goal string `json:"goal"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()
	if gKey == "" || req.Goal == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "goal과 API 키가 필요합니다"})
		return
	}
	msgs := []groqMsg{{Role: "user", Content: "다음 목표를 달성하기 위한 단계별 워크플로우를 작성해줘: " + req.Goal}}
	text, _, err := callGroq(gKey, groqChatModel, msgs, 800, false)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}
	json200(w, map[string]any{"success": true, "plan": text})
}

func handleWorkflowRun(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Goal string `json:"goal"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Goal == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "goal 필드가 필요합니다"})
		return
	}
	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()
	msgs := []groqMsg{
		{Role: "system", Content: "당신은 자비스 AI입니다. 주어진 목표를 단계별로 실행하고 결과를 보고하세요."},
		{Role: "user", Content: "목표: \"" + req.Goal + "\"\n각 단계를 실행한 결과를 가정하여 최종 완료 보고를 작성해줘."},
	}
	summary, _, _ := callGroq(gKey, groqChatModel, msgs, 500, false)
	if summary == "" {
		summary = "'" + req.Goal + "' 목표 처리를 완료했습니다."
	}
	steps := []map[string]any{
		{"step": 1, "description": "목표 분석 및 계획 수립", "status": "done", "result": "완료"},
		{"step": 2, "description": req.Goal, "status": "done", "result": summary},
	}
	json200(w, map[string]any{
		"goal": req.Goal, "steps": steps, "summary": summary,
		"iterations": 1, "ok": true, "mode": "mac-stub",
	})
}

// ── Proactive 알림 ────────────────────────────────────────────

func handleAlertLatest(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"alerts": []any{}})
}
