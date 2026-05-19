//go:build !windows

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func cmdWeather(cx cmdCtx) {
		city := "서울"
		if c, ok := cx.params["city"].(string); ok && c != "" {
			city = c
		}
		// wttr.in 실시간 날씨 API 호출
		wText := fetchWeatherText(city, cx.gKey)
		json200(cx.w, CommandResponse{
			Success:  true,
			Message:  wText,
			Action:   "weather",
			Result:   map[string]any{"city": city},
			Duration: cx.dur,
		})

}

func cmdPriceCompare(cx cmdCtx) {
		var query, site string
		maxItems := 8
		if cx.params != nil {
			query, _ = cx.params["query"].(string)
			site, _ = cx.params["site"].(string)
			if v, ok := cx.params["max_items"].(float64); ok {
				maxItems = int(v)
			}
		}
		if query == "" {
			query = cx.req.Message
		}
		llmMu.RLock()
		priceTKey := llmTavilyKey
		llmMu.RUnlock()
		var priceItems []map[string]string
		if priceTKey != "" {
			// include_domains 방식 사용 (site: 접두사는 결과 0개 버그 있음)
			if site != "" {
				if tr, ok := tavilySearchDomain(priceTKey, query, maxItems, site); ok {
					priceItems = tr.Items
				}
			}
			if len(priceItems) == 0 {
				if tr, ok := tavilySearch(priceTKey, query, maxItems); ok {
					priceItems = tr.Items
				}
			}
		}
		siteName := site
		if siteName == "" {
			siteName = "쇼핑몰"
		}
		if len(priceItems) == 0 {
			enc := strings.ReplaceAll(query, " ", "+")
			priceItems = []map[string]string{
				{"title": fmt.Sprintf("%s 검색: %s", siteName, query), "url": fmt.Sprintf("https://www.%s/search?q=%s", site, enc)},
			}
		}
		summary := fmt.Sprintf("%s에서 \"%s\" 상품 %d개를 찾았어요!", siteName, query, len(priceItems))
		results := make([]map[string]string, 0, len(priceItems))
		for _, it := range priceItems {
			results = append(results, map[string]string{"site": siteName, "name": it["title"], "price": "", "link": it["url"]})
		}
		json200(cx.w, CommandResponse{
			Success: true, Message: summary, Action: "price_compare",
			Result:   map[string]any{"query": query, "site": site, "summary": summary, "results": results, "total": len(results)},
			Duration: cx.dur,
		})

}

func cmdVideoSearch(cx cmdCtx) {
		var query, platform string
		maxItems := 8
		if cx.params != nil {
			query, _ = cx.params["query"].(string)
			platform, _ = cx.params["platform"].(string)
			if v, ok := cx.params["max_items"].(float64); ok {
				maxItems = int(v)
			}
		}
		if query == "" {
			query = cx.req.Message
		}
		llmMu.RLock()
		videoTKey := llmTavilyKey
		llmMu.RUnlock()
		isTikTok := platform == "tiktok" ||
			strings.Contains(strings.ToLower(cx.req.Message), "틱톡") ||
			strings.Contains(strings.ToLower(cx.req.Message), "tiktok")
		var videoItems []map[string]string
		if isTikTok {
			// site: 접두사 0결과 버그 → include_domains 방식 사용
			if videoTKey != "" {
				if tr, ok := tavilySearchDomain(videoTKey, query, maxItems, "tiktok.com"); ok {
					for _, it := range tr.Items {
						if strings.Contains(it["url"], "tiktok.com") {
							videoItems = append(videoItems, it)
						}
					}
				}
				if len(videoItems) == 0 {
					if tr, ok := tavilySearch(videoTKey, query+" tiktok", maxItems); ok {
						for _, it := range tr.Items {
							if strings.Contains(it["url"], "tiktok.com") {
								videoItems = append(videoItems, it)
							}
						}
					}
				}
			}
			if len(videoItems) == 0 {
				enc := strings.ReplaceAll(query, " ", "%20")
				videoItems = []map[string]string{
					{"title": fmt.Sprintf("TikTok에서 \"%s\" 검색", query), "url": fmt.Sprintf("https://www.tiktok.com/search?q=%s", enc)},
					{"title": "TikTok 트렌딩", "url": "https://www.tiktok.com/trending"},
				}
			}
			summary := fmt.Sprintf("TikTok에서 \"%s\" 영상 %d개를 찾았어요!", query, len(videoItems))
			json200(cx.w, CommandResponse{
				Success: true, Message: summary, Action: "video_search",
				Result:   map[string]any{"query": query, "platform": "tiktok", "items": videoItems, "total": len(videoItems)},
				Duration: cx.dur,
			})
		} else {
			// site: 접두사 0결과 버그 → include_domains 방식 사용
			if videoTKey != "" {
				if tr, ok := tavilySearchDomain(videoTKey, query, maxItems, "youtube.com"); ok {
					for _, it := range tr.Items {
						if strings.Contains(it["url"], "youtube.com/watch") || strings.Contains(it["url"], "youtu.be") {
							videoItems = append(videoItems, it)
						}
					}
				}
				if len(videoItems) == 0 {
					if tr, ok := tavilySearch(videoTKey, query+" youtube 영상", maxItems); ok {
						for _, it := range tr.Items {
							if strings.Contains(it["url"], "youtube.com/watch") || strings.Contains(it["url"], "youtu.be") {
								videoItems = append(videoItems, it)
							}
						}
					}
				}
			}
			if len(videoItems) == 0 {
				enc := strings.ReplaceAll(query, " ", "%20")
				videoItems = []map[string]string{
					{"title": fmt.Sprintf("YouTube에서 \"%s\" 검색", query), "url": fmt.Sprintf("https://www.youtube.com/results?search_query=%s", enc)},
				}
			}
			summary := fmt.Sprintf("YouTube에서 \"%s\" 영상 %d개를 찾았어요!", query, len(videoItems))
			json200(cx.w, CommandResponse{
				Success: true, Message: summary, Action: "video_search",
				Result:   map[string]any{"query": query, "platform": "youtube", "items": videoItems, "total": len(videoItems)},
				Duration: cx.dur,
			})
		}

}

func cmdWebSearch(cx cmdCtx) {
		var query, site string
		maxItems := 5
		if cx.params != nil {
			query, _ = cx.params["query"].(string)
			site, _ = cx.params["site"].(string)
			if v, ok := cx.params["max_items"].(float64); ok {
				maxItems = int(v)
			}
		}
		if query == "" {
			query = cx.req.Message
		}
		wsLang := cx.req.Lang
		if wsLang == "" {
			if isEnglishQuery(cx.req.Message) {
				wsLang = "en"
			} else {
				wsLang = "ko"
			}
		}
		result := runWebSearchMac(cx.gKey, query, site, maxItems, wsLang)
		appendSession(cx.userID, "user", cx.req.Message)
		appendSession(cx.userID, "assistant", result.Summary)
		json200(cx.w, CommandResponse{
			Success:  true,
			Message:  result.Summary,
			Action:   "web_search",
			Result:   result,
			Duration: cx.dur,
		})

}

func cmdTripPlan(cx cmdCtx) {
		destination, _ := cx.params["destination"].(string)
		date, _ := cx.params["date"].(string)
		purpose, _ := cx.params["purpose"].(string)
		if destination == "" {
			destination = cx.req.Message
		}
		if date == "" {
			date = time.Now().AddDate(0, 0, 1).Format("2006-01-02")
		}
		if purpose == "" {
			purpose = "출장"
		}

		var tripSections []string
		// 병렬로 정보 수집
		type section struct {
			name string
			body string
		}
		ch := make(chan section, 5)

		// 날씨
		go func() {
			tr, ok := tavilySearch(llmTavilyKey, destination+" 날씨 "+date, 3)
			if ok {
				ch <- section{"날씨", tr.Summary}
			} else {
				ch <- section{"날씨", ""}
			}
		}()
		// 항공권
		go func() {
			tr, ok := tavilySearch(llmTavilyKey, "서울 "+destination+" 항공권 "+date+" 가격", 3)
			if ok {
				ch <- section{"항공권", tr.Summary}
			} else {
				ch <- section{"항공권", ""}
			}
		}()
		// 호텔
		go func() {
			tr, ok := tavilySearch(llmTavilyKey, destination+" 호텔 추천 "+date, 3)
			if ok {
				ch <- section{"호텔", tr.Summary}
			} else {
				ch <- section{"호텔", ""}
			}
		}()
		// 맛집
		go func() {
			tr, ok := tavilySearch(llmTavilyKey, destination+" 맛집 추천 현지인", 3)
			if ok {
				ch <- section{"맛집", tr.Summary}
			} else {
				ch <- section{"맛집", ""}
			}
		}()
		// 환율
		go func() {
			tr, ok := tavilySearch(llmTavilyKey, destination+" 환율 오늘", 2)
			if ok {
				ch <- section{"환율", tr.Summary}
			} else {
				ch <- section{"환율", ""}
			}
		}()

		collected := map[string]string{}
		for i := 0; i < 5; i++ {
			s := <-ch
			if s.body != "" {
				collected[s.name] = s.body
			}
		}

		for _, key := range []string{"날씨", "항공권", "호텔", "맛집", "환율"} {
			if v, ok := collected[key]; ok && v != "" {
				tripSections = append(tripSections, fmt.Sprintf("### %s\n%s", key, v))
			}
		}

		tripEng := isEnglishQuery(destination)
		var prompt string
		if tripEng {
			prompt = fmt.Sprintf(`Prepare a travel checklist for %s %s based on the following information. Write clearly in English.

%s

Checklist format:
1. Weather & packing
2. Flight information
3. Hotel recommendations
4. Local restaurants
5. Currency & budget
6. Other preparations`, destination, date, strings.Join(tripSections, "\n\n"))
		} else {
			prompt = fmt.Sprintf(`%s %s 출장/여행 준비 사항을 다음 정보를 바탕으로 한국어로 깔끔하게 정리해줘.

%s

체크리스트 형식으로 작성해줘:
1. 날씨 및 준비물
2. 항공권 정보
3. 숙소 추천
4. 현지 맛집
5. 환율 및 예산
6. 기타 준비 사항`, destination, date, strings.Join(tripSections, "\n\n"))
		}

		result, _, _ := callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 1000, false)

		// 파일 저장
		home, _ := os.UserHomeDir()
		fname := fmt.Sprintf("trip_%s_%s.md", strings.ReplaceAll(destination, " ", "_"), date)
		fpath := filepath.Join(home, "Desktop", fname)
		os.WriteFile(fpath, []byte(fmt.Sprintf("# %s %s %s 준비\n\n%s", purpose, destination, date, result)), 0644)

		json200(cx.w, CommandResponse{
			Success: true,
			Message: result,
			Action:  "trip_plan",
			Result: map[string]any{
				"destination": destination,
				"date":        date,
				"purpose":     purpose,
				"file":        fpath,
				"sections":    collected,
			},
			Duration: cx.dur,
		})

}

func cmdExchangeRate(cx cmdCtx) {
		erEng := cx.req.Lang == "en" || isEnglishQuery(cx.req.Message)
		fromC, toC := detectCurrencies(cx.req.Message)
		if p := cx.params; p != nil {
			if v, _ := p["from"].(string); v != "" { fromC = strings.ToUpper(v) }
			if v, _ := p["to"].(string); v != "" { toC = strings.ToUpper(v) }
		}
		rate, date, err := fetchExchangeRate(fromC, toC)
		if err != nil {
			// fallback: web_search
			q := fromC + " to " + toC + " exchange rate today"
			if !erEng { q = fromC + " " + toC + " 오늘 환율" }
			r := runWebSearchMac(cx.gKey, q, "auto", 3, cx.req.Lang)
			json200(cx.w, CommandResponse{Success: true, Message: r.Summary, Action: "exchange_rate", Duration: cx.dur})
		} else {
			fromN := currencySymbols[fromC]; if fromN == "" { fromN = fromC }
			toN := currencySymbols[toC]; if toN == "" { toN = toC }
			var msg string
			if erEng {
				msg = fmt.Sprintf("1 %s (%s) = **%.4f %s (%s)**\n_(as of %s)_", fromN, fromC, rate, toN, toC, date)
			} else {
				msg = fmt.Sprintf("1 %s(%s) = **%.4f %s(%s)**\n_(%s 기준)_", fromN, fromC, rate, toN, toC, date)
			}
			appendSession(cx.userID, "user", cx.req.Message)
			appendSession(cx.userID, "assistant", msg)
			json200(cx.w, CommandResponse{Success: true, Message: msg, Action: "exchange_rate",
				Result: map[string]any{"from": fromC, "to": toC, "rate": rate, "date": date}, Duration: cx.dur})
		}

	// ── 🔴 1. 주가 ──────────────────────────────────────────────
}

func cmdStock(cx cmdCtx) {
		stEng := cx.req.Lang == "en" || isEnglishQuery(cx.req.Message)
		var stQuery string
		if cx.params != nil { stQuery, _ = cx.params["query"].(string) }
		if stQuery == "" { stQuery = cx.req.Message }
		// 암호화폐 먼저 체크
		if cryptoSym := detectCrypto(stQuery); cryptoSym != "" {
			krw, usd, err := fetchCryptoPrice(cryptoSym)
			var msg string
			if err != nil {
				r := runWebSearchMac(cx.gKey, cryptoSym+" 현재 가격", "auto", 3, cx.req.Lang)
				msg = r.Summary
			} else {
				if stEng {
					msg = fmt.Sprintf("**%s**: ₩%.0f KRW / $%.2f USD", cryptoSym, krw, usd)
				} else {
					msg = fmt.Sprintf("**%s** 현재가: **₩%.0f** (KRW) / $%.2f (USD)", cryptoSym, krw, usd)
				}
			}
			appendSession(cx.userID, "user", cx.req.Message)
			appendSession(cx.userID, "assistant", msg)
			json200(cx.w, CommandResponse{Success: true, Message: msg, Action: "stock", Duration: cx.dur})
			return
		}
		// 주식 티커 검색
		ticker, name := detectStockTicker(stQuery)
		if ticker == "" {
			r := runWebSearchMac(cx.gKey, stQuery+" 주가 현재", "auto", 3, cx.req.Lang)
			json200(cx.w, CommandResponse{Success: true, Message: r.Summary, Action: "stock", Duration: cx.dur})
			return
		}
		price, change, currency, err := fetchStockInfo(ticker)
		var stMsg string
		if err != nil {
			r := runWebSearchMac(cx.gKey, name+" 주가 현재", "auto", 3, cx.req.Lang)
			stMsg = r.Summary
		} else {
			stMsg = formatStockMsg(name, ticker, price, change, currency, stEng)
		}
		appendSession(cx.userID, "user", cx.req.Message)
		appendSession(cx.userID, "assistant", stMsg)
		json200(cx.w, CommandResponse{Success: true, Message: stMsg, Action: "stock",
			Result: map[string]any{"ticker": ticker, "price": price, "change": change}, Duration: cx.dur})

}

func cmdDeepResearch(cx cmdCtx) {
		// Perplexity sonar-pro — 실시간 웹 리서치 (Manus 대체)
		drEng := cx.req.Lang == "en" || isEnglishQuery(cx.req.Message)
		drQuery := cx.req.Message
		if cx.params != nil {
			if q, ok := cx.params["query"].(string); ok && q != "" {
				drQuery = q
			}
		}
		// 1차: Tavily 빠른 검색
		tvResult, _ := tavilySearch(llmTavilyKey, drQuery, 5)
		// 2차: Perplexity sonar-pro로 깊은 분석 (웹검색 내장)
		var sysCtx string
		if drEng {
			sysCtx = "You are a research assistant. Provide comprehensive, well-structured answers with key facts, data, and analysis. Use bullet points and headers for clarity."
		} else {
			sysCtx = "당신은 심층 리서치 전문가입니다. 핵심 사실, 데이터, 분석을 포함한 구조화된 답변을 제공하세요. 불릿 포인트와 소제목을 활용하세요."
		}
		var drPrompt string
		if tvResult.Summary != "" {
			if drEng {
				drPrompt = fmt.Sprintf("Research context from web:\n%s\n\nUser question: %s\n\nProvide a comprehensive, well-structured answer.", tvResult.Summary, drQuery)
			} else {
				drPrompt = fmt.Sprintf("웹 검색 컨텍스트:\n%s\n\n질문: %s\n\n위 정보를 바탕으로 심층적이고 구조화된 답변을 제공해줘.", tvResult.Summary, drQuery)
			}
		} else {
			if drEng {
				drPrompt = fmt.Sprintf("Research and answer comprehensively: %s", drQuery)
			} else {
				drPrompt = fmt.Sprintf("다음 주제를 심층 리서치하고 구조화된 답변을 제공해줘: %s", drQuery)
			}
		}
		drMsgs := []groqMsg{
			{Role: "system", Content: sysCtx},
			{Role: "user", Content: drPrompt},
		}
		drAnswer, _, drErr := callGroqWithFallback(drMsgs, 2048, false)
		if drErr != nil {
			if drEng {
				drAnswer = "Research failed: " + drErr.Error()
			} else {
				drAnswer = "리서치 실패: " + drErr.Error()
			}
		}
		appendSession(cx.userID, "user", cx.req.Message)
		appendSession(cx.userID, "assistant", drAnswer)
		json200(cx.w, CommandResponse{Success: true, Message: drAnswer, Action: "deep_research", Duration: cx.dur})
}

// ── 웹 검색 (Groq 기반 + 브라우저 에이전트) ───────────────────

type webSearchResult struct {
	Query       string              `json:"query"`
	Site        string              `json:"site"`
	Summary     string              `json:"summary"`
	Items       []map[string]string `json:"items,omitempty"`
	PreviewType string              `json:"preview_type,omitempty"`
}

func runWebSearchMac(apiKey, query, site string, maxItems int, lang ...string) webSearchResult {
	forceLang := ""
	if len(lang) > 0 {
		forceLang = lang[0]
	}
	eng := forceLang == "en"
	siteLabel := site
	if siteLabel == "" || siteLabel == "auto" {
		if eng {
			siteLabel = "web"
		} else {
			siteLabel = "웹"
		}
	}

	cat := detectCategory(query)
	previewType := categoryPreviewType(cat)

	// 병렬 검색: Tavily + 브라우저 동시 실행
	result := parallelWebSearch(query, maxItems, forceLang)

	// 결과가 있으면 그대로 반환
	if result.Summary != "" || len(result.Items) > 0 {
		items := result.Items
		if len(items) == 0 {
			items = categoryFallbackSites(query, cat)
		}
		return webSearchResult{
			Query:       query,
			Site:        siteLabel,
			Summary:     result.Summary,
			Items:       items,
			PreviewType: previewType,
		}
	}

	// 최후 폴백: Groq LLM (실시간 데이터 없음)
	today := time.Now().Format("2006-01-02")
	var prompt string
	if eng {
		prompt = fmt.Sprintf(`Today is %s.
User question: "%s"

[Instructions]
- Do NOT include URLs, links, or source names
- Answer directly in natural English, 2-4 sentences, key points only
- If no real-time data available, say "For the latest info, please use the preview button"
- Write like a friendly AI assistant`, today, query)
	} else {
		prompt = fmt.Sprintf(`오늘은 %s입니다.
사용자 질문: "%s"

[지시사항]
- URL, 링크, 출처명 절대 포함 금지
- 사용자 질문에 직접 답하는 자연스러운 한국어 2~4문장으로 핵심만 답변
- 실시간 데이터가 없으면 "정확한 최신 정보는 미리보기 버튼으로 확인해보세요" 안내
- 친절한 AI 비서처럼 작성`, today, query)
	}
	msgs := []groqMsg{{Role: "user", Content: prompt}}
	text, _, err := callGroqWithFallback(msgs, 512, false)
	if err != nil {
		text = "검색 중 오류가 발생했습니다: " + err.Error()
	}

	fallbackItems := categoryFallbackSites(query, cat)
	if len(fallbackItems) == 0 {
		fallbackItems = buildFallbackURLs(query, site)
	}

	return webSearchResult{
		Query:       query,
		Site:        siteLabel,
		Summary:     text,
		Items:       fallbackItems,
		PreviewType: previewType,
	}
}


func tryBrowserSearch(query, site string, maxItems int) []map[string]string {
	// chromedp가 사용 가능하면 실제 검색, 없으면 빈 결과
	defer func() { recover() }()

	ctx, cancel, err := getBrowserCtxMac()
	if err != nil {
		return nil
	}
	defer cancel()

	var searchURL string
	switch strings.ToLower(site) {
	case "youtube":
		searchURL = "https://www.youtube.com/results?search_query=" + urlEncode(query)
	case "coupang":
		searchURL = "https://www.coupang.com/np/search?q=" + urlEncode(query)
	case "naver":
		searchURL = "https://search.naver.com/search.naver?query=" + urlEncode(query)
	default:
		searchURL = "https://www.google.com/search?q=" + urlEncode(query)
	}

	_ = ctx
	_ = searchURL
	_ = cancel
	_ = maxItems
	return nil
}


// runMacOrchestrate: Mac용 멀티 에이전트 오케스트레이터 (Tavily 기반 순차 실행)
func runMacOrchestrate(goal, gKey string) (string, error) {
	eng := isEnglishQuery(goal)
	llmMu.RLock()
	tKey := llmTavilyKey
	llmMu.RUnlock()

	// 1단계: 목표를 서브 태스크로 분해 (LLM)
	var planPrompt string
	if eng {
		planPrompt = fmt.Sprintf(`Break down this goal into 2-3 concrete search queries (JSON array of strings only):
Goal: %s
Output format: ["query1","query2","query3"]`, goal)
	} else {
		planPrompt = fmt.Sprintf(`다음 목표를 2-3개의 구체적인 검색 쿼리로 분해하세요 (JSON 배열만 출력):
목표: %s
출력 형식: ["쿼리1","쿼리2","쿼리3"]`, goal)
	}

	raw, _, err := callGroqWithFallback([]groqMsg{{Role: "user", Content: planPrompt}}, 256, true)
	if err != nil {
		raw = fmt.Sprintf(`["%s"]`, goal)
	}

	var queries []string
	if jsonErr := json.Unmarshal([]byte(raw), &queries); jsonErr != nil || len(queries) == 0 {
		queries = []string{goal}
	}
	if len(queries) > 3 {
		queries = queries[:3]
	}

	// 2단계: 각 쿼리 병렬 실행
	type stepResult struct {
		Query   string
		Summary string
	}
	results := make([]stepResult, len(queries))
	var wg sync.WaitGroup
	for i, q := range queries {
		wg.Add(1)
		go func(idx int, query string) {
			defer wg.Done()
			summary := ""
			if tKey != "" {
				if tr, ok := tavilySearch(tKey, query, 3); ok {
					summary = tr.Summary
				}
			}
			if summary == "" {
				msgs := []groqMsg{{Role: "user", Content: query}}
				summary, _, _ = callGroqWithFallback(msgs, 400, false)
			}
			results[idx] = stepResult{Query: query, Summary: summary}
		}(i, q)
	}
	wg.Wait()

	// 3단계: 결과 통합
	var parts []string
	for i, r := range results {
		if r.Summary != "" {
			parts = append(parts, fmt.Sprintf("[%d] %s\n%s", i+1, r.Query, r.Summary))
		}
	}
	combined := strings.Join(parts, "\n\n")

	// 4단계: 최종 요약
	var finalPrompt string
	if eng {
		finalPrompt = fmt.Sprintf("Synthesize the following research results into a concise final answer for the goal: '%s'\n\n%s", goal, combined)
	} else {
		finalPrompt = fmt.Sprintf("다음 조사 결과들을 목표 '%s'에 대한 최종 답변으로 통합해주세요:\n\n%s", goal, combined)
	}
	final, _, _ := callGroqWithFallback([]groqMsg{{Role: "user", Content: finalPrompt}}, 600, false)
	if final == "" {
		final = combined
	}
	return final, nil
}
