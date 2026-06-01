# Nexus AI E2E 시뮬레이터 (Windows 환경 — 실 Haiku + Perplexity + Tavily 동작)
# 사용: PowerShell 에서
#   iex (irm https://raw.githubusercontent.com/sansarang/nexus-ai/main/test/e2e.ps1)
# 또는 직접:
#   .\e2e.ps1
#
# 결과: 콘솔 + Desktop\nx_e2e_result.json 에 저장

$ErrorActionPreference = "Continue"
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$OutputEncoding = [System.Text.Encoding]::UTF8

# 백엔드 헬스 체크
try {
    $h = Invoke-RestMethod -Uri "http://127.0.0.1:17891/api/health" -TimeoutSec 5
    Write-Host "✅ 백엔드 응답: $($h.status) ($($h.platform))" -ForegroundColor Green
} catch {
    Write-Host "❌ 백엔드 연결 실패. Nexus 앱 실행 중인지 확인하세요." -ForegroundColor Red
    Write-Host "   에러: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# 사용자 JWT 토큰 가져오기 (앱이 저장한 곳)
# Tauri WebView2 localStorage 는 직접 접근 어려움 → 인증 없이 진행 (백엔드 번들 키 + 로컬 fallback)
Write-Host "ℹ️  JWT 없이 진행 — 백엔드 번들 키 사용" -ForegroundColor Cyan
Write-Host ""

# ── 60개 실사용자 질문 (카테고리별) ────────────────────────
$tests = @(
    # 시스템 진단 (즉시 라우팅 기대)
    @{ q="PC 메모리 확인해줘"; cat="시스템"; expect=@("stats") },
    @{ q="내 컴퓨터 메모리 얼마나 남았어"; cat="시스템"; expect=@("stats") },
    @{ q="디스크 용량 좀 봐줘"; cat="시스템"; expect=@("stats") },
    @{ q="CPU 사용률 어때"; cat="시스템"; expect=@("stats") },
    @{ q="시스템 상태"; cat="시스템"; expect=@("stats") },
    @{ q="보안 스캔 해줘"; cat="시스템"; expect=@("scan","security_scan") },
    @{ q="바이러스 검사해"; cat="시스템"; expect=@("scan","security_scan") },
    @{ q="악성코드 확인"; cat="시스템"; expect=@("scan","security_scan") },
    @{ q="캐시 정리해줘"; cat="시스템"; expect=@("clean") },
    @{ q="임시파일 정리"; cat="시스템"; expect=@("clean") },
    @{ q="공간 확보해줘"; cat="시스템"; expect=@("clean") },
    @{ q="PC 너무 느려"; cat="시스템"; expect=@("scan","stats","full_scan") },
    @{ q="컴퓨터 왜 이래 버벅거려"; cat="시스템"; expect=@("scan","full_scan") },

    # 정보 검색
    @{ q="오늘 서울 날씨"; cat="검색"; expect=@("weather") },
    @{ q="부산 날씨 어때"; cat="검색"; expect=@("weather") },
    @{ q="뉴욕 날씨"; cat="검색"; expect=@("weather") },
    @{ q="오늘 IT 뉴스 알려줘"; cat="검색"; expect=@("web_search","news_search") },
    @{ q="AI 최근 트렌드"; cat="검색"; expect=@("web_search","chat") },
    @{ q="삼성전자 주가"; cat="검색"; expect=@("stock","web_search") },
    @{ q="원달러 환율"; cat="검색"; expect=@("exchange_rate","web_search","chat") },
    @{ q="강남역 맛집"; cat="검색"; expect=@("web_search") },

    # 쇼핑/가격 비교
    @{ q="아이폰 15 가격 비교"; cat="쇼핑"; expect=@("price_compare","multi_action") },
    @{ q="쿠팡에서 갤럭시 S24"; cat="쇼핑"; expect=@("price_compare","multi_action") },
    @{ q="에어팟 프로 2 최저가"; cat="쇼핑"; expect=@("price_compare") },
    @{ q="다이슨 청소기 가격"; cat="쇼핑"; expect=@("price_compare","web_search") },
    @{ q="네이버쇼핑에서 한정판 운동화"; cat="쇼핑"; expect=@("price_compare","multi_action") },
    @{ q="알리에서 USB 케이블 싸게"; cat="쇼핑"; expect=@("price_compare","web_search") },

    # 영상/미디어
    @{ q="유튜브에서 잔잔한 노래"; cat="영상"; expect=@("video_search") },
    @{ q="유튜브 음악 추천"; cat="영상"; expect=@("video_search") },
    @{ q="틱톡 바이럴 영상"; cat="영상"; expect=@("video_search") },
    @{ q="BTS 뮤직비디오 찾아줘"; cat="영상"; expect=@("video_search") },
    @{ q="요리 유튜브 영상"; cat="영상"; expect=@("video_search") },
    @{ q="아이유 라이브 영상"; cat="영상"; expect=@("video_search") },

    # 생산성 / 업무
    @{ q="오늘 일정 알려줘"; cat="업무"; expect=@("calendar_today","clarify") },
    @{ q="이번 주 일정"; cat="업무"; expect=@("calendar_week","clarify") },
    @{ q="받은 메일 보여줘"; cat="업무"; expect=@("email_inbox","clarify") },
    @{ q="이메일 요약해줘"; cat="업무"; expect=@("email_summarize","email_inbox") },
    @{ q="번역해줘 안녕하세요를 영어로"; cat="업무"; expect=@("translate","chat") },
    @{ q="볼륨 50으로"; cat="업무"; expect=@("system_control","volume_control") },
    @{ q="크롬 켜줘"; cat="업무"; expect=@("launch_app","system_control") },
    @{ q="다운로드 폴더 열어줘"; cat="업무"; expect=@("open_folder","system_control") },

    # 모호/짧은 질문
    @{ q="안녕"; cat="인사"; expect=@("chat","clarify") },
    @{ q="고마워"; cat="인사"; expect=@("chat","clarify") },
    @{ q="뭐해"; cat="인사"; expect=@("chat","clarify") },
    @{ q="도와줘"; cat="인사"; expect=@("chat","clarify") },

    # 한영 혼합
    @{ q="weather in Seoul"; cat="영문"; expect=@("weather") },
    @{ q="ChatGPT 와 Claude 차이"; cat="영문"; expect=@("chat","web_search") },
    @{ q="how to install python"; cat="영문"; expect=@("chat","web_search") },

    # 연속성 패턴 (프론트가 잡아야 — 백엔드는 chat/clarify)
    @{ q="방금 거 다시"; cat="연속성"; expect=@("chat","clarify") },
    @{ q="이거 복사해"; cat="연속성"; expect=@("chat","clarify") },
    @{ q="그만 말해"; cat="연속성"; expect=@("chat","clarify") },

    # 의도 모호 — Haiku 가 판단하는 영역
    @{ q="자비스 같은거 만들어줘"; cat="모호"; expect=@("chat","clarify") },
    @{ q="포토샵 비슷한거 추천"; cat="모호"; expect=@("web_search","chat") },
    @{ q="다음주 부산 출장 계획"; cat="모호"; expect=@("trip_plan","web_search","chat","calendar_add") },
    @{ q="강아지 산책 가능한 공원"; cat="모호"; expect=@("web_search","chat") },

    # AI 고급
    @{ q="이 사진 뭐야"; cat="AI"; expect=@("vision","screen_analyze","clarify") },
    @{ q="화면 캡쳐해서 분석"; cat="AI"; expect=@("vision","screen_analyze") },
    @{ q="문서 요약해줘"; cat="AI"; expect=@("doc_summary","clarify") },
    @{ q="이력서 검토해줘"; cat="AI"; expect=@("doc_summary","clarify","chat") }
)

# ── 실행 ────────────────────────────────────────────
$results = @()
$idx = 0
$total = $tests.Count
Write-Host "🧪 총 $total 개 질문 시작 (예상 시간: 5~10분)`n" -ForegroundColor Yellow

foreach ($t in $tests) {
    $idx++
    $q = $t.q
    $cat = $t.cat
    $expected = $t.expect

    $t0 = Get-Date
    $action = "?"; $message = ""; $success = $false; $err = $null
    try {
        $body = @{ message = $q; lang = "ko" } | ConvertTo-Json -Compress
        $bodyBytes = [System.Text.Encoding]::UTF8.GetBytes($body)
        $resp = Invoke-RestMethod -Uri "http://127.0.0.1:17891/api/command" `
            -Method POST -ContentType "application/json; charset=utf-8" `
            -Body $bodyBytes -TimeoutSec 90
        $action = if ($resp.action) { $resp.action } else { "?" }
        $message = if ($resp.message) {
            if ($resp.message.Length -gt 80) { $resp.message.Substring(0, 80) + "..." } else { $resp.message }
        } else { "" }
        $success = $resp.success
    } catch {
        $err = $_.Exception.Message
        $action = "ERROR"
    }
    $dur = ((Get-Date) - $t0).TotalSeconds

    # 분류
    $status = "PASS"
    $issues = @()
    if ($err) {
        $status = "ERROR"
        $issues += "네트워크: $err"
    } else {
        if ($expected -notcontains $action) {
            $issues += "기대 $($expected -join '/') ≠ '$action'"
            $status = "WARN"
        }
        if ($dur -gt 5) { $issues += "느림 $([math]::Round($dur,1))s"; if ($status -eq "PASS") { $status = "WARN" } }
        if (-not $message -and $action -ne "clarify") { $issues += "msg 비어있음"; $status = "WARN" }
    }

    $icon = switch ($status) {
        "PASS"  { "✅" }
        "WARN"  { "⚠️ " }
        "ERROR" { "💥" }
        default { "❓" }
    }
    $color = switch ($status) {
        "PASS"  { "Green" }
        "WARN"  { "Yellow" }
        "ERROR" { "Red" }
        default { "Gray" }
    }

    $line = ("{0} #{1,2}/{2,2} [{3,-4}] '{4}'  →  {5}  {6:F1}s" -f $icon, $idx, $total, $cat, $q, $action, $dur)
    Write-Host $line -ForegroundColor $color
    if ($message) {
        Write-Host ("    {0}" -f $message) -ForegroundColor DarkGray
    }
    if ($issues.Count -gt 0) {
        Write-Host ("    └─ {0}" -f ($issues -join " | ")) -ForegroundColor DarkYellow
    }

    $results += [pscustomobject]@{
        idx = $idx
        query = $q
        category = $cat
        expected = $expected -join ","
        action = $action
        duration_sec = [math]::Round($dur, 2)
        status = $status
        message = $message
        issues = $issues -join " | "
        success = $success
        error = $err
    }
}

# ── 요약 ────────────────────────────────────────────
Write-Host ""
Write-Host ("=" * 80) -ForegroundColor Cyan
$pass = ($results | Where-Object { $_.status -eq "PASS" }).Count
$warn = ($results | Where-Object { $_.status -eq "WARN" }).Count
$fail = ($results | Where-Object { $_.status -in @("FAIL","ERROR") }).Count
Write-Host ("📊 결과: ✅PASS {0}  ⚠️WARN {1}  💥FAIL {2}  =  {3}/{4} ({5}%)" -f $pass, $warn, $fail, $pass, $total, [math]::Round(100*$pass/$total)) -ForegroundColor Yellow

# 카테고리별 통계
Write-Host "`n📂 카테고리별:" -ForegroundColor Cyan
$results | Group-Object category | ForEach-Object {
    $catName = $_.Name
    $catPass = ($_.Group | Where-Object { $_.status -eq "PASS" }).Count
    $catTotal = $_.Group.Count
    Write-Host ("  {0,-8} {1}/{2}" -f $catName, $catPass, $catTotal)
}

# 결과 저장
$outFile = "$env:USERPROFILE\Desktop\nx_e2e_result.json"
$results | ConvertTo-Json -Depth 4 | Out-File -FilePath $outFile -Encoding UTF8
Write-Host "`n💾 상세 결과 저장: $outFile" -ForegroundColor Green

# 실패/경고 케이스만 따로 출력 (수정 우선순위)
$problems = $results | Where-Object { $_.status -ne "PASS" }
if ($problems.Count -gt 0) {
    Write-Host "`n🔴 수정 우선순위 ($($problems.Count)건):" -ForegroundColor Red
    $problems | ForEach-Object {
        Write-Host ("  • [{0}] '{1}' → action={2} ({3}s)" -f $_.category, $_.query, $_.action, $_.duration_sec) -ForegroundColor Red
        if ($_.issues) { Write-Host ("       {0}" -f $_.issues) -ForegroundColor DarkRed }
    }
}
