import { motion, useMotionValue, useTransform, animate } from 'framer-motion'
import { useEffect } from 'react'

interface ScoreGaugeProps {
  score: number
  size?: number
}

export function ScoreGauge({ score, size = 180 }: ScoreGaugeProps) {
  const radius = (size / 2) * 0.78
  const circumference = 2 * Math.PI * radius
  const cx = size / 2
  const cy = size / 2

  const color =
    score >= 71 ? '#22c55e' :
    score >= 41 ? '#f59e0b' :
                  '#ef4444'

  const label =
    score >= 71 ? '좋아요 👍' :
    score >= 41 ? '보통이에요' :
                  '위험해요 ⚠️'

  /* 숫자 카운트업 */
  const count = useMotionValue(0)
  const rounded = useTransform(count, (v) => Math.floor(v))

  useEffect(() => {
    const controls = animate(count, score, {
      duration: 1.5,
      ease: 'easeOut',
    })
    return controls.stop
  }, [score])

  const targetOffset = circumference - (score / 100) * circumference

  return (
    <div style={{ position: 'relative', width: size, height: size, flexShrink: 0 }}>
      <svg
        width={size}
        height={size}
        style={{ transform: 'rotate(-90deg)', display: 'block' }}
      >
        {/* 배경 트랙 */}
        <circle
          cx={cx} cy={cy} r={radius}
          fill="none"
          stroke="var(--border-subtle)"
          strokeWidth={8}
        />
        {/* 점수 arc */}
        <motion.circle
          cx={cx} cy={cy} r={radius}
          fill="none"
          stroke={color}
          strokeWidth={8}
          strokeLinecap="round"
          strokeDasharray={circumference}
          initial={{ strokeDashoffset: circumference }}
          animate={{ strokeDashoffset: targetOffset }}
          transition={{ duration: 1.5, ease: 'easeOut' }}
          style={{ filter: `drop-shadow(0 0 8px ${color})` }}
        />
      </svg>

      {/* 중앙 텍스트 */}
      <div
        style={{
          position: 'absolute',
          inset: 0,
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          gap: 2,
        }}
      >
        <div style={{ display: 'flex', alignItems: 'baseline', gap: 2 }}>
          <motion.span
            style={{
              fontSize: size * 0.22,
              fontWeight: 800,
              color: 'var(--text-primary)',
              fontVariantNumeric: 'tabular-nums',
              lineHeight: 1,
            }}
          >
            {rounded}
          </motion.span>
          <span style={{ fontSize: size * 0.08, color: 'var(--text-muted)', fontWeight: 500 }}>
            /100
          </span>
        </div>
        <span
          style={{
            fontSize: size * 0.075,
            color,
            fontWeight: 600,
            letterSpacing: '-0.01em',
          }}
        >
          {label}
        </span>
      </div>
    </div>
  )
}
