//go:build !windows

package main

import "net/http"

func handleDocUpload(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"success": false, "message": "Windows 전용 기능입니다"})
}

func handleDocAIEdit(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"success": false, "message": "Windows 전용 기능입니다"})
}

func handleReadExcel(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"success": false, "message": "Windows 전용 기능입니다"})
}
