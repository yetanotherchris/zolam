package docker

import (
	"fmt"
	"os/exec"
)

func (c *DockerClient) RcloneCopy(source, dest, configDir, configPass string) (*exec.Cmd, error) {
	args := []string{"run", "--rm",
		"-v", fmt.Sprintf("%s:/data", dest),
		"-v", fmt.Sprintf("%s:/config/rclone", configDir),
	}

	if configPass != "" {
		args = append(args, "-e", "RCLONE_CONFIG_PASS="+configPass)
	}

	args = append(args, "rclone/rclone", "copy", source, "/data", "--progress")

	cmd := exec.Command("docker", args...)
	return cmd, nil
}
