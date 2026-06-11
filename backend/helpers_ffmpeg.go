// helpers_ffmpeg.go — ffmpeg 실행파일 경로 탐색 (파일 변환 기능에서 공용 사용).
// (원래 handlers_video_local.go에 있었으나, 영상 기능 제거 후에도 file_process가 필요로 해 분리 보존.)
package main

import (
	"os"
	"os/exec"
	"path/filepath"
)

func findFFmpeg() string {
	if p, err := exec.LookPath("ffmpeg"); err == nil {
		return p
	}
	candidates := []string{
		`C:\ffmpeg\bin\ffmpeg.exe`,
		`C:\Program Files\ffmpeg\bin\ffmpeg.exe`,
		`C:\tools\ffmpeg\bin\ffmpeg.exe`,
	}
	if appdata := os.Getenv("APPDATA"); appdata != "" {
		candidates = append(candidates, filepath.Join(appdata, "Nexus", "ffmpeg", "bin", "ffmpeg.exe"))
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}
