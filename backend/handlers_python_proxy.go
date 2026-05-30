package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ══════════════════════════════════════════════════════════════
//  Python 사이드카 헬스 상태 관리
// ══════════════════════════════════════════════════════════════

var (
	pythonHealthy  atomic.Bool          // true = 연결 정상
	pythonHealthMu sync.Mutex
)

// waitForPython — 앱 시작 시 Python 준비될 때까지 최대 30초 retry (300ms 간격)
func waitForPython() {
	client := &http.Client{Timeout: 2 * time.Second}
	for i := 0; i < 100; i++ {
		resp, err := client.Get(pythonBase + "/health")
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			pythonHealthy.Store(true)
			log.Printf("[Python] sidecar ready (attempt %d)", i+1)
			go injectKeysToPython()
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(300 * time.Millisecond)
	}
	log.Printf("[Python] sidecar not ready after 30s — continuing without it")
}

// startPythonHealthLoop — 백그라운드에서 Python 상태를 주기적으로 체크 (30초 간격)
// 연결 끊어지면 exponential backoff (1s→2s→4s→8s→16s→30s)로 재연결 시도
func startPythonHealthLoop() {
	go func() {
		client := &http.Client{Timeout: 2 * time.Second}
		backoff := time.Second
		for {
			time.Sleep(30 * time.Second)
			resp, err := client.Get(pythonBase + "/health")
			if err == nil && resp.StatusCode == 200 {
				resp.Body.Close()
				if !pythonHealthy.Load() {
					log.Printf("[Python] sidecar reconnected")
					pythonHealthy.Store(true)
					go injectKeysToPython()
				}
				backoff = time.Second
				continue
			}
			if resp != nil {
				resp.Body.Close()
			}
			pythonHealthy.Store(false)
			log.Printf("[Python] sidecar unreachable — retry in %v", backoff)

			// exponential backoff 재시도
			for attempt := 0; attempt < 6; attempt++ {
				time.Sleep(backoff)
				r2, e2 := client.Get(pythonBase + "/health")
				if e2 == nil && r2.StatusCode == 200 {
					r2.Body.Close()
					pythonHealthy.Store(true)
					log.Printf("[Python] sidecar recovered (backoff attempt %d)", attempt+1)
					go injectKeysToPython()
					break
				}
				if r2 != nil {
					r2.Body.Close()
				}
				if backoff < 30*time.Second {
					backoff *= 2
				}
			}
		}
	}()
}

// handlePythonHealth — GET /api/python/health : 프론트엔드가 Python 상태를 폴링
func handlePythonHealth(w http.ResponseWriter, r *http.Request) {
	healthy := pythonHealthy.Load()
	status := "ok"
	if !healthy {
		status = "unavailable"
	}
	json200(w, map[string]any{"status": status, "healthy": healthy})
}

// ══════════════════════════════════════════════════════════════
//  Python 사이드카 프록시 헬퍼 (포트 17893) — 공통
// ══════════════════════════════════════════════════════════════

const pythonBase = "http://127.0.0.1:17893"

var pythonClient = &http.Client{Timeout: 60 * time.Second}

func proxyToPython(w http.ResponseWriter, r *http.Request, path string) {
	body, _ := io.ReadAll(io.LimitReader(r.Body, 8*1024*1024))
	req, err := http.NewRequestWithContext(r.Context(), r.Method,
		pythonBase+path, bytes.NewReader(body))
	if err != nil {
		writeJSON(w, 503, map[string]any{"success": false, "message": "Python 사이드카 요청 생성 실패: " + err.Error()})
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := pythonClient.Do(req)
	if err != nil {
		writeJSON(w, 503, map[string]any{"success": false, "message": "Python 사이드카 연결 실패 — 잠시 후 다시 시도해주세요."})
		return
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(io.LimitReader(resp.Body, 8*1024*1024))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(out)
}

func proxyToPythonGET(w http.ResponseWriter, r *http.Request, path string) {
	qs := r.URL.RawQuery
	fullPath := path
	if qs != "" {
		fullPath = path + "?" + qs
	}
	req, err := http.NewRequestWithContext(r.Context(), "GET", pythonBase+fullPath, nil)
	if err != nil {
		writeJSON(w, 503, map[string]any{"success": false, "message": "Python 사이드카 요청 생성 실패"})
		return
	}
	resp, err := pythonClient.Do(req)
	if err != nil {
		writeJSON(w, 503, map[string]any{"success": false, "message": "Python 사이드카 연결 실패"})
		return
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(io.LimitReader(resp.Body, 8*1024*1024))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(out)
}

func proxyToPythonDELETE(w http.ResponseWriter, r *http.Request, path string) {
	body, _ := io.ReadAll(io.LimitReader(r.Body, 1*1024*1024))
	qs := r.URL.RawQuery
	fullPath := path
	if qs != "" {
		fullPath = path + "?" + qs
	}
	req, err := http.NewRequestWithContext(r.Context(), "DELETE", pythonBase+fullPath, bytes.NewReader(body))
	if err != nil {
		writeJSON(w, 503, map[string]any{"success": false, "message": "Python 사이드카 요청 생성 실패"})
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := pythonClient.Do(req)
	if err != nil {
		writeJSON(w, 503, map[string]any{"success": false, "message": "Python 사이드카 연결 실패"})
		return
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(io.LimitReader(resp.Body, 1*1024*1024))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(out)
}

func callPython(method, path string, reqBody any) (map[string]any, error) {
	var bodyReader io.Reader
	if reqBody != nil {
		b, _ := json.Marshal(reqBody)
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, pythonBase+path, bodyReader)
	if err != nil {
		return nil, err
	}
	if bodyReader != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := pythonClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result map[string]any
	json.NewDecoder(io.LimitReader(resp.Body, 4*1024*1024)).Decode(&result)
	return result, nil
}

func isPythonReady() bool {
	resp, err := (&http.Client{Timeout: 2 * time.Second}).Get(pythonBase + "/health")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == 200
}

// injectKeysToPython: Python 사이드카에 API 키 주입 (Go가 llm_config에서 복호화한 키 전달)
func injectKeysToPython() {
	llmMu.RLock()
	groqKey   := llmGroqKey
	claudeKey := llmClaudeKey
	tavilyKey := llmTavilyKey
	llmMu.RUnlock()

	if groqKey == "" && claudeKey == "" && tavilyKey == "" {
		return
	}

	// Python이 뜰 때까지 최대 30초 대기
	for i := 0; i < 30; i++ {
		if isPythonReady() {
			break
		}
		time.Sleep(1 * time.Second)
	}

	callPython("POST", "/admin/keys", map[string]any{
		"groq_key":   groqKey,
		"claude_key": claudeKey,
		"tavily_key": tavilyKey,
	})
}

// ── YouTube 검색: Python yt-dlp 우선, Tavily fallback ──────────

func handleYouTubeSearchWithPython(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		Query    string `json:"query"`
		MaxItems int    `json:"max_items"`
	}
	tryDecodeBody(r, &req)
	if req.Query == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("query 필요", "query required", lang)})
		return
	}
	if req.MaxItems == 0 {
		req.MaxItems = 10
	}

	result, err := callPython("POST", "/youtube/search", map[string]any{
		"query": req.Query, "max_items": req.MaxItems,
	})
	if err == nil {
		if ok, _ := result["success"].(bool); ok {
			if items, _ := result["items"].([]any); len(items) > 0 {
				json200(w, result)
				return
			}
		}
	}

	llmMu.RLock()
	tKey := llmTavilyKey
	llmMu.RUnlock()
	if tKey != "" {
		tr, tavilyOk := tavilySearchDomain(tKey, req.Query, req.MaxItems, "youtube.com")
		if tavilyOk && len(tr.Items) > 0 {
			json200(w, map[string]any{
				"success": true, "source": "search_fallback",
				"items": tr.Items, "summary": tr.Summary, "count": len(tr.Items),
				"message": fmt.Sprintf("YouTube '%s' 검색 결과 %d개", req.Query, len(tr.Items)),
			})
			return
		}
	}

	json200(w, map[string]any{
		"success": false, "items": []any{}, "count": 0,
		"message": fmt.Sprintf(msgT("'%s' YouTube 검색 결과가 없어요.", "No YouTube results for '%s'.", lang), req.Query),
	})
}

// ── Multi-Agent: Python Groq 구현으로 전달 ──────────────────────

func handleMultiAgentRunV2WithPython(w http.ResponseWriter, r *http.Request) {
	proxyToPython(w, r, "/multi-agent/run")
}

func handleMultiAgentStreamWithPython(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/stream/")
	taskID := ""
	if len(parts) > 1 {
		taskID = strings.Trim(parts[1], "/")
	}
	proxyToPythonGET(w, r, "/multi-agent/stream/"+taskID)
}

func handleMultiAgentPlanWithPython(w http.ResponseWriter, r *http.Request) {
	proxyToPython(w, r, "/multi-agent/plan")
}

func handleAgentListWithPython(w http.ResponseWriter, r *http.Request) {
	proxyToPythonGET(w, r, "/multi-agent/agents")
}

// ── Workflow Builder: Python 구현으로 전달 ──────────────────────

func handleWorkflowListWithPython(w http.ResponseWriter, r *http.Request) {
	proxyToPythonGET(w, r, "/workflow/list")
}

func handleWorkflowSaveWithPython(w http.ResponseWriter, r *http.Request) {
	proxyToPython(w, r, "/workflow/save")
}

func handleWorkflowDeleteWithPython(w http.ResponseWriter, r *http.Request) {
	proxyToPythonDELETE(w, r, "/workflow/delete")
}

func handleWorkflowRunNowWithPython(w http.ResponseWriter, r *http.Request) {
	proxyToPython(w, r, "/workflow/run-now")
}

func handleWorkflowFromTextWithPython(w http.ResponseWriter, r *http.Request) {
	proxyToPython(w, r, "/workflow/from-text")
}

func handleWorkflowTemplatesWithPython(w http.ResponseWriter, r *http.Request) {
	proxyToPythonGET(w, r, "/workflow/templates")
}

// ── Desktop Agent: Python pyautogui 구현으로 전달 ───────────────

func handleDesktopClickWithPython(w http.ResponseWriter, r *http.Request) {
	proxyToPython(w, r, "/desktop/click")
}

func handleDesktopTypeWithPython(w http.ResponseWriter, r *http.Request) {
	proxyToPython(w, r, "/desktop/type")
}

func handleDesktopKeyWithPython(w http.ResponseWriter, r *http.Request) {
	proxyToPython(w, r, "/desktop/key")
}

func handleDesktopScrollWithPython(w http.ResponseWriter, r *http.Request) {
	proxyToPython(w, r, "/desktop/scroll")
}

func handleDesktopDragWithPython(w http.ResponseWriter, r *http.Request) {
	proxyToPython(w, r, "/desktop/drag")
}

func handleDesktopScreenshotWithPython(w http.ResponseWriter, r *http.Request) {
	proxyToPythonGET(w, r, "/desktop/screenshot")
}

func handleDesktopAgentRunWithPython(w http.ResponseWriter, r *http.Request) {
	proxyToPython(w, r, "/desktop/agent/run")
}

func handleDesktopAgentCancelWithPython(w http.ResponseWriter, r *http.Request) {
	proxyToPython(w, r, "/desktop/agent/cancel")
}

func handleDesktopStatusWithPython(w http.ResponseWriter, r *http.Request) {
	proxyToPythonGET(w, r, "/desktop/status")
}
