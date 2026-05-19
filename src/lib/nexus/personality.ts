const PERSONA_GREETINGS: Record<string, string[]> = {
  developer: [
    '코드 준비됐나요? 리뷰·디버깅·설계 뭐든 도와드릴게요 💻',
    '오늘 어떤 문제를 해결해볼까요? PR 리뷰, 버그 수정, 아키텍처 설계 다 됩니다.',
    '개발 모드 켜졌습니다! 코드 붙여넣으시면 바로 분석해드려요 🔍',
  ],
  marketer: [
    '오늘 트렌드 분석부터 시작할까요? 콘텐츠·캠페인·경쟁사 뭐든 도와드려요 📊',
    '마케팅 인사이트 준비됐어요! SNS 아이디어나 광고 카피 필요하신가요?',
    '데이터 기반 마케팅, 함께 시작해봐요. 오늘 어떤 캠페인을 기획하시나요? 🚀',
  ],
  sales: [
    '오늘 미팅 준비 도와드릴까요? 이메일 초안, 제안서, 고객 조사 다 가능해요 🤝',
    '영업 성과 올릴 준비됐습니다! 어떤 고객사를 공략할 계획인가요?',
    '오늘 클로징 목표는? 협상 자료나 제안서 작성 먼저 시작할까요?',
  ],
  pm: [
    '스프린트 계획이나 PRD 작성 도와드릴까요? 오늘 로드맵 정리부터 시작해요 📋',
    '의사결정 정리, 회의록, 요구사항 분석 — 뭐든 말씀해주세요.',
    'PM 모드 활성화! 우선순위 정리나 리스크 분석이 필요하신가요?',
  ],
  designer: [
    '레퍼런스 수집부터 트렌드 조사까지! 오늘 어떤 프로젝트를 작업 중이신가요? 🎨',
    '디자인 영감이 필요하신가요? 레퍼런스, 컬러, 폰트 추천 바로 도와드려요.',
    '크리에이티브 모드 ON! 오늘 어떤 결과물을 만들어볼까요?',
  ],
  freelancer: [
    '견적서, 계약서, 클라이언트 이메일 — 혼자서도 프로처럼! 뭐부터 도와드릴까요? 🚀',
    '오늘 작업할 클라이언트 프로젝트가 있나요? 효율적으로 처리해드릴게요.',
    '1인 사업의 모든 것, 함께 정리해봐요. 세금, 견적, 일정 관리 다 됩니다.',
  ],
}

export function getGreeting(
  assistantName = 'Nexus',
  userName = '',
  lang: 'ko' | 'en' = 'ko',
  personaId?: string,
): string {
  const hour = new Date().getHours()
  const honorific = userName
    ? (userName.endsWith('님') ? userName : `${userName}님`)
    : '주인님'

  if (lang === 'en') {
    const name = userName || 'there'
    const pool = hour < 12
      ? [`Good morning, ${name}! I'm ${assistantName}. Ready to help!`]
      : hour < 20
      ? [`Hello, ${name}! ${assistantName} at your service.`]
      : [`Working late, ${name}? ${assistantName} is here whenever you need me.`]
    return pool[0]
  }

  // 직업 페르소나 인사
  const storedPersona = personaId ?? (typeof localStorage !== 'undefined' ? localStorage.getItem('nexus-persona-id') ?? '' : '')
  if (storedPersona && PERSONA_GREETINGS[storedPersona]) {
    const pool = PERSONA_GREETINGS[storedPersona]
    const timePrefix = hour < 12 ? '좋은 아침이에요, ' + honorific + '! ' : hour < 20 ? '안녕하세요, ' + honorific + '! ' : '수고 많으세요, ' + honorific + '! '
    const base = pool[Math.floor(Math.random() * pool.length)]
    return timePrefix + base
  }

  // 기본 인사
  const am = [
    `좋은 아침이에요, ${honorific}! ☀️ 오늘도 ${assistantName}가 함께할게요.`,
    `굿모닝 ${honorific}! PC 상태 점검해드릴까요?`,
  ]
  const pm = [
    `안녕하세요, ${honorific}! 무엇을 도와드릴까요? 😊`,
    `네, ${assistantName}입니다! 말씀만 하세요.`,
  ]
  const night = [
    `늦게까지 수고하시네요, ${honorific} 🌙 PC 정리하고 쉬시겠어요?`,
    `야간 작업 중이시군요! 집중 모드 켜드릴까요? 🎯`,
  ]
  const pool = hour < 12 ? am : hour < 20 ? pm : night
  return pool[Math.floor(Math.random() * pool.length)]
}
