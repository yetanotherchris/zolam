# Design: Go TUI CLI

## Architecture Overview

```
┌─────────────────────────────────────────────┐
│                   CLI/TUI                    │
│  (cobra commands + bubbletea interactive)    │
├─────────────────────────────────────────────┤
│               Orchestration                  │
│  (ingester, hasher, config)                  │
├─────────────────────────────────────────────┤
│              Infrastructure                  │
│  (docker client, compose, rclone)            │
├─────────────────────────────────────────────┤
│           External Systems                   │
│  (Docker Engine, ChromaDB, rclone, GDrive)   │
└─────────────────────────────────────────────┘
```

## Component Design

### 1. Configuration (`internal/domain/config.go`)

```go
type Config struct {
    OpenRouterAPIKey   string
    OpenRouterModel    string
    CollectionName     string
    UseLocalEmbeddings bool
    RcloneRemote       string
    RcloneSource       string
    DataDir            string
    Extensions         []string
    Directories        []string
}
```

- Loaded from: environment variables → `.env` file → CLI flags (highest precedence)
- Validation method that returns warnings and errors

### 2. Docker Client (`internal/docker/client.go`)

Wraps `os/exec` calls to Docker:

```go
type DockerClient struct {
    composePath string  // path to docker-compose.yml
}

func (c *DockerClient) ComposeUp(services ...string) error
func (c *DockerClient) ComposeDown() error
func (c *DockerClient) ComposeRun(service string, args ...string) error
func (c *DockerClient) RunContainer(image string, args ...string) error
func (c *DockerClient) IsContainerRunning(name string) (bool, error)
func (c *DockerClient) StreamOutput(cmd *exec.Cmd) error
```

- All commands stream stdout/stderr in real-time
- Compose file is embedded via `go:embed` and written to `~/.ingester/` on first run

### 3. ChromaDB Manager (`internal/docker/chromadb.go`)

```go
func (c *DockerClient) StartChromaDB() error
func (c *DockerClient) StopChromaDB() error  
func (c *DockerClient) ChromaDBStatus() (bool, error)
func (c *DockerClient) WaitForChromaDB(timeout time.Duration) error
```

- Health check: HTTP GET to `http://localhost:8000/api/v1/heartbeat`
- Auto-start before ingestion if not running

### 4. Ingester (`internal/ingester/ingester.go`)

Orchestrates the full ingest pipeline:

```go
func (i *Ingester) Run(directories []string, opts IngestOptions) error
func (i *Ingester) RunUpdateOnly(directories []string) (*UpdateResult, error)
```

- Converts relative paths to absolute
- Builds volume mount arguments
- Delegates to DockerClient for actual execution

### 5. File Hasher (`internal/ingester/hasher.go`)

For update-only mode:

```go
type HashManifest struct {
    Files    map[string]string  // filepath -> sha256 hash
    Updated  time.Time
}

func ComputeHash(filepath string) (string, error)
func LoadManifest(path string) (*HashManifest, error)
func SaveManifest(manifest *HashManifest, path string) error
func DiffManifest(old, new *HashManifest) (added, changed, removed []string)
```

- Manifest stored as JSON at `.ingester-hashes.json` in the data directory
- SHA-256 of file contents

### 6. rclone Integration (`internal/docker/rclone.go`)

```go
func (c *DockerClient) RcloneSync(remote, source, dest string) error
```

- Uses `rclone/rclone` Docker image
- Mounts `~/.config/rclone` for config and destination directory for data
- Streams progress output

### 7. TUI (`internal/tui/`)

Built with bubbletea (Elm architecture):

- **app.go**: Root model, manages view switching
- **menu.go**: Main menu with arrow key navigation
- **ingest.go**: Ingest configuration view (select directories, extensions)
- **progress.go**: Shows Docker output streaming during operations
- **styles.go**: Lipgloss color/layout definitions

Menu flow:
```
Main Menu → Select Action → Configure (if needed) → Execute → Show Result → Back to Menu
```

### 8. Docker Compose File

Embedded in the binary via `go:embed`:

```yaml
services:
  chromadb:
    image: chromadb/chroma:latest
    ports:
      - "8000:8000"
    volumes:
      - ${INGESTER_DATA_DIR:-./chromadb-data}:/chroma/chroma
    environment:
      - IS_PERSISTENT=TRUE
      - ANONYMIZED_TELEMETRY=FALSE
  
  ingest:
    image: ghcr.io/yetanotherchris/ingester:latest
    volumes:
      - ${INGESTER_DATA_DIR:-./chromadb-data}:/data
    environment:
      - COLLECTION_NAME=${COLLECTION_NAME:-my-notes}
      - OPENROUTER_API_KEY=${OPENROUTER_API_KEY:-}
      - OPENROUTER_MODEL=${OPENROUTER_MODEL:-openai/text-embedding-3-small}
      - USE_LOCAL_EMBEDDINGS=${USE_LOCAL_EMBEDDINGS:-}
    depends_on:
      - chromadb
    profiles:
      - ingest
```

## CI/CD Design

### GitHub Actions Workflow: `build-release.yml`

**Trigger**: Push of tag `v*.*.*` or `workflow_dispatch`

**Jobs**:

1. **build** (matrix: linux-amd64, linux-arm64, darwin-amd64, darwin-arm64, windows-amd64)
   - Checkout with full history
   - Setup Go 1.22+
   - Determine version from git tag
   - `go build` with ldflags: `-X main.version=$VERSION`
   - Upload artifact

2. **release** (depends on build, only on tag push)
   - Download all artifacts
   - Create GitHub Release with binaries attached
   - Update Scoop manifest (`ingester.json`)
   - Update Homebrew formula (`Formula/ingester.rb`)
   - Auto-commit manifest updates to `main`

### Homebrew Formula Pattern (from tinycity)

```ruby
class Ingester < Formula
  desc "Semantic search file ingester for ChromaDB"
  homepage "https://github.com/yetanotherchris/ingester"
  version "VERSION"
  
  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/.../ingester-vVERSION-darwin-arm64.tar.gz"
      sha256 "HASH"
    else
      url "https://github.com/.../ingester-vVERSION-darwin-amd64.tar.gz"
      sha256 "HASH"
    end
  end
  
  on_linux do
    url "https://github.com/.../ingester-vVERSION-linux-amd64.tar.gz"
    sha256 "HASH"
  end

  def install
    bin.install "ingester"
  end
end
```

### Scoop Manifest Pattern (from tinycity)

```json
{
  "homepage": "https://github.com/yetanotherchris/ingester",
  "version": "VERSION",
  "architecture": {
    "64bit": {
      "url": "https://github.com/.../ingester-vVERSION-windows-amd64.exe",
      "hash": "HASH",
      "bin": ["ingester.exe"]
    }
  }
}
```
