//go:build !windows

package main

import "net/http"

func startBriefingScheduler()                                       {}
func handleBriefingNow(w http.ResponseWriter, r *http.Request)      { json200(w, map[string]any{"success": false, "message": "Windows 전용"}) }
func handleBriefingConfig(w http.ResponseWriter, r *http.Request)   { json200(w, map[string]any{"success": false, "message": "Windows 전용"}) }
