package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	fingerprint "github.com/miahwk/profileweave/internal/fingerprint/domain"
	"github.com/miahwk/profileweave/internal/profile/domain"
)

const testProfileID = "p_0123456789abcdef0123456789abcdef"

type repositoryStub struct {
	item       domain.Profile
	trash      *domain.TrashedProfile
	saveErr    error
	deleteErr  error
	restoreErr error
	purgeErr   error
	deleted    bool
}

func (r *repositoryStub) List(context.Context) ([]domain.Profile, error) {
	if r.item.ID == "" {
		return []domain.Profile{}, nil
	}
	return []domain.Profile{r.item}, nil
}

func (r *repositoryStub) Get(_ context.Context, id string) (domain.Profile, error) {
	if id != r.item.ID {
		return domain.Profile{}, domain.ErrNotFound
	}
	return r.item, nil
}

func (r *repositoryStub) Save(_ context.Context, profile domain.Profile, _ uint64) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	r.item = profile
	return nil
}

func (r *repositoryStub) ListTrash(context.Context) ([]domain.TrashedProfile, error) {
	if r.trash == nil {
		return []domain.TrashedProfile{}, nil
	}
	return []domain.TrashedProfile{*r.trash}, nil
}

func (r *repositoryStub) GetTrash(_ context.Context, id string) (domain.TrashedProfile, error) {
	if r.trash == nil || r.trash.Profile.ID != id {
		return domain.TrashedProfile{}, domain.ErrTrashNotFound
	}
	return *r.trash, nil
}

func (r *repositoryStub) MoveToTrash(_ context.Context, id, token string, deletedAt time.Time) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	if id != r.item.ID {
		return domain.ErrNotFound
	}
	entry := domain.TrashedProfile{Profile: r.item, DeletedAt: deletedAt, DataRestoreToken: token}
	r.trash = &entry
	r.item = domain.Profile{}
	r.deleted = true
	return nil
}

func (r *repositoryStub) RestoreTrash(_ context.Context, id string) (domain.Profile, error) {
	if r.restoreErr != nil {
		return domain.Profile{}, r.restoreErr
	}
	if r.trash == nil || r.trash.Profile.ID != id {
		return domain.Profile{}, domain.ErrTrashNotFound
	}
	r.item = r.trash.Profile
	r.trash = nil
	return r.item, nil
}

func (r *repositoryStub) PurgeTrash(_ context.Context, id string) error {
	if r.purgeErr != nil {
		return r.purgeErr
	}
	if r.trash == nil || r.trash.Profile.ID != id {
		return domain.ErrTrashNotFound
	}
	r.trash = nil
	return nil
}

type dataStub struct {
	created       bool
	trashToken    string
	trashCalls    int
	restoreCalls  int
	restoredToken string
	rollbackCalls int
	purgeCalls    int
}

func (d *dataStub) EnsureProfileData(string) (bool, error) { return d.created, nil }
func (d *dataStub) TrashProfileData(string) (string, error) {
	d.trashCalls++
	return d.trashToken, nil
}
func (d *dataStub) RestoreProfileData(_ string, token string) error {
	d.restoreCalls++
	d.restoredToken = token
	return nil
}
func (d *dataStub) RollbackRestoredProfileData(_ string, token string) error {
	d.rollbackCalls++
	d.restoredToken = token
	return nil
}
func (d *dataStub) PurgeProfileData(string, string) error {
	d.purgeCalls++
	return nil
}

type activityStub bool

func (a activityStub) IsRunning(string) bool { return bool(a) }

type blockingActivity struct {
	entered chan struct{}
	release chan struct{}
}

func (a *blockingActivity) IsRunning(string) bool { return false }
func (a *blockingActivity) LockProfile(string) func() {
	close(a.entered)
	<-a.release
	return func() {}
}

func validInput() domain.Input {
	return domain.Input{
		Name: "Test", Browser: domain.Browser{Kind: "auto"},
		Fingerprint: fingerprint.Default(), Proxy: fingerprint.Proxy{Mode: "direct"},
	}
}

func existingProfile(t *testing.T) domain.Profile {
	t.Helper()
	profile, err := domain.New(testProfileID, validInput(), testTime())
	if err != nil {
		t.Fatal(err)
	}
	return profile
}

func TestCreateArchivesNewDataWhenMetadataSaveFails(t *testing.T) {
	saveErr := errors.New("save failed")
	for _, test := range []struct {
		name        string
		created     bool
		wantTrashes int
	}{{"new directory", true, 1}, {"preexisting directory", false, 0}} {
		t.Run(test.name, func(t *testing.T) {
			repo := &repositoryStub{saveErr: saveErr}
			data := &dataStub{created: test.created, trashToken: "opaque"}
			service := NewService(repo, nil, data)
			service.newID = func() (string, error) { return testProfileID, nil }
			_, err := service.Create(context.Background(), validInput())
			if !errors.Is(err, saveErr) || data.trashCalls != test.wantTrashes {
				t.Fatalf("error=%v trash calls=%d", err, data.trashCalls)
			}
		})
	}
}

func TestDeleteRestoresDataWhenMetadataDeleteFails(t *testing.T) {
	deleteErr := errors.New("delete failed")
	repo := &repositoryStub{item: existingProfile(t), deleteErr: deleteErr}
	data := &dataStub{trashToken: "restore-me"}
	service := NewService(repo, nil, data)
	if err := service.Delete(context.Background(), testProfileID); !errors.Is(err, deleteErr) {
		t.Fatalf("Delete error = %v", err)
	}
	if data.trashCalls != 1 || data.restoreCalls != 1 || data.restoredToken != "restore-me" {
		t.Fatalf("unexpected lifecycle calls: %#v", data)
	}
}

func TestRunningProfileCannotBeUpdatedOrDeleted(t *testing.T) {
	repo := &repositoryStub{item: existingProfile(t)}
	service := NewService(repo, activityStub(true))
	if _, err := service.Update(context.Background(), testProfileID, 1, validInput()); !errors.Is(err, ErrProfileRunning) {
		t.Fatalf("Update error = %v", err)
	}
	if err := service.Delete(context.Background(), testProfileID); !errors.Is(err, ErrProfileRunning) {
		t.Fatalf("Delete error = %v", err)
	}
}

func TestDeleteRejectsInvalidIDBeforeDataLifecycle(t *testing.T) {
	repo := &repositoryStub{item: existingProfile(t)}
	data := &dataStub{trashToken: "unexpected"}
	service := NewService(repo, nil, data)
	if err := service.Delete(context.Background(), "../browser-data"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Delete error = %v", err)
	}
	if data.trashCalls != 0 {
		t.Fatalf("trash called %d times", data.trashCalls)
	}
}

func TestDeleteWaitsForSharedBrowserLifecycleLock(t *testing.T) {
	repo := &repositoryStub{item: existingProfile(t)}
	data := &dataStub{trashToken: "restore-me"}
	activity := &blockingActivity{entered: make(chan struct{}), release: make(chan struct{})}
	service := NewService(repo, activity, data)
	done := make(chan error, 1)
	go func() { done <- service.Delete(context.Background(), testProfileID) }()

	select {
	case <-activity.entered:
	case <-time.After(time.Second):
		t.Fatal("delete did not acquire the browser lifecycle lock")
	}
	if data.trashCalls != 0 {
		t.Fatal("delete changed browser data before acquiring the lifecycle lock")
	}
	close(activity.release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestDuplicateNamePreservesUTF8AndCharacterLimit(t *testing.T) {
	name := duplicateName("环")
	if name != "环 copy" {
		t.Fatalf("short duplicate name = %q", name)
	}
	long := duplicateName(strings.Repeat("隔", 100))
	if len([]rune(long)) > 100 {
		t.Fatalf("duplicate name has %d characters", len([]rune(long)))
	}
	if !strings.HasSuffix(long, " copy") {
		t.Fatalf("duplicate name suffix missing: %q", long)
	}
}

func testTime() time.Time { return time.Unix(1, 0) }
