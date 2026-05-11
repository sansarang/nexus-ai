import type { CharacterProps } from './types'

/**
 * 미라 — 핑크/골드 화려한 K-pop 담대한 여성
 * 참고: 2번 이미지 중단 "미라" 스타일 (핑크 레이어드 헤어, 골드 의상, 강렬한 눈빛)
 */
export function Mira({ emotion, speaking, listening }: CharacterProps) {
  /* ── 팔레트 ── */
  const SKIN_L  = '#fceee0'
  const SKIN_M  = '#f4d0b2'
  const SKIN_S  = '#d9a682'
  const HAIR1   = '#ff6ba8'   // 핑크 헤어 메인
  const HAIR2   = '#ff3d85'   // 핑크 진한
  const HAIR3   = '#ffb3d1'   // 핑크 하이라이트
  const HAIR4   = '#cc2060'   // 다크 핑크 언더
  const GOLD    = '#f4c430'   // 골드 의상
  const GOLD_D  = '#c8920a'
  const GOLD_L  = '#ffe880'
  const OUTFIT2 = '#1a0e2e'   // 다크 퍼플
  const EYE_C   = '#8b3a8a'   // 보라 눈동자
  const DARK    = '#110818'
  const LIP     = '#d42060'
  const SKIN_FG = 'url(#miraSkinFace)'

  const eyeY  = emotion === 'happy' ? 93 : 90
  const eyeRy = emotion === 'happy' ? 5.5 : emotion === 'alert' ? 9 : 7.5

  const mouthD = speaking
    ? 'M 92 136 Q 108 150 124 136'
    : emotion === 'happy'    ? 'M 88 134 Q 108 150 128 134'
    : emotion === 'concerned'? 'M 95 140 Q 108 133 121 140'
    : emotion === 'humorous' ? 'M 90 134 Q 108 148 126 134 Q 120 153 96 153 Z'
    :                          'M 91 136 Q 108 146 125 136'

  return (
    <svg viewBox="0 0 220 500" width="220" height="500" style={{ overflow: 'visible' }}>
      <defs>
        <radialGradient id="miraSkinFace" cx="50%" cy="36%" r="60%">
          <stop offset="0%"   stopColor="#fffaf5" />
          <stop offset="40%"  stopColor={SKIN_L} />
          <stop offset="100%" stopColor={SKIN_M} />
        </radialGradient>
        <radialGradient id="miraSkinArm" cx="35%" cy="28%" r="72%">
          <stop offset="0%"   stopColor={SKIN_L} />
          <stop offset="100%" stopColor={SKIN_M} />
        </radialGradient>
        <linearGradient id="miraHair1" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%"   stopColor={HAIR3} />
          <stop offset="35%"  stopColor={HAIR1} />
          <stop offset="75%"  stopColor={HAIR2} />
          <stop offset="100%" stopColor={HAIR4} />
        </linearGradient>
        <linearGradient id="miraHairSide" x1="100%" y1="0%" x2="0%" y2="100%">
          <stop offset="0%"   stopColor={HAIR1} />
          <stop offset="100%" stopColor={HAIR4} />
        </linearGradient>
        <linearGradient id="miraGold" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%"   stopColor={GOLD_L} />
          <stop offset="45%"  stopColor={GOLD} />
          <stop offset="100%" stopColor={GOLD_D} />
        </linearGradient>
        <linearGradient id="miraGoldR" x1="100%" y1="0%" x2="0%" y2="100%">
          <stop offset="0%"   stopColor={GOLD_L} />
          <stop offset="100%" stopColor={GOLD_D} />
        </linearGradient>
        <radialGradient id="miraIris" cx="40%" cy="35%" r="58%">
          <stop offset="0%"   stopColor="#c070c0" />
          <stop offset="55%"  stopColor={EYE_C} />
          <stop offset="100%" stopColor="#3d0840" />
        </radialGradient>
        <filter id="miraGlow">
          <feGaussianBlur stdDeviation="3" result="b"/>
          <feMerge><feMergeNode in="b"/><feMergeNode in="SourceGraphic"/></feMerge>
        </filter>
        <filter id="miraShine">
          <feGaussianBlur stdDeviation="1.2" result="b"/>
          <feMerge><feMergeNode in="b"/><feMergeNode in="SourceGraphic"/></feMerge>
        </filter>
        {/* 골드 글로우 */}
        <radialGradient id="goldGlow" cx="50%" cy="0%" r="100%">
          <stop offset="0%"   stopColor="#ffe060" stopOpacity="0.35" />
          <stop offset="100%" stopColor="#ffe060" stopOpacity="0" />
        </radialGradient>
      </defs>

      {/* ─ 골드 글로우 오라 (감정 효과) ─ */}
      {(emotion === 'happy' || emotion === 'humorous') && (
        <ellipse cx="110" cy="240" rx="95" ry="200" fill="url(#goldGlow)" />
      )}

      {/* ═══ 스커트 / 하의 ═══ */}
      <g>
        {/* 골드 미니 스커트 */}
        <path d="M 68 295 Q 58 360 54 430 Q 80 440 108 438 L 110 370 L 112 438 Q 140 440 166 430 Q 162 360 152 295 Z"
          fill={OUTFIT2} />
        {/* 스커트 골드 트림 */}
        <path d="M 68 295 L 152 295 L 155 305 Q 130 310 110 308 Q 90 310 65 305 Z"
          fill="url(#miraGold)" />
        {/* 스커트 주름선 */}
        {[80, 95, 110, 125, 140].map(x => (
          <line key={x} x1={x} y1={305} x2={x-3} y2={430}
            stroke={GOLD_D} strokeWidth="0.8" strokeOpacity="0.3" />
        ))}
        {/* 다리 */}
        <path d="M 78 430 Q 80 460 82 490 L 94 490 Q 94 460 92 430 Z"
          fill={SKIN_M} />
        <path d="M 128 430 Q 130 460 132 490 L 120 490 Q 120 460 118 430 Z"
          fill={SKIN_M} />
        {/* 하이힐 */}
        <path d="M 78 488 Q 74 496 84 498 Q 94 498 98 492 L 96 490 Q 92 495 84 494 Q 78 492 80 488 Z"
          fill={GOLD_D} />
        <path d="M 118 492 L 118 488 Q 120 484 116 490 Q 112 494 118 492 Z"
          fill={GOLD_D} />
        <path d="M 122 488 Q 126 496 116 498 Q 106 498 102 492 L 104 490 Q 108 495 116 494 Q 122 492 120 488 Z"
          fill={GOLD_D} />
        <path d="M 82 492 L 82 498" stroke={GOLD} strokeWidth="2" strokeLinecap="round" />
        <path d="M 120 492 L 120 498" stroke={GOLD} strokeWidth="2" strokeLinecap="round" />
      </g>

      {/* ═══ 골드 의상 / 토르소 ═══ */}
      <g style={{ animation: 'miraBreathe 3.2s ease-in-out infinite', transformOrigin: '110px 230px' }}>
        {/* 뒤판 */}
        <path d="M 56 155 Q 44 185 46 295 L 174 295 Q 176 185 164 155 Z"
          fill={OUTFIT2} />
        {/* 골드 코르셋 앞판 */}
        <path d="M 72 158 Q 68 200 72 295 L 148 295 Q 152 200 148 158 Q 130 148 110 150 Q 90 148 72 158 Z"
          fill="url(#miraGold)" />
        {/* 골드 바디스 상세 */}
        <path d="M 90 155 Q 100 165 110 168 Q 120 165 130 155"
          fill="none" stroke={GOLD_L} strokeWidth="2" strokeOpacity="0.6" />
        {/* 골드 장식선 */}
        {[180, 210, 240, 265, 285].map(y => (
          <path key={y}
            d={`M 72 ${y} Q 110 ${y+4} 148 ${y}`}
            fill="none" stroke={GOLD_L} strokeWidth="1" strokeOpacity="0.35" />
        ))}
        {/* 코르셋 버클 */}
        {[175, 205, 235, 265].map(y => (
          <rect key={y} x="106" y={y-4} width="8" height="8" rx="1"
            fill={GOLD_L} stroke={GOLD_D} strokeWidth="0.8" />
        ))}
        {/* 네크라인 */}
        <path d="M 78 160 Q 90 148 110 146 Q 130 148 142 160"
          fill="none" stroke={GOLD} strokeWidth="3" />
        {/* 골드 네크라인 빛 */}
        <path d="M 82 160 Q 94 149 110 148 Q 126 149 138 160"
          fill="none" stroke={GOLD_L} strokeWidth="1.5" strokeOpacity="0.7" />
      </g>

      {/* ═══ 왼팔 ═══ */}
      <g style={{
        transformOrigin: '50px 158px',
        animation: speaking
          ? 'miraArmSpeakL 0.9s ease-in-out infinite'
          : listening
          ? 'miraArmListen 0.4s ease forwards'
          : 'miraArmIdleL 4.5s ease-in-out infinite',
      }}>
        {/* 골드 소매 */}
        <path d="M 56 155 Q 36 180 26 242 Q 24 278 36 286 Q 46 292 56 282 Q 44 248 46 205 Q 50 178 60 160 Z"
          fill="url(#miraGoldR)" />
        {/* 소매 테두리 */}
        <path d="M 30 278 Q 30 292 50 284" fill="none" stroke={GOLD_L} strokeWidth="1.5" strokeOpacity="0.5" />
        {/* 팔찌 */}
        <ellipse cx="36" cy="285" rx="13" ry="6" fill={GOLD_L} />
        <ellipse cx="36" cy="284" rx="11" ry="4.5" fill={GOLD} />
        {/* 피부 손 */}
        <ellipse cx="34" cy="298" rx="13" ry="11" fill="url(#miraSkinArm)" />
        <path d="M 23 295 Q 21 288 25 285" stroke={SKIN_M} strokeWidth="3" fill="none" strokeLinecap="round" />
        <path d="M 29 293 Q 27 285 31 283" stroke={SKIN_M} strokeWidth="2.5" fill="none" strokeLinecap="round" />
        <path d="M 35 292 Q 34 284 37 283" stroke={SKIN_M} strokeWidth="2.5" fill="none" strokeLinecap="round" />
        <path d="M 41 293 Q 41 285 44 285" stroke={SKIN_M} strokeWidth="2.5" fill="none" strokeLinecap="round" />
        <ellipse cx="32" cy="300" rx="9" ry="5" fill={SKIN_S} fillOpacity="0.25" />
      </g>

      {/* ═══ 오른팔 ═══ */}
      <g style={{
        transformOrigin: '170px 158px',
        animation: speaking
          ? 'miraArmSpeakR 0.9s ease-in-out infinite 0.3s'
          : 'miraArmIdleR 4.5s ease-in-out infinite 2s',
      }}>
        <path d="M 164 155 Q 184 180 194 242 Q 196 278 184 286 Q 174 292 164 282 Q 176 248 174 205 Q 170 178 160 160 Z"
          fill="url(#miraGold)" />
        <path d="M 190 278 Q 190 292 170 284" fill="none" stroke={GOLD_L} strokeWidth="1.5" strokeOpacity="0.5" />
        <ellipse cx="184" cy="285" rx="13" ry="6" fill={GOLD_L} />
        <ellipse cx="184" cy="284" rx="11" ry="4.5" fill={GOLD} />
        <ellipse cx="186" cy="298" rx="13" ry="11" fill="url(#miraSkinArm)" />
        <path d="M 197 295 Q 199 288 195 285" stroke={SKIN_M} strokeWidth="3" fill="none" strokeLinecap="round" />
        <path d="M 191 293 Q 193 285 189 283" stroke={SKIN_M} strokeWidth="2.5" fill="none" strokeLinecap="round" />
        <path d="M 185 292 Q 186 284 183 283" stroke={SKIN_M} strokeWidth="2.5" fill="none" strokeLinecap="round" />
        <path d="M 179 293 Q 179 285 176 285" stroke={SKIN_M} strokeWidth="2.5" fill="none" strokeLinecap="round" />
        <ellipse cx="188" cy="300" rx="9" ry="5" fill={SKIN_S} fillOpacity="0.25" />
      </g>

      {/* ═══ 목 ═══ */}
      <path d="M 96 155 Q 94 178 100 180 L 120 180 Q 126 178 124 155 Z" fill={SKIN_FG} />
      <path d="M 104 158 L 102 178" stroke={SKIN_S} strokeWidth="1.5" strokeOpacity="0.2" />
      <path d="M 116 158 L 118 178" stroke={SKIN_S} strokeWidth="1.5" strokeOpacity="0.2" />
      {/* 목걸이 */}
      <path d="M 78 168 Q 94 162 110 164 Q 126 162 142 168"
        fill="none" stroke={GOLD} strokeWidth="2.5" />
      <circle cx="110" cy="167" r="5" fill={GOLD_L} stroke={GOLD_D} strokeWidth="1" />
      <circle cx="110" cy="167" r="3" fill="white" fillOpacity="0.6" />

      {/* ═══ 머리 ═══ */}
      <g style={{
        transformOrigin: '110px 92px',
        animation: listening
          ? 'miraHeadTilt 0.5s ease forwards'
          : 'miraHeadBob 5s ease-in-out infinite',
      }}>
        {/* 헤어 뒤 볼륨 */}
        <ellipse cx="110" cy="80" rx="54" ry="60" fill="url(#miraHair1)" />
        {/* 옆 헤어 — 레이어드 컬 */}
        <path d="M 56 82 Q 38 100 40 135 Q 44 155 52 162 Q 58 150 56 130 Q 54 108 60 90 Q 56 86 56 82 Z"
          fill="url(#miraHairSide)" />
        <path d="M 164 82 Q 182 100 180 135 Q 176 155 168 162 Q 162 150 164 130 Q 166 108 160 90 Q 164 86 164 82 Z"
          fill="url(#miraHairSide)" />
        {/* 헤어 레이어 추가 (볼류미너스) */}
        <path d="M 42 105 Q 36 125 40 142 Q 44 155 50 158 Q 46 145 46 128 Q 46 114 48 104 Q 44 104 42 105 Z"
          fill={HAIR4} />
        <path d="M 178 105 Q 184 125 180 142 Q 176 155 170 158 Q 174 145 174 128 Q 174 114 172 104 Q 176 104 178 105 Z"
          fill={HAIR4} />

        {/* 얼굴 */}
        <path d="M 58 72 Q 56 102 60 124 Q 64 148 78 162 Q 94 172 110 172 Q 126 172 142 162 Q 156 148 160 124 Q 164 102 162 72 Q 156 40 130 28 Q 110 22 90 28 Q 62 40 58 72 Z"
          fill={SKIN_FG} />

        {/* 측면 음영 */}
        <path d="M 58 70 Q 56 102 60 124 Q 64 148 76 162 Q 68 148 64 125 Q 60 104 62 75 Z"
          fill={SKIN_S} fillOpacity="0.18" />
        <path d="M 162 70 Q 164 102 160 124 Q 156 148 144 162 Q 152 148 156 125 Q 160 104 158 75 Z"
          fill={SKIN_S} fillOpacity="0.18" />

        {/* 앞머리 (핑크, 레이어드) */}
        <path d="M 60 70 Q 68 30 110 22 Q 152 30 160 70 Q 148 36 128 30 Q 110 24 92 30 Q 72 36 60 70 Z"
          fill="url(#miraHair1)" />
        {/* 뱅 레이어 1 */}
        <path d="M 62 68 Q 70 34 92 28 Q 82 42 78 58 Q 74 70 68 78 Q 64 72 62 68 Z"
          fill={HAIR2} />
        {/* 뱅 레이어 2 — 오른쪽 */}
        <path d="M 158 68 Q 150 34 128 28 Q 138 42 142 58 Q 146 70 152 78 Q 156 72 158 68 Z"
          fill={HAIR1} />
        {/* 앞머리 가르마 */}
        <path d="M 98 24 Q 110 14 126 22 Q 114 24 106 36 Q 100 48 98 62 Q 95 48 98 24 Z"
          fill={HAIR3} fillOpacity="0.6" />
        {/* 헤어 하이라이트 */}
        <path d="M 84 28 Q 102 22 118 26"
          fill="none" stroke={HAIR3} strokeWidth="4" strokeOpacity="0.5" strokeLinecap="round" />
        <path d="M 76 36 Q 92 28 108 32"
          fill="none" stroke="#ffcce0" strokeWidth="2.5" strokeOpacity="0.35" strokeLinecap="round" />
        {/* 헤어 샤인 스팟 */}
        <ellipse cx="92" cy="32" rx="8" ry="4" fill="white" fillOpacity="0.15" transform="rotate(-20,92,32)" />

        {/* 귀 */}
        <path d="M 58 92 Q 52 97 52 106 Q 52 116 58 120 Q 62 122 64 118 Q 60 114 60 106 Q 60 98 64 94 Q 61 92 58 92 Z"
          fill={SKIN_M} />
        <path d="M 162 92 Q 168 97 168 106 Q 168 116 162 120 Q 158 122 156 118 Q 160 114 160 106 Q 160 98 156 94 Q 159 92 162 92 Z"
          fill={SKIN_M} />
        {/* 귀걸이 */}
        <ellipse cx="54" cy="124" rx="4" ry="6" fill={GOLD} />
        <circle cx="54" cy="131" r="3" fill={GOLD_L} />
        <ellipse cx="166" cy="124" rx="4" ry="6" fill={GOLD} />
        <circle cx="166" cy="131" r="3" fill={GOLD_L} />

        {/* ─ 눈썹 ─ */}
        <path
          d={emotion === 'concerned'
            ? 'M 72 78 Q 87 74 102 80'
            : emotion === 'happy'
            ? 'M 72 74 Q 87 68 102 74'
            : emotion === 'humorous'
            ? 'M 72 76 Q 87 70 102 76'
            : 'M 72 76 Q 87 70 102 76'}
          fill="none" stroke={HAIR4} strokeWidth="4" strokeLinecap="round" />
        <path
          d={emotion === 'concerned'
            ? 'M 118 80 Q 133 74 148 78'
            : emotion === 'happy'
            ? 'M 118 74 Q 133 68 148 74'
            : 'M 118 76 Q 133 70 148 76'}
          fill="none" stroke={HAIR4} strokeWidth="4" strokeLinecap="round" />
        {/* 눈썹 아치 하이라이트 */}
        <path
          d={emotion === 'happy' ? 'M 75 74 Q 87 69 100 74' : 'M 75 76 Q 87 71 100 76'}
          fill="none" stroke={HAIR3} strokeWidth="1.5" strokeOpacity="0.4" strokeLinecap="round" />

        {/* ─ 눈 (글래머러스, 컬러 렌즈 느낌) ─ */}
        <ellipse cx="87" cy={eyeY - 4} rx="16" ry="7" fill={SKIN_S} fillOpacity="0.1" />
        <ellipse cx="133" cy={eyeY - 4} rx="16" ry="7" fill={SKIN_S} fillOpacity="0.1" />
        {/* 흰자 */}
        <path d={`M 74 ${eyeY} Q 87 ${eyeY - eyeRy - 3} 100 ${eyeY} Q 87 ${eyeY + eyeRy} 74 ${eyeY} Z`}
          fill="white" />
        <path d={`M 120 ${eyeY} Q 133 ${eyeY - eyeRy - 3} 146 ${eyeY} Q 133 ${eyeY + eyeRy} 120 ${eyeY} Z`}
          fill="white" />
        {/* 컬러 홍채 */}
        <ellipse cx="87" cy={eyeY} rx="9.5" ry={eyeRy} fill="url(#miraIris)" />
        <ellipse cx="133" cy={eyeY} rx="9.5" ry={eyeRy} fill="url(#miraIris)" />
        {/* 눈동자 */}
        <circle cx="87" cy={eyeY + 1} r="5.5" fill={DARK} />
        <circle cx="133" cy={eyeY + 1} r="5.5" fill={DARK} />
        {/* 반짝임 (2개) */}
        <ellipse cx="89" cy={eyeY - 3} rx="3.5" ry="2.5" fill="white" fillOpacity="0.92" />
        <circle cx="84" cy={eyeY + 3} r="1.5" fill="white" fillOpacity="0.55" />
        <ellipse cx="135" cy={eyeY - 3} rx="3.5" ry="2.5" fill="white" fillOpacity="0.92" />
        <circle cx="130" cy={eyeY + 3} r="1.5" fill="white" fillOpacity="0.55" />
        {/* 아이라인 (두껍고 날카로운 캣아이) */}
        <path d={`M 72 ${eyeY - 1} Q 87 ${eyeY - eyeRy - 4} 102 ${eyeY - 2} L 108 ${eyeY - 8}`}
          fill="none" stroke={DARK} strokeWidth="3" strokeLinecap="round" />
        <path d={`M 118 ${eyeY - 2} Q 133 ${eyeY - eyeRy - 4} 148 ${eyeY - 1} L 142 ${eyeY - 8}`}
          fill="none" stroke={DARK} strokeWidth="3" strokeLinecap="round" />
        {/* 속눈썹 */}
        {[-5,-3,-1,1,3,5,7].map(dx => (
          <line key={dx} x1={87+dx} y1={eyeY-eyeRy-2} x2={86+dx*0.85} y2={eyeY-eyeRy-7}
            stroke={DARK} strokeWidth="2" strokeLinecap="round" />
        ))}
        {[-5,-3,-1,1,3,5,7].map(dx => (
          <line key={dx} x1={133+dx} y1={eyeY-eyeRy-2} x2={132+dx*0.85} y2={eyeY-eyeRy-7}
            stroke={DARK} strokeWidth="2" strokeLinecap="round" />
        ))}
        {/* 아이섀도우 */}
        <ellipse cx="87" cy={eyeY - eyeRy + 1} rx="12" ry="3" fill="#cc60a8" fillOpacity="0.25" />
        <ellipse cx="133" cy={eyeY - eyeRy + 1} rx="12" ry="3" fill="#cc60a8" fillOpacity="0.25" />

        {/* ─ 코 ─ */}
        <path d="M 105 112 Q 103 120 105 128 Q 110 132 115 128 Q 117 120 115 112"
          fill="none" stroke={SKIN_S} strokeWidth="1.5" strokeOpacity="0.45" strokeLinecap="round" />
        <path d="M 103 126 Q 110 130 117 126"
          fill="none" stroke={SKIN_S} strokeWidth="1.5" strokeOpacity="0.4" />
        <ellipse cx="102" cy="126" rx="5" ry="4" fill={SKIN_S} fillOpacity="0.1" />
        <ellipse cx="118" cy="126" rx="5" ry="4" fill={SKIN_S} fillOpacity="0.1" />

        {/* ─ 입 (글래머러스 풀립) ─ */}
        {/* 윗입술 */}
        <path d="M 90 133 Q 100 127 110 132 Q 120 127 130 133 Q 122 138 110 140 Q 98 138 90 133 Z"
          fill={LIP} />
        {/* 아랫입술 */}
        {!speaking && (
          <path d="M 90 133 Q 98 146 110 148 Q 122 146 130 133 Q 122 142 110 143 Q 98 142 90 133 Z"
            fill="#e83070" fillOpacity="0.9" />
        )}
        {speaking && (
          <>
            <path d={mouthD} fill="rgba(60,10,30,0.7)" />
            <rect x="104" y="134" width="12" height="8" rx="2" fill="white" fillOpacity="0.85" />
          </>
        )}
        {/* 입술 광택 */}
        <ellipse cx="103" cy="131" rx="8" ry="3" fill="white" fillOpacity="0.2" transform="rotate(-5,103,131)" />
        {/* 입술 중앙 */}
        {!speaking && (
          <path d="M 90 133 Q 110 136 130 133" fill="none" stroke={HAIR4} strokeWidth="0.8" strokeOpacity="0.4" />
        )}

        {/* 볼 하이라이트 */}
        <ellipse cx="70" cy="116" rx="16" ry="10" fill="white" fillOpacity="0.07" transform="rotate(-10,70,116)" />
        <ellipse cx="150" cy="116" rx="16" ry="10" fill="white" fillOpacity="0.07" transform="rotate(10,150,116)" />
        {/* 볼 홍조 */}
        <ellipse cx="72" cy="120" rx="14" ry="8" fill="#ff80a0" fillOpacity="0.12" />
        <ellipse cx="148" cy="120" rx="14" ry="8" fill="#ff80a0" fillOpacity="0.12" />

        {/* 턱 음영 */}
        <path d="M 82 158 Q 110 168 138 158 Q 110 175 82 158 Z"
          fill={SKIN_S} fillOpacity="0.15" />

        {/* 골드 헤드밴드 장식 */}
        <path d="M 64 64 Q 84 52 110 50 Q 136 52 156 64"
          fill="none" stroke={GOLD} strokeWidth="3" />
        <circle cx="110" cy="48" r="7" fill={GOLD_L} stroke={GOLD_D} strokeWidth="1.5" />
        <path d="M 106 44 L 110 40 L 114 44" fill={GOLD} />
        {/* 스타 헤어핀 */}
        <path d="M 68 60 L 72 56 L 76 60 L 74 65 L 70 65 Z" fill={GOLD_L} />
      </g>

      <style>{`
        @keyframes miraBreathe {
          0%,100% { transform: scaleY(1) translateY(0); }
          50% { transform: scaleY(1.014) translateY(-2px); }
        }
        @keyframes miraHeadBob {
          0%,100% { transform: translateY(0) rotate(0deg); }
          25% { transform: translateY(-6px) rotate(1.2deg); }
          75% { transform: translateY(2px) rotate(-0.8deg); }
        }
        @keyframes miraHeadTilt {
          to { transform: rotate(-12deg) translateX(-8px); }
        }
        @keyframes miraArmIdleL {
          0%,100% { transform: rotate(0deg); }
          50% { transform: rotate(3deg); }
        }
        @keyframes miraArmIdleR {
          0%,100% { transform: rotate(0deg); }
          50% { transform: rotate(-3deg); }
        }
        @keyframes miraArmSpeakL {
          0%,100% { transform: rotate(-5deg) translateY(0); }
          50% { transform: rotate(15deg) translateY(-14px); }
        }
        @keyframes miraArmSpeakR {
          0%,100% { transform: rotate(5deg) translateY(0); }
          50% { transform: rotate(-12deg) translateY(-10px); }
        }
        @keyframes miraArmListen {
          to { transform: rotate(-10deg); }
        }
      `}</style>
    </svg>
  )
}
