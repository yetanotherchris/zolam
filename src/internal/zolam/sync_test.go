package zolam

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yetanotherchris/zolam/internal/domain"
)

func noopOutput(string) {}

func TestRunSync_FirstIngestRequiresDirs(t *testing.T) {
	root := t.TempDir()

	_, _, err := RunSync(SyncOptions{Root: root}, noopOutput)
	if err == nil {
		t.Fatal("expected an error when ingest is given no directories")
	}
	if !strings.Contains(err.Error(), "requires at least one directory") {
		t.Errorf("error = %q, want a message about requiring a directory", err.Error())
	}
	if domain.Exists(domain.LocalProjectDir(root)) {
		t.Error("expected no project.json to be created when the required-dirs error fires")
	}
}

func TestRunSync_FirstIngestWithDirsSucceeds(t *testing.T) {
	root := t.TempDir()
	// No matching files in root, so no chunks need embedding: RunSync
	// never has to shell out to the Python ingest pipeline, keeping this
	// test hermetic (no uv/network dependency).

	_, proj, err := RunSync(SyncOptions{Root: root, Dirs: []string{root}, Backend: "jsonl"}, noopOutput)
	if err != nil {
		t.Fatalf("RunSync() returned error: %v", err)
	}
	if len(proj.SourceDirs) != 1 || proj.SourceDirs[0] != root {
		t.Errorf("SourceDirs = %v, want [%s]", proj.SourceDirs, root)
	}
}

func TestRunSync_ResyncWithNoDirsErrors(t *testing.T) {
	root := t.TempDir()

	if _, _, err := RunSync(SyncOptions{Root: root, Dirs: []string{root}, Backend: "jsonl"}, noopOutput); err != nil {
		t.Fatalf("first RunSync() returned error: %v", err)
	}

	// A directory is required on every call, not just the first — omitting
	// it on a re-sync must error rather than silently reusing the stored
	// source_dirs.
	if _, _, err := RunSync(SyncOptions{Root: root}, noopOutput); err == nil {
		t.Fatal("expected an error when re-syncing with no directories")
	}
}

func TestRunSync_ResetWithNoDirsErrors(t *testing.T) {
	root := t.TempDir()

	if _, _, err := RunSync(SyncOptions{Root: root, Dirs: []string{root}, Backend: "jsonl"}, noopOutput); err != nil {
		t.Fatalf("first RunSync() returned error: %v", err)
	}

	if _, _, err := RunSync(SyncOptions{Root: root, Reset: true}, noopOutput); err == nil {
		t.Fatal("expected an error when resetting with no directories")
	}
}

func TestRunSync_ResyncWithDirsAddsToSourceDirs(t *testing.T) {
	root := t.TempDir()
	other := t.TempDir()

	if _, _, err := RunSync(SyncOptions{Root: root, Dirs: []string{root}, Backend: "jsonl"}, noopOutput); err != nil {
		t.Fatalf("first RunSync() returned error: %v", err)
	}

	// Naming a second, different directory on a later ingest must add to
	// the project's source_dirs, not replace them — only --reset drops a
	// previously-ingested directory. This mirrors the README's own
	// example of ingesting one directory and then another.
	_, proj, err := RunSync(SyncOptions{Root: root, Dirs: []string{other}}, noopOutput)
	if err != nil {
		t.Fatalf("second RunSync() returned error: %v", err)
	}
	if len(proj.SourceDirs) != 2 || proj.SourceDirs[0] != root || proj.SourceDirs[1] != other {
		t.Errorf("SourceDirs = %v, want [%s %s]", proj.SourceDirs, root, other)
	}
}

func TestRunSync_ResyncWithNewDirDoesNotWipeSidecarsOrChunks(t *testing.T) {
	prepareCachedEmbeddingAssets(t)

	root := t.TempDir()
	dir1 := filepath.Join(root, "dir1")
	dir2 := filepath.Join(root, "dir2")
	if err := os.MkdirAll(dir1, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir2, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir1, "a.txt"), []byte("hello from dir1"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir2, "b.txt"), []byte("hello from dir2"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, _, err := RunSync(SyncOptions{Root: root, Dirs: []string{dir1}, Backend: "jsonl"}, noopOutput); err != nil {
		t.Fatalf("first RunSync() returned error: %v", err)
	}

	projectDir := domain.LocalProjectDir(root)
	hashesAfterFirst, err := LoadFileHashes(projectDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := hashesAfterFirst[filepath.Join("dir1", "a.txt")]; !ok {
		t.Fatalf("expected dir1/a.txt to be hashed after first ingest, got %v", hashesAfterFirst)
	}

	result, _, err := RunSync(SyncOptions{Root: root, Dirs: []string{dir2}}, noopOutput)
	if err != nil {
		t.Fatalf("second RunSync() returned error: %v", err)
	}
	if result.Removed != 0 {
		t.Errorf("second RunSync() Removed = %d, want 0 (dir1 should stay tracked)", result.Removed)
	}

	hashesAfterSecond, err := LoadFileHashes(projectDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := hashesAfterSecond[filepath.Join("dir1", "a.txt")]; !ok {
		t.Errorf("dir1/a.txt was dropped from file-hashes.json after ingesting dir2: %v", hashesAfterSecond)
	}
	if _, ok := hashesAfterSecond[filepath.Join("dir2", "b.txt")]; !ok {
		t.Errorf("dir2/b.txt missing from file-hashes.json after ingesting dir2: %v", hashesAfterSecond)
	}
}

func TestRunUpdate_NoProjectErrors(t *testing.T) {
	root := t.TempDir()

	_, _, err := RunUpdate(root, false, noopOutput)
	if err == nil {
		t.Fatal("expected an error when no project exists yet")
	}
	if !strings.Contains(err.Error(), "run 'zolam ingest <dir>' there first") {
		t.Errorf("error = %q, want a message pointing at 'zolam ingest <dir>'", err.Error())
	}
}

func TestRunUpdate_ReusesStoredDirs(t *testing.T) {
	root := t.TempDir()

	if _, _, err := RunSync(SyncOptions{Root: root, Dirs: []string{root}, Backend: "jsonl"}, noopOutput); err != nil {
		t.Fatalf("first RunSync() returned error: %v", err)
	}

	_, proj, err := RunUpdate(root, false, noopOutput)
	if err != nil {
		t.Fatalf("RunUpdate() returned error: %v", err)
	}
	if len(proj.SourceDirs) != 1 || proj.SourceDirs[0] != root {
		t.Errorf("SourceDirs = %v, want [%s]", proj.SourceDirs, root)
	}
}

func TestRunUpdate_ResetReusesStoredDirs(t *testing.T) {
	root := t.TempDir()

	if _, _, err := RunSync(SyncOptions{Root: root, Dirs: []string{root}, Backend: "jsonl"}, noopOutput); err != nil {
		t.Fatalf("first RunSync() returned error: %v", err)
	}

	_, proj, err := RunUpdate(root, true, noopOutput)
	if err != nil {
		t.Fatalf("RunUpdate(reset=true) returned error: %v", err)
	}
	if len(proj.SourceDirs) != 1 || proj.SourceDirs[0] != root {
		t.Errorf("SourceDirs after reset = %v, want [%s]", proj.SourceDirs, root)
	}
}
