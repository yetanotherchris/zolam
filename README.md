# zolam

Ingest your personal files (PDF, Markdown, Docx, Txt, code) into ChromaDB for semantic search in Claude.

## Quick Start

**Prerequisites:** Docker and Docker Compose.

```bash
# Start ChromaDB
zolam chromadb start

# Ingest files into a named collection
zolam ingest ~/notes --collection my-project --extensions .md,.txt

# Re-ingest only changed files (zolam-file-hashes.json is created in the current directory)
zolam update ~/notes --collection my-project

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
> Note: the winget package may lag behind the latest release, as each new version requires a PR to the winget-pkgs repository.

**Scoop (Windows)**
```powershell
scoop bucket add zolam https://github.com/yetanotherchris/zolam
scoop install zolam
```

## Commands

### chromadb

```bash
zolam chromadb start    # Start the ChromaDB container
zolam chromadb stop     # Stop the ChromaDB container
zolam chromadb status   # Check if ChromaDB is running
```

### ingest

Ingest files from one or more directories into a ChromaDB collection. Both `--collection` and `--extensions` are required.

```bash
zolam ingest <dirs...> --collection <name> --extensions <exts>

# Ingest markdown files
zolam ingest ~/notes --collection my-project --extensions .md

# Ingest PDFs
zolam ingest ~/books --collection reading-list --extensions .pdf

# Multiple directories and extensions
zolam ingest ~/notes ~/docs --collection research --extensions .md,.txt,.pdf

# Reset the collection before ingesting
zolam ingest ~/notes --collection my-project --extensions .md --reset
```

Subdirectories are scanned recursively.

### update

Re-ingest only files that have changed since the last run. Directories must be specified explicitly.

```bash
zolam update <dirs...> --collection <name>

zolam update ~/notes --collection my-project
```

### collections

```bash
zolam collections list              # List all collections
zolam collections remove <name>     # Delete a collection
```

### mcp

Register the [chroma-mcp](https://github.com/chroma-core/chroma-mcp) server with an AI provider.

```bash
zolam mcp claude    # Register with Claude Code
```

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `ZOLAM_CHROMADB_DATA_DIR` | `~/.zolam/chromadb` | ChromaDB persistent storage path |

## Supported File Extensions

`.md`, `.pdf`, `.docx`, `.txt`, `.py`, `.cs`, `.js`, `.ts`, `.json`, `.yml`, `.yaml`

## AI Integration

Make sure ChromaDB is running (`zolam chromadb start`) before starting your AI tool.

### Claude Code

```bash
zolam mcp claude
```

### OpenCode

```bash
opencode mcp add chroma --type local -- uvx chroma-mcp --client-type http --host localhost --port 8000 --ssl false
```

Or add manually to `~/.config/opencode/opencode.json` (global) or `./opencode.json` (project):

```json
{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "chroma": {
      "type": "local",
      "command": ["uvx", "chroma-mcp", "--client-type", "http", "--host", "localhost", "--port", "8000", "--ssl", "false"],
      "enabled": true
    }
  }
}
```

Both approaches require `uv` to be installed (`brew install uv` or `pip install uv`).

## Building from Source

```bash
cd src
go build -o zolam ./cmd/zolam/
```

## Name

The name zolam comes from Midazolam, a sedative most people take for an endoscopy or other procedures. It induces sleepiness, decreases anxiety, and causes anterograde amnesia.
