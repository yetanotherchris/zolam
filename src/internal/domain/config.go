package domain

import (
	"os"
	"path/filepath"
)

var SupportedFileExtensions = []string{
	".md", ".pdf", ".docx", ".txt",
	".py", ".cs", ".js", ".ts",
	".json", ".yml", ".yaml",
}

type Config struct {
	DataDir string
}

func NewConfig() *Config {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	dataDir := filepath.ToSlash(filepath.Join(homeDir, ".zolam"))

	if _, exists := os.LookupEnv("ZOLAM_CHROMADB_DATA_DIR"); !exists {
		chromaDataDir := filepath.ToSlash(filepath.Join(homeDir, ".zolam", "chromadb"))
		os.Setenv("ZOLAM_CHROMADB_DATA_DIR", chromaDataDir)
	}

	return &Config{DataDir: dataDir}
}
