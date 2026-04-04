## Context

The current Dockerfile uses a single-stage build with `python:3.12-slim` and installs the full `chromadb` server package. The ingest container only needs the HTTP client and the local embedding model (all-MiniLM-L6-v2), but the full package pulls in server-side dependencies (kubernetes, uvicorn, uvloop, chromadb_rust_bindings) that are never used.

Embeddings run client-side using ChromaDB's default embedding function (ONNXMiniLM_L6_V2). This was established in the `local-embeddings` change.

## Goals / Non-Goals

**Goals:**
- Reduce the Docker image size from ~1 GB
- Use multi-stage build to separate build dependencies from runtime
- Switch to `chromadb-client` package (HTTP client only) with explicit embedding dependencies

**Non-Goals:**
- Changing ingest.py functionality
- Modifying the ChromaDB server container
- Changing the docker-compose configuration
- Changing the embedding approach (must remain client-side)

## Decisions

### Use `chromadb-client` instead of `chromadb`

`ingest.py` only uses `chromadb.HttpClient` to connect to the remote ChromaDB server. The full `chromadb` package pulls in server-side dependencies (kubernetes, uvicorn, uvloop, chromadb_rust_bindings) that are never used. `chromadb-client` provides the same `HttpClient` API without server dependencies.

The import (`import chromadb`) and usage (`chromadb.HttpClient(...)`) remain identical - `chromadb-client` uses the same package namespace.

### Keep embedding model and dependencies

The embedding model (all-MiniLM-L6-v2) runs client-side via `onnxruntime` and `tokenizers`. These must be explicitly installed since `chromadb-client` does not include them as transitive dependencies (unlike the full `chromadb` package). The model pre-download step is retained.

### Multi-stage Docker build

- **Stage 1 (builder)**: Install pip packages into a virtual environment, download embedding model
- **Stage 2 (runtime)**: Copy only the virtual environment, model cache, and application code

This avoids carrying pip, wheel caches, build tools, and compilation artifacts into the final image.

## Risks / Trade-offs

- [Risk] `chromadb-client` may not expose the same API surface as `chromadb` -> Mitigation: verified that `ingest.py` only uses `HttpClient`, `get_collection`, `get_or_create_collection`, `delete_collection`, `collection.upsert`, `collection.count` - all available in `chromadb-client`
- [Risk] Missing transitive dependencies -> Mitigation: `onnxruntime` and `tokenizers` are explicitly installed; embedding verified working in built image
