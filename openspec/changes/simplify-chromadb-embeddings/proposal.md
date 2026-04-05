# Proposal: Simplify ChromaDB Embeddings — Eliminate Python Ingest Container

## Why

The current architecture uses a separate Python Docker container (`ingest.py`) to extract text, chunk it, and send it to ChromaDB. This adds complexity:

- A ~750MB+ Docker image with Python, onnxruntime, tokenizers, and a pre-downloaded embedding model
- Docker-in-Docker orchestration from the Go binary
- Two languages to maintain for core functionality

ChromaDB's server already includes the default embedding function (all-MiniLM-L6-v2, 384 dimensions). When documents are sent via its REST API without pre-computed embeddings, the server generates them. This means:

1. Client-side embedding is unnecessary — the server handles it
2. The Python `HttpClient` is just making REST calls — Go can do the same with `net/http`
3. Text extraction and chunking are simple operations easily done in Go

## What Changes

- **Remove** `ingest.py` and the ingest Docker image (`Dockerfile`)
- **Remove** the `ingest` service from `docker-compose.yml` and embedded `compose.yml`
- **Add** `src/internal/chromadb/` — Go HTTP client for ChromaDB REST API
- **Add** `src/internal/zolam/extractor.go` — Text extraction (PDF, DOCX, plain text)
- **Add** `src/internal/zolam/chunker.go` — Text chunking (2000 chars, 200 overlap)
- **Modify** `src/internal/zolam/ingester.go` — Call ChromaDB directly instead of Docker
- **Modify** `src/internal/zolam/stats.go` — Query ChromaDB directly for stats
- **Update** `CLAUDE.md` — Architecture docs

## Impact

- Eliminates the Python ingest container and its Docker image entirely
- Removes Docker-in-Docker complexity for ingestion
- Faster ingestion (no container startup overhead)
- Single-language codebase (Go only, except ChromaDB server)
- Smaller overall footprint
