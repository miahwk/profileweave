//go:build !windows

package infrastructure

import "os"

func replaceFile(from, to string) error { return os.Rename(from, to) }
