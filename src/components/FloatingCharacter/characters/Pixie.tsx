import type { CharacterProps } from './types'

/* 픽시 — 귀여운 치비 스타일, 노랑/주황 */
export function Pixie({ emotion, speaking, listening }: CharacterProps) {
  const SKIN   = '#fef3c7'
  const HAIR   = '#78350f'
  const OUTFIT = '#f59e0b'
  const ACCENT = '#fb923c'
  const EYES   = '#7c3aed'

  /* 치비: 큰 머리, 작은 몸 비율 */
  const eyeRy = emotion === 'happy' ? 5 : emotion === 'alert' ? 14 : 11
  const smileD =
    speaking          ? 'M 82 100 Q 100 113 118 100'
    : emotion === 'happy'     ? 'M 79 99 Q 100 115 121 99'
    : emotion === 'concerned' ? 'M 83 105 Q 100 99 117 105'
    :                           'M 84 101 Q 100 110 116 101'

  return (
    <svg viewBox="0 0 200 380" width="200" height="380" style={{ overflow: 'visible' }}>
      {/* 통통한 다리 */}
      <g style={{ animation: 'pixieBounce 0.8s ease-in-out infinite', transformOrigin: '100px 320px' }}>
        <rect x="64" y="265" width="30" height="85" rx="14" fill="#d97706" />
        <rect x="106" y="265" width="30" height="85" rx="14" fill="#d97706" />
        {/* 귀여운 신발 */}
        <ellipse cx="79" cy="354" rx="22" ry="12" fill={ACCENT} />
        <ellipse cx="121" cy="354" rx="22" ry="12" fill={ACCENT} />
        {/* 신발 광택 */}
        <ellipse cx="73" cy="350" rx="8" ry="4" fill="white" fillOpacity="0.3" />
        <ellipse cx="115" cy="350" rx="8" ry="4" fill="white" fillOpacity="0.3" />
      </g>

      {/* 통통한 몸통 */}
      <g style={{ animation: 'pixieBreathe 2.5s ease-in-out infinite', transformOrigin: '100px 210px' }}>
        <ellipse cx="100" cy="215" rx="55" ry="60" fill={OUTFIT} />
        {/* 옷 포켓 */}
        <path d="M 72 220 Q 78 240 88 240 Q 82 240 78 220 Z" fill={ACCENT} fillOpacity="0.5" />
        <path d="M 128 220 Q 122 240 112 240 Q 118 240 122 220 Z" fill={ACCENT} fillOpacity="0.5" />
        {/* 단추 3개 */}
        {[195, 215, 235].map((y, i) => (
          <circle key={i} cx="100" cy={y} r="5" fill={ACCENT} />
        ))}
        {/* 옷 깃 */}
        <path d="M 80 165 L 92 205 L 100 195 L 108 205 L 120 165 Q 100 175 80 165 Z" fill="#fbbf24" />
      </g>

      {/* 짧은 팔 */}
      <g style={{
        transformOrigin: '48px 175px',
        animation: speaking
          ? 'pixieWaveL 0.6s ease-in-out infinite'
          : 'pixieArmIdle 2s ease-in-out infinite',
      }}>
        <ellipse cx="44" cy="190" rx="20" ry="28" fill={OUTFIT} />
        {/* 귀여운 손 */}
        <circle cx="40" cy="220" r="16" fill={SKIN} />
        {/* 손가락 */}
        <circle cx="32" cy="210" r="7" fill={SKIN} />
        <circle cx="26" cy="220" r="7" fill={SKIN} />
        <circle cx="29" cy="230" r="7" fill={SKIN} />
      </g>
      <g style={{
        transformOrigin: '152px 175px',
        animation: speaking
          ? 'pixieWaveR 0.6s ease-in-out infinite 0.2s'
          : 'pixieArmIdle 2s ease-in-out infinite 1s',
      }}>
        <ellipse cx="156" cy="190" rx="20" ry="28" fill={OUTFIT} />
        <circle cx="160" cy="220" r="16" fill={SKIN} />
        <circle cx="168" cy="210" r="7" fill={SKIN} />
        <circle cx="174" cy="220" r="7" fill={SKIN} />
        <circle cx="171" cy="230" r="7" fill={SKIN} />
      </g>

      {/* ── 큰 치비 머리 ── */}
      <g style={{
        transformOrigin: '100px 80px',
        animation: listening
          ? 'pixieHeadTilt 0.4s ease forwards'
          : emotion === 'happy'
          ? 'pixieHeadBounce 0.5s ease-in-out infinite'
          : 'pixieHeadIdle 3s ease-in-out infinite',
      }}>
        {/* 머리 */}
        <ellipse cx="100" cy="76" rx="64" ry="68" fill={SKIN} />

        {/* 머리카락 */}
        <path d="M 36 60 Q 44 10 100 14 Q 156 10 164 60 Q 148 22 100 24 Q 52 22 36 60 Z" fill={HAIR} />
        {/* 삐죽 앞머리 */}
        <path d="M 60 28 Q 64 8 72 16 Q 66 22 62 36 Z" fill={HAIR} />
        <path d="M 80 16 Q 84 4 92 10 Q 86 16 84 28 Z" fill={HAIR} />
        <path d="M 108 14 Q 114 3 120 10 Q 114 16 114 28 Z" fill={HAIR} />
        <path d="M 128 20 Q 136 8 142 18 Q 136 22 132 34 Z" fill={HAIR} />
        {/* 옆머리 */}
        <path d="M 36 60 Q 22 90 28 130 Q 36 90 44 72 Z" fill={HAIR} />
        <path d="M 164 60 Q 178 90 172 130 Q 164 90 156 72 Z" fill={HAIR} />
        {/* 귀여운 리본 */}
        <path d="M 28 75 Q 22 65 30 60 Q 36 70 30 78 Z" fill={ACCENT} />
        <path d="M 28 75 Q 38 70 36 80 Q 26 82 28 75 Z" fill={ACCENT} />
        <circle cx="29" cy="75" r="5" fill="white" />

        {/* 큰 귀 */}
        <ellipse cx="36" cy="80" rx="12" ry="14" fill={SKIN} />
        <ellipse cx="164" cy="80" rx="12" ry="14" fill={SKIN} />
        <ellipse cx="36" cy="80" rx="8" ry="10" fill={ACCENT} fillOpacity="0.3" />
        <ellipse cx="164" cy="80" rx="8" ry="10" fill={ACCENT} fillOpacity="0.3" />

        {/* 눈썹 (치비 스타일) */}
        <path d={emotion === 'concerned' ? 'M 70 58 Q 80 55 88 60' : 'M 70 56 Q 80 50 88 56'}
          fill="none" stroke={HAIR} strokeWidth="3.5" strokeLinecap="round" />
        <path d={emotion === 'concerned' ? 'M 112 58 Q 120 55 130 60' : 'M 112 56 Q 120 50 130 56'}
          fill="none" stroke={HAIR} strokeWidth="3.5" strokeLinecap="round" />

        {/* 매우 큰 눈 */}
        <ellipse cx="83" cy="72" rx={emotion === 'alert' ? 16 : 13} ry={eyeRy} fill="white" />
        <ellipse cx="117" cy="72" rx={emotion === 'alert' ? 16 : 13} ry={eyeRy} fill="white" />
        <ellipse cx="83" cy="73" rx="10" ry={eyeRy * 0.75} fill={EYES} />
        <ellipse cx="117" cy="73" rx="10" ry={eyeRy * 0.75} fill={EYES} />
        <ellipse cx="84" cy="72" rx="5" ry={eyeRy * 0.4} fill="#1e1b4b" />
        <ellipse cx="118" cy="72" rx="5" ry={eyeRy * 0.4} fill="#1e1b4b" />
        {/* 큰 반짝임 */}
        <circle cx="88" cy="66" r="4" fill="white" />
        <circle cx="122" cy="66" r="4" fill="white" />
        <circle cx="90" cy="76" r="2" fill="white" />
        <circle cx="124" cy="76" r="2" fill="white" />
        {/* 별 반짝임 (happy) */}
        {emotion === 'happy' && (
          <>
            <path d="M 94 57 L 95 53 L 96 57 L 100 58 L 96 59 L 95 63 L 94 59 L 90 58 Z"
              fill="#fbbf24" style={{ animation: 'pixieSpark 0.5s ease-in-out infinite' }} />
            <path d="M 108 55 L 109 52 L 110 55 L 113 56 L 110 57 L 109 60 L 108 57 L 105 56 Z"
              fill="#fbbf24" style={{ animation: 'pixieSpark 0.5s ease-in-out infinite 0.15s' }} />
          </>
        )}

        {/* 동그란 볼 */}
        <ellipse cx="65" cy="90" rx="14" ry="10" fill={ACCENT} fillOpacity="0.4" />
        <ellipse cx="135" cy="90" rx="14" ry="10" fill={ACCENT} fillOpacity="0.4" />

        {/* 작은 코 */}
        <ellipse cx="100" cy="88" rx="4" ry="3" fill={ACCENT} fillOpacity="0.4" />

        {/* 큰 입 (치비) */}
        <path
          d={smileD}
          fill={speaking ? '#fde8d8' : 'none'}
          stroke={HAIR}
          strokeWidth="2.5"
          strokeLinecap="round"
        />
        {speaking && (
          <>
            <ellipse cx="100" cy="105" rx="10" ry="7" fill="#fde8d8" />
            <ellipse cx="100" cy="107" rx="7" ry="4" fill="#f9a8d4" fillOpacity="0.6" />
          </>
        )}

        {/* 목 */}
        <rect x="86" y="138" width="28" height="18" rx="8" fill={SKIN} />
      </g>

      <style>{`
        @keyframes pixieBreathe {
          0%,100% { transform: scale(1); }
          50% { transform: scale(1.03) translateY(-3px); }
        }
        @keyframes pixieBounce {
          0%,100% { transform: translateY(0); }
          50% { transform: translateY(4px); }
        }
        @keyframes pixieHeadIdle {
          0%,100% { transform: translateY(0) rotate(0deg); }
          30% { transform: translateY(-4px) rotate(3deg); }
          60% { transform: translateY(2px) rotate(-2deg); }
        }
        @keyframes pixieHeadBounce {
          0%,100% { transform: translateY(0) rotate(-3deg); }
          50% { transform: translateY(-6px) rotate(3deg); }
        }
        @keyframes pixieHeadTilt {
          to { transform: rotate(-14deg) translateX(-6px); }
        }
        @keyframes pixieArmIdle {
          0%,100% { transform: rotate(0deg); }
          50% { transform: rotate(5deg); }
        }
        @keyframes pixieWaveL {
          0%,100% { transform: rotate(-10deg); }
          50% { transform: rotate(20deg) translateY(-8px); }
        }
        @keyframes pixieWaveR {
          0%,100% { transform: rotate(10deg); }
          50% { transform: rotate(-20deg) translateY(-8px); }
        }
        @keyframes pixieSpark {
          0%,100% { transform: scale(1) rotate(0deg); opacity: 1; }
          50% { transform: scale(1.4) rotate(20deg); opacity: 0.7; }
        }
      `}</style>
    </svg>
  )
}
