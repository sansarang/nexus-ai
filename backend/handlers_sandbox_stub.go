//go:build !windows

package main

import "net/http"

func loadOllamaConfig()                                               {}
func handleAuditLog(w http.ResponseWriter, r *http.Request)           { json200(w, map[string]any{"entries": []any{}}) }
func handleCheckPath(w http.ResponseWriter, r *http.Request)          { json200(w, map[string]any{"safe": true}) }
func handleOllamaConfig(w http.ResponseWriter, r *http.Request)       { json200(w, map[string]any{"enabled": false}) }
func handleOllamaTest(w http.ResponseWriter, r *http.Request)         { json200(w, map[string]any{"success": false}) }
func handleOllamaModels(w http.ResponseWriter, r *http.Request)       { json200(w, map[string]any{"success": false}) }
