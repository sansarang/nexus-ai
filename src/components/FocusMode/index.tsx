import { useEffect, useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'

type FocusPhase = 'work' | 'break'

interface Preset {
  label: string
  workMin: number
  breakMin: number
}

const PRESETS: Preset[] = [
  { label: '25 / 5',   workMin: 25, breakMin: 5  },
  { label: '50 / 10',  workMin: 50, breakMin: 10 },
  { label: '90 / 20',  workMin: 90, breakMin: 20 },
]

const ACTIVE_EFFECTS = ['알림 차단 중', '백그라운드 앱 슬립 중', '고성능 모드']

export function FocusView() {
  const [active, setActive] = useState(false)
  const [isBreak, setIsBreak] = useState(false)
  const [remaining, setRemaining] = useState(25 * 60)
  const [round, setRound] = useState(0)
  const [workMin, setWorkMin] = useState(25)
  const [breakMin, setBreakMin] = useState(5)
  const [presetIdx, setPresetIdx] = useState(0)

  const total = isBreak ? breakMin * 60 : workMin * 60
  const progress = (total - remaining) / total

  // Timer tick
  useEffect(() => {
    if (!active) return
    const id = setInterval(() => {
      setRemaining((prev) => {
        if (prev <= 1) {
          // transition phase
          setIsBreak((wasBreak) => {
            if (!wasBreak) {
              // work → break
              setRound((r) => r + 1)
              setRemaining(breakMin * 60)
            } else {
              // break → work
              setRemaining(workMin * 60)
            }
            return !wasBreak
          })
          return 0
        }
        return prev - 1
      })
    }, 1000)
    return () => clearInterval(id)
  }, [active, workMin, breakMin])

  const minutes = String(Math.floor(remaining / 60)).padStart(2, '0')
  const seconds = String(remaining % 60).padStart(2, '0')

  const r = 88
  const cx = 100
  const cy = 100
  const circumference = 2 * Math.PI * r
  const strokeDashoffset = circumference * (1 - progress)
  const strokeColor = isBreak ? 'var(--success)' : 'var(--accent-primary)'

  const selectPreset = (idx: number) => {
    if (active) return
    const p = PRESETS[idx]
    setPresetIdx(idx)
    setWorkMin(p.workMin)
    setBreakMin(p.breakMin)
    setRemaining(p.workMin * 60)
    setIsBreak(false)
    setRound(0)
  }

  const handleStart = () => {
    if (!active) {
      setActive(true)
    } else {
      setActive(false)
    }
  }

  const handleReset = () => {
    setActive(false)
    setIsBreak(false)
    setRemaining(workMin * 60)
    setRound(0)
  }

  return (
    <div style={{
      flex: 1, overflowY: 'auto', display: 'flex', alignItems: 'center', justifyContent: 'center',
      background: 'var(--bg-base)', color: 'var(--text-primary)', padding: 24,
    }}>
      <motion.div
        initial={{ opacity: 0, scale: 0.95 }}
        animate={{ opacity: 1, scale: 1 }}
        style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 20, maxWidth: 400, width: '100%' }}
      >
        <h2 style={{ margin: 0, fontSize: 20, fontWeight: 700 }}>🎯 집중 모드</h2>

        {/* Phase badge */}
        <motion.div
          key={isBreak ? 'break' : 'work'}
          initial={{ opacity: 0, y: -6 }}
          animate={{ opacity: 1, y: 0 }}
          style={{
            padding: '4px 16px', borderRadius: 20,
            background: isBreak ? 'rgba(34,197,94,0.12)' : 'rgba(79,126,247,0.12)',
            border: `1px solid ${isBreak ? 'rgba(34,197,94,0.4)' : 'rgba(79,126,247,0.4)'}`,
            fontSize: 13, fontWeight: 600,
            color: isBreak ? 'var(--success)' : 'var(--accent-primary)',
          }}
        >
          {active ? (isBreak ? '🍃 휴식 시간' : '💪 집중 시간') : '⏸ 대기 중'}
        </motion.div>

        {/* SVG Circle Timer */}
        <div style={{ position: 'relative', width: 200, height: 200 }}>
          <svg width={200} height={200} style={{ transform: 'rotate(-90deg)' }}>
            <circle cx={cx} cy={cy} r={r} fill="none" stroke="var(--border-subtle)" strokeWidth={8} />
            <circle
              cx={cx} cy={cy} r={r} fill="none"
              stroke={strokeColor}
              strokeWidth={8}
              strokeLinecap="round"
              strokeDasharray={circumference}
              strokeDashoffset={strokeDashoffset}
              style={{ transition: 'stroke-dashoffset 0.95s ease, stroke 0.3s ease' }}
            />
          </svg>
          <div style={{
            position: 'absolute', inset: 0,
            display: 'flex', flexDirection: 'column',
            alignItems: 'center', justifyContent: 'center',
          }}>
            <div style={{ fontSize: 40, fontWeight: 800, fontVariantNumeric: 'tabular-nums', letterSpacing: -2, color: 'var(--text-primary)' }}>
              {minutes}:{seconds}
            </div>
            <div style={{ fontSize: 11, color: 'var(--text-muted)', marginTop: 2 }}>
              {isBreak ? '휴식' : '집중'}
            </div>
          </div>
        </div>

        {/* Preset buttons */}
        {!active && (
          <div style={{ display: 'flex', gap: 8 }}>
            {PRESETS.map((p, i) => (
              <button
                key={p.label}
                onClick={() => selectPreset(i)}
                style={{
                  padding: '6px 14px', borderRadius: 'var(--radius-sm)',
                  border: `1px solid ${i === presetIdx ? 'var(--accent-primary)' : 'var(--glass-border)'}`,
                  background: i === presetIdx ? 'rgba(79,126,247,0.12)' : 'transparent',
                  color: i === presetIdx ? 'var(--accent-primary)' : 'var(--text-secondary)',
                  cursor: 'pointer', fontSize: 13,
                }}
              >{p.label}</button>
            ))}
          </div>
        )}

        {/* Round indicator */}
        <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} style={{
              width: 10, height: 10, borderRadius: '50%',
              background: i < (round % 4) ? strokeColor : 'rgba(255,255,255,0.1)',
              transition: 'background 0.3s ease',
            }} />
          ))}
          <span style={{ fontSize: 12, color: 'var(--text-muted)', marginLeft: 4 }}>라운드 {round}</span>
        </div>

        {/* Active effects */}
        <AnimatePresence>
          {active && (
            <motion.div
              initial={{ opacity: 0, height: 0 }}
              animate={{ opacity: 1, height: 'auto' }}
              exit={{ opacity: 0, height: 0 }}
              style={{
                display: 'flex', flexDirection: 'column', gap: 4,
                padding: '10px 16px', borderRadius: 'var(--radius-md)',
                background: 'rgba(79,126,247,0.06)', border: '1px solid rgba(79,126,247,0.2)',
                width: '100%',
              }}
            >
              {ACTIVE_EFFECTS.map((e) => (
                <div key={e} style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 12, color: 'var(--text-secondary)' }}>
                  <div style={{ width: 6, height: 6, borderRadius: '50%', background: 'var(--success)' }} />
                  {e}
                </div>
              ))}
            </motion.div>
          )}
        </AnimatePresence>

        {/* Health message every 4 rounds */}
        <AnimatePresence>
          {round > 0 && round % 4 === 0 && (
            <motion.div
              initial={{ opacity: 0, scale: 0.9 }}
              animate={{ opacity: 1, scale: 1 }}
              exit={{ opacity: 0 }}
              style={{
                padding: '10px 16px', borderRadius: 'var(--radius-md)',
                background: 'rgba(34,197,94,0.08)', border: '1px solid rgba(34,197,94,0.3)',
                fontSize: 13, color: 'var(--success)', textAlign: 'center',
              }}
            >
              🎉 {round}라운드 완료! 충분한 휴식을 취하세요.
            </motion.div>
          )}
        </AnimatePresence>

        {/* Control buttons */}
        <div style={{ display: 'flex', gap: 10 }}>
          <motion.button
            onClick={handleStart}
            whileHover={{ scale: 1.04 }}
            whileTap={{ scale: 0.96 }}
            style={{
              padding: '10px 28px', borderRadius: 'var(--radius-md)',
              border: 'none', background: active ? 'rgba(239,68,68,0.8)' : 'var(--accent-primary)',
              color: '#fff', fontSize: 14, fontWeight: 600, cursor: 'pointer',
            }}
          >{active ? '⏸ 일시정지' : '▶ 시작'}</motion.button>
          <motion.button
            onClick={handleReset}
            whileHover={{ scale: 1.04 }}
            whileTap={{ scale: 0.96 }}
            style={{
              padding: '10px 18px', borderRadius: 'var(--radius-md)',
              border: '1px solid var(--glass-border)', background: 'var(--glass-bg)',
              color: 'var(--text-primary)', fontSize: 14, cursor: 'pointer',
            }}
          >초기화</motion.button>
        </div>
      </motion.div>
    </div>
  )
}
