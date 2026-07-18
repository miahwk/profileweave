package desktop

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
)

const Product = "ProfileWeave"

// ManagementURL returns the fixed loopback management URL for a validated port.
func ManagementURL(port string) (string, error) {
	value, err := strconv.Atoi(port)
	if err != nil || value < 1 || value > 65535 {
		return "", errors.New("management port must be between 1 and 65535")
	}
	return fmt.Sprintf("http://127.0.0.1:%d", value), nil
}

// Open validates that rawURL is a plain HTTP loopback management URL before
// passing it to the operating system. Platform implementations never use a shell.
func Open(rawURL string) error {
	if err := validateManagementURL(rawURL); err != nil {
		return err
	}
	return openURL(rawURL)
}

func validateManagementURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "http" || parsed.Hostname() != "127.0.0.1" {
		return errors.New("management URL must use HTTP on 127.0.0.1")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path != "" {
		return errors.New("management URL contains unsupported components")
	}
	port, err := strconv.Atoi(parsed.Port())
	if err != nil || port < 1 || port > 65535 {
		return errors.New("management URL has an invalid port")
	}
	if host, _, err := net.SplitHostPort(parsed.Host); err != nil || host != "127.0.0.1" {
		return errors.New("management URL has an invalid host")
	}
	return nil
}
