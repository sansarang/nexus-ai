import React, { useState, useRef, useCallback, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { useAppStore } from '../../stores/appStore'

const API = 'http://127.0.0.1:17891'

// ── 타입 ──────────────────────────────────────────────────────────

type NodeType = 'trigger' | 'action' | 'condition' | 'wait' | 'notify' | 'llm'

interface WFNode {
  id: string
  type: NodeType
  label: string
  x: number
  y: number
  config: Record<string, string>
}

interface WFEdge {
  id: string
  from: string
  to: string
}

interface WorkflowBuilderProps {
  onClose: () => void
  primaryColor?: string
}

// ── Node 설정 ────────────────────────────────────────────────────

const NODE_TYPES: { type: NodeType; label: string; icon: string; color: string; desc: string }[] = [
  { type: 'trigger',   label: 'Trigger',   icon: '⚡', color: '#f59e0b', desc: '워크플로우 시작' },
  { type: 'action',    label: 'Action',    icon: '⚙️', color: '#6366f1', desc: 'API 호출/실행' },
  { type: 'condition', label: 'Condition', icon: '❓', color: '#ec4899', desc: '조건 분기' },
  { type: 'wait',      label: 'Wait',      icon: '⏳', color: '#8b5cf6', desc: '대기/지연' },
  { type: 'notify',    label: 'Notify',    icon: '🔔', color: '#22c55e', desc: '알림 전송' },
  { type: 'llm',       label: 'LLM Call',  icon: '🤖', color: '#06b6d4', desc: 'AI 호출' },
]

const NODE_DEFAULTS: Record<NodeType, Record<string, string>> = {
  trigger:   { schedule: '08:00', type: 'schedule' },
  action:    { endpoint: '/api/clean', method: 'POST' },
  condition: { field: 'memory', operator: 'gt', value: '80' },
  wait:      { duration: '60', unit: 'seconds' },
  notify:    { message: '작업 완료!', channel: 'bubble' },
  llm:       { prompt: 'Analyze and summarize:', model: 'sonar' },
}

type TemplateEntry = {
  name: string
  nodes: Array<{ id: string; type: NodeType; label: string; x: number; y: number; config: Record<string, string> }>
  edges: Array<{ id: string; from: string; to: string }>
}

const PERSONA_TEMPLATES: Record<string, TemplateEntry[]> = {
  developer: [
    {
      name: '💻 GitHub 트렌딩 브리핑',
      nodes: [
        { id: 'n1', type: 'trigger' as NodeType, label: '매일 09:00', x: 80, y: 120, config: { schedule: '09:00', type: 'schedule' } },
        { id: 'n2', type: 'llm' as NodeType,     label: 'GitHub 트렌딩 수집', x: 280, y: 120, config: { prompt: 'GitHub 트렌딩 레포, Hacker News 상위 글 요약', model: 'sonar' } },
        { id: 'n3', type: 'notify' as NodeType,  label: '개발자 브리핑 전송', x: 480, y: 120, config: { message: '오늘의 개발자 브리핑!', channel: 'bubble' } },
      ],
      edges: [{ id: 'e1', from: 'n1', to: 'n2' }, { id: 'e2', from: 'n2', to: 'n3' }],
    },
    {
      name: '🔍 코드 리뷰 자동화',
      nodes: [
        { id: 'n1', type: 'trigger' as NodeType,  label: '파일 변경 감지', x: 80, y: 120, config: { type: 'event', value: 'file_change' } },
        { id: 'n2', type: 'llm' as NodeType,      label: 'AI 코드 리뷰', x: 280, y: 120, config: { prompt: '코드 변경사항을 분석하고 버그·개선점을 찾아주세요', model: 'sonar' } },
        { id: 'n3', type: 'notify' as NodeType,   label: '리뷰 결과 전송', x: 480, y: 120, config: { message: '코드 리뷰 완료!', channel: 'bubble' } },
      ],
      edges: [{ id: 'e1', from: 'n1', to: 'n2' }, { id: 'e2', from: 'n2', to: 'n3' }],
    },
  ],
  legal: [
    {
      name: '⚖️ 법무 뉴스 브리핑',
      nodes: [
        { id: 'n1', type: 'trigger' as NodeType, label: '매일 08:30', x: 80, y: 120, config: { schedule: '08:30', type: 'schedule' } },
        { id: 'n2', type: 'llm' as NodeType,     label: '법률·판례 뉴스 수집', x: 280, y: 120, config: { prompt: '오늘의 법률 뉴스, 판례, 법령 개정 사항 요약', model: 'sonar' } },
        { id: 'n3', type: 'notify' as NodeType,  label: '법무 브리핑 전송', x: 480, y: 120, config: { message: '오늘의 법무 브리핑!', channel: 'bubble' } },
      ],
      edges: [{ id: 'e1', from: 'n1', to: 'n2' }, { id: 'e2', from: 'n2', to: 'n3' }],
    },
    {
      name: '📄 계약서 검토 알림',
      nodes: [
        { id: 'n1', type: 'trigger' as NodeType,  label: '새 파일 수신', x: 80, y: 120, config: { type: 'event', value: 'new_file' } },
        { id: 'n2', type: 'llm' as NodeType,      label: 'AI 계약서 분석', x: 280, y: 120, config: { prompt: '계약서의 위험 조항, 불리한 내용, 수정 제안을 분석하세요', model: 'sonar' } },
        { id: 'n3', type: 'notify' as NodeType,   label: '검토 결과 전송', x: 480, y: 120, config: { message: '계약서 검토 완료!', channel: 'bubble' } },
      ],
      edges: [{ id: 'e1', from: 'n1', to: 'n2' }, { id: 'e2', from: 'n2', to: 'n3' }],
    },
  ],
  medical: [
    {
      name: '🏥 의료 뉴스 브리핑',
      nodes: [
        { id: 'n1', type: 'trigger' as NodeType, label: '매일 07:30', x: 80, y: 120, config: { schedule: '07:30', type: 'schedule' } },
        { id: 'n2', type: 'llm' as NodeType,     label: '의료·임상 뉴스 수집', x: 280, y: 120, config: { prompt: '의료 뉴스, 건강보험 급여 변경, 임상 가이드라인 업데이트 요약', model: 'sonar' } },
        { id: 'n3', type: 'notify' as NodeType,  label: '의료 브리핑 전송', x: 480, y: 120, config: { message: '오늘의 의료 브리핑!', channel: 'bubble' } },
      ],
      edges: [{ id: 'e1', from: 'n1', to: 'n2' }, { id: 'e2', from: 'n2', to: 'n3' }],
    },
  ],
  accountant: [
    {
      name: '📊 세무 일정 브리핑',
      nodes: [
        { id: 'n1', type: 'trigger' as NodeType, label: '매일 09:00', x: 80, y: 120, config: { schedule: '09:00', type: 'schedule' } },
        { id: 'n2', type: 'llm' as NodeType,     label: '세무 일정·환율 수집', x: 280, y: 120, config: { prompt: '이번 달 세무 신고 일정, 실시간 환율, 국세청 공지 요약', model: 'sonar' } },
        { id: 'n3', type: 'notify' as NodeType,  label: '세무 브리핑 전송', x: 480, y: 120, config: { message: '오늘의 세무 브리핑!', channel: 'bubble' } },
      ],
      edges: [{ id: 'e1', from: 'n1', to: 'n2' }, { id: 'e2', from: 'n2', to: 'n3' }],
    },
  ],
  creator: [
    {
      name: '🎬 콘텐츠 트렌딩 브리핑',
      nodes: [
        { id: 'n1', type: 'trigger' as NodeType, label: '매일 10:00', x: 80, y: 120, config: { schedule: '10:00', type: 'schedule' } },
        { id: 'n2', type: 'llm' as NodeType,     label: '유튜브·틱톡 트렌딩', x: 280, y: 120, config: { prompt: '유튜브 트렌딩, 틱톡 바이럴, 오늘의 콘텐츠 아이디어 요약', model: 'sonar' } },
        { id: 'n3', type: 'notify' as NodeType,  label: '트렌딩 브리핑 전송', x: 480, y: 120, config: { message: '오늘의 트렌딩!', channel: 'bubble' } },
      ],
      edges: [{ id: 'e1', from: 'n1', to: 'n2' }, { id: 'e2', from: 'n2', to: 'n3' }],
    },
    {
      name: '✍️ 스크립트 자동 생성',
      nodes: [
        { id: 'n1', type: 'trigger' as NodeType,  label: '수동 실행', x: 80, y: 150, config: { type: 'manual', value: '' } },
        { id: 'n2', type: 'llm' as NodeType,      label: '트렌딩 주제 선정', x: 280, y: 80, config: { prompt: '오늘 유튜브 트렌딩 주제 3개 선정', model: 'sonar' } },
        { id: 'n3', type: 'llm' as NodeType,      label: '스크립트 생성', x: 280, y: 220, config: { prompt: '선정된 주제로 훅·본문·아웃트로 스크립트 작성', model: 'sonar' } },
        { id: 'n4', type: 'notify' as NodeType,   label: '스크립트 완성 알림', x: 480, y: 150, config: { message: '스크립트 생성 완료!', channel: 'bubble' } },
      ],
      edges: [
        { id: 'e1', from: 'n1', to: 'n2' }, { id: 'e2', from: 'n1', to: 'n3' },
        { id: 'e3', from: 'n2', to: 'n4' }, { id: 'e4', from: 'n3', to: 'n4' },
      ],
    },
  ],
  realtor: [
    {
      name: '🏠 부동산 시세 브리핑',
      nodes: [
        { id: 'n1', type: 'trigger' as NodeType, label: '매일 09:00', x: 80, y: 120, config: { schedule: '09:00', type: 'schedule' } },
        { id: 'n2', type: 'llm' as NodeType,     label: '부동산·청약 뉴스 수집', x: 280, y: 120, config: { prompt: '부동산 뉴스, 이번 달 청약 일정, 금리 동향 요약', model: 'sonar' } },
        { id: 'n3', type: 'notify' as NodeType,  label: '부동산 브리핑 전송', x: 480, y: 120, config: { message: '오늘의 부동산 브리핑!', channel: 'bubble' } },
      ],
      edges: [{ id: 'e1', from: 'n1', to: 'n2' }, { id: 'e2', from: 'n2', to: 'n3' }],
    },
  ],
  teacher: [
    {
      name: '📚 교육 뉴스 브리핑',
      nodes: [
        { id: 'n1', type: 'trigger' as NodeType, label: '매일 08:00', x: 80, y: 120, config: { schedule: '08:00', type: 'schedule' } },
        { id: 'n2', type: 'llm' as NodeType,     label: '교육부·수능 정보 수집', x: 280, y: 120, config: { prompt: '교육부 공지, 수능 일정, EBS 추천 콘텐츠 요약', model: 'sonar' } },
        { id: 'n3', type: 'notify' as NodeType,  label: '교육 브리핑 전송', x: 480, y: 120, config: { message: '오늘의 교육 브리핑!', channel: 'bubble' } },
      ],
      edges: [{ id: 'e1', from: 'n1', to: 'n2' }, { id: 'e2', from: 'n2', to: 'n3' }],
    },
  ],
  hr: [
    {
      name: '👥 채용 공고 모니터',
      nodes: [
        { id: 'n1', type: 'trigger' as NodeType, label: '매일 09:00', x: 80, y: 120, config: { schedule: '09:00', type: 'schedule' } },
        { id: 'n2', type: 'llm' as NodeType,     label: '채용·HR 뉴스 수집', x: 280, y: 120, config: { prompt: 'HR 뉴스, 최저임금 정보, 워크넷 채용 공고 요약', model: 'sonar' } },
        { id: 'n3', type: 'notify' as NodeType,  label: 'HR 브리핑 전송', x: 480, y: 120, config: { message: '오늘의 HR 브리핑!', channel: 'bubble' } },
      ],
      edges: [{ id: 'e1', from: 'n1', to: 'n2' }, { id: 'e2', from: 'n2', to: 'n3' }],
    },
  ],
  engineer: [
    {
      name: '⚙️ 산업·원자재 브리핑',
      nodes: [
        { id: 'n1', type: 'trigger' as NodeType, label: '매일 09:00', x: 80, y: 120, config: { schedule: '09:00', type: 'schedule' } },
        { id: 'n2', type: 'llm' as NodeType,     label: '산업뉴스·원자재 수집', x: 280, y: 120, config: { prompt: '산업·제조 뉴스, 원자재 시세, KS/ISO 규격 업데이트 요약', model: 'sonar' } },
        { id: 'n3', type: 'notify' as NodeType,  label: '엔지니어 브리핑 전송', x: 480, y: 120, config: { message: '오늘의 엔지니어링 브리핑!', channel: 'bubble' } },
      ],
      edges: [{ id: 'e1', from: 'n1', to: 'n2' }, { id: 'e2', from: 'n2', to: 'n3' }],
    },
  ],
  smallbiz: [
    {
      name: '🏪 소상공인 일일 브리핑',
      nodes: [
        { id: 'n1', type: 'trigger' as NodeType, label: '매일 08:00', x: 80, y: 120, config: { schedule: '08:00', type: 'schedule' } },
        { id: 'n2', type: 'llm' as NodeType,     label: '소상공인 정보 수집', x: 280, y: 120, config: { prompt: '소상공인 지원 정책, 배달앱 트렌드, 카드수수료 정보 요약', model: 'sonar' } },
        { id: 'n3', type: 'notify' as NodeType,  label: '소상공인 브리핑 전송', x: 480, y: 120, config: { message: '오늘의 소상공인 브리핑!', channel: 'bubble' } },
      ],
      edges: [{ id: 'e1', from: 'n1', to: 'n2' }, { id: 'e2', from: 'n2', to: 'n3' }],
    },
  ],
  investor: [
    {
      name: '📈 투자 시장 브리핑',
      nodes: [
        { id: 'n1', type: 'trigger' as NodeType, label: '매일 09:00', x: 80, y: 120, config: { schedule: '09:00', type: 'schedule' } },
        { id: 'n2', type: 'llm' as NodeType,     label: '시장·주가 정보 수집', x: 280, y: 120, config: { prompt: '코스피·나스닥 동향, 주요 종목 주가, 투자 인사이트 요약', model: 'sonar' } },
        { id: 'n3', type: 'notify' as NodeType,  label: '투자 브리핑 전송', x: 480, y: 120, config: { message: '오늘의 투자 브리핑!', channel: 'bubble' } },
      ],
      edges: [{ id: 'e1', from: 'n1', to: 'n2' }, { id: 'e2', from: 'n2', to: 'n3' }],
    },
  ],
  marketer: [
    {
      name: '📣 마케팅 트렌드 브리핑',
      nodes: [
        { id: 'n1', type: 'trigger' as NodeType, label: '매일 09:00', x: 80, y: 120, config: { schedule: '09:00', type: 'schedule' } },
        { id: 'n2', type: 'llm' as NodeType,     label: '마케팅 트렌드 수집', x: 280, y: 120, config: { prompt: 'SNS 트렌드, 경쟁사 동향, 마케팅 인사이트 요약', model: 'sonar' } },
        { id: 'n3', type: 'notify' as NodeType,  label: '마케팅 브리핑 전송', x: 480, y: 120, config: { message: '오늘의 마케팅 브리핑!', channel: 'bubble' } },
      ],
      edges: [{ id: 'e1', from: 'n1', to: 'n2' }, { id: 'e2', from: 'n2', to: 'n3' }],
    },
    {
      name: '📱 SNS 콘텐츠 자동화',
      nodes: [
        { id: 'n1', type: 'trigger' as NodeType, label: '매일 10:00', x: 80, y: 150, config: { schedule: '10:00', type: 'schedule' } },
        { id: 'n2', type: 'llm' as NodeType,     label: '트렌딩 주제 수집', x: 280, y: 80, config: { prompt: '오늘의 SNS 인기 주제 분석', model: 'sonar' } },
        { id: 'n3', type: 'llm' as NodeType,     label: 'SNS 포스트 생성', x: 280, y: 220, config: { prompt: '인스타그램·링크드인용 포스트 3개 작성', model: 'sonar' } },
        { id: 'n4', type: 'notify' as NodeType,  label: '콘텐츠 완성 알림', x: 480, y: 150, config: { message: 'SNS 콘텐츠 준비 완료!', channel: 'bubble' } },
      ],
      edges: [
        { id: 'e1', from: 'n1', to: 'n2' }, { id: 'e2', from: 'n1', to: 'n3' },
        { id: 'e3', from: 'n2', to: 'n4' }, { id: 'e4', from: 'n3', to: 'n4' },
      ],
    },
  ],
  sales: [
    {
      name: '🤝 고객 이메일 자동 분류',
      nodes: [
        { id: 'n1', type: 'trigger' as NodeType,  label: '새 이메일 수신', x: 80, y: 120, config: { type: 'event', value: 'new_email' } },
        { id: 'n2', type: 'llm' as NodeType,      label: 'AI 이메일 분류', x: 280, y: 120, config: { prompt: '영업 이메일을 우선순위별로 분류하고 답장 초안 작성', model: 'sonar' } },
        { id: 'n3', type: 'notify' as NodeType,   label: '분류 결과 알림', x: 480, y: 120, config: { message: '이메일 분류 완료!', channel: 'bubble' } },
      ],
      edges: [{ id: 'e1', from: 'n1', to: 'n2' }, { id: 'e2', from: 'n2', to: 'n3' }],
    },
  ],
  pm: [
    {
      name: '📋 주간 업무 요약',
      nodes: [
        { id: 'n1', type: 'trigger' as NodeType, label: '매주 금요일 17:00', x: 80, y: 120, config: { schedule: '17:00', type: 'schedule' } },
        { id: 'n2', type: 'llm' as NodeType,     label: '주간 업무 분석', x: 280, y: 120, config: { prompt: '이번 주 진행된 작업, 완료 항목, 다음 주 계획 요약', model: 'sonar' } },
        { id: 'n3', type: 'notify' as NodeType,  label: '주간 리포트 전송', x: 480, y: 120, config: { message: '주간 업무 요약 완료!', channel: 'bubble' } },
      ],
      edges: [{ id: 'e1', from: 'n1', to: 'n2' }, { id: 'e2', from: 'n2', to: 'n3' }],
    },
  ],
  designer: [
    {
      name: '🎨 디자인 트렌드 수집',
      nodes: [
        { id: 'n1', type: 'trigger' as NodeType, label: '매주 월요일 09:00', x: 80, y: 120, config: { schedule: '09:00', type: 'schedule' } },
        { id: 'n2', type: 'llm' as NodeType,     label: '디자인 트렌드 수집', x: 280, y: 120, config: { prompt: 'Dribbble, Behance 주간 디자인 트렌드 요약', model: 'sonar' } },
        { id: 'n3', type: 'notify' as NodeType,  label: '트렌드 브리핑 전송', x: 480, y: 120, config: { message: '이번 주 디자인 트렌드!', channel: 'bubble' } },
      ],
      edges: [{ id: 'e1', from: 'n1', to: 'n2' }, { id: 'e2', from: 'n2', to: 'n3' }],
    },
  ],
  freelancer: [
    {
      name: '🚀 프리랜서 업무 관리',
      nodes: [
        { id: 'n1', type: 'trigger' as NodeType, label: '매일 09:00', x: 80, y: 120, config: { schedule: '09:00', type: 'schedule' } },
        { id: 'n2', type: 'llm' as NodeType,     label: '오늘 업무 우선순위', x: 280, y: 120, config: { prompt: '진행 중인 프로젝트, 마감 일정, 오늘 집중할 작업 정리', model: 'sonar' } },
        { id: 'n3', type: 'notify' as NodeType,  label: '업무 계획 전송', x: 480, y: 120, config: { message: '오늘의 업무 계획!', channel: 'bubble' } },
      ],
      edges: [{ id: 'e1', from: 'n1', to: 'n2' }, { id: 'e2', from: 'n2', to: 'n3' }],
    },
  ],
}

const TEMPLATES: TemplateEntry[] = [
  {
    name: '🌅 매일 아침 브리핑',
    nodes: [
      { id: 'n1', type: 'trigger' as NodeType, label: '매일 08:00', x: 80, y: 120, config: { schedule: '08:00', type: 'schedule' } },
      { id: 'n2', type: 'llm' as NodeType,     label: 'AI 브리핑 생성', x: 280, y: 120, config: { prompt: '오늘의 날씨, 뉴스, PC 상태 브리핑', model: 'sonar' } },
      { id: 'n3', type: 'notify' as NodeType,  label: '알림 전송', x: 480, y: 120, config: { message: '아침 브리핑 완료!', channel: 'bubble' } },
    ],
    edges: [{ id: 'e1', from: 'n1', to: 'n2' }, { id: 'e2', from: 'n2', to: 'n3' }],
  },
  {
    name: '🗂️ 파일 자동 정리',
    nodes: [
      { id: 'n1', type: 'trigger' as NodeType,   label: '메모리 > 80%', x: 80, y: 120, config: { type: 'condition', field: 'memory', operator: 'gt', value: '80' } },
      { id: 'n2', type: 'action' as NodeType,    label: '파일 정리', x: 280, y: 120, config: { endpoint: '/api/clean', method: 'POST' } },
      { id: 'n3', type: 'notify' as NodeType,    label: '정리 완료 알림', x: 480, y: 120, config: { message: 'PC 자동 정리 완료!', channel: 'bubble' } },
    ],
    edges: [{ id: 'e1', from: 'n1', to: 'n2' }, { id: 'e2', from: 'n2', to: 'n3' }],
  },
  {
    name: '📧 이메일 자동 답장',
    nodes: [
      { id: 'n1', type: 'trigger' as NodeType,   label: '새 이메일 수신', x: 80, y: 120, config: { type: 'event', value: 'new_email' } },
      { id: 'n2', type: 'llm' as NodeType,       label: 'AI 답장 생성', x: 280, y: 120, config: { prompt: '이메일 내용을 분석하고 적절한 답장을 작성하세요', model: 'sonar' } },
      { id: 'n3', type: 'condition' as NodeType, label: '스팸 여부', x: 280, y: 260, config: { field: 'category', operator: 'eq', value: 'spam' } },
      { id: 'n4', type: 'action' as NodeType,    label: '답장 발송', x: 480, y: 120, config: { endpoint: '/api/imap/send', method: 'POST' } },
    ],
    edges: [{ id: 'e1', from: 'n1', to: 'n2' }, { id: 'e2', from: 'n2', to: 'n3' }, { id: 'e3', from: 'n3', to: 'n4' }],
  },
]

// ── 메인 컴포넌트 ─────────────────────────────────────────────────

export function WorkflowBuilder({ onClose, primaryColor = '#7c3aed' }: WorkflowBuilderProps) {
  const { activePersonaId } = useAppStore()
  const personaTemplates: TemplateEntry[] = PERSONA_TEMPLATES[activePersonaId] ?? []
  const [nodes, setNodes] = useState<WFNode[]>([])
  const [edges, setEdges] = useState<WFEdge[]>([])
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [draggingId, setDraggingId] = useState<string | null>(null)
  const [dragOffset, setDragOffset] = useState({ x: 0, y: 0 })
  const [connectFrom, setConnectFrom] = useState<string | null>(null)
  const [workflowName, setWorkflowName] = useState('새 워크플로우')
  const [savedId, setSavedId] = useState<string | null>(null)
  const [toast, setToast] = useState('')
  const [tab, setTab] = useState<'builder' | 'list'>('builder')
  const [savedWorkflows, setSavedWorkflows] = useState<any[]>([])
  const canvasRef = useRef<HTMLDivElement>(null)

  const showToast = (msg: string) => {
    setToast(msg)
    setTimeout(() => setToast(''), 3000)
  }

  const loadWorkflows = async () => {
    try {
      const res = await fetch(`${API}/api/workflow/list`)
      const data = await res.json()
      setSavedWorkflows(data.workflows || [])
    } catch { /* ignore */ }
  }

  useEffect(() => { loadWorkflows() }, [tab])

  // ── Node 드래그 ───────────────────────────────────────────────

  const handleNodeMouseDown = useCallback((e: React.MouseEvent, id: string) => {
    e.stopPropagation()
    if (connectFrom) {
      // 연결 모드: 엣지 생성
      if (connectFrom !== id) {
        const edgeId = `e_${Date.now()}`
        setEdges(prev => [...prev, { id: edgeId, from: connectFrom, to: id }])
      }
      setConnectFrom(null)
      return
    }
    setSelectedId(id)
    const node = nodes.find(n => n.id === id)
    if (!node) return
    setDraggingId(id)
    setDragOffset({ x: e.clientX - node.x, y: e.clientY - node.y })
  }, [connectFrom, nodes])

  const handleCanvasMouseMove = useCallback((e: React.MouseEvent) => {
    if (!draggingId) return
    const x = e.clientX - dragOffset.x
    const y = e.clientY - dragOffset.y
    setNodes(prev => prev.map(n => n.id === draggingId ? { ...n, x, y } : n))
  }, [draggingId, dragOffset])

  const handleCanvasMouseUp = useCallback(() => {
    setDraggingId(null)
  }, [])

  // ── 노드 추가 ─────────────────────────────────────────────────

  const addNode = (type: NodeType) => {
    const nodeDef = NODE_TYPES.find(t => t.type === type)!
    const id = `n_${Date.now()}`
    const newNode: WFNode = {
      id, type, label: nodeDef.label,
      x: 120 + Math.random() * 200,
      y: 100 + nodes.length * 80,
      config: { ...NODE_DEFAULTS[type] },
    }
    setNodes(prev => [...prev, newNode])
    setSelectedId(id)
  }

  // ── 노드 삭제 ─────────────────────────────────────────────────

  const deleteNode = (id: string) => {
    setNodes(prev => prev.filter(n => n.id !== id))
    setEdges(prev => prev.filter(e => e.from !== id && e.to !== id))
    setSelectedId(null)
  }

  // ── 저장 ──────────────────────────────────────────────────────

  const handleSave = async () => {
    const workflow = {
      id: savedId || undefined,
      name: workflowName,
      description: `${nodes.length}개 노드 워크플로우`,
      enabled: true,
      trigger: { type: 'manual', value: '', label: '수동 실행' },
      actions: nodes.map(n => ({
        id: n.id, type: n.type === 'action' ? 'api_call' : n.type,
        label: n.label,
        endpoint: n.config.endpoint,
        method: n.config.method,
        goal: n.config.prompt,
        params: n.config,
      })),
    }
    try {
      const res = await fetch(`${API}/api/workflow/save`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(workflow),
      })
      const data = await res.json()
      if (data.success) {
        setSavedId(data.id)
        showToast('✅ 저장 완료!')
      }
    } catch { showToast('❌ 저장 실패') }
  }

  const handleRunNow = async () => {
    if (!savedId) { showToast('먼저 저장해주세요'); return }
    try {
      const res = await fetch(`${API}/api/workflow/run-now`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ id: savedId }),
      })
      const data = await res.json()
      showToast(data.success ? '▶ 실행 시작!' : '❌ 실행 실패')
    } catch { showToast('❌ 실행 실패') }
  }

  const applyTemplate = (tpl: TemplateEntry) => {
    setNodes(tpl.nodes.map(n => ({ ...n, config: Object.fromEntries(Object.entries(n.config).filter(([, v]) => v !== undefined)) as Record<string, string> })))
    setEdges(tpl.edges)
    setWorkflowName(tpl.name)
    setSavedId(null)
    showToast(`✅ "${tpl.name}" 템플릿 적용`)
  }

  const selectedNode = nodes.find(n => n.id === selectedId)
  const nodeColor = (type: NodeType) => NODE_TYPES.find(t => t.type === type)?.color || primaryColor

  // SVG 엣지 좌표 계산
  const getEdgePath = (edge: WFEdge) => {
    const from = nodes.find(n => n.id === edge.from)
    const to = nodes.find(n => n.id === edge.to)
    if (!from || !to) return ''
    const x1 = from.x + 80, y1 = from.y + 20
    const x2 = to.x, y2 = to.y + 20
    const mx = (x1 + x2) / 2
    return `M ${x1} ${y1} C ${mx} ${y1} ${mx} ${y2} ${x2} ${y2}`
  }

  return (
    <motion.div
      initial={{ opacity: 0, y: 20, scale: 0.96 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      exit={{ opacity: 0, y: 20, scale: 0.96 }}
      style={{
        position: 'fixed', bottom: 80, right: 90,
        width: 760, height: 560,
        background: 'rgba(6,6,18,0.98)', backdropFilter: 'blur(20px)',
        border: `1px solid ${primaryColor}44`, borderRadius: 20,
        boxShadow: `0 24px 64px rgba(0,0,0,0.7)`,
        zIndex: 10005, display: 'flex', flexDirection: 'column', overflow: 'hidden',
        fontFamily: 'inherit', pointerEvents: 'auto',
      }}
    >
      {/* Header */}
      <div style={{
        padding: '12px 18px', display: 'flex', alignItems: 'center', gap: 12,
        borderBottom: '1px solid rgba(255,255,255,0.07)',
        background: `linear-gradient(135deg, ${primaryColor}18, transparent)`,
        flexShrink: 0,
      }}>
        <span style={{ fontSize: 18 }}>⚡</span>
        <input
          value={workflowName}
          onChange={e => setWorkflowName(e.target.value)}
          style={{
            background: 'transparent', border: 'none', outline: 'none',
            color: '#fff', fontWeight: 700, fontSize: 14, flex: 1,
          }}
        />
        <div style={{ display: 'flex', gap: 6 }}>
          {(['builder', 'list'] as const).map(t => (
            <button key={t} onClick={() => setTab(t)} style={{
              padding: '5px 12px', borderRadius: 8, border: 'none', cursor: 'pointer', fontSize: 11,
              background: tab === t ? `${primaryColor}44` : 'rgba(255,255,255,0.06)',
              color: tab === t ? primaryColor : 'rgba(255,255,255,0.5)', fontWeight: 600,
            }}>
              {t === 'builder' ? '🛠 빌더' : '📋 목록'}
            </button>
          ))}
        </div>
        <button onClick={handleSave} style={{
          padding: '6px 14px', borderRadius: 8, border: 'none',
          background: `${primaryColor}44`, color: primaryColor,
          fontSize: 11, fontWeight: 700, cursor: 'pointer',
        }}>💾 저장</button>
        <button onClick={handleRunNow} style={{
          padding: '6px 14px', borderRadius: 8, border: 'none',
          background: '#22c55e22', color: '#22c55e',
          fontSize: 11, fontWeight: 700, cursor: 'pointer',
        }}>▶ 실행</button>
        <button onClick={() => { setNodes([]); setEdges([]); setSavedId(null) }} style={{
          padding: '6px 12px', borderRadius: 8, border: 'none',
          background: 'rgba(239,68,68,0.1)', color: '#ef4444',
          fontSize: 11, cursor: 'pointer',
        }}>🗑</button>
        <button onClick={onClose} style={{
          background: 'none', border: 'none', color: 'rgba(255,255,255,0.4)',
          cursor: 'pointer', fontSize: 16, marginLeft: 4,
        }}>✕</button>
      </div>

      {tab === 'builder' ? (
        <div style={{ display: 'flex', flex: 1, overflow: 'hidden' }}>
          {/* Left panel: node types + templates */}
          <div style={{
            width: 160, borderRight: '1px solid rgba(255,255,255,0.07)',
            padding: '12px 10px', overflowY: 'auto', flexShrink: 0,
            display: 'flex', flexDirection: 'column', gap: 8,
          }}>
            <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.35)', fontWeight: 600, letterSpacing: '0.08em', marginBottom: 2 }}>노드 추가</div>
            {NODE_TYPES.map(nt => (
              <button key={nt.type} onClick={() => addNode(nt.type)} style={{
                display: 'flex', alignItems: 'center', gap: 7,
                padding: '7px 10px', borderRadius: 10,
                border: `1px solid ${nt.color}33`, background: `${nt.color}11`,
                color: nt.color, fontSize: 11, fontWeight: 600, cursor: 'pointer',
                textAlign: 'left', transition: 'all 0.15s',
              }}>
                <span style={{ fontSize: 14 }}>{nt.icon}</span>
                <span>{nt.label}</span>
              </button>
            ))}
            <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.35)', fontWeight: 600, letterSpacing: '0.08em', marginTop: 8, marginBottom: 2 }}>템플릿</div>
            {personaTemplates.length > 0 && (
              <>
                <div style={{ fontSize: 9, color: `${primaryColor}99`, fontWeight: 600, marginBottom: 3, letterSpacing: '0.06em' }}>내 직업군</div>
                {personaTemplates.map((tpl, i) => (
                  <button key={`p-${i}`} onClick={() => applyTemplate(tpl)} style={{
                    padding: '6px 10px', borderRadius: 8,
                    border: `1px solid ${primaryColor}44`,
                    background: `${primaryColor}11`,
                    color: primaryColor, fontSize: 10, cursor: 'pointer', textAlign: 'left',
                  }}>
                    {tpl.name}
                  </button>
                ))}
                <div style={{ fontSize: 9, color: 'rgba(255,255,255,0.2)', fontWeight: 600, marginTop: 6, marginBottom: 3, letterSpacing: '0.06em' }}>공통</div>
              </>
            )}
            {TEMPLATES.map((tpl, i) => (
              <button key={i} onClick={() => applyTemplate(tpl)} style={{
                padding: '6px 10px', borderRadius: 8,
                border: '1px solid rgba(255,255,255,0.08)',
                background: 'rgba(255,255,255,0.03)',
                color: 'rgba(255,255,255,0.6)', fontSize: 10, cursor: 'pointer', textAlign: 'left',
              }}>
                {tpl.name}
              </button>
            ))}
          </div>

          {/* Canvas */}
          <div
            ref={canvasRef}
            onMouseMove={handleCanvasMouseMove}
            onMouseUp={handleCanvasMouseUp}
            onClick={() => { if (!draggingId) { setSelectedId(null); setConnectFrom(null) } }}
            style={{
              flex: 1, position: 'relative', overflow: 'hidden',
              background: 'radial-gradient(circle at 50% 50%, rgba(124,58,237,0.04) 0%, transparent 70%)',
              cursor: connectFrom ? 'crosshair' : 'default',
            }}
          >
            {/* Grid dots */}
            <svg style={{ position: 'absolute', inset: 0, width: '100%', height: '100%', pointerEvents: 'none' }}>
              <defs>
                <pattern id="grid" width="30" height="30" patternUnits="userSpaceOnUse">
                  <circle cx="1" cy="1" r="0.8" fill="rgba(255,255,255,0.07)" />
                </pattern>
              </defs>
              <rect width="100%" height="100%" fill="url(#grid)" />
              {/* Edges */}
              {edges.map(edge => (
                <path key={edge.id} d={getEdgePath(edge)}
                  stroke={`${primaryColor}88`} strokeWidth={2} fill="none"
                  markerEnd={`url(#arrow_${primaryColor.slice(1)})`}
                />
              ))}
              <defs>
                <marker id={`arrow_${primaryColor.slice(1)}`} markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto">
                  <path d="M0,0 L0,6 L9,3 z" fill={`${primaryColor}88`} />
                </marker>
              </defs>
            </svg>

            {/* Nodes */}
            {nodes.map(node => {
              const color = nodeColor(node.type)
              const nodeDef = NODE_TYPES.find(t => t.type === node.type)!
              const isSelected = selectedId === node.id
              return (
                <div
                  key={node.id}
                  onMouseDown={e => handleNodeMouseDown(e, node.id)}
                  style={{
                    position: 'absolute', left: node.x, top: node.y,
                    width: 160, userSelect: 'none',
                    cursor: draggingId === node.id ? 'grabbing' : 'grab',
                  }}
                >
                  <div style={{
                    background: `rgba(10,10,25,0.95)`,
                    border: `2px solid ${isSelected ? color : color + '55'}`,
                    borderRadius: 12, padding: '8px 12px',
                    boxShadow: isSelected ? `0 0 20px ${color}44` : `0 4px 16px rgba(0,0,0,0.4)`,
                    transition: 'all 0.15s',
                  }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 4 }}>
                      <span style={{ fontSize: 14 }}>{nodeDef.icon}</span>
                      <span style={{ fontSize: 11, fontWeight: 700, color }}>
                        {node.label}
                      </span>
                    </div>
                    <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.4)' }}>
                      {Object.entries(node.config).slice(0, 2).map(([k, v]) => (
                        <div key={k} style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                          {k}: {v}
                        </div>
                      ))}
                    </div>
                    <div style={{ display: 'flex', gap: 4, marginTop: 6 }}>
                      <button
                        onClick={e => { e.stopPropagation(); setConnectFrom(node.id) }}
                        style={{
                          flex: 1, padding: '3px', borderRadius: 6, border: `1px solid ${color}44`,
                          background: `${color}11`, color, fontSize: 9, cursor: 'pointer',
                        }}
                      >{connectFrom === node.id ? '연결 중...' : '연결'}</button>
                      <button
                        onClick={e => { e.stopPropagation(); deleteNode(node.id) }}
                        style={{
                          padding: '3px 6px', borderRadius: 6, border: '1px solid rgba(239,68,68,0.3)',
                          background: 'rgba(239,68,68,0.1)', color: '#ef4444', fontSize: 9, cursor: 'pointer',
                        }}
                      >✕</button>
                    </div>
                  </div>
                </div>
              )
            })}

            {nodes.length === 0 && (
              <div style={{
                position: 'absolute', inset: 0, display: 'flex', flexDirection: 'column',
                alignItems: 'center', justifyContent: 'center', gap: 12,
                color: 'rgba(255,255,255,0.2)', pointerEvents: 'none',
              }}>
                <div style={{ fontSize: 48 }}>⚡</div>
                <div style={{ fontSize: 13 }}>왼쪽에서 노드를 추가하거나 템플릿을 선택하세요</div>
              </div>
            )}
          </div>

          {/* Right panel: properties */}
          <div style={{
            width: 180, borderLeft: '1px solid rgba(255,255,255,0.07)',
            padding: '12px 12px', overflowY: 'auto', flexShrink: 0,
          }}>
            <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.35)', fontWeight: 600, letterSpacing: '0.08em', marginBottom: 8 }}>속성 편집</div>
            {selectedNode ? (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
                <input
                  value={selectedNode.label}
                  onChange={e => setNodes(prev => prev.map(n =>
                    n.id === selectedNode.id ? { ...n, label: e.target.value } : n
                  ))}
                  placeholder="노드 이름"
                  style={{
                    background: 'rgba(255,255,255,0.05)', border: '1px solid rgba(255,255,255,0.1)',
                    borderRadius: 8, padding: '6px 10px', color: '#fff', fontSize: 12,
                    outline: 'none', width: '100%', boxSizing: 'border-box',
                  }}
                />
                {Object.entries(selectedNode.config).map(([key, val]) => (
                  <div key={key}>
                    <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.35)', marginBottom: 2 }}>{key}</div>
                    <input
                      value={val}
                      onChange={e => setNodes(prev => prev.map(n =>
                        n.id === selectedNode.id
                          ? { ...n, config: { ...n.config, [key]: e.target.value } }
                          : n
                      ))}
                      style={{
                        background: 'rgba(255,255,255,0.05)', border: '1px solid rgba(255,255,255,0.1)',
                        borderRadius: 6, padding: '5px 8px', color: '#fff', fontSize: 11,
                        outline: 'none', width: '100%', boxSizing: 'border-box',
                      }}
                    />
                  </div>
                ))}
                <button onClick={() => deleteNode(selectedNode.id)} style={{
                  marginTop: 4, padding: '6px', borderRadius: 8,
                  border: '1px solid rgba(239,68,68,0.4)', background: 'rgba(239,68,68,0.1)',
                  color: '#ef4444', fontSize: 11, cursor: 'pointer', fontWeight: 600,
                }}>🗑 노드 삭제</button>
              </div>
            ) : (
              <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.25)', textAlign: 'center', marginTop: 40 }}>
                노드를 선택하면 속성을 편집할 수 있어요
              </div>
            )}
          </div>
        </div>
      ) : (
        /* Saved workflows list */
        <div style={{ flex: 1, overflowY: 'auto', padding: 18 }}>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
            {savedWorkflows.length === 0 ? (
              <div style={{ textAlign: 'center', color: 'rgba(255,255,255,0.25)', fontSize: 13, marginTop: 60 }}>
                저장된 워크플로우가 없어요
              </div>
            ) : savedWorkflows.map((wf: any) => (
              <div key={wf.id} style={{
                background: 'rgba(255,255,255,0.04)', borderRadius: 12, padding: '12px 16px',
                border: '1px solid rgba(255,255,255,0.08)', display: 'flex', alignItems: 'center', gap: 12,
              }}>
                <div style={{ flex: 1 }}>
                  <div style={{ fontWeight: 600, fontSize: 13, color: '#fff' }}>{wf.name}</div>
                  <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.4)', marginTop: 2 }}>
                    {wf.description} · {wf.run_count || 0}회 실행
                  </div>
                </div>
                <button onClick={async () => {
                  await fetch(`${API}/api/workflow/run-now`, {
                    method: 'POST', headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ id: wf.id }),
                  })
                  showToast('▶ 실행 시작!')
                }} style={{
                  padding: '5px 12px', borderRadius: 8, border: 'none',
                  background: '#22c55e22', color: '#22c55e', fontSize: 11, cursor: 'pointer',
                }}>▶</button>
                <button onClick={async () => {
                  await fetch(`${API}/api/workflow/delete?id=${wf.id}`, { method: 'DELETE' })
                  loadWorkflows()
                }} style={{
                  padding: '5px 10px', borderRadius: 8, border: 'none',
                  background: 'rgba(239,68,68,0.1)', color: '#ef4444', fontSize: 11, cursor: 'pointer',
                }}>🗑</button>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Toast */}
      <AnimatePresence>
        {toast && (
          <motion.div
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0 }}
            style={{
              position: 'absolute', bottom: 16, left: '50%', transform: 'translateX(-50%)',
              background: `${primaryColor}ee`, borderRadius: 20, padding: '8px 18px',
              color: '#fff', fontSize: 12, fontWeight: 600, zIndex: 10,
            }}
          >{toast}</motion.div>
        )}
      </AnimatePresence>
    </motion.div>
  )
}

export default WorkflowBuilder
