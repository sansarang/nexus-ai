import { createClient } from '@supabase/supabase-js'
import { SUPABASE_URL, SUPABASE_ANON_KEY } from '../config/services'

const supabaseUrl  = SUPABASE_URL  || 'https://placeholder.supabase.co'
const supabaseKey  = SUPABASE_ANON_KEY || 'placeholder'

export const supabase = createClient(supabaseUrl, supabaseKey, {
  auth: {
    persistSession: true,
    autoRefreshToken: true,
    detectSessionInUrl: false,
    flowType: 'implicit',
  },
})

export type SubscriptionStatus = 'active' | 'trial' | 'expired' | 'none'

export interface SubscriptionRow {
  id: string
  user_id: string
  paddle_subscription_id: string | null
  paddle_customer_id: string | null
  status: SubscriptionStatus
  trial_ends_at: string | null
  current_period_end: string | null
  created_at: string
  updated_at: string
}

export const BACKEND = 'http://127.0.0.1:17891'

/**
 * Google OAuth — Rust 인증 서버(17891) 콜백 방식
 * Rust가 포트 17891을 직접 소유하므로 Go 백엔드 실행 여부 무관.
 * Chrome "앱 열기" 다이얼로그 없음.
 */
export async function signInWithGoogle(loginHint?: string): Promise<void> {
  const { data, error } = await supabase.auth.signInWithOAuth({
    provider: 'google',
    options: {
      redirectTo: 'nexus://auth/callback',
      skipBrowserRedirect: true,
      queryParams: {
        access_type: 'offline',
        prompt: 'consent',
        ...(loginHint ? { login_hint: loginHint } : {}),
      },
    },
  })
  if (error) throw error
  if (data.url) {
    const { open } = await import('@tauri-apps/plugin-shell')
    await open(data.url)
  }
}

/** 로그아웃 */
export async function signOut(): Promise<void> {
  await supabase.auth.signOut()
}

/** 현재 세션의 구독 정보 조회 */
export async function fetchSubscription(userId: string): Promise<SubscriptionRow | null> {
  const { data, error } = await supabase
    .from('subscriptions')
    .select('*')
    .eq('user_id', userId)
    .maybeSingle()
  if (error) {
    console.error('fetchSubscription error:', error)
    return null
  }
  return data as SubscriptionRow | null
}

/** 체험판 구독 생성 (로그인 직후) */
export async function createTrialSubscription(userId: string): Promise<void> {
  const trialEnd = new Date(Date.now() + 3 * 24 * 60 * 60 * 1000).toISOString()
  await supabase.from('subscriptions').upsert(
    {
      user_id: userId,
      status: 'trial',
      trial_ends_at: trialEnd,
      current_period_end: trialEnd,
      updated_at: new Date().toISOString(),
    },
    { onConflict: 'user_id' }
  )
}

export interface UserSettings {
  assistant_name: string
  user_name: string
  user_lang: string
  primary_color: string
  accent_color: string
  glb_url: string
  preset: string
  tts_voice: string
  character_id: string
  is_onboarded: boolean
}

/** 사용자 설정 저장 */
export async function saveUserSettings(userId: string, settings: Partial<UserSettings>): Promise<void> {
  await supabase.from('user_settings').upsert(
    { user_id: userId, ...settings, updated_at: new Date().toISOString() },
    { onConflict: 'user_id' }
  )
}

/** 사용자 설정 불러오기 */
export async function fetchUserSettings(userId: string): Promise<UserSettings | null> {
  const { data, error } = await supabase
    .from('user_settings')
    .select('*')
    .eq('user_id', userId)
    .maybeSingle()
  if (error) return null
  return data as UserSettings | null
}

// ── 팀 워크스페이스 ───────────────────────────────────────────────

export interface TeamRow {
  id: string
  name: string
  owner_id: string
  invite_code: string
  created_at: string
}

export interface TeamMemberRow {
  team_id: string
  user_id: string
  role: 'admin' | 'member'
  joined_at: string
}

export interface TeamWorkflowRow {
  id: string
  team_id: string
  shared_by: string
  name: string
  description: string
  workflow_json: string
  created_at: string
}

export interface TeamPersonaRow {
  team_id: string
  persona_id: string
  persona_name: string
  primary_color: string
  system_prompt: string
  updated_by: string
  updated_at: string
}

export async function createTeam(userId: string, name: string): Promise<TeamRow | null> {
  const inviteCode = Math.random().toString(36).slice(2, 8).toUpperCase()
  const { data, error } = await supabase
    .from('teams')
    .insert({ name, owner_id: userId, invite_code: inviteCode })
    .select()
    .single()
  if (error) { console.error('createTeam:', error); return null }
  await supabase.from('team_members').insert({ team_id: data.id, user_id: userId, role: 'admin' })
  localStorage.setItem('nexus-team-id', data.id)
  localStorage.setItem('nexus-user-id', userId)
  return data as TeamRow
}

export async function joinTeam(userId: string, inviteCode: string): Promise<TeamRow | null> {
  const { data: team, error } = await supabase
    .from('teams')
    .select('*')
    .eq('invite_code', inviteCode.toUpperCase())
    .maybeSingle()
  if (error || !team) return null
  await supabase.from('team_members').upsert(
    { team_id: team.id, user_id: userId, role: 'member' },
    { onConflict: 'team_id,user_id' }
  )
  localStorage.setItem('nexus-team-id', team.id)
  localStorage.setItem('nexus-user-id', userId)
  return team as TeamRow
}

export async function fetchMyTeam(userId: string): Promise<TeamRow | null> {
  const { data } = await supabase
    .from('team_members')
    .select('team_id, teams(*)')
    .eq('user_id', userId)
    .maybeSingle()
  if (!data) return null
  return (data as any).teams as TeamRow
}

export async function fetchTeamMembers(teamId: string): Promise<TeamMemberRow[]> {
  const { data } = await supabase
    .from('team_members')
    .select('*')
    .eq('team_id', teamId)
  return (data as TeamMemberRow[]) || []
}

export async function shareWorkflowToTeam(teamId: string, userId: string, name: string, description: string, workflowJson: string): Promise<boolean> {
  const { error } = await supabase.from('team_workflows').insert({
    team_id: teamId, shared_by: userId, name, description, workflow_json: workflowJson,
  })
  return !error
}

export async function fetchTeamWorkflows(teamId: string): Promise<TeamWorkflowRow[]> {
  const { data } = await supabase
    .from('team_workflows')
    .select('*')
    .eq('team_id', teamId)
    .order('created_at', { ascending: false })
  return (data as TeamWorkflowRow[]) || []
}

export async function setTeamPersona(teamId: string, userId: string, persona: Omit<TeamPersonaRow, 'team_id' | 'updated_by' | 'updated_at'>): Promise<boolean> {
  const { error } = await supabase.from('team_settings').upsert(
    { team_id: teamId, ...persona, updated_by: userId, updated_at: new Date().toISOString() },
    { onConflict: 'team_id' }
  )
  return !error
}

export async function fetchTeamPersona(teamId: string): Promise<TeamPersonaRow | null> {
  const { data } = await supabase
    .from('team_settings')
    .select('*')
    .eq('team_id', teamId)
    .maybeSingle()
  return data as TeamPersonaRow | null
}

/** 구독 상태 계산 (만료 여부 포함) */
export function resolveStatus(row: SubscriptionRow | null): SubscriptionStatus {
  if (!row) return 'trial'  // 조회 실패 시 trial로 간주 (false positive 방지)
  if (row.status === 'active') return 'active'
  if (row.status === 'trial') {
    const end = row.trial_ends_at ? new Date(row.trial_ends_at) : null
    if (end && end < new Date()) return 'expired'
    return 'trial'
  }
  return row.status
}
