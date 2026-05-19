// http_utils.go — 플랫폼 무관 HTTP 유틸리티 (빌드 태그 없음)
package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"
)

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func json200(w http.ResponseWriter, v any) {
	writeJSON(w, http.StatusOK, v)
}

const maxResponseBody = 8 * 1024 * 1024 // 8MB 응답 상한

// httpGet: 내부 HTTP GET 유틸
func httpGet(url string) ([]byte, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
}

// httpPost: 내부 HTTP POST 유틸
func httpPost(u string, body []byte) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(u, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
}

// urlEncode: 쿼리 파라미터 URL 인코딩
func urlEncode(s string) string {
	return url.QueryEscape(s)
}
