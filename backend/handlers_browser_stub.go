//go:build !windows

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
)

// ── Browser 세션 (Mac/Linux) ──────────────────────────────────

func getBrowserCtxMac() (context.Context, context.CancelFunc, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("no-default-browser-check", true),
		chromedp.Flag("disable-notifications", true),
		chromedp.WindowSize(1280, 900),
		chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"),
	)
	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, ctxCancel := chromedp.NewContext(allocCtx)
	pingCtx, pingCancel := context.WithTimeout(ctx, 5*time.Second)
	defer pingCancel()
	if err := chromedp.Run(pingCtx); err != nil {
		ctxCancel()
		allocCancel()
		return nil, nil, fmt.Errorf("Chrome 실행 실패: %w", err)
	}
	cancel := func() { ctxCancel(); allocCancel() }
	return ctx, cancel, nil
}

func handleBrowserStatus(w http.ResponseWriter, r *http.Request) {
	_, cancel, err := getBrowserCtxMac()
	if err != nil {
		json200(w, map[string]any{"running": false, "error": err.Error()})
		return
	}
	cancel()
	json200(w, map[string]any{"running": true, "platform": "mac"})
}

func handleBrowserNavigate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL    string `json:"url"`
		WaitFor string `json:"wait_for"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.URL == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "url 필요"})
		return
	}
	ctx, cancel, err := getBrowserCtxMac()
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}
	defer cancel()
	tCtx, tCancel := context.WithTimeout(ctx, 20*time.Second)
	defer tCancel()
	var title string
	err = chromedp.Run(tCtx,
		chromedp.Navigate(req.URL),
		chromedp.WaitReady("body"),
		chromedp.Title(&title),
	)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}
	json200(w, map[string]any{"success": true, "url": req.URL, "title": title})
}

func handleBrowserExtract(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL      string `json:"url"`
		Selector string `json:"selector"`
		Mode     string `json:"mode"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	ctx, cancel, err := getBrowserCtxMac()
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}
	defer cancel()
	tCtx, tCancel := context.WithTimeout(ctx, 30*time.Second)
	defer tCancel()
	actions := chromedp.Tasks{chromedp.Navigate(req.URL), chromedp.WaitReady("body")}
	var text string
	sel := req.Selector
	if sel == "" {
		sel = "body"
	}
	actions = append(actions, chromedp.Text(sel, &text, chromedp.ByQuery))
	if err := chromedp.Run(tCtx, actions); err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": "페이지 추출 실패: " + err.Error()})
		return
	}
	json200(w, map[string]any{"success": true, "text": text, "url": req.URL})
}

func handleBrowserClick(w http.ResponseWriter, r *http.Request)    { writeJSON(w, 200, map[string]any{"success": false, "message": "미구현"}) }
func handleBrowserFill(w http.ResponseWriter, r *http.Request)     { writeJSON(w, 200, map[string]any{"success": false, "message": "미구현"}) }
func handleBrowserScreenshot(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]any{"success": false, "message": "미구현"}) }
func handleBrowserAgent(w http.ResponseWriter, r *http.Request)    { writeJSON(w, 200, map[string]any{"success": false, "message": "미구현"}) }
func handleBrowserClose(w http.ResponseWriter, r *http.Request)    { json200(w, map[string]any{"success": true}) }

func handleBrowserSmartAgent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Command    string `json:"command"`
		MaxResults int    `json:"max_results"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()
	if gKey == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "Groq API 키 필요"})
		return
	}
	result := runWebSearchMac(gKey, req.Command, "auto", req.MaxResults)
	json200(w, map[string]any{"success": true, "summary": result.Summary, "items": result.Items})
}

func handleBrowserCollectPrice(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProductQuery string `json:"product_query"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()
	result := runWebSearchMac(gKey, req.ProductQuery+" 최저가", "coupang", 5)
	json200(w, map[string]any{"success": true, "summary": result.Summary, "items": result.Items})
}

func handleBrowserNewsCollect(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query string `json:"query"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()
	result := runWebSearchMac(gKey, req.Query+" 뉴스", "naver", 8)
	json200(w, map[string]any{"success": true, "summary": result.Summary, "items": result.Items})
}

func handleBrowserLoginSession(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"success": false, "message": "로그인 세션은 Windows에서 지원됩니다"})
}

func handleBrowserSearchAndPDF(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query    string `json:"query"`
		MaxItems int    `json:"max_items"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()
	result := runWebSearchMac(gKey, req.Query, "auto", req.MaxItems)
	json200(w, map[string]any{
		"success": true,
		"summary": result.Summary,
		"message": "Mac 환경에서는 PDF 저장 대신 텍스트로 제공됩니다.",
		"items":   result.Items,
	})
}

func handleOpenFile(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"success": false, "message": "파일 열기는 Windows에서 지원됩니다"})
}

// ── Excel ─────────────────────────────────────────────────────
func handleExcelSave(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"success": false, "message": "Excel 저장은 Windows에서 지원됩니다"})
}
func handleExcelList(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"files": []any{}})
}
func saveToExcel(data [][]string, outPath, sheetTitle string) error { return nil }

// ── Scheduler ─────────────────────────────────────────────────

type ScheduledTask struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Command    string    `json:"command"`
	CronExpr   string    `json:"cron_expr"`
	NextRun    time.Time `json:"next_run"`
	LastRun    time.Time `json:"last_run"`
	LastResult string    `json:"last_result"`
	Active     bool      `json:"active"`
	CreatedAt  time.Time `json:"created_at"`
}

var (
	scheduledTasks   []ScheduledTask
	schedulerTasksMu sync.RWMutex
)

func initScheduler() {}

func handleSchedulerAdd(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name    string `json:"name"`
		Command string `json:"command"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]any{"success": false, "message": "잘못된 요청"})
		return
	}
	task := ScheduledTask{
		ID: fmt.Sprintf("%d", time.Now().UnixMilli()), Name: req.Name,
		Command: req.Command, Active: true, CreatedAt: time.Now(),
	}
	schedulerTasksMu.Lock()
	scheduledTasks = append(scheduledTasks, task)
	schedulerTasksMu.Unlock()
	json200(w, map[string]any{"success": true, "task": task})
}

func handleSchedulerList(w http.ResponseWriter, r *http.Request) {
	schedulerTasksMu.RLock()
	tasks := make([]ScheduledTask, len(scheduledTasks))
	copy(tasks, scheduledTasks)
	schedulerTasksMu.RUnlock()
	json200(w, map[string]any{"tasks": tasks})
}

func handleSchedulerDelete(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": true})
}

// ── Memory ────────────────────────────────────────────────────

type AgentMemoryEntry struct {
	ID        string                 `json:"id"`
	Timestamp string                 `json:"timestamp"`
	Type      string                 `json:"type"`
	Command   string                 `json:"command"`
	Result    string                 `json:"result"`
	Success   bool                   `json:"success"`
	Tags      []string               `json:"tags,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

func initMemory()                                {}
func saveAgentMemory(_ AgentMemoryEntry)         {}
func buildContextFromMemory(_ string, _ int) string { return "" }

func handleMemoryList(w http.ResponseWriter, r *http.Request)   { json200(w, map[string]any{"entries": []any{}}) }
func handleMemorySearch(w http.ResponseWriter, r *http.Request) { json200(w, map[string]any{"results": []any{}}) }
func handleMemoryClear(w http.ResponseWriter, r *http.Request)  { json200(w, map[string]any{"success": true}) }
func handleMemoryStats(w http.ResponseWriter, r *http.Request)  { json200(w, map[string]any{"total": 0}) }
