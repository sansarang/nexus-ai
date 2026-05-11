interface SportsItem { team?: string; score?: string; opponent?: string; date?: string; result?: string }

export function SportsCard({ data }: { data: unknown }) {
  const items: SportsItem[] = Array.isArray(data) ? (data as SportsItem[]) : []

  if (items.length === 0) {
    return (
      <div style={{ padding: 16, color: 'var(--text-muted)', fontSize: 13, textAlign: 'center' }}>
        ⚽ 경기 정보가 없습니다.
      </div>
    )
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 4, padding: 8 }}>
      {items.map((item, i) => (
        <div
          key={i}
          style={{
            padding: '10px 14px',
            borderRadius: 8,
            background: 'var(--bg-elevated)',
            border: '1px solid var(--border-subtle)',
          }}
        >
          <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
            <span style={{ fontSize: 14, fontWeight: 600, color: 'var(--text-primary)' }}>{item.team ?? '--'}</span>
            <span style={{ fontSize: 16, fontWeight: 800, color: 'var(--accent-primary)' }}>{item.score ?? '--'}</span>
          </div>
          {item.opponent && (
            <div style={{ fontSize: 12, color: 'var(--text-muted)' }}>vs {item.opponent}</div>
          )}
          {item.date && (
            <div style={{ fontSize: 11, color: 'var(--text-muted)', marginTop: 2 }}>{item.date}</div>
          )}
        </div>
      ))}
    </div>
  )
}
