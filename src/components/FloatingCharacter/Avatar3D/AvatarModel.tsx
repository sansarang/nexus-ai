/**
 * AvatarModel — GLB 파일 로더 + Animation Controller
 * Ready Player Me (RPM) GLB 포맷에 최적화
 * Mixamo 호환 애니메이션도 지원
 *
 * 버그 수정:
 *  - scene null 체크 추가
 *  - viseme 배열 컴포넌트 상단으로 이동 (매 프레임 재생성 방지)
 *  - useEffect cleanup 강화
 */
import React, { useRef, useEffect, useMemo, Suspense } from 'react'
import { useGLTF, useAnimations } from '@react-three/drei'
import { useFrame } from '@react-three/fiber'
import * as THREE from 'three'
import type { AvatarEmotion } from './ProceduralHumanoid'
import { ProceduralHumanoid } from './ProceduralHumanoid'

// 매 프레임 재생성 방지 — 모듈 레벨 상수
const VISEME_NAMES = [
  'viseme_aa', 'viseme_E', 'viseme_I', 'viseme_O',
  'viseme_U', 'viseme_PP', 'viseme_FF', 'viseme_TH',
  'viseme_DD', 'viseme_kk', 'viseme_CH', 'viseme_SS',
  'viseme_nn', 'viseme_RR', 'viseme_sil',
]

interface AvatarModelProps {
  url: string
  emotion: AvatarEmotion
  speaking: boolean
  listening: boolean
  primaryColor: string
  accentColor: string
}

function GLBAvatar({ url, emotion, speaking, listening, primaryColor, accentColor }: AvatarModelProps) {
  const group   = useRef<THREE.Group>(null)
  const jawBone = useRef<THREE.Object3D | null>(null)
  const headBone= useRef<THREE.Object3D | null>(null)
  const spineRef= useRef<THREE.Object3D | null>(null)
  const lShldr  = useRef<THREE.Object3D | null>(null)
  const rShldr  = useRef<THREE.Object3D | null>(null)

  const blinkTimer  = useRef(0)
  const blinkOpen   = useRef(true)
  const speakTimer  = useRef(0)
  const visemeIdx   = useRef(0)

  const { scene, animations } = useGLTF(url)
  const { actions, mixer }    = useAnimations(animations, group)

  // ── 본 캐싱 ──────────────────────────────────
  useEffect(() => {
    if (!scene) return
    // 이전 본 참조 초기화
    jawBone.current  = null
    headBone.current = null
    spineRef.current = null
    lShldr.current   = null
    rShldr.current   = null

    scene.traverse((obj: THREE.Object3D) => {
      const n = obj.name.toLowerCase()
      if (n.includes('jaw') || n.includes('chin'))                                                  jawBone.current  = obj
      else if (n.includes('head') && !headBone.current)                                             headBone.current = obj
      else if ((n.includes('spine') || n.includes('chest')) && !spineRef.current)                   spineRef.current = obj
      else if ((n.includes('leftshoulder') || n.includes('l_shoulder') || n.includes('leftarm')) && !lShldr.current) lShldr.current = obj
      else if ((n.includes('rightshoulder') || n.includes('r_shoulder') || n.includes('rightarm')) && !rShldr.current) rShldr.current = obj
    })

    scene.rotation.y = 0
    scene.position.set(0, 0, 0)
  }, [scene])

  // ── 감정 morph target ──────────────────────────
  useEffect(() => {
    if (!scene) return

    scene.traverse(obj => {
      const mesh = obj as THREE.SkinnedMesh
      if (!mesh.isSkinnedMesh || !mesh.morphTargetDictionary || !mesh.morphTargetInfluences) return

      const set = (name: string, val: number) => {
        const idx = mesh.morphTargetDictionary![name]
        if (idx !== undefined) mesh.morphTargetInfluences![idx] = val
      }

      // 모두 리셋
      Object.keys(mesh.morphTargetDictionary).forEach(k => set(k, 0))

      switch (emotion) {
        case 'happy':
          set('mouthSmileLeft', 0.8); set('mouthSmileRight', 0.8)
          set('cheekSquintLeft', 0.4); set('cheekSquintRight', 0.4)
          set('eyeSquintLeft', 0.3); set('eyeSquintRight', 0.3)
          break
        case 'thinking':
          set('browInnerUp', 0.5); set('browDownLeft', 0.3)
          set('mouthPucker', 0.2)
          break
        case 'surprised':
          set('eyeWideLeft', 0.8); set('eyeWideRight', 0.8)
          set('browInnerUp', 0.7); set('jawOpen', 0.4)
          break
        case 'concerned':
          set('browDownLeft', 0.6); set('browDownRight', 0.6)
          set('mouthFrownLeft', 0.4); set('mouthFrownRight', 0.4)
          break
        default:
          break
      }
    })
  }, [scene, emotion])

  // ── 애니메이션 클립 선택 ──────────────────────
  useEffect(() => {
    if (!actions) return
    const names = Object.keys(actions)
    if (names.length === 0) return

    const find = (...keywords: string[]) =>
      names.find(n => keywords.some(k => n.toLowerCase().includes(k.toLowerCase()))) ?? null

    let clipName: string | null = null
    if (speaking)       clipName = find('talking', 'talk', 'speak') ?? find('idle') ?? names[0]
    else if (listening) clipName = find('listening', 'thinking', 'idle_2') ?? find('idle') ?? names[0]
    else                clipName = find('idle', 'breathing', 'stand') ?? names[0]

    // 현재 재생 중인 것 fadeOut
    Object.values(actions).forEach(a => a?.fadeOut(0.3))

    const clip = clipName ? actions[clipName] : null
    if (clip) clip.reset().fadeIn(0.3).play()

    return () => { clip?.fadeOut(0.2) }
  }, [actions, speaking, listening])

  // ── Per-frame animation ───────────────────────
  useFrame((_, delta) => {
    if (!group.current) return
    const t = performance.now() / 1000

    // 부유
    group.current.position.y = Math.sin(t * 1.1) * 0.025

    // 머리 idle
    if (headBone.current) {
      headBone.current.rotation.z = THREE.MathUtils.lerp(
        headBone.current.rotation.z,
        listening ? 0.1 : Math.sin(t * 0.65) * 0.028,
        0.05,
      )
      headBone.current.rotation.y = THREE.MathUtils.lerp(
        headBone.current.rotation.y,
        Math.sin(t * 0.45) * 0.038,
        0.04,
      )
    }

    // 척추 호흡
    if (spineRef.current) {
      spineRef.current.rotation.x = Math.sin(t * 1.75) * 0.009
    }

    // 턱 말하기
    if (jawBone.current) {
      const target = speaking ? Math.abs(Math.sin(t * 8.5)) * 0.11 : 0
      jawBone.current.rotation.x = THREE.MathUtils.lerp(jawBone.current.rotation.x, target, 0.22)
    }

    // 어깨 제스처
    if (speaking) {
      if (lShldr.current) lShldr.current.rotation.z = Math.sin(t * 2.4) * 0.11
      if (rShldr.current) rShldr.current.rotation.z = -Math.sin(t * 2.4 + 1.0) * 0.11
    } else {
      if (lShldr.current) lShldr.current.rotation.z = THREE.MathUtils.lerp(lShldr.current.rotation.z, 0, 0.08)
      if (rShldr.current) rShldr.current.rotation.z = THREE.MathUtils.lerp(rShldr.current.rotation.z, 0, 0.08)
    }

    // 눈 깜빡임
    blinkTimer.current += delta
    if (blinkOpen.current && blinkTimer.current > 3.5 + Math.random() * 2.5) {
      blinkOpen.current = false
      blinkTimer.current = 0
    } else if (!blinkOpen.current && blinkTimer.current > 0.1) {
      blinkOpen.current = true
      blinkTimer.current = 0
    }
    const blinkVal = blinkOpen.current ? 0 : 1
    if (scene) {
      scene.traverse((obj: THREE.Object3D) => {
        const mesh = obj as THREE.SkinnedMesh
        if (!mesh.morphTargetDictionary || !mesh.morphTargetInfluences) return
        const setM = (k: string, v: number) => {
          const i = mesh.morphTargetDictionary![k]
          if (i !== undefined) mesh.morphTargetInfluences![i] = v
        }
        setM('eyeBlinkLeft', blinkVal)
        setM('eyeBlinkRight', blinkVal)
      })
    }

    // Viseme 말하기 입모양 (모듈 레벨 상수 사용 — 매 프레임 배열 생성 없음)
    speakTimer.current += delta
    if (speaking && speakTimer.current > 0.07 && scene) {
      speakTimer.current = 0
      visemeIdx.current = (visemeIdx.current + 1) % VISEME_NAMES.length
      const active = VISEME_NAMES[visemeIdx.current]

      scene.traverse((obj: THREE.Object3D) => {
        const mesh = obj as THREE.SkinnedMesh
        if (!mesh.morphTargetDictionary || !mesh.morphTargetInfluences) return
        VISEME_NAMES.forEach(v => {
          const i = mesh.morphTargetDictionary![v]
          if (i !== undefined) mesh.morphTargetInfluences![i] = v === active ? 0.75 : 0
        })
      })
    } else if (!speaking && speakTimer.current > 0.1 && scene) {
      // 말하기 끝났을 때 viseme 초기화
      speakTimer.current = 0
      scene.traverse((obj: THREE.Object3D) => {
        const mesh = obj as THREE.SkinnedMesh
        if (!mesh.morphTargetDictionary || !mesh.morphTargetInfluences) return
        VISEME_NAMES.forEach(v => {
          const i = mesh.morphTargetDictionary![v]
          if (i !== undefined) mesh.morphTargetInfluences![i] = 0
        })
      })
    }

    mixer?.update(delta)
  })

  // scene이 null이면 렌더링 안 함
  if (!scene) return null

  return (
    <group ref={group}>
      <primitive object={scene} />
    </group>
  )
}

export function AvatarModel(props: AvatarModelProps) {
  return (
    <Suspense fallback={null}>
      <GLBAvatar {...props} />
    </Suspense>
  )
}
