package infrastructure

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestSafeLocalProcessErrorRedactsPath(t *testing.T) {
	err := &os.PathError{Op: "fork/exec", Path: `C:\\Users\\alice\\private\\chrome.exe`, Err: os.ErrPermission}
	message := safeLocalProcessError(err)
	if strings.Contains(message, "alice") || strings.Contains(message, "chrome.exe") {
		t.Fatalf("local error leaked a path: %q", message)
	}
	if !strings.Contains(message, "permission denied") {
		t.Fatalf("local error lost its useful cause: %q", message)
	}
}

func TestSafeLocalProcessErrorDoesNotEchoUnknownError(t *testing.T) {
	message := safeLocalProcessError(errors.New("secret proxy and argv details"))
	if strings.Contains(message, "secret") || strings.Contains(message, "proxy") || strings.Contains(message, "argv") {
		t.Fatalf("local error echoed sensitive details: %q", message)
	}
}

func TestConfigureProcessAddsPlatformControls(t *testing.T) {
	cmd := exec.Command(os.Args[0])
	configureProcess(cmd)
	if cmd.SysProcAttr == nil {
		t.Fatal("process controls were not configured")
	}
}
