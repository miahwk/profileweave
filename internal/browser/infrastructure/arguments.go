package infrastructure

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"strconv"

	"github.com/miahwk/profileweave/internal/browser/domain"
	profiledomain "github.com/miahwk/profileweave/internal/profile/domain"
)

func BuildArguments(dataRoot string, spec domain.LaunchSpec) (string, []string, error) {
	if !profiledomain.ValidID(spec.ProfileID) {
		return "", nil, errors.New("invalid profile ID")
	}
	root, err := filepath.Abs(dataRoot)
	if err != nil {
		return "", nil, errors.New("invalid browser data root")
	}
	userData := filepath.Join(root, spec.ProfileID)
	if filepath.Dir(userData) != filepath.Clean(root) {
		return "", nil, errors.New("profile data directory escaped its root")
	}
	if spec.Locale == "" || spec.Width < 320 || spec.Height < 240 || spec.DPR < 0.5 || spec.DPR > 4 {
		return "", nil, errors.New("invalid launch display configuration")
	}
	args := []string{
		"--user-data-dir=" + userData,
		"--no-first-run", "--no-default-browser-check",
		"--lang=" + spec.Locale,
		fmt.Sprintf("--window-size=%d,%d", spec.Width, spec.Height),
		"--force-device-scale-factor=" + strconv.FormatFloat(spec.DPR, 'f', -1, 64),
	}
	if spec.ProxyMode != "direct" {
		if spec.ProxyMode != "http" && spec.ProxyMode != "socks5" {
			return "", nil, errors.New("invalid proxy mode")
		}
		if spec.ProxyHost == "" || spec.ProxyPort < 1 || spec.ProxyPort > 65535 {
			return "", nil, errors.New("invalid proxy endpoint")
		}
		hostPort := net.JoinHostPort(spec.ProxyHost, strconv.Itoa(spec.ProxyPort))
		proxyURL := (&url.URL{Scheme: spec.ProxyMode, Host: hostPort}).String()
		args = append(args, "--proxy-server="+proxyURL)
	}
	if spec.WebRTCPolicy == "proxy_only" {
		args = append(args, "--force-webrtc-ip-handling-policy=disable_non_proxied_udp")
	}
	if spec.UAMode == "custom" {
		if spec.UserAgent == "" {
			return "", nil, errors.New("custom UA is empty")
		}
		args = append(args, "--user-agent="+spec.UserAgent)
	}
	if spec.StartURL == "" {
		args = append(args, "about:blank")
	} else {
		parsed, err := url.Parse(spec.StartURL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return "", nil, errors.New("invalid start URL")
		}
		args = append(args, spec.StartURL)
	}
	return userData, args, nil
}
