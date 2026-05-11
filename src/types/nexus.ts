export type NexusEmotion = 'neutral' | 'concerned' | 'happy' | 'alert' | 'humorous'

export interface NexusStep {
  action: string
  params: Record<string, unknown>
  confirmRequired: boolean
  inlineView?: string | null
}

export interface Message {
  id: string
  role: 'user' | 'nexus'
  text: string
  emotion?: NexusEmotion
  timestamp: Date
  steps?: NexusStep[]
  pendingSteps?: NexusStep[]
  actionDone?: boolean
}

export interface InlineView {
  type: string
  data: unknown
}

export interface GeminiResponse {
  text: string
  emotion?: NexusEmotion
  steps: NexusStep[]
  needs_clarify?: boolean
  clarify_question?: string
  clarify_intent?: string
  clarify_params?: Record<string, unknown>
  needs_preview?: boolean
  preview_items?: Array<{ title: string; url: string }>
}
