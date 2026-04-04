package zolam

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
"github.com/yetanotherchris/zolam/internal/docker"
	"github.com/yetanotherchris/zolam/internal/domain"
)

// IngestOptions holds the flags that control an ingest run.
type IngestOptions struct {
	Extensions     []string
	CollectionName string
	Reset          bool
	Stats          bool
}

// UpdateResult summarises what changed during an update-only ingest.
type UpdateResult struct {
	Added     int
	Changed   int
	Removed   int
	Unchanged int
}

// Ingester orchestrates the full ingest pipeline.
type Ingester struct {
	docker *docker.DockerClient
	config *domain.Config
}

// NewIngester creates a new Ingester backed by the given Docker client and
// application config.
func NewIngester(dc *docker.DockerClient, cfg *domain.Config) *Ingester {
	return &Ingester{
		docker: dc,
		config: cfg,
	}
}

// runAndStream executes the command and streams its combined stdout/stderr
// output line-by-line to outputFn.
type streamableCmd interface {
	StdoutPipe() (io.ReadCloser, error)
	StderrPipe() (io.ReadCloser, error)
	Start() error
	Wait() error
}

func runAndStream(cmd streamableCmd, outputFn func(string)) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("creating stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("creating stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting command: %w", err)
	}

	// Read stdout and stderr concurrently so neither blocks the other.
	// Split on \n or \r so tqdm progress bars (which use \r) are emitted
	// as they update rather than buffered into one giant line.
	var lastLines []string
	var mu sync.Mutex

	scanLines := func(r io.Reader) {
		scanner := bufio.NewScanner(r)
		scanner.Split(splitOnNewlineOrCR)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}
			mu.Lock()
			outputFn(line)
			lastLines = append(lastLines, line)
			if len(lastLines) > 20 {
				lastLines = lastLines[1:]
			}
			mu.Unlock()
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); scanLines(stdout) }()
	go func() { defer wg.Done(); scanLines(stderr) }()
	wg.Wait()

	if err := cmd.Wait(); err != nil {
		mu.Lock()
		detail := strings.Join(lastLines, "\n")
		mu.Unlock()
		if detail != "" {
			return fmt.Errorf("command failed: %w\nOutput:\n%s", err, detail)
		}
		return fmt.Errorf("command failed: %w", err)
	}

	return nil
}

// splitOnNewlineOrCR is a bufio.SplitFunc that splits on \n, \r\n, or \r.
// This allows tqdm-style progress bars (which use \r) to be streamed.
func splitOnNewlineOrCR(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	for i, b := range data {
		if b == '\n' {
			return i + 1, data[:i], nil
		}
		if b == '\r' {
			// \r\n counts as one line ending
			if i+1 < len(data) && data[i+1] == '\n' {
				return i + 2, data[:i], nil
			}
			return i + 1, data[:i], nil
		}
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

// Run executes a full ingest for the given directories.
func (i *Ingester) Run(directories []string, opts IngestOptions, outputFn func(string)) error {
	// Convert relative paths to absolute and build volume mount args.
	var volumeArgs []string
	var containerDirs []string

	for _, dir := range directories {
		absPath, err := filepath.Abs(dir)
		if err != nil {
			return fmt.Errorf("resolving path %s: %w", dir, err)
		}

		base := filepath.Base(absPath)
		containerDir := "/sources/" + base

		volumeArgs = append(volumeArgs, "-v", absPath+":"+containerDir)
		containerDirs = append(containerDirs, containerDir)
	}

	// Build docker run args (volumes, env vars) and container args separately.
	var runArgs []string
	runArgs = append(runArgs, volumeArgs...)

	if opts.CollectionName != "" {
		runArgs = append(runArgs, "-e", "COLLECTION_NAME="+opts.CollectionName)
	}

	var containerArgs []string
	containerArgs = append(containerArgs, "--directory")
	containerArgs = append(containerArgs, containerDirs...)

	if len(opts.Extensions) > 0 {
		containerArgs = append(containerArgs, "--extensions")
		containerArgs = append(containerArgs, opts.Extensions...)
	}

	if opts.Reset {
		containerArgs = append(containerArgs, "--reset")
	}

	if opts.Stats {
		containerArgs = append(containerArgs, "--stats")
	}

	cmd, err := i.docker.ComposeRun("ingest", runArgs, containerArgs)
	if err != nil {
		return fmt.Errorf("creating ingest command: %w", err)
	}

	return runAndStream(cmd, outputFn)
}

// manifestPath returns the path to the hash manifest file.
func (i *Ingester) manifestPath() string {
	return filepath.Join(i.config.DataDir, ".zolam-hashes.json")
}

// loadManifest reads the hash manifest from disk. If the file does not exist
// an empty map is returned.
func (i *Ingester) loadManifest() (map[string]string, error) {
	data, err := os.ReadFile(i.manifestPath())
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, fmt.Errorf("reading manifest: %w", err)
	}

	var manifest map[string]string
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}
	return manifest, nil
}

// saveManifest persists the hash manifest to disk.
func (i *Ingester) saveManifest(manifest map[string]string) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling manifest: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(i.manifestPath()), 0o755); err != nil {
		return fmt.Errorf("creating manifest directory: %w", err)
	}

	if err := os.WriteFile(i.manifestPath(), data, 0o644); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}
	return nil
}

// RunUpdateOnly performs a differential ingest: only files that have been added
// or changed since the last run are ingested, and removed files are tracked.
func (i *Ingester) RunUpdateOnly(directories []string, outputFn func(string)) (*UpdateResult, error) {
	oldManifest, err := i.loadManifest()
	if err != nil {
		return nil, err
	}

	// Compute current hashes across all directories.
	newManifest := make(map[string]string)
	for _, dir := range directories {
		absPath, err := filepath.Abs(dir)
		if err != nil {
			return nil, fmt.Errorf("resolving path %s: %w", dir, err)
		}

		hashes, err := HashDirectory(absPath, i.config.Extensions)
		if err != nil {
			return nil, fmt.Errorf("hashing directory %s: %w", absPath, err)
		}

		for k, v := range hashes {
			newManifest[k] = v
		}
	}

	// Diff the manifests.
	var added, changed, removed, unchanged int
	var filesToIngest []string

	for path, newHash := range newManifest {
		oldHash, exists := oldManifest[path]
		if !exists {
			added++
			filesToIngest = append(filesToIngest, path)
		} else if oldHash != newHash {
			changed++
			filesToIngest = append(filesToIngest, path)
		} else {
			unchanged++
		}
	}

	for path := range oldManifest {
		if _, exists := newManifest[path]; !exists {
			removed++
		}
	}

	result := &UpdateResult{
		Added:     added,
		Changed:   changed,
		Removed:   removed,
		Unchanged: unchanged,
	}

	// Run ingest only on the files that changed or were added.
	if len(filesToIngest) > 0 {
		outputFn(fmt.Sprintf("Ingesting %d file(s): %d added, %d changed (%d unchanged, %d removed)",
			len(filesToIngest), added, changed, unchanged, removed))

		opts := IngestOptions{
			Extensions:     i.config.Extensions,
			CollectionName: i.config.CollectionName,
		}

		// Determine the unique parent directories of files to ingest.
		dirSet := make(map[string]bool)
		for _, f := range filesToIngest {
			dirSet[filepath.Dir(f)] = true
		}
		var dirs []string
		for d := range dirSet {
			dirs = append(dirs, d)
		}

		if err := i.Run(dirs, opts, outputFn); err != nil {
			return nil, fmt.Errorf("running ingest: %w", err)
		}
	} else {
		outputFn("No changes detected, nothing to ingest.")
	}

	// Persist the updated manifest.
	if err := i.saveManifest(newManifest); err != nil {
		return nil, err
	}

	return result, nil
}
