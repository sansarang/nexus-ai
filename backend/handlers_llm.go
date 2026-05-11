//go:build windows

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

// callOpenAICompat: Perplexity API 호출 (OpenAI 호환 포맷)
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
	if jsonMode {
		rb.RespFmt = &struct{ Type string `json:"type"` }{Type: "json_object"}
	}

	body, _ := json.Marshal(rb)
	httpReq, err := http.NewRequest("POST", baseURL, bytes.NewReader(body))
	if err != nil {
		return "", 0, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", 0, fmt.Errorf("연결 실패 (%s): %w", model, err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)

	var gr struct {
		Choices []struct {
			Message struct{ Content string `json:"content"` } `json:"message"`
		} `json:"choices"`
		Usage *struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
		Error *struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &gr); err != nil {
		return "", 0, fmt.Errorf("응답 파싱 실패: %w", err)
	}
	if gr.Error != nil {
		return "", 0, fmt.Errorf("[%s] %s: %s", model, gr.Error.Type, gr.Error.Message)
	}
	if len(gr.Choices) == 0 {
		return "", 0, fmt.Errorf("응답 비어 있음 (%s)", model)
	}
	tokens := 0
	if gr.Usage != nil {
		tokens = gr.Usage.PromptTokens + gr.Usage.CompletionTokens
	}
	return gr.Choices[0].Message.Content, tokens, nil
}

// callGroq: Perplexity API 호출 (이름 유지 — 호출부 변경 최소화)
func callGroq(apiKey, model string, msgs []groqMsg, maxTokens int, jsonMode bool) (string, int, error) {
	llmMu.RLock()
	pKey := llmPerplexityKey
	llmMu.RUnlock()

	useKey := pKey
	if useKey == "" {
		useKey = apiKey
	}

	// 구 Groq 모델명 → Perplexity 모델로 교정
	pModel := model
	switch model {
	case "llama-3.3-70b-versatile", "llama-3.1-70b-versatile":
		pModel = pplxChatModel
	case "llama-3.1-8b-instant", "llama-3.2-3b-preview":
		pModel = pplxFastModel
	}

	return callOpenAICompat(useKey, pplxAPIBase, pModel, msgs, maxTokens, jsonMode)
}

// callGroqVision: Vision 미지원 (Perplexity는 이미지 입력 불가)
func callGroqVision(_, _, _, _ string) (string, error) {
	return "", fmt.Errorf("Vision 기능은 현재 지원되지 않습니다")
}

// callGroqWithFallback: Perplexity → Claude 순서로 시도
func callGroqWithFallback(msgs []groqMsg, maxTokens int, jsonMode bool) (string, string, error) {
	llmMu.RLock()
	pKey := llmPerplexityKey
	cKey := llmClaudeKey
	llmMu.RUnlock()

	if pKey != "" {
		answer, _, err := callOpenAICompat(pKey, pplxAPIBase, pplxChatModel, msgs, maxTokens, jsonMode)
		if err == nil {
			return answer, "perplexity", nil
		}
		if cKey != "" {
			ans, cErr := callClaude(cKey, msgs, maxTokens)
			if cErr == nil {
				return ans, "claude-fallback", nil
			}
		}
		return "", "", fmt.Errorf("Perplexity 오류: %v", err)
	}
	if cKey != "" {
		ans, err := callClaude(cKey, msgs, maxTokens)
		if err == nil {
			return ans, "claude", nil
		}
		return "", "", err
	}
	return "", "", fmt.Errorf("Perplexity API 키가 설정되지 않았습니다")
}

// callClaude: Anthropic 직접 호출 (fallback)
func callClaude(apiKey string, msgs []groqMsg, maxTokens int) (string, error) {
	if apiKey == "" {
		return "", fmt.Errorf("Claude API 키 미설정")
	}

	type cContent struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	type cMsg struct {
		Role    string     `json:"role"`
		Content []cContent `json:"content"`
	}

	var system string
	var cMsgs []cMsg
	for _, m := range msgs {
		if m.Role == "system" {
			if s, ok := m.Content.(string); ok {
				system = s
			}
			continue
		}
		if text, ok := m.Content.(string); ok && text != "" {
			cMsgs = append(cMsgs, cMsg{
				Role:    m.Role,
				Content: []cContent{{Type: "text", Text: text}},
			})
		}
	}
	if maxTokens == 0 {
		maxTokens = 1024
	}

	reqBody := map[string]any{
		"model":      claudeModel,
		"max_tokens": maxTokens,
		"system":     system,
		"messages":   cMsgs,
	}
	body, _ := json.Marshal(reqBody)
	httpReq, _ := http.NewRequest("POST", claudeAPIBase, bytes.NewReader(body))
	httpReq.Header.Set("x-api-key", apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("Claude 연결 실패: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Content []struct{ Text string `json:"text"` } `json:"content"`
		Error   *struct{ Message string `json:"message"` } `json:"error"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Error != nil {
		return "", fmt.Errorf("Claude: %s", result.Error.Message)
	}
	if len(result.Content) == 0 {
		return "", fmt.Errorf("Claude 응답 없음")
	}
	return result.Content[0].Text, nil
}

// ── 설정 영속화 ──────────────────────────────────────────────────

func llmConfigPath() string {
	appdata := os.Getenv("APPDATA")
	if appdata == "" {
		appdata = os.TempDir()
	}
	dir := filepath.Join(appdata, "Nexus")
	os.MkdirAll(dir, 0700)
	return filepath.Join(dir, "nexus_llm_config.json")
}

type llmConfigFile struct {
	PerplexityKey string `json:"perplexity_key"`
	ClaudeKey     string `json:"claude_key"`
	TavilyKey     string `json:"tavily_key"`
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
			llmPerplexityKey = decryptDPAPI(v)
		} else if v := raw["groq_key"]; v != "" {
			llmPerplexityKey = decryptDPAPI(v)
		}
		if v := raw["claude_key"]; v != "" {
			llmClaudeKey = decryptDPAPI(v)
		}
		if v := raw["tavily_key"]; v != "" {
			llmTavilyKey = decryptDPAPI(v)
		}
		llmMu.Unlock()
	}
}

func saveLLMConfig() {
	llmMu.RLock()
	cfg := llmConfigFile{
		PerplexityKey: encryptDPAPI(llmPerplexityKey),
		ClaudeKey:     encryptDPAPI(llmClaudeKey),
		TavilyKey:     encryptDPAPI(llmTavilyKey),
	}
	llmMu.RUnlock()
	data, _ := json.MarshalIndent(cfg, "", "  ")
	os.WriteFile(llmConfigPath(), data, 0600)
}

// GET|POST /api/llm/config
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
		ApiKey        string `json:"apiKey"` // 하위 호환
		ClaudeKey     string `json:"claude_key"`
		TavilyKey     string `json:"tavily_key"`
	}
	json.NewDecoder(r.Body).Decode(&req)
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

// POST /api/llm/chat
func handleLLMChat(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Messages  []groqMsg `json:"messages"`
		MaxTokens int       `json:"max_tokens"`
		JSONMode  bool      `json:"json_mode"`
		Fast      bool      `json:"fast"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Messages) == 0 {
		writeJSON(w, 400, map[string]any{"success": false, "message": "messages 배열 필요"})
		return
	}
	if req.MaxTokens == 0 {
		req.MaxTokens = 2048
	}

	llmMu.RLock()
	pKey := llmPerplexityKey
	cKey := llmClaudeKey
	llmMu.RUnlock()

	model := pplxChatModel
	if req.Fast {
		model = pplxFastModel
	}

	answer, tokens, err := callOpenAICompat(pKey, pplxAPIBase, model, req.Messages, req.MaxTokens, req.JSONMode)
	if err != nil {
		if cKey != "" {
			ans, err2 := callClaude(cKey, req.Messages, req.MaxTokens)
			if err2 == nil {
				json200(w, map[string]any{"success": true, "answer": ans, "model": "claude-fallback", "tokens": 0})
				return
			}
		}
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}
	json200(w, map[string]any{"success": true, "answer": answer, "model": model, "tokens": tokens})
}

// POST /api/llm/vision — Vision 미지원
func handleLLMVision(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 400, map[string]any{
		"success": false,
		"message": "Vision(이미지 분석) 기능은 현재 지원되지 않습니다.",
	})
}

// POST /api/llm/doc-summary
func handleLLMDocSummary(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FilePath string `json:"file_path"`
		Question string `json:"question"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.FilePath == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "file_path 필요"})
		return
	}

	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()
	if gKey == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "Perplexity API 키 미설정"})
		return
	}

	text, err := extractDocumentText(req.FilePath)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": "문서 읽기 실패: " + err.Error()})
		return
	}
	if len(text) > 8000 {
		text = text[:8000] + "\n...(문서가 길어 앞부분만 분석)"
	}

	question := req.Question
	if question == "" {
		question = "이 문서의 핵심 내용을 5줄로 요약하고, 중요 수치·날짜·이름을 목록으로 정리해주세요."
	}

	prompt := fmt.Sprintf("다음 문서를 분석해주세요:\n\n%s\n\n요청: %s", text, question)
	answer, _, err := callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 2048, false)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}
	json200(w, map[string]any{"success": true, "summary": answer, "file": req.FilePath})
}

// POST /api/llm/doc-compare
func handleLLMDocCompare(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FileA string `json:"file_a"`
		FileB string `json:"file_b"`
		Focus string `json:"focus"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.FileA == "" || req.FileB == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "file_a, file_b 필요"})
		return
	}

	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()
	if gKey == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "Perplexity API 키 미설정"})
		return
	}

	textA, errA := extractDocumentText(req.FileA)
	textB, errB := extractDocumentText(req.FileB)
	if errA != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": "파일A 오류: " + errA.Error()})
		return
	}
	if errB != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": "파일B 오류: " + errB.Error()})
		return
	}

	if len(textA) > 4000 {
		textA = textA[:4000] + "..."
	}
	if len(textB) > 4000 {
		textB = textB[:4000] + "..."
	}

	focusMap := map[string]string{
		"numbers": "숫자·금액·날짜 불일치 집중 분석",
		"changes": "추가·삭제·수정 문장 집중 분석",
		"both":    "숫자 불일치, 추가/삭제/수정, 의미 변화 종합 분석",
	}
	focus := req.Focus
	if focus == "" {
		focus = "both"
	}
	instr := focusMap[focus]
	if instr == "" {
		instr = focusMap["both"]
	}

	prompt := fmt.Sprintf(`두 문서를 비교 분석하세요. %s

=== 문서 A ===
%s

=== 문서 B ===
%s

반드시 다음 JSON 형식으로만 응답:
{
  "summary": "전체 차이 요약 2-3문장",
  "total_differences": 숫자,
  "differences": [
    {"type":"added|deleted|modified|number_mismatch","location":"위치","description":"설명","a_value":"A값","b_value":"B값","severity":"low|medium|high"}
  ],
  "risk_level": "low|medium|high",
  "recommendation": "검토 권고사항"
}`, instr, textA, textB)

	answer, _, err := callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 2048, true)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}

	var parsed map[string]any
	json.Unmarshal([]byte(answer), &parsed)
	json200(w, map[string]any{"success": true, "result": parsed, "file_a": req.FileA, "file_b": req.FileB})
}

// POST /api/llm/deep-search — AI 보강 파일 검색
func handleLLMDeepSearch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query      string `json:"query"`
		Folder     string `json:"folder"`
		MaxResults int    `json:"max_results"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Query == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "query 필요"})
		return
	}
	if req.MaxResults == 0 {
		req.MaxResults = 15
	}

	llmMu.RLock()
	gKey := llmPerplexityKey
	llmMu.RUnlock()

	keywords := strings.Fields(req.Query)
	if gKey != "" {
		extractPrompt := fmt.Sprintf(`검색 쿼리에서 파일 검색에 쓸 핵심 키워드만 추출:
쿼리: "%s"
JSON 응답: {"keywords": ["키워드1","키워드2"]}`, req.Query)

		raw, _, _ := callGroq(gKey, groqFastModel, []groqMsg{
			{Role: "user", Content: extractPrompt},
		}, 128, true)

		var p struct {
			Keywords []string `json:"keywords"`
		}
		if json.Unmarshal([]byte(raw), &p) == nil && len(p.Keywords) > 0 {
			keywords = p.Keywords
		}
	}

	searchFolder := req.Folder
	if searchFolder == "" {
		searchFolder, _ = os.UserHomeDir()
	}
	hits := deepSearchFiles(strings.Join(keywords, " "), searchFolder, req.MaxResults*3)

	if gKey != "" && len(hits) > 3 {
		type hitItem struct {
			Path    string `json:"path"`
			Snippet string `json:"snippet"`
		}
		hitList := make([]hitItem, 0, len(hits))
		for _, h := range hits {
			hitList = append(hitList, hitItem{Path: h.Path, Snippet: h.Snippet})
		}
		hitJSON, _ := json.Marshal(hitList)

		rankPrompt := fmt.Sprintf(`사용자 검색 의도: "%s"
파일 목록에서 관련성 높은 순으로 점수(0-100) 부여:
%s
JSON: {"ranked":[{"path":"경로","score":85},...]}`, req.Query, string(hitJSON))

		raw, _, _ := callGroq(gKey, groqFastModel, []groqMsg{
			{Role: "user", Content: rankPrompt},
		}, 512, true)

		var ranked struct {
			Ranked []struct {
				Path  string `json:"path"`
				Score int    `json:"score"`
			} `json:"ranked"`
		}
		if json.Unmarshal([]byte(raw), &ranked) == nil {
			scoreMap := map[string]int{}
			for _, rv := range ranked.Ranked {
				scoreMap[rv.Path] = rv.Score
			}
			for i, h := range hits {
				if s, ok := scoreMap[h.Path]; ok {
					hits[i].Score = s
				}
			}
			sortByScore(hits)
		}
	}

	if len(hits) > req.MaxResults {
		hits = hits[:req.MaxResults]
	}

	json200(w, map[string]any{
		"success":       true,
		"results":       hits,
		"total":         len(hits),
		"query":         req.Query,
		"keywords_used": keywords,
		"ai_enhanced":   gKey != "",
	})
}

func deepSearchFiles(query, folder string, maxResults int) []DeepSearchResult {
	searchExts := map[string]bool{
		".txt": true, ".md": true, ".csv": true, ".log": true,
		".docx": true, ".doc": true, ".xlsx": true, ".xls": true,
		".pdf": true, ".hwp": true, ".json": true, ".xml": true,
	}
	queryTerms := strings.Fields(strings.ToLower(query))
	var results []DeepSearchResult

	filepath.Walk(folder, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || len(results) >= maxResults {
			return nil
		}
		for _, skip := range []string{`\Windows\`, `\AppData\Local\Temp\`, `node_modules`, `.git`} {
			if strings.Contains(p, skip) {
				return nil
			}
		}
		ext := strings.ToLower(filepath.Ext(p))
		if !searchExts[ext] {
			return nil
		}

		score := 0
		snippet := ""
		nameLow := strings.ToLower(info.Name())

		for _, term := range queryTerms {
			if strings.Contains(nameLow, term) {
				score += 30
			}
		}
		if info.Size() < 10<<20 {
			text, err := extractDocumentText(p)
			if err == nil {
				tl := strings.ToLower(text)
				for _, term := range queryTerms {
					cnt := strings.Count(tl, term)
					if cnt > 0 {
						score += min(cnt*10, 40)
						if snippet == "" {
							idx := strings.Index(tl, term)
							if idx >= 0 {
								s := max2(idx-40, 0)
								e := idx + len(term) + 80
								if e > len(text) {
									e = len(text)
								}
								snippet = "..." + text[s:e] + "..."
							}
						}
					}
				}
			}
		}

		if score > 0 {
			results = append(results, DeepSearchResult{
				Name:    info.Name(),
				Path:    p,
				Ext:     ext,
				SizeMB:  float64(info.Size()) / (1 << 20),
				ModTime: info.ModTime().Format("2006-01-02 15:04"),
				Snippet: snippet,
				Score:   min(score, 100),
			})
		}
		return nil
	})

	sortByScore(results)
	return results
}

func max2(a, b int) int {
	if a > b {
		return a
	}
	return b
}
