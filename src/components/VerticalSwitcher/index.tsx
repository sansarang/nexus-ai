import { useState, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Check, Palette, Play, Loader } from 'lucide-react'

const BASE = 'http://127.0.0.1:17891'

interface VerticalConfig {
  id: string
  name: string
  theme: string
  logo: string
  default_persona: string
  features: string[]
  welcome_msg: string
  watermark: string
}

interface WorkflowStep {
  label: string
  result: string
  ok: boolean
}

interface WorkflowResult {
  vertical_id: string
  name: string
  steps: WorkflowStep[]
  summary: string
  run_at: string
}

const VERTICAL_EMOJI: Record<string, string> = {
  general:    '🤖',
  legal:      '⚖️',
  medical:    '🏥',
  accountant: '📊',
  creator:    '🎬',
  realtor:    '🏠',
  teacher:    '📚',
  hr:         '👥',
  developer:  '💻',
  engineer:   '⚙️',
}

const VERTICAL_DESC: Record<string, string> = {
  general:    '범용 AI 비서',
  legal:      '판례·계약서 분석',
  medical:    '임상정보·진료 지원',
  accountant: '세무·재무제표 분석',
  creator:    '유튜브·틱톡 트렌드',
  realtor:    '시세·청약·계약 분석',
  teacher:    '강의안·수업 계획',
  hr:         '채용·이력서·노동법',
  developer:  'GitHub·코드 리뷰',
  engineer:   '규격·공정 최적화',
}

export function VerticalSwitcher() {
  const [presets, setPresets] = useState<VerticalConfig[]>([])
  const [activeId, setActiveId] = useState<string>('general')
  const [loading, setLoading] = useState(true)
  const [applying, setApplying] = useState<string | null>(null)
  const [workflowRunning, setWorkflowRunning] = useState(false)
  const [workflowResult, setWorkflowResult] = useState<WorkflowResult | null>(null)
  const [showWorkflow, setShowWorkflow] = useState(false)

  useEffect(() => {
    const stored = localStorage.getItem('nexus_vertical_id')
    if (stored) setActiveId(stored)

    Promise.all([
      fetch(`${BASE}/api/vertical/presets`).then(r => r.json()),
      fetch(`${BASE}/api/vertical/config`).then(r => r.json()),
    ]).then(([presetsRes, configRes]) => {
      setPresets(presetsRes.presets ?? [])
      if (configRes.config?.id) {
        setActiveId(configRes.config.id)
        localStorage.setItem('nexus_vertical_id', configRes.config.id)
      }
    }).catch(() => {}).finally(() => setLoading(false))
  }, [])

  const apply = async (preset: VerticalConfig) => {
    setApplying(preset.id)
    try {
      await fetch(`${BASE}/api/vertical/config`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(preset),
      })
      setActiveId(preset.id)
      localStorage.setItem('nexus_vertical_id', preset.id)
      document.documentElement.style.setProperty('--accent-primary', preset.theme)
      // 직업군 전환 시 워크플로 자동 초기화
      setWorkflowResult(null)
      setShowWorkflow(false)
    } catch {
      setActiveId(preset.id)
      localStorage.setItem('nexus_vertical_id', preset.id)
      document.documentElement.style.setProperty('--accent-primary', preset.theme)
    } finally {
      setApplying(null)
    }
  }

  const runWorkflow = async () => {
    setWorkflowRunning(true)
    setShowWorkflow(true)
    setWorkflowResult(null)
    try {
      const res = await fetch(`${BASE}/api/vertical/workflow/run`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ vertical_id: activeId }),
      })
      const data = await res.json()
      if (data.ok && data.result) setWorkflowResult(data.result)
    } catch {
      setWorkflowResult(null)
    } finally {
      setWorkflowRunning(false)
    }
  }

  const activePreset = presets.find(p => p.id === activeId)

  if (loading) {
    return <p style={{ fontSize: 12, color: 'var(--text-muted)', textAlign: 'center', padding: '20px 0' }}>불러오는 중...</p>
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      {/* Header */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        <Palette size={16} color="var(--accent-primary)" />
        <span style={{ fontSize: 14, fontWeight: 700, color: 'var(--text-primary)' }}>직업군 / 버티컬</span>
      </div>
      <p style={{ fontSize: 12, color: 'var(--text-muted)', marginTop: -8 }}>
        업종을 선택하면 AI 성격·자동 브리핑·검색 방식이 전부 바뀝니다.
      </p>

      {/* Preset Grid */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(140px, 1fr))', gap: 8 }}>
        {presets.map(preset => {
          const isActive = activeId === preset.id
          const isApplying = applying === preset.id
          return (
            <motion.button
              key={preset.id}
              whileTap={{ scale: 0.97 }}
              whileHover={{ scale: 1.02 }}
              onClick={() => !isApplying && apply(preset)}
              style={{
                background: isActive ? `${preset.theme}15` : 'var(--bg-elevated)',
                border: `2px solid ${isActive ? preset.theme : 'var(--border-subtle)'}`,
                borderRadius: 12,
                padding: '12px 10px',
                cursor: 'pointer',
                textAlign: 'left',
                position: 'relative',
                transition: 'all 0.2s',
              }}
            >
              {isActive && (
                <motion.div initial={{ scale: 0 }} animate={{ scale: 1 }} style={{
                  position: 'absolute', top: 7, right: 7,
                  width: 16, height: 16, borderRadius: '50%',
                  background: preset.theme,
                  display: 'flex', alignItems: 'center', justifyContent: 'center',
                }}>
                  <Check size={10} color="#fff" strokeWidth={3} />
                </motion.div>
              )}
              <div style={{
                width: 28, height: 28, borderRadius: 7,
                background: isActive ? preset.theme : `${preset.theme}33`,
                marginBottom: 8,
                display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 14,
              }}>
                {VERTICAL_EMOJI[preset.id] ?? '🤖'}
              </div>
              <p style={{ fontSize: 11, fontWeight: 700, color: 'var(--text-primary)', marginBottom: 2, lineHeight: 1.3 }}>
                {preset.name.replace('Nexus for ', '').replace('Nexus AI', '일반')}
              </p>
              <p style={{ fontSize: 10, color: 'var(--text-muted)', lineHeight: 1.3 }}>
                {VERTICAL_DESC[preset.id] ?? ''}
              </p>
              {isApplying && <p style={{ fontSize: 10, color: preset.theme, marginTop: 4 }}>적용 중...</p>}
            </motion.button>
          )
        })}
      </div>

      {/* 현재 직업군 + 워크플로 실행 버튼 */}
      {activePreset && (
        <div style={{
          background: 'var(--bg-elevated)',
          border: `1px solid ${activePreset.theme}44`,
          borderRadius: 12, padding: '12px 14px',
        }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 6 }}>
            <div>
              <p style={{ fontSize: 10, fontWeight: 700, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.06em', marginBottom: 2 }}>현재 활성</p>
              <p style={{ fontSize: 13, fontWeight: 700, color: activePreset.theme }}>
                {VERTICAL_EMOJI[activePreset.id]} {activePreset.name}
              </p>
            </div>
            <motion.button
              whileTap={{ scale: 0.95 }}
              onClick={runWorkflow}
              disabled={workflowRunning}
              style={{
                display: 'flex', alignItems: 'center', gap: 5,
                background: activePreset.theme,
                color: '#fff', border: 'none', borderRadius: 8,
                padding: '7px 12px', cursor: workflowRunning ? 'wait' : 'pointer',
                fontSize: 11, fontWeight: 700,
                boxShadow: `0 2px 8px ${activePreset.theme}55`,
                opacity: workflowRunning ? 0.7 : 1,
              }}
            >
              {workflowRunning
                ? <><Loader size={12} style={{ animation: 'spin 1s linear infinite' }} /> 실행 중...</>
                : <><Play size={12} /> 브리핑 실행</>
              }
            </motion.button>
          </div>
          <p style={{ fontSize: 11, color: 'var(--text-muted)' }}>{activePreset.welcome_msg}</p>
        </div>
      )}

      {/* 워크플로 결과 */}
      <AnimatePresence>
        {showWorkflow && (
          <motion.div
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -8 }}
            style={{
              background: 'var(--bg-elevated)',
              border: '1px solid var(--border-subtle)',
              borderRadius: 12, padding: '12px 14px',
            }}
          >
            <p style={{ fontSize: 11, fontWeight: 700, color: 'var(--text-muted)', marginBottom: 8 }}>
              📋 직업군 브리핑 결과
            </p>
            {workflowRunning && !workflowResult && (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                {[1, 2, 3].map(i => (
                  <div key={i} style={{
                    height: 32, borderRadius: 6,
                    background: 'rgba(255,255,255,0.06)',
                    animation: 'pulse 1.5s ease-in-out infinite',
                  }} />
                ))}
              </div>
            )}
            {workflowResult && (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
                {workflowResult.steps.map((step, i) => (
                  <div key={i} style={{
                    background: step.ok ? 'rgba(34,197,94,0.06)' : 'rgba(255,255,255,0.04)',
                    border: `1px solid ${step.ok ? 'rgba(34,197,94,0.2)' : 'rgba(255,255,255,0.08)'}`,
                    borderRadius: 8, padding: '8px 10px',
                  }}>
                    <p style={{ fontSize: 11, fontWeight: 700, color: step.ok ? '#22c55e' : 'rgba(255,255,255,0.5)', marginBottom: 4 }}>
                      {step.ok ? '✅' : '⚠️'} {step.label}
                    </p>
                    <p style={{ fontSize: 10, color: 'rgba(255,255,255,0.7)', whiteSpace: 'pre-line', lineHeight: 1.6 }}>
                      {step.result}
                    </p>
                  </div>
                ))}
                <p style={{ fontSize: 10, color: 'var(--text-muted)', marginTop: 4 }}>
                  🕐 실행 시각: {workflowResult.run_at}
                </p>
              </div>
            )}
          </motion.div>
        )}
      </AnimatePresence>

      <style>{`
        @keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
        @keyframes pulse { 0%, 100% { opacity: 0.4; } 50% { opacity: 0.8; } }
      `}</style>
    </div>
  )
}
