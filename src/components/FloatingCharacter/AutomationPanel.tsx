// AutomationPanel — 데스크탑 자동화 라이브러리/실행 셸 (UIA 엔진 상태 + 워크플로 재생)
// 엔진 미가용(Mac / UIA 미완성) 시 graceful degrade: 'Windows 전용' 안내 + 실행 버튼 비활성.
import { useEffect, useState } from 'react'
import {
  automationStatus, automationWorkflows, automationReplay,
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

  if (!open) return null

  const available = status?.available ?? false

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

        {/* 저장된 자동화 목록 */}
        <div style={{ fontSize: 13, fontWeight: 700, color: '#555', marginBottom: 8 }}>
          {isEn ? 'Saved automations' : '저장된 자동화'} ({workflows.length})
        </div>
        {loading ? (
          <div style={{ color: '#888', fontSize: 13 }}>{isEn ? 'Loading...' : '불러오는 중...'}</div>
        ) : workflows.length === 0 ? (
          <div style={{ color: '#888', fontSize: 13, padding: '8px 0' }}>
            {isEn ? 'No automations yet. Record one on Windows.' : '아직 없어요. Windows에서 녹화해보세요.'}
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
