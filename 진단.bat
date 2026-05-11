@echo off
title NEXUS 진단
cd /d "%~dp0"

echo ==============================
echo  NEXUS 빌드 환경 진단
echo ==============================
echo.
echo 현재 폴더: %CD%
echo.

echo [Node.js]
where node 2>nul && node -v || echo   !! 미설치
echo.

echo [Go]
where go 2>nul && go version || echo   !! 미설치
echo.

echo [Rust]
where rustc 2>nul && rustc --version || echo   !! 미설치
echo.

echo [Cargo]
where cargo 2>nul && cargo --version || echo   !! 미설치
echo.

echo [npm]
where npm 2>nul && npm -v || echo   !! 미설치
echo.

echo [backend 폴더]
if exist backend\ (echo   OK) else echo   !! 없음
echo.

echo [src-tauri 폴더]
if exist src-tauri\ (echo   OK) else echo   !! 없음
echo.

echo [package.json]
if exist package.json (echo   OK) else echo   !! 없음
echo.

echo ==============================
echo  진단 완료. 이 창을 닫지 마세요.
echo  위 결과를 사진 찍어서 확인하세요.
echo ==============================
pause
