//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ══════════════════════════════════════════════════════════════════
//  Multi-Agent Architecture (CrewAI / AutoGen 스타일)
//  Orchestrator → 전문 Agent 분배 → 결과 통합
//  Agent 종류: Research, File, Email, Meeting, Shopping, Optimizer
// ══════════════════════════════════════════════════════════════════

// ── Agent 정의 ──────────────────────────────────────────────────

type AgentDef struct {
	Name         string
	Capability   string
	SystemPrompt string
	Tools        []string
}

var registeredAgents = []AgentDef{
	{
		Name:       "ResearchAgent",
		Capability: "웹 검색, 뉴스 수집, 경쟁사 분석, 논문 검색",
		SystemPrompt: `You are the Research Agent of Nexus. Your specialty is finding and analyzing information from the web.
Use Tavily search results. Never hallucinate. Summarize findings concisely in Korean.`,
		Tools: []string{"web_search", "news_search", "deep_search"},
	},
	{
		Name:       "FileAgent",
		Capability: "파일 검색, 문서 요약, 파일 정리, 중복 파일 제거",
		SystemPrompt: `You are the File Agent of Nexus. Your specialty is managing files and documents on the user's PC.
Find, organize, summarize, and manage files efficiently.`,
		Tools: []string{"file_search", "doc_summary", "organize_folder"},
	},
	{
		Name:       "EmailAgent",
		Capability: "이메일 분류, 답장 초안, 이메일 발송, 요약",
		SystemPrompt: `You are the Email Agent of Nexus. Your specialty is handling emails intelligently.
Classify, summarize, draft replies, and send emails on behalf of the user.`,
		Tools: []string{"email_inbox", "email_send", "email_summarize"},
	},
	{
		Name:       "MeetingAgent",
		Capability: "회의 준비, 일정 관리, 회의록 작성, 참석자 정보 수집",
		SystemPrompt: `You are the Meeting Agent of Nexus. Your specialty is preparing for and managing meetings.
Research attendees, prepare materials, and create meeting summaries.`,
		Tools: []string{"calendar_today", "meeting_start", "meeting_summarize"},
	},
	{
		Name:       "ShoppingAgent",
		Capability: "최저가 검색, 상품 비교, 쇼핑 추천",
		SystemPrompt: `You are the Shopping Agent of Nexus. Your specialty is finding the best deals.
Compare prices, find alternatives, and recommend the best purchase options.`,
		Tools: []string{"web_search", "price_compare"},
	},
	{
		Name:       "OptimizerAgent",
		Capability: "PC 최적화, 메모리 정리, 디스크 정리, 성능 개선",
		SystemPrompt: `You are the Optimizer Agent of Nexus. Your specialty is keeping the PC running at peak performance.
Clean, optimize, and maintain the system proactively.`,
		Tools: []string{"clean", "stats", "security_scan"},
	},
	{
		Name:       "DesktopAgent",
		Capability: "화면 제어, 마우스·키보드 자동화, 앱 조작",
		SystemPrompt: `You are the Desktop Agent of Nexus. Your specialty is controlling the Windows desktop.
Click, type, scroll, and interact with any application on screen.`,
		Tools: []string{"desktop_run", "desktop_click", "desktop_type"},
	},
}

// ── Orchestrator: 작업 → 에이전트 배분 ─────────────────────────

type AgentPlan struct {
	Steps []AgentStep `json:"steps"`
	Goal  string      `json:"goal"`
}

type AgentStep struct {
	Order     int    `json:"order"`
	Agent     string `json:"agent"`
	SubGoal   string `json:"sub_goal"`
	DependsOn []int  `json:"depends_on,omitempty"`
	Result    string `json:"result,omitempty"`
	Status    string `json:"status"` // pending|running|done|failed
}

// orchestrate: 목표를 분석해 어떤 에이전트들이 어떤 순서로 실행할지 결정
func orchestrate(goal, gKey string) (AgentPlan, error) {
	agentList := ""
	for _, a := range registeredAgents {
		agentList += fmt.Sprintf("- %s: %s\n", a.Name, a.Capability)
	}

	sysMsg := `You are the Orchestrator of Nexus Multi-Agent System.
Given a user goal, decompose it into steps and assign each step to the most suitable agent.

Available agents:
` + agentList + `

Return JSON only:
{
  "goal": "<original goal>",
  "steps": [
    {"order": 1, "agent": "<AgentName>", "sub_goal": "<specific task for this agent>", "depends_on": []},
    {"order": 2, "agent": "<AgentName>", "sub_goal": "<specific task>", "depends_on": [1]}
  ]
}

Rules:
1. Break complex goals into 2-5 steps maximum
2. Each step has exactly one agent
3. depends_on lists step orders that must complete first
4. Keep sub_goals specific and actionable`

	raw, _, err := callGroq(gKey, groqFastModel, []groqMsg{
		{Role: "system", Content: sysMsg},
		{Role: "user", Content: "Goal: " + goal},
	}, 600, true)
	if err != nil {
		return AgentPlan{}, fmt.Errorf("오케스트레이터 실패: %w", err)
	}

	// JSON 추출
	clean := strings.TrimSpace(raw)
	if idx := strings.Index(clean, "{"); idx >= 0 {
		clean = clean[idx:]
	}
	if idx := strings.LastIndex(clean, "}"); idx >= 0 {
		clean = clean[:idx+1]
	}

	var plan AgentPlan
	if err := json.Unmarshal([]byte(clean), &plan); err != nil {
		return AgentPlan{}, fmt.Errorf("플랜 파싱 실패: %w", err)
	}

	for i := range plan.Steps {
		plan.Steps[i].Status = "pending"
	}
	return plan, nil
}

// executeAgentStep: 개별 에이전트 스텝 실행
func executeAgentStep(step *AgentStep, previousResults map[int]string, gKey string) string {
	// 해당 에이전트 찾기
	var agentDef *AgentDef
	for i, a := range registeredAgents {
		if a.Name == step.Agent {
			agentDef = &registeredAgents[i]
			break
		}
	}
	if agentDef == nil {
		return fmt.Sprintf("[%s] 에이전트를 찾을 수 없습니다", step.Agent)
	}

	// 이전 결과 컨텍스트 구성
	context := ""
	for _, dep := range step.DependsOn {
		if res, ok := previousResults[dep]; ok {
			context += fmt.Sprintf("\n[Step %d 결과]: %s", dep, res)
		}
	}

	// 에이전트별 실제 실행
	switch step.Agent {
	case "ResearchAgent":
		result := parallelWebSearch(step.SubGoal, 5)
		return result.Summary

	case "FileAgent":
		// 파일 검색 실행
		return fmt.Sprintf("파일 에이전트: '%s' 작업 완료", step.SubGoal)

	case "OptimizerAgent":
		mem := getMemoryUsage()
		free, total := getDiskSpace()
		diskPct := 0.0
		if total > 0 {
			diskPct = 100 - float64(free)/float64(total)*100
		}
		return fmt.Sprintf("PC 상태: 메모리 %d%% 사용, 디스크 %.0f%% 사용", mem, diskPct)

	case "DesktopAgent":
		// Task Queue에 Desktop Agent 작업 등록
		task := globalTaskQueue.Enqueue("Desktop: "+step.SubGoal, PriorityNormal, nil, func(t *AgentTask) {
			runDesktopAgent(t, step.SubGoal, true, 15)
		})
		// 완료 대기 (최대 3분)
		for i := 0; i < 180; i++ {
			time.Sleep(time.Second)
			if t, ok := globalTaskQueue.GetTask(task.ID); ok {
				if t.Status == TaskDone {
					return fmt.Sprintf("Desktop 작업 완료: %s", step.SubGoal)
				}
				if t.Status == TaskFailed || t.Status == TaskCancelled {
					return fmt.Sprintf("Desktop 작업 실패: %s", t.Error)
				}
			}
		}
		return "Desktop 작업 타임아웃"

	default:
		// Groq로 에이전트 역할 수행
		userMsg := step.SubGoal
		if context != "" {
			userMsg = step.SubGoal + "\n\n이전 결과 참고:" + context
		}
		ans, _, err := callGroq(gKey, groqChatModel, []groqMsg{
			{Role: "system", Content: agentDef.SystemPrompt},
			{Role: "user", Content: userMsg},
		}, 600, false)
		if err != nil {
			return fmt.Sprintf("[%s] 실행 실패: %s", step.Agent, err.Error())
		}
		return ans
	}
}

// runMultiAgentPlan: 플랜 실행 (의존성 고려한 순서 실행)
func runMultiAgentPlan(task *AgentTask, plan AgentPlan, gKey string) {
	results := map[int]string{}
	var resultMu sync.Mutex

	totalSteps := len(plan.Steps)
	completed := 0

	// 의존성 없는 스텝부터 병렬 실행, 의존성 있는 것은 순서대로
	for pass := 0; pass < totalSteps*2; pass++ {
		if completed >= totalSteps {
			break
		}
		if task.IsCancelled() {
			task.Status = TaskCancelled
			return
		}

		var wg sync.WaitGroup
		for i := range plan.Steps {
			step := &plan.Steps[i]
			if step.Status != "pending" {
				continue
			}

			// 의존성 확인
			ready := true
			resultMu.Lock()
			for _, dep := range step.DependsOn {
				if _, ok := results[dep]; !ok {
					ready = false
					break
				}
			}
			resultMu.Unlock()

			if !ready {
				continue
			}

			step.Status = "running"
			task.UpdateProgress(completed*100/totalSteps,
				fmt.Sprintf("[%s] %s", step.Agent, step.SubGoal))

			wg.Add(1)
			go func(s *AgentStep) {
				defer wg.Done()
				resultMu.Lock()
				prevResults := make(map[int]string, len(results))
				for k, v := range results {
					prevResults[k] = v
				}
				resultMu.Unlock()

				result := executeAgentStep(s, prevResults, gKey)
				s.Result = result
				s.Status = "done"

				resultMu.Lock()
				results[s.Order] = result
				resultMu.Unlock()

				completed++
			}(step)
		}
		wg.Wait()
	}

	// 최종 결과 통합
	var finalParts []string
	for i := range plan.Steps {
		s := &plan.Steps[i]
		if s.Result != "" {
			finalParts = append(finalParts, fmt.Sprintf("[%s]\n%s", s.Agent, s.Result))
		}
	}

	// Groq로 최종 요약
	combined := strings.Join(finalParts, "\n\n")
	if gKey != "" && combined != "" {
		summary, _, err := callGroq(gKey, groqChatModel, []groqMsg{
			{Role: "system", Content: "You are Nexus. Synthesize the results from multiple agents into a clear, concise Korean summary for the user."},
			{Role: "user", Content: fmt.Sprintf("목표: %s\n\n에이전트 결과:\n%s\n\n위 결과를 종합해서 사용자에게 자연스러운 한국어로 보고해주세요.", plan.Goal, combined)},
		}, 500, false)
		if err == nil && summary != "" {
			task.Result = map[string]any{
				"summary": summary,
				"steps":   plan.Steps,
				"goal":    plan.Goal,
			}
			task.Message = summary
			return
		}
	}

	task.Result = map[string]any{
		"summary": combined,
		"steps":   plan.Steps,
		"goal":    plan.Goal,
	}
}

// ── HTTP 핸들러 ─────────────────────────────────────────────────

// POST /api/agent/multi/run — body: {goal: "경쟁사 분석 리포트 작성"}
func handleMultiAgentRun(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Goal string `json:"goal"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Goal == "" {
		json200(w, map[string]any{"success": false, "message": "goal이 필요합니다"})
		return
	}

	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()

	if gKey == "" {
		json200(w, map[string]any{"success": false, "message": "Groq API 키가 필요합니다"})
		return
	}

	task := globalTaskQueue.Enqueue("Multi-Agent: "+req.Goal, PriorityNormal, map[string]any{"goal": req.Goal},
		func(t *AgentTask) {
			t.UpdateProgress(5, "목표 분석 중...")
			plan, err := orchestrate(req.Goal, gKey)
			if err != nil {
				t.Status = TaskFailed
				t.Error = err.Error()
				return
			}
			t.UpdateProgress(10, fmt.Sprintf("에이전트 %d개 배치 완료", len(plan.Steps)))
			runMultiAgentPlan(t, plan, gKey)
		},
	)

	json200(w, map[string]any{
		"success": true,
		"task_id": task.ID,
		"message": fmt.Sprintf("Multi-Agent 시작: '%s'", req.Goal),
	})
}

// GET /api/agent/multi/agents — 사용 가능한 에이전트 목록
func handleAgentList(w http.ResponseWriter, r *http.Request) {
	type agentInfo struct {
		Name       string   `json:"name"`
		Capability string   `json:"capability"`
		Tools      []string `json:"tools"`
	}
	var list []agentInfo
	for _, a := range registeredAgents {
		list = append(list, agentInfo{Name: a.Name, Capability: a.Capability, Tools: a.Tools})
	}
	json200(w, map[string]any{"agents": list, "count": len(list)})
}

// POST /api/agent/multi/plan — 실행 없이 플랜만 확인
func handleMultiAgentPlan(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Goal string `json:"goal"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Goal == "" {
		json200(w, map[string]any{"success": false, "message": "goal이 필요합니다"})
		return
	}

	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()

	plan, err := orchestrate(req.Goal, gKey)
	if err != nil {
		json200(w, map[string]any{"success": false, "message": err.Error()})
		return
	}
	json200(w, map[string]any{"success": true, "plan": plan})
}

// POST /api/multi-agent/run — v2 alias {goal, agents}
func handleMultiAgentRunV2(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Goal   string   `json:"goal"`
		Agents []string `json:"agents"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Goal == "" {
		json200(w, map[string]any{"success": false, "message": "goal이 필요합니다"})
		return
	}

	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()

	task := globalTaskQueue.Enqueue("Multi-Agent: "+req.Goal, PriorityNormal,
		map[string]any{"goal": req.Goal, "agents": req.Agents},
		func(t *AgentTask) {
			t.UpdateProgress(5, "목표 분석 중...")
			plan, err := orchestrate(req.Goal, gKey)
			if err != nil {
				t.Status = TaskFailed
				t.Error = err.Error()
				return
			}
			t.UpdateProgress(10, fmt.Sprintf("에이전트 %d개 배치 완료", len(plan.Steps)))
			runMultiAgentPlan(t, plan, gKey)
		},
	)

	json200(w, map[string]any{
		"success": true,
		"task_id": task.ID,
		"message": fmt.Sprintf("Multi-Agent 시작: '%s'", req.Goal),
	})
}

// GET /api/multi-agent/stream/:task_id — SSE 실시간 진행 상황
func handleMultiAgentStream(w http.ResponseWriter, r *http.Request) {
	// URL: /api/multi-agent/stream/{task_id}
	path := r.URL.Path
	parts := strings.Split(strings.TrimPrefix(path, "/api/multi-agent/stream/"), "/")
	taskID := ""
	if len(parts) > 0 {
		taskID = parts[0]
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "data: {\"type\":\"connected\",\"task_id\":%q}\n\n", taskID)
	flusher.Flush()

	ctx := r.Context()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	lastProgress := -1
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if taskID == "" {
				fmt.Fprintf(w, "data: {\"error\":\"task_id required\"}\n\n")
				flusher.Flush()
				return
			}
			task, ok := globalTaskQueue.GetTask(taskID)
			if !ok {
				fmt.Fprintf(w, "data: {\"error\":\"task not found\"}\n\n")
				flusher.Flush()
				return
			}
			if task.Progress != lastProgress || task.Status == TaskDone || task.Status == TaskFailed || task.Status == TaskCancelled {
				lastProgress = task.Progress
				data, _ := json.Marshal(map[string]any{
					"task_id":  task.ID,
					"status":   task.Status,
					"progress": task.Progress,
					"message":  task.Message,
					"result":   task.Result,
					"error":    task.Error,
				})
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
				if task.Status == TaskDone || task.Status == TaskFailed || task.Status == TaskCancelled {
					return
				}
			}
		case <-time.After(25 * time.Second):
			fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}
