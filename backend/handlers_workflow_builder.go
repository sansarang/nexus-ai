//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ══════════════════════════════════════════════════════════════════
//  No-code Visual Workflow Builder
//  Trigger → Condition → Actions → Notification 구조
//  JSON으로 저장, 프론트에서 드래그앤드롭으로 편집
// ══════════════════════════════════════════════════════════════════

type WFTrigger struct {
	Type  string         `json:"type"`   // schedule|event|condition|manual
	Value string         `json:"value"`  // cron표현식 / 이벤트명 / 조건식
	Label string         `json:"label"`
}

type WFCondition struct {
	Field    string `json:"field"`    // cpu|memory|disk|time|email_count
	Operator string `json:"operator"` // gt|lt|eq|contains
	Value    string `json:"value"`
}

type WFAction struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`   // api_call|notification|desktop_agent|email|multi_agent
	Label    string         `json:"label"`
	Endpoint string         `json:"endpoint,omitempty"`
	Method   string         `json:"method,omitempty"`
	Params   map[string]any `json:"params,omitempty"`
	Goal     string         `json:"goal,omitempty"`  // multi_agent / desktop_agent용
	NextID   string         `json:"next_id,omitempty"` // 다음 액션 ID (연결)
}

type WFNotification struct {
	Channel string `json:"channel"` // bubble|alert|email
	Message string `json:"message"`
}

type VisualWorkflow struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	Enabled      bool            `json:"enabled"`
	Trigger      WFTrigger       `json:"trigger"`
	Conditions   []WFCondition   `json:"conditions,omitempty"`
	Actions      []WFAction      `json:"actions"`
	Notification *WFNotification `json:"notification,omitempty"`
	CreatedAt    string          `json:"created_at"`
	UpdatedAt    string          `json:"updated_at"`
	LastRun      string          `json:"last_run,omitempty"`
	RunCount     int             `json:"run_count"`
	Tags         []string        `json:"tags,omitempty"`
}

func workflowBuilderDir() string {
	appdata := os.Getenv("APPDATA")
	if appdata == "" {
		appdata = os.TempDir()
	}
	dir := filepath.Join(appdata, "Nexus", "workflows")
	os.MkdirAll(dir, 0755)
	return dir
}

func loadAllWorkflows() []VisualWorkflow {
	dir := workflowBuilderDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var workflows []VisualWorkflow
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var wf VisualWorkflow
		if json.Unmarshal(data, &wf) == nil {
			workflows = append(workflows, wf)
		}
	}
	// 최신순 정렬
	sort.Slice(workflows, func(i, j int) bool {
		return workflows[i].UpdatedAt > workflows[j].UpdatedAt
	})
	return workflows
}

func saveWorkflow(wf VisualWorkflow) error {
	dir := workflowBuilderDir()
	data, err := json.MarshalIndent(wf, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, wf.ID+".json"), data, 0644)
}

func deleteWorkflow(id string) error {
	return os.Remove(filepath.Join(workflowBuilderDir(), id+".json"))
}

// ── 워크플로우 실행 엔진 ────────────────────────────────────────

func checkWFConditions(wf VisualWorkflow) bool {
	for _, cond := range wf.Conditions {
		var actual float64
		switch cond.Field {
		case "cpu":
			actual = float64(getMemoryUsage()) // CPU 추정
		case "memory":
			actual = float64(getMemoryUsage())
		case "disk":
			free, total := getDiskSpace()
			if total > 0 {
				actual = 100 - float64(free)/float64(total)*100
			}
		case "time":
			actual = float64(time.Now().Hour())
		}

		var threshold float64
		fmt.Sscanf(cond.Value, "%f", &threshold)

		switch cond.Operator {
		case "gt":
			if actual <= threshold {
				return false
			}
		case "lt":
			if actual >= threshold {
				return false
			}
		case "eq":
			if actual != threshold {
				return false
			}
		}
	}
	return true
}

func executeWFActions(wf VisualWorkflow) string {
	results := []string{}

	for _, action := range wf.Actions {
		var result string

		switch action.Type {
		case "api_call":
			// 내부 API 호출
			result = fmt.Sprintf("API 호출: %s %s", action.Method, action.Endpoint)
			// 실제 구현 시 내부 HTTP 클라이언트로 호출

		case "notification":
			msg := action.Params["message"]
			if msg != nil {
				publishAlert(Alert{
					ID:      fmt.Sprintf("wf_%s_%d", wf.ID, time.Now().Unix()),
					Level:   "info",
					Title:   wf.Name,
					Message: fmt.Sprintf("%v", msg),
				})
				result = "알림 전송 완료"
			}

		case "desktop_agent":
			if action.Goal != "" {
				task := globalTaskQueue.Enqueue("WF-Desktop: "+action.Goal, PriorityNormal, nil,
					func(t *AgentTask) {
						runDesktopAgent(t, action.Goal, false, 15)
					})
				result = "Desktop Agent 시작: " + task.ID
			}

		case "multi_agent":
			if action.Goal != "" {
				llmMu.RLock()
				gKey := llmPerplexityKey
				llmMu.RUnlock()
				plan, err := orchestrate(action.Goal, gKey)
				if err == nil {
					task := globalTaskQueue.Enqueue("WF-MultiAgent: "+action.Goal, PriorityBackground, nil,
						func(t *AgentTask) {
							runMultiAgentPlan(t, plan, gKey)
						})
					result = "Multi-Agent 시작: " + task.ID
				}
			}

		case "email":
			to, _ := action.Params["to"].(string)
			subject, _ := action.Params["subject"].(string)
			body, _ := action.Params["body"].(string)
			if to != "" {
				err := sendOutlookEmail(to, subject, body)
				if err == nil {
					result = "이메일 발송 완료"
				} else {
					result = "이메일 발송 실패: " + err.Error()
				}
			}
		}

		results = append(results, fmt.Sprintf("[%s] %s", action.Label, result))
	}

	// 완료 알림
	if wf.Notification != nil {
		msg := wf.Notification.Message
		if msg == "" {
			msg = fmt.Sprintf("워크플로우 '%s' 완료", wf.Name)
		}
		publishAlert(Alert{
			ID:      fmt.Sprintf("wf_done_%s_%d", wf.ID, time.Now().Unix()),
			Level:   "info",
			Title:   "✅ " + wf.Name,
			Message: msg,
		})
	}

	return strings.Join(results, "\n")
}

// ── 워크플로우 스케줄러 ─────────────────────────────────────────

func startWorkflowScheduler() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		workflows := loadAllWorkflows()
		now := time.Now()

		for _, wf := range workflows {
			if !wf.Enabled {
				continue
			}

			shouldRun := false

			switch wf.Trigger.Type {
			case "schedule":
				// cron 간단 파싱: "08:00" → 매일 해당 시각
				var h, m int
				if n, _ := fmt.Sscanf(wf.Trigger.Value, "%d:%d", &h, &m); n == 2 {
					if now.Hour() == h && now.Minute() == m {
						shouldRun = true
					}
				}
			case "condition":
				shouldRun = checkWFConditions(wf)
			case "manual":
				shouldRun = false
			}

			if shouldRun && checkWFConditions(wf) {
				globalTaskQueue.Enqueue("Workflow: "+wf.Name, PriorityBackground, nil,
					func(t *AgentTask) {
						result := executeWFActions(wf)
						t.Result = map[string]any{"result": result}

						// 실행 카운트 업데이트
						wf.RunCount++
						wf.LastRun = time.Now().Format(time.RFC3339)
						saveWorkflow(wf)
					})
			}
		}
	}
}

// ── HTTP 핸들러 ─────────────────────────────────────────────────

// GET /api/workflow/list
func handleWorkflowList(w http.ResponseWriter, r *http.Request) {
	workflows := loadAllWorkflows()
	json200(w, map[string]any{"workflows": workflows, "count": len(workflows)})
}

// POST /api/workflow/save
func handleWorkflowSave(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var wf VisualWorkflow
	if err := json.NewDecoder(r.Body).Decode(&wf); err != nil {
		json200(w, map[string]any{"success": false, "message": msgT("형식 오류", "Invalid format", lang)})
		return
	}
	if wf.ID == "" {
		wf.ID = fmt.Sprintf("wf_%d", time.Now().UnixNano())
	}
	now := time.Now().Format(time.RFC3339)
	if wf.CreatedAt == "" {
		wf.CreatedAt = now
	}
	wf.UpdatedAt = now

	if err := saveWorkflow(wf); err != nil {
		json200(w, map[string]any{"success": false, "message": err.Error()})
		return
	}
	json200(w, map[string]any{"success": true, "id": wf.ID, "message": msgT("워크플로우 저장 완료", "Workflow saved", lang)})
}

// DELETE /api/workflow/delete — query: ?id=wf_xxx
func handleWorkflowDelete(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	id := r.URL.Query().Get("id")
	if id == "" {
		json200(w, map[string]any{"success": false, "message": msgT("id가 필요합니다", "id is required", lang)})
		return
	}
	if err := deleteWorkflow(id); err != nil {
		json200(w, map[string]any{"success": false, "message": err.Error()})
		return
	}
	json200(w, map[string]any{"success": true, "message": msgT("워크플로우 삭제 완료", "Workflow deleted", lang)})
}

// POST /api/workflow/run-now — body: {id: "wf_xxx"}
func handleWorkflowRunNow(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct{ ID string `json:"id"` }
	tryDecodeBody(r, &req)
	workflows := loadAllWorkflows()
	var target *VisualWorkflow
	for i, wf := range workflows {
		if wf.ID == req.ID {
			target = &workflows[i]
			break
		}
	}

	if target == nil {
		json200(w, map[string]any{"success": false, "message": msgT("워크플로우를 찾을 수 없습니다", "Workflow not found", lang)})
		return
	}

	wf := *target
	task := globalTaskQueue.Enqueue("Workflow: "+wf.Name, PriorityNormal, nil,
		func(t *AgentTask) {
			result := executeWFActions(wf)
			t.Result = map[string]any{"result": result}
			wf.RunCount++
			wf.LastRun = time.Now().Format(time.RFC3339)
			saveWorkflow(wf)
		})

	json200(w, map[string]any{"success": true, "task_id": task.ID, "message": msgT("워크플로우 실행 시작", "Workflow execution started", lang)})
}

// POST /api/workflow/from-text — 자연어로 워크플로우 생성
func handleWorkflowFromText(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct{ Text string `json:"text"` }
	tryDecodeBody(r, &req)
	if req.Text == "" {
		json200(w, map[string]any{"success": false, "message": msgT("텍스트가 필요합니다", "text is required", lang)})
		return
	}

	llmMu.RLock()
	_ = llmPerplexityKey
	llmMu.RUnlock()

	sysMsg := `Convert this natural language workflow description into a structured workflow JSON.
Return JSON only with this structure:
{
  "name": "<workflow name>",
  "description": "<description>",
  "trigger": {"type": "schedule|condition|manual", "value": "<time HH:MM or condition>", "label": "<label>"},
  "conditions": [{"field": "cpu|memory|disk|time", "operator": "gt|lt|eq", "value": "<threshold>"}],
  "actions": [
    {"id": "a1", "type": "api_call|notification|desktop_agent|multi_agent|email", "label": "<label>", "goal": "<for agent types>", "next_id": "a2"}
  ],
  "notification": {"channel": "bubble", "message": "<completion message>"}
}`

	raw, _, err := callGroqWithFallback([]groqMsg{
		{Role: "system", Content: sysMsg},
		{Role: "user", Content: req.Text},
	}, 500, true)

	if err != nil {
		json200(w, map[string]any{"success": false, "message": msgT("워크플로우 생성 실패", "Workflow generation failed", lang)})
		return
	}

	// JSON 추출
	clean := strings.TrimSpace(raw)
	if idx := strings.Index(clean, "{"); idx >= 0 {
		clean = clean[idx:]
	}
	if idx := strings.LastIndex(clean, "}"); idx >= 0 {
		clean = clean[:idx+1]
	}

	var wf VisualWorkflow
	if err := json.Unmarshal([]byte(clean), &wf); err != nil {
		json200(w, map[string]any{"success": false, "message": msgT("생성된 워크플로우 파싱 실패", "Failed to parse generated workflow", lang)})
		return
	}

	wf.ID = fmt.Sprintf("wf_%d", time.Now().UnixNano())
	wf.Enabled = false // 기본 비활성화, 사용자 확인 후 활성화
	wf.CreatedAt = time.Now().Format(time.RFC3339)
	wf.UpdatedAt = wf.CreatedAt

	json200(w, map[string]any{
		"success":  true,
		"workflow": wf,
		"message":  fmt.Sprintf(msgT("'%s' 워크플로우가 생성됐습니다. 저장 전 내용을 확인해주세요.", "'%s' workflow created. Please review before saving.", lang), wf.Name),
	})
}

// GET /api/workflow/templates — 기본 템플릿 목록
func handleWorkflowTemplates(w http.ResponseWriter, r *http.Request) {
	templates := []map[string]any{
		// ── 일상/생산성 ──────────────────────────────────
		{
			"id": "tpl_morning", "name": "매일 아침 브리핑",
			"description": "매일 오전 8시 날씨·뉴스·일정·PC 상태 자동 브리핑",
			"category": "productivity",
			"trigger": WFTrigger{Type: "schedule", Value: "0 8 * * *", Label: "매일 오전 8시"},
			"actions": []WFAction{
				{ID: "a1", Type: "api_call", Label: "브리핑 생성", Endpoint: "/api/briefing/now", Method: "POST"},
				{ID: "a2", Type: "notification", Label: "브리핑 알림", Params: map[string]any{"message": "오늘의 AI 브리핑이 준비됐습니다!"}},
			},
		},
		{
			"id": "tpl_evening_summary", "name": "저녁 업무 일지",
			"description": "매일 오후 6시 오늘 작업 정리 및 내일 일정 확인",
			"category": "productivity",
			"trigger": WFTrigger{Type: "schedule", Value: "0 18 * * 1-5", Label: "평일 오후 6시"},
			"actions": []WFAction{
				{ID: "a1", Type: "api_call", Label: "오늘 일지 생성", Endpoint: "/api/journal/generate", Method: "POST"},
				{ID: "a2", Type: "api_call", Label: "내일 일정 확인", Endpoint: "/api/calendar/today", Method: "GET"},
			},
		},
		{
			"id": "tpl_weekly_report", "name": "주간 리포트 이메일",
			"description": "매주 금요일 오후 5시 PC 리포트 자동 발송",
			"category": "report",
			"trigger": WFTrigger{Type: "schedule", Value: "0 17 * * 5", Label: "매주 금요일 오후 5시"},
			"actions": []WFAction{
				{ID: "a1", Type: "api_call", Label: "리포트 생성", Endpoint: "/api/report/generate", Method: "GET"},
				{ID: "a2", Type: "email", Label: "이메일 발송", Params: map[string]any{"subject": "주간 PC 리포트", "body": "이번 주 PC 상태 리포트입니다."}},
			},
		},
		{
			"id": "tpl_meeting_prep", "name": "미팅 자동 준비",
			"description": "미팅 1시간 전 관련 자료·이메일·의제 자동 수집",
			"category": "meeting",
			"trigger": WFTrigger{Type: "event", Value: "meeting_1hour", Label: "미팅 1시간 전"},
			"actions": []WFAction{
				{ID: "a1", Type: "api_call", Label: "받은 메일 확인", Endpoint: "/api/email/inbox", Method: "GET"},
				{ID: "a2", Type: "multi_agent", Label: "자료 수집·요약", Goal: "미팅 참석자·의제 관련 자료 수집 및 핵심 3줄 요약"},
				{ID: "a3", Type: "notification", Label: "준비 완료 알림", Params: map[string]any{"message": "미팅 자료 준비 완료!"}},
			},
		},
		// ── PC 관리/보안 ──────────────────────────────────
		{
			"id": "tpl_pc_optimize", "name": "PC 자동 최적화",
			"description": "메모리 90% 초과 시 자동 정리 + 알림",
			"category": "system",
			"trigger": WFTrigger{Type: "condition", Value: "memory>90", Label: "메모리 90% 초과"},
			"conditions": []WFCondition{{Field: "memory", Operator: "gt", Value: "90"}},
			"actions": []WFAction{
				{ID: "a1", Type: "api_call", Label: "캐시 정리", Endpoint: "/api/clean", Method: "POST"},
				{ID: "a2", Type: "notification", Label: "완료 알림", Params: map[string]any{"message": "PC 자동 최적화 완료!"}},
			},
		},
		{
			"id": "tpl_daily_scan", "name": "야간 보안 스캔",
			"description": "매일 새벽 2시 자동 보안 스캔 + 이상 시 알림",
			"category": "security",
			"trigger": WFTrigger{Type: "schedule", Value: "0 2 * * *", Label: "매일 새벽 2시"},
			"actions": []WFAction{
				{ID: "a1", Type: "api_call", Label: "보안 스캔", Endpoint: "/api/scan", Method: "POST"},
				{ID: "a2", Type: "api_call", Label: "감사 로그", Endpoint: "/api/security/audit", Method: "GET"},
				{ID: "a3", Type: "notification", Label: "스캔 결과 알림", Params: map[string]any{"message": "야간 보안 스캔 완료"}},
			},
		},
		{
			"id": "tpl_disk_alert", "name": "디스크 여유 경보",
			"description": "디스크 여유 공간 10% 미만 시 자동 정리 제안",
			"category": "system",
			"trigger": WFTrigger{Type: "condition", Value: "disk<10", Label: "디스크 10% 미만"},
			"conditions": []WFCondition{{Field: "disk", Operator: "lt", Value: "10"}},
			"actions": []WFAction{
				{ID: "a1", Type: "api_call", Label: "큰 파일 탐색", Endpoint: "/api/file/large", Method: "POST", Params: map[string]any{"min_mb": 500}},
				{ID: "a2", Type: "api_call", Label: "중복 파일 탐색", Endpoint: "/api/files/duplicates", Method: "POST"},
				{ID: "a3", Type: "notification", Label: "경보 알림", Params: map[string]any{"message": "⚠️ 디스크 여유 공간 부족!"}},
			},
		},
		// ── 이메일/커뮤니케이션 ──────────────────────────
		{
			"id": "tpl_email_digest", "name": "오전 이메일 요약",
			"description": "매일 오전 9시 받은 메일 AI 요약 + 중요 메일 알림",
			"category": "email",
			"trigger": WFTrigger{Type: "schedule", Value: "0 9 * * 1-5", Label: "평일 오전 9시"},
			"actions": []WFAction{
				{ID: "a1", Type: "api_call", Label: "메일 수집", Endpoint: "/api/email/inbox", Method: "GET"},
				{ID: "a2", Type: "api_call", Label: "메일 요약", Endpoint: "/api/email/summarize", Method: "POST"},
				{ID: "a3", Type: "notification", Label: "요약 알림", Params: map[string]any{"message": "오늘 받은 메일 요약 준비됐어요!"}},
			},
		},
		{
			"id": "tpl_vip_alert", "name": "VIP 메일 즉시 알림",
			"description": "특정 발신자 메일 수신 시 즉시 알림",
			"category": "email",
			"trigger": WFTrigger{Type: "event", Value: "email_received", Label: "메일 수신 시"},
			"conditions": []WFCondition{{Field: "email_from", Operator: "contains", Value: "vip@company.com"}},
			"actions": []WFAction{
				{ID: "a1", Type: "notification", Label: "VIP 메일 알림", Params: map[string]any{"message": "⭐ VIP 발신자로부터 메일이 도착했습니다!"}},
				{ID: "a2", Type: "api_call", Label: "메일 요약", Endpoint: "/api/email/summarize", Method: "POST"},
			},
		},
		// ── 재무/비즈니스 ─────────────────────────────────
		{
			"id": "tpl_stock_alert", "name": "주가 모니터링",
			"description": "매일 장 마감 후 관심 종목 시세 + 뉴스 요약",
			"category": "finance",
			"trigger": WFTrigger{Type: "schedule", Value: "0 16 * * 1-5", Label: "평일 오후 4시 (장마감)"},
			"actions": []WFAction{
				{ID: "a1", Type: "api_call", Label: "주가 조회", Endpoint: "/api/finance/stock", Method: "POST", Params: map[string]any{"ticker": "005930"}},
				{ID: "a2", Type: "api_call", Label: "관련 뉴스", Endpoint: "/api/command", Method: "POST", Params: map[string]any{"message": "삼성전자 오늘 주요 뉴스"}},
			},
		},
		{
			"id": "tpl_tax_reminder", "name": "세금 납부 알림",
			"description": "부가세·소득세 신고 마감 7일 전 자동 리마인더",
			"category": "finance",
			"trigger": WFTrigger{Type: "schedule", Value: "0 9 18 1,4,7,10 *", Label: "분기별 세금 신고 7일 전"},
			"actions": []WFAction{
				{ID: "a1", Type: "notification", Label: "세금 알림", Params: map[string]any{"message": "⚠️ 부가세 신고 마감 7일 전입니다!"}},
				{ID: "a2", Type: "api_call", Label: "관련 정보 검색", Endpoint: "/api/command", Method: "POST", Params: map[string]any{"message": "이번 분기 부가세 신고 일정과 준비 서류"}},
			},
		},
		// ── 콘텐츠/크리에이터 ────────────────────────────
		{
			"id": "tpl_content_research", "name": "콘텐츠 트렌드 수집",
			"description": "매일 오전 유튜브·틱톡 트렌딩 + AI 뉴스 자동 수집",
			"category": "content",
			"trigger": WFTrigger{Type: "schedule", Value: "0 7 * * *", Label: "매일 오전 7시"},
			"actions": []WFAction{
				{ID: "a1", Type: "api_call", Label: "유튜브 트렌딩", Endpoint: "/api/video/quick-search", Method: "GET", Params: map[string]any{"query": "AI 트렌드"}},
				{ID: "a2", Type: "api_call", Label: "틱톡 트렌딩", Endpoint: "/api/tiktok/trending", Method: "GET"},
				{ID: "a3", Type: "api_call", Label: "AI 뉴스", Endpoint: "/api/command", Method: "POST", Params: map[string]any{"message": "오늘 AI 트렌드 뉴스 3줄 요약"}},
				{ID: "a4", Type: "api_call", Label: "노트 저장", Endpoint: "/api/notes", Method: "POST"},
			},
		},
		{
			"id": "tpl_auto_subtitle", "name": "유튜브 영상 자막 추출",
			"description": "유튜브 URL 입력 시 자막 추출 → 요약 → 노트 저장",
			"category": "content",
			"trigger": WFTrigger{Type: "manual", Value: "", Label: "수동 실행 (URL 입력)"},
			"actions": []WFAction{
				{ID: "a1", Type: "api_call", Label: "자막 추출", Endpoint: "/api/video/transcript", Method: "POST"},
				{ID: "a2", Type: "api_call", Label: "핵심 요약", Endpoint: "/api/llm/doc-summary", Method: "POST"},
				{ID: "a3", Type: "api_call", Label: "노트 저장", Endpoint: "/api/notes", Method: "POST"},
			},
		},
		// ── 개발/기술 ─────────────────────────────────────
		{
			"id": "tpl_github_digest", "name": "GitHub 트렌딩 일일 정리",
			"description": "매일 오전 GitHub 트렌딩 저장소 + HackerNews 상위 글 수집",
			"category": "developer",
			"trigger": WFTrigger{Type: "schedule", Value: "0 8 * * 1-5", Label: "평일 오전 8시"},
			"actions": []WFAction{
				{ID: "a1", Type: "api_call", Label: "GitHub 트렌딩", Endpoint: "/api/command", Method: "POST", Params: map[string]any{"message": "GitHub 오늘 트렌딩 저장소 TOP 5", "persona": "developer"}},
				{ID: "a2", Type: "api_call", Label: "노트 저장", Endpoint: "/api/notes", Method: "POST"},
			},
		},
		{
			"id": "tpl_backup_reminder", "name": "중요 파일 백업 알림",
			"description": "매주 일요일 오후 중요 폴더 백업 리마인더",
			"category": "system",
			"trigger": WFTrigger{Type: "schedule", Value: "0 15 * * 0", Label: "매주 일요일 오후 3시"},
			"actions": []WFAction{
				{ID: "a1", Type: "api_call", Label: "디스크 상태 확인", Endpoint: "/api/stats", Method: "GET"},
				{ID: "a2", Type: "notification", Label: "백업 알림", Params: map[string]any{"message": "📦 이번 주 중요 파일 백업을 잊지 마세요!"}},
			},
		},
	}

	json200(w, map[string]any{"templates": templates, "count": len(templates)})
}
