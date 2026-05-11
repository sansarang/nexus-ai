/**
 * AvatarEngine — 전체 몸체 캐릭터 렌더링 + 파티클 + 글로우 + 커스텀 이미지
 *
 * 기능:
 *  - 6종 SVG 캐릭터 + 커스텀 이미지 업로드 지원
 *  - Speaking: 음파 + 파티클 + 말풍선 위치 계산
 *  - Listening: 귀 기울임 + 물음표 파티클
 *  - Idle: 부드러운 떠다니기 (float) 애니메이션
 *  - 감정별 배경 글로우 이펙트
 *  - 커스텀 이미지: 원형 마스크 + 동일 애니메이션 적용
 */
import { useEffect, useRef, useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Iron, Luna, Doc, Pixie, Kira, Nova, Sora, Hana, Jin, Mira, Lumi, Joy } from './characters'
import type { CharacterEmotion, CharacterId } from './characters'

interface AvatarEngineProps {
  characterId: CharacterId
  emotion: CharacterEmotion
  speaking: boolean
  listening: boolean
  primaryColor: string
  accentColor: string
  scale?: number
  customImageUrl?: string   // 커스텀 업로드 이미지 URL (base64 or objectURL)
}

const CHARACTER_MAP = {
  iron:   Iron,
  luna:   Luna,
  doc:    Doc,
  pixie:  Pixie,
  kira:   Kira,
  nova:   Nova,
  sora:   Sora,
  hana:   Hana,
  jin:    Jin,
  mira:   Mira,
  lumi:   Lumi,
  joy:    Joy,
} as const

/** 감정별 글로우 색상 */
function emotionGlow(emotion: CharacterEmotion, primary: string): string {
  switch (emotion) {
    case 'happy':    return '#fbbf24'
    case 'alert':    return '#ef4444'
    case 'concerned':return '#f97316'
    case 'humorous': return '#a78bfa'
    default:         return primary
  }
}

/** Speaking 음파 컴포넌트 */
function SoundWaves({ color, active }: { color: string; active: boolean }) {
  if (!active) return null
  return (
    <div style={{ position: 'absolute', bottom: 20, left: '50%', transform: 'translateX(-50%)', display: 'flex', gap: 4, alignItems: 'flex-end' }}>
      {[1, 2, 3, 4, 5, 4, 3, 2, 1].map((h, i) => (
        <motion.div
          key={i}
          animate={{ scaleY: [1, h * 0.8, 1, h * 1.2, 1] }}
          transition={{ duration: 0.5, repeat: Infinity, delay: i * 0.06, ease: 'easeInOut' }}
          style={{
            width: 3,
            height: h * 5 + 4,
            background: color,
            borderRadius: 2,
            opacity: 0.7,
            transformOrigin: 'bottom',
          }}
        />
      ))}
    </div>
  )
}

/** 파티클 시스템 */
interface Particle { id: number; x: number; y: number; size: number; color: string; vx: number; vy: number; life: number }

function ParticleSystem({ active, color, accent }: { active: boolean; color: string; accent: string }) {
  const [particles, setParticles] = useState<Particle[]>([])
  const nextId = useRef(0)

  useEffect(() => {
    if (!active) { setParticles([]); return }
    const interval = setInterval(() => {
      const id = nextId.current++
      const colors = [color, accent, '#ffffff', '#ffd700']
      setParticles(prev => [
        ...prev.slice(-15),
        {
          id,
          x: 30 + Math.random() * 140,
          y: 50 + Math.random() * 200,
          size: 3 + Math.random() * 5,
          color: colors[Math.floor(Math.random() * colors.length)],
          vx: (Math.random() - 0.5) * 2,
          vy: -1 - Math.random() * 2,
          life: 1,
        },
      ])
    }, 120)
    return () => clearInterval(interval)
  }, [active, color, accent])

  return (
    <AnimatePresence>
      {particles.map(p => (
        <motion.div
          key={p.id}
          initial={{ opacity: 0.9, x: p.x, y: p.y, scale: 1 }}
          animate={{ opacity: 0, x: p.x + p.vx * 30, y: p.y + p.vy * 40, scale: 0.2 }}
          exit={{ opacity: 0 }}
          transition={{ duration: 1.2, ease: 'easeOut' }}
          style={{
            position: 'absolute',
            width: p.size,
            height: p.size,
            borderRadius: '50%',
            background: p.color,
            pointerEvents: 'none',
            filter: `drop-shadow(0 0 ${p.size}px ${p.color})`,
          }}
        />
      ))}
    </AnimatePresence>
  )
}

/** 리스닝 파티클 (물음표/점) */
function ListenParticles({ active, color }: { active: boolean; color: string }) {
  if (!active) return null
  return (
    <div style={{ position: 'absolute', top: 0, left: 0, right: 0, bottom: 0, pointerEvents: 'none' }}>
      {['?', '...', '♪', '○'].map((char, i) => (
        <motion.div
          key={i}
          initial={{ opacity: 0, y: 0, x: [-20, 20, -10, 30][i] }}
          animate={{ opacity: [0, 0.8, 0], y: -40 }}
          transition={{ duration: 2, delay: i * 0.5, repeat: Infinity, ease: 'easeOut' }}
          style={{
            position: 'absolute',
            left: `${20 + i * 20}%`,
            bottom: '60%',
            fontSize: 14,
            color,
            fontWeight: 700,
            filter: `drop-shadow(0 0 4px ${color})`,
          }}
        >
          {char}
        </motion.div>
      ))}
    </div>
  )
}

/** 커스텀 이미지 캐릭터 */
function CustomCharacter({ imageUrl, emotion, speaking, listening, primaryColor }: {
  imageUrl: string
  emotion: CharacterEmotion
  speaking: boolean
  listening: boolean
  primaryColor: string
}) {
  return (
    <div style={{ position: 'relative', width: 200, height: 390, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center' }}>
      {/* 몸통 영역 */}
      <motion.div
        animate={{
          y: listening ? [-3, -6] : [0, -6, 0],
          rotate: listening ? -5 : [0, 0.5, 0],
        }}
        transition={listening
          ? { duration: 0.4, ease: 'easeOut' }
          : { duration: 4, repeat: Infinity, ease: 'easeInOut' }
        }
        style={{ position: 'relative' }}
      >
        {/* 원형 마스크 이미지 */}
        <div style={{
          width: 180,
          height: 180,
          borderRadius: '50%',
          overflow: 'hidden',
          border: `3px solid ${primaryColor}88`,
          boxShadow: `0 0 20px ${primaryColor}44`,
        }}>
          <img
            src={imageUrl}
            alt="custom character"
            style={{ width: '100%', height: '100%', objectFit: 'cover' }}
          />
        </div>
        {/* 감정 오버레이 */}
        {emotion === 'happy' && (
          <motion.div
            animate={{ scale: [1, 1.1, 1], opacity: [0.3, 0.6, 0.3] }}
            transition={{ duration: 1, repeat: Infinity }}
            style={{
              position: 'absolute', inset: -5, borderRadius: '50%',
              background: `radial-gradient(circle, ${primaryColor}33 0%, transparent 70%)`,
              pointerEvents: 'none',
            }}
          />
        )}
        {/* 말하는 중 표시 */}
        {speaking && (
          <motion.div
            animate={{ scale: [1, 1.05, 1] }}
            transition={{ duration: 0.4, repeat: Infinity }}
            style={{
              position: 'absolute', bottom: -8, left: '50%', transform: 'translateX(-50%)',
              background: primaryColor,
              borderRadius: 12,
              padding: '3px 10px',
              fontSize: 12,
              color: 'white',
              fontWeight: 600,
              whiteSpace: 'nowrap',
            }}
          >
            🎤 말하는 중...
          </motion.div>
        )}
      </motion.div>
      {/* 하단 플랫폼 */}
      <div style={{
        width: 160,
        height: 10,
        marginTop: 20,
        background: `radial-gradient(ellipse, ${primaryColor}44 0%, transparent 70%)`,
        borderRadius: '50%',
      }} />
    </div>
  )
}

export function AvatarEngine({
  characterId,
  emotion,
  speaking,
  listening,
  primaryColor,
  accentColor,
  scale = 1,
  customImageUrl,
}: AvatarEngineProps) {
  const CharacterComp = CHARACTER_MAP[characterId as keyof typeof CHARACTER_MAP]
  const glowColor = emotionGlow(emotion, primaryColor)
  const isCustom = characterId === 'custom' && customImageUrl

  return (
    <div style={{
      position: 'relative',
      width: 200 * scale,
      height: 430 * scale,
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',
      justifyContent: 'flex-end',
    }}>
      {/* 배경 글로우 (감정 반응형) */}
      <motion.div
        animate={{
          opacity: emotion === 'neutral' ? 0.15 : 0.35,
          scale: speaking ? 1.08 : 1,
        }}
        transition={{ duration: 0.5 }}
        style={{
          position: 'absolute',
          inset: 0,
          background: `radial-gradient(ellipse 80% 70% at 50% 50%, ${glowColor}22 0%, transparent 70%)`,
          pointerEvents: 'none',
          borderRadius: '50%',
        }}
      />

      {/* 파티클 시스템 */}
      <div style={{ position: 'absolute', inset: 0, pointerEvents: 'none', overflow: 'hidden' }}>
        <ParticleSystem active={speaking} color={primaryColor} accent={accentColor} />
        <ListenParticles active={listening} color={primaryColor} />
      </div>

      {/* 캐릭터 본체 */}
      <motion.div
        animate={{
          y: [0, -8, 0],
          filter: `drop-shadow(0 8px 20px ${glowColor}55)`,
        }}
        transition={{ duration: 4, repeat: Infinity, ease: 'easeInOut', times: [0, 0.5, 1] }}
        style={{
          position: 'relative',
          transform: `scale(${scale})`,
          transformOrigin: 'bottom center',
          zIndex: 2,
        }}
      >
        {isCustom ? (
          <CustomCharacter
            imageUrl={customImageUrl!}
            emotion={emotion}
            speaking={speaking}
            listening={listening}
            primaryColor={primaryColor}
          />
        ) : CharacterComp ? (
          <CharacterComp emotion={emotion} speaking={speaking} listening={listening} />
        ) : (
          <Luna emotion={emotion} speaking={speaking} listening={listening} />
        )}
      </motion.div>

      {/* 음파 (Speaking) */}
      <SoundWaves color={primaryColor} active={speaking} />

      {/* 하단 플랫폼 그림자 */}
      <motion.div
        animate={{ scaleX: speaking ? 0.85 : 1, opacity: speaking ? 0.4 : 0.25 }}
        transition={{ duration: 0.3 }}
        style={{
          position: 'absolute',
          bottom: 0,
          left: '50%',
          transform: 'translateX(-50%)',
          width: 120,
          height: 12,
          background: `radial-gradient(ellipse, ${primaryColor}66 0%, transparent 70%)`,
          borderRadius: '50%',
          filter: 'blur(4px)',
        }}
      />

      {/* Alert 상태 — 경고 링 */}
      <AnimatePresence>
        {emotion === 'alert' && (
          <motion.div
            initial={{ scale: 0.8, opacity: 0 }}
            animate={{ scale: [1, 1.3, 1], opacity: [0.8, 0.2, 0.8] }}
            exit={{ scale: 0.8, opacity: 0 }}
            transition={{ duration: 1.5, repeat: Infinity }}
            style={{
              position: 'absolute',
              inset: 20,
              borderRadius: '50%',
              border: '2px solid #ef4444',
              pointerEvents: 'none',
              filter: 'blur(2px)',
            }}
          />
        )}
      </AnimatePresence>
    </div>
  )
}

/** 커스텀 이미지 업로드 유틸 */
export function loadCustomImage(): Promise<string | null> {
  return new Promise(resolve => {
    const input = document.createElement('input')
    input.type = 'file'
    input.accept = 'image/png,image/jpeg,image/gif,image/webp'
    input.onchange = () => {
      const file = input.files?.[0]
      if (!file) { resolve(null); return }
      const reader = new FileReader()
      reader.onload = e => {
        const result = e.target?.result as string
        if (result) {
          localStorage.setItem('nexus-custom-avatar', result)
          resolve(result)
        } else {
          resolve(null)
        }
      }
      reader.readAsDataURL(file)
    }
    input.click()
  })
}

/** 저장된 커스텀 이미지 로드 */
export function getCustomImage(): string | null {
  return localStorage.getItem('nexus-custom-avatar')
}
