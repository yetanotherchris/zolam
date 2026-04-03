# ingester

Ingest your personal files into ChromaDB for semantic search in Claude.

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

```bash
mkdir -p chromadb-data
```

Add your API key to a `.env` file (docker compose picks this up automatically):

```bash
# .env
OPENROUTER_API_KEY=sk-or-...
```

## Usage

The repository includes wrapper scripts that handle the `docker compose` commands for you. They automatically start ChromaDB, set up volume mounts, and pass arguments through to the ingester.

**Bash:**

```bash
# Ingest a single directory
./ingest.sh ~/notes

# Ingest multiple directories
./ingest.sh ~/notes ~/docs

# Filter by extension
./ingest.sh --extensions .md,.txt ~/docs

# Check what's indexed
./ingest.sh --stats

# Wipe collection and re-ingest
./ingest.sh --reset ~/notes

# Custom collection name
./ingest.sh --collection "work docs" ~/docs
```

**PowerShell:**

```powershell
# Ingest a single directory
./ingest.ps1 ~/notes

# Ingest multiple directories
./ingest.ps1 ~/notes ~/docs

# Filter by extension
./ingest.ps1 -Extensions .md,.txt ~/docs

# Check what's indexed
./ingest.ps1 -Stats

# Wipe collection and re-ingest
./ingest.ps1 -Reset ~/notes

# Custom collection name
./ingest.ps1 -Collection "work docs" ~/docs
```

Run `./ingest.sh --help` or `./ingest.ps1 -Help` for full usage details.

## Configuration

All directories are ingested into a single collection. Configure via environment variables:

| Variable | Default | Description |
|---|---|---|
| `COLLECTION_NAME` | `my notes` | ChromaDB collection name |
| `OPENROUTER_API_KEY` | | API key for OpenRouter embeddings |
| `OPENROUTER_MODEL` | `openai/text-embedding-3-small` | Embedding model |
| `USE_LOCAL_EMBEDDINGS` | | Set to `1` for offline sentence-transformers |

Set `COLLECTION_NAME` in your `.env` file or pass it with `--collection` / `-Collection`:

```bash
./ingest.sh --collection "work docs" ~/docs
```
