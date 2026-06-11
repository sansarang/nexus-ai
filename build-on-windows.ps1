$ErrorActionPreference = "Stop"
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

function Ok   { param($m) Write-Host "[OK] $m" -ForegroundColor Green }
function Warn { param($m) Write-Host "[!!] $m" -ForegroundColor Yellow }
function Fail { param($m) Write-Host "[XX] $m" -ForegroundColor Red; Read-Host "Press Enter to exit"; exit 1 }
function Step { param($m) Write-Host "" ; Write-Host ">>> $m" -ForegroundColor Cyan }

Write-Host ""
Write-Host "  ========================================" -ForegroundColor Cyan
Write-Host "    NEXUS AI  --  Windows Build v2.7     " -ForegroundColor Cyan
Write-Host "  ========================================" -ForegroundColor Cyan
Write-Host ""

# ── Shared Drive 감지 → 로컬로 복사 ─────────────────────────────
$isShared = ($ScriptDir -like "C:\Mac\*") -or ($ScriptDir -like "\\*") -or ((Split-Path -Qualifier $ScriptDir) -ne "C:")
if ($isShared) {
    $local = "C:\NexusBuild"
    Warn "Shared drive detected ($ScriptDir). Copying to $local ..."
    if (Test-Path $local) { Remove-Item $local -Recurse -Force }
    robocopy $ScriptDir $local /E /XD node_modules target .git __pycache__ /NFL /NDL /NJH /NJS | Out-Null
    Ok "Copied to $local"
    Write-Host ""
    Write-Host "  >> C:\NexusBuild 에서 빌드를 시작합니다..." -ForegroundColor Cyan
    Start-Process powershell.exe -ArgumentList "-ExecutionPolicy Bypass -NoProfile -File `"$local\build-on-windows.ps1`"" -Wait -NoNewWindow
    exit $LASTEXITCODE
}

$Root = $ScriptDir
Set-Location $Root

# ── LLVM/clang PATH 설정 ─────────────────────────────────────────
$llvmCandidates = @(
    "C:\Program Files\LLVM\bin",
    "C:\LLVM\bin",
    "$env:USERPROFILE\scoop\apps\llvm\current\bin"
)
$llvmBin = $llvmCandidates | Where-Object { Test-Path $_ } | Select-Object -First 1
if ($llvmBin) {
    if ($env:PATH -notlike "*LLVM*") { $env:PATH = "$llvmBin;" + $env:PATH }
    $env:LIBCLANG_PATH = $llvmBin
    Ok "LLVM: $llvmBin"
} else {
    Warn "LLVM not found -- some Rust crates may fail"
    Warn "Install: winget install LLVM.LLVM"
}

# ── 아키텍처 감지 (Get-CimInstance 우선, WMI fallback) ───────────
Step "[0/4] Architecture detection"
try {
    $cpuArch = (Get-CimInstance -ClassName Win32_Processor | Select-Object -First 1).Architecture
} catch {
    try {
        $cpuArch = (Get-WmiObject Win32_Processor | Select-Object -First 1).Architecture
    } catch {
        $cpuArch = 9  # default x86_64
        Warn "CPU arch detection failed, defaulting to x86_64"
    }
}
# 12=ARM64, 9=x86_64, 0=x86
if ($cpuArch -eq 12) {
    $rustTarget = "aarch64-pc-windows-msvc"
    $goArch = "arm64"
} else {
    $rustTarget = "x86_64-pc-windows-msvc"
    $goArch = "amd64"
}
Ok "CPU arch=$cpuArch  Rust=$rustTarget  Go=$goArch"

# ── MSVC 링커 탐색 (BuildTools + Community + Enterprise) ─────────
$vsBases = @(
    "C:\Program Files (x86)\Microsoft Visual Studio\2022\BuildTools\VC\Tools\MSVC",
    "C:\Program Files\Microsoft Visual Studio\2022\BuildTools\VC\Tools\MSVC",
    "C:\Program Files\Microsoft Visual Studio\2022\Community\VC\Tools\MSVC",
    "C:\Program Files\Microsoft Visual Studio\2022\Enterprise\VC\Tools\MSVC",
    "C:\Program Files\Microsoft Visual Studio\2022\Professional\VC\Tools\MSVC",
    "C:\Program Files (x86)\Microsoft Visual Studio\2019\BuildTools\VC\Tools\MSVC"
)
$vsBase = $vsBases | Where-Object { Test-Path $_ } | Select-Object -First 1
if (-not $vsBase) { Fail "Visual Studio Build Tools not found. Run: winget install Microsoft.VisualStudio.2022.BuildTools" }

$msvcVer = Get-ChildItem $vsBase | Sort-Object Name -Descending | Select-Object -First 1 -ExpandProperty Name
if ($cpuArch -eq 12) {
    $linkerCandidates = @(
        "$vsBase\$msvcVer\bin\HostARM64\arm64\link.exe",
        "$vsBase\$msvcVer\bin\Hostx64\arm64\link.exe"
    )
} else {
    $linkerCandidates = @("$vsBase\$msvcVer\bin\Hostx64\x64\link.exe")
}
$linker = $linkerCandidates | Where-Object { Test-Path $_ } | Select-Object -First 1
if (-not $linker) { Fail "link.exe not found in $vsBase\$msvcVer -- run VS Installer and add C++ build tools" }
$ar = $linker -replace "link\.exe$","lib.exe"
Ok "Linker: $linker"

# ── .cargo/config.toml 작성 ──────────────────────────────────────
New-Item -ItemType Directory -Force -Path "$Root\.cargo" | Out-Null
$lEsc = $linker -replace "\\","\\\\"
$aEsc = $ar    -replace "\\","\\\\"
Set-Content "$Root\.cargo\config.toml" "[target.$rustTarget]`r`nlinker = `"$lEsc`"`r`nar = `"$aEsc`"`r`n" -Encoding UTF8
Ok ".cargo\config.toml written"

# ── [1/4] Rust 타겟 설치 ─────────────────────────────────────────
Step "[1/4] Rust target install"
rustup target add $rustTarget
if ($LASTEXITCODE -ne 0) { Fail "rustup failed" }
Ok "$rustTarget installed"

# ── [2/4] Go 백엔드 빌드 ─────────────────────────────────────────
Step "[2/4] Go backend build"
New-Item -ItemType Directory -Force -Path "$Root\src-tauri\binaries" | Out-Null
New-Item -ItemType Directory -Force -Path "$Root\src-tauri\backend-bin" | Out-Null

$env:GOOS = "windows"
$env:GOARCH = $goArch
$env:CGO_ENABLED = "0"

Push-Location "$Root\backend"
go mod tidy
if ($LASTEXITCODE -ne 0) { Pop-Location; Fail "go mod tidy failed" }

$goOut = "$Root\src-tauri\binaries\backend-$rustTarget.exe"
go build -tags windows -ldflags="-s -w" -o $goOut .
if ($LASTEXITCODE -ne 0) { Pop-Location; Fail "Go build failed" }
Pop-Location

Copy-Item $goOut "$Root\src-tauri\backend-bin\nexus-backend.exe" -Force
$goSize = (Get-Item $goOut).Length / 1MB
Ok ("Go backend: {0:F1}MB" -f $goSize)

# ── [2.5/4] Python 사이드카 빌드 (자동화·녹화기의 실제 엔진 — 없으면 그 기능 전부 죽음) ──
$pythonExe = "$Root\src-tauri\backend-bin\nexus-python.exe"
$pythonSrc = "$Root\backend-bin\nexus-python.exe"   # build_python.ps1 출력 경로

# 1) 아직 안 빌드됐으면 PyInstaller로 빌드 (python + pip 필요)
if (-not (Test-Path $pythonSrc)) {
    Step "[2.5/4] Python sidecar build (pywinauto/pynput 포함)"
    & powershell -ExecutionPolicy Bypass -File "$Root\backend\nexus_python\build_python.ps1"
    if ($LASTEXITCODE -ne 0) { Warn "Python sidecar 빌드 실패 — 아래 stub 폴백" }
}

# 2) 산출물 복사 — 실패 시에만 stub 폴백 (경고 강하게)
if (Test-Path $pythonSrc) {
    Copy-Item $pythonSrc $pythonExe -Force
    $pySize = (Get-Item $pythonExe).Length / 1MB
    Ok ("nexus-python.exe (real sidecar, {0:F0}MB)" -f $pySize)
} else {
    Copy-Item "$Root\src-tauri\backend-bin\nexus-backend.exe" $pythonExe -Force
    Warn "⚠️⚠️ nexus-python.exe = STUB — 자동화/녹화/OCR 전부 비활성! python+pip 설치 후 재실행 필요"
}

# ── [3/4] npm 의존성 ─────────────────────────────────────────────
Step "[3/4] npm install"
if (-not (Test-Path "$Root\node_modules")) {
    npm install
    if ($LASTEXITCODE -ne 0) { Fail "npm install failed" }
} else {
    Ok "node_modules already present (skipping)"
}

# ── [4/4] Tauri 빌드 ─────────────────────────────────────────────
Step "[4/4] Tauri build (10~20 min first run)"
Write-Host "      Rust crate download included -- please wait" -ForegroundColor DarkGray
npx tauri build --target $rustTarget
if ($LASTEXITCODE -ne 0) { Fail "Tauri build failed" }

# ── 결과 출력 ────────────────────────────────────────────────────
$bundleDir = "$Root\src-tauri\target\$rustTarget\release\bundle\nsis"
Write-Host ""
Write-Host "  ======================================" -ForegroundColor Green
Write-Host "    BUILD SUCCESS!" -ForegroundColor Green
Write-Host "  ======================================" -ForegroundColor Green
if (Test-Path $bundleDir) {
    $installers = Get-ChildItem $bundleDir -Filter "*.exe"
    $installers | ForEach-Object {
        $sz = $_.Length / 1MB
        Write-Host ("  {0}  ({1:F0}MB)" -f $_.Name, $sz) -ForegroundColor White
    }
    Write-Host ""
    Invoke-Item $bundleDir
} else {
    Warn "NSIS dir not found: $bundleDir"
    $rawExe = "$Root\src-tauri\target\$rustTarget\release\Nexus.exe"
    if (Test-Path $rawExe) { Write-Host "  Raw exe: $rawExe" -ForegroundColor White }
}
Write-Host ""
Read-Host "Press Enter to exit"
