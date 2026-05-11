import type { CharacterProps } from './types'

/* 닥터 — 전문가형 정장 AI 비서, 네이비/화이트 */
export function Doc({ emotion, speaking, listening }: CharacterProps) {
  const SKIN   = '#f0d9c8'
  const SUIT   = '#1e3a5f'
  const SHIRT  = '#f8fafc'
  const TIE    = '#2563eb'
  const HAIR   = '#374151'

  const smileD =
    speaking          ? 'M 84 88 Q 100 98 116 88'
    : emotion === 'happy'     ? 'M 83 87 Q 100 97 117 87'
    : emotion === 'concerned' ? 'M 86 91 Q 100 86 114 91'
    :                           'M 87 88 Q 100 94 113 88'

  return (
    <svg viewBox="0 0 200 380" width="200" height="380" style={{ overflow: 'visible' }}>
      {/* 다리 */}
      <rect x="62" y="238" width="32" height="112" rx="8" fill={SUIT} />
      <rect x="106" y="238" width="32" height="112" rx="8" fill={SUIT} />
      <rect x="58" y="334" width="42" height="18" rx="6" fill="#0f172a" />
      <rect x="100" y="334" width="42" height="18" rx="6" fill="#0f172a" />

      {/* 상체 (정장) */}
      <g style={{ animation: 'docBreathe 4s ease-in-out infinite', transformOrigin: '100px 185px' }}>
        {/* 정장 자켓 */}
        <path d="M 46 118 Q 38 145 40 238 L 160 238 Q 162 145 154 118 Z" fill={SUIT} />
        {/* 와이셔츠 */}
        <path d="M 82 118 L 90 238 L 110 238 L 118 118 Q 100 125 82 118 Z" fill={SHIRT} />
        {/* 넥타이 */}
        <path d="M 96 124 L 92 165 L 100 175 L 108 165 L 104 124 Q 100 128 96 124 Z" fill={TIE} />
        <path d="M 96 124 L 100 132 L 104 124 Q 100 120 96 124 Z" fill={TIE} fillOpacity="0.6" />
        {/* 자켓 라펠 */}
        <path d="M 82 118 L 70 145 L 80 145 L 90 130 Z" fill="#1a305a" />
        <path d="M 118 118 L 130 145 L 120 145 L 110 130 Z" fill="#1a305a" />
        {/* 단추 */}
        {[158, 178, 198, 218].map((y, i) => (
          <circle key={i} cx="100" cy={y} r="2.5" fill={SUIT} />
        ))}
        {/* 가슴 포켓 */}
        <rect x="118" y="138" width="24" height="18" rx="2" fill="#1a305a" />
        <rect x="122" y="134" width="16" height="8" rx="1" fill={SHIRT} />
      </g>

      {/* 왼팔 */}
      <g style={{
        transformOrigin: '45px 120px',
        animation: speaking ? 'docArmSpeak 1.4s ease-in-out infinite' : 'none',
      }}>
        <rect x="22" y="112" width="30" height="96" rx="8" fill={SUIT} />
        {/* 소매 커프 */}
        <rect x="22" y="194" width="30" height="14" rx="4" fill={SHIRT} />
        {/* 손 */}
        <ellipse cx="37" cy="216" rx="13" ry="11" fill={SKIN} />
        <path d="M 30 210 Q 27 203 31 201" stroke={SKIN} strokeWidth="3.5" strokeLinecap="round" />
        <path d="M 35 208 Q 33 200 37 198" stroke={SKIN} strokeWidth="3.5" strokeLinecap="round" />
        <path d="M 41 208 Q 40 200 43 199" stroke={SKIN} strokeWidth="3.5" strokeLinecap="round" />
      </g>

      {/* 오른팔 */}
      <g style={{
        transformOrigin: '155px 120px',
        animation: speaking ? 'docArmSpeakR 1.4s ease-in-out infinite 0.4s' : 'none',
      }}>
        <rect x="148" y="112" width="30" height="96" rx="8" fill={SUIT} />
        <rect x="148" y="194" width="30" height="14" rx="4" fill={SHIRT} />
        <ellipse cx="163" cy="216" rx="13" ry="11" fill={SKIN} />
        <path d="M 170 210 Q 173 203 169 201" stroke={SKIN} strokeWidth="3.5" strokeLinecap="round" />
        <path d="M 165 208 Q 167 200 163 198" stroke={SKIN} strokeWidth="3.5" strokeLinecap="round" />
        <path d="M 159 208 Q 160 200 157 199" stroke={SKIN} strokeWidth="3.5" strokeLinecap="round" />
      </g>

      {/* 머리 */}
      <g style={{
        transformOrigin: '100px 70px',
        animation: listening ? 'docHeadTilt 0.5s ease forwards' : 'docHeadIdle 5s ease-in-out infinite',
      }}>
        {/* 머리 */}
        <ellipse cx="100" cy="66" rx="42" ry="48" fill={SKIN} />
        {/* 머리카락 */}
        <path d="M 58 50 Q 65 20 100 22 Q 135 20 142 50 Q 130 30 100 32 Q 70 30 58 50 Z" fill={HAIR} />
        <path d="M 58 50 Q 56 60 58 72 Q 64 50 68 44 Z" fill={HAIR} />
        <path d="M 142 50 Q 144 60 142 72 Q 136 50 132 44 Z" fill={HAIR} />
        {/* 관자놀이 머리 */}
        <path d="M 60 68 Q 58 80 62 90" stroke={HAIR} strokeWidth="6" fill="none" strokeLinecap="round" />
        <path d="M 140 68 Q 142 80 138 90" stroke={HAIR} strokeWidth="6" fill="none" strokeLinecap="round" />

        {/* 귀 */}
        <ellipse cx="58" cy="70" rx="8" ry="10" fill={SKIN} />
        <ellipse cx="142" cy="70" rx="8" ry="10" fill={SKIN} />

        {/* 안경 */}
        <circle cx="82" cy="63" r="16" fill="none" stroke={HAIR} strokeWidth="2.5" />
        <circle cx="118" cy="63" r="16" fill="none" stroke={HAIR} strokeWidth="2.5" />
        <line x1="98" y1="63" x2="102" y2="63" stroke={HAIR} strokeWidth="2.5" />
        <line x1="66" y1="60" x2="58" y2="56" stroke={HAIR} strokeWidth="2" />
        <line x1="134" y1="60" x2="142" y2="56" stroke={HAIR} strokeWidth="2" />
        {/* 렌즈 반사 */}
        <path d="M 72 55 Q 76 52 82 54" fill="none" stroke="white" strokeWidth="1.5" strokeOpacity="0.4" />
        <path d="M 108 55 Q 112 52 118 54" fill="none" stroke="white" strokeWidth="1.5" strokeOpacity="0.4" />

        {/* 눈썹 */}
        <path d={emotion === 'concerned' ? 'M 74 52 Q 82 49 90 52' : 'M 74 51 Q 82 47 90 51'}
          fill="none" stroke={HAIR} strokeWidth="2.5" strokeLinecap="round" />
        <path d={emotion === 'concerned' ? 'M 110 52 Q 118 49 126 52' : 'M 110 51 Q 118 47 126 51'}
          fill="none" stroke={HAIR} strokeWidth="2.5" strokeLinecap="round" />

        {/* 눈 */}
        <ellipse cx="82" cy="63" rx={emotion === 'alert' ? 9 : 7} ry={emotion === 'happy' ? 5 : 8} fill="#1e293b" />
        <ellipse cx="118" cy="63" rx={emotion === 'alert' ? 9 : 7} ry={emotion === 'happy' ? 5 : 8} fill="#1e293b" />
        <circle cx="84" cy="61" r="2" fill="white" />
        <circle cx="120" cy="61" r="2" fill="white" />

        {/* 코 */}
        <path d="M 97 73 Q 95 80 98 82 Q 102 83 105 80 Q 107 77 105 73"
          fill="none" stroke={SKIN} strokeWidth="1.5" strokeOpacity="0.5" strokeLinecap="round" />

        {/* 입 */}
        <path d={smileD} fill={speaking ? SKIN : 'none'} stroke="#374151" strokeWidth="2" strokeLinecap="round" />
        {speaking && <rect x="94" y="88" width="12" height="5" rx="1" fill="white" />}

        {/* 목 */}
        <rect x="88" y="108" width="24" height="18" rx="4" fill={SKIN} />
      </g>

      <style>{`
        @keyframes docBreathe {
          0%,100% { transform: scaleY(1); }
          50% { transform: scaleY(1.012); }
        }
        @keyframes docHeadIdle {
          0%,100% { transform: translateY(0); }
          50% { transform: translateY(-2px); }
        }
        @keyframes docHeadTilt {
          to { transform: rotate(-7deg) translateX(-3px); }
        }
        @keyframes docArmSpeak {
          0%,100% { transform: rotate(0deg); }
          50% { transform: rotate(10deg) translateY(-4px); }
        }
        @keyframes docArmSpeakR {
          0%,100% { transform: rotate(0deg); }
          50% { transform: rotate(-8deg) translateY(-4px); }
        }
      `}</style>
    </svg>
  )
}
