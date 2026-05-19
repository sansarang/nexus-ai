//go:build !windows

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
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
	// 포맷 키워드 OR 저장 동사 중 하나라도 있으면 파일 저장 트리거
	isMultiAction := outFmt != outNone && req.PendingIntent == ""

	// ── Pre-routing: 액션 감지만 (Clarify 판단은 Groq에 위임) ──────
	if detectedShopSite != "" && req.PendingIntent == "" {
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
			"개발자": "developer", "개발자모드": "developer", "개발 모드": "developer", "it 모드": "developer", "코딩 모드": "developer",
			"마케터": "marketer", "마케팅 모드": "marketer", "마케팅모드": "marketer", "디지털 마케터": "marketer",
			"영업": "sales", "세일즈": "sales", "영업 모드": "sales", "세일즈 모드": "sales",
			"pm": "pm", "기획자": "pm", "pm 모드": "pm", "기획 모드": "pm", "프로덕트": "pm",
			"디자이너": "designer", "크리에이터": "designer", "디자인 모드": "designer", "크리에이티브 모드": "designer",
			"프리랜서": "freelancer", "1인 사업자": "freelancer", "프리랜서 모드": "freelancer", "사업자 모드": "freelancer",
			"기본": "developer", "기본 모드": "developer",
		}
		switchTriggers := []string{"모드로 바꿔", "모드 바꿔", "모드로 전환", "모드 전환", "페르소나", "으로 바꿔", "로 바꿔줘", "로 전환해"}
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
	var structuredResult *ClarifyResult
	if preRoutedAction != "workflow_preset" && req.PendingIntent == "" {
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
		if err := json.Unmarshal([]byte(raw), &intent); err != nil {
			intent.Action = "chat"
			intent.Message = raw
		}
	}

	// ── 사용량 체크 + 모델 티어 결정 ────────────────────────────
	tier := DecideModelTier(intent.Action)
	allowed, freeLeft, premiumLeft := globalUsage.CheckAndIncrement(userID, string(tier))
	if !allowed {
		json200(w, usageLimitResponse(tier, freeLeft, premiumLeft))
		return
	}

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
	default:
		cmdDefault(_cx)
	}
}
