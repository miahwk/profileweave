package domain

import "testing"

func TestEvaluateValidNativeProfile(t *testing.T) {
	report := Evaluate(Default(), Proxy{Mode: "direct"})
	if report.HasErrors() {
		t.Fatalf("expected no blocking issues, got %#v", report.Issues)
	}
	if report.Score < 90 {
		t.Fatalf("expected high score, got %d", report.Score)
	}
}

func TestEvaluateDetectsBlockingAndCoherenceIssues(t *testing.T) {
	fp := Default()
	fp.Timezone = "Mars/Olympus"
	fp.Screen = Screen{Width: 100, Height: 50, DPR: 9}
	fp.UAMode = "custom"
	fp.UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X) AppleWebKit/537.36 Chrome/125.0 Safari/537.36"
	fp.OS = "windows"
	report := Evaluate(fp, Proxy{Mode: "http", Host: "bad/path", Port: 0})
	wanted := map[string]bool{
		"timezone_invalid": false, "screen_invalid": false, "dpr_invalid": false,
		"proxy_host_invalid": false, "proxy_port_invalid": false, "ua_os_conflict": false,
	}
	for _, issue := range report.Issues {
		if _, ok := wanted[issue.Code]; ok {
			wanted[issue.Code] = true
		}
	}
	for code, found := range wanted {
		if !found {
			t.Errorf("expected issue %s in %#v", code, report.Issues)
		}
	}
	if !report.HasErrors() || report.Score >= 50 {
		t.Fatalf("expected blocking low-score report, got %#v", report)
	}
}

func TestEvaluateWarnsWhenProxyUsesNativeWebRTC(t *testing.T) {
	fp := Default()
	report := Evaluate(fp, Proxy{Mode: "socks5", Host: "127.0.0.1", Port: 1080})
	for _, issue := range report.Issues {
		if issue.Code == "webrtc_proxy_leak" && issue.Severity == SeverityWarning {
			return
		}
	}
	t.Fatal("expected WebRTC proxy warning")
}
