package ingester

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// ComputeHash returns the hex-encoded SHA-256 hash of the file at the given path.
func ComputeHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// HashDirectory walks dir and computes SHA-256 hashes for every file whose
// extension matches one of the provided extensions. It returns a map from
// file path to hex-encoded hash. Files are hashed concurrently using a worker
// pool sized to runtime.NumCPU().
func HashDirectory(dir string, extensions []string) (map[string]string, error) {
	extSet := make(map[string]bool, len(extensions))
	for _, ext := range extensions {
		e := ext
		if !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		extSet[strings.ToLower(e)] = true
	}

	// Collect matching file paths.
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if extSet[strings.ToLower(filepath.Ext(path))] {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Hash files concurrently.
	type result struct {
		path string
		hash string
		err  error
	}

	workers := runtime.NumCPU()
	if workers < 1 {
		workers = 1
	}

	jobs := make(chan string, len(files))
	results := make(chan result, len(files))

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				h, err := ComputeHash(path)
				results <- result{path: path, hash: h, err: err}
			}
		}()
	}

	for _, f := range files {
		jobs <- f
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	hashes := make(map[string]string, len(files))
	for r := range results {
		if r.err != nil {
			return nil, r.err
		}
		hashes[r.path] = r.hash
	}

	return hashes, nil
}
