//go:build !windows

package main

import "net/http"

func handlePaddleWebhook(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
