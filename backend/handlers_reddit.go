//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// ── Reddit 전역 상태 ────────────────────────────────────────────────────────

var (
	redditMu           sync.RWMutex
	redditClientID     string
	redditClientSecret string
	redditAccessToken  string
	redditTokenExpiry  time.Time
)

// ── Reddit OAuth2 토큰 (App-only / client_credentials) ─────────────────────

func redditGetToken() (string, error) {
	redditMu.RLock()
	if redditAccessToken != "" && time.Now().Before(redditTokenExpiry) {
		t := redditAccessToken
		redditMu.RUnlock()
		return t, nil
	}
	redditMu.RUnlock()

	redditMu.RLock()
	cid := redditClientID
	csec := redditClientSecret
	redditMu.RUnlock()

	if cid == "" || csec == "" {
		return "", fmt.Errorf("Reddit Client ID/Secret not configured")
	}

	body := url.Values{}
	body.Set("grant_type", "client_credentials")

	req, _ := http.NewRequest("POST", "https://www.reddit.com/api/v1/access_token", strings.NewReader(body.Encode()))
	req.SetBasicAuth(cid, csec)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "NexusAI/1.0 (by nexus-app)")

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var res struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}
	if res.Error != "" {
		return "", fmt.Errorf("reddit oauth error: %s", res.Error)
	}

	redditMu.Lock()
	redditAccessToken = res.AccessToken
	redditTokenExpiry = time.Now().Add(time.Duration(res.ExpiresIn-60) * time.Second)
	redditMu.Unlock()

	return res.AccessToken, nil
}

// ── Reddit 검색 핵심 함수 ───────────────────────────────────────────────────

type RedditPost struct {
	Title     string `json:"title"`
	Subreddit string `json:"subreddit"`
	Score     int    `json:"score"`
	URL       string `json:"url"`
	Body      string `json:"body"`
	Author    string `json:"author"`
	CreatedAt string `json:"created_at"`
	Comments  int    `json:"comments"`
}

type RedditSearchResult struct {
	Posts   []RedditPost `json:"posts"`
	Query   string       `json:"query"`
	Summary string       `json:"summary"`
}

func redditSearch(query, subreddit string, limit int, sort string) (RedditSearchResult, error) {
	token, err := redditGetToken()
	if err != nil {
		return RedditSearchResult{}, err
	}

	if limit <= 0 || limit > 25 {
		limit = 10
	}
	if sort == "" {
		sort = "relevance"
	}

	searchURL := fmt.Sprintf(
		"https://oauth.reddit.com/search?q=%s&limit=%d&sort=%s&type=link",
		url.QueryEscape(query), limit, sort,
	)
	if subreddit != "" {
		searchURL = fmt.Sprintf(
			"https://oauth.reddit.com/r/%s/search?q=%s&limit=%d&sort=%s&restrict_sr=1&type=link",
			subreddit, url.QueryEscape(query), limit, sort,
		)
	}

	req, _ := http.NewRequest("GET", searchURL, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", "NexusAI/1.0 (by nexus-app)")

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return RedditSearchResult{}, err
	}
	defer resp.Body.Close()

	var raw struct {
		Data struct {
			Children []struct {
				Data struct {
					Title     string  `json:"title"`
					Subreddit string  `json:"subreddit"`
					Score     int     `json:"score"`
					URL       string  `json:"url"`
					Selftext  string  `json:"selftext"`
					Author    string  `json:"author"`
					Created   float64 `json:"created_utc"`
					NumComments int   `json:"num_comments"`
					Permalink string  `json:"permalink"`
				} `json:"data"`
			} `json:"children"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return RedditSearchResult{}, err
	}

	posts := make([]RedditPost, 0, len(raw.Data.Children))
	for _, c := range raw.Data.Children {
		d := c.Data
		body := d.Selftext
		if len(body) > 300 {
			body = body[:300] + "..."
		}
		postURL := d.URL
		if d.Permalink != "" && !strings.HasPrefix(d.URL, "https://www.reddit.com") {
			postURL = "https://www.reddit.com" + d.Permalink
		}
		posts = append(posts, RedditPost{
			Title:     d.Title,
			Subreddit: d.Subreddit,
			Score:     d.Score,
			URL:       postURL,
			Body:      body,
			Author:    d.Author,
			CreatedAt: time.Unix(int64(d.Created), 0).Format("2006-01-02"),
			Comments:  d.NumComments,
		})
	}

	return RedditSearchResult{Query: query, Posts: posts}, nil
}

// ── HTTP 핸들러 ─────────────────────────────────────────────────────────────

// POST /api/reddit/search
func handleRedditSearch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query     string `json:"query"`
		Subreddit string `json:"subreddit"`
		Limit     int    `json:"limit"`
		Sort      string `json:"sort"` // relevance | hot | new | top
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Query == "" {
		http.Error(w, "query required", http.StatusBadRequest)
		return
	}

	result, err := redditSearch(req.Query, req.Subreddit, req.Limit, req.Sort)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GET /api/reddit/trending?subreddit=stocks
func handleRedditTrending(w http.ResponseWriter, r *http.Request) {
	subreddit := r.URL.Query().Get("subreddit")
	if subreddit == "" {
		subreddit = "all"
	}

	token, err := redditGetToken()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	trendURL := fmt.Sprintf("https://oauth.reddit.com/r/%s/hot?limit=10", subreddit)
	req, _ := http.NewRequest("GET", trendURL, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", "NexusAI/1.0 (by nexus-app)")

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var raw struct {
		Data struct {
			Children []struct {
				Data struct {
					Title     string  `json:"title"`
					Subreddit string  `json:"subreddit"`
					Score     int     `json:"score"`
					URL       string  `json:"url"`
					Author    string  `json:"author"`
					Created   float64 `json:"created_utc"`
					NumComments int   `json:"num_comments"`
					Permalink string  `json:"permalink"`
				} `json:"data"`
			} `json:"children"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	posts := make([]RedditPost, 0, len(raw.Data.Children))
	for _, c := range raw.Data.Children {
		d := c.Data
		posts = append(posts, RedditPost{
			Title:     d.Title,
			Subreddit: d.Subreddit,
			Score:     d.Score,
			URL:       "https://www.reddit.com" + d.Permalink,
			Author:    d.Author,
			CreatedAt: time.Unix(int64(d.Created), 0).Format("2006-01-02"),
			Comments:  d.NumComments,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"subreddit": subreddit, "posts": posts})
}

// POST /api/reddit/config — Reddit Client ID/Secret 저장
func handleRedditConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	redditMu.Lock()
	if req.ClientID != "" {
		redditClientID = req.ClientID
	}
	if req.ClientSecret != "" {
		redditClientSecret = req.ClientSecret
		redditAccessToken = "" // 토큰 초기화 → 재발급
	}
	redditMu.Unlock()

	saveRedditConfig()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

// GET /api/reddit/config/status
func handleRedditConfigStatus(w http.ResponseWriter, r *http.Request) {
	redditMu.RLock()
	configured := redditClientID != "" && redditClientSecret != ""
	redditMu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"configured": configured})
}

// ── 설정 파일 저장/로드 ─────────────────────────────────────────────────────

func redditConfigPath() string {
	home, _ := os.UserHomeDir()
	return home + "/.nexus/reddit_config.json"
}

func saveRedditConfig() {
	redditMu.RLock()
	data := map[string]string{
		"client_id":     redditClientID,
		"client_secret": encryptDPAPI(redditClientSecret),
	}
	redditMu.RUnlock()
	b, _ := json.MarshalIndent(data, "", "  ")
	path := redditConfigPath()
	_ = os.MkdirAll(path[:strings.LastIndex(path, "/")], 0755)
	_ = os.WriteFile(path, b, 0600)
}

func loadRedditConfig() {
	type redditCfg struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}
	data, err := os.ReadFile(redditConfigPath())
	if err != nil {
		return
	}
	var cfg redditCfg
	if err := json.Unmarshal(data, &cfg); err != nil {
		return
	}
	redditMu.Lock()
	redditClientID = cfg.ClientID
	if cfg.ClientSecret != "" {
		redditClientSecret = decryptDPAPI(cfg.ClientSecret)
	}
	redditMu.Unlock()
}
