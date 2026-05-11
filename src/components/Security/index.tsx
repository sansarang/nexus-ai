import { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Shield, ShieldAlert, ShieldCheck, RefreshCw } from 'lucide-react'

type ThreatLevel = 'safe' | 'warning' | 'danger'

interface ThreatItem {
  id: string
  title: string
  description: string
  level: ThreatLevel
  detected: string
}

const DUMMY_THREATS: ThreatItem[] = [
  { id: '1', title: '의심스러운 시작 프로그램', description: 'Unknown_app.exe 가 부팅 시 자동 실행됩니다', level: 'warning', detected: '2분 전' },
  { id: '2', title: '방화벽 예외 규칙 발견', description: '알 수 없는 포트 8899가 열려있습니다', level: 'warning', detected: '10분 전' },
]

export function SecurityView() {
  const [scanning, setScanning] = useState(false)
  const [scanned, setScanned] = useState(false)
  const [threats, setThreats] = useState<ThreatItem[]>([])
  const [removed, setRemoved] = useState<string[]>([])

  const runScan = async () => {
    setScanning(true)
    setScanned(false)
    setThreats([])
    await new Promise((r) => setTimeout(r, 2200))
    setThreats(DUMMY_THREATS)
    setScanned(true)
    setScanning(false)
  }

  const removeThreat = async (id: string) => {
    setRemoved((prev) => [...prev, id])
    await new Promise((r) => setTimeout(r, 800))
    setThreats((prev) => prev.filter((t) => t.id !== id))
    setRemoved((prev) => prev.filter((r) => r !== id))
  }

  const activeThreatCount = threats.length
  const overallLevel: ThreatLevel =
    threats.some((t) => t.level === 'danger') ? 'danger' :
    threats.some((t) => t.level === 'warning') ? 'warning' : 'safe'

  return (
    <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 20, padding: '20px 24px', overflowY: 'auto' }}>
      <h2 style={{ fontSize: 16, fontWeight: 800, color: 'var(--text-primary)', letterSpacing: '-0.02em' }}>
        🛡️ 해킹 탐지 & 보안
      </h2>

      {/* 전체 상태 카드 */}
      <motion.div
        initial={{ opacity: 0, y: 8 }}
        animate={{ opacity: 1, y: 0 }}
        style={{
          padding: '24px',
          borderRadius: 'var(--radius-lg)',
          background:
            !scanned ? 'var(--bg-elevated)' :
            overallLevel === 'safe' ? 'rgba(34,197,94,0.07)' :
            overallLevel === 'warning' ? 'rgba(245,158,11,0.07)' :
            'rgba(239,68,68,0.07)',
          border: `1px solid ${
            !scanned ? 'var(--border-subtle)' :
            overallLevel === 'safe' ? 'rgba(34,197,94,0.2)' :
            overallLevel === 'warning' ? 'rgba(245,158,11,0.2)' :
            'rgba(239,68,68,0.2)'
          }`,
          display: 'flex',
          alignItems: 'center',
          gap: 20,
        }}
      >
        <div style={{ flexShrink: 0 }}>
          {!scanned ? (
            <Shield size={40} style={{ color: 'var(--text-muted)' }} />
          ) : overallLevel === 'safe' ? (
            <ShieldCheck size={40} style={{ color: 'var(--success)' }} />
          ) : (
            <ShieldAlert size={40} style={{ color: overallLevel === 'danger' ? 'var(--danger)' : 'var(--warning)' }} />
          )}
        </div>
        <div style={{ flex: 1 }}>
          <p style={{
            fontSize: 16,
            fontWeight: 800,
            color: 'var(--text-primary)',
            marginBottom: 4,
          }}>
            {!scanned
              ? '보안 스캔을 실행하세요'
              : overallLevel === 'safe'
              ? '안전합니다 ✅'
              : `위협 ${activeThreatCount}개 발견됨`}
          </p>
          <p style={{ fontSize: 12, color: 'var(--text-secondary)' }}>
            {!scanned
              ? '악성코드, 의심스러운 프로세스, 방화벽 이상 탐지'
              : overallLevel === 'safe'
              ? '알려진 위협이 발견되지 않았어요'
              : '아래 항목을 확인하고 제거하세요'}
          </p>
        </div>
        <motion.button
          whileTap={{ scale: 0.95 }}
          onClick={runScan}
          disabled={scanning}
          style={{
            padding: '10px 20px',
            borderRadius: 10,
            border: 'none',
            background: scanning ? 'rgba(79,126,247,0.1)' : 'linear-gradient(135deg, var(--accent-primary), var(--accent-hover))',
            color: '#fff',
            fontSize: 13,
            fontWeight: 700,
            cursor: scanning ? 'default' : 'pointer',
            display: 'flex',
            alignItems: 'center',
            gap: 6,
            flexShrink: 0,
            boxShadow: scanning ? 'none' : '0 4px 12px var(--accent-glow)',
          }}
        >
          <RefreshCw size={13} className={scanning ? 'spin' : ''} />
          {scanning ? '스캔 중...' : scanned ? '재스캔' : '스캔 시작'}
        </motion.button>
      </motion.div>

      {/* 스캔 진행 애니메이션 */}
      <AnimatePresence>
        {scanning && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            style={{
              display: 'flex',
              flexDirection: 'column',
              gap: 10,
              padding: '20px',
              borderRadius: 'var(--radius-md)',
              background: 'var(--bg-elevated)',
              border: '1px solid var(--border-subtle)',
            }}
          >
            {['시작 프로그램 검사 중...', '프로세스 분석 중...', '네트워크 연결 확인 중...', '방화벽 규칙 검토 중...'].map((msg, i) => (
              <motion.div
                key={i}
                initial={{ opacity: 0, x: -8 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ delay: i * 0.4 }}
                style={{ display: 'flex', alignItems: 'center', gap: 10, fontSize: 13, color: 'var(--text-secondary)' }}
              >
                <span className="spin" style={{ width: 12, height: 12, border: '2px solid var(--border-default)', borderTopColor: 'var(--accent-primary)', borderRadius: '50%', display: 'inline-block', flexShrink: 0 }} />
                {msg}
              </motion.div>
            ))}
          </motion.div>
        )}
      </AnimatePresence>

      {/* 위협 목록 */}
      <AnimatePresence>
        {threats.map((t) => (
          <motion.div
            key={t.id}
            layout
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, height: 0 }}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 14,
              padding: '14px 16px',
              borderRadius: 'var(--radius-sm)',
              background:
                t.level === 'danger' ? 'rgba(239,68,68,0.07)' : 'rgba(245,158,11,0.07)',
              border: `1px solid ${t.level === 'danger' ? 'rgba(239,68,68,0.2)' : 'rgba(245,158,11,0.2)'}`,
            }}
          >
            <ShieldAlert
              size={20}
              style={{ color: t.level === 'danger' ? 'var(--danger)' : 'var(--warning)', flexShrink: 0 }}
            />
            <div style={{ flex: 1, minWidth: 0 }}>
              <p style={{ fontSize: 13, fontWeight: 700, color: 'var(--text-primary)', marginBottom: 2 }}>
                {t.title}
              </p>
              <p style={{ fontSize: 12, color: 'var(--text-secondary)' }}>{t.description}</p>
              <p style={{ fontSize: 11, color: 'var(--text-muted)', marginTop: 2 }}>탐지: {t.detected}</p>
            </div>
            <motion.button
              whileTap={{ scale: 0.95 }}
              onClick={() => removeThreat(t.id)}
              disabled={removed.includes(t.id)}
              style={{
                padding: '6px 14px',
                borderRadius: 8,
                border: 'none',
                background: removed.includes(t.id) ? 'rgba(239,68,68,0.05)' : 'rgba(239,68,68,0.15)',
                color: 'var(--danger)',
                fontSize: 12,
                fontWeight: 700,
                cursor: removed.includes(t.id) ? 'default' : 'pointer',
                display: 'flex',
                alignItems: 'center',
                gap: 5,
                flexShrink: 0,
              }}
            >
              {removed.includes(t.id) ? (
                <>
                  <span className="spin" style={{ width: 10, height: 10, border: '2px solid rgba(239,68,68,0.3)', borderTopColor: 'var(--danger)', borderRadius: '50%', display: 'inline-block' }} />
                  제거 중
                </>
              ) : '제거'}
            </motion.button>
          </motion.div>
        ))}
      </AnimatePresence>

      {/* 보안 팁 */}
      <div
        style={{
          padding: '16px',
          borderRadius: 'var(--radius-md)',
          background: 'var(--bg-elevated)',
          border: '1px solid var(--border-subtle)',
        }}
      >
        <p style={{ fontSize: 12, fontWeight: 700, color: 'var(--text-muted)', marginBottom: 10, textTransform: 'uppercase', letterSpacing: '0.05em' }}>
          💡 보안 팁
        </p>
        {[
          'Windows Defender를 항상 최신 상태로 유지하세요',
          '알 수 없는 이메일 첨부파일을 열지 마세요',
          '정기적으로 비밀번호를 변경하세요',
        ].map((tip, i) => (
          <p key={i} style={{ fontSize: 12, color: 'var(--text-secondary)', marginBottom: 6, paddingLeft: 12, borderLeft: '2px solid var(--border-subtle)' }}>
            {tip}
          </p>
        ))}
      </div>
    </div>
  )
}
