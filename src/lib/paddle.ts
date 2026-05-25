import { initializePaddle, Paddle } from '@paddle/paddle-js'
import { PADDLE_CLIENT_TOKEN, PADDLE_PRICE_ID, PADDLE_ENVIRONMENT } from '../config/services'

// Paddle price IDs
// pro_monthly: 실제 운영 중
// team_5, team_10, enterprise: Paddle 대시보드에서 플랜 생성 후 아래 값 교체 필요
export const PADDLE_PRICES = {
  pro_monthly: PADDLE_PRICE_ID,
  pro_yearly:  PADDLE_PRICE_ID, // TODO: Paddle 대시보드에서 yearly price 생성 후 VITE_PADDLE_PRICE_YEARLY 추가
  team_5:      PADDLE_PRICE_ID, // TODO: team_5 price → VITE_PADDLE_PRICE_TEAM5
  team_10:     PADDLE_PRICE_ID, // TODO: team_10 price → VITE_PADDLE_PRICE_TEAM10
  enterprise:  PADDLE_PRICE_ID, // TODO: enterprise price → VITE_PADDLE_PRICE_ENT
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
