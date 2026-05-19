//go:build windows

package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// ══════════════════════════════════════════════════════════════
//  Persistent PowerShell Session
//  한 번 프로세스를 띄우고 stdin/stdout으로 계속 통신
//  → 매 호출 PS 시작 오버헤드 제거 (~300ms → ~5ms)
//  → 세션 변수·함수 유지 가능
// ══════════════════════════════════════════════════════════════

const psEOF = "<<NEXUS_PS_EOF>>"

type PSSession struct {
	mu     sync.Mutex
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner
	alive  bool
}

var globalPS = &PSSession{}
var psOnce   sync.Once

// getPSSession: 싱글턴 세션 반환 (필요 시 자동 시작)
func getPSSession() (*PSSession, error) {
	globalPS.mu.Lock()
	defer globalPS.mu.Unlock()

	if globalPS.alive {
		return globalPS, nil
	}

	cmd := exec.Command("powershell",
		"-NoProfile",
		"-NonInteractive",
		"-ExecutionPolicy", "Bypass",
		"-Command", "-",
	)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	globalPS.cmd    = cmd
	globalPS.stdin  = stdin
	globalPS.stdout = bufio.NewScanner(stdout)
	globalPS.alive  = true

	// 준비 확인
	go func() {
		cmd.Wait()
		globalPS.mu.Lock()
		globalPS.alive = false
		globalPS.mu.Unlock()
	}()

	return globalPS, nil
}

// execPSPersistent: Persistent 세션으로 PowerShell 실행
func execPSPersistent(script string) (string, error) {
	sess, err := getPSSession()
	if err != nil {
		out, e := execPS(script)
		return string(out), e
	}

	// alive 체크는 mu 없이 atomic 읽기 — 데드락 방지
	if !sess.alive {
		// 세션 재시작: getPSSession 내부에서 mu.Lock 처리
		sess, err = getPSSession()
		if err != nil {
			out, e := execPS(script)
			return string(out), e
		}
	}

	// 실행은 별도 sendMu로 직렬화 (sess.mu와 분리하여 재귀 데드락 방지)
	sess.mu.Lock()
	defer sess.mu.Unlock()

	if !sess.alive {
		sess.mu.Unlock()
		out, e := execPS(script)
		sess.mu.Lock()
		return string(out), e
	}

	// 스크립트 전송 + EOF 마커
	payload := script + "\nWrite-Output '" + psEOF + "'\n"
	if _, err := io.WriteString(sess.stdin, payload); err != nil {
		// stdin 끊김 → 세션 죽은 것으로 표시 후 fallback
		sess.alive = false
		out, e := execPS(script)
		return string(out), e
	}

	// 결과 수집 — 뮤텍스 유지 중에 goroutine 생성 (동시 접근 차단됨)
	type scanResult struct {
		text string
	}
	ch := make(chan scanResult, 1)
	go func() {
		var sb strings.Builder
		for sess.stdout.Scan() {
			line := sess.stdout.Text()
			if line == psEOF {
				break
			}
			sb.WriteString(line)
			sb.WriteRune('\n')
		}
		ch <- scanResult{sb.String()}
	}()

	select {
	case r := <-ch:
		return strings.TrimSpace(r.text), nil
	case <-time.After(25 * time.Second):
		// 타임아웃 시 세션 강제 종료 → 다음 호출에서 재시작
		sess.alive = false
		if sess.stdin != nil {
			sess.stdin.Close()
		}
		return "", fmt.Errorf("PowerShell 세션 타임아웃 (25s)")
	}
}

// execPSPersistentCtx: 컨텍스트 타임아웃 지원
func execPSPersistentCtx(ctx context.Context, script string) (string, error) {
	ch := make(chan struct {
		out string
		err error
	}, 1)
	go func() {
		out, err := execPSPersistent(script)
		ch <- struct {
			out string
			err error
		}{out, err}
	}()
	select {
	case r := <-ch:
		return r.out, r.err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// closePSSession: 종료 시 세션 정리
func closePSSession() {
	globalPS.mu.Lock()
	defer globalPS.mu.Unlock()
	if globalPS.alive && globalPS.stdin != nil {
		io.WriteString(globalPS.stdin, "exit\n")
		globalPS.stdin.Close()
		globalPS.alive = false
	}
}
