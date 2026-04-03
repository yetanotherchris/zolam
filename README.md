# ingester

Ingest your personal files (Google Drive, Markdown, GitHub repos) into ChromaDB, so you can query them in Claude.

## Prerequisites

- Docker and Docker Compose
- An `OPENROUTER_API_KEY` (for embeddings). Set `USE_LOCAL_EMBEDDINGS=1` instead if you want fully offline embeddings.
- A local directory with your files (Obsidian vault, rclone'd Google Drive, GitHub repos, etc.)

## Setup

Create the local data directories used by Docker Compose:

```bash
mkdir -p chromadb-data rclone-config gdrive-data
```

| Directory | Purpose |
|---|---|
| `chromadb-data/` | ChromaDB persistent storage |
| `rclone-config/` | Rclone configuration (`rclone.conf`) |
| `gdrive-data/` | Google Drive files synced by rclone |

## Usage

Save the `docker-compose.yml` from this repo. Add your API key to a `.env` file (docker compose picks this up automatically):

```bash
# .env
OPENROUTER_API_KEY=sk-or-...
```

Then:

```bash
# 1. Start ChromaDB
docker compose up -d chromadb

# 2. Ingest your files (replace paths and source name)
docker compose --profile ingest run --rm \
  -v /path/to/your/files:/sources/obsidian \
  ingest --source obsidian
```

Source names: `obsidian`, `gdrive`, `repos`. Mount multiple `-v` flags and omit `--source` to ingest everything.

You can also ingest arbitrary directories with `--directory`:

```bash
docker compose --profile ingest run --rm \
  -v /path/to/docs:/sources/docs \
  ingest --directory /sources/docs

# Multiple directories at once
docker compose --profile ingest run --rm \
  -v /path/to/notes:/sources/notes \
  -v /path/to/docs:/sources/docs \
  ingest --directory /sources/notes /sources/docs

# Filter by extension
docker compose --profile ingest run --rm \
  -v /path/to/docs:/sources/docs \
  ingest --directory /sources/docs --extensions .md .txt
```

```bash
# Check what's indexed
docker compose --profile ingest run --rm ingest --stats

# Wipe a collection and re-ingest
docker compose --profile ingest run --rm \
  -v /path/to/your/files:/sources/obsidian \
  ingest --reset --source obsidian
```

### Syncing Google Drive with rclone

Place your `rclone.conf` in the `rclone-config/` directory. You can generate one with `rclone config` on your host machine (the config file is typically at `~/.config/rclone/rclone.conf`). It should contain a `[gdrive]` remote, for example:

```ini
[gdrive]
type = drive
client_id = your-client-id
client_secret = your-client-secret
token = {"access_token":"...","token_type":"Bearer","refresh_token":"...","expiry":"..."}
```

Then sync and ingest:

```bash
# Sync Google Drive files locally
docker compose --profile rclone run --rm rclone

# Ingest the synced files
docker compose --profile ingest run --rm ingest --source gdrive
```

## Claude Code Integration (chroma-mcp)

```bash
pip install uv
claude mcp add chroma -- uvx chroma-mcp --client-type persistent --data-dir /path/to/chromadb/data
```

This gives Claude access to `chroma_query_documents`, `chroma_list_collections`, and other Chroma tools for semantic search over your ingested files.

## More Information

See [docs/advanced.md](docs/advanced.md) for embedding options, running without Compose, MCP configuration details, and other notes.
