package domain

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_Defaults(t *testing.T) {
	// Point config.json to a non-existent temp file so the real one doesn't interfere.
	configPathOverride = filepath.Join(t.TempDir(), "config.json")
	t.Cleanup(func() { configPathOverride = "" })

	for _, key := range []string{
		"COLLECTION_NAME",
		"RCLONE_SOURCE",
		"RCLONE_CONFIG_DIR", "ZOLAM_DATA_DIR",
	} {
		t.Setenv(key, "")
	}

	cfg, _, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() returned unexpected error: %v", err)
	}

	if cfg.CollectionName != "my-notes" {
		t.Errorf("CollectionName = %q, want %q", cfg.CollectionName, "my-notes")
	}
	homeDir, _ := os.UserHomeDir()
	expectedDataDir := filepath.ToSlash(filepath.Join(homeDir, ".zolam"))
	if cfg.DataDir != expectedDataDir {
		t.Errorf("DataDir = %q, want %q", cfg.DataDir, expectedDataDir)
	}
	if envVal := os.Getenv("ZOLAM_DATA_DIR"); envVal != expectedDataDir {
		t.Errorf("ZOLAM_DATA_DIR env = %q, want %q", envVal, expectedDataDir)
	}
	expectedRcloneConfigDir := filepath.ToSlash(filepath.Join(homeDir, ".config", "rclone"))
	if cfg.RcloneConfigDir != expectedRcloneConfigDir {
		t.Errorf("RcloneConfigDir = %q, want %q", cfg.RcloneConfigDir, expectedRcloneConfigDir)
	}
}

func TestLoadConfig_EnvVars(t *testing.T) {
	configPathOverride = filepath.Join(t.TempDir(), "config.json")
	t.Cleanup(func() { configPathOverride = "" })

	t.Setenv("COLLECTION_NAME", "test-collection")
	t.Setenv("RCLONE_SOURCE", "")
	t.Setenv("RCLONE_CONFIG_DIR", "")
	t.Setenv("ZOLAM_DATA_DIR", "")

	cfg, _, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() returned unexpected error: %v", err)
	}

	if cfg.CollectionName != "test-collection" {
		t.Errorf("CollectionName = %q, want %q", cfg.CollectionName, "test-collection")
	}
}

func TestMergeFlags(t *testing.T) {
	cfg := &Config{
		CollectionName:  "original",
		DataDir:         "/original/path",
		RcloneConfigDir: "/original/rclone",
	}

	flags := map[string]string{
		"collection-name":   "overridden-collection",
		"data-dir":          "/new/path",
		"rclone-config-dir": "/new/rclone/config",
	}

	cfg.MergeFlags(flags)

	if cfg.CollectionName != "overridden-collection" {
		t.Errorf("CollectionName = %q, want %q", cfg.CollectionName, "overridden-collection")
	}
	if cfg.DataDir != "/new/path" {
		t.Errorf("DataDir = %q, want %q", cfg.DataDir, "/new/path")
	}
	if cfg.RcloneConfigDir != "/new/rclone/config" {
		t.Errorf("RcloneConfigDir = %q, want %q", cfg.RcloneConfigDir, "/new/rclone/config")
	}

	cfg.MergeFlags(map[string]string{"collection-name": ""})
	if cfg.CollectionName != "overridden-collection" {
		t.Errorf("CollectionName changed to %q after empty flag, should remain %q", cfg.CollectionName, "overridden-collection")
	}
}

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

func TestConfigJSON_LoadSave(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.json")

	cfg := &Config{
		CollectionName:  "test-col",
		RcloneSource:    "gdrive:docs",
		RcloneConfigDir: "/home/user/.config/rclone",
		DataDir:         "/home/user/.zolam",
		Directories: []DirectoryEntry{
			{Path: "/home/user/notes", Extensions: []string{".md", ".txt"}},
		},
	}

	// Write config via the JSON format directly (simulating SaveConfig)
	cj := configJSON{
		CollectionName:  cfg.CollectionName,
		RcloneSource:    cfg.RcloneSource,
		RcloneConfigDir: cfg.RcloneConfigDir,
		DataDir:         cfg.DataDir,
		Directories:     cfg.Directories,
	}
	data, err := json.MarshalIndent(cj, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	// Load it back
	loaded, err := loadConfigJSON(path)
	if err != nil {
		t.Fatalf("loadConfigJSON: %v", err)
	}

	if loaded.CollectionName != "test-col" {
		t.Errorf("CollectionName = %q, want %q", loaded.CollectionName, "test-col")
	}
	if len(loaded.Directories) != 1 {
		t.Fatalf("Directories length = %d, want 1", len(loaded.Directories))
	}
	if loaded.Directories[0].Path != "/home/user/notes" {
		t.Errorf("Directory path = %q, want %q", loaded.Directories[0].Path, "/home/user/notes")
	}
}

func TestAddOrUpdateDirectory(t *testing.T) {
	cfg := &Config{}

	cfg.AddOrUpdateDirectory("/home/user/notes", []string{".md"})
	if len(cfg.Directories) != 1 {
		t.Fatalf("expected 1 directory, got %d", len(cfg.Directories))
	}

	cfg.AddOrUpdateDirectory("/home/user/docs", []string{".pdf"})
	if len(cfg.Directories) != 2 {
		t.Fatalf("expected 2 directories, got %d", len(cfg.Directories))
	}

	// Update existing
	cfg.AddOrUpdateDirectory("/home/user/notes", []string{".md", ".txt"})
	if len(cfg.Directories) != 2 {
		t.Fatalf("expected 2 directories after update, got %d", len(cfg.Directories))
	}
	if len(cfg.Directories[0].Extensions) != 2 {
		t.Errorf("expected 2 extensions after update, got %d", len(cfg.Directories[0].Extensions))
	}
}

func TestRemoveDirectory(t *testing.T) {
	cfg := &Config{
		Directories: []DirectoryEntry{
			{Path: "/a", Extensions: []string{".md"}},
			{Path: "/b", Extensions: []string{".txt"}},
			{Path: "/c", Extensions: []string{".pdf"}},
		},
	}

	cfg.RemoveDirectory(1)
	if len(cfg.Directories) != 2 {
		t.Fatalf("expected 2 directories, got %d", len(cfg.Directories))
	}
	if cfg.Directories[0].Path != "/a" || cfg.Directories[1].Path != "/c" {
		t.Errorf("unexpected directories after removal: %v", cfg.Directories)
	}

	// Out of bounds should be a no-op
	cfg.RemoveDirectory(-1)
	cfg.RemoveDirectory(5)
	if len(cfg.Directories) != 2 {
		t.Errorf("out-of-bounds removal changed directories")
	}
}
