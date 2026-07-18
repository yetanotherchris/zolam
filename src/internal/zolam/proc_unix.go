//go:build !windows

package zolam

import "syscall"

// isProcessAlive reports whether pid is a running process, used to detect
// stale lock files left behind by a crashed run. Signal 0 performs no
// action but still returns ESRCH if the process doesn't exist; EPERM means
// it exists but we can't signal it, which still counts as alive.
func isProcessAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}
