package zolam

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yetanotherchris/zolam/internal/docker"
	"github.com/yetanotherchris/zolam/internal/domain"
)

// MigrateOptions controls a best-effort migration from an existing chroma
// collection into a v3 flat-file project of the same name.
type MigrateOptions struct {
	ProjectName string // also the chroma collection name
	Backend     string // duckdb or jsonl; defaults to duckdb
}

// MigrateFromChroma pulls every chunk's text + metadata from a running
// ChromaDB collection, reconstructs each file's text by concatenating its
// chunks in stored order, writes the reconstructed text as markdown under
// <project-dir>/migrated-sources/ (which becomes the project's permanent,
// greppable source_dirs entry), and runs it through the normal v3 ingest
// path so chunking/embedding/index-writing code is not duplicated.
//
// Original PDF/DOCX bytes are not recoverable from Chroma, so
// re-extraction is not attempted -- only re-embedding with the current
// model, per the spec.
func MigrateFromChroma(dc *docker.DockerClient, opts MigrateOptions, outputFn func(string)) (*UpdateResult, error) {
	backend := opts.Backend
	if backend == "" {
		backend = domain.DefaultBackend
	}
	if backend == "chroma" {
		return nil, fmt.Errorf("--backend chroma is not a valid migration target (migrate moves data out of chroma)")
	}

	projectDir, err := domain.ProjectDir(opts.ProjectName)
	if err != nil {
		return nil, err
	}

	outputFn(fmt.Sprintf("Fetching documents from chroma collection %q...", opts.ProjectName))
	docsByFile, err := dc.GetAllDocuments(opts.ProjectName)
	if err != nil {
		return nil, fmt.Errorf("fetching documents from chroma: %w", err)
	}
	if len(docsByFile) == 0 {
		return nil, fmt.Errorf("chroma collection %q has no documents to migrate", opts.ProjectName)
	}

	// Start clean: remove any prior v3 project/index at this name before
	// staging, since RunV3Sync treats a fresh directory as project creation.
	if err := domain.Remove(projectDir); err != nil {
		return nil, fmt.Errorf("clearing previous project %q: %w", opts.ProjectName, err)
	}

	sourcesDir := filepath.Join(projectDir, "migrated-sources")
	for relPath, text := range docsByFile {
		dest := filepath.Join(sourcesDir, sanitizeRelPath(relPath)+".md")
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return nil, fmt.Errorf("staging %s: %w", relPath, err)
		}
		if err := os.WriteFile(dest, []byte(text), 0o644); err != nil {
			return nil, fmt.Errorf("staging %s: %w", relPath, err)
		}
	}

	outputFn(fmt.Sprintf("Re-embedding %d file(s) with %s (backend=%s)...", len(docsByFile), domain.DefaultEmbeddingModel, backend))
	result, _, err := RunV3Sync(V3SyncOptions{
		ProjectName: opts.ProjectName,
		Dirs:        []string{sourcesDir},
		Extensions:  []string{".md"},
		Backend:     backend,
	}, outputFn)
	return result, err
}

func sanitizeRelPath(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	return strings.TrimPrefix(p, "/")
}
