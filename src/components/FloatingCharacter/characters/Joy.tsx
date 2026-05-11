import type { CharacterProps } from './types'

/**
 * 조이 — 쿨하고 에너지 넘치는 여성
 * 참고: 2번 이미지 "조이" (블루 톤 헤어, 강한 눈빛, 달빛 배경)
 */
export function Joy({ emotion, speaking, listening }: CharacterProps) {
  const SKIN_L  = '#eef0f8'   // 쿨톤 밝은 피부
  const SKIN_M  = '#dce0f0'
  const SKIN_S  = '#b0b8d8'
  const HAIR    = '#1e2a5c'   // 딥 블루 블랙
  const HAIR_H  = '#4060a8'   // 블루 하이라이트
  const HAIR_L  = '#6080c8'   // 라이트 블루 하이라이트
  const OUTFIT1 = '#0e1428'   // 다크 네이비 의상
  const OUTFIT2 = '#1a2444'   // 미드 네이비
  const OUTFIT3 = '#2a3c6e'   // 하이라이트 네이비
  const CYBER   = '#00b8e8'   // 사이버 블루 포인트
  const CYBER2  = '#0066cc'   // 딥 사이버 블루
  const EYE_C   = '#00609a'   // 블루 눈
  const DARK    = '#06101e'
  const LIP     = '#a0b8e0'
  const PANTS   = '#101828'

  const eyeY  = emotion === 'happy' ? 92 : 88
  const eyeRy = emotion === 'happy' ? 5.5 : emotion === 'alert' ? 9.5 : 8

  const mouthD = speaking
    ? 'M 93 137 Q 108 152 123 137'
    : emotion === 'happy'    ? 'M 90 135 Q 108 150 126 135'
    : emotion === 'concerned'? 'M 95 141 Q 108 134 121 141'
    : emotion === 'humorous' ? 'M 90 135 Q 108 148 126 135 Q 120 154 96 154 Z'
    :                          'M 93 137 Q 108 144 123 137'

  return (
    <svg viewBox="0 0 220 500" width="220" height="500" style={{ overflow: 'visible' }}>
      <defs>
        <radialGradient id="joySkinFace" cx="50%" cy="36%" r="62%">
          <stop offset="0%"   stopColor="#f8faff" />
          <stop offset="40%"  stopColor={SKIN_L} />
          <stop offset="100%" stopColor={SKIN_M} />
        </radialGradient>
        <radialGradient id="joySkinArm" cx="35%" cy="28%" r="72%">
          <stop offset="0%"   stopColor={SKIN_L} />
          <stop offset="100%" stopColor={SKIN_M} />
        </radialGradient>
        <linearGradient id="joyHair" x1="5%" y1="0%" x2="95%" y2="100%">
          <stop offset="0%"   stopColor={HAIR_L} />
          <stop offset="30%"  stopColor={HAIR_H} />
          <stop offset="70%"  stopColor={HAIR} />
          <stop offset="100%" stopColor="#060e28" />
        </linearGradient>
        <linearGradient id="joyHairSide" x1="100%" y1="0%" x2="0%" y2="100%">
          <stop offset="0%"   stopColor={HAIR_H} />
          <stop offset="100%" stopColor={HAIR} />
        </linearGradient>
        <linearGradient id="joyOutfit" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%"   stopColor={OUTFIT3} />
          <stop offset="50%"  stopColor={OUTFIT2} />
          <stop offset="100%" stopColor={OUTFIT1} />
        </linearGradient>
        <linearGradient id="joyOutfitR" x1="100%" y1="0%" x2="0%" y2="100%">
          <stop offset="0%"   stopColor={OUTFIT3} />
          <stop offset="100%" stopColor={OUTFIT1} />
        </linearGradient>
        <radialGradient id="joyIris" cx="40%" cy="35%" r="58%">
          <stop offset="0%"   stopColor="#40c0f8" />
          <stop offset="50%"  stopColor={EYE_C} />
          <stop offset="100%" stopColor="#001830" />
        </radialGradient>
        <filter id="joyCyberGlow">
          <feGaussianBlur stdDeviation="2.5" result="b"/>
          <feMerge><feMergeNode in="b"/><feMergeNode in="SourceGraphic"/></feMerge>
        </filter>
        <radialGradient id="joyAura" cx="50%" cy="20%" r="80%">
          <stop offset="0%"   stopColor="#0060c0" stopOpacity="0.25" />
          <stop offset="100%" stopColor="#0060c0" stopOpacity="0" />
        </radialGradient>
        <linearGradient id="joyCyberLine" x1="0%" y1="0%" x2="100%" y2="0%">
          <stop offset="0%"   stopColor={CYBER} stopOpacity="0" />
          <stop offset="50%"  stopColor={CYBER} />
          <stop offset="100%" stopColor={CYBER} stopOpacity="0" />
        </linearGradient>
      </defs>

      {/* 사이버 오라 */}
      <ellipse cx="110" cy="250" rx="105" ry="220" fill="url(#joyAura)" />

      {/* 사이버 파티클 라인 */}
      {speaking && (
        <g filter="url(#joyCyberGlow)">
          {[-40, -20, 0, 20, 40].map((dx, i) => (
            <line key={i} x1={110 + dx} y1={160} x2={110 + dx * 1.5} y2={360}
              stroke={CYBER} strokeWidth="0.5" strokeOpacity="0.2"
              style={{ animation: `joyLine ${0.8 + i * 0.15}s ease-in-out infinite alternate` }} />
          ))}
        </g>
      )}

      {/* ═══ 팬츠 & 부츠 ═══ */}
      <g>
        {/* 스키니 팬츠 */}
        <path d="M 72 310 Q 70 360 68 430 Q 76 436 88 434 Q 90 368 92 318 Z"
          fill={PANTS} />
        <path d="M 128 318 Q 130 368 132 434 Q 144 436 152 430 Q 150 360 148 310 Z"
          fill={PANTS} />
        {/* 사이버 팬츠 라인 */}
        <line x1="80" y1="318" x2="80" y2="430" stroke={CYBER} strokeWidth="1" strokeOpacity="0.3" />
        <line x1="140" y1="318" x2="140" y2="430" stroke={CYBER} strokeWidth="1" strokeOpacity="0.3" />
        {/* 부츠 */}
        <path d="M 64 430 Q 60 448 82 452 Q 96 454 100 442 L 96 438 Q 92 446 84 444 Q 68 440 68 430 Z"
          fill={OUTFIT1} stroke={CYBER} strokeWidth="0.8" />
        <path d="M 56 446 Q 54 454 80 455 Q 96 455 100 447" fill="none" stroke={CYBER2} strokeWidth="1.5" />
        <path d="M 120 442 L 116 438 Q 112 446 116 444 Q 130 440 132 430 Q 136 440 148 444 Q 160 448 156 452 Q 144 456 120 454 Q 108 454 104 442 Z"
          fill={OUTFIT1} stroke={CYBER} strokeWidth="0.8" />
        <path d="M 160 446 Q 162 454 136 455 Q 120 455 116 447" fill="none" stroke={CYBER2} strokeWidth="1.5" />
        {/* 부츠 사이버 트림 */}
        <path d="M 66 436 Q 84 440 100 436" fill="none" stroke={CYBER} strokeWidth="1.5" strokeOpacity="0.5" />
        <path d="M 120 436 Q 138 440 154 436" fill="none" stroke={CYBER} strokeWidth="1.5" strokeOpacity="0.5" />
      </g>

      {/* ═══ 네이비 전투복 스타일 의상 ═══ */}
      <g style={{ animation: 'joyBreathe 3s ease-in-out infinite', transformOrigin: '110px 225px' }}>
        {/* 뒤판 */}
        <path d="M 54 152 Q 42 185 44 310 L 176 310 Q 178 185 166 152 Z"
          fill={OUTFIT2} />
        {/* 앞 좌측 */}
        <path d="M 54 152 Q 46 185 48 310 L 110 310 L 110 152 Z"
          fill="url(#joyOutfit)" />
        {/* 앞 우측 */}
        <path d="M 110 152 L 110 310 L 172 310 Q 174 185 166 152 Z"
          fill={OUTFIT2} />
        {/* 사이버 수트 패널 */}
        <path d="M 78 162 L 78 310 L 100 310 L 98 162 Z"
          fill={OUTFIT3} fillOpacity="0.3" />
        <path d="M 120 162 L 122 310 L 142 310 L 142 162 Z"
          fill={OUTFIT3} fillOpacity="0.3" />
        {/* 사이버 발광 선 */}
        {[180, 210, 250, 285].map(y => (
          <line key={y} x1="50" y1={y} x2="170" y2={y}
            stroke={CYBER} strokeWidth="1.2" strokeOpacity="0.25" />
        ))}
        {/* 하이넥 */}
        <path d="M 80 155 Q 92 144 110 142 Q 128 144 140 155 Q 124 150 110 152 Q 96 150 80 155 Z"
          fill={OUTFIT3} />
        {/* 하이넥 사이버 트림 */}
        <path d="M 80 155 Q 96 147 110 145 Q 124 147 140 155"
          fill="none" stroke={CYBER} strokeWidth="1.5" strokeOpacity="0.5" />
        {/* 가슴 로고 (육각형 사이버) */}
        <path d="M 106 196 L 110 190 L 114 196 L 114 204 L 110 210 L 106 204 Z"
          fill="none" stroke={CYBER} strokeWidth="1.5" />
        <circle cx="110" cy="200" r="4" fill={CYBER} fillOpacity="0.4" />
        {/* 어깨 패드 */}
        <path d="M 54 152 Q 60 144 74 148 Q 64 155 54 155 Z" fill={OUTFIT3} />
        <path d="M 166 152 Q 160 144 146 148 Q 156 155 166 155 Z" fill={OUTFIT3} />
        <path d="M 56 148 Q 66 142 74 148" fill="none" stroke={CYBER} strokeWidth="1.5" />
        <path d="M 164 148 Q 154 142 146 148" fill="none" stroke={CYBER} strokeWidth="1.5" />
        {/* 사이버 벨트 */}
        <path d="M 46 280 Q 110 288 174 280 Q 110 296 46 280 Z" fill={OUTFIT1} />
        <path d="M 46 280 Q 110 284 174 280" stroke={CYBER} strokeWidth="2" fill="none" strokeOpacity="0.6" />
        <rect x="106" y="278" width="8" height="10" rx="1" fill={CYBER} fillOpacity="0.6" />
      </g>

      {/* ═══ 왼팔 ═══ */}
      <g style={{
        transformOrigin: '52px 152px',
        animation: speaking
          ? 'joyArmSpeakL 0.85s ease-in-out infinite'
          : listening
          ? 'joyArmListen 0.4s ease forwards'
          : 'joyArmIdleL 4.2s ease-in-out infinite',
      }}>
        <path d="M 54 152 Q 32 176 22 240 Q 20 278 34 284 Q 46 288 54 276 Q 42 242 44 200 Q 48 174 58 155 Z"
          fill="url(#joyOutfitR)" />
        <path d="M 38 280 Q 38 290 56 278"
          stroke={CYBER} strokeWidth="1.5" fill="none" strokeOpacity="0.5" />
        {/* 사이버 소매 라인 */}
        <line x1="36" y1="200" x2="32" y2="270" stroke={CYBER} strokeWidth="0.8" strokeOpacity="0.25" />
        {/* 사이버 암 밴드 */}
        <path d="M 22 250 Q 42 256 54 250" fill="none" stroke={CYBER} strokeWidth="2" strokeOpacity="0.4" />
        {/* 피부 손 */}
        <ellipse cx="32" cy="292" rx="14" ry="11" fill="url(#joySkinArm)" />
        <path d="M 20 289 Q 18 282 22 279" stroke={SKIN_M} strokeWidth="2.5" fill="none" strokeLinecap="round" />
        <path d="M 27 287 Q 25 280 29 278" stroke={SKIN_M} strokeWidth="2.5" fill="none" strokeLinecap="round" />
        <path d="M 33 286 Q 32 279 35 278" stroke={SKIN_M} strokeWidth="2.5" fill="none" strokeLinecap="round" />
        <path d="M 39 287 Q 39 280 42 280" stroke={SKIN_M} strokeWidth="2.5" fill="none" strokeLinecap="round" />
        {/* 사이버 글로브 */}
        <path d="M 18 286 Q 16 294 32 298 Q 46 296 46 290" fill={OUTFIT1} fillOpacity="0.5" />
        <path d="M 18 286 Q 32 292 46 286" fill="none" stroke={CYBER} strokeWidth="1" strokeOpacity="0.4" />
      </g>

      {/* ═══ 오른팔 ═══ */}
      <g style={{
        transformOrigin: '168px 152px',
        animation: speaking
          ? 'joyArmSpeakR 0.85s ease-in-out infinite 0.28s'
          : 'joyArmIdleR 4.2s ease-in-out infinite 2s',
      }}>
        <path d="M 166 152 Q 188 176 198 240 Q 200 278 186 284 Q 174 288 166 276 Q 178 242 176 200 Q 172 174 162 155 Z"
          fill="url(#joyOutfit)" />
        <path d="M 182 280 Q 182 290 164 278"
          stroke={CYBER} strokeWidth="1.5" fill="none" strokeOpacity="0.5" />
        <line x1="184" y1="200" x2="188" y2="270" stroke={CYBER} strokeWidth="0.8" strokeOpacity="0.25" />
        <path d="M 198 250 Q 178 256 166 250" fill="none" stroke={CYBER} strokeWidth="2" strokeOpacity="0.4" />
        <ellipse cx="188" cy="292" rx="14" ry="11" fill="url(#joySkinArm)" />
        <path d="M 200 289 Q 202 282 198 279" stroke={SKIN_M} strokeWidth="2.5" fill="none" strokeLinecap="round" />
        <path d="M 193 287 Q 195 280 191 278" stroke={SKIN_M} strokeWidth="2.5" fill="none" strokeLinecap="round" />
        <path d="M 187 286 Q 188 279 185 278" stroke={SKIN_M} strokeWidth="2.5" fill="none" strokeLinecap="round" />
        <path d="M 181 287 Q 181 280 178 280" stroke={SKIN_M} strokeWidth="2.5" fill="none" strokeLinecap="round" />
        <path d="M 202 286 Q 204 294 188 298 Q 174 296 174 290" fill={OUTFIT1} fillOpacity="0.5" />
        <path d="M 202 286 Q 188 292 174 286" fill="none" stroke={CYBER} strokeWidth="1" strokeOpacity="0.4" />
      </g>

      {/* ═══ 목 ═══ */}
      <path d="M 96 152 Q 94 172 100 175 L 120 175 Q 126 172 124 152 Z"
        fill="url(#joySkinFace)" />
      <path d="M 104 155 L 102 173" stroke={SKIN_S} strokeWidth="1.5" strokeOpacity="0.2" />
      <path d="M 116 155 L 118 173" stroke={SKIN_S} strokeWidth="1.5" strokeOpacity="0.2" />

      {/* ═══ 머리 ═══ */}
      <g style={{
        transformOrigin: '110px 88px',
        animation: listening
          ? 'joyHeadTilt 0.5s ease forwards'
          : 'joyHeadBob 4.5s ease-in-out infinite',
      }}>
        {/* 헤어 뒤 */}
        <ellipse cx="110" cy="80" rx="54" ry="60" fill="url(#joyHair)" />
        {/* 롱 헤어 아래로 */}
        <path d="M 56 80 Q 38 110 40 160 Q 44 195 52 210 Q 58 195 58 165 Q 56 132 60 98 Z"
          fill="url(#joyHairSide)" />
        <path d="M 164 80 Q 182 110 180 160 Q 176 195 168 210 Q 162 195 162 165 Q 164 132 160 98 Z"
          fill="url(#joyHairSide)" />
        {/* 헤어 뒤 롱 */}
        <path d="M 64 86 Q 50 130 52 200 Q 54 220 58 230 Q 62 215 62 180 Q 60 150 64 120 Z"
          fill={HAIR} />
        <path d="M 156 86 Q 170 130 168 200 Q 166 220 162 230 Q 158 215 158 180 Q 160 150 156 120 Z"
          fill={HAIR} />

        {/* 얼굴 */}
        <path d="M 58 74 Q 56 104 60 126 Q 64 150 78 164 Q 94 174 110 174 Q 126 174 142 164 Q 156 150 160 126 Q 164 104 162 74 Q 156 42 130 30 Q 110 24 90 30 Q 62 42 58 74 Z"
          fill="url(#joySkinFace)" />

        {/* 쿨톤 측면 음영 */}
        <path d="M 58 72 Q 56 104 60 126 Q 64 150 76 164 Q 68 150 64 127 Q 60 106 62 77 Z"
          fill={SKIN_S} fillOpacity="0.2" />
        <path d="M 162 72 Q 164 104 160 126 Q 156 150 144 164 Q 152 150 156 127 Q 160 106 158 77 Z"
          fill={SKIN_S} fillOpacity="0.2" />

        {/* 앞머리 (에지 있는 비대칭 뱅) */}
        <path d="M 60 72 Q 68 34 90 28 Q 80 42 76 60 Q 72 72 68 80 Z"
          fill={HAIR} />
        {/* 헤어 슬릭백 앞머리 */}
        <path d="M 62 70 Q 70 32 96 26 Q 84 38 80 56 Q 76 68 70 78 Q 65 74 62 70 Z"
          fill={HAIR_H} fillOpacity="0.5" />
        {/* 사이드 파트 */}
        <path d="M 96 26 Q 108 14 126 22 Q 112 24 104 36 Q 98 46 96 60 Q 93 46 96 26 Z"
          fill={HAIR_L} fillOpacity="0.45" />
        {/* 헤어 하이라이트 (블루 광택) */}
        <path d="M 80 30 Q 96 24 112 28"
          fill="none" stroke={HAIR_L} strokeWidth="4" strokeOpacity="0.45" strokeLinecap="round" />
        <path d="M 74 40 Q 88 32 104 36"
          fill="none" stroke={HAIR_H} strokeWidth="2.5" strokeOpacity="0.35" strokeLinecap="round" />
        {/* 사이버 헤어 LED */}
        <path d="M 62 70 Q 80 60 100 64"
          fill="none" stroke={CYBER} strokeWidth="1.5" strokeOpacity="0.2" filter="url(#joyCyberGlow)" />

        {/* 귀 */}
        <path d="M 58 94 Q 52 99 52 108 Q 52 118 58 122 Q 62 124 64 120 Q 60 116 60 108 Q 60 100 64 96 Z"
          fill={SKIN_M} />
        <path d="M 162 94 Q 168 99 168 108 Q 168 118 162 122 Q 158 124 156 120 Q 160 116 160 108 Q 160 100 156 96 Z"
          fill={SKIN_M} />
        {/* 사이버 귀걸이 */}
        <rect x="50" y="118" width="8" height="10" rx="2" fill={OUTFIT1} stroke={CYBER} strokeWidth="1" />
        <circle cx="54" cy="123" r="2" fill={CYBER} fillOpacity="0.7" filter="url(#joyCyberGlow)" />
        <rect x="162" y="118" width="8" height="10" rx="2" fill={OUTFIT1} stroke={CYBER} strokeWidth="1" />
        <circle cx="166" cy="123" r="2" fill={CYBER} fillOpacity="0.7" filter="url(#joyCyberGlow)" />

        {/* ─ 눈썹 (샤프, 에지) ─ */}
        <path
          d={emotion === 'concerned'
            ? 'M 73 79 Q 87 75 101 80'
            : emotion === 'happy'
            ? 'M 73 74 Q 87 68 101 73'
            : 'M 73 76 Q 87 70 101 75'}
          fill="none" stroke={HAIR} strokeWidth="4" strokeLinecap="square" />
        <path
          d={emotion === 'concerned'
            ? 'M 119 80 Q 133 75 147 79'
            : emotion === 'happy'
            ? 'M 119 73 Q 133 68 147 74'
            : 'M 119 75 Q 133 70 147 76'}
          fill="none" stroke={HAIR} strokeWidth="4" strokeLinecap="square" />

        {/* ─ 눈 (강렬한 사이버 블루) ─ */}
        <ellipse cx="87" cy={eyeY - 4} rx="15" ry="7" fill={SKIN_S} fillOpacity="0.1" />
        <ellipse cx="133" cy={eyeY - 4} rx="15" ry="7" fill={SKIN_S} fillOpacity="0.1" />
        <path d={`M 72 ${eyeY} Q 87 ${eyeY - eyeRy - 3} 102 ${eyeY} Q 87 ${eyeY + eyeRy - 1} 72 ${eyeY} Z`}
          fill="white" />
        <path d={`M 118 ${eyeY} Q 133 ${eyeY - eyeRy - 3} 148 ${eyeY} Q 133 ${eyeY + eyeRy - 1} 118 ${eyeY} Z`}
          fill="white" />
        {/* 홍채 */}
        <ellipse cx="87" cy={eyeY} rx="9.5" ry={eyeRy} fill="url(#joyIris)" />
        <ellipse cx="133" cy={eyeY} rx="9.5" ry={eyeRy} fill="url(#joyIris)" />
        {/* 눈동자 */}
        <circle cx="87" cy={eyeY + 1} r="5.5" fill={DARK} />
        <circle cx="133" cy={eyeY + 1} r="5.5" fill={DARK} />
        {/* 사이버 눈 반짝임 */}
        <ellipse cx="89" cy={eyeY - 3} rx="3.5" ry="2.5" fill="white" fillOpacity="0.92" />
        <ellipse cx="84" cy={eyeY + 2} rx="1.8" ry="1.2" fill={CYBER} fillOpacity="0.6" />
        <ellipse cx="135" cy={eyeY - 3} rx="3.5" ry="2.5" fill="white" fillOpacity="0.92" />
        <ellipse cx="130" cy={eyeY + 2} rx="1.8" ry="1.2" fill={CYBER} fillOpacity="0.6" />
        {/* 아이라인 (날카로운 캣아이 + 언더라인) */}
        <path d={`M 70 ${eyeY - 1} Q 87 ${eyeY - eyeRy - 4} 104 ${eyeY - 2} L 110 ${eyeY - 9}`}
          fill="none" stroke={DARK} strokeWidth="3" strokeLinecap="round" />
        <path d={`M 116 ${eyeY - 2} Q 133 ${eyeY - eyeRy - 4} 150 ${eyeY - 1} L 144 ${eyeY - 9}`}
          fill="none" stroke={DARK} strokeWidth="3" strokeLinecap="round" />
        {/* 언더라인 (사이버 느낌) */}
        <path d={`M 74 ${eyeY + eyeRy - 1} Q 87 ${eyeY + eyeRy + 3} 100 ${eyeY + eyeRy - 1}`}
          fill="none" stroke={CYBER} strokeWidth="1" strokeOpacity="0.35" />
        <path d={`M 120 ${eyeY + eyeRy - 1} Q 133 ${eyeY + eyeRy + 3} 146 ${eyeY + eyeRy - 1}`}
          fill="none" stroke={CYBER} strokeWidth="1" strokeOpacity="0.35" />
        {/* 속눈썹 */}
        {[-5,-3,-1,1,3,5,8].map(dx => (
          <line key={dx} x1={87+dx} y1={eyeY-eyeRy-2} x2={86+dx*0.88} y2={eyeY-eyeRy-8}
            stroke={DARK} strokeWidth="2" strokeLinecap="round" />
        ))}
        {[-5,-3,-1,1,3,5,8].map(dx => (
          <line key={dx} x1={133+dx} y1={eyeY-eyeRy-2} x2={132+dx*0.88} y2={eyeY-eyeRy-8}
            stroke={DARK} strokeWidth="2" strokeLinecap="round" />
        ))}
        {/* 아이섀도우 (블루) */}
        <ellipse cx="87" cy={eyeY - eyeRy + 1} rx="12" ry="3" fill={CYBER2} fillOpacity="0.2" />
        <ellipse cx="133" cy={eyeY - eyeRy + 1} rx="12" ry="3" fill={CYBER2} fillOpacity="0.2" />

        {/* ─ 코 ─ */}
        <path d="M 106 110 Q 104 118 106 127 Q 110 131 114 127 Q 116 118 114 110"
          fill="none" stroke={SKIN_S} strokeWidth="1.5" strokeOpacity="0.4" strokeLinecap="round" />
        <path d="M 103 125 Q 110 129 117 125"
          fill="none" stroke={SKIN_S} strokeWidth="1.4" strokeOpacity="0.35" />

        {/* ─ 입 ─ */}
        <path d="M 91 134 Q 101 128 110 132 Q 119 128 129 134 Q 121 138 110 139 Q 99 138 91 134 Z"
          fill={LIP} />
        {!speaking ? (
          <path d="M 91 134 Q 99 145 110 147 Q 121 145 129 134 Q 121 141 110 142 Q 99 141 91 134 Z"
            fill="#8898c8" fillOpacity="0.9" />
        ) : (
          <>
            <path d={mouthD} fill="rgba(10,20,50,0.7)" />
            <rect x="104" y="135" width="12" height="9" rx="2" fill="white" fillOpacity="0.85" />
          </>
        )}
        <ellipse cx="103" cy="132" rx="7" ry="2.5" fill="white" fillOpacity="0.2" transform="rotate(-5,103,132)" />
        {/* 아랫입술 사이버 틴트 */}
        {!speaking && (
          <path d="M 98 140 Q 110 145 122 140" fill="none" stroke={CYBER} strokeWidth="0.8" strokeOpacity="0.2" />
        )}

        {/* 쿨톤 볼 홍조 */}
        <ellipse cx="72" cy="118" rx="13" ry="8" fill="#8090d8" fillOpacity="0.1" />
        <ellipse cx="148" cy="118" rx="13" ry="8" fill="#8090d8" fillOpacity="0.1" />

        {/* 페이스 하이라이트 */}
        <ellipse cx="70" cy="110" rx="14" ry="8" fill="white" fillOpacity="0.07" transform="rotate(-10,70,110)" />
        <ellipse cx="150" cy="110" rx="14" ry="8" fill="white" fillOpacity="0.07" transform="rotate(10,150,110)" />

        {/* 이마 사이버 문신 (옵션) */}
        <path d="M 104 60 Q 110 56 116 60" fill="none" stroke={CYBER} strokeWidth="1" strokeOpacity="0.2" />

        {/* 턱 음영 */}
        <path d="M 82 160 Q 110 170 138 160 Q 110 176 82 160 Z"
          fill={SKIN_S} fillOpacity="0.14" />
      </g>

      <style>{`
        @keyframes joyBreathe {
          0%,100% { transform: scaleY(1) translateY(0); }
          50% { transform: scaleY(1.013) translateY(-2px); }
        }
        @keyframes joyHeadBob {
          0%,100% { transform: translateY(0) rotate(0deg); }
          25% { transform: translateY(-6px) rotate(1deg); }
          75% { transform: translateY(2px) rotate(-0.7deg); }
        }
        @keyframes joyHeadTilt {
          to { transform: rotate(-11deg) translateX(-6px); }
        }
        @keyframes joyArmIdleL {
          0%,100% { transform: rotate(0deg); }
          50% { transform: rotate(2.5deg); }
        }
        @keyframes joyArmIdleR {
          0%,100% { transform: rotate(0deg); }
          50% { transform: rotate(-2.5deg); }
        }
        @keyframes joyArmSpeakL {
          0%,100% { transform: rotate(-5deg) translateY(0); }
          50% { transform: rotate(16deg) translateY(-14px); }
        }
        @keyframes joyArmSpeakR {
          0%,100% { transform: rotate(5deg) translateY(0); }
          50% { transform: rotate(-14deg) translateY(-12px); }
        }
        @keyframes joyArmListen {
          to { transform: rotate(-10deg); }
        }
        @keyframes joyLine {
          0% { opacity: 0.1; }
          100% { opacity: 0.4; }
        }
      `}</style>
    </svg>
  )
}
