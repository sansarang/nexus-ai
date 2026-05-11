interface AirData { pm25?: number; pm10?: number; grade?: string }

function airColor(val: number): string {
  if (val <= 15) return '#22c55e'
  if (val <= 35) return '#f59e0b'
  return '#ef4444'
}

export function AirCard({ data }: { data: unknown }) {
  const d = (data ?? {}) as AirData

  return (
    <div style={{ padding: 16 }}>
      <div style={{ display: 'flex', gap: 12, marginBottom: 12 }}>
        <div style={{ flex: 1, padding: '10px 14px', borderRadius: 10, background: 'var(--bg-elevated)', border: '1px solid var(--border-subtle)', textAlign: 'center' }}>
          <div style={{ fontSize: 11, color: 'var(--text-muted)', marginBottom: 4 }}>PM2.5</div>
          <div style={{ fontSize: 24, fontWeight: 800, color: airColor(d.pm25 ?? 0) }}>
            {d.pm25 ?? '--'}
          </div>
          <div style={{ fontSize: 10, color: 'var(--text-muted)' }}>µg/m³</div>
        </div>
        <div style={{ flex: 1, padding: '10px 14px', borderRadius: 10, background: 'var(--bg-elevated)', border: '1px solid var(--border-subtle)', textAlign: 'center' }}>
          <div style={{ fontSize: 11, color: 'var(--text-muted)', marginBottom: 4 }}>PM10</div>
          <div style={{ fontSize: 24, fontWeight: 800, color: airColor(d.pm10 ?? 0) }}>
            {d.pm10 ?? '--'}
          </div>
          <div style={{ fontSize: 10, color: 'var(--text-muted)' }}>µg/m³</div>
        </div>
      </div>
      {d.grade && (
        <div style={{ textAlign: 'center', fontSize: 14, color: airColor(d.pm25 ?? 0), fontWeight: 600 }}>
          대기질: {d.grade}
        </div>
      )}
    </div>
  )
}
