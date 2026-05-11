import React, { useState, useEffect, useRef, useCallback } from 'react'
import { motion, AnimatePresence } from 'framer-motion'

const API = 'http://127.0.0.1:17891'

interface AgentStep {
  text: string
  status: 'done' | 'running' | 'pending'
  timestamp: string
}

interface ApprovalRequest {
  task_id: string
  action: string
  reason: string
}

interface DesktopAgentProps {
  onClose: () => void
  primaryColor?: string
}

export function DesktopAgent({ onClose, primaryColor = '#7c3aed' }: DesktopAgentProps) {
  const [goal, setGoal] = useState('')
  const [taskId, setTaskId] = useState<string | null>(null)
  const [status, setStatus] = useState<'idle' | 'running' | 'done' | 'failed' | 'cancelled'>('idle')
  const [progress, setProgress] = useState(0)
  const [message, setMessage] = useState('')
  const [steps, setSteps] = useState<AgentStep[]>([])
  const [screenshot, setScreenshot] = useState<string | null>(null)
  const [approval, setApproval] = useState<ApprovalRequest | null>(null)
  const [requireApproval, setRequireApproval] = useState(true)
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const ssRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const stopPolling = useCallback(() => {
    if (pollRef.current) { clearInterval(pollRef.current); pollRef.current = null }
    if (ssRef.current) { clearInterval(ssRef.current); ssRef.current = null }
  }, [])

  // Poll task status
  useEffect(() => {
    if (!taskId || status === 'idle') return
    pollRef.current = setInterval(async () => {
      try {
        const res = await fetch(`${API}/api/tasks/list`)
        const data = await res.json()
        const tasks: any[] = data.tasks || []
        const task = tasks.find((t: any) => t.id === taskId)
        if (!task) return
        setProgress(task.progress ?? 0)
        setMessage(task.message ?? '')
        if (task.message && task.message !== steps[steps.length - 1]?.text) {
          setSteps(prev => [...prev, {
            text: task.message,
            status: task.status === 'done' ? 'done' : 'running',
            timestamp: new Date().toLocaleTimeString(),
          }])
        }
        if (task.status === 'done' || task.status === 'failed' || task.status === 'cancelled') {
          setStatus(task.status)
          stopPolling()
        }
      } catch { /* ignore */ }
    }, 1000)
    return stopPolling
  }, [taskId, status, stopPolling])

  // Poll screenshot every 2s while running
  useEffect(() => {
    if (status !== 'running') return
    const fetchSS = async () => {
      try {
        const res = await fetch(`${API}/api/desktop/screenshot`)
        const data = await res.json()
        if (data.base64) setScreenshot(data.base64)
      } catch { /* ignore */ }
    }
    fetchSS()
    ssRef.current = setInterval(fetchSS, 2000)
    return () => { if (ssRef.current) clearInterval(ssRef.current) }
  }, [status])

  // Listen for approval alerts via SSE
  useEffect(() => {
    const es = new EventSource(`${API}/api/alerts/stream`)
    es.onmessage = (e) => {
      try {
        const alert = JSON.parse(e.data)
        if (alert.action?.startsWith('approve:') && taskId) {
          const alertTaskId = alert.action.replace('approve:', '')
          if (alertTaskId === taskId) {
            setApproval({ task_id: alertTaskId, action: '', reason: alert.message })
          }
        }
      } catch { /* ignore */ }
    }
    return () => es.close()
  }, [taskId])

  const handleRun = async () => {
    if (!goal.trim()) return
    setStatus('running')
    setSteps([{ text: '작업 시작 중...', status: 'running', timestamp: new Date().toLocaleTimeString() }])
    setProgress(0)
    setScreenshot(null)
    try {
      const res = await fetch(`${API}/api/desktop/agent/run`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ goal, require_approval: requireApproval, max_steps: 20 }),
      })
      const data = await res.json()
      if (data.success && data.task_id) {
        setTaskId(data.task_id)
      } else {
        setStatus('failed')
        setMessage(data.message || '실행 실패')
      }
    } catch (err) {
      setStatus('failed')
      setMessage('백엔드 연결 실패')
    }
  }

  const handleCancel = async () => {
    if (!taskId) return
    try {
      await fetch(`${API}/api/desktop/agent/cancel`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ task_id: taskId }),
      })
    } catch { /* ignore */ }
    setStatus('cancelled')
    stopPolling()
  }

  const handleApprove = async (approved: boolean) => {
    if (!approval) return
    try {
      await fetch(`${API}/api/desktop/approve`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ task_id: approval.task_id, approved }),
      })
    } catch { /* ignore */ }
    setApproval(null)
  }

  const reset = () => {
    stopPolling()
    setStatus('idle')
    setTaskId(null)
    setProgress(0)
    setMessage('')
    setSteps([])
    setScreenshot(null)
    setApproval(null)
  }

  const statusColor = {
    idle: '#6b7280', running: primaryColor, done: '#22c55e', failed: '#ef4444', cancelled: '#f59e0b',
  }[status]

  const statusLabel = {
    idle: '대기', running: '실행 중', done: '완료', failed: '실패', cancelled: '취소됨',
  }[status]

  return (
    <motion.div
      initial={{ opacity: 0, y: 20, scale: 0.96 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      exit={{ opacity: 0, y: 20, scale: 0.96 }}
      style={{
        position: 'fixed', bottom: 100, right: 90, width: 480,
        background: 'rgba(6,6,18,0.97)', backdropFilter: 'blur(20px)',
        border: `1px solid ${primaryColor}44`, borderRadius: 20,
        boxShadow: `0 24px 64px rgba(0,0,0,0.7), 0 0 0 1px ${primaryColor}22`,
        zIndex: 10005, overflow: 'hidden', fontFamily: 'inherit',
      }}
    >
      {/* Header */}
      <div style={{
        padding: '14px 18px', display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        borderBottom: `1px solid rgba(255,255,255,0.07)`,
        background: `linear-gradient(135deg, ${primaryColor}18, transparent)`,
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          <span style={{ fontSize: 18 }}>🖥️</span>
          <span style={{ fontWeight: 700, fontSize: 14, color: '#fff' }}>Desktop Agent</span>
          <span style={{
            fontSize: 10, fontWeight: 600, padding: '2px 8px', borderRadius: 20,
            background: `${statusColor}22`, color: statusColor, border: `1px solid ${statusColor}44`,
          }}>{statusLabel}</span>
        </div>
        <button onClick={onClose} style={{
          background: 'none', border: 'none', color: 'rgba(255,255,255,0.4)',
          cursor: 'pointer', fontSize: 16, lineHeight: 1,
        }}>✕</button>
      </div>

      <div style={{ padding: 18, display: 'flex', flexDirection: 'column', gap: 14 }}>
        {/* Goal input */}
        <div>
          <textarea
            value={goal}
            onChange={e => setGoal(e.target.value)}
            placeholder="수행할 작업을 입력하세요 (예: Chrome을 열고 네이버 검색 후 결과 캡처)"
            disabled={status === 'running'}
            rows={2}
            style={{
              width: '100%', background: 'rgba(255,255,255,0.04)',
              border: `1px solid ${goal ? primaryColor + '66' : 'rgba(255,255,255,0.1)'}`,
              borderRadius: 12, padding: '10px 14px', color: '#fff', fontSize: 13,
              resize: 'none', outline: 'none', fontFamily: 'inherit',
              opacity: status === 'running' ? 0.6 : 1, boxSizing: 'border-box',
            }}
          />
        </div>

        {/* Options + Buttons */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          <label style={{ display: 'flex', alignItems: 'center', gap: 6, cursor: 'pointer', flex: 1 }}>
            <input
              type="checkbox" checked={requireApproval}
              onChange={e => setRequireApproval(e.target.checked)}
              style={{ accentColor: primaryColor }}
            />
            <span style={{ fontSize: 11, color: 'rgba(255,255,255,0.5)' }}>위험 작업 승인 필요</span>
          </label>
          {status === 'idle' || status === 'done' || status === 'failed' || status === 'cancelled' ? (
            <>
              {status !== 'idle' && (
                <button onClick={reset} style={{
                  padding: '7px 14px', borderRadius: 10, border: '1px solid rgba(255,255,255,0.15)',
                  background: 'rgba(255,255,255,0.05)', color: 'rgba(255,255,255,0.6)',
                  fontSize: 12, cursor: 'pointer',
                }}>초기화</button>
              )}
              <button onClick={handleRun} disabled={!goal.trim()} style={{
                padding: '7px 18px', borderRadius: 10, border: 'none',
                background: goal.trim() ? `linear-gradient(135deg, ${primaryColor}, ${primaryColor}99)` : 'rgba(255,255,255,0.1)',
                color: goal.trim() ? '#fff' : 'rgba(255,255,255,0.3)',
                fontSize: 12, fontWeight: 700, cursor: goal.trim() ? 'pointer' : 'not-allowed',
              }}>▶ 실행</button>
            </>
          ) : (
            <button onClick={handleCancel} style={{
              padding: '7px 18px', borderRadius: 10, border: '1px solid #ef444444',
              background: 'rgba(239,68,68,0.1)', color: '#ef4444',
              fontSize: 12, fontWeight: 700, cursor: 'pointer',
            }}>⏹ 취소</button>
          )}
        </div>

        {/* Progress bar */}
        {status === 'running' && (
          <div style={{ background: 'rgba(255,255,255,0.06)', borderRadius: 6, height: 4, overflow: 'hidden' }}>
            <motion.div
              animate={{ width: `${progress}%` }}
              transition={{ duration: 0.3 }}
              style={{ height: '100%', background: `linear-gradient(90deg, ${primaryColor}, ${primaryColor}88)`, borderRadius: 6 }}
            />
          </div>
        )}

        {/* Screenshot */}
        {screenshot && (
          <div style={{
            borderRadius: 12, overflow: 'hidden', border: `1px solid rgba(255,255,255,0.08)`,
            position: 'relative',
          }}>
            <img
              src={`data:image/png;base64,${screenshot}`}
              alt="현재 화면"
              style={{ width: '100%', display: 'block', maxHeight: 200, objectFit: 'contain', background: '#000' }}
            />
            {status === 'running' && (
              <div style={{
                position: 'absolute', top: 6, right: 6,
                background: `${primaryColor}cc`, borderRadius: 6,
                padding: '2px 8px', fontSize: 10, color: '#fff', fontWeight: 600,
                display: 'flex', alignItems: 'center', gap: 4,
              }}>
                <motion.span animate={{ opacity: [1, 0.3, 1] }} transition={{ duration: 1, repeat: Infinity }}>●</motion.span>
                실시간
              </div>
            )}
          </div>
        )}

        {/* Step log */}
        {steps.length > 0 && (
          <div style={{
            background: 'rgba(0,0,0,0.3)', borderRadius: 12, padding: '10px 14px',
            maxHeight: 160, overflowY: 'auto', display: 'flex', flexDirection: 'column', gap: 6,
          }}>
            {steps.map((step, i) => (
              <div key={i} style={{ display: 'flex', alignItems: 'flex-start', gap: 8 }}>
                <span style={{ fontSize: 10, color: 'rgba(255,255,255,0.3)', minWidth: 50, paddingTop: 1 }}>
                  {step.timestamp}
                </span>
                <span style={{
                  fontSize: 12,
                  color: step.status === 'done' ? '#22c55e' : step.status === 'running' ? primaryColor : 'rgba(255,255,255,0.5)',
                }}>
                  {step.status === 'running' ? '⟳ ' : step.status === 'done' ? '✓ ' : '○ '}
                  {step.text}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Approval modal */}
      <AnimatePresence>
        {approval && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            style={{
              position: 'absolute', inset: 0,
              background: 'rgba(0,0,0,0.8)', backdropFilter: 'blur(8px)',
              display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 10,
            }}
          >
            <motion.div
              initial={{ scale: 0.9, y: 10 }}
              animate={{ scale: 1, y: 0 }}
              style={{
                background: 'rgba(20,20,40,0.98)', borderRadius: 16, padding: 24,
                border: `1px solid ${primaryColor}66`, maxWidth: 340, width: '90%',
              }}
            >
              <div style={{ fontSize: 24, textAlign: 'center', marginBottom: 12 }}>✋</div>
              <div style={{ fontSize: 14, fontWeight: 700, color: '#fff', marginBottom: 8, textAlign: 'center' }}>
                작업 승인 요청
              </div>
              <div style={{
                fontSize: 12, color: 'rgba(255,255,255,0.65)', textAlign: 'center',
                marginBottom: 20, lineHeight: 1.6,
              }}>
                {approval.reason}
              </div>
              <div style={{ display: 'flex', gap: 10 }}>
                <button onClick={() => handleApprove(false)} style={{
                  flex: 1, padding: '10px', borderRadius: 10,
                  border: '1px solid rgba(239,68,68,0.4)', background: 'rgba(239,68,68,0.1)',
                  color: '#ef4444', fontSize: 13, fontWeight: 600, cursor: 'pointer',
                }}>거부</button>
                <button onClick={() => handleApprove(true)} style={{
                  flex: 1, padding: '10px', borderRadius: 10,
                  border: 'none', background: `linear-gradient(135deg, ${primaryColor}, ${primaryColor}99)`,
                  color: '#fff', fontSize: 13, fontWeight: 600, cursor: 'pointer',
                }}>허용</button>
              </div>
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>
    </motion.div>
  )
}

export default DesktopAgent
