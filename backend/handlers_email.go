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
//  Email — Outlook MAPI 연동
//  PowerShell COM으로 받은 메일 읽기 + 전송
// ══════════════════════════════════════════════════════════════════

// GET /api/email/inbox — 받은 편지함 최근 N개
func handleEmailInbox(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}
	if limit > 50 {
		limit = 50
	}

	emails, err := getOutlookInbox(limit)
	if err != nil {
		json200(w, map[string]interface{}{
			"success": false,
			"emails":  []EmailItem{},
			"total":   0,
			"message": "Outlook이 설치되지 않았거나 접근 권한이 없어요.",
		})
		return
	}

	unread := 0
	for _, e := range emails {
		if !e.IsRead {
			unread++
		}
	}

	msg := fmt.Sprintf("최근 이메일 %d개를 가져왔어요. 읽지 않은 메일 %d개 있어요 📧", len(emails), unread)
	if len(emails) == 0 {
		msg = "받은 편지함이 비어있어요 📭"
	}

	json200(w, map[string]interface{}{
		"success": true,
		"emails":  emails,
		"total":   len(emails),
		"unread":  unread,
		"message": msg,
	})
}

// POST /api/email/send — 이메일 전송
func handleEmailSend(w http.ResponseWriter, r *http.Request) {
	var req struct {
		To      string `json:"to"`
		Subject string `json:"subject"`
		Body    string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.To == "" {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	script := fmt.Sprintf(`
try {
  $ol = New-Object -ComObject Outlook.Application -ErrorAction Stop
  $mail = $ol.CreateItem(0) # MailItem
  $mail.To = "%s"
  $mail.Subject = "%s"
  $mail.Body = "%s"
  $mail.Send()
  Write-Output "OK"
} catch {
  Write-Output "ERROR: $_"
}
`, escapePSString(req.To), escapePSString(req.Subject), escapePSString(req.Body))

	out, err := exec.Command("powershell", "-NoProfile", "-Command", script).Output()
	success := err == nil && strings.Contains(string(out), "OK")

	msg := fmt.Sprintf("'%s'에게 메일을 보냈어요 📤", req.To)
	if !success {
		msg = "메일 전송에 실패했어요. Outlook 설정을 확인해주세요."
	}

	json200(w, map[string]interface{}{
		"success": success,
		"message": msg,
	})
}

// POST /api/email/summarize — 받은 메일 AI 요약
func handleEmailSummarize(w http.ResponseWriter, r *http.Request) {
	emails, err := getOutlookInbox(5)
	if err != nil || len(emails) == 0 {
		json200(w, map[string]interface{}{
			"success": false,
			"summary": "",
			"message": "요약할 이메일이 없어요.",
		})
		return
	}

	// 제목 + 발신자 목록으로 요약 생성
	var parts []string
	for _, e := range emails {
		readMark := "📩"
		if e.IsRead {
			readMark = "📨"
		}
		parts = append(parts, fmt.Sprintf("%s %s (from: %s)", readMark, e.Subject, e.Sender))
	}

	summary := strings.Join(parts, "\n")
	json200(w, map[string]interface{}{
		"success": true,
		"emails":  emails,
		"summary": summary,
		"message": fmt.Sprintf("최근 이메일 %d개를 요약했어요 📧", len(emails)),
	})
}

// sendOutlookEmail — 워크플로우 등 내부에서 직접 호출용
func sendOutlookEmail(to, subject, body string) error {
	script := fmt.Sprintf(`
try {
  $ol = New-Object -ComObject Outlook.Application -ErrorAction Stop
  $mail = $ol.CreateItem(0)
  $mail.To = "%s"
  $mail.Subject = "%s"
  $mail.Body = "%s"
  $mail.Send()
  Write-Output "OK"
} catch {
  Write-Output "ERROR: $_"
}
`, escapePSString(to), escapePSString(subject), escapePSString(body))
	out, err := exec.Command("powershell", "-NoProfile", "-Command", script).Output()
	if err != nil {
		return err
	}
	if !strings.Contains(string(out), "OK") {
		return fmt.Errorf("Outlook 전송 실패")
	}
	return nil
}

// getOutlookInbox — PowerShell COM으로 받은 편지함 읽기
func getOutlookInbox(limit int) ([]EmailItem, error) {
	// 백틱 충돌 방지: 스크립트를 임시 파일에 저장 후 실행
	script := fmt.Sprintf(
		"try {\n"+
			"  $ol = New-Object -ComObject Outlook.Application -ErrorAction Stop\n"+
			"  $ns = $ol.GetNamespace(\"MAPI\")\n"+
			"  $inbox = $ns.GetDefaultFolder(6)\n"+
			"  $items = $inbox.Items\n"+
			"  $items.Sort(\"[ReceivedTime]\", $true)\n"+
			"  $result = @()\n"+
			"  $count = 0\n"+
			"  foreach ($mail in $items) {\n"+
			"    if ($count -ge %d) { break }\n"+
			"    $blen = $mail.Body.Length\n"+
			"    if ($blen -gt 200) { $b = $mail.Body.Substring(0,200) + '...' } else { $b = $mail.Body }\n"+
			"    $b2 = $b -replace \"`r`n\", ' ' -replace \"`n\", ' '\n"+
			"    $result += [PSCustomObject]@{ subject=$mail.Subject; sender=$mail.SenderName; received_at=$mail.ReceivedTime.ToString('yyyy-MM-dd HH:mm'); body=$b2; is_read=($mail.UnRead -eq $false); has_attachments=($mail.Attachments.Count -gt 0) }\n"+
			"    $count++\n"+
			"  }\n"+
			"  $result | ConvertTo-Json -Depth 2\n"+
			"} catch { Write-Output \"ERROR: $_\" }\n",
		limit,
	)

	out, err := exec.Command("powershell", "-NoProfile", "-Command", script).Output()
	if err != nil {
		return nil, err
	}

	outStr := strings.TrimSpace(string(out))
	if strings.HasPrefix(outStr, "ERROR") {
		return nil, fmt.Errorf(outStr)
	}
	if outStr == "" || outStr == "null" {
		return []EmailItem{}, nil
	}
	if strings.HasPrefix(outStr, "{") {
		outStr = "[" + outStr + "]"
	}

	var raw []struct {
		Subject    string `json:"subject"`
		Sender     string `json:"sender"`
		ReceivedAt string `json:"received_at"`
		Body       string `json:"body"`
		IsRead     bool   `json:"is_read"`
		HasAttach  bool   `json:"has_attachments"`
	}
	if err := json.Unmarshal([]byte(outStr), &raw); err != nil {
		return nil, err
	}

	items := make([]EmailItem, 0, len(raw))
	for _, v := range raw {
		items = append(items, EmailItem{
			Subject:    v.Subject,
			Sender:     v.Sender,
			ReceivedAt: v.ReceivedAt,
			Body:       v.Body,
			IsRead:     v.IsRead,
			HasAttach:  v.HasAttach,
		})
	}
	return items, nil
}

func escapePSString(s string) string {
	s = strings.ReplaceAll(s, `"`, `'`)
	s = strings.ReplaceAll(s, "`", "``")
	return s
}
