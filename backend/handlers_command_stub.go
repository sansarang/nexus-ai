//go:build !windows

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type CommandRequest struct {
	Message         string         `json:"message"`
	Context         string         `json:"context"`
	PendingIntent   string         `json:"pending_intent"`
	PendingParams   map[string]any `json:"pending_params"`
	PendingQuestion string         `json:"pending_question"`
}

type CommandResponse struct {
	Success         bool           `json:"success"`
	Message         string         `json:"message"`
	Action          string         `json:"action"`
	Result          any            `json:"result"`
	Duration        string         `json:"duration"`
	NeedsClarify    bool           `json:"needs_clarify,omitempty"`
	ClarifyQuestion string         `json:"clarify_question,omitempty"`
	PendingIntent   string         `json:"pending_intent,omitempty"`
	PendingParams   map[string]any `json:"pending_params,omitempty"`
}

const macSystemPrompt = `당신은 Nexus AI 비서입니다. 사용자 명령을 분석하여 아래 액션 중 하나를 선택하세요.
⚠️ 반드시 JSON만 출력하세요.
형식: {"action":"액션명","params":{...},"message":"사용자에게 보여줄 짧은 답변"}

액션 목록:
"chat" → 일반 대화, 질문, 설명 요청
  params: {}

"web_search" → 쇼핑/최저가/뉴스/맛집/유튜브/틱톡/쿠팡/네이버 검색
  params: {"query":"검색어","site":"coupang|naver|youtube|tiktok|google|auto","max_items":5}

"weather" → 날씨 확인
  params: {"city":"도시명"}

"calendar_today" → 오늘 일정
  params: {}

"calendar_add" → 일정 추가
  params: {"title":"제목","date":"YYYY-MM-DD","time":"HH:MM"}

"persona_switch" → AI 페르소나 변경
  params: {"id":"nexus|research|creative|finance"}

"workflow_plan" → 목표 달성 워크플로우 계획
  params: {"goal":"목표"}

"windows_only" → Windows PC 제어 기능 (볼륨, 보안, 프로세스 등)
  params: {"feature":"기능명"}

"clarify" → 실행에 필수 정보가 없을 때만 사용
  params: {"question":"주인님께 물을 질문(1가지만)","missing":"없는 정보","intent":"원래 액션명","collected":{...지금까지 파악된 파라미터...}}

판단 기준:
- 날씨/기상 → weather (도시 없으면 clarify)
- 일정/캘린더/스케줄 → calendar_today 또는 calendar_add (날짜 없으면 clarify)
- 쇼핑/검색/맛집/뉴스/유튜브/틱톡 → web_search (맛집인데 지역 없으면 clarify)
- PC제어/보안/최적화/볼륨/밝기 → windows_only
- 그 외 모든 대화 → chat

━━━ clarify 사용 기준 (2026년 기준 확장판) ━━━
아래 경우에만 clarify 사용 (나머지는 최선으로 추론해서 즉시 실행)

🔴 필수 Clarify (무조건 물어봐야 하는 경우)
- web_search / browse_page: query가 완전히 없거나 너무 모호할 때
  → "어떤 것을 검색할까요?" 또는 "어떤 키워드로 찾아드릴까요?"
- file_search / recall: 단서(이름, 키워드, 날짜, 발신자)가 전혀 없을 때
  → "어떤 파일을 찾으시나요? 이름이나 키워드, 날짜를 알려주세요"
- weather / 교통 / 일정: 지역이나 날짜가 명확하지 않을 때
  → "어느 지역 날씨를 알려드릴까요?" / "언제 출발하실 예정인가요?"
- scheduler / reminder / 자동 작업: 실행 내용, 시간, 반복 여부가 불완전할 때
  → "언제, 무엇을 자동으로 실행할까요?"
- doc_compare / doc_summary: 비교할 파일 경로나 개수가 불명확할 때
  → "비교하거나 요약할 파일 경로를 알려주세요"
- 상품 검색 (쿠팡, 테무, 네이버쇼핑 등): 브랜드, 모델, 스펙이 불명확할 때
  예) "콜라" → "코카콜라인지 펩시인지요?"
  예) "라면" → "신라면, 너구리, 짜파게티 중 어떤 걸 원하시나요?"
  예) "노트북 추천" → "예산과 용도(업무/게임/학습)를 알려주세요"
- 맛집/장소/예약 검색: 지역이나 종류가 없을 때
  → "어느 지역 맛집을 찾아드릴까요?"

🟠 강력 추천 Clarify (혼란을 크게 줄이는 경우)
- 동일 이름 업체·상품·파일이 여러 개 검색될 때
  → "OO 관련 결과가 여러 개 있습니다. 어느 것을 원하시나요?" (목록 간단히 나열)
- 대명사 / 모호한 참조 ("이거", "그거", "저거", "그 파일", "그 뉴스")
  → "어떤 걸 말씀하시는 건가요? 조금 더 자세히 알려주세요"
- 어휘 중의성 (한 단어가 여러 의미일 때)
  예) "파이썬 알려줘" → "프로그래밍 언어 파이썬인가요, 아니면 뱀 파이썬인가요?"
- 시간 모호성 ("오늘", "이번 주", "지난번")
  → "어느 날짜나 기간을 말씀하시는 건가요?"
- 유사한 의도가 여러 개일 때
  예) "보고서 만들어줘" → "어떤 주제의 보고서를 만드시겠습니까?"
- 클립보드 / 화면 관련 ("이거 번역해", "이 창 정리해")
  → "현재 클립보드 내용인가요, 아니면 화면에 있는 내용인가요?"
- 반복 작업 설정
  → "매일/매주/매월 반복할까요? 아니면 이번 한 번만 할까요?"

🟡 선택적 Clarify (가능하면 추론하고, 그래도 모호하면 물어보기)
- "최신" / "인기" / "추천" 같은 모호한 수식어
- "좋은 거" / "싼 거" / "비싼 거" 같은 주관적 표현
- 숫자/수량 모호성 (예: "커피 3개" → "3잔인가요, 3박스인가요?")

━━━ 철칙 ━━━
- clarify는 꼭 필요한 경우에만 최소 1회 사용
- 대부분의 경우는 최선의 추론으로 바로 실행
- Clarifying Question은 자연스럽고 친절하게, 옵션을 제시하면 더 좋음
- 한 번 clarify 후 사용자가 답하면 컨텍스트를 강하게 유지해서 바로 실행

[검색 결과 처리 — 중요]
동일한 이름의 업체·장소·상품이 여러 개 검색되면 절대 하나만 골라 답하지 말고,
반드시 목록을 보여주고 "어느 것을 원하시나요?" 라고 되물을 것.`

const macClarifyResolvePrompt = `당신은 Nexus AI 비서입니다. 사용자가 추가 정보를 제공했습니다.
이전 컨텍스트와 새 정보를 합쳐서 완전한 액션을 결정하세요.
⚠️ 반드시 JSON만 출력하세요.
형식: {"action":"액션명","params":{...완전한 파라미터...},"message":"짧은 답변"}

이전 액션: %s
이전 파라미터: %s
이전 질문: %s
사용자 새 답변: %s`

func handleCommand(w http.ResponseWriter, r *http.Request) {
	var req CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Message == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "message 필요"})
		return
	}

	start := time.Now()

	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()

	if gKey == "" {
		writeJSON(w, 400, map[string]any{
			"success": false,
			"message": "Groq API 키가 설정되지 않았습니다. 설정에서 API 키를 입력해주세요.",
		})
		return
	}

	// ── 멀티턴: 이전 clarify 컨텍스트가 있으면 해소 프롬프트 사용 ──
	var intentPrompt string
	if req.PendingIntent != "" {
		prevParamsJSON, _ := json.Marshal(req.PendingParams)
		intentPrompt = fmt.Sprintf(macClarifyResolvePrompt,
			req.PendingIntent,
			string(prevParamsJSON),
			req.PendingQuestion,
			req.Message,
		)
	} else {
		intentPrompt = req.Message
	}

	// LLM으로 의도 파악
	sysPrompt := macSystemPrompt
	msgs := []groqMsg{
		{Role: "system", Content: sysPrompt},
		{Role: "user", Content: intentPrompt},
	}
	raw, _, err := callGroq(gKey, groqFastModel, msgs, 500, true)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": "LLM 오류: " + err.Error()})
		return
	}

	var intent struct {
		Action  string         `json:"action"`
		Params  map[string]any `json:"params"`
		Message string         `json:"message"`
	}
	if err := json.Unmarshal([]byte(raw), &intent); err != nil {
		// JSON 파싱 실패 → chat으로 처리
		intent.Action = "chat"
		intent.Message = raw
	}

	dur := fmt.Sprintf("%.2fs", time.Since(start).Seconds())

	switch intent.Action {
	case "clarify":
		// 추가 정보 필요 — 프론트엔드에 질문 반환
		var question, missing, pendingIntent string
		var collected map[string]any
		if intent.Params != nil {
			question, _ = intent.Params["question"].(string)
			missing, _ = intent.Params["missing"].(string)
			pendingIntent, _ = intent.Params["intent"].(string)
			collected, _ = intent.Params["collected"].(map[string]any)
		}
		if question == "" {
			question = "조금 더 알려주시면 도움이 될 것 같아요. 어떻게 도와드릴까요?"
		}
		_ = missing
		json200(w, CommandResponse{
			Success:         true,
			Message:         question,
			Action:          "clarify",
			NeedsClarify:    true,
			ClarifyQuestion: question,
			PendingIntent:   pendingIntent,
			PendingParams:   collected,
			Duration:        dur,
		})

	case "chat":
		baseSys := getPersonaSystemPrompt()
		sysPrompt := baseSys + `

[Instructions - highest priority]
1. NEVER hallucinate real-time data (weather, stock prices, news, schedules)
2. If you don't know → say "정확한 정보를 알 수 없습니다"
3. Answer in natural Korean, 2~4 sentences max
4. No markdown headers, no excessive bullet points

[Example]
Q: "오늘 날씨 어때?" → A: "실시간 날씨 정보는 날씨 기능을 이용해 주세요."
Q: "파이썬이 뭐야?" → A: "파이썬은 읽기 쉬운 문법의 프로그래밍 언어로, 데이터 분석과 웹 개발에 많이 쓰입니다."`
		chatMsgs := []groqMsg{
			{Role: "system", Content: sysPrompt},
			{Role: "user", Content: req.Message},
		}
		answer, _, err := callGroq(gKey, groqChatModel, chatMsgs, 512, false)
		if err != nil {
			answer = "죄송합니다, 답변을 생성하는 중 오류가 발생했습니다."
		}
		json200(w, CommandResponse{
			Success:  true,
			Message:  answer,
			Action:   "chat",
			Duration: dur,
		})

	case "weather":
		city := "서울"
		if c, ok := intent.Params["city"].(string); ok && c != "" {
			city = c
		}
		// wttr.in 실시간 날씨 API 호출
		wText := fetchWeatherText(city, gKey)
		json200(w, CommandResponse{
			Success:  true,
			Message:  wText,
			Action:   "weather",
			Result:   map[string]any{"city": city},
			Duration: dur,
		})

	case "calendar_today":
		today := time.Now().Format("2006-01-02")
		evs := loadEvents()
		var todayEvs []CalEvent
		for _, e := range evs {
			if e.Date == today {
				todayEvs = append(todayEvs, e)
			}
		}
		msg := fmt.Sprintf("오늘(%s) 일정이 %d개 있습니다.", today, len(todayEvs))
		if len(todayEvs) == 0 {
			msg = "오늘 등록된 일정이 없습니다."
		}
		json200(w, CommandResponse{
			Success:  true,
			Message:  msg,
			Action:   "calendar_today",
			Result:   map[string]any{"events": todayEvs},
			Duration: dur,
		})

	case "calendar_add":
		var title, date, t string
		if intent.Params != nil {
			title, _ = intent.Params["title"].(string)
			date, _ = intent.Params["date"].(string)
			t, _ = intent.Params["time"].(string)
		}
		if title == "" {
			title = req.Message
		}
		if date == "" {
			date = time.Now().Format("2006-01-02")
		}
		ev := CalEvent{
			ID: fmt.Sprintf("%d", time.Now().UnixMilli()),
			Title: title, Date: date, Time: t,
		}
		evs := loadEvents()
		evs = append(evs, ev)
		saveEvents(evs)
		json200(w, CommandResponse{
			Success:  true,
			Message:  fmt.Sprintf("✅ 일정 추가됨: %s (%s)", title, date),
			Action:   "calendar_add",
			Result:   map[string]any{"event": ev},
			Duration: dur,
		})

	case "web_search":
		var query, site string
		maxItems := 5
		if intent.Params != nil {
			query, _ = intent.Params["query"].(string)
			site, _ = intent.Params["site"].(string)
			if v, ok := intent.Params["max_items"].(float64); ok {
				maxItems = int(v)
			}
		}
		if query == "" {
			query = req.Message
		}
		result := runWebSearchMac(gKey, query, site, maxItems)
		json200(w, CommandResponse{
			Success:  true,
			Message:  result.Summary,
			Action:   "web_search",
			Result:   result,
			Duration: dur,
		})

	case "persona_switch":
		var id string
		if intent.Params != nil {
			id, _ = intent.Params["id"].(string)
		}
		for _, p := range builtinPersonas {
			if p.ID == id {
				personaMu.Lock()
				activePersonaID = id
				personaMu.Unlock()
				savePersonaConfig()
				json200(w, CommandResponse{
					Success:  true,
					Message:  p.Emoji + " " + p.Name + " 페르소나로 전환했습니다.",
					Action:   "persona_switch",
					Duration: dur,
				})
				return
			}
		}
		json200(w, CommandResponse{Success: false, Message: "알 수 없는 페르소나입니다.", Action: "persona_switch"})

	case "workflow_plan":
		var goal string
		if intent.Params != nil {
			goal, _ = intent.Params["goal"].(string)
		}
		if goal == "" {
			goal = req.Message
		}
		wMsgs := []groqMsg{
			{Role: "system", Content: "당신은 효율적인 워크플로우 설계 전문가입니다."},
			{Role: "user", Content: "다음 목표를 달성하기 위한 구체적인 단계별 계획을 작성해줘: " + goal},
		}
		plan, _, _ := callGroq(gKey, groqChatModel, wMsgs, 800, false)
		json200(w, CommandResponse{
			Success:  true,
			Message:  plan,
			Action:   "workflow_plan",
			Duration: dur,
		})

	case "windows_only":
		var feature string
		if intent.Params != nil {
			feature, _ = intent.Params["feature"].(string)
		}
		msg := "이 기능은 Windows PC에서만 사용 가능합니다."
		if feature != "" {
			msg = fmt.Sprintf("'%s' 기능은 Windows PC에서만 사용 가능합니다.", feature)
		}
		json200(w, CommandResponse{
			Success:  false,
			Message:  msg,
			Action:   "windows_only",
			Duration: dur,
		})

	default:
		// 알 수 없는 액션 → chat으로 폴백
		chatMsgs := []groqMsg{
			{Role: "system", Content: getPersonaSystemPrompt()},
			{Role: "user", Content: req.Message},
		}
		answer, _, _ := callGroq(gKey, groqChatModel, chatMsgs, 1024, false)
		json200(w, CommandResponse{
			Success:  true,
			Message:  answer,
			Action:   "chat",
			Duration: dur,
		})
	}
}

// ── 웹 검색 (Groq 기반 + 브라우저 에이전트) ───────────────────

type webSearchResult struct {
	Query   string         `json:"query"`
	Site    string         `json:"site"`
	Summary string         `json:"summary"`
	Items   []map[string]string `json:"items,omitempty"`
}

func runWebSearchMac(apiKey, query, site string, maxItems int) webSearchResult {
	siteLabel := site
	if siteLabel == "" || siteLabel == "auto" {
		siteLabel = "웹"
	}

	// 병렬 검색: Tavily + 브라우저 동시 실행
	result := parallelWebSearch(query, maxItems)

	// 결과가 있으면 그대로 반환
	if result.Summary != "" || len(result.Items) > 0 {
		return webSearchResult{
			Query:   query,
			Site:    siteLabel,
			Summary: result.Summary,
			Items:   result.Items,
		}
	}

	// 최후 폴백: Groq LLM (실시간 데이터 없음)
	today := time.Now().Format("2006-01-02")
	prompt := fmt.Sprintf(`오늘은 %s입니다.
사용자 질문: "%s"

[지시사항]
- URL, 링크, 출처명 절대 포함 금지
- 사용자 질문에 직접 답하는 자연스러운 한국어 2~4문장으로 핵심만 답변
- 실시간 데이터가 없으면 "정확한 최신 정보는 미리보기 버튼으로 확인해보세요" 안내
- 친절한 AI 비서처럼 작성`, today, query)
	msgs := []groqMsg{{Role: "user", Content: prompt}}
	text, _, err := callGroq(apiKey, groqChatModel, msgs, 512, false)
	if err != nil {
		text = "검색 중 오류가 발생했습니다: " + err.Error()
	}

	// items가 없어도 검색 엔진 URL을 항상 제공해서 미리보기 가능하게
	fallbackItems := buildFallbackURLs(query, site)

	return webSearchResult{
		Query:   query,
		Site:    siteLabel,
		Summary: text,
		Items:   fallbackItems,
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

