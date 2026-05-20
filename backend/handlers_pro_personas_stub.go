//go:build !windows

package main

import "net/http"

func handleStockAnalysis(w http.ResponseWriter, r *http.Request)  { json200(w, map[string]any{"ok": false, "message": "windows only"}) }
func handleMedicalSearch(w http.ResponseWriter, r *http.Request)  { json200(w, map[string]any{"ok": false, "message": "windows only"}) }
func handleContractReview(w http.ResponseWriter, r *http.Request) { json200(w, map[string]any{"ok": false, "message": "windows only"}) }
func handleContentScript(w http.ResponseWriter, r *http.Request)  { json200(w, map[string]any{"ok": false, "message": "windows only"}) }
func handleLegalSearch(w http.ResponseWriter, r *http.Request)    { json200(w, map[string]any{"ok": false, "message": "windows only"}) }
