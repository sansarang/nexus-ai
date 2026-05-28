import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import { ErrorBoundary } from './components/ErrorBoundary'
import { inject } from '@vercel/analytics'
import './index.css'

inject()
import { supabase, fetchSubscription, createTrialSubscription, resolveStatus, fetchUserSettings } from './lib/supabase'
import { initPaddle } from './lib/paddle'

// oauth-callback 처리 중 플래그 — onAuthStateChange 중복 처리 방지
let _oauthProcessing = false

/* Tauri 이벤트 수신 — Alt+Space 시 Command Palette 열기 + OAuth 콜백 처리 */
async function setupTauriEvents() {
  try {
    const { listen } = await import('@tauri-apps/api/event')
    const { useAppStore } = await import('./stores/appStore')

    await listen('toggle-command', () => {
      useAppStore.getState().toggleCommand()
    })

    // Alt+S: 스크린샷 + AI 분석 → 채팅창에 "화면 분석해줘" 명령 전송
    await listen('shortcut-screenshot', () => {
      useAppStore.getState().triggerCommand?.('화면 분석해줘')
    })

    // Alt+V: 비전 / OCR → 클립보드 이미지 OCR
    await listen('shortcut-vision', () => {
      useAppStore.getState().triggerCommand?.('클립보드 이미지 OCR 해줘')
    })

    // Alt+C: 클립보드 히스토리
    await listen('shortcut-clipboard', () => {
      useAppStore.getState().triggerCommand?.('클립보드 히스토리 보여줘')
    })

    // Google OAuth 딥링크 콜백 처리
    await listen('oauth-callback', async (event) => {
      try {
        let raw = event.payload as string
        // payload가 JSON 배열 형태로 올 수 있음: ["nexus://..."]
        if (typeof raw === 'string' && raw.trimStart().startsWith('[')) {
          raw = (JSON.parse(raw) as string[])[0]
        }
        // 앞뒤 따옴표 제거
        raw = raw.replace(/^"|"$/g, '')
        const urlObj = new URL(raw.replace('nexus://', 'https://nexus.app/'))
        const code = urlObj.searchParams.get('code')
        if (code) {
          _oauthProcessing = true
          try {
            const { supabase, fetchSubscription, createTrialSubscription, resolveStatus } = await import('./lib/supabase')
            const { data, error } = await supabase.auth.exchangeCodeForSession(code)
            if (!error && data.session?.user) {
              const user = data.session.user
              const email = user.email ?? ''
              let row = await fetchSubscription(user.id)
              if (!row) {
                await createTrialSubscription(user.id)
                row = await fetchSubscription(user.id)
              }
              const status = resolveStatus(row)
              const expiry = row?.current_period_end ?? row?.trial_ends_at ?? ''
              useAppStore.getState().setLoggedIn(email, status, expiry, user.id)
            }
          } finally {
            setTimeout(() => { _oauthProcessing = false }, 3000)
          }
        }
      } catch { /* 무시 */ }
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
      useAppStore.getState().setLoggedIn(email, status, expiry, user.id)
    }

    // 세션 변경 감지 (로그인/로그아웃)
    supabase.auth.onAuthStateChange(async (event, newSession) => {
      try {
        if (event === 'SIGNED_IN' && newSession?.user) {
          // oauth-callback 리스너가 이미 처리 중이면 중복 실행 방지
          if (_oauthProcessing) return

          const user = newSession.user
          const email = user.email ?? ''
          // 일단 로그인 처리 먼저 — 구독 조회 실패해도 앱 진입 가능
          useAppStore.getState().setLoggedIn(email, 'trial', '', user.id)
          // 설정 복원 (Supabase → localStorage → store)
          try {
            const settings = await fetchUserSettings(user.id)
            if (settings?.is_onboarded) {
              if (settings.assistant_name) { localStorage.setItem('nexus-assistant-name', settings.assistant_name); useAppStore.getState().setAssistantName(settings.assistant_name) }
              if (settings.user_name) { localStorage.setItem('nexus-user-name', settings.user_name); useAppStore.getState().setUserName(settings.user_name) }
              if (settings.user_lang) { localStorage.setItem('nexus-lang', settings.user_lang); useAppStore.getState().setUserLang(settings.user_lang as 'ko' | 'en') }
              if (settings.primary_color) { localStorage.setItem('nexus-primary-color', settings.primary_color); useAppStore.getState().setPrimaryColor(settings.primary_color) }
              if (settings.accent_color) { localStorage.setItem('nexus-accent-color', settings.accent_color); useAppStore.getState().setAccentColor(settings.accent_color) }
              if (settings.glb_url) localStorage.setItem('nexus-glb-url', settings.glb_url)
              if (settings.preset) localStorage.setItem('nexus-preset', settings.preset)
              if (settings.tts_voice) { localStorage.setItem('nexus-tts-voice', settings.tts_voice); useAppStore.getState().setTtsVoice(settings.tts_voice) }
              if (settings.character_id) localStorage.setItem('nexus-character', settings.character_id)
              localStorage.setItem('nexus-onboarded', 'true')
              useAppStore.setState({ isOnboarded: true })
            }
          } catch { /* 설정 복원 실패 시 localStorage 유지 */ }
          try {
            let row = await fetchSubscription(user.id)
            if (!row) {
              await createTrialSubscription(user.id)
              row = await fetchSubscription(user.id)
            }
            const status = resolveStatus(row)
            const expiry = row?.current_period_end ?? row?.trial_ends_at ?? ''
            useAppStore.getState().setLoggedIn(email, status, expiry, user.id)
          } catch { /* 구독 조회 실패해도 로그인은 유지 */ }
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

async function checkForUpdates() {
  try {
    const { check } = await import('@tauri-apps/plugin-updater')
    const update = await check()
    if (update?.available) {
      const yes = window.confirm(
        `새 버전 ${update.version}이 출시되었습니다.\n지금 업데이트하시겠습니까?`
      )
      if (yes) {
        await update.downloadAndInstall()
        const { relaunch } = await import('@tauri-apps/plugin-process')
        await relaunch()
      }
    }
  } catch { /* 개발 환경 / 오프라인 — 무시 */ }
}

/* 버티컬 테마 복원 — 앱 시작 시 저장된 테마 색상 즉시 적용 */
function applyStoredVertical() {
  const THEME_MAP: Record<string, string> = {
    legal:   '#7c3aed',
    medical: '#0891b2',
    finance: '#059669',
    content: '#dc2626',
    general: '#cba6f7',
  }
  const id = localStorage.getItem('nexus_vertical_id') ?? 'general'
  const color = THEME_MAP[id] ?? THEME_MAP.general
  document.documentElement.style.setProperty('--accent-primary', color)
  document.documentElement.style.setProperty('--accent-glow', color + '40')
}
applyStoredVertical()

setupTauriEvents()
bootstrap()
// 업데이트 체크는 앱 준비 후 백그라운드로 실행 (5초 딜레이)
setTimeout(checkForUpdates, 5000)

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <ErrorBoundary>
      <App />
    </ErrorBoundary>
  </React.StrictMode>
)
