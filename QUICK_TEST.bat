@echo off
chcp 65001 >nul 2>&1
title Nexus AI — 빠른 테스트

:: ════════════════════════════════════════════════
::  QUICK_TEST.bat
::  Go 백엔드만으로 즉시 테스트 (Rust/Node 불필요)
::  nexus-backend.exe 하나만 있으면 됩니다.
:: ════════════════════════════════════════════════

set "ROOT=%~dp0"
set "ROOT=%ROOT:~0,-1%"
set "BACKEND=%ROOT%\nexus-backend.exe"

:: backend.exe가 같은 폴더에 있는지 확인
if not exist "%BACKEND%" (
    echo.
    echo  [오류] nexus-backend.exe 파일이 없습니다.
    echo.
    echo  해결방법:
    echo  1. Mac에서 Go 백엔드를 빌드해서 이 폴더에 복사하세요.
    echo  2. 또는 SETUP.bat을 실행해서 전체 빌드를 진행하세요.
    echo.
    pause
    exit /b 1
)

echo.
echo  Nexus AI 백엔드 시작 중...

:: 기존 프로세스 종료
taskkill /f /im nexus-backend.exe >nul 2>&1

:: 백엔드 실행
start /min "" "%BACKEND%"

:: 3초 대기 후 API 확인
timeout /t 3 /nobreak >nul

:: 헬스체크
curl -s http://127.0.0.1:17891/api/health >nul 2>&1
if %errorLevel%==0 (
    echo  [OK] Nexus 백엔드 정상 실행중!
    echo.
    echo  브라우저에서 아래 주소로 접속해서 테스트하세요:
    echo.
    echo  http://127.0.0.1:17891/api/health
    echo.
    start "" "http://127.0.0.1:17891/api/health"
) else (
    echo  [!] 백엔드가 아직 시작 중입니다. 5초 후 재시도...
    timeout /t 5 /nobreak >nul
    curl -s http://127.0.0.1:17891/api/health >nul 2>&1
    if %errorLevel%==0 (
        echo  [OK] Nexus 백엔드 실행됨!
        start "" "http://127.0.0.1:17891/api/health"
    ) else (
        echo  [오류] 백엔드 실행 실패. 로그를 확인하세요.
    )
)

pause
