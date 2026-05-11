import { invoke } from '@tauri-apps/api/core'
import { Minus, Square, X } from 'lucide-react'

function WinBtn({
  onClick,
  danger,
  children,
}: {
  onClick: () => void
  danger?: boolean
  children: React.ReactNode
}) {
  return (
    <button
      onClick={onClick}
      style={{
        width: 28,
        height: 28,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        borderRadius: 6,
        border: 'none',
        background: 'transparent',
        color: 'var(--text-muted)',
        cursor: 'pointer',
        WebkitAppRegion: 'no-drag',
      } as React.CSSProperties}
      onMouseEnter={(e) => {
        if (danger) {
          e.currentTarget.style.background = 'rgba(239,68,68,0.2)'
          e.currentTarget.style.color = '#ef4444'
        } else {
          e.currentTarget.style.background = 'rgba(255,255,255,0.08)'
          e.currentTarget.style.color = 'var(--text-primary)'
        }
      }}
      onMouseLeave={(e) => {
        e.currentTarget.style.background = 'transparent'
        e.currentTarget.style.color = 'var(--text-muted)'
      }}
    >
      {children}
    </button>
  )
}

export function TitleBar() {

  const minimize = async () => {
    try { await invoke('minimize_window') } catch { /* dev */ }
  }
  const maximize = async () => {
    try { await invoke('toggle_maximize') } catch { /* dev */ }
  }
  const close = async () => {
    try { await invoke('close_window') } catch { window.close() }
  }

  return (
    <div
      data-tauri-drag-region
      style={{
        height: 40,
        flexShrink: 0,
        background: 'var(--bg-surface)',
        borderBottom: '1px solid var(--border-subtle)',
        display: 'flex',
        alignItems: 'center',
        padding: '0 12px 0 16px',
        userSelect: 'none',
        WebkitUserSelect: 'none',
      } as React.CSSProperties}
    >
      {/* 로고 + 앱명 */}
      <div
        style={{ display: 'flex', alignItems: 'center', gap: 8 }}
        data-tauri-drag-region
      >
        <span style={{ fontSize: 15, lineHeight: 1 }}>◉</span>
        <span
          style={{
            color: 'var(--text-primary)',
            fontWeight: 700,
            fontSize: 13,
            letterSpacing: '0.04em',
          }}
        >
          NEXUS
        </span>
      </div>

      {/* 드래그 영역 (중앙) */}
      <div
        data-tauri-drag-region
        style={{ flex: 1 }}
      />

      {/* 윈도우 컨트롤 */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 2 }}>
        <WinBtn onClick={minimize}>
          <Minus size={13} />
        </WinBtn>
        <WinBtn onClick={maximize}>
          <Square size={11} />
        </WinBtn>
        <WinBtn onClick={close} danger>
          <X size={13} />
        </WinBtn>
      </div>
    </div>
  )
}
