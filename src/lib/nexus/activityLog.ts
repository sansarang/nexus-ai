/**
 * Activity Log — 사용자 작업 이력 + Undo 지원.
 *
 * 모든 destructive 작업(파일 정리·삭제·이동·전원 명령 등)을 기록.
 * "방금 작업 취소" 명령으로 가장 최근 작업의 reverse action 시도.
 */

import type { Intent } from './intentDetector'

export interface ActivityEntry {
  id: string
  ts: number
  intent: Intent | string
  /** 사용자에게 보일 설명 */
  label: string
  /** 결과 카테고리 */
  status: 'success' | 'failure' | 'cancelled'
  /** Undo 정보 (있으면 되돌리기 가능) */
  undo?: {
    /** 어떤 인텐트를 실행하면 되돌릴 수 있는지 */
    intent: Intent | string
    /** 어떤 명령어로 실행할지 (사용자에게 보여줄 자연어) */
    command: string
    /** 추가 메타데이터 (예: 백업 파일 경로) */
    meta?: Record<string, unknown>
  }
  /** 추가 컨텍스트 */
  detail?: string
}

const KEY = 'nexus-activity-log'
const MAX_ENTRIES = 200

export function logActivity(entry: Omit<ActivityEntry, 'id' | 'ts'>): ActivityEntry {
  if (typeof localStorage === 'undefined') {
    return { ...entry, id: '_no_storage', ts: Date.now() }
  }
  const full: ActivityEntry = {
    ...entry,
    id: `act_${Date.now()}_${Math.random().toString(36).slice(2, 7)}`,
    ts: Date.now(),
  }
  const list = loadActivityLog()
  list.unshift(full)
  if (list.length > MAX_ENTRIES) list.length = MAX_ENTRIES
  try { localStorage.setItem(KEY, JSON.stringify(list)) } catch { /* quota exceeded */ }
  return full
}

export function loadActivityLog(): ActivityEntry[] {
  if (typeof localStorage === 'undefined') return []
  try {
    return JSON.parse(localStorage.getItem(KEY) ?? '[]') as ActivityEntry[]
  } catch { return [] }
}

/** 가장 최근 undo 가능한 작업 찾기 */
export function lastUndoable(): ActivityEntry | null {
  const list = loadActivityLog()
  return list.find(e => e.undo) ?? null
}

/** undo 처리 — 해당 작업의 reverse intent를 반환 (caller가 sendText로 실행) */
export function markUndone(entryId: string) {
  if (typeof localStorage === 'undefined') return
  const list = loadActivityLog()
  const e = list.find(x => x.id === entryId)
  if (e) {
    e.status = 'cancelled'
    delete e.undo // 이미 되돌렸으니 재실행 방지
    localStorage.setItem(KEY, JSON.stringify(list))
  }
}

export function clearActivityLog() {
  if (typeof localStorage !== 'undefined') {
    localStorage.removeItem(KEY)
  }
}
