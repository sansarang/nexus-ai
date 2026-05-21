//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ══════════════════════════════════════════════════════════════
//  직업군별 자동 브리핑 워크플로
//  POST /api/vertical/workflow/run
// ══════════════════════════════════════════════════════════════

type VerticalWorkflowStep struct {
	Label  string `json:"label"`
	Result string `json:"result"`
	OK     bool   `json:"ok"`
}

type VerticalWorkflowResult struct {
	VerticalID string                 `json:"vertical_id"`
	Name       string                 `json:"name"`
	Steps      []VerticalWorkflowStep `json:"steps"`
	Summary    string                 `json:"summary"`
	RunAt      string                 `json:"run_at"`
}

func handleVerticalWorkflowRun(w http.ResponseWriter, r *http.Request) {
	var req struct {
		VerticalID string `json:"vertical_id"`
		Lang       string `json:"lang"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.VerticalID == "" {
		cfg := loadVerticalConfig()
		req.VerticalID = cfg.ID
	}
	if req.Lang == "" {
		llmMu.RLock()
		req.Lang = llmUserLang
		llmMu.RUnlock()
	}
	result := runVerticalWorkflow(req.VerticalID, req.Lang)
	json200(w, map[string]any{"ok": true, "result": result})
}

func runVerticalWorkflow(verticalID, lang string) VerticalWorkflowResult {
	now := time.Now().Format("2006-01-02 15:04")
	isKo := lang != "en"
	var steps []VerticalWorkflowStep

	// 레이블 헬퍼
	lbl := func(ko, en string) string {
		if isKo { return ko }
		return en
	}

	switch verticalID {

	// ── 법무 ──────────────────────────────────────────────────
	case "legal":
		steps = append(steps,
			runStep(lbl("⚖️ 오늘의 법률·판례 뉴스", "⚖️ Today's Legal & Case News"), fetchSupremeCourtNews),
			runStep(lbl("📋 최근 개정 법령 (법제처)", "📋 Recent Law Amendments"), fetchRecentLawAmendments),
			runStep(lbl("🗓️ 오늘 법원 일정", "🗓️ Court Schedule Today"), func() string {
				d := now[:10]
				if isKo {
					return fmt.Sprintf("민사 접수 마감: %s 17:00\n대법원 선고: scourt.go.kr 공개 일정 참조\n\n📌 법제처 API 키 등록 시 실시간 법령 검색 가능\n→ /api/vertical/apikeys/info", d)
				}
				return fmt.Sprintf("Court filing deadline: %s 17:00\nSupreme Court calendar: check official site\n\n📌 Register law.go.kr API key for real-time statute search", d)
			}),
		)

	// ── 의원 ──────────────────────────────────────────────────
	case "medical":
		weatherCity := "Seoul"
		steps = append(steps,
			runStep(lbl("🩺 의료·임상 뉴스 (청년의사)", "🩺 Medical & Clinical News"), fetchMedicalNews),
			runStep(lbl("💊 건강보험 급여 변경 동향", "💊 Health Insurance Coverage Updates"), fetchHIRANews),
			runStep(lbl("🌤️ 오늘 날씨 (환자 방문 영향)", "🌤️ Weather Today (patient visit impact)"), func() string {
				wLang := "ko"
				if !isKo { wLang = "en" }
				return fetchWeatherText(weatherCity, wLang)
			}),
		)

	// ── 회계사 ────────────────────────────────────────────────
	case "accountant":
		steps = append(steps,
			runStep(lbl("📅 이번 달 세무 신고 일정", "📅 This Month's Tax Filing Deadlines"), func() string {
				m := time.Now().Month()
				if isKo {
					deadlines := map[time.Month]string{
						1: "부가세 확정신고(1/25), 종합소득세 중간예납(1/31)",
						2: "원천세 납부(2/10), 법인세 예납(2/28)",
						3: "법인세 확정신고(3/31)",
						4: "부가세 예정신고(4/25)",
						5: "종합소득세 확정신고(5/31)",
						6: "원천세 납부(6/10)",
						7: "부가세 확정신고(7/25)",
						8: "원천세 납부(8/10)",
						9: "법인세 중간예납(9/30)",
						10: "부가세 예정신고(10/25)",
						11: "원천세 납부(11/10)",
						12: "원천세 납부(12/10), 연말정산 준비",
					}
					d := deadlines[m]
					if d == "" { d = "이번 달 주요 신고 일정 없음" }
					return "📌 " + d
				}
				// English — US tax calendar
				enDeadlines := map[time.Month]string{
					1:  "W-2/1099 distribution deadline (Jan 31)",
					2:  "S-Corp/Partnership estimated tax (Feb 15)",
					3:  "S-Corp/Partnership returns due (Mar 15)",
					4:  "Individual & C-Corp returns due (Apr 15)",
					6:  "2nd quarter estimated tax (Jun 15)",
					9:  "3rd quarter estimated tax (Sep 15)",
					10: "Extended returns due (Oct 15)",
					12: "Year-end tax planning window",
				}
				d := enDeadlines[m]
				if d == "" { d = "No major filing deadlines this month" }
				return "📌 " + d
			}),
			runStep(lbl("📊 국세청 세무·회계 공지", "📊 Tax & Accounting News"), fetchNTSNews),
			runStep(lbl("💱 실시간 환율 (USD/EUR/JPY/CNY)", "💱 Live Exchange Rates (USD/EUR/JPY/CNY)"), fetchExchangeRates),
		)

	// ── 크리에이터 ────────────────────────────────────────────
	case "creator":
		steps = append(steps,
			runStep(lbl("🔥 유튜브 트렌딩 (한국)", "🔥 YouTube Trending"), func() string {
				return scrapeYouTubeTrending(5)
			}),
			runStep(lbl("🎵 오늘 틱톡 트렌드", "🎵 TikTok Trends Today"), func() string {
				llmMu.RLock()
				k := llmTavilyKey
				llmMu.RUnlock()
				q := "tiktok trending viral today"
				if isKo { q = "오늘 틱톡 트렌드 바이럴 한국" }
				res, ok := tavilySearchDomain(k, q, 5, "tiktok.com")
				if !ok || len(res.Items) == 0 {
					return tavilyFallbackLines(q, 3)
				}
				var lines []string
				for _, item := range res.Items[:min(3, len(res.Items))] {
					lines = append(lines, "• "+item["title"])
				}
				return strings.Join(lines, "\n")
			}),
			runStep(lbl("🌐 오늘의 인터넷 이슈·밈", "🌐 Today's Internet Issues & Memes"), func() string {
				q := "today viral meme internet issue social media"
				if isKo { q = "오늘 인터넷 밈 이슈 화제 커뮤니티 SNS" }
				return tavilyFallbackSingle(q)
			}),
		)

	// ── 부동산 ────────────────────────────────────────────────
	case "realtor":
		steps = append(steps,
			runStep(lbl("🏠 오늘 부동산 뉴스", "🏠 Real Estate News Today"), fetchRealEstateNews),
			runStep(lbl("📋 이번 달 청약 일정 (청약홈)", "📋 Housing Application Schedule"), fetchApplyHomeSchedule),
			runStep(lbl("💰 금리·환율 동향", "💰 Interest Rate & FX Trends"), func() string {
				rate := fetchExchangeRates()
				q := "US Federal Reserve interest rate mortgage rate latest"
				if isKo { q = "한국은행 기준금리 주택담보대출 금리 최신" }
				interest := tavilyFallbackSingle(q)
				rateLabel := "Exchange rates:"
				if isKo { rateLabel = "환율:" }
				interestLabel := "\n\nRate trend: "
				if isKo { interestLabel = "\n\n금리 동향: " }
				return rateLabel + "\n" + rate + interestLabel + interest
			}),
		)

	// ── 교사 ──────────────────────────────────────────────────
	case "teacher":
		steps = append(steps,
			runStep(lbl("📚 교육부 공지·교육 뉴스", "📚 Education News"), fetchMOENews),
			runStep(lbl("🎓 수능·대입 일정", "🎓 College Admission Schedule"), fetchSuneungSchedule),
			runStep(lbl("📺 EBS 오늘의 추천 콘텐츠", "📺 Today's Recommended Learning Content"), func() string {
				llmMu.RLock()
				k := llmTavilyKey
				llmMu.RUnlock()
				q := fmt.Sprintf("EBS 강의 수능 추천 %d", time.Now().Year())
				domain := "ebs.co.kr"
				if !isKo {
					q = fmt.Sprintf("Khan Academy recommended free course %d", time.Now().Year())
					domain = "khanacademy.org"
				}
				res, ok := tavilySearchDomain(k, q, 3, domain)
				if !ok || len(res.Items) == 0 {
					fallQ := "EBS 교육 오늘의 추천 강의 수업"
					if !isKo { fallQ = "free online course learning today recommendation" }
					return tavilyFallbackLines(fallQ, 2)
				}
				var lines []string
				for _, item := range res.Items[:min(2, len(res.Items))] {
					lines = append(lines, "• "+item["title"])
				}
				return strings.Join(lines, "\n")
			}),
		)

	// ── HR ────────────────────────────────────────────────────
	case "hr":
		steps = append(steps,
			runStep(lbl("👥 오늘의 채용·HR 뉴스", "👥 HR & Hiring News Today"), func() string {
				q := "today hiring HR human resources labor news"
				if isKo { q = "오늘 채용 HR 인사 노동 뉴스" }
				return tavilyFallbackLines(q, 3)
			}),
			runStep(lbl("📋 최저임금·노동법 현황", "📋 Minimum Wage & Labor Law"), fetchMinimumWageInfo),
			runStep(lbl("💼 오늘 주요 채용 공고 (워크넷)", "💼 Top Job Postings Today"), fetchWorknetJobs),
		)

	// ── 개발자 ────────────────────────────────────────────────
	case "developer":
		steps = append(steps,
			runStep("⭐ GitHub Trending (API + chromedp)", func() string {
				return fetchGitHubTrending(5)
			}),
			runStep("🔶 Hacker News Top Stories", func() string {
				return fetchHackerNewsTop(4)
			}),
			runStep("🔴 Reddit r/programming Hot Posts", func() string {
				posts, err := crawlRedditFeed("https://www.reddit.com/r/programming/hot/", 5)
				if err != nil || len(posts) == 0 {
					return tavilyFallbackLines("reddit programming today trending posts", 3)
				}
				var lines []string
				for _, p := range posts[:min(3, len(posts))] {
					lines = append(lines, fmt.Sprintf("• %s ↑%s", p.Title, p.Score))
				}
				return strings.Join(lines, "\n")
			}),
		)

	// ── 엔지니어 ──────────────────────────────────────────────
	case "engineer":
		steps = append(steps,
			runStep(lbl("⚙️ 오늘 산업·제조 뉴스", "⚙️ Industry & Manufacturing News"), func() string {
				q := "today manufacturing engineering industry automation news"
				if isKo { q = "오늘 산업 제조 엔지니어링 공장 자동화 뉴스" }
				return tavilyFallbackLines(q, 3)
			}),
			runStep(lbl("📦 오늘 원자재 시세 (철강·구리)", "📦 Raw Material Prices (Steel·Copper)"), fetchMetalPrices),
			runStep(lbl("📐 KS/ISO 규격 업데이트", "📐 ISO/ASME Standards Updates"), fetchKSStandardsNews),
		)

	// ── 일반 ──────────────────────────────────────────────────
	default:
		steps = append(steps,
			runStep(lbl("🌤️ 오늘 날씨", "🌤️ Weather Today"), func() string {
				wLang := "ko"
				if !isKo { wLang = "en" }
				return fetchWeatherText("Seoul", wLang)
			}),
			runStep(lbl("📰 오늘의 주요 뉴스", "📰 Today's Top News"), func() string {
				q := "today top news headlines"
				if isKo { q = "오늘 주요 뉴스 이슈" }
				return tavilyFallbackLines(q, 3)
			}),
			runStep("🔶 Hacker News Top Stories", func() string {
				return fetchHackerNewsTop(3)
			}),
		)
	}

	// 이름 찾기
	name := verticalID
	for _, p := range verticalPresets {
		if p.ID == verticalID {
			name = p.Name
			break
		}
	}

	// 요약
	var summaryLines []string
	for _, s := range steps {
		icon := "✅"
		if !s.OK {
			icon = "⚠️"
		}
		summaryLines = append(summaryLines, fmt.Sprintf("%s %s", icon, s.Label))
	}
	doneMsg := "Briefing complete"
	if isKo { doneMsg = "브리핑 완료" }
	summary := fmt.Sprintf("%s %s (%s)\n%s", name, doneMsg, now, strings.Join(summaryLines, "\n"))

	return VerticalWorkflowResult{
		VerticalID: verticalID,
		Name:       name,
		Steps:      steps,
		Summary:    summary,
		RunAt:      now,
	}
}

func runStep(label string, fn func() string) VerticalWorkflowStep {
	result := fn()
	ok := result != "" &&
		!strings.Contains(result, "실패") &&
		!strings.Contains(result, "없음") &&
		!strings.Contains(result, "오류")
	return VerticalWorkflowStep{Label: label, Result: result, OK: ok}
}
