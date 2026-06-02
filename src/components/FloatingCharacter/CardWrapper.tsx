import { motion } from 'framer-motion'
import React from 'react'

export type CardVariant = 'default' | 'success' | 'warning' | 'info' | 'search' | 'dark'

// NEXUS 글래스 스타일 (Apple 날씨 영감) — 모든 카드 통일
const GLASS_BASE: React.CSSProperties = {
  backdropFilter: 'blur(40px) saturate(1.8)',
  WebkitBackdropFilter: 'blur(40px) saturate(1.8)',
  boxShadow: '0 8px 32px rgba(0,0,0,0.35), inset 0 1px 0 rgba(255,255,255,0.06)',
}
const variantStyles: Record<CardVariant, React.CSSProperties> = {
  default: { ...GLASS_BASE, background: 'rgba(255,255,255,0.06)', border: '1px solid rgba(255,255,255,0.10)', borderTop: '2px solid rgba(139,92,246,0.55)' },
  success: { ...GLASS_BASE, background: 'rgba(255,255,255,0.05)', border: '1px solid rgba(255,255,255,0.08)', borderTop: '2px solid rgba(34,197,94,0.65)' },
  warning: { ...GLASS_BASE, background: 'rgba(255,255,255,0.05)', border: '1px solid rgba(255,255,255,0.08)', borderTop: '2px solid rgba(245,158,11,0.65)' },
  info:    { ...GLASS_BASE, background: 'rgba(255,255,255,0.05)', border: '1px solid rgba(255,255,255,0.08)', borderTop: '2px solid rgba(96,165,250,0.65)' },
  search:  { ...GLASS_BASE, background: 'rgba(255,255,255,0.06)', border: '1px solid rgba(255,255,255,0.10)', borderTop: '2px solid rgba(168,85,247,0.55)' },
  dark:    { ...GLASS_BASE, background: 'rgba(255,255,255,0.04)', border: '1px solid rgba(255,255,255,0.06)' },
}

interface CardWrapperProps {
  children: React.ReactNode
  variant?: CardVariant
  accentColor?: string
  style?: React.CSSProperties
  animate?: boolean
}

export function CardWrapper({
  children,
  variant = 'default',
  accentColor,
  style,
  animate = true,
}: CardWrapperProps) {
  const base: React.CSSProperties = {
    ...variantStyles[variant],
    ...(accentColor ? { borderTop: `2px solid ${accentColor}88` } : {}),
    borderRadius: 18,
    padding: '14px 16px',
    marginTop: 8,
    fontSize: 13,
    color: '#f1f5f9',
    width: 'clamp(240px, 100%, 460px)',
    lineHeight: 1.6,
    boxSizing: 'border-box',
    ...style,
  }

  if (!animate) return <div style={base}>{children}</div>

  return (
    <motion.div
      initial={{ opacity: 0, y: 8, scale: 0.97 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      transition={{ duration: 0.22, ease: 'easeOut' }}
      style={base}
    >
      {children}
    </motion.div>
  )
}
