/**
 * OnboardingFlow v7 — Marketing Optimized
 *
 * Step 0: 직업군 선택 (10개) — "어떤 일을 하세요?"
 * Step 1: 직업군 전용 WOW 데모
 * Step 2: 플랜 선택 (Free / Pro $19 / Team $49)
 * Step 3: Google Login
 * Step 4: Avatar + Name + Nickname (combined)
 * Step 5: Complete
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
import { openCheckout, PADDLE_PRICES } from '../../lib/paddle'

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

const SUGGESTED_NAMES = ['넥서스', '아리아', '노바', '카이', 'Aria', 'Nova', 'Nexus', 'Eve']
const USER_NAMES_KO = ['주인님', '사용자', '선생님', '파트너']
const USER_NAMES_EN = ['Boss', 'User', 'Partner', 'Chief']

const STEPS_TOTAL = 6

const sleep = (ms: number) => new Promise(res => setTimeout(res, ms))

// ─── 10-persona arrays ───────────────────────────────────────────────────────
const JOB_PERSONAS_KO = [
  { id: 'developer',  emoji: '💻', name: '개발자 / IT 엔지니어',    desc: '코드·디버깅·아키텍처', color: '#6366f1', proFeature: null, proLabel: null },
  { id: 'marketer',   emoji: '📊', name: '마케터 / 디지털 마케터',  desc: '트렌드·SNS·경쟁사 분석', color: '#f59e0b', proFeature: null, proLabel: null },
  { id: 'sales',      emoji: '🤝', name: '영업 / 세일즈',           desc: '이메일·미팅·고객 설득', color: '#10b981', proFeature: null, proLabel: null },
  { id: 'pm',         emoji: '📋', name: 'PM / 기획자',             desc: '문서요약·로드맵·의사결정', color: '#0ea5e9', proFeature: null, proLabel: null },
  { id: 'designer',   emoji: '🎨', name: '디자이너 / 크리에이터',  desc: '레퍼런스·파일정리·콘텐츠', color: '#ec4899', proFeature: null, proLabel: null },
  { id: 'freelancer', emoji: '🚀', name: '프리랜서 / 1인 사업자',  desc: '견적·클라이언트·세금', color: '#8b5cf6', proFeature: null, proLabel: null },
  { id: 'investor',   emoji: '📈', name: '투자자 / 트레이더',      desc: '주식·코인·ETF 실시간 분석', color: '#22c55e', proFeature: 'stock_analysis', proLabel: '📈 Pro 전용 — 주식 분석 무제한' },
  { id: 'medical',    emoji: '🏥', name: '의사 / 의료진',          desc: '의학논문·약물·가이드라인', color: '#06b6d4', proFeature: 'medical_search', proLabel: '🏥 Pro 전용 — 의학 검색 무제한' },
  { id: 'legal',      emoji: '⚖️', name: '변호사 / 법무 담당자',  desc: '계약서검토·판례·법령', color: '#f97316', proFeature: 'contract_review', proLabel: '⚖️ Pro 전용 — 계약서 검토 무제한' },
  { id: 'creator',    emoji: '🎬', name: '유튜버 / 인플루언서',   desc: '스크립트·썸네일·해시태그', color: '#ef4444', proFeature: 'content_script', proLabel: '🎬 Pro 전용 — 스크립트 생성 무제한' },
]

const JOB_PERSONAS_EN = [
  { id: 'developer',  emoji: '💻', name: 'Developer / IT Engineer',    desc: 'Code · Debug · Architecture', color: '#6366f1', proFeature: null, proLabel: null },
  { id: 'marketer',   emoji: '📊', name: 'Marketer / Digital Marketer', desc: 'Trends · SNS · Competitor analysis', color: '#f59e0b', proFeature: null, proLabel: null },
  { id: 'sales',      emoji: '🤝', name: 'Sales / Account Executive',   desc: 'Email · Meetings · Client persuasion', color: '#10b981', proFeature: null, proLabel: null },
  { id: 'pm',         emoji: '📋', name: 'PM / Product Planner',        desc: 'Doc summaries · Roadmap · Decisions', color: '#0ea5e9', proFeature: null, proLabel: null },
  { id: 'designer',   emoji: '🎨', name: 'Designer / Creator',          desc: 'References · File org · Content', color: '#ec4899', proFeature: null, proLabel: null },
  { id: 'freelancer', emoji: '🚀', name: 'Freelancer / Solopreneur',    desc: 'Quotes · Clients · Tax', color: '#8b5cf6', proFeature: null, proLabel: null },
  { id: 'investor',   emoji: '📈', name: 'Investor / Trader',           desc: 'Stocks · Crypto · ETF real-time', color: '#22c55e', proFeature: 'stock_analysis', proLabel: '📈 Pro only — Unlimited stock analysis' },
  { id: 'medical',    emoji: '🏥', name: 'Doctor / Medical Staff',      desc: 'Med papers · Drugs · Guidelines', color: '#06b6d4', proFeature: 'medical_search', proLabel: '🏥 Pro only — Unlimited medical search' },
  { id: 'legal',      emoji: '⚖️', name: 'Lawyer / Legal Counsel',     desc: 'Contract review · Cases · Law', color: '#f97316', proFeature: 'contract_review', proLabel: '⚖️ Pro only — Unlimited contract review' },
  { id: 'creator',    emoji: '🎬', name: 'YouTuber / Influencer',       desc: 'Scripts · Thumbnails · Hashtags', color: '#ef4444', proFeature: 'content_script', proLabel: '🎬 Pro only — Unlimited script generation' },
]

// ─── Job-specific WOW demos ───────────────────────────────────────────────────
const JOB_DEMOS: Record<string, { query: string; steps: string[]; result: string; proHint?: string }> = {
  investor: {
    query: '삼성전자 주가 지금 사도 될까?',
    steps: ['📡 실시간 주가·재무 데이터 수집 중...', '📊 PER·PBR·ROE 분석 중...', '🧠 AI 투자 인사이트 생성 중...'],
    result: '📈 삼성전자 (005930) 분석 완료\n\n현재가 79,200원 ▲ +1.2%\nPER 14.3 (업종 평균 18.2 → 저평가)\nPBR 1.08 · 배당수익률 2.1%\n\n✅ 호재: HBM3E 엔비디아 공급 승인\n⚠️ 리스크: 원달러 1,380원 돌파 시 수출 감소\n\n💡 AI: 현 구간 분할매수 관심. 목표가 88,000원\n\n⚠️ 투자 판단은 본인 책임입니다.',
    proHint: 'Pro 플랜에서 이 분석을 매일 자동으로 받아보세요',
  },
  medical: {
    query: '메트포르민 신기능 저하 환자 용량?',
    steps: ['📚 PubMed 최신 논문 검색 중...', '🔬 근거 수준 분류 중...', '📋 임상 가이드라인 요약 중...'],
    result: '🏥 메트포르민 신기능별 용량 가이드\n\n📊 근거: ADA 2024 (Grade A)\n\neGFR ≥ 45: 표준용량 1,000mg bid\neGFR 30~44: 500mg bid (감량)\neGFR < 30: 투여 금기\n\n⚠️ 조영제 사용 전 48h 중단\n\n🆕 2024 업데이트: 서방형 제제 위장 부작용 68% 감소\n\n⚠️ 임상 결정 시 전문의 판단 필수',
    proHint: 'Pro 플랜에서 의학 검색을 무제한으로 사용하세요',
  },
  legal: {
    query: '이 근로계약서 검토해줘',
    steps: ['📄 계약서 조항 분류 중...', '⚖️ 판례 데이터베이스 대조 중...', '🔍 리스크 등급 분류 중...'],
    result: '⚖️ 근로계약서 검토 완료\n\n🔴 고위험 2개\n• 제7조 포괄임금제 → 대법원 판례 무효 가능\n  수정안: "연장·야간근로 별도 산정"\n• 제12조 경업금지 5년 → 과도, 법원 2년 축소\n\n🟡 주의 1개\n• 제15조 손해배상 금액 상한 미기재\n\n✅ 표준 조항 12개 이상 없음\n\n전체 리스크: 🟡 보통\n⚠️ 최종 판단은 변호사 확인 필요',
    proHint: 'Pro 플랜에서 계약서 검토를 무제한으로 사용하세요',
  },
  creator: {
    query: 'AI 활용법 유튜브 스크립트 만들어줘',
    steps: ['🔍 트렌드·경쟁 영상 분석 중...', '✍️ 훅·본문·아웃트로 구성 중...', '🏷️ SEO 최적화 중...'],
    result: '🎬 유튜브 스크립트 완성!\n\n🎯 훅: "AI 모르면 2026년 연봉 500만원 손해"\n\n📌 인트로: 저는 AI 하나로 업무 40시간을 절약했는데요...\n\n📋 본문\n① 반복업무 자동화로 월 20시간 절약\n② AI 리서치로 콘텐츠 준비 90% 단축\n③ 스크립트 자동화로 주 3편 업로드 가능\n\n🔚 아웃트로: 댓글에 자동화할 업무 알려주세요!\n\n📌 제목: "AI 쓰니까 퇴근이 3시간 빨라졌다"\n🏷️ #AI업무 #생산성 #유튜브스크립트',
    proHint: 'Pro 플랜에서 스크립트 생성을 무제한으로 사용하세요',
  },
  developer: {
    query: 'Python 메모리 누수 원인 찾아줘',
    steps: ['🔍 코드 패턴 스캔 중...', '🐛 메모리 프로파일 분석 중...'],
    result: '🐛 메모리 누수 발견!\n\n원인: 전역 리스트에 이벤트 핸들러 누적\n\n수정 전:\nhandlers.append(lambda: process(data))\n\n수정 후:\nweakref.ref(handler)  # 약한 참조\n\n✅ 수정 후 메모리 43% 감소 예상',
  },
  marketer: {
    query: '경쟁사 마케팅 전략 분석해줘',
    steps: ['🔍 경쟁사 SNS·웹사이트 크롤링 중...', '📊 트렌드 분석 중...'],
    result: '📊 경쟁사 마케팅 분석 완료\n\n상위 3개사 공통 전략:\n① 숏폼 영상 주 5회 이상\n② 인플루언서 마이크로 타겟팅\n③ 커뮤니티 기반 바이럴\n\n💡 기회: 롱폼 교육 콘텐츠 공백 발견',
  },
  sales: {
    query: '고객사 미팅 전 사전 조사해줘',
    steps: ['🔍 기업 정보 수집 중...', '📋 인사이트 정리 중...'],
    result: '📋 미팅 사전 조사 완료\n\n기업 현황: 최근 Series B 투자 유치\n핵심 과제: 영업팀 생산성 향상\n\n💡 공략 포인트:\n• ROI 중심 제안 (비용 절감 수치 강조)\n• 의사결정자: CTO + CFO 동시 공략',
  },
  pm: {
    query: '오늘 미팅 회의록 요약해줘',
    steps: ['📋 회의 내용 분석 중...', '✍️ 핵심 사항 추출 중...'],
    result: '✅ 회의록 요약 완료\n\n📌 주요 결정\n• Q2 예산 15% 증액 확정\n• 신규 파트너십 6월 말 체결\n\n📅 다음 액션\n• 김팀장: 계약서 초안 (5/30)\n• 이대리: 예산 보고서 (6/3)',
  },
  designer: {
    query: '앱 아이콘 레퍼런스 찾아줘',
    steps: ['🎨 디자인 레퍼런스 수집 중...', '✨ 트렌드 분석 중...'],
    result: '🎨 앱 아이콘 트렌드 2024\n\n1. 글래스모피즘 (투명+블러)\n2. 3D 미니멀 아이소메트릭\n3. 그라데이션 + 세리프 조합\n\n참고 앱: Notion, Linear, Arc Browser\n\n💡 추천: 다크 배경 + 보라 그라데이션',
  },
  freelancer: {
    query: '프로젝트 견적서 작성해줘',
    steps: ['📊 시장 단가 조사 중...', '💰 견적 계산 중...'],
    result: '💰 프리랜서 견적서 완성\n\n웹사이트 개발 프로젝트\n기획: 5일 × 25만원 = 125만원\n디자인: 7일 × 30만원 = 210만원\n개발: 14일 × 40만원 = 560만원\n\n소계: 895만원\nVAT(10%): 89.5만원\n\n총액: 984.5만원',
  },
}

const JOB_DEMOS_EN: Record<string, { query: string; steps: string[]; result: string; proHint?: string }> = {
  investor: {
    query: 'Should I buy Samsung stock now?',
    steps: ['📡 Fetching real-time price & financials...', '📊 Analyzing PER·PBR·ROE...', '🧠 Generating AI investment insight...'],
    result: '📈 Samsung (005930) Analysis\n\nCurrent: ₩79,200 ▲ +1.2%\nPER 14.3 (sector avg 18.2 → undervalued)\nPBR 1.08 · Dividend yield 2.1%\n\n✅ Catalyst: HBM3E supply to NVIDIA approved\n⚠️ Risk: KRW/USD above 1,380 → export headwind\n\n💡 AI: Gradual accumulation zone. Target ₩88,000\n\n⚠️ Investment decisions are your responsibility.',
    proHint: 'Get this analysis delivered daily with Pro plan',
  },
  medical: {
    query: 'Metformin dosing in renal impairment?',
    steps: ['📚 Searching latest PubMed papers...', '🔬 Classifying evidence levels...', '📋 Summarizing clinical guidelines...'],
    result: '🏥 Metformin Renal Dosing Guide\n\n📊 Evidence: ADA 2024 (Grade A)\n\neGFR ≥ 45: Standard 1,000mg bid\neGFR 30~44: 500mg bid (reduced)\neGFR < 30: Contraindicated\n\n⚠️ Hold 48h before contrast media\n\n🆕 2024 Update: XR formulation 68% less GI side effects\n\n⚠️ Clinical decisions require physician judgement',
    proHint: 'Unlimited medical search with Pro plan',
  },
  legal: {
    query: 'Review this employment contract',
    steps: ['📄 Classifying contract clauses...', '⚖️ Cross-referencing case law...', '🔍 Grading risk levels...'],
    result: '⚖️ Employment Contract Review\n\n🔴 High Risk (2)\n• Clause 7: Blanket overtime → Supreme Court may void\n  Suggested: "OT/night work calculated separately"\n• Clause 12: Non-compete 5 years → excessive, courts reduce to 2\n\n🟡 Caution (1)\n• Clause 15: No cap on damages specified\n\n✅ 12 standard clauses OK\n\nOverall Risk: 🟡 Moderate\n⚠️ Final review by licensed attorney required',
    proHint: 'Unlimited contract reviews with Pro plan',
  },
  creator: {
    query: 'Write a YouTube script on AI tools',
    steps: ['🔍 Analyzing trending & competitor videos...', '✍️ Building hook · body · outro...', '🏷️ SEO optimization...'],
    result: '🎬 YouTube Script Ready!\n\n🎯 Hook: "Not using AI in 2026 costs you $5k/year"\n\n📌 Intro: I saved 40 hours of work with a single AI tool...\n\n📋 Body\n① Automate repetitive tasks → save 20h/month\n② AI research → 90% faster content prep\n③ Script automation → upload 3 videos/week\n\n🔚 Outro: Drop your #1 task to automate in the comments!\n\n📌 Title: "AI Made Me Leave Work 3 Hours Early"\n🏷️ #AItools #productivity #youtubescript',
    proHint: 'Unlimited script generation with Pro plan',
  },
  developer: {
    query: 'Find the cause of a Python memory leak',
    steps: ['🔍 Scanning code patterns...', '🐛 Profiling memory usage...'],
    result: '🐛 Memory Leak Found!\n\nCause: Event handlers accumulating in a global list\n\nBefore:\nhandlers.append(lambda: process(data))\n\nAfter:\nweakref.ref(handler)  # weak reference\n\n✅ Expected 43% memory reduction after fix',
  },
  marketer: {
    query: 'Analyze competitor marketing strategy',
    steps: ['🔍 Crawling competitor SNS & websites...', '📊 Analyzing trends...'],
    result: '📊 Competitor Marketing Analysis\n\nTop 3 brands common strategies:\n① Short-form video 5+ times/week\n② Micro-influencer targeting\n③ Community-driven viral\n\n💡 Opportunity: Long-form educational content gap found',
  },
  sales: {
    query: 'Research client before the meeting',
    steps: ['🔍 Gathering company information...', '📋 Organizing insights...'],
    result: '📋 Pre-Meeting Research Done\n\nCompany: Recently raised Series B\nKey challenge: Sales team productivity\n\n💡 Pitch angle:\n• Lead with ROI (cost savings in numbers)\n• Target: CTO + CFO simultaneously',
  },
  pm: {
    query: 'Summarize today\'s meeting notes',
    steps: ['📋 Analyzing meeting content...', '✍️ Extracting key points...'],
    result: '✅ Meeting Summary Done\n\n📌 Key Decisions\n• Q2 budget +15% approved\n• New partnership deal by end of June\n\n📅 Action Items\n• Kim (lead): Contract draft (5/30)\n• Lee: Budget report (6/3)',
  },
  designer: {
    query: 'Find app icon references',
    steps: ['🎨 Collecting design references...', '✨ Analyzing trends...'],
    result: '🎨 App Icon Trends 2024\n\n1. Glassmorphism (transparent+blur)\n2. 3D minimal isometric\n3. Gradient + serif combo\n\nReference apps: Notion, Linear, Arc Browser\n\n💡 Recommended: Dark bg + purple gradient',
  },
  freelancer: {
    query: 'Write a project quote',
    steps: ['📊 Researching market rates...', '💰 Calculating quote...'],
    result: '💰 Freelance Quote Ready\n\nWebsite Development Project\nPlanning: 5d × $250 = $1,250\nDesign: 7d × $300 = $2,100\nDevelopment: 14d × $400 = $5,600\n\nSubtotal: $8,950\nTax (10%): $895\n\nTotal: $9,845',
  },
}

export function OnboardingFlow({ onComplete }: OnboardingFlowProps) {
  const { isLoggedIn, userEmail, setUserLang } = useAppStore()
  const didAutoComplete = useRef(false)
  const [lang, setLang]               = useState<'ko' | 'en'>('ko')
  const isEn                          = lang === 'en'
  const JOB_PERSONAS                  = isEn ? JOB_PERSONAS_EN : JOB_PERSONAS_KO
  const USER_NAMES                    = isEn ? USER_NAMES_EN : USER_NAMES_KO
  const [step, setStep]               = useState(0)
  const [styleId, setStyleId]         = useState<RealisticStyleId>('kpop_star')
  const [assistantName, setName]      = useState(isEn ? 'Nexus' : '넥서스')
  const [nameInput, setNameInput]     = useState(isEn ? 'Nexus' : '넥서스')
  const [userInput, setUserInput]     = useState('')
  const [userName, setUserName]       = useState('')
  const [hoverStyle, setHoverStyle]   = useState<RealisticStyleId | null>(null)
  const [openaiKey]                   = useState(() => localStorage.getItem('nexus-openai-key') ?? '')
  const [googleLoading, setGoogleLoading] = useState(false)
  const [googleEmail, setGoogleEmail]     = useState('')
  const [loginEmail, setLoginEmail]       = useState('')
  const [loginPassword, setLoginPassword] = useState('')
  const [loginError, setLoginError]       = useState('')
  const [showAdminLogin, setShowAdminLogin] = useState(false)
  const [selectedJobId, setSelectedJobId] = useState<string>('developer')
  const [jobSelected, setJobSelected]     = useState(false) // animation flag
  const [gcalConnected, setGcalConnected] = useState(false)
  const [gcalEmail, setGcalEmail]         = useState('')
  const [gcalLoading, setGcalLoading]     = useState(false)
  const [selectedPlan, setSelectedPlan]   = useState<'free' | 'pro' | 'team'>('free')

  // Demo state for Step 1
  const [demoRunning, setDemoRunning]     = useState(false)
  const [demoStarted, setDemoStarted]     = useState(false)
  const [demoThinkStep, setDemoThinkStep] = useState('')
  const [demoTyping, setDemoTyping]       = useState('')
  const [demoResult, setDemoResult]       = useState('')
  const [demoInputTyping, setDemoInputTyping] = useState('')
  const demoEndRef = useRef<HTMLDivElement>(null)

  const selectedStyle = REALISTIC_STYLE_PRESETS.find(s => s.id === styleId) ?? REALISTIC_STYLE_PRESETS[0]
  const selectedJob   = JOB_PERSONAS.find(p => p.id === selectedJobId) ?? JOB_PERSONAS[0]
  const jobColor      = selectedJob.color

  // Google OAuth callback
  useEffect(() => {
    if (isLoggedIn && userEmail && step === 3 && !didAutoComplete.current) {
      didAutoComplete.current = true
      setGoogleEmail(userEmail)
      setStep(4)
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isLoggedIn, userEmail])

  // ── Issue #8: step + selectedJobId 둘 다 의존해서 직업 재선택 시 확실히 재실행 ──
  const demoJobRef = useRef<string>('')
  useEffect(() => {
    if (step === 1 && (!demoStarted || demoJobRef.current !== selectedJobId)) {
      demoJobRef.current = selectedJobId
      setDemoStarted(true)
      void runJobDemo()
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [step, selectedJobId])

  const runJobDemo = async () => {
    const demos = isEn ? JOB_DEMOS_EN : JOB_DEMOS
    const demo = demos[selectedJobId] ?? demos['developer']
    setDemoRunning(true)
    setDemoResult('')
    setDemoTyping('')
    setDemoThinkStep('')
    setDemoInputTyping('')

    // Type query in fake input — 청크 단위로 업데이트 (성능)
    for (let i = 1; i <= demo.query.length; i++) {
      setDemoInputTyping(demo.query.slice(0, i))
      await sleep(38)
    }
    await sleep(400)

    // Thinking steps
    for (const s of demo.steps) {
      setDemoThinkStep(s)
      await sleep(900)
    }
    setDemoThinkStep('')

    // ── Issue #5: 글자 단위 sleep 대신 청크(5자) 단위로 업데이트 → 렌더 80% 감소 ──
    const CHUNK = 5
    const result = demo.result
    let typed = ''
    for (let i = 0; i < result.length; i += CHUNK) {
      typed = result.slice(0, i + CHUNK)
      setDemoTyping(typed)
      await sleep(20)
      if (i % 50 === 0) demoEndRef.current?.scrollIntoView({ behavior: 'smooth' })
    }
    setDemoResult(result)
    setDemoTyping('')
    setDemoRunning(false)
  }

  const handleAdminLogin = () => {
    if (loginEmail.trim() === ADMIN_EMAIL && loginPassword === ADMIN_PASSWORD) {
      localStorage.setItem('nexus-user-email', ADMIN_EMAIL)
      localStorage.setItem('nexus-sub-status', 'active')
      localStorage.setItem('nexus-sub-expiry', '2099-12-31T00:00:00.000Z')
      setGoogleEmail(ADMIN_EMAIL)
      setLoginError('')
      setStep(4)
    } else {
      setLoginError('이메일 또는 비밀번호가 올바르지 않습니다.')
    }
  }

  const handleGoogleLogin = async () => {
    if (!SUPABASE_URL || SUPABASE_URL.includes('placeholder')) {
      const trialExpiry = new Date(Date.now() + 3 * 24 * 60 * 60 * 1000).toISOString()
      const demoEmail = 'user@gmail.com'
      localStorage.setItem('nexus-user-email', demoEmail)
      localStorage.setItem('nexus-sub-status', 'trial')
      localStorage.setItem('nexus-sub-expiry', trialExpiry)
      setGoogleEmail(demoEmail)
      setStep(4)
      return
    }
    setGoogleLoading(true)
    try {
      const hint = localStorage.getItem('nexus-user-email') ?? undefined
      await signInWithGoogle(hint)
    } catch (e) {
      console.warn('Google OAuth failed, starting trial:', e)
      const trialExpiry = new Date(Date.now() + 3 * 24 * 60 * 60 * 1000).toISOString()
      const demoEmail = 'user@gmail.com'
      localStorage.setItem('nexus-user-email', demoEmail)
      localStorage.setItem('nexus-sub-status', 'trial')
      localStorage.setItem('nexus-sub-expiry', trialExpiry)
      setGoogleEmail(demoEmail)
      setStep(4)
    } finally {
      setGoogleLoading(false)
    }
  }

  const handleComplete = async (email?: string) => {
    if (openaiKey.trim()) localStorage.setItem('nexus-openai-key', openaiKey.trim())
    localStorage.setItem('nexus-persona-id', selectedJobId)

    const resolvedEmail = email ?? googleEmail
    if (resolvedEmail) localStorage.setItem('nexus-user-email', resolvedEmail)

    // If Pro or Team plan selected, open checkout
    if (selectedPlan !== 'free' && resolvedEmail) {
      const priceId = selectedPlan === 'team' ? PADDLE_PRICES.team_5 : PADDLE_PRICES.pro_monthly
      openCheckout(priceId, resolvedEmail).catch(console.warn)
    }

    try {
      const BASE = 'http://127.0.0.1:17891'
      fetch(`${BASE}/api/persona/set`, {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ id: selectedJobId }),
      }).catch(() => {})
      fetch(`${BASE}/api/settings/lang`, {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ lang: isEn ? 'en' : 'ko' }),
      }).catch(() => {})
      const configPayload: Record<string, string> = {}
      if (openaiKey.trim()) configPayload['claude_key'] = openaiKey.trim()
      if (Object.keys(configPayload).length > 0) {
        fetch(`${BASE}/api/llm/config`, {
          method: 'POST', headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(configPayload),
        }).catch(() => {})
      }
      fetch(`${BASE}/api/setup/status`)
        .then(r => r.json())
        .then((status: unknown) => { localStorage.setItem('nexus-setup-status', JSON.stringify(status)) })
        .catch(() => {})
    } catch {}

    onComplete({
      assistantName: assistantName || (isEn ? 'Nexus' : '넥서스'),
      userName: userName || userInput || (isEn ? 'Boss' : '주인님'),
      glbUrl: selectedStyle.glbUrl,
      previewUrl: null,
      primaryColor: selectedStyle.primaryColor,
      accentColor:  selectedStyle.accentColor,
      preset:       styleId as CharacterPreset,
      styleId,
      ttsVoice: selectedStyle.ttsVoice,
    })
  }

  /* ── 공통 스타일 ── */
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
        <span style={{ fontSize: 11, color: 'rgba(255,255,255,0.35)', letterSpacing: '0.06em' }}>NEXUS SETUP</span>
        <span style={{ fontSize: 11, color: 'rgba(255,255,255,0.35)' }}>{step + 1} / {STEPS_TOTAL}</span>
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
        cursor: 'pointer', letterSpacing: '0.02em', transition: 'all 0.15s',
      }}
      onMouseEnter={e => { e.currentTarget.style.background = 'rgba(255,255,255,0.08)'; e.currentTarget.style.color = 'rgba(255,255,255,0.75)' }}
      onMouseLeave={e => { e.currentTarget.style.background = 'rgba(255,255,255,0.04)'; e.currentTarget.style.color = 'rgba(255,255,255,0.45)' }}
    >
      {label}
    </button>
  )

  const nextBtn = (onClick: () => void, label = '다음', disabled = false) => (
    <button
      onClick={onClick}
      disabled={disabled}
      style={{
        width: '100%', padding: '14px 0',
        background: disabled
          ? 'rgba(255,255,255,0.08)'
          : `linear-gradient(135deg, ${selectedStyle.primaryColor}, ${selectedStyle.accentColor})`,
        border: 'none', borderRadius: 14,
        color: disabled ? 'rgba(255,255,255,0.3)' : 'white',
        fontSize: 15, fontWeight: 700,
        cursor: disabled ? 'not-allowed' : 'pointer', letterSpacing: '0.03em',
        boxShadow: disabled ? 'none' : `0 4px 24px ${selectedStyle.primaryColor}55`,
        transition: 'opacity 0.15s',
      }}
      onMouseEnter={e => { if (!disabled) e.currentTarget.style.opacity = '0.88' }}
      onMouseLeave={e => { e.currentTarget.style.opacity = '1' }}
      onMouseDown={e => { if (!disabled) e.currentTarget.style.transform = 'scale(0.97)' }}
      onMouseUp={e => { e.currentTarget.style.transform = 'scale(1)' }}
    >
      {label}
    </button>
  )

  return (
    <div style={overlay}>
      {/* Background gradient */}
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

        {/* ══════════════════════════════════════════════
            Step 0: 직업군 선택
        ══════════════════════════════════════════════ */}
        {step === 0 && (
          <motion.div
            key="step0-job"
            initial={{ opacity: 0, scale: 0.96 }}
            animate={{ opacity: 1, scale: 1 }}
            exit={{ opacity: 0, scale: 0.94 }}
            style={{ ...card, maxWidth: 620 }}
          >
            {/* Language toggle */}
            <div style={{ position: 'absolute', top: 20, right: 24, display: 'flex', gap: 4 }}>
              {(['ko', 'en'] as const).map(l => (
                <button
                  key={l}
                  onClick={() => {
                    setLang(l)
                    setUserLang(l)
                    setName(l === 'en' ? 'Nexus' : '넥서스')
                    setNameInput(l === 'en' ? 'Nexus' : '넥서스')
                    setUserName(l === 'en' ? 'Boss' : '주인님')
                  }}
                  style={{
                    padding: '3px 9px', borderRadius: 8,
                    background: lang === l ? 'rgba(255,255,255,0.18)' : 'rgba(255,255,255,0.05)',
                    border: lang === l ? '1px solid rgba(255,255,255,0.35)' : '1px solid rgba(255,255,255,0.1)',
                    color: lang === l ? 'white' : 'rgba(255,255,255,0.4)',
                    fontSize: 11, fontWeight: 700, cursor: 'pointer', transition: 'all 0.15s',
                  }}
                >
                  {l === 'ko' ? '🇰🇷 KO' : '🇺🇸 EN'}
                </button>
              ))}
            </div>

            <div style={{ marginBottom: 24, paddingRight: 80 }}>
              <div style={{ fontSize: 11, letterSpacing: '0.12em', color: selectedStyle.primaryColor, marginBottom: 8, fontWeight: 600 }}>
                NEXUS AI
              </div>
              <h2 style={{ fontSize: 22, fontWeight: 800, color: 'white', marginBottom: 6 }}>
                {isEn ? 'What do you do?' : '어떤 일을 하세요?'}
              </h2>
              <p style={{ fontSize: 13, color: 'rgba(255,255,255,0.42)', lineHeight: 1.6 }}>
                {isEn ? 'AI responses & workflows will be optimized for your role.' : 'AI 응답과 워크플로우가 내 직업에 맞게 최적화됩니다.'}
              </p>
            </div>

            {/* 10-persona grid */}
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10, marginBottom: 20 }}>
              {JOB_PERSONAS.map(p => {
                const isSelected = selectedJobId === p.id
                const isPro = !!p.proFeature
                return (
                  <button
                    key={p.id}
                    onClick={async () => {
                      if (jobSelected) return
                      setSelectedJobId(p.id)
                      setJobSelected(true)
                      await sleep(800)
                      setJobSelected(false)
                      setDemoStarted(false)
                      setDemoResult('')
                      setDemoTyping('')
                      setDemoThinkStep('')
                      setDemoInputTyping('')
                      setDemoRunning(false)
                      setStep(1)
                    }}
                    style={{
                      background: isSelected
                        ? `linear-gradient(135deg, ${p.color}28, ${p.color}12)`
                        : 'rgba(255,255,255,0.04)',
                      border: isSelected
                        ? `1.5px solid ${p.color}88`
                        : '1.5px solid rgba(255,255,255,0.08)',
                      borderRadius: 14, padding: '14px',
                      cursor: 'pointer', textAlign: 'left',
                      transition: 'all 0.18s', position: 'relative',
                    } as React.CSSProperties}
                  >
                    {/* PRO badge */}
                    {isPro && (
                      <div style={{
                        position: 'absolute', top: 7, right: 7,
                        background: p.color,
                        color: 'white', fontSize: 8, fontWeight: 800,
                        padding: '2px 5px', borderRadius: 4, letterSpacing: '0.05em',
                      }}>PRO</div>
                    )}

                    <div style={{ fontSize: 20, marginBottom: 5 }}>{p.emoji}</div>
                    <div style={{
                      fontSize: 12, fontWeight: 700,
                      color: isSelected ? 'white' : 'rgba(255,255,255,0.75)',
                      marginBottom: 3, lineHeight: 1.3,
                    }}>{p.name}</div>
                    <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.35)', lineHeight: 1.4 }}>
                      {p.desc}
                    </div>
                    {isSelected && (
                      <motion.div
                        initial={{ opacity: 0, scale: 0.7 }}
                        animate={{ opacity: 1, scale: 1 }}
                        style={{
                          position: 'absolute', bottom: 7, right: 7,
                          fontSize: 10, color: p.color, fontWeight: 700,
                        }}
                      >
                        {isEn ? '✓ Selected' : '선택됨 ✓'}
                      </motion.div>
                    )}
                  </button>
                )
              })}
            </div>

            <p style={{ fontSize: 11, color: 'rgba(255,255,255,0.25)', textAlign: 'center' }}>
              {isEn ? 'Click to select and continue' : '선택하면 자동으로 다음으로 넘어갑니다'}
            </p>
          </motion.div>
        )}

        {/* ══════════════════════════════════════════════
            Step 1: 직업군 전용 WOW 데모
        ══════════════════════════════════════════════ */}
        {step === 1 && (
          <motion.div
            key="step1-demo"
            initial={{ opacity: 0, y: 30 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            style={{
              width: '100%', maxWidth: 620,
              background: 'rgba(10,10,20,0.97)',
              border: '1px solid rgba(255,255,255,0.08)',
              borderRadius: 24,
              overflow: 'hidden',
              boxShadow: `0 0 80px ${jobColor}22, 0 32px 80px rgba(0,0,0,0.6)`,
              backdropFilter: 'blur(24px)',
              position: 'relative',
            }}
          >
            {progressBar && (
              <div style={{ padding: '20px 24px 0' }}>{progressBar}</div>
            )}

            {/* Header */}
            <div style={{
              padding: '0 24px 14px',
              display: 'flex', alignItems: 'center', gap: 10,
            }}>
              <div style={{
                width: 36, height: 36, borderRadius: 10, flexShrink: 0,
                background: `${jobColor}22`, border: `1px solid ${jobColor}44`,
                display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 18,
              }}>
                {selectedJob.emoji}
              </div>
              <div>
                <div style={{ fontSize: 14, fontWeight: 700, color: 'white' }}>
                  {isEn ? `${selectedJob.name} — Live Demo` : `${selectedJob.name} 전용 데모`}
                </div>
                <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.35)' }}>
                  {isEn ? 'Nexus executes this in real-time' : 'Nexus가 실시간으로 실행합니다'}
                </div>
              </div>
            </div>

            {/* Demo chat area */}
            <div style={{
              margin: '0 24px',
              background: 'rgba(255,255,255,0.03)',
              border: '1px solid rgba(255,255,255,0.07)',
              borderRadius: 16, overflow: 'hidden',
            }}>
              {/* Fake input */}
              <div style={{
                padding: '12px 16px',
                borderBottom: '1px solid rgba(255,255,255,0.06)',
                display: 'flex', alignItems: 'center', gap: 10,
              }}>
                <div style={{
                  flex: 1, padding: '8px 12px',
                  background: 'rgba(255,255,255,0.05)',
                  border: `1px solid ${demoRunning ? jobColor + '55' : 'rgba(255,255,255,0.1)'}`,
                  borderRadius: 10, fontSize: 13,
                  color: demoInputTyping ? 'white' : 'rgba(255,255,255,0.25)',
                  transition: 'border-color 0.2s', minHeight: 20,
                }}>
                  {demoInputTyping || (isEn ? 'Asking Nexus...' : 'Nexus에게 묻는 중...')}
                  {demoInputTyping && <span style={{ animation: 'blink 0.6s step-end infinite', opacity: 0.8 }}>|</span>}
                </div>
                <div style={{
                  width: 34, height: 34, borderRadius: 9, flexShrink: 0,
                  background: demoRunning
                    ? `linear-gradient(135deg, ${jobColor}, ${jobColor}99)`
                    : 'rgba(255,255,255,0.08)',
                  display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 15,
                }}>
                  {demoRunning ? '⚡' : '↑'}
                </div>
              </div>

              {/* Result area */}
              <div style={{ minHeight: 200, maxHeight: 260, overflowY: 'auto', padding: '16px', scrollbarWidth: 'none' } as React.CSSProperties}>
                {demoThinkStep && (
                  <motion.div
                    key={demoThinkStep}
                    initial={{ opacity: 0, x: -6 }}
                    animate={{ opacity: 1, x: 0 }}
                    style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 12 }}
                  >
                    <div style={{
                      fontSize: 11, color: jobColor,
                      background: `${jobColor}12`,
                      border: `1px solid ${jobColor}33`,
                      padding: '6px 12px', borderRadius: 20,
                    }}>{demoThinkStep}</div>
                  </motion.div>
                )}
                {(demoTyping || demoResult) && (
                  <div style={{
                    padding: '10px 14px',
                    borderRadius: '4px 16px 16px 16px',
                    background: 'rgba(255,255,255,0.06)',
                    border: '1px solid rgba(255,255,255,0.1)',
                    fontSize: 12, color: 'rgba(255,255,255,0.92)',
                    lineHeight: 1.7, whiteSpace: 'pre-wrap', fontFamily: 'monospace',
                  }}>
                    {demoTyping || demoResult}
                    {demoTyping && <span style={{ animation: 'blink 0.8s step-end infinite', opacity: 0.7 }}>▌</span>}
                  </div>
                )}
                {!demoRunning && !demoResult && !demoThinkStep && (
                  <div style={{ color: 'rgba(255,255,255,0.2)', fontSize: 12, textAlign: 'center', paddingTop: 60 }}>
                    {isEn ? 'Starting demo...' : '데모 시작 중...'}
                  </div>
                )}
                <div ref={demoEndRef} />
              </div>
            </div>

            {/* Pro hint */}
            {(() => {
              const demos = isEn ? JOB_DEMOS_EN : JOB_DEMOS
              const demo = demos[selectedJobId]
              return demo?.proHint && demoResult ? (
                <motion.div
                  initial={{ opacity: 0, y: 8 }}
                  animate={{ opacity: 1, y: 0 }}
                  style={{
                    margin: '12px 24px 0',
                    padding: '10px 14px',
                    background: `${jobColor}15`,
                    border: `1px solid ${jobColor}44`,
                    borderRadius: 12, fontSize: 12,
                    color: jobColor, fontWeight: 600,
                    boxShadow: `0 0 20px ${jobColor}22`,
                  }}
                >
                  ✨ {demo.proHint}
                </motion.div>
              ) : null
            })()}

            {/* CTA */}
            <div style={{ padding: '16px 24px 24px', display: 'flex', flexDirection: 'column', gap: 8 }}>
              {nextBtn(
                () => setStep(2),
                isEn ? 'Choose My Plan →' : '플랜 선택하기 →',
                demoRunning || !!demoTyping,
              )}
              {backBtn(() => { setJobSelected(false); setStep(0) }, isEn ? '← Change Job' : '← 직업 바꾸기')}
            </div>
          </motion.div>
        )}

        {/* ══════════════════════════════════════════════
            Step 2: 플랜 선택
        ══════════════════════════════════════════════ */}
        {step === 2 && (
          <motion.div
            key="step2-plan"
            initial={{ opacity: 0, y: 30 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            style={{ ...card, maxWidth: 620 }}
          >
            {progressBar}

            <div style={{ textAlign: 'center', marginBottom: 24 }}>
              <div style={{ fontSize: 36, marginBottom: 10 }}>💎</div>
              <h2 style={{ fontSize: 22, fontWeight: 800, color: 'white', marginBottom: 6 }}>
                {isEn ? 'Choose Your Plan' : '딱 맞는 플랜을 선택하세요'}
              </h2>
              <p style={{ fontSize: 13, color: 'rgba(255,255,255,0.4)', lineHeight: 1.6 }}>
                {isEn ? 'Change anytime · No credit card required to start' : '언제든 변경 가능 · 카드 등록 불필요'}
              </p>
            </div>

            {/* Pro hint for specific jobs */}
            {selectedJob.proFeature && (
              <div style={{
                marginBottom: 16, padding: '10px 14px',
                background: `${jobColor}18`,
                border: `1px solid ${jobColor}44`,
                borderRadius: 12, fontSize: 12,
                color: jobColor, fontWeight: 600,
              }}>
                ⚠️ {selectedJob.proLabel}
              </div>
            )}

            {/* Plan cards */}
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 10, marginBottom: 20 }}>
              {/* Free */}
              {(['free', 'pro', 'team'] as const).map(plan => {
                const isSelected = selectedPlan === plan
                const isPro = plan === 'pro'
                const isTeam = plan === 'team'
                const planColor = isPro ? '#a855f7' : isTeam ? '#0ea5e9' : '#6b7280'
                const planLabel = isPro ? (isEn ? '✨ PRO' : '✨ PRO') : isTeam ? (isEn ? 'TEAM' : 'TEAM') : (isEn ? 'FREE' : 'FREE')
                const planPrice = isPro ? '$19/mo' : isTeam ? '$49/mo' : (isEn ? 'Free' : '무료')
                const planFeats = isPro
                  ? (isEn ? ['Pro features unlimited', '+ Marketplace'] : ['Pro 기능 무제한', '+ 마켓플레이스'])
                  : isTeam
                  ? (isEn ? ['Team sharing + API', '+ Brand customization'] : ['팀 공유 + API', '+ 기업 브랜딩'])
                  : (isEn ? ['Basic features', 'Daily limit'] : ['기본 기능', '일일 제한'])

                return (
                  <button
                    key={plan}
                    onClick={() => setSelectedPlan(plan)}
                    style={{
                      background: isSelected
                        ? `linear-gradient(135deg, ${planColor}28, ${planColor}12)`
                        : 'rgba(255,255,255,0.04)',
                      border: isSelected
                        ? `2px solid ${planColor}88`
                        : '1.5px solid rgba(255,255,255,0.08)',
                      borderRadius: 16, padding: '16px 12px',
                      cursor: 'pointer', textAlign: 'center',
                      transition: 'all 0.18s', position: 'relative',
                    } as React.CSSProperties}
                  >
                    {isPro && (
                      <div style={{
                        position: 'absolute', top: -1, left: '50%', transform: 'translateX(-50%)',
                        background: planColor, color: 'white',
                        fontSize: 9, fontWeight: 800, padding: '2px 8px',
                        borderRadius: '0 0 6px 6px', letterSpacing: '0.05em',
                      }}>RECOMMENDED</div>
                    )}
                    <div style={{
                      fontSize: 13, fontWeight: 800,
                      color: isSelected ? planColor : 'rgba(255,255,255,0.6)',
                      marginBottom: 6, marginTop: isPro ? 10 : 0,
                    }}>{planLabel}</div>
                    <div style={{
                      fontSize: 16, fontWeight: 800, color: 'white', marginBottom: 10,
                    }}>{planPrice}</div>
                    {planFeats.map(f => (
                      <div key={f} style={{
                        fontSize: 10, color: 'rgba(255,255,255,0.5)', lineHeight: 1.6,
                      }}>{f}</div>
                    ))}
                    {isSelected && (
                      <div style={{
                        marginTop: 8, fontSize: 11, color: planColor, fontWeight: 700,
                      }}>✓ {isEn ? 'Selected' : '선택됨'}</div>
                    )}
                  </button>
                )
              })}
            </div>

            <p style={{ fontSize: 11, color: 'rgba(255,255,255,0.25)', textAlign: 'center', marginBottom: 16 }}>
              {selectedPlan !== 'free'
                ? (isEn ? `Payment will start after Google login` : `Google 로그인 후 결제가 시작됩니다`)
                : (isEn ? `Start for free — upgrade anytime` : `무료로 시작 — 언제든 업그레이드 가능`)}
            </p>

            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              {nextBtn(() => setStep(3), isEn ? 'Continue →' : '다음 →')}
              {backBtn(() => setStep(1), isEn ? '← Back to Demo' : '← 데모로 돌아가기')}
            </div>
          </motion.div>
        )}

        {/* ══════════════════════════════════════════════
            Step 3: Google Login
        ══════════════════════════════════════════════ */}
        {step === 3 && (
          <motion.div
            key="step3-login"
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
                {isEn
                  ? <>Sign in with Google and your<br />3-day free trial starts automatically.</>
                  : <>구글 계정으로 로그인하면<br />3일 무료 체험이 자동으로 시작됩니다.</>}
              </p>
            </div>

            {!googleEmail ? (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                {/* Close button */}
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
                    display: 'flex', alignItems: 'center', justifyContent: 'center', lineHeight: 1,
                  }}
                >✕</button>

                {/* Google button */}
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

                {/* Admin login toggle */}
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
                      type="email" placeholder="Email" value={loginEmail}
                      onChange={e => setLoginEmail(e.target.value)}
                      style={{
                        width: '100%', padding: '10px 14px', boxSizing: 'border-box',
                        background: 'rgba(255,255,255,0.05)', border: '1px solid rgba(255,255,255,0.12)',
                        borderRadius: 10, color: 'white', fontSize: 13, outline: 'none',
                      } as React.CSSProperties}
                    />
                    <input
                      type="password" placeholder="Password" value={loginPassword}
                      onChange={e => setLoginPassword(e.target.value)}
                      onKeyDown={e => e.key === 'Enter' && handleAdminLogin()}
                      style={{
                        width: '100%', padding: '10px 14px', boxSizing: 'border-box',
                        background: 'rgba(255,255,255,0.05)', border: '1px solid rgba(255,255,255,0.12)',
                        borderRadius: 10, color: 'white', fontSize: 13, outline: 'none',
                      } as React.CSSProperties}
                    />
                    {loginError && <p style={{ fontSize: 11, color: '#f87171', margin: 0 }}>{loginError}</p>}
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
                <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.4)', marginTop: 4 }}>
                  {isEn ? '3-day free trial ready to start' : '3일 무료 체험 시작 준비 완료'}
                </div>
              </div>
            )}

            {/* Benefits */}
            <div style={{
              background: `${selectedStyle.primaryColor}10`,
              border: `1px solid ${selectedStyle.primaryColor}25`,
              borderRadius: 12, padding: '14px 18px', marginTop: 16, fontSize: 12,
              color: 'rgba(255,255,255,0.6)', lineHeight: 2,
            }}>
              {(isEn
                ? ['✦ Unlimited access to all AI features', '✦ Deep Search · Real-time web search', '✦ Auto updates', '✦ Early Bird $12.99/mo after 3 days · Cancel anytime']
                : ['✦ 모든 AI 기능 무제한 사용', '✦ 딥서치 · 실시간 웹 검색', '✦ 자동 업데이트', '✦ 3일 후 Early Bird 월 14,900원 · 언제든 해지 가능']
              ).map(t => <div key={t}>{t}</div>)}
            </div>

            <div style={{ display: 'flex', flexDirection: 'column', gap: 8, marginTop: 16 }}>
              {googleEmail
                ? nextBtn(() => setStep(4), isEn ? 'Continue →' : '다음 →')
                : <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.25)', textAlign: 'center' }}>
                    {isEn ? 'Sign in with Google to continue' : '구글 로그인 후 계속 진행할 수 있습니다'}
                  </div>
              }
              {backBtn(() => setStep(2), isEn ? '← Change Plan' : '← 플랜 변경')}
            </div>
          </motion.div>
        )}

        {/* ══════════════════════════════════════════════
            Step 4: Avatar + Name + Nickname (combined)
        ══════════════════════════════════════════════ */}
        {step === 4 && (
          <motion.div
            key="step4-profile"
            initial={{ opacity: 0, y: 30 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            style={{ ...card, maxWidth: 620 }}
          >
            {progressBar}

            <h2 style={{ fontSize: 20, fontWeight: 800, color: 'white', marginBottom: 6 }}>
              {isEn ? 'Personalize Your AI' : 'AI를 나에게 맞게 설정하세요'}
            </h2>
            <p style={{ fontSize: 12, color: 'rgba(255,255,255,0.4)', marginBottom: 20, lineHeight: 1.6 }}>
              {isEn ? 'Choose an avatar style and set names.' : '아바타 스타일을 고르고 이름을 설정하세요.'}
            </p>

            {/* Avatar grid (2x2) */}
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10, marginBottom: 20 }}>
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
                      borderRadius: 16, padding: '14px',
                      cursor: 'pointer', textAlign: 'left', transition: 'all 0.22s',
                      position: 'relative', overflow: 'hidden',
                    } as React.CSSProperties}
                  >
                    <div style={{ height: 100, marginBottom: 8, overflow: 'hidden', borderRadius: 8 }}>
                      <Avatar3D
                        emotion={selected ? 'happy' : 'neutral'}
                        speaking={false} listening={false}
                        glbUrl={s.glbUrl}
                        primaryColor={s.primaryColor}
                        accentColor={s.accentColor}
                        preset={s.id as CharacterPreset}
                        width="100%" height={100}
                        preview scale={0.55} characterOffsetY={-0.6} quality="balanced"
                      />
                    </div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                      <span style={{ fontSize: 14 }}>{s.previewEmoji}</span>
                      <span style={{ fontSize: 12, fontWeight: 700, color: 'white' }}>{s.name}</span>
                      {selected && (
                        <span style={{
                          marginLeft: 'auto', fontSize: 9,
                          background: `${s.primaryColor}44`, color: s.primaryColor,
                          padding: '2px 6px', borderRadius: 20, fontWeight: 700,
                        }}>✓</span>
                      )}
                    </div>
                  </button>
                )
              })}
            </div>

            {/* Name input */}
            <div style={{ marginBottom: 14 }}>
              <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.4)', marginBottom: 6, fontWeight: 600, letterSpacing: '0.05em' }}>
                {isEn ? 'ASSISTANT NAME' : '비서 이름'}
              </div>
              <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap', marginBottom: 8 }}>
                {SUGGESTED_NAMES.slice(0, 4).map(n => (
                  <button key={n}
                    onClick={() => { setNameInput(n); setName(n) }}
                    style={{
                      padding: '4px 12px', borderRadius: 20, cursor: 'pointer',
                      background: nameInput === n ? `${selectedStyle.primaryColor}44` : 'rgba(255,255,255,0.06)',
                      border: nameInput === n ? `1px solid ${selectedStyle.primaryColor}88` : '1px solid rgba(255,255,255,0.08)',
                      color: nameInput === n ? selectedStyle.primaryColor : 'rgba(255,255,255,0.6)',
                      fontSize: 12, fontWeight: 600, transition: 'all 0.18s',
                    } as React.CSSProperties}
                  >{n}</button>
                ))}
              </div>
              <input
                value={nameInput}
                onChange={e => { setNameInput(e.target.value); setName(e.target.value) }}
                placeholder={isEn ? 'Custom name...' : '직접 입력...'}
                style={{
                  width: '100%', padding: '10px 14px',
                  background: 'rgba(255,255,255,0.05)',
                  border: `1px solid ${selectedStyle.primaryColor}44`,
                  borderRadius: 10, color: 'white', fontSize: 13,
                  outline: 'none', boxSizing: 'border-box',
                } as React.CSSProperties}
              />
            </div>

            {/* User nickname */}
            <div style={{ marginBottom: 20 }}>
              <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.4)', marginBottom: 6, fontWeight: 600, letterSpacing: '0.05em' }}>
                {isEn ? 'YOUR NICKNAME' : '내 호칭'}
              </div>
              <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap', marginBottom: 8 }}>
                {USER_NAMES.map(n => (
                  <button key={n}
                    onClick={() => { setUserInput(n); setUserName(n) }}
                    style={{
                      padding: '4px 12px', borderRadius: 20, cursor: 'pointer',
                      background: userInput === n ? `${selectedStyle.primaryColor}44` : 'rgba(255,255,255,0.06)',
                      border: userInput === n ? `1px solid ${selectedStyle.primaryColor}88` : '1px solid rgba(255,255,255,0.08)',
                      color: userInput === n ? selectedStyle.primaryColor : 'rgba(255,255,255,0.6)',
                      fontSize: 12, fontWeight: 600, transition: 'all 0.18s',
                    } as React.CSSProperties}
                  >{n}</button>
                ))}
              </div>
              <input
                value={userInput}
                onChange={e => { setUserInput(e.target.value); setUserName(e.target.value) }}
                placeholder={isEn ? 'Custom nickname...' : '직접 입력...'}
                style={{
                  width: '100%', padding: '10px 14px',
                  background: 'rgba(255,255,255,0.05)',
                  border: `1px solid ${selectedStyle.primaryColor}44`,
                  borderRadius: 10, color: 'white', fontSize: 13,
                  outline: 'none', boxSizing: 'border-box',
                } as React.CSSProperties}
              />
            </div>

            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              {nextBtn(
                () => {
                  setName(nameInput.trim() || (isEn ? 'Nexus' : '넥서스'))
                  setUserName(userInput.trim() || (isEn ? 'Boss' : '주인님'))
                  setStep(5)
                },
                isEn ? 'Next →' : '다음 →',
              )}
              {backBtn(() => setStep(3), isEn ? '← Back' : '← 이전')}
            </div>
          </motion.div>
        )}

        {/* ══════════════════════════════════════════════
            Step 5: Complete (Google Calendar + finish)
        ══════════════════════════════════════════════ */}
        {step === 5 && (
          <motion.div
            key="step5-complete"
            initial={{ opacity: 0, y: 30 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            style={card}
          >
            {progressBar}

            <div style={{ textAlign: 'center', marginBottom: 24 }}>
              <div style={{ fontSize: 40, marginBottom: 12 }}>🔗</div>
              <h2 style={{ fontSize: 20, fontWeight: 800, color: 'white', marginBottom: 8 }}>
                {isEn ? 'Connect Google Services' : 'Google 서비스 연동'}
              </h2>
              <p style={{ fontSize: 12, color: 'rgba(255,255,255,0.4)', lineHeight: 1.6 }}>
                {isEn
                  ? 'Connect Calendar & Gmail so Nexus can manage your schedule and emails.'
                  : 'Google 캘린더와 Gmail을 연동하면 일정과 메일을 관리할 수 있어요.'}
              </p>
            </div>

            {/* Google Calendar + Gmail */}
            <div style={{ display: 'flex', flexDirection: 'column', gap: 12, marginBottom: 20 }}>
              <div style={{
                background: gcalConnected ? 'rgba(34,197,94,0.1)' : 'rgba(255,255,255,0.04)',
                border: `1px solid ${gcalConnected ? 'rgba(34,197,94,0.4)' : 'rgba(255,255,255,0.1)'}`,
                borderRadius: 14, padding: '16px 18px',
                display: 'flex', alignItems: 'center', gap: 14,
              }}>
                <div style={{
                  width: 42, height: 42, borderRadius: 12, flexShrink: 0,
                  background: 'rgba(66,133,244,0.15)', border: '1px solid rgba(66,133,244,0.3)',
                  display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 22,
                }}>📅</div>
                <div style={{ flex: 1 }}>
                  <div style={{ fontWeight: 700, fontSize: 13, color: 'white', marginBottom: 2 }}>
                    Google Calendar & Gmail
                  </div>
                  <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.4)' }}>
                    {gcalConnected
                      ? (gcalEmail || (isEn ? 'Connected' : '연결됨'))
                      : (isEn ? 'Schedule management · Email read/send' : '일정 관리 · 메일 읽기/발송')}
                  </div>
                </div>
                {gcalConnected ? (
                  <div style={{ fontSize: 22 }}>✅</div>
                ) : (
                  <button
                    onClick={async () => {
                      setGcalLoading(true)
                      try {
                        const BASE = 'http://127.0.0.1:17891'
                        const res = await fetch(`${BASE}/api/calendar/google/auth`)
                        const data = await res.json()
                        if (data.url) {
                          const { open } = await import('@tauri-apps/plugin-shell')
                          await open(data.url)
                          let tries = 0
                          const poll = setInterval(async () => {
                            tries++
                            try {
                              const s = await fetch(`${BASE}/api/calendar/google/status`)
                              const st = await s.json()
                              if (st.connected) {
                                setGcalConnected(true)
                                setGcalEmail(st.email || '')
                                clearInterval(poll)
                              }
                            } catch {}
                            if (tries > 24) clearInterval(poll)
                          }, 5000)
                        } else {
                          alert(isEn ? 'Google OAuth not configured yet.' : 'Google OAuth가 아직 설정되지 않았습니다.')
                        }
                      } catch {}
                      setGcalLoading(false)
                    }}
                    disabled={gcalLoading}
                    style={{
                      padding: '7px 14px', borderRadius: 9, border: 'none', cursor: 'pointer',
                      background: 'rgba(66,133,244,0.8)', color: 'white',
                      fontSize: 12, fontWeight: 700, flexShrink: 0,
                      opacity: gcalLoading ? 0.6 : 1,
                    }}
                  >
                    {gcalLoading ? '...' : (isEn ? 'Connect' : '연동')}
                  </button>
                )}
              </div>

              {!gcalConnected && (
                <div style={{
                  padding: '10px 14px', background: 'rgba(255,255,255,0.02)',
                  border: '1px solid rgba(255,255,255,0.06)', borderRadius: 10,
                  fontSize: 11, color: 'rgba(255,255,255,0.3)', lineHeight: 1.6,
                }}>
                  {isEn
                    ? "💡 Can't connect now? Set it up later in Settings → Email/Calendar."
                    : '💡 지금 연동이 어려우면 나중에 설정 → 이메일/캘린더에서 하실 수 있어요.'}
                </div>
              )}
            </div>

            {/* Summary */}
            <div style={{
              background: 'rgba(255,255,255,0.03)', border: '1px solid rgba(255,255,255,0.08)',
              borderRadius: 12, padding: '14px 18px', marginBottom: 16, fontSize: 13,
              color: 'rgba(255,255,255,0.7)', lineHeight: 1.8,
            }}>
              <div>{isEn ? 'Job' : '직업'}: <strong style={{ color: 'white' }}>{selectedJob.emoji} {selectedJob.name}</strong></div>
              <div>{isEn ? 'Plan' : '플랜'}: <strong style={{ color: selectedPlan === 'pro' ? '#a855f7' : selectedPlan === 'team' ? '#0ea5e9' : '#6b7280' }}>
                {selectedPlan === 'pro' ? 'Pro $19/mo' : selectedPlan === 'team' ? 'Team $49/mo' : (isEn ? 'Free' : '무료')}
              </strong></div>
              <div>{isEn ? 'Assistant' : '비서'}: <strong style={{ color: 'white' }}>{assistantName}</strong></div>
              <div>{isEn ? 'Your name' : '호칭'}: <strong style={{ color: 'white' }}>{userInput || (isEn ? 'Boss' : '주인님')}</strong></div>
              <div>{isEn ? 'Character' : '캐릭터'}: <strong style={{ color: selectedStyle.primaryColor }}>{selectedStyle.name}</strong></div>
            </div>

            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              {nextBtn(
                () => void handleComplete(),
                isEn
                  ? (gcalConnected ? `Start with ${assistantName} ✦` : `Skip & Start ✦`)
                  : (gcalConnected ? `${assistantName} 시작하기 ✦` : `건너뛰고 시작하기 ✦`),
              )}
              {backBtn(() => setStep(4), isEn ? '← Back' : '← 이전')}
            </div>
          </motion.div>
        )}

      </AnimatePresence>
    </div>
  )
}
