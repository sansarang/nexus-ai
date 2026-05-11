interface EmailItem { sender?: string; subject?: string; preview?: string; date?: string; unread?: boolean }

export function EmailList({ data }: { data: unknown }) {
  const items: EmailItem[] = Array.isArray(data) ? (data as EmailItem[]) : []

  if (items.length === 0) {
    return (
      <div style={{ padding: 16, color: 'var(--text-muted)', fontSize: 13, textAlign: 'center' }}>
        📧 이메일이 없습니다.
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
            background: item.unread ? 'rgba(79,126,247,0.08)' : 'var(--bg-elevated)',
            border: `1px solid ${item.unread ? 'rgba(79,126,247,0.3)' : 'var(--border-subtle)'}`,
          }}
        >
          <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 2 }}>
            <span style={{ fontSize: 13, fontWeight: item.unread ? 700 : 400, color: 'var(--text-primary)' }}>
              {item.sender ?? '발신자 없음'}
            </span>
            <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>{item.date}</span>
          </div>
          <div style={{ fontSize: 12, color: 'var(--text-secondary)', marginBottom: 2 }}>
            {item.subject ?? '제목 없음'}
          </div>
          {item.preview && (
            <div style={{ fontSize: 11, color: 'var(--text-muted)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
              {item.preview}
            </div>
          )}
        </div>
      ))}
    </div>
  )
}
