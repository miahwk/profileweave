package instancelock

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gofrs/flock"
)

var ErrDataDirInUse = errors.New("profile data directory is already in use")

type Lock struct {
	handle *flock.Flock
}

func Acquire(dataDir string) (*Lock, error) {
	root, err := filepath.Abs(dataDir)
	if err != nil {
		return nil, fmt.Errorf("resolve profile data directory: %w", err)
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, fmt.Errorf("create profile data directory: %w", err)
	}
	handle := flock.New(filepath.Join(root, ".profileweave.lock"), flock.SetPermissions(0o600))
	locked, err := handle.TryLock()
	if err != nil {
		_ = handle.Close()
		return nil, fmt.Errorf("lock profile data directory: %w", err)
	}
	if !locked {
		_ = handle.Close()
		return nil, ErrDataDirInUse
	}
	return &Lock{handle: handle}, nil
}

func (l *Lock) Close() error {
	if l == nil || l.handle == nil {
		return nil
	}
	err := l.handle.Unlock()
	l.handle = nil
	return err
}
