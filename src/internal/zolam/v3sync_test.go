package zolam

import (
	"strings"
	"testing"

	"github.com/yetanotherchris/zolam/internal/domain"
)

func noopOutput(string) {}

func TestRunV3Sync_FirstIngestRequiresDirs(t *testing.T) {
	root := t.TempDir()

	_, _, err := RunV3Sync(V3SyncOptions{Root: root}, noopOutput)
	if err == nil {
		t.Fatal("expected an error when first-time ingest is given no directories")
	}
	if !strings.Contains(err.Error(), "pass one or more subdirectories") {
		t.Errorf("error = %q, want a message about naming a subdirectory", err.Error())
	}
	if domain.Exists(domain.LocalProjectDir(root)) {
		t.Error("expected no project.json to be created when the required-dirs error fires")
	}
}

func TestRunV3Sync_FirstIngestWithDirsSucceeds(t *testing.T) {
	root := t.TempDir()
	// No matching files in root, so no chunks need embedding: RunV3Sync
	// never has to shell out to the Python ingest pipeline, keeping this
	// test hermetic (no uv/network dependency).

	_, proj, err := RunV3Sync(V3SyncOptions{Root: root, Dirs: []string{root}, Backend: "jsonl"}, noopOutput)
	if err != nil {
		t.Fatalf("RunV3Sync() returned error: %v", err)
	}
	if len(proj.SourceDirs) != 1 || proj.SourceDirs[0] != root {
		t.Errorf("SourceDirs = %v, want [%s]", proj.SourceDirs, root)
	}
}

func TestRunV3Sync_ResyncWithNoDirsReusesStored(t *testing.T) {
	root := t.TempDir()

	if _, _, err := RunV3Sync(V3SyncOptions{Root: root, Dirs: []string{root}, Backend: "jsonl"}, noopOutput); err != nil {
		t.Fatalf("first RunV3Sync() returned error: %v", err)
	}

	// Re-running with no Dirs and no Reset must reuse the stored source_dirs.
	_, proj, err := RunV3Sync(V3SyncOptions{Root: root}, noopOutput)
	if err != nil {
		t.Fatalf("resync RunV3Sync() returned error: %v", err)
	}
	if len(proj.SourceDirs) != 1 || proj.SourceDirs[0] != root {
		t.Errorf("SourceDirs after resync = %v, want [%s]", proj.SourceDirs, root)
	}
}

func TestRunV3Sync_ResetWithNoDirsReusesStored(t *testing.T) {
	root := t.TempDir()

	if _, _, err := RunV3Sync(V3SyncOptions{Root: root, Dirs: []string{root}, Backend: "jsonl"}, noopOutput); err != nil {
		t.Fatalf("first RunV3Sync() returned error: %v", err)
	}

	// A bare '--reset' (no Dirs) is the documented recovery path for an
	// embedding-model mismatch; it must not hit the "name a directory" error.
	_, proj, err := RunV3Sync(V3SyncOptions{Root: root, Reset: true}, noopOutput)
	if err != nil {
		t.Fatalf("reset RunV3Sync() returned error: %v", err)
	}
	if len(proj.SourceDirs) != 1 || proj.SourceDirs[0] != root {
		t.Errorf("SourceDirs after reset = %v, want [%s]", proj.SourceDirs, root)
	}
}
