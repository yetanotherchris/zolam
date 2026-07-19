# CLAUDE.md

## Project Overview

Zolam is a semantic search tool that ingests personal files (markdown, PDF, DOCX, code) into a per-directory SQLite (or JSONL) index for semantic search via Claude. It's a single Go binary — no Docker, Python, or Node runtime required to ingest or query. (A legacy ChromaDB/Docker/MCP workflow still exists under `zolam chromadb` for pre-v3 data; see README's "Deprecated" section.)

## Tech Stack

- **Go CLI**: Located in `src/`, built with Cobra. CGO is enabled — several dependencies wrap native libraries (see below).
- **SQLite + sqlite-vec**: `github.com/mattn/go-sqlite3` + `github.com/asg017/sqlite-vec-go-bindings/cgo` — the per-project vector index (`.zolam/index.db`). Chunk metadata lives in a plain `chunks` table; embeddings live in a parallel `chunks_vec` sqlite-vec virtual table (vec0), joined by rowid.
- **PDF extraction/rendering**: `github.com/gen2brain/go-fitz` (MuPDF, statically bundled — no extra install).
- **OCR**: `github.com/otiai10/gosseract` (Tesseract) — needs Tesseract installed on the host; falls back gracefully (page left blank) if it isn't.
- **Embeddings**: `github.com/yalue/onnxruntime_go` + `github.com/daulet/tokenizers` (the real HuggingFace tokenizers library, for byte-exact tokenization) run `BAAI/bge-small-en-v1.5` (384 dims). The onnxruntime shared library, tokenizer, and model weights download once into `~/.zolam` on first use.
- **DOCX extraction**: `github.com/fumiama/go-docx` (pure Go, no native dependency).

## Architecture

```
┌───────────────────────────────────────────────────┐
│  Go CLI (src/cmd/zolam, src/internal/zolam)       │
│  1. Hashes files, diffs against file-hashes.json  │
│  2. Bounded worker pool (goroutines) per file:    │
│     extract (go-fitz/go-docx/plain) → OCR         │
│     fallback if needed → chunk → embed            │
│  3. Single writer commits chunks+vectors to the   │
│     project's index (SQLiteRepo or JsonlRepo)     │
└───────────────────────────────────────────────────┘
```

Both backends are kept to a single open connection/writer by design — `lock.go` enforces this with a project-local lock file so a second concurrent `zolam ingest`/`query` against the same project fails with a clear message instead of a corrupted or confusingly-locked index.

Building from source needs a C compiler (CGO) and a one-time fetch of the `daulet/tokenizers` static library — see README's "Building from Source".

## Build & Test

```bash
cd src
go run ./tools/fetchnative   # one-time: fetches native/tokenizers/libtokenizers.a
go build ./cmd/zolam/        # build (CGO_ENABLED=1)
go test ./...                # run all tests
go vet ./...                 # lint
```

## Project Structure

```
src/
├── cmd/zolam/main.go        # Entry point, CLI subcommands
├── internal/
│   ├── domain/                 # Config, project.json types
│   ├── docker/                 # Docker/compose client (legacy ChromaDB path only)
│   └── zolam/                  # Ingest/query pipeline: extraction, chunking, embedding,
│                                # SQLite/JSONL repos, worker pool, file hashing, lock file
├── native/tokenizers/        # Fetched by tools/fetchnative, gitignored
├── tools/fetchnative/        # Fetches the daulet/tokenizers static lib pre-build
├── go.mod
└── go.sum
```

## OpenSpec (Spec-Driven Development)

This project uses [OpenSpec](https://github.com/Fission-AI/OpenSpec) for spec-driven development. Install with:

```bash
npm install -g @fission-ai/openspec@latest
```

### OpenSpec Workflow

Changes are planned and tracked as structured artifacts in `openspec/changes/<change-name>/`:

- **proposal.md** - What & why (scope, motivation, decisions)
- **design.md** - How (architecture, components, patterns)
- **specs/** - Functional and non-functional requirements, scenarios
- **tasks.md** - Implementation checklist with checkboxes

### OpenSpec Commands

```bash
openspec list --json                              # List active changes
openspec status --change "<name>" --json          # Check change status
openspec instructions apply --change "<name>" --json  # Get implementation instructions
```

### OpenSpec Skills (Claude Code)

Use these skills when working with changes:

- `/openspec-explore` - Investigate problems and explore ideas (read-only)
- `/openspec-propose` - Create a new change with all artifacts
- `/openspec-apply-change` - Implement tasks from a change
- `/openspec-archive-change` - Finalize and archive a completed change

### Development Workflow

1. **Propose**: Create a change with `/openspec-propose` to define what you're building
2. **Implement**: Use `/openspec-apply-change` to work through tasks, marking each complete
3. **Archive**: Use `/openspec-archive-change` when all tasks are done

Always read the OpenSpec context files (proposal, design, specs, tasks) before implementing. Mark task checkboxes (`- [ ]` to `- [x]`) as you complete each one.
