import { motion } from 'framer-motion'
import { useEffect } from 'react'
import confetti from 'canvas-confetti'

interface RepairSuccessProps {
  before: number
  after: number
  onDismiss: () => void
}

export function RepairSuccess({ before, after, onDismiss }: RepairSuccessProps) {
  /* 파티클 confetti */
  useEffect(() => {
    const fire = (angle: number, origin: { x: number; y: number }) => {
      confetti({
        angle,
        spread: 55,
        particleCount: 60,
        origin,
        colors: ['#22c55e', '#4f7ef7', '#f59e0b', '#f0f0ff'],
        scalar: 0.9,
      })
    }
    const t = setTimeout(() => {
      fire(60, { x: 0.2, y: 0.5 })
      fire(120, { x: 0.8, y: 0.5 })
    }, 400)

    /* 2초 후 자동 닫기 */
    const dismiss = setTimeout(onDismiss, 2800)
    return () => { clearTimeout(t); clearTimeout(dismiss) }
  }, [onDismiss])

  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      transition={{ duration: 0.2 }}
      onClick={onDismiss}
      style={{
        position: 'fixed',
        inset: 0,
        zIndex: 9999,
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        background: 'rgba(0, 0, 0, 0.85)',
        backdropFilter: 'blur(8px)',
        gap: 24,
        cursor: 'pointer',
      }}
    >
      {/* 성공 원 + 체크마크 */}
      <motion.div
        initial={{ scale: 0, opacity: 0 }}
        animate={{ scale: 1, opacity: 1 }}
        transition={{ type: 'spring', damping: 12, stiffness: 200 }}
        style={{
          width: 120,
          height: 120,
          borderRadius: '50%',
          background: 'rgba(34, 197, 94, 0.12)',
          border: '2px solid #22c55e',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          boxShadow: '0 0 48px rgba(34,197,94,0.35)',
        }}
      >
        <svg width="60" height="60" viewBox="0 0 60 60" fill="none">
          <motion.path
            d="M12 30 L25 43 L48 17"
            stroke="#22c55e"
            strokeWidth="4"
            strokeLinecap="round"
            strokeLinejoin="round"
            initial={{ pathLength: 0, opacity: 0 }}
            animate={{ pathLength: 1, opacity: 1 }}
            transition={{ delay: 0.3, duration: 0.5, ease: 'easeOut' }}
          />
        </svg>
      </motion.div>

      {/* 텍스트 */}
      <motion.div
        initial={{ opacity: 0, y: 16 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.55, duration: 0.3 }}
        style={{ textAlign: 'center' }}
      >
        <div
          style={{
            fontSize: 26,
            fontWeight: 800,
            color: 'var(--text-primary)',
            letterSpacing: '-0.02em',
          }}
        >
          수리 완료! 🎉
        </div>
        <div
          style={{
            fontSize: 16,
            color: 'var(--text-secondary)',
            marginTop: 10,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            gap: 10,
          }}
        >
          <span
            style={{
              padding: '3px 10px',
              borderRadius: 8,
              background: 'rgba(239,68,68,0.15)',
              color: '#ef4444',
              fontWeight: 700,
              fontSize: 18,
            }}
          >
            {before}
          </span>
          <span style={{ color: 'var(--text-muted)', fontSize: 14 }}>→</span>
          <span
            style={{
              padding: '3px 10px',
              borderRadius: 8,
              background: 'rgba(34,197,94,0.15)',
              color: '#22c55e',
              fontWeight: 700,
              fontSize: 18,
            }}
          >
            {after}
          </span>
          <span style={{ color: 'var(--text-muted)', fontSize: 13 }}>점</span>
        </div>
      </motion.div>

      <motion.span
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ delay: 1.2 }}
        style={{ fontSize: 12, color: 'var(--text-muted)' }}
      >
        클릭하면 닫힙니다
      </motion.span>
    </motion.div>
  )
}
