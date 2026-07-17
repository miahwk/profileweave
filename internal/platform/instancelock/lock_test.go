package instancelock

import (
	"errors"
	"testing"
)

func TestAcquireRejectsConcurrentOwnerAndAllowsReuse(t *testing.T) {
	dir := t.TempDir()
	first, err := Acquire(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Acquire(dir); !errors.Is(err, ErrDataDirInUse) {
		t.Fatalf("expected data directory conflict, got %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	second, err := Acquire(dir)
	if err != nil {
		t.Fatalf("expected lock reuse after close: %v", err)
	}
	if err := second.Close(); err != nil {
		t.Fatal(err)
	}
}
