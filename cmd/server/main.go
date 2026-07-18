package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/miahwk/profileweave/internal/buildinfo"
	"github.com/miahwk/profileweave/internal/platform/desktop"
	"github.com/miahwk/profileweave/internal/platform/instancelock"
)

type commandOptions struct {
	showVersion bool
	open        bool
	shutdown    bool
	logFile     string
}

var (
	openManagementURL   = desktop.Open
	userConfigDirectory = os.UserConfigDir
)

func main() {
	if err := runCommand(os.Args[1:], os.Stderr); err != nil {
		os.Exit(1)
	}
}

func runCommand(args []string, flagOutput io.Writer) (result error) {
	options, err := parseOptions(args, flagOutput)
	if err != nil {
		log.Printf("ProfileWeave: %v", err)
		return err
	}
	if options.showVersion {
		info := buildinfo.Current()
		fmt.Printf("version=%s commit=%s date=%s\n", info.Version, info.Commit, info.Date)
		return nil
	}
	dataDir, err := resolveDataDir()
	if err != nil {
		return err
	}
	closeLog, err := configureLogFile(options.logFile)
	if err != nil {
		log.Printf("ProfileWeave: %v", err)
		return err
	}
	defer func() {
		if result != nil {
			log.Printf("ProfileWeave: %v", result)
		}
		closeLog()
	}()

	managementURL, err := desktop.ManagementURL(port())
	if err != nil {
		return err
	}
	if options.shutdown {
		ctx, cancel := commandContext(20 * time.Second)
		defer cancel()
		return runShutdown(ctx, managementURL, dataDir)
	}

	dataLock, err := instancelock.Acquire(dataDir)
	if errors.Is(err, instancelock.ErrDataDirInUse) && options.open {
		ctx, cancel := commandContext(5 * time.Second)
		defer cancel()
		if err := verifyProfileWeave(ctx, managementURL); err != nil {
			return fmt.Errorf("existing data owner could not be verified: %w", err)
		}
		return openManagementURL(managementURL)
	}
	if err != nil {
		return err
	}
	defer func() {
		if err := dataLock.Close(); err != nil {
			log.Printf("release data directory lock: %v", err)
		}
	}()
	return serveApplication(dataDir, managementURL, options.open)
}

func runShutdown(ctx context.Context, managementURL, dataDir string) error {
	info, err := os.Stat(dataDir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect profile data directory: %w", err)
	}
	if !info.IsDir() {
		return errors.New("profile data path is not a directory")
	}
	dataLock, err := instancelock.Acquire(dataDir)
	if err == nil {
		return dataLock.Close()
	}
	if !errors.Is(err, instancelock.ErrDataDirInUse) {
		return err
	}
	return shutdownExisting(ctx, managementURL, dataDir)
}

func parseOptions(args []string, output io.Writer) (commandOptions, error) {
	var options commandOptions
	flags := flag.NewFlagSet("profileweave", flag.ContinueOnError)
	flags.SetOutput(output)
	flags.BoolVar(&options.showVersion, "version", false, "print version information and exit")
	flags.BoolVar(&options.open, "open", false, "open the local management console")
	flags.BoolVar(&options.shutdown, "shutdown", false, "request the running application to exit")
	flags.StringVar(&options.logFile, "log-file", "", "append logs to a rotating local file")
	if err := flags.Parse(args); err != nil {
		return commandOptions{}, err
	}
	if flags.NArg() != 0 {
		return commandOptions{}, fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	if options.shutdown && options.open {
		return commandOptions{}, errors.New("--open and --shutdown cannot be used together")
	}
	return options, nil
}

func resolveDataDir() (string, error) {
	if configured := firstEnv("PROFILEWEAVE_DATA_DIR", "FINGERPRINT_BROWSER_DATA_DIR"); configured != "" {
		return filepath.Abs(configured)
	}
	config, err := userConfigDirectory()
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
