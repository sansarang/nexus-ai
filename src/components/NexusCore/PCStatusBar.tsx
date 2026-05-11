import { useState, useEffect } from 'react'

interface Stats {
  CPUPercent?: number
  RAMPercent?: number
  CPUTemp?: number
  DiskPercent?: number
}

function colorFor(val: number): string {
  if (val >= 90) return '#ef4444'
  if (val >= 70) return '#f59e0b'
  return '#22c55e'
}

export function PCStatusBar() {
  const [stats, setStats] = useState<Stats | null>(null)

  useEffect(() => {
    let unlisten: (() => void) | null = null

    // Try to listen for Tauri events
    const setupListener = async () => {
      try {
        const { listen } = await import('@tauri-apps/api/event')
        unlisten = await listen<Stats>('system-stats', event => {
          setStats(event.payload)
        })
      } catch {
        // Not in Tauri, use simulated data
      }
    }

    setupListener().catch(() => null)

    // Simulate stats in dev
    const sim: Stats = { CPUPercent: 25, RAMPercent: 60, CPUTemp: 55, DiskPercent: 70 }
    setStats(sim)

    const interval = setInterval(() => {
      setStats(prev => {
        if (!prev) return sim
        return {
          CPUPercent: Math.max(5, Math.min(99, (prev.CPUPercent ?? 25) + (Math.random() - 0.5) * 5)),
          RAMPercent: Math.max(10, Math.min(99, (prev.RAMPercent ?? 60) + (Math.random() - 0.5) * 3)),
          CPUTemp: Math.max(30, Math.min(95, (prev.CPUTemp ?? 55) + (Math.random() - 0.5) * 2)),
          DiskPercent: prev.DiskPercent ?? 70,
        }
      })
    }, 2000)

    return () => {
      clearInterval(interval)
      unlisten?.()
    }
  }, [])

  if (!stats) return null

  const items: Array<{ label: string; value: string; color: string }> = [
    { label: 'CPU', value: `${Math.round(stats.CPUPercent ?? 0)}%`, color: colorFor(stats.CPUPercent ?? 0) },
    { label: 'RAM', value: `${Math.round(stats.RAMPercent ?? 0)}%`, color: colorFor(stats.RAMPercent ?? 0) },
    { label: '온도', value: `${Math.round(stats.CPUTemp ?? 0)}°C`, color: colorFor(stats.CPUTemp ?? 0) },
    { label: 'Disk', value: `${Math.round(stats.DiskPercent ?? 0)}%`, color: colorFor(stats.DiskPercent ?? 0) },
  ]

  return (
    <div style={{ display: 'flex', gap: 12, alignItems: 'center', padding: '4px 0' }}>
      {items.map(item => (
        <div key={item.label} style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
          <div
            style={{
              width: 6,
              height: 6,
              borderRadius: '50%',
              background: item.color,
              boxShadow: `0 0 4px ${item.color}`,
            }}
          />
          <span style={{ fontSize: 11, color: 'var(--text-secondary)' }}>{item.label}</span>
          <span style={{ fontSize: 11, color: item.color, fontWeight: 600 }}>{item.value}</span>
        </div>
      ))}
    </div>
  )
}
