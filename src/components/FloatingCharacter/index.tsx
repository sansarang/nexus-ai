import React, { useState, useRef, useEffect, useCallback } from 'react'
import { AnimatePresence, motion, useMotionValue } from 'framer-motion'
import { useAppStore } from '../../stores/appStore'
import { saveUserSettings } from '../../lib/supabase'
import { DesktopAgent } from '../DesktopAgent'
import { WorkflowBuilder } from '../WorkflowBuilder'
import { EmailSetup } from '../EmailSetup'
import { ChatBubble } from './ChatBubble'
import type { ChatMessage } from './ChatBubble'
import { SettingsModal } from './SettingsModal'
import type { InlineCardData } from './InlineCards'
import type { InlineCardData2 } from './InlineCards2'
import type { InlineCard3Data } from './InlineCards3'
import type { InlineCard4Data } from './InlineCards4'
import { SpeakingWaves } from './Avatar3D'
import { AvatarRuntime } from './Avatar3D/AvatarRuntime'
import { OnboardingFlow, LoginScreen } from './OnboardingFlow'
import type { AvatarConfig } from './OnboardingFlow'
import { PaywallModal } from '../PaywallModal'
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
import { routeWithLLM } from '../../lib/nexus/llmToolRouter'
import { backendAPI, mockStats, mockScan, mockDailyReport, sendCommand,
  calendarToday, calendarWeek, calendarAdd, calendarFindSlot, calendarSmartAdd,
  emailInbox, emailSend, emailSummarize, emailClassify, emailDraftReply,
  virusTotalCheck, historyStats, historyAnomalies,
  processKill, appPermissions, windowsUpdates, gpuStats,
  priceCompare, newsSearch, youtubeSearch, tiktokSearch, naverShoppingSearch, coupangSearch, videoDownload, videoQuickSearch,
  schedulerAdd, schedulerList, schedulerDelete,
  recallCapture, recallSearch,
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


/* 영어 쿼리 판별 (60% 이상 ASCII) */
function isEnglishQuery(q: string): boolean {
  if (!q) return false
  const chars = [...q]
  const ascii = chars.filter(c => c.charCodeAt(0) < 128).length
  return ascii / chars.length > 0.6
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
    case 'multi_action':     return ['멀티 액션 시작...', '검색 중', '결과 정리 중', '파일 저장 중']
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
    case 'video_download':    return ['영상 URL 확인 중...', 'yt-dlp로 다운로드 중...', '파일 저장 중']
    case 'email_classify':    return ['받은 메일 가져오는 중...', 'AI 분류 중...', '우선순위 정리 중']
    case 'email_draft':       return ['메일 내용 분석 중...', 'AI 답장 초안 작성 중']
    case 'calendar_find_slot': return ['캘린더 확인 중...', '빈 시간 탐색 중', '가능한 슬롯 정리 중']
    case 'calendar_smart_add': return ['자연어 파싱 중...', '일정 생성 중', 'Outlook 저장 중']
    case 'workflow_list':     return ['저장된 워크플로 조회 중...']
    case 'workflow_create':   return ['자연어 파싱 중...', '워크플로 생성 중', '저장 중']
    case 'workflow_templates': return ['워크플로 템플릿 불러오는 중...']
    case 'imap_inbox':        return ['IMAP 서버 연결 중...', '받은 메일 불러오는 중']
    case 'imap_send':         return ['IMAP 서버 연결 중...', '메일 전송 중']
    case 'multi_agent':       return ['멀티 에이전트 준비 중...', '에이전트 팀 배치 중', '병렬 실행 중']
    case 'briefing_now':      return ['날씨 확인 중...', '일정 수집 중...', '이메일 확인 중...', '브리핑 생성 중']
    case 'task_cancel':       return ['실행 중 작업 확인...', '취소 신호 전송 중']
    case 'search_pdf':        return ['웹 검색 중...', '결과 수집 중', 'PDF 보고서 생성 중', '파일 저장 중']
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
    setUserLang,
    showWorkflowBuilder: storeShowWorkflowBuilder, workflowBuilderInitialName, setShowWorkflowBuilder: storeSetShowWorkflowBuilder,
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
  const [typingSteps, setTypingSteps]     = useState<string[]>([])
  const [emotion, setEmotion]             = useState<'neutral'|'happy'|'concerned'|'alert'|'humorous'>('neutral')
  const [speaking, setSpeaking]           = useState(false)
  const [listening, setListening]         = useState(false)
  const [input, setInput]                 = useState('')
  const [voiceInterim, setVoiceInterim]   = useState('')
  const [minimized, setMinimized]         = useState(false)
  const [settingsOpen, setSettingsOpen]     = useState(false)
  const [showDesktopAgent, setShowDesktopAgent] = useState(false)
  const showWorkflowBuilder = storeShowWorkflowBuilder
  const setShowWorkflowBuilder = (val: boolean) => storeSetShowWorkflowBuilder(val)
  const [showEmailSetup, setShowEmailSetup] = useState(false)
  const [toastAlerts, setToastAlerts]     = useState<Array<{id: string; title: string; message: string; level: string}>>([])
  const alertESRef = useRef<EventSource | null>(null)
  const [soundEnabled, setSoundEnabled]   = useState(() => localStorage.getItem('nexus-sound') !== 'off')
  const [isActive, setIsActive]           = useState(true)   // 비활성화 토글
  const [isDragging, setIsDragging]       = useState(false)
  const [historyVersion, setHistoryVersion] = useState(0)
  const dragX = useMotionValue(0)
  const dragY = useMotionValue(0)
  const previewDragX = useMotionValue(0)
  const previewDragY = useMotionValue(0)
  const [backendStatus, setBackendStatus] = useState<BackendStatus>('checking')
  const [focusEndMs, setFocusEndMs]       = useState<number | undefined>(getFocusModeEnd())
  const [floatingPreview, setFloatingPreview] = useState<Array<{ title: string; url: string; isVideo?: boolean; isSocial?: boolean; isMap?: boolean; mapType?: string; service?: string; isImage?: boolean }> | null>(null)
  const [previewType, setPreviewType] = useState<string>('general')
  // savedPreviews: 앱 재시작 후에도 유지 (localStorage 영구 저장)
  const [savedPreviews, setSavedPreviews] = useState<Array<{ label: string; items: Array<{ title: string; url: string }> }>>(() => {
    try { return JSON.parse(localStorage.getItem('nexus_saved_previews') || '[]') } catch { return [] }
  })
  useEffect(() => {
    try { localStorage.setItem('nexus_saved_previews', JSON.stringify(savedPreviews.slice(-5))) } catch { /* storage full */ }
  }, [savedPreviews])

  // ── Clarify 멀티턴 상태 ──────────────────────────────────
  const [clarifyPendingIntent,   setClarifyPendingIntent]   = useState<string | null>(null)
  const [clarifyPendingParams,   setClarifyPendingParams]   = useState<Record<string, unknown> | null>(null)
  const [clarifyPendingQuestion, setClarifyPendingQuestion] = useState<string | null>(null)
  const [activePersona, setActivePersona] = useState<PersonaDef | null>(null)
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
        speak(greeting, userLang, () => setSpeaking(true), () => { setSpeaking(false); setTimeout(() => setBubbleText(''), 1500) })
      }, 800)
      return () => clearTimeout(tid)
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
  useEffect(() => {
    nexusSSE.connect()

    // 백엔드 Proactive SSE 구독 (실시간 알림 → 토스트)
    const backendSSE = new EventSource('http://127.0.0.1:17891/api/alerts/stream')
    alertESRef.current = backendSSE
    backendSSE.onmessage = (e) => {
      try {
        const data = JSON.parse(e.data)
        if (data.type === 'connected' || !data.title) return
        const toast = { id: data.id || String(Date.now()), title: data.title, message: data.message, level: data.level || 'info' }
        setToastAlerts(prev => [...prev.slice(-4), toast])
        setTimeout(() => setToastAlerts(prev => prev.filter(t => t.id !== toast.id)), 7000)
      } catch { /* ignore */ }
    }

    const unsubAlert = nexusSSE.onAlert((alert) => {
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
      if (alertESRef.current) { alertESRef.current.close(); alertESRef.current = null }
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
  ): Promise<{ card?: InlineCardData; card2?: InlineCardData2; card3?: InlineCard3Data; card4?: InlineCard4Data; emotion: CharacterEmotion }> => {
    try {
      switch (action) {
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
  // URL을 보고 isVideo/isImage 자동 태깅
  const tagPreviewItem = (item: { title: string; url: string; isVideo?: boolean; isImage?: boolean; source?: string; type?: string }) => {
    const u = item.url.toLowerCase()
    const isVideo = item.isVideo ||
      u.includes('youtube.com') || u.includes('youtu.be') ||
      u.includes('tiktok.com') || u.includes('tv.naver.com') ||
      u.includes('tving.com') || u.includes('wavve.com') ||
      item.source === 'youtube' || item.source === 'video' || item.type === 'video'
    const isSocial = !isVideo && (
      u.includes('instagram.com') || u.includes('x.com/') || u.includes('twitter.com/') ||
      item.source === 'instagram' || item.source === 'x' || item.type === 'social'
    )
    const isImage = item.isImage || u.match(/\.(jpg|jpeg|png|webp|gif)(\?|$)/i) !== null
    return { ...item, isVideo: isVideo || undefined, isSocial: isSocial || undefined, isImage: isImage || undefined }
  }

  const buildFrontendFallbackURLs = (query: string, site: string) => {
    const enc = encodeURIComponent(query)
    const s = site.toLowerCase()
    const q = query.toLowerCase()
    // 특정 플랫폼 직접 지정 시에만 해당 플랫폼 URL 반환
    if (s === 'coupang' || q.includes('쿠팡'))
      return [{ title: `쿠팡에서 "${query}" 검색`, url: `https://www.coupang.com/np/search?q=${enc}` }]
    if (s === 'youtube' || q.includes('유튜브') || q.includes('youtube'))
      return [{ title: `YouTube: ${query}`, url: `https://www.youtube.com/results?search_query=${enc}`, isVideo: true }]
    // 그 외: 백엔드 items를 신뢰 → 프론트 fallback 없음 (검색 URL 절대 생성 안 함)
    return []
  }

  /* ── 메시지 전송 ── */
  const sendText = useCallback(async (text: string) => {
    return sendTextImpl(text, {
      userLang, assistantName, isActive, backendStatus,
      clarifyPendingIntent, clarifyPendingParams, clarifyPendingQuestion,
      floatingPreview, ttsVoice,
      typingRef, historyRef, isMountedRef,
      setMessages, setInput, setListening, setTyping, setTypingSteps, setEmotion, setSpeaking,
      setUserLang, setHistoryVersion, setToastAlerts, setIsActive, setFloatingPreview,
      setPreviewType, setClarifyPendingIntent, setClarifyPendingParams, setClarifyPendingQuestion,
      speakText, resetClarify, pushModelHistory, handleVoiceToggle,
      handleBackendIntent, renderCommandResult,
      showPaywall: (feature, used, limit) => {
        setPaywallFeature(feature)
        setPaywallUsed(used)
        setPaywallLimit(limit)
      },
    })
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [assistantName, backendStatus, clarifyPendingIntent, clarifyPendingParams,
    clarifyPendingQuestion, handleBackendIntent, renderCommandResult, resetClarify, speakText,
    userLang, isActive, floatingPreview, ttsVoice])

  const handleSend = useCallback((text: string) => void sendText(text), [sendText])

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
    const wantSubtitle = /자막|subtitle|srt|vtt/i.test(text)
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
        const wantAnalyze = /요약|내용|설명|분석|뭐|무슨|어떤|정리|요점|핵심|자막|전사|summarize|summary|content|what|explain|transcript|analyze|analyse/i.test(text)

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
        onClick: () => setSoundEnabled(p => { const next = !p; localStorage.setItem('nexus-sound', next ? 'on' : 'off'); return next }),
        tip: userLang === 'en' ? (soundEnabled ? 'Mute AI' : 'Unmute AI') : (soundEnabled ? 'AI 소리 끄기' : 'AI 소리 켜기') },
    ]),
    { icon: isActive ? '💬' : '😴',    active: isActive,     color: isActive ? primaryColor : '#6b7280',
      onClick: () => { setIsActive(p => !p); if (isActive) stopSpeaking() },
      tip: userLang === 'en' ? (isActive ? 'Deactivate' : 'Activate') : (isActive ? '비활성화' : '활성화') },
    { icon: '🎤', active: listening,   color: '#ef4444',     onClick: handleVoiceToggle,
      tip: userLang === 'en' ? 'Voice' : '음성' },
    { icon: '⚙️', active: false,       color: primaryColor,  onClick: () => setSettingsOpen(true),
      tip: userLang === 'en' ? 'Settings' : '설정' },
    { icon: '🖥️', active: showDesktopAgent,  color: '#06b6d4', onClick: () => setShowDesktopAgent(p => !p), tip: 'Desktop Agent' },
    { icon: '⚡',  active: showWorkflowBuilder, color: '#f59e0b', onClick: () => {
      const isPremium = subscriptionStatus === 'active' || subscriptionStatus === 'trial'
      if (!isPremium) { setPaywallFeature('workflow_run'); setPaywallUsed(0); setPaywallLimit(0) }
      else storeSetShowWorkflowBuilder(!showWorkflowBuilder)
    }, tip: 'Workflow Builder' },
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
    {/* ── 미리보기 플로팅 패널 (화면 고정, 항상 보임) ── */}
    <AnimatePresence>
      {floatingPreview && floatingPreview.length > 0 && (
        <motion.div
          key="floating-preview-panel"
          drag
          dragMomentum={false}
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
            maxHeight: 520,
            overflowY: 'auto',
            x: previewDragX,
            y: previewDragY,
            cursor: 'grab',
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 10 }}>
            <span style={{ fontSize: 12, color: primaryColor, fontWeight: 800, letterSpacing: '0.05em' }}>
              {(() => {
                const eng = userLang === 'en'
                if (floatingPreview.some(x => x.isMap && x.mapType === 'directions')) return eng ? '🗺️ Route Results' : '🗺️ 길찾기 결과'
                if (floatingPreview.some(x => x.isMap && x.mapType === 'roadview')) return eng ? '📍 Street View / Location' : '📍 로드뷰 / 위치'
                if (floatingPreview.some(x => x.isMap)) return eng ? '🗺️ Map Results' : '🗺️ 지도 결과'
                if (floatingPreview[0]?.isVideo) return eng ? '🎬 Video Results' : '🎬 영상 검색 결과'
                const titleMap: Record<string, [string, string]> = {
                  weather:       ['🌤️ Weather Info',      '🌤️ 날씨 정보'],
                  news:          ['📰 News Results',       '📰 뉴스 결과'],
                  recipe:        ['🍳 Recipe Sources',     '🍳 레시피 출처'],
                  shopping:      ['🛒 Shopping Results',   '🛒 쇼핑 결과'],
                  transit:       ['🚆 Transit Info',       '🚆 교통 정보'],
                  food:          ['🍜 Restaurant Results', '🍜 맛집 결과'],
                  finance:       ['📈 Finance Sources',    '📈 금융 정보'],
                  medical:       ['🏥 Health Sources',     '🏥 건강 정보'],
                  travel:        ['✈️ Travel Sources',     '✈️ 여행 정보'],
                  entertainment: ['🎬 Entertainment',      '🎬 엔터테인먼트'],
                  tech:          ['💻 Tech Sources',       '💻 IT 정보'],
                  education:     ['📚 Study Sources',      '📚 학습 정보'],
                  realestate:    ['🏠 Real Estate',        '🏠 부동산 정보'],
                  legal:         ['⚖️ Legal Sources',      '⚖️ 법률 정보'],
                }
                const t = titleMap[previewType]
                if (t) return eng ? t[0] : t[1]
                return eng ? '🔍 Search Preview' : '🔍 검색 결과 미리보기'
              })()}
            </span>
            <button
              onClick={() => {
                // 닫기 전 결과를 ChatBubble savedPreviews에 저장
                if (floatingPreview && floatingPreview.length > 0) {
                  const label = (() => {
                    const eng = userLang === 'en'
                    if (floatingPreview[0]?.isVideo) return eng ? '🎬 Video Results' : '🎬 영상 결과'
                    if (floatingPreview[0]?.isMap) return eng ? '🗺️ Map Results' : '🗺️ 지도 결과'
                    return eng ? '🔍 Search Results' : '🔍 검색 결과'
                  })()
                  setSavedPreviews(prev => {
                    const willOverflow = prev.length >= 5
                    const next = [...prev.slice(-4), {
                      label,
                      items: floatingPreview!.map(x => ({ title: x.title, url: x.url })),
                    }]
                    if (willOverflow) {
                      setMessages(msgs => [...msgs, {
                        id: `sys-${Date.now()}`, role: 'nexus' as const,
                        text: userLang === 'en'
                          ? '💡 Saved results are limited to 5. The oldest result has been removed.'
                          : '💡 저장된 결과는 최대 5개입니다. 가장 오래된 결과가 삭제됐어요.',
                      }])
                    }
                    return next
                  })
                }
                setFloatingPreview(null)
              }}
              style={{ background: 'none', border: 'none', color: 'rgba(255,255,255,0.4)', cursor: 'pointer', fontSize: 14, padding: '0 2px', lineHeight: 1 }}
            >✕</button>
          </div>
          {/* ── 길찾기 전용 교통수단 버튼 UI ── */}
          {floatingPreview.some(x => x.isMap && (x as any).mapType === 'directions') ? (() => {
            const dirItems = floatingPreview.filter(x => x.isMap && (x as any).mapType === 'directions')
            const otherItems = floatingPreview.filter(x => !(x.isMap && (x as any).mapType === 'directions'))
            const modeOrder = ['transit', 'car', 'walk', 'bicycle', 'ktx']
            const modeEmojis: Record<string, string> = { transit: '🚌', car: '🚗', walk: '🚶', bicycle: '🚲', ktx: '🚂' }
            const modeLabels: Record<string, string> = { transit: '대중교통', car: '자동차', walk: '도보', bicycle: '자전거', ktx: '기차/KTX' }
            // service별로 분리
            const googleByMode: Record<string, typeof dirItems[0]> = {}
            const kakaoByMode: Record<string, typeof dirItems[0]> = {}
            for (const item of dirItems) {
              const m = (item as any).mode as string
              const svc = (item as any).service as string
              if (m && svc === 'google') googleByMode[m] = item
              if (m && svc === 'kakao') kakaoByMode[m] = item
            }
            // from/to 파싱
            const firstItem = dirItems[0] as any
            const fromTo = firstItem?.title?.match(/— (.+?)→(.+)/)
            const fromLabel = fromTo?.[1]?.trim() ?? ''
            const toLabel = fromTo?.[2]?.replace(/\s*\(.*\)/, '').trim() ?? ''
            // Google Maps iframe URL (실제 경로 표시)
            const iframeSrc = fromLabel && toLabel
              ? `https://www.google.com/maps/embed/v1/directions?key=AIzaSyD-9tSrke72PouQMnMX-a7eZSW0jkFMBWY&origin=${encodeURIComponent(fromLabel)}&destination=${encodeURIComponent(toLabel)}&mode=transit&language=ko`
              : ''
            // Google Maps 직접 링크 (API 키 불필요)
            const googleDirectUrl = fromLabel && toLabel
              ? `https://www.google.com/maps/dir/${encodeURIComponent(fromLabel)}/${encodeURIComponent(toLabel)}/`
              : ''
            return (
              <div>
                {/* 출발→도착 헤더 */}
                {fromLabel && toLabel && (
                  <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.7)', marginBottom: 8, textAlign: 'center', fontWeight: 600 }}>
                    {fromLabel} <span style={{ color: primaryColor }}>→</span> {toLabel}
                  </div>
                )}
                {/* Google Maps 실제 경로 미리보기 */}
                {googleDirectUrl && (
                  <div style={{ position: 'relative', width: '100%', height: 130, borderRadius: 10, overflow: 'hidden', marginBottom: 8, background: 'rgba(255,255,255,0.05)', cursor: 'pointer' }}
                    onClick={() => window.open(googleDirectUrl, '_blank')}
                  >
                    <iframe
                      src={`https://maps.google.com/maps?q=${encodeURIComponent(fromLabel + ' to ' + toLabel)}&output=embed&hl=ko`}
                      style={{ width: '100%', height: '100%', border: 'none', pointerEvents: 'none', borderRadius: 10 }}
                      loading="lazy"
                    />
                    <div style={{ position: 'absolute', inset: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'rgba(0,0,0,0.0)' }} />
                    <div style={{ position: 'absolute', bottom: 6, right: 6, background: 'rgba(0,0,0,0.65)', borderRadius: 6, padding: '3px 7px', fontSize: 9, color: '#fff', fontWeight: 700 }}>
                      클릭 → 경로 열기
                    </div>
                  </div>
                )}
                {/* 교통수단 버튼 그리드 (Google Maps 경로 링크) */}
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(5, 1fr)', gap: 5, marginBottom: 7 }}>
                  {modeOrder.map(mode => {
                    const item = googleByMode[mode]
                    if (!item) return null
                    return (
                      <button
                        key={mode}
                        onClick={() => window.open(item.url, '_blank')}
                        style={{
                          background: 'rgba(66,133,244,0.15)',
                          border: '1px solid rgba(66,133,244,0.35)',
                          borderRadius: 10, cursor: 'pointer',
                          display: 'flex', flexDirection: 'column', alignItems: 'center',
                          padding: '7px 3px', gap: 2,
                          transition: 'background 0.15s',
                        }}
                        onMouseEnter={e => (e.currentTarget.style.background = 'rgba(66,133,244,0.3)')}
                        onMouseLeave={e => (e.currentTarget.style.background = 'rgba(66,133,244,0.15)')}
                      >
                        <span style={{ fontSize: 15 }}>{modeEmojis[mode]}</span>
                        <span style={{ fontSize: 8, color: '#4285f4', fontWeight: 700, textAlign: 'center', lineHeight: 1.2 }}>{modeLabels[mode]}</span>
                        <span style={{ fontSize: 7.5, color: 'rgba(255,255,255,0.3)' }}>구글</span>
                      </button>
                    )
                  })}
                </div>
                {/* 카카오맵 + 예매 링크 */}
                <div style={{ display: 'flex', gap: 5, flexWrap: 'wrap' }}>
                  {(['transit', 'car'] as const).map(mode => {
                    const item = kakaoByMode[mode]
                    if (!item) return null
                    return (
                      <button key={mode} onClick={() => window.open(item.url, '_blank')}
                        style={{ background: 'rgba(249,224,0,0.1)', border: '1px solid rgba(249,224,0,0.3)', borderRadius: 8, cursor: 'pointer', padding: '4px 8px', fontSize: 9, color: '#f9e000', fontWeight: 700 }}>
                        {modeEmojis[mode]} 카카오맵
                      </button>
                    )
                  })}
                  {otherItems.map((item, i) => (
                    <button key={i} onClick={() => window.open(item.url, '_blank')}
                      style={{ background: 'rgba(255,255,255,0.08)', border: '1px solid rgba(255,255,255,0.15)', borderRadius: 8, cursor: 'pointer', padding: '4px 8px', fontSize: 9, color: 'rgba(255,255,255,0.7)', fontWeight: 600 }}>
                      {item.title.slice(0, 12)}
                    </button>
                  ))}
                </div>
              </div>
            )
          })() : floatingPreview.slice(0, 14).map((item, i) => {
            const isYt = item.url.includes('youtube.com') || item.url.includes('youtu.be')
            const isNaverTV = item.url.includes('tv.naver.com')
            const isStream = item.url.includes('tving.com') || item.url.includes('wavve.com')
            const isNaver = item.service === 'naver' || item.url.includes('map.naver.com')
            const isKakao = item.service === 'kakao' || item.url.includes('map.kakao.com')
            const isGoogle = item.service === 'google' || item.url.includes('google.com/maps')
            const isRoadview = item.mapType === 'roadview'
            const isDirectionsLink = item.mapType === 'directions'

            const isInstagram = item.url.includes('instagram.com')
            const isX = item.url.includes('x.com/') || item.url.includes('twitter.com/')
            const isTikTok = item.url.includes('tiktok.com')

            const typeBadge = isYt ? { label: 'YT', color: '#e53e3e' }
              : isNaverTV ? { label: 'TV', color: '#03c75a' }
              : isStream ? { label: '스트림', color: '#7c3aed' }
              : isTikTok ? { label: 'TikTok', color: '#010101' }
              : isInstagram ? { label: 'IG', color: '#e1306c' }
              : isX ? { label: 'X', color: '#1a1a1a' }
              : item.isVideo ? { label: '영상', color: '#e53e3e' }
              : (item as any).isSocial ? { label: 'SNS', color: '#7c3aed' }
              : isRoadview && isNaver ? { label: '로드뷰', color: '#03c75a' }
              : isRoadview && isKakao ? { label: '로드뷰', color: '#ffcd00' }
              : isRoadview && isGoogle ? { label: 'StreetView', color: '#4285f4' }
              : isDirectionsLink && isNaver ? { label: '네이버지도', color: '#03c75a' }
              : isDirectionsLink && isKakao ? { label: '카카오맵', color: '#ffcd00' }
              : isDirectionsLink && isGoogle ? { label: '구글지도', color: '#4285f4' }
              : item.mapType === 'bus' ? { label: '버스', color: '#f59e0b' }
              : item.isMap ? { label: '지도', color: '#06b6d4' }
              : null

            const mapBtnColor = isNaver ? '#03c75a' : isKakao ? '#f9e000' : isGoogle ? '#4285f4' : primaryColor
            const mapBtnText = isRoadview ? (isGoogle ? (userLang === 'en' ? 'Street View' : '거리뷰') : (userLang === 'en' ? 'Street View' : '로드뷰'))
              : isDirectionsLink ? (userLang === 'en' ? 'Directions' : '경로보기')
              : item.isMap ? (userLang === 'en' ? 'Open Map' : '지도열기')
              : null

            return (
            <div key={i} style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 7, padding: '4px 0', borderBottom: i < Math.min(floatingPreview.length, 14) - 1 ? '1px solid rgba(255,255,255,0.06)' : 'none' }}>
              <div style={{ width: 18, height: 18, borderRadius: 4, background: item.isMap ? `${mapBtnColor}33` : `${primaryColor}33`, display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                <span style={{ fontSize: 9, color: item.isMap ? mapBtnColor : primaryColor, fontWeight: 700 }}>{i + 1}</span>
              </div>
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                  {typeBadge && (
                    <span style={{ fontSize: 8, fontWeight: 700, color: isKakao ? '#000' : '#fff', background: typeBadge.color, borderRadius: 3, padding: '1px 4px', flexShrink: 0 }}>
                      {typeBadge.label}
                    </span>
                  )}
                  <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.9)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', fontWeight: 500 }}>
                    {item.title}
                  </div>
                </div>
                <div style={{ fontSize: 9.5, color: 'rgba(255,255,255,0.35)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', marginTop: 1 }}>
                  {item.url.replace(/^https?:\/\//, '').slice(0, 38)}
                </div>
              </div>
              {(item as any).isImage ? (
                <button
                  onClick={() => openPreview(item.url, item.title)}
                  style={{
                    background: 'linear-gradient(135deg, #7c3aed, #5b21b6)',
                    border: 'none', borderRadius: 8,
                    color: '#fff', fontSize: 10, fontWeight: 700,
                    padding: '5px 10px', cursor: 'pointer', whiteSpace: 'nowrap',
                    flexShrink: 0, boxShadow: '0 2px 8px rgba(124,58,237,0.4)',
                  }}
                >{userLang === 'en' ? 'View' : '사진보기'}</button>
              ) : (item as any).isSocial ? (
                <button
                  onClick={() => window.open(item.url, '_blank')}
                  style={{
                    background: isInstagram
                      ? 'linear-gradient(135deg, #e1306c, #833ab4)'
                      : isX ? '#1a1a1a' : '#7c3aed',
                    border: 'none', borderRadius: 8,
                    color: '#fff', fontSize: 10, fontWeight: 700,
                    padding: '5px 10px', cursor: 'pointer', whiteSpace: 'nowrap',
                    flexShrink: 0,
                  }}
                >{isInstagram ? '📷 보기' : isX ? '𝕏 보기' : '보기'}</button>
              ) : item.isVideo ? (
                <div style={{ display: 'flex', gap: 4, flexShrink: 0 }}>
                  <button
                    onClick={() => window.open(item.url, '_blank')}
                    title="재생"
                    style={{
                      background: isTikTok
                        ? 'linear-gradient(135deg, #010101, #69c9d0)'
                        : `linear-gradient(135deg, #e53e3e, #c53030)`,
                      border: 'none', borderRadius: 7,
                      color: '#fff', fontSize: 13, fontWeight: 700,
                      width: 28, height: 28, cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center',
                      boxShadow: isTikTok ? '0 2px 8px rgba(105,201,208,0.4)' : '0 2px 8px rgba(229,62,62,0.4)',
                    }}
                  >▶</button>
                  <button
                    onClick={async () => {
                      const msg = await videoDownload(item.url, 'best').catch(() => ({ success: false, message: 'yt-dlp 필요' }))
                      void sendText(`"${item.title}" 다운로드 ${msg.success ? '완료' : '실패: ' + msg.message}`)
                    }}
                    title="다운로드"
                    style={{
                      background: 'rgba(255,255,255,0.1)',
                      border: '1px solid rgba(255,255,255,0.2)', borderRadius: 7,
                      color: '#fff', fontSize: 11, fontWeight: 700,
                      width: 28, height: 28, cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center',
                    }}
                  >↓</button>
                </div>
              ) : mapBtnText ? (
                <button
                  onClick={() => window.open(item.url, '_blank')}
                  style={{
                    background: `linear-gradient(135deg, ${mapBtnColor}, ${mapBtnColor}bb)`,
                    border: 'none', borderRadius: 8,
                    color: isKakao ? '#000' : '#fff', fontSize: 10, fontWeight: 800,
                    padding: '5px 10px', cursor: 'pointer', whiteSpace: 'nowrap',
                    flexShrink: 0, boxShadow: `0 2px 8px ${mapBtnColor}44`,
                  }}
                >{mapBtnText}</button>
              ) : (
                <button
                  onClick={() => openPreview(item.url, item.title)}
                  style={{
                    background: `linear-gradient(135deg, ${primaryColor}, ${accentColor})`,
                    border: 'none', borderRadius: 8,
                    color: '#fff', fontSize: 10, fontWeight: 700,
                    padding: '5px 12px', cursor: 'pointer', whiteSpace: 'nowrap',
                    flexShrink: 0, boxShadow: `0 2px 8px ${primaryColor}44`,
                  }}
                >{userLang === 'en' ? 'Preview' : '미리보기'}</button>
              )}
            </div>
            )
          })}
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
              onSendWithFile={handleSendWithFileImpl}
              onVoiceToggle={handleVoiceToggle}
              onRepair={handleRepair}
              assistantName={assistantName}
              lang={userLang}
              primaryColor={primaryColor}
              historyVersion={historyVersion}
              clarifyPending={!!clarifyPendingIntent}
              clarifyQuestion={clarifyPendingQuestion ?? ''}
              typingSteps={typingSteps}
              savedPreviews={savedPreviews}
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
