/**
 * InsightLine — 데이터만 보여주지 말고 AI가 한 줄 해석.
 *
 * 예:
 *   ❌ "CPU 87%, MEM 89%"
 *   ✅ "💡 CPU+MEM 모두 높음 — Chrome 탭 30개 닫으면 30% 회복 예상"
 */

import { motion } from 'framer-motion'

export type InsightLevel = 'info' | 'tip' | 'warning' | 'critical' | 'success'

const LEVEL_META: Record<InsightLevel, { icon: string; color: string; bg: string }> = {
  info:     { icon: 'ℹ️', color: '#60a5fa', bg: '#3b82f615' },
  tip:      { icon: '💡', color: '#fbbf24', bg: '#f59e0b15' },
  warning:  { icon: '⚠️', color: '#f97316', bg: '#f9731615' },
  critical: { icon: '🚨', color: '#ef4444', bg: '#ef444415' },
  success:  { icon: '✅', color: '#22c55e', bg: '#22c55e15' },
}

interface InsightLineProps {
  text: string
  level?: InsightLevel
  /** 인사이트 다음에 표시할 액션 버튼 (1개 권장) */
  action?: { label: string; onClick: () => void }
}

export function InsightLine({ text, level = 'tip', action }: InsightLineProps) {
  const meta = LEVEL_META[level]
  return (
    <motion.div
      initial={{ opacity: 0, x: -4 }}
      animate={{ opacity: 1, x: 0 }}
      transition={{ delay: 0.15 }}
      style={{
        display: 'flex', alignItems: 'flex-start', gap: 8,
        padding: '8px 10px', marginTop: 8, marginBottom: action ? 4 : 0,
        background: meta.bg,
        borderLeft: `2px solid ${meta.color}`,
        borderRadius: 6,
        fontSize: 11, color: 'rgba(255,255,255,0.88)', lineHeight: 1.5,
      }}
    >
      <span style={{ fontSize: 12, lineHeight: 1, flexShrink: 0, marginTop: 1 }}>{meta.icon}</span>
      <div style={{ flex: 1, minWidth: 0 }}>
        <div>{text}</div>
        {action && (
          <button
            onClick={action.onClick}
            style={{
              marginTop: 6, padding: '4px 10px',
              background: `${meta.color}33`,
              border: `1px solid ${meta.color}66`,
              color: meta.color,
              borderRadius: 6, fontSize: 10, fontWeight: 700,
              cursor: 'pointer',
              transition: 'all 0.15s',
            }}
            onMouseEnter={e => { e.currentTarget.style.background = `${meta.color}55` }}
            onMouseLeave={e => { e.currentTarget.style.background = `${meta.color}33` }}
          >
            → {action.label}
          </button>
        )}
      </div>
    </motion.div>
  )
}

/* ─────────────────────────────────────────────────────────── */
/* 도메인별 인사이트 자동 생성 헬퍼                            */
/* (LLM 호출 없이 로컬에서 규칙 기반 — 빠르고 무료)            */
/* ─────────────────────────────────────────────────────────── */

export interface PCStatusLike {
  cpu: number; mem: number; disk: number; cpu_temp?: number; gpu?: number
}

export function insightForPcStatus(d: PCStatusLike, lang: 'ko' | 'en' = 'ko'): { text: string; level: InsightLevel } | null {
  // 우선순위 높은 순으로 검사
  if (d.cpu_temp && d.cpu_temp >= 90) {
    return { text: lang === 'en'
        ? `CPU temp critically high (${d.cpu_temp}°C) — consider cleaning vents / pausing heavy tasks.`
        : `CPU 온도 위험 수준(${d.cpu_temp}°C) — 환기구 청소나 무거운 작업 일시정지를 권해요.`,
      level: 'critical' }
  }
  if (d.cpu >= 90 && d.mem >= 85) {
    return { text: lang === 'en'
        ? 'Both CPU & memory near max — restart the heaviest process or close some Chrome tabs.'
        : 'CPU + 메모리 모두 한계 근접 — 가장 무거운 프로세스 재시작이나 Chrome 탭 정리가 필요해요.',
      level: 'critical' }
  }
  if (d.cpu >= 85) {
    return { text: lang === 'en' ? 'High CPU load — something is working hard. Check Top Processes.' : 'CPU 부하 높음 — 어떤 프로세스가 많이 쓰는지 확인해보세요.', level: 'warning' }
  }
  if (d.mem >= 85) {
    return { text: lang === 'en' ? 'Memory pressure high — restarting heavy apps frees RAM.' : '메모리 부족 임박 — 무거운 앱을 재시작하면 회복돼요.', level: 'warning' }
  }
  if (d.disk >= 90) {
    return { text: lang === 'en' ? 'Disk almost full — run cleanup or move large files.' : '디스크 거의 가득 참 — 정리하기를 실행하면 공간을 확보해요.', level: 'warning' }
  }
  if (d.cpu < 30 && d.mem < 50 && d.disk < 70) {
    return { text: lang === 'en' ? 'PC running smoothly — plenty of headroom.' : 'PC가 여유롭게 동작 중이에요.', level: 'success' }
  }
  return null
}

export interface ScanLike { score: number; issues: Array<{ severity: string }> }

export function insightForScan(d: ScanLike, lang: 'ko' | 'en' = 'ko'): { text: string; level: InsightLevel } | null {
  const high = d.issues.filter(i => i.severity === 'high').length
  if (d.score < 60 || high >= 3) {
    return { text: lang === 'en'
        ? `${high} critical issues found — fix recommended immediately.`
        : `심각 이슈 ${high}건 발견 — 즉시 수리를 권해요.`,
      level: 'critical' }
  }
  if (d.score < 80 || high >= 1) {
    return { text: lang === 'en'
        ? `Found ${d.issues.length} issues — review and fix priorities.`
        : `이슈 ${d.issues.length}건 발견 — 우선순위 확인 후 수리하세요.`,
      level: 'warning' }
  }
  if (d.score >= 90) {
    return { text: lang === 'en' ? 'Security looks healthy.' : '보안 상태 양호해요.', level: 'success' }
  }
  return null
}

/* ── 중복 파일 ────────────────────────────────────────── */
export interface DuplicatesLike { total_groups: number; waste_mb: number }
export function insightForDuplicates(d: DuplicatesLike, lang: 'ko' | 'en' = 'ko'): { text: string; level: InsightLevel } | null {
  if (d.waste_mb >= 1000) {
    return { text: lang === 'en'
        ? `${(d.waste_mb/1024).toFixed(1)} GB wasted across ${d.total_groups} duplicate sets — significant cleanup opportunity.`
        : `${d.total_groups}개 중복 그룹에서 ${(d.waste_mb/1024).toFixed(1)} GB 낭비 중 — 정리하면 큰 공간 확보!`,
      level: 'warning' }
  }
  if (d.waste_mb >= 100) {
    return { text: lang === 'en'
        ? `~${d.waste_mb} MB recoverable. Worth a cleanup.`
        : `${d.waste_mb} MB 회복 가능 — 정리 권장.`,
      level: 'tip' }
  }
  if (d.total_groups === 0) {
    return { text: lang === 'en' ? 'No duplicates found — disk is tidy.' : '중복 파일 없음 — 디스크 정리 상태 양호!', level: 'success' }
  }
  return null
}

/* ── 프로세스 TOP ─────────────────────────────────────── */
export interface ProcessTopLike { by_cpu: Array<{ name: string; cpu: number; mem_mb: number }> }
export function insightForProcessTop(d: ProcessTopLike, lang: 'ko' | 'en' = 'ko'): { text: string; level: InsightLevel } | null {
  if (!d.by_cpu || d.by_cpu.length === 0) return null
  const top = d.by_cpu[0]
  if (top.cpu >= 80) {
    return { text: lang === 'en'
        ? `${top.name} is using ${top.cpu.toFixed(0)}% CPU — consider closing if not needed.`
        : `${top.name}가 CPU ${top.cpu.toFixed(0)}% 사용 중 — 필요 없으면 종료해주세요.`,
      level: 'warning' }
  }
  if (top.mem_mb >= 2048) {
    return { text: lang === 'en'
        ? `${top.name} consumes ${(top.mem_mb/1024).toFixed(1)} GB memory — heaviest app right now.`
        : `${top.name}가 메모리 ${(top.mem_mb/1024).toFixed(1)} GB 사용 — 현재 가장 무거운 앱.`,
      level: 'tip' }
  }
  return null
}

/* ── 네트워크 분석 ────────────────────────────────────── */
export interface NetworkLike { connected: boolean; ping_ms: string; public_ip: string }
export function insightForNetwork(d: NetworkLike, lang: 'ko' | 'en' = 'ko'): { text: string; level: InsightLevel } | null {
  if (!d.connected) {
    return { text: lang === 'en' ? 'No internet connection detected.' : '인터넷 연결 안 됨 — Wi-Fi/유선 확인 필요.', level: 'critical' }
  }
  const ping = parseInt(d.ping_ms) || 0
  if (ping >= 200) {
    return { text: lang === 'en'
        ? `Ping ${ping}ms — slow. Check Wi-Fi signal or router.`
        : `핑 ${ping}ms 으로 느림 — Wi-Fi 신호나 라우터 확인 권장.`,
      level: 'warning' }
  }
  if (ping > 0 && ping < 30) {
    return { text: lang === 'en' ? `Excellent connection (${ping}ms).` : `네트워크 양호 (${ping}ms).`, level: 'success' }
  }
  return null
}

/* ── 드라이버 점검 ────────────────────────────────────── */
export interface DriversLike { total: number; problem_count: number; score: number }
export function insightForDrivers(d: DriversLike, lang: 'ko' | 'en' = 'ko'): { text: string; level: InsightLevel } | null {
  if (d.problem_count >= 3) {
    return { text: lang === 'en'
        ? `${d.problem_count} drivers need attention — system instability likely.`
        : `드라이버 ${d.problem_count}개 문제 — 시스템 불안정 원인 가능성.`,
      level: 'critical' }
  }
  if (d.problem_count >= 1) {
    return { text: lang === 'en'
        ? `${d.problem_count} driver(s) flagged — check Device Manager.`
        : `드라이버 ${d.problem_count}개 주의 — 장치관리자에서 확인 권장.`,
      level: 'warning' }
  }
  if (d.score >= 95) {
    return { text: lang === 'en' ? `All ${d.total} drivers healthy.` : `드라이버 ${d.total}개 모두 정상.`, level: 'success' }
  }
  return null
}

/* ── 날씨 ─────────────────────────────────────────────── */
export interface WeatherLike { temp_c: number; condition: string; humidity?: number }
export function insightForWeather(d: WeatherLike, lang: 'ko' | 'en' = 'ko'): { text: string; level: InsightLevel } | null {
  if (d.temp_c >= 33) {
    return { text: lang === 'en' ? 'Very hot today — stay hydrated, limit outdoor time.' : '폭염 — 외출 자제, 충분한 수분 섭취 권장.', level: 'warning' }
  }
  if (d.temp_c <= -5) {
    return { text: lang === 'en' ? 'Freezing — bundle up, watch for icy roads.' : '강추위 — 보온 철저, 빙판 주의.', level: 'warning' }
  }
  if (/비|rain/i.test(d.condition)) {
    return { text: lang === 'en' ? 'Rainy — grab an umbrella.' : '비 예보 — 우산 챙기세요.', level: 'tip' }
  }
  if (/눈|snow/i.test(d.condition)) {
    return { text: lang === 'en' ? 'Snow — drive carefully and dress warmly.' : '눈 예보 — 운전 주의, 보온 챙기세요.', level: 'tip' }
  }
  if (d.temp_c >= 18 && d.temp_c <= 26 && !/비|눈|폭염|rain|snow/i.test(d.condition)) {
    return { text: lang === 'en' ? 'Beautiful weather — perfect for outdoor activities.' : '쾌적한 날씨 — 야외 활동하기 좋아요!', level: 'success' }
  }
  return null
}

/* ── VirusTotal ───────────────────────────────────────── */
export interface VirusLike { malicious: number; suspicious: number; clean: number; total_scans: number; verdict?: string }
export function insightForVirus(d: VirusLike, lang: 'ko' | 'en' = 'ko'): { text: string; level: InsightLevel } | null {
  if (d.malicious >= 1) {
    return { text: lang === 'en'
        ? `🦠 ${d.malicious} engines flagged as malware — DO NOT EXECUTE this file.`
        : `🦠 ${d.malicious}개 엔진이 악성코드로 판정 — 이 파일을 실행하지 마세요!`,
      level: 'critical' }
  }
  if (d.suspicious >= 2) {
    return { text: lang === 'en'
        ? `${d.suspicious} engines marked suspicious — investigate before opening.`
        : `${d.suspicious}개 엔진이 의심 판정 — 신중히 확인 후 실행하세요.`,
      level: 'warning' }
  }
  if (d.malicious === 0 && d.suspicious === 0 && d.total_scans > 0) {
    return { text: lang === 'en'
        ? `Clean — ${d.total_scans} engines reported no threats.`
        : `안전 — ${d.total_scans}개 엔진 모두 위협 없음.`,
      level: 'success' }
  }
  return null
}

/* ── 성능 이력 / 이상 감지 ───────────────────────────── */
export interface PerfHistoryLike { avg_cpu: number; avg_mem: number; cpu_trend?: string; total_samples?: number }
export function insightForPerfHistory(d: PerfHistoryLike, lang: 'ko' | 'en' = 'ko'): { text: string; level: InsightLevel } | null {
  if ((d.total_samples ?? 0) < 10) {
    return { text: lang === 'en' ? 'Not enough data yet — keep using the app to build history.' : '아직 데이터 부족 — 앱을 더 사용하면 패턴이 보여요.', level: 'info' }
  }
  if (d.cpu_trend === 'up' && d.avg_cpu > 60) {
    return { text: lang === 'en' ? 'CPU usage trending up — something is gradually using more resources.' : 'CPU 사용량 증가 추세 — 무언가 점점 더 많이 쓰고 있어요.', level: 'warning' }
  }
  if (d.avg_cpu < 40 && d.avg_mem < 60) {
    return { text: lang === 'en' ? 'Healthy long-term usage pattern.' : '장기 사용 패턴 양호.', level: 'success' }
  }
  return null
}

/* ── 부팅 분석 ────────────────────────────────────────── */
export interface BootLike { uptime_minutes: string; startup_count: string }
export function insightForBoot(d: BootLike, lang: 'ko' | 'en' = 'ko'): { text: string; level: InsightLevel } | null {
  const startups = parseInt(d.startup_count) || 0
  if (startups >= 20) {
    return { text: lang === 'en'
        ? `${startups} startup items — disable unnecessary ones to boot faster.`
        : `시작 프로그램 ${startups}개 — 필요 없는 것 끄면 부팅 속도 개선.`,
      level: 'warning' }
  }
  const uptime = parseInt(d.uptime_minutes) || 0
  if (uptime > 60 * 24 * 7) {
    return { text: lang === 'en'
        ? `Uptime > 7 days — consider rebooting for memory cleanup.`
        : `7일 넘게 켜둔 상태 — 한 번 재부팅으로 메모리 정리 권장.`,
      level: 'tip' }
  }
  return null
}

/* ── 이메일 받은편지함 ────────────────────────────────── */
export interface EmailInboxLike { total: number; unread: number }
export function insightForEmailInbox(d: EmailInboxLike, lang: 'ko' | 'en' = 'ko'): { text: string; level: InsightLevel } | null {
  if (d.unread >= 50) {
    return { text: lang === 'en'
        ? `${d.unread} unread — try filtering by sender or use Classify Email.`
        : `안읽음 ${d.unread}개 — 발신자별 필터링이나 메일 분류 기능 추천.`,
      level: 'warning' }
  }
  if (d.unread === 0 && d.total > 0) {
    return { text: lang === 'en' ? 'Inbox zero — well done!' : '받은편지함 비어있음 — 축하해요! 🎉', level: 'success' }
  }
  return null
}

/* ── 가격 비교 ────────────────────────────────────────── */
export interface PriceCompareLike { results: Array<{ price: string; site: string; blocked?: boolean }> }
export function insightForPriceCompare(d: PriceCompareLike, lang: 'ko' | 'en' = 'ko'): { text: string; level: InsightLevel } | null {
  const valid = d.results.filter(r => !r.blocked && r.price)
  if (valid.length < 2) return null
  const prices = valid
    .map(r => parseInt(r.price.replace(/[^0-9]/g, '')) || 0)
    .filter(p => p > 0)
    .sort((a, b) => a - b)
  if (prices.length < 2) return null
  const min = prices[0], max = prices[prices.length - 1]
  const diffPct = Math.round(((max - min) / max) * 100)
  if (diffPct >= 30) {
    return { text: lang === 'en'
        ? `Best price ${diffPct}% cheaper than highest — significant savings available.`
        : `최저가가 최고가보다 ${diffPct}% 저렴 — 큰 절약 가능!`,
      level: 'success' }
  }
  if (diffPct >= 10) {
    return { text: lang === 'en'
        ? `${diffPct}% spread between sites.`
        : `사이트 간 ${diffPct}% 가격 차이.`,
      level: 'tip' }
  }
  return null
}
