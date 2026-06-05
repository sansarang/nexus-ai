// card_type_mapping_shared.go — 액션 → 카드 타입 자동 매핑 (Mac/Windows 공용)
// 프론트 cardRegistry.ACTION_TO_CARD 와 1:1 동기화. 백엔드도 같은 표를 사용해서
// CommandResponse.CardType 을 채워주면 프론트 자동 추론 안 거치고 즉시 라우팅.
package main

// actionToCard: 액션명 → 카드 타입. 프론트 cardRegistry.ts ACTION_TO_CARD 와 동일.
// 49+ 고유 카드 타입 지원
var actionToCard = map[string]string{
	// ── 시스템 진단 (card: pc_status, scan_result, clean_result, repair_result, daily_report, pc_report) ──
	"stats":          "pc_status",
	"pc_status":      "pc_status",
	"gpu_stats":      "pc_status",
	"scan":           "scan_result",
	"security_scan":  "scan_result",
	"full_scan":      "scan_result",
	"daily_report":   "daily_report",
	"pc_report":      "pc_report",
	"health_report":  "pc_report",
	"clean":          "clean_result",
	"autoclean":      "clean_result",
	"smart_organize": "smart_organize",
	"repair":         "repair_result",

	// ── 보안/네트워크 (card: remote_access, process_security, defender, startup_items, process_top, network, drivers, programs_list, boot_analysis, security_alert) ──
	"remote_access":    "remote_access",
	"process_security": "process_security",
	"defender":         "defender",
	"defender_status":  "defender",
	"startup_items":    "startup_items",
	"process_top":      "process_top",
	"network_analysis": "network",
	"driver_check":     "drivers",
	"drivers":          "drivers",
	"programs_list":    "programs_list",
	"boot_analysis":    "boot_analysis",
	"virustotal":       "security_alert",
	"shodan":           "security_alert",

	// ── 검색/정보 (card: web_search, news_search, youtube, price_compare, deep_search, reddit_result, stock_card, map_result, translate_result) ──
	"web_search":       "web_search",
	"news_search":      "news_search",
	"youtube_search":   "youtube",
	"video_search":     "youtube",
	"ytmusic_search":   "youtube",
	"tiktok_search":    "tiktok_result",
	"tiktok_trending":  "tiktok_result",
	"netflix_trending": "media_result",
	"price_compare":    "price_compare",
	"deep_search":      "deep_search",
	"deep_research":    "deep_search",
	"reddit_search":    "reddit_result",
	"stock_price":      "stock_card",
	"stock_analysis":   "stock_card",
	"map_search":       "map_result",
	"directions":       "map_result",
	"travel_time":      "map_result",
	"translate":        "translate_result",
	"business_lookup":  "business_card",   // ★ 국세청 사업자 조회 전용 카드
	"wayback":          "web_search",

	// ── 파일/문서 (card: file_search, duplicates, doc_compare, doc_find, doc_summary, vision_result, vision_ocr, file_result, benchmark_result) ──
	"file_search":      "file_search",
	"file_organize":    "file_search",
	"file_duplicates":  "duplicates",
	"file_large":       "duplicates",
	"doc_compare":      "doc_compare",
	"doc_find":         "doc_find",
	"doc_summary":      "doc_summary",
	"vision":           "vision_result",
	"vision_screen":    "vision_result",
	"vision_ocr":       "vision_ocr",
	"ocr_clipboard":    "vision_ocr",
	"excel_analyze":    "file_result",
	"analyze_excel":    "file_result",
	"excel_auto_create":"file_result",
	"create_excel":     "file_result",
	"make_excel":       "file_result",
	"excel_save":       "file_result",
	"doc_auto_create":  "file_result",
	"create_doc":       "file_result",
	"make_doc":         "file_result",
	"pdf_auto_create":  "file_result",
	"create_pdf":       "file_result",
	"make_pdf":         "file_result",
	"video_workflow":   "file_result",
	"video_summary":    "file_result",
	"video_transcript": "doc_summary",   // 유튜브 자막 → 문서 요약 카드
	"video_download_summary": "file_result",

	// ── 이메일/캘린더 (card: email_list, email_draft, timeline, calendar_event) ──
	"email_inbox":        "email_list",
	"email_summarize":    "email_list",
	"email_classify":     "email_list",
	"email_draft_reply":  "email_draft",
	"email_send":         "email_draft",
	"imap_inbox":         "email_list",
	"gmail_inbox":        "email_list",
	"calendar_today":     "timeline",
	"calendar_week":      "timeline",
	"calendar_add":       "calendar_event",
	"calendar_find_slot": "calendar_event",
	"calendar_smart_add": "calendar_event",

	// ── 메모/생산성 (card: notes, weather_card, macro_list, macro_created, macro_run, journal_today, journal_history, focus_mode, folder_open, workflow_result, briefing_card, meeting_summary, memory_card) ──
	"notes":           "notes",
	"note":            "notes",
	"weather":         "weather_card",
	"macro_list":      "macro_list",
	"macro_create":    "macro_created",
	"macro_run":       "macro_run",
	"journal_today":   "journal_today",
	"journal_history": "journal_history",
	"focus_mode":      "focus_mode",
	"open_folder":     "folder_open",
	"workflow_run":    "workflow_result",
	"workflow_run_now":"workflow_result",
	"briefing_now":    "briefing_card",
	"daily_briefing":  "briefing_card",
	"meeting_summarize":"meeting_summary",
	"meeting_transcribe":"meeting_summary",
	"memory_search":   "memory_card",
	"recall_search":   "memory_card",

	// ── 시스템 제어 (card: cmd_result) ──
	"launch_app":    "cmd_result",
	"process_kill":  "cmd_result",
	"power_action":  "cmd_result",
	"volume_control":"cmd_result",
	"brightness":    "cmd_result",
	"wifi_toggle":   "cmd_result",
	"power_plan":    "cmd_result",
	"system_updates":"cmd_result",

	// ── Agent/자동화 (card: agent_result) ──
	"desktop_agent":  "agent_result",
	"multi_agent":    "agent_result",
	"browser_agent":  "agent_result",
	"smart_agent":    "agent_result",
}

// resolveCardTypeForAction: 액션명 → 카드 타입. 매핑 없으면 빈 문자열.
func resolveCardTypeForAction(action string) string {
	if action == "" {
		return ""
	}
	return actionToCard[action]
}
