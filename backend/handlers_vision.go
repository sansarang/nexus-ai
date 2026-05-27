//go:build windows

package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

// ──────────────────────────────────────────
// 화면 캡처 (Windows GDI API)
// ──────────────────────────────────────────

var (
	user32           = syscall.NewLazyDLL("user32.dll")
	gdi32            = syscall.NewLazyDLL("gdi32.dll")
	getSystemMetrics = user32.NewProc("GetSystemMetrics")
	getDC            = user32.NewProc("GetDC")
	releaseDC        = user32.NewProc("ReleaseDC")
	createCompatDC   = gdi32.NewProc("CreateCompatibleDC")
	createCompatBmp  = gdi32.NewProc("CreateCompatibleBitmap")
	selectObject     = gdi32.NewProc("SelectObject")
	bitBlt           = gdi32.NewProc("BitBlt")
	deleteObject     = gdi32.NewProc("DeleteObject")
	deleteDC         = gdi32.NewProc("DeleteDC")
	getBitmapBits    = gdi32.NewProc("GetBitmapBits")
)

const (
	SM_CXSCREEN = 0
	SM_CYSCREEN = 1
	SRCCOPY     = 0x00CC0020
)

// PowerShell로 스크린샷 캡처 (PNG) → Base64 반환
func captureScreenPowerShell() (string, int, int, error) {
	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("nexus_ss_%d.png", time.Now().UnixNano()))
	defer os.Remove(tmpFile)

	script := fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
$screen = [System.Windows.Forms.Screen]::PrimaryScreen.Bounds
$bmp = [System.Drawing.Bitmap]::new($screen.Width, $screen.Height)
$g = [System.Drawing.Graphics]::FromImage($bmp)
$g.CopyFromScreen($screen.Location, [System.Drawing.Point]::Empty, $screen.Size)
$g.Dispose()
$bmp.Save('%s', [System.Drawing.Imaging.ImageFormat]::Png)
$bmp.Dispose()
Write-Output "$($screen.Width)x$($screen.Height)"
`, tmpFile)

	out, err := execPS(script)
	if err != nil {
		return "", 0, 0, fmt.Errorf("스크린샷 캡처 실패: %w", err)
	}

	// 파일 읽기
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		return "", 0, 0, fmt.Errorf("스크린샷 파일 읽기 실패: %w", err)
	}

	// 크기 파싱
	dims := strings.TrimSpace(string(out))
	width, height := 1920, 1080
	fmt.Sscanf(dims, "%dx%d", &width, &height)

	b64 := base64.StdEncoding.EncodeToString(data)
	return b64, width, height, nil
}

// 화면 크기 가져오기
func getScreenSize() (width, height int) {
	w, _, _ := getSystemMetrics.Call(SM_CXSCREEN)
	h, _, _ := getSystemMetrics.Call(SM_CYSCREEN)
	return int(w), int(h)
}

// Windows OCR API (PowerShell)
func runWindowsOCR(imagePath string) (string, error) {
	script := fmt.Sprintf(`
Add-Type -AssemblyName System.Runtime.WindowsRuntime
$null = [Windows.Storage.StorageFile,Windows.Storage,ContentType=WindowsRuntime]
$null = [Windows.Media.Ocr.OcrEngine,Windows.Foundation,ContentType=WindowsRuntime]

function Await($WinRTTask,$ResultType) {
    $asTask = [System.WindowsRuntimeSystemExtensions]::AsTask($WinRTTask)
    $asTask.Wait()
    $asTask.Result
}

try {
    $engine = [Windows.Media.Ocr.OcrEngine]::TryCreateFromUserProfileLanguages()
    $file = Await ([Windows.Storage.StorageFile]::GetFileFromPathAsync('%s')) ([Windows.Storage.StorageFile])
    $stream = Await ($file.OpenAsync([Windows.Storage.FileAccessMode]::Read)) ([Windows.Storage.Streams.IRandomAccessStream])
    $decoder = Await ([Windows.Graphics.Imaging.BitmapDecoder]::CreateAsync($stream)) ([Windows.Graphics.Imaging.BitmapDecoder])
    $bitmap = Await ($decoder.GetSoftwareBitmapAsync()) ([Windows.Graphics.Imaging.SoftwareBitmap])
    $result = Await ($engine.RecognizeAsync($bitmap)) ([Windows.Media.Ocr.OcrResult])
    $result.Text
} catch {
    "OCR 실패: $_"
}
`, imagePath)

	out, err := execPS(script)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// ──────────────────────────────────────────
// 스크린샷 캡처 핸들러
// ──────────────────────────────────────────

func handleScreenshot(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Region  string `json:"region"`  // full | active_window
		WithOCR bool   `json:"with_ocr"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	b64, width, height, err := captureScreenPowerShell()
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}

	result := map[string]any{
		"success":  true,
		"base64":   b64,
		"width":    width,
		"height":   height,
		"mime":     "image/png",
		"captured": time.Now().Format("2006-01-02 15:04:05"),
	}

	// OCR 요청 시
	if req.WithOCR {
		tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("nexus_ocr_%d.png", time.Now().UnixNano()))
		defer os.Remove(tmpFile)

		imgData, decErr := base64.StdEncoding.DecodeString(b64)
		if decErr == nil {
			os.WriteFile(tmpFile, imgData, 0644)
			ocrText, ocrErr := runWindowsOCR(tmpFile)
			if ocrErr == nil {
				result["ocr_text"] = ocrText
			}
		}
	}

	json200(w, result)
}

// ──────────────────────────────────────────
// OCR — 클립보드 이미지 텍스트 추출
// ──────────────────────────────────────────

func handleOCRClipboard(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("nexus_clip_%d.png", time.Now().UnixNano()))
	defer os.Remove(tmpFile)

	// 클립보드 이미지를 파일로 저장
	script := fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
$img = [System.Windows.Forms.Clipboard]::GetImage()
if ($img -ne $null) {
    $img.Save('%s')
    Write-Output "OK"
} else {
    Write-Output "NO_IMAGE"
}
`, tmpFile)

	out, _ := execPS(script)
	if strings.TrimSpace(string(out)) != "OK" {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("클립보드에 이미지가 없어요", "No image in clipboard", lang)})
		return
	}

	ocrText, err := runWindowsOCR(tmpFile)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": msgT("OCR 실패: ", "OCR failed: ", lang) + err.Error()})
		return
	}

	json200(w, map[string]any{
		"success": true,
		"text":    ocrText,
		"message": fmt.Sprintf(msgT("텍스트 %d자 추출 완료", "%d characters extracted", lang), len(ocrText)),
	})
}

// ──────────────────────────────────────────
// 활성 창 정보
// ──────────────────────────────────────────

var (
	getForegroundWindow = user32.NewProc("GetForegroundWindow")
	getWindowTextW      = user32.NewProc("GetWindowTextW")
)

func getActiveWindowTitle() string {
	hwnd, _, _ := getForegroundWindow.Call()
	if hwnd == 0 {
		return ""
	}
	buf := make([]uint16, 256)
	getWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), 256)
	return syscall.UTF16ToString(buf)
}

func handleActiveWindow(w http.ResponseWriter, r *http.Request) {
	title := getActiveWindowTitle()

	// 활성 창 스크린샷
	out, _ := newHiddenCmd("powershell", "-NoProfile", "-Command",
		`Add-Type -AssemblyName System.Windows.Forms; [System.Windows.Forms.Screen]::PrimaryScreen.Bounds | Select-Object Width, Height | ConvertTo-Json -Compress`).Output()

	json200(w, map[string]any{
		"title":       title,
		"screen_info": string(out),
		"timestamp":   time.Now().Format("2006-01-02 15:04:05"),
	})
}

// ──────────────────────────────────────────
// Deep Search (파일 내용 기반)
// ──────────────────────────────────────────

// DeepSearchResult — types.go 에서 정의됨

func handleDeepSearch(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		Query      string `json:"query"`
		SearchIn   string `json:"search_in"` // content | filename | both
		Folder     string `json:"folder"`
		FileType   string `json:"file_type"`
		MaxResults int    `json:"max_results"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.Query == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("검색어를 입력해주세요", "Please enter a search term", lang)})
		return
	}
	if req.SearchIn == "" {
		req.SearchIn = "both"
	}
	if req.MaxResults == 0 {
		req.MaxResults = 20
	}
	if req.Folder == "" {
		req.Folder, _ = os.UserHomeDir()
	}

	// 검색 대상 확장자
	searchExts := map[string]bool{
		".txt": true, ".md": true, ".csv": true, ".log": true,
		".docx": true, ".doc": true, ".xlsx": true, ".xls": true,
		".pdf": true, ".hwp": true, ".json": true, ".xml": true,
	}
	if req.FileType != "" && req.FileType != "any" {
		ext := "." + strings.TrimPrefix(strings.ToLower(req.FileType), ".")
		searchExts = map[string]bool{ext: true}
	}

	queryTerms := strings.Fields(strings.ToLower(req.Query))
	var results []DeepSearchResult

	filepath.Walk(req.Folder, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || len(results) >= req.MaxResults {
			return nil
		}
		// 시스템 폴더 제외
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

		// 파일명 매칭
		if req.SearchIn != "content" {
			for _, term := range queryTerms {
				if strings.Contains(nameLow, term) {
					score += 30
				}
			}
		}

		// 내용 매칭 (10MB 이하)
		if req.SearchIn != "filename" && info.Size() < 10<<20 {
			text, err := extractDocumentText(p)
			if err == nil {
				textLow := strings.ToLower(text)
				for _, term := range queryTerms {
					count := strings.Count(textLow, term)
					if count > 0 {
						score += min(count*10, 40)
						// 첫 번째 매칭 위치 스니펫
						if snippet == "" {
							idx := strings.Index(textLow, term)
							if idx >= 0 {
								start := idx - 40
								if start < 0 {
									start = 0
								}
								end := idx + len(term) + 80
								if end > len(text) {
									end = len(text)
								}
								snippet = "..." + text[start:end] + "..."
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

	// 점수 기준 정렬
	sortByScore(results)
	if len(results) > req.MaxResults {
		results = results[:req.MaxResults]
	}

	json200(w, map[string]any{
		"results": results,
		"total":   len(results),
		"query":   req.Query,
		"message": fmt.Sprintf(msgT("'%s' 심층 검색 결과: %d개", "Deep search results for '%s': %d", lang), req.Query, len(results)),
	})
}

func sortByScore(results []DeepSearchResult) {
	for i := 1; i < len(results); i++ {
		for j := 0; j < len(results)-i; j++ {
			if results[j].Score < results[j+1].Score {
				results[j], results[j+1] = results[j+1], results[j]
			}
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
