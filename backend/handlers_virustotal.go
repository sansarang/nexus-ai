//go:build windows

package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// ══════════════════════════════════════════════════════════════════
//  VirusTotal — 파일 해시 조회로 악성 여부 판단
//  API key: localStorage nexus-virustotal-key (프론트에서 헤더로 전달)
// ══════════════════════════════════════════════════════════════════

type VTResult struct {
	Success     bool   `json:"success"`
	FilePath    string `json:"file_path"`
	FileHash    string `json:"file_hash"`
	Malicious   int    `json:"malicious"`
	Suspicious  int    `json:"suspicious"`
	Clean       int    `json:"clean"`
	TotalScans  int    `json:"total_scans"`
	Permalink   string `json:"permalink"`
	SafeScore   int    `json:"safe_score"`
	Verdict     string `json:"verdict"`
	Message     string `json:"message"`
}

// POST /api/security/virustotal — 파일 해시 VirusTotal 조회
func handleVirusTotal(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FilePath string `json:"file_path"`
		APIKey   string `json:"api_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	if req.FilePath == "" {
		json200(w, VTResult{
			Success: false,
			Message: "파일 경로를 입력해주세요.",
		})
		return
	}

	// 파일 해시 계산
	hash, err := md5File(req.FilePath)
	if err != nil {
		json200(w, VTResult{
			Success:  false,
			FilePath: req.FilePath,
			Message:  "파일을 찾을 수 없어요. 경로를 다시 확인해주세요.",
		})
		return
	}

	if req.APIKey == "" {
		// API 키 없으면 해시만 반환
		json200(w, VTResult{
			Success:  true,
			FilePath: req.FilePath,
			FileHash: hash,
			Message:  fmt.Sprintf("파일 MD5: %s\nVirusTotal API 키가 없어서 온라인 조회는 불가능해요. 설정에서 API 키를 입력해주세요.", hash),
		})
		return
	}

	// VirusTotal API 조회
	result, err := queryVirusTotal(hash, req.APIKey)
	if err != nil {
		json200(w, VTResult{
			Success:  false,
			FilePath: req.FilePath,
			FileHash: hash,
			Message:  "VirusTotal 조회 실패: " + err.Error(),
		})
		return
	}

	result.FilePath = req.FilePath
	result.FileHash = hash
	json200(w, result)
}

func md5File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func queryVirusTotal(hash, apiKey string) (*VTResult, error) {
	url := "https://www.virustotal.com/api/v3/files/" + hash

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-apikey", apiKey)

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return &VTResult{
			Success:    true,
			Malicious:  0,
			Suspicious: 0,
			Clean:      0,
			TotalScans: 0,
			SafeScore:  100,
			Verdict:    "unknown",
			Message:    "VirusTotal 데이터베이스에 없는 파일이에요. 매우 새 파일이거나 분석된 적 없는 파일입니다.",
		}, nil
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API 오류: HTTP %d", resp.StatusCode)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}

	data, ok := body["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("응답 파싱 실패")
	}

	attrs, ok := data["attributes"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("속성 파싱 실패")
	}

	stats, _ := attrs["last_analysis_stats"].(map[string]interface{})
	malicious := int(getFloat(stats, "malicious"))
	suspicious := int(getFloat(stats, "suspicious"))
	harmless := int(getFloat(stats, "harmless"))
	undetected := int(getFloat(stats, "undetected"))
	total := malicious + suspicious + harmless + undetected

	permalink := fmt.Sprintf("https://www.virustotal.com/gui/file/%s", hash)

	safeScore := 100
	if total > 0 {
		safeScore = 100 - (malicious*100/total) - (suspicious*30/total)
		if safeScore < 0 {
			safeScore = 0
		}
	}

	verdict := "safe"
	msg := "✅ 안전한 파일입니다."
	if malicious > 3 {
		verdict = "malicious"
		msg = fmt.Sprintf("🚨 위험! %d개 백신이 악성 파일로 탐지했어요. 즉시 삭제를 권장합니다.", malicious)
	} else if malicious > 0 || suspicious > 2 {
		verdict = "suspicious"
		msg = fmt.Sprintf("⚠️ 의심스러운 파일 — %d개 탐지, %d개 수상. 주의가 필요해요.", malicious, suspicious)
	} else if total == 0 {
		verdict = "unknown"
		msg = "분석 데이터가 없어요."
	}

	name, _ := attrs["meaningful_name"].(string)
	if name == "" {
		name = strings.TrimPrefix(hash, "")
	}

	return &VTResult{
		Success:    true,
		Malicious:  malicious,
		Suspicious: suspicious,
		Clean:      harmless + undetected,
		TotalScans: total,
		Permalink:  permalink,
		SafeScore:  safeScore,
		Verdict:    verdict,
		Message:    msg,
	}, nil
}

func getFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key].(float64); ok {
		return v
	}
	return 0
}
