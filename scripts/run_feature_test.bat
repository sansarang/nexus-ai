@echo off
chcp 65001 >nul
setlocal
echo ============================================
echo   Nexus AI - 전체 기능 테스트
echo ============================================
echo.

where python >nul 2>nul
if errorlevel 1 (
  echo [!] Python이 설치되어 있지 않습니다.
  echo     https://www.python.org/downloads/ 에서 Python 3.x 설치 후 다시 실행하세요.
  echo.
  pause
  exit /b 1
)

echo [안내] 이 테스트는 Nexus 앱이 "실행 중"이어야 합니다.
echo        (트레이에 Nexus 아이콘이 떠 있으면 OK — 백엔드 127.0.0.1:17891 / 사이드카 17893)
echo.
echo  - openpyxl 이 있으면 xlsx 픽스처도 생성됩니다 (선택):  pip install openpyxl
echo.
pause

echo.
echo [1/1] 픽스처 생성 + 전체 기능 테스트 실행...
echo.
python "%~dp0feature_test.py" --setup --run

echo.
echo ============================================
echo  결과 해석: docs\automation\feature-test-guide.md
echo  - SKIP = Windows 전용 / API 키 미설정 / 자동화 엔진(UIA) 미준비 (정상)
echo ============================================
echo.
pause
endlocal
