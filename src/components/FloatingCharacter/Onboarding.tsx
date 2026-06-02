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

export function hasCompletedOnboarding(): boolean {
  try { return localStorage.getItem(ONBOARDED_KEY) === '1' } catch { return false }
}

export function markOnboardingComplete() {
  try { localStorage.setItem(ONBOARDED_KEY, '1') } catch { /* ignore */ }
}

interface OnboardingProps {
  onComplete: () => void
  onTryCommand: (cmd: string) => void
}

const STEPS = [
  {
    icon: '🤖',
    title: 'NEXUS AI에 오신 것을 환영합니다',
    body: '한국어 자비스급 PC 비서.\n명령 60+ 종 / 카드 49+ / 자동 문서 생성.',
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
