//go:build windows

package desktop

import (
	"fmt"

	"golang.org/x/sys/windows"
)

func openURL(rawURL string) error {
	target, err := windows.UTF16PtrFromString(rawURL)
	if err != nil {
		return fmt.Errorf("encode management URL: %w", err)
	}
	if err := windows.ShellExecute(0, nil, target, nil, nil, windows.SW_SHOWNORMAL); err != nil {
		return fmt.Errorf("open management URL: %w", err)
	}
	return nil
}
