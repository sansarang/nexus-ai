package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

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
	json.NewDecoder(r.Body).Decode(&req)
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
