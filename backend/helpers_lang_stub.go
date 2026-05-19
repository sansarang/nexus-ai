//go:build !windows

package main

import "strings"

// detectSystemLang: Mac/Linux 시스템 언어 감지
func detectSystemLang() string {
	lang := strings.ToLower(strings.TrimSpace(getEnvLang()))
	if strings.HasPrefix(lang, "en") {
		return "en"
	}
	return "ko"
}
