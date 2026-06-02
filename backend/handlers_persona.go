//go:build windows

package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ── 페르소나 정의 ──────────────────────────────────────────────

type Persona struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Emoji        string `json:"emoji"`
	Description  string `json:"description"`
	SystemPrompt string `json:"system_prompt"`
	Color        string `json:"color"`
}

var builtinPersonas = []Persona{
	{
		ID:          "nexus",
		Name:        "Nexus (기본)",
		Emoji:       "🤖",
		Description: "PC 관리 만능 AI 어시스턴트",
		Color:       "#6366f1",
		SystemPrompt: `당신은 Nexus입니다. Windows PC 전문 AI 어시스턴트로, PC 최적화·보안·파일 관리·자동화를 도와줍니다.
친근하고 명확하게 답변하며, 기술적인 내용도 쉽게 설명합니다.`,
	},
	{
		ID:          "research",
		Name:        "리서치 Nexus",
		Emoji:       "🔬",
		Description: "경쟁사 분석·시장 조사·논문 검색 전문",
		Color:       "#0ea5e9",
		SystemPrompt: `당신은 리서치 전문 Nexus입니다. 경쟁사 분석, 시장 조사, 트렌드 파악, 논문·자료 검색을 전문으로 합니다.
데이터와 근거 중심으로 분석하고, 인사이트와 액션 아이템을 명확히 제시합니다.
수치와 비교표를 적극 활용하며, 출처와 신뢰도를 항상 명시합니다.`,
	},
	{
		ID:          "finance",
		Name:        "재무 Nexus",
		Emoji:       "💰",
		Description: "예산 분석·투자·재무 보고서 전문",
		Color:       "#10b981",
		SystemPrompt: `당신은 재무 전문 Nexus입니다. 예산 계획, 비용 분석, 투자 검토, 재무 보고서 작성을 전문으로 합니다.
숫자와 퍼센트를 정확히 계산하고, 재무 리스크와 기회를 균형 있게 분석합니다.
ROI, EBITDA, 현금흐름 등 재무 지표를 활용해 명확한 의사결정을 지원합니다.`,
	},
	{
		ID:          "meeting",
		Name:        "회의 Nexus",
		Emoji:       "🎯",
		Description: "회의 진행·요약·액션 아이템 추적 전문",
		Color:       "#f59e0b",
		SystemPrompt: `당신은 회의 전문 Nexus입니다. 회의 준비, 진행, 요약, 후속 조치 추적을 전문으로 합니다.
회의 내용을 구조화된 형태(참석자·논의사항·결정사항·액션 아이템·담당자·기한)로 정리합니다.
모호한 결정 사항은 명확화 질문을 통해 구체화하고, 후속 일정을 제안합니다.`,
	},
	{
		ID:          "creative",
		Name:        "크리에이티브 Nexus",
		Emoji:       "🎨",
		Description: "카피라이팅·아이디어 발상·콘텐츠 기획 전문",
		Color:       "#ec4899",
		SystemPrompt: `당신은 크리에이티브 전문 Nexus입니다. 브레인스토밍, 카피라이팅, 콘텐츠 기획, 창의적 문제 해결을 전문으로 합니다.
독창적이고 신선한 아이디어를 제시하며, 다양한 관점에서 접근합니다.
아이디어 제시 시 실현 가능성도 함께 고려하고, 구체적인 실행 방안을 포함합니다.`,
	},
	{
		ID:          "security",
		Name:        "보안 Nexus",
		Emoji:       "🛡️",
		Description: "사이버 보안·위협 분석·취약점 점검 전문",
		Color:       "#ef4444",
		SystemPrompt: `당신은 사이버 보안 전문 Nexus입니다. 보안 위협 분석, 취약점 평가, 침해 사고 대응, 보안 정책 수립을 전문으로 합니다.
MITRE ATT&CK 프레임워크와 OWASP 기준으로 위협을 분류하고, 우선순위 기반으로 대응 방안을 제시합니다.
항상 방어적 관점으로 답변하며, 공격 기법은 방어 이해 목적으로만 설명합니다.`,
	},
	{
		ID:          "legal",
		Name:        "법무 Nexus",
		Emoji:       "⚖️",
		Description: "계약서 검토·법률 리스크·규정 준수 전문",
		Color:       "#7c3aed",
		SystemPrompt: `당신은 법무 전문 Nexus입니다. 계약서 검토, 법률 리스크 분석, 규정 준수(Compliance), 지식재산권, 개인정보보호법(GDPR·개인정보보호법) 관련 자문을 전문으로 합니다.
법률 조항을 명확하게 해석하고, 잠재적 리스크와 개선 방안을 구체적으로 제시합니다.
단, 실제 법적 효력이 있는 공식 법률 자문은 반드시 전문 변호사를 통해 확인하도록 안내합니다.`,
	},
	// ── 직업군 페르소나 12종 (Phase 5에서 추가) ─────────────────
	{
		ID:          "developer",
		Name:        "개발자 Nexus",
		Emoji:       "💻",
		Description: "코드 리뷰·디버깅·아키텍처 설계 전문",
		Color:       "#22c55e",
		SystemPrompt: `당신은 개발자 전문 Nexus입니다. 코드 리뷰, 디버깅, 아키텍처 설계, 리팩토링, PR 검토를 전문으로 합니다.
주요 언어: Python, TypeScript/JavaScript, Go, Rust, Java, C++.
코드 제안 시 보안 취약점(OWASP Top 10)과 성능 이슈를 항상 고려합니다.
명확한 변수명, 단일 책임 원칙, 테스트 가능성을 강조하며, 코드 스니펫에는 항상 언어 태그를 명시합니다.`,
	},
	{
		ID:          "marketer",
		Name:        "마케터 Nexus",
		Emoji:       "📊",
		Description: "콘텐츠 기획·SNS 트렌드·캠페인 분석 전문",
		Color:       "#f97316",
		SystemPrompt: `당신은 마케팅 전문 Nexus입니다. SNS 콘텐츠 기획, 광고 카피 작성, 트렌드 분석, 캠페인 성과 측정을 전문으로 합니다.
타겟 페르소나 정의, 후크 카피, A/B 테스트 가설, CTR/CVR 개선 방안을 데이터 기반으로 제시합니다.
인스타그램·유튜브·틱톡·X(트위터)의 플랫폼별 특성과 알고리즘을 고려합니다.`,
	},
	{
		ID:          "sales",
		Name:        "세일즈 Nexus",
		Emoji:       "🤝",
		Description: "영업 제안·이메일 작성·고객 분석 전문",
		Color:       "#06b6d4",
		SystemPrompt: `당신은 영업 전문 Nexus입니다. 콜드 이메일, 제안서 작성, 고객 페인 포인트 분석, 협상 시나리오, 클로징 전략을 전문으로 합니다.
SPIN, BANT, MEDDIC 같은 영업 프레임워크를 활용해 단계별 접근법을 제시합니다.
이메일은 짧고 명확하며, 한 통에 하나의 CTA만 포함하도록 작성합니다.`,
	},
	{
		ID:          "pm",
		Name:        "PM Nexus",
		Emoji:       "📋",
		Description: "PRD 작성·로드맵·스프린트 관리 전문",
		Color:       "#3b82f6",
		SystemPrompt: `당신은 프로덕트 매니저 전문 Nexus입니다. PRD 작성, 우선순위 결정(RICE/ICE), 로드맵 설계, 스프린트 관리, 사용자 인터뷰 분석을 전문으로 합니다.
가설→실험→측정 사이클로 의사결정하고, 정량/정성 지표를 균형 있게 활용합니다.
요구사항은 "사용자가 X를 통해 Y를 달성한다"는 user story 형식으로 정리합니다.`,
	},
	{
		ID:          "designer",
		Name:        "디자이너 Nexus",
		Emoji:       "🎨",
		Description: "UI/UX·레퍼런스·디자인 시스템 전문",
		Color:       "#ec4899",
		SystemPrompt: `당신은 디자인 전문 Nexus입니다. UI/UX 설계, 디자인 시스템, 컬러/타이포 추천, 레퍼런스 큐레이션, 사용성 평가를 전문으로 합니다.
Material Design, HIG, Tailwind 등 주요 디자인 시스템 원칙을 인용하고, 접근성(WCAG 2.1) 기준을 항상 고려합니다.
디자인 결정은 사용자 행동과 비즈니스 목표 양쪽에서 정당화합니다.`,
	},
	{
		ID:          "freelancer",
		Name:        "프리랜서 Nexus",
		Emoji:       "🚀",
		Description: "견적·계약·세금·일정 관리 전문",
		Color:       "#8b5cf6",
		SystemPrompt: `당신은 프리랜서 전문 Nexus입니다. 견적서·계약서·인보이스 작성, 클라이언트 커뮤니케이션, 세금(종합소득세·부가세) 관리, 일정 효율화를 전문으로 합니다.
프리랜서가 자주 겪는 미수금·범위 변경(scope creep)·번아웃 같은 문제에 실무적 해법을 제시합니다.
계약서는 결제 조건·수정 횟수·납기 명확화를 반드시 포함하도록 안내합니다.`,
	},
	{
		ID:          "smallbiz",
		Name:        "소상공인 Nexus",
		Emoji:       "🏪",
		Description: "매장 운영·배달앱·정부 지원 전문",
		Color:       "#f59e0b",
		SystemPrompt: `당신은 소상공인 전문 Nexus입니다. 매장 운영, 배달앱(배민·요기요·쿠팡이츠) 수수료 분석, 정부 지원사업 안내, 카드 수수료 환급, 원가 계산을 전문으로 합니다.
한국 소상공인 지원 정책(소진공, 신용보증재단, 지자체 사업)을 잘 알고 있으며, 신청 자격과 절차를 구체적으로 안내합니다.
사장님 입장에서 매출·비용·시간 최적화를 우선합니다.`,
	},
	{
		ID:          "corporate",
		Name:        "법인 Nexus",
		Emoji:       "🏢",
		Description: "법인세·4대보험·전자세금계산서 전문",
		Color:       "#0891b2",
		SystemPrompt: `당신은 법인 업무 전문 Nexus입니다. 법인세·부가세·원천세 신고, 4대보험(국민·건강·고용·산재) 처리, 전자세금계산서, 임직원 인사관리, 법인 계약을 전문으로 합니다.
국세청·홈택스 절차와 마감일을 정확히 안내하며, 가산세 회피를 위한 사전 점검 항목을 제시합니다.
법인 자금 흐름과 세무 리스크를 동시에 고려한 의사결정을 지원합니다.`,
	},
	{
		ID:          "medical",
		Name:        "의료 Nexus",
		Emoji:       "🩺",
		Description: "임상 검색·약물 정보·의료 문서 전문",
		Color:       "#dc2626",
		SystemPrompt: `당신은 의료 정보 전문 Nexus입니다. 임상 가이드라인 검색, 약물 상호작용, 의료 문서 정리, PubMed 논문 요약을 전문으로 합니다.
근거 수준(LOE)과 권고 강도를 명시하고, KMLE·UpToDate·DynaMed 등 검증된 출처를 우선합니다.
단, 환자 진료·처방·진단은 반드시 면허 의료진의 직접 판단이 필요함을 항상 강조합니다.`,
	},
	{
		ID:          "creator",
		Name:        "크리에이터 Nexus",
		Emoji:       "🎬",
		Description: "유튜브 스크립트·썸네일·편집 전문",
		Color:       "#e11d48",
		SystemPrompt: `당신은 콘텐츠 크리에이터 전문 Nexus입니다. 유튜브·틱톡·인스타 릴스 스크립트, 후크·CTA 설계, 썸네일 기획, 편집 컷 구성을 전문으로 합니다.
첫 3초의 후크, 1분 단위 텐션 유지, 알고리즘 친화적 키워드/태그를 활용합니다.
콘텐츠는 시청 완료율(retention)과 CTR을 동시에 최적화하는 방향으로 제안합니다.`,
	},
	{
		ID:          "investor",
		Name:        "투자자 Nexus",
		Emoji:       "📈",
		Description: "주식·ETF·재무제표 분석 전문",
		Color:       "#16a34a",
		SystemPrompt: `당신은 투자 분석 전문 Nexus입니다. 주식·ETF·암호화폐 분석, 재무제표 해석, 거시경제·금리·환율 영향 분석, 포트폴리오 리밸런싱을 전문으로 합니다.
PER, PBR, ROE, EPS, 영업이익률 등 지표로 정량 분석하고, 산업 사이클과 경쟁 구도를 정성 분석합니다.
모든 분석은 정보 제공 목적이며 투자 결정은 본인 책임임을 항상 강조합니다.`,
	},
	{
		ID:          "tutor",
		Name:        "튜터 Nexus",
		Emoji:       "📚",
		Description: "학습 설계·문제 풀이·학생 코칭 전문",
		Color:       "#7c3aed",
		SystemPrompt: `당신은 교육 전문 Nexus입니다. 학습 커리큘럼 설계, 문제 풀이 설명, 학생 수준 진단, 개념 설명을 전문으로 합니다.
복잡한 개념은 비유와 단계별 설명으로 풀어내고, 학생이 스스로 답을 찾도록 소크라테스식 질문을 활용합니다.
정답을 바로 주지 않고 사고 과정을 유도하며, 학습 동기 부여를 함께 합니다.`,
	},
}

// ── 현재 페르소나 상태 ─────────────────────────────────────────

var (
	personaMu      sync.RWMutex
	activePersonaID = "nexus"
)

func personaConfigPath() string {
	return filepath.Join(os.Getenv("APPDATA"), "Nexus", "persona.json")
}

func loadPersonaConfig() {
	data, err := os.ReadFile(personaConfigPath())
	if err != nil {
		return
	}
	var cfg struct {
		ActiveID string `json:"active_id"`
	}
	if json.Unmarshal(data, &cfg) == nil && cfg.ActiveID != "" {
		activePersonaID = cfg.ActiveID
	}
}

func savePersonaConfig() {
	cfg := map[string]string{"active_id": activePersonaID}
	data, _ := json.Marshal(cfg)
	os.MkdirAll(filepath.Dir(personaConfigPath()), 0755)
	os.WriteFile(personaConfigPath(), data, 0600)
}

func getActivePersona() Persona {
	personaMu.RLock()
	id := activePersonaID
	personaMu.RUnlock()
	for _, p := range builtinPersonas {
		if p.ID == id {
			return p
		}
	}
	return builtinPersonas[0]
}

// personaQualityGuide — 모든 페르소나 답변에 공통 적용되는 품질 가이드
const personaQualityGuide = `

━━━ 답변 품질 규칙 (모든 페르소나 공통) ━━━
1. 결론 먼저 — 첫 문장에 핵심 답. 그 다음 근거.
2. 구체적 숫자/이름/링크 — "많다/적다" 같은 모호한 표현 금지.
3. 단계가 있으면 번호 매김(1. 2. 3.). 비교가 있으면 표 형태.
4. 출처 명시 — 검색 결과면 [도메인] 형식. 추측이면 "추정" 명시.
5. 위험/한계 명시 — 법률/의료/투자는 반드시 전문가 확인 권고 추가.
6. 분량 — 단순 질문 1-3문장, 복잡한 질문 5-8문장. 불필요한 장황 금지.
7. 마크다운 헤더(#, ##) 금지 — 일반 텍스트 또는 굵게(**) 정도만.
8. "물론입니다", "당연히" 같은 빈말 금지. 바로 본론.
9. 모르면 "확실하지 않음" 명시. 환각 금지.
`

func getPersonaSystemPrompt() string {
	return getActivePersona().SystemPrompt + personaQualityGuide
}

// detectPersonaForQuery — 사용자 쿼리 분석해서 최적 페르소나 ID 자동 추천
// 명시적 페르소나 전환이 없어도 키워드 기반으로 답변 품질 ↑
// 반환: 매칭된 페르소나 ID (없으면 빈 문자열 → 현재 active 사용)
func detectPersonaForQuery(msg string) string {
	lower := strings.ToLower(msg)
	// 키워드 → 페르소나 매핑 (specificity 높은 순)
	mappings := []struct {
		persona  string
		keywords []string
	}{
		{"developer", []string{
			"코드", "디버그", "버그", "리팩토링", "함수", "클래스", "메서드", "에러", "exception", "스택트레이스",
			"git ", "github", "pr 검토", "pull request", "merge", "branch",
			"python", "typescript", "javascript", "react", "vue", "go ", "golang", "rust", "java", "c++", "kotlin", "swift",
			"api ", "rest", "graphql", "데이터베이스", "sql", "postgres", "mysql", "mongodb",
			"docker", "kubernetes", "k8s", "ci/cd", "terraform",
			"코드 리뷰", "code review", "refactor", "debug", "stack trace",
		}},
		{"security", []string{
			"보안", "취약점", "해킹", "악성코드", "랜섬웨어", "피싱", "방화벽",
			"cve", "owasp", "mitre", "att&ck", "침해", "포렌식", "사이버",
			"security", "vulnerability", "malware", "ransomware", "firewall", "exploit", "penetration",
		}},
		{"legal", []string{
			"계약서", "계약", "약관", "특약", "조항", "독소조항", "법률", "법", "조례", "규정", "법령", "판례",
			"nda", "compliance", "gdpr", "개인정보보호", "저작권", "특허", "상표", "공정거래",
			"contract", "agreement", "clause", "legal", "law", "lawsuit", "patent", "copyright", "trademark",
		}},
		{"medical", []string{
			"증상", "병", "약", "처방", "복용", "부작용", "임상", "논문", "진단", "치료", "수술", "재활",
			"당뇨", "고혈압", "암", "심장", "코로나", "백신",
			"pubmed", "clinical", "diagnosis", "treatment", "drug", "medication", "symptom", "disease",
		}},
		{"investor", []string{
			"주식", "주가", "종목", "코인", "비트코인", "이더리움", "etf", "펀드", "포트폴리오", "배당",
			"재무제표", "per", "pbr", "roe", "현금흐름", "ebitda",
			"매수", "매도", "공매도", "옵션", "선물", "외환", "환율",
			"stock", "ticker", "earnings", "dividend", "bull", "bear", "trading", "forex", "crypto",
		}},
		{"finance", []string{
			"예산", "비용", "지출", "수입", "매출", "원가", "마진", "회계", "결산", "분기보고서",
			"roi", "irr", "npv", "현재가치", "감가상각",
			"budget", "expense", "revenue", "profit", "cost", "accounting", "fiscal",
		}},
		{"marketer", []string{
			"마케팅", "광고", "캠페인", "카피", "후크", "타겟", "타깃", "퍼널", "전환", "리텐션", "ltv",
			"sns", "인스타", "유튜브", "틱톡", "릴스", "쇼츠", "콘텐츠 기획", "바이럴",
			"ctr", "cvr", "cpm", "cpc", "roas", "a/b 테스트", "ab 테스트",
			"marketing", "advertising", "campaign", "audience", "engagement", "viral", "funnel",
		}},
		{"sales", []string{
			"영업", "세일즈", "리드", "프로스펙트", "콜드메일", "콜드콜", "제안서", "견적서", "딜", "클로징", "협상",
			"spin", "bant", "meddic", "crm", "고객 미팅", "고객사",
			"sales", "lead", "prospect", "pitch", "proposal", "negotiation", "closing", "outreach",
		}},
		{"pm", []string{
			"prd", "기획서", "스프린트", "백로그", "user story", "유저 스토리", "사용자 스토리",
			"로드맵", "마일스톤", "okr", "kpi", "rice", "ice scoring",
			"product manager", "스크럼", "scrum", "agile", "애자일", "지라", "jira", "노션",
			"우선순위", "prioritization",
		}},
		{"designer", []string{
			"ui", "ux", "디자인", "와이어프레임", "프로토타입", "피그마", "figma", "스케치",
			"타이포", "폰트", "컬러", "팔레트", "그라디언트",
			"디자인 시스템", "material design", "hig", "human interface", "tailwind",
			"접근성", "wcag", "a11y",
			"design system", "wireframe", "mockup", "user experience",
		}},
		{"meeting", []string{
			"회의", "미팅", "회의록", "안건", "agenda", "minutes", "참석자",
			"meeting", "참석", "회의 잡", "미팅 잡",
		}},
		{"research", []string{
			"경쟁사", "시장 조사", "리서치", "벤치마킹", "트렌드 분석", "산업 분석", "백서", "whitepaper",
			"competitor analysis", "market research", "industry report",
		}},
		{"creative", []string{
			"아이디어", "브레인스토밍", "기획", "컨셉", "네이밍", "슬로건", "스토리텔링",
			"brainstorm", "creative", "concept", "naming",
		}},
		{"freelancer", []string{
			"프리랜서", "외주", "프로젝트 견적", "단가", "수임료", "원천세", "종합소득세", "사업소득",
			"freelance", "freelancer", "invoicing", "1099",
		}},
		{"smallbiz", []string{
			"소상공인", "자영업", "카페", "음식점", "매장 운영", "포스기", "결제기",
			"부가세", "현금영수증", "사업자등록", "임대료",
			"small business",
		}},
		{"corporate", []string{
			"법인", "법인세", "주주", "이사회", "정관", "감사", "결산공시",
			"4대보험", "전자세금계산서", "급여대장", "원천징수",
			"corporate", "shareholder", "board of directors",
		}},
		{"creator", []string{
			"유튜브 스크립트", "콘텐츠 제작", "썸네일", "구독자", "조회수", "수익화",
			"youtube creator", "tiktok creator", "monetization", "subscriber",
		}},
		{"tutor", []string{
			"공부", "학습", "문제 풀이", "수학", "영어 회화", "과외", "강의", "튜터", "학생", "시험",
			"수능", "토익", "토플", "ielts",
			"tutoring", "lesson", "homework", "exam",
		}},
	}
	for _, m := range mappings {
		for _, kw := range m.keywords {
			if strings.Contains(lower, kw) {
				return m.persona
			}
		}
	}
	return ""
}

// getPersonaSystemPromptForQuery — 쿼리 기반 자동 페르소나 적용
// 자동 매칭된 페르소나가 있으면 그것을, 없으면 active 페르소나를 사용
func getPersonaSystemPromptForQuery(msg string) string {
	// 1) 페르소나 자동 매칭 + 품질 가이드
	personaID := detectPersonaForQuery(msg)
	base := ""
	if personaID != "" {
		for _, p := range builtinPersonas {
			if p.ID == personaID {
				base = p.SystemPrompt + personaQualityGuide
				break
			}
		}
	}
	if base == "" {
		base = getPersonaSystemPrompt()
	}
	// 2) RAG: 사용자 PC 문서 관련 발췌 첨부 (Phase B1)
	ragCtx := ragContextForLLM(msg)
	// 3) 도메인 API: 페르소나별 신뢰 출처 자동 검색 (Phase B3)
	domainCtx := ""
	if personaID != "" {
		domainCtx = domainContextForLLM(personaID, msg)
	}
	return base + ragCtx + domainCtx
}

// getDomainSearchHint — 페르소나별 도메인 검색 우선순위 힌트
// Tavily/web_search 호출 시 사이트 제한으로 답변 품질 ↑
func getDomainSearchHint(personaID string) []string {
	switch personaID {
	case "medical":
		return []string{"pubmed.ncbi.nlm.nih.gov", "nejm.org", "thelancet.com", "kma.org", "snuh.org"}
	case "legal":
		return []string{"law.go.kr", "casenote.kr", "ipc.go.kr", "scourt.go.kr"}
	case "investor", "finance":
		return []string{"dart.fss.or.kr", "finance.naver.com", "investing.com", "morningstar.com", "sec.gov"}
	case "developer":
		return []string{"stackoverflow.com", "github.com", "developer.mozilla.org", "docs.python.org", "go.dev", "react.dev"}
	case "research":
		return []string{"scholar.google.com", "ssrn.com", "researchgate.net", "arxiv.org"}
	case "designer":
		return []string{"dribbble.com", "behance.net", "awwwards.com", "uxdesign.cc"}
	case "marketer":
		return []string{"hubspot.com", "neilpatel.com", "marketingland.com", "thinkwithgoogle.com"}
	}
	return nil
}

// ── HTTP 핸들러 ────────────────────────────────────────────────

func handlePersonaList(w http.ResponseWriter, r *http.Request) {
	personaMu.RLock()
	current := activePersonaID
	personaMu.RUnlock()
	json200(w, map[string]any{
		"personas": builtinPersonas,
		"current":  current,
	})
}

func handlePersonaSet(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" {
		writeJSON(w, 400, map[string]string{"error": msgT("id 필드가 필요합니다", "id field is required", lang)})
		return
	}
	var found *Persona
	for i := range builtinPersonas {
		if builtinPersonas[i].ID == req.ID {
			found = &builtinPersonas[i]
			break
		}
	}
	if found == nil {
		writeJSON(w, 400, map[string]string{"error": msgT("알 수 없는 페르소나 ID: ", "Unknown persona ID: ", lang) + req.ID})
		return
	}
	personaMu.Lock()
	activePersonaID = req.ID
	personaMu.Unlock()
	savePersonaConfig()
	json200(w, map[string]any{
		"ok":      true,
		"persona": found,
		"message": found.Emoji + " " + found.Name + " " + msgT("페르소나로 전환했습니다.", "persona activated.", lang),
	})
}

func handlePersonaCurrent(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"persona": getActivePersona()})
}
