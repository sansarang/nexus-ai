//go:build !windows

package main

import "net/http"

func handleIMAPAccountList(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"accounts": []any{}, "count": 0})
}
func handleIMAPAccountAdd(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": false, "message": "Windows 전용 기능입니다"})
}
func handleIMAPAccountDelete(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": false, "message": "Windows 전용 기능입니다"})
}
func handleIMAPInbox(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": false, "emails": []any{}, "message": "Windows 전용 기능입니다"})
}
func handleIMAPSend(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": false, "message": "Windows 전용 기능입니다"})
}
func handleIMAPReplySuggestions(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": false, "suggestions": []string{}, "message": "Windows 전용 기능입니다"})
}
func handleIMAPClassify(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"success": false, "message": "Windows 전용 기능입니다"})
}
