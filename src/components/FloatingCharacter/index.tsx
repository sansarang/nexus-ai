import React, { useState, useRef, useEffect, useCallback } from 'react'
import { AnimatePresence, motion } from 'framer-motion'
import { useAppStore } from '../../stores/appStore'
import { saveUserSettings } from '../../lib/supabase'
import { DesktopAgent } from '../DesktopAgent'
import { WorkflowBuilder } from '../WorkflowBuilder'
import { EmailSetup } from '../EmailSetup'
import { ChatBubble } from './ChatBubble'
import type { ChatMessage, AttachedFile } from './ChatBubble'
import { SettingsModal } from './SettingsModal'
import type { InlineCardData } from './InlineCards'
import type { InlineCardData2 } from './InlineCards2'
import type { InlineCard3Data } from './InlineCards3'
import type { InlineCard4Data } from './InlineCards4'
import type { InlineCard5Data } from './InlineCards5'
// ResultDrawer / Avatar3D import 제거됨 (v2.6 Orb 레이아웃)
import { OnboardingFlow, LoginScreen } from './OnboardingFlow'
import type { AvatarConfig } from './OnboardingFlow'
import { PaywallModal } from '../PaywallModal'
import { appendHistory } from './ChatBubble'
import { callGemini, callOllama, fallbackResponse, trackUsage, getLastPreviewItems, clearLastPreviewItems, isFollowUpQuestion } from '../../lib/nexus/gemini_engine'
import { getDailyUsage, getMonthlyUsage } from '../../lib/nexus/usageTracker'
import { loadHistory, saveHistory, learnFromTurn, fromStoredTurns, toStoredTurns, buildMemoryContext } from '../../lib/nexus/memory'
import { getGreeting } from '../../lib/nexus/personality'
import { speak, stopSpeaking, isAudioPlaying } from '../../lib/nexus/tts'
import {
  detectIntent, extractFolderName,
  extractVolume, extractBrightness, extractWifiAction, extractPowerAction,
  extractAppName, extractNoteContent,
  extractTwoFilePaths, extractVisionQuestion, extractDeepSearchQuery,
} from '../../lib/nexus/intentDetector'
import type { Intent } from '../../lib/nexus/intentDetector'
import { routeWithLLM } from '../../lib/nexus/llmToolRouter'
import { backendAPI, mockStats, mockScan, mockDailyReport, sendCommand,
  calendarToday, calendarWeek, calendarAdd, calendarFindSlot, calendarSmartAdd,
  emailInbox, emailSend, emailSummarize, emailClassify, emailDraftReply,
  virusTotalCheck, historyStats, historyAnomalies,
  processKill, appPermissions, windowsUpdates, gpuStats,
  priceCompare, newsSearch, youtubeSearch, tiktokSearch, naverShoppingSearch, coupangSearch, videoDownload, videoQuickSearch,
  schedulerAdd, schedulerList, schedulerDelete,
  recallCapture, recallSearch,
  clipboardHistory, clipboardHistoryClear,
  meetingStart, meetingStop, meetingList, meetingTranscribe, meetingSummarize,
  dictationType, dictationPaste,
  weatherGet, travelTime,
  personaList, personaSet, personaCurrent,
  brainSearch, brainStats, brainRebuild,
  workflowRun, workflowPlan, workflowList, workflowFromText, workflowTemplates,
  captionStart, captionStop, captionLatest,
  briefingNow,
  taskList, taskCancel,
  multiAgentRun, multiAgentPlan,
  searchAndPDF,
  siteSearch,
  getAuthHeader,
} from '../../lib/nexus/backendAPI'
import type { PersonaDef } from '../../lib/nexus/backendAPI'
import {
  evaluateTriggers, getUptimeMs, getFocusModeEnd, setFocusModeEnd, clearFocusMode,
  STATS_POLL_MS, SECURITY_POLL_MS,
} from '../../lib/nexus/proactiveAI'
import type { ProactiveAlert } from '../../lib/nexus/proactiveAI'
import { safeCall, REQUIRE_REAL_BACKEND, checkBackendHealth } from '../../lib/nexus/environment'
import { nexusSSE } from '../../lib/nexus/sseClient'
import type { TaskUpdate } from '../../lib/nexus/sseClient'
import type { BackendStatus } from '../../lib/nexus/environment'
import type { NexusEmotion } from '../../types/nexus'
import { handleBackendIntentImpl } from './chatIntentImpl'
import { sendTextImpl } from './chatSenderImpl'
type CharacterEmotion = NexusEmotion

/* Web Speech API 타입 */
interface ConversationTurn {
  role: 'user' | 'model'
  parts: Array<{ text: string }>
}


/* 영어 쿼리 판별 (60% 이상 ASCII) */
function isEnglishQuery(q: string): boolean {
  if (!q) return false
  const chars = [...q]
  const ascii = chars.filter(c => c.charCodeAt(0) < 128).length
  return ascii / chars.length > 0.6
}


export function FloatingCharacter() {
  const {
    assistantName, userName, userLang,
    primaryColor: storePrimary, accentColor: storeAccent,
    isOnboarded, setOnboarded, setAssistantName, setUserName,
    setPrimaryColor, setAccentColor,
    micEnabled, ttsVoice, setTtsVoice,
    isLoggedIn, setLoggedIn, subscriptionStatus, userEmail,
    setUserLang,
    showWorkflowBuilder: storeShowWorkflowBuilder, workflowBuilderInitialName, setShowWorkflowBuilder: storeSetShowWorkflowBuilder,
    registerTriggerCommand,
  } = useAppStore()

  // 저장된 테마 색상이 있으면 우선 적용
  const savedThemeColor = localStorage.getItem('nexus-theme-color')
  const primaryColor = savedThemeColor || storePrimary || '#a78bfa'
  const accentColor  = storeAccent  || '#f9a8d4'

  // RPM GLB URL (있으면 RPM 아바타, 없으면 ProceduralHumanoid)
  const [glbUrl,  setGlbUrl]  = useState<string | null>(() =>
    isOnboarded ? localStorage.getItem('nexus-glb-url') : null
  )
  const [avatarPreset, setAvatarPreset] = useState<import('./Avatar3D').CharacterPreset>(
    () => (localStorage.getItem('nexus-preset') as import('./Avatar3D').CharacterPreset | null) ?? 'kpop_star'
  )

  const [chatOpen, setChatOpen]           = useState(false)
  const [messages, setMessages]           = useState<ChatMessage[]>([])
  const [typing, setTyping]               = useState(false)
  const [typingSteps, setTypingSteps]     = useState<string[]>([])
  const [emotion, setEmotion]             = useState<'neutral'|'happy'|'concerned'|'alert'|'humorous'>('neutral')
  const [speaking, setSpeaking]           = useState(false)
  const [input, setInput]                 = useState('')
  const [minimized, setMinimized]         = useState(false)
  const [settingsOpen, setSettingsOpen]     = useState(false)
  const [showDesktopAgent, setShowDesktopAgent] = useState(false)
  const showWorkflowBuilder = storeShowWorkflowBuilder
  const setShowWorkflowBuilder = (val: boolean) => storeSetShowWorkflowBuilder(val)
  const [showEmailSetup, setShowEmailSetup] = useState(false)
  const [toastAlerts, setToastAlerts]     = useState<Array<{id: string; title: string; message: string; level: string}>>([])
  const [soundEnabled, setSoundEnabled]   = useState(() => localStorage.getItem('nexus-sound') !== 'off')
  const [isActive, setIsActive]           = useState(true)
  const [beamEnabled, setBeamEnabled]     = useState(() => localStorage.getItem('nexus-beam') !== 'off')
  const [historyVersion, setHistoryVersion] = useState(0)
  const [backendStatus, setBackendStatus] = useState<BackendStatus>('checking')
  const [focusEndMs, setFocusEndMs]       = useState<number | undefined>(getFocusModeEnd())
  const [floatingPreview, setFloatingPreview] = useState<Array<{ title: string; url: string; isVideo?: boolean; isSocial?: boolean; isMap?: boolean; mapType?: string; service?: string; isImage?: boolean }> | null>(null)
  const [previewType, setPreviewType] = useState<string>('general')
  const [lastActionKey, setLastActionKey] = useState<string>('')
  const [lastResultPath, setLastResultPath] = useState<string>('')
  const [lastQuery, setLastQuery] = useState<string>('')

  interface DynamicResultAction { icon: string; label: string; onClick: () => void }
  interface DynamicResultState {
    icon: string
    title: string
    success: boolean
    stats?: Array<{ label: string; value: string }>
    items?: string[]
    links?: Array<{ title: string; url: string }>
    fileInfo?: { name: string; size?: string; path?: string; mimeType?: string }
    actions: DynamicResultAction[]
  }
  const [dynamicResult, setDynamicResult] = useState<DynamicResultState | null>(null)
  // resultDrawerOpen / favLinks / previewFilter 제거됨 — v2.6 인라인 결과창으로 대체
  // savedPreviews 제거됨 — floatingPreview 팝업과 함께 제거

  // ── Alt+S/V/C 글로벌 단축키 → sendText 콜백 등록 ─────────────
  useEffect(() => {
    registerTriggerCommand((text: string) => {
      // 창이 닫혀 있으면 열고 명령 전송
      sendText(text)
    })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // ── Clarify 멀티턴 상태 ──────────────────────────────────
  const [clarifyPendingIntent,   setClarifyPendingIntent]   = useState<string | null>(null)
  const [clarifyPendingParams,   setClarifyPendingParams]   = useState<Record<string, unknown> | null>(null)
  const [clarifyPendingQuestion, setClarifyPendingQuestion] = useState<string | null>(null)
  const [activePersona, setActivePersona] = useState<PersonaDef | null>(null)
  const [dailyUsedCount, setDailyUsedCount] = useState(() => getDailyUsage().count)
  const [showPersonaPopup, setShowPersonaPopup] = useState(false)
  const [personaListData, setPersonaListData] = useState<PersonaDef[]>([])
  // 영상 파일 첨부 시 의도 확인 팝업
  const [videoIntentPending, setVideoIntentPending] = useState<{ file: AttachedFile; extra: AttachedFile[] } | null>(null)
  const [captionRunning, setCaptionRunning] = useState(false)

  // ── Paywall Modal 상태 ────────────────────────────────────
  const [paywallFeature, setPaywallFeature] = useState<string | null>(null)
  const [paywallUsed,    setPaywallUsed]    = useState(0)
  const [paywallLimit,   setPaywallLimit]   = useState(0)

  // ── 미리보기 WebviewWindow 열기 ────────────────────────────
  const openPreview = useCallback(async (url: string, _title: string) => {
    try {
      // Tauri 환경: shell.open으로 시스템 기본 브라우저에서 열기
      const { open } = await import('@tauri-apps/plugin-shell')
      await open(url)
    } catch {
      // 브라우저 환경 폴백
      window.open(url, '_blank')
    }
  }, [])

  // 페르소나 칩 클릭 → 팝업 열기 + 목록 로드
  const handlePersonaChipClick = useCallback(async () => {
    try {
      const data = await personaList()
      setPersonaListData(data.personas)
    } catch { /* 목록 로드 실패 시 기존 빈 목록 */ }
    setShowPersonaPopup(true)
  }, [])

  // 페르소나 선택 → 전환
  const handlePersonaSelect = useCallback(async (id: string) => {
    setShowPersonaPopup(false)
    // 'general' = 기본 모드 → 백엔드 'nexus' ID로 매핑
    const backendId = id === 'general' ? 'nexus' : id
    try {
      const res = await personaSet(backendId)
      if (res.ok && res.persona) setActivePersona(res.persona)
    } catch { /* 전환 실패 무시 */ }
  }, [])

  // useCallback으로 안정적인 참조 유지 (sendText deps에 포함됨)
  const resetClarify = useCallback(() => {
    setClarifyPendingIntent(null)
    setClarifyPendingParams(null)
    setClarifyPendingQuestion(null)
  }, [])

  const historyRef         = useRef<ConversationTurn[]>(fromStoredTurns(loadHistory()) as ConversationTurn[])

  // 모델 응답을 히스토리에 추가하고 장기 메모리에 저장 + 프로필 학습
  const pushModelHistory = useCallback((userText: string, modelText: string) => {
    historyRef.current.push({ role: 'model', parts: [{ text: modelText }] })
    learnFromTurn(userText, modelText)
    saveHistory(toStoredTurns(historyRef.current as ConversationTurn[]))
  }, [])
  const typingRef          = useRef(false)
  const isMountedRef       = useRef(true)   // unmount 후 setState 방지
  const proactiveTimerRef  = useRef<ReturnType<typeof setInterval> | null>(null)
  const securityTimerRef   = useRef<ReturnType<typeof setInterval> | null>(null)
  const focusTimerRef      = useRef<ReturnType<typeof setInterval> | null>(null)
  const latestStatsRef     = useRef<import('../../lib/nexus/backendAPI').StatsData | null>(null)
  const latestSecRef       = useRef<{ found: boolean; score: number } | null>(null)

  useEffect(() => {
    isMountedRef.current = true
    return () => {
      isMountedRef.current = false
      stopSpeaking()
    }
  }, [])

  /* 오디오 재생 상태 실시간 동기화 — 중지 버튼이 항상 올바르게 표시되도록 */
  useEffect(() => {
    const id = setInterval(() => {
      const playing = isAudioPlaying()
      setSpeaking(prev => {
        if (prev !== playing) return playing
        return prev
      })
    }, 200)
    return () => clearInterval(id)
  }, [])

  /* nexus-groq-key → nexus-pplx-key 마이그레이션 */
  useEffect(() => {
    const old = localStorage.getItem('nexus-groq-key')
    if (old && !localStorage.getItem('nexus-pplx-key')) {
      localStorage.setItem('nexus-pplx-key', old)
    }
  }, [])

  /* 온보딩 완료 핸들러 */
  const handleOnboardingComplete = useCallback((config: AvatarConfig) => {
    setAssistantName(config.assistantName)
    setUserName(config.userName)
    setPrimaryColor(config.primaryColor)
    setAccentColor(config.accentColor)
    if (config.glbUrl) {
      setGlbUrl(config.glbUrl)
      localStorage.setItem('nexus-glb-url', config.glbUrl)
    }
    if (config.preset) {
      setAvatarPreset(config.preset)
      localStorage.setItem('nexus-preset', config.preset)
    }
    if (config.ttsVoice) setTtsVoice(config.ttsVoice)
    // 구글 로그인으로 저장된 계정 정보를 스토어에 반영
    const email = localStorage.getItem('nexus-user-email') ?? ''
    const status = (localStorage.getItem('nexus-sub-status') as 'active' | 'trial' | 'expired' | 'none') ?? 'trial'
    const expiry = localStorage.getItem('nexus-sub-expiry') ?? ''
    if (email) setLoggedIn(email, status, expiry)
    setOnboarded()
    // Supabase에 설정 저장
    const userId = localStorage.getItem('nexus-user-id')
    if (userId) {
      saveUserSettings(userId, {
        assistant_name: config.assistantName,
        user_name: config.userName,
        user_lang: localStorage.getItem('nexus-lang') ?? 'ko',
        primary_color: config.primaryColor,
        accent_color: config.accentColor,
        glb_url: config.glbUrl ?? '',
        preset: config.preset ?? '',
        tts_voice: config.ttsVoice ?? 'nova',
        character_id: localStorage.getItem('nexus-character') ?? 'sora',
        is_onboarded: true,
      }).catch(() => {})
    }
  }, [setAssistantName, setUserName, setPrimaryColor, setAccentColor, setOnboarded, setTtsVoice, setLoggedIn])

  /* 첫 인사 — 텍스트 + TTS (1회만 재생) */
  const hasGreetedRef = useRef(false)
  useEffect(() => {
    const greeting = getGreeting(assistantName, userName, userLang)
    setMessages([{ id: '0', role: 'nexus', text: greeting }])
    historyRef.current = []
    if (isOnboarded && !hasGreetedRef.current) {
      hasGreetedRef.current = true
      const tid = setTimeout(() => {
        const preview = greeting.replace(/\*\*/g, '').replace(/\n/g, ' ').slice(0, 60)
        setBubbleText(preview + (greeting.length > 60 ? '...' : ''))
        const isPro = subscriptionStatus === 'active' || subscriptionStatus === 'trial'
        speak(greeting, userLang, () => setSpeaking(true), () => { setSpeaking(false); setTimeout(() => setBubbleText(''), 1500) }, 'neutral', undefined, isPro)
      }, 800)
      return () => clearTimeout(tid)
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [assistantName, userName, userLang, isOnboarded])

  /* ── 백엔드 연결 상태 초기 체크 + 페르소나 로딩 + API 키 동기화 + 언어 동기화 ── */
  useEffect(() => {
    const connectAndSync = async () => {
      const status = await checkBackendHealth()
      setBackendStatus(status)
      if (status === 'connected') {
        try {
          const { syncAPIKeysToBackend } = await import('../../lib/nexus/gemini_engine')
          await syncAPIKeysToBackend()
        } catch { /* 무시 */ }

        // 백엔드에 현재 localStorage 언어 설정 동기화 (자동화 기능이 올바른 언어 사용하도록)
        try {
          const savedLang = localStorage.getItem('nexus-lang') ?? 'ko'
          await fetch('http://127.0.0.1:17891/api/settings/lang', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ lang: savedLang }),
          })
        } catch { /* 무시 */ }

        // ── Trial 만료 체크 ──────────────────────────────────────
        try {
          const subStatus = localStorage.getItem('nexus-sub-status')
          const subExpiry = localStorage.getItem('nexus-sub-expiry')
          if (subStatus === 'trial' && subExpiry) {
            const expired = new Date(subExpiry) < new Date()
            if (expired) {
              // Supabase에서 최신 구독 상태 확인
              const { supabase } = await import('../../lib/supabase')
              const { data: sessionData } = await supabase.auth.getSession()
              const jwt = sessionData.session?.access_token
              let isActiveSub = false
              if (jwt) {
                try {
                  const res = await fetch('https://dnlkhzoffyomqlqykmnc.supabase.co/rest/v1/subscriptions?select=status,current_period_end&order=created_at.desc&limit=1', {
                    headers: { 'Authorization': `Bearer ${jwt}`, 'apikey': (await import('../../config/services')).SUPABASE_ANON_KEY },
                  })
                  const rows = await res.json()
                  if (Array.isArray(rows) && rows[0]?.status === 'active') {
                    isActiveSub = true
                    localStorage.setItem('nexus-sub-status', 'active')
                    localStorage.setItem('nexus-sub-expiry', rows[0].current_period_end ?? '')
                    useAppStore.getState().setLoggedIn(
                      localStorage.getItem('nexus-user-email') ?? '',
                      'active',
                      rows[0].current_period_end ?? '',
                    )
                  }
                } catch { /* 무시 */ }
              }
              if (!isActiveSub) {
                // 진짜 만료 — expired로 전환
                localStorage.setItem('nexus-sub-status', 'expired')
                useAppStore.getState().setLoggedIn(
                  localStorage.getItem('nexus-user-email') ?? '',
                  'expired',
                  subExpiry,
                )
                const isEn = (localStorage.getItem('nexus-lang') ?? 'ko') === 'en'
                // 1.5초 후 결제 유도 메시지
                setTimeout(async () => {
                  setMessages(prev => [...prev, {
                    id: `trial-expired-${Date.now()}`,
                    role: 'nexus' as const,
                    text: isEn
                      ? `⏰ **Your 3-day free trial has ended.**\n\nTo keep using Nexus AI, please subscribe.\n👉 Click the banner below or go to **Settings → Subscription**.`
                      : `⏰ **3일 무료 체험이 종료되었습니다.**\n\nNexus AI를 계속 사용하려면 구독이 필요합니다.\n👉 아래 배너를 클릭하거나 **설정 → 구독**에서 결제해 주세요.`,
                  }])
                  // 3초 후 Paddle 결제창 자동 팝업
                  setTimeout(async () => {
                    try {
                      const email = localStorage.getItem('nexus-user-email') ?? ''
                      if (email) {
                        const { openCheckout } = await import('../../lib/paddle')
                        await openCheckout(email)
                      }
                    } catch { /* 무시 */ }
                  }, 3000)
                }, 1500)
              }
            }
          }
        } catch { /* 무시 */ }

        // API 키 상태 확인 — 로그인(JWT)된 사용자는 Supabase 프록시로 동작하므로 경고 불필요
        try {
          const { supabase } = await import('../../lib/supabase')
          const { data: sessionData } = await supabase.auth.getSession()
          const isLoggedIn = !!sessionData.session?.access_token

          if (!isLoggedIn) {
            // 미로그인: 직접 API 키 없으면 경고
            const cfg = await fetch('http://127.0.0.1:17891/api/llm/config').then(r => r.json())
            if (cfg && !cfg.ai_ready) {
              const isEn = (localStorage.getItem('nexus-lang') ?? 'ko') === 'en'
              setTimeout(() => {
                setMessages(prev => [...prev, {
                  id: `sys-apikey-${Date.now()}`,
                  role: 'nexus' as const,
                  text: isEn
                    ? `⚠️ **Not signed in & no API key set.**\nCore AI features won't work.\n\n👉 **Sign in** to use your subscription, or open **Settings (⚙️) → API Keys** to enter a Groq key (gsk_...)\n\nFree key: https://console.groq.com`
                    : `⚠️ **로그인되지 않았고 API 키도 없습니다.**\n핵심 AI 기능이 동작하지 않습니다.\n\n👉 **로그인**하면 구독 기능을 바로 사용할 수 있습니다.\n또는 **설정(⚙️) → API 키**에서 Groq 키(gsk_...) 직접 입력\n\n무료 키: https://console.groq.com`,
                }])
              }, 1500)
            }
          }
        } catch { /* 무시 */ }
      }
      return status
    }
    let active = true
    connectAndSync()
    personaCurrent().then((r) => { if (active) setActivePersona(r.persona) }).catch(() => {})

    // ⑩ 백엔드 자동 재연결: disconnected 상태면 30초마다 재시도
    let wasDisconnected = false
    let isReconnecting = false
    const reconnectId = setInterval(() => {
      if (isReconnecting) return
      setBackendStatus(prev => {
        if (prev === 'disconnected') {
          wasDisconnected = true
          isReconnecting = true
          connectAndSync().then(newStatus => {
            isReconnecting = false
            if (newStatus === 'connected' && wasDisconnected) {
              wasDisconnected = false
              setBubbleText(userLang === 'ko' ? '백엔드에 다시 연결됐습니다.' : 'Reconnected to backend.')
              setTimeout(() => setBubbleText(''), 3000)
            }
          }).catch(() => { isReconnecting = false })
        }
        return prev
      })
    }, 30000)
    return () => { active = false; clearInterval(reconnectId) }
  }, [])

  /* ── SSE 연결: Proactive 알림 + Task Queue 실시간 수신 ── */
  // nexusSSE 가 /api/alerts/stream 구독을 단독 관리 (onerror + 5초 재연결 내장).
  // 별도 EventSource를 만들지 않음 — 이전에는 중복 구독이라 에러 핸들러 누락 시 영구 silence.
  useEffect(() => {
    nexusSSE.connect()

    const unsubAlert = nexusSSE.onAlert((alert) => {
      // 토스트 알림 (단순 정보성)
      if (alert.title && !alert.action?.startsWith('approve:')) {
        const toast = { id: alert.id || String(Date.now()), title: alert.title, message: alert.message, level: alert.level || 'info' }
        setToastAlerts(prev => [...prev.slice(-4), toast])
        setTimeout(() => setToastAlerts(prev => prev.filter(t => t.id !== toast.id)), 7000)
      }

      // 승인 요청 알림 처리
      if (alert.action?.startsWith('approve:')) {
        const taskId = alert.action.replace('approve:', '')
        setMessages(prev => [...prev, {
          id: `approval-${taskId}`,
          role: 'nexus' as const,
          text: userLang === 'en'
            ? `⚠️ **Approval Required**\n${alert.message}\n\nType [Approve] or [Deny].`
            : `⚠️ **작업 승인 필요**\n${alert.message}\n\n[승인] 또는 [거부]를 입력해 주세요.`,
        }])
        setBubbleText(userLang === 'en' ? 'Action approval required ✋' : '작업 승인이 필요합니다 ✋')
        setChatOpen(true)
        return
      }
      // 일반 알림 → 말풍선 표시
      setBubbleText(alert.message.slice(0, 80))
      setMessages(prev => [...prev, {
        id: `sse-alert-${alert.id}`,
        role: 'nexus' as const,
        text: `${alert.title}: ${alert.message}`,
      }])
    })

    const unsubTask = nexusSSE.onTask((update: TaskUpdate) => {
      if (update.status === 'done') {
        const msg = `✅ 작업 완료: ${update.name}`
        setBubbleText(msg)
        setMessages(prev => [...prev, { id: `task-done-${update.id}`, role: 'nexus' as const, text: msg }])
        setTimeout(() => setBubbleText(''), 4000)
      } else if (update.status === 'failed') {
        const msg = `❌ 작업 실패: ${update.name} — ${update.error ?? ''}`
        setBubbleText(msg)
        setMessages(prev => [...prev, { id: `task-fail-${update.id}`, role: 'nexus' as const, text: msg }])
      }
    })

    return () => {
      unsubAlert()
      unsubTask()
      nexusSSE.disconnect()
    }
  }, [])

  /* ── Proactive AI 핵심 발화 함수 ── */
  const fireProactiveAlert = useCallback((alert: ProactiveAlert) => {
    if (typingRef.current) return
    setEmotion(alert.emotion)
    setMinimized(false)
    setChatOpen(true)

    // 액션 버튼이 있으면 메시지에 포함
    const actionHint = alert.actions.length > 0
      ? `\n\n${alert.actions.map(a => `→ "${a.autoText}"`).join('\n')}`
      : ''

    setMessages(prev => [...prev, {
      id:   `proactive-${alert.timestamp}`,
      role: 'nexus',
      text: alert.message + actionHint,
      inlineCard: undefined,
    }])

    const preview = alert.message.replace(/\*\*/g, '').replace(/\n/g, ' ').slice(0, 60)
    setBubbleText(preview + (alert.message.length > 60 ? '...' : ''))
    const isPro = subscriptionStatus === 'active' || subscriptionStatus === 'trial'
    speak(alert.message, userLang, () => setSpeaking(true), () => { setSpeaking(false); setTimeout(() => setBubbleText(''), 1500) }, 'neutral', undefined, isPro)
  }, [userLang, subscriptionStatus])

  /* ── Proactive AI — stats 폴링 (30초) ── */
  useEffect(() => {
    const pollStats = async () => {
      // 프로덕션: 실제 백엔드만 사용 / 개발: mock fallback 허용
      const stats = await safeCall(
        () => backendAPI.stats(),
        () => mockStats(),
      )
      if (!stats) return

      latestStatsRef.current = stats
      if (stats.cpu_temp > 75 || stats.cpu > 75 || stats.mem > 80 || stats.disk > 90) {
        setBackendStatus('connected')
      }

      const snapshot = {
        stats,
        security:     latestSecRef.current ?? undefined,
        uptimeMs:     getUptimeMs(),
        focusModeEndMs: focusEndMs,
      }
      const alert = evaluateTriggers(snapshot, userLang, assistantName)
      if (alert) fireProactiveAlert(alert)
    }

    pollStats() // 즉시 1회 실행
    proactiveTimerRef.current = setInterval(pollStats, STATS_POLL_MS)
    return () => { if (proactiveTimerRef.current) clearInterval(proactiveTimerRef.current) }
  }, [userLang, assistantName, focusEndMs, fireProactiveAlert])

  /* ── Proactive AI — 보안 폴링 (5분) ── */
  useEffect(() => {
    const pollSecurity = async () => {
      const data = await safeCall(
        () => backendAPI.securityRemote(),
        () => ({ found: false, tools: [], rdp_open: false, score: 100 }),
      )
      if (!data) return
      latestSecRef.current = { found: data.found, score: data.score }

      if (data.found && data.score < 70 && latestStatsRef.current) {
        const snapshot = {
          stats:        latestStatsRef.current,
          security:     { found: data.found, score: data.score },
          uptimeMs:     getUptimeMs(),
          focusModeEndMs: focusEndMs,
        }
        const alert = evaluateTriggers(snapshot, userLang, assistantName)
        if (alert) fireProactiveAlert(alert)
      }
    }

    securityTimerRef.current = setInterval(pollSecurity, SECURITY_POLL_MS)
    return () => { if (securityTimerRef.current) clearInterval(securityTimerRef.current) }
  }, [userLang, assistantName, focusEndMs, fireProactiveAlert])

  /* ── 집중 모드 타이머 감시 (1초) ── */
  useEffect(() => {
    if (!focusEndMs) return
    focusTimerRef.current = setInterval(() => {
      if (Date.now() >= focusEndMs) {
        clearFocusMode()
        setFocusEndMs(undefined)
        const msg = userLang === 'ko'
          ? `${assistantName}, 25분 집중 끝났어요! ☕ 잠깐 쉬고 오세요.`
          : `${assistantName}, focus session done! Take a break. ☕`
        setEmotion('happy')
        setMinimized(false)
        setChatOpen(true)
        setMessages(prev => [...prev, { id: `focus-done-${Date.now()}`, role: 'nexus', text: msg }])
        const isPro = subscriptionStatus === 'active' || subscriptionStatus === 'trial'
        speak(msg, userLang, () => setSpeaking(true), () => setSpeaking(false), 'neutral', undefined, isPro)
        clearInterval(focusTimerRef.current!)
      }
    }, 1_000)
    return () => { if (focusTimerRef.current) clearInterval(focusTimerRef.current) }
  }, [focusEndMs, userLang, assistantName])

  /* 사용량 배지 — 메시지 변경 시 갱신 */
  useEffect(() => {
    setDailyUsedCount(getDailyUsage().count)
  }, [messages.length])

  /* 말풍선에 표시할 최근 AI 발화 */
  const [bubbleText, setBubbleText] = useState('')
  const [bubbleExpanded, setBubbleExpanded] = useState(false)

  /* TTS — 감정 기반 톤 자동 조정 (Pro만 OpenAI TTS, 무료는 Web Speech) */
  const speakText = useCallback((text: string, em?: CharacterEmotion) => {
    const clean = text.replace(/\*\*/g, '').replace(/\n+/g, ' ').trim()
    setBubbleText(clean)
    if (!soundEnabled) return
    const ttsEmotion = em ?? emotion
    const isPro = subscriptionStatus === 'active' || subscriptionStatus === 'trial'
    speak(
      text, userLang,
      () => setSpeaking(true),
      () => setSpeaking(false),
      ttsEmotion as import('../../lib/nexus/tts').SpeakEmotion,
      ttsVoice,
      isPro,
    )
  }, [userLang, emotion, ttsVoice, soundEnabled, subscriptionStatus])

  /* STT */


  /* ── 백엔드 인텐트 처리 — LLM 없이 즉시 ── */
  const handleBackendIntent = useCallback(async (
    intent: Intent,
    msgId: string,
    originalText = '',
  ): Promise<{ text: string; card?: InlineCardData; card2?: InlineCardData2; card3?: InlineCard3Data; card4?: InlineCard4Data; emotion: CharacterEmotion }> => {
    return handleBackendIntentImpl(intent, msgId, originalText, {
      userLang, assistantName, emotion, isActive, soundEnabled,
      clarifyPendingIntent, clarifyPendingParams, clarifyPendingQuestion,
      historyRef, typingRef, isMountedRef,
      setMessages, setEmotion, setChatOpen, setMinimized, setSpeaking,
      setBubbleText, setActivePersona, setCaptionRunning, setFloatingPreview,
      setFocusEndMs, setPreviewType, setToastAlerts, setSoundEnabled,
      setIsActive, setHistoryVersion, speakText, resetClarify, openPreview,
      pushModelHistory, setUserLang,
      openEmailSetup: () => setShowEmailSetup(true),
    })
  }, [userLang, assistantName, emotion, isActive, soundEnabled,
    clarifyPendingIntent, clarifyPendingParams, clarifyPendingQuestion,
    historyRef, typingRef, isMountedRef,
    setMessages, setEmotion, setChatOpen, setMinimized, setSpeaking,
    setBubbleText, setActivePersona, setCaptionRunning, setFloatingPreview,
    setFocusEndMs, setPreviewType, setToastAlerts, setSoundEnabled,
    setIsActive, setHistoryVersion, speakText, resetClarify, openPreview,
    pushModelHistory, setUserLang])


  /* ── 수리 버튼 콜백 ── */
  const handleRepair = useCallback(async (ids: string[]) => {
    if (typingRef.current) return
    typingRef.current = true
    setTyping(true)
    const result = await backendAPI.repair(ids).catch(() => ({
      success: true,
      message: '수리가 완료되었어요!',
      freed: 0,
    }))
    setTyping(false)
    typingRef.current = false
    const text = result.success
      ? (userLang === 'ko' ? `✅ ${result.message}` : `✅ ${result.message}`)
      : (userLang === 'ko' ? '수리 중 오류가 발생했어요.' : 'Repair failed.')
    setMessages(prev => [...prev, {
      id: `repair-${Date.now()}`,
      role: 'nexus',
      text,
      inlineCard: { type: 'repair_result', data: result },
    }])
    setEmotion(result.success ? 'happy' : 'concerned')
    speakText(text)
  }, [userLang, speakText])

  /* ── action → 리치 카드 렌더링 ── */
  const renderCommandResult = useCallback(async (
    action: string, result: unknown, trimmed: string,
  ): Promise<{ card?: InlineCardData; card2?: InlineCardData2; card3?: InlineCard3Data; card4?: InlineCard4Data; card5?: InlineCard5Data; emotion: CharacterEmotion }> => {
    try {
      switch (action) {
        case 'web_search': {
          const r = result as { query?: string; summary?: string; items?: Array<{title?: string; url?: string; snippet?: string; source?: string; published?: string}> } | undefined
          const items = (r?.items ?? []).map(it => ({ title: it.title ?? it.url ?? '', url: it.url ?? '', snippet: it.snippet, source: it.source, published: it.published }))
          return {
            card5: { type: 'web_search', query: r?.query ?? trimmed, summary: r?.summary ?? '', items },
            emotion: 'happy',
          }
        }
        case 'news_search': {
          const r = result as { query?: string; summary?: string; items?: Array<{title?: string; url?: string; snippet?: string; source?: string; published?: string}> } | undefined
          const items = (r?.items ?? []).map(it => ({ title: it.title ?? it.url ?? '', url: it.url ?? '', snippet: it.snippet, source: it.source, published: it.published }))
          return {
            card5: { type: 'news_search', query: r?.query ?? trimmed, summary: r?.summary ?? '', items },
            emotion: 'happy',
          }
        }
        case 'youtube_search': {
          const r = result as { query?: string; items?: Array<{title?: string; url?: string; source?: string}> } | undefined
          const items = (r?.items ?? []).map(it => ({ title: it.title ?? '', url: it.url ?? '', source: it.source }))
          return {
            card5: { type: 'youtube', query: r?.query ?? trimmed, items },
            emotion: 'happy',
          }
        }
        case 'price_compare': {
          const r = result as { results?: {site:string;name:string;price:string;link:string}[]; query?: string; site?: string; summary?: string } | undefined
          const items = r?.results ?? []
          const query = r?.query || trimmed
          const siteName = (r?.site ?? '').replace('coupang.com','쿠팡').replace('shopping.naver.com','네이버쇼핑').replace('temu.com','태무').replace('11st.co.kr','11번가').replace('gmarket.co.kr','G마켓').replace('aliexpress.com','알리').replace('amazon.com','아마존') || '쇼핑몰'
          if (items.length > 0) setFloatingPreview(items.slice(0, 8).map(i => ({ title: i.name, url: i.link })))
          return {
            card2: { type: 'system_action', icon: '🛒', title: `${siteName}: ${query}`, detail: items.slice(0,5).map(i => `• ${i.name}${i.price ? ' — '+i.price : ''}`).join('\n') || '결과 없음', success: items.length > 0 },
            emotion: 'happy',
          }
        }
        case 'video_search': {
          const items = (result as { items?: { title: string; url: string }[] })?.items ?? []
          const platform = /tiktok|틱톡/i.test(trimmed) ? '틱톡' : '유튜브'
          const icon = platform === '틱톡' ? '🎵' : '🎬'
          const query = (result as { query?: string })?.query || trimmed
          // 플로팅 미리보기 패널에 영상 목록 표시 (재생/다운로드 버튼용)
          if (items.length > 0) {
            setFloatingPreview(items.slice(0, 8).map(a => ({ title: a.title, url: a.url, isVideo: true })))
          }
          return {
            card2: { type: 'system_action', icon, title: `${platform}: ${query}`, detail: items.length > 0 ? `${items.length}개 영상을 찾았어요. 오른쪽 패널에서 재생하세요!` : '검색 결과가 없어요.', success: items.length > 0 },
            emotion: items.length > 0 ? 'happy' : 'concerned',
          }
        }
        case 'multi_action': {
          const r = result as { results?: {site:string;name:string;price:string;link:string}[]; query?: string; summary?: string; file_path?: string; file_msg?: string; format?: string; sub_action?: string } | undefined
          const items = r?.results ?? []
          const query = r?.query || trimmed
          const fileMsg = r?.file_msg ?? ''
          const filePath = r?.file_path ?? ''
          const fmt = (r?.format ?? '').toUpperCase() || 'FILE'
          const isVideo = r?.sub_action === 'video_search'
          if (items.length > 0) {
            if (isVideo) {
              setFloatingPreview(items.slice(0, 8).map(i => ({ title: i.name, url: i.link, isVideo: true })))
            } else {
              setFloatingPreview(items.slice(0, 8).map(i => ({ title: i.name, url: i.link })))
            }
          }
          const itemLines = items.slice(0,5).map(i => `• ${i.name}${i.price ? ' — '+i.price : ''}`).join('\n') || '결과 없음'
          const detail = filePath ? itemLines + `\n\n📄 ${fmt} 파일 저장됨` : itemLines
          return {
            card2: { type: 'system_action', icon: '📋', title: `멀티액션: ${query}`, detail, success: items.length > 0 },
            emotion: items.length > 0 ? 'happy' : 'concerned',
          }
        }
        case 'scan':
        case 'security_scan': {
          const data = (result && typeof result === 'object' && 'score' in result)
            ? result as import('../../lib/nexus/backendAPI').ScanResult
            : await backendAPI.scan().catch(() => mockScan())
          return { card: { type: 'scan_result', data }, emotion: data.score < 70 ? 'alert' : data.score < 85 ? 'concerned' : 'happy' }
        }
        case 'stats': {
          const data = (result && typeof result === 'object' && 'cpu' in result)
            ? result as import('../../lib/nexus/backendAPI').StatsData
            : await backendAPI.stats().catch(() => mockStats())
          return { card: { type: 'pc_status', data }, emotion: 'happy' }
        }
        case 'clean': {
          const r = await backendAPI.autoClean(['temp', 'browser']).catch(async () => {
            const r2 = await backendAPI.clean(['temp']).catch(() => ({ freed: 0, message: '정리 완료' }))
            return r2 as { freed: number; message: string }
          })
          return { card: { type: 'clean_result', results: r }, emotion: 'happy' }
        }
        case 'journal': {
          const data = await backendAPI.journalToday().catch(() => ({} as Record<string, unknown>))
          return {
            card4: { type: 'journal_today', data: data as unknown as Parameters<typeof import('./InlineCards4').JournalTodayCard>[0]['data'] },
            emotion: 'happy',
          }
        }
        case 'health_report': {
          const data = await backendAPI.reportGenerate().catch(() => ({ score: 0 }))
          return {
            card4: { type: 'pc_report', data: data as unknown as Parameters<typeof import('./InlineCards4').PCReportCard>[0]['data'] },
            emotion: 'happy',
          }
        }
        case 'vision': {
          const ss = await backendAPI.screenshot(true).catch(() => ({ success: false, base64: '', width: 0, height: 0, mime: 'image/png', captured: '', ocr_text: '' }))
          if (!ss.success) return { emotion: 'concerned' }
          const { callGeminiWithImage } = await import('../../lib/nexus/gemini_engine')
          const answer = (await callGeminiWithImage(ss.base64, trimmed).catch(() => null)) ?? (ss as { ocr_text?: string }).ocr_text ?? '분석 불가'
          return { card3: { type: 'vision_result', data: { question: trimmed, answer: answer || '분석 불가', screenshot_b64: ss.base64 } }, emotion: 'happy' }
        }
        /* ── 이메일 카드 ── */
        case 'email_inbox':
        case 'imap_inbox':
        case 'email_classify': {
          const r = result as { emails?: unknown[]; count?: number; unread?: number; summary?: string } | undefined
          return {
            card2: { type: 'email_list', data: { emails: (r?.emails ?? []) as Parameters<typeof import('./InlineCards2').EmailListCard>[0]['data']['emails'], count: r?.count, unread: r?.unread, summary: r?.summary } },
            emotion: 'happy',
          }
        }

        /* ── 캘린더 카드 ── */
        case 'calendar_today':
        case 'calendar_week':
        case 'calendar_find_slot': {
          const r = result as { events?: unknown[]; slots?: unknown[]; count?: number; title?: string } | undefined
          return {
            card2: { type: 'timeline', data: { events: (r?.events ?? []) as Parameters<typeof import('./InlineCards2').TimelineCard>[0]['data']['events'], slots: (r?.slots ?? []) as Parameters<typeof import('./InlineCards2').TimelineCard>[0]['data']['slots'], count: r?.count, title: r?.title } },
            emotion: 'happy',
          }
        }

        /* ── 게이지 바 카드 ── */
        case 'perf_history':
        case 'perf_anomaly':
        case 'gpu_stats': {
          const r = result as Parameters<typeof import('./InlineCards2').GaugeBarCard>[0]['data'] | undefined
          return {
            card2: { type: 'gauge_bar', data: r ?? {} },
            emotion: 'neutral',
          }
        }

        /* ── 텍스트 블록 카드 ── */
        case 'email_draft':
        case 'email_summarize':
        case 'translate':
        case 'clipboard_ai':
        case 'meeting_summary':
        case 'recall_capture':
        case 'search_pdf':
        case 'dictation_start':
        case 'voice_todo': {
          const iconMap: Record<string, string> = {
            email_draft: '✉️', email_summarize: '📧', translate: '🌐',
            clipboard_ai: '📋', meeting_summary: '🎙️', recall_capture: '📸',
            search_pdf: '📄', dictation_start: '🎤', voice_todo: '📝',
          }
          const titleMap: Record<string, string> = {
            email_draft: '이메일 초안', email_summarize: '이메일 요약', translate: '번역 결과',
            clipboard_ai: 'AI 처리 결과', meeting_summary: '회의 요약', recall_capture: '화면 캡처 메모',
            search_pdf: 'PDF 보고서', dictation_start: '받아쓰기 결과', voice_todo: '음성 메모',
          }
          const r = result as Parameters<typeof import('./InlineCards2').TextBlockCard>[0]['data'] | undefined
          return {
            card2: { type: 'text_block', data: { ...(r ?? {}), icon: iconMap[action], title: titleMap[action] } },
            emotion: 'happy',
          }
        }

        /* ── 단계 목록 카드 ── */
        case 'workflow_plan':
        case 'workflow_list':
        case 'workflow_templates':
        case 'schedule_list': {
          const titleMap: Record<string, string> = {
            workflow_plan: '워크플로 계획', workflow_list: '워크플로 목록',
            workflow_templates: '워크플로 템플릿', schedule_list: '예약 스케줄',
          }
          const r = result as Parameters<typeof import('./InlineCards2').StepListCard>[0]['data'] | undefined
          return {
            card2: { type: 'step_list', data: { ...(r ?? {}), title: r?.title ?? titleMap[action] } },
            emotion: 'happy',
          }
        }

        /* ── 항목 목록 카드 ── */
        case 'brain_search':
        case 'brain_stats':
        case 'clipboard_history':
        case 'recall_search':
        case 'meeting_list':
        case 'windows_updates':
        case 'app_permissions':
        case 'virus_check': {
          const iconMap: Record<string, string> = {
            brain_search: '🧠', brain_stats: '📊', clipboard_history: '📋',
            recall_search: '🔍', meeting_list: '🎙️', windows_updates: '🪟',
            app_permissions: '🔐', virus_check: '🛡️',
          }
          const titleMap: Record<string, string> = {
            brain_search: 'Second Brain 검색', brain_stats: 'Brain 통계',
            clipboard_history: '클립보드 기록', recall_search: 'Recall 검색',
            meeting_list: '회의 목록', windows_updates: 'Windows 업데이트',
            app_permissions: '앱 권한', virus_check: '바이러스 스캔',
          }
          const r = result as Parameters<typeof import('./InlineCards2').ItemListCard>[0]['data'] | undefined
          return {
            card2: { type: 'item_list', data: { ...(r ?? {}), icon: iconMap[action], title: r?.title ?? titleMap[action] } },
            emotion: action === 'virus_check' && (r as { detections?: number })?.detections ? 'alert' : 'happy',
          }
        }

        /* ── 페르소나 그리드 ── */
        case 'persona_list': {
          const r = result as { personas?: unknown[]; title?: string } | undefined
          return {
            card2: { type: 'grid_select', data: { personas: (r?.personas ?? []) as Parameters<typeof import('./InlineCards2').GridSelectCard>[0]['data']['personas'], title: r?.title } },
            emotion: 'happy',
          }
        }

        /* ── 날씨 카드 ── */
        case 'weather': {
          const r = result as Parameters<typeof import('./InlineCards2').WeatherCard>[0]['data'] | undefined
          return {
            card2: { type: 'weather_card', data: r ?? {} },
            emotion: 'happy',
          }
        }

        /* ── 보안 상세 카드 ── */
        case 'remote_access': {
          const r = result as { found?: boolean; tools?: unknown[]; rdp_open?: boolean; score?: number } | undefined
          return {
            card2: { type: 'remote_access', data: { found: r?.found ?? false, tools: (r?.tools ?? []) as import('../../lib/nexus/backendAPI').RemoteAccessResult['tools'], rdp_open: r?.rdp_open ?? false, score: r?.score ?? 100 } },
            emotion: r?.found ? 'alert' : 'happy',
          }
        }
        case 'process_security': {
          const r = result as { suspicious_processes?: unknown[]; open_ports?: unknown[]; score?: number } | undefined
          return {
            card2: { type: 'process_security', data: { suspicious_processes: (r?.suspicious_processes ?? []) as import('../../lib/nexus/backendAPI').ProcessSecurityResult['suspicious_processes'], open_ports: (r?.open_ports ?? []) as import('../../lib/nexus/backendAPI').ProcessSecurityResult['open_ports'], score: r?.score ?? 100 } },
            emotion: (r?.score ?? 100) < 80 ? 'alert' : 'happy',
          }
        }
        case 'defender_status': {
          const r = result as import('../../lib/nexus/backendAPI').DefenderStatus | undefined
          return {
            card2: { type: 'defender', data: r ?? { antivirus_enabled: true, realtime_protection: true, quick_scan_age: 0, full_scan_age: 0, score: 100, issues: [] } },
            emotion: (r?.score ?? 100) >= 80 ? 'happy' : 'alert',
          }
        }
        case 'startup_items': {
          const r = result as { items?: unknown[]; total?: number; suspicious_count?: number } | undefined
          return {
            card2: { type: 'startup_items', data: { items: (r?.items ?? []) as import('../../lib/nexus/backendAPI').StartupItem[], total: r?.total ?? 0, suspicious_count: r?.suspicious_count ?? 0 } },
            emotion: (r?.suspicious_count ?? 0) > 0 ? 'concerned' : 'happy',
          }
        }
        case 'process_top': {
          const r = result as { by_cpu?: unknown[]; by_mem?: unknown[] } | undefined
          return {
            card2: { type: 'process_top', data: { by_cpu: (r?.by_cpu ?? []) as import('../../lib/nexus/backendAPI').ProcItem[], by_mem: (r?.by_mem ?? []) as import('../../lib/nexus/backendAPI').ProcItem[] } },
            emotion: 'neutral',
          }
        }
        case 'network_analysis': {
          const r = result as { adapters?: unknown[]; dns_servers?: string; public_ip?: string; ping_ms?: string; connected?: boolean } | undefined
          return {
            card2: { type: 'network', data: { adapters: (r?.adapters ?? []) as import('../../lib/nexus/backendAPI').NetworkAdapter[], dns_servers: r?.dns_servers ?? '', public_ip: r?.public_ip ?? '', ping_ms: r?.ping_ms ?? '', connected: r?.connected ?? false } },
            emotion: r?.connected ? 'happy' : 'concerned',
          }
        }
        case 'driver_check': {
          const r = result as { total?: number; problematic?: unknown[]; problem_count?: number; score?: number; message?: string } | undefined
          return {
            card2: { type: 'drivers', data: { total: r?.total ?? 0, problematic: (r?.problematic ?? []) as import('../../lib/nexus/backendAPI').DriverItem[], problem_count: r?.problem_count ?? 0, score: r?.score ?? 100, message: r?.message ?? '' } },
            emotion: (r?.problem_count ?? 0) > 0 ? 'concerned' : 'happy',
          }
        }
        case 'programs_list': {
          const r = result as { programs?: unknown[]; total?: number } | undefined
          return {
            card2: { type: 'programs_list', data: { programs: (r?.programs ?? []) as import('../../lib/nexus/backendAPI').ProgramItem[], total: r?.total ?? 0 } },
            emotion: 'neutral',
          }
        }
        case 'file_search': {
          const r = result as { results?: unknown[]; total?: number; message?: string } | undefined
          return {
            card2: { type: 'file_search', data: { results: (r?.results ?? []) as import('../../lib/nexus/backendAPI').FileResult[], total: r?.total ?? 0, message: r?.message ?? '' } },
            emotion: 'neutral',
          }
        }
        case 'file_duplicates': {
          const r = result as { groups?: unknown[]; total_groups?: number; waste_mb?: number; waste?: string; message?: string } | undefined
          return {
            card2: { type: 'duplicates', data: { groups: (r?.groups ?? []) as import('../../lib/nexus/backendAPI').DupGroup[], total_groups: r?.total_groups ?? 0, waste_mb: r?.waste_mb ?? 0, waste: r?.waste ?? '0B', message: r?.message ?? '' } },
            emotion: (r?.total_groups ?? 0) > 0 ? 'concerned' : 'happy',
          }
        }
        case 'notes': {
          const r = result as { notes?: unknown[]; total?: number } | undefined
          return {
            card2: { type: 'notes', data: { notes: (r?.notes ?? []) as import('../../lib/nexus/backendAPI').NoteItem[], total: r?.total ?? 0 } },
            emotion: 'happy',
          }
        }
        case 'boot_analysis': {
          const r = result as { uptime_minutes?: string; startup_count?: string; message?: string } | undefined
          return {
            card2: { type: 'boot_analysis', data: { uptime_minutes: r?.uptime_minutes ?? '0', startup_count: r?.startup_count ?? '?', message: r?.message ?? '' } },
            emotion: 'neutral',
          }
        }
        case 'focus_mode': {
          const r = result as { active?: boolean; duration?: number } | undefined
          return {
            card2: { type: 'focus_mode', active: r?.active ?? true, duration: r?.duration },
            emotion: 'happy',
          }
        }
        default:
          return { emotion: 'neutral' }
      }
    } catch {
      return { emotion: 'neutral' }
    }
  }, [])

  /* ── 이력 저장 전 raw 텍스트 정제 (tavily/URL/메타 제거) ── */
  const cleanForHistory = (text: string): string => {
    return text
      .replace(/\[tavily\]/gi, '')
      .replace(/https?:\/\/\S+/g, '')
      .replace(/•\s*/g, '')
      .replace(/\n{3,}/g, '\n\n')
      .trim()
  }

  /* ── 메시지 전송 ── */
  const sendText = useCallback(async (text: string) => {
    return sendTextImpl(text, {
      userLang, assistantName, isActive, backendStatus, subscriptionStatus,
      clarifyPendingIntent, clarifyPendingParams, clarifyPendingQuestion,
      floatingPreview, ttsVoice,
      typingRef, historyRef, isMountedRef,
      setMessages, setInput, setTyping, setTypingSteps, setEmotion, setSpeaking,
      setUserLang, setHistoryVersion, setToastAlerts, setIsActive, setFloatingPreview,
      setPreviewType, setClarifyPendingIntent, setClarifyPendingParams, setClarifyPendingQuestion,
      speakText, resetClarify, pushModelHistory,
      handleBackendIntent, renderCommandResult,
      setDynamicResult,
      showPaywall: (feature, used, limit) => {
        setPaywallFeature(feature)
        setPaywallUsed(used)
        setPaywallLimit(limit)
      },
    })
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [assistantName, backendStatus, clarifyPendingIntent, clarifyPendingParams,
    clarifyPendingQuestion, handleBackendIntent, renderCommandResult, resetClarify, speakText,
    userLang, isActive, floatingPreview, ttsVoice, setDynamicResult])

  const handleSend = useCallback((text: string) => void sendText(text), [sendText])

  /** ErrorCard "재시도" 버튼 — 가장 최근 사용자 입력을 다시 전송 (인텐트 재탐지 + 재실행) */
  const handleRetry = useCallback((intent: string) => {
    const lastUserMsg = [...messages].reverse().find(m => m.role === 'user')
    if (lastUserMsg && lastUserMsg.text) {
      sendText(lastUserMsg.text)
    } else {
      // 폴백: 메시지가 없으면 인텐트 키만 보냄 (그대로 다시 분류)
      sendText(intent)
    }
  }, [messages, sendText])

  /** ErrorCard "API 키 설정" 버튼 — 설정 모달 열기 */
  const handleOpenSettings = useCallback(() => {
    setSettingsOpen(true)
  }, [])

  const handleSendWithFileImpl = useCallback(async (text: string, file: { name: string; mimeType: string; dataUrl: string; text?: string; size: number; fileType: 'image' | 'document' | 'spreadsheet' | 'video' | 'other' }, extraFiles?: Array<{ name: string; mimeType: string; dataUrl: string; text?: string; size: number; fileType: string }>) => {
    setTyping(true)
    const personaId = localStorage.getItem('nexus-persona-id') ?? 'nexus'
    const personaLabel: Record<string, string> = {
      nexus: '기본', expert: '전문가', research: '리서치', creative: '크리에이티브', finance: '재무',
    }
    const mode = personaLabel[personaId] ?? '기본'

    // 파일 처리 의도 감지
    const wantGIF = /gif|움직이는|애니메이션|움짤/i.test(text)
    const wantResize = /리사이즈|사이즈|크기|resize|인스타|트위터|유튜브|틱톡|썸네일|맞춰|플랫폼|변경.*크기|크기.*변경/i.test(text)
    const wantConvert = /변환|convert|jpg로|png로|webp로|jpeg로/i.test(text)
    const wantCompare = /비교|compare|차이|다른점|같은점/i.test(text)
    // 영상 편집 의도
    const wantTrim = /잘라|자르기|trim|구간|초부터|분부터|까지|처음.*분|처음.*초/i.test(text)
    const wantCompress = /압축|용량.*줄|줄여|compress|가볍게|작게/i.test(text)
    const wantSpeed = /배속|빠르게|느리게|speed|빨리|천천히/i.test(text)
    const wantSubtitle = /자막.*합쳐|합쳐.*자막|자막.*삽입|삽입.*자막|자막.*넣어|자막.*태워|srt.*합쳐|합쳐.*srt|subtitle.*merge|merge.*subtitle|burn.*subtitle|subtitle.*burn/i.test(text)
    const allFiles = [file, ...(extraFiles ?? [])]
    const isImageFile = (f: { mimeType: string }) => f.mimeType.startsWith('image/')
    const isVideoFile = (f: { mimeType: string }) => f.mimeType.startsWith('video/')

    // 편집 의도 감지 (분석 vs 수정)
    const wantEdit = /수정|편집|바꿔|변경|삭제|추가|정렬|필터|합계|계산|저장|만들어|작성|고쳐|업데이트|넣어|지워|빼|이름변경|이름 변경|rename|sort|edit|modify|delete|add|insert|update/i.test(text)
    const wantSearch = /검색|웹|최신|찾아|서치|search/i.test(text)

    let analysisResult = ''

    // ── 파일 처리 인텐트 (이미지/문서 조작) ──────────────────────
    const needsFileProcess = wantGIF || wantResize || wantConvert || (wantCompare && allFiles.length >= 2)
    const needsVideoEdit = isVideoFile(file) && (wantTrim || wantCompress || wantSpeed || wantSubtitle)

    try {
      if (needsFileProcess) {
        // 진행 중 메시지
        const fileNames = allFiles.map(f => f.name).join(', ')
        const opLabel = wantGIF ? 'GIF 변환' : wantResize ? '리사이즈' : wantConvert ? '포맷 변환' : '비교 분석'
        setMessages(prev => [
          ...prev,
          { id: `u-${Date.now()}`, role: 'user', text: `📎 ${fileNames}\n${text}` },
          { id: `n-${Date.now()}-progress`, role: 'nexus', text: `⏳ ${opLabel} 진행 중...` },
        ])

        const payload = {
          files: allFiles.map(f => ({ name: f.name, mime_type: f.mimeType, data: f.dataUrl })),
          operation: 'auto',
          query: text,
          params: {},
        }
        const res = await fetch('http://127.0.0.1:17891/api/file/process', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json', ...await getAuthHeader() },
          body: JSON.stringify(payload),
        }).then(r => r.json()).catch(() => ({ success: false, message: '파일 처리 실패' }))

        if (res.success) {
          const msg = res.message ?? `${opLabel} 완료`
          // 다운로드 가능한 파일이 있으면 링크 제공
          if (res.data && res.file_name) {
            const mimeType = res.mime_type ?? 'application/octet-stream'
            const byteString = atob(res.data)
            const ab = new ArrayBuffer(byteString.length)
            const ia = new Uint8Array(ab)
            for (let i = 0; i < byteString.length; i++) ia[i] = byteString.charCodeAt(i)
            const blob = new Blob([ab], { type: mimeType })
            const url = URL.createObjectURL(blob)

            setMessages(prev => prev.filter(m => !m.id.includes('-progress')))
            setMessages(prev => [...prev, {
              id: `n-${Date.now()}-result`,
              role: 'nexus',
              text: msg,
              inlineCard2: {
                type: 'file_result',
                data: { fileName: res.file_name, url, mimeType, width: res.width, height: res.height, frames: res.frames, operation: res.operation },
              } as any,
            }])
          } else {
            setMessages(prev => prev.filter(m => !m.id.includes('-progress')))
            setMessages(prev => [...prev, { id: `n-${Date.now()}-result`, role: 'nexus', text: msg }])
          }
        } else {
          setMessages(prev => prev.filter(m => !m.id.includes('-progress')))
          setMessages(prev => [...prev, { id: `n-${Date.now()}-err`, role: 'nexus', text: `❌ ${res.message}` }])
        }
        setTyping(false)
        return

      } else if (needsVideoEdit) {
        // ── 동영상 편집 (trim / compress / speed / subtitle) ──────
        const opLabel = wantTrim ? '구간 자르기' : wantCompress ? '용량 압축' : wantSpeed ? '속도 변환' : '자막 삽입'
        const opKey   = wantTrim ? 'video_trim'  : wantCompress ? 'video_compress' : wantSpeed ? 'video_speed' : 'video_subtitle'
        setMessages(prev => [
          ...prev,
          { id: `u-${Date.now()}`, role: 'user', text: `🎬 ${file.name}\n${text}` },
          { id: `n-${Date.now()}-progress`, role: 'nexus', text: `⏳ ${opLabel} 중... (ffmpeg 처리 중이에요)` },
        ])
        const payload = {
          files: allFiles.map(f => ({ name: f.name, mime_type: f.mimeType, data: f.dataUrl })),
          operation: opKey,
          query: text,
          params: {},
        }
        const res = await fetch('http://127.0.0.1:17891/api/file/process', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json', ...await getAuthHeader() },
          body: JSON.stringify(payload),
        }).then(r => r.json()).catch(() => ({ success: false, message: '영상 편집 실패' }))

        if (res.success && res.data && res.file_name) {
          const mimeType = res.mime_type ?? 'video/mp4'
          const byteString = atob(res.data)
          const ab = new ArrayBuffer(byteString.length)
          const ia = new Uint8Array(ab)
          for (let i = 0; i < byteString.length; i++) ia[i] = byteString.charCodeAt(i)
          const blob = new Blob([ab], { type: mimeType })
          const url = URL.createObjectURL(blob)
          setMessages(prev => prev.filter(m => !m.id.includes('-progress')))
          setMessages(prev => [...prev, {
            id: `n-${Date.now()}-result`,
            role: 'nexus',
            text: res.message ?? `${opLabel} 완료!`,
            inlineCard2: {
              type: 'file_result',
              data: { fileName: res.file_name, url, mimeType, operation: res.operation },
            } as any,
          }])
        } else {
          setMessages(prev => prev.filter(m => !m.id.includes('-progress')))
          setMessages(prev => [...prev, { id: `n-${Date.now()}-err`, role: 'nexus', text: `❌ ${res.message ?? opLabel + ' 실패'}` }])
        }
        setTyping(false)
        return

      } else if (isVideoFile(file)) {
        // ── 동영상 → 내용 분석 요청이면 백엔드 Whisper 전사, 아니면 변환 안내 ──
        const wantAnalyze = /요약|내용|설명|분석|뭐|무슨|어떤|정리|요점|핵심|전사|summarize|summary|content|what|explain|transcript|analyze|analyse/i.test(text)

        // 텍스트 없이 영상만 첨부된 경우 → 의도 확인 팝업
        if (!text && !wantAnalyze) {
          setVideoIntentPending({ file, extra: (extraFiles ?? []) as AttachedFile[] })
          setTyping(false)
          return
        }

        if (wantAnalyze || !text) {
          // 의존성 사전 체크
          const depsCheck = await fetch('http://127.0.0.1:17891/api/video/check-deps').then(r => r.json()).catch(() => null)
          if (depsCheck && !depsCheck.ready) {
            const hint = depsCheck.message ?? '영상 분석 도구가 설치되지 않았습니다.'
            const installHint = depsCheck.install_hint?.ffmpeg ?? ''
            setMessages(prev => [
              ...prev,
              { id: `u-${Date.now()}`, role: 'user', text: `🎬 ${file.name}${text ? '\n' + text : ''}` },
              { id: `n-${Date.now()}`, role: 'nexus', text: `⚠️ **영상 분석 불가**\n\n${hint}\n\n${installHint ? `📦 설치 방법:\n${installHint}` : ''}` },
            ])
            setTyping(false)
            return
          }

          // 진행 중 메시지 먼저 표시
          setMessages(prev => [
            ...prev,
            { id: `u-${Date.now()}`, role: 'user', text: `🎬 ${file.name}${text ? '\n' + text : ''}` },
            { id: `n-${Date.now()}-progress`, role: 'nexus', text: `⏳ 영상 분석 중... (파일 크기: ${(file.size / 1024 / 1024).toFixed(1)}MB)\n음성을 텍스트로 전사하거나 내장 자막을 추출하고 있어요.` },
          ])

          try {
            const lang = userLang ?? 'ko'
            const res = await fetch('http://127.0.0.1:17891/api/video/analyze-file', {
              method: 'POST',
              headers: { 'Content-Type': 'application/json', ...await getAuthHeader() },
              body: JSON.stringify({
                file_data: file.dataUrl,
                file_name: file.name,
                lang,
                query: text || (lang === 'en' ? 'Summarize this video' : '이 영상 내용을 요약해줘'),
              }),
            }).then(r => r.json()).catch(() => ({ success: false, message: '영상 분석 요청 실패' }))

            analysisResult = res.message ?? (res.success ? '영상 분석 완료' : '영상 분석에 실패했습니다.')
          } catch (e) {
            analysisResult = `영상 분석 중 오류: ${e instanceof Error ? e.message : String(e)}`
          }

          // progress 메시지 교체
          setMessages(prev => prev.map(m =>
            m.id?.endsWith('-progress') ? { ...m, text: analysisResult } : m
          ))
          appendHistory({ id: `${Date.now()}`, ts: Date.now(), q: `🎬 ${file.name} - ${text || '영상 분석'}`, a: analysisResult.slice(0, 300) })
          setTyping(false)
          return

        } else {
          // 변환/편집 목적 → 기존 안내
          analysisResult = `🎬 **${file.name}** (${(file.size / 1024 / 1024).toFixed(1)}MB)\n\n다음 작업을 요청할 수 있어요:\n• "내용 요약해줘" — AI 영상 내용 분석\n• "30초부터 2분까지 잘라줘" — 구간 자르기\n• "용량 절반으로 줄여줘" — 파일 압축\n• "2배속으로 만들어줘" — 속도 변환\n• "자막 파일이랑 합쳐줘" — 자막 삽입 (.srt 함께 첨부)`
        }

      } else if (file.fileType === 'image') {
        // ── 이미지 → GPT-4o Vision ───────────────────────────────
        const { callGroqVision } = await import('../../lib/nexus/gemini_engine')
        const base64 = file.dataUrl.split(',')[1] ?? file.dataUrl
        const question = text || `이 이미지를 ${mode} 모드로 분석해줘. 내용, 특징, 시사점을 상세하게 설명해줘.`
        analysisResult = await callGroqVision(base64, question)

      } else if (wantEdit && (file.fileType === 'spreadsheet' || file.fileType === 'document')) {
        // ── 편집 의도 + 문서 → 백엔드 AI 편집 ──────────────────
        const { uploadDocFile, aiEditDoc, docTypeLabel } = await import('../../lib/nexus/docEditor')
        const fileExt = file.name.substring(file.name.lastIndexOf('.')).toLowerCase()
        const docLabel = docTypeLabel(fileExt)

        analysisResult = `⏳ ${docLabel} 파일을 백엔드에 업로드 중...`
        setMessages(prev => [
          ...prev,
          { id: `u-${Date.now()}`, role: 'user', text: `📎 ${file.name}\n${text}` },
          { id: `n-${Date.now()}-progress`, role: 'nexus', text: analysisResult },
        ])

        // 1. dataUrl → Blob → File 객체
        const response = await fetch(file.dataUrl)
        const blob = await response.blob()
        const fileObj = new File([blob], file.name, { type: file.mimeType })

        // 2. 백엔드 업로드
        const uploadResult = await uploadDocFile(fileObj)
        if (!uploadResult.success) {
          throw new Error(uploadResult.message ?? '업로드 실패')
        }

        // 미리보기 정보 표시
        let previewInfo = ''
        if (uploadResult.preview) {
          if (uploadResult.preview.sheets) {
            previewInfo = `\n📋 시트: ${uploadResult.preview.sheets.join(', ')} | 총 ${uploadResult.preview.total_rows ?? 0}행`
          } else if (uploadResult.preview.text) {
            previewInfo = `\n📄 내용 미리보기: ${uploadResult.preview.text.slice(0, 100)}...`
          }
        }

        // 3. AI 편집 실행
        setMessages(prev => prev.map(m =>
          m.id === `n-${Date.now()}-progress`
            ? { ...m, text: `📤 업로드 완료${previewInfo}\n⚙️ AI가 "${text}" 작업을 실행 중...` }
            : m
        ))

        const editResult = await aiEditDoc(uploadResult.file_path, text)
        if (!editResult.success) {
          throw new Error(editResult.message ?? '편집 실패')
        }

        const opsInfo = editResult.operations_count
          ? `\n적용된 연산: ${editResult.operations_count}개 (${(editResult.operations ?? []).slice(0, 5).join(', ')}${(editResult.operations?.length ?? 0) > 5 ? '...' : ''})`
          : ''

        analysisResult = `✅ **문서 편집 완료**\n\n📝 ${editResult.summary}${opsInfo}\n\n💾 저장 위치: \`${editResult.out_path}\`\n\n바탕화면에서 파일을 확인하세요.`

        // progress 메시지 교체
        setMessages(prev => prev.map(m =>
          m.id?.endsWith('-progress') ? { ...m, text: analysisResult } : m
        ))
        appendHistory({ id: `${Date.now()}`, ts: Date.now(), q: `📎 ${file.name} - ${text}`, a: editResult.summary })
        setTyping(false)
        return

      } else {
        // ── 분석 의도 → 텍스트 추출 후 LLM 분석 ────────────────
        let docContent = file.text ?? ''

        // 백엔드 업로드로 더 정확한 텍스트 추출 시도
        if (file.fileType !== 'other') {
          try {
            const { uploadDocFile } = await import('../../lib/nexus/docEditor')
            const resp = await fetch(file.dataUrl)
            const blob = await resp.blob()
            const fileObj = new File([blob], file.name, { type: file.mimeType })
            const uploaded = await uploadDocFile(fileObj)
            if (uploaded.success && uploaded.preview?.text) {
              docContent = uploaded.preview.text
            } else if (uploaded.success && uploaded.preview?.rows) {
              // Excel 미리보기 → 탭 구분 텍스트
              docContent = (uploaded.preview.rows as string[][]).map(r => r.join('\t')).join('\n')
            }
          } catch { /* 백엔드 없으면 기존 file.text 사용 */ }
        }

        const truncated = docContent.slice(0, 8000)
        const userQ = text || `이 문서를 ${mode} 모드로 분석해줘. 핵심 내용, 중요 데이터, 인사이트를 정리해줘.`
        const prompt = truncated
          ? `[첨부 파일: ${file.name}]\n\n${truncated}\n\n---\n사용자 질문: ${userQ}`
          : `[첨부 파일: ${file.name} — 텍스트 추출 불가]\n\n사용자 질문: ${userQ}`

        // Perplexity/Groq API 호출
        const pplxKey = localStorage.getItem('nexus-pplx-key') ?? ''
        const openaiKey = localStorage.getItem('nexus-openai-key') ?? ''

        if (pplxKey) {
          const res = await fetch('https://api.perplexity.ai/chat/completions', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${pplxKey}` },
            body: JSON.stringify({
              model: 'sonar-pro',
              messages: [{ role: 'user', content: prompt }],
              max_tokens: 2000,
            }),
            signal: AbortSignal.timeout(30000),
          })
          if (res.ok) {
            const d = await res.json() as { choices?: Array<{ message?: { content?: string } }> }
            analysisResult = d.choices?.[0]?.message?.content?.trim() ?? '분석 결과를 가져오지 못했습니다.'
          }
        } else if (openaiKey) {
          const res = await fetch('https://api.openai.com/v1/chat/completions', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${openaiKey}` },
            body: JSON.stringify({
              model: 'gpt-4o',
              messages: [{ role: 'user', content: prompt }],
              max_tokens: 2000,
            }),
            signal: AbortSignal.timeout(30000),
          })
          if (res.ok) {
            const d = await res.json() as { choices?: Array<{ message?: { content?: string } }> }
            analysisResult = d.choices?.[0]?.message?.content?.trim() ?? '분석 결과를 가져오지 못했습니다.'
          }
        } else {
          // API 키 없으면 백엔드 Groq LLM으로 폴백
          try {
            const res = await fetch('http://127.0.0.1:17891/api/llm/chat', {
              method: 'POST',
              headers: { 'Content-Type': 'application/json', ...await getAuthHeader() },
              body: JSON.stringify({ messages: [{ role: 'user', content: prompt }], max_tokens: 2000 }),
            }).then(r => r.json()).catch(() => null)
            if (res?.content) {
              analysisResult = res.content
            } else {
              analysisResult = '⚠️ API 키가 없습니다. 설정에서 Perplexity 또는 OpenAI API 키를 입력해주세요.'
            }
          } catch {
            analysisResult = '⚠️ API 키가 없습니다. 설정에서 Perplexity 또는 OpenAI API 키를 입력해주세요.'
          }
        }
      }

      // ── 웹서치 병행 ──────────────────────────────────────────
      if (wantSearch) {
        const searchQuery = text.replace(/검색|웹|최신|찾아|서치|search/gi, '').trim() || file.name.replace(/\.[^.]+$/, '')
        try {
          const tavilyKey = localStorage.getItem('nexus-tavily-key') ?? ''
          if (tavilyKey) {
            const res = await fetch('https://api.tavily.com/search', {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({ api_key: tavilyKey, query: searchQuery, max_results: 5, search_depth: 'advanced' }),
            })
            if (res.ok) {
              const data = await res.json() as { results?: Array<{ title: string; url: string; content: string }> }
              const searchSummary = (data.results ?? []).map(r => `• ${r.title}\n  ${r.content.slice(0, 150)}`).join('\n\n')
              if (searchSummary) {
                analysisResult += `\n\n---\n🔍 **웹 검색 결과** (${searchQuery})\n\n${searchSummary}`
              }
            }
          }
        } catch { /* 검색 실패 무시 */ }
      }
    } catch (e) {
      analysisResult = `파일 처리 중 오류가 발생했습니다: ${e instanceof Error ? e.message : String(e)}`
    }

    const displayText = text || `📎 ${file.name} 분석`
    const icon = file.fileType === 'image' ? '🖼️' : file.fileType === 'spreadsheet' ? '📊' : '📄'
    setMessages(prev => [
      ...prev,
      { id: `u-${Date.now()}`, role: 'user', text: `${icon} ${file.name}${text ? '\n' + text : ''}` },
      { id: `n-${Date.now()}`, role: 'nexus', text: analysisResult },
    ])
    appendHistory({ id: `${Date.now()}`, ts: Date.now(), q: displayText, a: analysisResult.slice(0, 300) })
    setTyping(false)
  }, [])

  /* 캐릭터 클릭 — Orb 클릭 시 최소화 해제만 */
  const handleCharacterClick = () => {
    setMinimized(false)
  }

  if (minimized) {
    return (
      <motion.div
        initial={{ scale: 0 }}
        animate={{ scale: 1 }}
        onClick={() => setMinimized(false)}
        title={userLang === 'ko' ? '클릭하여 Nexus 열기' : 'Click to open Nexus'}
        style={{
          position: 'fixed',
          bottom: 24, right: 24,
          width: 52, height: 52,
          borderRadius: '50%',
          background: `linear-gradient(135deg, ${primaryColor}, ${accentColor})`,
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          cursor: 'pointer',
          boxShadow: `0 4px 20px ${primaryColor}66`,
          fontSize: 22,
          zIndex: 9999,
        }}
      >
        ◉
      </motion.div>
    )
  }

  const displayInput = input

  /* ── 새 레이아웃: 캐릭터 중심 플로팅 오버레이 ──
     - 화면 우측 하단에 고정
     - 캐릭터가 전체 중심
     - 채팅창은 캐릭터 왼쪽에 슬라이드 인
     - 상태 버튼은 캐릭터 오른쪽 수직 배치
  ── */
  /* 드래그 가능한 전체 블록 */
  const btnList = [
    ...(speaking ? [{
      icon: '⏹', active: true, color: '#ef4444',
      onClick: () => { stopSpeaking(); setSpeaking(false) }, tip: '음성 중지',
    }] : [
      { icon: soundEnabled ? '🔊' : '🔇', active: soundEnabled, color: soundEnabled ? primaryColor : '#6b7280',
        onClick: () => setSoundEnabled(p => { const next = !p; localStorage.setItem('nexus-sound', next ? 'on' : 'off'); return next }),
        tip: userLang === 'en' ? (soundEnabled ? 'Mute AI' : 'Unmute AI') : (soundEnabled ? 'AI 소리 끄기' : 'AI 소리 켜기') },
    ]),
    { icon: isActive ? '💬' : '😴',    active: isActive,     color: isActive ? primaryColor : '#6b7280',
      onClick: () => { setIsActive(p => !p); if (isActive) stopSpeaking() },
      tip: userLang === 'en' ? (isActive ? 'Deactivate' : 'Activate') : (isActive ? '비활성화' : '활성화') },
    { icon: '⚙️', active: false,       color: primaryColor,  onClick: () => setSettingsOpen(true),
      tip: userLang === 'en' ? 'Settings' : '설정' },
    { icon: '—',  active: false,       color: '#6b7280',     onClick: () => setMinimized(true),
      tip: userLang === 'en' ? 'Minimize' : '최소화' },
    { icon: '✕',  active: false,       color: '#ef4444',     onClick: async () => {
      const { getCurrentWindow } = await import('@tauri-apps/api/window')
      getCurrentWindow().close()
    }, tip: userLang === 'en' ? 'Close' : '닫기' },
  ]

  if (!isOnboarded) {
    // character 창(380px 위젯)에서는 온보딩 렌더링 안 함 — main 창(760px)에서만
    if (window.innerWidth <= 420) return null
    return <OnboardingFlow onComplete={handleOnboardingComplete} />
  }

  if (!isLoggedIn) {
    return <LoginScreen />
  }

  return (
    <>
    {/* ── 전체 화면 패널 레이아웃 (v2.6) ── */}
    <style>{`
      @keyframes orb-speak { 0% { transform: scale(1); } 100% { transform: scale(1.06); } }
      @keyframes beam-sweep { 0%,100% { opacity:0.0; transform: translateX(-120px); } 50% { opacity:1; transform: translateX(240px); } }
    `}</style>
    <div style={{
      position: 'fixed', inset: 0,
      display: 'flex', flexDirection: 'column',
      background: 'rgba(6,6,18,0.62)',
      backdropFilter: 'blur(32px)',
      WebkitBackdropFilter: 'blur(32px)',
      borderRadius: 16,
      border: `1px solid ${primaryColor}33`,
      boxShadow: `0 0 0 1px ${primaryColor}18, 0 8px 48px rgba(0,0,0,0.55)`,
      overflow: 'hidden',
      zIndex: 9999,
    }}>

      {/* ── 상단 헤더: Orb + 상태바 + 컨트롤 ── */}
      <div style={{
        display: 'flex', alignItems: 'center', gap: 10,
        padding: '10px 14px 8px',
        background: 'rgba(8,8,24,0.85)',
        borderBottom: `1px solid ${primaryColor}28`,
        flexShrink: 0,
        position: 'relative', overflow: 'hidden',
      }}>
        {/* 빛줄기 */}
        {beamEnabled && (
          <div style={{
            position: 'absolute', top: 0, bottom: 0, left: 0, width: 80,
            background: `linear-gradient(90deg, transparent, ${primaryColor}18, transparent)`,
            animation: 'beam-sweep 5s ease-in-out infinite',
            pointerEvents: 'none',
          }} />
        )}

        {/* ─ Siri Orb ─ */}
        <div
          onClick={handleCharacterClick}
          title="클릭해서 대화하기"
          style={{ flexShrink: 0, cursor: 'pointer', position: 'relative', zIndex: 1 }}
        >
          <motion.div
            animate={speaking
              ? { scale: [1, 1.08, 1], boxShadow: [`0 0 14px ${primaryColor}88`, `0 0 28px ${primaryColor}dd`, `0 0 14px ${primaryColor}88`] }
              : { scale: [1, 1.03, 1], boxShadow: [`0 0 12px ${primaryColor}66`, `0 0 20px ${primaryColor}99`, `0 0 12px ${primaryColor}66`] }
            }
            transition={{ duration: speaking ? 0.5 : 3.5, repeat: Infinity, ease: 'easeInOut' }}
            style={{
              width: 46, height: 46, borderRadius: '50%',
              background: `radial-gradient(circle at 38% 32%, ${accentColor}ee, ${primaryColor}cc 40%, ${primaryColor}88 68%, ${primaryColor}44)`,
              display: 'flex', alignItems: 'center', justifyContent: 'center',
              fontSize: 18,
              opacity: isActive ? 1 : 0.55,
            }}
          >
            {!isActive && '😴'}
          </motion.div>
          {speaking && (
            <div style={{
              position: 'absolute', inset: -4, borderRadius: '50%',
              border: `1.5px solid ${primaryColor}66`,
              animation: 'orb-speak 0.5s ease-in-out infinite alternate',
              pointerEvents: 'none',
            }} />
          )}
        </div>

        {/* ─ 상태 정보 ─ */}
        <div style={{ flex: 1, minWidth: 0, zIndex: 1 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 5 }}>
            {activePersona && <span style={{ fontSize: 12 }}>{activePersona.emoji}</span>}
            <span style={{
              fontSize: 12, fontWeight: 700,
              color: activePersona ? activePersona.color : primaryColor,
              overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
            }}>
              {activePersona?.name ?? (userLang === 'en' ? 'General Mode' : '일반 모드')}
            </span>
          </div>
          <div style={{ fontSize: 9.5, color: 'rgba(255,255,255,0.35)', marginTop: 1 }}>
            {userLang === 'en'
              ? `${dailyUsedCount} / 15 today`
              : `오늘 ${dailyUsedCount} / 15회`}
          </div>
        </div>

        {/* ─ 컨트롤 버튼 ─ */}
        <div style={{ display: 'flex', gap: 4, zIndex: 1 }}>
          {/* 빛줄기 토글 */}
          <motion.button
            whileTap={{ scale: 0.9 }} whileHover={{ scale: 1.1 }}
            onClick={() => setBeamEnabled(p => { const next = !p; localStorage.setItem('nexus-beam', next ? 'on' : 'off'); return next })}
            title={beamEnabled ? '빛줄기 끄기' : '빛줄기 켜기'}
            style={{
              width: 28, height: 28, borderRadius: '50%', border: 'none', cursor: 'pointer',
              background: beamEnabled ? `${primaryColor}28` : 'rgba(255,255,255,0.08)',
              color: beamEnabled ? primaryColor : 'rgba(255,255,255,0.45)',
              fontSize: 12,
            }}
          >✦</motion.button>
          {btnList.map(btn => (
            <motion.button
              key={btn.tip}
              whileTap={{ scale: 0.9 }} whileHover={{ scale: 1.1 }}
              onClick={btn.onClick}
              title={btn.tip}
              style={{
                width: 28, height: 28, borderRadius: '50%', border: 'none', cursor: 'pointer',
                background: btn.active ? `${btn.color}28` : 'rgba(255,255,255,0.08)',
                color: btn.active ? btn.color : 'rgba(255,255,255,0.45)',
                fontSize: btn.icon === '—' ? 16 : 12,
              } as React.CSSProperties}
            >
              {btn.icon}
            </motion.button>
          ))}
        </div>
      </div>

      {/* ── TTS 말풍선 ── */}
      <AnimatePresence>
        {bubbleText && (
          <motion.div
            key="bubble"
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: 'auto' }}
            exit={{ opacity: 0, height: 0 }}
            style={{
              overflow: 'hidden',
              background: `${primaryColor}12`,
              borderBottom: `1px solid ${primaryColor}28`,
              flexShrink: 0,
            }}
          >
            <div style={{
              padding: '10px 14px',
              fontSize: 12.5, color: 'rgba(255,255,255,0.93)',
              lineHeight: 1.6, wordBreak: 'keep-all',
              display: 'flex', alignItems: 'flex-start', gap: 8,
            }}>
              <span style={{ fontSize: 14, flexShrink: 0, marginTop: 1 }}>💬</span>
              <span style={{ flex: 1 }}>
                {bubbleExpanded ? bubbleText : bubbleText.slice(0, 200) + (bubbleText.length > 200 ? '…' : '')}
              </span>
              <div style={{ display: 'flex', gap: 3, flexShrink: 0 }}>
                {bubbleText.length > 200 && (
                  <button onClick={() => setBubbleExpanded(e => !e)}
                    style={{ background: 'none', border: 'none', cursor: 'pointer', color: primaryColor, fontSize: 11, padding: '2px 4px' }}>
                    {bubbleExpanded ? '▲' : '▼'}
                  </button>
                )}
                <button onClick={() => navigator.clipboard.writeText(bubbleText).catch(() => {})}
                  style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'rgba(255,255,255,0.35)', fontSize: 11, padding: '2px 4px' }}>
                  ⎘
                </button>
                <button onClick={() => { setBubbleText(''); setBubbleExpanded(false) }}
                  style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'rgba(255,255,255,0.35)', fontSize: 11, padding: '2px 4px' }}>
                  ✕
                </button>
              </div>
            </div>
          </motion.div>
        )}
      </AnimatePresence>

      {/* ── 비활성화 상태 배너 ── */}
      {!isActive && (
        <div style={{
          padding: '5px 14px', fontSize: 10,
          color: 'rgba(255,255,255,0.45)',
          background: 'rgba(100,100,120,0.2)',
          borderBottom: '1px solid rgba(255,255,255,0.06)',
          flexShrink: 0, textAlign: 'center',
        }}>
          😴 비활성화 상태 — {assistantName}을(를) 불러주세요
        </div>
      )}

      {/* ── 집중 모드 배너 ── */}
      {focusEndMs && (
        <div style={{
          padding: '5px 14px', fontSize: 10,
          color: primaryColor,
          background: `${primaryColor}12`,
          borderBottom: `1px solid ${primaryColor}28`,
          flexShrink: 0,
          display: 'flex', alignItems: 'center', gap: 6,
        }}>
          <span>🎯</span>
          <span>집중 모드 — {Math.max(0, Math.ceil((focusEndMs - Date.now()) / 60_000))}분 남음</span>
        </div>
      )}

      {/* ── 동적 결과창 (floatingPreview 데이터 표시) ── */}
      <AnimatePresence>
        {floatingPreview && floatingPreview.length > 0 && (
          <motion.div
            key="result-panel"
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: 'auto' }}
            exit={{ opacity: 0, height: 0 }}
            style={{
              overflow: 'hidden', flexShrink: 0,
              borderBottom: `1px solid ${primaryColor}28`,
              maxHeight: 220,
              overflowY: 'auto',
            }}
          >
            <div style={{ padding: '10px 14px' }}>
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 8 }}>
                <span style={{ fontSize: 11, color: primaryColor, fontWeight: 800 }}>
                  {floatingPreview.some(x => x.isMap && x.mapType === 'directions') ? '🗺️ 길찾기 결과'
                    : floatingPreview.some(x => x.isMap) ? '🗺️ 지도 결과'
                    : floatingPreview[0]?.isVideo ? '🎬 영상 결과'
                    : '🔍 검색 결과'} ({floatingPreview.length})
                </span>
                <button onClick={() => setFloatingPreview(null)}
                  style={{ background: 'none', border: 'none', color: 'rgba(255,255,255,0.4)', cursor: 'pointer', fontSize: 13, lineHeight: 1 }}>✕</button>
              </div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 5 }}>
                {floatingPreview.slice(0, 6).map((item, i) => {
                  const isYt = item.url.includes('youtube.com') || item.url.includes('youtu.be')
                  const modeEmoji: Record<string, string> = { transit: '🚌', car: '🚗', walk: '🚶', bicycle: '🚲', ktx: '🚂' }
                  const mode = (item as any).mode as string | undefined
                  return (
                    <motion.div
                      key={i}
                      initial={{ opacity: 0, x: -10 }}
                      animate={{ opacity: 1, x: 0 }}
                      transition={{ delay: i * 0.05 }}
                      onClick={() => window.open(item.url, '_blank')}
                      style={{
                        display: 'flex', alignItems: 'center', gap: 7, cursor: 'pointer',
                        padding: '5px 6px', borderRadius: 8,
                        background: 'rgba(255,255,255,0.04)',
                        border: `1px solid ${primaryColor}22`,
                        transition: 'all 0.15s',
                      }}
                      whileHover={{ background: `${primaryColor}1a`, borderColor: `${primaryColor}55` } as any}
                    >
                      <div style={{
                        width: 20, height: 20, borderRadius: 5,
                        background: isYt ? '#e53e3e22' : `${primaryColor}22`,
                        display: 'flex', alignItems: 'center', justifyContent: 'center',
                        flexShrink: 0, fontSize: 10, fontWeight: 700,
                        color: isYt ? '#e53e3e' : primaryColor,
                      }}>
                        {isYt ? '▶' : mode ? modeEmoji[mode] ?? '→' : (i + 1)}
                      </div>
                      <div style={{ flex: 1, minWidth: 0 }}>
                        <div style={{ fontSize: 10.5, color: 'rgba(255,255,255,0.88)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', fontWeight: 600 }}>
                          {item.title}
                        </div>
                        <div style={{ fontSize: 9, color: 'rgba(255,255,255,0.3)', marginTop: 1 }}>
                          {(() => { try { return new URL(item.url).hostname.replace('www.','') } catch { return '' } })()}
                        </div>
                      </div>
                      <span style={{ fontSize: 9, color: primaryColor, flexShrink: 0 }}>↗</span>
                    </motion.div>
                  )
                })}
              </div>
            </div>
          </motion.div>
        )}
      </AnimatePresence>

      {/* ── 메인 콘텐츠: 좌우 분할 ── */}
      <div style={{ flex: 1, overflow: 'hidden', display: 'flex', flexDirection: 'row' }}>

        {/* ── 좌: 채팅 영역 ── */}
        <div style={{ flex: 1, overflow: 'hidden', borderRight: `1px solid ${primaryColor}20`, display: 'flex', flexDirection: 'column' }}>
          <ChatBubble
            messages={messages}
            typing={typing}
            input={displayInput}
            onInputChange={v => setInput(v)}
            onSend={handleSend}
            onSendWithFile={handleSendWithFileImpl}
            onRepair={handleRepair}
            assistantName={assistantName}
            lang={userLang}
            primaryColor={primaryColor}
            historyVersion={historyVersion}
            clarifyPending={!!clarifyPendingIntent}
            clarifyQuestion={clarifyPendingQuestion ?? ''}
            typingSteps={typingSteps}
            activePersona={activePersona ? { name: activePersona.name, emoji: activePersona.emoji, color: activePersona.color } : null}
            subscriptionStatus={subscriptionStatus}
            dailyUsed={dailyUsedCount}
            onPersonaClick={handlePersonaChipClick}
            onPersonaSelect={handlePersonaSelect}
            onRetry={handleRetry}
            onOpenSettings={handleOpenSettings}
            embedded={true}
          />
        </div>

        {/* ── 우: 기본 결과창 (항상 표시) ── */}
        <div style={{
          width: 172, flexShrink: 0,
          display: 'flex', flexDirection: 'column',
          background: 'rgba(0,0,0,0.18)',
          overflow: 'hidden',
        }}>
          <div style={{ padding: '8px 10px 4px', borderBottom: `1px solid ${primaryColor}22`, flexShrink: 0 }}>
            <span style={{ fontSize: 9.5, fontWeight: 800, color: `${primaryColor}99`, letterSpacing: '0.06em' }}>
              📋 {userLang === 'en' ? 'LAST RESULT' : '최근 결과'}
            </span>
          </div>
          <div style={{ flex: 1, overflowY: 'auto', padding: '6px 10px', scrollbarWidth: 'none' }}>
            {(() => {
              const lastMsg = messages.filter(m => m.role === 'nexus' && m.text).slice(-1)[0]
              if (!lastMsg) return (
                <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.22)', marginTop: 8, textAlign: 'center', lineHeight: 1.7 }}>
                  {userLang === 'en' ? 'Results appear\nhere after tasks' : '작업 완료 후\n결과 요약이\n여기 표시됩니다'}
                </div>
              )
              const lines = lastMsg.text.split('\n').filter(l => l.trim()).slice(0, 5)
              return lines.map((line, i) => (
                <div key={i} style={{ fontSize: 10, color: i === 0 ? 'rgba(255,255,255,0.88)' : 'rgba(255,255,255,0.55)', marginBottom: 4, lineHeight: 1.5, wordBreak: 'break-all' }}>
                  {i === 0 ? <span style={{ fontWeight: 700 }}>{line.slice(0, 40)}{line.length > 40 ? '…' : ''}</span> : `• ${line.slice(0, 32)}${line.length > 32 ? '…' : ''}`}
                </div>
              ))
            })()}
          </div>
          {/* 기본 결과창 하단 — 마지막 액션 시간 */}
          {messages.filter(m => m.role === 'nexus').length > 0 && (
            <div style={{ padding: '4px 10px 6px', borderTop: `1px solid ${primaryColor}18`, flexShrink: 0 }}>
              <span style={{ fontSize: 9, color: 'rgba(255,255,255,0.2)' }}>
                {new Date().toLocaleTimeString('ko-KR', { hour: '2-digit', minute: '2-digit' })}
              </span>
            </div>
          )}
        </div>
      </div>

      {/* ── 동적 결과창 (결과 완료 시만 슬라이드인) ── */}
      <AnimatePresence>
        {dynamicResult && (
          <motion.div
            key="dynamic-result"
            initial={{ opacity: 0, y: 60 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: 60 }}
            transition={{ type: 'spring', stiffness: 380, damping: 34 }}
            style={{
              position: 'absolute', left: 0, right: 0, bottom: 0,
              zIndex: 200,
              background: 'rgba(8,8,22,0.97)',
              borderTop: `2px solid ${primaryColor}88`,
              boxShadow: `0 -8px 32px rgba(0,0,0,0.7), 0 0 0 1px ${primaryColor}22`,
              maxHeight: '62%',
              display: 'flex', flexDirection: 'column',
              backdropFilter: 'blur(24px)',
            }}
          >
            {/* 헤더 */}
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '10px 14px 8px', borderBottom: `1px solid ${primaryColor}28`, flexShrink: 0 }}>
              <span style={{ fontSize: 13 }}>{dynamicResult.icon}</span>
              <span style={{ flex: 1, fontSize: 12, fontWeight: 800, color: dynamicResult.success ? '#22c55e' : '#f59e0b' }}>
                {dynamicResult.title}
              </span>
              <button
                onClick={() => setDynamicResult(null)}
                style={{ background: 'none', border: 'none', color: 'rgba(255,255,255,0.4)', cursor: 'pointer', fontSize: 14, padding: '0 2px' }}
              >✕</button>
            </div>

            {/* 내용 */}
            <div style={{ flex: 1, overflowY: 'auto', padding: '10px 14px', scrollbarWidth: 'none' }}>
              {/* 요약 숫자 칩 */}
              {dynamicResult.stats && dynamicResult.stats.length > 0 && (
                <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap', marginBottom: 10 }}>
                  {dynamicResult.stats.map((s, i) => (
                    <div key={i} style={{
                      background: `${primaryColor}18`, border: `1px solid ${primaryColor}33`,
                      borderRadius: 8, padding: '4px 10px',
                      fontSize: 11, fontWeight: 700, color: primaryColor,
                    }}>
                      {s.label}: <span style={{ color: '#fff' }}>{s.value}</span>
                    </div>
                  ))}
                </div>
              )}
              {/* 상세 항목 */}
              {dynamicResult.items && dynamicResult.items.length > 0 && (
                <div style={{ display: 'flex', flexDirection: 'column', gap: 4, marginBottom: 8 }}>
                  {dynamicResult.items.map((item, i) => (
                    <div key={i} style={{ fontSize: 11, color: 'rgba(255,255,255,0.75)', display: 'flex', gap: 6, alignItems: 'flex-start' }}>
                      <span style={{ color: primaryColor, flexShrink: 0 }}>•</span>
                      <span style={{ wordBreak: 'break-all' }}>{item}</span>
                    </div>
                  ))}
                </div>
              )}
              {/* 동영상 파일 결과 */}
              {dynamicResult.fileInfo && (
                <div style={{
                  background: 'rgba(255,255,255,0.04)', border: `1px solid ${primaryColor}33`,
                  borderRadius: 10, padding: '8px 12px', marginBottom: 8,
                  display: 'flex', alignItems: 'center', gap: 10,
                }}>
                  <span style={{ fontSize: 22, flexShrink: 0 }}>
                    {dynamicResult.fileInfo.mimeType?.startsWith('video/') ? '🎬' : dynamicResult.fileInfo.mimeType?.startsWith('image/') ? '🖼️' : '📄'}
                  </span>
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ fontSize: 11, fontWeight: 700, color: '#fff', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      {dynamicResult.fileInfo.name}
                    </div>
                    {dynamicResult.fileInfo.size && (
                      <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.45)', marginTop: 2 }}>
                        {dynamicResult.fileInfo.size}
                      </div>
                    )}
                    {dynamicResult.fileInfo.path && (
                      <div style={{ fontSize: 9.5, color: 'rgba(255,255,255,0.3)', marginTop: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                        📁 {dynamicResult.fileInfo.path}
                      </div>
                    )}
                  </div>
                </div>
              )}
              {/* 링크 목록 (검색 결과 등) */}
              {dynamicResult.links && dynamicResult.links.length > 0 && (
                <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
                  {dynamicResult.links.slice(0, 5).map((link, i) => (
                    <div
                      key={i}
                      onClick={() => window.open(link.url, '_blank')}
                      style={{
                        display: 'flex', alignItems: 'center', gap: 8,
                        padding: '5px 8px', borderRadius: 8, cursor: 'pointer',
                        background: 'rgba(255,255,255,0.04)',
                        border: `1px solid ${primaryColor}22`,
                        transition: 'background 0.15s',
                      }}
                      onMouseEnter={e => (e.currentTarget.style.background = `${primaryColor}18`)}
                      onMouseLeave={e => (e.currentTarget.style.background = 'rgba(255,255,255,0.04)')}
                    >
                      <span style={{ fontSize: 11, color: primaryColor, fontWeight: 700, flexShrink: 0 }}>{i + 1}</span>
                      <span style={{ flex: 1, fontSize: 11, color: 'rgba(255,255,255,0.85)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{link.title}</span>
                      <span style={{ fontSize: 9, color: primaryColor, flexShrink: 0 }}>↗</span>
                    </div>
                  ))}
                </div>
              )}
            </div>

            {/* 액션 버튼 */}
            <div style={{ display: 'flex', gap: 6, padding: '8px 14px 12px', borderTop: `1px solid ${primaryColor}22`, flexShrink: 0, flexWrap: 'wrap' }}>
              {dynamicResult.actions.map((act, i) => (
                <button
                  key={i}
                  onClick={act.onClick}
                  style={{
                    display: 'flex', alignItems: 'center', gap: 5,
                    padding: '6px 12px', borderRadius: 10, cursor: 'pointer',
                    background: i === 0 ? primaryColor : `${primaryColor}18`,
                    border: i === 0 ? 'none' : `1px solid ${primaryColor}44`,
                    color: i === 0 ? '#fff' : primaryColor,
                    fontSize: 11, fontWeight: 700, transition: 'all 0.15s',
                    whiteSpace: 'nowrap',
                  }}
                  onMouseEnter={e => { e.currentTarget.style.opacity = '0.82' }}
                  onMouseLeave={e => { e.currentTarget.style.opacity = '1' }}
                >
                  {act.icon} {act.label}
                </button>
              ))}
            </div>
          </motion.div>
        )}
      </AnimatePresence>

    </div>

    {/* ── 영상 의도 확인 팝업 ── */}
    {videoIntentPending && (
      <div style={{
        position: 'fixed', inset: 0, zIndex: 10100,
        background: 'rgba(0,0,0,0.55)', backdropFilter: 'blur(6px)',
        display: 'flex', alignItems: 'center', justifyContent: 'center',
      }}>
        <div style={{
          background: 'rgba(12,12,28,0.97)', border: `1.5px solid ${primaryColor}55`,
          borderRadius: 18, padding: '22px 24px', width: 300,
          boxShadow: `0 12px 40px ${primaryColor}44`,
        }}>
          <div style={{ fontSize: 13, fontWeight: 700, color: '#fff', marginBottom: 6 }}>🎬 영상을 어떻게 할까요?</div>
          <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.55)', marginBottom: 16 }}>{videoIntentPending.file.name}</div>
          {[
            { label: '🔍 내용 분석 / 요약', text: '이 영상 내용을 분석하고 요약해줘' },
            { label: '✂️ 구간 자르기', text: '이 영상에서 구간을 잘라줘' },
            { label: '📦 용량 압축', text: '이 영상 용량을 압축해줘' },
            { label: '⚡ 배속 변환', text: '이 영상 속도를 바꿔줘' },
          ].map(({ label, text }) => (
            <button key={label}
              onClick={() => { const { file, extra } = videoIntentPending; setVideoIntentPending(null); handleSendWithFileImpl(text, file, extra) }}
              style={{
                display: 'block', width: '100%', textAlign: 'left',
                background: `${primaryColor}18`, border: `1px solid ${primaryColor}33`,
                color: '#fff', fontSize: 12, fontWeight: 600,
                padding: '8px 12px', borderRadius: 10, marginBottom: 8, cursor: 'pointer',
                transition: 'background 0.15s',
              }}
              onMouseEnter={e => (e.currentTarget.style.background = `${primaryColor}35`)}
              onMouseLeave={e => (e.currentTarget.style.background = `${primaryColor}18`)}
            >{label}</button>
          ))}
          <button onClick={() => setVideoIntentPending(null)}
            style={{ width: '100%', background: 'transparent', border: 'none', color: 'rgba(255,255,255,0.4)', fontSize: 11, cursor: 'pointer', marginTop: 2 }}>
            취소
          </button>
        </div>
      </div>
    )}

    <SettingsModal
      open={settingsOpen}
      onClose={() => setSettingsOpen(false)}
      primaryColor={primaryColor}
      onPrimaryColorChange={(c) => {
        setPrimaryColor(c)
        localStorage.setItem('nexus-theme-color', c)
      }}
    />

    {/* ── 페르소나 선택 팝업 ── */}
    {showPersonaPopup && (
      <>
        <div
          onClick={() => setShowPersonaPopup(false)}
          style={{ position: 'fixed', inset: 0, zIndex: 10200, background: 'rgba(0,0,0,0.5)', backdropFilter: 'blur(4px)' }}
        />
        <div style={{
          position: 'fixed', top: '50%', left: '50%',
          transform: 'translate(-50%, -50%)',
          zIndex: 10201,
          background: 'rgba(10,10,24,0.98)',
          border: `1.5px solid ${primaryColor}44`,
          borderRadius: 18,
          padding: '20px 18px',
          width: 320,
          boxShadow: `0 16px 48px rgba(0,0,0,0.8), 0 0 0 1px ${primaryColor}22`,
        }}>
          <div style={{ fontSize: 13, fontWeight: 800, color: primaryColor, marginBottom: 4 }}>
            🤖 AI 모드 선택
          </div>
          <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.4)', marginBottom: 14 }}>
            현재: {activePersona?.name ?? '일반 모드'}
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 7 }}>
            {/* 기본 일반 모드 */}
            {[
              ...personaListData,
            ].map(p => {
              const isCurrentPersona = (activePersona?.id ?? 'nexus') === p.id
              return (
                <button
                  key={p.id}
                  onClick={() => handlePersonaSelect(p.id)}
                  style={{
                    display: 'flex', alignItems: 'center', gap: 10,
                    padding: '10px 12px', borderRadius: 12,
                    background: isCurrentPersona ? `${p.color}22` : 'rgba(255,255,255,0.04)',
                    border: isCurrentPersona ? `1.5px solid ${p.color}88` : '1px solid rgba(255,255,255,0.1)',
                    cursor: 'pointer', textAlign: 'left', transition: 'all 0.15s',
                  }}
                  onMouseEnter={e => { if (!isCurrentPersona) e.currentTarget.style.background = 'rgba(255,255,255,0.08)' }}
                  onMouseLeave={e => { if (!isCurrentPersona) e.currentTarget.style.background = 'rgba(255,255,255,0.04)' }}
                >
                  <span style={{ fontSize: 20, flexShrink: 0 }}>{p.emoji}</span>
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ fontSize: 12, fontWeight: 700, color: isCurrentPersona ? p.color : 'rgba(255,255,255,0.9)' }}>{p.name}</div>
                    <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.4)', marginTop: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{p.description}</div>
                  </div>
                  {isCurrentPersona && <span style={{ fontSize: 10, color: p.color, fontWeight: 700, flexShrink: 0 }}>✓ 현재</span>}
                </button>
              )
            })}
          </div>
          <button
            onClick={() => setShowPersonaPopup(false)}
            style={{
              marginTop: 14, width: '100%', padding: '8px 0',
              background: 'rgba(255,255,255,0.06)', border: '1px solid rgba(255,255,255,0.12)',
              borderRadius: 10, color: 'rgba(255,255,255,0.5)', fontSize: 11, cursor: 'pointer',
            }}
          >닫기</button>
        </div>
      </>
    )}

    {/* ── 새 패널들 ── */}
    <AnimatePresence>
      {showDesktopAgent && (
        <DesktopAgent
          key="desktop-agent"
          onClose={() => setShowDesktopAgent(false)}
          primaryColor={primaryColor}
        />
      )}
    </AnimatePresence>
    <AnimatePresence>
      {showWorkflowBuilder && (
        <WorkflowBuilder
          key="workflow-builder"
          onClose={() => storeSetShowWorkflowBuilder(false)}
          primaryColor={primaryColor}
          initialName={workflowBuilderInitialName}
        />
      )}
    </AnimatePresence>
    <AnimatePresence>
      {showEmailSetup && (
        <EmailSetup
          key="email-setup"
          onClose={() => setShowEmailSetup(false)}
          primaryColor={primaryColor}
        />
      )}
    </AnimatePresence>


    {/* ── Proactive 알림 토스트 ── */}
    <div style={{
      position: 'fixed', top: 20, right: 20, zIndex: 99999,
      display: 'flex', flexDirection: 'column', gap: 10, pointerEvents: 'none',
    }}>
      <AnimatePresence>
        {toastAlerts.map(toast => (
          <motion.div
            key={toast.id}
            initial={{ opacity: 0, x: 60, scale: 0.9 }}
            animate={{ opacity: 1, x: 0, scale: 1 }}
            exit={{ opacity: 0, x: 60, scale: 0.9 }}
            style={{
              background: toast.level === 'critical' ? 'rgba(239,68,68,0.95)' :
                          toast.level === 'warn'     ? 'rgba(245,158,11,0.95)' :
                                                       'rgba(30,30,50,0.95)',
              backdropFilter: 'blur(12px)',
              border: `1px solid ${toast.level === 'critical' ? 'rgba(239,68,68,0.5)' :
                                    toast.level === 'warn'    ? 'rgba(245,158,11,0.5)' :
                                                                `${primaryColor}44`}`,
              borderRadius: 14, padding: '12px 16px', maxWidth: 320,
              boxShadow: '0 8px 32px rgba(0,0,0,0.5)', pointerEvents: 'auto',
            }}
          >
            <div style={{ fontWeight: 700, fontSize: 12, color: '#fff', marginBottom: 4 }}>
              {toast.title}
            </div>
            <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.8)', lineHeight: 1.5 }}>
              {toast.message.slice(0, 120)}{toast.message.length > 120 ? '...' : ''}
            </div>
          </motion.div>
        ))}
      </AnimatePresence>
    </div>

    {/* 구독 만료 배너 */}
    {isOnboarded && isLoggedIn && subscriptionStatus === 'expired' && (
      <div style={{
        position: 'fixed', bottom: 24, left: '50%', transform: 'translateX(-50%)',
        background: 'rgba(248,113,113,0.15)', backdropFilter: 'blur(12px)',
        border: '1px solid rgba(248,113,113,0.4)', borderRadius: 14,
        padding: '10px 20px', zIndex: 99998,
        display: 'flex', alignItems: 'center', gap: 14,
        boxShadow: '0 8px 32px rgba(0,0,0,0.5)',
      }}>
        <span style={{ fontSize: 13, color: '#fca5a5', fontWeight: 600 }}>
          {userLang === 'en'
            ? '⚠️ Trial ended. AI features are limited.'
            : '⚠️ 무료 체험이 종료되었습니다. AI 기능이 제한됩니다.'}
        </span>
        <button
          onClick={() => {
            import('../../lib/paddle').then(m => m.openCheckout(userEmail)).catch(() => {})
          }}
          style={{
            padding: '6px 14px', borderRadius: 8, border: 'none', cursor: 'pointer',
            background: '#f87171', color: 'white', fontSize: 12, fontWeight: 700,
          }}
        >
          {userLang === 'en' ? 'Subscribe' : '구독하기'}
        </button>
      </div>
    )}
    {/* ── Paywall Modal ── */}
    {paywallFeature && (
      <PaywallModal
        feature={paywallFeature}
        used={paywallUsed}
        limit={paywallLimit}
        onClose={() => setPaywallFeature(null)}
      />
    )}
    </>
  )
}
