package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"unicode"

	"github.com/spf13/cobra"
	"github.com/yetanotherchris/zolam/internal/zolam"
)

var version = "dev"

func main() {
	rootCmd := &cobra.Command{
		Use:   "zolam",
		Short: "Daemon-free semantic search over your personal files",
		Long: "A CLI tool that indexes files in the current directory into a flat-file\n" +
			"index (sqlite/jsonl) for semantic search via Claude Code/OpenCode, with no\n" +
			"background service required.\n\n" +
			"  zolam ingest <dirs...>   index files (creates the project, or refreshes it)\n" +
			"  zolam ingest update      re-sync using the directories already in project.json\n" +
			"  zolam query <text>       search the index",
		Version: version,
	}
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	cobra.EnableCommandSorting = false

	rootCmd.AddCommand(
		newIngestCmd(),
		newQueryCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
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
