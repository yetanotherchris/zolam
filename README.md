# zolam

Ingest your personal files (PDF, Markdown, Docx, Txt, code) into ChromaDB for semantic search in Claude.

## Quick Start

**Prerequisites:** Docker and Docker Compose.

Install zolam, then:

```bash
# Start ChromaDB
zolam chromadb start

# Ingest your files
zolam ingest ~/notes ~/docs --extensions .md,.txt,.pdf

# Update only changed files (reads directories from config)
zolam update

# Register the MCP server so Claude can search your files
zolam mcp claude
```

### Installation

**Homebrew (macOS/Linux)**
```bash
brew install yetanotherchris/tap/zolam
```

**Winget (Windows)**
```powershell
winget install yetanotherchris.zolam
```

**Scoop (Windows)**
```powershell
scoop bucket add zolam https://github.com/yetanotherchris/zolam
scoop install zolam
```

## Commands

Run `zolam` with no arguments for the interactive TUI, or use CLI subcommands:

```bash
zolam ingest <dirs> [--extensions .md,.txt] [--collection name]  # Ingest files
zolam update [dirs]                                               # Re-ingest changed files
zolam stats                                                       # Collection info
zolam config                                                      # Show configuration
zolam chromadb start|stop|status                                  # Manage ChromaDB
zolam download --source gdrive:/path --dest ~/notes               # Download via rclone
zolam reset [--collection name]                                   # Reset a collection
zolam mcp claude                                                  # Register MCP server
```

## Configuration

Settings are loaded in this order (highest priority wins):

1. Defaults
2. `~/.zolam/config.json`
3. Environment variables
4. CLI flags

### config.json

Located at `~/.zolam/config.json`. Created automatically after your first ingest.

```json
{
  "collectionName": "my-notes",
  "rcloneSource": "",
  "rcloneConfigDir": "~/.config/rclone",
  "dataDir": "~/.zolam/chromadb-data",
  "directories": [
    {
      "path": "/home/user/notes",
      "extensions": [".md", ".txt"]
    }
  ]
}
```

The `directories` node stores previously ingested directories and their file extensions. This allows `zolam update` to run without arguments - it reads the directories from config.

### Environment Variables

| Variable | Default | Description |
|---|---|---|
| `COLLECTION_NAME` | `my-notes` | ChromaDB collection name |
| `RCLONE_SOURCE` | | Source path on remote |
| `RCLONE_CONFIG_DIR` | `~/.config/rclone` | rclone config directory |
| `ZOLAM_DATA_DIR` | `~/.zolam/chromadb-data` | Local ChromaDB data directory |

### Supported File Extensions

`.md`, `.pdf`, `.docx`, `.txt`, `.py`, `.cs`, `.js`, `.ts`, `.json`, `.yml`, `.yaml`

## Claude Integration

Register the chroma-mcp server to give Claude access to your ingested files:

```bash
zolam mcp claude
```

Make sure ChromaDB is running (`zolam chromadb start`) before using Claude with the MCP server.

## Building from Source

```bash
cd src
go build -o zolam ./cmd/zolam/
```

## Name

The name zolam comes from Midazolam, a sedative most people take for an endoscopy or other procedures. It induces sleepiness, decreases anxiety, and causes anterograde amnesia.
