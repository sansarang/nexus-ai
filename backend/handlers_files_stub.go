//go:build !windows

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

// ── 파일 검색 (cross-platform) ────────────────────────────────

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
		"doc":   {".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx"},
		"image": {".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp"},
		"video": {".mp4", ".mkv", ".avi", ".mov"},
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
			Name: info.Name(), Path: p,
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

// ── 폴더 자동 정리 (cross-platform) ──────────────────────────

var organizeExtMac = map[string]string{
	".jpg": "사진", ".jpeg": "사진", ".png": "사진", ".gif": "사진", ".webp": "사진", ".heic": "사진",
	".mp4": "동영상", ".mkv": "동영상", ".avi": "동영상", ".mov": "동영상",
	".mp3": "음악", ".wav": "음악", ".flac": "음악", ".aac": "음악",
	".pdf": "문서", ".doc": "문서", ".docx": "문서", ".xls": "문서", ".xlsx": "문서",
	".ppt": "문서", ".pptx": "문서", ".txt": "문서", ".md": "문서",
	".zip": "압축파일", ".rar": "압축파일", ".7z": "압축파일", ".tar": "압축파일", ".gz": "압축파일",
	".dmg": "프로그램", ".pkg": "프로그램", ".app": "프로그램",
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
			cat, ok := organizeExtMac[ext]
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

// ── 중복 파일 탐지 (cross-platform) ──────────────────────────

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
				Name: key.Name, SizeMB: float64(key.Size) / (1 << 20),
				Paths: paths, Count: len(paths),
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

func handleFileMove(w http.ResponseWriter, _ *http.Request) {
	json200(w, map[string]any{"success": false, "message": "Windows 전용 기능입니다"})
}

func handleFilesMetadata(w http.ResponseWriter, _ *http.Request) {
	json200(w, map[string]any{"files": []any{}, "count": 0, "message": "Windows 전용 기능입니다"})
}
