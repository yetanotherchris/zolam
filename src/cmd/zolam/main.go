package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/spf13/cobra"
	"github.com/yetanotherchris/zolam/internal/docker"
	"github.com/yetanotherchris/zolam/internal/domain"
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
		Long: "A CLI tool that ingests files into a per-project flat-file index (duckdb/jsonl) for\n" +
			"semantic search via Claude Code/OpenCode, with no background service required.\n" +
			"The legacy ChromaDB/Docker/MCP workflow (--backend chroma, 'zolam chromadb', 'zolam mcp')\n" +
			"is still supported but deprecated.",
		Version: version,
	}
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	rootCmd.AddCommand(
		newIngestCmd(),
		newUpdateCmd(),
		newQueryCmd(),
		newProjectsCmd(),
		newInitCmd(),
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

// printDeprecationNotice warns on stderr that a command belongs to the
// legacy ChromaDB/Docker/MCP workflow, without interrupting its output.
func printDeprecationNotice(oldCmd, newCmd string) {
	fmt.Fprintf(os.Stderr, "Note: %q is part of the deprecated ChromaDB/Docker workflow. See %q for the v3 daemon-free workflow.\n\n", oldCmd, newCmd)
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

// runLegacyIngest runs the pre-v3 Docker/ChromaDB ingest pipeline unchanged.
func runLegacyIngest(dirs []string, collection string, exts []string, reset bool) error {
	printDeprecationNotice("--backend chroma", "zolam ingest --backend duckdb (default) or jsonl")

	dc, ing, err := initServices()
	if err != nil {
		return err
	}
	if err := requireChromaDB(dc); err != nil {
		return err
	}
	result, err := ing.RunSync(dirs, collection, exts, reset, func(line string) {
		fmt.Println(line)
	})
	if err != nil {
		return err
	}
	fmt.Printf("\nSync complete: %d new, %d changed, %d removed, %d unchanged\n",
		result.Added, result.Changed, result.Removed, result.Unchanged)
	return nil
}

func newIngestCmd() *cobra.Command {
	var extensions []string
	var project string
	var collection string
	var backend string
	var reset bool

	cmd := &cobra.Command{
		Use:   "ingest [directories...]",
		Short: "Ingest files into a project's search index",
		Long: "Ingest files from one or more directories into a project index.\n" +
			"Defaults to the daemon-free duckdb backend (no Docker required);\n" +
			"pass --backend chroma for the legacy Docker/ChromaDB pipeline.",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := project
			if name == "" {
				name = collection
			}
			if name == "" {
				return fmt.Errorf("--project is required")
			}

			dirs, exts := splitArgsFromExtensions(args, extensions)
			if len(dirs) == 0 {
				return fmt.Errorf("no directories specified")
			}

			if backend == "chroma" {
				return runLegacyIngest(dirs, name, exts, reset)
			}

			result, proj, err := zolam.RunV3Sync(zolam.V3SyncOptions{
				ProjectName: name,
				Dirs:        dirs,
				Extensions:  exts,
				Backend:     backend,
				Reset:       reset,
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

	cmd.Flags().StringSliceVar(&extensions, "extensions", nil, "File extensions to include (e.g. .md,.txt)")
	cmd.Flags().StringVar(&project, "project", "", "Project name")
	cmd.Flags().StringVar(&collection, "collection", "", "Deprecated alias for --project")
	cmd.Flags().MarkHidden("collection")
	cmd.Flags().StringVar(&backend, "backend", "", "Index backend: duckdb (default), jsonl, or chroma (legacy)")
	cmd.Flags().BoolVar(&reset, "reset", false, "Delete the project and re-ingest from scratch")
	cmd.MarkFlagRequired("extensions")

	return cmd
}

func newUpdateCmd() *cobra.Command {
	var project string
	var collection string
	var backend string

	cmd := &cobra.Command{
		Use:   "update [directories...]",
		Short: "Re-ingest only files that have changed",
		Long: "Re-ingest only files whose content has changed since the last ingest/update.\n" +
			"For duckdb/jsonl projects, directories default to those recorded in project.json.",
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			name := project
			if name == "" {
				name = collection
			}
			if name == "" {
				return fmt.Errorf("--project is required")
			}

			projectDir, err := domain.ProjectDir(name)
			if err != nil {
				return err
			}

			// Legacy chroma collections never had a project.json. Treat a
			// never-before-seen name as the pre-v3 workflow unless a v3
			// backend was explicitly requested.
			if backend == "chroma" || (!domain.Exists(projectDir) && backend == "") {
				if len(args) == 0 {
					return fmt.Errorf("directories are required for the legacy chroma backend")
				}
				return runLegacyIngest(args, name, nil, false)
			}

			result, proj, err := zolam.RunV3Sync(zolam.V3SyncOptions{
				ProjectName: name,
				Dirs:        args,
				Backend:     backend,
			}, func(line string) {
				fmt.Println(line)
			})
			if err != nil {
				return err
			}

			fmt.Printf("\nUpdate complete (%s backend): %d new, %d changed, %d removed, %d unchanged\n",
				proj.Backend, result.Added, result.Changed, result.Removed, result.Unchanged)
			return nil
		},
	}

	cmd.Flags().StringVar(&project, "project", "", "Project name")
	cmd.Flags().StringVar(&collection, "collection", "", "Deprecated alias for --project")
	cmd.Flags().MarkHidden("collection")
	cmd.Flags().StringVar(&backend, "backend", "", "Force a backend (duckdb, jsonl, or chroma); normally inferred from the existing project")

	return cmd
}

func newQueryCmd() *cobra.Command {
	var project string
	var topK int
	var keyword bool
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "query <text>",
		Short: "Search a project's flat-file index",
		Long:  "Semantic (default) or keyword (--keyword) search against a duckdb/jsonl project index.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if project == "" {
				return fmt.Errorf("--project is required")
			}

			proj, projectDir, err := zolam.LoadV3Project(project)
			if err != nil {
				return err
			}

			resp, err := zolam.RunQuery(proj, projectDir, args[0], topK, keyword, func(line string) {
				fmt.Fprintln(os.Stderr, line)
			})
			if err != nil {
				return err
			}

			if jsonOut {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(resp)
			}

			if len(resp.Results) == 0 {
				fmt.Println("No results.")
				return nil
			}
			for i, hit := range resp.Results {
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

	cmd.Flags().StringVar(&project, "project", "", "Project name")
	cmd.Flags().IntVar(&topK, "top-k", 5, "Number of results to return")
	cmd.Flags().BoolVar(&keyword, "keyword", false, "Keyword search instead of semantic search")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output machine-readable JSON")

	return cmd
}

func newProjectsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "projects",
		Short: "Manage v3 flat-file projects (duckdb/jsonl)",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			names, err := domain.ListProjectNames()
			if err != nil {
				return err
			}
			if len(names) == 0 {
				fmt.Println("No projects found. Run 'zolam ingest <dirs> --project <name>' to create one.")
				return nil
			}
			sort.Strings(names)
			for _, name := range names {
				dir, err := domain.ProjectDir(name)
				if err != nil {
					return err
				}
				proj, err := domain.Load(dir)
				if err != nil {
					fmt.Printf("%s\t(error reading project.json: %v)\n", name, err)
					continue
				}
				hashes, _ := zolam.LoadFileHashes(dir)
				fmt.Printf("%-20s backend=%-8s files=%-6d model=%s last_ingest=%s\n",
					name, proj.Backend, len(hashes), proj.EmbeddingModel, proj.LastIngest.Format("2006-01-02 15:04"))
			}
			return nil
		},
	}

	removeCmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Delete a project (index, sidecars, and metadata)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := domain.ProjectDir(args[0])
			if err != nil {
				return err
			}
			if !domain.Exists(dir) {
				return fmt.Errorf("no project named %q", args[0])
			}
			if err := domain.Remove(dir); err != nil {
				return err
			}
			fmt.Printf("Removed project %q.\n", args[0])
			return nil
		},
	}

	cmd.AddCommand(listCmd, removeCmd)
	return cmd
}

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init <claude|opencode>",
		Short: "Install AI-tool integration for zolam v3 (skill/instructions, no MCP)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "claude":
				path, err := zolam.WriteClaudeSkill()
				if err != nil {
					return err
				}
				fmt.Printf("Installed skill: %s\n\n", path)
				fmt.Println("Suggested addition to a repo's CLAUDE.md:")
				fmt.Println()
				fmt.Print(zolam.ClaudeSkillSnippet)
				return nil
			case "opencode":
				path, err := zolam.WriteOpencodeInstructions()
				if err != nil {
					return err
				}
				fmt.Printf("Installed instructions: %s\n", path)
				fmt.Println("Verify this matches OpenCode's current custom-instructions convention; it may change between releases.")
				return nil
			default:
				return fmt.Errorf("unsupported target %q, supported: claude, opencode", args[0])
			}
		},
	}
}

func newChromaDBCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chromadb [start|stop|status]",
		Short: "Manage the ChromaDB container",
		Long:  "Start, stop, or check the status of the ChromaDB Docker container.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			printDeprecationNotice("zolam chromadb", "zolam ingest/update/query (duckdb or jsonl backend)")

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
		Short: "Manage ChromaDB collections (deprecated, see 'zolam projects')",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			printDeprecationNotice("zolam collections", "zolam projects")
			return nil
		},
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

	removeFileCmd := &cobra.Command{
		Use:   "remove-file <collection> <file>",
		Short: "Remove all chunks for a file from a collection",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			collection, fileName := args[0], args[1]
			dc, _, err := initServices()
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

func newMcpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp <provider>",
		Short: "Register chroma-mcp server with an AI provider",
		Long:  "Register the chroma-mcp MCP server with an AI provider. Supported providers: claude, opencode.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			printDeprecationNotice("zolam mcp", "zolam init claude/opencode")

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

