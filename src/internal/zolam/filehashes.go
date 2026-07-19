package zolam

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// fileHashesName is the flat JSON file recording path -> sha256 for
// a project's incremental-update state, stored directly in the
// project's own directory.
const fileHashesName = "file-hashes.json"

// LoadFileHashes reads <projectDir>/file-hashes.json. A missing file is not
// an error; it returns an empty map (a project's first ingest).
func LoadFileHashes(projectDir string) (map[string]string, error) {
	data, err := os.ReadFile(filepath.Join(projectDir, fileHashesName))
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("reading file-hashes.json: %w", err)
	}
	hashes := make(map[string]string)
	if len(data) > 0 {
		if err := json.Unmarshal(data, &hashes); err != nil {
			return nil, fmt.Errorf("parsing file-hashes.json: %w", err)
		}
	}
	return hashes, nil
}

// SaveFileHashes writes the path -> sha256 map atomically.
func SaveFileHashes(projectDir string, hashes map[string]string) error {
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		return fmt.Errorf("creating project directory: %w", err)
	}
	data, err := json.MarshalIndent(hashes, "", "  ")
	if err != nil {
		return fmt.Errorf("serialising file-hashes.json: %w", err)
	}
	path := filepath.Join(projectDir, fileHashesName)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("writing file-hashes.json: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("finalising file-hashes.json: %w", err)
	}
	return nil
}

// DiffResult is the outcome of comparing an old and new path->hash map.
type DiffResult struct {
	Added     []string
	Changed   []string
	Removed   []string
	Unchanged []string
}

// DiffHashes compares old and new file hash maps and classifies every path.
func DiffHashes(oldHashes, newHashes map[string]string) DiffResult {
	var r DiffResult
	for path, newHash := range newHashes {
		oldHash, exists := oldHashes[path]
		switch {
		case !exists:
			r.Added = append(r.Added, path)
		case oldHash != newHash:
			r.Changed = append(r.Changed, path)
		default:
			r.Unchanged = append(r.Unchanged, path)
		}
	}
	for path := range oldHashes {
		if _, exists := newHashes[path]; !exists {
			r.Removed = append(r.Removed, path)
		}
	}
	return r
}
