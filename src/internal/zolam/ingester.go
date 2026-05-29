package zolam

import (
	"bufio"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"

	"github.com/yetanotherchris/zolam/internal/docker"
)

// IngestOptions holds the flags that control a Docker ingest run.
type IngestOptions struct {
	Extensions     []string
	CollectionName string
	Reset          bool
	// Files, when non-nil, restricts ingestion to these specific host absolute
	// paths. Each path must fall under one of the mounted directories.
	Files []string
}

// UpdateResult summarises what changed during a sync.
type UpdateResult struct {
	Added     int
	Changed   int
	Removed   int
	Unchanged int
}

// Ingester orchestrates the full ingest pipeline.
type Ingester struct {
	docker *docker.DockerClient
}

func NewIngester(dc *docker.DockerClient) *Ingester {
	return &Ingester{docker: dc}
}

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

func splitOnNewlineOrCR(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	for i, b := range data {
		if b == '\n' {
			return i + 1, data[:i], nil
		}
		if b == '\r' {
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

type dirMount struct {
	absPath      string
	containerDir string
}

// Run calls Docker to ingest files. When opts.Files is set, only those specific
// files are passed to the container via --files; otherwise the full directory is
// processed.
func (i *Ingester) Run(directories []string, opts IngestOptions, outputFn func(string)) error {
	var mounts []dirMount
	var volumeArgs, containerDirs []string

	for _, dir := range directories {
		absPath, err := filepath.Abs(dir)
		if err != nil {
			return fmt.Errorf("resolving path %s: %w", dir, err)
		}
		base := filepath.Base(absPath)
		containerDir := "/sources/" + base
		mounts = append(mounts, dirMount{absPath, containerDir})
		volumeArgs = append(volumeArgs, "-v", filepath.ToSlash(absPath)+":"+containerDir)
		containerDirs = append(containerDirs, containerDir)
	}

	runArgs := append(volumeArgs, "-e", "COLLECTION_NAME="+opts.CollectionName)

	var containerArgs []string
	if len(containerDirs) > 0 {
		containerArgs = append(containerArgs, "--directory")
		containerArgs = append(containerArgs, containerDirs...)
	}
	if len(opts.Extensions) > 0 {
		containerArgs = append(containerArgs, "--extensions")
		containerArgs = append(containerArgs, opts.Extensions...)
	}
	if opts.Reset {
		containerArgs = append(containerArgs, "--reset")
	}

	if len(opts.Files) > 0 {
		var containerFiles []string
		sep := string(filepath.Separator)
		for _, hostFile := range opts.Files {
			for _, m := range mounts {
				if strings.HasPrefix(hostFile, m.absPath+sep) {
					rel := hostFile[len(m.absPath):]
					containerFiles = append(containerFiles, m.containerDir+filepath.ToSlash(rel))
					break
				}
			}
		}
		if len(containerFiles) > 0 {
			containerArgs = append(containerArgs, "--files")
			containerArgs = append(containerArgs, containerFiles...)
		}
	}

	if err := i.docker.ComposePull("ingest"); err != nil {
		return fmt.Errorf("pulling ingest image: %w", err)
	}

	cmd, err := i.docker.ComposeRun("ingest", runArgs, containerArgs)
	if err != nil {
		return fmt.Errorf("creating ingest command: %w", err)
	}

	return runAndStream(cmd, outputFn)
}

// RunSync performs a hash-aware sync: hashes files on disk, compares against
// the local hash store (~/.zolam/hashes.db), and passes only new or changed
// files to Docker. Both zolam ingest and zolam update call this.
//
// extensions controls which file types are considered; nil uses SupportedFileExtensions.
// When reset is true the hash store and ChromaDB collection are both cleared before ingesting.
func (i *Ingester) RunSync(directories []string, collection string, extensions []string, reset bool, outputFn func(string)) (*UpdateResult, error) {
	store, err := OpenHashStore(collection)
	if err != nil {
		return nil, fmt.Errorf("opening hash store: %w", err)
	}
	defer store.Close()

	if reset {
		if err := store.DeleteCollection(); err != nil {
			return nil, fmt.Errorf("clearing hash store: %w", err)
		}
	}

	absDirectories := make([]string, 0, len(directories))
	for _, dir := range directories {
		absPath, err := filepath.Abs(dir)
		if err != nil {
			return nil, fmt.Errorf("resolving path %s: %w", dir, err)
		}
		absDirectories = append(absDirectories, absPath)
	}

	exts := extensions
	if len(exts) == 0 {
		exts = SupportedFileExtensions
	}

	oldHashes, err := store.GetAll()
	if err != nil {
		return nil, fmt.Errorf("reading hash store: %w", err)
	}

	newHashes := make(map[string]string)
	for _, absPath := range absDirectories {
		hashes, err := HashDirectory(absPath, exts)
		if err != nil {
			return nil, fmt.Errorf("hashing directory %s: %w", absPath, err)
		}
		for k, v := range hashes {
			newHashes[k] = v
		}
	}

	var added, changed, removed, unchanged int
	var filesToIngest []string

	if reset {
		for path := range newHashes {
			filesToIngest = append(filesToIngest, path)
		}
		added = len(filesToIngest)
	} else {
		for path, newHash := range newHashes {
			oldHash, exists := oldHashes[path]
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
		for path := range oldHashes {
			if _, exists := newHashes[path]; !exists {
				removed++
			}
		}
	}

	result := &UpdateResult{Added: added, Changed: changed, Removed: removed, Unchanged: unchanged}

	if len(filesToIngest) > 0 {
		if !reset {
			outputFn(fmt.Sprintf("Ingesting %d file(s): %d added, %d changed (%d unchanged, %d removed)",
				len(filesToIngest), added, changed, unchanged, removed))
		}

		opts := IngestOptions{
			Extensions:     exts,
			CollectionName: collection,
			Reset:          reset,
			Files:          filesToIngest,
		}
		// On reset, Docker handles the collection wipe; let it process the whole
		// directory rather than a --files list (simpler, and all files are "new").
		if reset {
			opts.Files = nil
		}

		if err := i.Run(absDirectories, opts, outputFn); err != nil {
			return nil, err
		}

		// Record hashes for every file we passed to Docker. This marks
		// unprocessable files (e.g. scanned PDFs with no text) as "seen" so
		// they are not retried until their content actually changes.
		for _, f := range filesToIngest {
			if hash, ok := newHashes[f]; ok {
				if err := store.Set(f, hash); err != nil {
					return nil, fmt.Errorf("updating hash store: %w", err)
				}
			}
		}
	} else {
		outputFn("No changes detected, nothing to ingest.")
	}

	return result, nil
}
