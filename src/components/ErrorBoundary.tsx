import React from 'react'

interface State { hasError: boolean; message: string }

export class ErrorBoundary extends React.Component<React.PropsWithChildren, State> {
  state: State = { hasError: false, message: '' }

  static getDerivedStateFromError(err: unknown): State {
    return { hasError: true, message: err instanceof Error ? err.message : String(err) }
  }

  render() {
    if (!this.state.hasError) return this.props.children
    return (
      <div style={{
        display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center',
        height: '100vh', gap: '12px', fontFamily: 'sans-serif', padding: '24px', textAlign: 'center',
      }}>
        <div style={{ fontSize: '2rem' }}>⚠️</div>
        <h2 style={{ margin: 0, fontSize: '1.1rem' }}>오류가 발생했습니다</h2>
        <p style={{ margin: 0, color: '#888', fontSize: '0.85rem', maxWidth: '320px' }}>{this.state.message}</p>
        <button
          onClick={() => window.location.reload()}
          style={{ marginTop: '8px', padding: '8px 20px', borderRadius: '8px', border: 'none', background: '#6c63ff', color: '#fff', cursor: 'pointer' }}
        >
          새로고침
        </button>
      </div>
    )
  }
}
