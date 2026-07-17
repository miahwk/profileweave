//go:build unix

package infrastructure

import (
	"errors"
	"os/exec"
	"syscall"
	"testing"
)

func TestConfigureProcessCreatesIndependentUnixGroup(t *testing.T) {
	cmd := exec.Command("unused")
	configureProcess(cmd)
	if !cmd.SysProcAttr.Setpgid {
		t.Fatal("Setpgid was not enabled")
	}
}

func testProcessAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}
