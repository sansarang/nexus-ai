import type { CharacterProps } from './types'

/**
 * 루미 — 전통 현대 퓨전 엘리건트 여성
 * 참고: 2번 이미지 "루미" (다크 헤어, 전통 패턴 의상, 우아한 분위기)
 */
export function Lumi({ emotion, speaking, listening }: CharacterProps) {
  const SKIN_L  = '#fceee2'
  const SKIN_M  = '#f2d4b8'
  const SKIN_S  = '#d8a882'
  const HAIR    = '#1c1428'   // 다크 플럼 블랙
  const HAIR_H  = '#3c2c50'   // 헤어 하이라이트
  const HAIR_SH = '#0d0918'   // 헤어 딥 섀도우
  const HANBOK1 = '#2d1a42'   // 다크 퍼플 (치마)
  const HANBOK2 = '#1a0e2e'   // 딥 퍼플
  const HANBOK3 = '#4a2468'   // 미드 퍼플
  const PATTERN = '#7a3a9a'   // 패턴 컬러
  const PATTERN2= '#c060e8'   // 밝은 패턴
  const COLLAR  = '#f8f0ff'   // 흰 깃
  const WAIST   = '#6a1a8a'   // 허리띠
  const EYE_C   = '#4a2060'   // 퍼플 눈
  const DARK    = '#0e0918'
  const LIP     = '#c02850'

  const eyeY  = emotion === 'happy' ? 92 : 89
  const eyeRy = emotion === 'happy' ? 5 : emotion === 'alert' ? 8 : 6.5

  const mouthD = speaking
    ? 'M 93 136 Q 108 150 123 136'
    : emotion === 'happy'    ? 'M 90 134 Q 108 148 126 134'
    : emotion === 'concerned'? 'M 95 140 Q 108 134 121 140'
    : emotion === 'humorous' ? 'M 90 134 Q 108 146 126 134 Q 120 152 96 152 Z'
    :                          'M 92 135 Q 108 145 124 135'

  return (
    <svg viewBox="0 0 220 500" width="220" height="500" style={{ overflow: 'visible' }}>
      <defs>
        <radialGradient id="lumiSkinFace" cx="50%" cy="36%" r="60%">
          <stop offset="0%"   stopColor="#fffaf5" />
          <stop offset="40%"  stopColor={SKIN_L} />
          <stop offset="100%" stopColor={SKIN_M} />
        </radialGradient>
        <radialGradient id="lumiSkinArm" cx="35%" cy="28%" r="72%">
          <stop offset="0%"   stopColor={SKIN_L} />
          <stop offset="100%" stopColor={SKIN_M} />
        </radialGradient>
        <linearGradient id="lumiHair" x1="10%" y1="0%" x2="90%" y2="100%">
          <stop offset="0%"   stopColor={HAIR_H} />
          <stop offset="40%"  stopColor={HAIR} />
          <stop offset="100%" stopColor={HAIR_SH} />
        </linearGradient>
        <linearGradient id="lumiHanbok" x1="0%" y1="0%" x2="60%" y2="100%">
          <stop offset="0%"   stopColor={HANBOK3} />
          <stop offset="50%"  stopColor={HANBOK1} />
          <stop offset="100%" stopColor={HANBOK2} />
        </linearGradient>
        <linearGradient id="lumiHanbokR" x1="100%" y1="0%" x2="40%" y2="100%">
          <stop offset="0%"   stopColor={HANBOK3} />
          <stop offset="100%" stopColor={HANBOK2} />
        </linearGradient>
        <radialGradient id="lumiIris" cx="40%" cy="35%" r="58%">
          <stop offset="0%"   stopColor="#9060c0" />
          <stop offset="55%"  stopColor={EYE_C} />
          <stop offset="100%" stopColor="#1a0830" />
        </radialGradient>
        <pattern id="lumiPattern" x="0" y="0" width="20" height="20" patternUnits="userSpaceOnUse">
          <circle cx="10" cy="10" r="3" fill={PATTERN} fillOpacity="0.25" />
          <path d="M 5 5 Q 10 2 15 5" fill="none" stroke={PATTERN2} strokeWidth="0.8" strokeOpacity="0.3" />
        </pattern>
        <filter id="lumiGlow">
          <feGaussianBlur stdDeviation="2.5" result="b"/>
          <feMerge><feMergeNode in="b"/><feMergeNode in="SourceGraphic"/></feMerge>
        </filter>
        <radialGradient id="lumiAura" cx="50%" cy="20%" r="80%">
          <stop offset="0%"   stopColor="#9060c0" stopOpacity="0.2" />
          <stop offset="100%" stopColor="#9060c0" stopOpacity="0" />
        </radialGradient>
      </defs>

      {/* 퍼플 오라 */}
      {(emotion === 'happy' || emotion === 'humorous') && (
        <ellipse cx="110" cy="240" rx="100" ry="210" fill="url(#lumiAura)" />
      )}

      {/* ═══ 치마 (풍성한 한복 스타일) ═══ */}
      <g>
        {/* 치마 메인 */}
        <path d="M 56 265 Q 40 320 36 430 Q 68 445 108 442 L 110 340 L 112 442 Q 152 445 184 430 Q 180 320 164 265 Z"
          fill="url(#lumiHanbok)" />
        {/* 치마 패턴 오버레이 */}
        <path d="M 56 265 Q 40 320 36 430 Q 68 445 108 442 L 110 340 L 112 442 Q 152 445 184 430 Q 180 320 164 265 Z"
          fill="url(#lumiPattern)" />
        {/* 치마 주름 */}
        {[70, 85, 100, 110, 120, 135, 150].map(x => (
          <line key={x} x1={x} y1={270}
            x2={x + (x < 110 ? -5 : 5)} y2={440}
            stroke={HANBOK2} strokeWidth="0.8" strokeOpacity="0.35" />
        ))}
        {/* 치마 빛 반사 */}
        <path d="M 64 270 Q 88 280 108 275 Q 88 295 64 285 Z"
          fill={PATTERN2} fillOpacity="0.08" />
        {/* 치마 아랫단 */}
        <path d="M 36 430 Q 110 445 184 430 Q 184 440 110 448 Q 36 440 36 430 Z"
          fill={HANBOK2} />
        {/* 치마단 수 장식 */}
        {[50, 70, 90, 110, 130, 150, 170].map(x => (
          <path key={x} d={`M ${x} 434 Q ${x+3} 440 ${x+6} 434`}
            fill="none" stroke={PATTERN2} strokeWidth="1.2" strokeOpacity="0.5" />
        ))}
        {/* 신발 */}
        <path d="M 78 440 Q 68 450 80 456 Q 95 460 100 452 L 96 448 Q 92 455 82 452 Q 74 448 78 440 Z"
          fill={HANBOK2} stroke={PATTERN} strokeWidth="1" />
        <path d="M 142 452 L 140 448 Q 136 455 118 452 Q 110 448 114 440 Q 108 450 120 456 Q 135 460 142 452 Z"
          fill={HANBOK2} stroke={PATTERN} strokeWidth="1" />
      </g>

      {/* ═══ 상의 (한복 저고리 스타일) ═══ */}
      <g style={{ animation: 'lumiBreathe 3.8s ease-in-out infinite', transformOrigin: '110px 220px' }}>
        {/* 저고리 뒤판 */}
        <path d="M 60 155 Q 48 190 50 265 L 170 265 Q 172 190 160 155 Z"
          fill="url(#lumiHanbok)" />
        {/* 저고리 앞 왼쪽 */}
        <path d="M 60 155 Q 52 190 54 265 L 110 265 L 110 155 Z"
          fill={HANBOK1} />
        {/* 저고리 앞 오른쪽 */}
        <path d="M 110 155 L 110 265 L 166 265 Q 168 190 160 155 Z"
          fill={HANBOK2} />
        {/* 옷감 패턴 */}
        <path d="M 60 155 Q 52 190 54 265 L 110 265 L 110 155 Z"
          fill="url(#lumiPattern)" />
        {/* 깃 (흰색 collar) */}
        <path d="M 72 160 Q 80 148 110 145 Q 140 148 148 160 Q 130 155 110 158 Q 90 155 72 160 Z"
          fill={COLLAR} />
        {/* 깃 오른쪽 */}
        <path d="M 72 160 Q 80 155 110 158 L 110 174 Q 90 172 72 165 Z"
          fill={COLLAR} fillOpacity="0.9" />
        <path d="M 148 160 Q 140 155 110 158 L 110 174 Q 130 172 148 165 Z"
          fill={COLLAR} fillOpacity="0.85" />
        {/* 고름 (넥타이 리본) */}
        <path d="M 106 168 Q 98 175 90 180 Q 86 185 90 188 Q 96 186 102 178 L 106 192"
          fill="none" stroke="#e040b0" strokeWidth="3" strokeLinecap="round" />
        <path d="M 114 168 Q 118 175 122 180 Q 128 186 124 190 Q 118 188 115 182 L 114 192"
          fill="none" stroke="#e040b0" strokeWidth="3" strokeLinecap="round" />
        <ellipse cx="110" cy="168" rx="6" ry="4" fill="#e040b0" />
        {/* 허리띠 */}
        <path d="M 52 240 Q 110 248 168 240 Q 110 256 52 240 Z" fill={WAIST} />
        <path d="M 52 240 Q 110 244 168 240" fill="none" stroke={PATTERN2} strokeWidth="2" strokeOpacity="0.5" />
        {/* 옷 주름 선 */}
        <path d="M 80 175 Q 78 220 80 255" fill="none" stroke={PATTERN} strokeWidth="0.8" strokeOpacity="0.3" />
        <path d="M 140 175 Q 142 220 140 255" fill="none" stroke={PATTERN} strokeWidth="0.8" strokeOpacity="0.3" />
        {/* 패턴 장식 (꽃) */}
        {[[65, 200], [155, 195], [68, 230], [152, 232]].map(([x,y]) => (
          <g key={`${x},${y}`}>
            <circle cx={x} cy={y} r="4" fill={PATTERN2} fillOpacity="0.2" />
            {[0,60,120,180,240,300].map(ang => {
              const r2 = ang * Math.PI / 180
              return <circle key={ang} cx={x + Math.cos(r2)*5} cy={y + Math.sin(r2)*5} r="1.5"
                fill={PATTERN2} fillOpacity="0.2" />
            })}
          </g>
        ))}
      </g>

      {/* ═══ 왼팔 (소매) ═══ */}
      <g style={{
        transformOrigin: '55px 155px',
        animation: speaking
          ? 'lumiArmSpeakL 1.1s ease-in-out infinite'
          : listening
          ? 'lumiArmListen 0.5s ease forwards'
          : 'lumiArmIdleL 5s ease-in-out infinite',
      }}>
        <path d="M 60 155 Q 38 178 28 238 Q 26 274 38 280 Q 50 284 58 272 Q 46 238 48 198 Q 52 174 62 158 Z"
          fill={HANBOK1} />
        {/* 소매 패턴 */}
        <path d="M 60 155 Q 38 178 28 238 Q 26 274 38 280 Q 50 284 58 272 Q 46 238 48 198 Q 52 174 62 158 Z"
          fill="url(#lumiPattern)" />
        {/* 소매 끝단 */}
        <path d="M 26 266 Q 26 285 50 278" fill="none" stroke={PATTERN2} strokeWidth="2" strokeOpacity="0.6" />
        <path d="M 26 270 Q 50 280 52 272"
          fill={COLLAR} fillOpacity="0.8" stroke={COLLAR} strokeWidth="1" />
        <ellipse cx="34" cy="288" rx="13" ry="11" fill="url(#lumiSkinArm)" />
        <path d="M 23 285 Q 21 278 25 275" stroke={SKIN_M} strokeWidth="2.5" fill="none" strokeLinecap="round" />
        <path d="M 29 283 Q 27 276 31 274" stroke={SKIN_M} strokeWidth="2.5" fill="none" strokeLinecap="round" />
        <path d="M 35 282 Q 34 275 37 274" stroke={SKIN_M} strokeWidth="2.5" fill="none" strokeLinecap="round" />
        <path d="M 41 283 Q 41 276 44 276" stroke={SKIN_M} strokeWidth="2.5" fill="none" strokeLinecap="round" />
        <ellipse cx="32" cy="290" rx="8" ry="4.5" fill={SKIN_S} fillOpacity="0.22" />
      </g>

      {/* ═══ 오른팔 (소매) ═══ */}
      <g style={{
        transformOrigin: '165px 155px',
        animation: speaking
          ? 'lumiArmSpeakR 1.1s ease-in-out infinite 0.4s'
          : 'lumiArmIdleR 5s ease-in-out infinite 2.2s',
      }}>
        <path d="M 160 155 Q 182 178 192 238 Q 194 274 182 280 Q 170 284 162 272 Q 174 238 172 198 Q 168 174 158 158 Z"
          fill={HANBOK2} />
        <path d="M 160 155 Q 182 178 192 238 Q 194 274 182 280 Q 170 284 162 272 Q 174 238 172 198 Q 168 174 158 158 Z"
          fill="url(#lumiPattern)" />
        <path d="M 194 266 Q 194 285 170 278" fill="none" stroke={PATTERN2} strokeWidth="2" strokeOpacity="0.6" />
        <path d="M 194 270 Q 170 280 168 272"
          fill={COLLAR} fillOpacity="0.8" stroke={COLLAR} strokeWidth="1" />
        <ellipse cx="186" cy="288" rx="13" ry="11" fill="url(#lumiSkinArm)" />
        <path d="M 197 285 Q 199 278 195 275" stroke={SKIN_M} strokeWidth="2.5" fill="none" strokeLinecap="round" />
        <path d="M 191 283 Q 193 276 189 274" stroke={SKIN_M} strokeWidth="2.5" fill="none" strokeLinecap="round" />
        <path d="M 185 282 Q 186 275 183 274" stroke={SKIN_M} strokeWidth="2.5" fill="none" strokeLinecap="round" />
        <path d="M 179 283 Q 179 276 176 276" stroke={SKIN_M} strokeWidth="2.5" fill="none" strokeLinecap="round" />
        <ellipse cx="188" cy="290" rx="8" ry="4.5" fill={SKIN_S} fillOpacity="0.22" />
      </g>

      {/* ═══ 목 ═══ */}
      <path d="M 96 155 Q 94 176 100 178 L 120 178 Q 126 176 124 155 Z"
        fill="url(#lumiSkinFace)" />
      <path d="M 104 158 L 102 176" stroke={SKIN_S} strokeWidth="1.5" strokeOpacity="0.2" />
      <path d="M 116 158 L 118 176" stroke={SKIN_S} strokeWidth="1.5" strokeOpacity="0.2" />
      {/* 옥 목걸이 */}
      <path d="M 78 166 Q 94 158 110 160 Q 126 158 142 166"
        fill="none" stroke="#80c080" strokeWidth="2" />
      <ellipse cx="110" cy="162" rx="5" ry="4" fill="#a0e0a0" stroke="#40a040" strokeWidth="1" />
      <ellipse cx="97" cy="163" rx="3" ry="2.5" fill="#90d890" stroke="#40a040" strokeWidth="0.8" />
      <ellipse cx="123" cy="163" rx="3" ry="2.5" fill="#90d890" stroke="#40a040" strokeWidth="0.8" />

      {/* ═══ 머리 ═══ */}
      <g style={{
        transformOrigin: '110px 88px',
        animation: listening
          ? 'lumiHeadTilt 0.5s ease forwards'
          : 'lumiHeadBob 6s ease-in-out infinite',
      }}>
        {/* 뒷 머리 업스타일 */}
        <ellipse cx="110" cy="76" rx="54" ry="58" fill="url(#lumiHair)" />
        {/* 업스타일 번 */}
        <ellipse cx="110" cy="36" rx="24" ry="20" fill="url(#lumiHair)" />
        <path d="M 88 40 Q 110 28 132 40 Q 128 32 110 28 Q 92 32 88 40 Z"
          fill={HAIR_H} fillOpacity="0.3" />
        {/* 번 비녀 */}
        <line x1="92" y1="36" x2="128" y2="36" stroke={PATTERN2} strokeWidth="2.5" strokeLinecap="round" />
        <circle cx="92" cy="36" r="4" fill={PATTERN2} />
        <circle cx="128" cy="36" r="4" fill={PATTERN2} />
        {/* 비녀 꽃 장식 */}
        {[0,60,120,180,240,300].map(ang => {
          const r = ang * Math.PI / 180
          return <circle key={ang} cx={92 + Math.cos(r)*5} cy={36 + Math.sin(r)*5} r="2"
            fill={PATTERN2} fillOpacity="0.6" />
        })}
        {/* 번 위쪽 꽃핀 */}
        <circle cx="110" cy="24" r="6" fill={PATTERN2} />
        <path d="M 106 24 L 110 18 L 114 24 L 110 28 Z" fill={PATTERN} />

        {/* 옆 앞머리 (우아하게 내린) */}
        <path d="M 56 82 Q 42 104 44 135 Q 48 155 56 162 Q 60 145 58 125 Q 56 106 60 90 Z"
          fill="url(#lumiHair)" />
        <path d="M 164 82 Q 178 104 176 135 Q 172 155 164 162 Q 160 145 162 125 Q 164 106 160 90 Z"
          fill="url(#lumiHair)" />
        {/* 앞머리 */}
        <path d="M 58 72 Q 66 36 88 28 Q 80 40 76 58 Q 72 70 66 80 Z"
          fill={HAIR} />
        <path d="M 162 72 Q 154 36 132 28 Q 140 40 144 58 Q 148 70 154 80 Z"
          fill={HAIR} />

        {/* 얼굴 */}
        <path d="M 58 72 Q 56 102 60 124 Q 64 148 78 162 Q 94 172 110 172 Q 126 172 142 162 Q 156 148 160 124 Q 164 102 162 72 Q 156 40 130 28 Q 110 22 90 28 Q 62 40 58 72 Z"
          fill="url(#lumiSkinFace)" />
        {/* 측면 음영 */}
        <path d="M 58 70 Q 56 102 60 124 Q 64 148 76 162 Q 68 148 64 125 Q 60 104 62 75 Z"
          fill={SKIN_S} fillOpacity="0.18" />
        <path d="M 162 70 Q 164 102 160 124 Q 156 148 144 162 Q 152 148 156 125 Q 160 104 158 75 Z"
          fill={SKIN_S} fillOpacity="0.18" />

        {/* 헤어 하이라이트 */}
        <path d="M 82 28 Q 100 22 118 26"
          fill="none" stroke={HAIR_H} strokeWidth="3.5" strokeOpacity="0.35" strokeLinecap="round" />

        {/* 귀 */}
        <path d="M 58 92 Q 52 97 52 106 Q 52 116 58 120 Q 62 122 64 118 Q 60 114 60 106 Q 60 98 64 94 Z"
          fill={SKIN_M} />
        <path d="M 162 92 Q 168 97 168 106 Q 168 116 162 120 Q 158 122 156 118 Q 160 114 160 106 Q 160 98 156 94 Z"
          fill={SKIN_M} />
        {/* 진주 귀걸이 */}
        <circle cx="54" cy="120" r="5" fill="white" stroke="#ccc" strokeWidth="0.8" />
        <ellipse cx="55" cy="119" rx="2.5" ry="1.5" fill="white" fillOpacity="0.7" />
        <circle cx="166" cy="120" r="5" fill="white" stroke="#ccc" strokeWidth="0.8" />
        <ellipse cx="167" cy="119" rx="2.5" ry="1.5" fill="white" fillOpacity="0.7" />

        {/* ─ 눈썹 (우아하고 아치형) ─ */}
        <path
          d={emotion === 'concerned'
            ? 'M 74 78 Q 87 75 100 80'
            : emotion === 'happy'
            ? 'M 74 74 Q 87 68 100 73'
            : 'M 74 76 Q 87 70 100 76'}
          fill="none" stroke={HAIR} strokeWidth="3.5" strokeLinecap="round" />
        <path
          d={emotion === 'concerned'
            ? 'M 120 80 Q 133 75 146 78'
            : emotion === 'happy'
            ? 'M 120 73 Q 133 68 146 74'
            : 'M 120 76 Q 133 70 146 76'}
          fill="none" stroke={HAIR} strokeWidth="3.5" strokeLinecap="round" />

        {/* ─ 눈 ─ */}
        <ellipse cx="87" cy={eyeY - 3} rx="14" ry="6" fill={SKIN_S} fillOpacity="0.1" />
        <ellipse cx="133" cy={eyeY - 3} rx="14" ry="6" fill={SKIN_S} fillOpacity="0.1" />
        <path d={`M 74 ${eyeY} Q 87 ${eyeY - eyeRy - 2} 100 ${eyeY} Q 87 ${eyeY + eyeRy - 1} 74 ${eyeY} Z`}
          fill="white" />
        <path d={`M 120 ${eyeY} Q 133 ${eyeY - eyeRy - 2} 146 ${eyeY} Q 133 ${eyeY + eyeRy - 1} 120 ${eyeY} Z`}
          fill="white" />
        <ellipse cx="87" cy={eyeY} rx="9" ry={eyeRy} fill="url(#lumiIris)" />
        <ellipse cx="133" cy={eyeY} rx="9" ry={eyeRy} fill="url(#lumiIris)" />
        <circle cx="87" cy={eyeY + 1} r="5" fill={DARK} />
        <circle cx="133" cy={eyeY + 1} r="5" fill={DARK} />
        <ellipse cx="89" cy={eyeY - 2.5} rx="3" ry="2.2" fill="white" fillOpacity="0.9" />
        <circle cx="84" cy={eyeY + 2} r="1.2" fill="white" fillOpacity="0.5" />
        <ellipse cx="135" cy={eyeY - 2.5} rx="3" ry="2.2" fill="white" fillOpacity="0.9" />
        <circle cx="130" cy={eyeY + 2} r="1.2" fill="white" fillOpacity="0.5" />
        {/* 아이라인 */}
        <path d={`M 72 ${eyeY - 1} Q 87 ${eyeY - eyeRy - 3} 102 ${eyeY - 1}`}
          fill="none" stroke={DARK} strokeWidth="2.5" strokeLinecap="round" />
        <path d={`M 118 ${eyeY - 1} Q 133 ${eyeY - eyeRy - 3} 148 ${eyeY - 1}`}
          fill="none" stroke={DARK} strokeWidth="2.5" strokeLinecap="round" />
        {/* 속눈썹 */}
        {[-4,-2,0,2,4].map(dx => (
          <line key={dx} x1={87+dx} y1={eyeY-eyeRy-1} x2={87+dx*0.9} y2={eyeY-eyeRy-6}
            stroke={DARK} strokeWidth="1.8" strokeLinecap="round" />
        ))}
        {[-4,-2,0,2,4].map(dx => (
          <line key={dx} x1={133+dx} y1={eyeY-eyeRy-1} x2={133+dx*0.9} y2={eyeY-eyeRy-6}
            stroke={DARK} strokeWidth="1.8" strokeLinecap="round" />
        ))}
        {/* 아이섀도우 */}
        <ellipse cx="87" cy={eyeY - eyeRy + 1} rx="11" ry="2.5" fill="#8040a0" fillOpacity="0.2" />
        <ellipse cx="133" cy={eyeY - eyeRy + 1} rx="11" ry="2.5" fill="#8040a0" fillOpacity="0.2" />

        {/* ─ 코 ─ */}
        <path d="M 106 110 Q 104 118 106 126 Q 110 130 114 126 Q 116 118 114 110"
          fill="none" stroke={SKIN_S} strokeWidth="1.4" strokeOpacity="0.4" strokeLinecap="round" />
        <path d="M 103 124 Q 110 128 117 124"
          fill="none" stroke={SKIN_S} strokeWidth="1.4" strokeOpacity="0.35" />

        {/* ─ 입 ─ */}
        <path d="M 92 133 Q 102 127 110 131 Q 118 127 128 133 Q 120 137 110 138 Q 100 137 92 133 Z"
          fill={LIP} />
        {!speaking ? (
          <path d="M 92 133 Q 100 144 110 146 Q 120 144 128 133 Q 120 140 110 141 Q 100 140 92 133 Z"
            fill="#b02048" fillOpacity="0.85" />
        ) : (
          <>
            <path d={mouthD} fill="rgba(50,10,20,0.7)" />
            <rect x="104" y="133" width="12" height="8" rx="2" fill="white" fillOpacity="0.85" />
          </>
        )}
        <ellipse cx="104" cy="131" rx="7" ry="2.5" fill="white" fillOpacity="0.18" transform="rotate(-5,104,131)" />
        <path d="M 92 133 Q 110 136 128 133" fill="none" stroke="#800030" strokeWidth="0.7" strokeOpacity="0.35" />

        {/* 볼 홍조 */}
        <ellipse cx="72" cy="118" rx="13" ry="8" fill="#ff8898" fillOpacity="0.1" />
        <ellipse cx="148" cy="118" rx="13" ry="8" fill="#ff8898" fillOpacity="0.1" />

        {/* 페이스 하이라이트 */}
        <ellipse cx="70" cy="110" rx="14" ry="8" fill="white" fillOpacity="0.07" transform="rotate(-10,70,110)" />
        <ellipse cx="150" cy="110" rx="14" ry="8" fill="white" fillOpacity="0.07" transform="rotate(10,150,110)" />

        {/* 턱 음영 */}
        <path d="M 82 158 Q 110 168 138 158 Q 110 175 82 158 Z"
          fill={SKIN_S} fillOpacity="0.14" />
      </g>

      <style>{`
        @keyframes lumiBreathe {
          0%,100% { transform: scaleY(1) translateY(0); }
          50% { transform: scaleY(1.01) translateY(-2px); }
        }
        @keyframes lumiHeadBob {
          0%,100% { transform: translateY(0) rotate(0deg); }
          30% { transform: translateY(-5px) rotate(0.6deg); }
          70% { transform: translateY(2px) rotate(-0.4deg); }
        }
        @keyframes lumiHeadTilt {
          to { transform: rotate(-9deg) translateX(-5px); }
        }
        @keyframes lumiArmIdleL {
          0%,100% { transform: rotate(0deg); }
          50% { transform: rotate(2deg); }
        }
        @keyframes lumiArmIdleR {
          0%,100% { transform: rotate(0deg); }
          50% { transform: rotate(-2deg); }
        }
        @keyframes lumiArmSpeakL {
          0%,100% { transform: rotate(-3deg) translateY(0); }
          50% { transform: rotate(10deg) translateY(-10px); }
        }
        @keyframes lumiArmSpeakR {
          0%,100% { transform: rotate(3deg) translateY(0); }
          50% { transform: rotate(-8deg) translateY(-8px); }
        }
        @keyframes lumiArmListen {
          to { transform: rotate(-8deg); }
        }
      `}</style>
    </svg>
  )
}
