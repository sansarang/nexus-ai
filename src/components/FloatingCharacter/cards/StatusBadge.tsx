/**
 * StatusBadge — 카드 헤더에 일관된 상태 표시.
 *
 * 색맹 사용자를 위해 색 + 아이콘 + 텍스트 3중으로 의미 전달.
 */

import { motion } from 'framer-motion'

export type BadgeType =
  | 'live'      // 실시간 데이터
  | 'cached'    // 캐시된 데이터 (몇분 전 데이터)
  | 'loading'   // 조회 중
  | 'success'   // 성공 (완료)
  | 'warning'   // 주의
  | 'error'     // 실패
  | 'windows'   // Windows 전용
  | 'beta'      // 베타 기능
  | 'pro'       // Pro 구독 전용
  | 'info'      // 일반 정보

const BADGE_META: Record<BadgeType, { icon: string; color: string; ko: string; en: string }> = {
  live:    { icon: '🟢', color: '#22c55e', ko: '실시간',     en: 'LIVE' },
  cached:  { icon: '🕒', color: '#94a3b8', ko: '캐시됨',     en: 'CACHED' },
  loading: { icon: '⚪', color: '#94a3b8', ko: '조회 중',    en: 'LOADING' },
  success: { icon: '✅', color: '#22c55e', ko: '완료',       en: 'DONE' },
  warning: { icon: '🟡', color: '#f59e0b', ko: '주의',       en: 'WARN' },
  error:   { icon: '🔴', color: '#ef4444', ko: '실패',       en: 'ERROR' },
  windows: { icon: '🪟', color: '#3b82f6', ko: 'Win 전용',   en: 'WIN-ONLY' },
  beta:    { icon: '🧪', color: '#a855f7', ko: 'BETA',       en: 'BETA' },
  pro:     { icon: '⭐', color: '#eab308', ko: 'PRO',        en: 'PRO' },
  info:    { icon: 'ℹ️', color: '#3b82f6', ko: '안내',       en: 'INFO' },
}

interface StatusBadgeProps {
  type: BadgeType
  /** 커스텀 라벨 (예: "3분 전"). 없으면 type별 기본 라벨 사용. */
  label?: string
  /** ko (기본) 또는 en */
  lang?: 'ko' | 'en'
  /** 펄스 애니메이션 (loading/live 등) */
  pulse?: boolean
  size?: 'sm' | 'md'
}

export function StatusBadge({ type, label, lang = 'ko', pulse = false, size = 'sm' }: StatusBadgeProps) {
  const meta = BADGE_META[type]
  const text = label ?? (lang === 'en' ? meta.en : meta.ko)
  const fontSize = size === 'sm' ? 9 : 10
  const iconSize = size === 'sm' ? 9 : 11

  return (
    <motion.span
      initial={{ opacity: 0, scale: 0.9 }}
      animate={{ opacity: 1, scale: 1 }}
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 3,
        padding: size === 'sm' ? '2px 6px' : '3px 8px',
        borderRadius: 6,
        background: `${meta.color}1a`,
        border: `1px solid ${meta.color}44`,
        color: meta.color,
        fontSize,
        fontWeight: 700,
        letterSpacing: '0.04em',
        whiteSpace: 'nowrap',
        lineHeight: 1,
      }}
    >
      <span style={{ fontSize: iconSize }}>
        {pulse ? (
          <motion.span
            animate={{ opacity: [1, 0.4, 1] }}
            transition={{ duration: 1.4, repeat: Infinity }}
            style={{ display: 'inline-block' }}
          >
            {meta.icon}
          </motion.span>
        ) : meta.icon}
      </span>
      {text}
    </motion.span>
  )
}

/** "N분 전" / "방금" 등 상대 시간 라벨 */
export function relativeTimeLabel(ts: number | string | undefined, lang: 'ko' | 'en' = 'ko'): string {
  if (!ts) return ''
  const t = typeof ts === 'string' ? new Date(ts).getTime() : ts
  if (!t || isNaN(t)) return ''
  const diff = Math.floor((Date.now() - t) / 1000)
  if (diff < 5) return lang === 'en' ? 'just now' : '방금'
  if (diff < 60) return lang === 'en' ? `${diff}s ago` : `${diff}초 전`
  if (diff < 3600) return lang === 'en' ? `${Math.floor(diff/60)}m ago` : `${Math.floor(diff/60)}분 전`
  if (diff < 86400) return lang === 'en' ? `${Math.floor(diff/3600)}h ago` : `${Math.floor(diff/3600)}시간 전`
  return lang === 'en' ? `${Math.floor(diff/86400)}d ago` : `${Math.floor(diff/86400)}일 전`
}
