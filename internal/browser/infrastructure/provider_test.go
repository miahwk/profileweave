package infrastructure

import (
	"testing"

	"github.com/miahwk/profileweave/internal/browser/domain"
)

func TestProcessRuntimeProviderInfoIsHonest(t *testing.T) {
	var provider domain.Provider = &ProcessRuntime{}
	info := provider.Info()
	if info.ID != "system-chromium" || info.VersionManagement != "host-managed" {
		t.Fatalf("unexpected provider identity: %#v", info)
	}

	statuses := make(map[string]domain.CapabilityStatus)
	for _, item := range info.Capabilities {
		statuses[item.ID] = item.Status
		if item.Detail == "" {
			t.Fatalf("capability %q has no honest detail", item.ID)
		}
	}
	if statuses["profile-isolation"] != domain.CapabilityApplied {
		t.Fatalf("profile isolation status = %q", statuses["profile-isolation"])
	}
	for _, id := range []string{"os", "languages", "timezone", "hardware", "graphics"} {
		if statuses[id] != domain.CapabilityUnsupported {
			t.Fatalf("capability %q status = %q, want unsupported", id, statuses[id])
		}
	}
	for _, id := range []string{"locale", "screen", "user-agent", "proxy", "webrtc"} {
		if statuses[id] != domain.CapabilityPartial {
			t.Fatalf("capability %q status = %q, want partial", id, statuses[id])
		}
	}
}
