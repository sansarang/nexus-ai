const CATEGORIES = [
  { icon: '🖥️', name: 'PC관리', examples: ['진단', '정리', '보안'] },
  { icon: '⚙️', name: '시스템제어', examples: ['볼륨', '밝기', '전원'] },
  { icon: '📁', name: '파일관리', examples: ['검색', '정리', '변환'] },
  { icon: '🔍', name: '정보조회', examples: ['날씨', '뉴스', '환율'] },
  { icon: '🎯', name: '생산성', examples: ['집중모드', '타이머', '일정'] },
  { icon: '🌐', name: '번역', examples: ['번역', 'URL요약', '클립보드'] },
  { icon: '🛠️', name: '유틸리티', examples: ['QR', '비밀번호', '계산'] },
]

export function HelpCategories() {
  return (
    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(140px, 1fr))', gap: 8, padding: 12 }}>
      {CATEGORIES.map(cat => (
        <div
          key={cat.name}
          style={{
            padding: '12px 14px',
            borderRadius: 10,
            background: 'var(--bg-elevated)',
            border: '1px solid var(--border-subtle)',
          }}
        >
          <div style={{ fontSize: 22, marginBottom: 6 }}>{cat.icon}</div>
          <div style={{ fontSize: 13, fontWeight: 600, color: 'var(--text-primary)', marginBottom: 4 }}>{cat.name}</div>
          <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>{cat.examples.join(' · ')}</div>
        </div>
      ))}
    </div>
  )
}
