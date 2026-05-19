package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// ══════════════════════════════════════════════════════════════
//  Shodan / Censys — IP 노출 감사
//  POST /api/security/shodan   body: { "target": "IP or domain" }
//  GET  /api/security/myip     — 현재 공인 IP 자동 조회
// ══════════════════════════════════════════════════════════════

type ShodanPort struct {
	Port      int    `json:"port"`
	Proto     string `json:"proto"`
	Service   string `json:"service"`
	Product   string `json:"product,omitempty"`
	Version   string `json:"version,omitempty"`
	CVECount  int    `json:"cve_count,omitempty"`
	Risk      string `json:"risk"` // low / medium / high / critical
}

type ShodanResult struct {
	IP          string       `json:"ip"`
	Hostnames   []string     `json:"hostnames"`
	Country     string       `json:"country"`
	City        string       `json:"city"`
	ISP         string       `json:"isp"`
	Org         string       `json:"org"`
	OS          string       `json:"os,omitempty"`
	OpenPorts   []ShodanPort `json:"open_ports"`
	Vulns       []string     `json:"vulns"`
	RiskScore   int          `json:"risk_score"` // 0~100
	RiskLevel   string       `json:"risk_level"` // safe / low / medium / high / critical
	LastUpdated string       `json:"last_updated"`
}

// POST /api/security/shodan
func handleShodanAudit(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Target   string `json:"target"`    // IP or domain
		ShodanKey string `json:"shodan_key"` // 옵셔널 — 없으면 무료 API 사용
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.Target == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "target(IP 또는 도메인) 필요"})
		return
	}

	// 도메인이면 IP로 변환
	ip := resolveTarget(req.Target)
	if ip == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "IP 조회 실패: " + req.Target})
		return
	}

	// llmConfig에서 Shodan 키 읽기
	if req.ShodanKey == "" {
		llmMu.RLock()
		req.ShodanKey = llmShodanKey
		llmMu.RUnlock()
	}

	var result ShodanResult
	var err error

	if req.ShodanKey != "" {
		result, err = queryShodan(ip, req.ShodanKey)
	} else {
		// 키 없으면 무료 ipinfo + 포트스캔 조합
		result, err = queryFreeOSINT(ip)
	}

	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": "조회 실패: " + err.Error()})
		return
	}

	result.RiskScore = calculateRiskScore(result.OpenPorts, result.Vulns)
	result.RiskLevel = riskLevel(result.RiskScore)

	msg := buildShodanReport(req.Target, result)

	writeJSON(w, 200, map[string]any{
		"success": true,
		"target":  req.Target,
		"ip":      ip,
		"result":  result,
		"message": msg,
	})
}

// GET /api/security/myip — 내 공인 IP 조회 후 Shodan 감사
func handleMyIPAudit(w http.ResponseWriter, r *http.Request) {
	myIP := getPublicIP()
	if myIP == "" {
		writeJSON(w, 500, map[string]any{"success": false, "message": "공인 IP 조회 실패"})
		return
	}

	llmMu.RLock()
	shodanKey := llmShodanKey
	llmMu.RUnlock()

	var result ShodanResult
	var err error
	if shodanKey != "" {
		result, err = queryShodan(myIP, shodanKey)
	} else {
		result, err = queryFreeOSINT(myIP)
	}
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}

	result.RiskScore = calculateRiskScore(result.OpenPorts, result.Vulns)
	result.RiskLevel = riskLevel(result.RiskScore)
	msg := buildShodanReport(myIP+" (내 IP)", result)

	writeJSON(w, 200, map[string]any{
		"success": true,
		"my_ip":   myIP,
		"result":  result,
		"message": msg,
	})
}

// ── Shodan API v1 ─────────────────────────────────────────────
func queryShodan(ip, key string) (ShodanResult, error) {
	apiURL := fmt.Sprintf("https://api.shodan.io/shodan/host/%s?key=%s", ip, key)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return ShodanResult{}, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var raw struct {
		IP        string   `json:"ip_str"`
		Hostnames []string `json:"hostnames"`
		Country   string   `json:"country_name"`
		City      string   `json:"city"`
		ISP       string   `json:"isp"`
		Org       string   `json:"org"`
		OS        string   `json:"os"`
		Vulns     []string `json:"vulns"`
		LastUpdate string  `json:"last_update"`
		Data      []struct {
			Port    int    `json:"port"`
			Proto   string `json:"transport"`
			Product string `json:"product"`
			Version string `json:"version"`
			CPEs    []string `json:"cpe"`
		} `json:"data"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return ShodanResult{}, err
	}
	if raw.Error != "" {
		return ShodanResult{}, fmt.Errorf("Shodan: %s", raw.Error)
	}

	var ports []ShodanPort
	for _, d := range raw.Data {
		p := ShodanPort{
			Port:    d.Port,
			Proto:   d.Proto,
			Service: guessService(d.Port),
			Product: d.Product,
			Version: d.Version,
			Risk:    portRisk(d.Port),
		}
		ports = append(ports, p)
	}

	return ShodanResult{
		IP:          raw.IP,
		Hostnames:   raw.Hostnames,
		Country:     raw.Country,
		City:        raw.City,
		ISP:         raw.ISP,
		Org:         raw.Org,
		OS:          raw.OS,
		OpenPorts:   ports,
		Vulns:       raw.Vulns,
		LastUpdated: raw.LastUpdate,
	}, nil
}

// 무료 OSINT: ipinfo.io + 기본 포트 스캔
func queryFreeOSINT(ip string) (ShodanResult, error) {
	res := ShodanResult{IP: ip}

	// ipinfo.io 무료 API
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Get("https://ipinfo.io/" + ip + "/json")
	if err == nil {
		defer resp.Body.Close()
		var info struct {
			Hostname string `json:"hostname"`
			City     string `json:"city"`
			Country  string `json:"country"`
			Org      string `json:"org"`
		}
		json.NewDecoder(resp.Body).Decode(&info)
		res.Country = info.Country
		res.City = info.City
		res.ISP = info.Org
		if info.Hostname != "" {
			res.Hostnames = []string{info.Hostname}
		}
	}

	// 일반 노출 포트 스캔 (비동기 제한적)
	commonPorts := []int{21, 22, 23, 25, 80, 443, 3306, 3389, 5900, 6379, 8080, 8443, 27017}
	for _, port := range commonPorts {
		addr := fmt.Sprintf("%s:%d", ip, port)
		conn, err := net.DialTimeout("tcp", addr, 800*time.Millisecond)
		if err == nil {
			conn.Close()
			p := ShodanPort{
				Port:    port,
				Proto:   "tcp",
				Service: guessService(port),
				Risk:    portRisk(port),
			}
			res.OpenPorts = append(res.OpenPorts, p)
		}
	}

	res.LastUpdated = time.Now().Format("2006-01-02")
	return res, nil
}

// ── 헬퍼 ──────────────────────────────────────────────────────
func resolveTarget(target string) string {
	if net.ParseIP(target) != nil {
		return target
	}
	addrs, err := net.LookupHost(target)
	if err != nil || len(addrs) == 0 {
		return ""
	}
	return addrs[0]
}

func getPublicIP() string {
	client := &http.Client{Timeout: 5 * time.Second}
	for _, api := range []string{"https://api.ipify.org", "https://icanhazip.com"} {
		resp, err := client.Get(api)
		if err == nil {
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			ip := strings.TrimSpace(string(body))
			if net.ParseIP(ip) != nil {
				return ip
			}
		}
	}
	return ""
}

func guessService(port int) string {
	services := map[int]string{
		21: "FTP", 22: "SSH", 23: "Telnet", 25: "SMTP",
		53: "DNS", 80: "HTTP", 110: "POP3", 143: "IMAP",
		443: "HTTPS", 3306: "MySQL", 3389: "RDP", 5432: "PostgreSQL",
		5900: "VNC", 6379: "Redis", 8080: "HTTP-Alt", 8443: "HTTPS-Alt",
		27017: "MongoDB", 9200: "Elasticsearch", 11211: "Memcached",
	}
	if s, ok := services[port]; ok {
		return s
	}
	return fmt.Sprintf("Port/%d", port)
}

func portRisk(port int) string {
	critical := map[int]bool{23: true, 3389: true, 5900: true}
	high     := map[int]bool{21: true, 3306: true, 27017: true, 6379: true, 9200: true, 11211: true, 5432: true}
	medium   := map[int]bool{22: true, 25: true, 8080: true}

	if critical[port] {
		return "critical"
	}
	if high[port] {
		return "high"
	}
	if medium[port] {
		return "medium"
	}
	return "low"
}

func calculateRiskScore(ports []ShodanPort, vulns []string) int {
	score := 0
	for _, p := range ports {
		switch p.Risk {
		case "critical":
			score += 30
		case "high":
			score += 15
		case "medium":
			score += 8
		default:
			score += 2
		}
	}
	score += len(vulns) * 10
	if score > 100 {
		score = 100
	}
	return score
}

func riskLevel(score int) string {
	switch {
	case score == 0:
		return "safe"
	case score < 20:
		return "low"
	case score < 50:
		return "medium"
	case score < 75:
		return "high"
	default:
		return "critical"
	}
}

func buildShodanReport(target string, r ShodanResult) string {
	riskEmoji := map[string]string{
		"safe": "✅", "low": "🟡", "medium": "🟠", "high": "🔴", "critical": "💀",
	}
	emoji := riskEmoji[r.RiskLevel]
	if emoji == "" {
		emoji = "⚠️"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s **%s** 보안 감사 결과 (위험도: %s, 점수: %d/100)\n\n",
		emoji, target, strings.ToUpper(r.RiskLevel), r.RiskScore))

	if r.Country != "" {
		sb.WriteString(fmt.Sprintf("🌍 위치: %s %s | ISP: %s\n", r.Country, r.City, r.ISP))
	}

	if len(r.OpenPorts) > 0 {
		sb.WriteString(fmt.Sprintf("\n🔓 **열린 포트 %d개:**\n", len(r.OpenPorts)))
		for _, p := range r.OpenPorts {
			riskTag := ""
			if p.Risk == "critical" || p.Risk == "high" {
				riskTag = " ⚠️ 위험"
			}
			sb.WriteString(fmt.Sprintf("- %d/%s (%s)%s\n", p.Port, p.Proto, p.Service, riskTag))
		}
	} else {
		sb.WriteString("\n✅ 감지된 열린 포트 없음\n")
	}

	if len(r.Vulns) > 0 {
		sb.WriteString(fmt.Sprintf("\n🚨 **알려진 취약점 %d개:**\n", len(r.Vulns)))
		for _, v := range r.Vulns {
			sb.WriteString(fmt.Sprintf("- %s\n", v))
		}
	}

	// 권고사항
	if r.RiskScore > 0 {
		sb.WriteString("\n💡 **권고사항:**\n")
		for _, p := range r.OpenPorts {
			if p.Risk == "critical" {
				sb.WriteString(fmt.Sprintf("- 🚨 %s(%d) 즉시 차단 권고\n", p.Service, p.Port))
			} else if p.Risk == "high" {
				sb.WriteString(fmt.Sprintf("- ⚠️ %s(%d) 방화벽 규칙 검토 필요\n", p.Service, p.Port))
			}
		}
	}

	return sb.String()
}
