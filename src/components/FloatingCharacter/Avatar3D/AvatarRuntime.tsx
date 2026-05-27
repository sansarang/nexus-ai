import React from 'react'
import type { CharacterPreset } from './index'

export type AvatarRuntimeState = 'idle' | 'listening' | 'speaking'
export type AvatarQuality = 'high' | 'balanced'

interface AvatarRuntimeProps {
  glbUrl?: string | null
  runtimeState: AvatarRuntimeState
  emotion?: string
  primaryColor: string
  accentColor: string
  preset?: CharacterPreset
  preview?: boolean
  width?: number | string
  height?: number | string
  quality?: AvatarQuality
  scale?: number
  characterOffsetY?: number
  cameraY?: number
}

function SiriOrb({
  speaking, listening, color, accentColor, width = 200, height = 340, preview = false,
}: {
  speaking: boolean; listening: boolean; color: string; accentColor: string
  width?: number | string; height?: number | string; preview?: boolean
}) {
  const orbSize = preview ? 54 : 110
  const state = speaking ? 'speaking' : listening ? 'listening' : 'idle'
  const uid = `siri-${color.replace('#', '')}`

  return (
    <div style={{
      width, height,
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      position: 'relative', flexShrink: 0,
    }}>
      <style>{`
        @keyframes ${uid}-pulse {
          0%,100%{transform:scale(.95);box-shadow:0 0 ${orbSize*.28}px ${color}88,0 0 ${orbSize*.55}px ${color}44}
          50%{transform:scale(1.06);box-shadow:0 0 ${orbSize*.45}px ${color}bb,0 0 ${orbSize*.75}px ${color}66}
        }
        @keyframes ${uid}-listen {
          0%,100%{transform:scale(.9);box-shadow:0 0 ${orbSize*.35}px ${color}cc,0 0 ${orbSize*.7}px ${color}77}
          50%{transform:scale(1.1);box-shadow:0 0 ${orbSize*.6}px ${color}ff,0 0 ${orbSize*.95}px ${color}99}
        }
        @keyframes ${uid}-speak {
          0%{transform:scale(1.0) scaleX(1.0)}
          20%{transform:scale(1.07) scaleX(.95)}
          50%{transform:scale(.95) scaleX(1.06)}
          80%{transform:scale(1.05) scaleX(.97)}
          100%{transform:scale(1.0) scaleX(1.0)}
        }
        @keyframes ${uid}-ring {
          0%{transform:scale(1);opacity:.55}
          100%{transform:scale(2.4);opacity:0}
        }
      `}</style>

      {(listening || speaking) && [0, 1, 2].map(i => (
        <div key={i} style={{
          position: 'absolute',
          width: orbSize, height: orbSize,
          borderRadius: '50%',
          border: `1.5px solid ${color}`,
          animation: `${uid}-ring ${speaking ? .9 : 1.4}s ease-out ${i * (speaking ? .3 : .47)}s infinite`,
          pointerEvents: 'none',
        }} />
      ))}

      <div style={{
        width: orbSize, height: orbSize,
        borderRadius: '50%',
        background: `radial-gradient(circle at 38% 32%, ${accentColor}ee, ${color}cc 40%, ${color}88 68%, ${color}44)`,
        boxShadow: `0 0 ${orbSize*.28}px ${color}88, 0 0 ${orbSize*.55}px ${color}44, inset 0 2px 8px rgba(255,255,255,.18)`,
        animation:
          state === 'idle'      ? `${uid}-pulse 3.2s ease-in-out infinite` :
          state === 'listening' ? `${uid}-listen 1.1s ease-in-out infinite` :
                                  `${uid}-speak .38s ease-in-out infinite`,
        position: 'relative', zIndex: 2, flexShrink: 0,
      }} />
    </div>
  )
}

export function AvatarRuntime({
  runtimeState,
  primaryColor,
  accentColor,
  preview = false,
  width = 200,
  height = 340,
}: AvatarRuntimeProps) {
  return (
    <SiriOrb
      speaking={runtimeState === 'speaking'}
      listening={runtimeState === 'listening'}
      color={primaryColor}
      accentColor={accentColor}
      width={width}
      height={height}
      preview={preview}
    />
  )
}
