//go:build darwin

package desktop

import (
	"fmt"
	"os/exec"
)

func openURL(rawURL string) error {
	if err := exec.Command("open", rawURL).Start(); err != nil {
		return fmt.Errorf("open management URL: %w", err)
	}
	return nil
}
