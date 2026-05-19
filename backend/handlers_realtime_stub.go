//go:build !windows

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ══════════════════════════════════════════════════════════════
//  실시간 환율 / 주가 / 암호화폐
//  exchangerate.host (무료, 키 불필요) + CoinGecko
// ══════════════════════════════════════════════════════════════

type exchangeResult struct {
	Base   string             `json:"base"`
	Rates  map[string]float64 `json:"rates"`
	Date   string             `json:"date"`
	Source string             `json:"source"`
}

// currencySymbols: 통화 코드 ↔ 이름 매핑
var currencySymbols = map[string]string{
	"USD": "달러", "EUR": "유로", "JPY": "엔화", "CNY": "위안", "GBP": "파운드",
	"KRW": "원", "HKD": "홍콩달러", "SGD": "싱가포르달러", "AUD": "호주달러",
	"CAD": "캐나다달러", "CHF": "스위스프랑", "THB": "태국바트", "VND": "베트남동",
	"MYR": "말레이시아링깃", "PHP": "필리핀페소", "IDR": "인도네시아루피아",
}

// detectCurrencies: 메시지에서 통화 코드 추출
func detectCurrencies(msg string) (from, to string) {
	upper := strings.ToUpper(msg)
	lower := strings.ToLower(msg)

	// 명시적 코드
	for code := range currencySymbols {
		if strings.Contains(upper, code) {
			if from == "" {
				from = code
			} else if to == "" && code != from {
				to = code
			}
		}
	}

	// 자연어 → 코드 매핑
	naturalMap := map[string]string{
		"달러": "USD", "dollar": "USD", "엔화": "JPY", "엔": "JPY", "yen": "JPY",
		"유로": "EUR", "euro": "EUR", "위안": "CNY", "yuan": "CNY", "rmb": "CNY",
		"파운드": "GBP", "pound": "GBP", "원화": "KRW",
		"홍콩": "HKD", "싱가포르": "SGD", "호주": "AUD", "캐나다": "CAD",
		"스위스": "CHF", "태국": "THB", "베트남": "VND", "말레이시아": "MYR",
	}
	for word, code := range naturalMap {
		if strings.Contains(lower, word) {
			if from == "" {
				from = code
			} else if to == "" && code != from {
				to = code
			}
		}
	}

	if from == "" {
		from = "USD"
	}
	if to == "" {
		to = "KRW"
	}
	return
}

// fetchExchangeRate: Frankfurter API (ECB 기반, 무료, 키 불필요)
func fetchExchangeRate(from, to string) (float64, string, error) {
	// 1차: Frankfurter (EUR 기반이라 USD→KRW는 2단계)
	url := fmt.Sprintf("https://api.frankfurter.app/latest?from=%s&to=%s", from, to)
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Get(url)
	if err == nil && resp.StatusCode == 200 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		var d struct {
			Base  string             `json:"base"`
			Rates map[string]float64 `json:"rates"`
			Date  string             `json:"date"`
		}
		if json.Unmarshal(body, &d) == nil {
			if rate, ok := d.Rates[to]; ok {
				return rate, d.Date, nil
			}
		}
	}

	// 2차: ExchangeRate-API open endpoint
	url2 := fmt.Sprintf("https://open.er-api.com/v6/latest/%s", from)
	resp2, err2 := client.Get(url2)
	if err2 != nil {
		return 0, "", fmt.Errorf("환율 API 연결 실패")
	}
	defer resp2.Body.Close()
	body2, _ := io.ReadAll(io.LimitReader(resp2.Body, 64*1024))
	var d2 struct {
		Rates      map[string]float64 `json:"rates"`
		TimeLastUpdate string         `json:"time_last_update_utc"`
	}
	if err := json.Unmarshal(body2, &d2); err != nil {
		return 0, "", fmt.Errorf("환율 파싱 실패")
	}
	if rate, ok := d2.Rates[to]; ok {
		date := d2.TimeLastUpdate
		if len(date) > 16 {
			date = date[:16]
		}
		return rate, date, nil
	}
	return 0, "", fmt.Errorf("%s→%s 환율 정보 없음", from, to)
}

// handleExchangeRate: POST /api/exchange-rate
func handleExchangeRate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		From    string  `json:"from"`
		To      string  `json:"to"`
		Amount  float64 `json:"amount"`
		Message string  `json:"message"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	from, to := req.From, req.To
	if from == "" || to == "" {
		from, to = detectCurrencies(req.Message)
	}
	from = strings.ToUpper(from)
	to = strings.ToUpper(to)
	if req.Amount == 0 {
		req.Amount = 1
	}

	rate, date, err := fetchExchangeRate(from, to)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}

	converted := rate * req.Amount
	fromName := currencySymbols[from]
	if fromName == "" {
		fromName = from
	}
	toName := currencySymbols[to]
	if toName == "" {
		toName = to
	}

	var msg string
	if req.Amount != 1 {
		msg = fmt.Sprintf("%.2f %s(%s) = **%.2f %s(%s)**\n환율: 1%s = %.4f%s (%s 기준)",
			req.Amount, fromName, from,
			converted, toName, to,
			from, rate, to, date)
	} else {
		msg = fmt.Sprintf("1 %s(%s) = **%.4f %s(%s)**\n(%s 기준)",
			fromName, from, rate, toName, to, date)
	}

	json200(w, map[string]any{
		"success":   true,
		"from":      from,
		"to":        to,
		"rate":      rate,
		"amount":    req.Amount,
		"converted": converted,
		"date":      date,
		"message":   msg,
	})
}

// ── 암호화폐 ──────────────────────────────────────────────────

var cryptoIDs = map[string]string{
	"BTC": "bitcoin", "ETH": "ethereum", "XRP": "ripple",
	"SOL": "solana", "DOGE": "dogecoin", "ADA": "cardano",
	"BNB": "binancecoin", "AVAX": "avalanche-2", "DOT": "polkadot",
}

var cryptoNames = map[string]string{
	"비트코인": "BTC", "bitcoin": "BTC", "이더리움": "ETH", "ethereum": "ETH",
	"리플": "XRP", "xrp": "XRP", "ripple": "XRP",
	"솔라나": "SOL", "solana": "SOL", "도지": "DOGE", "doge": "DOGE", "dogecoin": "DOGE",
	"에이다": "ADA", "cardano": "ADA", "바이낸스": "BNB",
}

func detectCrypto(msg string) string {
	lower := strings.ToLower(msg)
	for name, code := range cryptoNames {
		if strings.Contains(lower, name) {
			return code
		}
	}
	upper := strings.ToUpper(msg)
	for code := range cryptoIDs {
		if strings.Contains(upper, code) {
			return code
		}
	}
	return ""
}

func fetchCryptoPrice(symbol string) (float64, float64, error) {
	id, ok := cryptoIDs[strings.ToUpper(symbol)]
	if !ok {
		return 0, 0, fmt.Errorf("지원하지 않는 코인: %s", symbol)
	}
	url := fmt.Sprintf("https://api.coingecko.com/api/v3/simple/price?ids=%s&vs_currencies=krw,usd&include_24hr_change=true", id)
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return 0, 0, fmt.Errorf("CoinGecko 연결 실패")
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 32*1024))
	var d map[string]map[string]float64
	if json.Unmarshal(body, &d) != nil {
		return 0, 0, fmt.Errorf("코인 데이터 파싱 실패")
	}
	if info, ok := d[id]; ok {
		return info["krw"], info["usd"], nil
	}
	return 0, 0, fmt.Errorf("코인 정보 없음")
}

// ── 주가 (한국 코스피 + 미국 나스닥) ─────────────────────────

// fetchStockInfo: Yahoo Finance 비공식 API (무료)
func fetchStockInfo(ticker string) (float64, float64, string, error) {
	// Yahoo Finance v8 quote API
	url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=1d", ticker)
	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, "", fmt.Errorf("주가 API 연결 실패")
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 128*1024))

	var d struct {
		Chart struct {
			Result []struct {
				Meta struct {
					RegularMarketPrice         float64 `json:"regularMarketPrice"`
					ChartPreviousClose         float64 `json:"chartPreviousClose"`
					Currency                   string  `json:"currency"`
					ExchangeName               string  `json:"exchangeName"`
				} `json:"meta"`
			} `json:"result"`
		} `json:"chart"`
	}
	if json.Unmarshal(body, &d) != nil || len(d.Chart.Result) == 0 {
		return 0, 0, "", fmt.Errorf("주가 파싱 실패")
	}
	m := d.Chart.Result[0].Meta
	change := 0.0
	if m.ChartPreviousClose != 0 {
		change = (m.RegularMarketPrice - m.ChartPreviousClose) / m.ChartPreviousClose * 100
	}
	return m.RegularMarketPrice, change, m.Currency, nil
}

// stockTickers: 자연어 → 티커 심볼
var stockTickers = map[string]string{
	"삼성전자": "005930.KS", "삼성": "005930.KS",
	"sk하이닉스": "000660.KS", "하이닉스": "000660.KS",
	"카카오": "035720.KS", "네이버": "035420.KS", "naver": "035420.KS",
	"lg전자": "066570.KS", "현대차": "005380.KS", "기아": "000270.KS",
	"셀트리온": "068270.KS", "포스코": "005490.KS",
	"코스피": "^KS11", "코스닥": "^KQ11",
	"나스닥": "^IXIC", "nasdaq": "^IXIC",
	"s&p500": "^GSPC", "sp500": "^GSPC", "다우": "^DJI",
	"애플": "AAPL", "apple": "AAPL",
	"테슬라": "TSLA", "tesla": "TSLA",
	"엔비디아": "NVDA", "nvidia": "NVDA",
	"구글": "GOOGL", "google": "GOOGL",
	"마이크로소프트": "MSFT", "microsoft": "MSFT",
	"아마존": "AMZN", "amazon": "AMZN",
	"메타": "META", "meta": "META",
	"넷플릭스": "NFLX", "netflix": "NFLX",
}

func detectStockTicker(msg string) (string, string) {
	lower := strings.ToLower(msg)
	for name, ticker := range stockTickers {
		if strings.Contains(lower, name) {
			return ticker, name
		}
	}
	return "", ""
}

// formatStockMsg: 주가 결과 자연어 포맷
func formatStockMsg(name, ticker string, price, change float64, currency string, eng bool) string {
	arrow := "▲"
	if change < 0 {
		arrow = "▼"
	}
	if eng {
		return fmt.Sprintf("%s (%s): **%.2f %s** %s%.2f%%",
			name, ticker, price, currency, arrow, change)
	}
	return fmt.Sprintf("%s(%s): **%.2f %s** %s%.2f%%",
		name, ticker, price, currency, arrow, change)
}
