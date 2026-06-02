// phase_a_safety.go — Phase A: 위험 액션 확인 시스템 (cross-platform)
// 사장님 요구: "확인/취소 카드, 안전장치, sandbox + dry-run + 롤백"
//
// 흐름:
//   사용자: "PC 재시작해"
//   → detectDangerousAction → 즉시 실행 X
//   → "PC 재시작 확인 카드" 반환 (사용자 클릭 → 진짜 실행)
//
// 적용 범위: shutdown/restart/sleep/delete/format/email-send/payment 등

package main

import (
	"strings"
	"time"
)

// DangerousAction — 위험 액션의 위험 등급
type DangerousAction struct {
	Action      string
	Severity    string // "low" / "medium" / "high" / "critical"
	Title       string
	Description string
	Reversible  bool
}

// dangerousActionRegistry — 확인 필요한 액션 정의
var dangerousActionRegistry = map[string]DangerousAction{
	"shutdown":          {Action: "shutdown", Severity: "critical", Title: "🛑 PC 종료", Description: "모든 작업이 종료됩니다. 저장 안 된 작업은 사라집니다.", Reversible: false},
	"restart":           {Action: "restart", Severity: "critical", Title: "🔄 PC 재시작", Description: "모든 앱이 닫히고 재시작됩니다.", Reversible: false},
	"sleep":             {Action: "sleep", Severity: "medium", Title: "😴 절전 모드", Description: "PC가 절전 모드로 들어갑니다.", Reversible: true},
	"file_delete":       {Action: "file_delete", Severity: "high", Title: "🗑️ 파일 삭제", Description: "선택한 파일이 영구 삭제됩니다.", Reversible: false},
	"file_move":         {Action: "file_move", Severity: "medium", Title: "📁 파일 이동", Description: "파일이 다른 위치로 이동됩니다.", Reversible: true},
	"clean_aggressive":  {Action: "clean_aggressive", Severity: "high", Title: "🧹 강제 정리", Description: "캐시·임시·로그 파일이 모두 삭제됩니다.", Reversible: false},
	"email_send":        {Action: "email_send", Severity: "high", Title: "📧 이메일 전송", Description: "메일이 즉시 발송됩니다. 회수 불가합니다.", Reversible: false},
	"payment":           {Action: "payment", Severity: "critical", Title: "💳 결제 실행", Description: "실제 결제가 이루어집니다.", Reversible: false},
	"app_uninstall":     {Action: "app_uninstall", Severity: "high", Title: "🗑️ 앱 제거", Description: "앱이 시스템에서 제거됩니다.", Reversible: false},
	"format_disk":       {Action: "format_disk", Severity: "critical", Title: "⚠️ 디스크 포맷", Description: "디스크가 완전히 초기화됩니다. 모든 데이터 손실.", Reversible: false},
	"registry_edit":     {Action: "registry_edit", Severity: "critical", Title: "⚙️ 레지스트리 편집", Description: "Windows 레지스트리가 변경됩니다. 시스템 불안정 가능.", Reversible: false},
}

// isNexusDangerousAction — 액션 이름으로 위험 여부 + 정보 반환
// (Windows의 handlers_desktop_agent.go isDangerousAction과 별개)
func isNexusDangerousAction(action string) (DangerousAction, bool) {
	if d, ok := dangerousActionRegistry[action]; ok {
		return d, true
	}
	// system_control 의 sub-control 도 매핑
	switch action {
	case "shutdown_confirmed", "restart_confirmed":
		return DangerousAction{}, false // 이미 확인됨 — 실행 통과
	}
	return DangerousAction{}, false
}

// buildConfirmCard — 확인 카드 응답 생성
func buildConfirmCard(action string, params map[string]any, msg, lang string) (map[string]any, string) {
	d, ok := isNexusDangerousAction(action)
	if !ok {
		return nil, ""
	}
	// 이미 사용자가 confirmed 라벨 붙였으면 통과
	if c, _ := params["confirmed"].(bool); c {
		return nil, ""
	}
	title := d.Title
	desc := d.Description
	if lang == "en" {
		desc = "Confirm to execute. " + desc
	}
	confirmMsg := title
	if msg != "" {
		confirmMsg = title + " — " + msg
	}
	return map[string]any{
		"needs_confirmation": true,
		"action":             action,
		"severity":           d.Severity,
		"title":              d.Title,
		"description":        desc,
		"reversible":         d.Reversible,
		"pending_params":     params,
		"timestamp":          time.Now().Unix(),
	}, confirmMsg
}

// detectDangerousInMessage — 사용자 원문에서 위험 의도 키워드 감지
func detectDangerousInMessage(msg string) string {
	lower := strings.ToLower(msg)
	patterns := map[string][]string{
		"shutdown":    {"전원 꺼", "pc 꺼", "컴퓨터 꺼", "전원 종료", "shutdown", "power off", "turn off"},
		"restart":     {"재시작", "리부팅", "리부트", "다시 시작", "restart", "reboot"},
		"sleep":       {"절전", "잠재워", "sleep mode"},
		"file_delete": {"파일 삭제", "지워", "remove file", "delete file"},
		"format_disk": {"포맷해", "디스크 초기화", "format disk", "wipe disk"},
		"payment":     {"결제해", "송금해", "transfer money", "pay now"},
	}
	for action, kws := range patterns {
		for _, kw := range kws {
			if strings.Contains(lower, kw) {
				return action
			}
		}
	}
	return ""
}
