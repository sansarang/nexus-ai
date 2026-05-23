// http_utils.go — 플랫폼 무관 HTTP 유틸리티 (빌드 태그 없음)
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

const maxRequestBody = 32 * 1024 * 1024 // 32MB 요청 상한

// decodeJSON: r.Body를 v에 JSON 디코딩. 실패 시 400 응답 후 false 반환.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeJSON(w, 400, map[string]any{"success": false, "message": "요청 형식 오류: " + err.Error()})
		return false
	}
	return true
}

func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[Nexus] panic recovered: %v", err)
				writeJSON(w, 500, map[string]any{"success": false, "message": fmt.Sprintf("서버 오류: %v", err)})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
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

// getLang: 요청에서 언어 코드 추출 (header X-Lang 또는 query lang, 기본 "ko")
func getLang(r *http.Request) string {
	if l := r.Header.Get("X-Lang"); l == "en" {
		return "en"
	}
	if l := r.URL.Query().Get("lang"); l == "en" {
		return "en"
	}
	return "ko"
}

// msgT: 언어에 따라 ko 또는 en 반환
func msgT(ko, en, lang string) string {
	if lang == "en" {
		return en
	}
	return ko
}
