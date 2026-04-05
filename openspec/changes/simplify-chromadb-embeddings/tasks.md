# Tasks: Simplify ChromaDB Embeddings

## 1. ChromaDB HTTP Client

- [ ] Create `src/internal/chromadb/client.go` with REST API client (heartbeat, get/create/delete collection, upsert, count)
- [ ] Add unit tests for ChromaDB client
- [ ] Move health check logic from `docker/chromadb.go` to new client

## 2. Text Extraction & Chunking

- [ ] Create `src/internal/zolam/extractor.go` with text extraction for plain text, PDF, DOCX
- [ ] Create `src/internal/zolam/chunker.go` with chunk_text port and chunk ID generation
- [ ] Add unit tests for chunker (short text, long text, overlap, deterministic IDs)
- [ ] Add Go dependencies for PDF and DOCX parsing to go.mod

## 3. Ingester Rewrite

- [ ] Rewrite `ingester.go` `Run()` to walk files, extract, chunk, and upsert directly via HTTP
- [ ] Rewrite `stats.go` to query ChromaDB directly via HTTP
- [ ] Preserve `outputFn` callback for progress output
- [ ] Preserve `RunUpdateOnly()` differential update flow
- [ ] Update unit tests

## 4. Remove Python Ingest Container

- [ ] Remove `ingest.py`
- [ ] Remove `Dockerfile`
- [ ] Remove `ingest` service from `docker-compose.yml`
- [ ] Remove `ingest` service from embedded `src/internal/docker/compose.yml`
- [ ] Remove ingest Docker image reference from `src/internal/docker/client.go`

## 5. Documentation & Verification

- [ ] Update `CLAUDE.md` architecture diagram and description
- [ ] Run `go build ./cmd/zolam/` — verify build succeeds
- [ ] Run `go test ./...` — verify all tests pass
- [ ] Run `go vet ./...` — verify no lint issues
