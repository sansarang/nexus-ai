// handlers_automation.go — 데스크탑 자동화 엔진 HTTP API (cross-platform)
//
// 안전 원칙:
//   - 실제 실행은 GetAutomator().Available() 게이트를 통과해야만 한다 (미가용 시 501, 무단 실행 금지).
//   - dry_run=true면 실행 없이 단계만 미리보기(파괴적 동작 사전 확인).
//   - 실제 UIA 동작 검증은 Windows 머신에서 (windowsAutomator 참고).
package main

import (
	"net/http"
	"runtime"
	"strings"
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

// registerAutomationRoutes — main.go / main_stub.go 양쪽에서 호출 (라우트 중복 정의 방지).
func registerAutomationRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/automation/status", handleAutomationStatus)
	mux.HandleFunc("GET /api/automation/workflows", handleAutomationWorkflows)
	mux.HandleFunc("POST /api/automation/workflows", handleAutomationWorkflows)
	mux.HandleFunc("GET /api/automation/workflows/{name}", handleAutomationWorkflowGet)
	mux.HandleFunc("POST /api/automation/workflows/{name}/replay", handleAutomationReplay)
	mux.HandleFunc("POST /api/automation/run", handleAutomationRun)
}
