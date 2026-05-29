$ErrorActionPreference = "Stop"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

function Ok   { param($m) Write-Host "[OK] $m" -ForegroundColor Green }
function Fail { param($m) Write-Host "[!!] $m" -ForegroundColor Red; Read-Host "Press Enter"; exit 1 }

# Copy to local drive if running from shared folder
$isShared = ($ScriptDir -like "C:\Mac\*") -or ((Split-Path -Qualifier $ScriptDir) -ne "C:")
if ($isShared) {
    $local = "C:\NexusBuild"
    Write-Host "Shared drive detected. Copying to $local ..." -ForegroundColor Yellow
    if (Test-Path $local) { Remove-Item $local -Recurse -Force }
    robocopy $ScriptDir $local /E /XD node_modules target .git /NFL /NDL /NJH /NJS | Out-Null
    Ok "Copied to $local"
    & powershell -ExecutionPolicy Bypass -File "$local\build-on-windows.ps1"
    exit $LASTEXITCODE
}

$Root = $ScriptDir
Set-Location $Root

# Ensure LLVM/clang is in PATH (required by some Rust crates via cc-rs)
$llvmBin = "C:\Program Files\LLVM\bin"
if (Test-Path $llvmBin) {
    if ($env:PATH -notlike "*LLVM*") {
        $env:PATH = "$llvmBin;" + $env:PATH
    }
    $env:LIBCLANG_PATH = $llvmBin
    Ok "LLVM clang: $llvmBin"
} else {
    Write-Host "[WARN] LLVM not found at $llvmBin — some crates may fail" -ForegroundColor Yellow
}

# Detect architecture
$osArch = (Get-WmiObject Win32_Processor | Select-Object -First 1).Architecture
# 9=x86_64, 12=ARM64
if ($osArch -eq 12) {
    $rustTarget = "aarch64-pc-windows-msvc"
    $goArch = "arm64"
} else {
    $rustTarget = "x86_64-pc-windows-msvc"
    $goArch = "amd64"
}
Write-Host "Arch=$osArch  Target=$rustTarget  GoArch=$goArch" -ForegroundColor Yellow

# Find MSVC linker
$vsBase = "C:\Program Files (x86)\Microsoft Visual Studio\2022\BuildTools\VC\Tools\MSVC"
if (-not (Test-Path $vsBase)) { Fail "Visual Studio Build Tools not found" }
$msvcVer = Get-ChildItem $vsBase | Sort-Object Name -Descending | Select-Object -First 1 -ExpandProperty Name

if ($osArch -eq 12) {
    $candidates = @(
        "$vsBase\$msvcVer\bin\HostARM64\arm64\link.exe",
        "$vsBase\$msvcVer\bin\Hostx64\arm64\link.exe"
    )
} else {
    $candidates = @("$vsBase\$msvcVer\bin\Hostx64\x64\link.exe")
}
$linker = $candidates | Where-Object { Test-Path $_ } | Select-Object -First 1
if (-not $linker) { Fail "Linker not found. Install ARM64 build tools from VS Installer." }
$ar = $linker -replace "link\.exe$","lib.exe"
Ok "Linker: $linker"

# Write .cargo/config.toml
New-Item -ItemType Directory -Force -Path "$Root\.cargo" | Out-Null
$lEsc = $linker -replace "\\","\\\\"
$aEsc = $ar -replace "\\","\\\\"
Set-Content "$Root\.cargo\config.toml" "[target.$rustTarget]`r`nlinker = `"$lEsc`"`r`nar = `"$aEsc`"`r`n" -Encoding UTF8
Ok ".cargo\config.toml written"

# Rust target
Write-Host "`n[1/4] Rust target..." -ForegroundColor Cyan
rustup target add $rustTarget
Ok "$rustTarget ready"

# Go backend
Write-Host "`n[2/4] Go backend build..." -ForegroundColor Cyan
New-Item -ItemType Directory -Force -Path "$Root\src-tauri\binaries" | Out-Null
$env:GOOS="windows"; $env:GOARCH=$goArch; $env:CGO_ENABLED="0"
Push-Location "$Root\backend"
go build -ldflags="-s -w" -o "..\src-tauri\binaries\backend-$rustTarget.exe" .
if ($LASTEXITCODE -ne 0) { Pop-Location; Fail "Go build failed" }
Pop-Location
Ok "Go backend done"

# npm install
Write-Host "`n[3/4] npm install..." -ForegroundColor Cyan
if (Test-Path "$Root\node_modules") {
    Write-Host "Removing old node_modules..." -ForegroundColor Yellow
    Remove-Item "$Root\node_modules" -Recurse -Force
}
npm install
if ($LASTEXITCODE -ne 0) { Fail "npm install failed" }
Ok "npm done"

# Tauri build
Write-Host "`n[4/4] Tauri build (10-20 min)..." -ForegroundColor Cyan
npx tauri build --target $rustTarget
if ($LASTEXITCODE -ne 0) { Fail "Tauri build failed" }

# Result
$bundleDir = "$Root\src-tauri\target\$rustTarget\release\bundle\nsis"
Write-Host "`n=== BUILD SUCCESS ===" -ForegroundColor Green
if (Test-Path $bundleDir) {
    Get-ChildItem $bundleDir -Filter "*.exe" | ForEach-Object {
        Write-Host "  $($_.Name)" -ForegroundColor White
    }
    Invoke-Item $bundleDir
}
Read-Host "Press Enter to exit"
