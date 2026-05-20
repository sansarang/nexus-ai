import { useState, useEffect, useCallback } from 'react'
import {
  getMarketplacePresets,
  purchasePreset,
  publishPreset,
  getMyPresets,
  getPurchasedPresets,
  deleteMarketplacePreset,
  type MarketPreset,
  type PublishPresetData,
} from '../../lib/nexus/backendAPI'

// ── 카테고리 설정 ─────────────────────────────────────────────

const CATEGORIES = [
  { id: 'all', label: '전체', emoji: '🌐' },
  { id: 'productivity', label: '생산성', emoji: '⚡' },
  { id: 'finance', label: '금융', emoji: '📈' },
  { id: 'legal', label: '법무', emoji: '⚖️' },
  { id: 'medical', label: '의료', emoji: '🏥' },
  { id: 'content', label: '콘텐츠', emoji: '🎬' },
  { id: 'real_estate', label: '부동산', emoji: '🏠' },
]

const CATEGORY_EMOJI: Record<string, string> = {
  productivity: '⚡',
  finance: '📈',
  legal: '⚖️',
  medical: '🏥',
  content: '🎬',
  real_estate: '🏠',
}

const SORT_OPTIONS = [
  { value: 'popular', label: '인기순' },
  { value: 'newest', label: '최신순' },
  { value: 'price_asc', label: '가격낮은순' },
  { value: 'free', label: '무료만' },
]

// ── 별점 렌더링 ───────────────────────────────────────────────

function StarRating({ rating }: { rating: number }) {
  const full = Math.floor(rating)
  const half = rating - full >= 0.5
  return (
    <span style={{ color: '#f59e0b', fontSize: 12 }}>
      {'★'.repeat(full)}
      {half ? '½' : ''}
      {'☆'.repeat(5 - full - (half ? 1 : 0))}
      <span style={{ color: 'var(--text-muted)', marginLeft: 4 }}>{rating.toFixed(1)}</span>
    </span>
  )
}

// ── 프리셋 카드 ───────────────────────────────────────────────

interface PresetCardProps {
  preset: MarketPreset
  onBuy: (preset: MarketPreset) => void
  onUse: (preset: MarketPreset) => void
  onDelete?: (preset: MarketPreset) => void
  showDelete?: boolean
}

function PresetCard({ preset, onBuy, onUse, onDelete, showDelete }: PresetCardProps) {
  const [hover, setHover] = useState(false)

  return (
    <div
      onMouseEnter={() => setHover(true)}
      onMouseLeave={() => setHover(false)}
      style={{
        background: hover ? 'var(--bg-elevated)' : 'var(--glass-bg)',
        border: `1px solid ${hover ? 'var(--accent-primary)' : 'var(--border-default)'}`,
        borderRadius: 12,
        padding: '16px',
        display: 'flex',
        flexDirection: 'column',
        gap: 8,
        transition: 'all 0.2s ease',
        cursor: 'default',
        minHeight: 200,
      }}
    >
      {/* 상단: 이모지 + 이름 */}
      <div style={{ display: 'flex', alignItems: 'flex-start', gap: 10 }}>
        <span style={{ fontSize: 28, flexShrink: 0 }}>{CATEGORY_EMOJI[preset.category] ?? '📦'}</span>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{
            fontWeight: 600,
            fontSize: 13,
            color: 'var(--text-primary)',
            lineHeight: 1.3,
            overflow: 'hidden',
            display: '-webkit-box',
            WebkitLineClamp: 2,
            WebkitBoxOrient: 'vertical',
          }}>
            {preset.name}
          </div>
          <div style={{ color: 'var(--text-muted)', fontSize: 11, marginTop: 2 }}>{preset.author}</div>
        </div>
      </div>

      {/* 설명 */}
      <div style={{
        color: 'var(--text-secondary)',
        fontSize: 12,
        lineHeight: 1.5,
        flex: 1,
        overflow: 'hidden',
        display: '-webkit-box',
        WebkitLineClamp: 2,
        WebkitBoxOrient: 'vertical',
      }}>
        {preset.description}
      </div>

      {/* 태그 */}
      {preset.tags?.length > 0 && (
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4 }}>
          {preset.tags.slice(0, 3).map(tag => (
            <span key={tag} style={{
              padding: '2px 8px',
              borderRadius: 10,
              background: 'var(--bg-surface)',
              color: 'var(--text-muted)',
              fontSize: 10,
              border: '1px solid var(--border-subtle)',
            }}>#{tag}</span>
          ))}
        </div>
      )}

      {/* 평점 + 다운로드 */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <StarRating rating={preset.rating} />
        <span style={{ color: 'var(--text-muted)', fontSize: 11 }}>
          ↓ {preset.downloads.toLocaleString()}
        </span>
      </div>

      {/* 가격 + 버튼 */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginTop: 4 }}>
        <span style={{
          fontWeight: 700,
          fontSize: 14,
          color: preset.is_free ? '#10b981' : 'var(--accent-primary)',
        }}>
          {preset.is_free ? 'FREE' : `$${preset.price.toFixed(2)}`}
        </span>

        <div style={{ display: 'flex', gap: 6 }}>
          {showDelete && onDelete && (
            <button
              onClick={() => onDelete(preset)}
              style={{
                padding: '5px 10px',
                borderRadius: 8,
                border: '1px solid rgba(239,68,68,0.4)',
                background: 'rgba(239,68,68,0.1)',
                color: '#ef4444',
                fontSize: 11,
                cursor: 'pointer',
              }}
            >삭제</button>
          )}
          {preset.is_owned ? (
            <button
              onClick={() => onUse(preset)}
              style={{
                padding: '5px 14px',
                borderRadius: 8,
                border: 'none',
                background: 'linear-gradient(135deg, #10b981, #059669)',
                color: '#fff',
                fontSize: 12,
                fontWeight: 600,
                cursor: 'pointer',
              }}
            >사용하기</button>
          ) : (
            <button
              onClick={() => onBuy(preset)}
              style={{
                padding: '5px 14px',
                borderRadius: 8,
                border: 'none',
                background: 'linear-gradient(135deg, var(--accent-primary), var(--accent-secondary, #6366f1))',
                color: '#fff',
                fontSize: 12,
                fontWeight: 600,
                cursor: 'pointer',
              }}
            >구매하기</button>
          )}
        </div>
      </div>
    </div>
  )
}

// ── 등록 모달 ─────────────────────────────────────────────────

interface PublishModalProps {
  onClose: () => void
  onPublished: () => void
}

function PublishModal({ onClose, onPublished }: PublishModalProps) {
  const [form, setForm] = useState<PublishPresetData>({
    name: '',
    description: '',
    category: 'productivity',
    price: 0,
    steps: [],
    tags: [],
    preview: '',
  })
  const [tagInput, setTagInput] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const handleSubmit = async () => {
    if (!form.name.trim() || !form.description.trim()) {
      setError('이름과 설명을 입력해주세요.')
      return
    }
    setLoading(true)
    try {
      await publishPreset(form)
      onPublished()
      onClose()
    } catch {
      setError('등록에 실패했습니다. 다시 시도해주세요.')
    } finally {
      setLoading(false)
    }
  }

  const addTag = () => {
    if (tagInput.trim() && !form.tags.includes(tagInput.trim())) {
      setForm(f => ({ ...f, tags: [...f.tags, tagInput.trim()] }))
      setTagInput('')
    }
  }

  const inputStyle = {
    width: '100%',
    padding: '10px 12px',
    borderRadius: 8,
    border: '1px solid var(--border-default)',
    background: 'var(--bg-elevated)',
    color: 'var(--text-primary)',
    fontSize: 13,
    outline: 'none',
    boxSizing: 'border-box' as const,
  }

  return (
    <div style={{
      position: 'fixed', inset: 0, zIndex: 1000,
      background: 'rgba(0,0,0,0.6)',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
    }}>
      <div style={{
        background: 'var(--bg-surface)',
        border: '1px solid var(--border-default)',
        borderRadius: 16,
        padding: 24,
        width: 480,
        maxWidth: '90vw',
        maxHeight: '85vh',
        overflowY: 'auto',
        display: 'flex',
        flexDirection: 'column',
        gap: 16,
      }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <h2 style={{ margin: 0, fontSize: 18, color: 'var(--text-primary)' }}>📤 워크플로우 등록</h2>
          <button onClick={onClose} style={{ background: 'none', border: 'none', color: 'var(--text-muted)', fontSize: 20, cursor: 'pointer' }}>✕</button>
        </div>

        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          <div>
            <label style={{ fontSize: 12, color: 'var(--text-secondary)', marginBottom: 4, display: 'block' }}>이름 *</label>
            <input style={inputStyle} value={form.name} onChange={e => setForm(f => ({ ...f, name: e.target.value }))} placeholder="워크플로우 이름" />
          </div>

          <div>
            <label style={{ fontSize: 12, color: 'var(--text-secondary)', marginBottom: 4, display: 'block' }}>설명 *</label>
            <textarea
              style={{ ...inputStyle, minHeight: 80, resize: 'vertical' }}
              value={form.description}
              onChange={e => setForm(f => ({ ...f, description: e.target.value }))}
              placeholder="워크플로우가 하는 일을 설명해주세요"
            />
          </div>

          <div>
            <label style={{ fontSize: 12, color: 'var(--text-secondary)', marginBottom: 4, display: 'block' }}>카테고리</label>
            <select
              style={{ ...inputStyle }}
              value={form.category}
              onChange={e => setForm(f => ({ ...f, category: e.target.value }))}
            >
              {CATEGORIES.filter(c => c.id !== 'all').map(c => (
                <option key={c.id} value={c.id}>{c.emoji} {c.label}</option>
              ))}
            </select>
          </div>

          <div>
            <label style={{ fontSize: 12, color: 'var(--text-secondary)', marginBottom: 4, display: 'block' }}>가격 (USD) — 0이면 무료</label>
            <input
              type="number"
              style={inputStyle}
              value={form.price}
              min={0}
              step={0.01}
              onChange={e => setForm(f => ({ ...f, price: parseFloat(e.target.value) || 0 }))}
              placeholder="0.00"
            />
          </div>

          <div>
            <label style={{ fontSize: 12, color: 'var(--text-secondary)', marginBottom: 4, display: 'block' }}>미리보기 텍스트</label>
            <input style={inputStyle} value={form.preview} onChange={e => setForm(f => ({ ...f, preview: e.target.value }))} placeholder="실행 결과 예시..." />
          </div>

          <div>
            <label style={{ fontSize: 12, color: 'var(--text-secondary)', marginBottom: 4, display: 'block' }}>태그</label>
            <div style={{ display: 'flex', gap: 8 }}>
              <input
                style={{ ...inputStyle, flex: 1 }}
                value={tagInput}
                onChange={e => setTagInput(e.target.value)}
                onKeyDown={e => e.key === 'Enter' && addTag()}
                placeholder="태그 입력 후 Enter"
              />
              <button onClick={addTag} style={{ padding: '8px 14px', borderRadius: 8, border: '1px solid var(--border-default)', background: 'var(--bg-elevated)', color: 'var(--text-primary)', cursor: 'pointer' }}>추가</button>
            </div>
            {form.tags.length > 0 && (
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4, marginTop: 8 }}>
                {form.tags.map(tag => (
                  <span key={tag} style={{ padding: '2px 10px', borderRadius: 10, background: 'var(--accent-primary)', color: '#fff', fontSize: 11, cursor: 'pointer' }} onClick={() => setForm(f => ({ ...f, tags: f.tags.filter(t => t !== tag) }))}>
                    #{tag} ✕
                  </span>
                ))}
              </div>
            )}
          </div>
        </div>

        {error && <div style={{ color: '#ef4444', fontSize: 12 }}>{error}</div>}

        <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
          <button onClick={onClose} style={{ padding: '8px 18px', borderRadius: 8, border: '1px solid var(--border-default)', background: 'none', color: 'var(--text-secondary)', cursor: 'pointer' }}>취소</button>
          <button
            onClick={handleSubmit}
            disabled={loading}
            style={{
              padding: '8px 18px', borderRadius: 8, border: 'none',
              background: 'linear-gradient(135deg, var(--accent-primary), #6366f1)',
              color: '#fff', fontWeight: 600, cursor: loading ? 'not-allowed' : 'pointer',
              opacity: loading ? 0.7 : 1,
            }}
          >
            {loading ? '등록 중...' : '등록하기'}
          </button>
        </div>
      </div>
    </div>
  )
}

// ── 메인 마켓플레이스 컴포넌트 ───────────────────────────────────

interface MarketplaceProps {
  onClose: () => void
}

export function Marketplace({ onClose }: MarketplaceProps) {
  const [tab, setTab] = useState<'browse' | 'purchased' | 'mine'>('browse')
  const [category, setCategory] = useState('all')
  const [search, setSearch] = useState('')
  const [sort, setSort] = useState('popular')
  const [presets, setPresets] = useState<MarketPreset[]>([])
  const [purchasedPresets, setPurchasedPresets] = useState<MarketPreset[]>([])
  const [myPresets, setMyPresets] = useState<MarketPreset[]>([])
  const [loading, setLoading] = useState(false)
  const [toast, setToast] = useState('')
  const [showPublish, setShowPublish] = useState(false)

  const showToast = (msg: string) => {
    setToast(msg)
    setTimeout(() => setToast(''), 3000)
  }

  const fetchBrowse = useCallback(async () => {
    setLoading(true)
    try {
      const data = await getMarketplacePresets({ category: category === 'all' ? undefined : category, search, sort })
      setPresets(data)
    } finally {
      setLoading(false)
    }
  }, [category, search, sort])

  const fetchPurchased = useCallback(async () => {
    setLoading(true)
    try {
      const data = await getPurchasedPresets()
      setPurchasedPresets(data)
    } finally {
      setLoading(false)
    }
  }, [])

  const fetchMine = useCallback(async () => {
    setLoading(true)
    try {
      const data = await getMyPresets()
      setMyPresets(data)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (tab === 'browse') fetchBrowse()
    else if (tab === 'purchased') fetchPurchased()
    else fetchMine()
  }, [tab, fetchBrowse, fetchPurchased, fetchMine])

  const handleBuy = async (preset: MarketPreset) => {
    try {
      const res = await purchasePreset(preset.id)
      if (res.already_owned) {
        showToast('이미 보유한 프리셋입니다.')
        return
      }
      if (res.requires_payment) {
        showToast(`$${res.price?.toFixed(2)} 결제가 필요합니다. Paddle 결제창이 열립니다.`)
        // Paddle checkout would be triggered here
        return
      }
      showToast(`✅ "${preset.name}" 을 무료로 획득했습니다!`)
      fetchBrowse()
    } catch {
      showToast('구매 중 오류가 발생했습니다.')
    }
  }

  const handleUse = (preset: MarketPreset) => {
    showToast(`"${preset.name}" 워크플로우를 시작합니다.`)
    // Dispatch to workflow runner
  }

  const handleDelete = async (preset: MarketPreset) => {
    if (!confirm(`"${preset.name}"을 마켓플레이스에서 내리시겠습니까?`)) return
    try {
      await deleteMarketplacePreset(preset.id)
      showToast('프리셋이 삭제되었습니다.')
      fetchMine()
    } catch {
      showToast('삭제 중 오류가 발생했습니다.')
    }
  }

  const displayedPresets = tab === 'browse' ? presets : tab === 'purchased' ? purchasedPresets : myPresets

  return (
    <div style={{
      position: 'fixed', inset: 0, zIndex: 900,
      background: 'rgba(0,0,0,0.5)',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
    }}>
      <div style={{
        background: 'var(--bg-surface)',
        border: '1px solid var(--border-default)',
        borderRadius: 20,
        width: 860,
        maxWidth: '95vw',
        height: '85vh',
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden',
      }}>
        {/* 헤더 */}
        <div style={{
          padding: '20px 24px 16px',
          borderBottom: '1px solid var(--border-subtle)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          flexShrink: 0,
        }}>
          <div>
            <h1 style={{ margin: 0, fontSize: 20, fontWeight: 700, color: 'var(--text-primary)' }}>
              🛒 워크플로우 마켓플레이스
            </h1>
            <p style={{ margin: '4px 0 0', fontSize: 13, color: 'var(--text-muted)' }}>
              검증된 AI 워크플로우를 검색하고 바로 사용하세요
            </p>
          </div>
          <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
            <button
              onClick={() => setShowPublish(true)}
              style={{
                padding: '8px 16px', borderRadius: 10,
                border: '1px solid var(--accent-primary)',
                background: 'rgba(99,102,241,0.1)',
                color: 'var(--accent-primary)',
                fontSize: 13, fontWeight: 600, cursor: 'pointer',
              }}
            >📤 내 프리셋 등록</button>
            <button onClick={onClose} style={{ background: 'none', border: 'none', color: 'var(--text-muted)', fontSize: 22, cursor: 'pointer', padding: '0 4px' }}>✕</button>
          </div>
        </div>

        {/* 탭 */}
        <div style={{ display: 'flex', gap: 0, borderBottom: '1px solid var(--border-subtle)', flexShrink: 0 }}>
          {([
            { id: 'browse', label: '🌐 전체 탐색' },
            { id: 'purchased', label: '✅ 구매한 프리셋' },
            { id: 'mine', label: '📤 내가 등록한' },
          ] as const).map(t => (
            <button
              key={t.id}
              onClick={() => setTab(t.id)}
              style={{
                padding: '12px 20px',
                background: 'none',
                border: 'none',
                borderBottom: tab === t.id ? '2px solid var(--accent-primary)' : '2px solid transparent',
                color: tab === t.id ? 'var(--accent-primary)' : 'var(--text-secondary)',
                fontWeight: tab === t.id ? 600 : 400,
                fontSize: 13,
                cursor: 'pointer',
                transition: 'all 0.15s',
              }}
            >{t.label}</button>
          ))}
        </div>

        {/* 필터바 (browse 탭만) */}
        {tab === 'browse' && (
          <div style={{ padding: '12px 24px', borderBottom: '1px solid var(--border-subtle)', flexShrink: 0 }}>
            {/* 카테고리 */}
            <div style={{ display: 'flex', gap: 6, overflowX: 'auto', scrollbarWidth: 'none', marginBottom: 10 }}>
              {CATEGORIES.map(c => (
                <button
                  key={c.id}
                  onClick={() => setCategory(c.id)}
                  style={{
                    flexShrink: 0,
                    padding: '5px 14px',
                    borderRadius: 20,
                    border: `1px solid ${category === c.id ? 'var(--accent-primary)' : 'var(--border-default)'}`,
                    background: category === c.id ? 'var(--accent-primary)' : 'var(--glass-bg)',
                    color: category === c.id ? '#fff' : 'var(--text-secondary)',
                    fontSize: 12, fontWeight: category === c.id ? 600 : 400,
                    cursor: 'pointer',
                  }}
                >{c.emoji} {c.label}</button>
              ))}
            </div>
            {/* 검색 + 정렬 */}
            <div style={{ display: 'flex', gap: 8 }}>
              <input
                value={search}
                onChange={e => setSearch(e.target.value)}
                onKeyDown={e => e.key === 'Enter' && fetchBrowse()}
                placeholder="🔍 워크플로우 검색..."
                style={{
                  flex: 1, padding: '8px 14px', borderRadius: 10,
                  border: '1px solid var(--border-default)',
                  background: 'var(--bg-elevated)',
                  color: 'var(--text-primary)', fontSize: 13, outline: 'none',
                }}
              />
              <select
                value={sort}
                onChange={e => setSort(e.target.value)}
                style={{
                  padding: '8px 12px', borderRadius: 10,
                  border: '1px solid var(--border-default)',
                  background: 'var(--bg-elevated)',
                  color: 'var(--text-primary)', fontSize: 13, outline: 'none',
                }}
              >
                {SORT_OPTIONS.map(o => <option key={o.value} value={o.value}>{o.label}</option>)}
              </select>
            </div>
          </div>
        )}

        {/* 그리드 */}
        <div style={{ flex: 1, overflowY: 'auto', padding: 24 }}>
          {loading ? (
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%', color: 'var(--text-muted)' }}>
              로딩 중...
            </div>
          ) : displayedPresets.length === 0 ? (
            <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '100%', color: 'var(--text-muted)', gap: 12 }}>
              <span style={{ fontSize: 48 }}>🔍</span>
              <div style={{ fontSize: 15 }}>
                {tab === 'purchased' ? '구매한 프리셋이 없습니다' : tab === 'mine' ? '등록한 프리셋이 없습니다' : '검색 결과가 없습니다'}
              </div>
              {tab === 'mine' && (
                <button onClick={() => setShowPublish(true)} style={{ padding: '8px 18px', borderRadius: 10, border: 'none', background: 'var(--accent-primary)', color: '#fff', cursor: 'pointer' }}>
                  첫 번째 프리셋 등록하기
                </button>
              )}
            </div>
          ) : (
            <div style={{
              display: 'grid',
              gridTemplateColumns: 'repeat(auto-fill, minmax(240px, 1fr))',
              gap: 16,
            }}>
              {displayedPresets.map(preset => (
                <PresetCard
                  key={preset.id}
                  preset={preset}
                  onBuy={handleBuy}
                  onUse={handleUse}
                  onDelete={handleDelete}
                  showDelete={tab === 'mine'}
                />
              ))}
            </div>
          )}
        </div>
      </div>

      {/* 토스트 */}
      {toast && (
        <div style={{
          position: 'fixed', bottom: 32, left: '50%', transform: 'translateX(-50%)',
          background: 'var(--bg-elevated)',
          border: '1px solid var(--border-default)',
          borderRadius: 12, padding: '12px 24px',
          color: 'var(--text-primary)', fontSize: 14,
          zIndex: 1100, boxShadow: '0 8px 32px rgba(0,0,0,0.3)',
        }}>
          {toast}
        </div>
      )}

      {/* 등록 모달 */}
      {showPublish && (
        <PublishModal
          onClose={() => setShowPublish(false)}
          onPublished={() => { fetchMine(); showToast('✅ 워크플로우가 마켓플레이스에 등록되었습니다!') }}
        />
      )}
    </div>
  )
}
