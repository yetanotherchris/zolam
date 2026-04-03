# Functional Requirements: Go TUI CLI

## FR-1: TUI Main Menu

The application launches into an interactive TUI with the following menu options:

1. **Ingest** - Run the full ingestion pipeline
2. **Update Only** - Re-ingest only changed files (based on content hash)
3. **Download (rclone)** - Download files from Google Drive via rclone
4. **Stats** - Show collection statistics
5. **Reset Collection** - Delete and recreate a ChromaDB collection
6. **Start ChromaDB** - Start the ChromaDB container
7. **Stop ChromaDB** - Stop the ChromaDB container
8. **Settings** - View current configuration/environment
9. **Quit** - Exit the application

## FR-2: Ingest Command

Replaces the functionality of `ingest.sh` / `ingest.ps1`:

- Accept one or more source directories as arguments
- Filter files by extension (configurable, defaults to: `.md`, `.pdf`, `.docx`, `.txt`, `.py`, `.cs`, `.js`, `.ts`, `.json`, `.yml`, `.yaml`)
- Convert relative paths to absolute paths
- Build Docker volume mount arguments for each directory
- Ensure ChromaDB container is running (start it if not)
- Run the ingester Docker container with appropriate volume mounts and environment variables
- Display real-time Docker output with progress
- Support `--collection` flag to specify collection name (default: `my-notes`)
- Support `--extensions` flag to filter file types

### Docker Invocation

```
docker compose --profile ingest run --rm \
  -v /path1:/sources/dirname1 \
  -v /path2:/sources/dirname2 \
  ingest --directory /sources/dirname1 /sources/dirname2 \
  [--extensions .md .txt ...] \
  [--reset] [--stats]
```

## FR-3: Update-Only Mode

- Maintain a local hash manifest file (`.ingester-hashes.json`) mapping file paths to their SHA-256 content hashes
- On update run: scan all configured directories, compute hashes, compare with manifest
- Only re-ingest files whose hash has changed or that are new
- Remove from ChromaDB any files that no longer exist on disk
- Update the manifest after successful ingestion
- Show summary: X new, Y updated, Z removed, W unchanged

## FR-4: rclone Google Drive Download

- Run rclone via Docker to download files from Google Drive
- Docker command: `docker run --rm -v /local/path:/data -v ~/.config/rclone:/config/rclone rclone/rclone copy gdrive:source-folder /data`
- Accept source remote path and local destination as arguments
- Support configuration of rclone remote name (default: `gdrive`)
- Display progress during download
- The rclone config is expected to already exist at `~/.config/rclone/rclone.conf`

## FR-5: ChromaDB Management

- **Start**: `docker compose up -d chromadb` - start ChromaDB in background
- **Stop**: `docker compose down` - stop all services
- **Status**: Check if ChromaDB container is running via `docker ps`
- **Health check**: Verify ChromaDB is responding on port 8000 before ingestion
- Auto-start ChromaDB if not running when ingestion is requested

## FR-6: Environment Variable Handling

Required/optional environment variables:

| Variable | Required | Default | Description |
|---|---|---|---|
| `OPENROUTER_API_KEY` | No* | - | API key for OpenRouter embeddings |
| `OPENROUTER_MODEL` | No | `openai/text-embedding-3-small` | Embedding model |
| `COLLECTION_NAME` | No | `my-notes` | ChromaDB collection name |
| `USE_LOCAL_EMBEDDINGS` | No | - | Set to `1` to use local embeddings |
| `RCLONE_REMOTE` | No | `gdrive` | rclone remote name |
| `RCLONE_SOURCE` | No | - | Source path on remote |
| `INGESTER_DATA_DIR` | No | `./chromadb-data` | Local ChromaDB data directory |

*Required unless `USE_LOCAL_EMBEDDINGS=1`

Behavior:
- On startup, validate all environment variables
- Warn (yellow) for missing optional variables with defaults
- Error (red) for missing required variables (e.g., `OPENROUTER_API_KEY` when not using local embeddings)
- All env vars can alternatively be passed as CLI flags (flags take precedence)
- Support `.env` file loading from the current directory

## FR-7: CLI (Non-Interactive) Mode

All TUI operations are also available as CLI subcommands for scripting:

```
ingester ingest [directories...] [--extensions .md .txt] [--collection name] [--reset]
ingester update [directories...]
ingester download [--remote gdrive] [--source path] [--dest path]
ingester stats [--collection name]
ingester reset [--collection name]
ingester chromadb start|stop|status
ingester config
```

## FR-8: Statistics Display

- Show number of documents in collection
- Show number of chunks
- Show collection name
- Show embedding function type (OpenRouter vs local)
- Show ChromaDB status (running/stopped)

## FR-9: Docker Compose File Management

- The application embeds the `docker-compose.yml` content or generates it dynamically
- Writes docker-compose.yml to a known location (e.g., `~/.ingester/docker-compose.yml`) if it doesn't exist
- Ensures the compose file reflects current configuration (env vars, data directory)
