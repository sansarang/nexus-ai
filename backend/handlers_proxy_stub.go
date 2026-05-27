//go:build !windows

package main

import (
	"fmt"
	"net/http"
)

func jwtMiddleware(next http.Handler) http.Handler { return next }

func requireAuth(w http.ResponseWriter, r *http.Request) bool { return true }

func getJWT() string { return "" }

func callGroqViaProxy(msgs []groqMsg, maxTokens int, jsonMode bool) (string, error) {
	return "", fmt.Errorf("proxy not available on non-Windows")
}

func callTavilyViaProxy(query string, maxResults int) (tavilyResult, bool) {
	return tavilyResult{}, false
}

func callTavilyDomainViaProxy(query string, maxResults int, domain string) (tavilyResult, bool) {
	return tavilyResult{}, false
}
