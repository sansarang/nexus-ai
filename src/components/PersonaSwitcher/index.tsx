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

// Phase 5+13-D: 백엔드 등록된 19개 페르소나 전체 반영
const PERSONAS: PersonaDef[] = [
  // 기본
  { id: 'nexus',      name: 'Nexus (기본)',          emoji: '🤖', description: 'PC 관리 만능 AI',              color: '#6366f1' },
  // 기본 7종 (전문 분야)
  { id: 'research',   name: '리서치 Nexus',          emoji: '🔬', description: '경쟁·시장·논문 조사',          color: '#0ea5e9' },
  { id: 'finance',    name: '재무 Nexus',            emoji: '💰', description: '예산·투자·재무 보고서',        color: '#10b981' },
  { id: 'meeting',    name: '회의 Nexus',            emoji: '🎯', description: '회의 진행·요약·액션 추적',     color: '#f59e0b' },
  { id: 'creative',   name: '크리에이티브 Nexus',    emoji: '🎨', description: '카피·아이디어·콘텐츠 기획',    color: '#ec4899' },
  { id: 'security',   name: '보안 Nexus',            emoji: '🛡️', description: '사이버 보안·취약점 분석',     color: '#ef4444' },
  { id: 'legal',      name: '법무 Nexus',            emoji: '⚖️', description: '계약서·법률·규정 준수',       color: '#7c3aed' },
  // 직업군 12종 (Phase 5)
  { id: 'developer',  name: '개발자 Nexus',          emoji: '💻', description: '코드 리뷰·디버깅·아키텍처',    color: '#22c55e' },
  { id: 'marketer',   name: '마케터 Nexus',          emoji: '📊', description: '콘텐츠·SNS·캠페인 분석',       color: '#f97316' },
  { id: 'sales',      name: '세일즈 Nexus',          emoji: '🤝', description: '제안서·콜드메일·협상',         color: '#06b6d4' },
  { id: 'pm',         name: 'PM Nexus',              emoji: '📋', description: 'PRD·로드맵·스프린트',          color: '#3b82f6' },
  { id: 'designer',   name: '디자이너 Nexus',        emoji: '🎨', description: 'UI/UX·디자인 시스템',          color: '#ec4899' },
  { id: 'freelancer', name: '프리랜서 Nexus',        emoji: '🚀', description: '견적·계약·세금·일정',          color: '#8b5cf6' },
  { id: 'smallbiz',   name: '소상공인 Nexus',        emoji: '🏪', description: '배달앱·정부지원·매장 운영',    color: '#f59e0b' },
  { id: 'corporate',  name: '법인 Nexus',            emoji: '🏢', description: '법인세·4대보험·인사관리',      color: '#0891b2' },
  { id: 'medical',    name: '의료 Nexus',            emoji: '🩺', description: '임상·약물·논문 검색',          color: '#dc2626' },
  { id: 'creator',    name: '크리에이터 Nexus',      emoji: '🎬', description: '유튜브·썸네일·편집',           color: '#e11d48' },
  { id: 'investor',   name: '투자자 Nexus',          emoji: '📈', description: '주식·ETF·재무제표 분석',       color: '#16a34a' },
  { id: 'tutor',      name: '튜터 Nexus',            emoji: '📚', description: '학습 설계·문제 풀이',          color: '#7c3aed' },
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
