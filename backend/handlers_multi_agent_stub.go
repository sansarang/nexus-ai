//go:build !windows

package main

import "net/http"

func handleMultiAgentRun(w http.ResponseWriter, r *http.Request)   { proxyToPython(w, r, "/multi-agent/run") }
func handleAgentList(w http.ResponseWriter, r *http.Request)        { proxyToPythonGET(w, r, "/multi-agent/agents") }
func handleMultiAgentPlan(w http.ResponseWriter, r *http.Request)   { proxyToPython(w, r, "/multi-agent/plan") }
func handleMultiAgentRunV2(w http.ResponseWriter, r *http.Request)  { proxyToPython(w, r, "/multi-agent/run") }
func handleMultiAgentStream(w http.ResponseWriter, r *http.Request) { handleMultiAgentStreamWithPython(w, r) }
