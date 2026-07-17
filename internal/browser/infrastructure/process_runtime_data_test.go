package infrastructure

import (
	"os"
	"path/filepath"
	"testing"
)

const runtimeTestProfileID = "p_abcdef0123456789abcdef0123456789"

func TestProfileDataTrashCanBeRestored(t *testing.T) {
	dataDir := t.TempDir()
	runtime, err := NewProcessRuntime(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	created, err := runtime.EnsureProfileData(runtimeTestProfileID)
	if err != nil || !created {
		t.Fatalf("first ensure: created=%v err=%v", created, err)
	}
	created, err = runtime.EnsureProfileData(runtimeTestProfileID)
	if err != nil || created {
		t.Fatalf("second ensure: created=%v err=%v", created, err)
	}
	marker := filepath.Join(dataDir, "browser-data", runtimeTestProfileID, "marker")
	if err := os.WriteFile(marker, []byte("preserved"), 0o600); err != nil {
		t.Fatal(err)
	}
	token, err := runtime.TrashProfileData(runtimeTestProfileID)
	if err != nil || token == "" {
		t.Fatalf("trash: token=%q err=%v", token, err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("marker still at active path: %v", err)
	}
	if err := runtime.RestoreProfileData(runtimeTestProfileID, token); err != nil {
		t.Fatal(err)
	}
	if raw, err := os.ReadFile(marker); err != nil || string(raw) != "preserved" {
		t.Fatalf("restored marker=%q err=%v", raw, err)
	}
	if err := runtime.RollbackRestoredProfileData(runtimeTestProfileID, token); err != nil {
		t.Fatal(err)
	}
	if err := runtime.PurgeProfileData(runtimeTestProfileID, token); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "trash", "browser-data", token)); !os.IsNotExist(err) {
		t.Fatalf("trashed data remains after purge: %v", err)
	}
}

func TestProfileDataLifecycleRejectsUnsafePaths(t *testing.T) {
	runtime, err := NewProcessRuntime(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.EnsureProfileData("../escape"); err == nil {
		t.Fatal("EnsureProfileData accepted unsafe ID")
	}
	if _, err := runtime.TrashProfileData("../escape"); err == nil {
		t.Fatal("TrashProfileData accepted unsafe ID")
	}
	if err := runtime.RestoreProfileData(runtimeTestProfileID, "../escape"); err == nil {
		t.Fatal("RestoreProfileData accepted unsafe token")
	}
	if err := runtime.PurgeProfileData(runtimeTestProfileID, "../escape"); err == nil {
		t.Fatal("PurgeProfileData accepted unsafe token")
	}
}
