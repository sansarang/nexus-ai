# Nexus AI v2.7.0 — Auth / Usage / License Smoke Test
# Run AFTER launching Nexus app
# Tests: license check, usage tracking, free-tier limits, limit enforcement

$ErrorActionPreference = "SilentlyContinue"
$BASE    = "http://127.0.0.1:17892"
$TIMEOUT = 10
$PASS = 0; $FAIL = 0; $WARN = 0

function hdr { param($t) Write-Host "`n━━━ $t ━━━" -ForegroundColor DarkCyan }
function ok  { param($n,$d) Write-Host "  [PASS] $n" -ForegroundColor Green;  Write-Host "         $d" -ForegroundColor DarkGray; $script:PASS++ }
function ng  { param($n,$d) Write-Host "  [FAIL] $n" -ForegroundColor Red;    Write-Host "         $d" -ForegroundColor DarkGray; $script:FAIL++ }
function wn  { param($n,$d) Write-Host "  [WARN] $n" -ForegroundColor Yellow; Write-Host "         $d" -ForegroundColor DarkGray; $script:WARN++ }
function inf { param($d)    Write-Host "  [INFO] $d" -ForegroundColor Gray }

function GET  {
    param($path)
    try { return Invoke-RestMethod "$BASE$path" -Method GET -TimeoutSec $TIMEOUT }
    catch { return $null }
}
function POST {
    param($path, $body)
    try { return Invoke-RestMethod "$BASE$path" -Method POST -Body ($body | ConvertTo-Json) -ContentType "application/json" -TimeoutSec $TIMEOUT }
    catch { return $null }
}

Write-Host "`n╔══════════════════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║   Nexus AI v2.7.0 — Auth / Usage / Limit Test       ║" -ForegroundColor Cyan
Write-Host "╚══════════════════════════════════════════════════════╝" -ForegroundColor Cyan
Write-Host "  Time: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')"

# ─── 0. 백엔드 생존 확인 ──────────────────────────────────────────
$h = GET "/api/health"
if (-not $h) {
    Write-Host "`n[중단] 백엔드 미응답 — Nexus 앱을 먼저 실행하세요." -ForegroundColor Red
    Read-Host "Enter"; exit 1
}
inf "백엔드 정상: status=$($h.status)"

# ─── 1. 라이선스 체크 ─────────────────────────────────────────────
hdr "1. 라이선스 (License)"

$lic = GET "/api/license/check"
if ($lic) {
    if ($lic.valid -eq $true) {
        ok "라이선스 인증됨" "key=$($lic.key)"
    } else {
        wn "라이선스 없음" "무료 플랜으로 동작 (정상)"
    }
} else {
    ng "라이선스 엔드포인트" "응답 없음"
}

# 잘못된 키 테스트
$badLic = POST "/api/license/activate" @{ key = "XXXX-XXXX-XXXX-XXXX" }
if ($badLic -and $badLic.valid -eq $false) {
    ok "잘못된 키 거부" "valid=false 정상 반환"
} else {
    ng "잘못된 키 거부" "예상된 거부 응답이 오지 않음"
}

# ─── 2. 사용량 현황 조회 ──────────────────────────────────────────
hdr "2. 사용량 현황 (Usage Status)"

$usage = GET "/api/usage"
if ($usage) {
    ok "사용량 조회" "free=$($usage.free.used)/$($usage.free.limit)  premium=$($usage.premium.used)/$($usage.premium.limit)"
    inf "무료 잔여: $($usage.free.left)회 / 프리미엄 잔여: $($usage.premium.left)회"
    inf "리셋 시각: $($usage.reset_at)"

    if ($usage.free.limit -eq 500)    { ok "무료 일일 한도" "500회 (Groq)" }
    else                               { ng "무료 일일 한도" "예상 500, 실제 $($usage.free.limit)" }

    if ($usage.premium.limit -eq 50)  { ok "프리미엄 일일 한도" "50회 (Claude/Perplexity)" }
    else                               { ng "프리미엄 일일 한도" "예상 50, 실제 $($usage.premium.limit)" }
} else {
    ng "사용량 조회" "응답 없음"
}

# ─── 3. Feature 사용량 한도 조회 ──────────────────────────────────
hdr "3. Feature 사용량 (Feature Limits)"

$usageAI = GET "/api/usage/ai"
if ($usageAI) {
    ok "Feature usage 조회" ($usageAI | ConvertTo-Json -Compress -Depth 2)[0..200]
} else {
    wn "Feature usage 조회" "응답 없거나 구조 미확인"
}

# ─── 4. LLM 채팅 → 사용량 증가 확인 ─────────────────────────────
hdr "4. LLM 채팅 호출 & 사용량 증가 확인"

$llmCfg = GET "/api/llm/config"
$hasKey = $llmCfg -and (($llmCfg.groq_key -ne "" -and $null -ne $llmCfg.groq_key) -or
                         ($llmCfg.claude_key -ne "" -and $null -ne $llmCfg.claude_key))

if (-not $hasKey) {
    wn "LLM 채팅 테스트" "API 키 없음 — 설정 후 재테스트 필요"
    inf "설정 방법: 앱 실행 → 설정 → API 키 입력 후 다시 실행"
} else {
    # 채팅 전 사용량
    $before = GET "/api/usage"
    $freeBefore = $before.free.used

    inf "채팅 전 사용량: free=$freeBefore"

    # 채팅 3회 호출
    $chatOk = 0
    for ($i = 1; $i -le 3; $i++) {
        $r = POST "/api/llm/chat" @{ message = "테스트 $i : 한 단어로만 대답해"; stream = $false }
        if ($r -and $r.response) {
            Write-Host "  ↳ [$i] $($r.response[0..60] -join '')" -ForegroundColor DarkGray
            $chatOk++
        }
        Start-Sleep -Milliseconds 500
    }

    if ($chatOk -eq 3) { ok "LLM 채팅 3회 응답" "모두 성공" }
    else               { ng "LLM 채팅 3회 응답" "$chatOk/3 성공" }

    # 채팅 후 사용량
    $after = GET "/api/usage"
    $freeAfter = $after.free.used
    inf "채팅 후 사용량: free=$freeAfter (증가: $($freeAfter - $freeBefore))"

    if ($freeAfter -gt $freeBefore) { ok "사용량 카운트 증가" "채팅 후 $($freeAfter - $freeBefore)회 증가 확인" }
    else                            { ng "사용량 카운트 증가" "사용량이 증가하지 않음 — 추적 버그" }
}

# ─── 5. 한도 초과 시뮬레이션 (POST /api/usage/ai) ─────────────────
hdr "5. 한도 초과 응답 시뮬레이션"

# 존재하지 않는 user_id로 조회 (새 사용자 시뮬레이션)
$newUser = GET "/api/usage?user_id=smoke-test-user-$(Get-Random)"
if ($newUser) {
    ok "신규 사용자 사용량" "free=$($newUser.free.used) (0이어야 함)"
    if ($newUser.free.used -eq 0) { ok "신규 사용자 초기값" "0으로 초기화 확인" }
    else                          { ng "신규 사용자 초기값" "0이어야 하는데 $($newUser.free.used)" }
}

# feature 한도 POST 테스트
$featTest = POST "/api/usage/ai" @{ feature = "ai_request"; user_id = "smoke-test-$(Get-Random)" }
if ($featTest) {
    ok "Feature usage 기록" ($featTest | ConvertTo-Json -Compress)[0..100]
} else {
    wn "Feature usage POST" "응답 없음"
}

# ─── 6. 회원가입 UI 수동 체크리스트 출력 ─────────────────────────
hdr "6. [수동 확인 필요] 회원가입 / 로그인 UI"

Write-Host @"

  아래 항목은 앱 UI에서 직접 확인이 필요합니다:

  [ ] 앱 실행 시 로그인 화면 표시 여부
  [ ] Google/이메일 회원가입 버튼 작동 여부
  [ ] 회원가입 완료 후 메인 화면 진입
  [ ] 로그인 상태에서 /api/usage?user_id=<내이메일> 호출 시 사용량 확인
  [ ] 무료 15회 소진 후 "횟수 초과" 메시지 표시 여부
  [ ] 내일 자정 리셋 메시지 표시 여부

  테스트 방법:
  1. 앱에서 회원가입 완료 후
  2. 아래 명령어로 사용량 직접 조회:

  Invoke-RestMethod "http://127.0.0.1:17892/api/usage?user_id=<이메일>" | ConvertTo-Json

"@ -ForegroundColor Yellow

# ─── 요약 ─────────────────────────────────────────────────────────
$TOTAL = $PASS + $FAIL + $WARN
Write-Host "`n╔══════════════════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║                   테스트 결과 요약                   ║" -ForegroundColor Cyan
Write-Host "╚══════════════════════════════════════════════════════╝" -ForegroundColor Cyan
Write-Host "  총 테스트 : $TOTAL"
Write-Host "  PASS      : $PASS" -ForegroundColor Green
Write-Host "  FAIL      : $FAIL" -ForegroundColor $(if($FAIL -gt 0){"Red"}else{"Gray"})
Write-Host "  WARN      : $WARN" -ForegroundColor Yellow

$reportPath = "$env:USERPROFILE\Desktop\nexus-auth-test-$(Get-Date -Format 'yyyyMMdd-HHmmss').txt"
@(
    "Nexus Auth/Usage Smoke Test — $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')"
    "PASS: $PASS  FAIL: $FAIL  WARN: $WARN"
    "백엔드: $BASE"
) | Out-File $reportPath -Encoding UTF8
Write-Host "  결과 저장: $reportPath" -ForegroundColor Gray

Read-Host "`nEnter 키로 종료"
