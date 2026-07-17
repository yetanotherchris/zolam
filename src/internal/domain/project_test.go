package domain

import (
	"path/filepath"
	"testing"
)

func TestDataDir_RespectsEnvOverride(t *testing.T) {
	t.Setenv("ZOLAM_DATA_DIR", "/custom/zolam/dir")
	dir, err := DataDir()
	if err != nil {
		t.Fatalf("DataDir() returned error: %v", err)
	}
	if dir != "/custom/zolam/dir" {
		t.Errorf("DataDir() = %q, want %q", dir, "/custom/zolam/dir")
	}
}

func TestProjectSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "my-project")

	if Exists(projectDir) {
		t.Fatalf("Exists() reported true before project.json was created")
	}

	p := New("duckdb", []string{"/notes"}, []string{".md"})
	if err := Save(projectDir, p); err != nil {
		t.Fatalf("Save() returned error: %v", err)
	}

	if !Exists(projectDir) {
		t.Fatalf("Exists() reported false after Save()")
	}

	loaded, err := Load(projectDir)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if loaded.Backend != "duckdb" || loaded.EmbeddingModel != DefaultEmbeddingModel || loaded.EmbeddingDims != DefaultEmbeddingDims {
		t.Errorf("Load() = %+v, want backend=duckdb model=%s dims=%d", loaded, DefaultEmbeddingModel, DefaultEmbeddingDims)
	}
	if len(loaded.SourceDirs) != 1 || loaded.SourceDirs[0] != "/notes" {
		t.Errorf("SourceDirs = %v, want [/notes]", loaded.SourceDirs)
	}
}

func TestListProjectNames(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ZOLAM_DATA_DIR", dir)

	names, err := ListProjectNames()
	if err != nil {
		t.Fatalf("ListProjectNames() on empty data dir returned error: %v", err)
	}
	if len(names) != 0 {
		t.Fatalf("expected no projects, got %v", names)
	}

	projDir, _ := ProjectDir("alpha")
	if err := Save(projDir, New("jsonl", []string{"/x"}, []string{".md"})); err != nil {
		t.Fatalf("Save() returned error: %v", err)
	}

	names, err = ListProjectNames()
	if err != nil {
		t.Fatalf("ListProjectNames() returned error: %v", err)
	}
	if len(names) != 1 || names[0] != "alpha" {
		t.Errorf("ListProjectNames() = %v, want [alpha]", names)
	}
}

func TestIsValidBackend(t *testing.T) {
	for _, b := range []string{"duckdb", "jsonl", "chroma"} {
		if !IsValidBackend(b) {
			t.Errorf("IsValidBackend(%q) = false, want true", b)
		}
	}
	if IsValidBackend("bogus") {
		t.Errorf("IsValidBackend(%q) = true, want false", "bogus")
	}
}

func TestRemove(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "gone")
	if err := Save(projectDir, New("duckdb", nil, nil)); err != nil {
		t.Fatalf("Save() returned error: %v", err)
	}
	if err := Remove(projectDir); err != nil {
		t.Fatalf("Remove() returned error: %v", err)
	}
	if Exists(projectDir) {
		t.Errorf("Exists() reported true after Remove()")
	}
}
