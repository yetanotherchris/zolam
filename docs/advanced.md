# Advanced Usage

## Architecture

```
[Obsidian .md files]  ──┐
[rclone'd GDrive]     ──┼──> docker run ingest ──> ChromaDB (persistent, local)
[GitHub repos]        ──┘                                       │
                                                                │
                                   Claude Code ──> chroma-mcp ──┘
```

Everything runs locally.

## Embeddings

By default, embeddings use OpenRouter (`OPENROUTER_API_KEY` required). To use a different OpenRouter model:

```bash
docker compose --profile ingest run --rm \
  -e OPENROUTER_API_KEY=sk-or-... \
  -e OPENROUTER_MODEL=google/gemini-embedding-001 \
  -v /path/to/vault:/sources/obsidian \
  ingest --source obsidian
```

### Local Embeddings

Set `USE_LOCAL_EMBEDDINGS=1` to use sentence-transformers (all-MiniLM-L6-v2) with no API key:

```bash
docker compose --profile ingest run --rm \
  -e USE_LOCAL_EMBEDDINGS=1 \
  -v /path/to/vault:/sources/obsidian \
  ingest --source obsidian
```

**Important:** If switching embedding models, run with `--reset` first. Vectors from different models are not compatible.

## Running without Compose

You can run the GHCR image directly. Mount `/data` for ChromaDB storage and `/sources/<name>` for each source.

```bash
mkdir -p ~/chromadb/data

docker run --rm \
  -v ~/chromadb/data:/data \
  -v ~/ObsidianVault:/sources/obsidian \
  ghcr.io/yetanotherchris/ingester:latest --source obsidian
```

Other examples:

```bash
# Google Drive
docker run --rm \
  -v ~/chromadb/data:/data \
  -v ~/GDrive:/sources/gdrive \
  ghcr.io/yetanotherchris/ingester:latest --source gdrive

# GitHub repos
docker run --rm \
  -v ~/chromadb/data:/data \
  -v ~/repos:/sources/repos \
  ghcr.io/yetanotherchris/ingester:latest --source repos

# Stats
docker run --rm \
  -v ~/chromadb/data:/data \
  ghcr.io/yetanotherchris/ingester:latest --stats

# Wipe and re-ingest
docker run --rm \
  -v ~/chromadb/data:/data \
  -v ~/ObsidianVault:/sources/obsidian \
  ghcr.io/yetanotherchris/ingester:latest --reset --source obsidian
```

## MCP Configuration (manual)

Instead of `claude mcp add`, you can manually edit `.claude/settings.local.json`:

```json
{
  "mcpServers": {
    "chroma": {
      "command": "uvx",
      "args": [
        "chroma-mcp",
        "--client-type", "persistent",
        "--data-dir", "/path/to/chromadb/data"
      ]
    }
  }
}
```

Claude Code will then have access to:

- `chroma_query_documents` — semantic search
- `chroma_list_collections` — list what's indexed
- `chroma_get_collection_info` — collection stats
- `chroma_get_documents` — retrieve by ID or metadata filter

## Notes

- First ingestion is slower — the sentence-transformers model downloads inside the container. Subsequent runs use Docker's layer cache.
- Re-running without `--reset` is safe — upserts overwrite existing chunks.
- chroma-mcp and the ingestion script both use PersistentClient on the same data directory. Don't run them simultaneously.
- The sentence-transformers model runs on CPU inside the container. For a personal vault this is fast enough.
