package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/miahwk/profileweave/internal/profile/domain"
)

func (s *Service) ListTrash(ctx context.Context) ([]domain.TrashedProfile, error) {
	return s.repo.ListTrash(ctx)
}

func (s *Service) Restore(ctx context.Context, id string) (domain.Profile, error) {
	if !domain.ValidID(id) {
		return domain.Profile{}, domain.ErrTrashNotFound
	}
	unlock := s.lockLifecycle(id)
	defer unlock()

	trashed, err := s.repo.GetTrash(ctx, id)
	if err != nil {
		return domain.Profile{}, err
	}
	if s.activity != nil && s.activity.IsRunning(id) {
		return domain.Profile{}, ErrProfileRunning
	}
	dataRestored := false
	if trashed.DataRestoreToken != "" {
		if s.data == nil {
			return domain.Profile{}, errors.New("profile data lifecycle is unavailable")
		}
		if err := s.data.RestoreProfileData(id, trashed.DataRestoreToken); err != nil {
			return domain.Profile{}, err
		}
		dataRestored = true
	}
	profile, err := s.repo.RestoreTrash(ctx, id)
	if err == nil {
		return profile, nil
	}
	if dataRestored {
		if rollbackErr := s.data.RollbackRestoredProfileData(id, trashed.DataRestoreToken); rollbackErr != nil {
			return domain.Profile{}, errors.Join(err, fmt.Errorf("roll back restored profile data: %w", rollbackErr))
		}
	}
	return domain.Profile{}, err
}

func (s *Service) Purge(ctx context.Context, id string) error {
	if !domain.ValidID(id) {
		return domain.ErrTrashNotFound
	}
	unlock := s.lockLifecycle(id)
	defer unlock()

	trashed, err := s.repo.GetTrash(ctx, id)
	if err != nil {
		return err
	}
	if s.activity != nil && s.activity.IsRunning(id) {
		return ErrProfileRunning
	}
	if trashed.DataRestoreToken != "" {
		if s.data == nil {
			return errors.New("profile data lifecycle is unavailable")
		}
		if err := s.data.PurgeProfileData(id, trashed.DataRestoreToken); err != nil {
			return err
		}
	}
	return s.repo.PurgeTrash(ctx, id)
}
