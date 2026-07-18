//go:build windows

package zolam

import (
	"os/exec"
	"strconv"
	"strings"
)

// setProcessGroup is a no-op on Windows: killProcessTree below handles
// cleanup directly via taskkill regardless of console process-group
// membership (uv may spawn python.exe in its own group).
func setProcessGroup(cmd *exec.Cmd) {}

func killProcessTree(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
}

// isProcessAlive reports whether pid is a running process, used to detect
// stale lock files left behind by a crashed run.
func isProcessAlive(pid int) bool {
	out, err := exec.Command("tasklist", "/FI", "PID eq "+strconv.Itoa(pid), "/NH").Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), strconv.Itoa(pid))
}
