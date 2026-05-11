import { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Plus, Trash2, CheckSquare, Square, Edit3 } from 'lucide-react'
import { useAppStore } from '../../stores/appStore'

/* ─── 메모 섹션 ─────────────────────────────────────────── */
function MemoSection() {
  const { memos, addMemo, updateMemo, deleteMemo } = useAppStore()
  const [editingId, setEditingId] = useState<string | null>(null)
  const [newTitle, setNewTitle] = useState('')
  const [newBody, setNewBody] = useState('')
  const [showNew, setShowNew] = useState(false)

  const handleAdd = () => {
    if (!newTitle.trim() && !newBody.trim()) return
    addMemo(newTitle.trim() || '제목 없음', newBody.trim())
    setNewTitle('')
    setNewBody('')
    setShowNew(false)
  }

  return (
    <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 10 }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <span style={{ fontSize: 12, fontWeight: 700, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
          메모 {memos.length}개
        </span>
        <motion.button
          whileTap={{ scale: 0.95 }}
          onClick={() => setShowNew(true)}
          style={{
            padding: '5px 12px',
            borderRadius: 8,
            border: 'none',
            background: 'rgba(79,126,247,0.15)',
            color: 'var(--accent-primary)',
            fontSize: 12,
            fontWeight: 600,
            cursor: 'pointer',
            display: 'flex',
            alignItems: 'center',
            gap: 5,
          }}
        >
          <Plus size={12} /> 새 메모
        </motion.button>
      </div>

      {/* 새 메모 입력 */}
      <AnimatePresence>
        {showNew && (
          <motion.div
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: 'auto' }}
            exit={{ opacity: 0, height: 0 }}
            style={{
              padding: 14,
              borderRadius: 'var(--radius-sm)',
              background: 'var(--bg-elevated)',
              border: '1px solid var(--accent-primary)',
              display: 'flex',
              flexDirection: 'column',
              gap: 8,
            }}
          >
            <input
              autoFocus
              value={newTitle}
              onChange={(e) => setNewTitle(e.target.value)}
              placeholder="제목"
              style={{
                background: 'none',
                border: 'none',
                outline: 'none',
                color: 'var(--text-primary)',
                fontSize: 14,
                fontWeight: 700,
                fontFamily: 'Pretendard, Inter, sans-serif',
              }}
            />
            <textarea
              value={newBody}
              onChange={(e) => setNewBody(e.target.value)}
              placeholder="내용을 입력하세요..."
              rows={3}
              style={{
                background: 'none',
                border: 'none',
                outline: 'none',
                color: 'var(--text-secondary)',
                fontSize: 13,
                resize: 'none',
                fontFamily: 'Pretendard, Inter, sans-serif',
                lineHeight: 1.6,
              }}
            />
            <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
              <button onClick={() => setShowNew(false)} style={{ padding: '5px 12px', borderRadius: 7, border: '1px solid var(--border-default)', background: 'transparent', color: 'var(--text-secondary)', fontSize: 12, cursor: 'pointer' }}>취소</button>
              <button onClick={handleAdd} style={{ padding: '5px 12px', borderRadius: 7, border: 'none', background: 'var(--accent-primary)', color: '#fff', fontSize: 12, fontWeight: 600, cursor: 'pointer' }}>저장</button>
            </div>
          </motion.div>
        )}
      </AnimatePresence>

      {/* 메모 목록 */}
      <AnimatePresence>
        {memos.map((memo) => (
          <motion.div
            key={memo.id}
            layout
            initial={{ opacity: 0, y: 6 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, height: 0 }}
            style={{
              padding: 14,
              borderRadius: 'var(--radius-sm)',
              background: 'var(--bg-elevated)',
              border: '1px solid var(--border-subtle)',
              display: 'flex',
              flexDirection: 'column',
              gap: 6,
              transition: 'border-color 0.1s',
            }}
            onMouseEnter={(e) => { e.currentTarget.style.borderColor = 'var(--border-default)' }}
            onMouseLeave={(e) => { e.currentTarget.style.borderColor = 'var(--border-subtle)' }}
          >
            {editingId === memo.id ? (
              <>
                <input
                  autoFocus
                  defaultValue={memo.title}
                  onBlur={(e) => updateMemo(memo.id, { title: e.target.value })}
                  style={{ background: 'none', border: 'none', outline: 'none', color: 'var(--text-primary)', fontSize: 13, fontWeight: 700, fontFamily: 'Pretendard, Inter, sans-serif' }}
                />
                <textarea
                  defaultValue={memo.body}
                  rows={3}
                  onBlur={(e) => { updateMemo(memo.id, { body: e.target.value }); setEditingId(null) }}
                  style={{ background: 'none', border: 'none', outline: 'none', color: 'var(--text-secondary)', fontSize: 12, resize: 'none', fontFamily: 'Pretendard, Inter, sans-serif', lineHeight: 1.6 }}
                />
              </>
            ) : (
              <>
                <p style={{ fontSize: 13, fontWeight: 700, color: 'var(--text-primary)' }}>{memo.title}</p>
                {memo.body && (
                  <p style={{ fontSize: 12, color: 'var(--text-secondary)', lineHeight: 1.5, whiteSpace: 'pre-wrap', overflow: 'hidden', display: '-webkit-box', WebkitLineClamp: 3, WebkitBoxOrient: 'vertical' } as React.CSSProperties}>
                    {memo.body}
                  </p>
                )}
              </>
            )}
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginTop: 2 }}>
              <span style={{ fontSize: 10, color: 'var(--text-muted)' }}>
                {memo.updatedAt.toLocaleDateString('ko-KR')}
              </span>
              <div style={{ display: 'flex', gap: 6 }}>
                <button onClick={() => setEditingId(memo.id)} style={{ background: 'none', border: 'none', color: 'var(--text-muted)', cursor: 'pointer', padding: 2 }}>
                  <Edit3 size={12} />
                </button>
                <button onClick={() => deleteMemo(memo.id)} style={{ background: 'none', border: 'none', color: 'var(--text-muted)', cursor: 'pointer', padding: 2 }}>
                  <Trash2 size={12} />
                </button>
              </div>
            </div>
          </motion.div>
        ))}
      </AnimatePresence>

      {memos.length === 0 && !showNew && (
        <div style={{ textAlign: 'center', padding: 32, color: 'var(--text-muted)', fontSize: 13 }}>
          메모가 없어요. 새 메모를 작성해보세요.
        </div>
      )}
    </div>
  )
}

/* ─── TODO 섹션 ──────────────────────────────────────────── */
function TodoSection() {
  const { todos, addTodo, toggleTodo, deleteTodo } = useAppStore()
  const [input, setInput] = useState('')

  const handleAdd = () => {
    if (!input.trim()) return
    addTodo(input.trim())
    setInput('')
  }

  const done = todos.filter((t) => t.done)
  const pending = todos.filter((t) => !t.done)

  return (
    <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 10 }}>
      <span style={{ fontSize: 12, fontWeight: 700, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
        할일 — {pending.length}개 남음
      </span>

      {/* 입력 */}
      <div style={{ display: 'flex', gap: 8 }}>
        <input
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && handleAdd()}
          placeholder="새 할일 추가..."
          style={{
            flex: 1,
            padding: '9px 12px',
            borderRadius: 8,
            border: '1px solid var(--border-default)',
            background: 'var(--bg-elevated)',
            color: 'var(--text-primary)',
            fontSize: 13,
            outline: 'none',
            fontFamily: 'Pretendard, Inter, sans-serif',
          }}
        />
        <motion.button
          whileTap={{ scale: 0.95 }}
          onClick={handleAdd}
          style={{
            padding: '9px 14px',
            borderRadius: 8,
            border: 'none',
            background: 'var(--accent-primary)',
            color: '#fff',
            fontSize: 13,
            fontWeight: 600,
            cursor: 'pointer',
            display: 'flex',
            alignItems: 'center',
          }}
        >
          <Plus size={14} />
        </motion.button>
      </div>

      <AnimatePresence>
        {pending.map((todo) => (
          <motion.div key={todo.id} layout initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0, height: 0 }}
            style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '8px 10px', borderRadius: 8, background: 'var(--bg-elevated)', border: '1px solid var(--border-subtle)' }}
          >
            <button onClick={() => toggleTodo(todo.id)} style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--text-muted)', padding: 0, display: 'flex' }}>
              <Square size={16} />
            </button>
            <span style={{ flex: 1, fontSize: 13, color: 'var(--text-primary)' }}>{todo.text}</span>
            <button onClick={() => deleteTodo(todo.id)} style={{ background: 'none', border: 'none', color: 'var(--text-muted)', cursor: 'pointer', padding: 0, display: 'flex' }}>
              <Trash2 size={13} />
            </button>
          </motion.div>
        ))}
      </AnimatePresence>

      {done.length > 0 && (
        <div style={{ marginTop: 8 }}>
          <p style={{ fontSize: 11, color: 'var(--text-muted)', marginBottom: 6, textTransform: 'uppercase', letterSpacing: '0.05em' }}>완료 {done.length}개</p>
          <AnimatePresence>
            {done.map((todo) => (
              <motion.div key={todo.id} layout initial={{ opacity: 0 }} animate={{ opacity: 0.5 }} exit={{ opacity: 0, height: 0 }}
                style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '7px 10px', borderRadius: 8, marginBottom: 4 }}
              >
                <button onClick={() => toggleTodo(todo.id)} style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--success)', padding: 0, display: 'flex' }}>
                  <CheckSquare size={16} />
                </button>
                <span style={{ flex: 1, fontSize: 13, color: 'var(--text-muted)', textDecoration: 'line-through' }}>{todo.text}</span>
                <button onClick={() => deleteTodo(todo.id)} style={{ background: 'none', border: 'none', color: 'var(--text-muted)', cursor: 'pointer', padding: 0, display: 'flex' }}>
                  <Trash2 size={13} />
                </button>
              </motion.div>
            ))}
          </AnimatePresence>
        </div>
      )}
    </div>
  )
}

/* ─── 메인 뷰 ────────────────────────────────────────────── */
export function MemoView() {
  const [tab, setTab] = useState<'memo' | 'todo'>('memo')
  return (
    <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
      {/* 탭 */}
      <div style={{ display: 'flex', borderBottom: '1px solid var(--border-subtle)', padding: '0 16px', flexShrink: 0 }}>
        {(['memo', 'todo'] as const).map((t) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            style={{
              padding: '12px 16px',
              border: 'none',
              background: 'none',
              color: tab === t ? 'var(--accent-primary)' : 'var(--text-secondary)',
              fontWeight: tab === t ? 700 : 400,
              fontSize: 13,
              cursor: 'pointer',
              borderBottom: tab === t ? '2px solid var(--accent-primary)' : '2px solid transparent',
              marginBottom: -1,
            }}
          >
            {t === 'memo' ? '📝 메모' : '✅ 할일'}
          </button>
        ))}
      </div>
      <div style={{ flex: 1, overflowY: 'auto', padding: '16px' }}>
        {tab === 'memo' ? <MemoSection /> : <TodoSection />}
      </div>
    </div>
  )
}
