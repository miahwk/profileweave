package infrastructure

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestValidateDiscoveredExecutableRequiresExecutePermissionOnUnix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows executable validation does not use Unix mode bits")
	}
	path := filepath.Join(t.TempDir(), "browser")
	if err := os.WriteFile(path, []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateDiscoveredExecutable(path); err == nil {
		t.Fatal("non-executable browser file was accepted")
	}
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := validateDiscoveredExecutable(path); err != nil {
		t.Fatalf("executable browser file was rejected: %v", err)
	}
}

func TestResolverRejectsLegacyCustomBrowserWithoutInspectingAPath(t *testing.T) {
	if _, err := (Resolver{}).Resolve(context.Background(), "custom"); err == nil {
		t.Fatal("custom browser selection was accepted")
	}
	if _, err := (Resolver{}).Resolve(context.Background(), "custom-disabled"); err == nil {
		t.Fatal("migrated custom browser selection was accepted")
	}
}
