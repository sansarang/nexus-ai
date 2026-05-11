import React, { useState, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'

const API = 'http://127.0.0.1:17891'

interface Provider {
  id: string
  name: string
  icon: string
  color: string
  instructions: string
  emailPlaceholder: string
  passwordPlaceholder: string
}

const PROVIDERS: Provider[] = [
  {
    id: 'naver',
    name: '네이버 메일',
    icon: 'N',
    color: '#03c75a',
    instructions: '네이버 앱 비밀번호 발급 방법:\n1. 네이버 로그인 → 내 정보\n2. 보안설정 → 2단계 인증 활성화\n3. 앱 비밀번호 발급 → "Nexus" 이름으로 생성\n4. 발급된 비밀번호를 아래에 입력하세요',
    emailPlaceholder: 'example@naver.com',
    passwordPlaceholder: '앱 비밀번호 (8자리)',
  },
  {
    id: 'daum',
    name: '다음 메일',
    icon: 'D',
    color: '#ff5722',
    instructions: '다음 메일 IMAP 활성화:\n1. mail.daum.net → 환경설정\n2. 메일 관리 → IMAP/POP3 설정\n3. IMAP 사용 활성화\n4. 카카오계정 비밀번호 입력',
    emailPlaceholder: 'example@daum.net',
    passwordPlaceholder: '카카오계정 비밀번호',
  },
  {
    id: 'kakao',
    name: '카카오 메일',
    icon: '🐱',
    color: '#fee500',
    instructions: '카카오 메일 IMAP 설정:\n1. mail.kakao.com 로그인\n2. 설정 → 메일 클라이언트 설정\n3. IMAP 사용 활성화\n4. 앱 비밀번호 발급 후 입력',
    emailPlaceholder: 'example@kakao.com',
    passwordPlaceholder: '앱 비밀번호',
  },
  {
    id: 'gmail',
    name: 'Gmail',
    icon: 'G',
    color: '#ea4335',
    instructions: 'Gmail 앱 비밀번호 발급:\n1. myaccount.google.com\n2. 보안 → 2단계 인증 활성화\n3. 앱 비밀번호 → Nexus 앱 생성\n4. 생성된 16자리 비밀번호 입력',
    emailPlaceholder: 'example@gmail.com',
    passwordPlaceholder: '앱 비밀번호 (16자리)',
  },
  {
    id: 'custom',
    name: '커스텀 IMAP',
    icon: '⚙️',
    color: '#6b7280',
    instructions: 'IMAP/SMTP 서버 정보를 직접 입력하세요.\n이메일 제공업체의 설정 페이지에서 서버 주소를 확인하세요.',
    emailPlaceholder: 'your@email.com',
    passwordPlaceholder: '이메일 비밀번호',
  },
]

interface IMAPAccount {
  id: string
  name: string
  email: string
  provider: string
  created_at: string
}

interface EmailSetupProps {
  onClose: () => void
  primaryColor?: string
}

export function EmailSetup({ onClose, primaryColor = '#7c3aed' }: EmailSetupProps) {
  const [step, setStep] = useState<1 | 2 | 3 | 4>(1)
  const [selectedProvider, setSelectedProvider] = useState<Provider | null>(null)
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [imapHost, setImapHost] = useState('')
  const [imapPort, setImapPort] = useState('993')
  const [smtpHost, setSmtpHost] = useState('')
  const [smtpPort, setSmtpPort] = useState('587')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState(false)
  const [accounts, setAccounts] = useState<IMAPAccount[]>([])
  const [showInstructions, setShowInstructions] = useState(false)
  const [tab, setTab] = useState<'add' | 'accounts'>('add')

  const loadAccounts = async () => {
    try {
      const res = await fetch(`${API}/api/imap/accounts`)
      const data = await res.json()
      setAccounts(data.accounts || [])
    } catch { /* ignore */ }
  }

  useEffect(() => { loadAccounts() }, [])

  const handleProviderSelect = (p: Provider) => {
    setSelectedProvider(p)
    setStep(2)
    setError('')
  }

  const handleTest = async () => {
    if (!selectedProvider || !email || !password) {
      setError('이메일과 비밀번호를 입력해주세요')
      return
    }
    setLoading(true)
    setError('')
    setStep(3)
    try {
      const body: any = {
        name: name || email.split('@')[0],
        email, password,
        provider: selectedProvider.id,
      }
      if (selectedProvider.id === 'custom') {
        body.imap_host = imapHost
        body.imap_port = parseInt(imapPort, 10)
        body.smtp_host = smtpHost
        body.smtp_port = parseInt(smtpPort, 10)
      }
      const res = await fetch(`${API}/api/imap/accounts`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
      const data = await res.json()
      if (data.success) {
        setSuccess(true)
        setStep(4)
        loadAccounts()
      } else {
        setError(data.message || '연결 실패')
        setStep(2)
      }
    } catch (err) {
      setError('백엔드 연결 실패')
      setStep(2)
    } finally {
      setLoading(false)
    }
  }

  const handleDelete = async (id: string) => {
    try {
      await fetch(`${API}/api/imap/accounts?id=${id}`, { method: 'DELETE' })
      loadAccounts()
    } catch { /* ignore */ }
  }

  const resetForm = () => {
    setStep(1)
    setSelectedProvider(null)
    setName('')
    setEmail('')
    setPassword('')
    setImapHost('')
    setSmtpHost('')
    setError('')
    setSuccess(false)
    setShowInstructions(false)
  }

  const providerColor = selectedProvider?.color || primaryColor

  return (
    <motion.div
      initial={{ opacity: 0, y: 20, scale: 0.96 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      exit={{ opacity: 0, y: 20, scale: 0.96 }}
      style={{
        position: 'fixed', bottom: 100, right: 90, width: 420,
        background: 'rgba(6,6,18,0.98)', backdropFilter: 'blur(20px)',
        border: `1px solid ${primaryColor}44`, borderRadius: 20,
        boxShadow: `0 24px 64px rgba(0,0,0,0.7)`,
        zIndex: 10005, overflow: 'hidden', fontFamily: 'inherit',
      }}
    >
      {/* Header */}
      <div style={{
        padding: '14px 18px', display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        borderBottom: '1px solid rgba(255,255,255,0.07)',
        background: `linear-gradient(135deg, ${primaryColor}18, transparent)`,
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <span style={{ fontSize: 18 }}>📧</span>
          <span style={{ fontWeight: 700, fontSize: 14, color: '#fff' }}>이메일 계정 설정</span>
        </div>
        <div style={{ display: 'flex', gap: 6 }}>
          {(['add', 'accounts'] as const).map(t => (
            <button key={t} onClick={() => setTab(t)} style={{
              padding: '4px 10px', borderRadius: 8, border: 'none', cursor: 'pointer', fontSize: 11,
              background: tab === t ? `${primaryColor}44` : 'rgba(255,255,255,0.06)',
              color: tab === t ? primaryColor : 'rgba(255,255,255,0.5)', fontWeight: 600,
            }}>
              {t === 'add' ? '+ 추가' : `계정 (${accounts.length})`}
            </button>
          ))}
          <button onClick={onClose} style={{
            background: 'none', border: 'none', color: 'rgba(255,255,255,0.4)',
            cursor: 'pointer', fontSize: 16, marginLeft: 4,
          }}>✕</button>
        </div>
      </div>

      <div style={{ padding: 20 }}>
        {tab === 'accounts' ? (
          /* 계정 목록 탭 */
          <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
            {accounts.length === 0 ? (
              <div style={{ textAlign: 'center', color: 'rgba(255,255,255,0.25)', fontSize: 13, padding: '40px 0' }}>
                <div style={{ fontSize: 32, marginBottom: 8 }}>📭</div>
                설정된 이메일 계정이 없어요
              </div>
            ) : accounts.map(acc => {
              const prov = PROVIDERS.find(p => p.id === acc.provider)
              return (
                <div key={acc.id} style={{
                  display: 'flex', alignItems: 'center', gap: 12, padding: '10px 14px',
                  background: 'rgba(255,255,255,0.04)', borderRadius: 12,
                  border: '1px solid rgba(255,255,255,0.07)',
                }}>
                  <div style={{
                    width: 36, height: 36, borderRadius: 10,
                    background: `${prov?.color || '#6b7280'}22`,
                    display: 'flex', alignItems: 'center', justifyContent: 'center',
                    color: prov?.color || '#6b7280', fontWeight: 800, fontSize: 14, flexShrink: 0,
                  }}>
                    {prov?.icon || '📧'}
                  </div>
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ fontWeight: 600, fontSize: 13, color: '#fff', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      {acc.name}
                    </div>
                    <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.4)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      {acc.email} · {prov?.name || acc.provider}
                    </div>
                  </div>
                  <button onClick={() => handleDelete(acc.id)} style={{
                    padding: '5px 10px', borderRadius: 8, border: '1px solid rgba(239,68,68,0.4)',
                    background: 'rgba(239,68,68,0.1)', color: '#ef4444', fontSize: 11, cursor: 'pointer',
                  }}>삭제</button>
                </div>
              )
            })}
          </div>
        ) : (
          /* 계정 추가 탭 */
          <AnimatePresence mode="wait">
            {step === 1 && (
              <motion.div key="step1" initial={{ opacity: 0, x: 10 }} animate={{ opacity: 1, x: 0 }} exit={{ opacity: 0, x: -10 }}>
                <div style={{ fontSize: 12, color: 'rgba(255,255,255,0.5)', marginBottom: 14, textAlign: 'center' }}>
                  이메일 제공업체를 선택하세요
                </div>
                <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
                  {PROVIDERS.map(p => (
                    <button key={p.id} onClick={() => handleProviderSelect(p)} style={{
                      display: 'flex', alignItems: 'center', gap: 14, padding: '12px 16px',
                      border: `1px solid ${p.color}44`, borderRadius: 14,
                      background: `${p.color}0a`, cursor: 'pointer', textAlign: 'left',
                      transition: 'all 0.15s',
                    }}>
                      <div style={{
                        width: 40, height: 40, borderRadius: 12,
                        background: `${p.color}22`, border: `1px solid ${p.color}44`,
                        display: 'flex', alignItems: 'center', justifyContent: 'center',
                        color: p.id === 'kakao' ? p.color : p.color,
                        fontWeight: 900, fontSize: p.icon.length === 1 ? 18 : 20, flexShrink: 0,
                      }}>
                        {p.icon}
                      </div>
                      <div>
                        <div style={{ fontWeight: 700, fontSize: 13, color: '#fff' }}>{p.name}</div>
                        <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.4)', marginTop: 1 }}>
                          {p.id === 'custom' ? '직접 서버 설정' : `${p.id}.com`}
                        </div>
                      </div>
                      <span style={{ marginLeft: 'auto', color: 'rgba(255,255,255,0.3)', fontSize: 14 }}>›</span>
                    </button>
                  ))}
                </div>
              </motion.div>
            )}

            {step === 2 && selectedProvider && (
              <motion.div key="step2" initial={{ opacity: 0, x: 10 }} animate={{ opacity: 1, x: 0 }} exit={{ opacity: 0, x: -10 }}>
                {/* Provider header */}
                <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 16 }}>
                  <button onClick={resetForm} style={{
                    background: 'none', border: 'none', color: 'rgba(255,255,255,0.4)',
                    cursor: 'pointer', fontSize: 16, padding: 0,
                  }}>‹</button>
                  <div style={{
                    width: 32, height: 32, borderRadius: 8,
                    background: `${selectedProvider.color}22`, display: 'flex', alignItems: 'center', justifyContent: 'center',
                    color: selectedProvider.color, fontWeight: 800, fontSize: 14,
                  }}>
                    {selectedProvider.icon}
                  </div>
                  <span style={{ fontWeight: 700, fontSize: 13, color: '#fff' }}>{selectedProvider.name}</span>
                </div>

                {/* Instructions toggle */}
                <button onClick={() => setShowInstructions(p => !p)} style={{
                  width: '100%', marginBottom: 14, padding: '8px 12px',
                  background: `${providerColor}11`, border: `1px solid ${providerColor}33`,
                  borderRadius: 10, color: providerColor, fontSize: 11, fontWeight: 600, cursor: 'pointer',
                  textAlign: 'left',
                }}>
                  ℹ️ {showInstructions ? '가이드 숨기기' : `${selectedProvider.name} 설정 방법 보기`}
                </button>
                <AnimatePresence>
                  {showInstructions && (
                    <motion.div
                      initial={{ height: 0, opacity: 0 }} animate={{ height: 'auto', opacity: 1 }} exit={{ height: 0, opacity: 0 }}
                      style={{ overflow: 'hidden', marginBottom: 14 }}
                    >
                      <div style={{
                        padding: '10px 14px', background: `${providerColor}0d`,
                        border: `1px solid ${providerColor}22`, borderRadius: 10,
                        fontSize: 11, color: 'rgba(255,255,255,0.6)', lineHeight: 1.7,
                        whiteSpace: 'pre-line',
                      }}>
                        {selectedProvider.instructions}
                      </div>
                    </motion.div>
                  )}
                </AnimatePresence>

                {/* Form */}
                <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
                  {[
                    { label: '이름', value: name, setter: setName, placeholder: '홍길동' },
                    { label: '이메일', value: email, setter: setEmail, placeholder: selectedProvider.emailPlaceholder, type: 'email' },
                    { label: selectedProvider.id === 'custom' ? '비밀번호' : '앱 비밀번호', value: password, setter: setPassword, placeholder: selectedProvider.passwordPlaceholder, type: 'password' },
                  ].map(field => (
                    <div key={field.label}>
                      <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.45)', marginBottom: 4 }}>{field.label}</div>
                      <input
                        type={(field as any).type || 'text'}
                        value={field.value}
                        onChange={e => field.setter(e.target.value)}
                        placeholder={field.placeholder}
                        style={{
                          width: '100%', background: 'rgba(255,255,255,0.05)',
                          border: `1px solid ${field.value ? providerColor + '66' : 'rgba(255,255,255,0.1)'}`,
                          borderRadius: 10, padding: '9px 12px', color: '#fff', fontSize: 12,
                          outline: 'none', boxSizing: 'border-box',
                        }}
                      />
                    </div>
                  ))}

                  {selectedProvider.id === 'custom' && (
                    <>
                      <div style={{ display: 'flex', gap: 8 }}>
                        <div style={{ flex: 3 }}>
                          <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.45)', marginBottom: 4 }}>IMAP 서버</div>
                          <input value={imapHost} onChange={e => setImapHost(e.target.value)} placeholder="imap.example.com"
                            style={{ width: '100%', background: 'rgba(255,255,255,0.05)', border: '1px solid rgba(255,255,255,0.1)', borderRadius: 10, padding: '9px 12px', color: '#fff', fontSize: 12, outline: 'none', boxSizing: 'border-box' }} />
                        </div>
                        <div style={{ flex: 1 }}>
                          <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.45)', marginBottom: 4 }}>포트</div>
                          <input value={imapPort} onChange={e => setImapPort(e.target.value)} placeholder="993"
                            style={{ width: '100%', background: 'rgba(255,255,255,0.05)', border: '1px solid rgba(255,255,255,0.1)', borderRadius: 10, padding: '9px 12px', color: '#fff', fontSize: 12, outline: 'none', boxSizing: 'border-box' }} />
                        </div>
                      </div>
                      <div style={{ display: 'flex', gap: 8 }}>
                        <div style={{ flex: 3 }}>
                          <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.45)', marginBottom: 4 }}>SMTP 서버</div>
                          <input value={smtpHost} onChange={e => setSmtpHost(e.target.value)} placeholder="smtp.example.com"
                            style={{ width: '100%', background: 'rgba(255,255,255,0.05)', border: '1px solid rgba(255,255,255,0.1)', borderRadius: 10, padding: '9px 12px', color: '#fff', fontSize: 12, outline: 'none', boxSizing: 'border-box' }} />
                        </div>
                        <div style={{ flex: 1 }}>
                          <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.45)', marginBottom: 4 }}>포트</div>
                          <input value={smtpPort} onChange={e => setSmtpPort(e.target.value)} placeholder="587"
                            style={{ width: '100%', background: 'rgba(255,255,255,0.05)', border: '1px solid rgba(255,255,255,0.1)', borderRadius: 10, padding: '9px 12px', color: '#fff', fontSize: 12, outline: 'none', boxSizing: 'border-box' }} />
                        </div>
                      </div>
                    </>
                  )}

                  {error && (
                    <div style={{ padding: '8px 12px', background: 'rgba(239,68,68,0.1)', borderRadius: 8, color: '#ef4444', fontSize: 11 }}>
                      ❌ {error}
                    </div>
                  )}

                  <button onClick={handleTest} disabled={!email || !password} style={{
                    padding: '11px', borderRadius: 12, border: 'none',
                    background: email && password ? `linear-gradient(135deg, ${providerColor}, ${providerColor}99)` : 'rgba(255,255,255,0.08)',
                    color: email && password ? '#fff' : 'rgba(255,255,255,0.3)',
                    fontSize: 13, fontWeight: 700, cursor: email && password ? 'pointer' : 'not-allowed',
                    marginTop: 4,
                  }}>
                    연결 테스트 및 저장
                  </button>
                </div>
              </motion.div>
            )}

            {step === 3 && (
              <motion.div key="step3" initial={{ opacity: 0 }} animate={{ opacity: 1 }} style={{ textAlign: 'center', padding: '30px 0' }}>
                <motion.div
                  animate={{ rotate: 360 }} transition={{ duration: 1, repeat: Infinity, ease: 'linear' }}
                  style={{ fontSize: 36, display: 'inline-block', marginBottom: 16 }}
                >⟳</motion.div>
                <div style={{ fontSize: 14, fontWeight: 600, color: '#fff', marginBottom: 6 }}>연결 테스트 중...</div>
                <div style={{ fontSize: 12, color: 'rgba(255,255,255,0.4)' }}>
                  {selectedProvider?.name} IMAP 서버에 연결 중
                </div>
              </motion.div>
            )}

            {step === 4 && success && (
              <motion.div key="step4" initial={{ opacity: 0, scale: 0.9 }} animate={{ opacity: 1, scale: 1 }} style={{ textAlign: 'center', padding: '30px 0' }}>
                <div style={{ fontSize: 48, marginBottom: 16 }}>✅</div>
                <div style={{ fontSize: 15, fontWeight: 700, color: '#22c55e', marginBottom: 8 }}>연결 성공!</div>
                <div style={{ fontSize: 12, color: 'rgba(255,255,255,0.5)', marginBottom: 24 }}>
                  {email} 계정이 추가됐습니다
                </div>
                <div style={{ display: 'flex', gap: 10, justifyContent: 'center' }}>
                  <button onClick={resetForm} style={{
                    padding: '9px 20px', borderRadius: 10,
                    border: `1px solid ${providerColor}44`, background: `${providerColor}11`,
                    color: providerColor, fontSize: 12, fontWeight: 600, cursor: 'pointer',
                  }}>다른 계정 추가</button>
                  <button onClick={() => setTab('accounts')} style={{
                    padding: '9px 20px', borderRadius: 10, border: 'none',
                    background: `linear-gradient(135deg, ${providerColor}, ${providerColor}99)`,
                    color: '#fff', fontSize: 12, fontWeight: 600, cursor: 'pointer',
                  }}>계정 목록 보기</button>
                </div>
              </motion.div>
            )}
          </AnimatePresence>
        )}
      </div>
    </motion.div>
  )
}

export default EmailSetup
