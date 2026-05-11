@echo off
chcp 65001 >nul 2>&1
setlocal EnableDelayedExpansion
title Nexus AI — 자동 설치 및 빌드

:: ════════════════════════════════════════════════
::  Nexus AI 자동 빌드 스크립트
::  더블클릭 하나로 설치부터 실행까지 자동 처리
:: ════════════════════════════════════════════════

cls
echo.
echo  ██████████████████████████████████████████
echo  ██   Nexus AI 자동 설치 프로그램       ██
echo  ██████████████████████████████████████████
echo.
echo  이 창이 열려있는 동안 설치가 진행됩니다.
echo  창을 닫지 마세요!
echo.

:: 관리자 권한 확인
net session >nul 2>&1
if %errorLevel% neq 0 (
    echo  [!] 관리자 권한으로 다시 실행합니다...
    powershell -Command "Start-Process '%~f0' -Verb RunAs"
    exit /b
)

set "ROOT=%~dp0"
set "ROOT=%ROOT:~0,-1%"

:: ── 도구 설치 확인 및 자동 설치 ──────────────────
echo  [1/5] 필수 도구 확인 중...

:: winget 있는지 확인
where winget >nul 2>&1
if %errorLevel% neq 0 (
    echo  [오류] winget이 없습니다.
    echo         Windows 10 2004 이상이 필요합니다.
    echo         Microsoft Store에서 "앱 설치 관리자"를 설치해주세요.
    pause
    exit /b 1
)

:: Go
where go >nul 2>&1
if %errorLevel% neq 0 (
    echo  [설치] Go 설치 중...
    winget install -e --id GoLang.Go --silent --accept-package-agreements --accept-source-agreements
    if !errorLevel! neq 0 (
        echo  [오류] Go 설치 실패. 수동으로 https://go.dev/dl/ 에서 설치해주세요.
        pause
        exit /b 1
    )
    :: 환경변수 새로고침
    call :RefreshPath
)

:: Node.js
where node >nul 2>&1
if %errorLevel% neq 0 (
    echo  [설치] Node.js 설치 중...
    winget install -e --id OpenJS.NodeJS.LTS --silent --accept-package-agreements --accept-source-agreements
    if !errorLevel! neq 0 (
        echo  [오류] Node.js 설치 실패.
        pause
        exit /b 1
    )
    call :RefreshPath
)

:: Rust
where rustc >nul 2>&1
if %errorLevel% neq 0 (
    echo  [설치] Rust 설치 중... (5~10분 소요)
    winget install -e --id Rustlang.Rustup --silent --accept-package-agreements --accept-source-agreements
    if !errorLevel! neq 0 (
        echo  [오류] Rust 설치 실패.
        pause
        exit /b 1
    )
    call :RefreshPath
)

:: Visual C++ Build Tools (Rust MSVC 컴파일러용)
where cl >nul 2>&1
if %errorLevel% neq 0 (
    echo  [설치] Visual C++ Build Tools 설치 중... (5~10분 소요)
    winget install -e --id Microsoft.VisualStudio.2022.BuildTools --silent --accept-package-agreements --accept-source-agreements ^
        --override "--quiet --add Microsoft.VisualStudio.Workload.VCTools --includeRecommended"
    call :RefreshPath
)

echo  [OK] 모든 도구 준비됨
echo.

:: ── STEP 2: 환경변수 최종 확인 ───────────────────
echo  [2/5] 환경 확인 중...

:: PATH에 Go, Node, Rust 추가
set "PATH=%PATH%;%LOCALAPPDATA%\Programs\Go\bin;%ProgramFiles%\Go\bin"
set "PATH=%PATH%;%ProgramFiles%\nodejs;%APPDATA%\npm"
set "PATH=%PATH%;%USERPROFILE%\.cargo\bin"

:: 버전 출력
for /f "tokens=3" %%v in ('go version 2^>nul') do set "GOVER=%%v"
for /f %%v in ('node --version 2^>nul') do set "NODEVER=%%v"
for /f "tokens=2" %%v in ('rustc --version 2^>nul') do set "RUSTVER=%%v"

if "!GOVER!"=="" (
    echo  [!] Go를 찾을 수 없습니다. 재시작 후 다시 실행해주세요.
    echo      재시작 후 이 파일을 다시 더블클릭하세요.
    pause
    exit /b 1
)

echo  Go: !GOVER!
echo  Node: !NODEVER!
echo  Rust: !RUSTVER!
echo.

:: ── STEP 3: Go 백엔드 빌드 ───────────────────────
echo  [3/5] Nexus 백엔드 빌드 중...
cd /d "%ROOT%\backend"

go mod tidy
if %errorLevel% neq 0 (
    echo  [오류] go mod tidy 실패
    pause
    exit /b 1
)

set "BINDIR=%ROOT%\src-tauri\backend-bin"
if not exist "%BINDIR%" mkdir "%BINDIR%"

set "GOOS=windows"
set "GOARCH=amd64"
set "CGO_ENABLED=0"
go build -tags windows -ldflags="-s -w" -o "%BINDIR%\nexus-backend.exe" .
if %errorLevel% neq 0 (
    echo  [오류] Go 백엔드 빌드 실패
    pause
    exit /b 1
)

for %%f in ("%BINDIR%\nexus-backend.exe") do set "BSIZE=%%~zf"
set /a "BSIZEMB=%BSIZE%/1048576"
echo  [OK] 백엔드 빌드 완료: %BSIZEMB%MB
echo.

:: ── STEP 4: npm 의존성 + Tauri 빌드 ─────────────
echo  [4/5] 프론트엔드 의존성 설치 중...
cd /d "%ROOT%"

if not exist "node_modules" (
    npm install
    if !errorLevel! neq 0 (
        echo  [오류] npm install 실패
        pause
        exit /b 1
    )
)
echo  [OK] npm 의존성 준비됨
echo.
echo  [5/5] Nexus 전체 빌드 중...
echo        첫 빌드는 15~20분 소요됩니다. (Rust 다운로드)
echo        기다려주세요...
echo.

npm run tauri build
if %errorLevel% neq 0 (
    echo.
    echo  [오류] Tauri 빌드 실패
    echo  위 오류 메시지를 사진 찍어서 개발자에게 전달해주세요.
    pause
    exit /b 1
)

:: ── 결과 출력 ─────────────────────────────────────
cls
echo.
echo  ████████████████████████████████████████████
echo  ██                                        ██
echo  ██   ✅ Nexus 빌드 완료!                  ██
echo  ██                                        ██
echo  ████████████████████████████████████████████
echo.

set "NSIS_DIR=%ROOT%\src-tauri\target\release\bundle\nsis"
set "EXE_DIR=%ROOT%\src-tauri\target\release"

:: NSIS 인스톨러 찾기
for /f "delims=" %%f in ('dir /b /a-d "%NSIS_DIR%\*.exe" 2^>nul') do (
    set "INSTALLER=%NSIS_DIR%\%%f"
)

if defined INSTALLER (
    for %%f in ("!INSTALLER!") do set "ISIZE=%%~zf"
    set /a "ISIZEMB=!ISIZE!/1048576"
    echo  📦 인스톨러 파일 (권장):
    echo     !INSTALLER!
    echo     크기: !ISIZEMB!MB
    echo.
    echo  ▶ 위 파일을 더블클릭하면 Nexus가 설치됩니다.
) else (
    echo  📦 포터블 exe:
    echo     %EXE_DIR%\Nexus.exe
    echo.
    echo  ▶ 위 파일을 더블클릭하면 Nexus가 실행됩니다.
)

echo.
echo  결과 폴더를 열까요?
choice /c YN /m "Y=예  N=아니오"
if %errorLevel%==1 (
    if defined INSTALLER (
        explorer "!NSIS_DIR!"
    ) else (
        explorer "%EXE_DIR%"
    )
)

echo.
echo  지금 Nexus를 실행할까요?
choice /c YN /m "Y=예  N=아니오"
if %errorLevel%==1 (
    if defined INSTALLER (
        start "" "!INSTALLER!"
    ) else (
        start "" "%EXE_DIR%\Nexus.exe"
    )
)

pause
exit /b 0

:: ── 환경변수 새로고침 서브루틴 ───────────────────
:RefreshPath
for /f "skip=2 tokens=3*" %%a in ('reg query "HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Environment" /v Path 2^>nul') do (
    if "%%b"=="" (set "PATH=%%a") else (set "PATH=%%a %%b")
)
for /f "skip=2 tokens=3*" %%a in ('reg query "HKCU\Environment" /v Path 2^>nul') do (
    if "%%b"=="" (set "PATH=%PATH%;%%a") else (set "PATH=%PATH%;%%a %%b")
)
exit /b 0
