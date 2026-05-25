//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var organizeExtWin = map[string]string{
	".jpg": "사진", ".jpeg": "사진", ".png": "사진", ".gif": "사진", ".webp": "사진", ".heic": "사진", ".bmp": "사진",
	".mp4": "동영상", ".mkv": "동영상", ".avi": "동영상", ".mov": "동영상", ".wmv": "동영상",
	".mp3": "음악", ".wav": "음악", ".flac": "음악", ".aac": "음악", ".wma": "음악",
	".pdf": "문서", ".doc": "문서", ".docx": "문서", ".xls": "문서", ".xlsx": "문서",
	".ppt": "문서", ".pptx": "문서", ".txt": "문서", ".md": "문서", ".hwp": "문서",
	".zip": "압축파일", ".rar": "압축파일", ".7z": "압축파일", ".tar": "압축파일", ".gz": "압축파일",
	".exe": "프로그램", ".msi": "프로그램", ".bat": "프로그램",
}

func handleFilesSearch(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		Query   string `json:"query"`
		Path    string `json:"path"`
		Type    string `json:"type"`
		MaxDays int    `json:"max_days"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Path == "" {
		home, _ := os.UserHomeDir()
		req.Path = home
	}
	if req.MaxDays == 0 {
		req.MaxDays = 30
	}
	extMap := map[string][]string{
		"pdf":   {".pdf"},
		"doc":   {".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".hwp"},
		"image": {".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp"},
		"video": {".mp4", ".mkv", ".avi", ".mov", ".wmv"},
		"any":   {},
	}
	allowedExts := extMap[req.Type]
	cutoff := time.Now().AddDate(0, 0, -req.MaxDays)
	queryLow := strings.ToLower(req.Query)

	type FileResult struct {
		Name    string  `json:"name"`
		Path    string  `json:"path"`
		SizeMB  float64 `json:"size_mb"`
		ModTime string  `json:"mod_time"`
	}
	var results []FileResult

	filepath.Walk(req.Path, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || len(results) >= 50 {
			return nil
		}
		if info.ModTime().Before(cutoff) {
			return nil
		}
		if queryLow != "" && !strings.Contains(strings.ToLower(info.Name()), queryLow) {
			return nil
		}
		if len(allowedExts) > 0 {
			ext := strings.ToLower(filepath.Ext(info.Name()))
			ok := false
			for _, e := range allowedExts {
				if e == ext {
					ok = true
					break
				}
			}
			if !ok {
				return nil
			}
		}
		results = append(results, FileResult{
			Name:    info.Name(),
			Path:    p,
			SizeMB:  float64(info.Size()) / (1 << 20),
			ModTime: info.ModTime().Format("2006-01-02 15:04"),
		})
		return nil
	})

	json200(w, map[string]any{
		"results": results,
		"total":   len(results),
		"message": fmt.Sprintf(msgT("'%s' 검색 결과: %d개", "'%s' search results: %d", lang), req.Query, len(results)),
	})
}

func handleFilesOrganize(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		Path string `json:"path"`
		Mode string `json:"mode"` // type | date
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Path == "" {
		home, _ := os.UserHomeDir()
		req.Path = filepath.Join(home, "Downloads")
	}
	if req.Mode == "" {
		req.Mode = "type"
	}
	entries, err := os.ReadDir(req.Path)
	if err != nil {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("폴더를 읽을 수 없어요", "Cannot read folder", lang)})
		return
	}
	moved, skipped := 0, 0
	for _, e := range entries {
		if e.IsDir() {
			skipped++
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		var subDir string
		if req.Mode == "date" {
			subDir = info.ModTime().Format("2006-01")
		} else {
			ext := strings.ToLower(filepath.Ext(e.Name()))
			cat, ok := organizeExtWin[ext]
			if !ok {
				cat = "기타"
			}
			subDir = cat
		}
		dst := filepath.Join(req.Path, subDir)
		os.MkdirAll(dst, 0755)
		src := filepath.Join(req.Path, e.Name())
		if err := os.Rename(src, filepath.Join(dst, e.Name())); err == nil {
			moved++
		}
	}
	json200(w, map[string]any{
		"success": true,
		"moved":   moved,
		"skipped": skipped,
		"message": fmt.Sprintf(msgT("%d개 파일 정리 완료 📁", "%d files organized 📁", lang), moved),
	})
}

func handleFilesDuplicates(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		Path string `json:"path"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Path == "" {
		home, _ := os.UserHomeDir()
		req.Path = filepath.Join(home, "Downloads")
	}
	type FileKey struct {
		Name string
		Size int64
	}
	seen := map[FileKey][]string{}
	filepath.Walk(req.Path, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		key := FileKey{Name: info.Name(), Size: info.Size()}
		seen[key] = append(seen[key], p)
		return nil
	})
	type DupGroup struct {
		Name   string   `json:"name"`
		SizeMB float64  `json:"size_mb"`
		Paths  []string `json:"paths"`
		Count  int      `json:"count"`
	}
	var groups []DupGroup
	var totalWaste int64
	for key, paths := range seen {
		if len(paths) > 1 {
			sort.Strings(paths)
			totalWaste += key.Size * int64(len(paths)-1)
			groups = append(groups, DupGroup{
				Name:   key.Name,
				SizeMB: float64(key.Size) / (1 << 20),
				Paths:  paths,
				Count:  len(paths),
			})
		}
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].SizeMB > groups[j].SizeMB })
	if len(groups) > 20 {
		groups = groups[:20]
	}
	waste := fmt.Sprintf("%.1f MB", float64(totalWaste)/(1<<20))
	json200(w, map[string]any{
		"groups":       groups,
		"total_groups": len(groups),
		"waste_mb":     float64(totalWaste) / (1 << 20),
		"waste":        waste,
		"message":      fmt.Sprintf(msgT("중복 파일 %d그룹 발견, 낭비 공간 %s", "Found %d duplicate groups, wasted space %s", lang), len(groups), waste),
	})
}

// ──────────────────────────────────────────
// POST /api/files/move — 파일/폴더 이동 또는 이름 변경
// ──────────────────────────────────────────

func handleFileMove(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		Src  string `json:"src"`  // 원본 경로 (파일 또는 폴더)
		Dst  string `json:"dst"`  // 대상 경로 또는 폴더
		Name string `json:"name"` // (선택) 이동 후 파일명
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.Src == "" || req.Dst == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("src, dst 필요", "src and dst required", lang)})
		return
	}

	// 홈 디렉토리 확장
	home, _ := os.UserHomeDir()
	expand := func(p string) string {
		if strings.HasPrefix(p, "~/") { return filepath.Join(home, p[2:]) }
		if strings.EqualFold(p, "desktop") || strings.EqualFold(p, "바탕화면") { return filepath.Join(home, "Desktop") }
		if strings.EqualFold(p, "downloads") || strings.EqualFold(p, "다운로드") { return filepath.Join(home, "Downloads") }
		if strings.EqualFold(p, "documents") || strings.EqualFold(p, "문서") { return filepath.Join(home, "Documents") }
		return p
	}
	src := expand(req.Src)
	dst := expand(req.Dst)

	// src 존재 확인
	srcInfo, err := os.Stat(src)
	if err != nil {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("원본 경로를 찾을 수 없어요: "+src, "Source not found: "+src, lang)})
		return
	}

	// dst가 폴더이면 src 파일명으로 결합
	dstInfo, _ := os.Stat(dst)
	if dstInfo != nil && dstInfo.IsDir() {
		name := req.Name
		if name == "" { name = srcInfo.Name() }
		dst = filepath.Join(dst, name)
	}

	if err := os.Rename(src, dst); err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": msgT("이동 실패: "+err.Error(), "Move failed: "+err.Error(), lang)})
		return
	}

	json200(w, map[string]any{
		"success": true,
		"src":     src,
		"dst":     dst,
		"message": fmt.Sprintf(msgT("✅ '%s' → '%s' 이동 완료", "✅ Moved '%s' → '%s'", lang), filepath.Base(src), dst),
	})
}

// ──────────────────────────────────────────
// POST /api/files/metadata — 폴더 내 파일 메타데이터 수집 (엑셀 생성용)
// ──────────────────────────────────────────

func handleFilesMetadata(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		Path      string `json:"path"`
		Recursive bool   `json:"recursive"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	home, _ := os.UserHomeDir()
	if req.Path == "" { req.Path = filepath.Join(home, "Desktop") }

	type FileMeta struct {
		Name     string  `json:"name"`
		Path     string  `json:"path"`
		SizeMB   float64 `json:"size_mb"`
		Modified string  `json:"modified"`
		Created  string  `json:"created"`
		Ext      string  `json:"ext"`
	}

	var files []FileMeta
	deadline := time.Now().Add(10 * time.Second)

	walk := func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || len(files) >= 500 || time.Now().After(deadline) { return nil }
		files = append(files, FileMeta{
			Name:     info.Name(),
			Path:     p,
			SizeMB:   float64(info.Size()) / (1 << 20),
			Modified: info.ModTime().Format("2006-01-02 15:04:05"),
			Created:  info.ModTime().Format("2006-01-02"), // Windows: ModTime으로 대체
			Ext:      strings.ToLower(filepath.Ext(info.Name())),
		})
		if !req.Recursive { return filepath.SkipDir }
		return nil
	}

	if req.Recursive {
		filepath.Walk(req.Path, walk)
	} else {
		entries, _ := os.ReadDir(req.Path)
		for _, e := range entries {
			if e.IsDir() { continue }
			info, err := e.Info()
			if err != nil { continue }
			walk(filepath.Join(req.Path, e.Name()), info, nil)
		}
	}

	json200(w, map[string]any{
		"files":   files,
		"count":   len(files),
		"message": fmt.Sprintf(msgT("파일 %d개 메타데이터 수집 완료", "Collected metadata for %d files", lang), len(files)),
	})
}
