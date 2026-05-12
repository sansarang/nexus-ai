package main

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// ── 전문가 정의 ─────────────────────────────────────────────────

type expert struct {
	id       string
	nameKo   string
	nameEn   string
	triggers []string // 한/영 트리거 키워드
	sysKo    string   // 한국어 시스템 프롬프트
	sysEn    string   // 영어 시스템 프롬프트
}

var experts = []expert{
	{
		id: "finance", nameKo: "재무/투자 전문가", nameEn: "Finance & Investment Expert",
		triggers: []string{
			"주가", "주식", "펀드", "etf", "코인", "비트코인", "환율", "금리", "배당", "투자", "세금", "부동산", "재테크",
			"stock", "stocks", "crypto", "bitcoin", "investment", "fund", "etf", "exchange rate", "dividend", "tax", "real estate", "finance", "nasdaq", "kospi",
		},
		sysKo: `당신은 Nexus AI의 공인 재무분석가(CFA)입니다.
검색된 최신 금융 데이터를 바탕으로 분석하세요.
[규칙] 수익률·리스크·세금 영향을 반드시 언급 | 투자 권유 금지, 참고용임을 명시 | URL/링크 금지 | 자연스러운 한국어 3~5문장`,
		sysEn: `You are Nexus AI's Chartered Financial Analyst (CFA).
Analyze using the latest financial data from search results.
[Rules] Always mention returns, risks, and tax implications | Never recommend investments, note it's for reference only | No URLs/links | Natural English 3-5 sentences`,
	},
	{
		id: "legal", nameKo: "법률 전문가", nameEn: "Legal Expert",
		triggers: []string{
			"법률", "법원", "법적", "법무", "계약", "소송", "위반", "처벌", "판례", "고소", "고발", "변호사", "형사", "민사", "규정", "조항",
			"law ", "legal ", "lawsuit", "contract violation", "penalty", "attorney", "court ruling", "criminal", "civil suit", "legal regulation", "clause",
		},
		sysKo: `당신은 Nexus AI의 법률 전문가(변호사)입니다.
검색된 법령·판례를 근거로 분석하세요.
[규칙] 관련 법조문·판례 근거 제시 | 최종 판단은 전문 변호사 상담 권고 | URL/링크 금지 | 자연스러운 한국어 3~5문장`,
		sysEn: `You are Nexus AI's Legal Expert (Attorney).
Analyze based on searched statutes and case law.
[Rules] Cite relevant laws and precedents | Recommend consulting a licensed attorney for final decisions | No URLs/links | Natural English 3-5 sentences`,
	},
	{
		id: "medical", nameKo: "의료/건강 전문가", nameEn: "Medical & Health Expert",
		triggers: []string{
			"증상", "병원", "약", "수술", "진단", "치료", "건강", "질병", "두통", "발열", "통증", "의사", "처방", "복용",
			"symptom", "hospital", "medicine", "surgery", "diagnosis", "treatment", "health", "disease", "headache", "fever", "pain", "doctor", "prescription",
		},
		sysKo: `당신은 Nexus AI의 의료 전문가(의사)입니다.
검색된 의학 정보를 바탕으로 설명하세요.
[규칙] 증상·원인·일반적 치료법 설명 | 자가진단 금지, 병원 방문 강력 권고 | URL/링크 금지 | 자연스러운 한국어 3~5문장`,
		sysEn: `You are Nexus AI's Medical Expert (Doctor).
Explain based on searched medical information.
[Rules] Describe symptoms, causes, and general treatment | Strongly recommend visiting a doctor, no self-diagnosis | No URLs/links | Natural English 3-5 sentences`,
	},
	{
		id: "marketing", nameKo: "마케팅 전문가", nameEn: "Marketing Expert",
		triggers: []string{
			"광고", "브랜드", "sns", "바이럴", "캠페인", "마케팅", "홍보", "콘텐츠", "인플루언서", "퍼포먼스", "ctr", "roas",
			"advertising", "brand", "viral", "campaign", "marketing", "promotion", "content", "influencer", "performance", "ctr", "roas", "seo", "sem",
		},
		sysKo: `당신은 Nexus AI의 마케팅 전문가(CMO)입니다.
검색된 시장 데이터와 트렌드를 바탕으로 전략을 제시하세요.
[규칙] 타겟·채널·전략·KPI 중심으로 실행 가능한 방향 제시 | URL/링크 금지 | 자연스러운 한국어 3~5문장`,
		sysEn: `You are Nexus AI's Marketing Expert (CMO).
Suggest strategies based on searched market data and trends.
[Rules] Focus on target, channel, strategy, and KPI with actionable directions | No URLs/links | Natural English 3-5 sentences`,
	},
	{
		id: "it", nameKo: "IT/개발 전문가", nameEn: "IT & Development Expert",
		triggers: []string{
			"코드", "프로그래밍", "버그", "api", "서버", "알고리즘", "데이터베이스", "클라우드", "보안", "개발", "소프트웨어", "프레임워크",
			"code", "programming", "bug", "api", "server", "algorithm", "database", "cloud", "security", "development", "software", "framework", "docker", "kubernetes",
		},
		sysKo: `당신은 Nexus AI의 시니어 소프트웨어 엔지니어입니다.
검색된 기술 문서·커뮤니티 답변을 바탕으로 해결책을 제시하세요.
[규칙] 코드·아키텍처·보안·성능 측면에서 구체적 해결책 제시 | URL/링크 금지 | 자연스러운 한국어 3~5문장`,
		sysEn: `You are Nexus AI's Senior Software Engineer.
Provide solutions based on searched technical docs and community answers.
[Rules] Give concrete solutions covering code, architecture, security, and performance | No URLs/links | Natural English 3-5 sentences`,
	},
	{
		id: "education", nameKo: "교육 전문가", nameEn: "Education Expert",
		triggers: []string{
			"공부", "시험", "입시", "학습", "커리큘럼", "수능", "교육", "학원", "과외", "성적", "대학",
			"study", "exam", "college", "learning", "curriculum", "education", "tutoring", "grade", "university", "scholarship", "gpa", "sat", "gre",
		},
		sysKo: `당신은 Nexus AI의 교육 전문가입니다.
검색된 교육 정보·학습법을 바탕으로 조언하세요.
[규칙] 학습 단계·방법론·효율적 커리큘럼 중심 조언 | URL/링크 금지 | 자연스러운 한국어 3~5문장`,
		sysEn: `You are Nexus AI's Education Expert.
Advise based on searched educational information and study methods.
[Rules] Focus on learning stages, methodology, and efficient curriculum | No URLs/links | Natural English 3-5 sentences`,
	},
	{
		id: "cooking", nameKo: "요리/식품 전문가", nameEn: "Culinary Expert",
		triggers: []string{
			"레시피", "요리", "재료", "칼로리", "조리법", "만드는법", "만드는 방법", "맛있게", "볶음", "찌개", "구이", "식단", "끓이는", "집밥", "반찬", "음식",
			"recipe", "cooking", "ingredient", "calorie", "how to cook", "how to make", "delicious", "stir fry", "stew", "grill", "diet", "meal", "homemade",
		},
		sysKo: `당신은 Nexus AI의 셰프(요리 전문가)입니다.
검색된 레시피·식품 정보를 바탕으로 설명하세요.
[규칙] 재료·순서·화력·대체재·팁까지 구체적으로 설명 | URL/링크 금지 | 자연스러운 한국어 3~5문장`,
		sysEn: `You are Nexus AI's Professional Chef.
Explain based on searched recipes and food information.
[Rules] Be specific about ingredients, steps, heat level, substitutes, and tips | No URLs/links | Natural English 3-5 sentences`,
	},
	{
		id: "travel", nameKo: "여행 전문가", nameEn: "Travel Expert",
		triggers: []string{
			"여행", "항공", "호텔", "비자", "여행일정", "관광", "해외", "숙소", "항공권", "패키지",
			"travel", "flight", "hotel", "visa", "itinerary", "tourism", "abroad", "accommodation", "airfare", "package tour", "destination",
		},
		sysKo: `당신은 Nexus AI의 여행 전문가입니다.
검색된 최신 여행 정보를 바탕으로 조언하세요.
[규칙] 계절·비용·이동수단·현지 주의사항 포함 | URL/링크 금지 | 자연스러운 한국어 3~5문장`,
		sysEn: `You are Nexus AI's Travel Expert.
Advise based on the latest searched travel information.
[Rules] Include season, cost, transportation, and local cautions | No URLs/links | Natural English 3-5 sentences`,
	},
	{
		id: "psychology", nameKo: "심리/상담 전문가", nameEn: "Psychology & Counseling Expert",
		triggers: []string{
			"우울", "스트레스", "불안", "관계", "상담", "심리", "감정", "외로움", "자존감", "트라우마", "번아웃",
			"depression", "stress", "anxiety", "relationship", "counseling", "psychology", "emotion", "loneliness", "self-esteem", "trauma", "burnout",
		},
		sysKo: `당신은 Nexus AI의 심리상담사입니다.
검색된 심리학 정보를 바탕으로 공감하며 조언하세요.
[규칙] 감정 공감 우선 | 실용적 대처법 제시 | 전문 상담사 연결 권고 | URL/링크 금지 | 자연스러운 한국어 3~5문장`,
		sysEn: `You are Nexus AI's Psychological Counselor.
Empathize and advise based on searched psychology information.
[Rules] Empathy first | Practical coping strategies | Recommend professional counseling | No URLs/links | Natural English 3-5 sentences`,
	},
	{
		id: "research", nameKo: "리서치/분석 전문가", nameEn: "Research & Analysis Expert",
		triggers: []string{
			"분석", "비교", "조사", "시장", "경쟁사", "트렌드", "통계", "보고서", "리서치",
			"analysis", "compare", "research", "market", "competitor", "trend", "statistics", "report", "survey", "data",
		},
		sysKo: `당신은 Nexus AI의 시장분석가입니다.
검색된 최신 데이터·보고서를 바탕으로 분석하세요.
[규칙] 데이터·트렌드·비교 분석 중심 | 수치 근거 제시 | URL/링크 금지 | 자연스러운 한국어 3~5문장`,
		sysEn: `You are Nexus AI's Market Research Analyst.
Analyze based on searched latest data and reports.
[Rules] Focus on data, trends, and comparative analysis | Provide numerical evidence | No URLs/links | Natural English 3-5 sentences`,
	},
	{
		id: "creative", nameKo: "크리에이티브 전문가", nameEn: "Creative Expert",
		triggers: []string{
			"아이디어", "기획", "콘텐츠", "글쓰기", "스토리", "카피", "디자인", "창작", "소설", "시나리오",
			"idea", "planning", "content", "writing", "story", "copywriting", "design", "creative", "novel", "script", "brainstorm",
		},
		sysKo: `당신은 Nexus AI의 크리에이티브 디렉터입니다.
검색된 트렌드·레퍼런스를 바탕으로 창의적 아이디어를 제시하세요.
[규칙] 독창성·실행가능성·타겟 감성 중심 | URL/링크 금지 | 자연스러운 한국어 3~5문장`,
		sysEn: `You are Nexus AI's Creative Director.
Suggest creative ideas based on searched trends and references.
[Rules] Focus on originality, feasibility, and target audience emotion | No URLs/links | Natural English 3-5 sentences`,
	},
	{
		id: "news", nameKo: "뉴스/시사 전문가", nameEn: "News & Current Affairs Expert",
		triggers: []string{
			"뉴스", "사건", "사고", "정치", "경제", "사회", "국제", "속보", "이슈", "논란",
			"news", "incident", "politics", "economy", "society", "international", "breaking", "issue", "controversy", "current events",
		},
		sysKo: `당신은 Nexus AI의 저널리스트입니다.
검색된 최신 뉴스를 바탕으로 사실 중심으로 보도하세요.
[규칙] 육하원칙 | 다양한 시각 | 팩트 중심, 의견 최소화 | URL/링크 금지 | 자연스러운 한국어 3~5문장`,
		sysEn: `You are Nexus AI's Journalist.
Report based on the latest searched news, fact-centered.
[Rules] Who/What/When/Where/Why/How | Multiple perspectives | Fact-based, minimize opinion | No URLs/links | Natural English 3-5 sentences`,
	},
}

// detectExperts: 질문에서 필요한 전문가 1~3개 자동 감지
func detectExperts(query, lang string) []expert {
	q := strings.ToLower(query)
	var matched []expert
	seen := map[string]bool{}

	for _, e := range experts {
		if seen[e.id] {
			continue
		}
		for _, t := range e.triggers {
			if strings.Contains(q, strings.ToLower(t)) {
				matched = append(matched, e)
				seen[e.id] = true
				break
			}
		}
		if len(matched) >= 3 {
			break
		}
	}
	return matched
}

// runExpertParallel: 감지된 전문가들을 병렬로 실행, citations 포함 반환
func runExpertParallel(query, lang, pKey string, expertList []expert, history []ConvHistoryMsg) (string, []string) {
	if len(expertList) == 0 || pKey == "" {
		return "", nil
	}

	type result struct {
		name  string
		ans   string
		cites []string
	}

	ch := make(chan result, len(expertList))
	var wg sync.WaitGroup

	kst := nowKST()
	timeCtx := fmt.Sprintf("현재 시각: %s KST", kst)
	if lang == "en" {
		timeCtx = fmt.Sprintf("Current time: %s KST", kst)
	}

	for _, e := range expertList {
		e := e
		wg.Add(1)
		go func() {
			defer wg.Done()
			var sys, name string
			if lang == "en" {
				sys = e.sysEn + "\n" + timeCtx
				name = e.nameEn
			} else {
				sys = e.sysKo + "\n" + timeCtx
				name = e.nameKo
			}
			msgs := []groqMsg{{Role: "system", Content: sys}}
			// 최근 대화 이력 포함 (최대 4턴)
			for i, h := range history {
				if i >= 8 {
					break
				}
				role := "user"
				if h.Role == "assistant" {
					role = "assistant"
				}
				content := h.Content
				if len([]rune(content)) > 200 {
					content = string([]rune(content)[:200]) + "..."
				}
				msgs = append(msgs, groqMsg{Role: role, Content: content})
			}
			msgs = append(msgs, groqMsg{Role: "user", Content: query})
			ans, cites, _ := callGroqWithCitations(pKey, pplxChatModel, msgs, 500)
			if ans != "" {
				ch <- result{name: name, ans: ans, cites: cites}
			}
		}()
	}

	wg.Wait()
	close(ch)

	var results []result
	for r := range ch {
		results = append(results, r)
	}

	if len(results) == 0 {
		return "", nil
	}

	// 모든 citations 합치기
	allCites := []string{}
	seen := map[string]bool{}
	for _, r := range results {
		for _, c := range r.cites {
			if !seen[c] {
				allCites = append(allCites, c)
				seen[c] = true
			}
		}
	}

	if len(results) == 1 {
		return results[0].ans, allCites
	}

	// 복수 전문가 결과 → Perplexity로 통합 요약
	var parts []string
	for _, r := range results {
		parts = append(parts, fmt.Sprintf("[%s]\n%s", r.name, r.ans))
	}
	combined := strings.Join(parts, "\n\n")

	var integrateSys, integrateUser string
	if lang == "en" {
		integrateSys = "You are Nexus AI. Synthesize the following expert opinions into one coherent, natural English answer (3-5 sentences). No headers, no bullets, no URLs."
		integrateUser = fmt.Sprintf("User question: \"%s\"\n\nExpert opinions:\n%s", query, combined)
	} else {
		integrateSys = "당신은 Nexus AI입니다. 아래 전문가 의견들을 하나의 자연스러운 한국어 답변으로 통합하세요 (3~5문장). 헤더·불릿·URL 금지."
		integrateUser = fmt.Sprintf("사용자 질문: \"%s\"\n\n전문가 의견:\n%s", query, combined)
	}

	integrated, integCites, _ := callGroqWithCitations(pKey, pplxChatModel, []groqMsg{
		{Role: "system", Content: integrateSys},
		{Role: "user", Content: integrateUser},
	}, 600)

	// 통합 요약의 citations도 합치기
	for _, c := range integCites {
		if !seen[c] {
			allCites = append(allCites, c)
			seen[c] = true
		}
	}

	if integrated != "" {
		return integrated, allCites
	}
	return results[0].ans, allCites
}

// expertCategoryMap: 전문가 ID → 카테고리 매핑
var expertCategoryMap = map[string]queryCategory{
	"finance":    catFinance,
	"legal":      catLegal,
	"medical":    catMedical,
	"marketing":  catGeneral,
	"it":         catTech,
	"education":  catEducation,
	"cooking":    catRecipe,
	"travel":     catTravel,
	"psychology": catGeneral,
	"research":   catGeneral,
	"creative":   catGeneral,
	"news":       catNews,
}

// expertsToCategory: 감지된 전문가 목록에서 가장 적합한 카테고리 반환
func expertsToCategory(expertList []expert) queryCategory {
	if len(expertList) == 0 {
		return -1
	}
	if cat, ok := expertCategoryMap[expertList[0].id]; ok {
		return cat
	}
	return -1
}

// nowKST: 현재 KST 시각 문자열
func nowKST() string {
	kst := time.FixedZone("KST", 9*3600)
	return time.Now().In(kst).Format("2006-01-02 15:04")
}
