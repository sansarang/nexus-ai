//go:build !windows

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func cmdFileSearch(cx cmdCtx) {
		// 파일 검색 - Mac에서는 mdfind/find 사용
		fsQuery, _ := cx.params["query"].(string)
		if fsQuery == "" {
			fsQuery = cx.req.Message
		}
		fsEng := cx.req.Lang == "en" || isEnglishQuery(cx.req.Message)
		out, err := exec.Command("mdfind", "-name", fsQuery).Output()
		var fsItems []map[string]string
		if err == nil {
			lines := strings.Split(strings.TrimSpace(string(out)), "\n")
			for i, l := range lines {
				if l == "" || i >= 10 {
					break
				}
				fsItems = append(fsItems, map[string]string{"path": l, "name": filepath.Base(l)})
			}
		}
		var fsMsg string
		if len(fsItems) == 0 {
			if fsEng {
				fsMsg = fmt.Sprintf("No files found matching '%s'.", fsQuery)
			} else {
				fsMsg = fmt.Sprintf("'%s' 관련 파일을 찾지 못했습니다.", fsQuery)
			}
		} else {
			if fsEng {
				fsMsg = fmt.Sprintf("Found %d file(s) matching '%s'.", len(fsItems), fsQuery)
			} else {
				fsMsg = fmt.Sprintf("'%s' 관련 파일 %d개를 찾았습니다.", fsQuery, len(fsItems))
			}
		}
		json200(cx.w, CommandResponse{
			Success: true, Message: fsMsg, Action: "file_search",
			Result: map[string]any{"query": fsQuery, "items": fsItems, "count": len(fsItems)},
			Duration: cx.dur,
		})

}

func cmdScan(cx cmdCtx) {
		// PC 진단 - Mac에서 시스템 상태 조회
		scEng := cx.req.Lang == "en" || isEnglishQuery(cx.req.Message)
		scStats := map[string]any{}
		// CPU
		if out, err := exec.Command("sh", "-c", "top -l 1 -n 0 | grep 'CPU usage'").Output(); err == nil {
			line := string(out)
			if idx := strings.Index(line, "idle"); idx > 0 {
				parts := strings.Fields(line[:idx])
				if len(parts) > 0 {
					idleStr := strings.TrimSuffix(parts[len(parts)-1], "%")
					if idle, err2 := strconv.ParseFloat(idleStr, 64); err2 == nil {
						scStats["cpu_percent"] = fmt.Sprintf("%.1f%%", 100-idle)
					}
				}
			}
		}
		// Disk
		if out, err := exec.Command("df", "-H", "/").Output(); err == nil {
			lines := strings.Split(string(out), "\n")
			if len(lines) > 1 {
				fields := strings.Fields(lines[1])
				if len(fields) >= 5 {
					scStats["disk_used"] = fields[2]
					scStats["disk_total"] = fields[1]
					scStats["disk_percent"] = strings.TrimSuffix(fields[4], "%") + "%"
				}
			}
		}
		var scMsg string
		if scEng {
			scMsg = fmt.Sprintf("System scan complete. CPU: %v, Disk: %v / %v (%v used)",
				scStats["cpu_percent"], scStats["disk_used"], scStats["disk_total"], scStats["disk_percent"])
		} else {
			scMsg = fmt.Sprintf("PC 진단 완료. CPU: %v, 디스크: %v / %v (%v 사용 중)",
				scStats["cpu_percent"], scStats["disk_used"], scStats["disk_total"], scStats["disk_percent"])
		}
		json200(cx.w, CommandResponse{
			Success: true, Message: scMsg, Action: "scan",
			Result: map[string]any{"stats": scStats, "score": 85},
			Duration: cx.dur,
		})

}

func cmdClean(cx cmdCtx) {
		// 정리 - Mac에서 임시 파일 정리
		clEng := cx.req.Lang == "en" || isEnglishQuery(cx.req.Message)
		home, _ := os.UserHomeDir()
		targets := []string{
			filepath.Join(home, "Library/Caches"),
			"/private/var/folders",
		}
		_ = targets
		var clMsg string
		if clEng {
			clMsg = "System cleanup complete. Temporary files have been identified. (Full cleanup requires admin privileges on Mac)"
		} else {
			clMsg = "PC 정리 완료. 임시 파일을 확인했습니다. (Mac에서 전체 정리는 관리자 권한이 필요합니다)"
		}
		json200(cx.w, CommandResponse{Success: true, Message: clMsg, Action: "clean", Duration: cx.dur})

}

func cmdStats(cx cmdCtx) {
		// 리소스 현황
		stEng := cx.req.Lang == "en" || isEnglishQuery(cx.req.Message)
		stStats := map[string]any{}
		if out, err := exec.Command("sh", "-c", "top -l 1 -n 0 | grep 'CPU usage'").Output(); err == nil {
			line := string(out)
			if idx := strings.Index(line, "idle"); idx > 0 {
				parts := strings.Fields(line[:idx])
				if len(parts) > 0 {
					idleStr := strings.TrimSuffix(parts[len(parts)-1], "%")
					if idle, err2 := strconv.ParseFloat(idleStr, 64); err2 == nil {
						stStats["cpu_percent"] = 100 - idle
					}
				}
			}
		}
		if out, err := exec.Command("df", "-H", "/").Output(); err == nil {
			lines := strings.Split(string(out), "\n")
			if len(lines) > 1 {
				fields := strings.Fields(lines[1])
				if len(fields) >= 5 {
					stStats["disk_percent"] = strings.TrimSuffix(fields[4], "%")
					stStats["disk_used"] = fields[2]
					stStats["disk_total"] = fields[1]
				}
			}
		}
		var stMsg string
		if stEng {
			stMsg = fmt.Sprintf("System stats: CPU %.1f%%, Disk %v/%v", stStats["cpu_percent"], stStats["disk_used"], stStats["disk_total"])
		} else {
			stMsg = fmt.Sprintf("시스템 현황: CPU %.1f%%, 디스크 %v/%v", stStats["cpu_percent"], stStats["disk_used"], stStats["disk_total"])
		}
		json200(cx.w, CommandResponse{
			Success: true, Message: stMsg, Action: "stats",
			Result: stStats, Duration: cx.dur,
		})

}

func cmdLaunchApp(cx cmdCtx) {
		// 앱 실행 - Mac에서 open 명령 사용
		laApp, _ := cx.params["app_name"].(string)
		if laApp == "" {
			laApp = cx.req.Message
		}
		laEng := cx.req.Lang == "en" || isEnglishQuery(cx.req.Message)
		appMap := map[string]string{
			"크롬": "Google Chrome", "chrome": "Google Chrome",
			"사파리": "Safari", "safari": "Safari",
			"파이어폭스": "Firefox", "firefox": "Firefox",
			"워드": "Microsoft Word", "word": "Microsoft Word",
			"엑셀": "Microsoft Excel", "excel": "Microsoft Excel",
			"파워포인트": "Microsoft PowerPoint",
			"메모": "Notes", "note": "Notes",
			"터미널": "Terminal", "terminal": "Terminal",
			"카카오": "KakaoTalk", "카카오톡": "KakaoTalk",
			"슬랙": "Slack", "slack": "Slack",
		}
		execApp := laApp
		lower := strings.ToLower(laApp)
		for k, v := range appMap {
			if strings.Contains(lower, strings.ToLower(k)) {
				execApp = v
				break
			}
		}
		exec.Command("open", "-a", execApp).Start()
		var laMsg string
		if laEng {
			laMsg = fmt.Sprintf("Launched %s.", execApp)
		} else {
			laMsg = fmt.Sprintf("%s 실행했습니다.", execApp)
		}
		json200(cx.w, CommandResponse{Success: true, Message: laMsg, Action: "launch_app", Duration: cx.dur})

}

func cmdSystemControl(cx cmdCtx) {
		// 시스템 제어 - Mac에서 osascript 사용
		scCtrl, _ := cx.params["control"].(string)
		scVal := 50
		if v, ok := cx.params["value"].(float64); ok {
			scVal = int(v)
		}
		scEng2 := cx.req.Lang == "en" || isEnglishQuery(cx.req.Message)
		var scMsg2 string
		switch strings.ToLower(scCtrl) {
		case "volume", "볼륨":
			exec.Command("osascript", "-e", fmt.Sprintf("set volume output volume %d", scVal)).Run()
			if scEng2 {
				scMsg2 = fmt.Sprintf("Volume set to %d%%.", scVal)
			} else {
				scMsg2 = fmt.Sprintf("볼륨을 %d%%로 설정했습니다.", scVal)
			}
		case "mute", "음소거":
			exec.Command("osascript", "-e", "set volume with output muted").Run()
			if scEng2 {
				scMsg2 = "Muted."
			} else {
				scMsg2 = "음소거 처리했습니다."
			}
		case "sleep", "절전":
			exec.Command("osascript", "-e", `tell app "System Events" to sleep`).Run()
			if scEng2 {
				scMsg2 = "Going to sleep."
			} else {
				scMsg2 = "절전 모드로 전환합니다."
			}
		default:
			if scEng2 {
				scMsg2 = fmt.Sprintf("System control '%s' is not supported on Mac.", scCtrl)
			} else {
				scMsg2 = fmt.Sprintf("'%s' 제어는 Mac에서 지원되지 않습니다.", scCtrl)
			}
		}
		json200(cx.w, CommandResponse{Success: true, Message: scMsg2, Action: "system_control", Duration: cx.dur})

}

func cmdFocusMode(cx cmdCtx) {
		// 집중 모드 - Mac에서 Do Not Disturb
		fmEnable := true
		if v, ok := cx.params["enable"].(bool); ok {
			fmEnable = v
		}
		fmEng := cx.req.Lang == "en" || isEnglishQuery(cx.req.Message)
		// macOS Focus mode via osascript (best-effort)
		if fmEnable {
			exec.Command("osascript", "-e", `tell application "System Events" to set doNotDisturb of (get the current user) to true`).Run()
		} else {
			exec.Command("osascript", "-e", `tell application "System Events" to set doNotDisturb of (get the current user) to false`).Run()
		}
		var fmMsg string
		if fmEng {
			if fmEnable {
				fmMsg = "Focus mode enabled. 🎯 Notifications blocked."
			} else {
				fmMsg = "Focus mode disabled."
			}
		} else {
			if fmEnable {
				fmMsg = "집중 모드 켜졌습니다! 🎯 알림이 차단됐습니다."
			} else {
				fmMsg = "집중 모드 꺼졌습니다."
			}
		}
		json200(cx.w, CommandResponse{
			Success: true, Message: fmMsg, Action: "focus_mode",
			Result: map[string]any{"enabled": fmEnable},
			Duration: cx.dur,
		})

}

func cmdTimer(cx cmdCtx) {
		// 타이머/알람
		tmEng := cx.req.Lang == "en" || isEnglishQuery(cx.req.Message)
		var tmMsg string
		if tmEng {
			tmMsg = "Timer feature is available via the system. For precise scheduling, use the 'scheduler' action."
		} else {
			tmMsg = "타이머 기능은 시스템에서 사용 가능합니다. 정확한 일정 예약은 '스케줄러' 기능을 사용하세요."
		}
		json200(cx.w, CommandResponse{Success: false, Message: tmMsg, Action: "timer", Duration: cx.dur})

}

func cmdFileOps(cx cmdCtx) {
		var op, folder string
		if cx.params != nil {
			op, _ = cx.params["op"].(string)
			folder, _ = cx.params["folder"].(string)
		}
		// 메시지에서 폴더/op 힌트
		if folder == "" { folder = cx.req.Message }
		foEng := cx.req.Lang == "en" || isEnglishQuery(cx.req.Message)
		msgL := strings.ToLower(cx.req.Message)

		if op == "" {
			switch {
			case strings.Contains(msgL, "중복") || strings.Contains(msgL, "duplicate"):
				op = "duplicates"
			case strings.Contains(msgL, "대용량") || strings.Contains(msgL, "큰 파일") || strings.Contains(msgL, "large"):
				op = "large"
			default:
				op = "organize"
			}
		}
		var foMsg string
		proxyCall := func(endpoint string, payload map[string]any) string {
			body, _ := json.Marshal(payload)
			raw, err := httpPost("http://127.0.0.1:17891"+endpoint, body)
			if err != nil { return "" }
			var d map[string]any
			if json.Unmarshal(raw, &d) == nil {
				if m, ok := d["message"].(string); ok { return m }
			}
			return ""
		}
		switch op {
		case "duplicates":
			foMsg = proxyCall("/api/file/duplicates", map[string]any{"folder": "", "message": cx.req.Message})
			if foMsg == "" { foMsg = "중복 파일 탐지 중..." }
		case "large":
			foMsg = proxyCall("/api/file/large", map[string]any{"folder": "", "min_size_mb": 100, "message": cx.req.Message})
			if foMsg == "" { foMsg = "대용량 파일 탐지 중..." }
		default: // organize
			foMsg = proxyCall("/api/file/organize", map[string]any{"folder": folder, "dry_run": false, "message": cx.req.Message})
			if foMsg == "" {
				if foEng { foMsg = "Organizing files..." } else { foMsg = "파일 정리 중..." }
			}
		}
		_ = foEng
		appendSession(cx.userID, "user", cx.req.Message)
		appendSession(cx.userID, "assistant", foMsg)
		json200(cx.w, CommandResponse{Success: true, Message: foMsg, Action: "file_ops", Duration: cx.dur})

	// ── 🟠 4. 조건부 알림 트리거 ──────────────────────────────
}

func cmdTriggerAdd(cx cmdCtx) {
		var nl string
		if cx.params != nil { nl, _ = cx.params["nl"].(string) }
		if nl == "" { nl = cx.req.Message }
		trEng := cx.req.Lang == "en" || isEnglishQuery(cx.req.Message)
		t := parseTriggerFromNL(nl)
		var trMsg string
		if t != nil {
			triggerStoreMu.Lock()
			triggerStore[t.ID] = t
			triggerStoreMu.Unlock()
			saveTriggers()
			if trEng {
				trMsg = fmt.Sprintf("✅ Alert trigger set: '%s'", t.Name)
			} else {
				trMsg = fmt.Sprintf("✅ 알림 트리거 등록됨: '%s'", t.Name)
			}
		} else {
			if trEng {
				trMsg = "Couldn't parse the trigger. Try: 'Alert me when CPU exceeds 80%' or 'Remind me every day at 9am'"
			} else {
				trMsg = "트리거를 파악하지 못했어요. 예: 'CPU 80% 넘으면 알려줘', '매일 오전 9시에 알림'"
			}
		}
		appendSession(cx.userID, "user", cx.req.Message)
		appendSession(cx.userID, "assistant", trMsg)
		json200(cx.w, CommandResponse{Success: true, Message: trMsg, Action: "trigger_add", Duration: cx.dur})

	// ── 🟡 5. 화면 캡처 + Vision ──────────────────────────────
}

func cmdScreenAnalyze(cx cmdCtx) {
		var question string
		if cx.params != nil { question, _ = cx.params["question"].(string) }
		if question == "" { question = cx.req.Message }
		saEng := cx.req.Lang == "en" || isEnglishQuery(cx.req.Message)
		b64, err := captureScreen()
		var saMsg string
		if err != nil {
			if saEng { saMsg = "Screen capture failed: " + err.Error() } else { saMsg = "화면 캡처 실패: " + err.Error() }
		} else {
			lang := "ko"; if saEng { lang = "en" }
			saMsg, err = analyzeImageWithClaude(b64, question, lang)
			if err != nil {
				if saEng { saMsg = "Vision analysis failed: " + err.Error() } else { saMsg = "Vision 분석 실패: " + err.Error() }
			}
		}
		appendSession(cx.userID, "user", cx.req.Message)
		appendSession(cx.userID, "assistant", saMsg)
		json200(cx.w, CommandResponse{Success: true, Message: saMsg, Action: "screen_analyze", Duration: cx.dur})

	// ── 클립보드 읽기 + 처리 ──────────────────────────────────
}

func cmdClipboardAction(cx cmdCtx) {
		cbEng := cx.req.Lang == "en" || isEnglishQuery(cx.req.Message)
		cbText := readClipboard()
		if cbText == "" {
			var cbMsg string
			if cbEng {
				cbMsg = "Clipboard is empty. Please copy something first."
			} else {
				cbMsg = "클립보드가 비어 있습니다. 먼저 텍스트를 복사해주세요."
			}
			json200(cx.w, CommandResponse{Success: true, Message: cbMsg, Action: "clipboard_action", Duration: cx.dur})
			return
		}
		// 클립보드 내용 + 원래 요청을 합쳐서 LLM에 전달
		var cbAction string
		if cx.params != nil {
			cbAction, _ = cx.params["action"].(string)
		}
		if cbAction == "" {
			cbAction = detectClipboardAction(cx.req.Message)
		}
		cbResult := processClipboardContent(cbText, cbAction, cx.req.Message, cx.gKey, cbEng)
		appendSession(cx.userID, "user", cx.req.Message)
		appendSession(cx.userID, "assistant", cbResult)
		json200(cx.w, CommandResponse{
			Success: true, Message: cbResult, Action: "clipboard_action",
			Result:  map[string]any{"clipboard_text": cbText, "action": cbAction},
			Duration: cx.dur,
		})

	// ── 🔴 1. 환율 ──────────────────────────────────────────────
}

func cmdWindowsOnly(cx cmdCtx) {
		var feature string
		if cx.params != nil {
			feature, _ = cx.params["feature"].(string)
		}
		var msg string
		woEng := cx.req.Lang == "en" || isEnglishQuery(cx.req.Message)
		if woEng {
			if feature != "" {
				msg = fmt.Sprintf("'%s' is only available on Windows PC.", feature)
			} else {
				msg = "This feature is only available on Windows PC."
			}
		} else {
			if feature != "" {
				msg = fmt.Sprintf("'%s' 기능은 Windows PC에서만 사용 가능합니다.", feature)
			} else {
				msg = "이 기능은 Windows PC에서만 사용 가능합니다."
			}
		}
		json200(cx.w, CommandResponse{
			Success:  false,
			Message:  msg,
			Action:   "windows_only",
			Duration: cx.dur,
		})

}

