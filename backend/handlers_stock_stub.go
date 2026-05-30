//go:build !windows

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ── 주식 관심 종목 저장 경로 ────────────────────────────────

func stockWatchlistPath() string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".nexus")
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "watchlist.json")
}

func loadWatchlist() []map[string]string {
	data, err := os.ReadFile(stockWatchlistPath())
	if err != nil {
		return defaultWatchlist()
	}
	var list []map[string]string
	if json.Unmarshal(data, &list) != nil {
		return defaultWatchlist()
	}
	return list
}

func defaultWatchlist() []map[string]string {
	return []map[string]string{
		{"symbol": "005930.KS", "name": "삼성전자"},
		{"symbol": "000660.KS", "name": "SK하이닉스"},
		{"symbol": "035420.KS", "name": "NAVER"},
		{"symbol": "AAPL", "name": "Apple"},
		{"symbol": "NVDA", "name": "NVIDIA"},
	}
}

func saveWatchlist(list []map[string]string) {
	data, _ := json.Marshal(list)
	os.WriteFile(stockWatchlistPath(), data, 0644)
}

// ── Yahoo Finance 비공식 API ────────────────────────────────

type StockQuote struct {
	Symbol        string  `json:"symbol"`
	Name          string  `json:"name"`
	Price         float64 `json:"price"`
	Change        float64 `json:"change"`
	ChangePercent float64 `json:"change_percent"`
	Currency      string  `json:"currency"`
	Market        string  `json:"market"`
	UpdatedAt     string  `json:"updated_at"`
}

func fetchYahooFinance(symbol string) (*StockQuote, error) {
	url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=1d", symbol)
	client := &http.Client{Timeout: 8 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Chart struct {
			Result []struct {
				Meta struct {
					Symbol             string  `json:"symbol"`
					ShortName          string  `json:"shortName"`
					RegularMarketPrice float64 `json:"regularMarketPrice"`
					PreviousClose      float64 `json:"previousClose"`
					Currency           string  `json:"currency"`
					ExchangeName       string  `json:"exchangeName"`
				} `json:"meta"`
			} `json:"result"`
			Error *struct{ Message string } `json:"error"`
		} `json:"chart"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	if result.Chart.Error != nil {
		return nil, fmt.Errorf("%s", result.Chart.Error.Message)
	}
	if len(result.Chart.Result) == 0 {
		return nil, fmt.Errorf("no data for %s", symbol)
	}

	meta := result.Chart.Result[0].Meta
	change := meta.RegularMarketPrice - meta.PreviousClose
	changePct := 0.0
	if meta.PreviousClose > 0 {
		changePct = change / meta.PreviousClose * 100
	}

	return &StockQuote{
		Symbol:        meta.Symbol,
		Name:          meta.ShortName,
		Price:         meta.RegularMarketPrice,
		Change:        change,
		ChangePercent: changePct,
		Currency:      meta.Currency,
		Market:        meta.ExchangeName,
		UpdatedAt:     time.Now().Format("15:04:05"),
	}, nil
}

// ── 핸들러 ────────────────────────────────────────────────

// GET /api/stock/quote?symbol=005930.KS
func handleStockQuote(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	if symbol == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "symbol 파라미터가 필요해요"})
		return
	}
	q, err := fetchYahooFinance(symbol)
	if err != nil {
		// Tavily 폴백
		llmMu.RLock()
		tKey := llmTavilyKey
		llmMu.RUnlock()
		if tKey != "" {
			tr, ok := tavilySearch(tKey, symbol+" 주가 오늘", 3)
			if ok {
				json200(w, map[string]any{
					"success": true,
					"symbol":  symbol,
					"source":  "search",
					"summary": tr.Summary,
					"message": fmt.Sprintf("%s 실시간 시세 조회 실패, 검색 결과로 대체", symbol),
				})
				return
			}
		}
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}
	sign := "+"
	if q.Change < 0 {
		sign = ""
	}
	json200(w, map[string]any{
		"success": true,
		"quote":   q,
		"message": fmt.Sprintf("%s %s %.2f (%s%.2f%%)", q.Name, q.Currency, q.Price, sign, q.ChangePercent),
	})
}

// GET /api/stock/watchlist
func handleStockWatchlist(w http.ResponseWriter, r *http.Request) {
	list := loadWatchlist()
	var quotes []map[string]any
	for _, item := range list {
		symbol := item["symbol"]
		name := item["name"]
		q, err := fetchYahooFinance(symbol)
		entry := map[string]any{"symbol": symbol, "name": name}
		if err == nil {
			entry["price"] = q.Price
			entry["change"] = q.Change
			entry["change_percent"] = q.ChangePercent
			entry["currency"] = q.Currency
			entry["updated_at"] = q.UpdatedAt
			sign := "+"
			if q.Change < 0 {
				sign = ""
			}
			entry["summary"] = fmt.Sprintf("%s%.2f%%", sign, q.ChangePercent)
		} else {
			entry["error"] = err.Error()
		}
		quotes = append(quotes, entry)
	}
	json200(w, map[string]any{"success": true, "watchlist": quotes, "count": len(quotes)})
}

// POST /api/stock/watchlist  {"symbol":"TSLA","name":"테슬라"}
func handleStockWatchlistAdd(w http.ResponseWriter, r *http.Request) {
	var req map[string]string
	tryDecodeBody(r, &req)
	symbol := strings.ToUpper(strings.TrimSpace(req["symbol"]))
	name := req["name"]
	if symbol == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("symbol 필요", "symbol required", getLang(r))})
		return
	}
	list := loadWatchlist()
	for _, item := range list {
		if item["symbol"] == symbol {
			json200(w, map[string]any{"success": true, "message": msgT(symbol+"은 이미 관심 종목이에요", symbol+" is already in your watchlist", getLang(r))})
			return
		}
	}
	if name == "" {
		name = symbol
	}
	list = append(list, map[string]string{"symbol": symbol, "name": name})
	saveWatchlist(list)
	json200(w, map[string]any{"success": true, "message": name + " 관심 종목 추가됨"})
}

// DELETE /api/stock/watchlist?symbol=TSLA
func handleStockWatchlistDelete(w http.ResponseWriter, r *http.Request) {
	symbol := strings.ToUpper(r.URL.Query().Get("symbol"))
	list := loadWatchlist()
	newList := list[:0]
	found := false
	for _, item := range list {
		if item["symbol"] == symbol {
			found = true
			continue
		}
		newList = append(newList, item)
	}
	if !found {
		writeJSON(w, 404, map[string]any{"success": false, "message": symbol + " 없음"})
		return
	}
	saveWatchlist(newList)
	json200(w, map[string]any{"success": true, "message": symbol + " 삭제됨"})
}

// GET /api/stock/summary  — 브리핑용 한줄 요약
func stockBriefSummary() string {
	list := loadWatchlist()
	if len(list) == 0 {
		return ""
	}
	var parts []string
	for _, item := range list[:min(3, len(list))] {
		q, err := fetchYahooFinance(item["symbol"])
		if err != nil {
			continue
		}
		sign := "▲"
		if q.Change < 0 {
			sign = "▼"
		}
		parts = append(parts, fmt.Sprintf("%s %s%.1f%%", q.Name, sign, q.ChangePercent))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " | ")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
