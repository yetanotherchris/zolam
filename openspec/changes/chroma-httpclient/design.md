# Design: Switch to ChromaDB HttpClient

## Changes

### Python ingester (`ingest.py`)
- Remove `CHROMA_DATA_DIR = "/data"` constant
- Add `CHROMA_HOST` (default: `chromadb`) and `CHROMA_PORT` (default: `8000`) from environment
- Replace `chromadb.PersistentClient(path=CHROMA_DATA_DIR)` with `chromadb.HttpClient(host=CHROMA_HOST, port=CHROMA_PORT)`
- The `chromadb` hostname is the Docker Compose service name, so it resolves within the Docker network

### Docker Compose (`docker-compose.yml`)
- Remove the `./chromadb-data:/data` volume mount from the `ingest` service
- Add `CHROMA_HOST` and `CHROMA_PORT` environment variables to `ingest` service
- The `chromadb` service remains unchanged (it still needs its own data volume)

### Embedded Docker Compose (`src/internal/docker/compose.yml`)
- Same changes as the root docker-compose.yml
- Remove the `${ZOLAM_DATA_DIR}:/data` volume mount from `ingest`
- Add `CHROMA_HOST` and `CHROMA_PORT` environment variables

### No changes needed
- Go code (`chromadb.go`) - already connects via HTTP for health checks
- Dockerfile - no volume-related changes needed
- CLI/TUI code - no changes needed
