package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/miahwk/profileweave/internal/profile/domain"
)

const schemaVersion = 3

type diskTrashEntry struct {
	Profile          domain.Profile `json:"profile"`
	DeletedAt        time.Time      `json:"deletedAt"`
	DataRestoreToken string         `json:"dataRestoreToken,omitempty"`
}

type diskData struct {
	SchemaVersion int              `json:"schemaVersion"`
	Profiles      []domain.Profile `json:"profiles"`
	Trash         []diskTrashEntry `json:"trash"`
}

type JSONRepository struct {
	mu   sync.RWMutex
	path string
}

func NewJSONRepository(dataDir string) (*JSONRepository, error) {
	if dataDir == "" {
		return nil, errors.New("data directory is required")
	}
	abs, err := filepath.Abs(dataDir)
	if err != nil {
		return nil, fmt.Errorf("resolve data directory: %w", err)
	}
	if err := os.MkdirAll(abs, 0o700); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}
	repo := &JSONRepository{path: filepath.Join(abs, "profiles.json")}
	data, migrated, err := repo.readWithMigrationState()
	if err != nil {
		return nil, err
	}
	if migrated {
		if err := repo.write(data); err != nil {
			return nil, fmt.Errorf("persist profile schema migration: %w", err)
		}
	}
	return repo, nil
}

func (r *JSONRepository) List(_ context.Context) ([]domain.Profile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	data, err := r.read()
	if err != nil {
		return nil, err
	}
	out := append([]domain.Profile(nil), data.Profiles...)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (r *JSONRepository) Get(_ context.Context, id string) (domain.Profile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	data, err := r.read()
	if err != nil {
		return domain.Profile{}, err
	}
	for _, profile := range data.Profiles {
		if profile.ID == id {
			return profile, nil
		}
	}
	return domain.Profile{}, domain.ErrNotFound
}

func (r *JSONRepository) Save(_ context.Context, profile domain.Profile, expected uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	data, err := r.read()
	if err != nil {
		return err
	}
	index := -1
	for i := range data.Profiles {
		if data.Profiles[i].ID == profile.ID {
			index = i
			break
		}
	}
	if index < 0 {
		if expected != 0 {
			return domain.ErrConflict
		}
		data.Profiles = append(data.Profiles, profile)
	} else {
		if expected == 0 || data.Profiles[index].Revision != expected {
			return domain.ErrConflict
		}
		data.Profiles[index] = profile
	}
	return r.write(data)
}

func (r *JSONRepository) read() (diskData, error) {
	data, _, err := r.readWithMigrationState()
	return data, err
}

func (r *JSONRepository) readWithMigrationState() (diskData, bool, error) {
	raw, err := os.ReadFile(r.path)
	if errors.Is(err, os.ErrNotExist) {
		return emptyDiskData(), false, nil
	}
	if err != nil {
		return diskData{}, false, fmt.Errorf("read profiles: %w", err)
	}
	var data diskData
	if err := json.Unmarshal(raw, &data); err != nil {
		return diskData{}, false, fmt.Errorf("decode profiles: %w", err)
	}
	originalSchemaVersion := data.SchemaVersion
	switch data.SchemaVersion {
	case 1:
		data.Trash = []diskTrashEntry{}
		fallthrough
	case 2:
		disableLegacyCustomBrowsers(&data)
		data.SchemaVersion = schemaVersion
	case schemaVersion:
	default:
		return diskData{}, false, fmt.Errorf("unsupported profile schema version %d", data.SchemaVersion)
	}
	if data.Profiles == nil {
		data.Profiles = []domain.Profile{}
	}
	if data.Trash == nil {
		data.Trash = []diskTrashEntry{}
	}
	return data, originalSchemaVersion != schemaVersion, nil
}

func disableLegacyCustomBrowsers(data *diskData) {
	for i := range data.Profiles {
		if data.Profiles[i].Browser.Kind == "custom" {
			data.Profiles[i].Browser.Kind = "custom-disabled"
		}
	}
	for i := range data.Trash {
		if data.Trash[i].Profile.Browser.Kind == "custom" {
			data.Trash[i].Profile.Browser.Kind = "custom-disabled"
		}
	}
}

func emptyDiskData() diskData {
	return diskData{SchemaVersion: schemaVersion, Profiles: []domain.Profile{}, Trash: []diskTrashEntry{}}
}

func (r *JSONRepository) write(data diskData) error {
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("encode profiles: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(r.path), "profiles-*.tmp")
	if err != nil {
		return fmt.Errorf("create profile temporary file: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }
	defer cleanup()
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("secure profile temporary file: %w", err)
	}
	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write profiles: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync profiles: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close profiles: %w", err)
	}
	if err := replaceFile(tmpName, r.path); err != nil {
		return fmt.Errorf("replace profiles: %w", err)
	}
	return nil
}
