//go:build windows

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"time"
)

// recallEnabled: 1=활성, 0=비활성 (원자적 플래그 — 경쟁조건 없음)
var recallEnabled int32 = 1

type recallConfig struct {
	Enabled       bool `json:"enabled"`
	IntervalSec   int  `json:"interval_sec"`    // 기본 300초 (5분)
	MaxEntries    int  `json:"max_entries"`     // 기본 100
}

func recallConfigPath() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = os.TempDir()
	}
	return filepath.Join(appData, "Nexus", "recall_config.json")
}

func loadRecallConfig() recallConfig {
	cfg := recallConfig{Enabled: true, IntervalSec: 300, MaxEntries: 100}
	data, err := os.ReadFile(recallConfigPath())
	if err == nil {
		json.Unmarshal(data, &cfg)
	}
	return cfg
}

func saveRecallConfig(cfg recallConfig) {
	data, _ := json.Marshal(cfg)
	os.WriteFile(recallConfigPath(), data, 0644)
}

// GET/POST /api/recall/config
func handleRecallConfig(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	if r.Method == http.MethodGet {
		cfg := loadRecallConfig()
		cfg.Enabled = atomic.LoadInt32(&recallEnabled) == 1
		json200(w, map[string]any{"success": true, "config": cfg})
		return
	}
	// POST
	var cfg recallConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeJSON(w, 400, map[string]any{"success": false, "message": "invalid body"})
		return
	}
	if cfg.IntervalSec < 60 {
		cfg.IntervalSec = 300
	}
	if cfg.MaxEntries < 10 {
		cfg.MaxEntries = 100
	}
	saveRecallConfig(cfg)
	if cfg.Enabled {
		atomic.StoreInt32(&recallEnabled, 1)
	} else {
		atomic.StoreInt32(&recallEnabled, 0)
	}
	json200(w, map[string]any{
		"success": true,
		"message": msgT("Recall 설정 저장됨", "Recall config saved", lang),
		"config":  cfg,
	})
}

// ══════════════════════════════════════════════════════════════════
//  Windows Recall — 주기적 스크린샷 + OCR 인덱싱 + 키워드 검색
// ══════════════════════════════════════════════════════════════════

type RecallEntry struct {
	Timestamp string `json:"timestamp"`
	OcrText   string `json:"ocr_text"`
	FilePath  string `json:"file_path"`
}

func recallDir() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = os.TempDir()
	}
	dir := filepath.Join(appData, "Nexus", "recall")
	os.MkdirAll(dir, 0755)
	return dir
}

// POST /api/recall/capture
func handleRecallCapture(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	ts := time.Now().Format("20060102_150405")
	dir := recallDir()
	imgPath := filepath.Join(dir, "screen_"+ts+".png")
	jsonPath := filepath.Join(dir, "recall_"+ts+".json")

	// PowerShell: 스크린샷 캡처 + Windows OCR
	script := `
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
$screen = [System.Windows.Forms.Screen]::PrimaryScreen.Bounds
$bitmap = New-Object System.Drawing.Bitmap($screen.Width, $screen.Height)
$graphics = [System.Drawing.Graphics]::FromImage($bitmap)
$graphics.CopyFromScreen($screen.Location, [System.Drawing.Point]::Empty, $screen.Size)
$bitmap.Save('` + imgPath + `', [System.Drawing.Imaging.ImageFormat]::Png)
$graphics.Dispose()
$bitmap.Dispose()

# Windows OCR
try {
    Add-Type -AssemblyName Windows.Foundation
    $null = [Windows.Storage.StorageFile,Windows.Storage,ContentType=WindowsRuntime]
    $null = [Windows.Media.Ocr.OcrEngine,Windows.Foundation,ContentType=WindowsRuntime]
    $null = [Windows.Graphics.Imaging.BitmapDecoder,Windows.Foundation,ContentType=WindowsRuntime]
    $file = [Windows.Storage.StorageFile]::GetFileFromPathAsync('` + imgPath + `').GetAwaiter().GetResult()
    $stream = $file.OpenAsync([Windows.Storage.FileAccessMode]::Read).GetAwaiter().GetResult()
    $decoder = [Windows.Graphics.Imaging.BitmapDecoder]::CreateAsync($stream).GetAwaiter().GetResult()
    $bitmap2 = $decoder.GetSoftwareBitmapAsync().GetAwaiter().GetResult()
    $engine = [Windows.Media.Ocr.OcrEngine]::TryCreateFromUserProfileLanguages()
    if ($engine) {
        $result = $engine.RecognizeAsync($bitmap2).GetAwaiter().GetResult()
        Write-Output $result.Text
    }
} catch {
    Write-Output ""
}
`
	out, _ := execPS(script)
	ocrText := strings.TrimSpace(string(out))

	entry := RecallEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		OcrText:   ocrText,
		FilePath:  imgPath,
	}
	data, _ := json.Marshal(entry)
	os.WriteFile(jsonPath, data, 0644)

	// 최대 500개 유지
	pruneRecallEntries(500)

	json200(w, map[string]interface{}{
		"success":    true,
		"timestamp":  entry.Timestamp,
		"file_path":  imgPath,
		"ocr_length": len(ocrText),
		"message":    msgT("화면 캡처 및 OCR 완료", "Screen capture and OCR complete", lang),
	})
}

// POST /api/recall/search — body: {query: string}
func handleRecallSearch(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		Query string `json:"query"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Query == "" {
		json200(w, map[string]interface{}{"success": false, "message": msgT("query가 필요해요", "query is required", lang)})
		return
	}

	dir := recallDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		json200(w, map[string]interface{}{"success": false, "message": msgT("recall 디렉토리 접근 실패", "Failed to access recall directory", lang)})
		return
	}

	type SearchResult struct {
		Timestamp string `json:"timestamp"`
		Snippet   string `json:"snippet"`
		FilePath  string `json:"file_path"`
	}

	var results []SearchResult
	queryLower := strings.ToLower(req.Query)

	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), "recall_") || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var entry RecallEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			continue
		}
		idx := strings.Index(strings.ToLower(entry.OcrText), queryLower)
		if idx < 0 {
			continue
		}
		// 매치 주변 50자 스니펫
		start := idx - 25
		if start < 0 {
			start = 0
		}
		end := idx + len(req.Query) + 25
		if end > len(entry.OcrText) {
			end = len(entry.OcrText)
		}
		snippet := entry.OcrText[start:end]
		results = append(results, SearchResult{
			Timestamp: entry.Timestamp,
			Snippet:   snippet,
			FilePath:  entry.FilePath,
		})
		if len(results) >= 10 {
			break
		}
	}

	json200(w, map[string]interface{}{
		"success": true,
		"results": results,
		"total":   len(results),
		"message": fmt.Sprintf(msgT("'%s' 검색 결과 %d건", "'%s' search results: %d", lang), req.Query, len(results)),
	})
}

func pruneRecallEntries(max int) {
	dir := recallDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	var jsonFiles []os.DirEntry
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "recall_") && strings.HasSuffix(e.Name(), ".json") {
			jsonFiles = append(jsonFiles, e)
		}
	}

	if len(jsonFiles) <= max {
		return
	}

	// 이름 기준 정렬(타임스탬프 포함)
	sort.Slice(jsonFiles, func(i, j int) bool {
		return jsonFiles[i].Name() < jsonFiles[j].Name()
	})

	toDelete := jsonFiles[:len(jsonFiles)-max]
	for _, f := range toDelete {
		name := f.Name()
		os.Remove(filepath.Join(dir, name))
		// 대응하는 PNG도 삭제
		imgName := strings.Replace(name, "recall_", "screen_", 1)
		imgName = strings.Replace(imgName, ".json", ".png", 1)
		os.Remove(filepath.Join(dir, imgName))
	}
}

// startRecallCollector — 설정 파일 기반 주기 자동 캡처 (기본 5분)
func startRecallCollector() {
	// 초기 설정 로드
	cfg := loadRecallConfig()
	if !cfg.Enabled {
		atomic.StoreInt32(&recallEnabled, 0)
	}
	interval := time.Duration(cfg.IntervalSec) * time.Second
	if interval < 60*time.Second {
		interval = 5 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		// 비활성화된 경우 skip
		if atomic.LoadInt32(&recallEnabled) == 0 {
			continue
		}
		ts := time.Now().Format("20060102_150405")
		dir := recallDir()
		imgPath := filepath.Join(dir, "screen_"+ts+".png")
		jsonPath := filepath.Join(dir, "recall_"+ts+".json")

		script := `
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
$screen = [System.Windows.Forms.Screen]::PrimaryScreen.Bounds
$bitmap = New-Object System.Drawing.Bitmap($screen.Width, $screen.Height)
$graphics = [System.Drawing.Graphics]::FromImage($bitmap)
$graphics.CopyFromScreen($screen.Location, [System.Drawing.Point]::Empty, $screen.Size)
$bitmap.Save('` + imgPath + `', [System.Drawing.Imaging.ImageFormat]::Png)
$graphics.Dispose()
$bitmap.Dispose()
try {
    Add-Type -AssemblyName Windows.Foundation
    $null = [Windows.Storage.StorageFile,Windows.Storage,ContentType=WindowsRuntime]
    $null = [Windows.Media.Ocr.OcrEngine,Windows.Foundation,ContentType=WindowsRuntime]
    $null = [Windows.Graphics.Imaging.BitmapDecoder,Windows.Foundation,ContentType=WindowsRuntime]
    $file = [Windows.Storage.StorageFile]::GetFileFromPathAsync('` + imgPath + `').GetAwaiter().GetResult()
    $stream = $file.OpenAsync([Windows.Storage.FileAccessMode]::Read).GetAwaiter().GetResult()
    $decoder = [Windows.Graphics.Imaging.BitmapDecoder]::CreateAsync($stream).GetAwaiter().GetResult()
    $bitmap2 = $decoder.GetSoftwareBitmapAsync().GetAwaiter().GetResult()
    $engine = [Windows.Media.Ocr.OcrEngine]::TryCreateFromUserProfileLanguages()
    if ($engine) {
        $result = $engine.RecognizeAsync($bitmap2).GetAwaiter().GetResult()
        Write-Output $result.Text
    }
} catch {
    Write-Output ""
}
`
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		out, _ := newHiddenCmdCtx(ctx, "powershell", "-NoProfile", "-Command", script).Output()
		cancel()
		ocrText := strings.TrimSpace(string(out))

		entry := RecallEntry{
			Timestamp: time.Now().Format(time.RFC3339),
			OcrText:   ocrText,
			FilePath:  imgPath,
		}
		data, _ := json.Marshal(entry)
		os.WriteFile(jsonPath, data, 0644)
		maxE := loadRecallConfig().MaxEntries
		if maxE < 10 { maxE = 100 }
		pruneRecallEntries(maxE) // 설정값 × ~2MB = 디스크 보호
	}
}
