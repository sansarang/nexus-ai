import { createClient } from '@supabase/supabase-js'
import { SUPABASE_URL, SUPABASE_ANON_KEY } from '../config/services'

const supabaseUrl  = SUPABASE_URL  || 'https://placeholder.supabase.co'
const supabaseKey  = SUPABASE_ANON_KEY || 'placeholder'

export const supabase = createClient(supabaseUrl, supabaseKey, {
  auth: {
    persistSession: true,
    autoRefreshToken: true,
    detectSessionInUrl: false,
    flowType: 'pkce',
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

export const BACKEND = 'http://127.0.0.1:17891'

/**
 * Google OAuth — Tauri 딥링크 방식 (nexus://auth/callback)
 * Go 백엔드 실행 여부와 무관하게 동작.
 * 1. redirectTo = nexus://auth/callback
 * 2. 외부 브라우저에서 Google 로그인
 * 3. Supabase → nexus://auth/callback?code=XXX
 * 4. OS가 Nexus 앱으로 딥링크 전달
 * 5. Tauri가 oauth-callback 이벤트 발생 → PKCE 코드 교환
 */
export async function signInWithGoogle(onSuccess?: () => void, loginHint?: string): Promise<void> {
  const { data, error } = await supabase.auth.signInWithOAuth({
    provider: 'google',
    options: {
      redirectTo: 'nexus://auth/callback',
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
    await listenForDeepLinkCallback(onSuccess)
  }
}

async function listenForDeepLinkCallback(onSuccess?: () => void): Promise<void> {
  const { listen } = await import('@tauri-apps/api/event')
  let done = false

  // 최대 5분 타임아웃
  const timeout = setTimeout(() => {
    if (!done) { done = true; console.warn('[OAuth] 딥링크 타임아웃') }
  }, 5 * 60 * 1000)

  const unlisten = await listen<string>('oauth-callback', async (event) => {
    if (done) return
    done = true
    clearTimeout(timeout)
    unlisten()

    const url = event.payload
    console.log('[OAuth] 딥링크 수신:', url)

    // query param에서 code 추출 (PKCE: nexus://auth/callback?code=XXX)
    const queryStr = url.includes('?') ? url.split('?')[1].split('#')[0] : ''
    const params = new URLSearchParams(queryStr)
    const code = params.get('code')

    // fragment에서 access_token 추출 (implicit fallback)
    const hashStr = url.includes('#') ? url.split('#')[1] : ''
    const hashParams = new URLSearchParams(hashStr)
    const accessToken = hashParams.get('access_token')
    const refreshToken = hashParams.get('refresh_token')

    if (code) {
      const { error } = await supabase.auth.exchangeCodeForSession(code)
      if (!error) {
        console.log('[OAuth] PKCE 로그인 성공')
        onSuccess?.()
      } else {
        console.error('[OAuth] 코드 교환 실패:', error)
      }
    } else if (accessToken && refreshToken) {
      const { error } = await supabase.auth.setSession({ access_token: accessToken, refresh_token: refreshToken })
      if (!error) {
        console.log('[OAuth] implicit 로그인 성공')
        onSuccess?.()
      }
    } else {
      console.error('[OAuth] 콜백 URL에서 토큰/코드를 찾을 수 없음:', url)
    }
  })
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
