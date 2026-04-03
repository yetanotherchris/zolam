package ingester

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestComputeHash(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	content := []byte("hello world\n")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	got, err := ComputeHash(path)
	if err != nil {
		t.Fatalf("ComputeHash() returned error: %v", err)
	}

	h := sha256.Sum256(content)
	want := hex.EncodeToString(h[:])

	if got != want {
		t.Errorf("ComputeHash() = %q, want %q", got, want)
	}
}

func TestComputeHash_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.txt")

	if err := os.WriteFile(path, []byte{}, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	got, err := ComputeHash(path)
	if err != nil {
		t.Fatalf("ComputeHash() returned error: %v", err)
	}

	// SHA-256 of empty input.
	h := sha256.Sum256([]byte{})
	want := hex.EncodeToString(h[:])

	if got != want {
		t.Errorf("ComputeHash() = %q, want %q", got, want)
	}
}

func TestHashDirectory(t *testing.T) {
	dir := t.TempDir()

	// Create test files with known content.
	files := map[string][]byte{
		"notes.md":   []byte("# Notes\nSome markdown content."),
		"readme.txt": []byte("A plain text file."),
		"main.go":    []byte("package main\nfunc main() {}"),
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), content, 0644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}

	hashes, err := HashDirectory(dir, []string{".md", ".txt"})
	if err != nil {
		t.Fatalf("HashDirectory() returned error: %v", err)
	}

	// Should only contain .md and .txt files (not .go).
	if len(hashes) != 2 {
		t.Fatalf("HashDirectory() returned %d entries, want 2", len(hashes))
	}

	// Verify the .go file is excluded.
	goPath := filepath.Join(dir, "main.go")
	if _, ok := hashes[goPath]; ok {
		t.Error("HashDirectory() included main.go, but .go was not in the extensions list")
	}

	// Verify hashes for included files.
	for _, name := range []string{"notes.md", "readme.txt"} {
		path := filepath.Join(dir, name)
		got, ok := hashes[path]
		if !ok {
			t.Errorf("HashDirectory() missing entry for %s", name)
			continue
		}
		h := sha256.Sum256(files[name])
		want := hex.EncodeToString(h[:])
		if got != want {
			t.Errorf("hash for %s = %q, want %q", name, got, want)
		}
	}
}

func TestHashDirectory_Empty(t *testing.T) {
	dir := t.TempDir()

	hashes, err := HashDirectory(dir, []string{".md", ".txt"})
	if err != nil {
		t.Fatalf("HashDirectory() returned error: %v", err)
	}

	if len(hashes) != 0 {
		t.Errorf("HashDirectory() returned %d entries for empty dir, want 0", len(hashes))
	}
}
