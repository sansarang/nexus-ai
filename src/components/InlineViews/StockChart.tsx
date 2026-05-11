interface StockData { symbol?: string; price?: number; change?: number }

export function StockChart({ data }: { data: unknown }) {
  const d = (data ?? {}) as StockData
  const isPositive = (d.change ?? 0) >= 0

  // Simple SVG line placeholder
  const points = [20, 35, 28, 42, 38, 30, 45, 38, 50, 43].map((y, i) => `${i * 22},${60 - y}`).join(' ')

  return (
    <div style={{ padding: 16 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 12 }}>
        <div>
          <div style={{ fontSize: 13, color: 'var(--text-muted)' }}>{d.symbol ?? '--'}</div>
          <div style={{ fontSize: 28, fontWeight: 800, color: 'var(--text-primary)' }}>
            {d.price?.toLocaleString() ?? '--'}
          </div>
        </div>
        <div
          style={{
            padding: '4px 10px',
            borderRadius: 8,
            background: isPositive ? 'rgba(34,197,94,0.15)' : 'rgba(239,68,68,0.15)',
            color: isPositive ? '#22c55e' : '#ef4444',
            fontSize: 14,
            fontWeight: 600,
          }}
        >
          {isPositive ? '▲' : '▼'} {Math.abs(d.change ?? 0).toFixed(2)}%
        </div>
      </div>

      <svg width="100%" height="60" viewBox="0 0 200 60" preserveAspectRatio="none">
        <polyline
          points={points}
          fill="none"
          stroke={isPositive ? '#22c55e' : '#ef4444'}
          strokeWidth={2}
        />
      </svg>
    </div>
  )
}
