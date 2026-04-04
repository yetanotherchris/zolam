package domain

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	OpenRouterAPIKey   string
	OpenRouterModel    string
	CollectionName     string
	UseLocalEmbeddings bool
	RcloneRemote       string
	RcloneSource       string
	RcloneConfigDir    string
	DataDir            string
	Extensions         []string
	Directories        []string
}

var defaultExtensions = []string{
	".md", ".pdf", ".docx", ".txt",
	".py", ".cs", ".js", ".ts",
	".json", ".yml", ".yaml",
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

func getEnvOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

// LoadConfig loads configuration from a .env file (if present in the current
// directory) and environment variables. It returns the config, any validation
// warnings, and the first validation error (if any).
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

func LoadConfig() (*Config, []string, error) {
	loadEnvFile(".env")

	cfg := &Config{
		OpenRouterAPIKey:   os.Getenv("OPENROUTER_API_KEY"),
		OpenRouterModel:    getEnvOrDefault("OPENROUTER_MODEL", "openai/text-embedding-3-small"),
		CollectionName:     getEnvOrDefault("COLLECTION_NAME", "my-notes"),
		UseLocalEmbeddings: os.Getenv("USE_LOCAL_EMBEDDINGS") == "1",
		RcloneRemote:       getEnvOrDefault("RCLONE_REMOTE", "gdrive"),
		RcloneSource:       os.Getenv("RCLONE_SOURCE"),
		RcloneConfigDir:    getEnvOrDefault("RCLONE_CONFIG_DIR", defaultRcloneConfigDir()),
		DataDir:            getEnvOrDefault("ZOLAM_DATA_DIR", defaultDataDir()),
		Extensions:         append([]string{}, defaultExtensions...),
	}

	warnings, errs := cfg.Validate()
	var firstErr error
	if len(errs) > 0 {
		firstErr = errs[0]
	}

	os.Setenv("ZOLAM_DATA_DIR", cfg.DataDir)

	return cfg, warnings, firstErr
}

// MergeFlags overrides config values with CLI flag values. Only non-empty flag
// values are applied. Recognised keys: openrouter-api-key, openrouter-model,
// collection-name, use-local-embeddings, rclone-remote, rclone-source,
// data-dir, extensions, directories.
func (c *Config) MergeFlags(flags map[string]string) {
	if v, ok := flags["openrouter-api-key"]; ok && v != "" {
		c.OpenRouterAPIKey = v
	}
	if v, ok := flags["openrouter-model"]; ok && v != "" {
		c.OpenRouterModel = v
	}
	if v, ok := flags["collection-name"]; ok && v != "" {
		c.CollectionName = v
	}
	if v, ok := flags["use-local-embeddings"]; ok && v != "" {
		c.UseLocalEmbeddings = v == "1" || strings.EqualFold(v, "true")
	}
	if v, ok := flags["rclone-remote"]; ok && v != "" {
		c.RcloneRemote = v
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
	if v, ok := flags["extensions"]; ok && v != "" {
		c.Extensions = strings.Split(v, ",")
		for i := range c.Extensions {
			c.Extensions[i] = strings.TrimSpace(c.Extensions[i])
		}
	}
	if v, ok := flags["directories"]; ok && v != "" {
		c.Directories = strings.Split(v, ",")
		for i := range c.Directories {
			c.Directories[i] = strings.TrimSpace(c.Directories[i])
		}
	}
}

// Validate checks the config and returns warnings for missing optional values
// that fell back to defaults, and errors for invalid or missing required values.
func (c *Config) Validate() (warnings []string, errs []error) {
	// Required: API key unless local embeddings are used
	if c.OpenRouterAPIKey == "" && !c.UseLocalEmbeddings {
		errs = append(errs, fmt.Errorf("OPENROUTER_API_KEY is required when USE_LOCAL_EMBEDDINGS is not enabled"))
	}

	// Warnings for values that fell back to defaults
	if os.Getenv("OPENROUTER_MODEL") == "" {
		warnings = append(warnings, "OPENROUTER_MODEL not set, using default: openai/text-embedding-3-small")
	}
	if os.Getenv("COLLECTION_NAME") == "" {
		warnings = append(warnings, "COLLECTION_NAME not set, using default: my-notes")
	}
	if os.Getenv("RCLONE_REMOTE") == "" {
		warnings = append(warnings, "RCLONE_REMOTE not set, using default: gdrive")
	}
	if os.Getenv("RCLONE_CONFIG_DIR") == "" {
		warnings = append(warnings, "RCLONE_CONFIG_DIR not set, using default: "+defaultRcloneConfigDir())
	}
	if os.Getenv("ZOLAM_DATA_DIR") == "" {
		warnings = append(warnings, "ZOLAM_DATA_DIR not set, using default: "+defaultDataDir())
	}

	return warnings, errs
}
