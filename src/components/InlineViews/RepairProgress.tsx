interface RepairStep { title?: string; status?: 'done' | 'running' | 'pending' }

export function RepairProgress({ data }: { data: unknown }) {
  const steps: RepairStep[] = Array.isArray(data) ? (data as RepairStep[]) : []

  if (steps.length === 0) {
    return (
      <div style={{ padding: 16, color: 'var(--text-muted)', fontSize: 13, textAlign: 'center' }}>
        🔧 수리 진행 중...
      </div>
    )
  }

  return (
    <div style={{ padding: 12, display: 'flex', flexDirection: 'column', gap: 6 }}>
      {steps.map((step, i) => {
        const icon = step.status === 'done' ? '✅' : step.status === 'running' ? '⚙️' : '⏳'
        const color = step.status === 'done' ? '#22c55e' : step.status === 'running' ? '#4f7ef7' : 'var(--text-muted)'
        return (
          <div
            key={i}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 10,
              padding: '8px 12px',
              borderRadius: 8,
              background: 'var(--bg-elevated)',
              border: '1px solid var(--border-subtle)',
            }}
          >
            <span style={{ fontSize: 16 }}>{icon}</span>
            <span style={{ fontSize: 13, color }}>{step.title ?? `단계 ${i + 1}`}</span>
          </div>
        )
      })}
    </div>
  )
}
