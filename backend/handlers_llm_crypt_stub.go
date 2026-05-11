//go:build !windows

package main

// Mac/Linux: DPAPI 없음 → 평문 그대로 (개발 환경용)
func encryptDPAPI(plaintext string) string { return plaintext }
func decryptDPAPI(encrypted string) string { return encrypted }
