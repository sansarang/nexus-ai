/**
 * InlineCards4 — 업무 일지 / 매크로 / PC 리포트 / 문서 요약 카드
 */
import React, { useState } from 'react'

/* ──────────────────────────────────────────
   공통 타입
────────────────────────────────────────── */
export interface ActivityEntry {
  name: string
  type: 'app' | 'file'
  path?: string
  duration_min?: number
  last_seen?: string
  count?: number
}

export interface DayJournalData {
  date: string
  work_hours: number
  app_usage: ActivityEntry[]
  recent_files: ActivityEntry[]
  summary: string
  generated: string
}

export interface MacroAction {
  type: string
  label: string
  params: Record<string, string>
}

export interface MacroTrigger {
  type: string
  time?: string
  days?: number[]
  interval_min?: number
}

export interface MacroData {
  id: string
  name: string
  description?: string
  trigger: MacroTrigger
  actions: MacroAction[]
  enabled: boolean
  last_run?: string
  run_count: number
  created_at: string
}

export interface MacroResult {
  action: string
  label: string
  success: boolean
  message: string
}

export interface ReportIssue {
  level: 'info' | 'warn' | 'critical'
  title: string
  detail: string
}

export interface PCHealthReportData {
  date: string
  score: number
  cpu_avg: number
  memory_avg: number
  disk_free_gb: number
  cpu_temp: number
  issues: ReportIssue[]
  suggestions: string[]
  security_ok: boolean
}

export interface DocSummaryData {
  file_name: string
  file_path: string
  file_size: string
  word_count: number
  key_points: string[]
  key_numbers: string[]
  dates: string[]
  summary: string
  language: string
  category: string
}

export type InlineCard4Data =
  | { type: 'journal_today'; data: DayJournalData }
  | { type: 'journal_history'; data: { history: Array<{ date: string; work_hours: number; file_count: number; app_count: number; top_app: string }> } }
  | { type: 'macro_list'; data: { macros: MacroData[]; total: number } }
  | { type: 'macro_created'; data: { macro: MacroData; message: string } }
  | { type: 'macro_run'; data: { name: string; results: MacroResult[]; message: string } }
  | { type: 'pc_report'; data: PCHealthReportData }
  | { type: 'doc_summary'; data: DocSummaryData }

/* ──────────────────────────────────────────
   공통 스타일
────────────────────────────────────────── */
const card: React.CSSProperties = {
  background: 'rgba(255,255,255,0.06)',
  border: '1px solid rgba(255,255,255,0.1)',
  borderRadius: 12,
  padding: '12px 14px',
  marginTop: 8,
  fontSize: 13,
  color: '#e2e8f0',
  maxWidth: 420,
  lineHeight: 1.55,
}

const row: React.CSSProperties = {
  display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 4,
}

const badge = (color: string): React.CSSProperties => ({
  display: 'inline-block', padding: '2px 8px', borderRadius: 6,
  fontSize: 11, fontWeight: 600, background: color, margin: '2px',
})

/* ──────────────────────────────────────────
   게이지 컴포넌트
────────────────────────────────────────── */
function Gauge({ value, max = 100, color }: { value: number; max?: number; color: string }) {
  const pct = Math.min((value / max) * 100, 100)
  return (
    <div style={{ background: 'rgba(255,255,255,0.08)', borderRadius: 4, height: 5, overflow: 'hidden' }}>
      <div style={{
        width: `${pct}%`, height: '100%', borderRadius: 4,
        background: color, transition: 'width 0.7s ease',
      }} />
    </div>
  )
}

/* ──────────────────────────────────────────
   1. 업무 일지 카드
────────────────────────────────────────── */
export function JournalTodayCard({ data }: { data: DayJournalData }) {
  const [tab, setTab] = useState<'apps' | 'files'>('apps')
  const hours = data.work_hours?.toFixed(1) || '0.0'

  return (
    <div style={card}>
      <div style={{ fontWeight: 700, fontSize: 14, marginBottom: 10, color: '#fbd38d', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <span>📋 오늘의 업무 일지</span>
        <span style={{ fontSize: 12, color: '#718096' }}>{data.date}</span>
      </div>

      {/* 업무 시간 */}
      <div style={{ background: 'rgba(251,211,141,0.1)', borderRadius: 8, padding: '8px 12px', marginBottom: 10 }}>
        <div style={row}>
          <span style={{ fontSize: 12 }}>⏱️ 추정 업무 시간</span>
          <span style={{ color: '#fbd38d', fontWeight: 700, fontSize: 18 }}>{hours}h</span>
        </div>
        <Gauge value={parseFloat(hours)} max={10} color="#fbd38d" />
      </div>

      {/* 탭 */}
      <div style={{ display: 'flex', gap: 4, marginBottom: 8 }}>
        {(['apps', 'files'] as const).map(t => (
          <button key={t} onClick={() => setTab(t)} style={{
            flex: 1, padding: '4px 0', borderRadius: 6, border: 'none', cursor: 'pointer',
            background: tab === t ? 'rgba(251,211,141,0.25)' : 'rgba(255,255,255,0.06)',
            color: tab === t ? '#fbd38d' : '#a0aec0', fontSize: 12, fontWeight: tab === t ? 700 : 400,
          }}>
            {t === 'apps' ? `💻 앱 (${data.app_usage?.length || 0})` : `📂 파일 (${data.recent_files?.length || 0})`}
          </button>
        ))}
      </div>

      {/* 내용 */}
      <div style={{ maxHeight: 180, overflowY: 'auto' }}>
        {tab === 'apps' && (data.app_usage || []).slice(0, 8).map((a, i) => (
          <div key={i} style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '4px 0', borderBottom: '1px solid rgba(255,255,255,0.05)' }}>
            <span style={{ fontSize: 16 }}>💻</span>
            <div style={{ flex: 1 }}>
              <div style={{ fontSize: 12, color: '#e2e8f0' }}>{a.name}</div>
              {a.duration_min && a.duration_min > 0 && (
                <Gauge value={a.duration_min} max={120} color="rgba(251,211,141,0.6)" />
              )}
            </div>
            {a.duration_min && <span style={{ fontSize: 11, color: '#718096' }}>{Math.round(a.duration_min)}분</span>}
          </div>
        ))}
        {tab === 'files' && (data.recent_files || []).slice(0, 10).map((f, i) => (
          <div key={i} style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '4px 0', borderBottom: '1px solid rgba(255,255,255,0.05)' }}>
            <span style={{ fontSize: 14 }}>{fileIcon(f.name)}</span>
            <div style={{ flex: 1, minWidth: 0 }}>
              <div style={{ fontSize: 12, color: '#e2e8f0', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{f.name}</div>
            </div>
            {f.last_seen && <span style={{ fontSize: 11, color: '#718096' }}>{f.last_seen}</span>}
          </div>
        ))}
      </div>

      {/* 요약 */}
      {data.summary && (
        <div style={{ background: 'rgba(0,0,0,0.2)', borderRadius: 6, padding: '6px 8px', marginTop: 8, fontSize: 11, color: '#a0aec0', whiteSpace: 'pre-line' }}>
          {data.summary}
        </div>
      )}
    </div>
  )
}

function fileIcon(name: string) {
  const ext = name.split('.').pop()?.toLowerCase() || ''
  if (['pdf'].includes(ext)) return '📕'
  if (['docx', 'doc'].includes(ext)) return '📘'
  if (['xlsx', 'xls'].includes(ext)) return '📗'
  if (['pptx', 'ppt'].includes(ext)) return '📙'
  if (['jpg', 'jpeg', 'png', 'gif'].includes(ext)) return '🖼️'
  if (['mp4', 'avi', 'mkv'].includes(ext)) return '🎬'
  if (['zip', 'rar', '7z'].includes(ext)) return '📦'
  return '📄'
}

/* ──────────────────────────────────────────
   2. 매크로 목록 카드
────────────────────────────────────────── */
export function MacroListCard({
  data,
  onRun,
}: {
  data: { macros: MacroData[]; total: number }
  onRun?: (id: string, name: string) => void
}) {
  const triggerLabel = (t: MacroTrigger) => {
    if (t.type === 'time') return `⏰ 매일 ${t.time}`
    if (t.type === 'startup') return '🚀 시작 시'
    if (t.type === 'interval') return `🔄 ${t.interval_min}분마다`
    return '▶️ 수동'
  }

  return (
    <div style={card}>
      <div style={{ fontWeight: 700, fontSize: 14, marginBottom: 8, color: '#9f7aea' }}>
        ⚡ 자동화 매크로 ({data.total}개)
      </div>
      {data.macros.length === 0 ? (
        <div style={{ color: '#718096', fontSize: 12 }}>등록된 매크로가 없어요. "매일 아침 9시에 크롬 열어줘" 처럼 말해보세요!</div>
      ) : (
        data.macros.map((m) => (
          <div key={m.id} style={{
            background: 'rgba(159,122,234,0.08)', borderRadius: 8, padding: '8px 10px',
            marginBottom: 6, display: 'flex', justifyContent: 'space-between', alignItems: 'center',
          }}>
            <div>
              <div style={{ fontWeight: 600, color: m.enabled ? '#e2e8f0' : '#718096' }}>{m.name}</div>
              <div style={{ fontSize: 11, color: '#a0aec0', marginTop: 2 }}>
                {triggerLabel(m.trigger)} · {m.actions.length}개 동작
                {m.run_count > 0 && ` · ${m.run_count}회 실행`}
              </div>
            </div>
            {onRun && (
              <button onClick={() => onRun(m.id, m.name)} style={{
                background: 'rgba(159,122,234,0.3)', border: '1px solid rgba(159,122,234,0.5)',
                borderRadius: 6, color: '#d6bcfa', padding: '4px 10px',
                fontSize: 12, cursor: 'pointer', fontWeight: 600,
              }}>
                ▶ 실행
              </button>
            )}
          </div>
        ))
      )}
    </div>
  )
}

/* ──────────────────────────────────────────
   3. 매크로 생성 확인 카드
────────────────────────────────────────── */
export function MacroCreatedCard({ data }: { data: { macro: MacroData; message: string } }) {
  const m = data.macro
  const actionIcons: Record<string, string> = {
    launch: '🚀', clean: '🧹', folder: '📁', volume: '🔊',
    brightness: '☀️', delay: '⏳', shell: '⚙️', message: '💬',
  }

  return (
    <div style={card}>
      <div style={{ fontWeight: 700, fontSize: 14, marginBottom: 8, color: '#68d391' }}>
        ✅ 매크로 등록 완료
      </div>
      <div style={{ background: 'rgba(72,187,120,0.1)', borderRadius: 8, padding: '8px 10px', marginBottom: 8 }}>
        <div style={{ fontWeight: 600, color: '#e2e8f0' }}>{m.name}</div>
        {m.trigger.type === 'time' && (
          <div style={{ fontSize: 12, color: '#a0aec0', marginTop: 2 }}>⏰ 매일 {m.trigger.time}에 자동 실행</div>
        )}
      </div>
      <div style={{ fontSize: 12, color: '#a0aec0', marginBottom: 6 }}>실행 동작:</div>
      {(m.actions || []).map((a, i) => (
        <div key={i} style={{ display: 'flex', alignItems: 'center', gap: 6, padding: '3px 0', fontSize: 12 }}>
          <span>{actionIcons[a.type] || '▶️'}</span>
          <span style={{ color: '#e2e8f0' }}>{a.label}</span>
        </div>
      ))}
    </div>
  )
}

/* ──────────────────────────────────────────
   4. 매크로 실행 결과 카드
────────────────────────────────────────── */
export function MacroRunCard({ data }: { data: { name: string; results: MacroResult[]; message: string } }) {
  return (
    <div style={card}>
      <div style={{ fontWeight: 700, fontSize: 14, marginBottom: 8, color: '#9f7aea' }}>
        ⚡ "{data.name}" 실행 완료
      </div>
      {data.results.map((r, i) => (
        <div key={i} style={{
          display: 'flex', alignItems: 'center', gap: 8,
          padding: '4px 0', borderBottom: '1px solid rgba(255,255,255,0.05)', fontSize: 12,
        }}>
          <span>{r.success ? '✅' : '❌'}</span>
          <div>
            <div style={{ color: '#e2e8f0' }}>{r.label || r.action}</div>
            <div style={{ color: '#718096', fontSize: 11 }}>{r.message}</div>
          </div>
        </div>
      ))}
    </div>
  )
}

/* ──────────────────────────────────────────
   5. PC 건강 리포트 카드
────────────────────────────────────────── */
export function PCReportCard({ data }: { data: PCHealthReportData }) {
  const scoreColor = data.score >= 80 ? '#48bb78' : data.score >= 60 ? '#ecc94b' : '#fc8181'
  const issueIcon = (level: string) => ({ info: '✅', warn: '⚠️', critical: '🔴' }[level] || '•')

  return (
    <div style={card}>
      <div style={{ fontWeight: 700, fontSize: 14, marginBottom: 10, color: '#90cdf4' }}>
        🖥️ PC 건강 리포트
      </div>

      {/* 점수 */}
      <div style={{ textAlign: 'center', marginBottom: 12 }}>
        <div style={{ fontSize: 52, fontWeight: 800, color: scoreColor, lineHeight: 1 }}>{data.score}</div>
        <div style={{ fontSize: 13, color: '#718096', marginTop: 2 }}>/ 100점</div>
      </div>

      {/* 스탯 */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 6, marginBottom: 10 }}>
        {[
          { label: '⚡ CPU', value: data.cpu_avg, unit: '%', max: 100, color: data.cpu_avg > 80 ? '#fc8181' : '#68d391' },
          { label: '🧠 메모리', value: data.memory_avg, unit: '%', max: 100, color: data.memory_avg > 80 ? '#fc8181' : '#68d391' },
          { label: '🌡️ 온도', value: data.cpu_temp, unit: '°C', max: 100, color: data.cpu_temp > 80 ? '#fc8181' : '#68d391' },
          { label: '💾 디스크', value: 100 - (data.disk_free_gb || 0), unit: '%', max: 100, color: '#90cdf4' },
        ].map((s, i) => (
          <div key={i} style={{ background: 'rgba(255,255,255,0.05)', borderRadius: 8, padding: '6px 10px' }}>
            <div style={row}>
              <span style={{ fontSize: 11, color: '#a0aec0' }}>{s.label}</span>
              <span style={{ fontSize: 13, fontWeight: 700, color: s.color }}>{s.value?.toFixed(0)}{s.unit}</span>
            </div>
            <Gauge value={s.value || 0} max={s.max} color={s.color} />
          </div>
        ))}
      </div>

      {/* 이슈 */}
      <div>
        {(data.issues || []).map((iss, i) => (
          <div key={i} style={{
            padding: '5px 8px', marginBottom: 4, borderRadius: 6,
            background: iss.level === 'critical' ? 'rgba(252,129,129,0.1)' :
              iss.level === 'warn' ? 'rgba(237,137,54,0.1)' : 'rgba(72,187,120,0.1)',
            fontSize: 12,
          }}>
            <div style={{ fontWeight: 600 }}>{issueIcon(iss.level)} {iss.title}</div>
            <div style={{ color: '#a0aec0', marginTop: 1 }}>{iss.detail}</div>
          </div>
        ))}
      </div>

      {/* 제안 */}
      {(data.suggestions || []).length > 0 && (
        <div style={{ marginTop: 8, padding: '6px 8px', background: 'rgba(144,205,244,0.08)', borderRadius: 6 }}>
          <div style={{ fontSize: 11, color: '#90cdf4', fontWeight: 600, marginBottom: 4 }}>💡 개선 제안</div>
          {data.suggestions.slice(0, 2).map((s, i) => (
            <div key={i} style={{ fontSize: 11, color: '#a0aec0', marginBottom: 2 }}>• {s}</div>
          ))}
        </div>
      )}
    </div>
  )
}

/* ──────────────────────────────────────────
   6. 문서 요약 카드
────────────────────────────────────────── */
export function DocSummaryCard({ data }: { data: DocSummaryData }) {
  const [showPoints, setShowPoints] = useState(false)
  const catLabel: Record<string, string> = {
    contract: '📜 계약서', invoice: '🧾 청구서', report: '📊 보고서',
    proposal: '💼 제안서', minutes: '📝 회의록', document: '📄 문서',
  }

  return (
    <div style={card}>
      <div style={{ fontWeight: 700, fontSize: 14, marginBottom: 8, color: '#f6ad55', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <span>📑 문서 요약</span>
        <span style={badge('rgba(246,173,85,0.2)')}>{catLabel[data.category] || '📄 문서'}</span>
      </div>

      {/* 파일 정보 */}
      <div style={{ background: 'rgba(246,173,85,0.08)', borderRadius: 8, padding: '6px 10px', marginBottom: 8, fontSize: 12 }}>
        <div style={{ color: '#e2e8f0', fontWeight: 600 }}>{data.file_name}</div>
        <div style={{ color: '#718096', marginTop: 2 }}>
          {data.file_size} · {data.word_count?.toLocaleString()}단어 · {data.language === 'ko' ? '한국어' : data.language === 'en' ? '영어' : '혼합'}
        </div>
      </div>

      {/* 요약 */}
      <div style={{ fontSize: 12, color: '#e2e8f0', lineHeight: 1.6, marginBottom: 8, background: 'rgba(0,0,0,0.2)', borderRadius: 6, padding: '6px 8px' }}>
        {data.summary}
      </div>

      {/* 주요 날짜 */}
      {(data.dates || []).length > 0 && (
        <div style={{ marginBottom: 6 }}>
          <div style={{ fontSize: 11, color: '#90cdf4', marginBottom: 4, fontWeight: 600 }}>📅 주요 날짜</div>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4 }}>
            {data.dates.map((d, i) => <span key={i} style={badge('rgba(144,205,244,0.2)')}>{d}</span>)}
          </div>
        </div>
      )}

      {/* 주요 수치 */}
      {(data.key_numbers || []).length > 0 && (
        <div style={{ marginBottom: 6 }}>
          <div style={{ fontSize: 11, color: '#68d391', marginBottom: 4, fontWeight: 600 }}>🔢 주요 수치</div>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4 }}>
            {data.key_numbers.slice(0, 5).map((n, i) => <span key={i} style={badge('rgba(104,211,145,0.2)')}>{n}</span>)}
          </div>
        </div>
      )}

      {/* 핵심 내용 토글 */}
      {(data.key_points || []).length > 0 && (
        <>
          <button onClick={() => setShowPoints(!showPoints)} style={{
            background: 'rgba(246,173,85,0.15)', border: '1px solid rgba(246,173,85,0.3)',
            borderRadius: 6, color: '#f6ad55', padding: '3px 10px', fontSize: 12, cursor: 'pointer',
          }}>
            📌 핵심 조항 {showPoints ? '접기' : `보기 (${data.key_points.length}개)`}
          </button>
          {showPoints && (
            <div style={{ marginTop: 6 }}>
              {data.key_points.map((p, i) => (
                <div key={i} style={{
                  borderLeft: '2px solid rgba(246,173,85,0.5)', paddingLeft: 8,
                  marginBottom: 4, fontSize: 12, color: '#e2e8f0', lineHeight: 1.5,
                }}>
                  {p}
                </div>
              ))}
            </div>
          )}
        </>
      )}
    </div>
  )
}

/* ──────────────────────────────────────────
   렌더러
────────────────────────────────────────── */
export function InlineCardRenderer4({
  card,
  onMacroRun,
}: {
  card: InlineCard4Data
  onMacroRun?: (id: string, name: string) => void
}) {
  switch (card.type) {
    case 'journal_today':   return <JournalTodayCard data={card.data} />
    case 'macro_list':      return <MacroListCard data={card.data} onRun={onMacroRun} />
    case 'macro_created':   return <MacroCreatedCard data={card.data} />
    case 'macro_run':       return <MacroRunCard data={card.data} />
    case 'pc_report':       return <PCReportCard data={card.data} />
    case 'doc_summary':     return <DocSummaryCard data={card.data} />
    default: return null
  }
}
