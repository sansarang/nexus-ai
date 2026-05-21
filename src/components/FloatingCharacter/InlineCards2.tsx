/**
 * InlineCards2.tsx — 보안·시스템제어·고급 기능 카드
 */
import { motion, AnimatePresence } from 'framer-motion'
import type {
  RemoteAccessResult, ProcessSecurityResult, DefenderStatus,
  StartupItem, ProcItem, NetworkAdapter, DriverItem, ProgramItem,
  FileResult, DupGroup, NoteItem, PriceItem,
} from '../../lib/nexus/backendAPI'

/* ─────────────────────────────────────────────────────────── */
/* 공통 유틸                                                    */
/* ─────────────────────────────────────────────────────────── */

function ScoreCircle({ score, size = 52 }: { score: number; size?: number }) {
  const color = score >= 80 ? '#22c55e' : score >= 60 ? '#f59e0b' : '#ef4444'
  return (
    <div style={{
      width: size, height: size, borderRadius: '50%',
      border: `3px solid ${color}`, display: 'flex',
      alignItems: 'center', justifyContent: 'center', flexShrink: 0,
    }}>
      <span style={{ fontSize: size * 0.28, fontWeight: 900, color }}>{score}</span>
    </div>
  )
}

function RiskBadge({ risk }: { risk: string }) {
  const c = risk === 'high' ? '#ef4444' : risk === 'medium' ? '#f59e0b' : '#22c55e'
  const label = risk === 'high' ? '위험' : risk === 'medium' ? '주의' : '안전'
  return (
    <span style={{ padding: '1px 6px', borderRadius: 4, background: `${c}22`, color: c, fontSize: 10, fontWeight: 700 }}>
      {label}
    </span>
  )
}

function CardWrap({ children, accent }: { children: React.ReactNode; accent: string }) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 8, scale: 0.96 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      style={{
        background: 'rgba(10,12,28,0.97)', border: `1px solid ${accent}33`,
        borderRadius: 14, padding: '12px 14px', display: 'flex',
        flexDirection: 'column', gap: 8, width: '100%',
      }}
    >
      {children}
    </motion.div>
  )
}

function SectionTitle({ icon, title, accentColor }: { icon: string; title: string; accentColor: string }) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 2 }}>
      <span>{icon}</span>
      <span style={{ fontSize: 12, fontWeight: 800, color: accentColor }}>{title}</span>
    </div>
  )
}

/* ─────────────────────────────────────────────────────────── */
/* 원격 접속 탐지 카드                                          */
/* ─────────────────────────────────────────────────────────── */

export function RemoteAccessCard({ data, accentColor }: { data: RemoteAccessResult; accentColor: string }) {
  return (
    <CardWrap accent={accentColor}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
        <ScoreCircle score={data.score} />
        <div>
          <div style={{ fontSize: 13, fontWeight: 800, color: data.found ? '#f59e0b' : '#22c55e' }}>
            {data.found ? '⚠️ 원격 접속 도구 감지됨' : '✅ 원격 접속 정상'}
          </div>
          <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.4)', marginTop: 2 }}>
            {data.rdp_open ? '🔓 RDP 포트 3389 열려있음' : '🔒 RDP 포트 닫힘'}
          </div>
        </div>
      </div>

      <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
        {data.tools?.filter(t => t.status === 'running').map((t, i) => (
          <div key={i} style={{
            display: 'flex', justifyContent: 'space-between', alignItems: 'center',
            padding: '5px 8px', borderRadius: 7,
            background: 'rgba(255,255,255,0.04)',
          }}>
            <span style={{ fontSize: 11, color: 'rgba(255,255,255,0.8)' }}>🔴 {t.name}</span>
            <RiskBadge risk={t.risk} />
          </div>
        ))}
        {!data.found && (
          <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.4)', textAlign: 'center', padding: '4px 0' }}>
            실행 중인 원격 접속 도구 없음
          </div>
        )}
      </div>
    </CardWrap>
  )
}

/* ─────────────────────────────────────────────────────────── */
/* 수상한 프로세스·포트 카드                                    */
/* ─────────────────────────────────────────────────────────── */

export function ProcessSecurityCard({ data, accentColor }: { data: ProcessSecurityResult; accentColor: string }) {
  const suspicious = data.suspicious_processes ?? []
  const dangerPorts = (data.open_ports ?? []).filter(p => p.risk === 'high')

  return (
    <CardWrap accent={accentColor}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
        <ScoreCircle score={data.score} />
        <div>
          <div style={{ fontSize: 13, fontWeight: 800, color: suspicious.length ? '#f59e0b' : '#22c55e' }}>
            {suspicious.length ? `⚠️ 수상한 프로세스 ${suspicious.length}개` : '✅ 프로세스 정상'}
          </div>
          <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.4)', marginTop: 2 }}>
            위험 포트: {dangerPorts.length}개
          </div>
        </div>
      </div>

      {suspicious.length > 0 && (
        <div>
          <SectionTitle icon="🔴" title="수상한 프로세스" accentColor="#ef4444" />
          {suspicious.slice(0, 5).map((p, i) => (
            <div key={i} style={{
              padding: '4px 8px', borderRadius: 6, background: 'rgba(239,68,68,0.08)',
              border: '1px solid rgba(239,68,68,0.15)', marginBottom: 3,
            }}>
              <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                <span style={{ fontSize: 11, fontWeight: 700, color: '#fca5a5' }}>{p.name}</span>
                <RiskBadge risk={p.risk} />
              </div>
              <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.4)' }}>
                {p.reason} · CPU {p.cpu?.toFixed(0)}% · {p.mem_mb?.toFixed(0)}MB
              </div>
            </div>
          ))}
        </div>
      )}

      {dangerPorts.length > 0 && (
        <div>
          <SectionTitle icon="🚪" title="위험 포트" accentColor="#f59e0b" />
          {dangerPorts.slice(0, 3).map((p, i) => (
            <div key={i} style={{
              padding: '4px 8px', borderRadius: 6, background: 'rgba(245,158,11,0.08)',
              border: '1px solid rgba(245,158,11,0.15)', marginBottom: 3,
            }}>
              <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                <span style={{ fontSize: 11, color: '#fcd34d' }}>Port {p.port}</span>
                <span style={{ fontSize: 10, color: 'rgba(255,255,255,0.4)' }}>PID {p.pid}</span>
              </div>
              <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.4)' }}>{p.reason}</div>
            </div>
          ))}
        </div>
      )}
    </CardWrap>
  )
}

/* ─────────────────────────────────────────────────────────── */
/* Windows Defender 상태 카드                                   */
/* ─────────────────────────────────────────────────────────── */

export function DefenderCard({ data, accentColor }: { data: DefenderStatus; accentColor: string }) {
  const items = [
    { label: '바이러스 백신', ok: data.antivirus_enabled },
    { label: '실시간 보호', ok: data.realtime_protection },
    { label: `마지막 검사 (${data.quick_scan_age ?? '?'}일 전)`, ok: (data.quick_scan_age ?? 99) <= 7 },
  ]

  return (
    <CardWrap accent={accentColor}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
        <ScoreCircle score={data.score} />
        <div>
          <div style={{ fontSize: 13, fontWeight: 800, color: data.score >= 80 ? '#22c55e' : '#ef4444' }}>
            {data.score >= 80 ? '🛡️ Defender 정상' : '⚠️ 보안 설정 점검 필요'}
          </div>
          <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.4)', marginTop: 2 }}>
            Windows Defender 상태
          </div>
        </div>
      </div>

      <div style={{ display: 'flex', flexDirection: 'column', gap: 5 }}>
        {items.map((item, i) => (
          <div key={i} style={{
            display: 'flex', justifyContent: 'space-between', alignItems: 'center',
            padding: '5px 10px', borderRadius: 7, background: 'rgba(255,255,255,0.04)',
          }}>
            <span style={{ fontSize: 11, color: 'rgba(255,255,255,0.7)' }}>{item.label}</span>
            <span style={{ fontSize: 14 }}>{item.ok ? '✅' : '❌'}</span>
          </div>
        ))}
      </div>
    </CardWrap>
  )
}

/* ─────────────────────────────────────────────────────────── */
/* 시작 프로그램 카드                                           */
/* ─────────────────────────────────────────────────────────── */

export function StartupItemsCard({ data, accentColor }: {
  data: { items: StartupItem[]; total: number; suspicious_count: number }
  accentColor: string
}) {
  const suspItems = (data.items ?? []).filter(i => i.risk === 'high')

  return (
    <CardWrap accent={accentColor}>
      <SectionTitle icon="🚀" title="시작 프로그램 현황" accentColor={accentColor} />
      <div style={{ display: 'flex', gap: 10, marginBottom: 4 }}>
        <div style={{ flex: 1, textAlign: 'center', padding: '6px', borderRadius: 8, background: 'rgba(255,255,255,0.04)' }}>
          <div style={{ fontSize: 18, fontWeight: 900, color: accentColor }}>{data.total ?? 0}</div>
          <div style={{ fontSize: 9, color: 'rgba(255,255,255,0.4)' }}>전체</div>
        </div>
        <div style={{ flex: 1, textAlign: 'center', padding: '6px', borderRadius: 8, background: 'rgba(239,68,68,0.08)' }}>
          <div style={{ fontSize: 18, fontWeight: 900, color: '#ef4444' }}>{data.suspicious_count ?? 0}</div>
          <div style={{ fontSize: 9, color: 'rgba(255,255,255,0.4)' }}>수상</div>
        </div>
      </div>

      {suspItems.slice(0, 4).map((item, i) => (
        <div key={i} style={{
          padding: '4px 8px', borderRadius: 6, background: 'rgba(239,68,68,0.06)',
          border: '1px solid rgba(239,68,68,0.12)',
        }}>
          <div style={{ fontSize: 11, fontWeight: 700, color: '#fca5a5' }}>{item.name}</div>
          <div style={{ fontSize: 9, color: 'rgba(255,255,255,0.35)', wordBreak: 'break-all' }}>{item.command}</div>
        </div>
      ))}
    </CardWrap>
  )
}

/* ─────────────────────────────────────────────────────────── */
/* 프로세스 TOP 카드                                            */
/* ─────────────────────────────────────────────────────────── */

export function ProcessTopCard({ data, accentColor }: {
  data: { by_cpu: ProcItem[]; by_mem: ProcItem[] }
  accentColor: string
}) {
  const cpuList = data.by_cpu?.slice(0, 5) ?? []
  const memList = data.by_mem?.slice(0, 5) ?? []

  return (
    <CardWrap accent={accentColor}>
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10 }}>
        <div>
          <SectionTitle icon="🔥" title="CPU 상위" accentColor="#f97316" />
          {cpuList.map((p, i) => (
            <div key={i} style={{ display: 'flex', justifyContent: 'space-between', padding: '3px 0' }}>
              <span style={{ fontSize: 10, color: 'rgba(255,255,255,0.7)', maxWidth: 80, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{p.name}</span>
              <span style={{ fontSize: 10, color: '#fb923c', fontWeight: 700 }}>{p.cpu?.toFixed(0)}%</span>
            </div>
          ))}
        </div>
        <div>
          <SectionTitle icon="💾" title="RAM 상위" accentColor="#818cf8" />
          {memList.map((p, i) => (
            <div key={i} style={{ display: 'flex', justifyContent: 'space-between', padding: '3px 0' }}>
              <span style={{ fontSize: 10, color: 'rgba(255,255,255,0.7)', maxWidth: 80, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{p.name}</span>
              <span style={{ fontSize: 10, color: '#a5b4fc', fontWeight: 700 }}>{p.mem_mb?.toFixed(0)}MB</span>
            </div>
          ))}
        </div>
      </div>
    </CardWrap>
  )
}

/* ─────────────────────────────────────────────────────────── */
/* 시스템 제어 결과 카드 (볼륨·밝기·WiFi·전원·앱 실행)         */
/* ─────────────────────────────────────────────────────────── */

export function SystemActionCard({ icon, title, detail, success = true, accentColor }: {
  icon: string; title: string; detail?: string; success?: boolean; accentColor: string
}) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 6, scale: 0.97 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      style={{
        background: 'rgba(10,12,28,0.97)',
        border: `1px solid ${success ? accentColor : '#ef4444'}33`,
        borderRadius: 12, padding: '10px 14px',
        display: 'flex', alignItems: 'center', gap: 12,
      }}
    >
      <span style={{ fontSize: 28 }}>{icon}</span>
      <div>
        <div style={{ fontSize: 13, fontWeight: 800, color: success ? accentColor : '#ef4444' }}>{title}</div>
        {detail && <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.4)', marginTop: 2 }}>{detail}</div>}
      </div>
    </motion.div>
  )
}

/* ─────────────────────────────────────────────────────────── */
/* 네트워크 분석 카드                                           */
/* ─────────────────────────────────────────────────────────── */

export function NetworkAnalysisCard({ data, accentColor }: {
  data: { adapters: NetworkAdapter[]; dns_servers: string; public_ip: string; ping_ms: string; connected: boolean }
  accentColor: string
}) {
  return (
    <CardWrap accent={accentColor}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 2 }}>
        <span style={{ fontSize: 20 }}>{data.connected ? '🌐' : '📵'}</span>
        <div style={{ fontSize: 13, fontWeight: 800, color: data.connected ? '#22c55e' : '#ef4444' }}>
          {data.connected ? '인터넷 연결됨' : '인터넷 연결 없음'}
        </div>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 6 }}>
        {[
          { label: '공개 IP', value: data.public_ip || '확인 중' },
          { label: 'Ping', value: data.ping_ms ? `${data.ping_ms}ms` : '측정 중' },
          { label: 'DNS 서버', value: data.dns_servers || '알 수 없음' },
          { label: '어댑터', value: `${data.adapters?.length ?? 0}개 활성` },
        ].map((item, i) => (
          <div key={i} style={{ padding: '6px 8px', borderRadius: 7, background: 'rgba(255,255,255,0.04)' }}>
            <div style={{ fontSize: 9, color: 'rgba(255,255,255,0.35)' }}>{item.label}</div>
            <div style={{ fontSize: 11, fontWeight: 700, color: accentColor, marginTop: 1 }}>{item.value}</div>
          </div>
        ))}
      </div>

      {data.adapters?.slice(0, 2).map((a, i) => (
        <div key={i} style={{ padding: '4px 8px', borderRadius: 6, background: 'rgba(255,255,255,0.03)' }}>
          <span style={{ fontSize: 10, color: 'rgba(255,255,255,0.6)' }}>{a.name}</span>
          <span style={{ fontSize: 10, color: 'rgba(255,255,255,0.3)', marginLeft: 8 }}>{a.speed_mbps}Mbps</span>
        </div>
      ))}
    </CardWrap>
  )
}

/* ─────────────────────────────────────────────────────────── */
/* 드라이버 카드                                                */
/* ─────────────────────────────────────────────────────────── */

export function DriverCard({ data, accentColor }: {
  data: { total: number; problematic: DriverItem[]; problem_count: number; score: number; message: string }
  accentColor: string
}) {
  return (
    <CardWrap accent={accentColor}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
        <ScoreCircle score={data.score} />
        <div>
          <div style={{ fontSize: 13, fontWeight: 800, color: data.problem_count ? '#f59e0b' : '#22c55e' }}>
            {data.problem_count ? `⚠️ 문제 드라이버 ${data.problem_count}개` : '✅ 드라이버 정상'}
          </div>
          <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.4)', marginTop: 2 }}>전체 {data.total}개</div>
        </div>
      </div>

      {data.problematic?.slice(0, 4).map((d, i) => (
        <div key={i} style={{
          padding: '4px 8px', borderRadius: 6, background: 'rgba(245,158,11,0.07)',
          border: '1px solid rgba(245,158,11,0.15)',
        }}>
          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
            <span style={{ fontSize: 11, color: '#fde68a' }}>{d.name}</span>
            <RiskBadge risk={d.risk} />
          </div>
          <div style={{ fontSize: 9, color: 'rgba(255,255,255,0.35)' }}>{d.status} · {d.class}</div>
        </div>
      ))}
    </CardWrap>
  )
}

/* ─────────────────────────────────────────────────────────── */
/* 프로그램 목록 카드                                           */
/* ─────────────────────────────────────────────────────────── */

export function ProgramsListCard({ data, accentColor }: {
  data: { programs: ProgramItem[]; total: number }
  accentColor: string
}) {
  const list = data.programs?.slice(0, 8) ?? []

  return (
    <CardWrap accent={accentColor}>
      <SectionTitle icon="📦" title={`설치된 프로그램 ${data.total}개`} accentColor={accentColor} />
      <div style={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
        {list.map((p, i) => (
          <div key={i} style={{
            display: 'flex', justifyContent: 'space-between',
            padding: '4px 8px', borderRadius: 6, background: 'rgba(255,255,255,0.03)',
          }}>
            <span style={{ fontSize: 11, color: 'rgba(255,255,255,0.7)' }}>{p.name}</span>
            <span style={{ fontSize: 10, color: 'rgba(255,255,255,0.3)' }}>{p.version}</span>
          </div>
        ))}
        {data.total > 8 && (
          <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.3)', textAlign: 'center', padding: '4px 0' }}>
            + {data.total - 8}개 더...
          </div>
        )}
      </div>
    </CardWrap>
  )
}

/* ─────────────────────────────────────────────────────────── */
/* 파일 검색 결과 카드                                          */
/* ─────────────────────────────────────────────────────────── */

export function FileSearchCard({ data, accentColor }: {
  data: { results: FileResult[]; total: number; message: string }
  accentColor: string
}) {
  const list = data.results?.slice(0, 6) ?? []

  return (
    <CardWrap accent={accentColor}>
      <SectionTitle icon="🔍" title={data.message} accentColor={accentColor} />
      {list.length === 0 ? (
        <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.4)', textAlign: 'center', padding: '8px 0' }}>
          검색 결과가 없어요
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
          {list.map((f, i) => (
            <div key={i} style={{
              padding: '5px 8px', borderRadius: 7, background: 'rgba(255,255,255,0.04)',
            }}>
              <div style={{ fontSize: 11, fontWeight: 700, color: accentColor }}>{f.name}</div>
              <div style={{ fontSize: 9, color: 'rgba(255,255,255,0.35)', marginTop: 1, wordBreak: 'break-all' }}>
                {f.path} · {f.size_mb.toFixed(1)}MB · {f.mod_time}
              </div>
            </div>
          ))}
        </div>
      )}
    </CardWrap>
  )
}

/* ─────────────────────────────────────────────────────────── */
/* 중복 파일 카드                                               */
/* ─────────────────────────────────────────────────────────── */

export function DuplicatesCard({ data, accentColor }: {
  data: { groups: DupGroup[]; total_groups: number; waste_mb: number; waste: string; message: string }
  accentColor: string
}) {
  return (
    <CardWrap accent={accentColor}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
        <span style={{ fontSize: 28 }}>📋</span>
        <div>
          <div style={{ fontSize: 13, fontWeight: 800, color: data.total_groups ? '#f59e0b' : '#22c55e' }}>
            {data.total_groups ? `중복 파일 ${data.total_groups}그룹 발견` : '중복 파일 없음'}
          </div>
          {data.waste_mb > 0 && (
            <div style={{ fontSize: 10, color: '#fbbf24', marginTop: 2 }}>낭비 공간: {data.waste}</div>
          )}
        </div>
      </div>

      {data.groups?.slice(0, 4).map((g, i) => (
        <div key={i} style={{
          padding: '5px 8px', borderRadius: 7, background: 'rgba(245,158,11,0.07)',
          border: '1px solid rgba(245,158,11,0.12)',
        }}>
          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
            <span style={{ fontSize: 11, color: '#fde68a' }}>{g.name}</span>
            <span style={{ fontSize: 10, color: '#f59e0b' }}>{g.count}개 · {g.size_mb.toFixed(1)}MB</span>
          </div>
        </div>
      ))}
    </CardWrap>
  )
}

/* ─────────────────────────────────────────────────────────── */
/* 메모 카드                                                    */
/* ─────────────────────────────────────────────────────────── */

export function NotesCard({ data, accentColor }: {
  data: { notes: NoteItem[]; total: number }
  accentColor: string
}) {
  const list = data.notes?.slice(0, 4) ?? []

  return (
    <CardWrap accent={accentColor}>
      <SectionTitle icon="📝" title={`메모 ${data.total}개`} accentColor={accentColor} />
      {list.length === 0 ? (
        <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.4)', textAlign: 'center', padding: '8px 0' }}>
          저장된 메모가 없어요
        </div>
      ) : (
        list.map((n, i) => (
          <div key={i} style={{
            padding: '6px 8px', borderRadius: 7, background: 'rgba(255,255,255,0.04)',
            borderLeft: `3px solid ${accentColor}`,
          }}>
            <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.8)' }}>{n.content}</div>
            <div style={{ fontSize: 9, color: 'rgba(255,255,255,0.35)', marginTop: 2 }}>{n.created}</div>
          </div>
        ))
      )}
    </CardWrap>
  )
}

/* ─────────────────────────────────────────────────────────── */
/* 부팅 분석 카드                                               */
/* ─────────────────────────────────────────────────────────── */

export function BootAnalysisCard({ data, accentColor }: {
  data: { uptime_minutes: string; startup_count: string; message: string }
  accentColor: string
}) {
  const uptime = parseFloat(data.uptime_minutes ?? '0')
  const uptimeH = Math.floor(uptime / 60)
  const uptimeM = Math.floor(uptime % 60)

  return (
    <CardWrap accent={accentColor}>
      <SectionTitle icon="⚡" title="부팅 속도 분석" accentColor={accentColor} />
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 }}>
        <div style={{ padding: '8px', borderRadius: 8, background: 'rgba(255,255,255,0.04)', textAlign: 'center' }}>
          <div style={{ fontSize: 20, fontWeight: 900, color: accentColor }}>
            {uptimeH > 0 ? `${uptimeH}h ${uptimeM}m` : `${uptimeM}m`}
          </div>
          <div style={{ fontSize: 9, color: 'rgba(255,255,255,0.4)' }}>현재 가동 시간</div>
        </div>
        <div style={{ padding: '8px', borderRadius: 8, background: 'rgba(255,255,255,0.04)', textAlign: 'center' }}>
          <div style={{ fontSize: 20, fontWeight: 900, color: accentColor }}>{data.startup_count ?? '?'}</div>
          <div style={{ fontSize: 9, color: 'rgba(255,255,255,0.4)' }}>시작 프로그램</div>
        </div>
      </div>
      <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.4)', textAlign: 'center' }}>
        시작 프로그램이 많을수록 부팅이 느려져요
      </div>
    </CardWrap>
  )
}

/* ─────────────────────────────────────────────────────────── */
/* 집중 모드 카드                                               */
/* ─────────────────────────────────────────────────────────── */

export function FocusModeCard({ active, duration, accentColor }: {
  active: boolean; duration?: number; accentColor: string
}) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 6 }}
      animate={{ opacity: 1, y: 0 }}
      style={{
        background: active ? `${accentColor}15` : 'rgba(10,12,28,0.97)',
        border: `1px solid ${accentColor}44`,
        borderRadius: 12, padding: '12px 16px',
        display: 'flex', alignItems: 'center', gap: 12,
      }}
    >
      <span style={{ fontSize: 32 }}>{active ? '🎯' : '🔔'}</span>
      <div>
        <div style={{ fontSize: 13, fontWeight: 800, color: accentColor }}>
          집중 모드 {active ? '시작!' : '해제됨'}
        </div>
        {active && duration && (
          <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.5)', marginTop: 2 }}>
            {duration}분 동안 알림 차단 중
          </div>
        )}
      </div>
    </motion.div>
  )
}

/* ─────────────────────────────────────────────────────────── */
/* InlineCardData2 타입 + 렌더러                               */
/* ─────────────────────────────────────────────────────────── */

export type InlineCardData2 =
  | { type: 'price_compare'; data: { query: string; results: PriceItem[]; total: number; summary: string } }
  | { type: 'remote_access'; data: RemoteAccessResult }
  | { type: 'process_security'; data: ProcessSecurityResult }
  | { type: 'defender'; data: DefenderStatus }
  | { type: 'startup_items'; data: { items: StartupItem[]; total: number; suspicious_count: number } }
  | { type: 'process_top'; data: { by_cpu: ProcItem[]; by_mem: ProcItem[] } }
  | { type: 'system_action'; icon: string; title: string; detail?: string; success?: boolean }
  | { type: 'network'; data: { adapters: NetworkAdapter[]; dns_servers: string; public_ip: string; ping_ms: string; connected: boolean } }
  | { type: 'drivers'; data: { total: number; problematic: DriverItem[]; problem_count: number; score: number; message: string } }
  | { type: 'programs_list'; data: { programs: ProgramItem[]; total: number } }
  | { type: 'file_search'; data: { results: FileResult[]; total: number; message: string } }
  | { type: 'duplicates'; data: { groups: DupGroup[]; total_groups: number; waste_mb: number; waste: string; message: string } }
  | { type: 'notes'; data: { notes: NoteItem[]; total: number } }
  | { type: 'boot_analysis'; data: { uptime_minutes: string; startup_count: string; message: string } }
  | { type: 'focus_mode'; active: boolean; duration?: number }
  | { type: 'file_result'; data: { fileName: string; url: string; mimeType: string; width?: number; height?: number; frames?: number; operation?: string } }

export function InlineCardRenderer2({
  card,
  accentColor,
}: {
  card: InlineCardData2
  accentColor: string
}) {
  switch (card.type) {
    case 'price_compare':    return <PriceCompareCard     data={card.data} accentColor={accentColor} />
    case 'remote_access':    return <RemoteAccessCard    data={card.data} accentColor={accentColor} />
    case 'process_security': return <ProcessSecurityCard data={card.data} accentColor={accentColor} />
    case 'defender':         return <DefenderCard        data={card.data} accentColor={accentColor} />
    case 'startup_items':    return <StartupItemsCard    data={card.data} accentColor={accentColor} />
    case 'process_top':      return <ProcessTopCard      data={card.data} accentColor={accentColor} />
    case 'system_action':    return <SystemActionCard    icon={card.icon} title={card.title} detail={card.detail} success={card.success} accentColor={accentColor} />
    case 'network':          return <NetworkAnalysisCard data={card.data} accentColor={accentColor} />
    case 'drivers':          return <DriverCard          data={card.data} accentColor={accentColor} />
    case 'programs_list':    return <ProgramsListCard    data={card.data} accentColor={accentColor} />
    case 'file_search':      return <FileSearchCard      data={card.data} accentColor={accentColor} />
    case 'duplicates':       return <DuplicatesCard      data={card.data} accentColor={accentColor} />
    case 'notes':            return <NotesCard           data={card.data} accentColor={accentColor} />
    case 'boot_analysis':    return <BootAnalysisCard    data={card.data} accentColor={accentColor} />
    case 'focus_mode':       return <FocusModeCard       active={card.active} duration={card.duration} accentColor={accentColor} />
    case 'file_result':      return <FileResultCard       data={card.data} accentColor={accentColor} />
    default:                 return null
  }
}

function FileResultCard({ data, accentColor }: { data: { fileName: string; url: string; mimeType: string; width?: number; height?: number; frames?: number; operation?: string }; accentColor: string }) {
  const isImage = data.mimeType.startsWith('image/')
  const opLabel: Record<string, string> = { resize: '리사이즈', to_gif: 'GIF 변환', convert: '포맷 변환', compare: '비교 분석' }
  const label = opLabel[data.operation ?? ''] ?? '파일 처리'

  return (
    <div style={{ background: 'rgba(255,255,255,0.04)', border: `1px solid ${accentColor}44`, borderRadius: 12, padding: '10px 12px', marginTop: 4 }}>
      <div style={{ fontSize: 11, color: accentColor, fontWeight: 700, marginBottom: 6 }}>
        ✅ {label} 완료
      </div>
      {isImage && (
        <img src={data.url} alt={data.fileName}
          style={{ width: '100%', maxHeight: 140, objectFit: 'contain', borderRadius: 8, marginBottom: 6, background: 'rgba(0,0,0,0.3)' }} />
      )}
      <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.6)', marginBottom: 8 }}>
        {data.fileName}
        {data.width && data.height && <span style={{ marginLeft: 6, color: accentColor }}>· {data.width}×{data.height}</span>}
        {data.frames && <span style={{ marginLeft: 6, color: accentColor }}>· {data.frames}프레임</span>}
      </div>
      <a href={data.url} download={data.fileName}
        style={{
          display: 'inline-block', background: `linear-gradient(135deg, ${accentColor}, ${accentColor}99)`,
          color: '#fff', fontSize: 11, fontWeight: 700, padding: '6px 14px',
          borderRadius: 8, textDecoration: 'none', boxShadow: `0 2px 8px ${accentColor}44`,
        }}>
        ⬇ 다운로드
      </a>
    </div>
  )
}

function PriceCompareCard({ data, accentColor }: { data: { query: string; results: PriceItem[]; total: number; summary: string }; accentColor: string }) {
  const siteIcon: Record<string, string> = { 'coupang.com': '🛒', 'naver.com': '🟢', 'gmarket.co.kr': '🔵', 'elevenst.com': '1️⃣' }

  // 사이트별 최저가만 뽑기
  const bysite: Record<string, PriceItem[]> = {}
  data.results.forEach(r => {
    const key = r.site.replace(/^www\./, '')
    if (!bysite[key]) bysite[key] = []
    bysite[key].push(r)
  })

  const cheapest = data.results
    .filter(r => !r.blocked && r.price)
    .sort((a, b) => {
      const pa = parseInt(a.price.replace(/[^0-9]/g, '')) || 99999999
      const pb = parseInt(b.price.replace(/[^0-9]/g, '')) || 99999999
      return pa - pb
    })
    .slice(0, 6)

  return (
    <CardWrap accent={accentColor}>
      <SectionTitle icon="🛍️" title={`가격 비교: ${data.query}`} accentColor={accentColor} />
      {data.summary && (
        <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.7)', marginBottom: 4 }}>{data.summary}</div>
      )}
      <div style={{ display: 'flex', flexDirection: 'column', gap: 5 }}>
        {cheapest.length === 0 ? (
          <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.4)', padding: '8px 0' }}>수집된 가격 정보가 없어요.</div>
        ) : cheapest.map((item, i) => {
          const siteKey = item.site.replace(/^www\./, '')
          const icon = Object.entries(siteIcon).find(([k]) => siteKey.includes(k))?.[1] ?? '🔗'
          const isLowest = i === 0
          return (
            <a key={i} href={item.link} target="_blank" rel="noreferrer"
              style={{ textDecoration: 'none', display: 'flex', alignItems: 'center', gap: 8,
                background: isLowest ? `${accentColor}18` : 'rgba(255,255,255,0.04)',
                border: `1px solid ${isLowest ? accentColor + '55' : 'rgba(255,255,255,0.08)'}`,
                borderRadius: 8, padding: '6px 10px', cursor: 'pointer' }}>
              <span style={{ fontSize: 14 }}>{icon}</span>
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.5)', marginBottom: 1 }}>{item.site}</div>
                <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.85)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                  {item.name}
                </div>
              </div>
              <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'flex-end', gap: 2, flexShrink: 0 }}>
                <span style={{ fontSize: 13, fontWeight: 800, color: isLowest ? accentColor : 'rgba(255,255,255,0.9)' }}>
                  {item.price}
                </span>
                {isLowest && <span style={{ fontSize: 9, color: accentColor, fontWeight: 700, background: `${accentColor}22`, padding: '1px 5px', borderRadius: 4 }}>최저가</span>}
              </div>
            </a>
          )
        })}
      </div>
      <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.35)', marginTop: 2 }}>
        총 {data.total}개 결과 · 클릭하면 구매 페이지로 이동
      </div>
    </CardWrap>
  )
}

// AnimatePresence export for use in ChatBubble
export { AnimatePresence }
