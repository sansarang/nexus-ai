import { motion } from 'framer-motion'
import { useAppStore } from '../../stores/appStore'
import type { StatsData, ScanResult, ScanIssue, DailyReport, CleanResult, RepairResult, BackendErrorCode } from '../../lib/nexus/backendAPI'

/* ── 공통 유틸 ── */
function statusColor(pct: number, reverse = false): string {
  const val = reverse ? 100 - pct : pct
  if (val >= 80) return '#ef4444'
  if (val >= 60) return '#f59e0b'
  return '#22c55e'
}

function tempColor(t: number): string {
  if (t >= 85) return '#ef4444'
  if (t >= 70) return '#f59e0b'
  return '#22c55e'
}

interface GaugeBarProps {
  label: string
  value: number
  max?: number
  unit?: string
  color: string
  icon: string
}

function GaugeBar({ label, value, max = 100, unit = '%', color, icon }: GaugeBarProps) {
  const pct = Math.min((value / max) * 100, 100)
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <span style={{ fontSize: 11, color: 'rgba(255,255,255,0.6)', display: 'flex', gap: 4 }}>
          {icon} {label}
        </span>
        <span style={{ fontSize: 12, fontWeight: 700, color, fontVariantNumeric: 'tabular-nums' }}>
          {value.toFixed(0)}{unit}
        </span>
      </div>
      <div style={{
        height: 6, borderRadius: 3,
        background: 'rgba(255,255,255,0.08)',
        overflow: 'hidden',
      }}>
        <motion.div
          initial={{ width: 0 }}
          animate={{ width: `${pct}%` }}
          transition={{ duration: 0.7, ease: 'easeOut' }}
          style={{ height: '100%', borderRadius: 3, background: color,
            boxShadow: `0 0 6px ${color}88` }}
        />
      </div>
    </div>
  )
}

/* ── PC 상태 카드 ── */
export function PCStatusCard({ data, accentColor }: { data: StatsData; accentColor: string }) {
  const cpuColor  = statusColor(data.cpu)
  const memColor  = statusColor(data.mem)
  const diskColor = statusColor(data.disk)
  const tempColor_ = tempColor(data.cpu_temp)
  const overallScore = Math.round(100 - (data.cpu * 0.3 + data.mem * 0.3 + data.disk * 0.2 + (data.cpu_temp / 100) * 20))

  return (
    <motion.div
      initial={{ opacity: 0, y: 8, scale: 0.96 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      transition={{ duration: 0.25 }}
      style={{
        background: '#0a0c1c',
        border: `1px solid ${accentColor}55`,
        borderLeft: `3px solid ${accentColor}`,
        borderRadius: 14,
        padding: '12px 14px',
        display: 'flex',
        flexDirection: 'column',
        gap: 10,
        boxShadow: `0 6px 28px rgba(0,0,0,0.75), inset 0 1px 0 rgba(255,255,255,0.05)`,
      }}
    >
      {/* 헤더 */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 7 }}>
          <span style={{ fontSize: 16 }}>💻</span>
          <div>
            <div style={{ fontSize: 12, fontWeight: 800, color: 'rgba(255,255,255,0.95)', lineHeight: 1.3 }}>실시간 PC 상태</div>
            <div style={{ fontSize: 9.5, color: 'rgba(255,255,255,0.35)', marginTop: 1 }}>
              {new Date(data.timestamp * 1000).toLocaleTimeString('ko-KR')} 기준
            </div>
          </div>
        </div>
        <div style={{
          display: 'flex', flexDirection: 'column', alignItems: 'center',
          padding: '4px 10px',
          borderRadius: 10,
          background: overallScore >= 80 ? 'rgba(34,197,94,0.15)' : overallScore >= 60 ? 'rgba(245,158,11,0.15)' : 'rgba(239,68,68,0.15)',
          border: `1px solid ${overallScore >= 80 ? '#22c55e' : overallScore >= 60 ? '#f59e0b' : '#ef4444'}66`,
        }}>
          <span style={{
            fontSize: 16, fontWeight: 900, lineHeight: 1,
            color: overallScore >= 80 ? '#22c55e' : overallScore >= 60 ? '#f59e0b' : '#ef4444',
          }}>{overallScore}</span>
          <span style={{
            fontSize: 9, fontWeight: 700,
            color: overallScore >= 80 ? '#22c55e' : overallScore >= 60 ? '#f59e0b' : '#ef4444',
            opacity: 0.8,
          }}>점</span>
        </div>
      </div>
      <div style={{ height: 1, background: `${accentColor}22`, margin: '0 -2px' }} />

      {/* 게이지 바 */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
        <GaugeBar label="CPU" value={data.cpu} color={cpuColor} icon="⚡" />
        <GaugeBar label="메모리" value={data.mem} color={memColor} icon="🧠" />
        <GaugeBar label="디스크" value={data.disk} color={diskColor} icon="💾" />
        <GaugeBar label="CPU 온도" value={data.cpu_temp} max={100} unit="°C" color={tempColor_} icon="🌡️" />
      </div>

      {/* 네트워크 */}
      <div style={{
        display: 'flex',
        justifyContent: 'space-between',
        padding: '6px 8px',
        background: 'rgba(255,255,255,0.04)',
        borderRadius: 8,
        fontSize: 11,
      }}>
        <span style={{ color: '#22c55e' }}>
          ↓ {(data.net_down / 1024).toFixed(1)} MB/s
        </span>
        <span style={{ color: 'rgba(255,255,255,0.3)' }}>|</span>
        <span style={{ color: accentColor }}>
          ↑ {(data.net_up / 1024).toFixed(2)} MB/s
        </span>
      </div>

    </motion.div>
  )
}

/* ── 보안 스캔 카드 ── */
const SEVERITY_META = {
  high:   { color: '#ef4444', bg: '#ef444415', icon: '🔴', label: '심각' },
  medium: { color: '#f59e0b', bg: '#f59e0b15', icon: '🟡', label: '주의' },
  low:    { color: '#22c55e', bg: '#22c55e15', icon: '🟢', label: '낮음' },
}

export function ScanResultCard({
  data,
  accentColor,
  onRepair,
}: {
  data: ScanResult
  accentColor: string
  onRepair?: (ids: string[]) => void
}) {
  const scoreColor = data.score >= 90 ? '#22c55e' : data.score >= 70 ? '#f59e0b' : '#ef4444'
  const fixableIds = data.issues.filter(i => i.fixable).map(i => i.id)

  return (
    <motion.div
      initial={{ opacity: 0, y: 8, scale: 0.96 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      transition={{ duration: 0.25 }}
      style={{
        background: 'rgba(10,12,28,0.96)',
        border: `1px solid ${accentColor}44`,
        borderRadius: 14,
        padding: '12px 14px',
        display: 'flex',
        flexDirection: 'column',
        gap: 10,
        boxShadow: `0 4px 20px rgba(0,0,0,0.4)`,
      }}
    >
      {/* 헤더 + 점수 */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
        <div style={{
          width: 44, height: 44,
          borderRadius: '50%',
          border: `3px solid ${scoreColor}`,
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          flexShrink: 0,
          boxShadow: `0 0 12px ${scoreColor}44`,
        }}>
          <span style={{ fontSize: 14, fontWeight: 900, color: scoreColor }}>
            {data.score}
          </span>
        </div>
        <div>
          <div style={{ fontSize: 12, fontWeight: 800, color: 'rgba(255,255,255,0.9)' }}>
            🔒 보안 & PC 진단 결과
          </div>
          <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.4)', marginTop: 1 }}>
            {data.issues.length === 0
              ? '✅ 모든 항목이 정상입니다'
              : `${data.issues.length}개 항목 발견`}
          </div>
        </div>
      </div>

      {/* 이슈 목록 */}
      {data.issues.length > 0 && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
          {data.issues.map((issue: ScanIssue) => {
            const meta = SEVERITY_META[issue.severity as keyof typeof SEVERITY_META] ?? SEVERITY_META.low
            return (
              <div key={issue.id} style={{
                display: 'flex',
                alignItems: 'flex-start',
                gap: 8,
                padding: '7px 9px',
                borderRadius: 10,
                background: meta.bg,
                border: `1px solid ${meta.color}22`,
              }}>
                <span style={{ fontSize: 12, flexShrink: 0 }}>{meta.icon}</span>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ fontSize: 11, fontWeight: 700, color: meta.color }}>
                    [{meta.label}] {issue.title}
                  </div>
                  <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.45)', marginTop: 2 }}>
                    {issue.description}
                  </div>
                </div>
                {issue.fixable && (
                  <div style={{
                    fontSize: 9, fontWeight: 700,
                    color: '#22c55e', background: '#22c55e15',
                    borderRadius: 4, padding: '2px 5px',
                    border: '1px solid #22c55e22',
                    flexShrink: 0,
                  }}>
                    수리 가능
                  </div>
                )}
              </div>
            )
          })}
        </div>
      )}

      {/* 정상 항목 */}
      {data.issues.length === 0 && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
          {['원격 접속 흔적', 'hosts 파일', '수상한 프로세스', '이상 계정'].map(item => (
            <div key={item} style={{
              display: 'flex', alignItems: 'center', gap: 6,
              padding: '5px 8px',
              borderRadius: 8,
              background: '#22c55e08',
            }}>
              <span style={{ color: '#22c55e', fontSize: 12 }}>✅</span>
              <span style={{ fontSize: 11, color: 'rgba(255,255,255,0.6)' }}>
                {item} 이상 없음
              </span>
            </div>
          ))}
        </div>
      )}

      {/* 액션 버튼 */}
      {fixableIds.length > 0 && onRepair && (
        <div style={{ display: 'flex', gap: 6 }}>
          <motion.button
            whileTap={{ scale: 0.96 }}
            onClick={() => onRepair(fixableIds)}
            style={{
              flex: 1, padding: '7px 0',
              borderRadius: 8, border: 'none',
              background: `linear-gradient(135deg, ${accentColor}, ${accentColor}cc)`,
              color: '#fff', fontSize: 11, fontWeight: 700, cursor: 'pointer',
            }}
          >
            🔧 자동 수리 ({fixableIds.length}개)
          </motion.button>
        </div>
      )}
    </motion.div>
  )
}

/* ── 데일리 리포트 카드 ── */
export function DailyReportCard({ data, accentColor }: { data: DailyReport; accentColor: string }) {
  const { userLang } = useAppStore()
  const isEn = userLang === 'en'
  const scoreColor = data.pc_score >= 90 ? '#22c55e' : data.pc_score >= 70 ? '#f59e0b' : '#ef4444'
  const trendIcon = (t: string) => t === 'up' ? '↑' : t === 'down' ? '↓' : '→'
  const trendColor = (t: string, label: string) => {
    const isGood = label.includes('여유')
    if (t === 'up') return isGood ? '#22c55e' : '#ef4444'
    if (t === 'down') return isGood ? '#ef4444' : '#22c55e'
    return '#f59e0b'
  }

  return (
    <motion.div
      initial={{ opacity: 0, y: 8, scale: 0.96 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      transition={{ duration: 0.25 }}
      style={{
        background: 'rgba(10,12,28,0.96)',
        border: `1px solid ${accentColor}44`,
        borderRadius: 14,
        padding: '12px 14px',
        display: 'flex',
        flexDirection: 'column',
        gap: 10,
        boxShadow: `0 4px 20px rgba(0,0,0,0.4)`,
      }}
    >
      {/* 헤더 */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <span style={{ fontSize: 12, fontWeight: 800, color: 'rgba(255,255,255,0.9)' }}>
          📊 오늘의 PC 리포트
        </span>
        <span style={{ fontSize: 10, color: 'rgba(255,255,255,0.35)' }}>{data.date}</span>
      </div>

      {/* 첫 실행 안내 */}
      {data.first_run && (
        <div style={{
          padding: '7px 10px', borderRadius: 8,
          background: 'rgba(251,211,141,0.07)', border: '1px solid rgba(251,211,141,0.2)',
          fontSize: 11, color: '#fbd38d',
        }}>
          ⏳ 오늘 첫 실행이에요. 10분마다 수집 중 — 하루가 쌓이면 정확한 평균을 보여드릴게요.
        </div>
      )}

      {/* 점수 + 주요 수치 */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr 1fr', gap: 6, opacity: data.first_run ? 0.6 : 1 }}>
        {[
          { label: 'PC 점수', value: `${data.pc_score}점`, color: scoreColor },
          { label: 'CPU 평균', value: `${data.cpu_avg.toFixed(0)}%`, color: statusColor(data.cpu_avg) },
          { label: '메모리', value: `${data.mem_avg.toFixed(0)}%`, color: statusColor(data.mem_avg) },
          { label: '디스크', value: `${data.disk_free_gb.toFixed(0)}GB`, color: '#22c55e' },
        ].map(({ label, value, color }) => (
          <div key={label} style={{
            padding: '6px',
            borderRadius: 8,
            background: `${color}11`,
            border: `1px solid ${color}22`,
            textAlign: 'center',
          }}>
            <div style={{ fontSize: 13, fontWeight: 800, color }}>{value}</div>
            <div style={{ fontSize: 9, color: 'rgba(255,255,255,0.4)', marginTop: 1 }}>{label}</div>
          </div>
        ))}
      </div>

      {/* 예측 트렌드 */}
      <div>
        <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.35)', marginBottom: 5 }}>{isEn ? 'Tomorrow Forecast' : '내일 예측'}</div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
          {data.predictions.map((pred: { label: string; value: number; trend: 'up' | 'down' | 'stable' }) => (
            <div key={pred.label} style={{
              display: 'flex', justifyContent: 'space-between', alignItems: 'center',
              fontSize: 11,
            }}>
              <span style={{ color: 'rgba(255,255,255,0.5)' }}>{pred.label}</span>
              <span style={{ fontWeight: 700, color: trendColor(pred.trend, pred.label) }}>
                {trendIcon(pred.trend)} {pred.value.toFixed(0)}
                {pred.label.includes('여유') ? 'GB' : '%'}
              </span>
            </div>
          ))}
        </div>
      </div>

      {/* 추천 */}
      {data.recommendations.length > 0 && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
          {data.recommendations.map((rec: string, i: number) => (
            <div key={i} style={{
              fontSize: 10, color: 'rgba(255,255,255,0.55)',
              padding: '4px 8px',
              borderRadius: 6,
              background: 'rgba(255,255,255,0.03)',
              borderLeft: `2px solid ${accentColor}66`,
            }}>
              {rec}
            </div>
          ))}
        </div>
      )}
    </motion.div>
  )
}

/* ── 정리 결과 카드 ── */
export function CleanResultCard({ results, accentColor }: { results: CleanResult[] | { freed: number; message: string }; accentColor: string }) {
  const isArray = Array.isArray(results)
  const totalFreed = isArray
    ? (results as CleanResult[]).reduce((s, r) => s + r.freed_bytes, 0)
    : (results as { freed: number }).freed

  const formatBytes = (b: number) => {
    if (b >= 1 << 30) return `${(b / (1 << 30)).toFixed(1)}GB`
    if (b >= 1 << 20) return `${(b / (1 << 20)).toFixed(0)}MB`
    return `${(b / (1 << 10)).toFixed(0)}KB`
  }

  return (
    <motion.div
      initial={{ opacity: 0, y: 8, scale: 0.96 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      transition={{ duration: 0.25 }}
      style={{
        background: 'rgba(10,12,28,0.96)',
        border: `1px solid #22c55e44`,
        borderRadius: 14,
        padding: '12px 14px',
        display: 'flex',
        flexDirection: 'column',
        gap: 8,
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        <span style={{ fontSize: 20 }}>🧹</span>
        <div>
          <div style={{ fontSize: 12, fontWeight: 800, color: '#22c55e' }}>
            정리 완료!
          </div>
          <div style={{ fontSize: 13, fontWeight: 900, color: 'rgba(255,255,255,0.9)' }}>
            {formatBytes(totalFreed)} 확보
          </div>
        </div>
      </div>
      {isArray && (results as CleanResult[]).map(r => (
        <div key={r.item} style={{
          fontSize: 10, color: 'rgba(255,255,255,0.5)',
          display: 'flex', justifyContent: 'space-between',
        }}>
          <span>{r.item}</span>
          <span style={{ color: '#22c55e' }}>{formatBytes(r.freed_bytes)}</span>
        </div>
      ))}
    </motion.div>
  )
}

/* ── 폴더 열기 결과 카드 ── */
export function FolderOpenCard({
  success,
  path,
  message,
  accentColor,
}: {
  success: boolean
  path?: string
  message: string
  accentColor: string
}) {
  const KNOWN: Record<string, string> = {
    Desktop: '🖥 바탕화면', Downloads: '⬇️ 다운로드',
    Documents: '📄 문서', Pictures: '🖼 사진',
    Music: '🎵 음악', Videos: '🎬 비디오·동영상',
  }
  const folderName = path
    ? (KNOWN[path.split(/[\\/]/).pop() ?? ''] ?? `📂 ${path.split(/[\\/]/).pop()}`)
    : '📂 폴더'

  return (
    <motion.div
      initial={{ opacity: 0, y: 8, scale: 0.96 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      style={{
        background: 'rgba(10,12,28,0.96)',
        border: `1px solid ${success ? '#22c55e' : '#ef4444'}44`,
        borderRadius: 14,
        padding: '12px 14px',
        display: 'flex',
        flexDirection: 'column',
        gap: 8,
      }}
    >
      {/* 헤더 */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
        <span style={{ fontSize: 24 }}>{success ? '📂' : '❌'}</span>
        <div>
          <div style={{ fontSize: 13, fontWeight: 800, color: success ? '#22c55e' : '#ef4444' }}>
            {success ? `${folderName} 열림!` : '폴더 열기 실패'}
          </div>
          <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.4)', marginTop: 2 }}>
            {message}
          </div>
        </div>
      </div>

      {/* 경로 표시 */}
      {success && path && (
        <div style={{
          padding: '6px 10px',
          borderRadius: 8,
          background: 'rgba(255,255,255,0.04)',
          border: '1px solid rgba(255,255,255,0.06)',
          fontSize: 11,
          color: 'rgba(255,255,255,0.5)',
          fontFamily: 'monospace',
          wordBreak: 'break-all',
        }}>
          {path}
        </div>
      )}

      {/* 즐겨찾는 폴더 빠른 접근 */}
      {success && (
        <div>
          <div style={{ fontSize: 9, color: 'rgba(255,255,255,0.3)', marginBottom: 5 }}>
            다른 폴더도 열기
          </div>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 5 }}>
            {['바탕화면', '다운로드', '문서', '사진'].map(f => (
              <div key={f} style={{
                padding: '3px 9px',
                borderRadius: 6,
                background: `${accentColor}11`,
                border: `1px solid ${accentColor}33`,
                fontSize: 10,
                color: accentColor,
                cursor: 'pointer',
              }}>
                {f}
              </div>
            ))}
          </div>
        </div>
      )}
    </motion.div>
  )
}

/* ── 수리 결과 카드 ── */
export function RepairResultCard({ data, accentColor }: { data: RepairResult; accentColor: string }) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 8, scale: 0.96 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      style={{
        background: 'rgba(10,12,28,0.96)',
        border: `1px solid ${data.success ? '#22c55e' : '#ef4444'}44`,
        borderRadius: 14,
        padding: '10px 14px',
        display: 'flex',
        alignItems: 'center',
        gap: 10,
      }}
    >
      <span style={{ fontSize: 20 }}>{data.success ? '✅' : '❌'}</span>
      <span style={{ fontSize: 12, color: 'rgba(255,255,255,0.85)' }}>{data.message}</span>
    </motion.div>
  )
}

/* ── 멀티스텝 에이전트 사고 카드 ── */
export function AgentThinkingCard({ steps, accentColor }: { steps: string[]; accentColor: string }) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 6 }}
      animate={{ opacity: 1, y: 0 }}
      style={{
        background: `${accentColor}0a`,
        border: `1px solid ${accentColor}33`,
        borderRadius: 12,
        padding: '10px 12px',
        display: 'flex',
        flexDirection: 'column',
        gap: 5,
      }}
    >
      <div style={{ fontSize: 10, fontWeight: 800, color: accentColor, letterSpacing: '0.06em', marginBottom: 2 }}>
        🤔 분석 중...
      </div>
      {steps.map((step, i) => (
        <motion.div
          key={i}
          initial={{ opacity: 0, x: -8 }}
          animate={{ opacity: 1, x: 0 }}
          transition={{ delay: i * 0.15 }}
          style={{
            fontSize: 11,
            color: 'rgba(255,255,255,0.65)',
            display: 'flex',
            gap: 6,
            alignItems: 'flex-start',
          }}
        >
          <span style={{ color: accentColor, flexShrink: 0 }}>
            {i + 1 < steps.length ? '✓' : '→'}
          </span>
          {step}
        </motion.div>
      ))}
    </motion.div>
  )
}

/* ── 인라인 카드 데이터 타입 ── */
export type InlineCardData =
  | { type: 'pc_status'; data: StatsData }
  | { type: 'scan_result'; data: ScanResult }
  | { type: 'daily_report'; data: DailyReport }
  | { type: 'clean_result'; results: CleanResult[] | { freed: number; message: string } }
  | { type: 'repair_result'; data: RepairResult }
  | { type: 'folder_open'; success: boolean; path?: string; message: string }
  | { type: 'agent_thinking'; steps: string[] }
  | { type: 'preview_confirm'; items: Array<{ title: string; url: string }>; onPreview: (url: string, title: string) => void }
  | { type: 'error'; intent: string; code: BackendErrorCode | 'not_implemented' | 'renderer_missing'; title: string; detail?: string; hint?: string; path?: string }

interface InlineCardRendererProps {
  card: InlineCardData
  accentColor: string
  onRepair?: (ids: string[]) => void
}

export function InlineCardRenderer({ card, accentColor, onRepair }: InlineCardRendererProps) {
  switch (card.type) {
    case 'pc_status':
      return <PCStatusCard data={card.data} accentColor={accentColor} />
    case 'scan_result':
      return <ScanResultCard data={card.data} accentColor={accentColor} onRepair={onRepair} />
    case 'daily_report':
      return <DailyReportCard data={card.data} accentColor={accentColor} />
    case 'clean_result':
      return <CleanResultCard results={card.results} accentColor={accentColor} />
    case 'repair_result':
      return <RepairResultCard data={card.data} accentColor={accentColor} />
    case 'folder_open':
      return (
        <FolderOpenCard
          success={card.success}
          path={card.path}
          message={card.message}
          accentColor={accentColor}
        />
      )
    case 'agent_thinking':
      return <AgentThinkingCard steps={card.steps} accentColor={accentColor} />
    case 'preview_confirm':
      return <PreviewConfirmCard items={card.items} accentColor={accentColor} onPreview={card.onPreview} />
    case 'error':
      return <ErrorCard intent={card.intent} code={card.code} title={card.title} detail={card.detail} hint={card.hint} path={card.path} />
    default: {
      const _exhaustive: never = card
      void _exhaustive
      return <ErrorCard intent="unknown" code="renderer_missing" title="알 수 없는 카드 타입" />
    }
  }
}

/* ── ErrorCard: 백엔드 실패·미구현 등 모든 에러의 통일된 표시 ── */
function ErrorCard({
  intent, code, title, detail, hint, path,
}: { intent: string; code: string; title: string; detail?: string; hint?: string; path?: string }) {
  const lang = (typeof localStorage !== 'undefined' ? localStorage.getItem('nexus-lang') : 'ko') ?? 'ko'
  const ko = lang === 'ko'
  const codeLabels: Record<string, { ko: string; en: string; icon: string; color: string }> = {
    no_backend:       { ko: '백엔드 연결 안 됨',   en: 'Backend not connected',  icon: '🔌', color: '#ef4444' },
    timeout:          { ko: '응답 시간 초과',     en: 'Request timeout',         icon: '⏱️', color: '#f59e0b' },
    no_api_key:       { ko: 'API 키 없음/오류',   en: 'API key missing/invalid', icon: '🔑', color: '#f59e0b' },
    forbidden:        { ko: '권한 부족',          en: 'Permission denied',       icon: '🚫', color: '#f59e0b' },
    not_implemented:  { ko: '아직 준비 중',       en: 'Not implemented yet',     icon: '🚧', color: '#94a3b8' },
    windows_only:     { ko: 'Windows 전용 기능', en: 'Windows-only feature',    icon: '🪟', color: '#3b82f6' },
    rate_limited:     { ko: '호출 한도 초과',     en: 'Rate limited',            icon: '🛑', color: '#f59e0b' },
    server_error:     { ko: '백엔드 내부 오류',   en: 'Backend error',           icon: '💥', color: '#ef4444' },
    bad_request:      { ko: '잘못된 요청',        en: 'Bad request',             icon: '⚠️', color: '#f59e0b' },
    renderer_missing: { ko: '카드 렌더러 누락',   en: 'Card renderer missing',   icon: '🧩', color: '#94a3b8' },
    unknown:          { ko: '알 수 없는 오류',    en: 'Unknown error',           icon: '❓', color: '#ef4444' },
  }
  const meta = codeLabels[code] ?? codeLabels.unknown
  const label = ko ? meta.ko : meta.en

  return (
    <motion.div
      initial={{ opacity: 0, y: 6 }} animate={{ opacity: 1, y: 0 }}
      style={{
        background: 'rgba(255,255,255,0.04)',
        border: `1px solid ${meta.color}55`,
        borderLeft: `3px solid ${meta.color}`,
        borderRadius: 10, padding: '10px 12px', marginTop: 6,
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6 }}>
        <span style={{ fontSize: 16 }}>{meta.icon}</span>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ fontSize: 11, fontWeight: 700, color: meta.color }}>{label}</div>
          <div style={{ fontSize: 12, color: 'rgba(255,255,255,0.85)', overflow: 'hidden', textOverflow: 'ellipsis' }}>
            {title}
          </div>
        </div>
      </div>
      {detail && (
        <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.55)', marginBottom: hint ? 4 : 0, lineHeight: 1.4 }}>
          {detail}
        </div>
      )}
      {hint && (
        <div style={{ fontSize: 10, color: `${meta.color}cc`, lineHeight: 1.4 }}>
          💡 {hint}
        </div>
      )}
      {(intent || path) && (
        <div style={{ fontSize: 9, color: 'rgba(255,255,255,0.25)', marginTop: 6, fontFamily: 'ui-monospace, monospace' }}>
          {intent}{path ? ` · ${path}` : ''}
        </div>
      )}
    </motion.div>
  )
}

function PreviewConfirmCard({
  items, accentColor, onPreview,
}: { items: Array<{ title: string; url: string }>; accentColor: string; onPreview: (url: string, title: string) => void }) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 6 }} animate={{ opacity: 1, y: 0 }}
      style={{ background: 'rgba(255,255,255,0.05)', border: `1px solid ${accentColor}40`, borderRadius: 12, padding: '10px 12px', marginTop: 6 }}
    >
      <div style={{ fontSize: 11, color: accentColor, fontWeight: 700, marginBottom: 8 }}>🔍 미리보기 가능한 페이지</div>
      {items.map((item, i) => (
        <div key={i} style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6 }}>
          <div style={{ flex: 1, fontSize: 11, color: '#e2e8f0', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {item.title}
          </div>
          <button
            onClick={() => onPreview(item.url, item.title)}
            style={{
              background: accentColor, border: 'none', borderRadius: 6, color: '#fff',
              fontSize: 10, fontWeight: 600, padding: '3px 10px', cursor: 'pointer', whiteSpace: 'nowrap',
            }}
          >
            미리보기
          </button>
        </div>
      ))}
    </motion.div>
  )
}
