package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ══════════════════════════════════════════════════════════════
//  영상 검색·수집 강화 — Twitter/TikTok 쿠키 인증 + 멀티플랫폼
// ══════════════════════════════════════════════════════════════

// POST /api/video/search-enhanced
// body: { "query": "...", "platforms": ["youtube","tiktok","twitter"], "limit": 10 }
func handleVideoSearchEnhanced(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query     string   `json:"query"`
		Platforms []string `json:"platforms"`
		Limit     int      `json:"limit"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Query == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "query 필요"})
		return
	}
	if req.Limit == 0 {
		req.Limit = 10
	}
	if len(req.Platforms) == 0 {
		req.Platforms = []string{"youtube", "tiktok", "twitter"}
	}

	type VideoItem struct {
		Title    string `json:"title"`
		URL      string `json:"url"`
		Platform string `json:"platform"`
		Duration string `json:"duration,omitempty"`
		Views    string `json:"views,omitempty"`
		Channel  string `json:"channel,omitempty"`
	}

	var results []VideoItem

	for _, platform := range req.Platforms {
		items := searchVideoPlatform(req.Query, platform, req.Limit/len(req.Platforms)+1)
		for _, item := range items {
			results = append(results, VideoItem{
				Title:    item["title"],
				URL:      item["url"],
				Platform: platform,
				Duration: item["duration"],
				Views:    item["views"],
				Channel:  item["channel"],
			})
		}
	}

	msg := fmt.Sprintf("✅ **\"%s\"** 멀티플랫폼 영상 검색 완료 — 총 **%d개** (플랫폼: %s)",
		req.Query, len(results), strings.Join(req.Platforms, ", "))

	writeJSON(w, 200, map[string]any{
		"success": true,
		"query":   req.Query,
		"total":   len(results),
		"items":   results,
		"message": msg,
	})
}

// POST /api/video/download-with-cookie
// body: { "url": "...", "platform": "twitter|tiktok|youtube", "cookie_file": "..." }
func handleVideoDownloadCookie(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL        string `json:"url"`
		Platform   string `json:"platform"`
		CookieFile string `json:"cookie_file"` // 절대 경로 or base64 쿠키
		OutputDir  string `json:"output_dir"`
		Format     string `json:"format"` // mp4, mp3, best
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.URL == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "url 필요"})
		return
	}

	ytdlp := findYtDlp()
	if ytdlp == "" {
		writeJSON(w, 500, map[string]any{"success": false, "message": "yt-dlp 미설치. 'pip install yt-dlp' 필요"})
		return
	}

	outDir := req.OutputDir
	if outDir == "" {
		home, _ := os.UserHomeDir()
		outDir = filepath.Join(home, "Desktop", "osint_results")
	}
	os.MkdirAll(outDir, 0755)

	args := buildYtDlpArgs(req.URL, req.Platform, req.CookieFile, req.Format, outDir)

	cmd := exec.Command(ytdlp, args...)
	cmd.Dir = outDir

	out, err := cmd.CombinedOutput()
	output := string(out)

	if err != nil {
		writeJSON(w, 200, map[string]any{
			"success": false,
			"message": fmt.Sprintf("❌ 다운로드 실패: %s\n\n```\n%s\n```", err.Error(), limitStr(output, 500)),
		})
		return
	}

	// 다운로드된 파일 찾기
	files, _ := filepath.Glob(filepath.Join(outDir, "*.mp4"))
	files2, _ := filepath.Glob(filepath.Join(outDir, "*.mp3"))
	files = append(files, files2...)

	msg := fmt.Sprintf("✅ **%s** 다운로드 완료!\n\n저장 위치: `%s`", req.URL, outDir)
	if len(files) > 0 {
		msg += fmt.Sprintf("\n파일: `%s`", filepath.Base(files[len(files)-1]))
	}

	writeJSON(w, 200, map[string]any{
		"success":    true,
		"output_dir": outDir,
		"files":      files,
		"message":    msg,
	})
}

// POST /api/video/set-cookie
// 플랫폼별 쿠키 파일 저장 (Netscape 형식)
func handleVideoSetCookie(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Platform string `json:"platform"` // twitter, tiktok, youtube
		Cookie   string `json:"cookie"`   // Netscape cookie 내용 (텍스트)
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Platform == "" || req.Cookie == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "platform, cookie 필요"})
		return
	}

	cookiePath := videoCookiePath(req.Platform)
	os.MkdirAll(filepath.Dir(cookiePath), 0755)
	if err := os.WriteFile(cookiePath, []byte(req.Cookie), 0600); err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": msgT("쿠키 저장 실패: "+err.Error(), "Failed to save cookie: "+err.Error(), getLang(r))})
		return
	}

	writeJSON(w, 200, map[string]any{
		"success":  true,
		"platform": req.Platform,
		"path":     cookiePath,
		"message":  fmt.Sprintf("✅ **%s** 쿠키 저장 완료! 이제 비공개 영상도 다운로드 가능합니다.", req.Platform),
	})
}

// GET /api/video/cookie-status
func handleVideoCookieStatus(w http.ResponseWriter, r *http.Request) {
	platforms := []string{"twitter", "tiktok", "youtube"}
	status := map[string]any{}
	for _, p := range platforms {
		path := videoCookiePath(p)
		info, err := os.Stat(path)
		if err == nil {
			status[p] = map[string]any{
				"exists":   true,
				"path":     path,
				"modified": info.ModTime().Format("2006-01-02 15:04"),
			}
		} else {
			status[p] = map[string]any{"exists": false}
		}
	}
	writeJSON(w, 200, map[string]any{"success": true, "cookies": status})
}

// ── 플랫폼별 영상 검색 (yt-dlp + fallback) ────────────────────
func searchVideoPlatform(query, platform string, limit int) []map[string]string {
	ytdlp := findYtDlp()
	if ytdlp == "" {
		return searchVideoFallback(query, platform, limit)
	}

	var searchURL string
	switch platform {
	case "youtube":
		searchURL = fmt.Sprintf("ytsearch%d:%s", limit, query)
	case "tiktok":
		searchURL = fmt.Sprintf("https://www.tiktok.com/search?q=%s", urlEncode(query))
	case "twitter":
		searchURL = fmt.Sprintf("https://twitter.com/search?q=%s&f=video", urlEncode(query))
	default:
		return nil
	}

	cookieFile := videoCookiePath(platform)
	args := []string{
		"--no-download",
		"--print", "%(title)s\t%(webpage_url)s\t%(duration_string)s\t%(uploader)s",
		"--flat-playlist",
		fmt.Sprintf("--playlist-end=%d", limit),
		"--no-warnings",
	}
	if _, err := os.Stat(cookieFile); err == nil {
		args = append(args, "--cookies", cookieFile)
	}
	args = append(args, searchURL)

	cmd := exec.Command(ytdlp, args...)
	cmd.Env = append(os.Environ(), "PYTHONIOENCODING=utf-8")
	out, err := cmd.Output()
	if err != nil {
		return searchVideoFallback(query, platform, limit)
	}

	var items []map[string]string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 2 {
			continue
		}
		item := map[string]string{
			"title":    parts[0],
			"url":      parts[1],
			"platform": platform,
		}
		if len(parts) >= 3 {
			item["duration"] = parts[2]
		}
		if len(parts) >= 4 {
			item["channel"] = parts[3]
		}
		items = append(items, item)
	}
	return items
}

// Tavily 기반 영상 검색 fallback
func searchVideoFallback(query, platform string, limit int) []map[string]string {
	llmMu.RLock()
	tKey := llmTavilyKey
	llmMu.RUnlock()

	if tKey == "" {
		return nil
	}

	var domain string
	switch platform {
	case "youtube":
		domain = "youtube.com"
	case "tiktok":
		domain = "tiktok.com"
	case "twitter":
		domain = "twitter.com"
	default:
		return nil
	}

	r, ok := tavilySearchDomain(tKey, query, limit, domain)
	if !ok {
		return nil
	}
	return r.Items
}

// ── yt-dlp args 빌더 ──────────────────────────────────────────
func buildYtDlpArgs(videoURL, platform, cookieFile, format, outDir string) []string {
	args := []string{
		"-o", filepath.Join(outDir, "%(title)s.%(ext)s"),
		"--no-playlist",
		"--no-warnings",
	}

	// 쿠키 파일 (지정된 경우 우선, 없으면 저장된 플랫폼 쿠키)
	if cookieFile != "" {
		args = append(args, "--cookies", cookieFile)
	} else if platform != "" {
		saved := videoCookiePath(platform)
		if _, err := os.Stat(saved); err == nil {
			args = append(args, "--cookies", saved)
		}
	}

	// 포맷
	switch format {
	case "mp3":
		args = append(args, "-x", "--audio-format", "mp3")
	case "best":
		args = append(args, "-f", "bestvideo[ext=mp4]+bestaudio[ext=m4a]/best[ext=mp4]")
	default:
		args = append(args, "-f", "mp4/best[ext=mp4]/best")
	}

	// Anti-bot headers
	args = append(args,
		"--user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_4) AppleWebKit/605.1.15 Safari/605.1.15",
		"--add-header", "Accept-Language:ko-KR,ko;q=0.9",
	)

	args = append(args, videoURL)
	return args
}

func videoCookiePath(platform string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".nexus", "cookies", platform+".txt")
}

func findYtDlp() string {
	for _, candidate := range []string{
		"yt-dlp",
		"yt-dlp.exe",
		`C:\yt-dlp\yt-dlp.exe`,
		`C:\tools\yt-dlp\yt-dlp.exe`,
	} {
		if path, err := exec.LookPath(candidate); err == nil {
			return path
		}
	}
	// 인스톨러가 설치하는 경로: %APPDATA%\Nexus\yt-dlp.exe
	if appdata := os.Getenv("APPDATA"); appdata != "" {
		p := filepath.Join(appdata, "Nexus", "yt-dlp.exe")
		if fileExists(p) {
			return p
		}
	}
	// %LOCALAPPDATA%\Programs\yt-dlp
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		p := filepath.Join(local, "Programs", "yt-dlp", "yt-dlp.exe")
		if fileExists(p) {
			return p
		}
	}
	return ""
}

func limitStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// POST /api/video/ytdlp-info
// URL 정보만 조회 (다운로드 없음)
func handleVideoInfo(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL      string `json:"url"`
		Platform string `json:"platform"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.URL == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "url 필요"})
		return
	}

	ytdlp := findYtDlp()
	if ytdlp == "" {
		writeJSON(w, 500, map[string]any{"success": false, "message": "yt-dlp 미설치"})
		return
	}

	args := []string{
		"--dump-json",
		"--no-playlist",
		"--no-warnings",
	}
	cookieFile := videoCookiePath(req.Platform)
	if _, err := os.Stat(cookieFile); err == nil {
		args = append(args, "--cookies", cookieFile)
	}
	args = append(args, req.URL)

	cmd := exec.Command(ytdlp, args...)
	out, err := cmd.Output()
	if err != nil {
		writeJSON(w, 200, map[string]any{"success": false, "message": msgT("정보 조회 실패: "+err.Error(), "Info retrieval failed: "+err.Error(), getLang(r))})
		return
	}

	var info map[string]any
	json.Unmarshal(out, &info)

	title, _ := info["title"].(string)
	duration, _ := info["duration_string"].(string)
	uploader, _ := info["uploader"].(string)
	viewCount, _ := info["view_count"].(float64)
	thumbnail, _ := info["thumbnail"].(string)

	writeJSON(w, 200, map[string]any{
		"success":    true,
		"title":      title,
		"duration":   duration,
		"uploader":   uploader,
		"view_count": int(viewCount),
		"thumbnail":  thumbnail,
		"message":    fmt.Sprintf("📹 **%s** | %s | 조회수 %d회\n채널: %s", title, duration, int(viewCount), uploader),
		"timestamp":  time.Now().Format("2006-01-02 15:04"),
	})
}

// enrichYouTubeItems: Tavily YouTube 검색 결과에 yt-dlp 메타데이터 보강
// watch URL이 있는 항목에 대해 duration, uploader, view_count 추가
func enrichYouTubeItems(items []map[string]string) []map[string]string {
	ytdlp := findYtDlp()
	if ytdlp == "" {
		return items // yt-dlp 없으면 그대로 반환
	}

	result := make([]map[string]string, len(items))
	for i, item := range result {
		result[i] = item
	}
	copy(result, items)

	for i, item := range result {
		u := item["url"]
		if !isYouTubeWatchURL(u) {
			continue
		}
		// 타임아웃 짧게 — 딥서치 전체가 블로킹되지 않도록
		info := ytDlpQuickInfo(ytdlp, u, 8)
		if info == nil {
			continue
		}
		if v, ok := info["duration_string"].(string); ok && v != "" {
			result[i]["duration"] = v
		}
		if v, ok := info["uploader"].(string); ok && v != "" {
			result[i]["channel"] = v
		}
		if v, ok := info["view_count"].(float64); ok && v > 0 {
			result[i]["views"] = formatViewCount(int(v))
		}
		if v, ok := info["description"].(string); ok && v != "" && result[i]["content"] == "" {
			result[i]["content"] = limitStr(v, 200)
		}
	}
	return result
}

func isYouTubeWatchURL(u string) bool {
	return strings.Contains(u, "youtube.com/watch") || strings.Contains(u, "youtu.be/")
}

func ytDlpQuickInfo(ytdlp, videoURL string, timeoutSec int) map[string]any {
	import_exec := exec.Command(ytdlp,
		"--dump-json", "--no-playlist", "--no-warnings",
		"--socket-timeout", "5",
		videoURL,
	)
	import_exec.Env = append(os.Environ(), "PYTHONIOENCODING=utf-8")

	done := make(chan []byte, 1)
	go func() {
		out, _ := import_exec.Output()
		done <- out
	}()

	select {
	case out := <-done:
		var info map[string]any
		json.Unmarshal(out, &info)
		return info
	case <-time.After(time.Duration(timeoutSec) * time.Second):
		import_exec.Process.Kill()
		return nil
	}
}

func formatViewCount(n int) string {
	switch {
	case n >= 100000000:
		return fmt.Sprintf("%.0f억", float64(n)/100000000)
	case n >= 10000:
		return fmt.Sprintf("%.0f만", float64(n)/10000)
	case n >= 1000:
		return fmt.Sprintf("%.1f천", float64(n)/1000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

