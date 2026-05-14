//go:build windows

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ── Auto Workflow Agent ────────────────────────────────────────
// LLM이 목표를 단계별 API 호출 계획으로 분해하고 순차 실행

type WorkflowStep struct {
	StepNum     int    `json:"step"`
	Description string `json:"description"`
	APIEndpoint string `json:"api_endpoint"`
	Method      string `json:"method"`
	Params      any    `json:"params"`
	Status      string `json:"status"` // pending | running | done | error
	Result      string `json:"result"`
}

type WorkflowPlan struct {
	Goal    string         `json:"goal"`
	Steps   []WorkflowStep `json:"steps"`
	Summary string         `json:"summary"`
}

// LLM에게 목표를 단계별 계획으로 분해 요청
func planWorkflow(goal string) (*WorkflowPlan, error) {
	availableAPIs := `
[PC 진단·최적화]
- POST /api/scan → PC 전체 진단 (바이러스·정크·드라이버·성능)
- POST /api/clean → 정크파일·캐시 정리
- POST /api/autoclean → 임시파일 자동 정리
- POST /api/repair → 문제 자동 수리
- GET /api/stats → 실시간 CPU·메모리·디스크 상태
- GET /api/history/stats → 성능 이력 통계
- GET /api/history/anomalies → 성능 이상 탐지
- GET /api/daily-report → 데일리 PC 리포트
- GET /api/drivers → 드라이버 목록·업데이트 필요 여부
- GET /api/power/plans → 전원 플랜 목록
- POST /api/power/plan → 전원 플랜 변경 {plan: "balanced|performance|powersave"}
- GET /api/boot/analysis → 부팅 속도 분석
- GET /api/programs → 설치된 프로그램 목록
- GET /api/processes/top → CPU·메모리 상위 프로세스
- POST /api/process/kill → 프로세스 강제 종료 {pid}
- POST /api/disk/check → 디스크 검사
- POST /api/registry/clean → 레지스트리 정리
- GET /api/gpu/stats → GPU 사용량·온도
- GET /api/network/analysis → 네트워크 분석
- POST /api/system/wifi → WiFi 제어 {action: "reconnect|disable|enable"}
- POST /api/system/volume → 볼륨 제어 {action, value}
- POST /api/system/power → 전원 제어 {action: "sleep|restart|shutdown"}
- POST /api/privacy → MS 기능 차단·프라이버시 설정
- POST /api/restore/create → 복원 지점 생성

[보안]
- GET /api/security/remote → 원격 접속 도구 탐지
- GET /api/security/processes → 수상한 프로세스 검사
- GET /api/security/hosts → hosts 파일 이상 검사
- GET /api/security/startup → 시작 프로그램 목록
- GET /api/security/defender → Windows Defender 상태
- GET /api/security/accounts → 계정 보안 점검
- GET /api/security/audit → 전체 보안 감사
- POST /api/security/virustotal → 파일 바이러스 검사 {file_path}
- POST /api/security/check-path → 경로 보안 검사 {path}
- GET /api/app/permissions → 앱 권한 목록

[파일·문서]
- POST /api/files/search → 파일 검색 {query, path}
- POST /api/files/duplicates → 중복 파일 탐지 {path}
- POST /api/files/organize → 폴더 자동 정리 {path}
- POST /api/docs/summary → 문서 AI 요약 {file_path}
- POST /api/docs/compare → 문서 비교 {file_path_a, file_path_b}
- POST /api/docs/find → 문서 검색 {query, path}
- POST /api/docs/export-report → 리포트 PDF 저장 {content, title}
- POST /api/docs/ai-edit → AI 문서 편집 {file_path, instruction}
- GET /api/excel/list → 엑셀 파일 목록
- GET /api/excel/read → 엑셀 읽기 {file_path}
- POST /api/excel/save → 엑셀 저장 {file_path, data}
- POST /api/vision/ocr-clipboard → 클립보드 이미지 OCR
- POST /api/vision/screenshot → 화면 캡처 + OCR
- POST /api/file/process → 파일 변환·처리 {files, action}

[이메일·캘린더]
- GET /api/imap/inbox → 이메일 받은편지함 {limit}
- POST /api/imap/classify → 메일 AI 분류 (중요도·카테고리)
- POST /api/imap/send → 이메일 발송 {to, subject, body}
- GET /api/imap/reply-suggestions → 답장 초안 자동 생성
- POST /api/email/draft-reply → 특정 메일 답장 초안 {email_id}
- POST /api/email/extract-events → 메일에서 일정 추출
- GET /api/email/config → 이메일 계정 설정
- GET /api/calendar/today → 오늘 일정
- GET /api/calendar/week → 이번 주 일정
- POST /api/calendar/add → 일정 추가 {subject, start, duration_minutes, location}
- POST /api/calendar/find-slot → 빈 시간 탐색 {duration_minutes, within_days}
- POST /api/calendar/smart-add → 자연어로 일정 추가 {text}

[웹 검색·크롤링·쇼핑]
- POST /api/browser/news-collect → 뉴스 수집 {query, site, max_items}
- POST /api/browser/collect-price → 쇼핑 가격 수집 {query, site}
- POST /api/browser/smart-agent → AI 브라우저 자동화 {goal}
- POST /api/browser/extract → 웹페이지 내용 추출 {url}
- POST /api/browser/search-and-pdf → 검색 후 PDF 저장 {query}
- POST /api/video/quick-search → 동영상 빠른 검색 {query, platform}
- POST /api/llm/deep-search-web → 웹 딥서치 {query}
- POST /api/site-search → 특정 사이트 검색 {query, site}
- POST /api/directions → 길찾기 {from, to, mode}

[AI·메모리·리포트]
- POST /api/llm/chat → LLM 대화 {messages}
- POST /api/llm/deep-search → AI 딥서치 {query}
- POST /api/llm/doc-summary → 문서 요약 {content}
- POST /api/memory/search → 기억 검색 {query}
- GET /api/memory/list → 저장된 기억 목록
- POST /api/brain/search → Second Brain 검색 {query}
- POST /api/notes → 메모 저장 {content}
- GET /api/report/generate → PC 건강 리포트
- POST /api/report/email → 리포트 이메일 발송 {email}
- POST /api/briefing/now → 아침 브리핑 즉시 실행
- GET /api/weather → 날씨 정보

[스케줄·자동화]
- POST /api/scheduler/add → 스케줄 등록 {command} (자연어)
- GET /api/scheduler/list → 스케줄 목록
- POST /api/productivity/focus → 집중 모드 {duration_minutes}
- POST /api/workflow/plan → 워크플로 계획 수립 {goal}
`

	prompt := fmt.Sprintf(`당신은 자동화 워크플로 플래너입니다.
사용자 목표: "%s"

%s

위 API를 조합하여 목표를 달성하는 단계별 실행 계획을 JSON으로 반환하세요.
반드시 아래 형식을 지키세요:

{
  "steps": [
    {
      "step": 1,
      "description": "단계 설명",
      "api_endpoint": "/api/endpoint",
      "method": "GET 또는 POST",
      "params": {}
    }
  ],
  "summary": "전체 워크플로 한 줄 요약"
}

규칙:
- 최대 6단계
- 각 단계는 실제 API 엔드포인트만 사용
- params는 해당 API에 필요한 JSON 파라미터
- 필요 없으면 params는 {}
- JSON만 반환, 설명 없음`, goal, availableAPIs)

	raw, _, err := callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 1024, true)
	if err != nil {
		return nil, err
	}

	var plan WorkflowPlan
	raw = strings.TrimSpace(raw)
	if err := json.Unmarshal([]byte(raw), &plan); err != nil {
		return nil, fmt.Errorf("계획 파싱 실패: %v", err)
	}
	plan.Goal = goal
	for i := range plan.Steps {
		plan.Steps[i].Status = "pending"
		if plan.Steps[i].StepNum == 0 {
			plan.Steps[i].StepNum = i + 1
		}
	}
	return &plan, nil
}

// 단계 실행: 로컬 백엔드 API 호출
func executeWorkflowStep(step *WorkflowStep) error {
	step.Status = "running"
	url := "http://127.0.0.1:17891" + step.APIEndpoint

	var body []byte
	var err error
	if step.Method == "POST" {
		body, err = json.Marshal(step.Params)
		if err != nil {
			return err
		}
	}

	var resp *http.Response
	client := &http.Client{Timeout: 30 * time.Second}
	if step.Method == "POST" {
		resp, err = client.Post(url, "application/json", bytes.NewReader(body))
	} else {
		resp, err = client.Get(url)
	}
	if err != nil {
		step.Status = "error"
		step.Result = "API 호출 실패: " + err.Error()
		return err
	}
	defer resp.Body.Close()

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	// 결과에서 핵심 텍스트 추출
	resultText := extractWorkflowResult(result)
	step.Result = resultText
	step.Status = "done"
	return nil
}

func extractWorkflowResult(result map[string]any) string {
	// 우선순위 키에서 결과 추출
	for _, key := range []string{"message", "summary", "answer", "result", "status", "text"} {
		if v, ok := result[key]; ok {
			return fmt.Sprintf("%v", v)
		}
	}
	// fallback: JSON 마샬
	b, _ := json.Marshal(result)
	s := string(b)
	if len(s) > 200 {
		s = s[:200] + "..."
	}
	return s
}

// ── HTTP 핸들러 ────────────────────────────────────────────────

func handleWorkflowPlan(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Goal string `json:"goal"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Goal == "" {
		writeJSON(w, 400, map[string]string{"error": "goal 필드가 필요합니다"})
		return
	}
	plan, err := planWorkflow(req.Goal)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "계획 생성 실패: " + err.Error()})
		return
	}
	json200(w, plan)
}

// ── Reflection Loop 핵심 구조 ─────────────────────────────────
//
// criticWorkflow: LLM이 실행 결과를 평가 → 목표 달성 여부 판단
type CriticResult struct {
	Satisfied bool     `json:"satisfied"`  // 목표 달성 여부
	Reason    string   `json:"reason"`     // 판단 이유
	Missing   []string `json:"missing"`    // 부족한 항목
}

func criticWorkflow(goal string, stepResults []string) CriticResult {
	prompt := fmt.Sprintf(`당신은 AI 비평가입니다. 아래 목표와 실행 결과를 보고 목표가 충분히 달성되었는지 판단하세요.

목표: "%s"

실행 결과:
%s

아래 JSON만 반환하세요:
{
  "satisfied": true 또는 false,
  "reason": "판단 이유 한 문장",
  "missing": ["부족한 항목1", "부족한 항목2"]
}

satisfied=true이면 missing은 빈 배열.
JSON만 반환, 설명 없음.`, goal, strings.Join(stepResults, "\n"))

	raw, _, err := callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 256, true)
	if err != nil {
		return CriticResult{Satisfied: true} // 판단 실패 시 완료로 처리
	}
	var result CriticResult
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &result); err != nil {
		return CriticResult{Satisfied: true}
	}
	return result
}

// replanWorkflow: Critic이 부족하다고 한 항목만 재계획
func replanWorkflow(goal string, missing []string) ([]WorkflowStep, error) {
	missingStr := strings.Join(missing, ", ")
	prompt := fmt.Sprintf(`목표: "%s"
아직 달성되지 않은 항목: %s

이 항목들만 처리하는 추가 단계를 JSON으로 반환하세요.
{
  "steps": [
    {"step": 1, "description": "...", "api_endpoint": "/api/...", "method": "GET또는POST", "params": {}}
  ]
}
최대 3단계. JSON만 반환.`, goal, missingStr)

	raw, _, err := callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 512, true)
	if err != nil {
		return nil, err
	}
	var plan struct {
		Steps []WorkflowStep `json:"steps"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &plan); err != nil {
		return nil, err
	}
	for i := range plan.Steps {
		plan.Steps[i].Status = "pending"
	}
	return plan.Steps, nil
}

// runWithReflection: Planner → Executor → Critic → Re-plan 루프 (최대 3회)
func runWithReflection(goal string) ([]WorkflowStep, string, int) {
	const maxIterations = 3
	var allSteps []WorkflowStep
	var allResults []string
	iteration := 0

	// 초기 계획 수립
	plan, err := planWorkflow(goal)
	if err != nil {
		return nil, "계획 생성 실패: " + err.Error(), 0
	}
	pendingSteps := plan.Steps

	for iteration < maxIterations && len(pendingSteps) > 0 {
		iteration++

		// 현재 대기 단계들 실행
		for i := range pendingSteps {
			step := &pendingSteps[i]
			executeWorkflowStep(step)
			allSteps = append(allSteps, *step)
			allResults = append(allResults, fmt.Sprintf("[%d회차 %d단계] %s: %s",
				iteration, step.StepNum, step.Description, step.Result))
		}

		// Critic: 결과 평가
		critic := criticWorkflow(goal, allResults)
		if critic.Satisfied || len(critic.Missing) == 0 || iteration >= maxIterations {
			break
		}

		// Re-plan: 부족한 부분만 재계획
		extraSteps, err := replanWorkflow(goal, critic.Missing)
		if err != nil || len(extraSteps) == 0 {
			break
		}
		for i := range extraSteps {
			extraSteps[i].StepNum = len(allSteps) + i + 1
		}
		pendingSteps = extraSteps
	}

	// 최종 요약 생성
	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()

	finalSummary := fmt.Sprintf("총 %d회 반복, %d단계 실행 완료", iteration, len(allSteps))
	if gKey != "" {
		prompt := fmt.Sprintf(`목표: "%s"
전체 실행 결과 (%d회 반복):
%s

사용자에게 친근하게 최종 완료 보고를 2-3문장으로 해주세요.`,
			goal, iteration, strings.Join(allResults, "\n"))
		if summary, _, err := callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 300, false); err == nil && summary != "" {
			finalSummary = summary
		}
	}

	return allSteps, finalSummary, iteration
}

func handleWorkflowRun(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Goal            string `json:"goal"`
		UseReflection   bool   `json:"use_reflection"` // true면 Reflection Loop 사용
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Goal == "" {
		writeJSON(w, 400, map[string]string{"error": "goal 필드가 필요합니다"})
		return
	}

	// Reflection Loop 모드 (기본값: true)
	if req.UseReflection || true {
		steps, summary, iterations := runWithReflection(req.Goal)
		json200(w, map[string]any{
			"goal":       req.Goal,
			"steps":      steps,
			"summary":    summary,
			"iterations": iterations,
			"ok":         true,
			"mode":       "reflection",
		})
		return
	}

	// 기존 단순 실행 (fallback)
	plan, err := planWorkflow(req.Goal)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "계획 생성 실패: " + err.Error()})
		return
	}
	var executedSteps []WorkflowStep
	var stepResults []string
	for i := range plan.Steps {
		step := &plan.Steps[i]
		executeWorkflowStep(step)
		executedSteps = append(executedSteps, *step)
		stepResults = append(stepResults, fmt.Sprintf("%d단계 (%s): %s", step.StepNum, step.Description, step.Result))
	}
	json200(w, map[string]any{
		"goal":    req.Goal,
		"steps":   executedSteps,
		"summary": plan.Summary,
		"ok":      true,
		"mode":    "simple",
	})
}
