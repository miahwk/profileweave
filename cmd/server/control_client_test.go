package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestVerifyProfileWeaveChecksProductIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","product":"AnotherProduct"}`))
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := verifyProfileWeave(ctx, server.URL); err == nil {
		t.Fatal("wrong product identity was accepted")
	}
}

func TestLocalClientRejectsRedirect(t *testing.T) {
	server := httptest.NewServer(http.RedirectHandler("https://example.test", http.StatusFound))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := verifyProfileWeave(ctx, server.URL); err == nil {
		t.Fatal("redirecting endpoint was accepted")
	}
}

func TestVerifyProfileWeaveRetriesWhileOwnerInitializes(t *testing.T) {
	portNumber := freePort(t)
	address := fmt.Sprintf("127.0.0.1:%d", portNumber)
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","product":"ProfileWeave"}`))
	})}
	started := make(chan struct{})
	go func() {
		time.Sleep(200 * time.Millisecond)
		listener, err := net.Listen("tcp", address)
		if err != nil {
			close(started)
			return
		}
		close(started)
		_ = server.Serve(listener)
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := verifyProfileWeave(ctx, "http://"+address); err != nil {
		t.Fatalf("initializing owner was not retried: %v", err)
	}
	<-started
	_ = server.Close()
}

func TestWaitForLockReleaseHonorsContext(t *testing.T) {
	dir := t.TempDir()
	dataLock, err := acquireTestLock(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer dataLock()
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	if err := waitForLockRelease(ctx, dir); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("waitForLockRelease error = %v", err)
	}
}
