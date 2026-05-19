import { useState } from 'react'
import type { Message, NexusStep, NexusEmotion } from '../../types/nexus'

const EMOTION_COLORS: Record<NexusEmotion, string> = {
  neutral: '#4f7ef7',
  concerned: '#f59e0b',
  happy: '#22c55e',
  alert: '#ef4444',
  humorous: '#a78bfa',
}

function formatTime(d: Date): string {
  return d.toLocaleTimeString('ko-KR', { hour: '2-digit', minute: '2-digit' })
}

/* 간단한 인라인 마크다운 렌더러 (외부 라이브러리 없이) */
function renderMarkdown(text: string): React.ReactNode[] {
  const lines = text.split('\n')
  const nodes: React.ReactNode[] = []
  let codeBlock = false
  let codeLines: string[] = []
  let codeKey = 0

  const inlineFormat = (line: string, key: number): React.ReactNode => {
    const parts: React.ReactNode[] = []
    let i = 0
    let buf = ''
    while (i < line.length) {
      // inline code
      if (line[i] === '`' && line[i + 1] !== '`') {
        const end = line.indexOf('`', i + 1)
        if (end !== -1) {
          if (buf) parts.push(buf); buf = ''
          parts.push(<code key={`c${i}`} style={{ background: 'rgba(255,255,255,0.1)', borderRadius: 3, padding: '1px 5px', fontFamily: 'monospace', fontSize: 12 }}>{line.slice(i + 1, end)}</code>)
          i = end + 1; continue
        }
      }
      // bold
      if (line[i] === '*' && line[i + 1] === '*') {
        const end = line.indexOf('**', i + 2)
        if (end !== -1) {
          if (buf) parts.push(buf); buf = ''
          parts.push(<strong key={`b${i}`}>{line.slice(i + 2, end)}</strong>)
          i = end + 2; continue
        }
      }
      buf += line[i]; i++
    }
    if (buf) parts.push(buf)
    return <span key={key}>{parts}</span>
  }

  lines.forEach((line, idx) => {
    if (line.startsWith('```')) {
      if (!codeBlock) {
        codeBlock = true; codeLines = []
      } else {
        const key = `code-${codeKey++}`
        nodes.push(
          <pre key={key} style={{
            background: 'rgba(0,0,0,0.35)', borderRadius: 8, padding: '10px 12px',
            overflowX: 'auto', fontSize: 12, fontFamily: 'monospace',
            margin: '6px 0', lineHeight: 1.6, border: '1px solid rgba(255,255,255,0.08)',
          }}>
            <code style={{ color: '#e2e8f0' }}>{codeLines.join('\n')}</code>
          </pre>
        )
        codeBlock = false
      }
      return
    }
    if (codeBlock) { codeLines.push(line); return }

    if (!line.trim()) { nodes.push(<div key={idx} style={{ height: 6 }} />); return }

    if (/^#{1,3}\s/.test(line)) {
      const level = line.match(/^(#+)/)?.[1].length ?? 1
      const content = line.replace(/^#+\s/, '')
      const sizes = [16, 15, 14]
      nodes.push(<div key={idx} style={{ fontWeight: 700, fontSize: sizes[Math.min(level - 1, 2)], marginTop: 10, marginBottom: 4, color: 'var(--text-primary)' }}>{content}</div>)
      return
    }
    if (/^[-*]\s/.test(line)) {
      nodes.push(<div key={idx} style={{ display: 'flex', gap: 6, marginTop: 2 }}><span style={{ color: 'var(--accent-primary)', flexShrink: 0 }}>•</span><span>{inlineFormat(line.replace(/^[-*]\s/, ''), idx)}</span></div>)
      return
    }
    if (/^\d+\.\s/.test(line)) {
      const num = line.match(/^(\d+)/)?.[1]
      nodes.push(<div key={idx} style={{ display: 'flex', gap: 6, marginTop: 2 }}><span style={{ color: 'var(--accent-primary)', flexShrink: 0, minWidth: 16 }}>{num}.</span><span>{inlineFormat(line.replace(/^\d+\.\s/, ''), idx)}</span></div>)
      return
    }
    if (line.startsWith('---') || line.startsWith('===')) {
      nodes.push(<hr key={idx} style={{ border: 'none', borderTop: '1px solid rgba(255,255,255,0.1)', margin: '8px 0' }} />)
      return
    }
    if (line.includes('|') && line.trim().startsWith('|')) {
      // 테이블 행
      const cells = line.split('|').filter((_, i, arr) => i > 0 && i < arr.length - 1)
      const isHeader = lines[idx + 1]?.includes('---')
      const isDivider = /^[\s|:-]+$/.test(line)
      if (!isDivider) {
        nodes.push(
          <div key={idx} style={{ display: 'flex', borderBottom: '1px solid rgba(255,255,255,0.08)' }}>
            {cells.map((cell, ci) => (
              <div key={ci} style={{
                flex: 1, padding: '4px 8px', fontSize: 12,
                fontWeight: isHeader ? 700 : 400,
                background: isHeader ? 'rgba(255,255,255,0.05)' : 'transparent',
              }}>{cell.trim()}</div>
            ))}
          </div>
        )
      }
      return
    }
    nodes.push(<div key={idx} style={{ lineHeight: 1.7, marginTop: 1 }}>{inlineFormat(line, idx)}</div>)
  })

  return nodes
}

export function MessageBubble({
  message,
  onStepConfirm,
}: {
  message: Message
  onStepConfirm: (step: NexusStep, msgId: string) => void
}) {
  const [copied, setCopied] = useState(false)
  const isUser = message.role === 'user'
  const emotionColor = message.emotion ? EMOTION_COLORS[message.emotion] ?? EMOTION_COLORS.neutral : EMOTION_COLORS.neutral
  const hasMarkdown = !isUser && /[#*`]|\n/.test(message.text)

  const handleCopy = () => {
    navigator.clipboard.writeText(message.text).then(() => {
      setCopied(true); setTimeout(() => setCopied(false), 1500)
    })
  }

  if (isUser) {
    return (
      <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 12 }}>
        <div style={{ maxWidth: '75%' }}>
          <div style={{
            background: 'var(--accent-primary)', color: '#fff',
            borderRadius: '18px 18px 4px 18px',
            padding: '10px 14px', fontSize: 14, lineHeight: 1.5,
            whiteSpace: 'pre-wrap',
          }}>
            {message.text}
          </div>
          <div style={{ textAlign: 'right', marginTop: 4, fontSize: 11, color: 'var(--text-muted)' }}>
            {formatTime(message.timestamp)}
          </div>
        </div>
      </div>
    )
  }

  return (
    <div style={{ display: 'flex', alignItems: 'flex-start', gap: 8, marginBottom: 12 }}>
      {/* Avatar dot */}
      <div style={{
        width: 28, height: 28, borderRadius: '50%',
        background: `${emotionColor}22`, border: `1.5px solid ${emotionColor}`,
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        fontSize: 12, flexShrink: 0, marginTop: 2,
        transition: 'border-color 0.3s, background 0.3s',
      }}>◉</div>

      <div style={{ maxWidth: '80%', minWidth: 0 }}>
        <div style={{
          background: 'var(--bg-elevated)',
          borderRadius: '4px 18px 18px 18px',
          border: `1px solid ${emotionColor}44`,
          padding: '10px 14px', fontSize: 14, lineHeight: 1.5,
          color: 'var(--text-primary)',
          transition: 'border-color 0.3s',
        }}>
          {hasMarkdown ? renderMarkdown(message.text) : message.text}

          {message.actionDone && (
            <div style={{ marginTop: 6, fontSize: 12, color: '#22c55e' }}>✅ 완료</div>
          )}

          {message.pendingSteps && message.pendingSteps.length > 0 && (
            <div style={{ marginTop: 10, display: 'flex', flexDirection: 'column', gap: 6 }}>
              {message.pendingSteps.map((step, i) => (
                <div key={i} style={{
                  display: 'flex', alignItems: 'center', justifyContent: 'space-between',
                  background: 'var(--bg-surface)', borderRadius: 8, padding: '6px 10px',
                  border: '1px solid var(--border-subtle)',
                }}>
                  <span style={{ fontSize: 12, color: 'var(--text-secondary)' }}>{step.action}</span>
                  <div style={{ display: 'flex', gap: 6 }}>
                    <button onClick={() => onStepConfirm(step, message.id)} style={{
                      padding: '3px 10px', borderRadius: 6, border: 'none',
                      background: 'var(--accent-primary)', color: '#fff', fontSize: 12, cursor: 'pointer',
                    }}>확인</button>
                    <button style={{
                      padding: '3px 10px', borderRadius: 6,
                      border: '1px solid var(--border-default)', background: 'transparent',
                      color: 'var(--text-secondary)', fontSize: 12, cursor: 'pointer',
                    }}>취소</button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginTop: 4 }}>
          <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>{formatTime(message.timestamp)}</span>
          {message.text.length > 80 && (
            <button onClick={handleCopy} style={{
              background: 'none', border: 'none', cursor: 'pointer',
              fontSize: 10, color: copied ? '#22c55e' : 'rgba(255,255,255,0.25)',
              padding: 0, transition: 'color 0.2s',
            }}>{copied ? '✓ 복사됨' : '복사'}</button>
          )}
        </div>
      </div>
    </div>
  )
}
