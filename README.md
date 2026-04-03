# ingester

Ingest your personal files into ChromaDB for semantic search in Claude.

## Prerequisites

- Docker and Docker Compose
- An `OPENROUTER_API_KEY` (for embeddings), or set `USE_LOCAL_EMBEDDINGS=1` for offline embeddings

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

Start ChromaDB, then ingest directories with `--directory`:

```bash
docker compose up -d chromadb
```

**Bash:**

```bash
# Ingest a single directory
docker compose --profile ingest run --rm \
  -v /path/to/notes:/sources/notes \
  ingest --directory /sources/notes

# Ingest multiple directories
docker compose --profile ingest run --rm \
  -v /path/to/notes:/sources/notes \
  -v /path/to/docs:/sources/docs \
  ingest --directory /sources/notes /sources/docs

# Filter by extension
docker compose --profile ingest run --rm \
  -v /path/to/docs:/sources/docs \
  ingest --directory /sources/docs --extensions .md .txt

# Check what's indexed
docker compose --profile ingest run --rm ingest --stats

# Wipe collection and re-ingest
docker compose --profile ingest run --rm \
  -v /path/to/notes:/sources/notes \
  ingest --reset --directory /sources/notes
```

**PowerShell:**

```powershell
# Ingest a single directory
docker compose --profile ingest run --rm `
  -v /path/to/notes:/sources/notes `
  ingest --directory /sources/notes

# Ingest multiple directories
docker compose --profile ingest run --rm `
  -v /path/to/notes:/sources/notes `
  -v /path/to/docs:/sources/docs `
  ingest --directory /sources/notes /sources/docs

# Filter by extension
docker compose --profile ingest run --rm `
  -v /path/to/docs:/sources/docs `
  ingest --directory /sources/docs --extensions .md .txt

# Check what's indexed
docker compose --profile ingest run --rm ingest --stats

# Wipe collection and re-ingest
docker compose --profile ingest run --rm `
  -v /path/to/notes:/sources/notes `
  ingest --reset --directory /sources/notes
```

## Configuration

All directories are ingested into a single collection. Configure via environment variables:

| Variable | Default | Description |
|---|---|---|
| `COLLECTION_NAME` | `my notes` | ChromaDB collection name |
| `OPENROUTER_API_KEY` | | API key for OpenRouter embeddings |
| `OPENROUTER_MODEL` | `openai/text-embedding-3-small` | Embedding model |
| `USE_LOCAL_EMBEDDINGS` | | Set to `1` for offline sentence-transformers |

Set `COLLECTION_NAME` in your `.env` file or pass it with `-e`:

```bash
docker compose --profile ingest run --rm \
  -e COLLECTION_NAME="work docs" \
  -v /path/to/docs:/sources/docs \
  ingest --directory /sources/docs
```

## Claude Code Integration (chroma-mcp)

```bash
pip install uv
claude mcp add chroma -- uvx chroma-mcp --client-type persistent --data-dir /path/to/chromadb/data
```

This gives Claude access to `chroma_query_documents`, `chroma_list_collections`, and other Chroma tools for semantic search over your ingested files.
