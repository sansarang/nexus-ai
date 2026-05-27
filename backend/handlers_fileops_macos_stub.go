//go:build !windows

package main

import "net/http"

func handleFileOrganize(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"ok": false, "error": "Windows only"})
}

func handleFileDuplicates(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"ok": false, "error": "Windows only"})
}

func handleFileLarge(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"ok": false, "error": "Windows only"})
}
