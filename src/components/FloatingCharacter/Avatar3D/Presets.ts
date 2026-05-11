/**
 * Presets.ts — Character Style Definitions
 */

export type RealisticStyleId =
  | 'kpop_star'
  | 'expert_professional'
  | 'natural_human'
  | 'creator_streamer'

export interface RealisticStylePreset {
  id: RealisticStyleId
  name: string
  tagline: string
  description: string
  primaryColor: string
  accentColor: string
  skinTone: string
  hairColor: string
  hairStyle: 'long_wavy' | 'bob_sleek' | 'undercut' | 'textured_short' | 'straight_long'
  eyeColor: string
  lipColor: string
  blush: boolean
  isFeminine: boolean
  glbUrl: string        // 캐릭터 GLB 파일 경로
  ttsVoice: string      // OpenAI TTS 보이스
  previewEmoji: string
}

export const REALISTIC_STYLE_PRESETS: RealisticStylePreset[] = [
  {
    id: 'kpop_star',
    name: '우주비행사 아리',
    tagline: '용감함 · 탐험가 · 활기참',
    description: '우주를 누비는 용감한 탐험가. 어떤 미션도 척척 해내는 에너제틱한 비서.',
    primaryColor: '#ff5da2',
    accentColor: '#ffd166',
    skinTone: '#fde4d0',
    hairColor: '#0d0d1a',
    hairStyle: 'long_wavy',
    eyeColor: '#3d1f0d',
    lipColor: '#ff4d7e',
    blush: true,
    isFeminine: true,
    glbUrl: '/char_astronaut.glb',
    ttsVoice: 'nova',
    previewEmoji: '🚀',
  },
  {
    id: 'expert_professional',
    name: '타이거 맥스',
    tagline: '강인함 · 카리스마 · 파워풀',
    description: '강렬한 존재감과 카리스마. 어떤 상황도 압도하는 강력한 비서.',
    primaryColor: '#f97316',
    accentColor: '#fbbf24',
    skinTone: '#e8c8a8',
    hairColor: '#2c1810',
    hairStyle: 'undercut',
    eyeColor: '#1a1a2e',
    lipColor: '#c0785a',
    blush: false,
    isFeminine: false,
    glbUrl: '/char_tiger.glb',
    ttsVoice: 'onyx',
    previewEmoji: '🐯',
  },
  {
    id: 'natural_human',
    name: '숲속 친구 나비',
    tagline: '자연스러움 · 따뜻함 · 친근함',
    description: '자연에서 온 따뜻한 친구. 언제나 편안하고 친근하게 도와주는 비서.',
    primaryColor: '#22c55e',
    accentColor: '#86efac',
    skinTone: '#f5d5b0',
    hairColor: '#6b3a2a',
    hairStyle: 'bob_sleek',
    eyeColor: '#2c3e50',
    lipColor: '#d4826a',
    blush: true,
    isFeminine: true,
    glbUrl: '/char_animal.glb',
    ttsVoice: 'shimmer',
    previewEmoji: '🌿',
  },
  {
    id: 'creator_streamer',
    name: '자유의 날개 노바',
    tagline: '자유로움 · 개성 · 무한한 가능성',
    description: '자유롭게 날아다니는 모험가. 창의적이고 개성 넘치는 만능 비서.',
    primaryColor: '#8b5cf6',
    accentColor: '#e879f9',
    skinTone: '#e8b896',
    hairColor: '#1a1a2e',
    hairStyle: 'textured_short',
    eyeColor: '#1c3a5a',
    lipColor: '#8b5a3a',
    blush: false,
    isFeminine: false,
    glbUrl: '/char_wings.glb',
    ttsVoice: 'alloy',
    previewEmoji: '🦋',
  },
]

export type CharacterPreset = RealisticStyleId

export const styleToPreset = (styleId: RealisticStyleId): RealisticStylePreset =>
  REALISTIC_STYLE_PRESETS.find(s => s.id === styleId) ?? REALISTIC_STYLE_PRESETS[0]
