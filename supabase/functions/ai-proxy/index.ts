// Supabase Edge Function: AI API 프록시
// API 키는 Supabase Secrets에만 저장 — 클라이언트에 절대 노출 안 됨
// 배포: supabase functions deploy ai-proxy

import { createClient } from 'https://esm.sh/@supabase/supabase-js@2'

const GROQ_KEY       = Deno.env.get('GROQ_KEY')!
const PERPLEXITY_KEY = Deno.env.get('PERPLEXITY_KEY') ?? GROQ_KEY  // fallback to Groq
const CLAUDE_KEY     = Deno.env.get('CLAUDE_KEY')!
const TAVILY_KEY     = Deno.env.get('TAVILY_KEY')!

const DAILY_FREE_LIMIT    = 500
const DAILY_PREMIUM_LIMIT = 50

const PREMIUM_ACTIONS = new Set([
  'web_search', 'trip_plan', 'price_compare',
  'video_search', 'workflow_preset', 'multi_action', 'weather',
])

Deno.serve(async (req) => {
  // CORS
  if (req.method === 'OPTIONS') {
    return new Response('ok', {
      headers: {
        'Access-Control-Allow-Origin': '*',
        'Access-Control-Allow-Headers': 'authorization, content-type',
      },
    })
  }

  try {
    // ── 1. 사용자 인증 ──────────────────────────────────────
    const authHeader = req.headers.get('Authorization')
    if (!authHeader) {
      return json({ error: '로그인이 필요합니다.' }, 401)
    }

    // 사용자 인증: anon key + user JWT
    const supabase = createClient(
      Deno.env.get('SUPABASE_URL')!,
      Deno.env.get('SUPABASE_ANON_KEY')!,
      { global: { headers: { Authorization: authHeader } } }
    )
    // 관리 작업(usage_logs 쓰기): service_role key — RLS 우회
    const adminSupabase = createClient(
      Deno.env.get('SUPABASE_URL')!,
      Deno.env.get('SUPABASE_SERVICE_ROLE_KEY')!,
    )

    const { data: { user }, error: authError } = await supabase.auth.getUser()
    if (authError || !user) {
      return json({ error: '인증 실패' }, 401)
    }

    // ── 2. 구독 상태 확인 ────────────────────────────────────
    const { data: sub } = await adminSupabase
      .from('subscriptions')
      .select('status, current_period_end')
      .eq('user_id', user.id)
      .maybeSingle()

    const isActive = sub?.status === 'active' ||
      (sub?.status === 'trial' && sub?.current_period_end && new Date(sub.current_period_end) > new Date())

    if (!isActive) {
      return json({
        error: '구독이 만료되었습니다. 요금제를 확인해주세요.',
        code: 'subscription_expired',
      }, 402)
    }

    // ── 3. 사용량 체크 ───────────────────────────────────────
    const body = await req.json()
    const { action, payload } = body
    const tier = PREMIUM_ACTIONS.has(action) ? 'premium' : 'free'
    const limitCol = tier === 'premium' ? 'premium_count' : 'free_count'
    const dailyLimit = tier === 'premium' ? DAILY_PREMIUM_LIMIT : DAILY_FREE_LIMIT

    const today = new Date().toISOString().split('T')[0]
    const { data: usage } = await adminSupabase
      .from('usage_logs')
      .select('free_count, premium_count')
      .eq('user_id', user.id)
      .eq('date', today)
      .maybeSingle()

    const currentCount = usage?.[limitCol as keyof typeof usage] ?? 0
    if (currentCount >= dailyLimit) {
      return json({
        error: tier === 'premium'
          ? `오늘 프리미엄 요청 ${DAILY_PREMIUM_LIMIT}회를 모두 사용했어요. 내일 자정에 충전됩니다.`
          : `오늘 사용량 ${DAILY_FREE_LIMIT}회를 초과했습니다.`,
        code: 'usage_limit',
        limit: dailyLimit,
        used: currentCount,
      }, 429)
    }

    // ── 4. 사용량 카운트 증가 (service_role — RLS 우회) ────────────────────
    await adminSupabase.from('usage_logs').upsert({
      user_id: user.id,
      date: today,
      [limitCol]: (currentCount as number) + 1,
      updated_at: new Date().toISOString(),
    }, { onConflict: 'user_id,date' })

    // ── 5. 실제 API 호출 ─────────────────────────────────────
    let result: unknown

    switch (action) {
      case 'groq_chat': {
        const res = await fetch('https://api.groq.com/openai/v1/chat/completions', {
          method: 'POST',
          headers: { Authorization: `Bearer ${GROQ_KEY}`, 'Content-Type': 'application/json' },
          body: JSON.stringify(payload),
        })
        result = await res.json()
        break
      }

      case 'perplexity_chat': {
        const res = await fetch('https://api.perplexity.ai/chat/completions', {
          method: 'POST',
          headers: { Authorization: `Bearer ${PERPLEXITY_KEY}`, 'Content-Type': 'application/json' },
          body: JSON.stringify(payload),
        })
        result = await res.json()
        break
      }

      case 'claude_intent': {
        const res = await fetch('https://api.anthropic.com/v1/messages', {
          method: 'POST',
          headers: {
            'x-api-key': CLAUDE_KEY,
            'anthropic-version': '2023-06-01',
            'content-type': 'application/json',
          },
          body: JSON.stringify(payload),
        })
        result = await res.json()
        break
      }

      case 'tavily_search':
      case 'tavily_search_domain': {
        const res = await fetch('https://api.tavily.com/search', {
          method: 'POST',
          headers: { Authorization: `Bearer ${TAVILY_KEY}`, 'Content-Type': 'application/json' },
          body: JSON.stringify(payload),
        })
        result = await res.json()
        break
      }

      case 'vision_analyze': {
        // Groq Vision (llama-4-scout-17b — 멀티모달)
        const { model, messages, max_tokens } = payload as {
          model: string
          messages: unknown[]
          max_tokens: number
        }
        const res = await fetch('https://api.groq.com/openai/v1/chat/completions', {
          method: 'POST',
          headers: { Authorization: `Bearer ${GROQ_KEY}`, 'Content-Type': 'application/json' },
          body: JSON.stringify({ model, messages, max_tokens }),
        })
        result = await res.json()
        break
      }

      case 'claude_vision': {
        const res = await fetch('https://api.anthropic.com/v1/messages', {
          method: 'POST',
          headers: {
            'x-api-key': CLAUDE_KEY,
            'anthropic-version': '2023-06-01',
            'content-type': 'application/json',
          },
          body: JSON.stringify(payload),
        })
        result = await res.json()
        break
      }

      default:
        return json({ error: `알 수 없는 액션: ${action}` }, 400)
    }

    return json({
      success: true,
      result,
      usage: {
        tier,
        used: (currentCount as number) + 1,
        limit: dailyLimit,
        left: dailyLimit - ((currentCount as number) + 1),
      },
    })

  } catch (e) {
    return json({ error: String(e) }, 500)
  }
})

function json(data: unknown, status = 200) {
  return new Response(JSON.stringify(data), {
    status,
    headers: {
      'Content-Type': 'application/json',
      'Access-Control-Allow-Origin': '*',
    },
  })
}
