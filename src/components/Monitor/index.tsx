import { useEffect } from 'react'
import { motion } from 'framer-motion'
import { useAppStore, type SystemStats } from '../../stores/appStore'

function colorForValue(v: number): string {
  if (v < 60) return 'var(--success)'
  if (v < 80) return 'var(--warning)'
  return 'var(--danger)'
}

function Sparkline({ history, color }: { history: number[]; color: string }) {
  const w = 40
  const h = 20
  if (history.length < 2) {
    return <svg width={w} height={h} />
  }
  const max = Math.max(...history, 1)
  const points = history.map((v, i) => {
    const x = (i / (history.length - 1)) * w
    const y = h - (v / max) * (h - 2) - 1
    return `${x},${y}`
  })
  return (
    <svg width={w} height={h} viewBox={`0 0 ${w} ${h}`} preserveAspectRatio="none">
      <polyline
        points={points.join(' ')}
        fill="none"
        stroke={color}
        strokeWidth={1.5}
        strokeLinejoin="round"
        strokeLinecap="round"
      />
    </svg>
  )
}

function StatCard({
  label,
  value,
  unit,
  history,
}: {
  label: string
  value: number
  unit: string
  history: number[]
}) {
  const color = colorForValue(value)
  return (
    <motion.div
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      style={{
        flex: '1 1 calc(50% - 8px)',
        minWidth: 160,
        padding: '16px 18px',
        borderRadius: 'var(--radius-lg)',
        background: 'var(--bg-surface)',
        border: '1px solid var(--glass-border)',
        display: 'flex',
        flexDirection: 'column',
        gap: 8,
      }}
    >
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
        <div>
          <div style={{ fontSize: 11, color: 'var(--text-secondary)', fontWeight: 500, marginBottom: 4 }}>{label}</div>
          <div style={{ fontSize: 26, fontWeight: 700, color, lineHeight: 1 }}>
            {value.toFixed(1)}<span style={{ fontSize: 13, fontWeight: 400, marginLeft: 2 }}>{unit}</span>
          </div>
        </div>
        <Sparkline history={history} color={color} />
      </div>
      {/* Progress bar */}
      <div style={{ height: 4, borderRadius: 2, background: 'var(--border-subtle)' }}>
        <motion.div
          animate={{ width: `${Math.min(100, value)}%` }}
          transition={{ duration: 0.4 }}
          style={{ height: '100%', borderRadius: 2, background: color, transition: 'background 0.3s' }}
        />
      </div>
    </motion.div>
  )
}

function NetSection({ netUp, netDown }: { netUp: number; netDown: number }) {
  const fmt = (v: number) => v >= 1024 ? `${(v / 1024).toFixed(1)} MB/s` : `${v.toFixed(0)} KB/s`
  return (
    <motion.div
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      style={{
        padding: '16px 20px',
        borderRadius: 'var(--radius-lg)',
        background: 'var(--bg-surface)',
        border: '1px solid var(--glass-border)',
        display: 'flex',
        gap: 40,
      }}
    >
      <div>
        <div style={{ fontSize: 11, color: 'var(--text-secondary)', marginBottom: 4 }}>▲ 업로드</div>
        <div style={{ fontSize: 22, fontWeight: 700, color: '#60a5fa' }}>{fmt(netUp)}</div>
      </div>
      <div>
        <div style={{ fontSize: 11, color: 'var(--text-secondary)', marginBottom: 4 }}>▼ 다운로드</div>
        <div style={{ fontSize: 22, fontWeight: 700, color: '#a78bfa' }}>{fmt(netDown)}</div>
      </div>
    </motion.div>
  )
}

export function MonitorView() {
  const { monitorHistory, startMonitoring } = useAppStore()

  useEffect(() => {
    const cleanup = startMonitoring()
    return cleanup
  }, [startMonitoring])

  const latest: SystemStats = monitorHistory[monitorHistory.length - 1] ?? {
    cpu: 0, mem: 0, disk: 0, cpuTemp: 0, netUp: 0, netDown: 0, timestamp: 0,
  }

  const pick = (key: keyof SystemStats) =>
    monitorHistory.map((s) => s[key] as number)

  return (
    <div style={{
      flex: 1, overflowY: 'auto', padding: 24,
      background: 'var(--bg-base)', color: 'var(--text-primary)',
    }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 20 }}>
        <h2 style={{ margin: 0, fontSize: 20, fontWeight: 700 }}>📊 실시간 시스템 모니터</h2>
        {/* Pulsing badge */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 6, padding: '4px 10px', borderRadius: 20, background: 'rgba(34,197,94,0.1)', border: '1px solid rgba(34,197,94,0.3)' }}>
          <motion.div
            animate={{ opacity: [1, 0.3, 1] }}
            transition={{ duration: 1.2, repeat: Infinity }}
            style={{ width: 6, height: 6, borderRadius: '50%', background: 'var(--success)' }}
          />
          <span style={{ fontSize: 11, color: 'var(--success)', fontWeight: 600 }}>실시간 업데이트 중</span>
        </div>
      </div>

      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16, marginBottom: 16 }}>
        <StatCard label="CPU 사용률"    value={latest.cpu}     unit="%"  history={pick('cpu')} />
        <StatCard label="메모리 사용률" value={latest.mem}     unit="%"  history={pick('mem')} />
        <StatCard label="디스크 사용률" value={latest.disk}    unit="%"  history={pick('disk')} />
        <StatCard label="CPU 온도"      value={latest.cpuTemp} unit="°C" history={pick('cpuTemp')} />
      </div>

      <NetSection netUp={latest.netUp} netDown={latest.netDown} />

      <div style={{ marginTop: 10, fontSize: 11, color: 'var(--text-muted)' }}>
        2초마다 자동 갱신 · {monitorHistory.length}/30 데이터 포인트
      </div>
    </div>
  )
}
