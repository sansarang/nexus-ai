interface BusArrival { route?: string; minutes?: number; station?: string }

export function BusCard({ data }: { data: unknown }) {
  const items: BusArrival[] = Array.isArray(data) ? (data as BusArrival[]) : []

  if (items.length === 0) {
    return (
      <div style={{ padding: 16, color: 'var(--text-muted)', fontSize: 13, textAlign: 'center' }}>
        🚌 도착 정보가 없습니다.
      </div>
    )
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 4, padding: 8 }}>
      {items.map((item, i) => (
        <div
          key={i}
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            padding: '10px 14px',
            borderRadius: 8,
            background: 'var(--bg-elevated)',
            border: '1px solid var(--border-subtle)',
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
            <span style={{ fontSize: 18 }}>🚌</span>
            <div>
              <div style={{ fontSize: 14, fontWeight: 600, color: 'var(--text-primary)' }}>{item.route ?? '--'}</div>
              {item.station && <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>{item.station}</div>}
            </div>
          </div>
          <div style={{ fontSize: 18, fontWeight: 700, color: (item.minutes ?? 99) <= 3 ? '#ef4444' : '#22c55e' }}>
            {item.minutes != null ? `${item.minutes}분` : '--'}
          </div>
        </div>
      ))}
    </div>
  )
}
