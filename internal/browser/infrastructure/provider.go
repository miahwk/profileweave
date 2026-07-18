package infrastructure

import "github.com/miahwk/profileweave/internal/browser/domain"

var _ domain.Provider = (*ProcessRuntime)(nil)

// Info identifies ProcessRuntime as the host-installed Chromium provider. The
// application isolates profiles and applies supported launch flags; it does not
// patch the browser engine or claim anti-detection guarantees.
func (*ProcessRuntime) Info() domain.ProviderInfo {
	return domain.ProviderInfo{
		ID: "system-chromium", Name: "System Chromium",
		Description: "Launches a host-installed Chromium-family browser with isolated profile data and supported command-line settings.",
		Source:      "host-installed browser", License: "browser-specific; not distributed by ProfileWeave",
		VersionManagement: "host-managed",
		Capabilities: []domain.ProviderCapability{
			capability("profile-isolation", "Profile isolation", domain.CapabilityApplied, "Uses a separate browser user-data directory for every profile."),
			capability("locale", "Locale", domain.CapabilityPartial, "Applies Chromium --lang; navigator.languages remains browser-managed."),
			capability("screen", "Screen", domain.CapabilityPartial, "Applies initial window size and device scale factor, not physical display identity."),
			capability("user-agent", "User agent", domain.CapabilityPartial, "Can set the legacy User-Agent string; Client Hints remain browser-managed."),
			capability("proxy", "Proxy", domain.CapabilityPartial, "Supports unauthenticated HTTP and SOCKS5 proxy launch settings."),
			capability("webrtc", "WebRTC policy", domain.CapabilityPartial, "Applies supported Chromium WebRTC policy flags."),
			capability("os", "Operating system", domain.CapabilityUnsupported, "Does not change the operating system exposed by the browser engine."),
			capability("languages", "Language list", domain.CapabilityUnsupported, "Does not override navigator.languages."),
			capability("timezone", "Timezone", domain.CapabilityUnsupported, "Does not override the browser timezone."),
			capability("hardware", "CPU and memory", domain.CapabilityUnsupported, "Does not override hardwareConcurrency or deviceMemory."),
			capability("graphics", "Canvas, WebGL and GPU", domain.CapabilityUnsupported, "Does not patch graphics or canvas output."),
			capability("audio-fonts", "Audio and fonts", domain.CapabilityUnsupported, "Does not patch AudioContext or installed font enumeration."),
		},
	}
}

func capability(id, name string, status domain.CapabilityStatus, detail string) domain.ProviderCapability {
	return domain.ProviderCapability{ID: id, Name: name, Status: status, Detail: detail}
}
