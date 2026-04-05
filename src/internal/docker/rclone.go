package docker

import (
	"fmt"
	"os"
	"os/exec"
)

func (c *DockerClient) RcloneCopy(source, dest, configDir, configPass string) (*exec.Cmd, error) {
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return nil, fmt.Errorf("create downloads dir: %w", err)
	}

	args := []string{"run", "--rm", "-i",
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
