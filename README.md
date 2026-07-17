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

Nothing runs between invocations: `ingest`/`query` are batch commands
that start, do their work, and exit. `ingest` is safe to re-run any
time — it's both the first-time indexer and the incremental updater.

## Quick Start

**Prerequisite:** [uv](https://docs.astral.sh/uv/getting-started/installation/) (`brew install uv`, `winget install astral-sh.uv`, or `scoop install uv`). uv provisions Python and every pipeline dependency itself on first run.

```bash
# Install uv 
curl -LsSf https://astral.sh/uv/install.sh | sh
brew install uv             # macOS/Linux
winget install astral-sh.uv # Windows
scoop install uv            # Windows

# Ingest files into the current directory's project (defaults to the duckdb backend)
cd ~/notes
zolam ingest --extensions .md,.pdf

# Safe to re-run any time — only added/changed/removed files are reprocessed
zolam ingest

# Ask a question — semantic search over the indexed chunks
zolam query "what did we agree on renewal terms?"

# Install the Claude Code skill so Claude can search your files itself
zolam init claude
```

A project is just a directory with a `.zolam/` folder in it — there's no
global registry or `--project` flag. `ingest` creates it on first run and
refreshes it on every subsequent run; `query` looks for it in the current
directory:

```
~/notes/.zolam/
  project.json        # backend, embedding model, source dirs, extensions
  index.duckdb         # (or index.jsonl) the vector index
  index.md             # human-readable summary of every indexed file
  extracted/            # markdown sidecars for PDFs/DOCX (grep-able text)
  file-hashes.json      # incremental-update state
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

Index files into the current directory's project — creating it on first run,
refreshing it (only added/changed/removed files) on every run after. With no
arguments it indexes the current directory itself using every supported
extension; pass one or more directories and/or `--extensions` to narrow it.

```bash
zolam ingest [dirs...] [--extensions <exts>] [--backend duckdb|jsonl] [--reset]

# Index the current directory (all supported extensions)
zolam ingest

# Ingest markdown files from a specific directory (duckdb backend)
zolam ingest ~/notes --extensions .md

# Multiple directories and extensions; re-run any time to pick up changes
zolam ingest ~/notes ~/docs --extensions .md,.txt,.pdf

# Reset (delete and re-ingest from scratch), e.g. to switch backends
zolam ingest ~/notes --extensions .md --backend jsonl --reset
```

Subdirectories are scanned recursively. Binary formats (PDF, DOCX) get a
markdown sidecar under `.zolam/extracted/`; plain-text/code files are indexed
and summarized straight from their original path. Directories passed on a
later run override the `source_dirs` recorded in `.zolam/project.json`;
omit them to reuse what's already stored.

### query

Search the current directory's index.

```bash
zolam query "<question>" [--top-k 5] [--keyword] [--json]

zolam query "renewal terms"
zolam query "invoice" --keyword   # substring/ILIKE, no embedding step
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
| `ZOLAM_DATA_DIR` | `~/.zolam` | Cache directory for the embedded Python pipeline script (project data itself lives in each project's own `.zolam/` folder) |
| `ZOLAM_CHROMADB_DATA_DIR` | `~/.zolam/chromadb` | Legacy `--backend chroma` ChromaDB persistent storage path |

## Supported File Extensions

`.md`, `.pdf`, `.docx`, `.txt`, `.py`, `.cs`, `.js`, `.ts`, `.json`, `.yml`, `.yaml`, `.csv`, `.html`, `.htm`

## Deprecated: ChromaDB / Docker / MCP workflow

Before v3, zolam ran a ChromaDB server in Docker and required registering
a `chroma-mcp` MCP server with Claude Code/OpenCode. It's deprecated in
favor of the daemon-free `duckdb`/`jsonl` workflow above — it requires
Docker Desktop, a background container, and an extra MCP registration
step that the v3 flow doesn't need. `zolam ingest`/`zolam query` no
longer support this backend at all (`--backend chroma` is rejected);
container management and MCP registration live under `zolam chromadb`
for existing ChromaDB data only:

```bash
zolam chromadb start                                              # start the container
zolam chromadb mcp claude                                         # register chroma-mcp
zolam chromadb collections list                                   # list chroma collections
```

Ingesting new data into ChromaDB requires the standalone
`docker-compose.yml` in this repo (bypassing the `zolam` CLI entirely):

```bash
docker compose --profile ingest run --rm \
  -v /path/to/notes:/sources/notes \
  ingest --directory /sources/notes
```

There's no automated migration path from `chroma` to `duckdb`/`jsonl` —
re-ingest your source directories into a new project with `zolam ingest`.

## Why Python?

The ingest pipeline (text extraction, chunking, embedding) runs as an
embedded Python script rather than pure Go, mainly because of the embedding
step: generating vectors with `BAAI/bge-small-en-v1.5` needs an ONNX
inference runtime plus a matching tokenizer, and Python's
[`fastembed`](https://github.com/qdrant/fastembed) package bundles both,
mature and battle-tested.

A Go-only pipeline is possible but not yet a clean swap:
- [`onnxruntime.ai`](https://onnxruntime.ai/docs/) (what `fastembed` runs on under the hood) has no official Go API — you'd rely on a community CGo binding like [`yalue/onnxruntime_go`](https://github.com/yalue/onnxruntime_go), trading a static Go binary for CGo plus a bundled native `onnxruntime` shared library per platform.
- Go still lacks a mature, bit-exact tokenizer matching `fastembed`'s output for `BAAI/bge-small-en-v1.5`, so tokenization would need to be built and verified against the Python output to avoid silently different embeddings.
- PDF/DOCX extraction, however, has solid pure-Go options — [`gopdf`](https://github.com/razvandimescu/gopdf) and [`go-docx`](https://github.com/fumiama/go-docx) — so that half of the pipeline could move to Go today.

`uv` keeps the Python dependency painless (no manual install, auto-provisioned
on first run), so it stays for now.

## Building from Source

```bash
cd src
go build -o zolam ./cmd/zolam/
go test ./...
```

## Name

The name zolam comes from Midazolam, a sedative most people take for an endoscopy or other procedures. It induces sleepiness, decreases anxiety, and causes anterograde amnesia.
