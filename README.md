# zolam

Ingest your personal files (PDF, Markdown, Docx, Txt, code) into a local,
daemon-free flat-file index for semantic search in Claude Code / OpenCode —
no Docker, no background service.

Zolam is two parts:
 - A Go CLI that walks your directories, hashes files for incremental
   updates, and generates a human-readable `index.md` summary.
 - An embedded Python script (run via [`uv`](https://docs.astral.sh/uv/)) that
   extracts text, chunks it, embeds it locally, and writes a per-project
   `index.duckdb` or `index.jsonl` file.

Nothing runs between invocations: `ingest`/`update`/`query` are batch
commands that start, do their work, and exit.

## Quick Start

**Prerequisite:** [uv](https://docs.astral.sh/uv/getting-started/installation/) (`brew install uv`, `winget install astral-sh.uv`, or `scoop install uv`). uv provisions Python and every pipeline dependency itself on first run.

```bash
# Ingest files into a project (defaults to the duckdb backend)
zolam ingest ~/notes --project my-project --extensions .md,.pdf

# Re-ingest only changed files; directories are remembered in project.json
zolam update --project my-project

# Ask a question — semantic search over the project's chunks
zolam query "what did we agree on renewal terms?" --project my-project

# Install the Claude Code skill so Claude can search your files itself
zolam init claude
```

Each project lives entirely under `~/.zolam/<project-name>/`:

```
~/.zolam/my-project/
  project.json       # backend, embedding model, source dirs, extensions
  index.duckdb        # (or index.jsonl) the vector index
  index.md            # human-readable summary of every indexed file
  extracted/           # markdown sidecars for PDFs/DOCX (grep-able text)
  file-hashes.json     # incremental-update state
```

### Installation

**Homebrew (macOS/Linux)**
```bash
brew install yetanotherchris/tap/zolam
```

**Winget (Windows)**
```powershell
winget install yetanotherchris.zolam
```
> Note: the winget package may lag behind the latest release, as each new version requires a PR to the winget-pkgs repository.

**Scoop (Windows)**
```powershell
scoop bucket add zolam https://github.com/yetanotherchris/zolam
scoop install zolam
```

## Index backends

Set with `--backend` on first `ingest` (recorded thereafter in `project.json`):

| Backend | When to use |
|---|---|
| `duckdb` (default) | General use — SQL-queryable, supports keyword (`ILIKE`) search alongside semantic search. |
| `jsonl` | You want the index itself to be plain-text: greppable, diffable, easy to inspect or version. |
| `chroma` (legacy) | You're already using the pre-v3 ChromaDB/Docker/MCP workflow and want to keep doing so. |

## Commands

### ingest

Ingest files from one or more directories into a project. `--project` and `--extensions` are required for a brand new project.

```bash
zolam ingest <dirs...> --project <name> --extensions <exts> [--backend duckdb|jsonl|chroma] [--reset]

# Ingest markdown files (duckdb backend)
zolam ingest ~/notes --project my-project --extensions .md

# Multiple directories and extensions
zolam ingest ~/notes ~/docs --project research --extensions .md,.txt,.pdf

# Reset (delete and re-ingest from scratch), e.g. to switch backends
zolam ingest ~/notes --project my-project --extensions .md --backend jsonl --reset
```

Subdirectories are scanned recursively. Binary formats (PDF, DOCX) get a
markdown sidecar under `extracted/`; plain-text/code files are indexed and
summarized straight from their original path.

### update

Re-ingest only files that have changed since the last ingest/update.
Directories are optional — they default to the `source_dirs` recorded in
`project.json` — but can be passed to override them.

```bash
zolam update --project my-project
zolam update ~/notes ~/more-notes --project my-project   # override source dirs
```

### query

Search a project's index.

```bash
zolam query "<question>" --project <name> [--top-k 5] [--keyword] [--json]

zolam query "renewal terms" --project my-project
zolam query "invoice" --project my-project --keyword   # substring/ILIKE, no embedding step
```

### projects

```bash
zolam projects list              # name, backend, file count, model, last ingest
zolam projects remove <name>     # delete a project's index, sidecars, and metadata
```

### init

Install AI-tool integration — no MCP server or registration step required.

```bash
zolam init claude      # installs ~/.claude/skills/zolam/SKILL.md
zolam init opencode    # installs ~/.config/opencode/AGENTS.md
```

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `ZOLAM_DATA_DIR` | `~/.zolam` | Root directory for all project data and embedded scripts |
| `ZOLAM_CHROMADB_DATA_DIR` | `~/.zolam/chromadb` | Legacy `--backend chroma` ChromaDB persistent storage path |

## Supported File Extensions

`.md`, `.pdf`, `.docx`, `.txt`, `.py`, `.cs`, `.js`, `.ts`, `.json`, `.yml`, `.yaml`, `.csv`, `.html`, `.htm`

## Deprecated: ChromaDB / Docker / MCP workflow

Before v3, zolam ran a ChromaDB server in Docker and required registering
a `chroma-mcp` MCP server with Claude Code/OpenCode. This still works via
`--backend chroma`, but is deprecated in favor of the daemon-free
`duckdb`/`jsonl` workflow above — it requires Docker Desktop, a background
container, and an extra MCP registration step that the v3 flow doesn't
need.

```bash
zolam chromadb start                                             # start the container
zolam ingest ~/notes --project my-project --extensions .md --backend chroma
zolam mcp claude                                                  # register chroma-mcp
zolam collections list                                            # list chroma collections
```

There's no automated migration path from `chroma` to `duckdb`/`jsonl` —
re-ingest your source directories into a new project with `zolam ingest`.

## Building from Source

```bash
cd src
go build -o zolam ./cmd/zolam/
go test ./...
```

## Name

The name zolam comes from Midazolam, a sedative most people take for an endoscopy or other procedures. It induces sleepiness, decreases anxiety, and causes anterograde amnesia.
