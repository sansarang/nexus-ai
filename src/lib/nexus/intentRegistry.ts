/**
 * Intent 레지스트리 — Single Source of Truth
 *
 * ┌─────────────────────────────────────────────────────────┐
 * │ Intent 타입에 새 인텐트를 추가하면 이 파일에 항목을      │
 * │ 추가해야만 컴파일됩니다. (satisfies Record<Intent, ...> │
 * │ 가 누락된 키를 컴파일 에러로 잡아냅니다.)               │
 * └─────────────────────────────────────────────────────────┘
 *
 * 사용처:
 *  - chatIntentShared.errorReturn → label, hint 결정
 *  - 향후 UI 메뉴, 미구현 인텐트 회색 표시, status 배지 등
 */

import type { Intent } from './intentDetector'

export type IntentStatus =
  | 'live'             // 모든 플랫폼에서 동작
  | 'windows_only'     // Windows 정식 빌드에서만 (WMI/PowerShell/COM 의존)
  | 'beta'             // 동작은 하지만 안정성 낮음
  | 'not_implemented'  // 메뉴/타입만 존재, 실제 동작 X
  | 'meta'             // LLM 폴백 등 특수 처리 ('none' 등)

export type IntentCategory =
  | 'system'           // PC 상태·리포트·진단
  | 'security'         // 보안·해킹·바이러스
  | 'system_control'   // 볼륨·전원·앱 실행 등
  | 'file'             // 파일·문서·검색
  | 'web'              // 웹 검색·뉴스·영상
  | 'productivity'     // 메모·일정·매크로·워크플로
  | 'media'            // OCR·화면 분석·녹음·자막
  | 'email_calendar'   // 이메일·캘린더·IMAP
  | 'ai'               // AI 에이전트·브레인·페르소나
  | 'pro'              // 전문가 페르소나 (Pro 한정)
  | 'weather_travel'   // 날씨·교통·번역
  | 'meta'             // 'none' 등 메타

export interface IntentSpec {
  status: IntentStatus
  category: IntentCategory
  emoji: string
  labelKo: string
  labelEn: string
  /** 사용자에게 표시할 짧은 설명 (선택) */
  descKo?: string
  descEn?: string
}

/* eslint-disable @typescript-eslint/no-unused-vars */
export const INTENT_REGISTRY = {
  // ── 시스템 모니터링 ─────────────────────────────────────
  pc_status:        { status: 'windows_only', category: 'system',         emoji: '📊', labelKo: 'PC 상태',          labelEn: 'PC Status' },
  daily_report:     { status: 'windows_only', category: 'system',         emoji: '📋', labelKo: '일일 리포트',      labelEn: 'Daily Report' },
  pc_report:        { status: 'windows_only', category: 'system',         emoji: '🏥', labelKo: 'PC 건강 리포트',   labelEn: 'PC Health Report' },
  perf_history:     { status: 'live',         category: 'system',         emoji: '📈', labelKo: '성능 이력',        labelEn: 'Performance History' },
  perf_anomaly:     { status: 'live',         category: 'system',         emoji: '🚨', labelKo: '이상 감지',        labelEn: 'Anomaly Detection' },
  gpu_stats:        { status: 'windows_only', category: 'system',         emoji: '🎮', labelKo: 'GPU 정보',         labelEn: 'GPU Stats' },
  process_top:      { status: 'windows_only', category: 'system',         emoji: '🏃', labelKo: '프로세스 TOP',     labelEn: 'Top Processes' },
  network_analysis: { status: 'windows_only', category: 'system',         emoji: '🌐', labelKo: '네트워크 분석',    labelEn: 'Network Analysis' },
  boot_analysis:    { status: 'windows_only', category: 'system',         emoji: '🚀', labelKo: '부팅 분석',        labelEn: 'Boot Analysis' },
  driver_check:     { status: 'windows_only', category: 'system',         emoji: '🔧', labelKo: '드라이버 점검',    labelEn: 'Driver Check' },
  programs_list:    { status: 'windows_only', category: 'system',         emoji: '📦', labelKo: '설치 프로그램',    labelEn: 'Installed Programs' },
  windows_updates:  { status: 'windows_only', category: 'system',         emoji: '🔄', labelKo: 'Windows 업데이트', labelEn: 'Windows Updates' },

  // ── 보안 ─────────────────────────────────────────────
  security_scan:    { status: 'windows_only', category: 'security',       emoji: '🔒', labelKo: '보안 스캔',        labelEn: 'Security Scan' },
  full_scan:        { status: 'windows_only', category: 'security',       emoji: '🔍', labelKo: '전체 진단',        labelEn: 'Full Diagnostic' },
  remote_access:    { status: 'windows_only', category: 'security',       emoji: '👁️', labelKo: '원격 접속 탐지',   labelEn: 'Remote Access Scan' },
  process_security: { status: 'windows_only', category: 'security',       emoji: '⚠️', labelKo: '프로세스 보안',    labelEn: 'Process Security' },
  hosts_check:      { status: 'windows_only', category: 'security',       emoji: '📝', labelKo: 'hosts 파일 점검',  labelEn: 'Hosts Check' },
  startup_items:    { status: 'windows_only', category: 'security',       emoji: '🟢', labelKo: '시작 프로그램',    labelEn: 'Startup Items' },
  defender_status:  { status: 'windows_only', category: 'security',       emoji: '🛡️', labelKo: 'Windows Defender', labelEn: 'Defender Status' },
  account_check:    { status: 'windows_only', category: 'security',       emoji: '👤', labelKo: '계정 점검',        labelEn: 'Account Check' },
  virus_check:      { status: 'live',         category: 'security',       emoji: '🦠', labelKo: 'VirusTotal 검사',  labelEn: 'VirusTotal Check' },
  app_permissions:  { status: 'windows_only', category: 'security',       emoji: '🔐', labelKo: '앱 권한 감사',     labelEn: 'App Permissions' },

  // ── 시스템 제어 ──────────────────────────────────────
  clean:            { status: 'windows_only', category: 'system_control', emoji: '🧹', labelKo: '디스크 정리',      labelEn: 'Disk Cleanup' },
  repair:           { status: 'windows_only', category: 'system_control', emoji: '🛠️', labelKo: '문제 수리',        labelEn: 'Repair' },
  open_folder:      { status: 'windows_only', category: 'system_control', emoji: '📂', labelKo: '폴더 열기',        labelEn: 'Open Folder' },
  volume_control:   { status: 'windows_only', category: 'system_control', emoji: '🔊', labelKo: '볼륨',             labelEn: 'Volume' },
  brightness:       { status: 'windows_only', category: 'system_control', emoji: '💡', labelKo: '밝기',             labelEn: 'Brightness' },
  wifi_toggle:      { status: 'windows_only', category: 'system_control', emoji: '📶', labelKo: 'Wi-Fi',            labelEn: 'Wi-Fi' },
  power_action:     { status: 'windows_only', category: 'system_control', emoji: '⚡', labelKo: '전원',             labelEn: 'Power' },
  power_plan:       { status: 'windows_only', category: 'system_control', emoji: '🔋', labelKo: '전원 계획',        labelEn: 'Power Plan' },
  launch_app:       { status: 'windows_only', category: 'system_control', emoji: '🚀', labelKo: '앱 실행',          labelEn: 'Launch App' },
  process_kill:     { status: 'windows_only', category: 'system_control', emoji: '🔫', labelKo: '프로세스 종료',    labelEn: 'Kill Process' },
  registry_clean:   { status: 'windows_only', category: 'system_control', emoji: '🗂️', labelKo: '레지스트리 정리',  labelEn: 'Registry Clean' },
  restore_create:   { status: 'windows_only', category: 'system_control', emoji: '♻️', labelKo: '복구 포인트 생성', labelEn: 'Create Restore Point' },
  disk_check:       { status: 'windows_only', category: 'system_control', emoji: '💽', labelKo: '디스크 검사',      labelEn: 'Disk Check' },
  browser_clean:    { status: 'windows_only', category: 'system_control', emoji: '🌐', labelKo: '브라우저 정리',    labelEn: 'Browser Clean' },
  focus_mode:       { status: 'live',         category: 'system_control', emoji: '🎯', labelKo: '집중 모드',        labelEn: 'Focus Mode' },

  // ── 파일·문서 ─────────────────────────────────────────
  file_search:      { status: 'windows_only', category: 'file',           emoji: '🔎', labelKo: '파일 검색',        labelEn: 'File Search' },
  file_organize:    { status: 'windows_only', category: 'file',           emoji: '📁', labelKo: '폴더 정리',        labelEn: 'File Organize' },
  file_duplicates:  { status: 'windows_only', category: 'file',           emoji: '👯', labelKo: '중복 파일 찾기',   labelEn: 'Find Duplicates' },
  smart_organize:   { status: 'windows_only', category: 'file',           emoji: '✨', labelKo: '스마트 정리',      labelEn: 'Smart Organize' },
  doc_compare:      { status: 'live',         category: 'file',           emoji: '📑', labelKo: '문서 비교',        labelEn: 'Document Compare' },
  doc_find:         { status: 'live',         category: 'file',           emoji: '📃', labelKo: '문서 찾기',        labelEn: 'Document Find' },
  doc_summary:      { status: 'live',         category: 'file',           emoji: '📄', labelKo: '문서 요약',        labelEn: 'Document Summary' },
  deep_search:      { status: 'live',         category: 'file',           emoji: '🔬', labelKo: '딥서치',           labelEn: 'Deep Search' },

  // ── 웹 검색·콘텐츠 ────────────────────────────────────
  price_compare:    { status: 'live',         category: 'web',            emoji: '🛍️', labelKo: '가격 비교',        labelEn: 'Price Compare' },
  news_search:      { status: 'live',         category: 'web',            emoji: '📰', labelKo: '뉴스 검색',        labelEn: 'News Search' },
  youtube_search:   { status: 'live',         category: 'web',            emoji: '▶️', labelKo: 'YouTube 검색',     labelEn: 'YouTube Search' },
  video_search:     { status: 'live',         category: 'web',            emoji: '🎥', labelKo: '영상 검색',        labelEn: 'Video Search' },
  video_download:   { status: 'live',         category: 'web',            emoji: '⬇️', labelKo: '영상 다운로드',    labelEn: 'Video Download' },
  video_transcript: { status: 'live',         category: 'web',            emoji: '🎬', labelKo: '영상 자막 요약',   labelEn: 'Video Transcript' },
  reddit_search:    { status: 'live',         category: 'web',            emoji: '🤖', labelKo: 'Reddit 검색',      labelEn: 'Reddit Search' },
  search_pdf:       { status: 'live',         category: 'web',            emoji: '📑', labelKo: 'PDF 리포트',       labelEn: 'PDF Report' },

  // ── 생산성 ────────────────────────────────────────────
  clipboard:        { status: 'windows_only', category: 'productivity',   emoji: '📋', labelKo: '클립보드',         labelEn: 'Clipboard' },
  clipboard_ai:     { status: 'windows_only', category: 'productivity',   emoji: '🪄', labelKo: '클립보드 AI',      labelEn: 'Clipboard AI' },
  clipboard_history:{ status: 'windows_only', category: 'productivity',   emoji: '📜', labelKo: '클립보드 기록',    labelEn: 'Clipboard History' },
  notes:            { status: 'live',         category: 'productivity',   emoji: '📝', labelKo: '메모',             labelEn: 'Notes' },
  schedule_list:    { status: 'live',         category: 'productivity',   emoji: '📅', labelKo: '스케줄 목록',      labelEn: 'Schedules' },
  schedule_add:     { status: 'live',         category: 'productivity',   emoji: '➕', labelKo: '스케줄 추가',      labelEn: 'Add Schedule' },
  schedule_delete:  { status: 'live',         category: 'productivity',   emoji: '🗑️', labelKo: '스케줄 삭제',      labelEn: 'Delete Schedule' },
  macro_list:       { status: 'windows_only', category: 'productivity',   emoji: '🎛️', labelKo: '매크로 목록',      labelEn: 'Macros' },
  macro_create:     { status: 'windows_only', category: 'productivity',   emoji: '🆕', labelKo: '매크로 생성',      labelEn: 'Create Macro' },
  macro_run:        { status: 'windows_only', category: 'productivity',   emoji: '▶️', labelKo: '매크로 실행',      labelEn: 'Run Macro' },
  journal_today:    { status: 'windows_only', category: 'productivity',   emoji: '📔', labelKo: '오늘 업무 일지',   labelEn: "Today's Journal" },
  journal_generate: { status: 'windows_only', category: 'productivity',   emoji: '📓', labelKo: '일지 생성',        labelEn: 'Generate Journal' },
  journal_history:  { status: 'windows_only', category: 'productivity',   emoji: '📚', labelKo: '일지 기록',        labelEn: 'Journal History' },
  workflow_run:     { status: 'beta',         category: 'productivity',   emoji: '⚡', labelKo: '워크플로 실행',    labelEn: 'Run Workflow' },
  workflow_plan:    { status: 'beta',         category: 'productivity',   emoji: '🧠', labelKo: '워크플로 계획',    labelEn: 'Workflow Plan' },
  workflow_list:    { status: 'beta',         category: 'productivity',   emoji: '📋', labelKo: '워크플로 목록',    labelEn: 'Workflows' },
  workflow_create:  { status: 'beta',         category: 'productivity',   emoji: '🆕', labelKo: '워크플로 생성',    labelEn: 'Create Workflow' },
  workflow_templates:{ status: 'beta',        category: 'productivity',   emoji: '📐', labelKo: '워크플로 템플릿',  labelEn: 'Workflow Templates' },
  report_email:     { status: 'live',         category: 'productivity',   emoji: '📧', labelKo: '리포트 이메일',    labelEn: 'Report Email' },
  voice_todo:       { status: 'live',         category: 'productivity',   emoji: '🎤', labelKo: '음성 할일',        labelEn: 'Voice Todo' },

  // ── 멀티미디어·AI 비전 ─────────────────────────────────
  vision_screen:    { status: 'windows_only', category: 'media',          emoji: '👁️', labelKo: '화면 분석',        labelEn: 'Screen Vision' },
  vision_ocr:       { status: 'windows_only', category: 'media',          emoji: '🔡', labelKo: '화면 OCR',         labelEn: 'Screen OCR' },
  caption_start:    { status: 'windows_only', category: 'media',          emoji: '🎬', labelKo: '실시간 자막 시작', labelEn: 'Start Captions' },
  caption_stop:     { status: 'windows_only', category: 'media',          emoji: '🛑', labelKo: '자막 종료',        labelEn: 'Stop Captions' },
  meeting_start:    { status: 'windows_only', category: 'media',          emoji: '🎙️', labelKo: '녹음 시작',        labelEn: 'Start Recording' },
  meeting_stop:     { status: 'windows_only', category: 'media',          emoji: '⏹️', labelKo: '녹음 종료',        labelEn: 'Stop Recording' },
  meeting_summary:  { status: 'windows_only', category: 'media',          emoji: '📝', labelKo: '회의 요약',        labelEn: 'Meeting Summary' },
  meeting_list:     { status: 'windows_only', category: 'media',          emoji: '📋', labelKo: '회의 목록',        labelEn: 'Meetings' },
  recall_capture:   { status: 'windows_only', category: 'media',          emoji: '📸', labelKo: '화면 기억 저장',   labelEn: 'Capture Memory' },
  recall_search:    { status: 'windows_only', category: 'media',          emoji: '🔍', labelKo: '화면 기억 검색',   labelEn: 'Search Memory' },
  dictation_start:  { status: 'windows_only', category: 'media',          emoji: '⌨️', labelKo: '받아쓰기',         labelEn: 'Dictation' },

  // ── 이메일·캘린더 ─────────────────────────────────────
  email_inbox:      { status: 'windows_only', category: 'email_calendar', emoji: '📥', labelKo: '받은 메일',        labelEn: 'Inbox' },
  email_send:       { status: 'windows_only', category: 'email_calendar', emoji: '📤', labelKo: '메일 전송',        labelEn: 'Send Email' },
  email_summarize:  { status: 'windows_only', category: 'email_calendar', emoji: '📊', labelKo: '메일 요약',        labelEn: 'Summarize Inbox' },
  email_classify:   { status: 'windows_only', category: 'email_calendar', emoji: '🏷️', labelKo: '메일 분류',        labelEn: 'Classify Email' },
  email_draft:      { status: 'windows_only', category: 'email_calendar', emoji: '✍️', labelKo: '메일 초안',        labelEn: 'Draft Reply' },
  calendar_today:   { status: 'windows_only', category: 'email_calendar', emoji: '📅', labelKo: '오늘 일정',        labelEn: "Today's Schedule" },
  calendar_week:    { status: 'windows_only', category: 'email_calendar', emoji: '🗓️', labelKo: '주간 일정',        labelEn: 'Weekly Schedule' },
  calendar_add:     { status: 'windows_only', category: 'email_calendar', emoji: '➕', labelKo: '일정 추가',        labelEn: 'Add Event' },
  calendar_find_slot:{status: 'windows_only', category: 'email_calendar', emoji: '🕐', labelKo: '빈 시간 찾기',     labelEn: 'Find Free Slot' },
  calendar_smart_add:{status: 'windows_only', category: 'email_calendar', emoji: '🪄', labelKo: '스마트 일정',      labelEn: 'Smart Add Event' },
  imap_inbox:       { status: 'live',         category: 'email_calendar', emoji: '📨', labelKo: 'IMAP 받은 메일',   labelEn: 'IMAP Inbox' },
  imap_send:        { status: 'live',         category: 'email_calendar', emoji: '📩', labelKo: 'IMAP 전송',        labelEn: 'IMAP Send' },

  // ── AI / 에이전트 ─────────────────────────────────────
  multi_action:     { status: 'live',         category: 'ai',             emoji: '🎯', labelKo: '멀티 액션',        labelEn: 'Multi Action' },
  multi_agent:      { status: 'beta',         category: 'ai',             emoji: '🤝', labelKo: '멀티 에이전트',    labelEn: 'Multi Agent' },
  parallel_queries: { status: 'live',         category: 'ai',             emoji: '⚡', labelKo: '병렬 검색',        labelEn: 'Parallel Queries' },
  briefing_now:     { status: 'live',         category: 'ai',             emoji: '☀️', labelKo: '모닝 브리핑',      labelEn: 'Morning Briefing' },
  persona_list:     { status: 'live',         category: 'ai',             emoji: '🎭', labelKo: '페르소나 목록',    labelEn: 'Personas' },
  persona_switch:   { status: 'live',         category: 'ai',             emoji: '🔄', labelKo: '페르소나 전환',    labelEn: 'Switch Persona' },
  brain_search:     { status: 'beta',         category: 'ai',             emoji: '🧠', labelKo: 'Second Brain 검색', labelEn: 'Brain Search' },
  brain_stats:      { status: 'beta',         category: 'ai',             emoji: '📊', labelKo: 'Brain 통계',       labelEn: 'Brain Stats' },
  task_cancel:      { status: 'live',         category: 'ai',             emoji: '❌', labelKo: '작업 취소',        labelEn: 'Cancel Task' },

  // ── 전문가 페르소나 (Pro) ─────────────────────────────
  stock_analysis:   { status: 'live',         category: 'pro',            emoji: '📈', labelKo: '주식 분석',        labelEn: 'Stock Analysis' },
  medical_search:   { status: 'live',         category: 'pro',            emoji: '🏥', labelKo: '의료 검색',        labelEn: 'Medical Search' },
  legal_search:     { status: 'live',         category: 'pro',            emoji: '⚖️', labelKo: '법률 검색',        labelEn: 'Legal Search' },
  contract_review:  { status: 'live',         category: 'pro',            emoji: '📜', labelKo: '계약서 검토',      labelEn: 'Contract Review' },
  content_script:   { status: 'live',         category: 'pro',            emoji: '🎬', labelKo: '콘텐츠 스크립트',  labelEn: 'Content Script' },

  // ── 날씨·교통·번역 ───────────────────────────────────
  weather:          { status: 'live',         category: 'weather_travel', emoji: '🌤️', labelKo: '날씨',             labelEn: 'Weather' },
  travel_time:      { status: 'live',         category: 'weather_travel', emoji: '🚗', labelKo: '소요 시간',        labelEn: 'Travel Time' },
  translate:        { status: 'live',         category: 'weather_travel', emoji: '🌍', labelKo: '번역',             labelEn: 'Translate' },

  // ── 메타 ──────────────────────────────────────────────
  none:             { status: 'meta',         category: 'meta',           emoji: '💬', labelKo: 'LLM 대화',         labelEn: 'LLM Chat' },
} as const satisfies Record<Intent, IntentSpec>

/* ─────────────────────────────────────────────────────────── */
/* 유틸 함수                                                    */
/* ─────────────────────────────────────────────────────────── */

export function getIntentSpec(intent: Intent | string): IntentSpec {
  return (INTENT_REGISTRY as Record<string, IntentSpec>)[intent] ?? {
    status: 'not_implemented',
    category: 'meta',
    emoji: '❓',
    labelKo: intent,
    labelEn: intent,
  }
}

export function intentLabel(intent: Intent | string, lang: 'ko' | 'en' = 'ko'): string {
  const spec = getIntentSpec(intent)
  return lang === 'en' ? spec.labelEn : spec.labelKo
}

export function intentEmoji(intent: Intent | string): string {
  return getIntentSpec(intent).emoji
}

export function isWindowsOnly(intent: Intent | string): boolean {
  return getIntentSpec(intent).status === 'windows_only'
}

export function isImplemented(intent: Intent | string): boolean {
  const s = getIntentSpec(intent).status
  return s !== 'not_implemented'
}

/** 카테고리별 인텐트 그룹화 (UI 메뉴용) */
export function intentsByCategory(): Record<IntentCategory, Intent[]> {
  const out: Record<string, Intent[]> = {}
  for (const [intent, spec] of Object.entries(INTENT_REGISTRY) as [Intent, IntentSpec][]) {
    if (!out[spec.category]) out[spec.category] = []
    out[spec.category].push(intent)
  }
  return out as Record<IntentCategory, Intent[]>
}
