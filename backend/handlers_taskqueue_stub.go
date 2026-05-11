//go:build !windows

package main

import "net/http"

func initTaskQueue() {}
func handleTaskStream(w http.ResponseWriter, r *http.Request) { json200(w, map[string]any{"status": "stub"}) }
func handleTaskList(w http.ResponseWriter, r *http.Request)   { json200(w, map[string]any{"tasks": []any{}}) }
func handleTaskCancel(w http.ResponseWriter, r *http.Request) { json200(w, map[string]any{"success": false}) }
