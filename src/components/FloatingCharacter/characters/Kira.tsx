import type { CharacterProps } from './types'

/* 키라 — K-pop 아이돌 스타일 · 그라데이션 핑크/민트 헤어 */
export function Kira({ emotion, speaking, listening }: CharacterProps) {
  const SKIN   = '#fde4cc'
  const HAIR1  = '#ff6eb4'   // 핑크
  const HAIR2  = '#42d4d0'   // 민트
  const OUTFIT = '#ee2b7b'   // 마젠타
  const ACCENT = '#ffd700'   // 골드
  const DARK   = '#1a0a12'
  const SKIRT  = '#f9a8d4'

  const eyeOpen = emotion === 'happy' ? 5 : emotion === 'alert' ? 10 : 8
  const eyeY    = emotion === 'happy' ? 70 : 68

  const mouthD =
    speaking          ? 'M 86 91 Q 100 102 114 91'
    : emotion === 'happy'    ? 'M 82 89 Q 100 102 118 89'
    : emotion === 'concerned'? 'M 88 94 Q 100 88 112 94'
    : emotion === 'humorous' ? 'M 84 89 Q 100 100 116 89 Q 108 104 92 104 Z'
    :                          'M 86 90 Q 100 98 114 90'

  return (
    <svg viewBox="0 0 200 390" width="200" height="390" style={{ overflow: 'visible' }}>
      <defs>
        <radialGradient id="kiraSkin" cx="50%" cy="35%" r="65%">
          <stop offset="0%" stopColor="#fff3eb" />
          <stop offset="100%" stopColor={SKIN} />
        </radialGradient>
        <linearGradient id="kiraHair" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor={HAIR1} />
          <stop offset="100%" stopColor={HAIR2} />
        </linearGradient>
        <linearGradient id="kiraSkirt" x1="0%" y1="0%" x2="0%" y2="100%">
          <stop offset="0%" stopColor={SKIRT} />
          <stop offset="100%" stopColor="#fbb6d8" />
        </linearGradient>
        <linearGradient id="kiraOutfit" x1="0%" y1="0%" x2="0%" y2="100%">
          <stop offset="0%" stopColor={OUTFIT} />
          <stop offset="100%" stopColor="#c0185a" />
        </linearGradient>
        <filter id="kiraGlow">
          <feGaussianBlur stdDeviation="2" result="blur" />
          <feMerge><feMergeNode in="blur" /><feMergeNode in="SourceGraphic" /></feMerge>
        </filter>
      </defs>

      {/* ── 다리 ── */}
      <g>
        <rect x="62" y="270" width="28" height="100" rx="13" fill="#fde4cc" />
        <rect x="110" y="270" width="28" height="100" rx="13" fill="#fde4cc" />
        {/* 무릎 */}
        <ellipse cx="76" cy="300" rx="14" ry="10" fill="#fcd5bb" />
        <ellipse cx="124" cy="300" rx="14" ry="10" fill="#fcd5bb" />
        {/* 신발 — 하이힐 */}
        <path d="M 58 370 Q 62 380 82 380 Q 90 380 92 372 L 86 368 Q 82 376 70 376 Q 60 375 58 370 Z" fill={DARK} />
        <path d="M 108 370 Q 112 380 132 380 Q 140 380 142 372 L 136 368 Q 132 376 120 376 Q 110 375 108 370 Z" fill={DARK} />
        {/* 굽 */}
        <rect x="58" y="375" width="6" height="12" rx="2" fill={ACCENT} />
        <rect x="108" y="375" width="6" height="12" rx="2" fill={ACCENT} />
        {/* 발목 리본 */}
        <path d="M 58 370 Q 76 365 92 370" fill="none" stroke={HAIR1} strokeWidth="3" strokeLinecap="round" />
        <path d="M 108 370 Q 124 365 142 370" fill="none" stroke={HAIR2} strokeWidth="3" strokeLinecap="round" />
      </g>

      {/* ── 스커트 ── */}
      <path
        d="M 46 218 Q 38 260 44 280 L 62 278 Q 60 260 62 240 L 138 240 Q 140 260 138 278 L 156 280 Q 162 260 154 218 Z"
        fill="url(#kiraSkirt)"
        style={{ animation: 'kiraSkirtFlow 3s ease-in-out infinite', transformOrigin: '100px 240px' }}
      />
      {/* 스커트 레이스 */}
      {[0,1,2,3,4,5].map(i => (
        <path key={i}
          d={`M ${44 + i * 22} 280 Q ${55 + i * 22} 285 ${66 + i * 22} 280`}
          fill="none" stroke="white" strokeWidth="1.5" strokeOpacity="0.6"
        />
      ))}

      {/* ── 상체 ── */}
      <g style={{ animation: 'kiraBreathe 3.2s ease-in-out infinite', transformOrigin: '100px 175px' }}>
        {/* 탑 */}
        <path d="M 54 118 Q 44 138 46 218 L 154 218 Q 156 138 146 118 Z" fill="url(#kiraOutfit)" />
        {/* 별 장식 */}
        {[[82, 155], [118, 148], [100, 170]].map(([cx, cy], i) => (
          <polygon key={i}
            points={`${cx},${cy-6} ${cx+2},${cy-2} ${cx+6},${cy-2} ${cx+3},${cy+1} ${cx+4},${cy+5} ${cx},${cy+3} ${cx-4},${cy+5} ${cx-3},${cy+1} ${cx-6},${cy-2} ${cx-2},${cy-2}`}
            fill={ACCENT} fillOpacity="0.8"
            style={{ animation: `kiraStar${i} 2s ease-in-out infinite ${i * 0.4}s` }}
          />
        ))}
        {/* 네크라인 */}
        <path d="M 80 118 Q 100 128 120 118" fill="none" stroke={ACCENT} strokeWidth="2" />
        {/* 허리 벨트 */}
        <rect x="52" y="210" width="96" height="12" rx="6" fill="#c0185a" />
        <circle cx="100" cy="216" r="6" fill={ACCENT} />
      </g>

      {/* ── 왼팔 ── */}
      <g style={{
        transformOrigin: '50px 120px',
        animation: speaking
          ? 'kiraArmSpeakL 0.9s ease-in-out infinite'
          : listening
          ? 'kiraArmListen 0.4s ease forwards'
          : 'kiraArmIdle 4s ease-in-out infinite',
      }}>
        <path d="M 50 118 Q 34 138 28 190 Q 40 198 54 192 Q 56 148 66 122 Z" fill={OUTFIT} />
        {/* 팔 밴드 */}
        <ellipse cx="40" cy="160" rx="8" ry="5" fill={ACCENT} fillOpacity="0.6" transform="rotate(-15,40,160)" />
        {/* 손 */}
        <ellipse cx="36" cy="196" rx="12" ry="10" fill="url(#kiraSkin)" />
        {/* 손가락 */}
        <path d="M 28 193 Q 26 187 30 185" stroke={SKIN} strokeWidth="2.5" strokeLinecap="round" />
        <path d="M 33 191 Q 31 184 35 182" stroke={SKIN} strokeWidth="2.5" strokeLinecap="round" />
        <path d="M 39 191 Q 38 184 41 183" stroke={SKIN} strokeWidth="2.5" strokeLinecap="round" />
        {/* 팔찌 */}
        <ellipse cx="36" cy="193" rx="12" ry="5" fill="none" stroke={ACCENT} strokeWidth="2" />
      </g>

      {/* ── 오른팔 ── */}
      <g style={{
        transformOrigin: '150px 120px',
        animation: speaking
          ? 'kiraArmSpeakR 0.9s ease-in-out infinite 0.3s'
          : 'kiraArmIdle 4s ease-in-out infinite 2s',
      }}>
        <path d="M 150 118 Q 166 138 172 190 Q 160 198 146 192 Q 144 148 134 122 Z" fill={OUTFIT} />
        <ellipse cx="160" cy="160" rx="8" ry="5" fill={ACCENT} fillOpacity="0.6" transform="rotate(15,160,160)" />
        <ellipse cx="164" cy="196" rx="12" ry="10" fill="url(#kiraSkin)" />
        <path d="M 172 193 Q 174 187 170 185" stroke={SKIN} strokeWidth="2.5" strokeLinecap="round" />
        <path d="M 167 191 Q 169 184 165 182" stroke={SKIN} strokeWidth="2.5" strokeLinecap="round" />
        <path d="M 161 191 Q 162 184 159 183" stroke={SKIN} strokeWidth="2.5" strokeLinecap="round" />
        <ellipse cx="164" cy="193" rx="12" ry="5" fill="none" stroke={ACCENT} strokeWidth="2" />
      </g>

      {/* ── 머리 ── */}
      <g style={{
        transformOrigin: '100px 68px',
        animation: listening
          ? 'kiraHeadTilt 0.4s ease forwards'
          : 'kiraHeadBob 4s ease-in-out infinite',
      }}>
        {/* 긴 트윈테일 */}
        <path d="M 56 72 Q 28 100 22 180 Q 34 188 44 178 Q 40 120 60 88 Z" fill="url(#kiraHair)" />
        <path d="M 144 72 Q 172 100 178 180 Q 166 188 156 178 Q 160 120 140 88 Z" fill="url(#kiraHair)" />
        {/* 헤어 뒤 */}
        <ellipse cx="100" cy="62" rx="46" ry="50" fill="url(#kiraHair)" />
        {/* 얼굴 */}
        <ellipse cx="100" cy="58" rx="42" ry="44" fill="url(#kiraSkin)" />
        {/* 앞머리 */}
        <path d="M 58 38 Q 72 14 100 18 Q 128 14 142 38 Q 128 24 100 26 Q 72 24 58 38 Z" fill="url(#kiraHair)" />
        <path d="M 58 40 Q 64 26 74 24 Q 68 34 66 46 Z" fill="url(#kiraHair)" />
        <path d="M 142 40 Q 136 26 126 24 Q 132 34 134 46 Z" fill="url(#kiraHair)" />
        {/* 헤어핀 별 */}
        <polygon points="76,30 78,26 80,30 84,30 81,33 82,37 78,35 74,37 75,33 72,30"
          fill={ACCENT} filter="url(#kiraGlow)" />
        <polygon points="124,26 126,22 128,26 132,26 129,29 130,33 126,31 122,33 123,29 120,26"
          fill={HAIR2} filter="url(#kiraGlow)" />

        {/* 귀 */}
        <ellipse cx="57" cy="70" rx="8" ry="10" fill={SKIN} />
        <ellipse cx="143" cy="70" rx="8" ry="10" fill={SKIN} />
        {/* 귀걸이 — 별 */}
        <polygon points="57,83 59,79 61,83 65,83 62,86 63,90 59,88 55,90 56,86 53,83"
          fill={HAIR1} style={{ filter: 'drop-shadow(0 0 3px rgba(255,110,180,0.8))' }} />
        <polygon points="143,83 145,79 147,83 151,83 148,86 149,90 145,88 141,90 142,86 139,83"
          fill={HAIR2} style={{ filter: 'drop-shadow(0 0 3px rgba(66,212,208,0.8))' }} />

        {/* 눈썹 */}
        <path
          d={emotion === 'concerned' ? 'M 74 52 Q 84 49 92 52' : 'M 72 50 Q 83 45 92 50'}
          fill="none" stroke={HAIR1} strokeWidth="3" strokeLinecap="round"
        />
        <path
          d={emotion === 'concerned' ? 'M 108 52 Q 116 49 126 52' : 'M 108 50 Q 117 45 128 50'}
          fill="none" stroke={HAIR1} strokeWidth="3" strokeLinecap="round"
        />

        {/* 눈 */}
        <ellipse cx="83" cy={eyeY} rx="11" ry={eyeOpen} fill={DARK} />
        <ellipse cx="117" cy={eyeY} rx="11" ry={eyeOpen} fill={DARK} />
        {/* 눈 그라데이션 */}
        <ellipse cx="83" cy={eyeY} rx="9" ry={eyeOpen - 1} fill={HAIR1} fillOpacity="0.7" />
        <ellipse cx="117" cy={eyeY} rx="9" ry={eyeOpen - 1} fill={HAIR2} fillOpacity="0.7" />
        {/* 눈동자 */}
        <circle cx="83" cy={eyeY} r="5" fill={DARK} />
        <circle cx="117" cy={eyeY} r="5" fill={DARK} />
        {/* 하이라이트 */}
        <circle cx="85" cy={eyeY - 3} r="2.5" fill="white" />
        <circle cx="119" cy={eyeY - 3} r="2.5" fill="white" />
        <circle cx="80" cy={eyeY + 2} r="1" fill="white" fillOpacity="0.5" />
        {/* 속눈썹 */}
        {[-4,-2,0,2,4].map(dx => (
          <line key={dx} x1={83+dx} y1={eyeY-eyeOpen} x2={82+dx*0.9} y2={eyeY-eyeOpen-4}
            stroke={DARK} strokeWidth="1.5" strokeLinecap="round" />
        ))}
        {[-4,-2,0,2,4].map(dx => (
          <line key={dx} x1={117+dx} y1={eyeY-eyeOpen} x2={116+dx*0.9} y2={eyeY-eyeOpen-4}
            stroke={DARK} strokeWidth="1.5" strokeLinecap="round" />
        ))}

        {/* 볼 홍조 */}
        <ellipse cx="70" cy="78" rx="11" ry="7" fill={HAIR1} fillOpacity="0.25" />
        <ellipse cx="130" cy="78" rx="11" ry="7" fill={HAIR1} fillOpacity="0.25" />

        {/* 코 */}
        <path d="M 97 77 Q 95 82 98 84 Q 102 85 105 82 Q 107 80 106 77"
          fill="none" stroke={SKIN} strokeWidth="1.5" strokeOpacity="0.5" />

        {/* 입 */}
        <path d={mouthD}
          fill={speaking ? '#ffe0ef' : 'none'}
          stroke={DARK} strokeWidth="2" strokeLinecap="round"
        />
        {speaking && <rect x="94" y="91" width="12" height="6" rx="1.5" fill="white" />}

        {/* 목 */}
        <rect x="88" y="100" width="24" height="20" rx="4" fill={SKIN} />
        {/* 목걸이 */}
        <path d="M 82 115 Q 100 122 118 115" fill="none" stroke={ACCENT} strokeWidth="2" />
        <circle cx="100" cy="122" r="4" fill={ACCENT} />
      </g>

      {/* 글리터 파티클 (항상) */}
      {speaking && [
        [60, 60], [140, 50], [170, 110], [30, 120],
      ].map(([x, y], i) => (
        <circle key={i} cx={x} cy={y} r="3" fill={i % 2 === 0 ? ACCENT : HAIR1}
          style={{ animation: `kiraGlitter${i} ${0.8 + i * 0.2}s ease-in-out infinite` }}
          fillOpacity="0.7"
        />
      ))}

      <style>{`
        @keyframes kiraBreathe {
          0%,100% { transform: scaleY(1); }
          50% { transform: scaleY(1.015) translateY(-1.5px); }
        }
        @keyframes kiraHeadBob {
          0%,100% { transform: translateY(0) rotate(0deg); }
          30% { transform: translateY(-4px) rotate(1.5deg); }
          70% { transform: translateY(1px) rotate(-1deg); }
        }
        @keyframes kiraHeadTilt {
          to { transform: rotate(-12deg) translateX(-6px); }
        }
        @keyframes kiraArmIdle {
          0%,100% { transform: rotate(0deg); }
          50% { transform: rotate(2.5deg); }
        }
        @keyframes kiraArmSpeakL {
          0%,100% { transform: rotate(-5deg) translateY(0); }
          50% { transform: rotate(14deg) translateY(-8px); }
        }
        @keyframes kiraArmSpeakR {
          0%,100% { transform: rotate(5deg) translateY(0); }
          50% { transform: rotate(-12deg) translateY(-7px); }
        }
        @keyframes kiraArmListen {
          to { transform: rotate(-10deg); }
        }
        @keyframes kiraSkirtFlow {
          0%,100% { transform: skewX(0deg); }
          50% { transform: skewX(1.5deg); }
        }
        @keyframes kiraStar0 {
          0%,100% { opacity: 0.8; transform: scale(1) rotate(0deg); }
          50% { opacity: 1; transform: scale(1.2) rotate(20deg); }
        }
        @keyframes kiraStar1 {
          0%,100% { opacity: 0.7; transform: scale(1) rotate(0deg); }
          50% { opacity: 1; transform: scale(1.15) rotate(-15deg); }
        }
        @keyframes kiraStar2 {
          0%,100% { opacity: 0.6; transform: scale(1) rotate(0deg); }
          50% { opacity: 1; transform: scale(1.1) rotate(25deg); }
        }
        @keyframes kiraGlitter0 {
          0%,100% { transform: translate(0,0); opacity: 0; }
          50% { transform: translate(-10px,-20px); opacity: 0.8; }
        }
        @keyframes kiraGlitter1 {
          0%,100% { transform: translate(0,0); opacity: 0; }
          50% { transform: translate(8px,-18px); opacity: 0.7; }
        }
        @keyframes kiraGlitter2 {
          0%,100% { transform: translate(0,0); opacity: 0; }
          50% { transform: translate(-5px,-22px); opacity: 0.9; }
        }
        @keyframes kiraGlitter3 {
          0%,100% { transform: translate(0,0); opacity: 0; }
          50% { transform: translate(12px,-15px); opacity: 0.6; }
        }
      `}</style>
    </svg>
  )
}
