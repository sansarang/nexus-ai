//go:build !windows

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)


func handleCommand(w http.ResponseWriter, r *http.Request) {
	if !requireAuth(w, r) {
		return
	}
	var req CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Message == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "message required / message 필요"})
		return
	}

	// ── 사용자 식별 (이메일 우선, 없으면 IP) ────────────────────
	userID := req.UserEmail
	if userID == "" {
		userID = r.RemoteAddr
	}

	// ── 응답 캐시 hit 체크 (Mac dev) ───────────────────────
	if req.PendingIntent == "" {
		if cached, ok := cmdCacheGet(req.Message, req.Lang); ok {
			cached.Duration = "0.00s (cached)"
			json200(w, cached)
			return
		}
	}

	start := time.Now()

	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()

	if gKey == "" {
		apiKeyMsg := "Groq API 키가 설정되지 않았습니다. 설정에서 API 키를 입력해주세요."
		if req.Lang == "en" || isEnglishQuery(req.Message) {
			apiKeyMsg = "Groq API key is not configured. Please enter your API key in settings."
		}
		writeJSON(w, 400, map[string]any{
			"success": false,
			"message": apiKeyMsg,
		})
		return
	}

	// ── 멀티턴: 이전 clarify 컨텍스트가 있으면 해소 프롬프트 사용 ──
	var intentPrompt string
	if req.PendingIntent != "" {
		prevParamsJSON, _ := json.Marshal(req.PendingParams)
		clarifyPrompt := macClarifyResolvePrompt
		if req.Lang == "en" || isEnglishQuery(req.Message) {
			clarifyPrompt = "You are Nexus AI assistant. The user has provided additional information. Combine previous context with new info to determine the complete action.\n⚠️ Output JSON ONLY. Format: {\"action\":\"action_name\",\"params\":{...complete params...},\"message\":\"short English response\"}\n\nPrevious action: %s\nPrevious params: %s\nPrevious question: %s\nUser's new answer: %s"
		}
		intentPrompt = fmt.Sprintf(clarifyPrompt,
			req.PendingIntent,
			string(prevParamsJSON),
			req.PendingQuestion,
			req.Message,
		)
	} else {
		intentPrompt = req.Message
	}

	// ── 세션 메모리: 대명사 해소 ────────────────────────────────
	intentPrompt = resolvePronouns(intentPrompt, userID)

	// ── 키워드 사전 라우팅 (LLM보다 우선, 틱톡/유튜브 영상 검색) ──
	msgLower := strings.ToLower(req.Message)
	videoVerbs := []string{"찾", "검색", "영상", "비디오", "보여", "추천", "viral", "바이럴", "트렌드", "노래", "음악", "music", "song", "플레이리스트", "playlist", "뮤직비디오", "mv", "들려"}
	isTikTokReq := strings.Contains(msgLower, "틱톡") || strings.Contains(msgLower, "tiktok")
	isYouTubeReq := strings.Contains(msgLower, "유튜브") || strings.Contains(msgLower, "youtube")
	hasVideoVerb := false
	for _, kw := range videoVerbs {
		if strings.Contains(msgLower, kw) {
			hasVideoVerb = true
			break
		}
	}

	var preRoutedAction string
	var preRoutedParams map[string]any
	systemPreRouted := false

	// ── ★ 최최우선 1: 위험 액션 감지 (Phase A2) — 즉시 확인 카드 ──
	if dangerKey := detectDangerousInMessage(req.Message); dangerKey != "" {
		preRoutedAction = dangerKey
		preRoutedParams = map[string]any{"confirmed": false, "_dangerous": true}
		systemPreRouted = true
	}

	// ── ★ 최최우선 2: 멀티스텝 감지 ──
	if !systemPreRouted && detectMultiStep(req.Message) {
		preRoutedAction = "workflow_run"
		preRoutedParams = map[string]any{"goal": req.Message}
		systemPreRouted = true
	}

	// ── ★ 최최우선 3: Vision 키워드 ──
	if !systemPreRouted {
		visionPat := []string{"화면 분석", "스크린샷 분석", "screen analyze", "analyze screen", "vision", "이 이미지", "이 화면", "보이는 거", "이 사진"}
		for _, kw := range visionPat {
			if strings.Contains(msgLower, strings.ToLower(kw)) {
				preRoutedAction = "vision"
				preRoutedParams = map[string]any{"question": req.Message}
				systemPreRouted = true
				break
			}
		}
	}

	// ── 최우선 명시 패턴 (Go map 순서 비결정성 회피) ──
	// 1) Excel 분석 (사용자 파일 활용) — 멀티스텝이 우선 매칭됐으면 스킵
	if !systemPreRouted {
		excelAnalyzePat := regexp.MustCompile(`(엑셀|excel|xlsx).*(분석|요약|보여|이해|읽어|확인|analyze|summarize|read)`)
		if excelAnalyzePat.MatchString(msgLower) {
			preRoutedAction = "excel_analyze"
			preRoutedParams = map[string]any{}
			systemPreRouted = true
		}
	}
	// 2) Excel 자동 생성
	if !systemPreRouted {
		excelPat := regexp.MustCompile(`(엑셀|excel|스프레드시트|spreadsheet)`)
		excelVerb := regexp.MustCompile(`(만들|생성|정리|create|make|generate)`)
		if excelPat.MatchString(msgLower) && excelVerb.MatchString(msgLower) {
			preRoutedAction = "excel_auto_create"
			preRoutedParams = map[string]any{"topic": extractExcelTopicMac(req.Message)}
			systemPreRouted = true
		}
	}
	// 3) PDF 자동 생성 (문서보다 우선)
	if !systemPreRouted {
		pdfPat := regexp.MustCompile(`(pdf|피디에프)`)
		pdfVerb := regexp.MustCompile(`(만들|생성|작성|create|make|generate|write|draft|export)`)
		if pdfPat.MatchString(msgLower) && pdfVerb.MatchString(msgLower) {
			preRoutedAction = "pdf_auto_create"
			preRoutedParams = map[string]any{}
			systemPreRouted = true
		}
	}

	// 4) 영상 워크플로 (URL 명시 + 영상 동작 키워드)
	if !systemPreRouted {
		videoUrlRe := regexp.MustCompile(`https?://[^\s]*(youtube|youtu\.be|vimeo|tiktok|twitch)[^\s]*`)
		videoVerbRe := regexp.MustCompile(`(다운로드|받아|save|자막|subtitle|transcript|스크립트|요약|summary|summarize)`)
		if videoUrlRe.MatchString(msgLower) && videoVerbRe.MatchString(msgLower) {
			preRoutedAction = "video_workflow"
			preRoutedParams = map[string]any{}
			systemPreRouted = true
		}
	}

	// 5) 문서 자동 생성 (보고서/메모/회의록)
	if !systemPreRouted {
		docPat := regexp.MustCompile(`(보고서|메모|회의록|기획서|문서|노트|report|memo|note|proposal|document)`)
		docVerb := regexp.MustCompile(`(작성|만들|생성|써|쓰|create|make|write|generate|draft)`)
		if docPat.MatchString(msgLower) && docVerb.MatchString(msgLower) {
			preRoutedAction = "doc_auto_create"
			preRoutedParams = map[string]any{}
			systemPreRouted = true
		}
	}

	// ── 시스템 인텐트 사전 라우팅 (LLM이 chat/clarify 로 잘못 떨어뜨리는 케이스 방지) ──
	// action 이름은 backend switch + frontend renderCommandResult 가 처리하는 표준 이름 사용
	// KO/EN 모두 동등 대응
	systemPatterns := map[string]string{
		"excel_auto_create": `(엑셀.*(만들|생성|정리)|스프레드시트.*(만들|생성)|표로.*정리|excel.*(create|make|generate)|create.*excel|generate.*spreadsheet|make.*table)`,
		"stats":         `(메모리|램|ram|cpu|디스크|하드|저장공간|pc\s*상태|내\s*pc|내\s*컴퓨터|시스템\s*상태|memory|disk\s*(space|usage)|hard\s*drive|storage|system\s*status|pc\s*status|free\s*space)`,
		"scan":          `(보안.*스캔|바이러스.*검사|악성코드|해킹.*확인|보안.*점검|느려|버벅|렉|왜.*이래|성능.*문제|컴퓨터.*문제|security\s*scan|virus\s*(scan|check)|malware|antivirus|slow|sluggish|laggy|why.*slow)`,
		"clean":         `(정리.*해|청소.*해|캐시.*비워|임시파일.*정리|공간.*확보|디스크.*정리|clean\s*(up|cache)|clear\s*cache|temp\s*files|free\s*up\s*space|disk\s*cleanup)`,
		"email_inbox":   `(받은\s*메일|받은편지|이메일\s*확인|inbox|메일\s*보여|check\s*(email|mail|inbox)|show.*(email|mail)|read.*mail)`,
		"price_compare": `(최저가|가격\s*비교|얼마야|얼마예요|할인|특가|가격대|싸게.*살|어디서.*싸|가성비|cheapest|lowest\s*price|price\s*compare|how\s*much.*cost|best\s*deal|where.*buy.*cheap)`,
		"video_search":  `(뮤직비디오|뮤비|mv\b|플레이리스트|playlist|커버\s*영상|라이브\s*영상|숏폼|쇼츠|reels|music\s*video|cover\s*song|live\s*performance|shorts)`,
		// 44개 추가
		"news_search":        `(뉴스|속보|이슈|오늘.*소식|news|breaking|headlines)`,
		"youtube_search":     `(유튜브|youtube|영상.*검색|동영상.*찾)`,
		"translate":          `(번역|영어로|한국어로|일본어로|중국어로|translate|translation)`,
		"calendar_today":     `(오늘.*일정|오늘.*스케줄|오늘.*캘린더|today.*schedule|today.*calendar)`,
		"calendar_week":      `(이번\s*주.*일정|주간.*일정|week.*schedule)`,
		"calendar_find_slot": `(빈.*시간|언제.*가능|미팅.*잡|회의.*잡|free\s*time|find.*slot)`,
		"email_classify":     `(이메일.*분류|메일.*분류|classify.*email)`,
		"email_draft":        `(이메일.*작성|답장.*써|메일.*초안|draft.*email|reply.*draft)`,
		"email_summarize":    `(이메일.*요약|메일.*요약|email.*summary)`,
		"imap_inbox":         `(imap|gmail.*받은|outlook.*받은|메일.*동기화)`,
		"meeting_list":       `(회의.*목록|미팅.*리스트|meeting.*list)`,
		"meeting_summary":    `(회의.*요약|미팅.*요약|meeting.*summary)`,
		"notes":              `(노트.*목록|메모.*목록|note.*list|show.*notes)`,
		"voice_todo":         `(음성.*할일|받아.*적|받아.*써|voice.*todo|dictate)`,
		"recall_capture":     `(기억해|저장해.*화면|capture.*recall|remember.*this)`,
		"recall_search":      `(기억.*검색|예전.*화면|search.*recall|find.*past|예전.*작성|예전.*문서|최근.*메모|어디 있|어디있)`,
		"brain_search":       `(브레인.*검색|지식.*검색|brain.*search|knowledge.*search|문서.*검색|파일.*검색|박부장|손해배상)`,
		"file_search":        `(파일.*찾|문서.*찾|계약서.*찾|보고서.*찾|file.*find|find.*document|search.*file)`,
		"brain_stats":        `(브레인.*상태|지식.*통계|brain.*stats|knowledge.*stats)`,
		"persona_list":       `(페르소나.*목록|성격.*목록|persona.*list)`,
		"workflow_list":      `(워크플로.*목록|자동화.*목록|workflow.*list)`,
		"workflow_templates": `(워크플로.*템플릿|자동화.*템플릿|workflow.*template)`,
		"schedule_list":      `(예약.*목록|스케줄.*목록|크론.*목록|schedule.*list|cron.*list)`,
		"clipboard_history":  `(클립보드.*기록|클립보드.*히스토리|clipboard.*history|paste.*history)`,
		"clipboard_ai":       `(클립보드.*ai|복사.*ai|smart.*clipboard)`,
		"dictation_start":    `(받아쓰기.*시작|딕테이션.*시작|start.*dictation)`,
		"file_duplicates":    `(중복.*파일|똑같은.*파일|duplicate.*files)`,
		"search_pdf":         `(pdf.*검색|pdf.*찾|search.*pdf|find.*pdf)`,
		"network_analysis":   `(네트워크.*분석|와이파이.*분석|인터넷.*상태|network.*analysis|wifi.*analysis)`,
		"perf_history":       `(성능.*기록|성능.*히스토리|performance.*history)`,
		"perf_anomaly":       `(성능.*이상|이상.*탐지|perf.*anomaly)`,
		"gpu_stats":          `(gpu|그래픽카드|video.*card)`,
		"process_top":        `(프로세스.*top|process.*top|top.*process|cpu.*점유)`,
		"process_security":   `(프로세스.*보안|의심.*프로세스|process.*security|suspicious.*process)`,
		"startup_items":      `(시작\s*프로그램|자동.*실행|startup.*items|startup.*programs)`,
		"programs_list":      `(설치.*프로그램.*목록|programs\s*list|installed.*apps)`,
		"app_permissions":    `(앱.*권한|권한.*확인|app.*permissions)`,
		"boot_analysis":      `(부팅.*분석|시작.*시간|boot.*analysis|boot.*time)`,
		"driver_check":       `(드라이버.*확인|드라이버.*검사|driver.*check|driver.*update)`,
		"defender_status":    `(디펜더|defender|windows.*security|보안.*상태)`,
		"virus_check":        `(바이러스.*확인|virus.*check|malware.*scan)`,
		"windows_updates":    `(윈도우.*업데이트|windows.*update|update.*check)`,
		"remote_access":      `(원격.*접속|remote.*access|teamviewer|anydesk)`,
		// ★ 직업 페르소나 액션 — Phase B 매칭률 향상
		"contract_review":    `(계약서.*검토|계약.*검토|nda.*검토|약관.*검토|독소조항|contract\s*review|review\s*contract)`,
		"stock_analysis":     `(주가.*분석|종목.*분석|주식.*분석|코인.*분석|etf.*분석|.*전망|portfolio.*analysis|stock\s*analysis)`,
		"medical_search":     `(약물.*상호작용|약물.*부작용|임상.*가이드|논문.*요약|pubmed.*검색)`,
		"legal_search":       `(판례.*검색|법령.*검색|규정.*검색)`,
	}
	// 매칭된 액션별로 파라미터 추출
	// systemPreRouted 는 이미 위에서 선언됨 (excel 우선 체크 시 true 가능)
	// 이미 set 되어 있으면 systemPatterns 매칭 스킵
	for action, pat := range systemPatterns {
		if systemPreRouted {
			break
		}
		if matched, _ := regexp.MatchString(pat, msgLower); matched {
			preRoutedAction = action
			preRoutedParams = map[string]any{}
			if action == "price_compare" {
				// 최저가/할인 등 키워드 제거하고 검색어로 사용
				q := req.Message
				for _, rm := range []string{"최저가", "가격 비교", "가격비교", "얼마야", "얼마예요", "할인", "특가", "가격대", "추천", "찾아줘", "검색"} {
					q = strings.ReplaceAll(q, rm, "")
				}
				preRoutedParams["query"] = strings.TrimSpace(q)
				preRoutedParams["site"] = ""
				preRoutedParams["max_items"] = 8
			} else if action == "video_search" {
				q := req.Message
				for _, rm := range []string{"뮤직비디오", "뮤비", "플레이리스트", "playlist", "커버 영상", "라이브 영상", "숏폼", "쇼츠", "reels", "찾아줘", "검색", "보여줘", "추천"} {
					q = strings.ReplaceAll(q, rm, "")
				}
				preRoutedParams["query"] = strings.TrimSpace(q)
				preRoutedParams["platform"] = "youtube"
				preRoutedParams["max_items"] = 8
			}
			systemPreRouted = true
			break
		}
	}

	// 날씨는 도시 파싱이 필요해서 별도 처리 (systemPatterns 매칭 후 도시 추출)
	if !systemPreRouted {
		cityPat := regexp.MustCompile(`(서울|부산|인천|대구|광주|대전|울산|수원|제주|뉴욕|도쿄|상하이|싱가포르|런던|파리|베를린|로마|시드니|방콕|홍콩|타이베이|상파울루|모스크바)`)
		weatherPat := regexp.MustCompile(`(날씨|기온|온도|비\s*와|눈\s*와|미세먼지|weather)`)
		if cityM := cityPat.FindString(req.Message); cityM != "" && weatherPat.MatchString(req.Message) {
			preRoutedAction = "weather"
			preRoutedParams = map[string]any{"city": cityM}
			systemPreRouted = true
		}
	}

	// 가격/쇼핑/도메인 사전 라우팅
	shoppingSites := map[string]string{
		// ── 쇼핑몰 ──────────────────────────────────────
		"태무": "temu.com", "테무": "temu.com", "temu": "temu.com",
		"쿠팡": "coupang.com", "coupang": "coupang.com",
		"네이버쇼핑": "shopping.naver.com", "네이버 쇼핑": "shopping.naver.com",
		"11번가": "11st.co.kr",
		"지마켓": "gmarket.co.kr", "gmarket": "gmarket.co.kr",
		"옥션": "auction.co.kr", "auction": "auction.co.kr",
		"위메프": "wemakeprice.com",
		"티몬": "tmon.co.kr",
		"알리": "aliexpress.com", "aliexpress": "aliexpress.com", "알리익스프레스": "aliexpress.com",
		"아마존": "amazon.com", "amazon": "amazon.com",
		"무신사": "musinsa.com",
		"에이블리": "a-bly.com",
		"지그재그": "zigzag.kr",
		"브랜디": "brandi.co.kr",
		"오늘의집": "ohou.se",
		"이케아": "ikea.com/kr", "ikea": "ikea.com/kr",
		// ── 중고차 ──────────────────────────────────────
		"헤이딜러": "heydealer.com", "heydealer": "heydealer.com",
		"엔카": "encar.com", "encar": "encar.com",
		"kb차차차": "kbchachacha.com", "차차차": "kbchachacha.com",
		"sk엔카": "encar.com",
		"오토피디아": "autopedia.co.kr",
		"보배드림": "bobaedream.co.kr",
		"중고차": "encar.com",
		// ── 중고거래 ────────────────────────────────────
		"당근": "daangn.com", "당근마켓": "daangn.com", "daangn": "daangn.com",
		"번개장터": "bunjang.co.kr", "번개": "bunjang.co.kr",
		"중고나라": "joongna.com",
		"헬로마켓": "hellomarket.com",
		// ── 부동산 ──────────────────────────────────────
		"직방": "zigbang.com", "zigbang": "zigbang.com",
		"다방": "dabangapp.com",
		"호갱노노": "hogangnono.com",
		"네이버부동산": "land.naver.com", "네이버 부동산": "land.naver.com",
		"부동산114": "r114.com",
		// ── 음식/배달 ────────────────────────────────────
		"배민": "baemin.com", "배달의민족": "baemin.com",
		"요기요": "yogiyo.co.kr",
		"쿠팡이츠": "coupangeats.com",
		// ── 여행/숙박 ────────────────────────────────────
		"야놀자": "yanolja.com",
		"여기어때": "goodchoice.kr",
		"에어비앤비": "airbnb.co.kr", "airbnb": "airbnb.com",
		"호텔스닷컴": "hotels.com",
		"익스피디아": "expedia.co.kr",
		// ── 전자기기 ─────────────────────────────────────
		"다나와": "danawa.com",
		"에누리": "enuri.com",
		"컴퓨존": "compuzone.co.kr",
		"아이셋톱": "isettop.com",
	}
	detectedShopSite := ""
	for keyword, domain := range shoppingSites {
		if strings.Contains(msgLower, strings.ToLower(keyword)) {
			detectedShopSite = domain
			break
		}
	}

	outFmt := detectOutputFormat(req.Message)
	// 포맷 키워드 OR 저장 동사 중 하나라도 있으면 파일 저장 트리거
	isMultiAction := outFmt != outNone && req.PendingIntent == ""

	// ── Pre-routing: 액션 감지만 (Clarify 판단은 Groq에 위임) ──────
	// 시스템 패턴이 이미 잡혔으면 쇼핑/영상 사전 라우팅 스킵
	if preRoutedAction != "" {
		// 이미 시스템 액션 (pc_status 등) 결정됨
	} else if detectedShopSite != "" && req.PendingIntent == "" {
		q := req.Message
		for kw := range shoppingSites {
			q = strings.ReplaceAll(q, kw, "")
		}
		for _, rm := range []string{"에서", "찾아줘", "검색해줘", "최저가", "가격", "얼마야", "구매", "사고 싶어", "추천", "알려줘"} {
			q = strings.ReplaceAll(q, rm, "")
		}
		q = strings.TrimSpace(q) // 비어있으면 "" 그대로 유지 — Groq이 "없음"으로 판단하도록
		if isMultiAction {
			preRoutedAction = "multi_action"
			preRoutedParams = map[string]any{"sub_action": "price_compare", "query": q, "site": detectedShopSite, "max_items": 8, "format": string(outFmt)}
		} else {
			preRoutedAction = "price_compare"
			preRoutedParams = map[string]any{"query": q, "site": detectedShopSite, "max_items": 8}
		}
	} else if isTikTokReq && hasVideoVerb && req.PendingIntent == "" {
		q := req.Message
		for _, rm := range []string{"틱톡에서", "틱톡", "tiktok", "찾아줘", "검색해줘", "보여줘", "영상", "추천해줘"} {
			q = strings.ReplaceAll(q, rm, "")
		}
		q = strings.TrimSpace(q)
		if isMultiAction {
			preRoutedAction = "multi_action"
			preRoutedParams = map[string]any{"sub_action": "video_search", "query": q, "platform": "tiktok", "max_items": 8, "format": string(outFmt)}
		} else {
			preRoutedAction = "video_search"
			preRoutedParams = map[string]any{"query": q, "platform": "tiktok", "max_items": 8}
		}
	} else if isYouTubeReq && hasVideoVerb && req.PendingIntent == "" {
		q := req.Message
		for _, rm := range []string{"유튜브에서", "유튜브", "youtube", "찾아줘", "검색해줘", "보여줘", "영상", "추천해줘"} {
			q = strings.ReplaceAll(q, rm, "")
		}
		q = strings.TrimSpace(q)
		if isMultiAction {
			preRoutedAction = "multi_action"
			preRoutedParams = map[string]any{"sub_action": "video_search", "query": q, "platform": "youtube", "max_items": 8, "format": string(outFmt)}
		} else {
			preRoutedAction = "video_search"
			preRoutedParams = map[string]any{"query": q, "platform": "youtube", "max_items": 8}
		}
	} else if isMultiAction && req.PendingIntent == "" {
		lower := strings.ToLower(req.Message)
		subAction := "summarize"
		for _, v := range []string{"비교해줘", "비교해", "비교 정리", "비교표", " vs ", "vs.", "대비"} {
			if strings.Contains(lower, v) {
				subAction = "doc_compare"
				break
			}
		}
		preRoutedAction = "multi_action"
		preRoutedParams = map[string]any{"sub_action": subAction, "query": req.Message, "format": string(outFmt), "max_items": 8}
	}

	// ── 채팅 페르소나 전환 감지 ────────────────────────────────────
	if preRoutedAction == "" && req.PendingIntent == "" {
		personaSwitchMap := map[string]string{
			"개발자": "developer", "개발자모드": "developer", "개발 모드": "developer", "it 모드": "developer", "코딩 모드": "developer", "엔지니어": "developer",
			"마케터": "marketer", "마케팅 모드": "marketer", "마케팅모드": "marketer", "디지털 마케터": "marketer", "마케팅 전문가": "marketer",
			"영업": "sales", "세일즈": "sales", "영업 모드": "sales", "세일즈 모드": "sales", "영업 전문가": "sales",
			"pm": "pm", "기획자": "pm", "pm 모드": "pm", "기획 모드": "pm", "프로덕트": "pm", "기획 전문가": "pm",
			"디자이너": "designer", "크리에이터": "designer", "디자인 모드": "designer", "크리에이티브 모드": "designer",
			"투자자": "investor", "투자 전문가": "investor", "트레이더": "investor", "투자 모드": "investor", "재테크": "investor",
			"의사": "medical", "의료진": "medical", "의료 전문가": "medical", "의학 전문가": "medical",
			"변호사": "legal", "법무": "legal", "법률 전문가": "legal", "법 전문가": "legal",
			"유튜버": "creator", "인플루언서": "creator", "크리에이터 모드": "creator", "콘텐츠 전문가": "creator",
			"프리랜서": "freelancer", "1인 사업자": "freelancer", "프리랜서 모드": "freelancer", "사업자 모드": "freelancer",
			"기본": "developer", "기본 모드": "developer",
		}
		switchTriggers := []string{"모드로 바꿔", "모드 바꿔", "모드로 전환", "모드 전환", "페르소나", "으로 바꿔", "로 바꿔줘", "로 전환해", "로 행동해", "행동해줘", "전문가로", "로 바꿔줘", "역할 바꿔"}
		hasTrigger := false
		for _, t := range switchTriggers {
			if strings.Contains(msgLower, t) {
				hasTrigger = true
				break
			}
		}
		if hasTrigger {
			for keyword, pid := range personaSwitchMap {
				if strings.Contains(msgLower, keyword) {
					for _, p := range builtinPersonas {
						if p.ID == pid {
							personaMu.Lock()
							activePersonaID = pid
							personaMu.Unlock()
							savePersonaConfig()
							json200(w, CommandResponse{
								Success:  true,
								Message:  p.Emoji + " " + p.Name + " 모드로 전환했습니다. 이제 " + p.Description + " 관점으로 답변합니다.",
								Action:   "persona_switch",
								Duration: fmt.Sprintf("%.2fs", time.Since(start).Seconds()),
							})
							return
						}
					}
				}
			}
		}
	}

	// ── 이메일 전송 pre-routing ───────────────────────────────────
	if preRoutedAction == "" && req.PendingIntent == "" {
		emailTriggers := []string{"이메일 보내", "메일 보내", "이메일 발송", "메일 발송", "이메일 써줘", "메일 써줘",
			"이메일 작성", "메일 작성", "send email", "이메일 전송"}
		for _, kw := range emailTriggers {
			if strings.Contains(msgLower, kw) {
				preRoutedAction = "email"
				preRoutedParams = map[string]any{"message": req.Message}
				break
			}
		}
	}

	// 출장/여행 준비 pre-routing (액션 감지만)
	if preRoutedAction == "" && req.PendingIntent == "" {
		for _, v := range []string{"출장 준비", "여행 준비", "출장 계획", "여행 계획", "출장 가", "출장이야", "출장인데", "출장 있", "여행 있", "trip 준비"} {
			if strings.Contains(msgLower, v) {
				preRoutedAction = "trip_plan"
				preRoutedParams = map[string]any{"destination": req.Message, "purpose": "출장"}
				break
			}
		}
	}

	// ── 직업군 워크플로우 프리셋 감지 ─────────────────────────────
	if preRoutedAction == "" && req.PendingIntent == "" {
		pid := getActivePersona().ID
		type presetDef struct {
			triggers []string
			preset   string
		}
		presetMap := map[string][]presetDef{
			"developer": {
				{[]string{"코드 리뷰", "pr 리뷰", "pull request"}, "dev_code_review"},
				{[]string{"버그 해결", "에러 어떻게", "버그 고쳐", "오류 고쳐", "이 에러"}, "dev_bug_fix"},
				{[]string{"리팩토링", "리팩터링", "refactor", "코드 개선"}, "dev_refactor"},
				{[]string{"github 이슈", "깃허브 이슈", "이슈 찾아", "pr 찾아"}, "dev_github_search"},
				{[]string{"터미널 명령", "명령어 최적화", "커맨드 최적화"}, "dev_terminal_command"},
				{[]string{"api 설계", "api 만들어", "openapi", "rest api 설계"}, "dev_api_design"},
				{[]string{"테스트 코드", "단위 테스트", "test code", "테스트 만들어"}, "dev_test_generate"},
				{[]string{"데일리 스탠드업", "스탠드업", "오늘 뭐 했어", "daily standup"}, "dev_daily_standup"},
				{[]string{"pr 만들어", "pr 자동", "풀리퀘스트 만들어"}, "dev_pr_create"},
				{[]string{"ci/cd", "cicd", "ci 개선", "cd 파이프라인", "파이프라인 최적화"}, "dev_ci_cd"},
				{[]string{"로그 분석", "로그 확인", "log 분석"}, "dev_log_analysis"},
				{[]string{"성능 느려", "성능 병목", "퍼포먼스", "performance 분석"}, "dev_performance"},
				{[]string{"보안 검사", "보안 취약점", "security scan", "취약점 스캔"}, "dev_security_scan"},
				{[]string{"docker", "도커", "kubernetes", "k8s", "도커 설정"}, "dev_docker"},
				{[]string{"쿼리 최적화", "db 최적화", "sql 최적화", "데이터베이스 최적화"}, "dev_db_optimize"},
				{[]string{"기술 학습", "기술 정리", "공부 자료", "정리해", "학습 자료"}, "dev_tech_summary"},
				{[]string{"코드 스타일", "lint", "코딩 컨벤션", "코드 스타일 체크"}, "dev_code_style"},
				{[]string{"마이그레이션", "db 마이그레이션", "스키마 변경", "migration"}, "dev_migration"},
				{[]string{"에러 로그 정리", "에러 분류", "로그 카테고리", "오류 분류"}, "dev_error_classify"},
				{[]string{"주간 리포트", "개발 리포트", "주간 개발", "weekly report"}, "dev_weekly_report"},
				{[]string{"배포 체크", "배포 준비", "릴리즈 체크", "배포 전"}, "dev_deploy_check"},
				{[]string{"기술 트렌드", "개발 트렌드", "tech 트렌드", "최신 기술"}, "dev_tech_trend"},
			},
			"marketer": {
				{[]string{"트렌드 분석", "시장 분석", "이번 주 트렌드", "트렌드 리포트"}, "mkt_trend_analysis"},
				{[]string{"콘텐츠 아이디어", "sns 아이디어", "아이디어 10개", "콘텐츠 기획"}, "mkt_content_idea"},
				{[]string{"경쟁사 분석", "경쟁사 조사", "경쟁사 모니터링", "competitor"}, "mkt_competitor_monitor"},
				{[]string{"광고 문구", "카피라이팅", "광고 카피", "ad copy"}, "mkt_ad_copy"},
				{[]string{"인스타 포스팅", "sns 게시물", "포스팅 만들어", "sns 글"}, "mkt_sns_post"},
				{[]string{"캠페인 기획", "마케팅 캠페인", "캠페인 계획"}, "mkt_campaign_plan"},
				{[]string{"성과 리포트", "마케팅 성과", "이번 달 성과", "kpi 리포트"}, "mkt_performance_report"},
				{[]string{"seo 키워드", "키워드 분석", "검색 키워드", "seo 분석"}, "mkt_seo_keyword"},
				{[]string{"뉴스레터", "이메일 뉴스레터", "newsletter"}, "mkt_email_newsletter"},
				{[]string{"인플루언서 찾아", "인플루언서 검색", "influencer"}, "mkt_influencer_search"},
				{[]string{"a/b 테스트", "ab 테스트", "split test", "ab 테스트 아이디어"}, "mkt_ab_test_idea"},
				{[]string{"해시태그", "hashtag", "태그 만들어"}, "mkt_hashtag_generator"},
				{[]string{"랜딩페이지", "landing page", "랜딩 문구", "cta 문구"}, "mkt_landing_page_copy"},
				{[]string{"소셜 캘린더", "sns 캘린더", "게시 계획", "콘텐츠 캘린더"}, "mkt_social_calendar"},
				{[]string{"예산 계획", "마케팅 예산", "채널 예산", "budget plan"}, "mkt_budget_plan"},
				{[]string{"바이럴", "viral", "바이럴 콘텐츠", "바이럴 전략"}, "mkt_viral_content"},
				{[]string{"고객 인사이트", "고객 분석", "customer insight", "소비자 분석"}, "mkt_customer_insight"},
				{[]string{"브랜드 톤", "브랜드 보이스", "brand voice", "톤 맞춰"}, "mkt_brand_voice"},
				{[]string{"주간 마케팅 요약", "주간 요약", "weekly digest", "마케팅 요약"}, "mkt_weekly_digest"},
				{[]string{"나 홍보", "개인 브랜딩", "personal brand", "linkedin 콘텐츠", "블로그 글"}, "mkt_personal_brand"},
			},
			"sales": {
				{[]string{"고객에게 메일", "영업 이메일", "메일 초안", "이메일 초안"}, "sales_email_draft"},
				{[]string{"미팅 준비", "고객 미팅", "영업 미팅", "내일 미팅"}, "sales_meeting_prep"},
				{[]string{"후속 메일", "followup", "팔로업", "후속 연락"}, "sales_followup"},
				{[]string{"제안서", "제안 초안", "영업 제안", "제안서 만들어"}, "sales_proposal"},
				{[]string{"이의제기", "이의 대응", "반론 대응", "objection"}, "sales_objection"},
				{[]string{"파이프라인", "pipeline", "영업 현황", "파이프라인 정리"}, "sales_pipeline"},
				{[]string{"계약서 만들어", "계약서 초안", "계약 초안"}, "sales_contract"},
				{[]string{"발견 질문", "discovery question", "고객 질문 만들어"}, "sales_discovery_question"},
				{[]string{"데모 스크립트", "demo script", "시연 대본"}, "sales_demo_script"},
				{[]string{"협상 전략", "가격 협상 어떻게", "협상 방법"}, "sales_negotiation"},
				{[]string{"영업 예측", "이번 달 예상", "매출 예측", "forecast"}, "sales_forecast"},
				{[]string{"crm 업데이트", "crm 정리", "crm 입력"}, "sales_crm_update"},
				{[]string{"통화 요약", "콜 요약", "call summary"}, "sales_call_summary"},
				{[]string{"제안서 후속", "proposal followup", "제안 후속"}, "sales_proposal_followup"},
				{[]string{"win loss", "win/loss", "승패 분석", "계약 분석"}, "sales_win_loss_analysis"},
				{[]string{"추천 요청", "referral", "소개 부탁"}, "sales_referral_request"},
				{[]string{"가격 협상", "가격 전략", "할인 전략"}, "sales_price_negotiation"},
				{[]string{"계약서 검토", "계약서 봐줘"}, "sales_contract_review"},
				{[]string{"분기 리뷰", "분기 영업", "quarterly", "분기 결과"}, "sales_quarterly_review"},
				{[]string{"고객 분석해", "고객 프로필", "고객 파악"}, "sales_client_portrait"},
			},
			"pm": {
				{[]string{"요구사항 정리", "요구사항 문서", "기능 정리"}, "pm_requirements"},
				{[]string{"로드맵", "roadmap", "로드맵 만들어"}, "pm_roadmap"},
				{[]string{"이해관계자 브리핑", "stakeholder", "이번 주 브리핑"}, "pm_stakeholder_summary"},
				{[]string{"리스크 분석", "risk", "위험 분석"}, "pm_risk_analysis"},
				{[]string{"미팅 노트", "회의 정리", "회의록 정리"}, "pm_meeting_note"},
				{[]string{"유저 스토리", "user story", "스토리 만들어"}, "pm_user_story"},
				{[]string{"주간 보고서", "주간 보고", "weekly report"}, "pm_weekly_report"},
				{[]string{"prd 작성", "prd 만들어", "기획서 써줘"}, "pm_prd_write"},
				{[]string{"기획서 검토", "스펙 검토", "spec review"}, "pm_spec_review"},
				{[]string{"우선순위 정리", "우선순위 매트릭스", "moscow"}, "pm_priority_matrix"},
				{[]string{"회고 정리", "retrospective", "레트로"}, "pm_retrospective"},
				{[]string{"okr", "okr 세워", "목표 설정"}, "pm_okr_setting"},
				{[]string{"리소스 계획", "인력 배치", "resource plan"}, "pm_resource_plan"},
				{[]string{"이해관계자 맵", "이해관계자 분석", "stakeholder map"}, "pm_stakeholder_map"},
				{[]string{"칸반 정리", "kanban", "보드 정리"}, "pm_feature_kanban"},
				{[]string{"인터뷰 요약", "사용자 인터뷰", "user interview"}, "pm_user_interview_summary"},
				{[]string{"경쟁사 분석", "경쟁 제품", "competitor analysis"}, "pm_competitor_analysis"},
				{[]string{"gtm", "go-to-market", "출시 전략"}, "pm_go_to_market"},
				{[]string{"스프린트 계획", "sprint planning", "sprint"}, "pm_sprint_planning"},
				{[]string{"kpi 대시보드", "지표 정리", "metrics"}, "pm_metrics_dashboard"},
			},
			"designer": {
				{[]string{"레퍼런스 찾아", "비슷한 디자인", "디자인 레퍼런스"}, "design_reference"},
				{[]string{"파일 정리해", "디자인 파일 정리", "폴더 정리"}, "design_file_organize"},
				{[]string{"컬러 팔레트", "color palette", "색상 팔레트"}, "design_color_palette"},
				{[]string{"이미지 정리해", "이미지 편집", "일괄 편집"}, "design_image_edit"},
				{[]string{"포스터 아이디어", "콘텐츠 디자인", "디자인 아이디어"}, "design_content_idea"},
				{[]string{"디자인 피드백", "이 디자인 봐줘", "피드백 해줘"}, "design_feedback"},
				{[]string{"무드보드", "moodboard", "분위기 참고"}, "design_moodboard"},
				{[]string{"ui kit", "ui 키트", "컴포넌트"}, "design_ui_kit"},
				{[]string{"프로토타입 봐줘", "prototype review", "figma 봐줘"}, "design_prototype_review"},
				{[]string{"에셋 정리", "asset export", "에셋 내보내기"}, "design_asset_export"},
				{[]string{"브랜드 가이드", "brand guideline", "브랜드 가이드라인"}, "design_brand_guideline"},
				{[]string{"sns 키트", "소셜 키트", "social media kit"}, "design_social_media_kit"},
				{[]string{"발표 자료 만들어", "슬라이드 만들어", "presentation"}, "design_presentation_deck"},
				{[]string{"아이콘 세트", "icon set", "아이콘 만들어"}, "design_icon_set"},
				{[]string{"폰트 시스템", "타이포그래피", "typography"}, "design_typography"},
				{[]string{"애니메이션 만들어", "lottie", "모션 아이디어"}, "design_animation_idea"},
				{[]string{"접근성 체크", "accessibility", "wcag"}, "design_accessibility_check"},
				{[]string{"반응형 확인", "모바일 확인", "responsive"}, "design_responsive_test"},
				{[]string{"클라이언트 자료", "클라이언트 발표", "client presentation"}, "design_client_presentation"},
				{[]string{"포트폴리오 업데이트", "포트폴리오 정리", "portfolio"}, "design_portfolio_update"},
			},
			"freelancer": {
				{[]string{"클라이언트 정리", "클라이언트 관리", "고객 정리"}, "freelancer_client_manage"},
				{[]string{"견적서 만들어", "견적서", "프로젝트 견적"}, "freelancer_estimate"},
				{[]string{"청구서 만들어", "invoice", "세금계산서"}, "freelancer_invoice"},
				{[]string{"세금 정리", "세금 계산", "종합소득세", "부가세"}, "freelancer_tax"},
				{[]string{"시간 기록", "time tracking", "작업 시간"}, "freelancer_time_track"},
				{[]string{"포트폴리오 업데이트", "포트폴리오 정리"}, "freelancer_portfolio"},
				{[]string{"나 홍보", "자기 pr", "self marketing"}, "freelancer_self_marketing"},
				{[]string{"계약서 검토", "계약서 봐줘"}, "freelancer_contract_review"},
				{[]string{"현금 흐름", "cash flow", "수입 지출"}, "freelancer_cashflow"},
				{[]string{"세금 신고 자료", "연말정산", "부가세 신고"}, "freelancer_tax_report"},
				{[]string{"신규 클라이언트", "온보딩", "client onboarding"}, "freelancer_client_onboarding"},
				{[]string{"프로젝트 시작", "킥오프", "kickoff"}, "freelancer_project_kickoff"},
				{[]string{"산출물 확인", "deliverable", "최종 파일 확인"}, "freelancer_deliverable_check"},
				{[]string{"미수금 독촉", "payment reminder", "미수금"}, "freelancer_payment_reminder"},
				{[]string{"제안서 템플릿", "proposal template"}, "freelancer_proposal_template"},
				{[]string{"단가 계산", "적정 단가", "시간당 단가"}, "freelancer_rate_calculation"},
				{[]string{"작업 로그", "work log", "오늘 작업"}, "freelancer_work_log"},
				{[]string{"사업 계획", "business plan", "사업 계획서"}, "freelancer_business_plan"},
				{[]string{"네트워킹 콘텐츠", "linkedin 포스팅", "networking"}, "freelancer_networking_content"},
				{[]string{"올해 정리", "연간 리뷰", "yearly review"}, "freelancer_yearly_review"},
			},
		}
		if presets, ok := presetMap[pid]; ok {
			for _, pd := range presets {
				for _, t := range pd.triggers {
					if strings.Contains(msgLower, t) {
						preRoutedAction = "workflow_preset"
						preRoutedParams = map[string]any{"preset": pd.preset, "query": req.Message}
						break
					}
				}
				if preRoutedAction != "" {
					break
				}
			}
		}
	}

	// ── Intent 분류 + Clarify 판단 (워크플로우 프리셋은 건너뜀) ─────────────
	// systemPreRouted (PC/보안/정리 패턴 매칭) 도 LLM 스킵 → clarify 로 가로채임 방지
	var structuredResult *ClarifyResult
	if preRoutedAction != "workflow_preset" && req.PendingIntent == "" && !systemPreRouted {
		clarifyNow := func(questions []string, pi string, pp map[string]any) {
			q := ""
			if len(questions) > 0 {
				q = questions[0]
			}
			d := fmt.Sprintf("%.2fs", time.Since(start).Seconds())
			json200(w, CommandResponse{
				Success: true, Message: q, Action: "clarify",
				NeedsClarify: true, ClarifyQuestion: q, ClarifyQuestions: questions,
				PendingIntent: pi, PendingParams: pp, Duration: d,
			})
		}

		groqCtx := req.Message
		if preRoutedAction != "" {
			groqCtx = fmt.Sprintf("[감지된 액션: %s]\n사용자 요청: %s", preRoutedAction, req.Message)
		}

		// Claude Haiku 우선, 없으면 Groq fallback
		var cr1 *ClarifyResult
		var err1 error
		llmMu.RLock()
		hasClaude := llmClaudeKey != ""
		llmMu.RUnlock()
		if hasClaude {
			cr1, err1 = callClaudeIntent(groqCtx)
		}
		if !hasClaude || err1 != nil {
			cr1, err1 = callGroqStructured(groqCtx)
		}
		if err1 == nil {
			structuredResult = cr1
			if cr1.NeedsClarify {
				pi := preRoutedAction
				if pi == "" && len(cr1.Intents) > 0 {
					pi = cr1.Intents[0].Action
				}
				clarifyNow(cr1.ClarifyQuestions, pi, preRoutedParams)
				return
			}
			// multi-intent: 2개 이상의 intent를 병렬로 처리
			if preRoutedAction == "" && len(cr1.Intents) >= 2 {
				type partResult struct {
					desc string
					text string
				}
				parts := make([]partResult, len(cr1.Intents))
				var wgM sync.WaitGroup
				for i, it := range cr1.Intents {
					wgM.Add(1)
					go func(idx int, item IntentItem) {
						defer wgM.Done()
						var txt string
						searchQ := func(q string) string {
							r := runWebSearchMac(gKey, q, "auto", 5)
							return r.Summary
						}
						switch item.Action {
						case "web_search", "trip_plan":
							q, _ := item.Params["query"].(string)
							if q == "" {
								q, _ = item.Params["destination"].(string)
							}
							if q == "" {
								q = req.Message
							}
							txt = searchQ(q)
						case "weather":
							city, _ := item.Params["city"].(string)
							txt = searchQ(city + " 날씨")
						default:
							q, _ := item.Params["query"].(string)
							if q == "" {
								q = req.Message
							}
							txt = searchQ(q)
						}
						parts[idx] = partResult{desc: item.Description, text: txt}
					}(i, it)
				}
				wgM.Wait()

				combined := ""
				for _, p := range parts {
					if p.text != "" {
						if p.desc != "" {
							combined += "### " + p.desc + "\n" + p.text + "\n\n"
						} else {
							combined += p.text + "\n\n"
						}
					}
				}
				if combined == "" {
					combined = "검색 결과를 가져오지 못했습니다."
				}
				dur := fmt.Sprintf("%.2fs", time.Since(start).Seconds())
				json200(w, CommandResponse{
					Success: true, Action: "web_search", Message: strings.TrimSpace(combined),
					Duration: dur,
				})
				return
			}
		} else {
			// Groq 에러 시 보수적 처리
			if len([]rune(strings.TrimSpace(req.Message))) < 8 {
				clarifyNow([]string{"무엇을 도와드릴까요? 조금 더 구체적으로 알려주세요."}, "chat", nil)
				return
			}
		}
	}

	var intent struct {
		Action  string         `json:"action"`
		Params  map[string]any `json:"params"`
		Message string         `json:"message"`
	}

	if preRoutedAction != "" {
		intent.Action = preRoutedAction
		intent.Params = preRoutedParams
	} else if structuredResult != nil && len(structuredResult.Intents) == 1 {
		// structured result에서 단일 intent를 바로 사용 — 두 번째 LLM 호출 불필요
		it := structuredResult.Intents[0]
		intent.Action = it.Action
		intent.Params = it.Params
	} else {
		// fallback: LLM으로 의도 파악
		sysPr := macSystemPrompt
		if req.Lang == "en" || isEnglishQuery(req.Message) {
			sysPr += "\n⚠️ IMPORTANT: The user is writing in English. The 'message' field in your JSON response MUST be in English."
		}
		msgs := []groqMsg{
			{Role: "system", Content: sysPr},
			{Role: "user", Content: intentPrompt},
		}
		raw, _, err := callGroqWithFallback(msgs, 500, true)
		if err != nil {
			writeJSON(w, 500, map[string]any{"success": false, "message": "LLM 오류: " + err.Error()})
			return
		}
		// 멀티 액션 응답 먼저 체크 — actions[] 배열 우선
		var multi struct {
			Actions []struct {
				Action string         `json:"action"`
				Params map[string]any `json:"params"`
			} `json:"actions"`
		}
		if jErr := json.Unmarshal([]byte(raw), &multi); jErr == nil && len(multi.Actions) > 1 {
			// 여러 액션 → workflow_run 으로 위임
			actionsJSON, _ := json.Marshal(multi.Actions)
			intent.Action = "workflow_run"
			intent.Params = map[string]any{
				"goal":            req.Message,
				"planned_actions": string(actionsJSON),
			}
		} else if err := json.Unmarshal([]byte(raw), &intent); err != nil {
			intent.Action = "chat"
			intent.Message = raw
		}
	}

	// ── AI 요청 사용량 체크 (플랜별 한도 적용) ──────────────────
	uid, _ := resolveUserID(userID)
	if allowed, used, lim := checkUsageLimit(uid, "ai_request"); !allowed {
		dur := fmt.Sprintf("%.2fs", time.Since(start).Seconds())
		resp := upgradeRequiredResponse("ai_request", used, lim)
		resp.Duration = dur
		json200(w, resp)
		return
	}
	incrementUsage(uid, "ai_request")

	dur := fmt.Sprintf("%.2fs", time.Since(start).Seconds())


	llmMu.RLock()
	cx_tKey := llmTavilyKey
	cx_req := req
	llmMu.RUnlock()
	_cx := cmdCtx{
		w: w, req: cx_req, params: intent.Params, msg: intent.Message,
		dur: dur, gKey: gKey, tKey: cx_tKey, userID: userID, start: start,
	}

	switch intent.Action {
	case "clarify":
		cmdClarify(_cx)
	case "chat":
		cmdChat(_cx)
	case "weather":
		cmdWeather(_cx)
	case "calendar_today":
		cmdCalendarToday(_cx)
	case "calendar_add":
		cmdCalendarAdd(_cx)
	case "price_compare":
		cmdPriceCompare(_cx)
	case "video_search":
		cmdVideoSearch(_cx)
	case "web_search":
		cmdWebSearch(_cx)
	case "persona_switch":
		cmdPersonaSwitch(_cx)
	case "workflow_run":
		cmdWorkflowRunStub(_cx)
	case "workflow_plan":
		cmdWorkflowPlan(_cx)
	case "trip_plan":
		cmdTripPlan(_cx)
	case "workflow_preset":
		cmdWorkflowPreset(_cx)
	case "multi_action":
		cmdMultiAction(_cx)
	case "directions":
		cmdDirections(_cx)
	case "place_view":
		cmdPlaceView(_cx)
	case "multi_agent":
		cmdMultiAgent(_cx)
	case "email":
		cmdEmail(_cx)
	case "meeting":
		cmdMeeting(_cx)
	case "briefing":
		cmdBriefing(_cx)
	case "file_search":
		cmdFileSearch(_cx)
	case "scan":
		cmdScan(_cx)
	case "clean":
		cmdClean(_cx)
	case "stats":
		cmdStats(_cx)
	case "launch_app":
		cmdLaunchApp(_cx)
	case "system_control":
		cmdSystemControl(_cx)
	case "note":
		cmdNote(_cx)
	case "focus_mode":
		cmdFocusMode(_cx)
	case "doc_summary":
		cmdDocSummary(_cx)
	case "health_report":
		cmdHealthReport(_cx)
	case "excel_save":
		cmdExcelSave(_cx)
	case "excel_auto_create", "create_excel", "make_excel":
		// Jarvis 원칙: 데이터 없으면 LLM이 만든다
		result, msg := cmdExcelAutoCreate(_cx.params, _cx.req.Message, _cx.gKey, _cx.req.Lang)
		json200(_cx.w, CommandResponse{
			Success:  true, Message: msg, Action: "excel_auto_create",
			Result:   result, Duration: _cx.dur,
		})
	case "doc_auto_create", "create_doc", "make_doc":
		// 문서 자동 생성 (TXT/MD/HTML)
		result, msg := cmdDocAutoCreate(_cx.params, _cx.req.Message, _cx.gKey, _cx.req.Lang)
		json200(_cx.w, CommandResponse{
			Success:  true, Message: msg, Action: "doc_auto_create",
			Result:   result, Duration: _cx.dur,
		})
	case "pdf_auto_create", "create_pdf", "make_pdf":
		// PDF 자동 생성 (gofpdf)
		result, msg := cmdPdfAutoCreate(_cx.params, _cx.req.Message, _cx.gKey, _cx.req.Lang)
		json200(_cx.w, CommandResponse{
			Success:  true, Message: msg, Action: "pdf_auto_create",
			Result:   result, Duration: _cx.dur,
		})
	case "video_workflow", "video_summary", "video_download_summary":
		// 영상 통합 워크플로 (Python sidecar 활용)
		result, msg := cmdVideoWorkflow(_cx.params, _cx.req.Message, _cx.gKey, _cx.req.Lang)
		json200(_cx.w, CommandResponse{
			Success:  true, Message: msg, Action: "video_workflow",
			Result:   result, Duration: _cx.dur,
		})
	case "excel_analyze", "analyze_excel":
		// 사용자 Excel 분석 (사장님 원칙 #3)
		result, msg := cmdExcelAnalyze(_cx.params, _cx.req.Message, _cx.gKey, _cx.req.Lang)
		json200(_cx.w, CommandResponse{
			Success:  true, Message: msg, Action: "excel_analyze",
			Result:   result, Duration: _cx.dur,
		})
	case "recall":
		cmdRecall(_cx)
	case "timer":
		cmdTimer(_cx)
	case "browse_page":
		cmdBrowsePage(_cx)
	case "file_ops":
		cmdFileOps(_cx)
	case "trigger_add":
		cmdTriggerAdd(_cx)
	// ── 직업 페르소나 전문 액션 (Mac stub: chat 폴백) ──
	case "contract_review", "legal_search", "medical_search", "stock_analysis":
		// 페르소나 컨텍스트가 이미 자동 적용됨 → cmdDefault 통해 chat 응답
		cmdDefault(_cx)
	// ── Phase A2 위험 액션 (확인 카드 응답) ──
	case "restart", "shutdown", "sleep", "format_disk", "payment", "file_delete",
		"clean_aggressive", "email_send", "app_uninstall", "registry_edit", "file_move":
		cmdDangerousConfirm(_cx, intent.Action)
	case "vision":
		// Phase C1 — Vision (스크린샷 분석)
		question, _ := _cx.params["question"].(string)
		if question == "" {
			question = _cx.req.Message
		}
		json200(_cx.w, CommandResponse{
			Success: true, Action: "vision",
			Message: "스크린샷을 분석하려면 화면 캡처가 필요해요 (프론트에서 자동 수행).",
			Result: map[string]any{
				"question": question,
				"hint":     "frontend should capture screen and call /api/vision/analyze",
			},
			Duration: _cx.dur,
		})
	case "screen_analyze":
		cmdScreenAnalyze(_cx)
	case "clipboard_action":
		cmdClipboardAction(_cx)
	case "exchange_rate":
		cmdExchangeRate(_cx)
	case "stock":
		cmdStock(_cx)
	case "windows_only":
		cmdWindowsOnly(_cx)
	case "deep_research":
		cmdDeepResearch(_cx)
	// ── 프론트 디스패치 액션 (Mac: 의도만 인식, 프론트가 backendAPI 호출) ──
	case "calendar_week", "calendar_find_slot",
		"email_classify", "email_draft", "email_summarize", "imap_inbox", "email_inbox",
		"meeting_list", "meeting_summary", "notes",
		"perf_history", "perf_anomaly", "gpu_stats", "process_top",
		"process_security", "startup_items", "programs_list", "app_permissions",
		"boot_analysis", "driver_check", "defender_status", "virus_check",
		"windows_updates", "remote_access", "network_analysis",
		"clipboard_history", "clipboard_ai", "dictation_start",
		"file_duplicates", "search_pdf",
		"recall_capture", "recall_search", "brain_search", "brain_stats",
		"persona_list", "workflow_list", "workflow_templates", "schedule_list",
		"voice_todo", "news_search", "youtube_search", "translate":
		cmdFrontendDispatch(_cx, intent.Action)
	default:
		cmdDefault(_cx)
	}
}

// cmdWorkflowRunStub — Mac: 멀티 액션 순차 실행 (Win은 runWithReflection, Mac은 단순 순차)
func cmdWorkflowRunStub(cx cmdCtx) {
	goal, _ := cx.params["goal"].(string)
	if goal == "" {
		goal = cx.req.Message
	}
	plannedJSON, _ := cx.params["planned_actions"].(string)
	var planned []struct {
		Action string         `json:"action"`
		Params map[string]any `json:"params"`
	}
	if plannedJSON != "" {
		_ = json.Unmarshal([]byte(plannedJSON), &planned)
	}

	// ★ planned_actions가 없으면 키워드 기반으로 단계 자동 생성 (LLM 호출 없이)
	if len(planned) == 0 {
		planned = synthesizePlannedSteps(goal)
	}

	results := make([]map[string]any, 0, len(planned))
	doneCount := 0
	for i, p := range planned {
		results = append(results, map[string]any{
			"step":   i + 1,
			"action": p.Action,
			"params": p.Params,
			"status": "queued",
		})
		doneCount++
	}
	var msg string
	if len(planned) > 0 {
		if cx.req.Lang == "en" {
			msg = fmt.Sprintf("Multi-step workflow: %d actions identified", len(planned))
		} else {
			msg = fmt.Sprintf("멀티 액션 %d개 단계 인식됨", len(planned))
		}
	} else {
		if cx.req.Lang == "en" {
			msg = fmt.Sprintf("Workflow goal recognized: %s", goal)
		} else {
			msg = fmt.Sprintf("워크플로 목표 인식: %s", goal)
		}
	}
	json200(cx.w, CommandResponse{
		Success: true, Message: msg, Action: "workflow_run",
		Result: map[string]any{
			"goal":    goal,
			"steps":   results,
			"summary": msg,
			"ok":      doneCount > 0,
			"mode":    "planned_actions_stub",
		},
		Duration: cx.dur,
	})
}


// cmdDangerousConfirm — Phase A2: 위험 액션 즉시 실행 대신 확인 카드 반환
func cmdDangerousConfirm(cx cmdCtx, action string) {
	// 이미 confirmed 면 (사용자가 확인 카드 클릭 후 재요청) 실제 실행 dispatch로
	if c, _ := cx.params["confirmed"].(bool); c {
		json200(cx.w, CommandResponse{
			Success: true, Action: action + "_confirmed",
			Message: "✅ 확인됨 — 실제 실행은 Windows 빌드에서만 동작합니다.",
			Result:  map[string]any{"action": action, "executed": false, "reason": "mac-dev-only"},
			Duration: cx.dur,
		})
		return
	}
	confirmResult, confirmMsg := buildConfirmCard(action, cx.params, cx.msg, cx.req.Lang)
	if confirmResult == nil {
		confirmResult = map[string]any{"needs_confirmation": true, "action": action}
		confirmMsg = "⚠️ " + action + " 실행 확인 필요"
	}
	json200(cx.w, CommandResponse{
		Success: true, Action: action,
		Message:  confirmMsg,
		Result:   confirmResult,
		CardType: "confirm_action",
		Duration: cx.dur,
	})
}

// synthesizePlannedSteps — 자연어에서 멀티스텝 분해 (LLM 없이 키워드 기반)
// "엑셀로 매출 정리하고 PDF로 저장" → [excel_auto_create, pdf_auto_create]
func synthesizePlannedSteps(goal string) []struct {
	Action string         `json:"action"`
	Params map[string]any `json:"params"`
} {
	out := []struct {
		Action string         `json:"action"`
		Params map[string]any `json:"params"`
	}{}
	lower := strings.ToLower(goal)
	add := func(a string, p map[string]any) {
		out = append(out, struct {
			Action string         `json:"action"`
			Params map[string]any `json:"params"`
		}{Action: a, Params: p})
	}
	// 단계 후보들 — 키워드 매칭 순서대로 추가
	candidates := []struct {
		action  string
		keywords []string
	}{
		{"calendar_today", []string{"오늘 일정", "오늘 스케줄"}},
		{"calendar_find_slot", []string{"빈 시간", "회의 잡", "미팅 잡"}},
		{"email_inbox", []string{"받은 메일", "메일 확인", "inbox"}},
		{"email_summarize", []string{"메일 요약", "이메일 요약"}},
		{"news_search", []string{"뉴스", "news"}},
		{"web_search", []string{"검색해", "찾아"}},
		{"excel_auto_create", []string{"엑셀", "excel", "스프레드시트"}},
		{"pdf_auto_create", []string{"pdf", "피디에프"}},
		{"doc_auto_create", []string{"보고서", "메모", "회의록"}},
		{"note", []string{"노트", "메모"}},
		{"scan", []string{"진단", "스캔"}},
		{"clean", []string{"정리해", "청소"}},
		{"translate", []string{"번역"}},
		{"stats", []string{"상태", "메모리", "디스크"}},
	}
	seen := map[string]bool{}
	for _, c := range candidates {
		for _, kw := range c.keywords {
			if strings.Contains(lower, kw) && !seen[c.action] {
				seen[c.action] = true
				add(c.action, map[string]any{"topic": goal, "query": goal})
				break
			}
		}
	}
	return out
}

// cmdFrontendDispatch — 프론트가 backendAPI 호출로 데이터를 가져오는 액션
// 백엔드는 의도만 confirm하고 action 이름 전달
func cmdFrontendDispatch(cx cmdCtx, action string) {
	labels := map[string]string{
		"calendar_today": "오늘 일정", "calendar_week": "주간 일정", "calendar_find_slot": "빈 시간",
		"email_classify": "이메일 분류", "email_draft": "이메일 초안", "email_summarize": "이메일 요약",
		"imap_inbox": "메일함", "email_inbox": "받은 메일",
		"meeting_list": "회의 목록", "meeting_summary": "회의 요약", "notes": "노트",
		"perf_history": "성능 기록", "perf_anomaly": "성능 이상", "gpu_stats": "GPU 상태",
		"process_top": "프로세스 TOP", "process_security": "프로세스 보안",
		"startup_items": "시작 프로그램", "programs_list": "설치 프로그램", "app_permissions": "앱 권한",
		"boot_analysis": "부팅 분석", "driver_check": "드라이버 점검", "defender_status": "디펜더 상태",
		"virus_check": "바이러스 검사", "windows_updates": "Windows 업데이트", "remote_access": "원격 접속",
		"network_analysis": "네트워크 분석",
		"clipboard_history": "클립보드 기록", "clipboard_ai": "클립보드 AI",
		"dictation_start": "받아쓰기", "file_duplicates": "중복 파일", "search_pdf": "PDF 검색",
		"recall_capture": "기억 저장", "recall_search": "기억 검색",
		"brain_search": "브레인 검색", "brain_stats": "브레인 통계",
		"persona_list": "페르소나 목록",
		"workflow_list": "워크플로 목록", "workflow_templates": "워크플로 템플릿", "schedule_list": "예약 목록",
		"voice_todo": "음성 할일", "news_search": "뉴스", "youtube_search": "유튜브", "translate": "번역",
	}
	label := labels[action]
	if label == "" {
		label = action
	}
	var msg string
	if cx.req.Lang == "en" {
		msg = fmt.Sprintf("%s — opening…", label)
	} else {
		msg = fmt.Sprintf("%s 준비 완료", label)
	}
	json200(cx.w, CommandResponse{
		Success: true, Message: msg, Action: action,
		Result: map[string]any{
			"action":            action,
			"params":            cx.params,
			"frontend_dispatch": true,
		},
		Duration: cx.dur,
	})
}
