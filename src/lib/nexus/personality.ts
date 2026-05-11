export function getGreeting(
  assistantName = 'Nexus',
  userName = '',
  lang: 'ko' | 'en' = 'ko',
): string {
  const hour = new Date().getHours()
  // "주인님"처럼 이미 님이 포함된 경우 중복 방지
  const honorific = userName
    ? (userName.endsWith('님') ? userName : `${userName}님`)
    : '주인님'

  if (lang === 'en') {
    const name = userName || 'there'
    const en_am = [
      `Good morning, ${name}! I'm ${assistantName}. Ready to help!`,
      `Morning, ${name}! How can ${assistantName} assist you today?`,
    ]
    const en_pm = [
      `Hello, ${name}! ${assistantName} at your service.`,
      `Hi ${name}! What can I do for you?`,
    ]
    const en_night = [
      `Working late, ${name}? ${assistantName} is here whenever you need me.`,
      `Still up, ${name}? Let me know if you need anything!`,
    ]
    const pool = hour < 12 ? en_am : hour < 20 ? en_pm : en_night
    return pool[Math.floor(Math.random() * pool.length)]
  }

  const am = [
    `좋은 아침이에요, ${honorific}! ☀️ 오늘도 ${assistantName}가 함께할게요.`,
    `굿모닝 ${honorific}! PC 상태 점검해드릴까요?`,
    `일어나셨군요, ${honorific}! 오늘 하루도 잘 부탁드려요 😊`,
  ]
  const pm = [
    `안녕하세요, ${honorific}! 무엇을 도와드릴까요? 😊`,
    `네, ${assistantName}입니다! 말씀만 하세요.`,
    `안녕하세요, ${honorific}! 뭐든지 말씀만 하시면 됩니다.`,
  ]
  const night = [
    `늦게까지 수고하시네요, ${honorific} 🌙 PC 정리하고 쉬시겠어요?`,
    `야간 작업 중이시군요! 집중 모드 켜드릴까요? 🎯`,
  ]
  const pool = hour < 12 ? am : hour < 20 ? pm : night
  return pool[Math.floor(Math.random() * pool.length)]
}
