package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/miahwk/profileweave/internal/browser/domain"
	fingerprint "github.com/miahwk/profileweave/internal/fingerprint/domain"
	profiledomain "github.com/miahwk/profileweave/internal/profile/domain"
)

type failingProfileReader struct{ profile profiledomain.Profile }

func (f failingProfileReader) Get(context.Context, string) (profiledomain.Profile, error) {
	return f.profile, nil
}

type failingRuntime struct{ err error }

func (f failingRuntime) Discover(context.Context) ([]domain.BrowserDescriptor, error) {
	return nil, nil
}

func (f failingRuntime) Launch(context.Context, domain.LaunchSpec) (domain.Process, error) {
	return domain.Process{}, f.err
}

func (f failingRuntime) Stop(context.Context, string) error { return f.err }

type stopFailRuntime struct {
	done chan error
	err  error
}

func (f stopFailRuntime) Discover(context.Context) ([]domain.BrowserDescriptor, error) {
	return nil, nil
}

func (f stopFailRuntime) Launch(context.Context, domain.LaunchSpec) (domain.Process, error) {
	return domain.Process{PID: 4321, Done: f.done}, nil
}

func (f stopFailRuntime) Stop(context.Context, string) error { return f.err }

func TestSafeRuntimeErrorDoesNotExposeSensitiveDetails(t *testing.T) {
	sensitive := strings.Join([]string{
		`C:\\Users\\alice\\private\\chrome.exe`,
		"socks5://secret.proxy.example:1080",
		"https://example.test/start?token=top-secret",
		"--user-agent=private-argv-value",
	}, " ")

	message := safeRuntimeError(errors.New(sensitive))
	if message != "browser process operation failed; verify the selected browser and try again" {
		t.Fatalf("unexpected safe message %q", message)
	}
	for _, secret := range []string{"alice", "secret.proxy.example", "top-secret", "private-argv-value"} {
		if strings.Contains(message, secret) {
			t.Fatalf("safe message leaked %q: %q", secret, message)
		}
	}
}

func TestSafeRuntimeErrorKeepsEmptyErrorEmpty(t *testing.T) {
	if message := safeRuntimeError(nil); message != "" {
		t.Fatalf("nil error produced %q", message)
	}
}

func TestLaunchFailureStoresOnlySafeSessionError(t *testing.T) {
	const profileID = "p_0123456789abcdef0123456789abcdef"
	sensitive := errors.New(`fork/exec C:\\Users\\alice\\chrome.exe --proxy-server=socks5://secret.proxy:1080 https://example.test/?token=secret`)
	profile := profiledomain.Profile{
		ID:          profileID,
		Browser:     profiledomain.Browser{Kind: "auto"},
		Fingerprint: fingerprint.Default(),
		Proxy:       fingerprint.Proxy{Mode: "direct"},
	}
	service := NewService(failingProfileReader{profile: profile}, failingRuntime{err: sensitive})

	session, err := service.Launch(context.Background(), profileID)
	if !errors.Is(err, sensitive) {
		t.Fatalf("launch error = %v, want original error", err)
	}
	if session.Status != domain.StatusFailed {
		t.Fatalf("session status = %s, want failed", session.Status)
	}
	if session.LastError != "browser process operation failed; verify the selected browser and try again" {
		t.Fatalf("unsafe session error %q", session.LastError)
	}
	for _, secret := range []string{"alice", "secret.proxy", "token=secret"} {
		if strings.Contains(session.LastError, secret) {
			t.Fatalf("session error leaked %q: %q", secret, session.LastError)
		}
	}
}

func TestStopFailureRemainsActiveUntilExitIsConfirmed(t *testing.T) {
	const profileID = "p_0123456789abcdef0123456789abcdef"
	sensitive := errors.New(`taskkill failed for C:\\Users\\alice\\chrome.exe`)
	profile := profiledomain.Profile{
		ID:          profileID,
		Browser:     profiledomain.Browser{Kind: "auto"},
		Fingerprint: fingerprint.Default(),
		Proxy:       fingerprint.Proxy{Mode: "direct"},
	}
	runtime := stopFailRuntime{done: make(chan error), err: sensitive}
	defer close(runtime.done)
	service := NewService(failingProfileReader{profile: profile}, runtime)
	if _, err := service.Launch(context.Background(), profileID); err != nil {
		t.Fatal(err)
	}

	session, err := service.Stop(context.Background(), profileID)
	if !errors.Is(err, sensitive) {
		t.Fatalf("stop error = %v, want original error", err)
	}
	if session.Status != domain.StatusStopping || !service.IsRunning(profileID) {
		t.Fatalf("failed stop must remain active, got %#v", session)
	}
	if session.StoppedAt != nil || strings.Contains(session.LastError, "alice") {
		t.Fatalf("failed stop exposed sensitive data or claimed exit: %#v", session)
	}
}
