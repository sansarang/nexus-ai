export type CharacterId = 'iron' | 'luna' | 'doc' | 'pixie' | 'kira' | 'nova' | 'sora' | 'hana' | 'jin' | 'mira' | 'lumi' | 'joy' | 'custom'
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
  preview?: string   // 미리보기 이미지 (custom 업로드 시 base64)
}

export const CHARACTER_CATALOG: CharacterMeta[] = [
  {
    id: 'luna',
    name: '루나',
    nameEn: 'Luna',
    desc: '캐주얼 친근 비서. 공감 능력이 뛰어나고 항상 옆에 있는 느낌.',
    descEn: 'Casual friendly assistant. Warm & empathetic.',
    tag: '프리랜서 · 학생 · 크리에이터',
    tagEn: 'Freelancer · Student · Creator',
    primaryColor: '#a78bfa',
    accentColor: '#f9a8d4',
    style: 'casual',
  },
  {
    id: 'kira',
    name: '키라',
    nameEn: 'Kira',
    desc: 'K-pop 아이돌 스타일. 화려하고 에너지 넘치는 스타 비서.',
    descEn: 'K-pop idol style. Glamorous & full of energy.',
    tag: 'K-pop 팬 · 엔터테인먼트 · 크리에이터',
    tagEn: 'K-pop Fan · Entertainment · Creator',
    primaryColor: '#ee2b7b',
    accentColor: '#ffd700',
    style: 'kpop',
  },
  {
    id: 'nova',
    name: '노바',
    nameEn: 'Nova',
    desc: 'SF 홀로그래픽 AI. 미래적이고 분석적인 첨단 비서.',
    descEn: 'Holographic SF AI. Futuristic & analytical.',
    tag: '개발자 · 연구자 · 테크 전문가',
    tagEn: 'Dev · Researcher · Tech Pro',
    primaryColor: '#22d3ee',
    accentColor: '#a78bfa',
    style: 'futuristic',
  },
  {
    id: 'doc',
    name: '닥터',
    nameEn: 'Doc',
    desc: '세련된 전문가 비서. 신뢰감 있고 격식 있는 스타일.',
    descEn: 'Professional suit assistant. Trustworthy & formal.',
    tag: '직장인 · 경영자 · 비즈니스',
    tagEn: 'Office · Executive · Business',
    primaryColor: '#1e40af',
    accentColor: '#93c5fd',
    style: 'professional',
  },
  {
    id: 'pixie',
    name: '픽시',
    nameEn: 'Pixie',
    desc: '귀여운 치비 스타일. 밝고 에너지 넘치는 일상 비서.',
    descEn: 'Cute chibi style. Bright & energetic vibe.',
    tag: '일상 · 홈 · 가족',
    tagEn: 'Casual · Home · Family',
    primaryColor: '#f59e0b',
    accentColor: '#fb923c',
    style: 'chibi',
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
    id: 'sora',
    name: '소라',
    nameEn: 'Sora',
    desc: 'K-pop 아이돌 + 귀여운 애니 스타일. 반짝이는 눈과 생동감.',
    descEn: 'K-pop idol + cute anime style. Sparkling eyes & energy.',
    tag: 'K-pop · 아이돌 · 귀여운',
    tagEn: 'K-pop · Idol · Cute',
    primaryColor: '#4fb3e8',
    accentColor: '#ff7eb3',
    style: 'kpop',
  },
  {
    id: 'hana',
    name: '하나',
    nameEn: 'Hana',
    desc: '따뜻하고 부드러운 현실적 스타일. 친근하고 포근한 비서.',
    descEn: 'Warm & soft realistic style. Friendly and cozy assistant.',
    tag: '일상 · 캐주얼 · 따뜻한',
    tagEn: 'Daily · Casual · Warm',
    primaryColor: '#d4956a',
    accentColor: '#8bc34a',
    style: 'casual',
  },
  {
    id: 'jin',
    name: '진우',
    nameEn: 'Jin',
    desc: '다크 헤어 K-pop 남성 아이돌. 세련되고 카리스마 넘치는 비서.',
    descEn: 'Dark-haired K-pop male idol. Stylish & charismatic assistant.',
    tag: 'K-pop · 남성 · 세련된',
    tagEn: 'K-pop · Male · Stylish',
    primaryColor: '#3d5a8a',
    accentColor: '#ffb3c6',
    style: 'kpop',
  },
  {
    id: 'mira',
    name: '미라',
    nameEn: 'Mira',
    desc: '핑크/골드 화려한 K-pop 여성 아이돌. 담대하고 글래머러스한 비서.',
    descEn: 'Pink/gold K-pop female idol. Bold & glamorous assistant.',
    tag: 'K-pop · 화려함 · 아이돌',
    tagEn: 'K-pop · Glamorous · Idol',
    primaryColor: '#ff3d85',
    accentColor: '#f4c430',
    style: 'kpop',
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
    id: 'joy',
    name: '조이',
    nameEn: 'Joy',
    desc: '사이버 블루 쿨한 여성. 강렬하고 에너지 넘치는 비서.',
    descEn: 'Cyber blue cool girl. Intense & energetic assistant.',
    tag: '쿨함 · 사이버 · 에너지',
    tagEn: 'Cool · Cyber · Energy',
    primaryColor: '#00b8e8',
    accentColor: '#4060a8',
    style: 'futuristic',
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
