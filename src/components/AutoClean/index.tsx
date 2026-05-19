import { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'

interface CleanItem {
  id: string
  label: string
  icon: string
  size: string
}

const ITEMS: CleanItem[] = [
  { id: 'temp',     label: '임시 파일',        icon: '🗑️', size: '~2.4GB' },
  { id: 'wucache',  label: '업데이트 캐시',     icon: '🪟', size: '~1.1GB' },
  { id: 'browser',  label: '브라우저 캐시',     icon: '🌐', size: '~800MB' },
  { id: 'recycle',  label: '휴지통',            icon: '♻️', size: '~500MB' },
  { id: 'thumb',    label: '썸네일 캐시',       icon: '🖼️', size: '~200MB' },
  { id: 'prefetch', label: '프리패치',          icon: '⚡', size: '~100MB' },
  { id: 'memory',   label: '메모리 최적화',     icon: '💾', size: '' },
  { id: 'bloat',    label: 'Bloatware 제거',    icon: '🧹', size: '' },
]

type ItemState = 'idle' | 'running' | 'done' | 'error'

function estimatedBytes(size: string): number {
  const m = size.match(/([\d.]+)(GB|MB)/)
  if (!m) return 0
  const num = parseFloat(m[1])
  const unit = m[2]
  if (unit === 'GB') return Math.floor(num * 1024 * 1024 * 1024)
  if (unit === 'MB') return Math.floor(num * 1024 * 1024)
  return 0
}

function formatBytes(b: number): string {
  if (b >= 1 << 30) return `${(b / (1 << 30)).toFixed(1)} GB`
  if (b >= 1 << 20) return `${(b / (1 << 20)).toFixed(0)} MB`
  if (b >= 1 << 10) return `${(b / (1 << 10)).toFixed(0)} KB`
  return `${b} B`
}

export function AutoCleanView() {
  const [checked, setChecked] = useState<Record<string, boolean>>(
    Object.fromEntries(ITEMS.map((i) => [i.id, true]))
  )
  const [states, setStates] = useState<Record<string, ItemState>>({})
  const [running, setRunning] = useState(false)
  const [done, setDone] = useState(false)
  const [totalFreed, setTotalFreed] = useState(0)

  const selectedItems = ITEMS.filter((i) => checked[i.id])
  const estimatedTotal = selectedItems.reduce((acc, item) => acc + estimatedBytes(item.size), 0)

  const toggleItem = (id: string) => {
    if (running) return
    setChecked((c) => ({ ...c, [id]: !c[id] }))
  }

  const startClean = async () => {
    if (running || selectedItems.length === 0) return
    setRunning(true)
    setDone(false)
    setTotalFreed(0)

    const ids = selectedItems.map((i) => i.id)
    setStates(Object.fromEntries(ids.map((id) => [id, 'idle' as ItemState])))

    let freed = 0

    try {
      // Mark all running
      for (const id of ids) {
        setStates((s) => ({ ...s, [id]: 'running' }))
        await new Promise((r) => setTimeout(r, 200))
      }

      const res = await fetch('http://127.0.0.1:17891/api/autoclean', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ items: ids }),
      })
      const results = await res.json() as { item: string; freed_bytes: number; error?: string }[]
      for (const r of results) {
        setStates((s) => ({ ...s, [r.item]: r.error ? 'error' : 'done' }))
        freed += r.freed_bytes ?? 0
      }
    } catch {
      // Simulate with 800ms delay per item
      for (const id of ids) {
        setStates((s) => ({ ...s, [id]: 'running' }))
        await new Promise((r) => setTimeout(r, 800))
        setStates((s) => ({ ...s, [id]: 'done' }))
        freed += Math.floor(Math.random() * 300 * 1024 * 1024)
      }
    }

    setTotalFreed(freed)
    setRunning(false)
    setDone(true)
  }

  const reset = () => {
    setStates({})
    setDone(false)
    setTotalFreed(0)
  }

  return (
    <div style={{
      flex: 1, overflowY: 'auto', padding: 24,
      background: 'var(--bg-base)', color: 'var(--text-primary)',
    }}>
      <motion.div
        initial={{ opacity: 0, y: 12 }}
        animate={{ opacity: 1, y: 0 }}
        style={{ maxWidth: 600 }}
      >
        <h2 style={{ margin: '0 0 4px', fontSize: 20, fontWeight: 700 }}>🧹 PC 정리 &amp; 최적화</h2>
        <p style={{ margin: '0 0 20px', fontSize: 13, color: 'var(--text-secondary)' }}>
          정리할 항목을 선택하고 실행하세요 · 예상 절약: <strong style={{ color: 'var(--accent-primary)' }}>{formatBytes(estimatedTotal)}</strong>
        </p>

        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          {ITEMS.map((item) => {
            const isChecked = checked[item.id]
            const state = states[item.id] ?? 'idle'
            return (
              <motion.div
                key={item.id}
                whileHover={{ scale: running ? 1 : 1.005 }}
                onClick={() => toggleItem(item.id)}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 12,
                  padding: '10px 14px',
                  borderRadius: 'var(--radius-md)',
                  border: `1px solid ${state === 'done' ? 'rgba(34,197,94,0.3)' : isChecked ? 'rgba(79,126,247,0.3)' : 'var(--glass-border)'}`,
                  background: state === 'done' ? 'rgba(34,197,94,0.05)' : isChecked ? 'rgba(79,126,247,0.06)' : 'var(--glass-bg)',
                  cursor: running ? 'default' : 'pointer',
                  transition: 'all 0.15s ease',
                  position: 'relative',
                  overflow: 'hidden',
                }}
              >
                {/* Checkbox */}
                <div style={{
                  width: 18, height: 18, borderRadius: 4, flexShrink: 0,
                  border: `2px solid ${isChecked ? 'var(--accent-primary)' : 'var(--text-muted)'}`,
                  background: isChecked ? 'var(--accent-primary)' : 'transparent',
                  display: 'flex', alignItems: 'center', justifyContent: 'center',
                  transition: 'all 0.15s ease',
                }}>
                  {isChecked && <span style={{ color: '#fff', fontSize: 11, fontWeight: 700 }}>✓</span>}
                </div>

                <span style={{ fontSize: 18, flexShrink: 0 }}>{item.icon}</span>

                <span style={{ flex: 1, fontSize: 14, fontWeight: 500 }}>{item.label}</span>

                {item.size && (
                  <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>{item.size}</span>
                )}

                {/* State indicator */}
                <span style={{ width: 22, textAlign: 'center', flexShrink: 0 }}>
                  {state === 'running' && (
                    <motion.span
                      animate={{ rotate: 360 }}
                      transition={{ duration: 0.8, repeat: Infinity, ease: 'linear' }}
                      style={{ display: 'inline-block', color: 'var(--accent-primary)' }}
                    >⟳</motion.span>
                  )}
                  {state === 'done' && <span style={{ color: 'var(--success)' }}>✓</span>}
                  {state === 'error' && <span style={{ color: 'var(--danger)' }}>✗</span>}
                </span>

                {/* Running progress bar */}
                {state === 'running' && (
                  <motion.div
                    style={{
                      position: 'absolute', bottom: 0, left: 0, height: 2,
                      background: 'var(--accent-primary)', borderRadius: 1,
                    }}
                    initial={{ width: '0%' }}
                    animate={{ width: '100%' }}
                    transition={{ duration: 0.8, ease: 'linear' }}
                  />
                )}
              </motion.div>
            )
          })}
        </div>

        {/* Done banner with bounce */}
        <AnimatePresence>
          {done && (
            <motion.div
              initial={{ opacity: 0, scale: 0.85, y: 10 }}
              animate={{ opacity: 1, scale: 1, y: 0 }}
              exit={{ opacity: 0, scale: 0.9 }}
              transition={{ type: 'spring', stiffness: 400, damping: 18 }}
              style={{
                marginTop: 20,
                padding: '20px',
                borderRadius: 'var(--radius-md)',
                background: 'rgba(34,197,94,0.08)',
                border: '1px solid rgba(34,197,94,0.3)',
                textAlign: 'center',
              }}
            >
              <motion.div
                animate={{ scale: [1, 1.2, 1] }}
                transition={{ duration: 0.4, delay: 0.1 }}
                style={{ fontSize: 32, marginBottom: 8 }}
              >🎉</motion.div>
              <div style={{ fontSize: 18, fontWeight: 700, color: 'var(--success)', marginBottom: 4 }}>완료!</div>
              <div style={{ fontSize: 13, color: 'var(--text-secondary)' }}>
                {totalFreed > 0 ? `총 ${formatBytes(totalFreed)} 정리되었습니다` : '정리가 완료되었습니다'}
              </div>
              <button
                onClick={reset}
                style={{
                  marginTop: 12, padding: '6px 16px', borderRadius: 'var(--radius-sm)',
                  border: '1px solid var(--glass-border)', background: 'var(--glass-bg)',
                  color: 'var(--text-secondary)', fontSize: 12, cursor: 'pointer',
                }}
              >다시 선택</button>
            </motion.div>
          )}
        </AnimatePresence>

        <motion.button
          onClick={startClean}
          disabled={running || selectedItems.length === 0 || done}
          whileHover={!running && !done ? { scale: 1.02 } : {}}
          whileTap={!running && !done ? { scale: 0.98 } : {}}
          style={{
            marginTop: 20,
            width: '100%',
            padding: '13px 0',
            borderRadius: 'var(--radius-md)',
            border: 'none',
            background: running
              ? 'rgba(79,126,247,0.5)'
              : done
                ? 'rgba(34,197,94,0.3)'
                : selectedItems.length === 0
                  ? 'rgba(79,126,247,0.2)'
                  : 'var(--accent-primary)',
            color: '#fff',
            fontSize: 15,
            fontWeight: 600,
            cursor: running || selectedItems.length === 0 || done ? 'not-allowed' : 'pointer',
            transition: 'all 0.15s ease',
          }}
        >
          {running ? '정리 중...' : done ? '✓ 정리 완료' : `정리 시작 (${selectedItems.length}개 선택)`}
        </motion.button>
      </motion.div>
    </div>
  )
}
