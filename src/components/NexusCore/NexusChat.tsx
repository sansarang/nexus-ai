import { useState, useRef, useEffect, useCallback } from 'react'
import { AnimatePresence, motion } from 'framer-motion'
import { useAppStore } from '../../stores/appStore'
import { callGemini, callOllama, fallbackResponse, trackUsage } from '../../lib/nexus/gemini_engine'
import { startWakeWordDetection, stopWakeWordDetection } from '../../lib/nexus/wakeWord'
import { getGreeting } from '../../lib/nexus/personality'
import { speak, stopSpeaking } from '../../lib/nexus/tts'
import {
  getDailyUsage, incrementDailyUsage,
  getMonthlyUsage, incrementMonthlyUsage,
  DAILY_FREE_LIMIT, MONTHLY_PREMIUM_LIMIT,
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
import type { Message, NexusStep, NexusEmotion } from '../../types/nexus'

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
  const { assistantName, userName, userLang, subscriptionStatus, activePersonaId } = useAppStore()

  const [showPersonaSwitcher, setShowPersonaSwitcher] = useState(false)
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
  const [emotion, setEmotion] = useState<NexusEmotion>('neutral')
  const [speaking, setSpeaking] = useState(false)
  const [listening, setListening] = useState(false)
  const [voiceInterim, setVoiceInterim] = useState('') // 실시간 음성 인식 중간 결과
  const [showMarketplace, setShowMarketplace] = useState(false)

  const messagesEndRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)
  const historyRef = useRef<ConversationTurn[]>([])
  const voiceRecRef = useRef<SRInstance | null>(null)
  const typingRef = useRef(false) // send 중복 방지용 ref

  /* 언마운트 시 음성 중지 */
  useEffect(() => () => { stopSpeaking() }, [])

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

  /* ── TTS: 텍스트를 음성으로 읽기 ── */
  const speakText = useCallback((text: string) => {
    speak(
      text,
      userLang,
      () => setSpeaking(true),
      () => setSpeaking(false),
    )
  }, [userLang])

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

  /* ── 메시지 전송 ── */
  const sendText = useCallback(async (text: string) => {
    const trimmed = text.trim()
    if (!trimmed || typingRef.current) return
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

    /* 사용량 카운터 증가 */
    setDailyCount(incrementDailyUsage())
    setMonthlyCount(incrementMonthlyUsage())

    historyRef.current.push({ role: 'user', parts: [{ text: trimmed }] })

    const apiKey = localStorage.getItem('nexus-gemini-key') ?? ''
    let response

    /* 1순위: Ollama */
    try {
      const r = await callOllama(trimmed, historyRef.current.slice(-10))
      if (r) response = r
    } catch { /* Ollama 미실행 */ }

    /* 2순위: Gemini */
    if (!response && apiKey && trackUsage()) {
      try { response = await callGemini(apiKey, trimmed, historyRef.current.slice(-10)) }
      catch { /* Gemini 실패 */ }
    }

    /* 3순위: 스마트 폴백 */
    if (!response) response = fallbackResponse(trimmed, assistantName)

    historyRef.current.push({ role: 'model', parts: [{ text: response.text }] })

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
    }

    setTyping(false)
    typingRef.current = false
    setEmotion(response.emotion ?? 'neutral')
    setMessages(prev => [...prev, nexusMsg])

    /* ── TTS: 응답을 음성으로 읽기 ── */
    speakText(response.text)
  }, [assistantName, speakText])

  const send = useCallback((text: string) => {
    void sendText(text)
  }, [sendText])

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
      style={{
        flex: 1,
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden',
        background: 'var(--bg-base)',
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

        {typing && <TypingIndicator />}
        <div ref={messagesEndRef} />
      </div>

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

      {/* 입력 바 */}
      <div
        style={{
          padding: '10px 12px',
          borderTop: '1px solid var(--border-subtle)',
          display: 'flex',
          alignItems: 'center',
          gap: 8,
          flexShrink: 0,
          background: 'var(--bg-surface)',
        }}
      >
        <VoiceButton listening={listening} onToggle={handleVoiceToggle} />
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
            listening
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
        <SendButton onClick={() => send(input)} disabled={!input.trim() || typing} />
      </div>

      {showMarketplace && <Marketplace onClose={() => setShowMarketplace(false)} />}

      <AnimatePresence>
        {showPersonaSwitcher && (
          <PersonaSwitcher onClose={() => setShowPersonaSwitcher(false)} />
        )}
      </AnimatePresence>
    </div>
  )
}
