package main

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// ── 카테고리 감지 ──────────────────────────────────────────────────

type queryCategory int

const (
	catGeneral queryCategory = iota
	catTransit       // 교통 (버스/기차/지하철/항공)
	catFood          // 맛집/음식/배달
	catShopping      // 쇼핑/최저가
	catFinance       // 금융 (환율/주가/코인/보험)
	catWeather       // 날씨
	catNews          // 뉴스/시사
	catMedical       // 의료/건강/병원/약
	catLegal         // 법률/행정/세금
	catEntertainment // 영화/공연/스포츠/게임
	catRecipe        // 요리/레시피
	catTravel        // 여행/숙박/관광
	catTech          // IT/기술/소프트웨어
	catEducation     // 교육/자격증/시험
	catRealEstate    // 부동산/아파트/전세
)

func detectCategory(q string) queryCategory {
	lower := strings.ToLower(q)
	has := func(words ...string) bool {
		for _, w := range words {
			if strings.Contains(lower, w) {
				return true
			}
		}
		return false
	}
	switch {
	case has("버스", "고속버스", "시외버스", "ktx", "기차", "열차", "지하철", "전철", "항공", "비행기", "시간표", "노선", "요금", "예매", "승차권", "터미널", "역에서", "공항", "택시요금", "어떻게가", "길찾기", "경로"):
		return catTransit
	case has("맛집", "식당", "카페", "음식점", "배달", "메뉴", "맛있는", "뭐먹", "점심", "저녁", "아침메뉴", "맛나는", "먹을곳", "밥집", "분식", "치킨", "피자", "초밥", "삼겹살"):
		return catFood
	case has("최저가", "가격비교", "쿠팡", "네이버쇼핑", "11번가", "지마켓", "옥션", "테무", "알리", "사고싶", "얼마", "싸게", "할인", "쇼핑"):
		return catShopping
	case has("환율", "주가", "코스피", "코스닥", "비트코인", "코인", "달러", "엔화", "유로", "금리", "적금", "예금", "펀드", "주식", "etf", "나스닥", "다우"):
		return catFinance
	case has("날씨", "기온", "비", "눈", "흐림", "맑음", "습도", "강수", "황사", "미세먼지"):
		return catWeather
	case has("뉴스", "속보", "이슈", "사건", "사고", "정치", "경제뉴스", "오늘뉴스", "최신뉴스"):
		return catNews
	case has("병원", "약국", "증상", "진료", "건강", "의사", "처방", "약먹", "아파", "질환", "치료", "수술", "건강보험"):
		return catMedical
	case has("세금", "신고", "건강보험료", "국세청", "홈택스", "법", "규정", "벌금", "과태료", "주민등록", "정부24", "민원", "행정"):
		return catLegal
	case has("영화", "cgv", "롯데시네마", "메가박스", "공연", "뮤지컬", "콘서트", "야구", "축구", "농구", "배구", "kbo", "k리그", "스포츠", "경기결과", "게임"):
		return catEntertainment
	case has("레시피", "만드는법", "요리법", "재료", "칼로리", "조리", "끓이는", "볶는", "굽는"):
		return catRecipe
	case has("여행", "호텔", "숙박", "관광", "펜션", "에어비앤비", "여행지", "해외여행", "국내여행", "가볼만한"):
		return catTravel
	case has("설치방법", "오류", "에러", "버그", "프로그램", "앱", "소프트웨어", "윈도우", "맥", "리눅스", "코딩", "개발", "api"):
		return catTech
	case has("자격증", "시험", "공부", "강의", "학원", "토익", "토플", "수능", "대학교", "입시"):
		return catEducation
	case has("아파트", "전세", "월세", "매매", "부동산", "청약", "분양", "집값", "임대"):
		return catRealEstate
	default:
		return catGeneral
	}
}

// optimizeQuery: 카테고리에 맞게 검색 쿼리 최적화
func optimizeQuery(original string, cat queryCategory) string {
	today := time.Now().Format("2006년 1월")
	q := original
	switch cat {
	case catTransit:
		if !strings.Contains(q, "시간표") && !strings.Contains(q, "요금") && !strings.Contains(q, "예매") {
			q += " 시간표 요금"
		}
		q += " " + time.Now().Format("2006")
	case catFinance:
		q += " " + today + " 실시간"
	case catNews:
		q += " " + today
	case catFood:
		if !strings.Contains(q, "추천") && !strings.Contains(q, "맛집") {
			q += " 추천"
		}
	case catMedical:
		q += " 원인 증상 치료"
	case catRealEstate:
		q += " " + today
	case catEntertainment:
		if strings.Contains(q, "야구") || strings.Contains(q, "축구") || strings.Contains(q, "농구") {
			q += " " + time.Now().Format("2006") + " 일정 결과"
		}
	}
	return q
}

// categoryFallbackSites: 카테고리별 공식 사이트 링크 DB
func categoryFallbackSites(query string, cat queryCategory) []map[string]string {
	enc := urlEncode(query)
	naverSearch := map[string]string{"title": "네이버 검색: " + query, "url": "https://search.naver.com/search.naver?query=" + enc}
	googleSearch := map[string]string{"title": "구글 검색: " + query, "url": "https://www.google.com/search?q=" + enc}

	lower := strings.ToLower(query)
	switch cat {
	case catTransit:
		isAir := strings.Contains(lower, "항공") || strings.Contains(lower, "비행기")
		isTrain := strings.Contains(lower, "ktx") || strings.Contains(lower, "기차") || strings.Contains(lower, "열차") || strings.Contains(lower, "무궁화") || strings.Contains(lower, "새마을")
		isSubway := strings.Contains(lower, "지하철") || strings.Contains(lower, "전철")
		isTaxi := strings.Contains(lower, "택시")
		if isAir {
			return []map[string]string{
				{"title": "네이버 항공권 검색", "url": "https://flight.naver.com/"},
				{"title": "스카이스캐너", "url": "https://www.skyscanner.co.kr/"},
				{"title": "카약 항공권", "url": "https://www.kayak.co.kr/flights"},
				naverSearch,
			}
		}
		if isTrain {
			return []map[string]string{
				{"title": "코레일 기차 예매", "url": "https://www.letskorail.com/"},
				{"title": "SRT 수서고속철도", "url": "https://etk.srail.kr/"},
				naverSearch,
			}
		}
		if isSubway {
			return []map[string]string{
				{"title": "네이버 지도 경로", "url": "https://map.naver.com/v5/directions/-/-/-/transit?query=" + enc},
				{"title": "카카오맵 대중교통", "url": "https://map.kakao.com/?q=" + enc},
				{"title": "서울 지하철 노선도", "url": "https://www.seoulmetro.co.kr/kr/cyberStation.do"},
			}
		}
		if isTaxi {
			return []map[string]string{
				{"title": "카카오T 택시", "url": "https://www.kakaomobility.com/"},
				{"title": "네이버 지도 경로", "url": "https://map.naver.com/v5/directions/-/-/-/car?query=" + enc},
			}
		}
		return []map[string]string{
			{"title": "버스타고 (시외/고속버스)", "url": "https://www.bustago.or.kr/"},
			{"title": "코버스 고속버스 예매", "url": "https://www.kobus.co.kr/"},
			{"title": "네이버 지도 경로", "url": "https://map.naver.com/v5/directions/-/-/-/transit?query=" + enc},
			{"title": "카카오맵 경로", "url": "https://map.kakao.com/?q=" + enc},
			{"title": "코레일 기차 예매", "url": "https://www.letskorail.com/"},
		}
	case catFood:
		return []map[string]string{
			{"title": "네이버 지도 맛집: " + query, "url": "https://map.naver.com/v5/search/" + enc},
			{"title": "카카오맵 음식점: " + query, "url": "https://map.kakao.com/?q=" + enc},
			{"title": "망고플레이트: " + query, "url": "https://www.mangoplate.com/search/" + enc},
			{"title": "배달의민족", "url": "https://www.baemin.com/"},
			{"title": "요기요", "url": "https://www.yogiyo.co.kr/"},
		}
	case catShopping:
		return []map[string]string{
			{"title": "쿠팡: " + query, "url": "https://www.coupang.com/np/search?q=" + enc},
			{"title": "네이버쇼핑: " + query, "url": "https://search.shopping.naver.com/search/all?query=" + enc},
			{"title": "다나와 가격비교: " + query, "url": "https://search.danawa.com/dsearch.php?query=" + enc},
			{"title": "지마켓: " + query, "url": "https://search.gmarket.co.kr/search?keyword=" + enc},
			{"title": "11번가: " + query, "url": "https://search.11st.co.kr/Search.tmall?kwd=" + enc},
		}
	case catFinance:
		isStock := strings.Contains(lower, "주가") || strings.Contains(lower, "주식") || strings.Contains(lower, "코스피") || strings.Contains(lower, "나스닥")
		isCrypto := strings.Contains(lower, "비트코인") || strings.Contains(lower, "코인") || strings.Contains(lower, "이더리움")
		isExchange := strings.Contains(lower, "환율") || strings.Contains(lower, "달러") || strings.Contains(lower, "엔화") || strings.Contains(lower, "유로")
		if isCrypto {
			return []map[string]string{
				{"title": "업비트 코인 시세", "url": "https://upbit.com/"},
				{"title": "빗썸 코인 시세", "url": "https://www.bithumb.com/"},
				naverSearch,
			}
		}
		if isStock {
			return []map[string]string{
				{"title": "네이버 금융 주식: " + query, "url": "https://finance.naver.com/search/searchResult.naver?query=" + enc},
				{"title": "한국거래소(KRX)", "url": "https://www.krx.co.kr/"},
				naverSearch,
			}
		}
		if isExchange {
			return []map[string]string{
				{"title": "네이버 환율", "url": "https://finance.naver.com/marketindex/"},
				{"title": "한국은행 환율", "url": "https://www.bok.or.kr/portal/main/contents.do?menuNo=200644"},
				naverSearch,
			}
		}
		return []map[string]string{
			{"title": "네이버 금융: " + query, "url": "https://finance.naver.com/search/searchResult.naver?query=" + enc},
			naverSearch,
		}
	case catWeather:
		return []map[string]string{
			{"title": "기상청 날씨: " + query, "url": "https://www.weather.go.kr/w/index.do"},
			{"title": "네이버 날씨: " + query, "url": "https://search.naver.com/search.naver?query=" + enc + "+날씨"},
			{"title": "케이웨더", "url": "https://www.kweather.co.kr/"},
		}
	case catNews:
		return []map[string]string{
			{"title": "네이버 뉴스: " + query, "url": "https://search.naver.com/search.naver?where=news&query=" + enc},
			{"title": "다음 뉴스: " + query, "url": "https://search.daum.net/search?w=news&q=" + enc},
			{"title": "구글 뉴스: " + query, "url": "https://news.google.com/search?q=" + enc + "&hl=ko"},
		}
	case catMedical:
		return []map[string]string{
			{"title": "건강보험공단 건강정보", "url": "https://www.nhis.or.kr/nhis/healthin/retrieveHealthInfoMain.do"},
			{"title": "서울아산병원 건강정보: " + query, "url": "https://www.amc.seoul.kr/asan/healthinfo/searchHealthInfo.do?searchText=" + enc},
			{"title": "네이버 지도 병원: " + query, "url": "https://map.naver.com/v5/search/" + enc + "+병원"},
			{"title": "약학정보원 (약 정보)", "url": "https://www.health.kr/"},
			naverSearch,
		}
	case catLegal:
		return []map[string]string{
			{"title": "정부24 민원: " + query, "url": "https://www.gov.kr/search?srchWord=" + enc},
			{"title": "국세청 홈택스", "url": "https://www.hometax.go.kr/"},
			{"title": "법제처 국가법령정보", "url": "https://www.law.go.kr/LSW/lsSc.do?menuId=1&query=" + enc},
			{"title": "건강보험공단", "url": "https://www.nhis.or.kr/"},
			naverSearch,
		}
	case catEntertainment:
		isMovie := strings.Contains(lower, "영화") || strings.Contains(lower, "cgv") || strings.Contains(lower, "롯데시네마") || strings.Contains(lower, "메가박스")
		isBasebell := strings.Contains(lower, "야구") || strings.Contains(lower, "kbo")
		isSoccer := strings.Contains(lower, "축구") || strings.Contains(lower, "k리그") || strings.Contains(lower, "월드컵")
		isConcert := strings.Contains(lower, "공연") || strings.Contains(lower, "뮤지컬") || strings.Contains(lower, "콘서트")
		if isMovie {
			return []map[string]string{
				{"title": "CGV 상영시간표", "url": "https://www.cgv.co.kr/movies/"},
				{"title": "롯데시네마 시간표", "url": "https://www.lottecinema.co.kr/NLCHS/Movie/MovieList"},
				{"title": "메가박스 시간표", "url": "https://www.megabox.co.kr/movie"},
				naverSearch,
			}
		}
		if isBasebell {
			return []map[string]string{
				{"title": "KBO 야구 일정/결과", "url": "https://www.koreabaseball.com/"},
				{"title": "네이버 스포츠 야구", "url": "https://sports.naver.com/kbaseball/index"},
			}
		}
		if isSoccer {
			return []map[string]string{
				{"title": "K리그 공식사이트", "url": "https://www.kleague.com/"},
				{"title": "네이버 스포츠 축구", "url": "https://sports.naver.com/football/index"},
			}
		}
		if isConcert {
			return []map[string]string{
				{"title": "인터파크 티켓: " + query, "url": "https://tickets.interpark.com/search?keyword=" + enc},
				{"title": "YES24 공연: " + query, "url": "https://ticket.yes24.com/Pages/Perf/PerfList.aspx"},
				{"title": "멜론 티켓", "url": "https://ticket.melon.com/"},
			}
		}
		return []map[string]string{
			{"title": "네이버 스포츠", "url": "https://sports.naver.com/"},
			naverSearch,
		}
	case catRecipe:
		return []map[string]string{
			{"title": "만개의레시피: " + query, "url": "https://www.10000recipe.com/recipe/list.html?q=" + enc},
			{"title": "해먹남녀: " + query, "url": "https://haemuknam.com/"},
			{"title": "유튜브 레시피: " + query, "url": "https://www.youtube.com/results?search_query=" + enc + "+레시피"},
			naverSearch,
		}
	case catTravel:
		return []map[string]string{
			{"title": "네이버 여행: " + query, "url": "https://search.naver.com/search.naver?query=" + enc},
			{"title": "야놀자: " + query, "url": "https://www.yanolja.com/"},
			{"title": "여기어때: " + query, "url": "https://www.yeogi.com/"},
			{"title": "에어비앤비: " + query, "url": "https://www.airbnb.co.kr/s/" + enc},
			{"title": "한국관광공사", "url": "https://www.visitkorea.or.kr/"},
		}
	case catTech:
		return []map[string]string{
			{"title": "구글 검색: " + query, "url": "https://www.google.com/search?q=" + enc},
			{"title": "스택오버플로우: " + query, "url": "https://stackoverflow.com/search?q=" + enc},
			{"title": "나무위키: " + query, "url": "https://namu.wiki/w/" + enc},
			naverSearch,
		}
	case catEducation:
		return []map[string]string{
			{"title": "큐넷 자격증: " + query, "url": "https://www.q-net.or.kr/crf005.do?id=crf00503&jmCd=&jmNm=" + enc},
			{"title": "EBS 강의: " + query, "url": "https://www.ebs.co.kr/search?query=" + enc},
			{"title": "네이버 지식IN: " + query, "url": "https://kin.naver.com/search/list.naver?query=" + enc},
			naverSearch,
		}
	case catRealEstate:
		return []map[string]string{
			{"title": "네이버 부동산: " + query, "url": "https://land.naver.com/"},
			{"title": "직방: " + query, "url": "https://www.zigbang.com/"},
			{"title": "호갱노노: " + query, "url": "https://hogangnono.com/"},
			{"title": "청약홈", "url": "https://www.applyhome.co.kr/"},
			naverSearch,
		}
	default:
		return []map[string]string{
			naverSearch,
			googleSearch,
			{"title": "다음 검색: " + query, "url": "https://search.daum.net/search?q=" + enc},
		}
	}
}

// parallelSearchResult: 병렬 수집 결과 묶음
type parallelSearchResult struct {
	Summary string
	Items   []map[string]string // {title, url, source}
}

// ── 5분 TTL 인메모리 캐시 ────────────────────────────────────────
type searchCacheEntry struct {
	result    parallelSearchResult
	expiresAt time.Time
}

var (
	searchCacheMu sync.RWMutex
	searchCache   = map[string]searchCacheEntry{}
)

func getCachedSearch(query string) (parallelSearchResult, bool) {
	searchCacheMu.RLock()
	defer searchCacheMu.RUnlock()
	e, ok := searchCache[query]
	if !ok || time.Now().After(e.expiresAt) {
		return parallelSearchResult{}, false
	}
	return e.result, true
}

func setCachedSearch(query string, r parallelSearchResult) {
	searchCacheMu.Lock()
	defer searchCacheMu.Unlock()
	searchCache[query] = searchCacheEntry{result: r, expiresAt: time.Now().Add(5 * time.Minute)}
	// 캐시 항목이 100개 초과 시 만료된 것 정리
	if len(searchCache) > 100 {
		now := time.Now()
		for k, v := range searchCache {
			if now.After(v.expiresAt) {
				delete(searchCache, k)
			}
		}
	}
}

// parallelWebSearch: Tavily + 브라우저(chromedp) 를 goroutine으로 동시 실행
// 각 소스 결과를 channel로 취합 → 중복 URL 제거 후 반환
func parallelWebSearch(query string, maxItems int) parallelSearchResult {
	if cached, ok := getCachedSearch(query); ok {
		return cached
	}

	// 카테고리 감지 + 쿼리 최적화
	cat := detectCategory(query)
	optimized := optimizeQuery(query, cat)
	type srcResult struct {
		source  string
		summary string
		items   []map[string]string
	}

	llmMu.RLock()
	tKey := llmTavilyKey
	gKey := llmPerplexityKey
	llmMu.RUnlock()

	ch := make(chan srcResult, 4)
	var wg sync.WaitGroup

	// ── 소스 1: Tavily 실시간 검색 (최적화 쿼리 사용) ──────────
	if tKey != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if r, ok := tavilySearch(tKey, optimized, maxItems); ok {
				ch <- srcResult{source: "tavily", summary: r.Summary, items: r.Items}
			}
		}()
	}

	// ── 소스 2: 플랫폼별 브라우저 크롤링 (최적화 쿼리 사용) ───
	wg.Add(1)
	go func() {
		defer wg.Done()
		items := browserParallelScrape(optimized, maxItems)
		if len(items) > 0 {
			ch <- srcResult{source: "browser", items: items}
		}
	}()

	// ── 모든 goroutine 완료 후 channel 닫기 ─────────────────────
	go func() {
		wg.Wait()
		close(ch)
	}()

	// ── 결과 수집 + URL 중복 제거 ────────────────────────────────
	seen := map[string]bool{}
	var merged []map[string]string
	var summaries []string

	for r := range ch {
		if r.summary != "" {
			summaries = append(summaries, r.summary) // 출처명 접두사 없이 내용만
		}
		for _, item := range r.items {
			url := item["url"]
			if url == "" || seen[url] {
				continue
			}
			seen[url] = true
			item["source"] = r.source
			merged = append(merged, item)
			if len(merged) >= maxItems*2 {
				break
			}
		}
	}

	// 제목만 추출 (URL·원문 절대 Groq에 전달 안 함)
	titleLines := make([]string, 0, len(merged))
	for _, it := range merged {
		if t := it["title"]; t != "" {
			titleLines = append(titleLines, "• "+t)
		}
	}
	// Tavily answer가 있으면 우선 사용 (이미 정제된 자연어)
	tavilySummary := ""
	for _, s := range summaries {
		if s != "" {
			tavilySummary = s
			break
		}
	}

	// 카테고리별 공식 사이트 안내 문구
	officialSiteHint := buildOfficialSiteHint(cat)

	var summary string
	if gKey != "" {
		today := time.Now().Format("2006-01-02")
		context := strings.Join(titleLines, "\n")
		if context == "" {
			context = "(검색 결과 없음)"
		}
		tavilyHint := ""
		if tavilySummary != "" {
			hint := tavilySummary
			if len([]rune(hint)) > 300 {
				hint = string([]rune(hint)[:300])
			}
			tavilyHint = "\nTavily 요약: " + hint
		}
		sysMsg := fmt.Sprintf(`당신은 Nexus AI 한국어 비서입니다.

[최우선 규칙]
1. 검색 결과를 바탕으로 자연스러운 한국어로 답하세요
2. URL, 링크, 출처명([tavily] 등) 절대 포함 금지
3. 마크다운 헤더(##), 과도한 불릿 금지
4. 검색 결과가 부족하더라도 절대 "모른다", "찾지 못했습니다"로 끝내지 마세요
5. 결과가 없으면 → 카테고리에 맞는 공식 사이트/방법을 안내하세요

[카테고리별 결과 부족 시 안내]
%s

[답변 형식]
- 검색 결과 있을 때: 핵심 정보를 2~4문장으로 자연스럽게 요약
- 검색 결과 없을 때: "~에서 확인하실 수 있습니다" 형식으로 공식 경로 안내
- 구체적 수치(가격, 시간, 요금 등) 있으면 반드시 포함
- 복잡한 질문은 4~6문장까지 허용

[절대 금지]
- "정확한 정보를 찾지 못했습니다. 미리보기 버튼으로 직접 확인해보세요." 이 문구 사용 금지
- "모릅니다", "알 수 없습니다" 로 끝내는 것 금지`, officialSiteHint)

		userMsg := fmt.Sprintf("오늘: %s\n사용자 질문: \"%s\"\n최적화 검색어: \"%s\"\n검색 결과 제목:\n%s%s\n\n위 정보를 바탕으로 답하되, 결과가 부족하면 공식 사이트 안내로 마무리하세요.", today, query, optimized, context, tavilyHint)
		msgs := []groqMsg{
			{Role: "system", Content: sysMsg},
			{Role: "user", Content: userMsg},
		}
		refined, _, err := callGroq(gKey, groqChatModel, msgs, 600, false)
		if err == nil && refined != "" {
			summary = refined
		}
	}

	// 3단 폴백: Groq 실패 시
	if summary == "" {
		if tavilySummary != "" {
			// 1순위: Tavily raw 요약 그대로 사용
			summary = tavilySummary
		} else if len(merged) > 0 {
			// 2순위: 검색 결과 개수 + 카테고리 안내
			summary = fmt.Sprintf("관련 정보 %d건을 찾았습니다. 아래 링크에서 확인해보세요.", len(merged))
		} else {
			// 3순위: 카테고리별 공식 사이트 안내
			summary = buildNoResultMessage(query, cat, tKey)
		}
	}

	// 검색 결과 없을 때 카테고리별 폴백 사이트 링크 주입
	if len(merged) == 0 {
		merged = categoryFallbackSites(query, cat)
	}

	// 최대 maxItems 개만 반환
	if len(merged) > maxItems {
		merged = merged[:maxItems]
	}

	result := parallelSearchResult{Summary: summary, Items: merged}
	setCachedSearch(query, result)
	return result
}

// buildOfficialSiteHint: Groq 프롬프트에 삽입할 카테고리별 안내 문구
func buildOfficialSiteHint(cat queryCategory) string {
	switch cat {
	case catTransit:
		return `교통/버스/기차/항공:
- 고속버스: "버스타고(bustago.or.kr) 또는 코버스(kobus.co.kr)에서 예매하세요"
- KTX/기차: "코레일(letskorail.com) 또는 SRT(srail.kr)에서 검색하세요"
- 지하철: "네이버 지도나 카카오맵에서 경로를 검색해보세요"
- 항공: "네이버 항공권(flight.naver.com) 또는 스카이스캐너에서 비교하세요"
- 택시: "카카오T 앱에서 요금 확인이 가능합니다"`
	case catFood:
		return `맛집/음식:
- "네이버 지도 또는 카카오맵에서 해당 지역 음식점을 검색해보세요"
- "배달은 배달의민족(baemin.com) 또는 요기요에서 확인하세요"
- "망고플레이트에서 리뷰와 평점을 볼 수 있습니다"`
	case catShopping:
		return `쇼핑/가격비교:
- "쿠팡, 네이버쇼핑, 다나와에서 가격을 비교해보세요"
- "다나와(danawa.com)는 전자제품 최저가 비교에 특화되어 있습니다"`
	case catFinance:
		return `금융/환율/주가:
- 환율: "네이버 금융(finance.naver.com/marketindex)에서 실시간 환율을 확인하세요"
- 주식: "네이버 금융 또는 증권사 앱에서 실시간 주가를 확인하세요"
- 코인: "업비트(upbit.com) 또는 빗썸(bithumb.com)에서 확인하세요"`
	case catWeather:
		return `날씨:
- "기상청(weather.go.kr) 또는 네이버 날씨에서 정확한 예보를 확인하세요"`
	case catNews:
		return `뉴스:
- "네이버 뉴스(news.naver.com) 또는 다음 뉴스에서 최신 기사를 확인하세요"`
	case catMedical:
		return `의료/건강/병원:
- "근처 병원은 네이버 지도에서 '병원' 검색으로 찾으세요"
- "증상 정보는 서울아산병원 건강정보(amc.seoul.kr) 또는 건강보험공단 사이트를 참고하세요"
- "약 정보는 약학정보원(health.kr)에서 확인하세요"`
	case catLegal:
		return `법률/행정/세금:
- "민원 서비스는 정부24(gov.kr)에서 처리 가능합니다"
- "세금 신고는 국세청 홈택스(hometax.go.kr)를 이용하세요"
- "법령 정보는 법제처 국가법령정보센터(law.go.kr)에서 확인하세요"`
	case catEntertainment:
		return `영화/공연/스포츠:
- 영화: "CGV, 롯데시네마, 메가박스 앱에서 시간표와 예매가 가능합니다"
- 공연: "인터파크 티켓(tickets.interpark.com) 또는 YES24에서 예매하세요"
- 야구: "KBO 공식사이트(koreabaseball.com)에서 일정과 결과를 확인하세요"
- 축구: "K리그 공식사이트(kleague.com)에서 확인하세요"`
	case catRecipe:
		return `요리/레시피:
- "만개의레시피(10000recipe.com)에서 레시피를 검색해보세요"
- "유튜브에서 요리 영상도 많이 찾을 수 있습니다"`
	case catTravel:
		return `여행/숙박:
- "야놀자(yanolja.com) 또는 여기어때에서 숙소를 예약하세요"
- "한국관광공사(visitkorea.or.kr)에서 국내 여행지 정보를 확인하세요"`
	case catRealEstate:
		return `부동산:
- "네이버 부동산(land.naver.com), 직방, 호갱노노에서 시세를 확인하세요"
- "청약은 청약홈(applyhome.co.kr)에서 신청 가능합니다"`
	default:
		return `결과 부족 시:
- "네이버 또는 구글에서 더 구체적인 키워드로 검색해보세요"
- 가능하면 지역명, 날짜, 브랜드명 등을 추가해서 다시 질문해주세요`
	}
}

// buildNoResultMessage: 검색 결과 완전 없을 때 카테고리별 안내 메시지
func buildNoResultMessage(query string, cat queryCategory, tKey string) string {
	if tKey == "" {
		return "Tavily API 키가 설정되지 않아 실시간 검색이 제한됩니다. 설정 → API 키에서 Tavily 키를 입력하면 훨씬 정확한 검색이 가능합니다."
	}
	switch cat {
	case catTransit:
		lower := strings.ToLower(query)
		if strings.Contains(lower, "ktx") || strings.Contains(lower, "기차") || strings.Contains(lower, "열차") {
			return "기차/KTX 시간표와 예매는 코레일(letskorail.com) 또는 SRT(srail.kr) 앱에서 확인하세요."
		}
		if strings.Contains(lower, "항공") || strings.Contains(lower, "비행기") {
			return "항공권은 네이버 항공권(flight.naver.com) 또는 스카이스캐너에서 비교 검색하세요."
		}
		if strings.Contains(lower, "지하철") || strings.Contains(lower, "전철") {
			return "지하철 경로는 카카오맵 또는 네이버 지도 앱에서 출발지와 도착지를 입력하면 정확한 시간표와 환승 정보를 볼 수 있습니다."
		}
		return "고속버스/시외버스 시간표와 예매는 버스타고(bustago.or.kr) 또는 코버스(kobus.co.kr)에서 출발지와 도착지를 검색하세요."
	case catFood:
		return fmt.Sprintf("'%s' 관련 맛집은 네이버 지도 또는 카카오맵에서 검색하시면 리뷰와 위치를 함께 볼 수 있습니다.", query)
	case catShopping:
		return fmt.Sprintf("'%s' 가격 비교는 쿠팡, 네이버쇼핑, 다나와(danawa.com)에서 확인해보세요.", query)
	case catFinance:
		return "실시간 금융 정보는 네이버 금융(finance.naver.com), 주식은 각 증권사 앱, 코인은 업비트·빗썸에서 확인하세요."
	case catWeather:
		return fmt.Sprintf("'%s' 날씨는 기상청(weather.go.kr) 또는 네이버 날씨에서 정확한 예보를 확인하세요.", query)
	case catNews:
		return fmt.Sprintf("'%s' 관련 뉴스는 네이버 뉴스(news.naver.com) 또는 구글 뉴스에서 검색해보세요.", query)
	case catMedical:
		return fmt.Sprintf("'%s' 관련 의료 정보는 서울아산병원 건강정보(amc.seoul.kr) 또는 건강보험공단 사이트를 참고하고, 정확한 진단은 전문의와 상담하세요.", query)
	case catLegal:
		return "행정 민원은 정부24(gov.kr), 세금은 홈택스(hometax.go.kr), 법령 확인은 법제처(law.go.kr)를 이용하세요."
	case catEntertainment:
		lower := strings.ToLower(query)
		if strings.Contains(lower, "영화") {
			return "영화 시간표와 예매는 CGV, 롯데시네마, 메가박스 앱에서 확인하세요."
		}
		if strings.Contains(lower, "공연") || strings.Contains(lower, "뮤지컬") || strings.Contains(lower, "콘서트") {
			return "공연/콘서트 예매는 인터파크 티켓(tickets.interpark.com) 또는 YES24에서 확인하세요."
		}
		return fmt.Sprintf("'%s' 관련 정보는 네이버 스포츠 또는 해당 리그 공식 사이트에서 확인하세요.", query)
	case catRecipe:
		return fmt.Sprintf("'%s' 레시피는 만개의레시피(10000recipe.com)에서 검색하거나 유튜브에서 요리 영상을 찾아보세요.", query)
	case catTravel:
		return fmt.Sprintf("'%s' 여행 정보는 한국관광공사(visitkorea.or.kr), 숙소 예약은 야놀자 또는 여기어때에서 확인하세요.", query)
	case catRealEstate:
		return "부동산 시세는 네이버 부동산(land.naver.com), 직방, 호갱노노에서 확인하고, 청약은 청약홈(applyhome.co.kr)을 이용하세요."
	default:
		return fmt.Sprintf("'%s'에 대한 검색 결과를 찾지 못했습니다. 네이버나 구글에서 더 구체적인 키워드로 검색해보세요.", query)
	}
}
