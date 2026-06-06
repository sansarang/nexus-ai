// ─────────────────────────────────────────────────────────
//  서비스 설정 — 실제 키는 .env 파일에서 관리 (절대 커밋 금지)
// ─────────────────────────────────────────────────────────

// AI API 키 — Supabase Edge Function Secrets에 저장. 프론트는 항상 빈 값.
export const PPLX_API_KEY   = ''
export const OPENAI_API_KEY = ''
export const TAVILY_API_KEY = ''

// 관리자 계정 (패스워드 프론트 노출 금지 — Supabase Auth 사용)
export const ADMIN_EMAIL    = import.meta.env.VITE_ADMIN_EMAIL    as string ?? ''
export const ADMIN_PASSWORD = import.meta.env.VITE_ADMIN_PASSWORD as string ?? ''

// Supabase (배포 전 설정)
export const SUPABASE_URL      = import.meta.env.VITE_SUPABASE_URL      as string ?? ''
export const SUPABASE_ANON_KEY = import.meta.env.VITE_SUPABASE_ANON_KEY as string ?? ''
export const SUPABASE_SERVICE_ROLE_KEY = ''  // Go 백엔드 전용 — 프론트 노출 금지

// Paddle Billing (배포 전 설정)
export const PADDLE_CLIENT_TOKEN   = import.meta.env.VITE_PADDLE_CLIENT_TOKEN as string ?? ''
export const PADDLE_PRICE_ID            = import.meta.env.VITE_PADDLE_PRICE_ID            as string ?? ''
export const PADDLE_PRICE_YEARLY       = import.meta.env.VITE_PADDLE_PRICE_YEARLY       as string ?? ''
export const PADDLE_PRICE_PROPLUS      = import.meta.env.VITE_PADDLE_PRICE_PROPLUS      as string ?? ''
export const PADDLE_PRICE_PROPLUS_YEARLY = import.meta.env.VITE_PADDLE_PRICE_PROPLUS_YEARLY as string ?? ''
export const PADDLE_PRICE_TEAM5        = import.meta.env.VITE_PADDLE_PRICE_TEAM5        as string ?? ''
export const PADDLE_PRICE_TEAM10       = import.meta.env.VITE_PADDLE_PRICE_TEAM10       as string ?? ''
export const PADDLE_PRICE_ENT          = import.meta.env.VITE_PADDLE_PRICE_ENT          as string ?? '' // paddle plan price
export const PADDLE_WEBHOOK_SECRET = import.meta.env.VITE_PADDLE_WEBHOOK_SECRET as string ?? ''  // Go 백엔드 전용
export const PADDLE_ENVIRONMENT: 'sandbox' | 'production' = (import.meta.env.VITE_PADDLE_ENVIRONMENT as string ?? 'production') as 'sandbox' | 'production'
