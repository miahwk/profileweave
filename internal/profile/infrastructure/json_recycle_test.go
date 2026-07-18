package infrastructure

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/miahwk/profileweave/internal/profile/domain"
)

func TestJSONRepositoryRecycleBinRoundTrip(t *testing.T) {
	dir := t.TempDir()
	repo, err := NewJSONRepository(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	profile := repositoryProfile("p_abcdef0123456789abcdef0123456789")
	if err := repo.Save(ctx, profile, 0); err != nil {
		t.Fatal(err)
	}
	deletedAt := time.Unix(1234, 0).UTC()
	if err := repo.MoveToTrash(ctx, profile.ID, "opaque-token", deletedAt); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Get(ctx, profile.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("active Get error = %v", err)
	}

	reopened, err := NewJSONRepository(dir)
	if err != nil {
		t.Fatal(err)
	}
	trash, err := reopened.ListTrash(ctx)
	if err != nil || len(trash) != 1 || trash[0].DataRestoreToken != "opaque-token" || !trash[0].DeletedAt.Equal(deletedAt) {
		t.Fatalf("trash=%#v err=%v", trash, err)
	}
	restored, err := reopened.RestoreTrash(ctx, profile.ID)
	if err != nil || restored.ID != profile.ID {
		t.Fatalf("restored=%#v err=%v", restored, err)
	}
	if err := reopened.MoveToTrash(ctx, profile.ID, "opaque-token", deletedAt); err != nil {
		t.Fatal(err)
	}
	if err := reopened.PurgeTrash(ctx, profile.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := reopened.GetTrash(ctx, profile.ID); !errors.Is(err, domain.ErrTrashNotFound) {
		t.Fatalf("GetTrash error = %v", err)
	}
}

func TestJSONRepositoryMigratesSchemaV1AtomicallyOnOpen(t *testing.T) {
	dir := t.TempDir()
	legacy := map[string]any{
		"schemaVersion": 1,
		"profiles":      []domain.Profile{repositoryProfile("p_11111111111111111111111111111111")},
	}
	raw, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "profiles.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	repo, err := NewJSONRepository(dir)
	if err != nil {
		t.Fatal(err)
	}
	_ = repo
	raw, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var persisted diskData
	if err := json.Unmarshal(raw, &persisted); err != nil || persisted.SchemaVersion != schemaVersion {
		t.Fatalf("schema=%d err=%v", persisted.SchemaVersion, err)
	}
}

func TestJSONRepositoryDisablesLegacyCustomBrowserPath(t *testing.T) {
	dir := t.TempDir()
	profile := repositoryProfile("p_22222222222222222222222222222222")
	encodedProfile, err := json.Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}
	var legacyProfile map[string]any
	if err := json.Unmarshal(encodedProfile, &legacyProfile); err != nil {
		t.Fatal(err)
	}
	legacyProfile["browser"] = map[string]any{"kind": "custom", "customPath": `C:\Portable\chrome.exe`}
	raw, err := json.Marshal(map[string]any{
		"schemaVersion": 2,
		"profiles":      []any{legacyProfile},
		"trash":         []any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "profiles.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}

	repo, err := NewJSONRepository(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, err := repo.Get(context.Background(), profile.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Browser.Kind != "custom-disabled" {
		t.Fatalf("legacy browser kind = %q, want custom-disabled", got.Browser.Kind)
	}
	persisted, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(persisted, []byte("customPath")) {
		t.Fatal("migration retained the legacy executable path")
	}
	var data diskData
	if err := json.Unmarshal(persisted, &data); err != nil || data.SchemaVersion != schemaVersion {
		t.Fatalf("schema=%d err=%v", data.SchemaVersion, err)
	}
}
