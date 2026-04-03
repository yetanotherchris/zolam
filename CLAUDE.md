# CLAUDE.md

## Project Overview

Ingester is a semantic search tool that ingests personal files (markdown, PDF, DOCX, code) into ChromaDB for semantic search via Claude. It provides both a TUI (interactive) and CLI (scriptable) interface.

## Tech Stack

- **Go TUI/CLI**: Located in `src/`, built with Cobra (CLI), Bubbletea + Lipgloss (TUI)
- **Python ingester**: `ingest.py` runs inside Docker to process files into ChromaDB
- **Docker**: ChromaDB and ingester run as Docker containers via Docker Compose

## Build & Test

```bash
cd src
go build ./cmd/ingester/        # build
go test ./...                    # run all tests
go vet ./...                     # lint
```

## Project Structure

```
src/
├── cmd/ingester/main.go        # Entry point, CLI subcommands
├── internal/
│   ├── domain/                 # Config, manifest types
│   ├── docker/                 # Docker/compose client, ChromaDB, rclone
│   ├── ingester/               # Ingest pipeline, file hashing, stats
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
