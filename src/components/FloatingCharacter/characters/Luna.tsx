import type { CharacterProps } from './types'

/* 루나 — 캐주얼 젊은 비서, 보라/핑크 */
export function Luna({ emotion, speaking, listening }: CharacterProps) {
  const SKIN   = '#fde8d8'
  const HAIR   = '#3b1f8c'
  const OUTFIT = '#7c3aed'
  const ACCENT = '#f9a8d4'
  const DARK   = '#1e1b2e'

  const eyeY  = emotion === 'happy' ? 65 : 63
  const smileD =
    speaking          ? 'M 84 90 Q 100 99 116 90'
    : emotion === 'happy'     ? 'M 82 89 Q 100 100 118 89'
    : emotion === 'concerned' ? 'M 86 93 Q 100 88 114 93'
    : emotion === 'humorous'  ? 'M 84 89 Q 100 98 116 89 Q 108 102 92 102 Z'
    :                           'M 86 90 Q 100 97 114 90'

  return (
    <svg viewBox="0 0 200 380" width="200" height="380" style={{ overflow: 'visible' }}>
      <defs>
        <radialGradient id="lunaSkin" cx="50%" cy="40%" r="60%">
          <stop offset="0%" stopColor="#fef3e8" />
          <stop offset="100%" stopColor={SKIN} />
        </radialGradient>
      </defs>

      {/* ── 다리 ── */}
      <g>
        <rect x="64" y="235" width="30" height="110" rx="12" fill="#4c1d95" />
        <rect x="106" y="235" width="30" height="110" rx="12" fill="#4c1d95" />
        {/* 신발 */}
        <ellipse cx="79" cy="348" rx="20" ry="9" fill={DARK} />
        <ellipse cx="121" cy="348" rx="20" ry="9" fill={DARK} />
        <ellipse cx="79" cy="345" rx="16" ry="6" fill={OUTFIT} fillOpacity="0.4" />
        <ellipse cx="121" cy="345" rx="16" ry="6" fill={OUTFIT} fillOpacity="0.4" />
      </g>

      {/* ── 상체 (후디) ── */}
      <g style={{ animation: 'lunaBreathe 3.5s ease-in-out infinite', transformOrigin: '100px 185px' }}>
        {/* 후디 본체 */}
        <path d="M 50 120 Q 40 140 42 235 L 158 235 Q 160 140 150 120 Z" fill={OUTFIT} />
        {/* 후디 주머니 */}
        <rect x="68" y="185" width="64" height="35" rx="8" fill="#6d28d9" />
        <line x1="100" y1="185" x2="100" y2="220" stroke={OUTFIT} strokeWidth="1.5" />
        {/* 후디 줄 */}
        <path d="M 84 128 L 82 148" stroke={ACCENT} strokeWidth="2.5" strokeLinecap="round" />
        <path d="M 116 128 L 118 148" stroke={ACCENT} strokeWidth="2.5" strokeLinecap="round" />
        <circle cx="82" cy="150" r="4" fill={ACCENT} />
        <circle cx="118" cy="150" r="4" fill={ACCENT} />
      </g>

      {/* ── 왼팔 ── */}
      <g style={{
        transformOrigin: '47px 122px',
        animation: speaking
          ? 'lunaArmSpeakL 1s ease-in-out infinite'
          : listening
          ? 'lunaArmListen 0.5s ease forwards'
          : 'lunaArmIdle 3s ease-in-out infinite',
      }}>
        <path d="M 48 120 Q 32 140 28 195 Q 40 200 52 195 Q 54 150 64 125 Z" fill={OUTFIT} />
        {/* 손 */}
        <ellipse cx="38" cy="200" rx="13" ry="11" fill="url(#lunaSkin)" />
        <path d="M 30 196 Q 28 190 32 188" stroke={SKIN} strokeWidth="3" strokeLinecap="round" />
        <path d="M 35 193 Q 33 186 37 184" stroke={SKIN} strokeWidth="3" strokeLinecap="round" />
        <path d="M 41 193 Q 40 186 43 185" stroke={SKIN} strokeWidth="3" strokeLinecap="round" />
      </g>

      {/* ── 오른팔 ── */}
      <g style={{
        transformOrigin: '153px 122px',
        animation: speaking
          ? 'lunaArmSpeakR 1s ease-in-out infinite 0.25s'
          : 'lunaArmIdle 3s ease-in-out infinite 1.5s',
      }}>
        <path d="M 152 120 Q 168 140 172 195 Q 160 200 148 195 Q 146 150 136 125 Z" fill={OUTFIT} />
        <ellipse cx="162" cy="200" rx="13" ry="11" fill="url(#lunaSkin)" />
        <path d="M 170 196 Q 172 190 168 188" stroke={SKIN} strokeWidth="3" strokeLinecap="round" />
        <path d="M 165 193 Q 167 186 163 184" stroke={SKIN} strokeWidth="3" strokeLinecap="round" />
        <path d="M 159 193 Q 160 186 157 185" stroke={SKIN} strokeWidth="3" strokeLinecap="round" />
      </g>

      {/* ── 머리 ── */}
      <g style={{
        transformOrigin: '100px 72px',
        animation: listening
          ? 'lunaHeadTilt 0.5s ease forwards'
          : 'lunaHeadBob 4s ease-in-out infinite',
      }}>
        {/* 머리카락 뒷부분 */}
        <ellipse cx="100" cy="66" rx="46" ry="52" fill={HAIR} />
        {/* 긴 머리 (양 옆으로 흘러내림) */}
        <path d="M 54 70 Q 30 120 36 185 Q 44 190 50 180 Q 46 130 62 90 Z" fill={HAIR} />
        <path d="M 146 70 Q 170 120 164 185 Q 156 190 150 180 Q 154 130 138 90 Z" fill={HAIR} />
        {/* 머리카락 앞부분 */}
        <ellipse cx="100" cy="58" rx="42" ry="44" fill="url(#lunaSkin)" />
        {/* 앞머리 */}
        <path d="M 58 42 Q 70 18 100 22 Q 130 18 142 42 Q 130 28 100 30 Q 70 28 58 42 Z" fill={HAIR} />
        <path d="M 58 44 Q 62 30 72 28 Q 66 36 64 48 Z" fill={HAIR} />
        <path d="M 142 44 Q 138 30 128 28 Q 134 36 136 48 Z" fill={HAIR} />
        {/* 앞머리 잔머리 */}
        <path d="M 76 26 Q 74 15 80 12" stroke={HAIR} strokeWidth="3" fill="none" strokeLinecap="round" />
        <path d="M 100 22 Q 100 10 104 8" stroke={HAIR} strokeWidth="3" fill="none" strokeLinecap="round" />

        {/* 귀 */}
        <ellipse cx="56" cy="74" rx="8" ry="10" fill={SKIN} />
        <ellipse cx="144" cy="74" rx="8" ry="10" fill={SKIN} />
        <ellipse cx="56" cy="74" rx="5" ry="7" fill={ACCENT} fillOpacity="0.4" />
        <ellipse cx="144" cy="74" rx="5" ry="7" fill={ACCENT} fillOpacity="0.4" />
        {/* 귀걸이 */}
        <circle cx="56" cy="86" r="4" fill={ACCENT} />
        <circle cx="144" cy="86" r="4" fill={ACCENT} />

        {/* 눈썹 */}
        <path
          d={emotion === 'concerned' ? 'M 74 55 Q 84 52 92 55' : emotion === 'happy' ? 'M 74 52 Q 84 48 92 52' : 'M 74 54 Q 84 50 92 54'}
          fill="none" stroke={HAIR} strokeWidth="3" strokeLinecap="round"
        />
        <path
          d={emotion === 'concerned' ? 'M 108 55 Q 116 52 126 55' : emotion === 'happy' ? 'M 108 52 Q 116 48 126 52' : 'M 108 54 Q 116 50 126 54'}
          fill="none" stroke={HAIR} strokeWidth="3" strokeLinecap="round"
        />

        {/* 눈 */}
        <ellipse cx="83" cy={eyeY} rx={emotion === 'alert' ? 11 : 9} ry={emotion === 'happy' ? 6 : 9} fill={DARK} />
        <ellipse cx="117" cy={eyeY} rx={emotion === 'alert' ? 11 : 9} ry={emotion === 'happy' ? 6 : 9} fill={DARK} />
        {/* 눈 흰자 */}
        <ellipse cx="83" cy={eyeY - 1} rx={emotion === 'happy' ? 7 : 6} ry={emotion === 'happy' ? 4 : 6} fill="white" />
        <ellipse cx="117" cy={eyeY - 1} rx={emotion === 'happy' ? 7 : 6} ry={emotion === 'happy' ? 4 : 6} fill="white" />
        {/* 홍채 */}
        <circle cx="83" cy={eyeY} r="4" fill={OUTFIT} style={{ animation: 'lunaEyeMove 6s ease-in-out infinite' }} />
        <circle cx="117" cy={eyeY} r="4" fill={OUTFIT} style={{ animation: 'lunaEyeMove 6s ease-in-out infinite' }} />
        {/* 눈동자 */}
        <circle cx="84" cy={eyeY - 1} r="2" fill={DARK} />
        <circle cx="118" cy={eyeY - 1} r="2" fill={DARK} />
        {/* 반짝임 */}
        <circle cx="86" cy={eyeY - 3} r="1.5" fill="white" />
        <circle cx="120" cy={eyeY - 3} r="1.5" fill="white" />
        {/* 속눈썹 */}
        {[-3,-1,1,3].map(dx => (
          <line key={dx} x1={83+dx} y1={eyeY-9} x2={82+dx*0.8} y2={eyeY-12}
            stroke={DARK} strokeWidth="1.5" strokeLinecap="round" />
        ))}
        {[-3,-1,1,3].map(dx => (
          <line key={dx} x1={117+dx} y1={eyeY-9} x2={116+dx*0.8} y2={eyeY-12}
            stroke={DARK} strokeWidth="1.5" strokeLinecap="round" />
        ))}
        {/* 볼 홍조 */}
        {(emotion === 'happy' || emotion === 'humorous') && (
          <>
            <ellipse cx="72" cy="82" rx="10" ry="6" fill={ACCENT} fillOpacity="0.35" />
            <ellipse cx="128" cy="82" rx="10" ry="6" fill={ACCENT} fillOpacity="0.35" />
          </>
        )}

        {/* 코 */}
        <path d="M 97 77 Q 95 83 98 85 Q 102 86 105 83 Q 108 81 106 77"
          fill="none" stroke={SKIN} strokeWidth="1.5" strokeOpacity="0.6" strokeLinecap="round" />

        {/* 입 */}
        <path
          d={smileD}
          fill={speaking ? SKIN : 'none'}
          stroke={DARK}
          strokeWidth="2"
          strokeLinecap="round"
        />
        {/* 말할 때 이 */}
        {speaking && <rect x="94" y="90" width="12" height="5" rx="1" fill="white" />}

        {/* 목 */}
        <rect x="88" y="106" width="24" height="20" rx="4" fill={SKIN} />
      </g>

      <style>{`
        @keyframes lunaBreathe {
          0%,100% { transform: scaleY(1); }
          50% { transform: scaleY(1.02) translateY(-2px); }
        }
        @keyframes lunaHeadBob {
          0%,100% { transform: translateY(0) rotate(0deg); }
          25% { transform: translateY(-3px) rotate(1deg); }
          75% { transform: translateY(1px) rotate(-1deg); }
        }
        @keyframes lunaHeadTilt {
          to { transform: rotate(-10deg) translateX(-5px); }
        }
        @keyframes lunaArmIdle {
          0%,100% { transform: rotate(0deg); }
          50% { transform: rotate(2deg); }
        }
        @keyframes lunaArmSpeakL {
          0%,100% { transform: rotate(0deg) translateY(0); }
          50% { transform: rotate(12deg) translateY(-6px); }
        }
        @keyframes lunaArmSpeakR {
          0%,100% { transform: rotate(0deg) translateY(0); }
          50% { transform: rotate(-10deg) translateY(-5px); }
        }
        @keyframes lunaArmListen {
          to { transform: rotate(-8deg); }
        }
        @keyframes lunaEyeMove {
          0%,40%,100% { transform: translateX(0); }
          50% { transform: translateX(2px); }
          70% { transform: translateX(-1px); }
        }
      `}</style>
    </svg>
  )
}
