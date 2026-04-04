package docker

import (
	"fmt"
	"os/exec"
)

func (c *DockerClient) RcloneSync(remote, source, dest, configDir string) (*exec.Cmd, error) {
	remotePath := fmt.Sprintf("%s:%s", remote, source)

	cmd := exec.Command("docker", "run", "--rm",
		"-v", fmt.Sprintf("%s:/data", dest),
		"-v", fmt.Sprintf("%s:/config/rclone", configDir),
		"rclone/rclone",
		"copy", remotePath, "/data", "--progress",
	)

	return cmd, nil
}
