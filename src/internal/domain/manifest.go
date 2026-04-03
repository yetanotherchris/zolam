package domain

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"sort"
	"time"
)

type HashManifest struct {
	Files   map[string]string `json:"files"`   // filepath -> sha256 hash
	Updated time.Time         `json:"updated"`
}

// LoadManifest loads a HashManifest from a JSON file at the given path.
// If the file does not exist, an empty manifest is returned.
func LoadManifest(path string) (*HashManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &HashManifest{
				Files: make(map[string]string),
			}, nil
		}
		return nil, err
	}

	var m HashManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if m.Files == nil {
		m.Files = make(map[string]string)
	}
	return &m, nil
}

// SaveManifest writes the HashManifest to a JSON file at the given path.
// The Updated field is set to the current time before saving.
func SaveManifest(manifest *HashManifest, path string) error {
	manifest.Updated = time.Now()

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// DiffManifest compares an old and new manifest and returns slices of file
// paths that were added, changed (different hash), or removed.
// All returned slices are sorted.
func DiffManifest(old, new *HashManifest) (added, changed, removed []string) {
	for path, newHash := range new.Files {
		oldHash, exists := old.Files[path]
		if !exists {
			added = append(added, path)
		} else if oldHash != newHash {
			changed = append(changed, path)
		}
	}

	for path := range old.Files {
		if _, exists := new.Files[path]; !exists {
			removed = append(removed, path)
		}
	}

	sort.Strings(added)
	sort.Strings(changed)
	sort.Strings(removed)

	return added, changed, removed
}
