package infrastructure

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/miahwk/profileweave/internal/browser/domain"
	profiledomain "github.com/miahwk/profileweave/internal/profile/domain"
)

var ErrProcessNotOwned = errors.New("browser process is not owned by this service")

const (
	processGracePeriod = 3 * time.Second
	processForcePeriod = 5 * time.Second
	processExitSettle  = 500 * time.Millisecond
)

type managedProcess struct {
	cmd      *exec.Cmd
	finished chan struct{}
}

type ProcessRuntime struct {
	mu        sync.Mutex
	dataMu    sync.Mutex
	resolver  Resolver
	dataRoot  string
	trashRoot string
	active    map[string]*managedProcess
}

func NewProcessRuntime(dataDir string) (*ProcessRuntime, error) {
	root, err := filepath.Abs(filepath.Join(dataDir, "browser-data"))
	if err != nil {
		return nil, fmt.Errorf("resolve browser data directory: %w", err)
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, fmt.Errorf("create browser data directory: %w", err)
	}
	trashRoot := filepath.Join(filepath.Dir(root), "trash", "browser-data")
	if err := os.MkdirAll(trashRoot, 0o700); err != nil {
		return nil, fmt.Errorf("create browser data trash directory: %w", err)
	}
	return &ProcessRuntime{dataRoot: root, trashRoot: trashRoot, active: make(map[string]*managedProcess)}, nil
}

func (r *ProcessRuntime) Discover(ctx context.Context) ([]domain.BrowserDescriptor, error) {
	return r.resolver.Discover(ctx)
}

func (r *ProcessRuntime) ValidateExecutable(path string) error {
	return r.resolver.ValidateExecutable(path)
}

func (r *ProcessRuntime) EnsureProfileData(profileID string) (bool, error) {
	profileDir, err := r.profileDataPath(profileID)
	if err != nil {
		return false, err
	}
	r.dataMu.Lock()
	defer r.dataMu.Unlock()
	info, err := os.Lstat(profileDir)
	if err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return false, errors.New("profile browser data path is not a directory")
		}
		return false, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return false, errors.New("inspect profile browser directory")
	}
	if err := os.Mkdir(profileDir, 0o700); err != nil {
		return false, errors.New("create profile browser directory")
	}
	return true, nil
}

func (r *ProcessRuntime) TrashProfileData(profileID string) (string, error) {
	profileDir, err := r.profileDataPath(profileID)
	if err != nil {
		return "", err
	}
	r.dataMu.Lock()
	defer r.dataMu.Unlock()
	info, err := os.Lstat(profileDir)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("inspect profile browser directory")
	}
	token, err := newTrashToken(profileID)
	if err != nil {
		return "", err
	}
	if err := os.Rename(profileDir, filepath.Join(r.trashRoot, token)); err != nil {
		return "", fmt.Errorf("move profile browser data to trash: %w", err)
	}
	return token, nil
}

func (r *ProcessRuntime) RestoreProfileData(profileID, restoreToken string) error {
	profileDir, err := r.profileDataPath(profileID)
	if err != nil {
		return err
	}
	trashed, err := r.trashedDataPath(profileID, restoreToken)
	if err != nil {
		return err
	}
	r.dataMu.Lock()
	defer r.dataMu.Unlock()
	if _, err := os.Lstat(profileDir); !errors.Is(err, os.ErrNotExist) {
		return errors.New("profile browser directory already exists")
	}
	info, err := os.Lstat(trashed)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("trashed profile browser path is not a directory")
	}
	if err := os.Rename(trashed, profileDir); err != nil {
		return fmt.Errorf("restore profile browser data: %w", err)
	}
	return nil
}

func (r *ProcessRuntime) Launch(ctx context.Context, spec domain.LaunchSpec) (domain.Process, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.active[spec.ProfileID]; exists {
		return domain.Process{}, errors.New("profile process is already active")
	}
	executable, err := r.resolver.Resolve(ctx, spec.BrowserKind, spec.CustomPath)
	if err != nil {
		return domain.Process{}, err
	}
	_, args, err := BuildArguments(r.dataRoot, spec)
	if err != nil {
		return domain.Process{}, err
	}
	if _, err := r.EnsureProfileData(spec.ProfileID); err != nil {
		return domain.Process{}, err
	}
	// Arguments are supplied as an argv slice. No shell parses profile-controlled values.
	cmd := exec.Command(executable, args...)
	cmd.Stdout, cmd.Stderr, cmd.Stdin = nil, nil, nil
	configureProcess(cmd)
	if err := cmd.Start(); err != nil {
		logProcessFailure("start", spec.ProfileID, err)
		return domain.Process{}, errors.New("start browser process")
	}
	done := make(chan error, 1)
	managed := &managedProcess{cmd: cmd, finished: make(chan struct{})}
	r.active[spec.ProfileID] = managed
	go r.wait(spec.ProfileID, managed, done)
	return domain.Process{PID: cmd.Process.Pid, Done: done}, nil
}

func (r *ProcessRuntime) profileDataPath(profileID string) (string, error) {
	if !profiledomain.ValidID(profileID) {
		return "", errors.New("invalid profile ID")
	}
	profileDir := filepath.Join(r.dataRoot, profileID)
	if filepath.Dir(profileDir) != filepath.Clean(r.dataRoot) {
		return "", errors.New("profile data directory escaped its root")
	}
	return profileDir, nil
}

func newTrashToken(profileID string) (string, error) {
	raw := make([]byte, 12)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate profile data restore token: %w", err)
	}
	return profileID + "-" + hex.EncodeToString(raw), nil
}

func validRestoreToken(profileID, token string) bool {
	const randomHexLength = 24
	prefix := profileID + "-"
	if len(token) != len(prefix)+randomHexLength || token[:len(prefix)] != prefix {
		return false
	}
	_, err := hex.DecodeString(token[len(prefix):])
	return err == nil
}

func (r *ProcessRuntime) Stop(ctx context.Context, profileID string) error {
	r.mu.Lock()
	managed, ok := r.active[profileID]
	r.mu.Unlock()
	if !ok {
		return ErrProcessNotOwned
	}
	requestCtx, cancel := context.WithTimeout(ctx, processGracePeriod)
	err := requestProcessTreeStop(requestCtx, managed.cmd.Process.Pid)
	cancel()
	if err != nil && !processFinished(managed) {
		logProcessFailure("request-stop", profileID, err)
	}
	timer := time.NewTimer(processGracePeriod)
	defer timer.Stop()
	select {
	case <-managed.finished:
		return nil
	case <-ctx.Done():
		return forceStopProcessTree(profileID, managed)
	case <-timer.C:
		return forceStopProcessTree(profileID, managed)
	}
}

func forceStopProcessTree(profileID string, managed *managedProcess) error {
	if processFinished(managed) {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), processForcePeriod)
	defer cancel()
	if err := forceProcessTreeStop(ctx, managed.cmd.Process.Pid); err != nil && !processFinished(managed) {
		logProcessFailure("force-stop", profileID, err)
		settle := time.NewTimer(processExitSettle)
		defer settle.Stop()
		select {
		case <-managed.finished:
			return nil
		case <-ctx.Done():
			return errors.New("terminate browser process tree")
		case <-settle.C:
			return errors.New("terminate browser process tree")
		}
	}
	select {
	case <-managed.finished:
		return nil
	case <-ctx.Done():
		logProcessFailure("wait-after-force-stop", profileID, ctx.Err())
		return errors.New("wait for terminated browser process tree")
	}
}

func processFinished(managed *managedProcess) bool {
	select {
	case <-managed.finished:
		return true
	default:
		return false
	}
}

func (r *ProcessRuntime) wait(profileID string, managed *managedProcess, done chan<- error) {
	err := managed.cmd.Wait()
	r.mu.Lock()
	if r.active[profileID] == managed {
		delete(r.active, profileID)
	}
	r.mu.Unlock()
	close(managed.finished)
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() != 0 {
		done <- fmt.Errorf("browser exited with code %d", exitErr.ExitCode())
	} else {
		done <- nil
	}
	close(done)
}
