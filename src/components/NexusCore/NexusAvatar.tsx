import type { NexusEmotion } from '../../types/nexus'

const EMOTION_COLORS: Record<NexusEmotion, string> = {
  neutral:   '#4f7ef7',
  concerned: '#f59e0b',
  happy:     '#22c55e',
  alert:     '#ef4444',
  humorous:  '#a78bfa',
}

const EMOTION_CHARS: Record<NexusEmotion, string> = {
  neutral:   '◉',
  concerned: '◎',
  happy:     '★',
  alert:     '◈',
  humorous:  '◐',
}

const EMOTION_ROTATE_SPEED: Record<NexusEmotion, string> = {
  neutral:   '8s',
  concerned: '4s',
  happy:     '3s',
  alert:     '1.5s',
  humorous:  '5s',
}

export function NexusAvatar({
  emotion,
  speaking,
  listening,
}: {
  emotion: NexusEmotion
  speaking: boolean
  listening: boolean
}) {
  const color = EMOTION_COLORS[emotion] ?? EMOTION_COLORS.neutral
  const char  = EMOTION_CHARS[emotion]  ?? EMOTION_CHARS.neutral
  const speed = EMOTION_ROTATE_SPEED[emotion] ?? '8s'

  const isAlert   = emotion === 'alert'
  const isHappy   = emotion === 'happy'
  const isConcerned = emotion === 'concerned'

  return (
    <div style={{ position: 'relative', width: 72, height: 72, flexShrink: 0 }}>
      <svg
        width={72}
        height={72}
        viewBox="0 0 72 72"
        style={{
          transform: speaking ? 'scale(1.08)' : isAlert ? 'scale(1.04)' : 'scale(1)',
          transition: 'transform 0.3s ease, filter 0.3s ease',
          filter: isAlert
            ? `drop-shadow(0 0 8px ${color}99)`
            : isHappy
            ? `drop-shadow(0 0 6px ${color}66)`
            : 'none',
        }}
      >
        <defs>
          <radialGradient id="coreGrad" cx="50%" cy="50%" r="50%">
            <stop offset="0%" stopColor={color} stopOpacity={0.7} />
            <stop offset="100%" stopColor={color} stopOpacity={0.1} />
          </radialGradient>
        </defs>

        {/* 외부 글로우 링 — 말하거나 듣거나 경보일 때 밝아짐 */}
        <circle
          cx={36} cy={36} r={34}
          fill="none"
          stroke={color}
          strokeWidth={1.5}
          strokeOpacity={speaking || listening || isAlert ? 0.9 : 0.2}
          style={{ transition: 'stroke-opacity 0.3s ease' }}
        />

        {/* 두 번째 글로우 링 — happy/alert 때만 */}
        {(isHappy || isAlert) && (
          <circle
            cx={36} cy={36} r={32}
            fill="none"
            stroke={color}
            strokeWidth={0.8}
            strokeOpacity={0.3}
            style={{ animation: `nexus-pulse-ring 1.5s ease-in-out infinite` }}
          />
        )}

        {/* 회전 점선 링 — 감정별 속도 다름 */}
        <circle
          cx={36} cy={36} r={30}
          fill="none"
          stroke={color}
          strokeWidth={isConcerned ? 2 : 1}
          strokeDasharray={isConcerned ? '4 6' : '6 4'}
          strokeOpacity={0.45}
          style={{
            transformOrigin: '36px 36px',
            animation: `nexus-rotate ${speed} linear infinite`,
          }}
        />

        {/* 코어 */}
        <circle cx={36} cy={36} r={24} fill="url(#coreGrad)" />
        <circle cx={36} cy={36} r={24} fill="none" stroke={color} strokeWidth={1.5} strokeOpacity={0.65} />

        {/* 중앙 캐릭터 */}
        <text
          x={36} y={41}
          textAnchor="middle"
          fontSize={isHappy ? 20 : 18}
          fill={color}
          style={{
            userSelect: 'none',
            animation: isHappy ? 'nexus-bounce 0.8s ease-in-out infinite alternate' : undefined,
          }}
        >
          {char}
        </text>
      </svg>

      {/* 말하는 중 파동 */}
      {speaking && (
        <div style={{
          position: 'absolute', inset: -6,
          borderRadius: '50%',
          border: `2px solid ${color}`,
          animation: 'nexus-speak-ring 1s ease-out infinite',
          pointerEvents: 'none',
        }} />
      )}

      {/* 듣는 중 마이크 배지 */}
      {listening && (
        <div style={{
          position: 'absolute', bottom: 2, right: 2,
          width: 18, height: 18, borderRadius: '50%',
          background: '#ef4444',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          fontSize: 10,
          animation: 'nexus-pulse 1s ease-in-out infinite',
          boxShadow: '0 0 8px #ef4444aa',
        }}>🎤</div>
      )}

      <style>{`
        @keyframes nexus-rotate {
          from { transform: rotate(0deg); }
          to   { transform: rotate(360deg); }
        }
        @keyframes nexus-pulse {
          0%,100% { opacity:1; transform:scale(1); }
          50%      { opacity:0.6; transform:scale(1.25); }
        }
        @keyframes nexus-pulse-ring {
          0%,100% { stroke-opacity:0.3; r:32; }
          50%      { stroke-opacity:0.7; r:34; }
        }
        @keyframes nexus-bounce {
          from { transform: translateY(0px); }
          to   { transform: translateY(-2px); }
        }
        @keyframes nexus-speak-ring {
          0%   { transform:scale(1); opacity:0.7; }
          100% { transform:scale(1.4); opacity:0; }
        }
      `}</style>
    </div>
  )
}
