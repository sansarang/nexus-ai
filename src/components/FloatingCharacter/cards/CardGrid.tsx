/**
 * CardGrid — Apple 날씨 위젯 그리드 (2×2)
 *
 * 흐름:
 *   - 답변마다 새 카드가 좌상단으로 진입 (slide + fade in)
 *   - 최대 4개 유지 (2×2 그리드)
 *   - 5개째 새 카드 들어오면 가장 오래된 카드 fade out
 *   - 카드 클릭 → ExpandedResultView 확대
 *   - 카드 우상단 ✕ 수동 제거
 *
 * 사용:
 *   <CardGrid
 *     items={cardItems}
 *     onCardClick={openCanvas}
 *     onCardDismiss={removeCard}
 *   />
 *
 * cardItems 는 ChatMessage 의 카드 정보를 추출한 배열.
 */

import { AnimatePresence, motion } from 'framer-motion'
import { useEffect, useRef, useState } from 'react'
import { CardShell } from './CardShell'
import { NEXUS } from '../../../theme/nexus'

export interface CardGridItem {
  id: string                          // 메시지 ID 등 고유 키
  icon?: string
  title: string
  subtitle?: string
  preview: React.ReactNode            // 카드 본문 압축 표시 (1~2줄)
  accent?: 'default' | 'success' | 'warning' | 'critical' | 'brand'
  timestamp?: number                  // 정렬용
}

interface CardGridProps {
  items: CardGridItem[]
  maxCards?: number                   // 기본 4 (2×2)
  onCardClick?: (item: CardGridItem) => void
  onCardDismiss?: (item: CardGridItem) => void
  emptyHint?: React.ReactNode         // 빈 상태 안내
}

export function CardGrid({
  items,
  maxCards = 4,
  onCardClick,
  onCardDismiss,
  emptyHint,
}: CardGridProps) {
  // 반응형: 좁으면 1열 + 적게, 넓으면 2열
  const wrapRef = useRef<HTMLDivElement>(null)
  const [width, setWidth] = useState(360)
  useEffect(() => {
    if (!wrapRef.current) return
    const ro = new ResizeObserver((entries) => {
      for (const e of entries) setWidth(e.contentRect.width)
    })
    ro.observe(wrapRef.current)
    return () => ro.disconnect()
  }, [])
  const isNarrow = width < 320
  const cols = isNarrow ? 1 : 2
  const effectiveMax = isNarrow ? Math.min(2, maxCards) : maxCards

  // 최신순 정렬 + 상한
  const visibleItems = items
    .slice()
    .sort((a, b) => (b.timestamp ?? 0) - (a.timestamp ?? 0))
    .slice(0, effectiveMax)

  // 빈 상태
  if (visibleItems.length === 0 && emptyHint) {
    return (
      <div ref={wrapRef} style={{
        padding: `${NEXUS.spacing.xl}px`,
        textAlign: 'center',
        color: NEXUS.text.tertiary,
        fontSize: NEXUS.font.size.sm,
      }}>
        {emptyHint}
      </div>
    )
  }

  return (
    <div ref={wrapRef} style={{
      display: 'grid',
      gridTemplateColumns: `repeat(${cols}, 1fr)`,
      gap: NEXUS.layout.cardGridGap,
      padding: `${NEXUS.spacing.md}px ${NEXUS.spacing.lg}px`,
      maxHeight: isNarrow ? 260 : 320,
      overflow: 'hidden',
    }}>
      <AnimatePresence mode="popLayout">
        {visibleItems.map((item) => (
          <motion.div
            key={item.id}
            layout
            initial={{ opacity: 0, y: -8, scale: 0.96 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, scale: 0.94, transition: { duration: 0.18 } }}
            transition={NEXUS.motion.spring}
          >
            <CardShell
              icon={item.icon}
              title={item.title}
              subtitle={item.subtitle}
              accent={item.accent}
              variant="compact"
              onClick={onCardClick ? () => onCardClick(item) : undefined}
              onClose={onCardDismiss ? () => onCardDismiss(item) : undefined}
            >
              {item.preview}
            </CardShell>
          </motion.div>
        ))}
      </AnimatePresence>
    </div>
  )
}
