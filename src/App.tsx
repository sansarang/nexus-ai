import { lazy, Suspense } from 'react'
import './styles/design-system.css'

// ── FloatingCharacter lazy load ──────────────────────────────────────────────
// 번들 분리 → 초기 렌더 블로킹 없음, 첫 프레임 속도 향상
const FloatingCharacter = lazy(() =>
  import('./components/FloatingCharacter').then(m => ({ default: m.FloatingCharacter }))
)

// 로딩 중 보여줄 최소 화면 (검은 배경 유지)
function AppLoader() {
  return (
    <div style={{
      width: '100vw', height: '100vh',
      background: '#14142a',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
    }}>
      <div style={{
        width: 40, height: 40, borderRadius: '50%',
        border: '3px solid rgba(155,89,182,0.3)',
        borderTopColor: '#9b59b6',
        animation: 'spin 0.8s linear infinite',
      }} />
      <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
    </div>
  )
}

export default function App() {
  return (
    <div
      style={{
        width: '100vw',
        height: '100vh',
        overflow: 'visible',
        position: 'relative',
        background: '#14142a',
      }}
    >
      <Suspense fallback={<AppLoader />}>
        <FloatingCharacter />
      </Suspense>
    </div>
  )
}
