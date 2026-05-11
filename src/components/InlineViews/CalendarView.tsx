interface CalEvent { time?: string; title?: string; location?: string }

export function CalendarView({ data }: { data: unknown }) {
  const events: CalEvent[] = Array.isArray(data) ? (data as CalEvent[]) : []

  if (events.length === 0) {
    return (
      <div style={{ padding: 16, color: 'var(--text-muted)', fontSize: 13, textAlign: 'center' }}>
        📅 오늘 일정이 없습니다.
      </div>
    )
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 4, padding: 8 }}>
      {events.map((ev, i) => (
        <div
          key={i}
          style={{
            display: 'flex',
            gap: 12,
            padding: '10px 12px',
            borderRadius: 8,
            background: 'var(--bg-elevated)',
            border: '1px solid var(--border-subtle)',
            borderLeft: '3px solid var(--accent-primary)',
          }}
        >
          <div style={{ fontSize: 12, color: 'var(--accent-primary)', fontWeight: 600, minWidth: 50 }}>
            {ev.time ?? '--:--'}
          </div>
          <div>
            <div style={{ fontSize: 13, color: 'var(--text-primary)' }}>{ev.title ?? '제목 없음'}</div>
            {ev.location && (
              <div style={{ fontSize: 11, color: 'var(--text-muted)', marginTop: 2 }}>
                📍 {ev.location}
              </div>
            )}
          </div>
        </div>
      ))}
    </div>
  )
}
