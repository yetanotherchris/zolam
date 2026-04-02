# ingester

Ingest your personal files (Google Drive, Markdown, GitHub repos) into ChromaDB, so you can query them in Claude.

## Prerequisites

- Docker and Docker Compose
- An `OPENROUTER_API_KEY` (for embeddings). Set `USE_LOCAL_EMBEDDINGS=1` instead if you want fully offline embeddings.
- A local directory with your files (Obsidian vault, rclone'd Google Drive, GitHub repos, etc.)

## Usage

Save the `docker-compose.yml` from this repo, then:

```bash
# 1. Start ChromaDB
docker compose up -d chromadb

# 2. Ingest your files (replace paths and source name)
docker compose --profile ingest run --rm \
  -e OPENROUTER_API_KEY=sk-or-... \
  -v /path/to/your/files:/sources/obsidian \
  ingest --source obsidian
```

Source names: `obsidian`, `gdrive`, `repos`. Mount multiple `-v` flags and omit `--source` to ingest everything.

```bash
# Check what's indexed
docker compose --profile ingest run --rm ingest --stats

# Wipe a collection and re-ingest
docker compose --profile ingest run --rm \
  -e OPENROUTER_API_KEY=sk-or-... \
  -v /path/to/your/files:/sources/obsidian \
  ingest --reset --source obsidian
```

## Claude Code Integration (chroma-mcp)

```bash
pip install uv
claude mcp add chroma -- uvx chroma-mcp --client-type persistent --data-dir /path/to/chromadb/data
```

This gives Claude access to `chroma_query_documents`, `chroma_list_collections`, and other Chroma tools for semantic search over your ingested files.

## More Information

See [docs/advanced.md](docs/advanced.md) for embedding options, running without Compose, MCP configuration details, and other notes.
