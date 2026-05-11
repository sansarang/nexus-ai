//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

// ──────────────────────────────────────────────────────────────
// POST /api/browser/smart-agent
// 자연어 명령 → 계획 수립 → 실행 → 결과 수집 → Excel/JSON 저장
// ──────────────────────────────────────────────────────────────

type SmartAgentStep struct {
	StepNum     int            `json:"step"`
	Action      string         `json:"action"`
	Description string         `json:"description"`
	Success     bool           `json:"success"`
	Data        interface{}    `json:"data,omitempty"`
	Error       string         `json:"error,omitempty"`
	Duration    string         `json:"duration"`
}

type SmartAgentResult struct {
	Success      bool             `json:"success"`
	Command      string           `json:"command"`
	Goal         string           `json:"goal"`
	Steps        []SmartAgentStep `json:"steps"`
	Summary      string           `json:"summary"`
	DataRows     [][]string       `json:"data_rows,omitempty"`   // 수집된 표 데이터
	ExcelPath    string           `json:"excel_path,omitempty"` // 저장된 Excel 파일
	JSONPath     string           `json:"json_path,omitempty"`
	Blocked      bool             `json:"blocked"`
	BlockReason  string           `json:"block_reason,omitempty"`
	Duration     string           `json:"duration"`
}

func handleBrowserSmartAgent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Command    string `json:"command"`    // "쿠팡에서 노트북 최저가 5곳 찾아 Excel로 정리해"
		MaxResults int    `json:"max_results"`
		SaveExcel  bool   `json:"save_excel"`
		SessionKey string `json:"session_key"` // 쿠키 세션 키
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Command == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "command 필요"})
		return
	}
	if req.MaxResults == 0 {
		req.MaxResults = 10
	}

	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()
	if gKey == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "Groq API 키 미설정"})
		return
	}

	// Chrome 미설치 시 Groq 폴백
	if !isChromeInstalled() {
		groqFallbackSearch(w, req.Command)
		return
	}

	startTime := time.Now()
	result := &SmartAgentResult{Command: req.Command}

	// ── Step 1: AI 작업 계획 수립 ──────────────────────────────
	planPrompt := fmt.Sprintf(`당신은 웹 브라우저 자동화 전문가입니다. 사용자 명령을 실행하기 위한 상세한 계획을 수립하세요.

명령: "%s"

단계별 실행 계획을 JSON으로 작성하세요:
{
  "goal": "달성할 목표 (한 문장)",
  "target_sites": ["사이트1", "사이트2"],
  "headers": ["수집할 데이터 열1", "열2", "열3"],
  "steps": [
    {
      "action": "navigate|search|extract_products|extract_table|extract_text|click|fill|wait|screenshot",
      "params": {"url": "...", "query": "...", "selector": "...", "site": "...", "max_items": 10},
      "description": "이 단계 설명"
    }
  ]
}

가능한 action 목록:
- navigate: params에 url 포함
- search: params에 site(coupang.com 등), query, max_items 포함
- extract_products: 현재 페이지에서 상품명/가격/링크 추출
- extract_table: 테이블 데이터 추출
- extract_text: 텍스트 추출, params에 selector
- click: params에 selector 또는 text
- fill: params에 selector, value
- wait: params에 ms(대기시간)

최대 10단계로 구성. 쿠팡/네이버는 anti-bot이 강하므로 search 단계 사용.`, req.Command)

	planStr, _, planErr := callGroq(gKey, groqChatModel, []groqMsg{{Role: "user", Content: planPrompt}}, 1024, true)
	if planErr != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": "계획 수립 실패: " + planErr.Error()})
		return
	}

	var plan struct {
		Goal        string     `json:"goal"`
		TargetSites []string   `json:"target_sites"`
		Headers     []string   `json:"headers"`
		Steps       []struct {
			Action      string                 `json:"action"`
			Params      map[string]interface{} `json:"params"`
			Description string                 `json:"description"`
		} `json:"steps"`
	}
	if err := json.Unmarshal([]byte(planStr), &plan); err != nil {
		// JSON 파싱 실패 → 기본 계획 사용
		plan.Goal = req.Command
		plan.Steps = []struct {
			Action      string                 `json:"action"`
			Params      map[string]interface{} `json:"params"`
			Description string                 `json:"description"`
		}{
			{Action: "search", Params: map[string]interface{}{"site": "coupang.com", "query": req.Command, "max_items": req.MaxResults}, Description: "쿠팡 검색"},
		}
	}
	result.Goal = plan.Goal

	// ── Step 2: Stealth 브라우저 실행 ─────────────────────────
	ctx, cancel, err := withStealthBrowserTimeout(5 * time.Minute)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": "브라우저 시작 실패: " + err.Error()})
		return
	}
	defer cancel()

	// 저장된 쿠키 복원
	if req.SessionKey != "" {
		loadCookies(ctx, req.SessionKey)
	}

	// ad blocking 활성화
	enableAdBlocking(ctx)

	// 수집된 데이터
	var collectedData [][]string
	if len(plan.Headers) > 0 {
		collectedData = append(collectedData, plan.Headers)
	}
	var rawTexts []string

	// ── Step 3: 계획 실행 ──────────────────────────────────────
	for i, step := range plan.Steps {
		stepStart := time.Now()
		sr := SmartAgentStep{
			StepNum:     i + 1,
			Action:      step.Action,
			Description: step.Description,
		}

		switch step.Action {
		case "navigate":
			url, _ := step.Params["url"].(string)
			if url != "" && !strings.HasPrefix(url, "http") {
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

		case "search":
			site, _ := step.Params["site"].(string)
			query, _ := step.Params["query"].(string)
			maxItemsF, _ := step.Params["max_items"].(float64)
			maxItems := int(maxItemsF)
			if maxItems == 0 {
				maxItems = req.MaxResults
			}

			products, searchErr := performSearch(ctx, site, query, maxItems)
			if searchErr != nil {
				sr.Success = false
				sr.Error = searchErr.Error()
				// 안티봇 감지 확인
				if blocked, reason := detectAntiBot(ctx); blocked {
					result.Blocked = true
					result.BlockReason = reason
				}
			} else {
				sr.Success = true
				sr.Data = products
				// 데이터 행으로 변환
				if len(collectedData) == 0 {
					collectedData = append(collectedData, []string{"상품명", "가격", "사이트", "링크"})
				}
				for _, p := range products {
					row := []string{
						p["name"], p["price"], site, p["link"],
					}
					collectedData = append(collectedData, row)
				}
			}

		case "extract_products":
			profile, _ := getSiteProfile("")
			var currentURL string
			chromedp.Run(ctx, chromedp.Location(&currentURL))
			profile, _ = getSiteProfile(currentURL)

			products, exErr := extractStructuredProducts(ctx, profile, req.MaxResults)
			if exErr != nil {
				sr.Success = false
				sr.Error = exErr.Error()
			} else {
				sr.Success = true
				sr.Data = products
				if len(collectedData) == 0 {
					collectedData = append(collectedData, []string{"상품명", "가격", "링크"})
				}
				for _, p := range products {
					collectedData = append(collectedData, []string{p["name"], p["price"], p["link"]})
				}
			}

		case "extract_table":
			tables, exErr := extractTableData(ctx)
			if exErr != nil {
				sr.Success = false
				sr.Error = exErr.Error()
			} else {
				sr.Success = true
				sr.Data = tables
				// 첫 번째 테이블 데이터 수집
				if len(tables) > 0 {
					if tbl, ok := tables[0]["rows"].([]interface{}); ok {
						if hdrs, ok := tables[0]["headers"].([]interface{}); ok && len(hdrs) > 0 {
							hdr := make([]string, len(hdrs))
							for j, h := range hdrs {
								hdr[j] = fmt.Sprint(h)
							}
							collectedData = append(collectedData, hdr)
						}
						for _, rowI := range tbl {
							if row, ok := rowI.([]interface{}); ok {
								r := make([]string, len(row))
								for j, c := range row {
									r[j] = fmt.Sprint(c)
								}
								collectedData = append(collectedData, r)
							}
						}
					}
				}
			}

		case "extract_text":
			sel, _ := step.Params["selector"].(string)
			if sel == "" {
				sel = "body"
			}
			var text string
			exErr := chromedp.Run(ctx, chromedp.Text(sel, &text, chromedp.ByQuery))
			if exErr != nil {
				sr.Success = false
				sr.Error = exErr.Error()
			} else {
				if len(text) > 3000 {
					text = text[:3000] + "..."
				}
				sr.Success = true
				sr.Data = text
				rawTexts = append(rawTexts, text)
			}

		case "click":
			sel, _ := step.Params["selector"].(string)
			txt, _ := step.Params["text"].(string)
			var clickErr error
			if sel != "" {
				clickErr = chromedp.Run(ctx,
					chromedp.WaitVisible(sel, chromedp.ByQuery),
					humanDelay(200, 600),
					chromedp.Click(sel, chromedp.ByQuery),
					humanDelay(500, 1500),
				)
			} else if txt != "" {
				xpath := fmt.Sprintf(`//*[contains(text(), '%s')]`, txt)
				clickErr = chromedp.Run(ctx,
					chromedp.WaitVisible(xpath, chromedp.BySearch),
					humanDelay(200, 600),
					chromedp.Click(xpath, chromedp.BySearch),
					humanDelay(500, 1500),
				)
			}
			sr.Success = clickErr == nil
			if clickErr != nil {
				sr.Error = clickErr.Error()
			}

		case "fill":
			sel, _ := step.Params["selector"].(string)
			val, _ := step.Params["value"].(string)
			submit, _ := step.Params["submit"].(bool)
			var fillErr error
			if sel != "" {
				fillErr = humanType(ctx, sel, val)
				if fillErr == nil && submit {
					fillErr = chromedp.Run(ctx,
						chromedp.SendKeys(sel, "\n", chromedp.ByQuery),
						humanDelay(1000, 2500),
					)
				}
			}
			sr.Success = fillErr == nil
			if fillErr != nil {
				sr.Error = fillErr.Error()
			}

		case "wait":
			msF, _ := step.Params["ms"].(float64)
			ms := int(msF)
			if ms == 0 {
				ms = 1000
			}
			chromedp.Run(ctx, chromedp.Sleep(time.Duration(ms)*time.Millisecond))
			sr.Success = true
		}

		sr.Duration = time.Since(stepStart).String()
		result.Steps = append(result.Steps, sr)

		// 치명적 실패 시 중단
		if !sr.Success && step.Action == "navigate" && i == 0 {
			break
		}
	}

	// 쿠키 저장
	if req.SessionKey != "" {
		saveCookies(ctx, req.SessionKey)
	}

	// ── Step 4: Excel 저장 ────────────────────────────────────
	if req.SaveExcel && len(collectedData) > 1 {
		desktopPath, _ := os.UserHomeDir()
		excelName := fmt.Sprintf("nexus_%s_%s.xlsx",
			sanitizeFilename(req.Command), time.Now().Format("20060102_150405"))
		excelPath := filepath.Join(desktopPath, "Desktop", excelName)

		if err := saveToExcel(collectedData, excelPath, result.Goal); err == nil {
			result.ExcelPath = excelPath
		}
	}

	// ── Step 5: AI 결과 요약 ──────────────────────────────────
	if gKey != "" && (len(collectedData) > 1 || len(rawTexts) > 0) {
		var dataForSummary string
		if len(collectedData) > 1 {
			lines := make([]string, 0, len(collectedData))
			for _, row := range collectedData {
				lines = append(lines, strings.Join(row, " | "))
			}
			dataForSummary = strings.Join(lines, "\n")
		} else {
			dataForSummary = strings.Join(rawTexts, "\n")
		}
		if len(dataForSummary) > 3000 {
			dataForSummary = dataForSummary[:3000]
		}

		summaryPrompt := fmt.Sprintf(`사용자 요청: "%s"
수집된 데이터:
%s

위 데이터를 바탕으로 사용자 요청에 완벽하게 답하는 요약을 한국어로 작성하세요.
- 가격 비교라면: 최저가 → 최고가 순서로 표 형태
- 뉴스/정보라면: 핵심 내용 3-5줄 요약
- 숫자/데이터라면: 주요 수치 하이라이트`, req.Command, dataForSummary)

		summary, _, _ := callGroq(gKey, groqChatModel, []groqMsg{{Role: "user", Content: summaryPrompt}}, 1024, false)
		result.Summary = summary
	}

	result.Success = true
	result.DataRows = collectedData
	result.Duration = time.Since(startTime).String()

	// 메모리 저장
	saveAgentMemory(AgentMemoryEntry{
		ID:        fmt.Sprintf("browser_%d", time.Now().Unix()),
		Timestamp: time.Now().Format(time.RFC3339),
		Type:      "browser_agent",
		Command:   req.Command,
		Result:    result.Summary,
		Success:   result.Success,
	})

	json200(w, result)
}

// performSearch: 사이트별 검색 실행 (stealth + human-like)
func performSearch(ctx interface{}, site, query string, maxItems int) ([]map[string]string, error) {
	// 새 stealth 컨텍스트 생성
	bCtx, bCancel, _ := withStealthBrowserTimeout(90 * time.Second)
	defer bCancel()

	profile, domain := getSiteProfile("https://" + site)
	searchURL := buildSearchURL(domain, query)
	_ = domain

	// 1. 검색 페이지 이동
	if err := chromedp.Run(bCtx,
		chromedp.Navigate(searchURL),
		humanDelay(1500, 3000),
	); err != nil {
		return nil, fmt.Errorf("검색 페이지 이동 실패: %w", err)
	}

	// 안티봇 확인
	if blocked, reason := detectAntiBot(bCtx); blocked {
		return nil, fmt.Errorf("봇 차단 감지: %s", reason)
	}

	// 2. 페이지 안정화 대기
	waitForPageStable(bCtx)
	chromedp.Run(bCtx, humanDelay(800, 1500))

	// 3. 자연스러운 스크롤
	humanScroll(bCtx, 300)

	// 4. 상품 추출
	products, err := extractStructuredProducts(bCtx, profile, maxItems)
	if err != nil || len(products) == 0 {
		// fallback: 일반 텍스트 추출
		var pageText string
		chromedp.Run(bCtx, chromedp.Text("body", &pageText, chromedp.ByQuery))
		if len(pageText) > 2000 {
			pageText = pageText[:2000]
		}
		return []map[string]string{{"name": "텍스트 추출", "price": "", "link": searchURL, "raw": pageText}}, nil
	}

	return products, nil
}

// ──────────────────────────────────────────────────────────────
// POST /api/browser/collect-price
// 쇼핑몰 가격 비교 특화 핸들러
// ──────────────────────────────────────────────────────────────

func handleBrowserCollectPrice(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProductQuery string   `json:"product_query"` // "삼성 갤럭시북4 프로 16인치"
		Sites        []string `json:"sites"`          // ["coupang.com", "danawa.com"]
		MaxPerSite   int      `json:"max_per_site"`
		SaveExcel    bool     `json:"save_excel"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.ProductQuery == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "product_query 필요"})
		return
	}
	if len(req.Sites) == 0 {
		req.Sites = []string{"coupang.com", "danawa.com", "gmarket.co.kr"}
	}
	if req.MaxPerSite == 0 {
		req.MaxPerSite = 5
	}

	type PriceResult struct {
		Site     string `json:"site"`
		Name     string `json:"name"`
		Price    string `json:"price"`
		Link     string `json:"link"`
		Blocked  bool   `json:"blocked"`
	}

	var allResults []PriceResult
	var headers = []string{"순위", "사이트", "상품명", "가격", "링크"}
	var excelData = [][]string{headers}

	for _, site := range req.Sites {
		sCtx, sCancel, sErr := withStealthBrowserTimeout(90 * time.Second)
		if sErr != nil {
			allResults = append(allResults, PriceResult{Site: site, Blocked: true, Name: "브라우저 실패"})
			sCancel()
			continue
		}

		products, searchErr := performSearch(sCtx, site, req.ProductQuery, req.MaxPerSite)
		sCancel()

		if searchErr != nil {
			allResults = append(allResults, PriceResult{Site: site, Blocked: true, Name: searchErr.Error()})
			continue
		}

		for _, p := range products {
			allResults = append(allResults, PriceResult{
				Site:  site,
				Name:  p["name"],
				Price: p["price"],
				Link:  p["link"],
			})
			excelData = append(excelData, []string{
				fmt.Sprintf("%d", len(excelData)),
				site, p["name"], p["price"], p["link"],
			})
		}
	}

	// Excel 저장
	excelPath := ""
	if req.SaveExcel && len(excelData) > 1 {
		desktopPath, _ := os.UserHomeDir()
		fname := fmt.Sprintf("price_compare_%s.xlsx", time.Now().Format("20060102_150405"))
		fpath := filepath.Join(desktopPath, "Desktop", fname)
		if err := saveToExcel(excelData, fpath, req.ProductQuery+" 가격 비교"); err == nil {
			excelPath = fpath
		}
	}

	// AI 최저가 분석
	summary := ""
	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()
	if gKey != "" && len(allResults) > 0 {
		dataLines := make([]string, 0, len(allResults))
		for _, r := range allResults {
			if !r.Blocked {
				dataLines = append(dataLines, fmt.Sprintf("%s | %s | %s", r.Site, r.Name, r.Price))
			}
		}
		summaryPrompt := fmt.Sprintf(`검색: "%s"
가격 데이터:
%s

최저가 순으로 정리하고, 구매 추천 상품을 한국어로 설명해주세요.`, req.ProductQuery, strings.Join(dataLines, "\n"))

		summary, _, _ = callGroq(gKey, groqChatModel, []groqMsg{{Role: "user", Content: summaryPrompt}}, 512, false)
	}

	json200(w, map[string]any{
		"success":    true,
		"query":      req.ProductQuery,
		"results":    allResults,
		"total":      len(allResults),
		"summary":    summary,
		"excel_path": excelPath,
	})
}

// ──────────────────────────────────────────────────────────────
// POST /api/browser/news-collect
// 뉴스/주가/정보 수집 특화
// ──────────────────────────────────────────────────────────────

func handleBrowserNewsCollect(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query     string `json:"query"`  // "삼성전자 오늘 뉴스"
		Site      string `json:"site"`   // "finance.naver.com"
		MaxItems  int    `json:"max_items"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.Query == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "query 필요"})
		return
	}
	if req.Site == "" {
		req.Site = "naver.com"
	}
	if req.MaxItems == 0 {
		req.MaxItems = 10
	}

	ctx, cancel, err := withStealthBrowserTimeout(90 * time.Second)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}
	defer cancel()

	searchURL := buildSearchURL(req.Site, req.Query)
	if err := chromedp.Run(ctx,
		chromedp.Navigate(searchURL),
		humanDelay(1500, 2500),
	); err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": "페이지 이동 실패: " + err.Error()})
		return
	}

	waitForPageStable(ctx)

	// 뉴스 기사 추출
	extractJS := fmt.Sprintf(`
	JSON.stringify(Array.from(document.querySelectorAll('a')).filter(a =>
		a.innerText.length > 20 && a.href.includes('news')
	).slice(0, %d).map(a => ({
		title: a.innerText.trim(),
		url: a.href
	})))
	`, req.MaxItems)

	var raw string
	chromedp.Run(ctx, chromedp.Evaluate(extractJS, &raw))

	var articles []map[string]string
	json.Unmarshal([]byte(raw), &articles)

	// AI 뉴스 요약
	summary := ""
	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()
	if gKey != "" && len(articles) > 0 {
		titles := make([]string, 0, len(articles))
		for _, a := range articles {
			titles = append(titles, a["title"])
		}
		summaryPrompt := fmt.Sprintf(`검색: "%s"
뉴스 기사 목록:
%s

위 뉴스들의 주요 트렌드와 핵심 내용을 3-5줄로 한국어로 요약해주세요.`, req.Query, strings.Join(titles, "\n"))
		summary, _, _ = callGroq(gKey, groqChatModel, []groqMsg{{Role: "user", Content: summaryPrompt}}, 512, false)
	}

	json200(w, map[string]any{
		"success":  true,
		"query":    req.Query,
		"articles": articles,
		"total":    len(articles),
		"summary":  summary,
	})
}

// ──────────────────────────────────────────────────────────────
// POST /api/browser/login-session
// 사이트 로그인 + 세션 저장
// ──────────────────────────────────────────────────────────────

func handleBrowserLoginSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL          string `json:"url"`
		UsernameSel  string `json:"username_selector"`
		PasswordSel  string `json:"password_selector"`
		SubmitSel    string `json:"submit_selector"`
		Username     string `json:"username"`
		Password     string `json:"password"`
		SessionKey   string `json:"session_key"`
		SuccessCheck string `json:"success_check"` // 로그인 성공 확인 셀렉터
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.URL == "" || req.Username == "" || req.Password == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "url, username, password 필요"})
		return
	}
	if req.SessionKey == "" {
		req.SessionKey = "default"
	}

	ctx, cancel, err := withStealthBrowserTimeout(90 * time.Second)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}
	defer cancel()

	// 기존 세션 로드
	loadCookies(ctx, req.SessionKey)

	usernameSel := req.UsernameSel
	if usernameSel == "" {
		usernameSel = "input[type='email'], input[name='username'], input[name='userId'], #id"
	}
	passwordSel := req.PasswordSel
	if passwordSel == "" {
		passwordSel = "input[type='password'], input[name='password'], #pw"
	}
	submitSel := req.SubmitSel
	if submitSel == "" {
		submitSel = "button[type='submit'], .btn-login, #loginBtn"
	}

	// 로그인 시도
	if err := chromedp.Run(ctx,
		chromedp.Navigate(req.URL),
		humanDelay(1000, 2000),
	); err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": "페이지 이동 실패"})
		return
	}

	if err := humanType(ctx, usernameSel, req.Username); err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": "아이디 입력 실패: " + err.Error()})
		return
	}
	chromedp.Run(ctx, humanDelay(300, 700))

	if err := humanType(ctx, passwordSel, req.Password); err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": "비밀번호 입력 실패: " + err.Error()})
		return
	}
	chromedp.Run(ctx, humanDelay(500, 1000))

	// 로그인 버튼 클릭
	chromedp.Run(ctx,
		chromedp.Click(submitSel, chromedp.ByQuery),
		humanDelay(2000, 3000),
	)

	// 성공 확인
	var loginSuccess bool
	if req.SuccessCheck != "" {
		var found bool
		chromedp.Run(ctx, chromedp.Evaluate(
			fmt.Sprintf(`document.querySelector('%s') !== null`, req.SuccessCheck),
			&found,
		))
		loginSuccess = found
	} else {
		loginSuccess = true // 확인 셀렉터 없으면 성공으로 간주
	}

	if loginSuccess {
		saveCookies(ctx, req.SessionKey)
	}

	json200(w, map[string]any{
		"success":     loginSuccess,
		"session_key": req.SessionKey,
		"message": func() string {
			if loginSuccess {
				return "로그인 성공 - 세션 저장됨"
			}
			return "로그인 실패 (비밀번호 확인 필요)"
		}(),
	})
}

// ──────────────────────────────────────────────────────────────
// 유틸리티
// ──────────────────────────────────────────────────────────────

func sanitizeFilename(s string) string {
	replacer := strings.NewReplacer(
		"/", "_", "\\", "_", ":", "_", "*", "_",
		"?", "_", "\"", "_", "<", "_", ">", "_", "|", "_",
		" ", "_",
	)
	s = replacer.Replace(s)
	if len(s) > 50 {
		s = s[:50]
	}
	return s
}
