import { useState, useRef, useEffect, useCallback } from 'react'
import { AnimatePresence, motion } from 'framer-motion'
import { useAppStore } from '../../stores/appStore'
import { callGemini, callOllama, fallbackResponse, trackUsage, callGroqVision } from '../../lib/nexus/gemini_engine'
import { getAuthHeader } from '../../lib/nexus/backendAPI'
import { routeWithLLM, routeWithLLMMulti } from '../../lib/nexus/llmToolRouter'
import { isMultiStepTask, runAgent, planAgent, hasDangerousSteps, runAgentWithPlan, DANGEROUS_STEPS } from '../../lib/nexus/agentExecutor'
import type { AgentPlan, AgentStep } from '../../lib/nexus/agentExecutor'
import { buildMemoryContext, learnFromTurn, learnPattern, getSuggestedPatterns, saveHistory, toStoredTurns } from '../../lib/nexus/memory'
import { evaluateTriggersFiltered, getUptimeMs, STATS_POLL_MS } from '../../lib/nexus/proactiveAI'
import { startWakeWordDetection, stopWakeWordDetection } from '../../lib/nexus/wakeWord'
import { getGreeting } from '../../lib/nexus/personality'
import { speak, stopSpeaking } from '../../lib/nexus/tts'
import {
  getDailyUsage, incrementDailyUsage,
  getMonthlyUsage, incrementMonthlyUsage,
  DAILY_FREE_LIMIT, MONTHLY_PREMIUM_LIMIT,
  syncUsageFromServer, incrementServerUsage,
} from '../../lib/nexus/usageTracker'
import { NexusAvatar } from './NexusAvatar'
import { MessageBubble } from './MessageBubble'
import { VoiceButton } from './VoiceButton'
import { SendButton } from './SendButton'
import { QuickActions } from './QuickActions'
import { TypingIndicator } from './TypingIndicator'
import { PCStatusBar } from './PCStatusBar'
import { Marketplace } from '../Marketplace'
import { PersonaSwitcher } from '../PersonaSwitcher'
import { PaywallModal } from '../PaywallModal'
import type { Message, NexusStep, NexusEmotion } from '../../types/nexus'

/* ── 첨부 파일 타입 ── */
interface AttachedFile {
  name: string
  mimeType: string
  dataUrl: string
  text?: string
  size: number
  fileType: 'image' | 'video' | 'document' | 'spreadsheet' | 'other'
}

/* ── 페르소나 아이콘 맵 ── */
const PERSONA_META: Record<string, { emoji: string; name: string; color: string }> = {
  developer:  { emoji: '💻', name: '개발자',     color: '#6366f1' },
  marketer:   { emoji: '📣', name: '마케터',     color: '#f59e0b' },
  sales:      { emoji: '🤝', name: '영업',       color: '#10b981' },
  pm:         { emoji: '📋', name: 'PM',         color: '#0ea5e9' },
  designer:   { emoji: '🎨', name: '디자이너',   color: '#ec4899' },
  freelancer: { emoji: '🚀', name: '프리랜서',   color: '#8b5cf6' },
  legal:      { emoji: '⚖️', name: '법무',       color: '#f97316' },
  medical:    { emoji: '🏥', name: '의료',       color: '#06b6d4' },
  accountant: { emoji: '📊', name: '회계',       color: '#f59e0b' },
  creator:    { emoji: '🎬', name: '크리에이터', color: '#ef4444' },
  realtor:    { emoji: '🏠', name: '부동산',     color: '#22c55e' },
  teacher:    { emoji: '📚', name: '교사',       color: '#0ea5e9' },
  hr:         { emoji: '👥', name: 'HR',         color: '#8b5cf6' },
  engineer:   { emoji: '⚙️', name: '엔지니어',  color: '#10b981' },
  smallbiz:   { emoji: '🏪', name: '소상공인',   color: '#f97316' },
  corporate:  { emoji: '🏢', name: '기업/법인',  color: '#0ea5e9' },
  investor:   { emoji: '📈', name: '투자',       color: '#22c55e' },
  general:    { emoji: '🌟', name: '일반',       color: '#ec4899' },
  nexus:      { emoji: '🤖', name: 'Nexus',      color: '#7c3aed' },
}

/* ── Web Speech API 타입 (로컬 정의) ── */
interface SRResult { [i: number]: { transcript: string; confidence: number }; isFinal: boolean; length: number }
interface SRResultList { [i: number]: SRResult; length: number }
interface SREvent extends Event { results: SRResultList }
interface SRErrorEvent extends Event { error: string }
interface SRInstance {
  lang: string; continuous: boolean; interimResults: boolean
  onresult: ((e: SREvent) => void) | null
  onend: (() => void) | null
  onerror: ((e: SRErrorEvent) => void) | null
  start(): void; stop(): void; abort(): void
}
type SRConstructor = { new(): SRInstance }

interface ConversationTurn {
  role: 'user' | 'model'
  parts: Array<{ text: string }>
}


export function NexusChat() {
  const { assistantName, userName, userLang, subscriptionStatus, activePersonaId, setShowWorkflowBuilder } = useAppStore()

  const [showPersonaSwitcher, setShowPersonaSwitcher] = useState(false)
  const [showPaywall, setShowPaywall] = useState(false)
  const [dailyCount, setDailyCount]     = useState(() => getDailyUsage().count)
  const [monthlyCount, setMonthlyCount] = useState(() => getMonthlyUsage().count)

  const [messages, setMessages] = useState<Message[]>(() => [
    {
      id: '0',
      role: 'nexus',
      text: getGreeting(assistantName, userName, userLang),
      emotion: 'happy',
      timestamp: new Date(),
    },
  ])
  const [input, setInput] = useState('')
  const [typing, setTyping] = useState(false)
  const [agentProgress, setAgentProgress] = useState('')
  const [pendingAgentPlan, setPendingAgentPlan] = useState<{ plan: AgentPlan; userMessage: string } | null>(null)
  const [stepResults, setStepResults] = useState<Array<{ id: number; description: string; success: boolean; result: string }>>([])
  const [workflowSuggestion, setWorkflowSuggestion] = useState<string | null>(null)
  const [workflowInitialName, setWorkflowInitialName] = useState<string | undefined>(undefined)
  const pendingMsgRef = useRef('')
  const [emotion, setEmotion] = useState<NexusEmotion>('neutral')
  const [speaking, setSpeaking] = useState(false)
  const [listening, setListening] = useState(false)
  const [voiceInterim, setVoiceInterim] = useState('') // 실시간 음성 인식 중간 결과
  const [showMarketplace, setShowMarketplace] = useState(false)

  const [attachedFiles, setAttachedFiles] = useState<AttachedFile[]>([])
  const [fileLoading, setFileLoading] = useState(false)
  const [isDragOver, setIsDragOver] = useState(false)

  const messagesEndRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const historyRef = useRef<ConversationTurn[]>([])
  const voiceRecRef = useRef<SRInstance | null>(null)
  const typingRef = useRef(false) // send 중복 방지용 ref

  /* ── 파일 유형 감지 ── */
  const detectFileType = (mime: string, name: string): AttachedFile['fileType'] => {
    if (mime.startsWith('image/')) return 'image'
    if (mime.startsWith('video/')) return 'video'
    if (mime.includes('spreadsheet') || mime.includes('excel') || name.endsWith('.xlsx') || name.endsWith('.csv')) return 'spreadsheet'
    if (mime.includes('pdf') || mime.includes('word') || mime.includes('document') ||
        name.endsWith('.pdf') || name.endsWith('.docx') || name.endsWith('.doc') ||
        name.endsWith('.txt') || name.endsWith('.md')) return 'document'
    return 'other'
  }

  /* ── 파일 단건 읽기 ── */
  const readOneFile = useCallback(async (file: File): Promise<AttachedFile> => {
    const name = file.name
    const fileType = detectFileType(file.type, name)
    let dataUrl = ''
    let text: string | undefined
    try {
      if (fileType === 'image' || fileType === 'video') {
        dataUrl = await new Promise<string>(resolve => {
          const r = new FileReader(); r.onload = e => resolve(e.target?.result as string); r.readAsDataURL(file)
        })
      } else if (fileType === 'spreadsheet') {
        const arrayBuffer = await file.arrayBuffer()
        const XLSX = await import('xlsx')
        const workbook = XLSX.read(arrayBuffer, { type: 'array' })
        const lines: string[] = []
        workbook.SheetNames.forEach(sheetName => {
          const sheet = workbook.Sheets[sheetName]
          const csv = XLSX.utils.sheet_to_csv(sheet)
          if (csv.trim()) lines.push(`[시트: ${sheetName}]\n${csv}`)
        })
        text = lines.join('\n\n').slice(0, 12000)
        dataUrl = ''
      } else if (name.endsWith('.txt') || name.endsWith('.md') || name.endsWith('.csv') || name.endsWith('.json') || file.type.includes('text')) {
        text = await new Promise<string>(resolve => {
          const r = new FileReader(); r.onload = e => resolve(e.target?.result as string); r.readAsText(file, 'utf-8')
        })
      } else {
        dataUrl = await new Promise<string>(resolve => {
          const r = new FileReader(); r.onload = e => resolve(e.target?.result as string); r.readAsDataURL(file)
        })
      }
    } catch (err) {
      console.error('파일 읽기 오류:', err)
    }
    return { name, mimeType: file.type, dataUrl, text, size: file.size, fileType }
  }, [])

  /* ── 파일 선택/드롭 처리 ── */
  const handleFileSelect = useCallback(async (files: FileList | File[]) => {
    setFileLoading(true)
    const arr = Array.from(files).slice(0, 3)
    const settled = await Promise.allSettled(arr.map(readOneFile))
    const results = settled.filter(r => r.status === 'fulfilled').map(r => (r as PromiseFulfilledResult<AttachedFile>).value)
    setAttachedFiles(prev => [...prev, ...results].slice(0, 3))
    setFileLoading(false)
  }, [readOneFile])

  /* 언마운트 시 음성 중지 */
  useEffect(() => () => { stopSpeaking() }, [])

  /* 마운트 시 서버 사용량과 동기화 */
  useEffect(() => {
    syncUsageFromServer().then(data => {
      if (data) setDailyCount(data.used)
    })
  }, [])

  /* ── 능동형 모니터링 — PC 상태 폴링 후 자동 알림 ── */
  useEffect(() => {
    const poll = async () => {
      if (typingRef.current) return
      try {
        const res = await fetch('http://127.0.0.1:17891/api/stats/all', { signal: AbortSignal.timeout(4000) })
        if (!res.ok) return
        const stats = await res.json()
        const snapshot = { stats, uptimeMs: getUptimeMs() }
        const alert = evaluateTriggersFiltered(snapshot, 'ko', assistantName)
        if (alert) {
          setMessages(prev => [...prev, {
            id: Date.now().toString(),
            role: 'nexus',
            text: alert.message,
            emotion: alert.emotion as NexusEmotion,
            timestamp: new Date(),
            animate: true,
          }])
          speakText(alert.message)
        }
      } catch { /* 백엔드 미실행 시 무시 */ }
    }
    const timer = setInterval(poll, STATS_POLL_MS)
    return () => clearInterval(timer)
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [assistantName])

  /* 스크롤 하단 유지 */
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, typing])

  /* 웨이크워드 감지 — 마이크 버튼 누를 때 일시 정지됨 */
  useEffect(() => {
    const customLower = assistantName.toLowerCase()
    const wakeWords = [
      assistantName,
      `hey ${customLower}`,
      `헤이 ${customLower}`,
      '자비스', 'hey jarvis',
      'nexus', 'hey nexus', '넥서스', '헤이 넥서스',
    ].filter((w, i, arr) => arr.indexOf(w.toLowerCase()) === i)

    startWakeWordDetection(wakeWords, () => {
      /* 웨이크워드 감지 → 마이크 자동 시작 */
      handleVoiceToggle()
    })
    return () => stopWakeWordDetection()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [assistantName])

  /* ── TTS: 텍스트를 음성으로 읽기 (Pro만 OpenAI TTS, 무료는 Web Speech) ── */
  const speakText = useCallback((text: string) => {
    const isPro = subscriptionStatus === 'active' || subscriptionStatus === 'trial'
    speak(
      text,
      userLang,
      () => setSpeaking(true),
      () => setSpeaking(false),
      'neutral',
      undefined,
      isPro,
    )
  }, [userLang, subscriptionStatus])

  /* ── 에이전트 승인·취소 ── */
  const handleApproveAgent = useCallback(async () => {
    if (!pendingAgentPlan) return
    const { plan, userMessage } = pendingAgentPlan
    setPendingAgentPlan(null)
    setTyping(true)
    typingRef.current = true
    setStepResults([])
    setAgentProgress('⚙️ 실행 중...')
    try {
      const agentResult = await runAgentWithPlan(userMessage, plan,
        (msg) => setAgentProgress(msg),
        (step: AgentStep, success: boolean, result: string) => setStepResults(prev => [...prev, { id: step.id, description: step.description, success, result }]),
      )
      const card = buildResultCard(stepResults)
      const fullResult = agentResult + card
      setAgentProgress('')
      setStepResults([])
      setTyping(false)
      typingRef.current = false
      historyRef.current.push({ role: 'model', parts: [{ text: fullResult }] })
      learnFromTurn(userMessage, agentResult)
      learnPattern(userMessage, agentResult.slice(0, 60))
      const suggested = getSuggestedPatterns(userMessage)
      if (suggested.length > 0) {
        setWorkflowSuggestion(suggested[0].trigger)
        setWorkflowInitialName(`${suggested[0].trigger} 자동화`)
      }
      saveHistory(toStoredTurns(historyRef.current))
      setMessages(prev => [...prev, {
        id: (Date.now() + 1).toString(), role: 'nexus', text: fullResult,
        emotion: 'happy', timestamp: new Date(), animate: true,
      }])
      speakText(agentResult.slice(0, 200))
    } catch {
      setAgentProgress('')
      setTyping(false)
      typingRef.current = false
    }
  }, [pendingAgentPlan, speakText])

  const handleRejectAgent = useCallback(() => {
    setPendingAgentPlan(null)
    setStepResults([])
    setTyping(false)
    typingRef.current = false
    setMessages(prev => [...prev, {
      id: Date.now().toString(), role: 'nexus',
      text: '작업을 취소했어요. 다른 방식으로 도와드릴까요?',
      emotion: 'neutral', timestamp: new Date(), animate: true,
    }])
  }, [])

  /* ── STT: 마이크 버튼 토글 ── */
  const handleVoiceToggle = useCallback(() => {
    /* 이미 듣는 중이면 중지 */
    if (listening) {
      voiceRecRef.current?.stop()
      voiceRecRef.current = null
      setListening(false)
      setVoiceInterim('')
      /* 웨이크워드 감지 재시작 */
      startWakeWordDetection([], () => handleVoiceToggle())
      return
    }

    const win = window as unknown as Record<string, SRConstructor | undefined>
    const SR = win['SpeechRecognition'] ?? win['webkitSpeechRecognition']
    if (!SR) {
      alert('이 브라우저는 음성 인식을 지원하지 않습니다.\nChrome을 사용해주세요.')
      return
    }

    /* 웨이크워드 감지 일시 정지 (충돌 방지) */
    stopWakeWordDetection()

    const rec = new SR()
    voiceRecRef.current = rec
    rec.lang = userLang === 'ko' ? 'ko-KR' : 'en-US'
    rec.continuous = false
    rec.interimResults = true

    rec.onresult = (e: SREvent) => {
      let interim = ''
      let final = ''
      for (let i = 0; i < e.results.length; i++) {
        const t = e.results[i][0].transcript
        if (e.results[i].isFinal) final += t
        else interim += t
      }
      setVoiceInterim(interim)
      if (final) {
        setInput(final)
        setVoiceInterim('')
      }
    }

    rec.onend = () => {
      setListening(false)
      setVoiceInterim('')
      voiceRecRef.current = null
      /* 인식된 텍스트가 있으면 자동 전송 */
      setInput(prev => {
        if (prev.trim() && !typingRef.current) {
          const captured = prev.trim()
          setTimeout(() => void sendText(captured), 100)
          return ''
        }
        return prev
      })
      /* 웨이크워드 감지 재시작 */
      const customLower = assistantName.toLowerCase()
      startWakeWordDetection(
        [assistantName, `hey ${customLower}`, '자비스', 'nexus'],
        () => handleVoiceToggle(),
      )
    }

    rec.onerror = (e: SRErrorEvent) => {
      if (e.error !== 'aborted') console.warn('STT error:', e.error)
      setListening(false)
      setVoiceInterim('')
    }

    setListening(true)
    try { rec.start() } catch { setListening(false) }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [listening, userLang, assistantName])

  /* ── 에이전트 결과 카드 생성 ── */
  const buildResultCard = useCallback((
    steps: Array<{ description: string; success: boolean; result: string }>
  ): string => {
    if (steps.length === 0) return ''
    const lines = steps.map(s => {
      const icon = s.success ? '✅' : '❌'
      // 결과에서 핵심 정보 추출 (첫 줄 또는 80자)
      const detail = s.result.split('\n')[0].slice(0, 80)
      return `${icon} **${s.description}**${detail ? `\n   └ ${detail}` : ''}`
    })
    return `\n\n---\n📋 **실행 결과 요약**\n${lines.join('\n')}`
  }, [])

  /* ── 메시지 전송 ── */
  const sendText = useCallback(async (text: string) => {
    const trimmed = text.trim()
    if (!trimmed || typingRef.current) return

    const isFree = subscriptionStatus === 'none' || subscriptionStatus === 'expired'
    if (isFree && dailyCount >= DAILY_FREE_LIMIT) {
      setShowPaywall(true)
      return
    }

    typingRef.current = true

    const userMsg: Message = {
      id: Date.now().toString(),
      role: 'user',
      text: trimmed,
      timestamp: new Date(),
    }
    setMessages(prev => [...prev, userMsg])
    setInput('')
    setTyping(true)
    setEmotion('neutral')

    /* 서버 사용량 카운터 증가 (로컬스토리지 우회 방지) */
    if (isFree) {
      const { allowed, used } = await incrementServerUsage()
      setDailyCount(used)
      if (!allowed) {
        typingRef.current = false
        setTyping(false)
        setShowPaywall(true)
        return
      }
    } else {
      setDailyCount(incrementDailyUsage())
    }
    setMonthlyCount(incrementMonthlyUsage())

    historyRef.current.push({ role: 'user', parts: [{ text: trimmed }] })

    /* ── -1순위: 멀티스텝 에이전트 ── */
    if (isMultiStepTask(trimmed)) {
      setAgentProgress('🧠 작업 계획 수립 중...')
      try {
        const plan = await planAgent(trimmed)
        setAgentProgress('')

        if (hasDangerousSteps(plan)) {
          // 위험 스텝 포함 → 승인 먼저
          const stepList = plan.steps.map(s =>
            `${DANGEROUS_STEPS.has(s.type) ? '⚠️' : '✅'} ${s.description}`
          ).join('\n')
          const previewMsg = `다음 작업들을 실행할 예정이에요:\n\n${stepList}\n\n계속할까요?`
          historyRef.current.push({ role: 'model', parts: [{ text: previewMsg }] })
          setTyping(false)
          typingRef.current = false
          setPendingAgentPlan({ plan, userMessage: trimmed })
          setStepResults([])
          pendingMsgRef.current = trimmed
          setMessages(prev => [...prev, {
            id: (Date.now() + 1).toString(), role: 'nexus',
            text: previewMsg, emotion: 'neutral', timestamp: new Date(), animate: true,
          }])
          return
        }

        // 안전한 스텝만 → 바로 실행
        setAgentProgress('⚙️ 실행 중...')
        setStepResults([])
        const agentResult = await runAgentWithPlan(trimmed, plan,
          (msg) => setAgentProgress(msg),
          (step: AgentStep, success: boolean, result: string) => setStepResults(prev => [...prev, { id: step.id, description: step.description, success, result }]),
        )
        const card = buildResultCard(stepResults)
        const fullResult = agentResult + card
        setAgentProgress('')
        setStepResults([])
        setTyping(false)
        typingRef.current = false
        historyRef.current.push({ role: 'model', parts: [{ text: fullResult }] })
        learnFromTurn(trimmed, agentResult)
        learnPattern(trimmed, agentResult.slice(0, 60))
        const suggestedB = getSuggestedPatterns(trimmed)
        if (suggestedB.length > 0) {
          setWorkflowSuggestion(suggestedB[0].trigger)
          setWorkflowInitialName(`${suggestedB[0].trigger} 자동화`)
        }
        saveHistory(toStoredTurns(historyRef.current))
        setMessages(prev => [...prev, {
          id: (Date.now() + 1).toString(), role: 'nexus', text: fullResult,
          emotion: 'happy', timestamp: new Date(), animate: true,
        }])
        speakText(agentResult.slice(0, 200))
        return
      } catch {
        setAgentProgress('')
        // 에이전트 실패 시 일반 LLM으로 폴백
      }
    }

    /* ── 0순위: LLM Tool Router — 멀티툴 인텐트 감지 ── */
    const routerHistory = historyRef.current.slice(-6).map(h => ({
      role: (h.role === 'user' ? 'user' : 'assistant') as 'user' | 'assistant',
      content: h.parts[0]?.text ?? '',
    })).filter(h => h.content.length > 0)

    try {
      const toolCalls = await routeWithLLMMulti(trimmed, routerHistory)
      const actionable = toolCalls.filter(t => t.tool !== 'general_answer')

      if (actionable.length >= 2) {
        /* 복합 툴 2개 이상 → agentExecutor가 순서대로 전부 실행 */
        setAgentProgress('🧠 복합 작업 계획 중...')
        try {
          const agentResult = await runAgent(trimmed, (msg) => setAgentProgress(msg))
          setAgentProgress('')
          setTyping(false)
          typingRef.current = false
          historyRef.current.push({ role: 'model', parts: [{ text: agentResult }] })
          learnFromTurn(trimmed, agentResult)
          saveHistory(toStoredTurns(historyRef.current))
          setMessages(prev => [...prev, {
            id: (Date.now() + 1).toString(), role: 'nexus', text: agentResult,
            emotion: 'happy', timestamp: new Date(), animate: true,
          }])
          speakText(agentResult.slice(0, 200))
          return
        } catch { setAgentProgress('') }
      } else if (actionable.length === 1) {
        /* 단일 툴 → 위젯 안내 힌트 */
        const toolLabels: Record<string, string> = {
          pc_status: 'PC 상태 조회', security_scan: '보안 스캔', clean: 'PC 정리',
          full_scan: '전체 진단', launch_app: '앱 실행', volume_control: '볼륨 조절',
          brightness: '밝기 조절', wifi_toggle: 'Wi-Fi 제어', power_action: '전원 제어',
          file_search: '파일 검색', weather: '날씨 조회', calendar_today: '일정 조회',
          email_inbox: '메일 확인', vision_screen: '화면 분석', meeting_start: '회의 녹음',
          workflow_run: '워크플로 실행', briefing_now: '브리핑', translator: '번역',
        }
        const label = toolLabels[actionable[0].tool] ?? actionable[0].tool
        const hint = `⚡ **${label}** 기능이에요.\n\n이 기능은 Nexus 캐릭터 창(우측 위젯)에서 실행할 수 있어요. 위젯에서 같은 말을 해보세요!`
        historyRef.current.push({ role: 'model', parts: [{ text: hint }] })
        setTyping(false)
        typingRef.current = false
        setMessages(prev => [...prev, {
          id: (Date.now() + 1).toString(), role: 'nexus', text: hint,
          emotion: 'happy', timestamp: new Date(), animate: true,
        }])
        speakText(hint)
        return
      }
    } catch { /* LLM 라우터 실패 시 일반 LLM으로 진행 */ }

    const apiKey = localStorage.getItem('nexus-gemini-key') ?? ''
    let response

    /* 장기 기억 컨텍스트 주입 */
    const memCtx = buildMemoryContext()
    const promptWithMemory = memCtx ? `${memCtx}\n\n---\n${trimmed}` : trimmed

    /* 1순위: Ollama */
    try {
      const r = await callOllama(promptWithMemory, historyRef.current.slice(-10))
      if (r) response = r
    } catch { /* Ollama 미실행 */ }

    /* 2순위: Gemini (Pro만 GPT-4o, 무료는 Perplexity) */
    if (!response && apiKey && trackUsage()) {
      const isPro = subscriptionStatus === 'active' || subscriptionStatus === 'trial'
      try { response = await callGemini(apiKey, promptWithMemory, historyRef.current.slice(-10), isPro) }
      catch { /* Gemini 실패 */ }
    }

    /* 3순위: 스마트 폴백 */
    if (!response) response = fallbackResponse(trimmed, assistantName)

    historyRef.current.push({ role: 'model', parts: [{ text: response.text }] })
    learnFromTurn(trimmed, response.text)
    saveHistory(toStoredTurns(historyRef.current))

    const pendingSteps = response.steps.filter(s => s.confirmRequired)
    const autoSteps = response.steps.filter(s => !s.confirmRequired)

    const nexusMsg: Message = {
      id: (Date.now() + 1).toString(),
      role: 'nexus',
      text: response.text,
      emotion: response.emotion ?? 'neutral',
      timestamp: new Date(),
      steps: response.steps,
      pendingSteps: pendingSteps.length > 0 ? pendingSteps : undefined,
      actionDone: autoSteps.length > 0,
      animate: true,
    }

    setTyping(false)
    typingRef.current = false
    setEmotion(response.emotion ?? 'neutral')
    setMessages(prev => [...prev, nexusMsg])

    /* ── TTS: 응답을 음성으로 읽기 ── */
    speakText(response.text)
  }, [assistantName, speakText])

  /* ── 파일 첨부 전송 ── */
  const handleSendWithFile = useCallback(async (text: string, files: AttachedFile[]) => {
    if (files.length === 0) return
    const file = files[0]
    const icon = file.fileType === 'image' ? '🖼️' : file.fileType === 'video' ? '🎬' : file.fileType === 'spreadsheet' ? '📊' : '📄'
    const displayNames = files.map(f => f.name).join(', ')

    setMessages(prev => [
      ...prev,
      {
        id: Date.now().toString(),
        role: 'user',
        text: `${icon} ${displayNames}${text ? '\n' + text : ''}`,
        timestamp: new Date(),
        imageDataUrl: file.fileType === 'image' ? file.dataUrl : undefined,
      },
    ])
    setAttachedFiles([])
    setInput('')
    setTyping(true)

    let result = ''
    try {
      if (file.fileType === 'image') {
        /* 이미지 → Groq Vision */
        const base64 = file.dataUrl.split(',')[1] ?? file.dataUrl
        const question = text || '이 이미지를 분석해줘. 내용, 특징, 시사점을 상세하게 설명해줘.'
        result = await callGroqVision(base64, question)

      } else if (file.fileType === 'video') {
        /* 동영상 → 편집 또는 분석 */
        const wantTrim     = /잘라|자르기|trim|구간|초부터|분부터|까지|처음.*분|처음.*초/i.test(text)
        const wantCompress = /압축|용량.*줄|줄여|compress|가볍게|작게/i.test(text)
        const wantSpeed    = /배속|빠르게|느리게|speed|빨리|천천히/i.test(text)
        const wantSubtitle = /자막.*합쳐|합쳐.*자막|자막.*삽입|삽입.*자막|자막.*넣어|자막.*태워|srt.*합쳐|합쳐.*srt|subtitle.*merge|merge.*subtitle|burn.*subtitle|subtitle.*burn/i.test(text)
        const needsEdit    = wantTrim || wantCompress || wantSpeed || wantSubtitle

        if (needsEdit) {
          const opLabel = wantTrim ? '구간 자르기' : wantCompress ? '용량 압축' : wantSpeed ? '속도 변환' : '자막 삽입'
          const opKey   = wantTrim ? 'video_trim'  : wantCompress ? 'video_compress' : wantSpeed ? 'video_speed' : 'video_subtitle'
          setMessages(prev => prev.map((m, i) =>
            i === prev.length - 1 ? { ...m, text: `🎬 ${file.name}\n⏳ ${opLabel} 중...` } : m
          ))
          const allFiles = [file, ...files.slice(1)]
          const res = await fetch('http://127.0.0.1:17891/api/file/process', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json', ...await getAuthHeader() },
            body: JSON.stringify({
              files: allFiles.map(f => ({ name: f.name, mime_type: f.mimeType, data: f.dataUrl })),
              operation: opKey,
              query: text,
              params: {},
            }),
          }).then(r => r.json()).catch(() => ({ success: false, message: '영상 편집 실패' }))
          if (res.success && res.data && res.file_name) {
            const mimeType = res.mime_type ?? 'video/mp4'
            const bytes = Uint8Array.from(atob(res.data), c => c.charCodeAt(0))
            const blob = new Blob([bytes], { type: mimeType })
            const url = URL.createObjectURL(blob)
            result = res.message ?? `${opLabel} 완료!`
            setMessages(prev => [...prev, {
              id: Date.now().toString(), role: 'nexus', text: result, timestamp: new Date(),
              inlineCard2: { type: 'file_result', data: { fileName: res.file_name, url, mimeType, operation: res.operation } } as any,
            }])
            setTyping(false)
            return
          }
          result = `❌ ${res.message ?? opLabel + ' 실패'}`
        } else {
          /* 내용 분석 → Whisper 전사 */
          setMessages(prev => prev.map((m, i) =>
            i === prev.length - 1 ? { ...m, text: `🎬 ${file.name}\n⏳ 영상 분석 중... (${(file.size / 1024 / 1024).toFixed(1)}MB)\n음성을 텍스트로 전사하고 있어요.` } : m
          ))
          const depsCheck = await fetch('http://127.0.0.1:17891/api/video/check-deps')
            .then(r => r.json()).catch(() => null)
          if (depsCheck && !depsCheck.ready) {
            result = `⚠️ **영상 분석 불가**\n\n${depsCheck.message ?? '영상 분석 도구가 설치되지 않았습니다.'}`
          } else {
            const res = await fetch('http://127.0.0.1:17891/api/video/analyze-file', {
              method: 'POST',
              headers: { 'Content-Type': 'application/json', ...await getAuthHeader() },
              body: JSON.stringify({
                file_data: file.dataUrl,
                file_name: file.name,
                lang: userLang,
                query: text || '이 영상 내용을 요약해줘',
              }),
            }).then(r => r.json()).catch(() => ({ success: false, message: '영상 분석 요청 실패' }))
            result = res.message ?? (res.success ? '영상 분석 완료' : '영상 분석에 실패했습니다.')
          }
        }

      } else {
        /* 문서/스프레드시트 → 텍스트 추출 후 LLM */
        let docContent = file.text ?? ''

        /* 백엔드 업로드로 더 정확한 텍스트 추출 시도 */
        if (file.fileType !== 'other' && file.dataUrl) {
          try {
            const { uploadDocFile } = await import('../../lib/nexus/docEditor')
            const resp = await fetch(file.dataUrl)
            const blob = await resp.blob()
            const fileObj = new File([blob], file.name, { type: file.mimeType })
            const uploaded = await uploadDocFile(fileObj)
            if (uploaded.success && uploaded.preview?.text) {
              docContent = uploaded.preview.text
            } else if (uploaded.success && uploaded.preview?.rows) {
              docContent = (uploaded.preview.rows as string[][]).map(r => r.join('\t')).join('\n')
            }
          } catch { /* 백엔드 없으면 기존 file.text 사용 */ }
        }

        const truncated = docContent.slice(0, 8000)
        const question = text || '이 문서의 핵심 내용, 중요 데이터, 인사이트를 정리해줘.'
        const prompt = truncated
          ? `[첨부 파일: ${file.name}]\n\n${truncated}\n\n---\n사용자 질문: ${question}`
          : `[첨부 파일: ${file.name} — 텍스트 추출 불가]\n\n사용자 질문: ${question}`

        const pplxKey = localStorage.getItem('nexus-pplx-key') ?? ''
        const openaiKey = localStorage.getItem('nexus-openai-key') ?? ''
        const geminiKey = localStorage.getItem('nexus-gemini-key') ?? ''

        if (pplxKey) {
          const res = await fetch('https://api.perplexity.ai/chat/completions', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${pplxKey}` },
            body: JSON.stringify({ model: 'sonar-pro', messages: [{ role: 'user', content: prompt }], max_tokens: 2000 }),
            signal: AbortSignal.timeout(30000),
          })
          if (res.ok) {
            const d = await res.json() as { choices?: Array<{ message?: { content?: string } }> }
            result = d.choices?.[0]?.message?.content?.trim() ?? ''
          }
        } else if (openaiKey) {
          const res = await fetch('https://api.openai.com/v1/chat/completions', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${openaiKey}` },
            body: JSON.stringify({ model: 'gpt-4o', messages: [{ role: 'user', content: prompt }], max_tokens: 2000 }),
            signal: AbortSignal.timeout(30000),
          })
          if (res.ok) {
            const d = await res.json() as { choices?: Array<{ message?: { content?: string } }> }
            result = d.choices?.[0]?.message?.content?.trim() ?? ''
          }
        } else if (geminiKey && trackUsage()) {
          result = (await callGemini(geminiKey, prompt, [])).text
        } else {
          try {
            const res = await fetch('http://127.0.0.1:17891/api/llm/chat', {
              method: 'POST',
              headers: { 'Content-Type': 'application/json', ...await getAuthHeader() },
              body: JSON.stringify({ messages: [{ role: 'user', content: prompt }], max_tokens: 2000 }),
            }).then(r => r.json()).catch(() => null)
            result = res?.content ?? '⚠️ API 키가 없습니다. 설정에서 API 키를 입력해주세요.'
          } catch {
            result = '⚠️ API 키가 없습니다. 설정에서 Perplexity 또는 Gemini API 키를 입력해주세요.'
          }
        }
      }
    } catch (e) {
      result = `파일 처리 중 오류: ${e instanceof Error ? e.message : String(e)}`
    }

    setMessages(prev => [
      ...prev,
      {
        id: (Date.now() + 1).toString(),
        role: 'nexus',
        text: result,
        emotion: 'happy',
        timestamp: new Date(),
        animate: true,
        fileInfo: { name: file.name, type: file.fileType },
      },
    ])
    setTyping(false)
    speakText(result)
  }, [userLang, speakText])

  const send = useCallback((text: string) => {
    if (attachedFiles.length > 0) {
      void handleSendWithFile(text, attachedFiles)
    } else {
      void sendText(text)
    }
  }, [sendText, handleSendWithFile, attachedFiles])

  const handleStepConfirm = useCallback((step: NexusStep, msgId: string) => {
    setMessages(prev =>
      prev.map(m =>
        m.id === msgId
          ? { ...m, pendingSteps: m.pendingSteps?.filter(s => s !== step), actionDone: true }
          : m
      )
    )
  }, [])

  const isFirstMessage = messages.length <= 1
  const displayInput = voiceInterim || input



  return (
    <div
      onDragOver={e => { e.preventDefault(); setIsDragOver(true) }}
      onDragLeave={() => setIsDragOver(false)}
      onDrop={e => {
        e.preventDefault()
        setIsDragOver(false)
        if (e.dataTransfer.files.length > 0) void handleFileSelect(e.dataTransfer.files)
      }}
      style={{
        flex: 1,
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden',
        background: 'var(--bg-base)',
        outline: isDragOver ? '2px dashed var(--accent-primary)' : 'none',
        outlineOffset: -2,
      }}
    >
      {/* 상단 PC 상태 바 */}
      <div
        style={{
          padding: '6px 16px',
          borderBottom: '1px solid var(--border-subtle)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          flexShrink: 0,
          background: 'var(--bg-surface)',
        }}
      >
        <PCStatusBar />
        <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          {/* 사용량 배지 */}
          {(() => {
            const isFree = subscriptionStatus === 'none' || subscriptionStatus === 'expired'
            if (isFree) {
              const remaining = Math.max(0, DAILY_FREE_LIMIT - dailyCount)
              const color = remaining <= 5 ? '#ef4444' : remaining <= 10 ? '#f59e0b' : '#22c55e'
              return (
                <div style={{ display: 'flex', alignItems: 'center', gap: 3, fontSize: 11 }}>
                  <span style={{ color: 'var(--text-muted)' }}>오늘</span>
                  <span style={{ color, fontWeight: 700 }}>{remaining}</span>
                  <span style={{ color: 'var(--text-muted)' }}>/{DAILY_FREE_LIMIT}회</span>
                </div>
              )
            }
            const pct = Math.min(100, Math.round((monthlyCount / MONTHLY_PREMIUM_LIMIT) * 100))
            const color = pct >= 90 ? '#ef4444' : pct >= 70 ? '#f59e0b' : 'var(--accent-primary)'
            return (
              <div style={{ display: 'flex', alignItems: 'center', gap: 3, fontSize: 11 }}>
                <span style={{ color: 'var(--text-muted)' }}>이번달</span>
                <span style={{ color, fontWeight: 700 }}>{monthlyCount.toLocaleString()}</span>
                <span style={{ color: 'var(--text-muted)' }}>/{MONTHLY_PREMIUM_LIMIT.toLocaleString()}</span>
              </div>
            )
          })()}
          {/* 페르소나 칩 */}
          {(() => {
            const meta = PERSONA_META[activePersonaId] ?? PERSONA_META['nexus']
            return (
              <button
                onClick={() => setShowPersonaSwitcher(v => !v)}
                title="AI 모드 변경"
                style={{
                  display: 'flex', alignItems: 'center', gap: 4,
                  padding: '3px 8px', borderRadius: 20,
                  border: `1px solid ${meta.color}55`,
                  background: `${meta.color}18`,
                  color: meta.color, fontSize: 11, fontWeight: 600,
                  cursor: 'pointer', transition: 'all 0.15s',
                }}
                onMouseEnter={e => { e.currentTarget.style.background = `${meta.color}30` }}
                onMouseLeave={e => { e.currentTarget.style.background = `${meta.color}18` }}
              >
                <span style={{ fontSize: 13 }}>{meta.emoji}</span>
                <span>{meta.name}</span>
              </button>
            )
          })()}
        </div>
      </div>

      {/* 메시지 영역 */}
      <div
        style={{
          flex: 1,
          overflowY: 'auto',
          padding: '16px 16px 8px',
          display: 'flex',
          flexDirection: 'column',
          scrollbarWidth: 'thin',
        }}
      >
        <AnimatePresence>
          {isFirstMessage && (
            <motion.div
              initial={{ opacity: 0, y: -10 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, scale: 0.8 }}
              style={{
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                padding: '24px 0 20px',
                gap: 10,
              }}
            >
              <NexusAvatar emotion={emotion} speaking={speaking} listening={listening} />
              <motion.div
                animate={{ opacity: [0.5, 1, 0.5] }}
                transition={{ duration: 2, repeat: Infinity, ease: 'easeInOut' }}
                style={{ fontSize: 12, color: 'var(--text-muted)', letterSpacing: '0.05em', fontWeight: 600 }}
              >
                {speaking
                  ? (userLang === 'ko' ? '말하는 중...' : 'Speaking...')
                  : listening
                  ? (userLang === 'ko' ? '듣고 있어요...' : 'Listening...')
                  : assistantName.toUpperCase()}
              </motion.div>
            </motion.div>
          )}
        </AnimatePresence>

        {messages.map(msg => (
          <MessageBubble key={msg.id} message={msg} onStepConfirm={handleStepConfirm} />
        ))}

        {typing && <TypingIndicator message={
          stepResults.length > 0
            ? stepResults.map(s => `${s.success ? '✅' : '❌'} ${s.description}`).join('\n') + (agentProgress ? `\n⏳ ${agentProgress}` : '')
            : agentProgress || undefined
        } />}
        <div ref={messagesEndRef} />
      </div>

      {/* 에이전트 승인 UI */}
      {pendingAgentPlan && (
        <div style={{ padding: '8px 16px', display: 'flex', gap: 8 }}>
          <button
            onClick={handleApproveAgent}
            style={{ flex: 1, padding: '10px 0', background: '#7c3aed', color: '#fff', border: 'none', borderRadius: 10, fontWeight: 700, fontSize: 14, cursor: 'pointer' }}
          >
            ✅ 실행할게요
          </button>
          <button
            onClick={handleRejectAgent}
            style={{ flex: 1, padding: '10px 0', background: '#374151', color: '#fff', border: 'none', borderRadius: 10, fontWeight: 700, fontSize: 14, cursor: 'pointer' }}
          >
            ❌ 취소
          </button>
        </div>
      )}

      {/* 워크플로우 자동화 제안 */}
      {workflowSuggestion && !pendingAgentPlan && (
        <motion.div
          initial={{ opacity: 0, y: 8 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0 }}
          style={{ padding: '6px 16px', display: 'flex', alignItems: 'center', gap: 8, background: 'rgba(245,158,11,0.08)', borderTop: '1px solid rgba(245,158,11,0.2)' }}
        >
          <span style={{ fontSize: 13 }}>⚡</span>
          <span style={{ fontSize: 12, color: 'rgba(255,255,255,0.6)', flex: 1 }}>
            "{workflowSuggestion}" 작업, 자동화할까요?
          </span>
          <button onClick={() => { setShowWorkflowBuilder(true, workflowInitialName); setWorkflowSuggestion(null) }} style={{ padding: '4px 12px', borderRadius: 8, border: 'none', background: 'rgba(245,158,11,0.25)', color: '#f59e0b', fontSize: 11, fontWeight: 700, cursor: 'pointer' }}>
            워크플로우로 만들기
          </button>
          <button onClick={() => setWorkflowSuggestion(null)} style={{ background: 'none', border: 'none', color: 'rgba(255,255,255,0.3)', cursor: 'pointer', fontSize: 14, padding: '0 4px' }}>✕</button>
        </motion.div>
      )}

      {/* 퀵 액션 */}
      <AnimatePresence>
        {isFirstMessage && (
          <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}>
            <QuickActions onSelect={cmd => send(cmd)} showFeatured />
          </motion.div>
        )}
        {!isFirstMessage && (
          <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}>
            <QuickActions onSelect={cmd => send(cmd)} />
          </motion.div>
        )}
      </AnimatePresence>

      {/* 첨부 파일 미리보기 */}
      {attachedFiles.length > 0 && (
        <div style={{
          padding: '6px 12px',
          borderTop: '1px solid var(--border-subtle)',
          display: 'flex', gap: 6, flexWrap: 'wrap',
          background: 'var(--bg-surface)',
        }}>
          {attachedFiles.map((f, i) => (
            <div key={i} style={{
              display: 'flex', alignItems: 'center', gap: 4,
              padding: '3px 8px', borderRadius: 20,
              background: 'var(--glass-bg)',
              border: '1px solid var(--border-default)',
              fontSize: 12, color: 'var(--text-secondary)',
              maxWidth: 200,
            }}>
              <span>{f.fileType === 'image' ? '🖼️' : f.fileType === 'video' ? '🎬' : f.fileType === 'spreadsheet' ? '📊' : '📄'}</span>
              <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{f.name}</span>
              <span style={{ fontSize: 10, color: 'var(--text-muted)', flexShrink: 0 }}>
                {(f.size / 1024).toFixed(0)}KB
              </span>
              <button
                onClick={() => setAttachedFiles(prev => prev.filter((_, idx) => idx !== i))}
                style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--text-muted)', padding: 0, fontSize: 12, lineHeight: 1, flexShrink: 0 }}
              >✕</button>
            </div>
          ))}
          {fileLoading && <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>읽는 중...</span>}
        </div>
      )}

      {/* 입력 바 */}
      <div
        style={{
          padding: '10px 12px',
          borderTop: attachedFiles.length === 0 ? '1px solid var(--border-subtle)' : 'none',
          display: 'flex',
          alignItems: 'center',
          gap: 8,
          flexShrink: 0,
          background: 'var(--bg-surface)',
        }}
      >
        {/* 숨김 파일 입력 */}
        <input
          ref={fileInputRef}
          type="file"
          multiple
          accept="image/*,video/*,.pdf,.docx,.doc,.txt,.md,.xlsx,.xls,.csv,.json"
          style={{ display: 'none' }}
          onChange={e => { if (e.target.files) { void handleFileSelect(e.target.files); e.target.value = '' } }}
        />

        <VoiceButton listening={listening} onToggle={handleVoiceToggle} />

        {/* 파일 첨부 버튼 */}
        <button
          onClick={() => fileInputRef.current?.click()}
          title="파일 첨부 (이미지/동영상/문서)"
          style={{
            flexShrink: 0,
            width: 36, height: 36,
            borderRadius: '50%',
            border: `1px solid ${attachedFiles.length > 0 ? 'var(--accent-primary)' : 'var(--border-default)'}`,
            background: attachedFiles.length > 0 ? 'rgba(99,102,241,0.15)' : 'var(--glass-bg)',
            color: attachedFiles.length > 0 ? 'var(--accent-primary)' : 'var(--text-secondary)',
            fontSize: 16,
            cursor: 'pointer',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            transition: 'all 0.15s',
          }}
          onMouseEnter={e => { e.currentTarget.style.borderColor = 'var(--accent-primary)'; e.currentTarget.style.color = 'var(--accent-primary)' }}
          onMouseLeave={e => {
            if (attachedFiles.length === 0) {
              e.currentTarget.style.borderColor = 'var(--border-default)'
              e.currentTarget.style.color = 'var(--text-secondary)'
            }
          }}
        >📎</button>

        <button
          onClick={() => setShowMarketplace(true)}
          title="워크플로우 마켓플레이스"
          style={{
            flexShrink: 0,
            width: 36, height: 36,
            borderRadius: '50%',
            border: '1px solid var(--border-default)',
            background: 'var(--glass-bg)',
            color: 'var(--text-secondary)',
            fontSize: 16,
            cursor: 'pointer',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            transition: 'all 0.15s',
          }}
          onMouseEnter={e => { e.currentTarget.style.borderColor = 'var(--accent-primary)'; e.currentTarget.style.color = 'var(--accent-primary)' }}
          onMouseLeave={e => { e.currentTarget.style.borderColor = 'var(--border-default)'; e.currentTarget.style.color = 'var(--text-secondary)' }}
        >🛒</button>
        <input
          ref={inputRef}
          value={displayInput}
          onChange={e => { setInput(e.target.value); setVoiceInterim('') }}
          onKeyDown={e => {
            if (e.key === 'Enter' && !e.shiftKey) {
              e.preventDefault()
              send(input)
            }
          }}
          placeholder={
            attachedFiles.length > 0
              ? (userLang === 'ko' ? '파일에 대해 질문하세요...' : 'Ask about the file...')
              : listening
              ? (userLang === 'ko' ? '말씀하세요...' : 'Speak now...')
              : userLang === 'ko'
              ? `${assistantName}에게 말해보세요...`
              : `Ask ${assistantName} anything...`
          }
          readOnly={listening && !voiceInterim}
          style={{
            flex: 1,
            background: listening ? 'rgba(239,68,68,0.08)' : 'var(--bg-elevated)',
            border: `1px solid ${listening ? 'rgba(239,68,68,0.5)' : 'var(--border-default)'}`,
            borderRadius: 20,
            padding: '10px 16px',
            color: voiceInterim ? 'var(--text-muted)' : 'var(--text-primary)',
            fontSize: 14,
            outline: 'none',
            fontFamily: 'Pretendard, Inter, sans-serif',
            transition: 'all 0.2s',
            fontStyle: voiceInterim ? 'italic' : 'normal',
          }}
          onFocus={e => { if (!listening) e.currentTarget.style.borderColor = 'var(--accent-primary)' }}
          onBlur={e => { if (!listening) e.currentTarget.style.borderColor = 'var(--border-default)' }}
        />
        <SendButton onClick={() => send(input)} disabled={(!input.trim() && attachedFiles.length === 0) || typing} />
      </div>

      {showMarketplace && <Marketplace onClose={() => setShowMarketplace(false)} />}

      <AnimatePresence>
        {showPersonaSwitcher && (
          <PersonaSwitcher onClose={() => setShowPersonaSwitcher(false)} />
        )}
      </AnimatePresence>

      {showPaywall && (
        <PaywallModal
          feature="ai_request"
          used={dailyCount}
          limit={DAILY_FREE_LIMIT}
          onClose={() => setShowPaywall(false)}
        />
      )}
    </div>
  )
}
