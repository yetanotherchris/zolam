package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/spf13/cobra"
	"github.com/yetanotherchris/zolam/internal/docker"
	"github.com/yetanotherchris/zolam/internal/zolam"
)

var version = "dev"

func registerOpencodeMCP() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".config", "opencode", "opencode.jsonc")

	var config map[string]any
	data, err := os.ReadFile(configPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading opencode config: %w", err)
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("parsing opencode config at %s: %w\nIf the file contains JSONC comments, add the mcp entry manually:\n  \"mcp\": { \"chroma\": { \"type\": \"local\", \"command\": [\"uvx\", \"chroma-mcp\", \"--client-type\", \"http\", \"--host\", \"localhost\", \"--port\", \"8000\", \"--ssl\", \"false\"] } }", configPath, err)
		}
	} else {
		config = make(map[string]any)
	}

	mcp, _ := config["mcp"].(map[string]any)
	if mcp == nil {
		mcp = make(map[string]any)
	}
	mcp["chroma"] = map[string]any{
		"type":    "local",
		"command": []string{"uvx", "chroma-mcp", "--client-type", "http", "--host", "localhost", "--port", "8000", "--ssl", "false"},
	}
	config["mcp"] = mcp

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("creating opencode config directory: %w", err)
	}

	out, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("serializing opencode config: %w", err)
	}

	tmpPath := configPath + ".tmp"
	if err := os.WriteFile(tmpPath, out, 0644); err != nil {
		return fmt.Errorf("writing opencode config: %w", err)
	}
	if err := os.Rename(tmpPath, configPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("finalizing opencode config: %w", err)
	}

	fmt.Printf("Registered chroma-mcp in %s\n", configPath)
	return nil
}

func main() {
	rootCmd := &cobra.Command{
		Use:     "zolam",
		Short:   "Semantic search file ingester for ChromaDB",
		Long:    "A CLI tool for ingesting files into ChromaDB for semantic search via Claude.",
		Version: version,
	}
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	rootCmd.AddCommand(
		newIngestCmd(),
		newUpdateCmd(),
		newChromaDBCmd(),
		newCollectionsCmd(),
		newMcpCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func requireChromaDB(dc *docker.DockerClient) error {
	running, _ := dc.ChromaDBStatus()
	if running {
		return nil
	}
	return fmt.Errorf("ChromaDB is not running. Start it first with: zolam chromadb start")
}

func initServices() (*docker.DockerClient, *zolam.Ingester, error) {
	if _, exists := os.LookupEnv("ZOLAM_CHROMADB_DATA_DIR"); !exists {
		homeDir, _ := os.UserHomeDir()
		os.Setenv("ZOLAM_CHROMADB_DATA_DIR", homeDir+"/.zolam/chromadb")
	}

	dc, err := docker.NewDockerClient()
	if err != nil {
		return nil, nil, fmt.Errorf("initializing docker: %w", err)
	}

	ing := zolam.NewIngester(dc)
	return dc, ing, nil
}

// looksLikeExtension returns true for tokens like ".md", ".csv,", ".html" —
// i.e. a dot followed only by letters/digits, with an optional trailing comma.
func looksLikeExtension(s string) bool {
	s = strings.TrimRight(s, ",")
	if len(s) < 2 || s[0] != '.' {
		return false
	}
	for _, c := range s[1:] {
		if !unicode.IsLetter(c) && !unicode.IsDigit(c) {
			return false
		}
	}
	return true
}

// normaliseExtension cleans a raw extension token: trims spaces and commas,
// then ensures it has a leading dot.
func normaliseExtension(s string) string {
	s = strings.TrimRight(strings.TrimSpace(s), ",")
	if s == "" {
		return ""
	}
	if s[0] != '.' {
		s = "." + s
	}
	return s
}

// splitArgsFromExtensions separates positional args that are actually extension
// tokens (e.g. ".md," from a space-after-comma invocation) from real directories.
func splitArgsFromExtensions(args []string, extensions []string) (dirs []string, exts []string) {
	exts = make([]string, 0, len(extensions))
	for _, e := range extensions {
		if t := normaliseExtension(e); t != "" {
			exts = append(exts, t)
		}
	}
	for _, a := range args {
		if looksLikeExtension(a) {
			exts = append(exts, normaliseExtension(a))
		} else {
			dirs = append(dirs, a)
		}
	}
	return dirs, exts
}

func newIngestCmd() *cobra.Command {
	var extensions []string
	var collection string
	var reset bool

	cmd := &cobra.Command{
		Use:   "ingest [directories...]",
		Short: "Run the full ingestion pipeline",
		Long:  "Ingest files from specified directories into ChromaDB for semantic search.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dc, ing, err := initServices()
			if err != nil {
				return err
			}

			dirs, exts := splitArgsFromExtensions(args, extensions)
			if len(dirs) == 0 {
				return fmt.Errorf("no directories specified")
			}

			opts := zolam.IngestOptions{
				CollectionName: collection,
				Reset:          reset,
				Extensions:     exts,
			}

			if err := requireChromaDB(dc); err != nil {
				return err
			}

			return ing.Run(dirs, opts, func(line string) {
				fmt.Println(line)
			})
		},
	}

	cmd.Flags().StringSliceVar(&extensions, "extensions", nil, "File extensions to include (e.g. .md,.txt)")
	cmd.Flags().StringVar(&collection, "collection", "", "ChromaDB collection name")
	cmd.Flags().BoolVar(&reset, "reset", false, "Reset collection before ingesting")
	cmd.MarkFlagRequired("collection")
	cmd.MarkFlagRequired("extensions")

	return cmd
}

func newUpdateCmd() *cobra.Command {
	var collection string

	cmd := &cobra.Command{
		Use:   "update <directories...>",
		Short: "Re-ingest only changed files",
		Long:  "Scan directories and re-ingest only files whose content has changed since last run.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dc, ing, err := initServices()
			if err != nil {
				return err
			}

			if err := requireChromaDB(dc); err != nil {
				return err
			}

			result, err := ing.RunUpdateOnly(args, collection, func(line string) {
				fmt.Println(line)
			})
			if err != nil {
				return err
			}

			fmt.Printf("\nUpdate complete: %d new, %d changed, %d removed, %d unchanged\n",
				result.Added, result.Changed, result.Removed, result.Unchanged)
			return nil
		},
	}

	cmd.Flags().StringVar(&collection, "collection", "", "ChromaDB collection name")
	cmd.MarkFlagRequired("collection")

	return cmd
}

func newChromaDBCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chromadb [start|stop|status]",
		Short: "Manage the ChromaDB container",
		Long:  "Start, stop, or check the status of the ChromaDB Docker container.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dc, _, err := initServices()
			if err != nil {
				return err
			}

			switch args[0] {
			case "start":
				fmt.Println("Starting ChromaDB...")
				if err := dc.StartChromaDB(); err != nil {
					return err
				}
				if err := dc.WaitForChromaDB(30 * time.Second); err != nil {
					return err
				}
				fmt.Println("ChromaDB is running.")
				return nil

			case "stop":
				fmt.Println("Stopping ChromaDB...")
				if err := dc.StopChromaDB(); err != nil {
					return err
				}
				fmt.Println("ChromaDB stopped.")
				return nil

			case "status":
				running, err := dc.ChromaDBStatus()
				if err != nil {
					return err
				}
				if running {
					fmt.Println("ChromaDB is running.")
				} else {
					fmt.Println("ChromaDB is not running.")
				}
				return nil

			default:
				return fmt.Errorf("unknown action %q: use start, stop, or status", args[0])
			}
		},
	}

	return cmd
}

func newCollectionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "collections",
		Short: "Manage ChromaDB collections",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all ChromaDB collections",
		RunE: func(cmd *cobra.Command, args []string) error {
			dc, _, err := initServices()
			if err != nil {
				return err
			}
			if err := requireChromaDB(dc); err != nil {
				return err
			}
			cols, err := dc.ListCollections()
			if err != nil {
				return err
			}
			if len(cols) == 0 {
				fmt.Println("No collections found.")
				return nil
			}
			for _, col := range cols {
				fmt.Println(col.Name)
			}
			return nil
		},
	}

	removeCmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a ChromaDB collection by name",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dc, _, err := initServices()
			if err != nil {
				return err
			}
			if err := requireChromaDB(dc); err != nil {
				return err
			}
			if err := dc.RemoveCollection(args[0]); err != nil {
				return err
			}
			fmt.Printf("Removed collection %q.\n", args[0])
			return nil
		},
	}

	cmd.AddCommand(listCmd, removeCmd)
	return cmd
}

func newMcpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp <provider>",
		Short: "Register chroma-mcp server with an AI provider",
		Long:  "Register the chroma-mcp MCP server with an AI provider. Supported providers: claude, opencode.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			provider := args[0]
			switch provider {
			case "claude":
				c := exec.Command("claude", "mcp", "add", "--scope", "user", "chroma", "--",
					"uvx", "chroma-mcp", "--client-type", "http", "--host", "localhost", "--port", "8000", "--ssl", "false")
				c.Stdin = os.Stdin
				c.Stdout = os.Stdout
				c.Stderr = os.Stderr
				if err := c.Run(); err != nil {
					if errors.Is(err, exec.ErrNotFound) {
						return fmt.Errorf("claude CLI is not installed or not on PATH")
					}
					return err
				}
				return nil
			case "opencode":
				return registerOpencodeMCP()
			default:
				return fmt.Errorf("unsupported provider %q, supported: claude, opencode", provider)
			}
		},
	}
}

