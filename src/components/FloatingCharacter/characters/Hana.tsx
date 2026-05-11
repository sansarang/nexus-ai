import type { CharacterProps } from './types'

/**
 * 하나 — 따뜻하고 현실적인 스타일
 * 갈색/골드 웨이브 헤어 + 코지 카디건 + 스커트
 * 친근하고 부드러운 분위기
 */
export function Hana({ emotion, speaking, listening }: CharacterProps) {
  const SKIN    = '#f5d5b8'
  const HAIR    = '#8b4513'   // 웜 갈색
  const HAIR2   = '#c97d3e'   // 골든 브라운 하이라이트
  const CARDI   = '#d4956a'   // 카디건 테라코타
  const INNER   = '#fff8f0'   // 이너 블라우스
  const SKIRT   = '#6b5b4e'   // 다크 브라운 스커트
  const DARK    = '#2c1a08'
  const BLUSH   = '#f4a88a'
  const GREEN   = '#8bc34a'   // 포인트 그린

  const eyeY  = emotion === 'happy' ? 66 : 64
  const eyeRy = emotion === 'happy' ? 5 : 8

  const mouthD =
    speaking           ? 'M 86 89 Q 100 101 114 89'
    : emotion === 'happy'    ? 'M 83 88 Q 100 101 117 88'
    : emotion === 'concerned'? 'M 88 93 Q 100 87 112 93'
    : emotion === 'humorous' ? 'M 85 88 Q 100 99 115 88 Q 108 104 92 104 Z'
    :                          'M 87 88 Q 100 97 113 88'

  return (
    <svg viewBox="0 0 200 410" width="200" height="410" style={{ overflow: 'visible' }}>
      <defs>
        <radialGradient id="hanaSkin" cx="50%" cy="35%" r="60%">
          <stop offset="0%" stopColor="#fff3e8" />
          <stop offset="100%" stopColor={SKIN} />
        </radialGradient>
        <linearGradient id="hanaHair" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor={HAIR} />
          <stop offset="40%" stopColor={HAIR2} />
          <stop offset="100%" stopColor={HAIR} />
        </linearGradient>
        <linearGradient id="hanaCardi" x1="0%" y1="0%" x2="0%" y2="100%">
          <stop offset="0%" stopColor={CARDI} />
          <stop offset="100%" stopColor="#b87350" />
        </linearGradient>
        <linearGradient id="hanaSkirt" x1="0%" y1="0%" x2="0%" y2="100%">
          <stop offset="0%" stopColor={SKIRT} />
          <stop offset="100%" stopColor="#4a3c32" />
        </linearGradient>
      </defs>

      {/* ── 다리 + 양말 + 신발 ── */}
      <g>
        {/* 레그 */}
        <rect x="64" y="268" width="28" height="115" rx="12" fill={SKIN} />
        <rect x="108" y="268" width="28" height="115" rx="12" fill={SKIN} />
        {/* 무릎 */}
        <ellipse cx="78" cy="298" rx="13" ry="9" fill="#e8c09a" />
        <ellipse cx="122" cy="298" rx="13" ry="9" fill="#e8c09a" />
        {/* 양말 (루스 삭스) */}
        <rect x="62" y="345" width="32" height="28" rx="8" fill="white" />
        <rect x="106" y="345" width="32" height="28" rx="8" fill="white" />
        {/* 양말 패턴 */}
        <path d="M 64 350 Q 78 347 92 350" fill="none" stroke="#e0e0e0" strokeWidth="1" />
        <path d="M 64 354 Q 78 351 92 354" fill="none" stroke="#e0e0e0" strokeWidth="1" />
        <path d="M 108 350 Q 122 347 136 350" fill="none" stroke="#e0e0e0" strokeWidth="1" />
        <path d="M 108 354 Q 122 351 136 354" fill="none" stroke="#e0e0e0" strokeWidth="1" />
        {/* 양말 접힌 부분 */}
        <path d="M 62 347 Q 78 342 94 347 Q 78 353 62 347 Z" fill="white" stroke="#d0d0d0" strokeWidth="0.5" />
        <path d="M 106 347 Q 122 342 138 347 Q 122 353 106 347 Z" fill="white" stroke="#d0d0d0" strokeWidth="0.5" />
        {/* 로퍼 신발 */}
        <path d="M 60 368 Q 62 382 84 382 Q 96 382 96 374 L 90 370 Q 88 378 74 378 Q 62 376 60 368 Z" fill={DARK} />
        <path d="M 104 368 Q 106 382 128 382 Q 140 382 140 374 L 134 370 Q 132 378 118 378 Q 106 376 104 368 Z" fill={DARK} />
        {/* 신발 버클 */}
        <rect x="68" y="372" width="20" height="6" rx="3" fill={HAIR2} />
        <rect x="112" y="372" width="20" height="6" rx="3" fill={HAIR2} />
      </g>

      {/* ── 스커트 ── */}
      <g style={{ animation: 'hanaSway 4s ease-in-out infinite', transformOrigin: '100px 245px' }}>
        <path d="M 46 218 Q 38 248 46 268 L 64 266 Q 62 252 64 240 L 136 240 Q 138 252 136 266 L 154 268 Q 162 248 154 218 Z"
          fill="url(#hanaSkirt)" />
        {/* 스커트 주름 */}
        {[58, 72, 86, 100, 114, 128, 142].map(x => (
          <line key={x} x1={x} y1={218} x2={x < 100 ? x - 4 : x + 4} y2={268}
            stroke="rgba(255,255,255,0.12)" strokeWidth="1" />
        ))}
        {/* 스커트 아랫단 */}
        <path d="M 46 267 Q 100 278 154 267" fill="none" stroke="rgba(255,255,255,0.2)" strokeWidth="1.5" />
      </g>

      {/* ── 상체 카디건 ── */}
      <g style={{ animation: 'hanaBreathe 3.8s ease-in-out infinite', transformOrigin: '100px 170px' }}>
        {/* 이너 블라우스 */}
        <path d="M 68 118 Q 60 130 62 220 L 138 220 Q 140 130 132 118 Z" fill={INNER} />
        {/* 카디건 */}
        <path d="M 52 116 Q 42 136 44 220 L 68 220 L 68 118 Z" fill="url(#hanaCardi)" />
        <path d="M 148 116 Q 158 136 156 220 L 132 220 L 132 118 Z" fill="url(#hanaCardi)" />
        {/* 카디건 단추 */}
        {[140, 160, 180, 200].map(y => (
          <circle key={y} cx="100" cy={y} r="3.5" fill={HAIR2} stroke={CARDI} strokeWidth="0.5" />
        ))}
        {/* 카디건 소매 */}
        <path d="M 52 116 L 44 118 Q 44 136 56 130 L 52 116 Z" fill="url(#hanaCardi)" />
        <path d="M 148 116 L 156 118 Q 156 136 144 130 L 148 116 Z" fill="url(#hanaCardi)" />
        {/* 이너 블라우스 칼라 */}
        <path d="M 82 118 Q 100 130 118 118" fill={INNER} stroke="#ddd" strokeWidth="1" />
        {/* 칼라 레이스 */}
        <path d="M 80 122 Q 100 133 120 122" fill="none" stroke="#ccc" strokeWidth="1" strokeDasharray="3,2" />
      </g>

      {/* ── 왼팔 ── */}
      <g style={{
        transformOrigin: '48px 120px',
        animation: speaking
          ? 'hanaArmSpeakL 1.1s ease-in-out infinite'
          : listening
          ? 'hanaArmListen 0.4s ease forwards'
          : 'hanaArmIdle 4s ease-in-out infinite',
      }}>
        <path d="M 50 116 Q 32 136 26 188 Q 38 198 54 192 Q 54 148 64 120 Z" fill="url(#hanaCardi)" />
        {/* 소매 커프 */}
        <path d="M 27 184 Q 38 192 54 188 Q 52 178 38 174 Z" fill="#b87350" />
        {/* 손 */}
        <ellipse cx="34" cy="196" rx="13" ry="11" fill="url(#hanaSkin)" />
        <path d="M 26 193 Q 24 187 28 185" stroke={SKIN} strokeWidth="2.5" strokeLinecap="round" />
        <path d="M 31 191 Q 29 184 33 182" stroke={SKIN} strokeWidth="2.5" strokeLinecap="round" />
        <path d="M 37 190 Q 36 183 39 182" stroke={SKIN} strokeWidth="2.5" strokeLinecap="round" />
        {/* 반지 */}
        <circle cx="30" cy="192" r="3" fill="none" stroke={HAIR2} strokeWidth="1.5" />
      </g>

      {/* ── 오른팔 ── */}
      <g style={{
        transformOrigin: '152px 120px',
        animation: speaking
          ? 'hanaArmSpeakR 1.1s ease-in-out infinite 0.4s'
          : 'hanaArmIdle 4s ease-in-out infinite 2s',
      }}>
        <path d="M 150 116 Q 168 136 174 188 Q 162 198 146 192 Q 146 148 136 120 Z" fill="url(#hanaCardi)" />
        <path d="M 173 184 Q 162 192 146 188 Q 148 178 162 174 Z" fill="#b87350" />
        <ellipse cx="166" cy="196" rx="13" ry="11" fill="url(#hanaSkin)" />
        <path d="M 174 193 Q 176 187 172 185" stroke={SKIN} strokeWidth="2.5" strokeLinecap="round" />
        <path d="M 169 191 Q 171 184 167 182" stroke={SKIN} strokeWidth="2.5" strokeLinecap="round" />
        <path d="M 163 190 Q 164 183 161 182" stroke={SKIN} strokeWidth="2.5" strokeLinecap="round" />
        <circle cx="170" cy="192" r="3" fill="none" stroke={HAIR2} strokeWidth="1.5" />
      </g>

      {/* ── 머리 ── */}
      <g style={{
        transformOrigin: '100px 62px',
        animation: listening
          ? 'hanaHeadTilt 0.4s ease forwards'
          : 'hanaHeadBob 5s ease-in-out infinite',
      }}>
        {/* 웨이브 긴 머리 */}
        {/* 뒷머리 */}
        <ellipse cx="100" cy="60" rx="47" ry="52" fill="url(#hanaHair)" />
        {/* 긴 웨이브 사이드 */}
        <g style={{ animation: 'hanaHairR 5s ease-in-out infinite 1s', transformOrigin: '150px 100px' }}>
          <path d="M 144 68 Q 172 90 168 170 Q 168 200 158 220 Q 148 230 144 225 Q 150 205 152 170 Q 154 100 144 82 Z"
            fill="url(#hanaHair)" />
          <path d="M 155 100 Q 160 130 157 165" fill="none" stroke={HAIR2} strokeWidth="2.5" strokeOpacity="0.5" />
        </g>
        <g style={{ animation: 'hanaHairL 5s ease-in-out infinite', transformOrigin: '50px 100px' }}>
          <path d="M 56 68 Q 28 90 32 170 Q 32 200 42 220 Q 52 230 56 225 Q 50 205 48 170 Q 46 100 56 82 Z"
            fill="url(#hanaHair)" />
          <path d="M 45 100 Q 40 130 43 165" fill="none" stroke={HAIR2} strokeWidth="2.5" strokeOpacity="0.5" />
        </g>
        {/* 얼굴 */}
        <ellipse cx="100" cy="58" rx="42" ry="44" fill="url(#hanaSkin)" />
        {/* 앞머리 자연스럽게 */}
        <path d="M 60 36 Q 76 12 100 16 Q 124 12 140 36 Q 124 20 100 22 Q 76 20 60 36 Z" fill="url(#hanaHair)" />
        <path d="M 60 38 Q 68 20 78 18 Q 72 30 70 44 Z" fill="url(#hanaHair)" />
        <path d="M 140 38 Q 132 20 122 18 Q 128 30 130 44 Z" fill="url(#hanaHair)" />
        {/* 앞머리 자연스러운 끝 */}
        <path d="M 82 18 Q 80 8 84 4" fill="none" stroke="url(#hanaHair)" strokeWidth="4" strokeLinecap="round" />
        <path d="M 90 16 Q 90 6 92 2" fill="none" stroke="url(#hanaHair)" strokeWidth="4" strokeLinecap="round" />
        {/* 헤어 하이라이트 */}
        <path d="M 76 22 Q 84 16 94 20" fill="none" stroke={HAIR2} strokeWidth="2.5" strokeOpacity="0.5" />
        <path d="M 106 20 Q 116 16 124 22" fill="none" stroke={HAIR2} strokeWidth="2.5" strokeOpacity="0.5" />

        {/* 귀 */}
        <ellipse cx="57" cy="66" rx="8" ry="10" fill={SKIN} />
        <ellipse cx="143" cy="66" rx="8" ry="10" fill={SKIN} />
        {/* 작은 꽃 귀걸이 */}
        <circle cx="57" cy="77" r="5" fill={HAIR2} />
        <circle cx="57" cy="77" r="2.5" fill={INNER} />
        <circle cx="143" cy="77" r="5" fill={GREEN} />
        <circle cx="143" cy="77" r="2.5" fill={INNER} />

        {/* 눈썹 (자연스러운) */}
        <path d={emotion === 'happy' ? 'M 73 48 Q 83 44 92 48' : emotion === 'concerned' ? 'M 73 52 Q 83 49 92 53' : 'M 73 50 Q 83 46 92 50'}
          fill="none" stroke={HAIR} strokeWidth="2.5" strokeLinecap="round" />
        <path d={emotion === 'happy' ? 'M 108 48 Q 117 44 127 48' : emotion === 'concerned' ? 'M 108 53 Q 117 49 127 52' : 'M 108 50 Q 117 46 127 50'}
          fill="none" stroke={HAIR} strokeWidth="2.5" strokeLinecap="round" />

        {/* 눈 (따뜻하고 자연스러운) */}
        <ellipse cx="83" cy={eyeY} rx="11" ry={eyeRy + 1} fill={DARK} />
        <ellipse cx="117" cy={eyeY} rx="11" ry={eyeRy + 1} fill={DARK} />
        {/* 흰자 */}
        <ellipse cx="83" cy={eyeY} rx="9.5" ry={eyeRy} fill="#fffef8" />
        <ellipse cx="117" cy={eyeY} rx="9.5" ry={eyeRy} fill="#fffef8" />
        {/* 홍채 — 따뜻한 갈색 */}
        <ellipse cx="83" cy={eyeY} rx="7" ry={eyeRy - 1} fill={HAIR} />
        <ellipse cx="117" cy={eyeY} rx="7" ry={eyeRy - 1} fill={HAIR} />
        <ellipse cx="83" cy={eyeY} rx="4.5" ry={eyeRy - 3} fill={HAIR2} fillOpacity="0.6" />
        <ellipse cx="117" cy={eyeY} rx="4.5" ry={eyeRy - 3} fill={HAIR2} fillOpacity="0.6" />
        {/* 눈동자 */}
        <circle cx="83" cy={eyeY + 1} r="4" fill={DARK} />
        <circle cx="117" cy={eyeY + 1} r="4" fill={DARK} />
        {/* 반짝임 */}
        <ellipse cx="85" cy={eyeY - 2} rx="3" ry="2" fill="white" />
        <circle cx="80" cy={eyeY + 2} r="1" fill="white" fillOpacity="0.6" />
        <ellipse cx="119" cy={eyeY - 2} rx="3" ry="2" fill="white" />
        <circle cx="114" cy={eyeY + 2} r="1" fill="white" fillOpacity="0.6" />
        {/* 속눈썹 (자연스러운) */}
        {[-4,-2,0,2,4].map(dx => (
          <line key={dx} x1={83+dx} y1={eyeY-eyeRy} x2={82+dx*0.9} y2={eyeY-eyeRy-4}
            stroke={DARK} strokeWidth="1.5" strokeLinecap="round" />
        ))}
        {[-4,-2,0,2,4].map(dx => (
          <line key={dx} x1={117+dx} y1={eyeY-eyeRy} x2={116+dx*0.9} y2={eyeY-eyeRy-4}
            stroke={DARK} strokeWidth="1.5" strokeLinecap="round" />
        ))}

        {/* 볼 홍조 */}
        <ellipse cx="70" cy="76" rx="12" ry="7" fill={BLUSH} fillOpacity="0.3" style={{ filter: 'blur(2px)' }} />
        <ellipse cx="130" cy="76" rx="12" ry="7" fill={BLUSH} fillOpacity="0.3" style={{ filter: 'blur(2px)' }} />

        {/* 코 (자연스러운) */}
        <path d="M 96 75 Q 93 80 96 83 Q 100 84 104 81 Q 107 78 106 75"
          fill="none" stroke="#d4956a" strokeWidth="1.5" strokeOpacity="0.5" />
        <path d="M 96 83 Q 100 85 104 83" fill="none" stroke="#d4956a" strokeWidth="1.5" strokeOpacity="0.3" />

        {/* 입 */}
        <path d={mouthD}
          fill={speaking ? '#f4a88a' : 'none'}
          stroke={DARK} strokeWidth="1.8" strokeLinecap="round"
        />
        {speaking && <rect x="93" y="89" width="14" height="7" rx="2" fill="white" />}
        {!speaking && (
          <path d={`M 88 ${emotion === 'happy' ? 99 : 95} Q 100 ${emotion === 'happy' ? 103 : 99} 112 ${emotion === 'happy' ? 99 : 95}`}
            fill={BLUSH} fillOpacity="0.4" stroke="none" />
        )}

        {/* 목 */}
        <rect x="89" y="100" width="22" height="20" rx="5" fill="url(#hanaSkin)" />
        {/* 목걸이 */}
        <path d="M 84 112 Q 100 118 116 112" fill="none" stroke={HAIR2} strokeWidth="1.5" />
        <circle cx="100" cy="118" r="4" fill={GREEN} />
      </g>

      {/* 리프 / 꽃 장식 효과 (항상 부유) */}
      {[0, 1, 2].map(i => (
        <text key={i} x={[32, 162, 50][i]} y={[100, 120, 150][i]}
          fontSize="11" fill={[GREEN, HAIR2, '#ffd700'][i]}
          style={{ animation: `hanaLeaf${i} ${3 + i}s ease-in-out infinite ${i * 0.8}s`, opacity: 0.6 }}
        >
          {['🌿', '✦', '◈'][i]}
        </text>
      ))}

      <style>{`
        @keyframes hanaBreathe {
          0%,100% { transform: scaleY(1); }
          50% { transform: scaleY(1.016) translateY(-1.5px); }
        }
        @keyframes hanaHeadBob {
          0%,100% { transform: translateY(0) rotate(0deg); }
          30% { transform: translateY(-4px) rotate(1deg); }
          70% { transform: translateY(1.5px) rotate(-0.7deg); }
        }
        @keyframes hanaHeadTilt {
          to { transform: rotate(-11deg) translateX(-5px); }
        }
        @keyframes hanaArmIdle {
          0%,100% { transform: rotate(0deg); }
          50% { transform: rotate(1.5deg); }
        }
        @keyframes hanaArmSpeakL {
          0%,100% { transform: rotate(-3deg) translateY(0); }
          50% { transform: rotate(12deg) translateY(-8px); }
        }
        @keyframes hanaArmSpeakR {
          0%,100% { transform: rotate(3deg) translateY(0); }
          50% { transform: rotate(-11deg) translateY(-7px); }
        }
        @keyframes hanaArmListen {
          to { transform: rotate(-9deg); }
        }
        @keyframes hanaSway {
          0%,100% { transform: skewX(0deg); }
          40% { transform: skewX(1.2deg); }
          70% { transform: skewX(-0.8deg); }
        }
        @keyframes hanaHairR {
          0%,100% { transform: rotate(0deg) translateY(0); }
          50% { transform: rotate(2deg) translateY(-2px); }
        }
        @keyframes hanaHairL {
          0%,100% { transform: rotate(0deg) translateY(0); }
          50% { transform: rotate(-2deg) translateY(-2px); }
        }
        @keyframes hanaLeaf0 {
          0%,100% { transform: translate(0,0) rotate(0deg); opacity: 0.3; }
          50% { transform: translate(-5px,-15px) rotate(15deg); opacity: 0.7; }
        }
        @keyframes hanaLeaf1 {
          0%,100% { transform: translate(0,0) rotate(0deg); opacity: 0.4; }
          50% { transform: translate(5px,-12px) rotate(-10deg); opacity: 0.8; }
        }
        @keyframes hanaLeaf2 {
          0%,100% { transform: translate(0,0); opacity: 0.2; }
          50% { transform: translate(-3px,-18px); opacity: 0.6; }
        }
      `}</style>
    </svg>
  )
}
