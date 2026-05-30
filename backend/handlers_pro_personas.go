//go:build windows

package main

import (
	"fmt"
	"net/http"
	"strings"
)

// ══════════════════════════════════════════════════════════════════
//  Pro Persona 전용 핸들러
//  - POST /api/finance/stock     → 주식/코인/ETF 분석
//  - POST /api/medical/search    → 의학 논문/약물 정보 검색
//  - POST /api/legal/review      → 계약서/법률 문서 검토
//  - POST /api/content/script    → 유튜브/SNS 스크립트 생성
//  - POST /api/content/hashtags  → 해시태그 생성
//  - POST /api/content/titles    → 영상 제목 SEO 최적화
// ══════════════════════════════════════════════════════════════════

// ── 📈 주식/코인 분석 ─────────────────────────────────────────────

func handleStockAnalysis(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Ticker string `json:"ticker"`
		Query  string `json:"query"`
		Lang   string `json:"lang"`
	}
	tryDecodeBody(r, &req)
	lang := req.Lang
	if lang == "" {
		lang = getLang(r)
	}
	if req.Ticker == "" && req.Query == "" {
		w.WriteHeader(http.StatusBadRequest)
		json200(w, map[string]any{"ok": false, "error": msgT("ticker 또는 query가 필요합니다", "ticker or query is required", lang)})
		return
	}
	res, _ := stockAnalysisLogic(req.Ticker, req.Query, lang)
	json200(w, res)
}

// ── 🏥 의학 검색 ──────────────────────────────────────────────────

func handleMedicalSearch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query string `json:"query"`
		Type  string `json:"type"`
		Lang  string `json:"lang"`
	}
	tryDecodeBody(r, &req)
	lang := req.Lang
	if lang == "" {
		lang = getLang(r)
	}
	if req.Query == "" {
		w.WriteHeader(http.StatusBadRequest)
		json200(w, map[string]any{"ok": false, "error": msgT("query가 필요합니다", "query is required", lang)})
		return
	}
	res, _ := medicalSearchLogic(req.Query, req.Type, lang)
	json200(w, res)
}

// ── ⚖️ 계약서/법률 문서 검토 ──────────────────────────────────────

func handleContractReview(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FilePath string `json:"file_path"`
		Content  string `json:"content"`
		Focus    string `json:"focus"`
		Lang     string `json:"lang"`
	}
	tryDecodeBody(r, &req)
	lang := req.Lang
	if lang == "" {
		lang = getLang(r)
	}
	res, _ := contractReviewLogic(req.FilePath, req.Content, req.Focus, lang)
	json200(w, res)
}

// ── 🎬 콘텐츠 스크립트 생성 ──────────────────────────────────────

func handleContentScript(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Topic    string `json:"topic"`
		Platform string `json:"platform"`
		Duration string `json:"duration"`
		Style    string `json:"style"`
		Lang     string `json:"lang"`
	}
	tryDecodeBody(r, &req)
	lang := req.Lang
	if lang == "" {
		lang = getLang(r)
	}
	if req.Topic == "" {
		w.WriteHeader(http.StatusBadRequest)
		json200(w, map[string]any{"ok": false, "error": msgT("topic이 필요합니다", "topic is required", lang)})
		return
	}
	res, _ := contentScriptLogic(req.Topic, req.Platform, req.Duration, req.Style, lang)
	json200(w, res)
}

// ══════════════════════════════════════════════════════════════════
//  Pure logic functions (called from dispatchAction without http.Request)
// ══════════════════════════════════════════════════════════════════

func proSearch(query string, maxItems int) []map[string]string {
	if tr, ok := tavilySearch(llmTavilyKey, query, maxItems); ok {
		return tr.Items
	}
	return nil
}

func proItemsToText(items []map[string]string, limit int) string {
	var sb strings.Builder
	for i, item := range items {
		if i >= limit {
			break
		}
		sb.WriteString(fmt.Sprintf("[%d] %s\n%s\n\n", i+1, item["title"], item["content"]))
	}
	return sb.String()
}

func stockAnalysisLogic(ticker, query, lang string) (map[string]any, string) {
	target := ticker
	if target == "" {
		target = query
	}
	var searchQuery string
	if lang == "en" {
		searchQuery = fmt.Sprintf("%s stock price analysis financials recent news 2025", target)
	} else {
		searchQuery = fmt.Sprintf("%s 주가 분석 재무정보 최근 뉴스 전망 2025", target)
	}
	results := proSearch(searchQuery, 8)
	ctx := proItemsToText(results, 6)
	var sysPrompt string
	if lang == "en" {
		sysPrompt = fmt.Sprintf(`You are a professional investment analyst for Nexus AI.
Analyze %s based on the search results below and provide:
1. Current price & trend (if available)
2. Key financial metrics (PER, PBR, ROE if available)
3. Recent news impact (bullish/bearish)
4. Risk factors
5. Short summary (buy/hold/watch - NOT investment advice)

⚠️ Always add: "This is for informational purposes only. Investment decisions are your responsibility."

Search results:
%s`, target, ctx)
	} else {
		sysPrompt = fmt.Sprintf(`당신은 Nexus AI의 전문 투자 분석 AI입니다.
아래 검색 결과를 바탕으로 %s 분석 리포트를 작성하세요:
1. 현재 시세 & 추세 (정보가 있으면)
2. 주요 지표 (PER, PBR, ROE 등 가능한 것)
3. 최근 뉴스 영향 (호재/악재)
4. 리스크 요인
5. 한 줄 요약 (매수검토/관망/주의 - 투자 권유 아님)

⚠️ 반드시 마지막에: "본 분석은 참고용이며 투자 판단은 본인 책임입니다." 추가

검색 결과:
%s`, target, ctx)
	}
	msgs := []groqMsg{{Role: "user", Content: sysPrompt}}
	report, _, err := callGroqWithFallback(msgs, 1024, false)
	if err != nil {
		report = ctx
	}
	return map[string]any{"ticker": target, "report": report, "sources": results, "ok": true}, report
}

func medicalSearchLogic(query, searchType, lang string) (map[string]any, string) {
	var searchQ string
	switch searchType {
	case "drug":
		searchQ = query + " drug dosage side effects interactions site:drugs.com OR site:pubmed.ncbi.nlm.nih.gov OR site:kmle.co.kr"
	case "paper":
		searchQ = query + " clinical study randomized controlled trial pubmed 2022 2023 2024 2025"
	case "guideline":
		searchQ = query + " clinical guideline recommendation 대한의학회 OR uptodate 2024 2025"
	default:
		if lang == "en" {
			searchQ = query + " medical clinical evidence pubmed"
		} else {
			searchQ = query + " 의학 임상 근거 대한의학회 최신"
		}
	}
	results := proSearch(searchQ, 8)
	ctx := proItemsToText(results, 6)
	var sysPrompt string
	if lang == "en" {
		sysPrompt = fmt.Sprintf(`You are a medical information assistant for Nexus AI (for healthcare professionals).
Summarize the following medical search results about "%s":
- Evidence level (RCT/meta-analysis/case report/guideline)
- Key findings with numbers (N, p-value, NNT if available)
- Clinical applicability
- Limitations or cautions
- References

⚠️ Add: "For clinical decisions, verify with current guidelines and specialist judgment."

Results:
%s`, query, ctx)
	} else {
		sysPrompt = fmt.Sprintf(`당신은 Nexus AI의 의학 정보 AI입니다 (의료 전문가 대상).
"%s" 관련 의학 검색 결과를 정리하세요:
- 근거 수준 (RCT/메타분석/증례보고/가이드라인)
- 핵심 결과 (N수, p값, NNT 가능한 경우)
- 임상 적용 가능성
- 주의사항 및 한계
- 출처

⚠️ 마지막에: "임상 결정 시 최신 가이드라인과 전문의 판단을 함께 확인하세요." 추가

결과:
%s`, query, ctx)
	}
	msgs := []groqMsg{{Role: "user", Content: sysPrompt}}
	summary, _, err := callGroqWithFallback(msgs, 1024, false)
	if err != nil {
		summary = ctx
	}
	return map[string]any{"query": query, "type": searchType, "summary": summary, "sources": results, "ok": true}, summary
}

func contractReviewLogic(filePath, content, focus, lang string) (map[string]any, string) {
	text := content
	if text == "" && filePath != "" {
		var err error
		text, err = extractDocumentText(filePath)
		if err != nil {
			msg := "파일 읽기 실패: " + err.Error()
			return map[string]any{"ok": false, "error": msg}, msg
		}
	}
	if text == "" {
		msg := "file_path 또는 content가 필요합니다"
		return map[string]any{"ok": false, "error": msg}, msg
	}
	if len(text) > 8000 {
		text = text[:8000]
	}
	if focus == "" {
		focus = "all"
	}
	var sysPrompt string
	if lang == "en" {
		sysPrompt = fmt.Sprintf(`You are a legal document review AI for Nexus AI (for legal professionals).
Review the following contract/legal document and provide:

1. 🔴 HIGH RISK clauses (unfair/dangerous terms, specific clause numbers)
2. 🟡 CAUTION items (ambiguous terms that need clarification)
3. ✅ Standard/acceptable clauses
4. 📝 Suggested revisions for risk items (provide exact replacement text)
5. Overall risk rating: LOW / MEDIUM / HIGH

⚠️ Add: "This review is AI-assisted. Final legal judgment requires qualified attorney review."

Document:
%s`, text)
	} else {
		sysPrompt = fmt.Sprintf(`당신은 Nexus AI의 법률 문서 검토 AI입니다 (법률 전문가 대상).
아래 계약서/법률 문서를 검토하고 다음을 제공하세요:

1. 🔴 고위험 조항 (불공정/위험 조항, 조항 번호 명시)
2. 🟡 주의 필요 항목 (모호하거나 명확화 필요한 조항)
3. ✅ 표준/수용 가능한 조항
4. 📝 위험 항목 수정 제안 (구체적인 대체 문구 제공)
5. 전체 리스크 등급: 낮음 / 보통 / 높음

⚠️ 마지막에: "본 검토는 AI 보조 결과이며 최종 법적 판단은 자격 있는 변호사 확인이 필요합니다." 추가

문서:
%s`, text)
	}
	msgs := []groqMsg{{Role: "user", Content: sysPrompt}}
	review, _, err := callGroqWithFallback(msgs, 2048, false)
	if err != nil {
		if lang == "en" {
			review = "Review failed. Please try again."
		} else {
			review = "검토 실패. 다시 시도해주세요."
		}
	}
	return map[string]any{"review": review, "focus": focus, "file_path": filePath, "ok": true}, review
}

func legalSearchLogic(query, searchType, lang string) (map[string]any, string) {
	var searchQ string
	switch searchType {
	case "case":
		searchQ = query + " 판례 대법원 site:law.go.kr OR site:casenote.kr"
	case "law":
		searchQ = query + " 법령 조문 site:law.go.kr"
	default:
		if lang == "en" {
			searchQ = query + " Korean law legal precedent court ruling"
		} else {
			searchQ = query + " 법률 판례 법령 대법원 헌법재판소"
		}
	}
	results := proSearch(searchQ, 8)
	ctx := proItemsToText(results, 5)
	var prompt string
	if lang == "en" {
		prompt = fmt.Sprintf(`Summarize the following legal search results for "%s":
- Applicable laws/precedents (with article numbers/case numbers)
- Key legal principles
- Practical implications
- Cautions

Results:
%s`, query, ctx)
	} else {
		prompt = fmt.Sprintf(`"%s" 관련 법률 검색 결과를 정리하세요:
- 관련 법령/판례 (조문 번호/사건번호 포함)
- 핵심 법리
- 실무적 시사점
- 주의사항

결과:
%s`, query, ctx)
	}
	summary, _, _ := callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 1024, false)
	return map[string]any{"query": query, "summary": summary, "sources": results, "ok": true}, summary
}

func contentScriptLogic(topic, platform, duration, style, lang string) (map[string]any, string) {
	if platform == "" {
		platform = "youtube"
	}
	if duration == "" {
		duration = "medium"
	}
	if style == "" {
		style = "educational"
	}
	var durationGuide string
	switch duration {
	case "short":
		durationGuide = "60초 이내 (숏폼/Shorts/Reels)"
	case "long":
		durationGuide = "15분 이상 (심층 콘텐츠)"
	default:
		durationGuide = "5~10분 (표준 유튜브)"
	}
	var sysPrompt string
	if lang == "en" {
		sysPrompt = fmt.Sprintf(`You are a professional content creator AI for Nexus AI.
Create a complete %s script for the topic: "%s"
Platform: %s | Duration: %s | Style: %s

Structure:
🎯 HOOK (first 5 seconds - grab attention immediately)
📌 INTRO (problem/promise - why watch this)
📋 MAIN CONTENT (3~5 key points with transitions)
🔚 OUTRO (CTA - subscribe/like/comment prompt)

Also provide:
- 3 title options (SEO-optimized, high CTR)
- 5 thumbnail text ideas
- 10 hashtags (mix of large/medium/small)
- Best upload time recommendation`, platform, topic, platform, durationGuide, style)
	} else {
		sysPrompt = fmt.Sprintf(`당신은 Nexus AI의 전문 콘텐츠 크리에이터 AI입니다.
주제 "%s"로 %s 완성 스크립트를 작성해주세요.
플랫폼: %s | 길이: %s | 스타일: %s

구성:
🎯 훅 (첫 5초 - 즉시 시선 끌기)
📌 인트로 (문제제기/약속 - 왜 봐야 하는지)
📋 본문 (핵심 포인트 3~5개, 전환 멘트 포함)
🔚 아웃트로 (CTA - 구독/좋아요/댓글 유도)

추가 제공:
- 제목 3가지 옵션 (SEO 최적화, 클릭률 높은 감정 트리거 포함)
- 썸네일 텍스트 아이디어 5개
- 해시태그 10개 (대형/중형/소형 혼합)
- 최적 업로드 시간대 추천`, topic, platform, platform, durationGuide, style)
	}
	msgs := []groqMsg{{Role: "user", Content: sysPrompt}}
	script, _, err := callGroqWithFallback(msgs, 2048, false)
	if err != nil {
		if lang == "en" {
			script = "Script generation failed. Please try again."
		} else {
			script = "스크립트 생성 실패. 다시 시도해주세요."
		}
	}
	return map[string]any{"topic": topic, "platform": platform, "script": script, "ok": true}, script
}

// ── 법률 판례/법령 검색 ────────────────────────────────────────────

func handleLegalSearch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query string `json:"query"`
		Type  string `json:"type"`
		Lang  string `json:"lang"`
	}
	tryDecodeBody(r, &req)
	lang := req.Lang
	if lang == "" {
		lang = getLang(r)
	}
	if req.Query == "" {
		w.WriteHeader(http.StatusBadRequest)
		json200(w, map[string]any{"ok": false, "error": msgT("query가 필요합니다", "query is required", lang)})
		return
	}
	res, _ := legalSearchLogic(req.Query, req.Type, lang)
	json200(w, res)
}
