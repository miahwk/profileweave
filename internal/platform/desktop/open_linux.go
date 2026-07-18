//go:build linux

package desktop

import (
	"fmt"
	"os/exec"
)

func openURL(rawURL string) error {
	if err := exec.Command("xdg-open", rawURL).Start(); err != nil {
		return fmt.Errorf("open management URL: %w", err)
	}
	return nil
}
