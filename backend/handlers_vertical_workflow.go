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
//  직업군별 자동 워크플로 — 앱 시작 시 또는 직업군 전환 시 실행
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

// POST /api/vertical/workflow/run
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
	isKo := lang != "en"
	now := time.Now().Format("2006-01-02 15:04")

	var steps []VerticalWorkflowStep

	switch verticalID {

	case "legal":
		// 1. 오늘 날짜 기반 법률 뉴스 검색
		steps = append(steps, runStep("⚖️ 오늘의 법률·판례 뉴스 수집", func() string {
			llmMu.RLock(); k := llmTavilyKey; llmMu.RUnlock()
			query := "오늘 법률 판례 대법원 헌법재판소 뉴스 " + time.Now().Format("2006년 01월")
			if !isKo { query = "today legal court ruling law news " + time.Now().Format("January 2006") }
			res, ok := tavilySearch(k, query, 5)
			if !ok || len(res.Items) == 0 { return "뉴스 수집 실패" }
			var lines []string
			for _, item := range res.Items[:min(3, len(res.Items))] {
				lines = append(lines, "• "+item["title"])
			}
			return strings.Join(lines, "\n")
		}))
		// 2. 국가법령정보 최신 개정 법령
		steps = append(steps, runStep("📋 최근 개정 법령 확인", func() string {
			llmMu.RLock(); k := llmTavilyKey; llmMu.RUnlock()
			res, ok := tavilySearch(k, "최근 법령 개정 시행 "+time.Now().Format("2006년"), 3)
			if !ok || len(res.Items) == 0 { return "개정 법령 정보 없음" }
			var lines []string
			for _, item := range res.Items[:min(3, len(res.Items))] {
				lines = append(lines, "• "+item["title"])
			}
			return strings.Join(lines, "\n")
		}))
		// 3. 오늘 법원 주요 일정
		steps = append(steps, runStep("🗓️ 오늘 주요 법원 일정", func() string {
			return fmt.Sprintf("민사 접수 마감: %s 17:00\n대법원 선고: 공개 일정 확인 필요", now[:10])
		}))

	case "medical":
		// 1. 최신 의학 뉴스
		steps = append(steps, runStep("🩺 오늘의 의학·임상 뉴스", func() string {
			llmMu.RLock(); k := llmTavilyKey; llmMu.RUnlock()
			res, ok := tavilySearch(k, "오늘 의학 임상 FDA 식약처 신약 승인 뉴스", 5)
			if !ok || len(res.Items) == 0 { return "의학 뉴스 수집 실패" }
			var lines []string
			for _, item := range res.Items[:min(3, len(res.Items))] {
				lines = append(lines, "• "+item["title"])
			}
			return strings.Join(lines, "\n")
		}))
		// 2. 건강보험심사평가원 공지
		steps = append(steps, runStep("💊 건강보험 급여 변경 사항", func() string {
			llmMu.RLock(); k := llmTavilyKey; llmMu.RUnlock()
			res, ok := tavilySearch(k, "건강보험 급여 약제 변경 "+time.Now().Format("2006년"), 3)
			if !ok || len(res.Items) == 0 { return "급여 변경 정보 없음" }
			var lines []string
			for _, item := range res.Items[:min(2, len(res.Items))] {
				lines = append(lines, "• "+item["title"])
			}
			return strings.Join(lines, "\n")
		}))
		// 3. 오늘 날씨 (외출 환자 고려)
		steps = append(steps, runStep("🌤️ 오늘 날씨 (환자 방문 영향)", func() string {
			return fetchWeatherText("Seoul", "ko")
		}))

	case "accountant":
		// 1. 오늘 세무 신고 마감일 체크
		steps = append(steps, runStep("📅 이번 달 세무 신고 일정", func() string {
			m := time.Now().Month()
			deadlines := map[time.Month]string{
				1: "부가세 확정신고(1/25), 종합소득세 중간예납(1/31)",
				2: "원천세 납부(2/10)",
				3: "법인세 확정신고(3/31)",
				4: "부가세 예정신고(4/25)",
				5: "종합소득세 확정신고(5/31)",
				6: "원천세 납부(6/10)",
				7: "부가세 확정신고(7/25)",
				8: "원천세 납부(8/10)",
				9: "법인세 중간예납(9/30)",
				10: "부가세 예정신고(10/25)",
				11: "원천세 납부(11/10)",
				12: "원천세 납부(12/10)",
			}
			d := deadlines[m]
			if d == "" { d = "이번 달 주요 신고 일정 없음" }
			return d
		}))
		// 2. 오늘의 세무·회계 뉴스
		steps = append(steps, runStep("📊 오늘의 세무·회계 뉴스", func() string {
			llmMu.RLock(); k := llmTavilyKey; llmMu.RUnlock()
			res, ok := tavilySearch(k, "오늘 국세청 세무 회계 세법 개정 뉴스", 5)
			if !ok || len(res.Items) == 0 { return "세무 뉴스 수집 실패" }
			var lines []string
			for _, item := range res.Items[:min(3, len(res.Items))] {
				lines = append(lines, "• "+item["title"])
			}
			return strings.Join(lines, "\n")
		}))
		// 3. 주요 환율 (외화 회계 필요 시)
		steps = append(steps, runStep("💱 오늘 주요 환율", func() string {
			llmMu.RLock(); k := llmTavilyKey; llmMu.RUnlock()
			res, ok := tavilySearch(k, "오늘 원달러 원유로 환율", 3)
			if !ok || len(res.Items) == 0 { return "환율 정보 없음" }
			return res.Summary
		}))

	case "creator":
		// 1. 유튜브 트렌딩 키워드
		steps = append(steps, runStep("🔥 오늘 유튜브 트렌딩", func() string {
			llmMu.RLock(); k := llmTavilyKey; llmMu.RUnlock()
			res, ok := tavilySearch(k, "오늘 유튜브 트렌딩 인기 급상승 키워드", 5)
			if !ok || len(res.Items) == 0 { return "트렌딩 수집 실패" }
			var lines []string
			for _, item := range res.Items[:min(4, len(res.Items))] {
				lines = append(lines, "• "+item["title"])
			}
			return strings.Join(lines, "\n")
		}))
		// 2. 틱톡 트렌드
		steps = append(steps, runStep("🎵 오늘 틱톡 트렌드", func() string {
			llmMu.RLock(); k := llmTavilyKey; llmMu.RUnlock()
			res, ok := tavilySearchDomain(k, "tiktok trending viral today Korea", 5, "tiktok.com")
			if !ok || len(res.Items) == 0 { return "틱톡 트렌드 수집 실패" }
			var lines []string
			for _, item := range res.Items[:min(3, len(res.Items))] {
				lines = append(lines, "• "+item["title"])
			}
			return strings.Join(lines, "\n")
		}))
		// 3. 오늘의 밈·이슈 (Reddit)
		steps = append(steps, runStep("🌐 오늘의 인터넷 이슈", func() string {
			llmMu.RLock(); k := llmTavilyKey; llmMu.RUnlock()
			res, ok := tavilySearch(k, "오늘 인터넷 밈 이슈 화제 커뮤니티", 5)
			if !ok || len(res.Items) == 0 { return "이슈 수집 실패" }
			return res.Summary
		}))

	case "realtor":
		// 1. 오늘 부동산 뉴스
		steps = append(steps, runStep("🏠 오늘의 부동산 뉴스", func() string {
			llmMu.RLock(); k := llmTavilyKey; llmMu.RUnlock()
			res, ok := tavilySearch(k, "오늘 부동산 아파트 시세 정책 뉴스", 5)
			if !ok || len(res.Items) == 0 { return "부동산 뉴스 수집 실패" }
			var lines []string
			for _, item := range res.Items[:min(3, len(res.Items))] {
				lines = append(lines, "• "+item["title"])
			}
			return strings.Join(lines, "\n")
		}))
		// 2. 청약 일정
		steps = append(steps, runStep("📋 이번 달 청약 일정", func() string {
			llmMu.RLock(); k := llmTavilyKey; llmMu.RUnlock()
			res, ok := tavilySearch(k, time.Now().Format("2006년 01월")+" 청약 분양 일정", 5)
			if !ok || len(res.Items) == 0 { return "청약 일정 정보 없음" }
			var lines []string
			for _, item := range res.Items[:min(3, len(res.Items))] {
				lines = append(lines, "• "+item["title"])
			}
			return strings.Join(lines, "\n")
		}))
		// 3. 금리 동향 (대출 영향)
		steps = append(steps, runStep("💰 금리·대출 동향", func() string {
			llmMu.RLock(); k := llmTavilyKey; llmMu.RUnlock()
			res, ok := tavilySearch(k, "한국은행 기준금리 주택담보대출 금리 최신", 3)
			if !ok || len(res.Items) == 0 { return "금리 정보 없음" }
			return res.Summary
		}))

	case "teacher":
		// 1. 교육 뉴스
		steps = append(steps, runStep("📚 오늘의 교육 뉴스", func() string {
			llmMu.RLock(); k := llmTavilyKey; llmMu.RUnlock()
			res, ok := tavilySearch(k, "오늘 교육부 교육청 학교 교육 뉴스", 5)
			if !ok || len(res.Items) == 0 { return "교육 뉴스 수집 실패" }
			var lines []string
			for _, item := range res.Items[:min(3, len(res.Items))] {
				lines = append(lines, "• "+item["title"])
			}
			return strings.Join(lines, "\n")
		}))
		// 2. 수능·입시 일정
		steps = append(steps, runStep("🎓 주요 입시 일정", func() string {
			llmMu.RLock(); k := llmTavilyKey; llmMu.RUnlock()
			res, ok := tavilySearch(k, time.Now().Format("2006년")+" 수능 입시 대입 일정", 3)
			if !ok || len(res.Items) == 0 { return "입시 일정 정보 없음" }
			var lines []string
			for _, item := range res.Items[:min(2, len(res.Items))] {
				lines = append(lines, "• "+item["title"])
			}
			return strings.Join(lines, "\n")
		}))
		// 3. EBS 오늘의 강의
		steps = append(steps, runStep("📺 EBS 오늘의 추천 콘텐츠", func() string {
			llmMu.RLock(); k := llmTavilyKey; llmMu.RUnlock()
			res, ok := tavilySearchDomain(k, "EBS 오늘의 강의 추천 "+time.Now().Format("2006"), 3, "ebs.co.kr")
			if !ok || len(res.Items) == 0 { return "EBS 정보 없음" }
			return res.Items[0]["title"]
		}))

	case "hr":
		// 1. 채용 트렌드
		steps = append(steps, runStep("👥 오늘의 채용·HR 뉴스", func() string {
			llmMu.RLock(); k := llmTavilyKey; llmMu.RUnlock()
			res, ok := tavilySearch(k, "오늘 채용 HR 인사 노동 뉴스", 5)
			if !ok || len(res.Items) == 0 { return "HR 뉴스 수집 실패" }
			var lines []string
			for _, item := range res.Items[:min(3, len(res.Items))] {
				lines = append(lines, "• "+item["title"])
			}
			return strings.Join(lines, "\n")
		}))
		// 2. 최저임금·노동법 변경
		steps = append(steps, runStep("📋 최저임금·노동법 현황", func() string {
			llmMu.RLock(); k := llmTavilyKey; llmMu.RUnlock()
			res, ok := tavilySearch(k, time.Now().Format("2006년")+" 최저임금 노동법 근로기준법 개정", 3)
			if !ok || len(res.Items) == 0 { return "노동법 정보 없음" }
			return res.Summary
		}))
		// 3. 주요 채용 공고 트렌드 (사람인/잡코리아)
		steps = append(steps, runStep("💼 오늘 주요 채용 공고 트렌드", func() string {
			llmMu.RLock(); k := llmTavilyKey; llmMu.RUnlock()
			res, ok := tavilySearch(k, "오늘 대기업 채용 공고 사람인 잡코리아", 5)
			if !ok || len(res.Items) == 0 { return "채용 공고 수집 실패" }
			var lines []string
			for _, item := range res.Items[:min(3, len(res.Items))] {
				lines = append(lines, "• "+item["title"])
			}
			return strings.Join(lines, "\n")
		}))

	case "developer":
		// 1. GitHub 트렌딩
		steps = append(steps, runStep("⭐ GitHub 오늘의 트렌딩", func() string {
			llmMu.RLock(); k := llmTavilyKey; llmMu.RUnlock()
			res, ok := tavilySearchDomain(k, "github trending today repositories stars", 5, "github.com")
			if !ok || len(res.Items) == 0 { return "GitHub 트렌딩 수집 실패" }
			var lines []string
			for _, item := range res.Items[:min(4, len(res.Items))] {
				lines = append(lines, "• "+item["title"])
			}
			return strings.Join(lines, "\n")
		}))
		// 2. 기술 뉴스 (Hacker News / Dev.to)
		steps = append(steps, runStep("💻 오늘의 개발 뉴스", func() string {
			llmMu.RLock(); k := llmTavilyKey; llmMu.RUnlock()
			res, ok := tavilySearch(k, "today developer tech news AI framework release", 5)
			if !ok || len(res.Items) == 0 { return "개발 뉴스 수집 실패" }
			var lines []string
			for _, item := range res.Items[:min(3, len(res.Items))] {
				lines = append(lines, "• "+item["title"])
			}
			return strings.Join(lines, "\n")
		}))
		// 3. Reddit r/programming 트렌딩
		steps = append(steps, runStep("🔴 Reddit r/programming 인기 글", func() string {
			posts, err := crawlRedditFeed("https://www.reddit.com/r/programming/hot/", 5)
			if err != nil || len(posts) == 0 { return "Reddit 수집 실패 (브라우저 미실행 시 정상)" }
			var lines []string
			for _, p := range posts[:min(3, len(posts))] {
				lines = append(lines, fmt.Sprintf("• %s ↑%s", p.Title, p.Score))
			}
			return strings.Join(lines, "\n")
		}))

	case "engineer":
		// 1. 산업·제조 뉴스
		steps = append(steps, runStep("⚙️ 오늘의 산업·제조 뉴스", func() string {
			llmMu.RLock(); k := llmTavilyKey; llmMu.RUnlock()
			res, ok := tavilySearch(k, "오늘 산업 제조 엔지니어링 공장 자동화 뉴스", 5)
			if !ok || len(res.Items) == 0 { return "산업 뉴스 수집 실패" }
			var lines []string
			for _, item := range res.Items[:min(3, len(res.Items))] {
				lines = append(lines, "• "+item["title"])
			}
			return strings.Join(lines, "\n")
		}))
		// 2. 원자재 가격 (철강, 구리 등)
		steps = append(steps, runStep("📦 오늘 원자재 시세", func() string {
			llmMu.RLock(); k := llmTavilyKey; llmMu.RUnlock()
			res, ok := tavilySearch(k, "오늘 철강 구리 알루미늄 원자재 가격", 3)
			if !ok || len(res.Items) == 0 { return "원자재 시세 없음" }
			return res.Summary
		}))
		// 3. KS/ISO 최신 규격
		steps = append(steps, runStep("📐 KS/ISO 최신 규격 업데이트", func() string {
			llmMu.RLock(); k := llmTavilyKey; llmMu.RUnlock()
			res, ok := tavilySearch(k, time.Now().Format("2006년")+" KS ISO IEC 규격 개정 표준", 3)
			if !ok || len(res.Items) == 0 { return "규격 정보 없음" }
			var lines []string
			for _, item := range res.Items[:min(2, len(res.Items))] {
				lines = append(lines, "• "+item["title"])
			}
			return strings.Join(lines, "\n")
		}))

	default: // general
		steps = append(steps, runStep("🌤️ 오늘 날씨", func() string {
			return fetchWeatherText("Seoul", "ko")
		}))
		steps = append(steps, runStep("📰 오늘의 주요 뉴스", func() string {
			llmMu.RLock(); k := llmTavilyKey; llmMu.RUnlock()
			res, ok := tavilySearch(k, "오늘 주요 뉴스", 5)
			if !ok || len(res.Items) == 0 { return "뉴스 수집 실패" }
			var lines []string
			for _, item := range res.Items[:min(3, len(res.Items))] {
				lines = append(lines, "• "+item["title"])
			}
			return strings.Join(lines, "\n")
		}))
	}

	// 이름 찾기
	name := verticalID
	for _, p := range verticalPresets {
		if p.ID == verticalID {
			name = p.Name
			break
		}
	}

	// 전체 요약 생성
	var summaryLines []string
	for _, s := range steps {
		if s.OK {
			summaryLines = append(summaryLines, fmt.Sprintf("✅ %s", s.Label))
		} else {
			summaryLines = append(summaryLines, fmt.Sprintf("⚠️ %s", s.Label))
		}
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
	ok := result != "" && !strings.Contains(result, "실패") && !strings.Contains(result, "없음")
	return VerticalWorkflowStep{Label: label, Result: result, OK: ok}
}


