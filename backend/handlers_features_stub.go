//go:build !windows

package main

import "net/http"

func isFeatureEnabled(_ string) bool            { return false }
func featureNotImplemented(w http.ResponseWriter, feature string) {
	writeJSON(w, 501, map[string]any{"success": false, "code": "feature_not_implemented", "feature": feature})
}
func handleFeatureList(w http.ResponseWriter, _ *http.Request) {
	json200(w, map[string]any{"features": map[string]any{}, "total": 0})
}
