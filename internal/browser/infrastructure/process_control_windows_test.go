//go:build windows

package infrastructure

import (
	"os/exec"
	"reflect"
	"syscall"
	"testing"
	"unsafe"
)

var (
	testKernel32    = syscall.NewLazyDLL("kernel32.dll")
	testOpenProcess = testKernel32.NewProc("OpenProcess")
	testGetExitCode = testKernel32.NewProc("GetExitCodeProcess")
	testCloseHandle = testKernel32.NewProc("CloseHandle")
)

func TestTaskkillArgumentsTargetTreeAndEscalateExplicitly(t *testing.T) {
	if got, want := taskkillArguments(4321, false), []string{"/PID", "4321", "/T"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("graceful args %v, want %v", got, want)
	}
	if got, want := taskkillArguments(4321, true), []string{"/PID", "4321", "/T", "/F"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("forced args %v, want %v", got, want)
	}
}

func TestConfigureProcessCreatesIndependentWindowsGroup(t *testing.T) {
	cmd := exec.Command("unused.exe")
	configureProcess(cmd)
	if cmd.SysProcAttr.CreationFlags&createNewProcessGroup == 0 {
		t.Fatalf("creation flags %#x do not contain CREATE_NEW_PROCESS_GROUP", cmd.SysProcAttr.CreationFlags)
	}
}

func testProcessAlive(pid int) bool {
	const processQueryLimitedInformation = 0x1000
	const stillActive = 259
	handle, _, _ := testOpenProcess.Call(processQueryLimitedInformation, 0, uintptr(pid))
	if handle == 0 {
		return false
	}
	defer testCloseHandle.Call(handle)
	var exitCode uint32
	result, _, _ := testGetExitCode.Call(handle, uintptr(unsafe.Pointer(&exitCode)))
	return result != 0 && exitCode == stillActive
}
