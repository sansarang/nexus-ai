/**
 * shouldExpand — Dynamic Block 결과가 "확장 뷰" 필요한지 판단.
 *
 * 좁은 채팅 버블엔 부적합한 블록:
 *  - 표 (table) — 3열+ 또는 5행+
 *  - 차트 (chart) — 항상 확장
 *  - keyvalue 6개+ (KPI 대시보드)
 *  - 파일 3개+ (다운로드 리스트)
 *  - steps 5개+ (긴 워크플로)
 *  - image (스크린샷·차트 이미지)
 *  - 블록 총 10개+ (긴 응답)
 */

import type { Block } from './DynamicBlocks'

export function shouldExpand(blocks: Block[] | undefined): boolean {
  if (!blocks || blocks.length === 0) return false
  // 블록 너무 많으면 무조건 확장
  if (blocks.length >= 10) return true

  for (const b of blocks) {
    switch (b.type) {
      case 'chart':
        // 차트는 항상 확장 (좁은 버블에 표시 불가)
        return true
      case 'table':
        // 3열 이상 또는 5행 이상
        if (b.headers && b.headers.length >= 3) return true
        if (b.rows && b.rows.length >= 5) return true
        break
      case 'keyvalue':
        if (b.pairs && b.pairs.length >= 6) return true
        break
      case 'file':
        // 파일 블록 자체가 있으면 다운로드 영역 필요
        // 단일 파일은 OK, 여러 파일은 확장
        // 여기선 단일 파일도 확장하지 않음 (인라인 표시 OK)
        break
      case 'steps':
        if (b.items && b.items.length >= 5) return true
        break
      case 'image':
        // 이미지는 확장 권장
        return true
    }
  }
  // 파일 블록 개수 카운트 (여러 개면 확장)
  const fileCount = blocks.filter(b => b.type === 'file').length
  if (fileCount >= 3) return true

  return false
}

/** 확장 모드 권장 제목 추출 (heading 블록 우선) */
export function expandTitle(blocks: Block[] | undefined): string {
  if (!blocks) return ''
  const h = blocks.find(b => b.type === 'heading')
  if (h && h.type === 'heading') return h.text
  const t = blocks.find(b => b.type === 'text')
  if (t && t.type === 'text') return t.content.slice(0, 50)
  return ''
}
