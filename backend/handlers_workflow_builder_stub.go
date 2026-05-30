//go:build !windows

package main

import "net/http"

func startWorkflowScheduler() {}

func handleWorkflowList(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"workflows": []any{}, "total": 0, "note": "워크플로우는 Windows 환경에서 Python sidecar 실행 시 동작합니다."})
}
func handleWorkflowSave(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": true, "message": "워크플로우 저장됨 (Windows sidecar 필요)"})
}
func handleWorkflowDelete(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": true, "message": "삭제됨"})
}
func handleWorkflowRunNow(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": false, "message": "워크플로우 실행은 Windows 환경에서만 가능합니다."})
}
func handleWorkflowFromText(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": false, "message": "워크플로우 생성은 Windows 환경에서만 가능합니다."})
}
func handleWorkflowTemplates(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"templates": []any{}, "note": "Windows sidecar 필요"})
}
