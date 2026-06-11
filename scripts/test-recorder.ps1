# test-recorder.ps1 — 녹화기/자동화 엔진 Windows 실기 스모크 테스트
# 사용: 앱(Nexus) 실행 후 → powershell -ExecutionPolicy Bypass -File scripts\test-recorder.ps1
# 통과 기준: [4]에서 메모장에 클릭/타이핑하면 단계 수가 올라가고, [5] stop에서 steps JSON이 보임.

$B = "http://127.0.0.1:17891"
function Get-Json($method, $path, $body) {
    try {
        if ($body) { return Invoke-RestMethod -Method $method -Uri "$B$path" -ContentType "application/json" -Body ($body | ConvertTo-Json -Depth 8) }
        return Invoke-RestMethod -Method $method -Uri "$B$path"
    } catch { Write-Host "  FAIL $path : $_" -ForegroundColor Red; return $null }
}

Write-Host "`n[1] Go 백엔드 헬스" -ForegroundColor Cyan
$h = Get-Json GET "/api/health"; Write-Host "  -> $($h | ConvertTo-Json -Compress)"

Write-Host "`n[2] Python 사이드카 헬스 (여기서 실패하면 nexus-python.exe가 stub — build_python.ps1 필요)" -ForegroundColor Cyan
$p = Get-Json GET "/api/python/health"; Write-Host "  -> $($p | ConvertTo-Json -Compress)"

Write-Host "`n[3] UIA 엔진 상태 (available:true + admin 확인)" -ForegroundColor Cyan
$s = Get-Json GET "/api/automation/status"; Write-Host "  -> $($s | ConvertTo-Json -Compress)"
if (-not $s.available) { Write-Host "  ⚠️ available:false — pywinauto 미번들 또는 Python 미기동. 여기서 멈추면 빌드 문제." -ForegroundColor Yellow }

Write-Host "`n[4] 녹화 시작 — 지금부터 15초간 메모장 등에서 클릭/타이핑 하세요!" -ForegroundColor Green
$r = Get-Json POST "/api/automation/record/start" @{}
Write-Host "  -> $($r | ConvertTo-Json -Compress)"
for ($i = 1; $i -le 5; $i++) {
    Start-Sleep 3
    $st = Get-Json GET "/api/automation/record/status"
    Write-Host ("  {0}s: count={1} skipped={2} last_error={3}" -f ($i*3), $st.count, $st.skipped, $st.last_error)
}

Write-Host "`n[5] 녹화 중지 + 'smoke-test' 워크플로로 저장" -ForegroundColor Cyan
$stop = Get-Json POST "/api/automation/record/stop" @{ name = "smoke-test" }
Write-Host "  -> saved=$($stop.saved) count=$($stop.count) skipped=$($stop.skipped) last_error=$($stop.last_error)"
Write-Host "  steps:" -ForegroundColor Gray
$stop.steps | ConvertTo-Json -Depth 8

Write-Host "`n[6] 판정" -ForegroundColor Cyan
if ($stop.count -gt 0) {
    Write-Host "  ✅ 캡처 동작! 이제 앱 UI에서 ▶실행으로 재생을 확인하세요." -ForegroundColor Green
} else {
    Write-Host "  ❌ 캡처 0건 — last_error를 개발자(Claude)에게 전달: $($stop.last_error)" -ForegroundColor Red
    Write-Host "     로그: %APPDATA%\Nexus\logs\backend.log" -ForegroundColor Gray
}
