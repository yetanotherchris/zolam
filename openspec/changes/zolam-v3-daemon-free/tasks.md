# Tasks: Zolam v3

## Foundation
- [x] `internal/domain`: `Project` struct, `project.json` load/save, data-dir resolution (`ZOLAM_DATA_DIR`)
- [x] `internal/zolam`: JSON file-hash store (`file-hashes.json`) replacing the SQLite store for v3 projects

## Python pipeline
- [x] `ingest.py` (PEP 723): extraction (md/txt/code passthrough, PDF via pymupdf, DOCX via python-docx)
- [x] Chunker: heading/paragraph boundaries, ~2000 chars, ~15% overlap; PDFs chunked per-page
- [x] Embedding via fastembed `BAAI/bge-small-en-v1.5`
- [x] `duckdb` backend: schema, upsert-by-delete-then-insert, cosine similarity query, ILIKE keyword search
- [x] `jsonl` backend: header record, append/rewrite, numpy cosine similarity, substring keyword search
- [x] `--mode query`: embed query, rank, JSON output

## Go orchestration
- [x] `go:embed` the script, write to `~/.zolam/scripts/`, content-hash versioning
- [x] `EnsureUV` check with install remedy message
- [x] `RunIngest`/`RunQuery` wrappers (stream stderr live, parse final stdout JSON)
- [x] Hash diff (added/changed/removed) reused from existing `HashDirectory`
- [x] `index.md` generation from files already on disk (no Python round-trip)

## CLI
- [x] `zolam ingest`: `--project` (with hidden `--collection` alias), `--backend duckdb|jsonl|chroma`, `--reset`
- [x] `zolam update`: dirs optional, defaults to `project.json.source_dirs`
- [x] `zolam query`: semantic (default) and `--keyword`, `--json`
- [x] `zolam projects list` / `zolam projects remove`
- [x] `zolam init claude` / `zolam init opencode`
- [x] Deprecation notices on `zolam chromadb`, `zolam mcp`, `zolam collections`

## Claude Code skill
- [x] Embed and install `~/.claude/skills/zolam/SKILL.md`

## Tests
- [x] Chunker boundary/overlap unit tests (Go-side heuristics used by index.md, plus Python chunker exercised via integration run)
- [x] Hash-diff (add/change/remove) unit tests
- [x] `index.md` generation test
- [x] Script version/rewrite test

## Verification
- [x] Real end-to-end run (fixture dir) against both `duckdb` and `jsonl` backends: ingest, query (semantic + keyword), update after modifying one file

## Docs
- [x] README quick start rewritten around v3 flow
- [x] Deprecation section + three-backends section
