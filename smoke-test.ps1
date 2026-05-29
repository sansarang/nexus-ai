# Nexus AI v2.7.0 — Smoke Test Script
# Run AFTER launching Nexus app (backend must be running on :17892)
# Usage: .\smoke-test.ps1

$ErrorActionPreference = "SilentlyContinue"
$BASE = "http://127.0.0.1:17892"
$TIMEOUT = 10
$PASS = 0; $FAIL = 0; $WARN = 0
$RESULTS = @()

function hdr { param($t) Write-Host "`n━━━ $t ━━━" -ForegroundColor DarkCyan }
function ok  { param($n,$d) Write-Host "  [PASS] $n : $d" -ForegroundColor Green;  $script:PASS++; $script:RESULTS += "PASS|$n|$d" }
function ng  { param($n,$d) Write-Host "  [FAIL] $n : $d" -ForegroundColor Red;    $script:FAIL++; $script:RESULTS += "FAIL|$n|$d" }
function wn  { param($n,$d) Write-Host "  [WARN] $n : $d" -ForegroundColor Yellow; $script:WARN++; $script:RESULTS += "WARN|$n|$d" }

function GET {
    param($path, $desc)
    try {
        $r = Invoke-RestMethod "$BASE$path" -Method GET -TimeoutSec $TIMEOUT
        ok $desc (($r | ConvertTo-Json -Compress -Depth 1)[0..120] -join "")
        return $r
    } catch {
        ng $desc $_.Exception.Message
        return $null
    }
}

function POST {
    param($path, $body, $desc)
    try {
        $json = $body | ConvertTo-Json -Compress
        $r = Invoke-RestMethod "$BASE$path" -Method POST -Body $json -ContentType "application/json" -TimeoutSec $TIMEOUT
        ok $desc (($r | ConvertTo-Json -Compress -Depth 1)[0..120] -join "")
        return $r
    } catch {
        ng $desc $_.Exception.Message
        return $null
    }
}

Write-Host "`n╔══════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║   Nexus AI v2.7.0 — Smoke Test           ║" -ForegroundColor Cyan
Write-Host "╚══════════════════════════════════════════╝" -ForegroundColor Cyan
Write-Host "  Target: $BASE"
Write-Host "  Time  : $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')"

# ─── 1. 백엔드 헬스 ────────────────────────────────────────
hdr "1. 백엔드 헬스"

try {
    $h = Invoke-RestMethod "$BASE/api/health" -Method GET -TimeoutSec 3
    ok "/api/health" "status=$($h.status)"
} catch {
    ng "/api/health" "백엔드 미응답 — Nexus 앱이 실행 중인지 확인하세요"
    Write-Host "`n[중단] 백엔드가 없어 테스트를 계속할 수 없습니다." -ForegroundColor Red
    Write-Host "Nexus 앱을 먼저 실행하고 다시 시도하세요." -ForegroundColor Yellow
    Read-Host "Enter"
    exit 1
}

$py = GET "/api/python/health" "/api/python/health"
if ($py -and $py.healthy -eq $true) { Write-Host "  ↳ Python sidecar: 정상" -ForegroundColor Green }
else { wn "/api/python/health" "Python sidecar 미응답 (Python 의존 기능 사용 불가)" }

# ─── 2. 시스템 기본 ───────────────────────────────────────
hdr "2. 시스템 기본"
$stats = GET "/api/stats" "/api/stats (PC 리소스)"
if ($stats) {
    $cpu = $stats.cpu_percent ?? $stats.cpu
    $ram = $stats.ram_percent ?? $stats.ram
    if ($cpu -ne $null) { Write-Host "  ↳ CPU: $cpu%  RAM: $ram%" -ForegroundColor Gray }
}
GET "/api/daily-report"   "/api/daily-report"
GET "/api/processes/top"  "/api/processes/top"
GET "/api/programs"       "/api/programs (설치된 앱 목록)"
GET "/api/gpu/stats"      "/api/gpu/stats"

# ─── 3. 보안 점검 ─────────────────────────────────────────
hdr "3. 보안 점검"
GET "/api/security/defender"  "/api/security/defender"
GET "/api/security/remote"    "/api/security/remote"
GET "/api/security/processes" "/api/security/processes"
GET "/api/security/startup"   "/api/security/startup"

# ─── 4. LLM / AI ─────────────────────────────────────────
hdr "4. LLM / AI"
$llmCfg = GET "/api/llm/config" "/api/llm/config"
$hasKey = $false
if ($llmCfg) {
    $hasKey = ($llmCfg.groq_key -ne "" -and $llmCfg.groq_key -ne $null) -or
              ($llmCfg.claude_key -ne "" -and $llmCfg.claude_key -ne $null)
    Write-Host "  ↳ Groq키: $(if($llmCfg.groq_key){'설정됨'}else{'없음'})  Claude키: $(if($llmCfg.claude_key){'설정됨'}else{'없음'})" -ForegroundColor Gray
}

if ($hasKey) {
    $chat = POST "/api/llm/chat" @{ message = "안녕, 테스트야. 한 문장으로 대답해줘."; stream = $false } "/api/llm/chat (AI 응답)"
    if ($chat -and $chat.response) {
        Write-Host "  ↳ AI 응답: $($chat.response[0..80] -join '')" -ForegroundColor Gray
    }
} else {
    wn "/api/llm/chat" "API 키 없음 — 설정 후 재테스트 필요"
}

GET "/api/features" "/api/features (Feature Flag)"

# ─── 5. 파일 & 메모리 ─────────────────────────────────────
hdr "5. 파일 & 메모리"
POST "/api/files/search" @{ query = "test"; path = "C:\Users"; limit = 5 } "/api/files/search"
GET  "/api/memory/stats"  "/api/memory/stats"
GET  "/api/memory/list"   "/api/memory/list"

# ─── 6. 생산성 도구 ───────────────────────────────────────
hdr "6. 생산성 도구"
GET "/api/notes"          "/api/notes"
GET "/api/macros"         "/api/macros"
GET "/api/scheduler/list" "/api/scheduler/list"
GET "/api/journal/today"  "/api/journal/today"
GET "/api/persona/list"   "/api/persona/list"
GET "/api/persona/current" "/api/persona/current"

# ─── 7. 네트워크 & 시스템 ─────────────────────────────────
hdr "7. 네트워크 & 시스템"
GET "/api/network/analysis" "/api/network/analysis"
GET "/api/power/plans"      "/api/power/plans"
GET "/api/boot/analysis"    "/api/boot/analysis"
GET "/api/system/updates"   "/api/system/updates"

# ─── 8. 브라우저 자동화 ───────────────────────────────────
hdr "8. 브라우저 자동화"
GET "/api/browser/status" "/api/browser/status (Chromedp)"

# ─── 9. 알림 & 히스토리 ───────────────────────────────────
hdr "9. 알림 & 히스토리"
GET "/api/alerts/latest"      "/api/alerts/latest"
GET "/api/history/stats"      "/api/history/stats"
GET "/api/history/anomalies"  "/api/history/anomalies"

# ─── 10. 날씨 & 캘린더 ────────────────────────────────────
hdr "10. 날씨 & 캘린더"
GET "/api/weather"          "/api/weather"
GET "/api/calendar/today"   "/api/calendar/today"
GET "/api/calendar/week"    "/api/calendar/week"

# ─── 요약 ─────────────────────────────────────────────────
$TOTAL = $PASS + $FAIL + $WARN
Write-Host "`n╔══════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║              테스트 결과 요약             ║" -ForegroundColor Cyan
Write-Host "╚══════════════════════════════════════════╝" -ForegroundColor Cyan
Write-Host "  총 테스트 : $TOTAL"
Write-Host "  PASS      : $PASS" -ForegroundColor Green
Write-Host "  FAIL      : $FAIL" -ForegroundColor Red
Write-Host "  WARN      : $WARN" -ForegroundColor Yellow

if ($FAIL -gt 0) {
    Write-Host "`n[실패 항목]" -ForegroundColor Red
    $RESULTS | Where-Object { $_ -like "FAIL*" } | ForEach-Object {
        $parts = $_ -split "\|"
        Write-Host "  ✗ $($parts[1]): $($parts[2])" -ForegroundColor Red
    }
}

# 결과 파일 저장
$reportPath = "$env:USERPROFILE\Desktop\nexus-smoke-test-$(Get-Date -Format 'yyyyMMdd-HHmmss').txt"
$RESULTS | ForEach-Object { $_ } | Out-File $reportPath -Encoding UTF8
Write-Host "`n결과 저장: $reportPath" -ForegroundColor Gray

if ($FAIL -eq 0) {
    Write-Host "`n[OK] 모든 핵심 테스트 통과!" -ForegroundColor Green
} else {
    Write-Host "`n[!!] 실패 항목을 확인하세요." -ForegroundColor Red
}

Read-Host "`nEnter 키로 종료"
