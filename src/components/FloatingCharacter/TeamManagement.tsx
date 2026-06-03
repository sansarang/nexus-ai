/**
 * TeamManagement — Team 워크스페이스 관리 UI (Phase Team)
 *
 * 사장님 비전: ₩79,000/월 Team 플랜 사용자 (owner) UI
 *
 * 기능:
 *  - 현재 workspace 정보 (이름/시트/사용 중 시트)
 *  - 멤버 목록 (owner / member 역할 표시)
 *  - 새 멤버 초대 (이메일 입력 → invite_url 복사)
 *  - 멤버 제거 (owner 만 가능)
 *
 * 비-owner 는 워크스페이스 정보만 조회.
 */

import { useEffect, useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'

const API = 'http://127.0.0.1:17891'

interface Workspace {
  id: string
  name: string
  owner_email: string
  plan: string
  seat_limit: number
  created_at: string
}

interface Member {
  workspace_id: string
  email: string
  role: string
  joined_at: string
  invited_by: string
}

interface TeamManagementProps {
  userEmail: string
  isOpen: boolean
  onClose: () => void
}

export function TeamManagement({ userEmail, isOpen, onClose }: TeamManagementProps) {
  const [workspace, setWorkspace] = useState<Workspace | null>(null)
  const [members, setMembers] = useState<Member[]>([])
  const [loading, setLoading] = useState(false)
  const [inviteEmail, setInviteEmail] = useState('')
  const [inviteUrl, setInviteUrl] = useState('')
  const [error, setError] = useState('')
  const [copied, setCopied] = useState(false)

  const isOwner = workspace?.owner_email === userEmail

  // 워크스페이스 + 멤버 로드
  const load = async () => {
    if (!userEmail) return
    setLoading(true)
    setError('')
    try {
      const res = await fetch(`${API}/api/team/members?email=${encodeURIComponent(userEmail)}`)
      const data = await res.json()
      if (data.success) {
        setWorkspace(data.workspace)
        setMembers(data.members ?? [])
      } else {
        setError('Team 워크스페이스가 없습니다. Team 플랜 구독 후 워크스페이스가 자동 생성됩니다.')
      }
    } catch (e: any) {
      setError(e?.message ?? 'load failed')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (isOpen) load()
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isOpen, userEmail])

  // 초대 발급
  const handleInvite = async () => {
    if (!workspace || !inviteEmail.trim()) return
    setError('')
    try {
      const res = await fetch(`${API}/api/team/invite`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          workspace_id: workspace.id,
          owner_email: userEmail,
          target_email: inviteEmail.trim(),
        }),
      })
      const data = await res.json()
      if (data.success) {
        setInviteUrl(data.invite_url ?? '')
        setInviteEmail('')
      } else {
        setError(data.message ?? 'invite failed')
      }
    } catch (e: any) {
      setError(e?.message ?? 'invite failed')
    }
  }

  const handleCopy = () => {
    if (!inviteUrl) return
    navigator.clipboard.writeText(inviteUrl)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  // 멤버 제거
  const handleRemove = async (targetEmail: string) => {
    if (!workspace || !isOwner) return
    if (!confirm(`${targetEmail} 님을 워크스페이스에서 제거하시겠습니까?`)) return
    try {
      const res = await fetch(`${API}/api/team/remove`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          workspace_id: workspace.id,
          owner_email: userEmail,
          target_email: targetEmail,
        }),
      })
      const data = await res.json()
      if (data.success) {
        await load()
      } else {
        setError(data.message ?? 'remove failed')
      }
    } catch (e: any) {
      setError(e?.message ?? 'remove failed')
    }
  }

  if (!isOpen) return null

  return (
    <AnimatePresence>
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        onClick={onClose}
        style={{
          position: 'fixed', inset: 0, zIndex: 9999,
          background: 'rgba(8,11,22,0.6)',
          backdropFilter: 'blur(12px) saturate(1.4)',
          WebkitBackdropFilter: 'blur(12px) saturate(1.4)',
          display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 24,
        }}
      >
        <motion.div
          onClick={e => e.stopPropagation()}
          initial={{ opacity: 0, y: 16, scale: 0.96 }}
          animate={{ opacity: 1, y: 0, scale: 1 }}
          exit={{ opacity: 0, y: -8, scale: 0.96 }}
          transition={{ duration: 0.24 }}
          style={{
            width: 'min(560px, 100%)',
            background: 'rgba(15,23,42,0.92)',
            backdropFilter: 'blur(40px) saturate(1.8)',
            WebkitBackdropFilter: 'blur(40px) saturate(1.8)',
            border: '1px solid rgba(255,255,255,0.10)',
            borderTop: '2px solid rgba(14,165,233,0.7)',
            borderRadius: 22,
            padding: '28px 28px 24px',
            color: '#f1f5f9',
            boxShadow: '0 24px 64px rgba(0,0,0,0.5), inset 0 1px 0 rgba(255,255,255,0.06)',
            maxHeight: '85vh', overflowY: 'auto',
          }}
        >
          {/* 헤더 */}
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 18 }}>
            <div>
              <div style={{ fontSize: 11, fontWeight: 700, color: '#7dd3fc', letterSpacing: '0.08em', textTransform: 'uppercase', marginBottom: 4 }}>
                🏢 Team Workspace
              </div>
              <div style={{ fontSize: 20, fontWeight: 800 }}>
                {workspace?.name ?? 'Team 관리'}
              </div>
            </div>
            <button onClick={onClose} style={{
              width: 32, height: 32, borderRadius: '50%',
              background: 'rgba(255,255,255,0.08)', border: 'none',
              color: '#f1f5f9', fontSize: 16, cursor: 'pointer',
            }}>✕</button>
          </div>

          {loading && <div style={{ padding: 24, textAlign: 'center', color: 'rgba(255,255,255,0.5)' }}>로딩 중...</div>}

          {error && !workspace && (
            <div style={{
              padding: 24, textAlign: 'center',
              background: 'rgba(245,158,11,0.08)', border: '1px solid rgba(245,158,11,0.3)',
              borderRadius: 14, fontSize: 13, color: '#fbbf24', lineHeight: 1.6,
            }}>
              {error}
            </div>
          )}

          {workspace && (
            <>
              {/* 워크스페이스 정보 */}
              <div style={{
                display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 10,
                marginBottom: 22,
              }}>
                <div style={{ padding: 14, background: 'rgba(255,255,255,0.04)', border: '1px solid rgba(255,255,255,0.08)', borderRadius: 12, textAlign: 'center' }}>
                  <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.55)', marginBottom: 4 }}>플랜</div>
                  <div style={{ fontSize: 14, fontWeight: 800, color: '#7dd3fc' }}>{workspace.plan.toUpperCase()}</div>
                </div>
                <div style={{ padding: 14, background: 'rgba(255,255,255,0.04)', border: '1px solid rgba(255,255,255,0.08)', borderRadius: 12, textAlign: 'center' }}>
                  <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.55)', marginBottom: 4 }}>시트</div>
                  <div style={{ fontSize: 14, fontWeight: 800 }}>{members.length} / {workspace.seat_limit}</div>
                </div>
                <div style={{ padding: 14, background: 'rgba(255,255,255,0.04)', border: '1px solid rgba(255,255,255,0.08)', borderRadius: 12, textAlign: 'center' }}>
                  <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.55)', marginBottom: 4 }}>역할</div>
                  <div style={{ fontSize: 14, fontWeight: 800, color: isOwner ? '#fbbf24' : '#94a3b8' }}>
                    {isOwner ? '👑 OWNER' : 'MEMBER'}
                  </div>
                </div>
              </div>

              {/* 초대 (owner 만) */}
              {isOwner && members.length < workspace.seat_limit && (
                <div style={{ marginBottom: 22 }}>
                  <div style={{ fontSize: 12, fontWeight: 700, color: 'rgba(255,255,255,0.85)', marginBottom: 8 }}>새 멤버 초대</div>
                  <div style={{ display: 'flex', gap: 8 }}>
                    <input
                      type="email"
                      value={inviteEmail}
                      onChange={e => setInviteEmail(e.target.value)}
                      placeholder="email@example.com"
                      style={{
                        flex: 1, padding: '10px 14px',
                        background: 'rgba(255,255,255,0.06)',
                        border: '1px solid rgba(255,255,255,0.12)',
                        borderRadius: 10, color: '#f1f5f9', fontSize: 13,
                        outline: 'none', boxSizing: 'border-box',
                      }}
                    />
                    <button
                      onClick={handleInvite}
                      disabled={!inviteEmail.trim()}
                      style={{
                        padding: '10px 18px', borderRadius: 10,
                        background: inviteEmail.trim() ? 'linear-gradient(135deg, #0ea5e9, #38bdf8)' : 'rgba(255,255,255,0.08)',
                        color: inviteEmail.trim() ? '#fff' : 'rgba(255,255,255,0.4)',
                        border: 'none', fontSize: 12, fontWeight: 700,
                        cursor: inviteEmail.trim() ? 'pointer' : 'not-allowed',
                      }}
                    >
                      초대 발급
                    </button>
                  </div>

                  {inviteUrl && (
                    <motion.div
                      initial={{ opacity: 0, y: -4 }}
                      animate={{ opacity: 1, y: 0 }}
                      style={{
                        marginTop: 10, padding: 12,
                        background: 'rgba(14,165,233,0.08)',
                        border: '1px solid rgba(14,165,233,0.3)',
                        borderRadius: 10,
                      }}
                    >
                      <div style={{ fontSize: 10, color: '#7dd3fc', fontWeight: 700, marginBottom: 6 }}>✅ 초대 링크 발급 (7일 유효)</div>
                      <div style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
                        <input
                          readOnly
                          value={inviteUrl}
                          style={{
                            flex: 1, padding: '6px 10px',
                            background: 'rgba(0,0,0,0.3)',
                            border: '1px solid rgba(255,255,255,0.08)',
                            borderRadius: 6, color: 'rgba(255,255,255,0.85)',
                            fontSize: 10, fontFamily: 'monospace',
                          }}
                        />
                        <button
                          onClick={handleCopy}
                          style={{
                            padding: '6px 12px', borderRadius: 6,
                            background: copied ? '#22c55e' : 'rgba(255,255,255,0.1)',
                            color: '#fff', border: 'none', fontSize: 11, fontWeight: 700,
                            cursor: 'pointer', minWidth: 56,
                          }}
                        >
                          {copied ? '복사됨' : '복사'}
                        </button>
                      </div>
                      <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.45)', marginTop: 6 }}>
                        멤버에게 이 링크를 전달하세요. 클릭 → 가입 → 자동 워크스페이스 참여.
                      </div>
                    </motion.div>
                  )}

                  {error && (
                    <div style={{ marginTop: 8, fontSize: 11, color: '#f87171' }}>⚠️ {error}</div>
                  )}
                </div>
              )}

              {isOwner && members.length >= workspace.seat_limit && (
                <div style={{
                  marginBottom: 22, padding: 12,
                  background: 'rgba(245,158,11,0.08)', border: '1px solid rgba(245,158,11,0.3)',
                  borderRadius: 10, fontSize: 12, color: '#fbbf24',
                }}>
                  ⚠️ 시트 한도 도달 — 멤버 제거 또는 플랜 업그레이드
                </div>
              )}

              {/* 멤버 목록 */}
              <div>
                <div style={{ fontSize: 12, fontWeight: 700, color: 'rgba(255,255,255,0.85)', marginBottom: 10 }}>
                  멤버 ({members.length})
                </div>
                <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                  {members.map(m => (
                    <div key={m.email} style={{
                      display: 'flex', alignItems: 'center', justifyContent: 'space-between',
                      padding: '10px 14px',
                      background: 'rgba(255,255,255,0.04)',
                      border: '1px solid rgba(255,255,255,0.06)',
                      borderRadius: 10,
                    }}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 10, flex: 1, minWidth: 0 }}>
                        <span style={{ fontSize: 16 }}>{m.role === 'owner' ? '👑' : '👤'}</span>
                        <div style={{ minWidth: 0 }}>
                          <div style={{ fontSize: 13, fontWeight: 600, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                            {m.email}
                          </div>
                          <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.5)', marginTop: 1 }}>
                            {m.role === 'owner' ? 'Owner' : `Joined ${new Date(m.joined_at).toLocaleDateString()}`}
                          </div>
                        </div>
                      </div>
                      {isOwner && m.role !== 'owner' && (
                        <button
                          onClick={() => handleRemove(m.email)}
                          style={{
                            padding: '6px 10px', borderRadius: 6,
                            background: 'rgba(239,68,68,0.12)',
                            color: '#f87171', border: '1px solid rgba(239,68,68,0.25)',
                            fontSize: 10, fontWeight: 700, cursor: 'pointer',
                          }}
                        >
                          제거
                        </button>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            </>
          )}
        </motion.div>
      </motion.div>
    </AnimatePresence>
  )
}
