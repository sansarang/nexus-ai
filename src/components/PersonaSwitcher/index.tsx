import React, { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { useAppStore } from '../../stores/appStore'
import { personaSet } from '../../lib/nexus/backendAPI'

interface PersonaDef {
  id: string
  name: string
  emoji: string
  description: string
  color: string
}

const PERSONAS: PersonaDef[] = [
  { id: 'nexus',    name: 'Nexus 기본',   emoji: '🤖', description: '만능 AI 어시스턴트',          color: '#6366f1' },
  { id: 'expert',   name: '전문가 모드',  emoji: '🧠', description: '심층 분석 · 딥서치 강화',     color: '#f59e0b' },
  { id: 'research', name: '리서치',       emoji: '🔬', description: '경쟁사 분석 · 시장 조사',     color: '#0ea5e9' },
  { id: 'creative', name: '크리에이티브', emoji: '🎨', description: '아이디어 · 콘텐츠 기획',      color: '#ec4899' },
  { id: 'finance',  name: '재무',         emoji: '💰', description: '예산 분석 · 재무 보고서',     color: '#10b981' },
]

interface Props {
  onClose: () => void
}

export function PersonaSwitcher({ onClose }: Props) {
  const { activePersonaId, setActivePersonaId } = useAppStore()
  const [switching, setSwitching] = useState<string | null>(null)

  const handleSelect = async (id: string) => {
    if (id === activePersonaId) { onClose(); return }
    setSwitching(id)
    try {
      await personaSet(id)
    } catch { /* 백엔드 미연결 시 무시 */ }
    setActivePersonaId(id)  // 백엔드 성공 여부와 무관하게 항상 로컬 적용
    setSwitching(null)
    onClose()
  }

  const active = PERSONAS.find(p => p.id === activePersonaId) ?? PERSONAS[0]

  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.9, y: 10 }}
      animate={{ opacity: 1, scale: 1, y: 0 }}
      exit={{ opacity: 0, scale: 0.9, y: 10 }}
      style={{
        position: 'fixed', right: 70, bottom: 200,
        background: 'rgba(15,10,30,0.97)',
        border: '1px solid rgba(255,255,255,0.12)',
        borderRadius: 16, padding: '16px 14px',
        width: 240, zIndex: 9999,
        boxShadow: '0 20px 60px rgba(0,0,0,0.7)',
        backdropFilter: 'blur(20px)',
      }}
    >
      {/* 헤더 */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 12 }}>
        <span style={{ color: 'white', fontWeight: 700, fontSize: 13 }}>AI 모드 선택</span>
        <button onClick={onClose} style={{ background: 'none', border: 'none', color: '#888', cursor: 'pointer', fontSize: 16 }}>✕</button>
      </div>

      {/* 현재 활성 모드 */}
      <div style={{
        background: `linear-gradient(135deg, ${active.color}22, ${active.color}11)`,
        border: `1px solid ${active.color}44`,
        borderRadius: 10, padding: '8px 10px', marginBottom: 10,
        display: 'flex', alignItems: 'center', gap: 8,
      }}>
        <span style={{ fontSize: 18 }}>{active.emoji}</span>
        <div>
          <div style={{ color: 'white', fontSize: 12, fontWeight: 600 }}>{active.name}</div>
          <div style={{ color: '#aaa', fontSize: 10 }}>현재 활성</div>
        </div>
        <div style={{ marginLeft: 'auto', width: 8, height: 8, borderRadius: '50%', background: active.color, boxShadow: `0 0 8px ${active.color}` }} />
      </div>

      {/* 페르소나 목록 */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
        {PERSONAS.map(p => {
          const isActive = p.id === activePersonaId
          const isLoading = switching === p.id
          return (
            <motion.button
              key={p.id}
              whileHover={{ x: 2 }}
              whileTap={{ scale: 0.97 }}
              onClick={() => handleSelect(p.id)}
              disabled={!!switching}
              style={{
                display: 'flex', alignItems: 'center', gap: 10,
                background: isActive ? `${p.color}22` : 'rgba(255,255,255,0.04)',
                border: `1px solid ${isActive ? p.color + '55' : 'rgba(255,255,255,0.08)'}`,
                borderRadius: 10, padding: '9px 10px',
                cursor: switching ? 'not-allowed' : 'pointer',
                width: '100%', textAlign: 'left',
                transition: 'all 0.15s',
              }}
            >
              <span style={{ fontSize: 16, minWidth: 22 }}>{isLoading ? '⏳' : p.emoji}</span>
              <div style={{ flex: 1 }}>
                <div style={{ color: isActive ? p.color : 'white', fontSize: 12, fontWeight: isActive ? 700 : 500 }}>{p.name}</div>
                <div style={{ color: '#777', fontSize: 10, marginTop: 1 }}>{p.description}</div>
              </div>
              {/* 전문가 모드 배지 */}
              {p.id === 'expert' && (
                <span style={{
                  fontSize: 9, background: '#f59e0b22', color: '#f59e0b',
                  border: '1px solid #f59e0b44', borderRadius: 4, padding: '2px 5px', fontWeight: 700,
                }}>딥서치</span>
              )}
              {isActive && <div style={{ width: 6, height: 6, borderRadius: '50%', background: p.color }} />}
            </motion.button>
          )
        })}
      </div>

      {/* 전문가 모드 안내 */}
      {activePersonaId === 'expert' && (
        <motion.div
          initial={{ opacity: 0, height: 0 }}
          animate={{ opacity: 1, height: 'auto' }}
          style={{
            marginTop: 10, padding: '8px 10px',
            background: 'rgba(245,158,11,0.08)', border: '1px solid rgba(245,158,11,0.2)',
            borderRadius: 8, fontSize: 10, color: '#f59e0b', lineHeight: 1.5,
          }}
        >
          🧠 웹 검색 시 학술·기술 자료 우선<br />
          🔍 딥서치 시 10개 이상 소스 분석<br />
          📊 데이터·근거 기반 심층 답변
        </motion.div>
      )}
    </motion.div>
  )
}
