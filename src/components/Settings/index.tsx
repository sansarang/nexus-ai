import { useState } from 'react'
import { motion } from 'framer-motion'
import { Keyboard, Bell, Shield, Globe, Info } from 'lucide-react'
import { useAppStore } from '../../stores/appStore'

function Toggle({ value, onChange }: { value: boolean; onChange: (v: boolean) => void }) {
  return (
    <motion.button
      onClick={() => onChange(!value)}
      animate={{ background: value ? 'var(--accent-primary)' : 'var(--border-default)' }}
      transition={{ duration: 0.15 }}
      style={{
        width: 40,
        height: 22,
        borderRadius: 11,
        border: 'none',
        position: 'relative',
        cursor: 'pointer',
        flexShrink: 0,
      }}
    >
      <motion.div
        animate={{ x: value ? 20 : 2 }}
        transition={{ type: 'spring', stiffness: 500, damping: 30 }}
        style={{
          position: 'absolute',
          top: 3,
          width: 16,
          height: 16,
          borderRadius: '50%',
          background: '#fff',
          boxShadow: '0 1px 4px rgba(0,0,0,0.3)',
        }}
      />
    </motion.button>
  )
}

function Row({ icon, label, desc, children }: { icon: React.ReactNode; label: string; desc?: string; children: React.ReactNode }) {
  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 12,
        padding: '12px 0',
        borderBottom: '1px solid var(--border-subtle)',
      }}
    >
      <div style={{ color: 'var(--text-muted)', flexShrink: 0 }}>{icon}</div>
      <div style={{ flex: 1, minWidth: 0 }}>
        <p style={{ fontSize: 13, fontWeight: 600, color: 'var(--text-primary)' }}>{label}</p>
        {desc && <p style={{ fontSize: 11, color: 'var(--text-muted)', marginTop: 2 }}>{desc}</p>}
      </div>
      {children}
    </div>
  )
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div
      style={{
        background: 'var(--bg-elevated)',
        border: '1px solid var(--border-subtle)',
        borderRadius: 'var(--radius-md)',
        padding: '4px 16px 4px',
      }}
    >
      <p
        style={{
          fontSize: 11,
          fontWeight: 700,
          color: 'var(--text-muted)',
          textTransform: 'uppercase',
          letterSpacing: '0.06em',
          padding: '12px 0 8px',
        }}
      >
        {title}
      </p>
      {children}
    </div>
  )
}

export function SettingsView() {
  const { isLoggedIn, userEmail, subscriptionStatus, subscriptionExpiry, setLoggedOut } = useAppStore()
  const [autostart, setAutostart] = useState(false)
  const [notifications, setNotifications] = useState(true)
  const [lang, setLang] = useState<'ko' | 'en'>('ko')

  return (
    <div style={{ flex: 1, overflowY: 'auto', padding: '20px 24px', display: 'flex', flexDirection: 'column', gap: 16 }}>
      <h2 style={{ fontSize: 16, fontWeight: 800, color: 'var(--text-primary)', letterSpacing: '-0.02em' }}>⚙️ 설정</h2>

      {/* 단축키 */}
      <Section title="단축키">
        <Row icon={<Keyboard size={15} />} label="빠른 실행" desc="전역 단축키로 어디서든 실행">
          <div style={{ display: 'flex', gap: 4, alignItems: 'center' }}>
            {['Alt', 'Space'].map((k, i) => (
              <span key={k} style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                {i > 0 && <span style={{ color: 'var(--text-muted)', fontSize: 11 }}>+</span>}
                <kbd style={{ padding: '3px 8px', borderRadius: 6, background: 'var(--bg-elevated)', border: '1px solid var(--border-default)', color: 'var(--text-secondary)', fontSize: 11, fontFamily: 'monospace' }}>{k}</kbd>
              </span>
            ))}
          </div>
        </Row>
        <Row icon={<Keyboard size={15} />} label="설정 열기" desc="">
          <div style={{ display: 'flex', gap: 4 }}>
            {['Ctrl', ','].map((k, i) => (
              <span key={k} style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                {i > 0 && <span style={{ color: 'var(--text-muted)', fontSize: 11 }}>+</span>}
                <kbd style={{ padding: '3px 8px', borderRadius: 6, background: 'var(--bg-elevated)', border: '1px solid var(--border-default)', color: 'var(--text-secondary)', fontSize: 11, fontFamily: 'monospace' }}>{k}</kbd>
              </span>
            ))}
          </div>
        </Row>
      </Section>

      {/* 앱 설정 */}
      <Section title="앱">
        <Row icon={<span style={{ fontSize: 14 }}>🚀</span>} label="시작 시 자동 실행" desc="Windows 부팅 시 자동으로 실행">
          <Toggle value={autostart} onChange={setAutostart} />
        </Row>
        <Row icon={<Bell size={15} />} label="알림" desc="PC 상태 변화 시 알림 표시">
          <Toggle value={notifications} onChange={setNotifications} />
        </Row>
        <Row icon={<span style={{ fontSize: 14 }}>🌙</span>} label="다크 모드" desc="항상 다크 모드 사용">
          <Toggle value={true} onChange={() => {}} />
        </Row>
      </Section>

      {/* 언어 */}
      <Section title="언어">
        <Row icon={<Globe size={15} />} label="언어" desc="">
          <div style={{ display: 'flex', gap: 6 }}>
            {(['ko', 'en'] as const).map((l) => (
              <button
                key={l}
                onClick={() => setLang(l)}
                style={{
                  padding: '5px 12px',
                  borderRadius: 8,
                  border: `1px solid ${lang === l ? 'var(--accent-primary)' : 'var(--border-default)'}`,
                  background: lang === l ? 'rgba(79,126,247,0.15)' : 'var(--glass-bg)',
                  color: lang === l ? 'var(--accent-primary)' : 'var(--text-secondary)',
                  fontSize: 12,
                  fontWeight: lang === l ? 700 : 400,
                  cursor: 'pointer',
                }}
              >
                {l === 'ko' ? '🇰🇷 한국어' : '🇺🇸 English'}
              </button>
            ))}
          </div>
        </Row>
      </Section>

      {/* 계정 */}
      <Section title="계정">
        {isLoggedIn ? (
          <>
            <Row icon={<Shield size={15} />} label={userEmail} desc={
              subscriptionStatus === 'trial'
                ? `체험판 · ${subscriptionExpiry ? new Date(subscriptionExpiry).toLocaleDateString('ko-KR') + '까지' : ''}`
                : subscriptionStatus === 'active' ? '구독 중' : '구독 만료'
            }>
              <span style={{
                padding: '3px 10px', borderRadius: 20, fontSize: 11, fontWeight: 700,
                background: subscriptionStatus === 'active' ? 'rgba(34,197,94,0.15)' : subscriptionStatus === 'trial' ? 'rgba(79,126,247,0.15)' : 'rgba(239,68,68,0.15)',
                color: subscriptionStatus === 'active' ? 'var(--success)' : subscriptionStatus === 'trial' ? 'var(--accent-primary)' : 'var(--error)',
              }}>
                {subscriptionStatus === 'active' ? '활성' : subscriptionStatus === 'trial' ? '체험' : '만료'}
              </span>
            </Row>
            <Row icon={<span style={{ fontSize: 14 }}>🚪</span>} label="로그아웃" desc="">
              <motion.button
                whileTap={{ scale: 0.95 }}
                onClick={setLoggedOut}
                style={{ padding: '6px 14px', borderRadius: 8, border: '1px solid var(--border-default)', background: 'transparent', color: 'var(--text-secondary)', fontSize: 12, fontWeight: 600, cursor: 'pointer' }}
              >
                로그아웃
              </motion.button>
            </Row>
          </>
        ) : (
          <Row icon={<Shield size={15} />} label="로그인 필요" desc="Google 계정으로 로그인하세요">
            <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>미로그인</span>
          </Row>
        )}
      </Section>

      {/* 앱 정보 */}
      <Section title="정보">
        <Row icon={<Info size={15} />} label="Nexus" desc="Windows PC 관리 도구">
          <span style={{ fontSize: 12, color: 'var(--text-muted)', fontFamily: 'monospace' }}>v1.0.0</span>
        </Row>
      </Section>
    </div>
  )
}
