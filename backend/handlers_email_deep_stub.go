//go:build !windows

package main

import "net/http"

func handleEmailClassify(w http.ResponseWriter, r *http.Request)      { json200(w, map[string]any{"success": false}) }
func handleEmailDraftReply(w http.ResponseWriter, r *http.Request)    { json200(w, map[string]any{"success": false}) }
func handleEmailExtractEvents(w http.ResponseWriter, r *http.Request) { json200(w, map[string]any{"success": false}) }
func handleCalendarFindSlot(w http.ResponseWriter, r *http.Request)   { json200(w, map[string]any{"success": false}) }
func handleCalendarSmartAdd(w http.ResponseWriter, r *http.Request)   { json200(w, map[string]any{"success": false}) }
