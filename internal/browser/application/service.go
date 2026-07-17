package application

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/miahwk/profileweave/internal/browser/domain"
	fingerprint "github.com/miahwk/profileweave/internal/fingerprint/domain"
	profiledomain "github.com/miahwk/profileweave/internal/profile/domain"
)

var (
	ErrAlreadyRunning = errors.New("profile is already running")
	ErrNotRunning     = errors.New("profile is not running")
	ErrInvalidProfile = errors.New("profile has blocking consistency errors")
)

type ProfileReader interface {
	Get(context.Context, string) (profiledomain.Profile, error)
}

type Service struct {
	mu          sync.RWMutex
	lifecycleMu sync.Mutex
	profiles    ProfileReader
	runtime     domain.Runtime
	sessions    map[string]domain.Session
	now         func() time.Time
}

func NewService(profiles ProfileReader, runtime domain.Runtime) *Service {
	return &Service{profiles: profiles, runtime: runtime, sessions: make(map[string]domain.Session), now: time.Now}
}

func (s *Service) IsRunning(profileID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[profileID]
	return ok && active(session.Status)
}

// LockProfile serializes profile mutations with browser start/stop transitions.
// The local application favors lifecycle integrity over parallel mutations.
func (s *Service) LockProfile(_ string) func() {
	s.lifecycleMu.Lock()
	return s.lifecycleMu.Unlock
}

func (s *Service) List() []domain.Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]domain.Session, 0, len(s.sessions))
	for _, session := range s.sessions {
		items = append(items, session)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].StartedAt == nil {
			return false
		}
		if items[j].StartedAt == nil {
			return true
		}
		return items[i].StartedAt.After(*items[j].StartedAt)
	})
	return items
}

func (s *Service) Launch(ctx context.Context, profileID string) (domain.Session, error) {
	unlock := s.LockProfile(profileID)
	defer unlock()

	profile, err := s.profiles.Get(ctx, profileID)
	if err != nil {
		return domain.Session{}, err
	}
	if fingerprint.Evaluate(profile.Fingerprint, profile.Proxy).HasErrors() {
		return domain.Session{}, ErrInvalidProfile
	}

	s.mu.Lock()
	if session, ok := s.sessions[profileID]; ok && active(session.Status) {
		s.mu.Unlock()
		return domain.Session{}, ErrAlreadyRunning
	}
	s.sessions[profileID] = domain.Session{ProfileID: profileID, Status: domain.StatusStarting}
	s.mu.Unlock()

	process, err := s.runtime.Launch(ctx, toLaunchSpec(profile))
	if err != nil {
		now := s.now().UTC()
		failed := domain.Session{ProfileID: profileID, Status: domain.StatusFailed, StoppedAt: &now, LastError: safeRuntimeError(err)}
		s.set(failed)
		return failed, err
	}
	now := s.now().UTC()
	running := domain.Session{ProfileID: profileID, Status: domain.StatusRunning, PID: process.PID, StartedAt: &now}
	s.set(running)
	go s.observe(profileID, process.Done)
	return running, nil
}

func (s *Service) Stop(ctx context.Context, profileID string) (domain.Session, error) {
	unlock := s.LockProfile(profileID)
	defer unlock()

	s.mu.Lock()
	current, ok := s.sessions[profileID]
	if !ok || !active(current.Status) {
		s.mu.Unlock()
		return domain.Session{}, ErrNotRunning
	}
	current.Status = domain.StatusStopping
	s.sessions[profileID] = current
	s.mu.Unlock()

	if err := s.runtime.Stop(ctx, profileID); err != nil {
		// Keep the session active until the runtime confirms process-tree exit.
		// This prevents profile data deletion while a failed stop may have left
		// browser processes alive and also allows a later stop retry.
		current.Status = domain.StatusStopping
		current.LastError = safeRuntimeError(err)
		s.set(current)
		return current, err
	}
	now := s.now().UTC()
	current.Status, current.StoppedAt, current.LastError = domain.StatusStopped, &now, ""
	s.set(current)
	return current, nil
}

func (s *Service) Discover(ctx context.Context) ([]domain.BrowserDescriptor, error) {
	return s.runtime.Discover(ctx)
}

func (s *Service) observe(profileID string, done <-chan error) {
	err, ok := <-done
	s.mu.Lock()
	defer s.mu.Unlock()
	current, exists := s.sessions[profileID]
	if !exists || current.Status == domain.StatusStopped || current.Status == domain.StatusFailed {
		return
	}
	now := s.now().UTC()
	current.StoppedAt = &now
	if ok && err != nil {
		current.Status, current.LastError = domain.StatusFailed, safeRuntimeError(err)
	} else {
		current.Status = domain.StatusStopped
	}
	s.sessions[profileID] = current
}

func (s *Service) set(session domain.Session) {
	s.mu.Lock()
	s.sessions[session.ProfileID] = session
	s.mu.Unlock()
}

func active(status domain.Status) bool {
	return status == domain.StatusStarting || status == domain.StatusRunning || status == domain.StatusStopping
}

func safeRuntimeError(err error) string {
	if err == nil {
		return ""
	}
	return "browser process operation failed; verify the selected browser and try again"
}

func toLaunchSpec(profile profiledomain.Profile) domain.LaunchSpec {
	fp, proxy := profile.Fingerprint, profile.Proxy
	return domain.LaunchSpec{
		ProfileID: profile.ID, BrowserKind: profile.Browser.Kind, CustomPath: profile.Browser.CustomPath,
		StartURL: profile.StartURL, Locale: fp.Locale, Width: fp.Screen.Width, Height: fp.Screen.Height,
		DPR: fp.Screen.DPR, UAMode: fp.UAMode, UserAgent: fp.UserAgent,
		ProxyMode: proxy.Mode, ProxyHost: proxy.Host, ProxyPort: proxy.Port, WebRTCPolicy: fp.WebRTCPolicy,
	}
}
