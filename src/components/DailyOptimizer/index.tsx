import { useEffect } from 'react'
import { motion } from 'framer-motion'
import { useAppStore } from '../../stores/appStore'
import { ScoreGauge } from '../ScoreGauge'

function MetricBar({ label, value, max, color }: { label: string; value: number; max: number; color: string }) {
  const pct = Math.min(100, (value / max) * 100)
  return (
    <div style={{ marginBottom: 12 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4, fontSize: 13 }}>
        <span style={{ color: 'var(--text-secondary)' }}>{label}</span>
        <span style={{ color: 'var(--text-primary)', fontWeight: 600 }}>{value.toFixed(1)}%</span>
      </div>
      <div style={{ height: 6, borderRadius: 3, background: 'rgba(255,255,255,0.08)' }}>
        <motion.div
          initial={{ width: 0 }}
          animate={{ width: `${pct}%` }}
          transition={{ duration: 0.8, ease: 'easeOut' }}
          style={{ height: '100%', borderRadius: 3, background: color }}
        />
      </div>
    </div>
  )
}

export function DailyView() {
  const { dailyReport, fetchDailyReport } = useAppStore()

  useEffect(() => {
    fetchDailyReport()
  }, [fetchDailyReport])

  const now = new Date()
  const dateStr = now.toLocaleDateString('ko-KR', { weekday: 'long', month: 'long', day: 'numeric' })
  const hour = now.getHours()
  const greeting = hour < 12 ? '좋은 아침이에요' : hour < 18 ? '좋은 오후예요' : '좋은 저녁이에요'

  const TOTAL_DISK_GB = 500

  return (
    <div style={{
      flex: 1, overflowY: 'auto', padding: 24,
      background: 'var(--bg-base)', color: 'var(--text-primary)',
    }}>
      <motion.div
        initial={{ opacity: 0, y: 12 }}
        animate={{ opacity: 1, y: 0 }}
        style={{ maxWidth: 640 }}
      >
        <div style={{ marginBottom: 24 }}>
          <h2 style={{ margin: '0 0 4px', fontSize: 22, fontWeight: 700 }}>☀️ {greeting}!</h2>
          <p style={{ margin: 0, fontSize: 13, color: 'var(--text-secondary)' }}>{dateStr}</p>
        </div>

        {!dailyReport ? (
          // Loading skeleton
          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            {[140, 80, 80, 60].map((h, i) => (
              <motion.div
                key={i}
                animate={{ opacity: [0.4, 0.7, 0.4] }}
                transition={{ duration: 1.4, repeat: Infinity, delay: i * 0.2 }}
                style={{ height: h, borderRadius: 'var(--radius-md)', background: 'var(--bg-surface)' }}
              />
            ))}
          </div>
        ) : (
          <>
            {/* PC Score + Metrics */}
            <motion.div
              initial={{ opacity: 0, scale: 0.95 }}
              animate={{ opacity: 1, scale: 1 }}
              style={{
                padding: '20px 24px',
                borderRadius: 'var(--radius-lg)',
                background: 'var(--bg-surface)',
                border: '1px solid var(--glass-border)',
                display: 'flex',
                alignItems: 'center',
                gap: 24,
                marginBottom: 16,
              }}
            >
              <div style={{ flexShrink: 0, display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 6 }}>
                <ScoreGauge score={dailyReport.pcScore} size={80} />
                <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>PC 건강 점수</div>
              </div>
              <div style={{ flex: 1 }}>
                <MetricBar label="CPU 평균" value={dailyReport.cpuAvg} max={100} color="#60a5fa" />
                <MetricBar label="메모리 평균" value={dailyReport.memAvg} max={100} color="#a78bfa" />
                <MetricBar
                  label="디스크 여유"
                  value={(dailyReport.diskFree / TOTAL_DISK_GB) * 100}
                  max={100}
                  color="#4ade80"
                />
              </div>
            </motion.div>

            {/* Recommendations */}
            <div style={{
              padding: '16px 20px', borderRadius: 'var(--radius-md)',
              background: 'var(--bg-surface)', border: '1px solid var(--glass-border)', marginBottom: 16,
            }}>
              <h3 style={{ margin: '0 0 12px', fontSize: 13, fontWeight: 600, color: 'var(--text-secondary)' }}>💡 권장사항</h3>
              <ul style={{ margin: 0, padding: 0, listStyle: 'none', display: 'flex', flexDirection: 'column', gap: 8 }}>
                {dailyReport.recommendations.map((rec, i) => (
                  <motion.li
                    key={i}
                    initial={{ opacity: 0, x: -8 }}
                    animate={{ opacity: 1, x: 0 }}
                    transition={{ delay: i * 0.06 }}
                    style={{ display: 'flex', gap: 10, fontSize: 13, color: 'var(--text-primary)' }}
                  >
                    <span style={{ color: 'var(--accent-primary)', flexShrink: 0 }}>💡</span>
                    {rec}
                  </motion.li>
                ))}
              </ul>
            </div>

            {/* Trend Predictions */}
            <h3 style={{ margin: '0 0 10px', fontSize: 13, fontWeight: 600, color: 'var(--text-secondary)' }}>📈 트렌드 예측</h3>
            <div style={{ display: 'flex', gap: 12, marginBottom: 20 }}>
              {dailyReport.predictions.map((p, i) => {
                const arrow = p.trend === 'up' ? '↑' : p.trend === 'down' ? '↓' : '→'
                const color = p.trend === 'up' ? 'var(--danger)' : p.trend === 'down' ? 'var(--success)' : 'var(--warning)'
                return (
                  <motion.div
                    key={i}
                    initial={{ opacity: 0, y: 8 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: i * 0.08 }}
                    style={{
                      flex: 1, padding: '14px 16px', borderRadius: 'var(--radius-md)',
                      background: 'var(--bg-elevated)', border: '1px solid var(--glass-border)',
                    }}
                  >
                    <div style={{ fontSize: 11, color: 'var(--text-muted)', marginBottom: 6 }}>{p.label}</div>
                    <div style={{ fontSize: 20, fontWeight: 700 }}>{p.value.toFixed(1)}%</div>
                    <div style={{ fontSize: 12, color, marginTop: 4, fontWeight: 600 }}>
                      {arrow} {p.trend === 'up' ? '증가' : p.trend === 'down' ? '감소' : '안정'}
                    </div>
                  </motion.div>
                )
              })}
            </div>

            <motion.button
              onClick={() => fetchDailyReport()}
              whileHover={{ scale: 1.02 }}
              whileTap={{ scale: 0.98 }}
              style={{
                padding: '10px 24px', borderRadius: 'var(--radius-md)',
                border: '1px solid var(--glass-border)', background: 'var(--glass-bg)',
                color: 'var(--text-primary)', cursor: 'pointer', fontSize: 13,
              }}
            >🔄 리포트 새로고침</motion.button>
          </>
        )}
      </motion.div>
    </div>
  )
}
