/**
 * CardHeader — 모든 결과 카드의 통일된 헤더.
 *
 * 구조:
 *   [icon] [Intent 라벨]  [Status Badge]              [⋯ 메뉴]
 *
 * Intent Registry 의 emoji + labelKo/En 을 자동으로 사용해
 * "어떤 작업의 결과인지" 사용자가 한눈에 파악.
 */

import { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { getIntentSpec } from '../../../lib/nexus/intentRegistry'
import { StatusBadge, type BadgeType, relativeTimeLabel } from './StatusBadge'

export interface CardAction {
  label: string
  icon?: string
  onClick: () => void
  variant?: 'default' | 'primary' | 'danger'
}

interface CardHeaderProps {
  /** Intent 키 — registry 에서 emoji/label 자동 로드 */
  intent?: string
  /** 직접 제목 지정 (intent 없을 때) */
  title?: string
  /** 직접 아이콘 지정 (intent 없을 때) */
  icon?: string
  /** 상태 뱃지 (live/loading/error 등) */
  status?: BadgeType
  /** 상태 라벨 커스텀 (예: "3분 전") */
  statusLabel?: string
  /** 데이터 timestamp — relativeTimeLabel 로 "N초 전" 자동 표시 */
  timestamp?: number | string
  /** 액션 메뉴 항목 (⋯ 클릭 시 드롭다운) */
  actions?: CardAction[]
  /** 헤더 우측에 항상 표시되는 빠른 액션 (1-2개) */
  quickActions?: CardAction[]
  accentColor?: string
  lang?: 'ko' | 'en'
}

export function CardHeader({
  intent, title, icon, status, statusLabel, timestamp,
  actions, quickActions, accentColor = '#9b59b6', lang = 'ko',
}: CardHeaderProps) {
  const [menuOpen, setMenuOpen] = useState(false)
  const spec = intent ? getIntentSpec(intent) : null

  const displayIcon = icon ?? spec?.emoji ?? '📋'
  const displayTitle = title ?? (spec ? (lang === 'en' ? spec.labelEn : spec.labelKo) : 'Unknown')
  const relTime = timestamp ? relativeTimeLabel(timestamp, lang) : ''

  return (
    <div style={{
      display: 'flex', alignItems: 'center', gap: 8,
      marginBottom: 8, paddingBottom: 8,
      borderBottom: '1px solid rgba(255,255,255,0.08)',
    }}>
      {/* 아이콘 */}
      <span style={{ fontSize: 15, flexShrink: 0 }}>{displayIcon}</span>

      {/* 제목 + 상태 */}
      <div style={{ flex: 1, minWidth: 0, display: 'flex', alignItems: 'center', gap: 6 }}>
        <span style={{
          fontSize: 11, fontWeight: 700, color: 'rgba(255,255,255,0.92)',
          overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
        }}>
          {displayTitle}
        </span>
        {status && (
          <StatusBadge
            type={status}
            label={statusLabel ?? (relTime || undefined)}
            lang={lang}
            pulse={status === 'loading' || status === 'live'}
          />
        )}
      </div>

      {/* Quick actions */}
      {quickActions?.map((a, i) => (
        <button
          key={i}
          onClick={a.onClick}
          title={a.label}
          style={{
            background: a.variant === 'primary' ? `${accentColor}33` : 'rgba(255,255,255,0.06)',
            border: `1px solid ${a.variant === 'primary' ? accentColor + '66' : 'rgba(255,255,255,0.1)'}`,
            color: a.variant === 'primary' ? accentColor : 'rgba(255,255,255,0.7)',
            borderRadius: 6, padding: '3px 8px', fontSize: 10, fontWeight: 600,
            cursor: 'pointer', whiteSpace: 'nowrap',
            transition: 'all 0.15s',
          }}
          onMouseEnter={e => { e.currentTarget.style.background = `${accentColor}33`; e.currentTarget.style.borderColor = `${accentColor}66` }}
          onMouseLeave={e => {
            e.currentTarget.style.background = a.variant === 'primary' ? `${accentColor}33` : 'rgba(255,255,255,0.06)'
            e.currentTarget.style.borderColor = a.variant === 'primary' ? accentColor + '66' : 'rgba(255,255,255,0.1)'
          }}
        >
          {a.icon && <span style={{ marginRight: 3 }}>{a.icon}</span>}
          {a.label}
        </button>
      ))}

      {/* 메뉴 ⋯ */}
      {actions && actions.length > 0 && (
        <div style={{ position: 'relative' }}>
          <button
            onClick={() => setMenuOpen(v => !v)}
            style={{
              background: 'transparent', border: 'none',
              color: 'rgba(255,255,255,0.5)', fontSize: 14, cursor: 'pointer',
              padding: '2px 6px', borderRadius: 4,
            }}
          >
            ⋯
          </button>
          <AnimatePresence>
            {menuOpen && (
              <motion.div
                initial={{ opacity: 0, y: -4 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: -4 }}
                onMouseLeave={() => setMenuOpen(false)}
                style={{
                  position: 'absolute', right: 0, top: '100%', marginTop: 4,
                  background: 'rgba(10,10,20,0.97)',
                  border: '1px solid rgba(255,255,255,0.12)',
                  borderRadius: 8, padding: 4, minWidth: 140,
                  boxShadow: '0 8px 24px rgba(0,0,0,0.5)',
                  zIndex: 100,
                }}
              >
                {actions.map((a, i) => (
                  <button
                    key={i}
                    onClick={() => { a.onClick(); setMenuOpen(false) }}
                    style={{
                      display: 'flex', alignItems: 'center', gap: 8, width: '100%',
                      background: 'transparent', border: 'none',
                      color: a.variant === 'danger' ? '#ef4444' : 'rgba(255,255,255,0.85)',
                      padding: '7px 10px', borderRadius: 6, fontSize: 11,
                      cursor: 'pointer', textAlign: 'left',
                      transition: 'background 0.15s',
                    }}
                    onMouseEnter={e => { e.currentTarget.style.background = 'rgba(255,255,255,0.08)' }}
                    onMouseLeave={e => { e.currentTarget.style.background = 'transparent' }}
                  >
                    {a.icon && <span>{a.icon}</span>}
                    <span>{a.label}</span>
                  </button>
                ))}
              </motion.div>
            )}
          </AnimatePresence>
        </div>
      )}
    </div>
  )
}
