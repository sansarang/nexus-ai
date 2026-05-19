import { useState } from 'react'
import { motion } from 'framer-motion'
import { useAppStore } from '../../stores/appStore'

interface PrivacyFeature {
  id: string
  name: string
  desc: string
  restart: boolean
}

const FEATURES: PrivacyFeature[] = [
  { id: 'copilot',   name: 'Windows Copilot',      desc: 'AI 어시스턴트 완전 비활성화',           restart: true },
  { id: 'onedrive',  name: 'OneDrive',              desc: 'OneDrive 자동 실행 및 연동 끄기',       restart: false },
  { id: 'telemetry', name: '원격 측정',             desc: 'Microsoft 데이터 수집 차단',            restart: true },
  { id: 'ads',       name: '개인화 광고',           desc: '앱 내 광고 ID 비활성화',                restart: false },
  { id: 'cortana',   name: 'Cortana',               desc: 'Cortana 완전 비활성화',                 restart: true },
  { id: 'widgets',   name: '위젯',                  desc: '작업표시줄 위젯 제거',                  restart: true },
  { id: 'telehosts', name: '텔레메트리 hosts 차단', desc: 'hosts 파일로 MS 서버 차단',             restart: false },
]

const RECOMMENDED = ['onedrive', 'telemetry', 'ads', 'telehosts']

export function PrivacyView() {
  const { privacySettings, setPrivacy } = useAppStore()
  const [loading, setLoading] = useState<Record<string, boolean>>({})

  const toggle = async (id: string) => {
    const newVal = !privacySettings[id]
    setLoading((l) => ({ ...l, [id]: true }))
    setPrivacy(id, newVal)
    // fire-and-forget
    fetch('http://127.0.0.1:17891/api/privacy', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ feature: id, enabled: newVal }),
    }).catch(() => {})
    setTimeout(() => setLoading((l) => ({ ...l, [id]: false })), 400)
  }

  const disableAll = () => {
    FEATURES.forEach((f) => {
      if (privacySettings[f.id]) toggle(f.id)
    })
  }

  const applyRecommended = () => {
    FEATURES.forEach((f) => {
      const shouldEnable = RECOMMENDED.includes(f.id)
      if (privacySettings[f.id] !== shouldEnable) toggle(f.id)
    })
  }

  return (
    <div style={{
      flex: 1, overflowY: 'auto', padding: 24,
      background: 'var(--bg-base)', color: 'var(--text-primary)',
    }}>
      <motion.div
        initial={{ opacity: 0, y: 12 }}
        animate={{ opacity: 1, y: 0 }}
        style={{ maxWidth: 640 }}
      >
        <h2 style={{ margin: '0 0 6px', fontSize: 20, fontWeight: 700 }}>🔒 프라이버시 &amp; MS 기능 제어</h2>
        <p style={{ margin: '0 0 16px', fontSize: 13, color: 'var(--text-secondary)' }}>
          Microsoft 내장 기능과 데이터 수집을 제어합니다
        </p>

        {/* Warning banner */}
        <div style={{
          padding: '12px 16px',
          borderRadius: 'var(--radius-md)',
          background: 'rgba(245,158,11,0.08)',
          border: '1px solid rgba(245,158,11,0.3)',
          marginBottom: 20,
          fontSize: 13,
          color: 'var(--warning)',
        }}>
          ⚠️ 시스템 레지스트리를 수정합니다. 관리자 권한이 필요합니다.
        </div>

        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          {FEATURES.map((f, i) => {
            const isOn = privacySettings[f.id] ?? false
            const isLoading = loading[f.id] ?? false
            return (
              <motion.div
                key={f.id}
                initial={{ opacity: 0, x: -10 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ delay: i * 0.04 }}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 14,
                  padding: '14px 16px',
                  borderRadius: 'var(--radius-md)',
                  background: 'var(--bg-surface)',
                  border: '1px solid var(--glass-border)',
                }}
              >
                <div style={{ flex: 1 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 2 }}>
                    <span style={{ fontSize: 14, fontWeight: 600 }}>{f.name}</span>
                    {f.restart && isOn && (
                      <span style={{
                        padding: '1px 7px',
                        borderRadius: 10,
                        background: 'rgba(245,158,11,0.12)',
                        border: '1px solid rgba(245,158,11,0.3)',
                        fontSize: 10,
                        color: 'var(--warning)',
                        fontWeight: 600,
                      }}>재시작 필요</span>
                    )}
                  </div>
                  <div style={{ fontSize: 12, color: 'var(--text-muted)' }}>{f.desc}</div>
                </div>
                {/* Toggle switch */}
                <button
                  onClick={() => toggle(f.id)}
                  disabled={isLoading}
                  style={{
                    width: 44,
                    height: 24,
                    borderRadius: 12,
                    border: 'none',
                    background: isOn ? 'var(--accent-primary)' : 'rgba(255,255,255,0.1)',
                    cursor: isLoading ? 'wait' : 'pointer',
                    position: 'relative',
                    flexShrink: 0,
                    opacity: isLoading ? 0.6 : 1,
                    transition: 'background 0.2s ease',
                  }}
                >
                  <motion.div
                    animate={{ x: isOn ? 20 : 2 }}
                    transition={{ type: 'spring', stiffness: 500, damping: 30 }}
                    style={{
                      position: 'absolute',
                      top: 3,
                      width: 18,
                      height: 18,
                      borderRadius: '50%',
                      background: '#fff',
                      boxShadow: '0 1px 4px rgba(0,0,0,0.3)',
                    }}
                  />
                </button>
              </motion.div>
            )
          })}
        </div>

        {/* Bottom action buttons */}
        <div style={{ display: 'flex', gap: 10, marginTop: 20 }}>
          <motion.button
            onClick={disableAll}
            whileHover={{ scale: 1.02 }}
            whileTap={{ scale: 0.98 }}
            style={{
              flex: 1, padding: '10px 0', borderRadius: 'var(--radius-md)',
              border: '1px solid var(--glass-border)', background: 'var(--glass-bg)',
              color: 'var(--text-primary)', fontSize: 13, fontWeight: 500, cursor: 'pointer',
            }}
          >전체 끄기</motion.button>
          <motion.button
            onClick={applyRecommended}
            whileHover={{ scale: 1.02 }}
            whileTap={{ scale: 0.98 }}
            style={{
              flex: 1, padding: '10px 0', borderRadius: 'var(--radius-md)',
              border: 'none', background: 'var(--accent-primary)',
              color: '#fff', fontSize: 13, fontWeight: 600, cursor: 'pointer',
            }}
          >권장 설정 적용</motion.button>
        </div>
      </motion.div>
    </div>
  )
}
