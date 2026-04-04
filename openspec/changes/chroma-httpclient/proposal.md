# Proposal: Switch ChromaDB ingester from PersistentClient to HttpClient

## Motivation

The ingester currently uses `chromadb.PersistentClient(path="/data")` which accesses the ChromaDB data directory directly via a shared Docker volume. This bypasses the ChromaDB server entirely - the `chromadb` container exposes port 8000 but the ingester never connects to it.

Switching to `HttpClient` means the ingester talks to the ChromaDB server over HTTP, which is the intended client-server architecture. This eliminates the need for the ingester container to mount the ChromaDB data volume, and means it uses the same API path as any other client (including the chroma-mcp server).

## Scope

- Replace `chromadb.PersistentClient(path=CHROMA_DATA_DIR)` with `chromadb.HttpClient(host=..., port=...)`
- Add `CHROMA_HOST` and `CHROMA_PORT` environment variables (defaulting to `chromadb` and `8000`)
- Remove the shared data volume mount from the ingest service in both docker-compose files
- Remove the `CHROMA_DATA_DIR` constant from ingest.py
