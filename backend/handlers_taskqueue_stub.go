//go:build !windows

package main

import "net/http"

func initTaskQueue() {}
func handleTaskStream(w http.ResponseWriter, r *http.Request) { proxyToPythonGET(w, r, "/tasks/list") }
func handleTaskList(w http.ResponseWriter, r *http.Request)   { proxyToPythonGET(w, r, "/tasks/list") }
func handleTaskCancel(w http.ResponseWriter, r *http.Request) { proxyToPython(w, r, "/tasks/cancel") }
