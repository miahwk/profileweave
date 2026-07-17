package domain

import (
	"regexp"
	"runtime"
	"strings"
	"time"
	_ "time/tzdata"
)

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

type Issue struct {
	Severity Severity `json:"severity"`
	Code     string   `json:"code"`
	Field    string   `json:"field,omitempty"`
	Message  string   `json:"message"`
}

type Report struct {
	Score  int     `json:"score"`
	Issues []Issue `json:"issues"`
}

func (r Report) HasErrors() bool {
	for _, issue := range r.Issues {
		if issue.Severity == SeverityError {
			return true
		}
	}
	return false
}

var localePattern = regexp.MustCompile(`^[A-Za-z]{2,3}(?:-[A-Za-z]{2}|-[0-9]{3})?$`)

func Evaluate(fp Fingerprint, proxy Proxy) Report {
	issues := make([]Issue, 0)
	add := func(level Severity, code, field, message string) {
		issues = append(issues, Issue{Severity: level, Code: code, Field: field, Message: message})
	}

	if fp.OS != "native" && fp.OS != "windows" && fp.OS != "macos" && fp.OS != "linux" {
		add(SeverityError, "os_invalid", "fingerprint.os", "OS target must be native, windows, macos, or linux")
	} else if fp.OS != "native" {
		if fp.OS == hostOS() {
			add(SeverityInfo, "os_diagnostic_only", "fingerprint.os", "OS target matches the host but is retained only as diagnostic metadata")
		} else {
			add(SeverityWarning, "os_not_applied", "fingerprint.os", "OS target is not emulated; the runtime retains the host operating system")
		}
	}
	if fp.UAMode != "native" && fp.UAMode != "custom" {
		add(SeverityError, "ua_mode_invalid", "fingerprint.uaMode", "UA mode must be native or custom")
	}
	if fp.UAMode == "custom" {
		if len(strings.TrimSpace(fp.UserAgent)) < 20 || strings.ContainsAny(fp.UserAgent, "\r\n") {
			add(SeverityError, "ua_invalid", "fingerprint.userAgent", "custom user agent is invalid")
		} else if conflictsWithOS(fp.UserAgent, fp.OS) {
			add(SeverityWarning, "ua_os_conflict", "fingerprint.userAgent", "user agent appears inconsistent with the OS target")
		}
		add(SeverityWarning, "ua_partial", "fingerprint.userAgent", "custom UA does not synchronize Client Hints or TLS")
	}
	if !localePattern.MatchString(fp.Locale) {
		add(SeverityError, "locale_invalid", "fingerprint.locale", "locale must use a language-region form such as en-US")
	}
	if len(fp.Languages) == 0 {
		add(SeverityError, "languages_empty", "fingerprint.languages", "at least one language is required")
	} else {
		if !strings.EqualFold(fp.Locale, fp.Languages[0]) {
			add(SeverityWarning, "locale_language_conflict", "fingerprint.languages", "the first language should match locale")
		}
		add(SeverityInfo, "languages_unsupported", "fingerprint.languages", "language preferences are saved for diagnostics but are not applied by the MVP runtime")
	}
	if _, err := time.LoadLocation(fp.Timezone); err != nil || fp.Timezone == "Local" {
		add(SeverityError, "timezone_invalid", "fingerprint.timezone", "timezone must be a known IANA timezone")
	} else {
		add(SeverityInfo, "timezone_unsupported", "fingerprint.timezone", "timezone is saved for diagnostics but is not applied by the MVP runtime")
	}
	if fp.Screen.Width < 320 || fp.Screen.Width > 7680 || fp.Screen.Height < 240 || fp.Screen.Height > 4320 {
		add(SeverityError, "screen_invalid", "fingerprint.screen", "screen dimensions are outside supported limits")
	} else if fp.Screen.Width < 800 || fp.Screen.Height < 600 {
		add(SeverityWarning, "screen_uncommon", "fingerprint.screen", "screen dimensions are uncommon for a desktop profile")
	}
	if fp.Screen.DPR < 0.5 || fp.Screen.DPR > 4 {
		add(SeverityError, "dpr_invalid", "fingerprint.screen.dpr", "DPR must be between 0.5 and 4")
	}
	if fp.CPUCores < 1 || fp.CPUCores > 128 || fp.MemoryGB < 1 || fp.MemoryGB > 256 {
		add(SeverityError, "hardware_invalid", "fingerprint", "CPU and memory values are outside supported limits")
	} else {
		add(SeverityInfo, "hardware_unsupported", "fingerprint", "CPU and memory are expected values only and are not overridden")
	}
	if fp.WebRTCPolicy != "native" && fp.WebRTCPolicy != "proxy_only" {
		add(SeverityError, "webrtc_invalid", "fingerprint.webrtcPolicy", "WebRTC policy must be native or proxy_only")
	}
	validateProxy(proxy, add)
	if proxy.Mode != "direct" && fp.WebRTCPolicy == "native" {
		add(SeverityWarning, "webrtc_proxy_leak", "fingerprint.webrtcPolicy", "native WebRTC can use traffic paths outside the configured proxy")
	}

	score := 100
	for _, issue := range issues {
		switch issue.Severity {
		case SeverityError:
			score -= 25
		case SeverityWarning:
			score -= 10
		}
	}
	if score < 0 {
		score = 0
	}
	return Report{Score: score, Issues: issues}
}

func hostOS() string {
	if runtime.GOOS == "darwin" {
		return "macos"
	}
	return runtime.GOOS
}

func validateProxy(proxy Proxy, add func(Severity, string, string, string)) {
	if proxy.Mode != "direct" && proxy.Mode != "http" && proxy.Mode != "socks5" {
		add(SeverityError, "proxy_mode_invalid", "proxy.mode", "proxy mode must be direct, http, or socks5")
		return
	}
	if proxy.Mode == "direct" {
		if proxy.Host != "" || proxy.Port != 0 {
			add(SeverityWarning, "proxy_direct_values", "proxy", "direct mode ignores the proxy endpoint")
		}
		return
	}
	if proxy.Host == "" || strings.ContainsAny(proxy.Host, "/\\:@?#\r\n \t") {
		add(SeverityError, "proxy_host_invalid", "proxy.host", "proxy host is invalid")
	}
	if proxy.Port < 1 || proxy.Port > 65535 {
		add(SeverityError, "proxy_port_invalid", "proxy.port", "proxy port must be between 1 and 65535")
	}
}

func conflictsWithOS(ua, target string) bool {
	if target == "native" {
		target = runtime.GOOS
	}
	ua = strings.ToLower(ua)
	markers := map[string]string{"windows": "windows", "macos": "macintosh", "linux": "linux"}
	marker, ok := markers[target]
	if !ok {
		return false
	}
	for osName, osMarker := range markers {
		if strings.Contains(ua, osMarker) {
			return osName != target
		}
	}
	return !strings.Contains(ua, marker)
}
