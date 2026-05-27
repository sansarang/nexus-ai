//go:build !windows

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// ── Stealth JS (Windows와 동일한 전체 fingerprint 차단) ───────

const macStealthJS = `
(function(){
  try { const p = navigator.__proto__; delete p.webdriver; Object.defineProperty(navigator,'webdriver',{get:()=>undefined,configurable:true}); } catch(e){}
  if(!window.chrome){window.chrome={};}
  if(!window.chrome.app){window.chrome.app={isInstalled:false,getDetails:function(){},getIsInstalled:function(){},installState:function(){},runningState:function(){}};}
  if(!window.chrome.runtime){window.chrome.runtime={connect:function(){},sendMessage:function(){}};}
  if(!window.chrome.csi)window.chrome.csi=function(){};
  if(!window.chrome.loadTimes)window.chrome.loadTimes=function(){return{requestTime:Date.now()/1000-Math.random()*2,navigationType:'Other',wasNpnNegotiated:false,npnNegotiatedProtocol:'http/1.1',connectionInfo:'http/1.1'};};
  try{Object.defineProperty(navigator,'languages',{get:()=>['ko-KR','ko','en-US','en'],configurable:true});}catch(e){}
  try{Object.defineProperty(navigator,'language',{get:()=>'ko-KR',configurable:true});}catch(e){}
  try{Object.defineProperty(navigator,'platform',{get:()=>'MacIntel',configurable:true});}catch(e){}
  try{Object.defineProperty(navigator,'hardwareConcurrency',{get:()=>8,configurable:true});}catch(e){}
  try{Object.defineProperty(navigator,'deviceMemory',{get:()=>8,configurable:true});}catch(e){}
  try{const o=window.navigator.permissions.query.bind(window.navigator.permissions);window.navigator.permissions.query=(p)=>{if(p.name==='notifications')return Promise.resolve({state:Notification.permission});return o(p);};}catch(e){}
  try{const d=Object.getOwnPropertyDescriptor(HTMLIFrameElement.prototype,'contentWindow');if(d){Object.defineProperty(HTMLIFrameElement.prototype,'contentWindow',{get:function(){const w=d.get.call(this);if(!w)return w;try{Object.defineProperty(w.navigator,'webdriver',{get:()=>undefined,configurable:true});}catch(e){}return w;}});}}catch(e){}
  try{const g=WebGLRenderingContext.prototype.getParameter;WebGLRenderingContext.prototype.getParameter=function(p){if(p===37445)return'Intel Inc.';if(p===37446)return'Intel(R) Iris(TM) Plus Graphics 640';return g.call(this,p);};}catch(e){}
  try{const g2=WebGL2RenderingContext.prototype.getParameter;WebGL2RenderingContext.prototype.getParameter=function(p){if(p===37445)return'Intel Inc.';if(p===37446)return'Intel(R) Iris(TM) Plus Graphics 640';return g2.call(this,p);};}catch(e){}
  try{const ot=HTMLCanvasElement.prototype.toDataURL;HTMLCanvasElement.prototype.toDataURL=function(){const c=this.getContext('2d');if(c){const id=c.getImageData(0,0,this.width,this.height);for(let i=0;i<id.data.length;i+=4){id.data[i]^=(Math.random()*2|0);id.data[i+1]^=(Math.random()*2|0);id.data[i+2]^=(Math.random()*2|0);}c.putImageData(id,0,0);}return ot.apply(this,arguments);};}catch(e){}
  try{const oa=AudioBuffer.prototype.getChannelData;AudioBuffer.prototype.getChannelData=function(){const d=oa.apply(this,arguments);for(let i=0;i<d.length;i+=100){d[i]+=Math.random()*0.0000001;}return d;};}catch(e){}
  try{Date.prototype.getTimezoneOffset=function(){return -540;};}catch(e){}
  try{['cdc_adoQpoasnfa76pfcZLmcfl_Array','cdc_adoQpoasnfa76pfcZLmcfl_Promise','cdc_adoQpoasnfa76pfcZLmcfl_Symbol','$chrome_asyncScriptInfo','domAutomation','domAutomationController','_phantom','__phantom','callPhantom','webdriver'].forEach(k=>{try{delete window[k];}catch(e){}});}catch(e){}
  try{if(window.outerWidth===0)Object.defineProperty(window,'outerWidth',{get:()=>1440,configurable:true});if(window.outerHeight===0)Object.defineProperty(window,'outerHeight',{get:()=>900,configurable:true});}catch(e){}
})();
`

var macStealthUserAgents = []string{
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_4_1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36",
}

func randomMacUA() string {
	return macStealthUserAgents[rand.Intn(len(macStealthUserAgents))]
}

// ── Browser 세션 (Mac/Linux) — Stealth 적용 ──────────────────

func getBrowserCtxMac() (context.Context, context.CancelFunc, error) {
	ua := randomMacUA()
	resolutions := [][2]int{{1440, 900}, {1920, 1080}, {1280, 800}, {1600, 900}}
	res := resolutions[rand.Intn(len(resolutions))]

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("no-default-browser-check", true),
		chromedp.Flag("disable-notifications", true),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("lang", "ko-KR"),
		chromedp.WindowSize(res[0], res[1]),
		chromedp.UserAgent(ua),
	)
	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, ctxCancel := chromedp.NewContext(allocCtx)
	pingCtx, pingCancel := context.WithTimeout(ctx, 8*time.Second)
	defer pingCancel()
	if err := chromedp.Run(pingCtx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			_, err := page.AddScriptToEvaluateOnNewDocument(macStealthJS).Do(ctx)
			return err
		}),
	); err != nil {
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
	lang := getLang(r)
	var req struct {
		URL    string `json:"url"`
		WaitFor string `json:"wait_for"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.URL == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("url 필요", "url required", lang)})
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
	if req.URL == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "url required"})
		return
	}
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
		lang := getLang(r)
		writeJSON(w, 500, map[string]any{"success": false, "message": msgT("페이지 추출 실패: ", "Page extraction failed: ", lang) + err.Error()})
		return
	}
	json200(w, map[string]any{"success": true, "text": text, "url": req.URL})
}

func handleBrowserClick(w http.ResponseWriter, r *http.Request)    { lang := getLang(r); writeJSON(w, 200, map[string]any{"success": false, "message": msgT("미구현", "Not implemented", lang)}) }
func handleBrowserFill(w http.ResponseWriter, r *http.Request)     { lang := getLang(r); writeJSON(w, 200, map[string]any{"success": false, "message": msgT("미구현", "Not implemented", lang)}) }
func handleBrowserScreenshot(w http.ResponseWriter, r *http.Request) { lang := getLang(r); writeJSON(w, 200, map[string]any{"success": false, "message": msgT("미구현", "Not implemented", lang)}) }
func handleBrowserAgent(w http.ResponseWriter, r *http.Request)    { lang := getLang(r); writeJSON(w, 200, map[string]any{"success": false, "message": msgT("미구현", "Not implemented", lang)}) }
func handleBrowserClose(w http.ResponseWriter, r *http.Request)    { json200(w, map[string]any{"success": true}) }

func handleBrowserSmartAgent(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		Command    string `json:"command"`
		MaxResults int    `json:"max_results"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()
	if gKey == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("Groq API 키 필요", "Groq API key required", lang)})
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
	lang := getLang(r)
	writeJSON(w, 200, map[string]any{"success": false, "message": msgT("로그인 세션은 Windows에서 지원됩니다", "Login session is supported on Windows only", lang)})
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
	lang := getLang(r)
	writeJSON(w, 200, map[string]any{"success": false, "message": msgT("파일 열기는 Windows에서 지원됩니다", "File open is supported on Windows only", lang)})
}

// ── Excel ─────────────────────────────────────────────────────
func handleExcelSave(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	writeJSON(w, 200, map[string]any{"success": false, "message": msgT("Excel 저장은 Windows에서 지원됩니다", "Excel save is supported on Windows only", lang)})
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
	Action     string    `json:"action"`
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
	lang := getLang(r)
	var req struct {
		Name    string `json:"name"`
		Command string `json:"command"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("잘못된 요청", "Invalid request", lang)})
		return
	}

	// LLM으로 자연어 → cron 표현식 파싱
	cronExpr := ""
	taskName := req.Name
	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()
	if gKey != "" && req.Command != "" {
		prompt := fmt.Sprintf(`자연어 스케줄을 cron 표현식(분 시 일 월 요일)으로 변환하세요.
입력: "%s"
JSON만 반환: {"cron": "0 18 * * 5", "name": "작업명"}
예시: 매주 금요일 저녁 6시 → {"cron":"0 18 * * 5","name":"금요일 저녁 작업"}`, req.Command)
		raw, _, err := callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 128, true)
		if err == nil {
			var parsed struct {
				Cron string `json:"cron"`
				Name string `json:"name"`
			}
			if json.Unmarshal([]byte(strings.TrimSpace(raw)), &parsed) == nil {
				cronExpr = parsed.Cron
				if taskName == "" {
					taskName = parsed.Name
				}
			}
		}
	}

	task := ScheduledTask{
		ID: fmt.Sprintf("%d", time.Now().UnixMilli()), Name: taskName,
		Command: req.Command, CronExpr: cronExpr, Active: true, CreatedAt: time.Now(),
	}
	schedulerTasksMu.Lock()
	scheduledTasks = append(scheduledTasks, task)
	schedulerTasksMu.Unlock()
	regLabel := msgT("스케줄 등록됨: ", "Schedule registered: ", lang)
	msg := regLabel + taskName
	if cronExpr != "" {
		msg += fmt.Sprintf(" (cron: %s)", cronExpr)
	}
	json200(w, map[string]any{"success": true, "task": task, "message": msg, "cron_expr": cronExpr})
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

func handleSchedulerRunNow(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct{ ID string `json:"id"` }
	json.NewDecoder(r.Body).Decode(&req)
	schedulerTasksMu.RLock()
	tasksCopy := make([]ScheduledTask, len(scheduledTasks))
	copy(tasksCopy, scheduledTasks)
	schedulerTasksMu.RUnlock()
	for _, t := range tasksCopy {
		if t.ID == req.ID {
			json200(w, map[string]any{"success": true, "message": msgT("작업 실행 요청됨: ", "Task execution requested: ", lang) + t.Name})
			return
		}
	}
	writeJSON(w, 404, map[string]any{"success": false, "message": msgT("작업을 찾을 수 없어요", "Task not found", lang)})
}

func handleSchedulerParse(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct{ Text string `json:"text"` }
	json.NewDecoder(r.Body).Decode(&req)
	if req.Text == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("text 필요", "text required", lang)})
		return
	}
	prompt := `자연어 스케줄을 cron 표현식(분 시 일 월 요일)으로 변환해줘.
입력: "` + req.Text + `"
JSON으로만 응답: {"cron": "0 9 * * 1-5", "description": "설명"}`
	result, _, _ := callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 100, true)
	var parsed map[string]string
	if json.Unmarshal([]byte(result), &parsed) == nil {
		json200(w, map[string]any{"success": true, "cron": parsed["cron"], "description": parsed["description"]})
		return
	}
	json200(w, map[string]any{"success": false, "message": msgT("파싱 실패", "Parsing failed", lang), "raw": result})
}

func handleClipboard(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	out, err := exec.Command("pbpaste").Output()
	if err != nil {
		json200(w, map[string]any{"success": false, "message": msgT("클립보드 읽기 실패", "Failed to read clipboard", lang), "text": ""})
		return
	}
	text := strings.TrimSpace(string(out))
	json200(w, map[string]any{"success": true, "text": text, "length": len(text)})
}
