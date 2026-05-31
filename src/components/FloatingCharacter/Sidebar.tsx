/**
 * Sidebar — Jarvis 3-Pane 좌측 패널.
 *
 * 영구 표시:
 *  - 페르소나 (큰 아이콘 + 이름)
 *  - 최근 결과 5개 (클릭으로 재조회 가능)
 *  - 음성 입력 표시
 *  - 단축키 안내
 *  - PRO 업그레이드 / 한도
 */

import { motion } from 'framer-motion'
import { useMemo } from 'react'
import type { ChatMessage } from './ChatBubble'

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
}

export function Sidebar({
  messages, accentColor, primaryColor, activePersona,
  dailyUsed, dailyLimit, isPro, lang,
  onRecentClick, onUpgradeClick, onVoiceToggle, voiceActive,
}: SidebarProps) {
  // 최근 결과 5개 — nexus 메시지 중 카드 있는 것만
  const recentResults = useMemo(() => {
    const hasCard = (m: ChatMessage) =>
      !!(m.inlineCard || m.inlineCard2 || m.inlineCard3 || m.inlineCard4 || m.inlineCard5)
    return messages
      .filter(m => m.role === 'nexus' && (hasCard(m) || m.text?.length > 30))
      .slice(-5)
      .reverse()
  }, [messages])

  const usagePct = (dailyUsed / dailyLimit) * 100
  const usageColor = usagePct >= 90 ? '#ef4444' : usagePct >= 70 ? '#f59e0b' : 'rgba(255,255,255,0.5)'
  const personaColor = activePersona?.color ?? primaryColor

  // 결과 미리보기 텍스트
  const previewText = (m: ChatMessage) => {
    if (m.inlineCard?.type === 'dynamic' && m.inlineCard.title) return m.inlineCard.title
    if (m.inlineCard?.type === 'pc_status') return 'PC 상태'
    if (m.inlineCard?.type === 'scan_result') return '보안 스캔'
    if (m.inlineCard?.type === 'daily_report') return '일일 리포트'
    if (m.inlineCard5?.type === 'web_search') return `웹: ${(m.inlineCard5.query ?? '').slice(0, 18)}`
    if (m.inlineCard5?.type === 'news_search') return `뉴스: ${(m.inlineCard5.query ?? '').slice(0, 18)}`
    if (m.inlineCard5?.type === 'youtube') return `유튜브: ${(m.inlineCard5.query ?? '').slice(0, 18)}`
    return (m.text ?? '').split('\n')[0].slice(0, 24) || '결과'
  }

  // 결과 아이콘
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
    <div style={{
      width: 200, flexShrink: 0,
      display: 'flex', flexDirection: 'column',
      background: 'linear-gradient(180deg, rgba(20,20,42,0.95) 0%, rgba(15,15,30,0.98) 100%)',
      borderRight: `1px solid ${accentColor}22`,
      overflow: 'hidden',
    }}>
      {/* 페르소나 카드 (상단) */}
      <div style={{
        padding: '14px 14px 10px',
        borderBottom: `1px solid ${accentColor}18`,
        background: `linear-gradient(135deg, ${personaColor}18, transparent)`,
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          <motion.div
            animate={{ scale: [1, 1.03, 1] }}
            transition={{ duration: 3, repeat: Infinity }}
            style={{
              width: 36, height: 36, borderRadius: '50%',
              background: `radial-gradient(circle at 38% 32%, ${personaColor}ee, ${personaColor}66 70%)`,
              display: 'flex', alignItems: 'center', justifyContent: 'center',
              fontSize: 18,
              boxShadow: `0 0 12px ${personaColor}55`,
            }}
          >
            {activePersona?.emoji ?? '🤖'}
          </motion.div>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{ fontSize: 11, fontWeight: 800, color: personaColor, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
              {activePersona?.name ?? (lang === 'en' ? 'Nexus' : 'Nexus')}
            </div>
            <div style={{ fontSize: 9, color: 'rgba(255,255,255,0.4)', marginTop: 1 }}>
              {isPro ? (lang === 'en' ? '⭐ PRO active' : '⭐ PRO 활성') : (lang === 'en' ? 'Free plan' : '무료 플랜')}
            </div>
          </div>
        </div>

        {/* 사용 한도 progress bar */}
        <div style={{ marginTop: 10 }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 9, color: usageColor, marginBottom: 3 }}>
            <span>{lang === 'en' ? 'Today' : '오늘'}</span>
            <span style={{ fontWeight: 700 }}>{dailyUsed} / {dailyLimit}</span>
          </div>
          <div style={{ height: 4, background: 'rgba(255,255,255,0.06)', borderRadius: 2, overflow: 'hidden' }}>
            <motion.div
              initial={{ width: 0 }}
              animate={{ width: `${Math.min(usagePct, 100)}%` }}
              transition={{ duration: 0.6 }}
              style={{ height: '100%', background: usageColor }}
            />
          </div>
          {!isPro && usagePct >= 70 && (
            <button
              onClick={onUpgradeClick}
              style={{
                marginTop: 6, width: '100%',
                padding: '5px 8px', borderRadius: 6,
                background: 'linear-gradient(135deg, #fbbf24, #f59e0b)',
                border: 'none', color: '#1f1f1f',
                fontSize: 9.5, fontWeight: 800, cursor: 'pointer',
              }}
            >
              ⭐ {lang === 'en' ? 'Upgrade to PRO' : 'PRO 업그레이드'}
            </button>
          )}
        </div>
      </div>

      {/* 최근 결과 */}
      <div style={{ flex: 1, overflowY: 'auto', padding: '10px 8px', scrollbarWidth: 'thin' }}>
        <div style={{ fontSize: 9, fontWeight: 800, color: `${accentColor}99`, letterSpacing: '0.06em', marginBottom: 6, padding: '0 4px' }}>
          📋 {lang === 'en' ? 'RECENT' : '최근 결과'}
        </div>
        {recentResults.length === 0 ? (
          <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.25)', textAlign: 'center', padding: '20px 6px', lineHeight: 1.6 }}>
            {lang === 'en' ? 'Results appear here as you use Nexus' : '결과가\n여기에 쌓여요'}
          </div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
            {recentResults.map((m) => (
              <button
                key={m.id}
                onClick={() => onRecentClick?.(m)}
                style={{
                  display: 'flex', alignItems: 'center', gap: 7,
                  padding: '7px 8px', borderRadius: 7,
                  background: 'rgba(255,255,255,0.04)',
                  border: `1px solid ${accentColor}11`,
                  cursor: 'pointer', textAlign: 'left',
                  transition: 'all 0.15s',
                }}
                onMouseEnter={e => { e.currentTarget.style.background = `${accentColor}18`; e.currentTarget.style.borderColor = `${accentColor}55` }}
                onMouseLeave={e => { e.currentTarget.style.background = 'rgba(255,255,255,0.04)'; e.currentTarget.style.borderColor = `${accentColor}11` }}
              >
                <span style={{ fontSize: 13, flexShrink: 0 }}>{previewIcon(m)}</span>
                <span style={{ fontSize: 10, color: 'rgba(255,255,255,0.85)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', flex: 1 }}>
                  {previewText(m)}
                </span>
              </button>
            ))}
          </div>
        )}
      </div>

      {/* 음성 + 단축키 (하단) */}
      <div style={{ padding: '10px 12px', borderTop: `1px solid ${accentColor}18` }}>
        {onVoiceToggle && (
          <button
            onClick={onVoiceToggle}
            style={{
              width: '100%', padding: '8px 10px',
              background: voiceActive ? `${accentColor}33` : 'rgba(255,255,255,0.05)',
              border: `1px solid ${voiceActive ? accentColor : 'rgba(255,255,255,0.1)'}`,
              borderRadius: 8, color: voiceActive ? accentColor : 'rgba(255,255,255,0.7)',
              fontSize: 11, fontWeight: 700, cursor: 'pointer',
              display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 5,
            }}
          >
            🎤 {voiceActive
              ? (lang === 'en' ? 'Listening...' : '듣는 중...')
              : (lang === 'en' ? 'Voice input' : '음성 입력')}
          </button>
        )}
        <div style={{ marginTop: 8, fontSize: 9, color: 'rgba(255,255,255,0.3)', lineHeight: 1.5 }}>
          <div>⌨️ <span style={{ color: 'rgba(255,255,255,0.5)' }}>Alt+Space</span> 빠른 호출</div>
          <div>🔊 <span style={{ color: 'rgba(255,255,255,0.5)' }}>"헤이 넥서스"</span> 음성</div>
        </div>
      </div>
    </div>
  )
}
