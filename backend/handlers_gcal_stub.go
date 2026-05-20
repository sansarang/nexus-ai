//go:build !windows

package main

import "net/http"

func handleGCalAuth(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]string{"url": ""})
}

func handleGCalCallback(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "windows only", http.StatusNotImplemented)
}

func handleGCalStatus(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"connected": false})
}

func handleGCalDisconnect(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"ok": true})
}

func gcalListEvents(timeMin, timeMax string) ([]map[string]any, error) {
	return nil, nil
}

func gcalAddEvent(subject, start, end, location string) error {
	return nil
}
