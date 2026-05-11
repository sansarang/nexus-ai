@echo off
title NEXUS AI Build
cd /d "%~dp0"
setlocal enabledelayedexpansion

echo.
echo ================================================
echo   NEXUS AI - Windows Build
echo   Windows 10 / 11  x64 / ARM64
echo ================================================
echo.

:: Detect CPU architecture
set ARCH=x64
set RUST_TARGET=x86_64-pc-windows-msvc
set GO_ARCH=amd64
if /i "%PROCESSOR_ARCHITECTURE%"=="ARM64" (
    set ARCH=arm64
    set RUST_TARGET=aarch64-pc-windows-msvc
    set GO_ARCH=arm64
)
echo Architecture: %ARCH%
echo Rust target : %RUST_TARGET%
echo.

:: Load MSVC environment
echo [0/5] Loading MSVC...
set VCVARS=
if exist "C:\Program Files\Microsoft Visual Studio\2022\BuildTools\VC\Auxiliary\Build\vcvarsall.bat" (
    set VCVARS=C:\Program Files\Microsoft Visual Studio\2022\BuildTools\VC\Auxiliary\Build\vcvarsall.bat
)
if exist "C:\Program Files (x86)\Microsoft Visual Studio\2022\BuildTools\VC\Auxiliary\Build\vcvarsall.bat" (
    set VCVARS=C:\Program Files (x86)\Microsoft Visual Studio\2022\BuildTools\VC\Auxiliary\Build\vcvarsall.bat
)
if exist "C:\Program Files\Microsoft Visual Studio\2022\Community\VC\Auxiliary\Build\vcvarsall.bat" (
    set VCVARS=C:\Program Files\Microsoft Visual Studio\2022\Community\VC\Auxiliary\Build\vcvarsall.bat
)
if defined VCVARS (
    call "%VCVARS%" %ARCH% >nul 2>&1
    echo     MSVC: OK
) else (
    echo     WARNING: MSVC not found. Tauri build may fail.
    echo     Install: winget install Microsoft.VisualStudio.2022.BuildTools
    echo.
)

:: Check required tools
echo [1/5] Checking build tools...
set MISSING=0

where rustc >nul 2>&1
if errorlevel 1 (
    echo     [ERROR] Rust not installed - https://rustup.rs
    set MISSING=1
) else (
    for /f "tokens=2" %%v in ('rustc --version') do echo     Rust:   %%v
)

where go >nul 2>&1
if errorlevel 1 (
    echo     [ERROR] Go not installed - https://go.dev/dl/
    set MISSING=1
) else (
    for /f "tokens=3" %%v in ('go version') do echo     Go:     %%v
)

where node >nul 2>&1
if errorlevel 1 (
    echo     [ERROR] Node.js not installed - https://nodejs.org
    set MISSING=1
) else (
    for /f %%v in ('node -v') do echo     Node:   %%v
)

if "%MISSING%"=="1" (
    echo.
    echo Install the missing tools above, then run this script again.
    pause & exit /b 1
)

:: Add Rust target
rustup target add %RUST_TARGET% >nul 2>&1
echo     Rust target %RUST_TARGET% ready
echo.

:: Build Go backend
echo [2/5] Building Go backend...
if not exist "src-tauri\backend-bin" mkdir "src-tauri\backend-bin"

cd backend
set CGO_ENABLED=0
set GOOS=windows
set GOARCH=%GO_ARCH%

go build -ldflags="-s -w -H windowsgui" -o "..\src-tauri\backend-bin\nexus-backend.exe" . 2>"..\go_build_error.log"
if errorlevel 1 (
    echo     [ERROR] Go backend build failed
    echo     Error details:
    type "..\go_build_error.log"
    cd ..
    pause & exit /b 1
)
cd ..

copy /y "src-tauri\backend-bin\nexus-backend.exe" "nexus-backend.exe" >nul
echo     nexus-backend.exe built OK (embedded in src-tauri\backend-bin\)
echo.

:: Install npm packages
echo [3/5] Installing Node.js packages...
if not exist "node_modules" (
    call npm install
    if errorlevel 1 (
        echo     [ERROR] npm install failed
        pause & exit /b 1
    )
) else (
    echo     node_modules exists - skipping (delete node_modules for a clean build)
)
echo.

:: Build frontend
echo [4/5] Building frontend (Vite + TypeScript)...
call npm run build
if errorlevel 1 (
    echo     [ERROR] Frontend build failed
    pause & exit /b 1
)
echo     dist/ created OK
echo.

:: Package Tauri app
echo [5/5] Packaging Tauri app... (first build: 10-20 min)
echo       Please wait while Rust crates are downloaded.
echo.

if exist "node_modules\.bin\tauri.cmd" (
    call node_modules\.bin\tauri build --target %RUST_TARGET%
) else (
    call npx --yes @tauri-apps/cli build --target %RUST_TARGET%
)
if errorlevel 1 (
    echo.
    echo     [ERROR] Tauri build failed
    echo     Common causes:
    echo       1. NSIS not installed  - winget install NSIS.NSIS
    echo       2. WebView2 missing    - https://developer.microsoft.com/microsoft-edge/webview2/
    echo       3. icon.ico missing    - src-tauri\icons\icon.ico required
    pause & exit /b 1
)

:: Show build results
echo.
echo ================================================
echo   BUILD SUCCESS!
echo ================================================
echo.

set BUNDLE=src-tauri\target\%RUST_TARGET%\release\bundle\nsis

echo Output files:
if exist "%BUNDLE%" (
    for %%f in ("%BUNDLE%\*.exe") do (
        echo     %%~nxf
        echo     Path: %%~ff
        echo.
    )
    echo Files to upload to GitHub Releases:
    for %%f in ("%BUNDLE%\*setup*.exe") do echo     - %%~nxf
) else (
    echo     Check folder: %BUNDLE%
)

echo.
set /p OPEN=Open build folder in Explorer? [Y/N]:
if /i "%OPEN%"=="Y" (
    if exist "%BUNDLE%" explorer "%~dp0%BUNDLE%"
)
echo.
echo Done!
pause
