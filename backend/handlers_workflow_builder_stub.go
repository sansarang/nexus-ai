//go:build !windows

package main

import "net/http"

func startWorkflowScheduler()                                        {}
func handleWorkflowList(w http.ResponseWriter, r *http.Request)      { json200(w, map[string]any{"workflows": []any{}}) }
func handleWorkflowSave(w http.ResponseWriter, r *http.Request)      { json200(w, map[string]any{"success": false}) }
func handleWorkflowDelete(w http.ResponseWriter, r *http.Request)    { json200(w, map[string]any{"success": false}) }
func handleWorkflowRunNow(w http.ResponseWriter, r *http.Request)    { json200(w, map[string]any{"success": false}) }
func handleWorkflowFromText(w http.ResponseWriter, r *http.Request)  { json200(w, map[string]any{"success": false}) }
func handleWorkflowTemplates(w http.ResponseWriter, r *http.Request) { json200(w, map[string]any{"templates": []any{}}) }
