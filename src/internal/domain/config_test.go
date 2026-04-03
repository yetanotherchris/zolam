package domain

import (
	"testing"
)

func TestLoadConfig_Defaults(t *testing.T) {
	// Clear all relevant env vars so defaults are used.
	for _, key := range []string{
		"OPENROUTER_API_KEY", "OPENROUTER_MODEL", "COLLECTION_NAME",
		"USE_LOCAL_EMBEDDINGS", "RCLONE_REMOTE", "RCLONE_SOURCE",
		"INGESTER_DATA_DIR",
	} {
		t.Setenv(key, "")
	}
	// USE_LOCAL_EMBEDDINGS must be "1" to avoid validation error for missing API key.
	t.Setenv("USE_LOCAL_EMBEDDINGS", "1")

	cfg, _, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() returned unexpected error: %v", err)
	}

	if cfg.CollectionName != "my-notes" {
		t.Errorf("CollectionName = %q, want %q", cfg.CollectionName, "my-notes")
	}
	if cfg.OpenRouterModel != "openai/text-embedding-3-small" {
		t.Errorf("OpenRouterModel = %q, want %q", cfg.OpenRouterModel, "openai/text-embedding-3-small")
	}
	if cfg.DataDir != "./chromadb-data" {
		t.Errorf("DataDir = %q, want %q", cfg.DataDir, "./chromadb-data")
	}
	if cfg.RcloneRemote != "gdrive" {
		t.Errorf("RcloneRemote = %q, want %q", cfg.RcloneRemote, "gdrive")
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
	t.Setenv("OPENROUTER_API_KEY", "test-key-123")
	t.Setenv("USE_LOCAL_EMBEDDINGS", "1")
	// Clear others to avoid stale state.
	t.Setenv("OPENROUTER_MODEL", "")
	t.Setenv("RCLONE_REMOTE", "")
	t.Setenv("RCLONE_SOURCE", "")
	t.Setenv("INGESTER_DATA_DIR", "")

	cfg, _, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() returned unexpected error: %v", err)
	}

	if cfg.CollectionName != "test-collection" {
		t.Errorf("CollectionName = %q, want %q", cfg.CollectionName, "test-collection")
	}
	if cfg.OpenRouterAPIKey != "test-key-123" {
		t.Errorf("OpenRouterAPIKey = %q, want %q", cfg.OpenRouterAPIKey, "test-key-123")
	}
	if !cfg.UseLocalEmbeddings {
		t.Error("UseLocalEmbeddings = false, want true")
	}
}

func TestValidate_MissingAPIKey(t *testing.T) {
	// Ensure env vars don't interfere with Validate's os.Getenv checks.
	t.Setenv("OPENROUTER_API_KEY", "")
	t.Setenv("OPENROUTER_MODEL", "some-model")
	t.Setenv("COLLECTION_NAME", "some-collection")
	t.Setenv("RCLONE_REMOTE", "some-remote")
	t.Setenv("INGESTER_DATA_DIR", "some-dir")

	cfg := &Config{
		UseLocalEmbeddings: false,
		OpenRouterAPIKey:   "",
	}

	_, errs := cfg.Validate()
	if len(errs) == 0 {
		t.Fatal("Validate() returned no errors, expected error for missing API key")
	}
}

func TestValidate_LocalEmbeddings(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "")
	t.Setenv("OPENROUTER_MODEL", "some-model")
	t.Setenv("COLLECTION_NAME", "some-collection")
	t.Setenv("RCLONE_REMOTE", "some-remote")
	t.Setenv("INGESTER_DATA_DIR", "some-dir")

	cfg := &Config{
		UseLocalEmbeddings: true,
		OpenRouterAPIKey:   "",
	}

	_, errs := cfg.Validate()
	if len(errs) != 0 {
		t.Fatalf("Validate() returned errors %v, expected none", errs)
	}
}

func TestMergeFlags(t *testing.T) {
	cfg := &Config{
		CollectionName: "original",
		DataDir:        "/original/path",
		OpenRouterModel: "original-model",
	}

	flags := map[string]string{
		"collection-name": "overridden-collection",
		"data-dir":        "/new/path",
	}

	cfg.MergeFlags(flags)

	if cfg.CollectionName != "overridden-collection" {
		t.Errorf("CollectionName = %q, want %q", cfg.CollectionName, "overridden-collection")
	}
	if cfg.DataDir != "/new/path" {
		t.Errorf("DataDir = %q, want %q", cfg.DataDir, "/new/path")
	}
	// Verify unset flags don't change existing values.
	if cfg.OpenRouterModel != "original-model" {
		t.Errorf("OpenRouterModel = %q, want %q (should be unchanged)", cfg.OpenRouterModel, "original-model")
	}

	// Verify that empty flag values don't override.
	cfg.MergeFlags(map[string]string{"collection-name": ""})
	if cfg.CollectionName != "overridden-collection" {
		t.Errorf("CollectionName changed to %q after empty flag, should remain %q", cfg.CollectionName, "overridden-collection")
	}

}
