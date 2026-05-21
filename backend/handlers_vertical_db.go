//go:build windows

package main

// ══════════════════════════════════════════════════════════════
//  직업군별 전문 데이터 연동
//
//  우선순위:
//   1. 키 불필요 공개 API (open.er-api.com, HackerNews, GitHub)
//   2. 정부 RSS 피드 (law.go.kr, nts.go.kr, moe.go.kr 등)
//   3. 공공데이터포털 API (~/.nexus/vertical_apis.json 키 설정 시)
//   4. Tavily 도메인 한정 검색 (항상 동작하는 폴백)
//   5. chromedp 스텔스 크롤링 (YouTube, GitHub trending)
//
//  공공데이터포털 키 발급(무료): https://www.data.go.kr
//  법제처 API 키 발급(무료): https://open.law.go.kr
// ══════════════════════════════════════════════════════════════

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

// ─────────────────────────────────────────────────────────────
//  API 키 설정 (~/.nexus/vertical_apis.json)
// ─────────────────────────────────────────────────────────────

type VerticalAPIKeys struct {
	// 법제처 국가법령정보 오픈API — 무료, https://open.law.go.kr 회원가입 후 발급
	LawGOKR string `json:"law_go_kr"`
	// 공공데이터포털 공통 인증키 — 무료, https://www.data.go.kr 회원가입 후 발급
	// 사용처: 의약품API, 부동산실거래가API, 워크넷구인정보API
	DataGOKR string `json:"data_go_kr"`
	// YouTube Data API v3 — 무료 10,000유닛/일, Google Cloud Console에서 발급
	YouTubeV3 string `json:"youtube_v3"`
	// GitHub Personal Access Token — 선택사항, 없으면 60req/hr (있으면 5000req/hr)
	GitHubToken string `json:"github_token"`
}

func loadVerticalAPIKeys() VerticalAPIKeys {
	home, _ := os.UserHomeDir()
	data, err := os.ReadFile(filepath.Join(home, ".nexus", "vertical_apis.json"))
	if err != nil {
		return VerticalAPIKeys{}
	}
	var keys VerticalAPIKeys
	json.Unmarshal(data, &keys)
	return keys
}

func saveVerticalAPIKeysTemplate() {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".nexus")
	os.MkdirAll(dir, 0755)
	path := filepath.Join(dir, "vertical_apis.json")
	if _, err := os.Stat(path); err == nil {
		return // 이미 존재
	}
	template := VerticalAPIKeys{}
	data, _ := json.MarshalIndent(template, "", "  ")
	os.WriteFile(path, data, 0600)
}

// ─────────────────────────────────────────────────────────────
//  공통 HTTP 클라이언트
// ─────────────────────────────────────────────────────────────

func httpGetTimeout(url string, timeout time.Duration) ([]byte, error) {
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// ─────────────────────────────────────────────────────────────
//  RSS 피드 파서 (정부 기관 공지 수집)
// ─────────────────────────────────────────────────────────────

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

type rssFeed struct {
	Channel struct {
		Title string    `xml:"title"`
		Items []rssItem `xml:"item"`
	} `xml:"channel"`
}

func fetchRSSFeed(feedURL string, limit int) ([]rssItem, error) {
	body, err := httpGetTimeout(feedURL, 8*time.Second)
	if err != nil {
		return nil, err
	}
	var feed rssFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, err
	}
	items := feed.Channel.Items
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func rssToLines(items []rssItem) string {
	if len(items) == 0 {
		return ""
	}
	var lines []string
	for _, item := range items {
		title := strings.TrimSpace(item.Title)
		title = strings.ReplaceAll(title, "<![CDATA[", "")
		title = strings.ReplaceAll(title, "]]>", "")
		if title != "" {
			lines = append(lines, "• "+title)
		}
	}
	return strings.Join(lines, "\n")
}

// ─────────────────────────────────────────────────────────────
//  환율 (open.er-api.com — 완전 무료, API 키 불필요)
// ─────────────────────────────────────────────────────────────

func fetchExchangeRates() string {
	body, err := httpGetTimeout("https://open.er-api.com/v6/latest/USD", 8*time.Second)
	if err != nil {
		return tavilyFallbackSingle("오늘 원달러 원유로 환율")
	}
	var data struct {
		Result string             `json:"result"`
		Rates  map[string]float64 `json:"rates"`
	}
	if err := json.Unmarshal(body, &data); err != nil || data.Result != "success" {
		return tavilyFallbackSingle("오늘 원달러 원유로 환율")
	}
	krw := data.Rates["KRW"]
	eur := data.Rates["EUR"]
	jpy := data.Rates["JPY"]
	cny := data.Rates["CNY"]
	var lines []string
	if krw > 0 {
		lines = append(lines, fmt.Sprintf("💵 USD/KRW: ₩%.0f", krw))
	}
	if eur > 0 && krw > 0 {
		lines = append(lines, fmt.Sprintf("💶 EUR/KRW: ₩%.0f", krw/eur))
	}
	if jpy > 0 && krw > 0 {
		lines = append(lines, fmt.Sprintf("💴 JPY/KRW: ₩%.2f", krw/jpy*100))
	}
	if cny > 0 && krw > 0 {
		lines = append(lines, fmt.Sprintf("🇨🇳 CNY/KRW: ₩%.0f", krw/cny))
	}
	if len(lines) == 0 {
		return "환율 정보 없음"
	}
	return strings.Join(lines, "\n")
}

// ─────────────────────────────────────────────────────────────
//  HackerNews (Firebase JSON API — 완전 무료, API 키 불필요)
// ─────────────────────────────────────────────────────────────

type hnItem struct {
	Title string `json:"title"`
	URL   string `json:"url"`
	Score int    `json:"score"`
	Type  string `json:"type"`
}

func fetchHackerNewsTop(limit int) string {
	body, err := httpGetTimeout("https://hacker-news.firebaseio.com/v0/topstories.json", 10*time.Second)
	if err != nil {
		return tavilyFallbackLines("hacker news today top developer tech", limit)
	}
	var ids []int
	if err := json.Unmarshal(body, &ids); err != nil {
		return tavilyFallbackLines("hacker news today top developer tech", limit)
	}
	fetch := limit * 2
	if len(ids) > fetch {
		ids = ids[:fetch]
	}
	client := &http.Client{Timeout: 5 * time.Second}
	var items []hnItem
	for _, id := range ids {
		if len(items) >= limit {
			break
		}
		resp, err := client.Get(fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json", id))
		if err != nil {
			continue
		}
		var item hnItem
		json.NewDecoder(resp.Body).Decode(&item)
		resp.Body.Close()
		if item.Title != "" && item.Type == "story" {
			items = append(items, item)
		}
	}
	if len(items) == 0 {
		return tavilyFallbackLines("hacker news today top tech news", limit)
	}
	var lines []string
	for _, item := range items {
		lines = append(lines, fmt.Sprintf("• %s (↑%d)", item.Title, item.Score))
	}
	return strings.Join(lines, "\n")
}

// ─────────────────────────────────────────────────────────────
//  GitHub API (인증 없이 60req/hr — 토큰 있으면 5000req/hr)
// ─────────────────────────────────────────────────────────────

type githubRepo struct {
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	Stars       int    `json:"stargazers_count"`
	Language    string `json:"language"`
	HTMLURL     string `json:"html_url"`
}

func fetchGitHubTrending(limit int) string {
	keys := loadVerticalAPIKeys()
	since := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	url := fmt.Sprintf("https://api.github.com/search/repositories?q=created:>%s&sort=stars&order=desc&per_page=%d", since, limit)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return tavilyFallbackLines("github trending today repositories stars", limit)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if keys.GitHubToken != "" {
		req.Header.Set("Authorization", "Bearer "+keys.GitHubToken)
	}

	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return scrapeGitHubTrendingPage(limit)
	}
	defer resp.Body.Close()

	var result struct {
		Items []githubRepo `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || len(result.Items) == 0 {
		return scrapeGitHubTrendingPage(limit)
	}

	var lines []string
	for _, r := range result.Items {
		lang := ""
		if r.Language != "" {
			lang = " [" + r.Language + "]"
		}
		desc := r.Description
		if len(desc) > 60 {
			desc = desc[:60] + "..."
		}
		lines = append(lines, fmt.Sprintf("⭐ %s%s — %s (★%d)", r.FullName, lang, desc, r.Stars))
	}
	return strings.Join(lines, "\n")
}

// GitHub Trending 페이지 chromedp 스크래핑 (API 실패 시 폴백)
func scrapeGitHubTrendingPage(limit int) string {
	ctx, cancel, err := withMobileStealthTimeout(30 * time.Second)
	if err != nil {
		return tavilyFallbackLines("github trending repositories today stars", limit)
	}
	defer cancel()

	var repoTexts []string
	scrapeErr := chromedp.Run(ctx,
		chromedp.Navigate("https://github.com/trending"),
		chromedp.WaitVisible("article.Box-row", chromedp.ByQuery),
		chromedp.Evaluate(fmt.Sprintf(`
			Array.from(document.querySelectorAll('article.Box-row')).slice(0,%d).map(el => {
				const name = el.querySelector('h2 a')?.textContent?.trim()?.replace(/\s+/g,' ') ?? '';
				const desc = el.querySelector('p')?.textContent?.trim() ?? '';
				const stars = el.querySelector('[aria-label*="star"]')?.textContent?.trim() ?? '';
				return name + (desc ? ' — ' + desc.slice(0,50) : '') + (stars ? ' ★'+stars : '');
			})
		`, limit), &repoTexts),
	)
	if scrapeErr != nil || len(repoTexts) == 0 {
		return tavilyFallbackLines("github trending repositories today stars", limit)
	}
	var lines []string
	for _, t := range repoTexts {
		if t != "" {
			lines = append(lines, "• "+t)
		}
	}
	return strings.Join(lines, "\n")
}

// ─────────────────────────────────────────────────────────────
//  YouTube 트렌딩 (chromedp 스크래핑 — 한국 지역)
// ─────────────────────────────────────────────────────────────

func scrapeYouTubeTrending(limit int) string {
	ctx, cancel, err := withMobileStealthTimeout(35 * time.Second)
	if err != nil {
		return tavilyFallbackLines("유튜브 트렌딩 인기 동영상 한국", limit)
	}
	defer cancel()

	var titles []string
	scrapeErr := chromedp.Run(ctx,
		chromedp.Navigate("https://www.youtube.com/feed/trending?gl=KR&hl=ko"),
		chromedp.Sleep(3*time.Second),
		chromedp.Evaluate(fmt.Sprintf(`
			Array.from(document.querySelectorAll('#video-title, ytd-video-renderer #video-title'))
				.slice(0,%d)
				.map(e => e.textContent?.trim())
				.filter(Boolean)
		`, limit), &titles),
	)
	if scrapeErr != nil || len(titles) == 0 {
		return tavilyFallbackLines("오늘 유튜브 트렌딩 인기 급상승 동영상", limit)
	}
	var lines []string
	for _, t := range titles {
		if t != "" {
			lines = append(lines, "• "+t)
		}
	}
	return strings.Join(lines, "\n")
}

// ─────────────────────────────────────────────────────────────
//  법령 개정 정보 (법제처 오픈API 키 → 연합뉴스 사회 RSS → Tavily)
// ─────────────────────────────────────────────────────────────

func fetchRecentLawAmendments() string {
	// 1순위: 법제처 오픈API (키 필요 — open.law.go.kr 무료 발급)
	keys := loadVerticalAPIKeys()
	if keys.LawGOKR != "" {
		url := fmt.Sprintf(
			"https://www.law.go.kr/DRF/lawSearch.do?OC=%s&target=lsr&type=JSON&query=개정&display=5&page=1",
			keys.LawGOKR,
		)
		body, err := httpGetTimeout(url, 8*time.Second)
		if err == nil {
			var result struct {
				LawSearch struct {
					Law []struct {
						LawName  string `json:"법령명한글"`
						ProcDate string `json:"공포일자"`
					} `json:"law"`
				} `json:"LawSearch"`
			}
			if json.Unmarshal(body, &result) == nil && len(result.LawSearch.Law) > 0 {
				var lines []string
				for _, l := range result.LawSearch.Law {
					lines = append(lines, fmt.Sprintf("• %s (공포: %s)", l.LawName, l.ProcDate))
				}
				return strings.Join(lines, "\n")
			}
		}
	}
	// 2순위: Tavily 법령 검색
	return tavilyFallbackLines("최근 법령 개정 시행 "+time.Now().Format("2006년"), 3)
}

// ─────────────────────────────────────────────────────────────
//  법원·판례 뉴스 (연합뉴스 사회 RSS — 동작 확인됨)
// ─────────────────────────────────────────────────────────────

func fetchSupremeCourtNews() string {
	// 연합뉴스 사회 RSS (대법원/법원 뉴스 포함, 동작 확인)
	items, err := fetchRSSFeed("https://www.yna.co.kr/rss/society.xml", 8)
	if err == nil && len(items) > 0 {
		// 법률/판례 관련 필터
		var lawItems []rssItem
		for _, item := range items {
			t := strings.ToLower(item.Title)
			if strings.ContainsAny(t, "법") || strings.Contains(t, "판결") ||
				strings.Contains(t, "대법") || strings.Contains(t, "헌재") ||
				strings.Contains(t, "재판") || strings.Contains(t, "판례") {
				lawItems = append(lawItems, item)
			}
		}
		if len(lawItems) == 0 {
			lawItems = items[:min(3, len(items))]
		}
		result := rssToLines(lawItems[:min(3, len(lawItems))])
		if result != "" {
			return result
		}
	}
	return tavilyFallbackLines("오늘 법률 판례 대법원 헌법재판소 뉴스", 3)
}

// ─────────────────────────────────────────────────────────────
//  의료 뉴스 (청년의사 RSS — 동작 확인됨)
// ─────────────────────────────────────────────────────────────

func fetchMedicalNews() string {
	// 1순위: 청년의사 RSS (동작 확인)
	items, err := fetchRSSFeed("https://www.docdocdoc.co.kr/rss/allArticle.xml", 5)
	if err == nil && len(items) > 0 {
		result := rssToLines(items)
		if result != "" {
			return result
		}
	}
	// 2순위: 헬스조선 RSS (동작 확인)
	items, err = fetchRSSFeed("https://health.chosun.com/site/data/rss/rss.xml", 5)
	if err == nil && len(items) > 0 {
		result := rssToLines(items)
		if result != "" {
			return result
		}
	}
	return tavilyFallbackLines("오늘 의학 임상 신약 의료 뉴스", 3)
}

// ─────────────────────────────────────────────────────────────
//  의약품 정보 (공공데이터포털 키 → 청년의사 RSS → Tavily)
// ─────────────────────────────────────────────────────────────

func fetchDrugApprovalNews() string {
	keys := loadVerticalAPIKeys()
	if keys.DataGOKR != "" {
		url := fmt.Sprintf(
			"https://apis.data.go.kr/1471000/DrugPrdtPrmsnInfoService04/getDrugPrdtPrmsnDtlInq04?serviceKey=%s&pageNo=1&numOfRows=5&type=json",
			keys.DataGOKR,
		)
		body, err := httpGetTimeout(url, 8*time.Second)
		if err == nil {
			var result struct {
				Body struct {
					Items []struct {
						ItemName  string `json:"ITEM_NAME"`
						EntrpName string `json:"ENTP_NAME"`
					} `json:"items"`
				} `json:"body"`
			}
			if json.Unmarshal(body, &result) == nil && len(result.Body.Items) > 0 {
				var lines []string
				for _, item := range result.Body.Items {
					lines = append(lines, fmt.Sprintf("• %s (%s)", item.ItemName, item.EntrpName))
				}
				return "최근 신규 허가 의약품:\n" + strings.Join(lines, "\n")
			}
		}
	}
	return fetchMedicalNews()
}

// ─────────────────────────────────────────────────────────────
//  건강보험 변경 뉴스 (헬스조선 RSS → Tavily)
// ─────────────────────────────────────────────────────────────

func fetchHIRANews() string {
	// 헬스조선 RSS (동작 확인)
	items, err := fetchRSSFeed("https://health.chosun.com/site/data/rss/rss.xml", 5)
	if err == nil && len(items) > 0 {
		result := rssToLines(items)
		if result != "" {
			return result
		}
	}
	return tavilyFallbackLines("건강보험 급여 약제 변경 심사평가원 "+time.Now().Format("2006년"), 3)
}

// ─────────────────────────────────────────────────────────────
//  세무·회계 뉴스 (한국경제 경제 RSS → Tavily)
// ─────────────────────────────────────────────────────────────

func fetchNTSNews() string {
	// 한국경제 경제 RSS (동작 확인)
	items, err := fetchRSSFeed("https://www.hankyung.com/feed/economy", 8)
	if err == nil && len(items) > 0 {
		// 세무/회계 관련 필터
		var taxItems []rssItem
		for _, item := range items {
			t := strings.ToLower(item.Title)
			if strings.Contains(t, "세") || strings.Contains(t, "회계") ||
				strings.Contains(t, "세금") || strings.Contains(t, "국세") ||
				strings.Contains(t, "부가") || strings.Contains(t, "소득세") {
				taxItems = append(taxItems, item)
			}
		}
		if len(taxItems) == 0 {
			taxItems = items[:min(3, len(items))]
		}
		result := rssToLines(taxItems[:min(3, len(taxItems))])
		if result != "" {
			return result
		}
	}
	return tavilyFallbackLines("국세청 세무 회계 세법 개정 뉴스 "+time.Now().Format("2006년"), 3)
}

// ─────────────────────────────────────────────────────────────
//  부동산 실거래가 (국토교통부 — 키 필요, 없으면 Tavily)
// ─────────────────────────────────────────────────────────────

func fetchRealEstateNews() string {
	keys := loadVerticalAPIKeys()
	if keys.DataGOKR != "" {
		// 국토교통부 아파트 실거래가 API (공공데이터포털 키 필요)
		now := time.Now()
		url := fmt.Sprintf(
			"https://apis.data.go.kr/1613000/RTMSDataSvcAptTradeDev/getRTMSDataSvcAptTradeDev?serviceKey=%s&pageNo=1&numOfRows=5&DEAL_YMD=%s&LAWD_CD=11110",
			keys.DataGOKR, now.Format("200601"),
		)
		body, err := httpGetTimeout(url, 8*time.Second)
		if err == nil {
			var result struct {
				Body struct {
					Items struct {
						Item []struct {
							AptName  string `json:"아파트"`
							DealAmt  string `json:"거래금액"`
							Area     string `json:"전용면적"`
							DealDate string `json:"일"`
						} `json:"item"`
					} `json:"items"`
				} `json:"body"`
			}
			if json.Unmarshal(body, &result) == nil && len(result.Body.Items.Item) > 0 {
				var lines []string
				lines = append(lines, fmt.Sprintf("[%s 서울 종로구 실거래]", now.Format("2006.01")))
				for _, item := range result.Body.Items.Item {
					amt := strings.TrimSpace(item.DealAmt)
					lines = append(lines, fmt.Sprintf("• %s %.1f㎡ — %s만원", item.AptName, parseFloat(item.Area), amt))
				}
				return strings.Join(lines, "\n")
			}
		}
	}
	// 한국경제 부동산 RSS (동작 확인됨)
	items, err := fetchRSSFeed("https://www.hankyung.com/feed/realestate", 5)
	if err == nil && len(items) > 0 {
		result := rssToLines(items[:min(3, len(items))])
		if result != "" {
			return result
		}
	}
	return tavilyFallbackLines("오늘 부동산 아파트 시세 정책 뉴스", 3)
}

func parseFloat(s string) float64 {
	s = strings.TrimSpace(s)
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

// ─────────────────────────────────────────────────────────────
//  청약홈 정보 (Tavily 도메인 검색)
// ─────────────────────────────────────────────────────────────

func fetchApplyHomeSchedule() string {
	llmMu.RLock()
	k := llmTavilyKey
	llmMu.RUnlock()
	query := time.Now().Format("2006년 01월") + " 청약 분양 일정 아파트"
	res, ok := tavilySearchDomain(k, query, 5, "applyhome.co.kr")
	if !ok || len(res.Items) == 0 {
		res, ok = tavilySearch(k, query, 5)
		if !ok || len(res.Items) == 0 {
			return "청약 일정 수집 실패"
		}
	}
	var lines []string
	for _, item := range res.Items[:min(3, len(res.Items))] {
		lines = append(lines, "• "+item["title"])
	}
	return strings.Join(lines, "\n")
}

// ─────────────────────────────────────────────────────────────
//  교육부 공지 RSS
// ─────────────────────────────────────────────────────────────

func fetchMOENews() string {
	// 연합뉴스 사회 RSS에서 교육 키워드 필터
	items, err := fetchRSSFeed("https://www.yna.co.kr/rss/society.xml", 10)
	if err == nil && len(items) > 0 {
		var eduItems []rssItem
		for _, item := range items {
			t := strings.ToLower(item.Title)
			if strings.Contains(t, "교육") || strings.Contains(t, "학교") ||
				strings.Contains(t, "교사") || strings.Contains(t, "학생") ||
				strings.Contains(t, "입시") || strings.Contains(t, "수업") {
				eduItems = append(eduItems, item)
			}
		}
		if len(eduItems) > 0 {
			result := rssToLines(eduItems[:min(3, len(eduItems))])
			if result != "" {
				return result
			}
		}
		// 필터 없이 최신 3개
		result := rssToLines(items[:min(3, len(items))])
		if result != "" {
			return result
		}
	}
	return tavilyFallbackLines("교육부 교육청 학교 교육 뉴스 "+time.Now().Format("2006년"), 3)
}

// ─────────────────────────────────────────────────────────────
//  수능·대입 일정 (한국교육과정평가원 Tavily 검색)
// ─────────────────────────────────────────────────────────────

func fetchSuneungSchedule() string {
	year := time.Now().Year()
	llmMu.RLock()
	k := llmTavilyKey
	llmMu.RUnlock()
	res, ok := tavilySearchDomain(k, fmt.Sprintf("%d 수능 대입 일정", year), 5, "suneung.re.kr")
	if !ok || len(res.Items) == 0 {
		res, ok = tavilySearch(k, fmt.Sprintf("%d년 수능 대입 일정", year), 3)
		if !ok || len(res.Items) == 0 {
			return "입시 일정 정보 없음"
		}
	}
	var lines []string
	for _, item := range res.Items[:min(3, len(res.Items))] {
		lines = append(lines, "• "+item["title"])
	}
	return strings.Join(lines, "\n")
}

// ─────────────────────────────────────────────────────────────
//  고용노동부 워크넷 구인 (공공데이터포털 키 필요, 없으면 Tavily)
// ─────────────────────────────────────────────────────────────

func fetchWorknetJobs() string {
	keys := loadVerticalAPIKeys()
	if keys.DataGOKR != "" {
		url := fmt.Sprintf(
			"https://apis.data.go.kr/B490001/getJobInfo?serviceKey=%s&pageNo=1&numOfRows=5&returnType=json",
			keys.DataGOKR,
		)
		body, err := httpGetTimeout(url, 8*time.Second)
		if err == nil {
			var result struct {
				Data struct {
					Contents []struct {
						WantedTitle string `json:"wantedTitle"`
						CompanyName string `json:"companyName"`
						Region      string `json:"region"`
					} `json:"contents"`
				} `json:"data"`
			}
			if json.Unmarshal(body, &result) == nil && len(result.Data.Contents) > 0 {
				var lines []string
				lines = append(lines, "[워크넷 최신 구인공고]")
				for _, job := range result.Data.Contents {
					lines = append(lines, fmt.Sprintf("• [%s] %s (%s)", job.CompanyName, job.WantedTitle, job.Region))
				}
				return strings.Join(lines, "\n")
			}
		}
	}
	llmMu.RLock()
	k := llmTavilyKey
	llmMu.RUnlock()
	res, ok := tavilySearch(k, "오늘 대기업 채용 공고 사람인 잡코리아", 5)
	if !ok || len(res.Items) == 0 {
		return "채용 공고 수집 실패"
	}
	var lines []string
	for _, item := range res.Items[:min(3, len(res.Items))] {
		lines = append(lines, "• "+item["title"])
	}
	return strings.Join(lines, "\n")
}

// 2025 최저임금 현황 (hardcoded + 업데이트 뉴스)
func fetchMinimumWageInfo() string {
	// 2025년 최저임금 = 10,030원/시간 (법정)
	base := "📋 2025년 최저임금: 시간당 ₩10,030\n" +
		"  월급 환산(209h): 2,096,270원\n" +
		"  주휴수당 포함 실수령 기준\n"

	// 변경 뉴스 확인
	update := tavilyFallbackLines(fmt.Sprintf("%d년 최저임금 고용노동부 노동법 개정", time.Now().Year()), 2)
	if update != "" && !strings.Contains(update, "실패") {
		return base + "\n🆕 최신 동향:\n" + update
	}
	return base
}

// ─────────────────────────────────────────────────────────────
//  원자재 시세 (metals-api 무료 티어 → 폴백 Tavily)
// ─────────────────────────────────────────────────────────────

func fetchMetalPrices() string {
	// 무료 API: exchangerate.host는 금/은만 지원 (키 없이 사용 불가 변경됨)
	// → Tavily 폴백 사용
	return tavilyFallbackSingle("오늘 철강 구리 알루미늄 금 원자재 가격")
}

// ─────────────────────────────────────────────────────────────
//  KS/ISO 표준 (국가기술표준원 RSS + Tavily 폴백)
// ─────────────────────────────────────────────────────────────

func fetchKSStandardsNews() string {
	// 연합뉴스 경제 RSS에서 산업/표준 키워드 필터
	items, err := fetchRSSFeed("https://www.yna.co.kr/rss/economy.xml", 10)
	if err == nil && len(items) > 0 {
		var stdItems []rssItem
		for _, item := range items {
			t := strings.ToLower(item.Title)
			if strings.Contains(t, "규격") || strings.Contains(t, "표준") ||
				strings.Contains(t, "인증") || strings.Contains(t, "제조") ||
				strings.Contains(t, "산업") || strings.Contains(t, "공정") {
				stdItems = append(stdItems, item)
			}
		}
		if len(stdItems) > 0 {
			result := rssToLines(stdItems[:min(3, len(stdItems))])
			if result != "" {
				return result
			}
		}
	}
	return tavilyFallbackLines(time.Now().Format("2006년")+" KS ISO IEC 규격 개정 표준 발행", 3)
}

// ─────────────────────────────────────────────────────────────
//  Tavily 헬퍼 — 단순 검색 결과 반환
// ─────────────────────────────────────────────────────────────

func tavilyFallbackLines(query string, limit int) string {
	llmMu.RLock()
	k := llmTavilyKey
	llmMu.RUnlock()
	res, ok := tavilySearch(k, query, limit+2)
	if !ok || len(res.Items) == 0 {
		return ""
	}
	var lines []string
	for _, item := range res.Items[:min(limit, len(res.Items))] {
		lines = append(lines, "• "+item["title"])
	}
	return strings.Join(lines, "\n")
}

func tavilyFallbackSingle(query string) string {
	llmMu.RLock()
	k := llmTavilyKey
	llmMu.RUnlock()
	res, ok := tavilySearch(k, query, 3)
	if !ok {
		return "정보 없음"
	}
	if res.Summary != "" {
		return res.Summary
	}
	if len(res.Items) > 0 {
		return res.Items[0]["title"]
	}
	return "정보 없음"
}

// ─────────────────────────────────────────────────────────────
//  API 키 설정 안내 핸들러 (GET /api/vertical/apikeys/info)
// ─────────────────────────────────────────────────────────────

func handleVerticalAPIKeysInfo(w http.ResponseWriter, r *http.Request) {
	saveVerticalAPIKeysTemplate()
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".nexus", "vertical_apis.json")
	keys := loadVerticalAPIKeys()

	status := map[string]any{
		"config_path": configPath,
		"keys": map[string]any{
			"law_go_kr": map[string]any{
				"set":         keys.LawGOKR != "",
				"description": "법제처 국가법령정보 API",
				"signup_url":  "https://open.law.go.kr",
				"price":       "무료",
				"usage":       "법령 검색, 최근 개정 법령 목록",
			},
			"data_go_kr": map[string]any{
				"set":         keys.DataGOKR != "",
				"description": "공공데이터포털 공통 인증키",
				"signup_url":  "https://www.data.go.kr",
				"price":       "무료",
				"usage":       "의약품정보, 부동산실거래가, 워크넷구인",
			},
			"youtube_v3": map[string]any{
				"set":         keys.YouTubeV3 != "",
				"description": "YouTube Data API v3",
				"signup_url":  "https://console.cloud.google.com",
				"price":       "무료 10,000유닛/일",
				"usage":       "유튜브 트렌딩 정확도 향상",
			},
			"github_token": map[string]any{
				"set":         keys.GitHubToken != "",
				"description": "GitHub Personal Access Token",
				"signup_url":  "https://github.com/settings/tokens",
				"price":       "무료",
				"usage":       "GitHub API 요청한도 60→5000/hr",
			},
		},
		"note": "키가 없어도 Tavily 검색·chromedp 크롤링·무료 API로 자동 동작합니다.",
	}
	json200(w, map[string]any{"ok": true, "status": status})
}

// ─────────────────────────────────────────────────────────────
//  API 키 저장 핸들러 (POST /api/vertical/apikeys/save)
// ─────────────────────────────────────────────────────────────

func handleVerticalAPIKeysSave(w http.ResponseWriter, r *http.Request) {
	var keys VerticalAPIKeys
	if err := json.NewDecoder(r.Body).Decode(&keys); err != nil {
		http.Error(w, `{"ok":false,"error":"invalid body"}`, 400)
		return
	}
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".nexus")
	os.MkdirAll(dir, 0755)
	data, _ := json.MarshalIndent(keys, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "vertical_apis.json"), data, 0600); err != nil {
		http.Error(w, `{"ok":false,"error":"save failed"}`, 500)
		return
	}
	json200(w, map[string]any{"ok": true, "message": "API 키가 저장되었습니다."})
}

// context 호환용 래퍼
func newCancelContext(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithCancel(parent)
}
