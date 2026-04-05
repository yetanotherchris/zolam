package docker

import (
	"embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

//go:embed compose.yml
var composeFS embed.FS

type DockerClient struct {
	composePath string
}

// CheckDockerAvailable verifies that Docker is installed and the daemon is running.
func CheckDockerAvailable() error {
	// Check if docker CLI is on PATH
	_, err := exec.LookPath("docker")
	if err != nil {
		return fmt.Errorf("Docker is required but not found. Please install Docker Desktop or Docker Engine.")
	}

	// Check if Docker daemon is running
	cmd := exec.Command("docker", "info")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Docker daemon is not running. Please start Docker and try again.")
	}

	return nil
}

func NewDockerClient() (*DockerClient, error) {
	if err := CheckDockerAvailable(); err != nil {
		return nil, err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	zolamDir := filepath.Join(homeDir, ".zolam")
	if err := os.MkdirAll(zolamDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create ~/.zolam directory: %w", err)
	}

	composePath := filepath.Join(zolamDir, "compose.yml")
	data, err := composeFS.ReadFile("compose.yml")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded compose.yml: %w", err)
	}

	existing, readErr := os.ReadFile(composePath)
	if readErr != nil || string(existing) != string(data) {
		if readErr == nil {
			backupPath := composePath + ".bak"
			os.Rename(composePath, backupPath)
		}
		if err := os.WriteFile(composePath, data, 0644); err != nil {
			return nil, fmt.Errorf("failed to write compose.yml: %w", err)
		}
	}

	return &DockerClient{
		composePath: composePath,
	}, nil
}

func (c *DockerClient) composeCmd(args ...string) *exec.Cmd {
	cmdArgs := []string{"compose", "-f", c.composePath}
	cmdArgs = append(cmdArgs, args...)
	return exec.Command("docker", cmdArgs...)
}

func (c *DockerClient) ComposeUp(services ...string) error {
	args := []string{"up", "-d", "--pull", "always"}
	args = append(args, services...)
	cmd := c.composeCmd(args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (c *DockerClient) ComposeDown() error {
	cmd := c.composeCmd("down")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (c *DockerClient) ComposeRun(service string, runArgs []string, containerArgs []string) (*exec.Cmd, error) {
	cmdArgs := []string{"run", "--rm"}
	cmdArgs = append(cmdArgs, runArgs...)
	cmdArgs = append(cmdArgs, service)
	cmdArgs = append(cmdArgs, containerArgs...)
	cmd := c.composeCmd(cmdArgs...)
	return cmd, nil
}

func (c *DockerClient) RunContainer(image string, args ...string) (*exec.Cmd, error) {
	cmdArgs := []string{"run", "--rm"}
	cmdArgs = append(cmdArgs, args...)
	cmdArgs = append(cmdArgs, image)
	cmd := exec.Command("docker", cmdArgs...)
	return cmd, nil
}

func (c *DockerClient) IsContainerRunning(name string) (bool, error) {
	cmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("name=%s", name), "--format", "{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check container status: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if strings.Contains(strings.TrimSpace(line), name) {
			return true, nil
		}
	}
	return false, nil
}

func (c *DockerClient) StreamOutput(cmd *exec.Cmd, w io.Writer) error {
	cmd.Stdout = w
	cmd.Stderr = w
	return cmd.Run()
}
