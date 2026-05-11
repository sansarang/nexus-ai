import { useState, useRef, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { useAppStore } from '../../stores/appStore'

interface SpeechRecognitionEvent extends Event {
  results: SpeechRecognitionResultList
  resultIndex: number
}

interface SpeechRecognitionErrorEvent extends Event {
  error: string
}

interface SpeechRecognitionInstance extends EventTarget {
  lang: string
  interimResults: boolean
  continuous: boolean
  start: () => void
  stop: () => void
  onresult: ((e: SpeechRecognitionEvent) => void) | null
  onerror: ((e: SpeechRecognitionErrorEvent) => void) | null
  onend: (() => void) | null
}

declare global {
  interface Window {
    SpeechRecognition?: new () => SpeechRecognitionInstance
    webkitSpeechRecognition?: new () => SpeechRecognitionInstance
  }
}

function extractTodosFromText(text: string): string[] {
  const results: string[] = []
  const seen = new Set<string>()

  const add = (s: string) => {
    const t = s.trim()
    if (t.length > 1 && !seen.has(t)) { seen.add(t); results.push(t) }
  }

  // Pattern 1: "3시 미팅" style
  const p1 = /(\d+시[에]?\s*\S+)/g
  let m: RegExpExecArray | null
  while ((m = p1.exec(text)) !== null) add(m[1])

  // Pattern 2: "내일 보고서 제출" style
  const p2 = /(오늘|내일|이번주|다음주)\s+([가-힣\s]{2,})/g
  while ((m = p2.exec(text)) !== null) add(`${m[1]} ${m[2].trim()}`)

  // Pattern 3: "보고서 제출" keywords
  const p3 = /([가-힣]+)\s*(해야|필요|처리|확인|완료|마감|제출)/g
  while ((m = p3.exec(text)) !== null) add(`${m[1]} ${m[2]}`)

  return results
}

export function VoiceMemoView() {
  const { addMemo, addTodo } = useAppStore()
  const [supported, setSupported] = useState(true)
  const [recording, setRecording] = useState(false)
  const [interimText, setInterimText] = useState('')
  const [transcript, setTranscript] = useState('')
  const [todos, setTodos] = useState<{ text: string; checked: boolean }[]>([])
  const [savedMemo, setSavedMemo] = useState(false)
  const [savedTodo, setSavedTodo] = useState(false)
  const recognitionRef = useRef<SpeechRecognitionInstance | null>(null)

  useEffect(() => {
    const SRClass = window.SpeechRecognition ?? window.webkitSpeechRecognition
    if (!SRClass) setSupported(false)
  }, [])

  const startRecording = () => {
    const SRClass = window.SpeechRecognition ?? window.webkitSpeechRecognition
    if (!SRClass) return
    const sr = new SRClass()
    sr.lang = 'ko-KR'
    sr.interimResults = true
    sr.continuous = true
    sr.onresult = (e: SpeechRecognitionEvent) => {
      let interim = ''
      let final = transcript
      for (let i = e.resultIndex; i < e.results.length; i++) {
        const result = e.results[i]
        if (result.isFinal) final += result[0].transcript
        else interim += result[0].transcript
      }
      setTranscript(final)
      setInterimText(interim)
    }
    sr.onerror = () => { setRecording(false) }
    sr.onend = () => { setRecording(false); setInterimText('') }
    recognitionRef.current = sr
    sr.start()
    setRecording(true)
    setSavedMemo(false)
    setSavedTodo(false)
  }

  const stopRecording = () => {
    recognitionRef.current?.stop()
    setRecording(false)
    setInterimText('')
  }

  const handleExtract = () => {
    const extracted = extractTodosFromText(transcript)
    setTodos(extracted.map((t) => ({ text: t, checked: true })))
  }

  const handleSaveToMemo = () => {
    if (!transcript.trim()) return
    addMemo('음성 메모 ' + new Date().toLocaleString(), transcript)
    setSavedMemo(true)
  }

  const handleAddAllTodos = () => {
    todos.filter((t) => t.checked).forEach((t) => addTodo(t.text))
    setSavedTodo(true)
  }

  const handleAddSingleTodo = (text: string) => {
    addTodo(text)
  }

  const clearAll = () => {
    setTranscript('')
    setInterimText('')
    setTodos([])
    setSavedMemo(false)
    setSavedTodo(false)
  }

  if (!supported) {
    return (
      <div style={{
        flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center',
        background: 'var(--bg-base)', color: 'var(--text-primary)', padding: 24,
      }}>
        <div style={{
          padding: '20px 28px', borderRadius: 'var(--radius-lg)',
          background: 'rgba(239,68,68,0.08)', border: '1px solid rgba(239,68,68,0.3)',
          textAlign: 'center', maxWidth: 360,
        }}>
          <div style={{ fontSize: 32, marginBottom: 12 }}>🎙️</div>
          <div style={{ fontWeight: 600, marginBottom: 8 }}>음성 인식을 지원하지 않는 환경입니다</div>
          <div style={{ fontSize: 13, color: 'var(--text-secondary)' }}>Chrome 또는 Edge 브라우저를 사용해 주세요.</div>
        </div>
      </div>
    )
  }

  return (
    <div style={{
      flex: 1, overflowY: 'auto', padding: 24,
      background: 'var(--bg-base)', color: 'var(--text-primary)',
    }}>
      <motion.div initial={{ opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }} style={{ maxWidth: 580 }}>
        <h2 style={{ margin: '0 0 20px', fontSize: 20, fontWeight: 700 }}>🎙️ 음성 메모</h2>

        {/* Mic button */}
        <div style={{ display: 'flex', justifyContent: 'center', marginBottom: 24 }}>
          <motion.button
            onClick={recording ? stopRecording : startRecording}
            whileHover={{ scale: 1.05 }}
            whileTap={{ scale: 0.95 }}
            animate={recording ? { boxShadow: ['0 0 0 0 rgba(239,68,68,0.4)', '0 0 0 16px rgba(239,68,68,0)'] } : {}}
            transition={recording ? { duration: 1.2, repeat: Infinity } : {}}
            style={{
              width: 88, height: 88, borderRadius: '50%', border: 'none',
              background: recording ? 'var(--danger)' : 'var(--accent-primary)',
              color: '#fff', fontSize: 32, cursor: 'pointer',
              display: 'flex', alignItems: 'center', justifyContent: 'center',
            }}
          >{recording ? '⏹' : '🎙️'}</motion.button>
        </div>

        <div style={{ textAlign: 'center', marginBottom: 16, fontSize: 13, color: 'var(--text-secondary)' }}>
          {recording ? '🔴 녹음 중... 말씀하세요' : '버튼을 눌러 녹음을 시작하세요'}
        </div>

        {/* Transcript area */}
        <div style={{
          minHeight: 100, padding: '14px 16px', borderRadius: 'var(--radius-md)',
          background: 'var(--bg-surface)', border: '1px solid var(--glass-border)',
          fontSize: 14, lineHeight: 1.6, marginBottom: 12, wordBreak: 'break-all',
        }}>
          {transcript
            ? <span style={{ color: 'var(--text-primary)' }}>{transcript}</span>
            : <span style={{ color: 'var(--text-muted)' }}>변환된 텍스트가 여기 표시됩니다...</span>
          }
          {interimText && (
            <span style={{ color: 'var(--text-muted)', fontStyle: 'italic' }}> {interimText}</span>
          )}
        </div>

        {/* Action buttons */}
        {transcript && (
          <motion.div
            initial={{ opacity: 0, y: 6 }} animate={{ opacity: 1, y: 0 }}
            style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginBottom: 16 }}
          >
            <Btn onClick={handleExtract} color="var(--accent-primary)">📋 TODO 추출</Btn>
            <Btn onClick={handleSaveToMemo} color={savedMemo ? 'var(--success)' : undefined}>
              {savedMemo ? '✓ 메모에 저장됨' : '💾 메모에 저장'}
            </Btn>
            <Btn onClick={clearAll}>🗑 초기화</Btn>
          </motion.div>
        )}

        {/* Extracted TODOs */}
        <AnimatePresence>
          {todos.length > 0 && (
            <motion.div
              initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0 }}
              style={{
                padding: '14px 16px', borderRadius: 'var(--radius-md)',
                background: 'var(--bg-surface)', border: '1px solid var(--glass-border)',
              }}
            >
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 10 }}>
                <h4 style={{ margin: 0, fontSize: 13, fontWeight: 600, color: 'var(--text-secondary)' }}>
                  추출된 할 일 ({todos.length}개)
                </h4>
                <Btn onClick={handleAddAllTodos} color={savedTodo ? 'var(--success)' : 'var(--accent-primary)'}>
                  {savedTodo ? '✓ 전체 추가됨' : '✅ 전체 추가'}
                </Btn>
              </div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
                {todos.map((t, i) => (
                  <div key={i} style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <label style={{ display: 'flex', gap: 8, cursor: 'pointer', flex: 1, fontSize: 13 }}>
                      <input
                        type="checkbox"
                        checked={t.checked}
                        onChange={() => setTodos((prev) => prev.map((item, idx) =>
                          idx === i ? { ...item, checked: !item.checked } : item
                        ))}
                      />
                      <span style={{ color: t.checked ? 'var(--text-primary)' : 'var(--text-muted)', textDecoration: t.checked ? 'none' : 'line-through' }}>
                        {t.text}
                      </span>
                    </label>
                    <button
                      onClick={() => handleAddSingleTodo(t.text)}
                      style={{
                        padding: '3px 10px', borderRadius: 'var(--radius-sm)',
                        border: '1px solid var(--glass-border)', background: 'transparent',
                        color: 'var(--text-muted)', fontSize: 11, cursor: 'pointer',
                      }}
                    >추가</button>
                  </div>
                ))}
              </div>
            </motion.div>
          )}
        </AnimatePresence>
      </motion.div>
    </div>
  )
}

function Btn({ children, onClick, color }: { children: React.ReactNode; onClick: () => void; color?: string }) {
  return (
    <button
      onClick={onClick}
      style={{
        padding: '7px 14px', borderRadius: 'var(--radius-sm)',
        border: '1px solid var(--glass-border)', background: color ?? 'var(--glass-bg)',
        color: '#fff', cursor: 'pointer', fontSize: 12, fontWeight: 500,
      }}
    >{children}</button>
  )
}
