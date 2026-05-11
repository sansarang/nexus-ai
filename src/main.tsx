import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import './index.css'
import { supabase, fetchSubscription, createTrialSubscription, resolveStatus } from './lib/supabase'
import { initPaddle } from './lib/paddle'

/* Tauri 이벤트 수신 — Alt+Space 시 Command Palette 열기 */
async function setupTauriEvents() {
  try {
    const { listen } = await import('@tauri-apps/api/event')
    const { useAppStore } = await import('./stores/appStore')
    await listen('toggle-command', () => {
      useAppStore.getState().toggleCommand()
    })
  } catch {
    /* 브라우저 개발 환경에선 무시 */
  }
}

/* Supabase 세션 복원 + Paddle 초기화 */
async function bootstrap() {
  // Paddle 비동기 초기화 (백그라운드, 실패해도 무시)
  initPaddle().catch(() => {})

  const { useAppStore } = await import('./stores/appStore')

  try {
    const { data: { session } } = await supabase.auth.getSession()

    if (session?.user) {
      const user = session.user
      const email = user.email ?? ''
      let row = await fetchSubscription(user.id)
      if (!row) {
        await createTrialSubscription(user.id)
        row = await fetchSubscription(user.id)
      }
      const status = resolveStatus(row)
      const expiry = row?.current_period_end ?? row?.trial_ends_at ?? ''
      useAppStore.getState().setLoggedIn(email, status, expiry)
    }

    // 세션 변경 감지 (로그인/로그아웃)
    supabase.auth.onAuthStateChange(async (event, newSession) => {
      try {
        if (event === 'SIGNED_IN' && newSession?.user) {
          const user = newSession.user
          const email = user.email ?? ''
          let row = await fetchSubscription(user.id)
          if (!row) {
            await createTrialSubscription(user.id)
            row = await fetchSubscription(user.id)
          }
          const status = resolveStatus(row)
          const expiry = row?.current_period_end ?? row?.trial_ends_at ?? ''
          useAppStore.getState().setLoggedIn(email, status, expiry)
        } else if (event === 'SIGNED_OUT') {
          localStorage.removeItem('nexus-user-email')
          localStorage.removeItem('nexus-sub-status')
          localStorage.removeItem('nexus-sub-expiry')
          useAppStore.setState({ isLoggedIn: false, userEmail: '', subscriptionStatus: 'none', subscriptionExpiry: '' })
        }
      } catch { /* 무시 */ }
    })
  } catch {
    // Supabase 미설정 시 localStorage 값으로 폴백 (개발/오프라인 모드)
    console.info('[Nexus] Supabase 미연결 — localStorage 모드로 동작합니다.')
  }
}

setupTauriEvents()
bootstrap()

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
)
