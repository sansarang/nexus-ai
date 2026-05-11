@echo off
chcp 65001 >nul
echo [Nexus] Building Go backend only...

if not exist "src-tauri\binaries" mkdir "src-tauri\binaries"

cd backend
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0

go build -ldflags="-s -w" -o "..\src-tauri\binaries\backend-x86_64-pc-windows-msvc.exe" .

if %errorlevel% equ 0 (
    echo [OK] Done: src-tauri\binaries\backend-x86_64-pc-windows-msvc.exe
) else (
    echo [ERROR] Go build failed - check errors above
)
cd ..
pause
