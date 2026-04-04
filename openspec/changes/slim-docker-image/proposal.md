## Why

The zolam Docker image (`ghcr.io/yetanotherchris/zolam`) is 1 GB uncompressed. The `ingest.py` script only needs to connect to a remote ChromaDB server via HTTP, but the Dockerfile installs the full `chromadb` package (530 MB of pip dependencies including onnxruntime, kubernetes, grpc, sympy, numpy) and bakes in the embedding model (88 MB). None of these are needed since the ChromaDB server handles embeddings.

## What Changes

- Replace `chromadb` pip package with `chromadb-client` (HTTP-only client, no server dependencies)
- Remove the embedding model pre-download step from the Dockerfile (server handles embeddings)
- Use a multi-stage Docker build to separate dependency installation from the final image
- Remove `pip` cache and package manager from the final image stage

## Capabilities

### New Capabilities
- `slim-docker-build`: Multi-stage Dockerfile that produces a minimal image containing only the runtime dependencies needed by the ingest client

### Modified Capabilities

## Impact

- `Dockerfile` - rewritten with multi-stage build
- `docker-compose.yml` - no changes expected (image reference stays the same)
- `src/internal/docker/compose.yml` - no changes expected
- `ingest.py` - no code changes (already uses `chromadb.HttpClient` only)
- Image size expected to drop from ~1 GB to under 300 MB
