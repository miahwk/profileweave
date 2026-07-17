//go:build windows

package infrastructure

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
)

const createNewProcessGroup = 0x00000200

func configureProcess(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags |= createNewProcessGroup
}

func requestProcessTreeStop(ctx context.Context, pid int) error {
	return runTaskkill(ctx, pid, false)
}

func forceProcessTreeStop(ctx context.Context, pid int) error {
	return runTaskkill(ctx, pid, true)
}

func runTaskkill(ctx context.Context, pid int, force bool) error {
	path, err := systemTaskkillPath()
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, path, taskkillArguments(pid, force)...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = nil, nil, nil
	return cmd.Run()
}

func taskkillArguments(pid int, force bool) []string {
	args := []string{"/PID", strconv.Itoa(pid), "/T"}
	if force {
		args = append(args, "/F")
	}
	return args
}

func systemTaskkillPath() (string, error) {
	root := os.Getenv("SystemRoot")
	if root == "" {
		root = os.Getenv("WINDIR")
	}
	if root == "" || !filepath.IsAbs(root) {
		return "", errors.New("Windows system directory is unavailable")
	}
	return filepath.Join(root, "System32", "taskkill.exe"), nil
}
