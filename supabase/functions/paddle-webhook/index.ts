import { createClient } from 'https://esm.sh/@supabase/supabase-js@2'

const supabase = createClient(
  Deno.env.get('SUPABASE_URL')!,
  Deno.env.get('SUPABASE_SERVICE_ROLE_KEY')!,
)

Deno.serve(async (req) => {
  // Paddle은 POST로 웹훅을 보냄
  if (req.method !== 'POST') {
    return new Response('Method not allowed', { status: 405 })
  }

  let body: Record<string, unknown>
  try {
    body = await req.json()
  } catch {
    return new Response('Invalid JSON', { status: 400 })
  }

  const eventType = body.event_type as string
  const data = body.data as Record<string, unknown>

  console.log('Paddle webhook:', eventType, JSON.stringify(data))

  // ── 구독 활성화 (결제 완료) ──────────────────────────────────
  if (
    eventType === 'subscription.activated' ||
    eventType === 'subscription.updated' ||
    eventType === 'transaction.completed'
  ) {
    const customData = (data.custom_data ?? data.subscription?.custom_data ?? {}) as Record<string, string>
    const userId = customData.user_id as string | undefined
    const paddleSubId = (data.id ?? data.subscription_id) as string | undefined
    const paddleCustomerId = (data.customer_id ?? data.subscription?.customer_id) as string | undefined
    const currentPeriodEnd = (data.current_billing_period as Record<string, string> | undefined)?.ends_at

    if (!userId) {
      console.warn('user_id not found in custom_data')
      return new Response('ok', { status: 200 })
    }

    const { error } = await supabase
      .from('subscriptions')
      .upsert(
        {
          user_id: userId,
          paddle_subscription_id: paddleSubId ?? null,
          paddle_customer_id: paddleCustomerId ?? null,
          status: 'active',
          trial_ends_at: null,
          current_period_end: currentPeriodEnd ?? null,
          updated_at: new Date().toISOString(),
        },
        { onConflict: 'user_id' },
      )

    if (error) {
      console.error('DB upsert error:', error)
      return new Response('DB error', { status: 500 })
    }
  }

  // ── 구독 취소 / 만료 ──────────────────────────────────────────
  if (
    eventType === 'subscription.canceled' ||
    eventType === 'subscription.paused' ||
    eventType === 'subscription.past_due'
  ) {
    const customData = (data.custom_data ?? {}) as Record<string, string>
    const userId = customData.user_id as string | undefined

    if (userId) {
      await supabase
        .from('subscriptions')
        .update({ status: 'expired', updated_at: new Date().toISOString() })
        .eq('user_id', userId)
    }
  }

  return new Response('ok', { status: 200 })
})
