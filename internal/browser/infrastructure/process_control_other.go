//go:build !windows && !unix

package infrastructure

import (
	"context"
	"os"
	"os/exec"
)

func configureProcess(_ *exec.Cmd) {}

func requestProcessTreeStop(_ context.Context, pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return process.Signal(os.Interrupt)
}

func forceProcessTreeStop(_ context.Context, pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return process.Kill()
}
