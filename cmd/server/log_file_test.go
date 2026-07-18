package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRotatingLogWriterKeepsOneBackup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "profileweave.log")
	writer, err := newRotatingLogWriter(path)
	if err != nil {
		t.Fatal(err)
	}
	writer.size = maxLogFileBytes
	if _, err := writer.Write([]byte("new log entry\n")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path + ".1"); err != nil {
		t.Fatalf("rotated backup: %v", err)
	}
	contents, err := os.ReadFile(path)
	if err != nil || string(contents) != "new log entry\n" {
		t.Fatalf("active log = %q, %v", contents, err)
	}
}

func TestRotationFailureKeepsActiveWriterUsable(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "profileweave.log")
	writer, err := newRotatingLogWriter(path)
	if err != nil {
		t.Fatal(err)
	}
	backup := path + ".1"
	if err := os.Mkdir(backup, 0o700); err != nil {
		t.Fatal(err)
	}
	blocker := filepath.Join(backup, "locked")
	if err := os.WriteFile(blocker, []byte("block rotation"), 0o600); err != nil {
		t.Fatal(err)
	}
	writer.size = maxLogFileBytes
	if _, err := writer.Write([]byte("first attempt\n")); err == nil {
		t.Fatal("rotation unexpectedly succeeded with a non-empty backup directory")
	}
	if writer.file == nil {
		t.Fatal("rotation failure left the writer without an active file")
	}
	if err := os.Remove(blocker); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(backup); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte("recovered\n")); err != nil {
		t.Fatalf("writer did not recover after rotation obstruction cleared: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
}
