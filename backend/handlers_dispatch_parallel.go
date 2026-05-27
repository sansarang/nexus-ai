//go:build windows

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ── 병렬 쿼리 디스패처 ──────────────────────────────────────────────
// POST /api/dispatch/parallel
// Body: { "queries": ["질문1", "질문2", ...], "task_id": "optional" }
// Response: SSE stream — 각 goroutine 완료 시 결과 즉시 전송

type parallelQueryReq struct {
	Queries []string `json:"queries"`
	TaskID  string   `json:"task_id"`
}

type parallelQueryResult struct {
	Index   int    `json:"index"`
	Query   string `json:"query"`
	Answer  string `json:"answer"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
	ElapsedMs int64  `json:"elapsed_ms"`
}

// handleDispatchParallel: 여러 쿼리를 동시에 goroutine으로 처리하고 SSE로 스트리밍
func handleDispatchParallel(w http.ResponseWriter, r *http.Request) {
	var req parallelQueryReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Queries) == 0 {
		writeJSON(w, 400, map[string]any{"error": "queries[] required"})
		return
	}
	if len(req.Queries) > 10 {
		req.Queries = req.Queries[:10] // 최대 10개
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	taskID := req.TaskID
	if taskID == "" {
		taskID = fmt.Sprintf("par-%d", time.Now().UnixMilli())
	}

	// 시작 알림
	startData, _ := json.Marshal(map[string]any{
		"type":    "start",
		"task_id": taskID,
		"total":   len(req.Queries),
	})
	fmt.Fprintf(w, "data: %s\n\n", startData)
	flusher.Flush()

	// 결과를 main goroutine으로 전달하는 채널
	resultCh := make(chan parallelQueryResult, len(req.Queries))

	jwt := getJWT()
	var wg sync.WaitGroup
	for i, q := range req.Queries {
		wg.Add(1)
		go func(idx int, query string) {
			defer wg.Done()
			start := time.Now()
			answer, err := dispatchSingleQuery(r.Context(), query, jwt)
			elapsed := time.Since(start).Milliseconds()
			result := parallelQueryResult{
				Index:     idx,
				Query:     query,
				ElapsedMs: elapsed,
			}
			if err != nil {
				result.Success = false
				result.Error = err.Error()
				result.Answer = fmt.Sprintf("오류: %s", err.Error())
			} else {
				result.Success = true
				result.Answer = answer
			}
			resultCh <- result
		}(i, q)
	}

	// 결과 수신 goroutine — 채널이 닫힐 때까지 SSE로 전송
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	received := 0
	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case res, more := <-resultCh:
			if !more {
				// 모두 완료
				doneData, _ := json.Marshal(map[string]any{
					"type":    "done",
					"task_id": taskID,
					"total":   len(req.Queries),
				})
				fmt.Fprintf(w, "data: %s\n\n", doneData)
				flusher.Flush()
				return
			}
			received++
			data, _ := json.Marshal(map[string]any{
				"type":       "result",
				"task_id":    taskID,
				"index":      res.Index,
				"query":      res.Query,
				"answer":     res.Answer,
				"success":    res.Success,
				"error":      res.Error,
				"elapsed_ms": res.ElapsedMs,
				"received":   received,
				"total":      len(req.Queries),
			})
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

// dispatchSingleQuery: 단일 쿼리를 LLM/Edge Function 을 통해 처리
func dispatchSingleQuery(ctx interface{ Done() <-chan struct{} }, query, jwt string) (string, error) {
	if jwt == "" {
		return "", fmt.Errorf("인증 토큰이 없습니다")
	}

	// Edge Function (ai-proxy) 호출
	body, _ := json.Marshal(map[string]any{
		"action": "chat",
		"payload": map[string]any{
			"message": query,
			"history": []any{},
			"lang":    "ko",
		},
	})

	req, err := http.NewRequest("POST", edgeFunctionURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("apikey", supabaseAnonKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var pr proxyResp
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return "", fmt.Errorf("응답 파싱 오류")
	}
	if resp.StatusCode != 200 || !pr.Success {
		if pr.Error != "" {
			return "", fmt.Errorf("%s", pr.Error)
		}
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	// result 에서 텍스트 추출
	if pr.Result != nil {
		if text, ok := pr.Result["text"].(string); ok && text != "" {
			return text, nil
		}
		if msg, ok := pr.Result["message"].(string); ok && msg != "" {
			return msg, nil
		}
		// JSON 직렬화 폴백
		b, _ := json.Marshal(pr.Result)
		return strings.TrimSpace(string(b)), nil
	}
	return "결과 없음", nil
}
