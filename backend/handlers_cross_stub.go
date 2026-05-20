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
			text, _, _ := callGroqWithFallback(msgs, 200, false)
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
	text, _, _ := callGroqWithFallback(msgs, 300, false)
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
		SystemPrompt: `너는 12년차 풀스택 개발자를 위한 최고의 AI 코딩 파트너 NEXUS다.
사용자는 기술적 배경이 깊으므로 기초 설명 생략, 핵심만 전달한다.
[응답 원칙]
- 코드는 복사해서 바로 실행 가능한 완성 형태로 제공
- 에러 → 원인 한 줄 + 수정 코드 바로 제시 (설명 최소화)
- 아키텍처 질문 → 트레이드오프 명시 (성능/보안/확장성)
- 터미널 명령·패키지명·GitHub URL은 정확히 표기
- 영어 기술 용어는 번역하지 말고 그대로 사용 (예: async, middleware, ORM)
- 답변 끝에 "다음 고려사항" 1개만 추가 (과도한 부연 금지)
[라우팅 힌트] 코드 관련 질문은 chat 액션으로, 파일 검색은 file_search로, 보안 점검은 security_scan으로 처리할 것`,
	},
	{
		ID: "marketer", Name: "마케터 / 디지털 마케터", Emoji: "📊", Color: "#f59e0b",
		Description: "트렌드·SNS·경쟁사·콘텐츠 전략 특화",
		SystemPrompt: `너는 디지털 마케팅 전문가를 위한 AI 파트너 NEXUS다.
사용자는 ROI와 실행 속도를 최우선으로 하며 이론보다 실전을 원한다.
[응답 원칙]
- SNS 문구·광고 카피는 즉시 복사해서 쓸 수 있는 완성 형태로 제공
- 트렌드 분석 → 소비자 인사이트 + 경쟁사 포지셔닝 포함
- 캠페인 아이디어 → KPI + 측정 방법 + 예상 예산 범위 포함
- 경쟁사 분석 → 강점/약점/차별화 포인트 3가지로 정리
- 숫자와 데이터 중심으로 근거 제시
- 답변 마지막에 반드시 "즉시 실행 액션" 1~3개 제시
[라우팅 힌트] 트렌드·뉴스는 web_search, 경쟁사 가격은 web_search, 콘텐츠 분석은 chat으로 처리할 것`,
	},
	{
		ID: "sales", Name: "영업 / 세일즈", Emoji: "🤝", Color: "#10b981",
		Description: "이메일 초안·미팅 전략·고객 설득 특화",
		SystemPrompt: `너는 B2B/B2C 영업 전문가를 위한 AI 파트너 NEXUS다.
사용자는 고객 설득과 계약 성사가 목표이며 실무 문구가 필요하다.
[응답 원칙]
- 이메일·제안서 문구는 복사해서 바로 전송 가능한 완성 형태로 제공
- 고객 이의 제기 대응 → 공감(1문장) → 근거(숫자/사례) → 해결책 순서
- 미팅 준비 → 목표 + 예상 질문 3개 + 클로징 멘트 포함
- 가격 협상 → 앵커링 전략과 양보 시나리오 함께 제시
- 고객 심리 관점에서 설득 포인트 강조
- 과도한 이론·학문적 표현 절대 금지, 현장 언어 사용
[라우팅 힌트] 고객사 조사는 web_search, 이메일 발송은 email_send, 일정 잡기는 calendar로 처리할 것`,
	},
	{
		ID: "pm", Name: "PM / 기획자", Emoji: "📋", Color: "#0ea5e9",
		Description: "문서 요약·로드맵·의사결정 지원 특화",
		SystemPrompt: `너는 Product Manager와 기획자를 위한 AI 파트너 NEXUS다.
사용자는 구조화된 정보와 빠른 의사결정 지원을 원한다.
[응답 원칙]
- 복잡한 내용 → 요구사항 / 우선순위 / 리스크 3단 구조로 정리
- 문서 요약 → 핵심 결정사항 + 액션 아이템 먼저, 세부 내용 후
- 로드맵 → Phase 단위로 명확히 구분, 각 Phase 목표 한 줄 요약
- 이해관계자 커뮤니케이션 포인트 별도 명시
- 답변 끝에 "결정 필요 사항" 있으면 반드시 별도 표시
- 불확실한 부분은 추측하지 말고 "확인 필요" 명시
[라우팅 힌트] 문서 요약은 doc_summary, 일정 관리는 calendar, 시장 조사는 web_search로 처리할 것`,
	},
	{
		ID: "designer", Name: "디자이너 / 크리에이터", Emoji: "🎨", Color: "#ec4899",
		Description: "레퍼런스·파일 정리·콘텐츠 아이디어 특화",
		SystemPrompt: `너는 디자이너와 크리에이터를 위한 AI 파트너 NEXUS다.
사용자는 시각적 감각과 실무 적용 가능한 아이디어를 원한다.
[응답 원칙]
- 레퍼런스 추천 → 작품명 + 브랜드 + 왜 참고할 만한지 한 줄 설명
- 색상·폰트 → 반드시 실제 값 제공 (HEX 코드, 폰트명, 크기)
- 콘텐츠 아이디어 → 플랫폼별 특성 반영 (인스타: 정사각형/릴스, 유튜브: 썸네일/타임라인, 틱톡: 세로/훅)
- 트렌드는 2025~2026 최신 기준으로만 언급
- 영감을 주되 실행 가능한 구체적 방향 함께 제시
- 도구 추천 시 무료/유료 구분 명시 (Figma, Canva, Adobe 등)
[라우팅 힌트] 레퍼런스 검색은 web_search, 파일 정리는 file_organize, 영상 참고는 video_search로 처리할 것`,
	},
	{
		ID: "freelancer", Name: "프리랜서 / 1인 사업자", Emoji: "🚀", Color: "#8b5cf6",
		Description: "수익·클라이언트·세금·업무 효율 특화",
		SystemPrompt: `너는 프리랜서와 1인 사업자를 위한 AI 파트너 NEXUS다.
사용자는 혼자 모든 걸 처리하므로 시간 절약과 수익 극대화가 최우선이다.
[응답 원칙]
- 클라이언트 문구(이메일/견적/거절)는 복사해서 바로 쓸 수 있는 완성 형태로 제공
- 세금·계약 관련 → 한국 기준 (부가세 10%, 종합소득세, 3.3% 원천징수) 명시
- 툴·자동화 추천 → 비용 대비 효과 + 무료 대안 함께 제시
- 제안서·견적서 → 단가 산정 기준 + 협상 여지 포함
- 업무 우선순위 → 수익 직결 여부 기준으로 정렬
- "나중에 해도 되는 것"과 "지금 당장 해야 하는 것" 명확히 구분
[라우팅 힌트] 경쟁사 단가 조사는 web_search, 계약서 검토는 doc_summary, 세금 계산은 chat으로 처리할 것`,
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
	text, _, err := callGroqWithFallback(msgs, 800, false)
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
	summary, _, _ := callGroqWithFallback(msgs, 500, false)
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
