export const DAILY_FREE_LIMIT = 15
export const MONTHLY_PREMIUM_LIMIT = 2000

interface DailyRecord { count: number; date: string }
interface MonthlyRecord { count: number; month: string }

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

export function incrementDailyUsage(): number {
  const today = new Date().toISOString().slice(0, 10)
  const current = getDailyUsage()
  const newCount = current.count + 1
  try {
    localStorage.setItem('nexus-daily-msgs', JSON.stringify({ count: newCount, date: today }))
  } catch { /* ignore */ }
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
