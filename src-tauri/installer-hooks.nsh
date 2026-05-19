; ══════════════════════════════════════════════════════════════════
; Nexus AI — NSIS 커스텀 설치 훅
; 설치 시 필수 의존성 자동 확인 및 설치
; ══════════════════════════════════════════════════════════════════
!include "LogicLib.nsh"
!include "FileFunc.nsh"

!macro customInstall

  ; ── 1. Microsoft Visual C++ Redistributable 확인 ─────────────────
  DetailPrint "Checking Visual C++ Redistributable..."
  ReadRegStr $0 HKLM "SOFTWARE\Microsoft\VisualStudio\14.0\VC\Runtimes\X64" "Installed"
  StrCmp $0 "1" VCRedistOk 0
    DetailPrint "Installing Visual C++ Redistributable 2015-2022..."
    ; 윈도우 기본 내장 VC++ 설치 (winget 사용)
    nsExec::ExecToStack 'powershell -WindowStyle Hidden -Command "winget install --id Microsoft.VCRedist.2015+.x64 -e --silent --accept-package-agreements --accept-source-agreements 2>$null; exit 0"'
    Pop $0
  VCRedistOk:

  ; ── 2. Google Chrome 확인 및 설치 ────────────────────────────────
  DetailPrint "Checking Google Chrome..."
  ; 레지스트리에서 Chrome 설치 여부 확인
  ReadRegStr $1 HKLM "SOFTWARE\Google\Chrome\BLBeacon" "version"
  StrCmp $1 "" 0 ChromeOk
    ReadRegStr $1 HKCU "SOFTWARE\Google\Chrome\BLBeacon" "version"
    StrCmp $1 "" 0 ChromeOk
      ; Program Files에서도 확인
      ${If} ${FileExists} "$PROGRAMFILES\Google\Chrome\Application\chrome.exe"
        StrCpy $1 "found"
      ${EndIf}
      ${If} ${FileExists} "$PROGRAMFILES64\Google\Chrome\Application\chrome.exe"
        StrCpy $1 "found"
      ${EndIf}
      StrCmp $1 "" 0 ChromeOk
        MessageBox MB_YESNO|MB_ICONQUESTION \
          "Nexus의 웹 자동화 기능(가격 비교·크롤링·딥서치)을 사용하려면 Google Chrome이 필요합니다.$\n$\n자동으로 Chrome을 설치할까요? (무료)" \
          IDYES InstallChrome IDNO ChromeSkip
        InstallChrome:
          DetailPrint "Installing Google Chrome..."
          nsExec::ExecToStack 'powershell -WindowStyle Hidden -Command "winget install --id Google.Chrome -e --silent --accept-package-agreements --accept-source-agreements 2>$null; exit 0"'
          Pop $0
          StrCmp $0 "0" ChromeOk 0
            ; winget 실패 시 직접 다운로드
            DetailPrint "Downloading Chrome installer..."
            NSISdl::download "https://dl.google.com/chrome/install/375.126/chrome_installer.exe" "$TEMP\chrome_installer.exe"
            Pop $0
            StrCmp $0 "success" 0 ChromeSkip
              ExecWait '"$TEMP\chrome_installer.exe" /silent /install'
              Delete "$TEMP\chrome_installer.exe"
        ChromeSkip:
  ChromeOk:
  DetailPrint "Chrome: OK"

  ; ── 3. Microsoft Outlook 확인 및 설치 제안 ───────────────────────
  DetailPrint "Checking Microsoft Outlook..."
  ; 데스크톱 Outlook (Microsoft 365)
  ReadRegStr $2 HKLM "SOFTWARE\Microsoft\Office\ClickToRun\Configuration" "ProductReleaseIds"
  StrCpy $3 ""
  ${If} ${FileExists} "$PROGRAMFILES\Microsoft Office\root\Office16\OUTLOOK.EXE"
    StrCpy $3 "found"
  ${EndIf}
  ${If} ${FileExists} "$PROGRAMFILES64\Microsoft Office\root\Office16\OUTLOOK.EXE"
    StrCpy $3 "found"
  ${EndIf}
  ; 새 Outlook for Windows (UWP)
  nsExec::ExecToStack 'powershell -WindowStyle Hidden -Command "if (Get-AppxPackage -Name Microsoft.OutlookForWindows -ErrorAction SilentlyContinue) { exit 0 } else { exit 1 }"'
  Pop $0
  StrCmp $0 "0" OutlookOk 0
  StrCmp $3 "found" OutlookOk 0
    MessageBox MB_YESNO|MB_ICONQUESTION \
      "Nexus의 이메일·일정 자동화 기능을 사용하려면 Microsoft Outlook이 필요합니다.$\n$\n새 Outlook (무료)을 설치할까요?$\n$\n※ 건너뛰어도 Gmail IMAP 연동은 가능합니다." \
      IDYES InstallOutlook IDNO OutlookSkip
    InstallOutlook:
      DetailPrint "Installing Microsoft Outlook (New)..."
      nsExec::ExecToStack 'powershell -WindowStyle Hidden -Command "winget install --id Microsoft.OutlookForWindows -e --silent --accept-package-agreements --accept-source-agreements 2>$null; exit 0"'
      Pop $0
    OutlookSkip:
  OutlookOk:
  DetailPrint "Outlook: OK"

  ; ── 4. WebView2 Runtime 확인 ────────────────────────────────────
  ; Tauri가 downloadBootstrapper로 처리하지만 fallback으로 확인
  DetailPrint "Checking WebView2 Runtime..."
  ReadRegStr $4 HKLM "SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}" "pv"
  StrCmp $4 "" 0 WebView2Ok
    ReadRegStr $4 HKCU "SOFTWARE\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}" "pv"
    StrCmp $4 "" 0 WebView2Ok
      DetailPrint "WebView2 will be installed by Tauri bootstrapper."
  WebView2Ok:

  ; ── 5. yt-dlp 다운로드 (영상 검색·다운로드용) ───────────────────
  DetailPrint "Installing yt-dlp..."
  SetOutPath "$APPDATA\Nexus"
  ${IfNot} ${FileExists} "$APPDATA\Nexus\yt-dlp.exe"
    nsExec::ExecToStack 'powershell -WindowStyle Hidden -Command "try { Invoke-WebRequest -Uri ''https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp.exe'' -OutFile ''$env:APPDATA\Nexus\yt-dlp.exe'' -UseBasicParsing } catch { exit 1 }; exit 0"'
    Pop $0
  ${EndIf}
  DetailPrint "yt-dlp: OK"

  ; ── 6. Nexus 설정 폴더 초기화 ───────────────────────────────────
  DetailPrint "Initializing Nexus config..."
  CreateDirectory "$APPDATA\Nexus"
  CreateDirectory "$APPDATA\Nexus\logs"
  CreateDirectory "$APPDATA\Nexus\workflows"
  CreateDirectory "$APPDATA\Nexus\memory"

  ; ── 7. 시작 메뉴 + 바탕화면 단축키 ─────────────────────────────
  CreateDirectory "$SMPROGRAMS\Nexus AI"
  CreateShortcut "$SMPROGRAMS\Nexus AI\Nexus AI.lnk" "$INSTDIR\Nexus.exe"
  CreateShortcut "$SMPROGRAMS\Nexus AI\제거.lnk" "$INSTDIR\uninstall.exe"
  CreateShortcut "$DESKTOP\Nexus AI.lnk" "$INSTDIR\Nexus.exe"

  DetailPrint "Nexus AI installation complete!"

!macroend

; ── 제거 시 정리 ──────────────────────────────────────────────────
!macro customUninstall
  ; 바탕화면 + 시작 메뉴 단축키 제거
  Delete "$DESKTOP\Nexus AI.lnk"
  RMDir /r "$SMPROGRAMS\Nexus AI"

  ; 백엔드 프로세스 종료
  nsExec::Exec 'taskkill /F /IM nexus-backend.exe /T'

  ; 사용자 데이터는 보존 (llm_config.json, memory 등)
  ; 완전 삭제 원하면 아래 주석 해제:
  ; RMDir /r "$APPDATA\Nexus"
!macroend
