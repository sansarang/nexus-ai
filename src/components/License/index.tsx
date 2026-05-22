import { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { useAppStore } from '../../stores/appStore'
import { signInWithGoogle } from '../../lib/supabase'
import { ADMIN_EMAIL, ADMIN_PASSWORD } from '../../config/services'

const GoogleIcon = () => (
  <svg width="20" height="20" viewBox="0 0 24 24">
    <path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"/>
    <path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/>
    <path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"/>
    <path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"/>
  </svg>
)

export function LicenseInput({ onSuccess }: { onSuccess?: () => void; compact?: boolean }) {
  const { setLoggedIn } = useAppStore()
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [showAdmin, setShowAdmin] = useState(false)
  const [adminEmail, setAdminEmail] = useState('')
  const [adminPassword, setAdminPassword] = useState('')

  const handleGoogleLogin = async () => {
    setLoading(true)
    setError('')
    try {
      await signInWithGoogle(() => {
        setLoading(false)
        onSuccess?.()
      })
    } catch (e: any) {
      console.error('Google OAuth failed:', e)
      setError('로그인 실패: ' + (e?.message || '다시 시도해주세요.'))
      setLoading(false)
    }
  }

  const handleAdminLogin = () => {
    setError('')
    if (adminEmail.trim() === ADMIN_EMAIL && adminPassword === ADMIN_PASSWORD) {
      setLoggedIn(ADMIN_EMAIL, 'active', '2099-12-31T00:00:00.000Z')
      onSuccess?.()
    } else {
      setError('이메일 또는 비밀번호가 올바르지 않습니다.')
    }
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      <button
        onClick={handleGoogleLogin}
        disabled={loading}
        style={{
          width: '100%', padding: '14px 20px',
          background: loading ? 'rgba(255,255,255,0.7)' : 'white',
          border: 'none', borderRadius: 12, cursor: loading ? 'wait' : 'pointer',
          display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 12,
          fontSize: 15, fontWeight: 700, color: '#1a1a2e',
          boxShadow: '0 4px 20px rgba(0,0,0,0.3)', transition: 'opacity 0.2s',
          opacity: loading ? 0.7 : 1,
        }}
      >
        <GoogleIcon />
        {loading ? '브라우저에서 로그인 중...' : 'Google 계정으로 시작하기'}
      </button>

      {loading && (
        <p style={{ fontSize: 12, color: 'var(--text-muted)', textAlign: 'center', margin: 0 }}>
          Chrome에서 Google 로그인 후 돌아오면 자동으로 시작됩니다
        </p>
      )}

      {error && <p style={{ fontSize: 12, color: '#f87171', margin: 0 }}>{error}</p>}

      <p style={{ fontSize: 11, color: 'var(--text-muted)', textAlign: 'center' }}>
        7일 무료 체험 · 이후 월 14,900원 · 언제든 해지 가능
      </p>

      {/* 관리자 로그인 숨김 */}
      <button
        onClick={() => { setShowAdmin(v => !v); setError('') }}
        style={{ background: 'none', border: 'none', cursor: 'pointer', fontSize: 11, color: 'transparent', padding: '2px 0' }}
      >.</button>

      <AnimatePresence>
        {showAdmin && (
          <motion.div
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: 'auto' }}
            exit={{ opacity: 0, height: 0 }}
            style={{ display: 'flex', flexDirection: 'column', gap: 8, overflow: 'hidden' }}
          >
            <input type="email" placeholder="관리자 이메일" value={adminEmail}
              onChange={e => setAdminEmail(e.target.value)}
              style={{ width: '100%', padding: '10px 14px', boxSizing: 'border-box', background: 'var(--bg-surface)', border: '1px solid var(--border-default)', borderRadius: 10, color: 'var(--text-primary)', fontSize: 13, outline: 'none' } as React.CSSProperties}
            />
            <input type="password" placeholder="비밀번호" value={adminPassword}
              onChange={e => setAdminPassword(e.target.value)}
              onKeyDown={e => e.key === 'Enter' && handleAdminLogin()}
              style={{ width: '100%', padding: '10px 14px', boxSizing: 'border-box', background: 'var(--bg-surface)', border: '1px solid var(--border-default)', borderRadius: 10, color: 'var(--text-primary)', fontSize: 13, outline: 'none' } as React.CSSProperties}
            />
            {error && <p style={{ fontSize: 11, color: 'var(--error, #f87171)', margin: 0 }}>{error}</p>}
            <button onClick={handleAdminLogin}
              style={{ width: '100%', padding: '10px', border: 'none', borderRadius: 10, background: 'var(--accent-primary, #4f7ef7)', color: 'white', fontSize: 13, fontWeight: 700, cursor: 'pointer' }}>
              로그인
            </button>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}

export function LicenseView() {
  const { setView } = useAppStore()
  return (
    <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 32 }}>
      <motion.div
        initial={{ opacity: 0, y: 12 }}
        animate={{ opacity: 1, y: 0 }}
        style={{
          width: '100%', maxWidth: 480,
          background: 'var(--bg-elevated)', border: '1px solid var(--border-subtle)',
          borderRadius: 'var(--radius-lg)', padding: 32,
          display: 'flex', flexDirection: 'column', gap: 20,
        }}
      >
        <div style={{ textAlign: 'center' }}>
          <div style={{ fontSize: 36, marginBottom: 8 }}>🔐</div>
          <h2 style={{ color: 'var(--text-primary)', fontWeight: 800, fontSize: 18, marginBottom: 6 }}>
            Nexus 시작하기
          </h2>
          <p style={{ color: 'var(--text-secondary)', fontSize: 13 }}>
            구글 계정으로 로그인하면 7일 무료 체험이 시작됩니다
          </p>
        </div>
        <LicenseInput onSuccess={() => setView('home')} />
      </motion.div>
    </div>
  )
}
