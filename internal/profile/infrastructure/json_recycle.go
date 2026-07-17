package infrastructure

import (
	"context"
	"sort"
	"time"

	"github.com/miahwk/profileweave/internal/profile/domain"
)

func (r *JSONRepository) ListTrash(_ context.Context) ([]domain.TrashedProfile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	data, err := r.read()
	if err != nil {
		return nil, err
	}
	items := make([]domain.TrashedProfile, 0, len(data.Trash))
	for _, entry := range data.Trash {
		items = append(items, toDomainTrash(entry))
	}
	sortTrash(items)
	return items, nil
}

func (r *JSONRepository) GetTrash(_ context.Context, id string) (domain.TrashedProfile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	data, err := r.read()
	if err != nil {
		return domain.TrashedProfile{}, err
	}
	for _, entry := range data.Trash {
		if entry.Profile.ID == id {
			return toDomainTrash(entry), nil
		}
	}
	return domain.TrashedProfile{}, domain.ErrTrashNotFound
}

func (r *JSONRepository) MoveToTrash(_ context.Context, id, token string, deletedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	data, err := r.read()
	if err != nil {
		return err
	}
	for _, entry := range data.Trash {
		if entry.Profile.ID == id {
			return domain.ErrConflict
		}
	}
	for i, profile := range data.Profiles {
		if profile.ID == id {
			data.Profiles = append(data.Profiles[:i], data.Profiles[i+1:]...)
			data.Trash = append(data.Trash, diskTrashEntry{
				Profile: profile, DeletedAt: deletedAt.UTC(), DataRestoreToken: token,
			})
			return r.write(data)
		}
	}
	return domain.ErrNotFound
}

func (r *JSONRepository) RestoreTrash(_ context.Context, id string) (domain.Profile, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	data, err := r.read()
	if err != nil {
		return domain.Profile{}, err
	}
	for _, profile := range data.Profiles {
		if profile.ID == id {
			return domain.Profile{}, domain.ErrConflict
		}
	}
	for i, entry := range data.Trash {
		if entry.Profile.ID == id {
			data.Trash = append(data.Trash[:i], data.Trash[i+1:]...)
			data.Profiles = append(data.Profiles, entry.Profile)
			if err := r.write(data); err != nil {
				return domain.Profile{}, err
			}
			return entry.Profile, nil
		}
	}
	return domain.Profile{}, domain.ErrTrashNotFound
}

func (r *JSONRepository) PurgeTrash(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	data, err := r.read()
	if err != nil {
		return err
	}
	for i, entry := range data.Trash {
		if entry.Profile.ID == id {
			data.Trash = append(data.Trash[:i], data.Trash[i+1:]...)
			return r.write(data)
		}
	}
	return domain.ErrTrashNotFound
}

func toDomainTrash(entry diskTrashEntry) domain.TrashedProfile {
	return domain.TrashedProfile{
		Profile: entry.Profile, DeletedAt: entry.DeletedAt, DataRestoreToken: entry.DataRestoreToken,
	}
}

func sortTrash(items []domain.TrashedProfile) {
	sort.Slice(items, func(i, j int) bool { return items[i].DeletedAt.After(items[j].DeletedAt) })
}
