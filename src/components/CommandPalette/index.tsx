import { useState, useEffect, useRef, useCallback, useMemo } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { useAppStore, type ViewId } from '../../stores/appStore'
import { processNaturalLanguage, AIResponseCard, type NLResult } from '../AIAssistant'

const RECENT_KEY = 'ttuktak-recent'
const PINNED_KEY = 'ttuktak-pinned'
const DEFAULT_PINNED = ['home', 'autoclean', 'clipboard']

interface Command {
  id: string
  icon: string
  label: string
  subtitle?: string
  keywords: string[]
  action: () => void
}

function loadRecent(): string[] {
  try {
    return JSON.parse(localStorage.getItem(RECENT_KEY) ?? '[]') as string[]
  } catch {
    return []
  }
}

function saveRecent(id: string) {
  const prev = loadRecent().filter((r) => r !== id)
  localStorage.setItem(RECENT_KEY, JSON.stringify([id, ...prev].slice(0, 5)))
}

function loadPinned(): string[] {
  try {
    const stored = localStorage.getItem(PINNED_KEY)
    return stored ? (JSON.parse(stored) as string[]) : DEFAULT_PINNED
  } catch {
    return DEFAULT_PINNED
  }
}

// Unit conversion
interface ConvResult {
  value: string
  label: string
}

function tryUnitConversion(input: string): ConvResult | null {
  const lower = input.toLowerCase().trim()

  // KB/MB/GB/TB conversions
  const sizeMatch = lower.match(/^([\d.]+)\s*(kb|mb|gb|tb)\s+to\s+(kb|mb|gb|tb)$/)
  if (sizeMatch) {
    const num = parseFloat(sizeMatch[1])
    const from = sizeMatch[2]
    const to = sizeMatch[3]
    const units: Record<string, number> = { kb: 1, mb: 1024, gb: 1024 * 1024, tb: 1024 * 1024 * 1024 }
    const result = (num * units[from]) / units[to]
    return { value: result % 1 === 0 ? String(result) : result.toFixed(4), label: to.toUpperCase() }
  }

  // Currency note
  if (/달러.*원화|원화.*달러|usd.*krw|krw.*usd/.test(lower)) {
    return { value: '실시간 환율 조회 필요', label: '(인터넷 연결 필요)' }
  }

  return null
}

// Calculator
function tryCalculate(input: string): number | null {
  const trimmed = input.trim()
  if (!/^[\d\s+\-*/().]+$/.test(trimmed)) return null
  if (!/\d/.test(trimmed)) return null
  try {
    // eslint-disable-next-line no-new-func
    const result = new Function('return ' + trimmed)() as unknown
    if (typeof result === 'number' && isFinite(result)) return result
  } catch {
    // ignore
  }
  return null
}

function highlight(text: string, query: string): React.ReactNode {
  if (!query.trim()) return text
  const idx = text.toLowerCase().indexOf(query.toLowerCase())
  if (idx === -1) return text
  return (
    <>
      {text.slice(0, idx)}
      <mark style={{ background: 'rgba(79,126,247,0.35)', color: 'inherit', borderRadius: 3, padding: '0 2px' }}>
        {text.slice(idx, idx + query.length)}
      </mark>
      {text.slice(idx + query.length)}
    </>
  )
}

function useCommands(): Command[] {
  const { setView, startScan } = useAppStore()
  return useMemo(() => [
    { id: 'home',       icon: '🏠', label: 'PC 진단',           subtitle: '전체 시스템 점검',       keywords: ['느려', '진단', '점검', '검사', 'scan', 'home'] },
    { id: 'repair',     icon: '🔧', label: '원클릭 수리',        subtitle: '발견된 문제 자동 수리',   keywords: ['고쳐', '수리', '치료', '고장', 'repair'] },
    { id: 'security',   icon: '🛡️', label: '해킹 탐지',          subtitle: '보안 위협 실시간 확인',   keywords: ['해킹', '보안', '바이러스', '악성', 'security'] },
    { id: 'autoclean',  icon: '🧹', label: 'PC 정리',            subtitle: '임시파일, 캐시 청소',     keywords: ['정리', '청소', '임시파일', '캐시', 'autoclean'] },
    { id: 'monitor',    icon: '📊', label: '실시간 모니터',      subtitle: 'CPU/메모리/온도 확인',    keywords: ['cpu', '온도', '메모리', '모니터', 'monitor', '발열'] },
    { id: 'clipboard',  icon: '📋', label: '클립보드 히스토리',  subtitle: '복사 기록 검색',          keywords: ['복사', '클립', '붙여', 'clipboard'] },
    { id: 'translate',  icon: '🌐', label: '번역하기',            subtitle: '텍스트 즉시 번역',        keywords: ['번역', '영어', '일어', '중국어', 'translate'] },
    { id: 'files',      icon: '🔄', label: '파일 변환',          subtitle: 'PDF, 이미지, 엑셀',       keywords: ['변환', 'pdf', '엑셀', '워드', 'convert'] },
    { id: 'organize',   icon: '📁', label: '스마트 파일 정리',   subtitle: '다운로드/바탕화면 정리',  keywords: ['정리', '폴더', '다운로드', '바탕화면', 'organize'] },
    { id: 'memo',       icon: '📝', label: '메모',               subtitle: '빠른 메모 & 할 일',       keywords: ['메모', '노트', '기록', 'memo', '할일', 'todo'] },
    { id: 'privacy',    icon: '🔒', label: '프라이버시',         subtitle: 'MS 기능 & 텔레메트리 제어', keywords: ['프라이버시', '텔레메트리', '코파일럿', '원드라이브'] },
    { id: 'focus',      icon: '🎯', label: '집중 모드',          subtitle: '포모도로 타이머',          keywords: ['집중', '포모도로', '방해금지', '타이머', 'focus'] },
    { id: 'daily',      icon: '☀️', label: '데일리 리포트',     subtitle: '오늘의 PC 건강 리포트',   keywords: ['데일리', '아침', '리포트', '오늘', 'daily'] },
    { id: 'voicememo',  icon: '🎙️', label: '음성 메모',          subtitle: '말하면 텍스트로 변환',    keywords: ['음성', '말하기', '녹음', 'voice'] },
    { id: 'predictive', icon: '🔮', label: 'AI 예측 관리',       subtitle: 'PC 미래 상태 예측',       keywords: ['예측', '미래', '예방', 'predictive'] },
    { id: 'settings',   icon: '⚙️', label: '설정',               subtitle: '단축키, 테마, 라이선스',  keywords: ['설정', '환경설정', 'settings'] },
  ].map((c) => ({
    ...c,
    action: () => {
      if (c.id === 'home') startScan()
      setView(c.id as ViewId)
    },
  })), [setView, startScan])
}

export function CommandPalette() {
  const { toggleCommand } = useAppStore()
  const [query, setQuery] = useState('')
  const [selected, setSelected] = useState(0)
  const inputRef = useRef<HTMLInputElement>(null)
  const listRef = useRef<HTMLDivElement>(null)
  const allCommands = useCommands()

  const [recentIds, setRecentIds] = useState<string[]>(loadRecent)
  const pinnedIds = useMemo(() => loadPinned(), [])

  const calcResult = useMemo(() => query.trim() ? tryCalculate(query) : null, [query])
  const unitResult = useMemo(() => query.trim() ? tryUnitConversion(query) : null, [query])
  const nlResult = useMemo<NLResult | null>(() => {
    if (!query.trim() || calcResult !== null || unitResult !== null) return null
    return processNaturalLanguage(query)
  }, [query, calcResult, unitResult])

  const filtered = useMemo(() => {
    if (!query.trim()) return []
    const q = query.toLowerCase()
    return allCommands.filter(
      (c) =>
        c.label.toLowerCase().includes(q) ||
        c.subtitle?.toLowerCase().includes(q) ||
        c.keywords.some((k) => k.toLowerCase().includes(q))
    )
  }, [query, allCommands])

  const recentCommands = useMemo(
    () => recentIds.map((id) => allCommands.find((c) => c.id === id)).filter(Boolean) as Command[],
    [recentIds, allCommands]
  )
  const pinnedCommands = useMemo(
    () => pinnedIds.map((id) => allCommands.find((c) => c.id === id)).filter(Boolean) as Command[],
    [pinnedIds, allCommands]
  )

  useEffect(() => { setSelected(0) }, [query])

  useEffect(() => {
    inputRef.current?.focus()
  }, [])

  useEffect(() => {
    const el = listRef.current?.children[selected] as HTMLElement | undefined
    el?.scrollIntoView({ block: 'nearest' })
  }, [selected])

  const run = useCallback(
    (cmd: Command) => {
      cmd.action()
      saveRecent(cmd.id)
      setRecentIds(loadRecent())
      toggleCommand()
    },
    [toggleCommand]
  )

  const runFiltered = useCallback(
    (idx: number) => {
      if (filtered[idx]) run(filtered[idx])
    },
    [filtered, run]
  )

  const handleKey = (e: React.KeyboardEvent) => {
    switch (e.key) {
      case 'ArrowDown': e.preventDefault(); setSelected((s) => Math.min(s + 1, filtered.length - 1)); break
      case 'ArrowUp':   e.preventDefault(); setSelected((s) => Math.max(s - 1, 0)); break
      case 'Enter':     e.preventDefault(); runFiltered(selected); break
      case 'Escape':    toggleCommand(); break
    }
  }

  const showEmpty = !query.trim()

  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      transition={{ duration: 0.12 }}
      onClick={toggleCommand}
      style={{
        position: 'fixed',
        inset: 0,
        background: 'rgba(0,0,0,0.65)',
        backdropFilter: 'blur(8px)',
        WebkitBackdropFilter: 'blur(8px)',
        zIndex: 1000,
      }}
    >
      <motion.div
        initial={{ opacity: 0, scale: 0.94, y: -12 }}
        animate={{ opacity: 1, scale: 1, y: 0 }}
        exit={{ opacity: 0, scale: 0.94, y: -12 }}
        transition={{ duration: 0.15, ease: [0.34, 1.56, 0.64, 1] }}
        onClick={(e) => e.stopPropagation()}
        style={{
          position: 'absolute',
          top: 80,
          left: '50%',
          transform: 'translateX(-50%)',
          width: 560,
          maxWidth: '94vw',
          maxHeight: 440,
          background: 'var(--bg-elevated)',
          border: '1px solid var(--glass-border)',
          borderRadius: 'var(--radius-lg)',
          boxShadow: 'var(--shadow-lg)',
          overflow: 'hidden',
          display: 'flex',
          flexDirection: 'column',
        }}
      >
        {/* Search bar */}
        <div style={{
          display: 'flex',
          alignItems: 'center',
          gap: 12,
          padding: '14px 18px',
          borderBottom: '1px solid var(--border-subtle)',
          flexShrink: 0,
        }}>
          <span style={{ fontSize: 16, flexShrink: 0 }}>🔍</span>
          <input
            ref={inputRef}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={handleKey}
            placeholder="무엇을 도와드릴까요? (계산: 100+200, 단위: 100kb to mb)"
            style={{
              flex: 1,
              background: 'none',
              border: 'none',
              outline: 'none',
              color: 'var(--text-primary)',
              fontSize: 15,
              fontFamily: 'Pretendard, Inter, sans-serif',
              caretColor: 'var(--accent-primary)',
            }}
          />
          {query && (
            <button
              onClick={() => setQuery('')}
              style={{ background: 'none', border: 'none', color: 'var(--text-muted)', fontSize: 13, cursor: 'pointer' }}
            >✕</button>
          )}
          <kbd style={{
            padding: '3px 8px',
            borderRadius: 6,
            background: 'var(--bg-surface)',
            border: '1px solid var(--border-default)',
            color: 'var(--text-muted)',
            fontSize: 11,
            fontFamily: 'monospace',
          }}>ESC</kbd>
        </div>

        {/* Scrollable body */}
        <div style={{ flex: 1, overflowY: 'auto' }}>
          {/* Calculator result */}
          {calcResult !== null && (
            <div style={{
              margin: '8px 8px 0',
              padding: '12px 16px',
              borderRadius: 'var(--radius-md)',
              background: 'rgba(34,197,94,0.08)',
              border: '1px solid rgba(34,197,94,0.25)',
              display: 'flex',
              alignItems: 'center',
              gap: 10,
            }}>
              <span style={{ fontSize: 18 }}>🧮</span>
              <span style={{ fontSize: 14, color: 'var(--text-secondary)' }}>{query} =</span>
              <span style={{ fontSize: 22, fontWeight: 700, color: 'var(--success)' }}>{calcResult}</span>
            </div>
          )}

          {/* Unit conversion result */}
          {unitResult !== null && (
            <div style={{
              margin: '8px 8px 0',
              padding: '12px 16px',
              borderRadius: 'var(--radius-md)',
              background: 'rgba(245,158,11,0.08)',
              border: '1px solid rgba(245,158,11,0.25)',
              display: 'flex',
              alignItems: 'center',
              gap: 10,
            }}>
              <span style={{ fontSize: 18 }}>🔄</span>
              <span style={{ fontSize: 14, color: 'var(--text-secondary)' }}>{query} →</span>
              <span style={{ fontSize: 20, fontWeight: 700, color: 'var(--warning)' }}>
                {unitResult.value} <span style={{ fontSize: 14, fontWeight: 400 }}>{unitResult.label}</span>
              </span>
            </div>
          )}

          {/* AI NL result */}
          {nlResult && (
            <AIResponseCard result={nlResult} onExecute={toggleCommand} />
          )}

          {/* Pinned chips (no query) */}
          {showEmpty && pinnedCommands.length > 0 && (
            <div style={{ padding: '10px 12px 4px' }}>
              <div style={{ fontSize: 10, fontWeight: 700, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.06em', marginBottom: 8 }}>
                📌 고정
              </div>
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>
                {pinnedCommands.map((cmd) => (
                  <motion.button
                    key={cmd.id}
                    onClick={() => run(cmd)}
                    whileHover={{ scale: 1.04 }}
                    whileTap={{ scale: 0.96 }}
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: 6,
                      padding: '5px 12px',
                      borderRadius: 20,
                      border: '1px solid var(--border-default)',
                      background: 'var(--bg-surface)',
                      color: 'var(--text-primary)',
                      fontSize: 12,
                      cursor: 'pointer',
                      fontWeight: 500,
                    }}
                  >
                    <span>{cmd.icon}</span>
                    <span>{cmd.label}</span>
                  </motion.button>
                ))}
              </div>
            </div>
          )}

          {/* Recent commands (no query) */}
          {showEmpty && recentCommands.length > 0 && (
            <div style={{ padding: '10px 12px 4px' }}>
              <div style={{ fontSize: 10, fontWeight: 700, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.06em', marginBottom: 4 }}>
                🕘 최근 사용
              </div>
              {recentCommands.map((cmd) => (
                <CommandRow key={cmd.id} cmd={cmd} selected={false} query="" onRun={() => run(cmd)} onHover={() => {}} />
              ))}
            </div>
          )}

          {/* Filtered command list */}
          {!showEmpty && (
            <div ref={listRef} style={{ padding: '6px 8px' }}>
              {query.trim() && (
                <div style={{ padding: '4px 12px 4px', fontSize: 10, color: 'var(--text-muted)', fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.05em' }}>
                  {filtered.length > 0 ? `검색 결과 ${filtered.length}개` : '검색 결과 없음'}
                </div>
              )}
              <AnimatePresence mode="popLayout">
                {filtered.map((cmd, i) => (
                  <CommandRow
                    key={cmd.id}
                    cmd={cmd}
                    selected={selected === i}
                    query={query}
                    onRun={() => run(cmd)}
                    onHover={() => setSelected(i)}
                  />
                ))}
              </AnimatePresence>
            </div>
          )}

          {/* Empty state with all commands */}
          {showEmpty && !recentCommands.length && !pinnedCommands.length && (
            <div style={{ padding: '6px 8px' }}>
              <div style={{ padding: '4px 12px 6px', fontSize: 10, color: 'var(--text-muted)', fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.05em' }}>
                ⚡ 빠른 액션
              </div>
              {allCommands.slice(0, 6).map((cmd, i) => (
                <CommandRow key={cmd.id} cmd={cmd} selected={false} query="" onRun={() => run(cmd)} onHover={() => setSelected(i)} />
              ))}
            </div>
          )}
        </div>

        {/* Footer hint */}
        <div style={{
          padding: '8px 18px',
          borderTop: '1px solid var(--border-subtle)',
          display: 'flex',
          gap: 14,
          color: 'var(--text-muted)',
          fontSize: 11,
          background: 'var(--bg-surface)',
          flexShrink: 0,
        }}>
          {[['↑↓', '이동'], ['Enter', '실행'], ['Esc', '닫기']].map(([k, v]) => (
            <span key={k} style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
              <kbd style={{ padding: '1px 5px', borderRadius: 4, background: 'var(--bg-elevated)', border: '1px solid var(--border-default)', fontFamily: 'monospace', fontSize: 10 }}>{k}</kbd>
              {v}
            </span>
          ))}
          <span style={{ flex: 1, textAlign: 'right' }}>Alt+Space로 언제든 열기</span>
        </div>
      </motion.div>
    </motion.div>
  )
}

function CommandRow({
  cmd,
  selected,
  query,
  onRun,
  onHover,
}: {
  cmd: Command
  selected: boolean
  query: string
  onRun: () => void
  onHover: () => void
}) {
  return (
    <motion.div
      layout
      initial={{ opacity: 0, x: -6 }}
      animate={{ opacity: 1, x: 0 }}
      exit={{ opacity: 0, x: -6 }}
      onClick={onRun}
      onMouseEnter={onHover}
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 12,
        padding: '9px 14px',
        borderRadius: 'var(--radius-sm)',
        background: selected ? 'var(--glass-bg)' : 'transparent',
        border: `1px solid ${selected ? 'var(--border-default)' : 'transparent'}`,
        cursor: 'pointer',
        transition: 'all 80ms',
      }}
    >
      <span style={{ fontSize: 17, width: 26, textAlign: 'center', flexShrink: 0 }}>{cmd.icon}</span>
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ fontSize: 13, fontWeight: 600, color: 'var(--text-primary)' }}>
          {highlight(cmd.label, query)}
        </div>
        {cmd.subtitle && (
          <div style={{ fontSize: 11, color: 'var(--text-secondary)', marginTop: 1 }}>{cmd.subtitle}</div>
        )}
      </div>
      {selected && (
        <kbd style={{
          padding: '2px 7px',
          borderRadius: 5,
          background: 'rgba(79,126,247,0.15)',
          border: '1px solid rgba(79,126,247,0.3)',
          color: 'var(--accent-primary)',
          fontSize: 10,
          fontFamily: 'monospace',
          flexShrink: 0,
        }}>Enter</kbd>
      )}
    </motion.div>
  )
}
