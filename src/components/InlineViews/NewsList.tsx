interface NewsItem { title?: string; source?: string; time?: string; url?: string }

export function NewsList({ data }: { data: unknown }) {
  const items: NewsItem[] = Array.isArray(data) ? (data as NewsItem[]) : []

  if (items.length === 0) {
    return (
      <div style={{ padding: 16, color: 'var(--text-muted)', fontSize: 13, textAlign: 'center' }}>
        📰 뉴스를 불러오는 중입니다.
      </div>
    )
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 2, padding: 8 }}>
      {items.map((item, i) => (
        <div
          key={i}
          style={{
            padding: '10px 12px',
            borderRadius: 8,
            background: 'var(--bg-elevated)',
            border: '1px solid var(--border-subtle)',
          }}
        >
          <div style={{ fontSize: 13, color: 'var(--text-primary)', marginBottom: 4, lineHeight: 1.4 }}>
            {item.title ?? '제목 없음'}
          </div>
          <div style={{ display: 'flex', gap: 8, fontSize: 11 }}>
            <span style={{ color: 'var(--accent-primary)' }}>{item.source}</span>
            <span style={{ color: 'var(--text-muted)' }}>{item.time}</span>
          </div>
        </div>
      ))}
    </div>
  )
}
