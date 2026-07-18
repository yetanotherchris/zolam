//go:build !windows

package zolam

import (
	"os/exec"
	"syscall"
)

// setProcessGroup puts the child in its own process group so we can kill
// the whole uv/python subtree at once, instead of only the direct child.
func setProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

func killProcessTree(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	if pgid, err := syscall.Getpgid(cmd.Process.Pid); err == nil {
		syscall.Kill(-pgid, syscall.SIGKILL)
		return
	}
	cmd.Process.Kill()
}

// isProcessAlive reports whether pid is a running process, used to detect
// stale lock files left behind by a crashed run. Signal 0 performs no
// action but still returns ESRCH if the process doesn't exist; EPERM means
// it exists but we can't signal it, which still counts as alive.
func isProcessAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}
