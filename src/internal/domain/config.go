package domain

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// SupportedFileExtensions lists the file types that zolam can ingest.
var SupportedFileExtensions = []string{
	".md", ".pdf", ".docx", ".txt",
	".py", ".cs", ".js", ".ts",
	".json", ".yml", ".yaml",
}

// DirectoryEntry records a previously ingested directory and the file
// extensions that were used for that directory.
type DirectoryEntry struct {
	Path       string   `json:"path"`
	Extensions []string `json:"extensions"`
}

type Config struct {
	CollectionName  string
	RcloneSource    string
	RcloneConfigDir string
	DataDir         string
	Directories     []DirectoryEntry
}

// configJSON mirrors the on-disk config.json with camelCase keys.
type configJSON struct {
	CollectionName string           `json:"collectionName,omitempty"`
	RcloneSource   string           `json:"rcloneSource,omitempty"`
	RcloneConfigDir string          `json:"rcloneConfigDir,omitempty"`
	DataDir        string           `json:"dataDir,omitempty"`
	Directories    []DirectoryEntry `json:"directories,omitempty"`
}

// loadEnvFile reads a .env file from the given path and sets environment
// variables for any keys not already present in the environment.
func loadEnvFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, "=")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])

		// Remove surrounding quotes if present
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		// Only set if not already in environment
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, value)
		}
	}
}

func defaultDataDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "./chromadb-data"
	}
	return filepath.ToSlash(filepath.Join(homeDir, ".zolam", "chromadb-data"))
}

func defaultRcloneConfigDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ".config/rclone"
	}
	return filepath.ToSlash(filepath.Join(homeDir, ".config", "rclone"))
}

// ConfigPath returns the path to the config.json file (~/.zolam/config.json).
func ConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "config.json"
	}
	return filepath.Join(homeDir, ".zolam", "config.json")
}

// loadConfigJSON reads config.json from disk. Returns zero-value struct if
// the file does not exist.
func loadConfigJSON(path string) (configJSON, error) {
	var cj configJSON
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cj, nil
		}
		return cj, err
	}
	if err := json.Unmarshal(data, &cj); err != nil {
		return cj, err
	}
	return cj, nil
}

// LoadConfig loads configuration with the following precedence (highest wins):
//  1. Defaults
//  2. config.json (~/.zolam/config.json)
//  3. .env file (current directory)
//  4. Environment variables
//
// CLI flags are applied later via MergeFlags.
func LoadConfig() (*Config, []string, error) {
	// 1. Start with defaults
	cfg := &Config{
		CollectionName:  "my-notes",
		RcloneConfigDir: defaultRcloneConfigDir(),
		DataDir:         defaultDataDir(),
	}

	// 2. Overlay config.json
	cj, err := loadConfigJSON(ConfigPath())
	if err != nil {
		return nil, nil, err
	}
	if cj.CollectionName != "" {
		cfg.CollectionName = cj.CollectionName
	}
	if cj.RcloneSource != "" {
		cfg.RcloneSource = cj.RcloneSource
	}
	if cj.RcloneConfigDir != "" {
		cfg.RcloneConfigDir = cj.RcloneConfigDir
	}
	if cj.DataDir != "" {
		cfg.DataDir = cj.DataDir
	}
	if len(cj.Directories) > 0 {
		cfg.Directories = cj.Directories
	}

	// 3. Load .env file (sets env vars for keys not already present)
	loadEnvFile(".env")

	// 4. Env vars override everything except CLI flags
	if v := os.Getenv("COLLECTION_NAME"); v != "" {
		cfg.CollectionName = v
	}
	if v := os.Getenv("RCLONE_SOURCE"); v != "" {
		cfg.RcloneSource = v
	}
	if v := os.Getenv("RCLONE_CONFIG_DIR"); v != "" {
		cfg.RcloneConfigDir = v
	}
	if v := os.Getenv("ZOLAM_DATA_DIR"); v != "" {
		cfg.DataDir = v
	}

	warnings, errs := cfg.Validate()
	var firstErr error
	if len(errs) > 0 {
		firstErr = errs[0]
	}

	os.Setenv("ZOLAM_DATA_DIR", cfg.DataDir)

	return cfg, warnings, firstErr
}

// SaveConfig persists the current configuration to ~/.zolam/config.json.
func SaveConfig(cfg *Config) error {
	cj := configJSON{
		CollectionName:  cfg.CollectionName,
		RcloneSource:    cfg.RcloneSource,
		RcloneConfigDir: cfg.RcloneConfigDir,
		DataDir:         cfg.DataDir,
		Directories:     cfg.Directories,
	}

	data, err := json.MarshalIndent(cj, "", "  ")
	if err != nil {
		return err
	}

	path := ConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// MergeFlags overrides config values with CLI flag values. Only non-empty flag
// values are applied.
func (c *Config) MergeFlags(flags map[string]string) {
	if v, ok := flags["collection-name"]; ok && v != "" {
		c.CollectionName = v
	}
	if v, ok := flags["rclone-source"]; ok && v != "" {
		c.RcloneSource = v
	}
	if v, ok := flags["rclone-config-dir"]; ok && v != "" {
		c.RcloneConfigDir = filepath.ToSlash(v)
	}
	if v, ok := flags["data-dir"]; ok && v != "" {
		c.DataDir = v
		os.Setenv("ZOLAM_DATA_DIR", v)
	}
}

// AddOrUpdateDirectory adds or updates a directory entry in the config.
// If the directory already exists, its extensions are updated.
func (c *Config) AddOrUpdateDirectory(dir string, extensions []string) {
	for i, d := range c.Directories {
		if d.Path == dir {
			c.Directories[i].Extensions = extensions
			return
		}
	}
	c.Directories = append(c.Directories, DirectoryEntry{
		Path:       dir,
		Extensions: extensions,
	})
}

// Validate checks the config and returns warnings for missing optional values
// that fell back to defaults, and errors for invalid or missing required values.
func (c *Config) Validate() (warnings []string, errs []error) {
	if os.Getenv("COLLECTION_NAME") == "" {
		warnings = append(warnings, "COLLECTION_NAME not set, using default: my-notes")
	}
	if os.Getenv("RCLONE_CONFIG_DIR") == "" {
		warnings = append(warnings, "RCLONE_CONFIG_DIR not set, using default: "+defaultRcloneConfigDir())
	}
	if os.Getenv("ZOLAM_DATA_DIR") == "" {
		warnings = append(warnings, "ZOLAM_DATA_DIR not set, using default: "+defaultDataDir())
	}

	return warnings, errs
}
