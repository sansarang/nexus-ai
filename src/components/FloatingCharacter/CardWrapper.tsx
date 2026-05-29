import { motion } from 'framer-motion'
import React from 'react'

export type CardVariant = 'default' | 'success' | 'warning' | 'info' | 'search' | 'dark'

const variantStyles: Record<CardVariant, React.CSSProperties> = {
  default: { background: 'rgba(255,255,255,0.06)', border: '1px solid rgba(255,255,255,0.1)' },
  success: { background: 'rgba(72,187,120,0.07)', border: '1px solid rgba(72,187,120,0.25)' },
  warning: { background: 'rgba(237,137,54,0.07)', border: '1px solid rgba(237,137,54,0.25)' },
  info:    { background: 'rgba(99,179,237,0.07)', border: '1px solid rgba(99,179,237,0.25)' },
  search:  { background: 'rgba(8,8,22,0.97)',     border: '1px solid rgba(255,255,255,0.1)' },
  dark:    { background: 'rgba(5,5,15,0.95)',     border: '1px solid rgba(255,255,255,0.08)' },
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
    ...(accentColor ? { border: `1px solid ${accentColor}33` } : {}),
    borderRadius: 12,
    padding: '12px 14px',
    marginTop: 8,
    fontSize: 13,
    color: '#e2e8f0',
    width: 'clamp(240px, 100%, 420px)',
    lineHeight: 1.55,
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
