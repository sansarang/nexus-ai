// AutomationPanel — 데스크탑 자동화 라이브러리/실행 셸 (UIA 엔진 상태 + 워크플로 재생)
// 엔진 미가용(Mac / UIA 미완성) 시 graceful degrade: 'Windows 전용' 안내 + 실행 버튼 비활성.
import { useEffect, useState, useRef } from 'react'
import {
  automationStatus, automationWorkflows, automationReplay,
  automationRecordStart, automationRecordStatus, automationRecordStop,
  automationWorkflowGet, automationBatch,
  type AutomationStatusResult, type AutoStepDef,
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

  // ── 배치 (엑셀 N행 반복) ─────────────────────────────────────
  const [batchWf, setBatchWf] = useState<string | null>(null)
  const [batchSteps, setBatchSteps] = useState<AutoStepDef[]>([])
  const [headers, setHeaders] = useState<string[]>([])
  const [rows, setRows] = useState<Record<string, string>[]>([])
  const [mapping, setMapping] = useState<Record<number, string>>({})
  const [batchBusy, setBatchBusy] = useState(false)
  const [batchMsg, setBatchMsg] = useState('')

  const openBatch = async (name: string) => {
    setBatchMsg(''); setHeaders([]); setRows([]); setMapping({})
    try {
      const r = await automationWorkflowGet(name)
      if (!r.success || !r.workflow) { setToast(isEn ? 'Failed to load workflow' : '워크플로 로드 실패'); return }
      setBatchSteps(r.workflow.steps ?? [])
      setBatchWf(name)
    } catch { setToast(isEn ? 'Failed to load workflow' : '워크플로 로드 실패') }
  }

  const onExcelFile = async (file: File) => {
    try {
      setBatchMsg(isEn ? 'Parsing...' : '파일 읽는 중...')
      const XLSX = await import('xlsx')
      const wb = XLSX.read(await file.arrayBuffer())
      const ws = wb.Sheets[wb.SheetNames[0]]
      const arr = XLSX.utils.sheet_to_json(ws, { header: 1, raw: false, defval: '' }) as unknown as string[][]
      const hdrs = (arr[0] ?? []).map((h) => String(h ?? '').trim())
      const parsed: Record<string, string>[] = arr.slice(1)
        .filter((r) => r.some((c) => String(c ?? '').trim() !== ''))
        .map((r) => {
          const o: Record<string, string> = {}
          hdrs.forEach((h, i) => { if (h) o[h] = String(r[i] ?? '') })
          return o
        })
      const cols = hdrs.filter(Boolean)
      setHeaders(cols)
      setRows(parsed)
      // 자동 매핑: 셀렉터 이름 ↔ 헤더 상호 포함이면 미리 선택
      const auto: Record<number, string> = {}
      batchSteps.forEach((s, i) => {
        if (s.kind !== 'set_text') return
        const selName = s.selector?.name ?? ''
        const hit = cols.find((c) => c && (selName.includes(c) || c.includes(selName)))
        if (hit && selName) auto[i] = hit
      })
      setMapping(auto)
      setBatchMsg(isEn ? `${parsed.length} rows · ${cols.length} columns` : `${parsed.length}행 · ${cols.length}열 인식`)
    } catch (e) {
      setBatchMsg(isEn ? 'Failed to parse file' : '엑셀 파싱 실패 — .xlsx 파일인지 확인하세요')
    }
  }

  const templatedSteps = (): AutoStepDef[] =>
    batchSteps.map((s, i) => (mapping[i] ? { ...s, value: `{{${mapping[i]}}}` } : s))

  const runBatch = async (firstRowOnly: boolean) => {
    const target = firstRowOnly ? rows.slice(0, 1) : rows
    if (target.length === 0) { setBatchMsg(isEn ? 'Load an Excel file first' : '엑셀 파일을 먼저 선택하세요'); return }
    setBatchBusy(true)
    setBatchMsg(firstRowOnly
      ? (isEn ? 'Testing with row 1...' : '1행으로 테스트 실행 중... (대상 앱을 앞에 띄워두세요)')
      : (isEn ? `Running ${target.length} rows...` : `${target.length}행 실행 중... (PC를 조작하지 마세요)`))
    try {
      const r = await automationBatch({ steps: templatedSteps(), rows: target, stop_on_error: true })
      const res = r.result
      if (res) {
        const pct = Math.round((res.success_rate ?? 0) * 100)
        setBatchMsg(res.succeeded === res.total
          ? (isEn ? `✅ ${res.succeeded}/${res.total} rows done` : `✅ ${res.succeeded}/${res.total}행 완료`)
          : (isEn ? `⚠️ ${res.succeeded}/${res.total} rows (${pct}%) — stopped at first failure` : `⚠️ ${res.succeeded}/${res.total}행 성공 (${pct}%) — 실패 행에서 중단됨`))
      } else {
        setBatchMsg(r.message ?? (isEn ? 'Engine unavailable' : '엔진 미가용 (Windows 전용)'))
      }
    } catch {
      setBatchMsg(isEn ? 'Batch failed' : '배치 실행 실패')
    } finally {
      setBatchBusy(false)
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

        {/* ── 배치 뷰 (엑셀 N행 반복) ── */}
        {batchWf && (
          <div>
            <button onClick={() => setBatchWf(null)} disabled={batchBusy} style={{ border: 'none', background: 'transparent', color: primaryColor, cursor: 'pointer', fontSize: 13, fontWeight: 700, padding: 0, marginBottom: 10 }}>
              ← {isEn ? 'Back' : '목록으로'}
            </button>
            <div style={{ fontSize: 15, fontWeight: 800, color: '#1a1d2e', marginBottom: 10 }}>📊 {batchWf} — {isEn ? 'Excel batch' : '엑셀 반복 실행'}</div>

            {/* 1. 엑셀 선택 */}
            <label style={{ display: 'block', fontSize: 13, fontWeight: 700, color: '#555', marginBottom: 6 }}>
              1. {isEn ? 'Excel file (row 1 = column names)' : '엑셀 파일 (첫 행 = 열 이름)'}
            </label>
            <input type="file" accept=".xlsx,.xls,.csv" disabled={batchBusy}
              onChange={(e) => { const f = e.target.files?.[0]; if (f) onExcelFile(f) }}
              style={{ fontSize: 13, marginBottom: 14 }} />

            {/* 2. 컬럼 매핑 */}
            {headers.length > 0 && (
              <div style={{ marginBottom: 14 }}>
                <div style={{ fontSize: 13, fontWeight: 700, color: '#555', marginBottom: 6 }}>
                  2. {isEn ? 'Which column goes into each field?' : '어느 칸에 어느 열을 넣을까요?'}
                </div>
                {batchSteps.map((s, i) => s.kind === 'set_text' ? (
                  <div key={i} style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '6px 0', fontSize: 13 }}>
                    <span style={{ flex: 1, color: '#1a1d2e', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      ✏️ {s.selector?.name || s.selector?.automation_id || (isEn ? 'field' : '입력칸')}
                      <span style={{ color: '#999' }}> ({isEn ? 'recorded' : '녹화값'}: {s.value})</span>
                    </span>
                    <select value={mapping[i] ?? ''} disabled={batchBusy}
                      onChange={(e) => setMapping((m) => ({ ...m, [i]: e.target.value }))}
                      style={{ fontSize: 13, padding: '4px 8px', borderRadius: 6, border: '1px solid #ddd' }}>
                      <option value="">{isEn ? 'keep recorded value' : '녹화값 그대로'}</option>
                      {headers.map((h) => <option key={h} value={h}>{h}</option>)}
                    </select>
                  </div>
                ) : null)}
                {batchSteps.every((s) => s.kind !== 'set_text') && (
                  <div style={{ fontSize: 12, color: '#999' }}>{isEn ? 'No text-input steps in this automation.' : '이 자동화엔 입력 단계가 없어요 (클릭만 반복).'}</div>
                )}
              </div>
            )}

            {/* 3. 실행 */}
            {rows.length > 0 && (
              <div style={{ display: 'flex', gap: 8 }}>
                <button onClick={() => runBatch(true)} disabled={batchBusy || !available}
                  style={{ flex: 1, border: `1px solid ${primaryColor}`, borderRadius: 8, padding: '10px', background: '#fff', color: primaryColor, cursor: batchBusy || !available ? 'not-allowed' : 'pointer', fontSize: 13, fontWeight: 700 }}>
                  {isEn ? 'Test row 1' : '1행만 테스트'}
                </button>
                <button onClick={() => runBatch(false)} disabled={batchBusy || !available}
                  style={{ flex: 2, border: 'none', borderRadius: 8, padding: '10px', background: batchBusy || !available ? '#ccc' : primaryColor, color: '#fff', cursor: batchBusy || !available ? 'not-allowed' : 'pointer', fontSize: 13, fontWeight: 700 }}>
                  ▶ {isEn ? `Run all ${rows.length} rows` : `전체 ${rows.length}행 실행`}
                </button>
              </div>
            )}
            {batchMsg && <div style={{ marginTop: 12, fontSize: 13, color: '#555' }}>{batchMsg}</div>}
          </div>
        )}

        {/* 녹화기 — 시연 한 번으로 자동화 만들기 */}
        {!batchWf && (!recording ? (
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
        ))}

        {/* 저장된 자동화 목록 */}
        {!batchWf && (<>
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
            <div key={name} style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 8, padding: '10px 12px', borderRadius: 10, border: '1px solid #eee', marginBottom: 8 }}>
              <span style={{ fontSize: 14, color: '#1a1d2e', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>📋 {name}</span>
              <div style={{ display: 'flex', gap: 6, flexShrink: 0 }}>
                <button onClick={() => openBatch(name)} style={{ border: `1px solid ${primaryColor}`, borderRadius: 8, padding: '6px 10px', background: '#fff', color: primaryColor, cursor: 'pointer', fontSize: 13, fontWeight: 600 }}>
                  📊 {isEn ? 'Batch' : '배치'}
                </button>
                <button onClick={() => replay(name)} disabled={!available} style={{ border: 'none', borderRadius: 8, padding: '6px 14px', background: available ? primaryColor : '#ccc', color: '#fff', cursor: available ? 'pointer' : 'not-allowed', fontSize: 13, fontWeight: 600 }}>
                  ▶ {isEn ? 'Run' : '실행'}
                </button>
              </div>
            </div>
          ))
        )}
        </>)}

        {toast && <div style={{ marginTop: 12, fontSize: 13, color: '#555', textAlign: 'center' }}>{toast}</div>}
      </div>
    </div>
  )
}
