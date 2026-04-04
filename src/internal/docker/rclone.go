package docker

import (
	"fmt"
	"os/exec"
)

func (c *DockerClient) RcloneCopy(source, dest, configDir string) (*exec.Cmd, error) {
	cmd := exec.Command("docker", "run", "--rm",
		"-v", fmt.Sprintf("%s:/data", dest),
		"-v", fmt.Sprintf("%s:/config/rclone", configDir),
		"rclone/rclone",
		"copy", source, "/data", "--progress",
	)

	return cmd, nil
}
