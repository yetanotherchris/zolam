# Design: `zolam mcp` command

## Changes

### CLI (`src/cmd/zolam/main.go`)
- Add `newMcpCmd()` function returning a `*cobra.Command`
- Use: `mcp <provider>`, Args: `cobra.ExactArgs(1)`
- Short: "Register chroma-mcp server with an AI provider"
- Switch on the provider argument:
  - `claude`: executes `claude mcp add --scope user chroma -- uvx chroma-mcp --client-type http --host localhost --port 8000 --ssl false` via `os/exec`
  - Default: returns error "unsupported provider %q, supported: claude"
- Streams stdout/stderr to the terminal so the user sees the CLI output
- Register it in `rootCmd.AddCommand()`

### README
- Update the Claude Code Integration section to mention `zolam mcp claude` as the preferred setup method
- Keep the manual command as a fallback reference

### No changes needed
- No new packages required (`os/exec` is stdlib)
- No Docker or config changes
- No TUI changes
