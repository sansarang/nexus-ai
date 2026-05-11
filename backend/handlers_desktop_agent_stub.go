//go:build !windows

package main

import (
	"net/http"
	"sync"
)

var approvalMu sync.Mutex
var pendingApprovals = make(map[string]chan bool)

func handleDesktopAgentRun(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": false, "message": "Windows 전용 기능입니다"})
}
func handleDesktopClick(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": false, "message": "Windows 전용 기능입니다"})
}
func handleDesktopType(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": false, "message": "Windows 전용 기능입니다"})
}
func handleDesktopKey(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": false, "message": "Windows 전용 기능입니다"})
}
func handleDesktopScroll(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": false, "message": "Windows 전용 기능입니다"})
}
func handleDesktopDrag(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": false, "message": "Windows 전용 기능입니다"})
}
func handleDesktopScreenshot(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": false, "message": "Windows 전용 기능입니다"})
}
func handleDesktopStatus(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": false, "message": "Windows 전용 기능입니다"})
}
func handleDesktopApprove(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": false, "message": "Windows 전용 기능입니다"})
}
