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
	var steps []VerticalWorkflowStep

	switch verticalID {

	// ── 법무사 ────────────────────────────────────────────────
	case "legal":
		steps = append(steps,
			runStep("⚖️ 오늘의 법률·판례 뉴스", fetchSupremeCourtNews),
			runStep("📋 최근 개정 법령 (법제처)", fetchRecentLawAmendments),
			runStep("🗓️ 오늘 법원 일정", func() string {
				d := now[:10]
				return fmt.Sprintf("민사 접수 마감: %s 17:00\n대법원 선고: scourt.go.kr 공개 일정 참조\n\n📌 법제처 API 키 등록 시 실시간 법령 검색 가능\n→ /api/vertical/apikeys/info", d)
			}),
		)

	// ── 의원 ──────────────────────────────────────────────────
	case "medical":
		steps = append(steps,
			runStep("🩺 식약처 의약품·임상 공지", fetchDrugApprovalNews),
			runStep("💊 건강보험심사평가원 급여 변경", fetchHIRANews),
			runStep("🌤️ 오늘 날씨 (환자 방문 영향)", func() string {
				return fetchWeatherText("Seoul", "ko")
			}),
		)

	// ── 회계사 ────────────────────────────────────────────────
	case "accountant":
		steps = append(steps,
			runStep("📅 이번 달 세무 신고 일정", func() string {
				m := time.Now().Month()
				deadlines := map[time.Month]string{
					1:  "부가세 확정신고(1/25), 종합소득세 중간예납(1/31)",
					2:  "원천세 납부(2/10), 법인세 예납(2/28)",
					3:  "법인세 확정신고(3/31)",
					4:  "부가세 예정신고(4/25)",
					5:  "종합소득세 확정신고(5/31)",
					6:  "원천세 납부(6/10)",
					7:  "부가세 확정신고(7/25)",
					8:  "원천세 납부(8/10)",
					9:  "법인세 중간예납(9/30)",
					10: "부가세 예정신고(10/25)",
					11: "원천세 납부(11/10)",
					12: "원천세 납부(12/10), 연말정산 준비",
				}
				d := deadlines[m]
				if d == "" {
					d = "이번 달 주요 신고 일정 없음"
				}
				return "📌 " + d
			}),
			runStep("📊 국세청 세무·회계 공지", fetchNTSNews),
			runStep("💱 실시간 환율 (USD/EUR/JPY/CNY)", fetchExchangeRates),
		)

	// ── 크리에이터 ────────────────────────────────────────────
	case "creator":
		steps = append(steps,
			runStep("🔥 유튜브 트렌딩 (한국)", func() string {
				return scrapeYouTubeTrending(5)
			}),
			runStep("🎵 오늘 틱톡 트렌드", func() string {
				llmMu.RLock()
				k := llmTavilyKey
				llmMu.RUnlock()
				res, ok := tavilySearchDomain(k, "tiktok trending viral today Korea", 5, "tiktok.com")
				if !ok || len(res.Items) == 0 {
					return tavilyFallbackLines("오늘 틱톡 트렌드 바이럴 한국", 3)
				}
				var lines []string
				for _, item := range res.Items[:min(3, len(res.Items))] {
					lines = append(lines, "• "+item["title"])
				}
				return strings.Join(lines, "\n")
			}),
			runStep("🌐 오늘의 인터넷 이슈·밈", func() string {
				return tavilyFallbackSingle("오늘 인터넷 밈 이슈 화제 커뮤니티 SNS")
			}),
		)

	// ── 부동산 ────────────────────────────────────────────────
	case "realtor":
		steps = append(steps,
			runStep("🏠 오늘 부동산 뉴스", fetchRealEstateNews),
			runStep("📋 이번 달 청약 일정 (청약홈)", fetchApplyHomeSchedule),
			runStep("💰 금리·환율 동향", func() string {
				rate := fetchExchangeRates()
				interest := tavilyFallbackSingle("한국은행 기준금리 주택담보대출 금리 최신")
				return "환율:\n" + rate + "\n\n금리 동향: " + interest
			}),
		)

	// ── 교사 ──────────────────────────────────────────────────
	case "teacher":
		steps = append(steps,
			runStep("📚 교육부 공지·교육 뉴스", fetchMOENews),
			runStep("🎓 수능·대입 일정", fetchSuneungSchedule),
			runStep("📺 EBS 오늘의 추천 콘텐츠", func() string {
				llmMu.RLock()
				k := llmTavilyKey
				llmMu.RUnlock()
				res, ok := tavilySearchDomain(k, fmt.Sprintf("EBS 강의 수능 추천 %d", time.Now().Year()), 3, "ebs.co.kr")
				if !ok || len(res.Items) == 0 {
					return tavilyFallbackLines("EBS 교육 오늘의 추천 강의 수업", 2)
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
			runStep("👥 오늘의 채용·HR 뉴스", func() string {
				return tavilyFallbackLines("오늘 채용 HR 인사 노동 뉴스", 3)
			}),
			runStep("📋 최저임금·노동법 현황", fetchMinimumWageInfo),
			runStep("💼 오늘 주요 채용 공고 (워크넷)", fetchWorknetJobs),
		)

	// ── 개발자 ────────────────────────────────────────────────
	case "developer":
		steps = append(steps,
			runStep("⭐ GitHub 트렌딩 (API + chromedp)", func() string {
				return fetchGitHubTrending(5)
			}),
			runStep("🔶 Hacker News Top Stories", func() string {
				return fetchHackerNewsTop(4)
			}),
			runStep("🔴 Reddit r/programming 인기 글", func() string {
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
			runStep("⚙️ 오늘 산업·제조 뉴스", func() string {
				return tavilyFallbackLines("오늘 산업 제조 엔지니어링 공장 자동화 뉴스", 3)
			}),
			runStep("📦 오늘 원자재 시세 (철강·구리)", fetchMetalPrices),
			runStep("📐 KS/ISO 규격 업데이트 (국가기술표준원)", fetchKSStandardsNews),
		)

	// ── 일반 ──────────────────────────────────────────────────
	default:
		steps = append(steps,
			runStep("🌤️ 오늘 날씨", func() string {
				return fetchWeatherText("Seoul", "ko")
			}),
			runStep("📰 오늘의 주요 뉴스", func() string {
				return tavilyFallbackLines("오늘 주요 뉴스 이슈", 3)
			}),
			runStep("🔶 Hacker News (글로벌 기술 동향)", func() string {
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
	summary := fmt.Sprintf("%s 브리핑 완료 (%s)\n%s", name, now, strings.Join(summaryLines, "\n"))

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
