//go:build unix

package infrastructure

import (
	"context"
	"errors"
	"os/exec"
	"syscall"
)

func configureProcess(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

func requestProcessTreeStop(_ context.Context, pid int) error {
	return signalProcessGroup(pid, syscall.SIGTERM)
}

func forceProcessTreeStop(_ context.Context, pid int) error {
	return signalProcessGroup(pid, syscall.SIGKILL)
}

func signalProcessGroup(pid int, signal syscall.Signal) error {
	err := syscall.Kill(-pid, signal)
	if errors.Is(err, syscall.ESRCH) {
		return nil
	}
	return err
}
