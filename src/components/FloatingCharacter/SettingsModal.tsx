import { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { useAppStore } from '../../stores/appStore'
import { openCheckout, openBillingPortal } from '../../lib/paddle'

// 미리 정의된 테마 컬러 팔레트
const THEME_COLORS = [
  { name: '💜 기본 퍼플', value: '#9b59b6' },
  { name: '🔵 오션 블루', value: '#3b82f6' },
  { name: '🟢 민트 그린', value: '#10b981' },
  { name: '🟠 선셋 오렌지', value: '#f97316' },
  { name: '🔴 루비 레드', value: '#ef4444' },
  { name: '🌸 벚꽃 핑크', value: '#ec4899' },
  { name: '💛 골든 옐로우', value: '#eab308' },
  { name: '🩵 스카이 시안', value: '#06b6d4' },
]

interface SettingsModalProps {
  open: boolean
  onClose: () => void
  primaryColor: string
  onPrimaryColorChange?: (color: string) => void
}

export function SettingsModal({ open, onClose, primaryColor, onPrimaryColorChange }: SettingsModalProps) {
  const { micEnabled, setMicEnabled, userEmail, subscriptionStatus, subscriptionExpiry, setLoggedOut, userLang, setUserLang } = useAppStore()
  const [clarifyAutoMic, setClarifyAutoMic] = useState(localStorage.getItem('nexus-clarify-auto-mic') !== 'false')
  const [themeColor, setThemeColor] = useState(localStorage.getItem('nexus-theme-color') ?? primaryColor)
  const [customColor, setCustomColor] = useState(localStorage.getItem('nexus-theme-color') ?? primaryColor)
  const [claudeKey,   setClaudeKey]   = useState(localStorage.getItem('nexus-claude-key') ?? '')
  const [openaiKey,   setOpenaiKey]   = useState(localStorage.getItem('nexus-openai-key') ?? '')
  const [ollamaUrl,   setOllamaUrl]   = useState(localStorage.getItem('nexus-ollama-url') ?? 'http://localhost:11434')
  const [emailTo,             setEmailTo]             = useState(localStorage.getItem('nexus-report-email') ?? '')
  const [customInstructions,  setCustomInstructions]  = useState(localStorage.getItem('nexus-custom-instructions') ?? '')
  const [saved,               setSaved]               = useState(false)
  const [tab, setTab] = useState<'account' | 'ai' | 'email' | 'about'>('account')

  const isEn = userLang === 'en'

  const subLabel = {
    active:  { text: isEn ? 'Active'          : '구독 중',          color: '#4ade80' },
    trial:   { text: isEn ? '3-Day Free Trial' : '3일 무료 체험 중', color: '#facc15' },
    expired: { text: isEn ? 'Expired'          : '구독 만료',        color: '#f87171' },
    none:    { text: isEn ? 'Not subscribed'   : '미가입',           color: '#718096' },
  }[subscriptionStatus]

  const expiryFormatted = subscriptionExpiry
    ? new Date(subscriptionExpiry).toLocaleDateString(isEn ? 'en-US' : 'ko-KR', { year: 'numeric', month: 'long', day: 'numeric' })
    : ''

  const applyThemeColor = (color: string) => {
    setThemeColor(color)
    setCustomColor(color)
    localStorage.setItem('nexus-theme-color', color)
    onPrimaryColorChange?.(color)
  }

  const save = () => {
    // 테마 색상 저장
    localStorage.setItem('nexus-theme-color', themeColor)
    onPrimaryColorChange?.(themeColor)

    if (claudeKey.trim()) localStorage.setItem('nexus-claude-key', claudeKey.trim())
    else                  localStorage.removeItem('nexus-claude-key')

    if (openaiKey.trim()) localStorage.setItem('nexus-openai-key', openaiKey.trim())
    else                  localStorage.removeItem('nexus-openai-key')

    localStorage.setItem('nexus-ollama-url', ollamaUrl.trim() || 'http://localhost:11434')

    if (emailTo.trim()) localStorage.setItem('nexus-report-email', emailTo.trim())

    if (customInstructions.trim()) localStorage.setItem('nexus-custom-instructions', customInstructions.trim())
    else localStorage.removeItem('nexus-custom-instructions')

    // Claude 키를 백엔드에 전송 (sk-ant- 키는 즉시 1순위 LLM으로 활성화)
    fetch('http://127.0.0.1:17891/api/llm/config', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        claude_key: claudeKey.trim() || undefined,
        ollama_url: ollamaUrl.trim() || undefined,
      }),
    }).catch(() => {})

    setSaved(true)
    setTimeout(() => { setSaved(false); onClose() }, 1400)
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

  const assistantName = localStorage.getItem('nexus-assistant-name') ?? 'Nexus'

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
                  {isEn ? 'Settings' : 'Nexus 설정'}
                </span>
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                {/* 언어 토글 */}
                <div style={{ display: 'flex', gap: 3, background: 'rgba(255,255,255,0.06)', borderRadius: 8, padding: 3 }}>
                  {(['ko', 'en'] as const).map(l => (
                    <button key={l} onClick={() => setUserLang(l)} style={{
                      padding: '2px 8px', borderRadius: 6, border: 'none', cursor: 'pointer', fontSize: 11, fontWeight: 700,
                      background: userLang === l ? `${primaryColor}55` : 'transparent',
                      color: userLang === l ? 'white' : 'rgba(255,255,255,0.35)',
                      transition: 'all 0.15s',
                    }}>
                      {l === 'ko' ? '🇰🇷 KO' : '🇺🇸 EN'}
                    </button>
                  ))}
                </div>
                <button onClick={onClose} style={{
                  width: 28, height: 28, borderRadius: '50%', border: 'none',
                  background: 'rgba(255,255,255,0.08)', color: 'rgba(255,255,255,0.5)',
                  fontSize: 14, cursor: 'pointer',
                }}>✕</button>
              </div>
            </div>

            {/* 탭 */}
            <div style={{ display: 'flex', gap: 4, background: 'rgba(255,255,255,0.04)', borderRadius: 10, padding: 4 }}>
              {(isEn
                ? [['account', '👤 Account'], ['ai', '🤖 AI'], ['email', '📧 Email'], ['about', 'ℹ️ About']] as const
                : [['account', '👤 계정'], ['ai', '🤖 AI 설정'], ['email', '📧 이메일'], ['about', 'ℹ️ 정보']] as const
              ).map(([key, label]) => (
                <button key={key} onClick={() => setTab(key as typeof tab)} style={tabStyle(tab === key)}>{label}</button>
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
                      {userEmail || (isEn ? 'Login required' : '로그인 필요')}
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
                    {isEn
                      ? `⚠️ Subscription ${subscriptionStatus === 'expired' ? 'has expired' : 'not active'}. Some AI features are restricted.`
                      : `⚠️ 구독이 ${subscriptionStatus === 'expired' ? '만료되었습니다' : '없습니다'}. 일부 AI 기능이 제한됩니다.`}
                  </div>
                )}

                {/* 구독하기 버튼 */}
                {(subscriptionStatus === 'expired' || subscriptionStatus === 'none' || subscriptionStatus === 'trial') && (
                  <button
                    onClick={() => openCheckout(userEmail, localStorage.getItem('nexus-user-id') ?? undefined)}
                    style={{
                      padding: '12px', borderRadius: 10, border: 'none', cursor: 'pointer',
                      background: `linear-gradient(135deg,${primaryColor},${primaryColor}cc)`,
                      color: 'white', fontSize: 14, fontWeight: 800,
                    }}
                  >
                    {isEn
                      ? `💳 ${subscriptionStatus === 'trial' ? 'Subscribe Now' : 'Subscribe'} — $19/mo`
                      : `💳 ${subscriptionStatus === 'trial' ? '지금 구독하기' : '구독하기'} — ₩14,900/월`}
                  </button>
                )}

                {/* 구독 관리 */}
                {subscriptionStatus === 'active' && (
                  <button
                    onClick={() => openBillingPortal(userEmail)}
                    style={{
                      padding: '10px', borderRadius: 10, border: '1px solid rgba(255,255,255,0.12)',
                      background: 'transparent', color: 'rgba(255,255,255,0.6)', fontSize: 13, cursor: 'pointer',
                    }}
                  >
                    {isEn ? 'Manage Subscription (change payment · cancel)' : '구독 관리 (결제수단 변경 · 해지)'}
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
                    {isEn ? 'Log Out' : '로그아웃'}
                  </button>
                )}

                {/* 🎨 테마 색상 */}
                <div style={{ background: 'rgba(255,255,255,0.04)', borderRadius: 10, padding: '14px 16px', display: 'flex', flexDirection: 'column', gap: 10 }}>
                  <div style={{ fontSize: 12, fontWeight: 700, color: 'rgba(255,255,255,0.7)' }}>
                    {isEn ? '🎨 Theme Color' : '🎨 테마 색상'}
                  </div>
                  {/* 팔레트 스와치 */}
                  <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
                    {THEME_COLORS.map(t => (
                      <button
                        key={t.value}
                        title={t.name}
                        onClick={() => applyThemeColor(t.value)}
                        style={{
                          width: 28, height: 28, borderRadius: '50%', border: 'none', cursor: 'pointer',
                          background: t.value,
                          outline: themeColor === t.value ? `3px solid ${t.value}` : '3px solid transparent',
                          outlineOffset: 2,
                          boxShadow: themeColor === t.value ? `0 0 8px ${t.value}88` : 'none',
                          transition: 'all 0.15s',
                          transform: themeColor === t.value ? 'scale(1.15)' : 'scale(1)',
                        }}
                      />
                    ))}
                    {/* 직접 입력 */}
                    <div style={{ position: 'relative', width: 28, height: 28 }}>
                      <input
                        type="color"
                        value={customColor}
                        onChange={e => {
                          setCustomColor(e.target.value)
                          applyThemeColor(e.target.value)
                        }}
                        title={isEn ? 'Custom color' : '직접 색상 선택'}
                        style={{
                          width: 28, height: 28, borderRadius: '50%',
                          border: 'none', cursor: 'pointer', padding: 0,
                          background: 'transparent', opacity: 0,
                          position: 'absolute', top: 0, left: 0,
                        }}
                      />
                      <div style={{
                        width: 28, height: 28, borderRadius: '50%',
                        background: `conic-gradient(red, yellow, lime, cyan, blue, magenta, red)`,
                        pointerEvents: 'none',
                        border: '2px solid rgba(255,255,255,0.3)',
                        boxSizing: 'border-box',
                      }} />
                    </div>
                  </div>
                  <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.3)' }}>
                    {isEn ? 'Current: ' : '현재: '}
                    <span style={{ color: themeColor, fontWeight: 700 }}>{themeColor}</span>
                    <span style={{ marginLeft: 4 }}>— {isEn ? 'changes apply immediately' : '즉시 적용됨'}</span>
                  </div>
                </div>

                {/* 혜택 안내 */}
                <div style={{ background: 'rgba(255,255,255,0.02)', borderRadius: 10, padding: '12px 14px', fontSize: 11, color: 'rgba(255,255,255,0.4)', lineHeight: 1.9 }}>
                  <div style={{ color: 'rgba(255,255,255,0.6)', fontWeight: 700, marginBottom: 4 }}>
                    {isEn ? 'Subscription Benefits' : '구독 혜택'}
                  </div>
                  <div>{isEn ? '✦ 2,000 AI requests/day (all features)' : '✦ 하루 2,000건 AI 요청 (모든 기능)'}</div>
                  <div>{isEn ? '✦ Real-time web search (Perplexity)' : '✦ 실시간 웹 검색 (Perplexity)'}</div>
                  <div>{isEn ? '✦ Screen analysis & translation' : '✦ 화면 분석 · 번역'}</div>
                  <div>{isEn ? '✦ Automatic updates' : '✦ 자동 업데이트'}</div>
                  <div>{isEn ? '✦ Cancel anytime' : '✦ 언제든 해지 가능'}</div>
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
                    <div style={{ fontSize: 13, fontWeight: 600, color: 'white' }}>
                      {isEn ? '🎙️ Voice Recognition (Wake Word)' : '🎙️ 음성 인식 (웨이크워드)'}
                    </div>
                    <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.4)', marginTop: 2 }}>
                      {micEnabled
                        ? (isEn ? `Say "${assistantName}" to activate` : `"${assistantName}" 라고 부르면 활성화`)
                        : (isEn ? 'Disabled — button only' : '비활성화 — 버튼으로만 사용')}
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
                    <div style={{ fontSize: 13, fontWeight: 600, color: 'white' }}>
                      {isEn ? '🎤 Auto-mic after AI question' : '🎤 질문 후 마이크 자동 시작'}
                    </div>
                    <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.4)', marginTop: 2 }}>
                      {isEn ? 'Automatically open mic when AI asks back' : 'AI가 되물을 때 자동으로 마이크 켜기'}
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

                {/* AI 서버 안내 */}
                <div style={{
                  background: 'rgba(72,187,120,0.08)', border: '1px solid rgba(72,187,120,0.2)',
                  borderRadius: 10, padding: '10px 14px', fontSize: 11, color: '#68d391',
                }}>
                  {isEn ? '✅ AI features are provided automatically via Nexus server.' : '✅ AI 기능은 Nexus 서버에서 자동으로 제공됩니다.'}<br/>
                  <span style={{ color: '#718096', marginTop: 4, display: 'block' }}>
                    {isEn
                      ? 'No API key needed — all AI features activate with your subscription.'
                      : '별도 API 키 입력 없이 구독만으로 모든 AI 기능이 활성화됩니다.'}
                  </span>
                </div>

                {/* Claude API Key — 1순위 LLM */}
                <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <label style={{ ...labelStyle, color: '#f6ad55' }}>
                      {isEn ? '✦ CLAUDE API KEY (highest quality · priority #1)' : '✦ CLAUDE API KEY (최고 품질 · 1순위)'}
                    </label>
                    {claudeKey.trim().startsWith('sk-ant-') && (
                      <span style={{
                        fontSize: 10, fontWeight: 700, padding: '2px 7px', borderRadius: 99,
                        background: 'rgba(246,173,85,0.18)', color: '#f6ad55', border: '1px solid rgba(246,173,85,0.4)',
                      }}>
                        {isEn ? 'ACTIVE' : '활성'}
                      </span>
                    )}
                  </div>
                  <input
                    type="password"
                    value={claudeKey}
                    onChange={e => setClaudeKey(e.target.value)}
                    placeholder="sk-ant-api03-..."
                    style={inputStyle(claudeKey.trim().startsWith('sk-ant-'))}
                  />
                  <span style={{ fontSize: 10, color: 'rgba(255,255,255,0.3)' }}>
                    {isEn
                      ? 'claude-sonnet-4-6 is used when set — Claude-level accuracy'
                      : '설정 시 claude-sonnet-4-6 사용 — Claude 수준의 정확도'}
                  </span>
                </div>

                {/* Ollama URL */}
                <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                  <label style={{ ...labelStyle, color: '#68d391' }}>
                    {isEn ? '🦙 OLLAMA SERVER (local free LLM · priority #1)' : '🦙 OLLAMA 서버 (로컬 무료 LLM · 우선순위 1위)'}
                  </label>
                  <input
                    type="text"
                    value={ollamaUrl}
                    onChange={e => setOllamaUrl(e.target.value)}
                    placeholder="http://localhost:11434"
                    style={inputStyle(true)}
                  />
                  <span style={{ fontSize: 10, color: 'rgba(255,255,255,0.3)' }}>
                    {isEn ? 'Use offline AI at zero cost when Ollama is running' : 'Ollama 실행 중이면 비용 0원으로 오프라인 AI 사용 가능'}
                  </span>
                </div>

                {/* OpenAI TTS */}
                <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                  <label style={{ ...labelStyle, color: primaryColor }}>
                    {isEn ? '🎵 OPENAI API KEY (high-quality TTS · optional)' : '🎵 OPENAI API KEY (고품질 TTS · 선택)'}
                  </label>
                  <input
                    type="password"
                    value={openaiKey}
                    onChange={e => setOpenaiKey(e.target.value)}
                    placeholder="sk-proj-..."
                    style={inputStyle(!!openaiKey)}
                  />
                  <span style={{ fontSize: 10, color: 'rgba(255,255,255,0.3)' }}>
                    {isEn ? 'Falls back to browser TTS if not set (free)' : '없으면 브라우저 기본 TTS 사용 (무료)'}
                  </span>
                </div>

                {/* AI 우선순위 안내 */}
                <div style={{ background: 'rgba(255,255,255,0.03)', borderRadius: 10, padding: '10px 12px', fontSize: 11 }}>
                  <div style={{ color: '#a0aec0', marginBottom: 4, fontWeight: 600 }}>
                    {isEn ? '⚡ AI Response Priority' : '⚡ AI 응답 우선순위'}
                  </div>
                  {(isEn ? [
                    ['#1', 'Ollama Local',   'Fully free · offline',                    '#68d391'],
                    ['#2', 'Claude API',     'sk-ant- key · highest accuracy',          '#f6ad55'],
                    ['#3', 'Nexus Server',   'Subscription incl. · no key',             '#63b3ed'],
                    ['#4', 'Built-in Logic', 'Always works · no LLM needed',            '#90cdf4'],
                  ] : [
                    ['1위', 'Ollama 로컬',  '완전 무료 · 오프라인',                    '#68d391'],
                    ['2위', 'Claude API',   'sk-ant- 키 · 최고 정확도',               '#f6ad55'],
                    ['3위', 'Nexus 서버',   '구독 포함 · 키 불필요',                  '#63b3ed'],
                    ['4위', '내장 키워드',  '항상 동작 · LLM 불필요',                '#90cdf4'],
                  ]).map(([rank, name, desc, color]) => (
                    <div key={rank} style={{ display: 'flex', gap: 8, alignItems: 'center', padding: '3px 0' }}>
                      <span style={{ color, fontWeight: 700, minWidth: 28 }}>{rank}</span>
                      <span style={{ color: claudeKey.trim().startsWith('sk-ant-') && name.includes('Claude') ? '#f6ad55' : '#e2e8f0', fontWeight: claudeKey.trim().startsWith('sk-ant-') && name.includes('Claude') ? 700 : 400 }}>{name}</span>
                      <span style={{ color: '#718096', fontSize: 10 }}>{desc}</span>
                      {claudeKey.trim().startsWith('sk-ant-') && name.includes('Claude') && (
                        <span style={{ fontSize: 9, color: '#f6ad55', fontWeight: 700 }}>●</span>
                      )}
                    </div>
                  ))}
                </div>
              </>
            )}

            {/* ── AI 설정 탭 — Custom Instructions ── */}
            {tab === 'ai' && (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                <label style={{ ...labelStyle, color: '#b794f4' }}>
                  {isEn ? '🧠 Custom Instructions (personalize AI style)' : '🧠 Custom Instructions (나만의 AI 스타일 설정)'}
                </label>
                <textarea
                  value={customInstructions}
                  onChange={e => setCustomInstructions(e.target.value)}
                  placeholder={isEn
                    ? 'Examples:\n- Always use bullet points\n- Keep answers under 3 lines\n- Show ingredients table first for recipes\n- Always write code in TypeScript'
                    : '예시:\n- 항상 bullet point로 정리해줘\n- 답변은 3줄 이내로\n- 요리 레시피는 재료표 먼저 보여줘\n- 코드는 항상 TypeScript로'}
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
                  {isEn
                    ? 'These instructions are applied automatically to all AI responses.'
                    : '여기에 입력한 내용은 모든 AI 답변에 자동으로 반영됩니다.'}
                </span>
              </div>
            )}

            {/* ── 이메일 탭 ── */}
            {tab === 'email' && (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                <label style={{ ...labelStyle, color: '#90cdf4' }}>
                  {isEn ? '📧 PC Health Report Email' : '📧 PC 건강 리포트 수신 이메일'}
                </label>
                <input
                  type="email"
                  value={emailTo}
                  onChange={e => setEmailTo(e.target.value)}
                  placeholder="example@gmail.com"
                  style={inputStyle(!!emailTo)}
                />
                <span style={{ fontSize: 11, color: 'rgba(255,255,255,0.35)' }}>
                  {isEn
                    ? '"Send PC report by email" command will deliver to this address. SMTP setup required.'
                    : '"PC 리포트 이메일로 보내줘" 명령 시 이 주소로 발송됩니다. 이메일 전송에는 SMTP 설정이 필요합니다.'}
                </span>
                <div style={{ background: 'rgba(144,205,244,0.06)', borderRadius: 8, padding: '8px 10px', fontSize: 11, color: '#718096' }}>
                  {isEn
                    ? '💡 Gmail: Google Account > Security > App Passwords to get SMTP password'
                    : '💡 Gmail 사용 시: 구글 계정 > 보안 > 앱 비밀번호에서 SMTP 비밀번호 발급'}
                </div>
              </div>
            )}

            {/* ── 정보 탭 ── */}
            {tab === 'about' && (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
                <div style={{ textAlign: 'center', padding: '8px 0' }}>
                  <div style={{ fontSize: 32 }}>🤖</div>
                  <div style={{ fontSize: 16, fontWeight: 800, color: '#e2e8f0', marginTop: 4 }}>
                    {isEn ? 'Nexus AI Assistant' : 'Nexus AI 비서'}
                  </div>
                  <div style={{ fontSize: 12, color: claudeKey.trim().startsWith('sk-ant-') ? '#f6ad55' : '#718096' }}>
                    v2.5.0 — {claudeKey.trim().startsWith('sk-ant-') ? 'Claude Sonnet 4.6' : 'Perplexity'} {isEn ? 'engine' : '엔진'}
                  </div>
                </div>
                {(isEn ? [
                  ['AI Engine',  claudeKey.trim().startsWith('sk-ant-') ? 'Claude Sonnet 4.6 (Anthropic · highest accuracy)' : 'Perplexity (sonar-pro · web search built-in)'],
                  ['Vision',     'Not supported'],
                  ['Local AI',   'Ollama (optional)'],
                  ['Backend',    'Go + Windows API'],
                  ['Frontend',   'React + Framer Motion'],
                  ['Packaging',  'Tauri (.exe)'],
                ] : [
                  ['AI 엔진',    claudeKey.trim().startsWith('sk-ant-') ? 'Claude Sonnet 4.6 (Anthropic · 최고 정확도)' : 'Perplexity (sonar-pro · 웹 검색 내장)'],
                  ['Vision',     '미지원'],
                  ['로컬 AI',    'Ollama (선택)'],
                  ['백엔드',     'Go + Windows API'],
                  ['프론트엔드', 'React + Framer Motion'],
                  ['배포',       'Tauri (.exe)'],
                ]).map(([k, v]) => (
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
              {saved ? (isEn ? '✓ Saved!' : '✓ 저장됨!') : (isEn ? '💾 Save' : '💾 저장하기')}
            </motion.button>

            <p style={{ fontSize: 10, color: 'rgba(255,255,255,0.2)', textAlign: 'center', margin: 0 }}>
              {isEn
                ? 'Keys are stored on this device only and never sent to external servers.'
                : '키는 이 기기에만 저장되며 외부 서버로 전송되지 않습니다'}
            </p>
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  )
}
