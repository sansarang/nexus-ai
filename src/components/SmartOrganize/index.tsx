import { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { useAppStore } from '../../stores/appStore'

interface FileRec {
  type: string
  icon: string
  desc: string
  savings: string
  action: string
}

const MOCK_RECS: FileRec[] = [
  {
    type: 'date_organize',
    icon: '📅',
    desc: '다운로드 폴더에 6개월 이상 된 파일이 많아요 → 날짜별로 정리하는 걸 추천해요',
    savings: '~2.1GB',
    action: '정리하기',
  },
  {
    type: 'remove_duplicates',
    icon: '🔁',
    desc: '중복 파일 47개가 1.8GB를 차지하고 있어요 → 삭제하면 공간이 넓어져요',
    savings: '~1.8GB',
    action: '삭제하기',
  },
  {
    type: 'review_big',
    icon: '📦',
    desc: '500MB 이상 파일이 12개 있어요 → 확인해보세요',
    savings: '',
    action: '확인하기',
  },
  {
    type: 'empty_folders',
    icon: '📁',
    desc: '빈 폴더 23개가 있어요 → 삭제할 수 있어요',
    savings: '',
    action: '삭제하기',
  },
]

type ItemState = 'idle' | 'running' | 'done'

export function SmartOrganizeView() {
  const { setView } = useAppStore()
  const [analyzed, setAnalyzed] = useState(false)
  const [analyzing, setAnalyzing] = useState(false)
  const [itemStates, setItemStates] = useState<Record<string, ItemState>>({})

  const analyze = async () => {
    setAnalyzing(true)
    await new Promise((r) => setTimeout(r, 1800))
    setAnalyzing(false)
    setAnalyzed(true)
  }

  const runAction = async (rec: FileRec) => {
    setItemStates((s) => ({ ...s, [rec.type]: 'running' }))
    await new Promise((r) => setTimeout(r, 1200))
    setItemStates((s) => ({ ...s, [rec.type]: 'done' }))
  }

  const runAll = async () => {
    for (const rec of MOCK_RECS) {
      await runAction(rec)
    }
  }

  return (
    <div style={{ flex: 1, overflowY: 'auto', padding: 24 }}>
      {/* Header */}
      <div style={{ marginBottom: 24 }}>
        <div style={{ fontSize: 20, fontWeight: 700, color: 'var(--text-primary)', marginBottom: 4 }}>
          📁 스마트 파일 정리
        </div>
        <div style={{ fontSize: 13, color: 'var(--text-secondary)' }}>
          AI가 파일 패턴을 분석해서 최적의 정리 방법을 추천해드려요
        </div>
      </div>

      {!analyzed && !analyzing && (
        <motion.button
          whileHover={{ scale: 1.02 }}
          whileTap={{ scale: 0.97 }}
          onClick={analyze}
          style={{
            width: '100%',
            padding: '48px 0',
            background: 'var(--glass-bg)',
            border: '2px dashed var(--border-default)',
            borderRadius: 'var(--radius-lg)',
            color: 'var(--text-secondary)',
            fontSize: 16,
            cursor: 'pointer',
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            gap: 12,
          }}
        >
          <span style={{ fontSize: 40 }}>🔍</span>
          <span>파일 시스템 분석 시작</span>
          <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>
            다운로드, 문서, 바탕화면 폴더를 스캔해요
          </span>
        </motion.button>
      )}

      {analyzing && (
        <div style={{ textAlign: 'center', padding: '48px 0' }}>
          <motion.div
            animate={{ rotate: 360 }}
            transition={{ duration: 1.5, repeat: Infinity, ease: 'linear' }}
            style={{ fontSize: 40, display: 'inline-block', marginBottom: 16 }}
          >
            🔍
          </motion.div>
          <div style={{ color: 'var(--text-secondary)', fontSize: 14 }}>파일 시스템 분석 중...</div>
        </div>
      )}

      <AnimatePresence>
        {analyzed && (
          <motion.div initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }}>
            <div style={{ marginBottom: 16, fontSize: 13, color: 'var(--text-secondary)' }}>
              💡 AI 추천 — 총 {MOCK_RECS.filter((r) => r.savings).length}가지 최적화 항목
            </div>

            <div style={{ display: 'flex', flexDirection: 'column', gap: 10, marginBottom: 20 }}>
              {MOCK_RECS.map((rec, i) => (
                <motion.div
                  key={rec.type}
                  initial={{ opacity: 0, x: -12 }}
                  animate={{ opacity: 1, x: 0 }}
                  transition={{ delay: i * 0.08 }}
                  style={{
                    background: 'var(--glass-bg)',
                    border: '1px solid var(--border-subtle)',
                    borderRadius: 'var(--radius-md)',
                    padding: '14px 16px',
                    display: 'flex',
                    alignItems: 'center',
                    gap: 14,
                  }}
                >
                  <span style={{ fontSize: 24, flexShrink: 0 }}>{rec.icon}</span>
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ fontSize: 13, color: 'var(--text-primary)', marginBottom: 2 }}>
                      {rec.desc}
                    </div>
                    {rec.savings && (
                      <span
                        style={{
                          fontSize: 11,
                          padding: '2px 8px',
                          background: 'rgba(79,126,247,0.15)',
                          color: 'var(--accent-primary)',
                          borderRadius: 10,
                        }}
                      >
                        {rec.savings} 절약 가능
                      </span>
                    )}
                  </div>
                  <motion.button
                    whileHover={{ scale: 1.03 }}
                    whileTap={{ scale: 0.97 }}
                    onClick={() => runAction(rec)}
                    disabled={itemStates[rec.type] === 'running' || itemStates[rec.type] === 'done'}
                    style={{
                      padding: '6px 14px',
                      background:
                        itemStates[rec.type] === 'done'
                          ? 'var(--success)'
                          : itemStates[rec.type] === 'running'
                          ? 'var(--glass-bg)'
                          : 'var(--accent-primary)',
                      border: 'none',
                      borderRadius: 'var(--radius-sm)',
                      color: 'white',
                      fontSize: 12,
                      cursor:
                        itemStates[rec.type] === 'done' || itemStates[rec.type] === 'running'
                          ? 'default'
                          : 'pointer',
                      flexShrink: 0,
                    }}
                  >
                    {itemStates[rec.type] === 'done'
                      ? '✅ 완료'
                      : itemStates[rec.type] === 'running'
                      ? '⏳...'
                      : rec.action}
                  </motion.button>
                </motion.div>
              ))}
            </div>

            <div style={{ display: 'flex', gap: 8 }}>
              <motion.button
                whileHover={{ scale: 1.02 }}
                whileTap={{ scale: 0.97 }}
                onClick={runAll}
                style={{
                  flex: 1,
                  padding: '12px 0',
                  background: 'var(--accent-primary)',
                  border: 'none',
                  borderRadius: 'var(--radius-md)',
                  color: 'white',
                  fontSize: 14,
                  fontWeight: 600,
                  cursor: 'pointer',
                }}
              >
                전체 추천대로 정리하기
              </motion.button>
              <motion.button
                whileHover={{ scale: 1.02 }}
                whileTap={{ scale: 0.97 }}
                onClick={() => setView('autoclean')}
                style={{
                  padding: '12px 20px',
                  background: 'var(--glass-bg)',
                  border: '1px solid var(--border-default)',
                  borderRadius: 'var(--radius-md)',
                  color: 'var(--text-secondary)',
                  fontSize: 14,
                  cursor: 'pointer',
                }}
              >
                PC 정리로 이동
              </motion.button>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}
