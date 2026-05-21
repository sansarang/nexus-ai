import { createClient } from '@supabase/supabase-js'
import { SUPABASE_URL, SUPABASE_ANON_KEY } from '../config/services'

const supabaseUrl  = SUPABASE_URL  || 'https://placeholder.supabase.co'
const supabaseKey  = SUPABASE_ANON_KEY || 'placeholder'

export const supabase = createClient(supabaseUrl, supabaseKey, {
  auth: {
    persistSession: true,
    autoRefreshToken: true,
    detectSessionInUrl: true,
  },
})

export type SubscriptionStatus = 'active' | 'trial' | 'expired' | 'none'

export interface SubscriptionRow {
  id: string
  user_id: string
  paddle_subscription_id: string | null
  paddle_customer_id: string | null
  status: SubscriptionStatus
  trial_ends_at: string | null
  current_period_end: string | null
  created_at: string
  updated_at: string
}

const BACKEND = 'http://127.0.0.1:17891'

/**
 * Google OAuth — localhost 콜백 서버 방식
 * 1. redirectTo를 Go 백엔드 로컬 서버로 설정
 * 2. 외부 브라우저에서 Google 로그인
 * 3. 완료 후 localhost로 리다이렉트 → "넥서스 열기" 팝업 없음
 * 4. 프론트가 폴링해서 code 수신 → exchangeCodeForSession
 */
export async function signInWithGoogle(loginHint?: string): Promise<void> {
  const { data, error } = await supabase.auth.signInWithOAuth({
    provider: 'google',
    options: {
      redirectTo: `${BACKEND}/auth/callback`,
      skipBrowserRedirect: true,
      queryParams: {
        access_type: 'offline',
        prompt: 'consent',
        ...(loginHint ? { login_hint: loginHint } : {}),
      },
    },
  })
  if (error) throw error
  if (data.url) {
    const { open } = await import('@tauri-apps/plugin-shell')
    await open(data.url)
    // 백엔드 폴링 시작 (최대 3분, 500ms 간격)
    startOAuthPolling()
  }
}

function startOAuthPolling() {
  const maxAttempts = 360 // 3분
  let attempts = 0
  const timer = setInterval(async () => {
    attempts++
    if (attempts > maxAttempts) { clearInterval(timer); return }
    try {
      const res = await fetch(`${BACKEND}/api/auth/callback/pending`)
      const json = await res.json() as { code?: string }
      if (json.code) {
        clearInterval(timer)
        const { data, error } = await supabase.auth.exchangeCodeForSession(json.code)
        if (!error && data.session) {
          // onAuthStateChange가 SIGNED_IN 이벤트를 받아 자동으로 setLoggedIn 호출
          console.log('[OAuth] 로그인 성공:', data.session.user.email)
        }
      }
    } catch { /* 백엔드 미응답 무시 */ }
  }, 500)
}

/** 로그아웃 */
export async function signOut(): Promise<void> {
  await supabase.auth.signOut()
}

/** 현재 세션의 구독 정보 조회 */
export async function fetchSubscription(userId: string): Promise<SubscriptionRow | null> {
  const { data, error } = await supabase
    .from('subscriptions')
    .select('*')
    .eq('user_id', userId)
    .maybeSingle()
  if (error) {
    console.error('fetchSubscription error:', error)
    return null
  }
  return data as SubscriptionRow | null
}

/** 체험판 구독 생성 (로그인 직후) */
export async function createTrialSubscription(userId: string): Promise<void> {
  const trialEnd = new Date(Date.now() + 3 * 24 * 60 * 60 * 1000).toISOString()
  await supabase.from('subscriptions').upsert(
    {
      user_id: userId,
      status: 'trial',
      trial_ends_at: trialEnd,
      current_period_end: trialEnd,
      updated_at: new Date().toISOString(),
    },
    { onConflict: 'user_id' }
  )
}

/** 구독 상태 계산 (만료 여부 포함) */
export function resolveStatus(row: SubscriptionRow | null): SubscriptionStatus {
  if (!row) return 'none'
  if (row.status === 'active') return 'active'
  if (row.status === 'trial') {
    const end = row.trial_ends_at ? new Date(row.trial_ends_at) : null
    if (end && end < new Date()) return 'expired'
    return 'trial'
  }
  return row.status
}
