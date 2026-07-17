//go:build windows

package zolam

import (
	"os/exec"
	"strconv"
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
