package zolam

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/yetanotherchris/zolam/internal/domain"
)

// UpdateResult summarises what changed during a sync.
type UpdateResult struct {
	Added     int
	Changed   int
	Removed   int
	Unchanged int
}

// V3SyncOptions controls a flat-file (sqlite/jsonl) ingest run.
type V3SyncOptions struct {
	// Root is the directory whose .zolam/ subdirectory holds the project's
	// files. Empty means the current working directory.
	Root string
	// Dirs sets/overrides the project's source_dirs and is always required,
	// on the first ingest and every re-sync alike (pass "." to index Root
	// itself). Incremental behaviour (only added/changed/removed files are
	// reprocessed) comes entirely from diffing against file-hashes.json, not
	// from omitting Dirs.
	Dirs []string
	// Extensions and Backend only matter when creating a brand new project;
	// they are ignored (a stored mismatch on Backend is an error) once a
	// project.json already exists.
	Extensions []string
	Backend    string
	Reset      bool
}

// ResolveRoot returns dir as an absolute path, defaulting to the current
// working directory when dir is empty.
func ResolveRoot(dir string) (string, error) {
	if dir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("finding working directory: %w", err)
		}
		return wd, nil
	}
	return filepath.Abs(dir)
}

// RunV3Sync creates or loads a flat-file project in Root's .zolam/
// subdirectory, diffs the current file set on disk against
// file-hashes.json, invokes the embedded Python pipeline for
// added/changed/removed files, and regenerates index.md. `zolam ingest`
// calls this for every run (both first-time ingest and incremental
// re-sync).
func RunV3Sync(opts V3SyncOptions, outputFn func(string)) (*UpdateResult, *domain.Project, error) {
	if len(opts.Dirs) == 0 {
		return nil, nil, fmt.Errorf("zolam ingest requires at least one directory, e.g. 'zolam ingest ./notes' (use 'zolam ingest .' to index the current directory itself)")
	}

	root, err := ResolveRoot(opts.Root)
	if err != nil {
		return nil, nil, err
	}
	projectDir := domain.LocalProjectDir(root)

	if opts.Reset {
		if err := domain.Remove(projectDir); err != nil {
			return nil, nil, fmt.Errorf("resetting project: %w", err)
		}
	}

	project, err := loadOrCreateProject(projectDir, root, opts)
	if err != nil {
		return nil, nil, err
	}

	if project.EmbeddingModel != domain.DefaultEmbeddingModel {
		return nil, nil, fmt.Errorf("this project was indexed with embedding model %q, but zolam now defaults to %q; re-run with --reset to re-index", project.EmbeddingModel, domain.DefaultEmbeddingModel)
	}

	oldHashes, err := LoadFileHashes(projectDir)
	if err != nil {
		return nil, nil, err
	}

	newHashes := make(map[string]string)
	for _, dir := range project.SourceDirs {
		hashes, err := HashDirectory(dir, root, project.Extensions)
		if err != nil {
			return nil, nil, fmt.Errorf("hashing directory %s: %w", dir, err)
		}
		for k, v := range hashes {
			newHashes[k] = v
		}
	}

	diff := DiffHashes(oldHashes, newHashes)
	result := &UpdateResult{
		Added:     len(diff.Added),
		Changed:   len(diff.Changed),
		Removed:   len(diff.Removed),
		Unchanged: len(diff.Unchanged),
	}

	filesToProcess := append(append([]string{}, diff.Added...), diff.Changed...)

	if len(filesToProcess) == 0 && len(diff.Removed) == 0 {
		outputFn("No changes detected, nothing to ingest.")
	} else {
		outputFn(fmt.Sprintf("Processing %d file(s): %d added, %d changed, %d removed (%d unchanged)",
			len(filesToProcess), result.Added, result.Changed, result.Removed, result.Unchanged))

		summary, err := RunIngest(project, projectDir, filesToProcess, diff.Removed, outputFn)
		if err != nil {
			return nil, nil, err
		}
		outputFn(fmt.Sprintf("%d chunk(s) written, %d file(s) errored", summary.ChunksWritten, summary.FilesErrored))
		for _, e := range summary.Errors {
			outputFn("  ERROR " + e)
		}
	}

	if err := SaveFileHashes(projectDir, newHashes); err != nil {
		return nil, nil, err
	}

	project.LastIngest = time.Now().UTC()
	if err := domain.Save(projectDir, project); err != nil {
		return nil, nil, err
	}

	if err := GenerateIndexMD(project, filepath.Base(root), projectDir, root, newHashes); err != nil {
		return nil, nil, err
	}

	return result, project, nil
}

// RunV3Update re-syncs an existing project using the source_dirs already
// recorded in its project.json, so callers (the 'zolam ingest update'
// subcommand) don't need to name a directory again. Unlike RunV3Sync, it
// errors if no project exists yet rather than creating one — first-time
// ingest still requires 'zolam ingest <dir>' to establish source_dirs
// explicitly.
func RunV3Update(root string, reset bool, outputFn func(string)) (*UpdateResult, *domain.Project, error) {
	resolvedRoot, err := ResolveRoot(root)
	if err != nil {
		return nil, nil, err
	}
	projectDir := domain.LocalProjectDir(resolvedRoot)
	if !domain.Exists(projectDir) {
		return nil, nil, fmt.Errorf("no zolam project in %s; run 'zolam ingest <dir>' there first", resolvedRoot)
	}
	project, err := domain.Load(projectDir)
	if err != nil {
		return nil, nil, fmt.Errorf("loading project in %s: %w", projectDir, err)
	}
	return RunV3Sync(V3SyncOptions{
		Root:  root,
		Dirs:  project.SourceDirs,
		Reset: reset,
	}, outputFn)
}

func loadOrCreateProject(projectDir, root string, opts V3SyncOptions) (*domain.Project, error) {
	absDirs, err := absPaths(opts.Dirs)
	if err != nil {
		return nil, err
	}

	if !domain.Exists(projectDir) {
		backend := opts.Backend
		if backend == "" {
			backend = domain.DefaultBackend
		}
		if backend != "sqlite" && backend != "jsonl" {
			return nil, fmt.Errorf("unknown backend %q (expected sqlite or jsonl; the legacy chroma backend is managed separately via 'zolam chromadb')", backend)
		}
		extensions := opts.Extensions
		if len(extensions) == 0 {
			extensions = SupportedFileExtensions
		}
		return domain.New(backend, absDirs, extensions), nil
	}

	project, err := domain.Load(projectDir)
	if err != nil {
		return nil, fmt.Errorf("loading project in %s: %w", projectDir, err)
	}
	if opts.Backend != "" && opts.Backend != project.Backend {
		return nil, fmt.Errorf("this project was created with backend %q; use --reset to switch to %q", project.Backend, opts.Backend)
	}
	project.SourceDirs = mergeDirs(project.SourceDirs, absDirs)
	if len(opts.Extensions) > 0 {
		project.Extensions = opts.Extensions
	}
	return project, nil
}

// LoadV3Project loads an existing flat-file project from dir's .zolam/
// subdirectory (dir defaulting to the current working directory),
// returning a clear, actionable error if it doesn't exist or was indexed
// with a now-unsupported embedding model.
func LoadV3Project(dir string) (*domain.Project, string, error) {
	root, err := ResolveRoot(dir)
	if err != nil {
		return nil, "", err
	}
	projectDir := domain.LocalProjectDir(root)
	if !domain.Exists(projectDir) {
		return nil, "", fmt.Errorf("no zolam project in %s; run 'zolam ingest <dir>' there first", root)
	}
	project, err := domain.Load(projectDir)
	if err != nil {
		return nil, "", fmt.Errorf("loading project in %s: %w", projectDir, err)
	}
	if project.EmbeddingModel != domain.DefaultEmbeddingModel {
		return nil, "", fmt.Errorf("this project was indexed with embedding model %q, but zolam now defaults to %q; re-run 'zolam ingest --reset' to re-index", project.EmbeddingModel, domain.DefaultEmbeddingModel)
	}
	return project, projectDir, nil
}

func absPaths(dirs []string) ([]string, error) {
	out := make([]string, 0, len(dirs))
	for _, d := range dirs {
		abs, err := filepath.Abs(d)
		if err != nil {
			return nil, fmt.Errorf("resolving path %s: %w", d, err)
		}
		out = append(out, abs)
	}
	return out, nil
}

// mergeDirs unions newDirs into existing, preserving existing's order and
// appending any newDirs not already present. A named source directory is
// additive across ingest runs — dropping one (and the files under it) is
// only supposed to happen via --reset, not by simply omitting it from a
// later 'zolam ingest <dir>' call.
func mergeDirs(existing, newDirs []string) []string {
	seen := make(map[string]bool, len(existing))
	out := make([]string, len(existing))
	copy(out, existing)
	for _, d := range existing {
		seen[d] = true
	}
	for _, d := range newDirs {
		if !seen[d] {
			seen[d] = true
			out = append(out, d)
		}
	}
	return out
}
