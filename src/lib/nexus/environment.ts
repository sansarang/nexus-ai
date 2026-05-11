/**
 * ══════════════════════════════════════════════════════════════
 * environment.ts — 실행 환경 감지 및 데이터 정책 명시
 * ══════════════════════════════════════════════════════════════
 *
 * [프로덕션 — 판매된 .exe]
 *   - Tauri 앱 안에서 실행됨 (window.__TAURI__ 존재)
 *   - Go 백엔드(port 17891)가 항상 함께 번들링되어 자동 시작됨
 *   - 모든 백엔드 호출은 실제 Windows API / PowerShell / WMI 결과를 반환
 *   - Mock 데이터 절대 사용 금지 → 실제 데이터로만 동작해야 함
 *
 * [개발 환경 — Mac / 브라우저]
 *   - Tauri 없이 Vite 개발 서버로 실행됨
 *   - Go 백엔드 미실행 → API 호출 실패
 *   - UI 개발/테스트 목적으로만 Mock 데이터 허용
 *   - 채팅창 상단에 "(개발 환경 · 모의 데이터)" 배너 표시
 */

/** Tauri 런타임 여부 (판매된 .exe에서 실행 중) */
export const IS_TAURI_APP: boolean =
  typeof window !== 'undefined' && '__TAURI__' in window

/**
 * 실제 백엔드 데이터를 강제해야 하는지 여부.
 *
 * true  → 프로덕션: backendAPI 호출 실패 시 에러를 그대로 사용자에게 노출
 *         "백엔드 서버에 연결할 수 없어요. 잠시 후 다시 시도해주세요."
 *
 * false → 개발:     backendAPI 호출 실패 시 Mock 데이터로 fallback
 *         UI 레이아웃 / 카드 디자인을 확인할 수 있도록 허용
 */
export const REQUIRE_REAL_BACKEND: boolean = IS_TAURI_APP

/**
 * 개발 환경에서 사용할 Mock 데이터 가져오기.
 * 프로덕션에서는 절대 호출되지 않음.
 *
 * 사용법:
 *   const data = await backendAPI.stats().catch(() => devMock(mockStats()))
 */
export function devMock<T>(mockValue: T): T {
  if (REQUIRE_REAL_BACKEND) {
    // 판매된 앱에서 이 함수를 호출하면 빌드 타임에 경고할 수 있도록 명시적 에러
    throw new Error(
      '[Nexus] devMock() called in production. Real backend data is required.'
    )
  }
  return mockValue
}

/**
 * 안전한 백엔드 호출 래퍼.
 *
 * - 프로덕션: 실패 시 null 반환 (UI에서 "연결 실패" 표시)
 * - 개발:     실패 시 fallback() 결과 반환
 */
export async function safeCall<T>(
  apiCall: () => Promise<T>,
  fallback?: () => T,
  onError?: (msg: string) => void
): Promise<T | null> {
  try {
    return await apiCall()
  } catch (err) {
    const msg =
      err instanceof Error ? err.message : '백엔드 연결 실패'

    if (REQUIRE_REAL_BACKEND) {
      // 프로덕션: 에러를 UI에 전달
      onError?.(msg)
      return null
    } else {
      // 개발: mock으로 fallback
      return fallback ? fallback() : null
    }
  }
}

/** 백엔드 연결 상태 타입 */
export type BackendStatus = 'connected' | 'disconnected' | 'checking'

/** 백엔드 헬스체크 */
export async function checkBackendHealth(): Promise<BackendStatus> {
  try {
    const res = await fetch('http://127.0.0.1:17891/api/health', {
      signal: AbortSignal.timeout(2000),
    })
    return res.ok ? 'connected' : 'disconnected'
  } catch {
    return 'disconnected'
  }
}
