package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yetanotherchris/zolam/internal/docker"
	"github.com/yetanotherchris/zolam/internal/domain"
	"github.com/yetanotherchris/zolam/internal/zolam"
	"github.com/yetanotherchris/zolam/internal/tui"
)

var version = "dev"

func main() {
	rootCmd := &cobra.Command{
		Use:     "zolam",
		Short:   "Semantic search file ingester for ChromaDB",
		Long:    "A TUI and CLI tool for ingesting files into ChromaDB for semantic search via Claude.",
		Version: version,
		RunE:    runTUI,
	}

	rootCmd.AddCommand(
		newIngestCmd(),
		newUpdateCmd(),
		newDownloadCmd(),
		newStatsCmd(),
		newResetCmd(),
		newChromaDBCmd(),
		newConfigCmd(),
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

func initServices() (*domain.Config, *docker.DockerClient, *zolam.Ingester, []string, error) {
	cfg, warnings, err := domain.LoadConfig()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("loading config: %w", err)
	}

	dc, err := docker.NewDockerClient()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("initializing docker: %w", err)
	}

	ing := zolam.NewIngester(dc, cfg)
	return cfg, dc, ing, warnings, nil
}

func runTUI(cmd *cobra.Command, args []string) error {
	cfg, dc, ing, warnings, err := initServices()
	if err != nil {
		return err
	}

	app := tui.NewApp(cfg, dc, ing, warnings)
	p := tea.NewProgram(app, tea.WithAltScreen())
	app.Sender().Program = p
	_, err = p.Run()
	return err
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
			cfg, dc, ing, _, err := initServices()
			if err != nil {
				return err
			}

			if collection != "" {
				cfg.CollectionName = collection
			}

			opts := zolam.IngestOptions{
				CollectionName: cfg.CollectionName,
				Reset:          reset,
			}
			if len(extensions) > 0 {
				opts.Extensions = extensions
			} else {
				opts.Extensions = domain.SupportedFileExtensions
			}

			if err := requireChromaDB(dc); err != nil {
				return err
			}

			if err := ing.Run(args, opts, func(line string) {
				fmt.Println(line)
			}); err != nil {
				return err
			}

			// Save ingested directories and extensions to config.json
			for _, dir := range args {
				absPath, absErr := filepath.Abs(dir)
				if absErr != nil {
					absPath = dir
				}
				cfg.AddOrUpdateDirectory(filepath.ToSlash(absPath), opts.Extensions)
			}
			if err := domain.SaveConfig(cfg); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not save config: %v\n", err)
			}

			return nil
		},
	}

	cmd.Flags().StringSliceVar(&extensions, "extensions", nil, "File extensions to include (e.g. .md,.txt)")
	cmd.Flags().StringVar(&collection, "collection", "", "ChromaDB collection name")
	cmd.Flags().BoolVar(&reset, "reset", false, "Reset collection before ingesting")

	return cmd
}

func newUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update [directories...]",
		Short: "Re-ingest only changed files",
		Long:  "Scan directories and only re-ingest files whose content has changed since last run.\nIf no directories are given, reads from ~/.zolam/config.json.",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, dc, ing, _, err := initServices()
			if err != nil {
				return err
			}

			if err := requireChromaDB(dc); err != nil {
				return err
			}

			dirs := args
			if len(dirs) == 0 {
				if len(cfg.Directories) == 0 {
					return fmt.Errorf("no directories specified and none found in config.json.\nRun 'zolam ingest <dir>' first, or pass directories as arguments")
				}
				for _, d := range cfg.Directories {
					dirs = append(dirs, d.Path)
				}
				fmt.Printf("Using %d directory(ies) from config.json\n", len(dirs))
			}

			result, err := ing.RunUpdateOnly(dirs, func(line string) {
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
}

func newDownloadCmd() *cobra.Command {
	var source string
	var dest string
	var configDir string

	cmd := &cobra.Command{
		Use:   "download",
		Short: "Download files from Google Drive via rclone",
		Long:  "Use rclone Docker container to download files from a configured remote.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, dc, _, _, err := initServices()
			if err != nil {
				return err
			}

			if source == "" {
				source = cfg.RcloneSource
			}
			if dest == "" {
				dest = cfg.DataDir
			}
			if configDir == "" {
				configDir = cfg.RcloneConfigDir
			} else {
				configDir = filepath.ToSlash(configDir)
			}

			if source == "" {
				return fmt.Errorf("RCLONE_SOURCE is required (--source flag or RCLONE_SOURCE env var)")
			}

			rcCmd, err := dc.RcloneCopy(source, dest, configDir)
			if err != nil {
				return err
			}

			return dc.StreamOutput(rcCmd, os.Stdout)
		},
	}

	cmd.Flags().StringVar(&source, "source", "", "rclone source (e.g. gdrive:/path/to/folder)")
	cmd.Flags().StringVar(&dest, "dest", "", "Local destination directory")
	cmd.Flags().StringVar(&configDir, "config-dir", "", "rclone config directory (default: ~/.config/rclone)")

	return cmd
}

func newStatsCmd() *cobra.Command {
	var collection string

	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show collection statistics",
		Long:  "Display information about the ChromaDB collection and service status.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, dc, ing, _, err := initServices()
			if err != nil {
				return err
			}

			if collection != "" {
				cfg.CollectionName = collection
			}

			if err := requireChromaDB(dc); err != nil {
				return err
			}

			_, err = ing.GetStats(func(line string) {
				fmt.Println(line)
			})
			if err != nil {
				return err
			}

			fmt.Printf("\nCollection: %s\n", cfg.CollectionName)
			fmt.Printf("ChromaDB:   running\n")
			fmt.Printf("Supported extensions: %s\n", strings.Join(domain.SupportedFileExtensions, ", "))

			if len(cfg.Directories) > 0 {
				fmt.Println("\nIngested directories:")
				for _, d := range cfg.Directories {
					fmt.Printf("  %s (%s)\n", d.Path, strings.Join(d.Extensions, ", "))
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&collection, "collection", "", "ChromaDB collection name")
	return cmd
}

func newResetCmd() *cobra.Command {
	var collection string

	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Delete and recreate a ChromaDB collection",
		Long:  "Reset the specified ChromaDB collection by deleting and recreating it.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, dc, ing, _, err := initServices()
			if err != nil {
				return err
			}

			if collection != "" {
				cfg.CollectionName = collection
			}

			if err := requireChromaDB(dc); err != nil {
				return err
			}

			return ing.Run(nil, zolam.IngestOptions{
				CollectionName: cfg.CollectionName,
				Reset:          true,
				Stats:          true,
			}, func(line string) {
				fmt.Println(line)
			})
		},
	}

	cmd.Flags().StringVar(&collection, "collection", "", "ChromaDB collection name")
	return cmd
}

func newChromaDBCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chromadb [start|stop|status]",
		Short: "Manage the ChromaDB container",
		Long:  "Start, stop, or check the status of the ChromaDB Docker container.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, dc, _, _, err := initServices()
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

func newMcpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp <provider>",
		Short: "Register chroma-mcp server with an AI provider",
		Long:  "Register the chroma-mcp MCP server with an AI provider. Currently supports: claude.",
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
			default:
				return fmt.Errorf("unsupported provider %q, supported: claude", provider)
			}
		},
	}
}

func newConfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Show current configuration",
		Long:  "Display the current configuration settings.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, _, warnings, err := initServices()
			if err != nil {
				return err
			}

			fmt.Println("Current Configuration:")
			fmt.Println("─────────────────────")
			fmt.Printf("Config file:         %s\n", domain.ConfigPath())
			fmt.Printf("Collection Name:     %s\n", cfg.CollectionName)
			fmt.Printf("Data Directory:      %s\n", cfg.DataDir)
			fmt.Printf("rclone Source:       %s\n", cfg.RcloneSource)
			fmt.Printf("rclone Config Dir:   %s\n", cfg.RcloneConfigDir)

			if len(cfg.Directories) > 0 {
				fmt.Println("\nIngested directories:")
				for _, d := range cfg.Directories {
					fmt.Printf("  %s (%s)\n", d.Path, strings.Join(d.Extensions, ", "))
				}
			}

			if len(warnings) > 0 {
				fmt.Println("\nWarnings:")
				for _, w := range warnings {
					fmt.Printf("  ! %s\n", w)
				}
			}

			return nil
		},
	}
}
