# Design: Simplify ChromaDB Embeddings

## Context

The Go binary currently orchestrates ingestion by running a Python Docker container that:
1. Extracts text from files (PDF, DOCX, plain text)
2. Chunks text into ~2000 char segments with 200 char overlap
3. Sends documents to ChromaDB via `chromadb.HttpClient`

ChromaDB's server generates embeddings server-side when documents are sent without pre-computed vectors. The Python client is just an HTTP wrapper.

## Goals

- Move all ingestion logic into Go, calling ChromaDB's REST API directly
- Maintain identical chunking behavior (2000 char chunks, 200 char overlap, deterministic SHA256 IDs)
- Maintain identical metadata schema (`source`, `file`, `chunk`, `total_chunks`)
- Keep the same CLI/TUI interface unchanged

## Non-Goals

- Changing the embedding model or dimensions (server handles this)
- Changing the ChromaDB server configuration
- Modifying the TUI beyond what's needed for the new ingestion flow
- Adding new file format support

## Design

### 1. ChromaDB HTTP Client (`src/internal/chromadb/`)

A thin Go HTTP client wrapping ChromaDB's REST API v1:

- `GET /api/v1/heartbeat` — Health check (already exists in docker/chromadb.go, will move)
- `POST /api/v1/collections` — Create collection
- `GET /api/v1/collections/{name}` — Get collection
- `DELETE /api/v1/collections/{name}` — Delete collection
- `POST /api/v1/collections/{id}/upsert` — Upsert documents (server generates embeddings)
- `GET /api/v1/collections/{id}/count` — Document count

Uses `net/http` from stdlib. No external dependencies needed.

### 2. Text Extraction (`src/internal/zolam/extractor.go`)

Go libraries for file parsing:
- **PDF**: `github.com/dslipak/pdf` or `github.com/ledongthuc/pdf` (pure Go, no CGO)
- **DOCX**: `github.com/nguyenthenguyen/docx` or similar
- **Plain text** (.md, .txt, .py, .cs, .js, .ts, .json, .yml, .yaml): direct `os.ReadFile`

### 3. Text Chunking (`src/internal/zolam/chunker.go`)

Port of the Python `chunk_text()` function — simple character-based splitting:
- Chunk size: 2000 characters
- Overlap: 200 characters
- Skip empty chunks
- Return whole text if shorter than chunk size

### 4. Ingester Changes (`src/internal/zolam/ingester.go`)

Replace Docker container execution with direct Go calls:
1. Walk directories matching extensions
2. Extract text from each file
3. Chunk text
4. Generate deterministic IDs (SHA256 of `{source}:{filepath}:{chunk_index}`)
5. Batch upsert to ChromaDB (50 documents per batch)

The `outputFn` callback pattern is preserved for TUI/CLI progress output.

### 5. Stats Changes (`src/internal/zolam/stats.go`)

Replace Docker container `--stats` call with direct HTTP: `GET /api/v1/collections/{id}/count`.

## Risks / Trade-offs

1. **PDF extraction quality**: Go PDF libraries may extract text differently than PyMuPDF. Mitigation: test with representative PDFs; PyMuPDF is known for good extraction, so verify Go library output quality.
2. **DOCX extraction**: Go DOCX libraries are less mature than python-docx. Mitigation: test with representative DOCX files.
3. **ChromaDB API compatibility**: REST API may change between versions. Mitigation: pin ChromaDB server version in docker-compose.yml.
