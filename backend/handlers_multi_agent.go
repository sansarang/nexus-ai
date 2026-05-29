//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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
	eng := isEnglishQuery(goal)
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

	raw, _, err := callGroqWithFallback([]groqMsg{
		{Role: "system", Content: sysMsg},
		{Role: "user", Content: "Goal: " + goal},
	}, 600, true)
	if err != nil {
		msg := "오케스트레이터 실패: " + err.Error()
		if eng { msg = "Orchestrator failed: " + err.Error() }
		return AgentPlan{}, fmt.Errorf("%s", msg)
	}

	clean := strings.TrimSpace(raw)
	if idx := strings.Index(clean, "{"); idx >= 0 {
		clean = clean[idx:]
	}
	if idx := strings.LastIndex(clean, "}"); idx >= 0 {
		clean = clean[:idx+1]
	}

	var plan AgentPlan
	if err := json.Unmarshal([]byte(clean), &plan); err != nil {
		msg := "플랜 파싱 실패: " + err.Error()
		if eng { msg = "Plan parsing failed: " + err.Error() }
		return AgentPlan{}, fmt.Errorf("%s", msg)
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
	stepEng := isEnglishQuery(step.SubGoal)
	if agentDef == nil {
		if stepEng { return fmt.Sprintf("[%s] agent not found", step.Agent) }
		return fmt.Sprintf("[%s] 에이전트를 찾을 수 없습니다", step.Agent)
	}

	// 이전 결과 컨텍스트 구성
	context := ""
	for _, dep := range step.DependsOn {
		if res, ok := previousResults[dep]; ok {
			if stepEng {
				context += fmt.Sprintf("\n[Step %d result]: %s", dep, res)
			} else {
				context += fmt.Sprintf("\n[Step %d 결과]: %s", dep, res)
			}
		}
	}

	// 에이전트별 실제 실행
	switch step.Agent {
	case "ResearchAgent":
		result := parallelWebSearch(step.SubGoal, 5)
		return result.Summary

	case "FileAgent":
		return executeFileAgent(step.SubGoal, stepEng)

	case "OptimizerAgent":
		mem := getMemoryUsage()
		free, total := getDiskSpace()
		diskPct := 0.0
		if total > 0 {
			diskPct = 100 - float64(free)/float64(total)*100
		}
		if stepEng { return fmt.Sprintf("PC status: Memory %d%% used, Disk %.0f%% used", mem, diskPct) }
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
		ans, _, err := callGroqWithFallback([]groqMsg{
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
		eng := isEnglishQuery(plan.Goal)
		var synthSys, synthUser string
		if eng {
			synthSys = "You are Nexus AI. Synthesize the results from multiple agents into a clear, concise English summary for the user."
			synthUser = fmt.Sprintf("Goal: %s\n\nAgent results:\n%s\n\nSummarize these results for the user in natural English.", plan.Goal, combined)
		} else {
			synthSys = "You are Nexus AI. Synthesize the results from multiple agents into a clear, concise Korean summary for the user."
			synthUser = fmt.Sprintf("목표: %s\n\n에이전트 결과:\n%s\n\n위 결과를 종합해서 사용자에게 자연스러운 한국어로 보고해주세요.", plan.Goal, combined)
		}
		summary, _, err := callGroqWithFallback([]groqMsg{
			{Role: "system", Content: synthSys},
			{Role: "user", Content: synthUser},
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

// executeFileAgent: 파일 관련 서브골을 파싱해 실제 작업 실행
func executeFileAgent(subGoal string, eng bool) string {
	lower := strings.ToLower(subGoal)

	// ── 파일/폴더 목록 조회 ──────────────────────────────────────
	if strings.Contains(lower, "list") || strings.Contains(lower, "목록") ||
		strings.Contains(lower, "찾") || strings.Contains(lower, "find") || strings.Contains(lower, "search") {
		// 경로 추출: 따옴표 또는 마지막 공백 이후 경로처럼 보이는 부분
		searchPath := extractPathFromText(subGoal)
		if searchPath == "" {
			searchPath = os.Getenv("USERPROFILE")
			if searchPath == "" {
				searchPath = "C:\\Users"
			}
		}
		entries, err := os.ReadDir(searchPath)
		if err != nil {
			if eng { return fmt.Sprintf("Cannot read directory '%s': %v", searchPath, err) }
			return fmt.Sprintf("'%s' 폴더를 읽을 수 없습니다: %v", searchPath, err)
		}
		var items []string
		for i, e := range entries {
			if i >= 20 { items = append(items, "..."); break }
			kind := "파일"
			if e.IsDir() { kind = "📁" } else { kind = "📄" }
			items = append(items, fmt.Sprintf("%s %s", kind, e.Name()))
		}
		if eng { return fmt.Sprintf("Contents of '%s' (%d items):\n%s", searchPath, len(entries), strings.Join(items, "\n")) }
		return fmt.Sprintf("'%s' 내용 (%d개):\n%s", searchPath, len(entries), strings.Join(items, "\n"))
	}

	// ── 폴더 생성 ───────────────────────────────────────────────
	if strings.Contains(lower, "create") || strings.Contains(lower, "make") ||
		strings.Contains(lower, "생성") || strings.Contains(lower, "만들") {
		targetPath := extractPathFromText(subGoal)
		if targetPath == "" {
			if eng { return "Folder path is required." }
			return "생성할 폴더 경로가 필요합니다."
		}
		if err := os.MkdirAll(targetPath, 0755); err != nil {
			if eng { return fmt.Sprintf("Failed to create folder '%s': %v", targetPath, err) }
			return fmt.Sprintf("폴더 생성 실패 '%s': %v", targetPath, err)
		}
		if eng { return fmt.Sprintf("✅ Folder created: %s", targetPath) }
		return fmt.Sprintf("✅ 폴더 생성 완료: %s", targetPath)
	}

	// ── 파일 삭제 ───────────────────────────────────────────────
	if strings.Contains(lower, "delete") || strings.Contains(lower, "remove") ||
		strings.Contains(lower, "삭제") || strings.Contains(lower, "지워") {
		targetPath := extractPathFromText(subGoal)
		if targetPath == "" {
			if eng { return "File/folder path is required." }
			return "삭제할 파일/폴더 경로가 필요합니다."
		}
		if err := os.Remove(targetPath); err != nil {
			if eng { return fmt.Sprintf("Delete failed '%s': %v", targetPath, err) }
			return fmt.Sprintf("삭제 실패 '%s': %v", targetPath, err)
		}
		if eng { return fmt.Sprintf("✅ Deleted: %s", targetPath) }
		return fmt.Sprintf("✅ 삭제 완료: %s", targetPath)
	}

	// ── 파일 읽기 ───────────────────────────────────────────────
	if strings.Contains(lower, "read") || strings.Contains(lower, "open") ||
		strings.Contains(lower, "읽") || strings.Contains(lower, "내용") {
		targetPath := extractPathFromText(subGoal)
		if targetPath == "" {
			if eng { return "File path is required." }
			return "읽을 파일 경로가 필요합니다."
		}
		data, err := os.ReadFile(targetPath)
		if err != nil {
			if eng { return fmt.Sprintf("Cannot read file '%s': %v", targetPath, err) }
			return fmt.Sprintf("파일 읽기 실패 '%s': %v", targetPath, err)
		}
		content := string(data)
		if len(content) > 2000 { content = content[:2000] + "\n...(생략)" }
		return content
	}

	// ── 폴백: LLM에게 위임 ──────────────────────────────────────
	prompt := fmt.Sprintf("파일 작업 요청: %s\n사용자 PC의 파일 시스템에서 할 수 있는 최선의 도움말을 한국어로 알려줘.", subGoal)
	if eng { prompt = fmt.Sprintf("File operation request: %s\nProvide the best helpful guidance in English.", subGoal) }
	msgs := []groqMsg{{Role: "user", Content: prompt}}
	result, _, _ := callGroqWithFallback(msgs, 500, true)
	if result == "" {
		if eng { return fmt.Sprintf("File Agent: '%s' — please specify exact file path.", subGoal) }
		return fmt.Sprintf("파일 에이전트: '%s' — 정확한 파일 경로를 지정해주세요.", subGoal)
	}
	return result
}

// extractPathFromText: 텍스트에서 파일/폴더 경로 추출
func extractPathFromText(text string) string {
	// 따옴표 안 경로
	for _, q := range []string{`"`, `'`, "`"} {
		if start := strings.Index(text, q); start >= 0 {
			rest := text[start+1:]
			if end := strings.Index(rest, q); end >= 0 {
				return rest[:end]
			}
		}
	}
	// C:\ 또는 %로 시작하는 Windows 경로
	for _, word := range strings.Fields(text) {
		if len(word) > 2 && (word[1] == ':' || strings.HasPrefix(word, "%")) {
			return word
		}
	}
	return ""
}

// ── HTTP 핸들러 ─────────────────────────────────────────────────

// POST /api/agent/multi/run — body: {goal: "경쟁사 분석 리포트 작성"}
func handleMultiAgentRun(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Goal string `json:"goal"`
	}
	lang := getLang(r)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json200(w, map[string]any{"success": false, "message": msgT("잘못된 요청", "Invalid request", lang)})
		return
	}
	if req.Goal == "" {
		json200(w, map[string]any{"success": false, "message": msgT("goal이 필요합니다", "goal is required", lang)})
		return
	}

	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()

	if gKey == "" {
		json200(w, map[string]any{"success": false, "message": msgT("Groq API 키가 필요합니다", "Groq API key is required", lang)})
		return
	}

	task := globalTaskQueue.Enqueue("Multi-Agent: "+req.Goal, PriorityNormal, map[string]any{"goal": req.Goal},
		func(t *AgentTask) {
			t.UpdateProgress(5, msgT("목표 분석 중...", "Analyzing goal...", lang))
			plan, err := orchestrate(req.Goal, gKey)
			if err != nil {
				t.Status = TaskFailed
				t.Error = err.Error()
				return
			}
			t.UpdateProgress(10, msgT(fmt.Sprintf("에이전트 %d개 배치 완료", len(plan.Steps)), fmt.Sprintf("%d agents assigned", len(plan.Steps)), lang))
			runMultiAgentPlan(t, plan, gKey)
		},
	)

	json200(w, map[string]any{
		"success": true,
		"task_id": task.ID,
		"message": msgT(fmt.Sprintf("Multi-Agent 시작: '%s'", req.Goal), fmt.Sprintf("Multi-Agent started: '%s'", req.Goal), lang),
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
	lang2 := getLang(r)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json200(w, map[string]any{"success": false, "message": msgT("잘못된 요청", "Invalid request", lang2)})
		return
	}
	if req.Goal == "" {
		json200(w, map[string]any{"success": false, "message": msgT("goal이 필요합니다", "goal is required", lang2)})
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
	lang3 := getLang(r)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json200(w, map[string]any{"success": false, "message": msgT("잘못된 요청", "Invalid request", lang3)})
		return
	}
	if req.Goal == "" {
		json200(w, map[string]any{"success": false, "message": msgT("goal이 필요합니다", "goal is required", lang3)})
		return
	}

	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()

	task := globalTaskQueue.Enqueue("Multi-Agent: "+req.Goal, PriorityNormal,
		map[string]any{"goal": req.Goal, "agents": req.Agents},
		func(t *AgentTask) {
			t.UpdateProgress(5, msgT("목표 분석 중...", "Analyzing goal...", lang3))
			plan, err := orchestrate(req.Goal, gKey)
			if err != nil {
				t.Status = TaskFailed
				t.Error = err.Error()
				return
			}
			t.UpdateProgress(10, msgT(fmt.Sprintf("에이전트 %d개 배치 완료", len(plan.Steps)), fmt.Sprintf("%d agents assigned", len(plan.Steps)), lang3))
			runMultiAgentPlan(t, plan, gKey)
		},
	)

	json200(w, map[string]any{
		"success": true,
		"task_id": task.ID,
		"message": msgT(fmt.Sprintf("Multi-Agent 시작: '%s'", req.Goal), fmt.Sprintf("Multi-Agent started: '%s'", req.Goal), lang3),
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
