//go:build !windows

package main

import "net/http"

func handleEnterpriseListKeys(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"ok": false, "error": "windows only"})
}

func handleEnterpriseCreateKey(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"ok": false, "error": "windows only"})
}

func handleEnterpriseRevokeKey(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"ok": false, "error": "windows only"})
}

func handleEnterpriseKeyUsage(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"ok": false, "error": "windows only"})
}

func handleEnterprisePlans(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"ok": false, "error": "windows only"})
}

func handleV1Chat(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"ok": false, "error": "windows only"})
}

func handleV1Search(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"ok": false, "error": "windows only"})
}

func handleV1Stock(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"ok": false, "error": "windows only"})
}

func handleV1Legal(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"ok": false, "error": "windows only"})
}

func handleV1Medical(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"ok": false, "error": "windows only"})
}
