package infrastructure

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestValidateExecutableRequiresExecutePermissionOnUnix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows executable validation does not use Unix mode bits")
	}
	path := filepath.Join(t.TempDir(), "browser")
	if err := os.WriteFile(path, []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateExecutable(path); err == nil {
		t.Fatal("non-executable browser file was accepted")
	}
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := validateExecutable(path); err != nil {
		t.Fatalf("executable browser file was rejected: %v", err)
	}
}
