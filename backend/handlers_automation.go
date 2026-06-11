// handlers_automation.go — 데스크탑 자동화 엔진 HTTP API (cross-platform)
//
// 안전 원칙:
//   - 실제 실행은 GetAutomator().Available() 게이트를 통과해야만 한다 (미가용 시 501, 무단 실행 금지).
//   - dry_run=true면 실행 없이 단계만 미리보기(파괴적 동작 사전 확인).
//   - 실제 UIA 동작 검증은 Windows 머신에서 (windowsAutomator 참고).
package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"runtime"
	"sort"
	"strings"

	"github.com/xuri/excelize/v2"
)

func automationPlatform() string { return runtime.GOOS }

func automationStatusMessage(avail bool) string {
	if avail {
		return "자동화 엔진 준비됨"
	}
	return "데스크탑 자동화는 Windows에서 사용 가능합니다 (UIA 엔진 준비 중)"
}

// GET /api/automation/status — 엔진 가용성 조회.
func handleAutomationStatus(w http.ResponseWriter, _ *http.Request) {
	a := GetAutomator()
	json200(w, map[string]any{
		"success":   true,
		"available": a.Available(),
		"platform":  automationPlatform(),
		"message":   automationStatusMessage(a.Available()),
	})
}

// GET  /api/automation/workflows — 저장된 워크플로 목록.
// POST /api/automation/workflows — {name, steps} 저장.
func handleAutomationWorkflows(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var req struct {
			Name  string     `json:"name"`
			Steps []AutoStep `json:"steps"`
		}
		if !tryDecodeBody(r, &req) || strings.TrimSpace(req.Name) == "" {
			writeJSON(w, 400, map[string]any{"success": false, "message": "name 필요"})
			return
		}
		wf := NewAutoWorkflow(req.Name, req.Steps)
		path, err := wf.Save()
		if err != nil {
			writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
			return
		}
		json200(w, map[string]any{"success": true, "name": req.Name, "path": path, "steps": len(req.Steps)})
		return
	}
	names, err := ListAutoWorkflows()
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}
	json200(w, map[string]any{"success": true, "workflows": names, "count": len(names)})
}

// GET /api/automation/workflows/{name} — 단일 워크플로 로드.
func handleAutomationWorkflowGet(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	wf, err := LoadAutoWorkflow(name)
	if err != nil {
		writeJSON(w, 404, map[string]any{"success": false, "message": "워크플로 없음: " + name})
		return
	}
	json200(w, map[string]any{"success": true, "workflow": wf})
}

// POST /api/automation/run — {steps:[...]} 또는 {workflow_name}, {dry_run}.
func handleAutomationRun(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Steps        []AutoStep `json:"steps"`
		WorkflowName string     `json:"workflow_name"`
		DryRun       bool       `json:"dry_run"`
	}
	tryDecodeBody(r, &req)

	steps := req.Steps
	if len(steps) == 0 && strings.TrimSpace(req.WorkflowName) != "" {
		wf, err := LoadAutoWorkflow(req.WorkflowName)
		if err != nil {
			writeJSON(w, 404, map[string]any{"success": false, "message": "워크플로 없음: " + req.WorkflowName})
			return
		}
		steps = wf.Steps
	}
	if len(steps) == 0 {
		writeJSON(w, 400, map[string]any{"success": false, "message": "steps 또는 workflow_name 필요"})
		return
	}

	// dry-run: 실행 없이 단계 미리보기 (파괴적 동작 사전 확인 게이트).
	if req.DryRun {
		json200(w, map[string]any{"success": true, "dry_run": true, "steps": steps, "count": len(steps)})
		return
	}

	a := GetAutomator()
	if !a.Available() {
		writeJSON(w, 501, map[string]any{
			"success":   false,
			"code":      "automation_unavailable",
			"available": false,
			"platform":  automationPlatform(),
			"message":   automationStatusMessage(false),
		})
		return
	}
	res := RunSteps(a, steps)
	code := http.StatusOK
	if !res.OK {
		code = http.StatusMultiStatus // 207 — 일부/전체 단계 실패
	}
	writeJSON(w, code, map[string]any{"success": res.OK, "result": res})
}

// POST /api/automation/workflows/{name}/replay — 저장된 워크플로 재생.
func handleAutomationReplay(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	wf, err := LoadAutoWorkflow(name)
	if err != nil {
		writeJSON(w, 404, map[string]any{"success": false, "message": "워크플로 없음: " + name})
		return
	}
	a := GetAutomator()
	if !a.Available() {
		writeJSON(w, 501, map[string]any{
			"success":   false,
			"code":      "automation_unavailable",
			"available": false,
			"platform":  automationPlatform(),
			"message":   automationStatusMessage(false),
		})
		return
	}
	res := wf.Replay()
	code := http.StatusOK
	if !res.OK {
		code = http.StatusMultiStatus
	}
	writeJSON(w, code, map[string]any{"success": res.OK, "result": res})
}

// POST /api/automation/batch — 템플릿을 N행 데이터로 반복 실행 (RPA 핵심: "한 번 가르치면 N행 반복").
// 템플릿: {steps:[...]} 직접 전달 또는 {workflow_name} 로 저장된 워크플로 로드.
// 데이터셋: {rows:[{col:val}]} 직접 전달 또는 {excel_path[, sheet]} (첫 행=헤더=placeholder 키).
// 단계의 {{col}} placeholder가 행값으로 치환되며, 행별 성공/실패 + 전체 성공률을 집계한다.
// dry_run=true면 실행 없이 첫 행 확장 결과만 미리보기.
func handleAutomationBatch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Steps        []AutoStep          `json:"steps"`
		WorkflowName string              `json:"workflow_name"`
		Rows         []map[string]string `json:"rows"`
		ExcelPath    string              `json:"excel_path"`
		Sheet        string              `json:"sheet"`
		StopOnError  bool                `json:"stop_on_error"`
		DryRun       bool                `json:"dry_run"`
	}
	tryDecodeBody(r, &req)

	// 1) 템플릿 단계 확정: steps 직접 전달 또는 저장된 워크플로 로드.
	steps := req.Steps
	if len(steps) == 0 && strings.TrimSpace(req.WorkflowName) != "" {
		wf, err := LoadAutoWorkflow(req.WorkflowName)
		if err != nil {
			writeJSON(w, 404, map[string]any{"success": false, "message": "워크플로 없음: " + req.WorkflowName})
			return
		}
		steps = wf.Steps
	}
	if len(steps) == 0 {
		writeJSON(w, 400, map[string]any{"success": false, "message": "steps 또는 workflow_name 필요"})
		return
	}

	// 2) 데이터셋 확정: rows 직접 전달 또는 엑셀 파일(첫 행=헤더=치환 키)에서 로드.
	rows := req.Rows
	if len(rows) == 0 && strings.TrimSpace(req.ExcelPath) != "" {
		parsed, err := excelRowsToMaps(req.ExcelPath, req.Sheet)
		if err != nil {
			writeJSON(w, 400, map[string]any{"success": false, "message": "엑셀 읽기 실패: " + err.Error()})
			return
		}
		rows = parsed
	}
	if len(rows) == 0 {
		writeJSON(w, 400, map[string]any{"success": false, "message": "rows 또는 excel_path 필요"})
		return
	}

	// 3) dry-run: 실행 없이 첫 행 확장 결과 미리보기 (파괴적 동작/치환 사전 확인).
	if req.DryRun {
		json200(w, map[string]any{
			"success": true,
			"dry_run": true,
			"total":   len(rows),
			"columns": rowKeys(rows[0]),
			"preview": expandSteps(steps, rows[0]),
		})
		return
	}

	// 4) 가용성 게이트 — 미가용 시 501 (무단 실행 금지).
	a := GetAutomator()
	if !a.Available() {
		writeJSON(w, 501, map[string]any{
			"success":   false,
			"code":      "automation_unavailable",
			"available": false,
			"platform":  automationPlatform(),
			"message":   automationStatusMessage(false),
		})
		return
	}

	// 5) 배치 실행 + 성공률 집계.
	res := BatchRun(a, steps, rows, req.StopOnError)
	code := http.StatusOK
	if res.Succeeded < res.Total {
		code = http.StatusMultiStatus // 207 — 일부/전체 행 실패
	}
	writeJSON(w, code, map[string]any{"success": res.Succeeded == res.Total, "result": res})
}

// excelRowsToMaps — 엑셀 파일을 [{header: cell}] 배열로 변환 (첫 행 = 헤더 = placeholder 키).
// 빈 헤더 열은 건너뛰고, 누락 셀은 빈 문자열로 채운다.
func excelRowsToMaps(path, sheet string) ([]map[string]string, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	target := strings.TrimSpace(sheet)
	if target == "" {
		sheets := f.GetSheetList()
		if len(sheets) == 0 {
			return nil, errors.New("시트가 없습니다")
		}
		target = sheets[0]
	}
	rows, err := f.GetRows(target)
	if err != nil {
		return nil, err
	}
	if len(rows) < 2 {
		return nil, errors.New("데이터 행이 없습니다 (첫 행=헤더 + 데이터 1행 이상 필요)")
	}
	headers := rows[0]
	out := make([]map[string]string, 0, len(rows)-1)
	for _, row := range rows[1:] {
		m := make(map[string]string, len(headers))
		for i, h := range headers {
			h = strings.TrimSpace(h)
			if h == "" {
				continue // 빈 헤더 열 무시
			}
			if i < len(row) {
				m[h] = row[i]
			} else {
				m[h] = "" // 짧은 행은 빈 문자열로 보정
			}
		}
		out = append(out, m)
	}
	return out, nil
}

// rowKeys — 행의 치환 가능 컬럼 키 목록 (dry-run 미리보기에서 노출, 정렬해 안정적 출력).
func rowKeys(row map[string]string) []string {
	keys := make([]string, 0, len(row))
	for k := range row {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ── 녹화기(Recorder) 프록시 ────────────────────────────────────
// 사용자 클릭/입력을 Python UIA 후킹으로 AutoStep으로 캡처한다.
// "한 번 시연하면 알아서 반복"의 '시연(녹화)' 부분 — 비개발자도 자동화를 만들 수 있게 한다.

// POST /api/automation/record/start — 녹화 시작.
func handleAutomationRecordStart(w http.ResponseWriter, _ *http.Request) {
	res, err := callPython("POST", "/desktop/uia/record/start", nil)
	if err != nil {
		writeJSON(w, 501, map[string]any{
			"success": false, "code": "automation_unavailable",
			"message": "녹화 엔진 미가용 (데스크탑 자동화는 Windows에서 사용 가능)",
			"detail":  err.Error(),
		})
		return
	}
	json200(w, res)
}

// GET /api/automation/record/status — 녹화 상태/캡처 단계 수 (라이브 UI 폴링).
func handleAutomationRecordStatus(w http.ResponseWriter, _ *http.Request) {
	res, err := callPython("GET", "/desktop/uia/record/status", nil)
	if err != nil {
		json200(w, map[string]any{"success": true, "recording": false, "count": 0})
		return
	}
	json200(w, res)
}

// POST /api/automation/record/stop — 녹화 종료. {name} 주어지면 캡처 결과를 워크플로로 저장.
func handleAutomationRecordStop(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	tryDecodeBody(r, &req)

	res, err := callPython("POST", "/desktop/uia/record/stop", nil)
	if err != nil {
		writeJSON(w, 501, map[string]any{
			"success": false, "code": "automation_unavailable",
			"message": "녹화 엔진 미가용", "detail": err.Error(),
		})
		return
	}

	// name이 주어지면 캡처된 steps를 워크플로로 저장 → 녹화→저장→재생 루프 완성.
	if strings.TrimSpace(req.Name) != "" {
		steps := decodeStepsFromPython(res["steps"])
		if len(steps) > 0 {
			wf := NewAutoWorkflow(req.Name, steps)
			if path, serr := wf.Save(); serr == nil {
				res["saved"] = true
				res["workflow"] = req.Name
				res["path"] = path
			} else {
				res["saved"] = false
				res["save_error"] = serr.Error()
			}
		}
	}
	json200(w, res)
}

// decodeStepsFromPython — Python이 돌려준 steps(JSON)를 []AutoStep으로 변환 (재마샬링).
func decodeStepsFromPython(v any) []AutoStep {
	if v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var steps []AutoStep
	if json.Unmarshal(b, &steps) != nil {
		return nil
	}
	return steps
}

// registerAutomationRoutes — main.go / main_stub.go 양쪽에서 호출 (라우트 중복 정의 방지).
func registerAutomationRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/automation/status", handleAutomationStatus)
	mux.HandleFunc("GET /api/automation/workflows", handleAutomationWorkflows)
	mux.HandleFunc("POST /api/automation/workflows", handleAutomationWorkflows)
	mux.HandleFunc("GET /api/automation/workflows/{name}", handleAutomationWorkflowGet)
	mux.HandleFunc("POST /api/automation/workflows/{name}/replay", handleAutomationReplay)
	mux.HandleFunc("POST /api/automation/run", handleAutomationRun)
	mux.HandleFunc("POST /api/automation/batch", handleAutomationBatch)
	mux.HandleFunc("POST /api/automation/record/start", handleAutomationRecordStart)
	mux.HandleFunc("GET /api/automation/record/status", handleAutomationRecordStatus)
	mux.HandleFunc("POST /api/automation/record/stop", handleAutomationRecordStop)
}
