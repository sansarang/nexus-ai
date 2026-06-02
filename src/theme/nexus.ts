/**
 * nexus.ts — NEXUS 디자인 시스템 (Apple 날씨 영감 + NEXUS 보라/블루 정체성)
 *
 * 사용법:
 *   import { NEXUS } from '@/theme/nexus'
 *   style={{ background: NEXUS.bg.card, border: `1px solid ${NEXUS.border.subtle}` }}
 *
 * 원칙:
 *   - 글래스모피즘: rgba + backdropFilter blur
 *   - 다크 베이스: slate-900 영역 + 월페이퍼 살짝 보이도록
 *   - 강조색: 인디고-바이올렛 그라디언트 (#6366f1 → #8b5cf6 → #a855f7)
 *   - 모든 카드 일관된 셸 (CardShell 컴포넌트)
 */

export const NEXUS = {
  // ── 색상 ──
  brand: {
    indigo:    '#6366f1',
    violet:    '#8b5cf6',
    purple:    '#a855f7',
    // 시그니처 그라디언트 (사용자 메시지, 강조 버튼, 헤더)
    gradient:        'linear-gradient(135deg, #6366f1, #8b5cf6)',
    gradientStrong:  'linear-gradient(135deg, #4f46e5, #7c3aed)',
    gradientSubtle:  'linear-gradient(135deg, rgba(99,102,241,0.18), rgba(168,85,247,0.18))',
  },

  bg: {
    base:     'rgba(15, 23, 42, 0.78)',     // 메인 (월페이퍼 살짝 보임)
    panel:    'rgba(15, 23, 42, 0.92)',     // Sidebar (덜 투명)
    card:     'rgba(255, 255, 255, 0.06)',  // 카드 글래스
    cardHover: 'rgba(255, 255, 255, 0.10)',
    cardActive: 'rgba(255, 255, 255, 0.14)',
    input:    'rgba(255, 255, 255, 0.05)',
    inputFocus: 'rgba(255, 255, 255, 0.09)',
    overlay:  'rgba(0, 0, 0, 0.45)',         // 모달 배경
  },

  border: {
    subtle:  'rgba(255, 255, 255, 0.08)',    // 기본 카드
    medium:  'rgba(255, 255, 255, 0.15)',    // 강조
    accent:  'rgba(139, 92, 246, 0.4)',      // 호버/포커스 (바이올렛)
    glow:    'rgba(139, 92, 246, 0.6)',      // 강조 (네온)
  },

  text: {
    primary:   '#f1f5f9',
    secondary: 'rgba(255, 255, 255, 0.65)',
    tertiary:  'rgba(255, 255, 255, 0.40)',
    quaternary: 'rgba(255, 255, 255, 0.25)',
    accent:    '#a78bfa',                     // 강조 텍스트 (바이올렛 lighter)
    onAccent:  '#ffffff',                     // 그라디언트 위 텍스트
  },

  // ── 글래스 효과 (backdrop-filter) ──
  blur: {
    light:    'blur(20px) saturate(1.4)',
    standard: 'blur(40px) saturate(1.8)',
    heavy:    'blur(60px) saturate(2)',
  },

  // ── 그림자 ──
  shadow: {
    sm:      '0 2px 8px rgba(0, 0, 0, 0.25)',
    md:      '0 4px 16px rgba(0, 0, 0, 0.30)',
    card:    '0 8px 32px rgba(0, 0, 0, 0.35), inset 0 1px 0 rgba(255, 255, 255, 0.06)',
    cardHover: '0 12px 40px rgba(99, 102, 241, 0.25), inset 0 1px 0 rgba(255, 255, 255, 0.10)',
    raised:  '0 20px 60px rgba(0, 0, 0, 0.5)',
    glow:    '0 0 24px rgba(139, 92, 246, 0.35)',
  },

  // ── 반지름 ──
  radius: {
    xs:   6,
    sm:   10,
    md:   14,
    lg:   18,
    xl:   24,
    pill: 999,
  },

  // ── 간격 ──
  spacing: {
    xs:  4,
    sm:  8,
    md:  12,
    lg:  16,
    xl:  24,
    xxl: 32,
  },

  // ── 타이포 ──
  font: {
    size: {
      xs:   10,
      sm:   11.5,
      base: 13,
      md:   14,
      lg:   16,
      xl:   20,
      xxl:  28,
      hero: 36,
    },
    weight: {
      regular: 400,
      medium:  500,
      semi:    600,
      bold:    700,
      heavy:   800,
    },
    family: {
      ui:   "-apple-system, BlinkMacSystemFont, 'SF Pro Display', 'Segoe UI', system-ui, sans-serif",
      mono: "'SF Mono', Consolas, 'Liberation Mono', monospace",
    },
  },

  // ── 애니메이션 ──
  motion: {
    fast:     '0.15s cubic-bezier(0.4, 0, 0.2, 1)',
    standard: '0.25s cubic-bezier(0.4, 0, 0.2, 1)',
    slow:     '0.4s cubic-bezier(0.4, 0, 0.2, 1)',
    spring:   { type: 'spring' as const, stiffness: 340, damping: 28 },
  },

  // ── 레이아웃 상수 ──
  layout: {
    sidebarWidth:     240,
    sidebarMinWidth:  64,    // 접힘 상태 (아이콘만)
    headerHeight:     56,
    inputHeight:      64,
    cardGridGap:      12,
    cardMinHeight:    120,
  },
} as const

// ── 편의 함수 ──
export const glass = (opts?: { strength?: 'light' | 'standard' | 'heavy' }) => ({
  background: NEXUS.bg.card,
  backdropFilter: NEXUS.blur[opts?.strength ?? 'standard'],
  WebkitBackdropFilter: NEXUS.blur[opts?.strength ?? 'standard'],
  border: `1px solid ${NEXUS.border.subtle}`,
  borderRadius: NEXUS.radius.lg,
  boxShadow: NEXUS.shadow.card,
})

// 카드 호버 효과 (CardShell 에서 사용)
export const glassHover = () => ({
  background: NEXUS.bg.cardHover,
  borderColor: NEXUS.border.accent,
  boxShadow: NEXUS.shadow.cardHover,
  transform: 'translateY(-1px)',
})

export type NexusTheme = typeof NEXUS
