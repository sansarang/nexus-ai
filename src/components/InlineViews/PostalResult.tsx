interface PostalData { address?: string; postal_code?: string }

export function PostalResult({ data }: { data: unknown }) {
  const d = (data ?? {}) as PostalData

  return (
    <div style={{ padding: 16 }}>
      <div style={{ padding: '14px 18px', borderRadius: 10, background: 'var(--bg-elevated)', border: '1px solid var(--border-subtle)' }}>
        <div style={{ fontSize: 11, color: 'var(--text-muted)', marginBottom: 6 }}>주소</div>
        <div style={{ fontSize: 14, color: 'var(--text-primary)', marginBottom: 12 }}>
          {d.address ?? '주소 정보 없음'}
        </div>
        <div style={{ fontSize: 11, color: 'var(--text-muted)', marginBottom: 4 }}>우편번호</div>
        <div style={{ fontSize: 24, fontWeight: 800, color: 'var(--accent-primary)' }}>
          {d.postal_code ?? '------'}
        </div>
      </div>
    </div>
  )
}
