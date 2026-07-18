//go:build windows

package zolam

import (
	"os/exec"
	"strconv"
	"strings"
)

// isProcessAlive reports whether pid is a running process, used to detect
// stale lock files left behind by a crashed run.
func isProcessAlive(pid int) bool {
	out, err := exec.Command("tasklist", "/FI", "PID eq "+strconv.Itoa(pid), "/NH").Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), strconv.Itoa(pid))
}
