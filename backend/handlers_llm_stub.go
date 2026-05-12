//go:build !windows

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	llmMu            sync.RWMutex
	llmPerplexityKey string
	llmClaudeKey     string
	llmTavilyKey     string
)

type llmConfigFile struct {
	PerplexityKey string `json:"perplexity_key"`
	ClaudeKey     string `json:"claude_key"`
	TavilyKey     string `json:"tavily_key"`
}

func llmConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".nexus", "llm_config.json")
}

func loadLLMConfig() {
	data, err := os.ReadFile(llmConfigPath())
	if err != nil {
		return
	}
	// 하위 호환: 기존 groq_key → perplexity_key 마이그레이션
	var raw map[string]string
	if json.Unmarshal(data, &raw) == nil {
		llmMu.Lock()
		if v := raw["perplexity_key"]; v != "" {
			llmPerplexityKey = v
		} else if v := raw["groq_key"]; v != "" {
			llmPerplexityKey = v
		}
		if v := raw["claude_key"]; v != "" {
			llmClaudeKey = v
		}
		if v := raw["tavily_key"]; v != "" {
			llmTavilyKey = v
		}
		llmMu.Unlock()
	}
}

func saveLLMConfig() {
	llmMu.RLock()
	cfg := llmConfigFile{
		PerplexityKey: llmPerplexityKey,
		ClaudeKey:     llmClaudeKey,
		TavilyKey:     llmTavilyKey,
	}
	llmMu.RUnlock()
	data, _ := json.MarshalIndent(cfg, "", "  ")
	os.MkdirAll(filepath.Dir(llmConfigPath()), 0755)
	os.WriteFile(llmConfigPath(), data, 0600)
}

// callOpenAICompat: OpenAI 호환 엔드포인트 범용 호출 (Perplexity 전용)
func callOpenAICompat(apiKey, baseURL, model string, msgs []groqMsg, maxTokens int, jsonMode bool) (string, int, error) {
	if apiKey == "" {
		return "", 0, fmt.Errorf("Perplexity API 키가 설정되지 않았습니다")
	}
	type reqBody struct {
		Model       string    `json:"model"`
		Messages    []groqMsg `json:"messages"`
		Temperature float64   `json:"temperature"`
		MaxTokens   int       `json:"max_tokens"`
		RespFmt     *struct {
			Type string `json:"type"`
		} `json:"response_format,omitempty"`
	}
	rb := reqBody{Model: model, Messages: msgs, Temperature: 0.1, MaxTokens: maxTokens}
	// Perplexity sonar는 response_format을 지원하지 않음 — JSON은 system prompt로 강제
	body, _ := json.Marshal(rb)
	req, _ := http.NewRequest("POST", baseURL, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("연결 실패 (%s): %w", model, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var gr struct {
		Choices []struct {
			Message struct{ Content string `json:"content"` } `json:"message"`
		} `json:"choices"`
		Citations []string `json:"citations"`
		Error     *struct{ Message string `json:"message"` } `json:"error"`
	}
	if err := json.Unmarshal(raw, &gr); err != nil {
		return "", 0, fmt.Errorf("응답 파싱 실패: %w", err)
	}
	if gr.Error != nil {
		return "", 0, fmt.Errorf("[%s] %s", model, gr.Error.Message)
	}
	if len(gr.Choices) == 0 {
		return "", 0, fmt.Errorf("응답 없음 (%s)", model)
	}
	lastCitationsMu.Lock()
	lastCitations = gr.Citations
	lastCitationsMu.Unlock()
	return gr.Choices[0].Message.Content, maxTokens, nil
}

var (
	lastCitationsMu sync.Mutex
	lastCitations   []string
)

func callGroqWithCitations(apiKey, model string, msgs []groqMsg, maxTokens int) (string, []string, error) {
	llmMu.RLock()
	pKey := llmPerplexityKey
	llmMu.RUnlock()
	useKey := pKey
	if useKey == "" {
		useKey = apiKey
	}
	pModel := model
	switch model {
	case "llama-3.3-70b-versatile", "llama-3.1-70b-versatile":
		pModel = pplxChatModel
	case "llama-3.1-8b-instant", "llama-3.2-3b-preview":
		pModel = pplxFastModel
	}
	text, _, err := callOpenAICompat(useKey, pplxAPIBase, pModel, msgs, maxTokens, false)
	if err != nil {
		return "", nil, err
	}
	lastCitationsMu.Lock()
	cites := make([]string, len(lastCitations))
	copy(cites, lastCitations)
	lastCitationsMu.Unlock()
	return text, cites, nil
}

// callGroq: Perplexity API 호출 (groqChatModel/groqFastModel은 pplx 별칭)
func callGroq(apiKey, model string, msgs []groqMsg, maxTokens int, jsonMode bool) (string, int, error) {
	llmMu.RLock()
	pKey := llmPerplexityKey
	llmMu.RUnlock()
	useKey := pKey
	if useKey == "" {
		useKey = apiKey
	}
	// 구 Groq 모델명이 넘어와도 Perplexity 모델로 교정
	pModel := model
	switch model {
	case "llama-3.3-70b-versatile", "llama-3.1-70b-versatile":
		pModel = pplxChatModel
	case "llama-3.1-8b-instant", "llama-3.2-3b-preview":
		pModel = pplxFastModel
	}
	return callOpenAICompat(useKey, pplxAPIBase, pModel, msgs, maxTokens, jsonMode)
}

func callGroqWithFallback(msgs []groqMsg, maxTokens int, jsonMode bool) (string, string, error) {
	llmMu.RLock()
	pKey := llmPerplexityKey
	llmMu.RUnlock()
	if pKey == "" {
		return "", "", fmt.Errorf("Perplexity API 키가 설정되지 않았습니다")
	}
	text, _, err := callOpenAICompat(pKey, pplxAPIBase, pplxChatModel, msgs, maxTokens, jsonMode)
	if err != nil {
		return "", "", err
	}
	return text, "perplexity", nil
}

func callGroqVision(_, _, _, _ string) (string, error) {
	return "", fmt.Errorf("Vision 기능은 현재 지원되지 않습니다")
}

func callClaude(apiKey string, msgs []groqMsg, maxTokens int) (string, error) {
	return "", fmt.Errorf("Claude 미구현")
}

func deepSearchFiles(_, _ string, _ int) []DeepSearchResult { return nil }

func max2(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func handleLLMConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		llmMu.RLock()
		pSet := llmPerplexityKey != ""
		cSet := llmClaudeKey != ""
		tSet := llmTavilyKey != ""
		llmMu.RUnlock()
		json200(w, map[string]any{
			"perplexity_configured": pSet,
			"claude_configured":     cSet,
			"tavily_configured":     tSet,
			"models": map[string]string{
				"chat": pplxChatModel,
				"fast": pplxFastModel,
			},
			"provider": "perplexity",
		})
		return
	}
	var req struct {
		PerplexityKey string `json:"perplexity_key"`
		ApiKey        string `json:"apiKey"`
		ClaudeKey     string `json:"claude_key"`
		TavilyKey     string `json:"tavily_key"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	// 하위 호환: apiKey / groq_key 필드도 perplexity_key로 처리
	if req.PerplexityKey == "" && req.ApiKey != "" {
		req.PerplexityKey = req.ApiKey
	}
	llmMu.Lock()
	if s := strings.TrimSpace(req.PerplexityKey); s != "" {
		llmPerplexityKey = s
	}
	if s := strings.TrimSpace(req.ClaudeKey); s != "" {
		llmClaudeKey = s
	}
	if s := strings.TrimSpace(req.TavilyKey); s != "" {
		llmTavilyKey = s
	}
	llmMu.Unlock()
	saveLLMConfig()
	json200(w, map[string]any{"success": true, "message": "API 키 저장 완료"})
}

func handleLLMChat(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Messages  []groqMsg `json:"messages"`
		MaxTokens int       `json:"max_tokens"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Messages) == 0 {
		writeJSON(w, 400, map[string]any{"success": false, "message": "messages 필요"})
		return
	}
	if req.MaxTokens == 0 {
		req.MaxTokens = 1024
	}
	text, provider, err := callGroqWithFallback(req.Messages, req.MaxTokens, false)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}
	json200(w, map[string]any{"success": true, "text": text, "provider": provider})
}
