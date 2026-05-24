import { createClient } from '@supabase/supabase-js'
import { SUPABASE_URL, SUPABASE_ANON_KEY } from '../config/services'

const supabaseUrl  = SUPABASE_URL  || 'https://placeholder.supabase.co'
const supabaseKey  = SUPABASE_ANON_KEY || 'placeholder'

export const supabase = createClient(supabaseUrl, supabaseKey, {
  auth: {
    persistSession: true,
    autoRefreshToken: true,
    detectSessionInUrl: false,
    flowType: 'implicit',
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
 * Google OAuth — Rust 인증 서버(17891) 콜백 방식
 * Rust가 포트 17891을 직접 소유하므로 Go 백엔드 실행 여부 무관.
 * Chrome "앱 열기" 다이얼로그 없음.
 */
export async function signInWithGoogle(loginHint?: string): Promise<void> {
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
  }
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

export interface UserSettings {
  assistant_name: string
  user_name: string
  user_lang: string
  primary_color: string
  accent_color: string
  glb_url: string
  preset: string
  tts_voice: string
  character_id: string
  is_onboarded: boolean
}

/** 사용자 설정 저장 */
export async function saveUserSettings(userId: string, settings: Partial<UserSettings>): Promise<void> {
  await supabase.from('user_settings').upsert(
    { user_id: userId, ...settings, updated_at: new Date().toISOString() },
    { onConflict: 'user_id' }
  )
}

/** 사용자 설정 불러오기 */
export async function fetchUserSettings(userId: string): Promise<UserSettings | null> {
  const { data, error } = await supabase
    .from('user_settings')
    .select('*')
    .eq('user_id', userId)
    .maybeSingle()
  if (error) return null
  return data as UserSettings | null
}

/** 구독 상태 계산 (만료 여부 포함) */
export function resolveStatus(row: SubscriptionRow | null): SubscriptionStatus {
  if (!row) return 'trial'  // 조회 실패 시 trial로 간주 (false positive 방지)
  if (row.status === 'active') return 'active'
  if (row.status === 'trial') {
    const end = row.trial_ends_at ? new Date(row.trial_ends_at) : null
    if (end && end < new Date()) return 'expired'
    return 'trial'
  }
  return row.status
}
