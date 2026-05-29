package docker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const chromaBaseURL = "http://localhost:8000"
const chromaTenant = "default_tenant"
const chromaDatabase = "default_database"

type Collection struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type FileHashRecord struct {
	Source string
	File   string
	Hash   string
}

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
		resp, err := client.Get("http://localhost:8000/api/v2/heartbeat")
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

func (c *DockerClient) ListCollections() ([]Collection, error) {
	url := fmt.Sprintf("%s/api/v2/tenants/%s/databases/%s/collections", chromaBaseURL, chromaTenant, chromaDatabase)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("chromadb returned status %d", resp.StatusCode)
	}
	var cols []Collection
	if err := json.NewDecoder(resp.Body).Decode(&cols); err != nil {
		return nil, err
	}
	return cols, nil
}

func (c *DockerClient) RemoveCollection(name string) error {
	cols, err := c.ListCollections()
	if err != nil {
		return err
	}
	var id string
	for _, col := range cols {
		if col.Name == name {
			id = col.ID
			break
		}
	}
	if id == "" {
		return fmt.Errorf("collection %q not found", name)
	}
	url := fmt.Sprintf("%s/api/v2/tenants/%s/databases/%s/collections/%s", chromaBaseURL, chromaTenant, chromaDatabase, id)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("chromadb returned status %d", resp.StatusCode)
	}
	return nil
}

// GetFileHashes queries the collection for one chunk per file (chunk index 0)
// and returns the stored file_hash metadata for each file.
func (c *DockerClient) GetFileHashes(collectionName string) ([]FileHashRecord, error) {
	cols, err := c.ListCollections()
	if err != nil {
		return nil, err
	}
	var collectionID string
	for _, col := range cols {
		if col.Name == collectionName {
			collectionID = col.ID
			break
		}
	}
	if collectionID == "" {
		return nil, nil
	}

	url := fmt.Sprintf("%s/api/v2/tenants/%s/databases/%s/collections/%s/get",
		chromaBaseURL, chromaTenant, chromaDatabase, collectionID)

	reqBody, err := json.Marshal(map[string]any{
		"where":   map[string]any{"chunk": map[string]any{"$eq": 0}},
		"include": []string{"metadatas"},
		"limit":   100000,
	})
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("chromadb returned status %d", resp.StatusCode)
	}

	var result struct {
		Metadatas []map[string]any `json:"metadatas"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	records := make([]FileHashRecord, 0, len(result.Metadatas))
	for _, meta := range result.Metadatas {
		source, _ := meta["source"].(string)
		file, _ := meta["file"].(string)
		hash, _ := meta["file_hash"].(string)
		if source == "" || file == "" || hash == "" {
			continue
		}
		records = append(records, FileHashRecord{Source: source, File: file, Hash: hash})
	}
	return records, nil
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
