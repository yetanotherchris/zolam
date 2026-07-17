package domain

import (
	"os"
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

func TestLocalProjectDir(t *testing.T) {
	got := LocalProjectDir("/home/user/notes")
	want := filepath.Join("/home/user/notes", ".zolam")
	if got != want {
		t.Errorf("LocalProjectDir() = %q, want %q", got, want)
	}
}

func TestRemove(t *testing.T) {
	root := t.TempDir()
	projectDir := LocalProjectDir(root)
	if err := Save(projectDir, New("duckdb", nil, nil)); err != nil {
		t.Fatalf("Save() returned error: %v", err)
	}

	// A real source file living in root, alongside the project's .zolam/
	// folder, since root is normally the user's own working directory.
	userFile := filepath.Join(root, "notes.md")
	if err := os.WriteFile(userFile, []byte("keep me"), 0o644); err != nil {
		t.Fatalf("writing fixture user file: %v", err)
	}

	if err := Remove(projectDir); err != nil {
		t.Fatalf("Remove() returned error: %v", err)
	}
	if Exists(projectDir) {
		t.Errorf("Exists() reported true after Remove()")
	}
	if _, err := os.Stat(userFile); err != nil {
		t.Errorf("Remove() deleted unrelated user file %q: %v", userFile, err)
	}
}
