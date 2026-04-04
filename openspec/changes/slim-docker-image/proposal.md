## Why

The zolam Docker image (`ghcr.io/yetanotherchris/zolam`) is 1 GB uncompressed. The `ingest.py` script only needs the ChromaDB HTTP client and the local embedding model, but the Dockerfile installs the full `chromadb` server package (530 MB of pip dependencies including kubernetes, uvicorn, uvloop, chromadb_rust_bindings) which are never used by the client.

## What Changes

- Replace `chromadb` pip package with `chromadb-client` (HTTP-only client, no server dependencies)
- Explicitly install `onnxruntime` and `tokenizers` (needed for client-side embedding with all-MiniLM-L6-v2)
- Use a multi-stage Docker build to separate dependency installation from the final image
- Remove `pip` and `__pycache__` from the final image stage

## Capabilities

### New Capabilities
- `slim-docker-build`: Multi-stage Dockerfile that produces a smaller image by removing server-side dependencies

### Modified Capabilities

## Impact

- `Dockerfile` - rewritten with multi-stage build and chromadb-client
- `docker-compose.yml` - no changes (image reference stays the same)
- `src/internal/docker/compose.yml` - no changes
- `ingest.py` - no code changes (already uses `chromadb.HttpClient` only)
- Image size reduced from ~1 GB to ~757 MB
