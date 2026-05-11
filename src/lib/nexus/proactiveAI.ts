/**
 * ══════════════════════════════════════════════════════════════
 * proactiveAI.ts — Nexus 능동형 AI 감시 시스템
 * ══════════════════════════════════════════════════════════════
 *
 * Nexus가 사용자가 먼저 말하지 않아도 PC 상태를 지켜보다가
 * 문제가 생기면 먼저 말을 걸어주는 능동형 AI 기능.
 *
 * 아키텍처:
 *   1. 폴링 루프       → 30초(stats) / 5분(security) 주기로 백엔드 호출
 *   2. 규칙 엔진       → 트리거 조건 평가 (우선순위 1~3)
 *   3. 쿨다운 매니저   → 같은 알림 중복 방지 (localStorage 기반)
 *   4. 알림 발화       → 감정 변경 + 채팅 메시지 + TTS + 액션 버튼
 */

import type { CharacterEmotion } from '../../components/FloatingCharacter/characters'
import type { StatsData } from './backendAPI'

/* ── 타입 ── */

export interface ProactiveAction {
  label: string         // "지금 정리해줘" | "보안 점검"
  intent: string        // 실행할 인텐트
  autoText: string      // sendText에 넣을 텍스트
}

export interface ProactiveTrigger {
  id: string
  label: string
  priority: 1 | 2 | 3  // 1=위험🔴 2=경고🟡 3=정보💙
  cooldownMs: number
  /** 트리거 조건 — true면 알림 발화 */
  check: (data: PollSnapshot) => boolean
  /** 알림 메시지 생성 */
  message: (data: PollSnapshot, lang: 'ko' | 'en', name: string) => string
  /** 캐릭터 감정 */
  emotion: CharacterEmotion
  /** 빠른 액션 버튼 (선택) */
  actions?: ProactiveAction[]
}

export interface PollSnapshot {
  stats: StatsData
  security?: { found: boolean; score: number }
  uptimeMs: number  // 앱 시작 후 경과 ms
  focusModeEndMs?: number  // 집중모드 종료 예정 timestamp
}

export interface ProactiveAlert {
  triggerId: string
  priority: 1 | 2 | 3
  message: string
  emotion: CharacterEmotion
  actions: ProactiveAction[]
  timestamp: number
}

/* ── 쿨다운 매니저 ── */

const COOLDOWN_KEY = 'nexus-proactive-cooldowns'

function loadCooldowns(): Record<string, number> {
  try {
    return JSON.parse(localStorage.getItem(COOLDOWN_KEY) ?? '{}')
  } catch {
    return {}
  }
}

function saveCooldowns(cd: Record<string, number>) {
  localStorage.setItem(COOLDOWN_KEY, JSON.stringify(cd))
}

function isOnCooldown(triggerId: string, cooldownMs: number): boolean {
  const cd = loadCooldowns()
  const last = cd[triggerId] ?? 0
  return Date.now() - last < cooldownMs
}

function markFired(triggerId: string) {
  const cd = loadCooldowns()
  cd[triggerId] = Date.now()
  saveCooldowns(cd)
}

export function resetCooldown(triggerId: string) {
  const cd = loadCooldowns()
  delete cd[triggerId]
  saveCooldowns(cd)
}

/* ── 트리거 규칙 목록 ── */

export const PROACTIVE_TRIGGERS: ProactiveTrigger[] = [
  // ────────────────────────────────────────────────
  // Priority 1 — 위험 🔴 (즉각 알림)
  // ────────────────────────────────────────────────
  {
    id: 'cpu_overheat',
    label: 'CPU 과열',
    priority: 1,
    cooldownMs: 5 * 60_000, // 5분
    check: d => d.stats.cpu_temp > 85,
    message: (d, lang, name) =>
      lang === 'ko'
        ? `${name}, CPU 온도가 ${d.stats.cpu_temp.toFixed(0)}°C까지 올라갔어요! 🌡️ 쿨링팬을 확인하거나 무거운 작업을 줄여보세요.`
        : `${name}, CPU temperature reached ${d.stats.cpu_temp.toFixed(0)}°C! Please check cooling or reduce heavy tasks.`,
    emotion: 'alert',
    actions: [
      { label: '무거운 프로세스 확인', intent: 'process_top', autoText: '어떤 앱이 CPU를 많이 써?' },
    ],
  },
  {
    id: 'cpu_maxed',
    label: 'CPU 100% 포화',
    priority: 1,
    cooldownMs: 10 * 60_000,
    check: d => d.stats.cpu > 90,
    message: (d, lang, name) =>
      lang === 'ko'
        ? `${name}, CPU가 ${d.stats.cpu.toFixed(0)}% 사용 중이에요. PC가 매우 힘들어하고 있어요 😰`
        : `${name}, CPU is at ${d.stats.cpu.toFixed(0)}%. Your PC is struggling.`,
    emotion: 'alert',
    actions: [
      { label: '상위 프로세스 보기', intent: 'process_top', autoText: 'CPU 많이 쓰는 프로세스 알려줘' },
    ],
  },
  {
    id: 'disk_critical',
    label: '디스크 포화',
    priority: 1,
    cooldownMs: 60 * 60_000, // 1시간
    check: d => d.stats.disk > 95,
    message: (d, lang, name) =>
      lang === 'ko'
        ? `${name}, 디스크가 ${d.stats.disk.toFixed(0)}%까지 찼어요 💾 지금 바로 정리하지 않으면 PC가 오작동할 수 있어요!`
        : `${name}, disk is ${d.stats.disk.toFixed(0)}% full. Cleanup needed immediately!`,
    emotion: 'alert',
    actions: [
      { label: '지금 정리해줘', intent: 'clean', autoText: 'PC 청소해줘' },
      { label: '중복 파일 찾기', intent: 'file_duplicates', autoText: '중복 파일 찾아줘' },
    ],
  },
  {
    id: 'security_threat',
    label: '보안 위협 감지',
    priority: 1,
    cooldownMs: 30 * 60_000,
    check: d => !!(d.security?.found && d.security.score < 70),
    message: (d, lang, name) =>
      lang === 'ko'
        ? `${name}, 수상한 원격 접속 흔적이 발견됐어요! 🚨 보안 점검이 필요해요.`
        : `${name}, suspicious remote access detected! Security check needed.`,
    emotion: 'alert',
    actions: [
      { label: '보안 점검하기', intent: 'security_scan', autoText: '해킹 탐지해줘' },
      { label: '원격 접속 확인', intent: 'remote_access', autoText: '원격 접속 확인해줘' },
    ],
  },

  // ────────────────────────────────────────────────
  // Priority 2 — 경고 🟡
  // ────────────────────────────────────────────────
  {
    id: 'ram_high',
    label: 'RAM 부족',
    priority: 2,
    cooldownMs: 15 * 60_000,
    check: d => d.stats.mem > 85,
    message: (d, lang, name) =>
      lang === 'ko'
        ? `${name}, 메모리가 ${d.stats.mem.toFixed(0)}% 사용 중이에요 🐌 불필요한 탭이나 앱을 닫으면 빨라져요.`
        : `${name}, RAM is at ${d.stats.mem.toFixed(0)}%. Close unused apps to speed things up.`,
    emotion: 'concerned',
    actions: [
      { label: '메모리 많이 쓰는 앱', intent: 'process_top', autoText: '메모리 많이 쓰는 앱 알려줘' },
    ],
  },
  {
    id: 'cpu_temp_warn',
    label: 'CPU 온도 경고',
    priority: 2,
    cooldownMs: 10 * 60_000,
    check: d => d.stats.cpu_temp > 75 && d.stats.cpu_temp <= 85,
    message: (d, lang, name) =>
      lang === 'ko'
        ? `${name}, CPU 온도가 ${d.stats.cpu_temp.toFixed(0)}°C로 조금 높아요 🌡️ 환기가 잘 되는 곳에 두세요.`
        : `${name}, CPU is ${d.stats.cpu_temp.toFixed(0)}°C, a bit warm. Ensure good airflow.`,
    emotion: 'concerned',
  },
  {
    id: 'disk_warn',
    label: '디스크 여유 부족',
    priority: 2,
    cooldownMs: 30 * 60_000,
    check: d => d.stats.disk > 90 && d.stats.disk <= 95,
    message: (d, lang, name) =>
      lang === 'ko'
        ? `${name}, 디스크가 ${d.stats.disk.toFixed(0)}% 찼어요. 슬슬 정리할 시간이에요 📁`
        : `${name}, disk is ${d.stats.disk.toFixed(0)}% full. Time for some cleanup.`,
    emotion: 'concerned',
    actions: [
      { label: 'PC 정리하기', intent: 'clean', autoText: 'PC 청소해줘' },
    ],
  },
  {
    id: 'cpu_warm',
    label: 'CPU 높음',
    priority: 2,
    cooldownMs: 10 * 60_000,
    check: d => d.stats.cpu > 75 && d.stats.cpu <= 90,
    message: (d, lang, name) =>
      lang === 'ko'
        ? `${name}, CPU가 ${d.stats.cpu.toFixed(0)}% 사용 중이에요. 백그라운드 앱을 확인해볼까요? 💡`
        : `${name}, CPU at ${d.stats.cpu.toFixed(0)}%. Want me to check background apps?`,
    emotion: 'concerned',
    actions: [
      { label: '프로세스 확인', intent: 'process_top', autoText: 'CPU 많이 쓰는 프로세스 알려줘' },
    ],
  },

  // ────────────────────────────────────────────────
  // Priority 3 — 정보 💙
  // ────────────────────────────────────────────────
  {
    id: 'focus_timer_done',
    label: '집중 모드 종료',
    priority: 3,
    cooldownMs: 0, // 1회성, 직접 호출 방식
    check: d => !!(d.focusModeEndMs && Date.now() >= d.focusModeEndMs),
    message: (_d, lang, name) =>
      lang === 'ko'
        ? `${name}, 25분 집중 끝났어요! ☕ 잠깐 휴식하고 오세요. Nexus가 기다릴게요.`
        : `${name}, focus session complete! Take a short break. ☕`,
    emotion: 'happy',
  },
  {
    id: 'long_usage',
    label: '장시간 사용',
    priority: 3,
    cooldownMs: 4 * 60 * 60_000, // 4시간
    check: d => d.uptimeMs > 4 * 60 * 60_000,
    message: (_d, lang, name) =>
      lang === 'ko'
        ? `${name}, 4시간 연속 사용 중이에요 💙 잠시 눈 쉬어주시는 건 어떨까요?`
        : `${name}, you've been working for 4 hours straight. Time for an eye break! 💙`,
    emotion: 'happy',
  },
  {
    id: 'daily_checkup',
    label: '일일 점검',
    priority: 3,
    cooldownMs: 24 * 60 * 60_000, // 24시간
    check: d => {
      const hour = new Date().getHours()
      return hour === 9 && d.uptimeMs > 60_000 // 오전 9시 이후 1분 이상 실행 중
    },
    message: (_d, lang, name) =>
      lang === 'ko'
        ? `좋은 아침이에요, ${name}! ☀️ 오늘 하루 시작 전에 PC 점검 한 번 해드릴까요?`
        : `Good morning, ${name}! ☀️ Want a quick PC checkup to start the day?`,
    emotion: 'happy',
    actions: [
      { label: '오늘 리포트 보기', intent: 'daily_report', autoText: '오늘 리포트 보여줘' },
    ],
  },
]

/* ── 규칙 엔진 ── */

/**
 * 현재 스냅샷을 받아 발화해야 할 알림 목록을 반환한다.
 * 쿨다운 중이거나 조건 불충족 시 해당 트리거는 건너뜀.
 * 우선순위 1 → 2 → 3 순서로 정렬, 한 번에 최대 1개만 반환 (스팸 방지).
 */
export function evaluateTriggers(
  snapshot: PollSnapshot,
  lang: 'ko' | 'en',
  assistantName: string,
): ProactiveAlert | null {
  // 우선순위 높은 순 정렬
  const sorted = [...PROACTIVE_TRIGGERS].sort((a, b) => a.priority - b.priority)

  for (const trigger of sorted) {
    if (isOnCooldown(trigger.id, trigger.cooldownMs)) continue
    if (!trigger.check(snapshot)) continue

    markFired(trigger.id)

    return {
      triggerId:  trigger.id,
      priority:   trigger.priority,
      message:    trigger.message(snapshot, lang, assistantName),
      emotion:    trigger.emotion,
      actions:    trigger.actions ?? [],
      timestamp:  Date.now(),
    }
  }

  return null
}

/* ── 폴링 인터벌 상수 ── */

/** PC 상태 폴링 주기 (ms) */
export const STATS_POLL_MS      = 30_000   // 30초
/** 보안 폴링 주기 (ms) */
export const SECURITY_POLL_MS   = 5 * 60_000 // 5분
/** 집중 모드 타이머 체크 주기 (ms) */
export const FOCUS_CHECK_MS     = 1_000    // 1초

/* ── 포커스 모드 매니저 ── */

const FOCUS_KEY = 'nexus-focus-mode-end'

export function setFocusModeEnd(durationMinutes: number) {
  const endMs = Date.now() + durationMinutes * 60_000
  localStorage.setItem(FOCUS_KEY, String(endMs))
}

export function getFocusModeEnd(): number | undefined {
  const v = localStorage.getItem(FOCUS_KEY)
  if (!v) return undefined
  const n = Number(v)
  // 이미 지났으면 제거
  if (Date.now() >= n) {
    localStorage.removeItem(FOCUS_KEY)
    return undefined
  }
  return n
}

export function clearFocusMode() {
  localStorage.removeItem(FOCUS_KEY)
}

/* ── 앱 시작 시각 ── */

const APP_START_MS = Date.now()

export function getUptimeMs(): number {
  return Date.now() - APP_START_MS
}

/* ──────────────────────────────────────────────────────────────
   Autonomous Mode — 안전한 작업 확인 없이 자동 실행
   ──────────────────────────────────────────────────────────────
   활성화 시: 위험도 낮은 읽기 작업은 확인 없이 즉시 실행
   비활성화 시: 모든 중요 작업에 확인 요청
*/

const AUTONOMOUS_KEY = 'nexus-autonomous-mode'

export function isAutonomousMode(): boolean {
  return localStorage.getItem(AUTONOMOUS_KEY) === 'true'
}

export function setAutonomousMode(enabled: boolean) {
  localStorage.setItem(AUTONOMOUS_KEY, String(enabled))
}

/** 자율 실행 가능한 액션 목록 (확인 없이 실행해도 안전한 작업) */
const AUTONOMOUS_SAFE_ACTIONS = new Set([
  'get_system_stats',
  'run_diagnostics',
  'security_scan',
  'get_processes',
  'process_top',
  'deep_search',
  'ai_deep_search',
  'find_document',
  'search_files',
  'ai_analyze_screen',
  'capture_screen_and_ask',
  'ai_summarize_document',
  'browser_navigate',
  'browser_extract_page',
  'get_weather',
  'get_top_news',
  'get_system_stats',
  'network_analysis',
  'boot_analysis',
  'daily_report',
])

/**
 * 액션이 자율 모드에서 자동 실행 가능한지 판단
 */
export function canAutoExecute(action: string): boolean {
  if (!isAutonomousMode()) return false
  return AUTONOMOUS_SAFE_ACTIONS.has(action)
}

/* ──────────────────────────────────────────────────────────────
   추가 트리거 규칙 (PROACTIVE_TRIGGERS에 push)
   ──────────────────────────────────────────────────────────────
*/

export const EXTRA_TRIGGERS: ProactiveTrigger[] = [
  {
    id: 'low_net_speed',
    label: '네트워크 느림',
    priority: 2,
    cooldownMs: 30 * 60_000,
    check: d => d.stats.net_down < 100 && d.stats.net_down > 0,
    message: (_d, lang, name) =>
      lang === 'ko'
        ? `${name}, 인터넷 속도가 평소보다 느린 것 같아요 🌐 네트워크 상태를 확인해볼까요?`
        : `${name}, internet seems slow. Want me to check network status?`,
    emotion: 'concerned',
    actions: [
      { label: '네트워크 분석', intent: 'network_analysis', autoText: '인터넷 상태 확인해줘' },
    ],
  },
  {
    id: 'disk_warn_full',
    label: '디스크 경고 (80%)',
    priority: 3,
    cooldownMs: 6 * 60 * 60_000,
    check: d => d.stats.disk > 80 && d.stats.disk <= 90,
    message: (d, lang, name) =>
      lang === 'ko'
        ? `${name}, 디스크가 ${d.stats.disk.toFixed(0)}%까지 찼어요. 조만간 정리가 필요할 것 같아요 📂`
        : `${name}, disk is ${d.stats.disk.toFixed(0)}% full. Consider cleanup soon.`,
    emotion: 'neutral',
    actions: [
      { label: '중복 파일 찾기', intent: 'file_duplicates', autoText: '중복 파일 찾아줘' },
    ],
  },
  {
    id: 'morning_greeting',
    label: '아침 인사',
    priority: 3,
    cooldownMs: 24 * 60 * 60_000,
    check: d => {
      const h = new Date().getHours()
      return h >= 7 && h < 9 && d.uptimeMs > 30_000
    },
    message: (_d, lang, name) =>
      lang === 'ko'
        ? `좋은 아침이에요, ${name}! 오늘도 잘 부탁드려요 ☀️`
        : `Good morning, ${name}! Have a great day ☀️`,
    emotion: 'happy',
  },
  {
    id: 'browser_idle_cleanup',
    label: '브라우저 캐시 알림',
    priority: 3,
    cooldownMs: 7 * 24 * 60 * 60_000,
    check: d => d.uptimeMs > 60 * 60_000 && d.stats.disk > 60,
    message: (_d, lang, name) =>
      lang === 'ko'
        ? `${name}, 일주일에 한 번 브라우저 캐시 정리를 권장해요 🧹 정리해드릴까요?`
        : `${name}, weekly browser cache cleanup is recommended. Want me to do it?`,
    emotion: 'neutral',
    actions: [
      { label: '브라우저 정리', intent: 'browser_clean', autoText: '브라우저 캐시 정리해줘' },
    ],
  },
]

// PROACTIVE_TRIGGERS에 추가 규칙 병합
PROACTIVE_TRIGGERS.push(...EXTRA_TRIGGERS)

/* ── GPU·업데이트 추가 트리거 ── */
export const SYSTEM_TRIGGERS: ProactiveTrigger[] = [
  {
    id: 'gpu_overheat',
    label: 'GPU 과열',
    priority: 1,
    cooldownMs: 5 * 60_000,
    check: d => !!(d.stats.gpu && d.stats.gpu > 95),
    message: (d, lang, name) =>
      lang === 'ko'
        ? `${name}, GPU 사용률이 ${d.stats.gpu?.toFixed(0)}%로 포화 상태예요! 🎮 게임이나 렌더링 작업을 확인해보세요.`
        : `${name}, GPU is at ${d.stats.gpu?.toFixed(0)}% — check running games or rendering tasks.`,
    emotion: 'alert',
    actions: [{ label: 'GPU 상태 확인', intent: 'gpu_stats', autoText: 'GPU 상태 알려줘' }],
  },
  {
    id: 'updates_pending',
    label: 'Windows 업데이트 대기',
    priority: 3,
    cooldownMs: 24 * 60 * 60_000,
    check: d => {
      const h = new Date().getHours()
      return h === 10 && d.uptimeMs > 60_000 // 오전 10시에 1회 체크
    },
    message: (_d, lang, name) =>
      lang === 'ko'
        ? `${name}, Windows 업데이트가 있을 수 있어요 🔄 확인해드릴까요?`
        : `${name}, want me to check for pending Windows updates?`,
    emotion: 'neutral',
    actions: [{ label: '업데이트 확인', intent: 'windows_updates', autoText: 'Windows 업데이트 확인해줘' }],
  },
  {
    id: 'perf_anomaly_alert',
    label: '성능 이상 탐지',
    priority: 2,
    cooldownMs: 2 * 60 * 60_000,
    check: d => d.stats.cpu > 85 && d.stats.cpu_temp > 80,
    message: (d, lang, name) =>
      lang === 'ko'
        ? `${name}, CPU ${d.stats.cpu.toFixed(0)}%에 온도 ${d.stats.cpu_temp.toFixed(0)}°C — 동시에 높아요! 성능 이력을 확인해드릴까요?`
        : `${name}, CPU at ${d.stats.cpu.toFixed(0)}% and ${d.stats.cpu_temp.toFixed(0)}°C simultaneously — worth investigating.`,
    emotion: 'alert',
    actions: [{ label: '성능 이력 보기', intent: 'perf_history', autoText: '성능 이력 보여줘' }],
  },
]

PROACTIVE_TRIGGERS.push(...SYSTEM_TRIGGERS)

/* ──────────────────────────────────────────────────────────────
   스마트 알림 필터 — 집중 모드 중 알림 억제
   ──────────────────────────────────────────────────────────────
*/

/**
 * 집중 모드 중이거나 DND 시간(22시~7시)이면 priority 3 알림 억제
 */
export function shouldSuppressAlert(priority: 1 | 2 | 3): boolean {
  const hour = new Date().getHours()
  const isDND = hour >= 22 || hour < 7
  const inFocus = !!getFocusModeEnd()

  if (priority === 3 && (isDND || inFocus)) return true
  if (priority === 2 && isDND) return true
  return false
}

/**
 * evaluateTriggers에 DND 필터 적용된 버전
 */
export function evaluateTriggersFiltered(
  snapshot: PollSnapshot,
  lang: 'ko' | 'en',
  assistantName: string,
): ProactiveAlert | null {
  const sorted = [...PROACTIVE_TRIGGERS].sort((a, b) => a.priority - b.priority)

  for (const trigger of sorted) {
    if (isOnCooldown(trigger.id, trigger.cooldownMs)) continue
    if (!trigger.check(snapshot)) continue
    if (shouldSuppressAlert(trigger.priority)) continue

    markFired(trigger.id)
    return {
      triggerId: trigger.id,
      priority:  trigger.priority,
      message:   trigger.message(snapshot, lang, assistantName),
      emotion:   trigger.emotion,
      actions:   trigger.actions ?? [],
      timestamp: Date.now(),
    }
  }
  return null
}
