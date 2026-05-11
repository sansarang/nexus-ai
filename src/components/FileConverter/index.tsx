import { useState, useCallback } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Upload, FileText, CheckCircle, AlertCircle } from 'lucide-react'

type ConversionFormat = 'pdf' | 'docx' | 'xlsx' | 'png' | 'jpg' | 'txt'

const FORMAT_GROUPS: { label: string; formats: ConversionFormat[] }[] = [
  { label: '문서', formats: ['pdf', 'docx', 'txt'] },
  { label: '스프레드시트', formats: ['xlsx'] },
  { label: '이미지', formats: ['png', 'jpg'] },
]

interface ConversionItem {
  id: string
  name: string
  size: string
  status: 'pending' | 'converting' | 'done' | 'error'
  targetFormat: ConversionFormat
}

export function FileConverterView() {
  const [items, setItems] = useState<ConversionItem[]>([])
  const [targetFormat, setTargetFormat] = useState<ConversionFormat>('pdf')
  const [dragging, setDragging] = useState(false)

  const addFile = (file: File) => {
    const id = Math.random().toString(36).slice(2)
    const size =
      file.size >= 1024 * 1024
        ? `${(file.size / (1024 * 1024)).toFixed(1)}MB`
        : `${(file.size / 1024).toFixed(0)}KB`

    setItems((prev) => [...prev, { id, name: file.name, size, status: 'pending', targetFormat }])
  }

  const onDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault()
      setDragging(false)
      Array.from(e.dataTransfer.files).forEach(addFile)
    },
    [targetFormat]
  )

  const simulateConvert = async (id: string) => {
    setItems((prev) => prev.map((i) => (i.id === id ? { ...i, status: 'converting' } : i)))
    await new Promise((r) => setTimeout(r, 1500 + Math.random() * 1000))
    setItems((prev) => prev.map((i) => (i.id === id ? { ...i, status: 'done' } : i)))
  }

  const convertAll = () => {
    items.filter((i) => i.status === 'pending').forEach((i) => simulateConvert(i.id))
  }

  const remove = (id: string) => setItems((prev) => prev.filter((i) => i.id !== id))
  const pendingCount = items.filter((i) => i.status === 'pending').length

  return (
    <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 16, padding: '20px 24px', overflowY: 'auto' }}>
      <h2 style={{ fontSize: 16, fontWeight: 800, color: 'var(--text-primary)', letterSpacing: '-0.02em' }}>
        🔄 파일 변환
      </h2>

      {/* 형식 선택 */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
        <label style={{ fontSize: 11, fontWeight: 700, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
          변환 형식
        </label>
        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
          {FORMAT_GROUPS.flatMap((g) => g.formats).map((fmt) => (
            <button
              key={fmt}
              onClick={() => setTargetFormat(fmt)}
              style={{
                padding: '5px 12px',
                borderRadius: 8,
                border: `1px solid ${targetFormat === fmt ? 'var(--accent-primary)' : 'var(--border-default)'}`,
                background: targetFormat === fmt ? 'rgba(79,126,247,0.15)' : 'var(--glass-bg)',
                color: targetFormat === fmt ? 'var(--accent-primary)' : 'var(--text-secondary)',
                fontSize: 12,
                fontWeight: targetFormat === fmt ? 700 : 400,
                cursor: 'pointer',
                textTransform: 'uppercase',
              }}
            >
              .{fmt}
            </button>
          ))}
        </div>
      </div>

      {/* 드롭존 */}
      <motion.div
        onDragEnter={() => setDragging(true)}
        onDragLeave={() => setDragging(false)}
        onDragOver={(e) => e.preventDefault()}
        onDrop={onDrop}
        animate={{ borderColor: dragging ? 'var(--accent-primary)' : 'var(--border-default)', background: dragging ? 'rgba(79,126,247,0.05)' : 'var(--bg-elevated)' }}
        style={{
          padding: '32px 20px',
          borderRadius: 'var(--radius-md)',
          border: '2px dashed var(--border-default)',
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          gap: 8,
          cursor: 'pointer',
          transition: 'all 0.15s',
        }}
        onClick={() => {
          /* 파일 선택 다이얼로그 시뮬레이션 */
          const input = document.createElement('input')
          input.type = 'file'
          input.multiple = true
          input.onchange = (e) => {
            Array.from((e.target as HTMLInputElement).files ?? []).forEach(addFile)
          }
          input.click()
        }}
      >
        <Upload size={24} style={{ color: dragging ? 'var(--accent-primary)' : 'var(--text-muted)' }} />
        <p style={{ fontSize: 13, fontWeight: 600, color: dragging ? 'var(--accent-primary)' : 'var(--text-secondary)' }}>
          파일을 드래그하거나 클릭해서 선택
        </p>
        <p style={{ fontSize: 11, color: 'var(--text-muted)' }}>
          PDF, Word, Excel, 이미지 등 지원
        </p>
      </motion.div>

      {/* 파일 목록 */}
      {items.length > 0 && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <span style={{ fontSize: 12, fontWeight: 700, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
              파일 {items.length}개
            </span>
            {pendingCount > 0 && (
              <motion.button
                whileTap={{ scale: 0.96 }}
                onClick={convertAll}
                style={{
                  padding: '6px 16px',
                  borderRadius: 8,
                  border: 'none',
                  background: 'linear-gradient(135deg, var(--accent-primary), var(--accent-hover))',
                  color: '#fff',
                  fontSize: 12,
                  fontWeight: 700,
                  cursor: 'pointer',
                }}
              >
                전체 변환 ({pendingCount})
              </motion.button>
            )}
          </div>

          <AnimatePresence>
            {items.map((item) => (
              <motion.div
                key={item.id}
                layout
                initial={{ opacity: 0, y: 6 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, height: 0 }}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 12,
                  padding: '10px 14px',
                  borderRadius: 'var(--radius-sm)',
                  background: 'var(--bg-elevated)',
                  border: '1px solid var(--border-subtle)',
                }}
              >
                <FileText size={16} style={{ color: 'var(--text-muted)', flexShrink: 0 }} />
                <div style={{ flex: 1, minWidth: 0 }}>
                  <p style={{ fontSize: 13, fontWeight: 600, color: 'var(--text-primary)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {item.name}
                  </p>
                  <p style={{ fontSize: 11, color: 'var(--text-muted)' }}>
                    {item.size} → .{item.targetFormat}
                  </p>
                </div>

                {/* 상태 */}
                {item.status === 'pending' && (
                  <motion.button
                    whileTap={{ scale: 0.95 }}
                    onClick={() => simulateConvert(item.id)}
                    style={{
                      padding: '5px 12px',
                      borderRadius: 7,
                      border: 'none',
                      background: 'rgba(79,126,247,0.15)',
                      color: 'var(--accent-primary)',
                      fontSize: 12,
                      fontWeight: 600,
                      cursor: 'pointer',
                    }}
                  >
                    변환
                  </motion.button>
                )}
                {item.status === 'converting' && (
                  <span className="spin" style={{ width: 16, height: 16, border: '2px solid var(--border-default)', borderTopColor: 'var(--accent-primary)', borderRadius: '50%', display: 'inline-block', flexShrink: 0 }} />
                )}
                {item.status === 'done' && (
                  <CheckCircle size={16} style={{ color: 'var(--success)', flexShrink: 0 }} />
                )}
                {item.status === 'error' && (
                  <AlertCircle size={16} style={{ color: 'var(--danger)', flexShrink: 0 }} />
                )}

                <button
                  onClick={() => remove(item.id)}
                  style={{ background: 'none', border: 'none', color: 'var(--text-muted)', cursor: 'pointer', fontSize: 14, lineHeight: 1 }}
                >
                  ✕
                </button>
              </motion.div>
            ))}
          </AnimatePresence>
        </div>
      )}
    </div>
  )
}
