# Nexus AI - 자동 빌드 스크립트 (ARM64/x64 자동 감지)
# 실행 방법: 이 파일을 우클릭 -> "PowerShell로 실행"

$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot

Write-Host "=====================================================" -ForegroundColor Cyan
Write-Host "  Nexus AI 자동 빌드" -ForegroundColor Cyan
Write-Host "=====================================================" -ForegroundColor Cyan

# ── 1. 아키텍처 감지 ──────────────────────────────────────────
$arch = (Get-WmiObject Win32_Processor).Architecture
# 9 = ARM64, 0 = x86, 9 with x64 capability
$cpuArch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture
Write-Host "`n[1/5] OS 아키텍처: $cpuArch" -ForegroundColor Yellow

if ($cpuArch -eq "Arm64") {
    $rustTarget = "aarch64-pc-windows-msvc"
    $goArch = "arm64"
    $hostArch = "arm64"
} else {
    $rustTarget = "x86_64-pc-windows-msvc"
    $goArch = "amd64"
    $hostArch = "x64"
}
Write-Host "   Rust 타겟: $rustTarget" -ForegroundColor Green
Write-Host "   Go GOARCH: $goArch" -ForegroundColor Green

# ── 2. MSVC 링커 자동 탐색 ───────────────────────────────────
Write-Host "`n[2/5] MSVC 링커 탐색..." -ForegroundColor Yellow

$vsBase = "C:\Program Files (x86)\Microsoft Visual Studio\2022\BuildTools\VC\Tools\MSVC"
$msvcVer = Get-ChildItem $vsBase | Sort-Object Name -Descending | Select-Object -First 1 -ExpandProperty Name
Write-Host "   MSVC 버전: $msvcVer" -ForegroundColor Green

# ARM64 링커 경로 (Hostx64에서 arm64 크로스, 또는 HostARM64 네이티브)
if ($cpuArch -eq "Arm64") {
    $linkerCandidates = @(
        "$vsBase\$msvcVer\bin\HostARM64\arm64\link.exe",
        "$vsBase\$msvcVer\bin\Hostx64\arm64\link.exe"
    )
} else {
    $linkerCandidates = @(
        "$vsBase\$msvcVer\bin\Hostx64\x64\link.exe"
    )
}

$linkerPath = $null
foreach ($candidate in $linkerCandidates) {
    if (Test-Path $candidate) {
        $linkerPath = $candidate
        break
    }
}

if (-not $linkerPath) {
    Write-Host "`n[오류] 링커를 찾을 수 없습니다." -ForegroundColor Red
    Write-Host "Visual Studio Installer -> 수정 -> 개별 구성 요소 에서" -ForegroundColor Red
    if ($cpuArch -eq "Arm64") {
        Write-Host "'MSVC v143 VS 2022 C++ ARM64 빌드 도구' 설치 후 재실행하세요." -ForegroundColor Red
    } else {
        Write-Host "'MSVC v143 VS 2022 C++ x64/x86 빌드 도구' 설치 후 재실행하세요." -ForegroundColor Red
    }
    Read-Host "Enter 키를 눌러 종료"
    exit 1
}
Write-Host "   링커: $linkerPath" -ForegroundColor Green

# ── 3. cargo config 자동 생성 ────────────────────────────────
Write-Host "`n[3/5] Cargo 링커 설정 중..." -ForegroundColor Yellow

$cargoDir = "$PSScriptRoot\.cargo"
New-Item -ItemType Directory -Force -Path $cargoDir | Out-Null

$linkerEscaped = $linkerPath -replace "\\", "\\\\"
$arPath = $linkerPath -replace "link.exe", "lib.exe"
$arEscaped = $arPath -replace "\\", "\\\\"

$cargoConfig = @"
[target.aarch64-pc-windows-msvc]
linker = "$linkerEscaped"
ar = "$arEscaped"

[target.x86_64-pc-windows-msvc]
linker = "$($linkerPath -replace "arm64\\link.exe","x64\link.exe" -replace "\\","\\\\")".Replace("arm64", "x64").Replace("ARM64","x64")
"@

# 더 간단하게 작성
$cargoConfigSimple = "[target.$rustTarget]`r`nlinker = `"$linkerEscaped`"`r`nar = `"$arEscaped`"`r`n"
Set-Content -Path "$cargoDir\config.toml" -Value $cargoConfigSimple -Encoding UTF8
Write-Host "   $cargoDir\config.toml 생성 완료" -ForegroundColor Green

# ── 4. Rust 타겟 설치 ────────────────────────────────────────
Write-Host "`n[4/5] Rust 타겟 설치 중..." -ForegroundColor Yellow
rustup target add $rustTarget
Write-Host "   $rustTarget 준비 완료" -ForegroundColor Green

# ── 5. Go 백엔드 빌드 ────────────────────────────────────────
Write-Host "`n[5/5] Go 백엔드 빌드..." -ForegroundColor Yellow

if (-not (Test-Path "$PSScriptRoot\src-tauri\binaries")) {
    New-Item -ItemType Directory -Force -Path "$PSScriptRoot\src-tauri\binaries" | Out-Null
}

$env:GOOS = "windows"
$env:GOARCH = $goArch
$env:CGO_ENABLED = "0"

Push-Location "$PSScriptRoot\backend"
$backendOut = "..\src-tauri\binaries\backend-$rustTarget.exe"
& go build -ldflags="-s -w" -o $backendOut .
if ($LASTEXITCODE -ne 0) {
    Write-Host "[오류] Go 빌드 실패" -ForegroundColor Red
    Pop-Location
    Read-Host "Enter 키를 눌러 종료"
    exit 1
}
Pop-Location
Write-Host "   Go 백엔드 빌드 완료" -ForegroundColor Green

# ── 6. npm install ────────────────────────────────────────────
Write-Host "`n[6/6] npm install..." -ForegroundColor Yellow
& npm install
if ($LASTEXITCODE -ne 0) {
    Write-Host "[오류] npm install 실패" -ForegroundColor Red
    Read-Host "Enter 키를 눌러 종료"
    exit 1
}
Write-Host "   npm install 완료" -ForegroundColor Green

# ── 7. Tauri 빌드 ────────────────────────────────────────────
Write-Host "`n[7/7] Tauri 빌드 (10~20분)..." -ForegroundColor Yellow
Write-Host "   진행상황이 아래에 표시됩니다..." -ForegroundColor Gray

& npx tauri build --target $rustTarget
if ($LASTEXITCODE -ne 0) {
    Write-Host "`n[오류] Tauri 빌드 실패" -ForegroundColor Red
    Read-Host "Enter 키를 눌러 종료"
    exit 1
}

# ── 결과 ──────────────────────────────────────────────────────
Write-Host "`n=====================================================" -ForegroundColor Green
Write-Host "  빌드 성공!" -ForegroundColor Green
Write-Host "=====================================================" -ForegroundColor Green

$bundleDir = "$PSScriptRoot\src-tauri\target\$rustTarget\release\bundle\nsis"
Write-Host "`n배포 파일:" -ForegroundColor Cyan
Get-ChildItem "$bundleDir\*_setup.exe"    -ErrorAction SilentlyContinue | ForEach-Object { Write-Host "  [설치형] $($_.Name)  ($([math]::Round($_.Length/1MB,1)) MB)" -ForegroundColor White }
Get-ChildItem "$bundleDir\*_portable.exe" -ErrorAction SilentlyContinue | ForEach-Object { Write-Host "  [포터블] $($_.Name)  ($([math]::Round($_.Length/1MB,1)) MB)" -ForegroundColor White }

if (Test-Path $bundleDir) {
    Invoke-Item $bundleDir
}

Read-Host "`nEnter 키를 눌러 종료"
