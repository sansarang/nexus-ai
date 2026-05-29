//go:build !windows

package main

import "net/http"

func startWorkflowScheduler()                                         {}
func handleWorkflowList(w http.ResponseWriter, r *http.Request)       { proxyToPythonGET(w, r, "/workflow/list") }
func handleWorkflowSave(w http.ResponseWriter, r *http.Request)       { proxyToPython(w, r, "/workflow/save") }
func handleWorkflowDelete(w http.ResponseWriter, r *http.Request)     { proxyToPythonDELETE(w, r, "/workflow/delete") }
func handleWorkflowRunNow(w http.ResponseWriter, r *http.Request)     { proxyToPython(w, r, "/workflow/run-now") }
func handleWorkflowFromText(w http.ResponseWriter, r *http.Request)   { proxyToPython(w, r, "/workflow/from-text") }
func handleWorkflowTemplates(w http.ResponseWriter, r *http.Request)  { proxyToPythonGET(w, r, "/workflow/templates") }
