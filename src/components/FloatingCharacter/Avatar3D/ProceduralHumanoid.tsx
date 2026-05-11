/**
 * ProceduralHumanoid v3 — Photorealistic Human Avatar
 * 2026 standards: PBR materials, SSS skin, organic geometry
 *
 * MeshPhysicalMaterial 기반:
 *  - 피부: SSS 근사 (transmission + clearcoat)
 *  - 눈:   각막 transmission (유리 질감)
 *  - 머리: sheen (실크 광택)
 *  - 입술: clearcoat (촉촉한 광택)
 *
 * 스타일 4종:
 *  kpop_star          — K-pop 여성 센터
 *  expert_professional — 전문가 남성
 *  natural_human      — 자연형 여성
 *  creator_streamer   — 크리에이터 남성
 */
import React, { useRef, useMemo, useEffect } from 'react'
import { useFrame } from '@react-three/fiber'
import * as THREE from 'three'
import { REALISTIC_STYLE_PRESETS, styleToPreset } from './Presets'
import type { RealisticStyleId } from './Presets'

export type AvatarEmotion = 'neutral' | 'happy' | 'thinking' | 'surprised' | 'concerned'
export type CharacterPreset = RealisticStyleId

// 하위호환
export { REALISTIC_STYLE_PRESETS as KPOP_PRESETS }

interface ProceduralHumanoidProps {
  emotion: AvatarEmotion
  speaking: boolean
  listening: boolean
  primaryColor: string
  accentColor: string
  preset?: CharacterPreset
}

export function ProceduralHumanoid({
  emotion,
  speaking,
  listening,
  primaryColor,
  accentColor,
  preset = 'kpop_star',
}: ProceduralHumanoidProps) {
  const cfg = styleToPreset(preset)

  const rootRef  = useRef<THREE.Group>(null)
  const headRef  = useRef<THREE.Group>(null)
  const neckRef  = useRef<THREE.Mesh>(null)
  const bodyRef  = useRef<THREE.Group>(null)
  const lArmRef  = useRef<THREE.Group>(null)
  const rArmRef  = useRef<THREE.Group>(null)
  const jawRef   = useRef<THREE.Mesh>(null)
  const eyeLRef  = useRef<THREE.Mesh>(null)
  const eyeRRef  = useRef<THREE.Mesh>(null)
  const blinkRef  = useRef(0)
  const blinkOpen = useRef(true)
  const mouthAmp  = useRef(0)

  /* ── PBR 재질 팩토리 ─────────────────────────── */

  // 피부 — SSS 근사 (MeshPhysicalMaterial)
  const skin = useMemo(() => new THREE.MeshPhysicalMaterial({
    color: new THREE.Color(cfg.skinTone),
    roughness: 0.52,
    metalness: 0,
    transmission: 0.04,   // 약한 반투명 → SSS 느낌
    thickness:    0.8,
    ior:          1.38,
    clearcoat:    0.18,
    clearcoatRoughness: 0.85,
    sheen:        0.1,
    sheenColor:   new THREE.Color(cfg.skinTone).multiplyScalar(1.2),
  }), [cfg.skinTone])

  // 머리카락 — silk sheen
  const hair = useMemo(() => new THREE.MeshPhysicalMaterial({
    color: new THREE.Color(cfg.hairColor),
    roughness: 0.62,
    metalness: 0.08,
    sheen:        0.75,
    sheenColor:   new THREE.Color('#ffffff').lerp(new THREE.Color(cfg.hairColor), 0.5),
    sheenRoughness: 0.28,
  }), [cfg.hairColor])

  // 눈 흰자
  const eyeWhite = useMemo(() => new THREE.MeshPhysicalMaterial({
    color: '#f8f8f5',
    roughness: 0.18,
    metalness: 0,
    clearcoat: 0.4,
    clearcoatRoughness: 0.1,
  }), [])

  // 홍채
  const iris = useMemo(() => new THREE.MeshPhysicalMaterial({
    color: new THREE.Color(cfg.eyeColor),
    roughness: 0.1,
    metalness: 0.2,
    clearcoat: 0.9,
    clearcoatRoughness: 0.05,
  }), [cfg.eyeColor])

  // 각막 — 유리 질감
  const cornea = useMemo(() => new THREE.MeshPhysicalMaterial({
    color: '#ffffff',
    transmission: 0.96,
    roughness: 0,
    metalness: 0,
    ior: 1.45,
    thickness: 0.04,
    transparent: true,
    opacity: 0.92,
    clearcoat: 1,
    clearcoatRoughness: 0,
  }), [])

  // 동공
  const pupil = useMemo(() => new THREE.MeshStandardMaterial({
    color: '#050508',
    roughness: 0.05,
    metalness: 0.5,
    emissive: new THREE.Color('#000000'),
  }), [])

  // 입술 — 촉촉한 광택
  const lip = useMemo(() => new THREE.MeshPhysicalMaterial({
    color: new THREE.Color(cfg.lipColor),
    roughness: 0.22,
    metalness: 0,
    clearcoat: 0.65,
    clearcoatRoughness: 0.15,
    sheen: 0.3,
    sheenColor: new THREE.Color(cfg.lipColor).multiplyScalar(1.4),
  }), [cfg.lipColor])

  // 눈썹
  const brow = useMemo(() => new THREE.MeshStandardMaterial({
    color: new THREE.Color(cfg.hairColor).multiplyScalar(1.1),
    roughness: 0.9,
  }), [cfg.hairColor])

  // 블러셔
  const blushMat = useMemo(() => new THREE.MeshStandardMaterial({
    color: '#ffaab4',
    roughness: 1,
    transparent: true,
    opacity: 0.28,
  }), [])

  // 의상
  const outfit = useMemo(() => new THREE.MeshPhysicalMaterial({
    color: new THREE.Color(primaryColor),
    roughness: 0.55,
    metalness: 0.1,
    sheen: 0.15,
    sheenColor: new THREE.Color(accentColor),
    sheenRoughness: 0.6,
  }), [primaryColor, accentColor])

  // 악세사리
  const accent = useMemo(() => new THREE.MeshPhysicalMaterial({
    color: new THREE.Color(accentColor),
    roughness: 0.3,
    metalness: 0.45,
    clearcoat: 0.6,
    clearcoatRoughness: 0.2,
  }), [accentColor])

  // 하의 / 신발
  const darkOutfit = useMemo(() => new THREE.MeshPhysicalMaterial({
    color: new THREE.Color(primaryColor).multiplyScalar(0.3),
    roughness: 0.68,
    metalness: 0.05,
  }), [primaryColor])

  // 코 끝 — 매 렌더 Color 생성 방지용 메모
  const noseTipMat = useMemo(() => new THREE.MeshPhysicalMaterial({
    color: new THREE.Color(cfg.skinTone).multiplyScalar(0.86),
    roughness: 0.6,
    clearcoat: 0.1,
  }), [cfg.skinTone])

  // ── 언마운트 시 모든 Material 해제 ───────────
  useEffect(() => {
    return () => {
      skin.dispose()
      hair.dispose()
      eyeWhite.dispose()
      iris.dispose()
      cornea.dispose()
      pupil.dispose()
      lip.dispose()
      brow.dispose()
      blushMat.dispose()
      outfit.dispose()
      accent.dispose()
      darkOutfit.dispose()
      noseTipMat.dispose()
    }
  }, [skin, hair, eyeWhite, iris, cornea, pupil, lip, brow, blushMat, outfit, accent, darkOutfit, noseTipMat])

  /* ── 애니메이션 ─────────────────────────────── */
  useFrame((_, delta) => {
    const t = performance.now() / 1000

    // 전신 부유
    if (rootRef.current) {
      rootRef.current.position.y = Math.sin(t * 1.05) * 0.022
    }

    // 상체 호흡
    if (bodyRef.current) {
      bodyRef.current.scale.y = 1 + Math.sin(t * 1.7) * 0.009
      bodyRef.current.scale.x = 1 - Math.sin(t * 1.7) * 0.003
    }

    // 머리 움직임
    if (headRef.current) {
      if (listening) {
        headRef.current.rotation.z = THREE.MathUtils.lerp(headRef.current.rotation.z, 0.095, 0.055)
        headRef.current.rotation.y = THREE.MathUtils.lerp(headRef.current.rotation.y, Math.sin(t * 0.45) * 0.04, 0.04)
        headRef.current.rotation.x = THREE.MathUtils.lerp(headRef.current.rotation.x, 0.02, 0.04)
      } else if (speaking) {
        headRef.current.rotation.x = THREE.MathUtils.lerp(headRef.current.rotation.x, Math.sin(t * 2.8) * 0.038, 0.1)
        headRef.current.rotation.y = THREE.MathUtils.lerp(headRef.current.rotation.y, Math.sin(t * 1.4) * 0.065, 0.06)
        headRef.current.rotation.z = THREE.MathUtils.lerp(headRef.current.rotation.z, Math.sin(t * 2.0) * 0.02, 0.06)
      } else {
        const targetZ = emotion === 'thinking'  ? 0.1
          : emotion === 'happy'     ? Math.sin(t * 1.9) * 0.035
          : Math.sin(t * 0.55) * 0.01
        const targetX = emotion === 'surprised' ? -0.11
          : emotion === 'concerned' ? 0.04
          : Math.sin(t * 0.45) * 0.008
        headRef.current.rotation.z = THREE.MathUtils.lerp(headRef.current.rotation.z, targetZ, 0.04)
        headRef.current.rotation.x = THREE.MathUtils.lerp(headRef.current.rotation.x, targetX, 0.04)
        headRef.current.rotation.y = THREE.MathUtils.lerp(headRef.current.rotation.y, Math.sin(t * 0.35) * 0.015, 0.025)
      }
    }

    // 팔 자연스러운 흔들기
    if (lArmRef.current) {
      const baseZ = cfg.isFeminine ? 0.24 : 0.2
      lArmRef.current.rotation.z = speaking
        ? baseZ + 0.18 + Math.sin(t * 2.8) * 0.2
        : baseZ + Math.sin(t * 1.05) * 0.04
      lArmRef.current.rotation.x = speaking
        ? Math.sin(t * 1.5) * 0.06
        : Math.sin(t * 0.8) * 0.015
    }
    if (rArmRef.current) {
      const baseZ = cfg.isFeminine ? -0.24 : -0.2
      rArmRef.current.rotation.z = speaking
        ? baseZ - 0.18 - Math.sin(t * 2.8 + 1.1) * 0.2
        : baseZ - Math.sin(t * 1.25 + 0.5) * 0.04
      rArmRef.current.rotation.x = speaking
        ? Math.sin(t * 1.5 + 0.5) * 0.06
        : Math.sin(t * 0.9 + 0.3) * 0.015
    }

    // 입 (말하기 — 부드러운 viseme 근사)
    mouthAmp.current = THREE.MathUtils.lerp(
      mouthAmp.current,
      speaking ? Math.abs(Math.sin(t * 9.5)) * 0.024 : 0,
      0.25,
    )
    if (jawRef.current) {
      jawRef.current.position.y = -0.038 - mouthAmp.current
      jawRef.current.scale.x   = 1 + mouthAmp.current * 8
    }

    // 눈 깜빡임
    blinkRef.current += delta
    if (blinkOpen.current && blinkRef.current > 3.2 + Math.random() * 2.5) {
      blinkOpen.current = false; blinkRef.current = 0
    } else if (!blinkOpen.current && blinkRef.current > 0.08) {
      blinkOpen.current = true; blinkRef.current = 0
    }
    const eyeScaleY = blinkOpen.current ? 1 : 0.06
    if (eyeLRef.current) eyeLRef.current.scale.y = THREE.MathUtils.lerp(eyeLRef.current.scale.y, eyeScaleY, 0.42)
    if (eyeRRef.current) eyeRRef.current.scale.y = THREE.MathUtils.lerp(eyeRRef.current.scale.y, eyeScaleY, 0.42)
  })

  const fem = cfg.isFeminine

  return (
    <group ref={rootRef} position={[0, -0.55, 0]}>

      {/* ── 하체 (바닥부터) ── */}
      <group position={[0, 0.18, 0]}>
        {/* 골반 */}
        <mesh castShadow>
          <cylinderGeometry args={fem ? [0.14, 0.16, 0.15, 20] : [0.18, 0.20, 0.15, 20]} />
          <primitive object={darkOutfit} />
        </mesh>
        {/* 왼 다리 */}
        <group position={fem ? [-0.082, -0.35, 0] : [-0.105, -0.35, 0]}>
          <mesh castShadow>
            <capsuleGeometry args={[fem ? 0.056 : 0.068, 0.38, 8, 16]} />
            <primitive object={darkOutfit} />
          </mesh>
          {/* 왼 종아리 */}
          <mesh position={[0, -0.28, 0.01]} castShadow>
            <capsuleGeometry args={[fem ? 0.044 : 0.055, 0.28, 6, 12]} />
            <primitive object={skin} />
          </mesh>
          {/* 왼 발 */}
          <mesh position={[0, -0.47, 0.03]} scale={[1, 0.42, 1.6]} castShadow>
            <sphereGeometry args={[fem ? 0.058 : 0.072, 14, 10]} />
            <primitive object={darkOutfit} />
          </mesh>
        </group>
        {/* 오른 다리 */}
        <group position={fem ? [0.082, -0.35, 0] : [0.105, -0.35, 0]}>
          <mesh castShadow>
            <capsuleGeometry args={[fem ? 0.056 : 0.068, 0.38, 8, 16]} />
            <primitive object={darkOutfit} />
          </mesh>
          <mesh position={[0, -0.28, 0.01]} castShadow>
            <capsuleGeometry args={[fem ? 0.044 : 0.055, 0.28, 6, 12]} />
            <primitive object={skin} />
          </mesh>
          <mesh position={[0, -0.47, 0.03]} scale={[1, 0.42, 1.6]} castShadow>
            <sphereGeometry args={[fem ? 0.058 : 0.072, 14, 10]} />
            <primitive object={darkOutfit} />
          </mesh>
        </group>
      </group>

      {/* ── 상체 ── */}
      <group ref={bodyRef} position={[0, 0.52, 0]}>
        {/* 몸통 (LatheGeometry 느낌 — 스케일 변형 캡슐) */}
        <mesh castShadow scale={fem ? [1, 1, 0.8] : [1, 1, 0.88]}>
          <capsuleGeometry args={fem ? [0.155, 0.3, 12, 24] : [0.2, 0.32, 12, 24]} />
          <primitive object={outfit} />
        </mesh>
        {/* 가슴 포켓 / 브레이드 디테일 */}
        <mesh position={[fem ? 0.1 : 0.1, 0.15, fem ? 0.16 : 0.2]} scale={[0.5, 0.4, 0.2]}>
          <boxGeometry args={[0.1, 0.08, 0.04]} />
          <primitive object={accent} />
        </mesh>
        {/* 칼라 */}
        <mesh position={[0, 0.24, fem ? 0.12 : 0.16]} scale={[1, 0.4, 0.4]}>
          <cylinderGeometry args={[fem ? 0.07 : 0.09, fem ? 0.1 : 0.13, 0.1, 16]} />
          <primitive object={accent} />
        </mesh>
        {/* 목 */}
        <mesh ref={neckRef} position={[0, 0.31, 0]} castShadow>
          <cylinderGeometry args={fem ? [0.044, 0.056, 0.13, 18] : [0.058, 0.07, 0.12, 18]} />
          <primitive object={skin} />
        </mesh>
      </group>

      {/* ── 왼팔 ── */}
      <group ref={lArmRef} position={fem ? [-0.2, 0.7, 0] : [-0.25, 0.72, 0]}>
        {/* 위팔 */}
        <mesh position={[0, -0.14, 0]} castShadow>
          <capsuleGeometry args={fem ? [0.044, 0.22, 6, 12] : [0.056, 0.24, 6, 12]} />
          <primitive object={outfit} />
        </mesh>
        {/* 아래팔 */}
        <mesh position={[0, -0.32, 0.01]} castShadow>
          <capsuleGeometry args={fem ? [0.036, 0.2, 6, 12] : [0.046, 0.22, 6, 12]} />
          <primitive object={skin} />
        </mesh>
        {/* 손 */}
        <mesh position={[0, -0.46, 0.01]} scale={fem ? [0.9, 0.78, 0.65] : [1, 0.82, 0.7]} castShadow>
          <sphereGeometry args={fem ? [0.052, 14, 12] : [0.062, 14, 12]} />
          <primitive object={skin} />
        </mesh>
        {/* 손가락 힌트 */}
        {[0, 1, 2].map(i => (
          <mesh key={i} position={[(i - 1) * 0.018, -0.52, 0.01]} castShadow>
            <capsuleGeometry args={[0.008, 0.025, 4, 8]} />
            <primitive object={skin} />
          </mesh>
        ))}
      </group>

      {/* ── 오른팔 ── */}
      <group ref={rArmRef} position={fem ? [0.2, 0.7, 0] : [0.25, 0.72, 0]}>
        <mesh position={[0, -0.14, 0]} castShadow>
          <capsuleGeometry args={fem ? [0.044, 0.22, 6, 12] : [0.056, 0.24, 6, 12]} />
          <primitive object={outfit} />
        </mesh>
        <mesh position={[0, -0.32, 0.01]} castShadow>
          <capsuleGeometry args={fem ? [0.036, 0.2, 6, 12] : [0.046, 0.22, 6, 12]} />
          <primitive object={skin} />
        </mesh>
        <mesh position={[0, -0.46, 0.01]} scale={fem ? [0.9, 0.78, 0.65] : [1, 0.82, 0.7]} castShadow>
          <sphereGeometry args={fem ? [0.052, 14, 12] : [0.062, 14, 12]} />
          <primitive object={skin} />
        </mesh>
        {[0, 1, 2].map(i => (
          <mesh key={i} position={[(i - 1) * 0.018, -0.52, 0.01]} castShadow>
            <capsuleGeometry args={[0.008, 0.025, 4, 8]} />
            <primitive object={skin} />
          </mesh>
        ))}
      </group>

      {/* ── 머리 ── */}
      <group ref={headRef} position={[0, 1.12, 0]}>

        {/* 두개골 (세로로 긴 타원) */}
        <mesh castShadow scale={fem ? [0.96, 1.12, 0.92] : [1, 1.08, 0.96]}>
          <sphereGeometry args={[0.152, 36, 32]} />
          <primitive object={skin} />
        </mesh>

        {/* 광대뼈 볼 (좌) */}
        <mesh position={[-0.1, -0.04, 0.1]} scale={[0.55, 0.45, 0.42]}>
          <sphereGeometry args={[0.1, 16, 14]} />
          <primitive object={skin} />
        </mesh>
        {/* 광대뼈 볼 (우) */}
        <mesh position={[0.1, -0.04, 0.1]} scale={[0.55, 0.45, 0.42]}>
          <sphereGeometry args={[0.1, 16, 14]} />
          <primitive object={skin} />
        </mesh>

        {/* 턱선 */}
        <mesh position={[0, -0.115, 0.025]} scale={fem ? [0.75, 0.52, 0.82] : [0.88, 0.56, 0.9]}>
          <sphereGeometry args={[0.1, 18, 16]} />
          <primitive object={skin} />
        </mesh>

        {/* ── 눈썹 ── */}
        {[[-0.058, 0.072], [0.058, 0.072]].map(([x, y], si) => (
          <mesh
            key={si}
            position={[x, y, 0.146]}
            rotation={[0, 0, si === 0
              ? (emotion === 'concerned' ? 0.22 : fem ? 0.1 : 0.15)
              : (emotion === 'concerned' ? -0.22 : fem ? -0.1 : -0.15)
            ]}
          >
            <capsuleGeometry args={[0.004, fem ? 0.042 : 0.048, 4, 8]} />
            <primitive object={brow} />
          </mesh>
        ))}

        {/* ── 눈 (좌우 공통 컴포넌트) ── */}
        {[[-0.057, 0.022], [0.057, 0.022]].map(([ex, ey], ei) => (
          <group key={ei} position={[ex, ey, 0]}>
            {/* 눈 소켓 (살짝 들어간 느낌) */}
            <mesh position={[0, 0, 0.128]} scale={[1.3, fem ? 0.68 : 0.60, 0.55]}>
              <sphereGeometry args={[0.038, 18, 16]} />
              <primitive object={eyeWhite} />
            </mesh>
            {/* 홍채 */}
            <mesh
              ref={ei === 0 ? eyeLRef : eyeRRef}
              position={[0, 0, 0.148]}
              scale={[1, fem ? 0.68 : 0.60, 0.55]}
            >
              <sphereGeometry args={[0.024, 16, 14]} />
              <primitive object={iris} />
            </mesh>
            {/* 동공 */}
            <mesh position={[0, 0, 0.155]} scale={[0.48, 0.48, 0.38]}>
              <sphereGeometry args={[0.024, 12, 10]} />
              <primitive object={pupil} />
            </mesh>
            {/* 각막 (유리 오버레이) */}
            <mesh position={[0, 0, 0.150]} scale={[1.18, fem ? 0.72 : 0.64, 0.52]}>
              <sphereGeometry args={[0.034, 14, 12]} />
              <primitive object={cornea} />
            </mesh>
            {/* 하이라이트 반짝임 */}
            <mesh position={[0.008, 0.01, 0.158]}>
              <sphereGeometry args={[0.005, 6, 6]} />
              <meshStandardMaterial color="#ffffff" emissive="#ffffff" emissiveIntensity={2.5} />
            </mesh>
            {/* 아이라이너 (여성) */}
            {fem && (
              <mesh position={[0, -0.01, 0.149]} scale={[1.5, 0.18, 0.38]}>
                <sphereGeometry args={[0.034, 12, 8]} />
                <meshStandardMaterial color="#0a0a12" roughness={0.4} />
              </mesh>
            )}
          </group>
        ))}

        {/* ── 코 ── */}
        {/* 코 다리 */}
        <mesh position={[0, 0.004, 0.15]} scale={[0.38, 1.25, 0.48]}>
          <sphereGeometry args={[0.022, 12, 10]} />
          <primitive object={skin} />
        </mesh>
        {/* 코 끝 */}
        <mesh position={[0, -0.02, 0.157]} scale={[1, 0.88, 0.9]}>
          <sphereGeometry args={[0.016, 10, 10]} />
          <primitive object={noseTipMat} />
        </mesh>
        {/* 콧망울 (좌우) */}
        {[[-0.018, 0], [0.018, 0]].map(([nx], ni) => (
          <mesh key={ni} position={[nx, -0.022, 0.154]} scale={[0.55, 0.5, 0.5]}>
            <sphereGeometry args={[0.018, 10, 10]} />
            <primitive object={skin} />
          </mesh>
        ))}

        {/* ── 입술 ── */}
        {/* 윗입술 */}
        <mesh position={[0, -0.056, 0.148]} scale={fem ? [1.35, 1.05, 1] : [1.25, 0.88, 1]}>
          <sphereGeometry args={[0.02, 14, 12]} />
          <primitive object={lip} />
        </mesh>
        {/* 큐피드 활 */}
        <mesh position={[-0.014, -0.05, 0.15]} scale={[0.55, 0.55, 0.7]}>
          <sphereGeometry args={[0.014, 10, 8]} />
          <primitive object={lip} />
        </mesh>
        <mesh position={[0.014, -0.05, 0.15]} scale={[0.55, 0.55, 0.7]}>
          <sphereGeometry args={[0.014, 10, 8]} />
          <primitive object={lip} />
        </mesh>
        {/* 아랫입술 (jawRef — 말하기) */}
        <mesh ref={jawRef} position={[0, -0.037, 0.152]} scale={fem ? [1.42, 0.92, 1] : [1.3, 0.88, 1]}>
          <sphereGeometry args={[0.022, 14, 12]} />
          <primitive object={lip} />
        </mesh>

        {/* ── 귀 ── */}
        {[[-0.157, 0.008], [0.157, 0.008]].map(([ex, ey], ei) => (
          <group key={ei} position={[ex, ey, 0]}>
            <mesh scale={[0.32, 0.6, 0.28]}>
              <sphereGeometry args={[0.04, 12, 10]} />
              <primitive object={skin} />
            </mesh>
            <mesh position={[ei === 0 ? -0.004 : 0.004, 0, 0.008]} scale={[0.18, 0.35, 0.18]}>
              <sphereGeometry args={[0.04, 10, 8]} />
              <primitive object={skin} />
            </mesh>
          </group>
        ))}

        {/* ── 블러셔 (여성/K-pop) ── */}
        {cfg.blush && (
          <>
            <mesh position={[-0.088, -0.018, 0.126]} scale={[1.25, 0.58, 0.35]}>
              <sphereGeometry args={[0.032, 10, 8]} />
              <primitive object={blushMat} />
            </mesh>
            <mesh position={[0.088, -0.018, 0.126]} scale={[1.25, 0.58, 0.35]}>
              <sphereGeometry args={[0.032, 10, 8]} />
              <primitive object={blushMat} />
            </mesh>
          </>
        )}

        {/* ── 헤어스타일 ── */}
        <HairMesh style={cfg.hairStyle} hairMat={hair} fem={fem} />

        {/* 이마 하이라이트 (skin 재질 재사용) */}
        <mesh position={[0, 0.14, 0.13]} scale={[0.7, 0.25, 0.3]}>
          <sphereGeometry args={[0.06, 10, 8]} />
          <primitive object={skin} />
        </mesh>
      </group>

      {/* ── 감정 이펙트 ── */}
      {emotion === 'happy' && <SparkEffect color={accentColor} />}
      {emotion === 'surprised' && <AuraRing color="#ffd166" scale={1.1} />}
      {listening && <AuraRing color={primaryColor} scale={0.95} />}
    </group>
  )
}

/* ── 헤어스타일 컴포넌트 ─────────────────────── */
function HairMesh({ style, hairMat, fem }: {
  style: string
  hairMat: THREE.Material
  fem: boolean
}) {
  switch (style) {
    case 'long_wavy':
      return (
        <group>
          {/* 정수리 볼륨 */}
          <mesh position={[0, 0.135, -0.008]} scale={[1.02, 1, 1]}>
            <sphereGeometry args={[0.162, 28, 24, 0, Math.PI * 2, 0, Math.PI * 0.54]} />
            <primitive object={hairMat} />
          </mesh>
          {/* 앞머리 */}
          <mesh position={[-0.03, 0.095, 0.105]} rotation={[0.1, 0.1, -0.15]} scale={[1.15, 0.55, 0.5]}>
            <sphereGeometry args={[0.11, 16, 14]} />
            <primitive object={hairMat} />
          </mesh>
          {/* 웨이브 옆머리 L */}
          <mesh position={[-0.11, -0.02, 0.065]} scale={[0.45, 2.1, 0.42]}>
            <sphereGeometry args={[0.1, 14, 12]} />
            <primitive object={hairMat} />
          </mesh>
          <mesh position={[-0.13, -0.18, 0.025]} rotation={[0.1, 0, 0.08]} scale={[0.4, 1.6, 0.38]}>
            <sphereGeometry args={[0.1, 12, 10]} />
            <primitive object={hairMat} />
          </mesh>
          {/* 웨이브 옆머리 R */}
          <mesh position={[0.11, -0.02, 0.065]} scale={[0.45, 2.1, 0.42]}>
            <sphereGeometry args={[0.1, 14, 12]} />
            <primitive object={hairMat} />
          </mesh>
          <mesh position={[0.13, -0.18, 0.025]} rotation={[0.1, 0, -0.08]} scale={[0.4, 1.6, 0.38]}>
            <sphereGeometry args={[0.1, 12, 10]} />
            <primitive object={hairMat} />
          </mesh>
          {/* 긴 뒷머리 */}
          <mesh position={[0, -0.22, -0.095]} scale={[1.05, 2.8, 0.52]}>
            <sphereGeometry args={[0.12, 16, 14]} />
            <primitive object={hairMat} />
          </mesh>
        </group>
      )

    case 'bob_sleek':
      return (
        <group>
          <mesh position={[0, 0.135, -0.006]}>
            <sphereGeometry args={[0.165, 28, 24, 0, Math.PI * 2, 0, Math.PI * 0.53]} />
            <primitive object={hairMat} />
          </mesh>
          {/* 직선 앞머리 */}
          <mesh position={[0, 0.098, 0.112]} scale={[1.3, 0.42, 0.44]}>
            <sphereGeometry args={[0.11, 16, 12]} />
            <primitive object={hairMat} />
          </mesh>
          {/* 단발 좌 */}
          <mesh position={[-0.115, -0.065, 0.01]} scale={[0.48, 1.35, 0.58]}>
            <sphereGeometry args={[0.1, 14, 12]} />
            <primitive object={hairMat} />
          </mesh>
          {/* 단발 우 */}
          <mesh position={[0.115, -0.065, 0.01]} scale={[0.48, 1.35, 0.58]}>
            <sphereGeometry args={[0.1, 14, 12]} />
            <primitive object={hairMat} />
          </mesh>
          {/* 뒷 단발 */}
          <mesh position={[0, -0.038, -0.105]} scale={[1.02, 1.15, 0.5]}>
            <sphereGeometry args={[0.14, 16, 14]} />
            <primitive object={hairMat} />
          </mesh>
        </group>
      )

    case 'straight_long':
      return (
        <group>
          <mesh position={[0, 0.13, -0.008]}>
            <sphereGeometry args={[0.163, 28, 24, 0, Math.PI * 2, 0, Math.PI * 0.52]} />
            <primitive object={hairMat} />
          </mesh>
          <mesh position={[0, 0.09, 0.1]} scale={[1.1, 0.58, 0.48]}>
            <sphereGeometry args={[0.12, 16, 12]} />
            <primitive object={hairMat} />
          </mesh>
          {/* 직선 긴 머리 */}
          <mesh position={[-0.1, -0.08, 0.04]} scale={[0.4, 2.4, 0.38]}>
            <sphereGeometry args={[0.1, 12, 10]} />
            <primitive object={hairMat} />
          </mesh>
          <mesh position={[0.1, -0.08, 0.04]} scale={[0.4, 2.4, 0.38]}>
            <sphereGeometry args={[0.1, 12, 10]} />
            <primitive object={hairMat} />
          </mesh>
          <mesh position={[0, -0.2, -0.1]} scale={[1.02, 2.6, 0.5]}>
            <sphereGeometry args={[0.12, 16, 12]} />
            <primitive object={hairMat} />
          </mesh>
        </group>
      )

    case 'undercut':
      return (
        <group>
          {/* 위쪽 볼륨 */}
          <mesh position={[0, 0.155, -0.008]}>
            <sphereGeometry args={[0.17, 28, 24, 0, Math.PI * 2, 0, Math.PI * 0.47]} />
            <primitive object={hairMat} />
          </mesh>
          {/* 옆으로 쓸어넘긴 앞머리 */}
          <mesh position={[-0.045, 0.112, 0.115]} rotation={[0, 0, -0.28]} scale={[1.55, 0.44, 0.5]}>
            <sphereGeometry args={[0.1, 16, 12]} />
            <primitive object={hairMat} />
          </mesh>
          {/* 언더컷 짧은 옆 */}
          <mesh position={[-0.155, 0.005, 0]} scale={[0.26, 0.78, 0.62]}>
            <sphereGeometry args={[0.08, 12, 10]} />
            <primitive object={hairMat} />
          </mesh>
          <mesh position={[0.155, 0.005, 0]} scale={[0.26, 0.78, 0.62]}>
            <sphereGeometry args={[0.08, 12, 10]} />
            <primitive object={hairMat} />
          </mesh>
          {/* 뒷면 짧게 */}
          <mesh position={[0, 0.02, -0.148]} scale={[1, 0.65, 0.3]}>
            <sphereGeometry args={[0.1, 12, 10]} />
            <primitive object={hairMat} />
          </mesh>
        </group>
      )

    case 'textured_short':
    default:
      return (
        <group>
          {/* 정수리 질감 */}
          <mesh position={[0, 0.148, -0.005]}>
            <sphereGeometry args={[0.168, 28, 24, 0, Math.PI * 2, 0, Math.PI * 0.49]} />
            <primitive object={hairMat} />
          </mesh>
          {/* 거친 앞머리 */}
          <mesh position={[0.02, 0.112, 0.12]} rotation={[0, 0, 0.12]} scale={[1.12, 0.48, 0.5]}>
            <sphereGeometry args={[0.1, 14, 12]} />
            <primitive object={hairMat} />
          </mesh>
          {/* 텍스처 덩어리들 */}
          {[[-0.06, 0.15, 0.08], [0.06, 0.16, 0.07], [0.01, 0.17, 0.04]].map(([tx, ty, tz], ti) => (
            <mesh key={ti} position={[tx, ty, tz]} scale={[0.45, 0.32, 0.38]}>
              <sphereGeometry args={[0.08, 10, 8]} />
              <primitive object={hairMat} />
            </mesh>
          ))}
          <mesh position={[-0.152, 0.01, 0]} scale={[0.28, 0.82, 0.6]}>
            <sphereGeometry args={[0.08, 10, 8]} />
            <primitive object={hairMat} />
          </mesh>
          <mesh position={[0.152, 0.01, 0]} scale={[0.28, 0.82, 0.6]}>
            <sphereGeometry args={[0.08, 10, 8]} />
            <primitive object={hairMat} />
          </mesh>
        </group>
      )
  }
}

/* ── 감정 이펙트 ─────────────────────────────── */
function SparkEffect({ color }: { color: string }) {
  const g = useRef<THREE.Group>(null)
  useFrame(() => { if (g.current) g.current.rotation.y += 0.022 })
  const c = useMemo(() => new THREE.Color(color), [color])
  return (
    <group ref={g}>
      {[0, 1, 2, 3, 4].map(i => {
        const a = (i / 5) * Math.PI * 2
        return (
          <mesh key={i} position={[Math.cos(a) * 0.28, 0.72 + Math.sin(a) * 0.07, Math.sin(a) * 0.28]}>
            <octahedronGeometry args={[0.022]} />
            <meshStandardMaterial color={c} emissive={c} emissiveIntensity={1.5} />
          </mesh>
        )
      })}
    </group>
  )
}

function AuraRing({ color, scale = 1 }: { color: string; scale?: number }) {
  const r = useRef<THREE.Mesh>(null)
  useFrame(() => {
    if (r.current) r.current.scale.setScalar(scale + Math.sin(performance.now() / 380) * 0.065)
  })
  const c = useMemo(() => new THREE.Color(color), [color])
  return (
    <mesh ref={r} position={[0, 0.44, 0]}>
      <torusGeometry args={[0.24, 0.006, 8, 36]} />
      <meshStandardMaterial color={c} emissive={c} emissiveIntensity={0.85} transparent opacity={0.5} />
    </mesh>
  )
}
