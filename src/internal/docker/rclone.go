package docker

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func (c *DockerClient) RcloneSync(remote, source, dest string) (*exec.Cmd, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	rcloneConfigDir := filepath.Join(homeDir, ".config", "rclone")
	remotePath := fmt.Sprintf("%s:%s", remote, source)

	cmd := exec.Command("docker", "run", "--rm",
		"-v", fmt.Sprintf("%s:/data", dest),
		"-v", fmt.Sprintf("%s:/config/rclone", rcloneConfigDir),
		"rclone/rclone",
		"copy", remotePath, "/data", "--progress",
	)

	return cmd, nil
}
