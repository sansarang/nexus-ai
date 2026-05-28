/**
 * Nexus TTS Engine
 * 우선순위: OpenAI TTS → Web Speech API (Neural 음성 우선)
 */

/* 마크다운 / 특수문자 제거 */
export function cleanForSpeech(text: string): string {
  return text
    .replace(/\*\*(.+?)\*\*/g, '$1')
    .replace(/\*(.+?)\*/g, '$1')
    .replace(/#+\s/g, '')
    .replace(/`{1,3}[^`]*`{1,3}/g, '')
    .replace(/\[(.+?)\]\(.+?\)/g, '$1')
    .replace(/https?:\/\/\S+/g, '')
    .replace(/[\u{1F300}-\u{1FFFF}]/gu, '')
    .replace(/[✅❌🔑🔍🧹🛡️📂🎯📅🕐🌤️💡⚡🎉✨◉]/g, '')
    .replace(/\n+/g, '. ')
    .replace(/\s{2,}/g, ' ')
    .trim()
}

/* ── OpenAI TTS (가장 자연스러운 음성) ── */
const OPENAI_TTS_VOICES_KO = ['nova', 'shimmer', 'alloy'] // 한국어 발음 좋은 순
const OPENAI_TTS_VOICES_EN = ['nova', 'alloy', 'shimmer']

export async function speakWithOpenAI(
  text: string,
  lang: 'ko' | 'en',
  onStart?: () => void,
  onEnd?: () => void,
  speedOverride?: number,
  voiceOverride?: string,
): Promise<boolean> {
  const apiKey = localStorage.getItem('nexus-openai-key') ?? ''
  if (!apiKey) return false

  const clean = cleanForSpeech(text).slice(0, 1000)
  if (!clean) return false

  try {
    onStart?.()
    const voices = lang === 'ko' ? OPENAI_TTS_VOICES_KO : OPENAI_TTS_VOICES_EN
    const voice = voiceOverride ?? voices[0]

    const res = await fetch('https://api.openai.com/v1/audio/speech', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${apiKey}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        model: 'tts-1',
        input: clean,
        voice,
        speed: speedOverride ?? 1.1,
      }),
    })

    if (!res.ok) return false

    const blob = await res.blob()
    const url = URL.createObjectURL(blob)
    // 이전 재생 중지 (핸들러 먼저 제거 → onEnd 중복 호출 방지)
    if (currentAudio) {
      currentAudio.onended = null
      currentAudio.onerror = null
      currentAudio.pause()
      currentAudio.src = ''
    }
    const audio = new Audio(url)
    currentAudio = audio
    audio.onended = () => {
      URL.revokeObjectURL(url)
      currentAudio = null
      onEnd?.()
    }
    // onerror: 재생 실패 시 URL 해제 + onEnd 호출 (silent fail 방지)
    audio.onerror = (e) => {
      console.warn('[TTS] Audio playback error:', e)
      URL.revokeObjectURL(url)
      currentAudio = null
      onEnd?.()
    }
    try {
      await audio.play()
    } catch (playErr) {
      console.warn('[TTS] audio.play() rejected:', playErr)
      URL.revokeObjectURL(url)
      onEnd?.()
      return false
    }
    return true
  } catch {
    onEnd?.()
    return false
  }
}

/* ── Web Speech API 음성 우선순위 선택 ── */
function getBestVoice(lang: 'ko' | 'en'): SpeechSynthesisVoice | null {
  const voices = window.speechSynthesis?.getVoices() ?? []
  const langPrefix = lang === 'ko' ? 'ko' : 'en'

  const candidates = voices.filter(v => v.lang.toLowerCase().startsWith(langPrefix))
  if (!candidates.length) return null

  // Neural / Google / 고품질 음성 우선 선택
  const PRIORITY_KEYWORDS = [
    'neural', 'google', 'heami', 'soyeon', 'sunhi', 'yuna', // 한국어 좋은 음성
    'ava', 'nova', 'allison', 'samantha',                    // 영어 좋은 음성
  ]

  for (const kw of PRIORITY_KEYWORDS) {
    const match = candidates.find(v =>
      v.name.toLowerCase().includes(kw)
    )
    if (match) return match
  }

  // 로컬 음성 우선 (원격보다 자연스럽게 들릴 수 있음)
  const local = candidates.find(v => v.localService)
  return local ?? candidates[0]
}

/* ── Web Speech API TTS ── */
export function speakWithWebSpeech(
  text: string,
  lang: 'ko' | 'en',
  onStart?: () => void,
  onEnd?: () => void,
  rateOverride?: number,
  pitchOverride?: number,
): void {
  if (!window.speechSynthesis) { onEnd?.(); return }

  window.speechSynthesis.cancel()
  const clean = cleanForSpeech(text).slice(0, 400)
  if (!clean) { onEnd?.(); return }

  // 문장 단위로 나눠서 자연스러운 호흡감 부여
  const sentences = clean
    .split(/(?<=[.!?。！？])\s+/)
    .map(s => s.trim())
    .filter(Boolean)

  // 분리된 문장이 없으면(구두점 없는 짧은 문장) 전체를 하나로 처리
  const finalSentences = sentences.length > 0 ? sentences : [clean]
  let idx = 0

  const speakNext = () => {
    // Chrome: cancel() 직후 speak() 호출 시 겹침 방지 — speaking 상태 확인
    if (window.speechSynthesis.speaking) {
      setTimeout(speakNext, 50)
      return
    }
    if (idx >= finalSentences.length) return

    const utterance = new SpeechSynthesisUtterance(finalSentences[idx])
    utterance.lang = lang === 'ko' ? 'ko-KR' : 'en-US'

    // 감정별 파라미터 (override 없으면 기본값)
    utterance.rate = rateOverride ?? 1.05
    utterance.pitch = pitchOverride ?? 1.1
    utterance.volume = 1.0

    const voice = getBestVoice(lang)
    if (voice) utterance.voice = voice

    if (idx === 0) utterance.onstart = () => onStart?.()
    utterance.onend = () => {
      idx++
      // 문장 사이 자연스러운 쉬기 (마지막 문장은 즉시 onEnd 호출)
      if (idx < finalSentences.length) {
        setTimeout(speakNext, 80)
      } else {
        onEnd?.()
      }
    }
    utterance.onerror = (e) => {
      console.warn('[TTS] Utterance error:', e.error)
      idx++
      speakNext()
    }

    window.speechSynthesis.speak(utterance)
  }

  // Chrome voices 비동기 로딩 대응 — 최대 1초 대기 후 재시도
  const trySpeak = () => {
    if (window.speechSynthesis.getVoices().length > 0) {
      speakNext()
    } else {
      let retries = 0
      const poll = setInterval(() => {
        retries++
        if (window.speechSynthesis.getVoices().length > 0 || retries > 10) {
          clearInterval(poll)
          speakNext()
        }
      }, 100)
      window.speechSynthesis.onvoiceschanged = () => { clearInterval(poll); speakNext() }
    }
  }
  trySpeak()
}

export type SpeakEmotion = 'neutral' | 'happy' | 'concerned' | 'alert' | 'humorous'

/* ── 감정별 TTS 파라미터 ── */
const EMOTION_PARAMS: Record<SpeakEmotion, { rate: number; pitch: number; openaiSpeed: number }> = {
  neutral:   { rate: 0.9,  pitch: 1.05, openaiSpeed: 0.95 },
  happy:     { rate: 1.0,  pitch: 1.2,  openaiSpeed: 1.05 },  // 밝고 경쾌
  concerned: { rate: 0.85, pitch: 0.95, openaiSpeed: 0.9  },  // 차분하고 진지
  alert:     { rate: 0.8,  pitch: 0.9,  openaiSpeed: 0.85 },  // 낮고 진중 (경고)
  humorous:  { rate: 1.05, pitch: 1.15, openaiSpeed: 1.0  },  // 유머러스
}

/* ── 통합 speak 함수 ── */
export async function speak(
  text: string,
  lang: 'ko' | 'en',
  onStart?: () => void,
  onEnd?: () => void,
  emotion: SpeakEmotion = 'neutral',
  voiceOverride?: string,
  isPro = false, // Pro 구독자만 OpenAI TTS 사용
): Promise<void> {
  // OpenAI TTS: Pro 구독자만 사용 (비용 절감)
  if (isPro) {
    const used = await speakWithOpenAI(text, lang, onStart, onEnd, EMOTION_PARAMS[emotion].openaiSpeed, voiceOverride)
    if (used) return
  }

  // 무료 사용자 또는 OpenAI 실패 → Web Speech API (비용 $0)
  speakWithWebSpeech(text, lang, onStart, onEnd, EMOTION_PARAMS[emotion].rate, EMOTION_PARAMS[emotion].pitch)
}

/* 현재 재생 중인 OpenAI Audio 전역 추적 */
let currentAudio: HTMLAudioElement | null = null

/* TTS 중지 — OpenAI Audio + WebSpeech 모두 정지 */
export function stopSpeaking(): void {
  if (currentAudio) {
    currentAudio.onended = null
    currentAudio.onerror = null
    currentAudio.pause()
    currentAudio.src = ''
    currentAudio = null
  }
  window.speechSynthesis?.cancel()
}

/* 현재 오디오 재생 중 여부 */
export function isAudioPlaying(): boolean {
  if (currentAudio && !currentAudio.paused && !currentAudio.ended) return true
  if (typeof window !== 'undefined' && window.speechSynthesis?.speaking) return true
  return false
}
