//go:build !windows

package main

import (
	"net/http"
	"sync"
)

var approvalMu sync.Mutex
var pendingApprovals = make(map[string]chan bool)

func handleDesktopAgentRun(w http.ResponseWriter, r *http.Request)    { proxyToPython(w, r, "/desktop/agent/run") }
func handleDesktopClick(w http.ResponseWriter, r *http.Request)        { proxyToPython(w, r, "/desktop/click") }
func handleDesktopType(w http.ResponseWriter, r *http.Request)         { proxyToPython(w, r, "/desktop/type") }
func handleDesktopKey(w http.ResponseWriter, r *http.Request)          { proxyToPython(w, r, "/desktop/key") }
func handleDesktopScroll(w http.ResponseWriter, r *http.Request)       { proxyToPython(w, r, "/desktop/scroll") }
func handleDesktopDrag(w http.ResponseWriter, r *http.Request)         { proxyToPython(w, r, "/desktop/drag") }
func handleDesktopScreenshot(w http.ResponseWriter, r *http.Request)   { proxyToPythonGET(w, r, "/desktop/screenshot") }
func handleDesktopStatus(w http.ResponseWriter, r *http.Request)       { proxyToPythonGET(w, r, "/desktop/status") }
func handleDesktopApprove(w http.ResponseWriter, r *http.Request)      { proxyToPython(w, r, "/desktop/approve") }
func handleDesktopAgentCancel(w http.ResponseWriter, r *http.Request)  { proxyToPython(w, r, "/desktop/agent/cancel") }
