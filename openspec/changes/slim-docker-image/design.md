## Context

The current Dockerfile uses a single-stage build with `python:3.12-slim`, installs the full `chromadb` server package, and pre-downloads the embedding model. The ingest container only acts as an HTTP client to the ChromaDB server running in a separate container.

## Goals / Non-Goals

**Goals:**
- Reduce the Docker image size from ~1 GB to under 300 MB
- Use multi-stage build to separate build dependencies from runtime
- Switch to `chromadb-client` package (HTTP client only)

**Non-Goals:**
- Changing ingest.py functionality
- Modifying the ChromaDB server container
- Changing the docker-compose configuration

## Decisions

### Use `chromadb-client` instead of `chromadb`

`ingest.py` only uses `chromadb.HttpClient` to connect to the remote ChromaDB server. The full `chromadb` package pulls in server-side dependencies (kubernetes, grpc, onnxruntime, uvicorn, chromadb_rust_bindings) that are never used. `chromadb-client` provides the same `HttpClient` API without server dependencies.

The import (`import chromadb`) and usage (`chromadb.HttpClient(...)`) remain identical - `chromadb-client` uses the same package namespace.

### Remove embedding model pre-download

The Dockerfile currently runs a Python snippet to pre-download the all-MiniLM-L6-v2 model (88 MB). This model is used for local embedding computation, but since the ingest client delegates embedding to the ChromaDB server, this step is unnecessary.

### Multi-stage Docker build

- **Stage 1 (builder)**: Install pip packages into a virtual environment
- **Stage 2 (runtime)**: Copy only the virtual environment and application code from stage 1

This avoids carrying pip, wheel caches, build tools, and compilation artifacts into the final image.

## Risks / Trade-offs

- [Risk] `chromadb-client` may not expose the same API surface as `chromadb` -> Mitigation: verified that `ingest.py` only uses `HttpClient`, `get_collection`, `get_or_create_collection`, `delete_collection`, `collection.upsert`, `collection.count` - all available in `chromadb-client`
- [Risk] Embedding model not available locally -> Mitigation: not needed since server handles embeddings; if needed in future, can add back
