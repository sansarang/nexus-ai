interface PCScoreData { score?: number; issues?: Array<{ title?: string; severity?: string }> }

function scoreColor(score: number): string {
  if (score >= 80) return '#22c55e'
  if (score >= 50) return '#f59e0b'
  return '#ef4444'
}

export function PCScoreCard({ data }: { data: unknown }) {
  const d = (data ?? {}) as PCScoreData
  const score = d.score ?? 0
  const issues = d.issues ?? []
  const color = scoreColor(score)

  return (
    <div style={{ padding: 16 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 20, marginBottom: 16 }}>
        <div
          style={{
            width: 80,
            height: 80,
            borderRadius: '50%',
            border: `4px solid ${color}`,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            flexShrink: 0,
            boxShadow: `0 0 20px ${color}44`,
          }}
        >
          <span style={{ fontSize: 28, fontWeight: 800, color }}>{score}</span>
        </div>
        <div>
          <div style={{ fontSize: 16, fontWeight: 700, color: 'var(--text-primary)' }}>PC 건강 점수</div>
          <div style={{ fontSize: 13, color }}>
            {score >= 80 ? '✅ 양호' : score >= 50 ? '⚠️ 주의 필요' : '🔴 즉시 수리 필요'}
          </div>
        </div>
      </div>

      {issues.length > 0 && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
          {issues.map((issue, i) => (
            <div
              key={i}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 8,
                padding: '6px 10px',
                borderRadius: 6,
                background: 'var(--bg-elevated)',
                border: '1px solid var(--border-subtle)',
                fontSize: 12,
                color: 'var(--text-secondary)',
              }}
            >
              <span>{issue.severity === 'high' ? '🔴' : issue.severity === 'medium' ? '🟡' : '🟢'}</span>
              {issue.title ?? '문제 항목'}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
