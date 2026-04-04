# zolam

Ingest your personal files (PDF, Markdown, Docx, Txt) into ChromaDB for semantic search in Claude.

## Installation

### Homebrew (macOS/Linux)

```bash
brew install yetanotherchris/tap/zolam
```

### Winget (Windows)

```powershell
winget install yetanotherchris.zolam
```

If you don't have Winget, you can also use [scoop](https://www.scoop.sh):

```powershell
scoop bucket add zolam https://github.com/yetanotherchris/zolam
scoop install zolam
```

## Setup

### Prerequisites

- Docker and Docker Compose

Once you have run zolam and ingested files, you can enable its database, ChromaDb, inside Claude (see below).

## Usage

Run `zolam` with no arguments to launch the interactive TUI, or use CLI subcommands:

```
$ zolam --help

A TUI and CLI tool for ingesting files into ChromaDB for semantic search via Claude.

Usage:
  zolam [flags]
  zolam [command]

Available Commands:
  chromadb    Manage the ChromaDB container
  config      Show current configuration
  download    Download files from Google Drive via rclone
  ingest      Run the full ingestion pipeline
  reset       Delete and recreate a ChromaDB collection
  stats       Show collection statistics
  update      Re-ingest only changed files

Flags:
  -h, --help      help for zolam
  -v, --version   version for zolam
```

### Examples

```bash
# Ingest directories
zolam ingest ~/notes ~/docs --extensions .md,.txt --collection my-docs

# Update only changed files
zolam update ~/notes ~/docs

# Download from Google Drive via rclone (uses rclone copy)
zolam download --remote gdrive --source Documents/notes --dest ~/notes

# Show collection statistics
zolam stats

# Reset a collection
zolam reset --collection my-docs

# Manage ChromaDB
zolam chromadb start
zolam chromadb stop
zolam chromadb status

# Show current configuration
zolam config
```

### rclone

The `download` command uses `rclone copy` via a Docker container to sync files from a configured remote (e.g. Google Drive). You need an rclone config at `~/.config/rclone/` with your remote configured.

## Advanced

### Configuration

All directories are ingested into a single collection. Configure via environment variables:

| Variable | Default | Description |
|---|---|---|
| `COLLECTION_NAME` | `my-notes` | ChromaDB collection name |
| `RCLONE_REMOTE` | `gdrive` | rclone remote name |
| `RCLONE_SOURCE` | | Source path on remote |
| `ZOLAM_DATA_DIR` | `./chromadb-data` | Local ChromaDB data directory |

Environment variables can also be passed as CLI flags (flags take precedence).

## Claude Code Integration (chroma-mcp)

Once setup is complete, you can install the MCP server to give Claude access to your ingested files:

```bash
pip install uv
claude mcp add --scope user chroma -- uvx chroma-mcp --client-type http --host localhost --port 8000 --ssl false
```

This connects to the running ChromaDB container and gives Claude access to `chroma_query_documents`, `chroma_list_collections`, and other Chroma tools for semantic search. Make sure ChromaDB is running (`zolam chromadb start`) before using Claude with the MCP server.

### Binary Download

*Download the latest binary from [GitHub Releases](https://github.com/yetanotherchris/zolam/releases) for your platform.*

### Build from Source

```bash
cd src
go build -o zolam ./cmd/zolam/
```

### Name
The name zolam comes from Midazolam, a sedative most people take for an endoscopy or other procedures. It induces sleepiness, decreases anxiety, and causes anterograde amnesia.