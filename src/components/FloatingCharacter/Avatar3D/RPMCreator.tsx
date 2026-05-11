/**
 * RPMCreator — Ready Player Me 아바타 크리에이터 iframe
 * 사용자가 자신만의 3D 사실적 아바타를 직접 생성
 * 완성 후 GLB URL을 자동으로 수신하여 저장
 */
import { useEffect, useRef, useState } from 'react'
import { motion } from 'framer-motion'

interface RPMCreatorProps {
  onAvatarCreated: (glbUrl: string, previewUrl: string) => void
  onClose: () => void
}

// RPM 서브도메인 — 개발자 계정에서 생성 (무료): https://docs.readyplayer.me
const RPM_SUBDOMAIN = 'demo' // 'nexus' 등 본인 subdomain으로 교체 가능

export function RPMCreator({ onAvatarCreated, onClose }: RPMCreatorProps) {
  const iframeRef = useRef<HTMLIFrameElement>(null)
  const [loading, setLoading]   = useState(true)
  const [progress, setProgress] = useState(0)

  const rpmUrl = `https://${RPM_SUBDOMAIN}.readyplayer.me/avatar?frameApi&clearCache&bodyType=fullbody&quickStart=false`

  // RPM postMessage 수신
  useEffect(() => {
    function onMessage(event: MessageEvent) {
      if (typeof event.data !== 'string') return

      // RPM이 완성된 아바타 GLB URL을 전송
      if (event.data.startsWith('https://models.readyplayer.me/') && event.data.endsWith('.glb')) {
        const glbUrl     = event.data
        const avatarId   = glbUrl.split('/').pop()?.replace('.glb', '') ?? ''
        const previewUrl = `https://models.readyplayer.me/${avatarId}/avatar.png?w=256&h=256&scene=fullbody-portrait-v1-transparent`
        onAvatarCreated(glbUrl, previewUrl)
      }

      // RPM 로딩 이벤트
      if (event.data === 'v1.frame.ready') setLoading(false)
    }

    window.addEventListener('message', onMessage)
    return () => window.removeEventListener('message', onMessage)
  }, [onAvatarCreated])

  // 가짜 로딩 프로그레스
  useEffect(() => {
    const t = setInterval(() => {
      setProgress(p => p < 90 ? p + 5 : p)
    }, 200)
    return () => clearInterval(t)
  }, [])

  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.95 }}
      animate={{ opacity: 1, scale: 1 }}
      exit={{ opacity: 0, scale: 0.95 }}
      style={{
        position: 'fixed', inset: 0, zIndex: 9999999,
        display: 'flex', flexDirection: 'column',
        background: '#0a0a14',
      }}
    >
      {/* 헤더 */}
      <div style={{
        display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        padding: '12px 20px',
        background: 'rgba(0,0,0,0.8)',
        borderBottom: '1px solid rgba(255,255,255,0.08)',
        flexShrink: 0,
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          <span style={{ fontSize: 20 }}>🧑‍🎤</span>
          <div>
            <div style={{ fontSize: 15, fontWeight: 700, color: 'white' }}>나만의 3D 아바타 만들기</div>
            <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.4)' }}>Ready Player Me — 무료 · 실시간 · 사진 업로드 지원</div>
          </div>
        </div>
        <motion.button
          whileTap={{ scale: 0.95 }}
          onClick={onClose}
          style={{
            background: 'rgba(255,255,255,0.08)',
            border: '1px solid rgba(255,255,255,0.12)',
            borderRadius: 8,
            color: 'rgba(255,255,255,0.7)',
            padding: '7px 16px',
            cursor: 'pointer',
            fontSize: 13,
          }}
        >
          ✕ 닫기
        </motion.button>
      </div>

      {/* RPM 크리에이터 안내 (로딩 중) */}
      {loading && (
        <div style={{
          position: 'absolute', inset: '60px 0 0 0',
          display: 'flex', flexDirection: 'column',
          alignItems: 'center', justifyContent: 'center',
          background: '#0a0a14', zIndex: 10,
          pointerEvents: 'none',
        }}>
          <motion.div
            animate={{ rotate: 360 }}
            transition={{ duration: 2, repeat: Infinity, ease: 'linear' }}
            style={{ fontSize: 48, marginBottom: 20 }}
          >
            🧬
          </motion.div>
          <div style={{ fontSize: 16, color: 'white', marginBottom: 12 }}>3D 아바타 크리에이터 로딩 중...</div>
          <div style={{ width: 240, height: 4, background: 'rgba(255,255,255,0.1)', borderRadius: 4 }}>
            <motion.div style={{
              width: `${progress}%`, height: '100%',
              background: 'linear-gradient(90deg, #7c3aed, #06b6d4)',
              borderRadius: 4,
              transition: 'width 0.2s',
            }} />
          </div>
        </div>
      )}

      {/* iframe */}
      <iframe
        ref={iframeRef}
        src={rpmUrl}
        allow="camera *; microphone *"
        style={{
          flex: 1, border: 'none',
          opacity: loading ? 0 : 1,
          transition: 'opacity 0.4s',
        }}
        onLoad={() => {
          setProgress(100)
          setTimeout(() => setLoading(false), 400)
        }}
      />

      {/* 하단 사용 안내 */}
      <div style={{
        padding: '10px 20px',
        background: 'rgba(0,0,0,0.7)',
        borderTop: '1px solid rgba(255,255,255,0.06)',
        display: 'flex', gap: 16, flexShrink: 0,
      }}>
        {[
          ['📸', '사진 업로드로 나와 닮은 아바타 생성'],
          ['🎨', '피부·머리·옷 전부 커스터마이즈'],
          ['✅', '완성 후 Nexus에 자동으로 적용됨'],
        ].map(([icon, text]) => (
          <div key={text} style={{ display: 'flex', alignItems: 'center', gap: 5, fontSize: 11, color: 'rgba(255,255,255,0.45)' }}>
            <span>{icon}</span><span>{text}</span>
          </div>
        ))}
      </div>
    </motion.div>
  )
}
