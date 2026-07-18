package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/miahwk/profileweave/internal/platform/instancelock"
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

func TestParseOptionsRejectsConflictsAndArguments(t *testing.T) {
	if _, err := parseOptions([]string{"--open", "--shutdown"}, io.Discard); err == nil {
		t.Fatal("conflicting lifecycle options accepted")
	}
	if _, err := parseOptions([]string{"unexpected"}, io.Discard); err == nil {
		t.Fatal("positional argument accepted")
	}
}

func TestShutdownDoesNotCreateMissingDataDirectory(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "never-started", "ProfileWeave")
	t.Setenv("PROFILEWEAVE_DATA_DIR", dataDir)
	if err := runCommand([]string{"--shutdown"}, io.Discard); err != nil {
		t.Fatalf("idempotent shutdown: %v", err)
	}
	if _, err := os.Stat(dataDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing data directory was created or cannot be inspected: %v", err)
	}
}

func TestLogSetupDoesNotShadowLegacyDataDirectory(t *testing.T) {
	configRoot := t.TempDir()
	legacyDir := filepath.Join(configRoot, "FingerprintBrowser")
	currentDir := filepath.Join(configRoot, "ProfileWeave")
	if err := os.MkdirAll(legacyDir, 0o700); err != nil {
		t.Fatal(err)
	}
	previousConfigDirectory := userConfigDirectory
	userConfigDirectory = func() (string, error) { return configRoot, nil }
	defer func() { userConfigDirectory = previousConfigDirectory }()
	t.Setenv("PROFILEWEAVE_DATA_DIR", "")
	t.Setenv("FINGERPRINT_BROWSER_DATA_DIR", "")

	logPath := filepath.Join(currentDir, "logs", "profileweave.log")
	if err := runCommand([]string{"--shutdown", "--log-file", logPath}, io.Discard); err != nil {
		t.Fatalf("shutdown with legacy data: %v", err)
	}
	if _, err := os.Stat(filepath.Join(legacyDir, ".profileweave.lock")); err != nil {
		t.Fatalf("legacy data directory was not selected before logging: %v", err)
	}
	if _, err := os.Stat(filepath.Join(currentDir, ".profileweave.lock")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("logging shadowed the legacy data directory: %v", err)
	}
}

func TestRunCommandOpenReuseAndShutdown(t *testing.T) {
	portNumber := freePort(t)
	t.Setenv("PROFILEWEAVE_PORT", strconv.Itoa(portNumber))
	t.Setenv("PROFILEWEAVE_DATA_DIR", t.TempDir())
	t.Setenv("PROFILEWEAVE_WEB_DIR", t.TempDir())
	opened := make(chan string, 2)
	previousOpen := openManagementURL
	var openCalls atomic.Int32
	openManagementURL = func(value string) error {
		opened <- value
		if openCalls.Add(1) == 1 {
			return errors.New("test opener unavailable")
		}
		return nil
	}
	defer func() { openManagementURL = previousOpen }()

	serveDone := make(chan error, 1)
	go func() { serveDone <- runCommand([]string{"--open"}, io.Discard) }()
	wantURL := fmt.Sprintf("http://127.0.0.1:%d", portNumber)
	waitForOpen(t, opened, wantURL)
	if err := runCommand([]string{"--open"}, io.Discard); err != nil {
		t.Fatalf("reuse running application: %v", err)
	}
	waitForOpen(t, opened, wantURL)
	if err := runCommand([]string{"--shutdown"}, io.Discard); err != nil {
		t.Fatalf("shutdown running application: %v", err)
	}
	select {
	case err := <-serveDone:
		if err != nil {
			t.Fatalf("server command returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not finish unified shutdown")
	}
	if err := runCommand([]string{"--shutdown"}, io.Discard); err != nil {
		t.Fatalf("idempotent shutdown: %v", err)
	}
}

func freePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

func waitForOpen(t *testing.T, opened <-chan string, want string) {
	t.Helper()
	select {
	case got := <-opened:
		if got != want {
			t.Fatalf("opened URL = %q, want %q", got, want)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("management URL was not opened")
	}
}

func acquireTestLock(dir string) (func(), error) {
	dataLock, err := instancelock.Acquire(dir)
	if err != nil {
		return nil, err
	}
	return func() { _ = dataLock.Close() }, nil
}
