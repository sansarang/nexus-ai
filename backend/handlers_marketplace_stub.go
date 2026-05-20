//go:build !windows

package main

import "net/http"

func handleMarketplaceList(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"presets": []any{}, "total": 0})
}

func handleMarketplaceDetail(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 404, map[string]any{"error": "not available on this platform"})
}

func handleMarketplacePublish(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"ok": false, "error": "not available on this platform"})
}

func handleMarketplacePurchase(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"ok": false, "error": "not available on this platform"})
}

func handleMarketplacePurchaseConfirm(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"ok": false, "error": "not available on this platform"})
}

func handleMarketplaceMyPresets(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"presets": []any{}})
}

func handleMarketplacePurchased(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"presets": []any{}})
}

func handleMarketplaceDelete(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 404, map[string]any{"error": "not available on this platform"})
}
