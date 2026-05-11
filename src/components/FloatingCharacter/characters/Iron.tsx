import type { CharacterProps } from './types'

/* 아이언 — SF 미래형 홀로그래픽 AI */
export function Iron({ emotion, speaking, listening }: CharacterProps) {
  const PRIMARY = '#4f7ef7'
  const DARK    = '#1a1a3a'
  const SILVER  = '#8899bb'
  const LIGHT   = '#c8d8ff'
  const GLOW    = speaking ? '#7dd3fc' : emotion === 'alert' ? '#ef4444' : PRIMARY

  /* 입 모양 */
  const mouthPath =
    speaking         ? 'M 82 148 Q 100 156 118 148'
    : emotion === 'happy'    ? 'M 82 146 Q 100 154 118 146'
    : emotion === 'concerned'? 'M 85 150 Q 100 145 115 150'
    : emotion === 'alert'    ? 'M 85 147 L 115 147'
    :                          'M 86 148 Q 100 152 114 148'

  return (
    <svg
      viewBox="0 0 200 380"
      width="200"
      height="380"
      style={{ overflow: 'visible', filter: speaking ? `drop-shadow(0 0 12px ${GLOW}44)` : 'none' }}
    >
      <defs>
        <radialGradient id="ironBody" cx="50%" cy="30%" r="70%">
          <stop offset="0%" stopColor="#2a3560" />
          <stop offset="100%" stopColor={DARK} />
        </radialGradient>
        <radialGradient id="ironCore" cx="50%" cy="50%" r="50%">
          <stop offset="0%" stopColor={GLOW} stopOpacity="0.9" />
          <stop offset="100%" stopColor={GLOW} stopOpacity="0.1" />
        </radialGradient>
        <filter id="glow">
          <feGaussianBlur stdDeviation="2" result="blur" />
          <feMerge><feMergeNode in="blur" /><feMergeNode in="SourceGraphic" /></feMerge>
        </filter>
      </defs>

      {/* ── 몸통 (호흡 애니메이션) ── */}
      <g style={{
        transformOrigin: '100px 200px',
        animation: 'ironBreathe 3s ease-in-out infinite',
      }}>
        {/* 어깨 연결부 */}
        <rect x="38" y="108" width="124" height="12" rx="6" fill={SILVER} />

        {/* 흉부 장갑 */}
        <path d="M 55 118 L 145 118 L 148 210 Q 100 215 52 210 Z" fill="url(#ironBody)" />
        {/* 흉부 디테일 라인 */}
        <line x1="100" y1="118" x2="100" y2="210" stroke={PRIMARY} strokeWidth="1" strokeOpacity="0.4" />
        <line x1="55" y1="155" x2="145" y2="155" stroke={PRIMARY} strokeWidth="1" strokeOpacity="0.4" />

        {/* 흉부 코어 */}
        <circle cx="100" cy="158" r="16" fill="url(#ironCore)" filter="url(#glow)" />
        <circle cx="100" cy="158" r="10" fill="none" stroke={GLOW} strokeWidth="1.5" />
        <circle cx="100" cy="158" r="4" fill={GLOW} />

        {/* 코어 회전 링 */}
        <circle cx="100" cy="158" r="13" fill="none" stroke={GLOW} strokeWidth="0.8"
          strokeDasharray="5 3" strokeOpacity="0.6"
          style={{ transformOrigin: '100px 158px', animation: 'ironRotate 4s linear infinite' }}
        />

        {/* 복부 */}
        <rect x="65" y="210" width="70" height="30" rx="4" fill={DARK} />
        <rect x="73" y="215" width="54" height="6" rx="2" fill={SILVER} fillOpacity="0.3" />
        <rect x="73" y="225" width="54" height="6" rx="2" fill={SILVER} fillOpacity="0.3" />
      </g>

      {/* ── 왼팔 (리스닝 시 앞으로) ── */}
      <g style={{
        transformOrigin: '43px 118px',
        animation: listening
          ? 'ironArmListenL 0.5s ease forwards'
          : speaking
          ? 'ironArmSpeakL 1.2s ease-in-out infinite'
          : 'none',
      }}>
        <rect x="20" y="110" width="28" height="88" rx="8" fill={DARK} />
        <rect x="22" y="112" width="24" height="84" rx="7" fill={SILVER} fillOpacity="0.15" />
        {/* 팔꿈치 조인트 */}
        <circle cx="34" cy="165" r="5" fill={SILVER} />
        {/* 손 */}
        <ellipse cx="34" cy="206" rx="13" ry="10" fill={DARK} />
        <ellipse cx="34" cy="206" rx="10" ry="7" fill={SILVER} fillOpacity="0.2" />
        {/* 손가락 라인 */}
        {[0,1,2].map(i => (
          <line key={i} x1={28 + i*4} y1="200" x2={27 + i*4} y2="212"
            stroke={SILVER} strokeWidth="1.5" strokeOpacity="0.5" strokeLinecap="round" />
        ))}
      </g>

      {/* ── 오른팔 ── */}
      <g style={{
        transformOrigin: '157px 118px',
        animation: speaking
          ? 'ironArmSpeakR 1.2s ease-in-out infinite 0.3s'
          : 'none',
      }}>
        <rect x="152" y="110" width="28" height="88" rx="8" fill={DARK} />
        <rect x="154" y="112" width="24" height="84" rx="7" fill={SILVER} fillOpacity="0.15" />
        <circle cx="166" cy="165" r="5" fill={SILVER} />
        <ellipse cx="166" cy="206" rx="13" ry="10" fill={DARK} />
        <ellipse cx="166" cy="206" rx="10" ry="7" fill={SILVER} fillOpacity="0.2" />
        {[0,1,2].map(i => (
          <line key={i} x1={160 + i*4} y1="200" x2={159 + i*4} y2="212"
            stroke={SILVER} strokeWidth="1.5" strokeOpacity="0.5" strokeLinecap="round" />
        ))}
      </g>

      {/* ── 다리 ── */}
      <g>
        <rect x="62" y="238" width="32" height="106" rx="8" fill={DARK} />
        <rect x="106" y="238" width="32" height="106" rx="8" fill={DARK} />
        {/* 무릎 조인트 */}
        <circle cx="78" cy="290" r="6" fill={SILVER} fillOpacity="0.5" />
        <circle cx="122" cy="290" r="6" fill={SILVER} fillOpacity="0.5" />
        {/* 부츠 */}
        <rect x="55" y="330" width="46" height="22" rx="5" fill={SILVER} fillOpacity="0.4" />
        <rect x="99" y="330" width="46" height="22" rx="5" fill={SILVER} fillOpacity="0.4" />
        {/* 부츠 글로우 */}
        <rect x="58" y="346" width="40" height="3" rx="2" fill={GLOW} fillOpacity="0.6" />
        <rect x="102" y="346" width="40" height="3" rx="2" fill={GLOW} fillOpacity="0.6" />
      </g>

      {/* ── 머리 (리스닝 시 기울기) ── */}
      <g style={{
        transformOrigin: '100px 75px',
        animation: listening
          ? 'ironHeadTilt 0.4s ease forwards'
          : 'ironHeadIdle 4s ease-in-out infinite',
      }}>
        {/* 헬멧 */}
        <ellipse cx="100" cy="62" rx="44" ry="48" fill="url(#ironBody)" />
        <ellipse cx="100" cy="58" rx="40" ry="42" fill={DARK} fillOpacity="0.6" />

        {/* 헬멧 상단 에너지 핀 */}
        <rect x="96" y="14" width="8" height="16" rx="3" fill={GLOW} filter="url(#glow)" />
        <ellipse cx="100" cy="14" rx="6" ry="4" fill={GLOW} filter="url(#glow)" />

        {/* 헬멧 사이드 디테일 */}
        <rect x="56" y="52" width="8" height="20" rx="3" fill={SILVER} fillOpacity="0.4" />
        <rect x="136" y="52" width="8" height="20" rx="3" fill={SILVER} fillOpacity="0.4" />

        {/* 바이저 (눈 영역) */}
        <rect x="66" y="46" width="68" height="28" rx="8" fill="#050510" />

        {/* 눈 */}
        <rect
          x="72" y="51"
          width={emotion === 'alert' ? 22 : 18}
          height={emotion === 'happy' ? 5 : emotion === 'alert' ? 12 : 8}
          rx="3"
          fill={GLOW}
          filter="url(#glow)"
          style={{ animation: speaking ? 'ironEyeBlink 2s ease-in-out infinite 0.5s' : 'none' }}
        />
        <rect
          x="110" y="51"
          width={emotion === 'alert' ? 22 : 18}
          height={emotion === 'happy' ? 5 : emotion === 'alert' ? 12 : 8}
          rx="3"
          fill={GLOW}
          filter="url(#glow)"
          style={{ animation: speaking ? 'ironEyeBlink 2s ease-in-out infinite 0.7s' : 'none' }}
        />
        {/* 눈 반사광 */}
        <rect x="74" y="53" width="6" height="2" rx="1" fill={LIGHT} fillOpacity="0.5" />
        <rect x="112" y="53" width="6" height="2" rx="1" fill={LIGHT} fillOpacity="0.5" />

        {/* 마이크/입 영역 */}
        <rect x="80" y="78" width="40" height="14" rx="5" fill="#050510" />
        <path
          d={mouthPath}
          fill="none"
          stroke={GLOW}
          strokeWidth="2.5"
          strokeLinecap="round"
          style={{ transition: 'd 0.15s ease' }}
        />
        {/* 말하는 중 입 진동 */}
        {speaking && [0,1,2,3,4].map(i => (
          <rect key={i}
            x={85 + i * 7} y="82"
            width="4"
            height={2 + Math.sin(i) * 2}
            rx="1"
            fill={GLOW}
            fillOpacity="0.7"
            style={{ animation: `ironVoice${i} 0.4s ease-in-out infinite ${i * 0.06}s` }}
          />
        ))}

        {/* 턱 */}
        <ellipse cx="100" cy="105" rx="32" ry="8" fill={DARK} fillOpacity="0.8" />
        {/* 목 */}
        <rect x="86" y="104" width="28" height="16" rx="4" fill={SILVER} fillOpacity="0.3" />
      </g>

      {/* ── CSS 애니메이션 ── */}
      <style>{`
        @keyframes ironBreathe {
          0%,100% { transform: scaleY(1); }
          50% { transform: scaleY(1.015); }
        }
        @keyframes ironHeadIdle {
          0%,100% { transform: translateY(0) rotate(0deg); }
          30% { transform: translateY(-2px) rotate(0.5deg); }
          60% { transform: translateY(1px) rotate(-0.5deg); }
        }
        @keyframes ironHeadTilt {
          to { transform: rotate(-8deg) translateX(-4px); }
        }
        @keyframes ironRotate {
          from { transform: rotate(0deg); }
          to { transform: rotate(360deg); }
        }
        @keyframes ironArmSpeakL {
          0%,100% { transform: rotate(0deg); }
          50% { transform: rotate(8deg) translateY(-4px); }
        }
        @keyframes ironArmSpeakR {
          0%,100% { transform: rotate(0deg); }
          50% { transform: rotate(-8deg) translateY(-4px); }
        }
        @keyframes ironArmListenL {
          to { transform: rotate(-12deg); }
        }
        @keyframes ironEyeBlink {
          0%,90%,100% { transform: scaleY(1); }
          95% { transform: scaleY(0.1); }
        }
        ${[0,1,2,3,4].map(i => `
          @keyframes ironVoice${i} {
            0%,100% { transform: scaleY(1); }
            50% { transform: scaleY(${1.5 + Math.random() * 1.5}); }
          }
        `).join('')}
      `}</style>
    </svg>
  )
}
