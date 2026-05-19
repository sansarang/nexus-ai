//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
)

// ══════════════════════════════════════════════════════════════════
//  System Extensions — 프로세스 강제 종료, 앱 권한 감사,
//  Windows Update 확인, GPU 상세 모니터링
// ══════════════════════════════════════════════════════════════════

// POST /api/process/kill — 프로세스 강제 종료
func handleProcessKill(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PID  int    `json:"pid"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	var out []byte
	var err error

	if req.PID > 0 {
		out, err = exec.Command("powershell", "-NoProfile", "-Command",
			fmt.Sprintf(`Stop-Process -Id %d -Force -ErrorAction SilentlyContinue; Write-Output "OK"`, req.PID)).Output()
	} else if req.Name != "" {
		// 프로세스 이름 검증: 영숫자, 공백, 하이픈, 점만 허용
		safeName := strings.Map(func(r rune) rune {
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
				return r
			}
			return -1
		}, req.Name)
		if safeName == "" {
			json200(w, map[string]interface{}{"success": false, "message": "잘못된 프로세스 이름"})
			return
		}
		out, err = exec.Command("powershell", "-NoProfile", "-Command",
			fmt.Sprintf(`Stop-Process -Name "%s" -Force -ErrorAction SilentlyContinue; Write-Output "OK"`, safeName)).Output()
	} else {
		json200(w, map[string]interface{}{
			"success": false,
			"message": "PID 또는 프로세스 이름을 입력해주세요.",
		})
		return
	}

	success := err == nil && strings.Contains(string(out), "OK")
	name := req.Name
	if name == "" {
		name = fmt.Sprintf("PID %d", req.PID)
	}

	msg := fmt.Sprintf("✅ '%s' 프로세스를 종료했어요.", name)
	if !success {
		msg = fmt.Sprintf("'%s' 종료에 실패했어요. 관리자 권한이 필요할 수 있어요.", name)
	}

	json200(w, map[string]interface{}{
		"success": success,
		"name":    name,
		"message": msg,
	})
}

// GET /api/app/permissions — 앱별 권한 사용 현황
func handleAppPermissions(w http.ResponseWriter, r *http.Request) {
	appName := r.URL.Query().Get("app")

	script := `
$result = @()

# 카메라 접근 앱
$cameraApps = (Get-ItemProperty "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\CapabilityAccessManager\ConsentStore\webcam\NonPackaged" -ErrorAction SilentlyContinue) |
  Select-Object PSChildName
$cameraApps2 = Get-ChildItem "HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\CapabilityAccessManager\ConsentStore\webcam" -ErrorAction SilentlyContinue

# 마이크 접근 앱
$micApps = Get-ChildItem "HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\CapabilityAccessManager\ConsentStore\microphone" -ErrorAction SilentlyContinue

# 위치 접근 앱
$locApps = Get-ChildItem "HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\CapabilityAccessManager\ConsentStore\location" -ErrorAction SilentlyContinue

$camList = @()
if ($cameraApps2) {
  foreach ($app in $cameraApps2) {
    $val = (Get-ItemProperty -Path $app.PSPath -Name "Value" -ErrorAction SilentlyContinue).Value
    $camList += [PSCustomObject]@{ app = $app.PSChildName; permission = "camera"; status = $val }
  }
}

$micList = @()
if ($micApps) {
  foreach ($app in $micApps) {
    $val = (Get-ItemProperty -Path $app.PSPath -Name "Value" -ErrorAction SilentlyContinue).Value
    $micList += [PSCustomObject]@{ app = $app.PSChildName; permission = "microphone"; status = $val }
  }
}

[PSCustomObject]@{
  camera = $camList
  microphone = $micList
} | ConvertTo-Json -Depth 3
`

	out, err := execPS(script)
	if err != nil {
		json200(w, map[string]interface{}{
			"success": false,
			"message": "권한 정보를 가져오지 못했어요.",
		})
		return
	}

	var parsed map[string]interface{}
	outStr := strings.TrimSpace(string(out))
	if err := json.Unmarshal([]byte(outStr), &parsed); err != nil {
		json200(w, map[string]interface{}{
			"success": false,
			"message": "권한 정보 파싱 실패",
		})
		return
	}

	msg := "앱 권한 현황을 확인했어요 🔑"
	if appName != "" {
		msg = fmt.Sprintf("'%s' 앱의 권한 현황이에요", appName)
	}

	json200(w, map[string]interface{}{
		"success":     true,
		"permissions": parsed,
		"app_filter":  appName,
		"message":     msg,
	})
}

// GET /api/system/updates — Windows Update 대기 목록
func handleWindowsUpdates(w http.ResponseWriter, r *http.Request) {
	script := `
try {
  $UpdateSession = New-Object -ComObject Microsoft.Update.Session
  $UpdateSearcher = $UpdateSession.CreateUpdateSearcher()
  $SearchResult = $UpdateSearcher.Search("IsInstalled=0 and Type='Software'")
  $updates = @()
  foreach ($update in $SearchResult.Updates) {
    $updates += [PSCustomObject]@{
      title      = $update.Title
      kb         = ($update.KBArticleIDs | Select-Object -First 1)
      severity   = $update.MsrcSeverity
      size_mb    = [math]::Round($update.MaxDownloadSize / 1MB, 1)
      important  = $update.AutoSelectOnWebSites
    }
  }
  [PSCustomObject]@{
    count   = $SearchResult.Updates.Count
    updates = $updates
  } | ConvertTo-Json -Depth 3
} catch {
  Write-Output "ERROR: $_"
}
`
	out, err := execPS(script)
	if err != nil {
		json200(w, map[string]interface{}{
			"success": false,
			"count":   0,
			"updates": []interface{}{},
			"message": "Windows Update 확인 실패",
		})
		return
	}

	outStr := strings.TrimSpace(string(out))
	if strings.HasPrefix(outStr, "ERROR") {
		json200(w, map[string]interface{}{
			"success": false,
			"count":   0,
			"updates": []interface{}{},
			"message": "Windows Update 서비스에 접근할 수 없어요.",
		})
		return
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(outStr), &parsed); err != nil {
		json200(w, map[string]interface{}{
			"success": false,
			"message": "파싱 실패",
		})
		return
	}

	count := 0
	if c, ok := parsed["count"].(float64); ok {
		count = int(c)
	}

	msg := "✅ 설치 대기 중인 업데이트가 없어요!"
	if count > 0 {
		msg = fmt.Sprintf("⚠️ Windows 업데이트 %d개가 대기 중이에요. 설치를 권장해요.", count)
	}

	json200(w, map[string]interface{}{
		"success": true,
		"count":   count,
		"updates": parsed["updates"],
		"message": msg,
	})
}

// GET /api/gpu/stats — GPU 상세 모니터링
func handleGPUStats(w http.ResponseWriter, r *http.Request) {
	// nvidia-smi 직접 실행 (PowerShell 경유 시 2>$null 백틱 충돌 방지)
	nvidiaScript := "try {" +
		" $s = & nvidia-smi --query-gpu=utilization.gpu,temperature.gpu,memory.used,memory.total --format=csv,noheader,nounits 2>$null;" +
		" $s" +
		"} catch { '' }"

	script := `
try {
  $gpus = Get-CimInstance Win32_VideoController -ErrorAction Stop
  $result = @()
  foreach ($gpu in $gpus) {
    $nvidiaStat = ""
    try {
      $nvidiaStat = (& nvidia-smi --query-gpu=utilization.gpu,temperature.gpu,memory.used,memory.total --format=csv,noheader,nounits 2>$null)
    } catch {}
    $usage = 0; $tempVal = 0; $memUsed = 0; $memTotal = 0
    if ($nvidiaStat -and $nvidiaStat -notmatch "ERROR") {
      $parts = $nvidiaStat -split ","
      if ($parts.Count -ge 4) {
        $usage    = [int]$parts[0].Trim()
        $tempVal  = [int]$parts[1].Trim()
        $memUsed  = [int]$parts[2].Trim()
        $memTotal = [int]$parts[3].Trim()
      }
    }
    $result += [PSCustomObject]@{
      name         = $gpu.Name
      usage_pct    = $usage
      temp_c       = $tempVal
      mem_used_mb  = $memUsed
      mem_total_mb = $memTotal
      driver_ver   = $gpu.DriverVersion
      status       = $gpu.Status
    }
  }
  $result | ConvertTo-Json -Depth 2
} catch {
  Write-Output "ERROR: $_"
}
`
	_ = nvidiaScript
	out, err := execPS(script)
	if err != nil {
		json200(w, map[string]interface{}{
			"success": false,
			"gpus":    []interface{}{},
			"message": "GPU 정보를 가져오지 못했어요.",
		})
		return
	}

	outStr := strings.TrimSpace(string(out))
	if strings.HasPrefix(outStr, "ERROR") || outStr == "" || outStr == "null" {
		json200(w, map[string]interface{}{
			"success": false,
			"gpus":    []interface{}{},
			"message": "GPU 정보를 가져오지 못했어요.",
		})
		return
	}

	if strings.HasPrefix(outStr, "{") {
		outStr = "[" + outStr + "]"
	}

	var gpus []interface{}
	if err2 := json.Unmarshal([]byte(outStr), &gpus); err2 != nil {
		json200(w, map[string]interface{}{
			"success": false,
			"gpus":    []interface{}{},
			"message": "GPU 데이터 파싱 실패",
		})
		return
	}

	msg := fmt.Sprintf("GPU %d개 확인했어요 🎮", len(gpus))
	json200(w, map[string]interface{}{
		"success": true,
		"gpus":    gpus,
		"message": msg,
	})
}
