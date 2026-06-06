/**
 * Sidebar — NEXUS 좌측 패널 (Apple 날씨 스타일 글래스)
 *
 * 구조:
 *  1. 상단: 작은 캐릭터 (36px) + 검색창
 *  2. PRO/사용량 카드 (글래스)
 *  3. 최근 결과 5개 (스크롤)
 *  4. 하단: 음성 입력 + 단축키 안내
 */

import { motion } from 'framer-motion'
import { useMemo, useState } from 'react'
import type { ChatMessage } from './ChatBubble'
import { NEXUS } from '../../theme/nexus'

// v9: 직업/페르소나는 채팅창에서 자유 전환 — Sidebar 드롭다운
export interface PersonaOption {
  id: string
  name: string
  emoji: string
  color: string
}

export const PERSONA_OPTIONS: PersonaOption[] = [
  { id: 'auto',       name: '자동 (스마트)',  emoji: '🪄', color: '#a78bfa' },
  { id: 'nexus',      name: 'Nexus 기본',      emoji: '🤖', color: '#6366f1' },
  { id: 'developer',  name: '개발자',          emoji: '💻', color: '#22c55e' },
  { id: 'marketer',   name: '마케터',          emoji: '📊', color: '#f97316' },
  { id: 'sales',      name: '세일즈',          emoji: '🤝', color: '#06b6d4' },
  { id: 'pm',         name: '프로덕트 매니저', emoji: '📋', color: '#3b82f6' },
  { id: 'designer',   name: '디자이너',        emoji: '🎨', color: '#ec4899' },
  { id: 'meeting',    name: '회의',            emoji: '🎯', color: '#f59e0b' },
  { id: 'research',   name: '리서치',          emoji: '🔬', color: '#0ea5e9' },
  { id: 'creative',   name: '크리에이티브',    emoji: '🎨', color: '#ec4899' },
  { id: 'security',   name: '보안',            emoji: '🛡️', color: '#ef4444' },
  { id: 'legal',      name: '법무',            emoji: '⚖️', color: '#7c3aed' },
  { id: 'medical',    name: '의료',            emoji: '🩺', color: '#dc2626' },
  { id: 'finance',    name: '재무',            emoji: '💰', color: '#10b981' },
  { id: 'investor',   name: '투자자',          emoji: '📈', color: '#16a34a' },
  { id: 'freelancer', name: '프리랜서',        emoji: '🚀', color: '#8b5cf6' },
  { id: 'smallbiz',   name: '소상공인',        emoji: '🏪', color: '#f59e0b' },
  { id: 'corporate',  name: '법인',            emoji: '🏢', color: '#0891b2' },
  { id: 'creator',    name: '크리에이터',      emoji: '🎬', color: '#e11d48' },
  { id: 'tutor',      name: '튜터',            emoji: '📚', color: '#7c3aed' },
]

interface SidebarProps {
  messages: ChatMessage[]
  accentColor: string
  primaryColor: string
  activePersona?: { name: string; emoji: string; color: string } | null
  dailyUsed: number
  dailyLimit: number
  isPro: boolean
  lang: 'ko' | 'en'
  onRecentClick?: (msg: ChatMessage) => void
  onUpgradeClick?: () => void
  onVoiceToggle?: () => void
  voiceActive?: boolean
  onPersonaClick?: () => void                  // 캐릭터 클릭 → 페르소나 변경
  onSearch?: (query: string) => void           // 검색창 → 새 명령 전송
  personaMode?: string                          // v9: 'auto' | personaID
  onPersonaModeChange?: (id: string) => void
}

export function Sidebar({
  messages, accentColor: _accentColor, primaryColor: _primaryColor, activePersona,
  dailyUsed, dailyLimit, isPro, lang,
  onRecentClick, onUpgradeClick, onVoiceToggle, voiceActive,
  onPersonaClick, onSearch, personaMode = 'auto', onPersonaModeChange,
}: SidebarProps) {
  const [query, setQuery] = useState('')
  const [personaOpen, setPersonaOpen] = useState(false)
  const currentPersonaOpt = PERSONA_OPTIONS.find(p => p.id === personaMode) ?? PERSONA_OPTIONS[0]

  // 최근 결과 — query 있으면 전체 메시지에서 필터링, 없으면 카드 있는 최근 5개
  const recentResults = useMemo(() => {
    const hasCard = (m: ChatMessage) =>
      !!(m.inlineCard || m.inlineCard2 || m.inlineCard3 || m.inlineCard4 || m.inlineCard5)
    const trimmedQuery = query.trim().toLowerCase()
    if (trimmedQuery.length > 0) {
      // 실시간 필터: user/nexus 모든 메시지에서 텍스트 검색
      return messages
        .filter(m => (m.text ?? '').toLowerCase().includes(trimmedQuery))
        .slice(-20)
        .reverse()
    }
    return messages
      .filter(m => m.role === 'nexus' && (hasCard(m) || (m.text?.length ?? 0) > 30))
      .slice(-5)
      .reverse()
  }, [messages, query])

  const usagePct = (dailyUsed / dailyLimit) * 100
  const usageColor = usagePct >= 90 ? '#ef4444' : usagePct >= 70 ? '#f59e0b' : NEXUS.text.secondary
  const personaColor = activePersona?.color ?? NEXUS.brand.violet

  // 결과 미리보기
  const previewText = (m: ChatMessage) => {
    if (m.inlineCard?.type === 'dynamic' && m.inlineCard.title) return m.inlineCard.title
    if (m.inlineCard?.type === 'pc_status') return 'PC 상태'
    if (m.inlineCard?.type === 'scan_result') return '보안 스캔'
    if (m.inlineCard?.type === 'daily_report') return '일일 리포트'
    if (m.inlineCard2?.type === 'price_compare') return `가격 비교: ${(m.inlineCard2.data?.query ?? '').slice(0,16)}`
    if (m.inlineCard5?.type === 'web_search') return `웹: ${(m.inlineCard5.query ?? '').slice(0, 18)}`
    if (m.inlineCard5?.type === 'news_search') return `뉴스: ${(m.inlineCard5.query ?? '').slice(0, 18)}`
    if (m.inlineCard5?.type === 'youtube') return `유튜브: ${(m.inlineCard5.query ?? '').slice(0, 18)}`
    return (m.text ?? '').split('\n')[0].slice(0, 24) || '결과'
  }

  const previewIcon = (m: ChatMessage): string => {
    if (m.inlineCard?.type === 'pc_status') return '🖥️'
    if (m.inlineCard?.type === 'scan_result') return '🔒'
    if (m.inlineCard?.type === 'daily_report') return '📊'
    if (m.inlineCard?.type === 'dynamic') return '✨'
    if (m.inlineCard2?.type === 'weather_card') return '🌤'
    if (m.inlineCard2?.type === 'email_list') return '📧'
    if (m.inlineCard2?.type === 'timeline') return '📅'
    if (m.inlineCard2?.type === 'price_compare') return '🛒'
    if (m.inlineCard3?.type === 'doc_compare') return '📑'
    if (m.inlineCard3?.type === 'deep_search') return '🔬'
    if (m.inlineCard4?.type === 'macro_list') return '⚙️'
    if (m.inlineCard5?.type === 'web_search') return '🌐'
    if (m.inlineCard5?.type === 'news_search') return '📰'
    if (m.inlineCard5?.type === 'youtube') return '▶️'
    return '💬'
  }

  return (
    <div data-tauri-drag-region style={{
      width: NEXUS.layout.sidebarWidth,
      flexShrink: 0,
      display: 'flex', flexDirection: 'column',
      background: NEXUS.bg.panel,
      backdropFilter: NEXUS.blur.standard,
      WebkitBackdropFilter: NEXUS.blur.standard,
      borderRight: `1px solid ${NEXUS.border.subtle}`,
      overflow: 'hidden',
    }}>
      {/* ── 상단: 작은 캐릭터 + 검색창 ── */}
      <div style={{
        padding: `${NEXUS.spacing.lg}px ${NEXUS.spacing.lg}px ${NEXUS.spacing.md}px`,
        borderBottom: `1px solid ${NEXUS.border.subtle}`,
        display: 'flex', flexDirection: 'column', gap: NEXUS.spacing.md,
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: NEXUS.spacing.md }}>
          <motion.button
            onClick={onPersonaClick}
            whileHover={{ scale: 1.08 }}
            whileTap={{ scale: 0.94 }}
            animate={{ scale: [1, 1.03, 1] }}
            transition={{ scale: { duration: 3, repeat: Infinity } }}
            style={{
              width: 36, height: 36, borderRadius: '50%', border: 'none',
              background: `radial-gradient(circle at 38% 32%, ${personaColor}ee, ${personaColor}66 70%)`,
              boxShadow: `0 0 16px ${personaColor}55`,
              display: 'flex', alignItems: 'center', justifyContent: 'center',
              fontSize: 16, cursor: 'pointer', flexShrink: 0,
            }}
            title={lang === 'en' ? 'Switch persona' : '페르소나 변경'}
          >
            {activePersona?.emoji ?? '🤖'}
          </motion.button>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{
              fontSize: NEXUS.font.size.base, fontWeight: NEXUS.font.weight.bold,
              color: NEXUS.text.primary, overflow: 'hidden',
              textOverflow: 'ellipsis', whiteSpace: 'nowrap',
            }}>
              {activePersona?.name ?? 'Nexus'}
            </div>
            <div style={{ fontSize: NEXUS.font.size.xs, color: NEXUS.text.tertiary, marginTop: 1 }}>
              {isPro ? (lang === 'en' ? '⭐ PRO active' : '⭐ PRO 활성') : (lang === 'en' ? 'Free plan' : '무료 플랜')}
            </div>
          </div>
        </div>

        {/* 검색창 (글래스) */}
        <div style={{
          background: NEXUS.bg.input,
          border: `1px solid ${NEXUS.border.subtle}`,
          borderRadius: NEXUS.radius.md,
          padding: '6px 10px',
          display: 'flex', alignItems: 'center', gap: 6,
        }}>
          <span style={{ fontSize: 12, opacity: 0.6 }}>🔍</span>
          <input
            value={query}
            onChange={e => setQuery(e.target.value)}
            onKeyDown={e => {
              if (e.key === 'Enter' && query.trim() && onSearch) {
                onSearch(query.trim()); setQuery('')
              }
            }}
            placeholder={lang === 'en' ? 'Ask anything…' : '무엇이든 물어보세요…'}
            style={{
              flex: 1, background: 'transparent', border: 'none', outline: 'none',
              fontSize: NEXUS.font.size.sm, color: NEXUS.text.primary,
              fontFamily: NEXUS.font.family.ui,
            }}
          />
        </div>

        {/* v9: 페르소나 드롭다운 (사용자가 자유 전환 + 자동 매칭) */}
        <div style={{ position: 'relative' }}>
          <button
            onClick={() => setPersonaOpen(o => !o)}
            style={{
              width: '100%',
              padding: '8px 12px',
              background: NEXUS.bg.input,
              border: `1px solid ${NEXUS.border.subtle}`,
              borderRadius: NEXUS.radius.md,
              display: 'flex', alignItems: 'center', justifyContent: 'space-between',
              cursor: 'pointer', color: NEXUS.text.primary,
              fontFamily: NEXUS.font.family.ui, fontSize: NEXUS.font.size.sm,
            }}
            title={lang === 'en' ? 'Switch persona' : '페르소나 전환'}
          >
            <span style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <span style={{ fontSize: 14 }}>{currentPersonaOpt.emoji}</span>
              <span style={{ fontWeight: NEXUS.font.weight.medium }}>{currentPersonaOpt.name}</span>
            </span>
            <span style={{ fontSize: 10, opacity: 0.5 }}>{personaOpen ? '▲' : '▼'}</span>
          </button>
          {personaOpen && (
            <motion.div
              initial={{ opacity: 0, y: -4 }}
              animate={{ opacity: 1, y: 0 }}
              style={{
                position: 'absolute', top: 'calc(100% + 4px)', left: 0, right: 0,
                zIndex: 50,
                background: 'rgba(15,23,42,0.95)',
                backdropFilter: 'blur(40px) saturate(1.6)',
                WebkitBackdropFilter: 'blur(40px) saturate(1.6)',
                border: `1px solid ${NEXUS.border.subtle}`,
                borderRadius: NEXUS.radius.md,
                maxHeight: 320, overflowY: 'auto',
                boxShadow: '0 16px 40px rgba(0,0,0,0.5)',
                padding: 4,
              }}
            >
              {PERSONA_OPTIONS.map(opt => (
                <button
                  key={opt.id}
                  onClick={() => { onPersonaModeChange?.(opt.id); setPersonaOpen(false) }}
                  style={{
                    width: '100%', textAlign: 'left',
                    padding: '8px 10px',
                    borderRadius: NEXUS.radius.sm,
                    background: opt.id === personaMode ? `${opt.color}22` : 'transparent',
                    border: 'none', cursor: 'pointer',
                    display: 'flex', alignItems: 'center', gap: 8,
                    color: NEXUS.text.primary, fontSize: NEXUS.font.size.sm,
                    fontFamily: NEXUS.font.family.ui,
                  }}
                  onMouseEnter={e => { e.currentTarget.style.background = `${opt.color}1a` }}
                  onMouseLeave={e => { e.currentTarget.style.background = opt.id === personaMode ? `${opt.color}22` : 'transparent' }}
                >
                  <span style={{ fontSize: 14 }}>{opt.emoji}</span>
                  <span style={{ flex: 1 }}>{opt.name}</span>
                  {opt.id === personaMode && (
                    <span style={{ fontSize: 11, color: opt.color }}>✓</span>
                  )}
                </button>
              ))}
            </motion.div>
          )}
        </div>
      </div>

      {/* ── 사용량 progress bar ── */}
      <div style={{ padding: `${NEXUS.spacing.md}px ${NEXUS.spacing.lg}px` }}>
        <div style={{
          display: 'flex', justifyContent: 'space-between',
          fontSize: NEXUS.font.size.xs, color: usageColor,
          marginBottom: 4, fontWeight: NEXUS.font.weight.medium,
        }}>
          <span>{lang === 'en' ? 'Today' : '오늘'}</span>
          <span style={{ fontWeight: NEXUS.font.weight.bold }}>{dailyUsed} / {dailyLimit}</span>
        </div>
        <div style={{
          height: 4, background: 'rgba(255,255,255,0.06)',
          borderRadius: NEXUS.radius.pill, overflow: 'hidden',
        }}>
          <motion.div
            initial={{ width: 0 }}
            animate={{ width: `${Math.min(usagePct, 100)}%` }}
            transition={{ duration: 0.6 }}
            style={{
              height: '100%',
              background: usagePct >= 70
                ? `linear-gradient(90deg, ${usageColor}, ${usageColor})`
                : NEXUS.brand.gradient,
            }}
          />
        </div>
        {!isPro && usagePct >= 70 && (
          <button
            onClick={onUpgradeClick}
            style={{
              marginTop: 8, width: '100%',
              padding: '7px 10px', borderRadius: NEXUS.radius.sm,
              background: 'linear-gradient(135deg, #fbbf24, #f59e0b)',
              border: 'none', color: '#1f1f1f',
              fontSize: NEXUS.font.size.xs, fontWeight: NEXUS.font.weight.heavy,
              cursor: 'pointer',
            }}
          >
            ⭐ {lang === 'en' ? 'Upgrade to PRO' : 'PRO 업그레이드'}
          </button>
        )}
      </div>

      {/* ── 최근 결과 ── */}
      <div style={{
        flex: 1, overflowY: 'auto',
        padding: `${NEXUS.spacing.sm}px ${NEXUS.spacing.md}px`,
        scrollbarWidth: 'thin',
      }}>
        <div style={{
          fontSize: NEXUS.font.size.xs, fontWeight: NEXUS.font.weight.bold,
          color: NEXUS.text.tertiary, letterSpacing: '0.08em',
          marginBottom: NEXUS.spacing.sm,
          padding: `0 ${NEXUS.spacing.xs}px`,
          textTransform: 'uppercase',
        }}>
          {query.trim()
            ? `🔎 ${lang === 'en' ? 'Matches' : '검색 결과'} (${recentResults.length})`
            : `📋 ${lang === 'en' ? 'Recent' : '최근 결과'}`}
        </div>
        {recentResults.length === 0 ? (
          <div style={{
            fontSize: NEXUS.font.size.xs, color: NEXUS.text.quaternary,
            textAlign: 'center', padding: '24px 6px', lineHeight: 1.7,
          }}>
            {lang === 'en' ? 'Results appear here\nas you use Nexus' : '결과가 여기에\n쌓여요'}
          </div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
            {recentResults.map((m) => (
              <motion.button
                key={m.id}
                onClick={() => onRecentClick?.(m)}
                whileHover={{ x: 2 }}
                style={{
                  display: 'flex', alignItems: 'center', gap: 8,
                  padding: '8px 10px', borderRadius: NEXUS.radius.sm,
                  background: NEXUS.bg.input,
                  border: `1px solid ${NEXUS.border.subtle}`,
                  cursor: 'pointer', textAlign: 'left',
                  transition: `all ${NEXUS.motion.fast}`,
                }}
                onMouseEnter={e => {
                  e.currentTarget.style.background = NEXUS.bg.cardHover
                  e.currentTarget.style.borderColor = NEXUS.border.accent
                }}
                onMouseLeave={e => {
                  e.currentTarget.style.background = NEXUS.bg.input
                  e.currentTarget.style.borderColor = NEXUS.border.subtle
                }}
              >
                <span style={{ fontSize: 14, flexShrink: 0 }}>{previewIcon(m)}</span>
                <span style={{
                  fontSize: NEXUS.font.size.sm, color: NEXUS.text.primary,
                  overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', flex: 1,
                  fontWeight: NEXUS.font.weight.medium,
                }}>
                  {previewText(m)}
                </span>
              </motion.button>
            ))}
          </div>
        )}
      </div>

      {/* ── 하단: 단축키 ── */}
      <div style={{
        padding: `${NEXUS.spacing.md}px ${NEXUS.spacing.lg}px ${NEXUS.spacing.lg}px`,
        borderTop: `1px solid ${NEXUS.border.subtle}`,
        display: 'flex', flexDirection: 'column', gap: 4,
      }}>
        <div style={{ fontSize: 9, color: NEXUS.text.tertiary, letterSpacing: '0.07em', textTransform: 'uppercase', marginBottom: 4 }}>
          {lang === 'en' ? 'Shortcuts' : '단축키'}
        </div>
        {[
          { key: 'Alt+Space', label: lang === 'en' ? 'Quick launch' : '빠른 호출' },
          { key: 'Alt+S', label: lang === 'en' ? 'Screen capture' : '화면 분석' },
          { key: 'Alt+C', label: lang === 'en' ? 'Clipboard' : '클립보드' },
        ].map(({ key, label }) => (
          <div key={key} style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', fontSize: 10.5, color: NEXUS.text.tertiary }}>
            <span>{label}</span>
            <span style={{ background: NEXUS.bg.input, border: `1px solid ${NEXUS.border.subtle}`, borderRadius: 4, padding: '1px 5px', fontSize: 9.5, color: NEXUS.text.secondary, fontFamily: 'monospace' }}>{key}</span>
          </div>
        ))}
      </div>
    </div>
  )
}
