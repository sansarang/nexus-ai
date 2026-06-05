/**
 * cardRegistry.ts — 49개 카드 자동 라우팅 (사장님 설계: 백엔드 판단 → 카드 자동 선택)
 *
 * 흐름:
 *   백엔드 응답 { action, result, card_type? } 받음
 *   → resolveCardType(action, result, card_type) 으로 카드 타입 결정
 *      1순위: 백엔드가 card_type 명시했으면 그대로 사용
 *      2순위: ACTION_TO_CARD 매핑표 (액션명 → 카드 타입)
 *      3순위: result 구조 자동 감지 (items 있으면 web_search, results 있으면 ...)
 *      4순위: null (텍스트만 표시)
 *   → buildCardFromResult(card_type, result, query) 로 카드 데이터 생성
 *
 * 장점:
 *   - 49개 카드 인프라 다 활용 가능 (현재 18개만 활용)
 *   - 새 액션 추가 시 매핑표 1줄 추가 (renderCommandResult switch case X)
 *   - 백엔드가 card_type 명시하면 매핑표 무시하고 정확히 라우팅
 */

import type { InlineCardData } from '../InlineCards'
import type { InlineCardData2 } from '../InlineCards2'
import type { InlineCard3Data } from '../InlineCards3'
import type { InlineCard4Data } from '../InlineCards4'
import type { InlineCard5Data } from '../InlineCards5'

// ── 액션 → 기본 카드 타입 매핑표 ──
// 백엔드 응답의 action 필드만 보고 자동으로 어떤 카드를 그릴지 결정
export const ACTION_TO_CARD: Record<string, string> = {
  // 시스템 진단
  stats:           'pc_status',
  pc_status:       'pc_status',
  scan:            'scan_result',
  security_scan:   'scan_result',
  full_scan:       'scan_result',
  daily_report:    'daily_report',
  pc_report:       'pc_report',
  health_report:   'pc_report',
  clean:           'clean_result',
  smart_organize:  'smart_organize',
  repair:          'repair_result',

  // 시스템 보안/네트워크
  remote_access:    'remote_access',
  process_security: 'process_security',
  defender:         'defender',
  startup_items:    'startup_items',
  process_top:      'process_top',
  network_analysis: 'network',
  driver_check:     'drivers',
  programs_list:    'programs_list',
  boot_analysis:    'boot_analysis',

  // 검색 & 컨텐츠
  web_search:      'web_search',
  news_search:     'news_search',
  youtube_search:  'youtube',
  video_search:    'youtube',
  price_compare:   'price_compare',
  multi_action:    'price_compare',  // 멀티액션도 price_compare 카드 활용
  deep_search:     'deep_search',
  deep_research:   'deep_search',

  // 파일/문서
  file_search:     'file_search',
  file_duplicates: 'duplicates',
  doc_compare:     'doc_compare',
  doc_find:        'doc_find',
  doc_summary:     'doc_summary',
  vision:          'vision_result',
  vision_screen:   'vision_result',
  vision_ocr:      'vision_ocr',

  // 생산성
  notes:           'notes',
  note:            'notes',
  weather:         'weather_card',
  email_inbox:     'email_list',
  email_summarize: 'email_list',
  email_classify:  'email_list',
  calendar_today:  'timeline',
  calendar_week:   'timeline',

  // 매크로/자동화
  macro_list:      'macro_list',
  macro_create:    'macro_created',
  macro_run:       'macro_run',
  journal_today:   'journal_today',
  journal_history: 'journal_history',

  // 시스템 제어 (시각 카드 가벼움)
  focus_mode:      'focus_mode',
  open_folder:     'folder_open',

  // Excel/문서 자동 생성 (사장님 원칙: 데이터 없으면 LLM이 만든다)
  excel_auto_create: 'file_result',
  create_excel:      'file_result',
  make_excel:        'file_result',
  excel_save:        'file_result',
  doc_auto_create:   'file_result',
  create_doc:        'file_result',
  make_doc:          'file_result',
  pdf_auto_create:   'file_result',
  create_pdf:        'file_result',
  make_pdf:          'file_result',
  // Excel 분석 (사용자 데이터 활용)
  excel_analyze:     'file_result',
  analyze_excel:     'file_result',
  // 영상 워크플로
  video_workflow:           'file_result',
  video_summary:            'file_result',
  video_download_summary:   'file_result',
  video_transcript:         'doc_summary',

  // ── 신규 추가 (v2.8.1) ──────────────────────────────────────
  // 사업자 조회 (국세청 NTS)
  business_lookup:     'business_card',

  // 주식/금융
  stock_price:         'stock_card',
  stock_analysis:      'stock_card',

  // 번역
  translate:           'translate_result',

  // 지도/경로
  map_search:          'map_result',
  directions:          'map_result',
  travel_time:         'map_result',
  place_view:          'map_result',

  // 이메일 작성/캘린더 이벤트
  email_draft_reply:   'email_draft',
  email_send:          'email_draft',
  calendar_add:        'calendar_event',
  calendar_find_slot:  'calendar_event',
  calendar_smart_add:  'calendar_event',

  // 워크플로우/자동화 결과
  workflow_run:        'workflow_result',
  workflow_run_now:    'workflow_result',

  // 브리핑 (Proactive AI)
  briefing_now:        'briefing_card',
  daily_briefing:      'briefing_card',

  // 미팅 요약/전사
  meeting_summarize:   'meeting_summary',
  meeting_transcribe:  'meeting_summary',
  meeting_stop:        'meeting_summary',

  // 메모리/리콜 검색
  memory_search:       'memory_card',
  recall_search:       'memory_card',

  // 시스템 명령 (볼륨/밝기/앱 실행 등)
  launch_app:          'cmd_result',
  process_kill:        'cmd_result',
  power_action:        'cmd_result',
  volume_control:      'cmd_result',
  brightness:          'cmd_result',
  wifi_toggle:         'cmd_result',
  power_plan:          'cmd_result',
  system_updates:      'cmd_result',
  clipboard_history:   'cmd_result',

  // 보안 경보 (VirusTotal / Shodan)
  virustotal:          'security_alert',
  shodan:              'security_alert',

  // 소셜 미디어
  tiktok_search:       'tiktok_result',
  tiktok_trending:     'tiktok_result',
  reddit_search:       'reddit_result',
  netflix_trending:    'media_result',
  ytmusic_search:      'youtube',

  // Agent 실행 결과
  desktop_agent:       'agent_result',
  multi_agent:         'agent_result',
  browser_agent:       'agent_result',
}

// 카드 슬롯 분류 — 어느 inlineCard{1-5} 슬롯에 들어가는지
// Cards1 = card, Cards2 = card2, Cards3 = card3, Cards4 = card4, Cards5 = card5
export const CARD_SLOT: Record<string, 1 | 2 | 3 | 4 | 5> = {
  // Cards1
  pc_status: 1, scan_result: 1, daily_report: 1, clean_result: 1, repair_result: 1,
  folder_open: 1, agent_thinking: 1, preview_confirm: 1, error: 1, dynamic: 1,
  security_alert: 1,   // ★ 신규: 보안 경보 (VirusTotal/Shodan)
  // Cards2
  price_compare: 2, remote_access: 2, process_security: 2, defender: 2, startup_items: 2,
  process_top: 2, system_action: 2, network: 2, drivers: 2, programs_list: 2,
  file_search: 2, duplicates: 2, notes: 2, boot_analysis: 2, focus_mode: 2,
  file_result: 2, email_list: 2, timeline: 2, gauge_bar: 2, text_block: 2,
  step_list: 2, item_list: 2, grid_select: 2, weather_card: 2,
  // ★ 신규 Cards2
  business_card: 2,    // 국세청 사업자 조회
  translate_result: 2, // 번역 결과
  map_result: 2,       // 지도/경로
  email_draft: 2,      // 이메일 초안
  calendar_event: 2,   // 캘린더 이벤트 추가
  workflow_result: 2,  // 워크플로우 실행 결과
  memory_card: 2,      // 메모리/리콜 검색
  cmd_result: 2,       // 시스템 명령 결과
  // Cards3
  doc_compare: 3, doc_find: 3, deep_search: 3, vision_result: 3, vision_ocr: 3,
  smart_organize: 3,
  agent_result: 3,     // ★ 신규: Agent 실행 결과
  // Cards4
  journal_today: 4, journal_history: 4, macro_list: 4, macro_created: 4, macro_run: 4,
  pc_report: 4, doc_summary: 4,
  briefing_card: 4,    // ★ 신규: Proactive AI 브리핑
  meeting_summary: 4,  // ★ 신규: 미팅 요약
  // Cards5
  web_search: 5, news_search: 5, youtube: 5,
  stock_card: 5,       // ★ 신규: 주식/금융 정보
  tiktok_result: 5,    // ★ 신규: 틱톡 결과
  reddit_result: 5,    // ★ 신규: 레딧 결과
  media_result: 5,     // ★ 신규: 넷플릭스/미디어
}

export type ResolvedCards = {
  card?:  InlineCardData
  card2?: InlineCardData2
  card3?: InlineCard3Data
  card4?: InlineCard4Data
  card5?: InlineCard5Data
}

/**
 * resolveCardType — 어떤 카드를 그릴지 결정
 *   1. 백엔드가 card_type 명시 (cmd.card_type) → 우선
 *   2. ACTION_TO_CARD 매핑표
 *   3. result 구조 자동 감지
 *   4. null (텍스트만)
 */
export function resolveCardType(
  action: string,
  result: unknown,
  explicitCardType?: string,
): string | null {
  // 1) 백엔드 명시 우선
  if (explicitCardType && explicitCardType !== '') return explicitCardType
  // 2) 액션명 매핑
  if (ACTION_TO_CARD[action]) return ACTION_TO_CARD[action]
  // 3) result 구조 자동 감지
  if (result && typeof result === 'object') {
    const r = result as Record<string, unknown>
    if (Array.isArray(r.items) && r.items.length > 0) {
      // items 안의 url 패턴 보고 web/news/youtube 분류
      const firstItem = r.items[0] as Record<string, unknown>
      const url = (firstItem?.url as string) ?? ''
      if (url.includes('youtube.com') || url.includes('youtu.be')) return 'youtube'
      if (r.preview_type === 'news') return 'news_search'
      return 'web_search'
    }
    if (Array.isArray(r.results) && r.results.length > 0) return 'price_compare'
  }
  // 4) 폴백: 텍스트만 (또는 system_action)
  return null
}

/**
 * buildCardFromResult — card_type + result → 카드 데이터 객체
 *   각 카드 type 별로 result 를 표준 데이터 형태로 변환
 *   타입 안전성보단 유연성 우선 (any 사용) — 백엔드 응답이 다양함
 */
export function buildCardFromResult(
  cardType: string,
  result: unknown,
  query: string,
  message?: string,
): ResolvedCards {
  if (!cardType) return {}
  const r = (result as Record<string, any>) ?? {}
  const out: ResolvedCards = {}

  switch (cardType) {
    // ── Cards1 ──
    case 'pc_status':
      out.card = { type: 'pc_status', data: r as any }
      break
    case 'scan_result':
      out.card = { type: 'scan_result', data: r as any }
      break
    case 'clean_result':
      out.card = { type: 'clean_result', results: (r.results ?? r) as any }
      break
    case 'daily_report':
      out.card = { type: 'daily_report', data: r as any }
      break
    case 'repair_result':
      out.card = { type: 'repair_result', data: r as any }
      break
    case 'folder_open':
      out.card = { type: 'folder_open', success: r.success !== false, path: r.path, message: r.message ?? message ?? '' }
      break

    // ── Cards2 ──
    case 'price_compare':
      out.card2 = {
        type: 'price_compare',
        data: {
          query: (r.query as string) ?? query,
          results: (r.results as any[]) ?? [],
          total: r.total ?? ((r.results as any[])?.length ?? 0),
          summary: r.summary ?? message ?? '',
        },
      }
      break
    case 'remote_access':
      out.card2 = { type: 'remote_access', data: r as any }
      break
    case 'process_security':
      out.card2 = { type: 'process_security', data: r as any }
      break
    case 'defender':
      out.card2 = { type: 'defender', data: r as any }
      break
    case 'startup_items':
      out.card2 = { type: 'startup_items', data: r as any }
      break
    case 'process_top':
      out.card2 = { type: 'process_top', data: r as any }
      break
    case 'system_action':
      out.card2 = {
        type: 'system_action',
        icon: (r.icon as string) ?? '📋',
        title: (r.title as string) ?? query,
        detail: (r.detail as string) ?? message,
        success: r.success !== false,
      }
      break
    case 'network':
      out.card2 = { type: 'network', data: r as any }
      break
    case 'drivers':
      out.card2 = { type: 'drivers', data: r as any }
      break
    case 'programs_list':
      out.card2 = { type: 'programs_list', data: r as any }
      break
    case 'file_search':
      out.card2 = { type: 'file_search', data: r as any }
      break
    case 'duplicates':
      out.card2 = { type: 'duplicates', data: r as any }
      break
    case 'notes':
      out.card2 = { type: 'notes', data: r as any }
      break
    case 'boot_analysis':
      out.card2 = { type: 'boot_analysis', data: r as any }
      break
    case 'focus_mode':
      out.card2 = { type: 'focus_mode', active: r.active !== false, duration: r.duration }
      break
    case 'file_result':
      // 자동 생성 파일 (Excel/PDF/Doc) → 파일 카드: 경로/이름/타입/행 수 + 열기 버튼
      out.card2 = {
        type: 'file_result',
        data: {
          fileName:  (r.fileName as string) ?? (r.path as string)?.split(/[\\/]/).pop() ?? 'untitled',
          url:       (r.url as string) ?? ((r.path as string) ? 'file:///' + (r.path as string).replace(/\\/g, '/') : ''),
          mimeType:  (r.mimeType as string) ?? 'application/octet-stream',
          frames:    r.rows,  // Excel: rows 수
          width:     r.cols,  // Excel: cols 수
          operation: (r.operation as string) ?? 'excel_create',
        },
      }
      break
    case 'email_list':
      out.card2 = { type: 'email_list', data: r as any }
      break
    case 'timeline':
      out.card2 = { type: 'timeline', data: r as any }
      break
    case 'weather_card':
      // 백엔드 응답(wind_kmh, forecast.max/min) → 카드 data(wind_kph, high_c/low_c) 정규화
      out.card2 = {
        type: 'weather_card',
        data: {
          city:       r.city,
          condition:  r.condition,
          temp_c:     r.temp_c,
          feels_like: r.feels_like,
          humidity:   r.humidity,
          wind_kph:   r.wind_kph ?? r.wind_kmh,
          icon:       r.icon,
          forecast: ((r.forecast as any[]) ?? []).map((f: any) => ({
            date:      f.date,
            condition: f.condition,
            high_c:    f.high_c ?? f.max,
            low_c:     f.low_c  ?? f.min,
            icon:      f.icon,
          })),
          summary:    r.summary ?? message,
        },
      }
      break

    // ── Cards3 ──
    case 'doc_compare':
      out.card3 = { type: 'doc_compare', data: r as any }
      break
    case 'doc_find':
      out.card3 = { type: 'doc_find', data: r as any }
      break
    case 'deep_search':
      out.card3 = { type: 'deep_search', data: r as any }
      break
    case 'vision_result':
      out.card3 = { type: 'vision_result', data: r as any }
      break
    case 'vision_ocr':
      out.card3 = { type: 'vision_ocr', data: r as any }
      break
    case 'smart_organize':
      out.card3 = { type: 'smart_organize', data: r as any }
      break

    // ── Cards4 ──
    case 'journal_today':
      out.card4 = { type: 'journal_today', data: r as any }
      break
    case 'journal_history':
      out.card4 = { type: 'journal_history', data: r as any }
      break
    case 'macro_list':
      out.card4 = { type: 'macro_list', data: r as any }
      break
    case 'macro_created':
      out.card4 = { type: 'macro_created', data: r as any }
      break
    case 'macro_run':
      out.card4 = { type: 'macro_run', data: r as any }
      break
    case 'pc_report':
      out.card4 = { type: 'pc_report', data: r as any }
      break
    case 'doc_summary':
      out.card4 = { type: 'doc_summary', data: r as any }
      break

    // ── Cards5 (검색 결과) ──
    case 'web_search':
      // items 정규화 (백엔드 응답 다양: items / results / articles)
      out.card5 = {
        type: 'web_search',
        query: (r.query as string) ?? query,
        summary: (r.summary as string) ?? message ?? '',
        items: ((r.items as any[]) ?? (r.results as any[]) ?? (r.articles as any[]) ?? []).map((it: any) => ({
          title:     it.title ?? it.name ?? '',
          url:       it.url ?? it.link ?? '',
          snippet:   it.snippet ?? it.description ?? it.content ?? '',
          source:    it.source ?? it.site ?? '',
          published: it.published ?? it.date ?? '',
          thumbnail: it.thumbnail ?? it.image ?? '',
        })).filter((it: any) => it.url),
      }
      break
    case 'news_search':
      // articles → items 자동 정규화 (백엔드 응답 다양)
      out.card5 = {
        type: 'news_search',
        query: (r.query as string) ?? query,
        summary: (r.summary as string) ?? message ?? '',
        items: ((r.items as any[]) ?? (r.articles as any[]) ?? []).map((it: any) => ({
          title:     it.title ?? it.name ?? '',
          url:       it.url ?? it.link ?? '',
          snippet:   it.snippet ?? it.description ?? it.content ?? '',
          source:    it.source ?? it.site ?? '',
          published: it.published ?? it.date ?? '',
          thumbnail: it.thumbnail ?? it.image ?? '',
        })).filter((it: any) => it.url),
      }
      break
    case 'youtube':
      out.card5 = {
        type: 'youtube',
        query: (r.query as string) ?? query,
        items: (r.items as any[]) ?? [],
      }
      break

    // ── 신규 Cards1 ──
    case 'security_alert':
      out.card = {
        type: 'scan_result',   // scan_result 카드 재활용 (위협 경보)
        data: {
          ...(r as any),
          alert: true,
          severity: r.severity ?? 'high',
          summary: r.summary ?? message ?? '보안 위협 감지',
        } as any,
      }
      break

    // ── 신규 Cards2 ──
    case 'business_card':
      out.card2 = {
        type: 'system_action',
        icon: '🏢',
        title: `사업자 조회: ${r.brno ?? query}`,
        detail: (r.raw as string) ?? (message ?? ''),
        success: !String(r.raw ?? '').includes('❌'),
      }
      break
    case 'translate_result':
      out.card2 = {
        type: 'system_action',
        icon: '🌐',
        title: `번역 결과 (→ ${r.target ?? ''})`,
        detail: (r.translated ?? r.result ?? message ?? '') as string,
        success: true,
      }
      break
    case 'map_result':
      out.card2 = {
        type: 'system_action',
        icon: '🗺️',
        title: (r.title as string) ?? query,
        detail: (r.description ?? r.duration ?? r.distance ?? message ?? '') as string,
        success: true,
      }
      break
    case 'email_draft':
      out.card2 = {
        type: 'system_action',
        icon: '✉️',
        title: (r.subject as string) ?? '이메일 초안',
        detail: (r.body ?? r.draft ?? message ?? '').slice(0, 200) as string,
        success: true,
      }
      break
    case 'calendar_event':
      out.card2 = {
        type: 'system_action',
        icon: '📅',
        title: (r.title as string) ?? '일정 추가됨',
        detail: `${r.datetime ?? r.date ?? ''} ${r.duration ? `(${r.duration}분)` : ''}`.trim() || (message ?? ''),
        success: r.success !== false,
      }
      break
    case 'workflow_result':
      out.card2 = {
        type: 'system_action',
        icon: '⚡',
        title: (r.name as string) ?? '워크플로우 실행',
        detail: (r.summary ?? r.output ?? message ?? '') as string,
        success: r.success !== false,
      }
      break
    case 'memory_card':
      out.card2 = {
        type: 'notes',
        data: {
          items: (r.results as any[]) ?? (r.items as any[]) ?? [],
          query,
          total: r.total ?? 0,
        } as any,
      }
      break
    case 'cmd_result':
      out.card2 = {
        type: 'system_action',
        icon: r.icon ?? '⚙️',
        title: (r.title as string) ?? query.slice(0, 40),
        detail: (r.output ?? r.detail ?? message ?? '') as string,
        success: r.success !== false,
      }
      break

    // ── 신규 Cards3 ──
    case 'agent_result':
      out.card3 = {
        type: 'deep_search',   // deep_search 카드 재활용 (단계별 결과 표시)
        data: {
          query,
          steps: (r.steps as any[]) ?? [],
          result: r.output ?? r.result ?? message ?? '',
          sources: (r.sources as any[]) ?? [],
          summary: r.summary ?? message ?? '',
        } as any,
      }
      break

    // ── 신규 Cards4 ──
    case 'briefing_card':
      out.card4 = {
        type: 'journal_today',  // 일지 카드 재활용 (브리핑 구조 유사)
        data: {
          date: new Date().toISOString().slice(0, 10),
          content: (r.briefing ?? r.summary ?? message ?? '') as string,
          weather: r.weather,
          events: r.events,
          news: r.news,
        } as any,
      }
      break
    case 'meeting_summary':
      out.card4 = {
        type: 'doc_summary',
        data: {
          title: (r.title as string) ?? '미팅 요약',
          summary: (r.summary ?? r.transcript ?? message ?? '') as string,
          keyPoints: (r.key_points as string[]) ?? [],
          actionItems: (r.action_items as string[]) ?? [],
        } as any,
      }
      break

    // ── 신규 Cards5 ──
    case 'stock_card':
      out.card5 = {
        type: 'web_search',   // web_search 카드 재활용 (주가 링크 목록)
        query: (r.ticker as string) ?? query,
        summary: (r.summary ?? message ?? '') as string,
        items: [
          ...(r.news ? (r.news as any[]).map((n: any) => ({
            title: n.title ?? '', url: n.url ?? '', snippet: n.snippet ?? '',
            source: n.source ?? 'finance', published: n.date ?? '',
          })) : []),
          ...(r.url ? [{ title: `${r.ticker} 주가 정보`, url: r.url, snippet: `현재가: ${r.price ?? '-'}  ${r.change ?? ''}`, source: r.exchange ?? 'market' }] : []),
        ],
      }
      break
    case 'tiktok_result':
      out.card5 = {
        type: 'youtube',   // youtube 카드 재활용
        query: (r.query as string) ?? query,
        items: (r.items as any[]) ?? (r.videos as any[]) ?? [],
      }
      break
    case 'reddit_result':
      out.card5 = {
        type: 'news_search',
        query: (r.query as string) ?? query,
        summary: (r.summary ?? '') as string,
        items: ((r.items as any[]) ?? (r.posts as any[]) ?? []).map((p: any) => ({
          title: p.title ?? '', url: p.url ?? '', snippet: p.snippet ?? p.selftext ?? '',
          source: `r/${p.subreddit ?? 'reddit'}`, published: p.created ?? '',
        })),
      }
      break
    case 'media_result':
      out.card5 = {
        type: 'web_search',
        query: (r.query as string) ?? query,
        summary: (r.summary ?? message ?? '') as string,
        items: ((r.items as any[]) ?? []).map((m: any) => ({
          title: m.title ?? '', url: m.url ?? '#',
          snippet: m.description ?? m.genre ?? '',
          source: m.platform ?? 'streaming',
          thumbnail: m.poster ?? m.thumbnail ?? '',
        })),
      }
      break

    default:
      // 알 수 없는 카드 타입 → system_action 폴백
      out.card2 = {
        type: 'system_action',
        icon: '📋',
        title: query.slice(0, 40),
        detail: message ?? '',
        success: true,
      }
  }
  return out
}

/**
 * autoBuildCard — 단일 라우터: 액션 + result → 카드 자동 생성
 *   백엔드 응답이 어떻든 적절한 카드를 알아서 만들어줌 (사장님 설계)
 */
export function autoBuildCard(
  action: string,
  result: unknown,
  query: string,
  message?: string,
  explicitCardType?: string,
): ResolvedCards {
  const cardType = resolveCardType(action, result, explicitCardType)
  if (!cardType) return {}
  return buildCardFromResult(cardType, result, query, message)
}
