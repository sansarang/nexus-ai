@echo off
title Nexus AI Build

cd /d "%~dp0"

:: MSVC environment (link.exe)
set VCVARS=C:\Program Files (x86)\Microsoft Visual Studio\2022\BuildTools\VC\Auxiliary\Build\vcvarsall.bat
if not exist "%VCVARS%" set VCVARS=C:\Program Files\Microsoft Visual Studio\2022\BuildTools\VC\Auxiliary\Build\vcvarsall.bat
if not exist "%VCVARS%" set VCVARS=C:\Program Files (x86)\Microsoft Visual Studio\2022\Community\VC\Auxiliary\Build\vcvarsall.bat
if not exist "%VCVARS%" (
    echo [ERROR] vcvarsall.bat not found. Install Visual Studio Build Tools with C++ workload.
    pause & exit /b 1
)
call "%VCVARS%" x64

echo.
echo [1/4] Checking tools...
where rustc >nul 2>&1 || (echo [ERROR] Rust not found. Install from https://rustup.rs & pause & exit /b 1)
where go   >nul 2>&1 || (echo [ERROR] Go not found. Install from https://go.dev/dl & pause & exit /b 1)
where node >nul 2>&1 || (echo [ERROR] Node.js not found. Install from https://nodejs.org & pause & exit /b 1)
rustup target add aarch64-pc-windows-msvc >nul 2>&1
echo [OK] Rust, Go, Node.js, MSVC ready

echo.
echo [2/4] Building Go backend...
if not exist "src-tauri\binaries" mkdir "src-tauri\binaries"
cd backend
set GOOS=windows
set GOARCH=arm64
set CGO_ENABLED=0
go build -ldflags="-s -w" -o "..\src-tauri\binaries\backend-aarch64-pc-windows-msvc.exe" .
if errorlevel 1 (
    echo [ERROR] Go build failed.
    cd ..
    pause & exit /b 1
)
cd ..
echo [OK] Go backend built

echo.
echo [3/4] npm install...
call npm install
if errorlevel 1 (
    echo [ERROR] npm install failed.
    pause & exit /b 1
)
echo [OK] npm install done

echo.
echo [4/4] Tauri build (10-20 min)...
if exist "node_modules\.bin\tauri.cmd" (
    call node_modules\.bin\tauri build --target aarch64-pc-windows-msvc
) else (
    call npx --yes tauri build --target aarch64-pc-windows-msvc
)
if errorlevel 1 (
    echo [ERROR] Tauri build failed.
    pause & exit /b 1
)

echo.
echo ============================================================
echo  BUILD SUCCESS
echo ============================================================
set BUNDLE=src-tauri\target\aarch64-pc-windows-msvc\release\bundle\nsis
echo.
echo [Installer]
for %%f in ("%BUNDLE%\*_setup.exe") do echo   %%~nxf  -- %%~ff
echo.
echo [Portable]
for %%f in ("%BUNDLE%\*_portable.exe") do echo   %%~nxf  -- %%~ff
echo.
if exist "%BUNDLE%" explorer "%~dp0%BUNDLE%"
pause
