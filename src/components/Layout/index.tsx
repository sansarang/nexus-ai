import { useState } from 'react'
import { motion } from 'framer-motion'
import { useAppStore, type ViewId } from '../../stores/appStore'
import { PaywallModal } from '../PaywallModal'

const PREMIUM_VIEWS = new Set<ViewId>(['voicememo', 'daily', 'predictive'])

const PREMIUM_FEATURE_KEY: Partial<Record<ViewId, string>> = {
  voicememo:  'voice_memo',
  daily:      'daily_briefing',
  predictive: 'ai_prediction',
}

const NAV_ITEMS: { id: ViewId; icon: string; label: string; shortcut: string }[] = [
  { id: 'home',      icon: '🏠', label: '홈',           shortcut: '1' },
  { id: 'repair',    icon: '🔧', label: '수리',          shortcut: '2' },
  { id: 'security',  icon: '🛡️', label: '보안',          shortcut: '3' },
  { id: 'files',     icon: '📁', label: '파일',          shortcut: '4' },
  { id: 'translate', icon: '🌐', label: '번역',          shortcut: '5' },
  { id: 'clipboard', icon: '📋', label: '클립보드',      shortcut: '6' },
  { id: 'memo',      icon: '📝', label: '메모',          shortcut: '7' },
  { id: 'autoclean', icon: '🧹', label: 'PC 정리',       shortcut: '8' },
  { id: 'monitor',   icon: '📊', label: '실시간 모니터', shortcut: '9' },
  { id: 'privacy',   icon: '🔒', label: '프라이버시',    shortcut: '0' },
  { id: 'focus',     icon: '🎯', label: '집중 모드',     shortcut: 'F' },
  { id: 'daily',     icon: '☀️', label: '데일리 리포트', shortcut: 'D' },
  { id: 'voicememo',  icon: '🎙️', label: '음성 메모',    shortcut: 'V' },
  { id: 'organize',   icon: '📁', label: '스마트 정리',  shortcut: 'O' },
  { id: 'predictive', icon: '🔮', label: 'AI 예측',      shortcut: 'P' },
]

function NavItem({
  item,
  active,
  isPremiumLocked,
  onClick,
}: {
  item: (typeof NAV_ITEMS)[0]
  active: boolean
  isPremiumLocked?: boolean
  onClick: () => void
}) {
  return (
    <motion.button
      onClick={onClick}
      whileHover={{ scale: 1.02 }}
      whileTap={{ scale: 0.97 }}
      style={{
        position: 'relative',
        width: '100%',
        display: 'flex',
        alignItems: 'center',
        gap: 10,
        padding: '8px 12px',
        borderRadius: 'var(--radius-sm)',
        border: 'none',
        background: active ? 'rgba(79,126,247,0.12)' : 'transparent',
        color: active ? 'var(--accent-primary)' : 'var(--text-secondary)',
        cursor: 'pointer',
        textAlign: 'left',
        fontSize: 13,
        fontWeight: active ? 600 : 400,
        transition: 'all var(--duration-fast) var(--ease-smooth)',
      }}
      onMouseEnter={(e) => {
        if (!active) e.currentTarget.style.background = 'var(--glass-bg)'
      }}
      onMouseLeave={(e) => {
        if (!active) e.currentTarget.style.background = 'transparent'
      }}
    >
      {active && (
        <motion.div
          layoutId="nav-indicator"
          style={{
            position: 'absolute',
            left: 0,
            top: 6,
            bottom: 6,
            width: 2,
            borderRadius: 2,
            background: 'var(--accent-primary)',
          }}
          transition={{ type: 'spring', stiffness: 500, damping: 35 }}
        />
      )}

      <span style={{ fontSize: 16, lineHeight: 1, flexShrink: 0 }}>{item.icon}</span>
      <span style={{ flex: 1 }}>{item.label}</span>
      {isPremiumLocked ? (
        <span style={{ fontSize: 9, fontWeight: 700, color: '#f59e0b', background: 'rgba(245,158,11,0.15)', borderRadius: 4, padding: '1px 4px' }}>
          PRO
        </span>
      ) : (
        <span style={{ fontSize: 10, color: 'var(--text-muted)', fontFamily: 'monospace', opacity: 0.6 }}>
          ⌘{item.shortcut}
        </span>
      )}
    </motion.button>
  )
}

export function Sidebar() {
  const { currentView, setView, subscriptionStatus } = useAppStore()
  const [paywallFeature, setPaywallFeature] = useState<string | null>(null)
  const isPremium = subscriptionStatus === 'active' || subscriptionStatus === 'trial'

  const handleNavClick = (id: ViewId) => {
    if (PREMIUM_VIEWS.has(id) && !isPremium) {
      setPaywallFeature(PREMIUM_FEATURE_KEY[id] ?? id)
      return
    }
    setView(id)
  }

  return (
    <div
      style={{
        width: 'var(--sidebar-w)',
        flexShrink: 0,
        background: 'var(--bg-surface)',
        borderRight: '1px solid var(--border-subtle)',
        display: 'flex',
        flexDirection: 'column',
        padding: '12px 8px',
        gap: 2,
        overflowY: 'auto',
      }}
    >
      {NAV_ITEMS.map((item) => (
        <NavItem
          key={item.id}
          item={item}
          active={currentView === item.id}
          isPremiumLocked={PREMIUM_VIEWS.has(item.id) && !isPremium}
          onClick={() => handleNavClick(item.id)}
        />
      ))}

      {/* 구분선 */}
      <div
        style={{
          height: 1,
          background: 'var(--border-subtle)',
          margin: '8px 4px',
        }}
      />

      {/* 설정 */}
      <NavItem
        item={{ id: 'settings', icon: '⚙️', label: '설정', shortcut: ',' }}
        active={currentView === 'settings'}
        onClick={() => setView('settings')}
      />

      {/* 하단 버전 */}
      <div style={{ flex: 1 }} />
      <div
        style={{
          padding: '8px 12px',
          fontSize: 11,
          color: 'var(--text-muted)',
        }}
      >
        v1.0.0
      </div>

      {paywallFeature && (
        <PaywallModal
          feature={paywallFeature}
          used={0}
          limit={0}
          onClose={() => setPaywallFeature(null)}
        />
      )}
    </div>
  )
}

export function Layout({ children }: { children: React.ReactNode }) {
  return (
    <div
      style={{
        flex: 1,
        display: 'flex',
        overflow: 'hidden',
        minHeight: 0,
      }}
    >
      <Sidebar />
      <main
        style={{
          flex: 1,
          overflow: 'hidden',
          display: 'flex',
          flexDirection: 'column',
          background: 'var(--bg-base)',
        }}
      >
        {children}
      </main>
    </div>
  )
}
