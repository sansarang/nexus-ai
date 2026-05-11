export function SendButton({ onClick, disabled }: { onClick: () => void; disabled: boolean }) {
  return (
    <button
      onClick={onClick}
      disabled={disabled}
      title="전송"
      style={{
        width: 40,
        height: 40,
        borderRadius: '50%',
        border: 'none',
        background: disabled ? 'var(--bg-elevated)' : 'var(--accent-primary)',
        color: disabled ? 'var(--text-muted)' : '#fff',
        fontSize: 16,
        cursor: disabled ? 'not-allowed' : 'pointer',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        transition: 'all 0.2s ease',
        flexShrink: 0,
        opacity: disabled ? 0.5 : 1,
      }}
    >
      ➤
    </button>
  )
}
