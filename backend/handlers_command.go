//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ── 멀티 액션: 출력 포맷 (Windows 빌드용) ────────────────────────
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

	buildMD := func() string {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("# %s\n\n", title))
		sb.WriteString(fmt.Sprintf("*생성: %s*\n\n", time.Now().Format("2006-01-02 15:04:05")))
		if summary != "" {
			sb.WriteString("## 요약\n\n" + summary + "\n\n")
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
		return sb.String()
	}

	switch format {
	case outMarkdown:
		path := filepath.Join(home, "Desktop", fmt.Sprintf("nexus_%s_%s.md", safeName, ts))
		return path, os.WriteFile(path, []byte(buildMD()), 0644)

	case outTXT:
		path := filepath.Join(home, "Desktop", fmt.Sprintf("nexus_%s_%s.txt", safeName, ts))
		var sb strings.Builder
		sb.WriteString(title + "\n" + strings.Repeat("=", 40) + "\n")
		sb.WriteString("생성: " + time.Now().Format("2006-01-02 15:04:05") + "\n\n")
		if summary != "" { sb.WriteString("[ 요약 ]\n" + summary + "\n\n") }
		if len(items) > 0 {
			sb.WriteString("[ 항목 ]\n")
			for i, it := range items {
				name := it["title"]; if name == "" { name = it["name"] }
				url := it["url"]; if url == "" { url = it["link"] }
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
		path := filepath.Join(home, "Desktop", fmt.Sprintf("nexus_%s_%s.csv", safeName, ts))
		var sb strings.Builder
		sb.WriteString("번호,제목/상품명,가격,링크\n")
		for i, it := range items {
			name := it["title"]; if name == "" { name = it["name"] }
			url := it["url"]; if url == "" { url = it["link"] }
			price := it["price"]
			sb.WriteString(fmt.Sprintf("%d,\"%s\",\"%s\",\"%s\"\n", i+1,
				strings.ReplaceAll(name, `"`, `""`),
				strings.ReplaceAll(price, `"`, `""`),
				url))
		}
		return path, os.WriteFile(path, []byte(sb.String()), 0644)

	case outPDF:
		// Windows: HTML → PDF via wkhtmltopdf (있으면), 없으면 MD로 fallback
		mdContent := buildMD()
		mdPath := filepath.Join(home, "Desktop", fmt.Sprintf("nexus_%s_%s.md", safeName, ts))
		pdfPath := filepath.Join(home, "Desktop", fmt.Sprintf("nexus_%s_%s.pdf", safeName, ts))
		_ = os.WriteFile(mdPath, []byte(mdContent), 0644)
		// wkhtmltopdf 시도
		if err := exec.Command("wkhtmltopdf", mdPath, pdfPath).Run(); err == nil {
			_ = os.Remove(mdPath)
			return pdfPath, nil
		}
		return mdPath, nil // fallback to MD

	case outWord:
		// Windows: pandoc 있으면 docx, 없으면 MD fallback
		mdContent := buildMD()
		mdPath := filepath.Join(home, "Desktop", fmt.Sprintf("nexus_%s_%s.md", safeName, ts))
		docxPath := filepath.Join(home, "Desktop", fmt.Sprintf("nexus_%s_%s.docx", safeName, ts))
		_ = os.WriteFile(mdPath, []byte(mdContent), 0644)
		if err := exec.Command("pandoc", mdPath, "-o", docxPath).Run(); err == nil {
			_ = os.Remove(mdPath)
			return docxPath, nil
		}
		return mdPath, nil
	}
	return "", fmt.Errorf("지원하지 않는 형식")
}

// ══════════════════════════════════════════════════════════════════
//  POST /api/command
//  사용자가 어떤 자연어로 말해도 Nexus가 알아서 처리합니다.
//  LLM이 의도를 파악 → 올바른 백엔드 함수 호출 → 결과 반환
// ══════════════════════════════════════════════════════════════════

type ConvHistoryMsg struct {
	Role    string `json:"role"`    // "user" | "assistant"
	Content string `json:"content"`
}

type CommandRequest struct {
	Message         string            `json:"message"`
	Context         string            `json:"context"`
	History         []ConvHistoryMsg  `json:"history"`
	// 멀티턴: 이전 clarify에서 넘어온 컨텍스트
	PendingIntent   string            `json:"pending_intent"`
	PendingParams   map[string]any    `json:"pending_params"`
	PendingQuestion string            `json:"pending_question"`
}

type CommandResponse struct {
	Success         bool           `json:"success"`
	Message         string         `json:"message"`
	Action          string         `json:"action"`
	Result          any            `json:"result"`
	Duration        string         `json:"duration"`
	// clarify 액션 전용
	NeedsClarify    bool           `json:"needs_clarify,omitempty"`
	ClarifyQuestion string         `json:"clarify_question,omitempty"`
	PendingIntent   string         `json:"pending_intent,omitempty"`
	PendingParams   map[string]any `json:"pending_params,omitempty"`
}

// Nexus가 할 수 있는 모든 일을 LLM에게 알려줍니다.
// 사용자가 어떤 말을 해도 이 중 가장 적합한 action을 고릅니다.
const nexusSystemPrompt = `당신은 Nexus AI 비서입니다. 사용자 명령을 분석하여 아래 액션 중 하나를 반드시 선택하세요.

⚠️ 규칙: 반드시 JSON만 출력하세요. 설명 금지.
형식: {"action":"액션명","params":{...}}

━━━ 액션 목록 & 트리거 키워드 ━━━

"web_search" → 쇼핑/최저가/뉴스/가격비교/쿠팡/네이버/테무/검색/버스/기차/지하철/교통/여행/맛집/식당/날씨/환율/주가/병원/약국/영화/공연/예약/길찾기/요금/시간표/노선/운임/경로/이동
  예) "쿠팡에서 에어팟 찾아줘" "네이버 AI 뉴스 10개" "삼성 노트북 최저가"
  예) "부산터미널에서 인천 청라 가는 버스" "서울역 부산 KTX 시간표" "강남 맛집 추천"
  예) "오늘 달러 환율" "삼성전자 주가" "가까운 응급실 위치" "CGV 영화 시간표"
  params: {"query":"검색어","site":"coupang|naver|temu|danawa|gmarket|11st|google|auto","max_items":5,"output":"pdf|excel|text"}
  ⚠️ site 값은 반드시 위 목록 중 하나만 사용 (youtube.com 형식 금지, 축약형 사용)
  ⚠️ 교통/맛집/장소/실시간 정보는 무조건 web_search 사용 (chat 사용 금지)
  ⚠️ 유튜브/틱톡/YouTube/TikTok 관련 쿼리는 반드시 video_search 사용 (web_search 절대 금지)

"video_search" → 유튜브/틱톡/YouTube/TikTok/영상/동영상/쇼츠/릴스 검색
  예) "유튜브에서 요리 영상 찾아줘" "틱톡에서 댄스 영상" "틱톡 유행하는 노래" "유튜브 강의 찾아줘"
  예) "YouTube에서 김치찌개 만드는 법" "TikTok viral 영상" "틱톡 트렌드"
  params: {"query":"검색어","platform":"youtube|tiktok","max_items":8}
  ⚠️ 유튜브/틱톡이 언급된 모든 쿼리는 무조건 이 액션 사용

"file_search" → 파일찾기/문서검색/계약서/보고서/~보낸 파일/~관련 파일
  예) "박부장이 보낸 계약서 찾아줘" "지난달 여행 사진 찾아줘" "엑셀 파일 찾아줘"
  params: {"query":"검색어","folder":"경로(없으면 홈)","max_results":10}

"doc_compare" → 두 문서 비교/변경사항/버전 비교/차이점
  예) "계약서 v1과 v2 비교해줘" "이 두 파일 다른 점 알려줘"
  params: {"file_a":"경로A","file_b":"경로B"}

"doc_summary" → 문서요약/보고서 핵심/파일 내용 분석/요약해줘
  예) "이 PDF 요약해줘" "계약서 핵심만 알려줘"
  params: {"file_path":"경로"}

"organize_folder" → 폴더정리/파일정리/바탕화면정리/다운로드정리/파일분류
  예) "다운로드 폴더 정리해" "바탕화면 깔끔하게 정리해줘"
  params: {"folder":"Downloads|Desktop|Documents","mode":"type|date|auto"}

"vision" → 화면보기/오류분석/지금 화면/창 내용/오류해결/화면 뭐라고 써있어
  예) "지금 화면에 뭐라고 써있어?" "이 오류 어떻게 고쳐?" "화면 분석해줘"
  params: {"question":"질문"}

"scan" → PC상태/건강점검/속도진단/PC 문제확인/진단해줘/PC 어때
  예) "PC 상태 알려줘" "PC 건강 체크해줘" "PC 진단해줘" "지금 PC 어때"
  params: {}

"clean" → 임시파일정리/디스크정리/느려졌어/빠르게해줘/용량확보/청소
  예) "PC 정리해줘" "임시 파일 지워줘" "디스크 청소해줘" "PC가 느려"
  params: {}

"security_scan" → 해킹탐지/악성코드/보안점검/원격접속/바이러스/수상한프로세스/침입탐지
  예) "해킹 탐지해" "해킹당했나 확인해줘" "바이러스 있어?" "악성코드 스캔해"
  예) "보안 점검해줘" "원격 접속 탐지해" "이상한 프로세스 있어?" "해킹 확인해"
  params: {}

"stats" → CPU온도/메모리/디스크용량/네트워크속도/현재 리소스 현황
  예) "CPU 온도 알려줘" "메모리 얼마나 써?" "지금 네트워크 속도 어때"
  params: {}

"focus_mode" → 집중모드/방해금지/알림차단/집중하고싶어/자동모드
  예) "집중 모드 켜줘" "방해 금지 설정해줘" "25분 집중 모드"
  params: {"enable":true}

"journal" → 업무일지/일지작성/오늘뭐했어/작업기록/일일리포트/오늘 정리
  예) "오늘 업무 일지 써줘" "오늘 업무 일지 작성해줘" "오늘 뭐 했어?" "일지 만들어줘"
  예) "오늘 작업 기록 정리해줘" "일일 리포트 만들어" "오늘 업무 정리해줘"
  params: {}

"health_report" → PC건강리포트/진단리포트/점검결과/리포트PDF/건강점수
  예) "PC 건강 리포트 만들어줘" "진단 리포트 PDF로 저장해줘"
  params: {}

"scheduler" → 매일/매주/내일/특정시간에/자동실행/반복/스케줄/예약
  예) "매주 월요일 9시에 보고서 정리해줘" "내일 아침 8시에 메일 요약해줘"
  params: {"command":"사용자 원문 전체"}

"launch_app" → 앱실행/프로그램열어/크롬열어/워드열어/카카오톡/실행해줘
  예) "크롬 열어줘" "워드 실행해줘" "카카오톡 켜줘"
  params: {"app_name":"앱이름"}

"system_control" → 볼륨/밝기/와이파이/절전/재시작/종료/음소거/꺼줘
  예) "볼륨 낮춰" "밝기 올려줘" "와이파이 꺼줘" "절전 모드로" "PC 재시작해"
  params: {"control":"volume|brightness|wifi|sleep|restart|shutdown|mute","value":50}

"excel_save" → 엑셀저장/표로정리/xlsx/스프레드시트
  예) "이 데이터 엑셀로 저장해줘" "표로 정리해줘"
  params: {"title":"제목","data":[["헤더1","헤더2"],["값1","값2"]]}

"note" → 메모/기록/적어둬/저장해줘(단순텍스트)
  예) "이거 메모해줘" "기록해줘" "적어둬"
  params: {"content":"내용"}

"chat" → 오직 인사/잡담/AI 자체 질문/일반 지식(역사·과학·IT 개념) 에만 사용
  예) "안녕" "고마워" "넌 누구야" "파이썬이 뭐야" "2차세계대전은 언제야"
  ⚠️ 실시간/외부 데이터가 필요한 모든 질문은 chat 금지 → web_search 사용

━━━ 중요 판단 규칙 ━━━
1. "해킹" 키워드 → 무조건 security_scan
2. "업무 일지", "일지 써", "일지 작성" → 무조건 journal
3. "PC 상태", "PC 어때", "진단" → scan
4. 시간/날짜 + 자동화 키워드 → scheduler
5. 의심스러우면 chat 대신 가장 가까운 액션을 선택하세요.
6. 교통(버스/기차/지하철/KTX/고속버스/시외버스/항공편/노선/시간표/요금) → 무조건 web_search
7. 맛집/식당/카페/병원/약국/마트/장소 → 무조건 web_search
8. 환율/주가/코인/암호화폐/날씨(특정 도시)/영화/공연 → 무조건 web_search
9. "어떻게 가?" "얼마야?" "몇 시에?" 같은 실시간 정보 → 무조건 web_search
10. chat은 오직 인사/잡담/AI 자체에 대한 질문만 사용

"clarify" → 의도는 파악됐지만 핵심 정보가 없어서 실행 불가능할 때만 사용
  예) "날씨 어때?" → 지역 모름
  예) "파일 찾아줘" → 무슨 파일인지 모름  
  예) "뉴스 알려줘" → 어떤 주제인지 모름
  params: {"question":"친절한 추가 질문","missing":"없는 정보","intent":"원래 액션명","collected":{...지금까지 파악된 파라미터...}}

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

동일 이름 업체·상품 여러 개 검색 결과 → 목록 나열 후 "어느 것을 원하시나요?" 물어볼 것.
`

// clarify 해소 시 사용하는 별도 시스템 프롬프트
const nexusClarifyResolvePrompt = `당신은 Nexus AI 비서입니다. 사용자가 이전 질문에 대한 추가 정보를 제공했습니다.

이전 컨텍스트와 새 정보를 합쳐서 완전한 액션을 결정하세요.

반드시 JSON만 출력하세요:
{"action":"액션명","params":{...완전한 파라미터...}}

이전에 파악한 액션: %s
이전에 수집한 파라미터: %s
이전 질문: %s
사용자 새 답변: %s

위 정보를 모두 합쳐서 실행 가능한 완전한 액션으로 만드세요.
예시: 이전 액션=web_search, 이전 파라미터={"site":"naver"}, 사용자 답변="서울 날씨"
→ {"action":"web_search","params":{"query":"서울 날씨","site":"naver"}}
`

func handleCommand(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	var req CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Message) == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "message 필요"})
		return
	}

	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()
	if gKey == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "Perplexity API 키 미설정. 설정 → API 키에서 Perplexity 키를 입력해주세요."})
		return
	}

	var intentAction string
	var intentParams map[string]any

	// ── 멀티턴: 이전 clarify 컨텍스트가 있으면 해소 프롬프트 사용 ──
	if req.PendingIntent != "" {
		prevParamsJSON, _ := json.Marshal(req.PendingParams)
		resolvePrompt := fmt.Sprintf(nexusClarifyResolvePrompt,
			req.PendingIntent,
			string(prevParamsJSON),
			req.PendingQuestion,
			req.Message,
		)
		raw, _, err := callGroq(gKey, groqFastModel, []groqMsg{
			{Role: "user", Content: resolvePrompt},
		}, 256, true)
		if err != nil || raw == "" {
			// 해소 실패 → 사용자 답변을 pending 파라미터에 병합해서 직접 실행
			intentAction = req.PendingIntent
			intentParams = req.PendingParams
			if intentParams == nil {
				intentParams = map[string]any{}
			}
			// 가장 빈번한 missing 필드에 사용자 답변 적용
			missing := ""
			if req.PendingParams != nil {
				if m, ok := req.PendingParams["__missing__"]; ok {
					missing = fmt.Sprintf("%v", m)
				}
			}
			if missing != "" {
				intentParams[missing] = req.Message
			} else {
				intentParams["query"] = req.Message
			}
		} else {
			var resolved struct {
				Action string         `json:"action"`
				Params map[string]any `json:"params"`
			}
			if err := json.Unmarshal([]byte(raw), &resolved); err == nil && resolved.Action != "" {
				intentAction = resolved.Action
				intentParams = resolved.Params
			} else {
				intentAction = req.PendingIntent
				intentParams = req.PendingParams
				if intentParams == nil {
					intentParams = map[string]any{}
				}
				intentParams["query"] = req.Message
			}
		}
	} else {
		// ── 키워드 사전 라우팅 (LLM이 무시하는 액션들) ────────────
		msgLower := strings.ToLower(req.Message)
		videoKeywords := []string{"찾", "검색", "영상", "보여", "추천", "viral", "바이럴", "트렌드"}
		isTikTokReq := strings.Contains(msgLower, "틱톡") || strings.Contains(msgLower, "tiktok")
		isYouTubeReq := strings.Contains(msgLower, "유튜브") || strings.Contains(msgLower, "youtube")
		hasVideoVerb := false
		for _, kw := range videoKeywords {
			if strings.Contains(msgLower, kw) {
				hasVideoVerb = true
				break
			}
		}

		// ── 쇼핑/도메인 사전 라우팅 ─────────────────────────────
		shoppingSites := map[string]string{
			// 쇼핑몰
			"태무": "temu.com", "테무": "temu.com", "temu": "temu.com",
			"쿠팡": "coupang.com", "coupang": "coupang.com",
			"네이버쇼핑": "shopping.naver.com", "네이버 쇼핑": "shopping.naver.com",
			"11번가": "11st.co.kr",
			"지마켓": "gmarket.co.kr", "gmarket": "gmarket.co.kr",
			"옥션": "auction.co.kr",
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
			// 중고차
			"헤이딜러": "heydealer.com", "heydealer": "heydealer.com",
			"엔카": "encar.com", "encar": "encar.com",
			"kb차차차": "kbchachacha.com", "차차차": "kbchachacha.com",
			"오토피디아": "autopedia.co.kr",
			"보배드림": "bobaedream.co.kr",
			"중고차": "encar.com",
			// 중고거래
			"당근": "daangn.com", "당근마켓": "daangn.com", "daangn": "daangn.com",
			"번개장터": "bunjang.co.kr", "번개": "bunjang.co.kr",
			"중고나라": "joongna.com",
			// 부동산
			"직방": "zigbang.com",
			"다방": "dabangapp.com",
			"호갱노노": "hogangnono.com",
			"네이버부동산": "land.naver.com", "네이버 부동산": "land.naver.com",
			"부동산114": "r114.com",
			// 음식/배달
			"배민": "baemin.com", "배달의민족": "baemin.com",
			"요기요": "yogiyo.co.kr",
			"쿠팡이츠": "coupangeats.com",
			// 여행/숙박
			"야놀자": "yanolja.com",
			"여기어때": "goodchoice.kr",
			"에어비앤비": "airbnb.co.kr", "airbnb": "airbnb.com",
			// 전자기기 가격비교
			"다나와": "danawa.com",
			"에누리": "enuri.com",
			"컴퓨존": "compuzone.co.kr",
		}
		priceVerbs := []string{"찾아", "검색", "최저가", "얼마", "가격", "사고 싶", "구매", "살 수", "추천", "알려줘", "보여줘", "있어"}
		hasPriceVerb := false
		for _, kw := range priceVerbs {
			if strings.Contains(msgLower, kw) {
				hasPriceVerb = true
				break
			}
		}
		detectedShopSite := ""
		for keyword, domain := range shoppingSites {
			if strings.Contains(msgLower, strings.ToLower(keyword)) {
				detectedShopSite = domain
				break
			}
		}

		outFmt := detectOutputFormat(req.Message)
		isMultiAct := outFmt != outNone && hasFileSaveVerb(req.Message)

		if detectedShopSite != "" && hasPriceVerb {
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
			if isMultiAct {
				intentAction = "multi_action"
				intentParams = map[string]any{"sub_action": "price_compare", "query": q, "site": detectedShopSite, "max_items": 8, "format": string(outFmt)}
			} else {
				intentAction = "price_compare"
				intentParams = map[string]any{"query": q, "site": detectedShopSite, "max_items": 8}
			}
		} else if isTikTokReq && hasVideoVerb {
			query := req.Message
			for _, rm := range []string{"틱톡에서", "틱톡", "tiktok", "찾아줘", "검색해줘", "보여줘", "영상", "추천해줘"} {
				query = strings.ReplaceAll(query, rm, "")
			}
			query = strings.TrimSpace(query)
			if query == "" {
				query = "바이럴 트렌드"
			}
			intentAction = "video_search"
			intentParams = map[string]any{"query": query, "platform": "tiktok", "max_items": 8}
		} else if isYouTubeReq && hasVideoVerb {
			query := req.Message
			for _, rm := range []string{"유튜브에서", "유튜브", "youtube", "찾아줘", "검색해줘", "보여줘", "영상", "추천해줘"} {
				query = strings.ReplaceAll(query, rm, "")
			}
			query = strings.TrimSpace(query)
			if query == "" {
				query = "인기 영상"
			}
			intentAction = "video_search"
			intentParams = map[string]any{"query": query, "platform": "youtube", "max_items": 8}
		} else {
			// ── 일반 모드: LLM 의도 파악 (대화 이력 포함) ────────────
			intentMsgs := []groqMsg{{Role: "system", Content: nexusSystemPrompt}}
			for _, h := range req.History {
				if len(h.Content) == 0 {
					continue
				}
				role := "user"
				if h.Role == "assistant" {
					role = "assistant"
				}
				content := h.Content
				if len([]rune(content)) > 200 {
					content = string([]rune(content)[:200]) + "..."
				}
				intentMsgs = append(intentMsgs, groqMsg{Role: role, Content: content})
			}
			intentMsgs = append(intentMsgs, groqMsg{Role: "user", Content: req.Message})

			raw, _, err := callGroq(gKey, groqFastModel, intentMsgs, 512, true)
			if err != nil {
				raw = `{"action":"chat","params":{}}`
			}
			var intent struct {
				Action string         `json:"action"`
				Params map[string]any `json:"params"`
			}
			if jsonErr := json.Unmarshal([]byte(raw), &intent); jsonErr != nil || intent.Action == "" {
				intent.Action = "chat"
				intent.Params = map[string]any{}
			}
			if intent.Params == nil {
				intent.Params = map[string]any{}
			}
			intentAction = intent.Action
			intentParams = intent.Params
		}
	}

	// ── clarify 액션: 실행 없이 질문 반환 ────────────────────
	if intentAction == "clarify" {
		question, _ := intentParams["question"].(string)
		missing, _ := intentParams["missing"].(string)
		pendingIntent, _ := intentParams["intent"].(string)
		collected, _ := intentParams["collected"].(map[string]any)
		if collected == nil {
			collected = map[string]any{}
		}
		if missing != "" {
			collected["__missing__"] = missing
		}
		if question == "" {
			question = "조금 더 자세히 알려주시겠어요?"
		}
		if pendingIntent == "" {
			pendingIntent = "chat"
		}
		json200(w, CommandResponse{
			Success:         true,
			Message:         question,
			Action:          "clarify",
			NeedsClarify:    true,
			ClarifyQuestion: question,
			PendingIntent:   pendingIntent,
			PendingParams:   collected,
			Duration:        time.Since(start).String(),
		})
		return
	}

	// ── 액션 실행 ────────────────────────────────────────────
	result, msg := dispatchAction(intentAction, intentParams, req.Message, gKey, req.History)

	saveAgentMemory(AgentMemoryEntry{
		ID:        fmt.Sprintf("cmd_%d", time.Now().Unix()),
		Timestamp: time.Now().Format(time.RFC3339),
		Type:      "command",
		Command:   req.Message,
		Result:    fmt.Sprintf("action=%s msg=%s", intentAction, truncateStr(msg, 100)),
		Tags:      []string{intentAction},
		Success:   true,
	})

	json200(w, CommandResponse{
		Success:  true,
		Message:  msg,
		Action:   intentAction,
		Result:   result,
		Duration: time.Since(start).String(),
	})
}

// ══════════════════════════════════════════════════════════════════
//  dispatchAction: 액션 → 실제 함수 실행
// ══════════════════════════════════════════════════════════════════
func dispatchAction(action string, params map[string]any, original, gKey string, history []ConvHistoryMsg) (result any, message string) {
	str := func(key string) string {
		if v, ok := params[key]; ok {
			return fmt.Sprintf("%v", v)
		}
		return ""
	}
	boolVal := func(key string, def bool) bool {
		if v, ok := params[key]; ok {
			if b, ok := v.(bool); ok {
				return b
			}
		}
		return def
	}
	intVal := func(key string, def int) int {
		if v, ok := params[key]; ok {
			if f, ok := v.(float64); ok {
				return int(f)
			}
		}
		return def
	}

	switch action {

	// ── 일반 대화 ───────────────────────────────────────────
	case "chat":
		// 실시간 정보 카테고리면 web_search로 리다이렉트
		// (이전 대화에서 이미 카테고리가 정해진 경우도 포함)
		resolvedQuery := resolveWithHistory(original, history)
		cat := detectCategory(resolvedQuery)
		realtime := cat == catTransit || cat == catFood || cat == catShopping ||
			cat == catFinance || cat == catWeather || cat == catNews ||
			cat == catMedical || cat == catEntertainment || cat == catTravel ||
			cat == catRealEstate
		if realtime {
			pr := parallelWebSearch(resolvedQuery, 5)
			items := pr.Items
			if len(items) == 0 {
				items = categoryFallbackSites(resolvedQuery, cat)
			}
			msg := pr.Summary
			if msg == "" {
				msg = buildNoResultMessage(resolvedQuery, cat, "")
			}
			return map[string]any{"query": resolvedQuery, "summary": msg, "items": items}, msg
		}
		// 순수 잡담/지식 질문 — 이전 대화를 컨텍스트로 포함
		chatSys := `당신은 Nexus AI 한국어 비서입니다.

[규칙]
1. 자연스러운 한국어로 2~4문장 답변
2. 마크다운 헤더(##) 금지
3. 인사, 잡담, 일반 지식(역사/과학/IT 개념/요리법 등) 질문에 직접 답변
4. 이전 대화 컨텍스트를 반드시 참고해서 연결된 답변을 해줘
5. "이거", "그거", "그때", "아까" 같은 지시어는 이전 대화를 보고 해석해`
		chatMsgs := []groqMsg{{Role: "system", Content: chatSys}}
		for _, h := range history {
			role := "user"
			if h.Role == "assistant" {
				role = "assistant"
			}
			content := h.Content
			if len([]rune(content)) > 300 {
				content = string([]rune(content)[:300]) + "..."
			}
			chatMsgs = append(chatMsgs, groqMsg{Role: role, Content: content})
		}
		chatMsgs = append(chatMsgs, groqMsg{Role: "user", Content: original})
		ans, _, _ := callGroq(gKey, groqChatModel, chatMsgs, 600, false)
		return map[string]any{"reply": ans}, ans

	// ── 날씨 ─────────────────────────────────────────────
	case "weather":
		city := str("city")
		if city == "" {
			city = "서울"
		}
		text := fetchWeatherText(city, gKey)
		return map[string]any{"reply": text}, text

	// ── 웹 검색 & 쇼핑 ──────────────────────────────────────
	case "web_search":
		query := str("query")
		if query == "" {
			query = original
		}
		// 이전 대화로 모호한 쿼리 보완
		query = resolveWithHistory(query, history)
		site := str("site")
		output := str("output")
		maxItems := intVal("max_items", 5)
		// output 없는 단순 검색 쿼리 → 병렬 검색으로 빠른 응답
		if output == "" {
			cat := detectCategory(query)
			pr := parallelWebSearch(query, maxItems)
			items := pr.Items
			if len(items) == 0 {
				items = buildFallbackURLs(query, site)
			}
			if len(items) == 0 {
				items = categoryFallbackSites(query, cat)
			}
			result := map[string]any{"query": query, "site": site, "summary": pr.Summary, "items": items}
			msg := pr.Summary
			if msg == "" {
				msg = buildNoResultMessage(query, cat, "")
			}
			return result, msg
		}
		return runWebSearch(query, site, output, maxItems, gKey)

	// ── 영상 검색 (YouTube / TikTok) ─────────────────────────
	case "video_search":
		query := str("query")
		if query == "" {
			query = original
		}
		platform := strings.ToLower(str("platform"))
		maxItems := intVal("max_items", 8)

		// tKey를 dispatchAction 스코프 내에서 직접 조회
		llmMu.RLock()
		videoTKey := llmTavilyKey
		llmMu.RUnlock()

		isTikTok := platform == "tiktok" ||
			strings.Contains(strings.ToLower(original), "틱톡") ||
			strings.Contains(strings.ToLower(original), "tiktok")

		if isTikTok {
			// TikTok: chromedp 봇차단으로 직접 접근 불가
			// → Tavily/Perplexity로 site:tiktok.com 검색 후 tiktok.com URL만 필터링
			tiktokQuery := "site:tiktok.com " + query
			var items []map[string]string

			if videoTKey != "" {
				if tr, ok := tavilySearch(videoTKey, tiktokQuery, maxItems); ok {
					for _, it := range tr.Items {
						if strings.Contains(it["url"], "tiktok.com") {
							items = append(items, it)
						}
					}
				}
			}
			// Tavily 결과가 없으면 LLM으로 보완
			if len(items) == 0 && gKey != "" {
				pplxPrompt := fmt.Sprintf(`TikTok에서 "%s" 관련 실제 영상 링크를 최대 %d개 찾아줘. 반드시 tiktok.com URL만 포함. JSON 배열로만 출력: [{"title":"...", "url":"https://tiktok.com/..."}]`, query, maxItems)
				raw, _, _ := callGroq(gKey, groqChatModel, []groqMsg{{Role: "user", Content: pplxPrompt}}, 512, true)
				var parsed []map[string]string
				if json.Unmarshal([]byte(raw), &parsed) == nil {
					for _, it := range parsed {
						if strings.Contains(it["url"], "tiktok.com") {
							items = append(items, it)
						}
					}
				}
			}
			// 최후 fallback: TikTok 검색 페이지 링크 제공
			if len(items) == 0 {
				enc := strings.ReplaceAll(query, " ", "%20")
				items = []map[string]string{
					{"title": fmt.Sprintf("TikTok에서 \"%s\" 검색", query), "url": fmt.Sprintf("https://www.tiktok.com/search?q=%s", enc)},
					{"title": "TikTok 트렌딩", "url": "https://www.tiktok.com/trending"},
				}
			}
			summary := fmt.Sprintf("TikTok에서 \"%s\" 영상 %d개를 찾았어요!", query, len(items))
			return map[string]any{"query": query, "platform": "tiktok", "items": items, "total": len(items), "summary": summary}, summary
		}

		// YouTube: Tavily로 site:youtube.com 검색
		ytQuery := "site:youtube.com " + query
		var ytItems []map[string]string
		if videoTKey != "" {
			if tr, ok := tavilySearch(videoTKey, ytQuery, maxItems); ok {
				for _, it := range tr.Items {
					if strings.Contains(it["url"], "youtube.com/watch") || strings.Contains(it["url"], "youtu.be") {
						ytItems = append(ytItems, it)
					}
				}
			}
		}
		if len(ytItems) == 0 {
			enc := strings.ReplaceAll(query, " ", "%20")
			ytItems = []map[string]string{
				{"title": fmt.Sprintf("YouTube에서 \"%s\" 검색", query), "url": fmt.Sprintf("https://www.youtube.com/results?search_query=%s", enc)},
			}
		}
		ytSummary := fmt.Sprintf("YouTube에서 \"%s\" 영상 %d개를 찾았어요!", query, len(ytItems))
		return map[string]any{"query": query, "platform": "youtube", "items": ytItems, "total": len(ytItems), "summary": ytSummary}, ytSummary

	// ── 가격/쇼핑 검색 ───────────────────────────────────────
	case "price_compare":
		pcQuery := str("query")
		if pcQuery == "" { pcQuery = original }
		pcSite := str("site")
		pcMax := intVal("max_items", 8)
		llmMu.RLock()
		pcTKey := llmTavilyKey
		llmMu.RUnlock()
		searchQ := pcQuery
		if pcSite != "" { searchQ = "site:" + pcSite + " " + pcQuery }
		var priceItems []map[string]string
		if pcTKey != "" {
			if tr, ok := tavilySearch(pcTKey, searchQ, pcMax); ok {
				for _, it := range tr.Items {
					if pcSite == "" || strings.Contains(it["url"], strings.Split(pcSite, ".")[0]) {
						priceItems = append(priceItems, it)
					}
				}
				if len(priceItems) == 0 { priceItems = tr.Items }
			}
		}
		siteName := pcSite; if siteName == "" { siteName = "쇼핑몰" }
		if len(priceItems) == 0 {
			enc := strings.ReplaceAll(pcQuery, " ", "+")
			priceItems = []map[string]string{{"title": pcQuery + " 검색", "url": "https://www." + pcSite + "/search?q=" + enc}}
		}
		results := make([]map[string]string, 0, len(priceItems))
		for _, it := range priceItems {
			results = append(results, map[string]string{"site": siteName, "name": it["title"], "price": "", "link": it["url"]})
		}
		pcSummary := fmt.Sprintf("%s에서 \"%s\" 상품 %d개를 찾았어요!", siteName, pcQuery, len(results))
		return map[string]any{"query": pcQuery, "site": pcSite, "summary": pcSummary, "results": results, "total": len(results)}, pcSummary

	// ── 멀티 액션: 검색 + 파일 저장 ────────────────────────────
	case "multi_action":
		maSubAction := str("sub_action")
		maQuery := str("query"); if maQuery == "" { maQuery = original }
		maSite := str("site")
		maPlatform := str("platform")
		maFmtStr := str("format")
		maMax := intVal("max_items", 8)
		maFmt := outputFormat(maFmtStr)
		llmMu.RLock()
		maTKey := llmTavilyKey
		llmMu.RUnlock()

		var maItems []map[string]string
		var maActionSummary string

		switch maSubAction {
		case "price_compare":
			searchQ := maQuery
			if maSite != "" { searchQ = "site:" + maSite + " " + maQuery }
			if maTKey != "" {
				if tr, ok := tavilySearch(maTKey, searchQ, maMax); ok {
					for _, it := range tr.Items {
						if maSite == "" || strings.Contains(it["url"], strings.Split(maSite, ".")[0]) {
							maItems = append(maItems, map[string]string{"title": it["title"], "url": it["url"], "price": ""})
						}
					}
				}
			}
			sn := maSite; if sn == "" { sn = "쇼핑몰" }
			maActionSummary = fmt.Sprintf("%s에서 \"%s\" 상품 %d개 검색 결과", sn, maQuery, len(maItems))
		case "video_search":
			prefix := "site:youtube.com"
			if maPlatform == "tiktok" { prefix = "site:tiktok.com" }
			if maTKey != "" {
				if tr, ok := tavilySearch(maTKey, prefix+" "+maQuery, maMax); ok {
					maItems = tr.Items
				}
			}
			pn := "YouTube"; if maPlatform == "tiktok" { pn = "TikTok" }
			maActionSummary = fmt.Sprintf("%s에서 \"%s\" 영상 %d개 검색 결과", pn, maQuery, len(maItems))
		default:
			if maTKey != "" {
				if tr, ok := tavilySearch(maTKey, maQuery, maMax); ok { maItems = tr.Items }
			}
			maActionSummary = fmt.Sprintf("\"%s\" 검색 결과 %d개", maQuery, len(maItems))
		}

		maTitle := maQuery
		if len([]rune(maTitle)) > 20 { maTitle = string([]rune(maTitle)[:20]) }
		maFilePath, maSaveErr := saveResultToFile(maFmt, maTitle, maItems, maActionSummary)
		var maFileMsg string
		if maSaveErr != nil {
			maFileMsg = "⚠️ 파일 저장 실패: " + maSaveErr.Error()
		} else {
			ext := strings.ToUpper(maFmtStr)
			maFileMsg = fmt.Sprintf("📄 %s 파일로 저장됨: %s", ext, maFilePath)
		}
		maResults := make([]map[string]string, 0, len(maItems))
		for _, it := range maItems {
			maResults = append(maResults, map[string]string{"site": maSite, "name": it["title"], "price": it["price"], "link": it["url"]})
		}
		return map[string]any{
			"query": maQuery, "summary": maActionSummary, "results": maResults, "total": len(maResults),
			"file_path": maFilePath, "file_msg": maFileMsg, "format": maFmtStr, "sub_action": maSubAction,
		}, maActionSummary + "\n" + maFileMsg

	// ── 파일 검색 ────────────────────────────────────────────
	case "file_search":
		query := str("query")
		if query == "" {
			query = original
		}
		folder := str("folder")
		if folder == "" {
			folder, _ = os.UserHomeDir()
		}
		maxResults := intVal("max_results", 15)
		// AI 키워드 추출 후 검색
		keywords := []string{query}
		if gKey != "" {
			ep := fmt.Sprintf(`파일 검색 쿼리에서 핵심 키워드만 추출: "%s"\nJSON: {"keywords":["k1","k2"]}`, query)
			raw, _, _ := callGroq(gKey, groqFastModel, []groqMsg{{Role: "user", Content: ep}}, 128, true)
			var kw struct {
				Keywords []string `json:"keywords"`
			}
			if json.Unmarshal([]byte(raw), &kw) == nil && len(kw.Keywords) > 0 {
				keywords = kw.Keywords
			}
		}
		hits := deepSearchFiles(strings.Join(keywords, " "), folder, maxResults)
		if len(hits) == 0 {
			return hits, fmt.Sprintf("'%s'와 관련된 파일을 찾지 못했습니다.", query)
		}
		msg := fmt.Sprintf("'%s' 검색 결과: %d개 파일 발견\n", query, len(hits))
		for i, h := range hits {
			if i >= 5 {
				msg += fmt.Sprintf("  ... 외 %d개\n", len(hits)-5)
				break
			}
			msg += fmt.Sprintf("  • %s\n", h.Path)
		}
		return hits, msg

	// ── 문서 비교 ────────────────────────────────────────────
	case "doc_compare":
		fileA := str("file_a")
		fileB := str("file_b")
		if fileA == "" || fileB == "" {
			return nil, "비교할 두 파일 경로를 알려주세요.\n예: '계약서_v1.pdf 와 계약서_v2.pdf 비교해줘'"
		}
		textA, errA := extractDocumentText(fileA)
		textB, errB := extractDocumentText(fileB)
		if errA != nil {
			return nil, "파일A를 읽을 수 없습니다: " + fileA
		}
		if errB != nil {
			return nil, "파일B를 읽을 수 없습니다: " + fileB
		}
		if len(textA) > 4000 {
			textA = textA[:4000]
		}
		if len(textB) > 4000 {
			textB = textB[:4000]
		}
		focus := str("focus")
		if focus == "" {
			focus = "both"
		}
		prompt := fmt.Sprintf(`두 문서를 비교 분석해서 JSON으로만 응답:
=== 문서A: %s ===
%s
=== 문서B: %s ===
%s
{"summary":"요약","total_differences":숫자,"differences":[{"type":"added|deleted|modified","description":"설명","severity":"low|medium|high"}],"risk_level":"low|medium|high","recommendation":"권고사항"}`,
			fileA, textA, fileB, textB)
		ans, _, err := callGroq(gKey, groqChatModel, []groqMsg{{Role: "user", Content: prompt}}, 2048, true)
		if err != nil {
			return nil, "문서 비교 실패: " + err.Error()
		}
		var parsed map[string]any
		json.Unmarshal([]byte(ans), &parsed)
		summary := "문서 비교 완료"
		if s, ok := parsed["summary"].(string); ok {
			summary = s
		}
		return parsed, "문서 비교 완료!\n" + summary

	// ── 문서 요약 ────────────────────────────────────────────
	case "doc_summary":
		filePath := str("file_path")
		if filePath == "" {
			return nil, "요약할 파일 경로를 알려주세요."
		}
		question := str("question")
		if question == "" {
			question = "핵심 내용을 5줄로 요약하고 중요 수치·날짜·이름을 정리해주세요."
		}
		text, err := extractDocumentText(filePath)
		if err != nil {
			return nil, "파일 읽기 실패: " + err.Error()
		}
		if len(text) > 8000 {
			text = text[:8000]
		}
		ans, _, err := callGroq(gKey, groqChatModel, []groqMsg{
			{Role: "user", Content: fmt.Sprintf("문서:\n%s\n\n요청: %s", text, question)},
		}, 2048, false)
		if err != nil {
			return nil, "요약 실패: " + err.Error()
		}
		return map[string]any{"summary": ans, "file": filePath}, ans

	// ── 폴더 정리 ────────────────────────────────────────────
	case "organize_folder":
		folder := str("folder")
		home, _ := os.UserHomeDir()
		// 자연어 폴더 이름 → 실제 경로 변환
		switch strings.ToLower(folder) {
		case "downloads", "다운로드":
			folder = home + `\Downloads`
		case "desktop", "바탕화면":
			folder = home + `\Desktop`
		case "documents", "문서":
			folder = home + `\Documents`
		default:
			if folder == "" {
				folder = home + `\Downloads`
			}
		}
		freed, fileCount := organizeFolder(folder)
		return map[string]any{"folder": folder, "files_organized": fileCount, "freed_mb": freed},
			fmt.Sprintf("'%s' 폴더 정리 완료!\n%d개 파일 정리됨", folder, fileCount)

	// ── 화면 분석 ────────────────────────────────────────────
	case "vision":
		question := str("question")
		if question == "" {
			question = "지금 화면을 분석해서 무슨 내용인지, 오류가 있으면 원인과 해결법을 한국어로 알려주세요."
		}
		b64, _, _, err := captureScreenPowerShell()
		if err != nil {
			return nil, "화면 캡처 실패: " + err.Error()
		}
		ans, err := callGroqVision(gKey, b64, "image/png", question)
		if err != nil {
			return nil, "화면 분석 실패: " + err.Error()
		}
		return map[string]any{"answer": ans}, ans

	// ── PC 진단 ─────────────────────────────────────────────
	case "scan":
		sr := buildScanResult()
		msg := fmt.Sprintf("PC 점수: %d점", sr.Score)
		if len(sr.Issues) == 0 {
			msg += " — 모두 정상이에요! ✅"
		} else {
			titles := make([]string, 0)
			for _, i := range sr.Issues {
				titles = append(titles, i.Title)
			}
			msg += fmt.Sprintf("\n발견된 문제 %d개:\n", len(sr.Issues))
			for _, t := range titles {
				msg += "  • " + t + "\n"
			}
		}
		return sr, msg

	// ── PC 정리 ──────────────────────────────────────────────
	case "clean":
		freed := cleanTempFiles()
		freedMB := float64(freed) / (1024 * 1024)
		return map[string]any{"freed_mb": freedMB},
			fmt.Sprintf("PC 정리 완료! %.0fMB 확보됐습니다. 🗑️", freedMB)

	// ── 보안 탐지 ────────────────────────────────────────────
	case "security_scan":
		result := runSecurityScan()
		riskCount := 0
		for _, v := range result {
			if m, ok := v.(map[string]any); ok {
				if risk, ok := m["risk"].(string); ok && risk != "low" && risk != "none" {
					riskCount++
				}
			}
		}
		if riskCount == 0 {
			return result, "보안 점검 완료! 위협 요소가 발견되지 않았습니다. 🛡️"
		}
		return result, fmt.Sprintf("⚠️ 보안 경고: %d개 위협 요소가 발견됐습니다. 상세 결과를 확인하세요.", riskCount)

	// ── PC 통계 ──────────────────────────────────────────────
	case "stats":
		mem := getMemoryUsage()
		free, total := getDiskSpace()
		diskPct := 0
		if total > 0 {
			diskPct = int(100 - float64(free)/float64(total)*100)
		}
		stats := map[string]any{"mem": mem, "disk": diskPct}
		return stats, fmt.Sprintf("현재 PC 상태:\n  💾 RAM: %d%% 사용 중\n  💿 디스크(C:): %d%% 사용 중", mem, diskPct)

	// ── 집중 모드 ────────────────────────────────────────────
	case "focus_mode":
		enable := boolVal("enable", true)
		return runFocusMode(enable)

	// ── 업무 일지 ────────────────────────────────────────────
	case "journal":
		j := buildJournalData(gKey)
		return j, fmt.Sprintf("오늘 업무 일지 작성 완료! 📝\n%s", j["summary"])

	// ── PC 건강 리포트 ───────────────────────────────────────
	case "health_report":
		reportPath, err := generateHealthReport(gKey)
		if err != nil {
			return nil, "리포트 생성 실패: " + err.Error()
		}
		return map[string]any{"path": reportPath}, "PC 건강 리포트 생성 완료! 📊\n파일: " + reportPath

	// ── 일정 등록 ────────────────────────────────────────────
	case "scheduler":
		command := str("command")
		if command == "" {
			command = original
		}
		parsed, err := parseNaturalSchedule(command, gKey)
		if err != nil {
			return nil, "일정 파싱 실패: " + err.Error()
		}
		paramsJSON, _ := json.Marshal(parsed.Params)
		task := &ScheduledTask{
			ID:           fmt.Sprintf("task_%d", time.Now().Unix()),
			Name:         parsed.TaskName,
			Command:      command,
			Action:       parsed.Action,
			ActionParams: string(paramsJSON),
			CronExpr:     parsed.CronExpr,
			NextRun:      parsed.NextRun,
			Active:       true,
			CreatedAt:    time.Now(),
		}
		globalScheduler.mu.Lock()
		globalScheduler.tasks[task.ID] = task
		globalScheduler.mu.Unlock()
		globalScheduler.save()
		return task, fmt.Sprintf("일정 등록 완료! ⏰\n'%s' (%s)", task.Name, task.CronExpr)

	// ── 앱 실행 ──────────────────────────────────────────────
	case "launch_app":
		appName := str("app_name")
		if appName == "" {
			appName = original
		}
		return runLaunchApp(appName)

	// ── 시스템 제어 ──────────────────────────────────────────
	case "system_control":
		control := str("control")
		value := intVal("value", -1)
		return runSystemControl(control, value)

	// ── 엑셀 저장 ────────────────────────────────────────────
	case "excel_save":
		title := str("title")
		rawData, _ := params["data"]
		var data [][]string
		if b, err := json.Marshal(rawData); err == nil {
			json.Unmarshal(b, &data)
		}
		if len(data) == 0 {
			return nil, "저장할 데이터가 없어요."
		}
		home, _ := os.UserHomeDir()
		savePath := fmt.Sprintf(`%s\Desktop\nexus_%s_%s.xlsx`,
			home, sanitizeFilename(title), time.Now().Format("20060102_150405"))
		if err := saveToExcel(data, savePath, title); err != nil {
			return nil, "엑셀 저장 실패: " + err.Error()
		}
		return map[string]any{"path": savePath}, "엑셀 저장 완료! 📊\n파일: " + savePath

	// ── 메모 저장 ────────────────────────────────────────────
	case "note":
		content := str("content")
		if content == "" {
			content = original
		}
		notePath := saveQuickNote(content)
		return map[string]any{"path": notePath, "content": content}, "메모 저장 완료! 📝"

	default:
		// 분류 안 된 질문 → 이력 보완 후 web_search
		resolved := resolveWithHistory(original, history)
		cat := detectCategory(resolved)
		pr := parallelWebSearch(resolved, 5)
		items := pr.Items
		if len(items) == 0 {
			items = categoryFallbackSites(resolved, cat)
		}
		msg := pr.Summary
		if msg == "" {
			msg = buildNoResultMessage(resolved, cat, "")
		}
		return map[string]any{"query": resolved, "summary": msg, "items": items}, msg
	}
}

// ══════════════════════════════════════════════════════════════════
//  액션 구현 함수들
// ══════════════════════════════════════════════════════════════════

// runWebSearch: 웹 검색 → 결과 PDF/Excel/텍스트 저장
func runWebSearch(query, site, output string, maxItems int, gKey string) (any, string) {
	if maxItems == 0 {
		maxItems = 5
	}

	ctx, cancel, err := withStealthBrowserTimeout(3 * time.Minute)
	if err != nil {
		return nil, "브라우저 시작 실패: " + err.Error()
	}
	defer cancel()

	products, _ := scrapeSearchResults(ctx, query, site, maxItems)
	if len(products) == 0 {
		// Tavily로 실시간 검색 시도
		llmMu.RLock()
		tKey := llmTavilyKey
		llmMu.RUnlock()
		if tKey != "" {
			if tr, ok := tavilySearch(tKey, query, maxItems); ok {
				return map[string]any{"summary": tr.Summary, "items": tr.Items}, tr.Summary
			}
		}
		products = generateFallbackProducts(query)
	}

	// AI 요약 (URL/출처 제외, 자연어 답변)
	var summary string
	if gKey != "" {
		lines := make([]string, 0, len(products))
		for _, p := range products {
			lines = append(lines, fmt.Sprintf("%s: %s — %s", p["rank"], p["name"], p["price"]))
		}
		today := time.Now().Format("2006-01-02")
		prompt := fmt.Sprintf(`오늘은 %s입니다.
사용자 질문: "%s"
검색 결과:
%s

[지시사항]
- URL, 링크, 출처명 절대 포함 금지
- 사용자 질문에 직접 답하는 자연스러운 한국어 2~4문장으로 핵심만 요약
- 수치(가격, 등수 등)는 포함해도 됨
- 친절한 AI 비서처럼 작성`, today, query, strings.Join(lines, "\n"))
		s, _, _ := callGroq(gKey, groqChatModel, []groqMsg{{Role: "user", Content: prompt}}, 512, false)
		summary = s
	}

	htmlContent := buildProductHTML(query, products, summary)
	home, _ := os.UserHomeDir()
	safeName := sanitizeFilename(query)
	ts := time.Now().Format("20060102_150405")

	// 출력 형식에 따라 저장
	if output == "excel" {
		data := [][]string{{"순위", "제품명", "가격", "배송", "평점"}}
		for _, p := range products {
			data = append(data, []string{p["rank"], p["name"], p["price"], p["delivery"], p["rating"]})
		}
		xlsxPath := fmt.Sprintf(`%s\Desktop\%s_%s.xlsx`, home, safeName, ts)
		if err := saveToExcel(data, xlsxPath, query); err != nil {
			return nil, "엑셀 저장 실패: " + err.Error()
		}
		return map[string]any{"path": xlsxPath, "count": len(products), "summary": summary},
			fmt.Sprintf("'%s' 검색 완료! %d개 수집 → 엑셀 저장됨\n%s\n파일: %s", query, len(products), summary, xlsxPath)
	}

	// 기본: HTML → PDF
	htmlPath := fmt.Sprintf(`%s\Desktop\%s_%s.html`, home, safeName, ts)
	pdfPath := fmt.Sprintf(`%s\Desktop\%s_%s.pdf`, home, safeName, ts)
	os.WriteFile(htmlPath, []byte(htmlContent), 0644)

	finalPath := htmlPath
	if pdfErr := chromeToPDF(ctx, htmlPath, pdfPath); pdfErr == nil {
		os.Remove(htmlPath)
		finalPath = pdfPath
	}

	msg := fmt.Sprintf("'%s' 검색 완료! %d개 결과 수집\n", query, len(products))
	if summary != "" {
		msg += summary + "\n"
	}
	msg += "파일: " + finalPath
	return map[string]any{"path": finalPath, "count": len(products), "summary": summary}, msg
}

// runSecurityScan: 보안 전반 점검
func runSecurityScan() map[string]any {
	result := map[string]any{}

	// 원격 접속 확인
	out, _ := exec.Command("powershell", "-NoProfile", "-Command",
		`Get-NetTCPConnection | Where-Object {$_.State -eq 'Established' -and $_.RemoteAddress -notlike '127.*' -and $_.RemoteAddress -ne '::1'} | Select-Object LocalPort,RemoteAddress,RemotePort,OwningProcess | ConvertTo-Json -Compress -Depth 2`).Output()
	var connections []map[string]any
	json.Unmarshal(out, &connections)
	result["remote_connections"] = connections
	result["connection_count"] = len(connections)

	// 의심 프로세스 확인
	out2, _ := exec.Command("powershell", "-NoProfile", "-Command",
		`Get-Process | Where-Object {$_.CPU -gt 50} | Select-Object Name,Id,CPU | ConvertTo-Json -Compress`).Output()
	var procs []map[string]any
	json.Unmarshal(out2, &procs)
	result["high_cpu_processes"] = procs

	// Windows Defender 상태
	out3, _ := exec.Command("powershell", "-NoProfile", "-Command",
		`(Get-MpComputerStatus | Select-Object -Property AMServiceEnabled,AntispywareEnabled,RealTimeProtectionEnabled | ConvertTo-Json -Compress)`).Output()
	var defender map[string]any
	json.Unmarshal(out3, &defender)
	result["defender"] = defender

	if len(connections) > 20 {
		result["risk"] = "high"
	} else if len(connections) > 10 {
		result["risk"] = "medium"
	} else {
		result["risk"] = "low"
	}
	return result
}

// runFocusMode: 집중 모드 켜기/끄기
func runFocusMode(enable bool) (any, string) {
	if enable {
		// 알림 끄기 (방해 금지 모드)
		exec.Command("powershell", "-NoProfile", "-Command",
			`Set-ItemProperty -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Notifications\Settings' -Name 'NOC_GLOBAL_SETTING_TOASTS_ENABLED' -Value 0 -ErrorAction SilentlyContinue`).Run()
		return map[string]any{"enabled": true}, "집중 모드 켜졌습니다! 🎯\n알림이 차단됐습니다. 집중하세요!"
	}
	exec.Command("powershell", "-NoProfile", "-Command",
		`Set-ItemProperty -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Notifications\Settings' -Name 'NOC_GLOBAL_SETTING_TOASTS_ENABLED' -Value 1 -ErrorAction SilentlyContinue`).Run()
	return map[string]any{"enabled": false}, "집중 모드 꺼졌습니다. 알림이 다시 켜졌어요."
}

// buildJournalData: 오늘 업무 일지 생성
func buildJournalData(gKey string) map[string]any {
	today := time.Now().Format("2006-01-02")
	appUsage := getAppUsageToday()
	recentFiles := getRecentFiles(time.Now().Truncate(24 * time.Hour))

	summary := buildJournalSummary(today, appUsage, recentFiles, 0)

	// AI로 더 풍부한 일지 생성
	if gKey != "" && len(appUsage) > 0 {
		appNames := make([]string, 0)
		for _, a := range appUsage {
			appNames = append(appNames, a.Name)
		}
		prompt := fmt.Sprintf("오늘 %s에 사용한 앱: %s\n오늘 작업한 파일: %d개\n\n오늘 업무를 자연스럽게 일지로 작성해주세요. (3-5줄)",
			today, strings.Join(appNames, ", "), len(recentFiles))
		aiSummary, _, _ := callGroq(gKey, groqChatModel, []groqMsg{{Role: "user", Content: prompt}}, 512, false)
		if aiSummary != "" {
			summary = aiSummary
		}
	}

	return map[string]any{
		"date":         today,
		"summary":      summary,
		"app_usage":    appUsage,
		"recent_files": recentFiles,
	}
}

// generateHealthReport: PC 건강 리포트 PDF 생성
func generateHealthReport(gKey string) (string, error) {
	sr := buildScanResult()
	mem := getMemoryUsage()
	free, total := getDiskSpace()
	diskPct := 0
	if total > 0 {
		diskPct = int(100 - float64(free)/float64(total)*100)
	}

	var aiAnalysis string
	if gKey != "" {
		prompt := fmt.Sprintf("PC 점수: %d점\n메모리: %d%%\n디스크: %d%%\n문제: %d개\n\n간단한 PC 건강 진단 보고서를 3-4줄로 작성해주세요.",
			sr.Score, mem, diskPct, len(sr.Issues))
		aiAnalysis, _, _ = callGroq(gKey, groqChatModel, []groqMsg{{Role: "user", Content: prompt}}, 512, false)
	}

	issueRows := ""
	for _, issue := range sr.Issues {
		color := "#ffc107"
		if issue.Severity == "high" {
			color = "#dc3545"
		}
		issueRows += fmt.Sprintf(`<tr><td>%s</td><td style="color:%s">%s</td><td>%s</td></tr>`,
			issue.Title, color, issue.Severity, issue.Description)
	}
	if issueRows == "" {
		issueRows = `<tr><td colspan="3" style="text-align:center;color:#28a745">✅ 모든 항목 정상</td></tr>`
	}

	html := fmt.Sprintf(`<!DOCTYPE html><html><head><meta charset="UTF-8">
<style>body{font-family:'맑은 고딕',Arial,sans-serif;margin:40px;color:#333}
h1{color:#2c3e50;border-bottom:3px solid #3498db;padding-bottom:10px}
.score{font-size:72px;font-weight:bold;color:%s;text-align:center;margin:20px}
.grid{display:grid;grid-template-columns:1fr 1fr 1fr;gap:20px;margin:20px 0}
.card{background:#f8f9fa;border-radius:8px;padding:20px;text-align:center}
.card h3{margin:0;color:#666;font-size:14px}
.card p{margin:5px 0;font-size:32px;font-weight:bold;color:#2c3e50}
table{width:100%%;border-collapse:collapse;margin:20px 0}
th{background:#3498db;color:white;padding:10px}
td{padding:8px;border-bottom:1px solid #dee2e6}
.analysis{background:#e8f4fd;border-left:4px solid #3498db;padding:15px;margin:20px 0}
</style></head><body>
<h1>🖥️ Nexus PC 건강 리포트</h1>
<p>생성일시: %s</p>
<div class="score">%d점</div>
<div class="grid">
<div class="card"><h3>💾 RAM 사용률</h3><p>%d%%</p></div>
<div class="card"><h3>💿 디스크(C:)</h3><p>%d%%</p></div>
<div class="card"><h3>⚠️ 발견된 문제</h3><p>%d개</p></div>
</div>
<div class="analysis"><strong>AI 진단:</strong> %s</div>
<h2>📋 상세 점검 결과</h2>
<table><thead><tr><th>항목</th><th>심각도</th><th>설명</th></tr></thead>
<tbody>%s</tbody></table>
<p style="color:#999;font-size:12px;text-align:center">Nexus AI 비서 — PC Health Report</p>
</body></html>`,
		scoreColor(sr.Score), time.Now().Format("2006-01-02 15:04:05"),
		sr.Score, mem, diskPct, len(sr.Issues), aiAnalysis, issueRows)

	home, _ := os.UserHomeDir()
	htmlPath := fmt.Sprintf(`%s\Desktop\nexus_health_report_%s.html`, home, time.Now().Format("20060102_150405"))
	pdfPath := strings.Replace(htmlPath, ".html", ".pdf", 1)

	if err := os.WriteFile(htmlPath, []byte(html), 0644); err != nil {
		return "", err
	}

	ctx, cancel, err := withStealthBrowserTimeout(2 * time.Minute)
	if err != nil {
		return htmlPath, nil
	}
	defer cancel()

	if pdfErr := chromeToPDF(ctx, htmlPath, pdfPath); pdfErr == nil {
		os.Remove(htmlPath)
		return pdfPath, nil
	}
	return htmlPath, nil
}

func scoreColor(score int) string {
	if score >= 80 {
		return "#28a745"
	} else if score >= 60 {
		return "#ffc107"
	}
	return "#dc3545"
}

// runLaunchApp: 앱 실행
func runLaunchApp(appName string) (any, string) {
	appMap := map[string]string{
		"크롬": "chrome", "chrome": "chrome",
		"엣지": "msedge", "edge": "msedge",
		"워드": "winword", "word": "winword",
		"엑셀": "excel",
		"메모장": "notepad", "notepad": "notepad",
		"탐색기": "explorer", "explorer": "explorer",
		"계산기": "calc",
		"파워포인트": "powerpnt", "ppt": "powerpnt",
	}

	lower := strings.ToLower(appName)
	for k, v := range appMap {
		if strings.Contains(lower, k) {
			exec.Command("cmd", "/c", "start", "", v).Start()
			return map[string]any{"app": v}, fmt.Sprintf("%s 실행했습니다! 🚀", appName)
		}
	}
	// 직접 실행 시도
	exec.Command("cmd", "/c", "start", "", appName).Start()
	return map[string]any{"app": appName}, fmt.Sprintf("'%s' 실행을 시도했습니다.", appName)
}

// runSystemControl: 볼륨/밝기/와이파이 등 시스템 제어
func runSystemControl(control string, value int) (any, string) {
	switch strings.ToLower(control) {
	case "volume", "볼륨":
		if value < 0 {
			value = 50
		}
		script := fmt.Sprintf(`Add-Type -TypeDefinition 'using System.Runtime.InteropServices; public class V{[DllImport("winmm.dll")]public static extern int waveOutSetVolume(System.IntPtr h,uint v);}';$v=[uint32](%d/100.0*65535);[V]::waveOutSetVolume([System.IntPtr]::Zero,($v -bor ($v -shl 16)))`, value)
		exec.Command("powershell", "-NoProfile", "-Command", script).Run()
		return map[string]any{"volume": value}, fmt.Sprintf("볼륨을 %d%%로 설정했습니다. 🔊", value)

	case "mute", "음소거":
		exec.Command("powershell", "-NoProfile", "-Command",
			`(New-Object -ComObject WScript.Shell).SendKeys([char]173)`).Run()
		return map[string]any{"muted": true}, "음소거 처리했습니다. 🔇"

	case "brightness", "밝기":
		if value < 0 {
			value = 70
		}
		script := fmt.Sprintf(`(Get-WmiObject -Namespace root/WMI -Class WmiMonitorBrightnessMethods).WmiSetBrightness(1,%d)`, value)
		exec.Command("powershell", "-NoProfile", "-Command", script).Run()
		return map[string]any{"brightness": value}, fmt.Sprintf("밝기를 %d%%로 설정했습니다. ☀️", value)

	case "wifi", "와이파이":
		exec.Command("powershell", "-NoProfile", "-Command",
			`(Get-NetAdapter | Where-Object {$_.InterfaceDescription -like '*Wi-Fi*' -or $_.Name -like '*Wi-Fi*'} | Enable-NetAdapter -Confirm:$false) 2>$null`).Run()
		return map[string]any{"wifi": "enabled"}, "Wi-Fi를 켰습니다. 📶"

	case "sleep", "절전":
		exec.Command("powershell", "-NoProfile", "-Command",
			`Add-Type -Assembly System.Windows.Forms; [System.Windows.Forms.Application]::SetSuspendState('Suspend',$false,$false)`).Run()
		return map[string]any{"sleep": true}, "절전 모드로 전환합니다. 💤"

	case "restart", "재시작":
		exec.Command("shutdown", "/r", "/t", "10").Run()
		return map[string]any{"restart": true}, "10초 후 재시작합니다. 🔄"

	case "shutdown", "종료":
		exec.Command("shutdown", "/s", "/t", "10").Run()
		return map[string]any{"shutdown": true}, "10초 후 종료합니다. ⏻"
	}
	return nil, fmt.Sprintf("'%s' 제어를 수행할 수 없습니다.", control)
}

// organizeFolder: 폴더 파일을 유형별로 분류
func organizeFolder(folder string) (float64, int) {
	entries, err := os.ReadDir(folder)
	if err != nil {
		return 0, 0
	}

	extMap := map[string]string{
		".jpg": "사진", ".jpeg": "사진", ".png": "사진", ".gif": "사진", ".bmp": "사진", ".webp": "사진",
		".mp4": "동영상", ".avi": "동영상", ".mov": "동영상", ".mkv": "동영상",
		".mp3": "음악", ".wav": "음악", ".flac": "음악",
		".pdf": "문서", ".docx": "문서", ".doc": "문서", ".txt": "문서",
		".xlsx": "스프레드시트", ".xls": "스프레드시트", ".csv": "스프레드시트",
		".pptx": "프레젠테이션", ".ppt": "프레젠테이션",
		".zip": "압축파일", ".rar": "압축파일", ".7z": "압축파일",
		".exe": "프로그램", ".msi": "프로그램",
	}

	count := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		subDir, ok := extMap[ext]
		if !ok {
			subDir = "기타"
		}
		targetDir := folder + `\` + subDir
		os.MkdirAll(targetDir, 0755)
		src := folder + `\` + e.Name()
		dst := targetDir + `\` + e.Name()
		if err := os.Rename(src, dst); err == nil {
			count++
		}
	}
	return 0, count
}

// saveQuickNote: 빠른 메모 저장
func saveQuickNote(content string) string {
	home, _ := os.UserHomeDir()
	notesDir := home + `\Documents\Nexus메모`
	os.MkdirAll(notesDir, 0755)
	path := fmt.Sprintf(`%s\메모_%s.txt`, notesDir, time.Now().Format("20060102_150405"))
	os.WriteFile(path, []byte(content), 0644)
	return path
}

// buildScanResult: PC 현재 상태 분석
func buildScanResult() ScanResult {
	var issues []Issue
	score := 100

	tempSize := getTempSize()
	if tempSize > 500<<20 {
		issues = append(issues, Issue{
			ID: "temp-files", Title: formatBytes(tempSize) + " 임시 파일이 쌓여있어요",
			Description: "정리하면 디스크 공간을 확보할 수 있어요", Severity: "medium", Category: "clean", Fixable: true,
		})
		score -= 10
	}
	free, total := getDiskSpace()
	if total > 0 && float64(free)/float64(total) < 0.1 {
		issues = append(issues, Issue{
			ID: "disk-space", Title: "디스크 공간 부족 (" + formatBytes(int64(free)) + " 남음)",
			Description: "불필요한 파일을 정리하세요", Severity: "high", Category: "disk", Fixable: false,
		})
		score -= 20
	}
	memUsage := getMemoryUsage()
	if memUsage > 85 {
		issues = append(issues, Issue{
			ID: "memory", Title: fmt.Sprintf("메모리 사용량 %d%% 높음", memUsage),
			Description: "불필요한 프로그램을 종료하면 빨라져요", Severity: "medium", Category: "memory", Fixable: false,
		})
		score -= 5
	}
	if score < 0 {
		score = 0
	}
	return ScanResult{Score: score, Issues: issues}
}

// ── 유틸 ────────────────────────────────────────────────────────

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// resolveWithHistory: 이전 대화 이력을 참고해 모호한 질문을 구체화
// 예) "버스 시간 알려줘" + 이전 대화 "부산 정관→인천터미널 버스" → "부산 정관에서 인천터미널 버스 시간"
func resolveWithHistory(current string, history []ConvHistoryMsg) string {
	if len(history) == 0 {
		return current
	}
	// 모호한 대명사/짧은 질문 여부 판단
	isVague := len([]rune(current)) < 15 ||
		strings.Contains(current, "그거") ||
		strings.Contains(current, "이거") ||
		strings.Contains(current, "그때") ||
		strings.Contains(current, "거기") ||
		strings.Contains(current, "아까") ||
		strings.Contains(current, "그 버스") ||
		strings.Contains(current, "그 노선") ||
		strings.Contains(current, "더 알려") ||
		strings.Contains(current, "자세히") ||
		strings.Contains(current, "시간 알") ||
		strings.Contains(current, "요금 알") ||
		strings.Contains(current, "예매") ||
		strings.Contains(current, "어떻게가") ||
		strings.Contains(current, "몇 시")
	if !isVague {
		return current
	}
	// 직전 user 메시지 + assistant 응답을 붙여서 컨텍스트 생성
	var contextParts []string
	start := len(history) - 4
	if start < 0 {
		start = 0
	}
	for _, h := range history[start:] {
		if h.Content == "" {
			continue
		}
		content := h.Content
		if len([]rune(content)) > 150 {
			content = string([]rune(content)[:150])
		}
		contextParts = append(contextParts, content)
	}
	if len(contextParts) == 0 {
		return current
	}
	return strings.Join(contextParts, " / ") + " → " + current
}

