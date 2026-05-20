import { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import { Check, Palette } from 'lucide-react'

const BASE = 'http://127.0.0.1:17891'

interface VerticalConfig {
  id: string
  name: string
  theme: string
  logo: string
  default_persona: string
  features: string[]
  welcome_msg: string
  watermark: string
}

const VERTICAL_EMOJI: Record<string, string> = {
  general: '🤖',
  legal: '⚖️',
  medical: '🏥',
  finance: '📈',
  content: '🎬',
}

export function VerticalSwitcher() {
  const [presets, setPresets] = useState<VerticalConfig[]>([])
  const [activeId, setActiveId] = useState<string>('general')
  const [loading, setLoading] = useState(true)
  const [applying, setApplying] = useState<string | null>(null)

  useEffect(() => {
    const stored = localStorage.getItem('nexus_vertical_id')
    if (stored) setActiveId(stored)

    Promise.all([
      fetch(`${BASE}/api/vertical/presets`).then(r => r.json()),
      fetch(`${BASE}/api/vertical/config`).then(r => r.json()),
    ]).then(([presetsRes, configRes]) => {
      setPresets(presetsRes.presets ?? [])
      if (configRes.config?.id) {
        setActiveId(configRes.config.id)
        localStorage.setItem('nexus_vertical_id', configRes.config.id)
      }
    }).catch(() => {}).finally(() => setLoading(false))
  }, [])

  const apply = async (preset: VerticalConfig) => {
    setApplying(preset.id)
    try {
      await fetch(`${BASE}/api/vertical/config`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(preset),
      })
      setActiveId(preset.id)
      localStorage.setItem('nexus_vertical_id', preset.id)
      // Apply theme color to CSS variable
      document.documentElement.style.setProperty('--accent-primary', preset.theme)
    } catch {
      // ignore — still apply locally
      setActiveId(preset.id)
      localStorage.setItem('nexus_vertical_id', preset.id)
      document.documentElement.style.setProperty('--accent-primary', preset.theme)
    } finally {
      setApplying(null)
    }
  }

  if (loading) {
    return <p style={{ fontSize: 12, color: 'var(--text-muted)', textAlign: 'center', padding: '20px 0' }}>불러오는 중...</p>
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      {/* Header */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        <Palette size={16} color="var(--accent-primary)" />
        <span style={{ fontSize: 14, fontWeight: 700, color: 'var(--text-primary)' }}>앱 테마 / 버티컬</span>
      </div>
      <p style={{ fontSize: 12, color: 'var(--text-muted)', marginTop: -8 }}>
        업종에 맞는 테마와 기능 세트를 선택하세요.
      </p>

      {/* Preset Grid */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(160px, 1fr))', gap: 10 }}>
        {presets.map(preset => {
          const isActive = activeId === preset.id
          const isApplying = applying === preset.id
          return (
            <motion.button
              key={preset.id}
              whileTap={{ scale: 0.97 }}
              whileHover={{ scale: 1.02 }}
              onClick={() => !isApplying && apply(preset)}
              style={{
                background: 'var(--bg-elevated)',
                border: `2px solid ${isActive ? preset.theme : 'var(--border-subtle)'}`,
                borderRadius: 12,
                padding: '14px 12px',
                cursor: 'pointer',
                textAlign: 'left',
                position: 'relative',
                transition: 'border-color 0.2s',
              }}
            >
              {/* Active checkmark */}
              {isActive && (
                <motion.div
                  initial={{ scale: 0 }}
                  animate={{ scale: 1 }}
                  style={{
                    position: 'absolute',
                    top: 8,
                    right: 8,
                    width: 18,
                    height: 18,
                    borderRadius: '50%',
                    background: preset.theme,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                  }}
                >
                  <Check size={11} color="#fff" strokeWidth={3} />
                </motion.div>
              )}

              {/* Color dot */}
              <div style={{
                width: 32,
                height: 32,
                borderRadius: 8,
                background: preset.theme,
                marginBottom: 10,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                fontSize: 16,
              }}>
                {VERTICAL_EMOJI[preset.id] ?? '🤖'}
              </div>

              <p style={{ fontSize: 12, fontWeight: 700, color: 'var(--text-primary)', marginBottom: 3, lineHeight: 1.3 }}>
                {preset.name}
              </p>
              <p style={{ fontSize: 10, color: 'var(--text-muted)', lineHeight: 1.4 }}>
                {preset.features.slice(0, 3).join(', ')}{preset.features.length > 3 ? '…' : ''}
              </p>

              {isApplying && (
                <div style={{ marginTop: 6 }}>
                  <p style={{ fontSize: 10, color: preset.theme }}>적용 중...</p>
                </div>
              )}
            </motion.button>
          )
        })}
      </div>

      {/* Current info */}
      {presets.find(p => p.id === activeId) && (
        <div style={{
          background: 'var(--bg-elevated)',
          border: '1px solid var(--border-subtle)',
          borderRadius: 10,
          padding: '10px 14px',
        }}>
          {(() => {
            const p = presets.find(pr => pr.id === activeId)!
            return (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
                <p style={{ fontSize: 11, fontWeight: 700, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.06em' }}>현재 활성</p>
                <p style={{ fontSize: 13, fontWeight: 700, color: p.theme }}>{p.name}</p>
                <p style={{ fontSize: 11, color: 'var(--text-muted)' }}>{p.welcome_msg}</p>
              </div>
            )
          })()}
        </div>
      )}
    </div>
  )
}
