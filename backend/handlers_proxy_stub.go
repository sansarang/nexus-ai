//go:build !windows

package main

import (
	"fmt"
	"net/http"
)

func jwtMiddleware(next http.Handler) http.Handler { return next }

func callGroqViaProxy(msgs []groqMsg, maxTokens int, jsonMode bool) (string, error) {
	return "", fmt.Errorf("proxy not available on non-Windows")
}

func callTavilyViaProxy(query string, maxResults int) (tavilyResult, bool) {
	return tavilyResult{}, false
}
