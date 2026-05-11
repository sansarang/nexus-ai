interface PhotoItem { name?: string; path?: string; size?: number }

export function PhotoGrid({ data }: { data: unknown }) {
  const items: PhotoItem[] = Array.isArray(data) ? (data as PhotoItem[]) : []

  if (items.length === 0) {
    return (
      <div style={{ padding: 16, color: 'var(--text-muted)', fontSize: 13, textAlign: 'center' }}>
        📷 사진이 없거나 데이터를 불러오는 중입니다.
      </div>
    )
  }

  return (
    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(80px, 1fr))', gap: 8, padding: 12 }}>
      {items.map((item, i) => (
        <div
          key={i}
          style={{
            background: 'var(--bg-elevated)',
            borderRadius: 8,
            padding: 8,
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            gap: 4,
            border: '1px solid var(--border-subtle)',
          }}
        >
          <div style={{ fontSize: 28 }}>📷</div>
          <div style={{ fontSize: 10, color: 'var(--text-secondary)', textAlign: 'center', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', width: '100%' }}>
            {item.name ?? `photo_${i + 1}`}
          </div>
          {item.size != null && (
            <div style={{ fontSize: 9, color: 'var(--text-muted)' }}>
              {(item.size / 1024 / 1024).toFixed(1)} MB
            </div>
          )}
        </div>
      ))}
    </div>
  )
}
