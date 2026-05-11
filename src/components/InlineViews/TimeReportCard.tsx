interface AppUsage { name?: string; minutes?: number }
interface TimeReportData { score?: number; focus_hours?: number; apps?: AppUsage[] }

export function TimeReportCard({ data }: { data: unknown }) {
  const d = (data ?? {}) as TimeReportData
  const apps = d.apps ?? []
  const maxMin = Math.max(...apps.map(a => a.minutes ?? 0), 1)

  return (
    <div style={{ padding: 16 }}>
      <div style={{ display: 'flex', gap: 16, marginBottom: 14 }}>
        <div style={{ flex: 1, padding: '10px 14px', borderRadius: 10, background: 'var(--bg-elevated)', border: '1px solid var(--border-subtle)', textAlign: 'center' }}>
          <div style={{ fontSize: 24, fontWeight: 800, color: '#22c55e' }}>{d.score ?? 0}</div>
          <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>생산성 점수</div>
        </div>
        <div style={{ flex: 1, padding: '10px 14px', borderRadius: 10, background: 'var(--bg-elevated)', border: '1px solid var(--border-subtle)', textAlign: 'center' }}>
          <div style={{ fontSize: 24, fontWeight: 800, color: 'var(--accent-primary)' }}>
            {d.focus_hours?.toFixed(1) ?? '0.0'}h
          </div>
          <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>집중 시간</div>
        </div>
      </div>

      {apps.length > 0 && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
          {apps.map((app, i) => (
            <div key={i}>
              <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 3, fontSize: 12, color: 'var(--text-secondary)' }}>
                <span>{app.name ?? `앱 ${i + 1}`}</span>
                <span>{Math.round((app.minutes ?? 0) / 60)}h {(app.minutes ?? 0) % 60}m</span>
              </div>
              <div style={{ height: 4, borderRadius: 2, background: 'var(--bg-elevated)' }}>
                <div style={{
                  height: '100%',
                  borderRadius: 2,
                  background: 'var(--accent-primary)',
                  width: `${((app.minutes ?? 0) / maxMin) * 100}%`,
                  transition: 'width 0.5s ease',
                }} />
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
