export const DAILY_FREE_LIMIT = 15
export const MONTHLY_PREMIUM_LIMIT = 2000

interface DailyRecord { count: number; date: string }
interface MonthlyRecord { count: number; month: string }

const BACKEND = 'http://127.0.0.1:17891'

// ── 로컬스토리지 (UI 캐시 용도만) ─────────────────────────────────

export function getDailyUsage(): DailyRecord {
  const today = new Date().toISOString().slice(0, 10)
  try {
    const raw = localStorage.getItem('nexus-daily-msgs')
    if (raw) {
      const parsed = JSON.parse(raw) as DailyRecord
      if (parsed.date === today) return parsed
    }
  } catch { /* ignore */ }
  return { count: 0, date: today }
}

export function setDailyUsageCache(count: number) {
  const today = new Date().toISOString().slice(0, 10)
  try {
    localStorage.setItem('nexus-daily-msgs', JSON.stringify({ count, date: today }))
  } catch { /* ignore */ }
}

export function incrementDailyUsage(): number {
  const today = new Date().toISOString().slice(0, 10)
  const current = getDailyUsage()
  const newCount = current.count + 1
  setDailyUsageCache(newCount)
  return newCount
}

export function getMonthlyUsage(): MonthlyRecord {
  const month = new Date().toISOString().slice(0, 7)
  try {
    const raw = localStorage.getItem('nexus-monthly-msgs')
    if (raw) {
      const parsed = JSON.parse(raw) as MonthlyRecord
      if (parsed.month === month) return parsed
    }
  } catch { /* ignore */ }
  return { count: 0, month }
}

export function incrementMonthlyUsage(): number {
  const month = new Date().toISOString().slice(0, 7)
  const current = getMonthlyUsage()
  const newCount = current.count + 1
  try {
    localStorage.setItem('nexus-monthly-msgs', JSON.stringify({ count: newCount, month }))
  } catch { /* ignore */ }
  return newCount
}

// ── 서버 측 사용량 동기화 ──────────────────────────────────────────

interface ServerUsage {
  used: number
  limit: number
  allowed: boolean
  plan: string
  reset_at: string
}

/** 서버에서 실제 사용량을 가져와 localStorage 캐시를 교정한다. */
export async function syncUsageFromServer(): Promise<ServerUsage | null> {
  try {
    const res = await fetch(`${BACKEND}/api/usage/ai`, { method: 'GET' })
    if (!res.ok) return null
    const data = await res.json() as ServerUsage
    setDailyUsageCache(data.used)
    return data
  } catch {
    return null
  }
}

/** 서버 카운터를 증가시키고 새 사용량을 반환. 허용 여부도 포함. */
export async function incrementServerUsage(): Promise<{ allowed: boolean; used: number; limit: number }> {
  try {
    const res = await fetch(`${BACKEND}/api/usage/ai`, { method: 'POST' })
    if (!res.ok) return { allowed: false, used: DAILY_FREE_LIMIT, limit: DAILY_FREE_LIMIT }
    const data = await res.json() as ServerUsage
    setDailyUsageCache(data.used)
    return { allowed: data.allowed, used: data.used, limit: data.limit }
  } catch {
    // 백엔드 연결 실패 시 로컬 캐시로 폴백
    const local = getDailyUsage()
    return { allowed: local.count < DAILY_FREE_LIMIT, used: local.count, limit: DAILY_FREE_LIMIT }
  }
}
