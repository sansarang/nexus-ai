/**
 * Nexus SSE Client
 * - /api/alerts/stream  → Proactive AI 알림
 * - /api/tasks/stream   → Task Queue 진행상황
 * 두 스트림을 관리하고 콜백으로 이벤트 전달
 */

const BASE = 'http://127.0.0.1:17891'

export interface ProactiveAlert {
  id: string
  level: 'info' | 'warn' | 'critical'
  title: string
  message: string
  action?: string
  timestamp: string
  dismissed: boolean
}

export interface TaskUpdate {
  type: 'task_update'
  id: string
  name: string
  status: 'pending' | 'running' | 'done' | 'failed' | 'cancelled'
  progress: number
  message: string
  error?: string
  result?: Record<string, unknown>
  finished_at?: string
}

type AlertCallback = (alert: ProactiveAlert) => void
type TaskCallback  = (update: TaskUpdate) => void

class NexusSSEClient {
  private alertES: EventSource | null = null
  private taskES:  EventSource | null = null
  private alertCallbacks: Set<AlertCallback> = new Set()
  private taskCallbacks:  Set<TaskCallback>  = new Set()
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private connected = false

  connect() {
    this.connectAlerts()
    this.connectTasks()
  }

  disconnect() {
    this.alertES?.close()
    this.taskES?.close()
    this.alertES = null
    this.taskES  = null
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer)
    this.connected = false
  }

  onAlert(cb: AlertCallback): () => void {
    this.alertCallbacks.add(cb)
    return () => this.alertCallbacks.delete(cb)
  }

  onTask(cb: TaskCallback): () => void {
    this.taskCallbacks.add(cb)
    return () => this.taskCallbacks.delete(cb)
  }

  isConnected() { return this.connected }

  private connectAlerts() {
    if (this.alertES) return
    try {
      this.alertES = new EventSource(`${BASE}/api/alerts/stream`)

      this.alertES.onopen = () => { this.connected = true }

      this.alertES.onmessage = (e) => {
        try {
          const data = JSON.parse(e.data)
          if (data.type === 'connected') return
          this.alertCallbacks.forEach(cb => cb(data as ProactiveAlert))
        } catch { /* 파싱 실패 무시 */ }
      }

      this.alertES.onerror = () => {
        this.alertES?.close()
        this.alertES = null
        this.connected = false
        this.scheduleReconnect()
      }
    } catch { /* EventSource 미지원 환경 */ }
  }

  private connectTasks() {
    if (this.taskES) return
    try {
      this.taskES = new EventSource(`${BASE}/api/tasks/stream`)

      this.taskES.onmessage = (e) => {
        try {
          const data = JSON.parse(e.data)
          if (data.type === 'connected') return
          this.taskCallbacks.forEach(cb => cb(data as TaskUpdate))
        } catch { /* 파싱 실패 무시 */ }
      }

      this.taskES.onerror = () => {
        this.taskES?.close()
        this.taskES = null
        this.scheduleReconnect()
      }
    } catch { /* EventSource 미지원 환경 */ }
  }

  private scheduleReconnect() {
    if (this.reconnectTimer) return
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null
      this.connectAlerts()
      this.connectTasks()
    }, 5000)
  }
}

export const nexusSSE = new NexusSSEClient()
