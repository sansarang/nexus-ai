//go:build windows

package main

import (
	"net/http"
)

// ══════════════════════════════════════════════════════════════
//  Feature Flag 시스템
//  — 미구현 또는 조건부 활성화 기능을 명시적으로 관리
//  — 비활성 기능 호출 시 feature_not_implemented 응답 반환
// ══════════════════════════════════════════════════════════════

type featureFlag struct {
	enabled     bool
	description string
}

// enabledFeatures — 기능별 활성화 상태
// false 로 설정된 기능은 호출 시 즉시 feature_not_implemented 반환
var enabledFeatures = map[string]featureFlag{
	// ── Python 프록시 연결 필요 기능 ──────────────────────────
	"shodan":         {enabled: true,  description: "Shodan IP/도메인 보안 조회"},
	"wayback":        {enabled: true,  description: "Wayback Machine 스냅샷 조회"},
	"searx":          {enabled: false, description: "SearX 익명 검색 (미구현)"},
	"video_enhanced": {enabled: false, description: "고급 동영상 처리 (미구현)"},
	"ocr":            {enabled: true,  description: "OCR 텍스트 추출"},
	"desktop_agent":  {enabled: true,  description: "데스크탑 자동화 에이전트"},
	"multi_agent":    {enabled: true,  description: "멀티에이전트 실행"},
	"workflow":       {enabled: true,  description: "워크플로우 실행"},
	"brain":          {enabled: true,  description: "Second Brain (FAISS 벡터 검색)"},
	"stock":          {enabled: true,  description: "주식 시세 조회"},
	"email_imap":     {enabled: true,  description: "IMAP 이메일 연동"},
	"calendar_gcal":  {enabled: true,  description: "Google Calendar 연동"},
	"vision":         {enabled: true,  description: "비전 AI 분석"},
	"youtube":        {enabled: true,  description: "YouTube 검색"},
	"tiktok":         {enabled: true,  description: "TikTok 검색"},
	"ytmusic":        {enabled: true,  description: "YouTube Music 검색"},
	"legal":          {enabled: true,  description: "법률 문서 검색"},
	"medical":        {enabled: true,  description: "의학 정보 검색"},
	"ollama":         {enabled: true,  description: "로컬 Ollama LLM"},
}

// isFeatureEnabled — 기능 활성화 여부 확인
func isFeatureEnabled(feature string) bool {
	f, ok := enabledFeatures[feature]
	return ok && f.enabled
}

// featureNotImplemented — 비활성 기능 호출 시 표준 응답 반환
func featureNotImplemented(w http.ResponseWriter, feature string) {
	f, ok := enabledFeatures[feature]
	desc := feature
	if ok {
		desc = f.description
	}
	writeJSON(w, 501, map[string]any{
		"success": false,
		"code":    "feature_not_implemented",
		"feature": feature,
		"message": desc + " 기능은 현재 준비 중이에요. 곧 업데이트될 예정입니다! 🔧",
	})
}

// handleFeatureList — GET /api/features : 전체 기능 플래그 목록 반환 (관리자용)
func handleFeatureList(w http.ResponseWriter, _ *http.Request) {
	result := make(map[string]any, len(enabledFeatures))
	for k, v := range enabledFeatures {
		result[k] = map[string]any{"enabled": v.enabled, "description": v.description}
	}
	json200(w, map[string]any{"features": result, "total": len(enabledFeatures)})
}
