# Implementation Tasks: Go TUI CLI

## Phase 1: Project Scaffolding

- [x] 1.1 Initialize Go module in `/src` with `go mod init github.com/yetanotherchris/ingester`
- [x] 1.2 Create directory structure: `cmd/ingester/`, `internal/domain/`, `internal/docker/`, `internal/ingester/`, `internal/tui/`
- [x] 1.3 Add core dependencies: cobra, bubbletea, lipgloss, bubbles
- [x] 1.4 Create `cmd/ingester/main.go` entry point with version flag and ldflags
- [x] 1.5 Create root cobra command with TUI as default (no subcommand) and CLI subcommands

## Phase 2: Configuration & Domain

- [x] 2.1 Implement `internal/domain/config.go` - Config struct, env var loading, .env file support, CLI flag merging
- [x] 2.2 Implement config validation (warn on missing optional vars, error on required vars)
- [x] 2.3 Implement `internal/domain/manifest.go` - HashManifest type for update-only mode

## Phase 3: Docker Integration

- [x] 3.1 Implement `internal/docker/client.go` - Docker/compose command wrapper with streaming output
- [x] 3.2 Embed `docker-compose.yml` via `go:embed`, write to `~/.ingester/` on first use
- [x] 3.3 Implement `internal/docker/chromadb.go` - Start, stop, status, health check, wait-for-ready
- [x] 3.4 Implement `internal/docker/rclone.go` - rclone Docker run for Google Drive sync

## Phase 4: Ingester Logic

- [x] 4.1 Implement `internal/ingester/ingester.go` - Full ingest pipeline (directory resolution, volume mounts, docker compose run)
- [x] 4.2 Implement `internal/ingester/hasher.go` - SHA-256 file hashing, manifest diff, update-only logic
- [x] 4.3 Implement stats retrieval (collection info via docker exec or direct ChromaDB API)

## Phase 5: TUI

- [x] 5.1 Implement `internal/tui/styles.go` - Lipgloss color scheme and layout styles
- [x] 5.2 Implement `internal/tui/menu.go` - Main menu model with arrow key navigation
- [x] 5.3 Implement `internal/tui/ingest.go` - Ingest configuration view (directory input, extension selection)
- [x] 5.4 Implement `internal/tui/progress.go` - Docker output streaming view with real-time updates
- [x] 5.5 Implement `internal/tui/app.go` - Root model that switches between views

## Phase 6: CLI Subcommands

- [x] 6.1 Implement `ingest` subcommand (non-interactive equivalent of TUI ingest)
- [x] 6.2 Implement `update` subcommand (update-only mode)
- [x] 6.3 Implement `download` subcommand (rclone Google Drive)
- [x] 6.4 Implement `stats` subcommand
- [x] 6.5 Implement `reset` subcommand
- [x] 6.6 Implement `chromadb` subcommand (start/stop/status)
- [x] 6.7 Implement `config` subcommand (show current settings)

## Phase 7: CI/CD & Distribution

- [x] 7.1 Create `.github/workflows/build-release.yml` - Matrix build for 5 platforms, release on tag
- [x] 7.2 Create `Formula/ingester.rb` - Homebrew formula template
- [x] 7.3 Create `ingester.json` - Scoop manifest template
- [x] 7.4 Create `updatebrew.ps1` - Script to update Homebrew formula with version/hashes
- [x] 7.5 Create `updatescoop.ps1` - Script to update Scoop manifest with version/hashes
- [x] 7.6 Add auto-commit step to release workflow for manifest updates

## Phase 8: Testing & Documentation

- [x] 8.1 Add unit tests for config loading and validation
- [x] 8.2 Add unit tests for file hasher and manifest diffing
- [x] 8.3 Update README.md with new installation instructions (Brew, Scoop, binary download)
- [x] 8.4 Update README.md with new CLI usage examples
- [x] 8.5 Add `--help` text for all subcommands
