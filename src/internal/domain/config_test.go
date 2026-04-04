package domain

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_Defaults(t *testing.T) {
	// Clear all relevant env vars so defaults are used.
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
	expectedDataDir := filepath.ToSlash(filepath.Join(homeDir, ".zolam", "chromadb-data"))
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

	expectedExts := []string{
		".md", ".pdf", ".docx", ".txt",
		".py", ".cs", ".js", ".ts",
		".json", ".yml", ".yaml",
	}
	if len(cfg.Extensions) != len(expectedExts) {
		t.Fatalf("Extensions length = %d, want %d", len(cfg.Extensions), len(expectedExts))
	}
	for i, ext := range expectedExts {
		if cfg.Extensions[i] != ext {
			t.Errorf("Extensions[%d] = %q, want %q", i, cfg.Extensions[i], ext)
		}
	}
}

func TestLoadConfig_EnvVars(t *testing.T) {
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
		"collection-name":  "overridden-collection",
		"data-dir":         "/new/path",
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

	// Verify that empty flag values don't override.
	cfg.MergeFlags(map[string]string{"collection-name": ""})
	if cfg.CollectionName != "overridden-collection" {
		t.Errorf("CollectionName changed to %q after empty flag, should remain %q", cfg.CollectionName, "overridden-collection")
	}
}
