// Presets.ts — CHARACTER_CATALOG 기반 호환 스텁 (3D Presets 제거 후)
import { CHARACTER_CATALOG } from '../characters'
import type { CharacterId } from '../characters'

export type RealisticStyleId = CharacterId

export interface RealisticStylePreset {
  id: RealisticStyleId
  name: string
  nameEn: string
  desc: string
  previewEmoji: string
  primaryColor: string
  accentColor: string
  glbUrl?: string | null
}

export const REALISTIC_STYLE_PRESETS: RealisticStylePreset[] = CHARACTER_CATALOG
  .filter(c => c.id !== 'custom')
  .map(c => ({
    id:           c.id,
    name:         c.name,
    nameEn:       c.nameEn,
    desc:         c.desc,
    previewEmoji: c.id === 'nexus' ? '🤖' : c.id === 'lumi' ? '✨' : '⚙️',
    primaryColor: c.primaryColor,
    accentColor:  c.accentColor,
    glbUrl:       null,
  }))

export const REALISTIC_STYLE_PRESETS_MAP: Record<RealisticStyleId, RealisticStylePreset> =
  Object.fromEntries(REALISTIC_STYLE_PRESETS.map(p => [p.id, p])) as Record<RealisticStyleId, RealisticStylePreset>

export function styleToPreset(id: RealisticStyleId): RealisticStylePreset {
  return REALISTIC_STYLE_PRESETS_MAP[id] ?? REALISTIC_STYLE_PRESETS[0]
}
