//go:build windows

package cleaner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

type CleanResult struct {
	Item       string `json:"item"`
	FreedBytes int64  `json:"freed_bytes"`
	Error      string `json:"error,omitempty"`
}

func AutoClean(items []string) []CleanResult {
	results := []CleanResult{}
	for _, item := range items {
		var r CleanResult
		r.Item = item
		switch item {
		case "temp":
			r.FreedBytes, r.Error = cleanTempFiles()
		case "wucache":
			r.FreedBytes, r.Error = cleanWindowsUpdateCache()
		case "browser":
			r.FreedBytes, r.Error = cleanBrowserCache()
		case "prefetch":
			r.FreedBytes, r.Error = cleanPrefetch()
		case "thumbnail":
			r.FreedBytes, r.Error = cleanThumbnailCache()
		case "memory":
			r.FreedBytes, r.Error = optimizeMemory()
		}
		results = append(results, r)
	}
	return results
}

func cleanTempFiles() (int64, string) {
	tempDir := os.TempDir()
	freed := dirSize(tempDir)
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		return 0, err.Error()
	}
	for _, e := range entries {
		os.RemoveAll(filepath.Join(tempDir, e.Name()))
	}
	return freed, ""
}

func cleanWindowsUpdateCache() (int64, string) {
	dir := `C:\Windows\SoftwareDistribution\Download`
	freed := dirSize(dir)
	exec.Command("net", "stop", "wuauserv").Run()
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		os.RemoveAll(filepath.Join(dir, e.Name()))
	}
	exec.Command("net", "start", "wuauserv").Run()
	return freed, ""
}

func cleanBrowserCache() (int64, string) {
	home, _ := os.UserHomeDir()
	dirs := []string{
		filepath.Join(home, `AppData\Local\Google\Chrome\User Data\Default\Cache`),
		filepath.Join(home, `AppData\Local\Microsoft\Edge\User Data\Default\Cache`),
		filepath.Join(home, `AppData\Local\Mozilla\Firefox\Profiles`),
	}
	var freed int64
	for _, d := range dirs {
		freed += dirSize(d)
		os.RemoveAll(d)
		os.MkdirAll(d, 0755)
	}
	return freed, ""
}

func cleanPrefetch() (int64, string) {
	dir := `C:\Windows\Prefetch`
	freed := dirSize(dir)
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		os.Remove(filepath.Join(dir, e.Name()))
	}
	return freed, ""
}

func cleanThumbnailCache() (int64, string) {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, `AppData\Local\Microsoft\Windows\Explorer`)
	freed := dirSize(dir)
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".db" {
			os.Remove(filepath.Join(dir, e.Name()))
		}
	}
	return freed, ""
}

func optimizeMemory() (int64, string) {
	runtime.GC()
	return 0, ""
}

func dirSize(path string) int64 {
	var size int64
	filepath.Walk(path, func(_ string, info os.FileInfo, _ error) error {
		if info != nil && !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}

var _ = fmt.Sprintf // avoid unused import
