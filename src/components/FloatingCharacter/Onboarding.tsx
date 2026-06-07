/**
 * Onboarding — 첫 실행 5분 마법사 (Phase A4)
 *
 * 5스텝:
 *  1. 환영 + 비전 ("자비스급 한국어 PC 비서")
 *  2. 첫 명령 데모 (PC 상태 자동 실행)
 *  3. 자동 문서 생성 데모 (Excel)
 *  4. 멀티 액션 데모 ("뉴스 찾고 PDF로")
 *  5. 페르소나 + 단축키 안내
 *
 * 표시 조건: localStorage 'nexus-onboarded' 없으면
 * 종료: 사용자가 완료 → 플래그 저장
 */

import { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'

const ONBOARDED_KEY = 'nexus-onboarded'
// 실제 빌드 버전(vite define 주입). 하드코딩 제거 → 설치 바이너리 버전과 동기화.
const APP_VERSION   = typeof __APP_VERSION__ !== 'undefined' ? __APP_VERSION__ : '0.0.0'
const VERSION_KEY   = 'nexus-app-version'

// major.minor만 추출 ("2.9.47" → "2.9"). 온보딩 재표시는 minor 변경에서만 일어나고
// 패치단위 자동업데이트(2.9.N → 2.9.N+1)는 온보딩을 유지한다.
function minorVersion(v: string | null): string {
  return v ? v.split('.').slice(0, 2).join('.') : ''
}
const APP_MINOR = minorVersion(APP_VERSION)

// ★ major.minor 업그레이드 시에만 이전 온보딩 상태 초기화 (import 시 즉시 실행 X)
function clearIfVersionChanged() {
  try {
    // 저장값이 full version(구버전 포맷, 예: "2.9.0")이어도 major.minor로 정규화 비교 →
    // 기존 사용자 1회성 재표시 회귀 방지.
    const savedMinor = minorVersion(localStorage.getItem(VERSION_KEY))
    if (savedMinor !== APP_MINOR) {
      localStorage.removeItem(ONBOARDED_KEY)
      localStorage.setItem(VERSION_KEY, APP_MINOR)
      console.log(`[Nexus] 버전 변경 (${savedMinor || '없음'} → ${APP_MINOR}): 온보딩 초기화`)
    }
  } catch { /* ignore */ }
}

export function hasCompletedOnboarding(): boolean {
  try {
    clearIfVersionChanged()  // ← 여기서 호출 (import 사이드 이펙트 제거)
    // '1'(현재) 또는 'true'(구버전 appStore.setOnboarded가 남긴 값) 모두 완료로 인정 →
    // 메인 게이트(appStore.isOnboarded = !!value)와 판정 일관성 확보, 온보딩 재표시 방지.
    const v = localStorage.getItem(ONBOARDED_KEY)
    return v === '1' || v === 'true'
  } catch { return false }
}

export function markOnboardingComplete() {
  try {
    localStorage.setItem(ONBOARDED_KEY, '1')
    localStorage.setItem(VERSION_KEY, APP_MINOR)
  } catch { /* ignore */ }
}

interface OnboardingProps {
  onComplete: () => void
  onTryCommand: (cmd: string) => void
}

const STEPS = [
  {
    icon: '🤖',
    title: '직접 PC를 조작하는 AI',
    body: '말만 하는 AI가 아니에요.\n파일 정리·화면 분석·엑셀 작성·예약 실행까지 직접 해드려요.',
    cta: '시작',
    cmd: '',
  },
  {
    icon: '🖥️',
    title: '내 PC 상태 한눈에',
    body: '"PC 상태" 한 마디로 CPU/메모리/디스크 시각화.',
    cta: '실행해 보기',
    cmd: 'PC 상태 보여줘',
  },
  {
    icon: '📊',
    title: '데이터 없어도 만든다',
    body: '"매출 정리 엑셀로" → AI가 가상 샘플 + 차트까지 자동 생성.',
    cta: '실행해 보기',
    cmd: '엑셀로 매출 정리해줘',
  },
  {
    icon: '🤖',
    title: '여러 액션 동시 처리',
    body: '"AI 뉴스 찾고 PDF로 저장" → 검색 + 저장 자동 워크플로.',
    cta: '실행해 보기',
    cmd: 'AI 뉴스 찾아서 PDF로 저장해',
  },
  {
    icon: '🎭',
    title: '19개 직업 페르소나 자동',
    body: '"코드 리뷰해줘" → 개발자 모드 자동.\n"계약서 검토" → 법무 모드 자동.\n\n단축키: Alt+Space (호출) / "헤이 넥서스" (음성)',
    cta: '완료',
    cmd: '',
  },
]

export function Onboarding({ onComplete, onTryCommand }: OnboardingProps) {
  const [step, setStep] = useState(0)

  const next = () => {
    if (step >= STEPS.length - 1) {
      markOnboardingComplete()
      onComplete()
      return
    }
    setStep(step + 1)
  }

  const skip = () => {
    markOnboardingComplete()
    onComplete()
  }

  const tryNow = () => {
    const cmd = STEPS[step].cmd
    if (cmd) {
      onTryCommand(cmd)
      // 다음 스텝으로
      setTimeout(() => next(), 400)
    } else {
      next()
    }
  }

  const s = STEPS[step]

  return (
    <div style={{
      position: 'fixed', inset: 0, zIndex: 9999,
      background: 'rgba(0,0,0,0.55)',
      backdropFilter: 'blur(12px) saturate(1.4)',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      padding: 24,
    }}>
      <AnimatePresence mode="wait">
        <motion.div
          key={step}
          initial={{ opacity: 0, y: 16, scale: 0.96 }}
          animate={{ opacity: 1, y: 0, scale: 1 }}
          exit={{ opacity: 0, y: -16, scale: 0.96 }}
          transition={{ duration: 0.28 }}
          style={{
            width: 'min(480px, 100%)',
            background: 'rgba(15,23,42,0.92)',
            border: '1px solid rgba(255,255,255,0.12)',
            borderTop: '2px solid rgba(139,92,246,0.7)',
            borderRadius: 24,
            padding: '36px 32px 28px',
            boxShadow: '0 24px 64px rgba(0,0,0,0.5), inset 0 1px 0 rgba(255,255,255,0.08)',
            color: '#f1f5f9',
            textAlign: 'center',
          }}
        >
          <div style={{ fontSize: 64, marginBottom: 14, filter: 'drop-shadow(0 6px 24px rgba(139,92,246,0.4))' }}>{s.icon}</div>
          <div style={{ fontSize: 20, fontWeight: 800, marginBottom: 12, letterSpacing: '-0.01em' }}>{s.title}</div>
          <div style={{
            fontSize: 13, color: 'rgba(255,255,255,0.75)', lineHeight: 1.7,
            whiteSpace: 'pre-line', marginBottom: 24,
          }}>{s.body}</div>

          {/* 진행 dot */}
          <div style={{ display: 'flex', justifyContent: 'center', gap: 6, marginBottom: 22 }}>
            {STEPS.map((_, i) => (
              <div key={i} style={{
                width: i === step ? 22 : 6, height: 6, borderRadius: 3,
                background: i === step ? '#a78bfa' : 'rgba(255,255,255,0.18)',
                transition: 'all 0.25s',
              }} />
            ))}
          </div>

          <div style={{ display: 'flex', gap: 10, justifyContent: 'center' }}>
            {step < STEPS.length - 1 && (
              <button
                onClick={skip}
                style={{
                  padding: '10px 18px', borderRadius: 12,
                  background: 'transparent', color: 'rgba(255,255,255,0.55)',
                  border: '1px solid rgba(255,255,255,0.10)',
                  fontSize: 12, fontWeight: 600, cursor: 'pointer',
                }}
              >
                건너뛰기
              </button>
            )}
            <button
              onClick={s.cmd ? tryNow : next}
              style={{
                padding: '10px 24px', borderRadius: 12,
                background: 'linear-gradient(135deg, #6366f1, #8b5cf6)',
                color: '#fff', border: 'none',
                fontSize: 13, fontWeight: 700, cursor: 'pointer',
                boxShadow: '0 8px 24px rgba(139,92,246,0.35)',
              }}
            >
              {s.cta} →
            </button>
          </div>
        </motion.div>
      </AnimatePresence>
    </div>
  )
}
