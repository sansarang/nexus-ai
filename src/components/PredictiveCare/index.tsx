import { useState } from 'react'
import { motion } from 'framer-motion'
import { useAppStore } from '../../stores/appStore'

interface Prediction {
  type: string
  probability: number
  timeFrame: string
  advice: string
  icon: string
  autoAction: string
}

function generatePredictions(): Prediction[] {
  const base = Math.random() * 0.2
  return [
    {
      type: 'cpu_overheat',
      probability: 0.65 + base,
      timeFrame: '3일 후',
      advice: 'CPU 온도가 꾸준히 오르고 있어요. 냉각 팬 먼지 청소를 권장해요.',
      icon: '🌡️',
      autoAction: 'autoclean',
    },
    {
      type: 'disk_full',
      probability: 0.48 + base,
      timeFrame: '11일 후',
      advice: '저장공간이 빠르게 줄고 있어요. 미리 정리해두세요.',
      icon: '💾',
      autoAction: 'autoclean',
    },
    {
      type: 'ram_ok',
      probability: 0.08,
      timeFrame: '안전',
      advice: '메모리 상태가 양호해요. 현재 패턴이 지속될 것으로 예측돼요.',
      icon: '🟢',
      autoAction: '',
    },
    {
      type: 'driver_ok',
      probability: 0.05,
      timeFrame: '안전',
      advice: '드라이버가 최신 상태예요. 안정적으로 동작하고 있어요.',
      icon: '✅',
      autoAction: '',
    },
  ]
}

function ProbBar({ prob, color }: { prob: number; color: string }) {
  return (
    <div
      style={{
        width: '100%',
        height: 4,
        background: 'var(--border-subtle)',
        borderRadius: 2,
        overflow: 'hidden',
        marginTop: 8,
      }}
    >
      <motion.div
        initial={{ width: 0 }}
        animate={{ width: `${prob * 100}%` }}
        transition={{ duration: 0.8, ease: 'easeOut' }}
        style={{ height: '100%', background: color, borderRadius: 2 }}
      />
    </div>
  )
}

export function PredictiveCareView() {
  const { setView } = useAppStore()
  const [predictions, setPredictions] = useState<Prediction[]>(generatePredictions)

  return (
    <div style={{ flex: 1, overflowY: 'auto', padding: 24 }}>
      {/* Header */}
      <div style={{ marginBottom: 8 }}>
        <div style={{ fontSize: 20, fontWeight: 700, color: 'var(--text-primary)', marginBottom: 4 }}>
          🔮 AI 예측 관리
        </div>
        <div style={{ fontSize: 13, color: 'var(--text-secondary)', marginBottom: 16 }}>
          내 PC의 미래를 미리 알려드려요 — 문제가 생기기 전에 예방해요
        </div>
      </div>

      <div style={{ display: 'flex', flexDirection: 'column', gap: 12, marginBottom: 20 }}>
        {predictions.map((pred, i) => {
          const pct = Math.round(pred.probability * 100)
          const color =
            pred.probability > 0.6
              ? 'var(--danger)'
              : pred.probability > 0.3
              ? 'var(--warning)'
              : 'var(--success)'
          const borderColor =
            pred.probability > 0.6
              ? 'rgba(239,68,68,0.25)'
              : pred.probability > 0.3
              ? 'rgba(245,158,11,0.25)'
              : 'rgba(34,197,94,0.2)'

          return (
            <motion.div
              key={pred.type}
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: i * 0.07 }}
              style={{
                background: 'var(--glass-bg)',
                border: `1px solid ${borderColor}`,
                borderRadius: 'var(--radius-md)',
                padding: '16px 18px',
              }}
            >
              <div style={{ display: 'flex', alignItems: 'flex-start', gap: 12 }}>
                <span style={{ fontSize: 28, flexShrink: 0 }}>{pred.icon}</span>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4 }}>
                    <span style={{ fontSize: 14, fontWeight: 600, color: 'var(--text-primary)' }}>
                      {pred.type === 'cpu_overheat'
                        ? 'CPU 과열'
                        : pred.type === 'disk_full'
                        ? '디스크 공간 부족'
                        : pred.type === 'ram_ok'
                        ? '메모리 정상'
                        : '드라이버 정상'}
                    </span>
                    <span
                      style={{
                        fontSize: 11,
                        padding: '2px 8px',
                        background:
                          pred.probability > 0.3 ? `${color}22` : 'rgba(34,197,94,0.1)',
                        color,
                        borderRadius: 10,
                      }}
                    >
                      {pred.timeFrame}
                    </span>
                  </div>
                  <div style={{ fontSize: 12, color: 'var(--text-secondary)', marginBottom: 6 }}>
                    {pred.advice}
                  </div>
                  {pred.probability > 0.1 && (
                    <>
                      <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>
                        발생 가능성 {pct}%
                      </div>
                      <ProbBar prob={pred.probability} color={color} />
                    </>
                  )}
                </div>
                {pred.autoAction && (
                  <motion.button
                    whileHover={{ scale: 1.05 }}
                    whileTap={{ scale: 0.95 }}
                    onClick={() => setView(pred.autoAction as Parameters<typeof setView>[0])}
                    style={{
                      padding: '6px 12px',
                      background: color,
                      border: 'none',
                      borderRadius: 'var(--radius-sm)',
                      color: 'white',
                      fontSize: 11,
                      cursor: 'pointer',
                      flexShrink: 0,
                    }}
                  >
                    대비하기
                  </motion.button>
                )}
              </div>
            </motion.div>
          )
        })}
      </div>

      <motion.button
        whileHover={{ scale: 1.02 }}
        whileTap={{ scale: 0.97 }}
        onClick={() => setPredictions(generatePredictions())}
        style={{
          width: '100%',
          padding: '10px 0',
          background: 'var(--glass-bg)',
          border: '1px solid var(--border-default)',
          borderRadius: 'var(--radius-md)',
          color: 'var(--text-secondary)',
          fontSize: 13,
          cursor: 'pointer',
        }}
      >
        🔄 예측 새로고침
      </motion.button>
    </div>
  )
}
