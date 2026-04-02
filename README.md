# ingester

Ingest your personal files (Google Drive, Markdown) into ChromaDB, so you can query them in Claude.

## Architecture

```
[Obsidian .md files]  ──┐
[rclone'd GDrive]     ──┼──> docker run ingest ──> ChromaDB (persistent, local)
[GitHub repos]        ──┘                                       │
                                                                │
                                   Claude Code ──> chroma-mcp ──┘
```

Everything runs locally. Default embedding model is sentence-transformers
(all-MiniLM-L6-v2) — no API keys needed.

## Quick Start

### 1. Create ChromaDB Data Directory

```powershell
mkdir C:\chromadb\data
```

### 2. Build the Ingestion Image

```powershell
cd C:\path\to\ingest-project
docker build -t ingest .
```

First build downloads the sentence-transformers model (~80MB) — subsequent
runs use the cached image.

### 3. Run Ingestion

Mount `/data` for ChromaDB storage, mount `/sources/<name>` for each source.

**Ingest Obsidian vault:**

```powershell
docker run --rm `
  -v C:\chromadb\data:/data `
  -v C:\Users\Chris\ObsidianVault:/sources/obsidian `
  ingest --source obsidian
```

**Ingest Google Drive (after rclone sync):**

```powershell
rclone sync gdrive:/ C:\Users\Chris\GDrive --progress

docker run --rm `
  -v C:\chromadb\data:/data `
  -v C:\Users\Chris\GDrive:/sources/gdrive `
  ingest --source gdrive
```

**Ingest GitHub repos:**

```powershell
docker run --rm `
  -v C:\chromadb\data:/data `
  -v C:\Users\Chris\repos:/sources/repos `
  ingest --source repos
```

**Ingest everything at once:**

```powershell
docker run --rm `
  -v C:\chromadb\data:/data `
  -v C:\Users\Chris\ObsidianVault:/sources/obsidian `
  -v C:\Users\Chris\GDrive:/sources/gdrive `
  -v C:\Users\Chris\repos:/sources/repos `
  ingest
```

**Check stats:**

```powershell
docker run --rm `
  -v C:\chromadb\data:/data `
  ingest --stats
```

**Wipe and re-ingest:**

```powershell
docker run --rm `
  -v C:\chromadb\data:/data `
  -v C:\Users\Chris\ObsidianVault:/sources/obsidian `
  ingest --reset --source obsidian
```

## Docker Compose

Start ChromaDB server:

```bash
docker compose up -d chromadb
```

Run ingestion via compose (with source volumes mounted manually or via override):

```bash
docker compose --profile ingest run --rm ingest --source obsidian
```

## OpenRouter Embeddings (optional)

Pass your API key as an environment variable to switch from local
sentence-transformers to OpenRouter:

```powershell
docker run --rm `
  -e OPENROUTER_API_KEY=sk-or-... `
  -v C:\chromadb\data:/data `
  -v C:\Users\Chris\ObsidianVault:/sources/obsidian `
  ingest --source obsidian
```

To use a different model:

```powershell
docker run --rm `
  -e OPENROUTER_API_KEY=sk-or-... `
  -e OPENROUTER_MODEL=google/gemini-embedding-001 `
  -v C:\chromadb\data:/data `
  -v C:\Users\Chris\ObsidianVault:/sources/obsidian `
  ingest --source obsidian
```

**Important:** If switching embedding models, run with `--reset` first.
Vectors from different models are not compatible.

## Claude Code Integration (chroma-mcp)

```powershell
pip install uv
claude mcp add chroma -- uvx chroma-mcp --client-type persistent --data-dir C:\chromadb\data
claude mcp list
```

Or manually edit `.claude/settings.local.json`:

```json
{
  "mcpServers": {
    "chroma": {
      "command": "uvx",
      "args": [
        "chroma-mcp",
        "--client-type", "persistent",
        "--data-dir", "C:\\chromadb\\data"
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

- First ingestion is slower — sentence-transformers model downloads inside the container. Subsequent runs use Docker's layer cache.
- Re-running without `--reset` is safe — upserts overwrite existing chunks.
- chroma-mcp and the ingestion script both use PersistentClient on the same data directory. Don't run them simultaneously.
- The sentence-transformers model runs on CPU inside the container. For a personal vault this is fast enough.
