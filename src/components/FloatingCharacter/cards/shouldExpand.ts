/**
 * shouldExpand — 카드/블록 → "Jarvis 캔버스 모드" 자동 라우팅 결정.
 *
 * 12 카테고리 매핑:
 *   Cat 1 (단순 Q&A) → chat
 *   Cat 2 (시스템 모니터링) → chat (게이지는 좁아도 OK)
 *   Cat 3 (보안) → canvas if 5+ issues
 *   Cat 4 (PC 제어) → chat (확인 메시지만)
 *   Cat 5 (파일·문서) → canvas (diff·목록)
 *   Cat 6 (웹 검색) → canvas always
 *   Cat 7 (생산성) → canvas if list 5+
 *   Cat 8 (이메일·캘린더) → canvas if 3+
 *   Cat 9 (Office COM) → canvas (표·차트)
 *   Cat 10 (AI 에이전트) → canvas (단계 진행)
 *   Cat 11 (미디어) → canvas (이미지·오디오)
 *   Cat 12 (Pro 분석) → canvas always
 */

import type { Block } from './DynamicBlocks'
import type { InlineCardData } from '../InlineCards'
import type { InlineCardData2 } from '../InlineCards2'
import type { InlineCard3Data } from '../InlineCards3'
import type { InlineCard4Data } from '../InlineCards4'
import type { InlineCard5Data } from '../InlineCards5'

/** Dynamic Block 배열 → 확장 여부 (기존 로직) */
export function shouldExpand(blocks: Block[] | undefined): boolean {
  if (!blocks || blocks.length === 0) return false
  if (blocks.length >= 10) return true

  for (const b of blocks) {
    switch (b.type) {
      case 'chart': return true
      case 'image': return true
      case 'table':
        if (b.headers && b.headers.length >= 3) return true
        if (b.rows && b.rows.length >= 5) return true
        break
      case 'keyvalue':
        if (b.pairs && b.pairs.length >= 6) return true
        break
      case 'steps':
        if (b.items && b.items.length >= 5) return true
        break
    }
  }
  const fileCount = blocks.filter(b => b.type === 'file').length
  if (fileCount >= 3) return true
  return false
}

/** Card 1 (PC·Scan·Daily·Clean·Repair·Folder) → 확장 여부 */
export function shouldExpandCard1(card: InlineCardData | undefined): boolean {
  if (!card) return false
  switch (card.type) {
    case 'scan_result':
      return (card.data.issues?.length ?? 0) >= 4
    case 'daily_report':
      return true  // 일일 리포트는 차트 + 권장사항 → 확장
    case 'repair_result':
      return false  // 단순 결과
    case 'pc_status':
      return false  // 게이지는 좁아도 OK
    case 'dynamic':
      return shouldExpand(card.blocks)
    default:
      return false
  }
}

/** Card 2 (보안·시스템제어·고급) → 확장 여부 */
export function shouldExpandCard2(card: InlineCardData2 | undefined): boolean {
  if (!card) return false
  switch (card.type) {
    case 'price_compare':
      return (card.data.results?.length ?? 0) >= 3
    case 'process_security':
      return (card.data.suspicious_processes?.length ?? 0) >= 3 || (card.data.open_ports?.length ?? 0) >= 3
    case 'process_top':
      return true  // CPU + MEM 두 컬럼 + 5개씩 → 가로 필요
    case 'network':
      return (card.data.adapters?.length ?? 0) >= 2
    case 'programs_list':
      return true  // 항상 길음
    case 'file_search':
      return (card.data.results?.length ?? 0) >= 4
    case 'duplicates':
      return (card.data.groups?.length ?? 0) >= 3
    case 'email_list':
      return (card.data.emails?.length ?? 0) >= 3
    case 'timeline':
      return (card.data.events?.length ?? 0) >= 3 || (card.data.slots?.length ?? 0) >= 3
    case 'gauge_bar':
      return true
    case 'step_list':
      return (card.data.steps?.length ?? 0) >= 4 || (card.data.workflows?.length ?? 0) >= 4
    case 'item_list':
      return (card.data.items?.length ?? 0) >= 4 || (card.data.results?.length ?? 0) >= 4
    case 'grid_select':
      return true  // 페르소나 그리드 등
    case 'weather_card':
      return false  // 단순 카드 OK
    case 'startup_items':
      return (card.data.items?.length ?? 0) >= 5
    case 'defender':
    case 'remote_access':
    case 'system_action':
    case 'focus_mode':
    case 'notes':
    case 'boot_analysis':
    case 'drivers':
    case 'file_result':
    case 'text_block':
      return false
    default:
      return false
  }
}

/** Card 3 (문서) → 확장 여부 — 거의 다 확장 */
export function shouldExpandCard3(card: InlineCard3Data | undefined): boolean {
  if (!card) return false
  switch (card.type) {
    case 'doc_compare': return true   // diff 화면 필요
    case 'doc_find':    return (card.data.results?.length ?? 0) >= 3
    case 'deep_search': return true   // 항상 긴 결과
    case 'vision_result': return true // 스크린샷 포함
    case 'vision_ocr':  return false  // 텍스트만
    case 'smart_organize': return (card.data.folders?.length ?? 0) >= 4
    default: return false
  }
}

/** Card 4 (매크로·일지·리포트) → 확장 여부 */
export function shouldExpandCard4(card: InlineCard4Data | undefined): boolean {
  if (!card) return false
  switch (card.type) {
    case 'journal_today':   return true  // 종합 일지
    case 'journal_history': return true  // 다일 히스토리
    case 'pc_report':       return true  // 종합 리포트
    case 'doc_summary':     return true  // 긴 텍스트
    case 'macro_list':      return true  // 매크로 그리드
    case 'macro_created':   return false
    case 'macro_run':       return false
    default: return false
  }
}

/** Card 5 (웹 검색) → 항상 확장 */
export function shouldExpandCard5(card: InlineCard5Data | undefined): boolean {
  if (!card) return false
  // 모든 웹 검색은 이미지+링크 → 캔버스
  return true
}

/** 메시지 전체 → 확장 여부 통합 */
export interface CardSlotData {
  inlineCard?:  InlineCardData
  inlineCard2?: InlineCardData2
  inlineCard3?: InlineCard3Data
  inlineCard4?: InlineCard4Data
  inlineCard5?: InlineCard5Data
}
export function shouldExpandMessage(msg: CardSlotData): boolean {
  return shouldExpandCard1(msg.inlineCard)
    || shouldExpandCard2(msg.inlineCard2)
    || shouldExpandCard3(msg.inlineCard3)
    || shouldExpandCard4(msg.inlineCard4)
    || shouldExpandCard5(msg.inlineCard5)
}

/** 확장 모드 제목 추출 */
export function expandTitle(blocks: Block[] | undefined): string {
  if (!blocks) return ''
  const h = blocks.find(b => b.type === 'heading')
  if (h && h.type === 'heading') return h.text
  const t = blocks.find(b => b.type === 'text')
  if (t && t.type === 'text') return t.content.slice(0, 50)
  return ''
}
