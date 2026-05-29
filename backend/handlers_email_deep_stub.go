//go:build !windows

package main

import "net/http"

func handleEmailClassify(w http.ResponseWriter, r *http.Request)      { proxyToPython(w, r, "/email/classify") }
func handleEmailDraftReply(w http.ResponseWriter, r *http.Request)    { proxyToPython(w, r, "/email/draft-reply") }
func handleEmailExtractEvents(w http.ResponseWriter, r *http.Request) { proxyToPython(w, r, "/email/extract-events") }
func handleCalendarFindSlot(w http.ResponseWriter, r *http.Request)   { proxyToPython(w, r, "/calendar/find-slot") }
func handleCalendarSmartAdd(w http.ResponseWriter, r *http.Request)   { proxyToPython(w, r, "/calendar/smart-add") }
