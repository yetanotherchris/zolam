# Proposal: Zolam v3 — daemon-free, flat-file architecture

## What

Replace the ChromaDB + Docker + MCP architecture with a batch-oriented,
daemon-free design. Zolam becomes a Go CLI that shells out to an embedded
Python script (via `uv run`) for extraction, chunking, and embedding, and
stores indexes as per-project flat files (DuckDB by default, or JSONL).
Claude Code integration moves from an MCP server to an installed Agent
Skill plus plain CLI commands (`zolam query`, `zolam projects`).

The legacy ChromaDB/Docker/MCP path (`zolam chromadb`, `zolam mcp`,
`--backend chroma`) is retained for existing users but deprecated.

## Why

- ChromaDB requires a Docker container running whenever an AI tool
  queries the index — the wrong shape for a corpus updated infrequently.
- Extracted text lives inside ChromaDB's sqlite storage, invisible to
  agentic tools that want to `grep`/read files directly.
- chroma-mcp exists only to bridge query-text-to-embedding; it is another
  moving part and a registration step.
- Docker Desktop is a heavy prerequisite on macOS/Windows.

Prerequisites drop from "Docker + Docker Compose + uv" to just "uv".

## Scope

In scope: extraction/chunking/embedding pipeline, `duckdb` and `jsonl`
backends, `zolam ingest/update/query/projects/init`, the Claude Code
skill file, incremental updates, README updates.

Out of scope (non-goals): pure-Go/ONNX embeddings, LLM-generated
summaries at ingest time, scaling beyond ~thousands of files per project.
`zolam migrate` (chroma → duckdb/jsonl) was dropped from scope: with a
single-user deployment, re-ingesting the original source directories is
simpler and lossless — see `design.md`.

## Rollout

1. Extraction + chunking + embedding pipeline, jsonl backend.
2. duckdb backend, made default.
3. `query`, `init claude`/`init opencode`, skill file.
4. `update` incremental logic.
5. Deprecation notices on `chromadb`/`mcp`; README rewrite.
6. Release as v3.0.0 (breaking: `--collection` → `--project`, data dir layout).

See `design.md` for architecture and deviations from the original spec,
and `tasks.md` for the implementation checklist.
