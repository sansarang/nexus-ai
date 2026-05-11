// nexus_test.go — Nexus 백엔드 통합 테스트
// 실행: GOOS=windows GOARCH=amd64 go test 는 크로스컴파일 테스트 불가
// → go test (macOS stub) 로 타입/구조 검증, 실제 동작은 Windows에서 검증

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ── 헬퍼 ──────────────────────────────────────────────────────

func newRequest(method, path, body string) *http.Request {
	var bodyReader *strings.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	} else {
		bodyReader = strings.NewReader("{}")
	}
	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func recordResponse(handler http.HandlerFunc, req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	handler(rr, req)
	return rr
}

// ── 타입 검증 ─────────────────────────────────────────────────

func TestDeepSearchResultType(t *testing.T) {
	r := DeepSearchResult{
		Name:    "test.txt",
		Path:    "C:/test.txt",
		Ext:     ".txt",
		SizeMB:  0.5,
		ModTime: "2026-01-01 10:00",
		Snippet: "...test content...",
		Score:   80,
	}
	if r.Score != 80 {
		t.Errorf("Score 필드 오류: got %d", r.Score)
	}
}

func TestGroqMsgType(t *testing.T) {
	msg := groqMsg{Role: "user", Content: "hello"}
	b, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("groqMsg 직렬화 실패: %v", err)
	}
	if !strings.Contains(string(b), `"role":"user"`) {
		t.Errorf("직렬화 결과 오류: %s", string(b))
	}
}

func TestMsgPartVisionType(t *testing.T) {
	part := msgPart{
		Type: "image_url",
		ImageURL: &imageURL{URL: "data:image/png;base64,abc123"},
	}
	b, err := json.Marshal(part)
	if err != nil {
		t.Fatalf("msgPart 직렬화 실패: %v", err)
	}
	if !strings.Contains(string(b), "image_url") {
		t.Errorf("직렬화 결과 오류: %s", string(b))
	}
}

// ── 상수 검증 ─────────────────────────────────────────────────

func TestLLMConstants(t *testing.T) {
	if groqChatModel == "" {
		t.Error("groqChatModel 비어 있음")
	}
	if groqVisionModel == "" {
		t.Error("groqVisionModel 비어 있음")
	}
	if !strings.HasPrefix(groqAPIBase, "https://") {
		t.Errorf("groqAPIBase 잘못된 URL: %s", groqAPIBase)
	}
	if !strings.HasPrefix(claudeAPIBase, "https://") {
		t.Errorf("claudeAPIBase 잘못된 URL: %s", claudeAPIBase)
	}
}

// ── stub 핸들러 응답 코드 검증 ────────────────────────────────

func TestStubHandlersReturnOK(t *testing.T) {
	handlers := []struct {
		name    string
		method  string
		handler http.HandlerFunc
	}{
		{"LLMConfig GET", "GET", handleLLMConfig},
		{"LLMConfig POST", "POST", handleLLMConfig},
		{"LLMChat", "POST", handleLLMChat},
		{"LLMVision", "POST", handleLLMVision},
		{"LLMDocSummary", "POST", handleLLMDocSummary},
		{"LLMDocCompare", "POST", handleLLMDocCompare},
		{"LLMDeepSearch", "POST", handleLLMDeepSearch},
		{"BrowserStatus", "GET", handleBrowserStatus},
		{"BrowserNavigate", "POST", handleBrowserNavigate},
		{"BrowserExtract", "POST", handleBrowserExtract},
		{"BrowserClick", "POST", handleBrowserClick},
		{"BrowserFill", "POST", handleBrowserFill},
		{"BrowserScreenshot", "POST", handleBrowserScreenshot},
		{"BrowserAgent", "POST", handleBrowserAgent},
		{"BrowserClose", "POST", handleBrowserClose},
	}

	for _, h := range handlers {
		t.Run(h.name, func(t *testing.T) {
			req := newRequest(h.method, "/test", "{}")
			rr := recordResponse(h.handler, req)
			// stub은 빈 응답 (200) 또는 실제 핸들러 응답
			if rr.Code >= 500 {
				t.Errorf("%s: 서버 오류 코드 %d", h.name, rr.Code)
			}
		})
	}
}

// ── CORS 미들웨어 검증 ────────────────────────────────────────

func TestCORSMiddleware(t *testing.T) {
	handler := cors(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	// OPTIONS preflight
	req := httptest.NewRequest("OPTIONS", "/api/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Errorf("OPTIONS 응답 코드: got %d, want 204", rr.Code)
	}
	if rr.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("CORS 헤더 누락")
	}

	// 일반 GET
	req2 := httptest.NewRequest("GET", "/api/health", nil)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)
	if rr2.Code != 200 {
		t.Errorf("GET 응답 코드: got %d, want 200", rr2.Code)
	}
}

// ── writeJSON 검증 ───────────────────────────────────────────

func TestWriteJSON(t *testing.T) {
	rr := httptest.NewRecorder()
	writeJSON(rr, 200, map[string]string{"status": "ok"})

	if rr.Code != 200 {
		t.Errorf("writeJSON 코드: got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type: got %s", ct)
	}
	var result map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("응답 파싱 실패: %v", err)
	}
	if result["status"] != "ok" {
		t.Errorf("응답 내용 오류: %v", result)
	}
}

// ── callGroq 입력 검증 (API 키 없음 → 오류) ──────────────────

func TestCallGroqNoKey(t *testing.T) {
	_, _, err := callGroq("", groqChatModel, []groqMsg{{Role: "user", Content: "test"}}, 100, false)
	if err == nil {
		t.Error("API 키 없을 때 오류 반환해야 함")
	}
	if !strings.Contains(err.Error(), "키") && !strings.Contains(err.Error(), "key") {
		t.Errorf("오류 메시지가 키 관련이 아님: %s", err.Error())
	}
}

// ── deepSearchFiles 빈 쿼리 처리 ─────────────────────────────

func TestDeepSearchFilesEmpty(t *testing.T) {
	results := deepSearchFiles("", "/nonexistent", 10)
	// 빈 쿼리 또는 존재하지 않는 폴더 → nil 또는 빈 슬라이스
	if len(results) > 0 {
		t.Errorf("존재하지 않는 폴더에서 결과 반환됨: %d개", len(results))
	}
}

// ── max2 함수 검증 ────────────────────────────────────────────

func TestMax2(t *testing.T) {
	tests := []struct{ a, b, want int }{
		{3, 5, 5},
		{10, 2, 10},
		{0, 0, 0},
		{-1, 1, 1},
	}
	for _, tt := range tests {
		if got := max2(tt.a, tt.b); got != tt.want {
			t.Errorf("max2(%d,%d) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

// ── LLM 설정 동시성 안전성 검증 ──────────────────────────────

func TestLLMConfigConcurrency(t *testing.T) {
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(n int) {
			llmMu.Lock()
			llmPerplexityKey = "test-key"
			llmMu.Unlock()

			llmMu.RLock()
			_ = llmPerplexityKey
			llmMu.RUnlock()
			done <- true
		}(i)
	}
	for i := 0; i < 10; i++ {
		<-done
	}
	// race detector로 검증 (-race 플래그)
}
