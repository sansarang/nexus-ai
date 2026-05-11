interface HWData { CPUPercent?: number; RAMPercent?: number; DiskPercent?: number; CPUTemp?: number }

interface GaugeProps { label: string; value: number; max?: number; unit?: string; color: string }

function Gauge({ label, value, max = 100, unit = '%', color }: GaugeProps) {
  const pct = Math.min(100, (value / max) * 100)
  return (
    <div style={{ flex: 1, padding: '10px 12px', borderRadius: 10, background: 'var(--bg-elevated)', border: '1px solid var(--border-subtle)' }}>
      <div style={{ fontSize: 11, color: 'var(--text-muted)', marginBottom: 6 }}>{label}</div>
      <div style={{ fontSize: 20, fontWeight: 800, color, marginBottom: 6 }}>
        {value.toFixed(0)}{unit}
      </div>
      <div style={{ height: 4, borderRadius: 2, background: 'var(--bg-surface)' }}>
        <div style={{ height: '100%', borderRadius: 2, background: color, width: `${pct}%` }} />
      </div>
    </div>
  )
}

export function HardwareMapMini({ data }: { data: unknown }) {
  const d = (data ?? {}) as HWData

  return (
    <div style={{ padding: 12, display: 'flex', gap: 8 }}>
      <Gauge label="CPU" value={d.CPUPercent ?? 0} color="#4f7ef7" />
      <Gauge label="RAM" value={d.RAMPercent ?? 0} color="#a78bfa" />
      <Gauge label="Disk" value={d.DiskPercent ?? 0} color="#22c55e" />
      <Gauge label="Temp" value={d.CPUTemp ?? 0} max={100} unit="°C" color="#f59e0b" />
    </div>
  )
}
