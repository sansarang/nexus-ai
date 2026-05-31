/**
 * ExpandedResultView — Jarvis 캔버스 모드.
 *
 * 38종 카드 + 11종 블록 모두 대응:
 *  - Dynamic Block (LLM 동적 응답) → DynamicCardRenderer
 *  - InlineCardData (PC·Scan·Daily) → InlineCardRenderer (큰 폭)
 *  - InlineCardData2 (보안·시스템) → InlineCardRenderer2 (큰 폭)
 *  - InlineCard3Data (문서) → InlineCardRenderer3 (큰 폭)
 *  - InlineCard4Data (매크로·일지·리포트) → InlineCardRenderer4
 *  - InlineCard5Data (웹 검색) → InlineCard5Renderer (큰 폭)
 *
 * 자동 트리거: shouldExpand* 함수가 true 인 메시지 도착 시
 * 닫기: X 버튼 또는 ESC
 */

import { useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { DynamicCardRenderer, type Block } from './DynamicBlocks'
import { CardSlots } from './index'
import type { InlineCardData } from '../InlineCards'
import type { InlineCardData2 } from '../InlineCards2'
import type { InlineCard3Data } from '../InlineCards3'
import type { InlineCard4Data } from '../InlineCards4'
import type { InlineCard5Data } from '../InlineCards5'

export interface CanvasContent {
  title?: string
  /** 원본 사용자 질문 (재실행용) */
  originalQuery?: string
  /** Dynamic Block (LLM 응답) */
  blocks?: Block[]
  /** InlineCardData 모든 슬롯 */
  inlineCard?:  InlineCardData
  inlineCard2?: InlineCardData2
  inlineCard3?: InlineCard3Data
  inlineCard4?: InlineCard4Data
  inlineCard5?: InlineCard5Data
}

interface ExpandedResultViewProps {
  open: boolean
  content: CanvasContent | null
  accentColor?: string
  onClose: () => void
  onAction?: (command: string) => void
  /** 마우스 위에 있을 때 자동 닫기 막기 (사장님 검토 중) */
  onRerun?: () => void
  onRepair?: (ids: string[]) => void
  onMacroRun?: (id: string, name: string) => void
  onPersonaSelect?: (id: string) => void
  lang?: 'ko' | 'en'
}

export function ExpandedResultView({
  open, content, accentColor = '#9b59b6', onClose,
  onAction, onRerun, onRepair, onMacroRun, onPersonaSelect, lang = 'ko',
}: ExpandedResultViewProps) {
  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose() }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open, onClose])

  if (!content) return null

  const hasBlocks = content.blocks && content.blocks.length > 0
  const hasCards = !!(content.inlineCard || content.inlineCard2 || content.inlineCard3 || content.inlineCard4 || content.inlineCard5)

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
              <span style={{ fontSize: 22 }}>📊</span>
              <div style={{ flex: 1 }}>
                <div style={{ fontSize: 9, fontWeight: 800, color: `${accentColor}cc`, letterSpacing: '0.06em' }}>
                  {lang === 'en' ? 'CANVAS · RESULT VIEW' : '캔버스 · 결과 대시보드'}
                </div>
                <div style={{ fontSize: 15, fontWeight: 700, color: '#f1f5f9', marginTop: 1 }}>
                  {content.title || (lang === 'en' ? 'Analysis Result' : '분석 결과')}
                </div>
              </div>
              {onRerun && (
                <button
                  onClick={onRerun}
                  title={lang === 'en' ? 'Re-run this command' : '다시 실행'}
                  style={{
                    background: `${accentColor}22`, border: `1px solid ${accentColor}55`,
                    color: accentColor, padding: '6px 12px', borderRadius: 8,
                    fontSize: 11, fontWeight: 700, cursor: 'pointer',
                  }}
                >
                  🔄 {lang === 'en' ? 'Rerun' : '재실행'}
                </button>
              )}
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
                onMouseEnter={e => { e.currentTarget.style.background = 'rgba(239,68,68,0.2)'; e.currentTarget.style.color = '#fca5a5' }}
                onMouseLeave={e => { e.currentTarget.style.background = 'rgba(255,255,255,0.08)'; e.currentTarget.style.color = 'rgba(255,255,255,0.7)' }}
              >
                ✕
              </button>
            </div>

            {/* 본문 - 스크롤 가능 */}
            <div style={{
              flex: 1, overflowY: 'auto', padding: '20px 24px',
              scrollbarWidth: 'thin',
              scrollbarColor: `${accentColor}44 transparent`,
              display: 'flex', flexDirection: 'column', gap: 16,
            }}>
              {/* Dynamic Block 렌더링 */}
              {hasBlocks && content.blocks && (
                <DynamicCardRenderer
                  blocks={content.blocks}
                  accentColor={accentColor}
                  onAction={(cmd) => { onClose(); onAction?.(cmd) }}
                />
              )}
              {/* 38종 카드 렌더링 (5 슬롯 통합) */}
              {hasCards && (
                <CardSlots
                  inlineCard={content.inlineCard}
                  inlineCard2={content.inlineCard2}
                  inlineCard3={content.inlineCard3}
                  inlineCard4={content.inlineCard4}
                  inlineCard5={content.inlineCard5}
                  accentColor={accentColor}
                  onRepair={onRepair}
                  onMacroRun={onMacroRun}
                  onPersonaSelect={onPersonaSelect}
                  onAction={(cmd) => { onClose(); onAction?.(cmd) }}
                  wrap
                />
              )}
            </div>

            {/* 푸터 */}
            <div style={{
              padding: '8px 20px',
              borderTop: '1px solid rgba(255,255,255,0.06)',
              fontSize: 10, color: 'rgba(255,255,255,0.4)',
              display: 'flex', justifyContent: 'space-between',
            }}>
              <span>
                {lang === 'en' ? '💡 Click actions to continue · ESC to close' : '💡 액션 클릭으로 계속 · ESC로 닫기'}
              </span>
              <span style={{ fontFamily: 'ui-monospace, monospace' }}>
                {hasBlocks ? `${content.blocks?.length} blocks` : ''}
                {hasBlocks && hasCards ? ' · ' : ''}
                {hasCards ? 'card' : ''}
              </span>
            </div>
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  )
}
