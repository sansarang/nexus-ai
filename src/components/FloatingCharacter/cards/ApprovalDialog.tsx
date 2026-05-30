/**
 * ApprovalDialog — 데스크톱 제어 첫 사용 시 승인 모드 선택.
 *
 * 사용자가 "여기 클릭" / "Chrome 최대화" 같은 PC 조작 명령을 처음 내릴 때
 * 다음 4가지 중 선택하게 함:
 *  - always  : 매번 확인 (가장 안전)
 *  - trust5m : 5분 신뢰
 *  - trust1h : 1시간 신뢰
 *  - never   : 영구 신뢰
 */

import { motion, AnimatePresence } from 'framer-motion'
import { setApprovalMode, type ApprovalMode } from '../../../lib/nexus/approvalMode'

interface ApprovalDialogProps {
  open: boolean
  /** 어떤 작업을 하려는지 사용자에게 보여줄 텍스트 */
  pendingAction?: string
  onChoose: (mode: ApprovalMode, approveThisOne: boolean) => void
  onCancel: () => void
  lang?: 'ko' | 'en'
}

const MODES: Array<{
  id: ApprovalMode
  icon: string
  ko: { label: string; desc: string }
  en: { label: string; desc: string }
  color: string
  recommended?: boolean
}> = [
  {
    id: 'always',
    icon: '🛡️',
    ko: { label: '매번 확인', desc: '모든 PC 조작 직전 묻기 (가장 안전)' },
    en: { label: 'Always Ask', desc: 'Confirm every PC action (safest)' },
    color: '#22c55e',
    recommended: true,
  },
  {
    id: 'trust5m',
    icon: '⏱️',
    ko: { label: '5분 신뢰', desc: '5분 동안 자동 실행 (편의)' },
    en: { label: 'Trust 5min', desc: 'Auto-execute for 5 minutes' },
    color: '#3b82f6',
  },
  {
    id: 'trust1h',
    icon: '🕐',
    ko: { label: '1시간 신뢰', desc: '집중 작업 시 추천' },
    en: { label: 'Trust 1hr', desc: 'For focused work sessions' },
    color: '#a855f7',
  },
  {
    id: 'never',
    icon: '🔓',
    ko: { label: '영구 신뢰', desc: '항상 자동 실행 (Pro 사용자 권장)' },
    en: { label: 'Always Trust', desc: 'Auto-execute forever (Pro)' },
    color: '#f59e0b',
  },
]

export function ApprovalDialog({ open, pendingAction, onChoose, onCancel, lang = 'ko' }: ApprovalDialogProps) {
  if (!open) return null

  const handleChoose = (mode: ApprovalMode) => {
    setApprovalMode(mode)
    onChoose(mode, true)
  }

  return (
    <AnimatePresence>
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        style={{
          position: 'fixed', inset: 0, zIndex: 9999,
          background: 'rgba(0,0,0,0.7)', backdropFilter: 'blur(4px)',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          padding: 20,
        }}
        onClick={onCancel}
      >
        <motion.div
          initial={{ opacity: 0, y: 20, scale: 0.95 }}
          animate={{ opacity: 1, y: 0, scale: 1 }}
          exit={{ opacity: 0, y: 20, scale: 0.95 }}
          onClick={e => e.stopPropagation()}
          style={{
            background: '#0f0f1e',
            border: '1px solid rgba(255,255,255,0.12)',
            borderRadius: 16, padding: 24,
            maxWidth: 480, width: '100%',
            boxShadow: '0 20px 60px rgba(0,0,0,0.6)',
          }}
        >
          {/* 헤더 */}
          <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 16 }}>
            <span style={{ fontSize: 32 }}>🖱️</span>
            <div>
              <div style={{ fontSize: 16, fontWeight: 800, color: '#fff', marginBottom: 2 }}>
                {lang === 'en' ? 'PC Control Permission' : 'PC 제어 권한 설정'}
              </div>
              <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.5)' }}>
                {lang === 'en'
                  ? 'Nexus wants to control your mouse, keyboard, and windows.'
                  : 'Nexus가 마우스·키보드·창을 제어하려고 합니다.'}
              </div>
            </div>
          </div>

          {/* 대기 중 작업 표시 */}
          {pendingAction && (
            <div style={{
              padding: '10px 12px', marginBottom: 16,
              background: 'rgba(245,158,11,0.08)',
              border: '1px solid rgba(245,158,11,0.3)',
              borderLeft: '3px solid #f59e0b',
              borderRadius: 8,
              fontSize: 12, color: 'rgba(255,255,255,0.85)',
            }}>
              <div style={{ fontSize: 10, color: '#fbbf24', fontWeight: 700, marginBottom: 3 }}>
                ⏳ {lang === 'en' ? 'PENDING ACTION' : '대기 중 작업'}
              </div>
              {pendingAction}
            </div>
          )}

          {/* 모드 선택 */}
          <div style={{ display: 'flex', flexDirection: 'column', gap: 8, marginBottom: 16 }}>
            {MODES.map(m => {
              const meta = lang === 'en' ? m.en : m.ko
              return (
                <button
                  key={m.id}
                  onClick={() => handleChoose(m.id)}
                  style={{
                    display: 'flex', alignItems: 'center', gap: 12,
                    padding: '12px 14px',
                    background: `${m.color}11`,
                    border: `1px solid ${m.color}44`,
                    borderRadius: 10,
                    cursor: 'pointer',
                    textAlign: 'left',
                    transition: 'all 0.15s',
                  }}
                  onMouseEnter={e => {
                    e.currentTarget.style.background = `${m.color}22`
                    e.currentTarget.style.borderColor = `${m.color}aa`
                    e.currentTarget.style.transform = 'translateX(2px)'
                  }}
                  onMouseLeave={e => {
                    e.currentTarget.style.background = `${m.color}11`
                    e.currentTarget.style.borderColor = `${m.color}44`
                    e.currentTarget.style.transform = 'translateX(0)'
                  }}
                >
                  <span style={{ fontSize: 22, flexShrink: 0 }}>{m.icon}</span>
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                      <span style={{ fontSize: 13, fontWeight: 700, color: '#fff' }}>{meta.label}</span>
                      {m.recommended && (
                        <span style={{
                          fontSize: 9, fontWeight: 700, color: m.color,
                          background: `${m.color}22`,
                          padding: '1px 6px', borderRadius: 4,
                        }}>
                          {lang === 'en' ? 'RECOMMENDED' : '추천'}
                        </span>
                      )}
                    </div>
                    <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.55)', marginTop: 2 }}>
                      {meta.desc}
                    </div>
                  </div>
                </button>
              )
            })}
          </div>

          {/* 취소 */}
          <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
            <button
              onClick={onCancel}
              style={{
                background: 'transparent',
                border: '1px solid rgba(255,255,255,0.15)',
                color: 'rgba(255,255,255,0.6)',
                padding: '8px 16px', borderRadius: 8,
                fontSize: 11, fontWeight: 600,
                cursor: 'pointer',
              }}
            >
              {lang === 'en' ? 'Cancel This Action' : '이 작업 취소'}
            </button>
          </div>

          {/* 안내 */}
          <div style={{
            marginTop: 16, padding: '8px 12px',
            background: 'rgba(255,255,255,0.03)',
            borderRadius: 6,
            fontSize: 9, color: 'rgba(255,255,255,0.4)', lineHeight: 1.5,
          }}>
            💡 {lang === 'en'
              ? 'You can change this anytime in Settings → Desktop Permission.'
              : '설정 → 데스크톱 권한에서 언제든 변경할 수 있어요. 위험 작업(삭제/결제)은 모드와 무관하게 항상 확인합니다.'}
          </div>
        </motion.div>
      </motion.div>
    </AnimatePresence>
  )
}
