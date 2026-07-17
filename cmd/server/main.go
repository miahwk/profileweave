package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	browserapp "github.com/miahwk/profileweave/internal/browser/application"
	browserinfra "github.com/miahwk/profileweave/internal/browser/infrastructure"
	"github.com/miahwk/profileweave/internal/buildinfo"
	"github.com/miahwk/profileweave/internal/platform/httpapi"
	"github.com/miahwk/profileweave/internal/platform/instancelock"
	profileapp "github.com/miahwk/profileweave/internal/profile/application"
	profileinfra "github.com/miahwk/profileweave/internal/profile/infrastructure"
)

func main() {
	showVersion := flag.Bool("version", false, "print version information and exit")
	flag.Parse()
	if *showVersion {
		info := buildinfo.Current()
		fmt.Printf("version=%s commit=%s date=%s\n", info.Version, info.Commit, info.Date)
		return
	}
	dataDir, err := resolveDataDir()
	if err != nil {
		log.Fatal(err)
	}
	dataLock, err := instancelock.Acquire(dataDir)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := dataLock.Close(); err != nil {
			log.Printf("release data directory lock: %v", err)
		}
	}()
	repository, err := profileinfra.NewJSONRepository(dataDir)
	if err != nil {
		log.Fatal(err)
	}
	runtime, err := browserinfra.NewProcessRuntime(dataDir)
	if err != nil {
		log.Fatal(err)
	}
	browsers := browserapp.NewService(repository, runtime)
	profiles := profileapp.NewService(repository, browsers, runtime, runtime)

	server := &http.Server{
		Addr:              "127.0.0.1:" + port(),
		Handler:           httpapi.WithWebDir(httpapi.New(profiles, browsers), resolveWebDir()),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}
	go func() {
		log.Printf("ProfileWeave API listening on http://%s", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	<-signals
	serverCtx, cancelServer := context.WithTimeout(context.Background(), 5*time.Second)
	_ = server.Shutdown(serverCtx)
	cancelServer()

	var stops sync.WaitGroup
	for _, session := range browsers.List() {
		if browsers.IsRunning(session.ProfileID) {
			stops.Add(1)
			go func(profileID string) {
				defer stops.Done()
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				if _, err := browsers.Stop(ctx, profileID); err != nil {
					log.Printf("stop browser profile=%s: %v", profileID, err)
				}
			}(session.ProfileID)
		}
	}
	stops.Wait()
}

func resolveDataDir() (string, error) {
	if configured := firstEnv("PROFILEWEAVE_DATA_DIR", "FINGERPRINT_BROWSER_DATA_DIR"); configured != "" {
		return filepath.Abs(configured)
	}
	config, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	current := filepath.Join(config, "ProfileWeave")
	legacy := filepath.Join(config, "FingerprintBrowser")
	if _, err := os.Stat(current); errors.Is(err, os.ErrNotExist) {
		if info, legacyErr := os.Stat(legacy); legacyErr == nil && info.IsDir() {
			return legacy, nil
		}
	}
	return current, nil
}

func port() string {
	configured := firstEnv("PROFILEWEAVE_PORT", "FINGERPRINT_BROWSER_PORT")
	value, err := strconv.Atoi(configured)
	if configured == "" || err != nil || value < 1 || value > 65535 {
		return "3210"
	}
	return configured
}

func resolveWebDir() string {
	if configured := firstEnv("PROFILEWEAVE_WEB_DIR", "FINGERPRINT_BROWSER_WEB_DIR"); configured != "" {
		return configured
	}
	if executable, err := os.Executable(); err == nil {
		packaged := filepath.Join(filepath.Dir(executable), "frontend", "dist")
		if info, statErr := os.Stat(filepath.Join(packaged, "index.html")); statErr == nil && info.Mode().IsRegular() {
			return packaged
		}
	}
	return filepath.Join("frontend", "dist")
}

func firstEnv(names ...string) string {
	for _, name := range names {
		if value := os.Getenv(name); value != "" {
			return value
		}
	}
	return ""
}
