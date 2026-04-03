# ingester

Ingest your personal files into ChromaDB for semantic search in Claude.

## Installation

### Homebrew (macOS/Linux)

```bash
brew install yetanotherchris/tap/ingester
```

### Winget (Windows)

```powershell
winget install yetanotherchris.zolam
```

### Scoop (Windows)

```powershell
scoop bucket add ingester https://github.com/yetanotherchris/ingester
scoop install ingester
```

### Binary Download

Download the latest binary from [GitHub Releases](https://github.com/yetanotherchris/ingester/releases) for your platform.

### Build from Source

```bash
cd src
go build -o ingester ./cmd/ingester/
```

## Prerequisites

- Docker and Docker Compose
- An `OPENROUTER_API_KEY` (for embeddings), or set `USE_LOCAL_EMBEDDINGS=1` for offline embeddings

## Claude Code Integration (chroma-mcp)

```bash
pip install uv
claude mcp add chroma -- uvx chroma-mcp --client-type persistent --data-dir /path/to/chromadb/data
```

This gives Claude access to `chroma_query_documents`, `chroma_list_collections`, and other Chroma tools for semantic search over your ingested files.

## Setup

Add your API key to a `.env` file:

```bash
# .env
OPENROUTER_API_KEY=sk-or-...
```

## Usage

### TUI (Interactive Mode)

Run `ingester` with no arguments to launch the interactive TUI:

```bash
ingester
```

The TUI provides a menu-driven interface for all operations: ingest, update, download, stats, reset, ChromaDB management, and settings.

### CLI (Non-Interactive Mode)

All operations are available as CLI subcommands for scripting:

```bash
# Ingest directories
ingester ingest ~/notes ~/docs --extensions .md,.txt --collection my-docs

# Update only changed files
ingester update ~/notes ~/docs

# Download from Google Drive via rclone
ingester download --remote gdrive --source Documents/notes --dest ~/notes

# Show collection statistics
ingester stats

# Reset a collection
ingester reset --collection my-docs

# Manage ChromaDB
ingester chromadb start
ingester chromadb stop
ingester chromadb status

# Show current configuration
ingester config
```

### Legacy Scripts

The original wrapper scripts (`ingest.sh`, `ingest.ps1`) are still available for backwards compatibility:

```bash
./ingest.sh ~/notes
./ingest.ps1 ~/notes
```

## Configuration

All directories are ingested into a single collection. Configure via environment variables or a `.env` file:

| Variable | Default | Description |
|---|---|---|
| `COLLECTION_NAME` | `my-notes` | ChromaDB collection name |
| `OPENROUTER_API_KEY` | | API key for OpenRouter embeddings |
| `OPENROUTER_MODEL` | `openai/text-embedding-3-small` | Embedding model |
| `USE_LOCAL_EMBEDDINGS` | | Set to `1` for offline sentence-transformers |
| `RCLONE_REMOTE` | `gdrive` | rclone remote name |
| `RCLONE_SOURCE` | | Source path on remote |
| `INGESTER_DATA_DIR` | `./chromadb-data` | Local ChromaDB data directory |

Environment variables can also be passed as CLI flags (flags take precedence).
