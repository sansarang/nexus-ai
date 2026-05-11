// Web Speech API types (not universally available in all TS DOM lib versions)
interface SRAlternative { transcript: string; confidence: number }
interface SRResult { [i: number]: SRAlternative; length: number }
interface SRResultList { [i: number]: SRResult; length: number }
interface SREvent extends Event { results: SRResultList }
interface SRErrorEvent extends Event { error: string }
interface SRInstance {
  lang: string
  continuous: boolean
  interimResults: boolean
  onresult: ((e: SREvent) => void) | null
  onend: (() => void) | null
  onerror: ((e: SRErrorEvent) => void) | null
  start(): void
  stop(): void
}
type SRConstructor = { new(): SRInstance }

let recognitionInstance: SRInstance | null = null
let isActive = false
let onWakeCallback: (() => void) | null = null
let activeWakeWords: string[] = ['hey nexus', '헤이 넥서스', '넥서스', '자비스', 'hey jarvis']
// setTimeout ID 저장 — cleanup 시 clearTimeout 가능
let retryTimer: ReturnType<typeof setTimeout> | null = null

export const startWakeWordDetection = (wakeWords: string[], onWake: () => void): void => {
  if (wakeWords.length > 0) {
    activeWakeWords = wakeWords.map(w => w.toLowerCase())
  }
  onWakeCallback = onWake
  const win = window as unknown as Record<string, SRConstructor | undefined>
  const SR: SRConstructor | undefined = win['SpeechRecognition'] ?? win['webkitSpeechRecognition']
  if (!SR) return
  isActive = true
  startLoop(SR)
}

function startLoop(SR: SRConstructor): void {
  if (!isActive) return

  // 중복 인스턴스 방지
  if (recognitionInstance) {
    try { recognitionInstance.stop() } catch { /* already stopped */ }
    recognitionInstance = null
  }

  const r = new SR()
  recognitionInstance = r
  r.lang = 'ko-KR'
  r.continuous = false
  r.interimResults = true

  r.onresult = (e: SREvent) => {
    const transcript = Array.from({ length: e.results.length })
      .map((_, i) => e.results[i][0].transcript.toLowerCase())
      .join(' ')
    if (activeWakeWords.some(w => transcript.includes(w))) {
      r.stop()
      onWakeCallback?.()
    }
  }

  r.onend = () => {
    if (!isActive) return
    // 이전 타이머 취소 후 재시작 (중복 방지)
    if (retryTimer !== null) clearTimeout(retryTimer)
    retryTimer = setTimeout(() => {
      retryTimer = null
      startLoop(SR)
    }, 500)
  }

  r.onerror = (e: SRErrorEvent) => {
    if (e.error === 'aborted' || !isActive) return
    if (retryTimer !== null) clearTimeout(retryTimer)
    retryTimer = setTimeout(() => {
      retryTimer = null
      startLoop(SR)
    }, 1000)
  }

  try { r.start() } catch { /* already running */ }
}

export const stopWakeWordDetection = (): void => {
  isActive = false
  // 대기 중인 재시작 타이머 취소
  if (retryTimer !== null) {
    clearTimeout(retryTimer)
    retryTimer = null
  }
  if (recognitionInstance) {
    try { recognitionInstance.stop() } catch { /* already stopped */ }
    recognitionInstance = null
  }
  onWakeCallback = null
}
