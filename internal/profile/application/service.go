package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	fingerprint "github.com/miahwk/profileweave/internal/fingerprint/domain"
	"github.com/miahwk/profileweave/internal/profile/domain"
)

var ErrProfileRunning = errors.New("profile is running")

type Activity interface {
	IsRunning(profileID string) bool
}

type LifecycleLocker interface {
	LockProfile(profileID string) func()
}

type PathValidator interface {
	ValidateExecutable(path string) error
}

type DataProvisioner interface {
	EnsureProfileData(profileID string) (created bool, err error)
	TrashProfileData(profileID string) (restoreToken string, err error)
	RestoreProfileData(profileID, restoreToken string) error
	RollbackRestoredProfileData(profileID, restoreToken string) error
	PurgeProfileData(profileID, restoreToken string) error
}

type Service struct {
	repo     domain.Repository
	activity Activity
	paths    PathValidator
	data     DataProvisioner
	now      func() time.Time
	newID    func() (string, error)
}

func NewService(repo domain.Repository, activity Activity, paths PathValidator, data ...DataProvisioner) *Service {
	service := &Service{repo: repo, activity: activity, paths: paths, now: time.Now, newID: generateID}
	if len(data) > 0 {
		service.data = data[0]
	}
	return service
}

func (s *Service) List(ctx context.Context, search string) ([]domain.Profile, error) {
	profiles, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	needle := strings.ToLower(strings.TrimSpace(search))
	if needle != "" {
		filtered := make([]domain.Profile, 0)
		for _, profile := range profiles {
			text := strings.ToLower(profile.Name + " " + profile.Notes + " " + strings.Join(profile.Tags, " "))
			if strings.Contains(text, needle) {
				filtered = append(filtered, profile)
			}
		}
		profiles = filtered
	}
	sort.Slice(profiles, func(i, j int) bool { return profiles[i].UpdatedAt.After(profiles[j].UpdatedAt) })
	return profiles, nil
}

func (s *Service) Get(ctx context.Context, id string) (domain.Profile, error) {
	if !domain.ValidID(id) {
		return domain.Profile{}, domain.ErrNotFound
	}
	return s.repo.Get(ctx, id)
}

func (s *Service) Create(ctx context.Context, in domain.Input) (domain.Profile, error) {
	if err := s.validatePath(in); err != nil {
		return domain.Profile{}, err
	}
	id, err := s.newID()
	if err != nil {
		return domain.Profile{}, err
	}
	profile, err := domain.New(id, in, s.now())
	if err != nil {
		return domain.Profile{}, err
	}
	dataCreated := false
	if s.data != nil {
		dataCreated, err = s.data.EnsureProfileData(profile.ID)
		if err != nil {
			return domain.Profile{}, err
		}
	}
	if err := s.repo.Save(ctx, profile, 0); err != nil {
		if dataCreated {
			token, cleanupErr := s.data.TrashProfileData(profile.ID)
			if cleanupErr != nil {
				return domain.Profile{}, errors.Join(err, fmt.Errorf("archive unused profile data: %w", cleanupErr))
			}
			if token != "" {
				if cleanupErr := s.data.PurgeProfileData(profile.ID, token); cleanupErr != nil {
					return domain.Profile{}, errors.Join(err, fmt.Errorf("remove unused profile data: %w", cleanupErr))
				}
			}
		}
		return domain.Profile{}, err
	}
	return profile, nil
}

func (s *Service) Update(ctx context.Context, id string, expected uint64, in domain.Input) (domain.Profile, error) {
	unlock := s.lockLifecycle(id)
	defer unlock()

	if err := s.validatePath(in); err != nil {
		return domain.Profile{}, err
	}
	current, err := s.Get(ctx, id)
	if err != nil {
		return domain.Profile{}, err
	}
	if s.activity != nil && s.activity.IsRunning(id) {
		return domain.Profile{}, ErrProfileRunning
	}
	if expected == 0 || current.Revision != expected {
		return domain.Profile{}, domain.ErrConflict
	}
	updated, err := current.Update(in, s.now())
	if err != nil {
		return domain.Profile{}, err
	}
	if err := s.repo.Save(ctx, updated, expected); err != nil {
		return domain.Profile{}, err
	}
	return updated, nil
}

func (s *Service) Duplicate(ctx context.Context, id string) (domain.Profile, error) {
	profile, err := s.Get(ctx, id)
	if err != nil {
		return domain.Profile{}, err
	}
	in := profile.Input()
	in.Name = duplicateName(in.Name)
	return s.Create(ctx, in)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	unlock := s.lockLifecycle(id)
	defer unlock()

	if _, err := s.Get(ctx, id); err != nil {
		return err
	}
	if s.activity != nil && s.activity.IsRunning(id) {
		return ErrProfileRunning
	}
	restoreToken := ""
	if s.data != nil {
		var err error
		restoreToken, err = s.data.TrashProfileData(id)
		if err != nil {
			return err
		}
	}
	if err := s.repo.MoveToTrash(ctx, id, restoreToken, s.now()); err != nil {
		if restoreToken != "" {
			if restoreErr := s.data.RestoreProfileData(id, restoreToken); restoreErr != nil {
				return errors.Join(err, fmt.Errorf("restore profile data: %w", restoreErr))
			}
		}
		return err
	}
	return nil
}

func (s *Service) Validate(ctx context.Context, id string) (fingerprint.Report, error) {
	profile, err := s.Get(ctx, id)
	if err != nil {
		return fingerprint.Report{}, err
	}
	return fingerprint.Evaluate(profile.Fingerprint, profile.Proxy), nil
}

func (s *Service) validatePath(in domain.Input) error {
	if in.Browser.Kind != "custom" || s.paths == nil {
		return nil
	}
	if err := s.paths.ValidateExecutable(in.Browser.CustomPath); err != nil {
		return &domain.ValidationError{Details: []domain.FieldError{{Field: "browser.customPath", Message: err.Error()}}}
	}
	return nil
}

func (s *Service) lockLifecycle(id string) func() {
	if locker, ok := s.activity.(LifecycleLocker); ok {
		return locker.LockProfile(id)
	}
	return func() {}
}

func generateID() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return "p_" + hex.EncodeToString(raw), nil
}

func duplicateName(name string) string {
	const suffix = " copy"
	nameRunes, suffixRunes := []rune(name), []rune(suffix)
	if len(nameRunes)+len(suffixRunes) <= 100 {
		return name + suffix
	}
	return string(nameRunes[:100-len(suffixRunes)]) + suffix
}
