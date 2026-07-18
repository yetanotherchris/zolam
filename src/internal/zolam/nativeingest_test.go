package zolam

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yetanotherchris/zolam/internal/domain"
)

func TestRunIngestThenRunQuery_EndToEnd(t *testing.T) {
	prepareCachedEmbeddingAssets(t)

	root := t.TempDir()
	projectDir := domain.LocalProjectDir(root)

	writeFile := func(name, content string) string {
		p := filepath.Join(root, name)
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
		return p
	}
	catFile := writeFile("cat.txt", "The cat sat on the mat and purred contentedly.")
	physicsFile := writeFile("physics.txt", "Quantum entanglement violates local realism in fascinating ways.")
	staleFile := writeFile("stale.txt", "this file will be removed before query time")

	project := domain.New("duckdb", []string{root}, []string{".txt"})

	summary, err := RunIngest(project, projectDir,
		[]string{catFile, physicsFile, staleFile}, nil, func(line string) { t.Log(line) })
	if err != nil {
		t.Fatalf("RunIngest: %v", err)
	}
	if summary.FilesProcessed != 3 || summary.FilesErrored != 0 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if summary.ChunksWritten != 3 {
		t.Fatalf("expected 3 chunks written, got %d", summary.ChunksWritten)
	}

	// Now remove staleFile and re-run ingest as an incremental update.
	summary, err = RunIngest(project, projectDir, nil, []string{staleFile}, func(line string) { t.Log(line) })
	if err != nil {
		t.Fatalf("RunIngest (removal): %v", err)
	}
	if summary.FilesRemoved != 1 {
		t.Fatalf("expected 1 file removed, got %+v", summary)
	}

	hits, err := RunQuery(project, projectDir, "a feline resting on a rug", 5, false)
	if err != nil {
		t.Fatalf("RunQuery: %v", err)
	}
	if len(hits) == 0 {
		t.Fatal("expected at least one hit")
	}
	if hits[0].Path != catFile {
		t.Errorf("expected cat.txt to be the top semantic hit for a cat-related query, got %+v", hits[0])
	}
	for _, h := range hits {
		if h.Path == staleFile {
			t.Errorf("removed file %s should not appear in query results", staleFile)
		}
	}

	kwHits, err := RunQuery(project, projectDir, "quantum", 5, true)
	if err != nil {
		t.Fatalf("RunQuery (keyword): %v", err)
	}
	if len(kwHits) != 1 || kwHits[0].Path != physicsFile {
		t.Fatalf("expected keyword search to find physics.txt only, got %+v", kwHits)
	}
}
