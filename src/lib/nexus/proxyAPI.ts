/**
 * Nexus Proxy API — Supabase ai-proxy Edge Function 연결
 *
 * Pro 구독 중: 모든 LLM 호출이 서버 사이드 키로 처리
 * Free / BYOK: localStorage 키로 직접 호출
 */

import { supabase } from '../supabase'
import { SUPABASE_URL } from '../../config/services'
import { useAppStore } from '../../stores/appStore'

const PROXY_URL = SUPABASE_URL ? `${SUPABASE_URL}/functions/v1/ai-proxy` : ''

export function isProActive(): boolean {
  const { subscriptionStatus } = useAppStore.getState()
  return subscriptionStatus === 'active' || subscriptionStatus === 'trial'
}

/**
 * Supabase ai-proxy를 통해 API 호출.
 * Pro 유저의 JWT를 Authorization 헤더로 전달.
 */
export async function callProxy(action: string, payload: unknown): Promise<unknown> {
  if (!PROXY_URL) throw new Error('Supabase not configured')

  const { data: { session } } = await supabase.auth.getSession()
  if (!session) throw new Error('no session')

  const res = await fetch(PROXY_URL, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${session.access_token}`,
    },
    body: JSON.stringify({ action, payload }),
    signal: AbortSignal.timeout(15000),
  })

  if (!res.ok) {
    const err = await res.json().catch(() => ({})) as { error?: string; code?: string }
    if (err.code === 'subscription_expired') throw new Error('subscription_required')
    if (err.code === 'usage_limit') throw new Error('usage_limit')
    throw new Error(err.error ?? `proxy ${res.status}`)
  }

  const data = await res.json() as { success: boolean; result: unknown }
  return data.result
}
