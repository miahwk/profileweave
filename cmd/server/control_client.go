package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/miahwk/profileweave/internal/platform/desktop"
	"github.com/miahwk/profileweave/internal/platform/instancelock"
)

const controlTokenHeader = "X-ProfileWeave-Token"

var errUnexpectedIdentity = errors.New("unexpected local application identity")

func newLocalHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 2 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return errors.New("local control endpoint must not redirect")
		},
	}
}

func verifyProfileWeave(ctx context.Context, baseURL string) error {
	client := newLocalHTTPClient()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	var lastErr error
	for {
		lastErr = verifyProfileWeaveWithClient(ctx, client, baseURL)
		if lastErr == nil || errors.Is(lastErr, errUnexpectedIdentity) {
			return lastErr
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for local application health: %w: %v", ctx.Err(), lastErr)
		case <-ticker.C:
		}
	}
}

func shutdownExisting(ctx context.Context, baseURL, dataDir string) error {
	client := newLocalHTTPClient()
	if err := verifyProfileWeaveWithClient(ctx, client, baseURL); err != nil {
		return err
	}
	var bootstrap struct {
		ControlToken string `json:"controlToken"`
	}
	if err := getLocalJSON(ctx, client, baseURL+"/api/v1/bootstrap", &bootstrap); err != nil {
		return fmt.Errorf("get local control token: %w", err)
	}
	if len(bootstrap.ControlToken) < 32 || len(bootstrap.ControlToken) > 256 {
		return errors.New("local control endpoint returned an invalid token")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/shutdown", nil)
	if err != nil {
		return err
	}
	req.Header.Set(controlTokenHeader, bootstrap.ControlToken)
	response, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request local shutdown: %w", err)
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
	_ = response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		return fmt.Errorf("local shutdown returned HTTP %d", response.StatusCode)
	}
	if err := waitForHTTPExit(ctx, client, baseURL); err != nil {
		return err
	}
	return waitForLockRelease(ctx, dataDir)
}

func verifyProfileWeaveWithClient(ctx context.Context, client *http.Client, baseURL string) error {
	var health struct {
		Product string `json:"product"`
		Status  string `json:"status"`
	}
	if err := getLocalJSON(ctx, client, baseURL+"/api/v1/health", &health); err != nil {
		return fmt.Errorf("verify local application: %w", err)
	}
	if health.Product != desktop.Product || health.Status != "ok" {
		return fmt.Errorf("%w: product=%q status=%q", errUnexpectedIdentity, health.Product, health.Status)
	}
	return nil
}

func getLocalJSON(ctx context.Context, client *http.Client, endpoint string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("local endpoint returned HTTP %d", response.StatusCode)
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, 4097))
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode local endpoint: %w", err)
	}
	return nil
}

func waitForHTTPExit(ctx context.Context, client *http.Client, baseURL string) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("wait for local API exit: %w", err)
		}
		if err := verifyProfileWeaveWithClient(ctx, client, baseURL); err != nil {
			if ctx.Err() != nil {
				return fmt.Errorf("wait for local API exit: %w", ctx.Err())
			}
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for local API exit: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

func waitForLockRelease(ctx context.Context, dataDir string) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		dataLock, err := instancelock.Acquire(dataDir)
		if err == nil {
			return dataLock.Close()
		}
		if !errors.Is(err, instancelock.ErrDataDirInUse) || strings.TrimSpace(dataDir) == "" {
			return fmt.Errorf("verify application exit: %w", err)
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for application exit: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}
