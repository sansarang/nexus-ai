import { useState } from 'react'
import { motion } from 'framer-motion'

type Mode = 'text' | 'bilingual' | 'url'

const LANGUAGES = [
  { code: 'ko', label: '한국어', flag: '🇰🇷' },
  { code: 'en', label: 'English', flag: '🇺🇸' },
  { code: 'ja', label: '日本語', flag: '🇯🇵' },
  { code: 'zh', label: '中文', flag: '🇨🇳' },
  { code: 'es', label: 'Español', flag: '🇪🇸' },
  { code: 'fr', label: 'Français', flag: '🇫🇷' },
  { code: 'de', label: 'Deutsch', flag: '🇩🇪' },
]

function translateText(text: string, from: string, to: string): string {
  if (!text.trim()) return ''
  if (from === to) return text
  return `[${to.toUpperCase()}] ${text}`
}

function makeBilingual(text: string, from: string, to: string): string {
  return text.split('\n').map((line) => {
    if (!line.trim()) return ''
    return `${line}  → ${translateText(line, from, to)}`
  }).join('\n')
}

function summarizeURL(url: string): string {
  return `URL 요약: ${url} - (실제 구현 시 fetch 후 텍스트 추출 필요)`
}

function LangSelector({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  return (
    <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap' }}>
      {LANGUAGES.map((lang) => (
        <button
          key={lang.code}
          onClick={() => onChange(lang.code)}
          style={{
            padding: '5px 10px', borderRadius: 8,
            border: `1px solid ${value === lang.code ? 'var(--accent-primary)' : 'var(--border-default)'}`,
            background: value === lang.code ? 'rgba(79,126,247,0.15)' : 'var(--glass-bg)',
            color: value === lang.code ? 'var(--accent-primary)' : 'var(--text-secondary)',
            fontSize: 12, fontWeight: value === lang.code ? 700 : 400,
            cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 4,
          }}
        >
          <span>{lang.flag}</span><span>{lang.label}</span>
        </button>
      ))}
    </div>
  )
}

export function TranslatorView() {
  const [mode, setMode] = useState<Mode>('text')
  const [from, setFrom] = useState('ko')
  const [to, setTo] = useState('en')
  const [input, setInput] = useState('')
  const [output, setOutput] = useState('')
  const [translating, setTranslating] = useState(false)
  const [copied, setCopied] = useState(false)

  const swap = () => {
    setFrom(to)
    setTo(from)
    setInput(output)
    setOutput(input)
  }

  const translate = async () => {
    if (!input.trim()) return
    setTranslating(true)
    await new Promise((r) => setTimeout(r, 500))
    let result = ''
    if (mode === 'text') {
      result = translateText(input, from, to)
    } else if (mode === 'bilingual') {
      result = makeBilingual(input, from, to)
    } else if (mode === 'url') {
      result = summarizeURL(input)
    }
    setOutput(result)
    setTranslating(false)
  }

  const copyOutput = async () => {
    if (!output) return
    try { await navigator.clipboard.writeText(output) } catch { /**/ }
    setCopied(true)
    setTimeout(() => setCopied(false), 1200)
  }

  const MODES: { id: Mode; label: string }[] = [
    { id: 'text', label: '텍스트 번역' },
    { id: 'bilingual', label: '2개 국어 병렬' },
    { id: 'url', label: 'URL 요약' },
  ]

  return (
    <div style={{ flex: 1, display: 'flex', flexDirection: 'column', padding: '20px 24px', gap: 16, overflowY: 'auto' }}>
      <h2 style={{ fontSize: 16, fontWeight: 800, color: 'var(--text-primary)', margin: 0 }}>🌐 번역기</h2>

      {/* Mode tabs */}
      <div style={{ display: 'flex', gap: 4 }}>
        {MODES.map((m) => (
          <button
            key={m.id}
            onClick={() => setMode(m.id)}
            style={{
              padding: '6px 14px', borderRadius: 'var(--radius-sm)',
              border: `1px solid ${mode === m.id ? 'var(--accent-primary)' : 'var(--border-default)'}`,
              background: mode === m.id ? 'rgba(79,126,247,0.12)' : 'transparent',
              color: mode === m.id ? 'var(--accent-primary)' : 'var(--text-secondary)',
              fontSize: 12, fontWeight: mode === m.id ? 600 : 400, cursor: 'pointer',
            }}
          >{m.label}</button>
        ))}
      </div>

      {/* Language selectors */}
      {mode !== 'url' && (
        <div style={{ display: 'flex', alignItems: 'center', gap: 12, flexWrap: 'wrap' }}>
          <LangSelector value={from} onChange={setFrom} />
          <button
            onClick={swap}
            style={{
              width: 32, height: 32, borderRadius: 8, border: '1px solid var(--border-default)',
              background: 'var(--glass-bg)', color: 'var(--text-secondary)',
              cursor: 'pointer', fontSize: 16, display: 'flex', alignItems: 'center', justifyContent: 'center',
            }}
          >⇄</button>
          <LangSelector value={to} onChange={setTo} />
        </div>
      )}

      {/* Input / Output panes */}
      <div style={{ display: 'flex', gap: 16, flex: 1, minHeight: 200 }}>
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 6 }}>
          <label style={{ fontSize: 10, fontWeight: 700, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
            {mode === 'url' ? 'URL' : (LANGUAGES.find((l) => l.code === from)?.label ?? from)}
          </label>
          <textarea
            value={input}
            onChange={(e) => { setInput(e.target.value); if (!e.target.value.trim()) setOutput('') }}
            placeholder={mode === 'url' ? 'https://example.com' : '텍스트를 입력하세요...'}
            style={{
              flex: 1, minHeight: 180, padding: '12px 14px',
              background: 'var(--bg-elevated)', border: '1px solid var(--border-subtle)',
              borderRadius: 'var(--radius-md)', color: 'var(--text-primary)',
              fontSize: 14, lineHeight: 1.6, resize: 'none', outline: 'none',
              fontFamily: 'Pretendard, Inter, sans-serif',
            }}
          />
          <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
            <motion.button
              onClick={translate}
              disabled={!input.trim() || translating}
              whileTap={{ scale: 0.96 }}
              style={{
                padding: '8px 20px', borderRadius: 8, border: 'none',
                background: 'var(--accent-primary)', color: '#fff',
                fontSize: 13, fontWeight: 700, cursor: input.trim() && !translating ? 'pointer' : 'not-allowed',
                opacity: input.trim() && !translating ? 1 : 0.5,
              }}
            >{translating ? '번역 중...' : '번역 →'}</motion.button>
          </div>
        </div>

        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 6 }}>
          <label style={{ fontSize: 10, fontWeight: 700, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
            {mode === 'url' ? '요약' : (LANGUAGES.find((l) => l.code === to)?.label ?? to)}
          </label>
          <div style={{
            flex: 1, minHeight: 180, padding: '12px 14px',
            background: 'var(--bg-elevated)', border: '1px solid var(--border-subtle)',
            borderRadius: 'var(--radius-md)', fontSize: 14, lineHeight: 1.6,
            color: output ? 'var(--text-primary)' : 'var(--text-muted)',
            whiteSpace: 'pre-wrap', wordBreak: 'break-word', overflowY: 'auto',
          }}>
            {output || '번역 결과가 여기에 나타납니다'}
          </div>
          <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
            <button
              onClick={copyOutput}
              disabled={!output}
              style={{
                padding: '6px 14px', borderRadius: 8, border: '1px solid var(--border-default)',
                background: 'var(--glass-bg)', color: copied ? 'var(--success)' : 'var(--text-secondary)',
                fontSize: 12, cursor: output ? 'pointer' : 'default', opacity: output ? 1 : 0.4,
              }}
            >{copied ? '✓ 복사됨' : '📋 복사'}</button>
          </div>
        </div>
      </div>
    </div>
  )
}
