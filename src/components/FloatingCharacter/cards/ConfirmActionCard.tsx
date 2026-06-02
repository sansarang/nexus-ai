/**
 * ConfirmActionCard — Phase A2: 위험 액션 확인 카드
 *
 * 백엔드가 needs_confirmation: true 응답 보내면 이 카드로 표시
 * 사용자가 [실행] 클릭 → confirmed:true 파라미터 추가해서 같은 명령 재요청
 * [취소] 클릭 → 카드 닫기
 */

import { motion } from 'framer-motion'

export interface ConfirmActionData {
  action: string
  severity: 'low' | 'medium' | 'high' | 'critical'
  title: string
  description: string
  reversible: boolean
  pending_params?: Record<string, unknown>
}

interface ConfirmActionCardProps {
  data: ConfirmActionData
  onConfirm: (action: string, params: Record<string, unknown>) => void
  onCancel: () => void
}

const severityColors: Record<string, { bg: string; border: string; text: string }> = {
  low:      { bg: 'rgba(99,102,241,0.10)',  border: 'rgba(99,102,241,0.40)',  text: '#a78bfa' },
  medium:   { bg: 'rgba(245,158,11,0.10)',  border: 'rgba(245,158,11,0.40)',  text: '#fbbf24' },
  high:     { bg: 'rgba(239,68,68,0.12)',   border: 'rgba(239,68,68,0.45)',   text: '#fca5a5' },
  critical: { bg: 'rgba(220,38,38,0.18)',   border: 'rgba(220,38,38,0.60)',   text: '#fecaca' },
}

export function ConfirmActionCard({ data, onConfirm, onCancel }: ConfirmActionCardProps) {
  const c = severityColors[data.severity] ?? severityColors.medium

  return (
    <motion.div
      initial={{ opacity: 0, y: 8, scale: 0.96 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      transition={{ duration: 0.22 }}
      style={{
        background: c.bg,
        backdropFilter: 'blur(40px) saturate(1.6)',
        WebkitBackdropFilter: 'blur(40px) saturate(1.6)',
        border: `1px solid ${c.border}`,
        borderTop: `3px solid ${c.text}`,
        borderRadius: 18,
        padding: '18px 20px',
        display: 'flex',
        flexDirection: 'column',
        gap: 14,
        boxShadow: '0 8px 32px rgba(0,0,0,0.4), inset 0 1px 0 rgba(255,255,255,0.05)',
        color: '#f1f5f9',
      }}
    >
      {/* 헤더 */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <div style={{ fontSize: 16, fontWeight: 800, color: c.text }}>
          {data.title}
        </div>
        <div style={{
          fontSize: 10, fontWeight: 700, padding: '4px 8px', borderRadius: 999,
          background: c.border, color: '#fff',
          textTransform: 'uppercase', letterSpacing: '0.05em',
        }}>
          {data.severity}
        </div>
      </div>

      {/* 설명 */}
      <div style={{ fontSize: 12.5, color: 'rgba(255,255,255,0.85)', lineHeight: 1.6 }}>
        {data.description}
      </div>

      {/* 되돌릴 수 있는지 여부 */}
      <div style={{
        fontSize: 10.5, color: 'rgba(255,255,255,0.55)',
        display: 'flex', alignItems: 'center', gap: 5,
      }}>
        {data.reversible
          ? <>↩️ 되돌릴 수 있는 작업</>
          : <>⚠️ 되돌릴 수 없는 작업</>}
      </div>

      {/* 버튼 */}
      <div style={{ display: 'flex', gap: 8, marginTop: 4 }}>
        <button
          onClick={onCancel}
          style={{
            flex: 1, padding: '10px 0', borderRadius: 12,
            background: 'rgba(255,255,255,0.08)',
            color: 'rgba(255,255,255,0.85)',
            border: '1px solid rgba(255,255,255,0.10)',
            fontSize: 12, fontWeight: 700, cursor: 'pointer',
          }}
        >
          취소
        </button>
        <button
          onClick={() => onConfirm(data.action, { ...(data.pending_params ?? {}), confirmed: true })}
          style={{
            flex: 1, padding: '10px 0', borderRadius: 12,
            background: c.text, color: '#0f172a',
            border: 'none',
            fontSize: 12, fontWeight: 800, cursor: 'pointer',
            boxShadow: `0 4px 16px ${c.border}`,
          }}
        >
          {data.severity === 'critical' ? '확인 후 실행' : '실행'}
        </button>
      </div>
    </motion.div>
  )
}
