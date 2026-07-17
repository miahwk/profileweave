package infrastructure

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/miahwk/profileweave/internal/browser/domain"
)

func launchSpec() domain.LaunchSpec {
	return domain.LaunchSpec{
		ProfileID: "p_0123456789abcdef0123456789abcdef", Locale: "en-US", Width: 1920, Height: 1080, DPR: 1.25,
		UAMode: "custom", UserAgent: "Mozilla/5.0 test", ProxyMode: "socks5", ProxyHost: "::1", ProxyPort: 1080,
		WebRTCPolicy: "proxy_only", StartURL: "https://example.test/",
	}
}

func TestBuildArgumentsUsesIsolatedDirectoryAndArgv(t *testing.T) {
	root := t.TempDir()
	userData, args, err := BuildArguments(root, launchSpec())
	if err != nil {
		t.Fatal(err)
	}
	wantDir := filepath.Join(root, launchSpec().ProfileID)
	if userData != wantDir || !contains(args, "--user-data-dir="+wantDir) {
		t.Fatalf("missing isolated user directory in %v", args)
	}
	for _, wanted := range []string{
		"--proxy-server=socks5://[::1]:1080", "--force-webrtc-ip-handling-policy=disable_non_proxied_udp",
		"--user-agent=Mozilla/5.0 test", "--window-size=1920,1080", "https://example.test/",
	} {
		if !contains(args, wanted) {
			t.Errorf("missing argument %q in %v", wanted, args)
		}
	}
}

func TestBuildArgumentsRejectsUnsafeInputs(t *testing.T) {
	tests := []struct {
		name string
		edit func(*domain.LaunchSpec)
	}{
		{"traversal ID", func(s *domain.LaunchSpec) { s.ProfileID = "../escape" }},
		{"javascript URL", func(s *domain.LaunchSpec) { s.StartURL = "javascript:alert(1)" }},
		{"proxy port", func(s *domain.LaunchSpec) { s.ProxyPort = 70000 }},
		{"display", func(s *domain.LaunchSpec) { s.Width = 10 }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			spec := launchSpec()
			tc.edit(&spec)
			if _, _, err := BuildArguments(t.TempDir(), spec); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if strings.Compare(value, wanted) == 0 {
			return true
		}
	}
	return false
}
