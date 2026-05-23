//go:build !windows

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ── 크로스플랫폼 cron 실행 엔진 ────────────────────────────

type CronTask struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Command    string    `json:"command"`    // 자연어 명령
	CronExpr   string    `json:"cron_expr"`  // "0 18 * * 5"
	Action     string    `json:"action"`     // llm_command | weekly_report | email_summary
	Params     string    `json:"params"`     // JSON
	Active     bool      `json:"active"`
	NextRun    time.Time `json:"next_run"`
	LastRun    time.Time `json:"last_run,omitempty"`
	LastResult string    `json:"last_result,omitempty"`
	RunCount   int       `json:"run_count"`
	CreatedAt  time.Time `json:"created_at"`
}

type cronEngine struct {
	mu    sync.RWMutex
	tasks map[string]*CronTask
	path  string
}

var globalCronEngine *cronEngine

func cronStorePath() string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".nexus")
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "cron_tasks.json")
}

func initCronEngine() {
	globalCronEngine = &cronEngine{
		tasks: map[string]*CronTask{},
		path:  cronStorePath(),
	}
	globalCronEngine.load()
	go globalCronEngine.runLoop()
	log.Println("[Cron] 실행 엔진 시작됨")
}

func (c *cronEngine) load() {
	data, err := os.ReadFile(c.path)
	if err != nil {
		return
	}
	var tasks []*CronTask
	if json.Unmarshal(data, &tasks) != nil {
		return
	}
	c.mu.Lock()
	for _, t := range tasks {
		c.tasks[t.ID] = t
	}
	c.mu.Unlock()
	log.Printf("[Cron] 작업 %d개 로드됨", len(tasks))
}

func (c *cronEngine) save() {
	c.mu.RLock()
	tasks := make([]*CronTask, 0, len(c.tasks))
	for _, t := range c.tasks {
		tasks = append(tasks, t)
	}
	c.mu.RUnlock()
	data, _ := json.MarshalIndent(tasks, "", "  ")
	os.WriteFile(c.path, data, 0644)
}

func (c *cronEngine) addTask(t *CronTask) {
	t.NextRun = calcNextRunCross(t.CronExpr, time.Now())
	c.mu.Lock()
	c.tasks[t.ID] = t
	c.mu.Unlock()
	c.save()
}

func (c *cronEngine) runLoop() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		c.checkAndRun()
	}
}

func (c *cronEngine) checkAndRun() {
	now := time.Now()
	c.mu.RLock()
	var toRun []*CronTask
	for _, t := range c.tasks {
		if t.Active && !t.NextRun.IsZero() && now.After(t.NextRun) {
			toRun = append(toRun, t)
		}
	}
	c.mu.RUnlock()
	for _, t := range toRun {
		go c.execute(t)
	}
}

func (c *cronEngine) execute(t *CronTask) {
	log.Printf("[Cron] 실행: %s (%s)", t.Name, t.CronExpr)
	var result string
	var execErr error

	switch t.Action {
	case "llm_command":
		result, execErr = cronRunCommand(t.Command)
	case "weekly_report":
		result, execErr = cronWeeklyReport()
	case "email_summary":
		result, execErr = cronEmailSummary()
	case "briefing":
		result, execErr = cronBriefing()
	default:
		result, execErr = cronRunCommand(t.Command)
	}

	c.mu.Lock()
	if st, ok := c.tasks[t.ID]; ok {
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
		st.NextRun = calcNextRunCross(st.CronExpr, time.Now())
		log.Printf("[Cron] 완료: %s → 다음 실행: %s", t.Name, st.NextRun.Format("01-02 15:04"))
	}
	c.mu.Unlock()
	c.save()
}

// ── cron 액션 구현 ────────────────────────────────────────

func cronRunCommand(cmd string) (string, error) {
	body, _ := json.Marshal(map[string]any{"message": cmd})
	resp, err := (&http.Client{Timeout: 120 * time.Second}).Post(
		"http://127.0.0.1:17891/api/command", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var d map[string]any
	json.Unmarshal(raw, &d)
	if msg, ok := d["message"].(string); ok {
		return msg, nil
	}
	return string(raw), nil
}

func cronWeeklyReport() (string, error) {
	// 이번 주 생산성 리포트 생성
	now := time.Now()
	weekStart := now.AddDate(0, 0, -int(now.Weekday()))
	prompt := fmt.Sprintf(`%s ~ %s 이번 주 생산성 리포트를 작성해줘.
포함 내용:
1. 이번 주 요약 및 주요 성과
2. 시스템 사용 패턴
3. 다음 주 개선 제안

한국어로 친절하게 작성해줘.`, weekStart.Format("1월 2일"), now.Format("1월 2일"))

	text, _, err := callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 800, false)
	if err != nil {
		return "", err
	}

	// 파일 저장
	home, _ := os.UserHomeDir()
	fname := fmt.Sprintf("weekly_report_%s.md", now.Format("20060102"))
	path := filepath.Join(home, "Desktop", fname)
	content := fmt.Sprintf("# 주간 생산성 리포트\n\n*%s 생성*\n\n%s", now.Format("2006-01-02 15:04"), text)
	os.WriteFile(path, []byte(content), 0644)

	// 이메일 발송 시도
	llmMu.RLock()
	accounts := imapAccounts
	llmMu.RUnlock()
	if len(accounts) > 0 {
		go sendIMAPEmail(accounts[0], accounts[0].Email, "이번 주 생산성 리포트", content)
	}

	return fmt.Sprintf("주간 리포트 생성 완료: %s", path), nil
}

func cronEmailSummary() (string, error) {
	return cronRunCommand("받은 이메일 요약해줘")
}

func cronBriefing() (string, error) {
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Post(
		"http://127.0.0.1:17891/api/briefing/now", "application/json", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var d map[string]any
	json.Unmarshal(raw, &d)
	if b, ok := d["briefing"].(string); ok {
		return b, nil
	}
	return "", nil
}

// ── cron 표현식 계산 ──────────────────────────────────────

func calcNextRunCross(cronExpr string, from time.Time) time.Time {
	parts := strings.Fields(cronExpr)
	if len(parts) != 5 {
		return from.Add(24 * time.Hour)
	}
	isWild := func(s string) bool { return s == "*" }
	parseInt := func(s string) int { n, _ := strconv.Atoi(s); return n }

	minute := parseInt(parts[0])
	hour := parseInt(parts[1])

	now := from.Truncate(time.Minute)
	base := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())

	for i := 0; i < 366; i++ {
		check := base.AddDate(0, 0, i)
		if !isWild(parts[3]) && int(check.Month()) != parseInt(parts[3]) {
			continue
		}
		if !isWild(parts[2]) && check.Day() != parseInt(parts[2]) {
			continue
		}
		if !isWild(parts[4]) && int(check.Weekday()) != parseInt(parts[4]) {
			continue
		}
		if isWild(parts[1]) {
			check = time.Date(check.Year(), check.Month(), check.Day(), now.Hour(), minute, 0, 0, now.Location())
		}
		if check.After(now) {
			return check
		}
	}
	return from.Add(24 * time.Hour)
}

// ── cron 핸들러 ────────────────────────────────────────────

// POST /api/cron/add
func handleCronAdd(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		Command  string `json:"command"`
		CronExpr string `json:"cron_expr"`
		Action   string `json:"action"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Command == "" && req.CronExpr == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "command, cron_expr 필요"})
		return
	}

	// cron 표현식 자동 파싱 (자연어 → cron)
	cronExpr := req.CronExpr
	if cronExpr == "" && req.Command != "" {
		prompt := fmt.Sprintf(`자연어를 cron 표현식(분 시 일 월 요일)으로 변환해줘.
입력: "%s"
JSON만 출력: {"cron":"0 18 * * 5","name":"작업명"}`, req.Command)
		raw, _, _ := callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 100, true)
		var p map[string]string
		if json.Unmarshal([]byte(strings.TrimSpace(raw)), &p) == nil {
			cronExpr = p["cron"]
			if req.Name == "" {
				req.Name = p["name"]
			}
		}
	}
	if cronExpr == "" {
		cronExpr = "0 9 * * 1-5" // 기본: 평일 오전 9시
	}

	action := req.Action
	if action == "" {
		action = "llm_command"
	}

	task := &CronTask{
		ID:        fmt.Sprintf("%d", time.Now().UnixMilli()),
		Name:      req.Name,
		Command:   req.Command,
		CronExpr:  cronExpr,
		Action:    action,
		Active:    true,
		CreatedAt: time.Now(),
	}
	if task.Name == "" {
		task.Name = req.Command
		if len(task.Name) > 30 {
			task.Name = task.Name[:30]
		}
	}

	globalCronEngine.addTask(task)

	json200(w, map[string]any{
		"success":  true,
		"task":     task,
		"message":  fmt.Sprintf("✅ '%s' 스케줄 등록됨 (cron: %s, 다음 실행: %s)", task.Name, cronExpr, task.NextRun.Format("01/02 15:04")),
		"cron_expr": cronExpr,
		"next_run": task.NextRun.Format("2006-01-02 15:04"),
	})
}

// GET /api/cron/list
func handleCronList(w http.ResponseWriter, r *http.Request) {
	globalCronEngine.mu.RLock()
	tasks := make([]*CronTask, 0, len(globalCronEngine.tasks))
	for _, t := range globalCronEngine.tasks {
		tasks = append(tasks, t)
	}
	globalCronEngine.mu.RUnlock()
	json200(w, map[string]any{"success": true, "tasks": tasks, "count": len(tasks)})
}

// DELETE /api/cron/delete?id=xxx
func handleCronDelete(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	globalCronEngine.mu.Lock()
	_, exists := globalCronEngine.tasks[id]
	if exists {
		delete(globalCronEngine.tasks, id)
	}
	globalCronEngine.mu.Unlock()
	if !exists {
		writeJSON(w, 404, map[string]any{"success": false, "message": msgT("작업을 찾을 수 없어요", "Task not found", getLang(r))})
		return
	}
	globalCronEngine.save()
	json200(w, map[string]any{"success": true, "message": msgT("삭제됨", "Deleted", getLang(r))})
}

// POST /api/cron/run-now?id=xxx
func handleCronRunNow(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	globalCronEngine.mu.RLock()
	task, exists := globalCronEngine.tasks[id]
	globalCronEngine.mu.RUnlock()
	if !exists {
		writeJSON(w, 404, map[string]any{"success": false, "message": msgT("작업을 찾을 수 없어요", "Task not found", getLang(r))})
		return
	}
	go globalCronEngine.execute(task)
	json200(w, map[string]any{"success": true, "message": msgT(fmt.Sprintf("'%s' 즉시 실행 시작됨", task.Name), fmt.Sprintf("'%s' running now", task.Name), getLang(r))})
}
