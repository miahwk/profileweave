package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
)

const maxLogFileBytes int64 = 5 << 20

type rotatingLogWriter struct {
	mu   sync.Mutex
	path string
	file *os.File
	size int64
}

func configureLogFile(path string) (func(), error) {
	if path == "" {
		return func() {}, nil
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve log file: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(absolute), 0o700); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}
	writer, err := newRotatingLogWriter(absolute)
	if err != nil {
		return nil, err
	}
	previous := log.Writer()
	log.SetOutput(writer)
	return func() {
		log.SetOutput(previous)
		_ = writer.Close()
	}, nil
}

func newRotatingLogWriter(path string) (*rotatingLogWriter, error) {
	writer := &rotatingLogWriter{path: path}
	if err := writer.open(); err != nil {
		return nil, err
	}
	return writer, nil
}

func (w *rotatingLogWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return 0, errors.New("log writer is unavailable")
	}
	if w.size > 0 && w.size+int64(len(data)) > maxLogFileBytes {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}
	written, err := w.file.Write(data)
	w.size += int64(written)
	return written, err
}

func (w *rotatingLogWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}

func (w *rotatingLogWriter) rotate() error {
	backup := w.path + ".1"
	// Remove the old backup before closing the active file. A transient lock on
	// the backup must not leave the writer without a usable file handle.
	if err := os.Remove(backup); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove old log backup: %w", err)
	}
	if err := w.file.Close(); err != nil {
		w.file = nil
		reopenErr := w.open()
		return errors.Join(fmt.Errorf("close log for rotation: %w", err), reopenErr)
	}
	w.file = nil
	if err := os.Rename(w.path, backup); err != nil {
		reopenErr := w.open()
		return errors.Join(fmt.Errorf("rotate log file: %w", err), reopenErr)
	}
	if err := w.open(); err != nil {
		openErr := err
		restoreErr := os.Rename(backup, w.path)
		reopenErr := w.open()
		return errors.Join(openErr, restoreErr, reopenErr)
	}
	return nil
}

func (w *rotatingLogWriter) open() error {
	file, err := os.OpenFile(w.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return fmt.Errorf("inspect log file: %w", err)
	}
	w.file = file
	w.size = info.Size()
	return nil
}

var _ io.Writer = (*rotatingLogWriter)(nil)
