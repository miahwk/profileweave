package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"sync/atomic"
	"testing"
	"time"

	browserapp "github.com/miahwk/profileweave/internal/browser/application"
	"github.com/miahwk/profileweave/internal/browser/domain"
	fingerprint "github.com/miahwk/profileweave/internal/fingerprint/domain"
	profiledomain "github.com/miahwk/profileweave/internal/profile/domain"
)

type shutdownProfileReader struct{ profile profiledomain.Profile }

func (r shutdownProfileReader) Get(context.Context, string) (profiledomain.Profile, error) {
	return r.profile, nil
}

type retryStopRuntime struct {
	calls    atomic.Int32
	attempts chan int
	done     chan error
}

type delayedStopRuntime struct {
	delay time.Duration
	done  chan error
}

func (r *delayedStopRuntime) Discover(context.Context) ([]domain.BrowserDescriptor, error) {
	return nil, nil
}

func (r *delayedStopRuntime) Launch(context.Context, domain.LaunchSpec) (domain.Process, error) {
	return domain.Process{PID: 4321, Done: r.done}, nil
}

func (r *delayedStopRuntime) Stop(ctx context.Context, _ string) error {
	select {
	case <-time.After(r.delay):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *retryStopRuntime) Discover(context.Context) ([]domain.BrowserDescriptor, error) {
	return nil, nil
}

func (r *retryStopRuntime) Launch(context.Context, domain.LaunchSpec) (domain.Process, error) {
	return domain.Process{PID: 4321, Done: r.done}, nil
}

func (r *retryStopRuntime) Stop(context.Context, string) error {
	attempt := int(r.calls.Add(1))
	r.attempts <- attempt
	if attempt == 1 {
		return errors.New("temporary stop failure")
	}
	return nil
}

func TestCleanShutdownStaysAliveUntilSessionStopRetrySucceeds(t *testing.T) {
	const profileID = "p_0123456789abcdef0123456789abcdef"
	profile := profiledomain.Profile{
		ID: profileID, Browser: profiledomain.Browser{Kind: "auto"},
		Fingerprint: fingerprint.Default(), Proxy: fingerprint.Proxy{Mode: "direct"},
	}
	runtime := &retryStopRuntime{attempts: make(chan int, 2), done: make(chan error)}
	defer close(runtime.done)
	browsers := browserapp.NewService(shutdownProfileReader{profile: profile}, runtime)
	if _, err := browsers.Launch(context.Background(), profileID); err != nil {
		t.Fatal(err)
	}

	stopRequests := make(chan struct{}, 1)
	serveErrors := make(chan error, 1)
	signals := make(chan os.Signal, 1)
	server := newHTTPServer("127.0.0.1:0", http.NotFoundHandler())
	finished := make(chan error, 1)
	stopRequests <- struct{}{}
	go func() {
		finished <- waitForCleanShutdown(browsers, server, stopRequests, serveErrors, signals)
	}()

	waitForStopAttempt(t, runtime.attempts, 1)
	select {
	case err := <-finished:
		t.Fatalf("application exited after failed session cleanup: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	if _, err := browsers.Launch(context.Background(), profileID); !errors.Is(err, browserapp.ErrShuttingDown) {
		t.Fatalf("launch during cleanup retry = %v, want ErrShuttingDown", err)
	}

	stopRequests <- struct{}{}
	waitForStopAttempt(t, runtime.attempts, 2)
	select {
	case err := <-finished:
		if err != nil {
			t.Fatalf("clean shutdown retry: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("application did not exit after successful cleanup retry")
	}
}

func TestStopBrowserSessionsGivesEachSessionAFullTimeout(t *testing.T) {
	runtime := &delayedStopRuntime{delay: 30 * time.Millisecond, done: make(chan error)}
	defer close(runtime.done)
	browsers := browserapp.NewService(shutdownProfileReader{profile: profiledomain.Profile{
		Browser:     profiledomain.Browser{Kind: "auto"},
		Fingerprint: fingerprint.Default(), Proxy: fingerprint.Proxy{Mode: "direct"},
	}}, runtime)
	for _, profileID := range []string{
		"p_0123456789abcdef0123456789abcdef",
		"p_1123456789abcdef0123456789abcdef",
	} {
		if _, err := browsers.Launch(context.Background(), profileID); err != nil {
			t.Fatal(err)
		}
	}

	if err := stopBrowserSessionsWithin(browsers, 45*time.Millisecond); err != nil {
		t.Fatalf("each sequential stop should receive its own timeout: %v", err)
	}
}

func waitForStopAttempt(t *testing.T, attempts <-chan int, want int) {
	t.Helper()
	select {
	case got := <-attempts:
		if got != want {
			t.Fatalf("stop attempt = %d, want %d", got, want)
		}
	case <-time.After(time.Second):
		t.Fatalf("stop attempt %d did not occur", want)
	}
}
