package main

import "os"

// getEnvLang: LANG / LANGUAGE / LC_ALL 순서로 환경변수 확인
func getEnvLang() string {
	for _, key := range []string{"LANG", "LANGUAGE", "LC_ALL"} {
		if v := os.Getenv(key); v != "" {
			return v
		}
	}
	return ""
}
