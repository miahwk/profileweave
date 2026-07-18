package application

import (
	"context"
	"errors"
	"testing"

	"github.com/miahwk/profileweave/internal/profile/domain"
)

func TestRecycleLifecycleRestoresAndPurges(t *testing.T) {
	repo := &repositoryStub{item: existingProfile(t)}
	data := &dataStub{trashToken: "restore-token"}
	service := NewService(repo, nil, data)
	ctx := context.Background()

	if err := service.Delete(ctx, testProfileID); err != nil {
		t.Fatal(err)
	}
	items, err := service.ListTrash(ctx)
	if err != nil || len(items) != 1 || items[0].Profile.ID != testProfileID {
		t.Fatalf("trash=%#v err=%v", items, err)
	}
	profile, err := service.Restore(ctx, testProfileID)
	if err != nil || profile.ID != testProfileID || data.restoreCalls != 1 {
		t.Fatalf("restore=%#v err=%v data=%#v", profile, err, data)
	}
	if err := service.Delete(ctx, testProfileID); err != nil {
		t.Fatal(err)
	}
	if err := service.Purge(ctx, testProfileID); err != nil {
		t.Fatal(err)
	}
	if repo.trash != nil || data.purgeCalls != 1 {
		t.Fatalf("purge repo=%#v data=%#v", repo.trash, data)
	}
}

func TestRestoreRollsBackDataWhenMetadataRestoreFails(t *testing.T) {
	restoreErr := errors.New("restore metadata failed")
	entry := domain.TrashedProfile{
		Profile: existingProfile(t), DataRestoreToken: "original-token", DeletedAt: testTime(),
	}
	repo := &repositoryStub{trash: &entry, restoreErr: restoreErr}
	data := &dataStub{}
	service := NewService(repo, nil, data)
	_, err := service.Restore(context.Background(), testProfileID)
	if !errors.Is(err, restoreErr) || data.restoreCalls != 1 || data.rollbackCalls != 1 {
		t.Fatalf("err=%v data=%#v", err, data)
	}
}

func TestRunningProfileCannotRestoreOrPurge(t *testing.T) {
	entry := domain.TrashedProfile{Profile: existingProfile(t), DataRestoreToken: "token", DeletedAt: testTime()}
	repo := &repositoryStub{trash: &entry}
	data := &dataStub{}
	service := NewService(repo, activityStub(true), data)
	if _, err := service.Restore(context.Background(), testProfileID); !errors.Is(err, ErrProfileRunning) {
		t.Fatalf("Restore error = %v", err)
	}
	if err := service.Purge(context.Background(), testProfileID); !errors.Is(err, ErrProfileRunning) {
		t.Fatalf("Purge error = %v", err)
	}
	if data.restoreCalls != 0 || data.purgeCalls != 0 {
		t.Fatalf("data lifecycle called while running: %#v", data)
	}
}
