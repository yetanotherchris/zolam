// Package domain holds the v3 flat-file project types shared between the
// CLI orchestration code and the embedded Python pipeline.
package domain

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	// CurrentProjectVersion is the project.json schema version written by
	// this build of zolam.
	CurrentProjectVersion = 3

	// DefaultBackend is used for new projects when --backend is not given.
	DefaultBackend = "duckdb"

	// DefaultEmbeddingModel and DefaultEmbeddingDims describe the fastembed
	// model used by the embedded Python script. Changing these requires a
	// major version bump, since existing indexes become incompatible.
	DefaultEmbeddingModel = "BAAI/bge-small-en-v1.5"
	DefaultEmbeddingDims  = 384
)

// ValidBackends lists the backends accepted by --backend.
var ValidBackends = []string{"duckdb", "jsonl", "chroma"}

func IsValidBackend(b string) bool {
	for _, v := range ValidBackends {
		if v == b {
			return true
		}
	}
	return false
}

// Project is the persisted contents of <project-dir>/project.json.
type Project struct {
	Version        int       `json:"version"`
	Backend        string    `json:"backend"`
	EmbeddingModel string    `json:"embedding_model"`
	EmbeddingDims  int       `json:"embedding_dims"`
	SourceDirs     []string  `json:"source_dirs"`
	Extensions     []string  `json:"extensions"`
	Created        time.Time `json:"created"`
	LastIngest     time.Time `json:"last_ingest"`
}

// DataDir returns the root zolam data directory, honouring ZOLAM_DATA_DIR.
// It holds only the embedded Python script cache and the legacy
// ChromaDB/Docker state; v3 project data lives in the project's own
// directory (see ProjectJSONPath), not here.
func DataDir() (string, error) {
	if d := os.Getenv("ZOLAM_DATA_DIR"); d != "" {
		return d, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding home directory: %w", err)
	}
	return filepath.Join(home, ".zolam"), nil
}

// LocalProjectDir returns the hidden .zolam/ subdirectory of root (normally
// the current working directory) where a v3 project's files live, replacing
// the old global ~/.zolam/<name> registry: a project is just "the directory
// with a .zolam/ folder in it".
func LocalProjectDir(root string) string {
	return filepath.Join(root, ".zolam")
}

// ProjectJSONPath returns the path to a project's project.json.
func ProjectJSONPath(projectDir string) string {
	return filepath.Join(projectDir, "project.json")
}

// Exists reports whether a project.json exists for the given project dir.
func Exists(projectDir string) bool {
	_, err := os.Stat(ProjectJSONPath(projectDir))
	return err == nil
}

// Load reads and parses project.json from a project directory.
func Load(projectDir string) (*Project, error) {
	data, err := os.ReadFile(ProjectJSONPath(projectDir))
	if err != nil {
		return nil, err
	}
	var p Project
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", ProjectJSONPath(projectDir), err)
	}
	return &p, nil
}

// Save writes project.json atomically to a project directory.
func Save(projectDir string, p *Project) error {
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		return fmt.Errorf("creating project directory: %w", err)
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("serialising project.json: %w", err)
	}
	path := ProjectJSONPath(projectDir)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("writing project.json: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("finalising project.json: %w", err)
	}
	return nil
}

// New creates a fresh Project with today's defaults for the given
// backend/dirs/extensions.
func New(backend string, sourceDirs, extensions []string) *Project {
	now := time.Now().UTC()
	return &Project{
		Version:        CurrentProjectVersion,
		Backend:        backend,
		EmbeddingModel: DefaultEmbeddingModel,
		EmbeddingDims:  DefaultEmbeddingDims,
		SourceDirs:     sourceDirs,
		Extensions:     extensions,
		Created:        now,
		LastIngest:     now,
	}
}

// Remove deletes a project's entire .zolam/ directory (project.json,
// hashes, index, sidecars). Safe to RemoveAll: projectDir is always a
// dedicated .zolam/ folder that zolam exclusively owns, never the user's
// working directory itself.
func Remove(projectDir string) error {
	return os.RemoveAll(projectDir)
}
