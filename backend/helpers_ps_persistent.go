//go:build !windows

package main

// Mac/Linux stub — Persistent PS Session은 Windows 전용
// Windows 빌드에서 실제 구현이 제공됩니다.

// execPSPersistent: Mac에서는 일반 bash로 fallback
func execPSPersistent(script string) (string, error) {
	return "", nil
}

func closePSSession() {}
