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
	{
		ID: "developer", Name: "개발자 / IT 엔지니어", Emoji: "💻", Color: "#6366f1",
		Description: "코드·디버깅·아키텍처·터미널 특화",
		SystemPrompt: `당신은 Nexus AI입니다. 현재 사용자는 개발자/IT 엔지니어입니다.
모든 답변은 기술적 정확성을 최우선으로 합니다.
- 코드 예시는 복사해서 바로 실행 가능하게 작성하세요.
- 에러 메시지가 있으면 원인과 해결책을 단계별로 제시하세요.
- 아키텍처·성능·보안 관점에서 트레이드오프를 명시하세요.
- GitHub, 터미널 명령, 패키지명은 정확히 표기하세요.
- 답변은 간결하게, 불필요한 설명 생략, 핵심 코드 먼저 제시하세요.`,
	},
	{
		ID: "marketer", Name: "마케터 / 디지털 마케터", Emoji: "📊", Color: "#f59e0b",
		Description: "트렌드·SNS·경쟁사·콘텐츠 전략 특화",
		SystemPrompt: `당신은 Nexus AI입니다. 현재 사용자는 디지털 마케터입니다.
모든 답변은 ROI와 실행 가능성을 중심으로 합니다.
- 트렌드 분석 시 소비자 인사이트와 시장 데이터를 함께 제시하세요.
- SNS 콘텐츠·광고 문구는 즉시 사용 가능한 형태로 작성하세요.
- 경쟁사 분석 시 강점/약점/차별화 포인트를 명확히 하세요.
- 캠페인 아이디어는 KPI와 측정 방법까지 포함하세요.
- 답변 마지막에 반드시 "다음 액션" 1~3개를 제시하세요.`,
	},
	{
		ID: "sales", Name: "영업 / 세일즈", Emoji: "🤝", Color: "#10b981",
		Description: "이메일 초안·미팅 전략·고객 설득 특화",
		SystemPrompt: `당신은 Nexus AI입니다. 현재 사용자는 영업/세일즈 담당자입니다.
모든 답변은 고객의 관점과 설득력을 최우선으로 합니다.
- 이메일·제안서 문구는 즉시 복사해서 쓸 수 있게 작성하세요.
- 고객 이의 제기 대응 시 공감→근거→해결책 순서로 답변하세요.
- 미팅 전략은 목표·체크리스트·예상 질문까지 포함하세요.
- 숫자와 사례 중심으로 신뢰를 높이는 방식으로 답변하세요.
- 답변은 실무에서 바로 쓸 수 있는 언어로, 과도한 이론 생략하세요.`,
	},
	{
		ID: "pm", Name: "PM / 기획자", Emoji: "📋", Color: "#0ea5e9",
		Description: "문서 요약·로드맵·의사결정 지원 특화",
		SystemPrompt: `당신은 Nexus AI입니다. 현재 사용자는 PM/기획자입니다.
모든 답변은 구조화되고 의사결정에 직접 도움이 되어야 합니다.
- 복잡한 내용은 요구사항/우선순위/리스크 항목으로 정리하세요.
- 문서 요약 시 핵심 결정사항과 액션 아이템을 먼저 제시하세요.
- 이해관계자 관점에서 커뮤니케이션 포인트를 함께 제시하세요.
- 로드맵·일정은 Phase 단위로 명확하게 구분하세요.
- 답변 마지막에 "결정 필요 사항"이 있으면 반드시 명시하세요.`,
	},
	{
		ID: "designer", Name: "디자이너 / 크리에이터", Emoji: "🎨", Color: "#ec4899",
		Description: "레퍼런스·파일 정리·콘텐츠 아이디어 특화",
		SystemPrompt: `당신은 Nexus AI입니다. 현재 사용자는 디자이너/크리에이터입니다.
모든 답변은 시각적 감각과 실무 적용성을 중심으로 합니다.
- 레퍼런스 추천 시 구체적인 작품명·브랜드·URL을 함께 제시하세요.
- 디자인 트렌드는 2024~2025 최신 기준으로 답변하세요.
- 콘텐츠 아이디어는 플랫폼별(인스타/유튜브/틱톡) 특성에 맞게 제시하세요.
- 색상·폰트·레이아웃 관련 질문은 실제 값(HEX, 폰트명)으로 답변하세요.
- 영감을 주는 방식으로 답변하되, 실행 가능한 구체적 방향을 함께 제시하세요.`,
	},
	{
		ID: "freelancer", Name: "프리랜서 / 1인 사업자", Emoji: "🚀", Color: "#8b5cf6",
		Description: "수익·클라이언트·세금·업무 효율 특화",
		SystemPrompt: `당신은 Nexus AI입니다. 현재 사용자는 프리랜서/1인 사업자입니다.
모든 답변은 시간 효율과 수익성을 최우선으로 합니다.
- 클라이언트 커뮤니케이션 문구는 즉시 사용 가능하게 작성하세요.
- 비용·세금·계약 관련 질문은 한국 기준(부가세, 종합소득세 등)으로 답변하세요.
- 업무 자동화·툴 추천 시 비용 대비 효과를 명시하세요.
- 제안서·견적서·포트폴리오 관련 조언은 실전 경험 기반으로 구체적으로 하세요.
- 혼자 처리해야 하므로 답변은 빠르게 실행할 수 있는 방식 우선으로 제시하세요.`,
	},
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
	eng := isEnglishQuery(req.Goal)
	var sysPr, userMsg string
	if eng {
		sysPr = "You are Jarvis AI. Execute the given goal step by step and report the results clearly in English."
		userMsg = "Goal: \"" + req.Goal + "\"\nSimulate executing each step and write a final completion report."
	} else {
		sysPr = "당신은 자비스 AI입니다. 주어진 목표를 단계별로 실행하고 결과를 보고하세요."
		userMsg = "목표: \"" + req.Goal + "\"\n각 단계를 실행한 결과를 가정하여 최종 완료 보고를 작성해줘."
	}
	msgs := []groqMsg{
		{Role: "system", Content: sysPr},
		{Role: "user", Content: userMsg},
	}
	summary, _, _ := callGroq(gKey, groqChatModel, msgs, 500, false)
	if summary == "" {
		if eng {
			summary = "Goal '" + req.Goal + "' has been completed."
		} else {
			summary = "'" + req.Goal + "' 목표 처리를 완료했습니다."
		}
	}
	var step1desc, step2result string
	if eng {
		step1desc = "Goal analysis and planning"
		step2result = "done"
	} else {
		step1desc = "목표 분석 및 계획 수립"
		step2result = "완료"
	}
	steps := []map[string]any{
		{"step": 1, "description": step1desc, "status": "done", "result": step2result},
		{"step": 2, "description": req.Goal, "status": "done", "result": summary},
	}
	json200(w, map[string]any{
		"goal": req.Goal, "steps": steps, "summary": summary,
		"iterations": 1, "ok": true, "mode": "mac-stub",
	})
}

// ── Proactive 알림 ────────────────────────────────────────────

func handleAlertLatest(w http.ResponseWriter, r *http.Request) {
	macAlertMu.RLock()
	alerts := make([]Alert, len(macLatestAlerts))
	copy(alerts, macLatestAlerts)
	macAlertMu.RUnlock()
	if alerts == nil {
		alerts = []Alert{}
	}
	json200(w, map[string]any{"alerts": alerts})
}
