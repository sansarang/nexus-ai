import { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { useAppStore, type ClipboardEntry } from '../../stores/appStore'

type ClipType = 'text' | 'url' | 'code' | 'email' | 'phone' | 'image'

interface ClipInfo {
  type: ClipType
  tags: string[]
  summary: string
}

function classifyContent(content: string): ClipInfo {
  if (/^https?:\/\//.test(content)) {
    return { type: 'url', tags: ['링크'], summary: content.slice(0, 80) }
  }
  if (/\S+@\S+\.\S+/.test(content)) {
    return { type: 'email', tags: ['이메일'], summary: content.slice(0, 80) }
  }
  if (/^[\d\-+() ]{8,}$/.test(content.trim())) {
    return { type: 'phone', tags: ['전화번호'], summary: content.slice(0, 80) }
  }
  if (/function|const |import |def |class |<div/.test(content)) {
    const lang = /function|const |import |<div/.test(content)
      ? 'js'
      : /def |class /.test(content) && !/</.test(content)
        ? 'py'
        : 'html'
    return { type: 'code', tags: ['코드', lang], summary: content.slice(0, 80) }
  }
  if (content.length > 200) {
    return { type: 'text', tags: ['문서'], summary: content.slice(0, 80) + '...' }
  }
  return { type: 'text', tags: [], summary: content.slice(0, 80) }
}

const TYPE_ICONS: Record<ClipType, string> = {
  url: '🔗', code: '💻', email: '📧', phone: '📱', text: '📝', image: '🖼️',
}

const TAG_COLORS: Record<string, string> = {
  '링크': 'rgba(79,126,247,0.15)',
  '코드': 'rgba(168,85,247,0.15)',
  '이메일': 'rgba(34,197,94,0.15)',
  '전화번호': 'rgba(245,158,11,0.15)',
  '문서': 'rgba(99,102,241,0.15)',
  'js': 'rgba(250,204,21,0.12)',
  'py': 'rgba(59,130,246,0.12)',
  'html': 'rgba(239,68,68,0.12)',
}

type FilterTab = '전체' | '링크' | '코드' | '이메일' | '문서' | '📌 고정'
const TABS: FilterTab[] = ['전체', '링크', '코드', '이메일', '문서', '📌 고정']

function timeAgo(date: Date): string {
  const sec = Math.floor((Date.now() - date.getTime()) / 1000)
  if (sec < 60) return '방금 전'
  if (sec < 3600) return `${Math.floor(sec / 60)}분 전`
  if (sec < 86400) return `${Math.floor(sec / 3600)}시간 전`
  return `${Math.floor(sec / 86400)}일 전`
}

function ClipCard({ entry, onCopy, onPin, onDelete }: {
  entry: ClipboardEntry
  onCopy: () => void
  onPin: () => void
  onDelete: () => void
}) {
  const [hovered, setHovered] = useState(false)
  const info = classifyContent(entry.content)

  return (
    <motion.div
      layout
      initial={{ opacity: 0, y: 6 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, height: 0, overflow: 'hidden' }}
      transition={{ layout: { duration: 0.2 } }}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        padding: '10px 14px',
        borderRadius: 'var(--radius-md)',
        background: hovered ? 'var(--bg-elevated)' : 'var(--bg-surface)',
        border: `1px solid ${entry.pinned ? 'rgba(79,126,247,0.3)' : hovered ? 'var(--border-default)' : 'var(--glass-border)'}`,
        cursor: 'pointer',
        transition: 'all 0.12s ease',
        marginBottom: 8,
      }}
    >
      <div style={{ display: 'flex', alignItems: 'flex-start', gap: 10 }}>
        <span style={{ fontSize: 18, flexShrink: 0, marginTop: 1 }}>{TYPE_ICONS[info.type]}</span>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ fontSize: 13, color: info.type === 'url' ? 'var(--accent-primary)' : 'var(--text-primary)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {info.summary}
          </div>
          {/* Tags */}
          {info.tags.length > 0 && (
            <div style={{ display: 'flex', gap: 4, marginTop: 5, flexWrap: 'wrap' }}>
              {info.tags.map((tag) => (
                <span key={tag} style={{
                  padding: '1px 7px', borderRadius: 10,
                  background: TAG_COLORS[tag] ?? 'rgba(255,255,255,0.06)',
                  fontSize: 10, color: 'var(--text-secondary)', fontWeight: 500,
                }}>{tag}</span>
              ))}
            </div>
          )}
          <div style={{ fontSize: 10, color: 'var(--text-muted)', marginTop: 4 }}>{timeAgo(entry.timestamp)}</div>
        </div>

        {/* Actions (visible on hover) */}
        <div style={{ display: 'flex', gap: 4, flexShrink: 0, opacity: hovered ? 1 : 0, transition: 'opacity 0.12s' }}>
          <ActionBtn onClick={(e) => { e.stopPropagation(); onPin() }} title={entry.pinned ? '고정 해제' : '고정'}>
            {entry.pinned ? '📌' : '📍'}
          </ActionBtn>
          <ActionBtn onClick={(e) => { e.stopPropagation(); onCopy() }} title="복사">📋</ActionBtn>
          <ActionBtn onClick={(e) => { e.stopPropagation(); onDelete() }} title="삭제" danger>🗑</ActionBtn>
        </div>
      </div>
    </motion.div>
  )
}

function ActionBtn({ children, onClick, title, danger }: {
  children: React.ReactNode
  onClick: (e: React.MouseEvent) => void
  title?: string
  danger?: boolean
}) {
  return (
    <button
      onClick={onClick}
      title={title}
      style={{
        width: 28, height: 28, borderRadius: 6,
        border: 'none', background: danger ? 'rgba(239,68,68,0.1)' : 'var(--glass-bg)',
        color: danger ? 'var(--danger)' : 'var(--text-secondary)',
        cursor: 'pointer', fontSize: 13,
        display: 'flex', alignItems: 'center', justifyContent: 'center',
      }}
    >{children}</button>
  )
}

export function ClipboardView() {
  const { clipboardHistory, removeClipboard, pinClipboard } = useAppStore()
  const [search, setSearch] = useState('')
  const [activeTab, setActiveTab] = useState<FilterTab>('전체')
  const [copied, setCopied] = useState<string | null>(null)

  const handleCopy = async (entry: ClipboardEntry) => {
    try { await navigator.clipboard.writeText(entry.content) } catch { /**/ }
    setCopied(entry.id)
    setTimeout(() => setCopied(null), 1200)
  }

  const filtered = clipboardHistory.filter((entry) => {
    if (search && !entry.content.toLowerCase().includes(search.toLowerCase())) return false
    if (activeTab === '전체') return true
    if (activeTab === '📌 고정') return entry.pinned
    const info = classifyContent(entry.content)
    if (activeTab === '링크') return info.type === 'url'
    if (activeTab === '코드') return info.type === 'code'
    if (activeTab === '이메일') return info.type === 'email'
    if (activeTab === '문서') return info.tags.includes('문서')
    return true
  })

  return (
    <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
      {/* Search */}
      <div style={{ padding: '12px 16px', borderBottom: '1px solid var(--border-subtle)', flexShrink: 0 }}>
        <div style={{
          display: 'flex', alignItems: 'center', gap: 8, padding: '7px 12px',
          background: 'var(--bg-elevated)', border: '1px solid var(--border-subtle)',
          borderRadius: 'var(--radius-sm)',
        }}>
          <span style={{ fontSize: 14 }}>🔍</span>
          <input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="클립보드 검색..."
            style={{ flex: 1, background: 'none', border: 'none', outline: 'none', color: 'var(--text-primary)', fontSize: 13 }}
          />
          {search && (
            <button onClick={() => setSearch('')} style={{ background: 'none', border: 'none', color: 'var(--text-muted)', cursor: 'pointer', fontSize: 12 }}>✕</button>
          )}
        </div>
      </div>

      {/* Filter tabs */}
      <div style={{ display: 'flex', gap: 4, padding: '8px 12px', borderBottom: '1px solid var(--border-subtle)', flexShrink: 0, overflowX: 'auto' }}>
        {TABS.map((tab) => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            style={{
              padding: '4px 12px', borderRadius: 16, border: 'none', whiteSpace: 'nowrap',
              background: activeTab === tab ? 'rgba(79,126,247,0.15)' : 'transparent',
              color: activeTab === tab ? 'var(--accent-primary)' : 'var(--text-secondary)',
              fontSize: 12, fontWeight: activeTab === tab ? 600 : 400, cursor: 'pointer',
            }}
          >{tab}</button>
        ))}
      </div>

      {/* List */}
      <div style={{ flex: 1, overflowY: 'auto', padding: '10px 12px' }}>
        <AnimatePresence>
          {filtered.map((entry) => (
            <ClipCard
              key={entry.id}
              entry={entry}
              onCopy={() => handleCopy(entry)}
              onPin={() => pinClipboard(entry.id)}
              onDelete={() => removeClipboard(entry.id)}
            />
          ))}
        </AnimatePresence>
        {filtered.length === 0 && (
          <div style={{ textAlign: 'center', padding: 32, color: 'var(--text-muted)', fontSize: 13 }}>
            {search ? `"${search}" 검색 결과 없음` : '클립보드가 비어있어요'}
          </div>
        )}
      </div>

      {/* Copy toast */}
      <AnimatePresence>
        {copied && (
          <motion.div
            initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: 8 }}
            style={{
              position: 'absolute', bottom: 16, left: '50%', transform: 'translateX(-50%)',
              padding: '8px 16px', borderRadius: 20, background: 'var(--bg-elevated)',
              border: '1px solid var(--border-default)', fontSize: 12, color: 'var(--success)',
              boxShadow: 'var(--shadow-md)', pointerEvents: 'none', zIndex: 100,
            }}
          >✓ 클립보드에 복사됨</motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}
