//go:build windows

package security

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var telemetryHosts = []string{
	"telemetry.microsoft.com",
	"vortex.data.microsoft.com",
	"settings-win.data.microsoft.com",
	"watson.telemetry.microsoft.com",
	"oca.telemetry.microsoft.com",
	"feedback.windows.com",
	"sqm.telemetry.microsoft.com",
	"watson.microsoft.com",
}

func BlockTelemetry() error {
	hostsPath := filepath.Join(os.Getenv("SystemRoot"), "System32", "drivers", "etc", "hosts")
	content, _ := os.ReadFile(hostsPath)
	newLines := "\n# Nexus - Telemetry 차단\n"
	for _, host := range telemetryHosts {
		if !strings.Contains(string(content), host) {
			newLines += fmt.Sprintf("0.0.0.0 %s\n", host)
		}
	}
	return os.WriteFile(hostsPath, append(content, []byte(newLines)...), 0644)
}

func UnblockTelemetry() error {
	hostsPath := filepath.Join(os.Getenv("SystemRoot"), "System32", "drivers", "etc", "hosts")
	content, _ := os.ReadFile(hostsPath)
	lines := strings.Split(string(content), "\n")
	var newLines []string
	skip := false
	for _, line := range lines {
		if strings.Contains(line, "Nexus - Telemetry 차단") {
			skip = true
		}
		if !skip {
			newLines = append(newLines, line)
		}
		if skip && strings.TrimSpace(line) == "" {
			skip = false
		}
	}
	return os.WriteFile(hostsPath, []byte(strings.Join(newLines, "\n")), 0644)
}
