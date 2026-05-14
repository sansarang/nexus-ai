//go:build windows

package main

import (
	"context"
	"os/exec"
	"time"
)

const defaultPSTimeout = 30 * time.Second

// execPS runs a PowerShell command with a 30-second timeout.
func execPS(script string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultPSTimeout)
	defer cancel()
	return exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", script).Output()
}

// execPSRun runs a PowerShell command (no output) with a 30-second timeout.
func execPSRun(script string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultPSTimeout)
	defer cancel()
	return exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", script).Run()
}
