import { useEffect, useRef, useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { InlineCardRenderer } from './InlineCards'
import type { InlineCardData } from './InlineCards'
import { InlineCardRenderer2 } from './InlineCards2'
import type { InlineCardData2 } from './InlineCards2'
import { InlineCardRenderer3 } from './InlineCards3'
import type { InlineCard3Data } from './InlineCards3'
import { InlineCardRenderer4 } from './InlineCards4'
import type { InlineCard4Data } from './InlineCards4'

interface ChatMessage {
  id: string
  role: 'user' | 'nexus'
  text: string
  inlineCard?: InlineCardData
  inlineCard2?: InlineCardData2
  inlineCard3?: InlineCard3Data
  inlineCard4?: InlineCard4Data
  onMacroRun?: (id: string, name: string) => void
}

export type { ChatMessage }

/* ── 대화 이력 ── */
export interface HistoryEntry {
  id: string
  ts: number
  q: string
  a: string
}

const HISTORY_KEY = 'nexus-chat-history'

export function loadHistory(): HistoryEntry[] {
  try { return JSON.parse(localStorage.getItem(HISTORY_KEY) ?? '[]') }
  catch { return [] }
}

export function appendHistory(entry: HistoryEntry) {
  const all = loadHistory()
  all.push(entry)
  localStorage.setItem(HISTORY_KEY, JSON.stringify(all))
}

function formatTime(ts: number) {
  return new Date(ts).toLocaleTimeString('ko-KR', { hour: '2-digit', minute: '2-digit' })
}
function formatDate(ts: number) {
  const d = new Date(ts)
  const today = new Date()
  const yesterday = new Date(today); yesterday.setDate(today.getDate() - 1)
  if (d.toDateString() === today.toDateString()) return '오늘'
  if (d.toDateString() === yesterday.toDateString()) return '어제'
  return d.toLocaleDateString('ko-KR', { year: 'numeric', month: 'long', day: 'numeric' })
}

function groupByDate(entries: HistoryEntry[]): { date: string; items: HistoryEntry[] }[] {
  const map = new Map<string, HistoryEntry[]>()
  for (const e of entries) {
    const key = formatDate(e.ts)
    if (!map.has(key)) map.set(key, [])
    map.get(key)!.push(e)
  }
  return Array.from(map.entries()).map(([date, items]) => ({ date, items }))
}

/* ── HistoryItem: 질문/답변 행 ── */
function HistoryItem({ entry, primaryColor }: { entry: HistoryEntry; primaryColor: string }) {
  const [expanded, setExpanded] = useState(false)
  const [copied, setCopied] = useState(false)
  const shortA = entry.a.replace(/\*\*/g, '').replace(/\n/g, ' ').slice(0, 40)
  const needsExpand = entry.a.length > 40

  const handleCopy = () => {
    navigator.clipboard.writeText(entry.a.replace(/\*\*/g, '')).then(() => {
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    }).catch(() => {})
  }

  return (
    <div style={{
      borderBottom: '1px solid rgba(255,255,255,0.06)',
      padding: '8px 0',
    }}>
      <div style={{ display: 'flex', alignItems: 'baseline', gap: 6, marginBottom: 3 }}>
        <span style={{ fontSize: 9, color: 'rgba(255,255,255,0.35)', whiteSpace: 'nowrap' }}>
          {formatTime(entry.ts)}
        </span>
        <span style={{
          fontSize: 12, color: 'rgba(255,255,255,0.85)', fontWeight: 600,
          background: `${primaryColor}22`, borderRadius: 6,
          padding: '1px 7px', maxWidth: 200, overflow: 'hidden',
          textOverflow: 'ellipsis', whiteSpace: 'nowrap',
        }}>
          {entry.q}
        </span>
      </div>
      <div
        onClick={() => needsExpand && setExpanded(p => !p)}
        style={{
          fontSize: 11, color: 'rgba(255,255,255,0.55)',
          paddingLeft: 8, lineHeight: 1.5,
          cursor: needsExpand ? 'pointer' : 'default',
          whiteSpace: expanded ? 'pre-wrap' : 'nowrap',
          overflow: expanded ? 'visible' : 'hidden',
          textOverflow: expanded ? 'clip' : 'ellipsis',
        }}
      >
        {expanded ? entry.a.replace(/\*\*/g, '') : shortA + (needsExpand ? '...' : '')}
        {needsExpand && (
          <span style={{ color: primaryColor, marginLeft: 4, fontSize: 10 }}>
            {expanded ? '접기' : '더보기'}
          </span>
        )}
      </div>
      {/* 복사 버튼: 50자 이상일 때만 표시 */}
      {entry.a.length > 50 && (
        <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: 3, paddingRight: 2 }}>
          <button
            onClick={handleCopy}
            style={{
              background: copied ? `${primaryColor}33` : 'none',
              border: `1px solid ${copied ? primaryColor : 'rgba(255,255,255,0.12)'}`,
              borderRadius: 5, color: copied ? primaryColor : 'rgba(255,255,255,0.3)',
              fontSize: 9, fontWeight: 700, padding: '2px 7px', cursor: 'pointer',
              transition: 'all 0.2s',
            }}
          >{copied ? '✓ 복사됨' : '복사'}</button>
        </div>
      )}
    </div>
  )
}

interface ChatBubbleProps {
  messages: ChatMessage[]
  typing: boolean
  listening: boolean
  input: string
  onInputChange: (v: string) => void
  onSend: (text: string) => void
  onVoiceToggle: () => void
  onRepair?: (ids: string[]) => void
  assistantName: string
  lang: 'ko' | 'en'
  primaryColor: string
  historyVersion?: number
  clarifyPending?: boolean   /* clarify 질문 대기 중 */
  clarifyQuestion?: string   /* 현재 clarify 질문 내용 */
}

export function ChatBubble({
  messages,
  typing,
  listening,
  input,
  clarifyPending = false,
  clarifyQuestion = '',
  onInputChange,
  onSend,
  onVoiceToggle,
  onRepair,
  assistantName,
  lang,
  primaryColor,
  historyVersion = 0,
}: ChatBubbleProps) {
  const bottomRef = useRef<HTMLDivElement>(null)
  const [history, setHistory] = useState<HistoryEntry[]>(() => loadHistory())

  /* historyVersion 변경 시 재로드 */
  useEffect(() => {
    setHistory(loadHistory())
  }, [historyVersion])

  /* 최신 메시지로 자동 스크롤 */
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [history, typing])

  const groups = groupByDate(history)
  const scrollRef = useRef<HTMLDivElement>(null)

  /* 카드가 붙은 메시지 — 최근 4개만 live 표시 */
  const liveCards = messages.filter(m => m.inlineCard || m.inlineCard2 || m.inlineCard3 || m.inlineCard4).slice(-2)

  /* 새 카드 / 타이핑 상태 변화 시 자동 스크롤 */
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
  }, [liveCards.length, typing])

  return (
    <motion.div
      initial={{ opacity: 0, y: 20, scale: 0.92 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      exit={{ opacity: 0, y: 16, scale: 0.9 }}
      transition={{ duration: 0.25, ease: [0.4, 0, 0.2, 1] }}
      style={{
        width: 300,
        background: 'rgba(10,10,20,0.93)',
        border: `1px solid ${primaryColor}44`,
        borderRadius: 18,
        backdropFilter: 'blur(16px)',
        boxShadow: `0 8px 32px rgba(0,0,0,0.5), 0 0 0 1px ${primaryColor}22`,
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden',
        maxHeight: 440,
      }}
    >
      {/* 타이틀 */}
      <div style={{
        padding: '10px 14px 8px',
        borderBottom: `1px solid ${primaryColor}22`,
        display: 'flex',
        alignItems: 'center',
        gap: 8,
      }}>
        <div style={{
          width: 8, height: 8, borderRadius: '50%',
          background: primaryColor,
          boxShadow: `0 0 6px ${primaryColor}`,
          animation: 'chatDot 2s ease-in-out infinite',
        }} />
        <span style={{ fontSize: 11, color: primaryColor, fontWeight: 700, letterSpacing: '0.06em', flex: 1 }}>
          대화 이력
        </span>
        {history.length > 0 && (
          <button
            onClick={() => {
              localStorage.removeItem(HISTORY_KEY)
              setHistory([])
            }}
            title="이력 전체 삭제"
            style={{
              background: 'none', border: 'none', cursor: 'pointer',
              color: 'rgba(255,255,255,0.25)', fontSize: 10, padding: '2px 4px',
            }}
          >
            전체삭제
          </button>
        )}
      </div>

      {/* 이력 + 실시간 카드 영역 */}
      <div ref={scrollRef} style={{
        flex: 1,
        overflowY: 'auto',
        padding: '8px 12px',
        display: 'flex',
        flexDirection: 'column',
        scrollbarWidth: 'none',
      }}>
        {history.length === 0 && !typing && (
          <div style={{
            flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center',
            color: 'rgba(255,255,255,0.2)', fontSize: 12, padding: '20px 0',
          }}>
            대화 이력이 없습니다
          </div>
        )}

        {/* 날짜 그룹 */}
        {groups.map(g => (
          <div key={g.date}>
            <div style={{
              fontSize: 10, color: 'rgba(255,255,255,0.3)',
              textAlign: 'center', margin: '6px 0 4px',
              borderBottom: '1px solid rgba(255,255,255,0.07)',
              paddingBottom: 4,
            }}>
              {g.date}
            </div>
            {g.items.map(entry => (
              <HistoryItem key={entry.id} entry={entry} primaryColor={primaryColor} />
            ))}
          </div>
        ))}

        {/* 최근 인라인 카드 (실시간) */}
        <AnimatePresence>
          {liveCards.map(msg => (
            <motion.div
              key={msg.id}
              initial={{ opacity: 0, y: 8 }}
              animate={{ opacity: 1, y: 0 }}
              style={{ marginTop: 8 }}
            >
              {msg.inlineCard && <InlineCardRenderer card={msg.inlineCard} accentColor={primaryColor} onRepair={onRepair} />}
              {msg.inlineCard2 && <InlineCardRenderer2 card={msg.inlineCard2} accentColor={primaryColor} />}
              {msg.inlineCard3 && <InlineCardRenderer3 card={msg.inlineCard3} />}
              {msg.inlineCard4 && <InlineCardRenderer4 card={msg.inlineCard4} onMacroRun={msg.onMacroRun} />}
            </motion.div>
          ))}
        </AnimatePresence>

        {/* 타이핑 인디케이터 */}
        {typing && (
          <motion.div initial={{ opacity: 0, y: 6 }} animate={{ opacity: 1, y: 0 }} style={{ marginTop: 8 }}>
            <div style={{
              background: 'rgba(255,255,255,0.07)',
              border: `1px solid ${primaryColor}33`,
              borderRadius: '4px 14px 14px 14px',
              padding: '10px 14px',
              display: 'flex', gap: 4, width: 'fit-content',
            }}>
              {[0,1,2].map(i => (
                <div key={i} style={{
                  width: 6, height: 6, borderRadius: '50%', background: primaryColor,
                  animation: `typingDot 1.2s ease-in-out infinite ${i * 0.2}s`,
                }} />
              ))}
            </div>
          </motion.div>
        )}
        <div ref={bottomRef} />
      </div>

      {/* 입력 바 */}
      <div style={{
        padding: clarifyPending ? '46px 10px 8px' : '8px 10px',
        borderTop: `1px solid ${clarifyPending ? primaryColor + '44' : primaryColor + '22'}`,
        display: 'flex',
        alignItems: 'center',
        gap: 6,
        position: 'relative',
        transition: 'padding 0.2s',
      }}>
        <button
          onClick={onVoiceToggle}
          style={{
            width: 32, height: 32, borderRadius: '50%', border: 'none',
            background: listening ? '#ef4444' : `${primaryColor}22`,
            color: listening ? '#fff' : primaryColor,
            fontSize: 14, cursor: 'pointer', flexShrink: 0,
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            boxShadow: listening ? '0 0 10px rgba(239,68,68,0.5)' : 'none',
            transition: 'all 0.2s',
          }}
        >
          🎤
        </button>

        {/* clarify 대기 중 안내 배너 */}
        {clarifyPending && (
          <div style={{
            position: 'absolute', top: -38, left: 0, right: 0,
            background: `linear-gradient(135deg, ${primaryColor}33, ${primaryColor}11)`,
            border: `1px solid ${primaryColor}66`,
            borderRadius: 10, padding: '5px 10px',
            fontSize: 10.5, color: primaryColor, fontWeight: 700,
            display: 'flex', alignItems: 'center', gap: 5,
          }}>
            <span>💬</span>
            <span style={{ flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
              {clarifyQuestion || '추가 정보가 필요합니다'}
            </span>
            <span style={{ opacity: 0.6, fontSize: 9 }}>텍스트 또는 음성으로 답해주세요</span>
          </div>
        )}
        <input
          value={input}
          onChange={e => onInputChange(e.target.value)}
          onKeyDown={e => {
            if (e.key === 'Enter' && !e.shiftKey && input.trim()) {
              e.preventDefault()
              onSend(input)
            }
          }}
          placeholder={
            clarifyPending
              ? '답변을 입력하거나 마이크로 말씀하세요...'
              : listening
                ? (lang === 'ko' ? '말씀하세요...' : 'Speak now...')
                : lang === 'ko' ? `${assistantName}에게...` : `Ask ${assistantName}...`
          }
          style={{
            flex: 1, background: clarifyPending ? `${primaryColor}11` : 'rgba(255,255,255,0.05)',
            border: `1.5px solid ${clarifyPending ? primaryColor : listening ? '#ef4444' : primaryColor}${clarifyPending ? 'aa' : '44'}`,
            borderRadius: 16, padding: '7px 12px',
            color: 'rgba(255,255,255,0.9)', fontSize: 13, outline: 'none',
            fontFamily: 'Pretendard, Inter, sans-serif',
            transition: 'border-color 0.2s, background 0.2s',
          }}
        />

        <button
          onClick={() => input.trim() && onSend(input)}
          disabled={!input.trim()}
          style={{
            width: 32, height: 32, borderRadius: '50%', border: 'none',
            background: input.trim() ? primaryColor : `${primaryColor}22`,
            color: '#fff', fontSize: 13, cursor: input.trim() ? 'pointer' : 'default',
            flexShrink: 0, display: 'flex', alignItems: 'center', justifyContent: 'center',
            opacity: input.trim() ? 1 : 0.4, transition: 'all 0.2s',
          }}
        >
          ➤
        </button>
      </div>

      <style>{`
        @keyframes chatDot { 0%,100%{opacity:1} 50%{opacity:0.4} }
        @keyframes typingDot { 0%,60%,100%{transform:translateY(0);opacity:0.5} 30%{transform:translateY(-5px);opacity:1} }
      `}</style>
    </motion.div>
  )
}
