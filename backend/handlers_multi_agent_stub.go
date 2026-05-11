//go:build !windows

package main

import "net/http"

func handleMultiAgentRun(w http.ResponseWriter, r *http.Request)  { json200(w, map[string]any{"success": false, "message": "Windows 전용"}) }
func handleAgentList(w http.ResponseWriter, r *http.Request)       { json200(w, map[string]any{"agents": []any{}}) }
func handleMultiAgentPlan(w http.ResponseWriter, r *http.Request)  { json200(w, map[string]any{"success": false, "message": "Windows 전용"}) }
