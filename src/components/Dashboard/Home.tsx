import { useState, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { RefreshCw, Wrench, Zap } from 'lucide-react'
import { useAppStore } from '../../stores/appStore'
import { ScoreGauge } from '../ScoreGauge'
import { RepairSuccess } from '../RepairSuccess'

/* 지표 카드 */
function MetricCard({
  label,
  value,
  status,
}: {
  label: string
  value: number
  status: 'good' | 'warning' | 'danger'
}) {
  const color =
    status === 'good' ? 'var(--success)' :
    status === 'warning' ? 'var(--warning)' :
                            'var(--danger)'
  const emoji = status === 'good' ? '🟢' : status === 'warning' ? '🟡' : '🔴'
  const statusLabel = status === 'good' ? '정상' : status === 'warning' ? '보통' : '부족'

  return (
    <motion.div
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      style={{
        flex: 1,
        background: 'var(--bg-elevated)',
        border: '1px solid var(--border-subtle)',
        borderRadius: 'var(--radius-md)',
        padding: '14px 16px',
        display: 'flex',
        flexDirection: 'column',
        gap: 8,
      }}
    >
      <span style={{ fontSize: 11, color: 'var(--text-muted)', fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.05em' }}>
        {label}
      </span>
      <span style={{ fontSize: 28, fontWeight: 800, color: 'var(--text-primary)', fontVariantNumeric: 'tabular-nums', lineHeight: 1 }}>
        {value}
        <span style={{ fontSize: 14, fontWeight: 400, color: 'var(--text-secondary)' }}>%</span>
      </span>
      {/* 바 */}
      <div style={{ height: 3, borderRadius: 2, background: 'var(--border-subtle)', overflow: 'hidden' }}>
        <motion.div
          initial={{ width: 0 }}
          animate={{ width: `${value}%` }}
          transition={{ duration: 1, ease: 'easeOut' }}
          style={{ height: '100%', background: color, borderRadius: 2 }}
        />
      </div>
      <span style={{ fontSize: 12, color, display: 'flex', alignItems: 'center', gap: 4 }}>
        {emoji} {statusLabel}
      </span>
    </motion.div>
  )
}

/* 문제 항목 */
function IssueItem({
  issue,
  onRepair,
}: {
  issue: ReturnType<typeof useAppStore.getState>['issues'][0]
  onRepair: (id: string) => Promise<void>
}) {
  const [state, setState] = useState<'idle' | 'loading' | 'done'>('idle')

  const severityColor =
    issue.severity === 'critical' || issue.severity === 'high' ? 'var(--danger)' :
    issue.severity === 'medium' ? 'var(--warning)' : 'var(--success)'
  const severityBg =
    issue.severity === 'critical' || issue.severity === 'high' ? 'rgba(239,68,68,0.12)' :
    issue.severity === 'medium' ? 'rgba(245,158,11,0.12)' : 'rgba(34,197,94,0.12)'
  const severityLabel =
    issue.severity === 'critical' ? '위험' :
    issue.severity === 'high' ? '높음' :
    issue.severity === 'medium' ? '보통' : '낮음'

  const handleRepair = async () => {
    setState('loading')
    await onRepair(issue.id)
    setState('done')
  }

  return (
    <AnimatePresence>
      {state !== 'done' && (
        <motion.div
          layout
          exit={{ opacity: 0, height: 0, marginBottom: 0 }}
          transition={{ duration: 0.25 }}
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
          <div
            style={{
              width: 36,
              height: 36,
              borderRadius: 8,
              background: severityBg,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              fontSize: 18,
              flexShrink: 0,
            }}
          >
            {issue.severity === 'high' || issue.severity === 'critical' ? '🔴' :
             issue.severity === 'medium' ? '🟡' : '🟢'}
          </div>

          <div style={{ flex: 1, minWidth: 0 }}>
            <p style={{ fontSize: 13, fontWeight: 600, color: 'var(--text-primary)', marginBottom: 2 }}>
              {issue.title}
            </p>
            <p style={{ fontSize: 12, color: 'var(--text-secondary)', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
              {issue.description}
            </p>
          </div>

          {/* 심각도 뱃지 */}
          <span
            style={{
              padding: '2px 8px',
              borderRadius: 6,
              fontSize: 11,
              fontWeight: 600,
              background: severityBg,
              color: severityColor,
              flexShrink: 0,
            }}
          >
            {severityLabel}
          </span>

          {/* 수리 버튼 */}
          {issue.fixable && (
            <motion.button
              whileTap={{ scale: 0.95 }}
              onClick={handleRepair}
              disabled={state === 'loading'}
              style={{
                padding: '6px 14px',
                borderRadius: 8,
                border: 'none',
                background: state === 'loading' ? 'rgba(79,126,247,0.1)' : 'rgba(79,126,247,0.15)',
                color: 'var(--accent-primary)',
                fontSize: 12,
                fontWeight: 600,
                cursor: state === 'loading' ? 'default' : 'pointer',
                display: 'flex',
                alignItems: 'center',
                gap: 5,
                flexShrink: 0,
              }}
            >
              {state === 'loading' ? (
                <>
                  <span className="spin" style={{ width: 12, height: 12, border: '2px solid rgba(79,126,247,0.3)', borderTopColor: 'var(--accent-primary)', borderRadius: '50%', display: 'inline-block' }} />
                  수리 중
                </>
              ) : (
                <>
                  <Wrench size={12} />
                  수리
                </>
              )}
            </motion.button>
          )}
        </motion.div>
      )}
    </AnimatePresence>
  )
}

export function Home() {
  const { pcScore, issues, isScanning, cpuUsage, memUsage, diskUsage, startScan, repairIssue, repairAll } = useAppStore()
  const [repairResult, setRepairResult] = useState<{ before: number; after: number } | null>(null)
  const [repairing, setRepairing] = useState(false)

  /* 초기 로드 시 자동 진단 (마운트 1회만 실행) */
  useEffect(() => {
    if (pcScore === 0) startScan()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  /* 2초마다 지표 갱신 시뮬레이션 */
  useEffect(() => {
    const interval = setInterval(() => {
      /* 실제 환경에선 백엔드 호출 */
    }, 2000)
    return () => clearInterval(interval)
  }, [])

  const handleRepairAll = async () => {
    setRepairing(true)
    const result = await repairAll()
    setRepairing(false)
    setRepairResult(result)
  }

  const fixableCount = issues.filter((i) => i.fixable).length

  const cpuStatus = cpuUsage < 50 ? 'good' : cpuUsage < 80 ? 'warning' : 'danger'
  const memStatus = memUsage < 60 ? 'good' : memUsage < 85 ? 'warning' : 'danger'
  const diskStatus = diskUsage < 70 ? 'good' : diskUsage < 90 ? 'warning' : 'danger'

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

      <div
        style={{
          flex: 1,
          overflowY: 'auto',
          padding: '20px 24px',
          display: 'flex',
          flexDirection: 'column',
          gap: 20,
        }}
      >
        {/* 상단: 점수 게이지 + 지표 카드 */}
        <motion.div
          initial={{ opacity: 0, y: 8 }}
          animate={{ opacity: 1, y: 0 }}
          style={{
            display: 'flex',
            gap: 20,
            alignItems: 'stretch',
          }}
        >
          {/* 점수 게이지 카드 */}
          <div
            style={{
              background: 'var(--bg-elevated)',
              border: '1px solid var(--border-subtle)',
              borderRadius: 'var(--radius-lg)',
              padding: '20px 24px',
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              gap: 12,
              flexShrink: 0,
            }}
          >
            <span style={{ fontSize: 12, color: 'var(--text-muted)', fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.05em' }}>
              PC 건강 점수
            </span>

            {isScanning ? (
              <div
                style={{
                  width: 180,
                  height: 180,
                  display: 'flex',
                  flexDirection: 'column',
                  alignItems: 'center',
                  justifyContent: 'center',
                  gap: 12,
                }}
              >
                <span className="spin" style={{ width: 32, height: 32, border: '3px solid var(--border-default)', borderTopColor: 'var(--accent-primary)', borderRadius: '50%', display: 'block' }} />
                <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>진단 중...</span>
              </div>
            ) : (
              <ScoreGauge score={pcScore} />
            )}

            <div style={{ display: 'flex', gap: 8 }}>
              <motion.button
                whileTap={{ scale: 0.95 }}
                onClick={startScan}
                disabled={isScanning}
                style={{
                  padding: '7px 14px',
                  borderRadius: 8,
                  border: 'none',
                  background: 'rgba(79,126,247,0.15)',
                  color: 'var(--accent-primary)',
                  fontSize: 12,
                  fontWeight: 600,
                  cursor: isScanning ? 'default' : 'pointer',
                  display: 'flex',
                  alignItems: 'center',
                  gap: 5,
                  opacity: isScanning ? 0.6 : 1,
                }}
              >
                <RefreshCw size={12} className={isScanning ? 'spin' : ''} />
                재진단
              </motion.button>

              {fixableCount > 0 && (
                <motion.button
                  whileTap={{ scale: 0.95 }}
                  onClick={handleRepairAll}
                  disabled={repairing}
                  style={{
                    padding: '7px 14px',
                    borderRadius: 8,
                    border: 'none',
                    background: repairing ? 'rgba(34,197,94,0.1)' : 'linear-gradient(135deg, #22c55e, #16a34a)',
                    color: '#fff',
                    fontSize: 12,
                    fontWeight: 600,
                    cursor: repairing ? 'default' : 'pointer',
                    display: 'flex',
                    alignItems: 'center',
                    gap: 5,
                    boxShadow: repairing ? 'none' : '0 4px 12px rgba(34,197,94,0.3)',
                  }}
                >
                  {repairing ? (
                    <>
                      <span className="spin" style={{ width: 12, height: 12, border: '2px solid rgba(255,255,255,0.3)', borderTopColor: '#fff', borderRadius: '50%', display: 'inline-block' }} />
                      수리 중
                    </>
                  ) : (
                    <>
                      <Zap size={12} />
                      모두 수리 ({fixableCount})
                    </>
                  )}
                </motion.button>
              )}
            </div>
          </div>

          {/* 지표 카드 3개 (세로 스택) */}
          <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 10 }}>
            <MetricCard label="CPU" value={cpuUsage || 23} status={cpuStatus} />
            <MetricCard label="메모리" value={memUsage || 67} status={memStatus} />
            <MetricCard label="디스크" value={diskUsage || 82} status={diskStatus} />
          </div>
        </motion.div>

        {/* 발견된 문제 목록 */}
        <motion.div
          initial={{ opacity: 0, y: 8 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.08 }}
        >
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              marginBottom: 12,
            }}
          >
            <span style={{ fontSize: 12, fontWeight: 700, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
              발견된 문제 {issues.length}개
            </span>
            {fixableCount > 0 && (
              <button
                onClick={handleRepairAll}
                disabled={repairing}
                style={{
                  background: 'none',
                  border: 'none',
                  color: 'var(--accent-primary)',
                  fontSize: 12,
                  fontWeight: 600,
                  cursor: 'pointer',
                }}
              >
                모두 수리 →
              </button>
            )}
          </div>

          {issues.length > 0 ? (
            <div>
              {[...issues]
                .sort((a, b) => {
                  const o = { critical: 0, high: 1, medium: 2, low: 3 }
                  return (o[a.severity] ?? 4) - (o[b.severity] ?? 4)
                })
                .map((issue) => (
                  <IssueItem key={issue.id} issue={issue} onRepair={repairIssue} />
                ))}
            </div>
          ) : pcScore > 0 ? (
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              style={{
                padding: '32px',
                textAlign: 'center',
                borderRadius: 'var(--radius-md)',
                background: 'rgba(34,197,94,0.05)',
                border: '1px solid rgba(34,197,94,0.15)',
              }}
            >
              <div style={{ fontSize: 32, marginBottom: 8 }}>✅</div>
              <p style={{ color: 'var(--success)', fontWeight: 700, fontSize: 14 }}>문제 없음</p>
              <p style={{ color: 'var(--text-muted)', fontSize: 12, marginTop: 4 }}>PC 상태가 최상이에요</p>
            </motion.div>
          ) : (
            <div
              style={{
                padding: '32px',
                textAlign: 'center',
                borderRadius: 'var(--radius-md)',
                background: 'var(--bg-elevated)',
                border: '1px solid var(--border-subtle)',
              }}
            >
              <p style={{ color: 'var(--text-muted)', fontSize: 13 }}>재진단을 눌러 PC 상태를 확인하세요</p>
            </div>
          )}
        </motion.div>
      </div>
    </>
  )
}
