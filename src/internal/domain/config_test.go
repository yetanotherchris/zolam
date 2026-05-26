package domain

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSupportedFileExtensions(t *testing.T) {
	if len(SupportedFileExtensions) == 0 {
		t.Fatal("SupportedFileExtensions should not be empty")
	}
	for _, ext := range SupportedFileExtensions {
		if ext[0] != '.' {
			t.Errorf("extension %q should start with a dot", ext)
		}
	}
}

func TestNewConfig_Defaults(t *testing.T) {
	os.Unsetenv("ZOLAM_CHROMADB_DATA_DIR")

	cfg := NewConfig()

	homeDir, _ := os.UserHomeDir()
	expectedDataDir := filepath.ToSlash(filepath.Join(homeDir, ".zolam"))
	if cfg.DataDir != expectedDataDir {
		t.Errorf("DataDir = %q, want %q", cfg.DataDir, expectedDataDir)
	}

	expectedChromaDir := filepath.ToSlash(filepath.Join(homeDir, ".zolam", "chromadb"))
	if v := os.Getenv("ZOLAM_CHROMADB_DATA_DIR"); v != expectedChromaDir {
		t.Errorf("ZOLAM_CHROMADB_DATA_DIR = %q, want %q", v, expectedChromaDir)
	}
}

func TestNewConfig_EnvOverride(t *testing.T) {
	t.Setenv("ZOLAM_CHROMADB_DATA_DIR", "/custom/chromadb")
	defer os.Unsetenv("ZOLAM_CHROMADB_DATA_DIR")

	cfg := NewConfig()

	if v := os.Getenv("ZOLAM_CHROMADB_DATA_DIR"); v != "/custom/chromadb" {
		t.Errorf("ZOLAM_CHROMADB_DATA_DIR = %q, want %q", v, "/custom/chromadb")
	}
	// DataDir is always the zolam home, not the chromadb-specific path
	homeDir, _ := os.UserHomeDir()
	expectedDataDir := filepath.ToSlash(filepath.Join(homeDir, ".zolam"))
	if cfg.DataDir != expectedDataDir {
		t.Errorf("DataDir = %q, want %q", cfg.DataDir, expectedDataDir)
	}
}
