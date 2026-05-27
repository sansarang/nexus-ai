import type { CharacterProps } from './types'

/* Nexus — SF 홀로그래픽 AI · Nexus 기본 캐릭터 */
export function Nexus({ emotion, speaking, listening }: CharacterProps) {
  const SKIN   = '#f0e8ff'
  const HAIR   = '#e8f8ff'
  const SUIT1  = '#0f2942'
  const SUIT2  = '#163d5e'
  const CYAN   = '#22d3ee'
  const PURPLE = '#a78bfa'
  const GLOW   = speaking ? CYAN : emotion === 'alert' ? '#ef4444' : PURPLE

  const eyeY  = emotion === 'happy' ? 66 : 64
  const eyeRy = emotion === 'happy' ? 5 : 9

  const mouthD =
    speaking           ? 'M 84 92 Q 100 103 116 92'
    : emotion === 'happy'    ? 'M 82 90 Q 100 101 118 90'
    : emotion === 'concerned'? 'M 86 95 Q 100 89 114 95'
    : emotion === 'alert'    ? 'M 84 92 L 116 92'
    :                          'M 86 91 Q 100 98 114 91'

  return (
    <svg viewBox="0 0 200 390" width="200" height="390" style={{ overflow: 'visible' }}>
      <defs>
        <radialGradient id="nexusSkin" cx="50%" cy="35%" r="60%">
          <stop offset="0%" stopColor="#f8f0ff" />
          <stop offset="100%" stopColor={SKIN} />
        </radialGradient>
        <linearGradient id="nexusSuit" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor={SUIT1} />
          <stop offset="100%" stopColor={SUIT2} />
        </linearGradient>
        <linearGradient id="nexusHolo" x1="0%" y1="0%" x2="100%" y2="0%">
          <stop offset="0%" stopColor={CYAN} stopOpacity="0.6" />
          <stop offset="50%" stopColor={PURPLE} stopOpacity="0.4" />
          <stop offset="100%" stopColor={CYAN} stopOpacity="0.6" />
        </linearGradient>
        <filter id="nexusGlow">
          <feGaussianBlur stdDeviation="3" result="blur" />
          <feMerge><feMergeNode in="blur" /><feMergeNode in="SourceGraphic" /></feMerge>
        </filter>
        <filter id="nexusGlowSoft">
          <feGaussianBlur stdDeviation="1.5" result="blur" />
          <feMerge><feMergeNode in="blur" /><feMergeNode in="SourceGraphic" /></feMerge>
        </filter>
      </defs>

      {/* 홀로그래픽 날개 */}
      {[[-1, 50], [1, 55]].map(([dir, _y], i) => (
        <g key={i}>
          <path
            d={dir === -1
              ? 'M 40 120 Q 0 100 -20 160 Q 0 200 40 190 Q 20 160 30 130 Z'
              : 'M 160 120 Q 200 100 220 160 Q 200 200 160 190 Q 180 160 170 130 Z'}
            fill="url(#nexusHolo)"
            style={{ filter: 'blur(2px)' }}
          />
        </g>
      ))}

      {/* 다리 */}
      <g>
        <rect x="64" y="255" width="28" height="110" rx="10" fill={SUIT1} />
        <rect x="108" y="255" width="28" height="110" rx="10" fill={SUIT1} />
        <line x1="78" y1="260" x2="78" y2="350" stroke={CYAN} strokeWidth="1" strokeOpacity="0.5" />
        <line x1="122" y1="260" x2="122" y2="350" stroke={CYAN} strokeWidth="1" strokeOpacity="0.5" />
        <rect x="63" y="298" width="30" height="16" rx="4" fill={SUIT2} stroke={CYAN} strokeWidth="1" strokeOpacity="0.6" />
        <rect x="107" y="298" width="30" height="16" rx="4" fill={SUIT2} stroke={CYAN} strokeWidth="1" strokeOpacity="0.6" />
        <rect x="60" y="352" width="36" height="22" rx="8" fill={SUIT1} />
        <rect x="104" y="352" width="36" height="22" rx="8" fill={SUIT1} />
        <rect x="60" y="352" width="36" height="8" rx="4" fill={SUIT2} />
        <rect x="104" y="352" width="36" height="8" rx="4" fill={SUIT2} />
        <path d="M 62 360 L 94 360" stroke={CYAN} strokeWidth="1.5" strokeOpacity="0.7" />
        <path d="M 106 360 L 138 360" stroke={CYAN} strokeWidth="1.5" strokeOpacity="0.7" />
      </g>

      {/* 상체 */}
      <g style={{ animation: 'nexusBreathe 3.8s ease-in-out infinite', transformOrigin: '100px 180px' }}>
        <path d="M 52 116 Q 42 136 44 255 L 156 255 Q 158 136 148 116 Z" fill="url(#nexusSuit)" />
        <path d="M 72 140 L 128 140 L 132 175 L 100 185 L 68 175 Z" fill="none" stroke={CYAN} strokeWidth="1.5" strokeOpacity="0.7" />
        <circle cx="100" cy="162" r="10" fill="none" stroke={CYAN} strokeWidth="2" />
        <circle cx="100" cy="162" r="6" fill={CYAN} fillOpacity="0.3"
          style={{ animation: 'nexusPulse 2s ease-in-out infinite', filter: 'url(#nexusGlowSoft)' }} />
        <circle cx="100" cy="162" r="3" fill={CYAN} style={{ filter: 'url(#nexusGlow)' }} />
        <path d="M 44 116 Q 36 106 48 100 Q 58 112 52 118 Z" fill={SUIT2} stroke={CYAN} strokeWidth="1" strokeOpacity="0.5" />
        <path d="M 156 116 Q 164 106 152 100 Q 142 112 148 118 Z" fill={SUIT2} stroke={CYAN} strokeWidth="1" strokeOpacity="0.5" />
      </g>

      {/* 왼팔 */}
      <g style={{
        transformOrigin: '48px 118px',
        animation: speaking ? 'nexusArmSpeakL 1.1s ease-in-out infinite' : listening ? 'nexusArmListen 0.5s ease forwards' : 'nexusArmIdle 4s ease-in-out infinite',
      }}>
        <path d="M 50 116 Q 32 138 26 192 Q 38 200 52 194 Q 54 150 64 120 Z" fill={SUIT1} />
        <path d="M 42 140 Q 36 160 32 185" stroke={CYAN} strokeWidth="1" strokeOpacity="0.5" />
        <ellipse cx="36" cy="198" rx="13" ry="10" fill={SUIT2} />
        {[30, 35, 41].map((x, i) => (
          <circle key={i} cx={x} cy={194} r="2.5" fill={CYAN} fillOpacity="0.8" style={{ filter: 'url(#nexusGlowSoft)' }} />
        ))}
      </g>

      {/* 오른팔 */}
      <g style={{
        transformOrigin: '152px 118px',
        animation: speaking ? 'nexusArmSpeakR 1.1s ease-in-out infinite 0.4s' : 'nexusArmIdle 4s ease-in-out infinite 2s',
      }}>
        <path d="M 150 116 Q 168 138 174 192 Q 162 200 148 194 Q 146 150 136 120 Z" fill={SUIT1} />
        <path d="M 158 140 Q 164 160 168 185" stroke={CYAN} strokeWidth="1" strokeOpacity="0.5" />
        <ellipse cx="164" cy="198" rx="13" ry="10" fill={SUIT2} />
        {[159, 165, 170].map((x, i) => (
          <circle key={i} cx={x} cy={194} r="2.5" fill={CYAN} fillOpacity="0.8" style={{ filter: 'url(#nexusGlowSoft)' }} />
        ))}
      </g>

      {/* 머리 */}
      <g style={{
        transformOrigin: '100px 64px',
        animation: listening ? 'nexusHeadTilt 0.5s ease forwards' : 'nexusHeadBob 5s ease-in-out infinite',
      }}>
        <ellipse cx="100" cy="60" rx="46" ry="52" fill={HAIR} />
        <path d="M 60 80 Q 44 130 50 200 Q 58 205 64 198 Q 58 140 70 90 Z" fill={HAIR} />
        <path d="M 140 80 Q 156 130 150 200 Q 142 205 136 198 Q 142 140 130 90 Z" fill={HAIR} />
        <ellipse cx="100" cy="58" rx="42" ry="44" fill="url(#nexusSkin)" />
        <path d="M 54 50 Q 54 30 68 25" fill="none" stroke={CYAN} strokeWidth="3" strokeLinecap="round" style={{ filter: 'url(#nexusGlowSoft)' }} />
        <path d="M 146 50 Q 146 30 132 25" fill="none" stroke={CYAN} strokeWidth="3" strokeLinecap="round" style={{ filter: 'url(#nexusGlowSoft)' }} />
        <circle cx="54" cy="50" r="6" fill={SUIT1} stroke={CYAN} strokeWidth="2" />
        <circle cx="146" cy="50" r="6" fill={SUIT1} stroke={CYAN} strokeWidth="2" />
        <circle cx="54" cy="50" r="2" fill={CYAN} style={{ animation: 'nexusLED 2s ease-in-out infinite' }} />
        <circle cx="146" cy="50" r="2" fill={GLOW} style={{ animation: 'nexusLED 2s ease-in-out infinite 0.5s' }} />
        {emotion === 'alert' && (
          <path d="M 58 45 L 142 45" stroke={GLOW} strokeWidth="1.5" strokeOpacity="0.5" strokeDasharray="4,3" style={{ animation: 'nexusHUD 0.5s linear infinite' }} />
        )}
        <path d="M 58 36 Q 72 12 100 16 Q 128 12 142 36 Q 128 22 100 24 Q 72 22 58 36 Z" fill={HAIR} />
        <path d="M 82 20 Q 80 6 88 2" fill="none" stroke={HAIR} strokeWidth="5" strokeLinecap="round" />
        <circle cx="88" cy="2" r="4" fill={CYAN} style={{ filter: 'url(#nexusGlow)' }} />
        <ellipse cx="57" cy="68" rx="8" ry="10" fill={SKIN} />
        <ellipse cx="143" cy="68" rx="8" ry="10" fill={SKIN} />
        <path d={emotion === 'happy' ? 'M 74 49 Q 83 44 92 49' : 'M 72 51 Q 83 46 94 50'} fill="none" stroke={HAIR} strokeWidth="3" strokeLinecap="round" />
        <path d={emotion === 'happy' ? 'M 108 49 Q 117 44 126 49' : 'M 106 50 Q 117 46 128 51'} fill="none" stroke={HAIR} strokeWidth="3" strokeLinecap="round" />
        <ellipse cx="83" cy={eyeY} rx="11" ry={eyeRy} fill={SUIT1} />
        <ellipse cx="117" cy={eyeY} rx="11" ry={eyeRy} fill={SUIT1} />
        <ellipse cx="83" cy={eyeY} rx="7" ry={eyeRy - 2} fill={CYAN} fillOpacity="0.5" style={{ filter: 'url(#nexusGlowSoft)' }} />
        <ellipse cx="117" cy={eyeY} rx="7" ry={eyeRy - 2} fill={GLOW} fillOpacity="0.5" style={{ filter: 'url(#nexusGlowSoft)' }} />
        <circle cx="83" cy={eyeY} r="4" fill={CYAN} style={{ filter: 'url(#nexusGlowSoft)', animation: 'nexusEye 6s ease-in-out infinite' }} />
        <circle cx="117" cy={eyeY} r="4" fill={GLOW} style={{ filter: 'url(#nexusGlowSoft)', animation: 'nexusEye 6s ease-in-out infinite 0.5s' }} />
        {emotion === 'alert' && <>
          <line x1="72" y1={eyeY} x2="94" y2={eyeY} stroke={CYAN} strokeWidth="1" strokeOpacity="0.4" />
          <line x1="106" y1={eyeY} x2="128" y2={eyeY} stroke={CYAN} strokeWidth="1" strokeOpacity="0.4" />
        </>}
        <ellipse cx="86" cy={eyeY - 3} rx="3" ry="2" fill="white" fillOpacity="0.8" />
        <ellipse cx="120" cy={eyeY - 3} rx="3" ry="2" fill="white" fillOpacity="0.8" />
        <line x1="98" y1="78" x2="98" y2="82" stroke={SKIN} strokeWidth="2" strokeLinecap="round" strokeOpacity="0.5" />
        <line x1="102" y1="82" x2="98" y2="82" stroke={SKIN} strokeWidth="2" strokeLinecap="round" strokeOpacity="0.5" />
        <path d={mouthD}
          fill={speaking ? '#c8e8ff' : 'none'}
          stroke={speaking ? CYAN : '#8899bb'}
          strokeWidth="2" strokeLinecap="round"
          style={{ filter: speaking ? 'url(#nexusGlowSoft)' : undefined }}
        />
        {speaking && <rect x="94" y="92" width="12" height="6" rx="2" fill="white" fillOpacity="0.8" />}
        <rect x="88" y="100" width="24" height="18" rx="4" fill={SKIN} />
        <path d="M 92 102 L 92 115" stroke={CYAN} strokeWidth="0.5" strokeOpacity="0.4" />
        <path d="M 108 102 L 108 115" stroke={CYAN} strokeWidth="0.5" strokeOpacity="0.4" />
      </g>

      {speaking && (
        <g style={{ animation: 'nexusHUDFade 0.8s ease-in-out infinite' }}>
          <rect x="20" y="180" width="60" height="2" rx="1" fill={CYAN} fillOpacity="0.3" />
          <rect x="20" y="185" width="40" height="1" rx="0.5" fill={CYAN} fillOpacity="0.2" />
          <rect x="120" y="180" width="60" height="2" rx="1" fill={CYAN} fillOpacity="0.3" />
          <rect x="140" y="185" width="40" height="1" rx="0.5" fill={CYAN} fillOpacity="0.2" />
        </g>
      )}

      <style>{`
        @keyframes nexusBreathe { 0%,100% { transform: scaleY(1); } 50% { transform: scaleY(1.012) translateY(-1px); } }
        @keyframes nexusHeadBob { 0%,100% { transform: translateY(0) rotate(0deg); } 35% { transform: translateY(-3px) rotate(0.8deg); } 70% { transform: translateY(1px) rotate(-0.5deg); } }
        @keyframes nexusHeadTilt { to { transform: rotate(-8deg) translateX(-4px); } }
        @keyframes nexusArmIdle { 0%,100% { transform: rotate(0deg); } 50% { transform: rotate(1.5deg); } }
        @keyframes nexusArmSpeakL { 0%,100% { transform: rotate(-3deg) translateY(0); } 50% { transform: rotate(10deg) translateY(-6px); } }
        @keyframes nexusArmSpeakR { 0%,100% { transform: rotate(3deg) translateY(0); } 50% { transform: rotate(-10deg) translateY(-5px); } }
        @keyframes nexusArmListen { to { transform: rotate(-7deg); } }
        @keyframes nexusPulse { 0%,100% { r: 6; opacity: 0.3; } 50% { r: 9; opacity: 0.6; } }
        @keyframes nexusLED { 0%,100% { opacity: 0.5; } 50% { opacity: 1; } }
        @keyframes nexusEye { 0%,38%,100% { transform: translateX(0); } 48% { transform: translateX(2.5px); } 68% { transform: translateX(-1.5px); } }
        @keyframes nexusHUD { 0% { stroke-dashoffset: 0; } 100% { stroke-dashoffset: -14; } }
        @keyframes nexusHUDFade { 0%,100% { opacity: 0.4; } 50% { opacity: 0.9; } }
      `}</style>
    </svg>
  )
}
