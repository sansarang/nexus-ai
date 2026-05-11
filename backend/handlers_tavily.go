package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type tavilyResult struct {
	Summary string
	Items   []map[string]string
}

// isNewsQuery 뉴스/최신 관련 쿼리 여부 판별
func isNewsQuery(query string) bool {
	keywords := []string{"뉴스", "news", "최신", "오늘", "어제", "이번주", "최근", "속보", "이슈", "today", "latest", "breaking"}
	q := strings.ToLower(query)
	for _, k := range keywords {
		if strings.Contains(q, k) {
			return true
		}
	}
	return false
}

func tavilySearch(apiKey, query string, maxItems int) (tavilyResult, bool) {
	payload := map[string]any{
		"api_key":      apiKey,
		"query":        query,
		"max_results":  maxItems,
		"search_depth": "advanced",
	}
	// 뉴스/최신 쿼리는 최근 3일 이내 결과만
	if isNewsQuery(query) {
		payload["days"] = 3
		payload["topic"] = "news"
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", "https://api.tavily.com/search", bytes.NewReader(body))
	if err != nil {
		return tavilyResult{}, false
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return tavilyResult{}, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return tavilyResult{}, false
	}
	raw, _ := io.ReadAll(resp.Body)
	var data struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
		} `json:"results"`
		Answer string `json:"answer"`
	}
	if json.Unmarshal(raw, &data) != nil || len(data.Results) == 0 {
		return tavilyResult{}, false
	}

	// 블로그 쓰레기 패턴 제거용 정규식
	urlPattern := regexp.MustCompile(`https?://\S+`)
	junkPattern := regexp.MustCompile(`(URL 복사|이웃추가|공유하기|신고하기|본문 기타|카테고리 이동|ALL DAY|프로파일|뽕개|\d{4}\.\s*\d{1,2}\.\s*\d{1,2}\.?\s*\d{1,2}:\d{2})`)

	items := make([]map[string]string, 0, len(data.Results))
	contentLines := make([]string, 0, len(data.Results))
	for _, r := range data.Results {
		items = append(items, map[string]string{"title": r.Title, "url": r.URL})
		// URL·블로그 메타 제거 후 핵심 텍스트만 추출
		snippet := urlPattern.ReplaceAllString(r.Content, "")
		snippet = junkPattern.ReplaceAllString(snippet, "")
		// 연속 공백/개행 정리
		snippet = strings.Join(strings.Fields(snippet), " ")
		if len([]rune(snippet)) > 200 {
			runes := []rune(snippet)
			snippet = string(runes[:200])
		}
		if snippet != "" {
			contentLines = append(contentLines, fmt.Sprintf("• %s: %s", r.Title, snippet))
		}
	}

	// Tavily answer가 있으면 그걸 우선 (이미 자연어 요약)
	// 없으면 정제된 본문만 전달 (Groq가 최종 정제)
	summary := data.Answer
	if summary == "" {
		summary = strings.Join(contentLines, "\n")
	}

	return tavilyResult{Summary: summary, Items: items}, true
}
