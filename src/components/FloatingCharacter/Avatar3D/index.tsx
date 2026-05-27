/**
 * Avatar3D — 2D SVG AvatarEngine 래퍼 (3D WebGL 경로 제거)
 * Three.js/Canvas 의존성 없이 AvatarEngine(SVG) 기반으로 단순화.
 */
import React from 'react'
import { AvatarEngine } from '../AvatarEngine'
import type { CharacterId } from '../characters'

// 하위 호환 타입 — 기존 코드가 import 하던 타입 유지
export type AvatarEmotion = 'neutral' | 'happy' | 'concerned' | 'surprised' | 'alert'
export type CharacterPreset = 'kpop_star' | 'expert_professional' | 'natural_human' | 'creator_streamer'

export interface Avatar3DProps {
  glbUrl?: string | null
  emotion: AvatarEmotion
  speaking: boolean
  listening: boolean
  primaryColor: string
  accentColor: string
  preset?: CharacterPreset
  scale?: number
  preview?: boolean
  width?: number | string
  height?: number | string
  quality?: 'high' | 'balanced'
  cameraY?: number
  characterOffsetY?: number
}

// AvatarEmotion → CharacterEmotion 매핑 (surprised → alert)
function toCharacterEmotion(e: AvatarEmotion): 'neutral' | 'happy' | 'concerned' | 'alert' | 'humorous' {
  if (e === 'surprised') return 'alert'
  return e as 'neutral' | 'happy' | 'concerned' | 'alert'
}

export function Avatar3D({
  emotion,
  speaking,
  listening,
  primaryColor,
  accentColor,
  scale = 1,
  width = 200,
  height = 340,
}: Avatar3DProps) {
  // localStorage에서 캐릭터 ID 읽기 (없으면 nexus 기본값)
  let characterId: CharacterId = 'nexus'
  try {
    const stored = localStorage.getItem('nexus-character-id') as CharacterId | null
    if (stored === 'iron' || stored === 'lumi' || stored === 'nexus') {
      characterId = stored
    }
  } catch { /* SSR/no-localStorage 환경 무시 */ }

  return (
    <div style={{ width, height, position: 'relative', display: 'flex', alignItems: 'flex-end', justifyContent: 'center' }}>
      <AvatarEngine
        characterId={characterId}
        emotion={toCharacterEmotion(emotion)}
        speaking={speaking}
        listening={listening}
        primaryColor={primaryColor}
        accentColor={accentColor}
        scale={scale}
      />
    </div>
  )
}

// SpeakingWaves — CSS 음파 (3D 불필요)
export function SpeakingWaves({ color, active }: { color: string; active: boolean }) {
  if (!active) return null
  return (
    <div style={{
      position: 'absolute', bottom: 12, left: '50%',
      transform: 'translateX(-50%)',
      display: 'flex', gap: 3, alignItems: 'flex-end',
    }}>
      {[0.4, 0.7, 1.0, 0.7, 0.4].map((h, i) => (
        <div
          key={i}
          style={{
            width: 3, borderRadius: 2,
            background: color,
            animation: `wave3d ${0.5 + i * 0.1}s ease-in-out infinite alternate`,
            height: 4 + h * 12,
          }}
        />
      ))}
      <style>{`
        @keyframes wave3d {
          from { transform: scaleY(0.3); opacity: 0.4; }
          to   { transform: scaleY(1);   opacity: 1; }
        }
      `}</style>
    </div>
  )
}

// 하위 호환 — ProceduralHumanoid/KPOP_PRESETS re-export 스텁
export const KPOP_PRESETS: Record<string, string> = {}
export const ProceduralHumanoid = Avatar3D
