import type { CharacterProps } from './types'

/**
 * 소라 — K-pop 아이돌 + 귀여운 애니 스타일
 * 라이트블루 + 화이트 + 핑크 컬러 팔레트
 * 사이드업 트윈테일 + 아이돌 의상
 */
export function Sora({ emotion, speaking, listening }: CharacterProps) {
  const SKIN    = '#fde8d0'
  const HAIR    = '#1a1a2e'   // 짙은 남흑색 베이스
  const HILIGHT = '#5b9bd5'   // 블루 하이라이트
  const TOP     = '#e8f4fd'   // 아이돌 재킷 화이트
  const BLUE    = '#4fb3e8'   // 메인 포인트 블루
  const PINK    = '#ff7eb3'   // 핑크 악센트
  const SKIRT   = '#4fb3e8'
  const DARK    = '#1a0a12'
  const ACCENT  = '#ffd6e8'

  const eyeY   = emotion === 'happy' ? 72 : 70
  const eyeRy  = emotion === 'happy' ? 5 : emotion === 'alert' ? 9 : 8

  const mouthD =
    speaking           ? 'M 87 93 Q 100 106 113 93'
    : emotion === 'happy'    ? 'M 83 91 Q 100 105 117 91'
    : emotion === 'concerned'? 'M 89 96 Q 100 90 111 96'
    : emotion === 'humorous' ? 'M 85 91 Q 100 103 115 91 Q 108 108 92 108 Z'
    : emotion === 'alert'    ? 'M 87 93 Q 100 97 113 93'
    :                          'M 87 91 Q 100 100 113 91'

  return (
    <svg viewBox="0 0 200 420" width="200" height="420" style={{ overflow: 'visible' }}>
      <defs>
        <radialGradient id="soraSkin" cx="50%" cy="35%" r="65%">
          <stop offset="0%" stopColor="#fff5ee" />
          <stop offset="100%" stopColor={SKIN} />
        </radialGradient>
        <radialGradient id="soraChest" cx="50%" cy="0%" r="100%">
          <stop offset="0%" stopColor={TOP} />
          <stop offset="100%" stopColor="#cce8f8" />
        </radialGradient>
        <linearGradient id="soraHair" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor={HAIR} />
          <stop offset="60%" stopColor="#2d2d4a" />
          <stop offset="100%" stopColor={HAIR} />
        </linearGradient>
        <linearGradient id="soraSkirt" x1="0%" y1="0%" x2="0%" y2="100%">
          <stop offset="0%" stopColor={BLUE} />
          <stop offset="100%" stopColor="#2e9fd8" />
        </linearGradient>
        <filter id="soraGlow">
          <feGaussianBlur stdDeviation="2.5" result="b"/>
          <feMerge><feMergeNode in="b"/><feMergeNode in="SourceGraphic"/></feMerge>
        </filter>
        <filter id="soraHairShine">
          <feGaussianBlur stdDeviation="1" result="b"/>
          <feMerge><feMergeNode in="b"/><feMergeNode in="SourceGraphic"/></feMerge>
        </filter>
      </defs>

      {/* ── 다리 / 스타킹 ── */}
      <g>
        {/* 레이스 트림 스타킹 */}
        <rect x="63" y="280" width="28" height="115" rx="12" fill="#e8f0f8" />
        <rect x="109" y="280" width="28" height="115" rx="12" fill="#e8f0f8" />
        {/* 스타킹 패턴 라인 */}
        {[300, 320, 340, 360].map(y => (
          <g key={y}>
            <line x1="65" y1={y} x2="89" y2={y} stroke="#b8d0e8" strokeWidth="0.7" />
            <line x1="111" y1={y} x2="135" y2={y} stroke="#b8d0e8" strokeWidth="0.7" />
          </g>
        ))}
        {/* 스타킹 상단 레이스 */}
        <path d="M 60 282 Q 77 276 92 282" fill="none" stroke={BLUE} strokeWidth="2" />
        <path d="M 108 282 Q 123 276 138 282" fill="none" stroke={BLUE} strokeWidth="2" />
        {/* 부츠 */}
        <rect x="59" y="380" width="38" height="28" rx="10" fill={HAIR} />
        <rect x="105" y="380" width="38" height="28" rx="10" fill={HAIR} />
        {/* 부츠 버클 */}
        <rect x="62" y="388" width="32" height="8" rx="4" fill={HAIR} stroke={BLUE} strokeWidth="1.5" />
        <rect x="108" y="388" width="32" height="8" rx="4" fill={HAIR} stroke={BLUE} strokeWidth="1.5" />
        {/* 부츠 하이라이트 */}
        <path d="M 64 382 Q 78 380 92 382" fill="none" stroke="#4a4a6a" strokeWidth="1.5" />
        <path d="M 110 382 Q 122 380 136 382" fill="none" stroke="#4a4a6a" strokeWidth="1.5" />
      </g>

      {/* ── 스커트 ── */}
      <g style={{ animation: 'soraSkirt 3.5s ease-in-out infinite', transformOrigin: '100px 255px' }}>
        <path d="M 44 220 Q 36 256 42 280 L 62 277 Q 60 260 63 245 L 137 245 Q 140 260 138 277 L 158 280 Q 164 256 156 220 Z"
          fill="url(#soraSkirt)" />
        {/* 스커트 플리츠 라인 */}
        {[55, 68, 81, 94, 107, 120, 133, 146].map(x => (
          <line key={x} x1={x} y1={220} x2={x < 100 ? x - 3 : x + 3} y2={280}
            stroke="rgba(255,255,255,0.25)" strokeWidth="1" />
        ))}
        {/* 스커트 하단 레이스 트림 */}
        <path d="M 42 278 Q 100 290 158 278" fill="none" stroke="white" strokeWidth="2" strokeOpacity="0.6" />
        {[50, 65, 80, 95, 110, 125, 140, 155].map((x, i) => (
          <path key={i}
            d={`M ${x} 278 Q ${x + 7} 284 ${x + 14} 278`}
            fill="none" stroke="white" strokeWidth="1.5" strokeOpacity="0.5"
          />
        ))}
      </g>

      {/* ── 상체 아이돌 재킷 ── */}
      <g style={{ animation: 'soraBreathe 3s ease-in-out infinite', transformOrigin: '100px 175px' }}>
        {/* 재킷 */}
        <path d="M 52 118 Q 42 138 44 220 L 156 220 Q 158 138 148 118 Z" fill="url(#soraChest)" />
        {/* 재킷 라펠 */}
        <path d="M 80 118 L 70 145 L 100 155 L 130 145 L 120 118" fill="white" fillOpacity="0.6" />
        {/* 넥타이 / 리본 */}
        <path d="M 92 130 L 100 155 L 108 130 L 100 135 Z" fill={PINK} />
        <circle cx="100" cy="128" r="5" fill={PINK} />
        {/* 재킷 단추 */}
        {[160, 180, 200].map(y => (
          <circle key={y} cx="100" cy={y} r="3" fill={BLUE} stroke="white" strokeWidth="0.5" />
        ))}
        {/* 재킷 포켓 */}
        <rect x="58" y="175" width="24" height="18" rx="4" fill="rgba(255,255,255,0.3)" stroke={BLUE} strokeWidth="1" />
        <rect x="118" y="175" width="24" height="18" rx="4" fill="rgba(255,255,255,0.3)" stroke={BLUE} strokeWidth="1" />
        {/* 포켓 별 장식 */}
        <text x="66" y="187" fontSize="10" fill={BLUE} textAnchor="middle">★</text>
        <text x="130" y="187" fontSize="10" fill={BLUE} textAnchor="middle">★</text>
        {/* 어깨 에폴렛 */}
        <path d="M 44 118 Q 40 110 52 108 Q 58 118 52 122 Z" fill={BLUE} />
        <path d="M 156 118 Q 160 110 148 108 Q 142 118 148 122 Z" fill={BLUE} />
      </g>

      {/* ── 왼팔 (말할 때 제스처) ── */}
      <g style={{
        transformOrigin: '48px 120px',
        animation: speaking
          ? 'soraArmSpeakL 0.9s ease-in-out infinite'
          : listening
          ? 'soraArmListen 0.4s ease forwards'
          : 'soraArmIdle 3.5s ease-in-out infinite',
      }}>
        <path d="M 50 118 Q 30 138 24 192 Q 36 202 52 196 Q 54 152 64 122 Z" fill="url(#soraChest)" />
        {/* 소매 블루 커프 */}
        <path d="M 26 188 Q 36 196 52 192 Q 50 182 36 178 Z" fill={BLUE} />
        {/* 화이트 글러브 손 */}
        <ellipse cx="34" cy="200" rx="13" ry="11" fill="white" />
        <path d="M 26 197 Q 24 191 28 189" stroke="#ddd" strokeWidth="2.5" strokeLinecap="round" />
        <path d="M 31 195 Q 29 188 33 186" stroke="#ddd" strokeWidth="2.5" strokeLinecap="round" />
        <path d="M 37 194 Q 36 187 39 186" stroke="#ddd" strokeWidth="2.5" strokeLinecap="round" />
        {/* 팔찌 리본 */}
        <path d="M 26 199 Q 34 202 42 199" fill="none" stroke={PINK} strokeWidth="2.5" />
        <circle cx="34" cy="201" r="3" fill={PINK} />
      </g>

      {/* ── 오른팔 ── */}
      <g style={{
        transformOrigin: '152px 120px',
        animation: speaking
          ? 'soraArmSpeakR 0.9s ease-in-out infinite 0.3s'
          : 'soraArmIdle 3.5s ease-in-out infinite 1.5s',
      }}>
        <path d="M 150 118 Q 170 138 176 192 Q 164 202 148 196 Q 146 152 136 122 Z" fill="url(#soraChest)" />
        <path d="M 174 188 Q 164 196 148 192 Q 150 182 164 178 Z" fill={BLUE} />
        <ellipse cx="166" cy="200" rx="13" ry="11" fill="white" />
        <path d="M 174 197 Q 176 191 172 189" stroke="#ddd" strokeWidth="2.5" strokeLinecap="round" />
        <path d="M 169 195 Q 171 188 167 186" stroke="#ddd" strokeWidth="2.5" strokeLinecap="round" />
        <path d="M 163 194 Q 164 187 161 186" stroke="#ddd" strokeWidth="2.5" strokeLinecap="round" />
        <path d="M 158 199 Q 166 202 174 199" fill="none" stroke={PINK} strokeWidth="2.5" />
        <circle cx="166" cy="201" r="3" fill={PINK} />
      </g>

      {/* ── 머리 ── */}
      <g style={{
        transformOrigin: '100px 65px',
        animation: listening
          ? 'soraHeadTilt 0.4s ease forwards'
          : 'soraHeadBob 4s ease-in-out infinite',
      }}>
        {/* ─ 긴 사이드 트윈 테일 ─ */}
        {/* 왼쪽 */}
        <g style={{ animation: 'soraTailL 3s ease-in-out infinite', transformOrigin: '52px 92px' }}>
          <path d="M 52 88 Q 18 110 12 195 Q 24 205 36 195 Q 32 130 52 100 Z" fill="url(#soraHair)" />
          {/* 헤어라이트 */}
          <path d="M 26 130 Q 22 155 24 178" fill="none" stroke={HILIGHT} strokeWidth="2" strokeOpacity="0.4" />
          {/* 테일 리본 */}
          <path d="M 44 90 Q 52 84 60 90 Q 56 98 52 100 Q 48 98 44 90 Z" fill={PINK} />
          <circle cx="52" cy="90" r="5" fill={PINK} filter="url(#soraGlow)" />
        </g>
        {/* 오른쪽 */}
        <g style={{ animation: 'soraTailR 3s ease-in-out infinite 0.5s', transformOrigin: '148px 92px' }}>
          <path d="M 148 88 Q 182 110 188 195 Q 176 205 164 195 Q 168 130 148 100 Z" fill="url(#soraHair)" />
          <path d="M 174 130 Q 178 155 176 178" fill="none" stroke={HILIGHT} strokeWidth="2" strokeOpacity="0.4" />
          <path d="M 140 90 Q 148 84 156 90 Q 152 98 148 100 Q 144 98 140 90 Z" fill={PINK} />
          <circle cx="148" cy="90" r="5" fill={PINK} filter="url(#soraGlow)" />
        </g>

        {/* 헤어 뒤 */}
        <ellipse cx="100" cy="60" rx="47" ry="52" fill="url(#soraHair)" />
        {/* 얼굴 */}
        <ellipse cx="100" cy="56" rx="42" ry="44" fill="url(#soraSkin)" />

        {/* 앞머리 */}
        <path d="M 58 34 Q 74 8 100 12 Q 126 8 142 34 Q 126 18 100 20 Q 74 18 58 34 Z" fill="url(#soraHair)" />
        <path d="M 58 36 Q 66 18 76 16 Q 70 28 68 42 Z" fill="url(#soraHair)" />
        <path d="M 142 36 Q 134 18 124 16 Q 130 28 132 42 Z" fill="url(#soraHair)" />
        {/* 앞머리 중앙 가리마 */}
        <path d="M 86 14 Q 84 4 88 0" fill="none" stroke="url(#soraHair)" strokeWidth="5" strokeLinecap="round" />
        <path d="M 100 12 Q 100 2 102 -2" fill="none" stroke="url(#soraHair)" strokeWidth="5" strokeLinecap="round" />
        {/* 헤어 샤인 */}
        <path d="M 74 20 Q 78 15 86 18" fill="none" stroke={HILIGHT} strokeWidth="2" strokeOpacity="0.5" filter="url(#soraHairShine)" />
        <path d="M 114 20 Q 118 15 124 18" fill="none" stroke={HILIGHT} strokeWidth="2" strokeOpacity="0.5" filter="url(#soraHairShine)" />

        {/* 귀 */}
        <ellipse cx="57" cy="66" rx="8" ry="10" fill={SKIN} />
        <ellipse cx="143" cy="66" rx="8" ry="10" fill={SKIN} />
        <ellipse cx="57" cy="66" rx="5" ry="7" fill={ACCENT} fillOpacity="0.4" />
        <ellipse cx="143" cy="66" rx="5" ry="7" fill={ACCENT} fillOpacity="0.4" />
        {/* 귀걸이 */}
        <circle cx="57" cy="79" r="4" fill={BLUE} style={{ filter: 'drop-shadow(0 0 3px rgba(79,179,232,0.8))' }} />
        <path d="M 57 83 L 57 89" stroke={BLUE} strokeWidth="1.5" />
        <circle cx="57" cy="90" r="3" fill={PINK} />
        <circle cx="143" cy="79" r="4" fill={BLUE} style={{ filter: 'drop-shadow(0 0 3px rgba(79,179,232,0.8))' }} />
        <path d="M 143 83 L 143 89" stroke={BLUE} strokeWidth="1.5" />
        <circle cx="143" cy="90" r="3" fill={PINK} />

        {/* 눈썹 (부드러운 아치형) */}
        <path d={emotion === 'concerned' ? 'M 73 50 Q 83 47 92 51' : emotion === 'happy' ? 'M 72 48 Q 83 43 93 48' : 'M 73 50 Q 83 45 92 50'}
          fill="none" stroke={HAIR} strokeWidth="2.5" strokeLinecap="round" />
        <path d={emotion === 'concerned' ? 'M 108 51 Q 117 47 127 50' : emotion === 'happy' ? 'M 107 48 Q 117 43 128 48' : 'M 108 50 Q 117 45 127 50'}
          fill="none" stroke={HAIR} strokeWidth="2.5" strokeLinecap="round" />

        {/* ─── 눈 (K-pop 아이돌 눈 — 크고 반짝임) ─── */}
        {/* 눈 외곽 검은 윤곽 */}
        <ellipse cx="83" cy={eyeY} rx="12.5" ry={eyeRy + 1} fill={DARK} />
        <ellipse cx="117" cy={eyeY} rx="12.5" ry={eyeRy + 1} fill={DARK} />
        {/* 흰자 */}
        <ellipse cx="83" cy={eyeY} rx="11" ry={eyeRy} fill="white" />
        <ellipse cx="117" cy={eyeY} rx="11" ry={eyeRy} fill="white" />
        {/* 홍채 — 블루 그라데이션 */}
        <radialGradient id="soraIris1" cx="50%" cy="40%" r="60%">
          <stop offset="0%" stopColor="#6ec4f0" />
          <stop offset="70%" stopColor={BLUE} />
          <stop offset="100%" stopColor="#1a6aa8" />
        </radialGradient>
        <radialGradient id="soraIris2" cx="50%" cy="40%" r="60%">
          <stop offset="0%" stopColor="#6ec4f0" />
          <stop offset="70%" stopColor={BLUE} />
          <stop offset="100%" stopColor="#1a6aa8" />
        </radialGradient>
        <ellipse cx="83" cy={eyeY} rx="8" ry={eyeRy - 1} fill="url(#soraIris1)" />
        <ellipse cx="117" cy={eyeY} rx="8" ry={eyeRy - 1} fill="url(#soraIris2)" />
        {/* 눈동자 */}
        <circle cx="83" cy={eyeY + 1} r="5" fill={DARK} />
        <circle cx="117" cy={eyeY + 1} r="5" fill={DARK} />
        {/* 반짝임 3개 (아이돌 눈) */}
        <ellipse cx="86" cy={eyeY - 3} rx="3.5" ry="2.5" fill="white" />
        <circle cx="80" cy={eyeY + 2} r="1.5" fill="white" fillOpacity="0.7" />
        <circle cx="86" cy={eyeY + 3} r="1" fill="white" fillOpacity="0.5" />
        <ellipse cx="120" cy={eyeY - 3} rx="3.5" ry="2.5" fill="white" />
        <circle cx="114" cy={eyeY + 2} r="1.5" fill="white" fillOpacity="0.7" />
        <circle cx="120" cy={eyeY + 3} r="1" fill="white" fillOpacity="0.5" />
        {/* 속눈썹 위 */}
        {[-5,-3,-1,1,3,5].map(dx => (
          <line key={dx} x1={83+dx} y1={eyeY-eyeRy} x2={82+dx*0.85} y2={eyeY-eyeRy-5}
            stroke={DARK} strokeWidth="1.8" strokeLinecap="round" />
        ))}
        {[-5,-3,-1,1,3,5].map(dx => (
          <line key={dx} x1={117+dx} y1={eyeY-eyeRy} x2={116+dx*0.85} y2={eyeY-eyeRy-5}
            stroke={DARK} strokeWidth="1.8" strokeLinecap="round" />
        ))}
        {/* 아래 속눈썹 */}
        {[-4,-1,2,5].map(dx => (
          <line key={dx} x1={83+dx} y1={eyeY+eyeRy} x2={83+dx} y2={eyeY+eyeRy+3}
            stroke={DARK} strokeWidth="1" strokeLinecap="round" strokeOpacity="0.5" />
        ))}
        {[-4,-1,2,5].map(dx => (
          <line key={dx} x1={117+dx} y1={eyeY+eyeRy} x2={117+dx} y2={eyeY+eyeRy+3}
            stroke={DARK} strokeWidth="1" strokeLinecap="round" strokeOpacity="0.5" />
        ))}
        {/* 아이라인 */}
        <path d={`M 71 ${eyeY-eyeRy+1} Q 83 ${eyeY-eyeRy-2} 95 ${eyeY-eyeRy+1}`}
          fill="none" stroke={DARK} strokeWidth="2" />
        <path d={`M 105 ${eyeY-eyeRy+1} Q 117 ${eyeY-eyeRy-2} 129 ${eyeY-eyeRy+1}`}
          fill="none" stroke={DARK} strokeWidth="2" />
        {/* 쌍꺼풀 라인 */}
        <path d={`M 73 ${eyeY-eyeRy+3} Q 83 ${eyeY-eyeRy+1} 93 ${eyeY-eyeRy+3}`}
          fill="none" stroke={SKIN} strokeWidth="1" strokeOpacity="0.5" />
        <path d={`M 107 ${eyeY-eyeRy+3} Q 117 ${eyeY-eyeRy+1} 127 ${eyeY-eyeRy+3}`}
          fill="none" stroke={SKIN} strokeWidth="1" strokeOpacity="0.5" />

        {/* 볼 홍조 */}
        <ellipse cx="70" cy="80" rx="13" ry="8" fill={PINK} fillOpacity="0.25"
          style={{ filter: 'blur(3px)' }} />
        <ellipse cx="130" cy="80" rx="13" ry="8" fill={PINK} fillOpacity="0.25"
          style={{ filter: 'blur(3px)' }} />
        {/* 기쁨 시 하트 홍조 */}
        {(emotion === 'happy' || emotion === 'humorous') && (
          <>
            <text x="68" y="84" fontSize="12" textAnchor="middle" fill={PINK} fillOpacity="0.6">♥</text>
            <text x="132" y="84" fontSize="12" textAnchor="middle" fill={PINK} fillOpacity="0.6">♥</text>
          </>
        )}

        {/* 코 (미니멀하게) */}
        <path d="M 97 79 Q 95 84 98 86 Q 102 87 105 84 Q 107 82 106 79"
          fill="none" stroke={SKIN} strokeWidth="1.5" strokeOpacity="0.45" />

        {/* 입 */}
        <path d={mouthD}
          fill={speaking ? '#ffc4d6' : 'none'}
          stroke={DARK} strokeWidth="1.8" strokeLinecap="round"
        />
        {/* 치아 */}
        {speaking && (
          <rect x="93" y="93" width="14" height="7" rx="2" fill="white" />
        )}
        {/* 아랫 입술 */}
        {!speaking && (
          <path d={`M 89 ${emotion === 'happy' ? 101 : 97} Q 100 ${emotion === 'happy' ? 105 : 101} 111 ${emotion === 'happy' ? 101 : 97}`}
            fill={PINK} fillOpacity="0.5" stroke="none" />
        )}

        {/* 목 */}
        <rect x="89" y="98" width="22" height="22" rx="5" fill="url(#soraSkin)" />
      </g>

      {/* 하트/별 이모티콘 (감정 표현) */}
      {emotion === 'happy' && (
        <g style={{ animation: 'soraHeart 2s ease-in-out infinite' }}>
          <text x="145" y="45" fontSize="18" fill={PINK} style={{ filter: 'drop-shadow(0 0 4px rgba(255,126,179,0.6))' }}>♥</text>
        </g>
      )}
      {emotion === 'humorous' && (
        <g style={{ animation: 'soraHeart 1.5s ease-in-out infinite' }}>
          <text x="148" y="48" fontSize="16" fill="#ffd700">★</text>
          <text x="52" y="52" fontSize="12" fill={PINK}>♪</text>
        </g>
      )}
      {emotion === 'alert' && (
        <g style={{ animation: 'soraAlert 0.5s ease-in-out infinite' }}>
          <text x="145" y="48" fontSize="18" fill="#ff6b6b">!</text>
        </g>
      )}

      <style>{`
        @keyframes soraBreathe {
          0%,100% { transform: scaleY(1); }
          50% { transform: scaleY(1.018) translateY(-2px); }
        }
        @keyframes soraHeadBob {
          0%,100% { transform: translateY(0) rotate(0deg); }
          28% { transform: translateY(-5px) rotate(1.2deg); }
          70% { transform: translateY(1.5px) rotate(-0.8deg); }
        }
        @keyframes soraHeadTilt {
          to { transform: rotate(-13deg) translateX(-7px); }
        }
        @keyframes soraArmIdle {
          0%,100% { transform: rotate(0deg); }
          50% { transform: rotate(2deg); }
        }
        @keyframes soraArmSpeakL {
          0%,100% { transform: rotate(-4deg) translateY(0); }
          50% { transform: rotate(16deg) translateY(-10px); }
        }
        @keyframes soraArmSpeakR {
          0%,100% { transform: rotate(4deg) translateY(0); }
          50% { transform: rotate(-14deg) translateY(-8px); }
        }
        @keyframes soraArmListen {
          to { transform: rotate(-10deg); }
        }
        @keyframes soraSkirt {
          0%,100% { transform: skewX(0deg); }
          33% { transform: skewX(1deg); }
          66% { transform: skewX(-1deg); }
        }
        @keyframes soraTailL {
          0%,100% { transform: rotate(0deg); }
          50% { transform: rotate(4deg); }
        }
        @keyframes soraTailR {
          0%,100% { transform: rotate(0deg); }
          50% { transform: rotate(-4deg); }
        }
        @keyframes soraHeart {
          0%,100% { transform: scale(1) translateY(0); opacity: 0.8; }
          50% { transform: scale(1.3) translateY(-5px); opacity: 1; }
        }
        @keyframes soraAlert {
          0%,100% { transform: translateY(0); }
          50% { transform: translateY(-3px); }
        }
      `}</style>
    </svg>
  )
}
