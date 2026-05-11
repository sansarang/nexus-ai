import { initializePaddle, Paddle } from '@paddle/paddle-js'
import { PADDLE_CLIENT_TOKEN, PADDLE_PRICE_ID, PADDLE_ENVIRONMENT } from '../config/services'

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

/** 구독 결제 체크아웃 열기 */
export async function openCheckout(email: string): Promise<void> {
  const paddle = await initPaddle()
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
