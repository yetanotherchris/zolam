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
		Use:   "zolam",
		Short: "Daemon-free semantic search over your personal files",
		Long: "A CLI tool that indexes files in the current directory into a flat-file\n" +
			"index (sqlite/jsonl) for semantic search via Claude Code/OpenCode, with no\n" +
			"background service required.\n\n" +
			"  zolam ingest <dirs...>   index files (creates the project, or refreshes it)\n" +
			"  zolam ingest update      re-sync using the directories already in project.json\n" +
			"  zolam query <text>       search the index\n\n" +
			"The legacy Docker/ChromaDB workflow lives under 'zolam chromadb'.",
		Version: version,
	}
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	cobra.EnableCommandSorting = false

	rootCmd.AddCommand(
		newIngestCmd(),
		newQueryCmd(),
		newChromaDBCmd(),
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

// truncateForDisplay collapses whitespace and caps length for terminal
// query output; use --json for the untruncated text.
func truncateForDisplay(text string) string {
	text = strings.Join(strings.Fields(text), " ")
	const maxLen = 300
	if len(text) > maxLen {
		return text[:maxLen] + "..."
	}
	return text
}

func initServices() (*docker.DockerClient, error) {
	if _, exists := os.LookupEnv("ZOLAM_CHROMADB_DATA_DIR"); !exists {
		homeDir, _ := os.UserHomeDir()
		os.Setenv("ZOLAM_CHROMADB_DATA_DIR", homeDir+"/.zolam/chromadb")
	}

	dc, err := docker.NewDockerClient()
	if err != nil {
		return nil, fmt.Errorf("initializing docker: %w", err)
	}

	return dc, nil
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
	var backend string
	var reset bool

	cmd := &cobra.Command{
		Use:   "ingest <directories...>",
		Short: "Index files into the project (creates it, or refreshes it)",
		Long: "Index files into the current directory's flat-file project (sqlite by\n" +
			"default, or jsonl). Always name one or more subdirectories to scope\n" +
			"what gets indexed (pass '.' to index the current directory itself,\n" +
			"including dotfiles/dirs) — this applies on every run, not just the\n" +
			"first. Safe to re-run at any time: incremental behaviour (only\n" +
			"added/changed/removed files are reprocessed) comes from diffing\n" +
			"against the stored file hashes, not from omitting directories.\n\n" +
			"To re-sync without naming directories again, use 'zolam ingest update',\n" +
			"which reuses the directories already recorded in project.json.",
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			dirs, exts := splitArgsFromExtensions(args, extensions)

			result, proj, err := zolam.RunSync(zolam.SyncOptions{
				Dirs:       dirs,
				Extensions: exts,
				Backend:    backend,
				Reset:      reset,
			}, func(line string) {
				fmt.Println(line)
			})
			if err != nil {
				return err
			}
			fmt.Printf("\nIngest complete (%s backend): %d new, %d changed, %d removed, %d unchanged\n",
				proj.Backend, result.Added, result.Changed, result.Removed, result.Unchanged)
			return nil
		},
	}

	cmd.Flags().StringSliceVar(&extensions, "extensions", nil, "File extensions to include (default: all supported types)")
	cmd.Flags().StringVar(&backend, "backend", "", "Index backend: sqlite (default) or jsonl")
	cmd.Flags().BoolVar(&reset, "reset", false, "Delete the local index and re-ingest from scratch")
	cmd.AddCommand(newIngestUpdateCmd())

	return cmd
}

func newIngestUpdateCmd() *cobra.Command {
	var reset bool

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Re-sync using the directories already recorded in project.json",
		Long: "Re-scans the source directories already stored in this project's\n" +
			"project.json — no directory argument needed — and reprocesses only\n" +
			"added, changed, or removed files. Fails if no project exists yet;\n" +
			"first-time ingest still requires 'zolam ingest <dir>'.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, proj, err := zolam.RunUpdate("", reset, func(line string) {
				fmt.Println(line)
			})
			if err != nil {
				return err
			}
			fmt.Printf("\nIngest complete (%s backend): %d new, %d changed, %d removed, %d unchanged\n",
				proj.Backend, result.Added, result.Changed, result.Removed, result.Unchanged)
			return nil
		},
	}

	cmd.Flags().BoolVar(&reset, "reset", false, "Delete the local index and re-ingest from scratch")

	return cmd
}

func newQueryCmd() *cobra.Command {
	var topK int
	var keyword bool
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "query <text>",
		Short: "Search the current directory's index",
		Long:  "Semantic (default) or keyword (--keyword) search against the current directory's sqlite/jsonl index.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			proj, projectDir, err := zolam.LoadProject("")
			if err != nil {
				return err
			}

			hits, err := zolam.RunQuery(proj, projectDir, args[0], topK, keyword)
			if err != nil {
				return err
			}

			if jsonOut {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(hits)
			}

			if len(hits) == 0 {
				fmt.Println("No results.")
				return nil
			}
			for i, hit := range hits {
				loc := fmt.Sprintf("chunk %d", hit.Chunk)
				if hit.Page != nil {
					loc = fmt.Sprintf("page %d, chunk %d", *hit.Page, hit.Chunk)
				}
				if hit.Score != nil {
					fmt.Printf("%d. [%.2f] %s  (%s)\n", i+1, *hit.Score, hit.Path, loc)
				} else {
					fmt.Printf("%d. %s  (%s)\n", i+1, hit.Path, loc)
				}
				fmt.Printf("   %s\n\n", truncateForDisplay(hit.Text))
			}
			return nil
		},
	}

	cmd.Flags().IntVar(&topK, "top-k", 5, "Number of results to return")
	cmd.Flags().BoolVar(&keyword, "keyword", false, "Keyword search instead of semantic search")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output machine-readable JSON")

	return cmd
}

// newChromaDBCmd groups the legacy Docker/ChromaDB workflow: the container
// itself, its collections, and MCP registration. The default
// ingest/query flat-file workflow above does not need any of this.
func newChromaDBCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chromadb",
		Short: "Legacy Docker/ChromaDB workflow (start/stop, collections, mcp)",
		Long: "Manage the legacy Docker-based ChromaDB workflow: the container itself,\n" +
			"its collections, and MCP registration. 'zolam ingest'/'zolam query' do not\n" +
			"use any of this.",
	}

	cmd.AddCommand(
		newChromaDBStartCmd(),
		newChromaDBStopCmd(),
		newChromaDBStatusCmd(),
		newChromaDBCollectionsCmd(),
		newChromaDBMcpCmd(),
	)
	return cmd
}

func newChromaDBStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start the ChromaDB container",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			dc, err := initServices()
			if err != nil {
				return err
			}
			fmt.Println("Starting ChromaDB...")
			if err := dc.StartChromaDB(); err != nil {
				return err
			}
			if err := dc.WaitForChromaDB(30 * time.Second); err != nil {
				return err
			}
			fmt.Println("ChromaDB is running.")
			return nil
		},
	}
}

func newChromaDBStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the ChromaDB container",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			dc, err := initServices()
			if err != nil {
				return err
			}
			fmt.Println("Stopping ChromaDB...")
			if err := dc.StopChromaDB(); err != nil {
				return err
			}
			fmt.Println("ChromaDB stopped.")
			return nil
		},
	}
}

func newChromaDBStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check whether the ChromaDB container is running",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			dc, err := initServices()
			if err != nil {
				return err
			}
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
		},
	}
}

func newChromaDBCollectionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "collections",
		Short: "Manage ChromaDB collections",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all ChromaDB collections",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			dc, err := initServices()
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
			dc, err := initServices()
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

	removeFileCmd := &cobra.Command{
		Use:   "remove-file <collection> <file>",
		Short: "Remove all chunks for a file from a collection",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			collection, fileName := args[0], args[1]
			dc, err := initServices()
			if err != nil {
				return err
			}
			if err := requireChromaDB(dc); err != nil {
				return err
			}
			n, err := dc.DeleteFile(collection, fileName)
			if err != nil {
				return err
			}
			store, err := zolam.OpenHashStore(collection)
			if err != nil {
				return err
			}
			defer store.Close()
			if err := store.DeleteFile(fileName); err != nil {
				return err
			}
			fmt.Printf("Removed %d chunk(s) for %q from collection %q.\n", n, fileName, collection)
			return nil
		},
	}

	cmd.AddCommand(listCmd, removeCmd, removeFileCmd)
	return cmd
}

func newChromaDBMcpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp <provider>",
		Short: "Register chroma-mcp server with an AI provider",
		Long: "Register the chroma-mcp MCP server with an AI provider. Supported providers: claude, opencode.\n" +
			"For the default sqlite/jsonl workflow, use 'zolam init claude|opencode' instead.",
		Args: cobra.ExactArgs(1),
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
