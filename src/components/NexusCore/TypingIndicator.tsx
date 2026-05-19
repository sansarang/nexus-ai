import { useState, useEffect } from 'react'

const LOADING_MESSAGES = [
  '생각하는 중...',
  '검색하는 중...',
  '답변 준비 중...',
  '분석하는 중...',
  '처리하는 중...',
]

export function TypingIndicator({ message }: { message?: string }) {
  const [msgIdx, setMsgIdx] = useState(0)
  const [elapsed, setElapsed] = useState(0)

  useEffect(() => {
    const t = setInterval(() => {
      setMsgIdx(i => (i + 1) % LOADING_MESSAGES.length)
      setElapsed(s => s + 1)
    }, 2000)
    return () => clearInterval(t)
  }, [])

  const displayMsg = message ?? LOADING_MESSAGES[msgIdx]

  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 12 }}>
      {/* Avatar dot — 분석 중 파동 */}
      <div style={{
        position: 'relative', width: 28, height: 28, flexShrink: 0,
      }}>
        <div style={{
          width: 28, height: 28, borderRadius: '50%',
          background: 'rgba(79,126,247,0.15)',
          border: '1.5px solid #4f7ef7',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          fontSize: 12,
        }}>◉</div>
        <div style={{
          position: 'absolute', inset: -4,
          borderRadius: '50%',
          border: '1.5px solid #4f7ef7',
          animation: 'ti-ring 1.5s ease-out infinite',
          pointerEvents: 'none',
        }} />
      </div>

      <div style={{
        background: 'var(--bg-elevated)',
        borderRadius: '4px 18px 18px 18px',
        border: '1px solid rgba(79,126,247,0.3)',
        padding: '10px 16px',
        display: 'flex', alignItems: 'center', gap: 10,
        minWidth: 140,
      }}>
        {/* 점 3개 */}
        <div style={{ display: 'flex', gap: 4, alignItems: 'center' }}>
          {[0, 1, 2].map(i => (
            <div key={i} style={{
              width: 6, height: 6, borderRadius: '50%',
              background: 'var(--accent-primary)',
              animation: 'typing-bounce 1.2s ease-in-out infinite',
              animationDelay: `${i * 0.2}s`,
            }} />
          ))}
        </div>

        {/* 상태 메시지 */}
        <span style={{
          fontSize: 11, color: 'rgba(255,255,255,0.4)',
          animation: 'ti-fade 2s ease-in-out infinite',
        }}>
          {displayMsg}
          {elapsed >= 5 && <span style={{ color: 'rgba(255,255,255,0.25)', marginLeft: 4 }}>({elapsed}s)</span>}
        </span>
      </div>

      <style>{`
        @keyframes typing-bounce {
          0%,60%,100% { transform:translateY(0); opacity:0.5; }
          30% { transform:translateY(-6px); opacity:1; }
        }
        @keyframes ti-ring {
          0%   { transform:scale(1); opacity:0.6; }
          100% { transform:scale(1.6); opacity:0; }
        }
        @keyframes ti-fade {
          0%,100% { opacity:0.4; }
          50%     { opacity:0.7; }
        }
      `}</style>
    </div>
  )
}
