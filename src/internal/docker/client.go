package docker

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const chromaContainerName = "zolam-chromadb"
const chromaImage = "chromadb/chroma:latest"

type DockerClient struct {
	dataDir string
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

	dataDir := os.Getenv("ZOLAM_CHROMADB_DATA_DIR")
	if dataDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		dataDir = filepath.Join(homeDir, ".zolam", "chromadb")
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create chromadb data directory: %w", err)
	}

	return &DockerClient{dataDir: dataDir}, nil
}

func (c *DockerClient) IsContainerRunning(name string) (bool, error) {
	cmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("name=^/%s$", name), "--format", "{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check container status: %w", err)
	}
	return len(output) > 0, nil
}

func (c *DockerClient) containerExists(name string) (bool, error) {
	cmd := exec.Command("docker", "ps", "-a", "--filter", fmt.Sprintf("name=^/%s$", name), "--format", "{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check container status: %w", err)
	}
	return len(output) > 0, nil
}
