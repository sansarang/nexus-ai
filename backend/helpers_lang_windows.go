//go:build windows

package main

import (
	"os/exec"
	"strings"
)

// detectSystemLang: Windows 시스템 언어 감지
// PowerShell로 CurrentCulture 조회 → "en" or "ko"
func detectSystemLang() string {
	out, err := exec.Command("powershell", "-NoProfile", "-Command",
		"[System.Globalization.CultureInfo]::CurrentUICulture.Name").Output()
	if err == nil {
		locale := strings.TrimSpace(strings.ToLower(string(out)))
		if strings.HasPrefix(locale, "en") {
			return "en"
		}
		if strings.HasPrefix(locale, "ko") {
			return "ko"
		}
	}
	// PowerShell 실패 시 LANG 환경변수 fallback
	lang := strings.ToLower(strings.TrimSpace(getEnvLang()))
	if strings.HasPrefix(lang, "en") {
		return "en"
	}
	return "ko"
}
