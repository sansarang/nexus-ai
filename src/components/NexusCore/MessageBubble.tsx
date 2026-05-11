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

export function MessageBubble({
  message,
  onStepConfirm,
}: {
  message: Message
  onStepConfirm: (step: NexusStep, msgId: string) => void
}) {
  const isUser = message.role === 'user'
  const emotionColor = message.emotion ? EMOTION_COLORS[message.emotion] : EMOTION_COLORS.neutral

  if (isUser) {
    return (
      <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 12 }}>
        <div style={{ maxWidth: '70%' }}>
          <div
            style={{
              background: 'var(--accent-primary)',
              color: '#fff',
              borderRadius: '18px 18px 4px 18px',
              padding: '10px 14px',
              fontSize: 14,
              lineHeight: 1.5,
            }}
          >
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
      {/* Small avatar circle */}
      <div
        style={{
          width: 28,
          height: 28,
          borderRadius: '50%',
          background: `${emotionColor}22`,
          border: `1.5px solid ${emotionColor}`,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          fontSize: 12,
          flexShrink: 0,
          marginTop: 2,
        }}
      >
        ◉
      </div>

      <div style={{ maxWidth: '75%' }}>
        <div
          style={{
            background: 'var(--bg-elevated)',
            borderRadius: '4px 18px 18px 18px',
            border: `1px solid ${emotionColor}44`,
            padding: '10px 14px',
            fontSize: 14,
            lineHeight: 1.5,
            color: 'var(--text-primary)',
          }}
        >
          {message.text}

          {message.actionDone && (
            <div style={{ marginTop: 6, fontSize: 12, color: '#22c55e' }}>✅ 완료</div>
          )}

          {/* Pending steps requiring confirmation */}
          {message.pendingSteps && message.pendingSteps.length > 0 && (
            <div style={{ marginTop: 10, display: 'flex', flexDirection: 'column', gap: 6 }}>
              {message.pendingSteps.map((step, i) => (
                <div
                  key={i}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    background: 'var(--bg-surface)',
                    borderRadius: 8,
                    padding: '6px 10px',
                    border: '1px solid var(--border-subtle)',
                  }}
                >
                  <span style={{ fontSize: 12, color: 'var(--text-secondary)' }}>
                    {step.action}
                  </span>
                  <div style={{ display: 'flex', gap: 6 }}>
                    <button
                      onClick={() => onStepConfirm(step, message.id)}
                      style={{
                        padding: '3px 10px',
                        borderRadius: 6,
                        border: 'none',
                        background: 'var(--accent-primary)',
                        color: '#fff',
                        fontSize: 12,
                        cursor: 'pointer',
                      }}
                    >
                      확인
                    </button>
                    <button
                      onClick={() => {/* cancel handled at parent */}}
                      style={{
                        padding: '3px 10px',
                        borderRadius: 6,
                        border: '1px solid var(--border-default)',
                        background: 'transparent',
                        color: 'var(--text-secondary)',
                        fontSize: 12,
                        cursor: 'pointer',
                      }}
                    >
                      취소
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        <div style={{ marginTop: 4, fontSize: 11, color: 'var(--text-muted)' }}>
          {formatTime(message.timestamp)}
        </div>
      </div>
    </div>
  )
}
