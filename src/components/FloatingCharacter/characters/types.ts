export type CharacterId = 'iron' | 'lumi' | 'nexus' | 'custom'
export type CharacterEmotion = 'neutral' | 'happy' | 'concerned' | 'alert' | 'humorous'

export interface CharacterProps {
  emotion: CharacterEmotion
  speaking: boolean
  listening: boolean
}

export interface CharacterMeta {
  id: CharacterId
  name: string
  nameEn: string
  desc: string
  descEn: string
  tag: string
  tagEn: string
  primaryColor: string
  accentColor: string
  style: 'sf' | 'casual' | 'professional' | 'chibi' | 'kpop' | 'futuristic' | 'custom'
  preview?: string
}

export const CHARACTER_CATALOG: CharacterMeta[] = [
  {
    id: 'nexus',
    name: '넥서스',
    nameEn: 'Nexus',
    desc: 'SF 홀로그래픽 AI. 미래적이고 분석적인 Nexus 기본 캐릭터.',
    descEn: 'Holographic SF AI. Futuristic & analytical. Default Nexus character.',
    tag: '기본 · SF · 분석가',
    tagEn: 'Default · SF · Analyst',
    primaryColor: '#22d3ee',
    accentColor: '#a78bfa',
    style: 'futuristic',
  },
  {
    id: 'lumi',
    name: '루미',
    nameEn: 'Lumi',
    desc: '전통 현대 퓨전 엘리건트 여성. 우아하고 신비로운 분위기.',
    descEn: 'Traditional-modern fusion elegant girl. Graceful & mysterious.',
    tag: '엘리건트 · 전통 · 퓨전',
    tagEn: 'Elegant · Traditional · Fusion',
    primaryColor: '#4a2468',
    accentColor: '#c060e8',
    style: 'professional',
  },
  {
    id: 'iron',
    name: '아이언',
    nameEn: 'Iron',
    desc: '정밀한 SF AI. 논리적이고 구조적인 분석가 스타일.',
    descEn: 'Precise SF AI. Logical & analytical.',
    tag: '개발자 · 엔지니어 · 연구자',
    tagEn: 'Dev · Engineer · Researcher',
    primaryColor: '#4f7ef7',
    accentColor: '#7dd3fc',
    style: 'sf',
  },
  {
    id: 'custom',
    name: '커스텀',
    nameEn: 'Custom',
    desc: '내가 직접 올린 이미지로 나만의 비서를 만들어보세요!',
    descEn: 'Upload your own image to create a custom assistant!',
    tag: '나만의 스타일 · 완전 커스텀',
    tagEn: 'My Style · Fully Custom',
    primaryColor: '#6b7280',
    accentColor: '#9ca3af',
    style: 'custom',
  },
]
