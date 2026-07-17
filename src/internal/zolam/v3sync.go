package zolam

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/yetanotherchris/zolam/internal/domain"
)

// V3SyncOptions controls a flat-file (duckdb/jsonl) ingest or update run.
type V3SyncOptions struct {
	ProjectName string
	// Dirs, when non-empty, sets/overrides the project's source_dirs. When
	// empty on an existing project, the stored source_dirs are used.
	Dirs []string
	// Extensions and Backend only matter when creating a brand new project;
	// they are ignored (a stored mismatch on Backend is an error) once a
	// project.json already exists.
	Extensions []string
	Backend    string
	Reset      bool
}

// RunV3Sync creates or loads a flat-file project, diffs the current file
// set on disk against file-hashes.json, invokes the embedded Python
// pipeline for added/changed/removed files, and regenerates index.md.
// Both `zolam ingest` and `zolam update` call this for non-chroma backends.
func RunV3Sync(opts V3SyncOptions, outputFn func(string)) (*UpdateResult, *domain.Project, error) {
	projectDir, err := domain.ProjectDir(opts.ProjectName)
	if err != nil {
		return nil, nil, err
	}

	if opts.Reset {
		if err := domain.Remove(projectDir); err != nil {
			return nil, nil, fmt.Errorf("resetting project: %w", err)
		}
	}

	project, err := loadOrCreateProject(projectDir, opts)
	if err != nil {
		return nil, nil, err
	}

	if project.Backend == "chroma" {
		return nil, nil, fmt.Errorf("project %q uses the legacy chroma backend; manage it with 'zolam chromadb'/'zolam collections', not the v3 flat-file pipeline", opts.ProjectName)
	}
	if project.EmbeddingModel != domain.DefaultEmbeddingModel {
		return nil, nil, fmt.Errorf("project %q was indexed with embedding model %q, but zolam now defaults to %q; re-run with --reset to re-index", opts.ProjectName, project.EmbeddingModel, domain.DefaultEmbeddingModel)
	}

	oldHashes, err := LoadFileHashes(projectDir)
	if err != nil {
		return nil, nil, err
	}

	newHashes := make(map[string]string)
	for _, dir := range project.SourceDirs {
		hashes, err := HashDirectory(dir, project.Extensions)
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

	if err := GenerateIndexMD(project, opts.ProjectName, projectDir, newHashes); err != nil {
		return nil, nil, err
	}

	return result, project, nil
}

func loadOrCreateProject(projectDir string, opts V3SyncOptions) (*domain.Project, error) {
	if !domain.Exists(projectDir) {
		if len(opts.Dirs) == 0 {
			return nil, fmt.Errorf("project %q does not exist yet; specify at least one directory to ingest", opts.ProjectName)
		}
		backend := opts.Backend
		if backend == "" {
			backend = domain.DefaultBackend
		}
		if !domain.IsValidBackend(backend) {
			return nil, fmt.Errorf("unknown backend %q (expected duckdb, jsonl, or chroma)", backend)
		}
		extensions := opts.Extensions
		if len(extensions) == 0 {
			extensions = SupportedFileExtensions
		}
		absDirs, err := absPaths(opts.Dirs)
		if err != nil {
			return nil, err
		}
		return domain.New(backend, absDirs, extensions), nil
	}

	project, err := domain.Load(projectDir)
	if err != nil {
		return nil, fmt.Errorf("loading project %q: %w", opts.ProjectName, err)
	}
	if opts.Backend != "" && opts.Backend != project.Backend {
		return nil, fmt.Errorf("project %q was created with backend %q; use --reset to switch to %q", opts.ProjectName, project.Backend, opts.Backend)
	}
	if len(opts.Dirs) > 0 {
		absDirs, err := absPaths(opts.Dirs)
		if err != nil {
			return nil, err
		}
		project.SourceDirs = absDirs
	}
	if len(opts.Extensions) > 0 {
		project.Extensions = opts.Extensions
	}
	return project, nil
}

// LoadV3Project loads an existing flat-file project by name, returning a
// clear, actionable error if it doesn't exist, uses the legacy chroma
// backend, or was indexed with a now-unsupported embedding model.
func LoadV3Project(name string) (*domain.Project, string, error) {
	projectDir, err := domain.ProjectDir(name)
	if err != nil {
		return nil, "", err
	}
	if !domain.Exists(projectDir) {
		return nil, "", fmt.Errorf("no project named %q; run 'zolam ingest <dirs> --project %s' first", name, name)
	}
	project, err := domain.Load(projectDir)
	if err != nil {
		return nil, "", fmt.Errorf("loading project %q: %w", name, err)
	}
	if project.Backend == "chroma" {
		return nil, "", fmt.Errorf("project %q uses the legacy chroma backend; 'zolam query' only supports duckdb/jsonl projects", name)
	}
	if project.EmbeddingModel != domain.DefaultEmbeddingModel {
		return nil, "", fmt.Errorf("project %q was indexed with embedding model %q, but zolam now defaults to %q; re-run 'zolam ingest --reset' to re-index", name, project.EmbeddingModel, domain.DefaultEmbeddingModel)
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
