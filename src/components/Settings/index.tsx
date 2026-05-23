import { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import { Keyboard, Bell, Shield, Globe, Info, Mail, Save, RefreshCw, Key, Palette } from 'lucide-react'
import { useAppStore } from '../../stores/appStore'
import { check } from '@tauri-apps/plugin-updater'
import { relaunch } from '@tauri-apps/plugin-process'
import { APIKeyManager } from '../Enterprise/APIKeyManager'
import { VerticalSwitcher } from '../VerticalSwitcher'

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

type SettingsTab = 'general' | 'api' | 'theme'

export function SettingsView() {
  const { isLoggedIn, userEmail, subscriptionStatus, subscriptionExpiry, setLoggedOut, userLang, setUserLang } = useAppStore()
  const [activeTab, setActiveTab] = useState<SettingsTab>('general')
  const [autostart, setAutostart] = useState(false)
  const [notifications, setNotifications] = useState(true)

  // 업데이트 상태
  const [updateChecking, setUpdateChecking] = useState(false)
  const [updateStatus, setUpdateStatus] = useState<'idle' | 'latest' | 'available' | 'error'>('idle')
  const [updateVersion, setUpdateVersion] = useState('')

  const checkUpdate = async () => {
    setUpdateChecking(true)
    setUpdateStatus('idle')
    try {
      const update = await check()
      if (update?.available) {
        setUpdateVersion(update.version)
        setUpdateStatus('available')
        await update.downloadAndInstall()
        await relaunch()
      } else {
        setUpdateStatus('latest')
        setTimeout(() => setUpdateStatus('idle'), 3000)
      }
    } catch {
      setUpdateStatus('error')
      setTimeout(() => setUpdateStatus('idle'), 3000)
    } finally {
      setUpdateChecking(false)
    }
  }
  const lang = userLang
  const setLang = (l: 'ko' | 'en') => setUserLang(l)

  // 이메일 IMAP 설정
  const [emailHost, setEmailHost] = useState('')
  const [emailPort, setEmailPort] = useState('993')
  const [emailUser, setEmailUser] = useState('')
  const [emailPass, setEmailPass] = useState('')
  const [smtpHost, setSmtpHost] = useState('')
  const [smtpPort, setSmtpPort] = useState('587')
  const [emailSaving, setEmailSaving] = useState(false)
  const [emailSaved, setEmailSaved] = useState(false)

  useEffect(() => {
    fetch('http://127.0.0.1:17891/api/email/imap/config')
      .then(r => r.json())
      .then((d: Record<string, string>) => {
        if (d.imap_host) setEmailHost(d.imap_host)
        if (d.imap_port) setEmailPort(d.imap_port)
        if (d.username) setEmailUser(d.username)
        if (d.smtp_host) setSmtpHost(d.smtp_host)
        if (d.smtp_port) setSmtpPort(d.smtp_port)
      })
      .catch(() => {})
  }, [])

  const saveEmailConfig = async () => {
    setEmailSaving(true)
    await fetch('http://127.0.0.1:17891/api/email/imap/config', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        imap_host: emailHost, imap_port: emailPort,
        smtp_host: smtpHost, smtp_port: smtpPort,
        username: emailUser, password: emailPass,
      }),
    }).catch(() => {})
    setEmailSaving(false)
    setEmailSaved(true)
    setTimeout(() => setEmailSaved(false), 2000)
  }

  const inputStyle = {
    background: 'var(--glass-bg)', border: '1px solid var(--border-default)',
    borderRadius: 6, color: 'var(--text-primary)', fontSize: 12,
    padding: '5px 8px', width: '100%', outline: 'none',
  } as React.CSSProperties

  const TABS: { id: SettingsTab; label: string; icon: React.ReactNode }[] = [
    { id: 'general', label: '일반', icon: <Bell size={13} /> },
    { id: 'api', label: 'API 관리', icon: <Key size={13} /> },
    { id: 'theme', label: '앱 테마', icon: <Palette size={13} /> },
  ]

  return (
    <div style={{ flex: 1, overflowY: 'auto', padding: '20px 24px', display: 'flex', flexDirection: 'column', gap: 16 }}>
      <h2 style={{ fontSize: 16, fontWeight: 800, color: 'var(--text-primary)', letterSpacing: '-0.02em' }}>⚙️ 설정</h2>

      {/* Tabs */}
      <div style={{ display: 'flex', gap: 4, background: 'var(--bg-elevated)', padding: 4, borderRadius: 10, border: '1px solid var(--border-subtle)' }}>
        {TABS.map(tab => (
          <motion.button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            whileTap={{ scale: 0.97 }}
            style={{
              flex: 1,
              display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 5,
              padding: '7px 0',
              borderRadius: 7,
              border: 'none',
              background: activeTab === tab.id ? 'var(--accent-primary)' : 'transparent',
              color: activeTab === tab.id ? '#fff' : 'var(--text-secondary)',
              fontSize: 12, fontWeight: activeTab === tab.id ? 700 : 400,
              cursor: 'pointer',
              transition: 'background 0.15s',
            }}
          >
            {tab.icon} {tab.label}
          </motion.button>
        ))}
      </div>

      {/* API 관리 tab */}
      {activeTab === 'api' && <APIKeyManager />}

      {/* 앱 테마 tab */}
      {activeTab === 'theme' && <VerticalSwitcher />}

      {/* General settings — only shown on 'general' tab */}
      {activeTab === 'general' && <>

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
      <Section title={lang === 'en' ? 'Account' : '계정'}>
        {isLoggedIn ? (
          <>
            <Row icon={<Shield size={15} />} label={userEmail} desc={
              subscriptionStatus === 'trial'
                ? lang === 'en' ? `Trial · ${subscriptionExpiry ? 'until ' + new Date(subscriptionExpiry).toLocaleDateString('en-US') : ''}` : `체험판 · ${subscriptionExpiry ? new Date(subscriptionExpiry).toLocaleDateString('ko-KR') + '까지' : ''}`
                : subscriptionStatus === 'active' ? (lang === 'en' ? 'Subscribed' : '구독 중') : (lang === 'en' ? 'Subscription expired' : '구독 만료')
            }>
              <span style={{
                padding: '3px 10px', borderRadius: 20, fontSize: 11, fontWeight: 700,
                background: subscriptionStatus === 'active' ? 'rgba(34,197,94,0.15)' : subscriptionStatus === 'trial' ? 'rgba(79,126,247,0.15)' : 'rgba(239,68,68,0.15)',
                color: subscriptionStatus === 'active' ? 'var(--success)' : subscriptionStatus === 'trial' ? 'var(--accent-primary)' : 'var(--error)',
              }}>
                {subscriptionStatus === 'active' ? (lang === 'en' ? 'Active' : '활성') : subscriptionStatus === 'trial' ? (lang === 'en' ? 'Trial' : '체험') : (lang === 'en' ? 'Expired' : '만료')}
              </span>
            </Row>
            <Row icon={<span style={{ fontSize: 14 }}>🚪</span>} label={lang === 'en' ? 'Sign out' : '로그아웃'} desc="">
              <motion.button
                whileTap={{ scale: 0.95 }}
                onClick={setLoggedOut}
                style={{ padding: '6px 14px', borderRadius: 8, border: '1px solid var(--border-default)', background: 'transparent', color: 'var(--text-secondary)', fontSize: 12, fontWeight: 600, cursor: 'pointer' }}
              >
                {lang === 'en' ? 'Sign out' : '로그아웃'}
              </motion.button>
            </Row>
          </>
        ) : (
          <Row icon={<Shield size={15} />} label={lang === 'en' ? 'Login required' : '로그인 필요'} desc={lang === 'en' ? 'Sign in with your Google account' : 'Google 계정으로 로그인하세요'}>
            <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>{lang === 'en' ? 'Not signed in' : '미로그인'}</span>
          </Row>
        )}
      </Section>

      {/* 이메일 설정 */}
      <Section title={lang === 'en' ? 'Email (IMAP/SMTP)' : '이메일 (IMAP/SMTP)'}>
        <div style={{ padding: '8px 0', display: 'flex', flexDirection: 'column', gap: 8 }}>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 80px', gap: 6 }}>
            <div>
              <p style={{ fontSize: 11, color: 'var(--text-muted)', marginBottom: 3 }}>IMAP 서버</p>
              <input style={inputStyle} placeholder="imap.gmail.com" value={emailHost} onChange={e => setEmailHost(e.target.value)} />
            </div>
            <div>
              <p style={{ fontSize: 11, color: 'var(--text-muted)', marginBottom: 3 }}>포트</p>
              <input style={inputStyle} placeholder="993" value={emailPort} onChange={e => setEmailPort(e.target.value)} />
            </div>
          </div>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 80px', gap: 6 }}>
            <div>
              <p style={{ fontSize: 11, color: 'var(--text-muted)', marginBottom: 3 }}>SMTP 서버</p>
              <input style={inputStyle} placeholder="smtp.gmail.com" value={smtpHost} onChange={e => setSmtpHost(e.target.value)} />
            </div>
            <div>
              <p style={{ fontSize: 11, color: 'var(--text-muted)', marginBottom: 3 }}>포트</p>
              <input style={inputStyle} placeholder="587" value={smtpPort} onChange={e => setSmtpPort(e.target.value)} />
            </div>
          </div>
          <div>
            <p style={{ fontSize: 11, color: 'var(--text-muted)', marginBottom: 3 }}>이메일 주소</p>
            <input style={inputStyle} placeholder="you@gmail.com" type="email" value={emailUser} onChange={e => setEmailUser(e.target.value)} />
          </div>
          <div>
            <p style={{ fontSize: 11, color: 'var(--text-muted)', marginBottom: 3 }}>앱 비밀번호</p>
            <input style={inputStyle} placeholder="앱 비밀번호 (Google: 앱 비밀번호 생성)" type="password" value={emailPass} onChange={e => setEmailPass(e.target.value)} />
          </div>
          <motion.button
            whileTap={{ scale: 0.97 }}
            onClick={saveEmailConfig}
            disabled={emailSaving}
            style={{
              display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 6,
              padding: '7px 0', borderRadius: 8, border: 'none',
              background: emailSaved ? 'rgba(34,197,94,0.2)' : 'rgba(79,126,247,0.15)',
              color: emailSaved ? 'var(--success)' : 'var(--accent-primary)',
              fontSize: 12, fontWeight: 600, cursor: 'pointer', width: '100%',
            }}
          >
            {emailSaved ? '✓ 저장됨' : <><Save size={12} /> {emailSaving ? '저장 중...' : '이메일 설정 저장'}</>}
          </motion.button>
          <p style={{ fontSize: 10, color: 'var(--text-muted)', lineHeight: 1.4 }}>
            Gmail 사용 시: Google 계정 → 보안 → 2단계 인증 → 앱 비밀번호에서 생성하세요
          </p>
        </div>
      </Section>

      {/* 업데이트 */}
      <Section title="업데이트">
        <Row icon={<RefreshCw size={15} />} label="앱 업데이트" desc={
          updateStatus === 'available' ? `v${updateVersion} 설치 중...` :
          updateStatus === 'latest' ? '최신 버전입니다' :
          updateStatus === 'error' ? '업데이트 확인 실패' : '최신 버전 확인'
        }>
          <motion.button
            whileTap={{ scale: 0.95 }}
            onClick={checkUpdate}
            disabled={updateChecking}
            style={{
              padding: '6px 14px', borderRadius: 8, border: '1px solid var(--border-default)',
              background: updateStatus === 'available' ? 'rgba(34,197,94,0.15)' :
                          updateStatus === 'error' ? 'rgba(239,68,68,0.1)' : 'transparent',
              color: updateStatus === 'available' ? 'var(--success)' :
                     updateStatus === 'error' ? 'var(--error)' : 'var(--text-secondary)',
              fontSize: 12, fontWeight: 600, cursor: updateChecking ? 'not-allowed' : 'pointer',
              display: 'flex', alignItems: 'center', gap: 5,
            }}
          >
            <motion.span animate={{ rotate: updateChecking ? 360 : 0 }} transition={{ repeat: updateChecking ? Infinity : 0, duration: 1, ease: 'linear' }}>
              <RefreshCw size={12} />
            </motion.span>
            {updateChecking ? '확인 중...' : updateStatus === 'latest' ? '✓ 최신' : updateStatus === 'error' ? '재시도' : '업데이트 확인'}
          </motion.button>
        </Row>
      </Section>

      {/* 앱 정보 */}
      <Section title="정보">
        <Row icon={<Info size={15} />} label="Nexus" desc="Windows PC 관리 도구">
          <span style={{ fontSize: 12, color: 'var(--text-muted)', fontFamily: 'monospace' }}>v1.0.0</span>
        </Row>
      </Section>

      </>}
    </div>
  )
}
