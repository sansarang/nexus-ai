//go:build windows

package main

import (
	"encoding/base64"
	"strings"
)

// encryptDPAPI: Windows DPAPI로 평문 암호화 → base64 반환
// PowerShell [System.Security.Cryptography.ProtectedData] 사용 (추가 패키지 불필요)
func encryptDPAPI(plaintext string) string {
	if plaintext == "" {
		return ""
	}
	script := `
$bytes = [System.Text.Encoding]::UTF8.GetBytes("` + strings.ReplaceAll(plaintext, `"`, "`\"") + `")
$enc = [System.Security.Cryptography.ProtectedData]::Protect($bytes, $null, [System.Security.Cryptography.DataProtectionScope]::CurrentUser)
[Convert]::ToBase64String($enc)
`
	out, err := execPS(script)
	if err != nil {
		return plaintext // 실패 시 평문 유지 (호환성)
	}
	return strings.TrimSpace(string(out))
}

// decryptDPAPI: base64 → DPAPI 복호화 → 평문
func decryptDPAPI(encrypted string) string {
	if encrypted == "" {
		return ""
	}
	// base64인지 확인 (평문 키는 그대로 반환)
	if _, err := base64.StdEncoding.DecodeString(encrypted); err != nil {
		return encrypted // 평문 그대로 반환 (기존 데이터 호환)
	}
	script := `
$enc = [Convert]::FromBase64String("` + encrypted + `")
$bytes = [System.Security.Cryptography.ProtectedData]::Unprotect($enc, $null, [System.Security.Cryptography.DataProtectionScope]::CurrentUser)
[System.Text.Encoding]::UTF8.GetString($bytes)
`
	out, err := execPS(script)
	if err != nil {
		return encrypted // 복호화 실패 시 원본 반환
	}
	return strings.TrimSpace(string(out))
}
