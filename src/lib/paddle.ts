import { initializePaddle, Paddle } from '@paddle/paddle-js'
import { PADDLE_CLIENT_TOKEN, PADDLE_PRICE_ID, PADDLE_ENVIRONMENT } from '../config/services'

// Paddle price IDs
// pro_monthly: 실제 운영 중 ($19/mo · ₩14,900)
// pro_plus_monthly: 신설 ($49/mo · ₩39,000) — Vision 무제한 + 우선 Agent 패치
// team_*: Paddle 대시보드에서 가격 ID 생성 후 환경변수 갱신
export const PADDLE_PRICES = {
  pro_monthly:       PADDLE_PRICE_ID,
  pro_yearly:        PADDLE_PRICE_ID, // TODO: VITE_PADDLE_PRICE_YEARLY
  pro_plus_monthly:  PADDLE_PRICE_ID, // TODO: VITE_PADDLE_PRICE_PROPLUS — Pro+ $49/mo
  pro_plus_yearly:   PADDLE_PRICE_ID, // TODO: VITE_PADDLE_PRICE_PROPLUS_YEARLY
  team_5:            PADDLE_PRICE_ID, // TODO: VITE_PADDLE_PRICE_TEAM5
  team_10:           PADDLE_PRICE_ID, // TODO: VITE_PADDLE_PRICE_TEAM10
  enterprise:        PADDLE_PRICE_ID, // TODO: VITE_PADDLE_PRICE_ENT
}

// 플랜 메타 (UI 표시용)
export const PLAN_META = {
  free:     { name: 'Free',     priceUSD: 0,  priceKRW: 0,      requests: 15,   highlight: '체험' },
  pro:      { name: 'Pro',      priceUSD: 19, priceKRW: 14900,  requests: 200,  highlight: '인기' },
  pro_plus: { name: 'Pro+',     priceUSD: 49, priceKRW: 39000,  requests: 1000, highlight: '🔥 NEW · Vision 무제한 + 우선 Agent 패치' },
  team:     { name: 'Team',     priceUSD: 99, priceKRW: 79000,  requests: 3000, highlight: '5명+ 워크스페이스' },
}

let paddleInstance: Paddle | undefined

/** Paddle.js 초기화 (앱 시작 시 1회 호출) */
export async function initPaddle(): Promise<Paddle> {
  if (paddleInstance) return paddleInstance
  paddleInstance = await initializePaddle({
    environment: PADDLE_ENVIRONMENT,
    token: PADDLE_CLIENT_TOKEN,
    eventCallback(event) {
      if (event.name === 'checkout.completed') {
        // 결제 완료 — 백엔드 웹훅이 DB를 업데이트할 때까지 잠시 대기 후 새로고침
        setTimeout(() => window.location.reload(), 2500)
      }
    },
  })
  return paddleInstance!
}

/** 구독 결제 체크아웃 열기 (기본 price ID 사용) */
export async function openCheckout(email: string, userId?: string): Promise<void>
/** 특정 priceId로 체크아웃 열기 */
export async function openCheckout(priceId: string, email?: string): Promise<void>
export async function openCheckout(emailOrPriceId: string, userIdOrEmail?: string): Promise<void> {
  const paddle = await initPaddle()
  // Detect if first arg looks like a Paddle price ID (starts with 'pri_')
  const isPriceId = emailOrPriceId.startsWith('pri_')
  const priceId = isPriceId ? emailOrPriceId : PADDLE_PRICE_ID
  const email   = isPriceId ? userIdOrEmail : emailOrPriceId
  paddle.Checkout.open({
    items: [{ priceId, quantity: 1 }],
    customer: email ? { email } : undefined,
    settings: {
      displayMode: 'overlay',
      theme: 'dark',
      locale: 'ko',
    },
  })
}

/** 구독 관리 포털 열기 (결제 수단 변경, 해지 등) */
export async function openBillingPortal(email: string): Promise<void> {
  const paddle = await initPaddle()
  // Paddle Billing: 기존 구독자는 체크아웃을 통해 관리 페이지로 이동
  paddle.Checkout.open({
    items: [{ priceId: PADDLE_PRICE_ID, quantity: 1 }],
    customer: { email },
    settings: {
      displayMode: 'overlay',
      theme: 'dark',
      locale: 'ko',
    },
  })
}
