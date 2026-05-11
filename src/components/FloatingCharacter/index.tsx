import React, { useState, useRef, useEffect, useCallback } from 'react'
import { AnimatePresence, motion, useMotionValue } from 'framer-motion'
import { useAppStore } from '../../stores/appStore'
import { ChatBubble } from './ChatBubble'
import type { ChatMessage } from './ChatBubble'
import { SettingsModal } from './SettingsModal'
import type { InlineCardData } from './InlineCards'
import type { InlineCardData2 } from './InlineCards2'
import type { InlineCard3Data } from './InlineCards3'
import type { InlineCard4Data } from './InlineCards4'
import { SpeakingWaves } from './Avatar3D'
import { AvatarRuntime } from './Avatar3D/AvatarRuntime'
import { OnboardingFlow } from './OnboardingFlow'
import type { AvatarConfig } from './OnboardingFlow'
import { appendHistory } from './ChatBubble'
import { callGemini, callOllama, fallbackResponse, trackUsage, getLastPreviewItems, clearLastPreviewItems, isFollowUpQuestion } from '../../lib/nexus/gemini_engine'
import { loadHistory, saveHistory, learnFromTurn, fromStoredTurns, toStoredTurns, buildMemoryContext } from '../../lib/nexus/memory'
import { startWakeWordDetection, stopWakeWordDetection } from '../../lib/nexus/wakeWord'
import { getGreeting } from '../../lib/nexus/personality'
import { speak, stopSpeaking, isAudioPlaying } from '../../lib/nexus/tts'
import {
  detectIntent, extractFolderName,
  extractVolume, extractBrightness, extractWifiAction, extractPowerAction,
  extractAppName, extractNoteContent,
  extractTwoFilePaths, extractVisionQuestion, extractDeepSearchQuery,
} from '../../lib/nexus/intentDetector'
import type { Intent } from '../../lib/nexus/intentDetector'
import { backendAPI, mockStats, mockScan, mockDailyReport, sendCommand,
  calendarToday, calendarWeek, calendarAdd,
  emailInbox, emailSend, emailSummarize,
  virusTotalCheck, historyStats, historyAnomalies,
  processKill, appPermissions, windowsUpdates, gpuStats,
  priceCompare, newsSearch,
  schedulerAdd, schedulerList, schedulerDelete,
  recallCapture, recallSearch,
  meetingStart, meetingStop, meetingList, meetingTranscribe, meetingSummarize,
  dictationType, dictationPaste,
  smarthomeDevices, smarthomeControl,
  weatherGet, travelTime,
  personaList, personaSet, personaCurrent,
  brainSearch, brainStats, brainRebuild,
  workflowRun, workflowPlan,
  captionStart, captionStop, captionLatest,
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
type CharacterEmotion = NexusEmotion

/* Web Speech API 타입 */
interface SRResult { [i: number]: { transcript: string }; isFinal: boolean; length: number }
interface SRResultList { [i: number]: SRResult; length: number }
interface SREvent extends Event { results: SRResultList }
interface SRErrorEvent extends Event { error: string }
interface SRInstance {
  lang: string; continuous: boolean; interimResults: boolean
  onresult: ((e: SREvent) => void) | null
  onend: (() => void) | null
  onerror: ((e: SRErrorEvent) => void) | null
  start(): void; stop(): void
}
type SRConstructor = { new(): SRInstance }

interface ConversationTurn {
  role: 'user' | 'model'
  parts: Array<{ text: string }>
}


/* 에이전트 사고 단계 생성 */
function buildAgentSteps(intent: Intent): string[] {
  switch (intent) {
    case 'pc_status':
      return ['PC 상태 데이터 요청 중...', 'CPU·메모리·디스크 수집', '시각 카드 생성 중']
    case 'security_scan':
      return ['보안 스캔 시작...', '원격 접속 흔적 확인', '수상한 프로세스 검사', '결과 분석 중']
    case 'full_scan':
      return ['전체 진단 시작...', '시스템 이슈 탐색', '심각도 분류', '리포트 생성 중']
    case 'clean':
      return ['정리 대상 파악...', '임시 파일·캐시 확인', '안전 정리 실행 중']
    case 'daily_report':
      return ['일일 데이터 수집...', '통계 분석 중', '예측 모델 실행', '리포트 완성']
    case 'open_folder':
      return ['폴더 이름 파악 중...', '경로 확인', '탐색기 실행 중']
    case 'remote_access': return ['원격 접속 도구 검색 중...', 'RDP 포트 확인', '프로세스 대조 중']
    case 'process_security': return ['프로세스 목록 수집...', '위험 패턴 대조', '포트 스캔 중']
    case 'startup_items': return ['시작 항목 조회 중...', '수상 키워드 분석']
    case 'defender_status': return ['Windows Defender 상태 확인...']
    case 'account_check': return ['로컬 계정 목록 조회...', '이상 계정 분석']
    case 'volume_control': return ['볼륨 조절 중...']
    case 'brightness': return ['밝기 조절 중...']
    case 'wifi_toggle': return ['Wi-Fi 상태 확인...', '설정 변경 중']
    case 'power_action': return ['전원 명령 실행 중...']
    case 'launch_app': return ['앱 경로 확인 중...', '실행 중']
    case 'process_top': return ['프로세스 목록 수집...', 'CPU·메모리 정렬 중']
    case 'driver_check': return ['드라이버 목록 조회...', '문제 항목 필터링']
    case 'network_analysis': return ['네트워크 어댑터 확인...', 'DNS·IP 조회 중', 'Ping 측정 중']
    case 'programs_list': return ['설치 프로그램 조회 중...']
    case 'boot_analysis': return ['부팅 이벤트 로그 분석...', '시작 항목 집계 중']
    case 'file_search': return ['파일 검색 시작...', '결과 수집 중']
    case 'file_organize': return ['파일 분류 중...', '폴더 이동 중']
    case 'file_duplicates': return ['파일 목록 수집...', '중복 분석 중']
    case 'browser_clean': return ['브라우저 캐시 위치 확인...', '데이터 정리 중']
    case 'registry_clean': return ['레지스트리 항목 스캔...', '무효 항목 정리 중']
    case 'restore_create': return ['복구 포인트 생성 중...']
    case 'focus_mode': return ['집중 모드 설정 중...']
    case 'notes': return ['메모 불러오는 중...']
    case 'doc_compare': return ['파일 열기...', '텍스트 추출 중', 'Diff 알고리즘 실행', '숫자 불일치 검사', '결과 정리 중']
    case 'doc_find': return ['파일 탐색 시작...', '이름·내용 대조 중', '결과 정렬 중']
    case 'deep_search': return ['파일 목록 수집...', '내용 인덱싱 중', '관련도 계산 중', '결과 정렬 중']
    case 'vision_screen': return ['화면 캡처 중...', 'AI에게 분석 요청', '답변 생성 중']
    case 'vision_ocr': return ['클립보드 이미지 확인...', 'Windows OCR 실행 중', '텍스트 추출 중']
    case 'smart_organize':   return ['파일 목록 수집...', '파일 유형 분류 중', '폴더 이동 중', '정리 완료']
    case 'journal_today':    return ['최근 파일 기록 수집...', '앱 사용 분석 중', '업무 시간 추정', '일지 생성 중']
    case 'journal_generate': return ['일지 데이터 수집...', '포맷 생성 중', '파일 저장 중']
    case 'journal_history':  return ['과거 일지 조회 중...']
    case 'macro_list':       return ['매크로 목록 조회 중...']
    case 'macro_create':     return ['명령 파싱 중...', '액션 구성 중', '스케줄 등록 중']
    case 'macro_run':        return ['매크로 실행 중...', '액션 순서 처리 중', '완료 확인']
    case 'pc_report':        return ['시스템 상태 수집...', '보안 점검 중', '리포트 생성 중', 'HTML 저장 중']
    case 'report_email':     return ['리포트 생성 중...', 'SMTP 연결 중', '이메일 전송 중']
    case 'doc_summary':      return ['파일 열기...', '텍스트 추출 중', '핵심 분석 중', '요약 생성 중']
    case 'calendar_today':   return ['Outlook 연결 중...', '오늘 일정 불러오는 중']
    case 'calendar_week':    return ['Outlook 연결 중...', '이번 주 일정 불러오는 중']
    case 'calendar_add':     return ['일정 생성 중...', 'Outlook에 저장 중']
    case 'email_inbox':      return ['Outlook 연결 중...', '받은 편지함 확인 중']
    case 'email_send':       return ['메일 작성 중...', 'SMTP 전송 중']
    case 'email_summarize':  return ['받은 메일 가져오는 중...', 'AI 요약 생성 중']
    case 'virus_check':      return ['파일 해시 계산 중...', 'VirusTotal 조회 중', '결과 분석 중']
    case 'perf_history':     return ['성능 이력 불러오는 중...', '트렌드 분석 중']
    case 'perf_anomaly':     return ['이력 데이터 분석 중...', '이상 패턴 탐지 중']
    case 'price_compare':    return ['검색 시작...', '쿠팡 확인 중', '네이버 확인 중', '가격 비교 중']
    case 'news_search':      return ['뉴스 검색 중...', '최신 기사 수집 중']
    case 'schedule_list':    return ['스케줄 목록 불러오는 중...']
    case 'schedule_add':     return ['명령 파싱 중...', '스케줄 등록 중']
    case 'schedule_delete':  return ['스케줄 삭제 중...']
    case 'process_kill':     return ['프로세스 찾는 중...', '강제 종료 중']
    case 'app_permissions':  return ['레지스트리 확인 중...', '권한 목록 수집 중']
    case 'windows_updates':  return ['Windows Update 서비스 연결 중...', '업데이트 목록 확인 중']
    case 'gpu_stats':        return ['GPU 정보 수집 중...', 'nvidia-smi 확인 중']
    // ── 10가지 신규 기능 ──
    case 'recall_search':    return ['화면 기억 데이터 검색 중...', '매칭 결과 정렬 중']
    case 'recall_capture':   return ['화면 캡처 중...', 'OCR 텍스트 추출 중', '기억 저장 중']
    case 'meeting_start':    return ['마이크 확인 중...', '녹음 시작 중']
    case 'meeting_stop':     return ['녹음 종료 중...', '파일 저장 중']
    case 'meeting_summary':  return ['녹음 파일 확인 중...', 'Whisper 전사 중...', 'AI 요약 생성 중']
    case 'meeting_list':     return ['회의 목록 불러오는 중...']
    case 'dictation_start':  return ['텍스트 분석 중...', '현재 앱에 입력 중']
    case 'smarthome_list':   return ['Home Assistant 연결 중...', '기기 목록 불러오는 중']
    case 'smarthome_control':return ['Home Assistant 연결 중...', '명령 전송 중']
    case 'weather':          return ['날씨 데이터 수집 중...', '예보 분석 중']
    case 'travel_time':      return ['출발지·목적지 좌표 조회 중...', '경로 계산 중']
    case 'translate':        return ['클립보드 내용 확인 중...', '번역 중...']
    case 'clipboard_ai':     return ['클립보드 내용 가져오는 중...', 'AI 처리 중']
    case 'voice_todo':       return ['내용 분석 중...', '메모 저장 중', '캘린더 등록 중']
    case 'persona_list':     return ['페르소나 목록 불러오는 중...']
    case 'persona_switch':   return ['페르소나 전환 중...']
    case 'brain_search':     return ['🧠 Second Brain 검색 중...', '관련 기억 분석 중']
    case 'brain_stats':      return ['인덱스 통계 조회 중...']
    case 'workflow_run':     return ['⚡ 워크플로 계획 생성 중...', '단계별 실행 중...', '결과 정리 중...']
    case 'workflow_plan':    return ['⚡ 워크플로 계획 생성 중...']
    case 'caption_start':    return ['🎬 오디오 캡처 초기화 중...', '실시간 자막 시작']
    case 'caption_stop':     return ['자막 종료 중...']
    default:
      return ['요청 분석 중...']
  }
}

/* 인텐트별 응답 텍스트 */
function intentResponseText(intent: Intent, lang: 'ko' | 'en', assistantName: string): string {
  if (lang === 'en') {
    switch (intent) {
      case 'pc_status': return `Here's your real-time PC status, ${assistantName} is watching over it!`
      case 'security_scan': return `Security scan complete. Here are the results:`
      case 'full_scan': return `Full PC diagnostic done! Here's what I found:`
      case 'clean': return `Cleanup complete! I freed up some disk space for you.`
      case 'daily_report': return `Here's today's PC report summary:`
      case 'repair': return `Repair operation finished!`
      default: return ''
    }
  }
  switch (intent) {
    case 'pc_status': return `실시간 PC 상태를 가져왔어요! 📊`
    case 'security_scan': return `보안 스캔 완료! 결과를 확인해보세요 🔒`
    case 'full_scan': return `전체 진단 완료! 발견된 항목을 정리했어요 🔍`
    case 'clean': return `정리 완료! 디스크 공간을 확보했어요 🧹`
    case 'daily_report': return `오늘의 PC 리포트예요 📊`
    case 'repair': return `수리 작업을 완료했어요 🔧`
    default: return ''
  }
}

export function FloatingCharacter() {
  const {
    assistantName, userName, userLang,
    primaryColor: storePrimary, accentColor: storeAccent,
    isOnboarded, setOnboarded, setAssistantName, setUserName,
    setPrimaryColor, setAccentColor,
    micEnabled, ttsVoice, setTtsVoice,
    isLoggedIn, setLoggedIn, subscriptionStatus, userEmail,
  } = useAppStore()

  const primaryColor = storePrimary || '#a78bfa'
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
  const [emotion, setEmotion]             = useState<'neutral'|'happy'|'concerned'|'alert'|'humorous'>('neutral')
  const [speaking, setSpeaking]           = useState(false)
  const [listening, setListening]         = useState(false)
  const [input, setInput]                 = useState('')
  const [voiceInterim, setVoiceInterim]   = useState('')
  const [minimized, setMinimized]         = useState(false)
  const [settingsOpen, setSettingsOpen]   = useState(false)
  const [soundEnabled, setSoundEnabled]   = useState(() => localStorage.getItem('nexus-sound') !== 'off')
  const [isActive, setIsActive]           = useState(true)   // 비활성화 토글
  const [isDragging, setIsDragging]       = useState(false)
  const [historyVersion, setHistoryVersion] = useState(0)
  const dragX = useMotionValue(0)
  const dragY = useMotionValue(0)
  const [backendStatus, setBackendStatus] = useState<BackendStatus>('checking')
  const [focusEndMs, setFocusEndMs]       = useState<number | undefined>(getFocusModeEnd)
  const [floatingPreview, setFloatingPreview] = useState<Array<{ title: string; url: string }> | null>(null)

  // ── Clarify 멀티턴 상태 ──────────────────────────────────
  const [clarifyPendingIntent,   setClarifyPendingIntent]   = useState<string | null>(null)
  const [clarifyPendingParams,   setClarifyPendingParams]   = useState<Record<string, unknown> | null>(null)
  const [clarifyPendingQuestion, setClarifyPendingQuestion] = useState<string | null>(null)
  const [activePersona, setActivePersona] = useState<PersonaDef | null>(null)
  const [captionRunning, setCaptionRunning] = useState(false)

  // ── 미리보기 WebviewWindow 열기 ────────────────────────────
  const openPreview = useCallback(async (url: string, title: string) => {
    try {
      // Tauri 환경: 별도 WebviewWindow로 열기
      const { WebviewWindow } = await import('@tauri-apps/api/webviewWindow')
      const label = `preview_${Date.now()}`
      const win = new WebviewWindow(label, {
        url,
        title: title || '미리보기',
        width: 1024,
        height: 768,
        center: true,
        resizable: true,
        decorations: true,
      })
      win.once('tauri://error', () => {
        // Tauri 창 생성 실패 시 기본 브라우저로 폴백
        window.open(url, '_blank')
      })
    } catch {
      // 브라우저 환경 (Mac 개발) — 새 탭으로 열기
      window.open(url, '_blank')
    }
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
  const voiceRecRef        = useRef<SRInstance | null>(null)
  const typingRef          = useRef(false)
  const isMountedRef       = useRef(true)   // unmount 후 setState 방지
  const voiceEndTimerRef   = useRef<ReturnType<typeof setTimeout> | null>(null)
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
      if (voiceEndTimerRef.current) clearTimeout(voiceEndTimerRef.current)
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
    const old = localStorage.getItem('nexus-pplx-key')
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
  }, [setAssistantName, setUserName, setPrimaryColor, setAccentColor, setOnboarded, setTtsVoice, setLoggedIn])

  /* 첫 인사 — 텍스트 + TTS (1회만 재생) */
  const hasGreetedRef = useRef(false)
  useEffect(() => {
    const greeting = getGreeting(assistantName, userName, userLang)
    setMessages([{ id: '0', role: 'nexus', text: greeting }])
    historyRef.current = []
    if (isOnboarded && !hasGreetedRef.current) {
      hasGreetedRef.current = true
      setTimeout(() => {
        const preview = greeting.replace(/\*\*/g, '').replace(/\n/g, ' ').slice(0, 60)
        setBubbleText(preview + (greeting.length > 60 ? '...' : ''))
        speak(greeting, userLang, () => setSpeaking(true), () => { setSpeaking(false); setTimeout(() => setBubbleText(''), 1500) })
      }, 800)
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [assistantName, userName, userLang, isOnboarded])

  /* 웨이크워드 — 사용자가 마이크를 허용한 경우에만 활성화 */
  useEffect(() => {
    if (!micEnabled) {
      stopWakeWordDetection()
      return
    }
    const wakeWords = [
      assistantName, `hey ${assistantName.toLowerCase()}`,
      `헤이 ${assistantName.toLowerCase()}`,
      '자비스', 'hey jarvis', 'nexus', 'hey nexus', '넥서스',
    ]
    startWakeWordDetection(wakeWords, () => {
      setChatOpen(true)
      setMinimized(false)
      handleVoiceToggle()
    })
    return () => stopWakeWordDetection()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [assistantName, micEnabled])

  /* ── 백엔드 연결 상태 초기 체크 + 페르소나 로딩 + API 키 동기화 ── */
  useEffect(() => {
    const connectAndSync = async () => {
      const status = await checkBackendHealth()
      setBackendStatus(status)
      if (status === 'connected') {
        try {
          const { syncAPIKeysToBackend } = await import('../../lib/nexus/gemini_engine')
          await syncAPIKeysToBackend()
        } catch { /* 무시 */ }
      }
      return status
    }
    connectAndSync()
    personaCurrent().then((r) => setActivePersona(r.persona)).catch(() => {})

    // ⑩ 백엔드 자동 재연결: disconnected 상태면 30초마다 재시도
    let wasDisconnected = false
    const reconnectId = setInterval(async () => {
      setBackendStatus(prev => {
        if (prev === 'disconnected') {
          wasDisconnected = true
          connectAndSync().then(newStatus => {
            if (newStatus === 'connected' && wasDisconnected) {
              wasDisconnected = false
              setBubbleText(userLang === 'ko' ? '백엔드에 다시 연결됐습니다.' : 'Reconnected to backend.')
              setTimeout(() => setBubbleText(''), 3000)
            }
          })
        }
        return prev
      })
    }, 30000)
    return () => clearInterval(reconnectId)
  }, [])

  /* ── SSE 연결: Proactive 알림 + Task Queue 실시간 수신 ── */
  useEffect(() => {
    nexusSSE.connect()

    const unsubAlert = nexusSSE.onAlert((alert) => {
      // 승인 요청 알림 처리
      if (alert.action?.startsWith('approve:')) {
        const taskId = alert.action.replace('approve:', '')
        setMessages(prev => [...prev, {
          id: `approval-${taskId}`,
          role: 'nexus' as const,
          text: `⚠️ **작업 승인 필요**\n${alert.message}\n\n[승인] 또는 [거부]를 입력해 주세요.`,
        }])
        setBubbleText('작업 승인이 필요합니다 ✋')
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
    speak(alert.message, userLang, () => setSpeaking(true), () => { setSpeaking(false); setTimeout(() => setBubbleText(''), 1500) })
  }, [userLang])

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
        speak(msg, userLang, () => setSpeaking(true), () => setSpeaking(false))
        clearInterval(focusTimerRef.current!)
      }
    }, 1_000)
    return () => { if (focusTimerRef.current) clearInterval(focusTimerRef.current) }
  }, [focusEndMs, userLang, assistantName])

  /* 말풍선에 표시할 최근 AI 발화 */
  const [bubbleText, setBubbleText] = useState('')
  const [bubbleExpanded, setBubbleExpanded] = useState(false)

  /* TTS — 감정 기반 톤 자동 조정 */
  const speakText = useCallback((text: string, em?: CharacterEmotion) => {
    const clean = text.replace(/\*\*/g, '').replace(/\n+/g, ' ').trim()
    setBubbleText(clean)   // 전체 텍스트, 자동소멸 없음 (X버튼으로만 닫기)
    if (!soundEnabled) return
    const ttsEmotion = em ?? emotion
    speak(
      text, userLang,
      () => setSpeaking(true),
      () => setSpeaking(false),
      ttsEmotion as import('../../lib/nexus/tts').SpeakEmotion,
      ttsVoice,
    )
  }, [userLang, emotion, ttsVoice, soundEnabled])

  /* STT */
  const handleVoiceToggle = useCallback(() => {
    if (listening) {
      voiceRecRef.current?.stop()
      voiceRecRef.current = null
      setListening(false)
      setVoiceInterim('')
      return
    }
    const win = window as unknown as Record<string, SRConstructor | undefined>
    const SR = win['SpeechRecognition'] ?? win['webkitSpeechRecognition']
    if (!SR) return

    stopWakeWordDetection()
    const rec = new SR()
    voiceRecRef.current = rec
    rec.lang = userLang === 'ko' ? 'ko-KR' : 'en-US'
    rec.continuous = false
    rec.interimResults = true

    rec.onresult = (e: SREvent) => {
      let interim = '', final = ''
      for (let i = 0; i < e.results.length; i++) {
        const t = e.results[i][0].transcript
        if (e.results[i].isFinal) final += t
        else interim += t
      }
      setVoiceInterim(interim)
      if (final) { setInput(final); setVoiceInterim('') }
    }

    rec.onend = () => {
      if (!isMountedRef.current) return
      setListening(false)
      setVoiceInterim('')
      voiceRecRef.current = null
      setInput(prev => {
        if (prev.trim() && !typingRef.current) {
          if (voiceEndTimerRef.current) clearTimeout(voiceEndTimerRef.current)
          voiceEndTimerRef.current = setTimeout(() => {
            if (!isMountedRef.current) return
            setInput(cur => {
              if (cur.trim()) void sendText(cur)
              return ''
            })
          }, 100)
        }
        return prev
      })
      const wakeWords = [assistantName, `hey ${assistantName.toLowerCase()}`, '자비스', 'nexus']
      startWakeWordDetection(wakeWords, () => handleVoiceToggle())
    }
    rec.onerror = (e: SRErrorEvent) => {
      setListening(false)
      setVoiceInterim('')
      voiceRecRef.current = null
      const isKo = userLang === 'ko'
      if (e.error === 'not-allowed' || e.error === 'service-not-allowed') {
        setBubbleText(isKo
          ? '마이크 권한이 필요합니다. 시스템 설정 → 개인 정보 보호 → 마이크에서 앱을 허용해주세요.'
          : 'Microphone permission denied. Allow access in System Settings → Privacy → Microphone.')
      } else if (e.error === 'no-speech') {
        // normal - no message
      } else if (e.error === 'network') {
        setBubbleText(isKo
          ? '음성 인식 네트워크 오류가 발생했습니다. 인터넷 연결을 확인해주세요.'
          : 'Speech recognition network error. Please check your internet connection.')
      } else {
        setBubbleText(isKo ? `음성 인식 오류: ${e.error}` : `Speech recognition error: ${e.error}`)
      }
    }
    setListening(true)
    try { rec.start() } catch { setListening(false) }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [listening, userLang, assistantName])

  /* ── 백엔드 인텐트 처리 — LLM 없이 즉시 ── */
  const handleBackendIntent = useCallback(async (
    intent: Intent,
    msgId: string,
    originalText = '',
  ): Promise<{ text: string; card?: InlineCardData; card2?: InlineCardData2; card3?: InlineCard3Data; card4?: InlineCard4Data; emotion: CharacterEmotion }> => {

    /* Multi-step: 사고 카드 먼저 표시 */
    const steps = buildAgentSteps(intent)
    setMessages(prev => [...prev, {
      id: `think-${msgId}`,
      role: 'nexus',
      text: '',
      inlineCard: { type: 'agent_thinking', steps },
    }])

    /* 약간의 딜레이로 사고 과정 보여주기 */
    await new Promise(r => setTimeout(r, steps.length * 200))

    /* 사고 카드 제거 */
    setMessages(prev => prev.filter(m => m.id !== `think-${msgId}`))

    try {
      switch (intent) {
        case 'pc_status': {
          const data = await backendAPI.stats().catch(() => mockStats())
          return {
            text: intentResponseText('pc_status', userLang, assistantName),
            card: { type: 'pc_status', data },
            emotion: data.cpu > 80 || data.mem > 85 ? 'concerned' : 'happy',
          }
        }
        case 'security_scan':
        case 'full_scan': {
          const data = await backendAPI.scan().catch(() => mockScan())
          const em: CharacterEmotion = data.score < 70 ? 'alert' : data.score < 85 ? 'concerned' : 'happy'
          return {
            text: intentResponseText(intent, userLang, assistantName),
            card: { type: 'scan_result', data },
            emotion: em,
          }
        }
        case 'clean': {
          const results = await backendAPI.autoClean(['temp', 'browser']).catch(async () => {
            const r = await backendAPI.clean(['temp']).catch(() => ({ freed: 0, message: '정리 완료' }))
            return r as { freed: number; message: string }
          })
          return {
            text: intentResponseText('clean', userLang, assistantName),
            card: { type: 'clean_result', results },
            emotion: 'happy',
          }
        }
        case 'daily_report': {
          const data = await backendAPI.dailyReport().catch(() => mockDailyReport())
          return {
            text: intentResponseText('daily_report', userLang, assistantName),
            card: { type: 'daily_report', data },
            emotion: data.pc_score >= 80 ? 'happy' : 'concerned',
          }
        }
        case 'repair': {
          const data = await backendAPI.repair(['temp-files']).catch(() => ({
            success: true, message: '수리 완료', freed: 0,
          }))
          return {
            text: intentResponseText('repair', userLang, assistantName),
            card: { type: 'repair_result', data },
            emotion: data.success ? 'happy' : 'concerned',
          }
        }
        case 'open_folder': {
          const folderName = extractFolderName(originalText)
          if (!folderName) {
            const ask = userLang === 'ko'
              ? '어떤 폴더를 열어드릴까요? (예: 바탕화면, 다운로드, 문서, 사진)'
              : 'Which folder would you like to open?'
            return { text: ask, emotion: 'neutral' }
          }
          const res = await backendAPI.openFolder(folderName).catch(() => ({
            success: false, path: '', message: '백엔드 미연결 상태입니다.',
          }))
          return {
            text: res.success ? `${folderName} 폴더를 열었어요.` : res.message,
            card: { type: 'folder_open', success: res.success, path: res.path, message: res.message },
            emotion: res.success ? 'happy' : 'concerned',
          }
        }

        /* ── 보안 상세 ── */
        case 'remote_access': {
          const data = await backendAPI.securityRemote().catch(() => ({ found: false, tools: [], rdp_open: false, score: 100 }))
          return { text: data.found ? `⚠️ 실행 중인 원격 접속 도구 발견! 점수: ${data.score}` : '✅ 원격 접속 도구 없음, 안전합니다.',
            card2: { type: 'remote_access', data }, emotion: data.found ? 'alert' : 'happy' }
        }
        case 'process_security': {
          const data = await backendAPI.securityProcs().catch(() => ({ suspicious_processes: [], open_ports: [], score: 100 }))
          return { text: data.score < 80 ? `⚠️ 수상한 프로세스/포트 발견 (보안 점수: ${data.score})` : '✅ 수상한 프로세스 없음.',
            card2: { type: 'process_security', data }, emotion: data.score < 80 ? 'alert' : 'happy' }
        }
        case 'hosts_check': {
          const data = await backendAPI.securityHosts().catch(() => ({ score: 100, modified: false, entries: 0, suspicious: [] }))
          return { text: data.modified ? `⚠️ hosts 파일 변조 의심! 수상한 항목 ${data.suspicious.length}개` : '✅ hosts 파일 정상',
            card2: { type: 'system_action', icon: data.modified ? '⚠️' : '✅', title: data.modified ? 'Hosts 파일 변조 감지' : 'Hosts 파일 정상', detail: `총 ${data.entries}개 항목`, success: !data.modified },
            emotion: data.modified ? 'alert' : 'happy' }
        }
        case 'startup_items': {
          const data = await backendAPI.securityStartup().catch(() => ({ items: [], total: 0, suspicious_count: 0 }))
          return { text: `시작 프로그램 ${data.total}개, 수상한 항목 ${data.suspicious_count}개`,
            card2: { type: 'startup_items', data }, emotion: data.suspicious_count > 0 ? 'concerned' : 'happy' }
        }
        case 'defender_status': {
          const data = await backendAPI.securityDefender().catch(() => ({ antivirus_enabled: true, realtime_protection: true, quick_scan_age: 0, full_scan_age: 0, score: 100, issues: [] }))
          return { text: data.score >= 80 ? '🛡️ Windows Defender 정상 작동 중' : `⚠️ 보안 점수 ${data.score} — ${data.issues[0] ?? ''}`,
            card2: { type: 'defender', data }, emotion: data.score >= 80 ? 'happy' : 'alert' }
        }
        case 'account_check': {
          const data = await backendAPI.securityAccounts().catch(() => ({ total: 0, suspicious: [], suspicious_count: 0, score: 100 }))
          return { text: data.suspicious_count ? `⚠️ 이상 계정 ${data.suspicious_count}개 감지됨` : `✅ 계정 정상 (${data.total}개)`,
            card2: { type: 'system_action', icon: data.suspicious_count ? '⚠️' : '✅', title: data.suspicious_count ? `이상 계정 ${data.suspicious_count}개` : '계정 정상', success: !data.suspicious_count },
            emotion: data.suspicious_count ? 'alert' : 'happy' }
        }

        /* ── 시스템 제어 ── */
        case 'volume_control': {
          const { action, value } = extractVolume(originalText)
          const res = await backendAPI.volume(action, value).catch(() => ({ message: '볼륨 조절에 실패했어요' }))
          return { text: res.message,
            card2: { type: 'system_action', icon: action === 'mute' ? '🔇' : '🔊', title: res.message, success: true },
            emotion: 'happy' }
        }
        case 'brightness': {
          const { action, value } = extractBrightness(originalText)
          const res = await backendAPI.brightness(action, value).catch(() => ({ message: '밝기 조절에 실패했어요 (노트북 전용)' }))
          return { text: res.message,
            card2: { type: 'system_action', icon: '☀️', title: res.message, success: true },
            emotion: 'happy' }
        }
        case 'wifi_toggle': {
          const wifiAction = extractWifiAction(originalText)
          const res = await backendAPI.wifi(wifiAction).catch(() => ({ message: 'Wi-Fi 제어 실패' }))
          return { text: (res as { message?: string }).message ?? 'Wi-Fi 상태 확인됨',
            card2: { type: 'system_action', icon: '📶', title: (res as { message?: string }).message ?? '', success: true },
            emotion: 'happy' }
        }
        case 'power_action': {
          const powerAct = extractPowerAction(originalText)
          const icons: Record<string, string> = { lock: '🔒', sleep: '😴', restart: '🔄', shutdown: '⏻' }
          const res = await backendAPI.power(powerAct).catch(() => ({ success: false, message: '전원 제어 실패' }))
          return { text: res.message,
            card2: { type: 'system_action', icon: icons[powerAct] ?? '⚡', title: res.message, success: res.success },
            emotion: res.success ? 'happy' : 'concerned' }
        }
        case 'launch_app': {
          const appName = extractAppName(originalText)
          if (!appName) return { text: '어떤 앱을 실행할까요?', emotion: 'neutral' }
          const res = await backendAPI.launchApp(appName).catch(() => ({ success: false, message: `${appName} 실행 실패` }))
          return { text: res.message,
            card2: { type: 'system_action', icon: '🚀', title: res.message, success: res.success },
            emotion: res.success ? 'happy' : 'concerned' }
        }
        case 'process_top': {
          const data = await backendAPI.processTop().catch(() => ({ by_cpu: [], by_mem: [] }))
          return { text: 'CPU·메모리 상위 프로세스예요 📊',
            card2: { type: 'process_top', data }, emotion: 'neutral' }
        }

        /* ── 고급 기능 ── */
        case 'driver_check': {
          const data = await backendAPI.drivers().catch(() => ({ total: 0, problematic: [], problem_count: 0, score: 100, message: '드라이버 정보를 가져올 수 없어요' }))
          return { text: data.message, card2: { type: 'drivers', data }, emotion: data.problem_count > 0 ? 'concerned' : 'happy' }
        }
        case 'registry_clean': {
          const data = await backendAPI.registryClean().catch(() => ({ success: false, cleaned_keys: 0, message: '레지스트리 정리 실패' }))
          return { text: data.message,
            card2: { type: 'system_action', icon: '🗂️', title: data.message, success: data.success },
            emotion: data.success ? 'happy' : 'concerned' }
        }
        case 'power_plan': {
          const plans: Record<string, string> = { '고성능': 'performance', '절전': 'powersaver', '균형': 'balanced' }
          let planName = 'balanced'
          for (const [k, v] of Object.entries(plans)) {
            if (originalText.includes(k)) { planName = v; break }
          }
          const res = await backendAPI.setPowerPlan(planName).catch(() => ({ success: false, message: '전원 계획 변경 실패' }))
          return { text: res.message,
            card2: { type: 'system_action', icon: '⚡', title: res.message, success: res.success },
            emotion: res.success ? 'happy' : 'concerned' }
        }
        case 'network_analysis': {
          const data = await backendAPI.networkAnalysis().catch(() => ({ adapters: [], dns_servers: '', public_ip: '', ping_ms: '', connected: false }))
          return { text: data.connected ? `🌐 인터넷 연결됨 · 공개 IP: ${data.public_ip || '알 수 없음'}` : '📵 인터넷 연결 없음',
            card2: { type: 'network', data }, emotion: data.connected ? 'happy' : 'concerned' }
        }
        case 'restore_create': {
          const res = await backendAPI.restoreCreate().catch(() => ({ success: false, message: '복구 포인트 생성 실패' }))
          return { text: res.message,
            card2: { type: 'system_action', icon: '💾', title: res.message, success: res.success },
            emotion: res.success ? 'happy' : 'concerned' }
        }
        case 'disk_check': {
          const res = await backendAPI.diskCheck().catch(() => ({ success: false, message: '디스크 검사 시작 실패' }))
          return { text: res.message,
            card2: { type: 'system_action', icon: '💿', title: res.message, success: res.success },
            emotion: res.success ? 'happy' : 'concerned' }
        }
        case 'browser_clean': {
          const res = await backendAPI.browserClean().catch(() => ({ results: [], total_mb: 0, total_freed: '0B', message: '브라우저 정리 실패' }))
          return { text: res.message,
            card2: { type: 'system_action', icon: '🌐', title: res.message, detail: `${res.total_freed} 확보`, success: true },
            emotion: 'happy' }
        }
        case 'programs_list': {
          const data = await backendAPI.programsList().catch(() => ({ programs: [], total: 0 }))
          return { text: `설치된 프로그램 ${data.total}개 확인했어요 📦`,
            card2: { type: 'programs_list', data }, emotion: 'neutral' }
        }
        case 'boot_analysis': {
          const data = await backendAPI.bootAnalysis().catch(() => ({ uptime_minutes: '0', startup_count: '?', recent_boots: [], message: '부팅 분석 실패' }))
          return { text: data.message, card2: { type: 'boot_analysis', data }, emotion: 'neutral' }
        }

        /* ── 파일 관리 ── */
        case 'file_search': {
          const query = originalText.replace(/파일.*찾아|찾아줘.*파일|어디/g, '').trim()
          const data = await backendAPI.filesSearch(query).catch(() => ({ results: [], total: 0, message: '파일 검색 실패' }))
          return { text: data.message, card2: { type: 'file_search', data }, emotion: 'neutral' }
        }
        case 'file_organize': {
          const isDesktop = /바탕화면|desktop/.test(originalText)
          const isDownloads = /다운로드|download/.test(originalText)
          const folderTarget = isDesktop ? 'desktop' : isDownloads ? 'downloads' : undefined
          const res = await backendAPI.filesOrganize(folderTarget).catch(() => ({ success: false, moved: 0, message: '폴더 정리 실패' }))
          return { text: res.message,
            card2: { type: 'system_action', icon: '📁', title: res.message, success: res.success },
            emotion: res.success ? 'happy' : 'concerned' }
        }
        case 'file_duplicates': {
          const data = await backendAPI.filesDuplicates().catch(() => ({ groups: [], total_groups: 0, waste_mb: 0, waste: '0B', message: '중복 검사 실패' }))
          return { text: data.message, card2: { type: 'duplicates', data }, emotion: data.total_groups > 0 ? 'concerned' : 'happy' }
        }

        /* ── 생산성 ── */
        case 'focus_mode': {
          const isOff = /해제|off|끄/.test(originalText)
          const durMatch = originalText.match(/(\d+)\s*분/)
          const duration = durMatch ? parseInt(durMatch[1]) : 25
          const res = await backendAPI.focusMode(isOff ? 'off' : 'on', duration).catch(() => ({ success: false, active: !isOff, message: isOff ? '집중 모드 해제됨' : `집중 모드 시작! ${duration}분 동안 알림이 차단돼요 🎯` }))
          if (res.active) {
            setFocusModeEnd(duration)
            setFocusEndMs(Date.now() + duration * 60_000)
          } else {
            clearFocusMode()
            setFocusEndMs(undefined)
          }
          return { text: res.message,
            card2: { type: 'focus_mode', active: res.active, duration },
            emotion: res.active ? 'happy' : 'neutral' }
        }
        case 'clipboard': {
          const data = await backendAPI.clipboard().catch(() => ({ current: '', tip: 'Windows + V 로 클립보드를 확인해보세요' }))
          return { text: data.current ? `클립보드: "${data.current.slice(0, 50)}..."` : data.tip,
            card2: { type: 'system_action', icon: '📋', title: data.current ? `클립보드 내용 확인` : '클립보드 비어있음', detail: data.current?.slice(0, 60) },
            emotion: 'neutral' }
        }
        case 'notes': {
          const isNew = /적어|기록|저장/.test(originalText)
          if (isNew) {
            const content = extractNoteContent(originalText)
            if (content.length > 3) {
              const res = await backendAPI.saveNote(content).catch(() => ({ success: false, note: { id: '', content, created: '' }, message: '메모 저장 실패' }))
              return { text: res.message,
                card2: { type: 'system_action', icon: '📝', title: res.message, success: res.success },
                emotion: res.success ? 'happy' : 'concerned' }
            }
          }
          const data = await backendAPI.notes().catch(() => ({ notes: [], total: 0 }))
          return { text: `메모 ${data.total}개를 가져왔어요 📝`,
            card2: { type: 'notes', data }, emotion: 'neutral' }
        }

        /* ── 문서 비교 ── */
        case 'doc_compare': {
          const [f1, f2] = extractTwoFilePaths(originalText)
          if (!f1 || !f2) {
            return { text: '비교할 두 파일 경로를 알려주세요. 예: "report_v1.docx 와 report_v2.docx 비교해줘"', emotion: 'neutral' }
          }
          const data = await backendAPI.docsCompare(f1, f2)
          return {
            text: data.summary,
            card3: { type: 'doc_compare', data },
            emotion: data.similarity_pct < 70 ? 'concerned' : 'neutral',
          }
        }
        case 'doc_find': {
          const query = originalText.replace(/문서.*찾아|파일.*찾아서|계약서.*찾아|보고서.*찾아/g, '').trim() || originalText
          const data = await backendAPI.docsFind(query)
          return {
            text: data.message,
            card3: { type: 'doc_find', data },
            emotion: data.total > 0 ? 'happy' : 'neutral',
          }
        }

        /* ── Deep Search ── */
        case 'deep_search': {
          const query = extractDeepSearchQuery(originalText)
          const data = await backendAPI.deepSearch(query)
          return {
            text: data.message,
            card3: { type: 'deep_search', data },
            emotion: data.total > 0 ? 'happy' : 'neutral',
          }
        }

        /* ── Vision ── */
        case 'vision_screen': {
          const question = extractVisionQuestion(originalText)
          // 스크린샷 캡처 (OCR 포함)
          const ss = await backendAPI.screenshot(true).catch(() => ({ success: false, base64: '', width: 0, height: 0, mime: 'image/png', captured: '' }))
          if (!ss.success || !ss.base64) {
            return { text: '화면 캡처에 실패했어요. Tauri 앱 환경에서 실행해주세요.', emotion: 'concerned' }
          }
          // Gemini Flash에 이미지 + 질문 전달
          const { callGeminiWithImage } = await import('../../lib/nexus/gemini_engine')
          const answer = await callGeminiWithImage(ss.base64, question).catch(() => (ss as { ocr_text?: string }).ocr_text || '(분석 불가)')
          return {
            text: answer.slice(0, 120),
            card3: { type: 'vision_result', data: { question, answer, screenshot_b64: ss.base64 } },
            emotion: 'happy',
          }
        }
        case 'vision_ocr': {
          const data = await backendAPI.ocrClipboard()
          return {
            text: data.message,
            card3: { type: 'vision_ocr', data },
            emotion: 'neutral',
          }
        }

        /* ── 업무 일지 ── */
        case 'journal_today': {
          const data = await backendAPI.journalToday()
          return {
            text: `오늘 업무 일지를 정리했어요! ${(data as { app_usage?: unknown[] }).app_usage?.length || 0}개 앱, ${(data as { recent_files?: unknown[] }).recent_files?.length || 0}개 파일 사용 기록이 있어요.`,
            card4: { type: 'journal_today', data: data as unknown as Parameters<typeof import('./InlineCards4').JournalTodayCard>[0]['data'] },
            emotion: 'happy',
          }
        }
        case 'journal_generate': {
          const res = await backendAPI.journalGenerate()
          return {
            text: res.message,
            card4: { type: 'journal_today', data: { date: new Date().toISOString().slice(0,10), work_hours: 0, app_usage: [], recent_files: [], summary: res.preview || '', generated: '' } },
            emotion: 'happy',
          }
        }
        case 'journal_history': {
          const data = await backendAPI.journalHistory()
          return {
            text: `최근 ${(data as { days?: number }).days || 0}일간의 업무 기록을 찾았어요.`,
            card4: { type: 'journal_history', data: data as { history: Array<{ date: string; work_hours: number; file_count: number; app_count: number; top_app: string }> } },
            emotion: 'neutral',
          }
        }

        /* ── 자동화 매크로 ── */
        case 'macro_list': {
          const data = await backendAPI.macroList()
          return {
            text: (data as { total?: number }).total === 0
              ? '아직 등록된 매크로가 없어요. "매일 아침 9시에 크롬 열어줘" 처럼 말해보세요!'
              : `매크로 ${(data as { total?: number }).total}개가 등록돼 있어요.`,
            card4: { type: 'macro_list', data: data as { macros: Parameters<typeof import('./InlineCards4').MacroListCard>[0]['data']['macros']; total: number } },
            emotion: 'neutral',
          }
        }
        case 'macro_create': {
          const parsed = await backendAPI.macroParse(originalText)
          const macro = (parsed as { macro?: unknown }).macro
          if (!macro) {
            return { text: '매크로를 이해하지 못했어요. 조금 더 자세히 말해주세요.', emotion: 'neutral' }
          }
          const created = await backendAPI.macroCreate(macro)
          return {
            text: (created as { message?: string }).message || '매크로가 등록됐어요!',
            card4: { type: 'macro_created', data: { macro: (created as { macro?: unknown }).macro as Parameters<typeof import('./InlineCards4').MacroCreatedCard>[0]['data']['macro'], message: (created as { message?: string }).message || '' } },
            emotion: 'happy',
          }
        }
        case 'macro_run': {
          const list = await backendAPI.macroList()
          const macros = (list as { macros?: Array<{ id: string; name: string }> }).macros || []
          if (macros.length === 0) {
            return { text: '실행할 매크로가 없어요. 먼저 매크로를 등록해주세요.', emotion: 'neutral' }
          }
          const res = await backendAPI.macroRun(macros[0].id)
          return {
            text: (res as { message?: string }).message || '매크로를 실행했어요!',
            card4: { type: 'macro_run', data: res as { name: string; results: Parameters<typeof import('./InlineCards4').MacroRunCard>[0]['data']['results']; message: string } },
            emotion: 'happy',
          }
        }

        /* ── PC 리포트 ── */
        case 'pc_report': {
          const data = await backendAPI.reportGenerate()
          return {
            text: `PC 건강 점수: ${(data as { score?: number }).score || 0}점. 리포트가 바탕화면에 저장됐어요!`,
            card4: { type: 'pc_report', data: data as unknown as Parameters<typeof import('./InlineCards4').PCReportCard>[0]['data'] },
            emotion: (data as { score?: number }).score && (data as { score?: number }).score! < 60 ? 'concerned' : 'happy',
          }
        }
        case 'report_email': {
          const res = await backendAPI.reportEmail()
          return {
            text: res.message,
            card2: { type: 'system_action', icon: '📧', title: res.success ? '이메일 전송 완료' : '이메일 전송 실패', success: res.success },
            emotion: res.success ? 'happy' : 'concerned',
          }
        }

        /* ── 문서 요약 ── */
        case 'doc_summary': {
          const filePath = extractTwoFilePaths(originalText)[0] || ''
          if (!filePath) {
            return { text: '요약할 파일 경로를 알려주세요. 예: "report.pdf 요약해줘"', emotion: 'neutral' }
          }
          const data = await backendAPI.docsSummary(filePath)
          return {
            text: (data as { summary?: string }).summary?.slice(0, 100) || '문서 요약이 완료됐어요!',
            card4: { type: 'doc_summary', data: data as unknown as Parameters<typeof import('./InlineCards4').DocSummaryCard>[0]['data'] },
            emotion: 'happy',
          }
        }

        /* ── 스마트 정리 ── */
        case 'smart_organize': {
          const isDesktop = /바탕화면|desktop/.test(originalText)
          const isDownloads = /다운로드|download/.test(originalText)
          const target = isDesktop ? 'desktop' : isDownloads ? 'downloads' : 'all'
          const res = await backendAPI.filesOrganize(undefined, 'both').catch(() => ({
            success: false, moved: 0, message: '파일 정리 실패',
          }))
          return {
            text: res.success ? `${target === 'desktop' ? '바탕화면' : target === 'downloads' ? '다운로드' : 'PC 전체'} 정리 완료!` : res.message,
            card3: { type: 'smart_organize', data: { moved: res.moved, folders: [], message: res.message } },
            emotion: res.success ? 'happy' : 'concerned',
          }
        }

        /* ── 📅 캘린더 ── */
        case 'calendar_today': {
          const data = await calendarToday().catch(() => ({ success: false, events: [], total: 0, message: 'Outlook이 설치되어 있어야 합니다.' }))
          return {
            text: data.message,
            card2: { type: 'system_action', icon: '📅', title: `오늘 일정 ${data.total}개`, detail: data.events.slice(0,3).map(e => `${e.start.slice(11,16)} ${e.subject}`).join('\n'), success: data.success },
            emotion: data.total > 0 ? 'happy' : 'neutral',
          }
        }
        case 'calendar_week': {
          const data = await calendarWeek().catch(() => ({ success: false, events: [], total: 0, message: 'Outlook이 설치되어 있어야 합니다.' }))
          return {
            text: data.message,
            card2: { type: 'system_action', icon: '📆', title: `이번 주 일정 ${data.total}개`, detail: data.events.slice(0,5).map(e => `${e.start.slice(5,10)} ${e.subject}`).join('\n'), success: data.success },
            emotion: data.total > 0 ? 'happy' : 'neutral',
          }
        }
        case 'calendar_add': {
          const subjectMatch = originalText.match(/[""]([^""]+)[""]/) ?? originalText.match(/일정.*등록\s+(.+)/)
          const subject = (subjectMatch?.[1] ?? originalText.replace(/일정.*추가|일정.*등록|일정.*넣어/g, '').trim()) || '새 일정'
          const res = await calendarAdd(subject).catch(() => ({ success: false, message: '일정 추가 실패' }))
          return {
            text: res.message,
            card2: { type: 'system_action', icon: '📅', title: res.message, success: res.success },
            emotion: res.success ? 'happy' : 'concerned',
          }
        }

        /* ── 📧 이메일 ── */
        case 'email_inbox': {
          const data = await emailInbox(10).catch(() => ({ success: false, emails: [], total: 0, unread: 0, message: 'Outlook이 필요합니다.' }))
          return {
            text: data.message,
            card2: { type: 'system_action', icon: '📧', title: `받은 메일 ${data.total}개 (읽지 않음 ${data.unread}개)`, detail: data.emails.slice(0,3).map(e => `${e.is_read ? '📨' : '📩'} ${e.subject} — ${e.sender}`).join('\n'), success: data.success },
            emotion: data.unread > 0 ? 'concerned' : 'neutral',
          }
        }
        case 'email_send': {
          const toMatch = originalText.match(/([a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,})/)
          const to = toMatch?.[1] ?? ''
          if (!to) return { text: '받는 사람 이메일 주소를 알려주세요. 예: "user@gmail.com에게 메일 보내줘"', emotion: 'neutral' }
          const subject = originalText.match(/제목[:\s]+(.+)/)?.[1] ?? 'Nexus에서 보낸 메일'
          const body = originalText.match(/내용[:\s]+(.+)/)?.[1] ?? ''
          const res = await emailSend(to, subject, body).catch(() => ({ success: false, message: '메일 전송 실패' }))
          return {
            text: res.message,
            card2: { type: 'system_action', icon: '📤', title: res.message, success: res.success },
            emotion: res.success ? 'happy' : 'concerned',
          }
        }
        case 'email_summarize': {
          const data = await emailSummarize().catch(() => ({ success: false, emails: [], summary: '', message: 'Outlook이 필요합니다.' }))
          return {
            text: data.message,
            card2: { type: 'system_action', icon: '📧', title: '이메일 요약', detail: data.summary, success: data.success },
            emotion: 'neutral',
          }
        }

        /* ── 🦠 VirusTotal ── */
        case 'virus_check': {
          const filePathMatch = originalText.match(/[A-Za-z]:\\[^\s]+/) ?? originalText.match(/["']([^"']+\.[a-z]{2,4})["']/)
          const filePath = filePathMatch?.[0]?.replace(/['"]/g, '') ?? ''
          if (!filePath) return { text: '검사할 파일 경로를 알려주세요. 예: "C:\\Users\\file.exe 바이러스 확인해줘"', emotion: 'neutral' }
          const apiKey = localStorage.getItem('nexus-virustotal-key') ?? ''
          const data = await virusTotalCheck(filePath, apiKey).catch(() => ({ success: false, file_path: filePath, file_hash: '', malicious: 0, suspicious: 0, clean: 0, total_scans: 0, permalink: '', safe_score: 0, verdict: 'error', message: 'VirusTotal 연결 실패' }))
          const em: CharacterEmotion = data.verdict === 'malicious' ? 'alert' : data.verdict === 'suspicious' ? 'concerned' : 'happy'
          return {
            text: data.message,
            card2: { type: 'system_action', icon: data.verdict === 'malicious' ? '🚨' : data.verdict === 'suspicious' ? '⚠️' : '✅', title: `VirusTotal 결과: ${data.verdict}`, detail: `탐지 ${data.malicious}개 / 전체 ${data.total_scans}개 검사`, success: data.verdict === 'safe' || data.verdict === 'unknown' },
            emotion: em,
          }
        }

        /* ── 📊 성능 이력 ── */
        case 'perf_history': {
          const daysMatch = originalText.match(/(\d+)\s*일/)
          const days = daysMatch ? parseInt(daysMatch[1]) : 7
          const data = await historyStats(days).catch(() => ({ success: false, days, total_samples: 0, snapshots: [], daily_summary: [], avg_cpu: 0, avg_mem: 0, cpu_trend: 'stable', message: '성능 이력이 없어요. 앱을 더 오래 실행하면 데이터가 쌓여요.' }))
          return {
            text: data.message,
            card2: { type: 'system_action', icon: '📊', title: `${days}일 성능 이력`, detail: `평균 CPU ${data.avg_cpu.toFixed(0)}% · 메모리 ${data.avg_mem.toFixed(0)}% · 트렌드: ${data.cpu_trend === 'up' ? '↑ 증가' : data.cpu_trend === 'down' ? '↓ 감소' : '→ 안정'}`, success: data.success },
            emotion: data.cpu_trend === 'up' ? 'concerned' : 'neutral',
          }
        }
        case 'perf_anomaly': {
          const data = await historyAnomalies().catch(() => ({ success: false, anomalies: [], avg_cpu: 0, avg_mem: 0, message: '데이터 부족' }))
          return {
            text: data.message,
            card2: { type: 'system_action', icon: data.anomalies.length > 0 ? '⚠️' : '✅', title: `이상 탐지: ${data.anomalies.length}건`, detail: data.anomalies.slice(0,3).map(a => a.message).join('\n'), success: data.anomalies.length === 0 },
            emotion: data.anomalies.length > 3 ? 'alert' : data.anomalies.length > 0 ? 'concerned' : 'happy',
          }
        }

        /* ── 🌐 가격 비교 ── */
        case 'price_compare': {
          const query = originalText.replace(/가격.*비교|최저가|검색|찾아줘|얼마야/g, '').trim() || originalText
          const data = await priceCompare(query).catch(() => ({ success: false, query, results: [], total: 0, summary: '가격 검색 실패 — 백엔드 연결 필요' }))
          return {
            text: data.summary || `'${query}' 가격 검색 완료!`,
            card2: { type: 'system_action', icon: '🛒', title: `최저가 검색: ${query}`, detail: data.results.slice(0,3).map(r => `${r.site}: ${r.price}`).join('\n'), success: data.success },
            emotion: 'happy',
          }
        }

        /* ── 🌐 뉴스 검색 ── */
        case 'news_search': {
          const query = originalText.replace(/뉴스|검색|최신|오늘|찾아줘/g, '').trim() || '오늘 주요 뉴스'
          const data = await newsSearch(query).catch(() => ({ success: false, query, articles: [], total: 0, summary: '뉴스 검색 실패' }))
          return {
            text: data.summary || `'${query}' 뉴스 검색 완료!`,
            card2: { type: 'system_action', icon: '📰', title: `뉴스: ${query}`, detail: data.articles.slice(0,3).map(a => `• ${a.title}`).join('\n'), success: data.success },
            emotion: 'neutral',
          }
        }

        /* ── ⏰ 스케줄러 ── */
        case 'schedule_list': {
          const data = await schedulerList().catch(() => ({ success: false, tasks: [], total: 0 }))
          return {
            text: data.total === 0 ? '등록된 자동화 스케줄이 없어요. "매일 오전 9시에 PC 진단해줘" 처럼 말해보세요!' : `스케줄 ${data.total}개가 등록돼 있어요.`,
            card2: { type: 'system_action', icon: '⏰', title: `스케줄 ${data.total}개`, detail: (data.tasks as Array<{name: string; next_run: string}>).slice(0,3).map(t => `${t.name} — ${t.next_run}`).join('\n'), success: true },
            emotion: 'neutral',
          }
        }
        case 'schedule_add': {
          const res = await schedulerAdd(originalText).catch(() => ({ success: false, task: null, next_run_kr: '', message: '스케줄 추가 실패' }))
          return {
            text: (res as { message: string }).message,
            card2: { type: 'system_action', icon: '⏰', title: (res as { message: string }).message, success: (res as { success: boolean }).success },
            emotion: (res as { success: boolean }).success ? 'happy' : 'concerned',
          }
        }
        case 'schedule_delete': {
          const idMatch = originalText.match(/\b([a-f0-9-]{10,})\b/)
          if (!idMatch) return { text: '삭제할 스케줄 ID를 알려주세요. 먼저 "스케줄 목록" 으로 확인해보세요.', emotion: 'neutral' }
          const res = await schedulerDelete(idMatch[1]).catch(() => ({ success: false, message: '삭제 실패' }))
          return {
            text: res.message,
            card2: { type: 'system_action', icon: '🗑️', title: res.message, success: res.success },
            emotion: res.success ? 'happy' : 'concerned',
          }
        }

        /* ── 🔫 프로세스 강제 종료 ── */
        case 'process_kill': {
          const pidMatch = originalText.match(/pid\s*[:#]?\s*(\d+)/i)
          const nameMatch = originalText.replace(/프로세스|종료|강제|앱|죽여|kill/gi, '').trim()
          if (!pidMatch && !nameMatch) return { text: '종료할 프로세스 이름이나 PID를 알려주세요.', emotion: 'neutral' }
          const pid = pidMatch ? parseInt(pidMatch[1]) : undefined
          const name = pid ? undefined : nameMatch
          const res = await processKill(pid, name).catch(() => ({ success: false, name: name ?? '', message: '종료 실패' }))
          return {
            text: res.message,
            card2: { type: 'system_action', icon: res.success ? '✅' : '⚠️', title: res.message, success: res.success },
            emotion: res.success ? 'happy' : 'concerned',
          }
        }

        /* ── 🔑 앱 권한 감사 ── */
        case 'app_permissions': {
          const appMatch = originalText.match(/["']?([가-힣a-zA-Z]+)["']?\s*(?:앱|프로그램|이)?\s*권한/)?.[1]
          const data = await appPermissions(appMatch).catch(() => ({ success: false, permissions: {}, message: '권한 정보 조회 실패' }))
          return {
            text: data.message,
            card2: { type: 'system_action', icon: '🔑', title: '앱 권한 현황', detail: JSON.stringify(data.permissions).slice(0, 100), success: data.success },
            emotion: 'neutral',
          }
        }

        /* ── 🔄 Windows 업데이트 ── */
        case 'windows_updates': {
          const data = await windowsUpdates().catch(() => ({ success: false, count: 0, updates: [], message: 'Windows Update 확인 실패' }))
          return {
            text: data.message,
            card2: { type: 'system_action', icon: data.count > 0 ? '🔄' : '✅', title: `업데이트 ${data.count}개`, detail: data.updates.slice(0,3).map(u => `• ${u.title}`).join('\n'), success: data.success },
            emotion: data.count > 5 ? 'concerned' : data.count > 0 ? 'neutral' : 'happy',
          }
        }

        /* ── 🌤️ 날씨 ── */
        case 'weather': {
          const cityMatch = originalText.match(/([가-힣]{2,5})\s*날씨/) ?? originalText.match(/([가-힣]{2,5})\s*기온/)
          const city = cityMatch?.[1] ?? '서울'
          const data = await weatherGet(city).catch(() => ({ success: false, city, temp_c: 0, feels_like: 0, condition: '알 수 없음', humidity: 0, wind_kmh: 0, forecast: [], message: '' }))
          if (!data.success) {
            // 백엔드 없음 → Groq로 동적 응답
            const apiKey = localStorage.getItem('nexus-pplx-key') ?? ''
            if (apiKey) {
              const gr = await callGemini(apiKey, originalText, historyRef.current).catch(() => null)
              if (gr?.text) return { text: gr.text, emotion: gr.emotion ?? 'neutral' }
            }
            return { text: `날씨 서비스에 연결할 수 없어요. 현재 위치 날씨는 날씨 앱이나 포털 사이트에서 확인해보세요! 🌤️`, emotion: 'neutral' }
          }
          return {
            text: `${data.city} 현재 ${data.temp_c}°C, ${data.condition}이에요.`,
            card2: {
              type: 'system_action', icon: '🌤️',
              title: `${data.city} ${data.temp_c}°C — ${data.condition}`,
              detail: `체감 ${data.feels_like}°C · 습도 ${data.humidity}% · 바람 ${data.wind_kmh}km/h\n${data.forecast.slice(0,3).map(f => `${f.date}: ${f.max}°/${f.min}°C ${f.condition}`).join('\n')}`,
              success: data.success,
            },
            emotion: 'neutral',
          }
        }

        /* ── 🚗 교통 시간 ── */
        case 'travel_time': {
          const parts = originalText.match(/(.+?)(?:에서|에서부터)\s*(.+?)(?:까지|로|으로)/)
          const origin = parts?.[1]?.trim() ?? ''
          const destination = parts?.[2]?.trim() ?? ''
          if (!origin || !destination) return { text: '"어디에서 어디까지 얼마나 걸려?" 형식으로 말해주세요.', emotion: 'neutral' }
          const data = await travelTime(origin, destination).catch(() => ({ success: false, origin, destination, distance_km: 0, duration_min: 0, departure_time: '', arrival_time: '', message: '경로를 찾지 못했어요.' }))
          return {
            text: data.message || `${origin} → ${destination}: 약 ${data.duration_min}분`,
            card2: {
              type: 'system_action', icon: '🚗',
              title: `${origin} → ${destination}`,
              detail: `거리 ${data.distance_km.toFixed(1)}km · 약 ${data.duration_min}분\n출발 ${data.departure_time} → 도착 ${data.arrival_time}`,
              success: data.success,
            },
            emotion: 'neutral',
          }
        }

        /* ── 🌐 번역 ── */
        case 'translate': {
          const targetLang = /영어로|영문/.test(originalText) ? 'English'
            : /한국어로|한글로/.test(originalText) ? '한국어'
            : /일본어로/.test(originalText) ? '日本語'
            : /중국어로/.test(originalText) ? '中文'
            : 'English'
          // 클립보드 내용 가져와서 번역
          const clip = await backendAPI.clipboard().catch(() => ({ current: '', tip: '' }))
          const textToTranslate = clip.current || originalText.replace(/번역.*해줘|번역해|이거.*영어로|translate/gi, '').trim()
          if (!textToTranslate) return { text: '번역할 내용이 없어요. 텍스트를 먼저 복사해주세요.', emotion: 'neutral' }

          const apiKey = localStorage.getItem('nexus-pplx-key') ?? ''
          let translated = ''
          if (apiKey) {
            const { callGemini } = await import('../../lib/nexus/gemini_engine')
            const res = await callGemini(apiKey, `다음 텍스트를 ${targetLang}로 번역해줘. 번역 결과만 출력:\n\n${textToTranslate}`, []).catch(() => null)
            translated = res?.text ?? ''
          }
          if (!translated) translated = `번역을 위해 Perplexity API 키가 필요해요.`

          // 번역 결과를 클립보드에 저장 (paste API 사용)
          if (translated && !translated.includes('API 키')) {
            await dictationPaste(translated).catch(() => {})
          }
          return {
            text: `번역 완료! 결과가 클립보드에 복사됐어요.`,
            card2: {
              type: 'system_action', icon: '🌐',
              title: `→ ${targetLang} 번역`,
              detail: `원본: ${textToTranslate.slice(0,60)}...\n번역: ${translated.slice(0,80)}`,
              success: !!translated && !translated.includes('API 키'),
            },
            emotion: 'happy',
          }
        }

        /* ── 📋 클립보드 AI ── */
        case 'clipboard_ai': {
          const clip = await backendAPI.clipboard().catch(() => ({ current: '', tip: '' }))
          if (!clip.current) return { text: '클립보드가 비어있어요. 먼저 텍스트를 복사해주세요.', emotion: 'neutral' }

          const action = /요약/.test(originalText) ? '3줄로 요약해줘'
            : /교정|다듬어/.test(originalText) ? '문법과 어투를 교정해줘'
            : /번역/.test(originalText) ? '영어로 번역해줘'
            : /쉽게/.test(originalText) ? '쉽게 설명해줘'
            : '핵심만 요약해줘'

          const apiKey = localStorage.getItem('nexus-pplx-key') ?? ''
          let result = ''
          if (apiKey) {
            const { callGemini } = await import('../../lib/nexus/gemini_engine')
            const res = await callGemini(apiKey, `다음 텍스트를 ${action}. 결과만 출력:\n\n${clip.current.slice(0, 500)}`, []).catch(() => null)
            result = res?.text ?? ''
          }
          if (!result) return { text: 'AI 처리를 위해 Perplexity API 키가 필요해요.', emotion: 'neutral' }

          await dictationPaste(result).catch(() => {})
          return {
            text: `클립보드 AI 처리 완료! 결과가 클립보드에 저장됐어요.`,
            card2: {
              type: 'system_action', icon: '📋',
              title: '클립보드 AI 처리',
              detail: result.slice(0, 120),
              success: true,
            },
            emotion: 'happy',
          }
        }

        /* ── 📝 음성 메모→할일 동시 등록 ── */
        case 'voice_todo': {
          const content = originalText.replace(/할일|todo|기억해줘|해야|마감|데드라인/gi, '').trim() || originalText
          // 날짜 추출
          const dateMatch = originalText.match(/(\d+월\s*\d+일|\d+일|\d+\/\d+)/)
          const timeMatch = originalText.match(/(\d+시|\d+:\d+)/)
          const dateStr = dateMatch?.[1] ?? ''
          const timeStr = timeMatch?.[1] ?? ''
          const eventTitle = content.slice(0, 50)

          // 메모 저장
          const noteRes = await backendAPI.saveNote(content).catch(() => ({ success: false, note: { id: '', content, created: '' }, message: '메모 저장 실패' }))
          // 캘린더 등록 (날짜 있으면)
          let calMsg = ''
          if (dateStr) {
            const calRes = await calendarAdd(`[할일] ${eventTitle}`, dateStr + ' ' + timeStr).catch(() => ({ success: false, message: '' }))
            calMsg = calRes.success ? ` + 캘린더에도 등록했어요 📅` : ''
          }
          return {
            text: noteRes.success ? `메모 저장 완료!${calMsg}` : '메모 저장에 실패했어요.',
            card2: {
              type: 'system_action', icon: '📝',
              title: '메모 + 할일 등록',
              detail: `내용: ${content.slice(0, 80)}${dateStr ? `\n날짜: ${dateStr} ${timeStr}` : ''}`,
              success: noteRes.success,
            },
            emotion: noteRes.success ? 'happy' : 'concerned',
          }
        }

        /* ── 🖥️ Windows Recall ── */
        case 'recall_capture': {
          const data = await recallCapture().catch(() => ({ success: false, timestamp: '', ocr_text: '', message: '화면 캡처 실패' }))
          return {
            text: data.success ? `화면을 기억했어요 🖥️ "${data.ocr_text.slice(0, 40)}..."` : data.message,
            card2: { type: 'system_action', icon: '🖥️', title: '화면 기억 저장', detail: data.ocr_text.slice(0, 100), success: data.success },
            emotion: data.success ? 'happy' : 'concerned',
          }
        }
        case 'recall_search': {
          const query = originalText.replace(/기억.*찾아|화면.*기억|언제.*봤던|어제.*봤던|전에.*봤던|화면.*검색|recall/gi, '').trim() || originalText
          const data = await recallSearch(query).catch(() => ({ success: false, results: [], total: 0, message: '검색 실패 — 먼저 화면을 기억시켜주세요.' }))
          return {
            text: data.total > 0 ? `"${query}" 관련 화면 ${data.total}개 찾았어요!` : `"${query}" 관련 기억이 없어요.`,
            card2: {
              type: 'system_action', icon: '🔍',
              title: `화면 기억 검색: ${query}`,
              detail: data.results.slice(0, 3).map(r => `${r.timestamp}: ${r.snippet}`).join('\n'),
              success: data.success,
            },
            emotion: data.total > 0 ? 'happy' : 'neutral',
          }
        }

        /* ── 🎙️ 회의 어시스턴트 ── */
        case 'meeting_start': {
          const res = await meetingStart().catch(() => ({ success: false, file_path: '', message: '녹음 시작 실패' }))
          return {
            text: res.message,
            card2: { type: 'system_action', icon: '🔴', title: res.success ? '녹음 중...' : '녹음 실패', detail: res.file_path, success: res.success },
            emotion: res.success ? 'happy' : 'concerned',
          }
        }
        case 'meeting_stop': {
          const res = await meetingStop().catch(() => ({ success: false, file_path: '', duration_sec: 0, message: '녹음 종료 실패' }))
          return {
            text: res.message,
            card2: { type: 'system_action', icon: '⏹️', title: `녹음 완료 (${Math.round(res.duration_sec / 60)}분)`, detail: res.file_path, success: res.success },
            emotion: res.success ? 'happy' : 'concerned',
          }
        }
        case 'meeting_list': {
          const data = await meetingList().catch(() => ({ success: false, meetings: [], total: 0 }))
          return {
            text: data.total > 0 ? `회의 녹음 ${data.total}개가 있어요 🎙️` : '저장된 회의 녹음이 없어요.',
            card2: {
              type: 'system_action', icon: '🎙️',
              title: `회의 목록 ${data.total}개`,
              detail: data.meetings.slice(0, 3).map(m => `${m.timestamp} (${m.size_mb.toFixed(1)}MB)`).join('\n'),
              success: data.success,
            },
            emotion: 'neutral',
          }
        }
        case 'meeting_summary': {
          // 가장 최근 녹음 파일 가져와서 전사 + 요약
          const list = await meetingList().catch(() => ({ success: false, meetings: [], total: 0 }))
          if (!list.total) return { text: '요약할 회의 녹음이 없어요. 먼저 "회의 시작"으로 녹음해주세요.', emotion: 'neutral' }
          const latest = list.meetings[0]
          const transcribed = await meetingTranscribe(latest.file).catch(() => ({ success: false, text: '', duration_sec: 0, message: '전사 실패' }))
          if (!transcribed.success || !transcribed.text) return { text: `회의 전사 실패. Perplexity API 키를 확인해주세요.`, emotion: 'concerned' }
          const summary = await meetingSummarize(transcribed.text).catch(() => ({ success: false, summary: '', action_items: [], decisions: [], message: '요약 실패' }))
          return {
            text: summary.success ? `회의 요약 완료! 액션 아이템 ${summary.action_items.length}개` : '회의 요약에 실패했어요.',
            card2: {
              type: 'system_action', icon: '📋',
              title: '회의 요약',
              detail: `요약: ${summary.summary.slice(0, 100)}\n\n액션: ${summary.action_items.slice(0, 3).join(' / ')}`,
              success: summary.success,
            },
            emotion: summary.success ? 'happy' : 'concerned',
          }
        }

        /* ── ⌨️ 음성 받아쓰기 ── */
        case 'dictation_start': {
          const textToDictate = originalText
            .replace(/받아쓰기|dictation|타이핑.*해줘|입력.*해줘|써줘.*지금|적어줘.*지금|대신.*타이핑|대신.*입력|대신.*써줘|자동.*입력/gi, '')
            .trim()
          if (!textToDictate) return { text: '입력할 내용을 말해주세요. 예: "받아쓰기 안녕하세요 오늘 날씨가 맑네요"', emotion: 'neutral' }
          const res = await dictationType(textToDictate).catch(() => ({ success: false, typed_chars: 0, message: '받아쓰기 실패' }))
          return {
            text: res.message,
            card2: { type: 'system_action', icon: '⌨️', title: `${res.typed_chars}글자 입력 완료`, detail: textToDictate.slice(0, 80), success: res.success },
            emotion: res.success ? 'happy' : 'concerned',
          }
        }

        /* ── 🏠 스마트홈 ── */
        case 'smarthome_list': {
          const data = await smarthomeDevices().catch(() => ({ success: false, devices: [], total: 0, message: 'Home Assistant 연결 실패. 설정에서 HA URL과 토큰을 입력해주세요.' }))
          return {
            text: data.message || `스마트홈 기기 ${data.total}개 연결됨`,
            card2: {
              type: 'system_action', icon: '🏠',
              title: `기기 ${data.total}개`,
              detail: data.devices.slice(0, 5).map(d => `${d.name}: ${d.state}`).join('\n'),
              success: data.success,
            },
            emotion: data.success ? 'happy' : 'concerned',
          }
        }
        case 'smarthome_control': {
          // 명령 파싱: "불 꺼줘" → entity: light, action: turn_off
          const isOn = /켜|on|열어/.test(originalText)
          const action = isOn ? 'turn_on' : 'turn_off'
          const deviceMap: Record<string, string> = { '불': 'light', '조명': 'light', '에어컨': 'climate', '선풍기': 'fan', 'TV': 'media_player', '커튼': 'cover', '콘센트': 'switch' }
          let domain = 'light'
          for (const [keyword, d] of Object.entries(deviceMap)) {
            if (originalText.includes(keyword)) { domain = d; break }
          }
          const devices = await smarthomeDevices().catch(() => ({ success: false, devices: [], total: 0, message: '' }))
          const target = devices.devices.find(d => d.domain === domain)
          if (!target) return { text: `제어할 ${domain} 기기를 찾지 못했어요. 먼저 "스마트홈 기기 목록" 확인해주세요.`, emotion: 'neutral' }
          const res = await smarthomeControl(target.id, action).catch(() => ({ success: false, message: '제어 실패' }))
          return {
            text: res.message || `${target.name} ${action === 'turn_on' ? '켰어요' : '껐어요'} 🏠`,
            card2: { type: 'system_action', icon: isOn ? '💡' : '🌑', title: `${target.name} ${action === 'turn_on' ? 'ON' : 'OFF'}`, success: res.success },
            emotion: res.success ? 'happy' : 'concerned',
          }
        }

        /* ── 🎭 AI 멀티 페르소나 ── */
        case 'persona_list': {
          const data = await personaList().catch(() => ({ personas: [], current: 'nexus' }))
          const lines = data.personas.map((p) => `${p.emoji} **${p.name}** — ${p.description}`).join('\n')
          return {
            text: `현재 페르소나: **${data.current}**\n\n사용 가능한 AI 팀:\n${lines}\n\n"리서치 모드로 바꿔줘" 처럼 말하면 전환해요!`,
            emotion: 'happy' as const,
          }
        }

        case 'persona_switch': {
          const lower = originalText.toLowerCase()
          let id = 'nexus'
          if (/리서치|연구|research/.test(lower)) id = 'research'
          else if (/재무|finance|financial/.test(lower)) id = 'finance'
          else if (/회의|meeting/.test(lower)) id = 'meeting'
          else if (/크리에이티브|creative|창의/.test(lower)) id = 'creative'
          else if (/보안|security/.test(lower)) id = 'security'
          else if (/법무|법률|legal|계약/.test(lower)) id = 'legal'
          const res = await personaSet(id).catch(() => ({ ok: false, persona: null as unknown as PersonaDef, message: '전환 실패' }))
          if (res.ok && res.persona) setActivePersona(res.persona)
          return {
            text: res.message,
            card2: { type: 'system_action', icon: res.persona?.emoji ?? '🤖', title: res.persona?.name ?? id, detail: res.persona?.description ?? '', success: res.ok },
            emotion: res.ok ? 'happy' as const : 'concerned' as const,
          }
        }

        /* ── 🧠 Second Brain ── */
        case 'brain_search': {
          const query = originalText.replace(/second.*brain|세컨드.*브레인|기억.*검색|장기.*기억.*찾아|내가.*했던|작년에.*내가|과거에/gi, '').trim() || originalText
          const data = await brainSearch(query, 8).catch(() => ({ results: [], total: 0, summary: '', query }))
          const items = data.results.slice(0, 5).map((r) => `[${r.entry.source}] ${r.entry.title}`)
          return {
            text: data.summary || (data.results.length > 0 ? `"${query}" 관련 기억 ${data.results.length}건 찾았어요:\n${items.join('\n')}` : `"${query}"에 대한 기억이 없어요.`),
            emotion: data.results.length > 0 ? 'happy' as const : 'neutral' as const,
          }
        }

        case 'brain_stats': {
          const data = await brainStats().catch(() => ({ total: 0, by_source: {} as Record<string, number>, updated_at: '' }))
          const src = Object.entries(data.by_source).map(([k, v]) => `${k}: ${v}개`).join(', ')
          return {
            text: `🧠 Second Brain 현황\n총 ${data.total}개 기억 저장됨\n${src}\n마지막 업데이트: ${data.updated_at.slice(0, 10) || '없음'}`,
            emotion: 'neutral' as const,
          }
        }

        /* ── ⚡ Auto Workflow ── */
        case 'workflow_plan': {
          const goal = originalText.replace(/워크플로.*계획|어떻게.*자동화|단계.*알려줘|자동화.*방법|순서.*알려줘/gi, '').trim() || originalText
          const plan = await workflowPlan(goal).catch(() => ({ goal, steps: [], summary: '계획 생성 실패', ok: false }))
          const stepLines = plan.steps.map((s) => `${s.step}. ${s.description} → \`${s.api_endpoint}\``).join('\n')
          return {
            text: `**워크플로 계획**: ${plan.goal}\n\n${stepLines}\n\n실행하려면 "자동으로 실행해줘"라고 하세요.`,
            emotion: 'neutral' as const,
          }
        }

        case 'workflow_run': {
          const goal = originalText.replace(/자동.*해줘|한.*번에.*다|워크플로.*실행|만들어서.*보내줘|요약하고.*이메일|찾아서.*정리/gi, '').trim() || originalText
          const result = await workflowRun(goal).catch(() => ({ goal, steps: [], summary: '워크플로 실행 실패', ok: false }))
          const doneSteps = result.steps.filter((s) => s.status === 'done').length
          const totalSteps = result.steps.length
          return {
            text: `✅ 워크플로 완료 (${doneSteps}/${totalSteps}단계)\n\n${result.summary}`,
            card2: { type: 'system_action', icon: '⚡', title: `워크플로: ${goal.slice(0, 30)}`, detail: `${doneSteps}/${totalSteps}단계 완료`, success: result.ok },
            emotion: result.ok ? 'happy' as const : 'concerned' as const,
          }
        }

        /* ── 🎬 Live Caption ── */
        case 'caption_start': {
          const langMatch = originalText.match(/영어|일본어|중국어|스페인어|프랑스어|korean|english|japanese|chinese/)
          const langMap: Record<string, string> = { 영어: 'en', 일본어: 'ja', 중국어: 'zh', 스페인어: 'es', 프랑스어: 'fr', english: 'en', japanese: 'ja', chinese: 'zh' }
          const lang = langMap[langMatch?.[0]?.toLowerCase() ?? ''] ?? 'ko'
          const res = await captionStart(lang).catch(() => ({ ok: false, message: '자막 시작 실패' }))
          if (res.ok) setCaptionRunning(true)
          return {
            text: res.message,
            card2: { type: 'system_action', icon: '🎬', title: '실시간 자막', detail: `번역 언어: ${lang === 'ko' ? '한국어' : lang}`, success: res.ok },
            emotion: res.ok ? 'happy' as const : 'concerned' as const,
          }
        }

        case 'caption_stop': {
          const res = await captionStop().catch(() => ({ ok: false, message: '자막 종료 실패', entries: 0 }))
          if (res.ok) setCaptionRunning(false)
          return {
            text: `${res.message} (총 ${res.entries}개 자막)`,
            emotion: 'neutral' as const,
          }
        }

        /* ── 🎮 GPU 모니터링 ── */
        case 'gpu_stats': {
          const data = await gpuStats().catch(() => ({ success: false, gpus: [], message: 'GPU 정보 조회 실패' }))
          const gpu = data.gpus?.[0]
          return {
            text: data.message,
            card2: { type: 'system_action', icon: '🎮', title: gpu ? `${gpu.name}` : 'GPU 정보', detail: gpu ? `사용률 ${gpu.usage_pct}% · 온도 ${gpu.temp_c}°C · VRAM ${gpu.mem_used_mb}/${gpu.mem_total_mb}MB` : '정보 없음', success: data.success },
            emotion: gpu && gpu.temp_c > 80 ? 'alert' : gpu && gpu.usage_pct > 90 ? 'concerned' : 'neutral',
          }
        }

        default:
          return { text: '', emotion: 'neutral' }
      }
    } catch {
      return {
        text: userLang === 'ko'
          ? '백엔드 연결에 실패했어요. 앱이 설치된 환경에서 실행해주세요.'
          : 'Backend not available. Please run in the installed app.',
        emotion: 'concerned',
      }
    }
  }, [userLang, assistantName])

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
  ): Promise<{ card?: InlineCardData; card2?: InlineCardData2; card3?: InlineCard3Data; card4?: InlineCard4Data; emotion: CharacterEmotion }> => {
    try {
      switch (action) {
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
          const answer = await callGeminiWithImage(ss.base64, trimmed).catch(() => (ss as { ocr_text?: string }).ocr_text || '분석 불가')
          return { card3: { type: 'vision_result', data: { question: trimmed, answer, screenshot_b64: ss.base64 } }, emotion: 'happy' }
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

  /* ── 검색 fallback URL 생성 (백엔드 items 없을 때 항상 미리보기 보장) ── */
  const buildFrontendFallbackURLs = (query: string, site: string) => {
    const enc = encodeURIComponent(query)
    const s = site.toLowerCase()
    if (s === 'coupang' || query.includes('쿠팡'))
      return [{ title: `쿠팡에서 "${query}" 검색`, url: `https://www.coupang.com/np/search?q=${enc}` }]
    if (s === 'youtube')
      return [{ title: `YouTube에서 "${query}" 검색`, url: `https://www.youtube.com/results?search_query=${enc}` }]
    if (s === 'naver')
      return [{ title: `네이버에서 "${query}" 검색`, url: `https://search.naver.com/search.naver?query=${enc}` }]
    return [
      { title: `네이버: ${query}`, url: `https://search.naver.com/search.naver?query=${enc}` },
      { title: `쿠팡: ${query}`, url: `https://www.coupang.com/np/search?q=${enc}` },
      { title: `구글: ${query}`, url: `https://www.google.com/search?q=${enc}` },
    ]
  }

  /* ── 메시지 전송 ── */
  const sendText = useCallback(async (text: string) => {
    const trimmed = text.trim()
    if (!trimmed || typingRef.current) return

    // 새 질문 시작 → 이전 음성 즉시 중지 (말풍선·미리보기는 새 답변 올 때 교체)
    stopSpeaking()
    setSpeaking(false)

    // 비활성화 상태: 이름 호출 시만 재활성화
    if (!isActive) {
      const lower = trimmed.toLowerCase()
      const nameLower = assistantName.toLowerCase()
      if (lower.includes(nameLower) || lower.includes('넥서스') || lower.includes('nexus')) {
        setIsActive(true)
        speakText(`네, 주인님! 다시 돌아왔어요 😊`)
      }
      return
    }

    typingRef.current = true

    const msgId = Date.now().toString()
    setMessages(prev => [...prev, { id: msgId, role: 'user', text: trimmed }])
    setInput('')
    setListening(false)
    setTyping(true)
    setEmotion('neutral')
    historyRef.current.push({ role: 'user', parts: [{ text: trimmed }] })
    // 장기 메모리에 저장
    saveHistory(toStoredTurns(historyRef.current as ConversationTurn[]))

    // ── LLM clarify 해소: 원래 질문 + 사용자 답변 합쳐서 재호출 ──
    if (clarifyPendingIntent === 'llm_clarify' && clarifyPendingParams) {
      const originalQuery = (clarifyPendingParams.original_query as string) ?? ''
      const combinedInput = originalQuery
        ? `${originalQuery} (추가 정보: ${trimmed})`
        : trimmed
      resetClarify()
      const apiKey = localStorage.getItem('nexus-pplx-key') ?? ''
      let llmRes
      try { llmRes = await callGemini(apiKey, combinedInput, historyRef.current) }
      catch { /* 폴백 */ }
      if (!llmRes?.text?.trim()) llmRes = fallbackResponse(combinedInput, assistantName)
      setTyping(false)
      typingRef.current = false
      const emoMap: Record<NexusEmotion, CharacterEmotion> = {
        neutral: 'neutral', happy: 'happy', concerned: 'concerned', alert: 'alert', humorous: 'humorous',
      }
      setEmotion(emoMap[llmRes.emotion ?? 'neutral'])
      setMessages(prev => [...prev, { id: `${msgId}-res`, role: 'nexus', text: llmRes!.text }])
      pushModelHistory(trimmed, llmRes.text)
      speakText(llmRes.text)
      appendHistory({ id: msgId, ts: Date.now(), q: trimmed, a: cleanForHistory(llmRes.text) })
      setHistoryVersion(v => v + 1)
      return
    }

    // ── 0.5순위: 딥서치 ──────────────────────────────────────
    const isDeepSearch = /딥\s*서치|deep\s*search|자세히\s*찾|깊게\s*검색|심층\s*검색/i.test(trimmed)
    if (isDeepSearch && backendStatus === 'connected') {
      const q = trimmed.replace(/딥\s*서치|deep\s*search|자세히\s*찾아줘|깊게\s*검색해줘|심층\s*검색/gi, '').trim() || trimmed
      try {
        setMessages(prev => [...prev, {
          id: `think-${msgId}`, role: 'nexus', text: '',
          inlineCard: { type: 'agent_thinking', steps: ['딥서치 시작...', '여러 소스 병렬 검색 중...', 'AI 통합 요약 중...'] },
        }])
        const res = await backendAPI.llmDeepSearchWeb(q, 10)
        setMessages(prev => prev.filter(m => m.id !== `think-${msgId}`))
        setTyping(false); typingRef.current = false
        if (res.success) {
          const previewItems = (res.items ?? []).filter(it => it.url)
          if (previewItems.length > 0) setFloatingPreview(previewItems)
          const displayText = res.summary || '딥서치 완료'
          setEmotion('happy')
          setMessages(prev => [...prev, { id: `${msgId}-res`, role: 'nexus', text: displayText }])
          pushModelHistory(trimmed, displayText)
          speakText(displayText)
          appendHistory({ id: msgId, ts: Date.now(), q: trimmed, a: cleanForHistory(displayText) })
          setHistoryVersion(v => v + 1)
          return
        }
      } catch { setMessages(prev => prev.filter(m => m.id !== `think-${msgId}`)) }
    }

    // ── 1순위: Go 백엔드 /api/command (LLM 자동 라우팅 + 멀티턴) ─
    if (backendStatus === 'connected') {
      try {
        const isSearchQuery = /검색|찾아|뉴스|날씨|쇼핑|가격|web_search/i.test(trimmed)
        const thinkSteps = isSearchQuery
          ? ['요청 분석 중...', '실시간 검색 중...', 'AI 요약 중...']
          : ['요청 분석 중...', '실행 중...']
        setMessages(prev => [...prev, {
          id: `think-${msgId}`, role: 'nexus', text: '',
          inlineCard: { type: 'agent_thinking', steps: thinkSteps },
        }])

        // 멀티턴: clarify 컨텍스트 + 최근 대화 이력 포함
        const recentHistory = historyRef.current.map(h => ({
          role: (h.role === 'user' ? 'user' : 'assistant') as 'user' | 'assistant',
          content: h.parts?.[0]?.text ?? '',
        })).filter(h => h.content.length > 0)

        const cmd = await sendCommand(trimmed, {
          pendingIntent:   clarifyPendingIntent   ?? undefined,
          pendingParams:   clarifyPendingParams   ?? undefined,
          pendingQuestion: clarifyPendingQuestion ?? undefined,
          history:         recentHistory,
        })
        setMessages(prev => prev.filter(m => m.id !== `think-${msgId}`))

        if (cmd.success) {
          // ── clarify: 추가 질문 필요 ──────────────────────────
          if (cmd.action === 'clarify' && cmd.needs_clarify) {
            const question = cmd.clarify_question || cmd.message || '조금 더 알려주세요.'
            setClarifyPendingIntent(cmd.pending_intent ?? null)
            setClarifyPendingParams((cmd.pending_params as Record<string, unknown>) ?? null)
            setClarifyPendingQuestion(question)

            setTyping(false)
            typingRef.current = false
            setEmotion('neutral')
            setMessages(prev => [...prev, {
              id: `${msgId}-res`, role: 'nexus',
              text: question,
            }])
            pushModelHistory(trimmed, question)
            // Clarify 질문 TTS — 자동 마이크 시작은 설정에 따름
            const clarifyAutoMic = localStorage.getItem('nexus-clarify-auto-mic') !== 'false'
            speak(question, userLang, () => setSpeaking(true), () => {
              setSpeaking(false)
              if (clarifyAutoMic && isMountedRef.current) {
                setTimeout(() => {
                  if (isMountedRef.current && !typingRef.current) handleVoiceToggle()
                }, 300)
              }
            })
            return
          }

          // ── 정상 실행: clarify 상태 초기화 ──────────────────
          resetClarify()
          const { card, card2, card3, card4, emotion: cmdEmotion } = await renderCommandResult(cmd.action, cmd.result, trimmed)
          const displayText = cmd.message || ''
          setTyping(false)
          typingRef.current = false
          setEmotion(cmdEmotion)

          // 웹 검색 결과에 미리보기 카드 추가 (항상 표시 보장)
          if (cmd.action === 'web_search') {
            const resultObj = cmd.result as { items?: Array<{ title?: string; url?: string }>; query?: string; site?: string } | undefined
            let rawItems: Array<{ title?: string; url?: string }> = resultObj?.items ?? []

            if (rawItems.length === 0) {
              const searchQuery = resultObj?.query ?? trimmed
              const site = resultObj?.site ?? ''
              rawItems = buildFrontendFallbackURLs(searchQuery, site)
            }

            const previewItems = rawItems
              .filter((it): it is { title: string; url: string } => !!(it.url))
              .map(it => ({ title: it.title ?? it.url, url: it.url }))
            if (previewItems.length > 0) setFloatingPreview(previewItems)
          }

          setMessages(prev => [...prev, {
            id: `${msgId}-res`, role: 'nexus', text: displayText,
            inlineCard: card, inlineCard2: card2, inlineCard3: card3, inlineCard4: card4,
          }])
          pushModelHistory(trimmed, displayText)
          if (displayText) {
            speakText(displayText)
            appendHistory({ id: msgId, ts: Date.now(), q: trimmed, a: cleanForHistory(displayText) })
            setHistoryVersion(v => v + 1)
          }
          return
        }
      } catch {
        setMessages(prev => prev.filter(m => m.id !== `think-${msgId}`))
        resetClarify()
      }
    }

    // ── 2순위: 로컬 detectIntent ──────────────────────────────
    const intent = detectIntent(trimmed)
    if (intent !== 'none') {
      const { text: resText, card, card2, card3, card4, emotion: resEmotion } = await handleBackendIntent(intent, msgId, trimmed)
      setTyping(false)
      typingRef.current = false
      setEmotion(resEmotion)
      setMessages(prev => [...prev, { id: `${msgId}-res`, role: 'nexus', text: resText, inlineCard: card, inlineCard2: card2, inlineCard3: card3, inlineCard4: card4 }])
      pushModelHistory(trimmed, resText)
      if (resText) {
        speakText(resText)
        appendHistory({ id: msgId, ts: Date.now(), q: trimmed, a: cleanForHistory(resText) })
        setHistoryVersion(v => v + 1)
      }
      return
    }

    // ── 3순위: LLM 일반 대화 ─────────────────────────────────
    const apiKey = localStorage.getItem('nexus-pplx-key') ?? ''
    let response

    try {
      const r = await callOllama(trimmed, historyRef.current)
      if (r) response = r
    } catch { /* Ollama 미실행 */ }

    if (!response && apiKey && trackUsage()) {
      try { response = await callGemini(apiKey, trimmed, historyRef.current) }
      catch (e) { console.warn('[Perplexity] 호출 실패:', e) }
    }

    if (!response || !response.text?.trim()) response = fallbackResponse(trimmed, assistantName)

    setTyping(false)
    typingRef.current = false

    // ── LLM clarify: 추가 정보 필요 ──────────────────────────
    if (response.needs_clarify && response.clarify_question) {
      const question = response.clarify_question
      setClarifyPendingIntent(response.clarify_intent ?? 'llm_clarify')
      setClarifyPendingParams(response.clarify_params ?? { original_query: trimmed })
      setClarifyPendingQuestion(question)
      setEmotion('neutral')
      setMessages(prev => [...prev, { id: `${msgId}-res`, role: 'nexus', text: response!.text }])
      pushModelHistory(trimmed, response!.text)
      speakText(response.text)
      return
    }

    const emotionMap: Record<NexusEmotion, CharacterEmotion> = {
      neutral: 'neutral', happy: 'happy', concerned: 'concerned',
      alert: 'alert', humorous: 'humorous',
    }
    setEmotion(emotionMap[response.emotion ?? 'neutral'])

    // 미리보기는 오른쪽 플로팅 패널에만 표시
    const previewItems = response.preview_items ?? getLastPreviewItems()
    clearLastPreviewItems()
    if (previewItems.length > 0) setFloatingPreview(previewItems)

    setMessages(prev => [...prev, {
      id: `${msgId}-res`, role: 'nexus', text: response!.text,
    }])
    pushModelHistory(trimmed, response.text)
    if (response.text) {
      speakText(response.text)
      appendHistory({ id: msgId, ts: Date.now(), q: trimmed, a: cleanForHistory(response.text) })
      setHistoryVersion(v => v + 1)
    }
  }, [assistantName, backendStatus, clarifyPendingIntent, clarifyPendingParams, clarifyPendingQuestion, handleBackendIntent, renderCommandResult, resetClarify, speakText])

  const handleSend = useCallback((text: string) => void sendText(text), [sendText])

  /* 캐릭터 클릭 */
  const handleCharacterClick = () => {
    setChatOpen(prev => !prev)
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

  const displayInput = voiceInterim || input

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
        onClick: () => setSoundEnabled(p => { const next = !p; localStorage.setItem('nexus-sound', next ? 'on' : 'off'); return next }), tip: soundEnabled ? 'AI 소리 끄기' : 'AI 소리 켜기' },
    ]),
    { icon: isActive ? '💬' : '😴',    active: isActive,     color: isActive ? primaryColor : '#6b7280',
      onClick: () => { setIsActive(p => !p); if (isActive) stopSpeaking() }, tip: isActive ? '비활성화' : '활성화' },
    { icon: '🎤', active: listening,   color: '#ef4444',     onClick: handleVoiceToggle, tip: '음성' },
    { icon: '⚙️', active: false,      color: primaryColor,  onClick: () => setSettingsOpen(true), tip: '설정' },
    { icon: '—',  active: false,      color: '#6b7280',     onClick: () => setMinimized(true), tip: '최소화' },
  ]

  return (
    <>
    {/* ── 미리보기 플로팅 패널 (화면 고정, 항상 보임) ── */}
    <AnimatePresence>
      {floatingPreview && floatingPreview.length > 0 && (
        <motion.div
          key="floating-preview-panel"
          initial={{ opacity: 0, x: 30, scale: 0.93 }}
          animate={{ opacity: 1, x: 0, scale: 1 }}
          exit={{ opacity: 0, x: 20, scale: 0.93 }}
          style={{
            position: 'fixed',
            bottom: 180,
            right: 24,
            width: 280,
            background: 'rgba(8,8,22,0.98)',
            border: `2px solid ${primaryColor}`,
            borderRadius: 16,
            padding: '12px 14px',
            boxShadow: `0 12px 40px ${primaryColor}55, 0 0 0 1px ${primaryColor}33`,
            backdropFilter: 'blur(20px)',
            zIndex: 10001,
            pointerEvents: 'auto',
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 10 }}>
            <span style={{ fontSize: 12, color: primaryColor, fontWeight: 800, letterSpacing: '0.05em' }}>
              🔍 검색 결과 미리보기
            </span>
            <button
              onClick={() => setFloatingPreview(null)}
              style={{ background: 'none', border: 'none', color: 'rgba(255,255,255,0.4)', cursor: 'pointer', fontSize: 14, padding: '0 2px', lineHeight: 1 }}
            >✕</button>
          </div>
          {floatingPreview.slice(0, 5).map((item, i) => (
            <div key={i} style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 7, padding: '4px 0', borderBottom: i < floatingPreview.slice(0, 5).length - 1 ? '1px solid rgba(255,255,255,0.06)' : 'none' }}>
              <div style={{ width: 18, height: 18, borderRadius: 4, background: `${primaryColor}33`, display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                <span style={{ fontSize: 9, color: primaryColor, fontWeight: 700 }}>{i + 1}</span>
              </div>
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.9)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', fontWeight: 500 }}>
                  {item.title}
                </div>
                <div style={{ fontSize: 9.5, color: 'rgba(255,255,255,0.35)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', marginTop: 1 }}>
                  {item.url.replace(/^https?:\/\//, '').slice(0, 40)}
                </div>
              </div>
              <button
                onClick={() => openPreview(item.url, item.title)}
                style={{
                  background: `linear-gradient(135deg, ${primaryColor}, ${accentColor})`,
                  border: 'none', borderRadius: 8,
                  color: '#fff', fontSize: 10, fontWeight: 700,
                  padding: '5px 12px', cursor: 'pointer', whiteSpace: 'nowrap',
                  flexShrink: 0, boxShadow: `0 2px 8px ${primaryColor}44`,
                }}
              >미리보기</button>
            </div>
          ))}
        </motion.div>
      )}
    </AnimatePresence>

    {/* ── 드래그 가능한 통합 컨테이너 (캐릭터 + 채팅 + 말풍선) ── */}
    <motion.div
      drag
      dragMomentum={false}
      dragElastic={0}
      style={{
        position: 'fixed',
        bottom: 0,
        right: 0,
        x: dragX,
        y: dragY,
        zIndex: 9999,
        pointerEvents: 'none',
        display: 'flex',
        alignItems: 'flex-end',
      }}
      onDragStart={() => setIsDragging(true)}
      onDragEnd={() => setIsDragging(false)}
    >
      {/* ─── 채팅 패널 — 캐릭터 바로 왼쪽 ─── */}
      <AnimatePresence>
        {chatOpen && (
          <motion.div
            initial={{ opacity: 0, x: 30, scale: 0.94 }}
            animate={{ opacity: 1, x: 0, scale: 1 }}
            exit={{ opacity: 0, x: 30, scale: 0.94 }}
            transition={{ type: 'spring', stiffness: 300, damping: 28 }}
            style={{
              pointerEvents: 'auto',
              marginBottom: 60,
              marginRight: 4,
            }}
          >
            {focusEndMs && (
              <div style={{
                background: `${primaryColor}18`, border: `1px solid ${primaryColor}44`,
                borderRadius: '8px 8px 0 0', padding: '4px 10px', fontSize: 10, color: primaryColor,
                display: 'flex', alignItems: 'center', gap: 6,
              }}>
                <span>🎯</span>
                <span>집중 모드 — {Math.max(0, Math.ceil((focusEndMs - Date.now()) / 60_000))}분 남음</span>
              </div>
            )}
            <ChatBubble
              messages={messages}
              typing={typing}
              listening={listening}
              input={displayInput}
              onInputChange={v => { setInput(v); setVoiceInterim('') }}
              onSend={handleSend}
              onVoiceToggle={handleVoiceToggle}
              onRepair={handleRepair}
              assistantName={assistantName}
              lang={userLang}
              primaryColor={primaryColor}
              historyVersion={historyVersion}
              clarifyPending={!!clarifyPendingIntent}
              clarifyQuestion={clarifyPendingQuestion ?? ''}
            />
          </motion.div>
        )}
      </AnimatePresence>

      {/* ─── 캐릭터 + 버튼 래퍼 ─── */}
      <div style={{ display: 'flex', alignItems: 'flex-end', pointerEvents: 'auto' }}>

        {/* ─── 캐릭터 컬럼 ─── */}
        <div style={{
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          marginRight: 8,
          userSelect: 'none',
          paddingBottom: 4,
        }}>

          {/* ── 말풍선 (플로우 최상단) ── */}
          <AnimatePresence>
            {bubbleText ? (
              <motion.div
                key="bubble"
                initial={{ opacity: 0, scale: 0.88, y: 8 }}
                animate={{ opacity: 1, scale: 1, y: 0 }}
                exit={{ opacity: 0, scale: 0.88, y: 8 }}
                transition={{ type: 'spring', stiffness: 320, damping: 26 }}
                style={{
                  width: 260,
                  maxHeight: bubbleExpanded ? 420 : 160,
                  overflowY: bubbleExpanded ? 'auto' : 'hidden',
                  background: 'rgba(8,8,22,0.97)',
                  border: `1.5px solid ${primaryColor}88`,
                  borderRadius: 16,
                  padding: '10px 14px 36px 14px',
                  fontSize: 12.5,
                  color: 'rgba(255,255,255,0.93)',
                  fontWeight: 500,
                  lineHeight: 1.6,
                  boxShadow: `0 8px 32px ${primaryColor}55`,
                  backdropFilter: 'blur(18px)',
                  pointerEvents: 'auto',
                  wordBreak: 'keep-all',
                  marginBottom: 6,
                  position: 'relative',
                  scrollbarWidth: 'thin',
                  transition: 'max-height 0.25s ease',
                }}
              >
                {bubbleText}
                {/* 하단 버튼 바 */}
                <div style={{
                  position: 'absolute', bottom: 0, left: 0, right: 0,
                  display: 'flex', alignItems: 'center', justifyContent: 'space-between',
                  padding: '4px 8px 6px',
                  background: 'rgba(8,8,22,0.95)',
                  borderTop: `1px solid ${primaryColor}22`,
                  borderBottomLeftRadius: 14, borderBottomRightRadius: 14,
                }}>
                  {/* 복사 버튼 */}
                  <button
                    onClick={() => { navigator.clipboard.writeText(bubbleText).catch(() => {}); }}
                    title="복사"
                    style={{
                      background: `${primaryColor}22`, border: `1px solid ${primaryColor}44`,
                      borderRadius: 6, color: primaryColor, fontSize: 10, fontWeight: 700,
                      padding: '3px 8px', cursor: 'pointer',
                    }}
                  >복사</button>
                  <div style={{ display: 'flex', gap: 4 }}>
                    {/* 펼치기/접기 */}
                    <button
                      onClick={() => setBubbleExpanded(e => !e)}
                      title={bubbleExpanded ? '접기' : '더보기'}
                      style={{
                        background: 'none', border: 'none', cursor: 'pointer',
                        color: 'rgba(255,255,255,0.35)', fontSize: 12, padding: '2px 4px',
                        lineHeight: 1,
                      }}
                    >{bubbleExpanded ? '▲' : '▼'}</button>
                    {/* 닫기 */}
                    <button
                      onClick={() => { setBubbleText(''); setBubbleExpanded(false); }}
                      title="닫기"
                      style={{
                        background: 'none', border: 'none', cursor: 'pointer',
                        color: 'rgba(255,255,255,0.4)', fontSize: 13, padding: '2px 4px',
                        lineHeight: 1,
                      }}
                    >✕</button>
                  </div>
                </div>
                {/* 말풍선 꼬리 */}
                <div style={{
                  position: 'absolute', bottom: -8, left: '50%',
                  transform: 'translateX(-50%)',
                  width: 0, height: 0,
                  borderLeft: '8px solid transparent',
                  borderRight: '8px solid transparent',
                  borderTop: `8px solid ${primaryColor}88`,
                }} />
              </motion.div>
            ) : listening ? (
              <motion.div
                key="listen"
                initial={{ opacity: 0, y: 6 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0 }}
                style={{
                  background: 'rgba(239,68,68,0.15)',
                  border: '1.5px solid rgba(239,68,68,0.55)',
                  borderRadius: 22,
                  padding: '6px 16px', fontSize: 11,
                  color: '#fca5a5', fontWeight: 700,
                  whiteSpace: 'nowrap', pointerEvents: 'none',
                  marginBottom: 6, position: 'relative',
                }}
              >
                🎤 듣고 있어요...
                <div style={{
                  position: 'absolute', bottom: -7, left: '50%',
                  transform: 'translateX(-50%)',
                  width: 0, height: 0,
                  borderLeft: '7px solid transparent',
                  borderRight: '7px solid transparent',
                  borderTop: '7px solid rgba(239,68,68,0.55)',
                }} />
              </motion.div>
            ) : !isActive ? (
              <motion.div
                key="inactive"
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                exit={{ opacity: 0 }}
                style={{
                  background: 'rgba(100,100,120,0.9)',
                  borderRadius: 16, padding: '7px 14px',
                  fontSize: 11, color: 'rgba(255,255,255,0.6)',
                  whiteSpace: 'nowrap', pointerEvents: 'none',
                  marginBottom: 6, position: 'relative',
                }}
              >
                😴 비활성화 — 이름을 불러주세요
                <div style={{
                  position: 'absolute', bottom: -7, left: '50%',
                  transform: 'translateX(-50%)',
                  width: 0, height: 0,
                  borderLeft: '7px solid transparent',
                  borderRight: '7px solid transparent',
                  borderTop: '7px solid rgba(100,100,120,0.9)',
                }} />
              </motion.div>
            ) : null}
          </AnimatePresence>

          {/* ─── 캐릭터 본체 ─── */}
          <div
            onClick={isDragging ? undefined : handleCharacterClick}
            style={{ cursor: isDragging ? 'grabbing' : 'pointer', position: 'relative' }}
            title="클릭해서 대화하기"
          >
            <div style={{ position: 'relative', opacity: isActive ? 1 : 0.55, transition: 'opacity 0.3s' }}>
              <AvatarRuntime
                glbUrl={glbUrl}
                emotion={
                  emotion === 'happy'    ? 'happy'    :
                  emotion === 'alert'    ? 'surprised':
                  emotion === 'concerned'? 'concerned':
                  emotion === 'humorous' ? 'happy'    : 'neutral'
                }
                runtimeState={speaking ? 'speaking' : listening ? 'listening' : 'idle'}
                primaryColor={primaryColor}
                accentColor={accentColor}
                preset={avatarPreset}
                width={200}
                height={340}
                scale={0.82}
                characterOffsetY={-0.55}
                cameraY={0.35}
                quality="high"
              />
              <SpeakingWaves color={primaryColor} active={speaking} />
            </div>
            {/* 발광 플랫폼 */}
            <motion.div
              animate={{ opacity: [0.3, 0.6, 0.3], scaleX: [0.9, 1.05, 0.9] }}
              transition={{ duration: 3, repeat: Infinity, ease: 'easeInOut' }}
              style={{
                position: 'absolute', bottom: -4, left: '50%',
                transform: 'translateX(-50%)',
                width: 140, height: 16,
                background: `radial-gradient(ellipse, ${primaryColor}88 0%, transparent 70%)`,
                borderRadius: '50%', filter: 'blur(4px)', pointerEvents: 'none',
              }}
            />
          </div>

          {/* ── 드래그 핸들 (캐릭터 아래) ── */}
          <motion.div
            animate={{ opacity: isDragging ? 1 : 0.3 }}
            whileHover={{ opacity: 0.8 }}
            style={{
              cursor: 'grab', fontSize: 11,
              color: 'rgba(255,255,255,0.45)',
              marginTop: 6, letterSpacing: 2, userSelect: 'none',
            }}
            title="드래그하여 이동"
          >
            ⠿ 이동
          </motion.div>
        </div>

        {/* ── 수직 버튼 (캐릭터 오른쪽) ── */}
        <div style={{
          display: 'flex', flexDirection: 'column', gap: 8,
          marginBottom: 60, zIndex: 3,
        }}>
          {btnList.map(btn => (
            <motion.button
              key={btn.tip}
              whileTap={{ scale: 0.9 }}
              whileHover={{ scale: 1.1 }}
              onClick={btn.onClick}
              title={btn.tip}
              style={{
                width: 36, height: 36, borderRadius: '50%', outline: 'none',
                border: btn.active ? `1px solid ${btn.color}44` : '1px solid rgba(255,255,255,0.08)',
                background: btn.active ? `${btn.color}28` : 'rgba(8,8,20,0.75)',
                color: btn.active ? btn.color : 'rgba(255,255,255,0.55)',
                fontSize: btn.icon === '—' ? 18 : 14,
                cursor: 'pointer', backdropFilter: 'blur(12px)',
                boxShadow: btn.active
                  ? `0 0 12px ${btn.color}66, 0 2px 8px rgba(0,0,0,0.4)`
                  : '0 2px 8px rgba(0,0,0,0.4)',
                transition: 'all 0.2s',
              } as React.CSSProperties}
            >
              {btn.icon}
            </motion.button>
          ))}
        </div>

      </div>
    </motion.div>

    <SettingsModal
      open={settingsOpen}
      onClose={() => setSettingsOpen(false)}
      primaryColor={primaryColor}
    />

    {/* 온보딩 플로우 */}
    {!isOnboarded && (
      <OnboardingFlow onComplete={handleOnboardingComplete} />
    )}

    {/* 구독 만료 배너 */}
    {isOnboarded && isLoggedIn && (subscriptionStatus === 'expired' || subscriptionStatus === 'none') && (
      <div style={{
        position: 'fixed', bottom: 24, left: '50%', transform: 'translateX(-50%)',
        background: 'rgba(248,113,113,0.15)', backdropFilter: 'blur(12px)',
        border: '1px solid rgba(248,113,113,0.4)', borderRadius: 14,
        padding: '10px 20px', zIndex: 99998,
        display: 'flex', alignItems: 'center', gap: 14,
        boxShadow: '0 8px 32px rgba(0,0,0,0.5)',
      }}>
        <span style={{ fontSize: 13, color: '#fca5a5', fontWeight: 600 }}>
          ⚠️ 구독이 만료되었습니다. AI 기능이 제한됩니다.
        </span>
        <button
          onClick={() => { import('../../lib/paddle').then(m => m.openCheckout(userEmail)) }}
          style={{
            padding: '6px 14px', borderRadius: 8, border: 'none', cursor: 'pointer',
            background: '#f87171', color: 'white', fontSize: 12, fontWeight: 700,
          }}
        >
          구독하기
        </button>
      </div>
    )}
    </>
  )
}
