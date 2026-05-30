//go:build windows

package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ══════════════════════════════════════════════════════════════
//  파일 시스템 조작 핸들러
//  - 폴더 정리 (확장자별 분류)
//  - 중복 파일 탐지
//  - 파일 검색 (이름/날짜/크기)
//  - 대용량 파일 탐지
// ══════════════════════════════════════════════════════════════

// fileCategory: 확장자 → 카테고리 분류
func fileCategory(ext string) string {
	ext = strings.ToLower(ext)
	images := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".bmp": true, ".webp": true, ".heic": true, ".svg": true, ".raw": true, ".tiff": true}
	videos := map[string]bool{".mp4": true, ".avi": true, ".mkv": true, ".mov": true, ".wmv": true, ".flv": true, ".webm": true, ".m4v": true}
	audio  := map[string]bool{".mp3": true, ".wav": true, ".flac": true, ".aac": true, ".m4a": true, ".ogg": true, ".wma": true}
	docs   := map[string]bool{".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true, ".ppt": true, ".pptx": true, ".txt": true, ".md": true, ".csv": true, ".hwp": true}
	code   := map[string]bool{".go": true, ".py": true, ".js": true, ".ts": true, ".java": true, ".c": true, ".cpp": true, ".rs": true, ".html": true, ".css": true, ".json": true, ".yaml": true, ".sh": true}
	compress := map[string]bool{".zip": true, ".rar": true, ".7z": true, ".tar": true, ".gz": true}

	switch {
	case images[ext]:   return "이미지"
	case videos[ext]:   return "동영상"
	case audio[ext]:    return "음악"
	case docs[ext]:     return "문서"
	case code[ext]:     return "코드"
	case compress[ext]: return "압축파일"
	}
	return "기타"
}

// ── POST /api/file/organize ────────────────────────────────

type organizeResult struct {
	Moved   []map[string]string `json:"moved"`
	Skipped []string            `json:"skipped"`
	Total   int                 `json:"total"`
	Message string              `json:"message"`
}

func handleFileOrganize(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Folder  string `json:"folder"`   // 정리할 폴더 경로
		DryRun  bool   `json:"dry_run"`  // true면 실제 이동 안 함
		Message string `json:"message"`
	}
	tryDecodeBody(r, &req)
	// 기본값: 바탕화면 or 다운로드
	if req.Folder == "" {
		home, _ := os.UserHomeDir()
		// 메시지에서 폴더 힌트 추출
		msg := strings.ToLower(req.Message)
		switch {
		case strings.Contains(msg, "다운로드") || strings.Contains(msg, "download"):
			req.Folder = filepath.Join(home, "Downloads")
		case strings.Contains(msg, "문서") || strings.Contains(msg, "documents"):
			req.Folder = filepath.Join(home, "Documents")
		default:
			req.Folder = filepath.Join(home, "Desktop")
		}
	}

	entries, err := os.ReadDir(req.Folder)
	if err != nil {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("폴더를 열 수 없습니다: "+err.Error(), "Cannot open folder: "+err.Error(), getLang(r))})
		return
	}

	var moved []map[string]string
	var skipped []string
	stats := map[string]int{}

	for _, e := range entries {
		if e.IsDir() {
			skipped = append(skipped, e.Name())
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext == "" {
			skipped = append(skipped, e.Name())
			continue
		}
		cat := fileCategory(ext)
		destDir := filepath.Join(req.Folder, cat)

		if !req.DryRun {
			os.MkdirAll(destDir, 0755)
			src := filepath.Join(req.Folder, e.Name())
			dst := filepath.Join(destDir, e.Name())
			// 동일 파일명 충돌 방지
			if _, err := os.Stat(dst); err == nil {
				base := strings.TrimSuffix(e.Name(), ext)
				ts := time.Now().Format("150405")
				dst = filepath.Join(destDir, base+"_"+ts+ext)
			}
			os.Rename(src, dst)
		}
		moved = append(moved, map[string]string{"file": e.Name(), "category": cat, "dest": filepath.Join(cat, e.Name())})
		stats[cat]++
	}

	// 요약 메시지
	parts := []string{}
	for cat, n := range stats {
		parts = append(parts, fmt.Sprintf("%s %d개", cat, n))
	}
	sort.Strings(parts)
	summary := strings.Join(parts, ", ")
	action := "정리"
	if req.DryRun {
		action = "정리 예정 (dry-run)"
	}
	msg := fmt.Sprintf("%s 폴더 %s 완료: %s", filepath.Base(req.Folder), action, summary)
	if len(moved) == 0 {
		msg = filepath.Base(req.Folder) + " 폴더에 정리할 파일이 없습니다."
	}

	json200(w, map[string]any{
		"success": true, "message": msg,
		"result": organizeResult{Moved: moved, Skipped: skipped, Total: len(moved), Message: msg},
	})
}

// ── POST /api/file/duplicates ──────────────────────────────

func handleFileDuplicates(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Folder  string `json:"folder"`
		Message string `json:"message"`
	}
	tryDecodeBody(r, &req)
	if req.Folder == "" {
		home, _ := os.UserHomeDir()
		msg := strings.ToLower(req.Message)
		if strings.Contains(msg, "다운로드") || strings.Contains(msg, "download") {
			req.Folder = filepath.Join(home, "Downloads")
		} else {
			req.Folder = filepath.Join(home, "Desktop")
		}
	}

	// MD5 해시 기반 중복 탐지 (최대 500파일, 각 최대 50MB)
	hashes := map[string][]string{}
	count := 0
	filepath.Walk(req.Folder, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || count > 500 {
			return nil
		}
		if info.Size() > 50*1024*1024 || info.Size() == 0 {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()
		h := md5.New()
		io.Copy(h, io.LimitReader(f, 10*1024*1024)) // 처음 10MB만 해시
		hash := fmt.Sprintf("%x", h.Sum(nil))
		hashes[hash] = append(hashes[hash], path)
		count++
		return nil
	})

	type dupGroup struct {
		Name   string   `json:"name"`
		Paths  []string `json:"paths"`
		Count  int      `json:"count"`
		SizeMB float64  `json:"size_mb"`
	}
	var groups []dupGroup
	totalWaste := int64(0)
	for _, files := range hashes {
		if len(files) < 2 {
			continue
		}
		info, _ := os.Stat(files[0])
		sz := int64(0)
		if info != nil {
			sz = info.Size()
			totalWaste += sz * int64(len(files)-1)
		}
		groups = append(groups, dupGroup{
			Name:   filepath.Base(files[0]),
			Paths:  files,
			Count:  len(files),
			SizeMB: float64(sz) / 1024 / 1024,
		})
	}

	wasteMB := float64(totalWaste) / 1024 / 1024
	msg := fmt.Sprintf("중복 파일 %d그룹 발견, 낭비 공간 약 %.1fMB", len(groups), wasteMB)
	if len(groups) == 0 {
		msg = "중복 파일이 없습니다."
	}

	json200(w, map[string]any{
		"success": true, "message": msg,
		"groups": groups, "total_groups": len(groups),
		"waste_mb": wasteMB, "waste": fmt.Sprintf("%.1fMB", wasteMB),
	})
}

// ── POST /api/file/large ───────────────────────────────────

func handleFileLarge(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Folder     string `json:"folder"`
		MinSizeMB  int    `json:"min_size_mb"`
		Message    string `json:"message"`
	}
	tryDecodeBody(r, &req)
	if req.MinSizeMB == 0 {
		req.MinSizeMB = 100
	}
	if req.Folder == "" {
		home, _ := os.UserHomeDir()
		req.Folder = home
	}

	type fileInfo struct {
		Path   string  `json:"path"`
		SizeMB float64 `json:"size_mb"`
		Ext    string  `json:"ext"`
	}
	var large []fileInfo
	minBytes := int64(req.MinSizeMB) * 1024 * 1024

	filepath.Walk(req.Folder, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || len(large) > 50 {
			return nil
		}
		if info.Size() >= minBytes {
			large = append(large, fileInfo{
				Path:   path,
				SizeMB: float64(info.Size()) / 1024 / 1024,
				Ext:    filepath.Ext(path),
			})
		}
		return nil
	})

	// 크기 내림차순 정렬
	sort.Slice(large, func(i, j int) bool { return large[i].SizeMB > large[j].SizeMB })

	msg := fmt.Sprintf("%dMB 이상 대용량 파일 %d개 발견", req.MinSizeMB, len(large))
	if len(large) == 0 {
		msg = fmt.Sprintf("%dMB 이상인 파일이 없습니다.", req.MinSizeMB)
	}
	json200(w, map[string]any{"success": true, "message": msg, "files": large})
}
