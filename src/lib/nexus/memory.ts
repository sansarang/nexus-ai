// Nexus 장기 기억 시스템 — 대화 영구 저장 + 사용자 프로필 학습

export interface StoredTurn {
  role: 'user' | 'model'
  text: string
  timestamp: number
}

export interface RecurringPattern {
  trigger: string    // 트리거 키워드 ("주간보고", "매주 월요일")
  action: string     // 실행된 작업 요약
  count: number      // 사용 횟수
  lastUsed: number   // 마지막 사용 timestamp
}

export interface UserProfile {
  preferredCity?: string
  preferredNewsCategory?: string
  preferredStocks?: string[]
  name?: string
  recentTopics?: string[]
  locale?: string
  facts?: string[]
  patterns?: RecurringPattern[]   // 반복 행동 패턴 (최대 20개)
}

const HISTORY_KEY = 'nexus-conversation-history'
const PROFILE_KEY = 'nexus-user-profile'
const MAX_STORED_TURNS = 200      // 영구 저장 최대 200턴
const MAX_CONTEXT_TURNS = 30      // AI에 전달할 최근 N턴

// ── 대화 저장 ────────────────────────────────────────────────

export function saveHistory(history: StoredTurn[]): void {
  try {
    const trimmed = history.slice(-MAX_STORED_TURNS)
    localStorage.setItem(HISTORY_KEY, JSON.stringify(trimmed))
  } catch { /* localStorage 꽉 차면 무시 */ }
}

export function loadHistory(): StoredTurn[] {
  try {
    const raw = localStorage.getItem(HISTORY_KEY)
    if (!raw) return []
    return JSON.parse(raw) as StoredTurn[]
  } catch { return [] }
}

export function clearHistory(): void {
  localStorage.removeItem(HISTORY_KEY)
}

// ── 사용자 프로필 ────────────────────────────────────────────

export function loadProfile(): UserProfile {
  try {
    const raw = localStorage.getItem(PROFILE_KEY)
    if (!raw) return {}
    return JSON.parse(raw) as UserProfile
  } catch { return {} }
}

export function saveProfile(profile: UserProfile): void {
  try {
    localStorage.setItem(PROFILE_KEY, JSON.stringify(profile))
  } catch { /* 무시 */ }
}

export function updateProfile(updates: Partial<UserProfile>): void {
  const current = loadProfile()
  const merged = { ...current, ...updates }
  // recentTopics 최대 10개
  if (merged.recentTopics && merged.recentTopics.length > 10) {
    merged.recentTopics = merged.recentTopics.slice(-10)
  }
  // facts 최대 20개
  if (merged.facts && merged.facts.length > 20) {
    merged.facts = merged.facts.slice(-20)
  }
  saveProfile(merged)
}

// ── 대화에서 자동으로 프로필 학습 ───────────────────────────

export function learnFromTurn(userText: string, _modelText: string): void {
  const profile = loadProfile()
  const updates: Partial<UserProfile> = {}

  // 날씨 도시 학습
  const cityMatch = userText.match(/([가-힣]{2,})\s*(날씨|기온|기상)/)
  if (cityMatch) {
    updates.preferredCity = cityMatch[1]
    updates.locale = cityMatch[1]
  }

  // 이름 학습
  const nameMatch = userText.match(/나는\s+([가-힣A-Za-z]{2,5})(이야|야|라고\s*해|이라고\s*해|입니다)/)
  if (nameMatch) {
    updates.name = nameMatch[1]
    const facts = [...(profile.facts ?? []), `사용자 이름: ${nameMatch[1]}`]
    updates.facts = [...new Set(facts)]
  }

  // 뉴스 카테고리 학습
  const newsMatch = userText.match(/(IT|경제|정치|스포츠|연예|사회|국제|과학)\s*(뉴스|소식)/)
  if (newsMatch) {
    updates.preferredNewsCategory = newsMatch[1]
  }

  // 주식 종목 학습
  const stockMatch = userText.match(/([가-힣A-Za-z0-9]{2,10})\s*(주가|주식|종목)/)
  if (stockMatch) {
    const stocks = [...new Set([...(profile.preferredStocks ?? []), stockMatch[1]])]
    updates.preferredStocks = stocks.slice(-5) // 최근 5개만
  }

  // 관심 주제 추가
  const topics = extractTopics(userText)
  if (topics.length > 0) {
    const existing = profile.recentTopics ?? []
    updates.recentTopics = [...new Set([...existing, ...topics])].slice(-10)
  }

  if (Object.keys(updates).length > 0) {
    updateProfile(updates)
  }
}

function extractTopics(text: string): string[] {
  const topics: string[] = []
  if (/날씨|기온/.test(text)) topics.push('날씨')
  if (/뉴스|소식/.test(text)) topics.push('뉴스')
  if (/주식|주가/.test(text)) topics.push('주식')
  if (/환율|달러|엔화|유로/.test(text)) topics.push('환율')
  if (/스포츠|야구|축구|농구/.test(text)) topics.push('스포츠')
  if (/맛집|식당|카페/.test(text)) topics.push('맛집')
  if (/쇼핑|구매|가격/.test(text)) topics.push('쇼핑')
  if (/건강|운동|다이어트/.test(text)) topics.push('건강')
  if (/여행|관광|숙소/.test(text)) topics.push('여행')
  return topics
}

// ── AI에 전달할 컨텍스트 문자열 생성 ────────────────────────

export function buildMemoryContext(): string {
  const profile = loadProfile()
  const history = loadHistory()
  const lines: string[] = []

  // 사용자 프로필 요약
  if (profile.name) lines.push(`사용자 이름: ${profile.name}`)
  if (profile.preferredCity) lines.push(`자주 묻는 지역: ${profile.preferredCity}`)
  if (profile.preferredNewsCategory) lines.push(`관심 뉴스 분야: ${profile.preferredNewsCategory}`)
  if (profile.preferredStocks?.length) lines.push(`관심 주식: ${profile.preferredStocks.join(', ')}`)
  if (profile.recentTopics?.length) lines.push(`최근 관심 주제: ${profile.recentTopics.join(', ')}`)
  if (profile.facts?.length) lines.push(`기억 사항: ${profile.facts.join(' / ')}`)
  const topPatterns = (profile.patterns ?? []).filter(p => p.count >= 2).slice(0, 5)
  if (topPatterns.length) lines.push(`반복 패턴: ${topPatterns.map(p => `"${p.trigger}"→${p.action}`).join(', ')}`)

  const profileSection = lines.length > 0
    ? `[사용자 기억]\n${lines.join('\n')}`
    : ''

  // 최근 대화 요약 (오래된 대화는 날짜별로 압축)
  const recent = history.slice(-MAX_CONTEXT_TURNS)
  const historySection = recent.length > 0
    ? buildHistorySummary(history.slice(0, -MAX_CONTEXT_TURNS)) + '\n[최근 대화]\n' +
      recent.map(t => `${t.role === 'user' ? '사용자' : 'Nexus'}: ${t.text}`).join('\n')
    : ''

  return [profileSection, historySection].filter(Boolean).join('\n\n')
}

function buildHistorySummary(oldTurns: StoredTurn[]): string {
  if (oldTurns.length === 0) return ''
  // 오래된 대화는 날짜별 주제만 요약
  const grouped: Record<string, string[]> = {}
  for (const t of oldTurns) {
    if (t.role !== 'user') continue
    const date = new Date(t.timestamp).toLocaleDateString('ko-KR')
    grouped[date] = grouped[date] ?? []
    grouped[date].push(t.text.slice(0, 40))
  }
  const summary = Object.entries(grouped)
    .map(([date, texts]) => `${date}: ${texts.slice(0, 3).join(', ')}`)
    .join('\n')
  return summary ? `[과거 대화 요약]\n${summary}` : ''
}

// ── 반복 패턴 학습 ───────────────────────────────────────────

const TIME_TRIGGERS = ['매일', '매주', '매월', '월요일', '화요일', '수요일', '목요일', '금요일', '아침', '저녁', '출근', '퇴근']
const TASK_TRIGGERS = ['주간보고', '일일보고', '브리핑', '회의록', '정리', '요약', '보고서', '메일확인', '일정확인']

export function learnPattern(userText: string, action: string): void {
  const profile = loadProfile()
  const patterns = profile.patterns ?? []
  const matched = [...TIME_TRIGGERS, ...TASK_TRIGGERS].find(t => userText.includes(t))
  if (!matched) return
  const existing = patterns.find(p => p.trigger === matched)
  if (existing) {
    existing.count++
    existing.lastUsed = Date.now()
    existing.action = action
  } else {
    patterns.push({ trigger: matched, action, count: 1, lastUsed: Date.now() })
  }
  updateProfile({ patterns: patterns.sort((a, b) => b.count - a.count).slice(0, 20) })
}

export function getSuggestedPatterns(text: string): RecurringPattern[] {
  const profile = loadProfile()
  return (profile.patterns ?? []).filter(p =>
    p.count >= 2 && [...TIME_TRIGGERS, ...TASK_TRIGGERS].some(t => text.includes(t) && p.trigger === t)
  )
}

// ── ConversationTurn ↔ StoredTurn 변환 ─────────────────────

export interface ConversationTurnLike {
  role: 'user' | 'model'
  parts: Array<{ text: string }>
}

export function toStoredTurns(history: ConversationTurnLike[]): StoredTurn[] {
  return history.map(t => ({
    role: t.role,
    text: t.parts.map(p => p.text).join(''),
    timestamp: Date.now(),
  }))
}

export function fromStoredTurns(stored: StoredTurn[]): ConversationTurnLike[] {
  return stored.map(t => ({
    role: t.role,
    parts: [{ text: t.text }],
  }))
}
