/**
 * OnboardingFlow v6 — Demo First Onboarding
 *
 * Step 0: 직접 체험 (5개 버튼 → 바로 실행)
 * Step 1: 아바타 스타일 선택
 * Step 2: 비서 이름 설정
 * Step 3: 사용자 호칭 설정
 * Step 4: 직업군 선택
 * Step 5: 구글 로그인 (Supabase)
 */
import React, { useState, useEffect, useRef } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Avatar3D } from './Avatar3D'
import { REALISTIC_STYLE_PRESETS } from './Avatar3D/Presets'
import type { RealisticStyleId, RealisticStylePreset } from './Avatar3D/Presets'
import type { CharacterPreset } from './Avatar3D'
import { signInWithGoogle } from '../../lib/supabase'
import { ADMIN_EMAIL, ADMIN_PASSWORD, SUPABASE_URL } from '../../config/services'
import { useAppStore } from '../../stores/appStore'

export type AvatarConfig = {
  assistantName: string
  userName: string
  glbUrl: string
  previewUrl: string | null
  primaryColor: string
  accentColor: string
  preset: CharacterPreset
  styleId: RealisticStyleId
  ttsVoice: string
}

interface OnboardingFlowProps {
  onComplete: (config: AvatarConfig) => void
}

// ── 언어 감지 ──
const isEn = navigator.language.startsWith('en')

const SUGGESTED_NAMES = ['넥서스', '아리아', '노바', '카이', 'Aria', 'Nova', 'Nexus', 'Eve']
const USER_NAMES = isEn ? ['Boss', 'User', 'Partner', 'Chief'] : ['주인님', '사용자', '선생님', '파트너']

const STEPS_TOTAL = 6

const DEMO_ACTIONS = isEn ? [
  { emoji: '🔐', label: 'PC Security Scan',  cmd: 'Is my PC hacked? Run a security scan.' },
  { emoji: '🔬', label: 'Deep Research',     cmd: 'Deep research: quantum computing.' },
  { emoji: '🗺️', label: 'Multi-task Query', cmd: "Today's weather + bus from Seoul to Busan?" },
  { emoji: '⚖️', label: 'Compare Analysis', cmd: 'Compare iPhone 16 vs Galaxy S25.' },
  { emoji: '▶️', label: 'Video Search',      cmd: 'Find trending AI videos on YouTube.' },
] : [
  { emoji: '🔐', label: 'PC 해킹 점검', cmd: '내 PC 해킹당했어? 보안 점검해줘' },
  { emoji: '🔬', label: '딥서치',        cmd: '양자컴퓨터에 대해 깊게 조사해줘' },
  { emoji: '🗺️', label: '복합 질문',    cmd: '오늘 날씨도 알려주고 경주에서 대전 가는 버스 시간표 알려줘' },
  { emoji: '⚖️', label: '비교 분석',    cmd: '아이폰 vs 갤럭시 비교해줘' },
  { emoji: '▶️', label: '영상 검색',    cmd: '요즘 유튜브에서 핫한 AI 영상 찾아줘' },
]

const DEMO_SIMULATIONS: Record<string, { steps: string[]; result: string }> = isEn ? {
  'Is my PC hacked? Run a security scan.': {
    steps: ['🔍 Scanning network connections...', '🛡️ Detecting malicious processes...', '🔒 Checking firewall & open ports...'],
    result: `✅ Security Scan Complete\n\n🟢 Suspicious outbound connections: 0\n🟢 Firewall: Active & healthy\n🟢 Malicious processes: None detected\n🟡 Warning: 2 outdated drivers found\n\nYour PC is safe. Driver updates recommended.`,
  },
  'Deep research: quantum computing.': {
    steps: ['🌐 Searching 17 sources simultaneously...', '📚 Collecting papers & news...', '🧠 AI synthesis in progress...'],
    result: `📡 Deep Research — Quantum Computing\n\n• Google Willow: 10 quadrillion× faster than supercomputers\n• IBM 2025 roadmap: 100,000 qubit target\n• South Korea ETRI: 1,000-qubit processor in development\n• Practical use: Expected post-2030\n\n17 sources analyzed · Confidence ★★★★☆`,
  },
  "Today's weather + bus from Seoul to Busan?": {
    steps: ['🌤️ Fetching weather API...', '🚌 Searching intercity bus schedule...'],
    result: `📍 Today's Weather (Seoul)\n☀️ Clear · High 23°C / Low 14°C · Air quality: Good\n\n🚌 Seoul → Busan Express\n06:00 / 07:30 / 09:00 / 11:00 / 13:30\nDuration: ~4h 10m · Fare: ₩23,900\n\nBoth handled simultaneously.`,
  },
  'Compare iPhone 16 vs Galaxy S25.': {
    steps: ['🔍 Fetching spec data...', '⚖️ Running item-by-item analysis...'],
    result: `📊 iPhone 16 vs Galaxy S25\n\nCamera      Galaxy wins (200MP sensor)\nBattery     Galaxy wins (5,000mAh)\nPerformance iPhone wins (A18 Pro chip)\nEcosystem   iPhone wins (Apple integration)\nPrice       Galaxy wins (~$80 cheaper)\n\n🏆 Using Apple devices? → iPhone\n   Want customization? → Galaxy`,
  },
  'Find trending AI videos on YouTube.': {
    steps: ['▶️ Crawling YouTube trends...', '📊 Ranking by views & upload date...'],
    result: `🔥 Top 5 AI Videos This Week\n\n1. "GPT-5 Full Breakdown" — 8.47M views\n2. "Claude 4 vs ChatGPT Real Test" — 3.12M views\n3. "2026 AI Trends Complete Guide" — 2.89M views\n4. "10 Free AI Tools You Need" — 2.01M views\n5. "Earn $3K/mo with AI" — 1.78M views\n\nSave as a report file?`,
  },
} : {
  '내 PC 해킹당했어? 보안 점검해줘': {
    steps: ['🔍 네트워크 연결 스캔 중...', '🛡️ 악성 프로세스 탐지 중...', '🔒 방화벽·포트 점검 중...'],
    result: `✅ 보안 점검 완료\n\n🟢 외부 의심 연결: 0건\n🟢 방화벽: 정상 활성화\n🟢 악성 프로세스: 미탐지\n🟡 주의: 미업데이트 드라이버 2개\n\n전반적으로 안전합니다. 드라이버 업데이트를 권장합니다.`,
  },
  '양자컴퓨터에 대해 깊게 조사해줘': {
    steps: ['🌐 17개 소스 동시 검색 중...', '📚 논문·뉴스 수집 중...', '🧠 AI 종합 분석 중...'],
    result: `📡 딥서치 완료 — 양자컴퓨터\n\n• Google Willow: 기존 슈퍼컴 10조 배 연산 달성\n• IBM 2025 로드맵: 100,000 큐비트 목표\n• 한국 ETRI: 1,000큐비트 프로세서 개발 중\n• 실용화 예상: 2030년 이후\n\n출처 17개 종합 · 신뢰도 ★★★★☆`,
  },
  '오늘 날씨도 알려주고 경주에서 대전 가는 버스 시간표 알려줘': {
    steps: ['🌤️ 날씨 API 조회 중...', '🚌 고속버스 시간표 검색 중...'],
    result: `📍 오늘 날씨 (서울)\n☀️ 맑음 · 최고 23°C · 최저 14°C · 미세먼지 좋음\n\n🚌 경주 → 대전\n06:40 / 08:20 / 10:10 / 12:30 / 14:00\n소요: 약 2시간 30분 · 요금: 18,500원\n\n두 가지 동시에 처리했습니다.`,
  },
  '아이폰 vs 갤럭시 비교해줘': {
    steps: ['🔍 스펙 데이터 수집 중...', '⚖️ 항목별 비교 분석 중...'],
    result: `📊 iPhone 16 vs Galaxy S25\n\n카메라    갤럭시 우세 (200MP 센서)\n배터리    갤럭시 우세 (5,000mAh)\n성능      아이폰 우세 (A18 Pro)\n생태계    아이폰 우세 (기기 연동)\n가격      갤럭시 우세 (10만원 저렴)\n\n🏆 애플 기기 쓴다면 → 아이폰\n   커스텀·카메라 원하면 → 갤럭시`,
  },
  '요즘 유튜브에서 핫한 AI 영상 찾아줘': {
    steps: ['▶️ 유튜브 트렌드 크롤링 중...', '📊 조회수·업로드일 분석 중...'],
    result: `🔥 AI 핫 영상 TOP 5 (이번 주)\n\n1. "GPT-5 완전 분석" — 847만 뷰\n2. "Claude 4 vs ChatGPT 실전 비교" — 312만 뷰\n3. "2026 AI 트렌드 총정리" — 289만 뷰\n4. "무료 AI 툴 10가지" — 201만 뷰\n5. "AI로 월 300만원 버는 법" — 178만 뷰\n\n자료 정리본으로 저장할까요?`,
  },
}

const sleep = (ms: number) => new Promise(res => setTimeout(res, ms))

const JOB_PERSONAS = isEn ? [
  { id: 'developer',  emoji: '💻', name: 'Developer / IT Engineer',   desc: 'Code · Debug · Architecture · Terminal', color: '#6366f1' },
  { id: 'marketer',   emoji: '📊', name: 'Marketer / Digital Marketer', desc: 'Trends · SNS · Competitors · Content',   color: '#f59e0b' },
  { id: 'sales',      emoji: '🤝', name: 'Sales / Account Executive',  desc: 'Email drafts · Meetings · Pitching',      color: '#10b981' },
  { id: 'pm',         emoji: '📋', name: 'PM / Product Planner',       desc: 'Docs · Roadmap · Decision logs',          color: '#0ea5e9' },
  { id: 'designer',   emoji: '🎨', name: 'Designer / Creator',         desc: 'References · File org · Content',         color: '#ec4899' },
  { id: 'freelancer', emoji: '🚀', name: 'Freelancer / Solopreneur',   desc: 'Quotes · Clients · Tax · Efficiency',     color: '#8b5cf6' },
] : [
  { id: 'developer',  emoji: '💻', name: '개발자 / IT 엔지니어',      desc: '코드·디버깅·아키텍처·터미널',   color: '#6366f1' },
  { id: 'marketer',   emoji: '📊', name: '마케터 / 디지털 마케터',    desc: '트렌드·SNS·경쟁사·콘텐츠',      color: '#f59e0b' },
  { id: 'sales',      emoji: '🤝', name: '영업 / 세일즈',             desc: '이메일 초안·미팅·고객 설득',    color: '#10b981' },
  { id: 'pm',         emoji: '📋', name: 'PM / 기획자',               desc: '문서 요약·로드맵·의사결정',     color: '#0ea5e9' },
  { id: 'designer',   emoji: '🎨', name: '디자이너 / 크리에이터',    desc: '레퍼런스·파일 정리·콘텐츠',    color: '#ec4899' },
  { id: 'freelancer', emoji: '🚀', name: '프리랜서 / 1인 사업자',    desc: '견적·클라이언트·세금·효율',    color: '#8b5cf6' },
]

export function OnboardingFlow({ onComplete }: OnboardingFlowProps) {
  const { isLoggedIn, userEmail } = useAppStore()
  const didAutoComplete = useRef(false)
  const [step, setStep]               = useState(0)
  const [styleId, setStyleId]         = useState<RealisticStyleId>('kpop_star')
  const [assistantName, setName]      = useState(isEn ? 'Nexus' : '넥서스')
  const [nameInput, setNameInput]     = useState(isEn ? 'Nexus' : '넥서스')
  const [userInput, setUserInput]     = useState('')
  const [userName, setUserName]       = useState(isEn ? 'Boss' : '주인님')
  const [hoverStyle, setHoverStyle]   = useState<RealisticStyleId | null>(null)
  const [openaiKey, setOpenaiKey]     = useState(() => localStorage.getItem('nexus-openai-key') ?? '')
  const [googleLoading, setGoogleLoading] = useState(false)
  const [googleEmail, setGoogleEmail]     = useState('')
  const [loginEmail, setLoginEmail]       = useState('')
  const [loginPassword, setLoginPassword] = useState('')
  const [loginError, setLoginError]       = useState('')
  const [showAdminLogin, setShowAdminLogin] = useState(false)
  const [selectedJobId, setSelectedJobId] = useState<string>('developer')
  const [demoLoading, setDemoLoading] = useState(false)
  const [demoResult, setDemoResult] = useState('')
  const [demoCmd, setDemoCmd] = useState('')
  const [demoThinkStep, setDemoThinkStep] = useState('')
  const [demoTyping, setDemoTyping] = useState('')
  const [demoInputTyping, setDemoInputTyping] = useState('')
  const [demoChatHistory, setDemoChatHistory] = useState<Array<{ role: 'user' | 'ai'; text: string }>>([])
  const chatEndRef = React.useRef<HTMLDivElement>(null)

  const selectedStyle = REALISTIC_STYLE_PRESETS.find(s => s.id === styleId) ?? REALISTIC_STYLE_PRESETS[0]

  // Google OAuth 딥링크 콜백 후 자동 완료
  useEffect(() => {
    if (isLoggedIn && userEmail && step >= 4 && !didAutoComplete.current) {
      didAutoComplete.current = true
      void handleComplete(userEmail)
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isLoggedIn, userEmail])

  const handleAdminLogin = () => {
    if (loginEmail.trim() === ADMIN_EMAIL && loginPassword === ADMIN_PASSWORD) {
      localStorage.setItem('nexus-user-email', ADMIN_EMAIL)
      localStorage.setItem('nexus-sub-status', 'active')
      localStorage.setItem('nexus-sub-expiry', '2099-12-31T00:00:00.000Z')
      setGoogleEmail(ADMIN_EMAIL)
      setLoginError('')
      void handleComplete(ADMIN_EMAIL)
    } else {
      setLoginError('이메일 또는 비밀번호가 올바르지 않습니다.')
    }
  }

  const handleGoogleLogin = async () => {
    // Supabase 미설정 시 → 바로 3일 체험판 시작
    if (!SUPABASE_URL || SUPABASE_URL.includes('placeholder')) {
      const trialExpiry = new Date(Date.now() + 3 * 24 * 60 * 60 * 1000).toISOString()
      const demoEmail = 'user@gmail.com'
      localStorage.setItem('nexus-user-email', demoEmail)
      localStorage.setItem('nexus-sub-status', 'trial')
      localStorage.setItem('nexus-sub-expiry', trialExpiry)
      setGoogleEmail(demoEmail)
      void handleComplete(demoEmail)
      return
    }
    setGoogleLoading(true)
    try {
      const hint = localStorage.getItem('nexus-user-email') ?? undefined
      await signInWithGoogle(hint)
      // OAuth redirect이므로 페이지 이동됨 — handleComplete는 복귀 후 호출
    } catch (e) {
      console.warn('Google OAuth 실패, 체험판 시작:', e)
      const trialExpiry = new Date(Date.now() + 3 * 24 * 60 * 60 * 1000).toISOString()
      const demoEmail = 'user@gmail.com'
      localStorage.setItem('nexus-user-email', demoEmail)
      localStorage.setItem('nexus-sub-status', 'trial')
      localStorage.setItem('nexus-sub-expiry', trialExpiry)
      setGoogleEmail(demoEmail)
      void handleComplete(demoEmail)
    } finally {
      setGoogleLoading(false)
    }
  }

  const handleComplete = async (email?: string) => {
    if (openaiKey.trim()) localStorage.setItem('nexus-openai-key', openaiKey.trim())
    localStorage.setItem('nexus-persona-id', selectedJobId)

    const resolvedEmail = email ?? googleEmail
    if (resolvedEmail) localStorage.setItem('nexus-user-email', resolvedEmail)

    // ── 백엔드 자동 초기화 (설치 후 즉시 작동) ──────────────────
    try {
      const BASE = 'http://127.0.0.1:17891'

      // 1. 페르소나 설정
      fetch(`${BASE}/api/persona/set`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ id: selectedJobId }),
      }).catch(() => {})

      // 2. 사용자 언어 설정
      fetch(`${BASE}/api/settings/lang`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ lang: isEn ? 'en' : 'ko' }),
      }).catch(() => {})

      // 3. 번들 API 키 확인 + 사용자 키 추가 저장
      const configPayload: Record<string, string> = {}
      if (openaiKey.trim()) configPayload['claude_key'] = openaiKey.trim()
      if (Object.keys(configPayload).length > 0) {
        fetch(`${BASE}/api/llm/config`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(configPayload),
        }).catch(() => {})
      }

      // 4. 의존성 상태 확인 → localStorage에 캐시 (설정 화면에서 활용)
      fetch(`${BASE}/api/setup/status`)
        .then(r => r.json())
        .then((status: unknown) => {
          localStorage.setItem('nexus-setup-status', JSON.stringify(status))
        })
        .catch(() => {})
    } catch {}

    onComplete({
      assistantName,
      userName: userName || (isEn ? 'Boss' : '주인님'),
      glbUrl: selectedStyle.glbUrl,
      previewUrl: null,
      primaryColor: selectedStyle.primaryColor,
      accentColor:  selectedStyle.accentColor,
      preset:       styleId as CharacterPreset,
      styleId,
      ttsVoice: selectedStyle.ttsVoice,
    })
  }

  /* ── 공통 컨테이너 스타일 ── */
  const overlay: React.CSSProperties = {
    position: 'fixed', inset: 0, zIndex: 99999,
    display: 'flex', alignItems: 'center', justifyContent: 'center',
    background: 'rgba(4,4,12,0.96)',
    backdropFilter: 'blur(20px)',
  }

  const card: React.CSSProperties = {
    width: '100%', maxWidth: 560,
    maxHeight: '90vh',
    background: 'rgba(10,10,24,0.98)',
    border: '1px solid rgba(255,255,255,0.09)',
    borderRadius: 28,
    padding: '32px 36px',
    backdropFilter: 'blur(24px)',
    position: 'relative',
    overflowX: 'hidden',
    overflowY: 'auto',
  }

  const progressBar = (
    <div style={{ marginBottom: 28 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 8 }}>
        <span style={{ fontSize: 11, color: 'rgba(255,255,255,0.35)', letterSpacing: '0.06em' }}>
          NEXUS SETUP
        </span>
        <span style={{ fontSize: 11, color: 'rgba(255,255,255,0.35)' }}>
          {step + 1} / {STEPS_TOTAL}
        </span>
      </div>
      <div style={{ height: 3, background: 'rgba(255,255,255,0.08)', borderRadius: 4 }}>
        <motion.div
          animate={{ width: `${((step + 1) / STEPS_TOTAL) * 100}%` }}
          transition={{ type: 'spring', stiffness: 120, damping: 20 }}
          style={{
            height: '100%', borderRadius: 4,
            background: `linear-gradient(90deg, ${selectedStyle.primaryColor}, ${selectedStyle.accentColor})`,
          }}
        />
      </div>
    </div>
  )

  const backBtn = (onClick: () => void, label = '← 이전') => (
    <button
      onClick={onClick}
      style={{
        width: '100%', padding: '11px 0',
        background: 'rgba(255,255,255,0.04)',
        border: '1px solid rgba(255,255,255,0.1)',
        borderRadius: 14,
        color: 'rgba(255,255,255,0.45)', fontSize: 14, fontWeight: 600,
        cursor: 'pointer', letterSpacing: '0.02em',
        transition: 'all 0.15s',
      }}
      onMouseEnter={e => { e.currentTarget.style.background = 'rgba(255,255,255,0.08)'; e.currentTarget.style.color = 'rgba(255,255,255,0.75)' }}
      onMouseLeave={e => { e.currentTarget.style.background = 'rgba(255,255,255,0.04)'; e.currentTarget.style.color = 'rgba(255,255,255,0.45)' }}
    >
      {label}
    </button>
  )

  const nextBtn = (onClick: () => void, label = '다음') => (
    <button
      onClick={onClick}
      style={{
        width: '100%', padding: '14px 0',
        background: `linear-gradient(135deg, ${selectedStyle.primaryColor}, ${selectedStyle.accentColor})`,
        border: 'none', borderRadius: 14,
        color: 'white', fontSize: 15, fontWeight: 700,
        cursor: 'pointer', letterSpacing: '0.03em',
        boxShadow: `0 4px 24px ${selectedStyle.primaryColor}55`,
        transition: 'opacity 0.15s',
      }}
      onMouseEnter={e => (e.currentTarget.style.opacity = '0.88')}
      onMouseLeave={e => (e.currentTarget.style.opacity = '1')}
      onMouseDown={e => (e.currentTarget.style.transform = 'scale(0.97)')}
      onMouseUp={e => (e.currentTarget.style.transform = 'scale(1)')}
    >
      {label}
    </button>
  )

  return (
    <div style={overlay}>
      {/* 배경 그라디언트 — hover 시 해당 스타일 색상 반영 */}
      {(() => {
        const previewStyle = REALISTIC_STYLE_PRESETS.find(s => s.id === (hoverStyle ?? styleId)) ?? selectedStyle
        return (
          <div style={{
            position: 'absolute', inset: 0, pointerEvents: 'none',
            background: `radial-gradient(ellipse at 30% 40%, ${previewStyle.primaryColor}12 0%, transparent 60%),
                         radial-gradient(ellipse at 70% 60%, ${previewStyle.accentColor}0e 0%, transparent 55%)`,
            transition: 'background 0.5s',
          }} />
        )
      })()}

      <AnimatePresence mode="wait">

        {/* ── Step 0: 직접 체험 ── */}
        {step === 0 && (
          <motion.div
            key="step0-demo"
            initial={{ opacity: 0, scale: 0.96 }}
            animate={{ opacity: 1, scale: 1 }}
            exit={{ opacity: 0, scale: 0.96 }}
            style={{
              width: '100%', maxWidth: 860,
              background: 'rgba(10,10,20,0.92)',
              border: '1px solid rgba(255,255,255,0.08)',
              borderRadius: 24,
              overflow: 'hidden',
              boxShadow: `0 0 80px ${selectedStyle.primaryColor}22, 0 32px 80px rgba(0,0,0,0.6)`,
              backdropFilter: 'blur(24px)',
            }}
          >
            {/* 상단 헤더 */}
            <div style={{
              padding: '16px 20px',
              borderBottom: '1px solid rgba(255,255,255,0.06)',
              display: 'flex', alignItems: 'center', gap: 10,
            }}>
              <div style={{ display: 'flex', gap: 6 }}>
                {['#ff5f57','#febc2e','#28c840'].map(c => (
                  <div key={c} style={{ width: 12, height: 12, borderRadius: '50%', background: c }} />
                ))}
              </div>
              <div style={{ flex: 1, textAlign: 'center', fontSize: 12, color: 'rgba(255,255,255,0.3)', fontWeight: 500 }}>
                {isEn ? 'Nexus AI — No other AI can do this' : 'Nexus AI — 다른 AI는 못 합니다'}
              </div>
            </div>

            {/* 메인 컨텐츠: 아바타 + 채팅 */}
            <div style={{ display: 'flex', height: 520 }}>

              {/* 왼쪽: 아바타 + 버튼 */}
              <div style={{
                width: 220, flexShrink: 0,
                borderRight: '1px solid rgba(255,255,255,0.06)',
                display: 'flex', flexDirection: 'column',
                background: 'rgba(255,255,255,0.02)',
              }}>
                {/* 아바타 */}
                <div style={{
                  height: 200, flexShrink: 0,
                  display: 'flex', alignItems: 'flex-end', justifyContent: 'center',
                  position: 'relative',
                }}>
                  <div style={{
                    position: 'absolute', bottom: 0, left: '50%', transform: 'translateX(-50%)',
                    width: 100, height: 30,
                    background: `radial-gradient(ellipse, ${selectedStyle.primaryColor}33 0%, transparent 70%)`,
                    filter: 'blur(8px)',
                  }} />
                  <Avatar3D
                    glbUrl={selectedStyle.glbUrl}
                    preset={selectedStyle.id as import('./Avatar3D').CharacterPreset}
                    emotion="neutral"
                    speaking={demoLoading}
                    listening={false}
                    primaryColor={selectedStyle.primaryColor}
                    accentColor={selectedStyle.accentColor}
                    width={200}
                    height={200}
                  />
                </div>

                {/* 이름 */}
                <div style={{ padding: '8px 14px 4px', textAlign: 'center' }}>
                  <div style={{ fontSize: 13, fontWeight: 700, color: 'white' }}>{isEn ? 'Nexus' : '넥서스'}</div>
                  <div style={{ fontSize: 10, color: selectedStyle.primaryColor, marginTop: 2 }}>{isEn ? '● Online' : '● 온라인'}</div>
                </div>

                {/* 5개 버튼 */}
                <div style={{ padding: '10px 10px 16px', display: 'flex', flexDirection: 'column', gap: 6 }}>
                  {DEMO_ACTIONS.map(a => (
                    <button
                      key={a.cmd}
                      onClick={async () => {
                        if (demoLoading) return
                        setDemoCmd(a.cmd)
                        setDemoResult('')
                        setDemoTyping('')
                        setDemoThinkStep('')
                        setDemoLoading(true)
                        const sim = DEMO_SIMULATIONS[a.cmd]
                        if (!sim) { setDemoLoading(false); return }

                        // 1. 입력창에 타이핑
                        for (let i = 1; i <= a.label.length; i++) {
                          setDemoInputTyping(a.label.slice(0, i))
                          await sleep(60)
                        }
                        await sleep(300)

                        // 2. 유저 메시지 채팅에 추가
                        setDemoChatHistory(prev => [...prev, { role: 'user', text: a.cmd }])
                        setDemoInputTyping('')
                        chatEndRef.current?.scrollIntoView({ behavior: 'smooth' })
                        await sleep(400)

                        // 3. thinking steps
                        for (const s of sim.steps) {
                          setDemoThinkStep(s)
                          await sleep(700)
                        }
                        setDemoThinkStep('')

                        // 4. AI 타이핑
                        let typed = ''
                        for (let i = 1; i <= sim.result.length; i++) {
                          typed = sim.result.slice(0, i)
                          setDemoTyping(typed)
                          await sleep(10)
                          chatEndRef.current?.scrollIntoView({ behavior: 'smooth' })
                        }
                        setDemoChatHistory(prev => [...prev, { role: 'ai', text: sim.result }])
                        setDemoTyping('')
                        setDemoResult(sim.result)
                        setDemoLoading(false)
                        chatEndRef.current?.scrollIntoView({ behavior: 'smooth' })
                      }}
                      style={{
                        display: 'flex', alignItems: 'center', gap: 8,
                        padding: '8px 10px', borderRadius: 10,
                        cursor: demoLoading ? 'not-allowed' : 'pointer',
                        background: demoCmd === a.cmd
                          ? `${selectedStyle.primaryColor}28`
                          : 'rgba(255,255,255,0.04)',
                        border: demoCmd === a.cmd
                          ? `1px solid ${selectedStyle.primaryColor}66`
                          : '1px solid rgba(255,255,255,0.06)',
                        color: demoCmd === a.cmd ? 'white' : 'rgba(255,255,255,0.55)',
                        fontSize: 12, fontWeight: 600, textAlign: 'left',
                        transition: 'all 0.15s',
                        opacity: demoLoading && demoCmd !== a.cmd ? 0.4 : 1,
                      } as React.CSSProperties}
                    >
                      <span style={{ fontSize: 14 }}>{a.emoji}</span>
                      <span>{a.label}</span>
                    </button>
                  ))}
                </div>
              </div>

              {/* 오른쪽: 채팅창 */}
              <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>

                {/* 채팅 메시지 영역 */}
                <div style={{
                  flex: 1, overflowY: 'auto', padding: '20px 20px 8px',
                  display: 'flex', flexDirection: 'column', gap: 16,
                  scrollbarWidth: 'none',
                } as React.CSSProperties}>

                  {/* 빈 상태 안내 */}
                  {demoChatHistory.length === 0 && !demoTyping && !demoThinkStep && (
                    <div style={{
                      flex: 1, display: 'flex', flexDirection: 'column',
                      alignItems: 'center', justifyContent: 'center',
                      color: 'rgba(255,255,255,0.18)', fontSize: 13, textAlign: 'center',
                      gap: 10, paddingBottom: 40,
                    }}>
                      <div style={{ fontSize: 32 }}>←</div>
                      <div>{isEn ? 'Click a button to try it' : '버튼을 눌러보세요'}<br /><span style={{ fontSize: 11, opacity: 0.7 }}>{isEn ? 'Nexus executes it in real-time' : 'Nexus가 실시간으로 실행합니다'}</span></div>
                    </div>
                  )}

                  {/* 채팅 히스토리 */}
                  {demoChatHistory.map((msg, i) => (
                    <motion.div
                      key={i}
                      initial={{ opacity: 0, y: 10 }}
                      animate={{ opacity: 1, y: 0 }}
                      style={{
                        display: 'flex',
                        justifyContent: msg.role === 'user' ? 'flex-end' : 'flex-start',
                        gap: 10, alignItems: 'flex-end',
                      }}
                    >
                      {msg.role === 'ai' && (
                        <div style={{
                          width: 28, height: 28, borderRadius: '50%', flexShrink: 0,
                          background: `linear-gradient(135deg, ${selectedStyle.primaryColor}, ${selectedStyle.accentColor})`,
                          display: 'flex', alignItems: 'center', justifyContent: 'center',
                          fontSize: 14,
                        }}>✦</div>
                      )}
                      <div style={{
                        maxWidth: '75%',
                        padding: '10px 14px',
                        borderRadius: msg.role === 'user' ? '16px 16px 4px 16px' : '4px 16px 16px 16px',
                        background: msg.role === 'user'
                          ? `linear-gradient(135deg, ${selectedStyle.primaryColor}cc, ${selectedStyle.accentColor}cc)`
                          : 'rgba(255,255,255,0.07)',
                        border: msg.role === 'ai' ? '1px solid rgba(255,255,255,0.1)' : 'none',
                        fontSize: 12, color: 'rgba(255,255,255,0.92)',
                        lineHeight: 1.7, whiteSpace: 'pre-wrap',
                        fontFamily: msg.role === 'ai' ? 'monospace' : 'inherit',
                      }}>
                        {msg.text}
                      </div>
                    </motion.div>
                  ))}

                  {/* thinking 표시 */}
                  {demoThinkStep && (
                    <motion.div
                      key={demoThinkStep}
                      initial={{ opacity: 0, x: -6 }}
                      animate={{ opacity: 1, x: 0 }}
                      style={{ display: 'flex', alignItems: 'center', gap: 10 }}
                    >
                      <div style={{
                        width: 28, height: 28, borderRadius: '50%', flexShrink: 0,
                        background: `linear-gradient(135deg, ${selectedStyle.primaryColor}, ${selectedStyle.accentColor})`,
                        display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 14,
                      }}>✦</div>
                      <div style={{
                        fontSize: 11, color: selectedStyle.primaryColor,
                        background: `${selectedStyle.primaryColor}12`,
                        border: `1px solid ${selectedStyle.primaryColor}33`,
                        padding: '6px 12px', borderRadius: 20,
                      }}>{demoThinkStep}</div>
                    </motion.div>
                  )}

                  {/* AI 타이핑 중 */}
                  {demoTyping && (
                    <div style={{ display: 'flex', gap: 10, alignItems: 'flex-end' }}>
                      <div style={{
                        width: 28, height: 28, borderRadius: '50%', flexShrink: 0,
                        background: `linear-gradient(135deg, ${selectedStyle.primaryColor}, ${selectedStyle.accentColor})`,
                        display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 14,
                      }}>✦</div>
                      <div style={{
                        maxWidth: '75%', padding: '10px 14px',
                        borderRadius: '4px 16px 16px 16px',
                        background: 'rgba(255,255,255,0.07)',
                        border: '1px solid rgba(255,255,255,0.1)',
                        fontSize: 12, color: 'rgba(255,255,255,0.92)',
                        lineHeight: 1.7, whiteSpace: 'pre-wrap', fontFamily: 'monospace',
                      }}>
                        {demoTyping}<span style={{ animation: 'blink 0.8s step-end infinite', opacity: 0.7 }}>▌</span>
                      </div>
                    </div>
                  )}

                  <div ref={chatEndRef} />
                </div>

                {/* 입력창 */}
                <div style={{
                  padding: '12px 16px',
                  borderTop: '1px solid rgba(255,255,255,0.06)',
                  display: 'flex', alignItems: 'center', gap: 10,
                }}>
                  <div style={{
                    flex: 1, padding: '10px 14px',
                    background: 'rgba(255,255,255,0.06)',
                    border: `1px solid ${demoLoading ? selectedStyle.primaryColor + '55' : 'rgba(255,255,255,0.1)'}`,
                    borderRadius: 12, fontSize: 13,
                    color: demoInputTyping ? 'white' : 'rgba(255,255,255,0.25)',
                    transition: 'border-color 0.2s',
                    minHeight: 20,
                  }}>
                    {demoInputTyping || (isEn ? 'Click a button to experience Nexus...' : '버튼을 눌러 체험해보세요...')}
                    {demoInputTyping && <span style={{ animation: 'blink 0.6s step-end infinite', opacity: 0.8 }}>|</span>}
                  </div>
                  <div style={{
                    width: 38, height: 38, borderRadius: 10, flexShrink: 0,
                    background: demoLoading
                      ? `linear-gradient(135deg, ${selectedStyle.primaryColor}, ${selectedStyle.accentColor})`
                      : 'rgba(255,255,255,0.08)',
                    display: 'flex', alignItems: 'center', justifyContent: 'center',
                    fontSize: 16, transition: 'all 0.2s',
                  }}>
                    {demoLoading ? '⚡' : '↑'}
                  </div>
                </div>

              </div>
            </div>

            {/* 하단 CTA */}
            <div style={{
              padding: '16px 20px',
              borderTop: '1px solid rgba(255,255,255,0.06)',
              display: 'flex', alignItems: 'center', gap: 12,
            }}>
              <div style={{ flex: 1, fontSize: 12, color: 'rgba(255,255,255,0.35)' }}>
                {demoResult
                  ? (isEn ? '✦ Impressed? Start now.' : '✦ 어떠셨나요? 지금 바로 시작해보세요.')
                  : (isEn ? '← Click a button on the left to try it' : '← 좌측 버튼을 눌러 체험해보세요')}
              </div>
              <button
                onClick={() => setStep(1)}
                style={{
                  padding: '10px 24px',
                  background: `linear-gradient(135deg, ${selectedStyle.primaryColor}, ${selectedStyle.accentColor})`,
                  border: 'none', borderRadius: 12,
                  color: 'white', fontSize: 13, fontWeight: 700,
                  cursor: 'pointer',
                  boxShadow: `0 4px 20px ${selectedStyle.primaryColor}44`,
                }}
              >
                {demoResult ? (isEn ? 'Get Started →' : '시작하기 →') : (isEn ? 'Skip →' : '건너뛰기 →')}
              </button>
            </div>
          </motion.div>
        )}

        {/* ── Step 1: 아바타 스타일 선택 ── */}
        {step === 1 && (
          <motion.div
            key="step1"
            initial={{ opacity: 0, y: 30 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            style={{ ...card, maxWidth: 620 }}
          >
            {progressBar}

            <h2 style={{ fontSize: 22, fontWeight: 800, color: 'white', marginBottom: 6 }}>
              {isEn ? 'Choose Avatar Style' : '아바타 스타일 선택'}
            </h2>
            <p style={{ fontSize: 13, color: 'rgba(255,255,255,0.42)', marginBottom: 24, lineHeight: 1.6 }}>
              {isEn ? 'Pick your Photorealistic 3D avatar style.' : 'Photorealistic 3D 아바타 스타일을 선택하세요.'}
            </p>

            {/* 스타일 카드 그리드 */}
            <div style={{
              display: 'grid', gridTemplateColumns: '1fr 1fr',
              gap: 12, marginBottom: 24,
            }}>
              {REALISTIC_STYLE_PRESETS.map((s: RealisticStylePreset) => {
                const selected = styleId === s.id
                return (
                  <button
                    key={s.id}
                    onMouseEnter={() => setHoverStyle(s.id)}
                    onMouseLeave={() => setHoverStyle(null)}
                    onClick={() => setStyleId(s.id)}
                    style={{
                      background: selected
                        ? `linear-gradient(135deg, ${s.primaryColor}28, ${s.accentColor}18)`
                        : 'rgba(255,255,255,0.04)',
                      border: selected
                        ? `1.5px solid ${s.primaryColor}88`
                        : '1.5px solid rgba(255,255,255,0.08)',
                      borderRadius: 16, padding: '18px 16px',
                      cursor: 'pointer', textAlign: 'left',
                      transition: 'all 0.22s',
                      position: 'relative', overflow: 'hidden',
                    } as React.CSSProperties}
                  >
                    {/* 아바타 미니 미리보기 */}
                    <div style={{ height: 140, marginBottom: 12, overflow: 'hidden', borderRadius: 10 }}>
                      <Avatar3D
                        emotion={selected ? 'happy' : 'neutral'}
                        speaking={false}
                        listening={false}
                        glbUrl={s.glbUrl}
                        primaryColor={s.primaryColor}
                        accentColor={s.accentColor}
                        preset={s.id as CharacterPreset}
                        width="100%"
                        height={140}
                        preview
                        scale={0.6}
                        characterOffsetY={-0.6}
                        quality="balanced"
                      />
                    </div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6 }}>
                      <span style={{ fontSize: 18 }}>{s.previewEmoji}</span>
                      <span style={{ fontSize: 14, fontWeight: 700, color: 'white' }}>{s.name}</span>
                      {selected && (
                        <span style={{
                          marginLeft: 'auto', fontSize: 10,
                          background: `${s.primaryColor}44`,
                          color: s.primaryColor,
                          padding: '2px 8px', borderRadius: 20, fontWeight: 700,
                        }}>선택됨</span>
                      )}
                    </div>
                    <p style={{ fontSize: 11, color: 'rgba(255,255,255,0.4)', margin: 0, lineHeight: 1.5 }}>
                      {s.tagline}
                    </p>
                  </button>
                )
              })}
            </div>

            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              {nextBtn(() => setStep(2))}
              {backBtn(() => setStep(0))}
            </div>
          </motion.div>
        )}

        {/* ── Step 2: 비서 이름 ── */}
        {step === 2 && (
          <motion.div
            key="step2"
            initial={{ opacity: 0, y: 30 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            style={card}
          >
            {progressBar}

            <div style={{ display: 'flex', gap: 20, alignItems: 'center', marginBottom: 28 }}>
              <div style={{ flexShrink: 0, width: 90, height: 150 }}>
                <Avatar3D
                  emotion="happy"
                  speaking
                  listening={false}
                  glbUrl={selectedStyle.glbUrl}
                  primaryColor={selectedStyle.primaryColor}
                  accentColor={selectedStyle.accentColor}
                  preset={styleId as CharacterPreset}
                  width={90}
                  height={150}
                  preview
                  scale={0.55}
                  characterOffsetY={-0.6}
                  quality="balanced"
                />
              </div>
              <div>
                <h2 style={{ fontSize: 22, fontWeight: 800, color: 'white', marginBottom: 8 }}>
                  {isEn ? 'Name Your Assistant' : '비서 이름을 설정해주세요'}
                </h2>
                <p style={{ fontSize: 13, color: 'rgba(255,255,255,0.4)', lineHeight: 1.6 }}>
                  {isEn
                    ? `"Hey ${nameInput || 'Nexus'}, what time is it?" — wake it by name.`
                    : `"${nameInput || '넥서스'}아, 지금 몇 시야?" 처럼 이름으로 깨울 수 있어요.`}
                </p>
              </div>
            </div>

            {/* 이름 추천 */}
            <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginBottom: 16 }}>
              {SUGGESTED_NAMES.map(n => (
                <button
                  key={n}
                  onClick={() => { setNameInput(n); setName(n) }}
                  style={{
                    padding: '6px 14px',
                    background: nameInput === n
                      ? `${selectedStyle.primaryColor}44`
                      : 'rgba(255,255,255,0.06)',
                    border: nameInput === n
                      ? `1px solid ${selectedStyle.primaryColor}88`
                      : '1px solid rgba(255,255,255,0.08)',
                    borderRadius: 20, cursor: 'pointer',
                    color: nameInput === n ? selectedStyle.primaryColor : 'rgba(255,255,255,0.6)',
                    fontSize: 13, fontWeight: 600,
                    transition: 'all 0.18s',
                  } as React.CSSProperties}
                >
                  {n}
                </button>
              ))}
            </div>

            <input
              value={nameInput}
              onChange={e => { setNameInput(e.target.value); setName(e.target.value) }}
              placeholder="직접 입력..."
              style={{
                width: '100%', padding: '12px 16px',
                background: 'rgba(255,255,255,0.05)',
                border: `1px solid ${selectedStyle.primaryColor}44`,
                borderRadius: 12, color: 'white', fontSize: 14,
                outline: 'none', marginBottom: 24, boxSizing: 'border-box',
              } as React.CSSProperties}
              onKeyDown={e => e.key === 'Enter' && setStep(3)}
            />

            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              {nextBtn(() => { setName(nameInput.trim() || (isEn ? 'Nexus' : '넥서스')); setStep(3) })}
              {backBtn(() => setStep(1), isEn ? '← Back to Character' : '← 캐릭터 다시 선택')}
            </div>
          </motion.div>
        )}

        {/* ── Step 3: 사용자 호칭 ── */}
        {step === 3 && (
          <motion.div
            key="step3"
            initial={{ opacity: 0, y: 30 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            style={card}
          >
            {progressBar}

            <h2 style={{ fontSize: 22, fontWeight: 800, color: 'white', marginBottom: 8 }}>
              {isEn ? 'What should Nexus call you?' : '어떻게 불러드릴까요?'}
            </h2>
            <p style={{ fontSize: 13, color: 'rgba(255,255,255,0.4)', marginBottom: 24, lineHeight: 1.6 }}>
              {isEn
                ? `"${assistantName} optimized your PC, ${userInput || 'Boss'}!" — used in notifications.`
                : `"${assistantName}이 ${userInput || '주인님'}의 PC를 최적화했어요!" 처럼 알림을 드릴 때 사용해요.`}
            </p>

            <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginBottom: 16 }}>
              {USER_NAMES.map(n => (
                <button
                  key={n}
                  onClick={() => { setUserInput(n); setUserName(n) }}
                  style={{
                    padding: '6px 16px',
                    background: userInput === n
                      ? `${selectedStyle.primaryColor}44`
                      : 'rgba(255,255,255,0.06)',
                    border: userInput === n
                      ? `1px solid ${selectedStyle.primaryColor}88`
                      : '1px solid rgba(255,255,255,0.08)',
                    borderRadius: 20, cursor: 'pointer',
                    color: userInput === n ? selectedStyle.primaryColor : 'rgba(255,255,255,0.6)',
                    fontSize: 13, fontWeight: 600, transition: 'all 0.18s',
                  } as React.CSSProperties}
                >
                  {n}
                </button>
              ))}
            </div>

            <input
              value={userInput}
              onChange={e => { setUserInput(e.target.value); setUserName(e.target.value) }}
              placeholder="직접 입력..."
              style={{
                width: '100%', padding: '12px 16px',
                background: 'rgba(255,255,255,0.05)',
                border: `1px solid ${selectedStyle.primaryColor}44`,
                borderRadius: 12, color: 'white', fontSize: 14,
                outline: 'none', marginBottom: 24, boxSizing: 'border-box',
              } as React.CSSProperties}
              onKeyDown={e => e.key === 'Enter' && setStep(4)}
            />

            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              {nextBtn(() => { setUserName(userInput.trim() || (isEn ? 'Boss' : '주인님')); setStep(4) }, isEn ? 'Next' : '다음')}
              {backBtn(() => setStep(2), isEn ? '← Back' : '← 이전')}
            </div>
          </motion.div>
        )}

        {/* ── Step 4: 직업군 선택 ── */}
        {step === 4 && (
          <motion.div
            key="step5"
            initial={{ opacity: 0, y: 30 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            style={card}
          >
            {progressBar}

            <div style={{ marginBottom: 20 }}>
              <div style={{
                fontSize: 11, letterSpacing: '0.12em',
                color: selectedStyle.primaryColor, marginBottom: 8, fontWeight: 600,
              }}>
                {isEn ? 'AI OPTIMIZATION' : 'AI 최적화'}
              </div>
              <h2 style={{ fontSize: 20, fontWeight: 800, color: 'white', marginBottom: 6 }}>
                {isEn ? 'What do you do?' : '직업군을 선택해주세요'}
              </h2>
              <p style={{ fontSize: 12, color: 'rgba(255,255,255,0.4)', lineHeight: 1.6 }}>
                {isEn ? 'AI responses and workflows are optimized for your role.' : '선택한 직업에 맞춰 AI 응답과 워크플로우가 최적화됩니다.'}
              </p>
            </div>

            <div style={{
              display: 'grid', gridTemplateColumns: '1fr 1fr',
              gap: 10, marginBottom: 24,
            }}>
              {JOB_PERSONAS.map(p => {
                const selected = selectedJobId === p.id
                return (
                  <button
                    key={p.id}
                    onClick={() => setSelectedJobId(p.id)}
                    style={{
                      background: selected
                        ? `linear-gradient(135deg, ${p.color}28, ${p.color}12)`
                        : 'rgba(255,255,255,0.04)',
                      border: selected
                        ? `1.5px solid ${p.color}88`
                        : '1.5px solid rgba(255,255,255,0.08)',
                      borderRadius: 14, padding: '14px 14px',
                      cursor: 'pointer', textAlign: 'left',
                      transition: 'all 0.18s', position: 'relative',
                    } as React.CSSProperties}
                  >
                    <div style={{ fontSize: 22, marginBottom: 6 }}>{p.emoji}</div>
                    <div style={{
                      fontSize: 12, fontWeight: 700,
                      color: selected ? 'white' : 'rgba(255,255,255,0.75)',
                      marginBottom: 3, lineHeight: 1.3,
                    }}>
                      {p.name}
                    </div>
                    <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.35)', lineHeight: 1.4 }}>
                      {p.desc}
                    </div>
                    {selected && (
                      <div style={{
                        position: 'absolute', top: 8, right: 8,
                        width: 16, height: 16, borderRadius: '50%',
                        background: p.color,
                        display: 'flex', alignItems: 'center', justifyContent: 'center',
                        fontSize: 9, color: 'white', fontWeight: 800,
                      }}>✓</div>
                    )}
                  </button>
                )
              })}
            </div>

            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              {nextBtn(() => setStep(5), isEn ? 'Next →' : '다음 →')}
              {backBtn(() => setStep(3), isEn ? '← Back' : '← 이전')}
            </div>
          </motion.div>
        )}

        {/* ── Step 5: 구글 로그인 / 3일 무료 체험 ── */}
        {step === 5 && (
          <motion.div
            key="step6"
            initial={{ opacity: 0, y: 30 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            style={card}
          >
            {progressBar}

            <div style={{ textAlign: 'center', marginBottom: 28 }}>
              <div style={{ fontSize: 40, marginBottom: 12 }}>🔐</div>
              <h2 style={{ fontSize: 22, fontWeight: 800, color: 'white', marginBottom: 8 }}>
                {isEn ? 'Get Started' : '시작하기'}
              </h2>
              <p style={{ fontSize: 13, color: 'rgba(255,255,255,0.4)', lineHeight: 1.6 }}>
                {isEn ? <>Sign in with Google and your<br />3-day free trial starts automatically.</> : <>구글 계정으로 로그인하면<br />3일 무료 체험이 자동으로 시작됩니다.</>}
              </p>
            </div>

            {/* 로그인 영역 */}
            {!googleEmail ? (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                {/* X 닫기 버튼 */}
                <button
                  onClick={async () => {
                    const { getCurrentWindow } = await import('@tauri-apps/api/window')
                    getCurrentWindow().close()
                  }}
                  style={{
                    position: 'absolute', top: 16, right: 16,
                    width: 28, height: 28, borderRadius: '50%',
                    background: 'rgba(255,255,255,0.08)', border: '1px solid rgba(255,255,255,0.15)',
                    color: 'rgba(255,255,255,0.5)', fontSize: 14, cursor: 'pointer',
                    display: 'flex', alignItems: 'center', justifyContent: 'center',
                    lineHeight: 1,
                  }}
                >✕</button>

                {/* Google 로그인 버튼 */}
                <button
                  onClick={handleGoogleLogin}
                  disabled={googleLoading}
                  style={{
                    width: '100%', padding: '14px 20px',
                    background: googleLoading ? 'rgba(255,255,255,0.1)' : 'white',
                    border: 'none', borderRadius: 12, cursor: googleLoading ? 'wait' : 'pointer',
                    display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 12,
                    fontSize: 15, fontWeight: 700, color: '#1a1a2e',
                    boxShadow: '0 4px 20px rgba(0,0,0,0.4)',
                    transition: 'opacity 0.2s', opacity: googleLoading ? 0.6 : 1,
                  } as React.CSSProperties}
                >
                  <svg width="20" height="20" viewBox="0 0 24 24">
                    <path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"/>
                    <path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/>
                    <path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"/>
                    <path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"/>
                  </svg>
                  {googleLoading ? (isEn ? 'Connecting...' : '연결 중...') : (isEn ? 'Continue with Google' : 'Google 계정으로 시작하기')}
                </button>


                {/* 관리자 로그인 토글 */}
                <button
                  onClick={() => setShowAdminLogin(v => !v)}
                  style={{
                    background: 'none', border: 'none', cursor: 'pointer',
                    fontSize: 11, color: 'rgba(255,255,255,0.2)', textAlign: 'center', padding: '4px 0',
                  }}
                >
                  {showAdminLogin ? '▲ Close' : 'Admin Login'}
                </button>

                {showAdminLogin && (
                  <motion.div
                    initial={{ opacity: 0, height: 0 }}
                    animate={{ opacity: 1, height: 'auto' }}
                    style={{ display: 'flex', flexDirection: 'column', gap: 8 }}
                  >
                    <input
                      type="email"
                      placeholder="Email"
                      value={loginEmail}
                      onChange={e => setLoginEmail(e.target.value)}
                      style={{
                        width: '100%', padding: '10px 14px', boxSizing: 'border-box',
                        background: 'rgba(255,255,255,0.05)', border: '1px solid rgba(255,255,255,0.12)',
                        borderRadius: 10, color: 'white', fontSize: 13, outline: 'none',
                      } as React.CSSProperties}
                    />
                    <input
                      type="password"
                      placeholder="Password"
                      value={loginPassword}
                      onChange={e => setLoginPassword(e.target.value)}
                      onKeyDown={e => e.key === 'Enter' && handleAdminLogin()}
                      style={{
                        width: '100%', padding: '10px 14px', boxSizing: 'border-box',
                        background: 'rgba(255,255,255,0.05)', border: '1px solid rgba(255,255,255,0.12)',
                        borderRadius: 10, color: 'white', fontSize: 13, outline: 'none',
                      } as React.CSSProperties}
                    />
                    {loginError && (
                      <p style={{ fontSize: 11, color: '#f87171', margin: 0 }}>{loginError}</p>
                    )}
                    <button
                      onClick={handleAdminLogin}
                      style={{
                        width: '100%', padding: '10px', border: 'none', borderRadius: 10,
                        background: 'rgba(79,126,247,0.8)', color: 'white',
                        fontSize: 13, fontWeight: 700, cursor: 'pointer',
                      }}
                    >
                      {isEn ? 'Login' : '로그인'}
                    </button>
                  </motion.div>
                )}
              </div>
            ) : (
              <div style={{
                background: 'rgba(74,222,128,0.1)', border: '1px solid rgba(74,222,128,0.3)',
                borderRadius: 12, padding: '16px 20px', textAlign: 'center',
              }}>
                <div style={{ fontSize: 24, marginBottom: 6 }}>✅</div>
                <div style={{ fontSize: 14, fontWeight: 700, color: '#4ade80' }}>{googleEmail}</div>
                <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.4)', marginTop: 4 }}>{isEn ? '3-day free trial ready to start' : '3일 무료 체험 시작 준비 완료'}</div>
              </div>
            )}

            {/* 구독 혜택 */}
            <div style={{
              background: `${selectedStyle.primaryColor}10`,
              border: `1px solid ${selectedStyle.primaryColor}25`,
              borderRadius: 12, padding: '14px 18px', marginTop: 16, fontSize: 12,
              color: 'rgba(255,255,255,0.6)', lineHeight: 2,
            }}>
              {(isEn
                ? ['✦ Unlimited access to all AI features', '✦ Deep Search · Real-time web search', '✦ Auto updates', '✦ Early Bird $12.99/mo after 3 days · Cancel anytime']
                : ['✦ 모든 AI 기능 무제한 사용', '✦ 딥서치 · 실시간 웹 검색', '✦ 자동 업데이트', '✦ 3일 후 Early Bird 월 14,900원 · 언제든 해지 가능']
              ).map(t => (
                <div key={t}>{t}</div>
              ))}
            </div>

            {/* 최종 요약 */}
            <div style={{
              background: 'rgba(255,255,255,0.03)', border: '1px solid rgba(255,255,255,0.08)',
              borderRadius: 12, padding: '14px 18px', marginTop: 12, fontSize: 13,
              color: 'rgba(255,255,255,0.7)', lineHeight: 1.8,
            }}>
              <div>{isEn ? 'Assistant' : '비서'}: <strong style={{ color: 'white' }}>{assistantName}</strong></div>
              <div>{isEn ? 'Your name' : '호칭'}: <strong style={{ color: 'white' }}>{userInput || (isEn ? 'Boss' : '주인님')}</strong></div>
              <div>{isEn ? 'Character' : '캐릭터'}: <strong style={{ color: selectedStyle.primaryColor }}>{selectedStyle.name}</strong></div>
              <div>{isEn ? 'Voice' : '음성'}: <strong style={{ color: 'rgba(255,255,255,0.4)' }}>{isEn ? 'Default voice' : '기본 음성'}</strong></div>
            </div>

            <div style={{ display: 'flex', flexDirection: 'column', gap: 8, marginTop: 16 }}>
              {googleEmail
                ? nextBtn(() => void handleComplete(), isEn ? `Start with ${assistantName} ✦` : `${assistantName} 시작하기 ✦`)
                : <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.25)', textAlign: 'center' }}>{isEn ? 'Sign in with Google to get started' : '구글 로그인 후 시작할 수 있습니다'}</div>
              }
              {backBtn(() => setStep(4), isEn ? '← Back' : '← 이전')}
            </div>
          </motion.div>
        )}

      </AnimatePresence>
    </div>
  )
}
