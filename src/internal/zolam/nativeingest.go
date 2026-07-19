package zolam

import (
	"fmt"
	"os"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/yetanotherchris/zolam/internal/domain"
)

// ingestWorkers bounds how many files are extracted/chunked/embedded
// concurrently. Extraction and embedding are CPU-bound (OCR, ONNX
// inference); the DB write itself is fast and always serialized through a
// single writer regardless of this number.
const ingestWorkers = 4

// IngestSummary is the final tally of an ingest/update run.
type IngestSummary struct {
	FilesProcessed int
	FilesErrored   int
	FilesRemoved   int
	ChunksWritten  int
	Errors         []string
}

// QueryHit is a single ranked or keyword-matched query result.
type QueryHit struct {
	Path  string
	Chunk int
	Page  *int
	Text  string
	Score *float64
}

// indexBackend is the storage interface both SQLiteRepo and JsonlRepo
// satisfy; RunIngest/RunQuery are written against it so the backend choice
// is just which Open* constructor gets called.
type indexBackend interface {
	DeletePaths(paths []string) error
	InsertChunks(records []ChunkRecord) error
	Search(queryEmbedding []float32, topK int) ([]SearchHit, error)
	KeywordSearch(term string, topK int) ([]SearchHit, error)
	Close() error
}

func openBackend(projectDir, backendName, model string, dims int) (indexBackend, error) {
	switch backendName {
	case "sqlite":
		return OpenSQLiteRepo(projectDir, model, dims)
	case "jsonl":
		return OpenJsonlRepo(projectDir, model, dims)
	default:
		return nil, fmt.Errorf("unsupported backend %q (expected sqlite or jsonl)", backendName)
	}
}

// RunIngest extracts, chunks, and embeds added/changed files (in parallel,
// bounded by ingestWorkers), removes deleted files' chunks, and writes
// everything to the project's index. Holds the project's lock file for
// its entire duration: only one process may have the index open at a
// time, whether that's this one or a concurrent 'zolam query'.
func RunIngest(project *domain.Project, projectDir string, files, removed []string, outputFn func(string)) (*IngestSummary, error) {
	// A brand new project's directory may not exist yet: RunSync only
	// persists project.json after ingest succeeds, matching the old
	// pipeline's behaviour of creating the directory itself on first use.
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating project directory: %w", err)
	}

	release, err := acquireLock(projectDir)
	if err != nil {
		return nil, err
	}
	defer release()

	backend, err := openBackend(projectDir, project.Backend, project.EmbeddingModel, project.EmbeddingDims)
	if err != nil {
		return nil, err
	}
	defer backend.Close()

	toDelete := append(append([]string{}, removed...), files...)
	if err := backend.DeletePaths(toDelete); err != nil {
		return nil, fmt.Errorf("deleting stale chunks: %w", err)
	}
	for _, p := range removed {
		if err := RemoveSidecar(projectDir, p); err != nil {
			return nil, fmt.Errorf("removing sidecar for %s: %w", p, err)
		}
	}

	summary := &IngestSummary{FilesRemoved: len(removed)}
	if len(files) == 0 {
		return summary, nil
	}

	embedder, err := NewEmbedder(outputFn)
	if err != nil {
		return nil, err
	}
	defer embedder.Close()

	type fileResult struct {
		records []ChunkRecord
		err     error
	}

	results := make([]fileResult, len(files))
	var progressMu sync.Mutex
	completed := 0

	var g errgroup.Group
	g.SetLimit(ingestWorkers)
	for i, path := range files {
		i, path := i, path
		g.Go(func() error {
			pending, err := ExtractAndChunk(path, projectDir)
			if err != nil {
				results[i] = fileResult{err: err}
			} else if len(pending) > 0 {
				texts := make([]string, len(pending))
				for j, c := range pending {
					texts[j] = c.Text
				}
				vectors, embErr := embedder.Embed(texts)
				if embErr != nil {
					results[i] = fileResult{err: embErr}
				} else {
					records := make([]ChunkRecord, len(pending))
					for j, c := range pending {
						rec := ChunkRecord{Path: path, ChunkNum: j, Text: c.Text, Embedding: vectors[j]}
						if c.Page != 0 {
							page := c.Page
							rec.Page = &page
						}
						records[j] = rec
					}
					results[i] = fileResult{records: records}
				}
			}

			progressMu.Lock()
			completed++
			if outputFn != nil {
				outputFn(fmt.Sprintf("[%d/%d] %s", completed, len(files), path))
			}
			progressMu.Unlock()
			return nil
		})
	}
	g.Wait() // errors are collected per-file in results, not propagated here

	for i, path := range files {
		r := results[i]
		if r.err != nil {
			summary.FilesErrored++
			summary.Errors = append(summary.Errors, fmt.Sprintf("%s: %v", path, r.err))
			if outputFn != nil {
				outputFn(fmt.Sprintf("  ERROR: %v", r.err))
			}
			continue
		}
		if len(r.records) > 0 {
			if err := backend.InsertChunks(r.records); err != nil {
				return nil, fmt.Errorf("writing chunks for %s: %w", path, err)
			}
			summary.ChunksWritten += len(r.records)
		}
		summary.FilesProcessed++
	}

	return summary, nil
}

// RunQuery embeds queryText (unless keyword search was asked for) and
// searches the project's index.
func RunQuery(project *domain.Project, projectDir, queryText string, topK int, keyword bool) ([]QueryHit, error) {
	release, err := acquireLock(projectDir)
	if err != nil {
		return nil, err
	}
	defer release()

	backend, err := openBackend(projectDir, project.Backend, project.EmbeddingModel, project.EmbeddingDims)
	if err != nil {
		return nil, err
	}
	defer backend.Close()

	var hits []SearchHit
	if keyword {
		hits, err = backend.KeywordSearch(queryText, topK)
	} else {
		embedder, embErr := NewEmbedder(nil)
		if embErr != nil {
			return nil, embErr
		}
		defer embedder.Close()
		vectors, embErr := embedder.Embed([]string{queryText})
		if embErr != nil {
			return nil, embErr
		}
		hits, err = backend.Search(vectors[0], topK)
	}
	if err != nil {
		return nil, err
	}

	out := make([]QueryHit, len(hits))
	for i, h := range hits {
		out[i] = QueryHit{Path: h.Path, Chunk: h.Chunk, Page: h.Page, Text: h.Text, Score: h.Score}
	}
	return out, nil
}
