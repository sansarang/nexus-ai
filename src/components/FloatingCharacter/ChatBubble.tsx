import { useEffect, useRef, useState, useCallback } from 'react'
import { motion, AnimatePresence } from 'framer-motion'

const TYPING_MSGS = ['생각하는 중...', '검색하는 중...', '답변 준비 중...', '분석하는 중...']

function TypingBar({ primaryColor, steps }: { primaryColor: string; steps?: string[] }) {
  const msgs = steps && steps.length > 0 ? steps : TYPING_MSGS
  const [idx, setIdx] = useState(0)
  const [sec, setSec] = useState(0)
  useEffect(() => {
    setIdx(0); setSec(0)
    const t = setInterval(() => {
      setIdx(i => Math.min(i + 1, msgs.length - 1))
      setSec(s => s + 2)
    }, 2000)
    return () => clearInterval(t)
  }, [msgs.length])
  return (
    <div style={{
      background: 'rgba(255,255,255,0.07)', border: `1px solid ${primaryColor}33`,
      borderRadius: '4px 14px 14px 14px', padding: '10px 14px',
      display: 'flex', alignItems: 'center', gap: 8, width: 'fit-content',
    }}>
      {[0,1,2].map(i => (
        <div key={i} style={{
          width: 6, height: 6, borderRadius: '50%', background: primaryColor,
          animation: `typingDot 1.2s ease-in-out infinite ${i * 0.2}s`,
        }} />
      ))}
      <span style={{ fontSize: 10, color: `${primaryColor}99`, marginLeft: 2 }}>
        {msgs[idx]}{sec >= 8 ? ` (${sec}s)` : ''}
      </span>
    </div>
  )
}
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

export interface AttachedFile {
  name: string
  mimeType: string
  dataUrl: string   // base64 data URL
  text?: string     // 텍스트 파일인 경우 추출된 내용
  size: number
  fileType: 'image' | 'document' | 'spreadsheet' | 'video' | 'other'
}

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
  return d.toLocaleDateString('ko-KR', { year: 'numeric', month: 'long', day: 'numeric', weekday: 'short' })
}

function formatDateTime(ts: number) {
  const d = new Date(ts)
  return d.toLocaleString('ko-KR', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', hour12: false })
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
function HistoryItem({ entry, primaryColor, onDelete }: { entry: HistoryEntry; primaryColor: string; onDelete: (id: string) => void }) {
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
      position: 'relative',
    }}>
      <div style={{ display: 'flex', alignItems: 'baseline', gap: 6, marginBottom: 3, paddingRight: 20 }}>
        <span style={{ fontSize: 9, color: 'rgba(255,255,255,0.35)', whiteSpace: 'nowrap' }}>
          {formatDateTime(entry.ts)}
        </span>
        <span style={{
          fontSize: 12, color: 'rgba(255,255,255,0.85)', fontWeight: 600,
          background: `${primaryColor}22`, borderRadius: 6,
          padding: '1px 7px', maxWidth: 160, overflow: 'hidden',
          textOverflow: 'ellipsis', whiteSpace: 'nowrap',
        }}>
          {entry.q}
        </span>
      </div>
      {/* 개별 삭제 버튼 */}
      <button
        onClick={() => onDelete(entry.id)}
        title="이 대화 삭제"
        style={{
          position: 'absolute', top: 8, right: 0,
          background: 'none', border: 'none', cursor: 'pointer',
          color: 'rgba(255,255,255,0.2)', fontSize: 11, padding: '1px 3px',
          lineHeight: 1,
          transition: 'color 0.15s',
        }}
        onMouseEnter={e => (e.currentTarget.style.color = '#ef4444')}
        onMouseLeave={e => (e.currentTarget.style.color = 'rgba(255,255,255,0.2)')}
      >✕</button>
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

const FEATURED_ACTIONS = [
  { emoji: '🔐', label: 'PC 해킹 점검', cmd: '내 PC 해킹당했어? 보안 점검해줘' },
  { emoji: '🔬', label: '딥서치', cmd: '양자컴퓨터에 대해 깊게 조사해줘' },
  { emoji: '🗺️', label: '복합 질문', cmd: '오늘 날씨도 알려주고 경주에서 대전 가는 버스 시간표 알려줘' },
  { emoji: '⚖️', label: '비교 분석', cmd: '아이폰 vs 갤럭시 비교해줘' },
  { emoji: '▶️', label: '영상 검색', cmd: '요즘 유튜브에서 핫한 AI 영상 찾아줘' },
]

const FOLLOW_UP_MAP: Record<string, Array<{ label: string; cmd: string }>> = {
  stock:         [{ label: '📰 관련 뉴스', cmd: '관련 뉴스 찾아줘' }, { label: '📊 차트 보기', cmd: '차트 보여줘' }, { label: '🔔 알림 설정', cmd: '가격 알림 설정해줘' }],
  exchange_rate: [{ label: '💱 다른 통화', cmd: '유로 환율도 알려줘' }, { label: '📈 환율 추이', cmd: '최근 환율 변화 알려줘' }],
  web_search:    [{ label: '🔍 더 찾기', cmd: '더 자세히 찾아줘' }, { label: '📄 요약', cmd: '요약해줘' }],
  deep_research: [{ label: '📁 파일 저장', cmd: '파일로 저장해줘' }, { label: '🔍 더 조사', cmd: '더 깊이 조사해줘' }],
  chat:          [{ label: '🔍 검색', cmd: '웹에서 찾아줘' }, { label: '📝 정리', cmd: '핵심만 정리해줘' }],
  file_ops:      [{ label: '📂 결과 열기', cmd: '정리된 폴더 열어줘' }, { label: '↩️ 취소', cmd: '방금 정리 취소해줘' }],
  screen_analyze:[{ label: '📋 텍스트 복사', cmd: '화면 텍스트 복사해줘' }, { label: '🔍 자세히', cmd: '더 자세히 분석해줘' }],
  clipboard_action: [{ label: '📁 저장', cmd: '파일로 저장해줘' }, { label: '🔄 다시', cmd: '다시 처리해줘' }],
  weather:       [{ label: '📅 주간 날씨', cmd: '이번 주 날씨 알려줘' }, { label: '🌍 다른 지역', cmd: '서울 날씨 알려줘' }],
}

interface ChatBubbleProps {
  messages: ChatMessage[]
  typing: boolean
  listening: boolean
  input: string
  onInputChange: (v: string) => void
  onSend: (text: string) => void
  onSendWithFile?: (text: string, file: AttachedFile, extraFiles?: AttachedFile[]) => void | Promise<void>
  onVoiceToggle: () => void
  onRepair?: (ids: string[]) => void
  assistantName: string
  typingSteps?: string[]
  lang: 'ko' | 'en'
  primaryColor: string
  historyVersion?: number
  clarifyPending?: boolean
  clarifyQuestion?: string
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
  onSendWithFile,
  onVoiceToggle,
  onRepair,
  assistantName,
  lang,
  primaryColor,
  historyVersion = 0,
  typingSteps,
}: ChatBubbleProps) {
  const bottomRef = useRef<HTMLDivElement>(null)
  const [history, setHistory] = useState<HistoryEntry[]>(() => loadHistory())

  const handleDeleteOne = useCallback((id: string) => {
    const updated = history.filter(e => e.id !== id)
    localStorage.setItem(HISTORY_KEY, JSON.stringify(updated))
    setHistory(updated)
  }, [history])
  const [attachedFiles, setAttachedFiles] = useState<AttachedFile[]>([])
  const attachedFile = attachedFiles[0] ?? null
  const setAttachedFile = (f: AttachedFile | null) => setAttachedFiles(f ? [f] : [])
  const [fileLoading, setFileLoading] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const detectFileType = (mime: string, name: string): AttachedFile['fileType'] => {
    if (mime.startsWith('image/')) return 'image'
    if (mime.startsWith('video/')) return 'video'
    if (mime.includes('spreadsheet') || mime.includes('excel') || name.endsWith('.xlsx') || name.endsWith('.csv')) return 'spreadsheet'
    if (mime.includes('pdf') || mime.includes('word') || mime.includes('document') ||
        name.endsWith('.pdf') || name.endsWith('.docx') || name.endsWith('.doc') ||
        name.endsWith('.txt') || name.endsWith('.md')) return 'document'
    return 'other'
  }

  const readOneFile = useCallback(async (file: File): Promise<AttachedFile> => {
    const name = file.name
    const fileType = detectFileType(file.type, name)
    let dataUrl = ''
    let text: string | undefined
    try {
      if (fileType === 'image' || fileType === 'video') {
        dataUrl = await new Promise<string>(resolve => {
          const r = new FileReader(); r.onload = e => resolve(e.target?.result as string); r.readAsDataURL(file)
        })
      } else if (fileType === 'spreadsheet' || name.endsWith('.xlsx') || name.endsWith('.xls') || name.endsWith('.csv')) {
        const arrayBuffer = await file.arrayBuffer()
        const XLSX = await import('xlsx')
        const workbook = XLSX.read(arrayBuffer, { type: 'array' })
        const lines: string[] = []
        workbook.SheetNames.forEach(sheetName => {
          const sheet = workbook.Sheets[sheetName]
          const csv = XLSX.utils.sheet_to_csv(sheet)
          if (csv.trim()) lines.push(`[시트: ${sheetName}]\n${csv}`)
        })
        text = lines.join('\n\n').slice(0, 12000)
        dataUrl = `data:application/vnd.ms-excel;base64,`
      } else if (name.endsWith('.txt') || name.endsWith('.md') || name.endsWith('.csv') || name.endsWith('.json') || file.type.includes('text')) {
        text = await new Promise<string>(resolve => {
          const r = new FileReader(); r.onload = e => resolve(e.target?.result as string); r.readAsText(file, 'utf-8')
        })
      } else {
        dataUrl = await new Promise<string>(resolve => {
          const r = new FileReader(); r.onload = e => resolve(e.target?.result as string); r.readAsDataURL(file)
        })
      }
    } catch (err) {
      console.error('파일 읽기 오류:', err)
    }
    return { name, mimeType: file.type, dataUrl, text, size: file.size, fileType }
  }, [])

  const handleFileSelect = useCallback(async (files: FileList | File[]) => {
    setFileLoading(true)
    const arr = Array.from(files).slice(0, 3) // 최대 3개
    const settled = await Promise.allSettled(arr.map(readOneFile))
    const results = settled.filter(r => r.status === 'fulfilled').map(r => (r as PromiseFulfilledResult<AttachedFile>).value)
    setAttachedFiles(prev => {
      const combined = [...prev, ...results].slice(0, 3)
      return combined
    })
    setFileLoading(false)
  }, [readOneFile])

  // 크기 조절 상태
  const [chatSize, setChatSize] = useState({ w: 300, h: 440 })
  const resizingRef = useRef<{ startX: number; startY: number; startW: number; startH: number } | null>(null)

  const startResize = useCallback((e: React.MouseEvent) => {
    e.preventDefault()
    resizingRef.current = { startX: e.clientX, startY: e.clientY, startW: chatSize.w, startH: chatSize.h }
    const onMove = (ev: MouseEvent) => {
      if (!resizingRef.current) return
      const dw = ev.clientX - resizingRef.current.startX
      const dh = resizingRef.current.startY - ev.clientY
      setChatSize({
        w: Math.max(260, Math.min(600, resizingRef.current.startW + dw)),
        h: Math.max(300, Math.min(700, resizingRef.current.startH + dh)),
      })
    }
    const onUp = () => {
      resizingRef.current = null
      window.removeEventListener('mousemove', onMove)
      window.removeEventListener('mouseup', onUp)
    }
    window.addEventListener('mousemove', onMove)
    window.addEventListener('mouseup', onUp)
  }, [chatSize])

  const handleSendAll = useCallback(() => {
    const text = input.trim()
    if (!text && attachedFiles.length === 0) return
    if (attachedFiles.length > 0 && onSendWithFile) {
      const [primary, ...extra] = attachedFiles
      onSendWithFile(text, primary, extra.length > 0 ? extra : undefined)
    } else if (text) {
      onSend(text)
    }
    setAttachedFiles([])
    if (fileInputRef.current) fileInputRef.current.value = ''
  }, [input, attachedFiles, onSend, onSendWithFile])

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
        width: chatSize.w,
        height: chatSize.h,
        background: 'rgba(10,10,20,0.93)',
        border: `1px solid ${primaryColor}44`,
        borderRadius: 18,
        backdropFilter: 'blur(16px)',
        boxShadow: `0 8px 32px rgba(0,0,0,0.5), 0 0 0 1px ${primaryColor}22`,
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden',
        position: 'relative',
      }}
    >
      {/* 크기 조절 핸들 (좌상단 모서리) */}
      <div
        onMouseDown={startResize}
        title="드래그하여 크기 조절"
        style={{
          position: 'absolute', top: 0, left: 0,
          width: 18, height: 18,
          cursor: 'nwse-resize',
          zIndex: 10,
          display: 'flex', alignItems: 'center', justifyContent: 'center',
        }}
      >
        <svg width="10" height="10" viewBox="0 0 10 10" style={{ opacity: 0.3 }}>
          <line x1="2" y1="8" x2="8" y2="2" stroke="white" strokeWidth="1.5" strokeLinecap="round"/>
          <line x1="5" y1="8" x2="8" y2="5" stroke="white" strokeWidth="1.5" strokeLinecap="round"/>
        </svg>
      </div>
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
          <div style={{ padding: '12px 4px' }}>
            <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.25)', marginBottom: 10, textAlign: 'center' }}>
              이런 걸 물어볼 수 있어요
            </div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
              {FEATURED_ACTIONS.map(a => (
                <button
                  key={a.cmd}
                  onClick={() => onSend(a.cmd)}
                  style={{
                    display: 'flex', alignItems: 'center', gap: 8,
                    padding: '8px 12px', borderRadius: 10, cursor: 'pointer',
                    background: 'rgba(255,255,255,0.05)',
                    border: `1px solid ${primaryColor}33`,
                    color: 'rgba(255,255,255,0.8)', fontSize: 12,
                    textAlign: 'left', transition: 'all 0.15s',
                  }}
                  onMouseEnter={e => { e.currentTarget.style.background = `${primaryColor}18`; e.currentTarget.style.borderColor = `${primaryColor}88` }}
                  onMouseLeave={e => { e.currentTarget.style.background = 'rgba(255,255,255,0.05)'; e.currentTarget.style.borderColor = `${primaryColor}33` }}
                >
                  <span style={{ fontSize: 15 }}>{a.emoji}</span>
                  <span>{a.label}</span>
                </button>
              ))}
            </div>
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
              <HistoryItem key={entry.id} entry={entry} primaryColor={primaryColor} onDelete={handleDeleteOne} />
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

        {/* 마지막 응답 후 follow-up 액션 */}
        {!typing && history.length > 0 && (() => {
          const last = history[history.length - 1]
          const lastAction = messages.filter(m => m.role === 'nexus').slice(-1)[0]
          const actionKey = (lastAction as any)?.action ?? ''
          const suggestions = FOLLOW_UP_MAP[actionKey] ?? FOLLOW_UP_MAP['chat']
          return (
            <AnimatePresence>
              <motion.div
                key={last.id + '-followup'}
                initial={{ opacity: 0, y: 4 }}
                animate={{ opacity: 1, y: 0 }}
                style={{ display: 'flex', gap: 5, flexWrap: 'wrap', marginTop: 6, marginBottom: 2 }}
              >
                {suggestions.map(s => (
                  <button
                    key={s.cmd}
                    onClick={() => onSend(s.cmd)}
                    style={{
                      padding: '4px 10px', borderRadius: 12, cursor: 'pointer',
                      background: 'rgba(255,255,255,0.05)',
                      border: `1px solid ${primaryColor}33`,
                      color: `${primaryColor}cc`, fontSize: 10, fontWeight: 600,
                      transition: 'all 0.15s', whiteSpace: 'nowrap',
                    }}
                    onMouseEnter={e => { e.currentTarget.style.background = `${primaryColor}22` }}
                    onMouseLeave={e => { e.currentTarget.style.background = 'rgba(255,255,255,0.05)' }}
                  >{s.label}</button>
                ))}
              </motion.div>
            </AnimatePresence>
          )
        })()}

        {/* 타이핑 인디케이터 */}
        {typing && (
          <motion.div initial={{ opacity: 0, y: 6 }} animate={{ opacity: 1, y: 0 }} style={{ marginTop: 8 }}>
            <TypingBar primaryColor={primaryColor} steps={typingSteps} />
          </motion.div>
        )}
        <div ref={bottomRef} />
      </div>

      {/* 첨부 파일 미리보기 (다중) */}
      {attachedFiles.length > 0 && (
        <div style={{ margin: '0 10px 0', display: 'flex', flexDirection: 'column', gap: 4 }}>
          {attachedFiles.map((af, idx) => (
            <div key={idx} style={{
              padding: '5px 10px',
              background: 'rgba(255,255,255,0.06)',
              border: `1px solid ${primaryColor}44`,
              borderRadius: 10,
              display: 'flex', alignItems: 'center', gap: 8,
            }}>
              {af.fileType === 'image' && af.dataUrl ? (
                <img src={af.dataUrl} alt="preview"
                  style={{ width: 32, height: 32, objectFit: 'cover', borderRadius: 5, flexShrink: 0 }} />
              ) : (
                <span style={{ fontSize: 15 }}>
                  {af.fileType === 'image' ? '🖼️' : af.fileType === 'video' ? '🎬'
                    : af.fileType === 'spreadsheet' ? '📊' : af.fileType === 'document' ? '📄' : '📎'}
                </span>
              )}
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ color: 'rgba(255,255,255,0.9)', fontSize: 10.5, fontWeight: 600,
                  overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                  {af.name}
                </div>
                <div style={{ color: 'rgba(255,255,255,0.4)', fontSize: 9.5 }}>
                  {(af.size / 1024).toFixed(0)}KB
                  {attachedFiles.length >= 2 && idx === 0 && <span style={{ color: primaryColor, marginLeft: 4 }}>· 비교 모드</span>}
                </div>
              </div>
              <button onClick={() => setAttachedFiles(prev => prev.filter((_, i) => i !== idx))}
                style={{ background: 'none', border: 'none', color: 'rgba(255,255,255,0.4)',
                  cursor: 'pointer', fontSize: 13, padding: 2 }}>✕</button>
            </div>
          ))}
          {attachedFiles.length < 3 && (
            <button onClick={() => fileInputRef.current?.click()}
              style={{ fontSize: 10, color: primaryColor, background: 'none', border: `1px dashed ${primaryColor}55`,
                borderRadius: 8, padding: '3px 8px', cursor: 'pointer', alignSelf: 'flex-start' }}>
              + 파일 추가 (최대 3개)
            </button>
          )}
        </div>
      )}

      {/* 입력 바 */}
      <div style={{
        padding: clarifyPending ? '46px 10px 8px' : attachedFiles.length > 0 ? '6px 10px 8px' : '8px 10px',
        borderTop: `1px solid ${clarifyPending ? primaryColor + '44' : primaryColor + '22'}`,
        display: 'flex',
        alignItems: 'center',
        gap: 6,
        position: 'relative',
        transition: 'padding 0.2s',
      }}>
        {/* 숨겨진 파일 인풋 */}
        <input
          ref={fileInputRef}
          type="file"
          multiple
          accept="image/*,video/*,.pdf,.doc,.docx,.txt,.md,.xlsx,.xls,.csv,.pptx"
          style={{ display: 'none' }}
          onChange={e => { if (e.target.files && e.target.files.length > 0) handleFileSelect(e.target.files) }}
        />

        {/* 📎 첨부 버튼 */}
        <button
          onClick={() => fileInputRef.current?.click()}
          disabled={fileLoading}
          title="파일 첨부 (이미지·문서·스프레드시트)"
          style={{
            width: 32, height: 32, borderRadius: '50%', border: 'none',
            background: attachedFiles.length > 0 ? `${primaryColor}44` : 'rgba(255,255,255,0.07)',
            color: attachedFiles.length > 0 ? primaryColor : 'rgba(255,255,255,0.5)',
            fontSize: 15, cursor: 'pointer', flexShrink: 0,
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            transition: 'all 0.2s',
          }}
        >
          {fileLoading ? '⏳' : '📎'}
        </button>

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
            if (e.key === 'Enter' && !e.shiftKey && !e.nativeEvent.isComposing && (input.trim() || attachedFile)) {
              e.preventDefault()
              handleSendAll()
            }
          }}
          placeholder={
            attachedFile
              ? '파일에 대해 질문하거나 Enter로 바로 분석...'
              : clarifyPending
                ? '답변을 입력하거나 마이크로 말씀하세요...'
                : listening
                  ? (lang === 'ko' ? '말씀하세요...' : 'Speak now...')
                  : lang === 'ko' ? `${assistantName}에게...` : `Ask ${assistantName}...`
          }
          style={{
            flex: 1, background: clarifyPending ? `${primaryColor}11` : 'rgba(255,255,255,0.05)',
            border: `1.5px solid ${clarifyPending ? primaryColor : listening ? '#ef4444' : attachedFile ? primaryColor : primaryColor}${clarifyPending || attachedFile ? 'aa' : '44'}`,
            borderRadius: 16, padding: '7px 12px',
            color: 'rgba(255,255,255,0.9)', fontSize: 13, outline: 'none',
            fontFamily: 'Pretendard, Inter, sans-serif',
            transition: 'border-color 0.2s, background 0.2s',
          }}
        />

        <button
          onClick={handleSendAll}
          disabled={!input.trim() && !attachedFile}
          style={{
            width: 32, height: 32, borderRadius: '50%', border: 'none',
            background: (input.trim() || attachedFile) ? primaryColor : `${primaryColor}22`,
            color: '#fff', fontSize: 13, cursor: (input.trim() || attachedFile) ? 'pointer' : 'default',
            flexShrink: 0, display: 'flex', alignItems: 'center', justifyContent: 'center',
            opacity: (input.trim() || attachedFile) ? 1 : 0.4, transition: 'all 0.2s',
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
