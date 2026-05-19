import { useAppStore } from '../../stores/appStore'

const FEATURED_ACTIONS: Array<{ label: string; cmd: string; emoji: string }> = [
  { emoji: '🔐', label: 'PC 해킹 점검', cmd: '내 PC 해킹당했어? 보안 점검해줘' },
  { emoji: '🔬', label: '딥서치', cmd: '양자컴퓨터에 대해 깊게 조사해줘' },
  { emoji: '🗺️', label: '복합 질문', cmd: '오늘 날씨도 알려주고 경주에서 대전 가는 버스 시간표 알려줘' },
  { emoji: '⚖️', label: '비교 분석', cmd: '아이폰 vs 갤럭시 비교해줘' },
  { emoji: '▶️', label: '영상 검색', cmd: '요즘 유튜브에서 핫한 AI 영상 찾아줘' },
]

const PERSONA_ACTIONS: Record<string, Array<{ label: string; cmd: string }>> = {
  developer: [
    { label: '코드 리뷰', cmd: '코드 리뷰 해줘' },
    { label: '버그 찾기', cmd: '버그 해결 도와줘' },
    { label: '리팩터링', cmd: '리팩토링 도와줘' },
    { label: 'PR 만들기', cmd: 'PR 만들어줘' },
    { label: '기술 트렌드', cmd: '최신 기술 트렌드 알려줘' },
    { label: '보안 점검', cmd: '보안 검사 해줘' },
    { label: 'Docker', cmd: '도커 설정 도와줘' },
    { label: '성능 분석', cmd: '성능 병목 찾아줘' },
  ],
  marketer: [
    { label: '트렌드 분석', cmd: '트렌드 분석해줘' },
    { label: '콘텐츠 아이디어', cmd: '콘텐츠 아이디어 10개 내줘' },
    { label: '경쟁사 조사', cmd: '경쟁사 분석해줘' },
    { label: '광고 카피', cmd: '광고 문구 만들어줘' },
    { label: 'SNS 게시물', cmd: '인스타 포스팅 만들어줘' },
    { label: '캠페인 기획', cmd: '캠페인 기획해줘' },
    { label: '성과 리포트', cmd: '성과 리포트 만들어줘' },
    { label: '뉴스', cmd: '오늘 마케팅 뉴스 알려줘' },
  ],
  sales: [
    { label: '이메일 초안', cmd: '영업 이메일 작성해줘' },
    { label: '미팅 준비', cmd: '미팅 자료 준비해줘' },
    { label: '제안서', cmd: '제안서 작성해줘' },
    { label: '고객 조사', cmd: '고객사 조사해줘' },
    { label: '협상 전략', cmd: '협상 전략 세워줘' },
    { label: '경쟁사 비교', cmd: '경쟁사 비교해줘' },
    { label: '주간 파이프라인', cmd: '주간 영업 리포트 만들어줘' },
    { label: 'CRM 업데이트', cmd: 'CRM 업데이트 도와줘' },
  ],
  pm: [
    { label: '문서 요약', cmd: '문서 요약해줘' },
    { label: '로드맵 작성', cmd: '로드맵 작성해줘' },
    { label: '스프린트 계획', cmd: '스프린트 계획 세워줘' },
    { label: 'PRD 작성', cmd: 'PRD 작성해줘' },
    { label: '의사결정 정리', cmd: '의사결정 사항 정리해줘' },
    { label: '회의록', cmd: '회의록 작성해줘' },
    { label: '리스크 분석', cmd: '리스크 분석해줘' },
    { label: '주간 리포트', cmd: '주간 PM 리포트 만들어줘' },
  ],
  designer: [
    { label: '레퍼런스 수집', cmd: '디자인 레퍼런스 찾아줘' },
    { label: '트렌드 조사', cmd: '디자인 트렌드 알려줘' },
    { label: '컬러 팔레트', cmd: '컬러 팔레트 추천해줘' },
    { label: '폰트 추천', cmd: '어울리는 폰트 추천해줘' },
    { label: 'SNS 콘텐츠', cmd: 'SNS 콘텐츠 기획해줘' },
    { label: '파일 정리', cmd: '바탕화면 정리해줘' },
    { label: '포트폴리오', cmd: '포트폴리오 구성 도와줘' },
    { label: '피드백 정리', cmd: '디자인 피드백 정리해줘' },
  ],
  freelancer: [
    { label: '견적서 작성', cmd: '견적서 작성해줘' },
    { label: '클라이언트 이메일', cmd: '클라이언트 이메일 써줘' },
    { label: '계약서 검토', cmd: '계약서 검토해줘' },
    { label: '세금 계산', cmd: '세금 계산 도와줘' },
    { label: '시간 추적', cmd: '작업 시간 정리해줘' },
    { label: '포트폴리오', cmd: '포트폴리오 업데이트 도와줘' },
    { label: '업무 자동화', cmd: '반복 업무 자동화해줘' },
    { label: '수입 리포트', cmd: '이번 달 수입 리포트 만들어줘' },
  ],
  default: [
    { label: 'PC 진단', cmd: 'PC 진단해줘' },
    { label: '자동 정리', cmd: '자동 정리해줘' },
    { label: '보안 점검', cmd: '보안 점검해줘' },
    { label: '날씨', cmd: '오늘 날씨 알려줘' },
    { label: '뉴스', cmd: '오늘 주요 뉴스 알려줘' },
    { label: '브리핑', cmd: '아침 브리핑 해줘' },
    { label: '파일 찾기', cmd: '파일 찾아줘' },
    { label: '집중 모드', cmd: '집중 모드 켜줘' },
  ],
}

export function QuickActions({ onSelect, showFeatured = false }: { onSelect: (cmd: string) => void; showFeatured?: boolean }) {
  const { activePersonaId } = useAppStore()
  const actions = PERSONA_ACTIONS[activePersonaId] ?? PERSONA_ACTIONS.default

  if (showFeatured) {
    return (
      <div style={{ padding: '8px 12px' }}>
        <div style={{ fontSize: 11, color: 'var(--text-secondary)', marginBottom: 8, letterSpacing: '0.06em', opacity: 0.6 }}>
          이런 걸 물어볼 수 있어요
        </div>
        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
          {FEATURED_ACTIONS.map(action => (
            <button
              key={action.label}
              onClick={() => onSelect(action.cmd)}
              style={{
                display: 'flex', alignItems: 'center', gap: 6,
                padding: '8px 14px',
                borderRadius: 20,
                border: '1px solid var(--border-default)',
                background: 'var(--glass-bg)',
                color: 'var(--text-primary)',
                fontSize: 13,
                cursor: 'pointer',
                whiteSpace: 'nowrap',
                transition: 'all 0.15s ease',
              }}
              onMouseEnter={e => {
                e.currentTarget.style.background = 'var(--bg-elevated)'
                e.currentTarget.style.borderColor = 'var(--accent-primary)'
              }}
              onMouseLeave={e => {
                e.currentTarget.style.background = 'var(--glass-bg)'
                e.currentTarget.style.borderColor = 'var(--border-default)'
              }}
            >
              <span>{action.emoji}</span>
              <span>{action.label}</span>
            </button>
          ))}
        </div>
      </div>
    )
  }

  return (
    <div style={{ display: 'flex', gap: 6, overflowX: 'auto', padding: '6px 12px', scrollbarWidth: 'none' }}>
      {actions.map(action => (
        <button
          key={action.label}
          onClick={() => onSelect(action.cmd)}
          style={{
            flexShrink: 0,
            padding: '5px 12px',
            borderRadius: 20,
            border: '1px solid var(--border-default)',
            background: 'var(--glass-bg)',
            color: 'var(--text-secondary)',
            fontSize: 12,
            cursor: 'pointer',
            whiteSpace: 'nowrap',
            transition: 'all 0.15s ease',
          }}
          onMouseEnter={e => {
            e.currentTarget.style.background = 'var(--bg-elevated)'
            e.currentTarget.style.color = 'var(--text-primary)'
            e.currentTarget.style.borderColor = 'var(--accent-primary)'
          }}
          onMouseLeave={e => {
            e.currentTarget.style.background = 'var(--glass-bg)'
            e.currentTarget.style.color = 'var(--text-secondary)'
            e.currentTarget.style.borderColor = 'var(--border-default)'
          }}
        >
          {action.label}
        </button>
      ))}
    </div>
  )
}
