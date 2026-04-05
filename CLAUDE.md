# CLAUDE.md

## Project Overview

Zolam is a semantic search tool that ingests personal files (markdown, PDF, DOCX, code) into ChromaDB for semantic search via Claude. It provides both a TUI (interactive) and CLI (scriptable) interface.

## Tech Stack

- **Go TUI/CLI**: Located in `src/`, built with Cobra (CLI), Bubbletea + Lipgloss (TUI)
- **Python ingester**: `ingest.py` runs inside Docker to process files into ChromaDB
- **Docker**: ChromaDB and zolam run as Docker containers via Docker Compose

## Architecture

```
┌─────────────────────────────────────────────────┐
│  Go TUI/CLI (src/)                              │
│  - Manages Docker containers via Docker Compose │
│  - Config, ingest orchestration, stats          │
└──────────────┬──────────────────────────────────┘
               │ starts
┌──────────────▼──────────────────────────────────┐
│  Ingest container (ingest.py)                   │
│  1. Extracts text from files (PDF, DOCX, etc.)  │
│  2. Chunks text into ~2000 char segments        │
│  3. Sends text chunks + metadata to ChromaDB    │
└──────────────┬──────────────────────────────────┘
               │ HTTP
┌──────────────▼──────────────────────────────────┐
│  ChromaDB server container                      │
│  - Generates embeddings server-side using       │
│    all-MiniLM-L6-v2 (384 dimensions)            │
│  - Stores and queries vectors                   │
└──────────────┬──────────────────────────────────┘
               │ queried by
┌──────────────▼──────────────────────────────────┐
│  chroma-mcp server (Claude tool)                │
│  - Must use same 384-dim embedding model        │
│    (all-MiniLM-L6-v2) for compatible queries    │
└─────────────────────────────────────────────────┘
```

Embeddings are generated server-side by ChromaDB using its default embedding function (all-MiniLM-L6-v2, 384 dimensions). The ingest container sends raw text; ChromaDB handles vectorization. The chroma-mcp plugin used by Claude must use the same model and vector size (384) or queries will fail due to dimension mismatch.

## Build & Test

```bash
cd src
go build ./cmd/zolam/        # build
go test ./...                    # run all tests
go vet ./...                     # lint
```

## Project Structure

```
src/
├── cmd/zolam/main.go        # Entry point, CLI subcommands
├── internal/
│   ├── domain/                 # Config, manifest types
│   ├── docker/                 # Docker/compose client, ChromaDB, rclone
│   ├── zolam/                  # Ingest pipeline, file hashing, stats
│   └── tui/                    # Bubbletea TUI (app, menu, ingest, progress, styles)
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
