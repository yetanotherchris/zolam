# Implementation Tasks: Go TUI CLI

## Phase 1: Project Scaffolding

- [ ] 1.1 Initialize Go module in `/src` with `go mod init github.com/yetanotherchris/ingester`
- [ ] 1.2 Create directory structure: `cmd/ingester/`, `internal/domain/`, `internal/docker/`, `internal/ingester/`, `internal/tui/`
- [ ] 1.3 Add core dependencies: cobra, bubbletea, lipgloss, bubbles
- [ ] 1.4 Create `cmd/ingester/main.go` entry point with version flag and ldflags
- [ ] 1.5 Create root cobra command with TUI as default (no subcommand) and CLI subcommands

## Phase 2: Configuration & Domain

- [ ] 2.1 Implement `internal/domain/config.go` - Config struct, env var loading, .env file support, CLI flag merging
- [ ] 2.2 Implement config validation (warn on missing optional vars, error on required vars)
- [ ] 2.3 Implement `internal/domain/manifest.go` - HashManifest type for update-only mode

## Phase 3: Docker Integration

- [ ] 3.1 Implement `internal/docker/client.go` - Docker/compose command wrapper with streaming output
- [ ] 3.2 Embed `docker-compose.yml` via `go:embed`, write to `~/.ingester/` on first use
- [ ] 3.3 Implement `internal/docker/chromadb.go` - Start, stop, status, health check, wait-for-ready
- [ ] 3.4 Implement `internal/docker/rclone.go` - rclone Docker run for Google Drive sync

## Phase 4: Ingester Logic

- [ ] 4.1 Implement `internal/ingester/ingester.go` - Full ingest pipeline (directory resolution, volume mounts, docker compose run)
- [ ] 4.2 Implement `internal/ingester/hasher.go` - SHA-256 file hashing, manifest diff, update-only logic
- [ ] 4.3 Implement stats retrieval (collection info via docker exec or direct ChromaDB API)

## Phase 5: TUI

- [ ] 5.1 Implement `internal/tui/styles.go` - Lipgloss color scheme and layout styles
- [ ] 5.2 Implement `internal/tui/menu.go` - Main menu model with arrow key navigation
- [ ] 5.3 Implement `internal/tui/ingest.go` - Ingest configuration view (directory input, extension selection)
- [ ] 5.4 Implement `internal/tui/progress.go` - Docker output streaming view with real-time updates
- [ ] 5.5 Implement `internal/tui/app.go` - Root model that switches between views

## Phase 6: CLI Subcommands

- [ ] 6.1 Implement `ingest` subcommand (non-interactive equivalent of TUI ingest)
- [ ] 6.2 Implement `update` subcommand (update-only mode)
- [ ] 6.3 Implement `download` subcommand (rclone Google Drive)
- [ ] 6.4 Implement `stats` subcommand
- [ ] 6.5 Implement `reset` subcommand
- [ ] 6.6 Implement `chromadb` subcommand (start/stop/status)
- [ ] 6.7 Implement `config` subcommand (show current settings)

## Phase 7: CI/CD & Distribution

- [ ] 7.1 Create `.github/workflows/build-release.yml` - Matrix build for 5 platforms, release on tag
- [ ] 7.2 Create `Formula/ingester.rb` - Homebrew formula template
- [ ] 7.3 Create `ingester.json` - Scoop manifest template
- [ ] 7.4 Create `updatebrew.ps1` - Script to update Homebrew formula with version/hashes
- [ ] 7.5 Create `updatescoop.ps1` - Script to update Scoop manifest with version/hashes
- [ ] 7.6 Add auto-commit step to release workflow for manifest updates

## Phase 8: Testing & Documentation

- [ ] 8.1 Add unit tests for config loading and validation
- [ ] 8.2 Add unit tests for file hasher and manifest diffing
- [ ] 8.3 Update README.md with new installation instructions (Brew, Scoop, binary download)
- [ ] 8.4 Update README.md with new CLI usage examples
- [ ] 8.5 Add `--help` text for all subcommands
