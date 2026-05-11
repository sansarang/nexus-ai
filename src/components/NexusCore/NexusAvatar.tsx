import type { NexusEmotion } from '../../types/nexus'

const EMOTION_COLORS: Record<NexusEmotion, string> = {
  neutral: '#4f7ef7',
  concerned: '#f59e0b',
  happy: '#22c55e',
  alert: '#ef4444',
  humorous: '#a78bfa',
}

const EMOTION_CHARS: Record<NexusEmotion, string> = {
  neutral: '◉',
  concerned: '◎',
  happy: '●',
  alert: '◈',
  humorous: '◐',
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
  const color = EMOTION_COLORS[emotion]
  const char = EMOTION_CHARS[emotion]

  return (
    <div style={{ position: 'relative', width: 72, height: 72, flexShrink: 0 }}>
      <svg
        width={72}
        height={72}
        viewBox="0 0 72 72"
        style={{
          transform: speaking ? 'scale(1.05)' : 'scale(1)',
          transition: 'transform 0.3s ease',
        }}
      >
        <defs>
          <radialGradient id="coreGrad" cx="50%" cy="50%" r="50%">
            <stop offset="0%" stopColor={color} stopOpacity={0.6} />
            <stop offset="100%" stopColor={color} stopOpacity={0.15} />
          </radialGradient>
        </defs>

        {/* Outer glow ring */}
        <circle
          cx={36}
          cy={36}
          r={34}
          fill="none"
          stroke={color}
          strokeWidth={1.5}
          strokeOpacity={speaking || listening ? 0.8 : 0.25}
          style={{ transition: 'stroke-opacity 0.3s ease' }}
        />

        {/* Rotating dashed ring */}
        <circle
          cx={36}
          cy={36}
          r={30}
          fill="none"
          stroke={color}
          strokeWidth={1}
          strokeDasharray="6 4"
          strokeOpacity={0.4}
          style={{
            transformOrigin: '36px 36px',
            animation: 'nexus-rotate 8s linear infinite',
          }}
        />

        {/* Core circle */}
        <circle cx={36} cy={36} r={24} fill="url(#coreGrad)" />
        <circle
          cx={36}
          cy={36}
          r={24}
          fill="none"
          stroke={color}
          strokeWidth={1.5}
          strokeOpacity={0.6}
        />

        {/* Central character */}
        <text
          x={36}
          y={41}
          textAnchor="middle"
          fontSize={18}
          fill={color}
          style={{ userSelect: 'none' }}
        >
          {char}
        </text>
      </svg>

      {/* Mic badge */}
      {listening && (
        <div
          style={{
            position: 'absolute',
            bottom: 2,
            right: 2,
            width: 16,
            height: 16,
            borderRadius: '50%',
            background: '#ef4444',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontSize: 9,
            animation: 'nexus-pulse 1s ease-in-out infinite',
          }}
        >
          🎤
        </div>
      )}

      <style>{`
        @keyframes nexus-rotate {
          from { transform: rotate(0deg); }
          to { transform: rotate(360deg); }
        }
        @keyframes nexus-pulse {
          0%, 100% { opacity: 1; transform: scale(1); }
          50% { opacity: 0.6; transform: scale(1.2); }
        }
      `}</style>
    </div>
  )
}
