import { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Zap, CheckCircle, RefreshCw } from 'lucide-react'
import { useAppStore } from '../../stores/appStore'
import { RepairSuccess } from '../RepairSuccess'

const QUICK_FIXES = [
  { id: 'temp',     icon: '🗑️', title: '임시 파일 정리',     desc: 'TEMP 폴더, 브라우저 캐시 삭제', gain: 8 },
  { id: 'startup',  icon: '🚀', title: '시작 프로그램 최적화', desc: '불필요한 자동 실행 프로그램 비활성화', gain: 10 },
  { id: 'registry', icon: '🔑', title: '레지스트리 정리',     desc: '오류 레지스트리 항목 제거', gain: 5 },
  { id: 'defrag',   icon: '💾', title: '디스크 최적화',       desc: 'SSD TRIM / HDD 조각 모음', gain: 6 },
  { id: 'network',  icon: '🌐', title: '네트워크 재설정',     desc: 'DNS 캐시, TCP/IP 스택 초기화', gain: 4 },
  { id: 'update',   icon: '🪟', title: 'Windows Update 수리', desc: '업데이트 서비스 재시작 및 캐시 초기화', gain: 7 },
]

type FixState = 'idle' | 'running' | 'done' | 'error'

export function RepairView() {
  const { pcScore, issues, repairIssue, repairAll } = useAppStore()
  const [fixStates, setFixStates] = useState<Record<string, FixState>>({})
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [repairResult, setRepairResult] = useState<{ before: number; after: number } | null>(null)
  const [runningAll, setRunningAll] = useState(false)

  const toggle = (id: string) =>
    setSelected((prev) => {
      const next = new Set(prev)
      next.has(id) ? next.delete(id) : next.add(id)
      return next
    })

  const runFix = async (id: string) => {
    setFixStates((s) => ({ ...s, [id]: 'running' }))
    await new Promise((r) => setTimeout(r, 1200 + Math.random() * 800))
    setFixStates((s) => ({ ...s, [id]: 'done' }))
  }

  const runSelected = async () => {
    if (selected.size === 0) return
    setRunningAll(true)
    const before = pcScore
    for (const id of selected) {
      await runFix(id)
    }
    const gain = [...selected].reduce((acc, id) => {
      const f = QUICK_FIXES.find((f) => f.id === id)
      return acc + (f?.gain ?? 0)
    }, 0)
    setRunningAll(false)
    setRepairResult({ before, after: Math.min(100, before + gain) })
  }

  const fixableIssues = issues.filter((i) => i.fixable)

  return (
    <>
      <AnimatePresence>
        {repairResult && (
          <RepairSuccess
            before={repairResult.before}
            after={repairResult.after}
            onDismiss={() => setRepairResult(null)}
          />
        )}
      </AnimatePresence>

      <div style={{ flex: 1, overflowY: 'auto', padding: '20px 24px', display: 'flex', flexDirection: 'column', gap: 20 }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <h2 style={{ fontSize: 16, fontWeight: 800, color: 'var(--text-primary)', letterSpacing: '-0.02em' }}>🔧 원클릭 수리</h2>
          {selected.size > 0 && (
            <motion.button
              initial={{ opacity: 0, scale: 0.9 }}
              animate={{ opacity: 1, scale: 1 }}
              whileTap={{ scale: 0.95 }}
              onClick={runSelected}
              disabled={runningAll}
              style={{
                padding: '8px 18px',
                borderRadius: 10,
                border: 'none',
                background: 'linear-gradient(135deg, var(--accent-primary), var(--accent-hover))',
                color: '#fff',
                fontSize: 13,
                fontWeight: 700,
                cursor: 'pointer',
                display: 'flex',
                alignItems: 'center',
                gap: 6,
                boxShadow: '0 4px 14px var(--accent-glow)',
              }}
            >
              {runningAll
                ? <><span className="spin" style={{ width: 12, height: 12, border: '2px solid rgba(255,255,255,0.3)', borderTopColor: '#fff', borderRadius: '50%', display: 'inline-block' }} />수리 중...</>
                : <><Zap size={13} />선택 수리 ({selected.size})</>
              }
            </motion.button>
          )}
        </div>

        {/* 빠른 수리 항목 */}
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10 }}>
          {QUICK_FIXES.map((fix) => {
            const state = fixStates[fix.id] ?? 'idle'
            const isSelected = selected.has(fix.id)
            return (
              <motion.div
                key={fix.id}
                layout
                whileHover={{ scale: 1.01 }}
                whileTap={{ scale: 0.98 }}
                onClick={() => state === 'idle' && toggle(fix.id)}
                style={{
                  padding: '14px 16px',
                  borderRadius: 'var(--radius-md)',
                  background: state === 'done'
                    ? 'rgba(34,197,94,0.07)'
                    : isSelected
                    ? 'rgba(79,126,247,0.1)'
                    : 'var(--bg-elevated)',
                  border: `1px solid ${
                    state === 'done' ? 'rgba(34,197,94,0.25)'
                    : isSelected ? 'rgba(79,126,247,0.4)'
                    : 'var(--border-subtle)'
                  }`,
                  cursor: state === 'idle' ? 'pointer' : 'default',
                  display: 'flex',
                  flexDirection: 'column',
                  gap: 8,
                  transition: 'all 0.12s',
                }}
              >
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <span style={{ fontSize: 18 }}>{fix.icon}</span>
                    <span style={{ fontSize: 13, fontWeight: 700, color: 'var(--text-primary)' }}>{fix.title}</span>
                  </div>
                  <div style={{ flexShrink: 0 }}>
                    {state === 'running' && (
                      <span className="spin" style={{ width: 14, height: 14, border: '2px solid var(--border-default)', borderTopColor: 'var(--accent-primary)', borderRadius: '50%', display: 'inline-block' }} />
                    )}
                    {state === 'done' && <CheckCircle size={16} style={{ color: 'var(--success)' }} />}
                    {state === 'idle' && isSelected && (
                      <div style={{ width: 16, height: 16, borderRadius: '50%', background: 'var(--accent-primary)', border: '2px solid var(--accent-primary)', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                        <div style={{ width: 6, height: 6, borderRadius: '50%', background: '#fff' }} />
                      </div>
                    )}
                    {state === 'idle' && !isSelected && (
                      <div style={{ width: 16, height: 16, borderRadius: '50%', border: '2px solid var(--border-default)' }} />
                    )}
                  </div>
                </div>
                <p style={{ fontSize: 11, color: 'var(--text-secondary)', lineHeight: 1.4 }}>{fix.desc}</p>
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                  <span style={{ fontSize: 11, color: 'var(--success)', fontWeight: 600 }}>+{fix.gain}점</span>
                  {state === 'idle' && (
                    <motion.button
                      whileTap={{ scale: 0.95 }}
                      onClick={(e) => { e.stopPropagation(); runFix(fix.id) }}
                      style={{
                        padding: '4px 10px',
                        borderRadius: 6,
                        border: 'none',
                        background: 'rgba(79,126,247,0.15)',
                        color: 'var(--accent-primary)',
                        fontSize: 11,
                        fontWeight: 600,
                        cursor: 'pointer',
                      }}
                    >
                      수리
                    </motion.button>
                  )}
                </div>
              </motion.div>
            )
          })}
        </div>

        {/* 진단에서 발견된 수리 가능 문제 */}
        {fixableIssues.length > 0 && (
          <div>
            <p style={{ fontSize: 12, fontWeight: 700, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.05em', marginBottom: 10 }}>
              진단 결과 — 수리 가능 {fixableIssues.length}개
            </p>
            {fixableIssues.map((issue) => (
              <motion.div
                key={issue.id}
                layout
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 12,
                  padding: '12px 14px',
                  borderRadius: 'var(--radius-sm)',
                  background: 'var(--bg-elevated)',
                  border: '1px solid var(--border-subtle)',
                  marginBottom: 8,
                }}
              >
                <span style={{ fontSize: 16 }}>{issue.severity === 'high' ? '🔴' : issue.severity === 'medium' ? '🟡' : '🟢'}</span>
                <div style={{ flex: 1 }}>
                  <p style={{ fontSize: 13, fontWeight: 600, color: 'var(--text-primary)' }}>{issue.title}</p>
                  <p style={{ fontSize: 11, color: 'var(--text-secondary)' }}>{issue.description}</p>
                </div>
                <motion.button
                  whileTap={{ scale: 0.95 }}
                  onClick={() => repairIssue(issue.id)}
                  style={{
                    padding: '6px 14px',
                    borderRadius: 8,
                    border: 'none',
                    background: 'rgba(79,126,247,0.15)',
                    color: 'var(--accent-primary)',
                    fontSize: 12,
                    fontWeight: 600,
                    cursor: 'pointer',
                    display: 'flex',
                    alignItems: 'center',
                    gap: 5,
                  }}
                >
                  <RefreshCw size={11} />수리
                </motion.button>
              </motion.div>
            ))}
          </div>
        )}
      </div>
    </>
  )
}
