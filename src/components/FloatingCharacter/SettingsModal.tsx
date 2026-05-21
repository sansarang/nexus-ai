import { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { useAppStore } from '../../stores/appStore'
import { openCheckout, openBillingPortal } from '../../lib/paddle'

interface SettingsModalProps {
  open: boolean
  onClose: () => void
  primaryColor: string
}

export function SettingsModal({ open, onClose, primaryColor }: SettingsModalProps) {
  const { micEnabled, setMicEnabled, userEmail, subscriptionStatus, subscriptionExpiry, setLoggedOut } = useAppStore()
  const [clarifyAutoMic, setClarifyAutoMic] = useState(localStorage.getItem('nexus-clarify-auto-mic') !== 'false')
  const [groqKey,     setGroqKey]     = useState(localStorage.getItem('nexus-groq-key') ?? '')
  const [pplxKey,     setPplxKey]     = useState(localStorage.getItem('nexus-pplx-key') ?? '')
  const [openaiKey,   setOpenaiKey]   = useState(localStorage.getItem('nexus-openai-key') ?? '')
  const [groqStatus,  setGroqStatus]  = useState<'idle' | 'testing' | 'ok' | 'fail'>('idle')
  const [ollamaUrl,   setOllamaUrl]   = useState(localStorage.getItem('nexus-ollama-url') ?? 'http://localhost:11434')
  const [emailTo,           setEmailTo]           = useState(localStorage.getItem('nexus-report-email') ?? '')
  const [customInstructions, setCustomInstructions] = useState(localStorage.getItem('nexus-custom-instructions') ?? '')
  const [saved,             setSaved]             = useState(false)
  const [tab,         setTab]         = useState<'account' | 'ai' | 'email' | 'about'>('account')
  const [pplxStatus,  setPplxStatus]  = useState<'idle' | 'testing' | 'ok' | 'fail'>('idle')

  const subLabel = {
    active:  { text: '구독 중', color: '#4ade80' },
    trial:   { text: '7일 무료 체험 중', color: '#facc15' },
    expired: { text: '구독 만료', color: '#f87171' },
    none:    { text: '미가입', color: '#718096' },
  }[subscriptionStatus]

  const expiryFormatted = subscriptionExpiry
    ? new Date(subscriptionExpiry).toLocaleDateString('ko-KR', { year: 'numeric', month: 'long', day: 'numeric' })
    : ''

  const testGroq = async () => {
    const key = groqKey.trim()
    if (!key) return
    setGroqStatus('testing')
    try {
      const res = await fetch('https://api.groq.com/openai/v1/chat/completions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${key}` },
        body: JSON.stringify({
          model: 'llama-3.1-8b-instant',
          messages: [{ role: 'user', content: 'hi' }],
          max_tokens: 5,
        }),
      })
      setGroqStatus(res.ok ? 'ok' : 'fail')
    } catch {
      setGroqStatus('fail')
    }
    setTimeout(() => setGroqStatus('idle'), 3000)
  }

  const save = () => {
    // Groq 키 저장 (핵심 AI)
    const gKey = groqKey.trim()
    if (gKey) localStorage.setItem('nexus-groq-key', gKey)
    else      localStorage.removeItem('nexus-groq-key')

    const key = pplxKey.trim()
    if (key) localStorage.setItem('nexus-pplx-key', key)
    else     localStorage.removeItem('nexus-pplx-key')

    if (openaiKey.trim()) localStorage.setItem('nexus-openai-key', openaiKey.trim())
    else                  localStorage.removeItem('nexus-openai-key')

    localStorage.setItem('nexus-ollama-url', ollamaUrl.trim() || 'http://localhost:11434')

    if (emailTo.trim()) localStorage.setItem('nexus-report-email', emailTo.trim())

    if (customInstructions.trim()) localStorage.setItem('nexus-custom-instructions', customInstructions.trim())
    else localStorage.removeItem('nexus-custom-instructions')

    // 백엔드에 API 키 즉시 동기화
    fetch('http://127.0.0.1:17891/api/llm/config', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        groq_key: gKey || undefined,
        perplexity_key: key || undefined,
        claude_key: openaiKey.trim() || undefined,
      }),
    }).catch(() => {})

    setSaved(true)
    setTimeout(() => { setSaved(false); onClose() }, 1400)
  }

  const testPplx = async () => {
    const key = pplxKey.trim()
    if (!key) return
    setPplxStatus('testing')
    try {
      const res = await fetch('https://api.perplexity.ai/chat/completions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${key}` },
        body: JSON.stringify({
          model: 'sonar',
          messages: [{ role: 'user', content: '안녕? 한 단어로만 대답해.' }],
          max_tokens: 10,
        }),
      })
      setPplxStatus(res.ok ? 'ok' : 'fail')
    } catch {
      setPplxStatus('fail')
    }
    setTimeout(() => setPplxStatus('idle'), 3000)
  }

  const inputStyle = (active: boolean): React.CSSProperties => ({
    background: 'rgba(255,255,255,0.05)',
    border: `1px solid ${active ? primaryColor + '88' : 'rgba(255,255,255,0.1)'}`,
    borderRadius: 10,
    padding: '9px 12px',
    color: 'rgba(255,255,255,0.9)',
    fontSize: 13,
    outline: 'none',
    fontFamily: 'monospace',
    width: '100%',
    boxSizing: 'border-box' as const,
    transition: 'border-color 0.2s',
  })

  const labelStyle: React.CSSProperties = {
    fontSize: 11, fontWeight: 700, letterSpacing: '0.05em',
  }

  const tabStyle = (active: boolean): React.CSSProperties => ({
    flex: 1, padding: '6px 0', borderRadius: 8, border: 'none', cursor: 'pointer',
    background: active ? `${primaryColor}33` : 'transparent',
    color: active ? primaryColor : 'rgba(255,255,255,0.4)',
    fontSize: 12, fontWeight: active ? 700 : 400,
    transition: 'all 0.15s',
  })

  return (
    <AnimatePresence>
      {open && (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          onClick={onClose}
          style={{
            position: 'fixed', inset: 0,
            background: 'rgba(0,0,0,0.65)',
            backdropFilter: 'blur(8px)',
            zIndex: 99999,
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            pointerEvents: 'auto',
          }}
        >
          <motion.div
            initial={{ scale: 0.88, y: 20, opacity: 0 }}
            animate={{ scale: 1, y: 0, opacity: 1 }}
            exit={{ scale: 0.88, y: 20, opacity: 0 }}
            transition={{ duration: 0.22, ease: [0.4, 0, 0.2, 1] }}
            onClick={e => e.stopPropagation()}
            style={{
              width: 400,
              maxHeight: '90vh',
              overflowY: 'auto',
              background: 'rgba(10,10,20,0.98)',
              border: `1px solid ${primaryColor}44`,
              borderRadius: 20,
              padding: 24,
              boxShadow: `0 24px 64px rgba(0,0,0,0.8), 0 0 0 1px ${primaryColor}22`,
              display: 'flex',
              flexDirection: 'column',
              gap: 16,
              pointerEvents: 'auto',
            }}
          >
            {/* 헤더 */}
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <span style={{ fontSize: 20 }}>⚙️</span>
                <span style={{ fontSize: 15, fontWeight: 800, color: 'rgba(255,255,255,0.95)' }}>
                  Nexus 설정
                </span>
              </div>
              <button onClick={onClose} style={{
                width: 28, height: 28, borderRadius: '50%', border: 'none',
                background: 'rgba(255,255,255,0.08)', color: 'rgba(255,255,255,0.5)',
                fontSize: 14, cursor: 'pointer',
              }}>✕</button>
            </div>

            {/* 탭 */}
            <div style={{ display: 'flex', gap: 4, background: 'rgba(255,255,255,0.04)', borderRadius: 10, padding: 4 }}>
              {([['account', '👤 계정'], ['ai', '🤖 AI 설정'], ['email', '📧 이메일'], ['about', 'ℹ️ 정보']] as const).map(([key, label]) => (
                <button key={key} onClick={() => setTab(key)} style={tabStyle(tab === key)}>{label}</button>
              ))}
            </div>

            {/* ── 계정 탭 ── */}
            {tab === 'account' && (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                {/* 계정 정보 */}
                <div style={{
                  background: 'rgba(255,255,255,0.04)', borderRadius: 12, padding: '16px',
                  display: 'flex', alignItems: 'center', gap: 14,
                }}>
                  <div style={{
                    width: 46, height: 46, borderRadius: '50%',
                    background: `linear-gradient(135deg,${primaryColor},${primaryColor}88)`,
                    display: 'flex', alignItems: 'center', justifyContent: 'center',
                    fontSize: 20, flexShrink: 0,
                  }}>👤</div>
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ fontSize: 13, fontWeight: 700, color: 'white', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      {userEmail || '로그인 필요'}
                    </div>
                    <div style={{ fontSize: 11, color: subLabel.color, marginTop: 3, fontWeight: 600 }}>
                      ● {subLabel.text}
                      {expiryFormatted && <span style={{ color: 'rgba(255,255,255,0.35)', fontWeight: 400, marginLeft: 6 }}>~{expiryFormatted}</span>}
                    </div>
                  </div>
                </div>

                {/* 구독 만료 경고 */}
                {(subscriptionStatus === 'expired' || subscriptionStatus === 'none') && (
                  <div style={{
                    background: 'rgba(248,113,113,0.1)', border: '1px solid rgba(248,113,113,0.3)',
                    borderRadius: 10, padding: '12px 14px', fontSize: 12, color: '#fca5a5',
                  }}>
                    ⚠️ 구독이 {subscriptionStatus === 'expired' ? '만료되었습니다' : '없습니다'}. 일부 AI 기능이 제한됩니다.
                  </div>
                )}

                {/* 구독하기 버튼 — Paddle Checkout */}
                {(subscriptionStatus === 'expired' || subscriptionStatus === 'none' || subscriptionStatus === 'trial') && (
                  <button
                    onClick={() => openCheckout(userEmail, localStorage.getItem('nexus-user-id') ?? undefined)}
                    style={{
                      padding: '12px', borderRadius: 10, border: 'none', cursor: 'pointer',
                      background: `linear-gradient(135deg,${primaryColor},${primaryColor}cc)`,
                      color: 'white', fontSize: 14, fontWeight: 800,
                    }}
                  >
                    💳 {subscriptionStatus === 'trial' ? '지금 구독하기' : '구독하기'} — 월 9,900원
                  </button>
                )}

                {/* 구독 관리 — Paddle 빌링 포털 */}
                {subscriptionStatus === 'active' && (
                  <button
                    onClick={() => openBillingPortal(userEmail)}
                    style={{
                      padding: '10px', borderRadius: 10, border: '1px solid rgba(255,255,255,0.12)',
                      background: 'transparent', color: 'rgba(255,255,255,0.6)', fontSize: 13, cursor: 'pointer',
                    }}
                  >
                    구독 관리 (결제수단 변경 · 해지)
                  </button>
                )}

                {/* 로그아웃 */}
                {userEmail && (
                  <button
                    onClick={() => { setLoggedOut(); onClose() }}
                    style={{
                      padding: '10px', borderRadius: 10, border: '1px solid rgba(248,113,113,0.3)',
                      background: 'rgba(248,113,113,0.06)', color: '#f87171', fontSize: 13, cursor: 'pointer',
                    }}
                  >
                    로그아웃
                  </button>
                )}

                {/* 혜택 안내 */}
                <div style={{ background: 'rgba(255,255,255,0.02)', borderRadius: 10, padding: '12px 14px', fontSize: 11, color: 'rgba(255,255,255,0.4)', lineHeight: 1.9 }}>
                  <div style={{ color: 'rgba(255,255,255,0.6)', fontWeight: 700, marginBottom: 4 }}>구독 혜택</div>
                  <div>✦ 모든 AI 기능 무제한</div>
                  <div>✦ 실시간 웹 검색 (Perplexity)</div>
                  <div>✦ 자동 업데이트</div>
                  <div>✦ 언제든 해지 가능</div>
                </div>
              </div>
            )}

            {/* ── AI 설정 탭 ── */}
            {tab === 'ai' && (
              <>
                {/* 마이크 / 웨이크워드 */}
                <div style={{
                  display: 'flex', justifyContent: 'space-between', alignItems: 'center',
                  background: 'rgba(255,255,255,0.04)', borderRadius: 10,
                  padding: '12px 16px', marginBottom: 4,
                }}>
                  <div>
                    <div style={{ fontSize: 13, fontWeight: 600, color: 'white' }}>🎙️ 음성 인식 (웨이크워드)</div>
                    <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.4)', marginTop: 2 }}>
                      {micEnabled ? `"${localStorage.getItem('nexus-assistant-name') ?? 'Nexus'}" 라고 부르면 활성화` : '비활성화 — 버튼으로만 사용'}
                    </div>
                  </div>
                  <button
                    onClick={() => setMicEnabled(!micEnabled)}
                    style={{
                      width: 44, height: 24, borderRadius: 12, border: 'none',
                      background: micEnabled ? primaryColor : 'rgba(255,255,255,0.15)',
                      cursor: 'pointer', position: 'relative', transition: 'background 0.2s',
                    }}
                  >
                    <div style={{
                      position: 'absolute', top: 3,
                      left: micEnabled ? 23 : 3,
                      width: 18, height: 18, borderRadius: '50%',
                      background: 'white', transition: 'left 0.2s',
                    }} />
                  </button>
                </div>

                {/* clarify 후 마이크 자동 시작 */}
                <div style={{
                  display: 'flex', justifyContent: 'space-between', alignItems: 'center',
                  background: 'rgba(255,255,255,0.04)', borderRadius: 10,
                  padding: '12px 16px', marginBottom: 4,
                }}>
                  <div>
                    <div style={{ fontSize: 13, fontWeight: 600, color: 'white' }}>🎤 질문 후 마이크 자동 시작</div>
                    <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.4)', marginTop: 2 }}>
                      AI가 되물을 때 자동으로 마이크 켜기
                    </div>
                  </div>
                  <button
                    onClick={() => {
                      const next = !clarifyAutoMic
                      setClarifyAutoMic(next)
                      localStorage.setItem('nexus-clarify-auto-mic', String(next))
                    }}
                    style={{
                      width: 44, height: 24, borderRadius: 12, border: 'none',
                      background: clarifyAutoMic ? primaryColor : 'rgba(255,255,255,0.15)',
                      cursor: 'pointer', position: 'relative', transition: 'background 0.2s',
                    }}
                  >
                    <div style={{
                      position: 'absolute', top: 3,
                      left: clarifyAutoMic ? 23 : 3,
                      width: 18, height: 18, borderRadius: '50%',
                      background: 'white', transition: 'left 0.2s',
                    }} />
                  </button>
                </div>

                {/* Groq API 키 (핵심 AI — 워크플로우/직업군/멀티에이전트/회의) */}
                <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <label style={{ ...labelStyle, color: '#f6e05e' }}>
                      🔑 GROQ API KEY <span style={{ color: '#fc8181', fontSize: 10 }}>★ 필수</span> (워크플로우·직업군·회의·멀티에이전트)
                    </label>
                    <button onClick={testGroq} style={{
                      padding: '2px 10px', borderRadius: 6, border: 'none', cursor: 'pointer',
                      background: groqStatus === 'ok' ? 'rgba(72,187,120,0.3)'
                        : groqStatus === 'fail' ? 'rgba(252,129,129,0.3)'
                        : groqStatus === 'testing' ? 'rgba(237,137,54,0.3)'
                        : 'rgba(255,255,255,0.08)',
                      color: groqStatus === 'ok' ? '#68d391' : groqStatus === 'fail' ? '#fc8181' : '#a0aec0',
                      fontSize: 11,
                    }}>
                      {groqStatus === 'testing' ? '⏳ 확인 중...' : groqStatus === 'ok' ? '✅ 연결됨' : groqStatus === 'fail' ? '❌ 실패' : '연결 테스트'}
                    </button>
                  </div>
                  <input
                    type="password"
                    value={groqKey}
                    onChange={e => setGroqKey(e.target.value)}
                    placeholder="gsk_..."
                    autoComplete="off"
                    style={inputStyle(!!groqKey)}
                  />
                  <div style={{ fontSize: 10, color: '#4a5568' }}>
                    무료 키 발급: https://console.groq.com
                  </div>
                </div>

                {/* Perplexity API 키 (메인) */}
                <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <label style={{ ...labelStyle, color: '#f6ad55' }}>
                      ⚡ PERPLEXITY API KEY (AI 검색 · 웹 실시간)
                    </label>
                    <button onClick={testPplx} style={{
                      padding: '2px 10px', borderRadius: 6, border: 'none', cursor: 'pointer',
                      background: pplxStatus === 'ok' ? 'rgba(72,187,120,0.3)'
                        : pplxStatus === 'fail' ? 'rgba(252,129,129,0.3)'
                        : pplxStatus === 'testing' ? 'rgba(237,137,54,0.3)'
                        : 'rgba(255,255,255,0.08)',
                      color: pplxStatus === 'ok' ? '#68d391' : pplxStatus === 'fail' ? '#fc8181' : '#a0aec0',
                      fontSize: 11,
                    }}>
                      {pplxStatus === 'testing' ? '⏳ 확인 중...' : pplxStatus === 'ok' ? '✅ 연결됨' : pplxStatus === 'fail' ? '❌ 실패' : '연결 테스트'}
                    </button>
                  </div>
                  <div style={{ position: 'relative' }}>
                    <input
                      type="password"
                      value={pplxKey}
                      onChange={e => setPplxKey(e.target.value)}
                      placeholder="pplx-..."
                      autoComplete="off"
                      style={inputStyle(!!pplxKey)}
                    />
                  </div>
                  <div style={{ fontSize: 10, color: '#4a5568' }}>
                    https://www.perplexity.ai/settings/api 에서 키 발급
                  </div>
                </div>

                {/* Ollama URL */}
                <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                  <label style={{ ...labelStyle, color: '#68d391' }}>
                    🦙 OLLAMA 서버 (로컬 무료 LLM · 우선순위 1위)
                  </label>
                  <input
                    type="text"
                    value={ollamaUrl}
                    onChange={e => setOllamaUrl(e.target.value)}
                    placeholder="http://localhost:11434"
                    style={inputStyle(true)}
                  />
                  <span style={{ fontSize: 10, color: 'rgba(255,255,255,0.3)' }}>
                    Ollama 실행 중이면 비용 0원으로 오프라인 AI 사용 가능
                  </span>
                </div>

                {/* OpenAI TTS */}
                <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                  <label style={{ ...labelStyle, color: primaryColor }}>
                    🎵 OPENAI API KEY (고품질 TTS · 선택)
                  </label>
                  <input
                    type="password"
                    value={openaiKey}
                    onChange={e => setOpenaiKey(e.target.value)}
                    placeholder="sk-proj-..."
                    style={inputStyle(!!openaiKey)}
                  />
                  <span style={{ fontSize: 10, color: 'rgba(255,255,255,0.3)' }}>
                    없으면 브라우저 기본 TTS 사용 (무료)
                  </span>
                </div>

                {/* AI 우선순위 안내 */}
                <div style={{ background: 'rgba(255,255,255,0.03)', borderRadius: 10, padding: '10px 12px', fontSize: 11 }}>
                  <div style={{ color: '#a0aec0', marginBottom: 4, fontWeight: 600 }}>⚡ AI 응답 우선순위</div>
                  {[
                    ['1위', 'Ollama 로컬', '완전 무료 · 오프라인', '#68d391'],
                    ['2위', 'Perplexity API', '웹 검색 내장 · 실시간 정보', '#f6ad55'],
                    ['3위', '내장 키워드', '항상 동작 · LLM 불필요', '#90cdf4'],
                  ].map(([rank, name, desc, color]) => (
                    <div key={rank} style={{ display: 'flex', gap: 8, alignItems: 'center', padding: '3px 0' }}>
                      <span style={{ color, fontWeight: 700, minWidth: 28 }}>{rank}</span>
                      <span style={{ color: '#e2e8f0' }}>{name}</span>
                      <span style={{ color: '#718096', fontSize: 10 }}>{desc}</span>
                    </div>
                  ))}
                </div>
              </>
            )}

            {/* ── AI 설정 탭 — Custom Instructions ── */}
            {tab === 'ai' && (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                <label style={{ ...labelStyle, color: '#b794f4' }}>
                  🧠 Custom Instructions (나만의 AI 스타일 설정)
                </label>
                <textarea
                  value={customInstructions}
                  onChange={e => setCustomInstructions(e.target.value)}
                  placeholder={"예시:\n- 항상 bullet point로 정리해줘\n- 답변은 3줄 이내로\n- 요리 레시피는 재료표 먼저 보여줘\n- 코드는 항상 TypeScript로"}
                  rows={5}
                  style={{
                    ...inputStyle(!!customInstructions),
                    resize: 'vertical',
                    fontFamily: 'inherit',
                    lineHeight: 1.6,
                    minHeight: 100,
                  }}
                />
                <span style={{ fontSize: 11, color: 'rgba(255,255,255,0.35)' }}>
                  여기에 입력한 내용은 모든 AI 답변에 자동으로 반영됩니다.
                </span>
              </div>
            )}

            {/* ── 이메일 탭 ── */}
            {tab === 'email' && (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                <label style={{ ...labelStyle, color: '#90cdf4' }}>
                  📧 PC 건강 리포트 수신 이메일
                </label>
                <input
                  type="email"
                  value={emailTo}
                  onChange={e => setEmailTo(e.target.value)}
                  placeholder="example@gmail.com"
                  style={inputStyle(!!emailTo)}
                />
                <span style={{ fontSize: 11, color: 'rgba(255,255,255,0.35)' }}>
                  "PC 리포트 이메일로 보내줘" 명령 시 이 주소로 발송됩니다.
                  이메일 전송에는 SMTP 설정이 필요합니다.
                </span>
                <div style={{ background: 'rgba(144,205,244,0.06)', borderRadius: 8, padding: '8px 10px', fontSize: 11, color: '#718096' }}>
                  💡 Gmail 사용 시: 구글 계정 &gt; 보안 &gt; 앱 비밀번호에서 SMTP 비밀번호 발급
                </div>
              </div>
            )}

            {/* ── 정보 탭 ── */}
            {tab === 'about' && (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
                <div style={{ textAlign: 'center', padding: '8px 0' }}>
                  <div style={{ fontSize: 32 }}>🤖</div>
                  <div style={{ fontSize: 16, fontWeight: 800, color: '#e2e8f0', marginTop: 4 }}>Nexus AI 비서</div>
                  <div style={{ fontSize: 12, color: '#718096' }}>v2.5.0 — Perplexity 엔진</div>
                </div>
                {[
                  ['AI 엔진', 'Perplexity (sonar-pro · 웹 검색 내장)'],
                  ['Vision', '미지원'],
                  ['로컬 AI', 'Ollama (선택)'],
                  ['백엔드', 'Go + Windows API'],
                  ['프론트엔드', 'React + Framer Motion'],
                  ['배포', 'Tauri (.exe)'],
                ].map(([k, v]) => (
                  <div key={k} style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12, padding: '4px 0', borderBottom: '1px solid rgba(255,255,255,0.05)' }}>
                    <span style={{ color: '#718096' }}>{k}</span>
                    <span style={{ color: '#e2e8f0' }}>{v}</span>
                  </div>
                ))}
                <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.2)', textAlign: 'center', marginTop: 8 }}>
                  Perplexity API · https://www.perplexity.ai/settings/api
                </div>
              </div>
            )}

            {/* 저장 버튼 */}
            <motion.button
              whileTap={{ scale: 0.96 }}
              onClick={save}
              style={{
                padding: '11px',
                borderRadius: 12,
                border: 'none',
                background: saved
                  ? 'linear-gradient(135deg,#34d399,#10b981)'
                  : `linear-gradient(135deg,${primaryColor},${primaryColor}cc)`,
                color: '#fff',
                fontSize: 14,
                fontWeight: 800,
                cursor: 'pointer',
                boxShadow: `0 4px 16px ${primaryColor}44`,
                transition: 'background 0.3s',
              }}
            >
              {saved ? '✓ 저장됨!' : '💾 저장하기'}
            </motion.button>

            <p style={{ fontSize: 10, color: 'rgba(255,255,255,0.2)', textAlign: 'center', margin: 0 }}>
              키는 이 기기에만 저장되며 외부 서버로 전송되지 않습니다
            </p>
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  )
}
