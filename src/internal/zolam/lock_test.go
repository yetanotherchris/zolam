package zolam

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestAcquireLock_SecondCallFailsWhileHeld(t *testing.T) {
	dir := t.TempDir()

	release, err := acquireLock(dir)
	if err != nil {
		t.Fatalf("first acquireLock() returned error: %v", err)
	}
	defer release()

	if _, err := acquireLock(dir); err == nil {
		t.Fatal("expected second acquireLock() to fail while the first is held")
	} else if !strings.Contains(err.Error(), "already running") {
		t.Errorf("expected an actionable 'already running' error, got: %v", err)
	}
}

func TestAcquireLock_ReleaseAllowsReacquire(t *testing.T) {
	dir := t.TempDir()

	release, err := acquireLock(dir)
	if err != nil {
		t.Fatalf("acquireLock() returned error: %v", err)
	}
	release()

	release2, err := acquireLock(dir)
	if err != nil {
		t.Fatalf("acquireLock() after release returned error: %v", err)
	}
	release2()

	if _, err := os.Stat(filepath.Join(dir, lockFileName)); !os.IsNotExist(err) {
		t.Errorf("expected lock file to be removed after release, stat err = %v", err)
	}
}

func TestAcquireLock_ReclaimsStaleLock(t *testing.T) {
	dir := t.TempDir()

	// A pid that is (almost certainly) not running: spawn and wait for exit.
	cmd := exec.Command("true")
	if err := cmd.Run(); err != nil {
		t.Fatalf("running throwaway process: %v", err)
	}
	deadPID := cmd.Process.Pid

	lockPath := filepath.Join(dir, lockFileName)
	if err := os.WriteFile(lockPath, []byte(strconv.Itoa(deadPID)), 0o644); err != nil {
		t.Fatalf("writing stale lock file: %v", err)
	}

	release, err := acquireLock(dir)
	if err != nil {
		t.Fatalf("expected stale lock to be reclaimed, got error: %v", err)
	}
	release()
}
