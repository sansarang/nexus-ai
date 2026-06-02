/**
 * CardShell — 49카드 공통 글래스 셸 (Apple 날씨 위젯 스타일)
 *
 * 모든 카드를 이 셸로 감싸면 디자인 일관성 보장:
 *   - 글래스모피즘 (rgba + backdropFilter)
 *   - 미세 테두리 + 그림자
 *   - 호버 시 살짝 떠오름 + 강조 테두리
 *   - 헤더 (icon + title) + body + footer (옵션)
 *
 * 사용:
 *   <CardShell icon="📊" title="PC 상태" onClick={openCanvas}>
 *     <div>CPU 24% / MEM 68% / DISK 35%</div>
 *   </CardShell>
 *
 * 사이즈: variant="compact"  (위젯 크기, 140px 높이)
 *         variant="standard" (기본, auto 높이)
 *         variant="hero"     (큰 카드, 280px+ 높이)
 */

import React, { useState } from 'react'
import { NEXUS } from '../../../theme/nexus'

export interface CardShellProps {
  icon?: React.ReactNode
  title?: React.ReactNode
  subtitle?: React.ReactNode
  badge?: React.ReactNode             // 우상단 작은 배지 (성공/경고/시간 등)
  variant?: 'compact' | 'standard' | 'hero'
  accent?: 'default' | 'success' | 'warning' | 'critical' | 'brand'
  onClick?: () => void
  onClose?: () => void                // 우상단 ✕ 버튼
  pinned?: boolean                    // 핀 표시 (📌)
  children: React.ReactNode
  style?: React.CSSProperties
}

const ACCENT_COLORS: Record<NonNullable<CardShellProps['accent']>, string> = {
  default:  NEXUS.border.subtle,
  success:  'rgba(34, 197, 94, 0.45)',
  warning:  'rgba(251, 191, 36, 0.45)',
  critical: 'rgba(239, 68, 68, 0.5)',
  brand:    NEXUS.border.accent,
}

export function CardShell({
  icon, title, subtitle, badge,
  variant = 'standard',
  accent = 'default',
  onClick, onClose, pinned,
  children, style,
}: CardShellProps) {
  const [hovered, setHovered] = useState(false)
  const clickable = !!onClick

  const minHeight = variant === 'compact' ? 130 : variant === 'hero' ? 280 : undefined
  const accentBorder = ACCENT_COLORS[accent]

  return (
    <div
      onClick={onClick}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        position: 'relative',
        background: hovered ? NEXUS.bg.cardHover : NEXUS.bg.card,
        backdropFilter: NEXUS.blur.standard,
        WebkitBackdropFilter: NEXUS.blur.standard,
        border: `1px solid ${hovered && clickable ? NEXUS.border.accent : accentBorder}`,
        borderRadius: NEXUS.radius.lg,
        padding: variant === 'compact' ? `${NEXUS.spacing.md}px ${NEXUS.spacing.lg}px` : `${NEXUS.spacing.lg}px`,
        minHeight,
        cursor: clickable ? 'pointer' : 'default',
        transition: `all ${NEXUS.motion.standard}`,
        boxShadow: hovered ? NEXUS.shadow.cardHover : NEXUS.shadow.card,
        transform: hovered && clickable ? 'translateY(-1px)' : 'translateY(0)',
        overflow: 'hidden',
        display: 'flex',
        flexDirection: 'column',
        gap: NEXUS.spacing.sm,
        ...style,
      }}
    >
      {/* ── 헤더 (아이콘 + 타이틀 + 배지/액션) ── */}
      {(icon || title || badge || onClose || pinned) && (
        <div style={{
          display: 'flex',
          alignItems: 'center',
          gap: NEXUS.spacing.sm,
          fontSize: NEXUS.font.size.sm,
          color: NEXUS.text.secondary,
          fontWeight: NEXUS.font.weight.semi,
          letterSpacing: '0.02em',
        }}>
          {icon && (
            <span style={{ fontSize: 14, flexShrink: 0, opacity: 0.85 }}>
              {icon}
            </span>
          )}
          {title && (
            <span style={{
              flex: 1,
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
              textTransform: 'uppercase',
            }}>
              {title}
            </span>
          )}
          {pinned && <span style={{ fontSize: 11, opacity: 0.7 }}>📌</span>}
          {badge && (
            <span style={{ fontSize: NEXUS.font.size.xs, color: NEXUS.text.tertiary }}>
              {badge}
            </span>
          )}
          {onClose && (
            <button
              onClick={(e) => { e.stopPropagation(); onClose() }}
              style={{
                background: 'transparent',
                border: 'none',
                color: NEXUS.text.tertiary,
                cursor: 'pointer',
                fontSize: 14,
                padding: '2px 4px',
                borderRadius: 4,
                marginLeft: 4,
                transition: `color ${NEXUS.motion.fast}`,
              }}
              onMouseEnter={(e) => { e.currentTarget.style.color = '#ef4444' }}
              onMouseLeave={(e) => { e.currentTarget.style.color = NEXUS.text.tertiary }}
              aria-label="close"
            >
              ✕
            </button>
          )}
        </div>
      )}
      {subtitle && (
        <div style={{
          fontSize: NEXUS.font.size.xs,
          color: NEXUS.text.tertiary,
          marginTop: -4,
        }}>
          {subtitle}
        </div>
      )}

      {/* ── 본문 ── */}
      <div style={{
        flex: 1,
        color: NEXUS.text.primary,
        fontSize: NEXUS.font.size.base,
        lineHeight: 1.55,
      }}>
        {children}
      </div>
    </div>
  )
}
