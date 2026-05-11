interface SearchResult { title?: string; snippet?: string; url?: string }

export function SearchResults({ data }: { data: unknown }) {
  const items: SearchResult[] = Array.isArray(data) ? (data as SearchResult[]) : []

  if (items.length === 0) {
    return (
      <div style={{ padding: 16, color: 'var(--text-muted)', fontSize: 13, textAlign: 'center' }}>
        검색 결과가 없습니다.
      </div>
    )
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 4, padding: 8 }}>
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
          <div style={{ fontSize: 13, fontWeight: 600, color: 'var(--accent-primary)', marginBottom: 3 }}>
            {item.title ?? '제목 없음'}
          </div>
          {item.snippet && (
            <div style={{ fontSize: 12, color: 'var(--text-secondary)', marginBottom: 4, lineHeight: 1.4 }}>
              {item.snippet}
            </div>
          )}
          {item.url && (
            <div style={{ fontSize: 11, color: 'var(--text-muted)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
              🔗 {item.url}
            </div>
          )}
        </div>
      ))}
    </div>
  )
}
