package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	browserapp "github.com/miahwk/profileweave/internal/browser/application"
	browserinfra "github.com/miahwk/profileweave/internal/browser/infrastructure"
	"github.com/miahwk/profileweave/internal/platform/httpapi"
	profileapp "github.com/miahwk/profileweave/internal/profile/application"
	profileinfra "github.com/miahwk/profileweave/internal/profile/infrastructure"
)

func serveApplication(dataDir, managementURL string, shouldOpen bool) error {
	parsed, err := url.Parse(managementURL)
	if err != nil {
		return fmt.Errorf("parse management URL: %w", err)
	}
	listener, err := net.Listen("tcp", parsed.Host)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", parsed.Host, err)
	}
	defer listener.Close()

	repository, err := profileinfra.NewJSONRepository(dataDir)
	if err != nil {
		return err
	}
	runtime, err := browserinfra.NewProcessRuntime(dataDir)
	if err != nil {
		return err
	}
	browsers := browserapp.NewService(repository, runtime)
	profiles := profileapp.NewService(repository, browsers, runtime)
	stopRequests := make(chan struct{}, 1)
	requestStop := func() {
		select {
		case stopRequests <- struct{}{}:
		default:
		}
	}
	server := newHTTPServer(parsed.Host, httpapi.WithWebDir(
		httpapi.NewWithShutdown(profiles, browsers, requestStop), resolveWebDir(),
	))
	serveErrors := make(chan error, 1)
	go func() { serveErrors <- server.Serve(listener) }()
	log.Printf("ProfileWeave API listening on %s", managementURL)

	if shouldOpen {
		if err := openManagementURL(managementURL); err != nil {
			log.Printf("open management console: %v", err)
		}
	}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	return waitForCleanShutdown(browsers, server, stopRequests, serveErrors, signals)
}

func newHTTPServer(address string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr: address, Handler: handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}
}

func waitForCleanShutdown(
	browsers *browserapp.Service,
	server *http.Server,
	stopRequests <-chan struct{},
	serveErrors <-chan error,
	signals <-chan os.Signal,
) error {
	serverStopped, triggerErr := waitForStop(stopRequests, serveErrors, signals)
	browsers.BeginShutdown()
	for {
		stopErr := stopBrowserSessions(browsers)
		if stopErr == nil {
			if serverStopped {
				return triggerErr
			}
			return errors.Join(triggerErr, shutdownHTTP(server))
		}
		log.Printf("managed browser cleanup failed; application remains available for retry: %v", stopErr)
		if serverStopped {
			return errors.Join(triggerErr, stopErr)
		}
		serverStopped, triggerErr = waitForStop(stopRequests, serveErrors, signals)
	}
}

func waitForStop(
	stopRequests <-chan struct{},
	serveErrors <-chan error,
	signals <-chan os.Signal,
) (bool, error) {
	select {
	case <-signals:
		return false, nil
	case <-stopRequests:
		return false, nil
	case err := <-serveErrors:
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			return true, nil
		}
		return true, fmt.Errorf("serve local API: %w", err)
	}
}

func shutdownHTTP(server *http.Server) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("shutdown local API: %w", err)
	}
	return nil
}

func stopBrowserSessions(browsers *browserapp.Service) error {
	return stopBrowserSessionsWithin(browsers, 10*time.Second)
}

func stopBrowserSessionsWithin(browsers *browserapp.Service, timeout time.Duration) error {
	sessions := browsers.List()
	var result error
	for _, session := range sessions {
		if !browsers.IsRunning(session.ProfileID) {
			continue
		}
		// Service serializes lifecycle transitions. Starting each timeout only
		// when its stop begins gives every session the full cleanup window.
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		_, err := browsers.Stop(ctx, session.ProfileID)
		cancel()
		if err != nil {
			result = errors.Join(result, fmt.Errorf("stop browser profile %s: %w", session.ProfileID, err))
		}
	}
	return result
}

func commandContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}
