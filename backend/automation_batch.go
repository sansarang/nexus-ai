// automation_batch.go — 템플릿 워크플로의 N행 배치 반복 + 성공률 집계 (cross-platform)
//
// RPA 핵심 가치: "한 번 가르치면 N행 반복". 사용자가 한 행을 시연/녹화한 템플릿의
// {{col}} placeholder를 데이터셋(rows)으로 치환해 행별로 닫힌 루프 실행하고,
// 행별 성공/실패 + 전체 성공률을 집계한다.
// (실제 UIA 실행은 GetAutomator()로 위임 — 오케스트레이션 로직 자체는 mock으로 검증 가능.)
package main

import "strings"

// expandPlaceholders — "{{key}}" → row[key]. 미존재 키는 빈 문자열, 닫힘 없는 "{{"는 원문 유지.
func expandPlaceholders(s string, row map[string]string) string {
	if !strings.Contains(s, "{{") {
		return s
	}
	var b strings.Builder
	for {
		i := strings.Index(s, "{{")
		if i < 0 {
			b.WriteString(s)
			break
		}
		b.WriteString(s[:i])
		rest := s[i+2:]
		j := strings.Index(rest, "}}")
		if j < 0 {
			b.WriteString(s[i:]) // 닫는 "}}" 없음 → 원문 보존
			break
		}
		key := strings.TrimSpace(rest[:j])
		b.WriteString(row[key])
		s = rest[j+2:]
	}
	return b.String()
}

// expandStep — 한 단계의 Value/Expect/Selector(Name·AutomationID) placeholder를 행으로 치환.
// Selector.Role/Index는 의도적으로 치환 대상 아님(컨트롤 타입·순번은 행마다 고정).
func expandStep(st AutoStep, row map[string]string) AutoStep {
	out := st
	out.Value = expandPlaceholders(st.Value, row)
	out.Expect = expandPlaceholders(st.Expect, row)
	out.Selector.Name = expandPlaceholders(st.Selector.Name, row)
	out.Selector.AutomationID = expandPlaceholders(st.Selector.AutomationID, row)
	return out
}

// expandSteps — 템플릿 단계 시퀀스를 한 행으로 구체화.
func expandSteps(template []AutoStep, row map[string]string) []AutoStep {
	out := make([]AutoStep, len(template))
	for i, st := range template {
		out[i] = expandStep(st, row)
	}
	return out
}

// RowResult — 한 행 처리 결과.
type RowResult struct {
	Index int               `json:"index"`
	Row   map[string]string `json:"row"`
	OK    bool              `json:"ok"`
	Run   RunResult         `json:"run"`
}

// BatchResult — N행 배치 결과 + 성공률.
type BatchResult struct {
	Total       int         `json:"total"`
	Succeeded   int         `json:"succeeded"`
	SuccessRate float64     `json:"success_rate"`
	Rows        []RowResult `json:"rows"`
}

// BatchRun — 템플릿을 각 행으로 확장해 닫힌 루프로 실행하고 성공률을 집계한다.
//   - 엔진 미가용이면 아무 행도 실행하지 않음(안전 거부, SuccessRate=0).
//   - stopOnError=true면 첫 실패 행에서 중단(잘못된 자동화 폭주 방지), false면 전 행 시도.
func BatchRun(a UIAutomator, template []AutoStep, rows []map[string]string, stopOnError bool) BatchResult {
	res := BatchResult{Total: len(rows)}
	if a == nil || !a.Available() {
		return res
	}
	for i, row := range rows {
		rr := RunSteps(a, expandSteps(template, row))
		res.Rows = append(res.Rows, RowResult{Index: i, Row: row, OK: rr.OK, Run: rr})
		if rr.OK {
			res.Succeeded++
		} else if stopOnError {
			break
		}
	}
	if res.Total > 0 {
		res.SuccessRate = float64(res.Succeeded) / float64(res.Total)
	}
	return res
}
