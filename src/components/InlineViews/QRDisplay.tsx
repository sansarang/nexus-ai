interface QRData { content?: string; success?: boolean }

export function QRDisplay({ data }: { data: unknown }) {
  const d = (data ?? {}) as QRData

  return (
    <div style={{ padding: 16, textAlign: 'center' }}>
      <div
        style={{
          width: 120,
          height: 120,
          margin: '0 auto 12px',
          background: 'var(--bg-elevated)',
          border: '2px solid var(--border-default)',
          borderRadius: 10,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          fontSize: 48,
        }}
      >
        ▦
      </div>
      {d.content && (
        <div style={{ fontSize: 12, color: 'var(--text-secondary)', wordBreak: 'break-all', padding: '0 8px' }}>
          {d.content}
        </div>
      )}
    </div>
  )
}
