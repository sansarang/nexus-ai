// card_type_mapping_shared.go — 액션 → 카드 타입 자동 매핑 (Mac/Windows 공용)
// 프론트 cardRegistry.ACTION_TO_CARD 와 1:1 동기화. 백엔드도 같은 표를 사용해서
// CommandResponse.CardType 을 채워주면 프론트 자동 추론 안 거치고 즉시 라우팅.
package main

// actionToCard: 액션명 → 카드 타입. 프론트 cardRegistry.ts ACTION_TO_CARD 와 동일.
var actionToCard = map[string]string{
	// 시스템 진단
	"stats":         "pc_status",
	"pc_status":     "pc_status",
	"scan":          "scan_result",
	"security_scan": "scan_result",
	"full_scan":     "scan_result",
	"daily_report":  "daily_report",
	"pc_report":     "pc_report",
	"health_report": "pc_report",
	"clean":         "clean_result",
	"smart_organize": "smart_organize",
	"repair":        "repair_result",

	// 시스템 보안/네트워크
	"remote_access":    "remote_access",
	"process_security": "process_security",
	"defender":         "defender",
	"defender_status":  "defender",
	"startup_items":    "startup_items",
	"process_top":      "process_top",
	"network_analysis": "network",
	"driver_check":     "drivers",
	"programs_list":    "programs_list",
	"boot_analysis":    "boot_analysis",

	// 검색 & 컨텐츠
	"web_search":     "web_search",
	"news_search":    "news_search",
	"youtube_search": "youtube",
	"video_search":   "youtube",
	"price_compare":  "price_compare",
	"multi_action":   "price_compare",
	"deep_search":    "deep_search",
	"deep_research":  "deep_search",

	// 파일/문서
	"file_search":    "file_search",
	"file_duplicates": "duplicates",
	"doc_compare":    "doc_compare",
	"doc_find":       "doc_find",
	"doc_summary":    "doc_summary",
	"vision":         "vision_result",
	"vision_screen":  "vision_result",
	"vision_ocr":     "vision_ocr",

	// 생산성
	"notes":           "notes",
	"note":            "notes",
	"weather":         "weather_card",
	"email_inbox":     "email_list",
	"email_summarize": "email_list",
	"email_classify":  "email_list",
	"calendar_today":  "timeline",
	"calendar_week":   "timeline",

	// 매크로/자동화
	"macro_list":      "macro_list",
	"macro_create":    "macro_created",
	"macro_run":       "macro_run",
	"journal_today":   "journal_today",
	"journal_history": "journal_history",

	// 시스템 제어
	"focus_mode":  "focus_mode",
	"open_folder": "folder_open",
}

// resolveCardTypeForAction: 액션명 → 카드 타입. 매핑 없으면 빈 문자열.
func resolveCardTypeForAction(action string) string {
	if action == "" {
		return ""
	}
	return actionToCard[action]
}
