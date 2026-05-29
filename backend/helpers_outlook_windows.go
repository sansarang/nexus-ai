//go:build windows

package main

// outlookProfileCheckPS is a PowerShell snippet that must be prepended to every
// script that uses Outlook COM automation.
//
// It checks the Windows registry for a configured MAPI profile WITHOUT launching
// Outlook. If no profile is found it writes "ERROR: Outlook profile not configured"
// and exits — preventing the Outlook first-run setup wizard from appearing.
const outlookProfileCheckPS = `
$_hasProfile = $false
foreach ($_ver in @("16.0","15.0","14.0","12.0")) {
    $_key = "HKCU:\SOFTWARE\Microsoft\Office\$_ver\Outlook\Profiles"
    if (Test-Path $_key) {
        $_items = Get-ChildItem $_key -ErrorAction SilentlyContinue
        if ($_items -and $_items.Count -gt 0) { $_hasProfile = $true; break }
    }
}
if (-not $_hasProfile) { Write-Output "ERROR: Outlook profile not configured"; exit 0 }
`
