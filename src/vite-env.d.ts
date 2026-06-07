/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_PPLX_KEY: string
  readonly VITE_OPENAI_KEY: string
  readonly VITE_TAVILY_KEY: string
  readonly VITE_ADMIN_EMAIL: string
  readonly VITE_ADMIN_PASSWORD: string
  readonly VITE_SUPABASE_URL: string
  readonly VITE_SUPABASE_ANON_KEY: string
  readonly VITE_PADDLE_CLIENT_TOKEN: string
  readonly VITE_PADDLE_PRICE_ID: string
  readonly VITE_PADDLE_PRICE_YEARLY: string
  readonly VITE_PADDLE_PRICE_PROPLUS: string
  readonly VITE_PADDLE_PRICE_PROPLUS_YEARLY: string
  readonly VITE_PADDLE_PRICE_TEAM5: string
  readonly VITE_PADDLE_PRICE_TEAM10: string
  readonly VITE_PADDLE_PRICE_ENT: string
  readonly VITE_PADDLE_ENVIRONMENT: 'sandbox' | 'production'
  readonly VITE_PADDLE_WEBHOOK_SECRET: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}

// vite define으로 주입되는 빌드 버전 (CI: 2.9.<run_number> / dev: tauri.conf.json)
declare const __APP_VERSION__: string
