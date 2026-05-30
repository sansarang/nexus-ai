/**
 * 데스크톱 제어 승인 모드 관리
 *
 * "마우스 자동화 승인모드는 사용자에게 물어서" 결정 반영.
 * 사용자는 다음 3가지 중 선택:
 *  - always  : 매번 확인 (가장 안전, 기본값)
 *  - trust5m : 5분간 신뢰 (편의)
 *  - trust1h : 1시간 신뢰 (집중 작업)
 *  - never   : 영구 신뢰 (Pro 사용자 권장)
 */

export type ApprovalMode = 'always' | 'trust5m' | 'trust1h' | 'never'

const MODE_KEY = 'nexus-desktop-approval-mode'
const TRUST_UNTIL_KEY = 'nexus-desktop-trust-until'

export function getApprovalMode(): ApprovalMode {
  if (typeof localStorage === 'undefined') return 'always'
  const v = localStorage.getItem(MODE_KEY) as ApprovalMode | null
  return v ?? 'always'
}

export function setApprovalMode(mode: ApprovalMode) {
  if (typeof localStorage === 'undefined') return
  localStorage.setItem(MODE_KEY, mode)
  if (mode === 'trust5m') {
    localStorage.setItem(TRUST_UNTIL_KEY, String(Date.now() + 5 * 60 * 1000))
  } else if (mode === 'trust1h') {
    localStorage.setItem(TRUST_UNTIL_KEY, String(Date.now() + 60 * 60 * 1000))
  } else {
    localStorage.removeItem(TRUST_UNTIL_KEY)
  }
}

/**
 * 현재 작업에 사용자 승인이 필요한가?
 * @param isDangerous - 위험 작업 (삭제/결제/시스템 명령) → 항상 승인 필요
 */
export function needsApproval(isDangerous = false): boolean {
  if (isDangerous) return true // 위험 작업은 모드 무관
  const mode = getApprovalMode()
  if (mode === 'always') return true
  if (mode === 'never') return false
  // trust5m / trust1h — 유효 기간 확인
  if (typeof localStorage === 'undefined') return true
  const until = parseInt(localStorage.getItem(TRUST_UNTIL_KEY) ?? '0', 10)
  if (until > Date.now()) return false
  // 만료 — 다시 always 모드로 복귀
  localStorage.setItem(MODE_KEY, 'always')
  localStorage.removeItem(TRUST_UNTIL_KEY)
  return true
}

/** 신뢰 모드 남은 시간 (ms). 없으면 0. */
export function trustRemaining(): number {
  if (typeof localStorage === 'undefined') return 0
  const until = parseInt(localStorage.getItem(TRUST_UNTIL_KEY) ?? '0', 10)
  return Math.max(0, until - Date.now())
}

/** 사용자가 최초로 데스크톱 제어를 시도했을 때 모드 선택했는지 */
export function hasChosenMode(): boolean {
  if (typeof localStorage === 'undefined') return false
  return localStorage.getItem(MODE_KEY) !== null
}

/** 위험 키워드 판별 — 백엔드 isDangerousAction과 동기화 */
const DANGEROUS_KEYWORDS = [
  '삭제', '지워', '제거', 'delete', 'remove', 'rm', 'uninstall',
  '결제', '구매', '주문', 'pay', 'payment', 'purchase', 'buy',
  '송금', '이체', 'transfer',
  '포맷', 'format',
  '종료', '재시작', 'shutdown', 'restart',
  '관리자', 'admin', 'sudo',
]

export function isDangerousCommand(text: string): boolean {
  const lower = text.toLowerCase()
  return DANGEROUS_KEYWORDS.some(k => lower.includes(k))
}
