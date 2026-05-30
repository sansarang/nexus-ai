/**
 * DynamicBlocks — "어떤 질문도, 어떤 결과도" 의 핵심.
 *
 * LLM 이 응답 구조를 직접 조립할 수 있도록 10개 UI 프리미티브 제공.
 * 사전 정의된 카드 타입으로 표현 불가능한 동적 결과는 Block[] 로 응답.
 *
 * 예: "Q1 매출 분석" → LLM 이 [heading, keyvalue, chart, callout, action] 조립
 */

import { motion } from 'framer-motion'
import { InsightLine, type InsightLevel } from './InsightLine'

/* ─────────────────────────────────────────────────────────── */
/* Block 스키마 (LLM 이 이 schema 따라 응답 조립)               */
/* ─────────────────────────────────────────────────────────── */

export type Block =
  | { type: 'text';     content: string; tone?: 'normal'|'highlight'|'muted' }
  | { type: 'heading';  level?: 1|2|3; text: string; icon?: string }
  | { type: 'list';     items: string[]; ordered?: boolean }
  | { type: 'keyvalue'; pairs: Array<{label: string; value: string; trend?: 'up'|'down'|'flat'; emphasis?: boolean}> }
  | { type: 'table';    headers: string[]; rows: string[][]; caption?: string }
  | { type: 'chart';    kind: 'bar'|'line'|'pie'; data: Array<{label: string; value: number; color?: string}>; title?: string; unit?: string }
  | { type: 'image';    url: string; alt?: string; caption?: string }
  | { type: 'file';     name: string; url: string; sizeKB?: number; mime?: string }
  | { type: 'action';   label: string; command: string; icon?: string; variant?: 'primary'|'default'|'danger' }
  | { type: 'steps';    items: Array<{label: string; status: 'pending'|'running'|'done'|'failed'; detail?: string}> }
  | { type: 'callout';  level: InsightLevel; text: string }
  | { type: 'divider' }

interface DynamicCardRendererProps {
  blocks: Block[]
  accentColor?: string
  /** action 블록 클릭 시 호출 (보통 sendText로 명령 재전송) */
  onAction?: (command: string) => void
}

export function DynamicCardRenderer({ blocks, accentColor = '#9b59b6', onAction }: DynamicCardRendererProps) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
      {blocks.map((b, i) => (
        <BlockRenderer key={i} block={b} accentColor={accentColor} onAction={onAction} />
      ))}
    </div>
  )
}

function BlockRenderer({ block, accentColor, onAction }: { block: Block; accentColor: string; onAction?: (cmd: string) => void }) {
  switch (block.type) {
    case 'text':     return <TextBlock {...block} />
    case 'heading':  return <HeadingBlock {...block} accentColor={accentColor} />
    case 'list':     return <ListBlock {...block} accentColor={accentColor} />
    case 'keyvalue': return <KeyValueBlock {...block} accentColor={accentColor} />
    case 'table':    return <TableBlock {...block} accentColor={accentColor} />
    case 'chart':    return <ChartBlock {...block} accentColor={accentColor} />
    case 'image':    return <ImageBlock {...block} />
    case 'file':     return <FileBlock {...block} accentColor={accentColor} />
    case 'action':   return <ActionBlock {...block} accentColor={accentColor} onAction={onAction} />
    case 'steps':    return <StepsBlock {...block} accentColor={accentColor} />
    case 'callout':  return <InsightLine text={block.text} level={block.level} />
    case 'divider':  return <div style={{ height: 1, background: 'rgba(255,255,255,0.08)', margin: '4px 0' }} />
    default: {
      const _exhaustive: never = block
      void _exhaustive
      return null
    }
  }
}

/* ─── Text ─────────────────────────────────────────────────── */
function TextBlock({ content, tone }: Extract<Block, { type: 'text' }>) {
  const color = tone === 'highlight' ? '#fbbf24' : tone === 'muted' ? 'rgba(255,255,255,0.5)' : 'rgba(255,255,255,0.88)'
  return <div style={{ fontSize: 12, color, lineHeight: 1.6 }}>{content}</div>
}

/* ─── Heading ──────────────────────────────────────────────── */
function HeadingBlock({ level = 2, text, icon, accentColor }: Extract<Block, { type: 'heading' }> & { accentColor: string }) {
  const size = level === 1 ? 16 : level === 2 ? 13 : 11
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginTop: level === 1 ? 4 : 2 }}>
      {icon && <span style={{ fontSize: size + 2 }}>{icon}</span>}
      <span style={{ fontSize: size, fontWeight: 800, color: level === 1 ? accentColor : 'rgba(255,255,255,0.95)' }}>{text}</span>
    </div>
  )
}

/* ─── List ─────────────────────────────────────────────────── */
function ListBlock({ items, ordered, accentColor }: Extract<Block, { type: 'list' }> & { accentColor: string }) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 4, paddingLeft: 4 }}>
      {items.map((item, i) => (
        <div key={i} style={{ display: 'flex', gap: 8, fontSize: 11.5, color: 'rgba(255,255,255,0.85)' }}>
          <span style={{ color: accentColor, fontWeight: 700, minWidth: 16 }}>
            {ordered ? `${i+1}.` : '•'}
          </span>
          <span>{item}</span>
        </div>
      ))}
    </div>
  )
}

/* ─── KeyValue ─────────────────────────────────────────────── */
function KeyValueBlock({ pairs, accentColor }: Extract<Block, { type: 'keyvalue' }> & { accentColor: string }) {
  const trendIcon = (t?: 'up'|'down'|'flat') => t === 'up' ? '▲' : t === 'down' ? '▼' : t === 'flat' ? '—' : ''
  const trendColor = (t?: 'up'|'down'|'flat') => t === 'up' ? '#22c55e' : t === 'down' ? '#ef4444' : 'rgba(255,255,255,0.4)'
  return (
    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(110px, 1fr))', gap: 6 }}>
      {pairs.map((p, i) => (
        <div key={i} style={{
          padding: '8px 10px',
          background: p.emphasis ? `${accentColor}15` : 'rgba(255,255,255,0.04)',
          border: `1px solid ${p.emphasis ? accentColor + '44' : 'rgba(255,255,255,0.08)'}`,
          borderRadius: 8,
        }}>
          <div style={{ fontSize: 9, color: 'rgba(255,255,255,0.45)', marginBottom: 2 }}>{p.label}</div>
          <div style={{ display: 'flex', alignItems: 'baseline', gap: 4 }}>
            <span style={{ fontSize: 14, fontWeight: 800, color: p.emphasis ? accentColor : 'rgba(255,255,255,0.95)' }}>
              {p.value}
            </span>
            {p.trend && <span style={{ fontSize: 10, color: trendColor(p.trend), fontWeight: 700 }}>{trendIcon(p.trend)}</span>}
          </div>
        </div>
      ))}
    </div>
  )
}

/* ─── Table ────────────────────────────────────────────────── */
function TableBlock({ headers, rows, caption, accentColor }: Extract<Block, { type: 'table' }> & { accentColor: string }) {
  return (
    <div style={{ overflow: 'auto', maxHeight: 280, border: '1px solid rgba(255,255,255,0.08)', borderRadius: 8 }}>
      <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 11 }}>
        <thead style={{ position: 'sticky', top: 0, background: `${accentColor}22` }}>
          <tr>
            {headers.map((h, i) => (
              <th key={i} style={{ textAlign: 'left', padding: '6px 8px', color: accentColor, fontWeight: 700, borderBottom: `1px solid ${accentColor}44` }}>{h}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row, ri) => (
            <tr key={ri} style={{ background: ri % 2 === 0 ? 'rgba(255,255,255,0.02)' : 'transparent' }}>
              {row.map((cell, ci) => (
                <td key={ci} style={{ padding: '5px 8px', color: 'rgba(255,255,255,0.8)', borderBottom: '1px solid rgba(255,255,255,0.04)' }}>{cell}</td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
      {caption && <div style={{ fontSize: 9, color: 'rgba(255,255,255,0.4)', padding: '4px 8px', borderTop: '1px solid rgba(255,255,255,0.04)' }}>{caption}</div>}
    </div>
  )
}

/* ─── Chart (SVG 미니멀) ───────────────────────────────────── */
function ChartBlock({ kind, data, title, unit, accentColor }: Extract<Block, { type: 'chart' }> & { accentColor: string }) {
  if (!data || data.length === 0) return null
  const max = Math.max(...data.map(d => d.value), 1)

  return (
    <div style={{ padding: '8px 10px', background: 'rgba(255,255,255,0.03)', borderRadius: 8 }}>
      {title && <div style={{ fontSize: 11, fontWeight: 700, color: 'rgba(255,255,255,0.85)', marginBottom: 6 }}>{title}</div>}
      {kind === 'bar' && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
          {data.map((d, i) => (
            <div key={i} style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
              <span style={{ fontSize: 10, color: 'rgba(255,255,255,0.6)', minWidth: 48 }}>{d.label}</span>
              <div style={{ flex: 1, height: 14, background: 'rgba(255,255,255,0.05)', borderRadius: 4, overflow: 'hidden' }}>
                <motion.div
                  initial={{ width: 0 }}
                  animate={{ width: `${(d.value / max) * 100}%` }}
                  transition={{ duration: 0.7, delay: i * 0.05 }}
                  style={{ height: '100%', background: d.color ?? accentColor }}
                />
              </div>
              <span style={{ fontSize: 10, color: accentColor, fontWeight: 700, minWidth: 48, textAlign: 'right' }}>
                {d.value}{unit ?? ''}
              </span>
            </div>
          ))}
        </div>
      )}
      {kind === 'line' && (
        <svg width="100%" height="80" viewBox="0 0 200 80" preserveAspectRatio="none">
          <polyline
            fill="none" stroke={accentColor} strokeWidth="2"
            points={data.map((d, i) => `${(i / Math.max(data.length-1, 1)) * 195 + 2.5},${75 - (d.value / max) * 70}`).join(' ')}
          />
          {data.map((d, i) => (
            <circle key={i} cx={(i / Math.max(data.length-1, 1)) * 195 + 2.5} cy={75 - (d.value / max) * 70} r="2" fill={accentColor} />
          ))}
        </svg>
      )}
      {kind === 'pie' && (
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <PieSVG data={data} accentColor={accentColor} />
          <div style={{ display: 'flex', flexDirection: 'column', gap: 3, fontSize: 10 }}>
            {data.map((d, i) => (
              <div key={i} style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                <span style={{ width: 8, height: 8, borderRadius: 2, background: d.color ?? pieColor(i, accentColor) }} />
                <span style={{ color: 'rgba(255,255,255,0.75)' }}>{d.label}</span>
                <span style={{ color: accentColor, fontWeight: 700, marginLeft: 'auto' }}>{d.value}{unit ?? ''}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

function PieSVG({ data, accentColor }: { data: Array<{label: string; value: number; color?: string}>; accentColor: string }) {
  const total = data.reduce((s, d) => s + d.value, 0) || 1
  let cumulative = 0
  return (
    <svg width="64" height="64" viewBox="0 0 32 32">
      {data.map((d, i) => {
        const pct = d.value / total
        const start = cumulative
        cumulative += pct
        return (
          <circle key={i}
            r="16" cx="16" cy="16"
            fill="transparent"
            stroke={d.color ?? pieColor(i, accentColor)}
            strokeWidth="32"
            strokeDasharray={`${pct * 100} ${100 - pct * 100}`}
            strokeDashoffset={`${25 - start * 100}`}
            style={{ transformOrigin: 'center', transition: 'stroke-dashoffset 0.5s' }}
          />
        )
      })}
    </svg>
  )
}

function pieColor(i: number, accent: string): string {
  const palette = [accent, '#22c55e', '#3b82f6', '#f59e0b', '#ec4899', '#a855f7', '#06b6d4', '#ef4444']
  return palette[i % palette.length]
}

/* ─── Image ────────────────────────────────────────────────── */
function ImageBlock({ url, alt, caption }: Extract<Block, { type: 'image' }>) {
  return (
    <div>
      <img src={url} alt={alt ?? ''} style={{ width: '100%', maxHeight: 200, objectFit: 'contain', borderRadius: 6, background: 'rgba(0,0,0,0.3)' }} />
      {caption && <div style={{ fontSize: 9, color: 'rgba(255,255,255,0.5)', marginTop: 3, textAlign: 'center' }}>{caption}</div>}
    </div>
  )
}

/* ─── File ─────────────────────────────────────────────────── */
function FileBlock({ name, url, sizeKB, mime, accentColor }: Extract<Block, { type: 'file' }> & { accentColor: string }) {
  const ext = name.split('.').pop()?.toUpperCase() ?? 'FILE'
  const icon = mime?.startsWith('image/') ? '🖼️' : mime?.includes('pdf') ? '📕' : mime?.includes('excel') ? '📗' : mime?.includes('word') ? '📘' : '📄'
  return (
    <a href={url} download={name} style={{
      display: 'flex', alignItems: 'center', gap: 10, textDecoration: 'none',
      padding: '8px 10px', background: 'rgba(255,255,255,0.04)',
      border: `1px solid ${accentColor}33`, borderRadius: 8,
    }}>
      <span style={{ fontSize: 24 }}>{icon}</span>
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ fontSize: 11, fontWeight: 700, color: accentColor, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{name}</div>
        <div style={{ fontSize: 9, color: 'rgba(255,255,255,0.4)' }}>
          {ext}{sizeKB ? ` · ${sizeKB < 1024 ? sizeKB + ' KB' : (sizeKB/1024).toFixed(1) + ' MB'}` : ''}
        </div>
      </div>
      <span style={{ color: accentColor, fontSize: 16 }}>⬇</span>
    </a>
  )
}

/* ─── Action (클릭 가능 명령 칩) ──────────────────────────── */
function ActionBlock({ label, command, icon, variant = 'default', accentColor, onAction }: Extract<Block, { type: 'action' }> & { accentColor: string; onAction?: (cmd: string) => void }) {
  const colors = {
    primary: { bg: accentColor,           text: '#fff',                      border: accentColor },
    default: { bg: 'rgba(255,255,255,0.06)', text: 'rgba(255,255,255,0.85)', border: 'rgba(255,255,255,0.15)' },
    danger:  { bg: 'rgba(239,68,68,0.15)',  text: '#fca5a5',                  border: 'rgba(239,68,68,0.4)' },
  }[variant]
  return (
    <button
      onClick={() => onAction?.(command)}
      style={{
        display: 'inline-flex', alignItems: 'center', gap: 5,
        padding: '6px 12px', borderRadius: 8,
        background: colors.bg, border: `1px solid ${colors.border}`,
        color: colors.text, fontSize: 11, fontWeight: 700,
        cursor: 'pointer', alignSelf: 'flex-start',
        transition: 'all 0.15s',
      }}
      onMouseEnter={e => { e.currentTarget.style.transform = 'translateY(-1px)'; e.currentTarget.style.filter = 'brightness(1.15)' }}
      onMouseLeave={e => { e.currentTarget.style.transform = 'translateY(0)'; e.currentTarget.style.filter = 'brightness(1)' }}
    >
      {icon && <span>{icon}</span>}
      <span>{label}</span>
    </button>
  )
}

/* ─── Steps (진행률 체크리스트) ───────────────────────────── */
function StepsBlock({ items, accentColor }: Extract<Block, { type: 'steps' }> & { accentColor: string }) {
  const statusIcon = (s: string) => s === 'done' ? '✅' : s === 'running' ? '⏳' : s === 'failed' ? '❌' : '⚪'
  const statusColor = (s: string) =>
    s === 'done' ? '#22c55e' : s === 'running' ? accentColor : s === 'failed' ? '#ef4444' : 'rgba(255,255,255,0.4)'
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 4, padding: '6px 8px', background: 'rgba(255,255,255,0.03)', borderRadius: 8 }}>
      {items.map((s, i) => (
        <div key={i} style={{ display: 'flex', gap: 8, alignItems: 'center', fontSize: 11 }}>
          <span style={{ fontSize: 11 }}>{statusIcon(s.status)}</span>
          <span style={{
            color: statusColor(s.status),
            fontWeight: s.status === 'running' ? 700 : 500,
            textDecoration: s.status === 'done' ? 'line-through' : 'none',
            opacity: s.status === 'done' ? 0.7 : 1,
          }}>
            {s.label}
          </span>
          {s.detail && <span style={{ color: 'rgba(255,255,255,0.4)', fontSize: 10 }}>· {s.detail}</span>}
        </div>
      ))}
    </div>
  )
}
