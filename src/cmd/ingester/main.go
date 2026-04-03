package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yetanotherchris/ingester/internal/docker"
	"github.com/yetanotherchris/ingester/internal/domain"
	"github.com/yetanotherchris/ingester/internal/ingester"
	"github.com/yetanotherchris/ingester/internal/tui"
)

var version = "dev"

func main() {
	rootCmd := &cobra.Command{
		Use:     "ingester",
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

	return fmt.Errorf("ChromaDB is not running. Start it first with: ingester chromadb start")
}

func initServices() (*domain.Config, *docker.DockerClient, *ingester.Ingester, []string, error) {
	cfg, warnings, err := domain.LoadConfig()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("loading config: %w", err)
	}

	dc, err := docker.NewDockerClient()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("initializing docker: %w", err)
	}

	ing := ingester.NewIngester(dc, cfg)
	return cfg, dc, ing, warnings, nil
}

func runTUI(cmd *cobra.Command, args []string) error {
	cfg, dc, ing, warnings, err := initServices()
	if err != nil {
		return err
	}

	app := tui.NewApp(cfg, dc, ing, warnings)
	p := tea.NewProgram(app, tea.WithAltScreen())
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

			opts := ingester.IngestOptions{
				CollectionName: cfg.CollectionName,
				Reset:          reset,
			}
			if len(extensions) > 0 {
				opts.Extensions = extensions
			} else {
				opts.Extensions = cfg.Extensions
			}

			if err := requireChromaDB(dc); err != nil {
				return err
			}

			return ing.Run(args, opts, func(line string) {
				fmt.Println(line)
			})
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
		Long:  "Scan directories and only re-ingest files whose content has changed since last run.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, dc, ing, _, err := initServices()
			if err != nil {
				return err
			}
			_ = cfg

			if err := requireChromaDB(dc); err != nil {
				return err
			}

			result, err := ing.RunUpdateOnly(args, func(line string) {
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
	var remote string
	var source string
	var dest string

	cmd := &cobra.Command{
		Use:   "download",
		Short: "Download files from Google Drive via rclone",
		Long:  "Use rclone Docker container to download files from a configured remote.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, dc, _, _, err := initServices()
			if err != nil {
				return err
			}

			if remote == "" {
				remote = cfg.RcloneRemote
			}
			if source == "" {
				source = cfg.RcloneSource
			}
			if dest == "" {
				dest = cfg.DataDir
			}

			if source == "" {
				return fmt.Errorf("rclone source path is required (--source or RCLONE_SOURCE env var)")
			}

			rcCmd, err := dc.RcloneSync(remote, source, dest)
			if err != nil {
				return err
			}

			return dc.StreamOutput(rcCmd, os.Stdout)
		},
	}

	cmd.Flags().StringVar(&remote, "remote", "", "rclone remote name (default: gdrive)")
	cmd.Flags().StringVar(&source, "source", "", "Source path on remote")
	cmd.Flags().StringVar(&dest, "dest", "", "Local destination directory")

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

			stats, err := ing.GetStats(func(line string) {
				fmt.Println(line)
			})
			if err != nil {
				return err
			}

			fmt.Printf("\nCollection: %s\n", cfg.CollectionName)
			fmt.Printf("ChromaDB:   running\n")
			fmt.Printf("Embeddings: %s\n", stats.EmbeddingType)
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

			return ing.Run(nil, ingester.IngestOptions{
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

func newConfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Show current configuration",
		Long:  "Display the current configuration settings from environment variables and .env file.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, _, warnings, err := initServices()
			if err != nil {
				return err
			}

			fmt.Println("Current Configuration:")
			fmt.Println("─────────────────────")
			fmt.Printf("Collection Name:     %s\n", cfg.CollectionName)
			fmt.Printf("Data Directory:      %s\n", cfg.DataDir)
			fmt.Printf("Local Embeddings:    %v\n", cfg.UseLocalEmbeddings)
			if !cfg.UseLocalEmbeddings {
				fmt.Printf("OpenRouter Model:    %s\n", cfg.OpenRouterModel)
				if cfg.OpenRouterAPIKey != "" {
					fmt.Printf("OpenRouter API Key:  %s...%s\n", cfg.OpenRouterAPIKey[:4], cfg.OpenRouterAPIKey[len(cfg.OpenRouterAPIKey)-4:])
				} else {
					fmt.Printf("OpenRouter API Key:  (not set)\n")
				}
			}
			fmt.Printf("rclone Remote:       %s\n", cfg.RcloneRemote)
			fmt.Printf("rclone Source:       %s\n", cfg.RcloneSource)
			fmt.Printf("Extensions:          %v\n", cfg.Extensions)

			if len(warnings) > 0 {
				fmt.Println("\nWarnings:")
				for _, w := range warnings {
					fmt.Printf("  ⚠ %s\n", w)
				}
			}

			return nil
		},
	}
}
