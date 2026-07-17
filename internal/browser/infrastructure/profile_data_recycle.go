package infrastructure

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func (r *ProcessRuntime) RollbackRestoredProfileData(profileID, restoreToken string) error {
	profileDir, err := r.profileDataPath(profileID)
	if err != nil {
		return err
	}
	trashed, err := r.trashedDataPath(profileID, restoreToken)
	if err != nil {
		return err
	}
	r.dataMu.Lock()
	defer r.dataMu.Unlock()
	info, err := os.Lstat(profileDir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("restored profile browser directory is unavailable")
	}
	if _, err := os.Lstat(trashed); !errors.Is(err, os.ErrNotExist) {
		return errors.New("profile data restore target already exists")
	}
	if err := os.Rename(profileDir, trashed); err != nil {
		return fmt.Errorf("roll back restored profile browser data: %w", err)
	}
	return nil
}

func (r *ProcessRuntime) PurgeProfileData(profileID, restoreToken string) error {
	trashed, err := r.trashedDataPath(profileID, restoreToken)
	if err != nil {
		return err
	}
	r.dataMu.Lock()
	defer r.dataMu.Unlock()
	info, err := os.Lstat(trashed)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("trashed profile browser path is not a directory")
	}
	if err := os.RemoveAll(trashed); err != nil {
		return fmt.Errorf("permanently remove profile browser data: %w", err)
	}
	return nil
}

func (r *ProcessRuntime) trashedDataPath(profileID, restoreToken string) (string, error) {
	if _, err := r.profileDataPath(profileID); err != nil {
		return "", err
	}
	if !validRestoreToken(profileID, restoreToken) {
		return "", errors.New("invalid profile data restore token")
	}
	trashed := filepath.Join(r.trashRoot, restoreToken)
	if filepath.Dir(trashed) != filepath.Clean(r.trashRoot) {
		return "", errors.New("profile data trash path escaped its root")
	}
	return trashed, nil
}
