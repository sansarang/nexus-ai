/**
 * Go 백엔드 API 클라이언트 (port 17891)
 * ──────────────────────────────────────────────────────────────
 * [프로덕션 — 판매된 .exe]
 *   Go 백엔드가 항상 함께 실행됨.
 *   모든 응답은 실제 Windows API / PowerShell / WMI 결과.
 *   Mock 데이터 절대 사용 금지.
 *
 * [개발 환경 — Mac / 브라우저]
 *   Go 백엔드 미실행 → fetch 실패.
 *   safeCall() 래퍼가 mock 데이터로 fallback 허용.
 *   UI 개발/테스트 전용.
 * ──────────────────────────────────────────────────────────────
 */
const BASE = 'http://127.0.0.1:17891'
const TIMEOUT = 8000 // 프로덕션에서 느린 PowerShell 쿼리 고려

async function request<T>(method: string, path: string, body?: unknown, timeout = TIMEOUT): Promise<T> {
  const ctrl = new AbortController()
  const timer = setTimeout(() => ctrl.abort(), timeout)
  try {
    const res = await fetch(`${BASE}${path}`, {
      method,
      headers: body ? { 'Content-Type': 'application/json' } : {},
      body: body ? JSON.stringify(body) : undefined,
      signal: ctrl.signal,
    })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    return res.json() as Promise<T>
  } finally {
    clearTimeout(timer)
  }
}

/* ── 타입 정의 ── */
export interface StatsData {
  cpu: number
  mem: number
  mem_used_gb?: number
  mem_total_gb?: number
  disk: number
  disks?: Array<{ name: string; used_gb: number; free_gb: number; total_gb: number; pct: number }>
  cpu_temp: number
  gpu?: number
  gpu_name?: string
  net_up: number
  net_down: number
  timestamp: number
}

export interface RemoteToolInfo { name: string; status: string; risk: string }
export interface RemoteAccessResult {
  found: boolean; tools: RemoteToolInfo[]; rdp_open: boolean; score: number
}
export interface SuspProc { name: string; pid: number; cpu: number; mem_mb: number; risk: string; reason: string }
export interface OpenPortInfo { port: number; state: string; pid: number; risk: string; reason: string }
export interface ProcessSecurityResult {
  suspicious_processes: SuspProc[]; open_ports: OpenPortInfo[]; score: number
}
export interface StartupItem { name: string; command: string; location: string; risk: string }
export interface DefenderStatus {
  antivirus_enabled: boolean; realtime_protection: boolean
  quick_scan_age: number; full_scan_age: number; score: number; issues: string[]
}
export interface ProcItem { name: string; pid: number; cpu: number; mem_mb: number }
export interface NetworkAdapter { name: string; desc: string; speed_mbps: number; mac_address: string }
export interface DriverItem { name: string; status: string; class: string; risk: string }
export interface ProgramItem { name: string; version: string }
export interface FileResult { name: string; path: string; size_mb: number; mod_time: string }
export interface DupGroup { name: string; size_mb: number; paths: string[]; count: number }
export interface NoteItem { id: string; content: string; created: string }

export interface ScanIssue {
  id: string
  title: string
  description: string
  severity: 'high' | 'medium' | 'low'
  category: string
  fixable: boolean
}

export interface ScanResult {
  score: number
  issues: ScanIssue[]
}

export interface DailyReport {
  date: string
  pc_score: number
  cpu_avg: number
  mem_avg: number
  disk_free_gb: number
  recommendations: string[]
  predictions: Array<{ label: string; value: number; trend: 'up' | 'down' | 'stable' }>
}

export interface CleanResult {
  item: string
  freed_bytes: number
  error?: string
}

export interface RepairResult {
  success: boolean
  message: string
  freed: number
}

/* ── API 함수 ── */
export const backendAPI = {
  health:      ()                         => request<{ status: string }>('GET',  '/api/health'),
  stats:       ()                         => request<StatsData>          ('GET',  '/api/stats'),
  scan:        ()                         => request<ScanResult>         ('POST', '/api/scan'),
  repair:      (items: string[])          => request<RepairResult>       ('POST', '/api/repair',      { items }),
  clean:       (targets: string[])        => request<{ freed: number; message: string }>('POST', '/api/clean', { targets }),
  autoClean:   (items: string[])          => request<CleanResult[]>      ('POST', '/api/autoclean',  { items }),
  dailyReport: ()                         => request<DailyReport>        ('GET',  '/api/daily-report'),
  privacy:     (feature: string, enabled: boolean) =>
                                             request<{ success: boolean }>('POST', '/api/privacy', { feature, enabled }),
  openFolder:  (path: string) =>
                                             request<{ success: boolean; path: string; message: string }>('POST', '/api/folder/open', { path }),

  // ── 보안 상세 ──────────────────────────────────────────
  securityRemote:  () => request<RemoteAccessResult>      ('GET', '/api/security/remote'),
  securityProcs:   () => request<ProcessSecurityResult>   ('GET', '/api/security/processes'),
  securityHosts:   () => request<{ score: number; modified: boolean; entries: number; suspicious: string[] }>('GET', '/api/security/hosts'),
  securityStartup: () => request<{ items: StartupItem[]; total: number; suspicious_count: number }>('GET', '/api/security/startup'),
  securityDefender:() => request<DefenderStatus>          ('GET', '/api/security/defender'),
  securityAccounts:() => request<{ total: number; suspicious: unknown[]; suspicious_count: number; score: number }>('GET', '/api/security/accounts'),

  // ── 시스템 제어 ──────────────────────────────────────────
  volume:      (action: string, value?: number) => request<{ success?: boolean; volume?: number; message: string }>('POST', '/api/system/volume', { action, value }),
  brightness:  (action: string, value?: number) => request<{ success?: boolean; brightness?: number; message: string }>('POST', '/api/system/brightness', { action, value }),
  wifi:        (action: string) => request<{ success?: boolean; connected?: boolean; status?: string; message?: string }>('POST', '/api/system/wifi', { action }),
  power:       (action: string) => request<{ success: boolean; message: string }>('POST', '/api/system/power', { action }),
  launchApp:   (app: string)    => request<{ success: boolean; message: string }>('POST', '/api/system/launch', { app }),
  processTop:  () => request<{ by_cpu: ProcItem[]; by_mem: ProcItem[] }>('GET', '/api/processes/top'),

  // ── 고급 기능 ──────────────────────────────────────────
  drivers:       () => request<{ total: number; problematic: DriverItem[]; problem_count: number; score: number; message: string }>('GET', '/api/drivers'),
  registryClean: () => request<{ success: boolean; cleaned_keys: number; message: string }>('POST', '/api/registry/clean', {}),
  powerPlans:    () => request<{ plans: Array<{ name: string; guid: string; active: boolean }>; count: number }>('GET', '/api/power/plans'),
  setPowerPlan:  (name: string) => request<{ success: boolean; message: string }>('POST', '/api/power/plan', { name }),
  networkAnalysis: () => request<{ adapters: NetworkAdapter[]; dns_servers: string; public_ip: string; ping_ms: string; connected: boolean }>('GET', '/api/network/analysis'),
  restoreCreate: (description?: string) => request<{ success: boolean; message: string }>('POST', '/api/restore/create', { description }),
  diskCheck:     (drive?: string) => request<{ success: boolean; scheduled?: boolean; message: string }>('POST', '/api/disk/check', { drive }),
  browserClean:  (browsers?: string[], targets?: string[]) => request<{ results: unknown[]; total_mb: number; total_freed: string; message: string }>('POST', '/api/browser/clean', { browsers, targets }),
  programsList:  () => request<{ programs: ProgramItem[]; total: number }>('GET', '/api/programs'),
  bootAnalysis:  () => request<{ uptime_minutes: string; startup_count: string; recent_boots: unknown[]; message: string }>('GET', '/api/boot/analysis'),

  // ── 파일 관리 ──────────────────────────────────────────
  filesSearch:     (query: string, type?: string, path?: string) => request<{ results: FileResult[]; total: number; message: string }>('POST', '/api/files/search', { query, type, path }),
  filesOrganize:   (path?: string, mode?: string) => request<{ success: boolean; moved: number; message: string }>('POST', '/api/files/organize', { path, mode }),
  filesDuplicates: (path?: string) => request<{ groups: DupGroup[]; total_groups: number; waste_mb: number; waste: string; message: string }>('POST', '/api/files/duplicates', { path }),

  // ── 생산성 ──────────────────────────────────────────────
  focusMode:  (action: string, duration?: number) => request<{ success: boolean; active: boolean; message: string }>('POST', '/api/productivity/focus', { action, duration }),
  clipboard:  () => request<{ current: string; tip: string }>('GET', '/api/productivity/clipboard'),
  notes:      () => request<{ notes: NoteItem[]; total: number }>('GET', '/api/notes'),
  saveNote:   (content: string) => request<{ success: boolean; note: NoteItem; message: string }>('POST', '/api/notes', { content }),
  deleteNote: (id: string) => request<{ success: boolean; message: string }>('POST', '/api/notes', { delete_id: id }),

  // ── 업무 일지 ────────────────────────────────────────────
  journalToday:    () => request<Record<string, unknown>>('GET', '/api/journal/today'),
  journalGenerate: (date?: string, format?: string) =>
    request<{ success: boolean; path: string; filename: string; message: string; preview: string }>(
      'POST', '/api/journal/generate', { date, format }
    ),
  journalHistory:  () => request<{ history: unknown[]; days: number }>('GET', '/api/journal/history'),

  // ── 자동화 매크로 ─────────────────────────────────────────
  macroList:   () => request<{ macros: unknown[]; total: number }>('GET', '/api/macros'),
  macroCreate: (macro: unknown) => request<{ success: boolean; macro: unknown; message: string }>('POST', '/api/macros', macro),
  macroRun:    (id: string) => request<{ success: boolean; name: string; results: unknown[]; message: string }>('POST', '/api/macros/run', { id }),
  macroDelete: (id: string) => request<{ success: boolean; message: string }>('POST', '/api/macros/delete', { id }),
  macroParse:  (text: string) => request<{ macro: unknown; parsed: boolean; message: string }>('POST', '/api/macros/parse', { text }),

  // ── PC 건강 리포트 ────────────────────────────────────────
  reportGenerate: () => request<Record<string, unknown>>('GET', '/api/report/generate'),
  reportEmail:    (toEmail?: string) => request<{ success: boolean; message: string }>('POST', '/api/report/email', { to_email: toEmail }),
  reportSchedule: (schedule: string, toEmail: string, time?: string) =>
    request<{ success: boolean; message: string }>('POST', '/api/report/schedule', { schedule, to_email: toEmail, time }),
  emailConfigGet:  () => request<Record<string, unknown>>('GET', '/api/email/config'),
  emailConfigSet:  (cfg: unknown) => request<{ success: boolean; message: string }>('POST', '/api/email/config', cfg),

  // ── 문서 요약 ────────────────────────────────────────────
  docsSummary:      (filePath: string, useAI?: boolean) =>
    request<Record<string, unknown>>('POST', '/api/docs/summary', { file_path: filePath, use_ai: useAI }),
  docsExportReport: (file1: string, file2: string) =>
    request<{ success: boolean; path: string; message: string }>('POST', '/api/docs/export-report', { file1, file2 }),

  // ── 문서 비교 ────────────────────────────────────────────
  docsCompare: (file1: string, file2: string) =>
    request<DocCompareResult>('POST', '/api/docs/compare', { file1, file2 }),
  docsFind: (query: string, fileType?: string, maxDays?: number, folder?: string) =>
    request<{ results: DocFindResult[]; total: number; message: string }>(
      'POST', '/api/docs/find', { query, file_type: fileType, max_days: maxDays, folder }
    ),

  // ── Deep Search ──────────────────────────────────────────
  deepSearch: (query: string, searchIn?: string, folder?: string, fileType?: string, maxResults?: number) =>
    request<{ results: DeepSearchResult[]; total: number; query: string; message: string }>(
      'POST', '/api/search/deep', { query, search_in: searchIn, folder, file_type: fileType, max_results: maxResults }
    ),

  // ── Vision & OCR ─────────────────────────────────────────
  screenshot:     (withOCR?: boolean) =>
    request<{ success: boolean; base64: string; width: number; height: number; mime: string; ocr_text?: string; captured: string }>(
      'POST', '/api/vision/screenshot', { with_ocr: withOCR }
    ),
  activeWindow:   () =>
    request<{ title: string; screen_info: string; timestamp: string }>('GET', '/api/vision/active-window'),
  ocrClipboard:   () =>
    request<{ success: boolean; text: string; message: string }>('POST', '/api/vision/ocr-clipboard', {}),

  // ── LLM (Go 백엔드 직접 Perplexity/Claude 호출) ──────────
  llmConfigGet: () =>
    request<{ perplexity_configured: boolean; claude_configured: boolean; models: Record<string, string> }>(
      'GET', '/api/llm/config'
    ),
  llmConfigSet: (pplxKey?: string, claudeKey?: string, tavilyKey?: string) =>
    request<{ success: boolean; message: string }>(
      'POST', '/api/llm/config', { perplexity_key: pplxKey, claude_key: claudeKey, tavily_key: tavilyKey }
    ),
  llmChat: (messages: Array<{ role: string; content: string }>, options?: { maxTokens?: number; jsonMode?: boolean; fast?: boolean }) =>
    request<{ success: boolean; answer: string; model: string; tokens: number }>(
      'POST', '/api/llm/chat', { messages, max_tokens: options?.maxTokens, json_mode: options?.jsonMode, fast: options?.fast }
    ),
  llmVision: (question?: string, imageBase64?: string, mime?: string) =>
    request<{ success: boolean; answer: string; question: string; model: string; width: number; height: number }>(
      'POST', '/api/llm/vision', { question, image_base64: imageBase64, mime }
    ),
  llmDocSummary: (filePath: string, question?: string) =>
    request<{ success: boolean; summary: string; file: string }>(
      'POST', '/api/llm/doc-summary', { file_path: filePath, question }
    ),
  llmDocCompare: (fileA: string, fileB: string, focus?: 'numbers' | 'changes' | 'both') =>
    request<{ success: boolean; result: LLMDocCompareResult; file_a: string; file_b: string }>(
      'POST', '/api/llm/doc-compare', { file_a: fileA, file_b: fileB, focus }
    ),
  llmDeepSearch: (query: string, folder?: string, maxResults?: number) =>
    request<{ success: boolean; results: DeepSearchResult[]; total: number; keywords_used: string[]; ai_enhanced: boolean }>(
      'POST', '/api/llm/deep-search', { query, folder, max_results: maxResults }
    ),
  llmDeepSearchWeb: (query: string, maxResults?: number) =>
    request<{ success: boolean; query: string; summary: string; items: Array<{ title: string; url: string; source?: string }>; total: number }>(
      'POST', '/api/llm/deep-search-web', { query, max_results: maxResults ?? 10 }
    ),

  // ── Browser Agent (chromedp) ──────────────────────────────
  browserStatus: () =>
    request<{ active: boolean; chrome_installed: boolean; message: string }>(
      'GET', '/api/browser/status'
    ),
  browserNavigate: (url: string, waitFor?: string) =>
    request<{ success: boolean; title: string; url: string; message: string }>(
      'POST', '/api/browser/navigate', { url, wait_for: waitFor }
    ),
  browserExtract: (selector?: string, mode?: 'text' | 'html' | 'links' | 'table', url?: string) =>
    request<{ success: boolean; content: string; selector: string; mode: string; length: number }>(
      'POST', '/api/browser/extract', { selector, mode, url }
    ),
  browserClick: (selector?: string, text?: string) =>
    request<{ success: boolean; message: string }>(
      'POST', '/api/browser/click', { selector, text }
    ),
  browserFill: (selector: string, value: string, submit?: boolean) =>
    request<{ success: boolean; message: string }>(
      'POST', '/api/browser/fill', { selector, value, submit }
    ),
  browserScreenshot: (selector?: string) =>
    request<{ success: boolean; base64: string; mime: string; title: string; url: string; size_kb: number }>(
      'POST', '/api/browser/screenshot', { selector }
    ),
  browserAgent: (command: string, maxSteps?: number) =>
    request<BrowserAgentResult>(
      'POST', '/api/browser/agent', { command, max_steps: maxSteps }
    ),
  browserClose: () =>
    request<{ success: boolean; message: string }>('POST', '/api/browser/close', {}),
}

/* ── 타입 정의 ── */

export interface LLMDocDiff {
  type: 'added' | 'deleted' | 'modified' | 'number_mismatch'
  location: string
  description: string
  a_value: string
  b_value: string
  severity: 'low' | 'medium' | 'high'
}

export interface LLMDocCompareResult {
  summary: string
  total_differences: number
  differences: LLMDocDiff[]
  risk_level: 'low' | 'medium' | 'high'
  recommendation: string
}

export interface BrowserStepResult {
  step: number
  action: string
  description: string
  success: boolean
  data?: string
  error?: string
}

export interface BrowserAgentResult {
  success: boolean
  goal: string
  steps_executed: number
  steps: BrowserStepResult[]
  summary: string
  command: string
}

/* ── 기존 타입 정의 ── */
export interface DiffLine {
  type: 'equal' | 'added' | 'removed' | 'changed'
  old?: string
  new?: string
  line: number
}
export interface NumberMismatch {
  context: string
  old_val: string
  new_val: string
  change_pct: number
}
export interface DocCompareResult {
  file1_name: string
  file2_name: string
  file1_size: string
  file2_size: string
  similarity_pct: number
  added_count: number
  removed_count: number
  changed_count: number
  diff: DiffLine[]
  number_mismatches: NumberMismatch[]
  summary: string
}
export interface DocFindResult {
  name: string
  path: string
  size_mb: number
  mod_time: string
  match: 'filename' | 'content' | 'both'
  snippet?: string
}
export interface DeepSearchResult {
  name: string
  path: string
  ext: string
  size_mb: number
  mod_time: string
  snippet: string
  score: number
}

/* ── 신규 타입 ── */

export interface SmartAgentStep {
  step: number
  action: string
  description: string
  success: boolean
  data?: unknown
  error?: string
  duration: string
}

export interface SmartAgentResult {
  success: boolean
  command: string
  goal: string
  steps: SmartAgentStep[]
  summary: string
  data_rows?: string[][]
  excel_path?: string
  blocked: boolean
  block_reason?: string
  duration: string
}

export interface PriceItem {
  site: string
  name: string
  price: string
  link: string
  blocked: boolean
}

export interface CollectPriceResult {
  success: boolean
  query: string
  results: PriceItem[]
  total: number
  summary: string
  excel_path?: string
}

export interface NewsArticle {
  title: string
  url: string
}

export interface NewsCollectResult {
  success: boolean
  query: string
  articles: NewsArticle[]
  total: number
  summary: string
}

export interface ScheduledTask {
  id: string
  name: string
  command: string
  action: string
  cron_expr: string
  next_run: string
  last_run: string
  last_result: string
  active: boolean
  created_at: string
  run_count: number
}

export interface SchedulerAddResult {
  success: boolean
  task: ScheduledTask
  next_run_kr: string
  message: string
}

export interface SchedulerParseResult {
  success: boolean
  cron_expr: string
  task_name: string
  action: string
  params: Record<string, unknown>
  next_run: string
  next_run_kr: string
}

export interface AgentMemoryEntry {
  id: string
  timestamp: string
  type: string
  command: string
  result: string
  success: boolean
  tags?: string[]
}

export interface MemoryStats {
  success: boolean
  total: number
  by_type: Record<string, number>
  success_rate: number
}

export interface ExcelFileInfo {
  name: string
  path: string
  size: number
  modified: string
}

/* ── ★ 핵심: 검색 → PDF 자동 생성 ── */
export interface SearchPDFResult {
  success: boolean
  pdf_path: string
  html_path: string
  query: string
  item_count: number
  summary: string
  duration: string
  error?: string
  items?: Array<Record<string, string>>
}

export const searchAndPDF = (query: string, maxItems = 5, savePath = '', openAfter = true) =>
  request<SearchPDFResult>('POST', '/api/browser/search-and-pdf', {
    query, max_items: maxItems, save_path: savePath, open_after: openAfter,
  }, 180000) // 3분 타임아웃 (브라우저 작업)

/* ── 신규 API 메서드 ── */
export const browserSmartAgent = (command: string, maxResults = 10, saveExcel = false, sessionKey = '') =>
  request<SmartAgentResult>('POST', '/api/browser/smart-agent', { command, max_results: maxResults, save_excel: saveExcel, session_key: sessionKey })

export const browserCollectPrice = (productQuery: string, sites?: string[], maxPerSite = 5, saveExcel = true) =>
  request<CollectPriceResult>('POST', '/api/browser/collect-price', { product_query: productQuery, sites, max_per_site: maxPerSite, save_excel: saveExcel })

export const browserNewsCollect = (query: string, site = 'naver.com', maxItems = 10) =>
  request<NewsCollectResult>('POST', '/api/browser/news-collect', { query, site, max_items: maxItems })

export const browserLoginSession = (url: string, username: string, password: string, sessionKey: string, opts?: { username_selector?: string; password_selector?: string; submit_selector?: string }) =>
  request<{ success: boolean; session_key: string; message: string }>('POST', '/api/browser/login-session', { url, username, password, session_key: sessionKey, ...opts })

export const excelSave = (data: string[][], title: string, filename?: string, savePath?: string) =>
  request<{ success: boolean; path: string; rows: number; message: string }>('POST', '/api/excel/save', { data, title, filename, save_path: savePath })

export const excelList = () =>
  request<{ success: boolean; files: ExcelFileInfo[]; total: number }>('GET', '/api/excel/list')

export const schedulerAdd = (command: string, useWindows = false) =>
  request<SchedulerAddResult>('POST', '/api/scheduler/add', { command, use_windows: useWindows })

export const schedulerList = () =>
  request<{ success: boolean; tasks: ScheduledTask[]; total: number }>('GET', '/api/scheduler/list')

export const schedulerDelete = (id: string) =>
  request<{ success: boolean; message: string }>('DELETE', `/api/scheduler/delete?id=${encodeURIComponent(id)}`)

export const schedulerRunNow = (id: string) =>
  request<{ success: boolean; message: string }>('POST', `/api/scheduler/run-now?id=${encodeURIComponent(id)}`)

export const schedulerParse = (command: string) =>
  request<SchedulerParseResult>('POST', '/api/scheduler/parse', { command })

export const memoryList = (type?: string, keyword?: string, limit = 20) => {
  const params = new URLSearchParams()
  if (type) params.set('type', type)
  if (keyword) params.set('keyword', keyword)
  params.set('limit', String(limit))
  return request<{ success: boolean; entries: AgentMemoryEntry[]; total: number }>('GET', `/api/memory/list?${params}`)
}

export const memorySearch = (keyword: string, type?: string, limit = 10) =>
  request<{ success: boolean; entries: AgentMemoryEntry[]; total: number; summary: string }>('POST', '/api/memory/search', { keyword, type, limit })

export const memoryClear = (type?: string) =>
  request<{ success: boolean; message: string }>('DELETE', type ? `/api/memory/clear?type=${type}` : '/api/memory/clear')

export const memoryStats = () =>
  request<MemoryStats>('GET', '/api/memory/stats')

/* ── ★ 핵심: 자연어 명령 통합 처리 ── */
export interface CommandResult {
  success: boolean
  message: string
  action: string
  result: unknown
  duration: string
  // clarify 멀티턴 필드
  needs_clarify?: boolean
  clarify_question?: string
  pending_intent?: string
  pending_params?: Record<string, unknown>
}

/**
 * POST /api/command
 * 사용자의 자연어 메시지를 받아 Perplexity AI가 의도를 파악하고
 * 알맞은 백엔드 함수를 실행 후 결과를 반환합니다.
 * 멀티턴 clarify 컨텍스트를 지원합니다.
 */
export const sendCommand = (
  message: string,
  options?: {
    context?: string
    pendingIntent?: string
    pendingParams?: Record<string, unknown>
    pendingQuestion?: string
    history?: Array<{ role: 'user' | 'assistant'; content: string }>
  }
) =>
  request<CommandResult>('POST', '/api/command', {
    message,
    context: options?.context,
    pending_intent: options?.pendingIntent,
    pending_params: options?.pendingParams,
    pending_question: options?.pendingQuestion,
    history: options?.history ?? [],
  }, 60000)

/* ── 🖥️ Windows Recall ── */
export const recallCapture = () =>
  request<{ success: boolean; timestamp: string; ocr_text: string; message: string }>('POST', '/api/recall/capture', {})
export const recallSearch  = (query: string) =>
  request<{ success: boolean; results: Array<{ timestamp: string; snippet: string; file_path: string }>; total: number; message: string }>('POST', '/api/recall/search', { query })

/* ── 🎙️ 회의 어시스턴트 ── */
export const meetingStart      = () =>
  request<{ success: boolean; file_path: string; message: string }>('POST', '/api/meeting/start', {})
export const meetingStop       = () =>
  request<{ success: boolean; file_path: string; duration_sec: number; message: string }>('POST', '/api/meeting/stop', {})
export const meetingTranscribe = (filePath: string) =>
  request<{ success: boolean; text: string; duration_sec: number; message: string }>('POST', '/api/meeting/transcribe', { file_path: filePath })
export const meetingList       = () =>
  request<{ success: boolean; meetings: Array<{ file: string; timestamp: string; size_mb: number }>; total: number }>('GET', '/api/meeting/list')
export const meetingSummarize  = (text: string) =>
  request<{ success: boolean; summary: string; action_items: string[]; decisions: string[]; message: string }>('POST', '/api/meeting/summarize', { text })

/* ── ⌨️ 음성 받아쓰기 ── */
export const dictationType  = (text: string, app?: string) =>
  request<{ success: boolean; typed_chars: number; message: string }>('POST', '/api/dictation/type', { text, app })
export const dictationPaste = (text: string) =>
  request<{ success: boolean; message: string }>('POST', '/api/dictation/paste', { text })


/* ── 🌤️ 날씨 + 교통 ── */
export const weatherGet  = (city = '서울') =>
  request<{ success: boolean; city: string; temp_c: number; feels_like: number; condition: string; humidity: number; wind_kmh: number; forecast: Array<{ date: string; max: number; min: number; condition: string }>; message: string }>('GET', `/api/weather?city=${encodeURIComponent(city)}`)
export const travelTime  = (origin: string, destination: string, departureTime?: string) =>
  request<{ success: boolean; origin: string; destination: string; distance_km: number; duration_min: number; departure_time: string; arrival_time: string; message: string }>('POST', '/api/travel/time', { origin, destination, departure_time: departureTime })

/* ── 📅 캘린더 ── */
export interface CalendarEvent {
  subject: string; start: string; end: string; location: string; organizer: string; is_all_day: boolean
}
export const calendarToday  = () => request<{ success: boolean; events: CalendarEvent[]; total: number; message: string }>('GET', '/api/calendar/today')
export const calendarWeek   = () => request<{ success: boolean; events: CalendarEvent[]; total: number; message: string }>('GET', '/api/calendar/week')
export const calendarAdd    = (subject: string, start?: string, end?: string, location?: string) =>
  request<{ success: boolean; message: string }>('POST', '/api/calendar/add', { subject, start, end, location })

/* ── 📧 이메일 ── */
export interface EmailItem {
  subject: string; sender: string; received_at: string; body: string; is_read: boolean; has_attachments: boolean
}
export const emailInbox     = (limit = 10) => request<{ success: boolean; emails: EmailItem[]; total: number; unread: number; message: string }>('GET', `/api/email/inbox?limit=${limit}`)
export const emailSend      = (to: string, subject: string, body: string) => request<{ success: boolean; message: string }>('POST', '/api/email/send', { to, subject, body })
export const emailSummarize = () => request<{ success: boolean; emails: EmailItem[]; summary: string; message: string }>('POST', '/api/email/summarize', {})

/* ── 🦠 VirusTotal ── */
export interface VTResult {
  success: boolean; file_path: string; file_hash: string; malicious: number; suspicious: number
  clean: number; total_scans: number; permalink: string; safe_score: number; verdict: string; message: string
}
export const virusTotalCheck = (filePath: string, apiKey: string) =>
  request<VTResult>('POST', '/api/security/virustotal', { file_path: filePath, api_key: apiKey })

/* ── 📊 성능 이력 ── */
export interface PerfSnapshot {
  timestamp: string; cpu: number; mem: number; disk: number; cpu_temp: number; gpu?: number
}
export interface DaySummary {
  date: string; avg_cpu: number; max_cpu: number; avg_mem: number; max_mem: number; avg_temp: number; max_temp: number; samples: number
}
export const historyStats   = (days = 7) => request<{ success: boolean; days: number; total_samples: number; snapshots: PerfSnapshot[]; daily_summary: DaySummary[]; avg_cpu: number; avg_mem: number; cpu_trend: string; message: string }>('GET', `/api/history/stats?days=${days}`)
export const historyAnomalies = () => request<{ success: boolean; anomalies: Array<{ timestamp: string; type: string; value: number; avg_value: number; diff_pct: number; message: string }>; avg_cpu: number; avg_mem: number; message: string }>('GET', '/api/history/anomalies')

/* ── 🔧 시스템 확장 ── */
export const processKill     = (pid?: number, name?: string) => request<{ success: boolean; name: string; message: string }>('POST', '/api/process/kill', { pid, name })
export const appPermissions  = (app?: string) => request<{ success: boolean; permissions: Record<string, unknown>; message: string }>('GET', app ? `/api/app/permissions?app=${encodeURIComponent(app)}` : '/api/app/permissions')
export const windowsUpdates  = () => request<{ success: boolean; count: number; updates: Array<{ title: string; kb: string; severity: string; size_mb: number; important: boolean }>; message: string }>('GET', '/api/system/updates')
export const gpuStats        = () => request<{ success: boolean; gpus: Array<{ name: string; usage_pct: number; temp_c: number; mem_used_mb: number; mem_total_mb: number; driver_ver: string; status: string }>; message: string }>('GET', '/api/gpu/stats')

/* ── 🌐 브라우저 래퍼 (price/news 바로 실행) ── */
export const priceCompare = (query: string) =>
  request<CollectPriceResult>('POST', '/api/browser/collect-price', {
    product_query: query,
    sites: ['coupang.com', 'naver.com', '11st.co.kr', 'gmarket.co.kr'],
    max_per_site: 5,
    save_excel: false,
  })

export const youtubeSearch = (query: string) =>
  request<NewsCollectResult>('POST', '/api/browser/news-collect', { query, site: 'youtube.com', max_items: 8 })

export const tiktokSearch = (query: string) =>
  request<NewsCollectResult>('POST', '/api/browser/news-collect', { query, site: 'tiktok.com', max_items: 8 })

export const naverShoppingSearch = (query: string) =>
  request<CollectPriceResult>('POST', '/api/browser/collect-price', {
    product_query: query,
    sites: ['naver.com'],
    max_per_site: 8,
    save_excel: false,
  })

export const coupangSearch = (query: string) =>
  request<CollectPriceResult>('POST', '/api/browser/collect-price', {
    product_query: query,
    sites: ['coupang.com'],
    max_per_site: 8,
    save_excel: false,
  })

export const newsSearch = (query: string) =>
  request<NewsCollectResult>('POST', '/api/browser/news-collect', { query, site: 'naver.com', max_items: 8 })

export const videoDownload = (url: string, quality = 'best', savePath = '') =>
  request<{ success: boolean; url: string; save_path: string; message: string; install_url?: string; output?: string }>(
    'POST', '/api/browser/video-download', { url, quality, save_path: savePath }, 300000 // 5분 타임아웃
  )

/* ── 🎭 AI 멀티 페르소나 ── */
export interface PersonaDef {
  id: string; name: string; emoji: string; description: string; color: string; system_prompt: string
}
export const personaList = () =>
  request<{ personas: PersonaDef[]; current: string }>('GET', '/api/persona/list')
export const personaSet = (id: string) =>
  request<{ ok: boolean; persona: PersonaDef; message: string }>('POST', '/api/persona/set', { id })
export const personaCurrent = () =>
  request<{ persona: PersonaDef }>('GET', '/api/persona/current')

/* ── 🧠 Second Brain ── */
export interface BrainEntry {
  id: string; source: string; title: string; content: string; tags: string[]; timestamp: string
}
export interface BrainSearchResult { entry: BrainEntry; score: number; highlight: string }
export const brainSearch = (query: string, limit = 8, source = '') =>
  request<{ results: BrainSearchResult[]; total: number; summary: string; query: string }>(
    'POST', '/api/brain/search', { query, limit, source })
export const brainRebuild = () =>
  request<{ ok: boolean; message: string }>('POST', '/api/brain/rebuild', {})
export const brainStats = () =>
  request<{ total: number; by_source: Record<string, number>; updated_at: string }>('GET', '/api/brain/stats')

/* ── ⚡ Auto Workflow ── */
export interface WorkflowStep {
  step: number; description: string; api_endpoint: string; method: string; params: unknown
  status: string; result: string
}
export interface WorkflowResult {
  goal: string; steps: WorkflowStep[]; summary: string; ok: boolean
}
export const workflowPlan = (goal: string) =>
  request<WorkflowResult>('POST', '/api/workflow/plan', { goal })
export const workflowRun = (goal: string) =>
  request<WorkflowResult>('POST', '/api/workflow/run', { goal })

/* ── 🎬 Live Caption ── */
export interface CaptionEntry { text: string; translated?: string; timestamp: string; lang: string }
export const captionStart = (lang = 'ko') =>
  request<{ ok: boolean; message: string }>('POST', '/api/caption/start', { lang })
export const captionStop = () =>
  request<{ ok: boolean; message: string; entries: number }>('POST', '/api/caption/stop', {})
export const captionLatest = () =>
  request<{ entries: CaptionEntry[]; running: boolean; total: number }>('GET', '/api/caption/latest')

/* ── Desktop Computer Use Agent ── */
export const desktopAgentRun = (goal: string, requireApproval = true, maxSteps = 20) =>
  request<{ success: boolean; task_id: string; message: string }>('POST', '/api/agent/desktop/run', { goal, require_approval: requireApproval, max_steps: maxSteps })
export const desktopClick    = (x: number, y: number, button: 'left'|'right'|'double' = 'left') =>
  request<{ success: boolean; message: string }>('POST', '/api/agent/desktop/click', { x, y, button })
export const desktopType     = (text: string) =>
  request<{ success: boolean; message: string }>('POST', '/api/agent/desktop/type', { text })
export const desktopKey      = (key: string) =>
  request<{ success: boolean; message: string }>('POST', '/api/agent/desktop/key', { key })
export const desktopScroll   = (x: number, y: number, direction: 'up'|'down', amount = 3) =>
  request<{ success: boolean }>('POST', '/api/agent/desktop/scroll', { x, y, direction, amount })
export const desktopApprove  = (taskId: string, approved: boolean) =>
  request<{ success: boolean; approved: boolean }>('POST', '/api/agent/desktop/approve', { task_id: taskId, approved })
export const desktopStatus   = () =>
  request<{ success: boolean; active_title: string; cursor_x: number; cursor_y: number; screen_w: number; screen_h: number }>('GET', '/api/agent/desktop/status')
export const desktopScreenshot = (ocr = false) =>
  request<{ success: boolean; base64: string; width: number; height: number; ocr_text?: string }>('GET', `/api/agent/desktop/screenshot?ocr=${ocr}`)

/* ── Task Queue ── */
export const taskList   = () => request<{ tasks: unknown[]; count: number }>('GET', '/api/tasks/list')
export const taskCancel = (id: string) => request<{ success: boolean; message: string }>('POST', '/api/tasks/cancel', { id })

/* ── Multi-Agent ── */
export const multiAgentRun  = (goal: string) =>
  request<{ success: boolean; task_id: string; message: string }>('POST', '/api/agent/multi/run', { goal })
export const multiAgentPlan = (goal: string) =>
  request<{ success: boolean; plan: unknown }>('POST', '/api/agent/multi/plan', { goal })
export const agentList      = () =>
  request<{ agents: unknown[]; count: number }>('GET', '/api/agent/multi/agents')

/* ── Email Deep Agency ── */
export const emailClassify      = (limit = 20) =>
  request<{ success: boolean; classified: unknown[]; counts: Record<string, number>; message: string }>('POST', '/api/email/classify', { limit })
export const emailDraftReply    = (subject: string, sender: string, body: string, tone = 'formal') =>
  request<{ success: boolean; draft: string; message: string }>('POST', '/api/email/draft-reply', { subject, sender, body, tone })
export const emailExtractEvents = (subject: string, body: string, sender: string) =>
  request<{ success: boolean; result: unknown; message: string }>('POST', '/api/email/extract-events', { subject, body, sender })
export const calendarFindSlot   = (durationMin = 60, preferTime = 'morning', withinDays = 7) =>
  request<{ success: boolean; slots: unknown[]; message: string }>('POST', '/api/calendar/find-slot', { duration_min: durationMin, prefer_time: preferTime, within_days: withinDays })
export const calendarSmartAdd   = (text: string) =>
  request<{ success: boolean; event: unknown; message: string; confirm_needed: boolean }>('POST', '/api/calendar/smart-add', { text })

/* ── Visual Workflow Builder ── */
export const workflowList        = () => request<{ workflows: unknown[]; count: number }>('GET', '/api/workflow/list')
export const workflowSave        = (workflow: unknown) => request<{ success: boolean; id: string; message: string }>('POST', '/api/workflow/save', workflow)
export const workflowDeleteById  = (id: string) => request<{ success: boolean; message: string }>('DELETE', `/api/workflow/delete?id=${id}`)
export const workflowRunNow      = (id: string) => request<{ success: boolean; task_id: string; message: string }>('POST', '/api/workflow/run-now', { id })
export const workflowFromText    = (text: string) => request<{ success: boolean; workflow: unknown; message: string }>('POST', '/api/workflow/from-text', { text })
export const workflowTemplates   = () => request<{ templates: unknown[]; count: number }>('GET', '/api/workflow/templates')

/* ── Privacy & Sandbox ── */
export const auditLog      = (limit = 50) => request<{ success: boolean; entries: unknown[]; total: number }>('GET', `/api/security/audit?limit=${limit}`)
export const ollamaConfig  = () => request<{ enabled: boolean; url: string; model: string }>('GET', '/api/ollama/config')
export const ollamaSetConfig = (cfg: { enabled: boolean; url?: string; model?: string }) =>
  request<{ success: boolean; message: string }>('POST', '/api/ollama/config', cfg)
export const ollamaTest    = () => request<{ success: boolean; response: string; message: string }>('POST', '/api/ollama/test', {})
export const ollamaModels  = () => request<{ success: boolean; models: unknown }>('GET', '/api/ollama/models')

/* ── Briefing ── */
export const briefingNow    = () => request<{ success: boolean; task_id: string; message: string }>('POST', '/api/briefing/now', {})
export const briefingConfig = () => request<{ enabled: boolean; hour: number; weather_city: string }>('GET', '/api/briefing/config')
export const briefingSetConfig = (cfg: { enabled: boolean; hour?: number; weather_city?: string }) =>
  request<{ success: boolean; message: string }>('POST', '/api/briefing/config', cfg)

/* ── 사이트 직접 검색 (LLM 우회) ── */
export const siteSearch = (query: string, site: string, maxItems = 8) =>
  request<{ success: boolean; query: string; site: string; summary: string; results: Array<{ name: string; link: string; price: string; site: string }>; total: number }>(
    'POST', '/api/site-search', { query, site, max_items: maxItems }
  )

/* ── 개발환경 더미 데이터 ── */
export function mockStats(): StatsData {
  return {
    cpu:       Math.random() * 40 + 20,
    mem:       Math.random() * 30 + 40,
    disk:      Math.random() * 20 + 55,
    cpu_temp:  Math.random() * 20 + 45,
    net_up:    Math.random() * 500,
    net_down:  Math.random() * 2000,
    timestamp: Math.floor(Date.now() / 1000),
  }
}

export function mockScan(): ScanResult {
  return {
    score: 85,
    issues: [
      {
        id: 'temp-files',
        title: '1.2GB 임시 파일이 쌓여있어요',
        description: '정리하면 디스크 공간을 확보할 수 있어요',
        severity: 'medium',
        category: 'clean',
        fixable: true,
      },
    ],
  }
}

export function mockDailyReport(): DailyReport {
  return {
    date: new Date().toISOString().slice(0, 10),
    pc_score: 82,
    cpu_avg: 34,
    mem_avg: 58,
    disk_free_gb: 120,
    recommendations: [
      '오늘도 PC 상태가 양호합니다.',
      '주기적인 임시 파일 정리를 권장합니다.',
    ],
    predictions: [
      { label: 'CPU 사용률',  value: 38,  trend: 'up'     },
      { label: '메모리 사용률', value: 60, trend: 'stable' },
      { label: '디스크 여유', value: 118,  trend: 'down'   },
    ],
  }
}
