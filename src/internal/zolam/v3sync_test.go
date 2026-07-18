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
		t.Fatal("expected an error when ingest is given no directories")
	}
	if !strings.Contains(err.Error(), "requires at least one directory") {
		t.Errorf("error = %q, want a message about requiring a directory", err.Error())
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

func TestRunV3Sync_ResyncWithNoDirsErrors(t *testing.T) {
	root := t.TempDir()

	if _, _, err := RunV3Sync(V3SyncOptions{Root: root, Dirs: []string{root}, Backend: "jsonl"}, noopOutput); err != nil {
		t.Fatalf("first RunV3Sync() returned error: %v", err)
	}

	// A directory is required on every call, not just the first — omitting
	// it on a re-sync must error rather than silently reusing the stored
	// source_dirs.
	if _, _, err := RunV3Sync(V3SyncOptions{Root: root}, noopOutput); err == nil {
		t.Fatal("expected an error when re-syncing with no directories")
	}
}

func TestRunV3Sync_ResetWithNoDirsErrors(t *testing.T) {
	root := t.TempDir()

	if _, _, err := RunV3Sync(V3SyncOptions{Root: root, Dirs: []string{root}, Backend: "jsonl"}, noopOutput); err != nil {
		t.Fatalf("first RunV3Sync() returned error: %v", err)
	}

	if _, _, err := RunV3Sync(V3SyncOptions{Root: root, Reset: true}, noopOutput); err == nil {
		t.Fatal("expected an error when resetting with no directories")
	}
}

func TestRunV3Sync_ResyncWithDirsUpdatesSourceDirs(t *testing.T) {
	root := t.TempDir()
	other := t.TempDir()

	if _, _, err := RunV3Sync(V3SyncOptions{Root: root, Dirs: []string{root}, Backend: "jsonl"}, noopOutput); err != nil {
		t.Fatalf("first RunV3Sync() returned error: %v", err)
	}

	_, proj, err := RunV3Sync(V3SyncOptions{Root: root, Dirs: []string{other}}, noopOutput)
	if err != nil {
		t.Fatalf("second RunV3Sync() returned error: %v", err)
	}
	if len(proj.SourceDirs) != 1 || proj.SourceDirs[0] != other {
		t.Errorf("SourceDirs = %v, want [%s]", proj.SourceDirs, other)
	}
}

func TestRunV3Update_NoProjectErrors(t *testing.T) {
	root := t.TempDir()

	_, _, err := RunV3Update(root, false, noopOutput)
	if err == nil {
		t.Fatal("expected an error when no project exists yet")
	}
	if !strings.Contains(err.Error(), "run 'zolam ingest <dir>' there first") {
		t.Errorf("error = %q, want a message pointing at 'zolam ingest <dir>'", err.Error())
	}
}

func TestRunV3Update_ReusesStoredDirs(t *testing.T) {
	root := t.TempDir()

	if _, _, err := RunV3Sync(V3SyncOptions{Root: root, Dirs: []string{root}, Backend: "jsonl"}, noopOutput); err != nil {
		t.Fatalf("first RunV3Sync() returned error: %v", err)
	}

	_, proj, err := RunV3Update(root, false, noopOutput)
	if err != nil {
		t.Fatalf("RunV3Update() returned error: %v", err)
	}
	if len(proj.SourceDirs) != 1 || proj.SourceDirs[0] != root {
		t.Errorf("SourceDirs = %v, want [%s]", proj.SourceDirs, root)
	}
}

func TestRunV3Update_ResetReusesStoredDirs(t *testing.T) {
	root := t.TempDir()

	if _, _, err := RunV3Sync(V3SyncOptions{Root: root, Dirs: []string{root}, Backend: "jsonl"}, noopOutput); err != nil {
		t.Fatalf("first RunV3Sync() returned error: %v", err)
	}

	_, proj, err := RunV3Update(root, true, noopOutput)
	if err != nil {
		t.Fatalf("RunV3Update(reset=true) returned error: %v", err)
	}
	if len(proj.SourceDirs) != 1 || proj.SourceDirs[0] != root {
		t.Errorf("SourceDirs after reset = %v, want [%s]", proj.SourceDirs, root)
	}
}
