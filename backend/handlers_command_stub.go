//go:build !windows

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ── 멀티 액션: 출력 포맷 감지 ──────────────────────────────────
type outputFormat string

const (
	outPDF      outputFormat = "pdf"
	outExcel    outputFormat = "excel"
	outWord     outputFormat = "word"
	outMarkdown outputFormat = "markdown"
	outTXT      outputFormat = "txt"
	outNone     outputFormat = ""
)

func detectOutputFormat(msg string) outputFormat {
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "pdf") || strings.Contains(lower, "피디에프"):
		return outPDF
	case strings.Contains(lower, "excel") || strings.Contains(lower, "엑셀") || strings.Contains(lower, "xlsx"):
		return outExcel
	case strings.Contains(lower, "word") || strings.Contains(lower, "워드") || strings.Contains(lower, "docx"):
		return outWord
	case strings.Contains(lower, "마크다운") || strings.Contains(lower, "markdown") || strings.Contains(lower, ".md"):
		return outMarkdown
	case strings.Contains(lower, "txt") || strings.Contains(lower, "텍스트 파일") || strings.Contains(lower, "텍스트로 저장"):
		return outTXT
	}
	return outNone
}

// 파일 저장 동사 감지 (멀티 액션 트리거)
func hasFileSaveVerb(msg string) bool {
	lower := strings.ToLower(msg)
	saveVerbs := []string{
		"저장", "만들어", "작성", "정리", "보고서", "리포트", "report",
		"파일로", "제품설명서", "설명서", "요약해서", "뽑아줘", "출력",
	}
	for _, v := range saveVerbs {
		if strings.Contains(lower, v) {
			return true
		}
	}
	return false
}

// 멀티 액션 결과를 파일로 저장
func saveResultToFile(format outputFormat, title string, items []map[string]string, summary string) (string, error) {
	home, _ := os.UserHomeDir()
	ts := time.Now().Format("20060102_150405")
	safeName := strings.Map(func(r rune) rune {
		if r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r >= '가' && r <= '힣' {
			return r
		}
		return '_'
	}, title)
	if len([]rune(safeName)) > 20 {
		safeName = string([]rune(safeName)[:20])
	}

	switch format {
	case outMarkdown:
		path := filepath.Join(home, "Desktop", fmt.Sprintf("nexus_%s_%s.md", safeName, ts))
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("# %s\n\n", title))
		sb.WriteString(fmt.Sprintf("*생성: %s*\n\n", time.Now().Format("2006-01-02 15:04:05")))
		if summary != "" {
			sb.WriteString("## 요약\n\n")
			sb.WriteString(summary + "\n\n")
		}
		if len(items) > 0 {
			sb.WriteString("## 항목\n\n")
			for i, it := range items {
				name := it["title"]
				if name == "" { name = it["name"] }
				url := it["url"]
				if url == "" { url = it["link"] }
				price := it["price"]
				if price != "" {
					sb.WriteString(fmt.Sprintf("%d. **%s** — %s\n   %s\n\n", i+1, name, price, url))
				} else {
					sb.WriteString(fmt.Sprintf("%d. [%s](%s)\n\n", i+1, name, url))
				}
			}
		}
		return path, os.WriteFile(path, []byte(sb.String()), 0644)

	case outTXT:
		path := filepath.Join(home, "Desktop", fmt.Sprintf("nexus_%s_%s.txt", safeName, ts))
		var sb strings.Builder
		sb.WriteString(title + "\n")
		sb.WriteString(strings.Repeat("=", 40) + "\n")
		sb.WriteString("생성: " + time.Now().Format("2006-01-02 15:04:05") + "\n\n")
		if summary != "" {
			sb.WriteString("[ 요약 ]\n" + summary + "\n\n")
		}
		if len(items) > 0 {
			sb.WriteString("[ 항목 ]\n")
			for i, it := range items {
				name := it["title"]
				if name == "" { name = it["name"] }
				url := it["url"]
				if url == "" { url = it["link"] }
				price := it["price"]
				if price != "" {
					sb.WriteString(fmt.Sprintf("%d. %s — %s\n   %s\n\n", i+1, name, price, url))
				} else {
					sb.WriteString(fmt.Sprintf("%d. %s\n   %s\n\n", i+1, name, url))
				}
			}
		}
		return path, os.WriteFile(path, []byte(sb.String()), 0644)

	case outExcel:
		// CSV 형식으로 저장 (Excel에서 열 수 있음)
		path := filepath.Join(home, "Desktop", fmt.Sprintf("nexus_%s_%s.csv", safeName, ts))
		var sb strings.Builder
		sb.WriteString("번호,제목/상품명,가격,링크\n")
		for i, it := range items {
			name := it["title"]
			if name == "" { name = it["name"] }
			url := it["url"]
			if url == "" { url = it["link"] }
			price := it["price"]
			sb.WriteString(fmt.Sprintf("%d,\"%s\",\"%s\",\"%s\"\n", i+1,
				strings.ReplaceAll(name, `"`, `""`),
				strings.ReplaceAll(price, `"`, `""`),
				url))
		}
		return path, os.WriteFile(path, []byte(sb.String()), 0644)

	case outPDF, outWord:
		// Mac에서는 HTML → PDF 변환 없이 Markdown으로 대체 저장
		path := filepath.Join(home, "Desktop", fmt.Sprintf("nexus_%s_%s.md", safeName, ts))
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("# %s\n\n", title))
		sb.WriteString(fmt.Sprintf("> 생성: %s  \n> 형식: %s (Mac에서는 Markdown으로 저장)\n\n", time.Now().Format("2006-01-02 15:04"), strings.ToUpper(string(format))))
		if summary != "" {
			sb.WriteString("## 요약\n\n" + summary + "\n\n")
		}
		if len(items) > 0 {
			sb.WriteString("## 항목 목록\n\n")
			for i, it := range items {
				name := it["title"]
				if name == "" { name = it["name"] }
				url := it["url"]
				if url == "" { url = it["link"] }
				price := it["price"]
				sb.WriteString(fmt.Sprintf("### %d. %s\n", i+1, name))
				if price != "" { sb.WriteString(fmt.Sprintf("- **가격**: %s\n", price)) }
				if url != "" { sb.WriteString(fmt.Sprintf("- **링크**: %s\n", url)) }
				sb.WriteString("\n")
			}
		}
		return path, os.WriteFile(path, []byte(sb.String()), 0644)
	}
	return "", fmt.Errorf("지원하지 않는 형식")
}

type CommandRequest struct {
	Message         string              `json:"message"`
	Context         string              `json:"context"`
	Lang            string              `json:"lang"`
	PendingIntent   string              `json:"pending_intent"`
	PendingParams   map[string]any      `json:"pending_params"`
	PendingQuestion string              `json:"pending_question"`
	History         []ConvHistoryMsg    `json:"history"`
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

"web_search" → 쇼핑/최저가/뉴스/맛집/유튜브/틱톡/쿠팡/네이버 검색 (파일 저장 없는 단순 검색)
  params: {"query":"검색어","site":"coupang|naver|youtube|tiktok|google|auto","max_items":5}

"multi_action" → 검색/비교/정리 결과를 파일(PDF·Excel·MD·TXT)로 저장하거나, 가격비교·영상검색을 수행할 때
  트리거 키워드: "정리해줘", "요약해줘", "pdf로", "엑셀로", "엑셀 작성", "파일로 만들어줘", "저장해줘", "보고서 만들어줘", "비교해줘", "비교 정리", "비교표", "vs", "차이점 정리", "표로 만들어줘"
  params: {
    "sub_action": "price_compare|video_search|doc_compare|summarize|web_search",
    "query": "검색/비교/요약 대상",
    "format": "pdf|excel|markdown|txt",
    "max_items": 8
  }
  sub_action 선택 기준:
  - "비교해줘" / "vs" / "차이점" → "doc_compare"
  - "요약해줘" / "정리해줘" (특정 주제) → "summarize"
  - "가격 비교" / "최저가" → "price_compare"
  - 유튜브/틱톡 + 저장 → "video_search"
  - 그 외 검색 + 저장 → "web_search"
  format 선택 기준:
  - "pdf로" / "PDF" → "pdf"
  - "엑셀로" / "xlsx" / "엑셀" → "excel"
  - "마크다운" / ".md" → "markdown"
  - "텍스트" / ".txt" → "txt"
  - 키워드 없으면 → "markdown" (기본값)

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

"trip_plan" → 출장/여행 자동 준비 (항공권·호텔·날씨·맛집·환율 한 번에)
  트리거: "출장", "여행 준비", "출장 준비", "trip", "여행 계획"
  params: {"destination":"목적지","date":"출발일YYYY-MM-DD","days":1,"purpose":"출장|여행"}

"windows_only" → Windows PC 제어 기능 (볼륨, 보안, 프로세스 등)
  params: {"feature":"기능명"}

"clarify" → 실행에 필수 정보가 없을 때만 사용
  params: {"question":"주인님께 물을 질문(1가지만)","missing":"없는 정보","intent":"원래 액션명","collected":{...지금까지 파악된 파라미터...}}

판단 기준:
- 날씨/기상 → weather (도시 없으면 clarify)
- 일정/캘린더/스케줄 → calendar_today 또는 calendar_add (날짜 없으면 clarify)
- 쇼핑/검색/맛집/뉴스/유튜브/틱톡 → web_search (맛집인데 지역 없으면 clarify)
- 다음 중 하나라도 포함 → 반드시 multi_action:
  · "정리해줘", "정리해", "정리하여", "정리 좀"
  · "요약해줘", "요약해", "요약 정리"
  · "pdf로", "PDF", "피디에프"
  · "엑셀로", "엑셀에", "엑셀 파일", "xlsx", "Excel"
  · "마크다운으로", "md로"
  · "파일로 만들어", "저장해줘", "저장해"
  · "비교해줘", "비교해", "비교 정리", "비교표", "vs", "VS", "대비"
  · "표로 만들어", "표로 정리"
  · "보고서", "리포트", "report"
- "출장", "여행 준비", "출장 준비" → trip_plan (목적지 없으면 clarify)
- PC제어/보안/최적화/볼륨/밝기 → windows_only
- 그 외 모든 대화 → chat

⚠️ 중요: "엑셀로 정리해줘", "엑셀 파일로 만들어줘" 처럼 엑셀 키워드가 있으면 무조건 multi_action + format=excel

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

	// ── 키워드 사전 라우팅 (LLM보다 우선, 틱톡/유튜브 영상 검색) ──
	msgLower := strings.ToLower(req.Message)
	videoVerbs := []string{"찾", "검색", "영상", "보여", "추천", "viral", "바이럴", "트렌드"}
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
	isMultiAction := outFmt != outNone && hasFileSaveVerb(req.Message) && req.PendingIntent == ""

	// 특정 사이트가 감지되면 바로 price_compare (priceVerb 불필요)
	if detectedShopSite != "" && req.PendingIntent == "" {
		q := req.Message
		for kw := range shoppingSites {
			q = strings.ReplaceAll(q, kw, "")
		}
		for _, rm := range []string{"에서", "찾아줘", "검색해줘", "최저가", "가격", "얼마야", "구매", "사고 싶어"} {
			q = strings.ReplaceAll(q, rm, "")
		}
		q = strings.TrimSpace(q)
		if q == "" {
			q = req.Message
		}
		if isMultiAction {
			preRoutedAction = "multi_action"
			preRoutedParams = map[string]any{
				"sub_action": "price_compare",
				"query":      q,
				"site":       detectedShopSite,
				"max_items":  8,
				"format":     string(outFmt),
			}
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
		if q == "" {
			q = "바이럴 트렌드"
		}
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
		if q == "" {
			q = "인기 영상"
		}
		if isMultiAction {
			preRoutedAction = "multi_action"
			preRoutedParams = map[string]any{"sub_action": "video_search", "query": q, "platform": "youtube", "max_items": 8, "format": string(outFmt)}
		} else {
			preRoutedAction = "video_search"
			preRoutedParams = map[string]any{"query": q, "platform": "youtube", "max_items": 8}
		}
	} else if isMultiAction && req.PendingIntent == "" {
		// 정리/요약/비교/파일 저장 키워드 감지 → pre-route to multi_action
		lower := strings.ToLower(req.Message)
		subAction := "summarize"
		compareVerbs := []string{"비교해줘", "비교해", "비교 정리", "비교표", " vs ", "vs.", "대비"}
		for _, v := range compareVerbs {
			if strings.Contains(lower, v) {
				subAction = "doc_compare"
				break
			}
		}
		preRoutedAction = "multi_action"
		preRoutedParams = map[string]any{
			"sub_action": subAction,
			"query":      req.Message,
			"format":     string(outFmt),
			"max_items":  8,
		}
	}

	// 출장/여행 준비 pre-routing
	if preRoutedAction == "" && req.PendingIntent == "" {
		tripVerbs := []string{"출장 준비", "여행 준비", "출장 계획", "여행 계획", "출장 가", "출장이야", "출장인데", "출장 있", "여행 있", "trip 준비"}
		for _, v := range tripVerbs {
			if strings.Contains(msgLower, v) {
				// 목적지 추출 시도 (LLM에 위임)
				preRoutedAction = "trip_plan"
				preRoutedParams = map[string]any{
					"destination": req.Message,
					"purpose":     "출장",
				}
				break
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
	} else {
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
		if err := json.Unmarshal([]byte(raw), &intent); err != nil {
			intent.Action = "chat"
			intent.Message = raw
		}
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
		cat := detectCategory(req.Message)
		expertList := detectExperts(req.Message, req.Lang)
		previewType := categoryPreviewType(cat)

		var answer string
		var chatItems []map[string]string
		var wg sync.WaitGroup
		wg.Add(2)

		// 고루틴 A: LLM 답변 (전문가 or 일반)
		go func() {
			defer wg.Done()
			if len(expertList) > 0 {
				answer, _ = runExpertParallel(req.Message, req.Lang, gKey, expertList, req.History)
			}
			if answer == "" {
				lang := req.Lang
				var sysPrompt string
				if lang == "en" {
					sysPrompt = "You are Nexus AI, a helpful assistant. Answer in natural English, 2-4 sentences. No markdown headers."
				} else {
					sysPrompt = "당신은 Nexus AI 한국어 비서입니다. 자연스러운 한국어로 2~4문장 답변. 마크다운 헤더 금지."
				}
				msgs := []groqMsg{{Role: "system", Content: sysPrompt}, {Role: "user", Content: req.Message}}
				answer, _, _ = callGroqWithCitations(gKey, groqChatModel, msgs, 600)
				if answer == "" {
					answer = "죄송합니다, 답변을 생성하는 중 오류가 발생했습니다."
				}
			}
		}()

		// 고루틴 B: 카테고리별 상세 페이지 검색
		go func() {
			defer wg.Done()
			expertCat := expertsToCategory(expertList)
			pr := parallelWebSearch(req.Message, 6, expertCat)
			if len(pr.Items) > 0 {
				chatItems = pr.Items
			} else {
				searchCat := cat
				if expertCat >= 0 {
					searchCat = expertCat
				}
				chatItems = categoryFallbackSites(req.Message, searchCat)
			}
		}()
		wg.Wait()

		json200(w, CommandResponse{
			Success:  true,
			Message:  answer,
			Action:   "chat",
			Result:   map[string]any{"reply": answer, "items": chatItems, "preview_type": previewType},
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

	case "price_compare":
		var query, site string
		maxItems := 8
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
		json200(w, CommandResponse{
			Success: true, Message: summary, Action: "price_compare",
			Result:   map[string]any{"query": query, "site": site, "summary": summary, "results": results, "total": len(results)},
			Duration: dur,
		})

	case "video_search":
		var query, platform string
		maxItems := 8
		if intent.Params != nil {
			query, _ = intent.Params["query"].(string)
			platform, _ = intent.Params["platform"].(string)
			if v, ok := intent.Params["max_items"].(float64); ok {
				maxItems = int(v)
			}
		}
		if query == "" {
			query = req.Message
		}
		llmMu.RLock()
		videoTKey := llmTavilyKey
		llmMu.RUnlock()
		isTikTok := platform == "tiktok" ||
			strings.Contains(strings.ToLower(req.Message), "틱톡") ||
			strings.Contains(strings.ToLower(req.Message), "tiktok")
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
			json200(w, CommandResponse{
				Success: true, Message: summary, Action: "video_search",
				Result:   map[string]any{"query": query, "platform": "tiktok", "items": videoItems, "total": len(videoItems)},
				Duration: dur,
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
			json200(w, CommandResponse{
				Success: true, Message: summary, Action: "video_search",
				Result:   map[string]any{"query": query, "platform": "youtube", "items": videoItems, "total": len(videoItems)},
				Duration: dur,
			})
		}

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
		// Reflection Loop: /api/workflow/run으로 내부 위임
		wfReqBody, _ := json.Marshal(map[string]any{"goal": goal, "use_reflection": true})
		wfResp, wfErr := (&http.Client{Timeout: 120 * time.Second}).Post(
			"http://127.0.0.1:17891/api/workflow/run", "application/json",
			bytes.NewReader(wfReqBody),
		)
		if wfErr == nil && wfResp != nil {
			var wfResult map[string]any
			json.NewDecoder(wfResp.Body).Decode(&wfResult)
			wfResp.Body.Close()
			summary, _ := wfResult["summary"].(string)
			if summary == "" {
				summary = fmt.Sprintf("'%s' 워크플로우 완료", goal)
			}
			json200(w, CommandResponse{
				Success:  true,
				Message:  summary,
				Action:   "workflow_plan",
				Result:   wfResult,
				Duration: dur,
			})
			return
		}
		// fallback: LLM 계획만 반환
		wMsgs := []groqMsg{
			{Role: "system", Content: "당신은 자비스 AI입니다. 주어진 목표를 단계별로 실행 완료 보고 형식으로 작성하세요."},
			{Role: "user", Content: "목표: " + goal},
		}
		plan, _, _ := callGroq(gKey, groqChatModel, wMsgs, 800, false)
		json200(w, CommandResponse{
			Success:  true,
			Message:  plan,
			Action:   "workflow_plan",
			Duration: dur,
		})

	case "trip_plan":
		destination, _ := intent.Params["destination"].(string)
		date, _ := intent.Params["date"].(string)
		purpose, _ := intent.Params["purpose"].(string)
		if destination == "" {
			destination = req.Message
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

		prompt := fmt.Sprintf(`%s %s 출장/여행 준비 사항을 다음 정보를 바탕으로 한국어로 깔끔하게 정리해줘.

%s

체크리스트 형식으로 작성해줘:
1. 날씨 및 준비물
2. 항공권 정보
3. 숙소 추천
4. 현지 맛집
5. 환율 및 예산
6. 기타 준비 사항`, destination, date, strings.Join(tripSections, "\n\n"))

		result, _, _ := callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 1000, false)

		// 파일 저장
		home, _ := os.UserHomeDir()
		fname := fmt.Sprintf("trip_%s_%s.md", strings.ReplaceAll(destination, " ", "_"), date)
		fpath := filepath.Join(home, "Desktop", fname)
		os.WriteFile(fpath, []byte(fmt.Sprintf("# %s %s %s 준비\n\n%s", purpose, destination, date, result)), 0644)

		json200(w, CommandResponse{
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
			Duration: dur,
		})

	case "multi_action":
		subAction, _ := intent.Params["sub_action"].(string)
		query, _ := intent.Params["query"].(string)
		site, _ := intent.Params["site"].(string)
		platform, _ := intent.Params["platform"].(string)
		fmtStr, _ := intent.Params["format"].(string)
		maxItemsF, _ := intent.Params["max_items"].(float64)
		maxItems := int(maxItemsF)
		if maxItems == 0 {
			maxItems = 8
		}
		if query == "" {
			query = req.Message
		}
		outputFmt := outputFormat(fmtStr)

		llmMu.RLock()
		tKey := llmTavilyKey
		llmMu.RUnlock()

		var collectedItems []map[string]string
		var actionSummary string

		switch subAction {
		case "price_compare":
			if tKey != "" {
				if site != "" {
					if tr, ok := tavilySearchDomain(tKey, query, maxItems, site); ok {
						collectedItems = tr.Items
					}
				}
				if len(collectedItems) == 0 {
					if tr, ok := tavilySearch(tKey, query, maxItems); ok {
						collectedItems = tr.Items
					}
				}
			}
			siteName := site
			if siteName == "" {
				siteName = "쇼핑몰"
			}
			actionSummary = fmt.Sprintf("%s에서 \"%s\" 상품 %d개 검색 결과", siteName, query, len(collectedItems))

		case "video_search":
			targetDomain := "youtube.com"
			if platform == "tiktok" {
				targetDomain = "tiktok.com"
			}
			if tKey != "" {
				if tr, ok := tavilySearchDomain(tKey, query, maxItems, targetDomain); ok {
					collectedItems = tr.Items
				}
				if len(collectedItems) == 0 {
					fallbackQ := query + " " + targetDomain
					if tr, ok := tavilySearch(tKey, fallbackQ, maxItems); ok {
						collectedItems = tr.Items
					}
				}
			}
			pName := "YouTube"
			if platform == "tiktok" {
				pName = "TikTok"
			}
			actionSummary = fmt.Sprintf("%s에서 \"%s\" 영상 %d개 검색 결과", pName, query, len(collectedItems))

		case "doc_compare":
			// 두 대상 비교 - Tavily 검색 후 LLM이 비교표 생성
			llmMu.RLock()
			gKey := llmPerplexityKey
			llmMu.RUnlock()
			var compareText string
			if tKey != "" {
				if tr, ok := tavilySearch(tKey, query, maxItems); ok {
					contextText := tr.Summary
					prompt := fmt.Sprintf(`다음 정보를 바탕으로 "%s"를 항목별로 비교 정리해줘.
비교표 형식으로 깔끔하게 한국어로 작성해줘.

참고 자료:
%s`, query, contextText)
					if gKey != "" {
						compareText, _, _ = callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 1200, false)
					}
					collectedItems = tr.Items
				}
			}
			if compareText == "" {
				compareText = fmt.Sprintf("\"%s\" 비교 결과를 생성했습니다.", query)
			}
			actionSummary = compareText

		case "summarize":
			// 주제 요약 - Tavily 검색 후 LLM 요약
			llmMu.RLock()
			gKey := llmPerplexityKey
			llmMu.RUnlock()
			var summaryText string
			if tKey != "" {
				if tr, ok := tavilySearch(tKey, query, maxItems); ok {
					prompt := fmt.Sprintf(`다음 정보를 바탕으로 "%s"에 대해 한국어로 명확하게 요약 정리해줘.
핵심 내용만 항목별로 구조화해서 작성해줘.

참고 자료:
%s`, query, tr.Summary)
					if gKey != "" {
						summaryText, _, _ = callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 1200, false)
					}
					collectedItems = tr.Items
				}
			}
			if summaryText == "" {
				summaryText = fmt.Sprintf("\"%s\" 요약을 완료했습니다.", query)
			}
			actionSummary = summaryText

		default:
			// 일반 web_search
			if tKey != "" {
				if tr, ok := tavilySearch(tKey, query, maxItems); ok {
					collectedItems = tr.Items
				}
			}
			actionSummary = fmt.Sprintf("\"%s\" 검색 결과 %d개", query, len(collectedItems))
		}

		// format 기본값: markdown
		if outputFmt == "" {
			outputFmt = outMarkdown
		}

		// 파일 저장
		title := query
		if len([]rune(title)) > 20 {
			title = string([]rune(title)[:20])
		}
		filePath, saveErr := saveResultToFile(outputFmt, title, collectedItems, actionSummary)
		var fileMsg string
		if saveErr != nil {
			fileMsg = fmt.Sprintf("⚠️ 파일 저장 실패: %s", saveErr.Error())
		} else {
			ext := strings.ToUpper(string(outputFmt))
			if outputFmt == outPDF || outputFmt == outWord {
				ext = "MD (Mac 호환)"
			}
			fileMsg = fmt.Sprintf("📄 %s 파일로 저장됨: %s", ext, filePath)
		}

		resultItems := make([]map[string]string, 0, len(collectedItems))
		for _, it := range collectedItems {
			resultItems = append(resultItems, map[string]string{
				"site": site, "name": it["title"], "price": it["price"], "link": it["url"],
			})
		}

		json200(w, CommandResponse{
			Success:  true,
			Message:  actionSummary + "\n" + fileMsg,
			Action:   "multi_action",
			Result: map[string]any{
				"query":     query,
				"summary":   actionSummary,
				"results":   resultItems,
				"total":     len(resultItems),
				"file_path": filePath,
				"file_msg":  fileMsg,
				"format":    fmtStr,
				"sub_action": subAction,
			},
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
	Query       string              `json:"query"`
	Site        string              `json:"site"`
	Summary     string              `json:"summary"`
	Items       []map[string]string `json:"items,omitempty"`
	PreviewType string              `json:"preview_type,omitempty"`
}

func runWebSearchMac(apiKey, query, site string, maxItems int) webSearchResult {
	siteLabel := site
	if siteLabel == "" || siteLabel == "auto" {
		siteLabel = "웹"
	}

	cat := detectCategory(query)
	previewType := categoryPreviewType(cat)

	// 병렬 검색: Tavily + 브라우저 동시 실행
	result := parallelWebSearch(query, maxItems)

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

