/**
 * ExpandedResultView — Dynamic Block 결과를 풍부한 대시보드로 표시.
 *
 * 트리거: shouldExpand() === true (table 5+행, chart, KPI 6+, 등)
 * 동작: 전체 화면 모달로 슬라이드 인 → 충분한 가로/세로 공간 제공
 * 닫기: X 버튼 또는 ESC
 *
 * Jarvis/Claude Artifacts 스타일 — 채팅은 유지하면서 결과만 큰 화면에
 */

import { useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { DynamicCardRenderer, type Block } from './DynamicBlocks'

interface ExpandedResultViewProps {
  open: boolean
  title?: string
  blocks: Block[]
  /** 페르소나 색상 (강조용) */
  accentColor?: string
  onClose: () => void
  /** action 블록 클릭 시 — 명령 재전송 + 모달 닫기 */
  onAction?: (command: string) => void
  lang?: 'ko' | 'en'
}

export function ExpandedResultView({
  open, title, blocks, accentColor = '#9b59b6', onClose, onAction, lang = 'ko',
}: ExpandedResultViewProps) {
  // ESC로 닫기
  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose() }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open, onClose])

  return (
    <AnimatePresence>
      {open && (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          transition={{ duration: 0.2 }}
          style={{
            position: 'fixed', inset: 0, zIndex: 9999,
            background: 'rgba(10,10,20,0.85)',
            backdropFilter: 'blur(8px)',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            padding: 32,
          }}
          onClick={onClose}
        >
          <motion.div
            initial={{ opacity: 0, y: 20, scale: 0.96 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: 20, scale: 0.96 }}
            transition={{ type: 'spring', stiffness: 360, damping: 32 }}
            onClick={e => e.stopPropagation()}
            style={{
              width: '100%', maxWidth: 1100, maxHeight: '92vh',
              background: '#1a1f33',
              border: `1px solid ${accentColor}44`,
              borderRadius: 18,
              boxShadow: `0 20px 80px rgba(0,0,0,0.6), 0 0 0 1px ${accentColor}22`,
              display: 'flex', flexDirection: 'column',
              overflow: 'hidden',
            }}
          >
            {/* 헤더 */}
            <div style={{
              display: 'flex', alignItems: 'center', gap: 12,
              padding: '14px 20px',
              borderBottom: `1px solid ${accentColor}22`,
              background: `linear-gradient(135deg, ${accentColor}11, transparent)`,
            }}>
              <span style={{ fontSize: 18 }}>📊</span>
              <div style={{ flex: 1 }}>
                <div style={{ fontSize: 9, fontWeight: 800, color: `${accentColor}cc`, letterSpacing: '0.06em' }}>
                  {lang === 'en' ? 'RESULT VIEW' : '결과 대시보드'}
                </div>
                <div style={{ fontSize: 15, fontWeight: 700, color: '#f1f5f9', marginTop: 1 }}>
                  {title || (lang === 'en' ? 'Analysis Result' : '분석 결과')}
                </div>
              </div>
              <button
                onClick={onClose}
                title={lang === 'en' ? 'Close (ESC)' : '닫기 (ESC)'}
                style={{
                  background: 'rgba(255,255,255,0.08)',
                  border: '1px solid rgba(255,255,255,0.15)',
                  color: 'rgba(255,255,255,0.7)',
                  width: 32, height: 32, borderRadius: 8,
                  fontSize: 16, cursor: 'pointer',
                  display: 'flex', alignItems: 'center', justifyContent: 'center',
                  transition: 'all 0.15s',
                }}
                onMouseEnter={e => { e.currentTarget.style.background = 'rgba(239,68,68,0.2)'; e.currentTarget.style.borderColor = 'rgba(239,68,68,0.5)'; e.currentTarget.style.color = '#fca5a5' }}
                onMouseLeave={e => { e.currentTarget.style.background = 'rgba(255,255,255,0.08)'; e.currentTarget.style.borderColor = 'rgba(255,255,255,0.15)'; e.currentTarget.style.color = 'rgba(255,255,255,0.7)' }}
              >
                ✕
              </button>
            </div>

            {/* 본문 - 스크롤 가능 */}
            <div style={{
              flex: 1, overflowY: 'auto', padding: '20px 24px',
              scrollbarWidth: 'thin',
              scrollbarColor: `${accentColor}44 transparent`,
            }}>
              <DynamicCardRenderer
                blocks={blocks}
                accentColor={accentColor}
                onAction={(cmd) => {
                  onClose()  // 액션 클릭 시 모달 닫고 명령 전송
                  onAction?.(cmd)
                }}
              />
            </div>

            {/* 푸터 - 도움말 */}
            <div style={{
              padding: '8px 20px',
              borderTop: '1px solid rgba(255,255,255,0.06)',
              fontSize: 10, color: 'rgba(255,255,255,0.4)',
              display: 'flex', justifyContent: 'space-between',
            }}>
              <span>
                {lang === 'en' ? '💡 Tip: Click any action button to continue, ESC to close' : '💡 액션 버튼 클릭으로 이어서 진행, ESC 로 닫기'}
              </span>
              <span style={{ fontFamily: 'ui-monospace, monospace' }}>
                {blocks.length} {lang === 'en' ? 'blocks' : '개 블록'}
              </span>
            </div>
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  )
}
