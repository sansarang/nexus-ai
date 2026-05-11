const ACTIONS = [
  { label: 'PC 진단', cmd: 'PC 진단해줘' },
  { label: '자동 정리', cmd: '자동 정리해줘' },
  { label: '보안 점검', cmd: '보안 점검해줘' },
  { label: '파일 찾기', cmd: '파일 찾아줘' },
  { label: '집중 모드', cmd: '집중 모드 켜줘' },
  { label: '날씨', cmd: '오늘 날씨 알려줘' },
  { label: '뉴스', cmd: '오늘 주요 뉴스 알려줘' },
  { label: '브리핑', cmd: '아침 브리핑 해줘' },
  { label: '도움말', cmd: '뭘 할 수 있어?' },
  { label: 'AI 예측', cmd: 'PC 상태 예측해줘' },
]

export function QuickActions({ onSelect }: { onSelect: (cmd: string) => void }) {
  return (
    <div
      style={{
        display: 'flex',
        gap: 6,
        overflowX: 'auto',
        padding: '6px 12px',
        scrollbarWidth: 'none',
      }}
    >
      {ACTIONS.map(action => (
        <button
          key={action.label}
          onClick={() => onSelect(action.cmd)}
          style={{
            flexShrink: 0,
            padding: '5px 12px',
            borderRadius: 20,
            border: '1px solid var(--border-default)',
            background: 'var(--glass-bg)',
            color: 'var(--text-secondary)',
            fontSize: 12,
            cursor: 'pointer',
            whiteSpace: 'nowrap',
            transition: 'all 0.15s ease',
          }}
          onMouseEnter={e => {
            const el = e.currentTarget
            el.style.background = 'var(--bg-elevated)'
            el.style.color = 'var(--text-primary)'
            el.style.borderColor = 'var(--accent-primary)'
          }}
          onMouseLeave={e => {
            const el = e.currentTarget
            el.style.background = 'var(--glass-bg)'
            el.style.color = 'var(--text-secondary)'
            el.style.borderColor = 'var(--border-default)'
          }}
        >
          {action.label}
        </button>
      ))}
    </div>
  )
}
