// AutomationPanel — 데스크탑 자동화 라이브러리/실행 셸 (UIA 엔진 상태 + 워크플로 재생)
// 엔진 미가용(Mac / UIA 미완성) 시 graceful degrade: 'Windows 전용' 안내 + 실행 버튼 비활성.
import { useEffect, useState, useRef } from 'react'
import {
  automationStatus, automationWorkflows, automationReplay,
  automationRecordStart, automationRecordStatus, automationRecordStop,
  type AutomationStatusResult,
} from '../../lib/nexus/backendAPI'

interface Props {
  open: boolean
  onClose: () => void
  lang?: 'ko' | 'en'
  primaryColor?: string
}

export function AutomationPanel({ open, onClose, lang = 'ko', primaryColor = '#4f6cf7' }: Props) {
  const isEn = lang === 'en'
  const [status, setStatus] = useState<AutomationStatusResult | null>(null)
  const [workflows, setWorkflows] = useState<string[]>([])
  const [loading, setLoading] = useState(false)
  const [toast, setToast] = useState('')
  const [recording, setRecording] = useState(false)
  const [recCount, setRecCount] = useState(0)
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const refreshWorkflows = () =>
    automationWorkflows().then((w) => setWorkflows(w.workflows ?? [])).catch(() => {})

  useEffect(() => {
    if (!open) return
    setLoading(true)
    setToast('')
    Promise.allSettled([automationStatus(), automationWorkflows()]).then(([s, w]) => {
      if (s.status === 'fulfilled') setStatus(s.value)
      if (w.status === 'fulfilled') setWorkflows(w.value.workflows ?? [])
      setLoading(false)
    })
  }, [open])

  // 패널 닫힐 때 폴링 정리
  useEffect(() => () => { if (pollRef.current) clearInterval(pollRef.current) }, [])

  if (!open) return null

  const available = status?.available ?? false

  const startRecording = async () => {
    setToast('')
    try {
      const r = await automationRecordStart()
      if (!r.success) { setToast(r.message ?? (isEn ? 'Recorder unavailable' : '녹화 엔진 미가용 (Windows 전용)')); return }
      setRecording(true); setRecCount(0)
      pollRef.current = setInterval(async () => {
        try { const s = await automationRecordStatus(); setRecCount(s.count ?? 0); if (!s.recording) stopPolling() } catch {}
      }, 1000)
    } catch { setToast(isEn ? 'Failed to start' : '시작 실패') }
  }

  const stopPolling = () => { if (pollRef.current) { clearInterval(pollRef.current); pollRef.current = null } }

  const stopRecording = async () => {
    stopPolling()
    const def = isEn ? 'my-automation' : '내자동화'
    const name = window.prompt(isEn ? 'Name this automation:' : '이 자동화의 이름을 정하세요:', def)
    setRecording(false)
    if (name === null) { setToast(isEn ? 'Discarded' : '취소됨'); return }
    setToast(isEn ? 'Saving...' : '저장 중...')
    try {
      const r = await automationRecordStop(name.trim() || def)
      if (r.saved) { setToast(isEn ? `✅ Saved "${r.workflow}" (${r.count} steps)` : `✅ "${r.workflow}" 저장됨 (${r.count}단계)`); refreshWorkflows() }
      else setToast(r.message ?? (isEn ? `Captured ${r.count ?? 0} steps` : `${r.count ?? 0}단계 캡처됨`))
    } catch { setToast(isEn ? 'Save failed' : '저장 실패') }
  }

  const replay = async (name: string) => {
    setToast(isEn ? `Running "${name}"...` : `"${name}" 실행 중...`)
    try {
      const r = await automationReplay(name)
      setToast(r.success ? (isEn ? '✅ Done' : '✅ 완료') : (r.message ?? (isEn ? 'Not available' : '사용 불가')))
    } catch {
      setToast(isEn ? 'Failed' : '실패')
    }
  }

  return (
    <div onClick={onClose} style={{ position: 'fixed', inset: 0, background: 'rgba(15,20,40,0.35)', zIndex: 10000, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
      <div onClick={(e) => e.stopPropagation()} style={{ width: 480, maxWidth: '92vw', maxHeight: '80vh', overflow: 'auto', background: '#fff', borderRadius: 16, padding: 24, boxShadow: '0 24px 64px rgba(79,108,247,0.18)' }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 14 }}>
          <h2 style={{ fontSize: 18, fontWeight: 800, color: '#1a1d2e', margin: 0 }}>🤖 {isEn ? 'Desktop Automation' : '데스크탑 자동화'}</h2>
          <button onClick={onClose} style={{ border: 'none', background: 'transparent', fontSize: 20, cursor: 'pointer', color: '#888' }}>✕</button>
        </div>

        {/* 엔진 상태 배너 */}
        <div style={{ padding: '10px 14px', borderRadius: 10, marginBottom: 16, background: available ? '#e8f7ee' : '#fff4e5', border: `1px solid ${available ? '#34c759' : '#ff9500'}33`, fontSize: 13 }}>
          <b style={{ color: available ? '#1a7f37' : '#b25e00' }}>
            {available ? (isEn ? 'Engine ready' : '엔진 준비됨') : (isEn ? 'Windows only' : 'Windows 전용')}
          </b>
          <div style={{ marginTop: 4, color: '#555' }}>{status?.message ?? (isEn ? 'Checking...' : '확인 중...')}</div>
        </div>

        {/* 녹화기 — 시연 한 번으로 자동화 만들기 */}
        {!recording ? (
          <button
            onClick={startRecording}
            disabled={!available}
            style={{ width: '100%', border: 'none', borderRadius: 10, padding: '12px', marginBottom: 16, background: available ? '#ff3b30' : '#ccc', color: '#fff', cursor: available ? 'pointer' : 'not-allowed', fontSize: 14, fontWeight: 700 }}
          >
            ● {isEn ? 'Record a task' : '새 자동화 녹화'}
          </button>
        ) : (
          <div style={{ marginBottom: 16, padding: '12px 14px', borderRadius: 10, background: '#fff0ef', border: '1px solid #ff3b3055' }}>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <span style={{ fontSize: 14, fontWeight: 700, color: '#ff3b30' }}>
                <span style={{ animation: 'recBlink 1s steps(2) infinite' }}>●</span> {isEn ? 'Recording' : '녹화 중'} · {recCount}{isEn ? ' steps' : '단계'}
              </span>
              <button onClick={stopRecording} style={{ border: 'none', borderRadius: 8, padding: '6px 14px', background: '#1a1d2e', color: '#fff', cursor: 'pointer', fontSize: 13, fontWeight: 700 }}>
                ■ {isEn ? 'Stop & Save' : '중지 & 저장'}
              </button>
            </div>
            <div style={{ marginTop: 6, fontSize: 12, color: '#b25e00' }}>
              {isEn ? 'Click and type in the target app as usual.' : '대상 앱에서 평소처럼 클릭·입력하세요.'}
            </div>
            <style>{`@keyframes recBlink { 50% { opacity: 0 } }`}</style>
          </div>
        )}

        {/* 저장된 자동화 목록 */}
        <div style={{ fontSize: 13, fontWeight: 700, color: '#555', marginBottom: 8 }}>
          {isEn ? 'Saved automations' : '저장된 자동화'} ({workflows.length})
        </div>
        {loading ? (
          <div style={{ color: '#888', fontSize: 13 }}>{isEn ? 'Loading...' : '불러오는 중...'}</div>
        ) : workflows.length === 0 ? (
          <div style={{ color: '#888', fontSize: 13, padding: '8px 0' }}>
            {isEn ? 'No automations yet. Hit “Record a task” above.' : '아직 없어요. 위 “새 자동화 녹화”를 눌러보세요.'}
          </div>
        ) : (
          workflows.map((name) => (
            <div key={name} style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '10px 12px', borderRadius: 10, border: '1px solid #eee', marginBottom: 8 }}>
              <span style={{ fontSize: 14, color: '#1a1d2e' }}>📋 {name}</span>
              <button onClick={() => replay(name)} disabled={!available} style={{ border: 'none', borderRadius: 8, padding: '6px 14px', background: available ? primaryColor : '#ccc', color: '#fff', cursor: available ? 'pointer' : 'not-allowed', fontSize: 13, fontWeight: 600 }}>
                ▶ {isEn ? 'Run' : '실행'}
              </button>
            </div>
          ))
        )}

        {toast && <div style={{ marginTop: 12, fontSize: 13, color: '#555', textAlign: 'center' }}>{toast}</div>}
      </div>
    </div>
  )
}
