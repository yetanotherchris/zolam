package docker

import (
	"fmt"
	"net/http"
	"time"
)

func (c *DockerClient) StartChromaDB() error {
	return c.ComposeUp("chromadb")
}

func (c *DockerClient) StopChromaDB() error {
	return c.ComposeDown()
}

func (c *DockerClient) ChromaDBStatus() (bool, error) {
	return c.IsContainerRunning("chromadb")
}

func (c *DockerClient) WaitForChromaDB(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}

	for time.Now().Before(deadline) {
		resp, err := client.Get("http://localhost:8000/api/v1/heartbeat")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("chromadb did not become ready within %s", timeout)
}

func (c *DockerClient) EnsureChromaDB(timeout time.Duration) error {
	running, err := c.ChromaDBStatus()
	if err != nil {
		return fmt.Errorf("failed to check chromadb status: %w", err)
	}

	if !running {
		if err := c.StartChromaDB(); err != nil {
			return fmt.Errorf("failed to start chromadb: %w", err)
		}
	}

	return c.WaitForChromaDB(timeout)
}
