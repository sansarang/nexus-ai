import { useEffect, useRef, useState, useCallback, useReducer, useMemo } from 'react'
import { motion, AnimatePresence } from 'framer-motion'

function renderMarkdown(text: string): React.ReactNode[] {
  const lines = text.split('\n')
  const nodes: React.ReactNode[] = []
  let codeBlock = false; let codeLines: string[] = []; let codeKey = 0

  const inline = (line: string, k: number): React.ReactNode => {
    const parts: React.ReactNode[] = []; let i = 0; let buf = ''
    while (i < line.length) {
      if (line[i] === '`' && line[i+1] !== '`') {
        const end = line.indexOf('`', i+1)
        if (end !== -1) { if (buf) parts.push(buf); buf = ''; parts.push(<code key={`c${i}`} style={{ background: 'rgba(255,255,255,0.12)', borderRadius: 3, padding: '1px 4px', fontFamily: 'monospace', fontSize: 10.5 }}>{line.slice(i+1, end)}</code>); i = end+1; continue }
      }
      if (line[i] === '*' && line[i+1] === '*') {
        const end = line.indexOf('**', i+2)
        if (end !== -1) { if (buf) parts.push(buf); buf = ''; parts.push(<strong key={`b${i}`}>{line.slice(i+2, end)}</strong>); i = end+2; continue }
      }
      buf += line[i]; i++
    }
    if (buf) parts.push(buf)
    return <span key={k}>{parts}</span>
  }

  lines.forEach((line, idx) => {
    if (line.startsWith('```')) {
      if (!codeBlock) { codeBlock = true; codeLines = [] }
      else { const k = `code-${codeKey++}`; nodes.push(<pre key={k} style={{ background: 'rgba(0,0,0,0.3)', borderRadius: 6, padding: '6px 10px', overflowX: 'auto', fontSize: 10.5, fontFamily: 'monospace', margin: '4px 0', lineHeight: 1.5 }}><code style={{ color: '#e2e8f0' }}>{codeLines.join('\n')}</code></pre>); codeBlock = false }
      return
    }
    if (codeBlock) { codeLines.push(line); return }
    if (!line.trim()) { nodes.push(<div key={idx} style={{ height: 4 }} />); return }
    if (/^#{1,3}\s/.test(line)) {
      const lv = line.match(/^(#+)/)?.[1].length ?? 1
      nodes.push(<div key={idx} style={{ fontWeight: 700, fontSize: [13,12.5,12][Math.min(lv-1,2)], marginTop: 6, marginBottom: 2 }}>{line.replace(/^#+\s/, '')}</div>)
      return
    }
    if (/^[-*]\s/.test(line)) { nodes.push(<div key={idx} style={{ display: 'flex', gap: 5, marginTop: 1 }}><span style={{ opacity: 0.5, flexShrink: 0 }}>•</span><span>{inline(line.replace(/^[-*]\s/, ''), idx)}</span></div>); return }
    if (/^\d+\.\s/.test(line)) { const n = line.match(/^(\d+)/)?.[1]; nodes.push(<div key={idx} style={{ display: 'flex', gap: 5, marginTop: 1 }}><span style={{ opacity: 0.5, flexShrink: 0, minWidth: 14 }}>{n}.</span><span>{inline(line.replace(/^\d+\.\s/, ''), idx)}</span></div>); return }
    if (line.startsWith('---') || line.startsWith('===')) { nodes.push(<hr key={idx} style={{ border: 'none', borderTop: '1px solid rgba(255,255,255,0.1)', margin: '5px 0' }} />); return }
    nodes.push(<div key={idx} style={{ lineHeight: 1.65, marginTop: 1 }}>{inline(line, idx)}</div>)
  })
  return nodes
}

const TYPING_MSGS_KO = ['생각하는 중...', '검색하는 중...', '답변 준비 중...', '분석하는 중...']
const TYPING_MSGS_EN = ['Thinking...', 'Searching...', 'Preparing answer...', 'Analyzing...']

function TypingBar({ primaryColor, steps, lang }: { primaryColor: string; steps?: string[]; lang?: 'ko' | 'en' }) {
  const fallback = lang === 'en' ? TYPING_MSGS_EN : TYPING_MSGS_KO
  const msgs = steps && steps.length > 0 ? steps : fallback
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
import { InlineCard5Renderer } from './InlineCards5'
import type { InlineCard5Data } from './InlineCards5'

interface ChatMessage {
  id: string
  role: 'user' | 'nexus'
  text: string
  inlineCard?: InlineCardData
  inlineCard2?: InlineCardData2
  inlineCard3?: InlineCard3Data
  inlineCard4?: InlineCard4Data
  inlineCard5?: InlineCard5Data
  onMacroRun?: (id: string, name: string) => void
  clarifyOptions?: string[]       // 명확화 질문 선택 버튼
  onClarifySelect?: (option: string) => void  // 버튼 클릭 핸들러
  action?: string                 // follow-up 칩용 액션 키 (web_search, stock 등)
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

/* HistoryItem 제거 — 이력은 HistoryBubbles 형식으로 렌더링 */

const FEATURED_ACTIONS_KO = [
  { emoji: '🔐', label: 'PC 해킹 점검', cmd: '내 PC 해킹당했어? 보안 점검해줘' },
  { emoji: '🔬', label: '딥서치', cmd: '양자컴퓨터에 대해 깊게 조사해줘' },
  { emoji: '🗺️', label: '복합 질문', cmd: '오늘 날씨도 알려주고 경주에서 대전 가는 버스 시간표 알려줘' },
  { emoji: '⚖️', label: '비교 분석', cmd: '아이폰 vs 갤럭시 비교해줘' },
  { emoji: '▶️', label: '영상 검색', cmd: '요즘 유튜브에서 핫한 AI 영상 찾아줘' },
]

const FEATURED_ACTIONS_EN = [
  { emoji: '🔐', label: 'PC Security Scan', cmd: 'Check if my PC has been hacked' },
  { emoji: '🔬', label: 'Deep Research', cmd: 'Do a deep dive on quantum computing' },
  { emoji: '🗺️', label: 'Multi-task Query', cmd: "What's the weather today and find flights from NYC to LA?" },
  { emoji: '⚖️', label: 'Compare & Analyze', cmd: 'Compare iPhone vs Samsung Galaxy' },
  { emoji: '▶️', label: 'Video Search', cmd: 'Find trending AI videos on YouTube right now' },
]

const FOLLOW_UP_MAP_KO: Record<string, Array<{ label: string; cmd: string }>> = {
  stock:            [{ label: '📰 관련 뉴스', cmd: '관련 뉴스 찾아줘' }, { label: '📊 차트 보기', cmd: '차트 보여줘' }, { label: '🔔 알림 설정', cmd: '가격 알림 설정해줘' }],
  exchange_rate:    [{ label: '💱 다른 통화', cmd: '유로 환율도 알려줘' }, { label: '📈 환율 추이', cmd: '최근 환율 변화 알려줘' }],
  web_search:       [{ label: '🔍 더 찾기', cmd: '더 자세히 찾아줘' }, { label: '📄 요약', cmd: '요약해줘' }],
  deep_research:    [{ label: '📁 파일 저장', cmd: '파일로 저장해줘' }, { label: '🔍 더 조사', cmd: '더 깊이 조사해줘' }],
  chat:             [{ label: '🔍 검색', cmd: '웹에서 찾아줘' }, { label: '📝 정리', cmd: '핵심만 정리해줘' }],
  file_ops:         [{ label: '📂 결과 열기', cmd: '정리된 폴더 열어줘' }, { label: '↩️ 취소', cmd: '방금 정리 취소해줘' }],
  screen_analyze:   [{ label: '📋 텍스트 복사', cmd: '화면 텍스트 복사해줘' }, { label: '🔍 자세히', cmd: '더 자세히 분석해줘' }],
  clipboard_action: [{ label: '📁 저장', cmd: '파일로 저장해줘' }, { label: '🔄 다시', cmd: '다시 처리해줘' }],
  weather:          [{ label: '📅 주간 날씨', cmd: '이번 주 날씨 알려줘' }, { label: '🌍 다른 지역', cmd: '서울 날씨 알려줘' }],
}

const FOLLOW_UP_MAP_EN: Record<string, Array<{ label: string; cmd: string }>> = {
  stock:            [{ label: '📰 Related News', cmd: 'Find related news' }, { label: '📊 View Chart', cmd: 'Show me the chart' }, { label: '🔔 Set Alert', cmd: 'Set a price alert' }],
  exchange_rate:    [{ label: '💱 Other Currency', cmd: 'What about Euro exchange rate?' }, { label: '📈 Rate Trend', cmd: 'Show recent exchange rate changes' }],
  web_search:       [{ label: '🔍 Find More', cmd: 'Search for more details' }, { label: '📄 Summarize', cmd: 'Summarize this' }],
  deep_research:    [{ label: '📁 Save File', cmd: 'Save this to a file' }, { label: '🔍 Dig Deeper', cmd: 'Research this more deeply' }],
  chat:             [{ label: '🔍 Search', cmd: 'Search the web for this' }, { label: '📝 Summarize', cmd: 'Summarize the key points' }],
  file_ops:         [{ label: '📂 Open Folder', cmd: 'Open the organized folder' }, { label: '↩️ Undo', cmd: 'Undo the last action' }],
  screen_analyze:   [{ label: '📋 Copy Text', cmd: 'Copy the text from screen' }, { label: '🔍 More Detail', cmd: 'Analyze this in more detail' }],
  clipboard_action: [{ label: '📁 Save', cmd: 'Save to a file' }, { label: '🔄 Redo', cmd: 'Process this again' }],
  weather:          [{ label: '📅 Weekly Forecast', cmd: 'Show me this week\'s weather' }, { label: '🌍 Other City', cmd: 'What\'s the weather in New York?' }],
}

interface ChatBubbleProps {
  messages: ChatMessage[]
  typing: boolean
  input: string
  onInputChange: (v: string) => void
  onSend: (text: string) => void
  onSendWithFile?: (text: string, file: AttachedFile, extraFiles?: AttachedFile[]) => void | Promise<void>
  onRepair?: (ids: string[]) => void
  assistantName: string
  typingSteps?: string[]
  lang: 'ko' | 'en'
  primaryColor: string
  historyVersion?: number
  clarifyPending?: boolean
  clarifyQuestion?: string
  // 페르소나 칩 + 사용량 배지
  activePersona?: { name: string; emoji: string; color: string } | null
  subscriptionStatus?: string
  dailyUsed?: number
  onPersonaClick?: () => void
  onPersonaSelect?: (id: string) => void
  embedded?: boolean
}

export function ChatBubble({
  messages,
  typing,
  input,
  clarifyPending = false,
  clarifyQuestion = '',
  onInputChange,
  onSend,
  onSendWithFile,
  onRepair,
  assistantName,
  lang,
  primaryColor,
  historyVersion = 0,
  typingSteps,
  activePersona,
  subscriptionStatus,
  dailyUsed = 0,
  onPersonaClick,
  onPersonaSelect,
  embedded = false,
}: ChatBubbleProps) {
  const bottomRef = useRef<HTMLDivElement>(null)

  // ── Issue #6: useReducer로 history 관리 → handleDeleteOne 재생성 없음 ──
  const [history, dispatchHistory] = useReducer(
    (state: HistoryEntry[], action: { type: 'set'; payload: HistoryEntry[] } | { type: 'delete'; id: string }) => {
      if (action.type === 'delete') {
        const updated = state.filter(e => e.id !== action.id)
        localStorage.setItem(HISTORY_KEY, JSON.stringify(updated))
        return updated
      }
      return action.payload
    },
    undefined,
    () => loadHistory(),
  )

  const handleDeleteOne = useCallback((id: string) => {
    dispatchHistory({ type: 'delete', id })
  }, [])
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

  // ── Issue #7: cleanup ref로 언마운트 시 리스너 누수 방어 ──
  const resizeCleanupRef = useRef<(() => void) | null>(null)
  useEffect(() => () => { resizeCleanupRef.current?.() }, [])

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
      resizeCleanupRef.current = null
    }
    window.addEventListener('mousemove', onMove)
    window.addEventListener('mouseup', onUp)
    resizeCleanupRef.current = () => {
      window.removeEventListener('mousemove', onMove)
      window.removeEventListener('mouseup', onUp)
    }
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
    dispatchHistory({ type: 'set', payload: loadHistory() })
  }, [historyVersion])

  /* 최신 메시지로 자동 스크롤 */
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [history, typing])

  const groups = groupByDate(history)
  const scrollRef = useRef<HTMLDivElement>(null)
  const isEn = lang === 'en'

  // ── Issue #3/4: lang에 따라 액션 맵 선택 ──
  const FEATURED_ACTIONS = isEn ? FEATURED_ACTIONS_EN : FEATURED_ACTIONS_KO
  const FOLLOW_UP_MAP = isEn ? FOLLOW_UP_MAP_EN : FOLLOW_UP_MAP_KO

  /* 카드가 붙은 메시지 — 최근 6개 표시 */
  const liveCards = useMemo(
    () => messages.filter(m => m.inlineCard || m.inlineCard2 || m.inlineCard3 || m.inlineCard4 || m.inlineCard5).slice(-6),
    [messages]
  )

  // 실시간 대화 메시지 — 최근 20개
  const liveMessages = useMemo(() => messages.slice(-20), [messages])

  // 긴 메시지 펼치기 상태
  const [expandedMsgs, setExpandedMsgs] = useState<Set<string>>(new Set())
  const [showUsagePopup, setShowUsagePopup] = useState(false)
  const toggleExpand = useCallback((id: string) => {
    setExpandedMsgs(prev => {
      const next = new Set(prev)
      next.has(id) ? next.delete(id) : next.add(id)
      return next
    })
  }, [])

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
      style={embedded ? {
        width: '100%', height: '100%',
        background: 'transparent',
        border: 'none', borderRadius: 0, boxShadow: 'none',
        display: 'flex', flexDirection: 'column',
        overflow: 'hidden', position: 'relative',
      } : {
        width: chatSize.w,
        height: chatSize.h,
        background: '#0a0a14',
        border: `1px solid ${primaryColor}55`,
        borderRadius: 18,
        boxShadow: `0 16px 48px rgba(0,0,0,0.85), 0 0 0 1px ${primaryColor}22`,
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
        padding: '11px 14px 9px',
        borderBottom: `1px solid ${primaryColor}33`,
        background: `${primaryColor}0d`,
        display: 'flex',
        alignItems: 'center',
        gap: 8,
        flexShrink: 0,
      }}>
        <div style={{
          width: 9, height: 9, borderRadius: '50%',
          background: primaryColor,
          boxShadow: `0 0 8px ${primaryColor}, 0 0 16px ${primaryColor}55`,
          animation: 'chatDot 2s ease-in-out infinite',
          flexShrink: 0,
        }} />
        <span style={{ fontSize: 12, color: 'rgba(255,255,255,0.92)', fontWeight: 700, letterSpacing: '0.04em', flex: 1 }}>
          {isEn ? 'Chat History' : '대화 이력'}
        </span>
        <span style={{ fontSize: 10, color: 'rgba(255,255,255,0.28)', fontWeight: 500 }}>
          {history.length > 0 ? `${history.length}개` : ''}
        </span>
        {history.length > 0 && (
          <button
            onClick={() => {
              if (window.confirm(isEn ? 'Delete all chat history?' : '대화 이력을 전부 삭제할까요?')) {
                localStorage.removeItem(HISTORY_KEY)
                dispatchHistory({ type: 'set', payload: [] })
              }
            }}
            title={isEn ? 'Clear all history' : '이력 전체 삭제'}
            style={{
              background: 'rgba(239,68,68,0.12)',
              border: '1px solid rgba(239,68,68,0.35)',
              borderRadius: 6,
              cursor: 'pointer',
              color: '#f87171',
              fontSize: 10,
              fontWeight: 700,
              padding: '3px 8px',
              transition: 'all 0.15s',
              marginLeft: 2,
            }}
            onMouseEnter={e => {
              e.currentTarget.style.background = 'rgba(239,68,68,0.25)'
              e.currentTarget.style.borderColor = 'rgba(239,68,68,0.6)'
            }}
            onMouseLeave={e => {
              e.currentTarget.style.background = 'rgba(239,68,68,0.12)'
              e.currentTarget.style.borderColor = 'rgba(239,68,68,0.35)'
            }}
          >
            {isEn ? 'Clear all' : '전체삭제'}
          </button>
        )}
      </div>

      {/* ── 페르소나 칩 + 사용량 배지 (embedded 모드에선 상단 헤더에서 이미 표시) ── */}
      {!embedded && (activePersona || subscriptionStatus) && (
        <div style={{
          padding: '6px 14px',
          borderBottom: `1px solid ${primaryColor}22`,
          display: 'flex', alignItems: 'center', justifyContent: 'space-between',
          flexShrink: 0,
          background: 'rgba(0,0,0,0.2)',
        }}>
          {/* 페르소나 칩 */}
          {activePersona ? (
            <button
              onClick={onPersonaClick}
              style={{
                display: 'flex', alignItems: 'center', gap: 4,
                padding: '3px 8px', borderRadius: 20,
                border: `1px solid ${activePersona.color}55`,
                background: `${activePersona.color}18`,
                color: activePersona.color,
                fontSize: 11, fontWeight: 600,
                cursor: onPersonaClick ? 'pointer' : 'default',
                transition: 'all 0.15s',
              }}
              title={isEn ? 'Change AI mode' : 'AI 모드 변경'}
            >
              <span style={{ fontSize: 13 }}>{activePersona.emoji}</span>
              <span>{activePersona.name}</span>
            </button>
          ) : <div />}

          {/* 사용량 배지 — 클릭 시 상세 팝업 */}
          {(() => {
            const DAILY_LIMIT = 15
            const MONTHLY_LIMIT = 2000
            const isFree = !subscriptionStatus || subscriptionStatus === 'none' || subscriptionStatus === 'expired'
            if (isFree) {
              const remaining = Math.max(0, DAILY_LIMIT - dailyUsed)
              const color = remaining <= 3 ? '#ef4444' : remaining <= 7 ? '#f59e0b' : '#22c55e'
              return (
                <button
                  onClick={() => setShowUsagePopup(v => !v)}
                  title={isEn ? 'Click to view usage details' : '클릭하면 상세 사용량 확인'}
                  style={{ display: 'flex', alignItems: 'center', gap: 3, fontSize: 11, background: 'none', border: 'none', cursor: 'pointer', padding: '2px 4px', borderRadius: 6 }}
                >
                  <span style={{ color: 'rgba(255,255,255,0.4)' }}>{isEn ? 'Today' : '오늘'}</span>
                  <span style={{ color, fontWeight: 700 }}>{remaining}</span>
                  <span style={{ color: 'rgba(255,255,255,0.4)' }}>/{DAILY_LIMIT}{isEn ? ' left' : '회'}</span>
                  <span style={{ color: 'rgba(255,255,255,0.25)', fontSize: 9 }}>▾</span>
                </button>
              )
            }
            const pct = Math.min(100, Math.round((dailyUsed / MONTHLY_LIMIT) * 100))
            const color = pct >= 90 ? '#ef4444' : pct >= 70 ? '#f59e0b' : primaryColor
            return (
              <button
                onClick={() => setShowUsagePopup(v => !v)}
                title={isEn ? 'Click to view usage details' : '클릭하면 상세 사용량 확인'}
                style={{ display: 'flex', alignItems: 'center', gap: 3, fontSize: 11, background: 'none', border: 'none', cursor: 'pointer', padding: '2px 4px', borderRadius: 6 }}
              >
                <span style={{ color: 'rgba(255,255,255,0.4)' }}>{isEn ? 'This month' : '이번달'}</span>
                <span style={{ color, fontWeight: 700 }}>{dailyUsed.toLocaleString()}</span>
                <span style={{ color: 'rgba(255,255,255,0.4)' }}>/{MONTHLY_LIMIT.toLocaleString()}</span>
                <span style={{ color: 'rgba(255,255,255,0.25)', fontSize: 9 }}>▾</span>
              </button>
            )
          })()}
        </div>
      )}

      {/* 사용량 상세 팝업 */}
      {showUsagePopup && (() => {
        const DAILY_LIMIT = 15
        const MONTHLY_LIMIT = 2000
        const isFree = !subscriptionStatus || subscriptionStatus === 'none' || subscriptionStatus === 'expired'
        const usedToday = isFree ? dailyUsed : 0
        const remaining = Math.max(0, DAILY_LIMIT - usedToday)
        const pct = isFree
          ? Math.min(100, Math.round((usedToday / DAILY_LIMIT) * 100))
          : Math.min(100, Math.round((dailyUsed / MONTHLY_LIMIT) * 100))
        const barColor = pct >= 90 ? '#ef4444' : pct >= 70 ? '#f59e0b' : '#22c55e'
        return (
          <div style={{
            margin: '0 10px 8px',
            background: 'rgba(0,0,0,0.6)',
            border: `1px solid ${primaryColor}33`,
            borderRadius: 12,
            padding: '12px 14px',
            backdropFilter: 'blur(12px)',
            flexShrink: 0,
          }}>
            <div style={{ fontSize: 11, fontWeight: 700, color: primaryColor, marginBottom: 10 }}>
              📊 {isEn ? 'Usage Details' : '사용량 상세'}
            </div>
            {isFree ? (
              <>
                <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 10, color: 'rgba(255,255,255,0.6)', marginBottom: 4 }}>
                  <span>{isEn ? 'Daily limit (free)' : '오늘 무료 한도'}</span>
                  <span style={{ color: barColor, fontWeight: 700 }}>{usedToday} / {DAILY_LIMIT}{isEn ? ' used' : '회 사용'}</span>
                </div>
                <div style={{ height: 6, borderRadius: 3, background: 'rgba(255,255,255,0.1)', overflow: 'hidden', marginBottom: 8 }}>
                  <div style={{ height: '100%', width: `${pct}%`, background: barColor, borderRadius: 3, transition: 'width 0.4s' }} />
                </div>
                <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.4)', marginBottom: 4 }}>
                  {isEn ? `${remaining} requests remaining today` : `오늘 ${remaining}회 남음 · 자정에 초기화`}
                </div>
                <div style={{ fontSize: 10, color: primaryColor, fontWeight: 600 }}>
                  {isEn ? '✨ Upgrade for 2,000/month' : '✨ 프리미엄 업그레이드 시 월 2,000회'}
                </div>
              </>
            ) : (
              <>
                <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 10, color: 'rgba(255,255,255,0.6)', marginBottom: 4 }}>
                  <span>{isEn ? 'Monthly (premium)' : '이번달 (프리미엄)'}</span>
                  <span style={{ color: barColor, fontWeight: 700 }}>{dailyUsed.toLocaleString()} / {MONTHLY_LIMIT.toLocaleString()}</span>
                </div>
                <div style={{ height: 6, borderRadius: 3, background: 'rgba(255,255,255,0.1)', overflow: 'hidden', marginBottom: 8 }}>
                  <div style={{ height: '100%', width: `${pct}%`, background: barColor, borderRadius: 3, transition: 'width 0.4s' }} />
                </div>
                <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.4)' }}>
                  {isEn ? `${(MONTHLY_LIMIT - dailyUsed).toLocaleString()} remaining this month` : `이번달 ${(MONTHLY_LIMIT - dailyUsed).toLocaleString()}회 남음`}
                </div>
              </>
            )}
          </div>
        )
      })()}

      {/* 이력 + 실시간 카드 영역 */}
      <div ref={scrollRef} style={{
        flex: 1,
        overflowY: 'auto',
        padding: '8px 12px',
        display: 'flex',
        flexDirection: 'column',
        scrollbarWidth: 'none',
      }}>
        {history.length === 0 && liveMessages.length === 0 && !typing && (
          <div style={{ padding: '12px 4px' }}>
            <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.25)', marginBottom: 10, textAlign: 'center' }}>
              {isEn ? 'You can ask things like...' : '이런 걸 물어볼 수 있어요'}
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

        {/* 이전 대화 이력 (날짜 그룹 + 버블 형식) */}
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
              <div key={entry.id} style={{ marginBottom: 10 }}>
                {/* 사용자 질문 버블 */}
                <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 4 }}>
                  <div style={{
                    maxWidth: '86%', padding: '8px 12px',
                    borderRadius: '14px 14px 4px 14px',
                    background: `${primaryColor}55`,
                    border: `1px solid ${primaryColor}88`,
                    fontSize: 12, color: 'rgba(255,255,255,0.95)',
                    lineHeight: 1.6, wordBreak: 'break-word',
                  }}>
                    {entry.q}
                  </div>
                </div>
                {/* AI 응답 버블 */}
                <div style={{ display: 'flex', justifyContent: 'flex-start', position: 'relative' }}>
                  <div style={{
                    maxWidth: '86%', padding: '8px 12px',
                    borderRadius: '4px 14px 14px 14px',
                    background: 'rgba(255,255,255,0.09)',
                    border: '1px solid rgba(255,255,255,0.13)',
                    fontSize: 12, color: 'rgba(255,255,255,0.9)',
                    lineHeight: 1.65, wordBreak: 'break-word',
                  }}>
                    {renderMarkdown(entry.a)}
                  </div>
                  <button
                    onClick={() => handleDeleteOne(entry.id)}
                    title="이 대화 삭제"
                    style={{
                      background: 'none', border: 'none', cursor: 'pointer',
                      color: 'rgba(255,255,255,0.2)', fontSize: 10, padding: '0 4px',
                      alignSelf: 'flex-start', marginTop: 4, transition: 'color 0.15s',
                    }}
                    onMouseEnter={e => (e.currentTarget.style.color = '#ef4444')}
                    onMouseLeave={e => (e.currentTarget.style.color = 'rgba(255,255,255,0.2)')}
                  >✕</button>
                </div>
                <div style={{ fontSize: 9, color: 'rgba(255,255,255,0.25)', textAlign: 'right', marginTop: 2, paddingRight: 22 }}>
                  {formatDateTime(entry.ts)}
                </div>
              </div>
            ))}
          </div>
        ))}

        {/* ── Issue #1: 실시간 대화 메시지 ── */}
        {liveMessages.length > 0 && (
          <div style={{ marginTop: history.length > 0 ? 8 : 0 }}>
            {history.length > 0 && (
              <div style={{
                fontSize: 10, color: 'rgba(255,255,255,0.3)',
                textAlign: 'center', margin: '4px 0 6px',
                borderBottom: '1px solid rgba(255,255,255,0.07)',
                paddingBottom: 4,
              }}>
                {isEn ? '— Current Session —' : '— 현재 대화 —'}
              </div>
            )}
            <AnimatePresence initial={false}>
              {liveMessages.map(msg => {
                const isUser = msg.role === 'user'
                const isLong = msg.text.length > 300
                const expanded = expandedMsgs.has(msg.id)
                const displayText = isLong && !expanded ? msg.text.slice(0, 280) + '...' : msg.text
                return (
                  <motion.div
                    key={msg.id}
                    initial={{ opacity: 0, y: 6 }}
                    animate={{ opacity: 1, y: 0 }}
                    style={{ display: 'flex', justifyContent: isUser ? 'flex-end' : 'flex-start', marginBottom: 6 }}
                  >
                    <div style={{
                      maxWidth: '86%',
                      padding: '8px 12px',
                      borderRadius: isUser ? '14px 14px 4px 14px' : '4px 14px 14px 14px',
                      background: isUser ? `${primaryColor}66` : 'rgba(255,255,255,0.11)',
                      border: `1px solid ${isUser ? primaryColor + '99' : 'rgba(255,255,255,0.15)'}`,
                      fontSize: 12,
                      color: 'rgba(255,255,255,0.97)',
                      lineHeight: 1.65,
                      whiteSpace: isUser ? 'pre-wrap' : 'normal',
                      wordBreak: 'break-word',
                      boxShadow: isUser ? `0 2px 10px ${primaryColor}33` : '0 2px 8px rgba(0,0,0,0.3)',
                    }}>
                      {isUser ? displayText : renderMarkdown(displayText)}
                      {isLong && (
                        <span
                          onClick={() => toggleExpand(msg.id)}
                          style={{ display: 'block', color: primaryColor, fontSize: 10, fontWeight: 700, marginTop: 4, cursor: 'pointer' }}
                        >
                          {expanded ? (isEn ? '▲ Collapse' : '▲ 접기') : (isEn ? '▼ Show more' : '▼ 더보기')}
                        </span>
                      )}
                    </div>
                    {/* 명확화 선택 버튼 */}
                    {!isUser && msg.clarifyOptions && msg.clarifyOptions.length > 0 && (
                      <div style={{ marginTop: 10, maxWidth: '90%' }}>
                        <div style={{ fontSize: 9.5, color: 'rgba(255,255,255,0.35)', marginBottom: 6, fontWeight: 600, letterSpacing: '0.05em' }}>
                          {isEn ? '▸ SELECT AN OPTION' : '▸ 아래에서 선택하세요'}
                        </div>
                        <div style={{ display: 'flex', flexDirection: 'column', gap: 5 }}>
                          {msg.clarifyOptions.map((opt, oi) => (
                            <button
                              key={oi}
                              onClick={() => msg.onClarifySelect?.(opt)}
                              style={{
                                padding: '8px 14px',
                                borderRadius: 10,
                                border: `1px solid ${primaryColor}55`,
                                background: `${primaryColor}15`,
                                color: 'rgba(255,255,255,0.88)',
                                fontSize: 12,
                                fontWeight: 500,
                                cursor: 'pointer',
                                textAlign: 'left',
                                transition: 'all 0.15s',
                                display: 'flex', alignItems: 'center', gap: 8,
                              }}
                              onMouseEnter={e => {
                                const b = e.currentTarget
                                b.style.background = `${primaryColor}30`
                                b.style.borderColor = `${primaryColor}99`
                                b.style.color = '#fff'
                              }}
                              onMouseLeave={e => {
                                const b = e.currentTarget
                                b.style.background = `${primaryColor}15`
                                b.style.borderColor = `${primaryColor}55`
                                b.style.color = 'rgba(255,255,255,0.88)'
                              }}
                            >
                              <span style={{ fontSize: 10, color: primaryColor, fontWeight: 800, flexShrink: 0 }}>{oi + 1}</span>
                              <span>{opt}</span>
                            </button>
                          ))}
                        </div>
                      </div>
                    )}
                  </motion.div>
                )
              })}
            </AnimatePresence>
          </div>
        )}

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
              {msg.inlineCard2 && <InlineCardRenderer2 card={msg.inlineCard2} accentColor={primaryColor} onPersonaSelect={onPersonaSelect} />}
              {msg.inlineCard3 && <InlineCardRenderer3 card={msg.inlineCard3} />}
              {msg.inlineCard4 && <InlineCardRenderer4 card={msg.inlineCard4} onMacroRun={msg.onMacroRun} />}
              {msg.inlineCard5 && <InlineCard5Renderer card={msg.inlineCard5} accentColor={primaryColor} />}
            </motion.div>
          ))}
        </AnimatePresence>

        {/* savedPreviews 카드 제거됨 — floatingPreview 팝업과 함께 제거 */}

        {/* 마지막 응답 후 follow-up 액션 */}
        {!typing && history.length > 0 && (() => {
          const last = history[history.length - 1]
          const lastAction = messages.filter(m => m.role === 'nexus').slice(-1)[0]
          const actionKey = lastAction?.action ?? ''
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
            <TypingBar primaryColor={primaryColor} steps={typingSteps} lang={lang} />
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

      {/* clarify 대기 중 안내 (투명 박스 없이 인라인으로) */}
      {clarifyPending && (
        <div style={{
          padding: '5px 12px',
          borderTop: `1px solid ${primaryColor}44`,
          background: `${primaryColor}11`,
          display: 'flex', alignItems: 'center', gap: 6,
          fontSize: 10.5, color: primaryColor, fontWeight: 600,
        }}>
          <span>💬</span>
          <span style={{ flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {clarifyQuestion || (lang === 'en' ? 'Additional info needed' : '추가 정보가 필요합니다')}
          </span>
        </div>
      )}

      {/* 입력 바 */}
      <div style={{
        padding: attachedFiles.length > 0 ? '6px 10px 10px' : '8px 10px 10px',
        borderTop: `1px solid ${primaryColor}33`,
        background: `${primaryColor}08`,
        display: 'flex',
        alignItems: 'center',
        gap: 6,
        flexShrink: 0,
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

        <input
          value={input}
          onChange={e => onInputChange(e.target.value)}
          onKeyDown={e => {
            if (e.key === 'Enter' && !e.shiftKey && (input.trim() || attachedFile)) {
              e.preventDefault()
              if (e.nativeEvent.isComposing) {
                setTimeout(() => handleSendAll(), 10)
              } else {
                handleSendAll()
              }
            }
          }}
          placeholder={
            attachedFile
              ? '파일에 대해 질문하거나 Enter로 바로 분석...'
              : clarifyPending
                ? '답변을 입력하세요...'
                : lang === 'ko' ? `${assistantName}에게...` : `Ask ${assistantName}...`
          }
          style={{
            flex: 1, background: clarifyPending ? `${primaryColor}18` : 'rgba(255,255,255,0.07)',
            border: `1.5px solid ${clarifyPending ? primaryColor : attachedFile ? primaryColor : primaryColor}${clarifyPending || attachedFile ? 'bb' : '55'}`,
            borderRadius: 16, padding: '8px 14px',
            color: 'rgba(255,255,255,0.97)', fontSize: 13, outline: 'none',
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
