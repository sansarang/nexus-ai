/**
 * OnboardingFlow v8 — Demo-First Marketing Optimized
 *
 * Step 0: 범용 인터랙티브 데모 (직업 무관 WOW)
 * Step 1: 직업군 선택 (10개) — "어떤 일을 하세요?"
 * Step 2: 직업군 전용 WOW 데모
 * Step 3: 플랜 선택 (Free / Pro $19 / Team $49)
 * Step 4: Google Login
 * Step 5: Avatar + Name + Nickname (combined)
 * Step 6: Complete
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

// ─── Generic demo sequences (Step 0) ────────────────────────────────────────
const GENERIC_DEMOS_KO = [
  {
    query: '오늘 주요 뉴스 3가지 요약해줘',
    steps: ['🔍 실시간 뉴스 수집 중...', '📊 중요도 분석 중...', '✍️ 요약 생성 중...'],
    result: '📰 오늘의 주요 뉴스\n\n1️⃣ 삼성전자, HBM4 양산 돌입\n   엔비디아·AMD 공급 계약 확정 — 주가 +3.2%\n\n2️⃣ 한국은행 기준금리 동결 (3.25%)\n   "물가 안정세 확인 후 인하 검토"\n\n3️⃣ 카카오, AI 신사업 발표\n   자체 LLM 기반 업무 자동화 플랫폼 출시 예정',
  },
  {
    query: '내일 서울 날씨 알려줘',
    steps: ['🌐 기상청 데이터 수집 중...', '🧮 예보 분석 중...'],
    result: '🌤️ 내일 서울 날씨\n\n최고 23°C / 최저 14°C\n오전: 맑음 ☀️\n오후: 구름 조금 🌤️\n강수 확률: 10%\n\n💡 얇은 겉옷 챙기시면 좋을 것 같아요!',
  },
  {
    query: '업무 이메일 초안 작성해줘 (미팅 일정 조율)',
    steps: ['📋 요청 분석 중...', '✍️ 이메일 초안 작성 중...'],
    result: '📧 이메일 초안 완성\n\n제목: [미팅 일정 조율 요청] 다음 주 회의 관련\n\n안녕하세요, 홍길동 과장님\n\n다음 주 프로젝트 논의를 위한 미팅을 제안드립니다.\n\n📅 가능 일정: 5/27(화) 14:00 또는 5/28(수) 10:00\n⏱️ 소요 시간: 약 1시간\n📍 장소: 화상 회의 (링크 별도 공유)\n\n편하신 시간으로 회신 부탁드립니다.\n\n감사합니다.',
  },
]
const GENERIC_DEMOS_EN = [
  {
    query: 'Summarize today\'s top 3 news',
    steps: ['🔍 Fetching live news...', '📊 Ranking by importance...', '✍️ Generating summary...'],
    result: '📰 Today\'s Top News\n\n1️⃣ Samsung starts HBM4 mass production\n   Supply deals with NVIDIA & AMD confirmed — stock +3.2%\n\n2️⃣ Fed holds interest rates steady\n   "Will monitor inflation before cutting"\n\n3️⃣ OpenAI launches GPT-5 preview\n   Multimodal reasoning benchmark sets new record',
  },
  {
    query: 'What\'s the weather in Seoul tomorrow?',
    steps: ['🌐 Fetching weather data...', '🧮 Analyzing forecast...'],
    result: '🌤️ Seoul Weather Tomorrow\n\nHigh 23°C / Low 14°C\nMorning: Clear ☀️\nAfternoon: Partly cloudy 🌤️\nRain chance: 10%\n\n💡 A light jacket should be enough!',
  },
  {
    query: 'Draft a meeting scheduling email',
    steps: ['📋 Analyzing request...', '✍️ Writing email draft...'],
    result: '📧 Email Draft Ready\n\nSubject: Meeting Request — Project Discussion\n\nHi John,\n\nI\'d like to schedule a meeting to discuss the project progress.\n\n📅 Available: Tue 5/27 2:00 PM or Wed 5/28 10:00 AM\n⏱️ Duration: ~1 hour\n📍 Location: Video call (link to follow)\n\nPlease let me know what works best for you.\n\nBest regards',
  },
]

const sleep = (ms: number) => new Promise(res => setTimeout(res, ms))

// ─── 10-persona arrays (backend vertical IDs) ────────────────────────────────
const JOB_PERSONAS_KO = [
  { id: 'developer',  emoji: '💻', name: '개발자 / IT 엔지니어',    desc: 'GitHub 트렌딩·해커뉴스·기술 동향', color: '#6366f1', proFeature: null, proLabel: null },
  { id: 'legal',      emoji: '⚖️', name: '변호사 / 법무 담당자',   desc: '판례·법령 개정·계약서 검토', color: '#f97316', proFeature: 'contract_review', proLabel: '⚖️ Pro 전용 — 계약서 검토 무제한' },
  { id: 'medical',    emoji: '🏥', name: '의사 / 의료진',           desc: '의료뉴스·건보 급여·임상 정보', color: '#06b6d4', proFeature: 'medical_search', proLabel: '🏥 Pro 전용 — 의학 검색 무제한' },
  { id: 'accountant', emoji: '📊', name: '회계사 / 세무사',         desc: '세금신고일정·환율·국세청 공지', color: '#f59e0b', proFeature: null, proLabel: null },
  { id: 'creator',    emoji: '🎬', name: '유튜버 / 크리에이터',    desc: '유튜브 트렌딩·틱톡·인터넷 이슈', color: '#ef4444', proFeature: 'content_script', proLabel: '🎬 Pro 전용 — 스크립트 생성 무제한' },
  { id: 'realtor',    emoji: '🏠', name: '부동산 전문가',           desc: '부동산뉴스·청약일정·금리동향', color: '#22c55e', proFeature: null, proLabel: null },
  { id: 'teacher',    emoji: '📚', name: '교사 / 강사',             desc: '교육부 공지·수능일정·EBS 콘텐츠', color: '#0ea5e9', proFeature: null, proLabel: null },
  { id: 'hr',         emoji: '👥', name: 'HR / 인사 담당자',        desc: '채용뉴스·최저임금·워크넷 공고', color: '#8b5cf6', proFeature: null, proLabel: null },
  { id: 'engineer',   emoji: '⚙️', name: '엔지니어 / 제조업',      desc: '산업뉴스·원자재시세·KS규격', color: '#10b981', proFeature: null, proLabel: null },
  { id: 'general',    emoji: '🌟', name: '일반 사용자',             desc: '날씨·뉴스·해커뉴스 토픽', color: '#ec4899', proFeature: null, proLabel: null },
]

const JOB_PERSONAS_EN = [
  { id: 'developer',  emoji: '💻', name: 'Developer / IT Engineer',    desc: 'GitHub Trending · HN · Tech news', color: '#6366f1', proFeature: null, proLabel: null },
  { id: 'legal',      emoji: '⚖️', name: 'Lawyer / Legal Counsel',     desc: 'Cases · Law amendments · Contract review', color: '#f97316', proFeature: 'contract_review', proLabel: '⚖️ Pro only — Unlimited contract review' },
  { id: 'medical',    emoji: '🏥', name: 'Doctor / Medical Staff',      desc: 'Medical news · Insurance · Clinical info', color: '#06b6d4', proFeature: 'medical_search', proLabel: '🏥 Pro only — Unlimited medical search' },
  { id: 'accountant', emoji: '📊', name: 'Accountant / Tax Advisor',    desc: 'Tax deadlines · FX rates · IRS news', color: '#f59e0b', proFeature: null, proLabel: null },
  { id: 'creator',    emoji: '🎬', name: 'YouTuber / Creator',          desc: 'YouTube Trending · TikTok · Viral memes', color: '#ef4444', proFeature: 'content_script', proLabel: '🎬 Pro only — Unlimited script generation' },
  { id: 'realtor',    emoji: '🏠', name: 'Real Estate Agent',           desc: 'Property news · Listings · Rate trends', color: '#22c55e', proFeature: null, proLabel: null },
  { id: 'teacher',    emoji: '📚', name: 'Teacher / Instructor',        desc: 'Education news · Exam schedule · Resources', color: '#0ea5e9', proFeature: null, proLabel: null },
  { id: 'hr',         emoji: '👥', name: 'HR / Recruiter',              desc: 'Hiring news · Min wage · Job postings', color: '#8b5cf6', proFeature: null, proLabel: null },
  { id: 'engineer',   emoji: '⚙️', name: 'Engineer / Manufacturing',   desc: 'Industry news · Metal prices · ISO standards', color: '#10b981', proFeature: null, proLabel: null },
  { id: 'general',    emoji: '🌟', name: 'General User',                desc: 'Weather · News · Hacker News topics', color: '#ec4899', proFeature: null, proLabel: null },
]

// ─── Job-specific WOW demos ───────────────────────────────────────────────────
const JOB_DEMOS: Record<string, { query: string; steps: string[]; result: string; proHint?: string }> = {
  developer: {
    query: 'GitHub 오늘 트렌딩 레포 알려줘',
    steps: ['⭐ GitHub Trending 수집 중...', '🔶 Hacker News 상위 글 수집 중...', '📋 요약 정리 중...'],
    result: '⭐ GitHub Trending (오늘)\n\n1. microsoft/TypeScript ↑1,234⭐\n   "TypeScript 5.5 RC — 타입 추론 대폭 개선"\n2. vercel/ai ↑987⭐\n   "AI SDK 4.0 — 스트리밍 & 툴 콜 새 API"\n3. golang/go ↑756⭐\n   "Go 1.23 iterator 문법 안정화"\n\n🔶 Hacker News Top\n• "We built a 10M req/day service on SQLite"\n• "LLM context windows are getting too large"\n\n💡 오늘의 키워드: TypeScript · AI SDK · SQLite',
  },
  legal: {
    query: '오늘 법률·판례 뉴스 브리핑해줘',
    steps: ['⚖️ 법률 뉴스 수집 중...', '📋 최근 법령 개정 확인 중...', '🗓️ 법원 일정 조회 중...'],
    result: '⚖️ 오늘의 법무 브리핑\n\n📰 주요 뉴스\n• 대법원, 포괄임금제 무효 판결 재확인\n  "연장·야간근로 별도 산정 의무화"\n• 개인정보보호법 시행령 개정 공포\n  2025.07.01 시행 — 위반 과징금 상향\n\n📋 최근 법령 개정\n• 근로기준법 제56조 개정 (연장근로 한도)\n• 전자금융거래법 보완 입법 추진 중\n\n🗓️ 법원 일정\n민사 접수 마감: 오늘 17:00\n\n⚠️ 최종 법적 판단은 변호사 확인 필요',
    proHint: 'Pro 플랜에서 계약서 검토를 무제한으로 사용하세요',
  },
  medical: {
    query: '오늘 의료·임상 뉴스 브리핑해줘',
    steps: ['🩺 의료 뉴스 수집 중...', '💊 건강보험 급여 변경 확인 중...', '🌤️ 날씨 조회 중...'],
    result: '🩺 오늘의 의료 브리핑\n\n📰 의료 뉴스 (청년의사)\n• GLP-1 계열 비만치료제 급여 기준 확대\n  당뇨 없는 고도비만 환자도 적용 가능\n• 응급실 과부하 해소책 — 경증 환자 분리 시범\n\n💊 건보 급여 동향\n• 희귀질환 신약 급여 등재 절차 간소화\n• 한의과 초음파 급여 시범사업 2026 확대\n\n🌤️ 서울 날씨\n최고 23°C · 맑음 — 외래 환자 방문 많을 예정\n\n⚠️ 임상 결정 시 전문의 판단 필수',
    proHint: 'Pro 플랜에서 의학 검색을 무제한으로 사용하세요',
  },
  accountant: {
    query: '이번달 세금 신고 일정이랑 환율 알려줘',
    steps: ['📅 세무 신고 일정 조회 중...', '💱 실시간 환율 수집 중...', '📊 국세청 공지 확인 중...'],
    result: '📅 이번 달 세무 일정\n\n📌 종합소득세 확정신고 (5/31 마감)\n• 사업소득·임대소득·프리랜서 해당\n• 홈택스 신고 권장 (5/25 이후 혼잡)\n\n💱 실시간 환율 (USD 기준)\nUSD/KRW  1,382.50\nEUR/KRW  1,498.20\nJPY/KRW     9.21\nCNY/KRW   190.40\n\n📊 국세청 최신 공지\n• 전자세금계산서 의무 발급 범위 확대\n• 성실신고확인제도 업종 추가\n\n💡 이번 달 핵심: 5/31 종합소득세 마감!',
  },
  creator: {
    query: '오늘 유튜브 트렌딩이랑 틱톡 바이럴 알려줘',
    steps: ['🔥 유튜브 트렌딩 수집 중...', '🎵 틱톡 트렌드 분석 중...', '🌐 인터넷 이슈 정리 중...'],
    result: '🔥 유튜브 트렌딩 (한국 TOP)\n\n1. "AI로 월 1000만원 버는 법" — 조회 120만\n2. "2026 서울 맛집 최신 업데이트" — 조회 87만\n3. "챗GPT로 유튜브 스크립트 자동화" — 조회 64만\n\n🎵 틱톡 바이럴\n• #AI사용법 — 15억 뷰 누적\n• #퇴사브이로그 — 오늘 급상승\n\n🌐 인터넷 이슈\n• 오늘의 밈: "AI가 내 일자리를 빼앗았다"\n• 커뮤니티 화제: 편의점 신메뉴 논란\n\n💡 오늘 영상 주제 추천: "AI 자동화 실제 사례"',
    proHint: 'Pro 플랜에서 스크립트 생성을 무제한으로 사용하세요',
  },
  realtor: {
    query: '오늘 부동산 뉴스랑 청약 일정 알려줘',
    steps: ['🏠 부동산 뉴스 수집 중...', '📋 청약 일정 조회 중...', '💰 금리·환율 확인 중...'],
    result: '🏠 오늘의 부동산 브리핑\n\n📰 주요 뉴스\n• 서울 아파트 매매 거래량 전월비 12% 증가\n• 강남 3구 토지거래허가구역 지정 6개월 연장\n• 2030 패닉바잉 재개 조짐 — 전문가 의견 엇갈려\n\n📋 이번 달 청약 일정\n• 동탄역 SK뷰 (5/27~5/29, 84㎡ 4.2억)\n• 광명 센트럴아이파크 (5/30~6/1)\n\n💰 금리·환율 동향\nUSD 1,382 / 기준금리 3.25% 동결\n주담대 변동금리: 연 4.8~5.3%\n\n💡 핵심: 이번 주 동탄역 청약 놓치지 마세요!',
  },
  teacher: {
    query: '오늘 교육부 공지랑 수능 일정 알려줘',
    steps: ['📚 교육 뉴스 수집 중...', '🎓 수능·대입 일정 확인 중...', '📺 EBS 콘텐츠 조회 중...'],
    result: '📚 오늘의 교육 브리핑\n\n📰 교육 뉴스\n• 2026학년도 수능 출제 기조 발표\n  "EBS 연계율 50% 유지, 킬러문항 배제"\n• 방과후학교 AI·코딩 프로그램 확대\n  전국 초등 1,200개교 시범 운영\n\n🎓 수능·대입 일정\n• 2026 수능: 2025.11.13(목)\n• 성적 발표: 2025.12.05\n• 정시 원서접수: 2026.01.09~12\n\n📺 EBS 오늘의 추천\n• 수능특강 수학Ⅱ 3강 (적분 활용)\n• EBSi 영어듣기 모의고사\n\n💡 핵심: 11월 수능까지 D-176일!',
  },
  hr: {
    query: '오늘 채용·HR 뉴스랑 최저임금 알려줘',
    steps: ['👥 HR 뉴스 수집 중...', '📋 최저임금 정보 조회 중...', '💼 워크넷 채용 공고 확인 중...'],
    result: '👥 오늘의 HR 브리핑\n\n📰 채용·HR 뉴스\n• 삼성·SK·LG 2026 상반기 공채 일정 확정\n  서류: 6/2~6/13, 필기: 7/5\n• AI 채용 도구 도입 기업 3년새 3배 증가\n\n📋 최저임금 현황 (2025)\n시급: ₩10,030\n일급(8h): ₩80,240\n월급: ₩2,096,270\n\n💼 오늘 주요 채용 (워크넷)\n• 네이버 — AI 서비스 개발자 (서울)\n• 카카오 — 데이터 애널리스트 (판교)\n• 현대자동차 — 품질관리 엔지니어 (울산)\n\n💡 핵심: 6월 대기업 공채 시즌 시작!',
  },
  engineer: {
    query: '오늘 산업 뉴스랑 원자재 시세 알려줘',
    steps: ['⚙️ 산업·제조 뉴스 수집 중...', '📦 원자재 시세 조회 중...', '📐 KS/ISO 규격 업데이트 확인 중...'],
    result: '⚙️ 오늘의 엔지니어링 브리핑\n\n📰 산업·제조 뉴스\n• 반도체 후공정 자동화 투자 급증\n  패키징 공정 로봇화 2025년 40% 확대\n• 탄소중립 공정 전환 보조금 신청 시작\n  중소 제조업 최대 5억원 지원\n\n📦 원자재 시세\n철강(열연) ₩750,000/톤 ▲+1.2%\n구리(LME) $9,850/톤 ▲+0.8%\n알루미늄 $2,420/톤 ▼-0.3%\n\n📐 KS/ISO 업데이트\n• KS B ISO 9001:2025 개정 준비 중\n• 전기차 배터리 안전규격 KS C 신설\n\n💡 핵심: 탄소중립 보조금 신청 기한 확인하세요!',
  },
  general: {
    query: '오늘 날씨랑 주요 뉴스 알려줘',
    steps: ['🌤️ 날씨 데이터 수집 중...', '📰 주요 뉴스 수집 중...', '🔶 Hacker News 확인 중...'],
    result: '🌤️ 오늘 서울 날씨\n최고 23°C / 최저 14°C · 맑음 ☀️\n강수 확률 10% · 미세먼지 좋음\n\n📰 오늘의 주요 뉴스\n1. 한국은행 기준금리 3.25% 동결\n2. 삼성전자 HBM4 양산 돌입 — 주가 +3.2%\n3. 2026 서울시 버스 노선 개편 확정\n\n🔶 Hacker News Top\n• "We built a 10M req/day service on SQLite"\n• "The death of the junior developer"\n\n💡 오늘도 좋은 하루 되세요!',
  },
}

const JOB_DEMOS_EN: Record<string, { query: string; steps: string[]; result: string; proHint?: string }> = {
  developer: {
    query: "Show me today's GitHub trending repos",
    steps: ['⭐ Fetching GitHub Trending...', '🔶 Fetching Hacker News top stories...', '📋 Compiling summary...'],
    result: "⭐ GitHub Trending (Today)\n\n1. microsoft/TypeScript ↑1,234⭐\n   \"TypeScript 5.5 RC — Improved type inference\"\n2. vercel/ai ↑987⭐\n   \"AI SDK 4.0 — New streaming & tool call API\"\n3. golang/go ↑756⭐\n   \"Go 1.23 iterator syntax stabilized\"\n\n🔶 Hacker News Top\n• \"We built a 10M req/day service on SQLite\"\n• \"LLM context windows are getting too large\"\n\n💡 Today's keywords: TypeScript · AI SDK · SQLite",
  },
  legal: {
    query: 'Review this employment contract',
    steps: ['📄 Classifying contract clauses...', '⚖️ Cross-referencing case law...', '🔍 Grading risk levels...'],
    result: '⚖️ Employment Contract Review\n\n🔴 High Risk (2)\n• Clause 7: Blanket overtime → Supreme Court may void\n  Suggested: "OT/night work calculated separately"\n• Clause 12: Non-compete 5 years → excessive, courts reduce to 2\n\n🟡 Caution (1)\n• Clause 15: No cap on damages specified\n\n✅ 12 standard clauses OK\n\nOverall Risk: 🟡 Moderate\n⚠️ Final review by licensed attorney required',
    proHint: 'Unlimited contract reviews with Pro plan',
  },
  medical: {
    query: 'Metformin dosing in renal impairment?',
    steps: ['📚 Searching latest PubMed papers...', '🔬 Classifying evidence levels...', '📋 Summarizing clinical guidelines...'],
    result: '🏥 Metformin Renal Dosing Guide\n\n📊 Evidence: ADA 2024 (Grade A)\n\neGFR ≥ 45: Standard 1,000mg bid\neGFR 30~44: 500mg bid (reduced)\neGFR < 30: Contraindicated\n\n⚠️ Hold 48h before contrast media\n\n🆕 2024 Update: XR formulation 68% less GI side effects\n\n⚠️ Clinical decisions require physician judgement',
    proHint: 'Unlimited medical search with Pro plan',
  },
  accountant: {
    query: "This month's tax deadlines and exchange rates?",
    steps: ['📅 Looking up tax filing schedule...', '💱 Fetching live exchange rates...', '📊 Checking IRS/tax news...'],
    result: "📅 This Month's Tax Deadlines\n\n📌 Individual Returns Due (Apr 15)\n• File or extend by April 15\n• Q1 estimated tax also due April 15\n\n💱 Live Exchange Rates (USD base)\nEUR/USD  1.082\nJPY/USD  0.0066\nGBP/USD  1.271\nCNY/USD  0.138\n\n📊 IRS News\n• New crypto reporting rules effective 2025\n• Standard deduction increased to $14,600\n\n💡 Key this month: April 15 deadline approaching!",
  },
  creator: {
    query: 'Write a YouTube script on AI tools',
    steps: ['🔍 Analyzing trending & competitor videos...', '✍️ Building hook · body · outro...', '🏷️ SEO optimization...'],
    result: '🎬 YouTube Script Ready!\n\n🎯 Hook: "Not using AI in 2026 costs you $5k/year"\n\n📌 Intro: I saved 40 hours of work with a single AI tool...\n\n📋 Body\n① Automate repetitive tasks → save 20h/month\n② AI research → 90% faster content prep\n③ Script automation → upload 3 videos/week\n\n🔚 Outro: Drop your #1 task to automate in the comments!\n\n📌 Title: "AI Made Me Leave Work 3 Hours Early"\n🏷️ #AItools #productivity #youtubescript',
    proHint: 'Unlimited script generation with Pro plan',
  },
  realtor: {
    query: "Today's real estate news and listings",
    steps: ['🏠 Fetching real estate news...', '📋 Checking housing application schedule...', '💰 Fetching interest rates & FX...'],
    result: "🏠 Real Estate Briefing\n\n📰 Top News\n• Fed holds rates — mortgage rates stable at 6.8%\n• NYC apartment inventory up 18% YoY\n• Commercial real estate vacancies hit 10-year high\n\n📋 New Listings This Week\n• Austin TX — 3BR/2BA $425k (↓$15k)\n• Remote-friendly suburbs seeing surge\n\n💰 Rate Snapshot\nFed Funds: 5.25% (hold)\n30yr Fixed: 6.82%\nUSD Index: 104.2\n\n💡 Key: Rate-hold = buyer confidence returning",
  },
  teacher: {
    query: "Today's education news and exam schedule",
    steps: ['📚 Fetching education news...', '🎓 Checking exam schedule...', '📺 Finding recommended content...'],
    result: "📚 Education Briefing\n\n📰 Education News\n• AI tutoring tools adoption in K-12 up 3x\n  Majority of districts piloting ChatGPT-based tools\n• College Board announces SAT score inflation review\n\n🎓 Key Dates\n• SAT: June 7, 2025\n• ACT: June 14, 2025\n• AP Score Release: July 2025\n• Common App opens: Aug 1, 2025\n\n📺 Recommended Resources\n• Khan Academy: AP Calculus BC full course\n• Crash Course: US History for AP exam\n\n💡 Key: AP season in full swing — prep your students!",
  },
  hr: {
    query: "Today's hiring news and minimum wage info",
    steps: ['👥 Fetching HR & hiring news...', '📋 Checking minimum wage data...', '💼 Finding top job postings...'],
    result: "👥 HR Briefing\n\n📰 Hiring News\n• Big Tech layoffs slow — hiring restarts Q3\n  Amazon, Google announcing 8,000+ new roles\n• AI skills now in 40% of all job postings\n\n📋 Minimum Wage (US Federal)\nFederal: $7.25/hr\nCA: $16.50 · NY: $16.00 · WA: $16.28\n\n💼 Top Job Postings Today\n• Google — Staff ML Engineer (Remote)\n• Stripe — Senior Data Analyst (SF/NYC)\n• Shopify — Product Manager (Remote)\n\n💡 Key: AI skills premium +23% salary bump average",
  },
  engineer: {
    query: "Today's industry news and metal prices",
    steps: ['⚙️ Fetching industry & manufacturing news...', '📦 Fetching raw material prices...', '📐 Checking ISO standards updates...'],
    result: "⚙️ Engineering Briefing\n\n📰 Industry News\n• US manufacturing PMI 52.3 — expansion continues\n• EV battery production capacity up 35% YoY\n• Semiconductor supply chain normalizing by Q4\n\n📦 Raw Material Prices\nSteel (HRC) $680/ton ▲+1.2%\nCopper (LME) $9,850/ton ▲+0.8%\nAluminum $2,420/ton ▼-0.3%\n\n📐 Standards Updates\n• ISO 9001:2025 revision in progress\n• ASME B31.3 process piping code updated\n\n💡 Key: Copper prices rising — plan procurement now",
  },
  general: {
    query: "What's today's weather and top news?",
    steps: ['🌤️ Fetching weather data...', '📰 Fetching top news...', '🔶 Checking Hacker News...'],
    result: "🌤️ Seoul Weather Today\nHigh 23°C / Low 14°C · Clear ☀️\nRain chance 10% · Air quality: Good\n\n📰 Top News Today\n1. Fed holds interest rates steady at 5.25%\n2. Samsung starts HBM4 mass production — stock +3.2%\n3. OpenAI GPT-5 preview launches for Pro users\n\n🔶 Hacker News Top\n• \"We built a 10M req/day service on SQLite\"\n• \"The death of the junior developer\"\n\n💡 Have a great day!",
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

  const [selectedPlan, setSelectedPlan]   = useState<'free' | 'pro' | 'team'>('free')

  // Demo state for Step 2 (job-specific)
  const [demoRunning, setDemoRunning]     = useState(false)
  const [demoStarted, setDemoStarted]     = useState(false)
  const [demoThinkStep, setDemoThinkStep] = useState('')
  const [demoTyping, setDemoTyping]       = useState('')
  const [demoResult, setDemoResult]       = useState('')
  const [demoInputTyping, setDemoInputTyping] = useState('')
  const demoEndRef = useRef<HTMLDivElement>(null)

  // Generic demo state for Step 0
  const [gDemoIdx, setGDemoIdx]           = useState(0)
  const [gDemoRunning, setGDemoRunning]   = useState(false)
  const [gDemoStarted, setGDemoStarted]   = useState(false)
  const [gDemoInput, setGDemoInput]       = useState('')
  const [gDemoThink, setGDemoThink]       = useState('')
  const [gDemoTyping, setGDemoTyping]     = useState('')
  const [gDemoResult, setGDemoResult]     = useState('')
  const gDemoEndRef = useRef<HTMLDivElement>(null)

  const selectedStyle = REALISTIC_STYLE_PRESETS.find(s => s.id === styleId) ?? REALISTIC_STYLE_PRESETS[0]
  const selectedJob   = JOB_PERSONAS.find(p => p.id === selectedJobId) ?? JOB_PERSONAS[0]
  const jobColor      = selectedJob.color

  // Google OAuth callback
  useEffect(() => {
    if (isLoggedIn && userEmail && step >= 3 && step <= 5 && !didAutoComplete.current) {
      didAutoComplete.current = true
      setGoogleEmail(userEmail)
      setStep(5)
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isLoggedIn, userEmail])

  // Generic demo auto-run (Step 0)
  useEffect(() => {
    if (step === 0 && !gDemoStarted) {
      setGDemoStarted(true)
      void runGenericDemo(0)
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [step])

  // Job demo auto-run (Step 2)
  const demoJobRef = useRef<string>('')
  useEffect(() => {
    if (step === 2 && (!demoStarted || demoJobRef.current !== selectedJobId)) {
      demoJobRef.current = selectedJobId
      setDemoStarted(true)
      void runJobDemo()
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [step, selectedJobId])

  const runGenericDemo = async (idx: number, forceLang?: 'ko' | 'en') => {
    const demos = (forceLang ?? lang) === 'en' ? GENERIC_DEMOS_EN : GENERIC_DEMOS_KO
    const demo = demos[idx % demos.length]
    setGDemoIdx(idx % demos.length)
    setGDemoRunning(true)
    setGDemoResult('')
    setGDemoTyping('')
    setGDemoThink('')
    setGDemoInput('')

    for (let i = 1; i <= demo.query.length; i++) {
      setGDemoInput(demo.query.slice(0, i))
      await sleep(30)
    }
    await sleep(300)

    for (const s of demo.steps) {
      setGDemoThink(s)
      await sleep(800)
    }
    setGDemoThink('')

    const CHUNK = 5
    let typed = ''
    for (let i = 0; i < demo.result.length; i += CHUNK) {
      typed = demo.result.slice(0, i + CHUNK)
      setGDemoTyping(typed)
      await sleep(18)
      if (i % 50 === 0) gDemoEndRef.current?.scrollIntoView({ behavior: 'smooth' })
    }
    setGDemoResult(demo.result)
    setGDemoTyping('')
    setGDemoRunning(false)
  }

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
      setStep(5)
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
      setStep(5)
      return
    }
    setGoogleLoading(true)
    try {
      const hint = localStorage.getItem('nexus-user-email') ?? undefined
      await signInWithGoogle(undefined, hint)
    } catch (e) {
      console.warn('Google OAuth failed, starting trial:', e)
      const trialExpiry = new Date(Date.now() + 3 * 24 * 60 * 60 * 1000).toISOString()
      const demoEmail = 'user@gmail.com'
      localStorage.setItem('nexus-user-email', demoEmail)
      localStorage.setItem('nexus-sub-status', 'trial')
      localStorage.setItem('nexus-sub-expiry', trialExpiry)
      setGoogleEmail(demoEmail)
      setStep(5)
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
      fetch(`${BASE}/api/vertical/config`, {
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
  const card: React.CSSProperties = {
    position: 'fixed',
    top: '50%', left: '50%',
    transform: 'translate(-50%, -50%)',
    zIndex: 99999,
    width: 560,
    maxHeight: '90vh',
    background: '#1e2035',
    border: '1px solid rgba(255,255,255,0.12)',
    borderRadius: 28,
    padding: '32px 36px',
    boxShadow: '0 24px 64px rgba(0,0,0,0.5)',
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
    <>
      {/* 닫기 버튼 — 우측 상단 고정 */}
      <button
        onClick={async () => {
          try {
            const { getCurrentWindow } = await import('@tauri-apps/api/window')
            getCurrentWindow().close()
          } catch { window.close() }
        }}
        style={{
          position: 'fixed', top: 16, right: 16, zIndex: 100000,
          width: 32, height: 32, borderRadius: '50%',
          background: 'rgba(0,0,0,0.35)', border: '1px solid rgba(255,255,255,0.2)',
          color: 'rgba(255,255,255,0.7)', fontSize: 16, cursor: 'pointer',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          backdropFilter: 'blur(8px)', transition: 'all 0.15s',
        }}
        onMouseEnter={e => { e.currentTarget.style.background = 'rgba(239,68,68,0.7)'; e.currentTarget.style.color = 'white' }}
        onMouseLeave={e => { e.currentTarget.style.background = 'rgba(0,0,0,0.35)'; e.currentTarget.style.color = 'rgba(255,255,255,0.7)' }}
      >✕</button>

      <AnimatePresence mode="wait">

        {/* ══════════════════════════════════════════════
            Step 0: 범용 인터랙티브 데모
        ══════════════════════════════════════════════ */}
        {step === 0 && (
          <motion.div
            key="step0-demo"
            initial={{ opacity: 0, y: 30 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            style={{
              width: '100%', maxWidth: 620,
              background: '#0d0f1a',
              border: '1px solid rgba(255,255,255,0.08)',
              borderRadius: 24, overflow: 'hidden',
              boxShadow: `0 0 80px ${selectedStyle.primaryColor}22, 0 32px 80px rgba(0,0,0,0.6)`,
              position: 'relative',
            }}
          >
            {/* Language toggle */}
            <div style={{ position: 'absolute', top: 16, right: 16, display: 'flex', gap: 4, zIndex: 10 }}>
              {(['ko', 'en'] as const).map(l => (
                <button key={l}
                  onClick={() => {
                    setLang(l); setUserLang(l)
                    setName(l === 'en' ? 'Nexus' : '넥서스')
                    setNameInput(l === 'en' ? 'Nexus' : '넥서스')
                    setUserName(l === 'en' ? 'Boss' : '주인님')
                    setGDemoStarted(false); setGDemoResult(''); setGDemoTyping(''); setGDemoThink(''); setGDemoInput(''); setGDemoRunning(false)
                    // 언어 변경 후 데모 재실행 (setTimeout으로 state 반영 후 실행)
                    setTimeout(() => void runGenericDemo(0, l), 50)
                  }}
                  style={{
                    padding: '3px 9px', borderRadius: 8,
                    background: lang === l ? 'rgba(255,255,255,0.18)' : 'rgba(255,255,255,0.05)',
                    border: lang === l ? '1px solid rgba(255,255,255,0.35)' : '1px solid rgba(255,255,255,0.1)',
                    color: lang === l ? 'white' : 'rgba(255,255,255,0.4)',
                    fontSize: 11, fontWeight: 700, cursor: 'pointer',
                  }}
                >{l === 'ko' ? '🇰🇷 KO' : '🇺🇸 EN'}</button>
              ))}
            </div>

            {/* 우주 캐릭터 + Header */}
            <div style={{ display: 'flex', alignItems: 'center', gap: 16, padding: '20px 24px 10px' }}>
              <div style={{
                flexShrink: 0, width: 80, height: 80,
                background: 'linear-gradient(135deg, #6366f122, #a855f722)',
                border: '1.5px solid #6366f144',
                borderRadius: 20,
                display: 'flex', alignItems: 'center', justifyContent: 'center',
                fontSize: 42,
              }}>🚀</div>
              <div>
                <div style={{ fontSize: 11, letterSpacing: '0.12em', color: selectedStyle.primaryColor, marginBottom: 4, fontWeight: 600 }}>NEXUS AI</div>
                <h2 style={{ fontSize: 20, fontWeight: 800, color: 'white', marginBottom: 3, lineHeight: 1.3 }}>
                  {isEn ? 'Your AI PC Assistant' : 'AI PC 비서를 만나보세요'}
                </h2>
                <p style={{ fontSize: 12, color: 'rgba(255,255,255,0.4)', lineHeight: 1.5 }}>
                  {isEn ? 'Watch Nexus work in real-time ↓' : '넥서스가 실시간으로 일하는 모습을 보세요 ↓'}
                </p>
              </div>
            </div>

            {/* Demo area */}
            <div style={{ margin: '0 24px', background: 'rgba(255,255,255,0.03)', border: '1px solid rgba(255,255,255,0.07)', borderRadius: 16, overflow: 'hidden' }}>
              {/* Fake input */}
              <div style={{ padding: '12px 16px', borderBottom: '1px solid rgba(255,255,255,0.06)', display: 'flex', alignItems: 'center', gap: 10 }}>
                <div style={{
                  flex: 1, padding: '8px 12px',
                  background: 'rgba(255,255,255,0.05)',
                  border: `1px solid ${gDemoRunning ? selectedStyle.primaryColor + '55' : 'rgba(255,255,255,0.1)'}`,
                  borderRadius: 10, fontSize: 13,
                  color: gDemoInput ? 'white' : 'rgba(255,255,255,0.25)',
                  minHeight: 20, transition: 'border-color 0.2s',
                }}>
                  {gDemoInput || (isEn ? 'Asking Nexus...' : 'Nexus에게 묻는 중...')}
                  {gDemoInput && gDemoRunning && <span style={{ animation: 'blink 0.6s step-end infinite', opacity: 0.8 }}>|</span>}
                </div>
                <div style={{
                  width: 34, height: 34, borderRadius: 9, flexShrink: 0,
                  background: gDemoRunning ? `linear-gradient(135deg, ${selectedStyle.primaryColor}, ${selectedStyle.accentColor})` : 'rgba(255,255,255,0.08)',
                  display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 15,
                }}>{gDemoRunning ? '⚡' : '↑'}</div>
              </div>
              {/* Result */}
              <div style={{ minHeight: 180, maxHeight: 240, overflowY: 'auto', padding: '16px', scrollbarWidth: 'none' } as React.CSSProperties}>
                {gDemoThink && (
                  <motion.div key={gDemoThink} initial={{ opacity: 0, x: -6 }} animate={{ opacity: 1, x: 0 }}
                    style={{ display: 'flex', marginBottom: 12 }}>
                    <div style={{ fontSize: 11, color: selectedStyle.primaryColor, background: `${selectedStyle.primaryColor}12`, border: `1px solid ${selectedStyle.primaryColor}33`, padding: '6px 12px', borderRadius: 20 }}>{gDemoThink}</div>
                  </motion.div>
                )}
                {(gDemoTyping || gDemoResult) && (
                  <div style={{ padding: '10px 14px', borderRadius: '4px 16px 16px 16px', background: 'rgba(255,255,255,0.06)', border: '1px solid rgba(255,255,255,0.1)', fontSize: 12, color: 'rgba(255,255,255,0.92)', lineHeight: 1.7, whiteSpace: 'pre-wrap', fontFamily: 'monospace' }}>
                    {gDemoTyping || gDemoResult}
                    {gDemoTyping && <span style={{ animation: 'blink 0.8s step-end infinite', opacity: 0.7 }}>▌</span>}
                  </div>
                )}
                {!gDemoRunning && !gDemoResult && !gDemoThink && (
                  <div style={{ color: 'rgba(255,255,255,0.2)', fontSize: 12, textAlign: 'center', paddingTop: 50 }}>
                    {isEn ? 'Starting demo...' : '데모 시작 중...'}
                  </div>
                )}
                <div ref={gDemoEndRef} />
              </div>
            </div>

            {/* Demo pagination dots */}
            <div style={{ display: 'flex', justifyContent: 'center', gap: 6, padding: '12px 0 4px' }}>
              {(isEn ? GENERIC_DEMOS_EN : GENERIC_DEMOS_KO).map((_, i) => (
                <button key={i}
                  onClick={() => { if (!gDemoRunning) { setGDemoResult(''); setGDemoTyping(''); void runGenericDemo(i) } }}
                  style={{
                    width: i === gDemoIdx ? 20 : 7, height: 7, borderRadius: 4,
                    background: i === gDemoIdx ? selectedStyle.primaryColor : 'rgba(255,255,255,0.2)',
                    border: 'none', cursor: gDemoRunning ? 'default' : 'pointer', transition: 'all 0.3s', padding: 0,
                  }}
                />
              ))}
            </div>

            {/* CTA */}
            <div style={{ padding: '12px 24px 24px', display: 'flex', flexDirection: 'column', gap: 8 }}>
              {nextBtn(
                () => setStep(1),
                isEn ? 'I want this! →' : '나도 써보고 싶다! →',
                false,
              )}
              <p style={{ fontSize: 11, color: 'rgba(255,255,255,0.25)', textAlign: 'center', margin: 0 }}>
                {isEn ? 'Select your job next → personalized AI for you' : '다음에 직업 선택 → 나만의 맞춤 AI'}
              </p>
            </div>
          </motion.div>
        )}

        {/* ══════════════════════════════════════════════
            Step 1: 직업군 선택
        ══════════════════════════════════════════════ */}
        {step === 1 && (
          <motion.div
            key="step1-job"
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
                      setStep(2)
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

            <p style={{ fontSize: 11, color: 'rgba(255,255,255,0.25)', textAlign: 'center', marginBottom: 12 }}>
              {isEn ? 'Click to select and continue' : '선택하면 자동으로 다음으로 넘어갑니다'}
            </p>
            {backBtn(() => setStep(0), isEn ? '← Back to Demo' : '← 데모로 돌아가기')}
          </motion.div>
        )}

        {/* ══════════════════════════════════════════════
            Step 2: 직업군 전용 WOW 데모
        ══════════════════════════════════════════════ */}
        {step === 2 && (
          <motion.div
            key="step2-demo"
            initial={{ opacity: 0, y: 30 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            style={{
              width: '100%', maxWidth: 620,
              background: '#0d0f1a',
              border: '1px solid rgba(255,255,255,0.08)',
              borderRadius: 24,
              overflow: 'hidden',
              boxShadow: `0 0 80px ${jobColor}22, 0 32px 80px rgba(0,0,0,0.6)`,
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
                () => setStep(3),
                isEn ? 'Choose My Plan →' : '플랜 선택하기 →',
                demoRunning || !!demoTyping,
              )}
              {backBtn(() => { setJobSelected(false); setStep(1) }, isEn ? '← Change Job' : '← 직업 바꾸기')}
            </div>
          </motion.div>
        )}

        {/* ══════════════════════════════════════════════
            Step 2: 플랜 선택
        ══════════════════════════════════════════════ */}
        {step === 3 && (
          <motion.div
            key="step3-plan"
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
              {nextBtn(() => setStep(4), isEn ? 'Continue →' : '다음 →')}
              {backBtn(() => setStep(2), isEn ? '← Back to Demo' : '← 데모로 돌아가기')}
            </div>
          </motion.div>
        )}

        {/* ══════════════════════════════════════════════
            Step 3: Google Login
        ══════════════════════════════════════════════ */}
        {step === 4 && (
          <motion.div
            key="step4-login"
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
                    fontSize: 11, color: 'transparent', textAlign: 'center', padding: '4px 0',
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
                ? nextBtn(() => setStep(5), isEn ? 'Continue →' : '다음 →')
                : <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.25)', textAlign: 'center' }}>
                    {isEn ? 'Sign in with Google to continue' : '구글 로그인 후 계속 진행할 수 있습니다'}
                  </div>
              }
              {backBtn(() => setStep(3), isEn ? '← Change Plan' : '← 플랜 변경')}
            </div>
          </motion.div>
        )}

        {/* ══════════════════════════════════════════════
            Step 4: Avatar + Name + Nickname (combined)
        ══════════════════════════════════════════════ */}
        {step === 5 && (
          <motion.div
            key="step5-profile"
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
                  void handleComplete()
                },
                isEn ? `Start with ${nameInput.trim() || 'Nexus'} ✦` : `${nameInput.trim() || '넥서스'} 시작하기 ✦`,
              )}
              {backBtn(() => setStep(4), isEn ? '← Back' : '← 이전')}
            </div>
          </motion.div>
        )}

      </AnimatePresence>
    </>
  )
}

/* ══════════════════════════════════════════════
   LoginScreen — 온보딩 완료 사용자 전용 로그인 화면
══════════════════════════════════════════════ */
export function LoginScreen() {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const { isLoggedIn } = useAppStore()

  useEffect(() => {
    if (isLoggedIn) setLoading(false)
  }, [isLoggedIn])

  const handleLogin = async () => {
    setLoading(true)
    setError('')
    try {
      await signInWithGoogle(() => setLoading(false))
    } catch (e: any) {
      setError(e?.message || '로그인 실패. 다시 시도해주세요.')
      setLoading(false)
    }
  }

  return (
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        style={{
          position: 'fixed',
          top: '50%', left: '50%',
          transform: 'translate(-50%, -50%)',
          zIndex: 99999,
          width: 400,
          background: '#1e2035',
          border: '1px solid rgba(255,255,255,0.12)',
          borderRadius: 24,
          padding: '40px 36px',
          boxShadow: '0 24px 64px rgba(0,0,0,0.5)',
          display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 20,
        }}
      >
        <div style={{ textAlign: 'center' }}>
          <div style={{ fontSize: 36, marginBottom: 12 }}>👋</div>
          <h2 style={{ fontSize: 20, fontWeight: 800, color: 'white', marginBottom: 6 }}>다시 오셨군요!</h2>
          <p style={{ fontSize: 13, color: 'rgba(255,255,255,0.45)', lineHeight: 1.6 }}>
            Google 계정으로 로그인하면<br />바로 시작됩니다.
          </p>
        </div>

        <button
          onClick={handleLogin}
          disabled={loading}
          style={{
            width: '100%', padding: '14px 20px',
            background: loading ? 'rgba(255,255,255,0.1)' : 'white',
            border: 'none', borderRadius: 12, cursor: loading ? 'wait' : 'pointer',
            display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 12,
            fontSize: 15, fontWeight: 700, color: '#1a1a2e',
            boxShadow: '0 4px 20px rgba(0,0,0,0.4)',
            opacity: loading ? 0.6 : 1,
          } as React.CSSProperties}
        >
          <svg width="20" height="20" viewBox="0 0 24 24">
            <path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"/>
            <path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/>
            <path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"/>
            <path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"/>
          </svg>
          {loading ? '로그인 중...' : 'Google로 로그인'}
        </button>

        {loading && (
          <p style={{ fontSize: 12, color: 'rgba(255,255,255,0.4)', textAlign: 'center', margin: 0 }}>
            브라우저에서 로그인 후 자동으로 돌아옵니다
          </p>
        )}
        {error && <p style={{ fontSize: 12, color: '#f87171', margin: 0 }}>{error}</p>}
      </motion.div>
  )
}
