/**
 * Card Registry — 메시지의 5종 인라인 카드 슬롯을 단일 진입점으로 통합.
 *
 * 기존엔 ChatBubble.tsx에서 다음 5줄을 두 곳(히스토리 + 라이브)에서 중복했습니다:
 *   {msg.inlineCard  && <InlineCardRenderer  ... />}
 *   {msg.inlineCard2 && <InlineCardRenderer2 ... />}
 *   {msg.inlineCard3 && <InlineCardRenderer3 ... />}
 *   {msg.inlineCard4 && <InlineCardRenderer4 ... />}
 *   {msg.inlineCard5 && <InlineCard5Renderer ... />}
 *
 * 새 카드 타입을 InlineCardData[1-5] 어느 곳에 넣을지는 여전히 분산이지만,
 * 렌더링 호출부는 <CardSlots msg={...} /> 한 줄로 통일됩니다.
 *
 * 향후: 5개 InlineCards 파일을 cards/ 디렉토리 하위로 분류 (cards/system/, cards/web/ 등)
 * 하면서 type 이름이 충돌 안 하는 점을 활용한 단일 discriminated union 으로 통합 가능.
 */

import { InlineCardRenderer,   type InlineCardData }   from '../InlineCards'
import { InlineCardRenderer2,  type InlineCardData2 }  from '../InlineCards2'
import { InlineCardRenderer3,  type InlineCard3Data }  from '../InlineCards3'
import { InlineCardRenderer4,  type InlineCard4Data }  from '../InlineCards4'
import { InlineCard5Renderer,  type InlineCard5Data }  from '../InlineCards5'
import { CardWrapper } from '../CardWrapper'
import { shouldExpandMessage } from './shouldExpand'

export type { InlineCardData, InlineCardData2, InlineCard3Data, InlineCard4Data, InlineCard5Data }

export interface CardCallbacks {
  onRepair?: (ids: string[]) => void
  onMacroRun?: (id: string, name: string) => void
  onPersonaSelect?: (id: string) => void
  /** 에러 카드의 "재시도" 버튼 — 동일 인텐트 재실행 */
  onRetry?: (intent: string) => void
  /** 에러 카드의 "API 키 설정" — Settings 모달 열기 */
  onOpenSettings?: () => void
  /** Dynamic 카드의 action 블록 클릭 — sendText 호출 */
  onAction?: (command: string) => void
}

export interface CardSlotData {
  inlineCard?:  InlineCardData
  inlineCard2?: InlineCardData2
  inlineCard3?: InlineCard3Data
  inlineCard4?: InlineCard4Data
  inlineCard5?: InlineCard5Data
}

interface CardSlotsProps extends CardSlotData, CardCallbacks {
  accentColor: string
  /** true 시 inlineCard/inlineCard2 를 CardWrapper(dark/default) 로 감쌈 — 라이브 채팅용 */
  wrap?: boolean
  /** "캔버스에서 보기" 버튼 표시 (wide card 일 때만) */
  onExpandToCanvas?: () => void
  /** 캔버스 표시 중인지 (이미 떠있으면 버튼 숨김) */
  isCanvasOpen?: boolean
  lang?: 'ko' | 'en'
}

/**
 * 메시지에 들어있는 0~5개의 카드 슬롯을 한 번에 렌더링.
 * 각 카드별 onXxx 콜백은 props로 일괄 전달.
 */
export function CardSlots({
  inlineCard, inlineCard2, inlineCard3, inlineCard4, inlineCard5,
  accentColor, onRepair, onMacroRun, onPersonaSelect, onRetry, onOpenSettings, onAction,
  wrap = false, onExpandToCanvas, isCanvasOpen = false, lang = 'ko',
}: CardSlotsProps) {
  // wide 카드가 있고 캔버스 미오픈 시 "캔버스로 보기" 버튼 표시
  const isWide = shouldExpandMessage({ inlineCard, inlineCard2, inlineCard3, inlineCard4, inlineCard5 })
  const showExpandBtn = isWide && !isCanvasOpen && onExpandToCanvas

  return (
    <>
      {inlineCard && (
        wrap
          ? <CardWrapper variant="dark" accentColor={accentColor} animate={false}>
              <InlineCardRenderer card={inlineCard} accentColor={accentColor} onRepair={onRepair} onRetry={onRetry} onOpenSettings={onOpenSettings} onAction={onAction} />
            </CardWrapper>
          : <InlineCardRenderer card={inlineCard} accentColor={accentColor} onRepair={onRepair} onRetry={onRetry} onOpenSettings={onOpenSettings} onAction={onAction} />
      )}
      {inlineCard2 && (
        wrap
          ? <CardWrapper variant="default" accentColor={accentColor} animate={false}>
              <InlineCardRenderer2 card={inlineCard2} accentColor={accentColor} onPersonaSelect={onPersonaSelect} />
            </CardWrapper>
          : <InlineCardRenderer2 card={inlineCard2} accentColor={accentColor} onPersonaSelect={onPersonaSelect} />
      )}
      {inlineCard3 && <InlineCardRenderer3 card={inlineCard3} />}
      {inlineCard4 && <InlineCardRenderer4 card={inlineCard4} onMacroRun={onMacroRun} />}
      {inlineCard5 && <InlineCard5Renderer card={inlineCard5} accentColor={accentColor} />}

      {/* Jarvis 캔버스 트리거 — wide 카드일 때 "큰 화면으로 보기" 버튼 */}
      {showExpandBtn && (
        <button
          onClick={onExpandToCanvas}
          style={{
            marginTop: 6, padding: '6px 12px',
            background: `${accentColor}22`,
            border: `1px solid ${accentColor}66`,
            borderRadius: 8,
            color: accentColor,
            fontSize: 11, fontWeight: 700,
            cursor: 'pointer',
            display: 'inline-flex', alignItems: 'center', gap: 5,
            alignSelf: 'flex-start',
            transition: 'all 0.15s',
          }}
          onMouseEnter={e => { e.currentTarget.style.background = `${accentColor}44`; e.currentTarget.style.transform = 'translateY(-1px)' }}
          onMouseLeave={e => { e.currentTarget.style.background = `${accentColor}22`; e.currentTarget.style.transform = 'translateY(0)' }}
        >
          🔍 {lang === 'en' ? 'Open in Canvas' : '캔버스에서 크게 보기'}
        </button>
      )}
    </>
  )
}

/**
 * 메시지 객체에서 채워진 카드 슬롯이 하나라도 있는지 확인.
 * — ChatBubble의 `hasCard` 헬퍼와 동일 의미.
 */
export function hasAnyCard(msg: CardSlotData): boolean {
  return !!(msg.inlineCard || msg.inlineCard2 || msg.inlineCard3 || msg.inlineCard4 || msg.inlineCard5)
}
