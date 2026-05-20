import { useState, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Copy, Trash2, Plus, Key, TrendingUp, ChevronDown, ChevronUp } from 'lucide-react'

const BASE = 'http://127.0.0.1:17891'

interface APIKey {
  id: string
  key: string
  name: string
  plan: string
  created_at: string
  last_used_at: string
  monthly_limit: number
  used_this_month: number
  endpoints: string[]
  active: boolean
}

interface PlanInfo {
  id: string
  name: string
  price: number
  monthly_limit: number
  description: string
}

const PLAN_COLORS: Record<string, string> = {
  starter: '#6366f1',
  growth: '#0891b2',
  enterprise: '#059669',
}

const PLAN_LABELS: Record<string, string> = {
  starter: 'Starter',
  growth: 'Growth',
  enterprise: 'Enterprise',
}

function usagePct(used: number, limit: number) {
  if (limit === -1) return 0
  return Math.min(100, Math.round((used / limit) * 100))
}

function formatDate(s: string) {
  if (!s) return '없음'
  try {
    const d = new Date(s)
    const diff = Date.now() - d.getTime()
    const mins = Math.floor(diff / 60000)
    if (mins < 1) return '방금 전'
    if (mins < 60) return `${mins}분 전`
    const hrs = Math.floor(mins / 60)
    if (hrs < 24) return `${hrs}시간 전`
    return d.toLocaleDateString('ko-KR')
  } catch {
    return s
  }
}

export function APIKeyManager() {
  const [keys, setKeys] = useState<APIKey[]>([])
  const [plans, setPlans] = useState<PlanInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [creating, setCreating] = useState(false)
  const [showCreate, setShowCreate] = useState(false)
  const [newName, setNewName] = useState('')
  const [newPlan, setNewPlan] = useState('starter')
  const [copiedId, setCopiedId] = useState<string | null>(null)
  const [expandedId, setExpandedId] = useState<string | null>(null)

  const load = async () => {
    setLoading(true)
    try {
      const [keysRes, plansRes] = await Promise.all([
        fetch(`${BASE}/api/enterprise/keys`).then(r => r.json()),
        fetch(`${BASE}/api/enterprise/plans`).then(r => r.json()),
      ])
      setKeys(keysRes.keys ?? [])
      setPlans(plansRes.plans ?? [])
    } catch {
      // backend not available
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { load() }, [])

  const createKey = async () => {
    if (!newName.trim()) return
    setCreating(true)
    try {
      await fetch(`${BASE}/api/enterprise/keys`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: newName.trim(), plan: newPlan }),
      })
      setNewName('')
      setShowCreate(false)
      await load()
    } catch {
      // ignore
    } finally {
      setCreating(false)
    }
  }

  const revokeKey = async (id: string) => {
    if (!confirm('이 API 키를 삭제하시겠습니까?')) return
    try {
      await fetch(`${BASE}/api/enterprise/keys/${id}`, { method: 'DELETE' })
      await load()
    } catch {
      // ignore
    }
  }

  const copyKey = (key: string, id: string) => {
    navigator.clipboard.writeText(key).catch(() => {})
    setCopiedId(id)
    setTimeout(() => setCopiedId(null), 2000)
  }

  const inputStyle: React.CSSProperties = {
    background: 'var(--glass-bg)',
    border: '1px solid var(--border-default)',
    borderRadius: 6,
    color: 'var(--text-primary)',
    fontSize: 12,
    padding: '6px 10px',
    outline: 'none',
    width: '100%',
  }

  const btnStyle: React.CSSProperties = {
    border: 'none',
    borderRadius: 8,
    fontSize: 12,
    fontWeight: 600,
    cursor: 'pointer',
    padding: '7px 14px',
    display: 'flex',
    alignItems: 'center',
    gap: 5,
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      {/* Header */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <Key size={16} color="var(--accent-primary)" />
          <span style={{ fontSize: 14, fontWeight: 700, color: 'var(--text-primary)' }}>기업 API 관리</span>
        </div>
        <motion.button
          whileTap={{ scale: 0.95 }}
          onClick={() => setShowCreate(!showCreate)}
          style={{ ...btnStyle, background: 'rgba(79,126,247,0.15)', color: 'var(--accent-primary)' }}
        >
          <Plus size={13} /> 새 API 키 생성
        </motion.button>
      </div>

      {/* Create Form */}
      <AnimatePresence>
        {showCreate && (
          <motion.div
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: 'auto' }}
            exit={{ opacity: 0, height: 0 }}
            style={{
              background: 'var(--bg-elevated)',
              border: '1px solid var(--border-subtle)',
              borderRadius: 10,
              padding: 14,
              display: 'flex',
              flexDirection: 'column',
              gap: 10,
              overflow: 'hidden',
            }}
          >
            <p style={{ fontSize: 12, fontWeight: 700, color: 'var(--text-secondary)' }}>새 API 키</p>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr auto', gap: 8 }}>
              <input
                style={inputStyle}
                placeholder="키 이름 (예: my-app)"
                value={newName}
                onChange={e => setNewName(e.target.value)}
                onKeyDown={e => e.key === 'Enter' && createKey()}
              />
              <select
                style={{ ...inputStyle, width: 'auto' }}
                value={newPlan}
                onChange={e => setNewPlan(e.target.value)}
              >
                {plans.map(p => (
                  <option key={p.id} value={p.id}>
                    {p.name} (${p.price}/mo)
                  </option>
                ))}
              </select>
            </div>
            <div style={{ display: 'flex', gap: 8 }}>
              <motion.button
                whileTap={{ scale: 0.97 }}
                onClick={createKey}
                disabled={creating || !newName.trim()}
                style={{
                  ...btnStyle,
                  background: 'var(--accent-primary)',
                  color: '#fff',
                  opacity: creating || !newName.trim() ? 0.6 : 1,
                }}
              >
                {creating ? '생성 중...' : '생성'}
              </motion.button>
              <motion.button
                whileTap={{ scale: 0.97 }}
                onClick={() => setShowCreate(false)}
                style={{ ...btnStyle, background: 'var(--glass-bg)', color: 'var(--text-secondary)', border: '1px solid var(--border-default)' }}
              >
                취소
              </motion.button>
            </div>
          </motion.div>
        )}
      </AnimatePresence>

      {/* Keys List */}
      {loading ? (
        <p style={{ fontSize: 12, color: 'var(--text-muted)', textAlign: 'center', padding: '20px 0' }}>불러오는 중...</p>
      ) : keys.filter(k => k.active).length === 0 ? (
        <div style={{
          background: 'var(--bg-elevated)',
          border: '1px dashed var(--border-subtle)',
          borderRadius: 10,
          padding: '28px 0',
          textAlign: 'center',
        }}>
          <p style={{ fontSize: 12, color: 'var(--text-muted)' }}>API 키가 없습니다. 새로 생성하세요.</p>
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          {keys.filter(k => k.active).map(k => {
            const pct = usagePct(k.used_this_month, k.monthly_limit)
            const color = PLAN_COLORS[k.plan] ?? '#4f7ef7'
            const expanded = expandedId === k.id
            return (
              <motion.div
                key={k.id}
                layout
                style={{
                  background: 'var(--bg-elevated)',
                  border: '1px solid var(--border-subtle)',
                  borderRadius: 10,
                  padding: '12px 14px',
                  display: 'flex',
                  flexDirection: 'column',
                  gap: 8,
                }}
              >
                {/* Row 1: plan badge + name + actions */}
                <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                  <TrendingUp size={14} color={color} style={{ flexShrink: 0 }} />
                  <span style={{
                    fontSize: 10, fontWeight: 700, color, background: `${color}20`,
                    padding: '2px 8px', borderRadius: 20, flexShrink: 0,
                  }}>
                    {PLAN_LABELS[k.plan] ?? k.plan}
                  </span>
                  <span style={{ fontSize: 13, fontWeight: 600, color: 'var(--text-primary)', flex: 1, minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {k.name}
                  </span>
                  <motion.button
                    whileTap={{ scale: 0.9 }}
                    onClick={() => setExpandedId(expanded ? null : k.id)}
                    style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--text-muted)', padding: 4 }}
                  >
                    {expanded ? <ChevronUp size={14} /> : <ChevronDown size={14} />}
                  </motion.button>
                  <motion.button
                    whileTap={{ scale: 0.9 }}
                    onClick={() => revokeKey(k.id)}
                    style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--error)', padding: 4 }}
                  >
                    <Trash2 size={14} />
                  </motion.button>
                </div>

                {/* Row 2: key + copy */}
                <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                  <code style={{ fontSize: 11, color: 'var(--text-secondary)', background: 'var(--glass-bg)', padding: '3px 8px', borderRadius: 5, flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {k.key.slice(0, 20)}…
                  </code>
                  <motion.button
                    whileTap={{ scale: 0.9 }}
                    onClick={() => copyKey(k.key, k.id)}
                    style={{ ...btnStyle, padding: '4px 10px', background: copiedId === k.id ? 'rgba(34,197,94,0.15)' : 'var(--glass-bg)', color: copiedId === k.id ? 'var(--success)' : 'var(--text-secondary)', border: '1px solid var(--border-default)' }}
                  >
                    <Copy size={11} /> {copiedId === k.id ? '복사됨' : '복사'}
                  </motion.button>
                </div>

                {/* Row 3: usage bar */}
                <div>
                  <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
                    <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>
                      이번 달: {k.used_this_month.toLocaleString()} / {k.monthly_limit === -1 ? '무제한' : k.monthly_limit.toLocaleString()} 호출
                    </span>
                    {k.monthly_limit !== -1 && (
                      <span style={{ fontSize: 11, color: pct > 80 ? 'var(--error)' : 'var(--text-muted)' }}>{pct}%</span>
                    )}
                  </div>
                  {k.monthly_limit !== -1 && (
                    <div style={{ height: 4, background: 'var(--border-subtle)', borderRadius: 4, overflow: 'hidden' }}>
                      <motion.div
                        initial={{ width: 0 }}
                        animate={{ width: `${pct}%` }}
                        transition={{ duration: 0.6, ease: 'easeOut' }}
                        style={{ height: '100%', background: pct > 80 ? 'var(--error)' : color, borderRadius: 4 }}
                      />
                    </div>
                  )}
                </div>

                {/* Expanded: last used */}
                <AnimatePresence>
                  {expanded && (
                    <motion.div
                      initial={{ opacity: 0, height: 0 }}
                      animate={{ opacity: 1, height: 'auto' }}
                      exit={{ opacity: 0, height: 0 }}
                      style={{ overflow: 'hidden' }}
                    >
                      <div style={{ borderTop: '1px solid var(--border-subtle)', paddingTop: 8, display: 'flex', flexDirection: 'column', gap: 4 }}>
                        <p style={{ fontSize: 11, color: 'var(--text-muted)' }}>마지막 사용: {formatDate(k.last_used_at)}</p>
                        <p style={{ fontSize: 11, color: 'var(--text-muted)' }}>생성일: {formatDate(k.created_at)}</p>
                        <p style={{ fontSize: 11, color: 'var(--text-muted)' }}>허용 엔드포인트: {k.endpoints?.join(', ') ?? '없음'}</p>
                        <code style={{ fontSize: 10, color: 'var(--text-secondary)', background: 'var(--glass-bg)', padding: '4px 8px', borderRadius: 5, wordBreak: 'break-all', userSelect: 'all' }}>
                          {k.key}
                        </code>
                      </div>
                    </motion.div>
                  )}
                </AnimatePresence>
              </motion.div>
            )
          })}
        </div>
      )}

      {/* Plans */}
      {plans.length > 0 && (
        <div style={{
          background: 'var(--bg-elevated)',
          border: '1px solid var(--border-subtle)',
          borderRadius: 10,
          padding: '12px 14px',
        }}>
          <p style={{ fontSize: 11, fontWeight: 700, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.06em', marginBottom: 10 }}>플랜 업그레이드</p>
          <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
            {plans.map(p => (
              <div key={p.id} style={{
                flex: 1, minWidth: 100,
                background: 'var(--glass-bg)',
                border: `1px solid ${PLAN_COLORS[p.id] ?? '#4f7ef7'}40`,
                borderRadius: 8,
                padding: '8px 10px',
                textAlign: 'center',
              }}>
                <p style={{ fontSize: 11, fontWeight: 700, color: PLAN_COLORS[p.id] ?? '#4f7ef7' }}>{p.name}</p>
                <p style={{ fontSize: 13, fontWeight: 800, color: 'var(--text-primary)', margin: '3px 0' }}>${p.price}<span style={{ fontSize: 10, fontWeight: 400, color: 'var(--text-muted)' }}>/mo</span></p>
                <p style={{ fontSize: 10, color: 'var(--text-muted)' }}>{p.description}</p>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
