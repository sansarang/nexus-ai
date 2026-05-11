export function VoiceButton({
  listening,
  onToggle,
}: {
  listening: boolean
  onToggle: () => void
}) {
  return (
    <button
      onClick={onToggle}
      title={listening ? '음성 입력 중지' : '음성 입력 시작'}
      style={{
        width: 40,
        height: 40,
        borderRadius: '50%',
        border: 'none',
        background: listening ? '#ef4444' : 'var(--bg-elevated)',
        color: listening ? '#fff' : 'var(--text-secondary)',
        fontSize: 18,
        cursor: 'pointer',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        transition: 'all 0.2s ease',
        boxShadow: listening ? '0 0 12px rgba(239,68,68,0.5)' : 'none',
        flexShrink: 0,
      }}
    >
      🎤
    </button>
  )
}
