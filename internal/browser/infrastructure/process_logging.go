package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
)

func logProcessFailure(operation, profileID string, err error) {
	log.Printf("browser runtime operation=%s profile=%s error=%s", operation, profileID, safeLocalProcessError(err))
}

func safeLocalProcessError(err error) string {
	if err == nil {
		return "none"
	}
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		switch {
		case errors.Is(pathErr.Err, os.ErrPermission):
			return pathErr.Op + ": permission denied"
		case errors.Is(pathErr.Err, os.ErrNotExist):
			return pathErr.Op + ": file not found"
		default:
			return pathErr.Op + ": operating system error"
		}
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return fmt.Sprintf("process exited with code %d", exitErr.ExitCode())
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "deadline exceeded"
	}
	if errors.Is(err, context.Canceled) {
		return "canceled"
	}
	return fmt.Sprintf("%T", err)
}
