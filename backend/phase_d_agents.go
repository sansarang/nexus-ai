// phase_d_agents.go — Phase D: 12개 분야 자동 수정 Agent
// 사장님 비전: 자가 치유 자비스 — 각 Agent가 자기 분야 문제 감지 + 패치 제안
//
// Agent 종류 (12):
//   1. Go Backend Engineer
//   2. Frontend/Tauri Engineer
//   3. Card Designer Engineer
//   4. Python Sidecar Engineer
//   5. LLM Prompt Engineer
//   6. Korean NLP Engineer
//   7. LLM Integration Engineer
//   8. RAG/Knowledge Engineer
//   9. Domain API Engineer
//   10. Security/Safety Engineer
//   11. Build/Deploy Engineer
//   12. Test/Validation Engineer

package main

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// Severity 4단계 — 사용자 승인 정책
type PatchSeverity string

const (
	SevInfo     PatchSeverity = "info"     // 알림만
	SevLow      PatchSeverity = "low"      // 자동 적용 (제한 파일만)
	SevMedium   PatchSeverity = "medium"   // 알림 후 10분 자동
	SevHigh     PatchSeverity = "high"     // 사용자 승인 필수
	SevCritical PatchSeverity = "critical" // 2단계 승인
)

// PatchProposal — Agent가 제안하는 단일 패치
type PatchProposal struct {
	ID             string         `json:"id"`
	AgentName      string         `json:"agent"`
	Severity       PatchSeverity  `json:"severity"`
	Title          string         `json:"title"`
	Description    string         `json:"description"`
	TargetFile     string         `json:"target_file"`
	TargetSymbol   string         `json:"target_symbol,omitempty"`
	ChangeSummary  string         `json:"change_summary"`
	Reasoning      string         `json:"reasoning"`
	Evidence       map[string]any `json:"evidence"` // 어떤 메트릭 임계치 도달
	ProposedAt     int64          `json:"proposed_at"`
	AutoApplyAfter int64          `json:"auto_apply_after,omitempty"`
	Status         string         `json:"status"` // proposed / approved / applied / rejected / rolled_back
}

// Agent 인터페이스
type SelfHealingAgent interface {
	Name() string
	Domain() string
	Analyze(stats TelemetryStats, entries []TelemetryEntry) []PatchProposal
}

// 모든 Agent 등록 (Phase D 자가치유)
var phaseDAgents = []SelfHealingAgent{
	&GoBackendAgent{},
	&FrontendTauriAgent{},
	&CardDesignerAgent{},
	&PythonSidecarAgent{},
	&LLMPromptAgent{},
	&KoreanNLPAgent{},
	&LLMIntegrationAgent{},
	&RAGKnowledgeAgent{},
	&DomainAPIAgent{},
	&SecurityAgent{},
	&BuildDeployAgent{},
	&TestValidationAgent{},
}

// ── Helper ─────────────────────────────────────────────

func newProposal(agent, title, desc, file string, sev PatchSeverity, evidence map[string]any) PatchProposal {
	autoAt := int64(0)
	switch sev {
	case SevLow:
		autoAt = time.Now().Unix() + 60 // 1분 후
	case SevMedium:
		autoAt = time.Now().Unix() + 600 // 10분 후
	}
	return PatchProposal{
		ID:             fmt.Sprintf("p-%d-%s", time.Now().UnixNano(), agent),
		AgentName:      agent,
		Severity:       sev,
		Title:          title,
		Description:    desc,
		TargetFile:     file,
		Evidence:       evidence,
		ProposedAt:     time.Now().Unix(),
		AutoApplyAfter: autoAt,
		Status:         "proposed",
	}
}

// ════════════════════════════════════════════════════════════
// 1. Go Backend Engineer — handler 누락, case 미존재, dispatch 오류
// ════════════════════════════════════════════════════════════
type GoBackendAgent struct{}

func (a *GoBackendAgent) Name() string   { return "GoBackend" }
func (a *GoBackendAgent) Domain() string { return "backend/handlers_*.go" }

func (a *GoBackendAgent) Analyze(stats TelemetryStats, entries []TelemetryEntry) []PatchProposal {
	var out []PatchProposal
	// 패턴 1: 빈 응답 action별 카운트 → 핸들러 case 누락 의심
	for action, cnt := range stats.EmptyByAction {
		if cnt >= 3 {
			out = append(out, newProposal(
				a.Name(),
				fmt.Sprintf("'%s' 액션 빈 응답 %d회", action, cnt),
				fmt.Sprintf("최근 %d회 호출 중 %d회 빈 응답. handler case 또는 응답 누락 의심.", stats.Total, cnt),
				"backend/handlers_command.go",
				SevMedium,
				map[string]any{"action": action, "empty_count": cnt, "total": stats.Total},
			))
		}
	}
	return out
}

// ════════════════════════════════════════════════════════════
// 2. Frontend/Tauri Engineer
// ════════════════════════════════════════════════════════════
type FrontendTauriAgent struct{}

func (a *FrontendTauriAgent) Name() string   { return "FrontendTauri" }
func (a *FrontendTauriAgent) Domain() string { return "src/components/FloatingCharacter/*.tsx" }

func (a *FrontendTauriAgent) Analyze(stats TelemetryStats, entries []TelemetryEntry) []PatchProposal {
	var out []PatchProposal
	// 패턴: action 은 있는데 card_type 없는 응답 비율 ↑ → 프론트 case 누락
	noCardCnt := 0
	for _, e := range entries {
		if e.Action != "" && e.Action != "clarify" && e.Action != "chat" && e.CardType == "" {
			noCardCnt++
		}
	}
	if stats.Total > 0 && float64(noCardCnt)/float64(stats.Total) > 0.15 {
		out = append(out, newProposal(
			a.Name(),
			"카드 타입 누락 응답 비율 15%+",
			"renderCommandResult default 폴백에 액션 라벨 자동 추가 권장.",
			"src/components/FloatingCharacter/index.tsx",
			SevLow,
			map[string]any{"no_card_count": noCardCnt, "total": stats.Total},
		))
	}
	return out
}

// ════════════════════════════════════════════════════════════
// 3. Card Designer Engineer
// ════════════════════════════════════════════════════════════
type CardDesignerAgent struct{}

func (a *CardDesignerAgent) Name() string   { return "CardDesigner" }
func (a *CardDesignerAgent) Domain() string { return "src/components/FloatingCharacter/InlineCards*.tsx" }

func (a *CardDesignerAgent) Analyze(stats TelemetryStats, entries []TelemetryEntry) []PatchProposal {
	var out []PatchProposal
	// 새 card_type 가 등장하면 (기존 19종 외) → 카드 신규 디자인 제안
	knownCards := map[string]bool{
		"pc_status": true, "scan_result": true, "clean_result": true, "weather_card": true,
		"price_compare": true, "system_action": true, "web_search": true, "news_search": true,
		"youtube": true, "doc_compare": true, "vision_result": true, "macro_list": true,
		"confirm_action": true, "agent_thinking": true, "repair_result": true, "daily_report": true,
		"email_list": true, "focus_mode": true, "boot_analysis": true, "dynamic": true,
	}
	newCards := map[string]int{}
	for _, e := range entries {
		if e.CardType != "" && !knownCards[e.CardType] {
			newCards[e.CardType]++
		}
	}
	for ct, cnt := range newCards {
		if cnt >= 5 {
			out = append(out, newProposal(
				a.Name(),
				fmt.Sprintf("신규 카드 타입 '%s' %d회 등장", ct, cnt),
				"전용 카드 디자인 컴포넌트 추가 권장.",
				"src/components/FloatingCharacter/InlineCards5.tsx",
				SevHigh,
				map[string]any{"card_type": ct, "count": cnt},
			))
		}
	}
	return out
}

// ════════════════════════════════════════════════════════════
// 4. Python Sidecar Engineer
// ════════════════════════════════════════════════════════════
type PythonSidecarAgent struct{}

func (a *PythonSidecarAgent) Name() string   { return "PythonSidecar" }
func (a *PythonSidecarAgent) Domain() string { return "backend/nexus_python/main.py" }

func (a *PythonSidecarAgent) Analyze(stats TelemetryStats, entries []TelemetryEntry) []PatchProposal {
	var out []PatchProposal
	// 패턴: vision/ocr/pdf-extract 등 sidecar 의존 액션이 timeout 다발
	sidecarActions := []string{"vision", "screen_analyze", "search_pdf", "doc_summary"}
	slowSidecar := 0
	for _, e := range entries {
		for _, sa := range sidecarActions {
			if e.Action == sa && e.DurationMs > 8000 {
				slowSidecar++
				break
			}
		}
	}
	if slowSidecar >= 3 {
		out = append(out, newProposal(
			a.Name(),
			"Python sidecar 응답 지연",
			fmt.Sprintf("vision/ocr/pdf 액션 %d회 8초 초과. sidecar 재시작 또는 endpoint 최적화 권장.", slowSidecar),
			"backend/nexus_python/main.py",
			SevMedium,
			map[string]any{"slow_count": slowSidecar},
		))
	}
	return out
}

// ════════════════════════════════════════════════════════════
// 5. LLM Prompt Engineer
// ════════════════════════════════════════════════════════════
type LLMPromptAgent struct{}

func (a *LLMPromptAgent) Name() string   { return "LLMPrompt" }
func (a *LLMPromptAgent) Domain() string { return "nexusSystemPrompt + persona prompts" }

func (a *LLMPromptAgent) Analyze(stats TelemetryStats, entries []TelemetryEntry) []PatchProposal {
	var out []PatchProposal
	// clarify 비율 > 12% → 의도 분류 프롬프트 약함
	if stats.ClarifyRate > 0.12 && stats.Total >= 20 {
		out = append(out, newProposal(
			a.Name(),
			fmt.Sprintf("Clarify 비율 %.1f%% (>12%%)", stats.ClarifyRate*100),
			"의도 분류 프롬프트 또는 사전 패턴 누락. 자주 clarify로 떨어진 메시지 패턴 추가 권장.",
			"backend/handlers_command.go",
			SevMedium,
			map[string]any{"clarify_rate": stats.ClarifyRate, "total": stats.Total},
		))
	}
	return out
}

// ════════════════════════════════════════════════════════════
// 6. Korean NLP Engineer
// ════════════════════════════════════════════════════════════
type KoreanNLPAgent struct{}

func (a *KoreanNLPAgent) Name() string   { return "KoreanNLP" }
func (a *KoreanNLPAgent) Domain() string { return "systemPatterns / detectMultiStep / detectPersonaForQuery" }

func (a *KoreanNLPAgent) Analyze(stats TelemetryStats, entries []TelemetryEntry) []PatchProposal {
	var out []PatchProposal
	// 패턴: clarify로 떨어진 메시지의 공통 키워드 추출
	clarifyMsgs := []string{}
	for _, e := range entries {
		if e.Clarify {
			clarifyMsgs = append(clarifyMsgs, e.UserMessage)
		}
	}
	if len(clarifyMsgs) >= 3 {
		out = append(out, newProposal(
			a.Name(),
			fmt.Sprintf("미인식 한국어 패턴 %d건", len(clarifyMsgs)),
			"systemPatterns에 새 키워드 추가 권장. 예: "+strings.Join(clarifyMsgs[:phaseDMin(3, len(clarifyMsgs))], " / "),
			"backend/handlers_command_stub.go",
			SevLow,
			map[string]any{"samples": clarifyMsgs[:phaseDMin(5, len(clarifyMsgs))]},
		))
	}
	return out
}

// ════════════════════════════════════════════════════════════
// 7. LLM Integration Engineer
// ════════════════════════════════════════════════════════════
type LLMIntegrationAgent struct{}

func (a *LLMIntegrationAgent) Name() string   { return "LLMIntegration" }
func (a *LLMIntegrationAgent) Domain() string { return "callGroqWithFallback / Ollama / OpenAI" }

func (a *LLMIntegrationAgent) Analyze(stats TelemetryStats, entries []TelemetryEntry) []PatchProposal {
	var out []PatchProposal
	// P95 > 5000ms → timeout/maxTokens 단축
	if stats.P95DurationMs > 5000 && stats.Total >= 10 {
		out = append(out, newProposal(
			a.Name(),
			fmt.Sprintf("P95 응답 시간 %.1fs", stats.P95DurationMs/1000),
			"maxTokens 단축 또는 비동기 응답 권장. 자동문서 생성에서 발생 빈도 높음.",
			"backend/handlers_pdf_auto_shared.go",
			SevHigh,
			map[string]any{"p95_ms": stats.P95DurationMs, "slow_actions": stats.SlowActions},
		))
	}
	// 빈 응답률 > 5%
	if stats.EmptyRate > 0.05 && stats.Total >= 20 {
		out = append(out, newProposal(
			a.Name(),
			fmt.Sprintf("빈 응답 비율 %.1f%%", stats.EmptyRate*100),
			"callGroqWithFallback 폴백 체인 점검. Ollama 미설치 또는 키 만료 의심.",
			"backend/handlers_llm.go",
			SevCritical,
			map[string]any{"empty_rate": stats.EmptyRate, "empty_actions": stats.EmptyByAction},
		))
	}
	return out
}

// ════════════════════════════════════════════════════════════
// 8. RAG/Knowledge Engineer
// ════════════════════════════════════════════════════════════
type RAGKnowledgeAgent struct{}

func (a *RAGKnowledgeAgent) Name() string   { return "RAGKnowledge" }
func (a *RAGKnowledgeAgent) Domain() string { return "phase_b_rag.go / brain_index" }

func (a *RAGKnowledgeAgent) Analyze(stats TelemetryStats, entries []TelemetryEntry) []PatchProposal {
	var out []PatchProposal
	// 패턴: 사용자 메시지에 파일/문서 키워드 + chat 폴백
	docKeywords := []string{"문서", "파일", "어디 있", "예전에", "지난번", "찾아", "보낸", "썼던"}
	hitsWithoutRAG := 0
	for _, e := range entries {
		if e.Action != "chat" {
			continue
		}
		for _, kw := range docKeywords {
			if strings.Contains(e.UserMessage, kw) {
				hitsWithoutRAG++
				break
			}
		}
	}
	if hitsWithoutRAG >= 3 {
		stats_ := ragStats()
		total := 0
		if v, ok := stats_["total_docs"].(int); ok {
			total = v
		}
		if total < 10 {
			out = append(out, newProposal(
				a.Name(),
				fmt.Sprintf("RAG 인덱스 %d건 (적음)", total),
				"바탕화면/문서 폴더 인덱스 부족. 파일 확장자 추가 또는 chunk 전략 개선 권장.",
				"backend/phase_b_rag.go",
				SevMedium,
				map[string]any{"index_size": total, "doc_queries_without_rag": hitsWithoutRAG},
			))
		}
	}
	return out
}

// ════════════════════════════════════════════════════════════
// 9. Domain API Engineer
// ════════════════════════════════════════════════════════════
type DomainAPIAgent struct{}

func (a *DomainAPIAgent) Name() string   { return "DomainAPI" }
func (a *DomainAPIAgent) Domain() string { return "phase_b_domain_apis.go (PubMed/law/DART/GitHub)" }

func (a *DomainAPIAgent) Analyze(stats TelemetryStats, entries []TelemetryEntry) []PatchProposal {
	var out []PatchProposal
	// medical/legal/investor/developer 페르소나 자동매칭 + 응답 느리거나 빈
	domainActions := map[string]string{
		"medical_search":  "medical",
		"legal_search":    "legal",
		"contract_review": "legal",
		"stock_analysis":  "investor",
	}
	slowCnt := 0
	for _, e := range entries {
		if _, ok := domainActions[e.Action]; ok && e.DurationMs > 6000 {
			slowCnt++
		}
	}
	if slowCnt >= 2 {
		out = append(out, newProposal(
			a.Name(),
			fmt.Sprintf("도메인 API 의존 액션 지연 %d건", slowCnt),
			"PubMed/law.go.kr 응답 시간 ↑ — domainHTTPClient timeout 조정 또는 fallback URL 전환 권장.",
			"backend/phase_b_domain_apis.go",
			SevMedium,
			map[string]any{"slow_count": slowCnt},
		))
	}
	return out
}

// ════════════════════════════════════════════════════════════
// 10. Security Engineer
// ════════════════════════════════════════════════════════════
type SecurityAgent struct{}

func (a *SecurityAgent) Name() string   { return "Security" }
func (a *SecurityAgent) Domain() string { return "phase_a_safety.go / handlers_security.go" }

func (a *SecurityAgent) Analyze(stats TelemetryStats, entries []TelemetryEntry) []PatchProposal {
	var out []PatchProposal
	// 위험 액션이 confirmed 없이 통과한 경우 (있어선 안 됨)
	dangerous := map[string]bool{"shutdown": true, "restart": true, "format_disk": true, "payment": true, "file_delete": true}
	bypassed := 0
	for _, e := range entries {
		if dangerous[e.Action] && e.CardType != "confirm_action" {
			if c, _ := e.Params["confirmed"].(bool); !c {
				bypassed++
			}
		}
	}
	if bypassed > 0 {
		out = append(out, newProposal(
			a.Name(),
			fmt.Sprintf("위험 액션 확인 우회 %d건", bypassed),
			"위험 액션이 confirm_action 카드 없이 실행됨. detectDangerousInMessage 우선순위 강화 필요.",
			"backend/phase_a_safety.go",
			SevCritical,
			map[string]any{"bypassed": bypassed},
		))
	}
	return out
}

// ════════════════════════════════════════════════════════════
// 11. Build/Deploy Engineer
// ════════════════════════════════════════════════════════════
type BuildDeployAgent struct{}

func (a *BuildDeployAgent) Name() string   { return "BuildDeploy" }
func (a *BuildDeployAgent) Domain() string { return "src-tauri/tauri.conf.json / GitHub Actions" }

func (a *BuildDeployAgent) Analyze(stats TelemetryStats, entries []TelemetryEntry) []PatchProposal {
	// Build agent는 텔레메트리 기반보단 git/CI 이벤트 기반 — MVP에선 스킵
	// 향후: 빌드 실패율 추적 / Tauri 버전 업데이트 알림
	return nil
}

// ════════════════════════════════════════════════════════════
// 12. Test/Validation Engineer
// ════════════════════════════════════════════════════════════
type TestValidationAgent struct{}

func (a *TestValidationAgent) Name() string   { return "TestValidation" }
func (a *TestValidationAgent) Domain() string { return "e2e_truth.py / e2e_phase_validation.py" }

func (a *TestValidationAgent) Analyze(stats TelemetryStats, entries []TelemetryEntry) []PatchProposal {
	var out []PatchProposal
	// 패치 적용 후 회귀 검증 트리거 — patcher에서 신호 받음 (MVP: 단순 알림)
	if stats.Total > 0 && stats.EmptyRate+stats.ClarifyRate > 0.30 {
		out = append(out, newProposal(
			a.Name(),
			"전체 품질 지표 하락",
			fmt.Sprintf("빈 응답 + clarify 합계 %.1f%% (>30%%) — e2e 자동 회귀 실행 권장.", (stats.EmptyRate+stats.ClarifyRate)*100),
			"e2e_truth.py",
			SevHigh,
			map[string]any{"total_failure_rate": stats.EmptyRate + stats.ClarifyRate},
		))
	}
	return out
}

// ── 전체 Agent 실행 ─────────────────────────────────────────

var (
	agentRunMu sync.Mutex
)

// runAllAgents — 모든 등록된 Agent 실행 → 패치 제안 큐에 추가
func runAllAgents() []PatchProposal {
	agentRunMu.Lock()
	defer agentRunMu.Unlock()

	entries := getTelemetrySnapshot(500)
	if len(entries) < 5 {
		return nil // 데이터 너무 적음
	}
	stats := computeTelemetryStats(entries)

	var allProposals []PatchProposal
	for _, agent := range phaseDAgents {
		proposals := agent.Analyze(stats, entries)
		allProposals = append(allProposals, proposals...)
	}
	return allProposals
}

func phaseDMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}
