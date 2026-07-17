package main

import (
	"path/filepath"
	"testing"
)

func TestConfigurationUsesNewNamesWithLegacyFallback(t *testing.T) {
	t.Setenv("PROFILEWEAVE_PORT", "4123")
	t.Setenv("FINGERPRINT_BROWSER_PORT", "5123")
	if got := port(); got != "4123" {
		t.Fatalf("port = %q", got)
	}

	t.Setenv("PROFILEWEAVE_DATA_DIR", filepath.Join(t.TempDir(), "current"))
	t.Setenv("FINGERPRINT_BROWSER_DATA_DIR", filepath.Join(t.TempDir(), "legacy"))
	dir, err := resolveDataDir()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(dir) != "current" {
		t.Fatalf("data dir = %q", dir)
	}
}

func TestInvalidPortFallsBackToDefault(t *testing.T) {
	t.Setenv("PROFILEWEAVE_PORT", "70000")
	t.Setenv("FINGERPRINT_BROWSER_PORT", "")
	if got := port(); got != "3210" {
		t.Fatalf("port = %q", got)
	}
}
