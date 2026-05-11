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
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
