//go:build windows || unix

package infrastructure

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"
)

const processTreeHelperMode = "PROFILEWEAVE_PROCESS_TREE_HELPER"

func TestForceProcessTreeStopsParentAndChild(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=^TestProcessTreeHelper$")
	cmd.Env = append(os.Environ(), processTreeHelperMode+"=parent")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	configureProcess(cmd)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill() }()

	childPID := readHelperPID(t, stdout)
	defer stopTestHelper(childPID)
	managed := &managedProcess{cmd: cmd, finished: make(chan struct{})}
	wait := make(chan error, 1)
	go func() {
		wait <- cmd.Wait()
		close(managed.finished)
	}()
	if err := forceStopProcessTree("test-helper", managed); err != nil {
		t.Skipf("process-tree termination is unavailable in this restricted environment: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	select {
	case <-wait:
	case <-ctx.Done():
		t.Fatal("helper parent did not stop")
	}
	deadline := time.Now().Add(3 * time.Second)
	for testProcessAlive(childPID) && time.Now().Before(deadline) {
		time.Sleep(25 * time.Millisecond)
	}
	if testProcessAlive(childPID) {
		t.Fatalf("helper child %d survived process-tree stop", childPID)
	}
}

func stopTestHelper(pid int) {
	process, err := os.FindProcess(pid)
	if err == nil {
		_ = process.Kill()
	}
	deadline := time.Now().Add(2 * time.Second)
	for testProcessAlive(pid) && time.Now().Before(deadline) {
		time.Sleep(25 * time.Millisecond)
	}
}

func readHelperPID(t *testing.T, stdout interface{ Read([]byte) (int, error) }) int {
	t.Helper()
	result := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		if scanner.Scan() {
			result <- scanner.Text()
			return
		}
		result <- ""
	}()
	select {
	case raw := <-result:
		pid, err := strconv.Atoi(raw)
		if err != nil || pid < 1 {
			t.Fatalf("invalid helper child PID %q", raw)
		}
		return pid
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for helper child PID")
		return 0
	}
}

func TestProcessTreeHelper(t *testing.T) {
	switch os.Getenv(processTreeHelperMode) {
	case "parent":
		child := exec.Command(os.Args[0], "-test.run=^TestProcessTreeHelper$")
		child.Env = append(os.Environ(), processTreeHelperMode+"=child")
		if err := child.Start(); err != nil {
			os.Exit(2)
		}
		fmt.Println(child.Process.Pid)
		for {
			time.Sleep(time.Hour)
		}
	case "child":
		for {
			time.Sleep(time.Hour)
		}
	}
}
