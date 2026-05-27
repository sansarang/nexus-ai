import React from 'react'
import { Avatar3D } from './index'
import type { AvatarEmotion, CharacterPreset } from './index'

export type AvatarRuntimeState = 'idle' | 'listening' | 'speaking'
export type AvatarQuality = 'high' | 'balanced'

interface AvatarRuntimeProps {
  glbUrl?: string | null
  runtimeState: AvatarRuntimeState
  emotion: AvatarEmotion
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

export function AvatarRuntime({
  glbUrl,
  runtimeState,
  emotion,
  primaryColor,
  accentColor,
  preset,
  preview = false,
  width = 200,
  height = 340,
  quality = 'high',
  scale,
  characterOffsetY,
  cameraY,
}: AvatarRuntimeProps) {
  const speaking = runtimeState === 'speaking'
  const listening = runtimeState === 'listening'

  return (
    <Avatar3D
      glbUrl={glbUrl ?? undefined}
      emotion={emotion}
      speaking={speaking}
      listening={listening}
      primaryColor={primaryColor}
      accentColor={accentColor}
      preset={preset}
      preview={preview}
      width={width}
      height={height}
      quality={quality}
      scale={scale}
      characterOffsetY={characterOffsetY}
      cameraY={cameraY}
    />
  )
}
