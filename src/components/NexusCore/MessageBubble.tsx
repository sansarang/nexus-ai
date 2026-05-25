import { useState, useEffect, useRef } from 'react'
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

/* 코드 블록 컴포넌트 (복사 버튼 + 언어 레이블) */
function CodeBlock({ code, lang }: { code: string; lang?: string }) {
  const [copied, setCopied] = useState(false)
  const handleCopy = () => {
    navigator.clipboard.writeText(code).then(() => {
      setCopied(true); setTimeout(() => setCopied(false), 1500)
    })
  }
  return (
    <div style={{ position: 'relative', margin: '8px 0', borderRadius: 8, overflow: 'hidden', border: '1px solid rgba(255,255,255,0.08)' }}>
      {/* 상단 바: 언어 + 복사 */}
      <div style={{
        display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        padding: '4px 10px',
        background: 'rgba(0,0,0,0.4)',
        borderBottom: '1px solid rgba(255,255,255,0.06)',
      }}>
        <span style={{ fontSize: 11, color: 'rgba(255,255,255,0.4)', fontFamily: 'monospace' }}>
          {lang || 'code'}
        </span>
        <button
          onClick={handleCopy}
          style={{
            background: 'none', border: 'none', cursor: 'pointer',
            fontSize: 11, color: copied ? '#22c55e' : 'rgba(255,255,255,0.4)',
            padding: '2px 6px', borderRadius: 4,
            transition: 'color 0.2s',
          }}
        >{copied ? '✓ 복사됨' : '복사'}</button>
      </div>
      <pre style={{
        background: 'rgba(0,0,0,0.3)', margin: 0,
        padding: '10px 12px', overflowX: 'auto',
        fontSize: 12, fontFamily: 'monospace', lineHeight: 1.6,
      }}>
        <code style={{ color: '#e2e8f0' }}>{code}</code>
      </pre>
    </div>
  )
}

/* URL을 클릭 가능한 링크로 변환 */
function linkify(text: string): React.ReactNode[] {
  const urlPattern = /https?:\/\/[^\s<>"'）)】\]]+/g
  const parts: React.ReactNode[] = []
  let lastIdx = 0
  let match
  while ((match = urlPattern.exec(text)) !== null) {
    if (match.index > lastIdx) parts.push(text.slice(lastIdx, match.index))
    const url = match[0]
    parts.push(
      <a
        key={match.index}
        href={url}
        onClick={async e => {
          e.preventDefault()
          try {
            const { open } = await import('@tauri-apps/plugin-shell')
            await open(url)
          } catch { window.open(url, '_blank') }
        }}
        style={{ color: '#60a5fa', textDecoration: 'underline', cursor: 'pointer', wordBreak: 'break-all' }}
      >{url}</a>
    )
    lastIdx = match.index + url.length
  }
  if (lastIdx < text.length) parts.push(text.slice(lastIdx))
  return parts
}

/* 인라인 서식 (bold, italic, inline code) + URL linkify */
function inlineFormat(line: string, key: number): React.ReactNode {
  const parts: React.ReactNode[] = []
  let i = 0
  let buf = ''

  const flush = () => {
    if (!buf) return
    const linked = linkify(buf)
    parts.push(...linked)
    buf = ''
  }

  while (i < line.length) {
    // inline code
    if (line[i] === '`' && line[i + 1] !== '`') {
      const end = line.indexOf('`', i + 1)
      if (end !== -1) {
        flush()
        parts.push(<code key={`c${i}`} style={{ background: 'rgba(255,255,255,0.1)', borderRadius: 3, padding: '1px 5px', fontFamily: 'monospace', fontSize: 12 }}>{line.slice(i + 1, end)}</code>)
        i = end + 1; continue
      }
    }
    // bold **text**
    if (line[i] === '*' && line[i + 1] === '*') {
      const end = line.indexOf('**', i + 2)
      if (end !== -1) {
        flush()
        parts.push(<strong key={`b${i}`}>{line.slice(i + 2, end)}</strong>)
        i = end + 2; continue
      }
    }
    // italic *text* (single asterisk)
    if (line[i] === '*' && line[i + 1] !== '*' && line[i - 1] !== '*') {
      const end = line.indexOf('*', i + 1)
      if (end !== -1 && line[end + 1] !== '*') {
        flush()
        parts.push(<em key={`em${i}`} style={{ fontStyle: 'italic', opacity: 0.85 }}>{line.slice(i + 1, end)}</em>)
        i = end + 1; continue
      }
    }
    buf += line[i]; i++
  }
  flush()
  return <span key={key}>{parts}</span>
}

/* 마크다운 렌더러 */
function renderMarkdown(text: string): React.ReactNode[] {
  const lines = text.split('\n')
  const nodes: React.ReactNode[] = []
  let codeBlock = false
  let codeLines: string[] = []
  let codeLang = ''
  let codeKey = 0

  lines.forEach((line, idx) => {
    if (line.startsWith('```')) {
      if (!codeBlock) {
        codeBlock = true
        codeLang = line.slice(3).trim()
        codeLines = []
      } else {
        nodes.push(<CodeBlock key={`code-${codeKey++}`} code={codeLines.join('\n')} lang={codeLang || undefined} />)
        codeBlock = false
        codeLang = ''
      }
      return
    }
    if (codeBlock) { codeLines.push(line); return }

    if (!line.trim()) { nodes.push(<div key={idx} style={{ height: 6 }} />); return }

    // 헤더
    if (/^#{1,3}\s/.test(line)) {
      const level = line.match(/^(#+)/)?.[1].length ?? 1
      const content = line.replace(/^#+\s/, '')
      const sizes = [17, 15, 14]
      nodes.push(
        <div key={idx} style={{ fontWeight: 700, fontSize: sizes[Math.min(level - 1, 2)], marginTop: level === 1 ? 14 : 10, marginBottom: 4, color: 'var(--text-primary)', borderBottom: level === 1 ? '1px solid rgba(255,255,255,0.08)' : 'none', paddingBottom: level === 1 ? 4 : 0 }}>
          {inlineFormat(content, idx)}
        </div>
      )
      return
    }
    // 불릿 리스트
    if (/^[-*]\s/.test(line)) {
      nodes.push(
        <div key={idx} style={{ display: 'flex', gap: 6, marginTop: 2 }}>
          <span style={{ color: 'var(--accent-primary)', flexShrink: 0 }}>•</span>
          <span>{inlineFormat(line.replace(/^[-*]\s/, ''), idx)}</span>
        </div>
      )
      return
    }
    // 번호 목록
    if (/^\d+\.\s/.test(line)) {
      const num = line.match(/^(\d+)/)?.[1]
      nodes.push(
        <div key={idx} style={{ display: 'flex', gap: 6, marginTop: 2 }}>
          <span style={{ color: 'var(--accent-primary)', flexShrink: 0, minWidth: 16 }}>{num}.</span>
          <span>{inlineFormat(line.replace(/^\d+\.\s/, ''), idx)}</span>
        </div>
      )
      return
    }
    // 구분선
    if (/^---+$/.test(line.trim()) || /^===+$/.test(line.trim())) {
      nodes.push(<hr key={idx} style={{ border: 'none', borderTop: '1px solid rgba(255,255,255,0.1)', margin: '8px 0' }} />)
      return
    }
    // 테이블
    if (line.includes('|') && line.trim().startsWith('|')) {
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
                borderRight: ci < cells.length - 1 ? '1px solid rgba(255,255,255,0.08)' : 'none',
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

/* 파일 분석 결과 카드 헤더 */
function FileAnalysisHeader({ fileInfo }: { fileInfo: NonNullable<Message['fileInfo']> }) {
  const icons = { image: '🖼️', video: '🎬', document: '📄', spreadsheet: '📊', other: '📎' }
  const labels = { image: '이미지 분석', video: '영상 분석', document: '문서 분석', spreadsheet: '스프레드시트 분석', other: '파일 분석' }
  return (
    <div style={{
      display: 'flex', alignItems: 'center', gap: 6, marginBottom: 8,
      paddingBottom: 8, borderBottom: '1px solid rgba(255,255,255,0.08)',
    }}>
      <span style={{ fontSize: 14 }}>{icons[fileInfo.type]}</span>
      <div>
        <div style={{ fontSize: 11, color: 'var(--text-muted)', marginBottom: 1 }}>{labels[fileInfo.type]}</div>
        <div style={{ fontSize: 12, fontWeight: 600, color: 'var(--text-secondary)' }}>{fileInfo.name}</div>
      </div>
    </div>
  )
}

/* 타이핑 커서 */
function TypingCursor() {
  return (
    <span style={{
      display: 'inline-block', width: 2, height: '1em',
      background: 'var(--accent-primary)', marginLeft: 1,
      verticalAlign: 'text-bottom',
      animation: 'nexus-blink 1s step-end infinite',
    }} />
  )
}

export function MessageBubble({
  message,
  onStepConfirm,
}: {
  message: Message
  onStepConfirm: (step: NexusStep, msgId: string) => void
}) {
  const [copied, setCopied] = useState(false)
  const [displayText, setDisplayText] = useState(message.animate ? '' : message.text)
  const [isStreaming, setIsStreaming] = useState(message.animate ?? false)
  const animRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const isUser = message.role === 'user'
  const emotionColor = message.emotion ? EMOTION_COLORS[message.emotion] ?? EMOTION_COLORS.neutral : EMOTION_COLORS.neutral
  const hasMarkdown = !isUser && /[#*`\n|]/.test(message.text)

  /* 타이핑 애니메이션 */
  useEffect(() => {
    if (!message.animate || isUser) return
    const full = message.text
    if (full.length > 800) {
      // 긴 텍스트: 즉시 표시
      setDisplayText(full)
      setIsStreaming(false)
      return
    }
    let i = 0
    const chunkSize = Math.max(1, Math.ceil(full.length / 80)) // ~80 틱에 완성
    animRef.current = setInterval(() => {
      i += chunkSize
      if (i >= full.length) {
        setDisplayText(full)
        setIsStreaming(false)
        if (animRef.current) clearInterval(animRef.current)
      } else {
        setDisplayText(full.slice(0, i))
      }
    }, 18)
    return () => { if (animRef.current) clearInterval(animRef.current) }
  // animate가 true인 메시지에만 최초 1회 실행
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [message.id])

  /* 텍스트 업데이트 (스트리밍 중이 아닐 때) */
  useEffect(() => {
    if (!isStreaming) setDisplayText(message.text)
  }, [message.text, isStreaming])

  const handleCopy = () => {
    navigator.clipboard.writeText(message.text).then(() => {
      setCopied(true); setTimeout(() => setCopied(false), 1500)
    })
  }

  /* 사용자 메시지 */
  if (isUser) {
    return (
      <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 12 }}>
        <div style={{ maxWidth: '75%' }}>
          {/* 이미지 썸네일 */}
          {message.imageDataUrl && (
            <div style={{ marginBottom: 6, display: 'flex', justifyContent: 'flex-end' }}>
              <img
                src={message.imageDataUrl}
                alt="첨부 이미지"
                style={{
                  maxWidth: 200, maxHeight: 160,
                  borderRadius: 12, objectFit: 'cover',
                  border: '1px solid rgba(255,255,255,0.1)',
                }}
              />
            </div>
          )}
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

  /* AI 응답 메시지 */
  return (
    <div style={{ display: 'flex', alignItems: 'flex-start', gap: 8, marginBottom: 12 }}>
      {/* 감정 아바타 점 */}
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
          {/* 파일 분석 카드 헤더 */}
          {message.fileInfo && <FileAnalysisHeader fileInfo={message.fileInfo} />}

          {/* 본문 */}
          {hasMarkdown
            ? <>{renderMarkdown(displayText)}{isStreaming && <TypingCursor />}</>
            : <>{displayText}{isStreaming && <TypingCursor />}</>
          }

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

      {/* 커서 blink CSS */}
      <style>{`@keyframes nexus-blink { 0%,100%{opacity:1} 50%{opacity:0} }`}</style>
    </div>
  )
}
