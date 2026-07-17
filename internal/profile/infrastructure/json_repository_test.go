package infrastructure

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	fingerprint "github.com/miahwk/profileweave/internal/fingerprint/domain"
	"github.com/miahwk/profileweave/internal/profile/domain"
)

func repositoryProfile(id string) domain.Profile {
	input := domain.Input{
		Name: "Persisted", Browser: domain.Browser{Kind: "auto"}, Fingerprint: fingerprint.Default(),
		Proxy: fingerprint.Proxy{Mode: "direct"},
	}
	profile, err := domain.New(id, input, time.Now())
	if err != nil {
		panic(err)
	}
	return profile
}

func TestJSONRepositoryPersistsAndChecksRevision(t *testing.T) {
	dir := t.TempDir()
	repo, err := NewJSONRepository(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	profile := repositoryProfile("p_0123456789abcdef0123456789abcdef")
	if err := repo.Save(ctx, profile, 0); err != nil {
		t.Fatal(err)
	}
	profile.Name, profile.Revision = "Updated", 2
	if err := repo.Save(ctx, profile, 99); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
	if err := repo.Save(ctx, profile, 1); err != nil {
		t.Fatal(err)
	}
	reopened, err := NewJSONRepository(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, err := reopened.Get(ctx, profile.ID)
	if err != nil || got.Name != "Updated" || got.Revision != 2 {
		t.Fatalf("unexpected persisted profile %#v, %v", got, err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "profiles.json"))
	if err != nil || len(raw) == 0 || len(raw) > 0 && raw[0] != '{' {
		t.Fatalf("invalid JSON file: %q, %v", raw, err)
	}
}

func TestJSONRepositorySerializesConcurrentCreates(t *testing.T) {
	repo, err := NewJSONRepository(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{
		"p_00000000000000000000000000000000", "p_11111111111111111111111111111111",
		"p_22222222222222222222222222222222", "p_33333333333333333333333333333333",
	}
	var wg sync.WaitGroup
	for _, id := range ids {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			if err := repo.Save(context.Background(), repositoryProfile(id), 0); err != nil {
				t.Errorf("save %s: %v", id, err)
			}
		}(id)
	}
	wg.Wait()
	items, err := repo.List(context.Background())
	if err != nil || len(items) != len(ids) {
		t.Fatalf("got %d profiles: %v", len(items), err)
	}
}
