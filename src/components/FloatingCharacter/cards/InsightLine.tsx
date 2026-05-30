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
