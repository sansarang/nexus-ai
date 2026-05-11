import type { CharacterProps } from './types'

/**
 * 진 — K-pop 남성 아이돌 스타일
 * 참고: 첨부 이미지 (단발 다크 헤어, 데님 재킷, 세련된 얼굴)
 * 리얼리스틱한 3D 렌더링 느낌의 고품질 SVG
 */
export function Jin({ emotion, speaking, listening }: CharacterProps) {
  /* ── 팔레트 ── */
  const SKIN_L  = '#f7e8d8'   // 밝은 피부
  const SKIN_M  = '#f0d0b0'   // 중간 피부
  const SKIN_S  = '#d4a882'   // 어두운 피부 (그림자)
  const HAIR    = '#1a1a26'   // 다크 네이비 블랙
  const HAIR_H  = '#3a3a52'   // 헤어 하이라이트
  const JACKET  = '#3d5a8a'   // 데님 블루
  const JACKET_D= '#2a3f60'   // 데님 어두운
  const JACKET_L= '#5577aa'   // 데님 밝은
  const INNER   = '#f8f4ee'   // 이너 셔츠
  const DARK    = '#1a1022'
  const EYE_C   = '#6b4e3d'   // 눈 갈색
  const PANTS   = '#2c2c3e'   // 다크 팬츠
  const SHOES   = '#1a1a1a'   // 슈즈

  const eyeY  = emotion === 'happy' ? 90 : 88
  const eyeRy = emotion === 'happy' ? 5 : emotion === 'alert' ? 8 : 7
  const eyeLid = emotion === 'happy' ? 'M 78 85 Q 89 80 98 85' : 'M 76 84 Q 88 79 100 84'

  const mouthD =
    speaking           ? 'M 94 130 Q 108 142 122 130'
    : emotion === 'happy'    ? 'M 90 128 Q 108 143 124 128'
    : emotion === 'concerned'? 'M 94 133 Q 108 127 122 133'
    : emotion === 'humorous' ? 'M 92 128 Q 108 140 124 128 Q 118 147 98 147 Z'
    :                          'M 93 130 Q 108 138 121 130'

  return (
    <svg viewBox="0 0 220 500" width="220" height="500" style={{ overflow: 'visible' }}>
      <defs>
        {/* 피부 그라디언트 */}
        <radialGradient id="jinSkinFace" cx="52%" cy="38%" r="62%">
          <stop offset="0%"   stopColor="#fff5eb" />
          <stop offset="45%"  stopColor={SKIN_L} />
          <stop offset="100%" stopColor={SKIN_M} />
        </radialGradient>
        <radialGradient id="jinSkinArm" cx="40%" cy="30%" r="70%">
          <stop offset="0%"   stopColor={SKIN_L} />
          <stop offset="100%" stopColor={SKIN_M} />
        </radialGradient>
        {/* 헤어 그라디언트 */}
        <linearGradient id="jinHair" x1="20%" y1="0%" x2="80%" y2="100%">
          <stop offset="0%"   stopColor={HAIR_H} />
          <stop offset="50%"  stopColor={HAIR} />
          <stop offset="100%" stopColor="#0d0d18" />
        </linearGradient>
        {/* 재킷 그라디언트 */}
        <linearGradient id="jinJacket" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%"   stopColor={JACKET_L} />
          <stop offset="50%"  stopColor={JACKET} />
          <stop offset="100%" stopColor={JACKET_D} />
        </linearGradient>
        <linearGradient id="jinJacketL" x1="100%" y1="0%" x2="0%" y2="100%">
          <stop offset="0%"   stopColor={JACKET_L} />
          <stop offset="100%" stopColor={JACKET_D} />
        </linearGradient>
        {/* 눈 그라디언트 */}
        <radialGradient id="jinIris" cx="45%" cy="38%" r="58%">
          <stop offset="0%"   stopColor="#9b7a65" />
          <stop offset="60%"  stopColor={EYE_C} />
          <stop offset="100%" stopColor="#3d2215" />
        </radialGradient>
        {/* 팬츠 */}
        <linearGradient id="jinPants" x1="0%" y1="0%" x2="100%" y2="0%">
          <stop offset="0%"   stopColor="#252535" />
          <stop offset="50%"  stopColor={PANTS} />
          <stop offset="100%" stopColor="#252535" />
        </linearGradient>
        {/* 그림자 필터 */}
        <filter id="jinSoftShadow">
          <feGaussianBlur stdDeviation="1.5" result="blur"/>
          <feComposite in="SourceGraphic" in2="blur" operator="over"/>
        </filter>
        <filter id="jinGlow">
          <feGaussianBlur stdDeviation="2" result="blur"/>
          <feMerge><feMergeNode in="blur"/><feMergeNode in="SourceGraphic"/></feMerge>
        </filter>
      </defs>

      {/* ═══ 다리 / 팬츠 ═══ */}
      <g>
        {/* 왼다리 */}
        <path d="M 74 310 Q 72 360 70 420 Q 76 426 86 424 Q 90 368 92 318 Z"
          fill="url(#jinPants)" />
        {/* 오른다리 */}
        <path d="M 128 318 Q 130 368 134 424 Q 144 426 150 420 Q 148 360 146 310 Z"
          fill="url(#jinPants)" />
        {/* 무릎 하이라이트 */}
        <ellipse cx="81" cy="368" rx="10" ry="6" fill="#383848" fillOpacity="0.6" />
        <ellipse cx="139" cy="368" rx="10" ry="6" fill="#383848" fillOpacity="0.6" />
        {/* 신발 */}
        <path d="M 68 420 Q 66 434 88 438 Q 100 439 102 430 L 96 426 Q 92 435 76 432 Q 66 429 68 420 Z"
          fill={SHOES} />
        <path d="M 132 430 L 126 426 Q 122 435 136 432 Q 152 429 152 420 Q 154 434 134 438 Q 122 439 132 430 Z"
          fill={SHOES} />
        {/* 신발 밑창 */}
        <path d="M 68 432 Q 85 440 100 436" fill="none" stroke="#333" strokeWidth="2" />
        <path d="M 122 436 Q 138 440 154 432" fill="none" stroke="#333" strokeWidth="2" />
      </g>

      {/* ═══ 데님 재킷 ═══ */}
      <g style={{ animation: 'jinBreathe 3.5s ease-in-out infinite', transformOrigin: '110px 220px' }}>
        {/* 재킷 뒤판 & 옆구리 */}
        <path d="M 54 160 Q 40 185 42 310 L 178 310 Q 180 185 166 160 Z"
          fill="url(#jinJacket)" />
        {/* 재킷 앞 좌측 */}
        <path d="M 54 160 Q 46 182 48 310 L 110 310 L 110 160 Z"
          fill={JACKET} />
        {/* 재킷 앞 우측 */}
        <path d="M 110 160 L 110 310 L 172 310 Q 174 182 166 160 Z"
          fill={JACKET_D} />
        {/* 이너 셔츠 앞 */}
        <path d="M 86 162 Q 92 175 110 180 Q 128 175 134 162 L 124 160 Q 118 172 110 175 Q 102 172 96 160 Z"
          fill={INNER} />
        {/* 재킷 칼라 */}
        <path d="M 86 162 Q 80 150 74 155 Q 72 165 80 168 Q 88 165 86 162 Z"
          fill={JACKET_L} />
        <path d="M 134 162 Q 140 150 146 155 Q 148 165 140 168 Q 132 165 134 162 Z"
          fill={JACKET} />
        {/* 재킷 단추 */}
        {[195, 220, 245, 270].map(y => (
          <circle key={y} cx="110" cy={y} r="4" fill={JACKET_D} stroke={JACKET_L} strokeWidth="1" />
        ))}
        {/* 데님 스티치 라인 */}
        <line x1="48" y1="165" x2="48" y2="300" stroke={JACKET_L} strokeWidth="0.8" strokeDasharray="4,3" strokeOpacity="0.5" />
        <line x1="172" y1="165" x2="172" y2="300" stroke={JACKET_L} strokeWidth="0.8" strokeDasharray="4,3" strokeOpacity="0.5" />
        <line x1="60" y1="200" x2="84" y2="205" stroke={JACKET_L} strokeWidth="0.7" strokeOpacity="0.4" />
        {/* 포켓 */}
        <rect x="56" y="210" width="28" height="22" rx="3" fill={JACKET_D} stroke={JACKET_L} strokeWidth="0.8" strokeOpacity="0.6" />
        <rect x="136" y="210" width="28" height="22" rx="3" fill={JACKET_D} stroke={JACKET_L} strokeWidth="0.8" strokeOpacity="0.6" />
        {/* 어깨 봉제선 */}
        <path d="M 54 160 Q 65 155 80 160" fill="none" stroke={JACKET_L} strokeWidth="1" strokeOpacity="0.6" />
        <path d="M 140 160 Q 155 155 166 160" fill="none" stroke={JACKET_L} strokeWidth="1" strokeOpacity="0.6" />
      </g>

      {/* ═══ 왼팔 ═══ */}
      <g style={{
        transformOrigin: '50px 162px',
        animation: speaking
          ? 'jinArmSpeakL 1s ease-in-out infinite'
          : listening
          ? 'jinArmListen 0.4s ease forwards'
          : 'jinArmIdleL 4s ease-in-out infinite',
      }}>
        <path d="M 54 160 Q 34 185 28 240 Q 30 280 38 285 Q 46 290 54 280 Q 48 240 50 200 Q 54 175 60 162 Z"
          fill="url(#jinJacketL)" />
        {/* 소매 끝 */}
        <path d="M 30 276 Q 34 288 50 282 Q 48 272 34 268 Z" fill={JACKET_D} />
        {/* 손목 & 손 */}
        <ellipse cx="38" cy="292" rx="12" ry="10" fill="url(#jinSkinArm)" />
        {/* 손가락 */}
        <path d="M 28 289 Q 26 283 30 280" stroke={SKIN_M} strokeWidth="2.5" fill="none" strokeLinecap="round" />
        <path d="M 33 287 Q 31 280 35 278" stroke={SKIN_M} strokeWidth="2.5" fill="none" strokeLinecap="round" />
        <path d="M 39 286 Q 38 279 41 278" stroke={SKIN_M} strokeWidth="2.5" fill="none" strokeLinecap="round" />
        <path d="M 44 287 Q 44 280 47 280" stroke={SKIN_M} strokeWidth="2.5" fill="none" strokeLinecap="round" />
        {/* 손 음영 */}
        <ellipse cx="36" cy="294" rx="8" ry="5" fill={SKIN_S} fillOpacity="0.3" />
      </g>

      {/* ═══ 오른팔 ═══ */}
      <g style={{
        transformOrigin: '170px 162px',
        animation: speaking
          ? 'jinArmSpeakR 1s ease-in-out infinite 0.35s'
          : 'jinArmIdleR 4s ease-in-out infinite 2s',
      }}>
        <path d="M 166 160 Q 186 185 192 240 Q 190 280 182 285 Q 174 290 166 280 Q 172 240 170 200 Q 166 175 160 162 Z"
          fill="url(#jinJacket)" />
        <path d="M 190 276 Q 186 288 170 282 Q 172 272 186 268 Z" fill={JACKET_D} />
        <ellipse cx="182" cy="292" rx="12" ry="10" fill="url(#jinSkinArm)" />
        <path d="M 192 289 Q 194 283 190 280" stroke={SKIN_M} strokeWidth="2.5" fill="none" strokeLinecap="round" />
        <path d="M 187 287 Q 189 280 185 278" stroke={SKIN_M} strokeWidth="2.5" fill="none" strokeLinecap="round" />
        <path d="M 181 286 Q 182 279 179 278" stroke={SKIN_M} strokeWidth="2.5" fill="none" strokeLinecap="round" />
        <path d="M 176 287 Q 176 280 173 280" stroke={SKIN_M} strokeWidth="2.5" fill="none" strokeLinecap="round" />
        <ellipse cx="184" cy="294" rx="8" ry="5" fill={SKIN_S} fillOpacity="0.3" />
      </g>

      {/* ═══ 목 ═══ */}
      <path d="M 96 155 Q 94 175 98 178 L 122 178 Q 126 175 124 155 Z" fill="url(#jinSkinFace)" />
      {/* 목 음영 */}
      <path d="M 104 158 L 102 176" stroke={SKIN_S} strokeWidth="2" strokeOpacity="0.25" />
      <path d="M 116 158 L 118 176" stroke={SKIN_S} strokeWidth="2" strokeOpacity="0.25" />
      {/* 쇄골 */}
      <path d="M 72 168 Q 90 164 110 166" fill="none" stroke={SKIN_S} strokeWidth="1.5" strokeOpacity="0.35" />
      <path d="M 110 166 Q 130 164 148 168" fill="none" stroke={SKIN_S} strokeWidth="1.5" strokeOpacity="0.35" />

      {/* ═══ 머리 ═══ */}
      <g style={{
        transformOrigin: '110px 90px',
        animation: listening
          ? 'jinHeadTilt 0.4s ease forwards'
          : 'jinHeadBob 5s ease-in-out infinite',
      }}>
        {/* 헤어 뒷부분 */}
        <ellipse cx="110" cy="80" rx="52" ry="58" fill="url(#jinHair)" />

        {/* 얼굴 형태 */}
        <path d="M 60 70 Q 58 100 62 120 Q 66 145 80 158 Q 95 168 110 168 Q 125 168 140 158 Q 154 145 158 120 Q 162 100 160 70 Q 156 40 130 30 Q 110 24 90 30 Q 64 40 60 70 Z"
          fill="url(#jinSkinFace)" />

        {/* 페이스 측면 음영 (입체감) */}
        <path d="M 60 68 Q 58 100 62 120 Q 66 145 78 158 Q 70 148 66 128 Q 62 108 64 78 Z"
          fill={SKIN_S} fillOpacity="0.2" />
        <path d="M 160 68 Q 162 100 158 120 Q 154 145 142 158 Q 150 148 154 128 Q 158 108 156 78 Z"
          fill={SKIN_S} fillOpacity="0.2" />

        {/* 앞머리 — 자연스러운 K-pop 스타일 */}
        <path d="M 62 68 Q 70 28 110 22 Q 150 28 158 68 Q 148 38 130 32 Q 110 26 90 32 Q 72 38 62 68 Z"
          fill="url(#jinHair)" />
        {/* 앞머리 레이어 1 — 오른쪽으로 쓸린 앞머리 */}
        <path d="M 64 66 Q 72 38 96 30 Q 88 44 82 60 Q 76 72 70 78 Q 66 72 64 66 Z"
          fill={HAIR} />
        <path d="M 75 26 Q 85 16 100 18 Q 88 22 82 36 Q 76 50 72 64 Z"
          fill={HAIR} />
        {/* 앞머리 가르마 뱅 */}
        <path d="M 96 24 Q 108 14 124 20 Q 112 22 104 34 Q 98 44 95 58 Q 92 44 96 24 Z"
          fill={HAIR_H} fillOpacity="0.5" />
        {/* 헤어 하이라이트 광택 */}
        <path d="M 88 28 Q 104 22 120 26"
          fill="none" stroke={HAIR_H} strokeWidth="3" strokeOpacity="0.4" strokeLinecap="round" />
        <path d="M 80 36 Q 94 28 108 30"
          fill="none" stroke="#5a5a7a" strokeWidth="2" strokeOpacity="0.3" strokeLinecap="round" />

        {/* 귀 */}
        <path d="M 60 90 Q 54 95 54 104 Q 54 114 60 118 Q 64 120 66 116 Q 62 112 62 104 Q 62 96 66 92 Q 63 90 60 90 Z"
          fill={SKIN_M} />
        <path d="M 160 90 Q 166 95 166 104 Q 166 114 160 118 Q 156 120 154 116 Q 158 112 158 104 Q 158 96 154 92 Q 157 90 160 90 Z"
          fill={SKIN_M} />
        {/* 귓구멍 */}
        <path d="M 60 100 Q 60 106 62 108 Q 61 104 61 100 Z" fill={SKIN_S} fillOpacity="0.4" />
        <path d="M 160 100 Q 160 106 158 108 Q 159 104 159 100 Z" fill={SKIN_S} fillOpacity="0.4" />

        {/* ─ 눈썹 (자연스러운 K-pop 남성 눈썹) ─ */}
        <path
          d={emotion === 'concerned'
            ? 'M 76 76 Q 89 72 102 77'
            : emotion === 'happy'
            ? 'M 76 73 Q 89 68 102 73'
            : 'M 76 75 Q 89 70 102 75'}
          fill="none" stroke={HAIR} strokeWidth="3.5" strokeLinecap="round" />
        <path
          d={emotion === 'concerned'
            ? 'M 118 77 Q 131 72 144 76'
            : emotion === 'happy'
            ? 'M 118 73 Q 131 68 144 73'
            : 'M 118 75 Q 131 70 144 75'}
          fill="none" stroke={HAIR} strokeWidth="3.5" strokeLinecap="round" />
        {/* 눈썹 아래 그림자 */}
        <path
          d={emotion === 'happy' ? 'M 78 74 Q 89 71 100 75' : 'M 78 77 Q 89 73 100 77'}
          fill="none" stroke={SKIN_S} strokeWidth="1.5" strokeOpacity="0.25" strokeLinecap="round" />
        <path
          d={emotion === 'happy' ? 'M 120 75 Q 131 71 142 74' : 'M 120 77 Q 131 73 142 77'}
          fill="none" stroke={SKIN_S} strokeWidth="1.5" strokeOpacity="0.25" strokeLinecap="round" />

        {/* ─ 눈 (K-pop 남성 세련된 눈) ─ */}
        {/* 눈두덩 그림자 */}
        <ellipse cx="89" cy={eyeY - 3} rx="14" ry="6" fill={SKIN_S} fillOpacity="0.12" />
        <ellipse cx="131" cy={eyeY - 3} rx="14" ry="6" fill={SKIN_S} fillOpacity="0.12" />
        {/* 눈 흰자 */}
        <path d={`M 76 ${eyeY} Q 89 ${eyeY - eyeRy - 2} 102 ${eyeY} Q 89 ${eyeY + eyeRy - 1} 76 ${eyeY} Z`}
          fill="white" />
        <path d={`M 118 ${eyeY} Q 131 ${eyeY - eyeRy - 2} 144 ${eyeY} Q 131 ${eyeY + eyeRy - 1} 118 ${eyeY} Z`}
          fill="white" />
        {/* 홍채 */}
        <ellipse cx="89" cy={eyeY} rx="8.5" ry={eyeRy} fill="url(#jinIris)" />
        <ellipse cx="131" cy={eyeY} rx="8.5" ry={eyeRy} fill="url(#jinIris)" />
        {/* 눈동자 */}
        <circle cx="89" cy={eyeY + 1} r="5" fill={DARK} />
        <circle cx="131" cy={eyeY + 1} r="5" fill={DARK} />
        {/* 반짝임 */}
        <ellipse cx="91" cy={eyeY - 2} rx="3" ry="2.2" fill="white" fillOpacity="0.9" />
        <circle cx="85" cy={eyeY + 2} r="1.2" fill="white" fillOpacity="0.5" />
        <ellipse cx="133" cy={eyeY - 2} rx="3" ry="2.2" fill="white" fillOpacity="0.9" />
        <circle cx="127" cy={eyeY + 2} r="1.2" fill="white" fillOpacity="0.5" />
        {/* 아이라인 위 */}
        <path d={eyeLid} fill="none" stroke={DARK} strokeWidth="2.5" strokeLinecap="round" />
        <path d={eyeLid.replace('78', '118').replace('89', '131').replace('98', '144')}
          fill="none" stroke={DARK} strokeWidth="2.5" strokeLinecap="round" />
        {/* 아이라인 아래 */}
        <path d={`M 78 ${eyeY + 1} Q 89 ${eyeY + eyeRy + 1} 100 ${eyeY + 1}`}
          fill="none" stroke={DARK} strokeWidth="1.5" strokeLinecap="round" strokeOpacity="0.7" />
        <path d={`M 120 ${eyeY + 1} Q 131 ${eyeY + eyeRy + 1} 142 ${eyeY + 1}`}
          fill="none" stroke={DARK} strokeWidth="1.5" strokeLinecap="round" strokeOpacity="0.7" />
        {/* 속눈썹 */}
        {[-4,-2,0,2,4].map(dx => (
          <line key={dx} x1={89+dx} y1={eyeY-eyeRy-1} x2={88+dx*0.9} y2={eyeY-eyeRy-5}
            stroke={DARK} strokeWidth="1.8" strokeLinecap="round" />
        ))}
        {[-4,-2,0,2,4].map(dx => (
          <line key={dx} x1={131+dx} y1={eyeY-eyeRy-1} x2={130+dx*0.9} y2={eyeY-eyeRy-5}
            stroke={DARK} strokeWidth="1.8" strokeLinecap="round" />
        ))}

        {/* ─ 코 (입체적) ─ */}
        <path d="M 106 108 Q 104 118 106 124 Q 110 128 114 124 Q 116 118 114 108"
          fill="none" stroke={SKIN_S} strokeWidth="1.5" strokeOpacity="0.5" strokeLinecap="round" />
        <path d="M 104 122 Q 110 126 116 122"
          fill="none" stroke={SKIN_S} strokeWidth="1.5" strokeOpacity="0.4" />
        {/* 콧볼 그림자 */}
        <ellipse cx="103" cy="122" rx="5" ry="4" fill={SKIN_S} fillOpacity="0.12" />
        <ellipse cx="117" cy="122" rx="5" ry="4" fill={SKIN_S} fillOpacity="0.12" />

        {/* ─ 입 ─ */}
        {/* 입술 윗쪽 */}
        <path d="M 94 128 Q 102 124 110 128 Q 118 124 122 128"
          fill={SKIN_M} stroke={SKIN_S} strokeWidth="0.5" strokeOpacity="0.5" />
        {/* 입 경계 */}
        <path d={mouthD}
          fill={speaking ? 'rgba(80,30,20,0.6)' : 'none'}
          stroke={SKIN_S} strokeWidth="1.5" strokeLinecap="round" strokeOpacity="0.7" />
        {/* 아랫입술 */}
        {!speaking && (
          <path d={`M ${emotion === 'happy' ? '93' : '95'} ${emotion === 'happy' ? '139' : '135'} Q 108 ${emotion === 'happy' ? '145' : '142'} ${emotion === 'happy' ? '123' : '121'} ${emotion === 'happy' ? '139' : '135'}`}
            fill={SKIN_L} fillOpacity="0.4" stroke="none" />
        )}
        {speaking && <rect x="100" y="130" width="16" height="8" rx="2" fill="white" fillOpacity="0.85" />}
        {/* 입술 광택 */}
        <path d="M 97 128 Q 108 126 119 128"
          fill="none" stroke="white" strokeWidth="1" strokeOpacity="0.25" />

        {/* 볼 뼈 하이라이트 */}
        <ellipse cx="72" cy="112" rx="14" ry="8" fill="white" fillOpacity="0.08" transform="rotate(-10,72,112)" />
        <ellipse cx="148" cy="112" rx="14" ry="8" fill="white" fillOpacity="0.08" transform="rotate(10,148,112)" />

        {/* 아랫 턱 음영 */}
        <path d="M 84 155 Q 110 162 136 155 Q 110 170 84 155 Z"
          fill={SKIN_S} fillOpacity="0.15" />
      </g>

      {/* 작은 핑크 버니 마스코트 (어깨 위) */}
      <g style={{ animation: 'jinBunny 3s ease-in-out infinite', transformOrigin: '168px 148px' }}>
        {/* 바디 */}
        <ellipse cx="168" cy="152" rx="14" ry="16" fill="#ffb3c6" />
        {/* 귀 */}
        <path d="M 162 138 Q 159 124 162 118 Q 165 122 166 134 Z" fill="#ffb3c6" />
        <path d="M 173 137 Q 176 123 173 117 Q 170 121 169 133 Z" fill="#ffb3c6" />
        <path d="M 162 136 Q 160 124 162 120 Q 164 123 165 133 Z" fill="#ff90aa" fillOpacity="0.6" />
        <path d="M 172 135 Q 174 123 172 119 Q 170 122 169 132 Z" fill="#ff90aa" fillOpacity="0.6" />
        {/* 얼굴 */}
        <ellipse cx="168" cy="148" rx="10" ry="10" fill="#ffc8d8" />
        {/* 눈 */}
        <ellipse cx="164" cy="145" rx="2.5" ry="3" fill={DARK} />
        <ellipse cx="172" cy="145" rx="2.5" ry="3" fill={DARK} />
        <circle cx="165" cy="144" r="1" fill="white" />
        <circle cx="173" cy="144" r="1" fill="white" />
        {/* 볼 */}
        <ellipse cx="161" cy="149" rx="3" ry="2" fill="#ff8fa3" fillOpacity="0.5" />
        <ellipse cx="175" cy="149" rx="3" ry="2" fill="#ff8fa3" fillOpacity="0.5" />
        {/* 코 */}
        <ellipse cx="168" cy="150" rx="1.5" ry="1" fill="#ff6080" />
        {/* 입 */}
        <path d="M 165 152 Q 168 154 171 152" fill="none" stroke="#ff6080" strokeWidth="1" strokeLinecap="round" />
        {/* 팔 */}
        <ellipse cx="156" cy="155" rx="5" ry="7" fill="#ffb3c6" transform="rotate(-20,156,155)" />
        <ellipse cx="180" cy="155" rx="5" ry="7" fill="#ffb3c6" transform="rotate(20,180,155)" />
        {/* 발 */}
        <ellipse cx="163" cy="166" rx="6" ry="4" fill="#ffb3c6" />
        <ellipse cx="173" cy="166" rx="6" ry="4" fill="#ffb3c6" />
      </g>

      <style>{`
        @keyframes jinBreathe {
          0%,100% { transform: scaleY(1) translateY(0); }
          50% { transform: scaleY(1.012) translateY(-2px); }
        }
        @keyframes jinHeadBob {
          0%,100% { transform: translateY(0) rotate(0deg); }
          30% { transform: translateY(-5px) rotate(0.8deg); }
          70% { transform: translateY(2px) rotate(-0.5deg); }
        }
        @keyframes jinHeadTilt {
          to { transform: rotate(-10deg) translateX(-6px); }
        }
        @keyframes jinArmIdleL {
          0%,100% { transform: rotate(0deg); }
          50% { transform: rotate(2deg); }
        }
        @keyframes jinArmIdleR {
          0%,100% { transform: rotate(0deg); }
          50% { transform: rotate(-2deg); }
        }
        @keyframes jinArmSpeakL {
          0%,100% { transform: rotate(-3deg) translateY(0); }
          50% { transform: rotate(12deg) translateY(-10px); }
        }
        @keyframes jinArmSpeakR {
          0%,100% { transform: rotate(3deg) translateY(0); }
          50% { transform: rotate(-10deg) translateY(-8px); }
        }
        @keyframes jinArmListen {
          to { transform: rotate(-8deg); }
        }
        @keyframes jinBunny {
          0%,100% { transform: rotate(0deg) translateY(0); }
          50% { transform: rotate(5deg) translateY(-3px); }
        }
      `}</style>
    </svg>
  )
}
