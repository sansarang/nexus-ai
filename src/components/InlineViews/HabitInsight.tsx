interface HabitData { peak_hours?: string[]; productivity?: number; patterns?: string[] }

export function HabitInsight({ data }: { data: unknown }) {
  const d = (data ?? {}) as HabitData

  return (
    <div style={{ padding: 16 }}>
      {d.productivity != null && (
        <div style={{ marginBottom: 14 }}>
          <div style={{ fontSize: 11, color: 'var(--text-muted)', marginBottom: 4 }}>생산성 지수</div>
          <div style={{ height: 8, borderRadius: 4, background: 'var(--bg-elevated)' }}>
            <div style={{ height: '100%', borderRadius: 4, background: 'var(--accent-primary)', width: `${d.productivity}%` }} />
          </div>
          <div style={{ fontSize: 12, color: 'var(--accent-primary)', marginTop: 3 }}>{d.productivity}%</div>
        </div>
      )}

      {d.peak_hours && d.peak_hours.length > 0 && (
        <div style={{ marginBottom: 12 }}>
          <div style={{ fontSize: 11, color: 'var(--text-muted)', marginBottom: 6 }}>집중 피크 시간</div>
          <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
            {d.peak_hours.map(h => (
              <span key={h} style={{ padding: '3px 10px', borderRadius: 12, background: 'rgba(79,126,247,0.15)', color: 'var(--accent-primary)', fontSize: 12 }}>{h}</span>
            ))}
          </div>
        </div>
      )}

      {d.patterns && d.patterns.length > 0 && (
        <div>
          <div style={{ fontSize: 11, color: 'var(--text-muted)', marginBottom: 6 }}>패턴 분석</div>
          {d.patterns.map((p, i) => (
            <div key={i} style={{ fontSize: 12, color: 'var(--text-secondary)', marginBottom: 3 }}>• {p}</div>
          ))}
        </div>
      )}

      {!d.productivity && !d.peak_hours && !d.patterns && (
        <div style={{ color: 'var(--text-muted)', fontSize: 13, textAlign: 'center' }}>
          📊 습관 데이터를 분석 중입니다.
        </div>
      )}
    </div>
  )
}
