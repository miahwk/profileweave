package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/miahwk/profileweave/internal/browser/domain"
)

type Resolver struct{}

func (Resolver) Discover(_ context.Context) ([]domain.BrowserDescriptor, error) {
	candidates := browserCandidates()
	for i := range candidates {
		if candidates[i].Path == "" {
			continue
		}
		if err := validateDiscoveredExecutable(candidates[i].Path); err == nil {
			candidates[i].Available = true
		} else {
			candidates[i].Path = ""
		}
	}
	return candidates, nil
}

func (r Resolver) Resolve(ctx context.Context, kind string) (string, error) {
	if kind == "custom" || kind == "custom-disabled" {
		return "", errors.New("custom browser executables are disabled; choose a discovered browser")
	}
	browsers, err := r.Discover(ctx)
	if err != nil {
		return "", err
	}
	for _, browser := range browsers {
		if kind == "auto" && browser.Available {
			return browser.Path, nil
		}
		if browser.ID == kind && browser.Available {
			return browser.Path, nil
		}
	}
	return "", fmt.Errorf("selected %s browser is not available", kind)
}

func validateDiscoveredExecutable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return errors.New("browser executable does not exist")
	}
	if !info.Mode().IsRegular() {
		return errors.New("browser path must identify a regular file")
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o111 == 0 {
		return errors.New("browser file is not executable")
	}
	return nil
}

func browserCandidates() []domain.BrowserDescriptor {
	items := []domain.BrowserDescriptor{
		{ID: "chrome", Name: "Google Chrome"}, {ID: "edge", Name: "Microsoft Edge"},
		{ID: "brave", Name: "Brave"}, {ID: "chromium", Name: "Chromium"},
	}
	paths := candidatePaths()
	for i := range items {
		for _, path := range paths[items[i].ID] {
			if path != "" {
				if _, err := os.Stat(path); err == nil {
					items[i].Path = path
					break
				}
			}
		}
	}
	return items
}

func candidatePaths() map[string][]string {
	if runtime.GOOS == "windows" {
		local, program, program86 := os.Getenv("LOCALAPPDATA"), os.Getenv("PROGRAMFILES"), os.Getenv("PROGRAMFILES(X86)")
		return map[string][]string{
			"chrome":   {filepath.Join(program, "Google", "Chrome", "Application", "chrome.exe"), filepath.Join(local, "Google", "Chrome", "Application", "chrome.exe"), filepath.Join(program86, "Google", "Chrome", "Application", "chrome.exe")},
			"edge":     {filepath.Join(program86, "Microsoft", "Edge", "Application", "msedge.exe"), filepath.Join(program, "Microsoft", "Edge", "Application", "msedge.exe")},
			"brave":    {filepath.Join(program, "BraveSoftware", "Brave-Browser", "Application", "brave.exe"), filepath.Join(local, "BraveSoftware", "Brave-Browser", "Application", "brave.exe")},
			"chromium": {filepath.Join(local, "Chromium", "Application", "chrome.exe")},
		}
	}
	return map[string][]string{
		"chrome":   {"/usr/bin/google-chrome", "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"},
		"edge":     {"/usr/bin/microsoft-edge", "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge"},
		"brave":    {"/usr/bin/brave-browser", "/Applications/Brave Browser.app/Contents/MacOS/Brave Browser"},
		"chromium": {"/usr/bin/chromium", "/usr/bin/chromium-browser", "/Applications/Chromium.app/Contents/MacOS/Chromium"},
	}
}
