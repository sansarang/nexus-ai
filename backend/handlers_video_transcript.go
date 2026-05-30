package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ══════════════════════════════════════════════════════════════
//  영상 Transcript 추출 + AI 요약
//  YouTube 자동자막 / TikTok / Twitter 등 yt-dlp 지원 플랫폼
// ══════════════════════════════════════════════════════════════

// POST /api/video/transcript
// body: { "url": "...", "platform": "youtube|tiktok|twitter", "lang": "ko", "summarize": true }
func handleVideoTranscript(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		URL       string `json:"url"`
		Platform  string `json:"platform"`
		Lang      string `json:"lang"`       // "ko", "en" — 자막 언어 우선순위
		Summarize bool   `json:"summarize"`  // true면 AI 요약 포함
	}
	tryDecodeBody(r, &req)
	if req.URL == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("url 필요", "url required", lang)})
		return
	}
	if req.Lang == "" {
		req.Lang = "ko"
	}

	ytdlp := findYtDlpOrInstall()
	if ytdlp == "" {
		writeJSON(w, 500, map[string]any{"success": false, "message": msgT("yt-dlp 설치 실패. 터미널에서 `pip install yt-dlp` 실행 후 재시도해주세요.", "yt-dlp installation failed. Run `pip install yt-dlp` in terminal and try again.", lang)})
		return
	}

	tmp, _ := os.MkdirTemp("", "nexus_transcript_*")
	defer os.RemoveAll(tmp)

	// yt-dlp로 자막 추출 (자동자막 포함)
	args := []string{
		"--skip-download",
		"--write-auto-sub",   // 자동 생성 자막 (YouTube)
		"--write-sub",        // 수동 자막
		"--sub-langs", fmt.Sprintf("%s,en,ja", req.Lang),
		"--sub-format", "vtt/srt/best",
		"--convert-subs", "srt",
		"-o", filepath.Join(tmp, "sub"),
		"--no-warnings",
		"--no-playlist",
	}
	// 저장된 쿠키 파일 자동 적용
	if req.Platform != "" {
		if cp := videoCookiePath(req.Platform); fileExists(cp) {
			args = append(args, "--cookies", cp)
		}
	}
	args = append(args, req.URL)

	cmd := exec.Command(ytdlp, args...)
	cmd.Env = append(os.Environ(), "PYTHONIOENCODING=utf-8")
	rawOut, err := cmd.CombinedOutput()

	// SRT 파일 찾기
	srtFiles, _ := filepath.Glob(filepath.Join(tmp, "*.srt"))
	vttFiles, _ := filepath.Glob(filepath.Join(tmp, "*.vtt"))
	allSubs := append(srtFiles, vttFiles...)

	var rawText string
	if len(allSubs) > 0 {
		// 언어 우선순위: req.Lang > en > 첫 번째
		chosen := allSubs[0]
		for _, f := range allSubs {
			base := filepath.Base(f)
			if strings.Contains(base, "."+req.Lang+".") {
				chosen = f
				break
			}
		}
		data, _ := os.ReadFile(chosen)
		rawText = parseSRT(string(data))
	} else if err != nil {
		// 자막 없음 → yt-dlp 메타데이터의 description으로 폴백
		rawText = extractDescFromYtDlpOutput(string(rawOut))
	}

	if rawText == "" {
		writeJSON(w, 200, map[string]any{
			"success":    false,
			"message":    msgT("이 영상에는 자막이 없거나 추출할 수 없습니다.\n\n```\n"+limitStr(string(rawOut), 300)+"\n```", "No subtitles found or could not be extracted.\n\n```\n"+limitStr(string(rawOut), 300)+"\n```", lang),
			"transcript": "",
		})
		return
	}

	// 자막 너무 길면 압축
	transcript := limitStr(rawText, 6000)

	resp := map[string]any{
		"success":    true,
		"transcript": transcript,
		"length":     len(rawText),
		"url":        req.URL,
		"timestamp":  time.Now().Format("2006-01-02 15:04"),
	}

	if req.Summarize {
		summary, tip := summarizeTranscript(transcript, req.Lang)
		resp["summary"] = summary
		resp["tip"] = tip
		resp["message"] = fmt.Sprintf("📝 **영상 요약 완료** | 자막 %d자 분석\n\n%s", len(rawText), summary)
	} else {
		resp["message"] = fmt.Sprintf("📝 **자막 추출 완료** | %d자", len(rawText))
	}

	writeJSON(w, 200, resp)
}

// POST /api/video/transcript-batch
// body: { "urls": ["...","..."], "lang": "ko" }
// 여러 영상을 한 번에 요약 → 통합 리포트 생성
func handleVideoTranscriptBatch(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		URLs  []string `json:"urls"`
		Lang  string   `json:"lang"`
		Topic string   `json:"topic"` // 리포트 주제 (선택)
	}
	tryDecodeBody(r, &req)
	if len(req.URLs) == 0 {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("urls 필요", "urls required", lang)})
		return
	}
	if req.Lang == "" {
		req.Lang = "ko"
	}
	if len(req.URLs) > 10 {
		req.URLs = req.URLs[:10] // 최대 10개
	}

	ytdlp := findYtDlpOrInstall()
	if ytdlp == "" {
		writeJSON(w, 500, map[string]any{"success": false, "message": msgT("yt-dlp 미설치", "yt-dlp not installed", lang)})
		return
	}

	type result struct {
		URL       string `json:"url"`
		Title     string `json:"title,omitempty"`
		Summary   string `json:"summary"`
		HasSub    bool   `json:"has_subtitle"`
	}

	var results []result
	var allSummaries []string

	for _, u := range req.URLs {
		r := result{URL: u}

		tmp, _ := os.MkdirTemp("", "nexus_batch_*")
		args := []string{
			"--skip-download", "--write-auto-sub", "--write-sub",
			"--sub-langs", req.Lang + ",en",
			"--sub-format", "vtt/srt/best",
			"--convert-subs", "srt",
			"-o", filepath.Join(tmp, "sub"),
			"--no-warnings", "--no-playlist",
			u,
		}
		cmd := exec.Command(ytdlp, args...)
		cmd.Env = append(os.Environ(), "PYTHONIOENCODING=utf-8")
		cmd.Run()

		srtFiles, _ := filepath.Glob(filepath.Join(tmp, "*.srt"))
		if len(srtFiles) > 0 {
			data, _ := os.ReadFile(srtFiles[0])
			text := limitStr(parseSRT(string(data)), 3000)
			if text != "" {
				r.HasSub = true
				summary, _ := summarizeTranscript(text, req.Lang)
				r.Summary = summary
				allSummaries = append(allSummaries, fmt.Sprintf("🎬 %s\n%s", u, summary))
			}
		}
		os.RemoveAll(tmp)

		if !r.HasSub {
			r.Summary = "자막 없음 또는 추출 실패"
		}
		results = append(results, r)
	}

	// 통합 리포트
	var report string
	if len(allSummaries) > 0 {
		topic := req.Topic
		if topic == "" {
			topic = "영상들"
		}
		prompt := fmt.Sprintf("다음 %d개 영상 요약을 종합해서 '%s'에 대한 핵심 인사이트를 한국어로 리포트로 작성해줘:\n\n%s",
			len(allSummaries), topic, strings.Join(allSummaries, "\n\n"))
		report, _, _ = callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 800, false)
	}

	writeJSON(w, 200, map[string]any{
		"success": true,
		"results": results,
		"report":  report,
		"count":   len(results),
		"message": fmt.Sprintf("📊 **%d개 영상 분석 완료**\n\n%s", len(results), report),
	})
}

// ── 헬퍼 ──────────────────────────────────────────────────────

// SRT/VTT 텍스트에서 타임코드 제거, 순수 자막 텍스트만 추출
func parseSRT(raw string) string {
	var lines []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// 숫자만인 줄 (자막 번호) 스킵
		if isNumericOnly(line) {
			continue
		}
		// 타임코드 줄 스킵 (00:00:01,234 --> 00:00:03,456)
		if strings.Contains(line, "-->") {
			continue
		}
		// VTT 헤더
		if strings.HasPrefix(line, "WEBVTT") || strings.HasPrefix(line, "Kind:") || strings.HasPrefix(line, "Language:") {
			continue
		}
		// HTML 태그 제거
		line = stripHTMLTags(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	// 연속 중복 제거 (자동자막 특성상 같은 문장 반복)
	return dedupLines(strings.Join(lines, " "))
}

func isNumericOnly(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func stripHTMLTags(s string) string {
	var out strings.Builder
	inTag := false
	for _, c := range s {
		if c == '<' {
			inTag = true
		} else if c == '>' {
			inTag = false
		} else if !inTag {
			out.WriteRune(c)
		}
	}
	return strings.TrimSpace(out.String())
}

// 연속으로 반복되는 구절 제거 (YouTube 자동자막 특성)
func dedupLines(text string) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return text
	}
	var result []string
	i := 0
	for i < len(words) {
		// 같은 단어가 3번 이상 연속이면 1개만 남김
		j := i + 1
		for j < len(words) && words[j] == words[i] {
			j++
		}
		result = append(result, words[i])
		if j-i <= 3 {
			for k := i + 1; k < j; k++ {
				result = append(result, words[k])
			}
		}
		i = j
	}
	return strings.Join(result, " ")
}

func extractDescFromYtDlpOutput(output string) string {
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "description") {
			return line
		}
	}
	return ""
}

func summarizeTranscript(transcript, lang string) (summary, tip string) {
	langInstr := "한국어"
	if lang == "en" {
		langInstr = "영어"
	}
	prompt := fmt.Sprintf(`다음 영상 자막을 분석해서 %s로 핵심 내용을 요약해줘.

**요약 형식:**
• 핵심 주제 1줄
• 주요 내용 3~5개 (불릿 포인트)
• 핵심 인사이트 또는 결론

자막:
%s`, langInstr, transcript)

	summary, _, _ = callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 600, false)

	// 팁/액션 아이템 추출
	tipPrompt := fmt.Sprintf("위 자막에서 실행 가능한 팁이나 액션 아이템 3개를 %s로 한 줄씩 뽑아줘. 없으면 '없음'이라고 해.", langInstr)
	tip, _, _ = callGroqWithFallback([]groqMsg{
		{Role: "user", Content: prompt},
		{Role: "assistant", Content: summary},
		{Role: "user", Content: tipPrompt},
	}, 300, false)
	return
}

// yt-dlp 찾기 + 없으면 자동 설치 시도
func findYtDlpOrInstall() string {
	if path := findYtDlp(); path != "" {
		return path
	}
	// pip / pip3 으로 자동 설치 시도 (Python이 설치된 Windows 환경)
	for _, pip := range []string{"pip", "pip3", "python -m pip"} {
		if _, err := exec.LookPath(pip); err == nil {
			cmd := exec.Command(pip, "install", "--quiet", "yt-dlp")
			cmd.Env = os.Environ()
			if err := cmd.Run(); err == nil {
				if path := findYtDlp(); path != "" {
					return path
				}
			}
		}
	}
	// winget으로 설치 시도
	if _, err := exec.LookPath("winget"); err == nil {
		exec.Command("winget", "install", "--id", "yt-dlp.yt-dlp", "-e", "--silent").Run()
		if path := findYtDlp(); path != "" {
			return path
		}
	}
	return ""
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
