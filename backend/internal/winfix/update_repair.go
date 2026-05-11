//go:build windows

package winfix

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

type RepairStep struct {
	Name  string `json:"name"`
	Done  bool   `json:"done"`
	Error string `json:"error,omitempty"`
}

func RepairWindowsUpdate() chan RepairStep {
	ch := make(chan RepairStep, 10)
	go func() {
		defer close(ch)
		steps := []struct {
			name string
			fn   func() error
		}{
			{"Windows Update 서비스 중지", stopUpdateServices},
			{"업데이트 캐시 삭제", deleteUpdateCache},
			{"catroot2 초기화", resetCatroot2},
			{"서비스 재시작", startUpdateServices},
			{"SFC 시스템 파일 검사", runSFC},
			{"DISM 이미지 복구", runDISM},
		}
		for _, step := range steps {
			err := step.fn()
			r := RepairStep{Name: step.name, Done: err == nil}
			if err != nil {
				r.Error = err.Error()
			}
			ch <- r
		}
	}()
	return ch
}

func stopUpdateServices() error {
	for _, svc := range []string{"wuauserv", "cryptSvc", "bits", "msiserver"} {
		exec.Command("net", "stop", svc, "/y").Run()
	}
	return nil
}

func deleteUpdateCache() error {
	return os.RemoveAll(filepath.Join(os.Getenv("SystemRoot"), "SoftwareDistribution"))
}

func resetCatroot2() error {
	return os.RemoveAll(filepath.Join(os.Getenv("SystemRoot"), "System32", "catroot2"))
}

func startUpdateServices() error {
	for _, svc := range []string{"bits", "cryptSvc", "wuauserv"} {
		exec.Command("net", "start", svc).Run()
	}
	return nil
}

func runSFC() error {
	cmd := exec.Command("sfc", "/scannow")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Run()
}

func runDISM() error {
	cmd := exec.Command("DISM", "/Online", "/Cleanup-Image", "/RestoreHealth")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Run()
}
