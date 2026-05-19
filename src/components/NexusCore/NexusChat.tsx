import { useState, useRef, useEffect, useCallback } from 'react'
import { AnimatePresence, motion } from 'framer-motion'
import { useAppStore } from '../../stores/appStore'
import { callGemini, callOllama, fallbackResponse, trackUsage } from '../../lib/nexus/gemini_engine'
import { startWakeWordDetection, stopWakeWordDetection } from '../../lib/nexus/wakeWord'
import { getGreeting } from '../../lib/nexus/personality'
import { speak, stopSpeaking } from '../../lib/nexus/tts'
import { NexusAvatar } from './NexusAvatar'
import { MessageBubble } from './MessageBubble'
import { VoiceButton } from './VoiceButton'
import { SendButton } from './SendButton'
import { QuickActions } from './QuickActions'
import { TypingIndicator } from './TypingIndicator'
import { PCStatusBar } from './PCStatusBar'
import type { Message, NexusStep, NexusEmotion } from '../../types/nexus'

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
  const { assistantName, userName, userLang } = useAppStore()

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
          setTimeout(() => {
            setInput(cur => {
              if (cur.trim()) void sendText(cur)
              return ''
            })
          }, 100)
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
        <span style={{ fontSize: 11, color: 'var(--text-muted)', fontWeight: 500 }}>
          {userLang === 'ko' ? `${assistantName} · AI 비서` : `${assistantName} · AI Assistant`}
        </span>
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
    </div>
  )
}
