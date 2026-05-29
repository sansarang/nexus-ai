# ══════════════════════════════════════════════════════════════════
# Nexus Python Sidecar — PyInstaller 빌드 스크립트
# 실행: cd backend/nexus_python && powershell -ExecutionPolicy Bypass -File build_python.ps1
# 출력: backend-bin/nexus-python.exe
# ══════════════════════════════════════════════════════════════════

$ErrorActionPreference = "Stop"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RepoRoot  = Split-Path -Parent (Split-Path -Parent $ScriptDir)
$OutDir    = Join-Path $RepoRoot "backend-bin"

Write-Host "[nexus-python] 빌드 시작..." -ForegroundColor Cyan

# 의존성 설치
Write-Host "[nexus-python] pip install..." -ForegroundColor Yellow
pip install -r "$ScriptDir\requirements.txt" --quiet

# PyInstaller 설치 (없으면)
pip show pyinstaller | Out-Null
if ($LASTEXITCODE -ne 0) {
    pip install pyinstaller --quiet
}

# 출력 폴더 생성
New-Item -ItemType Directory -Path $OutDir -Force | Out-Null

# PyInstaller 실행
Write-Host "[nexus-python] PyInstaller 빌드 중..." -ForegroundColor Yellow
Set-Location $ScriptDir

pyinstaller `
    --onefile `
    --name nexus-python `
    --distpath "$OutDir" `
    --workpath "$ScriptDir\build_tmp" `
    --specpath "$ScriptDir" `
    --hidden-import "uvicorn.logging" `
    --hidden-import "uvicorn.loops.auto" `
    --hidden-import "uvicorn.protocols.http.auto" `
    --hidden-import "uvicorn.protocols.websockets.auto" `
    --hidden-import "uvicorn.lifespan.on" `
    --hidden-import "fastapi" `
    --hidden-import "yt_dlp" `
    --hidden-import "ytmusicapi" `
    --hidden-import "easyocr" `
    --hidden-import "fitz" `
    --hidden-import "pandas" `
    --hidden-import "yfinance" `
    --hidden-import "sentence_transformers" `
    --hidden-import "faiss" `
    --hidden-import "pyautogui" `
    --hidden-import "pygetwindow" `
    --hidden-import "groq" `
    --hidden-import "sklearn" `
    --collect-all "yt_dlp" `
    --collect-all "ytmusicapi" `
    --collect-all "sentence_transformers" `
    --noconfirm `
    "$ScriptDir\main.py"

if ($LASTEXITCODE -eq 0) {
    $size = [math]::Round((Get-Item "$OutDir\nexus-python.exe").Length / 1MB, 1)
    Write-Host "[nexus-python] 빌드 완료: $OutDir\nexus-python.exe ($size MB)" -ForegroundColor Green
} else {
    Write-Host "[nexus-python] 빌드 실패!" -ForegroundColor Red
    exit 1
}

# 임시 파일 정리
Remove-Item "$ScriptDir\build_tmp" -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item "$ScriptDir\nexus-python.spec" -Force -ErrorAction SilentlyContinue

Write-Host "[nexus-python] 완료." -ForegroundColor Cyan
