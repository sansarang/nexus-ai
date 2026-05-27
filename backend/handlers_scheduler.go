//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ──────────────────────────────────────────────────────────────
// 자연어 스케줄 파서 + 실행 엔진
// ──────────────────────────────────────────────────────────────

type ScheduledTask struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Command     string    `json:"command"`     // 원래 자연어 명령
	Action      string    `json:"action"`      // 실행할 action 타입
	ActionParams string   `json:"action_params"` // JSON 파라미터
	CronExpr    string    `json:"cron_expr"`   // "0 8 * * *" 형태
	NextRun     time.Time `json:"next_run"`
	LastRun     time.Time `json:"last_run"`
	LastResult  string    `json:"last_result"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
	RunCount    int       `json:"run_count"`
}

type SchedulerStore struct {
	mu    sync.RWMutex
	tasks map[string]*ScheduledTask
	path  string
}

var globalScheduler *SchedulerStore

func initScheduler() {
	storePath := filepath.Join(os.TempDir(), "nexus_scheduler.json")
	globalScheduler = &SchedulerStore{
		tasks: make(map[string]*ScheduledTask),
		path:  storePath,
	}
	globalScheduler.load()
	go globalScheduler.runLoop()
}

func (s *SchedulerStore) load() {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var tasks []*ScheduledTask
	if err := json.Unmarshal(data, &tasks); err != nil {
		return
	}
	for _, t := range tasks {
		s.tasks[t.ID] = t
	}
}

func (s *SchedulerStore) save() {
	s.mu.RLock()
	tasks := make([]*ScheduledTask, 0, len(s.tasks))
	for _, t := range s.tasks {
		tasks = append(tasks, t)
	}
	s.mu.RUnlock()

	data, _ := json.MarshalIndent(tasks, "", "  ")
	os.WriteFile(s.path, data, 0644)
}

// runLoop: 60초마다 실행할 태스크 확인
func (s *SchedulerStore) runLoop() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		s.checkAndRun()
	}
}

func (s *SchedulerStore) checkAndRun() {
	now := time.Now()
	s.mu.RLock()
	var toRun []*ScheduledTask
	for _, t := range s.tasks {
		if t.Active && !t.NextRun.IsZero() && now.After(t.NextRun) {
			toRun = append(toRun, t)
		}
	}
	s.mu.RUnlock()

	for _, t := range toRun {
		go s.executeTask(t)
	}
}

func (s *SchedulerStore) executeTask(t *ScheduledTask) {
	type taskResult struct {
		result string
		err    error
	}
	ch := make(chan taskResult, 1)

	go func() {
		r, e := s.runTaskAction(t)
		ch <- taskResult{r, e}
	}()

	var result string
	var execErr error
	select {
	case res := <-ch:
		result, execErr = res.result, res.err
	case <-time.After(5 * time.Minute):
		execErr = fmt.Errorf("태스크 타임아웃 (5분 초과)")
	}

	s.mu.Lock()
	if st, ok := s.tasks[t.ID]; ok {
		st.LastRun = time.Now()
		st.RunCount++
		if execErr != nil {
			st.LastResult = "오류: " + execErr.Error()
		} else {
			st.LastResult = result
			if len(st.LastResult) > 500 {
				st.LastResult = st.LastResult[:500] + "..."
			}
		}
		st.NextRun = calcNextRun(st.CronExpr, time.Now())
	}
	s.mu.Unlock()
	s.save()

	saveAgentMemory(AgentMemoryEntry{
		ID:        fmt.Sprintf("sched_%s_%d", t.ID, time.Now().Unix()),
		Timestamp: time.Now().Format(time.RFC3339),
		Type:      "scheduled_task",
		Command:   t.Name,
		Result:    result,
		Success:   execErr == nil,
	})
}

func (s *SchedulerStore) runTaskAction(t *ScheduledTask) (string, error) {
	switch t.Action {
	case "browser_agent":
		var params map[string]interface{}
		json.Unmarshal([]byte(t.ActionParams), &params)
		cmd, _ := params["command"].(string)
		return runBrowserAgentTask(cmd)

	case "summarize_emails":
		return runEmailSummaryTask()

	case "weekly_report":
		return runWeeklyReportTask()

	case "pc_report":
		return runPCReportTask()

	case "powershell":
		var params map[string]interface{}
		json.Unmarshal([]byte(t.ActionParams), &params)
		script, _ := params["script"].(string)
		return runPowerShellScript(script)

	case "llm_task":
		var params map[string]interface{}
		json.Unmarshal([]byte(t.ActionParams), &params)
		prompt, _ := params["prompt"].(string)
		return runLLMTask(prompt)

	default:
		return fmt.Sprintf("알 수 없는 액션: %s", t.Action), nil
	}
}

// ──────────────────────────────────────────────────────────────
// 자연어 → 스케줄 파싱
// ──────────────────────────────────────────────────────────────

type ParsedSchedule struct {
	CronExpr string
	TaskName string
	Action   string
	Params   map[string]interface{}
	NextRun  time.Time
}

// parseNaturalSchedule: 자연어를 cron 표현식 + 액션으로 변환
func parseNaturalSchedule(command string, llmKey string) (*ParsedSchedule, error) {
	// LLM으로 파싱
	if llmKey != "" {
		result, err := parseScheduleWithLLM(command, llmKey)
		if err == nil {
			return result, nil
		}
	}
	// fallback: 로컬 규칙 기반 파싱
	return parseScheduleLocally(command)
}

func parseScheduleWithLLM(command, llmKey string) (*ParsedSchedule, error) {
	prompt := fmt.Sprintf(`당신은 자연어를 cron 표현식으로 변환하는 전문가입니다.

사용자 명령: "%s"

다음 JSON 형식으로 분석하세요:
{
  "cron_expr": "분 시 일 월 요일 (표준 cron 5필드)",
  "task_name": "작업 이름 (짧게)",
  "action": "browser_agent|summarize_emails|weekly_report|pc_report|llm_task|powershell",
  "params": {"command": "브라우저 에이전트에게 전달할 명령"},
  "description": "이 스케줄 설명"
}

cron 예시:
- "매일 저녁 6시" → "0 18 * * *"
- "내일 아침 8시" → "0 8 * * *" (1회성은 * 대신 내일 날짜)
- "매주 월요일 9시" → "0 9 * * 1"
- "매시간" → "0 * * * *"
- "매일 오전 9시" → "0 9 * * *"

action 선택:
- "메일 요약" 관련 → summarize_emails
- "주간 보고서" 관련 → weekly_report
- "PC 리포트/사용 현황" 관련 → pc_report
- "웹 검색/수집" 관련 → browser_agent (params.command에 원래 명령 포함)
- PowerShell 실행 → powershell
- 기타 AI 분석 → llm_task

현재 시각: %s
현재 요일: %s`, command, time.Now().Format("2006-01-02 15:04"), koreanWeekday(time.Now().Weekday()))

	response, _, err := callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 512, true)
	if err != nil {
		return nil, err
	}

	var parsed struct {
		CronExpr    string                 `json:"cron_expr"`
		TaskName    string                 `json:"task_name"`
		Action      string                 `json:"action"`
		Params      map[string]interface{} `json:"params"`
		Description string                 `json:"description"`
	}
	if err := json.Unmarshal([]byte(response), &parsed); err != nil {
		return nil, fmt.Errorf("JSON 파싱 실패: %s", response)
	}

	nextRun := calcNextRun(parsed.CronExpr, time.Now())

	return &ParsedSchedule{
		CronExpr: parsed.CronExpr,
		TaskName: parsed.TaskName,
		Action:   parsed.Action,
		Params:   parsed.Params,
		NextRun:  nextRun,
	}, nil
}

func parseScheduleLocally(command string) (*ParsedSchedule, error) {
	cmd := strings.ToLower(command)
	result := &ParsedSchedule{Params: make(map[string]interface{})}

	// 시간 파싱
	hourMatch := regexp.MustCompile(`(\d+)시`).FindStringSubmatch(cmd)
	hour := 9
	if len(hourMatch) > 1 {
		h, _ := strconv.Atoi(hourMatch[1])
		if strings.Contains(cmd, "오후") && h < 12 {
			h += 12
		}
		hour = h
	}
	minute := 0
	minMatch := regexp.MustCompile(`(\d+)분`).FindStringSubmatch(cmd)
	if len(minMatch) > 1 {
		minute, _ = strconv.Atoi(minMatch[1])
	}

	// 반복 패턴
	switch {
	case strings.Contains(cmd, "매주 월요일") || strings.Contains(cmd, "월요일마다"):
		result.CronExpr = fmt.Sprintf("%d %d * * 1", minute, hour)
		result.TaskName = "주간 반복 작업"
	case strings.Contains(cmd, "매일") || strings.Contains(cmd, "날마다"):
		result.CronExpr = fmt.Sprintf("%d %d * * *", minute, hour)
		result.TaskName = "매일 반복 작업"
	case strings.Contains(cmd, "내일"):
		tomorrow := time.Now().AddDate(0, 0, 1)
		result.CronExpr = fmt.Sprintf("%d %d %d %d *", minute, hour, tomorrow.Day(), int(tomorrow.Month()))
		result.TaskName = "내일 1회 작업"
	default:
		result.CronExpr = fmt.Sprintf("%d %d * * *", minute, hour)
		result.TaskName = "스케줄 작업"
	}

	// 액션 결정
	switch {
	case strings.Contains(cmd, "메일"):
		result.Action = "summarize_emails"
	case strings.Contains(cmd, "주간 보고"):
		result.Action = "weekly_report"
	case strings.Contains(cmd, "pc") || strings.Contains(cmd, "컴퓨터") || strings.Contains(cmd, "사용 리포트"):
		result.Action = "pc_report"
	case strings.Contains(cmd, "웹") || strings.Contains(cmd, "검색") || strings.Contains(cmd, "쿠팡") || strings.Contains(cmd, "네이버"):
		result.Action = "browser_agent"
		result.Params["command"] = command
	default:
		result.Action = "llm_task"
		result.Params["prompt"] = command
	}

	result.NextRun = calcNextRun(result.CronExpr, time.Now())
	return result, nil
}

func koreanWeekday(d time.Weekday) string {
	days := []string{"일요일", "월요일", "화요일", "수요일", "목요일", "금요일", "토요일"}
	return days[d]
}

// ──────────────────────────────────────────────────────────────
// Windows Task Scheduler 연동
// ──────────────────────────────────────────────────────────────

func registerWindowsTask(task *ScheduledTask) error {
	// schtasks를 사용하여 Windows 작업 스케줄러에 등록
	// 내일 아침 8시 1회성 등록
	if task.NextRun.IsZero() {
		return fmt.Errorf("NextRun이 설정되지 않음")
	}

	nexusExe, err := os.Executable()
	if err != nil {
		nexusExe = "nexus.exe"
	}

	schedTime := task.NextRun.Format("15:04")
	schedDate := task.NextRun.Format("2006/01/02")

	// 주간 반복
	var cmdArgs []string
	if strings.Contains(task.CronExpr, "* * 1") {
		cmdArgs = []string{"/Create", "/F", "/TN", "Nexus\\" + task.ID,
			"/TR", fmt.Sprintf(`"%s" --task-id "%s"`, nexusExe, task.ID),
			"/SC", "WEEKLY", "/D", "MON", "/ST", schedTime}
	} else if strings.Contains(task.CronExpr, "* * *") {
		// 매일
		cmdArgs = []string{"/Create", "/F", "/TN", "Nexus\\" + task.ID,
			"/TR", fmt.Sprintf(`"%s" --task-id "%s"`, nexusExe, task.ID),
			"/SC", "DAILY", "/ST", schedTime}
	} else {
		// 1회성
		cmdArgs = []string{"/Create", "/F", "/TN", "Nexus\\" + task.ID,
			"/TR", fmt.Sprintf(`"%s" --task-id "%s"`, nexusExe, task.ID),
			"/SC", "ONCE", "/SD", schedDate, "/ST", schedTime}
	}

	cmd := newHiddenCmd("schtasks", cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("schtasks 실패: %s (%v)", string(output), err)
	}
	return nil
}

func unregisterWindowsTask(taskID string) error {
	cmd := newHiddenCmd("schtasks", "/Delete", "/F", "/TN", "Nexus\\"+taskID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("schtasks 삭제 실패: %s", string(output))
	}
	return nil
}

// ──────────────────────────────────────────────────────────────
// 내장 태스크 실행 함수들
// ──────────────────────────────────────────────────────────────

func runEmailSummaryTask() (string, error) {
	// 아웃룩 이메일 최근 3개 가져오기
	script := `$outlook = New-Object -ComObject Outlook.Application; ` +
		`$ns = $outlook.GetNamespace('MAPI'); ` +
		`$inbox = $ns.GetDefaultFolder(6); ` +
		`$mails = $inbox.Items | Sort-Object ReceivedTime -Descending | Select-Object -First 3; ` +
		`$mails | ForEach-Object { "제목: " + $_.Subject + " | 보낸이: " + $_.SenderName + " | 날짜: " + $_.ReceivedTime }`
	output, err := runPowerShellScript(script)
	if err != nil {
		return "Outlook 미설치 - 이메일 요약 불가", nil
	}

	// LLM으로 요약
	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()

	if gKey != "" && len(output) > 0 {
		summaryPrompt := fmt.Sprintf("다음 이메일 3개를 핵심만 요약해주세요:\n%s", output)
		summary, _, err := callGroqWithFallback([]groqMsg{{Role: "user", Content: summaryPrompt}}, 512, false)
		if err == nil {
			return summary, nil
		}
	}
	return output, nil
}

func runWeeklyReportTask() (string, error) {
	// 이번 주 생성된 파일 목록
	script := `$weekAgo = (Get-Date).AddDays(-7); ` +
		`$docs = $env:USERPROFILE + '\Documents'; ` +
		`$desk = $env:USERPROFILE + '\Desktop'; ` +
		`$files = Get-ChildItem -Path $docs,$desk -Recurse -ErrorAction SilentlyContinue | ` +
		`Where-Object { $_.LastWriteTime -gt $weekAgo -and -not $_.PSIsContainer } | ` +
		`Select-Object Name,LastWriteTime | Sort-Object LastWriteTime -Descending | Select-Object -First 20; ` +
		`$files | Format-Table -AutoSize | Out-String`
	output, err := runPowerShellScript(script)
	if err != nil {
		return "", err
	}

	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()

	if gKey != "" {
		prompt := fmt.Sprintf("이번 주 생성/수정된 파일 목록:\n%s\n\n주간 업무 요약 보고서를 작성해주세요.", output)
		summary, _, err := callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 1024, false)
		if err == nil {
			return summary, nil
		}
	}
	return output, nil
}

func runPCReportTask() (string, error) {
	// CPU/메모리/디스크/네트워크 현황
	script := `$cpu = (Get-WmiObject Win32_Processor | Measure-Object -Property LoadPercentage -Average).Average; ` +
		`$mem = Get-WmiObject Win32_OperatingSystem; ` +
		`$mu = [math]::Round(($mem.TotalVisibleMemorySize - $mem.FreePhysicalMemory) / 1MB, 1); ` +
		`$mt = [math]::Round($mem.TotalVisibleMemorySize / 1MB, 1); ` +
		`$disk = Get-PSDrive C | Select-Object Used,Free; ` +
		`$du = [math]::Round($disk.Used / 1GB, 1); ` +
		`$df = [math]::Round($disk.Free / 1GB, 1); ` +
		`$uptime = (Get-Date) - (gcim Win32_OperatingSystem).LastBootUpTime; ` +
		`"CPU: " + $cpu + "% | RAM: " + $mu + "/" + $mt + "GB | Disk: " + $du + "/" + ($du+$df) + "GB | Uptime: " + [math]::Floor($uptime.TotalHours) + "h"`
	output, err := runPowerShellScript(script)
	if err != nil {
		return "", err
	}

	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()

	if gKey != "" {
		prompt := fmt.Sprintf("%s\n위 PC 현황을 분석하고, 주의사항이나 최적화 제안을 간략히 해주세요.", output)
		analysis, _, err := callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 512, false)
		if err == nil {
			return output + "\n\n📊 AI 분석:\n" + analysis, nil
		}
	}
	return output, nil
}

func runLLMTask(prompt string) (string, error) {
	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()
	if gKey == "" {
		return "", fmt.Errorf("Groq API 키 미설정")
	}
	result, _, err := callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 1024, false)
	return result, err
}

func runBrowserAgentTask(command string) (string, error) {
	// 간소화된 실행: LLM에게 결과 요청
	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()
	if gKey == "" {
		return "", fmt.Errorf("Groq API 키 미설정")
	}
	prompt := fmt.Sprintf("스케줄된 작업을 실행합니다: %s\n결과 또는 진행 상황을 보고해주세요.", command)
	result, _, err := callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 512, false)
	return result, err
}

func runPowerShellScript(script string) (string, error) {
	cmd := newHiddenCmd("powershell", "-NoProfile", "-NonInteractive",
		"-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

// ──────────────────────────────────────────────────────────────
// HTTP 핸들러들
// ──────────────────────────────────────────────────────────────

// POST /api/scheduler/add
func handleSchedulerAdd(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		Command    string `json:"command"`       // 자연어 명령
		UseWindows bool   `json:"use_windows"`   // Windows Task Scheduler 연동 여부
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Command == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("command 필요", "command is required", lang)})
		return
	}

	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()

	parsed, err := parseNaturalSchedule(req.Command, gKey)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": msgT("스케줄 파싱 실패: ", "Schedule parsing failed: ", lang) + err.Error()})
		return
	}

	paramsJSONBytes, _ := json.Marshal(parsed.Params)
	task := &ScheduledTask{
		ID:           fmt.Sprintf("task_%d", time.Now().UnixNano()),
		Name:         parsed.TaskName,
		Command:      req.Command,
		Action:       parsed.Action,
		ActionParams: string(paramsJSONBytes),
		CronExpr:     parsed.CronExpr,
		NextRun:      parsed.NextRun,
		Active:       true,
		CreatedAt:    time.Now(),
	}

	if req.UseWindows {
		if err := registerWindowsTask(task); err != nil {
			// Windows 등록 실패해도 내부 스케줄러로 진행
			_ = err
		}
	}

	globalScheduler.mu.Lock()
	globalScheduler.tasks[task.ID] = task
	globalScheduler.mu.Unlock()
	globalScheduler.save()

	json200(w, map[string]any{
		"success":     true,
		"task":        task,
		"next_run_kr": task.NextRun.Format("2006년 01월 02일 15:04"),
		"message":     fmt.Sprintf(msgT("'%s' 스케줄 등록 완료. 다음 실행: %s", "Schedule '%s' registered. Next run: %s", lang), task.Name, task.NextRun.Format("2006-01-02 15:04")),
	})
}

// GET /api/scheduler/list
func handleSchedulerList(w http.ResponseWriter, r *http.Request) {
	globalScheduler.mu.RLock()
	tasks := make([]*ScheduledTask, 0, len(globalScheduler.tasks))
	for _, t := range globalScheduler.tasks {
		tasks = append(tasks, t)
	}
	globalScheduler.mu.RUnlock()

	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].NextRun.Before(tasks[j].NextRun)
	})

	json200(w, map[string]any{
		"success": true,
		"tasks":   tasks,
		"total":   len(tasks),
	})
}

// DELETE /api/scheduler/delete?id=xxx
func handleSchedulerDelete(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	id := r.URL.Query().Get("id")
	if id == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("id 필요", "id is required", lang)})
		return
	}

	globalScheduler.mu.Lock()
	task, ok := globalScheduler.tasks[id]
	if ok {
		delete(globalScheduler.tasks, id)
	}
	globalScheduler.mu.Unlock()

	if !ok {
		writeJSON(w, 404, map[string]any{"success": false, "message": msgT("태스크 없음", "Task not found", lang)})
		return
	}

	unregisterWindowsTask(id)
	globalScheduler.save()

	json200(w, map[string]any{
		"success": true,
		"message": fmt.Sprintf(msgT("'%s' 스케줄 삭제 완료", "Schedule '%s' deleted", lang), task.Name),
	})
}

// POST /api/scheduler/run-now?id=xxx
func handleSchedulerRunNow(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	id := r.URL.Query().Get("id")
	if id == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("id 필요", "id is required", lang)})
		return
	}

	globalScheduler.mu.RLock()
	task, ok := globalScheduler.tasks[id]
	globalScheduler.mu.RUnlock()

	if !ok {
		writeJSON(w, 404, map[string]any{"success": false, "message": msgT("태스크 없음", "Task not found", lang)})
		return
	}

	go globalScheduler.executeTask(task)

	json200(w, map[string]any{
		"success": true,
		"message": fmt.Sprintf(msgT("'%s' 즉시 실행 시작됨", "Running '%s' now", lang), task.Name),
	})
}

// POST /api/scheduler/parse
// 자연어 → 스케줄 미리보기 (실제 등록 없이)
func handleSchedulerParse(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		Command string `json:"command"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Command == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("command 필요", "command is required", lang)})
		return
	}

	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()

	parsed, err := parseNaturalSchedule(req.Command, gKey)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": msgT("파싱 실패: ", "Parse failed: ", lang) + err.Error()})
		return
	}

	json200(w, map[string]any{
		"success":     true,
		"cron_expr":   parsed.CronExpr,
		"task_name":   parsed.TaskName,
		"action":      parsed.Action,
		"params":      parsed.Params,
		"next_run":    parsed.NextRun.Format(time.RFC3339),
		"next_run_kr": parsed.NextRun.Format("2006년 01월 02일 (월) 15:04"),
	})
}
