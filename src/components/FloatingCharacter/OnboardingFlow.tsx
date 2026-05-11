/**
 * OnboardingFlow v4 — Photorealistic Avatar Onboarding
 *
 * Step 0: 환영 인트로
 * Step 1: 아바타 스타일 선택
 * Step 2: 비서 이름 설정
 * Step 3: 사용자 호칭 설정
 * Step 4: OpenAI API 키 입력
 * Step 5: 구글 계정 로그인 (7일 무료 체험)
 */
import React, { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Avatar3D } from './Avatar3D'
import { REALISTIC_STYLE_PRESETS } from './Avatar3D/Presets'
import type { RealisticStyleId, RealisticStylePreset } from './Avatar3D/Presets'
import type { CharacterPreset } from './Avatar3D'
import { signInWithGoogle } from '../../lib/supabase'
import { ADMIN_EMAIL, ADMIN_PASSWORD } from '../../config/services'

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
const USER_NAMES = ['주인님', '사용자', '선생님', '파트너']

const STEPS_TOTAL = 6

export function OnboardingFlow({ onComplete }: OnboardingFlowProps) {
  const [step, setStep]               = useState(0)
  const [styleId, setStyleId]         = useState<RealisticStyleId>('kpop_star')
  const [assistantName, setName]      = useState('넥서스')
  const [nameInput, setNameInput]     = useState('넥서스')
  const [userInput, setUserInput]     = useState('')
  const [userName, setUserName]       = useState('주인님')
  const [hoverStyle, setHoverStyle]   = useState<RealisticStyleId | null>(null)
  const [openaiKey, setOpenaiKey]     = useState(() => localStorage.getItem('nexus-openai-key') ?? '')
  const [googleLoading, setGoogleLoading] = useState(false)
  const [googleEmail, setGoogleEmail]     = useState(() => localStorage.getItem('nexus-user-email') ?? '')
  const [loginEmail, setLoginEmail]       = useState('')
  const [loginPassword, setLoginPassword] = useState('')
  const [loginError, setLoginError]       = useState('')
  const [showAdminLogin, setShowAdminLogin] = useState(false)


  const selectedStyle = REALISTIC_STYLE_PRESETS.find(s => s.id === styleId) ?? REALISTIC_STYLE_PRESETS[0]

  const handleAdminLogin = () => {
    if (loginEmail.trim() === ADMIN_EMAIL && loginPassword === ADMIN_PASSWORD) {
      localStorage.setItem('nexus-user-email', ADMIN_EMAIL)
      localStorage.setItem('nexus-sub-status', 'active')
      localStorage.setItem('nexus-sub-expiry', '2099-12-31T00:00:00.000Z')
      setGoogleEmail(ADMIN_EMAIL)
      setLoginError('')
      handleComplete(ADMIN_EMAIL)
    } else {
      setLoginError('이메일 또는 비밀번호가 올바르지 않습니다.')
    }
  }

  const handleGoogleLogin = async () => {
    setGoogleLoading(true)
    try {
      await signInWithGoogle()
      // 성공 시 onAuthStateChange → main.tsx bootstrap에서 세션 처리
      // OAuth redirect이므로 페이지 이동됨 — handleComplete는 복귀 후 호출
    } catch (e) {
      // Supabase 미설정 fallback
      console.warn('Google OAuth 미설정, 체험판 시작:', e)
      const trialExpiry = new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString()
      const demoEmail = 'user@gmail.com'
      localStorage.setItem('nexus-user-email', demoEmail)
      localStorage.setItem('nexus-sub-status', 'trial')
      localStorage.setItem('nexus-sub-expiry', trialExpiry)
      setGoogleEmail(demoEmail)
      handleComplete(demoEmail)
    } finally {
      setGoogleLoading(false)
    }
  }

  const handleComplete = (email?: string) => {
    if (openaiKey.trim()) localStorage.setItem('nexus-openai-key', openaiKey.trim())
    const resolvedEmail = email ?? googleEmail
    if (resolvedEmail) localStorage.setItem('nexus-user-email', resolvedEmail)
    onComplete({
      assistantName,
      userName: userName || '주인님',
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
    background: 'rgba(10,10,24,0.98)',
    border: '1px solid rgba(255,255,255,0.09)',
    borderRadius: 28,
    padding: '40px 44px',
    backdropFilter: 'blur(24px)',
    position: 'relative',
    overflow: 'hidden',
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

        {/* ── Step 0: 환영 인트로 ── */}
        {step === 0 && (
          <motion.div
            key="step0"
            initial={{ opacity: 0, y: 30 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            style={{ ...card, overflow: 'visible', padding: '24px 44px 36px' }}
          >
            {/* 상단 3D 아바타 미리보기 */}
            <div style={{ display: 'flex', justifyContent: 'center', marginBottom: 8 }}>
              <div style={{ width: 320, height: 320, position: 'relative' }}>
                <Avatar3D
                  emotion="happy"
                  speaking={false}
                  listening={false}
                  glbUrl={selectedStyle.glbUrl}
                  primaryColor={selectedStyle.primaryColor}
                  accentColor={selectedStyle.accentColor}
                  preset={styleId as CharacterPreset}
                  width={320}
                  height={320}
                  preview
                  scale={0.55}
                  cameraY={0.2}
                  characterOffsetY={-0.3}
                />
              </div>
            </div>

            <div style={{ textAlign: 'center', marginBottom: 32 }}>
              <div style={{
                fontSize: 11, letterSpacing: '0.18em', color: selectedStyle.primaryColor,
                marginBottom: 10, fontWeight: 600, textTransform: 'uppercase',
              }}>
                NEXUS AI · 2026
              </div>
              <h1 style={{
                fontSize: 26, fontWeight: 800, color: 'white',
                marginBottom: 12, letterSpacing: '-0.02em', lineHeight: 1.3,
              }}>
                당신이 말만 하면<br />PC가 알아서 움직이는
              </h1>
              <p style={{ fontSize: 16, color: selectedStyle.primaryColor, fontWeight: 700, lineHeight: 1.5 }}>
                진짜 개인 비서, Nexus 입니다.
              </p>
            </div>

            {nextBtn(() => setStep(1), '시작하기 →')}

            <div style={{
              display: 'flex', justifyContent: 'center', gap: 20,
              marginTop: 20, fontSize: 11, color: 'rgba(255,255,255,0.25)',
            }}>
              {['3D 리얼리스틱 아바타', '음성 인식', 'PC AI 관리'].map(t => (
                <span key={t}>✦ {t}</span>
              ))}
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
              아바타 스타일 선택
            </h2>
            <p style={{ fontSize: 13, color: 'rgba(255,255,255,0.42)', marginBottom: 24, lineHeight: 1.6 }}>
              Photorealistic 3D 아바타 스타일을 선택하세요.
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
                  비서 이름을 설정해주세요
                </h2>
                <p style={{ fontSize: 13, color: 'rgba(255,255,255,0.4)', lineHeight: 1.6 }}>
                  "{nameInput || '넥서스'}아, 지금 몇 시야?" 처럼<br />
                  이름으로 깨울 수 있어요.
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
              {nextBtn(() => { setName(nameInput.trim() || '넥서스'); setStep(3) })}
              {backBtn(() => setStep(1), '← 캐릭터 다시 선택')}
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
              어떻게 불러드릴까요?
            </h2>
            <p style={{ fontSize: 13, color: 'rgba(255,255,255,0.4)', marginBottom: 24, lineHeight: 1.6 }}>
              "{assistantName}이 {userInput || '주인님'}의 PC를 최적화했어요!"<br />
              처럼 알림을 드릴 때 사용해요.
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
              {nextBtn(() => { setUserName(userInput.trim() || '주인님'); setStep(4) }, '다음')}
              {backBtn(() => setStep(2))}
            </div>
          </motion.div>
        )}

        {/* ── Step 4: OpenAI API 키 ── */}
        {step === 4 && (
          <motion.div
            key="step4"
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
                API 키 설정
              </div>
              <h2 style={{ fontSize: 20, fontWeight: 800, color: 'white', marginBottom: 6 }}>
                OpenAI API 키 입력
              </h2>
              <p style={{ fontSize: 12, color: 'rgba(255,255,255,0.4)', lineHeight: 1.6 }}>
                음성(TTS) 기능을 사용하려면 OpenAI 키가 필요합니다.
              </p>
            </div>

            {/* OpenAI 키 */}
            <div style={{ marginBottom: 16 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 6 }}>
                <label style={{ fontSize: 12, fontWeight: 700, color: 'rgba(255,255,255,0.7)' }}>
                  🔊 OpenAI API 키 <span style={{ color: 'rgba(255,255,255,0.3)', fontWeight: 400 }}>(음성 TTS)</span>
                </label>
                {openaiKey.startsWith('sk-') && <span style={{ fontSize: 10, color: '#4ade80' }}>✓ 확인됨</span>}
              </div>
              <input
                value={openaiKey}
                onChange={e => setOpenaiKey(e.target.value)}
                placeholder="sk-..."
                type="password"
                style={{
                  width: '100%', padding: '11px 14px',
                  background: 'rgba(255,255,255,0.05)',
                  border: `1px solid ${openaiKey.startsWith('sk-') ? '#4ade8066' : 'rgba(255,255,255,0.12)'}`,
                  borderRadius: 10, color: 'white', fontSize: 13,
                  outline: 'none', boxSizing: 'border-box', fontFamily: 'monospace',
                } as React.CSSProperties}
              />
              <p style={{ fontSize: 10, color: 'rgba(255,255,255,0.28)', marginTop: 4 }}>
                platform.openai.com → API Keys
              </p>
            </div>

            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              {nextBtn(() => setStep(5), openaiKey.startsWith('sk-') ? '다음 →' : '건너뛰기')}
              {backBtn(() => setStep(3))}
            </div>
          </motion.div>
        )}

        {/* ── Step 5: 구글 로그인 / 7일 무료 체험 ── */}
        {step === 5 && (
          <motion.div
            key="step5"
            initial={{ opacity: 0, y: 30 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            style={card}
          >
            {progressBar}

            <div style={{ textAlign: 'center', marginBottom: 28 }}>
              <div style={{ fontSize: 40, marginBottom: 12 }}>🔐</div>
              <h2 style={{ fontSize: 22, fontWeight: 800, color: 'white', marginBottom: 8 }}>
                시작하기
              </h2>
              <p style={{ fontSize: 13, color: 'rgba(255,255,255,0.4)', lineHeight: 1.6 }}>
                구글 계정으로 로그인하면<br />
                7일 무료 체험이 자동으로 시작됩니다.
              </p>
            </div>

            {/* 로그인 영역 */}
            {!googleEmail ? (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
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
                  {googleLoading ? '연결 중...' : 'Google 계정으로 시작하기'}
                </button>

                {/* 관리자 로그인 토글 */}
                <button
                  onClick={() => setShowAdminLogin(v => !v)}
                  style={{
                    background: 'none', border: 'none', cursor: 'pointer',
                    fontSize: 11, color: 'rgba(255,255,255,0.2)', textAlign: 'center', padding: '4px 0',
                  }}
                >
                  {showAdminLogin ? '▲ 닫기' : '관리자 로그인'}
                </button>

                {showAdminLogin && (
                  <motion.div
                    initial={{ opacity: 0, height: 0 }}
                    animate={{ opacity: 1, height: 'auto' }}
                    style={{ display: 'flex', flexDirection: 'column', gap: 8 }}
                  >
                    <input
                      type="email"
                      placeholder="이메일"
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
                      placeholder="비밀번호"
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
                      로그인
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
                <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.4)', marginTop: 4 }}>7일 무료 체험 시작 준비 완료</div>
              </div>
            )}

            {/* 구독 혜택 */}
            <div style={{
              background: `${selectedStyle.primaryColor}10`,
              border: `1px solid ${selectedStyle.primaryColor}25`,
              borderRadius: 12, padding: '14px 18px', marginTop: 16, fontSize: 12,
              color: 'rgba(255,255,255,0.6)', lineHeight: 2,
            }}>
              {['✦ 모든 AI 기능 무제한 사용', '✦ 실시간 웹 검색 (Perplexity)', '✦ 자동 업데이트', '✦ 7일 후 월 9,900원 · 언제든 해지 가능'].map(t => (
                <div key={t}>{t}</div>
              ))}
            </div>

            {/* 최종 요약 */}
            <div style={{
              background: 'rgba(255,255,255,0.03)', border: '1px solid rgba(255,255,255,0.08)',
              borderRadius: 12, padding: '14px 18px', marginTop: 12, fontSize: 13,
              color: 'rgba(255,255,255,0.7)', lineHeight: 1.8,
            }}>
              <div>비서: <strong style={{ color: 'white' }}>{assistantName}</strong></div>
              <div>호칭: <strong style={{ color: 'white' }}>{userInput || '주인님'}</strong></div>
              <div>캐릭터: <strong style={{ color: selectedStyle.primaryColor }}>{selectedStyle.name}</strong></div>
              <div>음성: <strong style={{ color: openaiKey.startsWith('sk-') ? '#4ade80' : 'rgba(255,255,255,0.4)' }}>
                {openaiKey.startsWith('sk-') ? 'OpenAI TTS ✓' : '기본 음성'}
              </strong></div>
            </div>

            <div style={{ display: 'flex', flexDirection: 'column', gap: 8, marginTop: 16 }}>
              {googleEmail
                ? nextBtn(() => handleComplete(), `${assistantName} 시작하기 ✦`)
                : <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.25)', textAlign: 'center' }}>구글 로그인 후 시작할 수 있습니다</div>
              }
              {backBtn(() => setStep(4))}
            </div>
          </motion.div>
        )}

      </AnimatePresence>
    </div>
  )
}
