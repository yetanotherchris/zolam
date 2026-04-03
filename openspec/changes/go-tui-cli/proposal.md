# Proposal: Go TUI CLI to Replace Bash/PowerShell Scripts

## Summary

Replace the existing `ingest.sh` and `ingest.ps1` wrapper scripts with a single cross-platform Go CLI application that provides a terminal user interface (TUI) for managing the ingester pipeline. The Go binary will be distributed via GitHub Releases, Homebrew, and Scoop.

## Motivation

- **Cross-platform**: The current approach requires maintaining two separate scripts (bash + PowerShell) with duplicated logic. A single Go binary works on all platforms.
- **Better UX**: A TUI provides interactive menus, progress indicators, and status displays rather than raw CLI flags.
- **Distribution**: Native binaries can be installed via package managers (Brew, Scoop) instead of requiring users to clone the repo.
- **Extended functionality**: Adding new features (rclone integration, update-only mode, environment validation) is cleaner in Go than in shell scripts.

## What's Changing

1. **New Go application** in `/src` directory using clean architecture
2. **GitHub Actions workflow** for building and releasing multi-platform binaries
3. **Homebrew formula** and **Scoop manifest** for package manager distribution
4. **rclone integration** for Google Drive downloading via Docker
5. **Update-only mode** using file content hashing to detect changes
6. **Environment validation** with warnings for missing env vars

## What's NOT Changing

- The Python `ingest.py` script (still runs inside Docker)
- The Docker image build/publish workflow
- The ChromaDB persistent storage model
- The existing bash/PowerShell scripts (kept for backwards compatibility initially)

## Key Decisions

- **TUI framework**: Use [bubbletea](https://github.com/charmbracelet/bubbletea) + [lipgloss](https://github.com/charmbracelet/lipgloss) for the terminal UI (industry standard for Go TUIs)
- **CLI framework**: Use [cobra](https://github.com/spf13/cobra) for subcommand structure with TUI as the default mode
- **Docker interaction**: Shell out to `docker` and `docker compose` commands (assumes Docker is installed)
- **Architecture**: Follow clean architecture pattern (cmd/, internal/domain/, internal/infrastructure/, internal/presentation/)
- **Versioning**: Use git tags + GoReleaser or custom GHA workflow (following tinycity pattern)
