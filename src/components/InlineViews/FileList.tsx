interface FileItem { name?: string; path?: string; size?: number; date?: string; type?: string }

const FILE_ICONS: Record<string, string> = {
  pdf: '📄', doc: '📝', docx: '📝', xls: '📊', xlsx: '📊',
  jpg: '🖼️', png: '🖼️', mp4: '🎬', mp3: '🎵', zip: '📦', default: '📁',
}

function fileIcon(name?: string): string {
  const ext = name?.split('.').pop()?.toLowerCase() ?? ''
  return FILE_ICONS[ext] ?? FILE_ICONS.default
}

export function FileList({ data }: { data: unknown }) {
  const items: FileItem[] = Array.isArray(data) ? (data as FileItem[]) : []

  if (items.length === 0) {
    return (
      <div style={{ padding: 16, color: 'var(--text-muted)', fontSize: 13, textAlign: 'center' }}>
        파일 목록이 비어있습니다.
      </div>
    )
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 2, padding: 8 }}>
      {items.map((item, i) => (
        <div
          key={i}
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 10,
            padding: '8px 10px',
            borderRadius: 8,
            background: 'var(--bg-elevated)',
            border: '1px solid var(--border-subtle)',
          }}
        >
          <span style={{ fontSize: 18 }}>{fileIcon(item.name)}</span>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{ fontSize: 13, color: 'var(--text-primary)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
              {item.name ?? `파일 ${i + 1}`}
            </div>
            {item.path && (
              <div style={{ fontSize: 11, color: 'var(--text-muted)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                {item.path}
              </div>
            )}
          </div>
          <div style={{ textAlign: 'right', flexShrink: 0 }}>
            {item.size != null && (
              <div style={{ fontSize: 11, color: 'var(--text-secondary)' }}>
                {item.size > 1048576 ? `${(item.size / 1048576).toFixed(1)} MB` : `${(item.size / 1024).toFixed(0)} KB`}
              </div>
            )}
            {item.date && (
              <div style={{ fontSize: 10, color: 'var(--text-muted)' }}>{item.date}</div>
            )}
          </div>
        </div>
      ))}
    </div>
  )
}
