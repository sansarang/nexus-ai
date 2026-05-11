import { useState } from 'react'

export function TitleBar() {
  const [hoverClose, setHoverClose] = useState(false)
  const [hoverMin, setHoverMin] = useState(false)

  const invokeWindow = async (cmd: string) => {
    try {
      const { invoke } = await import('@tauri-apps/api/core')
      await invoke(cmd)
    } catch {
      // Not in Tauri environment
    }
  }

  return (
    <div
      data-tauri-drag-region
      style={{
        height: 36,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background: 'var(--bg-surface)',
        borderBottom: '1px solid var(--border-subtle)',
        position: 'relative',
        flexShrink: 0,
        cursor: 'default',
        userSelect: 'none',
      }}
    >
      {/* Window controls */}
      <div
        style={{
          position: 'absolute',
          left: 12,
          display: 'flex',
          gap: 6,
          alignItems: 'center',
        }}
      >
        <button
          onMouseEnter={() => setHoverClose(true)}
          onMouseLeave={() => setHoverClose(false)}
          onClick={() => invokeWindow('close_window')}
          style={{
            width: 12,
            height: 12,
            borderRadius: '50%',
            border: 'none',
            background: hoverClose ? '#ef4444' : 'rgba(239,68,68,0.5)',
            cursor: 'pointer',
            padding: 0,
          }}
        />
        <button
          onMouseEnter={() => setHoverMin(true)}
          onMouseLeave={() => setHoverMin(false)}
          onClick={() => invokeWindow('minimize_window')}
          style={{
            width: 12,
            height: 12,
            borderRadius: '50%',
            border: 'none',
            background: hoverMin ? '#f59e0b' : 'rgba(245,158,11,0.5)',
            cursor: 'pointer',
            padding: 0,
          }}
        />
      </div>

      {/* Title */}
      <span
        style={{
          fontSize: 13,
          fontWeight: 600,
          color: 'var(--text-secondary)',
          letterSpacing: '0.08em',
        }}
      >
        NEXUS
      </span>
    </div>
  )
}
