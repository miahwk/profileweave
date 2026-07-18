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

type doctorRuntime struct {
	browsers []domain.BrowserDescriptor
	err      error
}

func (f doctorRuntime) Discover(context.Context) ([]domain.BrowserDescriptor, error) {
	return f.browsers, f.err
}

func (doctorRuntime) Launch(context.Context, domain.LaunchSpec) (domain.Process, error) {
	return domain.Process{}, nil
}

func (doctorRuntime) Stop(context.Context, string) error { return nil }

func (doctorRuntime) Info() domain.ProviderInfo {
	return domain.ProviderInfo{ID: "test-provider", Name: "Test Provider"}
}

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

func TestDoctorReportsProviderAndAvailableBrowsers(t *testing.T) {
	runtime := doctorRuntime{browsers: []domain.BrowserDescriptor{
		{ID: "chrome", Name: "Chrome", Path: `C:\\Users\\alice\\chrome.exe`, Available: true},
		{ID: "chromium", Name: "Chromium", Available: false},
	}}
	service := NewService(failingProfileReader{}, runtime)
	service.set(domain.Session{ProfileID: "running", Status: domain.StatusRunning})
	service.set(domain.Session{ProfileID: "stopped", Status: domain.StatusStopped})

	report, err := service.Doctor(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !report.Healthy || report.Provider.ID != "test-provider" {
		t.Fatalf("unexpected doctor health/provider: %#v", report)
	}
	if report.InspectedBrowsers != 2 || report.AvailableBrowsers != 1 {
		t.Fatalf("unexpected browser counts: %#v", report)
	}
	if report.ActiveSessions != 1 {
		t.Fatalf("active sessions = %d, want 1", report.ActiveSessions)
	}
	if report.Browsers == nil || report.Issues == nil {
		t.Fatal("doctor collections must encode as arrays")
	}
	if report.Provider.Capabilities == nil {
		t.Fatal("provider capabilities must encode as an array")
	}
	for _, browser := range report.Browsers {
		if browser.Path != "" {
			t.Fatalf("doctor exposed browser path %q", browser.Path)
		}
	}
}

func TestDoctorNormalizesFallbackRuntimeCollections(t *testing.T) {
	service := NewService(failingProfileReader{}, failingRuntime{})

	report, err := service.Doctor(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if report.Provider.ID != "custom-runtime" {
		t.Fatalf("fallback provider = %q", report.Provider.ID)
	}
	if report.Provider.Capabilities == nil || report.Browsers == nil || report.Issues == nil {
		t.Fatalf("fallback report contains nil collections: %#v", report)
	}
	if report.Healthy || len(report.Issues) != 1 || report.Issues[0].Code != "no_browser_available" {
		t.Fatalf("fallback report did not explain missing browsers: %#v", report)
	}
}

func TestDoctorExplainsMissingBrowser(t *testing.T) {
	service := NewService(failingProfileReader{}, doctorRuntime{
		browsers: []domain.BrowserDescriptor{{ID: "chrome", Name: "Chrome"}},
	})

	report, err := service.Doctor(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if report.Healthy || len(report.Issues) != 1 || report.Issues[0].Code != "no_browser_available" {
		t.Fatalf("missing browser was not explained: %#v", report)
	}
	if report.Issues[0].Suggestion == "" {
		t.Fatal("missing browser issue needs an actionable suggestion")
	}
}

func TestDoctorReturnsStructuredDiscoveryFailure(t *testing.T) {
	service := NewService(failingProfileReader{}, doctorRuntime{err: errors.New("private path details")})

	report, err := service.Doctor(context.Background())
	if err != nil {
		t.Fatalf("operational discovery errors should be structured: %v", err)
	}
	if report.Healthy || len(report.Issues) != 1 || report.Issues[0].Code != "browser_discovery_failed" {
		t.Fatalf("unexpected discovery report: %#v", report)
	}
	if strings.Contains(report.Issues[0].Message, "private") {
		t.Fatal("doctor report leaked runtime error details")
	}
}
