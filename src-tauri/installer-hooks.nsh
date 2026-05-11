; Nexus 설치 훅 — Microsoft Outlook 자동 설치 제안
!macro customInstall
  DetailPrint "Checking Microsoft Outlook..."
  nsExec::ExecToStack 'powershell -Command "Get-AppxPackage -Name Microsoft.OutlookForWindows | Select-Object -ExpandProperty Name"'
  Pop $0
  StrCmp $0 "" 0 OutlookInstalled
    MessageBox MB_YESNO "Nexus의 이메일 기능을 위해 Microsoft Outlook (무료)을 설치할까요?" IDYES InstallOutlook IDNO OutlookSkip
    InstallOutlook:
      DetailPrint "Installing Microsoft Outlook..."
      nsExec::Exec 'winget install --id Microsoft.OutlookForWindows -e --silent --accept-package-agreements --accept-source-agreements'
    OutlookSkip:
  OutlookInstalled:
!macroend
