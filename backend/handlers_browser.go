//go:build windows

package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
)

// ──────────────────────────────────────────────────────────────
// Chrome 설치 확인 (공통)
// ──────────────────────────────────────────────────────────────

func isChromeInstalled() bool {
	// 시스템 전체 설치 경로
	fixed := []string{
		`C:\Program Files\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
		`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
	}
	for _, p := range fixed {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	// 사용자별 설치 경로 (%LOCALAPPDATA%)
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		user := []string{
			local + `\Google\Chrome\Application\chrome.exe`,
			local + `\Microsoft\Edge\Application\msedge.exe`,
			local + `\Chromium\Application\chrome.exe`,
		}
		for _, p := range user {
			if _, err := os.Stat(p); err == nil {
				return true
			}
		}
	}
	_, e1 := exec.LookPath("chrome")
	_, e2 := exec.LookPath("msedge")
	return e1 == nil || e2 == nil
}

// Chrome 미설치 시 Groq로 폴백하여 응답 반환
func groqFallbackSearch(w http.ResponseWriter, query string) {
	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()
	if gKey == "" {
		writeJSON(w, 503, map[string]any{
			"success": false,
			"message": "Chrome/Edge가 설치되지 않았고 Groq API 키도 없습니다. Chrome을 설치하거나 API 키를 설정해주세요.",
		})
		return
	}
	msgs := []groqMsg{{Role: "user", Content: query + " — 알고 있는 정보를 상세하게 알려줘."}}
	text, _, _ := callGroq(gKey, groqChatModel, msgs, 1024, false)
	json200(w, map[string]any{
		"success": true,
		"summary": text,
		"items":   []any{},
		"message": "⚠️ Chrome/Edge 미설치 — AI 지식 기반으로 답변합니다. 실시간 크롤링을 원하면 Chrome을 설치해주세요.",
	})
}

// ──────────────────────────────────────────────────────────────
// 브라우저 세션 관리
// ──────────────────────────────────────────────────────────────

var (
	browserMu     sync.Mutex
	browserAlloc  context.Context
	browserCancel context.CancelFunc
	browserCtx    context.Context
	browserBroken bool
)

// ensureBrowser: 브라우저 세션을 초기화하거나 재사용
func ensureBrowser() (context.Context, error) {
	browserMu.Lock()
	defer browserMu.Unlock()

	// 이미 정상 세션이 있으면 재사용
	if browserAlloc != nil && !browserBroken {
		select {
		case <-browserAlloc.Done():
			// 세션 만료 → 재생성
		default:
			return browserCtx, nil
		}
	}

	// 이전 세션 정리
	if browserCancel != nil {
		browserCancel()
	}

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),       // 사용자가 볼 수 있게
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("no-default-browser-check", true),
		chromedp.Flag("disable-notifications", true),
		chromedp.Flag("disable-popup-blocking", false),
		chromedp.WindowSize(1280, 900),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, ctxCancel := chromedp.NewContext(allocCtx)

	// 브라우저 연결 확인 (5초 타임아웃)
	pingCtx, pingCancel := context.WithTimeout(ctx, 5*time.Second)
	defer pingCancel()
	if err := chromedp.Run(pingCtx); err != nil {
		ctxCancel()
		allocCancel()
		return nil, fmt.Errorf("Chrome 실행 실패 (Chrome/Edge 설치 필요): %w", err)
	}

	browserAlloc = allocCtx
	browserCancel = func() {
		ctxCancel()
		allocCancel()
	}
	browserCtx = ctx
	browserBroken = false
	return ctx, nil
}

// withBrowserTimeout: 타임아웃이 있는 브라우저 컨텍스트 생성
func withBrowserTimeout(timeout time.Duration) (context.Context, context.CancelFunc, error) {
	base, err := ensureBrowser()
	if err != nil {
		return nil, nil, err
	}
	ctx, cancel := context.WithTimeout(base, timeout)
	return ctx, cancel, nil
}

// ──────────────────────────────────────────────────────────────
// POST /api/browser/navigate  — URL 이동
// ──────────────────────────────────────────────────────────────

func handleBrowserNavigate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL     string `json:"url"`
		WaitFor string `json:"wait_for"` // selector to wait for
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.URL == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "url 필요"})
		return
	}
	if !strings.HasPrefix(req.URL, "http") {
		req.URL = "https://" + req.URL
	}

	ctx, cancel, err := withBrowserTimeout(30 * time.Second)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}
	defer cancel()

	var title, currentURL string
	tasks := chromedp.Tasks{
		chromedp.Navigate(req.URL),
		chromedp.Sleep(500 * time.Millisecond),
	}
	if req.WaitFor != "" {
		tasks = append(tasks, chromedp.WaitVisible(req.WaitFor, chromedp.ByQuery))
	}
	tasks = append(tasks,
		chromedp.Title(&title),
		chromedp.Location(&currentURL),
	)

	if err := chromedp.Run(ctx, tasks...); err != nil {
		browserBroken = true
		writeJSON(w, 500, map[string]any{"success": false, "message": "탐색 실패: " + err.Error()})
		return
	}

	json200(w, map[string]any{
		"success":  true,
		"title":    title,
		"url":      currentURL,
		"message":  fmt.Sprintf("'%s' 로 이동했습니다", title),
	})
}

// ──────────────────────────────────────────────────────────────
// POST /api/browser/extract  — 페이지 텍스트/HTML 추출
// ──────────────────────────────────────────────────────────────

func handleBrowserExtract(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Selector string `json:"selector"` // CSS selector, 없으면 body
		Mode     string `json:"mode"`     // "text" | "html" | "links" | "table"
		URL      string `json:"url"`      // 먼저 이동 후 추출 (optional)
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Selector == "" {
		req.Selector = "body"
	}
	if req.Mode == "" {
		req.Mode = "text"
	}

	ctx, cancel, err := withBrowserTimeout(30 * time.Second)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}
	defer cancel()

	tasks := chromedp.Tasks{}
	if req.URL != "" {
		if !strings.HasPrefix(req.URL, "http") {
			req.URL = "https://" + req.URL
		}
		tasks = append(tasks,
			chromedp.Navigate(req.URL),
			chromedp.Sleep(1*time.Second),
		)
	}

	var result string
	switch req.Mode {
	case "html":
		tasks = append(tasks, chromedp.OuterHTML(req.Selector, &result, chromedp.ByQuery))
	case "links":
		// 모든 링크 추출
		tasks = append(tasks, chromedp.Evaluate(`
			Array.from(document.querySelectorAll('a[href]'))
				.map(a => ({text: a.innerText.trim(), href: a.href}))
				.filter(l => l.text && l.href)
				.slice(0, 50)
		`, &result))
	case "table":
		// 첫 번째 테이블 CSV 변환
		tasks = append(tasks, chromedp.Evaluate(`
			const tbl = document.querySelector('table');
			if (!tbl) return '테이블 없음';
			return Array.from(tbl.rows).map(r =>
				Array.from(r.cells).map(c => c.innerText.trim()).join('\t')
			).join('\n');
		`, &result))
	default:
		tasks = append(tasks, chromedp.Text(req.Selector, &result, chromedp.ByQuery))
	}

	if err := chromedp.Run(ctx, tasks...); err != nil {
		browserBroken = true
		writeJSON(w, 500, map[string]any{"success": false, "message": "추출 실패: " + err.Error()})
		return
	}

	// 텍스트 압축 (너무 길면 자름)
	if len(result) > 5000 {
		result = result[:5000] + "\n...(이하 생략)"
	}

	json200(w, map[string]any{
		"success":  true,
		"content":  result,
		"selector": req.Selector,
		"mode":     req.Mode,
		"length":   len(result),
	})
}

// ──────────────────────────────────────────────────────────────
// POST /api/browser/click  — 요소 클릭
// ──────────────────────────────────────────────────────────────

func handleBrowserClick(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Selector string `json:"selector"` // CSS selector
		Text     string `json:"text"`     // 텍스트로 찾기 (selector 없을 때)
		WaitMs   int    `json:"wait_ms"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	ctx, cancel, err := withBrowserTimeout(20 * time.Second)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}
	defer cancel()

	tasks := chromedp.Tasks{}

	if req.Selector != "" {
		tasks = append(tasks,
			chromedp.WaitVisible(req.Selector, chromedp.ByQuery),
			chromedp.Click(req.Selector, chromedp.ByQuery),
		)
	} else if req.Text != "" {
		// XPath로 텍스트 기반 클릭
		xpath := fmt.Sprintf(`//*[contains(text(), '%s')]`, req.Text)
		tasks = append(tasks,
			chromedp.WaitVisible(xpath, chromedp.BySearch),
			chromedp.Click(xpath, chromedp.BySearch),
		)
	} else {
		writeJSON(w, 400, map[string]any{"success": false, "message": "selector 또는 text 필요"})
		return
	}

	waitMs := req.WaitMs
	if waitMs == 0 {
		waitMs = 500
	}
	tasks = append(tasks, chromedp.Sleep(time.Duration(waitMs)*time.Millisecond))

	if err := chromedp.Run(ctx, tasks...); err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": "클릭 실패: " + err.Error()})
		return
	}

	json200(w, map[string]any{"success": true, "message": "클릭 완료"})
}

// ──────────────────────────────────────────────────────────────
// POST /api/browser/fill  — 입력 필드 채우기
// ──────────────────────────────────────────────────────────────

func handleBrowserFill(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Selector string `json:"selector"`
		Value    string `json:"value"`
		Submit   bool   `json:"submit"` // Enter 키 전송
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Selector == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "selector 필요"})
		return
	}

	ctx, cancel, err := withBrowserTimeout(20 * time.Second)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}
	defer cancel()

	tasks := chromedp.Tasks{
		chromedp.WaitVisible(req.Selector, chromedp.ByQuery),
		chromedp.Clear(req.Selector, chromedp.ByQuery),
		chromedp.SendKeys(req.Selector, req.Value, chromedp.ByQuery),
	}
	if req.Submit {
		tasks = append(tasks, chromedp.SendKeys(req.Selector, "\n", chromedp.ByQuery))
	}

	if err := chromedp.Run(ctx, tasks...); err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": "입력 실패: " + err.Error()})
		return
	}

	json200(w, map[string]any{"success": true, "message": "입력 완료"})
}

// ──────────────────────────────────────────────────────────────
// POST /api/browser/screenshot  — 브라우저 화면 캡처
// ──────────────────────────────────────────────────────────────

func handleBrowserScreenshot(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Selector string `json:"selector"` // 특정 요소만 캡처 (optional)
	}
	json.NewDecoder(r.Body).Decode(&req)

	ctx, cancel, err := withBrowserTimeout(15 * time.Second)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}
	defer cancel()

	var buf []byte
	var title, location string

	tasks := chromedp.Tasks{
		chromedp.Title(&title),
		chromedp.Location(&location),
	}

	if req.Selector != "" {
		tasks = append(tasks, chromedp.Screenshot(req.Selector, &buf, chromedp.NodeVisible, chromedp.ByQuery))
	} else {
		tasks = append(tasks, chromedp.FullScreenshot(&buf, 90))
	}

	if err := chromedp.Run(ctx, tasks...); err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": "캡처 실패: " + err.Error()})
		return
	}

	b64 := base64.StdEncoding.EncodeToString(buf)
	json200(w, map[string]any{
		"success":  true,
		"base64":   b64,
		"mime":     "image/png",
		"title":    title,
		"url":      location,
		"size_kb":  len(buf) / 1024,
	})
}

// ──────────────────────────────────────────────────────────────
// POST /api/browser/agent  — 자연어 명령 → 브라우저 자동화 + AI
// ──────────────────────────────────────────────────────────────

func handleBrowserAgent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Command string `json:"command"` // "쿠팡에서 노트북 최저가 찾아줘"
		MaxSteps int   `json:"max_steps"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Command == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "command 필요"})
		return
	}
	if req.MaxSteps == 0 {
		req.MaxSteps = 10
	}

	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()
	if gKey == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "Groq API 키 미설정"})
		return
	}

	// Step 1: AI가 브라우저 액션 계획 수립
	planPrompt := fmt.Sprintf(`당신은 웹 브라우저 자동화 에이전트입니다.
사용자 명령: "%s"

다음 JSON 형식으로 실행 단계를 계획하세요:
{
  "target_url": "이동할 URL",
  "goal": "달성 목표 설명",
  "steps": [
    {"action": "navigate|extract|click|fill|screenshot|search_and_extract", "params": {}, "description": "이 단계 설명"}
  ]
}

사용 가능한 actions:
- navigate: {"url": "URL"}
- extract: {"selector": "CSS셀렉터", "mode": "text|table|links"}
- click: {"selector": "셀렉터" or "text": "버튼텍스트"}
- fill: {"selector": "셀렉터", "value": "입력값", "submit": true/false}
- screenshot: {}
- search_and_extract: {"site": "coupang.com|naver.com 등", "query": "검색어", "extract_selector": "결과 셀렉터"}

최대 %d 단계로 구성하세요.`, req.Command, req.MaxSteps)

	planStr, _, err := callGroq(gKey, groqChatModel, []groqMsg{
		{Role: "user", Content: planPrompt},
	}, 1024, true)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": "계획 수립 실패: " + err.Error()})
		return
	}

	var plan struct {
		TargetURL string `json:"target_url"`
		Goal      string `json:"goal"`
		Steps     []struct {
			Action      string         `json:"action"`
			Params      map[string]any `json:"params"`
			Description string         `json:"description"`
		} `json:"steps"`
	}
	if err := json.Unmarshal([]byte(planStr), &plan); err != nil {
		writeJSON(w, 500, map[string]any{
			"success": false,
			"message": "계획 파싱 실패: " + err.Error(),
			"raw_plan": planStr,
		})
		return
	}

	// Step 2: 계획 실행
	type StepResult struct {
		Step        int    `json:"step"`
		Action      string `json:"action"`
		Description string `json:"description"`
		Success     bool   `json:"success"`
		Data        string `json:"data"`
		Error       string `json:"error,omitempty"`
	}

	var stepResults []StepResult
	var collectedData []string

	ctx, cancel, err := withBrowserTimeout(3 * time.Minute)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}
	defer cancel()

	for i, step := range plan.Steps {
		sr := StepResult{
			Step:        i + 1,
			Action:      step.Action,
			Description: step.Description,
		}

		switch step.Action {
		case "navigate":
			url, _ := step.Params["url"].(string)
			if url == "" {
				url = plan.TargetURL
			}
			if !strings.HasPrefix(url, "http") {
				url = "https://" + url
			}
			err := chromedp.Run(ctx,
				chromedp.Navigate(url),
				chromedp.Sleep(1500*time.Millisecond),
			)
			sr.Success = err == nil
			if err != nil {
				sr.Error = err.Error()
			}

		case "extract", "search_and_extract":
			selector, _ := step.Params["selector"].(string)
			mode, _ := step.Params["mode"].(string)
			query, _ := step.Params["query"].(string)
			site, _ := step.Params["site"].(string)

			// search_and_extract: 검색 후 추출
			if step.Action == "search_and_extract" && query != "" {
				searchURL := buildSearchURL(site, query)
				chromedp.Run(ctx, chromedp.Navigate(searchURL), chromedp.Sleep(2*time.Second))
				if selector == "" {
					extractSel, _ := step.Params["extract_selector"].(string)
					if extractSel != "" {
						selector = extractSel
					} else {
						selector = "body"
					}
				}
			}

			if selector == "" {
				selector = "body"
			}
			if mode == "" {
				mode = "text"
			}

			var content string
			var extractErr error
			switch mode {
			case "table":
				extractErr = chromedp.Run(ctx, chromedp.Evaluate(`
					const tbl = document.querySelector('table');
					if (!tbl) { return ''; }
					Array.from(tbl.rows).map(r =>
						Array.from(r.cells).map(c => c.innerText.trim()).join('\t')
					).join('\n')
				`, &content))
			default:
				extractErr = chromedp.Run(ctx, chromedp.Text(selector, &content, chromedp.ByQuery))
			}

			if len(content) > 3000 {
				content = content[:3000] + "..."
			}
			sr.Success = extractErr == nil
			sr.Data = content
			if extractErr != nil {
				sr.Error = extractErr.Error()
			} else {
				collectedData = append(collectedData, fmt.Sprintf("=== Step %d: %s ===\n%s", i+1, step.Description, content))
			}

		case "click":
			selector, _ := step.Params["selector"].(string)
			text, _ := step.Params["text"].(string)
			var clickErr error
			if selector != "" {
				clickErr = chromedp.Run(ctx,
					chromedp.WaitVisible(selector, chromedp.ByQuery),
					chromedp.Click(selector, chromedp.ByQuery),
					chromedp.Sleep(500*time.Millisecond),
				)
			} else if text != "" {
				xpath := fmt.Sprintf(`//*[contains(text(), '%s')]`, text)
				clickErr = chromedp.Run(ctx,
					chromedp.WaitVisible(xpath, chromedp.BySearch),
					chromedp.Click(xpath, chromedp.BySearch),
					chromedp.Sleep(500*time.Millisecond),
				)
			}
			sr.Success = clickErr == nil
			if clickErr != nil {
				sr.Error = clickErr.Error()
			}

		case "fill":
			selector, _ := step.Params["selector"].(string)
			value, _ := step.Params["value"].(string)
			submit, _ := step.Params["submit"].(bool)
			tasks := chromedp.Tasks{
				chromedp.WaitVisible(selector, chromedp.ByQuery),
				chromedp.Clear(selector, chromedp.ByQuery),
				chromedp.SendKeys(selector, value, chromedp.ByQuery),
			}
			if submit {
				tasks = append(tasks, chromedp.SendKeys(selector, "\n", chromedp.ByQuery))
				tasks = append(tasks, chromedp.Sleep(1500*time.Millisecond))
			}
			fillErr := chromedp.Run(ctx, tasks...)
			sr.Success = fillErr == nil
			if fillErr != nil {
				sr.Error = fillErr.Error()
			}

		case "screenshot":
			var buf []byte
			ssErr := chromedp.Run(ctx, chromedp.FullScreenshot(&buf, 85))
			sr.Success = ssErr == nil
			if ssErr == nil {
				sr.Data = base64.StdEncoding.EncodeToString(buf)
			} else {
				sr.Error = ssErr.Error()
			}
		}

		stepResults = append(stepResults, sr)

		// 치명적 오류 시 중단
		if !sr.Success && step.Action == "navigate" {
			break
		}
	}

	// Step 3: 수집 데이터 AI 요약
	finalSummary := ""
	if gKey != "" && len(collectedData) > 0 {
		var summaryPrompt string
		if isEnglishQuery(req.Command) {
			summaryPrompt = fmt.Sprintf(`User request: "%s"

Collected web data:
%s

Based on the data above, answer the user's request in English.
If price comparison, use a table format. If information gathering, use a key content list.`,
				req.Command, strings.Join(collectedData, "\n\n"))
		} else {
			summaryPrompt = fmt.Sprintf(`사용자 요청: "%s"

수집된 웹 데이터:
%s

위 데이터를 바탕으로 사용자 요청에 대한 답변을 한국어로 정리해주세요.
가격 비교라면 표 형식으로, 정보 수집이라면 핵심 내용 목록으로 정리하세요.`,
				req.Command, strings.Join(collectedData, "\n\n"))
		}

		summary, _, _ := callGroq(gKey, groqChatModel, []groqMsg{
			{Role: "user", Content: summaryPrompt},
		}, 1024, false)
		finalSummary = summary
	}

	json200(w, map[string]any{
		"success":       true,
		"goal":          plan.Goal,
		"steps_executed": len(stepResults),
		"steps":         stepResults,
		"summary":       finalSummary,
		"command":       req.Command,
	})
}

// ──────────────────────────────────────────────────────────────
// POST /api/browser/close  — 브라우저 닫기
// ──────────────────────────────────────────────────────────────

func handleBrowserClose(w http.ResponseWriter, r *http.Request) {
	browserMu.Lock()
	defer browserMu.Unlock()
	if browserCancel != nil {
		browserCancel()
		browserAlloc = nil
		browserCancel = nil
		browserCtx = nil
		browserBroken = false
	}
	json200(w, map[string]any{"success": true, "message": "브라우저가 닫혔습니다"})
}

// ──────────────────────────────────────────────────────────────
// GET /api/browser/status
// ──────────────────────────────────────────────────────────────

func handleBrowserStatus(w http.ResponseWriter, r *http.Request) {
	browserMu.Lock()
	active := browserAlloc != nil && !browserBroken
	browserMu.Unlock()

	// Chrome 설치 여부 확인
	chromeInstalled := isChromeInstalled()

	json200(w, map[string]any{
		"active":           active,
		"chrome_installed": chromeInstalled,
		"message": func() string {
			if active {
				return "브라우저 세션 활성"
			}
			return "브라우저 세션 없음"
		}(),
	})
}
