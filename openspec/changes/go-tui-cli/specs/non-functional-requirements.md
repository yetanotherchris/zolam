# Non-Functional Requirements: Go TUI CLI

## NFR-1: Build and Distribution

- Single binary with no runtime dependencies (except Docker)
- Build targets: `linux-amd64`, `linux-arm64`, `darwin-amd64`, `darwin-arm64`, `windows-amd64`
- Binary size target: < 15MB
- GitHub Releases with auto-generated release notes
- Homebrew formula (macOS + Linux) in `Formula/ingester.rb`
- Scoop manifest (Windows) in `ingester.json`
- Version stamped at build time via `-ldflags`

## NFR-2: Performance

- TUI startup time: < 500ms
- Docker command execution: stream output in real-time (no buffering)
- Hash computation for update-only mode: process files concurrently

## NFR-3: Error Handling

- All Docker errors surfaced clearly to the user
- Graceful handling of Docker not being installed
- Graceful handling of Docker daemon not running
- Timeout on ChromaDB health checks (30 seconds default)
- Non-zero exit codes for failures in CLI mode

## NFR-4: Compatibility

- Go 1.22+ required for build
- Docker Engine 20.10+ and Docker Compose V2 required at runtime
- Works on macOS, Linux, and Windows (PowerShell/CMD/WSL)

## NFR-5: Project Structure

Follow clean architecture in `/src`:

```
src/
├── cmd/
│   └── ingester/
│       └── main.go              # Entry point
├── internal/
│   ├── domain/
│   │   ├── config.go            # Configuration types
│   │   └── manifest.go          # Hash manifest types
│   ├── docker/
│   │   ├── client.go            # Docker/compose command execution
│   │   ├── chromadb.go          # ChromaDB container management
│   │   └── rclone.go            # rclone Docker operations
│   ├── ingester/
│   │   ├── ingester.go          # Ingest orchestration
│   │   └── hasher.go            # File hashing for updates
│   └── tui/
│       ├── app.go               # Bubbletea main model
│       ├── menu.go              # Main menu view
│       ├── ingest.go            # Ingest view
│       ├── progress.go          # Progress display
│       └── styles.go            # Lipgloss styles
├── go.mod
├── go.sum
└── .goreleaser.yml              # (optional) GoReleaser config
```
