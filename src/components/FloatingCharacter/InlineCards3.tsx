/**
 * InlineCards3 — 문서 비교 / Vision / Deep Search 전용 카드
 */
import React, { useState, useCallback } from 'react'
import type { DocCompareResult, DocFindResult, DeepSearchResult, DiffLine, NumberMismatch } from '../../lib/nexus/backendAPI'
import { CardWrapper } from './CardWrapper'
import { CardHeader } from './cards/CardHeader'
import { InsightLine } from './cards/InsightLine'

/* ──────────────────────────────────────────
   타입 정의
────────────────────────────────────────── */
export type InlineCard3Data =
  | { type: 'doc_compare';   data: DocCompareResult }
  | { type: 'doc_find';      data: { results: DocFindResult[]; total: number; message: string } }
  | { type: 'deep_search';   data: { results: DeepSearchResult[]; total: number; query: string; message: string } }
  | { type: 'vision_result'; data: { question: string; answer: string; screenshot_b64?: string } }
  | { type: 'vision_ocr';    data: { text: string; message: string } }
  | { type: 'smart_organize';data: { moved: number; folders: Array<{ name: string; count: number }>; message: string } }

/* ──────────────────────────────────────────
   공통 스타일
────────────────────────────────────────── */
const card: React.CSSProperties = {
  background: 'rgba(255,255,255,0.07)',
  border: '1px solid rgba(255,255,255,0.12)',
  borderRadius: 12,
  padding: '12px 14px',
  marginTop: 8,
  fontSize: 13,
  color: '#e2e8f0',
  width: 'clamp(240px, 100%, 420px)',
  lineHeight: 1.55,
}

const badge = (color: string): React.CSSProperties => ({
  display: 'inline-block',
  padding: '1px 7px',
  borderRadius: 6,
  fontSize: 11,
  fontWeight: 600,
  background: color,
  marginRight: 4,
})

const row: React.CSSProperties = {
  display: 'flex',
  justifyContent: 'space-between',
  alignItems: 'center',
  marginBottom: 4,
}

/* ──────────────────────────────────────────
   1. 문서 비교 카드
────────────────────────────────────────── */
export function DocCompareCard({ data }: { data: DocCompareResult }) {
  const [showDiff, setShowDiff] = useState(false)
  const [showNums, setShowNums] = useState(false)
  const [diffPage, setDiffPage] = useState(1)
  const DIFF_PAGE_SIZE = 30

  const sim = data.similarity_pct
  const simColor = sim >= 80 ? '#48bb78' : sim >= 50 ? '#ecc94b' : '#fc8181'

  return (
    <div style={card}>
      {/* 헤더 */}
      <div style={{ fontWeight: 700, fontSize: 14, marginBottom: 8, color: '#90cdf4' }}>
        📄 문서 비교 결과
      </div>

      {/* 파일 정보 */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8, marginBottom: 10 }}>
        <div style={{ background: 'rgba(66,153,225,0.15)', borderRadius: 8, padding: '6px 8px', fontSize: 12 }}>
          <div style={{ color: '#90cdf4', fontWeight: 600 }}>📁 파일 A</div>
          <div style={{ color: '#fff', marginTop: 2 }}>{data.file1_name}</div>
          <div style={{ color: '#a0aec0', fontSize: 11 }}>{data.file1_size}</div>
        </div>
        <div style={{ background: 'rgba(72,187,120,0.15)', borderRadius: 8, padding: '6px 8px', fontSize: 12 }}>
          <div style={{ color: '#68d391', fontWeight: 600 }}>📁 파일 B</div>
          <div style={{ color: '#fff', marginTop: 2 }}>{data.file2_name}</div>
          <div style={{ color: '#a0aec0', fontSize: 11 }}>{data.file2_size}</div>
        </div>
      </div>

      {/* 유사도 게이지 */}
      <div style={{ marginBottom: 10 }}>
        <div style={row}>
          <span style={{ fontSize: 12 }}>유사도</span>
          <span style={{ color: simColor, fontWeight: 700 }}>{sim}%</span>
        </div>
        <div style={{ background: 'rgba(255,255,255,0.1)', borderRadius: 4, height: 6 }}>
          <div style={{
            width: `${sim}%`, height: '100%', borderRadius: 4,
            background: `linear-gradient(90deg, ${simColor}, ${simColor}aa)`,
            transition: 'width 0.6s ease',
          }} />
        </div>
      </div>

      {/* 통계 배지 */}
      <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap', marginBottom: 8 }}>
        <span style={badge('rgba(72,187,120,0.3)')}>+ 추가 {data.added_count}줄</span>
        <span style={badge('rgba(252,129,129,0.3)')}>– 삭제 {data.removed_count}줄</span>
        {data.number_mismatches.length > 0 &&
          <span style={badge('rgba(237,137,54,0.3)')}>🔢 숫자 불일치 {data.number_mismatches.length}건</span>
        }
      </div>

      {/* 요약 */}
      <div style={{ fontSize: 12, color: '#cbd5e0', marginBottom: 8 }}>{data.summary}</div>

      {/* 숫자 불일치 */}
      {data.number_mismatches.length > 0 && (
        <>
          <button onClick={() => setShowNums(!showNums)} style={{
            background: 'rgba(237,137,54,0.2)', border: '1px solid rgba(237,137,54,0.4)',
            borderRadius: 6, color: '#fbd38d', padding: '3px 10px', fontSize: 12,
            cursor: 'pointer', marginBottom: 6,
          }}>
            🔢 숫자 불일치 {showNums ? '닫기' : '보기'}
          </button>
          {showNums && (
            <div style={{ maxHeight: 160, overflowY: 'auto' }}>
              {data.number_mismatches.map((m, i) => (
                <NumberMismatchRow key={i} m={m} />
              ))}
            </div>
          )}
        </>
      )}

      {/* Diff 뷰 */}
      {data.diff.length > 0 && (
        <>
          <button onClick={() => setShowDiff(!showDiff)} style={{
            background: 'rgba(66,153,225,0.2)', border: '1px solid rgba(66,153,225,0.4)',
            borderRadius: 6, color: '#90cdf4', padding: '3px 10px', fontSize: 12,
            cursor: 'pointer', marginTop: 2,
          }}>
            📝 변경 내용 {showDiff ? '접기' : '펼치기'} ({data.diff.length}건)
          </button>
          {showDiff && (
            <div style={{ marginTop: 6 }}>
              <div style={{ maxHeight: 300, overflowY: 'auto' }}>
                {data.diff.slice(0, diffPage * DIFF_PAGE_SIZE).map((d, i) => <DiffRow key={i} d={d} />)}
              </div>
              {diffPage * DIFF_PAGE_SIZE < data.diff.length && (
                <button onClick={() => setDiffPage(p => p + 1)} style={{
                  background: 'rgba(66,153,225,0.15)', border: '1px solid rgba(66,153,225,0.3)',
                  borderRadius: 6, color: '#90cdf4', padding: '3px 10px', fontSize: 11,
                  cursor: 'pointer', marginTop: 4, width: '100%',
                }}>
                  더 보기 ({Math.min((diffPage) * DIFF_PAGE_SIZE + 1, data.diff.length)}–{Math.min((diffPage + 1) * DIFF_PAGE_SIZE, data.diff.length)} / {data.diff.length})
                </button>
              )}
            </div>
          )}
        </>
      )}
    </div>
  )
}

function NumberMismatchRow({ m }: { m: NumberMismatch }) {
  const isUp = parseFloat(m.new_val) > parseFloat(m.old_val)
  return (
    <div style={{
      background: 'rgba(237,137,54,0.1)', borderRadius: 6, padding: '5px 8px',
      marginBottom: 4, fontSize: 12,
    }}>
      <div style={{ color: '#a0aec0', marginBottom: 2 }}>{m.context}</div>
      <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
        <span style={{ color: '#fc8181' }}>{m.old_val}</span>
        <span>→</span>
        <span style={{ color: '#68d391' }}>{m.new_val}</span>
        <span style={{ color: isUp ? '#68d391' : '#fc8181', fontSize: 11 }}>
          {isUp ? '▲' : '▼'} {m.change_pct.toFixed(1)}%
        </span>
      </div>
    </div>
  )
}

function DiffRow({ d }: { d: DiffLine }) {
  if (d.type === 'added') return (
    <div style={{ background: 'rgba(72,187,120,0.1)', borderLeft: '3px solid #68d391', padding: '2px 6px', marginBottom: 2, fontSize: 12, color: '#c6f6d5' }}>
      + {d.new}
    </div>
  )
  if (d.type === 'removed') return (
    <div style={{ background: 'rgba(252,129,129,0.1)', borderLeft: '3px solid #fc8181', padding: '2px 6px', marginBottom: 2, fontSize: 12, color: '#fed7d7', textDecoration: 'line-through' }}>
      – {d.old}
    </div>
  )
  return null
}

/* ──────────────────────────────────────────
   2. 문서 찾기 카드
────────────────────────────────────────── */
export function DocFindCard({ data }: { data: { results: DocFindResult[]; total: number; message: string } }) {
  const iconFor = (name: string) => {
    if (name.endsWith('.pdf')) return '📕'
    if (name.endsWith('.docx') || name.endsWith('.doc')) return '📘'
    if (name.endsWith('.xlsx') || name.endsWith('.xls')) return '📗'
    if (name.endsWith('.hwp')) return '📄'
    return '📂'
  }

  const openFile = useCallback(async (path: string) => {
    try {
      const { open } = await import('@tauri-apps/plugin-shell')
      await open(path)
    } catch { /* desktop only */ }
  }, [])

  return (
    <div style={card}>
      <CardHeader
        intent="doc_find"
        status={data.results.length > 0 ? 'success' : 'info'}
        statusLabel={data.results.length > 0 ? `${data.total}건` : '0건'}
      />
      {data.results.length === 0 ? (
        <div style={{ color: '#a0aec0', fontSize: 12 }}>해당하는 문서를 찾지 못했어요.</div>
      ) : (
        data.results.slice(0, 8).map((r, i) => (
          <div key={i} style={{
            display: 'flex', gap: 8, alignItems: 'flex-start',
            padding: '6px 0', borderBottom: '1px solid rgba(255,255,255,0.07)',
            cursor: 'pointer',
          }} onClick={() => r.path && openFile(r.path)}>
            <span style={{ fontSize: 16 }}>{iconFor(r.name)}</span>
            <div style={{ flex: 1, minWidth: 0 }}>
              <div style={{ color: '#90cdf4', fontWeight: 600, fontSize: 12, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis', textDecoration: 'underline' }}>
                {r.name}
              </div>
              {r.path && (
                <div style={{ color: '#4a5568', fontSize: 10, marginTop: 1, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                  {r.path}
                </div>
              )}
              {r.snippet && (
                <div style={{ color: '#a0aec0', fontSize: 11, marginTop: 2, lineHeight: 1.4 }}>
                  {r.snippet.slice(0, 100)}
                </div>
              )}
              <div style={{ display: 'flex', gap: 8, marginTop: 3 }}>
                <span style={{ ...badge(r.match === 'filename' ? 'rgba(66,153,225,0.3)' : r.match === 'content' ? 'rgba(72,187,120,0.3)' : 'rgba(159,122,234,0.3)') }}>
                  {r.match === 'filename' ? '파일명' : r.match === 'content' ? '내용' : '파일명+내용'}
                </span>
                <span style={{ color: '#718096', fontSize: 11 }}>{r.mod_time}</span>
                <span style={{ color: '#718096', fontSize: 11 }}>{r.size_mb.toFixed(1)}MB</span>
              </div>
            </div>
          </div>
        ))
      )}
    </div>
  )
}

/* ──────────────────────────────────────────
   3. Deep Search 카드
────────────────────────────────────────── */
export function DeepSearchCard({ data }: { data: { results: DeepSearchResult[]; total: number; query: string; message: string } }) {
  const openFile = useCallback(async (path: string) => {
    try {
      const { open } = await import('@tauri-apps/plugin-shell')
      await open(path)
    } catch { /* desktop only */ }
  }, [])

  return (
    <div style={card}>
      <CardHeader
        intent="deep_search"
        title={`심층 검색: "${data.query}"`}
        status={data.results.length > 0 ? 'success' : 'info'}
        statusLabel={`${data.total}건`}
      />
      {data.results.length === 0 ? (
        <div style={{ color: '#a0aec0', fontSize: 12 }}>관련 파일을 찾지 못했어요.</div>
      ) : (
        data.results.slice(0, 8).map((r, i) => (
          <div key={i} style={{
            padding: '6px 0', borderBottom: '1px solid rgba(255,255,255,0.07)',
            cursor: 'pointer',
          }} onClick={() => r.path && openFile(r.path)}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <span style={{ color: '#b794f4', fontWeight: 600, fontSize: 12, textDecoration: 'underline' }}>
                {r.name}
              </span>
              <ScoreBadge score={r.score} />
            </div>
            {r.path && (
              <div style={{ color: '#4a5568', fontSize: 10, marginTop: 1, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                {r.path}
              </div>
            )}
            {r.snippet && (
              <div style={{ color: '#a0aec0', fontSize: 11, marginTop: 2, lineHeight: 1.4 }}>
                {r.snippet.slice(0, 120)}
              </div>
            )}
            <div style={{ color: '#718096', fontSize: 11, marginTop: 2 }}>
              {r.ext.toUpperCase()} · {r.mod_time} · {r.size_mb.toFixed(1)}MB
            </div>
          </div>
        ))
      )}
    </div>
  )
}

function ScoreBadge({ score }: { score: number }) {
  const color = score >= 70 ? '#68d391' : score >= 40 ? '#ecc94b' : '#a0aec0'
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
      <div style={{ width: 36, height: 4, background: 'rgba(255,255,255,0.1)', borderRadius: 2 }}>
        <div style={{ width: `${score}%`, height: '100%', background: color, borderRadius: 2 }} />
      </div>
      <span style={{ color, fontSize: 11, fontWeight: 600 }}>{score}</span>
    </div>
  )
}

/* ──────────────────────────────────────────
   4. Vision 결과 카드
────────────────────────────────────────── */
export function VisionResultCard({ data }: { data: { question: string; answer: string; screenshot_b64?: string } }) {
  const [showImg, setShowImg] = useState(false)

  return (
    <div style={card}>
      <div style={{ fontWeight: 700, fontSize: 14, marginBottom: 8, color: '#76e4f7' }}>
        👁️ 화면 분석 결과
      </div>
      <div style={{ fontSize: 12, color: '#a0aec0', marginBottom: 6 }}>
        Q: {data.question}
      </div>
      <div style={{
        background: 'rgba(118,228,247,0.08)', borderRadius: 8, padding: '8px 10px',
        fontSize: 13, color: '#e2e8f0', lineHeight: 1.6,
      }}>
        {data.answer}
      </div>
      {data.screenshot_b64 && (
        <div style={{ marginTop: 8 }}>
          <button onClick={() => setShowImg(!showImg)} style={{
            background: 'rgba(118,228,247,0.15)', border: '1px solid rgba(118,228,247,0.3)',
            borderRadius: 6, color: '#76e4f7', padding: '3px 10px', fontSize: 12, cursor: 'pointer',
          }}>
            🖼️ 캡처 화면 {showImg ? '닫기' : '보기'}
          </button>
          {showImg && (
            <img
              src={`data:image/png;base64,${data.screenshot_b64}`}
              alt="캡처된 화면"
              style={{ width: '100%', borderRadius: 8, marginTop: 6, border: '1px solid rgba(255,255,255,0.1)' }}
            />
          )}
        </div>
      )}
    </div>
  )
}

/* ──────────────────────────────────────────
   5. OCR 결과 카드
────────────────────────────────────────── */
export function VisionOCRCard({ data }: { data: { text: string; message: string } }) {
  const [copied, setCopied] = useState(false)

  const handleCopy = () => {
    navigator.clipboard.writeText(data.text).then(() => {
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    })
  }

  return (
    <div style={card}>
      <div style={{ fontWeight: 700, fontSize: 14, marginBottom: 8, color: '#76e4f7', display: 'flex', justifyContent: 'space-between' }}>
        <span>🔤 OCR 텍스트 추출</span>
        <button onClick={handleCopy} style={{
          background: copied ? 'rgba(72,187,120,0.3)' : 'rgba(255,255,255,0.1)',
          border: 'none', borderRadius: 5, color: copied ? '#68d391' : '#a0aec0',
          padding: '2px 8px', fontSize: 11, cursor: 'pointer',
        }}>
          {copied ? '✓ 복사됨' : '복사'}
        </button>
      </div>
      <div style={{
        background: 'rgba(0,0,0,0.3)', borderRadius: 8, padding: '8px 10px',
        fontSize: 12, color: '#e2e8f0', whiteSpace: 'pre-wrap', maxHeight: 200,
        overflowY: 'auto', fontFamily: 'monospace', lineHeight: 1.6,
      }}>
        {data.text || '(추출된 텍스트 없음)'}
      </div>
      <div style={{ color: '#718096', fontSize: 11, marginTop: 6 }}>
        {data.message}
      </div>
    </div>
  )
}

/* ──────────────────────────────────────────
   6. 스마트 정리 결과 카드
────────────────────────────────────────── */
export function SmartOrganizeCard({ data }: { data: { moved: number; folders: Array<{ name: string; count: number }>; message: string } }) {
  const lang = ((typeof localStorage !== 'undefined' ? localStorage.getItem('nexus-lang') : 'ko') ?? 'ko') as 'ko' | 'en'
  return (
    <div style={card}>
      <CardHeader
        intent="smart_organize"
        status="success"
        statusLabel={lang === 'en' ? `${data.moved} moved` : `${data.moved}개 정리됨`}
      />
      {data.folders.length > 0 && (
        <div>
          {data.folders.map((f, i) => (
            <div key={i} style={{ display: 'flex', justifyContent: 'space-between', padding: '4px 0', fontSize: 12, borderBottom: '1px solid rgba(255,255,255,0.06)' }}>
              <span style={{ color: '#e2e8f0' }}>📁 {f.name}</span>
              <span style={{ color: '#68d391' }}>{f.count}개</span>
            </div>
          ))}
        </div>
      )}
      <InsightLine
        text={lang === 'en'
          ? `${data.moved} files organized across ${data.folders.length} categories.`
          : `총 ${data.moved}개 파일이 ${data.folders.length}개 카테고리로 정리됐어요.`}
        level="success"
      />
    </div>
  )
}

/* ──────────────────────────────────────────
   렌더러
────────────────────────────────────────── */
export function InlineCardRenderer3({ card }: { card: InlineCard3Data }) {
  switch (card.type) {
    case 'doc_compare':   return <DocCompareCard data={card.data} />
    case 'doc_find':      return <DocFindCard data={card.data} />
    case 'deep_search':   return <DeepSearchCard data={card.data} />
    case 'vision_result': return <VisionResultCard data={card.data} />
    case 'vision_ocr':    return <VisionOCRCard data={card.data} />
    case 'smart_organize':return <SmartOrganizeCard data={card.data} />
    default:              return null
  }
}
