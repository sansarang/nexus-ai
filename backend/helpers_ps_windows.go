//go:build windows

package main

import (
	"context"
	"os/exec"
	"syscall"
	"time"
)

const defaultPSTimeout = 30 * time.Second

// newHiddenCmd creates an exec.Cmd that won't spawn a visible console window.
// Use this for ALL system commands (powershell, cmd, schtasks, taskkill, etc.)
func newHiddenCmd(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd
}

// newHiddenCmdCtx is the context-aware version of newHiddenCmd.
func newHiddenCmdCtx(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd
}

// execPS runs a PowerShell command with a 30-second timeout (no visible window).
func execPS(script string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultPSTimeout)
	defer cancel()
	return newHiddenCmdCtx(ctx, "powershell", "-NoProfile", "-Command", script).Output()
}

// execPSRun runs a PowerShell command (no output, no visible window).
func execPSRun(script string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultPSTimeout)
	defer cancel()
	return newHiddenCmdCtx(ctx, "powershell", "-NoProfile", "-Command", script).Run()
}
