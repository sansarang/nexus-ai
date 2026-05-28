import { create } from 'zustand'
import { supabase, fetchSubscription, createTrialSubscription, resolveStatus, signOut as supabaseSignOut } from '../lib/supabase'

export type ViewId =
  | 'home' | 'repair' | 'security' | 'files'
  | 'translate' | 'clipboard' | 'memo' | 'settings'
  | 'autoclean' | 'updaterepair' | 'privacy' | 'monitor'
  | 'focus' | 'organize' | 'daily' | 'predictive' | 'voicememo'

export interface Issue {
  id: string
  title: string
  description: string
  severity: 'low' | 'medium' | 'high' | 'critical'
  category: string
  fixable: boolean
}

export interface ClipboardEntry {
  id: string
  type: 'text' | 'url' | 'image'
  content: string
  preview?: string
  pinned: boolean
  timestamp: Date
}

export interface Memo {
  id: string
  title: string
  body: string
  createdAt: Date
  updatedAt: Date
}

export interface Todo {
  id: string
  text: string
  done: boolean
  createdAt: Date
}

export interface SystemStats {
  cpu: number
  mem: number
  disk: number
  cpuTemp: number
  netUp: number
  netDown: number
  timestamp: number
}

export interface FocusSession {
  active: boolean
  mode: 'work' | 'break'
  elapsed: number
  total: number
  rounds: number
}

export interface DailyReport {
  date: string
  pcScore: number
  cpuAvg: number
  memAvg: number
  diskFree: number
  recommendations: string[]
  predictions: { label: string; value: number; trend: 'up' | 'down' | 'stable' }[]
}

interface AppState {
  /* 계정 / 구독 / 온보딩 */
  isLoggedIn: boolean
  isOnboarded: boolean
  userEmail: string
  subscriptionStatus: 'active' | 'trial' | 'expired' | 'none'
  subscriptionExpiry: string  // ISO date string

  /* 사용자 설정 */
  micEnabled: boolean
  setMicEnabled: (v: boolean) => void
  ttsVoice: string
  setTtsVoice: (v: string) => void
  assistantName: string
  userName: string
  userLang: 'ko' | 'en'
  characterId: 'iron' | 'luna' | 'doc' | 'pixie' | 'kira' | 'nova' | 'sora' | 'hana' | 'jin' | 'mira' | 'lumi' | 'joy' | 'custom'
  primaryColor: string
  accentColor: string

  /* PC 상태 */
  pcScore: number
  issues: Issue[]
  isScanning: boolean
  cpuUsage: number
  memUsage: number
  diskUsage: number

  /* 페르소나 */
  activePersonaId: string
  setActivePersonaId: (id: string) => void

  /* UI */
  currentView: ViewId
  commandOpen: boolean
  showWorkflowBuilder: boolean
  workflowBuilderInitialName: string | undefined
  setShowWorkflowBuilder: (val: boolean, name?: string) => void

  /* 클립보드 */
  clipboardHistory: ClipboardEntry[]

  /* 메모 / TODO */
  memos: Memo[]
  todos: Todo[]

  /* 모니터 */
  monitorHistory: SystemStats[]

  /* 집중 모드 */
  focusSession: FocusSession | null

  /* 데일리 리포트 */
  dailyReport: DailyReport | null

  /* 프라이버시 설정 */
  privacySettings: Record<string, boolean>

  /* Actions */
  setView: (view: ViewId) => void
  toggleCommand: () => void
  /** Alt+S/V/C 단축키 → 채팅 명령 자동 입력용 콜백 (FloatingCharacter가 등록) */
  triggerCommand?: (text: string) => void
  /** triggerCommand 콜백 등록 */
  registerTriggerCommand: (fn: (text: string) => void) => void
  setLoggedIn: (email: string, status: 'active' | 'trial' | 'expired' | 'none', expiry: string, userId?: string) => void
  setLoggedOut: () => Promise<void>
  refreshSubscription: () => Promise<void>
  setOnboarded: () => void
  setAssistantName: (name: string) => void
  setUserName: (name: string) => void
  setUserLang: (lang: 'ko' | 'en') => void
  setCharacterId: (id: 'iron' | 'luna' | 'doc' | 'pixie' | 'kira' | 'nova' | 'sora' | 'hana' | 'jin' | 'mira' | 'lumi' | 'joy' | 'custom') => void
  setPrimaryColor: (color: string) => void
  setAccentColor: (color: string) => void

  startScan: () => Promise<void>
  repairIssue: (id: string) => Promise<void>
  repairAll: () => Promise<{ before: number; after: number }>

  addClipboard: (entry: Omit<ClipboardEntry, 'id' | 'timestamp'>) => void
  removeClipboard: (id: string) => void
  pinClipboard: (id: string) => void

  addMemo: (title: string, body: string) => void
  updateMemo: (id: string, patch: Partial<Pick<Memo, 'title' | 'body'>>) => void
  deleteMemo: (id: string) => void

  addTodo: (text: string) => void
  toggleTodo: (id: string) => void
  deleteTodo: (id: string) => void

  pushStats: (s: SystemStats) => void
  setFocusSession: (s: FocusSession | null) => void
  setDailyReport: (r: DailyReport) => void
  setPrivacy: (key: string, val: boolean) => void
  fetchDailyReport: () => Promise<void>
  startMonitoring: () => () => void
}

const BACKEND = 'http://127.0.0.1:17891'

function uid() {
  return Math.random().toString(36).slice(2)
}

/* 더미 진단 데이터 (백엔드 없을 때) */
function dummyScan(): { score: number; issues: Issue[] } {
  return {
    score: 72,
    issues: [
      {
        id: 'temp',
        title: '임시 파일 3.2GB',
        description: '불필요한 임시 파일이 쌓여있어요',
        severity: 'medium',
        category: 'clean',
        fixable: true,
      },
      {
        id: 'startup',
        title: '시작 프로그램 12개',
        description: '부팅 속도를 느리게 만드는 프로그램들이 있어요',
        severity: 'low',
        category: 'startup',
        fixable: true,
      },
      {
        id: 'disk',
        title: '디스크 공간 82% 사용 중',
        description: 'C 드라이브 여유 공간이 부족해요',
        severity: 'high',
        category: 'disk',
        fixable: false,
      },
    ],
  }
}

function mockStats(): SystemStats {
  return {
    cpu: Math.random() * 40 + 10,
    mem: Math.random() * 30 + 50,
    disk: Math.random() * 20 + 60,
    cpuTemp: Math.random() * 20 + 45,
    netUp: Math.random() * 500,
    netDown: Math.random() * 2000,
    timestamp: Date.now(),
  }
}

function mockDailyReport(): DailyReport {
  return {
    date: new Date().toISOString().slice(0, 10),
    pcScore: 78,
    cpuAvg: 22.5,
    memAvg: 58.0,
    diskFree: 45.0,
    recommendations: [
      'PC 상태가 양호합니다.',
      '정기적인 임시 파일 정리를 권장합니다.',
      '시작 프로그램을 최적화하면 부팅 속도가 향상됩니다.',
    ],
    predictions: [
      { label: 'CPU 사용률', value: 25, trend: 'stable' },
      { label: '메모리 사용률', value: 60, trend: 'up' },
      { label: '디스크 여유', value: 45, trend: 'down' },
    ],
  }
}

export const useAppStore = create<AppState>((set, get) => ({
  isLoggedIn: !!localStorage.getItem('nexus-user-email'),
  isOnboarded: !!localStorage.getItem('nexus-onboarded'),
  userEmail: localStorage.getItem('nexus-user-email') ?? '',
  subscriptionStatus: (localStorage.getItem('nexus-sub-status') as 'active' | 'trial' | 'expired' | 'none') ?? 'trial',
  subscriptionExpiry: localStorage.getItem('nexus-sub-expiry') ?? '',
  micEnabled: localStorage.getItem('nexus-mic-enabled') === 'true',
  setMicEnabled: (v) => { localStorage.setItem('nexus-mic-enabled', String(v)); set({ micEnabled: v }) },
  ttsVoice: localStorage.getItem('nexus-tts-voice') ?? 'nova',
  setTtsVoice: (v) => { localStorage.setItem('nexus-tts-voice', v); set({ ttsVoice: v }) },
  assistantName: localStorage.getItem('nexus-assistant-name') ?? 'Nexus',
  userName: localStorage.getItem('nexus-user-name') ?? '',
  userLang: (localStorage.getItem('nexus-lang') as 'ko' | 'en') ?? 'ko',
  characterId: (localStorage.getItem('nexus-character') as 'iron' | 'luna' | 'doc' | 'pixie' | 'kira' | 'nova' | 'sora' | 'hana' | 'jin' | 'mira' | 'lumi' | 'joy' | 'custom') ?? 'sora',
  primaryColor: localStorage.getItem('nexus-primary-color') ?? '#a78bfa',
  accentColor: localStorage.getItem('nexus-accent-color') ?? '#f9a8d4',
  pcScore: 0,
  issues: [],
  isScanning: false,
  cpuUsage: 0,
  memUsage: 0,
  diskUsage: 0,
  activePersonaId: localStorage.getItem('nexus-persona-id') ?? 'nexus',
  setActivePersonaId: (id) => {
    localStorage.setItem('nexus-persona-id', id)
    set({ activePersonaId: id })
  },
  currentView: 'home',
  commandOpen: false,
  showWorkflowBuilder: false,
  workflowBuilderInitialName: undefined,
  setShowWorkflowBuilder: (val, name) => set({ showWorkflowBuilder: val, workflowBuilderInitialName: name }),
  clipboardHistory: [
    {
      id: '1',
      type: 'text',
      content: '안녕하세요! Nexus 테스트 클립보드입니다.',
      pinned: true,
      timestamp: new Date(Date.now() - 60000),
    },
    {
      id: '2',
      type: 'url',
      content: 'https://example.com',
      pinned: false,
      timestamp: new Date(Date.now() - 180000),
    },
    {
      id: '3',
      type: 'text',
      content: '회의록 내용: 1) 프로젝트 진행상황 2) 다음 스프린트 계획 3) 이슈 리뷰',
      pinned: false,
      timestamp: new Date(Date.now() - 600000),
    },
  ],
  memos: [],
  todos: [],
  monitorHistory: [],
  focusSession: null,
  dailyReport: null,
  privacySettings: {
    copilot: false,
    onedrive: false,
    telemetry: false,
    ads: false,
    cortana: false,
    widgets: false,
  },

  setView: (view) => set({ currentView: view }),
  toggleCommand: () => set((s) => ({ commandOpen: !s.commandOpen })),
  registerTriggerCommand: (fn) => set({ triggerCommand: fn }),
  setLoggedIn: (email, status, expiry, userId?: string) => {
    localStorage.setItem('nexus-user-email', email)
    localStorage.setItem('nexus-sub-status', status)
    localStorage.setItem('nexus-sub-expiry', expiry)
    if (userId) localStorage.setItem('nexus-user-id', userId)
    set({ isLoggedIn: true, userEmail: email, subscriptionStatus: status, subscriptionExpiry: expiry })
  },
  setLoggedOut: async () => {
    await supabaseSignOut().catch(() => {})
    localStorage.removeItem('nexus-user-email')
    localStorage.removeItem('nexus-sub-status')
    localStorage.removeItem('nexus-sub-expiry')
    localStorage.removeItem('nexus-user-id')
    set({ isLoggedIn: false, userEmail: '', subscriptionStatus: 'none', subscriptionExpiry: '' })
  },
  refreshSubscription: async () => {
    const email = localStorage.getItem('nexus-user-email')
    if (!email) return
    try {
      const { data: { session } } = await supabase.auth.getSession()
      if (!session) return
      localStorage.setItem('nexus-user-id', session.user.id)
      const row = await fetchSubscription(session.user.id)
      const status = resolveStatus(row)
      const expiry = row?.current_period_end ?? row?.trial_ends_at ?? ''
      localStorage.setItem('nexus-sub-status', status)
      localStorage.setItem('nexus-sub-expiry', expiry)
      set({ subscriptionStatus: status, subscriptionExpiry: expiry })
    } catch { /* 오프라인 시 무시 */ }
  },
  setOnboarded: () => {
    localStorage.setItem('nexus-onboarded', 'true')
    set({ isOnboarded: true })
  },
  setAssistantName: (name) => {
    localStorage.setItem('nexus-assistant-name', name)
    set({ assistantName: name })
  },
  setUserName: (name) => {
    localStorage.setItem('nexus-user-name', name)
    set({ userName: name })
  },
  setUserLang: (lang) => {
    localStorage.setItem('nexus-lang', lang)
    set({ userLang: lang })
    // 백엔드에 영속 저장 — 자동화 기능(모닝 브리핑, 알림 등)이 이 값을 사용
    fetch('http://127.0.0.1:17891/api/settings/lang', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ lang }),
    }).catch(() => {}) // 백엔드 미연결 시 무시
  },
  setCharacterId: (id) => {
    localStorage.setItem('nexus-character', id)
    set({ characterId: id })
  },
  setPrimaryColor: (color) => {
    localStorage.setItem('nexus-primary-color', color)
    set({ primaryColor: color })
  },
  setAccentColor: (color) => {
    localStorage.setItem('nexus-accent-color', color)
    set({ accentColor: color })
  },

  startScan: async () => {
    if (get().isScanning) return
    set({ isScanning: true })
    try {
      const res = await fetch(`${BACKEND}/api/scan`, { method: 'POST' })
      const data = await res.json() as { score: number; issues: Issue[] }
      set({ pcScore: data.score, issues: data.issues, isScanning: false })
    } catch {
      await new Promise((r) => setTimeout(r, 1800))
      const d = dummyScan()
      set({ pcScore: d.score, issues: d.issues, isScanning: false })
    }
    set({
      cpuUsage: Math.floor(Math.random() * 40) + 10,
      memUsage: Math.floor(Math.random() * 30) + 50,
      diskUsage: 82,
    })
  },

  repairIssue: async (id) => {
    try {
      await fetch(`${BACKEND}/api/repair`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ items: [id] }),
      })
    } catch { /* 무시 */ }
    await new Promise((r) => setTimeout(r, 1200))
    set((s) => ({
      issues: s.issues.filter((i) => i.id !== id),
      pcScore: Math.min(100, s.pcScore + 5),
    }))
  },

  repairAll: async () => {
    const before = get().pcScore
    const fixable = get().issues.filter((i) => i.fixable)
    try {
      await fetch(`${BACKEND}/api/repair`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ items: fixable.map((i) => i.id) }),
      })
    } catch { /* 무시 */ }
    await new Promise((r) => setTimeout(r, 2000))
    const after = Math.min(100, before + fixable.length * 8)
    set((s) => ({
      issues: s.issues.filter((i) => !i.fixable),
      pcScore: after,
    }))
    return { before, after }
  },

  addClipboard: (entry) =>
    set((s) => ({
      clipboardHistory: [
        { ...entry, id: uid(), timestamp: new Date() },
        ...s.clipboardHistory,
      ].slice(0, 50),
    })),
  removeClipboard: (id) =>
    set((s) => ({ clipboardHistory: s.clipboardHistory.filter((c) => c.id !== id) })),
  pinClipboard: (id) =>
    set((s) => ({
      clipboardHistory: s.clipboardHistory.map((c) =>
        c.id === id ? { ...c, pinned: !c.pinned } : c
      ),
    })),

  addMemo: (title, body) =>
    set((s) => ({
      memos: [{ id: uid(), title, body, createdAt: new Date(), updatedAt: new Date() }, ...s.memos],
    })),
  updateMemo: (id, patch) =>
    set((s) => ({
      memos: s.memos.map((m) => (m.id === id ? { ...m, ...patch, updatedAt: new Date() } : m)),
    })),
  deleteMemo: (id) => set((s) => ({ memos: s.memos.filter((m) => m.id !== id) })),

  addTodo: (text) =>
    set((s) => ({
      todos: [{ id: uid(), text, done: false, createdAt: new Date() }, ...s.todos],
    })),
  toggleTodo: (id) =>
    set((s) => ({
      todos: s.todos.map((t) => (t.id === id ? { ...t, done: !t.done } : t)),
    })),
  deleteTodo: (id) => set((s) => ({ todos: s.todos.filter((t) => t.id !== id) })),

  pushStats: (s) =>
    set((state) => ({
      monitorHistory: [...state.monitorHistory, s].slice(-30),
    })),

  setFocusSession: (s) => set({ focusSession: s }),
  setDailyReport: (r) => set({ dailyReport: r }),
  setPrivacy: (key, val) =>
    set((s) => ({ privacySettings: { ...s.privacySettings, [key]: val } })),

  fetchDailyReport: async () => {
    try {
      const res = await fetch(`${BACKEND}/api/daily-report`)
      const data = await res.json() as {
        date: string; pc_score: number; cpu_avg: number; mem_avg: number
        disk_free_gb: number; recommendations: string[]
        predictions: { label: string; value: number; trend: 'up' | 'down' | 'stable' }[]
      }
      get().setDailyReport({
        date: data.date,
        pcScore: data.pc_score,
        cpuAvg: data.cpu_avg,
        memAvg: data.mem_avg,
        diskFree: data.disk_free_gb,
        recommendations: data.recommendations,
        predictions: data.predictions,
      })
    } catch {
      get().setDailyReport(mockDailyReport())
    }
  },

  startMonitoring: () => {
    let handle: ReturnType<typeof setInterval>
    const poll = async () => {
      try {
        const res = await fetch(`${BACKEND}/api/stats`)
        const data = await res.json() as {
          cpu: number; mem: number; disk: number
          cpu_temp: number; net_up: number; net_down: number; timestamp: number
        }
        get().pushStats({
          cpu: data.cpu,
          mem: data.mem,
          disk: data.disk,
          cpuTemp: data.cpu_temp,
          netUp: data.net_up,
          netDown: data.net_down,
          timestamp: data.timestamp,
        })
      } catch {
        get().pushStats(mockStats())
      }
    }
    poll()
    handle = setInterval(poll, 2000)
    return () => clearInterval(handle)
  },
}))

/* DEV: 관리자 테스트용 — localStorage 초기값 세팅 */
;(function devAdminInit() {
  if (!localStorage.getItem('nexus-glb-url'))
    localStorage.setItem('nexus-glb-url', '/char_astronaut.glb')
  if (!localStorage.getItem('nexus-preset'))
    localStorage.setItem('nexus-preset', 'kpop_star')
  if (!localStorage.getItem('nexus-assistant-name'))
    localStorage.setItem('nexus-assistant-name', '아리')
  if (!localStorage.getItem('nexus-user-name'))
    localStorage.setItem('nexus-user-name', '주인님')
  if (!localStorage.getItem('nexus-primary-color'))
    localStorage.setItem('nexus-primary-color', '#f472b6')
  if (!localStorage.getItem('nexus-accent-color'))
    localStorage.setItem('nexus-accent-color', '#818cf8')
  if (!localStorage.getItem('nexus-tts-voice'))
    localStorage.setItem('nexus-tts-voice', 'nova')
  // API 키는 .env → services.ts → gemini_engine.ts 에서 직접 읽음 (localStorage 불필요)
})()
