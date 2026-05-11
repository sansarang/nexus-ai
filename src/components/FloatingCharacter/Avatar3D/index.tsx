/**
 * Avatar3D — Photorealistic 3D Avatar Renderer
 * 2026 standards: PBR + Environment IBL + RTX-level shaders
 *
 * GLB URL 있을 때 → Ready Player Me / Mixamo 아바타 렌더링
 * GLB URL 없을 때 → PBR ProceduralHumanoid (피부 SSS, 각막 유리, 머리 silk sheen)
 *
 * 핵심 업그레이드:
 *  - Environment IBL (City/Studio preset)
 *  - ACESFilmic tone mapping
 *  - 투명 배경 (Tauri overlay 지원)
 *  - 고품질 directional + rim 조명
 */
import React from 'react'
import { Canvas } from '@react-three/fiber'
import { ContactShadows, OrbitControls, Environment } from '@react-three/drei'
import * as THREE from 'three'
import type { AvatarEmotion, CharacterPreset } from './ProceduralHumanoid'
import { AvatarModel } from './AvatarModel'
import { ProceduralHumanoid, KPOP_PRESETS } from './ProceduralHumanoid'

export interface Avatar3DProps {
  glbUrl?: string | null
  emotion: AvatarEmotion
  speaking: boolean
  listening: boolean
  primaryColor: string
  accentColor: string
  preset?: CharacterPreset
  scale?: number
  preview?: boolean
  width?: number | string
  height?: number | string
  quality?: 'high' | 'balanced'
  cameraY?: number
  characterOffsetY?: number
}

export function Avatar3D({
  glbUrl,
  emotion,
  speaking,
  listening,
  primaryColor,
  accentColor,
  preset,
  scale = 1,
  preview = false,
  width = 200,
  height = 340,
  quality = 'high',
  cameraY,
  characterOffsetY = -0.3,
}: Avatar3DProps) {

  const resolvedCameraY = cameraY ?? 0.8
  const cameraPos: [number, number, number] = preview
    ? [0, resolvedCameraY, 1.8]
    : [0, resolvedCameraY, 2.0]
  const effectiveScale = preview ? scale * 1.4 : scale
  const shadowSize = quality === 'high' ? 1024 : 512

  const pColor = new THREE.Color(primaryColor)
  const aColor = new THREE.Color(accentColor)

  return (
    <div style={{ width, height, position: 'relative', overflow: 'visible' }}>
      <Canvas
        style={{ background: 'transparent', width: '100%', height: '100%' }}
        camera={{
          position: cameraPos,
          fov: preview ? 50 : 40,
          near: 0.01,
          far: 100,
        }}
        gl={{
          alpha: true,
          antialias: true,
          toneMapping: THREE.ACESFilmicToneMapping,
          toneMappingExposure: 1.15,
          outputColorSpace: THREE.SRGBColorSpace,
        }}
        shadows
        dpr={quality === 'high' ? [1, 2] : [1, 1.5]}
      >
        {/* ── IBL 환경광 (PBR 핵심) ── */}
        <Environment
          preset="city"
          environmentIntensity={0.55}
          backgroundBlurriness={0}
          background={false}
        />

        {/* ── 주 조명 (키 라이트) ── */}
        <directionalLight
          position={[1.2, 2.8, 2.5]}
          intensity={1.6}
          castShadow
          shadow-mapSize-width={shadowSize}
          shadow-mapSize-height={shadowSize}
          shadow-camera-near={0.5}
          shadow-camera-far={10}
          shadow-bias={-0.0004}
        />

        {/* ── 보조 조명 (필 라이트 — 왼쪽) ── */}
        <directionalLight
          position={[-1.8, 1.5, 1.0]}
          intensity={0.55}
          color={new THREE.Color('#cce0ff')}
        />

        {/* ── 림 라이트 (캐릭터 색상 기반) ── */}
        <pointLight
          position={[-1.2, 1.8, -1.0]}
          intensity={0.9}
          color={pColor}
          distance={4}
        />

        {/* ── 어깨 하이라이트 (accent 색) ── */}
        <pointLight
          position={[0, 1.5, -0.8]}
          intensity={0.45}
          color={aColor}
          distance={3}
        />

        {/* ── 전면 소프트 필 ── */}
        <directionalLight
          position={[0, 0.5, 3.5]}
          intensity={0.3}
          color={new THREE.Color('#fff5e8')}
        />

        {/* ── 앰비언트 (약하게) ── */}
        <ambientLight intensity={0.22} />

        {/* ── 아바타 렌더링 ── */}
        <group
          scale={[effectiveScale, effectiveScale, effectiveScale]}
          position={[0, characterOffsetY, 0]}
          rotation={[0, 0, 0]}
        >
          {glbUrl ? (
            <AvatarModel
              url={glbUrl}
              emotion={emotion}
              speaking={speaking}
              listening={listening}
              primaryColor={primaryColor}
              accentColor={accentColor}
            />
          ) : (
            <ProceduralHumanoid
              emotion={emotion}
              speaking={speaking}
              listening={listening}
              primaryColor={primaryColor}
              accentColor={accentColor}
              preset={preset}
            />
          )}
        </group>

        {/* ── 발밑 그림자 ── */}
        <ContactShadows
          position={[0, -1.35, 0]}
          opacity={quality === 'high' ? 0.35 : 0.22}
          scale={2.2}
          blur={3}
          far={2}
          color={primaryColor}
        />

        {/* ── 미리보기 모드 회전 컨트롤 ── */}
        {preview && (
          <OrbitControls
            enableZoom={false}
            enablePan={false}
            minPolarAngle={Math.PI / 4}
            maxPolarAngle={Math.PI * 0.66}
            autoRotate
            autoRotateSpeed={1.8}
          />
        )}
      </Canvas>

      {/* 발광 그라디언트 배경 */}
      <div style={{
        position: 'absolute',
        bottom: 0, left: '50%',
        transform: 'translateX(-50%)',
        width: '65%', height: '28%',
        background: `radial-gradient(ellipse at 50% 100%, ${primaryColor}28 0%, transparent 70%)`,
        pointerEvents: 'none',
        zIndex: -1,
      }} />
    </div>
  )
}

/** 말하기 음파 overlay */
export function SpeakingWaves({ color, active }: { color: string; active: boolean }) {
  if (!active) return null
  return (
    <div style={{
      position: 'absolute', bottom: 12, left: '50%',
      transform: 'translateX(-50%)',
      display: 'flex', gap: 3, alignItems: 'flex-end',
    }}>
      {[0.4, 0.7, 1.0, 0.7, 0.4].map((h, i) => (
        <div
          key={i}
          style={{
            width: 3, borderRadius: 2,
            background: color,
            animation: `wave3d ${0.5 + i * 0.1}s ease-in-out infinite alternate`,
            height: 4 + h * 12,
          }}
        />
      ))}
      <style>{`
        @keyframes wave3d {
          from { transform: scaleY(0.3); opacity: 0.4; }
          to   { transform: scaleY(1);   opacity: 1; }
        }
      `}</style>
    </div>
  )
}

export { ProceduralHumanoid, KPOP_PRESETS }
export type { AvatarEmotion, CharacterPreset }
